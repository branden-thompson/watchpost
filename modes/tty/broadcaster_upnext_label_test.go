package tty

// broadcaster_upnext_label_test.go — the box's own caption.
//
// HUM LEAD, 2026-09-14: "Let's make 'UP NEXT' in the up next box BOLD WHITE so
// it contrasts a bit more."

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestTheUpNextCaptionIsBoldWhite(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	b := broadcasterWithOneCard(t)
	var caption string
	for _, r := range b.upNextBox() {
		if strings.Contains(stripANSITest(r), bcUpNextLabel) {
			caption = r
			break
		}
	}
	if caption == "" {
		t.Fatalf("the box drew no %q row at all", bcUpNextLabel)
	}

	// BOLD IS THE ASSERTION THE GROUND CANNOT SATISFY. D-114 paints the whole
	// box — borders included — so EVERY row carries escape codes and "has
	// colour" is true of all of them. The ground never sets bold, so this is the
	// caption's own and nothing else's.
	if !strings.Contains(caption, "\x1b[1m") {
		t.Errorf("the caption is not bold; row was:\n  %q", caption)
	}
	if want := render.Tok(render.TextBright); !strings.Contains(caption, want) {
		t.Errorf("the caption does not carry the bright-text token %q; row was:\n  %q", want, caption)
	}
}

// AND THE CELL IS STILL EXACTLY AS WIDE AS IT WAS. Styling is escape codes, and
// escape codes are not cells — a caption padded after it was tinted would push
// the card's whole column right by the width of the codes.
func TestTheStyledCaptionDoesNotWidenTheBox(t *testing.T) {
	b := broadcasterWithOneCard(t)
	rows := b.upNextBox()
	if len(rows) == 0 {
		t.Fatal("the box drew nothing")
	}
	want := render.Width(stripANSITest(rows[0]))
	for i, r := range rows {
		if got := render.Width(stripANSITest(r)); got != want {
			t.Fatalf("row %d is %d cells wide against the box's %d: the caption's styling is being counted as content", i, got, want)
		}
	}
}
