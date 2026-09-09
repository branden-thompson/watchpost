package tty

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// A LOOKED-UP LOCATION THAT NEVER RETURNS DATA STOPS SAYING "LOADING" (#13).
//
// "If you look up a location and the commit succeeds but the data never lands —
// bad geocode, no NWS coverage, somewhere the API just doesn't answer for — the
// ref is already in RECENT/SEARCHED. It sits there with no data, permanently."
// It shimmered, so it read as *still loading*, for ever and across restarts.
//
// THE ISSUE ASKED WHETHER THE TWO ARE EVEN DISTINGUISHABLE, and at the time
// they were not: nothing recorded that a fetch had been ATTEMPTED for a
// location, so "no data yet" and "no data ever" were the same value. This
// tests the distinction, which is the actual fix; the treatment was already
// there, because a post-load nil has always drawn an honest "n/a".
func TestALocationTheFeedCannotServeStopsShimmering(t *testing.T) {
	answered := time.Now().Add(-2 * time.Minute)

	// BEFORE THE FEED HAS ANSWERED: shimmer. This is the case that must keep
	// working — a fix that simply stopped shimmering would pass a test that
	// only looked at the empty row.
	waiting := &snapshot.Location{Label: "Vista, CA"}
	if !rowLoading(waiting) {
		t.Error("a location no fetch has covered yet is still loading; it must shimmer")
	}

	// AFTER A FETCH COVERED IT AND BROUGHT NOTHING: not loading.
	unserved := &snapshot.Location{Label: "Nowhere, XX", WeatherAsOf: answered}
	if rowLoading(unserved) {
		t.Error("the feed answered and had nothing for this place: that is a fact, not a wait")
	}

	// AND THE ROW SAYS SO ON SCREEN rather than shimmering, using the "n/a"
	// the post-load path already drew.
	d := dash(t)
	s := snap()
	s.Locations = []snapshot.Location{{Label: "Nowhere, XX", Tag: "NOWH", WeatherAsOf: answered}}
	m, _ := d.Update(SnapshotMsg{Snap: s})
	frame := stripANSITest(m.(Dashboard).View().Content)
	dots := m.(Dashboard).opts().LoadingDots()
	row := ""
	for _, l := range strings.Split(frame, "\n") {
		if strings.Contains(l, "Nowhere, XX") {
			row = l
		}
	}
	if row == "" {
		t.Fatalf("the location is not on the frame at all:\n%s", frame)
	}
	if strings.Contains(row, strings.TrimSpace(dots)) {
		t.Errorf("the row still shimmers after the feed answered:\n%q", row)
	}
	if !strings.Contains(row, "n/a") {
		t.Errorf("the row should read n/a — an honest absence, not a wait:\n%q", row)
	}
}
