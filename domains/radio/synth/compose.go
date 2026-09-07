package synth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Segment is one narrated unit; Key identifies its content so rendered
// audio can be cached across cycles (§5: keyed on product issuance).
type Segment struct {
	Key  string
	Text string

	// Role is who reads this segment. The zero value is cast.All — the root —
	// so a segment nobody tagged is read by the voice that read everything
	// before 0.14.0 (FR-2).
	Role cast.Role

	// SelfIntro marks a segment that NAMES ITS OWN SPEAKER — today only the
	// sign-off ("This is Samantha for Watchpost Weather Radio…").
	//
	// A hand-over into such a segment is a double introduction by
	// construction: the listener hears "This is Eddie, taking over for Karen."
	// and then, seconds later, "This is Eddie for Watchpost Weather Radio."
	// Nobody introduces themselves twice, so the Source suppresses the
	// hand-over here and lets the segment do the introducing.
	// (UAT 2026-08-30: heard when a location had no seismic report, so the
	// fire report ran straight into the sign-off across a voice change.)
	SelfIntro bool

	Pause time.Duration // extra silence after the text, beyond the standard gap (UAT 112.3)
}

// handoverBuiltin is the line a correspondent speaks when taking over. It is
// the fallback for a broken script override: a hand-over that fails to render
// must never become silence (FR-5).
const handoverBuiltin = "This is %s, taking over for %s."

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

// Reports carries the optional reports a cycle may include, so adding one is
// a new FIELD rather than a new positional parameter (RS-12: Compose already
// took eight, and the maritime report would have made nine). The ZERO VALUE
// composes exactly the 0.13.0 broadcast — that is FR-2's anchor, and the
// property every later batch is measured against.
//
// Maritime joins this struct at P3 (Task 3.4). That it can, without touching
// this signature or a single call site, is the whole point of the type.
type Reports struct {
	Fire     FireReport
	Seismic  SeismicReport
	Maritime MarineReport
}

// Compose builds one broadcast cycle the way NWR does (AI-13): the lead
// (UAT 79 script), current conditions, active alerts, the office's products
// in broadcast order, then the tail naming the correspondent voice.
// Temperatures are read in the location's display units; everything is
// plain sentences for the voice.
func (c Composer) Compose(loc snapshot.Location, products []Product, now time.Time, imperial bool, voiceName string, station Station, reports Reports, clock render.Clock) []Segment {
	var segs []Segment
	notice, span := c.LeadParts(loc.Label, station, now, clock)
	segs = append(segs, Segment{Key: "lead:" + loc.Label + station.Callsign, Text: notice, Role: cast.Station, Pause: leadPause},
		Segment{Key: "lead-span:" + now.Format("2006-01-02"), Text: span, Role: cast.Station})
	if cond := c.conditions(loc, imperial); cond != "" {
		segs = append(segs, Segment{Key: "wx:" + cond, Text: cond, Role: cast.Weather})
	}
	for _, a := range loc.Alerts {
		text := c.say("weather-radio", "alert", map[string]string{"Headline": strings.TrimSuffix(NormalizeLine(a.Headline), "."), "Description": strings.Join(Normalize(a.Description), " ")})
		for i, piece := range Segments([]string{ExpandStates(text)}) {
			segs = append(segs, Segment{Key: fmt.Sprintf("alert:%s:%d", a.ID, i), Text: piece, Role: cast.Weather})
		}
	}
	for _, p := range products {
		for i, piece := range Segments(Normalize(p.Text)) {
			segs = append(segs, Segment{Key: fmt.Sprintf("%s:%s:%d", p.Type, p.ID, i), Text: piece, Role: cast.Weather})
		}
	}
	// UAT 115: two seconds of air between reports (forecast → fire → …),
	// one second before the sign-off — never one report running into the next.
	// The ruled order (MVS-D-18): products → MARITIME → fire → seismic → tail,
	// with the same two seconds of air between reports.
	if marineSegs := c.MarineSegments(loc.Label, reports.Maritime, imperial, now); len(marineSegs) > 0 {
		pauseLast(segs, reportPause)
		segs = append(segs, marineSegs...)
	}
	if fireSegs := c.FireSegments(loc.Label, reports.Fire, imperial, now); len(fireSegs) > 0 { // UAT 114: after the forecast, before the tail; skipped without fire data
		pauseLast(segs, reportPause)
		segs = append(segs, fireSegs...)
	}
	if seismicSegs := c.SeismicSegments(loc.Label, reports.Seismic, imperial, now); len(seismicSegs) > 0 { // P4: after the fire report; skipped without seismic entries
		pauseLast(segs, reportPause)
		segs = append(segs, seismicSegs...)
	}
	pauseLast(segs, tailPause)
	// Keyed "tail", not "tail:<voice>": the voice lives in the cache key
	// (VoiceToken), so a change of correspondent does not mint a new segment.
	segs = append(segs, Segment{Key: "tail:" + VoiceToken, Text: c.Tail(voiceName), Role: cast.Station, SelfIntro: true})
	return segs
}

// HandoffLine is what the incoming correspondent says when the voice changes
// mid-broadcast: "This is Rishi, taking over for Samantha." (MVS-D-6, FR-5).
//
// It reads the script tree so a listener can reword it, and falls back to the
// built-in line when the override is missing or fails to render. That fallback
// is the whole contract: a broken override must never turn a hand-over into
// silence or, worse, into a splice with no explanation.
//
// Both names arrive already plain and capped (the Source's spokenName owns
// that), so nothing here re-sanitises them.
func (c Composer) HandoffLine(from, to string) string {
	if line := c.say("handover", "line", map[string]string{"From": from, "To": to}); line != "" {
		return line
	}
	return fmt.Sprintf(handoverBuiltin, to, from)
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
func (c Composer) Lead(location string, station Station, now time.Time, clock render.Clock) string {
	notice, span := c.LeadParts(location, station, now, clock)
	return notice + " " + span
}

// LeadParts is the lead in its two spoken pieces: the notice (through "life
// safety use.") and the forecast span — a two-second pause sits between
// them on air (UAT 112.3).
func (c Composer) LeadParts(location string, station Station, now time.Time, clock render.Clock) (notice, span string) {
	live := ""
	if station.Callsign != "" {
		where := station.Site
		if station.State != "" {
			where += ", " + station.State
		}
		if station.FreqMHz != "" {
			where += " broadcasting on " + station.FreqMHz + " MHz"
		}
		// The callsign goes through the clock's ID reading: under MILITARY it is
		// spelled in NATO phonetics, because a callsign misheard on a weather
		// broadcast sends somebody to the wrong frequency (HUM LEAD, UAT
		// 2026-08-30).
		live = c.say("weather-radio", "live", map[string]string{"Callsign": clock.SpokenID(station.Callsign), "Where": ExpandStates(where)})
	}
	notice = c.say("weather-radio", "head", map[string]string{"Location": ExpandStates(location), "Live": live})
	span = c.say("weather-radio", "span", map[string]string{"From": now.Format("Monday, January 2"), "Until": now.AddDate(0, 0, forecastDays-1).Format("Monday, January 2")})
	return notice, span
}

// Tail is the broadcast sign-off (UAT 79, HUM LEAD script).
// Tail names the correspondent who reaches the sign-off.
//
// The empty-name branch is gone: spokenName (source.go) is the one owner of
// "your correspondent", so the substitution happens once, where the name is
// resolved, rather than being re-derived by everything that displays or speaks
// it (RS-18).
func (c Composer) Tail(voiceName string) string {
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
