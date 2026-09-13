package tty

// broadcaster_pooldata_test.go — the pool's weather is Observer's weather
// (D-112).

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// ONE CONVERTER, AND THE POOL GOES THROUGH IT.
//
// THE POOL BUILT ITS OWN ROW BY HAND and copied six of sixteen fields: no HI, no
// LOW, no TOMORROW at all, no trend arrow, no fire or seismic marks. Every one of
// those columns drew "n/a" on the table whose purpose the HUM LEAD stated as
// "basic weather info to determine if they want to have that location
// prioritized" — the operator was being asked to decide on data that was there
// and not being shown.
//
// ASSERTED FIELD BY FIELD AGAINST OBSERVER'S OWN ROW, not against literals: what
// makes this stay fixed is that the two rows come from ONE function, and a test
// that restated the expected values would pass just as well if they forked again.
func TestThePoolsWeatherIsObserversWeather(t *testing.T) {
	loc := poolLoc()
	ref := snapshot.LocationRef{Label: loc.Label, Zip: loc.Zip, Lat: loc.Lat, Lon: loc.Lon, Population: 4000}

	b := bcWith(t, card(t, "a", "Oceanside, CA"))
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(StationAreaMsg{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
		RadiusMi:    50, Pool: []snapshot.LocationRef{ref}})
	b, _ = b.Update(RecentSnapshotMsg{Snap: &snapshot.Snapshot{Locations: []snapshot.Location{*loc}}})

	rows := b.poolRows()
	if len(rows) != 1 {
		t.Fatalf("the pool drew %d rows for one candidate", len(rows))
	}
	got := rows[0]
	want := weatherRow(loc, bcFireBoldMW)

	for _, tc := range []struct {
		name      string
		got, want any
	}{
		{"CONDITIONS", got.Conditions, want.Conditions},
		{"NOW", derefF(got.Now), derefF(want.Now)},
		{"trend", got.Trend, want.Trend},
		{"HI", derefF(got.Hi), derefF(want.Hi)},
		{"LOW", derefF(got.Lo), derefF(want.Lo)},
		{"TOMORROW CONDITIONS", got.TomorrowConditions, want.TomorrowConditions},
		{"TOMORROW HI", derefF(got.TomorrowHi), derefF(want.TomorrowHi)},
		{"TOMORROW LOW", derefF(got.TomorrowLo), derefF(want.TomorrowLo)},
		{"WX STN", got.Station, want.Station},
		{"alerts", got.AlertCount, want.AlertCount},
		{"fire", got.Fire, want.Fire},
		{"seismic", got.Seismic, want.Seismic},
		{"loading", got.Loading, want.Loading},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: the pool row says %v, the shared converter says %v", tc.name, tc.got, tc.want)
		}
	}
	// AND THE PREMISE: the fixture actually HAS a forecast, or every comparison
	// above is nil against nil and proves nothing.
	if want.Hi == nil || want.TomorrowHi == nil {
		t.Fatal("the fixture carries no forecast, so this test measures nothing")
	}
}

// WHAT THE POOL KNOWS THAT THE SNAPSHOT DOES NOT stays the pool's: the row's
// number, the population, and how far the place is from the TRANSMITTER — which
// is a different question from how far the observing station is from the place.
func TestThePoolKeepsItsOwnColumns(t *testing.T) {
	loc := poolLoc()
	ref := snapshot.LocationRef{Label: loc.Label, Zip: loc.Zip, Lat: loc.Lat, Lon: loc.Lon, Population: 4000}

	b := bcWith(t, card(t, "a", "Oceanside, CA"))
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(StationAreaMsg{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
		RadiusMi:    50, Pool: []snapshot.LocationRef{ref}})
	b, _ = b.Update(RecentSnapshotMsg{Snap: &snapshot.Snapshot{Locations: []snapshot.Location{*loc}}})

	got := b.poolRows()[0]
	if got.Index != 1 || got.Population != 4000 {
		t.Errorf("the pool lost its own columns: index %d, population %d", got.Index, got.Population)
	}
	// FROM THE TOWER, NOT FROM THE WX STATION. The shared converter fills this
	// with the observing station's distance; the pool overwrites it.
	if got.StationKM == nil {
		t.Fatal("the pool row has no distance")
	}
	if same := weatherRow(loc, bcFireBoldMW).StationKM; same != nil && *same == *got.StationKM {
		t.Error("the pool's DIST is the observing station's, not the transmitter's")
	}
}

