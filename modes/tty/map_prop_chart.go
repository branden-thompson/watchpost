package tty

import (
	"fmt"
	"github.com/branden-thompson/watchpost/platform/propanswers"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/term"
)

// actMapChart opens the reference chart over the map (D-176).
const actMapChart term.Action = "map.chart"

// The chart window's own actions (FR-1.14, D-256): moving between its
// targets, rebindable in [keys] as every control in the mode is, in a scope
// of the window's own - the map's arrows pan, and only while the chart is
// open are they its rows.
const (
	actChartPrev term.Action = "chart.prev"
	actChartNext term.Action = "chart.next"
)

// defaultChartKeyMap is the chart window's bindings: up and down, as every
// open window owns its arrows (0.18.0 D-61, D-62).
func defaultChartKeyMap() term.KeyMap {
	return term.KeyMap{
		actChartPrev: {Keys: []string{"up"}, Help: "Previous Target"},
		actChartNext: {Keys: []string{"down"}, Help: "Next Target"},
	}
}

// chartKeysFrom merges the [keys] entries that name the chart's actions
// into its own scope, as mapKeysFrom does for the map's.
func chartKeysFrom(overrides term.KeyMap) (term.KeyMap, error) {
	return scopeKeysFrom(defaultChartKeyMap(), overrides)
}

// chartKeyMap is the chart's bindings as the listener has them; the
// defaults for a hand-built dashboard.
func (d Dashboard) chartKeyMap() term.KeyMap {
	if d.chartKeys == nil {
		return defaultChartKeyMap()
	}
	return d.chartKeys
}

// propChartKey is what the chart's answers are worked out from.
type propChartKey struct {
	made     propMade // the snapshot as kept: a rebuild, or a new noise, is another (L-F3, D-187)
	hour     time.Time
	origin   ionomaps.LatLon
	radiusKm float64
	found    propanswers.Target // the place found, a target beside the continents on the Observer (D-204); zero for none
}

// targets are the chart's targets: the continents, and the place found
// after them (D-204).
func (k propChartKey) targets() []propanswers.Target {
	if k.found.Name == "" {
		return propanswers.Continents
	}
	return append(append([]propanswers.Target(nil), propanswers.Continents...), k.found)
}

// propChart is the chart's answers (W5.1): the near-vertical area's bands
// and each continent's, each band with its typical day; or why there are
// none, which is ours to fix (D-124).
type propChart struct {
	key     propChartKey
	local   ionomaps.Answer
	targets []ionomaps.Answer // by the key's targets: the continents, then the place found
	err     error
}

// propChartMsg is a chart worked out.
type propChartMsg struct{ chart propChart }

// propChartWanted is the chart the window wants: for the snapshot shown and
// the mode's origin (D-193), in the Propagation mode with the map shown;
// false when there is nothing to work it out from.
func (d Dashboard) propChartWanted() (propChartKey, bool) {
	shown, ok := d.propShown()
	origin := d.mapPane.propOrigin
	if !ok || !d.mapShown() || !d.inPropagation() || !origin.ok {
		return propChartKey{}, false
	}
	key := propChartKey{made: d.madeFromHeld(shown), hour: shown.Now.At, origin: origin.at, radiusKm: float64(d.propRadiusKm())}
	if f := d.mapPane.propFound; f != nil && !d.propRole().target {
		key.found = propanswers.Target{Name: f.Label, Short: "FOUND", At: ionomaps.LatLon{Lat: f.Lat, Lon: f.Lon}} // on the Observer a target row; on the Broadcaster the centre is the target (D-204)
	}
	return key, true
}

// armPropChart asks for the chart when what it is worked out from has
// changed, once, in a command (FR-1.17): after every Update, as the map's
// clock is armed. A chart from the map's centre waits for the view to settle.
func (d Dashboard) armPropChart() (Dashboard, tea.Cmd) {
	if d.viewSettling() && d.propRole().origin == propFromCentre {
		return d, nil
	}
	key, ok := d.propChartWanted()
	cmd := askAnswer(&d, key, ok, d.mapPane.propChart.key, &d.mapPane.propChartAsked, func(snap ionomaps.Snapshot, key propChartKey) tea.Msg {
		return propChartMsg{chart: workPropChart(snap, key)}
	})
	return d, cmd
}

