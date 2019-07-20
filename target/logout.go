package target

import (
	"unsafe"

	"github.com/wk8/go-win-iscsidsc"
	"github.com/wk8/go-win-iscsidsc/internal"
)

var procLogoutIScsiTarget = internal.GetDllProc("LogoutIScsiTarget")

// LogoutIScsiTarget closes the specified login session.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-logoutiscsitarget
func LogoutIScsiTarget(sessionId iscsidsc.SessionId) error {
	_, err := internal.CallWinApi(procLogoutIScsiTarget, uintptr(unsafe.Pointer(&sessionId)))
	return err
}
