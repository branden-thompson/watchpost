package tty

import (
	"fmt"
	"math"
	"strconv"
	"strings"

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
// layer alone (D-112); none outside the mode or before a snapshot.
func (d Dashboard) propOverlays() []tuimaps.Overlay {
	snap := d.mapPane.propSnap
	if d.mapMode() == modeRadar || d.mapMode() == modeForecast || snap == nil || len(snap.Hours) == 0 {
		return nil
	}
	h := snap.Hours[0]
	if d.mapPane.propLayer == propFoF2 {
		return []tuimaps.Overlay{tuimaps.FoF2Grid("propagation/fof2", fieldGrid(h.FoF2), h.At)}
	}
	return []tuimaps.Overlay{tuimaps.MUFGrid("propagation/muf", fieldGrid(h.MUF3000), h.At)}
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
	return d
}

// flipPropLayer is L: MUF(3000) or foF2, in the Propagation mode alone.
func (d Dashboard) flipPropLayer() Dashboard {
	if d.mapMode() == modeRadar || d.mapMode() == modeForecast {
		return d
	}
	d.mapPane.propLayer = 1 - d.mapPane.propLayer
	return d.setProp().renderMap()
}

// refreshProp is U: an update now, in the Propagation mode alone, under the
// library's throttle (D-120): an early ask is answered from the last field
// with its reason.
func (d Dashboard) refreshProp() (Dashboard, tea.Cmd) {
	if d.mapMode() == modeRadar || d.mapMode() == modeForecast {
		return d, nil
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
	var classes []tuimaps.Class
	for _, e := range d.mapPane.legend {
		if e.Preset == preset {
			classes = e.Classes
		}
	}
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

// cellValue is a field's value at a place, MHz; false off the field or
// where it has no data.
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

// propBlock is D-157's block under the map, in the timeline's five rows:
// the header with the sources' badges; the mode and when it was computed;
// the selected place now, and the bands under its foF2 and MUF(3000).
func (d Dashboard) propBlock(width int) []string {
	badges := []string{chipFace("SWPC"), chipFace("PYIRI"), chipFace("IGRF-14")}
	out := []string{"RADIO FREQUENCY PROPAGATION    " + strings.Join(badges, "  ")}
	snap, loc := d.mapPane.propSnap, d.selectedLocation()
	if snap == nil || len(snap.Hours) == 0 {
		out = append(out, "MODE: "+propLayerName(d.mapPane.propLayer)+" · "+strings.TrimPrefix(d.propStatusWords(), "Propagation · "))
		return padRows(out, width)
	}
	at := snap.Computed.In(d.now().Location())
	ago := int(d.now().Sub(snap.Computed).Minutes())
	out = append(out, "MODE: "+propLayerName(d.mapPane.propLayer)+" · COMPUTED "+d.clockFmt.Time(at)+" "+at.Format("MST")+
		" ("+strconv.Itoa(max(ago, 0))+" min ago) · upper limits only", "") // D-163: a blank row before the place
	if loc == nil {
		return padRows(out, width)
	}
	place := strings.ToUpper(loc.Label) + "   " // D-163: the place in a column of its own, its answer under NOW
	under := strings.Repeat(" ", render.Width(place))
	fo, okF := cellValue(snap.Hours[0].FoF2, loc.Lat, loc.Lon)
	muf, okM := cellValue(snap.Hours[0].MUF3000, loc.Lat, loc.Lon)
	if !okF || !okM {
		return padRows(append(out, place+"NOW · no data at the place"), width)
	}
	out = append(out, fmt.Sprintf("%sNOW · foF2 %.1f MHz · MUF(3000) %.1f MHz", place, fo, muf))
	out = append(out, under+"Local, to ~400 km (NVIS): "+bandNames(bandsUnder(fo)))
	hops := bandsUnder(muf)
	if len(hops) == 0 {
		return padRows(append(out, under+"~3,000 km hops through here: none of the bands"), width)
	}
	return padRows(append(out, fmt.Sprintf("%s~3,000 km hops through here: up to %s (%.1f MHz)", under, hops[len(hops)-1].name, muf)), width)
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

// padRows are the block's rows, propBlockRows of them, each cut to the width.
func padRows(rows []string, width int) []string {
	out := make([]string, propBlockRows)
	for i := range out {
		if i < len(rows) {
			out[i] = render.TruncateCells(rows[i], width)
		}
	}
	return out
}

// propBlockRows are the block's rows (D-163): the header, the mode, a blank
// row, the place and its two answers - one more than the weather modes' loop
// row, timeline and estimate, which mapChromeRows counts.
const propBlockRows = radarRows + 3

// propWordsWidth is the width the block is laid out at for the words: wide
// enough that no line of it is cut.
const propWordsWidth = 200
