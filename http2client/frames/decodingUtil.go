package frames

import "encoding/binary"

func uint32_ignoreFirstBit(data []byte) uint32 {
	buffer := make([]byte, 4)
	i := 0
	for i < 4 && i < len(data) {
		buffer[len(buffer)-(1+i)] = data[len(data)-(1+i)]
		i++
	}
	for i < 4 {
		buffer[len(buffer)-(1+i)] = 0
		i++
	}
	buffer[0] = buffer[0] & 0x7F // clear first bit
	return binary.BigEndian.Uint32(buffer)
}
