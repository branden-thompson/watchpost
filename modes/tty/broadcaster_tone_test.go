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
