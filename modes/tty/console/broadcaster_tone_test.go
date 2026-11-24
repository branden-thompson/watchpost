package console

// broadcaster_tone_test.go — what a card is painted on
// (0.16.0 D-86).
//
// HUM LEAD, 2026-09-11: "Main Track Cards should have some color BKGs that use
// tokens so it can be themeable like Observer … tint them based on report type
// and card origin: Operator requested cards get a slightly different Tint than
// Producer created cards. Alert cards should be color coded to match the most
// severe alert based on the [w] category bkgs in Observer (they should match)."

import (
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/category"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

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

	mine := cardTone(ttytest.Toned("b", lineup.LocationReport, lineup.FromOperator))
	theirs := cardTone(ttytest.Toned("a", lineup.LocationReport, lineup.FromDirector))

	if mine == theirs {
		t.Fatalf("the operator's card and the station's are painted the same: %q", mine)
	}
	for name, want := range map[string]render.Token{"the station's": render.CardBG, "the operator's": render.CardOperatorBG} {
		got := theirs
		if want == render.CardOperatorBG {
			got = mine
		}
		if !ttytest.ContainsToken(got, render.Tok(want)) {
			t.Errorf("%s card is %q; it does not carry %s", name, got, want)
		}
	}
	// AND BOTH CARRY THE CONSOLE'S OWN TEXT TONE, which is what keeps the AA
	// lift inside the console — registering the base text tone against these
	// grounds would move it in two themes and take Observer's tables with it.
	for _, got := range []string{mine, theirs} {
		if !ttytest.ContainsToken(got, render.Tok(render.CardText)) {
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
		got := cardTone(ttytest.Toned("burst", lineup.BreakingAlert, lineup.FromDirector, k))
		if want := render.Tok(category.Of(k).Tint); !ttytest.ContainsToken(got, want) {
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

	mixed := ttytest.Toned("burst", lineup.BreakingAlert, lineup.FromDirector,
		category.Watches, category.Emergency, category.Advisories)

	got := cardTone(mixed)
	if want := render.Tok(category.Of(category.Emergency).Tint); !ttytest.ContainsToken(got, want) {
		t.Errorf("a burst carrying an Emergency is painted %q, want the Emergency tint %q", got, want)
	}
	if soft := render.Tok(category.Of(category.Watches).Tint); ttytest.ContainsToken(got, soft) {
		t.Errorf("the burst was painted by the Watch it happens to list first: %q", got)
	}
}

// A HAZARD WITH NOTHING TO READ A CATEGORY FROM IS NOT GUESSED AT. Painting it
// the wrong severity is worse than painting it none.
func TestABurstWithNoArrivalsIsNotGivenASeverity(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	got := cardTone(ttytest.Toned("burst", lineup.BreakingAlert, lineup.FromDirector))
	if want := render.Tok(render.CardBG); !ttytest.ContainsToken(got, want) {
		t.Errorf("a burst with no arrivals is painted %q, want the ordinary card ground %q", got, want)
	}
	for _, k := range []category.Category{category.Emergency, category.Warnings} {
		if tint := render.Tok(category.Of(k).Tint); ttytest.ContainsToken(got, tint) {
			t.Errorf("a burst with no arrivals was given %v's colour: %q", k, got)
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
	if got := b.CardWindowGroundFor(1); got != "" {
		t.Errorf("a location report's window asked for the ground %q; it takes the standard tone", got)
	}

	// AND THE CASE THAT ACTUALLY DISCRIMINATES. The fixture above cannot: a
	// location report has no arrivals, so `worstCategory` refuses it and the
	// ground comes back "" whether or not the SLOT is checked — mutant mAV2,
	// the slot guard deleted, passes it.
	//
	// A NON-HAZARD CARD CARRYING ARRIVALS is the state that separates the two
	// guards. The app does not produce one today; the rule is "only a HAZARD
	// carries its ground in", and a rule tested only where a second condition
	// happens to agree with it is not tested.
	report := ttytest.Toned("rep", lineup.LocationReport, lineup.FromDirector, category.Warnings)
	if got := cardWindowGround(report); got != "" {
		t.Errorf("a LocationReport carrying arrivals asked for the ground %q; only a hazard does", got)
	}
}

// AN ALERT CARD AND ITS WINDOW ARE ONE COLOUR, the card's half (0.19.6 D-53):
// the takeover box is painted with the [w] window's tint for its category, the
// ground the Router's test sees its window float on, so one keypress does not
// change the colour of one hazard.
func TestAnAlertCardCarriesItsCategorysTint(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })

	for _, k := range []category.Category{category.Emergency, category.Warnings, category.Watches} {
		burst := ttytest.Admitted(t, ttytest.Toned("burst", lineup.BreakingAlert, lineup.FromDirector, k))
		want := render.Tok(category.Of(k).Tint)
		if got := cardTone(burst); !ttytest.ContainsToken(got, want) {
			t.Errorf("%v: the card is painted %q and the window %q", k, got, want)
		}
	}
}
