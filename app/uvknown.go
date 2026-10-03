package app

// uvknown.go — EPA's UV city readings kept in the history (D-224), and the
// cities it knows drawn when they are in view again the same day (D-225).

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/domains/locations/geodata"
	"github.com/branden-thompson/watchpost/domains/uv"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/history"
	"github.com/branden-thompson/watchpost/platform/tz"
)

// uvCityDoc is what a city's record says of the city: enough for a reader -
// this map in another session, or an analyst - to know which city the series
// is and to read its hours in the city's own day.
type uvCityDoc struct {
	Name  string `json:"name"`
	State string `json:"state"`
	TZ    string `json:"tz"`
}

// knownUVCities are the cities the history holds EPA readings for, read from
// its series at first use and again each hour - another instance may have
// read more - and added to as this one records.
type knownUVCities struct {
	mu     sync.Mutex
	listed time.Time // when the history's series were last read
	cities map[history.Key]geodata.City
}

// knownUVCitiesMost is the most series the history's list is read for: far
// more cities than a session's views pass over.
const knownUVCitiesMost = 4096

// knownUVListEvery is how often the history's series are read again.
const knownUVListEvery = time.Hour

// uvCityKey is a city's series: EPA's, named by the city and its state in the
// history's alphabet.
func uvCityKey(c geodata.City) history.Key {
	name := c.ASCII
	if name == "" {
		name = c.Name
	}
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name + " " + c.State) { // a city's name (P10-02)
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
		place = place[:64]
	}
	return history.Key{Source: "epa", Place: place}
}

// record keeps a city's hours in the history, an hour a record, and knows the
// city from then on.
func (k *knownUVCities) record(store *history.Store, c geodata.City, readings []uv.Reading) {
	if store == nil || len(readings) == 0 {
		return
	}
	doc, err := json.Marshal(uvCityDoc{Name: c.Name, State: c.State, TZ: c.TZ})
	if err != nil {
		return
	}
	key, shape := uvCityKey(c), history.Shape{Box: geo.Box{W: c.Lon, S: c.Lat, E: c.Lon, N: c.Lat}, Cols: 1, Rows: 1}
	for _, r := range readings { // a day's hours (P10-02)
		store.Put(epaUVCities.Name, history.Record{Key: key, At: r.At, IssuedAt: r.At, Shape: shape, Values: map[string][]float64{"uv": {r.Index}}, Doc: doc})
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.cities == nil {
		k.cities = map[history.Key]geodata.City{}
	}
	k.cities[key] = c
}

// inView are the known cities in view with readings for their day at anchor,
// beside the cities drawn: none within the spacing of a city drawn or of one
// another (D-202, D-225), the history read for each city's day.
func (k *knownUVCities) inView(store *history.Store, view geo.Box, n int, anchor time.Time, drawn []cityReadings) []cityReadings {
	_, _, spacing, ok := uvCells(view, n)
	if store == nil || !ok {
		return nil
	}
	taken := make([]geodata.City, 0, len(drawn))
	for _, d := range drawn { // the cities asked (P10-02)
		taken = append(taken, d.city)
	}
	near := func(c geodata.City) bool {
		for _, o := range taken { // the cities drawn so far (P10-02)
			if geo.HaversineKM(c.Lat, c.Lon, o.Lat, o.Lon) < spacing {
				return true
			}
		}
		return false
	}
	var out []cityReadings
	for _, kc := range k.candidates(store, view, anchor) { // the known cities in view (P10-02)
		if near(kc.city) {
			continue
		}
		if r := dayReadings(store, kc.key, kc.city, anchor); len(r) > 0 {
			out = append(out, cityReadings{kc.city, r})
			taken = append(taken, kc.city)
		}
	}
	return out
}

// knownCity is a known city and its series.
type knownCity struct {
	key  history.Key
	city geodata.City
}

// candidates are the known cities in view, in their series' order so every
// ask draws the same ones.
func (k *knownUVCities) candidates(store *history.Store, view geo.Box, now time.Time) []knownCity {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.listed.IsZero() || now.Sub(k.listed) >= knownUVListEvery || now.Before(k.listed) {
		k.listLocked(store, now)
	}
	var out []knownCity
	for key, c := range k.cities { // the cities known (P10-02)
		if view.Contains(c.Lat, c.Lon) {
			out = append(out, knownCity{key, c})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key.Place < out[j].key.Place })
	return out
}

// listLocked reads the history's series into the known cities: each city from
// its newest record's point and document.
func (k *knownUVCities) listLocked(store *history.Store, now time.Time) {
	k.listed = now
	if k.cities == nil {
		k.cities = map[history.Key]geodata.City{}
	}
	for _, key := range store.Series(epaUVCities.Name, knownUVCitiesMost) { // at most knownUVCitiesMost (P10-02)
		if _, ok := k.cities[key]; ok {
			continue
		}
		rec, ok := store.Latest(epaUVCities.Name, key, now, epaUVCities.Hours)
		var doc uvCityDoc
		if !ok || json.Unmarshal(rec.Doc, &doc) != nil || doc.TZ == "" {
			continue
		}
		k.cities[key] = geodata.City{Name: doc.Name, ASCII: doc.Name, State: doc.State, Lat: rec.Shape.Box.S, Lon: rec.Shape.Box.W, TZ: doc.TZ}
	}
}

// dayReadings are a city's recorded hours for its own day at anchor: never
// another day's.
func dayReadings(store *history.Store, key history.Key, c geodata.City, anchor time.Time) []uv.Reading {
	loc, err := tz.Location(c.TZ)
	if err != nil {
		return nil
	}
	local := anchor.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	var out []uv.Reading
	for _, rec := range store.Range(epaUVCities.Name, key, start, start.AddDate(0, 0, 1).Add(-time.Nanosecond), 25) { // a day's hours (P10-02)
		if v := rec.Values["uv"]; len(v) == 1 && v[0] == v[0] {
			out = append(out, uv.Reading{At: rec.At.In(loc), Index: v[0]})
		}
	}
	return out
}
