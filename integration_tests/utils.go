package integrationtests

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wk8/go-win-iscsidsc/internal"
	"github.com/wk8/go-win-iscsidsc/targetportal"
)

var targetPotalEndpointRegex = regexp.MustCompile("^((?:[0-9]{1,3}\\.){3}(?:[0-9]{1,3})):([1-9][0-9]*)$")

// getLocalTargetPortals returns a map of the target portals currently listening locally,
// mapping the IP they listen on to the portal structure.
// Fails the test immediately if there are none, or if the WinTarget service is not running.
func getLocalTargetPortals(t *testing.T) map[string]*targetportal.Portal {
	powershellCommand := "(Get-IscsiTargetServerSetting).Portals | Where Enabled | ForEach {$_.IPEndpoint.ToString()}"
	output, err := exec.Command("powershell", "/c", powershellCommand).Output()
	if err != nil {
		t.Fatalf("Unable to list local target portals: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	portals := make(map[string]*targetportal.Portal)
	for _, line := range lines {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		match := targetPotalEndpointRegex.FindStringSubmatch(line)
		if match == nil {
			t.Fatalf("Unexpected line when listing local target portals: %q", line)
		}

		port, err := strconv.ParseUint(match[2], 10, 16)
		if err != nil {
			t.Fatalf("Unable to convert %q to a uint16: %v", match[2], err)
		}
		portUint16 := uint16(port)

		portals[match[1]] = &targetportal.Portal{
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
func getUnregisteredLocalTargetPortals(t *testing.T) ([]*targetportal.Portal, []targetportal.PortalInfo) {
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

// registerLocalTargetPortal gets the first unregistered portal from getUnregisteredLocalTargetPortals,
// registers it with the default options, and returns it along with a targetPortalCleaner to unregister it
// when done with testing
func registerLocalTargetPortal(t *testing.T) (*targetportal.Portal, *targetPortalCleaner) {
	portals, _ := getUnregisteredLocalTargetPortals(t)
	portal := portals[0]

	err := targetportal.AddIScsiSendTargetPortal(nil, nil, nil, nil, portal)
	require.Nil(t, err)

	return portal, newTargetPortalCleaner(portal)
}

// setupIscsiTarget calls hack/setup_iscsi_target to create a target with a random IQN,
// then returns both the target's IQN as well as a function to tear it down when done with testing.
func setupIscsiTarget(t *testing.T, extraArgs ...string) (string, func()) {
	iqn := "iqn.2019-06.com.github.wk8.go-win-iscsids.test:" + randomHexString(t, 80)

	runHackPowershellScript(t, "setup_iscsi_target", append([]string{iqn}, extraArgs...)...)

	return iqn, func() {
		runHackPowershellScript(t, "teardown_iscsi_targets", iqn)
	}
}

// runHackPowershellScript calls a script from the hack/ directory
func runHackPowershellScript(t *testing.T, scriptName string, args ...string) {
	scriptPath := path.Join(repoRoot(t), "hack", scriptName+".ps1")

	if output, err := exec.Command("powershell", append([]string{scriptPath}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("Error when running script %q: %v, and output: %s", scriptPath, err, string(output))
	}
}

func repoRoot(t *testing.T) string {
	_, filePath, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("Unable to resolve path to repo root")
	}
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

func randomHexString(t *testing.T, length int) string {
	b := length / 2
	randBytes := make([]byte, b)

	if n, err := rand.Read(randBytes); err != nil || n != b {
		if err == nil {
			err = fmt.Errorf("only got %v random bytes, expected %v", n, b)
		}
		t.Fatal(err)
	}

	return hex.EncodeToString(randBytes)
}

func assertStringInSlice(t *testing.T, needle string, slice []string) {
	for _, item := range slice {
		if item == needle {
			return
		}
	}
	t.Errorf("%q not found in %v", needle, slice)
}

// setSmallInitialApiBufferSize changes the value of internal.InitialApiBufferSize to 1,
// and returns a func to revert that change when done with testing.
func setSmallInitialApiBufferSize() func() {
	previousBufferSize := internal.InitialApiBufferSize
	internal.InitialApiBufferSize = 1
	return func() {
		internal.InitialApiBufferSize = previousBufferSize
	}
}
