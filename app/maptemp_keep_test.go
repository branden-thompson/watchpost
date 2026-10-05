package app

import (
	"context"
	"slices"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
)

// boxNames is the field boxes' names a view takes.
func boxNames(ask tty.MapAsk) []string {
	var out []string
	for _, b := range fieldBoxes(ask.Region, geo.Box(ask.View)) {
		out = append(out, b.Name)
	}
	return out
}

// THE TEMPERATURE ANSWER IS KEPT FOR ITS HOUR (W14 P-16, D-212): within the
// hour an answer for the same boxes, source, mode and units is the same
// answer, so a second ask - or a pan that keeps the same boxes - asks no
// source again and draws what was built; the next hour asks again.
func TestTheTemperatureAnswerIsKeptForItsHour(t *testing.T) {
	get := &costGet{asked: map[string]bool{}, now: time.Now()}
	lp := &livePipelines{temp: tempSourcesAt(get, "", "", "")}
	ask := tempAsk(false)
	ask.TempNDFD, ask.Anchor = true, get.now.Truncate(time.Hour)
	first := lp.mapTemperature(context.Background(), ask)
	if len(first.Overlays) == 0 || len(get.asked) == 0 {
		t.Fatalf("the first answer drew %d grids from %d asks; this test measures nothing", len(first.Overlays), len(get.asked))
	}
	moved := ask
	moved.View = tty.MapView{W: ask.View.W + 0.5, S: ask.View.S + 0.3, E: ask.View.E + 0.5, N: ask.View.N + 0.3}
	if !slices.Equal(boxNames(ask), boxNames(moved)) {
		t.Fatalf("the pan changed the boxes %v -> %v; it must keep them", boxNames(ask), boxNames(moved))
	}
	for _, again := range []tty.MapAsk{ask, moved} {
		get.asked = map[string]bool{}
		got := lp.mapTemperature(context.Background(), again)
		if len(get.asked) != 0 {
			t.Errorf("an ask within the hour for the same boxes asked %d addresses again", len(get.asked))
		}
		if len(got.Overlays) != len(first.Overlays) {
			t.Errorf("the kept answer drew %d grids; the first %d", len(got.Overlays), len(first.Overlays))
		}
	}
	next := ask
	next.Anchor = ask.Anchor.Add(time.Hour)
	get.asked = map[string]bool{}
	lp.mapTemperature(context.Background(), next)
	if len(get.asked) == 0 {
		t.Error("the next hour asked nothing: the last hour's answer was drawn for it")
	}
}

// A PARTIAL ANSWER IS NOT KEPT: an answer a source failed part of is asked
// again at the next ask, so a source that recovers within the hour is drawn
// within it. Forecast mode's rain days ask Open-Meteo, which the measure
// refuses.
func TestAPartialTemperatureAnswerIsNotKept(t *testing.T) {
	get := &costGet{asked: map[string]bool{}, now: time.Now()}
	lp := &livePipelines{temp: tempSourcesAt(get, "", "", "")}
	ask := tempAsk(true)
	ask.TempNDFD, ask.Anchor = true, get.now.Truncate(time.Hour)
	lp.mapTemperature(context.Background(), ask)
	get.asked = map[string]bool{}
	lp.mapTemperature(context.Background(), ask)
	if len(get.asked) == 0 {
		t.Error("an answer whose rain days failed was kept: nothing was asked again")
	}
}

