package tty

import (
	"strings"
	"testing"
	"time"

	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// TestThePropagationRowsFollowTheMock is D-157: under the Propagation map,
// the header with the sources' badges, the mode and computed line, and the
// answer for the selected place now - its foF2 and MUF(3000), the bands
// under each - said as upper limits; the map's height as in the weather
// mode it came from.
func TestThePropagationRowsFollowTheMock(t *testing.T) {
	s := &seam{answer: PropagationResult{Snapshot: propSnapAt(time.Now().UTC().Add(-12*time.Minute).Truncate(time.Minute), 18)}}
	d := propDash(t, s, true)
	weather := d.mapBodySize()
	d, cmd := keyCmd(t, d, "P")
	d = runProp(t, d, cmd)
	rows := stripANSITest(strings.Join(d.scrubRows(d.mapTextW()), "\n"))
	for _, want := range []string{"RADIO FREQUENCY PROPAGATION", "SWPC", "PYIRI", "IGRF-14",
		"MODE: MUF(3000)", "COMPUTED", "(12 min ago) · upper limits only",
		"NOW · foF2 6.0 MHz · MUF(3000) 18.0 MHz",
		"Local, to ~400 km (NVIS): 160m 80m 60m",
		"~3,000 km hops through here: up to 20m (18.0 MHz)"} {
		if !strings.Contains(rows, want) {
			t.Errorf("the rows under the map do not say %q:\n%s", want, rows)
		}
	}
	for _, l := range strings.Split(rows, "\n") {
		if strings.Contains(l, "NVIS") && strings.Contains(l, "40m") {
			t.Errorf("40m (7 MHz) is above foF2 6 MHz, yet listed: %q", l)
		}
	}
	words := strings.Join(d.describeLinesAll(), " ")
	for _, want := range []string{"NOW · foF2 6.0 MHz · MUF(3000) 18.0 MHz.", "Local, to ~400 km (NVIS): 160m 80m 60m.", "up to 20m (18.0 MHz)."} {
		if !strings.Contains(words, want) {
			t.Errorf("without the picture the words do not say %q: %q", want, words)
		}
	}
	if got := d.mapBodySize(); got != weather {
		t.Errorf("the map is %v in the Propagation mode and %v in the weather mode", got, weather)
	}
}

// TestThePropagationLegendIsBandOverMHz is D-160: the legend is two lines,
// each class by its band, its lower edge in MHz under it, nothing cut.
func TestThePropagationLegendIsBandOverMHz(t *testing.T) {
	d, _ := drawnDash(t)
	rows := d.scrubRows(d.mapTextW())
	bands, mhz := stripANSITest(rows[0]), stripANSITest(rows[1])
	for _, want := range []string{"MUF(3000)", "160m", "80m", "60m", "40m", "30m", "20m", "17m", "15m", "12m", "10m"} {
		if !strings.Contains(bands, want) {
			t.Errorf("the bands line %q lacks %q", bands, want)
		}
	}
	for _, want := range []string{"MHz", "1.8", "3.5", "5.3", "7", "10.1", "14", "18.07", "21", "24.89", "28"} {
		if !strings.Contains(mhz, want) {
			t.Errorf("the MHz line %q lacks %q", mhz, want)
		}
	}
	if i, j := strings.Index(bands, "20m"), strings.Index(mhz, "14"); i < 0 || i != j {
		t.Errorf("20m stands at %d and its 14 MHz at %d: not one over the other", i, j)
	}
	d = pressMap(t, d, "L")
	if bands := stripANSITest(d.scrubRows(d.mapTextW())[0]); !strings.Contains(bands, "foF2") || strings.Contains(bands, "17m") {
		t.Errorf("foF2's legend %q: want its six bands, 160m to 20m", bands)
	}
	var classes []tuimaps.Class
	for _, e := range d.mapPane.legend {
		if e.Preset == "fof2" {
			classes = e.Classes
		}
	}
	if len(classes) != fof2Bands+1 {
		t.Fatalf("foF2's legend has %d classes; the test reads %d, under 1.8 MHz first", len(classes), fof2Bands+1)
	}
	aligned, ok := bandClasses(classes, fof2Bands)
	if !ok || !strings.HasPrefix(aligned[0].Label, "1.8") || !strings.HasPrefix(aligned[fof2Bands-1].Label, "14") {
		t.Errorf("foF2's bands are keyed by %+v; want 160m by 1.8 MHz and 20m by 14 MHz, the class under 1.8 keying none", aligned)
	}
	if muf, ok := bandClasses(classes[:3], fof2Bands); ok || muf != nil {
		t.Error("a legend of three classes was aligned to six bands")
	}
}
