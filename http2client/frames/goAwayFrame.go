package frames

import (
	"encoding/binary"
	"fmt"
)

type GoAwayFrame struct {
	StreamId     uint32
	LastStreamId uint32
	ErrorCode    ErrorCode
}

func NewGoAwayFrame(streamId uint32, lastStreamId uint32, errorCode ErrorCode) *GoAwayFrame {
	return &GoAwayFrame{
		StreamId:     streamId,
		LastStreamId: lastStreamId,
		ErrorCode:    errorCode,
	}
}

func DecodeGoAwayFrame(flags byte, streamId uint32, payload []byte, context *DecodingContext) (Frame, error) {
	if len(payload) < 8 {
		return nil, fmt.Errorf("FRAME_SIZE_ERROR: Received GOAWAY frame of length %v", len(payload))
	}
	lastStreamId := uint32_ignoreFirstBit(payload[0:4])
	errorCode := ErrorCode(binary.BigEndian.Uint32(payload[4:8]))
	return NewGoAwayFrame(streamId, lastStreamId, errorCode), nil
}

func (f *GoAwayFrame) Type() Type {
	return GOAWAY_TYPE
}

func (f *GoAwayFrame) Encode(context *EncodingContext) ([]byte, error) {
	payload := make([]byte, 8)
	binary.BigEndian.PutUint32(payload[0:4], f.LastStreamId)
	binary.BigEndian.PutUint32(payload[4:8], uint32(f.ErrorCode))
	return payload, nil
}

func (f *GoAwayFrame) GetStreamId() uint32 {
	return f.StreamId
}
