package cast

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/category"
)

// emergency_tone_test.go — #18, and the mapping that stops it recurring.
//
// An Evacuation Immediate sounded the WARNING tone: toneRank 5, byte-identical
// to a Severe Thunderstorm Warning. C-2 taught the feed, the window and the
// read ladder that an emergency order is its own thing; #15 taught the marquee
// at 0.14.2; the tone was never told, because nothing connected the category
// set to the tone set.

// TestAnEvacuationOrderSoundsThreeTimes pins the HUM LEAD ruling: a warning is
// one dual tone, an evacuation order is three. The count carries the urgency
// before a word is spoken, which is the whole job of a tone for a listener who
// is not looking at a screen.
func TestAnEvacuationOrderSoundsThreeTimes(t *testing.T) {
	emg, ok := ClassFor(category.Emergency)
	if !ok {
		t.Fatal("Emergency Orders has no tone class")
	}
	if got := ToneName(emg); got != PresetDualTone {
		t.Errorf("emergency preset: got %q want %q — the ruling reuses the dual tone", got, PresetDualTone)
	}
	if got := ToneRepeats(emg); got != 3 {
		t.Errorf("emergency repeats: got %d want 3", got)
	}
	warn, _ := ClassFor(category.Warnings)
	if got := ToneRepeats(warn); got != 1 {
		t.Errorf("warning repeats: got %d want 1 — the distinction IS the count", got)
	}
	if ToneName(warn) != ToneName(emg) {
		t.Error("the ruling reuses one preset; a different sound is a different ruling")
	}
}

// TestEveryCategoryThatReachesAReadHasATone is the anti-recurrence guard, and
// it is the point. #18 happened because category.Emergency was added and
// cast.Class was never told. Every member is carried or declared with a reason;
// a member in neither FAILS. Never skipped.
func TestEveryCategoryThatReachesAReadHasATone(t *testing.T) {
	// unreachable is a category no read can carry, with the reason written
	// down. It is not a silencer: a member here that starts mapping is still a
	// change somebody has to look at.
	unreachable := map[category.Category]string{
		category.Forecasts: "a forecast is not an alert; it never reaches the read ladder",
	}
	all := category.All()
	if len(all) == 0 {
		t.Fatal("no categories to check — the instrument cannot fail, so it proves nothing")
	}
	for _, c := range all {
		cls, ok := ClassFor(c)
		if !ok {
			if _, declared := unreachable[c]; declared {
				continue
			}
			t.Errorf("category %v has no tone class and no written reason: give it one, "+
				"or declare why no read can carry it — this is how #18 happened", c)
			continue
		}
		if _, declared := unreachable[c]; declared {
			t.Errorf("category %v is declared unreachable but maps to %v — delete the row "+
				"or the next category to regress here will look expected", c, cls)
		}
		if !cls.valid() {
			t.Errorf("category %v maps to an out-of-range class %v", c, cls)
		}
	}
}

// TestEveryClassIsListedOrDeclared is the guard that adding ClassEmergency
// proved was missing.
//
// FOUND BY MAKING THE MISTAKE. Classes() and ClassKeys() are hand-written lists
// that stood at six while numClasses moved to seven, and modes/tty's "parity
// test" compares its own hand list against itself (setup_tone_layout_test.go
// asserts len(lines) == len(classRowOrder())), so nothing anywhere noticed.
// Three stale six-lists and a tautological guard, from one enum member.
func TestEveryClassIsListedOrDeclared(t *testing.T) {
	// notListed is a class deliberately absent from the listener-facing set,
	// with the reason written down.
	notListed := map[Class]string{
		ClassEmergency: "an evacuation order's tone is not mutable: Classes() feeds the " +
			"Settings mute list, and a listener must not be able to silence " +
			"leave-now (#18). Absent by decision, not by omission — HUM LEAD 2026-09-07, " +
			"provisional: revisit at UAT if a listener wants it",
	}
	listed := map[Class]bool{}
	for _, c := range Classes() {
		listed[c] = true
	}
	for c := Class(0); c < numClasses; c++ {
		reason, declared := notListed[c]
		switch {
		case listed[c] && declared:
			t.Errorf("%v is both listed and declared absent (%q) — delete the row, or the "+
				"next class to regress here will look expected", c, reason)
		case !listed[c] && !declared:
			t.Errorf("%v is in neither Classes() nor the declared-absent set: add it, or "+
				"write down why a listener never sees it", c)
		}
	}
	if len(Classes()) != len(ClassKeys()) {
		t.Errorf("Classes() has %d and ClassKeys() has %d — two hand lists of one set",
			len(Classes()), len(ClassKeys()))
	}
}
