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
// air. What was missing is that only ONE of the deck's entry points asked.
//
// SIZED BEFORE IT WAS BUILT (FR-2.1a, 02-analysis/air-reachability-survey.md).
// Nineteen functions in this package can change what is audible; three were live
// defects, four were latent, and twelve owe nothing. A mechanism aimed at
// "everything that can reach the air" would have been aimed at nineteen when the
// population that needs a decision is seven.
//
// THE CLASSIFICATION ITSELF LIVES IN THE TEST (air_boundary_test.go), and the
// `wires` gate is what put it there: every member of `airReach` was reported
// NO WRITER, because the guards check `monitorHasTheAir()` directly and never
// consulted the table. A table production does not read is a SECOND CARRIER of
// what the guards already say, and it would drift from them — which is the exact
// shape this whole batch exists to remove. What the table is FOR is the gate:
// "every seam is classified, and every monitor row has a behaviour test." That
// is a test's job, so it lives with the test.
//
// THE GUARD IS AT THE CALLER, NOT THE METHOD, and the survey is what settled
// that: `tune` serves the monitor's `SetMode` AND the Director's `Tune` effect;
// `tuneCallsign` serves Observer's relay pick AND the console's own bed selector
// (D-90). A guard inside either would break the half that is entitled to the air.
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
