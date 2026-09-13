package tty

// broadcaster_tone_test.go — what a card is painted on (D-86).
//
// HUM LEAD, 2026-09-11: "Main Track Cards should have some color BKGs that use
// tokens so it can be themeable like Observer … tint them based on report type
// and card origin: Operator requested cards get a slightly different Tint than
// Producer created cards. Alert cards should be color coded to match the most
// severe alert based on the [w] category bkgs in Observer (they should match)."

import (
	"testing"

	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func toned(id string, slot lineup.Slot, origin lineup.Origin, cats ...category.Category) lineup.Card {
	c := lineup.Card{ID: id, Slot: slot, Origin: origin, Subject: id, Headline: id}
	for _, k := range cats {
		c.From = append(c.From, lineup.Arrival{ID: id, Category: k})
	}
	return c
}

// AN EMPTY SLOT IS NOT A CARD. The waiting placeholder and the LIVE slot on a
// station at rest carry no identity, and a ground there is a card that is not
// there.
func TestASlotWithNoCardIsNotPainted(t *testing.T) {
	if got := cardTone(lineup.Card{}); got != "" {
		t.Errorf("an empty slot was painted %q", got)
	}
	if got := cardTone(lineup.Card{Headline: "waiting for the line-up ..."}); got != "" {
		t.Errorf("the waiting placeholder was painted %q", got)
	}
}

// THE ORIGIN IS WHAT SEPARATES THEM, narrow by ruling: the station's own
// proposals on one ground, the operator's requests on another.
func TestAnOperatorsCardIsToldFromTheStationsOwn(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	mine := cardTone(toned("b", lineup.LocationReport, lineup.FromOperator))
	theirs := cardTone(toned("a", lineup.LocationReport, lineup.FromDirector))

	if mine == theirs {
		t.Fatalf("the operator's card and the station's are painted the same: %q", mine)
	}
	for name, want := range map[string]render.Token{"the station's": render.CardBG, "the operator's": render.CardOperatorBG} {
		got := theirs
		if want == render.CardOperatorBG {
			got = mine
		}
		if !contains(got, render.Tok(want)) {
			t.Errorf("%s card is %q; it does not carry %s", name, got, want)
		}
	}
	// AND BOTH CARRY THE CONSOLE'S OWN TEXT TONE, which is what keeps the AA
	// lift inside the console — registering the base text tone against these
	// grounds moved it in two themes and took Observer's tables with it.
	for _, got := range []string{mine, theirs} {
		if !contains(got, render.Tok(render.CardText)) {
			t.Errorf("%q does not carry the card's own text tone", got)
		}
	}
}

// AN ALERT CARD WEARS ITS CATEGORY, AND IT IS THE [w] WINDOW'S OWN TINT.
//
// ASKED OF THE REGISTRY, NOT OF A VALUE. `category.Of(k).Tint` IS the background
// the [w] window paints that category on, so "they should match" holds by
// construction — a literal here would let the two drift and still pass.
func TestAnAlertCardWearsTheSameTintAsTheEventsWindow(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	for _, k := range []category.Category{category.Emergency, category.Warnings, category.Watches, category.Marine} {
		got := cardTone(toned("burst", lineup.BreakingAlert, lineup.FromDirector, k))
		if want := render.Tok(category.Of(k).Tint); !contains(got, want) {
			t.Errorf("a %v burst is painted %q; the [w] window paints it %q", k, got, want)
		}
	}
}

// AND A BURST WEARS ITS WORST. A card reads several alerts (MVS-D-77) and the
// colour has to promise the most severe of them, or an Emergency inside a burst
// led by a Watch is drawn as a Watch.
func TestABurstIsPaintedByItsMostSevereAlert(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	mixed := toned("burst", lineup.BreakingAlert, lineup.FromDirector,
		category.Watches, category.Emergency, category.Advisories)

	got := cardTone(mixed)
	if want := render.Tok(category.Of(category.Emergency).Tint); !contains(got, want) {
		t.Errorf("a burst carrying an Emergency is painted %q, want the Emergency tint %q", got, want)
	}
	if soft := render.Tok(category.Of(category.Watches).Tint); contains(got, soft) {
		t.Errorf("the burst was painted by the Watch it happens to list first: %q", got)
	}
}

// A HAZARD WITH NOTHING TO READ A CATEGORY FROM IS NOT GUESSED AT. Painting it
// the wrong severity is worse than painting it none.
func TestABurstWithNoArrivalsIsNotGivenASeverity(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	got := cardTone(toned("burst", lineup.BreakingAlert, lineup.FromDirector))
	if want := render.Tok(render.CardBG); !contains(got, want) {
		t.Errorf("a burst with no arrivals is painted %q, want the ordinary card ground %q", got, want)
	}
	for _, k := range []category.Category{category.Emergency, category.Warnings} {
		if tint := render.Tok(category.Of(k).Tint); contains(got, tint) {
			t.Errorf("a burst with no arrivals was given %v's colour: %q", k, got)
		}
	}
}

func contains(s, sub string) bool { return sub != "" && len(s) >= len(sub) && indexOfSub(s, sub) >= 0 }

func indexOfSub(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// admitted puts a card through the states the schedule requires before it may be
// queued — the lineup holds admitted cards only, and a fixture that skips this
// trips the invariant rather than testing anything.
func admitted(t *testing.T, c lineup.Card) lineup.Card {
	t.Helper()
	c.State = lineup.Proposed
	out, err := lineup.Propose(c)
	if err != nil {
		t.Fatalf("proposing %s: %v", c.ID, err)
	}
	if out, err = out.To(lineup.Admitted); err != nil {
		t.Fatalf("admitting %s: %v", c.ID, err)
	}
	return out
}

// TestTheAlertWindowWearsTheCardsGround.
//
// HUM LEAD, UAT 2026-09-13: "I'm talking about the *modal* that is rendered when
// you press shift+a … So if the Card on the layout is the [w] orange - when I
// press <shift+a> that modal should MATCH the tone, not be the blue that is
// currently is."
//
// THE CARD AND ITS WINDOW WERE TWO COLOURS ONE KEYPRESS APART. `cardTone` painted
// the takeover box with the [w] window's category tint — correctly, and that rule
// stands — and `modalCard` floated on the standard modal ground, so opening the
// hazard threw its severity away.
//
// ASKED OF THE REGISTRY, NOT OF A LITERAL, which is the rule the card's own tint
// already follows: `category.Of(k).Tint` IS what [w] paints that category, so the
// three surfaces agree by construction.
func TestTheAlertWindowWearsTheCardsGround(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	for _, k := range []category.Category{category.Emergency, category.Warnings, category.Watches} {
		burst := admitted(t, toned("burst", lineup.BreakingAlert, lineup.FromDirector, k))
		b := bcWith(t)
		l, err := b.lineup.Queue(lineup.AlertRail, burst)
		if err != nil {
			t.Fatalf("seeding the rail: %v", err)
		}
		b, _ = b.Update(LineupMsg{Lineup: l})
		b.width, b.height, b.ascii = 150, 74, true

		r := Router{observer: Dashboard{}, broadcaster: b, active: SurfaceBroadcaster, keys: broadcasterKeyMap()}
		out := press(t, r, "A")
		if out.observer.modal != modalCard {
			t.Fatalf("%v: [A] did not open the card window", k)
		}
		// ASSERTED ON THE RENDERED WINDOW, NOT ON THE FIELD. The first version of
		// this checked `out.observer.cardGround` — the value STORED — and passed
		// against a build where `modalCard` ignored it entirely. A stored ground
		// is not a painted one, and the operator sees the paint.
		want := render.Tok(category.Of(k).Tint)
		win := out.observer.renderModal(out.observer.opts())
		if !contains(win, want) {
			t.Errorf("%v: the window does not paint the card's ground %q", k, want)
		}
		if standard := render.Tok(render.ModalBGDark); contains(win, standard) && want != standard {
			t.Errorf("%v: the window still carries the standard modal ground %q", k, standard)
		}
		// AND THE CARD IT CAME FROM AGREES, which is the whole ruling: one
		// keypress must not change the colour of one hazard.
		if got := cardTone(burst); !contains(got, want) {
			t.Errorf("%v: the card is painted %q and the window %q", k, got, want)
		}
	}
}

// AND AN ORDINARY REPORT'S WINDOW IS UNCHANGED.
//
// Only a hazard carries its ground in. A location report's ground is `CardBG` —
// not a severity, not a signal — and floating every other window on it would
// restyle the whole console to say nothing new.
func TestAReportsWindowKeepsTheStandardGround(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	b := broadcasterWithOneCard(t)
	if got := b.cardWindowGroundFor(1); got != "" {
		t.Errorf("a location report's window asked for the ground %q; it takes the standard tone", got)
	}

	// AND THE CASE THAT ACTUALLY DISCRIMINATES. The fixture above cannot: a
	// location report has no arrivals, so `worstCategory` refuses it and the
	// ground comes back "" whether or not the SLOT is checked — mutant mAV2
	// deleted the slot guard and this test passed anyway.
	//
	// A NON-HAZARD CARD CARRYING ARRIVALS is the state that separates the two
	// guards. The app does not produce one today; the rule is "only a HAZARD
	// carries its ground in", and a rule tested only where a second condition
	// happens to agree with it is not tested.
	report := toned("rep", lineup.LocationReport, lineup.FromDirector, category.Warnings)
	if got := cardWindowGround(report); got != "" {
		t.Errorf("a LocationReport carrying arrivals asked for the ground %q; only a hazard does", got)
	}
}

// TestTheOpenHazardWindowRefreshes.
//
// `refreshCardWindow` searched the MAIN TRACK alone — complete while a digit was
// the only way in, and incomplete the moment `[A]` opened a card on the ALERT
// RAIL (D-126). A window that never refreshes goes stale exactly where staleness
// matters most: a burst gains hazards while the operator is reading it, and the
// window goes on showing the old list.
func TestTheOpenHazardWindowRefreshes(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	burst := admitted(t, toned("burst", lineup.BreakingAlert, lineup.FromDirector, category.Watches))
	b := bcWith(t)
	l, err := b.lineup.Queue(lineup.AlertRail, burst)
	if err != nil {
		t.Fatalf("seeding the rail: %v", err)
	}
	b, _ = b.Update(LineupMsg{Lineup: l})
	b.width, b.height, b.ascii = 150, 74, true

	r := Router{observer: Dashboard{}, broadcaster: b, active: SurfaceBroadcaster, keys: broadcasterKeyMap()}
	r = press(t, r, "A")
	if r.observer.modal != modalCard {
		t.Fatal("[A] did not open the hazard's window")
	}
	gen := r.observer.cardGen

	// THE BURST GAINS AN EMERGENCY while the operator is reading it — the whole
	// reason the window has to follow the card.
	worse := admitted(t, toned("burst", lineup.BreakingAlert, lineup.FromDirector,
		category.Watches, category.Emergency))
	l2, err := lineup.Lineup{}.Queue(lineup.AlertRail, worse)
	if err != nil {
		t.Fatalf("re-seeding the rail: %v", err)
	}
	r.broadcaster, _ = r.broadcaster.Update(LineupMsg{Lineup: l2})
	r = r.refreshCardWindow()

	if r.observer.cardGen == gen {
		t.Error("the burst changed and the open window did not follow it")
	}
	if want := render.Tok(category.Of(category.Emergency).Tint); r.observer.cardGround != want {
		t.Errorf("the refreshed window floats on %q, the worse burst wears %q",
			r.observer.cardGround, want)
	}
}
