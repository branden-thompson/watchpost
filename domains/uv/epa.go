// Package uv is the US EPA's UV index forecast (W18.4, D-167): hour by hour
// for today, one city (or ZIP) at a time, from Envirofacts - no key. The map
// draws it as markers on a cold start, while Open-Meteo refuses and nothing
// is yet recorded: the one UV source that is not a lattice.
package uv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/httpx"
)

// Getter is the fetch the provider needs: the hardened client's.
type Getter interface {
	GetText(ctx context.Context, rawURL string, opts ...httpx.Option) ([]byte, error)
}

// Attribution is the EPA's credit.
const Attribution = "UV index: U.S. EPA (Envirofacts), forecast for today"

// Host is the host the forecast comes from - FR-3.8's closed list, and the
// Status window's.
const Host = "data.epa.gov"

// defaultBase is Envirofacts' address.
const defaultBase = "https://" + Host

// forecastAge is how long an answer is kept: the forecast is made once a day
// and read hour by hour.
const forecastAge = time.Hour

// Reading is one hour's UV index.
type Reading struct {
	At    time.Time // the hour's start, in the city's zone
	Index float64
}

// EPA reads the hourly UV forecast.
type EPA struct {
	get  Getter
	base string
}

// NewEPA builds the provider; base "" is the production host.
func NewEPA(get Getter, base string) *EPA {
	if base == "" {
		base = defaultBase
	}
	return &EPA{get: get, base: base}
}

// hourLayout is how the answer writes an hour: "Sep/30/2026 04 AM", in the
// city's local time, no zone said.
const hourLayout = "Jan/02/2006 03 PM"

// maxRows is the most hours an answer may hold: a day's, twice over - EPA's
// evening hours run past midnight. More is no day's forecast.
const maxRows = 48

// maxIndex is past any UV index measured at the surface (the highest, on the
// Altiplano, near 43): a value beyond it, or below zero, is not a reading.
const maxIndex = 50

// twoLetters reports whether s is two letters A to Z: a state as EPA's
// address takes it, and nothing that could change the address's path (IS-M7).
func twoLetters(s string) bool {
	return len(s) == 2 && s[0] >= 'A' && s[0] <= 'Z' && s[1] >= 'A' && s[1] <= 'Z'
}

// Hourly is a city's UV index for today, hour by hour, its hours read in loc -
// the city's zone, which the answer does not say.
func (e *EPA) Hourly(ctx context.Context, city, state string, loc *time.Location) ([]Reading, error) {
	if e == nil || e.get == nil {
		return nil, errors.New("uv: no EPA source")
	}
	state = strings.ToUpper(state)
	if city == "" || !twoLetters(state) || loc == nil {
		return nil, fmt.Errorf("uv: a city, a two-letter state and a zone are needed, not %q, %q", city, state)
	}
	name := city
	if city == "New York City" {
		name = "New York" // EPA's name for it: asked as GeoNames names it, EPA answers an error (UAT-2 U2-51)
	}
	u := e.base + "/efservice/getEnvirofactsUVHOURLY/CITY/" + url.PathEscape(strings.ToUpper(name)) + "/STATE/" + url.PathEscape(state) + "/JSON"
	body, err := e.get.GetText(ctx, u, httpx.TTL(forecastAge))
	if err != nil {
		return nil, fmt.Errorf("uv: EPA: %w", err)
	}
	var rows []struct {
		DateTime string  `json:"DATE_TIME"`
		Value    float64 `json:"UV_VALUE"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("uv: EPA's answer is not its list: %w", err)
	}
	if len(rows) == 0 {
		return nil, errors.New("uv: EPA has no forecast for " + city + ", " + state)
	}
	if len(rows) > maxRows {
		return nil, fmt.Errorf("uv: EPA answered %d hours for one day", len(rows))
	}
	out := make([]Reading, 0, len(rows))
	for _, r := range rows { // a day's hours (P10-02)
		at, err := time.ParseInLocation(hourLayout, r.DateTime, loc)
		if err != nil {
			return nil, fmt.Errorf("uv: EPA's hour %q: %w", r.DateTime, err)
		}
		if r.Value < 0 || r.Value > maxIndex {
			return nil, fmt.Errorf("uv: EPA's index %v at %q is no UV index", r.Value, r.DateTime)
		}
		// NOT IN ORDER, AND NOT TO BE PUT IN IT: EPA's answer dates its evening
		// hours the day before (a Sep/30 forecast runs 04 AM..04 PM Sep/30, then
		// 05 PM..11 PM Sep/29) - hours whose index is 0 or 1, left unmatched
		// rather than guessed at. What is refused is an hour a day or more from
		// the first: that is not today's forecast.
		if n := len(out); n > 0 && (at.Sub(out[0].At) >= 24*time.Hour || out[0].At.Sub(at) >= 24*time.Hour) {
			return nil, fmt.Errorf("uv: EPA's hour %q is not the forecast's day", r.DateTime)
		}
		out = append(out, Reading{At: at, Index: r.Value})
	}
	return out, nil
}

// At is the reading for the hour t falls in; false where the forecast has none.
func At(readings []Reading, t time.Time) (float64, bool) {
	if len(readings) == 0 || t.IsZero() {
		return 0, false
	}
	for _, r := range readings { // a day's hours (P10-02)
		if !t.Before(r.At) && t.Before(r.At.Add(time.Hour)) {
			return r.Index, true
		}
	}
	return 0, false
}
