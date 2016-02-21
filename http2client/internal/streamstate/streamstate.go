package streamstate

import (
	"fmt"
	"github.com/fstab/h2c/http2client/frames"
)

type StreamState string

// Stream states as defined in RFC 7540 section 5.1
const (
	IDLE               StreamState = "idle"
	RESERVED_LOCAL     StreamState = "reserved (local)"
	RESERVED_REMOTE    StreamState = "reserved (remote)"
	OPEN               StreamState = "open"
	HALF_CLOSED_REMOTE StreamState = "half closed (remote)"
	HALF_CLOSED_LOCAL  StreamState = "half closed (local)"
	CLOSED             StreamState = "closed"
)

type StreamStateError struct {
	ErrorCode frames.ErrorCode
	Message   string
}

type stateful interface {
	GetState() StreamState
	SetState(state StreamState)
}

func HandleIncomingFrame(stream stateful, frame frames.Frame) *StreamStateError {
	switch frame := frame.(type) {
	case *frames.DataFrame:
		if !stream.GetState().In(OPEN, HALF_CLOSED_LOCAL) {
			return newStreamStateError("Received %v frame for stream in state %v.", frame.Type(), stream.GetState())
		}
		if frame.EndStream {
			if stream.GetState() == OPEN {
				stream.SetState(HALF_CLOSED_REMOTE)
			}
			if stream.GetState() == HALF_CLOSED_LOCAL {
				stream.SetState(CLOSED)
			}
		}
	case *frames.HeadersFrame:
		if !stream.GetState().In(OPEN, HALF_CLOSED_LOCAL, RESERVED_REMOTE) {
			return newStreamStateError("Received %v frame for stream in state %v.", frame.Type(), stream.GetState())
		}
		if stream.GetState() == RESERVED_REMOTE {
			stream.SetState(HALF_CLOSED_LOCAL)
		}
		// At this point, state cannot be RESERVED_REMOTE anymore.
		if frame.EndStream {
			if stream.GetState() == OPEN {
				stream.SetState(HALF_CLOSED_REMOTE)
			}
			if stream.GetState() == HALF_CLOSED_LOCAL {
				stream.SetState(CLOSED)
			}
		}
	case *frames.RstStreamFrame:
		stream.SetState(CLOSED)
	case *frames.PushPromiseFrame:
		// This is called for the promised stream, not for the associated stream.
		if !stream.GetState().In(IDLE) {
			return newStreamStateError("Received %v frame, but promised stream already exists in state %v.", frame.Type(), stream.GetState())
		}
		stream.SetState(RESERVED_REMOTE)
	}
	return nil
}

func HandleOutgoingFrame(stream stateful, frame frames.Frame) {
	state := stream.GetState()
	switch frame := frame.(type) {
	case *frames.DataFrame:
		state.MustBeIn(OPEN, HALF_CLOSED_REMOTE)
		if frame.EndStream {
			if stream.GetState() == OPEN {
				stream.SetState(HALF_CLOSED_LOCAL)
			}
			if stream.GetState() == HALF_CLOSED_REMOTE {
				stream.SetState(CLOSED)
			}
		}
	case *frames.HeadersFrame:
		state.MustBeIn(IDLE, OPEN, HALF_CLOSED_REMOTE) // RESERVED_LOCAL is only for servers, not for clients.
		if stream.GetState() == IDLE {
			stream.SetState(OPEN)
		}
		// At this point, state cannot be IDLE anymore.
		if frame.EndStream {
			if stream.GetState() == OPEN {
				stream.SetState(HALF_CLOSED_LOCAL)
			}
			if stream.GetState() == HALF_CLOSED_REMOTE {
				stream.SetState(CLOSED)
			}
		}
	case *frames.RstStreamFrame:
		state.MustNotBeIn(IDLE)
		stream.SetState(CLOSED)
	}
}

func newStreamStateError(format string, a ...interface{}) *StreamStateError {
	return &StreamStateError{
		frames.STREAM_CLOSED,
		fmt.Sprintf(format, a...),
	}
}

func (state StreamState) String() string {
	return string(state)
}

func (state StreamState) In(states ...StreamState) bool {
	for _, item := range states {
		if state == item {
			return true
		}
	}
	return false
}

func (state StreamState) MustBeIn(states ...StreamState) {
	if !state.In(states...) {
		panic(fmt.Sprintf("Stream is in illegal state %v. This is a bug.", state))
	}
}

func (state StreamState) MustNotBeIn(states ...StreamState) {
	if state.In(states...) {
		panic(fmt.Sprintf("Stream is in illegal state %v. This is a bug.", state))
	}
}
