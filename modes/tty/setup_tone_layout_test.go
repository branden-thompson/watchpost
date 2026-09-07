package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
)

// ONE CLASS PER LINE, at every width.
//
// The classes were laid two abreast — first padded to a common width, then with
// each sub-column sized to its own labels. Both are gone: six rows of one thing
// each is a list, and the eye runs down a single column of state words instead
// of scanning across a gap and back.

func TestToneDrawsOneClassPerLineAtEveryWidth(t *testing.T) {
	for _, size := range []struct {
		name string
		w, h int
	}{{"wide", 133, 44}, {"narrow", 80, 24}} {
		d := setupGolden(t, size.w, size.h, false, rowClassDisaster)
		lines := d.toneLines(d.opts())
		if len(lines) != len(classRowOrder()) {
			t.Errorf("%s: %d lines for %d classes — one class per line", size.name, len(lines), len(classRowOrder()))
		}
		// One toggle per line. Two would mean a class had been paired onto a
		// neighbour's row, which is the arrangement this replaced.
		for i, l := range lines {
			if got := strings.Count(render.Plain(l), "Enabled"); got != 1 {
				t.Errorf("%s: line %d carries %d state words, want 1", size.name, i, got)
			}
		}
	}
}

// The class column is MEASURED from the labels the app supplied, not fixed. The
// labels come from the registry, and a constant would silently truncate or
// over-pad the next class added.
func TestToneClassColumnFitsTheLongestLabel(t *testing.T) {
	d := setupGolden(t, 133, 44, false, rowClassDisaster)
	labels := d.toneClassLabels()
	got := toneLabelW(classRowOrder(), labels)
	if min := render.Width("Special Statements") + toneLabelAir; got < min {
		t.Errorf("class column %d cells, needs at least %d — the widest label plus air", got, min)
	}
	// And never narrower than a correspondent row's, so the two groups begin
	// their controls in the same cell.
	if got != max(rowControlW, render.Width("Special Statements")+toneLabelAir) {
		t.Errorf("class column %d cells — the wider of the labels and the correspondents' column (%d)", got, rowControlW)
	}
	// Every toggle starts at the same column, which is the point of measuring it.
	var col = -1
	for i, l := range d.toneLines(d.opts()) {
		plain := render.Plain(l)
		at := render.Width(plain[:strings.Index(plain, "Enabled")])
		if col == -1 {
			col = at
		}
		if at != col {
			t.Errorf("line %d starts its toggle at %d, the first at %d — the column does not line up", i, at, col)
		}
	}
}

// The scroll reads the line the row is actually drawn on. A focus line computed
// against a different arrangement puts the mark just off the edge, which reads
// as a dead keyboard — the listener presses a key and sees nothing.
func TestToneFocusLineFollowsTheDrawnLayout(t *testing.T) {
	for _, id := range classRowOrder() {
		d := setupGolden(t, 133, 44, false, id)
		lines := d.toneLines(d.opts())
		at := toneLineOf(classRowOrder(), id)
		if at < 0 || at >= len(lines) {
			t.Fatalf("focus line %d for %v is outside the %d lines drawn", at, id, len(lines))
		}
		for i, l := range lines {
			if marked := strings.Contains(l, "›"); marked != (i == at) {
				t.Errorf("focus on %v — line %d %s the mark", id, i,
					map[bool]string{true: "should not carry", false: "should carry"}[marked])
			}
		}
	}
}
