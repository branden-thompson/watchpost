package tty

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// broadcaster_memo_completeness_test.go — F-30's guard, for the console.
//
// THE PROPERTY, stated as an implication:
//
//	tables(a) != tables(b)  =>  consoleKeyFor(a) != consoleKeyFor(b)
//
// "If the tables would look different, the key must differ." The converse is not
// required — a key that changes without the tables changing is a wasted render,
// not a lie. A memo may MISS; it must never wrongly HIT.
//
// DERIVED, NOT LISTED. The fields come from the struct by reflection, so a field
// added to the console next release is perturbed without anyone remembering to
// add it here. Every hand-kept version of this guard in this codebase has been
// found incomplete — a table with 15 rows for 22 fields, a list of "windows with
// a cursor" that could not describe a window whose moving state was not one.
func TestTheConsoleMemoKeyCoversEverythingTheTablesShow(t *testing.T) {
	// WITH COLOUR ON, AND THAT IS NOT A DETAIL. Perturbing `selected` moves the
	// FOCUS TINT and nothing else — so with colour off the tables came back
	// byte-identical, the guard reported "this field does not reach the tables",
	// and it would have gone on reporting that while a stale highlight replayed
	// on the operator's screen. A guard run in a mode the app does not ship in
	// measures a frame the app does not draw.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)

	base := loadedConsole(t, loadedPoolSize)
	base.width, base.height, base.ascii = 150, 74, true
	const used = 20

	// WITH A FIRE BURNING, because the threshold is an input and a fixture with
	// no hotspots cannot tell 50 from 200. Dropping `fireBoldMW` from the key
	// passed this guard until this line existed: the field reached nothing,
	// truthfully, in a world with nothing to reach. Set here rather than in
	// `loadedConsole` so the benchmark and the alloc budget keep measuring the
	// fixture they were pinned against.
	// TEN MEGAWATTS, UNDER THE DEFAULT AND OVER THE PERTURBATION. `bump` adds 1
	// to a float, so the threshold moves 50 → 1 — and a 100 MW fire is over BOTH,
	// which is why the first fixture here still did not catch a dropped
	// `fireBoldMW`. A perturbation that cannot cross the boundary it is testing
	// is not a perturbation.
	frp := 10.0
	base.pool.Locations[0].Fire.Hotspots = []snapshot.Hotspot{{FRPMW: &frp}}

	// AND WITH ONE LOCATION STILL WAITING ON ITS WEATHER, so the shimmer
	// animates and the FRAME is a live input. Without a loading row `anyLoading`
	// is false, the key's whole shimmer arm is dead, and deleting it passed this
	// guard — the one field of twelve that a perturbation of the MODEL could not
	// reach on its own, because the state that makes it matter is in the data.
	base.pool.Locations[1].WeatherAsOf = time.Time{}
	base.pool.Locations[1].Harmonized.Source.Provider = ""
	base.pool.Locations[1].Daily = nil
	if !base.anyLoading() {
		t.Fatal("the fixture must have something loading, or the shimmer is not measured")
	}

	// A GUARD THAT MEASURES NOTHING PASSES VACUOUSLY, so the premise is checked
	// first: the fixture must actually draw both tables.
	s0, p0 := base.buildSpans(used)
	if len(s0.lines) == 0 || len(p0.lines) == 0 {
		t.Fatalf("the fixture draws no tables (%d scheduled, %d pool rows); this guard would "+
			"pass on anything", len(s0.lines), len(p0.lines))
	}

	n, reached := 0, 0
	perturbEach(t, base, consoleExcuse, consoleWriter, func(field string, next Broadcaster) {
		n++
		a, b := base.buildSpans(used)
		c, d := next.buildSpans(used)
		if sameSpan(a, c) && sameSpan(b, d) {
			return // this field does not reach the tables; nothing is owed
		}
		reached++
		if base.consoleKeyFor(used) == next.consoleKeyFor(used) {
			t.Errorf("changing %s changes the tables and NOT the key: the memo will replay a "+
				"stale table while the model moves underneath it (F-30)", field)
		}
	})
	t.Logf("%d fields perturbed, %d reach the tables", n, reached)

	// AND `used` IS AN INPUT THE WALK STRUCTURALLY CANNOT REACH: it is a
	// PARAMETER, not a field, so no perturbation of the model varies it. It sets
	// the pool's room and the running order's window, so a frame that grew a row
	// above the tables draws different ones — and the key that forgot it would
	// replay the old height for ever. Dropping it from the key passed this guard
	// until this was added, which is why it is here rather than assumed.
	for _, u := range []int{used - 4, used + 4} {
		a, b := base.buildSpans(used)
		c, d := base.buildSpans(u)
		if sameSpan(a, c) && sameSpan(b, d) {
			continue
		}
		if base.consoleKeyFor(used) == base.consoleKeyFor(u) {
			t.Errorf("drawing at used=%d instead of %d changes the tables and NOT the key", u, used)
		}
	}

	// AND THE TWO INPUTS THE WALK CANNOT PERTURB EITHER. A `lineup.Lineup` holds
	// an array of slices and a `StationAreaMsg` holds one, so neither is a kind
	// the walk changes — which means their GENERATIONS are never exercised by it,
	// and dropping `lineupGen` from the key passed the whole guard. They are the
	// two most important inputs there are: the running order and the pool.
	for _, tc := range []struct {
		name string
		with func(Broadcaster) Broadcaster
	}{
		{"the running order loses a card", func(b Broadcaster) Broadcaster {
			l, err := b.lineup.Remove("c3")
			if err != nil {
				t.Fatalf("dropping from the fixture's running order: %v", err)
			}
			b, _ = b.Update(LineupMsg{Lineup: l})
			return b
		}},
		{"the weather behind the pool is republished", func(b Broadcaster) Broadcaster {
			// A WHOLE NEW SNAPSHOT, which is how a publish arrives — the pointer
			// moving IS the data changing. The walk never perturbs a pointer, so
			// without this case the tables' own weather was outside the guard.
			next := *b.pool
			next.Locations = append([]snapshot.Location(nil), b.pool.Locations...)
			hotter := 44.4
			next.Locations[0].Harmonized.Temp = &hotter
			b, _ = b.Update(RecentSnapshotMsg{Snap: &next})
			return b
		}},
		{"the pool loses a location", func(b Broadcaster) Broadcaster {
			a := b.area
			a.Pool = a.Pool[:len(a.Pool)-1]
			b, _ = b.Update(StationAreaMsg(a))
			return b
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next := tc.with(base)
			a, b := base.buildSpans(used)
			c, d := next.buildSpans(used)
			if sameSpan(a, c) && sameSpan(b, d) {
				t.Fatalf("%s changed nothing on either table; this case measures nothing", tc.name)
			}
			if base.consoleKeyFor(used) == next.consoleKeyFor(used) {
				t.Errorf("%s changes the tables and NOT the key: the memo will replay the old "+
					"running order while the Director moves underneath it (F-30)", tc.name)
			}
		})
	}
	// AND THE WALK ITSELF IS CHECKED. An engine that enumerated nothing would
	// report this guard green, which is the failure mode a guard cannot have.
	if n < 15 || reached == 0 {
		t.Fatalf("the walk perturbed %d fields of which %d reached the tables; it is not measuring", n, reached)
	}
}

// consoleWriter models the real writer for the two inputs the app replaces
// wholesale, exactly as `dashboardWriter` does for setup's generation: nothing
// in the console edits a Lineup or a StationAreaMsg in place, so a perturbation
// that changed one without moving its counter would be a state the app cannot
// reach — and reporting it would be reporting the design, not a defect.
func consoleWriter(name string, b *Broadcaster) {
	switch name {
	case "lineup":
		b.lineupGen++
	case "area":
		b.areaGen++
	}
}

// consoleExcuse names a field this walk does NOT descend into, and why.
var consoleExcuse = map[string]string{
	"memo": "the memo itself: perturbing the cache is not a state the model can be in",
}

// sameSpan compares two spans by everything the frame reads out of them.
func sameSpan(a, b scrollSpan) bool {
	if a.from != b.from || a.off != b.off || a.total != b.total || len(a.lines) != len(b.lines) {
		return false
	}
	for i := range a.lines {
		if a.lines[i] != b.lines[i] {
			return false
		}
	}
	return true
}
