package temperature

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// The sources' hosts, FR-3.8's closed list (D-93, D-125).
const (
	ndfdBase      = "https://graphical.weather.gov"
	openMeteoBase = "https://api.open-meteo.com"
	marineBase    = "https://marine-api.open-meteo.com" // Open-Meteo's waves (D-125)
)

// Hosts are the sources' addresses, FR-3.8's closed list: the app names them
// in the Status window and a test holds them to the table.
func Hosts() map[string]string {
	return map[string]string{"NWS NDFD": ndfdBase, "Open-Meteo": openMeteoBase, "Open-Meteo Marine": marineBase}
}

// OpenMeteoCredit is Open-Meteo's credit line: its data is CC BY 4.0, which
// asks that a change be said - the map interpolates between its points.
const OpenMeteoCredit = "Temperature: Open-Meteo.com (CC BY 4.0), interpolated"

// NDFD is the NWS's National Digital Forecast Database, through its XML
// service: the lower 48, Alaska, Hawaii, Puerto Rico and Guam; nothing
// offshore past its grid; no hour before the current one (D-96).
type NDFD struct {
	get  Getter
	base string
}

// NewNDFD builds the source; base "" is the production host.
func NewNDFD(get Getter, base string) *NDFD {
	if base == "" {
		base = ndfdBase
	}
	return &NDFD{get: get, base: base}
}

func (s *NDFD) Name() string { return "NDFD" }

func (s *NDFD) Covers(region string) bool { return region != geo.RegionSamoa }

// Fetch asks twice: the days' highs and lows, then the hours from the current
// one. THE HOUR IS SENT WITH ITS ZONE: without it the service reads it as each
// point's own local time, and answered seven hours ahead (2026-09-26).
func (s *NDFD) Fetch(ctx context.Context, l Lattice, now time.Time) (Series, error) {
	pts := l.Points()
	var list []string
	for _, p := range pts {
		list = append(list, ftoa(p.Lat)+","+ftoa(p.Lon))
	}
	q := url.Values{"listLatLon": {strings.Join(list, " ")}, "product": {"time-series"}}
	ttl := httpx.TTL(untilNextHour(now))
	days := cloneValues(q)
	days.Set("maxt", "maxt")
	days.Set("mint", "mint")
	days.Set("wspd", "wspd") // the wind (W11): hourly for about two and a half days, whose peaks are worked out here
	days.Set("wdir", "wdir")
	days.Set("appt", "appt") // feels-like (D-119): hourly, then every few hours; its days worked out here
	out := newSeries(l)
	body, err := s.get.GetText(ctx, s.base+"/xml/sample_products/browser_interface/ndfdXMLclient.php?"+days.Encode(), ttl)
	if err != nil {
		return Series{}, fmt.Errorf("NDFD days: %w", err)
	}
	if err := parseDWML(body, now, &out); err != nil {
		return Series{}, fmt.Errorf("NDFD days: %w", err)
	}
	hour := now.UTC().Truncate(time.Hour)
	hours := cloneValues(q)
	hours.Set("temp", "temp")
	hours.Set("wspd", "wspd") // the current hour's wind: the days' answer starts at the next (W11)
	hours.Set("wdir", "wdir")
	hours.Set("appt", "appt")
	hours.Set("begin", hour.Format("2006-01-02T15:04:05Z"))
	hours.Set("end", hour.Add(time.Hour).Format("2006-01-02T15:04:05Z"))
	body, err = s.get.GetText(ctx, s.base+"/xml/sample_products/browser_interface/ndfdXMLclient.php?"+hours.Encode(), ttl)
	if err != nil {
		return Series{}, fmt.Errorf("NDFD hours: %w", err)
	}
	if err := parseDWML(body, now, &out); err != nil {
		return Series{}, fmt.Errorf("NDFD hours: %w", err)
	}
	return out, nil
}

func cloneValues(v url.Values) url.Values {
	out := url.Values{}
	for k, vs := range v {
		out[k] = append([]string(nil), vs...)
	}
	return out
}

