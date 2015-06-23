package frames

import (
	"bytes"
	"fmt"
	"github.com/bradfitz/http2/hpack"
)

const (
	HEADERS_FLAG_END_STREAM  Flag = 0x01
	HEADERS_FLAG_END_HEADERS Flag = 0x04
	HEADERS_FLAG_PADDED      Flag = 0x08
	HEADERS_FLAG_PRIORITY    Flag = 0x20
)

type HeadersFrame struct {
	StreamId   uint32
	EndStream  bool
	EndHeaders bool
	Priority   bool
	Headers    []hpack.HeaderField
}

func NewHeadersFrame(streamId uint32, headers []hpack.HeaderField) *HeadersFrame {
	return &HeadersFrame{
		StreamId:   streamId,
		Headers:    headers,
		EndStream:  true,
		EndHeaders: true,
		Priority:   false,
	}
}

func DecodeHeadersFrame(flags byte, streamId uint32, payload []byte, context *DecodingContext) (Frame, error) {
	endStream := HEADERS_FLAG_END_STREAM.isSet(flags)
	endHeaders := HEADERS_FLAG_END_HEADERS.isSet(flags)
	padded := HEADERS_FLAG_PADDED.isSet(flags)
	priority := HEADERS_FLAG_PRIORITY.isSet(flags)
	if padded {
		return nil, fmt.Errorf("Padded header frames not implemented yet.")
	}
	if priority {
		return nil, fmt.Errorf("Priority headers not implemented yet.")
	}
	headers, err := context.decoder.DecodeFull(payload)
	if err != nil {
		return nil, fmt.Errorf("Error decoding header fields: ", err.Error())
	}
	return &HeadersFrame{
		StreamId:   streamId,
		EndStream:  endStream,
		EndHeaders: endHeaders,
		Priority:   priority,
		Headers:    headers,
	}, nil
}

func (f *HeadersFrame) Type() Type {
	return HEADERS_TYPE
}

func (f *HeadersFrame) flags() []Flag {
	flags := make([]Flag, 0)
	if f.EndStream {
		flags = append(flags, HEADERS_FLAG_END_STREAM)
	}
	if f.EndHeaders {
		flags = append(flags, HEADERS_FLAG_END_HEADERS)
	}
	return flags
}

func (f *HeadersFrame) Encode(context *EncodingContext) ([]byte, error) {
	defer context.headerBlockBuffer.Reset()
	for _, header := range f.Headers {
		err := context.encoder.WriteField(header)
		if err != nil {
			return nil, fmt.Errorf("Failed to encode HEADER frame: %v", err)
		}
	}
	headerBlockFragment := context.headerBlockBuffer.Bytes()
	length := uint32(len(headerBlockFragment))
	var result bytes.Buffer
	result.Write(encodeHeader(f.Type(), f.StreamId, length, f.flags()))
	result.Write(headerBlockFragment)
	return result.Bytes(), nil
}

func (f *HeadersFrame) String() string {
	return fmt.Sprintf("HEADERS(%v)", f.StreamId)
}
