package frames

import (
	"encoding/binary"
	"fmt"
)

type RstStreamFrame struct {
	StreamId  uint32
	ErrorCode ErrorCode
}

func NewRstStreamFrame(streamId uint32, errorCode ErrorCode) *RstStreamFrame {
	return &RstStreamFrame{
		StreamId:  streamId,
		ErrorCode: errorCode,
	}
}

func DecodeRstStreamFrame(flags byte, streamId uint32, payload []byte, context *DecodingContext) (Frame, error) {
	if len(payload) != 4 {
		return nil, fmt.Errorf("FRAME_SIZE_ERROR: Received RST_STREAM frame of length %v", len(payload))
	}
	return NewRstStreamFrame(streamId, ErrorCode(binary.BigEndian.Uint32(payload))), nil
}

func (f *RstStreamFrame) Type() Type {
	return RST_STREAM_TYPE
}

func (f *RstStreamFrame) Encode(context *EncodingContext) ([]byte, error) {
	result := make([]byte, 4)
	binary.BigEndian.PutUint32(result, uint32(f.ErrorCode))
	return result, nil
}
