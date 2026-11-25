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
func (mm mapModel) propOverlays(e mapEnv) []tuimaps.Overlay {
	shown := mm.propOnScreen()
	if !mm.inPropagation(e) || !propHasField(shown) {
		return nil
	}
	h := shown.Now
	field := tuimaps.MUFGrid("propagation/muf", fieldGrid(h.MUF3000), h.At)
	if mm.mapPane.propLayer == propFoF2 {
		field = tuimaps.FoF2Grid("propagation/fof2", fieldGrid(h.FoF2), h.At)
	}
	out := []tuimaps.Overlay{field}
	if reach, ok := mm.propReachShown(e); ok {
		out = append(out, propReachOverlay(reach, h.At)) // after the field: its edge alone over it (L-2.6)
	}
	return out // the night is the map's, drawn in every mode that wants it (D-199)
}

// propNightMoment is the moment the night is drawn for: the field's own
// time, to the step - the sun as it stood when the field was made, however
// old the field (W3.1) - an hour ahead's being its start, the moment its
// field and its words are for; the clock before any field.
func (mm mapModel) propNightMoment(e mapEnv) time.Time {
	shown, ok := mm.propShown()
	if !ok {
		return e.Now().UTC().Truncate(nightStep)
	}
	return shown.Now.At.UTC().Truncate(nightStep)
}

// setProp hands the library the Propagation mode's overlays as they stand,
// and takes off the ones no longer wanted.
func (mm mapModel) setProp(e mapEnv) mapModel {
	if mm.mapPane.m == nil {
		return mm
	}
	given, _ := mm.reconcile(mm.mapPane.propGiven, mm.propOverlays(e), func(o tuimaps.Overlay, err error) {
		mm.problem("Propagation: "+o.ID+" not drawn - "+err.Error(), e) // ours to fix, never the listener's (D-124)
	})
	mm.mapPane.propGiven = given
	mm.mapPane.gen++
	return mm.setNight(e) // the night with the mode: always in it, by the detail switch out of it (D-199)
}

// flipPropLayer is L: MUF(3000) or foF2, in the Propagation mode alone.
func (mm mapModel) flipPropLayer(e mapEnv) mapModel {
	if mm.mapMode(e) == modePropagation {
		mm.mapPane.propLayer = 1 - mm.mapPane.propLayer
		return mm.setProp(e).renderMap(e)
	}
	return mm
}

// refreshProp is U: an update now, in the Propagation mode alone (as every
// ask is), under the library's throttle (D-120): an early ask is answered
// from the last field with its reason.
func (mm mapModel) refreshProp(e mapEnv) (mapModel, tea.Cmd) {
	if mm.mapPane.propChart.err != nil { // an answer that failed is asked again with the update (AX-9)
		mm.mapPane.propChart, mm.mapPane.propChartAsked = propChart{}, propChartKey{}
	}
	if mm.mapPane.propReach.err != nil {
		mm.mapPane.propReach, mm.mapPane.propReachAsked = propReach{}, propReachKey{}
	}
	if mm.mapPane.propCentre.err != nil {
		mm.mapPane.propCentre, mm.mapPane.propCentreAsked = propCentre{}, propCentreKey{}
	}
	return mm.askPropagation(e)
}

// propLegendRow keys the layer drawn by its colours, in MHz.
func (mm mapModel) propLegendRow(width int) string {
	if mm.mapPane.propLayer == propFoF2 {
		return mm.presetRow("fof2", "foF2, MHz │ ", "LOWER ", " HIGHER", width)
	}
	return mm.presetRow("muf", "MUF(3000), MHz │ ", "LOWER ", " HIGHER", width)
}

// hfBand is an amateur HF band by its lower edge: the MUF and foF2
// presets' class edges (D-144, D-145), 160 metres to 10.
type hfBand struct {
	name string
	low  float64 // MHz
	mhz  string  // as the legend writes it
}

