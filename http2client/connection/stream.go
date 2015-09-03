package connection

import (
	"bytes"
	"fmt"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/h2c/http2client/util"
	"github.com/fstab/http2/hpack"
	"os"
)

type Stream interface {
	StreamId() uint32
	Write(frame frames.Frame, timeoutInSeconds int) error
	Error() error
	ReceivedHeaders() map[string]string
	ReceivedData() []byte
	SetOnClosedCallback(onClosed *util.AsyncTask)
}

type stream struct {
	receivedHeaders            map[string]string // TODO: Does not allow multiple headers with same name
	receivedData               bytes.Buffer
	err                        error // RST_STREAM received
	isClosed                   bool
	onClosed                   *util.AsyncTask
	remainingSendWindowSize    int64
	remainingReceiveWindowSize int64
	pendingDataFrameWrites     []*writeFrameRequest // only touched in the single threaded frame processing loop
	streamId                   uint32
	out                        chan *writeFrameRequest
}

func newStream(streamId uint32, onClosed *util.AsyncTask, initialSendWindowSize uint32, initialReceiveWindowSize uint32, out chan *writeFrameRequest) *stream {
	return &stream{
		receivedHeaders:            make(map[string]string),
		streamId:                   streamId,
		isClosed:                   false,
		onClosed:                   onClosed,
		remainingSendWindowSize:    int64(initialSendWindowSize),
		remainingReceiveWindowSize: int64(initialReceiveWindowSize),
		pendingDataFrameWrites:     make([]*writeFrameRequest, 0),
		out: out,
	}
}

func (s *stream) scheduleDataFrameWrite(frame *writeFrameRequest) {
	s.pendingDataFrameWrites = append(s.pendingDataFrameWrites, frame)
}

func (s *stream) firstPendingDataFrameWrite() *writeFrameRequest {
	if len(s.pendingDataFrameWrites) > 0 {
		return s.pendingDataFrameWrites[0]
	}
	return nil
}

// called after firstPendingDataFrame() returned != nil, so we know len(s.pendingDataFrames) > 0
func (s *stream) popFirstPendingDataFrameWrite() *writeFrameRequest {
	result := s.pendingDataFrameWrites[0]
	s.pendingDataFrameWrites = s.pendingDataFrameWrites[1:]
	return result
}

func (s *stream) addReceivedHeaders(headers ...hpack.HeaderField) {
	for _, header := range headers {
		s.receivedHeaders[header.Name] = header.Value
	}
}

func (s *stream) appendReceivedData(data []byte) {
	s.receivedData.Write(data)
}

func (s *stream) endStream() {
	s.isClosed = true
	if s.onClosed != nil {
		s.onClosed.CompleteSuccessfully()
	}
}

func (s *stream) Error() error {
	return s.err
}

func (s *stream) setError(err error) {
	s.err = err
}

func (s *stream) ReceivedHeaders() map[string]string {
	return s.receivedHeaders
}

func (s *stream) ReceivedData() []byte {
	return s.receivedData.Bytes()
}

func (s *stream) StreamId() uint32 {
	return s.streamId
}

func (s *stream) Write(frame frames.Frame, timeoutInSeconds int) error {
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

func (s *stream) SetOnClosedCallback(onClosed *util.AsyncTask) {
	if s.onClosed != nil {
		fmt.Fprintf(os.Stderr, "Trying to set more than one onClosed callback for a stream. This is a bug.")
		os.Exit(-1)
	}
	// This is not thread safe. What if the stream gets closed after we check IsClosed but before we set onClosed?
	if s.Error() != nil {
		onClosed.CompleteWithError(s.Error())
	} else if s.isClosed {
		onClosed.CompleteSuccessfully()
	} else {
		s.onClosed = onClosed
	}
}
