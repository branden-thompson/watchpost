package synth

import (
	"flag"
	"fmt"
	"github.com/branden-thompson/watchpost/platform/render"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

var updateGolden = flag.Bool("update-golden", false, "re-capture testdata/cycle.golden")

// The composed cycle, as a golden of KEYS, ROLES and PAUSES.
//
// FR-2 says a 0.14.0 broadcast is 0.13.0's except for three named deltas. That
// claim is only worth something if it is machine-checked: a golden of the
// segment list catches a role tagged wrongly, a pause dropped, a segment
// reordered or a key changed — none of which any single assertion above would
// notice, and all of which a listener would.
//
// Text is deliberately NOT in the golden. The wording is the HUM LEAD's and
// changes without the structure changing; pinning it here would make every
// script edit look like an architectural regression.
func TestComposedCycleGolden(t *testing.T) {
	loc := snapshot.Location{
		Label: "Oceanside, CA",
		Harmonized: snapshot.Conditions{
			Condition: "partly_cloudy", Temp: f64(22.8), HumidityPct: f64(66),
			Source: snapshot.SourceInfo{Provider: "nws"},
		},
		Alerts: []snapshot.Alert{{ID: "a1", Headline: "Heat Advisory", Description: "* WHAT...Hot."}},
	}
	now := time.Date(2026, 8, 24, 16, 5, 0, 0, time.UTC)
	products := []Product{{ID: "p1", Type: "ZFP", Text: ".TONIGHT...Mostly clear. Lows 66 to 69.\n\n$$"}}
	sr := SeismicReport{Known: true, Lat: 35.62, Lon: -117.67, State: snapshot.SeismicState{AsOf: now, Quakes: []snapshot.Quake{
		quake(4.2, 30, 8, "NE", 2*time.Hour, now),
	}}}
	segs := std.Compose(loc, products, now, true, "Samantha", Station{Callsign: "KEC62"}, Reports{Seismic: sr}, render.Clock12)

	var b strings.Builder
	for _, s := range segs {
		fmt.Fprintf(&b, "%-28s role=%-24s pause=%s\n", s.Key, s.Role.Key(), s.Pause)
	}
	got := b.String()

	path := filepath.Join("testdata", "cycle.golden")
	if *updateGolden {
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Log("golden re-captured; review the diff line by line")
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no golden yet: run with -update-golden (%v)", err)
	}
	if got != string(want) {
		t.Errorf("the composed cycle changed:\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}
