package render

// Quality pass Q4a: the kit patches, pinned from the app's side (the kit's
// own tests are not carried): 001 the lazy terminal probe, 003 the
// composite-aware SGR, 008 the bounded loops, and Plain's variation-selector
// drop (A11-8).

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/term"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestTerminalProbeIsLazyAndFallsBackToTheEnvironment(t *testing.T) {
	if caps := rendering.DetectTerminalCapabilities(); caps.Width != 0 || caps.Height != 0 {
		t.Fatalf("capability detection must not probe the terminal size (001): %dx%d", caps.Width, caps.Height)
	}
	tf := rendering.NewTextFormatter()
	if caps := tf.GetCapabilities(); caps.Width <= 0 || caps.Height <= 0 {
		t.Fatalf("the first GetCapabilities resolves a size: %dx%d", caps.Width, caps.Height)
	}
	tty, err := os.Open("/dev/tty")
	haveTTY := err == nil
	if haveTTY {
		_ = tty.Close()
	}
	if haveTTY || term.IsTerminal(int(os.Stdin.Fd())) || term.IsTerminal(int(os.Stdout.Fd())) {
		t.Skip("a real terminal answers the probe; the environment fallback is tested where there is none (CI)")
	}
	t.Setenv("COLUMNS", "123")
	t.Setenv("LINES", "45")
	if w, h := rendering.GetTerminalSize(); w != 123 || h != 45 {
		t.Fatalf("no terminal: COLUMNS/LINES win, got %dx%d", w, h)
	}
	t.Setenv("COLUMNS", "")
	t.Setenv("LINES", "")
	if w, h := rendering.GetTerminalSize(); w != 80 || h != 24 {
		t.Fatalf("no terminal, no environment: 80x24, got %dx%d", w, h)
	}
}

func TestSGRConsumesQualifiedComposites(t *testing.T) {
	rows := map[string]string{
		"1;38;5;220":         "\x1b[1;38;5;220m", // was 1;38;5;38;5;38;5;220 (L5-F5)
		"38;5;208":           "\x1b[38;5;208m",
		"48;2;86;86;86":      "\x1b[48;2;86;86;86m",
		"1;97;48;2;86;86;86": "\x1b[1;97;48;2;86;86;86m",
		"38;2;10;20;30;1":    "\x1b[38;2;10;20;30;1m",
		"245":                "\x1b[38;5;245m", // the classification of bare codes is unchanged
		"97":                 "\x1b[97m",
		"93;3":               "\x1b[93;3m",
		"38":                 "\x1b[38;5;38m", // a bare 38 with no qualifier is still the band code it always was
		"38;5":               "\x1b[38;5;38;5m",
		"38;2;1;2":           "\x1b[38;5;38;2;1;2m", // a malformed composite classifies element by element, as before
	}
	for in, want := range rows {
		if got := rendering.SGR(in); got != want {
			t.Errorf("SGR(%q) = %q, want %q", in, got, want)
		}
	}
	if got, want := rendering.SGR("245", "1"), "\x1b[38;5;245;1m"; got != want {
		t.Errorf("multi-param join: %q, want %q", got, want)
	}
	// The escape is built in one pre-sized buffer: two allocations (the
	// buffer and the variadic slice), down from five (L5-F3).
	if n := testing.AllocsPerRun(50, func() { _ = rendering.SGR("1;38;5;220") }); n > 2 {
		t.Errorf("SGR allocates %.0f times, want ≤ 2 (L5-F3)", n)
	}
}

func TestTintAcceptsCompositesSinceTheKitConsumesThem(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.ResetColorEnabledForTest()
	for _, params := range []string{"1;38;5;220", "38;2;190;84;84", "1;97;48;2;86;86;86"} {
		if Tint("x", params) != TintRaw("x", params) {
			t.Errorf("Tint(%q) must equal the raw form now: %q vs %q", params, Tint("x", params), TintRaw("x", params))
		}
	}
	// The kit still classifies the SGR defaults 39/49 as 256-palette codes
	// (its D-28 rule), so compositions that carry them stay on TintRaw.
	if got := TintRaw("x", "38;5;196;49"); got != "\x1b[38;5;196;49mx\x1b[0m" {
		t.Errorf("raw stays raw: %q", got)
	}
}

func TestStripANSIIsBoundedByTheEscapeCount(t *testing.T) {
	f := rendering.NewANSIFormatter()
	in := strings.Repeat("\x1b[1m", 3) + "text" + "\x1b[0m" + "\x1b[" // a trailing torn escape must not loop
	if got := f.StripANSI(in); got != "text\x1b[" {
		t.Fatalf("StripANSI: %q", got)
	}
}

func TestPlainDropsVariationSelectors(t *testing.T) {
	if got := Plain("⚠️ warn ☀︎"); got != "⚠ warn ☀" {
		t.Fatalf("Plain keeps the glyphs, drops the selectors: %q", got)
	}
}
