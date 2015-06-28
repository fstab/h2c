package frames

import (
	"github.com/bradfitz/http2/hpack"
	"reflect"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	headers := []hpack.HeaderField{
		hpack.HeaderField{Name: ":authority", Value: "localhost"},
		hpack.HeaderField{Name: ":method", Value: "GET"},
		hpack.HeaderField{Name: ":path", Value: "/index.html"},
		hpack.HeaderField{Name: ":scheme", Value: "https"},
	}
	var flags byte
	HEADERS_FLAG_END_STREAM.set(&flags)
	HEADERS_FLAG_END_HEADERS.set(&flags)
	frame := NewHeadersFrame(31, headers)
	data, err := frame.Encode(NewEncodingContext())
	if err != nil {
		t.Error("Encoding error:", err.Error())
	}
	frameHeader := DecodeHeader(data[0:9])
	result, err := DecodeHeadersFrame(frameHeader.Flags, frameHeader.StreamId, data[9:], NewDecodingContext())
	if err != nil {
		t.Error("Decoding error:", err.Error())
	}
	if !reflect.DeepEqual(frame, result) {
		t.Error("Result does not equal expected frame.")
	}
}
