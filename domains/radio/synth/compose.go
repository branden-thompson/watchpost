package synth

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// Compose builds one broadcast cycle the way NWR does (AI-13): the lead
// (UAT 79 script), current conditions, active alerts, the office's products
// in broadcast order, then the tail naming the correspondent voice.
// Temperatures are read in the location's display units; everything is
// plain sentences for the voice.
func Compose(loc snapshot.Location, products []Product, now time.Time, imperial bool, voiceName string, station Station, fire FireReport) []Segment {
	var segs []Segment
	notice, span := LeadParts(loc.Label, station, now)
	segs = append(segs, Segment{Key: "lead:" + loc.Label + station.Callsign, Text: notice, Pause: leadPause},
		Segment{Key: "lead-span:" + now.Format("2006-01-02"), Text: span})
	if c := conditions(loc, imperial); c != "" {
		segs = append(segs, Segment{Key: "wx:" + c, Text: c})
	}
	for _, a := range loc.Alerts {
		text := fmt.Sprintf("%s. %s", strings.TrimSuffix(NormalizeLine(a.Headline), "."), strings.Join(Normalize(a.Description), " "))
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
	if fireSegs := FireSegments(loc.Label, fire, imperial, now); len(fireSegs) > 0 { // UAT 114: after the forecast, before the tail; skipped without fire data
		pauseLast(segs, reportPause)
		segs = append(segs, fireSegs...)
	}
	pauseLast(segs, tailPause)
	segs = append(segs, Segment{Key: "tail:" + voiceName, Text: Tail(voiceName)})
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
func Lead(location string, station Station, now time.Time) string {
	notice, span := LeadParts(location, station, now)
	return notice + " " + span
}

// LeadParts is the lead in its two spoken pieces: the notice (through "life
// safety use.") and the forecast span — a two-second pause sits between
// them on air (UAT 112.3).
func LeadParts(location string, station Station, now time.Time) (notice, span string) {
	from := now.Format("Monday, January 2")
	until := now.AddDate(0, 0, forecastDays-1).Format("Monday, January 2")
	live := ""
	if station.Callsign != "" {
		where := station.Site
		if station.State != "" {
			where += ", " + station.State
		}
		if station.FreqMHz != "" {
			where += " broadcasting on " + station.FreqMHz + " MHz"
		}
		live = fmt.Sprintf(" A version of this forecast is also broadcast live from %s, %s and is accessible via NOAA radio devices and receivers.", station.Callsign, ExpandStates(where))
	}
	notice = fmt.Sprintf("This is Watchpost Weather Radio serving %s.%s Watchpost Weather Radio forecasts may be delayed and are not intended for life safety use.", ExpandStates(location), live)
	span = fmt.Sprintf("This forecast is from the National Oceanic and Atmospheric Administration and is for %s until %s.", from, until)
	return notice, span
}

// Tail is the broadcast sign-off (UAT 79, HUM LEAD script).
func Tail(voiceName string) string {
	if voiceName == "" {
		voiceName = "your correspondent"
	}
	return fmt.Sprintf("This is %s for Watchpost Weather Radio. You can change your correspondent voice in your Watchpost CLI application settings.", voiceName)
}

// conditions narrates the current observation, when there is one.
func conditions(loc snapshot.Location, imperial bool) string {
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
	return "Current conditions: " + strings.Join(parts, ", ") + "."
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
	return names[int((*deg+22.5)/45)%8]
}

// Sample is the voice chooser's preview line (UAT 86).
func Sample(voiceName string) string {
	return fmt.Sprintf("This is %s for Watchpost Weather Radio.", voiceName)
}

// SamplePCM renders the preview line in a voice as 16-bit LE stereo PCM at
// the voice's rate.
func SamplePCM(ctx context.Context, v Voice) ([]byte, error) {
	mono, err := v.Say(ctx, Pronounce(Sample(v.Name())))
	if err != nil {
		return nil, err
	}
	return monoToStereo(mono), nil
}
