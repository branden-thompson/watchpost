package tty

// request_blocker_test.go — the chip must name the thing that is actually
// missing.
//
// FOUND BY RED TEAM AT BUILD EXIT, 0.16.0 (2026-09-15). D-130 replaced
// `requestState.ref` with `locate` and LEFT THE FIELD BEHIND. Nothing assigned
// it, so `blocker()` returned "Choose a location" unconditionally and three of
// its four cases were dead — an operator with a correctly-filled Location field
// and no reports chosen was told to fix the location.
//
// AND ITS TWO EXISTING TESTS COULD NOT FAIL: both asserted
// `blocker() == "Choose a location"`, which is true of a constant function, so
// both would have passed with the whole switch deleted. That is this project's
// own named anti-pattern, produced inside the change that introduced the bug.

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/report"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// reachableLocate is a settled, in-radius answer — a Location field the
// operator has filled correctly.
func reachableLocate(t *testing.T) locateState {
	t.Helper()
	return settledLocate(locateRequest, "Vista",
		snapshot.LocationRef{Label: "Vista, CA", Zip: "92084"}, true, true)
}

func TestTheBlockerNamesEachUnmetConditionInTurn(t *testing.T) {
	for _, tc := range []struct {
		name string
		st   requestState
		want string
	}{
		{"nothing typed", requestOpen(), "Choose a location"},
		{
			"typed, nothing found",
			func() requestState {
				st := requestOpen()
				st.locate = settledLocate(locateRequest, "zzzz", snapshot.LocationRef{}, false, false)
				return st
			}(),
			"Choose a location",
		},
		{
			"found but out of radius",
			func() requestState {
				st := requestOpen()
				st.locate = settledLocate(locateRequest, "Lone Pine, CA",
					snapshot.LocationRef{Label: "Lone Pine, CA"}, false, true)
				return st
			}(),
			"Location is outside the service radius",
		},
		{
			// A FRESH WINDOW ALREADY CARRIES EVERY REPORT AND SLOT 15
			// (requestOpen), so reaching this case means CLEARING the set —
			// the operator untoggling their way to none.
			"location fine, no reports chosen",
			func() requestState {
				st := requestOpen()
				st.locate = reachableLocate(t)
				var none report.Set
				st.chosen = none
				return st
			}(),
			"Choose at least one report",
		},
		{
			"location and reports fine, position cleared",
			func() requestState {
				st := requestOpen()
				st.locate = reachableLocate(t)
				st.slot, st.prioritize = "", false
				return st
			}(),
			"Choose a position",
		},
		{
			"everything answered",
			func() requestState {
				st := requestOpen()
				st.locate = reachableLocate(t)
				return st
			}(),
			"Schedule",
		},
	} {
		if got := tc.st.blocker(); got != tc.want {
			t.Errorf("%s: the chip says %q, want %q", tc.name, got, tc.want)
		}
	}
}

// AND THE SWITCH IS NOT A CONSTANT. Stated as its own assertion because that is
// precisely what the two superseded tests could not see: a function returning
// one string forever satisfies any test that only ever asks for that string.
func TestTheBlockerIsNotAConstantFunction(t *testing.T) {
	st := requestOpen()
	first := st.blocker()

	st.locate = reachableLocate(t)
	second := st.blocker()
	if second == first {
		t.Fatalf("filling the location changed nothing: blocker() is %q either way — "+
			"three of its four cases are unreachable", first)
	}
	var none report.Set
	st.chosen = none
	third := st.blocker()
	if third == second || third == first {
		t.Fatalf("clearing the reports changed nothing: %q -> %q -> %q", first, second, third)
	}
	// THREE DISTINCT ANSWERS FROM THREE DISTINCT STATES is the property. The
	// two superseded tests each asked for one string and got it, which a
	// constant function satisfies perfectly.
	if first == second || second == third || first == third {
		t.Errorf("blocker() does not discriminate: %q / %q / %q", first, second, third)
	}
}
