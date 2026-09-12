package app

// livesource.go — what the deck holds while a synthesized broadcast is running
// (D-91).
//
// AN INTERFACE DECLARED WHERE IT IS CONSUMED, which is where Go puts one and
// where this repo's architecture puts one: `domains/radio/synth` owns what a
// Source IS, and `app` says what it needs from one. Nothing is added to the
// domain — the domain does not learn that this package exists, let alone that it
// has tests.
//
// WHY IT EXISTS AT ALL. The field was `*synth.Source`, a concrete pointer, so no
// test in this package could observe whether the air guard actually stopped a
// `Loop` or a `Recast` — only that the PREDICATE said it should. That is exactly
// how D-74's plant `y4` survived: "my tests asserted `monitorHasTheAir()` — the
// PREDICATE — and never that `needsRead` actually skips the audio." P-1 states
// the rule: a seam a test cannot drive is not a covered seam.
//
// THE FIRST DRAFT REACHED INTO THE DOMAIN INSTEAD, adding `RepeatingForTest` to
// `synth.Source` and citing `lineup.MonitorAdvancesForTest` as precedent. The
// HUM LEAD caught it. All four `ForTest` exports in the tree are in `platform/`;
// there is none in `domains/`, and the rule is that a domain owns its own
// business and shared things live in `platform`. The precedent did not say what
// I claimed it said.
//
// THE SHAPE IS ALREADY IN THIS PACKAGE. `synthSource()` returns
// `interface{ Cached() (int, int) }` — the same move, one method wide, for the
// [S] window. This is that, for the whole surface the deck uses.

import (
	"context"
	"io"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
)

// liveSource is the running synthesized broadcast, as the deck uses it.
//
// EXACTLY THE NINE METHODS `app` CALLS, and no more: an interface wider than its
// consumer is a second copy of the type's documentation, and it goes stale the
// first time the domain grows a method nobody here wants.
type liveSource interface {
	// The cast, installed before the source starts so the first segment already
	// resolves through it.
	SetResolver(r func(cast.Role) (synth.Voice, error))
	SetHandoffLine(f func(from, to string) string)

	// THE TWO THAT REACH A BROADCAST IN FLIGHT, and therefore the two the air
	// guard is about (D-91). Everything else here is set up before the source
	// plays or read after it.
	Loop(on bool)
	Recast()

	// Invalidate is the SOFT change — a host fact landed, and it takes effect at
	// the next segment rendered rather than now.
	Invalidate()

	Err() error
	Rate() int
	Cached() (entries int, bytes int)
	Open(ctx context.Context) io.Reader
}

// *synth.Source is the only production implementation, and this is where that is
// checked rather than at the first assignment.
var _ liveSource = (*synth.Source)(nil)
