package tty

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	ionomaps "github.com/branden-thompson/go-ionomaps"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/term"
)

// The Propagation mode's controls (FR-1.14): L flips the layer (D-155), U
// asks for an update now (D-156).
const (
	actMapLayer   term.Action = "map.layer"
	actMapRefresh term.Action = "map.refresh"
)

// The Propagation mode's layers (D-24): MUF(3000), or foF2.
const (
	propMUF = iota
	propFoF2
)

// propLayerName is a layer as the chips, the legend and the words name it.
func propLayerName(layer int) string {
	if layer == propFoF2 {
		return "foF2"
	}
	return "MUF(3000)"
}

// fieldGrid is a go-ionomaps field as go-tuiMaps draws it: its box, its
// cells, a cell without data no value at all (NaN), over the sea too
// (FR-1.6, D-32).
func fieldGrid(f ionomaps.Field) tuimaps.Grid {
	values := make([]float64, len(f.Values))
	for i, v := range f.Values {
		values[i] = float64(v)
		if f.NoData.Has(i) {
			values[i] = math.NaN()
		}
	}
	return tuimaps.Grid{West: f.West, South: f.South, East: f.East, North: f.North, Cols: f.Cols, Rows: f.Rows,
		Values: values, Lines: true, OverWater: true}
}

// propOverlays are the Propagation mode's overlays: the hour on screen, its
// layer alone (D-112), and the night with its edge for the moment shown;
// none outside the mode or before a snapshot.
func (d Dashboard) propOverlays() []tuimaps.Overlay {
	shown := d.propOnScreen()
	if !d.inPropagation() || !propHasField(shown) {
		return nil
	}
	h := shown.Now
	field := tuimaps.MUFGrid("propagation/muf", fieldGrid(h.MUF3000), h.At)
	if d.mapPane.propLayer == propFoF2 {
		field = tuimaps.FoF2Grid("propagation/fof2", fieldGrid(h.FoF2), h.At)
	}
	out := []tuimaps.Overlay{field}
	if reach, ok := d.propReachShown(); ok {
		out = append(out, propReachOverlay(reach, h.At)) // after the field: its edge alone over it (L-2.6)
	}
	return out // the night is the map's, drawn in every mode that wants it (D-199)
}

// propNightMoment is the moment the night is drawn for: the field's own
// time, to the step - the sun as it stood when the field was made, however
// old the field (W3.1); an hour ahead, the middle of that hour; the clock
// before any field.
func (d Dashboard) propNightMoment() time.Time {
	shown, ok := d.propShown()
	switch {
	case !ok:
		return d.now().UTC().Truncate(nightStep)
	case d.mapPane.propAhead > 0:
		return shown.Now.At.Add(30 * time.Minute).UTC()
	}
	return shown.Now.At.UTC().Truncate(nightStep)
}

// setProp hands the library the Propagation mode's overlays as they stand,
// and takes off the ones no longer wanted.
func (d Dashboard) setProp() Dashboard {
	if d.mapPane.m == nil {
		return d
	}
	given, _ := d.reconcile(d.mapPane.propGiven, d.propOverlays(), func(o tuimaps.Overlay, err error) {
		d.problem("Propagation: " + o.ID + " not drawn - " + err.Error()) // ours to fix, never the listener's (D-124)
	})
	d.mapPane.propGiven = given
	d.mapPane.gen++
	return d.setNight() // the night with the mode: always in it, by the detail switch out of it (D-199)
}

// flipPropLayer is L: MUF(3000) or foF2, in the Propagation mode alone.
func (d Dashboard) flipPropLayer() Dashboard {
	if d.mapMode() == modePropagation {
		d.mapPane.propLayer = 1 - d.mapPane.propLayer
		return d.setProp().renderMap()
	}
	return d
}

