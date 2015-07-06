package frames

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

type Setting uint16

const (
	SETTINGS_HEADER_TABLE_SIZE      Setting = 0x01
	SETTINGS_ENABLE_PUSH            Setting = 0x02
	SETTINGS_MAX_CONCURRENT_STREAMS Setting = 0x03
	SETTINGS_INITIAL_WINDOW_SIZE    Setting = 0x04
	SETTINGS_MAX_FRAME_SIZE         Setting = 0x05
	SETTINGS_MAX_HEADER_LIST_SIZE   Setting = 0x06
)

const (
	SETTINGS_FLAG_ACK Flag = 0x01
)

func (s Setting) String() string {
	switch s {
	case SETTINGS_HEADER_TABLE_SIZE:
		return "SETTINGS_HEADER_TABLE_SIZE"
	case SETTINGS_ENABLE_PUSH:
		return "SETTINGS_ENABLE_PUSH"
	case SETTINGS_MAX_CONCURRENT_STREAMS:
		return "SETTINGS_MAX_CONCURRENT_STREAMS"
	case SETTINGS_INITIAL_WINDOW_SIZE:
		return "SETTINGS_INITIAL_WINDOW_SIZE"
	case SETTINGS_MAX_FRAME_SIZE:
		return "SETTINGS_MAX_FRAME_SIZE"
	case SETTINGS_MAX_HEADER_LIST_SIZE:
		return "SETTINGS_MAX_HEADER_LIST_SIZE"
	default:
		fmt.Fprintf(os.Stderr, "ERROR: Unknown setting %v", s)
		os.Exit(-1)
		return ""
	}
}

func (s Setting) IsSet(f *SettingsFrame) bool {
	_, ok := f.Settings[s]
	return ok
}

func (s Setting) Get(f *SettingsFrame) uint32 {
	val, _ := f.Settings[s]
	return val
}

type SettingsFrame struct {
	StreamId uint32
	Ack      bool
	Settings map[Setting]uint32
}

func NewSettingsFrame(streamId uint32) *SettingsFrame {
	return &SettingsFrame{
		StreamId: streamId,
		Ack:      false,
		Settings: make(map[Setting]uint32),
	}
}

func DecodeSettingsFrame(flags byte, streamId uint32, payload []byte, context *DecodingContext) (Frame, error) {
	if len(payload)%6 != 0 {
		return nil, fmt.Errorf("Invalid SETTINGS frame.")
	}
	result := NewSettingsFrame(streamId)
	result.Ack = SETTINGS_FLAG_ACK.isSet(flags)
	for i := 0; i < len(payload); i += 6 {
		setting := Setting(binary.BigEndian.Uint16(payload[i : i+2]))
		value := binary.BigEndian.Uint32(payload[i+2 : i+6])
		if isUnknownSetting(setting) {
			return nil, fmt.Errorf("Unknown setting in SETTINGS frame.")
		}
		result.Settings[setting] = value
	}
	return result, nil
}

func isUnknownSetting(setting Setting) bool {
	return setting != SETTINGS_HEADER_TABLE_SIZE &&
		setting != SETTINGS_ENABLE_PUSH &&
		setting != SETTINGS_MAX_CONCURRENT_STREAMS &&
		setting != SETTINGS_INITIAL_WINDOW_SIZE &&
		setting != SETTINGS_MAX_FRAME_SIZE &&
		setting != SETTINGS_MAX_HEADER_LIST_SIZE
}

func (f *SettingsFrame) Type() Type {
	return SETTINGS_TYPE
}

func (f *SettingsFrame) flags() []Flag {
	flags := make([]Flag, 0)
	if f.Ack {
		flags = append(flags, SETTINGS_FLAG_ACK)
	}
	return flags
}

func (f *SettingsFrame) Encode(context *EncodingContext) ([]byte, error) {
	var result bytes.Buffer
	length := uint32(len(f.Settings) * 6)
	result.Write(encodeHeader(f.Type(), f.StreamId, length, f.flags()))
	for id, value := range f.Settings {
		idBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(idBytes, uint16(id))
		result.Write(idBytes)
		valueBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(valueBytes, value)
		result.Write(valueBytes)
	}
	return result.Bytes(), nil
}
