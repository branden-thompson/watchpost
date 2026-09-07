# P3 — the maritime report and the tones (multi-voice-support, 0.14.0)

> **PRIOR ART — not the plan.** A verbatim snapshot of this batch document *before* the PLAN artefacts were
> stripped to task shape (`AP-PLANCODE-01`, `../../06-key_learnings/retro-notes.md` RN-1). The artefact of
> record is `../p3-maritime-tones.md`. Read `README.md` in this folder first: it says which of these blocks were actually
> compiled and run, and lists the known defects (D-1 … D-9) not to copy forward.



```
Goal:         A full maritime report in the broadcast — coastal-waters forecast, buoy observations, tides,
              currents (FR-6, MVS-D-4/14/18/21) — and the five ratified tone presets behind the classifier
              (FR-11, MVS-D-9/11/12/15).
Architecture: plan.md §2.5–2.6; maritime-report.md §7–10; tones.md. P1 (Reports{}, cast) and P2 (Segment.Role,
              tone(class)) are the base.
Tech Stack:   Go 1.27 · the script library · httpx · the recVoice fake
Branch:       feature/multi-voice-support
Gate:         go test ./domains/... ./platform/... ./app -race -count=2; make verify; make p10; the R6 soak pattern
```

## File map

```
CREATE: platform/render/marine.go                — SeaState, TideTrend, NextTide, CurrentPhase (lifted from modes/tty; one owner for screen and voice)
CREATE: platform/render/marine_test.go
MODIFY: modes/tty/detail_marine.go               — calls render.SeaState/TideTrend/NextTide/CurrentPhase
CREATE: domains/radio/synth/marine.go            — MarineReport, MarineSegments, the word helpers
CREATE: domains/radio/synth/marine_test.go
CREATE: domains/radio/script/scripts/maritime-report/{head,forecast,observed,sea,water,wind,tide,tide-next,current,absence,link}.txt
MODIFY: domains/radio/script/script_test.go      — the report list; the parts
MODIFY: domains/radio/synth/compose.go           — Reports.Maritime; the order products → maritime → fire → seismic
MODIFY: domains/radio/synth/synth_test.go        — TestReportsAreSeparatedByAir with three reports
MODIFY: platform/snapshot/{assembler,harmonize}.go (MarineFor; mergeMarine shared)           — MarineFor (narrow read, the fillMarine merge)
CREATE: app/marine.go                            — marineReportOf; livePipelines.marineFor
MODIFY: app/radio.go, app/dashboard.go           — the maritime hook beside fire/seismic
MODIFY: domains/radio/synth/products.go          — Marine(ctx, office): the CWF
CREATE: domains/weather/nws/marinezone.go        — MarineZoneFor: the CWF's zones → geometry → nearest
CREATE: domains/weather/nws/marinezone_test.go
MODIFY: domains/radio/synth/tone.go              — the other four presets (P2 Task 2.0 landed Preset/Classic/ToneRate)
MODIFY: domains/radio/synth/tone_test.go
```

---

### Task 3.1 — lift the marine words to `platform/render` (one owner for screen and voice; RS-11)

**File:** `platform/render/marine_test.go` (RED)

```go
package render

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestSeaStateBands(t *testing.T) {
	for m, want := range map[float64]string{0.05: "Calm (glassy)", 0.3: "Smooth", 0.9: "Slight Chop", 2.0: "Moderate Chop", 3.0: "Rough", 5.0: "Very Rough"} {
		if got := SeaState(m); got != want {
			t.Errorf("SeaState(%.2f) = %q, want %q", m, got, want)
		}
	}
}

func TestTideTrendAndNext(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	tides := []snapshot.TideEvent{{Time: now.Add(-time.Hour), Height: 0.1, Type: "L"}, {Time: now.Add(2 * time.Hour), Height: 1.7, Type: "H"}, {Time: now.Add(8 * time.Hour), Height: -0.03, Type: "L"}}
	nh, nl := NextTide(tides, "H", now), NextTide(tides, "L", now)
	if nh == nil || nl == nil || nh.Height != 1.7 || nl.Height != -0.03 || TideTrend(nh, nl) != "Rising" {
		t.Fatalf("next high %+v, next low %+v", nh, nl)
	}
	cur, next := CurrentPhase([]snapshot.CurrentEvent{{Time: now.Add(-time.Hour), Speed: 0.7, Type: "flood"}, {Time: now.Add(time.Hour), Speed: 0, Type: "slack"}}, now)
	if cur == nil || cur.Type != "flood" || next == nil || next.Type != "slack" {
		t.Fatalf("phase %+v next %+v", cur, next)
	}
}
```

