//go:build !watchpost_debug

package app

// inject_release.go — the injection seam's ABSENCE in a release build (F-21b).
//
// The tick calls takeInjected unconditionally so the two builds share one call
// site rather than a conditional the reader has to hold in their head. Here it
// returns nothing, and no Inject exists to put anything there: the capability is
// absent from the binary, not disabled within it.

import (
	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/modes/tty"
)

// injectQueue is EMPTY here — zero bytes on the deck — and it still answers
// take(), so the field is USED in both builds rather than dead in one. An
// unused field is what P10 reports and AP-DEAD-01 forbids, and the honest
// shape is a queue that exists and holds nothing rather than a field nobody
// reads.
type injectQueue struct{}

func (q *injectQueue) take() []globalfeed.Event { return nil }

func (t *tickerDeck) takeInjected() []globalfeed.Event { return t.inject.take() }

// No injector and no scenarios: the window renders its "not available in this
// build" text because the app has nothing to give it.
func debugScenarios() []tty.DebugScenario          { return nil }
func (lp *livePipelines) injectHook() func(string) { return nil }
