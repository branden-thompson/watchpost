package tty

// router_typing_test.go — an open text field owns every printable key.
//
// THE SWAP KEYS ARE BOUND TO CAPITAL `B` AND `O` as well as ctrl+b / ctrl+o, so
// a Router keymap switch that returns UNCONDITIONALLY — above the "a window on
// top owns the keys" check — loses those letters out of either new location
// field, and `O` SWAPS THE SURFACE MID-WORD:
//
//	typed "Oceanside" -> box "ceanside"  active=Observer
//	typed "Bonsall"   -> box "onsall"
//
// "Oceanside, CA" is the HUM LEAD's own station and "Bonsall, CA" is the
// hyper-local case D-130 was ruled for.  The UAT never caught it because
// "Lone Pine", "Rainbow" and "Vista" contain no capital B or O.
//
// D-58 ALREADY RULED THIS — "a window on top owns the keyboard" — and the
// guarded cases below it say so in as many words.  The two swap cases were
// simply written above the guard.

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func consoleTypingInto(t *testing.T, open func(Router) Router) Router {
	t.Helper()
	d := goldenDash(t, false)
	d.cfg.LocateInRadius = func(string) (snapshot.LocationRef, bool, bool, bool) {
		return snapshot.LocationRef{Label: "Oceanside, CA", Zip: "92057"}, true, true, true
	}
	return open(consoleWith(t, d))
}

// THE LOOKUP BOX, WITH THE STATION'S OWN NAME.
func TestTheConsoleLookupTakesEveryLetterOfTheStationsName(t *testing.T) {
	for _, q := range []string{"Oceanside", "Bonsall", "Vista", "BOOM"} {
		r := consoleTypingInto(t, func(r Router) Router { return pressAction(t, r, actLookup) })
		out := typeInto(t, r, q)
		if got := out.observer.addQuery; got != q {
			t.Errorf("typed %q, the box holds %q — a letter was eaten by a binding", q, got)
		}
		if out.active != SurfaceBroadcaster {
			t.Errorf("typing %q left the console for surface %v mid-word", q, out.active)
		}
	}
}

// AND THE REQUEST WINDOW'S LOCATION FIELD, which shares the defect.
func TestTheRequestWindowTakesEveryLetterOfTheStationsName(t *testing.T) {
	for _, q := range []string{"Oceanside", "Bonsall"} {
		r := consoleTypingInto(t, func(r Router) Router { return pressAction(t, r, actRequest) })
		if r.observer.modal != modalRequest {
			t.Fatalf("[r] did not open the request window; modal %v", r.observer.modal)
		}
		out := typeInto(t, r, q)
		if got := out.observer.request.query; got != q {
			t.Errorf("typed %q, the Location field holds %q", q, got)
		}
		if out.active != SurfaceBroadcaster {
			t.Errorf("typing %q left the console for surface %v", q, out.active)
		}
	}
}

// AND THE SWAP STILL WORKS WITH NO WINDOW OPEN — the fix must not take the
// binding away, only stop it reaching through an open field.
func TestTheSwapStillWorksWithNoWindowOpen(t *testing.T) {
	r := consoleWith(t, goldenDash(t, false))
	if r.active != SurfaceBroadcaster {
		t.Fatal("fixture must start on the console")
	}
	out := typeInto(t, r, "O")
	if out.active != SurfaceObserver {
		t.Error("capital O no longer swaps to Observer with nothing open")
	}
	back := typeInto(t, out, "B")
	if back.active != SurfaceBroadcaster {
		t.Error("capital B no longer swaps to the console with nothing open")
	}
	// AND ctrl+o / ctrl+b ARE UNAFFECTED EVEN WITH A WINDOW OPEN: they are not
	// printable, so no text field can want them.
	open := consoleTypingInto(t, func(r Router) Router { return pressAction(t, r, actLookup) })
	if got := pressAction(t, open, actSwapObserver); got.active != SurfaceObserver {
		t.Error("ctrl+o must still swap out of an open window")
	}
	_ = strings.TrimSpace("")
}
