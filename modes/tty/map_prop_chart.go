package tty

import (
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	ionomaps "github.com/branden-thompson/go-ionomaps"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/term"
)

// actMapChart opens the reference chart over the map (D-176).
const actMapChart term.Action = "map.chart"

// propTarget is a place the chart answers for.
type propTarget struct {
	name, short string
	at          ionomaps.LatLon
}

// propContinents are the chart's default targets (FR-1.9, A-45): on each
// inhabited continent, the city where its amateur radio is busiest.
var propContinents = []propTarget{
	{"North America", "NA", ionomaps.LatLon{Lat: 41.88, Lon: -87.63}},  // Chicago
	{"South America", "SA", ionomaps.LatLon{Lat: -23.55, Lon: -46.63}}, // São Paulo
	{"Europe", "EU", ionomaps.LatLon{Lat: 50.11, Lon: 8.68}},           // Frankfurt
	{"Africa", "AF", ionomaps.LatLon{Lat: -26.20, Lon: 28.05}},         // Johannesburg
	{"Asia", "AS", ionomaps.LatLon{Lat: 35.68, Lon: 139.69}},           // Tokyo
	{"Oceania", "OC", ionomaps.LatLon{Lat: -33.87, Lon: 151.21}},       // Sydney
}

// near reports whether a target is within the near-vertical area around an
// origin: no path to it, its answer the area's.
func (t propTarget) near(origin ionomaps.LatLon, radiusKm float64) bool {
	return geo.HaversineKM(origin.Lat, origin.Lon, t.at.Lat, t.at.Lon) <= radiusKm
}

// propChartKey is what the chart's answers are worked out from.
type propChartKey struct {
	computed, hour time.Time
	origin         ionomaps.LatLon
	radiusKm       float64
	noise          ionomaps.Noise // the snapshot's: a new noise rebuilds it at the same time (D-187)
}

// propChart is the chart's answers (W5.1): the near-vertical area's bands
// and each continent's, each band with its typical day; or why there are
// none, which is ours to fix (D-124).
type propChart struct {
	key     propChartKey
	local   ionomaps.Answer
	targets []ionomaps.Answer // by propContinents
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
	return propChartKey{computed: shown.Computed, hour: shown.Hours[0].At, origin: origin.at, radiusKm: float64(d.propRadiusKm()), noise: shown.Noise}, true
}

// armPropChart asks for the chart when what it is worked out from has
// changed, once, in a command (FR-1.17): after every Update, as the map's
// clock is armed.
func (d Dashboard) armPropChart() (Dashboard, tea.Cmd) { return d.armAnswer(answerChart) }

// propAnswer names one of the Propagation mode's answers worked out off the
// UI goroutine.
type propAnswer uint8

// The answers.
const (
	answerChart  propAnswer = iota // the chart (W5.1)
	answerReach                    // the frequency's reach (W5.2)
	answerCentre                   // the map centre's (W5.4)
)

