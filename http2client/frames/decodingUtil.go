package frames

import (
	"encoding/binary"
)

// uint32_ignoreFirstBit is like binary.BigEndian.Uint32, but with some differences:
//  * data may have an arbitrary length:
//  ** If the length is < four bytes, the leading bytes are assumed to be zero.
//  ** If the length is > four bytes, the leading bytes are ignored.
//  * The first Bit is ignored.
func uint32_ignoreFirstBit(data []byte) uint32 {
	buffer := make([]byte, 4) // implicitly initialized with zero bytes
	for i := 0; i < 4 && i < len(data); i++ {
		buffer[len(buffer)-(1+i)] = data[len(data)-(1+i)]
	}
	buffer[0] = buffer[0] & 0x7F // clear first bit
	return binary.BigEndian.Uint32(buffer)
}
