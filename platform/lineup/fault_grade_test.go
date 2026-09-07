package lineup

import "testing"

// I-2 — THE GRADE IS "WAS THIS DELIBERATE", NOT "IS THE SCHEDULE EMPTY".
//
// stopped() cannot tell a station fault from a listener pressing esc: a burst
// is ONE card (MVS-D-77) and nothing queues the main track, so the schedule is
// empty after ANY rail card leaves. Both rows below run against that same empty
// schedule, which is the whole point — only Routed separates them.
func TestOnlyAnUndeliberateFailureRaisesTheWindow(t *testing.T) {
	d := failing(t, 0) // powered, running, nothing scheduled
	if !d.stopped() {
		t.Fatal("the fixture's schedule is not empty; the two rows would differ for the wrong reason")
	}
	for _, tc := range []struct {
		name string
		ev   Failed
		want int
	}{
		{"a listener muted the station", Failed{ID: "burst:a", Reason: "muted", Routed: true}, 0},
		{"a read the listener stopped", Failed{ID: "burst:a", Reason: "cut short", Routed: true}, 0},
		{"an executor panicked", Failed{ID: "burst:a", Reason: "speak(x) panicked"}, 1},
	} {
		if got := len(d.escalation(tc.ev)); got != tc.want {
			t.Errorf("%s: raised %d escalation(s), want %d", tc.name, got, tc.want)
		}
	}
}
