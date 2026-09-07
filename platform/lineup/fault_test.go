package lineup

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/category"
)

// DR-21 — A FAULT IS SURFACED BY WHAT IT LEAVES, NOT BY WHAT IT WAS.
//
// The table is the rule, written before the code. Each row is one Failed
// arriving; the outcome is what a listener gets.
//
//	# | the schedule after it | surfacing
//	--+-----------------------+------------------------------------------------
//	1 | another card waiting  | none — routed around; the next card reads
//	2 | a card still on air   | none — the station is still talking
//	3 | nothing at all        | ESCALATE — nothing is coming, and only a person
//	  |                       | can fix that
//	4 | nothing, no reason    | ESCALATE, with words rather than an empty string
//
// Row 1 is the one the requirement is written around: "escalating a
// self-healing failure would be a noise regression". A rail whose second takeover
// cannot be rendered still has its third, and the listener hears them — so a
// modal there teaches them to dismiss the window that matters.

// failing is a director with the radio running and n TAKEOVERS on the rail —
// one per burst, because a burst is one card (MVS-D-77).
//
// n BURSTS, NOT n ALERTS. Row 1 is "another card waiting", and the alerts inside
// a burst are its content: fail its card and there is nothing behind it to route
// around to, so a fixture built from one burst would test row 3 while claiming
// row 1.
func failing(t *testing.T, n int) Director {
	t.Helper()
	d := New(Settings{Max: 10}, planNow)
	d, _ = run(d, Powered{To: Running})
	for _, p := range []string{"a", "b", "c", "d"}[:n] {
		d, _ = run(d, Arrived{Arrivals: many(p, category.Warnings, 2)})
	}
	return d
}

func TestDR21AFaultRoutedAroundRaisesNothing(t *testing.T) {
	// Rows 1 and 2: three takeovers, one fails. Two remain, so the rail carries on.
	d := failing(t, 3)
	var id string
	for tr := Track(0); tr < numTracks; tr++ {
		for _, c := range d.lineup.Cards(tr) {
			if id == "" {
				id = c.ID
			}
		}
	}
	if id == "" {
		t.Fatal("no card to fail")
	}
	_, fx := d.Step(Failed{ID: id, Reason: "the voice could not render"})
	for _, f := range fx {
		if _, ok := f.(Escalate); ok {
			t.Errorf("a fault the schedule routed around must raise nothing: %v", describeAll(fx))
		}
	}
}

func TestDR21AFaultThatStopsTheScheduleEscalatesOnce(t *testing.T) {
	// Row 3: one takeover, and it fails. Nothing is left.
	d := failing(t, 1)
	var id string
	for tr := Track(0); tr < numTracks; tr++ {
		for _, c := range d.lineup.Cards(tr) {
			id = c.ID
		}
	}
	if id == "" {
		t.Fatal("no card to fail")
	}
	_, fx := d.Step(Failed{ID: id, Reason: "the voice could not render"})

	var got []Escalate
	for _, f := range fx {
		if e, ok := f.(Escalate); ok {
			got = append(got, e)
		}
	}
	if len(got) != 1 {
		t.Fatalf("ONE escalation channel: got %d in %v", len(got), describeAll(fx))
	}
	if got[0].ID != id {
		t.Errorf("the escalation names the card that failed, got %q", got[0].ID)
	}
	// Row 4: the producer's own words travel, because "something went wrong"
	// tells a listener only what the silence already told them.
	if !strings.Contains(got[0].Reason, "could not render") {
		t.Errorf("the producer's reason travels, got %q", got[0].Reason)
	}
}

func TestDR21AnEscalationAlwaysSaysSomething(t *testing.T) {
	// Row 4: a producer that failed without words still yields words.
	d := failing(t, 1)
	var id string
	for tr := Track(0); tr < numTracks; tr++ {
		for _, c := range d.lineup.Cards(tr) {
			id = c.ID
		}
	}
	_, fx := d.Step(Failed{ID: id})
	for _, f := range fx {
		if e, ok := f.(Escalate); ok {
			if strings.TrimSpace(e.Reason) == "" {
				t.Error("an escalation with no reason is a window that says nothing")
			}
			return
		}
	}
	t.Fatal("a fault that stops the schedule escalates")
}

// A fault for a card the schedule no longer holds changes nothing at all — it
// was discarded while its work was in flight, which is ordinary.
func TestDR21AFaultForAForgottenCardIsSilent(t *testing.T) {
	d := failing(t, 2)
	_, fx := d.Step(Failed{ID: "not-a-card", Reason: "gone"})
	if len(fx) != 0 {
		t.Errorf("a fault for a card that is not held does nothing, got %v", describeAll(fx))
	}
}

func describeAll(fx []Effect) []string {
	out := make([]string, 0, len(fx))
	for _, f := range fx {
		out = append(out, Describe(f))
	}
	return out
}

var _ = time.Second
