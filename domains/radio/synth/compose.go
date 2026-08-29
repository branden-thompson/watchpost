package synth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Segment is one narrated unit; Key identifies its content so rendered
// audio can be cached across cycles (§5: keyed on product issuance).
type Segment struct {
	Key   string
	Text  string
	Pause time.Duration // extra silence after the text, beyond the standard gap (UAT 112.3)
}

// leadPause separates the safety notice from the forecast span (UAT 112.3).
const leadPause = 2 * time.Second

// forecastDays is the span a zone forecast covers (seven days).
const forecastDays = 7

// Composer builds the broadcast's spoken text from the script library
// (domains/radio/script, 0.13.0): every sentence frame is a script file —
// weather-radio/ for the cycle's own lead, conditions, alerts and tail,
// fire-report/ and seismic-report/ for the two reports inside it,
// voice-preview/ for the chooser's line — and the Go here only computes the
// words the frames take (counts, distances, durations). Scripts nil = the
// built-in scripts.
type Composer struct {
	Scripts *script.Library
}

// say renders one phrase for the air (script.Library.Say: Plain'd, silence
// for a missing or broken part; nil speaks from the built-in tree).
func (c Composer) say(report, part string, data any) string { return c.Scripts.Say(report, part, data) }

// Compose builds one broadcast cycle the way NWR does (AI-13): the lead
// (UAT 79 script), current conditions, active alerts, the office's products
// in broadcast order, then the tail naming the correspondent voice.
// Temperatures are read in the location's display units; everything is
// plain sentences for the voice.
func (c Composer) Compose(loc snapshot.Location, products []Product, now time.Time, imperial bool, voiceName string, station Station, fire FireReport, seismic SeismicReport) []Segment {
	var segs []Segment
	notice, span := c.LeadParts(loc.Label, station, now)
	segs = append(segs, Segment{Key: "lead:" + loc.Label + station.Callsign, Text: notice, Pause: leadPause},
		Segment{Key: "lead-span:" + now.Format("2006-01-02"), Text: span})
	if cond := c.conditions(loc, imperial); cond != "" {
		segs = append(segs, Segment{Key: "wx:" + cond, Text: cond})
	}
	for _, a := range loc.Alerts {
		text := c.say("weather-radio", "alert", map[string]string{"Headline": strings.TrimSuffix(NormalizeLine(a.Headline), "."), "Description": strings.Join(Normalize(a.Description), " ")})
		for i, piece := range Segments([]string{ExpandStates(text)}) {
			segs = append(segs, Segment{Key: fmt.Sprintf("alert:%s:%d", a.ID, i), Text: piece})
		}
	}
	for _, p := range products {
		for i, piece := range Segments(Normalize(p.Text)) {
			segs = append(segs, Segment{Key: fmt.Sprintf("%s:%s:%d", p.Type, p.ID, i), Text: piece})
		}
	}
	// UAT 115: two seconds of air between reports (forecast → fire → …),
	// one second before the sign-off — never one report running into the next.
	if fireSegs := c.FireSegments(loc.Label, fire, imperial, now); len(fireSegs) > 0 { // UAT 114: after the forecast, before the tail; skipped without fire data
		pauseLast(segs, reportPause)
		segs = append(segs, fireSegs...)
	}
	if seismicSegs := c.SeismicSegments(loc.Label, seismic, imperial, now); len(seismicSegs) > 0 { // P4: after the fire report; skipped without seismic entries
		pauseLast(segs, reportPause)
		segs = append(segs, seismicSegs...)
	}
	pauseLast(segs, tailPause)
	segs = append(segs, Segment{Key: "tail:" + voiceName, Text: c.Tail(voiceName)})
	return segs
}

// reportPause / tailPause are the air between reports and before the tail (UAT 115).
const (
	reportPause = 2 * time.Second
	tailPause   = time.Second
)

// pauseLast gives the last segment at least d of pause.
func pauseLast(segs []Segment, d time.Duration) {
	if n := len(segs); n > 0 && segs[n-1].Pause < d {
		segs[n-1].Pause = d
	}
}

