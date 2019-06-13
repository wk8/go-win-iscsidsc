package integrationtests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wk8/go-win-iscsidsc/target"
)

func TestReportIScsiTargets(t *testing.T) {
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
}

func TestReportIScsiTargetsWithSmallerInitialBuffer(t *testing.T) {
	_, portalCleaner := registerLocalTargetPortal(t)
	defer portalCleaner.cleanup()

	targetIqn, cleanupTarget := setupIscsiTarget(t, "-DiskCount", "1")
	defer cleanupTarget()

	// now we get to the interesting part: we lower the initial buffer size for listing calls
	defer setSmallInitialApiBufferSize()

	// and make that listing call
	targets, err := target.ReportIScsiTargets(true)
	require.Nil(t, err)
	assertStringInSlice(t, targetIqn, targets)
}