// dwml is the parts of an NDFD answer read: the points, the time layouts and
// each point's temperatures.
type dwml struct {
	Data struct {
		Locations []struct {
			Key   string `xml:"location-key"`
			Point struct {
				Lat string `xml:"latitude,attr"`
				Lon string `xml:"longitude,attr"`
			} `xml:"point"`
		} `xml:"location"`
		Layouts []struct {
			Key    string   `xml:"layout-key"`
			Starts []string `xml:"start-valid-time"`
			Ends   []string `xml:"end-valid-time"`
		} `xml:"time-layout"`
		Parameters []struct {
			Location   string       `xml:"applicable-location,attr"`
			Winds      []dwmlSeries `xml:"wind-speed"`
			Dirs       []dwmlSeries `xml:"direction"`
			WaterState []struct {
				Layout string       `xml:"time-layout,attr"`
				Waves  []dwmlSeries `xml:"waves"`
			} `xml:"water-state"` // significant wave height (D-125)
			Temperatures []struct {
				Type   string `xml:"type,attr"`
				Units  string `xml:"units,attr"`
				Layout string `xml:"time-layout,attr"`
				Values []struct {
					Nil  string `xml:"nil,attr"`
					Text string `xml:",chardata"`
				} `xml:"value"`
			} `xml:"temperature"`
		} `xml:"parameters"`
	} `xml:"data"`
}

// dwmlSeries is one of an NDFD point's element series: its kind and unit,
// its time layout, and its values.
type dwmlSeries struct {
	Type   string `xml:"type,attr"`
	Units  string `xml:"units,attr"`
	Layout string `xml:"time-layout,attr"`
	Values []struct {
		Nil  string `xml:"nil,attr"`
		Text string `xml:",chardata"`
	} `xml:"value"`
}

// parseDWML adds an NDFD answer to a series. A day's high is its date's; a
// low is the date its night ends on, the morning low, as Open-Meteo's is.
func parseDWML(body []byte, now time.Time, out *Series) error {
	var doc dwml
	if err := xml.Unmarshal(body, &doc); err != nil {
		return err
	}
	if len(doc.Data.Locations) == 0 {
		return fmt.Errorf("the answer names no point")
	}
	index, layouts := doc.pointIndex(out.Lattice), doc.layouts()
	dayOf := map[time.Time]int{} // each wind hour's local day, for the day's peak
	for _, p := range doc.Data.Parameters {
		at, ok := index[p.Location]
		if !ok {
			continue
		}
		for _, w := range p.Winds {
			if w.Type != "sustained" {
				continue
			}
			eachValue(w, layouts[w.Layout].starts, func(t time.Time, v float64) {
				if w.Units == "knots" {
					v = knotsToKmh(v)
				}
				out.WindSpeed[out.hourIndex(t)][at] = v
				dayOf[t.UTC().Truncate(time.Hour)] = dayOffset(t, now)
			})
		}
		for _, w := range p.Dirs {
			if w.Type != "wind" {
				continue
			}
			eachValue(w, layouts[w.Layout].starts, func(t time.Time, v float64) { out.WindFrom[out.hourIndex(t)][at] = v })
		}
		for _, temp := range p.Temperatures {
			lay := layouts[temp.Layout]
			for i, v := range temp.Values {
				if v.Nil == "true" || i >= len(lay.starts) || lay.starts[i].IsZero() {
					continue
				}
				f, err := strconv.ParseFloat(strings.TrimSpace(v.Text), 64)
				if err != nil {
					continue
				}
				if temp.Units == "Fahrenheit" {
					f = fahrenheitToC(f)
				}
				switch temp.Type {
				case "hourly":
					out.Hourly[out.hourIndex(lay.starts[i])][at] = f
				case "apparent":
					out.Feels[out.hourIndex(lay.starts[i])][at] = f
					dayOf[lay.starts[i].UTC().Truncate(time.Hour)] = dayOffset(lay.starts[i], now)
				case "maximum":
					if k := dayOffset(lay.starts[i], now); k >= 0 && k < Days {
						out.High[k][at] = f
					}
				case "minimum":
					if i < len(lay.ends) && !lay.ends[i].IsZero() {
						if k := dayOffset(lay.ends[i], now); k >= 0 && k < Days {
							out.Low[k][at] = f
						}
					}
				}
			}
		}
	}
	windPeaks(out, dayOf)
	feelsDays(out, dayOf)
	return nil
}