**File:** `platform/render/marine.go` (GREEN) — the four functions moved verbatim from
`modes/tty/detail_marine.go` (`seaState` → `SeaState`, `tideTrend` → `TideTrend`, `nextTide` → `NextTide`,
`currentRow`'s phase scan → `CurrentPhase(events, now) (cur, next *snapshot.CurrentEvent)`), plus `FirstOf(vals ...*float64) *float64`
(lifted from `detail_marine.go:245`; the tty file calls `render.FirstOf`), with the doc comment: *"the screen and
the voice describe the same sea in the same words (RS-11)"*. Add the parity test **in `app/marine_test.go`** (`modes/tty` must never import a domain — `make lint-imports`, PR2-5):

```go
func TestScreenAndVoiceShareTheSeaWords(t *testing.T) {
	now := time.Date(2026, 8, 24, 17, 26, 0, 0, time.UTC)
	m := snapshot.Marine{WaveHeight: f64(0.9)}
	if got := render.SeaState(*m.WaveHeight); got != "Slight Chop" {
		t.Fatalf("the screen's words: %q", got)
	}
	spoken := synth.Composer{}.MarineSegments("Oceanside, CA", synth.MarineReport{Known: true, State: m, TZ: time.UTC}, true, now)
	var joined string
	for _, s := range spoken {
		joined += s.Text + " "
	}
	if !strings.Contains(joined, "Seas are slight chop") {
		t.Fatalf("the voice's words derive from the same SeaState: %q", joined)
	}
}
```

(`f64` is `func f64(v float64) *float64 { return &v }` — define it in `app/marine_test.go`; imports `strings`, `testing`, `time`, `render`, `snapshot`, `synth`.) In
`modes/tty/detail_marine.go` delete the four locals and call the `render` ones (`seaState(x)` →
`render.SeaState(x)`, etc.; `currentRow` keeps its formatting and calls `render.CurrentPhase`). Re-capture the
`platform/render` declset (`go test ./platform/render -run DeclarationSet -update-declset`).
**Verify:** `go test ./platform/render ./modes/tty -run 'Sea|Tide|Marine|Detail' -count=1`

---

### Task 3.2 — `MarineReport` and `MarineSegments` (FR-6; MVS-D-21 wording)

**File:** `domains/radio/synth/marine_test.go` (RED)

```go
package synth

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"slices"
)

func coastal(now time.Time) MarineReport { // f64 is the package's existing pointer helper (synth_test.go:23)
	la, _ := time.LoadLocation("America/Los_Angeles")
	return MarineReport{Known: true, TZ: la, State: snapshot.Marine{
		WaveHeight: f64(0.9), WavePeriod: f64(14), SwellHeight: f64(0.6), SwellDirDeg: f64(280), WaterTemp: f64(23.3),
		WindSpeed: f64(5.1), WindGust: f64(7.2), Buoy: "46224", BuoyDistanceKM: f64(9), ObservedAt: now.Add(-39 * time.Minute),
		TideLevel: f64(1.13), TideStation: "La Jolla (Scripps Institution Wharf)", TideStationKM: f64(38.8),
		Tides:    []snapshot.TideEvent{{Time: now.Add(2 * time.Hour), Height: 1.74, Type: "H"}, {Time: now.Add(9 * time.Hour), Height: -0.03, Type: "L"}},
		Currents: []snapshot.CurrentEvent{{Time: now.Add(-30 * time.Minute), Speed: 0.72, Type: "flood"}, {Time: now.Add(90 * time.Minute), Speed: 0, Type: "slack"}},
	}}
}

func TestMarineSegmentsReadTheScript(t *testing.T) {
	now := time.Date(2026, 8, 24, 17, 26, 0, 0, time.UTC) // 10:26 AM in Los Angeles
	segs := std.MarineSegments("Oceanside, CA", coastal(now), true, now)
	texts := segmentsText(segs)
	want := []string{
		"This is the Maritime report for Watchpost Radio, for Oceanside, California.",
		"Conditions at the nearest buoy, 6 miles offshore, observed 39 minutes ago.",
		"Seas are slight chop, with waves of about 3 feet, and a primary swell from the west at 2 feet with a period of 14 seconds.",
		"Water temperature 74 degrees.",
		"Wind at the buoy 11 miles per hour, gusting to 16.",
		"The tide is rising, 3.7 feet above the low-water mark, at La Jolla, 24 miles from you.",
		"The next high tide is at 12:26 PM at 5.7 feet; the next low at 7:26 PM, just below the low-water mark.",
		"Tidal currents are flooding at 1.4 knots, with slack water at 11:56 AM.",
		"For coastal conditions in your area, visit https://www.ndbc.noaa.gov",
	}
	if len(texts) != len(want) {
		t.Fatalf("segments:\n%s", strings.Join(texts, "\n"))
	}
	for i := range want {
		if texts[i] != want[i] {
			t.Errorf("segment %d:\n got %q\nwant %q", i, texts[i], want[i])
		}
	}
	for _, s := range segs {
		if s.Role != cast.Maritime || !strings.HasPrefix(s.Key, "maritime:") {
			t.Fatalf("every segment is the maritime role, content-keyed: %+v", s)
		}
	}
	if segs[0].Pause != reportPause {
		t.Fatal("the head carries the 2 s pause")
	}
}

func TestMarineSegmentsAbsenceAndSkip(t *testing.T) {
	now := time.Date(2026, 8, 24, 17, 26, 0, 0, time.UTC)
	if segs := std.MarineSegments("Boise, ID", MarineReport{}, true, now); segs != nil {
		t.Fatal("inland: the report is skipped entirely")
	}
	r := coastal(now)
	r.State.Tides, r.State.Currents, r.State.TideStation, r.State.TideLevel = nil, nil, "", nil
	texts := segmentsText(std.MarineSegments("Oceanside, CA", r, true, now))
	if !slices.Contains(texts, "There are no tide or current predictions available for your area.") {
		t.Fatalf("no station in range: one absence line: %v", texts)
	}
	r = coastal(now)
	r.State.Buoy, r.State.ObservedAt = "", time.Time{}
	for _, tx := range segmentsText(std.MarineSegments("Oceanside, CA", r, true, now)) {
		if strings.Contains(tx, "observed") {
			t.Fatalf("no buoy: the observed sentence is omitted, never 'observed 0 minutes ago': %q", tx)
		}
	}
}

func TestForecastPeriodsCapsAtThree(t *testing.T) {
	raw := "PZZ740-301015-\nCoastal Waters-\n\n.TONIGHT...W 10 kt.\n.SUNDAY...NW 15 kt.\n.SUNDAY NIGHT...W 10 kt.\n.MONDAY...N 5 kt.\n\n$$\n"
	got := ForecastPeriods(raw, 3)
	if strings.Contains(got, "MONDAY") || !strings.Contains(got, "SUNDAY NIGHT") {
		t.Fatalf("three periods kept, the fourth cut: %q", got)
	}
}

func TestMarineWordsMetric(t *testing.T) {
	now := time.Date(2026, 8, 24, 17, 26, 0, 0, time.UTC)
	texts := segmentsText(std.MarineSegments("Oceanside, CA", coastal(now), false, now))
	joined := strings.Join(texts, " ")
	for _, want := range []string{"waves of about 1 metre", "Water temperature 23 degrees Celsius", "18 kilometres per hour", "1.1 metres above the low-water mark", "1.4 knots"} {
		if !strings.Contains(joined, want) {
			t.Errorf("metric wording missing %q in:\n%s", want, joined)
		}
	}
}
```

(`segmentsText` exists in the synth tests; membership checks use `slices.Contains`; `reportPause` is the new shared constant below.)

**File:** `domains/radio/synth/marine.go` (GREEN)

```go
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

// MarineReport is what the broadcast's maritime section reads from (0.14.0,
// FR-6): the location's merged coastal-waters block, its time zone for the
// spoken clock, and the office's coastal-waters forecast cut to the nearest
// zone (MVS-D-14). Known is false inland — the report is skipped entirely.
type MarineReport struct {
	Known    bool
	State    snapshot.Marine
	TZ       *time.Location
	Lat, Lon float64
	Forecast string // the CWF's synopsis + nearshore zone, Normalized; "" when none
}

// (The head's pause reuses reportPause — the 2 s the fire and seismic heads already use; no third constant.)
// (reportHeadPause is not introduced; the sentence above is the rule.)
// reportPause separated a report's head from its body (the fire and
// seismic reports' own constants say the same 2 s).
const reportPause = 2 * time.Second

// SpokenPeriodsCap is how many forecast periods of the coastal-waters
// forecast are read (RAT-6): tonight and the next two.
const SpokenPeriodsCap = 3

// MarineSegments composes the Maritime report (maritime-report.md §7): the
// head, the coastal forecast, the buoy's observations, tides, currents, one
// absence line when no station is in range, and where to learn more. Every
// phrase whose data the feeds did not give is left out, never guessed.
func (c Composer) MarineSegments(location string, mr MarineReport, imperial bool, now time.Time) []Segment {
	if !mr.Known {
		return nil
	}
	tz := mr.TZ
	if tz == nil {
		tz = time.UTC
	}
	m := &mr.State
	place := ExpandStates(location)
	head := c.say("maritime-report", "head", map[string]string{"Location": place})
	segs := []Segment{{Key: "maritime:notice:" + contentKey(head), Text: head, Pause: reportPause}}
	body := c.observationSentences(mr, imperial, now)
	// The forecast is prose of several sentences: it is split into segments the
	// way the location forecast is (Segments, one Say per sentence group) so a
	// listener's [r] repeats a sentence, not a minute of text, and a hand-over
	// lands between sentences. The lead ("The coastal forecast…") stays one segment.
	if len(body) > 0 && mr.Forecast != "" {
		body = append(Segments([]string{body[0]}), body[1:]...) // Segments takes paragraphs
	}
	if len(body) > maxMaritimePieces { // a product is network text: the section is bounded whatever it says (SEC round 3)
		body = body[:maxMaritimePieces]
	}
	body = append(body, c.tideSentences(m, tz, now, imperial)...)
	if cur, next := render.CurrentPhase(m.Currents, now); cur != nil || next != nil {
		body = append(body, c.currentSentence(cur, next, tz))
	}
	if len(m.Tides) == 0 && len(m.Currents) == 0 {
		body = append(body, c.say("maritime-report", "absence", nil))
	}
	body = append(body, c.say("maritime-report", "link", nil))
	for _, piece := range body {
		if piece == "" {
			continue
		}
		segs = append(segs, Segment{Key: "maritime:" + contentKey(piece), Text: piece})
	}
	return tagged(segs, cast.Maritime)
}
```

`contentKey` is `fire.go:78`'s existing content hash (reused, not duplicated — the fire and seismic reports key
their segments the same way; the `maritime:` prefix keeps the caches apart). `Segments` is the existing
sentence splitter the location forecast goes through (`compose.go`); its output for the forecast is capped by
`SpokenPeriodsCap` upstream (`marineFor`).