// Station is the NWR transmitter the lead points listeners to (UAT 112):
// the covering transmitter, else the nearest; empty when none is known.
type Station struct {
	Callsign string // "KEC62"
	Site     string // "San Diego"
	State    string // "CA"
	FreqMHz  string // "162.400" — read digit by digit (UAT 112.2)
}

// Lead is the broadcast opening (UAT 79 / 112, HUM LEAD script): the
// location, where the live NOAA broadcast can be heard on a radio, the
// delay/safety notice, the source, and the forecast span from today through
// the seventh day. Without a known station the live-broadcast sentence is
// left out rather than pointed at nothing.
func (c Composer) Lead(location string, station Station, now time.Time) string {
	notice, span := c.LeadParts(location, station, now)
	return notice + " " + span
}

// LeadParts is the lead in its two spoken pieces: the notice (through "life
// safety use.") and the forecast span — a two-second pause sits between
// them on air (UAT 112.3).
func (c Composer) LeadParts(location string, station Station, now time.Time) (notice, span string) {
	live := ""
	if station.Callsign != "" {
		where := station.Site
		if station.State != "" {
			where += ", " + station.State
		}
		if station.FreqMHz != "" {
			where += " broadcasting on " + station.FreqMHz + " MHz"
		}
		live = c.say("weather-radio", "live", map[string]string{"Callsign": station.Callsign, "Where": ExpandStates(where)})
	}
	notice = c.say("weather-radio", "head", map[string]string{"Location": ExpandStates(location), "Live": live})
	span = c.say("weather-radio", "span", map[string]string{"From": now.Format("Monday, January 2"), "Until": now.AddDate(0, 0, forecastDays-1).Format("Monday, January 2")})
	return notice, span
}

// Tail is the broadcast sign-off (UAT 79, HUM LEAD script).
func (c Composer) Tail(voiceName string) string {
	if voiceName == "" {
		voiceName = "your correspondent"
	}
	return c.say("weather-radio", "tail", map[string]string{"Voice": voiceName})
}

// conditions narrates the current observation, when there is one.
func (c Composer) conditions(loc snapshot.Location, imperial bool) string {
	h := loc.Harmonized
	if h.Source.Provider == "" {
		return ""
	}
	var parts []string
	if h.Condition != "" && h.Condition != "unknown" {
		parts = append(parts, strings.ReplaceAll(h.Condition, "_", " "))
	}
	if h.Temp != nil {
		parts = append(parts, fmt.Sprintf("temperature %s", degrees(*h.Temp, imperial)))
	}
	if h.HumidityPct != nil {
		parts = append(parts, fmt.Sprintf("humidity %.0f percent", *h.HumidityPct))
	}
	if h.Wind != nil {
		if imperial {
			parts = append(parts, fmt.Sprintf("wind %s at %.0f miles per hour", compass(h.WindDirDeg), *h.Wind*2.23694))
		} else {
			parts = append(parts, fmt.Sprintf("wind %s at %.0f kilometres per hour", compass(h.WindDirDeg), *h.Wind*3.6))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return c.say("weather-radio", "conditions", map[string]string{"Items": strings.Join(parts, ", ")})
}

func degrees(c float64, imperial bool) string {
	if imperial {
		return fmt.Sprintf("%.0f degrees", c*9/5+32)
	}
	return fmt.Sprintf("%.0f degrees Celsius", c)
}

func compass(deg *float64) string {
	if deg == nil {
		return "variable"
	}
	names := []string{"north", "northeast", "east", "southeast", "south", "southwest", "west", "northwest"}
	return names[geo.CompassIndex(*deg, 8)] // 8 points for the spoken wind — the 16-point wording is a §0.9 decision, not a nit
}

// Sample is the voice chooser's preview line (UAT 86).
func (c Composer) Sample(voiceName string) string {
	return c.say("voice-preview", "sample", map[string]string{"Voice": voiceName})
}

// SamplePCM renders the preview line in a voice as 16-bit LE stereo PCM at
// the voice's rate.
func (c Composer) SamplePCM(ctx context.Context, v Voice) ([]byte, error) {
	mono, err := v.Say(ctx, Pronounce(c.Sample(v.Name())))
	if err != nil {
		return nil, err
	}
	return monoToStereo(mono), nil
}