// refreshProp is U: an update now, in the Propagation mode alone (as every
// ask is), under the library's throttle (D-120): an early ask is answered
// from the last field with its reason.
func (d Dashboard) refreshProp() (Dashboard, tea.Cmd) {
	if d.mapPane.propChart.err != nil { // an answer that failed is asked again with the update (AX-9)
		d.mapPane.propChart, d.mapPane.propChartAsked = propChart{}, propChartKey{}
	}
	if d.mapPane.propReach.err != nil {
		d.mapPane.propReach, d.mapPane.propReachAsked = propReach{}, propReachKey{}
	}
	if d.mapPane.propCentre.err != nil {
		d.mapPane.propCentre, d.mapPane.propCentreAsked = propCentre{}, propCentreKey{}
	}
	return d.askPropagation()
}

// propLegendRow keys the layer drawn by its colours, in MHz.
func (d Dashboard) propLegendRow(width int) string {
	if d.mapPane.propLayer == propFoF2 {
		return d.presetRow("fof2", "foF2, MHz │ ", "LOWER ", " HIGHER", width)
	}
	return d.presetRow("muf", "MUF(3000), MHz │ ", "LOWER ", " HIGHER", width)
}

// hfBand is an amateur HF band by its lower edge: the MUF and foF2
// presets' class edges (D-144, D-145), 160 metres to 10.
type hfBand struct {
	name string
	low  float64 // MHz
	mhz  string  // as the legend writes it
}

// hfBands are the bands, low to high; foF2's classes are the first six.
var hfBands = []hfBand{{"160m", 1.8, "1.8"}, {"80m", 3.5, "3.5"}, {"60m", 5.3, "5.3"}, {"40m", 7, "7"}, {"30m", 10.1, "10.1"},
	{"20m", 14, "14"}, {"17m", 18.068, "18.07"}, {"15m", 21, "21"}, {"12m", 24.89, "24.89"}, {"10m", 28, "28"}}

// fof2Bands is how many of the bands foF2's classes key (D-145).
const fof2Bands = 6

// bandsUnder are the bands whose lower edge is under a frequency.
func bandsUnder(mhz float64) []hfBand {
	var out []hfBand
	for _, b := range hfBands {
		if b.low < mhz {
			out = append(out, b)
		}
	}
	return out
}

// propLegendRows are D-160's two legend lines: each class by its band on
// its colour, and under each its lower edge in MHz; one line, cut as
// presetRow cuts it, where the library's classes are not the bands (with at
// most a leading class below them, which keys none).
func (d Dashboard) propLegendRows(width int) [2]string {
	preset, head, bands := "muf", "MUF(3000) │ ", hfBands
	if d.mapPane.propLayer == propFoF2 {
		preset, head, bands = "fof2", "foF2 │ ", hfBands[:fof2Bands]
	}
	classes := d.legendClasses(preset)
	each := 0
	if len(bands) > 0 {
		each = (width - render.Width(head)) / len(bands)
	}
	aligned, ok := bandClasses(classes, len(bands))
	if !ok || each < 7 {
		return [2]string{d.propLegendRow(width), ""}
	}
	var names, edges strings.Builder
	for i, b := range bands {
		c := aligned[i]
		names.WriteString(render.SwatchText(render.PadTo(b.name, each-1), c.Colour.R, c.Colour.G, c.Colour.B) + " ")
		edges.WriteString(render.PadTo(b.mhz, each))
	}
	mhzHead := strings.Repeat(" ", max(render.Width(head)-render.Width("MHz │ "), 0)) + "MHz │ "
	return [2]string{head + strings.TrimRight(names.String(), " "), mhzHead + strings.TrimRight(edges.String(), " ")}
}

// bandClasses are a legend's classes one a band, n of them: the classes as
// they are, or without a leading class below the first band - foF2's under
// 1.8 MHz, drawn as nothing (D-149); false for any other count.
func bandClasses(classes []tuimaps.Class, n int) ([]tuimaps.Class, bool) {
	switch len(classes) {
	case n:
		return classes, true
	case n + 1:
		return classes[1:], true
	}
	return nil, false
}

