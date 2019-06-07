// +build windows

// This file contains the procs to create, get and remove target portals.

package targetportal

import (
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"

	"github.com/wk8/go-win-iscsidsc"
)

var (
	procAddIScsiSendTargetPortalW       = iscsidsc.IscsidscDLL.NewProc("AddIScsiSendTargetPortalW")
	procReportIScsiSendTargetPortalsExW = iscsidsc.IscsidscDLL.NewProc("ReportIScsiSendTargetPortalsExW")
	procRemoveIScsiSendTargetPortalW    = iscsidsc.IscsidscDLL.NewProc("RemoveIScsiSendTargetPortalW")
)

// AddIScsiSendTargetPortal adds a static target portal to the list of target portals to which the iSCSI initiator service transmits SendTargets requests.
// Only the `portal` is a required argument - all others can be left `nil`.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-addiscsisendtargetportalw
func AddIScsiSendTargetPortal(initiatorInstance *string, initiatorPortNumber *uint32, loginOptions *LoginOptions, securityFlags *SecurityFlags, portal *Portal) error {
	initiatorInstancePtr, initiatorPortNumberValue, err := convertInitiatorArgs(initiatorInstance, initiatorPortNumber)
	if err != nil {
		return err
	}

	if loginOptions == nil {
		loginOptions = &LoginOptions{}
	}
	privateLoginOptions, userNamePtr, passwordPtr, err := checkAndConvertLoginOptions(loginOptions)
	if err != nil {
		return errors.Wrap(err, "invalid loginOptions argument")
	}

	var securityFlagsValue SecurityFlags
	if securityFlags != nil {
		securityFlagsValue = *securityFlags
	}

	privatePortal, err := checkAndConvertPortal(portal)
	if err != nil {
		return errors.Wrap(err, "invalid portal argument")
	}

	_, err = callProcAddIScsiSendTargetPortalW(
		initiatorInstancePtr,
		initiatorPortNumberValue,
		privateLoginOptions,
		securityFlagsValue,
		privatePortal,
		uintptr(unsafe.Pointer(userNamePtr)),
		uintptr(unsafe.Pointer(passwordPtr)),
	)

	return err
}

// Alias for AddIScsiSendTargetPortal/5.
func (portalInfo *PortalInfo) AddIScsiSendTargetPortal() error {
	initiatorInstance, initiatorPortNumber := portalInfo.extractInitiatorArgs()
	return AddIScsiSendTargetPortal(initiatorInstance, initiatorPortNumber, &portalInfo.LoginOptions, &portalInfo.SecurityFlags, &portalInfo.Portal)
}

// FIXME: unit tests
// checkAndConvertLoginOptions translates the user-facing `LoginOptions` struct
// into the `loginOptions` struct that the syscall expects.
// Note that this latter struct contains two `uintptr`s  that map to `PUCHAR`s on the
// C++ side, which means they need to be converted to unsafe pointers in a safe way;
// we achieve that by taking advantage of converting them to unsafe pointers as part
// of a function call's argument lists.
// See https://golang.org/pkg/unsafe/#Pointer (point 4) for more info.
func checkAndConvertLoginOptions(optsIn *LoginOptions) (opts *loginOptions, userNamePtr, passwordPtr *byte, err error) {
	opts = &loginOptions{
		// that one must always be the same as per Windows' doc
		version:    loginOptionsVersion,
		loginFlags: optsIn.LoginFlags,
	}

	if optsIn.AuthType != nil {
		opts.authType = *optsIn.AuthType
		opts.informationSpecified |= informationSpecifiedAuthType
	}
	if optsIn.HeaderDigest != nil {
		opts.headerDigest = *optsIn.HeaderDigest
		opts.informationSpecified |= informationSpecifiedHeaderDigest
	}
	if optsIn.DataDigest != nil {
		opts.dataDigest = *optsIn.DataDigest
		opts.informationSpecified |= informationSpecifiedDataDigest
	}
	if optsIn.MaximumConnections != nil {
		opts.maximumConnections = *optsIn.MaximumConnections
		opts.informationSpecified |= informationSpecifiedMaximumConnections
	}
	if optsIn.DefaultTime2Wait != nil {
		opts.defaultTime2Wait = *optsIn.DefaultTime2Wait
		opts.informationSpecified |= informationSpecifiedDefaultTime2Wait
	}
	if optsIn.DefaultTime2Retain != nil {
		opts.defaultTime2Retain = *optsIn.DefaultTime2Retain
		opts.informationSpecified |= informationSpecifiedDefaultTime2Retain
	}

	if optsIn.Username != nil {
		userNamePtr, err = windows.BytePtrFromString(*optsIn.Username)
		if err != nil {
			err = errors.Wrapf(err, "invalid username: %q", *optsIn.Username)
			return
		}

		opts.usernameLength = uint32(len(*optsIn.Username))
		opts.informationSpecified |= informationSpecifiedUsername
	}
	if optsIn.Password != nil {
		passwordPtr, err = windows.BytePtrFromString(*optsIn.Password)
		if err != nil {
			err = errors.Wrapf(err, "invalid password: %q", *optsIn.Username)
			return
		}

		opts.passwordLength = uint32(len(*optsIn.Password))
		opts.informationSpecified |= informationSpecifiedPassword
	}

	return
}

