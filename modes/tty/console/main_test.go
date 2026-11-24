package console

import (
	"flag"
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
)

// TestMain stands the console's tests on the ground every package under
// modes/tty shares: the zone pinned to UTC before any goroutine exists (0.19.6
// TG-3).
func TestMain(m *testing.M) {
	ttytest.Main(m)
}

// updateGolden re-captures the console's goldens; each package's flag is its
// own, and the golden helper takes its value (0.19.6 A-33).
var updateGolden = flag.Bool("update-golden", false, "re-capture testdata/console-*.golden")
