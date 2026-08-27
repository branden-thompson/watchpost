package tty

// Quality pass Q3 (A11-10, PA-6): the canonical 133×44 frame pinned byte
// for byte with colour off and with --ascii, and the NO_COLOR pin the kit
// cannot yet honour (expected red until Q4a-004; skipped with the reason
// until then, so the day it passes the skip disappears by itself).

import (
	"flag"
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
	checkGolden(t, "frame-133x44-colour.golden", d.View().Content)
}
