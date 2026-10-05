package app

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/airquality"
	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/temperature"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// THE FIRES THE STATION IS TOLD OF ARE KEPT (W22.2, D-231): HMS's and FIRMS's
// detections near each location, by the feed's time; each WFIGS incident's
// acres and containment, an hour a record - the scheduler's, its one writer.
func TestTheStationsFiresAreKept(t *testing.T) {
	now := time.Date(2026, 10, 3, 15, 20, 0, 0, time.UTC)
	store := history.Open(t.TempDir(), func() time.Time { return now }, hmsHotspots, firmsHotspots, wfigsIncidents)
	rec := stationRecorder{store: store}
	ref := snapshot.LocationRef{Label: "Lone Pine, CA", Lat: 36.6, Lon: -118.06}
	key := snapshot.Key(ref)
	part := func(p snapshot.PartialData) map[snapshot.LocationKey]snapshot.PartialData {
		return map[snapshot.LocationKey]snapshot.PartialData{key: p}
	}
	asOf := now.Add(-20 * time.Minute)
	spot := snapshot.Hotspot{Lat: 36.7, Lon: -118.1, DetectedAt: asOf, Confidence: "nominal"}
	rec.record(snapshot.Fragment{Provider: "hms", FetchedAt: now, PerLocation: part(snapshot.PartialData{Fire: &snapshot.FireState{AsOf: asOf, Hotspots: []snapshot.Hotspot{spot}}})}, []snapshot.LocationRef{ref})
	h, ok := store.Get(hmsHotspots.Name, history.Key{Source: "hms", Place: "lone-pine-ca"}, asOf)
	var spots []snapshot.Hotspot
	if !ok || json.Unmarshal(h.Doc, &spots) != nil || len(spots) != 1 || spots[0].Lat != 36.7 {
		t.Errorf("HMS's detections kept are %+v (%v)", spots, ok)
	}
	rec.record(snapshot.Fragment{Provider: "firms", FetchedAt: now, PerLocation: part(snapshot.PartialData{Fire: &snapshot.FireState{AsOf: asOf, Hotspots: []snapshot.Hotspot{spot}}})}, []snapshot.LocationRef{ref})
	if _, ok := store.Get(firmsHotspots.Name, history.Key{Source: "firms", Place: "lone-pine-ca"}, asOf); !ok {
		t.Error("FIRMS's detections were not kept")
	}

	acres := 1200.0
	inc := snapshot.Incident{Name: "Owens Fire", State: "CA", Lat: 36.5, Lon: -118.0, Discovered: now.Add(-48 * time.Hour), Acres: &acres}
	rec.record(snapshot.Fragment{Provider: "wfigs", FetchedAt: now, PerLocation: part(snapshot.PartialData{Fire: &snapshot.FireState{AsOf: asOf, Incidents: []snapshot.Incident{inc}}})}, []snapshot.LocationRef{ref})
	i, ok := store.Get(wfigsIncidents.Name, incidentKey(inc), now)
	if !ok || i.Values["acres"][0] != 1200 || i.Values["contained"][0] == i.Values["contained"][0] {
		t.Errorf("the incident kept is %+v, %v; want its acres, its containment missing", i, ok)
	}

}

