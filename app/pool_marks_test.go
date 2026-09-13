package app

// pool_marks_test.go — the pool's alert, fire and seismic marks (D-113).
//
// HUM LEAD, UAT 2026-09-12: "fix this issues so seismic / fire / alerts show up
// in the location pool as expected."
//
// TWO OF THE THREE WERE NEVER READ. `fillPoolWeather` copied the alert fields
// and NOT `Fire` or `Seismic`, so those two marks could not appear whatever the
// data said — D-112 fixed that by putting the pool through `weatherRow`. Alerts
// were already being copied, which is why they are the one worth pinning at the
// FETCH end: if they still do not show, the wiring is not where the fault is.

import (
	"slices"
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// THE RECENT PIPELINE ASKS FOR EVERY KIND A MARK IS DRAWN FROM.
//
// THE POOL RIDES THIS TIER SET (D-99) and the console's marks column reads three
// things the weather tiers alone do not carry. A tier dropped from here is a
// column that goes quietly blank on both surfaces — Observer's RECENT table and
// the console's pool — and neither would look broken.
func TestTheRecentTiersFetchWhatTheMarksAreDrawnFrom(t *testing.T) {
	kinds := make([]snapshot.FetchKind, 0, len(recentTiers()))
	for _, tr := range recentTiers() { // bounded by the tier set (P10-02)
		kinds = append(kinds, tr.Kind)
	}
	// FIRE AND SEISMIC ARE PER-LOCATION TIERS; alerts are batched across the
	// whole list at `recentAlertsEvery`, which is why they are not in here.
	for _, want := range []snapshot.FetchKind{snapshot.KindFire, snapshot.KindSeismic} {
		if !slices.Contains(kinds, want) {
			t.Errorf("the recent pipeline never fetches %v, so that mark cannot appear", want)
		}
	}
	if slices.Contains(kinds, snapshot.KindAlerts) {
		t.Error("alerts are batched across the list, not fetched per location (UAT 72)")
	}
	if recentAlertsEvery <= 0 {
		t.Error("and the batched alerts tier has no cadence, so it never runs")
	}
}

// AND THE POOL'S OWN LOCATIONS ARE IN THE LIST BOTH OF THOSE COVER.
//
// ONE LIST, TWO CONSUMERS: the per-location schedulers and the batched alerts
// scheduler are both built from the refs `startRecent` is given, and `commit`
// hands the same composed list to `update`. A pool that reached one and not the
// other would show fire and seismic marks and no alert ones — which is the shape
// this test exists to rule out.
func TestThePoolIsInTheListBothSchedulersCover(t *testing.T) {
	pool := []snapshot.LocationRef{{Label: "Fallbrook, CA", Zip: "92028", Lat: 33.37, Lon: -117.25}}
	recent := []snapshot.LocationRef{{Label: "Boise, ID", Zip: "83702", Lat: 43.62, Lon: -116.2}}

	list := withPool(recent, pool)
	for _, want := range append(append([]snapshot.LocationRef(nil), recent...), pool...) {
		if !slices.ContainsFunc(list, func(r snapshot.LocationRef) bool {
			return snapshot.Key(r) == snapshot.Key(want)
		}) {
			t.Errorf("%s is not in the list the schedulers are built from", want.Label)
		}
	}
}
