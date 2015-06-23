package frames

import (
	"bytes"
	"fmt"
)

const (
	DATA_FLAG_END_STREAM Flag = 0x01
	DATA_FLAG_PADDED     Flag = 0x08
)

type DataFrame struct {
	StreamId  uint32
	Data      []byte
	EndStream bool
}

func NewDataFrame(streamId uint32, data []byte, endStream bool) *DataFrame {
	return &DataFrame{
		StreamId:  streamId,
		Data:      data,
		EndStream: endStream,
	}
}

func DecodeDataFrame(flags byte, streamId uint32, framePayload []byte, context *DecodingContext) (Frame, error) {
	endStream := flags&0x01 != 0
	padded := flags&0x08 != 0
	payload := framePayload
	if padded {
		return nil, fmt.Errorf("Padded data frames not implemented yet.")
	}
	return NewDataFrame(streamId, payload, endStream), nil
}

func (f *DataFrame) Type() Type {
	return DATA_TYPE
}

func (f *DataFrame) flags() []Flag {
	flags := make([]Flag, 0)
	if f.EndStream {
		flags = append(flags, DATA_FLAG_END_STREAM)
	}
	return flags
}

func (f *DataFrame) Encode(context *EncodingContext) ([]byte, error) {
	length := uint32(len(f.Data))
	var result bytes.Buffer
	result.Write(encodeHeader(f.Type(), f.StreamId, length, f.flags()))
	result.Write(f.Data)
	return result.Bytes(), nil
}

func (f *DataFrame) String() string {
	return fmt.Sprintf("DATA(%v)", f.StreamId)
}
