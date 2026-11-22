package frame

import (
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
)

// TestMain stands frame's tests on the ground every package under modes/tty
// shares: the zone pinned to UTC before any goroutine exists (0.19.6 TG-3).
func TestMain(m *testing.M) {
	ttytest.Main(m)
}
