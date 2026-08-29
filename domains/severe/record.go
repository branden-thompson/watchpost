package severe

// record.go — RecordOf: the ONE place the per-class switch happens (AX-2 A,
// red-team C-18). Every field a feed supplied passes through render.Plain here,
// so no renderer can reintroduce an unstripped path (NFR-6, red-team S1).

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/render"
)

// Record is the [A]-shaped presentation of one row: a title, a bracketed meta
// line, the timing line, the area/position line and unwrapped paragraphs.
type Record struct {
	Title  string
	Meta   string
	Timing string
	Area   string
	Paras  []string
}

// stamp is the record's clock: "08/28 08:45 CDT" in the given zone.
func stamp(t time.Time, in *time.Location) string {
	if t.IsZero() {
		return ""
	}
	if in == nil {
		in = time.Local
	}
	return t.In(in).Format("01/02 15:04 MST")
}

// compass names a direction in degrees ("W", "NE").
func compass(deg int) string {
	names := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	return names[(((deg%360)+360)%360+22)/45%8]
}

func title(s string) string { return strings.ToUpper(render.Plain(s)) }

func cap1(s string) string {
	s = render.Plain(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s) // by rune: a non-ASCII initial is a rune, not two U+FFFD (R5-B-06)
	return string(unicode.ToUpper(r)) + strings.ToLower(s[n:])
}

// RecordOf renders a row's record in the zone `in` (the tied location's clock,
// else the viewer's local one — FR-9).
func RecordOf(r Row, in *time.Location) Record {
	if err := invariant.Check(r.Product != "", "a row needs a product name to title its record"); err != nil {
		r.Product = "Event"
	}
	switch {
	case r.Detail.Alert != nil:
		return alertRecord(r, in)
	case r.Detail.Severe != nil:
		return severeRecord(r, in)
	case r.Detail.Quake != nil:
		return quakeRecord(r, in)
	case r.Detail.Tropical != nil:
		return tropicalRecord(r, in)
	}
	return Record{Title: title(r.Product), Timing: "Declared " + stamp(r.At, in), Area: "Area: " + render.Plain(r.Location)}
}

// timing is the record's clock line: "Declared <issued>   Starts <onset>
// Expires <until>   (~window)" — Starts only when the hazard begins after the
// declaration (an advisory issued in the morning for the evening), the window
// measured from the start.
func timing(verb string, at, onset, until time.Time, in *time.Location) string {
	s := verb + " " + stamp(at, in)
	from := at
	if onset.After(at) {
		s += "   Starts " + stamp(onset, in)
		from = onset
	}
	if until.After(from) {
		s += "   Expires " + stamp(until, in) + "   (~" + shortDur(until.Sub(from)) + ")"
	}
	return s
}

