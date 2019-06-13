package targetportal

import (
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wk8/go-win-iscsidsc/internal"
)

func TestCheckAndConvertLoginOptions(t *testing.T) {
	runSingleTest := func(testName string, input *LoginOptions, expectedOutput *loginOptions) {
		// that's always the same
		expectedOutput.version = loginOptionsVersion

		t.Run(testName, func(t *testing.T) {
			output, userNamePtr, passwordPtr, err := checkAndConvertLoginOptions(input)

			assert.Nil(t, err)
			assert.Equal(t, expectedOutput, output)
			assertIsBytePointerFromString(t, userNamePtr, input.Username)
			assertIsBytePointerFromString(t, passwordPtr, input.Password)
		})
	}

	runSingleTest("with an empty input", &LoginOptions{}, &loginOptions{})

	testCases := []struct {
		name               string
		inputFunc          func(*LoginOptions)
		expectedOutputFunc func(*loginOptions)
	}{
		{
			name: "auth type",
			inputFunc: func(optsIn *LoginOptions) {
				chapAuthType := ChapAuthType
				optsIn.AuthType = &chapAuthType
			},
			expectedOutputFunc: func(opts *loginOptions) {
				opts.informationSpecified |= informationSpecifiedAuthType
				opts.authType = ChapAuthType
			},
		},
		{
			name: "header digest",
			inputFunc: func(optsIn *LoginOptions) {
				digestTypeCRC32C := DigestTypeCRC32C
				optsIn.HeaderDigest = &digestTypeCRC32C
			},
			expectedOutputFunc: func(opts *loginOptions) {
				opts.informationSpecified |= informationSpecifiedHeaderDigest
				opts.headerDigest = DigestTypeCRC32C
			},
		},
		{
			name: "data digest",
			inputFunc: func(optsIn *LoginOptions) {
				digestTypeCRC32C := DigestTypeCRC32C
				optsIn.DataDigest = &digestTypeCRC32C
			},
			expectedOutputFunc: func(opts *loginOptions) {
				opts.informationSpecified |= informationSpecifiedDataDigest
				opts.dataDigest = DigestTypeCRC32C
			},
		},
		{
			name: "max connections",
			inputFunc: func(optsIn *LoginOptions) {
				maximumConnections := uint32(12)
				optsIn.MaximumConnections = &maximumConnections
			},
			expectedOutputFunc: func(opts *loginOptions) {
				opts.informationSpecified |= informationSpecifiedMaximumConnections
				opts.maximumConnections = uint32(12)
			},
		},
		{
			name: "time to wait",
			inputFunc: func(optsIn *LoginOptions) {
				timeToWait := uint32(28)
				optsIn.DefaultTime2Wait = &timeToWait
			},
			expectedOutputFunc: func(opts *loginOptions) {
				opts.informationSpecified |= informationSpecifiedDefaultTime2Wait
				opts.defaultTime2Wait = uint32(28)
			},
		},
		{
			name: "time to retain",
			inputFunc: func(optsIn *LoginOptions) {
				timeToRetain := uint32(31)
				optsIn.DefaultTime2Retain = &timeToRetain
			},
			expectedOutputFunc: func(opts *loginOptions) {
				opts.informationSpecified |= informationSpecifiedDefaultTime2Retain
				opts.defaultTime2Retain = uint32(31)
			},
		},
		{
			name: "username",
			inputFunc: func(optsIn *LoginOptions) {
				username := "username"
				optsIn.Username = &username
			},
			expectedOutputFunc: func(opts *loginOptions) {
				opts.informationSpecified |= informationSpecifiedUsername
				opts.usernameLength = uint32(8)
			},
		},
		{
			name: "password",
			inputFunc: func(optsIn *LoginOptions) {
				password := "super_password"
				optsIn.Password = &password
			},
			expectedOutputFunc: func(opts *loginOptions) {
				opts.informationSpecified |= informationSpecifiedPassword
				opts.passwordLength = uint32(14)
			},
		},
	}

	internal.IterateOverAllSubsets(uint(len(testCases)), func(indices []uint) {
		testName := "with "
		input := &LoginOptions{}
		expectedOutput := &loginOptions{}

		for j, index := range indices {
			if j != 0 {
				testName += " ,"
			}
			testName += testCases[index].name
			testCases[index].inputFunc(input)
			testCases[index].expectedOutputFunc(expectedOutput)
		}

		runSingleTest(testName, input, expectedOutput)
	})
}

func TestCheckAndConvertPortal(t *testing.T) {
	symbolicName := "symbolic_name"
	address := "1.1.1.1"
	socket := uint16(2828)

	toUTF16 := func(s string) [256]uint16 {
		utf16, err := windows.UTF16FromString(s)
		require.Nil(t, err)
		var result [maxIscsiPortalNameLen]uint16
		copy(result[:], utf16)
		return result
	}

	testCases := []struct {
		name           string
		input          *Portal
		expectedOutput *portal
		expectedError  string
	}{
		{
			name: "happy path",
			input: &Portal{
				SymbolicName: symbolicName,
				Address:      address,
				Socket:       &socket,
			},
			expectedOutput: &portal{
				symbolicName: toUTF16(symbolicName),
				address:      toUTF16(address),
				socket:       socket,
			},
		},
		{
			name:          "portal name too long",
			input:         &Portal{SymbolicName: strings.Repeat("A", maxIscsiPortalNameLen+1)},
			expectedError: "portal name too long, cannot be more than 256 characters",
		},
		{
			name:          "invalid portal name",
			input:         &Portal{SymbolicName: "symbolic\x00name"},
			expectedError: "invalid portal name",
		},
		{
			name:          "portal address too long",
			input:         &Portal{Address: strings.Repeat("A", maxIscsiPortalAddressLen+1)},
			expectedError: "portal address too long, cannot be more than 256 characters",
		},
		{
			name:          "invalid portal address",
			input:         &Portal{Address: "1.1.1.1\x00"},
			expectedError: "invalid portal address",
		},
		{
			name:           "empty input",
			input:          &Portal{},
			expectedOutput: &portal{socket: defaultPortalPortNumber},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			output, err := checkAndConvertPortal(testCase.input)

			if testCase.expectedError == "" {
				assert.Nil(t, err)
				assert.Equal(t, testCase.expectedOutput, output)
			} else {
				assert.Contains(t, err.Error(), testCase.expectedError)

				assert.Nil(t, output)
				assert.Nil(t, testCase.expectedOutput)
			}
		})
	}
}

// assertIsBytePointerFromString asserts that ptr was obtained by calling windows.BytePtrFromString(*str).
// also checks that either both pointers are nil, or both are not-nil.
func assertIsBytePointerFromString(t *testing.T, ptr *byte, str *string) {
	if ptr == nil {
		assert.Nil(t, str)
		return
	}
	if !assert.NotNil(t, str) {
		return
	}

	byteSlice := make([]byte, len(*str)+1)

	for i := 0; i < len(byteSlice); i++ {
		byteSlice[i] = *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(i)))
	}

	assert.Equal(t, *str, string(byteSlice[:len(byteSlice)-1]))
	assert.Equal(t, byte(0), byteSlice[len(byteSlice)-1])
}
