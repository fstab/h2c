package frames

import (
	"bytes"
	"encoding/binary"
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
	String() string
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

func (flag Flag) isSet(flagsByte byte) bool {
	return flagsByte&byte(flag) != 0
}

func (flag Flag) set(flagsByte *byte) {
	*flagsByte = *flagsByte | byte(flag)
}
