package tty

// broadcaster_pooldata_test.go — the pool's weather is Observer's weather.

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty/internal/frame"
	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// ONE CONVERTER, AND THE POOL GOES THROUGH IT.
//
// A POOL ROW BUILT BY HAND copies a few fields and drops the rest — HI, LOW,
// TOMORROW, the trend arrow, the fire and seismic marks. Every one of those
// columns then draws "n/a" on the table whose purpose the HUM LEAD stated as
// "basic weather info to determine if they want to have that location
// prioritized" — asking the operator to decide on data that is there and not
// shown.
//
// ASSERTED FIELD BY FIELD AGAINST OBSERVER'S OWN ROW, not against literals: what
// makes this stay fixed is that the two rows come from ONE function, and a test
// that restated the expected values would pass just as well if they forked.
func TestThePoolsWeatherIsObserversWeather(t *testing.T) {
	loc := ttytest.PoolLoc()
	ref := snapshot.LocationRef{Label: loc.Label, Zip: loc.Zip, Lat: loc.Lat, Lon: loc.Lon, Population: 4000}

	b := bcWith(t, ttytest.Card(t, "a", "Oceanside, CA"))
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(StationAreaMsg{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
		RadiusMi:    50, Pool: []snapshot.LocationRef{ref}})
	b, _ = b.WithRecent(&snapshot.Snapshot{Locations: []snapshot.Location{*loc}})

	rows := b.poolRows(b.locIndex())
	if len(rows) != 1 {
		t.Fatalf("the pool drew %d rows for one candidate", len(rows))
	}
	got := rows[0]
	want := frame.WeatherRow(loc, frame.FireBoldDefaultMW, 0)

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
	loc := ttytest.PoolLoc()
	ref := snapshot.LocationRef{Label: loc.Label, Zip: loc.Zip, Lat: loc.Lat, Lon: loc.Lon, Population: 4000}

	b := bcWith(t, ttytest.Card(t, "a", "Oceanside, CA"))
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(StationAreaMsg{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
		RadiusMi:    50, Pool: []snapshot.LocationRef{ref}})
	b, _ = b.WithRecent(&snapshot.Snapshot{Locations: []snapshot.Location{*loc}})

	got := b.poolRows(b.locIndex())[0]
	if got.Index != 1 || got.Population != 4000 {
		t.Errorf("the pool lost its own columns: index %d, population %d", got.Index, got.Population)
	}
	// FROM THE TOWER, NOT FROM THE WX STATION. The shared converter fills this
	// with the observing station's distance; the pool overwrites it.
	if got.StationKM == nil {
		t.Fatal("the pool row has no distance")
	}
	if same := frame.WeatherRow(loc, frame.FireBoldDefaultMW, 0).StationKM; same != nil && *same == *got.StationKM {
		t.Error("the pool's DIST is the observing station's, not the transmitter's")
	}
}

func derefF(v *float64) float64 {
	if v == nil {
		return -999
	}
	return *v
}

// THE MARKS REACH THE ROW, ALL THREE OF THEM.
//
// HUM LEAD, UAT 2026-09-12: "fix this issues so seismic / fire / alerts show up
// in the location pool as expected."
//
// A FILL THAT COPIES ONLY THE ALERT FIELDS leaves `Fire` and `Seismic` blank by
// construction, whatever the data says — and a blank mark reads as "nothing is
// happening there", which on a table for deciding what to put on the air is the
// wrong answer told confidently.
func TestThePoolRowCarriesEveryMark(t *testing.T) {
	loc := ttytest.PoolLoc()
	frp := 80.0
	loc.Alerts = []snapshot.Alert{{ID: "a1", Event: "Severe Thunderstorm Warning", Severity: "Severe"}}
	loc.Fire.Hotspots = []snapshot.Hotspot{{FRPMW: &frp}}
	loc.Seismic = &snapshot.SeismicState{Quakes: []snapshot.Quake{{Mag: 4.5}}}
	ref := snapshot.LocationRef{Label: loc.Label, Zip: loc.Zip, Lat: loc.Lat, Lon: loc.Lon}

	b := bcWith(t, ttytest.Card(t, "a", "Oceanside, CA"))
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(StationAreaMsg{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
		RadiusMi:    50, Pool: []snapshot.LocationRef{ref}})
	b, _ = b.WithRecent(&snapshot.Snapshot{Locations: []snapshot.Location{*loc}})

	row := b.poolRows(b.locIndex())[0]
	if row.AlertCount != 1 || !row.HasAlert || !row.WarnAlert {
		t.Errorf("the alert marks are missing: has=%v count=%d warn=%v", row.HasAlert, row.AlertCount, row.WarnAlert)
	}
	if row.Fire != 1 || !row.FireHot {
		t.Errorf("the fire marks are missing: count=%d hot=%v", row.Fire, row.FireHot)
	}
	if row.Seismic == 0 {
		t.Error("the seismic mark is missing")
	}
	// The marks are drawn, which is a different claim from being on the row: the
	// marks block is thirteen cells the table can silently leave blank.
	drawn := ttytest.StripANSI(strings.Split(b.opts().PoolTable(b.poolRows(b.locIndex()), 143), "\n")[4])
	g := b.opts().Glyphs()
	for _, want := range []string{g.Seismic[row.Seismic-1], g.Fire, g.Alert} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the row does not draw %q:\n%q", want, drawn)
		}
	}
}