// FIXME: unit tests
// checkAndConvertPortal translates the user-facing `Portal` struct
// into the `portal` struct that the syscall expects.
func checkAndConvertPortal(ptlIn *Portal) (*portal, error) {
	if ptlIn == nil {
		return nil, errors.Errorf("portal is nil")
	}

	ptl := &portal{}

	symbolicNameRunes, err := windows.UTF16FromString(ptlIn.SymbolicName)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid portal name: %q", ptlIn.SymbolicName)
	}
	var symbolicName [maxIscsiPortalNameLen]uint16
	if n := copy(symbolicName[:], symbolicNameRunes); n == maxIscsiPortalNameLen {
		return nil, errors.Errorf("portal name too long, cannot be more than %d characters", maxIscsiPortalNameLen)
	}
	ptl.symbolicName = symbolicName

	addressRunes, err := windows.UTF16FromString(ptlIn.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid portal address: %q", ptlIn.Address)
	}
	var address [maxIscsiPortalAddressLen]uint16
	if n := copy(address[:], addressRunes); n == maxIscsiPortalAddressLen {
		return nil, errors.Errorf("portal address too long, cannot be more than %d characters", maxIscsiPortalAddressLen)
	}
	ptl.address = address

	if ptlIn.Socket == nil {
		ptl.socket = defaultPortalPortNumber
	} else {
		ptl.socket = *ptlIn.Socket
	}

	return ptl, nil
}

// DO NOT remove these 2 comments, that's really important! See below.

//go:uintptrescapes
//go:noinline

// callProcAddIScsiSendTargetPortalW is only a wrapper around `procAddIScsiSendTargetPortalW.Call`.
// Its main purpose is that the unsafe pointers to the username and password strings are
// guaranteed to stay in the same place in memory until this function returns.
// See `checkAndConvertLoginOptions`'s doc comment as well as https://golang.org/pkg/unsafe/#Pointer
// for more context.
func callProcAddIScsiSendTargetPortalW(initiatorInstancePtr *uint16, initiatorPortNumberValue uint32,
	privateLoginOptions *loginOptions, securityFlagsValue SecurityFlags, privatePortal *portal,
	userNameUintptr, passwordUintptr uintptr) (uintptr, error) {

	privateLoginOptions.username = userNameUintptr
	privateLoginOptions.password = passwordUintptr

	return iscsidsc.CallWinApi(procAddIScsiSendTargetPortalW,
		uintptr(unsafe.Pointer(initiatorInstancePtr)),
		uintptr(initiatorPortNumberValue),
		uintptr(unsafe.Pointer(privateLoginOptions)),
		uintptr(securityFlagsValue),
		uintptr(unsafe.Pointer(privatePortal)),
	)
}

// ReportIscsiSendTargetPortalsEx function retrieves a list of static target portals that the iSCSI initiator
// service uses to perform automatic discovery with SendTarget requests.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-reportiscsisendtargetportalsexw
func ReportIScsiSendTargetPortals() ([]PortalInfo, error) {
	buffer, bufferPointer, totalSize, count, err := retrievePortalInfo()
	if err != nil {
		return nil, err
	}

	portalInfos, bytesRead, err := hydrateTargetPortalInfos(buffer, bufferPointer, totalSize, int(count))
	if err != nil {
		return nil, err
	}
	if bytesRead != totalSize {
		return nil, hydrateTargetPortalError("reply was %d bytes long, read %d bytes", totalSize, bytesRead)
	}

	return portalInfos, nil
}

// retrievePortalInfo gets the raw portals info from the Windows API.
// FIXME: ensure that it does work when making a call with too small a buffer, with a test!
func retrievePortalInfo() (buffer []byte, bufferPointer, bufferSize uintptr, count int32, err error) {
	bufferSize = iscsidsc.InitialApiBufferSize
	var exitCode uintptr

	for {
		buffer = make([]byte, bufferSize)

		exitCode, bufferPointer, err = callProcReportIScsiSendTargetPortalsExW(
			uintptr(unsafe.Pointer(&count)),
			uintptr(unsafe.Pointer(&bufferSize)),
			uintptr(unsafe.Pointer(&buffer[0])),
		)

		if exitCode != uintptr(syscall.ERROR_INSUFFICIENT_BUFFER) {
			return
		}

		// sanity check: is the new buffer size indeed bigger than the previous one?
		if int(bufferSize) <= len(buffer) {
			// this should never happen
			err = errors.Errorf("Error when calling %q: buffer of size %d deemed too small but bigger than the new advised size of %d", procReportIScsiSendTargetPortalsExW.Name, len(buffer), bufferSize)
			return
		}
		// try again with a bigger buffer
	}
}

