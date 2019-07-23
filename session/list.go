package session

import (
	"fmt"
	"unsafe"

	"github.com/pkg/errors"
	iscsidsc "github.com/wk8/go-win-iscsidsc"
	"github.com/wk8/go-win-iscsidsc/internal"
)

var procGetIScsiSessionListW = internal.GetDllProc("GetIScsiSessionListW")

// GetIScsiSessionList retrieves the list of active iSCSI sessions.
// see https://docs.microsoft.com/en-us/windows/desktop/api/iscsidsc/nf-iscsidsc-getiscsisessionlistw
func GetIScsiSessionList() ([]iscsidsc.SessionInfo, error) {
	buffer, bufferPointer, count, err := retrieveSessionInfos()
	if err != nil {
		return nil, err
	}

	// this really baffles me, but it seems that on some Windows versions GetIScsiSessionListW returns
	// a buffer size that's actually quite bigger than the space it actually uses... so here we can't check
	// that we've used all of the declared buffer size, sadly.
	sessionInfos, _, err := hydrateSessionInfos(buffer, bufferPointer, int(count))

	return sessionInfos, nil
}

// retrieveSessionInfos gets the raw session infos from the Windows API.
func retrieveSessionInfos() (buffer []byte, bufferPointer uintptr, count int32, err error) {
	return internal.HandleBufferedWinAPICall(
		func(s, c, b uintptr) (uintptr, error) {
			return internal.CallWinAPI(procGetIScsiSessionListW, s, c, b)
		},
		procGetIScsiSessionListW.Name,
		1,
	)
}

// hydrateSessionInfos takes the raw bytes returned by the `GetIScsiSessionListW` C++ proc,
// and casts the raw data into Go structs.
// Also returns the total number of bytes it's read from the buffer.
func hydrateSessionInfos(buffer []byte, bufferPointer uintptr, count int) ([]iscsidsc.SessionInfo, uintptr, error) {
	// sanity check: the total size should be at least enough to contain the session infos
	minimumExpectedSize := count * int(internal.SessionInfoSize)
	if len(buffer) < minimumExpectedSize {
		return nil, 0, hydrateSessionError("expected the reply to be at least %d bytes, only got %d bytes", minimumExpectedSize, len(buffer))
	}

	sessions := make([]iscsidsc.SessionInfo, count)
	var bytesRead uintptr
	for i := 0; i < count; i++ {
		read, err := hydrateSessionInfo(buffer, bufferPointer, i, &sessions[i])
		bytesRead += read
		if err != nil {
			return nil, bytesRead, err
		}
	}

	return sessions, bytesRead, nil
}

// hydrateSessionInfo hydrates a single `SessionInfo` struct.
// It returns the number of bytes it's read from the buffer.
func hydrateSessionInfo(buffer []byte, bufferPointer uintptr, i int, info *iscsidsc.SessionInfo) (uintptr, error) {
	// we already know that we're still in the buffer here - we check that at the very start of `hydrateSessionInfos`,
	// so this is safe as per rule (1) of https://golang.org/pkg/unsafe/#Pointer
	infoIn := (*internal.SessionInfo)(unsafe.Pointer(&buffer[uintptr(i)*internal.SessionInfoSize]))
	bytesRead := internal.SessionInfoSize

	info.SessionID = infoIn.SessionID

	initiatorName, read, err := internal.ExtractWideStringFromBuffer(buffer, bufferPointer, infoIn.InitiatorName)
	bytesRead += read
	if err != nil {
		return bytesRead, hydrateSessionError(" could not read session initiator name: %v", err)
	}
	info.InitiatorName = initiatorName

	targetNodeName, read, err := internal.ExtractWideStringFromBuffer(buffer, bufferPointer, infoIn.TargetNodeName)
	bytesRead += read
	if err != nil {
		return bytesRead, hydrateSessionError(" could not read session target node name: %v", err)
	}
	info.TargetNodeName = targetNodeName

	targetName, read, err := internal.ExtractWideStringFromBuffer(buffer, bufferPointer, infoIn.TargetName)
	bytesRead += read
	if err != nil {
		return bytesRead, hydrateSessionError(" could not read session target name: %v", err)
	}
	info.TargetName = targetName

	info.ISID = infoIn.ISID
	info.TSID = infoIn.TSID

	if infoIn.ConnectionCount > 0 && infoIn.Connections != 0 {
		connectionsOffset := infoIn.Connections - bufferPointer

		// sanity check: this should still be inside the buffer
		if connectionsOffset < 0 || connectionsOffset >= uintptr(len(buffer)) {
			return bytesRead, hydrateSessionError("connections pointer pointing out of the buffer")
		}

		connections, read, err := hydrateConnectionInfos(
			buffer,
			bufferPointer,
			connectionsOffset,
			int(infoIn.ConnectionCount),
		)
		bytesRead += read
		if err != nil {
			return bytesRead, err
		}
		info.Connections = connections
	}

	return bytesRead, nil
}

func hydrateConnectionInfos(buffer []byte, bufferPointer, connectionsOffset uintptr, connectionCount int) ([]iscsidsc.ConnectionInfo, uintptr, error) {
	// sanity check: the total size should be at least enough to contain the connection infos
	minimumExpectedSize := connectionCount * int(internal.ConnectionInfoSize)
	if len(buffer)-int(connectionsOffset) < minimumExpectedSize {
		return nil, 0, hydrateSessionError("expected the buffer for connections to be at least %d bytes, only got %d bytes", minimumExpectedSize, len(buffer)-int(connectionsOffset))
	}

	connections := make([]iscsidsc.ConnectionInfo, connectionCount)
	var bytesRead uintptr
	for i := 0; i < connectionCount; i++ {
		read, err := hydrateConnectionInfo(buffer, bufferPointer, connectionsOffset, i, &connections[i])
		bytesRead += read
		if err != nil {
			return nil, bytesRead, err
		}
	}

	return connections, bytesRead, nil
}

func hydrateConnectionInfo(buffer []byte, bufferPointer, connectionsOffset uintptr, i int, info *iscsidsc.ConnectionInfo) (uintptr, error) {
	infoIn := (*internal.ConnectionInfo)(unsafe.Pointer(&buffer[connectionsOffset+uintptr(i)*internal.ConnectionInfoSize]))
	bytesRead := internal.ConnectionInfoSize

	info.ConnectionID = infoIn.ConnectionID

	initiatorAddress, read, err := internal.ExtractWideStringFromBuffer(buffer, bufferPointer, infoIn.InitiatorAddress)
	bytesRead += read
	if err != nil {
		return bytesRead, hydrateSessionError(" could not read connection initiator address: %v", err)
	}
	info.InitiatorAddress = initiatorAddress

	targetAddress, read, err := internal.ExtractWideStringFromBuffer(buffer, bufferPointer, infoIn.TargetAddress)
	bytesRead += read
	if err != nil {
		return bytesRead, hydrateSessionError(" could not read connection target address: %v", err)
	}
	info.TargetAddress = targetAddress

	info.InitiatorSocket = infoIn.InitiatorSocket
	info.TargetSocket = infoIn.TargetSocket
	info.CID = infoIn.CID

	return bytesRead, nil
}

func hydrateSessionError(format string, args ...interface{}) error {
	msg := fmt.Sprintf("Error when hydrating the response from %s - it might be that your Windows version is not supported: ", procGetIScsiSessionListW.Name)
	return errors.Errorf(msg+format, args...)
}