```go

// observationSentences: the forecast, the buoy's provenance, the sea, the
// water and the wind (split from MarineSegments — P10-04 complexity).
func (c Composer) observationSentences(mr MarineReport, imperial bool, now time.Time) []string {
	m := &mr.State
	var out []string
	if mr.Forecast != "" { // already cut to SpokenPeriodsCap and Normalized by the deck
		out = append(out, c.say("maritime-report", "forecast", map[string]string{"Text": strings.ReplaceAll(mr.Forecast, "\n", " ")}))
	}
	if !m.ObservedAt.IsZero() && m.Buoy != "" { // the screen's exact guard: a forecast-only block stamps ObservedAt = now (RS-11)
		out = append(out, c.say("maritime-report", "observed", map[string]string{"Distance": distanceWordsPtr(m.BuoyDistanceKM, imperial), "Ago": durationWords(now.Sub(m.ObservedAt))}))
	}
	if sea := c.seaSentence(m, imperial); sea != "" {
		out = append(out, sea)
	}
	if m.WaterTemp != nil {
		out = append(out, c.say("maritime-report", "water", map[string]string{"Temp": degrees(*m.WaterTemp, imperial)}))
	}
	if m.WindSpeed != nil {
		gust := ""
		if m.WindGust != nil {
			gust = speedNumber(*m.WindGust, imperial)
		}
		out = append(out, c.say("maritime-report", "wind", map[string]string{"Speed": speedWords(*m.WindSpeed, imperial), "Gust": gust}))
	}
	return out
}

// seaSentence: sea state, wave height and the primary swell in one sentence
// (kept to one so the report stays short — maritime-report.md §7).
func (c Composer) seaSentence(m *snapshot.Marine, imperial bool) string {
	if m.WaveHeight == nil {
		return ""
	}
	data := map[string]string{"State": strings.ToLower(strings.TrimSuffix(render.SeaState(*m.WaveHeight), " (glassy)")), "Waves": heightAbout(*m.WaveHeight, imperial), "Swell": ""}
	if h := render.FirstOf(m.SwellHeight, m.WaveHeight); h != nil && m.SwellDirDeg != nil {
		swell := "a primary swell from the " + compass(m.SwellDirDeg) + " at " + heightWords(*h, imperial)
		if m.WavePeriod != nil {
			swell += fmt.Sprintf(" with a period of %.0f seconds", *m.WavePeriod)
		}
		data["Swell"] = swell
	}
	return c.say("maritime-report", "sea", data)
}

// tideSentences: the trend and level with the station, then the next high and low.
func (c Composer) tideSentences(m *snapshot.Marine, tz *time.Location, now time.Time, imperial bool) []string {
	var out []string
	nh, nl := render.NextTide(m.Tides, "H", now), render.NextTide(m.Tides, "L", now)
	if m.TideStation != "" || m.TideLevel != nil || nh != nil || nl != nil {
		data := map[string]string{"Trend": strings.ToLower(render.TideTrend(nh, nl)), "Level": "", "Station": stationWords(m.TideStation), "Distance": distanceWordsPtr(m.TideStationKM, imperial)}
		if m.TideLevel != nil {
			data["Level"] = levelWords(*m.TideLevel, imperial)
		}
		out = append(out, c.say("maritime-report", "tide", data))
	}
	if nh != nil || nl != nil {
		data := map[string]string{"High": "", "HighLevel": "", "Low": "", "LowLevel": ""}
		if nh != nil {
			data["High"], data["HighLevel"] = clockWords(nh.Time, tz), heightTenths(nh.Height, imperial) // a high is a bare height ("5.7 feet"); a low may be below the mark
		}
		if nl != nil {
			data["Low"], data["LowLevel"] = clockWords(nl.Time, tz), levelWords(nl.Height, imperial)
		}
		out = append(out, c.say("maritime-report", "tide-next", data))
	}
	return out
}

// currentSentence: the phase in force with its speed, and the next event.
func (c Composer) currentSentence(cur, next *snapshot.CurrentEvent, tz *time.Location) string {
	data := map[string]string{"Phase": "slack water", "Speed": "", "Next": "", "NextAt": ""}
	if cur != nil && cur.Type != "slack" {
		data["Phase"], data["Speed"] = flowWords(cur.Type), knotWords(cur.Speed)
	}
	if next != nil {
		data["Next"], data["NextAt"] = flowWords(next.Type), clockWords(next.Time, tz)
	}
	return c.say("maritime-report", "current", data)
}

// --- the words (MVS-D-21: the listener's unit for heights, wind and
// distance; knots for currents under both systems; "the low-water mark" for
// MLLW; a 12-hour spoken clock in the location's zone) ---

// heightWords: "2 feet" / "0.6 metres" — feet whole, metres to a tenth.
func heightWords(m float64, imperial bool) string {
	if imperial {
		ft := int(math.Round(m * 3.28084))
		if ft == 1 {
			return "1 foot"
		}
		return fmt.Sprintf("%d feet", ft)
	}
	return fmt.Sprintf("%s metres", trimFloat(m, 1))
}

// heightTenths: "5.7 feet" / "1.74 metres" — a tide height to a tenth of a foot / a hundredth of a metre.
func heightTenths(m float64, imperial bool) string {
	if imperial {
		return trimFloat(m*3.28084, 1) + " feet"
	}
	return trimFloat(m, 2) + " metres"
}

// heightAbout: "about 3 feet" / "about 1 metre" (waves are not read to a tenth).
func heightAbout(m float64, imperial bool) string {
	if imperial {
		return "about " + heightWords(m, true)
	}
	if r := math.Round(m); r == 1 {
		return "about 1 metre"
	}
	return fmt.Sprintf("about %.0f metres", math.Round(m))
}

// levelWords is a tide level relative to the low-water mark (MLLW):
// "3.7 feet above the low-water mark", "just below the low-water mark" when
// within a tenth of a foot below, "0.5 feet below the low-water mark".
func levelWords(m float64, imperial bool) string {
	v, unit := m, "metres"
	if imperial {
		v, unit = m*3.28084, "feet"
	}
	switch {
	case v >= 0:
		return fmt.Sprintf("%s %s above the low-water mark", trimFloat(v, 1), unit)
	case v > -0.15 && imperial, v > -0.05 && !imperial:
		return "just below the low-water mark"
	}
	return fmt.Sprintf("%s %s below the low-water mark", trimFloat(-v, 1), unit)
}

// knotWords: "1.4 knots" / "1 knot" from m/s.
func knotWords(mps float64) string {
	kt := mps * 1.94384
	if math.Round(kt*10) == 10 {
		return "1 knot"
	}
	return trimFloat(kt, 1) + " knots"
}

// speedWords: buoy wind in the listener's unit ("11 miles per hour").
func speedWords(mps float64, imperial bool) string {
	if imperial {
		return speedNumber(mps, true) + " miles per hour"
	}
	return speedNumber(mps, false) + " kilometres per hour"
}

func speedNumber(mps float64, imperial bool) string {
	if imperial {
		return fmt.Sprintf("%.0f", mps*2.23694)
	}
	return fmt.Sprintf("%.0f", mps*3.6)
}

// clockWords: "12:26 PM" in the location's zone (the pronunciation pass reads
// it as a time — normalize.go's clock rule).
func clockWords(t time.Time, tz *time.Location) string { return t.In(tz).Format("3:04 PM") }

// flowWords: the current's phase as a verb.
func flowWords(typ string) string {
	switch typ {
	case "flood":
		return "flooding"
	case "ebb":
		return "ebbing"
	}
	return "slack water"
}

// stationWords cuts a CO-OPS station name at its parenthetical qualifier
// ("La Jolla (Scripps Institution Wharf)" → "La Jolla"), plain and bounded.
func stationWords(name string) string {
	name, _, _ = strings.Cut(render.PlainLine(name), " (")
	return capRunes(strings.TrimSpace(name), 48)
}

func capRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// distanceWordsPtr: distanceWords for an optional distance ("" when unknown).
func distanceWordsPtr(km *float64, imperial bool) string {
	if km == nil {
		return ""
	}
	return distanceWords(*km, imperial)
}

// trimFloat formats to n decimals and drops trailing zeros ("1.70" → "1.7", "2.00" → "2").
func trimFloat(v float64, n int) string {
	s := fmt.Sprintf("%.*f", n, v)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}

// ForecastPeriods keeps the synopsis and the first n ".PERIOD..." blocks of a
// RAW coastal-waters forecast (RAT-6) — before Normalize, whose labelLine
// rewrites the period tags into prose. Bounded by the lines.
func ForecastPeriods(text string, n int) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	periods := 0
	for _, l := range lines {
		if periodTag.MatchString(strings.TrimSpace(l)) { // normalize.go's own period regex (".TONIGHT...")
			periods++
			if periods > n {
				break
			}
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}
```

