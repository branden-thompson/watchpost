package app

// memokeys_test.go — 0.19.0 W9.1 (FR-8.2, #12): the app-side memos' keys
// cover what their answers are built from.

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
)

// tempKeyInputs says, for every field of the map's ask, which field of the
// temperature answer's key carries it - or why the answer is not built
// from it. A field added to the ask fails the test below until it is
// classified here: "not in the key" is a decision, never an oversight (#12).
var tempKeyInputs = map[string]string{
	"Region":             "region",
	"View":               "boxes",  // the region's boxes the view holds
	"TempNDFD":           "source", // the source chosen from it, by name
	"Forecast":           "forecast",
	"Fahrenheit":         "fahrenheit",
	"RainFull":           "rainFull",
	"RadarAhead":         "radarAhead",
	"Anchor":             "anchor",
	"Snap":               "not an input: the grids are the region's, not the watched places'",
	"Place":              "not an input: the view, not the selected place, sets the boxes",
	"RadarIEM":           "not an input: the radar's source, not the temperature's",
	"UVCities":           "not an input: UV is read after the grids, from the exact view (tempCore's note)",
	"QuakeFeed":          "not an input: the quakes' feed",
	"Clock":              "not an input: how times are written, not what the grids hold",
	"Buoys":              "not an input: a row of its own, asked apart",
	"Tides":              "not an input: a row of its own, asked apart",
	"Fire":               "not an input: a row of its own, asked apart",
	"Quakes":             "not an input: a row of its own, asked apart",
	"AlertsOff":          "not an input: the alerts' rows",
	"AlertCategoriesOff": "not an input: the alerts' rows",
	"UV":                 "not an input: UV is read after the grids, from the exact view",
	"Air":                "not an input: air quality is read after the grids, from the exact view",
	"FireMode":           "not an input: the fire's choice",
	"Part":               "not an input: which part of the feed is delivered first",
}

// TestTheAppMemoKeysCoverTheirInputs is FR-8.2 (#12): every field of the
// map's ask is either carried by the temperature answer's key or said not
// to be its input; each carried field changes the key; and the keys built
// from their data's own names - a radar frame's, an air grid's, a UV day's -
// are pinned field by field with why nothing else changes their answer.
func TestTheAppMemoKeysCoverTheirInputs(t *testing.T) {
	keyFields := map[string]bool{}
	for f := range reflect.TypeFor[tempKey]().Fields() {
		keyFields[f.Name] = true
	}
	for f := range reflect.TypeFor[tty.MapAsk]().Fields() {
		into, ok := tempKeyInputs[f.Name]
		switch {
		case !ok:
			t.Errorf("the ask's %s is not classified for the temperature key: carry it, or say why it is no input", f.Name)
		case len(into) < 13 || into[:13] != "not an input:":
			if !keyFields[into] {
				t.Errorf("the ask's %s is said to be carried by tempKey.%s, which does not exist", f.Name, into)
			}
		}
	}
	lp, src, now := &livePipelines{}, &nowOnly{}, time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	base := tty.MapAsk{Region: geo.RegionContiguous, View: tty.MapView{W: -100, S: 30, E: -90, N: 40}, Anchor: now}
	key := lp.tempKeyFor(base, src, now)
	for name, change := range map[string]func(*tty.MapAsk){
		"Region":     func(a *tty.MapAsk) { a.Region = geo.RegionAlaska },
		"View":       func(a *tty.MapAsk) { a.View = tty.MapView{W: -124.5, S: 32.5, E: -114, N: 42} },
		"Forecast":   func(a *tty.MapAsk) { a.Forecast = true },
		"Fahrenheit": func(a *tty.MapAsk) { a.Fahrenheit = true },
		"RainFull":   func(a *tty.MapAsk) { a.RainFull = true },
		"RadarAhead": func(a *tty.MapAsk) { a.RadarAhead = 6 },
		"Anchor":     func(a *tty.MapAsk) { a.Anchor = now.Add(time.Hour) },
	} {
		ask := base
		change(&ask)
		if lp.tempKeyFor(ask, src, now) == key {
			t.Errorf("a changed %s leaves the temperature key as it was", name)
		}
	}
	for _, pin := range []struct {
		name, why string
		typ       reflect.Type
		fields    []string
	}{
		{"frameKey", "a radar frame at its time never changes; HRRR's hours ahead are a run's", reflect.TypeFor[frameKey](), []string{"source", "region", "box", "at", "run", "size"}},
		{"airGridKey", "AirNow's contours are one file an hour", reflect.TypeFor[airGridKey](), []string{"box", "hour"}},
		{"uvDayKey", "EPA's forecast is a city's day", reflect.TypeFor[uvDayKey](), []string{"name", "state", "date"}},
	} {
		var got []string
		for f := range pin.typ.Fields() {
			got = append(got, f.Name)
		}
		if !slices.Equal(got, pin.fields) {
			t.Errorf("%s's fields are %v; pinned as %v because %s - an input added is a reason to revisit", pin.name, got, pin.fields, pin.why)
		}
	}
}
