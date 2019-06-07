package integrationtests

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wk8/go-win-iscsidsc"
	"github.com/wk8/go-win-iscsidsc/targetportal"
)

func TestTargetPortalListCreateDelete(t *testing.T) {
	remainingLocalPortals, existingTargets := getAvailableLocalTargetPortals(t)

	iterateOverAllSubsets(uint(len(remainingLocalPortals)), func(indices []uint) {
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

// used to clean up the target portals added for discovery targets
type targetPortalCleaner struct {
	portals []*targetportal.Portal
	ran     bool
}

func newTargetPortalCleaner(portals ...*targetportal.Portal) *targetPortalCleaner {
	return &targetPortalCleaner{
		portals: portals,
	}
}

func (cleaner *targetPortalCleaner) cleanup() (errors []error) {
	if cleaner.ran {
		return
	}

	for _, portal := range cleaner.portals {
		if err := targetportal.RemoveIScsiSendTargetPortal(nil, nil, portal); err != nil {
			errors = append(errors, err)
		}
	}

	cleaner.ran = true
	return
}

func (cleaner *targetPortalCleaner) assertCleanupSuccessful(t *testing.T) {
	require.False(t, cleaner.ran)
	require.Nil(t, cleaner.cleanup())
}

func TestAddTargetPortalWithLoginOptionsAndSecurityFlags(t *testing.T) {
	remainingLocalPortals, existingTargets := getAvailableLocalTargetPortals(t)
	portal := remainingLocalPortals[0]
	portal.SymbolicName = "test-portal-with-login-options-and-security-flags"

	dataDigest := targetportal.DigestTypeCRC32C
	defaultTime2Wait := uint32(28)
	loginOptions := &targetportal.LoginOptions{
		LoginFlags:       targetportal.LoginFlagMultipathEnabled | targetportal.LoginFlagUseRadiusVerification,
		DataDigest:       &dataDigest,
		DefaultTime2Wait: &defaultTime2Wait,
	}
	securityFlags := targetportal.SecurityFlagIkeIpsecEnabled | targetportal.SecurityFlagTransportModePreferred

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
	remainingLocalPortals, _ := getAvailableLocalTargetPortals(t)
	portal := remainingLocalPortals[0]
	portal.SymbolicName = "test-unresponsive-portal"
	// let's try to connect on the wrong port
	require.NotNil(t, portal.Socket)
	*portal.Socket += 1

	portalCleaner := newTargetPortalCleaner(portal)
	defer portalCleaner.cleanup()

	err := targetportal.AddIScsiSendTargetPortal(nil, nil, nil, nil, portal)
	require.NotNil(t, err)
	winApiErr, ok := err.(*iscsidsc.WinApiCallError)
	require.True(t, ok)
	// see https://docs.microsoft.com/en-us/windows-hardware/drivers/storage/iscsi-status-qualifiers
	// 0xEFFF0003 is a connection failure
	require.Equal(t, "0xEFFF0003", winApiErr.HexCode())

	// cleanup
	portalCleaner.assertCleanupSuccessful(t)
}

func TestAddTargetPortalWithDiscoveryChapAuthentication(t *testing.T) {
	remainingLocalPortals, existingTargets := getAvailableLocalTargetPortals(t)
	portal := remainingLocalPortals[0]
	portal.SymbolicName = "test-portal-with-chap-authentication"

	authType := targetportal.ChapAuthType
	chapUser := "username"
	chapPassword := "passwordpassword"
	loginOptions := &targetportal.LoginOptions{
		AuthType: &authType,
		Username: &chapUser,
		Password: &chapPassword,
	}

	portalCleaner := newTargetPortalCleaner(portal)
	defer portalCleaner.cleanup()

	// creating the portal should result in an error as Windows targets don't support CHAP for discovery (yet?)
	err := targetportal.AddIScsiSendTargetPortal(nil, nil, loginOptions, nil, portal)
	require.NotNil(t, err)
	winApiErr, ok := err.(*iscsidsc.WinApiCallError)
	require.True(t, ok)
	// see https://docs.microsoft.com/en-us/windows-hardware/drivers/storage/iscsi-status-qualifiers
	// 0xEFFF0009 is an authentication failure
	require.Equal(t, "0xEFFF0009", winApiErr.HexCode())

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
	remainingLocalPortals, existingTargets := getAvailableLocalTargetPortals(t)

	portalCleaner := newTargetPortalCleaner(remainingLocalPortals...)
	defer portalCleaner.cleanup()

	// create all the target portals
	for _, portal := range remainingLocalPortals {
		err := targetportal.AddIScsiSendTargetPortal(nil, nil, nil, nil, portal)
		require.Nil(t, err)
	}

	// now we get to the interesting part: we lower the initial buffer size for listing calls
	previousBufferSize := iscsidsc.InitialApiBufferSize
	iscsidsc.InitialApiBufferSize = 1
	defer func() {
		iscsidsc.InitialApiBufferSize = previousBufferSize
	}()

	// and make that listing call
	allTargets, err := targetportal.ReportIScsiSendTargetPortals()
	require.Nil(t, err)
	assert.Equal(t, len(existingTargets)+len(remainingLocalPortals), len(allTargets))

	// cleanup
	portalCleaner.assertCleanupSuccessful(t)
}

// getAvailableLocalTargetPortals looks for at least one local target that has not been added
// as a discovery target yet, and otherwise fails the test: we don't want to make potentially
// destructive changes to the system.
// It returns a list of available local target portals, as well as the other existing targets
// that have already been added as discovery targets.
func getAvailableLocalTargetPortals(t *testing.T) ([]*targetportal.Portal, []targetportal.PortalInfo) {
	localPortals := getLocalTargetPortals(t)

	existingTargets, err := targetportal.ReportIScsiSendTargetPortals()
	require.Nil(t, err)

	// look for at least one local target that has not been added as a discovery target yet
	for _, target := range existingTargets {
		delete(localPortals, target.Address)
	}
	if len(localPortals) == 0 {
		t.Fatalf("All local targets have already been added, cowardly refusing to run this test")
	}

	remainingLocalPortals := make([]*targetportal.Portal, len(localPortals))
	i := 0
	for _, portal := range localPortals {
		remainingLocalPortals[i] = portal
		i++
	}

	return remainingLocalPortals, existingTargets
}

// findPortal looks amongst the registered target portals for the given portal, and returns its info data.
func findPortal(t *testing.T, portal *targetportal.Portal, expectedPortalsCount int) *targetportal.PortalInfo {
	portals, err := targetportal.ReportIScsiSendTargetPortals()
	if assert.Nil(t, err) {
		assert.Equal(t, expectedPortalsCount, len(portals))

		// find the portal we want in there
		var result *targetportal.PortalInfo
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
