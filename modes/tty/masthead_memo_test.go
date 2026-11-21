package tty

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
)

// TestTheMastheadKeyCoversEverythingTheMastheadShows is the masthead memo's
// completeness guard on Observer (0.19.5 Q4.2, A-15): if two models would draw
// different mastheads, their keys differ, for every field the walk can change.
// The console's half is broadcaster_masthead_memo_test.go's, and the inputs no
// field walk reaches are the kit's, kit_masthead_test.go (0.19.6 D-58).
func TestTheMastheadKeyCoversEverythingTheMastheadShows(t *testing.T) {
	obs := benchDash(t, 133, 44).(Dashboard)
	at := DataAsOf(obs.snap).Add(20 * time.Second)
	obs.now = func() time.Time { return at }
	if obs.snap == nil {
		t.Fatal("the fixture has no snapshot; its stamp and API summary would check nothing")
	}
	observer := func(d Dashboard) (string, any) {
		o := d.opts()
		bare := d
		bare.memo = nil
		return bare.header(o), MastheadKeyOf(o, render.EditionObserver, d.cfg.Version, d.snap, d.clock())
	}
	reached := 0
	before, k0 := observer(obs)
	for _, p := range perturbations(t, obs) {
		after, k1 := observer(p.model)
		if MastheadMoved(t, "Observer", p.field, before, after, k0, k1) {
			reached++
		}
	}
	if reached == 0 {
		t.Error("no field the walk changes reached Observer's masthead: the fixture stopped exercising it")
	}
}
