package app

import (
	"testing"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/platform/category"
)

// TestAnEvacuationOrderSoundsThreeDualTones pins the audible half of #18.
// cast knows the ruling; this is what makes the speaker obey it.
func TestAnEvacuationOrderSoundsThreeDualTones(t *testing.T) {
	emg, ok := cast.ClassFor(category.Emergency)
	if !ok {
		t.Fatal("Emergency Orders has no tone class")
	}
	warn, _ := cast.ClassFor(category.Warnings)

	one, three := alertTonePCM(warn), alertTonePCM(emg)
	if len(one) == 0 {
		t.Fatal("a warning produced no tone — the instrument cannot fail")
	}
	if got, want := len(three), 3*len(one); got != want {
		t.Errorf("evacuation tone is %d bytes, want %d (3 × a warning's %d) — the "+
			"count IS the signal (#18)", got, want, len(one))
	}
}

// TestTheToneClassPrefersTheCategoryOnlyWhereWordsCannotCarryIt pins the wiring
// AND its blast radius.
//
// cast.Classify reads a product's own words, which is right for the severity
// family — warning, watch, advisory, statement, storm all say what they are.
// It is wrong for the civil-emergency family: "Evacuation Immediate" contains
// no "warning", "watch" or "advisory", so it falls through to Classify's loud
// default. That fall-through IS #18.
//
// The fix is narrow on purpose. Preferring the CATEGORY everywhere would
// regress advisories — globalfeed.LaneOf has no Advisory arm and defaults to
// Warnings, so a Small Craft Advisory would start sounding like a warning. The
// second half of this test is what stops that over-correction.
func TestTheToneClassPrefersTheCategoryOnlyWhereWordsCannotCarryIt(t *testing.T) {
	for _, tc := range []struct {
		product string
		want    cast.Class
		why     string
	}{
		{"Evacuation Immediate", mustClassFor(t, category.Emergency), "the words carry no severity; the category does (#18)"},
		{"Civil Emergency Message", mustClassFor(t, category.Disasters), "same family, same reason"},
		{"Tornado Warning", cast.Classify("Tornado Warning"), "the words are authoritative"},
		{"Small Craft Advisory", cast.Classify("Small Craft Advisory"), "MUST NOT regress to a warning"},
		{"Coastal Flood Statement", cast.Classify("Coastal Flood Statement"), "MUST NOT regress"},
		{"Nonexistent Product Type", cast.Classify("Nonexistent Product Type"), "unknown keeps the loud default"},
	} {
		e := globalfeed.Event{Class: globalfeed.ClassSevereWx, Type: tc.product}
		if got := toneClassOfEvent(e); got != tc.want {
			t.Errorf("%q sounds %v, want %v — %s", tc.product, got, tc.want, tc.why)
		}
	}
}

func mustClassFor(t *testing.T, c category.Category) cast.Class {
	t.Helper()
	cls, ok := cast.ClassFor(c)
	if !ok {
		t.Fatalf("category %v has no tone class", c)
	}
	return cls
}
