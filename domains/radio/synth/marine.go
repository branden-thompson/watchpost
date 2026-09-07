package synth

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// The maritime report (FR-6): the coastal-waters forecast, the nearest buoy,
// the tides and the currents, read between the location forecast and the fire
// report (MVS-D-18).
//
// The words come from the SAME helpers the Details view uses (platform/render's
// SeaState, TideTrend, NextTide, CurrentPhase — lifted there at Task 3.1), so
// the screen and the ear can never describe the same sea differently (RS-11).

// MarineReport is one location's coastal block, as the deck hands it over.
type MarineReport struct {
	Known    bool
	State    snapshot.Marine
	TZ       *time.Location // the location's zone: tide times are read in local 12-hour form (MVS-D-21)
	Lat, Lon float64
	// Forecast is the office's coastal-waters forecast, already cut to this
	// location's zone and to its first periods (Task 3.5). Prose from the
	// network, so it is split into sentences before it is spoken.
	Forecast string
}

// maxMaritimePieces bounds the whole section.
//
// The forecast is PROSE FROM THE NETWORK and a product can be long. A report is
// a fixed part of a cycle, not an open-ended broadcast: without a bound one
// unusual product could hold the air for minutes, and a listener waiting for
// the fire report would simply never reach it.
const maxMaritimePieces = 12

// MarineSegments builds the spoken maritime report, or nothing when this
// location has no coastal data (an inland location skips the report entirely —
// it is not "no data available", it is not applicable).
func (c Composer) MarineSegments(location string, mr MarineReport, imperial bool, now time.Time) []Segment {
	if !mr.Known {
		return nil
	}
	place := ExpandStates(location)
	notice := c.say("marine-report", "head", map[string]string{"Location": place, "Sources": marineSources(mr.State)})
	segs := []Segment{{Key: "maritime:notice:" + contentKey(notice), Text: notice, Role: cast.Maritime, Pause: reportPause}}

	var body []string
	if mr.Forecast != "" {
		// Through the same sentence splitter every other product uses: the
		// listener's [r] repeats a SENTENCE, not a minute of text, and a
		// hand-over can land on a sentence boundary rather than mid-clause.
		body = append(body, Segments([]string{ExpandStates(c.say("marine-report", "forecast", map[string]string{"Text": mr.Forecast}))})...)
	}
	body = append(body, c.observationSentences(mr, imperial, now)...)
	body = append(body, c.tideSentences(mr, imperial, now)...)
	body = append(body, c.say("marine-report", "link", nil)) // no trailing slash: it would read an extra "slash" (HUM LEAD UAT)

	for _, piece := range body {
		if piece == "" {
			continue // a phrase without data is not spoken
		}
		if len(segs) >= maxMaritimePieces {
			break
		}
		segs = append(segs, Segment{Key: "maritime:" + contentKey(piece), Text: piece, Role: cast.Maritime})
	}
	return segs
}

// observationSentences are what the buoy reports: provenance, the sea, the
// water and the wind. Kept out of MarineSegments so neither function carries
// more decisions than the safety gate allows (P10-04).
func (c Composer) observationSentences(mr MarineReport, imperial bool, now time.Time) []string {
	m := mr.State
	var out []string
	// The provenance guard is exactly the screen's: an observation with no
	// buoy and no time is not "0 miles offshore, observed just now".
	if m.Buoy != "" && !m.ObservedAt.IsZero() {
		out = append(out, c.say("marine-report", "observed", map[string]string{
			"Distance": distanceWords(deref(m.BuoyDistanceKM), imperial),
			"Ago":      durationWords(now.Sub(m.ObservedAt)),
		}))
	}
	if h := render.FirstOf(m.WaveHeight, m.SwellHeight, m.WindWaveHeight); h != nil {
		out = append(out, c.say("marine-report", "sea", map[string]string{
			"State":  strings.ToLower(render.SeaState(*h)),
			"Height": heightWords(h, imperial),
			"Swell":  c.swellPhrase(m, imperial),
		}))
	}
	if m.WaterTemp != nil {
		out = append(out, c.say("marine-report", "water", map[string]string{"Temp": tempWords(m.WaterTemp, imperial)}))
	}
	if m.WindSpeed != nil {
		out = append(out, c.say("marine-report", "wind", map[string]string{
			"Speed": windWords(m.WindSpeed, imperial),
			"Gust":  gustPhrase(m.WindGust, imperial),
		}))
	}
	return out
}

