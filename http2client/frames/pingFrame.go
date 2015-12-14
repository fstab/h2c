package frames

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	ACK Flag = 0x01
)

type PingFrame struct {
	StreamId uint32
	Payload  uint64
	Ack      bool
}

func NewPingFrame(streamId uint32, payload uint64, ack bool) *PingFrame {
	return &PingFrame{
		StreamId: streamId,
		Payload:  payload,
		Ack:      ack,
	}
}

func DecodePingFrame(flags byte, streamId uint32, payload []byte, context *DecodingContext) (Frame, error) {
	if streamId != 0 {
		return nil, fmt.Errorf("Connection error: Received ping frame with stream id %v.", streamId)
	}
	if len(payload) != 8 {
		return nil, fmt.Errorf("Connection error: Received ping frame with %v bytes payload.", len(payload))
	}
	return NewPingFrame(streamId, binary.BigEndian.Uint64(payload), ACK.isSet(flags)), nil
}

func (f *PingFrame) Type() Type {
	return PING_TYPE
}

func (f *PingFrame) flags() []Flag {
	flags := make([]Flag, 0)
	if f.Ack {
		flags = append(flags, ACK)
	}
	return flags
}

func (f *PingFrame) Encode(context *EncodingContext) ([]byte, error) {
	payloadBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(payloadBytes, f.Payload)

	length := uint32(8)
	var result bytes.Buffer
	result.Write(encodeHeader(f.Type(), f.StreamId, length, f.flags()))
	result.Write(payloadBytes)
	return result.Bytes(), nil
}

func (f *PingFrame) GetStreamId() uint32 {
	return f.StreamId
}
