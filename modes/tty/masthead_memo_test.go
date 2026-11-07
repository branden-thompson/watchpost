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

// TestTheMastheadKeyCoversEverythingTheMastheadShows is the masthead memo's
// completeness guard (0.19.5 Q4.2, A-15), on both surfaces: if two models would
// draw different mastheads, their keys differ. Every field the walk can change
// on each model, and the named inputs above.
func TestTheMastheadKeyCoversEverythingTheMastheadShows(t *testing.T) {
	t.Cleanup(rendering.ResetColorEnabledForTest)
	obs := benchDash(t, 133, 44).(Dashboard)
	at := dataAsOf(obs.snap).Add(20 * time.Second)
	obs.now = func() time.Time { return at }
	con := loadedConsoleAt(t, loadedPoolSize, at)
	con.now = func() time.Time { return at }
	con.width, con.height = 160, 74
	con.snap = obs.snap // the console's stamp and summary read the same snapshot Observer does (D-59)
	if con.snap == nil || obs.snap == nil {
		t.Fatal("a fixture has no snapshot; its stamp and API summary would check nothing")
	}

	observer := func(d Dashboard) (string, mastheadKey) {
		o := d.opts()
		bare := d
		bare.memo = nil
		return bare.header(o), mastheadKeyOf(o, render.EditionObserver, d.cfg.Version, d.snap, d.clock())
	}
	console := func(b Broadcaster) (string, mastheadKey) {
		o := b.opts()
		bare := b
		bare.memo = nil
		return bare.header(o), mastheadKeyOf(o, render.EditionBroadcaster, b.version, b.snap, b.clock())
	}
	check := func(surface, field string, before, after string, k0, k1 mastheadKey) bool {
		if before == after {
			return false
		}
		if k0 == k1 {
			t.Errorf("%s: changing %s changes the masthead and NOT mastheadKey: the masthead memo (masthead.go, mastheadSlot) will draw the old masthead; add what %s changes to mastheadKeyOf", surface, field, field)
		}
		return true
	}

	reached := 0
	before, k0 := observer(obs)
	for _, p := range perturbations(t, obs) {
		after, k1 := observer(p.model)
		if check("Observer", p.field, before, after, k0, k1) {
			reached++
		}
	}
	cBefore, c0 := console(con)
	perturbEach(t, con, consoleExcuse, consoleWriter, func(field string, next Broadcaster) {
		after, k1 := console(next)
		if check("the console", field, cBefore, after, c0, k1) {
			reached++
		}
	})
	for _, in := range mastheadInputs {
		// THE BASELINE IS TAKEN AGAIN FOR EACH INPUT: undoing the theme input
		// is a SetTheme, which moves the theme generation, so a baseline taken
		// once would differ from every later input by the theme alone.
		before, k0 := observer(obs)
		cBefore, c0 := console(con)
		d, b := obs, con
		now := at
		undo := in.apply(&d.snap, &now)
		b.snap = d.snap
		d.now, b.now = func() time.Time { return now }, func() time.Time { return now }
		after, k1 := observer(d)
		cAfter, c1 := console(b)
		undo()
		if check("Observer", in.name, before, after, k0, k1) {
			reached++
		}
		if check("the console", in.name, cBefore, cAfter, c0, c1) {
			reached++
		}
	}
	if reached < 2*len(mastheadInputs) {
		t.Errorf("only %d perturbations reached a masthead, under the %d named ones on both surfaces: the fixtures stopped exercising it", reached, 2*len(mastheadInputs))
	}
}
