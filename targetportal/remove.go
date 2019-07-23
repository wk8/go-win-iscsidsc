// This file contains the procs to create, get and remove target portals.

package targetportal

import (
	"unsafe"

	"github.com/pkg/errors"
	iscsidsc "github.com/wk8/go-win-iscsidsc"
	"github.com/wk8/go-win-iscsidsc/internal"
)

var procRemoveIScsiSendTargetPortalW = internal.GetDllProc("RemoveIScsiSendTargetPortalW")

// RemoveIScsiSendTargetPortal removes a portal from the list of portals to which the iSCSI initiator service sends
// SendTargets requests for target discovery.
// Only portal is a required argument.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-removeiscsisendtargetportalw
func RemoveIScsiSendTargetPortal(initiatorInstance *string, initiatorPortNumber *uint32, portal *iscsidsc.Portal) error {
	initiatorInstancePtr, initiatorPortNumberValue, err := internal.ConvertInitiatorArgs(initiatorInstance, initiatorPortNumber)
	if err != nil {
		return err
	}

	if portal == nil {
		return errors.Errorf("portal is required")
	}
	internalPortal, err := internal.CheckAndConvertPortal(portal)
	if err != nil {
		return errors.Wrap(err, "invalid portal argument")
	}

	_, err = internal.CallWinAPI(procRemoveIScsiSendTargetPortalW,
		uintptr(unsafe.Pointer(initiatorInstancePtr)),
		uintptr(initiatorPortNumberValue),
		uintptr(unsafe.Pointer(internalPortal)),
	)

	return err
}
