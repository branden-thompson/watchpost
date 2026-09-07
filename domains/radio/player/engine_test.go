package player

import (
	"context"
	"io"
	"testing"
)

// THE TREATMENT FOLLOWS THE AUDIO, not the moment the alert arrived.
//
// A relay dips under an alert because it is live radio — pausing it means
// resuming into audio that is minutes stale. A rendered cycle holds, because it
// has nowhere to be and dipping loses its words for good. Deciding that once,
// when the alert starts, fixes an answer the source can outlive: a listener who
// tunes a relay while an alert is reading, or a relay that falls back to a
// synthesised cycle mid-alert, would carry the previous source's treatment.
func TestGivingWayFollowsTheSourceNotTheMomentTheAlertArrived(t *testing.T) {
	engine, err := New(&recordingOutput{}, "watchpost/test (t@example.com)", nil)
	if err != nil {
		t.Fatal(err)
	}
	scaleAndPause := func() (float64, bool) {
		engine.mu.Lock()
		defer engine.mu.Unlock()
		return engine.giveWayLocked()
	}

	// Nothing on air: the broadcast plays at the knob.
	if scale, pause := scaleAndPause(); scale != 1 || pause {
		t.Errorf("with no alert the broadcast is untouched, got scale %v pause %v", scale, pause)
	}

	// An alert arrives over a rendered cycle: it holds.
	engine.setLive(false)
	engine.Suppress()
	if scale, pause := scaleAndPause(); !pause || scale != 1 {
		t.Errorf("a rendered cycle holds under an alert, got scale %v pause %v", scale, pause)
	}

	// The listener tunes a relay while that same alert is still reading. It must
	// DIP — a paused relay would resume into audio that had already expired,
	// and silencing it outright tells the listener nothing.
	engine.setLive(true)
	if scale, pause := scaleAndPause(); pause || scale != alertDuck {
		t.Errorf("a relay dips under an alert even one that started elsewhere, got scale %v pause %v", scale, pause)
	}

	// And the relay falls back to a synthesised cycle, still under the alert:
	// the new source holds rather than playing on dipped for its whole life.
	engine.setLive(false)
	if scale, pause := scaleAndPause(); !pause || scale != 1 {
		t.Errorf("a fallback cycle holds, got scale %v pause %v", scale, pause)
	}

	// The alert ends: whatever is playing returns to the knob.
	engine.Restore()
	if scale, pause := scaleAndPause(); scale != 1 || pause {
		t.Errorf("the broadcast returns when the alert is done, got scale %v pause %v", scale, pause)
	}
}

// STARTING A SOURCE RECORDS WHAT KIND IT IS, so the engine can answer the
// question above without being told again.
func TestStartingASourceRecordsWhetherItIsLive(t *testing.T) {
	engine, err := New(&recordingOutput{}, "watchpost/test (t@example.com)", nil)
	if err != nil {
		t.Fatal(err)
	}
	engine.StartSource("cycle", OutputRate, func(context.Context) io.Reader { return endlessPCM{} })
	engine.mu.Lock()
	rendered := engine.live
	engine.mu.Unlock()
	if rendered {
		t.Error("a synthesised cycle is not live radio")
	}
	engine.Start([]string{"http://127.0.0.1:1/nothing"}, "relay") // unreachable: it fails, but the kind is recorded first
	engine.mu.Lock()
	live := engine.live
	engine.mu.Unlock()
	if !live {
		t.Error("a relay is live radio")
	}

	// AND BACK AGAIN. The first check ran on a fresh engine, where `live` is
	// already false — it asserts the zero value, not the recording. Only a
	// rendered source started AFTER a relay shows the line does its work, and
	// that is the direction that matters: a fallback from relay to synth
	// mid-alert would otherwise keep the relay's answer and dip a spoken report
	// to a quarter volume, losing the words the hold exists to preserve.
	engine.StartSource("cycle", OutputRate, func(context.Context) io.Reader { return endlessPCM{} })
	engine.mu.Lock()
	afterRelay := engine.live
	engine.mu.Unlock()
	if afterRelay {
		t.Error("a synthesised cycle started after a relay is still not live radio")
	}
	engine.Halt()
}

// THE DIP DEPTH IS PART OF THE BEHAVIOUR, NOT AN IMPLEMENTATION DETAIL.
//
// Every other give-way assertion is written as knob × alertDuck, so the
// expectation moves with the constant and the whole suite stays green whatever
// it is set to. That leaves the two ends unguarded, and both are hazards: at 1
// a relay plays at the listener's full volume over a live severe-weather alert,
// permanently, which is worse than any timing defect; at 0 it goes silent, and
// silence is what the constant's own comment forbids, because a listener cannot
// tell a dipped broadcast from a stopped one.
func TestTheDipDepthIsPinned(t *testing.T) {
	// 0.15 since MVS-D-70 (UAT 2026-09-03). It was 0.25, ratified 2026-08-27 as
	// "duck, not interrupt"; heard on a live relay the listener called that
	// "still loud enough to be distracting" and asked for another 5-15 % off.
	// Read as PERCENTAGE POINTS of full scale — 5 % of 0.25 is 0.2375, which no
	// ear would separate from 0.25 — putting the range at 0.10-0.20, and this is
	// its midpoint. The direction was the ruling; the exact figure is a knob and
	// this test is where it turns.
	if alertDuck != 0.15 {
		t.Errorf("the dip is 15 %% of the listener's volume, got %v — quiet enough that the alert is what you hear, loud enough that the broadcast is not mistaken for stopped", alertDuck)
	}
}

// MVS-D-75 — STOPPING A LINE PAUSES IT BEFORE CLOSING IT.
//
// THE GAP THIS CLOSES IS WHY THE DEFECT SHIPPED TWICE. Two fixes passed their
// tests and failed in the listener's ears, because every test reached only the
// ARBITER'S DECISION — which was correct all along — and nothing reached the
// audio layer beneath it. The trace eventually reported the truth and it read
// like success: "a player was found and closed", while the report played on.
//
// Close RELEASES a player. Pause is what stops the device emitting what is
// already BUFFERED, and for a minutes-long event read the buffered remainder is
// not a tail — it is the rest of the report.
func TestStoppingALinePausesBeforeClosing(t *testing.T) {
	out := &recordingOutput{}
	e, err := New(out, "watchpost-test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Preview(OutputRate, endlessPCM{}); err != nil {
		t.Fatal(err)
	}
	out.mu.Lock()
	if len(out.players) != 1 {
		out.mu.Unlock()
		t.Fatalf("one clip must be playing, got %d", len(out.players))
	}
	p := out.players[0]
	out.mu.Unlock()

	e.StopPreview()

	acts := p.actions()
	iPause, iClose := indexOf(acts, "pause"), indexOf(acts, "close")
	if iPause < 0 {
		t.Fatalf("a stopped line must be PAUSED, or its buffered audio plays on: %v", acts)
	}
	if iClose < 0 {
		t.Fatalf("a stopped line must be closed: %v", acts)
	}
	if iPause > iClose {
		t.Errorf("pause must come BEFORE close, got %v", acts)
	}
	// And the line is no longer in flight, so nothing later resumes it.
	if p.IsPlaying() {
		t.Error("a stopped line is not playing")
	}
}

func indexOf(ss []string, want string) int {
	for i, s := range ss {
		if s == want {
			return i
		}
	}
	return -1
}
