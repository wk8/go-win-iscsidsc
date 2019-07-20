package integrationtests

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wk8/go-win-iscsidsc"
	"github.com/wk8/go-win-iscsidsc/internal"
	"github.com/wk8/go-win-iscsidsc/targetportal"
)

func TestTargetPortalListCreateDelete(t *testing.T) {
	remainingLocalPortals, existingTargets := getUnregisteredLocalTargetPortals(t)

	internal.IterateOverAllSubsets(uint(len(remainingLocalPortals)), func(indices []uint) {
		shuffle(indices)

		cleanedUp := false
		cleanup := func() (errors []error) {
			if cleanedUp {
				return nil
			}

			for i := 0; i < len(indices); i++ {
				if err := targetportal.RemoveIScsiSendTargetPortal(nil, nil, remainingLocalPortals[indices[i]]); err != nil {
					errors = append(errors, err)
				}
			}

			cleanedUp = true
			return
		}
		defer cleanup()

		// add the portals
		for _, index := range indices {
			portal := remainingLocalPortals[index]
			err := targetportal.AddIScsiSendTargetPortal(nil, nil, nil, nil, portal)
			require.Nil(t, err)
		}

		// check they come up when listing target portals
		newTargets, err := targetportal.ReportIScsiSendTargetPortals()
		if assert.Nil(t, err) {
			assert.Equal(t, len(indices), len(newTargets)-len(existingTargets))
		}

		// and then remove them
		require.False(t, cleanedUp)
		require.Nil(t, cleanup())
	})
}

func TestAddTargetPortalWithLoginOptionsAndSecurityFlags(t *testing.T) {
	remainingLocalPortals, existingTargets := getUnregisteredLocalTargetPortals(t)
	portal := remainingLocalPortals[0]
	portal.SymbolicName = "test-portal-with-login-options-and-security-flags"

	dataDigest := iscsidsc.DigestTypeCRC32C
	defaultTime2Wait := uint32(28)
	loginOptions := &iscsidsc.LoginOptions{
		LoginFlags:       iscsidsc.LoginFlagMultipathEnabled | iscsidsc.LoginFlagUseRadiusVerification,
		DataDigest:       &dataDigest,
		DefaultTime2Wait: &defaultTime2Wait,
	}
	securityFlags := iscsidsc.SecurityFlagIkeIpsecEnabled | iscsidsc.SecurityFlagTransportModePreferred

	portalCleaner := newTargetPortalCleaner(portal)
	defer portalCleaner.cleanup()

	// create the portal
	err := targetportal.AddIScsiSendTargetPortal(nil, nil, loginOptions, &securityFlags, portal)
	require.Nil(t, err)

	// and check it shows up when listing all target portals
	portalInfo := findPortal(t, portal, len(existingTargets)+1)
	require.NotNil(t, portalInfo)

	// check all is as expected
	assert.Equal(t, *loginOptions, portalInfo.LoginOptions)
	assert.Equal(t, securityFlags, portalInfo.SecurityFlags)

	// cleanup
	portalCleaner.assertCleanupSuccessful(t)
}

func TestAddUnresponsiveTargetPortal(t *testing.T) {
	remainingLocalPortals, _ := getUnregisteredLocalTargetPortals(t)
	portal := remainingLocalPortals[0]
	portal.SymbolicName = "test-unresponsive-portal"
	// let's try to connect on the wrong port
	require.NotNil(t, portal.Socket)
	*portal.Socket += 1

	portalCleaner := newTargetPortalCleaner(portal)
	defer portalCleaner.cleanup()

	err := targetportal.AddIScsiSendTargetPortal(nil, nil, nil, nil, portal)
	// see https://docs.microsoft.com/en-us/windows-hardware/drivers/storage/iscsi-status-qualifiers
	// 0xEFFF0003 is a connection failure
	assertWinApiErrorCode(t, err, "0xEFFF0003")

	// cleanup
	portalCleaner.assertCleanupSuccessful(t)
}

func TestAddTargetPortalWithDiscoveryChapAuthentication(t *testing.T) {
	remainingLocalPortals, existingTargets := getUnregisteredLocalTargetPortals(t)
	portal := remainingLocalPortals[0]
	portal.SymbolicName = "test-portal-with-chap-authentication"

	authType := iscsidsc.ChapAuthType
	chapUser := "username"
	chapPassword := "passwordpassword"
	loginOptions := &iscsidsc.LoginOptions{
		AuthType: &authType,
		Username: &chapUser,
		Password: &chapPassword,
	}

	portalCleaner := newTargetPortalCleaner(portal)
	defer portalCleaner.cleanup()

	// creating the portal should result in an error as Windows targets don't support CHAP for discovery (yet?)
	err := targetportal.AddIScsiSendTargetPortal(nil, nil, loginOptions, nil, portal)
	// see https://docs.microsoft.com/en-us/windows-hardware/drivers/storage/iscsi-status-qualifiers
	// 0xEFFF0009 is an authentication failure
	assertWinApiErrorCode(t, err, "0xEFFF0009")

	// it should show up anyway when listing all target portals
	portalInfo := findPortal(t, portal, len(existingTargets)+1)
	require.NotNil(t, portalInfo)

	// sadly, the win API does not return the password...
	loginOptions.Password = nil
	assert.Equal(t, *loginOptions, portalInfo.LoginOptions)

	// cleanup
	portalCleaner.assertCleanupSuccessful(t)
}

// tests that if there are a lot of targets, the mechanism to list
// targets again with a bigger buffer works as intended.
func TestListingTargetsWithSmallerInitialBuffer(t *testing.T) {
	remainingLocalPortals, existingTargets := getUnregisteredLocalTargetPortals(t)

	portalCleaner := newTargetPortalCleaner(remainingLocalPortals...)
	defer portalCleaner.cleanup()

	// create all the target portals
	for _, portal := range remainingLocalPortals {
		err := targetportal.AddIScsiSendTargetPortal(nil, nil, nil, nil, portal)
		require.Nil(t, err)
	}

	// now we get to the interesting part: we lower the initial buffer size for listing calls
	defer setSmallInitialApiBufferSize()()

	// and make that listing call
	allTargets, err := targetportal.ReportIScsiSendTargetPortals()
	require.Nil(t, err)
	assert.Equal(t, len(existingTargets)+len(remainingLocalPortals), len(allTargets))

	// cleanup
	portalCleaner.assertCleanupSuccessful(t)
}

// findPortal looks amongst the registered target portals for the given portal, and returns its info data.
func findPortal(t *testing.T, portal *iscsidsc.Portal, expectedPortalsCount int) *iscsidsc.PortalInfo {
	portals, err := targetportal.ReportIScsiSendTargetPortals()
	if assert.Nil(t, err) {
		assert.Equal(t, expectedPortalsCount, len(portals))

		// find the portal we want in there
		var result *iscsidsc.PortalInfo
		for _, portalInfo := range portals {
			if reflect.DeepEqual(portalInfo.Portal, *portal) {
				result = &portalInfo
				break
			}
		}
		assert.NotNil(t, result)

		return result
	}

	return nil
}
