package internal

// This file contains helpers to hydrate objects from the buffers returned by Windows' API procs.

import (
	"unicode/utf16"
	"unsafe"

	"github.com/pkg/errors"
)

// ExtractWideStringFromBuffer extracts a null-terminated wide string from a buffer returned by the Windows API.
func ExtractWideStringFromBuffer(buffer []byte, bufferPointer, stringPointer uintptr) (string, uintptr, error) {
	if stringPointer == 0 && bufferPointer != 0 {
		// null pointer
		return "", 0, nil
	}

	// first let's compute the offset at which we should find the string in the buffer:
	// the pointer address we have might not be valid any more, since the GC might have moved the
	// buffer internally; that's why we use the address of the start of the buffer as it was when we made
	// the syscall
	bufferOffset := stringPointer - bufferPointer
	// sanity check: this should still be inside the buffer
	if stringPointer < bufferPointer || bufferOffset+1 >= uintptr(len(buffer)) {
		return "", 0, errors.New("wide string pointer pointing out of the buffer")
	}

	// read the buffer as wide chars until we hit a wide null byte
	// starting the wide chars buffer with a reasonable capacity allows avoiding too many resizes when adding to it
	wideChars := make([]uint16, 0, 50)
	for bufferOffset+1 < uintptr(len(buffer)) {
		// this is safe as per rule (1) of https://golang.org/pkg/unsafe/#Pointer:
		// any 2 bytes can be read as a uint16
		char := *(*uint16)(unsafe.Pointer(&buffer[bufferOffset]))
		if char == 0 {
			break
		}
		wideChars = append(wideChars, char)
		bufferOffset += 2
	}
	if bufferOffset+1 >= uintptr(len(buffer)) {
		// we never found a wide null byte
		return "", 0, errors.New("missing null character in wide string")
	}

	// now we can finally decode
	return string(utf16.Decode(wideChars)), 2 * (uintptr(len(wideChars)) + 1), nil
}

// ExtractStringFromBuffer extracts a regular string (PCHAR) with known length from a buffer returned by the Windows API.
func ExtractStringFromBuffer(buffer []byte, bufferPointer, stringPointer, stringSize uintptr) (string, error) {
	// first let's compute the offset at which we should find the string in the buffer:
	// the pointer address we have might not be valid any more, since the GC might have moved the
	// buffer internally; that's why we use the address of the start of the buffer as it was when we made
	// the syscall
	bufferOffset := stringPointer - bufferPointer
	// sanity check: this should still be inside the buffer
	if stringPointer < bufferPointer || bufferOffset+stringSize > uintptr(len(buffer)) {
		return "", errors.New("string pointer pointing out of the buffer")
	}

	strBytes := make([]byte, stringSize)
	copy(strBytes, buffer[bufferOffset:bufferOffset+stringSize])
	return string(strBytes), nil
}
