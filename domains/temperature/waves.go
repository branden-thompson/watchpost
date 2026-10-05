package temperature

// waves.go — the map's wave height (0.18.0 D-125, D-126): NDFD's
// significant wave height where it reaches - the NWS's coastal numbers -
// and Open-Meteo Marine's beyond it and on the hours and days it lacks. The
// sea's alone: a point ashore has none from either.

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/units"
)

// OpenMeteoWavesCredit is the waves' credit line where Open-Meteo drew them.
const OpenMeteoWavesCredit = "Waves: NWS NDFD near shore, Open-Meteo.com (CC BY 4.0) beyond, interpolated"

// Measure is what a source answered of one measure over a lattice: Hourly by
// hour then point; Max each day's highest, by day from today then point, on
// the point's local date. Missing is NaN - ashore, or past a source's reach.
type Measure struct {
	Lattice Lattice
	Hours   []time.Time // on the hour, UTC, oldest first
	Hourly  [][]float64
	Max     [Days][]float64
}

// Waves is the wave height, in metres (D-125).
type Waves = Measure

// newWaves is an answer with every value missing.
func newWaves(l Lattice) Measure {
	w := Measure{Lattice: l}
	for k := range Days {
		w.Max[k] = missing(l.Cols * l.Rows)
	}
	return w
}

// hourIndex is the index of an hour, adding it in order.
func (w *Measure) hourIndex(t time.Time) int {
	return hourRow(&w.Hours, &w.Hourly, w.Lattice.Cols*w.Lattice.Rows, t)
}

// SetHour puts an hour's values in, where the source gave none for it -
// the history's (D-194); false where the source has the hour or the values
// do not cover the lattice.
func (w *Measure) SetHour(t time.Time, vals []float64) bool {
	n := w.Lattice.Cols * w.Lattice.Rows
	if len(vals) != n {
		return false
	}
	row := w.Hourly[w.hourIndex(t.UTC().Truncate(time.Hour))]
	for _, v := range row { // a lattice's points (P10-02)
		if !math.IsNaN(v) {
			return false
		}
	}
	copy(row, vals)
	return true
}

// At is the values of the newest hour at or before t, within an hour of it:
// a frame never shows an hour it is not in.
func (w Measure) At(t time.Time) ([]float64, bool) {
	for i := len(w.Hours) - 1; i >= 0; i-- {
		if !w.Hours[i].After(t) {
			if t.Sub(w.Hours[i]) >= time.Hour {
				return nil, false
			}
			return w.Hourly[i], true
		}
	}
	return nil, false
}

// FilledFrom is the waves with every missing value, hour by hour and day by
// day, taken from another source's (D-125): NDFD where it reaches,
// Open-Meteo beyond.
func (w Measure) FilledFrom(fill Measure) Measure {
	out := newWaves(w.Lattice)
	for i, h := range w.Hours {
		copy(out.Hourly[out.hourIndex(h)], w.Hourly[i])
	}
	for i, h := range fill.Hours {
		row := out.Hourly[out.hourIndex(h)]
		for p, v := range fill.Hourly[i] {
			if math.IsNaN(row[p]) {
				row[p] = v
			}
		}
	}
	for k := range Days {
		for p := range out.Max[k] {
			out.Max[k][p] = w.Max[k][p]
			if math.IsNaN(out.Max[k][p]) {
				out.Max[k][p] = fill.Max[k][p]
			}
		}
	}
	return out
}

