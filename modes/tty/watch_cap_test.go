package tty

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// fullWatchlist is a dashboard watching WatchCap places, with the Lookup
// window open to add one more.
func fullWatchlist(t *testing.T) Dashboard {
	t.Helper()
	d, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	locs := make([]snapshot.Location, WatchCap)
	for i := range locs {
		locs[i] = snapshot.Location{Label: fmt.Sprintf("Place %d, CA", i), Zip: fmt.Sprintf("920%02d", i), Lat: 33 + float64(i)/10, Lon: -117}
	}
	d.snap = &snapshot.Snapshot{Locations: locs}
	d.addMode = "add"
	return d
}

// TestTheWatchlistStopsAtItsCap is 0.19.5 Q2.7 (RN-8): ten favourites at most
// (README, UAT 26.3). A full list says so first in the Lookup window, its
// enter does nothing, a place resolved anyway is refused by name of the cap,
// and the focused place's add chip is unavailable.
func TestTheWatchlistStopsAtItsCap(t *testing.T) {
	d := fullWatchlist(t)
	limit := strconv.Itoa(WatchCap)
	if lines := strings.Join(d.addLines(d.opts()), "\n"); !strings.Contains(lines, "Only "+limit+" locations are allowed in the priority list") {
		t.Errorf("a full watchlist's Lookup window does not lead with the cap's note:\n%s", lines)
	}
	m, cmd := d.handleAddKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil || m.(Dashboard).addErr != "" {
		t.Error("enter acted on a full watchlist; it is inert")
	}
	m, _ = d.handleResolved(resolvedMsg{mode: "add", ref: snapshot.LocationRef{Label: "Eleventh, CA", Zip: "92099"}})
	if got := m.(Dashboard).addErr; got != "the priority list is full ("+limit+" locations)" {
		t.Errorf("a place resolved into a full watchlist is refused with %q", got)
	}
	if d.canAddFocused() {
		t.Error("the focused place can be added to a full watchlist")
	}
}