// viewSettling is whether the view or the hour is still moving: an answer
// from or to the map's centre, and the reach, are asked once it settles, not
// at every step (P-1, D-227).
func (d Dashboard) viewSettling() bool { return d.mapPane.viewGen != d.mapPane.viewAsked }

// applyPropAnswer keeps an answer when it is to the ask made last - an older
// ask's is dropped - and draws it; a reach is handed to the map as well.
func (d Dashboard) applyPropAnswer(msg tea.Msg) Dashboard {
	switch v := msg.(type) {
	case propChartMsg:
		if v.chart.key != d.mapPane.propChartAsked {
			return d
		}
		d.mapPane.propChart = v.chart
		return d.answered(v.chart.err, "the chart not worked out")
	case propReachMsg:
		if v.reach.key != d.mapPane.propReachAsked {
			return d
		}
		d.mapPane.propReach = v.reach
		return d.setProp().answered(v.reach.err, "the reach not worked out")
	case propCentreMsg:
		if v.centre.key != d.mapPane.propCentreAsked {
			return d
		}
		d.mapPane.propCentre = v.centre
		return d.answered(v.centre.err, "the centre not answered")
	}
	return d
}

// askAnswer is how each of the Propagation mode's answers - the chart, the
// reach, the centre - is asked for: in a command, never in Update
// (FR-1.17), when what it is worked out from has changed, and once; the
// snapshot on screen is copied for it only then (P-5).
func askAnswer[K comparable](d *Dashboard, wanted K, ok bool, held K, asked *K, work func(ionomaps.Snapshot, K) tea.Msg) tea.Cmd {
	if !ok || wanted == held || wanted == *asked {
		return nil
	}
	*asked = wanted
	snap := d.propOnScreen()
	return func() tea.Msg { return work(snap, wanted) }
}

// answered draws an answer kept, and says a refused one in diagnostics
// (D-124), never to the listener.
func (d Dashboard) answered(err error, what string) Dashboard {
	if err != nil {
		d.problem("Propagation: " + what + " - " + err.Error())
	}
	d.mapPane.gen++
	if d.modal == modalMap {
		return d.renderMap()
	}
	return d
}

// workPropChart works the chart out off the UI goroutine (W5.1): the answers
// package's, for the key's origin, radius and targets (D-212).
func workPropChart(snap ionomaps.Snapshot, key propChartKey) propChart {
	got, err := propanswers.WorkChart(snap, propanswers.Ask{Origin: key.origin, RadiusKm: key.radiusKm, Targets: key.targets()})
	return propChart{key: key, local: got.Local, targets: got.Targets, err: err}
}

// propChartShown reports whether the chart answers for what the window
// shows now.
func (d Dashboard) propChartShown() bool {
	key, ok := d.propChartWanted()
	c := d.mapPane.propChart
	return ok && key == c.key && c.err == nil && len(c.targets) == len(key.targets())
}

// propChartFailed reports whether the chart wanted could not be worked out.
func (d Dashboard) propChartFailed() bool {
	key, ok := d.propChartWanted()
	c := d.mapPane.propChart
	return ok && key == c.key && c.err != nil
}

// propNotAvailable is what is said for an answer that could not be worked
// out (AX-9): not "being worked out" for ever, and the key that asks again
// (D-124: never an error without a way to resolve it).
func (d Dashboard) propNotAvailable() string {
	return "not available; " + d.firstKey(actMapRefresh) + " tries again"
}

// propBestLine is the block's line for the continents (D-176): each one's
// highest open band, and the key that opens the chart.
func (d Dashboard) propBestLine() string {
	if !d.propChartShown() {
		return ""
	}
	parts := make([]string, len(propanswers.Continents))
	for i, t := range propanswers.Continents {
		parts[i] = t.Short + " " + propanswers.HighestOpen(d.mapPane.propChart.targets[i].Bands)
	}
	line := "Up to: " + strings.Join(parts, " · ")
	if keys := d.mapKeys[actMapChart].Keys; len(keys) > 0 {
		line += "   " + keys[0] + " chart"
	}
	return line
}

