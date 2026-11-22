package tty

import (
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/modes/tty/internal/frame"
	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/render"
)

// TestTheConsolesMastheadKeyCoversEverythingItsMastheadShows is the masthead
// memo's completeness guard on the console (0.19.5 Q4.2, A-15; 0.19.6 D-58): if
// two consoles would draw different mastheads, their keys differ, for every
// field the walk can change. Observer's half and the kit's named inputs are
// their own tests.
func TestTheConsolesMastheadKeyCoversEverythingItsMastheadShows(t *testing.T) {
	obs := benchDash(t, 133, 44).(Dashboard)
	at := frame.DataAsOf(obs.snap).Add(20 * time.Second)
	con := loadedConsoleAt(t, ttytest.LoadedPoolSize, at)
	con.now = func() time.Time { return at }
	con.width, con.height = 160, 74
	con.snap = obs.snap // the console's stamp and summary read the same snapshot Observer does (D-59)
	if con.snap == nil {
		t.Fatal("the fixture has no snapshot; its stamp and API summary would check nothing")
	}
	console := func(b Broadcaster) (string, any) {
		o := b.opts()
		bare := b
		bare.memo = nil
		return bare.header(o), frame.MastheadKeyOf(o, render.EditionBroadcaster, b.version, b.snap, b.clock())
	}
	reached := 0
	before, k0 := console(con)
	ttytest.PerturbEach(t, con, consoleExcuse, consoleWriter, func(field string, next Broadcaster) {
		after, k1 := console(next)
		if ttytest.MastheadMoved(t, "the console", field, before, after, k0, k1) {
			reached++
		}
	})
	if reached == 0 {
		t.Error("no field the walk changes reached the console's masthead: the fixture stopped exercising it")
	}
}