// shortDur renders a window as "15m", "2h", "1h30m", "3d" — never "15m0s"
// (time.Duration's String keeps the zero seconds).
func shortDur(d time.Duration) string {
	d = d.Round(time.Minute)
	if d >= 24*time.Hour {
		return fmt.Sprintf("%dd", int(d.Hours())/24)
	}
	h, m := int(d.Hours()), int(d.Minutes())%60
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh%dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

// capExtras renders the allowlisted CAP parameters as one line ("Wind gusts
// to 60 mph · Hail to 1.00 in"), "" when none.
// onsetOf reads the location record's optional onset.
func onsetOf(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func capExtras(d *globalfeed.SevereDetail) string {
	if d == nil {
		return ""
	}
	var extra []string
	if d.MaxWindGust != "" {
		extra = append(extra, "Wind gusts to "+render.Plain(d.MaxWindGust))
	}
	if d.MaxHailSize != "" {
		extra = append(extra, "Hail to "+render.Plain(d.MaxHailSize))
	}
	if d.EventMotion != "" {
		extra = append(extra, render.Plain(d.EventMotion))
	}
	return strings.Join(extra, " · ")
}

func alertRecord(r Row, in *time.Location) Record {
	a := r.Detail.Alert
	rec := Record{
		Title:  title(a.Event),
		Meta:   "[" + cap1(a.Severity) + " · " + cap1(a.Urgency) + " · " + cap1(a.Certainty) + "]",
		Timing: timing("Declared", r.At, onsetOf(a.Onset), r.Until, in),
	}
	area := render.Plain(a.AreaDesc)
	if a.SenderName != "" {
		area += " · " + render.Plain(a.SenderName)
	}
	rec.Area = "Area: " + area
	rec.Paras = paragraphs(a.Description, a.Instruction)
	if extra := capExtras(r.Detail.Severe); extra != "" { // the national feed's CAP parameters, when the same alert came both ways
		rec.Paras = append([]string{extra}, rec.Paras...)
	}
	return rec
}

func severeRecord(r Row, in *time.Location) Record {
	d := r.Detail.Severe
	rec := Record{
		Title:  title(r.Product),
		Meta:   "[" + cap1(d.Severity) + " · " + cap1(d.Urgency) + " · " + cap1(d.Certainty) + "]",
		Timing: timing("Declared", r.At, d.Onset, r.Until, in),
	}
	area := render.Plain(r.Location)
	if d.SenderName != "" {
		area += " · " + render.Plain(d.SenderName)
	}
	rec.Area = "Area: " + area
	rec.Paras = paragraphs(d.Description, d.Instruction)
	if extra := capExtras(d); extra != "" {
		rec.Paras = append([]string{extra}, rec.Paras...)
	}
	return rec
}

func quakeRecord(r Row, in *time.Location) Record {
	q := r.Detail.Quake
	mag := "M ?"
	if q.Mag != nil {
		mag = fmt.Sprintf("M %.1f", *q.Mag)
	}
	var meta []string
	if q.Mag != nil {
		meta = append(meta, strings.TrimSpace(fmt.Sprintf("Magnitude %.1f %s", *q.Mag, render.Plain(q.MagType))))
	}
	if q.DepthKm > 0 {
		meta = append(meta, fmt.Sprintf("Depth %.0f km", q.DepthKm))
	}
	if q.Alert != "" {
		meta = append(meta, "PAGER "+q.Alert)
	}
	if q.Tsunami {
		meta = append(meta, "Tsunami yes")
	} else {
		meta = append(meta, "Tsunami no")
	}
	rec := Record{Title: mag + " " + title(r.Product), Meta: "[" + strings.Join(meta, " · ") + "]", Timing: "Recorded " + stamp(r.At, in)}
	if !q.UpdatedAt.IsZero() {
		rec.Timing += "   Updated " + stamp(q.UpdatedAt, in)
	}
	rec.Area = "Location: " + render.Plain(r.Location)
	if r.HasPoint {
		rec.Area += fmt.Sprintf(" (%.2f %s, %.2f %s)", abs(r.Lat), ns(r.Lat), abs(r.Lon), ew(r.Lon))
	}
	var facts []string
	if q.Felt > 0 {
		facts = append(facts, fmt.Sprintf("Felt reports %d", q.Felt))
	}
	if q.CDI != nil {
		facts = append(facts, fmt.Sprintf("Community intensity %.1f", *q.CDI))
	}
	if q.MMI != nil {
		facts = append(facts, fmt.Sprintf("Modelled intensity %.1f", *q.MMI))
	}
	if q.Sig > 0 {
		facts = append(facts, fmt.Sprintf("Significance %d", q.Sig))
	}
	if q.Status != "" {
		facts = append(facts, cap1(q.Status))
	}
	if len(facts) > 0 {
		rec.Paras = []string{strings.Join(facts, " · ")}
	}
	return rec
}

func tropicalRecord(r Row, in *time.Location) Record {
	d := r.Detail.Tropical
	head := title(r.Product)
	if d.Name != "" {
		head += " " + title(d.Name)
	}
	if d.BinNumber != "" {
		head += " (" + title(d.BinNumber) + ")"
	}
	var meta []string
	if d.WindKt > 0 {
		meta = append(meta, fmt.Sprintf("Winds %d kt", d.WindKt))
	}
	if d.PressureMb > 0 {
		meta = append(meta, fmt.Sprintf("Pressure %d mb", d.PressureMb))
	}
	if d.MoveSpeedKt > 0 {
		meta = append(meta, fmt.Sprintf("Moving %s at %d kt", compass(d.MoveDirDeg), d.MoveSpeedKt))
	}
	rec := Record{Title: head, Meta: "[" + strings.Join(meta, " · ") + "]", Timing: "Reported " + stamp(r.At, in)}
	if d.AdvisoryNum != "" {
		rec.Timing += "   Advisory " + render.Plain(d.AdvisoryNum)
		if !d.AdvisoryAt.IsZero() {
			rec.Timing += " issued " + stamp(d.AdvisoryAt, in)
		}
	}
	pos := "Position: " + render.Plain(d.LatText) + " " + render.Plain(d.LonText)
	if d.Basin != "" {
		pos += " · " + render.Plain(d.Basin) + " basin"
	}
	rec.Area = pos
	var adv []string
	if d.AdvisoryNum != "" {
		adv = append(adv, "public "+render.Plain(d.AdvisoryNum))
	}
	if d.ForecastNum != "" {
		adv = append(adv, "forecast "+render.Plain(d.ForecastNum))
	}
	if d.DiscussionNum != "" {
		adv = append(adv, "discussion "+render.Plain(d.DiscussionNum))
	}
	if len(adv) > 0 {
		rec.Paras = []string{"Advisories: " + strings.Join(adv, ", ") + " (nhc.noaa.gov)"} // named, never linked (S6)
	}
	return rec
}

// paragraphs splits CAP prose into paragraphs (blank-line separated), Plain'd,
// with the instruction as a final "Instructions: …" paragraph.
func paragraphs(description, instruction string) []string {
	var out []string
	for _, p := range strings.Split(render.Plain(description), "\n\n") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if s := strings.TrimSpace(render.Plain(instruction)); s != "" {
		out = append(out, "Instructions: "+s)
	}
	return out
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func ns(lat float64) string {
	if lat < 0 {
		return "S"
	}
	return "N"
}

func ew(lon float64) string {
	if lon < 0 {
		return "W"
	}
	return "E"
}
