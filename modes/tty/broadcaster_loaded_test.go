package tty

// broadcaster_loaded_test.go — a console with everything on it (D-120).
//
// THE FIXTURE THE ALLOCATION BUDGET MEASURES, and the reason it exists: the
// budget used to build a console with no pool and no snapshot, so the path that
// actually costs — two tables joining weather for forty rows — was never in the
// number at all.

import (
	"fmt"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// loadedPoolSize is the pool's own cap, stated here because `modes/tty` may not
// import `domains/locations` (make lint-imports) — the same tie the service
// radius bounds already carry.
const loadedPoolSize = 25

// loadedConsole is a station with a full pool, a full running order, and real
// weather behind every one of them.
//
// SEVEN DAYS OF FORECAST EACH, because that is what the providers return and it
// is what makes the day-cell work real: a fixture with two days would measure a
// path the app never takes.
func loadedConsole(t testing.TB, n int) Broadcaster {
	t.Helper()
	pool := make([]snapshot.LocationRef, 0, n)
	locs := make([]snapshot.Location, 0, n)
	now, hi, lo := 20.0, 31.0, 12.0
	for i := range n { // bounded by the ask (P10-02)
		ref := snapshot.LocationRef{Label: fmt.Sprintf("Place %02d, CA", i), Zip: fmt.Sprintf("920%02d", i),
			Lat: 33.2 + float64(i)/50, Lon: -117.2, Population: 1000 * (i + 1)}
		pool = append(pool, ref)
		daily := make([]snapshot.Daily, 0, 7)
		for d := range 7 { // bounded by the forecast (P10-02)
			daily = append(daily, snapshot.Daily{Date: fmt.Sprintf("2026-09-%02d", 13+d),
				Condition: "CLEAR", TempMax: &hi, TempMin: &lo})
		}
		locs = append(locs, snapshot.Location{Label: ref.Label, Zip: ref.Zip, Lat: ref.Lat, Lon: ref.Lon,
			WeatherAsOf: time.Now(),
			Harmonized: snapshot.Conditions{Condition: "CLEAR", Temp: &now,
				Source: snapshot.SourceInfo{Provider: "nws", ModelOrStation: "KOKB"}},
			Daily: daily})
	}
	var l lineup.Lineup
	for i := range MainTrackSlots { // bounded by the track (P10-02)
		c, err := lineup.Propose(lineup.Card{ID: fmt.Sprint("c", i), Slot: lineup.LocationReport,
			Subject: string(snapshot.Key(pool[i%n])), Headline: pool[i%n].Label, State: lineup.Proposed})
		if err != nil {
			t.Fatalf("proposing c%d: %v", i, err)
		}
		if c, err = c.To(lineup.Admitted); err != nil {
			t.Fatalf("admitting c%d: %v", i, err)
		}
		if l, err = l.Queue(lineup.MainTrack, c); err != nil {
			t.Fatalf("queueing c%d: %v", i, err)
		}
	}
	b := NewBroadcaster()
	b.width, b.height = 150, 74
	b, _ = b.Update(LineupMsg{Lineup: l})
	b, _ = b.Update(StationAreaMsg{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
		RadiusMi:    100, Pool: pool})
	b, _ = b.Update(RecentSnapshotMsg{Snap: &snapshot.Snapshot{Locations: locs}})
	return b
}

// AND THE FIXTURE ACTUALLY LOADS IT, which is the premise the budget rests on: a
// console that quietly failed to join would measure the cheap path again under a
// new name.
func TestTheLoadedFixtureActuallyJoins(t *testing.T) {
	b := loadedConsole(t, 25)
	b.ascii = true

	if n := len(b.locIndex()); n != 25 {
		t.Fatalf("the weather index holds %d locations, want 25", n)
	}
	hit := 0
	for _, ref := range b.area.Pool { // bounded by the pool (P10-02)
		if b.locIndex().at(ref) != nil {
			hit++
		}
	}
	if hit != len(b.area.Pool) {
		t.Errorf("%d of %d pool rows join their weather", hit, len(b.area.Pool))
	}
	// AND THE RUNNING ORDER JOINS TOO, which is what D-116 added and what makes
	// this fixture different from the old one.
	rows := b.lineupRows(b.mainTrack(), b.locIndex())
	joined := 0
	for _, r := range rows { // bounded by the table (P10-02)
		if r.Conditions != "" {
			joined++
		}
	}
	if joined == 0 {
		t.Error("no line-up row joined its weather; the budget would measure the cheap path again")
	}
}

// AND OBSERVER STILL GETS ITS EXTENDED DAYS (D-120).
//
// THE CONSOLE SKIPS THEM and Observer draws them, which is the whole shape of
// that change: the day cells are the CALLER's ask. A skip that reached Observer
// would empty its ultra-wide columns — a regression in the shipped surface, paid
// for by an optimisation on the new one, which is the trade the release posture
// exists to refuse.
func TestObserverKeepsItsExtendedDays(t *testing.T) {
	hi, lo := 31.0, 12.0
	daily := make([]snapshot.Daily, 0, 7)
	for d := range 7 { // bounded by the forecast (P10-02)
		daily = append(daily, snapshot.Daily{Date: fmt.Sprintf("2026-09-%02d", 13+d),
			Condition: "CLEAR", TempMax: &hi, TempMin: &lo})
	}
	loc := &snapshot.Location{Label: "Oceanside, CA", Zip: "92057", Daily: daily}

	if got := weatherRow(loc, 50, extRowDays); len(got.Extended) != 5 {
		t.Errorf("Observer's row carries %d extended days, want 5", len(got.Extended))
	}
	// AND THE CONSOLE ASKS FOR NONE, which is the saving.
	if got := weatherRow(loc, 50, 0); len(got.Extended) != 0 {
		t.Errorf("the console's row built %d extended days it never draws", len(got.Extended))
	}
}