The scripts (Task 3.3) supply the sentence frames these data fill; the expected strings in the test are the
built-in scripts' output. **Verify:** `go test ./domains/radio/synth -run Marine -count=1`

---

### Task 3.3 — the maritime scripts (FR-6; the HUM LEAD's words, defaults per MVS-D-21)

Every file's first line is its `{{/* … */}}` doc comment (the convention `script_test.go` pins).

`scripts/maritime-report/head.txt`
```
{{/* maritime-report/head: the notice. Data: .Location. */}}
This is the Maritime report for Watchpost Radio, for {{.Location}}.
```
`forecast.txt`
```
{{/* maritime-report/forecast: the coastal-waters forecast, synopsis and nearshore zone. Data: .Text. */}}
{{.Text}}
```
`observed.txt`
```
{{/* maritime-report/observed: the buoy's provenance. Data: .Distance ("6 miles", or empty), .Ago ("39 minutes"). */}}
Conditions at the nearest buoy{{if .Distance}}, {{.Distance}} offshore{{end}}, observed {{.Ago}} ago.
```
`sea.txt`
```
{{/* maritime-report/sea: sea state, waves and the primary swell. Data: .State ("slight chop"), .Waves ("about 3 feet"), .Swell (a phrase, or empty). */}}
Seas are {{.State}}, with waves of {{.Waves}}{{if .Swell}}, and {{.Swell}}{{end}}.
```
`water.txt`
```
{{/* maritime-report/water: the water temperature. Data: .Temp ("74 degrees"). */}}
Water temperature {{.Temp}}.
```
`wind.txt`
```
{{/* maritime-report/wind: the wind at the buoy. Data: .Speed ("11 miles per hour"), .Gust ("16", or empty). */}}
Wind at the buoy {{.Speed}}{{if .Gust}}, gusting to {{.Gust}}{{end}}.
```
`tide.txt`
```
{{/* maritime-report/tide: the tide in force. Data: .Trend (rising | falling | empty), .Level ("3.7 feet above the low-water mark", or empty), .Station (or empty), .Distance (or empty). */}}
The tide is{{if .Trend}} {{.Trend}}{{end}}{{if .Level}}, {{.Level}}{{end}}{{if .Station}}, at {{.Station}}{{if .Distance}}, {{.Distance}} from you{{end}}{{end}}.
```
`tide-next.txt`
```
{{/* maritime-report/tide-next: the next high and low. Data: .High, .HighLevel, .Low, .LowLevel (each may be empty). */}}
{{if .High}}The next high tide is at {{.High}} at {{.HighLevel}}{{if .Low}}; the next low at {{.Low}}, {{.LowLevel}}{{end}}.{{else if .Low}}The next low tide is at {{.Low}}, {{.LowLevel}}.{{end}}
```
`current.txt`
```
{{/* maritime-report/current: the tidal current. Data: .Phase ("flooding" | "ebbing" | "slack water"), .Speed ("1.4 knots", or empty), .Next (the next phase, or empty), .NextAt (its time). */}}
Tidal currents are {{.Phase}}{{if .Speed}} at {{.Speed}}{{end}}{{if .Next}}, with {{.Next}} at {{.NextAt}}{{end}}.
```
`absence.txt`
```
{{/* maritime-report/absence: no tide or current station in range. No data. */}}
There are no tide or current predictions available for your area.
```
`link.txt`
```
{{/* maritime-report/link: where to learn more. No data. No trailing slash. */}}
For coastal conditions in your area, visit https://www.ndbc.noaa.gov
```