func derefF(v *float64) float64 {
	if v == nil {
		return -999
	}
	return *v
}

// poolLoc is a candidate with everything a pool row can show.
func poolLoc() *snapshot.Location {
	now, hi, lo, thi, tlo, stnKM := 73.0, 88.0, 55.0, 91.0, 62.0, 4.2
	return &snapshot.Location{
		Label: "Fallbrook, CA", Zip: "92028", Lat: 33.37, Lon: -117.25,
		WeatherAsOf: time.Now(),
		Harmonized: snapshot.Conditions{
			Condition: "CLEAR", Temp: &now,
			Source: snapshot.SourceInfo{Provider: "nws", ModelOrStation: "KOKB", DistanceKm: &stnKM},
		},
		Daily: []snapshot.Daily{
			{Date: "2026-09-12", Condition: "CLEAR", TempMax: &hi, TempMin: &lo},
			{Date: "2026-09-13", Condition: "RAIN", TempMax: &thi, TempMin: &tlo},
		},
	}
}

// THE MARKS REACH THE ROW, ALL THREE OF THEM (D-113).
//
// HUM LEAD, UAT 2026-09-12: "fix this issues so seismic / fire / alerts show up
// in the location pool as expected."
//
// TWO OF THE THREE COULD NOT APPEAR WHATEVER THE DATA SAID. `fillPoolWeather`
// copied the alert fields and never touched `Fire` or `Seismic`, so those columns
// were blank by construction — and a blank mark reads as "nothing is happening
// there", which on a table for deciding what to put on the air is the wrong
// answer told confidently.
func TestThePoolRowCarriesEveryMark(t *testing.T) {
	loc := poolLoc()
	frp := 80.0
	loc.Alerts = []snapshot.Alert{{ID: "a1", Event: "Severe Thunderstorm Warning", Severity: "Severe"}}
	loc.Fire.Hotspots = []snapshot.Hotspot{{FRPMW: &frp}}
	loc.Seismic = &snapshot.SeismicState{Quakes: []snapshot.Quake{{Mag: 4.5}}}
	ref := snapshot.LocationRef{Label: loc.Label, Zip: loc.Zip, Lat: loc.Lat, Lon: loc.Lon}

	b := bcWith(t, card(t, "a", "Oceanside, CA"))
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(StationAreaMsg{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
		RadiusMi:    50, Pool: []snapshot.LocationRef{ref}})
	b, _ = b.Update(RecentSnapshotMsg{Snap: &snapshot.Snapshot{Locations: []snapshot.Location{*loc}}})

	row := b.poolRows()[0]
	if row.AlertCount != 1 || !row.HasAlert || !row.WarnAlert {
		t.Errorf("the alert marks are missing: has=%v count=%d warn=%v", row.HasAlert, row.AlertCount, row.WarnAlert)
	}
	if row.Fire != 1 || !row.FireHot {
		t.Errorf("the fire marks are missing: count=%d hot=%v", row.Fire, row.FireHot)
	}
	if row.Seismic == 0 {
		t.Error("the seismic mark is missing")
	}
	// AND THEY ARE DRAWN, which is a different claim from being on the row: the
	// marks block is thirteen cells the table can silently leave blank.
	drawn := stripANSITest(strings.Split(b.opts().PoolTable(b.poolRows(), 143), "\n")[4])
	g := b.opts().Glyphs()
	for _, want := range []string{g.Seismic[row.Seismic-1], g.Fire, g.Alert} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the row does not draw %q:\n%q", want, drawn)
		}
	}
}
