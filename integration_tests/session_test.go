package integrationtests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wk8/go-win-iscsidsc"
	"github.com/wk8/go-win-iscsidsc/session"
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
			setupFunc: setSmallInitialApiBufferSize,
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
				sessionId, connectionId, err := logIntoTargetWithDefaultArgs(targetIqn)
				defer assertTargetLoginSuccessful(t, sessionId, connectionId, err)()

				// now that new session should show up in active sessions
				s := findSession(t, sessionId, targetIqn, len(originalSessions)+i+1)

				assertSessionConnectionsEqual(t, s, portal.Address, connectionId)
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
		sessionId, connectionId, err := logIntoTargetWithDefaultArgs(targetIqn)
		defer assertTargetLoginSuccessful(t, sessionId, connectionId, err)()

		// let's check everything is in order
		s := findSession(t, sessionId, targetIqn, len(originalSessions)+1)
		assertSessionConnectionsEqual(t, s, portal.Address, connectionId)

		// now let's add a connection
		newConnectionId, err := session.AddIScsiConnectionW(*sessionId, nil, portal, nil, nil, nil)

		// either the call should have succeeded, or we should have got a
		// ISDSC_TOO_MANY_CONNECTIONS (0xEFFF000E) error, depending on the local setup and configuration;
		// even in that latter case, that's proof enough that the call wen through as intended.
		if err == nil {
			require.NotNil(t, newConnectionId)
			s = findSession(t, sessionId, targetIqn, len(originalSessions)+1)
			assertSessionConnectionsEqual(t, s, portal.Address, connectionId, newConnectionId)
		} else {
			require.Nil(t, newConnectionId)
			assertWinApiErrorCode(t, err, "0xEFFF000E")
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
		sessionId, connectionId, err := logIntoTargetWithDefaultArgs(targetIqn)
		assertTargetLoginSuccessful(t, sessionId, connectionId, err)()

		// now let's try adding a connection, it should yield a
		// ISDSC_INVALID_SESSION_ID (0xEFFF001C) error, see
		// https://docs.microsoft.com/en-us/windows-hardware/drivers/storage/iscsi-status-qualifiers
		newConnectionId, err := session.AddIScsiConnectionW(*sessionId, nil, portal, nil, nil, nil)
		assert.Nil(t, newConnectionId)
		assertWinApiErrorCode(t, err, "0xEFFF001C")
	})

	t.Run("it errors out if passed a nil portal", func(t *testing.T) {
		connectionId, err := session.AddIScsiConnectionW(iscsidsc.SessionId{}, nil, nil, nil, nil, nil)

		assert.Nil(t, connectionId)
		if assert.NotNil(t, err) {
			assert.Equal(t, "targetPortal is required", err.Error())
		}
	})
}

// findSession looks amongst existing sessions for the given session,
// checks its target is the one given, and returns its info data.
// expectedSessionsCount is how many sessions are expected to be found in total.
func findSession(t *testing.T, sessionId *iscsidsc.SessionId, targetIqn string, expectedSessionsCount int) *iscsidsc.SessionInfo {
	sessions, err := session.GetIScsiSessionList()
	require.Nil(t, err)
	require.Equal(t, expectedSessionsCount, len(sessions))

	var session *iscsidsc.SessionInfo
	for _, s := range sessions {
		if s.SessionId == *sessionId {
			session = &s
			break
		}
	}
	require.NotNil(t, session)
	require.Equal(t, targetIqn, session.TargetName)

	return session
}

func assertSessionConnectionsEqual(t *testing.T, session *iscsidsc.SessionInfo,
	portalAddress string, connectionIds ...*iscsidsc.ConnectionId) {

	if assert.Equal(t, len(connectionIds), len(session.Connections)) {
		for i, connection := range session.Connections {
			assert.Equal(t, *connectionIds[i], connection.ConnectionId)

			assert.Equal(t, portalAddress, connection.TargetAddress)
		}
	}
}
