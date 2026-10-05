package app

// air_boundary.go — who may reach the air, and the closed set that says so (D-91).
//
// HUM LEAD, 2026-09-12:
//
//	"masterControl is the one who determines who gets the air.  In Observer mode,
//	 Observer ALWAYS gets the air … In Broadcaster Mode, Broadcaster ALWAYS gets
//	 the air, so nothing from Observer should ever be able to re-tune, take over,
//	 or 'sneak under' to get on the air."
//
// THE MODEL IS NOT NEW AND IS NOT IN QUESTION (D-74, and 00-REQUIRED-READING's
// "ONE DECK, ONE AIR"): both surfaces run through ONE deck on purpose, the
// Director arbitrates both rotations, and MasterControl gates the deck from the
// air. What this file holds is that EVERY one of the deck's entry points asks.
//
// SIZED BY THE SURVEY (FR-2.1a, 02-analysis/air-reachability-survey.md).
// Nineteen functions in this package can change what is audible; seven need a
// decision and twelve owe nothing. The mechanism is aimed at those seven, not at
// "everything that can reach the air".
//
// THE CLASSIFICATION ITSELF LIVES IN THE TEST (air_boundary_test.go). The guards
// check `monitorHasTheAir()` directly, so a table in production has no reader —
// the `wires` gate reports every member NO WRITER — and is a SECOND CARRIER of
// what the guards already say, free to drift from them. What the table is FOR is
// the gate: "every seam is classified, and every monitor row has a behaviour
// test." That is a test's job, so it lives with the test.
//
// THE GUARD IS AT THE CALLER, NOT THE METHOD: `tune` serves the monitor's
// `SetMode` AND the Director's `Tune` effect; `tuneCallsign` serves Observer's
// relay pick (MVS-D-76). A guard inside it would break the half that is
// entitled to the air.
//
// THE CONSOLE'S BED IS ON THE OTHER SIDE OF THAT SEAM (D-117). Tuning it through
// `tuneCallsign` requires the callsign to already be in the list the LISTENER's
// last tune left behind, so the bed silently does nothing. It resolves at the
// STATION's epicentre and tunes what it resolved (`tuneResolved`), which is a
// different seam with the same entitlement.
// The EXPORTED `tty.Radio` methods are the monitor's control surface; the
// lower-case internals are shared, and that split is the seam.

import (
	"github.com/branden-thompson/watchpost/modes/tty"
)

// monitorMayReachTheAir reports whether OBSERVER'S OWN controls may touch the
// engine right now.
//
// ONE PREDICATE, ASKED BY EVERY MONITOR-INITIATED ENTRY. The deck already has
// `monitorHasTheAir()` for its own use and this is the same question asked from
// the app's side of the boundary — both read `lp.owner`, which is what
// `takeTheAir` sets and therefore the one thing that knows.
func (lp *livePipelines) monitorMayReachTheAir() bool {
	return lp != nil && lp.owner.get() != tty.SurfaceBroadcaster
}