// propChartWidth is the chart window's width: within the 80-column floor.
const propChartWidth = 78

// The chart's glyphs (D-192): open, disturbed (open, its lower edge under
// D-RAP's 1 dB frequency), absorbed and weak (closed by the lower limit:
// the day's absorption, or too faint even without it), and above the upper
// limit; no data is blank.
const (
	propOpenGlyph      = '●'
	propDisturbedGlyph = '◐'
	propAbsorbedGlyph  = '○'
	propWeakGlyph      = '◌'
	propClosedGlyph    = '·'
	propNowGlyph       = '▼' // the typical day's hour shown
)

// propASCIIMarks are the chart's glyphs under --ascii (AX-4): letters a
// screen reader says, where the circles are read as their Unicode names;
// above the upper limit a dash, the dot being the station shade's.
var propASCIIMarks = map[rune]string{propOpenGlyph: "O", propDisturbedGlyph: "D", propAbsorbedGlyph: "A", propWeakGlyph: "W", propClosedGlyph: "-", propNowGlyph: "v"}

// mark is a chart glyph as the window draws it: under --ascii, its letter.
func (d Dashboard) mark(g rune) string {
	if d.cfg.ASCII {
		return propASCIIMarks[g]
	}
	return string(g)
}

// propChartRows are the chart's rows: the near-vertical area, then each
// continent.
func (d Dashboard) propChartRows() (names []string, rows []ionomaps.Answer) {
	names = []string{fmt.Sprintf("Local, ~%d km", int(d.mapPane.propChart.key.radiusKm))}
	rows = []ionomaps.Answer{d.mapPane.propChart.local}
	for i, t := range d.mapPane.propChart.key.targets() {
		if i < len(d.mapPane.propChart.targets) {
			names, rows = append(names, t.Name), append(rows, d.mapPane.propChart.targets[i])
		}
	}
	return names, rows
}

// propChartLines are the chart window's lines (D-176): the origin and the
// hour; each row's bands now and its limit; the selected row's typical day,
// band by hour, and whether now differs from it (D-135).
func (d Dashboard) propChartLines(o render.Opts) []string {
	if d.propChartFailed() {
		return insetModalLines([]string{"The chart is " + d.propNotAvailable() + "."}, debugProseWidth(o, propChartWidth))
	}
	if !d.propChartShown() {
		return insetModalLines([]string{"The chart is being worked out."}, debugProseWidth(o, propChartWidth))
	}
	c := d.mapPane.propChart
	name := "the selected place"
	if o := d.mapPane.propOrigin; o.ok {
		name = strings.ToUpper(o.name)
	}
	out := []string{"From " + name + " · " + c.key.hour.UTC().Format("15:04 UTC") + " · 100 W, 0 dBi, +13 dB · " + strings.ToLower(propNoiseLabel(c.key.made.opts.Noise)) + " noise", ""}
	if s := d.mapPane.propSnap; s != nil && propOffsetNote(s.Offset) != "" {
		out[1] = propOffsetNote(s.Offset) // FR-5.4: the chart's status label, under its header
	}
	if s := d.mapPane.propSnap; s != nil {
		if l := propScalesLine(s.Scales); l != "" {
			out = append(out[:1], l, out[1]) // FR-1.18: under the header, before the offset's row
		}
	}
	head := render.PadTo("", propNameW+4) // the pointer and the shade (D-198) before the names
	for _, b := range hfBands {
		head += fmt.Sprintf("%3s", strings.TrimSuffix(b.name, "m")) // each band's number over its glyph
	}
	out = append(out, head)
	names, rows := d.propChartRows()
	row := min(d.mapPane.propChartRow, len(names)-1)
	for i, a := range rows {
		pointer := "  "
		if i == row {
			pointer = o.Glyphs().Pointer + " "
		}
		line := pointer + propShade(a.Nearest, o.ASCII) + " " + render.PadTo(names[i], propNameW) + d.propBandCells(a.Bands)
		if w := propanswers.RowWords(a.Bands); w != "" {
			line += " " + w
		}
		out = append(out, line)
	}
	out = append(out, d.mark(propOpenGlyph)+" open   "+d.mark(propDisturbedGlyph)+" disturbed (D-RAP)   "+d.mark(propAbsorbedGlyph)+" absorbed   "+d.mark(propWeakGlyph)+" weak",
		d.mark(propClosedGlyph)+" above the upper limit   blank: no data   figures in MHz",
		propShadeLegend(o.ASCII), // D-198
		propAgreementLine)        // D-330
	if f := d.propFreqLine(); f != "" {
		out = append(out, f) // D-178: what is held
	}
	out = append(out, "")
	out = append(out, "TYPICAL DAY, UTC: "+names[row])
	if propanswers.Differs(rows[row].Bands, c.key.hour) {
		out[len(out)-1] += " · now differs from a typical day"
	}
	out = append(out, d.propDayHead(c.key.hour))
	out = append(out, d.propDayRows(rows[row].Bands)...)
	if d.cfg.ASCII {
		for i := range out {
			out[i] = strings.ReplaceAll(out[i], " · ", " "+o.Glyphs().Dot+" ") // the separators too (AX-4)
		}
	}
	return append(out, "", d.propChartControls(o))
}