`script_test.go`: both report lists (`:39` and `:114`) gain `maritime-report` after `handover`; `TestTheAppsPartsExist`
gains the eleven `maritime-report/*` parts; the convention test's data map (`:20-26`) gains every field the new
parts name: `"State": "slight chop", "Waves": "about 3 feet", "Swell": "a primary swell from the west at 2 feet", "Temp": "74 degrees", "Speed": "11 miles per hour", "Gust": "16", "Trend": "rising", "Level": "3.7 feet above the low-water mark", "Station": "La Jolla", "High": "12:26 PM", "HighLevel": "5.7 feet", "Low": "7:26 PM", "LowLevel": "just below the low-water mark", "Phase": "flooding", "Next": "slack water", "NextAt": "11:56 AM"`
from P2 Task 2.2. Note: the `sea` frame reads "slight chop"
because `seaSentence` lower-cases the screen's "Slight Chop" and strips "(glassy)" — "Seas are calm" for the
first band. **Verify:** `go test ./domains/radio/... -count=1`

---

### Task 3.4 — the hook: `Assembler.MarineFor`, `marineReportOf`, `marineFor`, `Reports.Maritime`, the order (OQ-12 / MVS-D-18)

**File:** `platform/snapshot/assembler_test.go` (RED — append)

```go
func TestMarineForMergesProvidersWithoutCloningTheSnapshot(t *testing.T) {
	ref := LocationRef{Label: "Oceanside, CA", Lat: 33.2, Lon: -117.4, TZ: "America/Los_Angeles"}
	a := NewAssembler([]LocationRef{ref}, []string{"nws-marine", "ndbc"})
	f := func(v float64) *float64 { return &v }
	a.Apply(Fragment{Provider: "nws-marine", Kind: KindMarine, FetchedAt: time.Now(), PerLocation: map[LocationKey]PartialData{Key(ref): {Marine: &Marine{SwellHeight: f(0.6)}}}})
	a.Apply(Fragment{Provider: "ndbc", Kind: KindMarine, FetchedAt: time.Now(), PerLocation: map[LocationKey]PartialData{Key(ref): {Marine: &Marine{WaterTemp: f(23.3), Buoy: "46224"}}}})
	m, tz, lat, lon, ok := a.MarineFor(ref)
	if !ok || m == nil || m.SwellHeight == nil || m.WaterTemp == nil || m.Buoy != "46224" || tz != "America/Los_Angeles" || lat != 33.2 || lon != -117.4 {
		t.Fatalf("MarineFor: %+v %q %v %v %v", m, tz, lat, lon, ok)
	}
	if _, _, _, _, ok := a.MarineFor(LocationRef{Label: "Boise, ID", Lat: 43.6, Lon: -116.2}); ok {
		t.Fatal("an untracked or inland location is not ok")
	}
}
```

(the `Fragment{Provider, Kind, FetchedAt, PerLocation}` shape `assembler_test.go:15-20` uses.)

**File:** `platform/snapshot/assembler.go` (GREEN)

```go
// MarineFor is the radio deck's narrow read (0.14.0): a location's merged
// coastal-waters block — the same field-wise merge the publisher does
// (mergeMarine, shared with harmonizeMarine) — with its time zone and
// coordinates, without cloning the snapshot per cycle. ok is false when the
// location is not tracked or no provider has a marine block for it (inland).
func (a *Assembler) MarineFor(ref LocationRef) (m *Marine, tz string, lat, lon float64, ok bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	k := Key(ref)
	secs := a.sections[k]
	if secs == nil {
		return nil, "", 0, 0, false
	}
	m = mergeMarine(a.providers, func(id string) *Marine {
		if sec := secs[id]; sec != nil {
			return sec.Marine
		}
		return nil
	})
	if m == nil {
		return nil, "", 0, 0, false
	}
	for _, r := range a.refs {
		if Key(r) == k {
			tz, lat, lon = r.TZ, r.Lat, r.Lon
		}
	}
	return m, tz, lat, lon, true
}
```

**File:** `platform/snapshot/harmonize.go` (GREEN) — the loop `harmonizeMarine` owns today (`:189-203`) becomes
the shared body; `harmonizeMarine` calls it (the second caller rule):

```go
// mergeMarine folds the providers' marine blocks in order: the first non-nil
// is cloned, the rest fill its nil fields (fillMarine — water temperature
// only ever comes from the buoy). nil when no provider has one (inland).
func mergeMarine(order []string, blockOf func(id string) *Marine) *Marine {
	var m *Marine
	for _, id := range order {
		src := blockOf(id)
		if src == nil {
			continue
		}
		if m == nil {
			m = src.Clone()
			continue
		}
		fillMarine(m, src)
	}
	return m
}

// harmonizeMarine merges the coastal-waters section field-wise across
// providers in order. nil stays nil inland.
func harmonizeMarine(loc *Location, order []string) {
	loc.Marine = mergeMarine(order, func(id string) *Marine {
		if sec, ok := loc.ByProvider[id]; ok {
			return sec.Marine
		}
		return nil
	})
}
```

