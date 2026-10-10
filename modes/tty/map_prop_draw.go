package tty

import (
	"math"

	tea "charm.land/bubbletea/v2"
	ionomaps "github.com/branden-thompson/go-ionomaps"
	tuimaps "github.com/branden-thompson/go-tuimaps"

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
