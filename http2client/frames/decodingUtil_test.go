package frames

import "testing"

func TestZeroBytes(t *testing.T) {
	result := uint32_ignoreFirstBit([]byte{})
	if result != 0 {
		t.Fatalf("Expected 0, but got %v\n", result)
	}
}

func TestOneByte(t *testing.T) {
	result := uint32_ignoreFirstBit([]byte{7})
	if result != 7 {
		t.Fatalf("Expected 7, but got %v\n", result)
	}
}

func TestTwoBytes(t *testing.T) {
	result := uint32_ignoreFirstBit([]byte{13, 7})
	expectedResult := uint32(13*1<<8 + 7)
	if result != expectedResult {
		t.Fatalf("Expected %v, but got %v\n", expectedResult, result)
	}
}

func TestThreeBytes(t *testing.T) {
	result := uint32_ignoreFirstBit([]byte{29, 13, 7})
	expectedResult := uint32(29*1<<16 + 13*1<<8 + 7)
	if result != expectedResult {
		t.Fatalf("Expected %v, but got %v\n", expectedResult, result)
	}
}

func TestFourBytesFirstBitClear(t *testing.T) {
	result := uint32_ignoreFirstBit([]byte{41, 29, 13, 7})
	expectedResult := uint32(41*1<<24 + 29*1<<16 + 13*1<<8 + 7)
	if result != expectedResult {
		t.Fatalf("Expected %v, but got %v\n", expectedResult, result)
	}
}

func TestFourBytesFirstBitSet(t *testing.T) {
	result := uint32_ignoreFirstBit([]byte{41 | 0x80, 29, 13, 7})
	expectedResult := uint32(41*1<<24 + 29*1<<16 + 13*1<<8 + 7)
	if result != expectedResult {
		t.Fatalf("Expected %v, but got %v\n", expectedResult, result)
	}
}

// first byte should be ignored
func TestFiveBytesFirstBitClear(t *testing.T) {
	result := uint32_ignoreFirstBit([]byte{53, 41, 29, 13, 7})
	expectedResult := uint32(41*1<<24 + 29*1<<16 + 13*1<<8 + 7)
	if result != expectedResult {
		t.Fatalf("Expected %v, but got %v\n", expectedResult, result)
	}
}