// propChartControls is the chart window's controls line (AX-6), as every
// window has: the arrows choose the row whose typical day is drawn, PgUp and
// PgDn scroll, Esc or the chart's key closes it.
func (d Dashboard) propChartControls(o render.Opts) string {
	closeKeys := "esc"
	if k := d.firstKey(actMapChart); k != "" {
		closeKeys = k + "/esc"
	}
	return "  " + o.Controls("   ", render.Ctl("↑↓", "Row"), render.Ctl("pgup/pgdn", "Scroll"), render.Ctl(closeKeys, "Close"))
}

// The nearest reporting station's distances the shades break at (D-116,
// D-198): beyond the far one an answer is about as good as typical, or
// worse (D-129), the legend's "~typical".
const (
	propNearKm = 500
	propFarKm  = 2000
)

// propShade is a chart row's nearest reporting station as a shade (D-198),
// go-tuiMaps' own no-colour key: within 500 km, 500 to 2000 km, farther or
// none; ? where the nearest is unscored. Under --ascii, #, + and . for them.
func propShade(n ionomaps.Nearest, ascii bool) string {
	shades := [3]string{"▓", "▒", "░"}
	if ascii {
		shades = [3]string{"#", "+", "."}
	}
	switch {
	case !n.Known:
		return shades[2]
	case n.Unscored:
		return "?"
	case n.Km < propNearKm:
		return shades[0]
	case n.Km < propFarKm:
		return shades[1]
	}
	return shades[2]
}

// propShadeLegend is the shades' key (D-198).
func propShadeLegend(ascii bool) string {
	near, mid, far := propShade(ionomaps.Nearest{Known: true}, ascii), propShade(ionomaps.Nearest{Known: true, Km: propNearKm}, ascii), propShade(ionomaps.Nearest{}, ascii)
	return "nearest station: " + near + " <500 km   " + mid + " 500-2000 km   " + far + " farther, ~typical   ? unscored"
}

// propAgreementLine is how often the chart's answers agreed with what WSPR
// heard (D-330): the graded run on the fresh days (D-221, D-328), each band
// drawn half open and half not, so a constant answer scores 50%. The Local
// row is M1's scenario and the continents M4's; the climatology alone beside
// them.
const propAgreementLine = "agrees with WSPR reports: Local 54%, continents 68% (typical alone 52%, 64%)"

// propNameW is the chart's name column: the longest row name and a space.
const propNameW = 16

