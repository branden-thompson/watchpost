package console

// broadcaster_loaded_test.go — a console with everything on it
// (0.16.0 D-120).
//
// THE FIXTURE THE ALLOCATION BUDGET MEASURES, and the reason it exists: a budget
// built against a console with no pool and no snapshot leaves the path that
// actually costs — two tables joining weather for forty rows — out of the number
// altogether.

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/internal/allocpin"
	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
)

// loadedConsole is a station with a full pool, a full running order, and real
// weather behind every one of them: ttytest's loaded station, sent to a console.
func loadedConsole(t testing.TB, n int) Broadcaster {
	t.Helper()
	return loadedConsoleAt(t, n, time.Now())
}

// loadedConsoleAt is loadedConsole with its weather observed at a given time,
// so a golden frame of it is the same on every run.
func loadedConsoleAt(t testing.TB, n int, observed time.Time) Broadcaster {
	t.Helper()
	s := ttytest.Loaded(t, n, observed)
	b := New(Opening{})
	b.width, b.height = 150, 74
	b, _ = b.Update(LineupMsg{Lineup: s.Lineup})
	b, _ = b.Update(StationAreaMsg{Transmitter: s.Transmitter, RadiusMi: s.RadiusMi, Pool: s.Pool})
	b, _ = b.WithRecent(s.Recent)
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
	// AND THE RUNNING ORDER JOINS TOO
	// (0.16.0 D-116).
	rows := b.lineupRows(b.MainTrack(), b.locIndex())
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
	for _, r := range b.lineupRows(b.MainTrack(), b.locIndex()) { // bounded by the table (P10-02)
		if r.Conditions != "" {
			n++
		}
	}
	return n
}

// THE CONSOLE'S ROWS CARRY NO EXTENDED DAYS
// (0.16.0 D-120).
//
// PINNED AS A RULE, NOT LEFT TO THE BUDGET. Putting the day cells back costs 351
// allocations a frame and the budget's own five-percent headroom is 464 — so the
// ratchet cannot see it.
func TestTheConsolesRowsBuildNoDayCells(t *testing.T) {
	b := loadedConsole(t, ttytest.LoadedPoolSize)
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

// THE LINE-UP'S ROWS BUILD NO DAY CELLS EITHER — AND ONLY A MEASUREMENT CAN SAY
// SO (0.16.0 D-120).
//
// `render.LineupRow` HAS NO `Extended` FIELD. The console's running order draws
// CONDITIONS and NOW, so day cells built for it are DISCARDED — the work costs
// allocations and changes no output, which means no assertion about what the
// table SAYS can ever catch it. Putting them back passes a rule asserted on the
// pool's rows and passes the frame budget too: 351 allocations against 464 of
// five-percent headroom.
//
// THE ROWS ARE MEASURED ON THEIR OWN, where 351 is most of the number rather
// than four percent of it. The budget stays a ratchet for DRIFT; this is the
// rule.
func TestTheLineupRowsBuildNoDayCells(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	b := loadedConsole(t, ttytest.LoadedPoolSize)
	idx := b.locIndex()
	cards := b.MainTrack()
	// THE PREMISE: the fixture joins, and carries days beyond tomorrow. Without
	// both, this measures a path that builds nothing and would pass on anything.
	if loadedJoins(b) == 0 || len(idx.at(b.area.Pool[0]).Daily) < 3 {
		t.Fatal("the fixture does not exercise the join, so this measures nothing")
	}
	_ = b.lineupRows(cards, idx)

	got := allocpin.Pin(t, "console running-order rows", 50, func() { _ = b.lineupRows(cards, idx) })
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
