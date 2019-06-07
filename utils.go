package iscsidsc

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"

	"github.com/pkg/errors"
)

var (
	// TODO: we could (should?) check the version
	IscsidscDLL = windows.NewLazySystemDLL(getEnv("GO_WIN_ISCSI_DLL_NAME", "iscsidsc.dll"))
	// initialApiBufferSize is the size of the buffer used for the 1st call to APIs that need one.
	// It should big enough to ensure we won't need to make another call with a bigger buffer in most situations.
	// Having it as a var and not a constant allows overriding it during tests.
	InitialApiBufferSize uintptr = 200000
)

// getEnv looks up the `key` env variable, or returns `ifAbsent` if it's not defined
// or empty.
func getEnv(key, ifAbsent string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return ifAbsent
}

//go:uintptrescapes
//go:noinline

type WinApiCallError struct {
	procName string
	exitCode uintptr
}

func (err *WinApiCallError) Error() string {
	return fmt.Sprintf("exit code when calling %q: %s - please see Windows' documentation at https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-erref/18d8fbe8-a967-4f1c-ae50-99ca8e491d2d or https://docs.microsoft.com/en-us/windows-hardware/drivers/storage/iscsi-status-qualifiers and/or check your system logs in the event viewer",
		err.procName,
		err.HexCode())
}

func (err *WinApiCallError) ProcName() string {
	return err.procName
}

func (err *WinApiCallError) ExitCode() uintptr {
	return err.exitCode
}

// FIXME: unit tests
// ExitCodeToHexString formats a Windows exit status into the hexadecimal representation one can find
// in Windows' documentation.
// see https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-erref/18d8fbe8-a967-4f1c-ae50-99ca8e491d2d
// and https://docs.microsoft.com/en-us/windows-hardware/drivers/storage/iscsi-status-qualifiers
func (err *WinApiCallError) HexCode() string {
	return fmt.Sprintf("0x%08X", err.exitCode)
}

// CallWinApi makes a call to Windows' API.
func CallWinApi(proc *windows.LazyProc, args ...uintptr) (uintptr, error) {
	if err := proc.Find(); err != nil {
		return 0, errors.Wrapf(err, "Unable to locate %q function", proc.Name)
	}

	exitCode, _, _ := proc.Call(args...)
	if exitCode == 0 {
		return exitCode, nil
	}

	return exitCode, &WinApiCallError{
		procName: proc.Name,
		exitCode: exitCode,
	}
}
