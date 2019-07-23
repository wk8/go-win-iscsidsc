package integrationtests

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	iscsidsc "github.com/wk8/go-win-iscsidsc"
	"github.com/wk8/go-win-iscsidsc/session"
	"gotest.tools/poll"
)

func TestGetIScsiSessionList(t *testing.T) {
	testCases := []struct {
		name      string
		setupFunc func() func()
	}{
		{
			name: "with normal initial buffer size",
		},
		{
			name:      "with a too small initial buffer size",
			setupFunc: setSmallInitialAPIBufferSize,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.setupFunc != nil {
				defer testCase.setupFunc()()
			}

			// let's get the list of sessions before adding ours
			originalSessions, err := session.GetIScsiSessionList()
			require.Nil(t, err)

			// now let's add 3 new targets
			targetIqns, cleanupTargets := setupIscsiTargets(t, 3, "-DiskCount", "1")
			defer cleanupTargets()

			// and let's discover them
			portal, portalCleaner := registerLocalTargetPortal(t)
			defer portalCleaner.cleanup()
			require.NotNil(t, portal)

			// we can now log into them
			for i, targetIqn := range targetIqns {
				sessionID, connectionID, err := logIntoTargetWithDefaultArgs(targetIqn)
				defer assertTargetLoginSuccessful(t, sessionID, connectionID, err)()

				// now that new session should show up in active sessions
				s := findSession(t, sessionID, targetIqn, len(originalSessions)+i+1)

				assertSessionConnectionsEqual(t, s, portal.Address, connectionID)
			}
		})
	}
}

func TestAddConnection(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// let's get the list of sessions before adding ours
		originalSessions, err := session.GetIScsiSessionList()
		require.Nil(t, err)

		// now let's add a target
		targetIqn, cleanupTarget := setupIscsiTarget(t, "-DiskCount", "1")
		defer cleanupTarget()

		// and let's discover it
		portal, portalCleaner := registerLocalTargetPortal(t)
		defer portalCleaner.cleanup()
		require.NotNil(t, portal)

		// and log into it
		sessionID, connectionID, err := logIntoTargetWithDefaultArgs(targetIqn)
		defer assertTargetLoginSuccessful(t, sessionID, connectionID, err)()

		// let's check everything is in order
		s := findSession(t, sessionID, targetIqn, len(originalSessions)+1)
		assertSessionConnectionsEqual(t, s, portal.Address, connectionID)

		// now let's add a connection
		newConnectionID, err := session.AddIScsiConnection(*sessionID, nil, portal, nil, nil, nil)

		// either the call should have succeeded, or we should have got a
		// ISDSC_TOO_MANY_CONNECTIONS (0xEFFF000E) error, depending on the local setup and configuration;
		// even in that latter case, that's proof enough that the call wen through as intended.
		if err == nil {
			require.NotNil(t, newConnectionID)
			s = findSession(t, sessionID, targetIqn, len(originalSessions)+1)
			assertSessionConnectionsEqual(t, s, portal.Address, connectionID, newConnectionID)
		} else {
			require.Nil(t, newConnectionID)
			assertWinAPIErrorCode(t, err, "0xEFFF000E")
		}
	})

	t.Run("when trying to add a connection to a session that's been logged out from, it errors out gracefully", func(t *testing.T) {
		// let's add a target
		targetIqn, cleanupTarget := setupIscsiTarget(t, "-DiskCount", "1")
		defer cleanupTarget()

		// and let's discover it
		portal, portalCleaner := registerLocalTargetPortal(t)
		defer portalCleaner.cleanup()
		require.NotNil(t, portal)

		// and log into it, then log out right after
		sessionID, connectionID, err := logIntoTargetWithDefaultArgs(targetIqn)
		assertTargetLoginSuccessful(t, sessionID, connectionID, err)()

		// now let's try adding a connection, it should yield a
		// ISDSC_INVALID_SESSION_ID (0xEFFF001C) error, see
		// https://docs.microsoft.com/en-us/windows-hardware/drivers/storage/iscsi-status-qualifiers
		newConnectionID, err := session.AddIScsiConnection(*sessionID, nil, portal, nil, nil, nil)
		assert.Nil(t, newConnectionID)
		assertWinAPIErrorCode(t, err, "0xEFFF001C")
	})

	t.Run("it errors out if passed a nil portal", func(t *testing.T) {
		connectionID, err := session.AddIScsiConnection(iscsidsc.SessionID{}, nil, nil, nil, nil, nil)

		assert.Nil(t, connectionID)
		if assert.NotNil(t, err) {
			assert.Equal(t, "targetPortal is required", err.Error())
		}
	})
}

