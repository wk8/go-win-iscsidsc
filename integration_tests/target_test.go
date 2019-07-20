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
			setupFunc: setSmallInitialApiBufferSize,
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
	sessionId1, connectionId1, err := logIntoTargetWithDefaultArgs(targetIqn)
	require.Nil(t, err)
	require.NotNil(t, sessionId1)
	require.NotNil(t, connectionId1)

	// trying to log in a second time should yield a ISDSC_TARGET_ALREADY_LOGGED_IN (0xEFFF003F) error
	// see https://docs.microsoft.com/en-us/windows-hardware/drivers/storage/iscsi-status-qualifiers
	sessionId2, connectionId2, err := logIntoTargetWithDefaultArgs(targetIqn)
	assertWinApiErrorCode(t, err, "0xEFFF003F")
	assert.Nil(t, sessionId2)
	assert.Nil(t, connectionId2)

	// now let's logout
	err = target.LogoutIScsiTarget(*sessionId1)
	require.Nil(t, err)

	// trying to log out a second time should yield a ISDSC_INVALID_SESSION_ID (0xEFFF001C) error
	err = target.LogoutIScsiTarget(*sessionId1)
	assertWinApiErrorCode(t, err, "0xEFFF001C")

	// now we should be able to log in again
	sessionId3, connectionId3, err := logIntoTargetWithDefaultArgs(targetIqn)
	require.Nil(t, err)
	require.NotNil(t, sessionId1)
	require.NotNil(t, connectionId1)
	// we should have been given new IDs
	assert.NotEqual(t, sessionId1, sessionId3)
	assert.NotEqual(t, connectionId1, connectionId3)

	// and finally let's log out again
	err = target.LogoutIScsiTarget(*sessionId3)
	require.Nil(t, err)
}
