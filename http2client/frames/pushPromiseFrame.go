package frames

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/fstab/http2/hpack"
)

const (
	PUSH_PROMISE_FLAG_END_HEADERS Flag = 0x04
	PUSH_PROMISE_FLAG_PADDED      Flag = 0x08
)

type PushPromiseFrame struct {
	StreamId         uint32
	EndHeaders       bool
	PromisedStreamId uint32
	Headers          []hpack.HeaderField
}

func NewPushPromiseFrame(streamId uint32, promisedStreamId uint32, headers []hpack.HeaderField) *PushPromiseFrame {
	return &PushPromiseFrame{
		StreamId:         streamId,
		EndHeaders:       true,
		PromisedStreamId: promisedStreamId,
		Headers:          headers,
	}
}

func DecodePushPromiseFrame(flags byte, streamId uint32, payload []byte, context *DecodingContext) (Frame, error) {
	endHeaders := PUSH_PROMISE_FLAG_END_HEADERS.isSet(flags)
	padded := PUSH_PROMISE_FLAG_PADDED.isSet(flags)
	var err error
	if padded {
		payload, err = stripPadding(payload)
		if err != nil {
			return nil, err
		}
	}
	promisedStreamId := uint32_ignoreFirstBit(payload[0:4])
	headers, err := context.decoder.DecodeFull(payload[4:])
	if err != nil {
		return nil, fmt.Errorf("Error decoding header fields: ", err.Error())
	}
	return &PushPromiseFrame{
		StreamId:         streamId,
		PromisedStreamId: promisedStreamId,
		EndHeaders:       endHeaders,
		Headers:          headers,
	}, nil
}

func (f *PushPromiseFrame) Type() Type {
	return PUSH_PROMISE_TYPE
}

func (f *PushPromiseFrame) flags() []Flag {
	flags := make([]Flag, 0)
	if f.EndHeaders {
		flags = append(flags, HEADERS_FLAG_END_HEADERS)
	}
	return flags
}

func (f *PushPromiseFrame) Encode(context *EncodingContext) ([]byte, error) {
	defer context.headerBlockBuffer.Reset()
	for _, header := range f.Headers {
		err := context.encoder.WriteField(header)
		if err != nil {
			return nil, fmt.Errorf("Failed to encode HEADER frame: %v", err)
		}
	}
	promisedStreamId := make([]byte, 4)
	binary.BigEndian.PutUint32(promisedStreamId, f.PromisedStreamId)
	headerBlockFragment := context.headerBlockBuffer.Bytes()
	length := uint32(len(headerBlockFragment)) + uint32(len(promisedStreamId))
	var result bytes.Buffer
	result.Write(encodeHeader(f.Type(), f.StreamId, length, f.flags()))
	result.Write(promisedStreamId)
	result.Write(headerBlockFragment)
	return result.Bytes(), nil
}

func (f *PushPromiseFrame) GetStreamId() uint32 {
	return f.StreamId
}
