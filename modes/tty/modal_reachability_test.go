package tty

// modal_reachability_test.go — FR-5, the set-level property.
//
// THE PROPERTY: at 80x24, the app's documented floor, every line a window draws
// can be brought on screen with the keys that window offers.
//
// WHY IT IS A SET GATE. The same defect has now been found three times in three
// windows, each time by someone opening that one window: the relay-fault
// window's ways out (2026-09-05), the ctrl+d window's scenario list, and the
// ctrl+d window's whole message in a release build (both 2026-09-07). Each fix
// was correct and none of them generalised, because nothing measured the other
// windows. A window is not a place to look; it is a member of a set.
//
// IT IS A BASELINE AND A RATCHET, not a clean bill of health. Four windows carry
// a non-zero count that has NOT been diagnosed, and they are written down here
// with what is known about each rather than declared fine. The ratchet is both
// ways: a count that rises fails, and a count that falls fails too, so a window
// that gets better takes its baseline down with it and cannot silently regress
// back to the old number.
//
// WHAT "REACHABLE" MEANS HERE. The body is asked of the renderer's OWN owners
// (focusBody + wrapModal for the pinned-footer windows, modalLines for the
// scrolling ones), so the reference and the frame cannot wrap differently — the
// first attempt at this measurement compared an 80x24 render against an 80x200
// one and reported every line unreachable, because a window that overflows
// wraps three columns narrower than one that does not.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/closedset"
)

// unreachableAtTheFloor is what a listener at 80x24 cannot bring on screen.
//
// It drives the REAL key path — Update, the keys the window advertises — and
// exhausts it: a probe that pressed Down eight times against a 39-line body
// reported 23 unreachable lines that were simply not scrolled to yet.
func unreachableAtTheFloor(d Dashboard) []string {
	want, seen := map[string]bool{}, map[string]bool{}
	var model tea.Model = d
	// Twice the body, plus a turn: enough to walk a list that wraps around to
	// the top and then some, whatever the window's own bound is.
	for i := 0; i < 2*len(d.modalLines())+4; i++ {
		cur := model.(Dashboard)
		for _, l := range cur.modalLines() {
			if text := modalTextOf(l); text != "" {
				want[text] = true
			}
		}
		for _, l := range strings.Split(cur.renderModal(cur.opts()), "\n") {
			seen[modalTextOf(l)] = true
		}
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	var out []string
	for l := range want {
		if !seen[l] {
			out = append(out, l)
		}
	}
	return out
}

// reachabilityBaseline is what each window cannot reach today, and why the
// number is not zero. A new window MUST appear here — closedset below fails on
// one that does not — so it is measured rather than assumed.
var reachabilityBaseline = map[modal]int{
	modalHelp:       0,
	modalDetails:    0,
	modalAdd:        0,
	modalRemove:     0,
	modalAlerts:     0,
	modalStatus:     0,
	modalAbout:      0,
	modalSetup:      2,
	modalSevere:     0,
	modalRelayFault: 1,
	modalDebug:      0,
}

// reachabilityNote is what is KNOWN about a non-zero baseline, so the number is
// a recorded question rather than a silent allowance.
//
// ALL THREE ARE ONE MECHANISM, which is why they are written down together: in a
// window whose scroll FOLLOWS THE FOCUS, the body above the first focusable row
// is unreachable whenever that row sits more than a window's height down. The
// scroll can go no further up without taking the focused row off screen, and
// nothing else moves it. What is lost is the window's own identity — the
// relay-fault window's "*** ERROR ***", the group a setting belongs to, the
// ctrl+d window's title and its FABRICATED caveat — at 80x24 and not above it.
//
// THE FIX IS A HUM LEAD RULING, not a defect to close quietly: pin the head as
// chrome the way the footer is pinned, add a key that scrolls the body free of
// the focus, or accept it at the floor. FR-5 follow-up.
var reachabilityNote = map[modal]string{
	modalSetup:      "the two group headers: WATCHPOST RADIO - CORRESPONDENTS and WATCHPOST UI",
	modalRelayFault: "*** ERROR ***, the window's own head; its three ways out are reachable and its own test proves it",
}

func TestEveryLineOfEveryWindowIsReachableAtTheFloor(t *testing.T) {
	// EVERY WINDOW IN THE ENUM, derived. A hand-written list of windows would
	// miss the next one added in exactly the way the three fixed defects were
	// missed.
	var modals []modal
	for m := modalHelp; m < numModals; m++ {
		modals = append(modals, m)
	}
	closedset.EachMember(t, "modal reachability baseline", modals, nil, func(m modal) bool {
		_, ok := reachabilityBaseline[m]
		return ok
	})

	for _, m := range modals {
		t.Run(modalName(m), func(t *testing.T) {
			d := fixtureFor(t, m)
			d.width, d.height = 80, 24 // the documented floor
			if m == modalSevere {
				// THE RECORD, NOT THE TABLE. severeDetailLines is the body that
				// scrolls; the table windows itself and the keys that walk it are
				// not ↑↓ over a body. Measured on the table, this compares the
				// frame against a body it is not drawing — so the window under
				// test here is the record, and the table's own reachability is
				// an FR-5 follow-up rather than a number nobody can read.
				d.severeDetail = true
			}
			if d.renderModal(d.opts()) == "" {
				t.Fatalf("%s draws nothing at this fixture, so this measures nothing", modalName(m))
			}
			got, want := unreachableAtTheFloor(d), reachabilityBaseline[m]
			switch {
			case len(got) > want:
				t.Errorf("%s: %d lines cannot be reached at 80x24, baseline %d — a line the "+
					"keyboard cannot bring on screen is not in the window:\n  %s",
					modalName(m), len(got), want, strings.Join(got, "\n  "))
			case len(got) < want:
				t.Errorf("%s: %d unreachable, baseline %d — the window improved; take the "+
					"baseline down to %d so it cannot drift back up (%s)",
					modalName(m), len(got), want, len(got), reachabilityNote[m])
			}
		})
	}
}
