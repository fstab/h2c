package frames

import (
	"bytes"
	"encoding/binary"
	"fmt"
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

type SettingsFrame struct {
	streamId uint32
	ack      bool
	settings map[Setting]uint32
}

func NewSettingsFrame(streamId uint32) *SettingsFrame {
	return &SettingsFrame{
		streamId: streamId,
		ack:      false,
		settings: make(map[Setting]uint32),
	}
}

func DecodeSettingsFrame(flags byte, streamId uint32, payload []byte, context *DecodingContext) (Frame, error) {
	if len(payload)%6 != 0 {
		return nil, fmt.Errorf("Invalid SETTINGS frame.")
	}
	result := NewSettingsFrame(streamId)
	result.ack = SETTINGS_FLAG_ACK.isSet(flags)
	for i := 0; i < len(payload); i += 6 {
		setting := Setting(binary.BigEndian.Uint16(payload[i : i+2]))
		value := binary.BigEndian.Uint32(payload[i+2 : i+6])
		if isUnknownSetting(setting) {
			return nil, fmt.Errorf("Unknown setting in SETTINGS frame.")
		}
		result.settings[setting] = value
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
	if f.ack {
		flags = append(flags, SETTINGS_FLAG_ACK)
	}
	return flags
}

func (f *SettingsFrame) Encode(context *EncodingContext) ([]byte, error) {
	var result bytes.Buffer
	length := uint32(len(f.settings) * 6)
	result.Write(encodeHeader(f.Type(), f.streamId, length, f.flags()))
	for id, value := range f.settings {
		idBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(idBytes, uint16(id))
		result.Write(idBytes)
		valueBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(valueBytes, value)
		result.Write(valueBytes)
	}
	return result.Bytes(), nil
}

func (f *SettingsFrame) String() string {
	return fmt.Sprintf("SETTINGS(%v)", f.streamId)
}
