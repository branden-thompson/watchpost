package category

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
)

// THE REGISTRY IS COMPLETE AND UNAMBIGUOUS.
//
// Everything else in the app is a view of this table, so a hole here is a hole
// everywhere — a category with no tint renders invisible, one with no label
// reads as a blank tab, and two sharing a rotation slot lose a lane from the
// band with nothing to say so.
func TestEveryCategoryIsFullyDescribed(t *testing.T) {
	seenShort, seenLabel := map[string]Category{}, map[string]Category{}
	seenTint := map[render.Token]Category{}
	rotations := map[int]Category{}
	for _, c := range All() {
		s := Of(c)
		if s.Bucket == "" || s.TabLabel == "" || s.TabShort == "" {
			t.Errorf("category %d is missing a name: %+v", c, s)
		}
		if s.Tint == "" {
			t.Errorf("%s has no row tint", s.TabLabel)
		}
		if other, dup := seenShort[s.TabShort]; dup {
			t.Errorf("%s and %s share the short form %q — the narrow tab bar could not tell them apart", Of(other).TabLabel, s.TabLabel, s.TabShort)
		}
		if other, dup := seenLabel[s.TabLabel]; dup {
			t.Errorf("%s and %s share the label %q", Of(other).TabLabel, s.TabLabel, s.TabLabel)
		}
		if other, dup := seenTint[s.Tint]; dup {
			t.Errorf("%s and %s share the row tint %s — the colour would stop saying which category a row is",
				Of(other).TabLabel, s.TabLabel, s.Tint)
		}
		seenShort[s.TabShort], seenLabel[s.TabLabel], seenTint[s.Tint] = c, c, c

		if s.BandTone == "" { // no lane: the band fields must be silent, not half-set
			if s.BandLabel != "" {
				t.Errorf("%s has no band colour but claims the band name %q", s.TabLabel, s.BandLabel)
			}
			continue
		}
		if s.BandLabel == "" {
			t.Errorf("%s has a band but no name for it; colour alone is not a channel (R-12a)", s.TabLabel)
		}
		if other, dup := rotations[s.Rotation]; dup {
			t.Errorf("%s and %s both sit at rotation %d — one of them never reaches the band", Of(other).TabLabel, s.TabLabel, s.Rotation)
		}
		rotations[s.Rotation] = c
	}
}

// The rotation is dense and starts at zero, or Lanes() silently drops a lane:
// it walks 0..n-1, so a gap ends the walk early.
func TestTheRotationHasNoGaps(t *testing.T) {
	lanes := Lanes()
	want := 0
	for _, c := range All() {
		if HasLane(c) {
			want++
		}
	}
	if len(lanes) != want {
		t.Fatalf("Lanes() returned %d of %d lanes — the rotation numbering has a gap or a repeat", len(lanes), want)
	}
	seen := map[Category]bool{}
	for _, c := range lanes {
		if seen[c] {
			t.Errorf("%s appears twice in the rotation", Of(c).TabLabel)
		}
		seen[c] = true
	}
}

// Emergency leads the tabs and the band; Forecasts is the one category with no
// lane. Both are HUM LEAD rulings (MVS-D-59, MVS-D-63) and both are the kind of
// thing a later edit reorders without meaning to.
func TestTheRulingsThatDecidedThisOrder(t *testing.T) {
	if All()[0] != Emergency {
		t.Errorf("Emergency Orders leads the tabs (MVS-D-63), found %s", Of(All()[0]).TabLabel)
	}
	if Lanes()[0] != Emergency {
		t.Errorf("Emergency Orders leads the rotation, found %s", Of(Lanes()[0]).TabLabel)
	}
	if HasLane(Forecasts) {
		t.Error("Forecasts has no ticker lane (MVS-D-59): the marquee is for what is happening")
	}
	for _, c := range []Category{Advisories, Statements, Forecasts} {
		if !Of(c).Watchlist {
			t.Errorf("%s reaches the app only through tracked locations and must say so on an empty tab", Of(c).TabLabel)
		}
	}
}

// A category outside the registry never indexes it.
func TestAnUnknownCategoryIsInert(t *testing.T) {
	for _, c := range []Category{None, Count, Category(99), Category(-99)} {
		if s := Of(c); s.TabLabel != "" || s.Tint != "" {
			t.Errorf("category %d must describe nothing, got %+v", c, s)
		}
		if HasLane(c) {
			t.Errorf("category %d must not claim a lane", c)
		}
	}
}

// THE THREE ORDERINGS ARE INDEPENDENTLY DECLARED, AND MUST STAY THAT WAY.
//
// A category is ordered three different ways for three different jobs: the tab
// bar reads straight down by seriousness, the marquee rotates leading with the
// urgent and the two the national feed fills, and the READ order decides what a
// listener hears first in a burst. They are not the same sequence and they are
// not meant to be.
//
// This exists because a future reader will notice three orderings on one type
// and tidy them into one — and doing that silently changes what is SPOKEN,
// which is the class of failure this package was built to end (F-21).
func TestTheThreeOrderingsAreIndependentlyDeclared(t *testing.T) {
	tabs, lanes, reads := All(), Lanes(), ReadOrder()
	if same(tabs, lanes) {
		t.Error("tab order and lane rotation are identical — one of them has been harmonised away")
	}
	if same(tabs, reads) {
		t.Error("tab order and read order are identical — one of them has been harmonised away")
	}
	if same(lanes, reads) {
		t.Error("lane rotation and read order are identical — one of them has been harmonised away")
	}
}

func same(a, b []Category) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// The read order is the one the HUM LEAD ratified (lineup-model.md): Emergency
// Orders lead, then Disasters above Warnings — which is NOT the tab order, and
// is deliberate — and Forecasts are never read as an alert at all.
func TestTheReadOrderIsTheRatifiedOne(t *testing.T) {
	want := []Category{Emergency, Disasters, Warnings, Watches, Advisories, Statements, Marine}
	got := ReadOrder()
	if !same(got, want) {
		t.Fatalf("read order = %v, want %v", names(got), names(want))
	}
	if Of(Forecasts).ReadRank != 0 {
		t.Error("Forecasts are never read AS AN ALERT (MVS-D-59) — they stay part of the location report")
	}
	// Disasters outranking Warnings is the ruling most likely to be "corrected"
	// by someone reading the tab bar, where Disasters sits lower.
	if Of(Disasters).ReadRank > Of(Warnings).ReadRank {
		t.Error("a disaster is read before a warning (R-2); the tab bar's order is a different job")
	}
}

func names(cs []Category) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, Of(c).TabLabel)
	}
	return out
}
