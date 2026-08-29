package render

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestPanelWidthBound(t *testing.T) {
	out := (Opts{Width: 60}).Panel("Watchpost Weather Radio", "content line")
	if !strings.Contains(out, "Watchpost Weather Radio") {
		t.Fatalf("panel title missing: %s", out)
	}
	for _, line := range strings.Split(stripANSI(out), "\n") {
		if w := displayWidth(line); w > 60 {
			t.Fatalf("panel line %d wide: %q", w, line)
		}
	}
}

func TestBlockPaintsFullWidthAndRearms(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	fg, bg := "38;5;196", "49"
	o := Opts{Width: 40}
	out := o.Block("hi \x1b[1mchip\x1b[0m tail", fg, bg)
	// A hidden bg (49) while the text tone stays.
	if !strings.Contains(out, "38;5;196") {
		t.Fatalf("block tones (fg red, bg hidden): %q", out)
	}
	// An inner reset must re-arm the block tone, never tear the background.
	if !strings.Contains(out, "\x1b[0;"+fg+";"+bg+"m") {
		t.Fatalf("inner reset must re-arm the block: %q", out)
	}
	if !strings.HasSuffix(out, "\x1b[0m") {
		t.Fatalf("block must close clean: %q", out)
	}
	if w := displayWidth(out); w != 40 {
		t.Fatalf("block must paint the full width, got %d", w)
	}
	afg, abg := "38;5;220", "49"
	// No tone of its own (round 4, B-01: the header box rode an invalid
	// "\x1b[250;49m"): padded, and not one escape added.
	if none := o.Block("a \x1b[1mb\x1b[0m c", "", ""); displayWidth(none) != 40 || strings.Count(none, "\x1b[") != 2 || !strings.HasPrefix(none, "a \x1b[1mb\x1b[0m c ") {
		t.Fatalf("an untoned block adds no SGR: %q", none)
	}
	// Color off: content passes through untinted (text carries the signal).
	rendering.SetColorEnabledForTest(false)
	if plain := o.Block("hello", afg, abg); plain != "hello" {
		t.Fatalf("color-off block must pass through: %q", plain)
	}
	rendering.SetColorEnabledForTest(true)
}

func TestModalBlockKeepsTileBGAfterInnerSpans(t *testing.T) {
	// Session-13 regression: a chip or tint inside a modal line must never
	// drop the tile background for the rest of the line (the reset re-arm
	// must carry BOTH the base fg and the tile bg).
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	o := Opts{Width: 40}
	fg, bg := ModalTone(true)
	line := o.Block("chip "+o.KeyCap("esc")+" end", fg, bg)
	if !strings.Contains(line, "\x1b[0;"+fg+";"+bg+"m") {
		t.Fatalf("inner reset must re-arm fg+tile bg: %q", line)
	}
	if strings.Contains(line, ";49m") || strings.Contains(strings.TrimSuffix(line, "\x1b[0m"), "\x1b[0m"+" ") {
		t.Fatalf("no default-bg fallthrough inside the modal line: %q", line)
	}
}

func TestUntitledPanelHasUnbrokenTopRule(t *testing.T) {
	// UAT 68: the About window has no title; the top rule must not carry
	// the empty title's spaces.
	top := strings.Split((Opts{Width: 20}).PanelColored("", "x", ""), "\n")[0]
	if strings.Contains(top, " ") || displayWidth(top) != 20 {
		t.Fatalf("untitled top rule: %q", top)
	}
}

// HUM LEAD UAT 2026-08-27: a window's title reads bold white against the
// tile; the rule around it keeps the panel's tone; a caller-tinted title is
// left alone; the width is unchanged.
func TestPanelTitleIsBoldWhite(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.ResetColorEnabledForTest()
	o := Opts{Width: 40}
	head := strings.SplitN(o.PanelColored("Title", "x", "38;5;250"), "\n", 2)[0]
	if !strings.Contains(head, "\x1b[1;97mTitle\x1b[0m") {
		t.Fatalf("the title carries ModalTitle: %q", head)
	}
	if w := displayWidth(head); w != 40 {
		t.Fatalf("width %d", w)
	}
	pre := Tint("Mine", "38;5;196")
	if head := strings.SplitN(o.PanelColored(pre, "x", ""), "\n", 2)[0]; strings.Count(head, "\x1b[") != 2 {
		t.Fatalf("a pre-tinted title is not wrapped again: %q", head)
	}
	rendering.SetColorEnabledForTest(false)
	if head := strings.SplitN(o.PanelColored("Title", "x", ""), "\n", 2)[0]; !strings.HasPrefix(head, "┌── Title ─") || strings.Contains(head, "\x1b") || displayWidth(head) != 40 {
		t.Fatalf("colour off: plain, same width: %q", head)
	}
}
