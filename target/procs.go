package target

import (
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/wk8/go-win-iscsidsc/internal"
)

var (
	procReportIScsiTargetsW = internal.IscsidscDLL.NewProc("ReportIScsiTargetsW")
)

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
func retrieveIscsiTargets(forceUpdate bool) (buffer []uint16, err error) {
	bufferSize := internal.InitialApiBufferSize
	var exitCode uintptr

	for {
		if bufferSize%2 == 1 {
			// makes the rest of the function easier to write if bufferSize is always even
			bufferSize++
		}
		buffer = make([]uint16, bufferSize/2)

		exitCode, err = internal.CallWinApi(procReportIScsiTargetsW,
			uintptr(internal.BoolToByte(forceUpdate)),
			uintptr(unsafe.Pointer(&bufferSize)),
			uintptr(unsafe.Pointer(&buffer[0])),
		)

		if exitCode != uintptr(syscall.ERROR_INSUFFICIENT_BUFFER) {
			buffer = buffer[:bufferSize]
			return
		}

		// sanity check: is the new buffer size indeed bigger than the previous one?
		if int(bufferSize) <= 2*len(buffer) {
			// this should never happen
			err = errors.Errorf("Error when calling %q: buffer of size %d deemed too small but bigger than the new advised size of %d", procReportIScsiTargetsW.Name, 2*len(buffer), bufferSize)
			return
		}
	}
}

var invalidIscsiTargetsOutput = errors.Errorf("Error when parsing the response from %q: invalid output", procReportIScsiTargetsW.Name)

// parseIscsiTargets parses the output from retrieveIscsiTargets, which is
// a list of UTF16-encoded, null-terminated strings; and the last string is
// double null-terminated
// Note that in practice there can be any amount of random bytes past the final
// double null character.
func parseIscsiTargets(buffer []uint16) ([]string, error) {
	// there must be at least 2 null bytes, if nothing else
	if len(buffer) < 2 {
		return nil, invalidIscsiTargetsOutput
	}

	targets := make([]string, 0)
	start := 0
	for end, b := range buffer {
		if b == 0 {
			if start == end {
				// means that either the buffer starts with a null byte, or there are 2 null bytes in a row
				// either way, we're done
				if start != 0 || buffer[1] == 0 {
					return targets, nil
				}
				// if we get here, it means that the buffer starts with a null byte, but the next one is not
				// null; shouldn't happen
				return nil, invalidIscsiTargetsOutput
			}

			targets = append(targets, string(utf16.Decode(buffer[start:end])))
			start = end + 1
		}
	}

	// we didn't find a double null byte
	return nil, invalidIscsiTargetsOutput
}
