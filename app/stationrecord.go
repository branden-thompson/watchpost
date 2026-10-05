package app

// stationrecord.go — the station's feeds kept as the scheduler applies them
// (W22.2, D-230, D-231): NWS observations and alerts, which NWS keeps about a
// week, and the buoys' and tide stations' readings, whose archives are kept as
// a local copy for the Analyst mode.

import (
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// nwsObservations is NWS's latest observation for each location, an hour a
// record (D-230).
var nwsObservations = history.Dataset{
	Name: "nws-observations", Version: 1, Step: time.Hour,
	Title:       "NWS, the observations",
	Description: "The National Weather Service's latest station observation for each location, an hour a record.",
	Fields: []history.Field{
		{Name: "temp", Label: "Temperature", Unit: "°C", Decimals: 1},
		{Name: "feels", Label: "Feels like", Unit: "°C", Decimals: 1},
		{Name: "dewpoint", Label: "Dew point", Unit: "°C", Decimals: 1},
		{Name: "humidity", Label: "Humidity", Unit: "%", Decimals: 0},
		{Name: "pressure", Label: "Pressure", Unit: "hPa", Decimals: 1},
		{Name: "wind", Label: "Wind", Unit: "m/s", Decimals: 1},
		{Name: "wind_from", Label: "Wind from", Unit: "°", Decimals: 0},
		{Name: "gust", Label: "Gusts", Unit: "m/s", Decimals: 1},
		{Name: "precip_1h", Label: "Precipitation, the hour", Unit: "mm", Decimals: 1},
		{Name: "visibility", Label: "Visibility", Unit: "m", Decimals: 0},
	},
	Hours: 72 * time.Hour,
	Days:  30 * 24 * time.Hour,
}

// nwsAlerts is each NWS alert the station's locations were sent, its own
// series, its document as NWS sent it, kept at the hour it was sent (D-230).
var nwsAlerts = history.Dataset{
	Name: "nws-alerts", Version: 1, Step: time.Hour,
	Title:       "NWS, the alerts",
	Description: "Each National Weather Service alert the station's locations were sent, its document as sent, at the hour it was sent.",
	Hours:       72 * time.Hour, // the Data tab's retention applies to every dataset alike (D-175, D-231)
}

// ndbcBuoys is each NDBC buoy's readings, an hour a record (D-231).
var ndbcBuoys = history.Dataset{
	Name: "ndbc-buoys", Version: 1, Step: time.Hour,
	Title:       "NDBC, the buoys",
	Description: "The National Data Buoy Center's readings at each buoy the station's locations are near, an hour a record.",
	Fields: []history.Field{
		{Name: "wave_height", Label: "Wave height", Unit: "m", Decimals: 1},
		{Name: "wave_period", Label: "Wave period", Unit: "s", Decimals: 0},
		{Name: "swell_height", Label: "Swell height", Unit: "m", Decimals: 1},
		{Name: "swell_from", Label: "Swell from", Unit: "°", Decimals: 0},
		{Name: "wind_wave_height", Label: "Wind-wave height", Unit: "m", Decimals: 1},
		{Name: "wind", Label: "Wind", Unit: "m/s", Decimals: 1},
		{Name: "gust", Label: "Gusts", Unit: "m/s", Decimals: 1},
		{Name: "water_temp", Label: "Water temperature", Unit: "°C", Decimals: 1},
	},
	Hours: 72 * time.Hour,
	Days:  30 * 24 * time.Hour,
}

// coopsTides is each CO-OPS tide station's observed level, an hour a record
// (D-231).
var coopsTides = history.Dataset{
	Name: "coops-tides", Version: 1, Step: time.Hour,
	Title:       "CO-OPS, the tide level",
	Description: "NOAA CO-OPS's observed water level at each tide station the station's locations are near, above MLLW, an hour a record.",
	Fields:      []history.Field{{Name: "tide_level", Label: "Tide level", Unit: "m", Decimals: 2}},
	Hours:       72 * time.Hour,
	Days:        30 * 24 * time.Hour,
}

// recordFragment keeps what a fetch the scheduler applied holds, in the
// history as it stands at that moment.
func (lp *livePipelines) recordFragment(f snapshot.Fragment, refs []snapshot.LocationRef) {
	stationRecorder{store: lp.historyStore()}.record(f, refs)
}

// stationRecorder writes the station's feeds into the history.
type stationRecorder struct {
	store *history.Store
}

// record keeps each location's part of a fetch: its observation, its alerts,
// its buoy's and its tide station's readings - each by the source the
// dataset is of, so another provider's answer is never kept as theirs.
func (r stationRecorder) record(f snapshot.Fragment, refs []snapshot.LocationRef) {
	if r.store == nil {
		return
	}
	for _, ref := range refs { // the locations asked (P10-02)
		part, ok := f.PerLocation[snapshot.Key(ref)]
		if !ok {
			continue
		}
		switch f.Provider {
		case "nws":
			r.observation(ref, part.Current)
			r.alerts(part.Alerts)
		case "ndbc":
			r.buoy(ref, part.Marine)
		case "coops-obs":
			r.tide(ref, part.Marine, f.FetchedAt)
		case "hms", "firms", "wfigs":
			r.fires(ref, part.Fire, f.Provider, f.FetchedAt)
		}
	}
}

// observation keeps a location's NWS observation at its hour.
func (r stationRecorder) observation(ref snapshot.LocationRef, c *snapshot.Conditions) {
	if c == nil || c.ObservedAt.IsZero() {
		return
	}
	r.store.Put(nwsObservations.Name, history.Record{Key: history.Key{Source: "nws", Place: historyPlace(ref.Label)}, At: c.ObservedAt, IssuedAt: c.ObservedAt,
		Shape: pointShape(ref.Lat, ref.Lon), Values: map[string][]float64{
			"temp": reading(c.Temp), "feels": reading(c.Feels), "dewpoint": reading(c.Dewpoint), "humidity": reading(c.HumidityPct),
			"pressure": reading(c.Pressure), "wind": reading(c.Wind), "wind_from": reading(c.WindDirDeg), "gust": reading(c.WindGust),
			"precip_1h": reading(c.Precip1h), "visibility": reading(c.Visibility)}})
}

// alerts keeps each alert, its own series, its document as sent.
func (r stationRecorder) alerts(alerts []snapshot.Alert) {
	for _, a := range alerts { // a location's alerts (P10-02)
		doc, err := json.Marshal(a)
		if err != nil || a.ID == "" || a.Sent.IsZero() {
			continue
		}
		r.store.Put(nwsAlerts.Name, history.Record{Key: alertKey(a.ID), At: a.Sent, IssuedAt: a.Sent, Doc: doc})
	}
}

// buoy keeps a buoy's readings at their hour, at the place it was read for.
func (r stationRecorder) buoy(ref snapshot.LocationRef, m *snapshot.Marine) {
	if m == nil || m.Buoy == "" || m.ObservedAt.IsZero() {
		return
	}
	r.store.Put(ndbcBuoys.Name, history.Record{Key: history.Key{Source: "ndbc", Place: historyPlace(m.Buoy)}, At: m.ObservedAt, IssuedAt: m.ObservedAt,
		Shape: pointShape(ref.Lat, ref.Lon), Values: map[string][]float64{
			"wave_height": reading(m.WaveHeight), "wave_period": reading(m.WavePeriod), "swell_height": reading(m.SwellHeight),
			"swell_from": reading(m.SwellDirDeg), "wind_wave_height": reading(m.WindWaveHeight), "wind": reading(m.WindSpeed),
			"gust": reading(m.WindGust), "water_temp": reading(m.WaterTemp)}})
}

// tide keeps a tide station's observed level at the fetch's hour: the level
// carries no time of its own.
func (r stationRecorder) tide(ref snapshot.LocationRef, m *snapshot.Marine, at time.Time) {
	if m == nil || m.TideStation == "" || m.TideLevel == nil || at.IsZero() {
		return
	}
	r.store.Put(coopsTides.Name, history.Record{Key: history.Key{Source: "coops", Place: historyPlace(m.TideStation)}, At: at, IssuedAt: at,
		Shape: pointShape(ref.Lat, ref.Lon), Values: map[string][]float64{"tide_level": {*m.TideLevel}}})
}

// alertKey is an alert's series: its id, which outruns the history's
// alphabet and length, by its digest.
func alertKey(id string) history.Key {
	return history.Key{Source: "nws", Place: "a" + digest(id)}
}

// reading is an optional reading as one value, missing where there is none.
func reading(v *float64) []float64 {
	if v == nil {
		return []float64{math.NaN()}
	}
	return []float64{*v}
}

// pointShape is a place as the history's shape: a point.
func pointShape(lat, lon float64) history.Shape {
	return history.Shape{Box: geo.Box{W: lon, S: lat, E: lon, N: lat}, Cols: 1, Rows: 1}
}

// historyPlace is a name in the history's alphabet: lower case letters and
// digits, a dash between words, at most 64.
func historyPlace(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) { // a name's runes (P10-02)
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	place := strings.TrimRight(b.String(), "-")
	if len(place) > 64 {
		place = strings.TrimRight(place[:64], "-")
	}
	return place
}
