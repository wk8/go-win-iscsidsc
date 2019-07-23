package internal

import (
	"encoding/binary"
	"unicode/utf16"
	"unsafe"
)

// IterateOverAllSubsets will call f with all the 2^n - 1 (unordered) subsets of {0,1,2,...,n}.
func IterateOverAllSubsets(n uint, f func(subset []uint)) {
	max := uint(1<<n - 1)
	subset := make([]uint, n)

	generateSubset := func(i uint) []uint {
		index := 0
		for j := uint(0); j < n; j++ {
			if i&(1<<j) != 0 {
				subset[index] = j
				index++
			}
		}
		return subset[:index]
	}

	for i := uint(1); i <= max; i++ {
		f(generateSubset(i))
	}
}

// StringToUTF16ByteBuffer converts a string to UTF16 characters, then casts that to a null-terminated byte slice.
func StringToUTF16ByteBuffer(s string) []byte {
	utf16bytes := utf16.Encode([]rune(s + "\x00"))
	result := make([]byte, 2*len(utf16bytes))
	for i, utf16byte := range utf16bytes {
		endianness.PutUint16(result[2*i:], utf16byte)
	}
	return result
}

var endianness = determineEndiannness()

// shamelessly stolen from https://github.com/tensorflow/tensorflow/blob/v2.0.0-beta1/tensorflow/go/tensor.go#L488-L505
func determineEndiannness() binary.ByteOrder {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)

	switch buf {
	case [2]byte{0xCD, 0xAB}:
		return binary.LittleEndian
	case [2]byte{0xAB, 0xCD}:
		return binary.BigEndian
	default:
		panic("Could not determine native endianness.")
	}
}
