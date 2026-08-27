package render

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestAlertBannerSeverityIsText(t *testing.T) {
	a := snapshot.Alert{Event: "Extreme Heat Watch", Severity: "severe", Headline: "until Friday"}
	out := (Opts{Width: 100}).AlertBanner(a, 1, 3)
	if !strings.Contains(out, "[severe]") && !strings.Contains(strings.ToUpper(out), "SEVERE") {
		t.Fatalf("severity must be text (R-12a): %s", out)
	}
	if !strings.Contains(out, "01 / 03") {
		t.Fatalf("alert paging (mock: '01 / 88') missing: %s", out)
	}
}

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
	fg, bg := AlertBlockTone("Flash Flood Warning", "moderate")
	o := Opts{Width: 40}
	out := o.Block("hi \x1b[1mchip\x1b[0m tail", fg, bg)
	// UAT 17.2: tile bg hidden (default 49) while the red text tone stays.
	if bg != "49" || !strings.Contains(out, "38;5;196") {
		t.Fatalf("warning block tones (fg red, bg hidden): %q", out)
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
	afg, abg := AlertBlockTone("Heat Advisory", "minor")
	if afg != "38;5;220" || abg != "49" {
		t.Fatalf("advisory tone (fg yellow, bg hidden): %s %s", afg, abg)
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

func TestModuleInsetFollowsBGVisibility(t *testing.T) {
	// UAT 19.1: visible-bg modules inset 3 cols each side + padded blank
	// top/bottom; hidden-bg modules run flush with no padding lines.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	o := Opts{Width: 40}
	visible := o.Module([]string{"line one"}, "38;5;250", "48;2;48;48;48")
	if got := len(strings.Split(visible, "\n")); got != 3 {
		t.Fatalf("visible-bg module must add padding lines, got %d", got)
	}
	if !strings.Contains(visible, "m   line one") {
		t.Fatalf("visible-bg module must inset content 3 cols: %q", visible)
	}
	hidden := o.Module([]string{"line one"}, "38;5;250", "49")
	if got := len(strings.Split(hidden, "\n")); got != 1 {
		t.Fatalf("hidden-bg module must not pad, got %d lines", got)
	}
	if !strings.Contains(hidden, "mline one") {
		t.Fatalf("hidden-bg module must run flush: %q", hidden)
	}
	if ModuleHeight(7, "49") != 7 || ModuleHeight(7, "48;2;1;1;1") != 9 {
		t.Fatal("ModuleHeight must track visibility")
	}
	if BGVisible("49") || !BGVisible("48;2;48;48;48") {
		t.Fatal("BGVisible classification")
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
