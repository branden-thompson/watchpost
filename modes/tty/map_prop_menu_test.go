package tty

import (
	"errors"
	"slices"
	"testing"
)

// menuOn opens the Overlays menu with its cursor on a row, by key.
func menuOn(t *testing.T, d Dashboard, key string) Dashboard {
	t.Helper()
	if !d.mapPane.menuOn {
		d = pressMap(t, d, "O")
	}
	i := slices.IndexFunc(d.overlayRows(), func(r overlayRow) bool { return r.key == key })
	if i < 0 {
		t.Fatalf("the menu has no %q row: %+v", key, d.overlayRows())
	}
	d.mapPane.menuAt = i
	return d
}

// TestTheOverlaysMenuOffersThePropagationMode is D-158: the tints carry a
// Radio Propagation row; space on it is P, and space on it chosen leaves
// for the weather mode it came from, as P does.
func TestTheOverlaysMenuOffersThePropagationMode(t *testing.T) {
	d := propDash(t, &seam{answer: PropagationResult{Err: errors.New("offline")}}, true)
	from := d.mapMode()
	d = pressMap(t, menuOn(t, d, propagationRowKey), "space")
	if d.mapMode() != modePropagation || !d.tintChosen(propagationRowKey) {
		t.Fatalf("space on Radio Propagation gave mode %d, chosen %v", d.mapMode(), d.tintChosen(propagationRowKey))
	}
	if r := d.overlayRows()[d.mapPane.menuAt]; r.key != propagationRowKey {
		t.Errorf("the cursor moved to %q", r.key)
	}
	if d = pressMap(t, menuOn(t, d, propagationRowKey), "space"); d.mapMode() != from {
		t.Errorf("space on the chosen row gave mode %d; want %d again", d.mapMode(), from)
	}
}

// TestAWeatherTintLeavesThePropagationMode is D-158: a weather tint chosen
// in the Propagation mode returns to the weather mode it came from with that
// tint drawn - chosen, never cleared, even when it was on before.
func TestAWeatherTintLeavesThePropagationMode(t *testing.T) {
	var asks []MapAsk
	d := openTempMap(t, false, &asks) // Forecast mode, temperature on
	d.cfg.PropagationUpdate = (&seam{answer: PropagationResult{Err: errors.New("offline")}}).update
	d.cfg.PropagationAck = propAckVersion
	d = pressMap(t, d, "P")
	for _, c := range []struct{ key string }{{TemperatureLayer}, {UVLayer}} {
		e := pressMap(t, menuOn(t, d, c.key), "space")
		if e.mapMode() != modeForecast || !e.tintChosen(c.key) {
			t.Errorf("%s from the Propagation mode gave mode %d, chosen %v; want Forecast with it drawn", c.key, e.mapMode(), e.tintChosen(c.key))
		}
	}
}

// TestTheMenuInThePropagationModeShowsTintsAndDetail is D-159: in the
// Propagation mode the menu is the tints and MAP DETAILS; the weather groups
// return in the weather modes, their ticks kept.
func TestTheMenuInThePropagationModeShowsTintsAndDetail(t *testing.T) {
	d := screenDash(t, "with", false)
	d.cfg.PropagationAck = propAckVersion
	weather, choice := len(d.overlayRows()), d.mapLayerChoice
	d = pressMap(t, d, "P")
	for _, r := range d.overlayRows() {
		if r.kind != menuRadio && r.kind != menuPreset && r.kind != menuDetail {
			t.Errorf("the Propagation mode's menu has a %d row, %q", r.kind, r.key)
		}
	}
	if d = pressMap(t, d, "P"); len(d.overlayRows()) != weather || d.mapLayerChoice != choice {
		t.Errorf("back in the weather mode the menu has %d rows of %d, choices %q of %q", len(d.overlayRows()), weather, d.mapLayerChoice, choice)
	}
}
