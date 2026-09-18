package lineup

import (
	"strconv"
	"strings"
	"testing"

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

// F-150 — A FAULT ON A LIVE STATION MUST STILL BE HEARD. `stopped()` is never
// true on a topped-off station (the producer refills on every publish), so a
// fault that only escalates when the schedule is empty escalates never; and a
// cool-off that is noted only for ROUTED failures leaves the one class that is
// NOT deliberate — a compose error, no composer, an empty report — free to be
// re-admitted at pump speed. Ten consecutive faults, three cards deep:
// every failed location sits out, and the window is owed by the third.
func TestAFaultRunOnAToppedOffStationEscalatesAndSitsOut(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	d, _ = run(d, Powered{To: Running})
	for _, ref := range []string{"a", "b", "c"} { // bounded by the fixture (P10-02)
		d, _ = run(d, NeedsRead{Ref: ref, Headline: ref})
	}
	if got := len(d.lineup.Cards(MainTrack)); got != 3 {
		t.Fatalf("the fixture holds %d main-track cards, want 3", got)
	}
	var escalations int
	for i := 0; i < 10; i++ { // bounded by the fault count (P10-02)
		// THE PRODUCER REFILLS WITH FRESH PLACES ON EVERY PUBLISH, so the schedule
		// never empties and stopped() is never true; the run, not emptiness, is
		// what decides.
		tag := "p" + strconv.Itoa(i)
		var fx []string
		d, fx = faultOnce(t, d, tag)
		if escalated(fx) {
			escalations++
		}
		if !d.sittingOut(tag + "a") {
			t.Errorf("fault %d: %s was not put in cool-off — it will be re-admitted at pump speed", i+1, tag+"a")
		}
		if i == 2 && escalations == 0 {
			t.Errorf("three consecutive faults on a live station raised no escalation; the operator sees ON AIR over dead air")
		}
	}
	if escalations == 0 {
		t.Errorf("ten consecutive faults raised no escalation")
	}
	// THE CONTROL: a routed decline never escalates on a live station.
	c := New(Settings{Max: 10}, planNow)
	c, _ = run(c, Powered{To: Running})
	c, _ = run(c, NeedsRead{Ref: "x", Headline: "x"})
	id := c.lineup.Cards(MainTrack)[0].ID
	_, fx := run(c, Failed{ID: id, Reason: "muted", Routed: true})
	if escalated(fx) {
		t.Errorf("a routed decline escalated: %v", fx)
	}
}

// topOff is the producer refilling a live station with fresh places so that
// stopped() is never the reason a fault escalates (the reviewer's condition).
func topOff(t *testing.T, d Director, tag string) (Director, []Card) {
	t.Helper()
	for _, ref := range []string{tag + "a", tag + "b"} { // bounded by the fixture (P10-02)
		d, _ = run(d, NeedsRead{Ref: ref, Headline: ref})
	}
	cards := d.lineup.Cards(MainTrack)
	if len(cards) < 2 {
		t.Fatalf("%s: the station is not topped off (%d cards); stopped() would decide, not the run", tag, len(cards))
	}
	return d, cards
}

// faultOnce tops the station off with the tag's places and fails the first of
// them as a fault, returning the effects; the place is then `tag+"a"`.
func faultOnce(t *testing.T, d Director, tag string) (Director, []string) {
	t.Helper()
	var cards []Card
	d, cards = topOff(t, d, tag)
	for _, c := range cards { // bounded by the schedule (P10-02)
		if c.Subject == tag+"a" {
			return run(d, Failed{ID: c.ID, Reason: "the report could not be composed: no key", Routed: false})
		}
	}
	t.Fatalf("%s: the topped-off station does not hold %sa", tag, tag)
	return d, nil
}

func escalated(fx []string) bool {
	for _, f := range fx { // bounded by the effects (P10-02)
		if strings.Contains(f, "escalate(") {
			return true
		}
	}
	return false
}

// R2 REVIEW F2 (2026-09-17) — THE COOL-OFF HAS TWO DOORS AND BOTH ARE SHUT. Only
// the top-off asked sittingOut; a NeedsRead for the same place queued it straight
// back in, and the deck raises one on every relay failure — the UAT 2026-09-10
// loop through the second door.
func TestANeedsReadDoesNotWalkPastTheCoolOff(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	d, _ = run(d, Powered{To: Running})
	d, _ = faultOnce(t, d, "p")
	if !d.sittingOut("pa") {
		t.Fatal("the fixture's fault did not put the place in cool-off")
	}
	d, _ = run(d, NeedsRead{Ref: "pa", Headline: "pa"})
	for _, c := range d.lineup.Cards(MainTrack) { // bounded by the schedule (P10-02)
		if c.Subject == "pa" {
			t.Fatalf("a place in cool-off was re-admitted through NeedsRead: %s", c.ID)
		}
	}
}

// R2 REVIEW F4 — A READ THAT FINISHED ENDS THE RUN, and the escalation carries
// the count. Two faults, a finished read, two more faults: no escalation; the
// third after the finish is the third of a NEW run.
func TestAFinishedReadBetweenFaultsResetsTheRun(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	d, _ = run(d, Powered{To: Running})
	d, _ = faultOnce(t, d, "q1")
	d, _ = faultOnce(t, d, "q2")
	var cards []Card
	d, cards = topOff(t, d, "q3")
	d, _ = run(d, Finished{ID: cards[0].ID})
	var fx []string
	d, fx = faultOnce(t, d, "q4")
	if escalated(fx) {
		t.Fatalf("the first fault after a finished read escalated — the run did not reset: %v", fx)
	}
	d, fx = faultOnce(t, d, "q5")
	if escalated(fx) {
		t.Fatalf("the second fault after a finished read escalated: %v", fx)
	}
	_, fx = faultOnce(t, d, "q6")
	if !escalated(fx) {
		t.Fatalf("the third fault of the new run did not escalate: %v", fx)
	}
	for _, f := range fx { // bounded by the effects (P10-02)
		if strings.Contains(f, "escalate(") && !strings.Contains(f, "run=3") {
			t.Errorf("the escalation does not carry the run it was raised at: %s", f)
		}
	}
}

// R2 REVIEW F5 — STANDBY ENDS THE RUN, on the Director as on the console. The
// console clears the band when the operator takes the station off the air; a
// Director that kept counting would raise "3 CARD(S) FAILED" after one fault
// on the way back up.
func TestStandbyEndsTheFaultRun(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	d, _ = run(d, Powered{To: Running})
	d, _ = faultOnce(t, d, "s1")
	d, _ = faultOnce(t, d, "s2")
	d, _ = run(d, Powered{To: OffAir})
	d, _ = run(d, Powered{To: Running})
	var fx []string
	d, fx = faultOnce(t, d, "s3")
	if escalated(fx) {
		t.Fatalf("one fault after standby escalated — the run survived the operator's standby: %v", fx)
	}
	d, _ = faultOnce(t, d, "s4")
	_, fx = faultOnce(t, d, "s5")
	if !escalated(fx) {
		t.Fatalf("three faults after standby did not escalate: %v", fx)
	}
}

// R2b REVIEW S1 — A FINISHED FOR A CARD THE SCHEDULE DOES NOT HOLD IS NOT A
// READ. A stale Finished from a superseded read arrives exactly in the churn
// where faults happen; if it reset the run, the count the band exists to
// deliver would be wiped by a ghost.
func TestAGhostFinishedDoesNotResetTheRun(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	d, _ = run(d, Powered{To: Running})
	d, _ = faultOnce(t, d, "g1")
	d, _ = faultOnce(t, d, "g2")
	d, _ = run(d, Finished{ID: "read:ghost"})
	_, fx := faultOnce(t, d, "g3")
	if !escalated(fx) {
		t.Fatalf("a Finished for a card the schedule never held reset the run — the third fault did not escalate: %v", fx)
	}
}

// R2b REVIEW S4 — THE RUN IS A LIVE STATION'S. A fault that lands while the
// station is OFF AIR (one in-flight compose, say) is not a card the operator's
// listeners missed, and it does not count toward the run that owes them the
// band; standby ends the run on the way down, and this keeps it ended.
func TestFaultsOffAirDoNotCountTowardTheRun(t *testing.T) {
	d := New(Settings{Max: 10}, planNow)
	d, _ = run(d, Powered{To: Running})
	d, _ = run(d, Powered{To: OffAir})
	for _, tag := range []string{"o1", "o2", "o3"} { // bounded by the fixture (P10-02)
		var fx []string
		d, fx = faultOnce(t, d, tag)
		if escalated(fx) {
			t.Fatalf("a fault while OFF AIR escalated: %v", fx)
		}
	}
	d, _ = run(d, Powered{To: Running})
	d, _ = faultOnce(t, d, "o4")
	var fx []string
	d, fx = faultOnce(t, d, "o5")
	if escalated(fx) {
		t.Fatalf("two faults after coming back ON AIR escalated — the off-air faults were counted: %v", fx)
	}
	_, fx = faultOnce(t, d, "o6")
	if !escalated(fx) {
		t.Fatalf("three faults ON AIR did not escalate: %v", fx)
	}
}
