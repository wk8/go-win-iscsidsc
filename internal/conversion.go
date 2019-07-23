package internal

// This file contains helpers to convert from public-facing to internal structs.

import (
	"golang.org/x/sys/windows"

	"github.com/pkg/errors"
	iscsidsc "github.com/wk8/go-win-iscsidsc"
)

// CheckAndConvertLoginOptions translates the user-facing `LoginOptions` struct
// into the internal `LoginOptions` struct that the syscalls expect.
// Note that this latter struct contains two `uintptr`s  that map to `PUCHAR`s on the
// C++ side, which means they need to be converted to unsafe pointers in a safe way;
// we achieve that by taking advantage of converting them to unsafe pointers as part
// of a function call's argument lists.
// See https://golang.org/pkg/unsafe/#Pointer (point 4) for more info.
func CheckAndConvertLoginOptions(optsIn *iscsidsc.LoginOptions) (opts *LoginOptions, userNamePtr, passwordPtr *byte, err error) {
	if optsIn == nil {
		optsIn = &iscsidsc.LoginOptions{}
	}

	opts = &LoginOptions{
		// that one must always be the same as per Windows' doc
		Version:    LoginOptionsVersion,
		LoginFlags: optsIn.LoginFlags,
	}

	if optsIn.AuthType != nil {
		opts.AuthType = *optsIn.AuthType
		opts.InformationSpecified |= InformationSpecifiedAuthType
	}
	if optsIn.HeaderDigest != nil {
		opts.HeaderDigest = *optsIn.HeaderDigest
		opts.InformationSpecified |= InformationSpecifiedHeaderDigest
	}
	if optsIn.DataDigest != nil {
		opts.DataDigest = *optsIn.DataDigest
		opts.InformationSpecified |= InformationSpecifiedDataDigest
	}
	if optsIn.MaximumConnections != nil {
		opts.MaximumConnections = *optsIn.MaximumConnections
		opts.InformationSpecified |= InformationSpecifiedMaximumConnections
	}
	if optsIn.DefaultTime2Wait != nil {
		opts.DefaultTime2Wait = *optsIn.DefaultTime2Wait
		opts.InformationSpecified |= InformationSpecifiedDefaultTime2Wait
	}
	if optsIn.DefaultTime2Retain != nil {
		opts.DefaultTime2Retain = *optsIn.DefaultTime2Retain
		opts.InformationSpecified |= InformationSpecifiedDefaultTime2Retain
	}

	if optsIn.Username != nil {
		userNamePtr, err = windows.BytePtrFromString(*optsIn.Username)
		if err != nil {
			err = errors.Wrapf(err, "invalid username: %q", *optsIn.Username)
			return
		}

		opts.UsernameLength = uint32(len(*optsIn.Username))
		opts.InformationSpecified |= InformationSpecifiedUsername
	}
	if optsIn.Password != nil {
		passwordPtr, err = windows.BytePtrFromString(*optsIn.Password)
		if err != nil {
			err = errors.Wrapf(err, "invalid password: %q", *optsIn.Username)
			return
		}

		opts.PasswordLength = uint32(len(*optsIn.Password))
		opts.InformationSpecified |= InformationSpecifiedPassword
	}

	return
}

// CheckAndConvertPortal translates the user-facing `Portal` struct
// into the internal `Portal` struct that the syscalls expect.
func CheckAndConvertPortal(ptlIn *iscsidsc.Portal) (*Portal, error) {
	if ptlIn == nil {
		return nil, nil
	}

	ptl := &Portal{}

	symbolicNameRunes, err := windows.UTF16FromString(ptlIn.SymbolicName)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid portal name: %q", ptlIn.SymbolicName)
	}
	var symbolicName [MaxIscsiPortalNameLen]uint16
	if n := copy(symbolicName[:], symbolicNameRunes); n == MaxIscsiPortalNameLen {
		return nil, errors.Errorf("portal name too long, cannot be more than %d characters", MaxIscsiPortalNameLen)
	}
	ptl.SymbolicName = symbolicName

	addressRunes, err := windows.UTF16FromString(ptlIn.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid portal address: %q", ptlIn.Address)
	}
	var address [MaxIscsiPortalAddressLen]uint16
	if n := copy(address[:], addressRunes); n == MaxIscsiPortalAddressLen {
		return nil, errors.Errorf("portal address too long, cannot be more than %d characters", MaxIscsiPortalAddressLen)
	}
	ptl.Address = address

	if ptlIn.Socket == nil {
		ptl.Socket = DefaultPortalPortNumber
	} else {
		ptl.Socket = *ptlIn.Socket
	}

	return ptl, nil
}

// ConvertInitiatorArgs converts user-facing initiator arguments to internal
// types compatible with Windows' API.
func ConvertInitiatorArgs(initiatorInstance *string, initiatorPortNumber *uint32) (*uint16, uint32, error) {
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

	return initiatorInstancePtr, ConvertInitiatorPortNumber(initiatorPortNumber), nil
}

// ConvertInitiatorPortNumber converts the user-facing initiatorPortNumber
// to an internal type compatible with Windows' API.
func ConvertInitiatorPortNumber(initiatorPortNumber *uint32) uint32 {
	initiatorPortNumberValue := AllInititatorPorts
	if initiatorPortNumber != nil {
		initiatorPortNumberValue = *initiatorPortNumber
	}
	return initiatorPortNumberValue
}

// CheckAndConvertKey converts the user-facing key argument, common to several
// procs, into internal types compatible with Windows' API.
func CheckAndConvertKey(key *string) (keyPtr *byte, keySize uint32, err error) {
	if key != nil {
		if keyPtr, err = windows.BytePtrFromString(*key); err != nil {
			err = errors.Wrapf(err, "invalid key: %q", *key)
			return
		}
		keySize = uint32(len(*key))
	}

	return
}