// armAnswer asks for one answer when what it is worked out from has
// changed, once, in a command (FR-1.17), from the snapshot on screen.
func (d Dashboard) armAnswer(which propAnswer) (Dashboard, tea.Cmd) {
	snap := d.propOnScreen()
	var cmd tea.Cmd
	switch which {
	case answerChart:
		key, ok := d.propChartWanted()
		cmd = askAnswer(key, ok, d.mapPane.propChart.key, &d.mapPane.propChartAsked, func() tea.Msg { return propChartMsg{chart: workPropChart(snap, key)} })
	case answerReach:
		key, ok := d.propReachWanted()
		cmd = askAnswer(key, ok, d.mapPane.propReach.key, &d.mapPane.propReachAsked, func() tea.Msg {
			field, err := snap.Reach(key.origin, key.mhz, key.hour)
			return propReachMsg{reach: propReach{key: key, field: field, err: err}}
		})
	case answerCentre:
		key, ok := d.propCentreWanted()
		cmd = askAnswer(key, ok, d.mapPane.propCentre.key, &d.mapPane.propCentreAsked, func() tea.Msg { return propCentreMsg{centre: workPropCentre(snap, key)} })
	}
	return d, cmd
}

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
// (FR-1.17), when what it is worked out from has changed, and once.
func askAnswer[K comparable](wanted K, ok bool, held K, asked *K, work tea.Cmd) tea.Cmd {
	if !ok || wanted == held || wanted == *asked {
		return nil
	}
	*asked = wanted
	return work
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

// errPropAnswers is an answer of another length than the places asked:
// the library's to fix, said in diagnostics (D-124).
var errPropAnswers = errors.New("go-ionomaps answered a different number of places than were asked")

// workPropChart works the chart out: off the UI goroutine, about 5 ms.
func workPropChart(snap ionomaps.Snapshot, key propChartKey) propChart {
	c := propChart{key: key}
	local, err := snap.Bands([]ionomaps.LatLon{key.origin}, key.radiusKm, key.hour)
	if err != nil {
		c.err = err
		return c
	}
	to := make([]ionomaps.LatLon, len(propContinents))
	for i, t := range propContinents {
		to[i] = t.at
	}
	paths, err := snap.Path(key.origin, to, key.hour)
	if err != nil {
		c.err = err
		return c
	}
	if len(local) != 1 || len(paths) != len(propContinents) {
		c.err = errPropAnswers
		return c
	}
	for i, t := range propContinents {
		if t.near(key.origin, key.radiusKm) {
			paths[i] = local[0] // no path: the near-vertical area's answer
		}
	}
	c.local, c.targets = local[0], paths
	return c
}

// propChartShown reports whether the chart answers for what the window
// shows now.
func (d Dashboard) propChartShown() bool {
	key, ok := d.propChartWanted()
	c := d.mapPane.propChart
	return ok && key == c.key && c.err == nil && len(c.targets) == len(propContinents)
}

// highestOpen is the highest band a target's answer calls open, by name;
// "none", or "no data" where it has none.
func highestOpen(bands []ionomaps.BandStatus) string {
	best, data := "none", false
	for i, b := range bands {
		data = data || b.Status != ionomaps.StatusNoData
		if b.Status == ionomaps.Open && i < len(hfBands) {
			best = hfBands[i].name
		}
	}
	if !data {
		return "no data"
	}
	return best
}

// propBestLine is the block's line for the continents (D-176): each one's
// highest open band, and the key that opens the chart.
func (d Dashboard) propBestLine() string {
	if !d.propChartShown() {
		return ""
	}
	parts := make([]string, len(propContinents))
	for i, t := range propContinents {
		parts[i] = t.short + " " + highestOpen(d.mapPane.propChart.targets[i].Bands)
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
)

// propChartRows are the chart's rows: the near-vertical area, then each
// continent.
func (d Dashboard) propChartRows() (names []string, answers []ionomaps.Answer) {
	names = []string{fmt.Sprintf("Local, ~%d km", int(d.mapPane.propChart.key.radiusKm))}
	answers = []ionomaps.Answer{d.mapPane.propChart.local}
	for i, t := range propContinents {
		names, answers = append(names, t.name), append(answers, d.mapPane.propChart.targets[i])
	}
	return names, answers
}

// propChartLines are the chart window's lines (D-176): the origin and the
// hour; each row's bands now and its limit; the selected row's typical day,
// band by hour, and whether now differs from it (D-135).
func (d Dashboard) propChartLines(o render.Opts) []string {
	if !d.propChartShown() {
		return insetModalLines([]string{"The chart is being worked out."}, debugProseWidth(o, propChartWidth))
	}
	c := d.mapPane.propChart
	name := "the selected place"
	if o := d.mapPane.propOrigin; o.ok {
		name = strings.ToUpper(o.name)
	}
	out := []string{"From " + name + " · " + c.key.hour.UTC().Format("15:04 UTC") + " · 100 W, 0 dBi, +13 dB · " + strings.ToLower(propNoiseLabel(c.key.noise)) + " noise", ""}
	head := render.PadTo("", propNameW+2)
	for _, b := range hfBands {
		head += fmt.Sprintf("%3s", strings.TrimSuffix(b.name, "m")) // each band's number over its glyph
	}
	out = append(out, head)
	names, answers := d.propChartRows()
	row := min(d.mapPane.propChartRow, len(names)-1)
	var unscored []string
	for i, a := range answers {
		pointer := "  "
		if i == row {
			pointer = "▸ "
		}
		line := pointer + render.PadTo(names[i], propNameW) + d.propBandCells(a.Bands)
		if w := propRowLimitWords(a.Bands); w != "" {
			line += " " + w
		}
		out = append(out, line)
		if a.Nearest.Known && a.Nearest.Unscored {
			unscored = append(unscored, names[i])
		}
	}
	out = append(out, string(propOpenGlyph)+" open   "+string(propDisturbedGlyph)+" disturbed (D-RAP)   "+string(propAbsorbedGlyph)+" absorbed   "+string(propWeakGlyph)+" weak",
		string(propClosedGlyph)+" above the upper limit   blank: no data")
	if len(unscored) > 0 {
		out = append(out, "Nearest station unscored, verify on air: "+strings.Join(unscored, ", ")) // D-181
	}
	if f := d.propFreqLine(); f != "" {
		out = append(out, f) // D-178: what is held
	}
	out = append(out, "")
	out = append(out, "TYPICAL DAY, UTC: "+names[row])
	if propDiffers(answers[row].Bands, c.key.hour) {
		out[len(out)-1] += " · now differs from a typical day"
	}
	out = append(out, d.propDayHead(c.key.hour))
	return append(out, d.propDayRows(answers[row].Bands)...)
}

// propNameW is the chart's name column: the longest row name and a space.
const propNameW = 16

// propBandCells are a row's bands now, a glyph each under its band's
// number, three cells a band.
func (d Dashboard) propBandCells(bands []ionomaps.BandStatus) string {
	var b strings.Builder
	for _, s := range bands {
		g := d.propGlyph(s.Status == ionomaps.Open, s.Status == ionomaps.StatusNoData)
		switch s.Status {
		case ionomaps.Disturbed:
			g = render.Tint(string(propDisturbedGlyph), render.Tok(render.NameAdvisory))
		case ionomaps.Absorbed:
			g = render.Tint(string(propAbsorbedGlyph), render.Tok(render.KeyChipMuted))
		case ionomaps.Weak:
			g = render.Tint(string(propWeakGlyph), render.Tok(render.KeyChipMuted))
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
		return render.Tint(string(propOpenGlyph), render.Tok(render.ProviderOK))
	}
	return render.Tint(string(propClosedGlyph), render.Tok(render.KeyChipMuted))
}

// propRowLimitWords name a row's limits (D-192): its upper limit, from a
// band it decided; the LUF when the lower limit closes any band; and D-RAP's
// 1 dB frequency when it disturbs any; empty with no data.
func propRowLimitWords(bands []ionomaps.BandStatus) string {
	l := limitsOf(bands)
	upper := "MUF"
	for _, b := range bands {
		if b.Limit == ionomaps.LimitFoF2 {
			upper = "foF2"
		}
	}
	var parts []string
	for _, w := range []struct {
		name string
		mhz  float64
	}{{upper, l.upper}, {"LUF", l.luf}, {"D-RAP", l.drap}} {
		if w.mhz > 0 {
			parts = append(parts, w.name+" "+mhzWords(w.mhz)+" MHz")
		}
	}
	return strings.Join(parts, " · ")
}

// propDayHead marks the typical day's hours every three, and the hour shown.
func (d Dashboard) propDayHead(hour time.Time) string {
	var b strings.Builder
	b.WriteString(render.PadTo("", 6))
	now := hour.UTC().Hour()
	for h := range 24 {
		switch {
		case h == now:
			b.WriteString("▼ ")
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
	for i, s := range bands {
		var b strings.Builder
		b.WriteString(render.PadTo(hfBands[min(i, len(hfBands)-1)].name, 6))
		day := d.mapPane.propChart.key.hour.UTC().Truncate(24 * time.Hour)
		for h := range 24 {
			b.WriteString(d.propGlyph(openAt(s.OpenHours, day.Add(time.Duration(h)*time.Hour)), false))
			if h < 23 {
				b.WriteString(" ")
			}
		}
		out = append(out, b.String())
	}
	return out
}

// openAt reports whether a moment falls in one of the spans.
func openAt(spans []ionomaps.Span, at time.Time) bool {
	for _, s := range spans {
		if !at.Before(s.From) && at.Before(s.To) {
			return true
		}
	}
	return false
}

// propDiffers reports whether any band's status now differs from its typical
// day at the hour (D-135); a band with no data now is no difference.
func propDiffers(bands []ionomaps.BandStatus, hour time.Time) bool {
	at := hour.UTC().Truncate(time.Hour)
	for _, b := range bands {
		if b.Status != ionomaps.StatusNoData && (b.Status == ionomaps.Open) != openAt(b.OpenHours, at) {
			return true
		}
	}
	return false
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
// closes it, back to the map; the arrows move the target its day is drawn
// for; the page keys scroll; every other key does nothing.
func (d Dashboard) handlePropChartKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := key.String()
	if act, ok := d.mapKeys.Lookup(k); k == "esc" || (ok && act == actMapChart) {
		return d.close(), nil
	}
	switch k {
	case "up":
		d.mapPane.propChartRow = max(d.mapPane.propChartRow-1, 0)
	case "down":
		d.mapPane.propChartRow = min(d.mapPane.propChartRow+1, len(propContinents))
	case "pgup":
		return d.handleModalNav("nav-up"), nil
	case "pgdown":
		return d.handleModalNav("nav-down"), nil
	}
	return d, nil
}