// THE RUNNING ORDER SHOWS THE WEATHER WHERE EACH BEAT IS.
//
// HUM LEAD, 2026-09-13, choosing this over wiring the correspondents: "I think
// we change P R E S E N T A T I O N / CORRESPONDENT to C U R R E N T L Y /
// CONDITIONS NOW … Useful information and doesn't require the correspondents
// wiring work."
//
// THE JOIN IS FREE, and that is why it is the right trade: `producer()` may only
// offer locations from the station's POOL, so every card on this track is about a
// place the pool already holds — and the pool's weather is already fetched. No
// new call, no new cadence.
func TestTheRunningOrderCarriesEachBeatsWeather(t *testing.T) {
	loc := ttytest.PoolLoc()
	frp := 80.0
	loc.Alerts = []snapshot.Alert{{ID: "a1", Event: "Severe Thunderstorm Warning", Severity: "Severe"}}
	loc.Fire.Hotspots = []snapshot.Hotspot{{FRPMW: &frp}}
	loc.Seismic = &snapshot.SeismicState{Quakes: []snapshot.Quake{{Mag: 4.5}}}
	ref := snapshot.LocationRef{Label: loc.Label, Zip: loc.Zip, Lat: loc.Lat, Lon: loc.Lon}

	// A CARD ABOUT THAT PLACE. `Card.Subject` IS the location key (topoff.go), so
	// this is the identity the join matches on.
	// THE TABLE STARTS AT SLOT 2 (`ScheduledFrom`): slot 0 is LIVE and slot 1 is
	// the UP NEXT card, so the beat under test needs two ahead of it to be DRAWN
	// at all. Without them this passes on an empty table for the wrong reason.
	c := ttytest.Card(t, "c", string(snapshot.Key(ref)))
	b := bcWith(t, ttytest.Card(t, "a", "live"), ttytest.Card(t, "b", "upnext"), c)
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(StationAreaMsg{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.28, Lon: -117.23},
		RadiusMi:    50, Pool: []snapshot.LocationRef{ref}})
	b, _ = b.WithRecent(&snapshot.Snapshot{Locations: []snapshot.Location{*loc}})

	rows := b.lineupRows(b.MainTrack(), b.locIndex())
	var got *render.LineupRow
	for i := range rows { // bounded by the track (P10-02)
		if rows[i].Zip == loc.Zip {
			got = &rows[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("no row joined to the pool entry; the card's subject is %q", c.Subject)
	}
	want := frame.WeatherRow(loc, frame.FireBoldDefaultMW, 0)
	if got.Conditions != want.Conditions || got.Now == nil || *got.Now != *want.Now {
		t.Errorf("the row says %q/%v; the shared converter says %q/%v",
			got.Conditions, got.Now, want.Conditions, want.Now)
	}
	// The prefix marks are the place's (HUM LEAD: "Relevant Prefix Alerts").
	if got.Marks.AlertCount != want.AlertCount || got.Marks.Fire != want.Fire || got.Marks.Seismic != want.Seismic {
		t.Errorf("the row's marks are alerts=%d fire=%d seismic=%d; the place has %d/%d/%d",
			got.Marks.AlertCount, got.Marks.Fire, got.Marks.Seismic,
			want.AlertCount, want.Fire, want.Seismic)
	}
	// There is no correspondent column, and no `N/A` in its place.
	drawn := ttytest.StripANSI(b.opts().LineupTable(rows, 143))
	if strings.Contains(drawn, "CORRESPONDENT") || strings.Contains(drawn, "N/A") {
		t.Errorf("the running order still carries the correspondent column:\n%s", drawn)
	}
	if !strings.Contains(drawn, "C U R R E N T L Y") {
		t.Errorf("the running order does not name what replaced it:\n%s", drawn)
	}
}
