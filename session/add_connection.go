package session

import (
	"unsafe"

	"github.com/pkg/errors"
	iscsidsc "github.com/wk8/go-win-iscsidsc"
	"github.com/wk8/go-win-iscsidsc/internal"
)

var procAddIScsiConnectionW = internal.GetDllProc("AddIScsiConnectionW")

// AddIScsiConnection adds a new iSCSI connection to an existing session.
// Only the session ID and the targetPortal are required.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-addiscsiconnectionw
func AddIScsiConnection(id iscsidsc.SessionID, initiatorPortNumber *uint32, targetPortal *iscsidsc.Portal,
	securityFlags *iscsidsc.SecurityFlags, loginOptions *iscsidsc.LoginOptions, key *string) (*iscsidsc.ConnectionID, error) {

	initiatorPortNumberValue := internal.ConvertInitiatorPortNumber(initiatorPortNumber)

	if targetPortal == nil {
		return nil, errors.Errorf("targetPortal is required")
	}
	internalPortal, err := internal.CheckAndConvertPortal(targetPortal)
	if err != nil {
		return nil, errors.Wrap(err, "invalid targetPortal argument")
	}

	var securityFlagsValue iscsidsc.SecurityFlags
	if securityFlags != nil {
		securityFlagsValue = *securityFlags
	}

	internalLoginOptions, userNamePtr, passwordPtr, err := internal.CheckAndConvertLoginOptions(loginOptions)
	if err != nil {
		return nil, errors.Wrap(err, "invalid loginOptions argument")
	}

	keyPtr, keySize, err := internal.CheckAndConvertKey(key)
	if err != nil {
		return nil, err
	}

	return callProcAddIScsiConnectionW(id, initiatorPortNumberValue, internalPortal, securityFlagsValue,
		internalLoginOptions, uintptr(unsafe.Pointer(userNamePtr)), uintptr(unsafe.Pointer(passwordPtr)),
		keyPtr, keySize)
}

//go:uintptrescapes
//go:noinline

func callProcAddIScsiConnectionW(id iscsidsc.SessionID, initiatorPortNumberValue uint32, internalPortal *internal.Portal,
	securityFlagsValue iscsidsc.SecurityFlags, internalLoginOptions *internal.LoginOptions,
	userNameUintptr, passwordUintptr uintptr, keyPtr *byte, keySize uint32) (*iscsidsc.ConnectionID, error) {

	internalLoginOptions.Username = userNameUintptr
	internalLoginOptions.Password = passwordUintptr

	connectionID := &iscsidsc.ConnectionID{}

	if _, err := internal.CallWinAPI(procAddIScsiConnectionW,
		uintptr(unsafe.Pointer(&id)),
		0, // reserved pointer argument, must be null on input
		uintptr(initiatorPortNumberValue),
		uintptr(unsafe.Pointer(internalPortal)),
		uintptr(securityFlagsValue),
		uintptr(unsafe.Pointer(internalLoginOptions)),
		uintptr(keySize),
		uintptr(unsafe.Pointer(keyPtr)),
		uintptr(unsafe.Pointer(connectionID)),
	); err != nil {
		return nil, err
	}

	return connectionID, nil
}
