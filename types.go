package iscsidsc

// This file contains the public types and constants needed to call this package's API.
//
// For all enums below, note that the MSVC compiler seems to always use 4-byte ints for enums,
// though I couldn't get any official documentation on this... Hopefully it doesn't change any time soon!

import (
	"github.com/google/uuid"
)

// Portal maps to the `ISCSI_TARGET_PORTALW` C++ struct.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_target_portalw
type Portal struct {
	// the Windows name
	SymbolicName string
	// IP address or DNS name
	Address string
	// port number - if left empty, defaults to 3260
	Socket *uint16
}

// LoginFlags are used in `LoginOptions`.
// see the "LoginFlags" section of https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_login_options
type LoginFlags uint32

// See the Windows documentation for an explanation of each of which each of
// these maps to.
// The 3rd bit maps to `ISCSI_LOGIN_FLAG_RESERVED1`, and is reserved for
// Windows' API internal usage.
const (
	LoginFlagRequireIPSec          LoginFlags = 0x00000001
	LoginFlagMultipathEnabled      LoginFlags = 0x00000002
	LoginFlagAllowPortalHopping    LoginFlags = 0x00000008
	LoginFlagUseRadiusResponse     LoginFlags = 0x00000010
	LoginFlagUseRadiusVerification LoginFlags = 0x00000020
)

// AuthType maps to the `ISCSI_AUTH_TYPES` C++ enum.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ne-iscsidsc-iscsi_auth_types
type AuthType uint32

// The various auth types available.
const (
	NoAuthAuthType AuthType = iota
	CHAPAuthType
	MutualCHAPAuthType
)

// DigestType maps to the `ISCSI_DIGEST_TYPES` C++ enum.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ne-iscsidsc-iscsi_digest_types
type DigestType uint32

// The various digest types available.
const (
	DigestTypeNone DigestType = iota
	DigestTypeCRC32C
)

// LoginOptions maps to the `ISCSI_LOGIN_OPTIONS` C++ struct.
// All pointer fields are optional and can be left nil.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_login_options
type LoginOptions struct {
	LoginFlags         LoginFlags
	AuthType           *AuthType
	HeaderDigest       *DigestType
	DataDigest         *DigestType
	MaximumConnections *uint32
	DefaultTime2Wait   *uint32
	DefaultTime2Retain *uint32
	Username           *string
	Password           *string
}

// SecurityFlags are one of`AddIScsiSendTargetPortalW`'s arguments.
// see the "SecurityFlags" section of https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-addiscsisendtargetportalw
type SecurityFlags uint64

// The various security flags available.
const (
	SecurityFlagIkeIpsecEnabled        SecurityFlags = 0x00000002
	SecurityFlagMainModeEnabled        SecurityFlags = 0x00000004
	SecurityFlagAggressiveModeEnabled  SecurityFlags = 0x00000008
	SecurityFlagPfsEnabled             SecurityFlags = 0x00000010
	SecurityFlagTransportModePreferred SecurityFlags = 0x00000020
	SecurityFlagTunnelModePreferred    SecurityFlags = 0x00000040
)

// PortalInfo aggregates a portal with all the metadata attached to it.
// It maps to the `ISCSI_TARGET_PORTAL_INFO_EXW` C++ struct.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_target_portal_info_exw
type PortalInfo struct {
	Portal
	InitiatorName       string
	InitiatorPortNumber uint32
	SecurityFlags       SecurityFlags
	LoginOptions        LoginOptions
}

// SessionID maps to the `ISCSI_UNIQUE_SESSION_ID` C++ struct.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_unique_session_id
type SessionID struct {
	AdapterUnique   uint64
	AdapterSpecific uint64
}

// ConnectionID maps to the `ISCSI_UNIQUE_CONNECTION_ID` C++ struct.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_unique_session_id
type ConnectionID struct {
	AdapterUnique   uint64
	AdapterSpecific uint64
}

// ConnectionInfo maps to the `ISCSI_CONNECTION_INFOW` C++ struct.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_connection_infow
type ConnectionInfo struct {
	ConnectionID     ConnectionID
	InitiatorAddress string
	TargetAddress    string
	InitiatorSocket  uint16
	TargetSocket     uint16
	CID              [2]byte
}

// SessionInfo maps to the `ISCSI_SESSION_INFOW` C++ struct.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/ns-iscsidsc-iscsi_session_infow
type SessionInfo struct {
	SessionID      SessionID
	InitiatorName  string
	TargetNodeName string
	TargetName     string
	ISID           [6]byte
	TSID           [2]byte
	Connections    []ConnectionInfo
}

// Device maps to the `ISCSI_DEVICE_ON_SESSIONW` C++ struct.
// see https://docs.microsoft.com/en-us/windows/win32/api/iscsidsc/ns-iscsidsc-iscsi_device_on_sessionw
type Device struct {
	InitiatorName       string
	TargetName          string
	ScsiAddress         ScsiAddress
	DeviceInterfaceType uuid.UUID
	DeviceInterfaceName string
	LegacyName          string
	StorageDeviceNumber StorageDeviceNumber
	DeviceInstance      uint32
}

// ScsiAddress maps to the `SCSI_ADDRESS` C++ struct.
// see https://docs.microsoft.com/en-us/windows-hardware/drivers/ddi/content/ntddscsi/ns-ntddscsi-_scsi_address
type ScsiAddress struct {
	PortNumber uint8
	PathID     uint8
	TargetID   uint8
	Lun        uint8
}

// StorageDeviceNumber maps to the `STORAGE_DEVICE_NUMBER` C++ struct.
// see https://docs.microsoft.com/en-us/windows/win32/api/winioctl/ns-winioctl-_storage_device_number
type StorageDeviceNumber struct {
	DeviceType      uint32
	DeviceNumber    uint32
	PartitionNumber uint32
}
