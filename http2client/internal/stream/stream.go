package stream

import (
	"bytes"
	"fmt"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/internal/eventloop/commands"
	"github.com/fstab/h2c/http2client/internal/streamstate"
	"golang.org/x/net/http2/hpack"
	"os"
)

type Stream interface {
	StreamId() uint32
	GetState() streamstate.StreamState
	SetState(state streamstate.StreamState)

	RequestHeaders() []hpack.HeaderField
	// ResponseHeaders() []hpack.HeaderField

	// Get the received HTTP body (concatenated payloads of DATA frames).
	ResponseBody() []byte

	// With push promises it may happen that a stream is created before the client created an HttpRequest.
	// This method is for associating these streams with a request.
	// If the stream is already associated with a request, the method returns an error.
	AssociateWithCommand(cmd *commands.HttpCommand) error

	// SendFrame doesn't mean the frame is sent directly.
	// DATA frames can be postponed by flow control.
	// However, this method will return immediately, postponed frames will be cached and
	// handled under the hood as soon as a WINDOW_UPDATE is received.
	SendFrame(frame frames.Frame)
	// Handle a received frame for this stream.
	ReceiveFrame(frame frames.Frame)
	// Send RST_STREAM
	CloseWithError(errorCode frames.ErrorCode, msg string)
	// Called by the connection if a WINDOW_UPDATE for the connection is received.
	ProcessPendingDataFrames()
}

type FlowControlledFrameWriter interface {
	Write(frame frames.Frame)
	RemainingFlowControlWindowIsEnough(nBytesToWrite int64) bool
	DecreaseFlowControlWindow(nBytesToWrite int64)
}

type streamError struct {
	msg string
}

func (err *streamError) Error() string {
	return err.msg
}

func newStreamError(format string, a ...interface{}) *streamError {
	return &streamError{
		msg: fmt.Sprintf(format, a...),
	}
}

type stream struct {
	state                      streamstate.StreamState
	requestHeaders             []hpack.HeaderField
	responseHeaders            []hpack.HeaderField
	responseBody               bytes.Buffer
	err                        *streamError // RST_STREAM sent or received.
	cmd                        *commands.HttpCommand
	initialReceiveWindowSize   int64
	remainingSendWindowSize    int64
	remainingReceiveWindowSize int64
	pendingDataFrameWrites     []*frames.DataFrame
	streamId                   uint32
	out                        FlowControlledFrameWriter
}

func New(streamId uint32, cmd *commands.HttpCommand, initialSendWindowSize uint32, initialReceiveWindowSize uint32, out FlowControlledFrameWriter) *stream {
	return &stream{
		state:           streamstate.IDLE,
		requestHeaders:  make([]hpack.HeaderField, 0),
		responseHeaders: make([]hpack.HeaderField, 0),
		streamId:        streamId,
		cmd:             cmd,
		remainingSendWindowSize:    int64(initialSendWindowSize),
		initialReceiveWindowSize:   int64(initialReceiveWindowSize),
		remainingReceiveWindowSize: int64(initialReceiveWindowSize),
		pendingDataFrameWrites:     make([]*frames.DataFrame, 0),
		out: out,
	}
}

func (s *stream) ReceiveFrame(frame frames.Frame) {
	wasClosedBefore := s.state == streamstate.CLOSED
	err := streamstate.HandleIncomingFrame(s, frame)
	if err != nil {
		s.CloseWithError(err.ErrorCode, err.Message)
		return
	}
	switch frame := frame.(type) {
	case *frames.DataFrame:
		s.receiveDataFrame(frame)
	case *frames.HeadersFrame:
		s.receiveHeadersFrame(frame)
	case *frames.PriorityFrame:
		s.notImplementedYet(frame)
	case *frames.RstStreamFrame:
		s.receiveRstStreamFrame(frame)
	case *frames.PushPromiseFrame:
		s.receivePushPromiseFrame(frame)
	case *frames.WindowUpdateFrame:
		s.receiveWindowUpdateFrame(frame)
	default:
		// TODO: error handling
		fmt.Fprintf(os.Stderr, "Received unknown frame type %v\n", frame.Type())
	}
	if s.state == streamstate.CLOSED && !wasClosedBefore {
		s.finalizeCommand()
	}
}

func (s *stream) receiveDataFrame(frame *frames.DataFrame) {
	s.flowControlForIncomingDataFrame(frame)
	s.appendResponseBody(frame.Data)
}

func (s *stream) receiveHeadersFrame(frame *frames.HeadersFrame) {
	if !frame.EndHeaders {
		s.CloseWithError(frames.REFUSED_STREAM, fmt.Sprintf("Unable to process %v without the END_HEADERS flag, because CONTINUATIONs are not implemented yet.", frame.Type()))
	} else {
		s.addResponseHeaders(frame.Headers...)
	}
}

func (s *stream) receiveRstStreamFrame(frame *frames.RstStreamFrame) {
	if frame.ErrorCode == frames.NO_ERROR {
		s.err = newStreamError("Server sent %v.", frame.Type())
	} else {
		s.err = newStreamError("Server sent %v with error code %v.", frame.Type(), frame.ErrorCode)
	}
}

func (s *stream) receivePushPromiseFrame(frame *frames.PushPromiseFrame) {
	if !frame.EndHeaders {
		s.CloseWithError(frames.REFUSED_STREAM, fmt.Sprintf("%v with multiple header frames not supported.", frame.Type()))
	} else {
		s.addRequestHeaders(frame.Headers...)
	}
}

