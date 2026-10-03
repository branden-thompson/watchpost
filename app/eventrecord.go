package app

// eventrecord.go — the fires, quakes and air quality the app reads, kept in
// the history (W22.2, D-231): their sources keep archives of their own, and
// these are the local copy, shaped and catalogued for the Analyst mode.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"time"

	"github.com/branden-thompson/watchpost/domains/airquality"
	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/history"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// hmsHotspots is HMS's satellite detections near each location, the list at
// the feed's time.
var hmsHotspots = history.Dataset{
	Name: "hms-hotspots", Version: 1, Step: time.Hour,
	Title:       "HMS, the hotspots",
	Description: "NOAA HMS's satellite fire detections near each location, the list at the feed's time.",
	Hours:       72 * time.Hour, // the Data tab's retention applies to every dataset alike (D-175, D-231)
}

// firmsHotspots is FIRMS's detections near each location, the list at the
// feed's time.
var firmsHotspots = history.Dataset{
	Name: "firms-hotspots", Version: 1, Step: time.Hour,
	Title:       "FIRMS, the hotspots",
	Description: "NASA FIRMS's fire detections near each location, the list at the feed's time.",
	Hours:       72 * time.Hour, // the Data tab's retention applies to every dataset alike (D-175, D-231)
}

// wfigsIncidents is each WFIGS incident the station's locations are near,
// its size and containment an hour a record.
var wfigsIncidents = history.Dataset{
	Name: "wfigs-incidents", Version: 1, Step: time.Hour,
	Title:       "WFIGS, the incidents",
	Description: "Each NIFC WFIGS wildfire incident near the station's locations: its acres and containment, an hour a record.",
	Fields: []history.Field{
		{Name: "acres", Label: "Acres", Unit: "acres", Decimals: 0},
		{Name: "contained", Label: "Contained", Unit: "%", Decimals: 0},
	},
	Hours: 72 * time.Hour, // the Data tab's retention applies to every dataset alike (D-175, D-231)
	Days:  30 * 24 * time.Hour,
}

// usgsQuakes is the USGS feed's earthquakes, an hour a record: each record the
// quakes whose origin falls in that hour, rewritten as late reports arrive
// (D-234: the historian's, whether or not the map is used).
var usgsQuakes = history.Dataset{
	Name: "usgs-quakes", Version: 1, Step: time.Hour,
	Title:       "USGS, the earthquakes",
	Description: "The USGS feed's earthquakes of magnitude 1.0 and up, each hour's by their origin time.",
	Hours:       72 * time.Hour, // the Data tab's retention applies to every dataset alike (D-175, D-231)
}

// airnowHourly is AirNow's national file, an hour a record: every reporting
// area's AQI and its forecasts for today and tomorrow (D-234: one record, not
// a series an area, which cost seconds of disk an ask).
var airnowHourly = history.Dataset{
	Name: "airnow-hourly", Version: 1, Step: time.Hour,
	Title:       "AirNow, the national file",
	Description: "U.S. EPA AirNow's reporting areas (preliminary data, not fully verified), the national file an hour: each area's AQI and its forecasts for today and tomorrow.",
	Hours:       72 * time.Hour, // the Data tab's retention applies to every dataset alike (D-175, D-231)
}

// quakeDoc is a quake as the history keeps it, whichever feed told of it.
type quakeDoc struct {
	ID      string    `json:"id,omitempty"` // the USGS id where the feed gave one
	Mag     float64   `json:"mag"`
	MagType string    `json:"mag_type,omitempty"`
	Place   string    `json:"place"`
	DepthKm float64   `json:"depth_km"`
	At      time.Time `json:"at"`
	Lat     float64   `json:"lat"`
	Lon     float64   `json:"lon"`
	Tsunami bool      `json:"tsunami"`
}

// incidentDoc names an incident's series.
type incidentDoc struct {
	Name       string    `json:"name"`
	State      string    `json:"state"`
	Discovered time.Time `json:"discovered"`
}

// fires keeps a location's fire feed: HMS's and FIRMS's detections as their
// list, WFIGS's incidents each its own series.
func (r stationRecorder) fires(ref snapshot.LocationRef, f *snapshot.FireState, provider string, at time.Time) {
	if f == nil {
		return
	}
	switch provider {
	case "hms", "firms":
		r.hotspots(ref, f, provider)
	case "wfigs":
		for _, inc := range f.Incidents { // a location's incidents (P10-02)
			r.incident(inc, at)
		}
	}
}

// hotspots keeps a feed's detections near a location at the feed's time.
func (r stationRecorder) hotspots(ref snapshot.LocationRef, f *snapshot.FireState, provider string) {
	doc, err := json.Marshal(f.Hotspots)
	if err != nil || f.AsOf.IsZero() || len(f.Hotspots) == 0 {
		return
	}
	set := hmsHotspots
	if provider == "firms" {
		set = firmsHotspots
	}
	r.store.Put(set.Name, history.Record{Key: history.Key{Source: provider, Place: historyPlace(ref.Label)}, At: f.AsOf, IssuedAt: f.AsOf, Doc: doc})
}

