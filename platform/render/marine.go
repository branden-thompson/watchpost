package render

import (
	"time"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// The words the sea is described in — lifted here from modes/tty so the SCREEN
// and the VOICE cannot describe the same sea differently (RS-11).
//
// Before 0.14.0 these lived in the Details view alone, because only the Details
// view spoke about the sea. The maritime report reads the same records aloud,
// and two implementations of "is this rough?" is precisely the kind of drift a
// listener would catch and never trust again. The lift is a MOVE, not a
// rewrite: the Details view's output is byte-for-byte what it was.

// SeaState words a significant wave height using the Douglas sea-state bands.
func SeaState(m float64) string {
	switch {
	case m < 0.1:
		return "Calm (glassy)"
	case m < 0.5:
		return "Smooth"
	case m < 1.25:
		return "Slight Chop"
	case m < 2.5:
		return "Moderate Chop"
	case m < 4:
		return "Rough"
	}
	return "Very Rough"
}

// TideTrend is "Rising" when the next high comes before the next low,
// "Falling" when it does not, and empty when neither is predicted.
func TideTrend(nextHigh, nextLow *snapshot.TideEvent) string {
	switch {
	case nextHigh != nil && (nextLow == nil || nextHigh.Time.Before(nextLow.Time)):
		return "Rising"
	case nextLow != nil:
		return "Falling"
	}
	return ""
}

// NextTide is the first event of typ after now (events are time-ordered).
func NextTide(events []snapshot.TideEvent, typ string, now time.Time) *snapshot.TideEvent {
	for i := range events {
		if events[i].Type == typ && events[i].Time.After(now) {
			return &events[i]
		}
	}
	return nil
}

// CurrentPhase is the tidal current in force — the last predicted extreme at or
// before now — and the next predicted event after it. Either may be nil: before
// the first prediction there is no phase yet, and after the last there is no
// next.
func CurrentPhase(events []snapshot.CurrentEvent, now time.Time) (inForce, next *snapshot.CurrentEvent) {
	for i := range events {
		if !events[i].Time.After(now) {
			inForce = &events[i]
		} else if next == nil {
			next = &events[i]
		}
	}
	return inForce, next
}

// FirstOf returns the first non-nil value — the "use the swell height, or the
// wave height, or the wind-wave height" rule the marine records need in several
// places.
func FirstOf(vals ...*float64) *float64 {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}