// cellValue is a field's value at a place, MHz, from the cell it falls in
// (a place off the field, from the nearest edge cell); false for a field
// with no cells or where the cell has no data.
func cellValue(f ionomaps.Field, lat, lon float64) (float64, bool) {
	if f.Cols <= 0 || f.Rows <= 0 || len(f.Values) != f.Cols*f.Rows {
		return 0, false
	}
	col := int((lon - f.West) / (f.East - f.West) * float64(f.Cols))
	row := int((f.North - lat) / (f.North - f.South) * float64(f.Rows))
	col, row = min(max(col, 0), f.Cols-1), min(max(row, 0), f.Rows-1)
	i := row*f.Cols + col
	if f.NoData.Has(i) {
		return 0, false
	}
	return float64(f.Values[i]), true
}

// propBlock is D-157's block under the map, styled for the picture: the
// place's name bold (D-164), the fields as chips (D-165), the age on the
// freshness ramp (D-166).
func (d Dashboard) propBlock(width int) []string {
	return d.propBlockAs(width, max(width, d.mapTextW()))
}

// propBlockAs is the block: the header with the sources' badges; the mode
// and when it was computed; a blank row; the selected place now, and the
// bands under its foF2 and MUF(3000); the continents. The rows beside the
// Controls box are cut at width, the rows below it, with nothing beside
// them, at below.
func (d Dashboard) propBlockAs(width, below int) []string {
	padRows := func(rows []string, width int) []string { return padRowsBeside(rows, width, below) }
	field := chipFace // D-165: a chip of its own
	tint := func(text string, tok render.Token) string { return render.Tint(text, render.Tok(tok)) }
	out := []string{"RADIO FREQUENCY PROPAGATION    " + strings.Join(d.propBadges(true), "  ")}
	snap := d.mapPane.propSnap
	origin := d.mapPane.propOrigin // D-193: the centre on Observer, the station on the Broadcaster
	if snap == nil || !propHasField(*snap) {
		out = append(out, "MODE: "+field(propLayerName(d.mapPane.propLayer))+" · "+strings.TrimPrefix(d.propStatusWords(), "Propagation · "))
		return padRows(out, width)
	}
	now := d.now()
	at := snap.Computed.In(now.Location())
	words, age, stale := d.propAge(*snap) // FR-5.1: the oldest input's age, never the time computed
	tone := render.FreshnessTone(age)
	if stale {
		tone = render.AlertLabel // FR-5.2
	}
	mode := "MODE: " + field(propLayerName(d.mapPane.propLayer)) + " · COMPUTED " + d.clockFmt.Time(at) + " " + at.Format("MST") +
		" " + tint("("+words+")", tone) + " · " + strings.ToLower(propNoiseLabel(snap.Noise)) + " noise" // D-166; D-187: the circuit's noise, the chart saying the rest
	if origin.ok && d.mapPane.propAhead > 0 { // an hour ahead: typical, said where the time was (FR-1.13, D-109)
		mode = "MODE: " + field(propLayerName(d.mapPane.propLayer)) + " · " + tint(d.propAheadLabel(origin.at.Lat, origin.at.Lon), render.AlertLabel)
	}
	held := d.propFreqLine()
	if n := propOffsetNote(snap.Offset); n != "" { // FR-5.4: the offset's state, in the row the mode line leaves, never cut off its end
		held = strings.TrimPrefix(held+" · ", " · ") + tint(n, render.AlertLabel)
	}
	out = append(out, mode, held) // D-163: a blank row before the place, the frequency held there (D-179)
	if !origin.ok {
		return padRows(out, width)
	}
	name := strings.ToUpper(origin.name)
	place := tint(name, render.MapPropPlace) + "   " // D-164; D-163: the place in a column of its own, its answer under NOW
	under := strings.Repeat(" ", render.Width(name)+3)
	if render.Width(name) > propPlaceColMax { // a long name (a view's, D-193): its answers under it at a short indent, the block's rows unchanged
		under = strings.Repeat(" ", propLongIndent)
	}
	shown := d.propOnScreen()
	when := "NOW"
	if k := d.mapPane.propAhead; k > 0 {
		when = "+" + strconv.Itoa(k) + " h"
	}
	fo, okF := cellValue(shown.Now.FoF2, origin.at.Lat, origin.at.Lon)
	muf, okM := cellValue(shown.Now.MUF3000, origin.at.Lat, origin.at.Lon)
	if !okF || !okM {
		return padRows(append(out, place+when+" · no data at the place"), width)
	}
	out = append(out, fmt.Sprintf("%s%s · %s %.1f MHz · %s %.1f MHz", place, when, field("foF2"), fo, field("MUF(3000)"), muf))
	out = append(out, under+"Local, to ~"+strconv.Itoa(d.propRadiusKm())+" km ("+d.propLocalName()+"): "+bandNames(bandsUnder(fo))) // D-100: the Setting's radius; the service radius on the Broadcaster (FR-2.3)
	if best := d.propBestLine(); best != "" {
		out = append(out, under+best) // D-176, D-177: the continents, in the 3,000 km hops' place, and the key to the chart
	}
	return padRows(out, width)
}

