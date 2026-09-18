package tty

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// NFR-7 — a silent station is BOUNDED.
//
// STANDBY holds every track INCLUDING the alert rail, which is correct and
// deliberate: it is what stops a station on standby putting a tornado warning
// to air. But it means a station can sit silent with a severe-weather card
// HELD and nothing tells anyone. The safety lens found this at PLAN; the
// bound is an escalating notice the operator cannot miss.

func bcStandby(t *testing.T, held bool, forDur time.Duration) Broadcaster {
	t.Helper()
	base := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	b := NewBroadcaster()
	b.width, b.height = 150, 74
	b.now = func() time.Time { return base.Add(forDur) }
	if held {
		var l lineup.Lineup
		next, err := l.Queue(lineup.AlertRail, card(t, "x", "TORNADO WARNING"))
		if err != nil {
			t.Fatal(err)
		}
		b, _ = b.Update(LineupMsg{Lineup: next})
	}
	b.now = func() time.Time { return base }
	b, _ = b.Update(StationMsg{Power: lineup.OffAir}) // standby starts now
	b.now = func() time.Time { return base.Add(forDur) }
	return b
}

func TestAHeldAlertInStandbyRaisesANotice(t *testing.T) {
	got := bcStandby(t, true, 90*time.Second).View().Content
	if !strings.Contains(strings.ToUpper(got), "HELD") {
		t.Error("NFR-7: a hazard held on the rail while the station is silent must be visible — " +
			"STANDBY holds the rail too, which is exactly why nobody is told otherwise")
	}
}

func TestTheHeldNoticeEscalatesWithTime(t *testing.T) {
	early := bcStandby(t, true, 30*time.Second).View().Content
	late := bcStandby(t, true, 20*time.Minute).View().Content
	if early == late {
		t.Error("NFR-7's exit condition is that the console ESCALATES over time rather than sitting " +
			"unchanged; the notice reads identically at 30s and at 20m")
	}
}

func TestAnEmptyRailInStandbyRaisesNothing(t *testing.T) {
	got := bcStandby(t, false, 30*time.Minute).View().Content
	if strings.Contains(strings.ToUpper(got), "HELD") {
		t.Error("a silent station holding NOTHING is a station at rest, not a hazard withheld — " +
			"crying wolf here would train the operator to ignore the one that matters")
	}
}

func TestARunningStationRaisesNoStandbyNotice(t *testing.T) {
	b := bcStandby(t, true, 30*time.Minute)
	b, _ = b.Update(StationMsg{Power: lineup.Running})
	if strings.Contains(strings.ToUpper(b.View().Content), "HELD") {
		t.Error("a RUNNING station is draining the rail; the standby notice must not persist")
	}
}

// A REPEATED StationMsg MUST NOT RESTART THE CLOCK.
//
// Found by a plant that SURVIVED: changing the transition test to `if true`
// made every StationMsg restart the standby clock, and nothing failed —
// because every fixture above sends exactly one. A station silent for an hour
// would have looked freshly quiet the moment any other message arrived, and
// the escalation would never have climbed past its first rung.
func TestARepeatedStationMessageDoesNotRestartTheStandbyClock(t *testing.T) {
	base := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	b := NewBroadcaster()
	b.width, b.height = 150, 74
	var l lineup.Lineup
	next, err := l.Queue(lineup.AlertRail, card(t, "x", "TORNADO WARNING"))
	if err != nil {
		t.Fatal(err)
	}
	b, _ = b.Update(LineupMsg{Lineup: next})

	b.now = func() time.Time { return base }
	b, _ = b.Update(StationMsg{Power: lineup.OffAir}) // silent from here

	// Twenty minutes later the SAME power is published again — a refresh, not
	// a transition.
	b.now = func() time.Time { return base.Add(20 * time.Minute) }
	b, _ = b.Update(StationMsg{Power: lineup.OffAir})

	got := b.View().Content
	if !strings.Contains(got, "!!!") {
		t.Errorf("a repeat of the same power restarted the standby clock: after 20 minutes held the "+
			"notice should be at its most severe rung, got:\n%s", got)
	}
}
