package target

import (
	"fmt"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		padding := make([]uint16, paddingLength)
		for i := 0; i < paddingLength; i++ {
			padding[i] = 1
		}

		for _, targets := range testCases {
			t.Run(fmt.Sprintf("for %v with padding length %d", targets, paddingLength), func(t *testing.T) {
				output := append(buildIscsiTargetsOutput(t, targets...), padding...)
				parsed, err := parseIscsiTargets(output)

				assert.Nil(t, err)
				assert.Equal(t, targets, parsed)
			})
		}
	}

	t.Run("not double-null terminated", func(t *testing.T) {
		output := buildIscsiTargetsOutput(t, "foo", "bar")

		_, err := parseIscsiTargets(output[:len(output)-1])

		assert.Equal(t, invalidIscsiTargetsOutput, err)
	})

	t.Run("with too short a buffer", func(t *testing.T) {
		_, err := parseIscsiTargets([]uint16{0})

		assert.Equal(t, invalidIscsiTargetsOutput, err)
	})

	t.Run("with a buffer with 2 characters, but the second one is not a null byte", func(t *testing.T) {
		_, err := parseIscsiTargets([]uint16{0, 1})

		assert.Equal(t, invalidIscsiTargetsOutput, err)
	})
}

// buildIscsiTargetsOutput builds a well-formed output for ReportIScsiTargetsW, i.e. a list
// of UTF16-encoded, null-terminated strings, with the last string double null-terminated.
func buildIscsiTargetsOutput(t *testing.T, targets ...string) []uint16 {
	result := make([]uint16, 0)

	for _, target := range targets {
		encoded, err := windows.UTF16FromString(target)
		require.Nil(t, err)
		result = append(result, encoded...)
	}
	result = append(result, 0)
	if len(targets) == 0 {
		// need another null byte then
		result = append(result, 0)
	}

	return result
}
