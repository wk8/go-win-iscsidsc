package iscsidsc

import (
	"fmt"
)

type WinApiCallError struct {
	procName string
	exitCode uintptr
}

func NewWinApiCallError(procName string, exitCode uintptr) *WinApiCallError {
	return &WinApiCallError{
		procName: procName,
		exitCode: exitCode,
	}
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

// ExitCodeToHexString formats a Windows exit status into the hexadecimal representation one can find
// in Windows' documentation.
// see https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-erref/18d8fbe8-a967-4f1c-ae50-99ca8e491d2d
// and https://docs.microsoft.com/en-us/windows-hardware/drivers/storage/iscsi-status-qualifiers
func (err *WinApiCallError) HexCode() string {
	return fmt.Sprintf("0x%08X", err.exitCode)
}
