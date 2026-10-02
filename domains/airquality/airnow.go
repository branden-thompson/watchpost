// Package airquality is EPA AirNow's reporting areas (0.18.0 D-138): the
// official US AQI measured in about 500 areas, and AirNow's forecasts for
// today and tomorrow, from one national file - no key.
package airquality

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/bodymemo"
	"github.com/branden-thompson/watchpost/platform/httpx"
)

// Getter is the fetch the provider needs: the hardened client's.
type Getter interface {
	GetText(ctx context.Context, rawURL string, opts ...httpx.Option) ([]byte, error)
}

// Attribution is AirNow's credit, as its data use asks.
const Attribution = "U.S. EPA AirNow (preliminary data, not fully verified)"

// defaultBase is AirNow's file host, FR-3.8's closed list.
const defaultBase = "https://files.airnowtech.org"

// Host is the host the file comes from, for the Status window.
func Host() string { return strings.TrimPrefix(defaultBase, "https://") }

// fileAge is how long the file is kept: AirNow writes it about hourly.
const fileAge = 20 * time.Minute

// Provider reads the reporting areas.
type Provider struct {
	get  Getter
	base string
	// memo is the file's parse by the UTC hour it was read in (W14, P-7):
	// the file is served from the cache on every map ask, and without the memo
	// each ask parses its ~1.9 MB again. The parse reads the moment only to the
	// hour - each area's today, a whole-hour offset - so an hour's parse is its
	// own. Two entries: the hour, and the one before it at the turn.
	memo *bodymemo.Memo[int64, []Area]
}

// newAreaMemo is the memo's constructor as a value: P10's call graph matches a
// call by its bare name, and bodymemo.New called inside this package's New
// reads as New calling itself (W14).
var newAreaMemo = bodymemo.New[int64, []Area]

// Parses is how many times the file has been parsed: once an hour a body.
func (p *Provider) Parses() int {
	if p.memo == nil {
		return 0
	}
	_, n := p.memo.Stats()
	return n
}

// New builds the provider; base "" is AirNow's host.
func New(get Getter, base string) *Provider {
	if base == "" {
		base = defaultBase
	}
	return &Provider{get: get, base: base, memo: newAreaMemo(2)}
}

// Reading is an area's AQI: the number where AirNow gives one, NaN where it
// gives the category alone - as most forecasts do - and the category.
type Reading struct {
	AQI      float64
	Category string
}

// Value is the reading as an AQI: its number, or its category's floor where
// it has none, so a marker takes its category's colour either way.
func (r Reading) Value() float64 {
	if !math.IsNaN(r.AQI) {
		return r.AQI
	}
	return categoryFloor[strings.ToLower(r.Category)]
}

// categoryFloor is each category's least AQI.
var categoryFloor = map[string]float64{"good": 0, "moderate": 51, "unhealthy for sensitive groups": 101,
	"unhealthy": 151, "very unhealthy": 201, "hazardous": 301}

// Area is one reporting area: where it is, its latest measured AQI, and
// AirNow's forecast by days from the area's today (0 today, 1 tomorrow).
type Area struct {
	Name, State string
	Lat, Lon    float64
	Now         *Reading
	Forecast    map[int]Reading
	issued      map[int]time.Time // each day's forecast's issue, the latest kept
}

// Areas reads the national file. Each area's primary pollutant's rows alone
// are its AQI: the area's AQI is its worst pollutant's.
func (p *Provider) Areas(ctx context.Context, now time.Time) ([]Area, error) {
	body, err := p.get.GetText(ctx, p.base+"/airnow/today/reportingarea.dat", httpx.TTL(fileAge))
	if err != nil {
		return nil, fmt.Errorf("AirNow: %w", err)
	}
	parseAt := func(b []byte) ([]Area, error) { return parse(string(b), now), nil }
	if p.memo == nil {
		return parseAt(body)
	}
	return p.memo.Parsed(now.UTC().Truncate(time.Hour).Unix(), body, parseAt) // read-only to its callers: shared by the hour's asks
}

// zoneHours are the file's zone words as hours from UTC: an area's today is
// its own.
var zoneHours = map[string]int{"EDT": -4, "EST": -5, "CDT": -5, "CST": -6, "MDT": -6, "MST": -7, "PDT": -7, "PST": -8,
	"ADT": -8, "AKDT": -8, "AKST": -9, "HST": -10, "AST": -4, "SST": -11, "CHST": 10}

// dayOf is how many days a valid date lies after the area's today; false
// when the date does not read.
func dayOf(valid, zone string, now time.Time) (int, bool) {
	d, err := time.Parse("01/02/06", valid)
	if err != nil {
		return 0, false
	}
	h, ok := zoneHours[zone]
	if !ok {
		h = -5
	}
	local := now.UTC().Add(time.Duration(h) * time.Hour)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	return int(d.Sub(today).Hours() / 24), true
}

// parse reads the file: pipe-separated, one pollutant of one area a row -
// issued, valid date, valid time, zone, day offset, O(bserved), F(orecast) or
// Y(esterday), primary, area, state, latitude, longitude, pollutant, AQI,
// category, action day, discussion, agency. A FORECAST'S OFFSET COUNTS FROM
// ITS ISSUE: yesterday's "tomorrow" is today, and the file holds both issues,
// so a forecast is placed by its valid date against the area's today, the
// latest issue kept.
func parse(body string, now time.Time) []Area {
	var out []Area
	at := map[string]int{}
	for _, line := range strings.Split(body, "\n") {
		f := strings.Split(strings.TrimRight(line, "\r"), "|")
		if len(f) < 14 || f[6] != "Y" || (f[5] != "O" && f[5] != "F") {
			continue
		}
		lat, err1 := strconv.ParseFloat(f[9], 64)
		lon, err2 := strconv.ParseFloat(f[10], 64)
		if err1 != nil || err2 != nil {
			continue
		}
		key := f[7] + "|" + f[8]
		i, ok := at[key]
		if !ok {
			i, at[key] = len(out), len(out)
			out = append(out, Area{Name: f[7], State: f[8], Lat: lat, Lon: lon, Forecast: map[int]Reading{}, issued: map[int]time.Time{}})
		}
		r := Reading{AQI: math.NaN(), Category: f[13]}
		if v, err := strconv.ParseFloat(f[12], 64); err == nil && v >= 0 {
			r.AQI = v
		}
		if f[5] == "O" {
			out[i].Now = &r
			continue
		}
		day, ok := dayOf(f[1], f[3], now)
		issued, err := time.Parse("01/02/06", f[0])
		if !ok || err != nil || day < 0 || issued.Before(out[i].issued[day]) {
			continue
		}
		out[i].Forecast[day], out[i].issued[day] = r, issued
	}
	return out
}