// A KEPT ANSWER IS NOT CHANGED BY WHAT IS ADDED TO ANOTHER: the answer drawn
// is a copy whose slices and maps are its own, so UV's and air's grids and
// chips added to one ask's answer never reach the next ask's.
func TestAKeptAnswerIsNotChangedByWhatIsAddedToAnother(t *testing.T) {
	get := &costGet{asked: map[string]bool{}, now: time.Now()}
	lp := &livePipelines{temp: tempSourcesAt(get, "", "", "")}
	ask := tempAsk(false)
	ask.TempNDFD, ask.Anchor = true, get.now.Truncate(time.Hour)
	first := lp.mapTemperature(context.Background(), ask)
	n := len(first.Overlays)
	first.Overlays = append(first.Overlays, tuimaps.Overlay{ID: "added"})
	if first.Chips == nil {
		first.Chips = map[string][]string{}
	}
	first.Chips["added"] = []string{"X"}
	again := lp.mapTemperature(context.Background(), ask)
	if len(again.Overlays) != n || again.Chips["added"] != nil {
		t.Errorf("the kept answer has %d grids (want %d) and chips %v: what was added to one answer reached the next", len(again.Overlays), n, again.Chips)
	}
	// AND TWO ANSWERS HANDED OUT DO NOT SHARE ROOM TO GROW: each appends into
	// its own backing array, never over the other's addition.
	a, b := lp.mapTemperature(context.Background(), ask), lp.mapTemperature(context.Background(), ask)
	a.Overlays = append(a.Overlays, tuimaps.Overlay{ID: "a"})
	b.Overlays = append(b.Overlays, tuimaps.Overlay{ID: "b"})
	if a.Overlays[n].ID != "a" {
		t.Errorf("one answer's addition was overwritten by another's: %q", a.Overlays[n].ID)
	}
}

// EACH PART SAYS WHETHER IT IS WHOLE: the temperature where a box was refused
// or filled by another source, the waves where NDFD refused a box - the
// signal a partial answer is not kept by.
func TestEachPartSaysWhetherItIsWhole(t *testing.T) {
	ask := tempAsk(false)
	if _, whole := buildTemperatureWhole(context.Background(), &fakeTemp{name: "NDFD", now: tempNow}, nil, ask, tempNow, nil); !whole {
		t.Error("a temperature every box of which its source drew says it is partial")
	}
	if _, whole := buildTemperatureWhole(context.Background(), &fakeTemp{name: "NDFD", now: tempNow, failed: true}, &fakeTemp{name: "Open-Meteo", now: tempNow}, ask, tempNow, nil); whole {
		t.Error("a temperature filled by Open-Meteo where NDFD refused says it is whole")
	}
	if _, whole := buildTemperatureWhole(context.Background(), &fakeTemp{name: "NDFD", now: tempNow, failed: true}, nil, ask, tempNow, nil); whole {
		t.Error("a temperature no source drew says it is whole")
	}
	keep := waveKeep{land: &landPoints{}}
	if _, whole := withWavesWhole(context.Background(), tty.MapTemperature{}, fakeWaves{metres: 1, max: 2}, fakeWaves{metres: 1, max: 2}, ask, tempNow, keep); !whole {
		t.Error("waves NDFD answered says it is partial")
	}
	if _, whole := withWavesWhole(context.Background(), tty.MapTemperature{}, fakeWaves{failed: true}, fakeWaves{metres: 1, max: 2}, ask, tempNow, keep); whole {
		t.Error("waves NDFD refused says it is whole")
	}
}

// A PAN INTO NEW BOXES ASKS AGAIN: the kept answer is the boxes' it was built
// for, never drawn for others.
func TestAPanIntoNewBoxesAsksAgain(t *testing.T) {
	get := &costGet{asked: map[string]bool{}, now: time.Now()}
	lp := &livePipelines{temp: tempSourcesAt(get, "", "", "")}
	ask := tempAsk(false)
	ask.TempNDFD, ask.Anchor = true, get.now.Truncate(time.Hour)
	lp.mapTemperature(context.Background(), ask)
	east := ask
	east.View = tty.MapView{W: -90, S: 32, E: -85, N: 36}
	if slices.Equal(boxNames(ask), boxNames(east)) {
		t.Fatalf("the pan kept the boxes %v; it must change them", boxNames(ask))
	}
	get.asked = map[string]bool{}
	lp.mapTemperature(context.Background(), east)
	if len(get.asked) == 0 {
		t.Error("a pan into new boxes asked nothing: another view's answer was drawn")
	}
}
