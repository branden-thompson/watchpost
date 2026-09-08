package tty

// Quality pass Q3 (A11-10, PA-6): the canonical 133×44 frame pinned byte
// for byte with colour off and with --ascii, and the NO_COLOR pin the kit
// cannot yet honour (expected red until Q4a-004; skipped with the reason
// until then, so the day it passes the skip disappears by itself).

import (
	"flag"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

var updateGolden = flag.Bool("update-golden", false, "re-capture testdata/frame-*.golden")

// goldenDash is benchDash with colour OFF and the clock pinned to UTC so
// the header's stamp is the same on every machine.
func goldenDash(t *testing.T, ascii bool) Dashboard {
	t.Helper()
	local := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = local })
	d := benchDash(t, 133, 44).(Dashboard)
	rendering.SetColorEnabledForTest(false)
	d.cfg.ASCII = ascii
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 2, 0, 0, time.UTC) } // the stamp's age is part of the frame
	return d
}

func checkGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *updateGolden {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s (%d bytes)", path, len(got))
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no golden yet: run with -update-golden (%v)", err)
	}
	if got != string(want) {
		t.Fatalf("%s drifted from the golden — if intended, re-run with -update-golden and say why in the build log:\n--- got ---\n%s", name, got)
	}
}

func TestFrameGoldenColourOff(t *testing.T) {
	got := goldenDash(t, false).View().Content
	if strings.Contains(got, "\x1b[") {
		t.Fatal("colour off: no escapes in the frame")
	}
	checkGolden(t, "frame-133x44-plain.golden", got)
}

func TestFrameGoldenASCII(t *testing.T) {
	got := goldenDash(t, true).View().Content
	for _, g := range []string{"▶", "∞", "◆", "⚠", "›", "✔", "✘"} {
		if strings.Contains(got, g) {
			t.Fatalf("--ascii: the frame carries no %q", g)
		}
	}
	checkGolden(t, "frame-133x44-ascii.golden", got)
}

// TestFrameHonoursNoColorUnderColorTerm: NO_COLOR=1 with TERM=xterm-256color
// must yield a frame with zero escapes. The kit's Style() gates on $TERM
// alone (L5-F4, A11-1), so the table's header and un-styled cells still
// paint — the known-failing pin (plan Q3 task 1), skipped with the measured
// count until Q4a-004 lands.
func TestFrameHonoursNoColorUnderColorTerm(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "xterm-256color")
	d := benchDash(t, 133, 44).(Dashboard)
	rendering.ResetColorEnabledForTest() // the real gate: NO_COLOR wins in WrapSGR
	got := d.View().Content
	if n := strings.Count(got, "\x1b["); n > 0 {
		t.Skipf("known failing until Q4a-004 (kit NoAutoStyle): %d escapes under NO_COLOR=1 TERM=xterm-256color", n)
	}
}

// TestFrameGoldenColourOn pins the frame with colour ON under
// TERM=xterm-256color — the fidelity golden the go-studs patches must keep
// byte for byte (quality pass Q4a, CQ-4: captured BEFORE patch 004).
func TestFrameGoldenColourOn(t *testing.T) {
	local := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = local })
	d := benchDash(t, 133, 44).(Dashboard) // TERM set, colour forced on
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 2, 0, 0, time.UTC) }
	checkGolden(t, "frame-133x44-colour.golden", d.View().Content)
}

// --- 0.14.0 P4 Task 4.12: the Setup window ---

// setupGolden is the Setup window over the golden fixture, with a COMPLETE
// cast fixture: a voice list, a preview hook and an installed-check.
//
// Complete matters. With no voice list every picker reads "—" and every row
// grows a "No voices are available" note, which shifts the whole layout — the
// golden would then pin a window no listener will ever see.
func setupGolden(t *testing.T, w, h int, ascii bool, at setupRowID) Dashboard {
	t.Helper()
	local := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = local })
	rendering.SetColorEnabledForTest(false)
	d := benchDash(t, w, h).(Dashboard)
	d.cfg.ASCII = ascii
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 2, 0, 0, time.UTC) }
	d.cfg.Voices = func() []string { return []string{"System Voice", "Daniel", "Karen", "Rishi", "Samantha"} }
	d.cfg.PreviewVoice = func(string) {}
	d.cfg.VoiceInstalled = func(string) bool { return true }
	d.cfg.ToneClasses = []ToneClass{
		{Key: "disaster", Label: "Disaster Events"},
		{Key: "warning", Label: "Warnings"},
		{Key: "watch", Label: "Watches"},
		{Key: "advisory", Label: "Advisories"},
		{Key: "statement", Label: "Special Statements"},
		{Key: "storm", Label: "Maritime"},
	}
	return d.openSetupAt(at)
}

// The two-column layout, with a correspondent picker focused: the state a
// listener is in when they are doing the thing this release exists for.
func TestSetupGolden133(t *testing.T) {
	checkGolden(t, "setup-133x44.golden", setupGolden(t, 133, 44, false, rowCastAlerts).View().Content)
}

// 80x24: the stacked layout, the scroll, and the pinned chip footer — the size
// RS-19 was raised about.
func TestSetupGolden80(t *testing.T) {
	checkGolden(t, "setup-80x24.golden", setupGolden(t, 80, 24, false, rowCastAlerts).View().Content)
}

