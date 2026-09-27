package render

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestKeyCapFallsBackToBrackets(t *testing.T) {
	// With color off (tests run without a tty / NO_COLOR) the chip must
	// degrade to the mock's [key] form — the affordance never disappears
	// (RS-14). Styled output is exercised in live UAT.
	if got := (Opts{}).KeyCap("tab"); got != "[tab]" && !strings.Contains(got, " tab ") {
		t.Fatalf("keycap: %q", got)
	}
	if got := (Opts{ThinBands: true, ASCII: true}).KeyCap("q"); got != "[q]" {
		t.Fatalf("ascii keycap must be plain brackets: %q", got)
	}
}

func TestColorPassStyling(t *testing.T) {
	// UAT session 3.3/3.4/3.6 with color forced on: HI orange / LO cyan,
	// chips bold-white on light grey, group headers carry their muted
	// backgrounds with the brackets as chip edges.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)

	if chip := (Opts{}).KeyCap("tab"); !strings.Contains(chip, "48;2;86;86;86") || !strings.Contains(chip, " tab ") {
		t.Fatalf("keycap must be bold white on #565656: %q", chip)
	}
	r := testRow()
	r.Extended = []DayCell{{Date: "08/26", Hi: f64(30.0), Lo: f64(20.0)}}
	out := (Opts{ThinBands: true, Width: 220, Units: UnitF}).LocationTable([]LocationRow{r}, 1)
	lines := strings.Split(out, "\n")
	for _, code := range []string{"48;2;97;97;97", "48;2;66;94;122", "48;2;66;122;122", "48;2;94;94;122"} {
		if !strings.Contains(lines[0], code) {
			t.Fatalf("group header missing background %s:\n%q", code, lines[0])
		}
	}
	if strings.ContainsAny(stripANSI(lines[0]), "[]") {
		t.Fatalf("brackets are the chip edges — swallowed when styled:\n%q", stripANSI(lines[0]))
	}
	if !strings.Contains(lines[2], "38;5;208") || !strings.Contains(lines[2], "38;5;51") {
		t.Fatalf("row must color HIs orange and LOs cyan:\n%q", lines[2])
	}
	// Styling must never disturb geometry: the row still spans its layout width.
	if w := displayWidth(lines[2]); w > 220 {
		t.Fatalf("styled row overflows: %d", w)
	}
}

// The edition is TINTED, and by its OWN token — a plain-text "Observer" beside a
// gradient wordmark would read as an accident rather than a name.
//
// "WATCHPOST Observer" (HUM LEAD, 2026-08-30): the wordmark keeps its gradient
// and the edition names which experience this build is, so the Broadcaster
// dashboard a later version brings arrives as a different word here rather than
// as a rename.
func TestWordmarkTintsTheEditionWithItsOwnToken(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	marked := Wordmark(EditionObserver)
	// The rendered form, not the raw token: Tint expands a bare palette index
	// ("1;117") into its SGR ("1;38;5;117"), so comparing the token text would
	// pass or fail on the notation rather than on the colour.
	if want := Tint(EditionObserver, Tok(TitleEdition)); !strings.Contains(marked, want) {
		t.Errorf("the edition must be tinted with TitleEdition (%q):\n%q", want, marked)
	}
	// The wordmark keeps the gradient it always had: per-rune, so no single SGR
	// run covers the whole of it.
	if !strings.Contains(marked, "\x1b[1;38;2;") {
		t.Errorf("the wordmark keeps its gradient:\n%q", marked)
	}
	if got := StripSGRForTest(marked); got != "WATCHPOST Observer" {
		t.Errorf("with colour stripped the masthead still reads the name, got %q", got)
	}
	// And an empty edition is the wordmark alone — the ladder's last rung.
	if got := StripSGRForTest(Wordmark("")); got != "WATCHPOST" {
		t.Errorf("an empty edition is the wordmark alone, got %q", got)
	}
	// Colour off, the words survive (R-12a).
	rendering.SetColorEnabledForTest(false)
	if got := Wordmark(EditionObserver); got != "WATCHPOST Observer" {
		t.Errorf("with colour disabled the masthead is plain text, got %q", got)
	}
}

// TestSwatchTextReadsOnEveryColour is UAT-2 U2-18: the temperature key's
// words sat in white on its pale middle bands. Black or white, whichever
// stands out more - never under 4.5:1 on any colour.
func TestSwatchTextReadsOnEveryColour(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })
	for _, c := range [][3]int{{240, 232, 144}, {238, 255, 187}, {221, 255, 255}, {34, 0, 68}, {0, 34, 119}, {255, 136, 68}, {128, 128, 128}} {
		out := SwatchText("68", uint8(c[0]), uint8(c[1]), uint8(c[2]))
		fg := luminance(255, 255, 255)
		if strings.Contains(out, "38;2;0;0;0") {
			fg = luminance(0, 0, 0)
		} else if !strings.Contains(out, "38;2;255;255;255") {
			t.Fatalf("%v: no foreground set: %q", c, out)
		}
		if r := contrastRatio(fg, luminance(c[0], c[1], c[2])); r < 4.5 {
			t.Errorf("%v: the words read at %.2f:1", c, r)
		}
	}
}
