package app

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/uv"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
)

// uvTestCity is a city of the EPA fixture's zone.
func uvTestCity(name string, lat, lon float64) geodata.City {
	return geodata.City{Name: name, ASCII: name, State: "CA", Lat: lat, Lon: lon, Population: 100_000, TZ: "America/Los_Angeles"}
}

// labelsOf are every marker's label, overlay by overlay.
func labelsOf(overlays []tuimaps.Overlay) []string {
	var out []string
	for _, o := range overlays {
		for _, f := range o.Features {
			out = append(out, f.Label)
		}
	}
	return out
}

// drawsCity reports whether any marker is labelled with the city.
func drawsCity(overlays []tuimaps.Overlay, name string) bool {
	return slices.ContainsFunc(labelsOf(overlays), func(l string) bool { return strings.HasPrefix(l, name+" ") })
}

// A CITY'S UV IS KEPT IN THE HISTORY (D-224): each hour EPA forecast for a city
// is a record of the city's series - a point at the city, its name, state and
// zone beside it - so the history holds what the map read, for trends.
func TestACitysUVIsKeptInTheHistory(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("no zone data")
	}
	noon := time.Date(2026, 9, 30, 12, 0, 0, 0, la)
	store := history.Open(t.TempDir(), func() time.Time { return noon }, epaUVCities)
	vista := uvTestCity("Vista", 33.2, -117.24)
	c := &uvCities{epa: uv.NewEPA(&epaFixture{}, ""), cities: func(geo.Box, int) []geodata.City { return []geodata.City{vista} }}
	c.markers(context.Background(), geo.Box{W: -118, S: 32, E: -116, N: 35}, 24, noon, false, store)

	day := time.Date(2026, 9, 30, 0, 0, 0, 0, la)
	recs := store.Range(epaUVCities.Name, uvCityKey(vista), day, day.Add(24*time.Hour), 48)
	if len(recs) < 10 {
		t.Fatalf("%d hours of Vista's day were kept; want EPA's day", len(recs))
	}
	r := recs[0]
	if len(r.Values["uv"]) != 1 || r.Shape.Cols != 1 || r.Shape.Rows != 1 || r.Shape.Box.S != vista.Lat || r.Shape.Box.W != vista.Lon {
		t.Errorf("a record is %+v; want one UV value at Vista's point", r)
	}
	var doc uvCityDoc
	if err := json.Unmarshal(r.Doc, &doc); err != nil || doc.Name != "Vista" || doc.State != "CA" || doc.TZ != vista.TZ {
		t.Errorf("the record names %+v (%v); want Vista, CA, its zone", doc, err)
	}
	if !slices.ContainsFunc(historyDatasets, func(d history.Dataset) bool { return d.Name == epaUVCities.Name && d.Days > 0 }) {
		t.Error("the dataset is not among the history's, rolled up for trends")
	}
}

// EVERY KNOWN CITY IN VIEW IS DRAWN (D-225): a city read today - in another
// view, or another session - is drawn beside the spread asked, without asking
// EPA for it; never within the spacing of a city drawn, and never on another
// day than the one it was read for.
func TestEveryKnownCityInViewIsDrawn(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("no zone data")
	}
	noon := time.Date(2026, 9, 30, 12, 0, 0, 0, la)
	now := noon
	dir := t.TempDir()
	box := geo.Box{W: -118, S: 32, E: -116, N: 35}
	vista := uvTestCity("Vista", 33.2, -117.24)
	first := &uvCities{epa: uv.NewEPA(&epaFixture{}, ""), cities: func(geo.Box, int) []geodata.City { return []geodata.City{vista} }}
	first.markers(context.Background(), box, 24, noon, false, history.Open(dir, func() time.Time { return now }, epaUVCities))

	// A later session: the spread asks Barstow alone.
	get := &countingEPA{}
	barstow := uvTestCity("Barstow", 34.9, -117.02)
	spread := []geodata.City{barstow}
	later := &uvCities{epa: uv.NewEPA(get, ""), cities: func(geo.Box, int) []geodata.City { return spread }}
	store := history.Open(dir, func() time.Time { return now }, epaUVCities)
	got := later.markers(context.Background(), box, 24, noon, false, store)
	if !drawsCity(got, "Vista") || !drawsCity(got, "Barstow") {
		t.Fatalf("the view draws %v; want Barstow asked and Vista known", labelsOf(got))
	}
	if n := get.asks(); n != 1 {
		t.Errorf("EPA was asked %d times; want Barstow alone, Vista being known", n)
	}

	// A city asked within the spacing of a known one: the known one gives way.
	spread = []geodata.City{uvTestCity("Oceanside", 33.2, -117.38)}
	near := later.markers(context.Background(), box, 24, noon, false, store)
	if drawsCity(near, "Vista") {
		t.Errorf("the view draws %v; Vista is within the spacing of Oceanside", labelsOf(near))
	}

	// The next day, yesterday's reading is not today's.
	now = noon.Add(24 * time.Hour)
	spread = []geodata.City{barstow}
	next := later.markers(context.Background(), box, 24, now, false, store)
	if drawsCity(next, "Vista") {
		t.Errorf("the next day draws %v; Vista's reading was yesterday's", labelsOf(next))
	}
}

