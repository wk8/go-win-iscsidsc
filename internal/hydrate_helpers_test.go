package internal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractWideStringFromBuffer(t *testing.T) {
	runeLen := func(s string) uintptr {
		return uintptr(2 * len([]rune(s)))
	}

	for _, input := range []string{"", "foo", "a", "&=)", "Ŋ", "fŏŎ"} {
		t.Run("simple buffer with "+input, func(t *testing.T) {
			output, read, err := ExtractWideStringFromBuffer(StringToUTF16ByteBuffer(input), 0, 0)

			assert.Equal(t, input, output)
			assert.Equal(t, runeLen(input)+2, read)
			assert.Nil(t, err)
		})
	}

	t.Run("when passed a nil string pointer from a non-nil buffer pointer, it returns an empty string", func(t *testing.T) {
		output, read, err := ExtractWideStringFromBuffer(make([]byte, 100), 100, 0)

		assert.Equal(t, "", output)
		assert.Equal(t, uintptr(0), read)
		assert.Nil(t, err)
	})

	t.Run("when passed a string pointer pointing before the buffer, it errors out", func(t *testing.T) {
		output, read, err := ExtractWideStringFromBuffer(make([]byte, 100), 100, 90)

		assert.Equal(t, "", output)
		assert.Equal(t, uintptr(0), read)
		if assert.NotNil(t, err) {
			assert.Equal(t, "wide string pointer pointing out of the buffer", err.Error())
		}
	})

	t.Run("when passed a string pointer pointing after the buffer, it errors out", func(t *testing.T) {
		// 109 is still pointing after, since any UTF16 char is 2 byte
		for _, stringPointer := range []uintptr{109, 110, 111, 200} {
			output, read, err := ExtractWideStringFromBuffer(make([]byte, 10), 100, stringPointer)

			assert.Equal(t, "", output)
			assert.Equal(t, uintptr(0), read)
			if assert.NotNil(t, err) {
				assert.Equal(t, "wide string pointer pointing out of the buffer", err.Error())
			}
		}
	})

	bufferPointer := uintptr(100)
	stringPointer := uintptr(130)
	buffer := make([]byte, 100)
	for i := range buffer {
		buffer[i] = 12
	}
	input := "coucou"
	copy(buffer[stringPointer-bufferPointer:], StringToUTF16ByteBuffer(input))

	t.Run("happy path extracting from the middle of a buffer", func(t *testing.T) {
		output, read, err := ExtractWideStringFromBuffer(buffer, bufferPointer, stringPointer)

		assert.Equal(t, input, output)
		assert.Equal(t, runeLen(input)+2, read)
		assert.Nil(t, err)
	})

	// let's find the double null byte
	nullByteIndex := stringPointer - bufferPointer + runeLen(input)
	assert.Equal(t, byte(0), buffer[nullByteIndex])
	assert.Equal(t, byte(0), buffer[nullByteIndex+1])
	// and test that changing it to a non-null character errors out
	for _, modifiedIndex := range []uintptr{nullByteIndex, nullByteIndex + 1} {
		t.Run(fmt.Sprintf("when there is no terminating null byte, it errors out - changing byte %d", modifiedIndex), func(t *testing.T) {
			buffer[modifiedIndex] = 28
			defer func() { buffer[modifiedIndex] = 0 }()

			output, read, err := ExtractWideStringFromBuffer(buffer, bufferPointer, stringPointer)

			assert.Equal(t, "", output)
			assert.Equal(t, uintptr(0), read)
			if assert.NotNil(t, err) {
				assert.Equal(t, "missing null character in wide string", err.Error())
			}
		})
	}
}

func TestExtractStringFromBuffer(t *testing.T) {
	stringToByteSlice := func(s string, bufferLen, strPointer uintptr) []byte {
		buffer := make([]byte, bufferLen)
		copy(buffer[strPointer:], s)
		return buffer
	}

	for _, input := range []string{"", "foo", "a", "&=)"} {
		t.Run("simple buffer with "+input, func(t *testing.T) {
			output, err := ExtractStringFromBuffer(stringToByteSlice(input, uintptr(len(input)), 0), 0, 0, uintptr(len(input)))

			assert.Equal(t, input, output)
			assert.Nil(t, err)
		})
	}

	t.Run("when passed a string pointer pointing before the buffer, it errors out", func(t *testing.T) {
		output, err := ExtractStringFromBuffer(make([]byte, 100), 100, 99, 1)

		assert.Equal(t, "", output)
		if assert.NotNil(t, err) {
			assert.Equal(t, "string pointer pointing out of the buffer", err.Error())
		}
	})

	t.Run("when passed a string pointer pointing after the buffer, it errors out", func(t *testing.T) {
		// 191 is still pointing after, since there wouldn't be room for our 10 characters
		for _, stringPointer := range []uintptr{191, 195, 200, 300} {
			output, err := ExtractStringFromBuffer(make([]byte, 100), 100, stringPointer, 10)

			assert.Equal(t, "", output)
			if assert.NotNil(t, err) {
				assert.Equal(t, "string pointer pointing out of the buffer", err.Error())
			}
		}
	})

	t.Run("happy path extracting from the middle of a buffer", func(t *testing.T) {
		output, err := ExtractStringFromBuffer(stringToByteSlice("coucou", 100, 30), 100, 130, 6)

		assert.Equal(t, "coucou", output)
		assert.Nil(t, err)
	})

	t.Run("it doesn't care about null bytes", func(t *testing.T) {
		output, err := ExtractStringFromBuffer(stringToByteSlice("coucou", 100, 30), 100, 128, 12)

		assert.Equal(t, "\x00\x00coucou\x00\x00\x00\x00", output)
		assert.Nil(t, err)
	})
}
