package tty

import (
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
}

// propChart is the chart's answers (W5.1): the near-vertical area's bands
// and each continent's, each band with its typical day; or why there are
// none, which is ours to fix (D-124).
type propChart struct {
	key     propChartKey
	local   []ionomaps.BandStatus
	targets [][]ionomaps.BandStatus // by propContinents
	err     error
}

// propChartMsg is a chart worked out.
type propChartMsg struct{ chart propChart }

// propChartWanted is the chart the window wants: for the snapshot shown and
// the selected place, in the Propagation mode with the map shown; false
// when there is nothing to work it out from.
func (d Dashboard) propChartWanted() (propChartKey, bool) {
	snap, loc := d.mapPane.propSnap, d.selectedLocation()
	if !d.mapShown() || !d.inPropagation() || snap == nil || len(snap.Hours) == 0 || loc == nil {
		return propChartKey{}, false
	}
	return propChartKey{computed: snap.Computed, hour: snap.Hours[0].At, origin: ionomaps.LatLon{Lat: loc.Lat, Lon: loc.Lon}, radiusKm: float64(d.mapPropRadiusKm)}, true
}

// armPropChart asks for the chart when what it is worked out from has
// changed, once, in a command (FR-1.17): after every Update, as the map's
// clock is armed.
func (d Dashboard) armPropChart() (Dashboard, tea.Cmd) {
	key, ok := d.propChartWanted()
	if !ok || key == d.mapPane.propChart.key || key == d.mapPane.propChartAsked {
		return d, nil
	}
	d.mapPane.propChartAsked = key
	snap := *d.mapPane.propSnap
	return d, func() tea.Msg { return propChartMsg{chart: workPropChart(snap, key)} }
}

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
	for i, t := range propContinents {
		if t.near(key.origin, key.radiusKm) {
			paths[i] = local[0] // no path: the near-vertical area's answer
		}
	}
	c.local, c.targets = local[0], paths
	return c
}

// applyPropChart keeps the chart asked for last; a refused one is said in
// diagnostics (D-124) and the window shows no chart.
func (d Dashboard) applyPropChart(v propChartMsg) Dashboard {
	if v.chart.key != d.mapPane.propChartAsked {
		return d // an older ask's
	}
	d.mapPane.propChart = v.chart
	if v.chart.err != nil {
		d.problem("Propagation: the chart not worked out - " + v.chart.err.Error())
	}
	d.mapPane.gen++
	if d.modal == modalMap {
		return d.renderMap()
	}
	return d
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
		parts[i] = t.short + " " + highestOpen(d.mapPane.propChart.targets[i])
	}
	line := "Up to: " + strings.Join(parts, " · ")
	if keys := d.mapKeys[actMapChart].Keys; len(keys) > 0 {
		line += "   " + keys[0] + " chart"
	}
	return line
}

// propChartWidth is the chart window's width: within the 80-column floor.
const propChartWidth = 78

// The chart's glyphs: open, and above the upper limit; no data is blank.
const (
	propOpenGlyph   = '●'
	propClosedGlyph = '·'
)

// propChartRows are the chart's rows: the near-vertical area, then each
// continent.
func (d Dashboard) propChartRows() (names []string, answers [][]ionomaps.BandStatus) {
	names = []string{fmt.Sprintf("Local, ~%d km", int(d.mapPane.propChart.key.radiusKm))}
	answers = [][]ionomaps.BandStatus{d.mapPane.propChart.local}
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
	if loc := d.selectedLocation(); loc != nil {
		name = strings.ToUpper(loc.Label)
	}
	out := []string{"From " + name + " · " + c.key.hour.UTC().Format("15:04 UTC") + " · upper limits only", ""}
	head := render.PadTo("", propNameW+2)
	for _, b := range hfBands {
		head += fmt.Sprintf("%3s", strings.TrimSuffix(b.name, "m")) // each band's number over its glyph
	}
	out = append(out, head)
	names, answers := d.propChartRows()
	row := min(d.mapPane.propChartRow, len(names)-1)
	for i, bands := range answers {
		pointer := "  "
		if i == row {
			pointer = "▸ "
		}
		line := pointer + render.PadTo(names[i], propNameW) + d.propBandCells(bands)
		if len(bands) > 0 && bands[0].Status != ionomaps.StatusNoData {
			line += " " + propLimitWords(bands[0])
		}
		out = append(out, line)
	}
	out = append(out, string(propOpenGlyph)+" open under the upper limit   "+string(propClosedGlyph)+" above it   blank: no data", "")
	out = append(out, "TYPICAL DAY, UTC: "+names[row])
	if propDiffers(answers[row], c.key.hour) {
		out[len(out)-1] += " · now differs from a typical day"
	}
	out = append(out, d.propDayHead(c.key.hour))
	return append(out, d.propDayRows(answers[row])...)
}

// propNameW is the chart's name column: the longest row name and a space.
const propNameW = 16

// propBandCells are a row's bands now, a glyph each under its band's
// number, three cells a band.
func (d Dashboard) propBandCells(bands []ionomaps.BandStatus) string {
	var b strings.Builder
	for _, s := range bands {
		b.WriteString(" " + d.propGlyph(s.Status == ionomaps.Open, s.Status == ionomaps.StatusNoData) + " ")
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

// propLimitWords name the limit that decided a row and its value (R-2.6).
func propLimitWords(b ionomaps.BandStatus) string {
	name := "MUF"
	if b.Limit == ionomaps.LimitFoF2 {
		name = "foF2"
	}
	return fmt.Sprintf("%s %.1f MHz", name, b.LimitMHz)
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
