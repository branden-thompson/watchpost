package temperature

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// The two sources' hosts, FR-3.8's closed list (D-93).
const (
	ndfdBase      = "https://graphical.weather.gov"
	openMeteoBase = "https://api.open-meteo.com"
)

// Hosts are the sources' addresses, FR-3.8's closed list: the app names them
// in the Status window and a test holds them to the table.
func Hosts() map[string]string {
	return map[string]string{"NWS NDFD": ndfdBase, "Open-Meteo": openMeteoBase}
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
			Location     string `xml:"applicable-location,attr"`
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
	index := map[string]int{}
	pts := out.Lattice.Points()
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
	type layout struct{ starts, ends []time.Time }
	layouts := map[string]layout{}
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
		layouts[l.Key] = lay
	}
	for _, p := range doc.Data.Parameters {
		at, ok := index[p.Location]
		if !ok {
			continue
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
	return nil
}

// OpenMeteo is Open-Meteo's forecast API: everywhere, over water too, with
// the past hours NDFD lacks - Radar mode's source always (D-96).
type OpenMeteo struct {
	get  Getter
	base string
}

// NewOpenMeteo builds the source; base "" is the production host.
func NewOpenMeteo(get Getter, base string) *OpenMeteo {
	if base == "" {
		base = openMeteoBase
	}
	return &OpenMeteo{get: get, base: base}
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
		"hourly": {"temperature_2m"}, "past_hours": {"3"}, "forecast_hours": {"2"},
		"daily": {"temperature_2m_max,temperature_2m_min"}, "forecast_days": {strconv.Itoa(Days)}, "timezone": {"auto"}}
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
		Time []string   `json:"time"`
		Temp []*float64 `json:"temperature_2m"`
	} `json:"hourly"`
	Daily struct {
		Time []string   `json:"time"`
		Max  []*float64 `json:"temperature_2m_max"`
		Min  []*float64 `json:"temperature_2m_min"`
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
			if err != nil || i >= len(p.Hourly.Temp) || p.Hourly.Temp[i] == nil {
				continue
			}
			out.Hourly[out.hourIndex(t)][at] = *p.Hourly.Temp[i]
		}
		for k := range min(len(p.Daily.Time), Days) {
			if k < len(p.Daily.Max) && p.Daily.Max[k] != nil {
				out.High[k][at] = *p.Daily.Max[k]
			}
			if k < len(p.Daily.Min) && p.Daily.Min[k] != nil {
				out.Low[k][at] = *p.Daily.Min[k]
			}
		}
	}
	return nil
}

func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
