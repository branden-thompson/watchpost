package tty

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// detailWithSeismic drives a snapshot with a SEISMIC state through Update and
// returns the rendered detail lines (raw, with ANSI).
func detailWithSeismic(t *testing.T, ss *snapshot.SeismicState) string {
	t.Helper()
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Seismic = ss
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.modal = modalDetails
	return strings.Join(d.detailLines(), "\n")
}

func TestDetailSeismicSectionShowsGraduatedRows(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	now := time.Now()
	ss := &snapshot.SeismicState{AsOf: now, Quakes: []snapshot.Quake{
		{Mag: 5.1, Place: "N ridge", DepthKm: 15, At: now.Add(-72 * time.Hour), DistanceKm: 141, Bearing: "N"},  // ◉ Significant
		{Mag: 4.2, Place: "NE fault", DepthKm: 8, At: now.Add(-2 * time.Hour), DistanceKm: 19, Bearing: "NE"},   // ● Might feel it
		{Mag: 2.8, Place: "SSW", DepthKm: 3, At: now.Add(-26 * time.Hour), DistanceKm: 6, Bearing: "SSW"},       // ○ Below feeling
		{Mag: 1.4, Place: "underfoot", DepthKm: 2, At: now.Add(-90 * time.Minute), DistanceKm: 2, Bearing: "E"}, // shown too — the whole list (P4)
	}}
	raw := detailWithSeismic(t, ss)
	joined := stripANSITest(raw)
	for _, want := range []string{
		"SEISMIC │ 4 nearby in the last 7 days",
		"(USGS)",
		"◉ M5.1", "Significant",
		"● M4.2", "Might feel it",
		"○ M2.8", "Below feeling",
		"○ M1.4", // the 4th quake is listed too (the radio sends users here for the full list)
		"depth 15 km",
		"3d ago", "2h ago", "1d ago",
		"NE", "SSW",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("SEISMIC section missing %q:\n%s", want, joined)
		}
	}
	// Largest magnitude first (the provider sorts; the section preserves it).
	if strings.Index(joined, "M5.1") > strings.Index(joined, "M4.2") {
		t.Fatal("largest magnitude must render first")
	}
	// The mark reads in the violet SeismicMark tone (colour on).
	if !strings.Contains(raw, render.Tint("◉", render.Tok(render.SeismicMark))) {
		t.Fatalf("the significant mark must read in the SeismicMark tone:\n%q", raw)
	}
}

func TestDetailSeismicUnavailableVsQuiet(t *testing.T) {
	rendering.SetColorEnabledForTest(false)
	// Cold / down feed (AsOf zero): "unavailable", never a fake "none".
	if got := stripANSITest(detailWithSeismic(t, &snapshot.SeismicState{})); !strings.Contains(got, "SEISMIC │ Recent       seismic data unavailable") && !strings.Contains(got, "seismic data unavailable") {
		t.Fatalf("a cold feed reads 'unavailable':\n%s", got)
	}
	// nil (no state at all) is also unavailable.
	if got := stripANSITest(detailWithSeismic(t, nil)); !strings.Contains(got, "seismic data unavailable") {
		t.Fatalf("no state reads 'unavailable':\n%s", got)
	}
	// Answered and empty: the honest quiet answer.
	now := time.Now()
	if got := stripANSITest(detailWithSeismic(t, &snapshot.SeismicState{AsOf: now})); !strings.Contains(got, "no recent seismic activity") {
		t.Fatalf("an answered-empty feed reads 'no recent seismic activity':\n%s", got)
	}
}

func TestDetailSeismicASCIIGlyphs(t *testing.T) {
	rendering.SetColorEnabledForTest(false)
	now := time.Now()
	ss := &snapshot.SeismicState{AsOf: now, Quakes: []snapshot.Quake{
		{Mag: 5.1, Place: "a", DepthKm: 15, At: now.Add(-72 * time.Hour), DistanceKm: 141, Bearing: "N"},
		{Mag: 4.2, Place: "b", DepthKm: 8, At: now.Add(-2 * time.Hour), DistanceKm: 19, Bearing: "NE"},
		{Mag: 2.8, Place: "c", DepthKm: 3, At: now.Add(-26 * time.Hour), DistanceKm: 6, Bearing: "SSW"},
	}}
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Seismic = ss
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.cfg.ASCII = true // the --ascii glyph set
	d.modal = modalDetails
	joined := stripANSITest(strings.Join(d.detailLines(), "\n"))
	for _, want := range []string{"O M5.1", "o M4.2", ". M2.8"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("--ascii ramp .oO missing %q:\n%s", want, joined)
		}
	}
	for _, no := range []string{"◉", "●", "○"} {
		if strings.Contains(joined, no) {
			t.Fatalf("--ascii must not emit the unicode glyph %q", no)
		}
	}
}

func TestDetailSeismicTsunamiReadsWarning(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	now := time.Now()
	ss := &snapshot.SeismicState{AsOf: now, Quakes: []snapshot.Quake{
		{Mag: 3.2, Place: "coast", DepthKm: 5, At: now.Add(-time.Hour), DistanceKm: 10, Bearing: "W", Tsunami: true},
	}}
	raw := detailWithSeismic(t, ss)
	if !strings.Contains(stripANSITest(raw), "Tsunami") {
		t.Fatalf("a tsunami quake labels the reason:\n%s", stripANSITest(raw))
	}
	// A tsunami reads in the warning tone regardless of its (low) magnitude —
	// the ○ below-feeling glyph, but tinted danger, not the violet mark.
	if !strings.Contains(raw, render.Tint("○", render.Tok(render.AlertDanger))) {
		t.Fatalf("a tsunami quake's mark must read in the warning tone:\n%q", raw)
	}
	// Under --ascii the warn label uses an ASCII hyphen, never a unicode em-dash
	// (REVIEW P5 finding: the em-dash leaked into the ascii path).
	m := dash(t)
	s2 := snap()
	s2.Locations[0].Seismic = ss
	m2, _ := m.Update(SnapshotMsg{Snap: s2})
	d := m2.(Dashboard)
	d.cfg.ASCII = true
	d.modal = modalDetails
	asciiOut := stripANSITest(strings.Join(d.detailLines(), "\n"))
	if strings.Contains(asciiOut, "—") {
		t.Fatalf("--ascii must not emit an em-dash:\n%s", asciiOut)
	}
	if !strings.Contains(asciiOut, "Tsunami - Below feeling") {
		t.Fatalf("--ascii warn label uses an ASCII hyphen:\n%s", asciiOut)
	}
}
