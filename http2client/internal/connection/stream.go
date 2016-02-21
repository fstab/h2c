package connection

import (
	"bytes"
	"fmt"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/internal/message"
	"github.com/fstab/h2c/http2client/internal/streamstate"
	"golang.org/x/net/http2/hpack"
	"os"
)

type Stream interface {
	StreamId() uint32
	GetState() streamstate.StreamState
	SetState(state streamstate.StreamState)
	//	ReceivedHeaders() map[string]string
	ReceivedData() []byte
	AssociateWithRequest(request message.HttpRequest) error

	// sendFrame doesn't mean the frame is sent directly. DATA frames can be postponed by flow control. However, this method will return immediately, postponed frames will be handled under the hood when a WINDOW_UPDATE is received.
	sendFrame(frame frames.Frame)
	receiveFrame(frame frames.Frame)
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
	receivedHeaders            []hpack.HeaderField
	receivedData               bytes.Buffer
	err                        *streamError // RST_STREAM sent or received.
	request                    message.HttpRequest
	initialReceiveWindowSize   int64
	remainingSendWindowSize    int64
	remainingReceiveWindowSize int64
	pendingDataFrameWrites     []*frames.DataFrame
	streamId                   uint32
	out                        frameWriter
}

type frameWriter interface {
	write(frame frames.Frame)
	remainingFlowControlWindowIsEnough(nBytesToWrite int64) bool
	decreaseFlowControlWindow(nBytesToWrite int64)
}

func newStream(streamId uint32, request message.HttpRequest, initialSendWindowSize uint32, initialReceiveWindowSize uint32, out frameWriter) *stream {
	return &stream{
		state:                      streamstate.IDLE,
		receivedHeaders:            make([]hpack.HeaderField, 0),
		streamId:                   streamId,
		request:                    request,
		remainingSendWindowSize:    int64(initialSendWindowSize),
		initialReceiveWindowSize:   int64(initialReceiveWindowSize),
		remainingReceiveWindowSize: int64(initialReceiveWindowSize),
		pendingDataFrameWrites:     make([]*frames.DataFrame, 0),
		out: out,
	}
}

func (s *stream) receiveFrame(frame frames.Frame) {
	wasClosedBefore := s.state == streamstate.CLOSED
	err := streamstate.HandleIncomingFrame(s, frame)
	if err != nil {
		s.closeWithError(err.ErrorCode, err.Message)
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
		s.finalizeRequest()
	}
}

func (s *stream) receiveDataFrame(frame *frames.DataFrame) {
	s.flowControlForIncomingDataFrame(frame)
	s.appendReceivedData(frame.Data)
}

func (s *stream) receiveHeadersFrame(frame *frames.HeadersFrame) {
	if !frame.EndHeaders {
		s.closeWithError(frames.REFUSED_STREAM, fmt.Sprintf("Unable to process %v without the END_HEADERS flag, because CONTINUATIONs are not implemented yet.", frame.Type()))
	} else {
		s.addReceivedHeaders(frame.Headers...)
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
		s.closeWithError(frames.REFUSED_STREAM, fmt.Sprintf("%v with multiple header frames not supported.", frame.Type()))
	} else {
		s.addReceivedHeaders(frame.Headers...)
	}
}

func (s *stream) notImplementedYet(frame frames.Frame) {
	fmt.Fprintf(os.Stderr, "Ignoring %v frame, because this frame type is not implemented yet.", frame.Type())
}

func (s *stream) closeWithError(errorCode frames.ErrorCode, msg string) {
	if s.state == streamstate.CLOSED {
		return
	}
	rstStream := frames.NewRstStreamFrame(s.streamId, errorCode)
	s.err = newStreamError("%v", msg)
	s.sendFrame(rstStream)
}

func (s *stream) sendFrame(frame frames.Frame) {
	wasClosedBefore := s.state == streamstate.CLOSED
	switch frame := frame.(type) {
	case *frames.DataFrame:
		size := int64(len(frame.Data))
		if s.remainingFlowControlWindowIsEnough(size) {
			s.decreaseFlowControlWindow(size)
			streamstate.HandleOutgoingFrame(s, frame)
			s.out.write(frame)
		} else {
			s.scheduleDataFrameWrite(frame)
		}
	default:
		streamstate.HandleOutgoingFrame(s, frame)
		s.out.write(frame)
	}
	if s.state == streamstate.CLOSED && !wasClosedBefore {
		s.finalizeRequest()
	}
}

func (s *stream) remainingFlowControlWindowIsEnough(nBytesToWrite int64) bool {
	return s.remainingReceiveWindowSize > nBytesToWrite && s.out.remainingFlowControlWindowIsEnough(nBytesToWrite)
}

func (s *stream) decreaseFlowControlWindow(nBytesToWrite int64) {
	s.out.decreaseFlowControlWindow(nBytesToWrite)
	s.remainingSendWindowSize -= nBytesToWrite
}

func (s *stream) receiveWindowUpdateFrame(frame *frames.WindowUpdateFrame) {
	// TODO: stream error if increment is 0.
	s.remainingSendWindowSize += int64(frame.WindowSizeIncrement)
	s.processPendingDataFrames()
}

// Just a quick implementation to make large downloads work.
// Should be replaced with a more sophisticated flow control strategy
func (s *stream) flowControlForIncomingDataFrame(frame *frames.DataFrame) {
	threshold := int64(2 << 13) // size of one frame
	s.remainingReceiveWindowSize -= int64(len(frame.Data))
	if s.remainingReceiveWindowSize < threshold {
		diff := s.initialReceiveWindowSize - s.remainingReceiveWindowSize
		s.remainingReceiveWindowSize += diff
		s.sendFrame(frames.NewWindowUpdateFrame(s.streamId, uint32(diff)))
	}
}

func (s *stream) processPendingDataFrames() {
	for _, frame := range s.pendingDataFrameWrites {
		if !s.remainingFlowControlWindowIsEnough(int64(len(frame.Data))) {
			return // must stop here, because data frames must be sent in the right order
		}
		s.sendFrame(frame)
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

func (s *stream) addReceivedHeaders(headers ...hpack.HeaderField) {
	for _, header := range headers {
		s.receivedHeaders = append(s.receivedHeaders, header)
	}
}

func (s *stream) appendReceivedData(data []byte) {
	s.receivedData.Write(data)
}

func (s *stream) finalizeRequest() {
	if s.request != nil {
		if s.err != nil {
			s.request.CompleteWithError(s.err)
		} else {
			s.request.CompleteSuccessfully(s.makeResponse())
		}
	}
}

func (s *stream) makeResponse() message.HttpResponse {
	resp := message.NewResponse()
	for _, header := range s.receivedHeaders {
		resp.AddHeader(header.Name, header.Value)
	}
	resp.AddData(s.receivedData.Bytes(), false)
	return resp
}

func (s *stream) ReceivedData() []byte {
	return s.receivedData.Bytes()
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

func (s *stream) AssociateWithRequest(request message.HttpRequest) error {
	if s.request != nil {
		return fmt.Errorf("Trying to set more than one request for a stream.")
	}
	s.request = request
	if s.state == streamstate.CLOSED {
		s.finalizeRequest()
	}
	return nil
}
