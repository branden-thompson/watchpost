package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// broadcaster_uat_test.go — the HUM LEAD's UAT of 2026-09-10, one test per
// symptom. They are kept together because they share ONE root cause and the
// grouping is the record of that: `TruncateCells` counted an escape sequence's
// bytes as display cells, the console clamps every row of its frame through it,
// and the masthead's wordmark carries a truecolor escape per rune.

// THE FRAME IS NEVER CUT THROUGH AN ESCAPE, AND NEVER LOSES CONTENT THAT FITS.
//
//	"1A. 'WATCHPOS     ' — the rest of the title missing
//	 1B. Top of the box draw missing
//	 1C. Updated Missing
//	 1D. API only shows ✔9 and missing the rest
//	 1E. Chips … change color as the terminal window expands / shrinks
//	 1F. GAIN/VOL control missing — only showing the left arrow"
//
// Six symptoms, one measure. The masthead is the strictest case in the frame —
// it is the row with the most escapes per cell — so it is what this asserts.
func TestTheFrameSurvivesColour(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	for _, w := range []int{100, 120, 133, 150, 200} {
		b := NewBroadcaster()
		b.width, b.height, b.version = w, 50, "0.16.0"
		rows := strings.Split(b.View().Content, "\n")
		for i, r := range rows { // every row, not just the masthead's
			if got := render.Width(r); got != w {
				t.Fatalf("at %d cols, row %d measures %d cells", w, i, got)
			}
			if strings.Contains(render.StripSGRForTest(r), "\x1b") {
				t.Fatalf("at %d cols, row %d was cut through an escape: %q", w, i, r)
			}
		}
		head := render.StripSGRForTest(rows[0])
		for _, want := range []string{"WATCHPOST", "Broadcaster", "v0.16.0"} {
			if !strings.Contains(head, want) {
				t.Errorf("at %d cols the masthead lost %q: %q", w, want, head)
			}
		}
		if !strings.Contains(render.StripSGRForTest(rows[1]), "API:") {
			t.Errorf("at %d cols the masthead lost its API summary: %q", w, rows[1])
		}
		// THE GAIN CONTROL KEEPS BOTH ENDS. It showed only its left arrow.
		station := strings.Join(rows[4:9], "\n")
		if !strings.Contains(render.StripSGRForTest(station), "+") {
			t.Errorf("at %d cols the gain control lost its right end:\n%s", w, render.StripSGRForTest(station))
		}
	}
}

// EVERY KEY THE MASTHEAD NAMES IS A CHIP.
//
//	"Chips dont render their bkg … tells me something about coloring and tokens
//	 are broken in broadcaster ui"
//
// Nothing was broken. The console had TYPED its keys as text, so there was no
// chip to paint — while the station section beside it, which uses `KeyCap`,
// painted its chips correctly. That difference is what the report describes.
func TestTheMastheadsKeysAreChips(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	b := NewBroadcaster()
	b.width, b.height, b.version = 150, 50, "0.16.0"
	o := b.opts()
	row := b.controlRow(o)
	for _, key := range []string{"s", "a", "S", "ctrl+o", "?", "q"} {
		if !strings.Contains(row, o.KeyCap(key)) {
			t.Errorf("the masthead names %q without a chip: %q", key, row)
		}
	}
}

// THE REGIONS BREATHE.
//
//	"the sections of the line up are missing their blank row between the
//	 sections like the mocks"
//
// BETWEEN REGIONS, NEVER BETWEEN CARDS — the reference draws its cards flush
// inside a region, and a gap between every card would say each card is its own
// section, which is the opposite of what the rail is for.
func TestOneBlankRowSeparatesTheRegions(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	rows := strings.Split(b.View().Content, "\n")
	// BETWEEN THE CARDS, which is what "between the sections" means: the frame
	// pads its own tail with walled empty rows to fill the terminal, and those
	// are not gaps between anything.
	last := 0
	for i, r := range rows {
		if strings.Contains(r, "+---") {
			last = i
		}
	}
	gaps := 0
	for _, r := range rows[:last] {
		cells := []rune(r)
		if !strings.HasPrefix(r, "|   |") || len(cells) < b.width {
			continue
		}
		// The rail's own walls with an EMPTY lane beside them. Checked by column
		// rather than by trimming, because a card's blank interior row is also
		// nothing but walls and spaces and the two must not be confused.
		if strings.TrimSpace(string(cells[bcRailWidth:b.width-bcRightChrome+3])) == "" {
			gaps++
		}
	}
	if want := len(bcRegions) - 1; gaps != want {
		t.Errorf("%d blank rows between %d regions, want %d", gaps, len(bcRegions), want)
	}
	// AND THE CARDS INSIDE A REGION STAY FLUSH: the LINE UP's five slots draw
	// twenty rows with nothing between them.
	lane := newCardLane(b.cardBoxWidth(), b.opts().Glyphs())
	r := bcRegions[len(bcRegions)-1]
	body := b.slotRows(r, nil, lane)
	if got, want := len(body), (r.upto-r.from)*4; got != want {
		t.Errorf("the last region draws %d rows for %d slots, want %d", got, r.upto-r.from, want)
	}
}
