package connection

import (
	"bytes"
	"fmt"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/internal/message"
	"golang.org/x/net/http2/hpack"
)

type Stream interface {
	StreamId() uint32
	//	ReceivedHeaders() map[string]string
	ReceivedData() []byte
	AssociateWithRequest(request message.HttpRequest) error
	scheduleDataFrameWrite(frame *frames.DataFrame)
}

type stream struct {
	receivedHeaders            []hpack.HeaderField
	receivedData               bytes.Buffer
	err                        error // RST_STREAM received
	isClosed                   bool
	request                    message.HttpRequest
	remainingSendWindowSize    int64
	remainingReceiveWindowSize int64
	pendingDataFrameWrites     []*frames.DataFrame
	streamId                   uint32
}

func newStream(streamId uint32, request message.HttpRequest, initialSendWindowSize uint32, initialReceiveWindowSize uint32) *stream {
	return &stream{
		receivedHeaders:            make([]hpack.HeaderField, 0),
		streamId:                   streamId,
		isClosed:                   false,
		request:                    request,
		remainingSendWindowSize:    int64(initialSendWindowSize),
		remainingReceiveWindowSize: int64(initialReceiveWindowSize),
		pendingDataFrameWrites:     make([]*frames.DataFrame, 0),
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