//go:uintptrescapes
//go:noinline

// callProcReportIScsiSendTargetPortalsExW fulfills the same role as `callProcAddIScsiSendTargetPortalW`:
// it ensures that we can get the address of the buffer we use before it's moved in memory, so that
// we can then resolve pointers to itself it contains.
func callProcReportIScsiSendTargetPortalsExW(count, bufferSize, buffer uintptr) (uintptr, uintptr, error) {
	exitCode, err := iscsidsc.CallWinApi(procReportIScsiSendTargetPortalsExW, count, bufferSize, buffer)
	return exitCode, buffer, err
}

// hydrateTargetPortalInfos takes the raw bytes returned by the `ReportIscsiSendTargetPortalsEx` C++ proc,
// and casts the raw data into Go structs.
// Also returns the total number of bytes it's read from the buffer.
func hydrateTargetPortalInfos(buffer []byte, bufferPointer, totalSize uintptr, count int) ([]PortalInfo, uintptr, error) {
	// sanity check: the total size should be at least enough to contain the portal infos
	minimumExpectedSize := uintptr(count) * privatePortalInfoSize
	if totalSize < minimumExpectedSize {
		return nil, 0, hydrateTargetPortalError("expected the reply to be at least %d bytes, only got %d bytes", minimumExpectedSize, totalSize)
	}

	portalInfos := make([]PortalInfo, count)
	var bytesRead uintptr = 0
	for i := 0; i < count; i++ {
		read, err := hydrateTargetPortalInfo(buffer, bufferPointer, totalSize, i, &portalInfos[i])
		bytesRead += read
		if err != nil {
			return nil, bytesRead, err
		}
	}

	return portalInfos, bytesRead, nil
}

// hydrateTargetPortalInfo hydrates a single `PortalInfo` struct.
// It returns the number of bytes it's read from the buffer.
func hydrateTargetPortalInfo(buffer []byte, bufferPointer, totalSize uintptr, i int, info *PortalInfo) (uintptr, error) {
	// we already know that we're still in the buffer here - we check that at the very start of `hydrateTargetPortalInfos`,
	// so this is safe as per rule (1) of https://golang.org/pkg/unsafe/#Pointer
	infoIn := (*portalInfo)(unsafe.Pointer(&buffer[uintptr(i)*privatePortalInfoSize]))
	bytesRead := privatePortalInfoSize

	info.Portal = *hydratePortal(infoIn)
	info.InitiatorName = windows.UTF16ToString(infoIn.initiatorName[:])
	info.InitiatorPortNumber = infoIn.initiatorPortNumber
	info.SecurityFlags = infoIn.SecurityFlags

	loginOptions, read, err := hydrateLoginOptions(&infoIn.loginOptions, buffer, bufferPointer, totalSize)
	info.LoginOptions = *loginOptions
	bytesRead += read
	return bytesRead, err
}

func hydratePortal(infoIn *portalInfo) *Portal {
	socket := infoIn.Socket
	return &Portal{
		SymbolicName: windows.UTF16ToString(infoIn.symbolicName[:]),
		Address:      windows.UTF16ToString(infoIn.address[:]),
		Socket:       &socket,
	}
}

