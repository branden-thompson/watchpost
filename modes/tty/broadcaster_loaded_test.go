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

// loadedJoins is how many of the console's rows actually carry joined weather.
//
// SHARED WITH THE BUDGET, which asks it before reporting a number.
func loadedJoins(b Broadcaster) int {
	n := 0
	for _, r := range b.lineupRows(b.mainTrack(), b.locIndex()) { // bounded by the table (P10-02)
		if r.Conditions != "" {
			n++
		}
	}
	return n
}

// THE CONSOLE'S ROWS CARRY NO EXTENDED DAYS (D-120).
//
// PINNED AS A RULE, NOT LEFT TO THE BUDGET. Putting the day cells back costs 351
// allocations a frame and the budget's own five-percent headroom is 464 — so the
// ratchet cannot see the regression it was re-based alongside. A mutant proved
// exactly that by surviving.
func TestTheConsolesRowsBuildNoDayCells(t *testing.T) {
	b := loadedConsole(t, loadedPoolSize)
	for _, r := range b.poolRows(b.locIndex()) { // bounded by the pool (P10-02)
		if len(r.Extended) != 0 {
			t.Fatalf("a pool row carries %d extended days; neither console table draws one",
				len(r.Extended))
		}
	}
	// AND THE PREMISE: the fixture HAS days beyond tomorrow, or this asserts that
	// nothing was built out of nothing.
	if len(b.locIndex().at(b.area.Pool[0]).Daily) < 3 {
		t.Fatal("the fixture carries no extended forecast, so this measures nothing")
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

// THE LINE-UP'S ROWS BUILD NO DAY CELLS EITHER — AND ONLY A MEASUREMENT CAN SAY
// SO (D-120).
//
// `render.LineupRow` HAS NO `Extended` FIELD. The console's running order draws
// CONDITIONS and NOW, so day cells built for it are DISCARDED — the work costs
// allocations and changes no output, which means no assertion about what the
// table SAYS can ever catch it. A mutant that put them back survived a rule
// asserted on the pool's rows and survived the frame budget too: 351 allocations
// against 464 of five-percent headroom.
//
// SO THE ROWS ARE MEASURED ON THEIR OWN, where 351 is most of the number rather
// than four percent of it. The budget stays a ratchet for DRIFT; this is the
// rule.
func TestTheLineupRowsBuildNoDayCells(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	b := loadedConsole(t, loadedPoolSize)
	idx := b.locIndex()
	cards := b.mainTrack()
	// THE PREMISE: the fixture joins, and carries days beyond tomorrow. Without
	// both, this measures a path that builds nothing and would pass on anything.
	if loadedJoins(b) == 0 || len(idx.at(b.area.Pool[0]).Daily) < 3 {
		t.Fatal("the fixture does not exercise the join, so this measures nothing")
	}
	_ = b.lineupRows(cards, idx)

	got := testing.AllocsPerRun(50, func() { _ = b.lineupRows(cards, idx) })
	t.Logf("lineupRows over %d slots: %.0f allocs (budget %d)", MainTrackSlots, got, bcLineupRowAllocs)
	if got > bcLineupRowAllocs {
		t.Errorf("building the running order's rows allocates %.0f, budget %d — the day cells "+
			"nothing draws are %d of them; re-pin DELIBERATELY with the reason",
			got, bcLineupRowAllocs, 351)
	}
}

// bcLineupRowAllocs is what building the running order's rows costs.
//
// MEASURED AT 260 AND PINNED AT x1.05. Tight on purpose: the thing it guards is
// 351 allocations — more than the whole of this number — so a five-percent band
// around the frame's nine thousand could never see it, and one around this can
// see it several times over.
const bcLineupRowAllocs = 273
