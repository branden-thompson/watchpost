package app

import "testing"

// I-1 — DR-21's ONE ESCALATION CHANNEL SURVIVES A STATION WITH NO AUDIO.
//
// A nil deck is first-class: buildDirector handles it explicitly and its own
// comment says "a nil deck (no audio) is fine". startSchedule wires
// `escalate: func(reason string) { deck.escalate(reason) }` with no guard,
// while the same function guards `if deck != nil` for cutTo further down.
//
// NOTHING ELSE COULD SEE THIS, which is why it is pinned at the receiver. The
// invariant in newExecutors passes — the closure is non-nil while the channel
// behind it is dead. The pump contains the panic; Escalate names no card, so
// no Failed is emitted; and onFault writes one line to a log that is off by
// default. On an audio-less machine a fault therefore silenced the station and
// told nobody, which is exactly what DR-21 exists to prevent.
func TestEscalatingOnAStationWithNoAudioIsRefusedNotFatal(t *testing.T) {
	var deck *radioDeck // no audio device: buildDirector's supported case
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("the fault channel panicked on a station with no audio: %v", r)
		}
	}()
	deck.escalate("the schedule stopped")
}