// propBandCells are a row's bands now, a glyph each under its band's
// number, three cells a band: each band found by the band it names, never
// by its place in the answer, a band the answer lacks left blank.
func (d Dashboard) propBandCells(bands []ionomaps.BandStatus) string {
	var b strings.Builder
	for _, hb := range hfBands {
		s, found := ionomaps.BandStatus{}, false
		for _, a := range bands {
			if a.Band.String() == hb.name {
				s, found = a, true
				break
			}
		}
		if !found {
			b.WriteString("   ")
			continue
		}
		g := d.propGlyph(s.Status == ionomaps.Open, s.Status == ionomaps.StatusNoData)
		switch s.Status {
		case ionomaps.Disturbed:
			g = render.Tint(d.mark(propDisturbedGlyph), render.Tok(render.NameAdvisory))
		case ionomaps.Absorbed:
			g = render.Tint(d.mark(propAbsorbedGlyph), render.Tok(render.KeyChipMuted))
		case ionomaps.Weak:
			g = render.Tint(d.mark(propWeakGlyph), render.Tok(render.KeyChipMuted))
		}
		b.WriteString(" " + g + " ")
	}
	return b.String()
}

// propGlyph is a band's mark: open in the providers' working tone, above the
// upper limit muted, no data blank.
func (d Dashboard) propGlyph(open, noData bool) string {
	switch {
	case noData:
		return " "
	case open:
		return render.Tint(d.mark(propOpenGlyph), render.Tok(render.ProviderOK))
	}
	return render.Tint(d.mark(propClosedGlyph), render.Tok(render.KeyChipMuted))
}

// propDayHead marks the typical day's hours every three, and the hour shown.
func (d Dashboard) propDayHead(hour time.Time) string {
	var b strings.Builder
	b.WriteString(render.PadTo("", 6))
	now := hour.UTC().Hour()
	for h := range 24 {
		switch {
		case h == now:
			b.WriteString(d.mark(propNowGlyph) + " ")
		case h%3 == 0:
			b.WriteString(render.PadTo(fmt.Sprintf("%02d", h), 2))
		default:
			b.WriteString("  ")
		}
	}
	return strings.TrimRight(b.String(), " ")
}

// propDayRows are a target's typical day, a band a row, an hour a column:
// open where the band is typically open that hour (D-135).
func (d Dashboard) propDayRows(bands []ionomaps.BandStatus) []string {
	out := make([]string, 0, len(bands))
	for _, s := range bands {
		var b strings.Builder
		b.WriteString(render.PadTo(s.Band.String(), 6))
		day := d.mapPane.propChart.key.hour.UTC().Truncate(24 * time.Hour)
		for h := range 24 {
			b.WriteString(d.propGlyph(propanswers.OpenAt(s.OpenHours, day.Add(time.Duration(h)*time.Hour)), false))
			if h < 23 {
				b.WriteString(" ")
			}
		}
		out = append(out, b.String())
	}
	return out
}

// openPropChart is B: the chart over the map, in the Propagation mode alone.
func (d Dashboard) openPropChart() Dashboard {
	if !d.inPropagation() {
		return d
	}
	return d.open(modalPropChart)
}

// inPropagation reports whether the map is in the Propagation mode, every
// mode named (W2.0).
func (d Dashboard) inPropagation() bool {
	switch d.mapMode() {
	case modePropagation:
		return true
	case modeRadar, modeForecast:
	}
	return false
}

// handlePropChartKey is the chart window's keyboard: Esc or the chart's key
// closes it, back to the map; its own previous and next target keys - the
// arrows by default (D-256) - move the target its day is drawn for; the page
// keys scroll; every other key does nothing.
func (d Dashboard) handlePropChartKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := key.String()
	if act, ok := d.mapKeys.Lookup(k); k == "esc" || (ok && act == actMapChart) {
		return d.close(), nil
	}
	if act, ok := d.chartKeyMap().Lookup(k); ok && act == actChartPrev {
		d.mapPane.propChartRow = max(d.mapPane.propChartRow-1, 0)
		return d, nil
	}
	if act, ok := d.chartKeyMap().Lookup(k); ok && act == actChartNext {
		d.mapPane.propChartRow = min(d.mapPane.propChartRow+1, len(d.mapPane.propChart.key.targets()))
		return d, nil
	}
	switch k {
	case "pgup":
		return d.handleModalNav("nav-up"), nil
	case "pgdown":
		return d.handleModalNav("nav-down"), nil
	}
	return d, nil
}
