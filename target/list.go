package target

import (
	"github.com/pkg/errors"
	"github.com/wk8/go-win-iscsidsc/internal"
)

var procReportIScsiTargetsW = internal.GetDllProc("ReportIScsiTargetsW")

// ReportIScsiTargets retrieves the list of targets that the iSCSI initiator service has discovered.
// if forceUpdate is true,  the iSCSI initiator service updates the list of discovered targets before
// returning the target list data to the caller.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-reportiscsitargetsw
func ReportIScsiTargets(forceUpdate bool) ([]string, error) {
	buffer, err := retrieveIscsiTargets(forceUpdate)
	if err != nil {
		return nil, err
	}

	return parseIscsiTargets(buffer)
}

// retrieveIscsiTargets gets the raw target list from the Windows API.
func retrieveIscsiTargets(forceUpdate bool) (buffer []byte, err error) {
	buffer, _, _, err = internal.HandleBufferedWinAPICall(
		func(s, _, b uintptr) (uintptr, error) {
			return internal.CallWinAPI(procReportIScsiTargetsW,
				uintptr(internal.BoolToByte(forceUpdate)),
				s,
				b)
		},
		procReportIScsiTargetsW.Name,
		2,
	)
	return
}

var invalidIscsiTargetsOutput = errors.Errorf("Error when parsing the response from %q: invalid output", procReportIScsiTargetsW.Name)

// parseIscsiTargets parses the output from retrieveIscsiTargets, which is
// a list of UTF16-encoded, null-terminated strings; and the last string is
// double null-terminated
// Note that in practice there can be any amount of random bytes past the final
// double null character.
func parseIscsiTargets(buffer []byte) ([]string, error) {
	// no matter what, this buffer can't be shorter than 4 bytes (2 null wide chars)
	if len(buffer) < 4 {
		return nil, invalidIscsiTargetsOutput
	}

	targets := make([]string, 0)
	offset := uintptr(0)

	for {
		target, read, err := internal.ExtractWideStringFromBuffer(buffer, 0, offset)
		if err != nil {
			return nil, invalidIscsiTargetsOutput
		}
		if target == "" {
			if offset != 0 || (buffer[2] == 0 && buffer[3] == 0) {
				// we've found the double null char, we're done
				return targets, nil
			}
			// the buffer started with a null char, but the next char wasn't a null char
			return nil, invalidIscsiTargetsOutput
		}
		targets = append(targets, target)
		offset += read
	}
}