// swellPhrase folds the swell into the sea sentence rather than giving it one
// of its own: the report is already the longest in the cycle (~10 sentences),
// and the two facts belong together to a listener.
func (c Composer) swellPhrase(m snapshot.Marine, imperial bool) string {
	if m.SwellHeight == nil {
		return ""
	}
	phrase := "a primary swell of " + heightWords(m.SwellHeight, imperial)
	if m.SwellDirDeg != nil {
		phrase = "a primary swell from the " + bearingLong(compassPoint(*m.SwellDirDeg)) + " at " + heightWords(m.SwellHeight, imperial)
	}
	if m.WavePeriod != nil {
		phrase += fmt.Sprintf(", with a dominant period of %.0f seconds", *m.WavePeriod)
	}
	return phrase
}

func gustPhrase(gust *float64, imperial bool) string {
	if gust == nil {
		return ""
	}
	return windWords(gust, imperial)
}

// The SPOKEN forms of the marine numbers.
//
// platform/render's Opts formats for a COLUMN — " 3.0 ft", "74°F", " 1.4 kt" —
// which is right on screen and wrong in the ear: a voice reading "ft" says
// "eff tee", and a degree symbol is anybody's guess. RS-11's "one owner" is
// about the WORDS (is this sea rough? is the tide rising?), which do come from
// render; the number's register is the surface's own business.

// heightWords is "3 feet" / "1.5 metres" — one decimal only when it matters.
func heightWords(m *float64, imperial bool) string {
	if m == nil {
		return ""
	}
	if imperial {
		return decimalWords(*m*3.28084, "foot", "feet")
	}
	return decimalWords(*m, "metre", "metres")
}

// tempWords is "74 degrees" — the unit is the listener's and is not spoken,
// exactly as the location forecast reads temperatures.
func tempWords(c *float64, imperial bool) string {
	if c == nil {
		return ""
	}
	v := *c
	if imperial {
		v = v*9/5 + 32
	}
	return fmt.Sprintf("%.0f degrees", v)
}

// windWords is "13 miles per hour" / "21 kilometres per hour", spelled out
// because "mph" reads as three letters.
func windWords(mps *float64, imperial bool) string {
	if mps == nil {
		return ""
	}
	if imperial {
		return fmt.Sprintf("%.0f miles per hour", *mps*2.23694)
	}
	return fmt.Sprintf("%.0f kilometers per hour", *mps*3.6)
}

// knotWords is "1.4 knots". Currents are ALWAYS knots (MVS-D-21) — the
// mariner's convention, whatever the listener's unit.
func knotWords(mps float64) string { return decimalWords(mps*1.94384, "knot", "knots") }

// decimalWords reads a number with one decimal only when the fraction matters,
// so "3.0 feet" is spoken as "3 feet" and "0.6 metres" keeps its decimal.
func decimalWords(v float64, singular, plural string) string {
	unit := plural
	if math.Abs(v-1) < 0.05 {
		unit = singular
	}
	if math.Abs(v-math.Round(v)) < 0.05 {
		return fmt.Sprintf("%.0f %s", v, unit)
	}
	return fmt.Sprintf("%.1f %s", v, unit)
}