`maxMaritimePieces = 12` (a `const` beside `SpokenPeriodsCap`) bounds the section; `TestMaritimeSectionIsBounded`
feeds a 100 KB forecast and expects ≤ 12 segments.

**File:** `app/marine.go` (GREEN)

```go
package app

import (
	"context"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// marineReportOf builds the broadcast's maritime report (FR-6) from a
// location's merged coastal-waters block: Known only when a provider gave
// one (nil = inland, skipped). The forecast is the office's coastal-waters
// product cut to the nearest zone, fetched by the deck (MVS-D-14).
func marineReportOf(m *snapshot.Marine, tz string, lat, lon float64, forecast string) synth.MarineReport {
	if m == nil {
		return synth.MarineReport{}
	}
	loc := time.UTC
	if z, err := time.LoadLocation(tz); err == nil && tz != "" {
		loc = z
	}
	return synth.MarineReport{Known: true, State: *m, TZ: loc, Lat: lat, Lon: lon, Forecast: forecast}
}

// marineFor is the radio deck's maritime hook (FR-6): the location's
// coastal-waters block from whichever pipeline carries it (favourites first),
// plus the coastal-waters forecast. Inland → the report is skipped.
func (lp *livePipelines) marineFor(ctx context.Context, ref snapshot.LocationRef) synth.MarineReport {
	var asms []*snapshot.Assembler
	if lp.priority != nil {
		asms = append(asms, lp.priority.asm)
	}
	if lp.recent != nil {
		asms = append(asms, lp.recent.asm)
	}
	for _, asm := range asms {
		if m, tz, lat, lon, ok := asm.MarineFor(ref); ok {
			return marineReportOf(m, tz, lat, lon, lp.coastalForecast(ctx, ref))
		}
	}
	return synth.MarineReport{}
}
```

`lp.coastalForecast` is Task 3.5. The deck's hook field: `maritime func(context.Context, snapshot.LocationRef)
synth.MarineReport` in `radioDeck` (`app/radio.go:43-44` beside fire/seismic). `attachRadio` (P2 Task 2.10's
signature) gains a trailing `maritime func(context.Context, snapshot.LocationRef) synth.MarineReport` parameter,
assigns `deck.maritime = maritime`, and the call at `app/dashboard.go:86` passes `lp.marineFor`. Read in `segments()`:

```go
	if d.maritime != nil {
		r.Maritime = d.maritime(ctx, ref)
	}
