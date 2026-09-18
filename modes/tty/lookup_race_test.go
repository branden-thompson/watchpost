package tty

// lookup_race_test.go — the console's scope cannot be escaped by being quick.
//
// D-130's GATE IS THREE-STATE — a definite no is refused, "not yet known" is
// not — and the not-yet-known branch must not fall through to `cfg.Resolve`, the
// UNSCOPED geocoder. Falling through leaves the UAT defect D-129 was filed for
// reachable: type a location outside the service radius and press enter before
// the 300 ms pause elapses, and it opens. The chip is drawn AVAILABLE while
// unsettled, so the operator has no cue to wait.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// consoleLookupTyping is a scoped search box mid-type: text entered, no verdict
// yet — the window between the last keystroke and the answer.
func consoleLookupTyping(t *testing.T, resolved *int) Dashboard {
	t.Helper()
	d := goldenDash(t, false)
	d.surface, d.addMode, d.addQuery = SurfaceBroadcaster, "lookup", "Lone Pine, CA"
	d.cfg.LocateInRadius = func(string) (snapshot.LocationRef, bool, bool, bool) {
		return snapshot.LocationRef{Label: "Lone Pine, CA"}, false, true, true // real, out of radius
	}
	// THE TEMPTING ANSWER MUST BE AVAILABLE and refused, or the test passes for
	// the wrong reason: this is the hook the leak went through.
	d.cfg.Resolve = func(string) (snapshot.LocationRef, error) {
		*resolved++
		return snapshot.LocationRef{Label: "Lone Pine, CA"}, nil
	}
	m, _ := d.afterLookupEdit()
	return m.(Dashboard).open(modalAdd)
}

func TestEnterBeforeTheAnswerDoesNotEscapeTheScope(t *testing.T) {
	var resolved int
	d := consoleLookupTyping(t, &resolved)
	if d.addLocate.settled() {
		t.Fatal("fixture: the field must be UNSETTLED; that is the window being tested")
	}

	m, cmd := d.handleAddKey(enterKey())
	if cmd != nil {
		if msg := cmd(); msg != nil {
			m, _ = m.(Dashboard).Update(msg)
		}
	}
	if resolved != 0 {
		t.Errorf("enter reached the unscoped resolver %d time(s): the console's scope is "+
			"escapable by pressing enter before the debounce settles", resolved)
	}
	if out := m.(Dashboard); out.modal != modalAdd {
		t.Errorf("the window left the search box (modal %v) on a location the station cannot reach", out.modal)
	}
}

// AND ENTER IS NOT INERT — it ASKS. A key that did nothing while the field was
// thinking would be the dead control D-129 exists to prevent, so pressing it
// must bring the answer forward rather than wait for the timer.
func TestEnterBeforeTheAnswerAsksTheScopedHookAtOnce(t *testing.T) {
	var resolved int
	asked := 0
	d := consoleLookupTyping(t, &resolved)
	d.cfg.LocateInRadius = func(string) (snapshot.LocationRef, bool, bool, bool) {
		asked++
		return snapshot.LocationRef{Label: "Lone Pine, CA"}, false, true, true
	}

	_, cmd := d.handleAddKey(enterKey())
	if cmd == nil {
		t.Fatal("enter did nothing at all while the field was unsettled")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("enter produced no message")
	}
	if asked != 1 {
		t.Errorf("enter asked the scoped hook %d time(s), want 1", asked)
	}
}

// AND A REACHABLE LOCATION STILL OPENS when the answer arrives after an enter —
// the press must not be swallowed, or the operator has to press it twice.
func TestAnEnterHeldOverAReachableAnswerStillOpens(t *testing.T) {
	d := goldenDash(t, false)
	d.surface, d.addMode, d.addQuery = SurfaceBroadcaster, "lookup", "Vista"
	d.cfg.LocateInRadius = func(string) (snapshot.LocationRef, bool, bool, bool) {
		return snapshot.LocationRef{Label: "Vista, CA", Zip: "92084"}, true, true, true
	}
	m0, _ := d.afterLookupEdit()
	d = m0.(Dashboard).open(modalAdd)

	var m tea.Model = d
	_, cmd := m.(Dashboard).handleAddKey(enterKey())
	if cmd == nil {
		t.Fatal("enter asked nothing")
	}
	m, _ = m.(Dashboard).handleAddKey(enterKey())
	msg := cmd()
	m2, cmd2 := m.(Dashboard).Update(msg)
	if cmd2 != nil {
		if out := cmd2(); out != nil {
			m2, _ = m2.(Dashboard).Update(out)
		}
	}
	if out := m2.(Dashboard); out.modal == modalAdd {
		t.Error("a reachable location did not open after the answer landed: the enter was swallowed")
	}
}

// A CHECK THAT COULD NOT BE MADE IS NOT "NO SUCH PLACE" (D-151).
//
// A 5-second timeout, a DNS blip or a cancelled context must not return
// `found=false` — the value a genuine no-match returns — or the window tells the
// operator a real location does not exist AND disables the key that would have
// retried it. `locateInRadius` answers in FOUR states, not the three its shape
// suggests, and the fourth must never be reported as the second.
func TestALookupThatCouldNotBeMadeSaysSoAndStaysRetryable(t *testing.T) {
	d := goldenDash(t, false)
	d.surface, d.addMode, d.addQuery = SurfaceBroadcaster, "lookup", "Rainbow, CA"
	d.cfg.LocateInRadius = func(string) (snapshot.LocationRef, bool, bool, bool) {
		return snapshot.LocationRef{}, false, false, false // the question could not be put
	}
	m0, _ := d.afterLookupEdit()
	d = m0.(Dashboard).open(modalAdd)
	d.addLocate = d.addLocate.apply(locateVerdictMsg{
		field: locateLookup, seq: d.addLocate.gate.Seq(), query: "Rainbow, CA", asked: false,
	})

	fact, aside := d.addLocate.locateNote()
	if strings.Contains(fact, "not found") {
		t.Errorf("a failed check is reported as a missing place: %q", fact)
	}
	if fact == "" || aside == "" {
		t.Errorf("the operator is told nothing: %q / %q", fact, aside)
	}
	// AND ENTER MUST STILL WORK — it is the retry.
	if _, cmd := d.handleAddKey(enterKey()); cmd == nil {
		t.Error("enter is inert after a failed check: the operator has a real location they " +
			"cannot request and no way to ask again")
	}
}