// incident keeps an incident's size and containment at the fetch's hour.
func (r stationRecorder) incident(inc snapshot.Incident, at time.Time) {
	doc, err := json.Marshal(incidentDoc{Name: inc.Name, State: inc.State, Discovered: inc.Discovered})
	if err != nil || at.IsZero() || inc.Name == "" {
		return
	}
	r.store.Put(wfigsIncidents.Name, history.Record{Key: incidentKey(inc), At: at, IssuedAt: at, Shape: pointShape(inc.Lat, inc.Lon),
		Values: map[string][]float64{"acres": reading(inc.Acres), "contained": reading(inc.PercentContained)}, Doc: doc})
}

// quakeHours keeps the feed's quakes by their origin hour, each hour's
// record rewritten with what the feed holds for it now - a late report joins
// its hour.
func quakeHours(store *history.Store, events []globalfeed.Event, now time.Time) {
	byHour := map[time.Time][]quakeDoc{}
	for _, e := range events { // the feed's quakes (P10-02)
		if e.Quake == nil || !e.HasPoint || e.Quake.Mag == nil || e.At.IsZero() {
			continue
		}
		h := e.At.UTC().Truncate(time.Hour)
		byHour[h] = append(byHour[h], quakeDoc{ID: e.ID, Mag: *e.Quake.Mag, MagType: e.Quake.MagType, Place: e.Quake.Title, DepthKm: e.Quake.DepthKm, At: e.At, Lat: e.Lat, Lon: e.Lon, Tsunami: e.Quake.Tsunami})
	}
	for h, qs := range byHour { // a day's hours (P10-02)
		doc, err := json.Marshal(qs)
		if err != nil {
			continue
		}
		store.Put(usgsQuakes.Name, history.Record{Key: quakeSeries, At: h, IssuedAt: now, Doc: doc})
	}
}

// quakeSeries is the feed's one series.
var quakeSeries = history.Key{Source: "usgs", Place: "m1-day"}

// airSeries is the national file's one series.
var airSeries = history.Key{Source: "airnow", Place: "national"}

// airArea is a reporting area as the history keeps it: a missing reading is
// absent, never NaN, which JSON cannot say.
type airArea struct {
	Name     string             `json:"name"`
	State    string             `json:"state"`
	Lat      float64            `json:"lat"`
	Lon      float64            `json:"lon"`
	Now      *airReading        `json:"now,omitempty"`
	Forecast map[int]airReading `json:"forecast,omitempty"`
}

// airReading is an AQI, its number where AirNow gave one, and its category.
type airReading struct {
	AQI      *float64 `json:"aqi,omitempty"`
	Category string   `json:"category,omitempty"`
}

// toAir is a reading as kept.
func toAir(r airquality.Reading) airReading {
	out := airReading{Category: r.Category}
	if !math.IsNaN(r.AQI) {
		v := r.AQI
		out.AQI = &v
	}
	return out
}

// fromAir is a kept reading as AirNow's.
func fromAir(r airReading) airquality.Reading {
	out := airquality.Reading{AQI: math.NaN(), Category: r.Category}
	if r.AQI != nil {
		out.AQI = *r.AQI
	}
	return out
}

// airNational keeps the national file's areas as the hour's one record.
func airNational(store *history.Store, areas []airquality.Area, hour time.Time) {
	kept := make([]airArea, 0, len(areas))
	for _, a := range areas { // the file's areas (P10-02)
		k := airArea{Name: a.Name, State: a.State, Lat: a.Lat, Lon: a.Lon, Forecast: map[int]airReading{}}
		if a.Now != nil {
			r := toAir(*a.Now)
			k.Now = &r
		}
		for d, r := range a.Forecast { // today and tomorrow (P10-02)
			k.Forecast[d] = toAir(r)
		}
		kept = append(kept, k)
	}
	doc, err := json.Marshal(kept)
	if err != nil {
		return
	}
	store.Put(airnowHourly.Name, history.Record{Key: airSeries, At: hour, IssuedAt: hour, Doc: doc})
}

// airAt is the national file as the history holds it for an hour, and
// whether it does: the map draws from it rather than fetching the file again
// (D-234).
func airAt(store *history.Store, hour time.Time) ([]airquality.Area, bool) {
	if store == nil {
		return nil, false
	}
	rec, ok := store.Get(airnowHourly.Name, airSeries, hour)
	var kept []airArea
	if !ok || json.Unmarshal(rec.Doc, &kept) != nil || len(kept) == 0 {
		return nil, false
	}
	out := make([]airquality.Area, 0, len(kept))
	for _, k := range kept { // the hour's areas (P10-02)
		a := airquality.Area{Name: k.Name, State: k.State, Lat: k.Lat, Lon: k.Lon, Forecast: map[int]airquality.Reading{}}
		if k.Now != nil {
			r := fromAir(*k.Now)
			a.Now = &r
		}
		for d, r := range k.Forecast { // today and tomorrow (P10-02)
			a.Forecast[d] = fromAir(r)
		}
		out = append(out, a)
	}
	return out, true
}

// incidentKey is an incident's series: its name, state and discovery, by
// digest.
func incidentKey(inc snapshot.Incident) history.Key {
	return history.Key{Source: "wfigs", Place: "i" + digest(inc.Name+" "+inc.State+" "+inc.Discovered.UTC().Format(time.RFC3339))}
}

// digest is sixteen hex digits of a text's SHA-256: a name in the history's
// alphabet for an identity that is not one.
func digest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:8])
}
