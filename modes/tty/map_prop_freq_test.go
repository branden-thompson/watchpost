package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/geo"
)

// enterFreq presses F, types text and presses Enter, then works out the
// reach the Update armed, as the app's command would.
func enterFreq(t *testing.T, d Dashboard, text string) Dashboard {
	t.Helper()
	d = pressMap(t, d, "F")
	if d.modal != modalPropFreq {
		t.Fatalf("F opened window %v; want the frequency entry (D-178)", d.modal)
	}
	for _, r := range text {
		m, _ := d.Update(mapKeyMsg(t, string(r)))
		d = m.(Dashboard)
	}
	m, _ := d.Update(mapKeyMsg(t, "enter"))
	d = m.(Dashboard)
	d.mapPane.propReachAsked = propReachKey{} // the harness dropped the command the Update armed: ask again
	d, cmd := d.armPropReach()
	if cmd != nil {
		m, _ = d.Update(cmd())
		d = m.(Dashboard)
	}
	return d
}

// TestYourFrequencyDrawsItsReach is FR-1.16 (D-76, D-178): a frequency typed
// is drawn as its reach from the selected place, a field of reached or not
// over the layer drawn: with foF2 6 and MUF(3000) 18 MHz everywhere, 14.2
// MHz skips the place's own area and reaches 3,000 km away.
func TestYourFrequencyDrawsItsReach(t *testing.T) {
	d := enterFreq(t, chartDash(t), "14.2")
	if d.modal != modalMap || d.mapPane.propFreqMHz != 14.2 {
		t.Fatalf("after Enter: window %v, %v MHz held; want the map, 14.2", d.modal, d.mapPane.propFreqMHz)
	}
	o, ok := d.mapPane.propGiven["propagation/reach"]
	if !ok || o.Grid == nil || len(o.Grid.Type.Breaks) != 1 || o.Grid.Type.Preset != "" {
		t.Fatalf("the reach handed in is %+v (%v); want a two-class host field", o.Grid, ok)
	}
	loc := d.selectedLocation()
	near, far := -1.0, -1.0
	g := o.Grid
	for i, v := range g.Values {
		lat := g.North - (float64(i/g.Cols)+0.5)*(g.North-g.South)/float64(g.Rows)
		lon := g.West + (float64(i%g.Cols)+0.5)*(g.East-g.West)/float64(g.Cols)
		switch km := geo.HaversineKM(loc.Lat, loc.Lon, lat, lon); {
		case km < 150:
			near = v
		case km > 2900 && km < 3100:
			far = v
		}
	}
	if near != 0 || far != 1 {
		t.Errorf("the reach is %v at the place and %v about 3,000 km out; 14.2 MHz is over foF2 6 and under MUF(3000) 18", near, far)
	}
}

// TestYourFrequencyIsSaid is FR-1.16 and D-179: the block's row between the
// mode and the place, the chart and the words say the frequency held and
// how far its skip zone runs; a band is taken by name; an empty Enter
// clears it, the reach taken off and the row blank again.
func TestYourFrequencyIsSaid(t *testing.T) {
	d := enterFreq(t, chartDash(t), "14.2")
	row := stripANSITest(d.propBlock(d.scrubW())[2])
	for _, want := range []string{"YOUR FREQUENCY 14.200 MHz", "skip zone to ~", "F change"} {
		if !strings.Contains(row, want) {
			t.Errorf("the block's frequency row %q does not say %q", row, want)
		}
	}
	if words := strings.Join(d.describeLinesAll(), " "); !strings.Contains(words, "YOUR FREQUENCY 14.200 MHz") {
		t.Errorf("without the picture the frequency is not said: %q", words)
	}
	if chart := stripANSITest(strings.Join(d.propChartLines(d.opts()), "\n")); !strings.Contains(chart, "YOUR FREQUENCY 14.200 MHz") {
		t.Errorf("the chart does not say the frequency held:\n%s", chart)
	}
	d = enterFreq(t, d, "40 m")
	if d.mapPane.propFreqLabel != "40m" || d.mapPane.propFreqMHz != 7 {
		t.Errorf("\"40 m\" is held as %q at %v MHz; want the band 40m at its lower edge, 7", d.mapPane.propFreqLabel, d.mapPane.propFreqMHz)
	}
	d = enterFreq(t, d, "")
	if _, drawn := d.mapPane.propGiven["propagation/reach"]; drawn || d.mapPane.propFreqMHz != 0 {
		t.Error("an empty Enter left the frequency held or its reach drawn")
	}
	if row := stripANSITest(d.propBlock(d.scrubW())[2]); strings.TrimSpace(row) != "" {
		t.Errorf("with no frequency held the row is %q; D-163's blank row", row)
	}
}

// TestABadFrequencyIsSaidNotGuessed is FR-1.16 and IS-4: what is not a
// frequency from 1.8 to 30 MHz or a band from 160m to 10m is refused with
// what is accepted, never the text typed, and nothing held changes; Esc
// leaves the frequency held as it was.
func TestABadFrequencyIsSaidNotGuessed(t *testing.T) {
	held := enterFreq(t, chartDash(t), "7.188")
	for _, bad := range []string{"31", "1.7", "45m", "abc", "7.1.8"} {
		d := pressMap(t, held, "F")
		for _, r := range bad {
			m, _ := d.Update(mapKeyMsg(t, string(r)))
			d = m.(Dashboard)
		}
		m, _ := d.Update(mapKeyMsg(t, "enter"))
		d = m.(Dashboard)
		text := stripANSITest(strings.Join(d.propFreqLines(d.opts()), "\n"))
		if d.modal != modalPropFreq || !strings.Contains(text, errFreqWords) {
			t.Errorf("%q: window %v, the entry says:\n%s", bad, d.modal, text)
		}
		if strings.Contains(strings.ReplaceAll(text, "MHz or band: "+bad, ""), bad) {
			t.Errorf("%q: the refusal quotes what was typed:\n%s", bad, text)
		}
		if d.mapPane.propFreqMHz != 7.188 {
			t.Errorf("%q: the frequency held moved to %v", bad, d.mapPane.propFreqMHz)
		}
		m, _ = d.Update(mapKeyMsg(t, "esc"))
		if back := m.(Dashboard); back.modal != modalMap || back.mapPane.propFreqMHz != 7.188 {
			t.Errorf("%q: Esc left window %v holding %v MHz", bad, back.modal, back.mapPane.propFreqMHz)
		}
	}
}