func TestGetDevices(t *testing.T) {
	setups := []struct {
		name      string
		setupFunc func() func()
	}{
		{
			name: "with normal initial buffer size",
		},
		{
			name:      "with a too small initial buffer size",
			setupFunc: setSmallInitialAPIBufferSize,
		},
	}

	for _, setup := range setups {
		for _, diskCount := range []int{1, 2, 5} {
			t.Run(fmt.Sprintf("%s and %d disks", setup.name, diskCount), func(t *testing.T) {
				if setup.setupFunc != nil {
					defer setup.setupFunc()()
				}

				// let's get the list of sessions before adding ours
				originalSessions, err := session.GetIScsiSessionList()
				require.Nil(t, err)

				// now let's add a target, with the desired number of disks
				targetIqn, cleanupTarget := setupIscsiTarget(t, "-DiskCount", strconv.Itoa(diskCount))
				defer cleanupTarget()

				// and let's discover it
				portal, portalCleaner := registerLocalTargetPortal(t)
				defer portalCleaner.cleanup()
				require.NotNil(t, portal)

				// and log into it
				sessionID, connectionID, err := logIntoTargetWithDefaultArgs(targetIqn)
				defer assertTargetLoginSuccessful(t, sessionID, connectionID, err)()

				// let's check everything is in order
				s := findSession(t, sessionID, targetIqn, len(originalSessions)+1)
				assertSessionConnectionsEqual(t, s, portal.Address, connectionID)

				// now let's get the devices for our session; sadly, the Windows API can sometimes
				// take a little longer to actually start reporting the devices...
				var devices []iscsidsc.Device
				poll.WaitOn(t, func(t poll.LogT) poll.Result {
					devices, err = session.GetDevicesForIScsiSession(*sessionID)

					if err != nil {
						return poll.Error(err)
					}
					if len(devices) == diskCount {
						return poll.Success()
					}
					return poll.Continue("waiting to have %d devices, currently %d", diskCount, len(devices))
				})
				require.Nil(t, err)
				require.Equal(t, diskCount, len(devices))

				initiatorName := ""
				for i, device := range devices {
					if i == 0 {
						// this should be the same for all subsequent devices in this same session
						initiatorName = device.InitiatorName
					} else {
						assert.Equal(t, initiatorName, device.InitiatorName)
					}

					assert.Equal(t, targetIqn, device.TargetName)
					assert.Equal(t, i, int(device.ScsiAddress.Lun))
				}
			})
		}
	}

	t.Run("when trying to list devices for a session that's been logged out from, it errors out gracefully", func(t *testing.T) {
		// let's add a target
		targetIqn, cleanupTarget := setupIscsiTarget(t, "-DiskCount", "1")
		defer cleanupTarget()

		// and let's discover it
		portal, portalCleaner := registerLocalTargetPortal(t)
		defer portalCleaner.cleanup()
		require.NotNil(t, portal)

		// and log into it, then log out right after
		sessionID, connectionID, err := logIntoTargetWithDefaultArgs(targetIqn)
		assertTargetLoginSuccessful(t, sessionID, connectionID, err)()

		// now let's try listing the session's devices, it should yield a
		// ISDSC_INVALID_SESSION_ID (0xEFFF001C) error, see
		// https://docs.microsoft.com/en-us/windows-hardware/drivers/storage/iscsi-status-qualifiers
		devices, err := session.GetDevicesForIScsiSession(*sessionID)
		assert.Nil(t, devices)
		assertWinAPIErrorCode(t, err, "0xEFFF001C")
	})
}

// findSession looks amongst existing sessions for the given session,
// checks its target is the one given, and returns its info data.
// expectedSessionsCount is how many sessions are expected to be found in total.
func findSession(t *testing.T, sessionID *iscsidsc.SessionID, targetIqn string, expectedSessionsCount int) *iscsidsc.SessionInfo {
	sessions, err := session.GetIScsiSessionList()
	require.Nil(t, err)
	require.Equal(t, expectedSessionsCount, len(sessions))

	var session *iscsidsc.SessionInfo
	for _, s := range sessions {
		if s.SessionID == *sessionID {
			session = &s
			break
		}
	}
	require.NotNil(t, session)
	require.Equal(t, targetIqn, session.TargetName)

	return session
}

func assertSessionConnectionsEqual(t *testing.T, session *iscsidsc.SessionInfo,
	portalAddress string, connectionIDs ...*iscsidsc.ConnectionID) {

	if assert.Equal(t, len(connectionIDs), len(session.Connections)) {
		for i, connection := range session.Connections {
			assert.Equal(t, *connectionIDs[i], connection.ConnectionID)

			assert.Equal(t, portalAddress, connection.TargetAddress)
		}
	}
}
