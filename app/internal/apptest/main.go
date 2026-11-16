package apptest

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Main runs a package's tests with a config and a state directory of the run's
// own, then removes them and exits with the tests' code. No test writes the
// developer's config or state: every preference a setter keeps (D-214) and the
// quota's hold and the kept land (D-218) go there, unless a test points them at
// a directory of its own. Every package under app calls it from its TestMain
// (0.19.6 TG-3, RX-6); a package whose code reads the platform seam applies
// WATCHPOST_TEST_GOOS first (0.19.6 D-24).
func Main(m *testing.M) {
	dir, err := os.MkdirTemp("", "watchpost-app-test-config-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "apptest: no temporary config directory:", err)
		os.Exit(2)
	}
	if err := setDirs(dir); err != nil {
		fmt.Fprintln(os.Stderr, "apptest:", err)
		os.Exit(2)
	}
	code := m.Run()
	_ = os.RemoveAll(dir) // a scratch directory; a failed removal loses nothing
	os.Exit(code)
}

// setDirs points the config and state directories inside dir.
func setDirs(dir string) error {
	if err := os.Setenv("XDG_CONFIG_HOME", dir); err != nil {
		return err
	}
	return os.Setenv("XDG_STATE_HOME", filepath.Join(dir, "state"))
}
