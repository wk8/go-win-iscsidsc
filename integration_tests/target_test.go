package integrationtests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wk8/go-win-iscsidsc/target"
)

func TestReportIScsiTargets(t *testing.T) {
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

			_, portalCleaner := registerLocalTargetPortal(t)
			defer portalCleaner.cleanup()

			// let's get the list of targets before adding ours
			originalTargets, err := target.ReportIScsiTargets(true)
			require.Nil(t, err)

			// now let's add a target to our local portal
			targetIqn1, cleanupTarget1 := setupIscsiTarget(t, "-DiskCount", "1")
			defer cleanupTarget1()

			// targets still shouldn't have changed if we don't force a refresh
			newTargets, err := target.ReportIScsiTargets(false)
			require.Nil(t, err)
			assert.Equal(t, originalTargets, newTargets)

			// now let's force a refresh, our new target should show up
			newTargets, err = target.ReportIScsiTargets(true)
			require.Nil(t, err)
			assert.Equal(t, 1, len(newTargets)-len(originalTargets))
			assertStringInSlice(t, targetIqn1, newTargets)

			// for good measure let's create another target
			targetIqn2, cleanupTarget2 := setupIscsiTarget(t, "-DiskCount", "1")
			defer cleanupTarget2()
			// and let's check it shows up too
			newTargets, err = target.ReportIScsiTargets(true)
			require.Nil(t, err)
			assert.Equal(t, 2, len(newTargets)-len(originalTargets))
			assertStringInSlice(t, targetIqn2, newTargets)
		})
	}
}

func TestLoginLogout(t *testing.T) {
	targetIqn, cleanupTarget := setupIscsiTarget(t, "-DiskCount", "1")
	defer cleanupTarget()

	_, portalCleaner := registerLocalTargetPortal(t)
	defer portalCleaner.cleanup()

	// logging in a first time should work
	sessionID1, connectionID1, err := logIntoTargetWithDefaultArgs(targetIqn)
	require.Nil(t, err)
	require.NotNil(t, sessionID1)
	require.NotNil(t, connectionID1)

	// trying to log in a second time should yield a ISDSC_TARGET_ALREADY_LOGGED_IN (0xEFFF003F) error
	// see https://docs.microsoft.com/en-us/windows-hardware/drivers/storage/iscsi-status-qualifiers
	sessionID2, connectionID2, err := logIntoTargetWithDefaultArgs(targetIqn)
	assertWinAPIErrorCode(t, err, "0xEFFF003F")
	assert.Nil(t, sessionID2)
	assert.Nil(t, connectionID2)

	// now let's logout
	err = target.LogoutIScsiTarget(*sessionID1)
	require.Nil(t, err)

	// trying to log out a second time should yield a ISDSC_INVALID_SESSION_ID (0xEFFF001C) error
	err = target.LogoutIScsiTarget(*sessionID1)
	assertWinAPIErrorCode(t, err, "0xEFFF001C")

	// now we should be able to log in again
	sessionID3, connectionID3, err := logIntoTargetWithDefaultArgs(targetIqn)
	require.Nil(t, err)
	require.NotNil(t, sessionID1)
	require.NotNil(t, connectionID1)
	// we should have been given new IDs
	assert.NotEqual(t, sessionID1, sessionID3)
	assert.NotEqual(t, connectionID1, connectionID3)

	// and finally let's log out again
	err = target.LogoutIScsiTarget(*sessionID3)
	require.Nil(t, err)
}