```

**File:** `domains/radio/synth/compose.go` — `Reports` gains `Maritime MarineReport`; the report loop becomes:

```go
	for _, report := range [][]Segment{
		c.MarineSegments(loc.Label, r.Maritime, imperial, now),   // 0.14.0 (MVS-D-18): after the forecast — the sea continues the weather
		c.FireSegments(loc.Label, r.Fire, imperial, now),         // UAT 114: before the tail; skipped without fire data
		c.SeismicSegments(loc.Label, r.Seismic, imperial, now),   // P4: after the fire report; skipped without seismic entries
	} {
```

**File:** `domains/radio/synth/synth_test.go` — append to `TestReportsAreSeparatedByAir` (its `byKey` closure and
`fire` fixture exist at `:583-591`; `quake(...)` is `seismic_test.go`'s helper):

```go
	// 0.14.0: three reports — maritime between the forecast and fire (MVS-D-18); 2 s between each, 1 s before the tail.
	sr := SeismicReport{Known: true, Lat: 33.2, Lon: -117.4, State: snapshot.SeismicState{AsOf: now, Quakes: []snapshot.Quake{quake(4.2, 30, 8, "NE", 2*time.Hour, now)}}}
	segs = std.Compose(loc, products, now, true, Station{}, Reports{Fire: fire, Seismic: sr, Maritime: coastal(now)})
	first := func(prefix string) int {
		for i, s := range segs {
			if strings.HasPrefix(s.Key, prefix) {
				return i
			}
		}
		return -1
	}
	if !(first("ZFP:") < first("maritime:") && first("maritime:") < first("fire:") && first("fire:") < first("seismic:")) {
		t.Fatalf("order: forecast → maritime → fire → seismic: %v", segmentsText(segs))
	}
	for _, c := range []struct {
		prefix string
		want   time.Duration
	}{{"ZFP:", 2 * time.Second}, {"maritime:", 2 * time.Second}, {"fire:", 2 * time.Second}, {"seismic:", time.Second}} {
		if last := byKey(c.prefix); len(last) == 0 || last[len(last)-1].Pause != c.want {
			t.Fatalf("%s ends with %s of air", c.prefix, c.want)
		}
	}
``` **Verify:**
`go test ./platform/snapshot ./domains/radio/synth ./app -run 'Marine|Air|Reports' -count=1`

---

### Task 3.5 — the coastal-waters forecast for free (MVS-D-14; maritime-report.md §10; AX-7)

**File:** `domains/radio/synth/ugc.go` (append — the zone list a product names, ranges expanded; reuses `splitUGC`/`expandUGC`)

```go
// UGCCodes lists every zone/county code a product's UGC headers name, in
// order, ranges expanded ("PZZ750>775" → each member) — the maritime report's
// zone candidates. Bounded by the blocks.
func UGCCodes(text string) []string {
	var out []string
	seen := map[string]bool{}
	for _, block := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "$$") {
		_, codes, _, ok := splitUGC(block)
		if !ok {
			continue
		}
		block := make([]string, 0, len(codes)) // codes is a set: sort within the block for a stable order
		for code := range codes {
			block = append(block, code)
		}
		sort.Strings(block)
		for _, code := range block {
			if !seen[code] {
				seen[code] = true
				out = append(out, code)
			}
		}
	}
	return out // blocks in document order, codes sorted within a block (the CWF lists the synopsis block first, then the zones nearest the office): the fan-out cap keeps the first 32, never an alphabetical first
}
```

(`sort` is used for the in-block order; `splitUGC` returns the expanded code set — `ugc.go:62` `expandUGC`.) Test: `TestUGCCodesExpandsRangesAndLists` in `ugc_test.go` — `"PZZ750>752-775-301015-"` → `[PZZ750 PZZ751 PZZ752 PZZ775]`.

**File:** `domains/radio/synth/products.go` (append)

```go
// Marine fetches the office's latest Coastal Waters Forecast (CWF) — the
// same endpoint the land products use (maritime-report.md §10); ok is false
// when the office issues none (inland offices).
func (p *Products) Marine(ctx context.Context, office string) (Product, bool, error) {
	return p.latestOf(ctx, office, "CWF")
}
```

**File:** `domains/weather/nws/marinezone_test.go` (RED — an httptest server serving two zone geometries)

```go
func TestMarineZoneForPicksTheNearestNearshoreZone(t *testing.T) {
	cwf := "000\nFZUS56 KSGX 292113\nCWFSGX\n\nPZZ700-301015-\n.Synopsis...\n\n$$\n\nPZZ740-301015-\nCoastal Waters out to 10 nm-\n.TONIGHT...W 10 kt.\n\n$$\n\nPZZ750-301015-\nWaters from 10 to 60 nm-\n.TONIGHT...NW 15 kt.\n\n$$\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/zones/coastal/PZZ740":
			_, _ = w.Write([]byte(`{"geometry":{"type":"Polygon","coordinates":[[[-117.5,33.0],[-117.2,33.0],[-117.2,33.4],[-117.5,33.4],[-117.5,33.0]]]},"properties":{"id":"PZZ740","name":"out to 10 nm"}}`))
		case "/zones/coastal/PZZ750":
			_, _ = w.Write([]byte(`{"geometry":{"type":"Polygon","coordinates":[[[-118.5,32.0],[-117.6,32.0],[-117.6,33.4],[-118.5,33.4],[-118.5,32.0]]]},"properties":{"id":"PZZ750","name":"10 to 60 nm"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	p := newProvider(t, srv.URL)
	zone, synopsis := p.MarineZoneFor(context.Background(), []string{"PZZ700", "PZZ740", "PZZ750"}, 33.2, -117.38)
	if zone != "PZZ740" || synopsis != "PZZ700" {
		t.Fatalf("zone %q synopsis %q", zone, synopsis)
	}
}
```

(`newProvider(t, base)` is the provider tests' helper, `provider_test.go:67`.)

**File:** `domains/weather/nws/marinezone.go` (GREEN)

```go
package nws

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
)

// marineZoneID is a well-formed coastal-waters zone id ("PZZ740") — the only
// shape that may become a URL path element. A compiled regexp, the tree's
// convention (synth/ugc.go ugcLine), not mutable state.
var marineZoneID = regexp.MustCompile(`^[A-Z]{2}Z\d{3}$`)

// MarineZoneFor picks the coastal-waters zone a location's forecast should
// read (maritime-report.md §10, AX-7) from the zones the CWF names (the
// caller expands its UGC headers with synth.UGCCodes — ranges and lists
// included): each zone's polygon is fetched once (24 h), its centroid
// memoised on the Provider (a `centroids map[string][2]float64` field under
// the existing `mu`), and the
// nearest by centroid wins; the synopsis zone (the office's x00/x10 block)
// rides along. Falls back to the first non-synopsis zone when geometry is
// unavailable (AX-7 (B)). "" when the product names no zones.
func (p *Provider) MarineZoneFor(ctx context.Context, codes []string, lat, lon float64) (zone, synopsis string) {
	ctx, cancel := context.WithTimeout(ctx, marineZoneBudget) // the whole fan-out, not each fetch: a slow office never stalls the cycle
	defer cancel()
	seen := map[string]bool{}
	var zones []string
	for _, id := range codes {
		if seen[id] || !marineZoneID.MatchString(id) || len(zones) >= maxMarineZones { // only well-formed ids reach a URL; a CWF names ~10 zones, 32 is the ceiling
			continue
		}
		seen[id] = true
		if strings.HasSuffix(id, "00") || strings.HasSuffix(id, "10") { // the synopsis blocks (PZZ700 / PZZ710…)
			if synopsis == "" {
				synopsis = id
			}
			continue
		}
		zones = append(zones, id)
	}
	if len(zones) == 0 {
		return "", synopsis
	}
	best, bestKM := "", math.MaxFloat64
	for _, id := range zones {
		clat, clon, ok := p.zoneCentroid(ctx, id)
		if !ok {
			continue
		}
		if d := geo.HaversineKM(lat, lon, clat, clon); d < bestKM {
			best, bestKM = id, d
		}
	}
	if best == "" {
		return zones[0], synopsis // AX-7 (B): the first nearshore block after the synopsis
	}
	return best, synopsis
}

// marineZoneBudget bounds one MarineZoneFor call; maxMarineZones its fan-out.
const (
	marineZoneBudget = 20 * time.Second
	maxMarineZones   = 32
)

// zoneCentroid is a coastal zone's polygon centroid (the first ring of the
// first polygon — a MultiPolygon's too). The HTTP cache keeps the geometry a
// day; the centroid itself is memoised on the Provider so a cycle costs no
// JSON decode (a zone polygon is ~100 KB).
func (p *Provider) zoneCentroid(ctx context.Context, id string) (lat, lon float64, ok bool) {
	p.mu.Lock()
	if c, hit := p.centroids[id]; hit {
		p.mu.Unlock()
		return c[0], c[1], true
	}
	p.mu.Unlock()
	lat, lon, ok = p.fetchCentroid(ctx, id)
	if ok {
		p.mu.Lock()
		if p.centroids == nil {
			p.centroids = map[string][2]float64{}
		}
		p.centroids[id] = [2]float64{lat, lon}
		p.mu.Unlock()
	}
	return lat, lon, ok
}

// (Provider gains one field, under its existing mu: `centroids map[string][2]float64 // zone id → centroid, memoised for the process`.)

// fetchCentroid decodes one zone's geometry and averages its outer ring.
// The I/O edge checks its own parameter (P10-07): only a well-formed zone
// id ever becomes a URL element, whoever calls.
func (p *Provider) fetchCentroid(ctx context.Context, id string) (lat, lon float64, ok bool) {
	if err := invariant.Check(marineZoneID.MatchString(id), "nws: a coastal zone id is two letters, Z, three digits"); err != nil {
		return 0, 0, false
	}
	var doc struct {
		Geometry struct {
			Type        string          `json:"type"`
			Coordinates json.RawMessage `json:"coordinates"`
		} `json:"geometry"`
	}
	if _, err := p.client.GetJSON(ctx, fmt.Sprintf("%s/zones/coastal/%s", p.base, id), &doc, httpx.TTL(24*time.Hour)); err != nil {
		return 0, 0, false
	}
	ring := firstRing(doc.Geometry.Type, doc.Geometry.Coordinates)
	if len(ring) == 0 {
		return 0, 0, false
	}
	for _, pt := range ring {
		if len(pt) >= 2 {
			lon += pt[0]
			lat += pt[1]
		}
	}
	n := float64(len(ring))
	return lat / n, lon / n, true
}
```

```go
// firstRing decodes a Polygon's outer ring or a MultiPolygon's first
// polygon's outer ring; nil for anything else.
func firstRing(typ string, raw json.RawMessage) [][]float64 {
	switch typ {
	case "Polygon":
		var poly [][][]float64
		if json.Unmarshal(raw, &poly) == nil && len(poly) > 0 {
			return poly[0]
		}
	case "MultiPolygon":
		var multi [][][][]float64
		if json.Unmarshal(raw, &multi) == nil && len(multi) > 0 && len(multi[0]) > 0 {
			return multi[0][0]
		}
	}
	return nil
}
```

(`encoding/json` imported.)

**File:** `app/marine.go` (append; imports `strings` and `synth`)

```go
// coastalForecast is the office's CWF cut to the location's zone and the
// synopsis, Normalized for the voice; "" when the office issues none.
func (lp *livePipelines) coastalForecast(ctx context.Context, ref snapshot.LocationRef) string {
	d := lp.deck
	if d == nil {
		return ""
	}
	office := d.nws.Office(ctx, ref)
	cwf, ok, err := d.products.Marine(ctx, office)
	if err != nil || !ok {
		return ""
	}
	zone, synopsis := d.nws.MarineZoneFor(ctx, synth.UGCCodes(cwf.Text), ref.Lat, ref.Lon)
	if zone == "" {
		return ""
	}
	return strings.Join(synth.Normalize(synth.ForecastPeriods(synth.FilterUGC(cwf.Text, zone, synopsis), synth.SpokenPeriodsCap)), "\n") // cut BEFORE Normalize: labelLine rewrites ".TONIGHT..." into prose
}
```

`FilterUGC(text, zone, other)` (`ugc.go:20`; its second parameter is renamed from `county` — it is "any other
UGC to keep", which the synopsis id now is) already keeps blocks whose UGC set contains either code — passing
the zone and the synopsis id keeps both blocks. **Verify:**
`go test ./domains/weather/nws -run MarineZone -count=1 && go test ./app -run Coastal -count=1`

---

### Task 3.6 — the other four presets (FR-11; tones.md; AX-5) — on P2 Task 2.0's `Preset`/`Classic()`

**File:** `domains/radio/synth/tone_test.go` (RED — append)

```go
func TestPresetsHaveTheirRatifiedLengths(t *testing.T) {
	const rate = 22050
	ms := func(p Preset) float64 { return float64(len(AlertTone(p, rate))/4) / rate * 1000 }
	for _, c := range []struct {
		p    Preset
		want float64
	}{{Classic(), 2800}, {DualTone(), 4000}, {Watch1050(), 4000}, {Chime(), 3200}, {Sweep(), 3600}} {
		if got := ms(c.p); got < c.want-10 || got > c.want+10 {
			t.Errorf("%s: %.0f ms, want ~%.0f", c.p.Name, got, c.want)
		}
	}
}

func TestEveryRatifiedPresetIsNamed(t *testing.T) {
	for _, name := range []string{"classic", "dual-tone", "1050", "soft-chime", "low-sweep"} {
		if PresetByName(name).Name != name {
			t.Errorf("PresetByName(%q) = %q", name, PresetByName(name).Name)
		}
	}
	if len(Presets()) != 5 {
		t.Fatalf("five presets, got %d", len(Presets()))
	}
}

```

**File:** `domains/radio/synth/tone.go` (GREEN — functions, not package variables, P10-06)

```go
// DualTone is the Warnings / Disasters signal (EAS attention-signal style).
func DualTone() Preset {
	return Preset{Name: "dual-tone", Freqs: []float64{853, 960}, Pulses: 1, PulseDur: 2 * time.Second, Amp: 0.45}
}

// Watch1050 is the Watches signal (NOAA Weather Radio warning-alarm style — the HUM LEAD assigned it to Watches).
func Watch1050() Preset {
	return Preset{Name: "1050", Freqs: []float64{1050}, Pulses: 1, PulseDur: 2 * time.Second, Amp: 0.45}
}

// Chime is the Special Weather Statements signal (a public-address chime).
func Chime() Preset {
	return Preset{Name: "soft-chime", Freqs: []float64{880, 1760}, Pulses: 1, PulseDur: 1200 * time.Millisecond, Decay: 350 * time.Millisecond, Amp: 0.40}
}

// Sweep is the Tropical / Winter Storm signal (a horn-like sweep, twice).
func Sweep() Preset {
	return Preset{Name: "low-sweep", Sweep: [2]float64{330, 520}, Pulses: 2, PulseDur: 700 * time.Millisecond, GapDur: 200 * time.Millisecond, Amp: 0.45}
}

// Presets lists every preset the cast can name (tones.md §1).
func Presets() []Preset { return []Preset{Classic(), DualTone(), Watch1050(), Chime(), Sweep()} }
```

(replace P2's one-entry `Presets()`.) The deck's `tonePCM(class)` (P2 Task 2.7) calls
`AlertTone(PresetByName(cast.ToneName(class)), ToneRate)` — no memo; nothing else changes. **Verify:**
`go test ./domains/radio/synth -run 'Tone|Preset' -count=1`

---

### Task 3.7 — P3 gate (the batch-exit checklist)

```
go test ./domains/... ./platform/... ./app -race -count=2 -timeout 300s
make verify
a2dh validate && make alloc-budget && golangci-lint run ./... && staticcheck ./...   # gates.md §1
A2DH=<framework build> make p10             # the PINNED framework build (gates.md batch record)
cp dist/p10.json 06_docs/02_features/multi-voice-support/07-readiness/p10-p3.json
WATCHPOST_SEISMIC_SOAK=1 go test ./domains/radio/synth -run Soak   # the R6 pattern; soak_test.go:40's Compose gains `Maritime: coastal(now)` (a fixture MarineReport with tides, currents and a three-period forecast) so the hour-long cycle carries the maritime section too
go test ./platform/render ./platform/snapshot ./domains/weather/nws ./app -run DeclarationSet -update-declset   # every package whose exported set moved
git commit -m "multi-voice-support P3: the maritime report (incl. the coastal-waters forecast), the five tone presets and the classifier"
```

- **P10 ledger:** rows only from `dist/p10.json`. Expected: reason refreshes on the `synth`/`nws`/`snapshot`/
  `render` package rows (the sentence helpers carry no invariants by design); `MarineZoneFor`'s loops are all
  `range` and raise nothing. **Build log:**
  `04-development/p3-build-log.md`. No attribution trailers.

UAT-able alone: tune Oceanside — the broadcast reads the coastal forecast, the buoy, the tides and currents
between the forecast and the fire report; a takeover opens with the dual-tone for a warning, the 1050 Hz for
a watch; `[space]` on a Special Weather Statement opens with the chime unless `[M]` is on.
