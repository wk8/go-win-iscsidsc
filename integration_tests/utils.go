package integrationtests

import (
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	iscsidsc "github.com/wk8/go-win-iscsidsc"
	"github.com/wk8/go-win-iscsidsc/internal"
	"github.com/wk8/go-win-iscsidsc/target"
	"github.com/wk8/go-win-iscsidsc/targetportal"
)

var targetPotalEndpointRegex = regexp.MustCompile("^((?:[0-9]{1,3}\\.){3}(?:[0-9]{1,3})):([1-9][0-9]*)$")

// getLocalTargetPortals returns a map of the target portals currently listening locally,
// mapping the IP they listen on to the portal structure.
// Fails the test immediately if there are none, or if the WinTarget service is not running.
func getLocalTargetPortals(t *testing.T) map[string]*iscsidsc.Portal {
	powershellCommand := "(Get-IscsiTargetServerSetting).Portals | Where Enabled | ForEach {$_.IPEndpoint.ToString()}"
	output, err := exec.Command("powershell", "/c", powershellCommand).Output()
	require.Nil(t, err, "Unable to list local target portals: %v", err)

	lines := strings.Split(string(output), "\n")
	portals := make(map[string]*iscsidsc.Portal)
	for _, line := range lines {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		match := targetPotalEndpointRegex.FindStringSubmatch(line)
		require.NotNil(t, match, "Unexpected line when listing local target portals: %q", line)

		port, err := strconv.ParseUint(match[2], 10, 16)
		require.Nil(t, err, "Unable to convert %q to a uint16: %v", match[2], err)
		portUint16 := uint16(port)

		portals[match[1]] = &iscsidsc.Portal{
			Address: match[1],
			Socket:  &portUint16,
		}
	}

	return portals
}

// getUnregisteredLocalTargetPortals looks for at least one local target that has not been added
// as a discovery target yet, and otherwise fails the test: we don't want to make potentially
// destructive changes to the system.
// It returns a list of available local target portals, as well as the other existing targets
// that have already been added as discovery targets.
func getUnregisteredLocalTargetPortals(t *testing.T) ([]*iscsidsc.Portal, []iscsidsc.PortalInfo) {
	localPortals := getLocalTargetPortals(t)

	existingTargets, err := targetportal.ReportIScsiSendTargetPortals()
	require.Nil(t, err)

	// look for at least one local target that has not been added as a discovery target yet
	for _, target := range existingTargets {
		delete(localPortals, target.Address)
	}
	require.NotEqual(t, 0, len(localPortals), "All local targets have already been added, cowardly refusing to run this test")

	remainingLocalPortals := make([]*iscsidsc.Portal, len(localPortals))
	i := 0
	for _, portal := range localPortals {
		remainingLocalPortals[i] = portal
		i++
	}

	return remainingLocalPortals, existingTargets
}

// used to clean up the target portals added for discovery targets
type targetPortalCleaner struct {
	portals []*iscsidsc.Portal
	ran     bool
}

func newTargetPortalCleaner(portals ...*iscsidsc.Portal) *targetPortalCleaner {
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

// registerLocalTargetPortal gets the first unregistered portal from getUnregisteredLocalTargetPortals,
// registers it with the default options, and returns it along with a targetPortalCleaner to unregister it
// when done with testing
func registerLocalTargetPortal(t *testing.T) (*iscsidsc.Portal, *targetPortalCleaner) {
	portals, _ := getUnregisteredLocalTargetPortals(t)
	portal := portals[0]

	err := targetportal.AddIScsiSendTargetPortal(nil, nil, nil, nil, portal)
	require.Nil(t, err)

	return portal, newTargetPortalCleaner(portal)
}

// setupIscsiTarget calls hack/setup_iscsi_target to create a target with a random IQN,
// then returns both the target's IQN as well as a function to tear it down when done with testing.
func setupIscsiTarget(t *testing.T, extraArgs ...string) (string, func()) {
	iqnFile, err := ioutil.TempFile("", "")
	require.Nil(t, err, "Error when creating temp file: %v", err)
	defer func() {
		require.Nil(t, os.Remove(iqnFile.Name()))
	}()
	// need to close it so that the powershell script can write to it
	require.Nil(t, iqnFile.Close())

	args := []string{
		"-TestIQN",
		"-WriteIQNTo", iqnFile.Name(),
		"-OverwriteIQNFile",
	}

	runHackPowershellScript(t, "setup_iscsi_target", append(args, extraArgs...)...)

	iqn := strings.TrimSpace(readFile(t, iqnFile.Name()))
	require.NotEqual(t, 0, len(iqn))

	return iqn, func() {
		runHackPowershellScript(t, "teardown_iscsi_targets", iqn)
	}
}

// setupIscsiTargets sets up count ISCSI targets, and then returns their IQNs as well as a function
// to tear them down when done with testing.
func setupIscsiTargets(t *testing.T, count int, extraArgs ...string) ([]string, func()) {
	iqns := make([]string, count)
	cleanups := make([]func(), count)

	for i := 0; i < count; i++ {
		iqn, cleanup := setupIscsiTarget(t, extraArgs...)
		iqns[i] = iqn
		cleanups[i] = cleanup
	}

	return iqns, func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}
}

