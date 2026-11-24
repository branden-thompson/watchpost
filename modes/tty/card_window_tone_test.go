package tty

// card_window_tone_test.go — a hazard's window wears its card's ground, and follows
// the card when the burst changes (D-127, D-126).

import (
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty/console"
	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// TestTheAlertWindowWearsTheCardsGround.
//
// HUM LEAD, UAT 2026-09-13: "I'm talking about the *modal* that is rendered when
// you press shift+a … So if the Card on the layout is the [w] orange - when I
// press <shift+a> that modal should MATCH the tone, not be the blue that is
// currently is."
//
// THE CARD AND ITS WINDOW ARE ONE COLOUR. `cardTone` paints the takeover box
// with the [w] window's category tint, and `modalCard` floats on that same
// ground rather than the standard modal one, so opening the hazard keeps its
// severity.
//
// ASKED OF THE REGISTRY, NOT OF A LITERAL, which is the rule the card's own tint
// already follows: `category.Of(k).Tint` IS what [w] paints that category, so the
// three surfaces agree by construction. That the card itself wears the same
// tint is the console's own tone tests' (0.19.6 D-53).
func TestTheAlertWindowWearsTheCardsGround(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	for _, k := range []category.Category{category.Emergency, category.Warnings, category.Watches} {
		burst := ttytest.Admitted(t, ttytest.Toned("burst", lineup.BreakingAlert, lineup.FromDirector, k))
		l, err := lineup.Lineup{}.Queue(lineup.AlertRail, burst)
		if err != nil {
			t.Fatalf("seeding the rail: %v", err)
		}
		b := consoleOf(t, console.Opening{ASCII: true}, l)

		r := Router{observer: Dashboard{}, broadcaster: b, active: SurfaceBroadcaster, keys: broadcasterKeyMap()}
		out := press(t, r, "A")
		if out.observer.modal != modalCard {
			t.Fatalf("%v: [A] did not open the card window", k)
		}
		// ASSERTED ON THE RENDERED WINDOW, NOT ON THE FIELD. A check of
		// `out.observer.cardGround` — the value STORED — passes against a build
		// where `modalCard` ignores it entirely. A stored ground is not a painted
		// one, and the operator sees the paint.
		want := render.Tok(category.Of(k).Tint)
		win := out.observer.renderModal(out.observer.opts())
		if !ttytest.ContainsToken(win, want) {
			t.Errorf("%v: the window does not paint the card's ground %q", k, want)
		}
		if standard := render.Tok(render.ModalBGDark); ttytest.ContainsToken(win, standard) && want != standard {
			t.Errorf("%v: the window still carries the standard modal ground %q", k, standard)
		}
	}
}

// TestTheOpenHazardWindowRefreshes.
//
// `refreshCardWindow` searches the ALERT RAIL as well as the MAIN TRACK, because
// `[A]` opens a card there (D-126). A window that never refreshes goes stale
// exactly where staleness matters most: a burst gains hazards while the
// operator is reading it, and the window goes on showing the old list.
func TestTheOpenHazardWindowRefreshes(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	burst := ttytest.Admitted(t, ttytest.Toned("burst", lineup.BreakingAlert, lineup.FromDirector, category.Watches))
	l, err := lineup.Lineup{}.Queue(lineup.AlertRail, burst)
	if err != nil {
		t.Fatalf("seeding the rail: %v", err)
	}
	b := consoleOf(t, console.Opening{ASCII: true}, l)

	r := Router{observer: Dashboard{}, broadcaster: b, active: SurfaceBroadcaster, keys: broadcasterKeyMap()}
	r = press(t, r, "A")
	if r.observer.modal != modalCard {
		t.Fatal("[A] did not open the hazard's window")
	}
	gen := r.observer.cardGen

	// THE BURST GAINS AN EMERGENCY while the operator is reading it — the whole
	// reason the window has to follow the card.
	worse := ttytest.Admitted(t, ttytest.Toned("burst", lineup.BreakingAlert, lineup.FromDirector,
		category.Watches, category.Emergency))
	l2, err := lineup.Lineup{}.Queue(lineup.AlertRail, worse)
	if err != nil {
		t.Fatalf("re-seeding the rail: %v", err)
	}
	r.broadcaster, _ = r.broadcaster.Update(console.LineupMsg{Lineup: l2})
	r = r.refreshCardWindow()

	if r.observer.cardGen == gen {
		t.Error("the burst changed and the open window did not follow it")
	}
	if want := render.Tok(category.Of(category.Emergency).Tint); r.observer.cardGround != want {
		t.Errorf("the refreshed window floats on %q, the worse burst wears %q",
			r.observer.cardGround, want)
	}
}
