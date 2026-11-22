package ttytest_test

import (
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
)

// TestMain runs ttytest's own tests through the Main it gives every package
// under modes/tty (0.19.6 TG-3). They are an external test package so that the
// call reads ttytest.Main, the selector the check on every TestMain looks for.
func TestMain(m *testing.M) {
	ttytest.Main(m)
}