// when running scripts from the hack/ directory, runHackPowershellScript below will try that many times
// before aborting - iSCSI commands can sometimes error out for no apparent reasons, especially on small
// (eg CI) boxes
const maxHackPowershellScriptTries = 3

// runHackPowershellScript calls a script from the hack/ directory
func runHackPowershellScript(t *testing.T, scriptName string, args ...string) {
	scriptPath := path.Join(repoRoot(t), "hack", scriptName+".ps1")

	var output []byte
	var err error
	for i := 0; i < maxHackPowershellScriptTries; i++ {
		output, err = exec.Command("powershell", append([]string{scriptPath}, args...)...).CombinedOutput()
		if err == nil {
			return
		}
	}

	require.FailNow(t, "runHackPowershellScript failed", "Failed %d times to run script %q: %v, and output: %s", maxHackPowershellScriptTries, scriptPath, err, string(output))
}

func repoRoot(t *testing.T) string {
	_, filePath, _, ok := runtime.Caller(0)
	require.True(t, ok, "Unable to resolve path to repo root")
	return filepath.Dir(filepath.Dir(filePath))
}

func shuffle(a []uint) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(a) - 1; i > 0; i-- {
		j := r.Int() % (i + 1)
		tmp := a[j]
		a[j] = a[i]
		a[i] = tmp
	}
}

func assertStringInSlice(t *testing.T, needle string, slice []string) {
	for _, item := range slice {
		if item == needle {
			return
		}
	}
	require.Fail(t, "assertStringInSlice failed", "%q not found in %v", needle, slice)
}

// setSmallInitialAPIBufferSize changes the value of internal.InitialAPIBufferSize to 1,
// and returns a func to revert that change when done with testing.
func setSmallInitialAPIBufferSize() func() {
	previousBufferSize := internal.InitialAPIBufferSize
	internal.InitialAPIBufferSize = 1
	return func() {
		internal.InitialAPIBufferSize = previousBufferSize
	}
}

func logIntoTargetWithDefaultArgs(targetIqn string) (*iscsidsc.SessionID, *iscsidsc.ConnectionID, error) {
	return target.LoginIscsiTarget(targetIqn, false, nil, nil, nil, nil, nil, nil, false)
}

// assertTargetLoginSuccessful should be called with return values from target.LoginIscsiTarget.
// It asserts the login has been successful, and returns a cleanup function to be called when
// done with testing to log out from the target.
func assertTargetLoginSuccessful(t *testing.T, sessionID *iscsidsc.SessionID, connectionID *iscsidsc.ConnectionID, err error) func() {
	require.Nil(t, err)
	require.NotNil(t, sessionID)
	require.NotNil(t, connectionID)

	return func() {
		assert.Nil(t, target.LogoutIScsiTarget(*sessionID))
	}
}

func readFile(t *testing.T, path string) string {
	file, err := os.Open(path)
	require.Nil(t, err, "Unable to open %q: %v", path, err)
	defer func() {
		require.Nil(t, file.Close())
	}()

	contents, err := ioutil.ReadAll(file)
	require.Nil(t, err, "Unable to read %q: %v", path, err)

	return string(contents)
}

func assertWinAPIErrorCode(t *testing.T, err error, expectedErrorCode string) bool {
	if !assert.NotNil(t, err) {
		return false
	}
	if winAPIErr, ok := err.(*iscsidsc.WinAPICallError); assert.True(t, ok) {
		return assert.Equal(t, expectedErrorCode, winAPIErr.HexCode())
	}
	return false
}
