package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/report"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// requestDash is a dashboard with the request window open and a pool behind it.
func requestDash(t *testing.T, sent *int) Dashboard {
	t.Helper()
	vista := snapshot.LocationRef{Label: "Vista, CA", Zip: "92084", Lat: 33.2, Lon: -117.24}
	d, err := NewDashboard(Config{
		PoolLookup: func(q string) (snapshot.LocationRef, bool, bool) {
			switch q {
			case "vista, ca", "vista":
				return vista, true, true
			case "denver, co":
				return snapshot.LocationRef{Label: "Denver, CO"}, false, true // real, outside the radius
			}
			return snapshot.LocationRef{}, false, false
		},
		RequestCard: func(snapshot.LocationRef, report.Set, int) { *sent++ },
	})
	if err != nil {
		t.Fatal(err)
	}
	return d.openRequest()
}

// TestAnIncompleteRequestIsNotScheduledAndTheWindowStaysOpen.
//
// FR-3.3: "an action must never be shown as taken unless the schedule took it."
// A window that closed on an invalid form would be exactly that — the operator
// would believe they had scheduled a report, and nothing would have been sent.
//
// MUTANT mAZ3 CLOSED IT REGARDLESS AND SURVIVED: the window's own validity was
// asserted nowhere, on the one path where being wrong is silent.
func TestAnIncompleteRequestIsNotScheduledAndTheWindowStaysOpen(t *testing.T) {
	var sent int
	d := requestDash(t, &sent)

	// NO LOCATION YET. Everything else is filled by default — the whole report,
	// and a position — so the location is the only thing missing.
	m, _ := d.handleRequestKey(keyPress(t, "enter"))
	out := m.(Dashboard)
	if sent != 0 {
		t.Errorf("an incomplete request was scheduled %d times", sent)
	}
	if out.modal != modalRequest {
		t.Error("the window closed on a form it could not schedule; the operator would " +
			"believe the report was scheduled")
	}
	// AND THE CHIP SAYS WHICH THING IS MISSING rather than going quiet.
	if got := out.request.blocker(); got != "Choose a location" {
		t.Errorf("the chip says %q; it names the first unmet condition", got)
	}
}

// AND A COMPLETE ONE IS SENT ONCE, AND CLOSES.
func TestACompleteRequestIsScheduledAndTheWindowCloses(t *testing.T) {
	var sent int
	d := requestDash(t, &sent)
	for _, r := range "vista" {
		d = d.requestType(string(r))
	}
	// AND A POSITION, WHICH IS NOT DEFAULTED. The operator came here to schedule
	// something and must say WHERE — putting it at the front unless they choose
	// otherwise would make the most disruptive act the one they get by not
	// deciding.
	d.request.field = requestPosition
	d = d.requestToggle()
	if !d.request.valid() {
		t.Fatalf("the form is not valid after typing a pooled location: %s", d.request.blocker())
	}
	m, cmd := d.requestSchedule()
	if cmd != nil {
		cmd() // the send is a Cmd, as everything that talks back to the program is (D-79)
	}
	if sent != 1 {
		t.Errorf("a complete request was scheduled %d times, want 1", sent)
	}
	if out := m.(Dashboard); out.modal == modalRequest {
		t.Error("the window stayed open after the schedule was told")
	}
}

// AND A LOCATION OUTSIDE THE RADIUS IS NAMED, NOT REFUSED (ruling 2).
//
// HUM LEAD, 2026-09-14: it gets helper text — "Location not found in Pool" and
// "Observer supports location lookup outside Broadcast Radius" — because a place
// the station cannot broadcast about is a real place, not a typo.
func TestALocationOutsideTheRadiusIsNamedRatherThanRefused(t *testing.T) {
	var sent int
	d := requestDash(t, &sent)
	for _, r := range "denver, co" {
		d = d.requestType(string(r))
	}
	if d.request.ref == nil {
		t.Fatal("a real location outside the radius resolved to nothing; it is not a typo")
	}
	if !d.request.outside {
		t.Error("a location outside the service radius was accepted as broadcastable")
	}
	if d.request.valid() {
		t.Error("a request for a location the station cannot broadcast about is schedulable")
	}
	fact, aside := d.request.note()
	if fact == "" || aside == "" {
		t.Errorf("the window says %q / %q; it owes both the fact and the way out", fact, aside)
	}
}

// TestTheOutOfRadiusHelperWearsObserversCaveatTone.
//
// HUM LEAD, UAT 2026-09-14: "Location not found helper color can be orange/red
// (similar to the red used by the 'this is not your local station' color in
// Observer)."
//
// ASKED OF THE TOKEN OBSERVER USES, not of a colour. `NameWarning` is what
// detail.go tints "This is not your local station" with, and it says the same
// kind of thing — what you are looking at is not what you think it is. Borrowing
// it means the two cannot drift and one theme change moves both.
func TestTheOutOfRadiusHelperWearsObserversCaveatTone(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	var sent int
	d := requestDash(t, &sent)
	for _, r := range "denver, co" {
		d = d.requestType(string(r))
	}
	lines, _, _ := d.requestBody(d.opts())
	fact, aside := d.request.note()
	if fact == "" || aside == "" {
		t.Fatalf("the window has nothing to say about a location outside the radius: %q / %q", fact, aside)
	}

	// EACH LINE ON ITS OWN. The first version of this joined them and asked
	// whether the TINT appeared anywhere in the result — so removing it from the
	// fact was masked by the aside still carrying it, and mutant mBB1 survived.
	// Two lines, two assertions.
	want := render.Tok(render.NameWarning)
	seen := 0
	for _, l := range lines {
		// MATCHED ON WHAT `note` ACTUALLY SAYS, asked of it rather than retyped.
		// The first version looked for "not found in Pool", which is the
		// NOT-FOUND case's wording — the OUTSIDE case says "Outside the
		// station's service radius", so only one line ever matched and the test
		// failed for a reason that was about the test.
		plain := stripANSITest(l)
		if !strings.Contains(plain, fact) && !strings.Contains(plain, aside) {
			continue
		}
		seen++
		if !strings.Contains(l, want) {
			t.Errorf("this helper line is not tinted %q: %q", want, plain)
		}
	}
	if seen != 2 {
		t.Fatalf("the window drew %d helper lines for a location outside the radius, want 2 — "+
			"the fact and the way out", seen)
	}
}
