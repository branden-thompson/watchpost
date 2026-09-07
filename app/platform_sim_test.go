package app

import (
	"os"
	"testing"
)

// RUN THE WHOLE PACKAGE AS ANOTHER PLATFORM, because the first time 0.14.0 ran
// on Linux was its release PR, and it panicked.
//
// asPlatform steers one test at a time, which only helps for a test that already
// knew it was platform-dependent. The defects that reached CI were the ones that
// did NOT know: app/voices.go read runtime.GOOS behind the seam's back, and a
// test branched on the real OS while the code under test used the seam. Neither
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
	os.Exit(m.Run())
}
