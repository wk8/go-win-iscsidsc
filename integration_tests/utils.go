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

// TODO wkpo
//_, cleanupTarget := setupIscsiTarget(t, "-ChapUser", chapUser, "-ChapPassword", chapPassword, "-DiskCount", "1")
//defer cleanupTarget()
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

// iterateOverAllSubsets will call f with all the 2^n - 1 (unordered) subsets of {0,1,2,...,n}
func iterateOverAllSubsets(n uint, f func(subset []uint)) {
	max := uint(1<<n - 1)
	subset := make([]uint, n)

	generateSubset := func(i uint) []uint {
		index := 0
		for j := uint(0); j < n; j++ {
			if i&(1<<j) != 0 {
				subset[index] = j
				index++
			}
		}
		return subset[:index]
	}

	for i := uint(1); i <= max; i++ {
		f(generateSubset(i))
	}
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
