package frames

import (
	"encoding/binary"
	"fmt"
)

type PriorityFrame struct {
	StreamId           uint32
	StreamDependencyId uint32
	Weight             uint8
	Exclusive          bool
}

func NewPriorityFrame(streamId uint32, streamDependencyId uint32, weight uint8, exclusive bool) *PriorityFrame {
	return &PriorityFrame{
		StreamId:           streamId,
		StreamDependencyId: streamDependencyId,
		Weight:             weight,
		Exclusive:          exclusive,
	}
}

func DecodePriorityFrame(flags byte, streamId uint32, payload []byte, context *DecodingContext) (Frame, error) {
	if len(payload) != 5 {
		return nil, fmt.Errorf("FRAME_SIZE_ERROR: Received PRIORITY frame of length %v", len(payload))
	}
	streamDependencyId := uint32_ignoreFirstBit(payload[0:4])
	weight := payload[4]
	exclusive := payload[0]&0x80 == 1
	return NewPriorityFrame(streamId, streamDependencyId, weight, exclusive), nil
}

func (f *PriorityFrame) Type() Type {
	return PRIORITY_TYPE
}

func (f *PriorityFrame) Encode(context *EncodingContext) ([]byte, error) {
	payload := make([]byte, 5)
	binary.BigEndian.PutUint32(payload[0:4], f.StreamDependencyId)
	payload[4] = f.Weight
	if f.Exclusive {
		payload[0] |= 0x80
	}
	return payload, nil
}

func (f *PriorityFrame) GetStreamId() uint32 {
	return f.StreamId
}
