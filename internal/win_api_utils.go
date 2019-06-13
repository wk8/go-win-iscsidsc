package internal

import (
	"golang.org/x/sys/windows"
	"os"

	"github.com/pkg/errors"
	"github.com/wk8/go-win-iscsidsc"
)

var (
	// TODO: we could (should?) check the version
	IscsidscDLL = windows.NewLazySystemDLL(getEnv("GO_WIN_ISCSI_DLL_NAME", "iscsidsc.dll"))

	// InitialApiBufferSize is the size of the buffer used for the 1st call to APIs that need one.
	// It should big enough to ensure we won't need to make another call with a bigger buffer in most situations.
	// Having it as a var and not a constant allows overriding it during tests.
	// Note that on some versions of Windows, if this is set to more than 100000, it will result in a ERROR_NOACCESS
	// error (...?)
	InitialApiBufferSize uintptr = 100000
)

//go:uintptrescapes
//go:noinline

// CallWinApi makes a call to Windows' API.
func CallWinApi(proc *windows.LazyProc, args ...uintptr) (uintptr, error) {
	if err := proc.Find(); err != nil {
		return 0, errors.Wrapf(err, "Unable to locate %q function", proc.Name)
	}

	exitCode, _, _ := proc.Call(args...)

	if exitCode == 0 {
		return exitCode, nil
	}
	return exitCode, iscsidsc.NewWinApiCallError(proc.Name, exitCode)
}

// BoolToByte converts a boolean to a C++ byte.
func BoolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// getEnv looks up the `key` env variable, or returns `ifAbsent` if it's not defined
// or empty.
func getEnv(key, ifAbsent string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return ifAbsent
}
