package internal

import (
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wk8/go-win-iscsidsc"
)

func TestCheckAndConvertLoginOptions(t *testing.T) {
	runSingleTest := func(testName string, input *iscsidsc.LoginOptions, expectedOutput *LoginOptions) {
		// that's always the same
		expectedOutput.Version = LoginOptionsVersion

		t.Run(testName, func(t *testing.T) {
			output, userNamePtr, passwordPtr, err := CheckAndConvertLoginOptions(input)

			assert.Nil(t, err)
			assert.Equal(t, expectedOutput, output)
			assertIsBytePointerFromString(t, userNamePtr, input.Username)
			assertIsBytePointerFromString(t, passwordPtr, input.Password)
		})
	}

	runSingleTest("with an empty input", &iscsidsc.LoginOptions{}, &LoginOptions{})

	testCases := []struct {
		name               string
		inputFunc          func(*iscsidsc.LoginOptions)
		expectedOutputFunc func(*LoginOptions)
	}{
		{
			name: "auth type",
			inputFunc: func(optsIn *iscsidsc.LoginOptions) {
				chapAuthType := iscsidsc.ChapAuthType
				optsIn.AuthType = &chapAuthType
			},
			expectedOutputFunc: func(opts *LoginOptions) {
				opts.InformationSpecified |= InformationSpecifiedAuthType
				opts.AuthType = iscsidsc.ChapAuthType
			},
		},
		{
			name: "header digest",
			inputFunc: func(optsIn *iscsidsc.LoginOptions) {
				digestTypeCRC32C := iscsidsc.DigestTypeCRC32C
				optsIn.HeaderDigest = &digestTypeCRC32C
			},
			expectedOutputFunc: func(opts *LoginOptions) {
				opts.InformationSpecified |= InformationSpecifiedHeaderDigest
				opts.HeaderDigest = iscsidsc.DigestTypeCRC32C
			},
		},
		{
			name: "data digest",
			inputFunc: func(optsIn *iscsidsc.LoginOptions) {
				digestTypeCRC32C := iscsidsc.DigestTypeCRC32C
				optsIn.DataDigest = &digestTypeCRC32C
			},
			expectedOutputFunc: func(opts *LoginOptions) {
				opts.InformationSpecified |= InformationSpecifiedDataDigest
				opts.DataDigest = iscsidsc.DigestTypeCRC32C
			},
		},
		{
			name: "max connections",
			inputFunc: func(optsIn *iscsidsc.LoginOptions) {
				maximumConnections := uint32(12)
				optsIn.MaximumConnections = &maximumConnections
			},
			expectedOutputFunc: func(opts *LoginOptions) {
				opts.InformationSpecified |= InformationSpecifiedMaximumConnections
				opts.MaximumConnections = uint32(12)
			},
		},
		{
			name: "time to wait",
			inputFunc: func(optsIn *iscsidsc.LoginOptions) {
				timeToWait := uint32(28)
				optsIn.DefaultTime2Wait = &timeToWait
			},
			expectedOutputFunc: func(opts *LoginOptions) {
				opts.InformationSpecified |= InformationSpecifiedDefaultTime2Wait
				opts.DefaultTime2Wait = uint32(28)
			},
		},
		{
			name: "time to retain",
			inputFunc: func(optsIn *iscsidsc.LoginOptions) {
				timeToRetain := uint32(31)
				optsIn.DefaultTime2Retain = &timeToRetain
			},
			expectedOutputFunc: func(opts *LoginOptions) {
				opts.InformationSpecified |= InformationSpecifiedDefaultTime2Retain
				opts.DefaultTime2Retain = uint32(31)
			},
		},
		{
			name: "username",
			inputFunc: func(optsIn *iscsidsc.LoginOptions) {
				username := "username"
				optsIn.Username = &username
			},
			expectedOutputFunc: func(opts *LoginOptions) {
				opts.InformationSpecified |= InformationSpecifiedUsername
				opts.UsernameLength = uint32(8)
			},
		},
		{
			name: "password",
			inputFunc: func(optsIn *iscsidsc.LoginOptions) {
				password := "super_password"
				optsIn.Password = &password
			},
			expectedOutputFunc: func(opts *LoginOptions) {
				opts.InformationSpecified |= InformationSpecifiedPassword
				opts.PasswordLength = uint32(14)
			},
		},
	}

	IterateOverAllSubsets(uint(len(testCases)), func(indices []uint) {
		testName := "with "
		input := &iscsidsc.LoginOptions{}
		expectedOutput := &LoginOptions{}

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
		var result [MaxIscsiPortalNameLen]uint16
		copy(result[:], utf16)
		return result
	}

	testCases := []struct {
		name           string
		input          *iscsidsc.Portal
		expectedOutput *Portal
		expectedError  string
	}{
		{
			name: "happy path",
			input: &iscsidsc.Portal{
				SymbolicName: symbolicName,
				Address:      address,
				Socket:       &socket,
			},
			expectedOutput: &Portal{
				SymbolicName: toUTF16(symbolicName),
				Address:      toUTF16(address),
				Socket:       socket,
			},
		},
		{
			name:          "portal name too long",
			input:         &iscsidsc.Portal{SymbolicName: strings.Repeat("A", MaxIscsiPortalNameLen+1)},
			expectedError: "portal name too long, cannot be more than 256 characters",
		},
		{
			name:          "invalid portal name",
			input:         &iscsidsc.Portal{SymbolicName: "symbolic\x00name"},
			expectedError: "invalid portal name",
		},
		{
			name:          "portal address too long",
			input:         &iscsidsc.Portal{Address: strings.Repeat("A", MaxIscsiPortalAddressLen+1)},
			expectedError: "portal address too long, cannot be more than 256 characters",
		},
		{
			name:          "invalid portal address",
			input:         &iscsidsc.Portal{Address: "1.1.1.1\x00"},
			expectedError: "invalid portal address",
		},
		{
			name:           "empty input",
			input:          &iscsidsc.Portal{},
			expectedOutput: &Portal{Socket: DefaultPortalPortNumber},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			output, err := CheckAndConvertPortal(testCase.input)

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
