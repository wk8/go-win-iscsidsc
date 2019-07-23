package iscsidsc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWinAPICallErrorHexStrings(t *testing.T) {
	testCases := map[uintptr]string{
		0:          "0x00000000",
		4026466307: "0xEFFF0003",
	}

	for exitCode, hexString := range testCases {
		t.Run(fmt.Sprintf("%d should convert to %s", exitCode, hexString), func(t *testing.T) {
			err := &WinAPICallError{exitCode: exitCode}
			assert.Equal(t, hexString, err.HexCode())
		})
	}
}
