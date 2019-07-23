package targetportal

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/pkg/errors"
	iscsidsc "github.com/wk8/go-win-iscsidsc"
	"github.com/wk8/go-win-iscsidsc/internal"
)

var procReportIScsiSendTargetPortalsExW = internal.GetDllProc("ReportIScsiSendTargetPortalsExW")

// ReportIScsiSendTargetPortals retrieves a list of static target portals that the iSCSI initiator
// service uses to perform automatic discovery with SendTarget requests.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-reportiscsisendtargetportalsexw
func ReportIScsiSendTargetPortals() ([]iscsidsc.PortalInfo, error) {
	buffer, bufferPointer, count, err := retrievePortalInfos()
	if err != nil {
		return nil, err
	}

	portalInfos, bytesRead, err := hydrateTargetPortalInfos(buffer, bufferPointer, int(count))
	if err != nil {
		return nil, err
	}
	if bytesRead != uintptr(len(buffer)) {
		return nil, hydrateTargetPortalError("reply was %d bytes long, read %d bytes", len(buffer), bytesRead)
	}

	return portalInfos, nil
}

// retrievePortalInfos gets the raw portal infos from the Windows API.
func retrievePortalInfos() (buffer []byte, bufferPointer uintptr, count int32, err error) {
	return internal.HandleBufferedWinAPICall(
		func(s, c, b uintptr) (uintptr, error) {
			return internal.CallWinAPI(procReportIScsiSendTargetPortalsExW, c, s, b)
		},
		procReportIScsiSendTargetPortalsExW.Name,
		1,
	)
}

// hydrateTargetPortalInfos takes the raw bytes returned by the `ReportIscsiSendTargetPortalsEx` C++ proc,
// and casts the raw data into Go structs.
// Also returns the total number of bytes it's read from the buffer.
func hydrateTargetPortalInfos(buffer []byte, bufferPointer uintptr, count int) ([]iscsidsc.PortalInfo, uintptr, error) {
	// sanity check: the total size should be at least enough to contain the portal infos
	minimumExpectedSize := count * int(internal.PortalInfoSize)
	if len(buffer) < minimumExpectedSize {
		return nil, 0, hydrateTargetPortalError("expected the reply to be at least %d bytes, only got %d bytes", minimumExpectedSize, len(buffer))
	}

	portalInfos := make([]iscsidsc.PortalInfo, count)
	var bytesRead uintptr
	for i := 0; i < count; i++ {
		read, err := hydrateTargetPortalInfo(buffer, bufferPointer, i, &portalInfos[i])
		bytesRead += read
		if err != nil {
			return nil, bytesRead, err
		}
	}

	return portalInfos, bytesRead, nil
}

// hydrateTargetPortalInfo hydrates a single `PortalInfo` struct.
// It returns the number of bytes it's read from the buffer.
func hydrateTargetPortalInfo(buffer []byte, bufferPointer uintptr, i int, info *iscsidsc.PortalInfo) (uintptr, error) {
	// we already know that we're still in the buffer here - we check that at the very start of `hydrateTargetPortalInfos`,
	// so this is safe as per rule (1) of https://golang.org/pkg/unsafe/#Pointer
	infoIn := (*internal.PortalInfo)(unsafe.Pointer(&buffer[uintptr(i)*internal.PortalInfoSize]))
	bytesRead := internal.PortalInfoSize

	info.Portal = *hydratePortal(infoIn)
	info.InitiatorName = windows.UTF16ToString(infoIn.InitiatorName[:])
	info.InitiatorPortNumber = infoIn.InitiatorPortNumber
	info.SecurityFlags = infoIn.SecurityFlags

	loginOptions, read, err := hydrateLoginOptions(&infoIn.LoginOptions, buffer, bufferPointer)
	info.LoginOptions = *loginOptions
	bytesRead += read
	return bytesRead, err
}

func hydratePortal(infoIn *internal.PortalInfo) *iscsidsc.Portal {
	socket := infoIn.Socket
	return &iscsidsc.Portal{
		SymbolicName: windows.UTF16ToString(infoIn.SymbolicName[:]),
		Address:      windows.UTF16ToString(infoIn.Address[:]),
		Socket:       &socket,
	}
}

func hydrateLoginOptions(optsIn *internal.LoginOptions, buffer []byte, bufferPointer uintptr) (*iscsidsc.LoginOptions, uintptr, error) {
	opts := &iscsidsc.LoginOptions{LoginFlags: optsIn.LoginFlags}

	if optsIn.InformationSpecified&internal.InformationSpecifiedAuthType != 0 {
		authType := optsIn.AuthType
		opts.AuthType = &authType
	}
	if optsIn.InformationSpecified&internal.InformationSpecifiedHeaderDigest != 0 {
		headerDigest := optsIn.HeaderDigest
		opts.HeaderDigest = &headerDigest
	}
	if optsIn.InformationSpecified&internal.InformationSpecifiedDataDigest != 0 {
		dataDigest := optsIn.DataDigest
		opts.DataDigest = &dataDigest
	}
	if optsIn.InformationSpecified&internal.InformationSpecifiedMaximumConnections != 0 {
		maximumConnections := optsIn.MaximumConnections
		opts.MaximumConnections = &maximumConnections
	}
	if optsIn.InformationSpecified&internal.InformationSpecifiedDefaultTime2Wait != 0 {
		defaultTime2Wait := optsIn.DefaultTime2Wait
		opts.DefaultTime2Wait = &defaultTime2Wait
	}
	if optsIn.InformationSpecified&internal.InformationSpecifiedDefaultTime2Retain != 0 {
		defaultTime2Retain := optsIn.DefaultTime2Retain
		opts.DefaultTime2Retain = &defaultTime2Retain
	}

	var bytesRead uintptr
	if optsIn.InformationSpecified&internal.InformationSpecifiedUsername != 0 && optsIn.UsernameLength != 0 && optsIn.Username != 0 {
		username, err := internal.ExtractStringFromBuffer(buffer, bufferPointer, optsIn.Username, uintptr(optsIn.UsernameLength))
		if err != nil {
			return nil, bytesRead, hydrateTargetPortalError(" could not read login username: %v", err)
		}
		bytesRead += uintptr(optsIn.UsernameLength)
		opts.Username = &username
	}
	if optsIn.InformationSpecified&internal.InformationSpecifiedPassword != 0 && optsIn.PasswordLength != 0 && optsIn.Password != 0 {
		password, err := internal.ExtractStringFromBuffer(buffer, bufferPointer, optsIn.Password, uintptr(optsIn.PasswordLength))
		if err != nil {
			return nil, bytesRead, hydrateTargetPortalError(" could not read login password: %v", err)
		}
		bytesRead += uintptr(optsIn.PasswordLength)
		opts.Password = &password
	}

	return opts, bytesRead, nil
}

func hydrateTargetPortalError(format string, args ...interface{}) error {
	msg := fmt.Sprintf("Error when hydrating the response from %s - it might be that your Windows version is not supported: ", procReportIScsiSendTargetPortalsExW.Name)
	return errors.Errorf(msg+format, args...)
}
