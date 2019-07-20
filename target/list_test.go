package target

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wk8/go-win-iscsidsc/internal"
)

func TestParseIscsiTargets(t *testing.T) {
	testCases := [][]string{
		{},
		{"foo"},
		{"foo", "bar"},
		{"foo", "bar", "baz"},
		{"a"},
		{"a", "foo", "b"},
	}
	paddingLengths := []int{0, 100, 1000}

	for _, paddingLength := range paddingLengths {
		padding := make([]byte, paddingLength)
		for i := 0; i < paddingLength; i++ {
			padding[i] = 1
		}

		for _, targets := range testCases {
			t.Run(fmt.Sprintf("for %v with padding length %d", targets, paddingLength), func(t *testing.T) {
				output := append(buildIscsiTargetsOutput(targets...), padding...)
				parsed, err := parseIscsiTargets(output)

				assert.Nil(t, err)
				assert.Equal(t, targets, parsed)
			})
		}
	}

	t.Run("not double-null terminated", func(t *testing.T) {
		output := buildIscsiTargetsOutput("foo", "bar")

		_, err := parseIscsiTargets(output[:len(output)-1])

		assert.Equal(t, invalidIscsiTargetsOutput, err)
	})

	t.Run("with too short a buffer", func(t *testing.T) {
		for _, length := range []int{0, 1, 2, 3} {
			_, err := parseIscsiTargets(make([]byte, length))

			assert.Equal(t, invalidIscsiTargetsOutput, err, "buffer of length %d", length)
		}
	})

	t.Run("with a buffer with 2 wide characters, but the second one is not a null byte", func(t *testing.T) {
		_, err := parseIscsiTargets([]byte{0, 0, 0, 1})

		assert.Equal(t, invalidIscsiTargetsOutput, err)
	})
}

// buildIscsiTargetsOutput builds a well-formed output for ReportIScsiTargetsW, i.e. a list
// of UTF16-encoded, null-terminated strings, with the last string double null-terminated.
func buildIscsiTargetsOutput(targets ...string) []byte {
	result := make([]byte, 0)

	for _, target := range targets {
		result = append(result, internal.StringToUTF16ByteBuffer(target)...)
	}
	// add a wide null byte
	result = append(result, 0, 0)
	if len(targets) == 0 {
		// need another wide null byte then
		result = append(result, 0, 0)
	}

	return result
}