func (s *stream) notImplementedYet(frame frames.Frame) {
	fmt.Fprintf(os.Stderr, "Ignoring %v frame, because this frame type is not implemented yet.", frame.Type())
}

func (s *stream) CloseWithError(errorCode frames.ErrorCode, msg string) {
	if s.state == streamstate.CLOSED {
		return
	}
	rstStream := frames.NewRstStreamFrame(s.streamId, errorCode)
	s.err = newStreamError("%v", msg)
	s.SendFrame(rstStream)
}

func (s *stream) SendFrame(frame frames.Frame) {
	wasClosedBefore := s.state == streamstate.CLOSED
	switch frame := frame.(type) {
	case *frames.DataFrame:
		size := int64(len(frame.Data))
		if s.RemainingFlowControlWindowIsEnough(size) {
			s.DecreaseFlowControlWindow(size)
			streamstate.HandleOutgoingFrame(s, frame)
			s.out.Write(frame)
		} else {
			s.scheduleDataFrameWrite(frame)
		}
	case *frames.HeadersFrame:
		s.addRequestHeaders(frame.Headers...)
		streamstate.HandleOutgoingFrame(s, frame)
		s.out.Write(frame)
	default:
		streamstate.HandleOutgoingFrame(s, frame)
		s.out.Write(frame)
	}
	if s.state == streamstate.CLOSED && !wasClosedBefore {
		s.finalizeCommand()
	}
}

func (s *stream) RemainingFlowControlWindowIsEnough(nBytesToWrite int64) bool {
	return s.remainingReceiveWindowSize > nBytesToWrite && s.out.RemainingFlowControlWindowIsEnough(nBytesToWrite)
}

func (s *stream) DecreaseFlowControlWindow(nBytesToWrite int64) {
	s.out.DecreaseFlowControlWindow(nBytesToWrite)
	s.remainingSendWindowSize -= nBytesToWrite
}

func (s *stream) receiveWindowUpdateFrame(frame *frames.WindowUpdateFrame) {
	// TODO: stream error if increment is 0.
	s.remainingSendWindowSize += int64(frame.WindowSizeIncrement)
	s.ProcessPendingDataFrames()
}

// Just a quick implementation to make large downloads work.
// Should be replaced with a more sophisticated flow control strategy
func (s *stream) flowControlForIncomingDataFrame(frame *frames.DataFrame) {
	threshold := int64(2 << 13) // size of one frame
	s.remainingReceiveWindowSize -= int64(len(frame.Data))
	if s.remainingReceiveWindowSize < threshold {
		diff := s.initialReceiveWindowSize - s.remainingReceiveWindowSize
		s.remainingReceiveWindowSize += diff
		s.SendFrame(frames.NewWindowUpdateFrame(s.streamId, uint32(diff)))
	}
}

func (s *stream) ProcessPendingDataFrames() {
	for _, frame := range s.pendingDataFrameWrites {
		if !s.RemainingFlowControlWindowIsEnough(int64(len(frame.Data))) {
			return // must stop here, because data frames must be sent in the right order
		}
		s.SendFrame(frame)
	}
}

func (s *stream) scheduleDataFrameWrite(frame *frames.DataFrame) {
	s.pendingDataFrameWrites = append(s.pendingDataFrameWrites, frame)
}

// called after firstPendingDataFrame() returned != nil, so we know len(s.pendingDataFrames) > 0
func (s *stream) popFirstPendingDataFrameWrite() *frames.DataFrame {
	result := s.pendingDataFrameWrites[0]
	s.pendingDataFrameWrites = s.pendingDataFrameWrites[1:] // TODO: Will pendingDataFrameWrites[0] ever get free() ?
	return result
}

func (s *stream) addRequestHeaders(headers ...hpack.HeaderField) {
	for _, header := range headers {
		s.requestHeaders = append(s.requestHeaders, header)
	}
}

func (s *stream) addResponseHeaders(headers ...hpack.HeaderField) {
	for _, header := range headers {
		s.responseHeaders = append(s.responseHeaders, header)
	}
}

func (s *stream) appendResponseBody(data []byte) {
	s.responseBody.Write(data)
}

func (s *stream) finalizeCommand() {
	if s.cmd != nil {
		if s.err != nil {
			s.cmd.CompleteWithError(s.err)
		} else {
			for _, header := range s.responseHeaders {
				s.cmd.Response.AddHeader(header.Name, header.Value)
			}
			s.cmd.Response.SetBody(s.responseBody.Bytes(), false)
			s.cmd.CompleteSuccessfully()
		}
	}
}

func (s *stream) RequestHeaders() []hpack.HeaderField {
	return s.requestHeaders
}

func (s *stream) ResponseBody() []byte {
	return s.responseBody.Bytes()
}

func (s *stream) StreamId() uint32 {
	return s.streamId
}

func (s *stream) GetState() streamstate.StreamState {
	return s.state
}

func (s *stream) SetState(state streamstate.StreamState) {
	s.state = state
}

func (s *stream) AssociateWithCommand(cmd *commands.HttpCommand) error {
	if s.cmd != nil {
		return fmt.Errorf("Trying to set more than one command for a stream.")
	}
	s.cmd = cmd
	if s.state == streamstate.CLOSED {
		s.finalizeCommand()
	}
	return nil
}