// The place's column in the block: under a name wider than this, the
// answers are indented by the second, not under NOW (A-51).
const (
	propPlaceColMax = 24
	propLongIndent  = 2
)

// propAgeShown reports whether the block says a snapshot's age, which moves
// with the clock (D-166).
func (d Dashboard) propAgeShown() bool {
	return d.modal == modalMap && d.mapMode() == modePropagation && d.mapPane.propSnap != nil && propHasField(*d.mapPane.propSnap)
}

// propBadges are the sources' badges (D-168): lit for a source that fed the
// snapshot shown, dimmed for one that did not; as plain words, only the ones
// that fed it.
func (d Dashboard) propBadges(styled bool) []string {
	snap := d.mapPane.propSnap
	shown := snap != nil && propHasField(*snap)
	var out []string
	for _, s := range []struct {
		name string
		fed  bool
	}{
		{"SWPC", shown && snap.Background.FoF2 == ionomaps.GloTEC}, // the solar index alone, under the climatology, counts as unused
		{"PYIRI", shown}, // M(3000)F2 is the climatology's at every cell
		{"IGRF-14", shown},
		{"GIRO", shown && snap.Inputs.Stations > 0}, // only when stations were assimilated into the field
	} {
		switch {
		case styled && s.fed:
			out = append(out, chipFace(s.name))
		case styled:
			out = append(out, chipDimmed(s.name))
		case s.fed:
			out = append(out, s.name)
		}
	}
	return out
}

// bandNames are bands by name, or "none of the bands".
func bandNames(bands []hfBand) string {
	if len(bands) == 0 {
		return "none of the bands"
	}
	names := make([]string, len(bands))
	for i, b := range bands {
		names[i] = b.name
	}
	return strings.Join(names, " ")
}

// padRowsBeside are the block's rows, propBlockRows of them, each cut to
// the width beside the Controls box, or below it to the width below.
func padRowsBeside(rows []string, width, below int) []string {
	out := make([]string, propBlockRows)
	for i := range out {
		w := width
		if i >= scrubBoxRows {
			w = below
		}
		if i < len(rows) {
			out[i] = render.TruncateCells(rows[i], w)
		}
	}
	return out
}

// propBlockRows are the block's rows (D-163, D-177): the header, the mode, a
// blank row, the place, its near-vertical answer and the continents' line -
// one more than the weather modes' loop row, timeline and estimate, which
// mapChromeRows counts.
const propBlockRows = scrubBoxRows + 1
