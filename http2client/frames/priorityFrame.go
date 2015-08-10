package frames

import (
	"encoding/binary"
	"fmt"
)

type PriorityFrame struct {
	StreamId           uint32
	StreamDependencyId uint32
	weight             uint8
	exclusive          bool
}

func NewPriorityFrame(streamId uint32, streamDependencyId uint32, weight uint8, exclusive bool) *PriorityFrame {
	return &PriorityFrame{
		StreamId:           streamId,
		StreamDependencyId: streamDependencyId,
		weight:             weight,
		exclusive:          exclusive,
	}
}

func DecodePriorityFrame(flags byte, streamId uint32, payload []byte, context *DecodingContext) (Frame, error) {
	if len(payload) != 5 {
		return nil, fmt.Errorf("FRAME_SIZE_ERROR: Received PRIORITY frame of length %v", len(payload))
	}
	streamDependencyId := streamDependencyId(payload[0:4])
	weight := payload[4]
	exclusive := payload[0]&0x80 == 1
	return NewPriorityFrame(streamId, streamDependencyId, weight, exclusive), nil
}

func streamDependencyId(payload []byte) uint32 {
	buffer := make([]byte, 4)
	copy(buffer, payload[0:4])
	buffer[0] = buffer[0] & 0x7F // clear reserved bit
	return binary.BigEndian.Uint32(buffer)
}

func (f *PriorityFrame) Type() Type {
	return PRIORITY_TYPE
}

func (f *PriorityFrame) Encode(context *EncodingContext) ([]byte, error) {
	payload := make([]byte, 5)
	binary.BigEndian.PutUint32(payload[0:4], f.StreamDependencyId)
	payload[4] = f.weight
	if f.exclusive {
		payload[0] |= 0x80
	}
	return payload, nil
}

func (f *PriorityFrame) GetStreamId() uint32 {
	return f.StreamId
}