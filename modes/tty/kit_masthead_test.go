package tty

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// mastheadInputs are the masthead's inputs no field walk reaches, as named
// perturbations of a model: the clock (the age words, then staleness), the
// snapshot's providers (their health, and the as-of instant), the theme and
// the colour switch, which are process-wide.
type mastheadInput struct {
	name  string
	apply func(snap **snapshot.Snapshot, now *time.Time) (undo func())
}

var mastheadInputs = []mastheadInput{
	{"the clock, two minutes on", func(_ **snapshot.Snapshot, now *time.Time) func() { *now = now.Add(2 * time.Minute); return func() {} }},
	{"the clock, past staleAfter", func(_ **snapshot.Snapshot, now *time.Time) func() {
		*now = now.Add(staleAfter + 30*time.Second)
		return func() {}
	}},
	{"a provider's health", func(snap **snapshot.Snapshot, _ *time.Time) func() {
		next := **snap
		next.Providers = append([]snapshot.ProviderStatus(nil), next.Providers...)
		next.Providers[0].Status = snapshot.ProviderDegraded
		*snap = &next
		return func() {}
	}},
	{"a provider fetched later", func(snap **snapshot.Snapshot, _ *time.Time) func() {
		next := **snap
		next.Providers = append([]snapshot.ProviderStatus(nil), next.Providers...)
		next.Providers[0].FetchedAt = next.Providers[0].FetchedAt.Add(time.Minute)
		*snap = &next
		return func() {}
	}},
	{"the theme", func(_ **snapshot.Snapshot, _ *time.Time) func() {
		was := render.ThemeName()
		render.SetTheme("Watchpost Light")
		return func() { render.SetTheme(was) }
	}},
	{"the colour switch", func(_ **snapshot.Snapshot, _ *time.Time) func() {
		rendering.SetColorEnabledForTest(false)
		return func() { rendering.SetColorEnabledForTest(true) }
	}},
}

// TestTheMastheadKeyCoversItsNamedInputs is the masthead memo's guard on the
// inputs no field walk reaches (0.19.5 Q4.2, A-15; 0.19.6 D-58), for both
// editions: when one of them changes the masthead the kit draws, its title,
// stamp and API summary, MastheadKeyOf changes too.
func TestTheMastheadKeyCoversItsNamedInputs(t *testing.T) {
	// COLOUR ON, so the theme and the colour switch have something to change;
	// the switch's undo turns it back on for the inputs after it.
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(rendering.ResetColorEnabledForTest)
	base := MastheadSnap()
	at := DataAsOf(base).Add(20 * time.Second)
	o := render.Opts{Width: 133}
	drawn := func(edition string, snap *snapshot.Snapshot, now time.Time) (string, any) {
		rule := o.BoxRuleWidth()
		title := MastheadTitle(edition, "9.9.9", rule)
		return title + "\n" + MastheadStamp(o, title, snap, now, rule) + "\n" + APISummaryOf(o, snap),
			MastheadKeyOf(o, edition, "9.9.9", snap, now)
	}
	reached := 0
	for _, edition := range []string{render.EditionObserver, render.EditionBroadcaster} {
		for _, in := range mastheadInputs {
			// THE BASELINE IS TAKEN AGAIN FOR EACH INPUT: undoing the theme input
			// is a SetTheme, which moves the theme generation, so a baseline taken
			// once would differ from every later input by the theme alone.
			before, k0 := drawn(edition, base, at)
			snap, now := base, at
			undo := in.apply(&snap, &now)
			after, k1 := drawn(edition, snap, now)
			undo()
			if MastheadMoved(t, edition, in.name, before, after, k0, k1) {
				reached++
			}
		}
	}
	if reached < 2*len(mastheadInputs) {
		t.Errorf("only %d of the %d named inputs reached a masthead on the two editions: the fixture stopped exercising them", reached, 2*len(mastheadInputs))
	}
}
