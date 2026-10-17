package app

// mapnames_test.go — 0.18.0 UAT-1 U1-12 (D-64): the title names what is in
// view, by scale, from the station's city index and the regions.

import (
	"testing"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
)

func TestTheTitleNamesTheViewByScale(t *testing.T) {
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	name := mapAreaNamer(idx)
	oceanside := tuimaps.LonLat{Lon: -117.38, Lat: 33.2}
	for _, c := range []struct {
		at      tuimaps.LonLat
		widthKm float64
		want    string
	}{
		{oceanside, 40, "Oceanside, CA"},
		{oceanside, 400, "Southern California"},
		{tuimaps.LonLat{Lon: -122.4, Lat: 40.6}, 400, "Northern California"},
		{tuimaps.LonLat{Lon: -95.37, Lat: 29.76}, 400, "Eastern Texas"},
		{tuimaps.LonLat{Lon: -119.5, Lat: 37.2}, 400, "Central California"},
		{oceanside, 1200, "California"},
		{oceanside, 3000, "the contiguous United States"},
		{tuimaps.LonLat{Lon: -150, Lat: 61.2}, 3000, "Alaska"},
		{tuimaps.LonLat{Lon: 0, Lat: 0}, 400, ""},
	} {
		if got := name(c.at, c.widthKm); got != c.want {
			t.Errorf("at %v, %v km across: %q, want %q", c.at, c.widthKm, got, c.want)
		}
	}
	if mapAreaNamer(nil)(oceanside, 40) != "" {
		t.Error("with no index the namer names something")
	}
}

// TestAPartOfAStateIsSpelledWhole: far out on both axes a part is one word.
func TestAPartOfAStateIsSpelledWhole(t *testing.T) {
	e := geodata.Extent{W: 0, S: 0, E: 10, N: 10}
	for at, want := range map[tuimaps.LonLat]string{
		{Lon: 1, Lat: 9}: "Northwestern", {Lon: 9, Lat: 1}: "Southeastern", {Lon: 9, Lat: 5}: "Eastern",
		{Lon: 1, Lat: 5}: "Western", {Lon: 5, Lat: 5}: "Central", {Lon: 5, Lat: 1}: "Southern",
	} {
		if got := partOfState(e, at); got != want {
			t.Errorf("at %v: %q, want %q", at, got, want)
		}
	}
	if got := partOfState(geodata.Extent{}, tuimaps.LonLat{}); got != "Central" {
		t.Errorf("an empty extent reads %q", got)
	}
}

// TestTheDetailAndTheNamerReachTheWindow is D-64 and D-65's wiring: the
// file's detail choices are handed to the window and written back, and the
// window is handed the station's namer.
func TestTheDetailAndTheNamerReachTheWindow(t *testing.T) {
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	lp := &livePipelines{idx: idx}
	cfg := lp.ttyConfig("t", Options{}, false, config.Config{MapDetail: map[string]bool{"roads": true}, MapDetailLevel: "standard"}, nil, nil, nil, nil, nil, nil)
	if !cfg.MapDetailChoice["roads"] || cfg.MapDetailLevel != "standard" {
		t.Errorf("the window is handed detail %v at %q", cfg.MapDetailChoice, cfg.MapDetailLevel)
	}
	if cfg.MapAreaName == nil || cfg.MapAreaName(tuimaps.LonLat{Lon: -117.38, Lat: 33.2}, 400) != "Southern California" {
		t.Error("the window is not handed the station's namer")
	}
	if cfg.PlaceNear == nil || cfg.PlaceNear(35.5, 139.5) == "" {
		t.Error("the window is not handed the world's namer")
	}
	withConfigFile(t)
	if err := setUIHook(tty.UIPrefs{Units: "metric", MapDetail: map[string]bool{"parks": true}, MapDetailLevel: "full"}); err != nil {
		t.Fatal(err)
	}
	if got, err := config.Load(); err != nil || !got.MapDetail["parks"] || got.MapDetailLevel != "full" {
		t.Errorf("the save wrote %v at %q (%v)", got.MapDetail, got.MapDetailLevel, err)
	}
}

// TestAHamletAtTheCentreNamesTheTown: the place named is the nearest town of
// some size, not a hamlet a listener would not know.
func TestAHamletAtTheCentreNamesTheTown(t *testing.T) {
	near := []geodata.City{{Name: "Hamlet", Population: 200}, {Name: "Town", Population: 9000}}
	if got := placeInView(near); got.Name != "Town" {
		t.Errorf("named %q", got.Name)
	}
	if got := placeInView(near[:1]); got.Name != "Hamlet" {
		t.Errorf("with no town, named %q", got.Name)
	}
}

// TestAPointIsNamedAnywhere is BUILD red team AX-1: the Propagation mode's
// centre, a point anywhere on the globe, is named by the town nearest it in
// any country; with none within placeNearKm, nothing, and the window says the
// coordinates. Asked again at the same point, the name is the one kept.
func TestAPointIsNamedAnywhere(t *testing.T) {
	idx, err := geodata.Load()
	if err != nil {
		t.Fatal(err)
	}
	name := placeNamer(idx)
	for _, c := range []struct {
		lat, lon float64
		suffix   string
	}{{35.5, 139.5, ", JP"}, {51.5, 0, ", GB"}, {-34, 151, ", AU"}, {33, -117, ", CA"}} {
		if got := name(c.lat, c.lon); len(got) <= len(c.suffix) || got[len(got)-len(c.suffix):] != c.suffix {
			t.Errorf("(%.1f, %.1f) is named %q; a town ending %q", c.lat, c.lon, got, c.suffix)
		}
	}
	if got := name(-40, -140); got != "" {
		t.Errorf("the open South Pacific is named %q; nothing", got)
	}
	if got := placeNamer(nil)(35.5, 139.5); got != "" {
		t.Errorf("with no index, %q; nothing", got)
	}
	name(35.5, 139.5)
	allocs := testing.AllocsPerRun(20, func() { name(35.5, 139.5) }) // a point with towns: a scan would parse them
	if allocs != 0 {
		t.Errorf("the same point asked again cost %.0f allocations; the name kept costs none", allocs)
	}
}
