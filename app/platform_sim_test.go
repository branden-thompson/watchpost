package app

import (
	"os"
	"testing"
)

// RUN THE WHOLE PACKAGE AS ANOTHER PLATFORM, so a platform defect shows here
// rather than the first time the code runs on Linux.
//
// asPlatform steers one test at a time, which only helps for a test that already
// knows it is platform-dependent. The defects that slip through are the ones that
// do NOT know: code that reads runtime.GOOS behind the seam's back, or a test
// that branches on the real OS while the code under test uses the seam. Neither
// is visible on the machine it was written on, because there the two agree.
//
//	WATCHPOST_TEST_GOOS=linux go test ./app     (or `make test-linux`)
//
// This is TEST-ONLY: production still initialises the seam from runtime.GOOS,
// and nothing here is compiled into the binary. It simulates the SEAM, not the
// operating system — real syscalls, paths and audio are still the host's — so a
// green run here is evidence about platform BRANCHING, not a substitute for
// running on Linux. That distinction is the whole reason the Linux protocol
// still has to happen on real hardware.
func TestMain(m *testing.M) {
	if goos := os.Getenv("WATCHPOST_TEST_GOOS"); goos != "" {
		setRuntimeGOOS(goos)
	}
	// NO TEST WRITES THE DEVELOPER'S CONFIG: every preference a setter keeps
	// (D-214) goes to a directory of this run's own unless a test points it at
	// one of its own (withConfigFile).
	dir, err := os.MkdirTemp("", "watchpost-app-test-config-")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("XDG_CONFIG_HOME", dir); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
