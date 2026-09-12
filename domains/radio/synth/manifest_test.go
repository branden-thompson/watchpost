package synth

// manifest_test.go — what a read reports about itself (D-87).

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// A ZERO COUNT IS ONLY A FACT WHEN ITS OWN FEED ANSWERED, which is the rule
// FireReport already states — and the manifest is where breaking it would do the
// most damage, because that line is what the operator decides on.
//
// "0 Hotspots" from a provider that was DOWN tells them the opposite of the
// truth, and it reads exactly like a measurement.
func TestFireCountsReportOnlyWhatTheFeedsAnswered(t *testing.T) {
	both := FireReport{HotspotsKnown: true, IncidentsKnown: true,
		State: snapshot.FireState{Hotspots: make([]snapshot.Hotspot, 3), Incidents: make([]snapshot.Incident, 10)}}
	if got := fireCounts(both); got != "3 Hotspots / 10 incidents" {
		t.Errorf("both feeds answered: got %q", got)
	}

	// ONE FEED DOWN LISTS THE OTHER AND SAYS NOTHING ABOUT THE SILENT ONE.
	half := both
	half.IncidentsKnown = false
	if got := fireCounts(half); got != "3 Hotspots" {
		t.Errorf("with the incident feed silent: got %q", got)
	}

	// NEITHER ANSWERED: the line is empty rather than "0 / 0".
	if got := fireCounts(FireReport{}); got != "" {
		t.Errorf("with no feed answering: got %q, want nothing", got)
	}

	// AND A ZERO THE FEED DID ANSWER IS REPORTED, because that is news.
	quiet := FireReport{HotspotsKnown: true, IncidentsKnown: true}
	if got := fireCounts(quiet); got != "0 Hotspots / 0 incidents" {
		t.Errorf("a feed that answered zero: got %q", got)
	}
}

// ONE IS SINGULAR. "1 Hotspots" on a safety surface reads as a rendering fault
// rather than as a number.
func TestACountOfOneReadsAsOne(t *testing.T) {
	one := FireReport{HotspotsKnown: true, State: snapshot.FireState{Hotspots: make([]snapshot.Hotspot, 1)}}
	if got := fireCounts(one); got != "1 Hotspot" {
		t.Errorf("got %q, want the singular", got)
	}
	if got := seismicCounts(SeismicReport{Known: true, State: snapshot.SeismicState{Quakes: make([]snapshot.Quake, 1)}}); got != "1 Quake" {
		t.Errorf("got %q, want the singular", got)
	}
}

// A SEISMIC REPORT NOBODY ANSWERED FOR COUNTS NOTHING, for fireCounts' reason.
func TestSeismicCountsSayNothingWhenTheFeedDidNot(t *testing.T) {
	if got := seismicCounts(SeismicReport{}); got != "" {
		t.Errorf("an unanswered seismic feed reported %q", got)
	}
}

// THE FORECAST'S SPAN IS DATED FROM THE PRODUCT, NOT FROM NOW. A product fetched
// at midnight is still the day's forecast, and dating it from the clock would
// shift the span across midnight while the words stayed the same.
func TestTheForecastSpanIsDatedFromTheProduct(t *testing.T) {
	issued := time.Date(2026, 9, 11, 4, 0, 0, 0, time.UTC)
	midnight := time.Date(2026, 9, 12, 0, 5, 0, 0, time.UTC)

	got := productSpan(Product{Issued: issued}, midnight)
	if got != "09/11 - 09/17" {
		t.Errorf("got %q, want the product's own week", got)
	}
	// A PRODUCT WITH NO ISSUE TIME falls back to now rather than to the epoch.
	if got := productSpan(Product{}, midnight); got != "09/12 - 09/18" {
		t.Errorf("an undated product reported %q", got)
	}
}

// EVERY SOURCE IN A READ NAMES ITSELF EXACTLY ONCE. A report is many segments
// and one line on the card; tagging every segment would list the same source
// nine times and push the others off the manifest.
func TestEachSourceTagsItsFirstSegmentOnly(t *testing.T) {
	c := Composer{}
	loc := snapshot.Location{Label: "Oceanside, CA"}
	// A PRODUCT LONG ENOUGH TO BE SEVERAL SEGMENTS, which is the whole premise:
	// a one-segment forecast cannot tell "the first" from "every one", and the
	// first draft of this test used one — a plant that tagged every segment
	// SURVIVED it.
	long := strings.Repeat("The forecast calls for clear skies and light winds through the period. ", 12)
	segs := c.Compose(loc, []Product{{ID: "p1", Type: "AFD", Text: long}},
		time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC), true, "Voice", Station{},
		Reports{Fire: FireReport{Known: true, HotspotsKnown: true}}, 0)

	forecastSegs := 0
	for _, s := range segs {
		if strings.HasPrefix(s.Key, "AFD:") {
			forecastSegs++
		}
	}
	if forecastSegs < 2 {
		t.Fatalf("the fixture produced %d forecast segments; with one, this test cannot fail", forecastSegs)
	}

	seen := map[string]int{}
	for _, s := range segs {
		if s.Source != "" {
			seen[s.Source]++
		}
	}
	if seen[forecastSource] != 1 {
		t.Errorf("the forecast named itself %d times; a report is one line", seen[forecastSource])
	}
	for src, n := range seen {
		if n != 1 {
			t.Errorf("%s named itself %d times", src, n)
		}
	}
}
