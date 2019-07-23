package iscsidsc

import (
	"fmt"
)

// WinAPICallError is used when a Windows API call fails, and it allows
// the caller to get some meta-data about the failure.
type WinAPICallError struct {
	procName string
	exitCode uintptr
}

// NewWinAPICallError builds a new WinAPICallError.
func NewWinAPICallError(procName string, exitCode uintptr) *WinAPICallError {
	return &WinAPICallError{
		procName: procName,
		exitCode: exitCode,
	}
}

func (err *WinAPICallError) Error() string {
	return fmt.Sprintf("exit code when calling %q: %s - please see Windows' documentation at https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-erref/18d8fbe8-a967-4f1c-ae50-99ca8e491d2d or https://docs.microsoft.com/en-us/windows-hardware/drivers/storage/iscsi-status-qualifiers and/or check your system logs in Windows' event viewer",
		err.procName,
		err.HexCode())
}

// ProcName returns the name of the proc that failed.
func (err *WinAPICallError) ProcName() string {
	return err.procName
}

// ExitCode returns the numerical exit code.
func (err *WinAPICallError) ExitCode() uintptr {
	return err.exitCode
}

// HexCode formats a Windows exit status into the hexadecimal representation one
// can find in Windows' documentation.
// see https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-erref/18d8fbe8-a967-4f1c-ae50-99ca8e491d2d
// and https://docs.microsoft.com/en-us/windows-hardware/drivers/storage/iscsi-status-qualifiers
func (err *WinAPICallError) HexCode() string {
	return fmt.Sprintf("0x%08X", err.exitCode)
}
