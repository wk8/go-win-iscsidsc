package internal

// This file contains the types and constants used internally throughout this repo.
// We need separates struct from the public-facing structs to be able
// to distinguish which fields have been set by the caller - and still have the
// same memory layout as the C++ struct.

import (
	"math"
	"unsafe"

	iscsidsc "github.com/wk8/go-win-iscsidsc"
)

// LoginOptions maps to the `ISCSI_LOGIN_OPTIONS` C++ struct.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_login_options
type LoginOptions struct {
	Version              uint32
	InformationSpecified InformationSpecified
	LoginFlags           iscsidsc.LoginFlags
	AuthType             iscsidsc.AuthType
	HeaderDigest         iscsidsc.DigestType
	DataDigest           iscsidsc.DigestType
	MaximumConnections   uint32
	DefaultTime2Wait     uint32
	DefaultTime2Retain   uint32
	UsernameLength       uint32
	PasswordLength       uint32
	Username             uintptr
	Password             uintptr
}

// AllInititatorPorts maps to the `ISCSI_ALL_INITIATOR_PORTS` C++ constant.
// see the "InitiatorPortNumber" section of https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-addiscsisendtargetportalw
const AllInititatorPorts uint32 = math.MaxUint32

// LoginOptionsVersion maps to the `ISCSI_LOGIN_OPTIONS_VERSION` C++ constant.
// see the "Version" section of https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_login_options
const LoginOptionsVersion uint32 = 0

// InformationSpecified are used in `LoginOptions`.
// see the "InformationSpecified" section of https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_login_options
type InformationSpecified uint32

// The flags for the various pieces of information that can be specified in login options.
const (
	InformationSpecifiedHeaderDigest       InformationSpecified = 0x00000001
	InformationSpecifiedDataDigest         InformationSpecified = 0x00000002
	InformationSpecifiedMaximumConnections InformationSpecified = 0x00000004
	InformationSpecifiedDefaultTime2Wait   InformationSpecified = 0x00000008
	InformationSpecifiedDefaultTime2Retain InformationSpecified = 0x00000010
	InformationSpecifiedUsername           InformationSpecified = 0x00000020
	InformationSpecifiedPassword           InformationSpecified = 0x00000040
	InformationSpecifiedAuthType           InformationSpecified = 0x00000080
)

// Portal maps to the `ISCSI_TARGET_PORTALW` C++ struct.
// We need a separate struct from the public-facing Portal struct for the same reason
// we outlined above for loginOptions/LoginOptions.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_target_portalw
type Portal struct {
	SymbolicName [MaxIscsiPortalNameLen]uint16
	Address      [MaxIscsiPortalAddressLen]uint16
	Socket       uint16
}

// MaxIscsiPortalNameLen maps to the `MAX_ISCSI_PORTAL_NAME_LEN` C++ constant.
const MaxIscsiPortalNameLen = 256

// MaxIscsiPortalAddressLen maps to the `MAX_ISCSI_PORTAL_ADDRESS_LEN` C++ constant.
const MaxIscsiPortalAddressLen = 256

// DefaultPortalPortNumber is the default port on which target portals are expected
// to be listening on if no other port is explicitly provided.
const DefaultPortalPortNumber uint16 = 3260

// MaxHbaNameLen maps to the `MAX_ISCSI_HBANAME_LEN` C++ constant.
const MaxHbaNameLen = 256

// MaxIscsiNameLen maps to the `MAX_ISCSI_NAME_LEN` C++ constant.
const MaxIscsiNameLen = 223

// MaxPath maps to the `MAX_PATH` C++ constant.
const MaxPath = 260

// PortalInfo maps to the `ISCSI_TARGET_PORTAL_INFO_EXW` C++ struct.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_target_portal_info_exw
type PortalInfo struct {
	InitiatorName       [MaxHbaNameLen]uint16
	InitiatorPortNumber uint32
	SymbolicName        [MaxIscsiPortalNameLen]uint16
	Address             [MaxIscsiPortalAddressLen]uint16
	Socket              uint16
	SecurityFlags       iscsidsc.SecurityFlags
	LoginOptions        LoginOptions
}

var (
	emptyPrivatePortalInfo = PortalInfo{}

	// PortalInfoSize is the size, in bytes, of the internal `PortalInfo` type.
	PortalInfoSize = unsafe.Sizeof(emptyPrivatePortalInfo)
)

// ConnectionInfo maps to the `ISCSI_CONNECTION_INFOW` C++ struct.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_connection_infow
type ConnectionInfo struct {
	ConnectionID     iscsidsc.ConnectionID
	InitiatorAddress uintptr
	TargetAddress    uintptr
	InitiatorSocket  uint16
	TargetSocket     uint16
	CID              [2]byte
}

var (
	emptyConnectionInfo = ConnectionInfo{}
	// ConnectionInfoSize is the size, in bytes, of the internal `ConnectionInfo` type.
	ConnectionInfoSize = unsafe.Sizeof(emptyConnectionInfo)
)

// SessionInfo maps to the `ISCSI_SESSION_INFOW` C++ struct.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_session_infow
type SessionInfo struct {
	SessionID       iscsidsc.SessionID
	InitiatorName   uintptr
	TargetNodeName  uintptr
	TargetName      uintptr
	ISID            [6]byte
	TSID            [2]byte
	ConnectionCount uint32
	Connections     uintptr
}

var (
	emptySessionInfo = SessionInfo{}
	// SessionInfoSize is the size, in bytes, of the internal `SessionInfo` type.
	SessionInfoSize = unsafe.Sizeof(emptySessionInfo)
)

// Device maps to the `ISCSI_DEVICE_ON_SESSIONW` C++ struct.
// see https://docs.microsoft.com/en-us/windows/win32/api/iscsidsc/ns-iscsidsc-iscsi_device_on_sessionw
type Device struct {
	InitiatorName       [MaxHbaNameLen]uint16
	TargetName          [MaxIscsiNameLen + 1]uint16
	ScsiAddress         ScsiAddress
	DeviceInterfaceType GUID
	DeviceInterfaceName [MaxPath]uint16
	LegacyName          [MaxPath]uint16
	StorageDeviceNumber iscsidsc.StorageDeviceNumber
	DeviceInstance      uint32
}

var (
	emptyDevice = Device{}
	// DeviceSize is the size, in bytes, of the internal `Device` type.
	DeviceSize = unsafe.Sizeof(emptyDevice)
)

// GUID maps to the `GUID` C++ struct
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// ScsiAddress maps to the `SCSI_ADDRESS` C++ struct.
// see https://docs.microsoft.com/en-us/windows-hardware/drivers/ddi/content/ntddscsi/ns-ntddscsi-_scsi_address
type ScsiAddress struct {
	Length     uint32
	PortNumber uint8
	PathID     uint8
	TargetID   uint8
	Lun        uint8
}
