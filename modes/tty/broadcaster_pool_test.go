package tty

// broadcaster_pool_test.go — D-98/D-99: the station's candidates, with the
// weather the operator decides on.

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func poolConsole(t *testing.T) Broadcaster {
	t.Helper()
	b := bcWith(t, card(t, "a", "Oceanside, CA"))
	b.height = 96
	b, _ = b.Update(StationAreaMsg{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
		RadiusMi:    25,
		Pool: []snapshot.LocationRef{
			{Label: "Bonsall, CA", Zip: "92003", Lat: 33.28, Lon: -117.23, Population: 4000},
			{Label: "Fallbrook, CA", Zip: "92028", Lat: 33.37, Lon: -117.25, Population: 35000},
		},
	})
	return b
}

// THE POOL DRAWS WHAT THE OPERATOR DECIDES ON: the place, how far out it is, how
// many people it serves, and its weather.
func TestThePoolTableDrawsTheStationsCandidates(t *testing.T) {
	got := stripANSITest(poolConsole(t).View().Content)
	for _, want := range []string{
		"L O C A T I O N     P O O L", "POPULATION",
		"Fallbrook, CA", "92028", "35,000",
		"Showing 1 - 2 of 2 Location Pool Locations",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the pool table is missing %q", want)
		}
	}
}

// AND IT IS THE SAME TABLE OBSERVER DRAWS, which is the requirement: the group
// bands and the weather columns come from one renderer, so they cannot drift.
func TestThePoolTableIsObserversTable(t *testing.T) {
	got := stripANSITest(poolConsole(t).View().Content)
	for _, want := range []string{"T O D A Y", "T O M O R R O W", "CONDITIONS", "NOW", "HI", "LOW"} {
		if !strings.Contains(got, want) {
			t.Errorf("the pool table is missing Observer's %q column", want)
		}
	}
}

// A LOCATION WITH NO DATA YET SHIMMERS, which is Observer's own answer while its
// data is in flight (UAT 18.2) — and the honest one, because the pool rides the
// recent pipeline and the data IS coming.
func TestAPoolRowWithNoDataYetShimmers(t *testing.T) {
	b := poolConsole(t)
	if b.pool != nil {
		t.Fatalf("the fixture has already been given a snapshot; this measures nothing")
	}
	for _, r := range b.poolRows() {
		if !r.Loading {
			t.Errorf("%q draws as settled with no snapshot behind it", r.Name)
		}
	}
}

// AND ONE WITH DATA STOPS SHIMMERING, through the SAME harmonised fields
// Observer's own row reads — so a pool row and a watchlist row for one place
// cannot disagree about the weather there.
func TestAPoolRowTakesItsWeatherFromTheRecentSnapshot(t *testing.T) {
	b := poolConsole(t)
	temp := 72.0
	snap := &snapshot.Snapshot{Locations: []snapshot.Location{
		{Label: "Fallbrook, CA", Lat: 33.37, Lon: -117.25},
	}}
	// `WeatherAsOf` IS WHAT SETTLES A ROW (rowLoading): a location whose weather
	// has landed stops shimmering, and the fixture has to say it landed or it is
	// asserting the shimmer rather than the data.
	snap.Locations[0].WeatherAsOf = time.Now()
	snap.Locations[0].Harmonized.Condition = "Clear"
	snap.Locations[0].Harmonized.Temp = &temp
	b, _ = b.Update(RecentSnapshotMsg{Snap: snap})

	var found bool
	for _, r := range b.poolRows() {
		if r.Name != "Fallbrook, CA" {
			continue
		}
		found = true
		if r.Loading {
			t.Error("Fallbrook has a snapshot and still shimmers")
		}
		if r.Conditions != "Clear" || r.Now == nil || *r.Now != temp {
			t.Errorf("the row did not take the snapshot's weather: %+v", r)
		}
	}
	if !found {
		t.Fatal("Fallbrook is in the pool and produced no row")
	}
}

// THE MASTHEAD'S STAMP IS NOT THE POOL'S. The two snapshots are kept apart: the
// priority pipeline's is what the masthead reports, and merging them would have
// the console claim a freshness it does not have.
func TestTheRecentSnapshotDoesNotBecomeTheMastheads(t *testing.T) {
	b := poolConsole(t)
	before := b.snap
	b, _ = b.Update(RecentSnapshotMsg{Snap: &snapshot.Snapshot{}})
	if b.snap != before {
		t.Error("the recent snapshot overwrote the one the masthead reports")
	}
	if b.pool == nil {
		t.Error("the recent snapshot did not reach the pool")
	}
}
