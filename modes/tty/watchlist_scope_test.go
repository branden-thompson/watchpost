package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// AN EMPTY WATCHLIST-SCOPED TAB SAYS WHY, WITH OR WITHOUT LOCATIONS.
//
// The national feed carries nine products and no statements (SAM-D-10), so the
// Warnings tab is nationwide and Advisories and Spec. Statements are not — they
// come only from the zones of the listener's own locations. Nothing else on that
// screen can explain the asymmetry.
//
// The note used to appear ONLY when the watchlist was empty, which is the case
// where a listener already knows why the tab is bare. With locations set — the
// case where it looks like a national view that has missed something — it said
// nothing. Found at UAT 2026-09-06 against a real Alabama statement seen in
// another app.
func TestAnEmptyWatchlistTabExplainsItsScopeEvenWithLocationsSet(t *testing.T) {
	rendering.SetColorEnabledForTest(false)
	for _, tc := range []struct {
		name      string
		withSnap  bool
		mustMatch string
	}{
		{"no locations yet", false, "add locations with ctrl+a"},
		{"locations set", true, "not in the national feed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// dash(t) ALREADY CARRIES A WATCHLIST, so the empty case needs a
			// dashboard that never received a snapshot. Asserting the fixture's
			// numPriority both ways is what caught that: the first version used
			// dash(t) for both and silently tested one case twice.
			var d Dashboard
			if tc.withSnap {
				m, _ := dash(t).(Dashboard).Update(SnapshotMsg{Snap: snap()})
				d = m.(Dashboard)
			} else {
				m, err := NewDashboard(Config{Version: "t"})
				if err != nil {
					t.Fatal(err)
				}
				d = m
			}
			if (d.numPriority() > 0) != tc.withSnap {
				t.Fatalf("the fixture has numPriority=%d for withSnap=%v; it does not pose this case",
					d.numPriority(), tc.withSnap)
			}
			got := stripANSITest(strings.Join(
				d.severeEmptyLines(render.Opts{Width: 133}, 120, 116, category.Of(category.Statements), "0 Total"), "\n"))
			if !strings.Contains(got, "No active") {
				t.Fatalf("the fixture is not showing an empty tab:\n%s", got)
			}
			if !strings.Contains(got, tc.mustMatch) {
				t.Errorf("an empty watchlist-scoped tab must say why it is empty; wanted %q in:\n%s",
					tc.mustMatch, got)
			}
		})
	}
}
