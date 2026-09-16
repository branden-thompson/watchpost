package tty

import (
	tea "charm.land/bubbletea/v2"

	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
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
		LocateInRadius: func(q string) (snapshot.LocationRef, bool, bool, bool) {
			switch q {
			case "vista, ca", "vista":
				return vista, true, true, true
			case "denver, co":
				return snapshot.LocationRef{Label: "Denver, CO"}, false, true, true // real, outside the radius
			}
			return snapshot.LocationRef{}, false, false, true
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
	//
	// THIS ASSERTION ALONE CANNOT SEE A CONSTANT FUNCTION, and for a release it
	// did not: `blocker()` returned this string unconditionally and this test
	// passed throughout. The discrimination is proved in
	// request_blocker_test.go — TestTheBlockerIsNotAConstantFunction — which is
	// where a reader should look before trusting this line.
	if got := out.request.blocker(); got != "Choose a location" {
		t.Errorf("the chip says %q; it names the first unmet condition", got)
	}
}

// typeLocation types into the Location field AND lets the debounce run to its
// answer, which is what the operator experiences: keys, a pause, a verdict.
//
// IT DRIVES THE REAL PATH (D-130). A test that set `locate` directly would keep
// passing on the day the pause stopped arming or the verdict stopped being
// filed — and those two are the whole mechanism.
func typeLocation(t *testing.T, d Dashboard, text string) Dashboard {
	t.Helper()
	var m tea.Model = d
	var cmd tea.Cmd
	for _, r := range text {
		m, cmd = m.(Dashboard).requestType(string(r))
	}
	if cmd == nil {
		t.Fatal("typing into the Location field armed no pause: the debounce is not wired")
	}
	pause, ok := cmd().(locatePauseMsg)
	if !ok {
		t.Fatalf("the Location field armed something other than a pause: %T", cmd())
	}
	m, cmd = m.(Dashboard).handleLocatePause(pause)
	if cmd == nil {
		t.Fatal("the pause asked nothing")
	}
	m, _ = m.(Dashboard).handleLocateVerdict(cmd().(locateVerdictMsg))
	return m.(Dashboard)
}

// AND A COMPLETE ONE IS SENT ONCE, AND CLOSES.
func TestACompleteRequestIsScheduledAndTheWindowCloses(t *testing.T) {
	var sent int
	d := requestDash(t, &sent)
	d = typeLocation(t, d, "vista")
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
	d = typeLocation(t, d, "denver, co")
	if d.request.locate.ref == nil {
		t.Fatal("a real location outside the radius resolved to nothing; it is not a typo")
	}
	if d.request.locate.within {
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
	d = typeLocation(t, d, "denver, co")
	// THE RENDERED WINDOW, NOT THE BODY.
	//
	// THIS TEST READ `requestBody` AND PASSED WHILE THE COLOUR WAS BROKEN. The
	// body is PRE-WRAP; the window wraps it afterwards, and the wrap is what
	// dropped the tint — "Broadcast Radius" landed on a second line in plain
	// grey and no assertion here could see it, because no assertion here looked
	// at what the operator does.
	d.width, d.height = 120, 40
	lines := strings.Split(d.renderModal(d.opts()), "\n")
	fact, aside := d.request.note()
	if fact == "" || aside == "" {
		t.Fatalf("the window has nothing to say about a location outside the radius: %q / %q", fact, aside)
	}

	// EACH LINE ON ITS OWN. The first version of this joined them and asked
	// whether the TINT appeared anywhere in the result — so removing it from the
	// fact was masked by the aside still carrying it, and mutant mBB1 survived.
	// Two lines, two assertions.
	// THREE PROBES, NAMED RATHER THAN MATCHED BY WORDS. A generic word matcher
	// caught the FOOTER chip, which shares "outside", "service" and "radius"
	// with the fact — so each probe is a word that appears in the helper and
	// nowhere else in the window.
	//
	// "Radius" IS THE ONE THAT MATTERS: it is the wrapped continuation of the
	// aside, the line that lost its colour, and the reason this test now reads
	// the RENDERED window instead of the body.
	want := render.Tok(render.NameWarning)
	for _, probe := range []string{"station's", "Observer", "Radius"} {
		found := false
		for _, l := range lines {
			if !strings.Contains(stripANSITest(l), probe) {
				continue
			}
			found = true
			if !strings.Contains(l, want) {
				t.Errorf("the helper line carrying %q is not tinted %q: %q",
					probe, want, stripANSITest(l))
			}
		}
		if !found {
			t.Errorf("the window drew no helper line carrying %q", probe)
		}
	}
}

// TestTheWindowOpensOnTheBottomSlot.
//
// HUM LEAD, 2026-09-14: "default to the bottom - position 15."
//
// A REQUEST HAS TO GO SOMEWHERE, and the bottom is where it disturbs nothing.
// PRIORITIZE pushes every card down, so the disruptive act is the one CHOSEN
// rather than the one arrived at by not deciding — which is what an unset
// position used to force.
func TestTheWindowOpensOnTheBottomSlot(t *testing.T) {
	st := requestOpen()
	if st.prioritize {
		t.Error("the window opens on PRIORITIZE; the bottom is the default")
	}
	if got, want := st.position(), lineup.MainTrackCap-1; got != want {
		t.Errorf("the window opens on position %d, want %d — the bottom of the running order", got, want)
	}
	if !st.positionOK() {
		t.Error("the default position is not one the running order has")
	}
}

// AND THE WINDOW IS SCHEDULABLE AS SOON AS A LOCATION RESOLVES, because the
// reports and the position both start with an answer.
func TestOnlyTheLocationIsOwedWhenTheWindowOpens(t *testing.T) {
	var sent int
	d := requestDash(t, &sent)
	// SAME CAVEAT AS ABOVE: true of a constant function too. The non-location
	// blockers are pinned in request_blocker_test.go.
	if got := d.request.blocker(); got != "Choose a location" {
		t.Errorf("a fresh window is blocked on %q; only the location is owed", got)
	}
	d = typeLocation(t, d, "vista")
	if !d.request.valid() {
		t.Errorf("a resolved location is still not schedulable: %s", d.request.blocker())
	}
}

// TestPrioritizeIsBoldAndYellow.
//
// HUM LEAD, 2026-09-14: "make 'PRIORITIZE' bold and yellow in the request modal."
//
// IT IS THE ONE CHOICE IN THIS WINDOW THAT MOVES EVERY OTHER CARD, and the
// advisory tone is the app's own word for "this one is different" — the family
// `NameWarning` belongs to, a step down in urgency.
func TestPrioritizeIsBoldAndYellow(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	var sent int
	d := requestDash(t, &sent)
	d.width, d.height = 120, 40
	for _, l := range strings.Split(d.renderModal(d.opts()), "\n") {
		if !strings.Contains(stripANSITest(l), "PRIORITIZE") {
			continue
		}
		if want := render.Tok(render.NameAdvisory); !strings.Contains(l, want) {
			t.Errorf("PRIORITIZE is not tinted %q: %q", want, stripANSITest(l))
		}
		// BOLD IS `1`, and it is asserted on the rendered line for the same
		// reason the tint is: the operator sees the frame, not the string that
		// went into it.
		if !strings.Contains(l, "\x1b[1m") && !strings.Contains(l, ";1m") && !strings.Contains(l, "[1;") {
			t.Errorf("PRIORITIZE is not bold: %q", l)
		}
		return
	}
	t.Fatal("the window drew no PRIORITIZE line")
}
