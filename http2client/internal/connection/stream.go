package connection

import (
	"bytes"
	"fmt"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/internal/message"
	"golang.org/x/net/http2/hpack"
	"os"
)

type Stream interface {
	StreamId() uint32
	//	ReceivedHeaders() map[string]string
	ReceivedData() []byte
	AssociateWithRequest(request message.HttpRequest) error
	scheduleDataFrameWrite(frame *frames.DataFrame)
}

// Stream states as defined in RFC 7540 section 5.1
type state string

const (
	IDLE               state = "idle"
	RESERVED_LOCAL     state = "reserved (local)"
	RESERVED_REMOTE    state = "reserved (remote)"
	OPEN               state = "open"
	HALF_CLOSED_REMOTE state = "half closed (remote)"
	HALF_CLOSED_LOCAL  state = "half closed (local)"
	CLOSED             state = "closed"
)

type stream struct {
	state                      state
	receivedHeaders            []hpack.HeaderField
	receivedData               bytes.Buffer
	err                        error // RST_STREAM received
	isClosed                   bool
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
}

func newStream(streamId uint32, request message.HttpRequest, initialSendWindowSize uint32, initialReceiveWindowSize uint32, out frameWriter) *stream {
	return &stream{
		state:                      IDLE,
		receivedHeaders:            make([]hpack.HeaderField, 0),
		streamId:                   streamId,
		isClosed:                   false,
		request:                    request,
		remainingSendWindowSize:    int64(initialSendWindowSize),
		initialReceiveWindowSize:   int64(initialReceiveWindowSize),
		remainingReceiveWindowSize: int64(initialReceiveWindowSize),
		pendingDataFrameWrites:     make([]*frames.DataFrame, 0),
		out: out,
	}
}

func (s *stream) handleIncomingFrame(frame frames.Frame) {
	switch frame := frame.(type) {
	case *frames.HeadersFrame:
		s.addReceivedHeaders(frame.Headers...)
		// TODO: continuations
		// TODO: error handling
		if frame.EndStream {
			s.endStream()
		}
	case *frames.DataFrame:
		s.flowControlForIncomingDataFrame(frame)
		s.appendReceivedData(frame.Data)
		if frame.EndStream {
			s.endStream()
		}
	case *frames.PushPromiseFrame:
		if !frame.EndHeaders {
			fmt.Fprintf(os.Stderr, "ERROR: Push promise with multiple header frames not supported.")
			return
		}
	case *frames.RstStreamFrame:
		s.setError(fmt.Errorf("ERROR: Server sent RST_STREAM with error code %v.", frame.ErrorCode.String()))
		s.endStream()
	default:
		// TODO: error handling
		fmt.Fprintf(os.Stderr, "Received unknown frame type %v\n", frame.Type())
	}
}

// Just a quick implementation to make large downloads work.
// Should be replaced with a more sophisticated flow control strategy
func (s *stream) flowControlForIncomingDataFrame(frame *frames.DataFrame) {
	threshold := int64(2 << 13) // size of one frame
	s.remainingReceiveWindowSize -= int64(len(frame.Data))
	if s.remainingReceiveWindowSize < threshold {
		diff := s.initialReceiveWindowSize - s.remainingReceiveWindowSize
		s.remainingReceiveWindowSize += diff
		s.out.write(frames.NewWindowUpdateFrame(s.streamId, uint32(diff)))
	}
}

func (s *stream) scheduleDataFrameWrite(frame *frames.DataFrame) {
	s.pendingDataFrameWrites = append(s.pendingDataFrameWrites, frame)
}

func (s *stream) firstPendingDataFrameWrite() *frames.DataFrame {
	if len(s.pendingDataFrameWrites) > 0 {
		return s.pendingDataFrameWrites[0]
	}
	return nil
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

func (s *stream) endStream() {
	s.isClosed = true
	if s.request != nil {
		s.request.CompleteSuccessfully(s.makeResponse())
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

func (s *stream) setError(err error) {
	s.err = err
}

func (s *stream) ReceivedData() []byte {
	return s.receivedData.Bytes()
}

func (s *stream) StreamId() uint32 {
	return s.streamId
}

func (s *stream) AssociateWithRequest(request message.HttpRequest) error {
	if s.request != nil {
		return fmt.Errorf("Trying to set more than one request for a stream.")
	}
	s.request = request
	if s.err != nil {
		s.request.CompleteWithError(s.err)
	} else if s.isClosed {
		s.request.CompleteSuccessfully(s.makeResponse())
	}
	return nil
}
