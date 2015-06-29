package frames

import (
	"encoding/binary"
	"github.com/bradfitz/http2/hpack"
	"reflect"
	"testing"
)

func makeExampleFrame() *HeadersFrame {
	headers := []hpack.HeaderField{
		hpack.HeaderField{Name: ":authority", Value: "localhost"},
		hpack.HeaderField{Name: ":method", Value: "GET"},
		hpack.HeaderField{Name: ":path", Value: "/index.html"},
		hpack.HeaderField{Name: ":scheme", Value: "https"},
	}
	return NewHeadersFrame(31, headers)
}

func TestEncodeDecode(t *testing.T) {
	frame := makeExampleFrame()
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

func addPadding(paddingLength int, data []byte) []byte {
	result := make([]byte, len(data)+paddingLength+1)
	copy(result[0:9], data[0:9])
	HEADERS_FLAG_PADDED.set(&result[4])
	result[9] = byte(paddingLength)
	copy(result[10:], data[9:])
	updateLength(result)
	return result
}

func TestPadding(t *testing.T) {
	frame := makeExampleFrame()
	data, err := frame.Encode(NewEncodingContext())
	if err != nil {
		t.Error("Encoding error:", err.Error())
	}
	for i := 0; i < 255; i++ {
		paddedData := addPadding(i, data)
		frameHeader := DecodeHeader(paddedData[0:9])
		result, err := DecodeHeadersFrame(frameHeader.Flags, frameHeader.StreamId, paddedData[9:], NewDecodingContext())
		if err != nil {
			t.Error("Decoding error:", err.Error())
		}
		if !reflect.DeepEqual(frame, result) {
			t.Error("Result does not equal expected frame.")
		}
	}
}

func addPriority(data []byte) []byte {
	// priority is currently ignored, so just pre-pend 5 zero bytes.
	result := make([]byte, len(data)+5)
	copy(result[0:9], data[0:9])
	HEADERS_FLAG_PRIORITY.set(&result[4])
	copy(result[14:], data[9:])
	updateLength(result)
	return result
}

func updateLength(data []byte) {
	length := len(data) - 9
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, uint32(length))
	copy(data[0:3], bytes[1:4])
}

func TestPriority(t *testing.T) {
	frame := makeExampleFrame()
	data, err := frame.Encode(NewEncodingContext())
	if err != nil {
		t.Error("Encoding error:", err.Error())
	}
	withPriority := addPriority(data)
	frameHeader := DecodeHeader(withPriority[0:9])
	result, err := DecodeHeadersFrame(frameHeader.Flags, frameHeader.StreamId, withPriority[9:], NewDecodingContext())
	if err != nil {
		t.Error("Decoding error:", err.Error())
	}
	frame.Priority = true // to compare with result
	if !reflect.DeepEqual(frame, result) {
		t.Error("Result does not equal expected frame.")
	}
}

func TestPaddingAndPriority(t *testing.T) {
	frame := makeExampleFrame()
	data, err := frame.Encode(NewEncodingContext())
	if err != nil {
		t.Error("Encoding error:", err.Error())
	}
	withPriority := addPriority(data)
	withPaddingAndPriority := addPadding(7, withPriority)
	frameHeader := DecodeHeader(withPaddingAndPriority[0:9])
	result, err := DecodeHeadersFrame(frameHeader.Flags, frameHeader.StreamId, withPaddingAndPriority[9:], NewDecodingContext())
	if err != nil {
		t.Error("Decoding error:", err.Error())
	}
	frame.Priority = true // to compare with result
	if !reflect.DeepEqual(frame, result) {
		t.Error("Result does not equal expected frame.")
	}
}