// Waves asks NDFD for the lattice's significant wave height: hourly from the
// next hour, every day's highest worked out on the local date.
func (s *NDFD) Waves(ctx context.Context, l Lattice, now time.Time) (Waves, error) {
	if err := noLattice("NDFD waves", l); err != nil {
		return Waves{}, err
	}
	var list []string
	for _, p := range l.Points() {
		list = append(list, ftoa(p.Lat)+","+ftoa(p.Lon))
	}
	q := url.Values{"listLatLon": {strings.Join(list, " ")}, "product": {"time-series"}, "waveh": {"waveh"}}
	body, err := s.get.GetText(ctx, s.base+"/xml/sample_products/browser_interface/ndfdXMLclient.php?"+q.Encode(), httpx.TTL(untilNextHour(now)))
	if err != nil {
		return Waves{}, fmt.Errorf("NDFD waves: %w", err)
	}
	var doc dwml
	if err := xml.Unmarshal(body, &doc); err != nil {
		return Waves{}, fmt.Errorf("NDFD waves: %w", err)
	}
	out := newWaves(l)
	index, layouts := doc.pointIndex(l), doc.layouts(now)
	for _, p := range doc.Data.Parameters {
		at, ok := index[p.Location]
		if !ok {
			continue
		}
		for _, ws := range p.WaterState {
			for _, series := range ws.Waves {
				if series.Type != "significant" {
					continue
				}
				eachValue(series, layouts[ws.Layout].starts, func(t time.Time, v float64) {
					if series.Units == "feet" {
						v = units.MetresOfFeet(v)
					}
					if !waveBound.holds(v) {
						return
					}
					out.Hourly[out.hourIndex(t)][at] = v
					if k := dayOffset(t, now); k >= 0 && k < Days {
						if cur := out.Max[k][at]; math.IsNaN(cur) || v > cur {
							out.Max[k][at] = v
						}
					}
				})
			}
		}
	}
	return out, nil
}

// Waves asks Open-Meteo Marine, on its own host, for every point of a
// lattice: three hours back and two on, and seven days' highest, each
// point's dates its own local ones.
func (s *OpenMeteo) Waves(ctx context.Context, l Lattice, now time.Time) (Waves, error) {
	return s.WavesAt(ctx, l, nil, now)
}

// WavesAt is Waves for the lattice's points named alone - nil is every one -
// the rest missing: Open-Meteo bills every point asked, so a box's points
// NDFD reaches, and the land, are not (D-194).
func (s *OpenMeteo) WavesAt(ctx context.Context, l Lattice, only []int, now time.Time) (Waves, error) {
	all := l.Points()
	if only == nil {
		only = make([]int, len(all))
		for i := range only {
			only[i] = i
		}
	}
	if len(only) == 0 {
		return Waves{}, errors.New("Open-Meteo waves: no point asked for")
	}
	var lats, lons []string
	for _, i := range only { // the points named (P10-02)
		if i < 0 || i >= len(all) {
			return Waves{}, fmt.Errorf("Open-Meteo waves: point %d of %d", i, len(all))
		}
		lats, lons = append(lats, ftoa(all[i].Lat)), append(lons, ftoa(all[i].Lon))
	}
	q := url.Values{"latitude": {strings.Join(lats, ",")}, "longitude": {strings.Join(lons, ",")},
		"hourly": {"wave_height"}, "past_hours": {"3"}, "forecast_hours": {strconv.Itoa(hoursAhead)},
		"daily": {"wave_height_max"}, "forecast_days": {strconv.Itoa(Days)}, "timezone": {"auto"}}
	body, err := s.get.GetText(ctx, s.marine+"/v1/marine?"+q.Encode(), httpx.TTL(untilNextHour(now)))
	if err != nil {
		return Waves{}, fmt.Errorf("Open-Meteo waves: %w", err)
	}
	type point struct {
		Offset int `json:"utc_offset_seconds"`
		Hourly struct {
			Time   []string   `json:"time"`
			Height []*float64 `json:"wave_height"`
		} `json:"hourly"`
		Daily struct {
			Time []string   `json:"time"`
			Max  []*float64 `json:"wave_height_max"`
		} `json:"daily"`
	}
	var pts []point
	if err := json.Unmarshal(body, &pts); err != nil {
		var one point
		if err1 := json.Unmarshal(body, &one); err1 != nil {
			return Waves{}, fmt.Errorf("Open-Meteo waves: %w", err)
		}
		pts = []point{one}
	}
	if len(pts) != len(only) {
		return Waves{}, fmt.Errorf("Open-Meteo waves: %d points answered for %d asked", len(pts), len(only))
	}
	out := newWaves(l)
	for k, p := range pts {
		at := only[k] // the lattice point it answers for
		zone := time.FixedZone("", p.Offset)
		for i, ts := range p.Hourly.Time {
			t, err := time.ParseInLocation("2006-01-02T15:04", ts, zone)
			if err != nil || !inReach(t, now) {
				continue
			}
			set(out.Hourly[out.hourIndex(t)], at, p.Hourly.Height, i, waveBound)
		}
		for k := range min(len(p.Daily.Time), Days) {
			set(out.Max[k], at, p.Daily.Max, k, waveBound)
		}
	}
	return out, nil
}
