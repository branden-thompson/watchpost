package app

// transition.go — the Director's own words at a seam (F-27).
//
// A listener should never be handed a hard cut from a takeover or a [w] read
// back into the programme. The station identification on going ON AIR from
// standby (F-24) was the second such seam; it was never wired, and is removed
// (D-158) - its words are kept, and mastercontrol.GoOnAir says where it would
// be read.
//
// THESE ARE THE DIRECTOR'S ONLY ADDITIVE ACT (role-model.md). It composes no
// alert and no report — the Producer and the Composer own those — but it is the
// only thing that knows two adjacent cards came from different places, and the
// only thing that knows the station has just gone on air.

import (
	"github.com/branden-thompson/watchpost/domains/radio/script"
)

// programmeReturnLine is the transition back to the programme after a read ends —
// including an [esc] that cut one short (HUM LEAD, UAT 2026-09-05).
//
// ONE SCRIPT FOR BOTH MODES. What differs is only whether the programme kept
// running underneath: a LIVE RELAY did, so the listener is rejoined to
// something already under way; a synth read did not, and starts its next card
// cleanly. Two scripts would restate the sentence to vary one clause.
//
// NAMED programmeReturnLine, NOT resumeLine: mastercontrol already has a
// resumeLine method — the arbiter's suspension mechanic, which plays a HELD line
// on from where it stopped. Two different things under one name in one package
// is how a reader reaches for the wrong one, and P10-01 resolves by NAME.
//
// It is what makes the return audible. Without it a read simply stops and the
// next thing starts in the same voice, which the HUM LEAD reported as jarring
// in synth mode — a human station never does this.
func programmeReturnLine(lib *script.Library, live bool) string {
	return scriptText(lib, "transition", "resume", map[string]any{"InProgress": live})
}
