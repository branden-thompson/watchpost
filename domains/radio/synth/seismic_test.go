package synth

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// recVoice records every line it is asked to speak, so a test can prove the
// seismic narration was actually rendered to audio.
type recVoice struct {
	mu   sync.Mutex
	said []string
}

func (v *recVoice) Name() string { return "rec" }
func (v *recVoice) Rate() int    { return 22050 }
func (v *recVoice) Say(_ context.Context, text string) ([]byte, error) {
	v.mu.Lock()
	v.said = append(v.said, text)
	v.mu.Unlock()
	return make([]byte, 22050/10*2), nil // 100 ms of mono PCM
}
func (v *recVoice) spoke(substr string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, s := range v.said {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// R6 gate (P4): the synth broadcast, composed for a location with quakes,
// plays end-to-end and the seismic report is rendered to audio — after the
// fire report, before the sign-off.
func TestSynthPlaysTheSeismicBroadcast(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	sr := SeismicReport{Known: true, Lat: 35.62, Lon: -117.67, State: snapshot.SeismicState{AsOf: now, Quakes: []snapshot.Quake{
		quake(5.1, 141, 15, "N", 72*time.Hour, now),
		quake(4.2, 30, 8, "NE", 2*time.Hour, now),
	}}}
	segs := Compose(snapshot.Location{Label: "Ridgecrest, CA"}, nil, now, true, "rec", Station{}, FireReport{}, sr)
	// Order: the seismic report sits before the tail.
	texts := segmentsText(segs)
	seismicAt, tailAt := -1, -1
	for i, tx := range texts {
		if strings.Contains(tx, "Seismic Activity report") {
			seismicAt = i
		}
		if strings.HasPrefix(segs[i].Key, "tail:") {
			tailAt = i
		}
	}
	if seismicAt < 0 || tailAt < 0 || seismicAt > tailAt {
		t.Fatalf("the seismic report must play before the sign-off (seismic@%d tail@%d)", seismicAt, tailAt)
	}
	// Render one cycle to PCM through the real Source and prove it played.
	v := &recVoice{}
	src, err := NewSource(v, func(context.Context) ([]Segment, error) { return segs, nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	src.gap = time.Millisecond
	src.Loop(false) // one cycle then EOF
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	n, err := io.Copy(io.Discard, src.Open(ctx))
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("the broadcast produced no audio")
	}
	if !v.spoke("Watchpost Radio Seismic Activity report for Ridgecrest") || !v.spoke("A magnitude 5.1 earthquake") || !v.spoke("please visit") {
		t.Fatalf("the seismic notice, a quake and the tail must all reach the voice:\n%v", v.said)
	}
}

func quake(mag, distKm, depthKm float64, bearing string, ago time.Duration, now time.Time) snapshot.Quake {
	return snapshot.Quake{Mag: mag, DistanceKm: distKm, DepthKm: depthKm, Bearing: bearing, At: now.Add(-ago)}
}

func TestSeismicSegmentsReadTheScript(t *testing.T) {
	// P4 (HUM LEAD script): the notice with the USGS source, a two-second
	// pause, the count, the strongest quakes largest-first with magnitude,
	// distance + bearing, depth and age, each closing with the magnitude's
	// felt-likelihood, then the where-to-learn-more tail.
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	sr := SeismicReport{Known: true, Lat: 33.24, Lon: -117.29, State: snapshot.SeismicState{AsOf: now, Quakes: []snapshot.Quake{
		quake(5.1, 141, 15, "N", 72*time.Hour, now),
	}}}
	segs := SeismicSegments("Oceanside, CA", sr, true, now)
	want := []string{
		"This is the Watchpost Radio Seismic Activity report for Oceanside, California. This report is derived from the United States Geological Survey real-time GeoJSON earthquake notification service. Data for this report may be delayed or incomplete, and is not intended for life safety use.",
		"There has been 1 nearby quake in the last seven days:",
		"A magnitude 5.1 earthquake, 88 miles north of your location, at a depth of 15 kilometers, recorded 3 days ago. A quake of this magnitude has a strong likelihood of being felt when it occurs.",
		"For additional and up-to-date information regarding earthquakes in your area, please visit https://earthquake.usgs.gov/earthquakes/map",
	}
	if len(segs) != len(want) {
		t.Fatalf("want %d segments, got %d:\n%s", len(want), len(segs), segmentsText(segs))
	}
	for i, w := range want {
		if segs[i].Text != w {
			t.Fatalf("segment %d:\n got %q\nwant %q", i, segs[i].Text, w)
		}
	}
	if segs[0].Pause != seismicPause {
		t.Fatalf("the notice must be followed by the 2-second pause, got %s", segs[0].Pause)
	}
}

func TestSeismicSkippedWithoutEntries(t *testing.T) {
	now := time.Now()
	// No feed yet (cold): skipped.
	if s := SeismicSegments("X", SeismicReport{}, true, now); s != nil {
		t.Fatalf("an unknown report is skipped entirely, got %d segments", len(s))
	}
	// Answered but empty: still skipped — the report plays only with entries.
	empty := SeismicReport{Known: true, State: snapshot.SeismicState{AsOf: now}}
	if s := SeismicSegments("X", empty, true, now); s != nil {
		t.Fatalf("no quakes ⇒ no report, got %d segments", len(s))
	}
}

func TestSeismicCountAndOverflowCapAtThree(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	qs := []snapshot.Quake{
		quake(5.1, 141, 15, "N", 72*time.Hour, now),
		quake(4.2, 30, 8, "NE", 2*time.Hour, now),
		quake(3.6, 20, 5, "SSW", 26*time.Hour, now),
		quake(2.8, 6, 3, "E", 90*time.Minute, now),
		quake(1.4, 2, 2, "W", 30*time.Minute, now),
	}
	segs := segmentsText(SeismicSegments("Ridgecrest, CA", SeismicReport{Known: true, State: snapshot.SeismicState{AsOf: now, Quakes: qs}}, true, now))
	joined := strings.Join(segs, "\n")
	if !strings.Contains(joined, "There have been 5 nearby quakes in the last seven days:") {
		t.Fatalf("the count is the full total (5):\n%s", joined)
	}
	// Exactly three quakes are read (the strongest, largest-first).
	spoken := strings.Count(joined, "A magnitude ")
	if spoken != 3 {
		t.Fatalf("only the strongest 3 are read, got %d:\n%s", spoken, joined)
	}
	if !strings.Contains(joined, "A magnitude 5.1 earthquake") || !strings.Contains(joined, "A magnitude 3.6 earthquake") || strings.Contains(joined, "A magnitude 2.8 earthquake") {
		t.Fatalf("reads M5.1/M4.2/M3.6, not M2.8:\n%s", joined)
	}
	if !strings.Contains(joined, "and 2 more recent quakes, which can be found in the Ridgecrest, California details report in the Watchpost CLI application view.") {
		t.Fatalf("the overflow line names the remaining count and the location:\n%s", joined)
	}
}

func TestFeltLikelihoodTiers(t *testing.T) {
	rows := []struct {
		mag  float64
		word string
	}{
		{2.8, "low"}, {3.4, "low"}, // below feeling (< 3.5)
		{3.5, "moderate"}, {4.4, "moderate"}, // might feel it (3.5–4.5)
		{4.5, "strong"}, {4.9, "strong"}, {5.0, "strong"}, {6.5, "strong"}, // almost certainly felt + significant (≥ 4.5) — matches the screen label (REVIEW P5 F2)
	}
	for _, r := range rows {
		got := feltLikelihood(r.mag)
		if !strings.Contains(got, r.word+" likelihood") {
			t.Fatalf("M%.1f felt-likelihood = %q, want %q", r.mag, got, r.word)
		}
	}
}

func TestSeismicOverflowNounAgreesWithCount(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	// Exactly four quakes: three read, one overflow — singular "quake".
	four := make([]snapshot.Quake, 4)
	for i := range four {
		four[i] = quake(5.0-float64(i)*0.5, 20, 5, "N", time.Hour, now)
	}
	one := strings.Join(segmentsText(SeismicSegments("Town, CA", SeismicReport{Known: true, State: snapshot.SeismicState{AsOf: now, Quakes: four}}, true, now)), "\n")
	if !strings.Contains(one, "and 1 more recent quake,") || strings.Contains(one, "and 1 more recent quakes") {
		t.Fatalf("exactly one overflow reads singular 'quake':\n%s", one)
	}
	// Five quakes: two overflow — plural "quakes".
	five := append(four, quake(1.2, 5, 2, "E", time.Hour, now))
	two := strings.Join(segmentsText(SeismicSegments("Town, CA", SeismicReport{Known: true, State: snapshot.SeismicState{AsOf: now, Quakes: five}}, true, now)), "\n")
	if !strings.Contains(two, "and 2 more recent quakes,") {
		t.Fatalf("two overflow reads plural 'quakes':\n%s", two)
	}
}

func TestQuakeSentenceOmitsMissingFields(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	// A quake with no depth and no bearing: those clauses are left out.
	q := snapshot.Quake{Mag: 4.0, At: now.Add(-time.Hour)}
	s := quakeSentence(q, true, now)
	if strings.Contains(s, "depth") || strings.Contains(s, "of your location") {
		t.Fatalf("missing depth/bearing must be omitted, not guessed: %q", s)
	}
	if !strings.Contains(s, "A magnitude 4.0 earthquake") || !strings.Contains(s, "recorded 1 hour ago") {
		t.Fatalf("magnitude and age still read: %q", s)
	}
}

func segmentsText(segs []Segment) []string {
	out := make([]string, len(segs))
	for i, s := range segs {
		out[i] = s.Text
	}
	return out
}
