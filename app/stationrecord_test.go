package app

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/history"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// THE STATION'S FEEDS ARE KEPT (W22.2, D-230, D-231): each fetch the scheduler
// applies is recorded - NWS observations a location's hour, each NWS alert its
// own record with its document as sent, each buoy's readings and each tide
// station's level an hour - and a feed no dataset holds is left alone.
func TestTheStationsFeedsAreKept(t *testing.T) {
	now := time.Date(2026, 10, 3, 15, 20, 0, 0, time.UTC)
	store := history.Open(t.TempDir(), func() time.Time { return now }, nwsObservations, nwsAlerts, ndbcBuoys, coopsTides)
	rec := stationRecorder{store: store}
	ref := snapshot.LocationRef{Label: "Lone Pine, CA", Tag: "LPINE", Lat: 36.6, Lon: -118.06}
	key := snapshot.Key(ref)
	obsAt := time.Date(2026, 10, 3, 14, 53, 0, 0, time.UTC)

	rec.record(snapshot.Fragment{Provider: "nws", Kind: snapshot.KindObs, FetchedAt: now,
		PerLocation: map[snapshot.LocationKey]snapshot.PartialData{key: {Current: &snapshot.Conditions{ObservedAt: obsAt, Temp: f64(21.5), Wind: f64(3)}}}}, []snapshot.LocationRef{ref})
	o, ok := store.Get(nwsObservations.Name, history.Key{Source: "nws", Place: "lone-pine-ca"}, obsAt)
	if !ok || o.Values["temp"][0] != 21.5 || o.Values["wind"][0] != 3 || o.Values["pressure"][0] == o.Values["pressure"][0] {
		t.Errorf("the observation kept is %+v, %v; want the temperature and wind, the pressure missing", o, ok)
	}

	alert := snapshot.Alert{ID: "urn:oid:2.49.0.1.840.0.abc", Event: "Wind Advisory", Sent: now.Add(-time.Hour)}
	rec.record(snapshot.Fragment{Provider: "nws", Kind: snapshot.KindAlerts, FetchedAt: now,
		PerLocation: map[snapshot.LocationKey]snapshot.PartialData{key: {Alerts: []snapshot.Alert{alert}}}}, []snapshot.LocationRef{ref})
	a, ok := store.Get(nwsAlerts.Name, alertKey(alert.ID), alert.Sent)
	var got snapshot.Alert
	if !ok || json.Unmarshal(a.Doc, &got) != nil || got.Event != "Wind Advisory" || got.ID != alert.ID {
		t.Errorf("the alert kept is %+v, %v; want its document as sent", got, ok)
	}

	rec.record(snapshot.Fragment{Provider: "ndbc", Kind: snapshot.KindMarineObs, FetchedAt: now,
		PerLocation: map[snapshot.LocationKey]snapshot.PartialData{key: {Marine: &snapshot.Marine{Buoy: "46225", ObservedAt: obsAt, WaveHeight: f64(1.2), WaterTemp: f64(18)}}}}, []snapshot.LocationRef{ref})
	b, ok := store.Get(ndbcBuoys.Name, history.Key{Source: "ndbc", Place: "46225"}, obsAt)
	if !ok || b.Values["wave_height"][0] != 1.2 || b.Values["water_temp"][0] != 18 {
		t.Errorf("the buoy kept is %+v, %v", b, ok)
	}

	rec.record(snapshot.Fragment{Provider: "coops-obs", Kind: snapshot.KindMarineObs, FetchedAt: now,
		PerLocation: map[snapshot.LocationKey]snapshot.PartialData{key: {Marine: &snapshot.Marine{TideStation: "9410170", TideLevel: f64(0.8)}}}}, []snapshot.LocationRef{ref})
	tl, ok := store.Get(coopsTides.Name, history.Key{Source: "coops", Place: "9410170"}, now)
	if !ok || tl.Values["tide_level"][0] != 0.8 {
		t.Errorf("the tide level kept is %+v, %v", tl, ok)
	}

	later := obsAt.Add(2 * time.Hour) // an hour of its own, so nothing already kept can stand in for it
	rec.record(snapshot.Fragment{Provider: "openmeteo", Kind: snapshot.KindObs, FetchedAt: now,
		PerLocation: map[snapshot.LocationKey]snapshot.PartialData{key: {Current: &snapshot.Conditions{ObservedAt: later, Temp: f64(30)}}}}, []snapshot.LocationRef{ref})
	if _, ok := store.Get(nwsObservations.Name, history.Key{Source: "nws", Place: "lone-pine-ca"}, later); ok {
		t.Error("another provider's conditions were kept as NWS's observations")
	}
}