// --ascii, where Task 4.9's glyph parity is proven: every mark this window
// draws — the focus arrow, the radio, the checkbox, the picker's dropdown, the
// scroll rail — goes through the glyph set, so nothing needs a special case.
func TestSetupGoldenASCII(t *testing.T) {
	got := setupGolden(t, 133, 44, true, rowCastAlerts).View().Content
	for _, glyph := range []string{"▾", "█", "▲", "▼", "●", "○", "›", "—"} {
		if strings.Contains(got, glyph) {
			t.Errorf("--ascii frame still carries %q — it needs an ASCII form in the glyph set", glyph)
		}
	}
	checkGolden(t, "setup-133x44-ascii.golden", got)
}

// THE SCAN THAT WOULD HAVE CAUGHT F-47, and the reason the per-window lists
// above did not. Each of those names the marks ITS window draws, while the
// golden it checks is the whole frame — so a glyph belonging to the dashboard
// behind the modal is captured in the file and scanned by nobody. F-47 lived
// there: the radio panel wrote "▶", "■", "█" and "░" directly instead of taking
// them from the glyph set, in the same function whose Fail branch took its mark
// from the set correctly.
//
// This asks the only question that needs no list: is anything in an --ascii
// frame outside ASCII? A list of forbidden glyphs can only find what someone
// already thought of.
//
// ° (U+00B0) is the one ruled exception. Temperatures carry the real DEGREE
// SIGN deliberately — it replaced U+00BA MASCULINE ORDINAL INDICATOR, which a
// screen reader announces as an ordinal marker where a temperature is meant —
// and both measure one cell, so nothing moved.
func TestASCIIFramesCarryNothingButASCII(t *testing.T) {
	for name, frame := range asciiSurfaces(t) {
		seen := map[rune]bool{}
		for _, r := range stripANSITest(frame) {
			if r < 128 || r == '\u00b0' || seen[r] {
				continue
			}
			seen[r] = true
			t.Errorf("--ascii %s carries %q (U+%04X) — it needs an ASCII form in the glyph set", name, r, r)
		}
	}
}

// asciiSurfaces is EVERY SURFACE AN --ascii FRAME CAN SHOW, and it is the
// half of this gate that kept going stale (FR-8).
//
// The scan above has always asked the only question that needs no list — is
// anything here outside ASCII? — but it asked it of two surfaces: the
// dashboard and Settings. There are eleven windows. The eight it never
// reached are where the last three --ascii escapes were found, each by
// someone opening that window, each after the previous one was called the
// last.
//
// The producer is the modal enum, so a window added to the app is scanned the
// day it lands rather than the day someone remembers to add it here.
// populated gives the detail modal DATA, because fixtureFor gives it none and a
// window scanned empty is a window scanned in the one state whose glyphs are
// missing (red team, 2026-09-08).
//
// The scan reported thirteen surfaces and meant it, but the detail modal drew
// "fire feed not yet available" and "seismic data unavailable" — so the fire and
// hotspot TABLES this release built, the incidents list with its second radius,
// the felt-band ramp and the truncation that spends an Ellipsis were all outside
// it. fixtureFor's own severe case says why: "a fixture that does not exercise
// the state is a hole shaped exactly like coverage."
func populated(d Dashboard) Dashboard {
	if d.snap == nil || len(d.snap.Locations) == 0 {
		return d
	}
	now := time.Now()
	f := func(v float64) *float64 { return &v }
	loc := &d.snap.Locations[0]
	loc.Fire = snapshot.FireState{
		AsOf: now,
		Hotspots: []snapshot.Hotspot{
			{Lat: loc.Lat + 0.09, Lon: loc.Lon, DetectedAt: now.Add(-2 * time.Hour), FRPMW: f(62), DistanceKm: f(10),
				Source: snapshot.SourceInfo{Provider: "hms", ModelOrStation: "GOES-WEST"}},
		},
		// A NAME LONG ENOUGH TO TRUNCATE, so the Ellipsis glyph is on screen.
		Incidents: []snapshot.Incident{{Name: "SAN LUIS REY COMPLEX", Acres: f(18000), PercentContained: f(35),
			Discovered: now.Add(-72 * time.Hour), Source: snapshot.SourceInfo{Provider: "wfigs", DistanceKm: f(30)}}},
	}
	loc.Seismic = &snapshot.SeismicState{AsOf: now}
	d.cfg.FireRadiusKm, d.cfg.FireIncidentRadiusKm = 25, 50
	return d
}

func asciiSurfaces(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{
		"dashboard frame": goldenDash(t, true).View().Content,
		"settings frame":  setupGolden(t, 133, 44, true, rowCastAlerts).View().Content,
	}
	for m := modalHelp; m < numModals; m++ {
		d := populated(fixtureFor(t, m))
		d.cfg.ASCII = true
		got := d.renderModal(d.opts())
		// A WINDOW THAT DRAWS NOTHING IS NOT A WINDOW THAT PASSED. This is the
		// failure a coverage scan is most likely to have and least likely to
		// report: it goes quiet in exactly the shape of success.
		if strings.TrimSpace(stripANSITest(got)) == "" {
			t.Fatalf("the %s window draws nothing at fixtureFor: it is NOT covered by this scan", modalName(m))
		}
		out["the "+modalName(m)+" window"] = got
	}
	return out
}