// A CITY'S SERIES IS ITS OWN: two cities of one name in two states are two
// series, and a name is written in the history's alphabet.
func TestACitysSeriesIsItsOwn(t *testing.T) {
	a := uvCityKey(geodata.City{Name: "Springfield", ASCII: "Springfield", State: "IL"})
	b := uvCityKey(geodata.City{Name: "Springfield", ASCII: "Springfield", State: "MO"})
	if a == b {
		t.Errorf("Springfield, IL and Springfield, MO share the series %v", a)
	}
	if k := uvCityKey(geodata.City{Name: "Coeur d'Alene", ASCII: "Coeur d'Alene", State: "ID"}); k.Place != "coeur-d-alene-id" || k.Source != "epa" {
		t.Errorf("Coeur d'Alene is %+v; want epa/coeur-d-alene-id", k)
	}
}

// ANOTHER INSTANCE'S CITIES ARE KNOWN WITHIN THE HOUR: the history's series
// are read again each hour, so a city a second watchpost read is drawn here.
func TestAnotherInstancesCitiesAreKnownWithinTheHour(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("no zone data")
	}
	noon := time.Date(2026, 9, 30, 12, 0, 0, 0, la)
	now := noon
	dir := t.TempDir()
	box := geo.Box{W: -118, S: 32, E: -116, N: 35}
	here := &uvCities{epa: uv.NewEPA(&countingEPA{}, ""), cities: func(geo.Box, int) []geodata.City { return nil }}
	store := history.Open(dir, func() time.Time { return now }, epaUVCities)
	if got := here.markers(context.Background(), box, 24, now, false, store); drawsCity(got, "Vista") {
		t.Fatal("Vista drawn before anyone read it")
	}
	vista := uvTestCity("Vista", 33.2, -117.24)
	there := &uvCities{epa: uv.NewEPA(&epaFixture{}, ""), cities: func(geo.Box, int) []geodata.City { return []geodata.City{vista} }}
	there.markers(context.Background(), box, 24, now, false, history.Open(dir, func() time.Time { return now }, epaUVCities))
	now = noon.Add(knownUVListEvery)
	if got := here.markers(context.Background(), box, 24, now, false, store); !drawsCity(got, "Vista") {
		t.Errorf("an hour after another instance read Vista, this one draws %v", labelsOf(got))
	}
}

// KNOWN CITIES KEEP THE SPACING AMONG THEMSELVES, AND THE VIEW: of two known
// cities within the spacing one is drawn, and a known city outside the view is
// not drawn at all.
func TestKnownCitiesKeepTheSpacingAndTheView(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("no zone data")
	}
	noon := time.Date(2026, 9, 30, 12, 0, 0, 0, la)
	store := history.Open(t.TempDir(), func() time.Time { return noon }, epaUVCities)
	box := geo.Box{W: -118, S: 32, E: -116, N: 35}
	known := []geodata.City{uvTestCity("Vista", 33.2, -117.24), uvTestCity("Oceanside", 33.2, -117.38), uvTestCity("Fresno", 36.74, -119.79)}
	reader := &uvCities{epa: uv.NewEPA(&countingEPA{}, ""), cities: func(geo.Box, int) []geodata.City { return known }} // asked for three at once: the fake that counts under a lock
	reader.markers(context.Background(), box, 24, noon, false, store)

	fresh := &uvCities{epa: uv.NewEPA(&countingEPA{}, ""), cities: func(geo.Box, int) []geodata.City { return nil }}
	got := fresh.markers(context.Background(), box, 24, noon, false, store)
	if drawsCity(got, "Vista") == drawsCity(got, "Oceanside") {
		t.Errorf("the view draws %v; want one of Vista and Oceanside, within the spacing of each other", labelsOf(got))
	}
	if drawsCity(got, "Fresno") {
		t.Errorf("the view draws %v; Fresno is outside it", labelsOf(got))
	}
}
