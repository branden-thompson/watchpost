package app

// observed_test.go — UAT-2 U2-55 to U2-57: every frame of Radar mode's loop
// draws the temperature, feels-like and wind. NDFD has no hour before the
// current one, so the past is the history's (D-166) - whatever it holds, on
// whichever of the box's lattices - and an hour with nothing is drawn from the
// next newer grid.

import (
	"context"
	"slices"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
)

// covered fails for each observed minute of the loop's past - every five
// minutes back to pastHours before the hour - that no overlay of a layer
// draws during.
func covered(t *testing.T, name string, grids []tuimaps.Overlay, anchor, now time.Time) {
	t.Helper()
	for at := anchor.Add(-pastHours * time.Hour); !at.After(now); at = at.Add(5 * time.Minute) {
		hit := false
		for _, o := range grids {
			if !o.During.From.After(at) && (o.During.Until.IsZero() || !o.During.Until.Before(at)) {
				hit = true
				break
			}
		}
		if !hit {
			t.Errorf("%s: nothing is drawn at %s, an observed frame of the loop", name, at.Format("15:04"))
			return
		}
	}
}

// EVERY OBSERVED FRAME DRAWS THE TEMPERATURE, FEELS LIKE AND WIND (U2-55 to
// U2-57): nothing recorded, the current hour is drawn under all of the loop's
// past; an hour recorded and the next missing, the newer is drawn under the
// gap - never a frame of nothing between two hours that have something.
func TestEveryObservedFrameDrawsTheTemperature(t *testing.T) {
	now := tempNow
	anchor := now.Truncate(time.Hour)
	box := fieldBoxes(tempAsk(false).Region, tempAsk(false).View)[0]
	lat := temperature.NDFDLatticeFor(box.Name, box.Box)
	for name, seed := range map[string][]time.Duration{"nothing recorded": nil, "three hours back alone": {3 * time.Hour}} {
		store := history.Open(t.TempDir(), func() time.Time { return now }, ndfdHourly)
		f := &fakeHour{}
		for _, back := range seed {
			s, _ := f.hour(context.Background(), lat, anchor.Add(-back))
			rec, ok := ndfdRecord(s, history.Key{Source: "ndfd", Place: box.Name}, anchor.Add(-back))
			if !ok || !store.Put(ndfdHourly.Name, rec) {
				t.Fatal("could not seed the history")
			}
		}
		got := buildTemperature(context.Background(), &nowOnly{}, &fakeTemp{name: "Open-Meteo", now: now}, tempAsk(false), now, &fallback{past: recordedHour(store)})
		covered(t, name+", temperature", got.Overlays, anchor, now)
		covered(t, name+", feels like", got.Feels, anchor, now)
		covered(t, name+", wind", got.Wind, anchor, now)
	}
}

// AN HOUR RECORDED ON THE BOX'S COARSER LATTICE IS REPLAYED (D-201): put on
// NDFD's denser lattice's points, and drawn in its own hour.
func TestAnHourRecordedOnTheOldLatticeIsReplayed(t *testing.T) {
	now := tempNow
	anchor := now.Truncate(time.Hour)
	box := fieldBoxes(tempAsk(false).Region, tempAsk(false).View)[0]
	old := temperature.LatticeFor(box.Name, box.Box) // batch 96's NDFD lattice
	store := history.Open(t.TempDir(), func() time.Time { return now }, ndfdHourly)
	f := &fakeHour{}
	s, _ := f.hour(context.Background(), old, anchor.Add(-time.Hour))
	rec, ok := ndfdRecord(s, history.Key{Source: "ndfd", Place: box.Name}, anchor.Add(-time.Hour))
	if !ok || !store.Put(ndfdHourly.Name, rec) {
		t.Fatal("could not seed the history")
	}
	got := buildTemperature(context.Background(), &nowOnly{}, &fakeTemp{name: "Open-Meteo", now: now}, tempAsk(false), now, &fallback{past: recordedHour(store)})
	replayed := false
	for _, o := range got.Overlays {
		if !o.During.From.After(anchor.Add(-time.Hour)) && o.During.Until.Before(anchor) && !o.During.Until.Before(anchor.Add(-time.Second)) {
			replayed = true // a grid of its own, ending with its hour
		}
	}
	if !replayed || !slices.Contains(got.Chips[tty.TemperatureLayer], "RECORDED") {
		t.Errorf("the hour recorded on the old lattice was not replayed in its own hour (chips %v)", got.Chips)
	}
}

// AN OLD RECORD'S WIND DIRECTION IS ITS NEAREST POINT'S (D-201): the speed
// and the temperature are put on the new points by weight, the direction
// never - 350 and 10 degrees would say 180.
func TestAnOldRecordsWindDirectionIsNeverBlended(t *testing.T) {
	box := fieldBoxes(tempAsk(false).Region, tempAsk(false).View)[0]
	old, now := temperature.LatticeFor(box.Name, box.Box), temperature.NDFDLatticeFor(box.Name, box.Box)
	from := make([]float64, old.Cols*old.Rows)
	for i := range from {
		from[i] = 350
		if i%2 == 1 {
			from[i] = 10
		}
	}
	v, ok := onLattice(history.Record{Shape: shapeOf(old), Values: map[string][]float64{"wind_from": from, "temp": from}}, now)
	if !ok || len(v["wind_from"]) != now.Cols*now.Rows {
		t.Fatalf("the old record was not put on the new lattice: %v", ok)
	}
	for i, d := range v["wind_from"] {
		if d != 350 && d != 10 {
			t.Fatalf("point %d's direction is %v; want a point's own, 350 or 10", i, d)
		}
	}
	if _, ok := onLattice(history.Record{Shape: history.Shape{Box: geo.Box{W: 0, S: 0, E: 1, N: 1}, Cols: 2, Rows: 2}}, now); ok {
		t.Error("a record of another box was put on this one's lattice")
	}
}

// THE WAVES TOO (U2-55): NDFD's start at the next hour and nothing recorded,
// Open-Meteo refusing, the next hour is drawn under all of the loop's past.
func TestEveryObservedFrameDrawsTheWaves(t *testing.T) {
	ask := tempAsk(false)
	got := withWaves(context.Background(), tty.MapTemperature{}, fakeWaves{metres: 1, max: 2, fromNext: true}, fakeWaves{failed: true}, ask, tempNow, waveKeep{land: &landPoints{}})
	if len(got.Waves) == 0 {
		t.Fatal("no waves drawn")
	}
	covered(t, "the waves", got.Waves, ask.Anchor, tempNow)
}

// A GAP AT THE CURRENT HOUR IS DRAWN TOO (U2-57): NDFD's current hour can
// come back with no wind; the next newer grid is drawn under it, so no frame
// between two grids is blank.
func TestAGapAtTheCurrentHourIsDrawn(t *testing.T) {
	anchor := tempNow.Truncate(time.Hour)
	hour := func(h time.Duration) tuimaps.Overlay {
		from := anchor.Add(h * time.Hour)
		return tuimaps.Overlay{During: tuimaps.Span{From: from, Until: from.Add(time.Hour - time.Nanosecond)}}
	}
	grids := []tuimaps.Overlay{hour(-1), hour(1), hour(2)} // the current hour missing
	fillPast(grids, anchor)
	covered(t, "the wind with its current hour missing", grids, anchor, anchor.Add(119*time.Minute))
}
