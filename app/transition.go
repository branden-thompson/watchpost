package app

// transition.go — the Director's own words at a seam (F-27, F-24).
//
// TWO BOUNDARIES, ONE IDEA. A listener should never be handed a hard cut: from
// a takeover or a [w] read back into the programme, and from standby onto the
// air. Both are moments where the station has something to say and, not
// coincidentally, needs a moment to have something ready.
//
// THESE ARE THE DIRECTOR'S ONLY ADDITIVE ACT (role-model.md). It composes no
// alert and no report — the Producer and the Composer own those — but it is the
// only thing that knows two adjacent cards came from different places, and the
// only thing that knows the station has just gone on air.

import (
	"strconv"

	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/platform/snapshot"
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

// mastheadLine is the station identification read on going ON AIR from standby
// (F-24, HUM LEAD 2026-09-05).
//
// IT IS DELIBERATELY LONG, and that is the point rather than a cost. Coming back
// on air, the first card must be composed and rendered before anything can be
// said — about a second on macOS, and the Reader falls back to a five-second
// hold when a render yields nothing. So the moment a listener says "go" is
// otherwise the moment of the longest silence, which reads as the station being
// broken at exactly the wrong time. This covers that gap with something the
// station means, and its last sentence is the limitation a weather service is
// obliged to state.
func mastheadLine(lib *script.Library, at snapshot.LocationRef, radiusMi int, providers []string) string {
	return scriptText(lib, "transition", "masthead", map[string]any{
		"Location":  at.Label,
		"Coverage":  coveragePhrase(radiusMi),
		"Providers": spokenList(providers),
	})
}

// coveragePhrase is the service area as a person says it.
//
// ZERO IS "ALL LOCATIONS", not "a 0 mile radius". It is the same `0 = All`
// the ALERTS - EVENTS setting already uses, and the masthead is the one place
// that value is spoken rather than shown — a station that announced a nought-
// mile radius would be stating the opposite of what it covers.
func coveragePhrase(radiusMi int) string {
	if radiusMi <= 0 {
		return "all locations"
	}
	return "a " + strconv.Itoa(radiusMi) + " mile radius"
}
