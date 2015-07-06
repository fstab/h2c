package connection

import (
	"bytes"
	"fmt"
	"github.com/bradfitz/http2/hpack"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/util"
)

type Stream struct {
	receivedHeaders                map[string]string // TODO: Does not allow multiple headers with same name
	receivedData                   bytes.Buffer
	err                            error // RST_STREAM received
	onClosed                       *util.AsyncTask
	remainingFlowControlWindowSize int64
	streamId                       uint32
	out                            chan (frames.Frame)
}

func newStream(streamId uint32, onClosed *util.AsyncTask, initialFlowControlWindowSize uint32, out chan (frames.Frame)) *Stream {
	return &Stream{
		streamId:                       streamId,
		receivedHeaders:                make(map[string]string),
		onClosed:                       onClosed,
		remainingFlowControlWindowSize: int64(initialFlowControlWindowSize),
		out: out,
	}
}

func (s *Stream) addReceivedHeaders(headers ...hpack.HeaderField) {
	for _, header := range headers {
		s.receivedHeaders[header.Name] = header.Value
	}
}

func (s *Stream) appendReceivedData(data []byte) {
	s.receivedData.Write(data)
}

func (s *Stream) endStream() {
	if s.onClosed != nil {
		s.onClosed.Complete()
	}
}

func (s *Stream) Error() error {
	return s.err
}

func (s *Stream) setError(err error) {
	s.err = err
}

func (s *Stream) ReceivedHeaders() map[string]string {
	return s.receivedHeaders
}

func (s *Stream) ReceivedData() []byte {
	return s.receivedData.Bytes()
}

func (s *Stream) StreamId() uint32 {
	return s.streamId
}

func (s *Stream) Write(frame frames.Frame) error {
	if frame.GetStreamId() != s.streamId {
		return fmt.Errorf("Tried to write frame with stream id %v to stream with id %v. This is a bug.", frame.GetStreamId(), s.streamId)
	}
	s.out <- frame
	return nil
}
