package frames

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Type byte
type Flag byte

const (
	DATA_TYPE     Type = 0x00
	HEADERS_TYPE  Type = 0x01
	SETTINGS_TYPE Type = 0x04
)

type Frame interface {
	Encode(*EncodingContext) ([]byte, error)
	Type() Type
}

func FindDecoder(frameType Type) func(flags byte, streamId uint32, payload []byte, context *DecodingContext) (Frame, error) {
	switch frameType {
	case DATA_TYPE:
		return DecodeDataFrame
	case HEADERS_TYPE:
		return DecodeHeadersFrame
	case SETTINGS_TYPE:
		return DecodeSettingsFrame
	default:
		return nil
	}
}

func encodeHeader(frameType Type, streamId uint32, length uint32, flags []Flag) []byte {
	var result bytes.Buffer
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, length)
	result.Write(bytes[1:])
	bytes[0] = byte(frameType)
	result.Write(bytes[0:1])
	bytes[0] = 0
	for _, flag := range flags {
		flag.set(&bytes[0])
	}
	result.Write(bytes[0:1])
	binary.BigEndian.PutUint32(bytes, streamId)
	result.Write(bytes)
	return result.Bytes()
}

type FrameHeader struct {
	Length     uint32
	HeaderType Type
	Flags      byte
	StreamId   uint32
}

func DecodeHeader(data []byte) *FrameHeader {
	length := append(make([]byte, 1), data[0:3]...)   // 4 Byte Big Endian
	streamId := append(make([]byte, 0), data[5:9]...) // 4 Byte Big Endian
	streamId[0] = streamId[0] & 0x7F                  // clear reserved bit
	return &FrameHeader{
		Length:     binary.BigEndian.Uint32(length),
		HeaderType: Type(data[3]),
		Flags:      data[4],
		StreamId:   binary.BigEndian.Uint32(streamId),
	}
}

func (flag Flag) isSet(flagsByte byte) bool {
	return flagsByte&byte(flag) != 0
}

func (flag Flag) set(flagsByte *byte) {
	*flagsByte = *flagsByte | byte(flag)
}

func stripPadding(payload []byte) ([]byte, error) {
	padLength := int(payload[0])
	if len(payload) <= padLength {
		// TODO: trigger connection error.
		return nil, fmt.Errorf("Invalid HEADERS frame: padding >= payload.")
	}
	return payload[1 : len(payload)-padLength], nil
}
