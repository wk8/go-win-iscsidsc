package session

import (
	"fmt"
	"unsafe"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	iscsidsc "github.com/wk8/go-win-iscsidsc"
	"github.com/wk8/go-win-iscsidsc/internal"
	"golang.org/x/sys/windows"
)

var procGetDevicesForIScsiSessionW = internal.GetDllProc("GetDevicesForIScsiSessionW")

// GetDevicesForIScsiSession retrieves information about the devices associated with an existing session.
// see https://docs.microsoft.com/en-us/windows/win32/api/iscsidsc/nf-iscsidsc-getdevicesforiscsisessionw
func GetDevicesForIScsiSession(id iscsidsc.SessionID) ([]iscsidsc.Device, error) {
	buffer, bufferPointer, _, err := retrieveDevices(id)
	if err != nil {
		return nil, err
	}

	if len(buffer)%int(internal.DeviceSize) != 0 {
		return nil, hydrateDevicesError("expected reply size to be a multiple of %d, actual size %d",
			internal.DeviceSize, len(buffer))
	}
	count := len(buffer) / int(internal.DeviceSize)

	return hydrateDevices(buffer, bufferPointer, count)
}

// retrieveDevices gets the raw devices' infos from the Windows API.
func retrieveDevices(id iscsidsc.SessionID) (buffer []byte, bufferPointer uintptr, count int32, err error) {
	return internal.HandleBufferedWinAPICall(
		func(s, _, b uintptr) (uintptr, error) {
			return internal.CallWinAPI(procGetDevicesForIScsiSessionW,
				uintptr(unsafe.Pointer(&id)),
				s,
				b)
		},
		procGetDevicesForIScsiSessionW.Name,
		internal.DeviceSize,
	)
}

// hydrateDevices takes the raw bytes returned by the `GetDevicesForIScsiSessionW` C++ proc,
// and casts the raw data into Go structs.
func hydrateDevices(buffer []byte, bufferPointer uintptr, count int) ([]iscsidsc.Device, error) {
	devices := make([]iscsidsc.Device, count)
	for i := 0; i < count; i++ {
		if err := hydrateDevice(buffer, bufferPointer, i, &devices[i]); err != nil {
			return nil, err
		}
	}

	return devices, nil
}

// hydrateDevice hydrates a single `Device` struct.
func hydrateDevice(buffer []byte, bufferPointer uintptr, i int, device *iscsidsc.Device) error {
	// we already know that we're still in the buffer here - we check that at the very start of `hydrateDevices`,
	// so this is safe as per rule (1) of https://golang.org/pkg/unsafe/#Pointer
	deviceIn := (*internal.Device)(unsafe.Pointer(&buffer[uintptr(i)*internal.DeviceSize]))

	device.InitiatorName = windows.UTF16ToString(deviceIn.InitiatorName[:])
	device.TargetName = windows.UTF16ToString(deviceIn.TargetName[:])

	scsiAddress, err := hydrateScsiAddress(deviceIn.ScsiAddress)
	if err != nil {
		return err
	}
	device.ScsiAddress = *scsiAddress

	guid, err := hydrateGUID(deviceIn.DeviceInterfaceType)
	if err != nil {
		return err
	}
	device.DeviceInterfaceType = guid

	device.DeviceInterfaceName = windows.UTF16ToString(deviceIn.DeviceInterfaceName[:])
	device.LegacyName = windows.UTF16ToString(deviceIn.LegacyName[:])
	device.StorageDeviceNumber = deviceIn.StorageDeviceNumber
	device.DeviceInstance = deviceIn.DeviceInstance

	return nil
}

func hydrateScsiAddress(addressIn internal.ScsiAddress) (*iscsidsc.ScsiAddress, error) {
	if addressIn.Length != 8 {
		return nil, hydrateDevicesError("Unexpected SCSI address length: %d", addressIn.Length)
	}

	return &iscsidsc.ScsiAddress{
		PortNumber: addressIn.PortNumber,
		PathID:     addressIn.PathID,
		TargetID:   addressIn.TargetID,
		Lun:        addressIn.Lun,
	}, nil
}

// hydrateGUID casts a GUID as stored internally by Windows' API into
// a more easily usable struct.
func hydrateGUID(guidIn internal.GUID) (uuid.UUID, error) {
	b := make([]byte, 16)

	b1 := (*[4]byte)(unsafe.Pointer(&guidIn.Data1))
	copy(b, b1[:])

	b2 := (*[2]byte)(unsafe.Pointer(&guidIn.Data2))
	copy(b[4:], b2[:])

	b3 := (*[2]byte)(unsafe.Pointer(&guidIn.Data3))
	copy(b[6:], b3[:])

	copy(b[8:], guidIn.Data4[:])

	guid, err := uuid.FromBytes(b)

	if err != nil {
		return guid, hydrateDevicesError("error when parsing GUID: %v", err)
	}
	return guid, nil
}

func hydrateDevicesError(format string, args ...interface{}) error {
	msg := fmt.Sprintf("Error when hydrating the response from %s - it might be that your Windows version is not supported: ", procGetDevicesForIScsiSessionW.Name)
	return errors.Errorf(msg+format, args...)
}