// hfBands are the bands go-ionomaps answers, low to high, by its own names
// and lower edges, the legend's to two places; foF2's classes are the first
// six.
var hfBands = func() []hfBand {
	var out []hfBand
	for _, b := range ionomaps.AllBands() {
		out = append(out, hfBand{name: b.String(), low: b.LowMHz(), mhz: strconv.FormatFloat(math.Round(b.LowMHz()*100)/100, 'f', -1, 64)})
	}
	return out
}()

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
func (mm mapModel) propLegendRows(width int) [2]string {
	preset, head, bands := "muf", "MUF(3000) │ ", hfBands
	if mm.mapPane.propLayer == propFoF2 {
		preset, head, bands = "fof2", "foF2 │ ", hfBands[:fof2Bands]
	}
	classes := mm.legendClasses(preset)
	each := 0
	if len(bands) > 0 {
		each = (width - render.Width(head)) / len(bands)
	}
	aligned, ok := bandClasses(classes, len(bands))
	if !ok || each < 7 {
		return [2]string{mm.propLegendRow(width), ""}
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
func (mm mapModel) propBlock(width int, e mapEnv) []string {
	return mm.propBlockAs(width, max(width, mm.mapTextW(e)), e)
}

// propBlockAs is the block: the header with the sources' badges; the mode
// and when it was computed; a blank row; the selected place now, and the
// bands under its foF2 and MUF(3000); the continents. The rows beside the
// Controls box are cut at width, the rows below it, with nothing beside
// them, at below.
func (mm mapModel) propBlockAs(width, below int, e mapEnv) []string {
	padRows := func(rows []string, width int) []string { return padRowsBeside(rows, width, below) }
	field := chipFace // D-165: a chip of its own
	tint := func(text string, tok render.Token) string { return render.Tint(text, render.Tok(tok)) }
	out := []string{"RADIO FREQUENCY PROPAGATION    " + strings.Join(mm.propBadges(true), "  ")}
	snap := mm.mapPane.propSnap
	origin := mm.mapPane.propOrigin // D-193: the centre on Observer, the station on the Broadcaster
	if snap == nil || !propHasField(*snap) {
		out = append(out, "MODE: "+field(propLayerName(mm.mapPane.propLayer))+" · "+strings.TrimPrefix(mm.propStatusWords(e), "Propagation · "))
		return padRows(out, width)
	}
	now := e.Now()
	at := snap.Computed.In(now.Location())
	words, age, stale := mm.propAge(*snap, e) // FR-5.1: the oldest input's age, never the time computed
	tone := render.FreshnessTone(age)
	if stale {
		tone = render.AlertLabel // FR-5.2
	}
	mode := "MODE: " + field(propLayerName(mm.mapPane.propLayer)) + " · COMPUTED " + e.Opts.Clock.Time(at) + " " + at.Format("MST") +
		" " + tint("("+words+")", tone) + " · " + strings.ToLower(propNoiseLabel(snap.Noise)) + " noise" // D-166; D-187: the circuit's noise, the chart saying the rest
	if origin.ok && mm.mapPane.propAhead > 0 { // an hour ahead: typical, said where the time was (FR-1.13, D-109)
		mode = "MODE: " + field(propLayerName(mm.mapPane.propLayer)) + " · " + tint(mm.propAheadLabel(origin.at.Lat, origin.at.Lon), render.AlertLabel)
	}
	held := mm.propFreqLine(e)
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
	shown := mm.propOnScreen()
	when := "NOW"
	if k := mm.mapPane.propAhead; k > 0 {
		when = "+" + strconv.Itoa(k) + " h"
	}
	fo, okF := cellValue(shown.Now.FoF2, origin.at.Lat, origin.at.Lon)
	muf, okM := cellValue(shown.Now.MUF3000, origin.at.Lat, origin.at.Lon)
	if !okF || !okM {
		return padRows(append(out, place+when+" · no data at the place"), width)
	}
	out = append(out, fmt.Sprintf("%s%s · %s %.1f MHz · %s %.1f MHz", place, when, field("foF2"), fo, field("MUF(3000)"), muf))
	out = append(out, under+"Local, to ~"+strconv.Itoa(mm.propRadiusKm(e))+" km ("+mm.propLocalName(e)+"): "+bandNames(bandsUnder(fo))) // D-100: the Setting's radius; the service radius on the Broadcaster (FR-2.3)
	if best := mm.propBestLine(e); best != "" {
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
func (mm mapModel) propAgeShown(e mapEnv) bool {
	return e.Window == mapModalMap && mm.mapMode(e) == modePropagation && mm.mapPane.propSnap != nil && propHasField(*mm.mapPane.propSnap)
}

// propBadges are the sources' badges (D-168): lit for a source that fed the
// snapshot shown, dimmed for one that did not; as plain words, only the ones
// that fed it.
func (mm mapModel) propBadges(styled bool) []string {
	snap := mm.mapPane.propSnap
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
