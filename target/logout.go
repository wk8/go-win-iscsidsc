package target

import (
	"unsafe"

	iscsidsc "github.com/wk8/go-win-iscsidsc"
	"github.com/wk8/go-win-iscsidsc/internal"
)

var procLogoutIScsiTarget = internal.GetDllProc("LogoutIScsiTarget")

// LogoutIScsiTarget closes the specified login session.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-logoutiscsitarget
func LogoutIScsiTarget(sessionID iscsidsc.SessionID) error {
	_, err := internal.CallWinAPI(procLogoutIScsiTarget, uintptr(unsafe.Pointer(&sessionID)))
	return err
}
