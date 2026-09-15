package tty

// router_lookup_scope_test.go — on the CONSOLE, a location the station cannot
// reach is not a location.
//
// HUM LEAD, UAT 2026-09-14:
//
//	1. <l> opens location modal  2. Type "Lone Pine, CA"  3. wait 5s  4. Press enter
//	EXPECTED: A. … should trigger "invalid location message"
//	          B. <enter> should be disabled for invalid location
//	ACTUAL:   i. No invalid location message
//	          ii. <enter> opens location details modal for Lone Pine, CA

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// poolOf wires a LocateInRadius over a fixed set, the way app's pool arm
// lookInPool answers: in the pool or nowhere.
func poolOf(refs ...snapshot.LocationRef) func(string) (snapshot.LocationRef, bool, bool) {
	return func(q string) (snapshot.LocationRef, bool, bool) {
		q = strings.ToLower(strings.TrimSpace(q))
		for _, r := range refs {
			if strings.HasPrefix(strings.ToLower(r.Label), q) || strings.HasPrefix(r.Zip, q) {
				return r, true, true
			}
		}
		return snapshot.LocationRef{}, false, false
	}
}

func consoleLookupWith(t *testing.T, pool ...snapshot.LocationRef) Router {
	t.Helper()
	d := goldenDash(t, false)
	d.cfg.LocateInRadius = poolOf(pool...)
	// A RESOLVER THAT WOULD SAY YES. The defect was that the console asked THIS
	// instead of the pool, so a fixture without it could pass for the wrong
	// reason — the test needs the tempting answer to be available and refused.
	d.cfg.Resolve = func(string) (snapshot.LocationRef, error) {
		return snapshot.LocationRef{Label: "Lone Pine, CA", Tag: "LONEPINE", Zip: "93545"}, nil
	}
	return openSearchOnTheConsole(t, d)
}

// settleLookup runs the debounce to its answer THROUGH THE ROUTER, which is
// also how the console's copy of D-128 gets measured: the pause and the verdict
// are Observer's, and a router that dropped them would leave the search box
// waiting for ever.
func settleLookup(t *testing.T, r Router) Router {
	t.Helper()
	seq := r.observer.addLocate.gate.Seq()
	m, cmd := r.Update(locatePauseMsg{field: locateLookup, seq: seq})
	out, ok := m.(Router)
	if !ok {
		t.Fatal("the router must stay the program's model")
	}
	if cmd == nil {
		t.Fatal("the pause asked nothing: either it never reached Observer, or the hook is unwired")
	}
	m2, _ := out.Update(cmd())
	out2, ok := m2.(Router)
	if !ok {
		t.Fatal("the router must stay the program's model")
	}
	return out2
}

// A. THE REFUSAL HAS TO BE READABLE, and before enter is pressed.
func TestAnUnreachableLocationIsRefusedOnTheConsole(t *testing.T) {
	r := consoleLookupWith(t, snapshot.LocationRef{Label: "Vista, CA", Zip: "92084"})
	r = settleLookup(t, typeInto(t, r, "Lone Pine, CA"))

	got := r.View().Content
	if !strings.Contains(got, "Location not found in Pool") {
		t.Errorf("the operator must be told the station cannot reach it; frame was:\n%s", got)
	}
}

// B. AND ENTER MUST BE INERT. A control drawn as available that does something
// the window just said it would not is worse than one drawn as unavailable.
func TestEnterIsRefusedForAnUnreachableLocationOnTheConsole(t *testing.T) {
	r := consoleLookupWith(t, snapshot.LocationRef{Label: "Vista, CA", Zip: "92084"})
	r = settleLookup(t, typeInto(t, r, "Lone Pine, CA"))

	m, cmd := r.Update(enterKey())
	out := m.(Router)
	if cmd != nil {
		t.Error("enter asked something on a location the window had already refused")
	}
	if out.observer.modal != modalAdd {
		t.Errorf("the window must stay open on a refusal; modal is %v", out.observer.modal)
	}
	// NOT "the frame must not say Lone Pine" — the SEARCH BOX echoes what was
	// typed, and it should. What must not happen is the details window opening
	// on a location the station cannot reach, which the modal check above pins;
	// and the refusal must survive the refused press rather than blinking away.
	if !strings.Contains(out.View().Content, "Location not found in Pool") {
		t.Error("the refusal vanished on the press it was refusing")
	}
}

// AND A LOCATION THE STATION CAN REACH STILL WORKS. Pinning only the refusal
// would leave a window that refuses everything looking correct.
func TestAPooledLocationIsStillAcceptedOnTheConsole(t *testing.T) {
	vista := snapshot.LocationRef{Label: "Vista, CA", Tag: "VISTA", Zip: "92084", Lat: 33.2, Lon: -117.2}
	r := consoleLookupWith(t, vista)
	r = settleLookup(t, typeInto(t, r, "Vista"))

	if got := r.View().Content; strings.Contains(got, "not found in Pool") {
		t.Errorf("a pooled location must not be refused; frame was:\n%s", got)
	}
	m, cmd := r.Update(enterKey())
	if cmd == nil {
		t.Fatal("enter on a pooled location did nothing")
	}
	out := m.(Router)
	if msg := cmd(); msg != nil {
		m2, _ := out.Update(msg)
		out = m2.(Router)
	}
	if out.observer.modal == modalAdd {
		t.Error("the search window must close once the location is accepted")
	}
}

// AND OBSERVER IS UNCHANGED (D-56: one key, one meaning PER SURFACE). The
// listener's lookup reaches anywhere; that is what Observer is for, and the
// console's own refusal points them at it.
func TestTheListenersLookupStillReachesOutsideTheRadius(t *testing.T) {
	d := goldenDash(t, false)
	d.cfg.LocateInRadius = poolOf(snapshot.LocationRef{Label: "Vista, CA", Zip: "92084"})
	asked := ""
	d.cfg.Resolve = func(q string) (snapshot.LocationRef, error) {
		asked = q
		return snapshot.LocationRef{Label: "Lone Pine, CA", Tag: "LONEPINE", Zip: "93545"}, nil
	}
	r := NewRouter(d)
	r.observer.width, r.observer.height = 150, 74
	if r.active != SurfaceObserver {
		t.Fatal("the fixture must start on Observer")
	}
	r = pressAction(t, r, actLookup)
	// NO SETTLE HERE, AND THAT IS THE ASSERTION BEHIND THE ASSERTION: Observer
	// arms no pause at all, because it has nothing to check what the listener
	// typed against. The debounce belongs to the console's scoping, not to the
	// search box.
	r = typeInto(t, r, "Lone Pine, CA")
	if r.observer.addLocate.gate.Seq() != 0 {
		t.Error("Observer armed a location check; its lookup reaches anywhere and owes no radius test")
	}
	if got := r.View().Content; strings.Contains(got, "not found in Pool") {
		t.Errorf("Observer must not be pool-scoped; frame was:\n%s", got)
	}
	if _, cmd := r.Update(enterKey()); cmd == nil {
		t.Fatal("enter on Observer must still ask the resolver")
	} else {
		cmd()
	}
	if asked != "Lone Pine, CA" {
		t.Errorf("Observer's lookup must reach the resolver; asked %q", asked)
	}
}
