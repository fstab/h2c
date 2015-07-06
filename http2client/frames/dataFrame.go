package frames

import (
	"bytes"
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
	endStream := DATA_FLAG_END_STREAM.isSet(flags)
	padded := DATA_FLAG_PADDED.isSet(flags)
	payload := framePayload
	var err error
	if padded {
		payload, err = stripPadding(payload)
		if err != nil {
			return nil, err
		}
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

func (f *DataFrame) GetStreamId() uint32 {
	return f.StreamId
}
