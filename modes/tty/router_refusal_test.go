package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// A REFUSED SWAP SAYS WHY, ON THE SURFACE THE OPERATOR IS LOOKING AT.
//
// THE UAT DEFECT (HUM LEAD, 2026-09-10): "ctrl+o will soft lock randomly — so
// ctrl+o -> ctrl+b -> ctrl+o (doesn't work the 2nd time) … While radio is on
// standby I should flip back and forth easily."
//
// It is not a lock and it is not random. Visiting Observer TUNES, and a tune
// tells the Director the programme is RUNNING (mM3's own fix) — so the station
// the operator left STOPPED is ON AIR when they come back, and FR-1.4 refuses
// to let them leave a live console. The gate was right. What was missing is that
// `Router.refusal` was RECORDED AND NEVER DRAWN, under a comment saying exactly
// why that must not happen: "a refusal they cannot read is indistinguishable
// from a broken control."
func TestARefusedSwapIsShownToTheOperator(t *testing.T) {
	d, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	r := NewRouter(d)
	m, _ := r.Update(keyPress(t, "ctrl+b"))
	r = m.(Router)
	r.broadcaster.width, r.broadcaster.height = 150, 50

	// What the Observer's own tune publishes, without the operator asking.
	m, _ = r.Update(StationMsg{Power: lineup.Running})
	r = m.(Router)

	m, _ = r.Update(keyPress(t, "ctrl+o"))
	r = m.(Router)
	if r.active != SurfaceBroadcaster {
		t.Fatal("precondition: a live station refuses the swap (FR-1.4)")
	}
	frame := stripANSITest(r.View().Content)
	if !strings.Contains(frame, "STANDBY before leaving") {
		t.Errorf("the refusal is not on the frame the operator is looking at:\n%s", strings.Join(strings.Split(frame, "\n")[:9], "\n"))
	}

	// AND IT GOES WHEN THE REASON GOES. A message that outlived the state it
	// describes would be its own defect one move along.
	m, _ = r.Update(StationMsg{Power: lineup.OffAir})
	r = m.(Router)
	if strings.Contains(stripANSITest(r.View().Content), "STANDBY before leaving") {
		t.Error("the refusal outlived the state it describes")
	}
	m, _ = r.Update(keyPress(t, "ctrl+o"))
	if m.(Router).active != SurfaceObserver {
		t.Error("and a station off the air lets the operator leave")
	}
}

// THE SWAP TELLS THE APP WHICH SURFACE OWNS THE AIR (D-73).
//
// THE ALERT RAIL IS SCOPED TO IT — the listener's filter on Observer, the
// station's service area on the console — and `swapTo` is the ONE place a swap
// is granted, so it is the one place that can say so.
//
// IT IS ASSERTED THROUGH THE KEYS, not by calling `swapTo`: the defect this
// release keeps producing is a seam nothing drives, and a test that called the
// method directly would pass over an unbound control.
func TestASwapTellsTheAppWhoOwnsTheAir(t *testing.T) {
	d, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	var told []Surface
	d.cfg.OnSurface = func(s Surface) { told = append(told, s) }
	var m tea.Model = NewRouter(d)

	m, _ = m.Update(keyPress(t, "ctrl+b"))
	m, _ = m.Update(keyPress(t, "ctrl+o"))
	if len(told) != 2 || told[0] != SurfaceBroadcaster || told[1] != SurfaceObserver {
		t.Fatalf("both swaps are announced, in order; got %v", told)
	}

	// A REFUSED SWAP ANNOUNCES NOTHING, because nothing moved. Telling the app
	// the console owns the air while the operator is still on Observer would
	// scope their alerts to a station they did not reach.
	r := m.(Router)
	r.broadcaster.power = lineup.Running
	r.active = SurfaceBroadcaster
	told = nil
	m, _ = r.Update(keyPress(t, "ctrl+o"))
	if m.(Router).active != SurfaceBroadcaster {
		t.Fatal("precondition: a live station refuses the swap")
	}
	if len(told) != 0 {
		t.Errorf("a refused swap moved the air: %v", told)
	}
}
