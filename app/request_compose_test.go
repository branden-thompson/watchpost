package app

// request_compose_test.go — a location the operator may REQUEST must be a
// location the Composer can RESOLVE.
//
// FOUND BY RED TEAM AT BUILD EXIT, 0.16.0 (2026-09-15). D-130 widened the
// lookup past the 25-slot pool on the HUM LEAD's ruling — the hyper-local case,
// Rainbow CA, is inside the service radius and in neither the city nor the zip
// table. The COMPOSER was not widened with it: `composeFor(deck, pool)` resolves
// a card's Subject against `currentPool()`, so a requested out-of-pool ref
// errored, the Director declined it, and `Failed{Routed:true}` is treated as
// deliberate and self-healing — so NOTHING was surfaced and the card vanished
// after the window had closed as though it were scheduled.
//
// THAT IS FR-3.3's NAMED TRAP VERBATIM: "an action must never be shown as taken
// unless the schedule took it."

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/report"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestARequestedOutOfPoolLocationCanStillBeComposed(t *testing.T) {
	vista := snapshot.LocationRef{Label: "Vista, CA", Zip: "92084", Lat: 33.2, Lon: -117.24}
	// Rainbow is the HUM LEAD's own example: 14.7 mi from Oceanside, inside the
	// 25-mile radius, and absent from the pool.
	rainbow := snapshot.LocationRef{Label: "Rainbow, CA", Tag: "RAINBOW", Lat: 33.41031, Lon: -117.14781}

	lp := &livePipelines{poolRefs: []snapshot.LocationRef{vista}}

	if _, ok := refFor(lp.currentPool, string(snapshot.Key(rainbow))); ok {
		t.Fatal("fixture: Rainbow must NOT be in the pool, or this test proves nothing")
	}

	lp.rememberRequested(rainbow)

	if _, ok := refFor(lp.resolvable, string(snapshot.Key(rainbow))); !ok {
		t.Error("a location the operator requested cannot be resolved by the Composer: " +
			"the card is minted, the window closes as if scheduled, the build fails and " +
			"Failed{Routed:true} swallows it — the operator is shown an action nothing took")
	}
	// AND THE POOL IS STILL THERE. Widening must not replace.
	if _, ok := refFor(lp.resolvable, string(snapshot.Key(vista))); !ok {
		t.Error("the pool's own locations stopped resolving")
	}
}

// AND THE PRODUCER IS NOT WIDENED WITH IT (D-72, D-40). What the Director may
// be OFFERED is the station's pool; what the Composer must be able to RESOLVE
// now also includes what the operator asked for. Merging the two would let the
// Producer propose a location the station never chose.
func TestRememberingARequestDoesNotWidenWhatTheProducerOffers(t *testing.T) {
	vista := snapshot.LocationRef{Label: "Vista, CA", Zip: "92084", Lat: 33.2, Lon: -117.24}
	rainbow := snapshot.LocationRef{Label: "Rainbow, CA", Tag: "RAINBOW", Lat: 33.41031, Lon: -117.14781}

	lp := &livePipelines{poolRefs: []snapshot.LocationRef{vista}}
	lp.rememberRequested(rainbow)

	for _, r := range lp.producer()() {
		if r.Label == rainbow.Label {
			t.Error("the Producer may now offer a location the station's pool never held")
		}
	}
}

// AND THE SAME REQUEST TWICE DOES NOT GROW THE LIST. The operator re-requesting
// a location is ordinary; an unbounded list behind it is not (P10-02).
func TestARepeatedRequestIsRememberedOnce(t *testing.T) {
	rainbow := snapshot.LocationRef{Label: "Rainbow, CA", Tag: "RAINBOW", Lat: 33.41031, Lon: -117.14781}
	lp := &livePipelines{}
	for range 5 {
		lp.rememberRequested(rainbow)
	}
	var n int
	for _, r := range lp.resolvable() {
		if r.Label == rainbow.Label {
			n++
		}
	}
	if n != 1 {
		t.Errorf("the same request is held %d times, want 1", n)
	}
}

// AND requestCard REMEMBERS, which is the wiring that makes the rest true.
func TestRequestCardRemembersWhatItAsksFor(t *testing.T) {
	rainbow := snapshot.LocationRef{Label: "Rainbow, CA", Tag: "RAINBOW", Lat: 33.41031, Lon: -117.14781}
	lp := &livePipelines{}
	lp.requestCard(rainbow, report.Everything(), 3) // no director: the remembering must not depend on one

	if _, ok := refFor(lp.resolvable, string(snapshot.Key(rainbow))); !ok {
		t.Error("requestCard did not record the ref it asked the Director to schedule")
	}
}

// AND THE SCHEDULE'S OWN COMPOSER USES IT — the wiring, not just the set.
//
// THE FIRST VERSION OF THIS FILE DID NOT CHECK THIS, and reverting
// `compose: composeFor(deck, resolvable)` back to `pool` left every test above
// green: they exercised `resolvable` directly and never asked what the schedule
// was handed. A fix nothing drives is a fix that can be undone silently — which
// is the lesson pool_test.go's own `cutTo` wiring test already records, found
// there by a plant that survived.
//
// A REAL DECK, BECAUSE A NIL ONE CANNOT TELL THE WIRINGS APART. `composeFor`
// refuses a nil deck BEFORE it resolves, so both wirings return "no audio deck"
// and the assertion would pass on either — my second draft did exactly that and
// its own control caught it.
func TestTheSchedulesComposerResolvesARequestedLocation(t *testing.T) {
	vista := snapshot.LocationRef{Label: "Vista, CA", Zip: "92084", Lat: 33.2, Lon: -117.24}
	rainbow := snapshot.LocationRef{Label: "Rainbow, CA", Tag: "RAINBOW", Lat: 33.41031, Lon: -117.14781}

	lp := &livePipelines{poolRefs: []snapshot.LocationRef{vista}}
	lp.rememberRequested(rainbow)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deck, _ := offlineDeck(t)
	tick := &tickerDeck{muted: &atomic.Bool{}, seen: loadSeen(t.TempDir(), time.Hour), alerts: newAlertStore()}
	s := startSchedule(ctx, testDirector(nil, func(tea.Msg) {}), nil,
		func() render.Clock { return render.Clock12 }, deck,
		lp.producer(), lp.resolvable, lp.currentWatch, tick, func(tea.Msg) {}, bedSeams{})
	if s == nil {
		t.Fatal("the schedule refused to start")
	}

	_, err := s.x.compose(ctx, string(snapshot.Key(rainbow)), report.Everything())
	if err != nil && strings.Contains(err.Error(), "no watched location") {
		t.Errorf("the schedule's composer cannot resolve a requested location: %v — "+
			"the card builds to nothing, the decline is routed, and the operator is shown "+
			"an action nothing took (FR-3.3)", err)
	}

	// THE CONTROL: the pool-only wiring must fail, or the assertion above is
	// true of any wiring at all.
	poolOnly := composeFor(deck, lp.currentPool)
	if _, err := poolOnly(ctx, string(snapshot.Key(rainbow)), report.Everything()); err == nil ||
		!strings.Contains(err.Error(), "no watched location") {
		t.Errorf("the pool-only composer resolved a location the pool does not hold (%v); "+
			"this control is what proves the check above discriminates", err)
	}
}
