package streamstate

import (
	"github.com/fstab/h2c/http2client/frames"
	"testing"
)

func TestStringer(t *testing.T) {
	if RESERVED_REMOTE.String() != "reserved (remote)" {
		t.Fail()
	}
}

func TestIn(t *testing.T) {
	if !OPEN.In(RESERVED_REMOTE, OPEN) {
		t.Fail()
	}
}

func TestNotIn(t *testing.T) {
	if OPEN.In(IDLE) {
		t.Fail()
	}
}

func TestGetRequest(t *testing.T) {
	stream := &mockStream{
		state: IDLE,
	}
	HandleOutgoingFrame(stream, newHeadersFrame(true)) // GET request
	assertState(t, stream, HALF_CLOSED_LOCAL)
	err := HandleIncomingFrame(stream, newHeadersFrame(false)) // 202 OK
	assertState(t, stream, HALF_CLOSED_LOCAL)
	assertNil(t, err)
	err = HandleIncomingFrame(stream, newDataFrame(false)) // DATA part 1
	assertState(t, stream, HALF_CLOSED_LOCAL)
	assertNil(t, err)
	err = HandleIncomingFrame(stream, newDataFrame(true)) // DATA part 2
	assertState(t, stream, CLOSED)
	assertNil(t, err)
}

func TestPostRequest(t *testing.T) {
	stream := &mockStream{
		state: IDLE,
	}
	HandleOutgoingFrame(stream, newHeadersFrame(false)) // POST request
	assertState(t, stream, OPEN)
	HandleOutgoingFrame(stream, newDataFrame(false)) // DATA part 1
	assertState(t, stream, OPEN)
	HandleOutgoingFrame(stream, newDataFrame(true)) // DATA part 2
	assertState(t, stream, HALF_CLOSED_LOCAL)
	err := HandleIncomingFrame(stream, newHeadersFrame(true)) // OK
	assertState(t, stream, CLOSED)
	assertNil(t, err)
}

func TestPushPromise(t *testing.T) {
	stream := &mockStream{
		state: IDLE,
	}
	err := HandleIncomingFrame(stream, newPushPromiseFrame())
	assertState(t, stream, RESERVED_REMOTE)
	assertNil(t, err)
	err = HandleIncomingFrame(stream, newHeadersFrame(false)) // 200 OK
	assertState(t, stream, HALF_CLOSED_LOCAL)
	assertNil(t, err)
	err = HandleIncomingFrame(stream, newDataFrame(true)) // DATA
	assertState(t, stream, CLOSED)
	assertNil(t, err)
}

func TestPushPromiseRejected(t *testing.T) {
	stream := &mockStream{
		state: IDLE,
	}
	HandleIncomingFrame(stream, newPushPromiseFrame())
	assertState(t, stream, RESERVED_REMOTE)
	HandleOutgoingFrame(stream, newRstStreamFrame()) // Reject push promise
	assertState(t, stream, CLOSED)
	err := HandleIncomingFrame(stream, newHeadersFrame(false)) // Could be sent by server before RST_STREAM is received.
	assertState(t, stream, CLOSED)
	assertNotNil(t, err) // Should result in error. However, as the stream is already closed the client will not send another RST_STREAM when this error occurs.
}

func newHeadersFrame(endStream bool) *frames.HeadersFrame {
	result := frames.NewHeadersFrame(0, nil)
	result.EndStream = endStream
	return result
}

func newDataFrame(endStream bool) *frames.DataFrame {
	return frames.NewDataFrame(0, nil, endStream)
}

func newPushPromiseFrame() *frames.PushPromiseFrame {
	return frames.NewPushPromiseFrame(0, 0, nil)
}

func newRstStreamFrame() *frames.RstStreamFrame {
	return frames.NewRstStreamFrame(0, frames.STREAM_CLOSED)
}

func assertState(t *testing.T, stream *mockStream, state StreamState) {
	if stream.state != state {
		t.Fatalf("Expected stream state %v, but got %v.", state, stream.state)
	}
}

func assertNil(t *testing.T, err *StreamStateError) {
	if err != nil {
		t.Fatalf("Unexpected error: %v", err.Message)
	}
}

func assertNotNil(t *testing.T, err *StreamStateError) {
	if err == nil {
		t.Fatalf("Expected error, but error was nil.")
	}
}

type mockStream struct {
	state StreamState
}

func (s *mockStream) GetState() StreamState {
	return s.state
}

func (s *mockStream) SetState(state StreamState) {
	s.state = state
}
