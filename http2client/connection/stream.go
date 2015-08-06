package connection

import (
	"bytes"
	"fmt"
	"github.com/bradfitz/http2/hpack"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/util"
)

type Stream struct {
	receivedHeaders         map[string]string // TODO: Does not allow multiple headers with same name
	receivedData            bytes.Buffer
	err                     error // RST_STREAM received
	onClosed                *util.AsyncTask
	remainingSendWindowSize int64
	pendingDataFrameWrites  []*writeFrameRequest // only touched in the single threaded frame processing loop
	streamId                uint32
	out                     chan *writeFrameRequest
}

func newStream(streamId uint32, onClosed *util.AsyncTask, initialSendWindowSize uint32, out chan *writeFrameRequest) *Stream {
	return &Stream{
		receivedHeaders:         make(map[string]string),
		streamId:                streamId,
		onClosed:                onClosed,
		remainingSendWindowSize: int64(initialSendWindowSize),
		pendingDataFrameWrites:  make([]*writeFrameRequest, 0),
		out: out,
	}
}

func (s *Stream) scheduleDataFrameWrite(frame *writeFrameRequest) {
	s.pendingDataFrameWrites = append(s.pendingDataFrameWrites, frame)
}

func (s *Stream) firstPendingDataFrameWrite() *writeFrameRequest {
	if len(s.pendingDataFrameWrites) > 0 {
		return s.pendingDataFrameWrites[0]
	}
	return nil
}

// called after firstPendingDataFrame() returned != nil, so we know len(s.pendingDataFrames) > 0
func (s *Stream) popFirstPendingDataFrameWrite() *writeFrameRequest {
	result := s.pendingDataFrameWrites[0]
	s.pendingDataFrameWrites = s.pendingDataFrameWrites[1:]
	return result
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
		s.onClosed.CompleteSuccessfully()
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

func (s *Stream) Write(frame frames.Frame, timeoutInSeconds int) error {
	if frame.GetStreamId() != s.streamId {
		return fmt.Errorf("Tried to write frame with stream id %v to stream with id %v. This is a bug.", frame.GetStreamId(), s.streamId)
	}
	task := util.NewAsyncTask()
	s.out <- &writeFrameRequest{
		task:  task,
		frame: frame,
	}
	return task.WaitForCompletion(timeoutInSeconds)
}