// THE HISTORIAN KEEPS THE MAP'S SOURCES, WHETHER OR NOT THE MAP IS USED (D-234):
// once an hour, its one writer - AirNow's national file as one record, the
// USGS feed's quakes by their origin hour (a late report joining its hour),
// NDFD's rain and snow for each box.
func TestTheHistorianKeepsTheMapsSources(t *testing.T) {
	now := time.Date(2026, 10, 3, 15, 20, 0, 0, time.UTC)
	airAsks, quakeAsks := 0, 0
	areas := []airquality.Area{{Name: "Anchorage", State: "AK", Lat: 61.2, Lon: -149.9, Now: &airquality.Reading{AQI: 11},
		Forecast: map[int]airquality.Reading{0: {AQI: 20}, 1: {AQI: math.NaN(), Category: "Good"}}}}
	mag := 4.2
	quake := func(id string, at time.Time) globalfeed.Event {
		return globalfeed.Event{ID: id, At: at, Lat: 36.69, Lon: -118.06, HasPoint: true, Quake: &globalfeed.QuakeDetail{Mag: &mag, DepthKm: 6, Title: id}}
	}
	feed := []globalfeed.Event{quake("ci1", now.Add(-3*time.Hour)), quake("ci2", now.Add(-time.Hour))}
	h := &historian{store: history.Open(t.TempDir(), func() time.Time { return now }, airnowHourly, usgsQuakes, ndfdRainDays),
		air:    func(context.Context, time.Time) ([]airquality.Area, error) { airAsks++; return areas, nil },
		quakes: func(context.Context) []globalfeed.Event { quakeAsks++; return feed },
		totals: func(_ context.Context, l temperature.Lattice, _ time.Time) (temperature.Totals, error) {
			var t temperature.Totals
			t.Lattice = l
			for k := range temperature.Days {
				t.QPF[k], t.Snow[k] = make([]float64, l.Cols*l.Rows), make([]float64, l.Cols*l.Rows)
			}
			return t, nil
		},
		hour: func(context.Context, temperature.Lattice, time.Time) (temperature.Series, error) {
			return temperature.Series{}, errors.New("not this test's")
		},
		regions: func() []string { return []string{geo.RegionHawaii} }}
	h.pass(context.Background(), now)
	h.pass(context.Background(), now.Add(5*time.Minute))
	if airAsks != 1 || quakeAsks != 1 {
		t.Errorf("AirNow was asked %d times and the feed %d in one hour; want once each", airAsks, quakeAsks)
	}
	got, ok := airAt(h.store, now.Truncate(time.Hour))
	if !ok || len(got) != 1 || got[0].Now.AQI != 11 || got[0].Forecast[0].AQI != 20 || !math.IsNaN(got[0].Forecast[1].AQI) || got[0].Forecast[1].Category != "Good" {
		t.Fatalf("the hour's national record reads %+v, %v; want Anchorage 11 now, 20 today, Good tomorrow", got, ok)
	}
	hourOf := func(at time.Time) []quakeDoc {
		rec, _ := h.store.Get(usgsQuakes.Name, quakeSeries, at)
		var qs []quakeDoc
		_ = json.Unmarshal(rec.Doc, &qs)
		return qs
	}
	if q := hourOf(now.Add(-3 * time.Hour)); len(q) != 1 || q[0].ID != "ci1" {
		t.Errorf("three hours ago holds %+v; want ci1", q)
	}
	feed = append(feed, quake("ci3", now.Add(-3*time.Hour).Add(10*time.Minute))) // a late report
	h.pass(context.Background(), now.Add(time.Hour))
	if q := hourOf(now.Add(-3 * time.Hour)); len(q) != 2 {
		t.Errorf("the late report did not join its hour: %+v", q)
	}
	box := recordedBoxes(geo.RegionHawaii)[0]
	if _, ok := h.store.Get(ndfdRainDays.Name, history.Key{Source: "ndfd", Place: box.Name}, dayStart(now.In(time.Local), 0)); !ok {
		t.Error("NDFD's rain and snow for today were not kept")
	}
}

// THE MAP DRAWS AIRNOW FROM THE HOUR'S RECORD (D-234): where the historian has
// kept the hour, the map reads it and does not fetch the file again.
func TestTheMapDrawsAirNowFromTheHoursRecord(t *testing.T) {
	now := time.Now()
	get := &airnowFile{}
	store := history.Open(t.TempDir(), func() time.Time { return now }, airnowHourly)
	airNational(store, []airquality.Area{{Name: "Fresno", State: "CA", Lat: 36.7, Lon: -119.8, Now: &airquality.Reading{AQI: 77}}}, now.Truncate(time.Hour))
	lp := &livePipelines{airnow: airquality.New(get, ""), history: store}
	got := lp.airnowIn(context.Background(), tty.MapAsk{Air: true, Anchor: now.Truncate(time.Hour)})
	if len(got) != 1 || got[0].Name != "Fresno" || get.asked != 0 {
		t.Errorf("the map drew %+v and asked the file %d times; want the hour's record, no ask", got, get.asked)
	}
}
