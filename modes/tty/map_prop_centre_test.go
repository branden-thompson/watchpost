package tty

import (
	"strings"
	"testing"
)

// workCentre runs the centre's command, when one is due, and applies it.
func workCentre(t *testing.T, d Dashboard) Dashboard {
	t.Helper()
	d.mapPane.propCentreAsked = propCentreKey{}
	d, cmd := d.armPropCentre()
	if cmd == nil {
		return d
	}
	m, _ := d.Update(cmd())
	return m.(Dashboard)
}

// TestTheCentreIsAnsweredAsTheMapPans is FR-1.11 (D-37, D-180): the map's
// centre is answered in a box over the map's upper left - its place, its
// MUF(3000) and foF2, and from the selected place the distance, the path's
// MUF and the highest band open - and a pan moves the answer with it.
func TestTheCentreIsAnsweredAsTheMapPans(t *testing.T) {
	d := workCentre(t, chartDash(t))
	box := stripANSITest(strings.Join(d.propCentreBox(), "\n"))
	for _, want := range []string{"Centre", "MUF(3000) 18.0", "foF2 6.0", "OCEANSIDE, CA", "up to"} {
		if !strings.Contains(box, want) {
			t.Errorf("the centre box does not say %q:\n%s", want, box)
		}
	}
	window := stripANSITest(strings.Join(d.mapBodyLines(), "\n"))
	if !strings.Contains(window, "┌─ Centre") {
		t.Errorf("the centre box is not over the map:\n%s", window)
	}
	before := d.mapPane.propCentre.key.centre
	for range 6 {
		d = pressMap(t, d, "right")
	}
	d = workCentre(t, d)
	if after := d.mapPane.propCentre.key.centre; after == before {
		t.Errorf("six pans east left the centre answered at %v", after)
	}
	if box := stripANSITest(strings.Join(d.propCentreBox(), "\n")); !strings.Contains(box, "From OCEANSIDE, CA: ") || !strings.Contains(box, "MUF ") {
		t.Errorf("a centre far from the place is not given its distance and the path's MUF:\n%s", box)
	}
}

// TestTheCentreAnswerIsSaid is FR-1.11 and FR-3.1: without the picture the
// centre's answer is said, last.
func TestTheCentreAnswerIsSaid(t *testing.T) {
	d := workCentre(t, chartDash(t))
	words := d.describeLinesAll()
	last := words[len(words)-1]
	if !strings.Contains(last, "The map's centre") || !strings.Contains(last, "MUF(3000) 18.0") {
		t.Errorf("the words do not end with the centre's answer: %q", last)
	}
}
