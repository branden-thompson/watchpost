package tty

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// consoleUnder is the loaded console behind a Router whose Observer was built
// with these units and this clock, after one message has passed through.
func consoleUnder(t *testing.T, units, clock string) Router {
	t.Helper()
	d, err := NewDashboard(Config{Units: units, Clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	r := NewRouter(d)
	r.broadcaster = loadedConsole(t, loadedPoolSize)
	r.active = SurfaceBroadcaster
	return step(t, r, tickMsg{})
}

// step passes one message through the Router.
func step(t *testing.T, r Router, msg any) Router {
	t.Helper()
	m, _ := r.Update(msg)
	out, ok := m.(Router)
	if !ok {
		t.Fatal("the router must stay the program's model")
	}
	return out
}

// TestTheConsolesPoolWearsTheListenersUnits is 0.19.5 D-5 (DEF-2): the console
// draws the pool in the units Setting, as Observer draws its tables (0.16.0
// FR-6.2). The fixture's places are at 20 °C.
func TestTheConsolesPoolWearsTheListenersUnits(t *testing.T) {
	for _, c := range []struct{ units, want, not string }{
		{"metric", "20°C", "68°F"},
		{"imperial", "68°F", "20°C"},
	} {
		got := stripANSITest(consoleUnder(t, c.units, "").View().Content)
		if !strings.Contains(got, c.want) || strings.Contains(got, c.not) {
			t.Errorf("under %s the console's pool should say %s and never %s", c.units, c.want, c.not)
		}
	}
}

// TestTheConsolesMastheadWearsTheListenersClock is 0.19.5 D-21: the console's
// "Updated:" stamp is written in the clock Setting.
func TestTheConsolesMastheadWearsTheListenersClock(t *testing.T) {
	at := time.Date(2026, 11, 6, 20, 5, 9, 0, time.Local)
	for _, c := range []struct{ clock, want string }{
		{"24h", "20:05:09"},
		{"", "8:05:09 PM"},
	} {
		r := consoleUnder(t, "", c.clock)
		r.broadcaster.now = func() time.Time { return at.Add(time.Minute) }
		r = step(t, r, SnapshotMsg{Snap: &snapshot.Snapshot{GeneratedAt: at}})
		if got := stripANSITest(r.View().Content); !strings.Contains(got, "Updated: 11/06/2026 "+c.want) {
			t.Errorf("under the %q clock the console's masthead should be stamped %s", c.clock, c.want)
		}
	}
}

// TestAUnitsChangeRedrawsTheConsolesPool is 0.19.5 D-21: units changed with
// the console up reach its pool at once, not at the next publish. The pool is
// memoised, so this fails if the units are missing from its key.
func TestAUnitsChangeRedrawsTheConsolesPool(t *testing.T) {
	r := consoleUnder(t, "metric", "")
	_ = r.View().Content // fill the memo in °C
	r.observer.units = r.observer.units ^ 1
	r = step(t, r, tickMsg{})
	if got := stripANSITest(r.View().Content); !strings.Contains(got, "68°F") {
		t.Error("the units changed with the console up, and its pool still draws the old units")
	}
}
