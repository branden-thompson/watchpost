package tty

import (
	"regexp"
	"strings"
	"testing"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// workCentre runs the centre's command, when one is due, and applies it.
func workCentre(t *testing.T, d Dashboard) Dashboard {
	t.Helper()
	d = settled(t, d)
	d.mapPane.propCentreAsked = propCentreKey{}
	d, cmd := d.armPropCentre()
	if cmd == nil {
		return d
	}
	m, _ := d.Update(cmd())
	return m.(Dashboard)
}

// stationDash is chartDash on the Broadcaster (D-193): the station the
// origin - here none is set, so the selected place, said - and the map's
// centre the target.
func stationDash(t *testing.T) Dashboard {
	t.Helper()
	d := chartDash(t)
	d.surface = SurfaceBroadcaster
	return workChart(t, d.renderMap())
}

// TestTheCentreIsAnsweredAsTheMapPans is FR-1.11 (D-37, D-180, D-193): on the
// Broadcaster the map's centre is the target, answered in a box over the
// map's upper left - its place, its MUF(3000) and foF2, and from the station
// the distance, the bands usable and the limits - and a pan moves the answer
// with it.
func TestTheCentreIsAnsweredAsTheMapPans(t *testing.T) {
	d := workCentre(t, stationDash(t))
	box := stripANSITest(strings.Join(d.propCentreBox(), "\n"))
	for _, want := range []string{"Centre", "MUF(3000) 18.0", "foF2 6.0", "OCEANSIDE, CA (NO TRANSMITTER SET)", "usable ", "best ~"} { // the LUF only at hours the absorption closes a band
		if !strings.Contains(box, want) {
			t.Errorf("the centre box does not say %q:\n%s", want, box)
		}
	}
	if !regexp.MustCompile(`usable [^\n│]* · foF2 `).MatchString(box) {
		t.Errorf("a centre within the near-vertical radius is not answered from the area, its limit foF2:\n%s", box)
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
	pathLimit := regexp.MustCompile(`usable[^\n│]* · MUF \d`) // the usable line's limit: the best frequency's words say "MUF" too
	if box := stripANSITest(strings.Join(d.propCentreBox(), "\n")); !strings.Contains(box, "From OCEANSIDE, CA (NO TRANSMITTER SET): ") || !pathLimit.MatchString(box) {
		t.Errorf("a centre far from the place is not given its distance and the path's MUF:\n%s", box)
	}
}

// TestTheCentreAnswerIsSaid is FR-1.11 and FR-3.1: without the picture the
// centre's answer is said, the last of the mode's words - the place's
// weather and the night after them (D-210) - naming the station as it is
// written, not in the box's capitals (AX-8).
func TestTheCentreAnswerIsSaid(t *testing.T) {
	d := workCentre(t, stationDash(t))
	loc := d.mapPlace()
	mode := d.propWords(loc.Lat, loc.Lon)
	last := mode[len(mode)-1]
	if !strings.Contains(last, "The map's centre") || !strings.Contains(last, "MUF(3000) 18.0") {
		t.Errorf("the mode's words do not end with the centre's answer: %q", last)
	}
	if !strings.Contains(last, "near-vertical from Oceanside, CA") || strings.Contains(last, "OCEANSIDE") || strings.Contains(last, "; No station") {
		t.Errorf("the centre's words name the station in capitals, or begin a clause with one: %q", last)
	}
	words := d.describeLinesAll()
	if at := lineWith(words, "The map's centre"); at != len(mode)-1 || !strings.HasPrefix(words[at+1], "Oceanside, CA") {
		t.Errorf("the place's weather heading is not after the mode's words:\n%s", strings.Join(words, "\n"))
	}
}

// TestTheHeldFrequencyIsJudgedOnThePath is D-193's Broadcaster answer: from
// the station, the frequency held is judged against the path's limits and
// told which way to move - and with the LUF at or over the MUF, that none
// serves, never "come down" into the absorption.
func TestTheHeldFrequencyIsJudgedOnThePath(t *testing.T) {
	for _, c := range []struct {
		f    float64
		l    propLimits
		want string
	}{
		{20, propLimits{upper: 14}, "above the MUF - come down under 14.0"},
		{14, propLimits{upper: 14}, "above the MUF - come down under 14.0"},
		{20, propLimits{upper: 14, luf: 15}, "too weak at 100 W on this path"},
		{20, propLimits{upper: 14, luf: 14}, "too weak at 100 W on this path"},
		{3, propLimits{upper: 14, luf: 5, weak: true}, "too weak at 100 W - go up to 5.0+"},
		{3, propLimits{upper: 14, luf: 5}, "absorbed - go up to 5.0+"},
		{7, propLimits{upper: 14, luf: 5, drap: 9}, "disturbed (D-RAP to 9.0)"},
		{10, propLimits{upper: 14, luf: 5, drap: 9}, "open"},
		{10, propLimits{}, "open"},
	} {
		if got := propFreqVerdict(c.f, "7.2 MHz", c.l); got != "7.2 MHz: "+c.want {
			t.Errorf("%.1f MHz under %+v: %q; want %q", c.f, c.l, got, c.want)
		}
	}
}

// TestTheTargetSaysWhatIsUsable is D-193's first line: the usable bands from
// the station, one or a span, or nothing; then the path's limits; and the
// held frequency's verdict only when one is held.
func TestTheTargetSaysWhatIsUsable(t *testing.T) {
	band := func(s ionomaps.Status) ionomaps.BandStatus { return ionomaps.BandStatus{Status: s} }
	bands := make([]ionomaps.BandStatus, len(hfBands))
	for i := range bands {
		bands[i] = band(ionomaps.AboveUpper)
	}
	bands[0] = ionomaps.BandStatus{Status: ionomaps.Absorbed, Limit: ionomaps.LimitAbsorption, LimitMHz: 4}
	bands[9] = ionomaps.BandStatus{Status: ionomaps.AboveUpper, Limit: ionomaps.LimitMUF, LimitMHz: 16}
	var d Dashboard
	if got := d.propTarget(bands); len(got) != 2 || got[0] != "nothing usable at 100 W · MUF 16.0 MHz · LUF 4.0 MHz" || got[1] != "best ~13.6 MHz (30m), 85% of the MUF" {
		t.Errorf("no band open: %q", got)
	}
	bands[3] = band(ionomaps.Open)
	if got := d.propTarget(bands); got[0] != "usable 40m · MUF 16.0 MHz · LUF 4.0 MHz" {
		t.Errorf("one band open: %q", got[0])
	}
	bands[5] = band(ionomaps.Disturbed)
	d.mapPane.propFreqMHz, d.mapPane.propFreqLabel = 7.2, "7.2 MHz"
	got := d.propTarget(bands)
	if got[0] != "usable 40m-20m · MUF 16.0 MHz · LUF 4.0 MHz" || len(got) != 3 || got[2] != "7.2 MHz: open" {
		t.Errorf("40m open and 20m disturbed, 7.2 MHz held: %q", got)
	}
	bands[0].Limit, d.mapPane.propFreqMHz, d.mapPane.propFreqLabel = ionomaps.LimitCircuit, 3, "3.0 MHz"
	if got := d.propTarget(bands); got[len(got)-1] != "3.0 MHz: too weak at 100 W - go up to 4.0+" {
		t.Errorf("under a LUF the circuit decides, 3 MHz held: %q", got)
	}
}

// TestTheCentreBoxIsForWhatIsShown is the BUILD red team's P-6: the Centre
// box is the answer for the hour, the noise and the station shown - after an
// hour step it is gone until the hour's answer is worked out, never the
// hour before's.
func TestTheCentreBoxIsForWhatIsShown(t *testing.T) {
	d := workCentre(t, stationDash(t))
	if d.propCentreBox() == nil {
		t.Fatal("control: the centre is not answered")
	}
	d = stepAhead(t, d, "shift+right")
	if box := d.propCentreBox(); box != nil {
		t.Errorf("after an hour step the box shows the hour before's answer:\n%s", strings.Join(box, "\n"))
	}
	if d = workCentre(t, d); d.propCentreBox() == nil {
		t.Error("the hour's answer, worked out, is not shown")
	}
}
