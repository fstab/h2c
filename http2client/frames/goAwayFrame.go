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
	lastStreamId := readLastStreamId(payload[0:4])
	errorCode := ErrorCode(binary.BigEndian.Uint32(payload[4:8]))
	return NewGoAwayFrame(streamId, lastStreamId, errorCode), nil
}

func readLastStreamId(payload []byte) uint32 {
	buffer := make([]byte, 4)
	copy(buffer, payload[0:4])
	buffer[0] = buffer[0] & 0x7F // clear reserved bit
	return binary.BigEndian.Uint32(buffer)
}

func (f *GoAwayFrame) Type() Type {
	return GOAWAY_TYPE
}

func (f *GoAwayFrame) Encode(context *EncodingContext) ([]byte, error) {
	result := make([]byte, 4)
	binary.BigEndian.PutUint32(result, uint32(f.ErrorCode))
	return result, nil
}
