package targetportal

import (
	"math"
	"unsafe"
)

// This file contains the private types and constants used internally throughout this package.

// loginOptions maps to the `ISCSI_LOGIN_OPTIONS` C++ struct.
// We need a separate struct from the public-facing LoginOptions struct here to be able
// to distinguish which fields have been set by the caller - and still have the
// same memory layout as the C°° struct.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_login_options
type loginOptions struct {
	version              uint32
	informationSpecified informationSpecified
	loginFlags           LoginFlags
	authType             AuthType
	headerDigest         DigestType
	dataDigest           DigestType
	maximumConnections   uint32
	defaultTime2Wait     uint32
	defaultTime2Retain   uint32
	usernameLength       uint32
	passwordLength       uint32
	username             uintptr
	password             uintptr
}

// allInititatorPorts maps to the `ISCSI_ALL_INITIATOR_PORTS` C++ constant.
// see the "InitiatorPortNumber" section of https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-addiscsisendtargetportalw
const allInititatorPorts uint32 = math.MaxUint32

// loginOptionsVersion maps to the `ISCSI_LOGIN_OPTIONS_VERSION` C++ constant.
// see the "Version" section of https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_login_options
const loginOptionsVersion uint32 = 0

// informationSpecified are used in `LoginOptions`.
// see the "InformationSpecified" section of https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_login_options
type informationSpecified uint32

const (
	informationSpecifiedHeaderDigest       informationSpecified = 0x00000001
	informationSpecifiedDataDigest         informationSpecified = 0x00000002
	informationSpecifiedMaximumConnections informationSpecified = 0x00000004
	informationSpecifiedDefaultTime2Wait   informationSpecified = 0x00000008
	informationSpecifiedDefaultTime2Retain informationSpecified = 0x00000010
	informationSpecifiedUsername           informationSpecified = 0x00000020
	informationSpecifiedPassword           informationSpecified = 0x00000040
	informationSpecifiedAuthType           informationSpecified = 0x00000080
)

// portal maps to the `ISCSI_TARGET_PORTALW` C++ struct.
// We need a separate struct from the public-facing Portal struct for the same reason
// we outlined above for loginOptions/LoginOptions.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_target_portalw
type portal struct {
	symbolicName [maxIscsiPortalNameLen]uint16
	address      [maxIscsiPortalAddressLen]uint16
	socket       uint16
}

// MaxIscsiPortalNameLen maps to the `MAX_ISCSI_PORTAL_NAME_LEN` C++ constant.
const maxIscsiPortalNameLen = 256

// MaxIscsiPortalAddressLen maps to the `MAX_ISCSI_PORTAL_ADDRESS_LEN` C++ constant.
const maxIscsiPortalAddressLen = 256

const defaultPortalPortNumber uint16 = 3260

// maxHbaNameLen maps to the `MAX_ISCSI_HBANAME_LEN` C++ constant.
const maxHbaNameLen = 256

// portalInfo maps to the `ISCSI_TARGET_PORTAL_INFO_EXW` C++ struct.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_target_portal_info_exw
type portalInfo struct {
	initiatorName       [maxHbaNameLen]uint16
	initiatorPortNumber uint32
	symbolicName        [maxIscsiPortalNameLen]uint16
	address             [maxIscsiPortalAddressLen]uint16
	Socket              uint16
	SecurityFlags       SecurityFlags
	loginOptions        loginOptions
}

var emptyPrivatePortalInfo = portalInfo{}
var privatePortalInfoSize = unsafe.Sizeof(emptyPrivatePortalInfo)
