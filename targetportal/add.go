package targetportal

import (
	"unsafe"

	"github.com/pkg/errors"
	iscsidsc "github.com/wk8/go-win-iscsidsc"
	"github.com/wk8/go-win-iscsidsc/internal"
)

var procAddIScsiSendTargetPortalW = internal.GetDllProc("AddIScsiSendTargetPortalW")

// AddIScsiSendTargetPortal adds a static target portal to the list of target portals to which the iSCSI initiator service transmits SendTargets requests.
// Only the `portal` is a required argument - all others can be left `nil`.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-addiscsisendtargetportalw
func AddIScsiSendTargetPortal(initiatorInstance *string, initiatorPortNumber *uint32, loginOptions *iscsidsc.LoginOptions, securityFlags *iscsidsc.SecurityFlags, portal *iscsidsc.Portal) error {
	initiatorInstancePtr, initiatorPortNumberValue, err := internal.ConvertInitiatorArgs(initiatorInstance, initiatorPortNumber)
	if err != nil {
		return err
	}

	internalLoginOptions, userNamePtr, passwordPtr, err := internal.CheckAndConvertLoginOptions(loginOptions)
	if err != nil {
		return errors.Wrap(err, "invalid loginOptions argument")
	}

	var securityFlagsValue iscsidsc.SecurityFlags
	if securityFlags != nil {
		securityFlagsValue = *securityFlags
	}

	if portal == nil {
		return errors.Errorf("portal is required")
	}
	internalPortal, err := internal.CheckAndConvertPortal(portal)
	if err != nil {
		return errors.Wrap(err, "invalid portal argument")
	}

	_, err = callProcAddIScsiSendTargetPortalW(
		initiatorInstancePtr,
		initiatorPortNumberValue,
		internalLoginOptions,
		securityFlagsValue,
		internalPortal,
		uintptr(unsafe.Pointer(userNamePtr)),
		uintptr(unsafe.Pointer(passwordPtr)),
	)

	return err
}

//go:uintptrescapes
//go:noinline

// callProcAddIScsiSendTargetPortalW is only a wrapper around `internal.CallWinAPI`.
// Its main purpose is that the unsafe pointers to the username and password strings are
// guaranteed to stay in the same place in memory until this function returns.
// See `internal.CheckAndConvertLoginOptions`'s doc comment as well as https://golang.org/pkg/unsafe/#Pointer
// for more context.
func callProcAddIScsiSendTargetPortalW(initiatorInstancePtr *uint16, initiatorPortNumberValue uint32,
	internalLoginOptions *internal.LoginOptions, securityFlagsValue iscsidsc.SecurityFlags, internalPortal *internal.Portal,
	userNameUintptr, passwordUintptr uintptr) (uintptr, error) {

	internalLoginOptions.Username = userNameUintptr
	internalLoginOptions.Password = passwordUintptr

	return internal.CallWinAPI(procAddIScsiSendTargetPortalW,
		uintptr(unsafe.Pointer(initiatorInstancePtr)),
		uintptr(initiatorPortNumberValue),
		uintptr(unsafe.Pointer(internalLoginOptions)),
		uintptr(securityFlagsValue),
		uintptr(unsafe.Pointer(internalPortal)),
	)
}
