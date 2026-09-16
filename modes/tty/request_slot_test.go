package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/report"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// THE SLOT THE OPERATOR TYPES IS NOT THE INDEX THE SCHEDULE TAKES (D-119/D-156).
//
// THE DEFECT, SAID PLAINLY: `position()` returned the typed number verbatim, and
// `Requested.To` is documented as "the same number `Moved.To` carries" — the one
// the card window's move path has subtracted `liveOffset` from since D-119. Two
// operator paths into one field, disagreeing by one on STANDBY, which is the
// console's normal state. The card landed a row below the slot that was asked
// for, and the window's own confirmation named the slot.
//
// IT IS DRIVEN THROUGH THE ROUTER WITH REAL KEYS (P-1). Calling
// `requestSchedule` directly would pass with the wiring absent: `Dashboard`'s
// offset field reads zero unset, which is exactly the buggy answer. The seam the
// KEY takes is the only one that proves the Router carries the offset across.
func TestARequestedCardLandsInTheSlotTheOperatorTyped(t *testing.T) {
	for _, tc := range []struct {
		name  string
		power lineup.Power
		typed string
		want  int // the running-order index the schedule must receive
	}{
		// LIVE IS EMPTY ON STANDBY and the line-up is drawn from UP NEXT down,
		// so slot 4 is the third card in the running order.
		{"standby: the line-up sits one below LIVE", lineup.OffAir, "4", 3},
		// ON AIR the LIVE slot is occupied and the two coincide.
		{"on air: slot and index coincide", lineup.Running, "4", 4},
		// AND THE LOWEST SLOT THE WINDOW ACCEPTS IS `bcScheduledFrom` — the two
		// above it are the cards the operator READS from (D-68), not addresses
		// they can file a request into.
		{"standby: the lowest typable slot", lineup.OffAir, "2", 1},
		{"on air: the lowest typable slot", lineup.Running, "2", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got int
			var sent bool
			vista := snapshot.LocationRef{Label: "Vista, CA"}
			d, err := NewDashboard(Config{
				LocateInRadius: func(string) (snapshot.LocationRef, bool, bool, bool) {
					return vista, true, true, true
				},
				RequestCard: func(_ snapshot.LocationRef, _ report.Set, at int) {
					got, sent = at, true
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			r := NewRouter(d)
			r.active = SurfaceBroadcaster
			r.broadcaster.power = tc.power
			r.observer = r.observer.openRequest()

			// THE FORM, FILLED THE WAY THE OPERATOR FILLS IT.
			st := r.observer.request
			st.locate.ref, st.locate.found, st.locate.within = &vista, true, true
			st.locate.asked = true
			st.slot, st.prioritize = tc.typed, false
			r.observer.request = st

			m, cmd := r.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
			if rr, ok := m.(Router); ok {
				r = rr
			}
			if cmd != nil {
				cmd()
			}
			if !sent {
				t.Fatalf("the request was never scheduled; the form was valid")
			}
			if got != tc.want {
				t.Errorf("slot %q on %v reached the schedule as index %d, want %d",
					tc.typed, tc.power, got, tc.want)
			}
		})
	}
}

// THE REQUEST WINDOW RETRIES A LOOKUP THAT COULD NOT BE ASKED (D-157).
//
// THE FOURTH STATE, TAUGHT TO ONE OF TWO TWINS. `[l]` and the Line-Up Request
// window share `locateState`, and `[l]` handled all four answers while this one
// handled two. On "could not ask" — a timeout, which is a verdict about the
// LOOKUP and not about the place — enter did nothing at all.
//
// THREE SENTENCES IN ONE WINDOW, AND THEY DISAGREED. The helper line, shared and
// therefore right, read "The lookup did not answer; press enter to try again".
// The chip read "Choose a location", about a place that was never checked. And
// the key itself did neither. D-151's dead control, reached from a third side.
func TestTheRequestWindowRetriesALookupThatCouldNotBeAsked(t *testing.T) {
	var asks int
	vista := snapshot.LocationRef{Label: "Vista, CA"}
	d, err := NewDashboard(Config{
		// THE HOOK THAT CANNOT ANSWER: `asked` false is "the question could not
		// be put", which is not the same as "no such place".
		LocateInRadius: func(string) (snapshot.LocationRef, bool, bool, bool) {
			asks++
			return snapshot.LocationRef{}, false, false, false
		},
		RequestCard: func(snapshot.LocationRef, report.Set, int) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	r := NewRouter(d)
	r.active = SurfaceBroadcaster
	r.observer = r.observer.openRequest()

	// THE OPERATOR HAS TYPED A REAL PLACE AND THE LOOKUP TIMED OUT.
	st := r.observer.request
	st.query = "rainbow, ca"
	st.locate.query = "rainbow, ca" // what `edit` records as the operator types
	st.locate = st.locate.apply(locateVerdictMsg{field: locateRequest, seq: st.locate.gate.Seq(),
		query: "rainbow, ca", ref: vista, within: false, found: false, asked: false})
	r.observer.request = st

	if !r.observer.request.locate.couldNotAsk() {
		t.Fatal("the fixture did not reach the state under test")
	}
	// THE CHIP AGREES WITH THE HELPER LINE.
	if got := r.observer.request.blocker(); got != "Try again" {
		t.Errorf("the chip reads %q; the helper under the same field says to press enter to try again", got)
	}
	if _, help := r.observer.request.note(); !strings.Contains(help, "try again") {
		t.Errorf("the helper line no longer offers the retry the chip promises: %q", help)
	}

	// AND THE KEY DOES WHAT BOTH OF THEM SAY.
	before := asks
	m, cmd := r.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if rr, ok := m.(Router); ok {
		r = rr
	}
	if cmd == nil {
		t.Fatal("enter did nothing on a lookup that could not be asked")
	}
	cmd()
	if asks == before {
		t.Error("enter produced a command that did not put the question again")
	}
}

// AND THE SLOT IS STILL RIGHT WHEN THE WINDOW IS SUBMITTED FROM OBSERVER (D-160).
//
// THE PATH D-156 DID NOT WIRE, and it is reachable by a route the console
// deliberately supports. `ctrl+o` swaps out of an open window on purpose — the
// escape hatch an operator needs — so the Line-Up Request window opened with `r`
// on the console outlives the surface that opened it. With Observer active,
// `update` calls `r.observer.Update` DIRECTLY; `throughToObserver`, the only
// place that carried the offset, is never reached.
//
// SO THE OFFSET READ ZERO, which is the RUNNING station's answer, and on STANDBY
// it is wrong by one: the card lands a row below the slot the operator typed,
// with the window's own confirmation naming the slot they asked for. That is
// D-119 verbatim, surviving inside its own fix.
//
// IT DRIVES THE ROUTER, like its sibling above, because a direct call to
// `requestSchedule` passes with the wiring absent: the offset field reads zero
// unset, which is exactly the buggy answer.
func TestARequestSubmittedFromObserverStillLandsInTheTypedSlot(t *testing.T) {
	var got int
	var sent bool
	vista := snapshot.LocationRef{Label: "Vista, CA"}
	d, err := NewDashboard(Config{
		LocateInRadius: func(string) (snapshot.LocationRef, bool, bool, bool) {
			return vista, true, true, true
		},
		RequestCard: func(_ snapshot.LocationRef, _ report.Set, at int) { got, sent = at, true },
	})
	if err != nil {
		t.Fatal(err)
	}
	r := NewRouter(d)
	// THE OPERATOR OPENS IT ON THE CONSOLE, at STANDBY.
	r.active = SurfaceBroadcaster
	r.broadcaster.power = lineup.OffAir
	r.observer = r.observer.openRequest()
	// AND THEN SWAPS TO OBSERVER WITH IT OPEN, which `ctrl+o` is meant to allow.
	r.active = SurfaceObserver

	st := r.observer.request
	st.locate.ref, st.locate.found, st.locate.within = &vista, true, true
	st.locate.asked = true
	st.slot, st.prioritize = "4", false
	r.observer.request = st

	m, cmd := r.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if rr, ok := m.(Router); ok {
		r = rr
	}
	if cmd != nil {
		cmd()
	}
	if !sent {
		t.Fatal("the request was never scheduled; the form was valid")
	}
	// STANDBY: LIVE is empty and the line-up is drawn from UP NEXT down, so slot
	// 4 is the third card in the running order.
	if got != 3 {
		t.Errorf("slot 4 submitted from Observer reached the schedule as index %d, want 3 — "+
			"the console's live offset did not cross to the window", got)
	}
}