// feelsDays works out each day's feels-like high and low from the hours
// NDFD gave, on each hour's local date (D-119): NDFD has no daily apparent
// temperature.
func feelsDays(out *Series, dayOf map[time.Time]int) {
	for i, h := range out.Hours {
		k, ok := dayOf[h]
		if !ok || k < 0 || k >= Days {
			continue
		}
		for p, v := range out.Feels[i] {
			if math.IsNaN(v) {
				continue
			}
			if cur := out.FeelsHigh[k][p]; math.IsNaN(cur) || v > cur {
				out.FeelsHigh[k][p] = v
			}
			if cur := out.FeelsLow[k][p]; math.IsNaN(cur) || v < cur {
				out.FeelsLow[k][p] = v
			}
		}
	}
}

// layout is one of an answer's time layouts: when each value starts, and
// ends where it says.
type layout struct{ starts, ends []time.Time }

// pointIndex is each of the answer's points by its key, as the lattice's
// point it is - matched on the rounded coordinates that were sent.
func (doc *dwml) pointIndex(l Lattice) map[string]int {
	index := map[string]int{}
	pts := l.Points()
	for _, loc := range doc.Data.Locations {
		lat, err1 := strconv.ParseFloat(loc.Point.Lat, 64)
		lon, err2 := strconv.ParseFloat(loc.Point.Lon, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		for i, p := range pts {
			if p.Lat == round2(lat) && p.Lon == round2(lon) {
				index[loc.Key] = i
			}
		}
	}
	return index
}

// layouts are the answer's time layouts by key.
func (doc *dwml) layouts() map[string]layout {
	out := map[string]layout{}
	for _, l := range doc.Data.Layouts {
		var lay layout
		for _, s := range l.Starts {
			t, _ := time.Parse(time.RFC3339, strings.TrimSpace(s))
			lay.starts = append(lay.starts, t)
		}
		for _, e := range l.Ends {
			t, _ := time.Parse(time.RFC3339, strings.TrimSpace(e))
			lay.ends = append(lay.ends, t)
		}
		out[l.Key] = lay
	}
	return out
}

// eachValue calls f with each of a series' values that is a number, at its
// layout's time.
func eachValue(s dwmlSeries, starts []time.Time, f func(time.Time, float64)) {
	for i, v := range s.Values {
		if v.Nil == "true" || i >= len(starts) || starts[i].IsZero() {
			continue
		}
		if x, err := strconv.ParseFloat(strings.TrimSpace(v.Text), 64); err == nil {
			f(starts[i], x)
		}
	}
}

// windPeaks works out each day's peak sustained wind and the direction it
// blew from then, from the hours NDFD gave (W11): NDFD has no daily wind.
func windPeaks(out *Series, dayOf map[time.Time]int) {
	for i, h := range out.Hours {
		k, ok := dayOf[h]
		if !ok || k < 0 || k >= Days {
			continue
		}
		for p, v := range out.WindSpeed[i] {
			if math.IsNaN(v) || math.IsNaN(out.WindFrom[i][p]) {
				continue
			}
			if cur := out.PeakSpeed[k][p]; math.IsNaN(cur) || v > cur {
				out.PeakSpeed[k][p], out.PeakFrom[k][p] = v, out.WindFrom[i][p]
			}
		}
	}
}

// OpenMeteo is Open-Meteo's forecast API: everywhere, over water too, with
// the past hours NDFD lacks - Radar mode's source always (D-96).
type OpenMeteo struct {
	get    Getter
	base   string
	marine string // the marine API's host, for the waves (D-125)
}

// NewOpenMeteo builds the source; base "" is the production host.
func NewOpenMeteo(get Getter, base string) *OpenMeteo {
	marine := marineBase
	if base == "" {
		base = openMeteoBase
	} else {
		marine = base // a test's one server answers both
	}
	return &OpenMeteo{get: get, base: base, marine: marine}
}

func (s *OpenMeteo) Name() string { return "Open-Meteo" }

func (s *OpenMeteo) Covers(string) bool { return true }

// Fetch asks once: three hours back and two on, and seven days, each point's
// dates its own local ones.
func (s *OpenMeteo) Fetch(ctx context.Context, l Lattice, now time.Time) (Series, error) {
	var lats, lons []string
	for _, p := range l.Points() {
		lats, lons = append(lats, ftoa(p.Lat)), append(lons, ftoa(p.Lon))
	}
	q := url.Values{"latitude": {strings.Join(lats, ",")}, "longitude": {strings.Join(lons, ",")},
		"hourly": {"temperature_2m,wind_speed_10m,wind_direction_10m,apparent_temperature"}, "past_hours": {"3"}, "forecast_hours": {"2"},
		"daily": {"temperature_2m_max,temperature_2m_min,wind_speed_10m_max,wind_direction_10m_dominant,apparent_temperature_max,apparent_temperature_min"}, "forecast_days": {strconv.Itoa(Days)}, "timezone": {"auto"}}
	body, err := s.get.GetText(ctx, s.base+"/v1/forecast?"+q.Encode(), httpx.TTL(untilNextHour(now)))
	if err != nil {
		return Series{}, fmt.Errorf("Open-Meteo: %w", err)
	}
	out := newSeries(l)
	if err := parseOpenMeteo(body, &out); err != nil {
		return Series{}, fmt.Errorf("Open-Meteo: %w", err)
	}
	return out, nil
}

// openMeteoPoint is one point's answer.
type openMeteoPoint struct {
	Offset int `json:"utc_offset_seconds"`
	Hourly struct {
		Time      []string   `json:"time"`
		Temp      []*float64 `json:"temperature_2m"`
		WindSpeed []*float64 `json:"wind_speed_10m"`
		WindFrom  []*float64 `json:"wind_direction_10m"`
		Feels     []*float64 `json:"apparent_temperature"`
	} `json:"hourly"`
	Daily struct {
		Time     []string   `json:"time"`
		Max      []*float64 `json:"temperature_2m_max"`
		Min      []*float64 `json:"temperature_2m_min"`
		WindMax  []*float64 `json:"wind_speed_10m_max"`
		WindFrom []*float64 `json:"wind_direction_10m_dominant"`
		FeelsMax []*float64 `json:"apparent_temperature_max"`
		FeelsMin []*float64 `json:"apparent_temperature_min"`
	} `json:"daily"`
}

// parseOpenMeteo adds an answer to a series: a list, one a point in the
// order asked; one point alone is answered as an object.
func parseOpenMeteo(body []byte, out *Series) error {
	var pts []openMeteoPoint
	if err := json.Unmarshal(body, &pts); err != nil {
		var one openMeteoPoint
		if err1 := json.Unmarshal(body, &one); err1 != nil {
			return err
		}
		pts = []openMeteoPoint{one}
	}
	if n := out.Lattice.Cols * out.Lattice.Rows; len(pts) != n {
		return fmt.Errorf("%d points answered for %d asked", len(pts), n)
	}
	for at, p := range pts {
		zone := time.FixedZone("", p.Offset)
		for i, ts := range p.Hourly.Time {
			t, err := time.ParseInLocation("2006-01-02T15:04", ts, zone)
			if err != nil {
				continue
			}
			h := out.hourIndex(t)
			set(out.Hourly[h], at, p.Hourly.Temp, i)
			set(out.WindSpeed[h], at, p.Hourly.WindSpeed, i)
			set(out.WindFrom[h], at, p.Hourly.WindFrom, i)
			set(out.Feels[h], at, p.Hourly.Feels, i)
		}
		for k := range min(len(p.Daily.Time), Days) {
			set(out.High[k], at, p.Daily.Max, k)
			set(out.Low[k], at, p.Daily.Min, k)
			set(out.PeakSpeed[k], at, p.Daily.WindMax, k)
			set(out.PeakFrom[k], at, p.Daily.WindFrom, k)
			set(out.FeelsHigh[k], at, p.Daily.FeelsMax, k)
			set(out.FeelsLow[k], at, p.Daily.FeelsMin, k)
		}
	}
	return nil
}

// set puts one answered value in place; a value not answered stays missing.
func set(row []float64, at int, vals []*float64, i int) {
	if i < len(vals) && vals[i] != nil {
		row[at] = *vals[i]
	}
}

func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
