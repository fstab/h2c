package frames

import (
	"bytes"
	"golang.org/x/net/http2/hpack"
)

type EncodingContext struct {
	headerBlockBuffer bytes.Buffer
	encoder           *hpack.Encoder
}

type DecodingContext struct {
	decoder *hpack.Decoder
}

func NewDecodingContext() *DecodingContext {
	return &DecodingContext{
		decoder: hpack.NewDecoder(4096, func(f hpack.HeaderField) {}),
	}
}

func NewEncodingContext() *EncodingContext {
	result := &EncodingContext{}
	result.encoder = hpack.NewEncoder(&result.headerBlockBuffer)
	return result
}
