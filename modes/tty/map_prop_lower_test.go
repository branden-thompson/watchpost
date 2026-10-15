package tty

import (
	"strings"
	"testing"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// TestAClosedBandIsDrawnByItsKind is D-192 (go-ionomaps D-70): a band the
// lower limit closes is ○ where the day's absorption closes it and ◌ where
// it is too faint even without it; the legend names both; the row's words
// keep its upper limit and add the LUF.
func TestAClosedBandIsDrawnByItsKind(t *testing.T) {
	d := pressMap(t, chartDash(t), "B")
	c := &d.mapPane.propChart
	// The row is written whole: the fixture's snapshot is for the wall clock's
	// hour, whose absorption can close another band at another LUF.
	bands := append([]ionomaps.BandStatus(nil), c.local.Bands...)
	for i := range bands {
		bands[i] = ionomaps.BandStatus{Band: bands[i].Band, Status: ionomaps.AboveUpper, Limit: ionomaps.LimitFoF2, LimitMHz: 6}
		if i < len(hfBands) && hfBands[i].low < 6 {
			bands[i].Status = ionomaps.Open
		}
	}
	bands[0] = ionomaps.BandStatus{Band: bands[0].Band, Status: ionomaps.Absorbed, Limit: ionomaps.LimitAbsorption, LimitMHz: 4.2}
	bands[1] = ionomaps.BandStatus{Band: bands[1].Band, Status: ionomaps.Weak, Limit: ionomaps.LimitCircuit, LimitMHz: 4.2}
	c.local.Bands = bands
	text := stripANSITest(strings.Join(d.propChartLines(d.opts()), "\n"))
	var local string
	for _, l := range strings.Split(text, "\n") {
		if strings.Contains(l, "Local, ~") && local == "" {
			local = l
		}
	}
	cells := []rune(local)
	if !strings.Contains(local, string(propAbsorbedGlyph)) || !strings.Contains(local, string(propWeakGlyph)) || !strings.Contains(local, "foF2 6.0 MHz · LUF 4.2 MHz") {
		t.Errorf("the local row, 160 m absorbed and 80 m weak, reads %q (%d cells)", local, len(cells))
	}
	for _, want := range []string{string(propAbsorbedGlyph) + " absorbed", string(propWeakGlyph) + " weak", string(propClosedGlyph) + " above the upper limit"} {
		if !strings.Contains(text, want) {
			t.Errorf("the legend does not say %q:\n%s", want, text)
		}
	}
	if w := propRowLimitWords([]ionomaps.BandStatus{{Status: ionomaps.Weak, Limit: ionomaps.LimitCircuit, LimitMHz: 9}, {Status: ionomaps.Disturbed, Limit: ionomaps.LimitDRAP, LimitMHz: 3}, {Status: ionomaps.Open, Limit: ionomaps.LimitMUF, LimitMHz: 18}}); w != "MUF 18.0 MHz · LUF 9.0 MHz · D-RAP 3.0 MHz" {
		t.Errorf("a row's limits read %q; the upper, then the LUF, then D-RAP's", w)
	}
}

// TestTheNoiseSettingIsTheUpdates is D-187: the Maps tab's RX noise picker
// steps through P.372's environments, Residential first; the update is asked
// with the one chosen; a snapshot made for another noise is asked again at
// once, and one made for this noise is not; the chart answers are keyed on
// the snapshot's noise, so a rebuilt snapshot at the same time is answered
// anew.
func TestTheNoiseSettingIsTheUpdates(t *testing.T) {
	s := &seam{answer: PropagationResult{Snapshot: stationsSnap(0, ionomaps.Fresh)}}
	d, cmd := keyCmd(t, propDash(t, s, true), "P")
	d = runProp(t, d, cmd)
	if len(s.noises) != 1 || s.noises[0] != ionomaps.NoiseResidential {
		t.Fatalf("the first update was asked with %v; Residential by default", s.noises)
	}
	if propNoiseLabel(d.mapPropNoise) != "Residential" {
		t.Errorf("the Setting starts at %q", propNoiseLabel(d.mapPropNoise))
	}
	var seen []string
	for range propNoiseChoices {
		d = d.cyclePropNoise(true)
		seen = append(seen, propNoiseLabel(d.mapPropNoise))
	}
	if strings.Join(seen, ",") != "City,Rural,Quiet rural,Residential" {
		t.Errorf("the picker steps %v", seen)
	}
	d = d.cyclePropNoise(true) // City
	d, again := d.armPropNoise()
	if again == nil {
		t.Fatal("the Setting changed and the snapshot, made for Residential, was not asked again")
	}
	if runProp(t, d, again); s.noises[len(s.noises)-1] != ionomaps.NoiseCity {
		t.Errorf("the update after choosing City was asked with %v", s.noises[len(s.noises)-1])
	}
	city := stationsSnap(0, ionomaps.Fresh)
	city.Noise = ionomaps.NoiseCity
	d.mapPane.propSnap = &city
	if _, again := d.armPropNoise(); again != nil {
		t.Error("a snapshot made for the Setting's noise was asked again")
	}
	if propNoiseKey(ionomaps.NoiseCity) != "city" || propNoiseByKey("quiet-rural") != ionomaps.NoiseQuietRural || propNoiseByKey("nonsense") != ionomaps.NoiseResidential || propNoiseKey(ionomaps.NoiseResidential) != "" {
		t.Error("the file's names for the environments do not round-trip, or Residential is not the default")
	}
}

// TestTheChartIsAskedAnewForANewNoise is D-187 with FR-1.17: a snapshot
// rebuilt for another noise keeps its time, so the chart's key carries the
// noise and asks again.
func TestTheChartIsAskedAnewForANewNoise(t *testing.T) {
	d := chartDash(t)
	key, ok := d.propChartWanted()
	if !ok {
		t.Fatal("no chart wanted")
	}
	snap := *d.mapPane.propSnap
	snap.Noise = ionomaps.NoiseQuietRural
	d.mapPane.propSnap = &snap
	again, _ := d.propChartWanted()
	if again == key || again.noise != ionomaps.NoiseQuietRural {
		t.Errorf("the chart's key %+v did not change with the noise", again)
	}
	d.surface = SurfaceBroadcaster // the centre is answered as a target on the Broadcaster (D-193)
	if c, _ := d.propCentreWanted(); c.noise != ionomaps.NoiseQuietRural {
		t.Errorf("the centre's key carries noise %v", c.noise)
	}
}
