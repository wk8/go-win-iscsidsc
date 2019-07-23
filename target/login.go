package target

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/pkg/errors"
	iscsidsc "github.com/wk8/go-win-iscsidsc"
	"github.com/wk8/go-win-iscsidsc/internal"
)

var procLoginIScsiTargetW = internal.GetDllProc("LoginIScsiTargetW")

// LoginIscsiTarget establishes a full featured login session with the indicated target.
// All pointer arguments are optional.
// TODO: we don't support passing custom mappings yet.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-loginiscsitargetw
func LoginIscsiTarget(targetName string, isInformationalSession bool, initiatorInstance *string, initiatorPortNumber *uint32, targetPortal *iscsidsc.Portal,
	securityFlags *iscsidsc.SecurityFlags, loginOptions *iscsidsc.LoginOptions, key *string, isPersistent bool) (*iscsidsc.SessionID, *iscsidsc.ConnectionID, error) {
	targetNamePtr, err := windows.UTF16PtrFromString(targetName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "invalid target name: %q", targetName)
	}

	initiatorInstancePtr, initiatorPortNumberValue, err := internal.ConvertInitiatorArgs(initiatorInstance, initiatorPortNumber)
	if err != nil {
		return nil, nil, err
	}

	internalPortal, err := internal.CheckAndConvertPortal(targetPortal)
	if err != nil {
		return nil, nil, errors.Wrap(err, "invalid portal argument")
	}

	var securityFlagsValue iscsidsc.SecurityFlags
	if securityFlags != nil {
		securityFlagsValue = *securityFlags
	}

	internalLoginOptions, userNamePtr, passwordPtr, err := internal.CheckAndConvertLoginOptions(loginOptions)
	if err != nil {
		return nil, nil, errors.Wrap(err, "invalid loginOptions argument")
	}

	keyPtr, keySize, err := internal.CheckAndConvertKey(key)
	if err != nil {
		return nil, nil, err
	}

	return callProcLoginIScsiTargetW(targetNamePtr, isInformationalSession, initiatorInstancePtr, initiatorPortNumberValue,
		internalPortal, securityFlagsValue, internalLoginOptions, uintptr(unsafe.Pointer(userNamePtr)), uintptr(unsafe.Pointer(passwordPtr)),
		keyPtr, keySize, isPersistent)
}

//go:uintptrescapes
//go:noinline

func callProcLoginIScsiTargetW(targetNamePtr *uint16, isInformationalSession bool, initiatorInstancePtr *uint16, initiatorPortNumberValue uint32,
	internalPortal *internal.Portal, securityFlagsValue iscsidsc.SecurityFlags, internalLoginOptions *internal.LoginOptions,
	userNameUintptr, passwordUintptr uintptr, keyPtr *byte, keySize uint32, isPersistent bool) (*iscsidsc.SessionID, *iscsidsc.ConnectionID, error) {

	internalLoginOptions.Username = userNameUintptr
	internalLoginOptions.Password = passwordUintptr

	sessionID := &iscsidsc.SessionID{}
	connectionID := &iscsidsc.ConnectionID{}

	if _, err := internal.CallWinAPI(procLoginIScsiTargetW,
		uintptr(unsafe.Pointer(targetNamePtr)),
		uintptr(internal.BoolToByte(isInformationalSession)),
		uintptr(unsafe.Pointer(initiatorInstancePtr)),
		uintptr(initiatorPortNumberValue),
		uintptr(unsafe.Pointer(internalPortal)),
		uintptr(securityFlagsValue),
		0, // we don't support mappings yet
		uintptr(unsafe.Pointer(internalLoginOptions)),
		uintptr(keySize),
		uintptr(unsafe.Pointer(keyPtr)),
		uintptr(internal.BoolToByte(isPersistent)),
		uintptr(unsafe.Pointer(sessionID)),
		uintptr(unsafe.Pointer(connectionID)),
	); err != nil {
		return nil, nil, err
	}

	return sessionID, connectionID, nil
}
