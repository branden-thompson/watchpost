package report

import (
	"strings"
	"testing"
)

// TestDescribeIsTheRunningOrdersRule.
//
// HUM LEAD, 2026-09-14: "We show both in the modal, only the Label in the
// Scheduled-Line up table in the base Broadcaster UI as comma delimited list:
// 02. NWS, FIRE, QUAKE".  Everything reads "Location Report, Full".
func TestDescribeIsTheRunningOrdersRule(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  Set
		want string
	}{
		{"nothing chosen", 0, ""},
		{"one kind is a one-element list, not a special case", Set(0).Add(Fire), "FIRE"},
		{"three", Set(0).Add(NWS).Add(Fire).Add(Seismic), "NWS, FIRE, QUAKE"},
		{"everything", Everything(), FullLabel},
	} {
		if got := tc.set.Describe(); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

// AND THE ORDER IS THE REGISTRY'S, NOT THE ORDER THEY WERE CHOSEN IN.
//
// Two operators picking the same three reports in different orders must produce
// the same card, or the running order reads differently for one decision and the
// string cannot be compared, memoised or golden-tested.
func TestDescribeIsStableWhateverOrderTheyWereChosen(t *testing.T) {
	forwards := Set(0).Add(NWS).Add(Fire).Add(Seismic)
	backwards := Set(0).Add(Seismic).Add(Fire).Add(NWS)
	if forwards != backwards {
		t.Errorf("the same three kinds made two different sets: %v vs %v", forwards, backwards)
	}
	if a, b := forwards.Describe(), backwards.Describe(); a != b {
		t.Errorf("the same three kinds read %q one way and %q the other", a, b)
	}
}

// TestAFifthKindCostsOneRow.
//
// THIS TEST IS THE DESIGN CLAIM, and it is the reason this package exists at all.
//
// HUM LEAD, 2026-09-14: "we WILL have more report types in the future, and
// ensuring Broadcaster is flexible enough for that list to grow and change
// WITHOUT having to completely re-architect the code and flow every time is
// critical.  It's worth the careful thinking and cost now vs. trying to bolt it
// on later when additional features may code us into a corner."
//
// So the claim is not "four kinds work".  It is that a FIFTH costs ONE ROW — and
// a claim nobody checks is a promise.  This walks every kind the registry knows
// through everything that consumes one, so the day a row is added the only way
// for this test to fail is for something to have hard-coded four.
func TestAFifthKindCostsOneRow(t *testing.T) {
	kinds := All()
	if len(kinds) != int(numKinds) {
		t.Fatalf("All() returns %d kinds and the registry holds %d", len(kinds), numKinds)
	}

	// EVERY KIND IS DESCRIBED. A row added with no Spec would draw an empty
	// label into the middle of a comma list and read as a rendering fault.
	for _, k := range kinds {
		spec := Of(k)
		if strings.TrimSpace(spec.FullName) == "" {
			t.Errorf("kind %d has no FullName; the modal would offer a blank line to choose", k)
		}
		if strings.TrimSpace(spec.Label) == "" {
			t.Errorf("kind %d has no Label; the running order would draw an empty cell in a comma list", k)
		}
	}

	// EVERY KIND ROUND-TRIPS THROUGH THE SET, so a new one is selectable,
	// deselectable and visible without a line of new code.
	for _, k := range kinds {
		s := Set(0).Add(k)
		if !s.Has(k) {
			t.Errorf("kind %d cannot be added to a set", k)
		}
		if s.Describe() != Of(k).Label {
			t.Errorf("kind %d alone describes as %q, want its label %q", k, s.Describe(), Of(k).Label)
		}
		if s.Toggle(k).Has(k) {
			t.Errorf("kind %d cannot be toggled off", k)
		}
	}

	// AND "EVERYTHING" MEANS EVERY KIND THE REGISTRY HOLDS, not four. This is
	// the assertion that fails if a fifth row is added and something still
	// believes the set is complete at four.
	if got := len(Everything().Kinds()); got != int(numKinds) {
		t.Errorf("Everything() carries %d kinds, the registry holds %d", got, numKinds)
	}
	if !Everything().Full() {
		t.Error("the set of every kind does not report itself as full")
	}
	if Everything().Describe() != FullLabel {
		t.Errorf("every kind describes as %q, want %q", Everything().Describe(), FullLabel)
	}

	// THE LABELS ARE DISTINCT, because the running order's whole job is to let
	// the operator tell them apart at a glance.
	seen := map[string]Kind{}
	for _, k := range kinds {
		l := Of(k).Label
		if prev, dup := seen[l]; dup {
			t.Errorf("kinds %d and %d both label themselves %q", prev, k, l)
		}
		seen[l] = k
	}
}

// AND AN UNKNOWN KIND SAYS NOTHING RATHER THAN STOPPING.
//
// A set written by a later build names a kind this one does not have. The honest
// answer is to describe what is known and stay quiet about the rest — a panic
// here would take the console down over a report type someone else added.
func TestAKindThisBuildDoesNotKnowIsSilent(t *testing.T) {
	future := Kind(numKinds + 3)
	if got := Of(future); got != (Spec{}) {
		t.Errorf("an unknown kind described itself as %+v", got)
	}
	if Set(0).Add(future) != 0 {
		t.Error("an unknown kind was added to a set")
	}
	if Set(0).Has(future) {
		t.Error("an empty set claims to hold an unknown kind")
	}
}
