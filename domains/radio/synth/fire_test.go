package synth

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestFireSegmentsReadTheScript(t *testing.T) {
	// UAT 114 (HUM LEAD script): the notice with the feeds joined for speech,
	// a two-second pause, the count inside the fire ring, the strongest
	// hotspot, the named fires inside the ring, then those beyond it —
	// phrases without data left out.
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	fr := FireReport{
		Known: true, RadiusKm: 25, IncidentRadiusKm: 50, Lat: 33.24, Lon: -117.29,
		Sources: []string{"NOAA's Hazard Mapping System", "the National Interagency Fire Center", "NASA FIRMS"},
		State: snapshot.FireState{
			AsOf: now,
			Hotspots: []snapshot.Hotspot{
				{Lat: 33.33, Lon: -117.29, DetectedAt: now.Add(-2 * time.Hour), FRPMW: f64(62), DistanceKm: f64(10), Source: snapshot.SourceInfo{ModelOrStation: "GOES-WEST"}},
				{Lat: 33.24, Lon: -117.10, DetectedAt: now.Add(-40 * time.Minute), FRPMW: f64(8), DistanceKm: f64(18)},
			},
			Incidents: []snapshot.Incident{
				{Name: "Timber", Lat: 33.24, Lon: -117.09, Acres: f64(12915), PercentContained: f64(26), Discovered: now.Add(-76 * time.Hour), Source: snapshot.SourceInfo{DistanceKm: f64(19)}},
				{Name: "Convoy", Discovered: now.Add(-3 * time.Hour), Source: snapshot.SourceInfo{DistanceKm: f64(47)}}, // no size, no containment, no point yet
			},
		},
	}
	segs := std.FireSegments("Oceanside, CA", fr, true, now)
	want := []string{
		"This is the Watchpost Fire and Hotspot report for Oceanside, California. This report is derived from data from NOAA's Hazard Mapping System, the National Interagency Fire Center, and NASA FIRMS. Data for this report may be delayed or incomplete, and is not intended for life safety use.",
		"There are currently 2 hotspots within a 16 mile fire ring in your area.",
		"The strongest hotspot is 6 miles north of your location, with a fire radiative power of 62 megawatts, detected 2 hours ago by GOES-West.",
		"Timber is 12 miles east of your location, with a size of 12,915 acres, has been active for 3 days and 4 hours, and is 26 percent contained.",
		"Nearby fires outside of your fire ring that may be worth noting are:",
		"Convoy, at a distance of 29 miles, has been active for 3 hours.",
	}
	if len(segs) != len(want) {
		t.Fatalf("want %d segments, got %d:\n%s", len(want), len(segs), join(segs))
	}
	for i, w := range want {
		if segs[i].Text != w {
			t.Fatalf("segment %d:\n got %q\nwant %q", i, segs[i].Text, w)
		}
	}
	if segs[0].Pause != 2*time.Second || segs[1].Pause != 0 {
		t.Fatal("the notice carries the two-second pause")
	}
	// Metric, no fire: the zero sentence in kilometers; one hotspot reads singular.
	quiet := FireReport{Known: true, RadiusKm: 25, Sources: []string{"NOAA's Hazard Mapping System"}, State: snapshot.FireState{AsOf: now}}
	if s := std.FireSegments("Oceanside, CA", quiet, false, now); len(s) != 2 || s[1].Text != "There are currently no hotspots within a 25 kilometer fire ring in your area." || !strings.Contains(s[0].Text, "data from NOAA's Hazard Mapping System.") {
		t.Fatalf("quiet report: %s", join(s))
	}
	// No fire data at all: skipped — the broadcast goes straight to the tail.
	if s := std.FireSegments("Oceanside, CA", FireReport{}, true, now); s != nil {
		t.Fatalf("no fire data must skip the report, got %s", join(s))
	}
}

func TestFireWords(t *testing.T) {
	for d, want := range map[time.Duration]string{
		5*time.Hour + 35*time.Minute: "5 hours and 35 minutes",
		3*24*time.Hour + 4*time.Hour: "3 days and 4 hours",
		12 * time.Minute:             "12 minutes",
		time.Hour:                    "1 hour",
		49 * time.Hour:               "2 days and 1 hour",
	} {
		if got := durationWords(d); got != want {
			t.Fatalf("durationWords(%v) = %q, want %q", d, got, want)
		}
	}
	if bearingWords(33, -117, 33.5, -118.2) != "west-northwest" || bearingWords(33, -117, 34, -117) != "north" {
		t.Fatal("bearing words")
	}
	if joinAnd([]string{"a"}) != "a" || joinAnd([]string{"a", "b"}) != "a and b" || joinAnd([]string{"a", "b", "c"}) != "a, b, and c" {
		t.Fatal("joinAnd")
	}
	if distanceWords(1.2, true) != "1 mile" || distanceWords(25, true) != "16 miles" || distanceWords(1.4, false) != "1 kilometer" {
		t.Fatal("distance words")
	}
}

func join(segs []Segment) string {
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		out = append(out, s.Text)
	}
	return strings.Join(out, "\n")
}