func hydrateLoginOptions(optsIn *loginOptions, buffer []byte, bufferPointer, totalSize uintptr) (*LoginOptions, uintptr, error) {
	opts := &LoginOptions{LoginFlags: optsIn.loginFlags}

	if optsIn.informationSpecified&informationSpecifiedAuthType != 0 {
		authType := optsIn.authType
		opts.AuthType = &authType
	}
	if optsIn.informationSpecified&informationSpecifiedHeaderDigest != 0 {
		headerDigest := optsIn.headerDigest
		opts.HeaderDigest = &headerDigest
	}
	if optsIn.informationSpecified&informationSpecifiedDataDigest != 0 {
		dataDigest := optsIn.dataDigest
		opts.DataDigest = &dataDigest
	}
	if optsIn.informationSpecified&informationSpecifiedMaximumConnections != 0 {
		maximumConnections := optsIn.maximumConnections
		opts.MaximumConnections = &maximumConnections
	}
	if optsIn.informationSpecified&informationSpecifiedDefaultTime2Wait != 0 {
		defaultTime2Wait := optsIn.defaultTime2Wait
		opts.DefaultTime2Wait = &defaultTime2Wait
	}
	if optsIn.informationSpecified&informationSpecifiedDefaultTime2Retain != 0 {
		defaultTime2Retain := optsIn.defaultTime2Retain
		opts.DefaultTime2Retain = &defaultTime2Retain
	}

	var bytesRead uintptr = 0
	if optsIn.informationSpecified&informationSpecifiedUsername != 0 && optsIn.usernameLength != 0 && optsIn.username != 0 {
		username, read, err := extractStringFromBuffer(buffer, bufferPointer, optsIn.username, totalSize, uintptr(optsIn.usernameLength))
		bytesRead += read
		if err != nil {
			return nil, bytesRead, err
		}
		opts.Username = &username
	}
	if optsIn.informationSpecified&informationSpecifiedPassword != 0 && optsIn.passwordLength != 0 && optsIn.password != 0 {
		password, read, err := extractStringFromBuffer(buffer, bufferPointer, optsIn.password, totalSize, uintptr(optsIn.passwordLength))
		bytesRead += read
		if err != nil {
			return nil, bytesRead, err
		}
		opts.Password = &password
	}

	return opts, bytesRead, nil
}

func extractStringFromBuffer(buffer []byte, bufferPointer, stringPointer, totalSize, stringSize uintptr) (string, uintptr, error) {
	// first let's compute the offset at which we should find the string in the buffer:
	// the pointer address we have might not be valid any more, since the GC might have moved the
	// buffer internally; that's why we use the address of the start of the buffer as it was when we made
	// the syscall
	bufferOffset := stringPointer - bufferPointer
	// sanity check: this should still be inside the buffer
	if bufferOffset < 0 || bufferOffset+stringSize > totalSize {
		return "", 0, hydrateTargetPortalError("string pointer pointing out of the buffer")
	}

	strBytes := make([]byte, stringSize)
	copy(strBytes, buffer[bufferOffset:bufferOffset+stringSize])
	return string(strBytes), stringSize, nil
}

func hydrateTargetPortalError(format string, args ...interface{}) error {
	return errors.Errorf("Error when hydrating the response from ReportIScsiSendTargetPortalsExW - it might be that your Windows version is not supported: "+format, args...)
}

// RemoveIscsiSendTargetPortal removes a portal from the list of portals to which the iSCSI initiator service sends
// SendTargets requests for target discovery.
// Only portal is a required argument.
func RemoveIScsiSendTargetPortal(initiatorInstance *string, initiatorPortNumber *uint32, portal *Portal) error {
	initiatorInstancePtr, initiatorPortNumberValue, err := convertInitiatorArgs(initiatorInstance, initiatorPortNumber)
	if err != nil {
		return err
	}

	privatePortal, err := checkAndConvertPortal(portal)
	if err != nil {
		return errors.Wrap(err, "invalid portal argument")
	}

	_, err = iscsidsc.CallWinApi(procRemoveIScsiSendTargetPortalW,
		uintptr(unsafe.Pointer(initiatorInstancePtr)),
		uintptr(initiatorPortNumberValue),
		uintptr(unsafe.Pointer(privatePortal)),
	)

	return err
}

// Alias for RemoveIScsiSendTargetPortal/3.
func (portalInfo *PortalInfo) RemoveIScsiSendTargetPortal() error {
	initiatorInstance, initiatorPortNumber := portalInfo.extractInitiatorArgs()
	return RemoveIScsiSendTargetPortal(initiatorInstance, initiatorPortNumber, &portalInfo.Portal)

}

func convertInitiatorArgs(initiatorInstance *string, initiatorPortNumber *uint32) (*uint16, uint32, error) {
	var (
		initiatorInstancePtr *uint16
		err                  error
	)
	if initiatorInstance != nil {
		initiatorInstancePtr, err = windows.UTF16PtrFromString(*initiatorInstance)
		if err != nil {
			return nil, 0, errors.Wrapf(err, "invalid initiatorInstance argument: %q", *initiatorInstance)
		}
	}

	initiatorPortNumberValue := allInititatorPorts
	if initiatorPortNumber != nil {
		initiatorPortNumberValue = *initiatorPortNumber
	}

	return initiatorInstancePtr, initiatorPortNumberValue, nil
}

func (portalInfo *PortalInfo) extractInitiatorArgs() (initiatorInstance *string, initiatorPortNumber *uint32) {
	if portalInfo.InitiatorName != "" {
		initiatorInstance = &portalInfo.InitiatorName
	}
	if portalInfo.InitiatorPortNumber > 0 {
		initiatorPortNumber = &portalInfo.InitiatorPortNumber
	}
	return
}