// tideSentences are the tide in force, the next high and low, and the current —
// or the absence line when this location has neither prediction.
func (c Composer) tideSentences(mr MarineReport, imperial bool, now time.Time) []string {
	m := mr.State
	if len(m.Tides) == 0 && len(m.Currents) == 0 {
		return []string{c.say("marine-report", "absence", nil)}
	}
	var out []string
	high, low := render.NextTide(m.Tides, "H", now), render.NextTide(m.Tides, "L", now)
	if trend := render.TideTrend(high, low); trend != "" && m.TideLevel != nil {
		out = append(out, c.say("marine-report", "tide", map[string]string{
			"Trend": strings.ToLower(trend), "Level": heightWords(m.TideLevel, imperial),
			"Station": stationName(m.TideStation), "Distance": distanceWords(deref(m.TideStationKM), imperial),
		}))
	}
	if high != nil || low != nil {
		out = append(out, c.say("marine-report", "tide-next", map[string]string{
			"High": tideWords(high, imperial, mr.TZ), "Low": tideWords(low, imperial, mr.TZ),
		}))
	}
	if phase, next := render.CurrentPhase(m.Currents, now); phase != nil || next != nil {
		out = append(out, c.say("marine-report", "current", map[string]string{
			"Phase": currentPhaseWords(phase), "Next": currentNextWords(next, mr.TZ),
		}))
	}
	return out
}

// tideWords is "7:40 PM at 5.7 feet" — a 12-hour clock in the LOCATION's zone,
// not the listener's (MVS-D-21): a tide happens where the water is.
func tideWords(e *snapshot.TideEvent, imperial bool, tz *time.Location) string {
	if e == nil {
		return ""
	}
	when := e.Time
	if tz != nil {
		when = when.In(tz)
	}
	return when.Format("3:04 PM") + " at " + heightWords(&e.Height, imperial)
}

// currentPhaseWords is "flooding at 1.4 knots", or "slack" when the current in
// force is slack water or unknown. Currents are ALWAYS in knots, whatever the
// listener's unit (MVS-D-21) — that is the mariner's convention.
func currentPhaseWords(e *snapshot.CurrentEvent) string {
	if e == nil || e.Type == "slack" {
		return "slack"
	}
	return currentVerb(e.Type) + " at " + knotWords(e.Speed)
}

func currentNextWords(e *snapshot.CurrentEvent, tz *time.Location) string {
	if e == nil {
		return ""
	}
	when := e.Time
	if tz != nil {
		when = when.In(tz)
	}
	if e.Type == "slack" {
		return "slack water at " + when.Format("3:04 PM")
	}
	return currentVerb(e.Type) + " at " + when.Format("3:04 PM")
}

func currentVerb(typ string) string {
	switch typ {
	case "flood":
		return "flooding"
	case "ebb":
		return "ebbing"
	}
	return "running"
}

// marineSources names the providers that answered, so the notice is honest
// about where the numbers came from.
func marineSources(m snapshot.Marine) string {
	var have []string
	if m.Buoy != "" {
		have = append(have, "the National Data Buoy Center")
	}
	if len(m.Tides) > 0 || len(m.Currents) > 0 || m.TideStation != "" {
		have = append(have, "NOAA Tides and Currents")
	}
	switch len(have) {
	case 0:
		return "the National Weather Service"
	case 1:
		return have[0]
	}
	return have[0] + " and " + have[1]
}

// stationName cuts a station's parenthetical id: "La Jolla (9410230)" is read
// as "La Jolla". It is PlainLine'd because the name comes from the network.
func stationName(name string) string {
	if i := strings.Index(name, " ("); i > 0 {
		name = name[:i]
	}
	return capName(render.PlainLine(name))
}

// capName bounds a station name that will be spoken and shown (NFR-6).
func capName(s string) string {
	const cap = 48
	if r := []rune(s); len(r) > cap {
		return string(r[:cap])
	}
	return s
}

func deref(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

// compassPoint maps degrees true to the 16-point heading bearingLong words.
func compassPoint(deg float64) string {
	pts := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	i := int(math.Mod(math.Round(deg/22.5), 16))
	if i < 0 {
		i += 16
	}
	return pts[i]
}
