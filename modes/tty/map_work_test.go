package tty

// map_work_test.go — 0.18.0 W2.2 (library calls on the Bubble Tea goroutine,
// Work in a command; the NextCall tick) and W2.6 (every map worker joined on
// close, FR-8.2).

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// runCmdElsewhere runs a command on a goroutine of its own, as Bubble Tea
// does, and returns its message.
func runCmdElsewhere(cmd tea.Cmd) tea.Msg {
	out := make(chan tea.Msg, 1)
	go func() { out <- cmd() }()
	return <-out
}

// TestEveryCallButWorkIsOnTheUpdateGoroutine is W2.2 (C-7, FR-3.3): through
// the wrapper's record, every library call but Work ran on the goroutine that
// drives Update, and Work ran on another.
func TestEveryCallButWorkIsOnTheUpdateGoroutine(t *testing.T) {
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false)})
	where := &[]callSite{}
	d.mapPane.where = where
	d, _ = pressKey(d, "g")
	if feed := d.mapFeedCmd(); feed != nil {
		m, _ := d.Update(runCmdElsewhere(feed))
		d = m.(Dashboard)
	}
	for work := d.mapWorkCmd(); work != nil; {
		m, next := d.Update(runCmdElsewhere(work))
		d, work = m.(Dashboard), next
	}
	here := goroutineID()
	works := 0
	for _, c := range *where {
		switch {
		case c.name == "Work":
			works++
			if c.goroutine == here {
				t.Error("Work ran on the Update goroutine")
			}
		case c.goroutine != here:
			t.Errorf("%s ran on goroutine %d, not the Update goroutine %d", c.name, c.goroutine, here)
		}
	}
	if works == 0 || len(*where) < 5 {
		t.Fatalf("the record holds %d calls and %d Work: this measures nothing", len(*where), works)
	}
}

// TestTheMapAsksToBeDrawnWhenItSaysSo is W2.2's tick: while the window is
// open, Update keeps one tick outstanding at the library's NextCall; that
// tick draws, a tick for a time no longer wanted is dropped, and a closed
// window keeps none.
func TestTheMapAsksToBeDrawnWhenItSaysSo(t *testing.T) {
	now := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	stale := func(context.Context, MapAsk) MapFeed { // an overlay that goes stale ten minutes from now
		f := boxFeed(-117.6, -117.1, false)(context.Background(), MapAsk{})
		f.Overlays[0].Valid, f.Overlays[0].Keeps = now.Add(-50*time.Minute), time.Hour
		return f
	}
	d := mapDash(t, Config{MapFeed: stale})
	d.now = func() time.Time { return now }
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	at, ok := d.mapPane.m.NextCall(now)
	if !ok {
		t.Fatal("the library wants no call: this measures nothing")
	}
	if !d.mapPane.tickAt.Equal(at) {
		t.Fatalf("the outstanding tick is for %v, the library wants %v", d.mapPane.tickAt, at)
	}
	if _, again := d.armMapTick(); again != nil {
		t.Error("a tick already outstanding was armed again")
	}
	gen := d.mapPane.gen
	m, _ := d.Update(mapTickMsg{at: at.Add(-time.Minute)})
	if m.(Dashboard).mapPane.gen != gen {
		t.Error("a tick for a time no longer wanted drew")
	}
	m, _ = d.Update(mapTickMsg{at: at})
	if m.(Dashboard).mapPane.gen == gen {
		t.Error("the outstanding tick did not draw")
	}
	d, _ = pressKey(m.(Dashboard), "g") // closed
	if !d.mapPane.tickAt.IsZero() {
		t.Errorf("a closed window keeps a tick for %v", d.mapPane.tickAt)
	}
	if _, cmd := d.Update(mapTickMsg{at: at}); cmd != nil {
		t.Error("a tick to a closed window scheduled more")
	}
}

// TestClosingTheMapJoinsItsWorkers is W2.6 (FR-8.2): closing the map cancels
// the commands still running for it and waits for them; a command that
// starts after the close touches nothing.
func TestClosingTheMapJoinsItsWorkers(t *testing.T) {
	var started, finished atomic.Int32
	blocking := func(ctx context.Context, _ MapAsk) MapFeed {
		started.Add(1)
		<-ctx.Done() // held until the close cancels it
		finished.Add(1)
		return MapFeed{}
	}
	d := mapDash(t, Config{MapFeed: blocking})
	d, _ = pressKey(d, "g")
	feed := d.mapFeedCmd()
	go feed()
	for started.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	calls := &[]string{}
	d.mapPane.calls = calls
	d.closeMap()
	if finished.Load() != 1 {
		t.Fatal("the close returned with a worker still running")
	}
	late := d.mapFeedCmd()
	if late == nil {
		late = func() tea.Msg { return nil }
	}
	late()
	if started.Load() != 1 {
		t.Error("a command started after the close asked the app for the feed")
	}
	if strings.Join(*calls, " ") != "Close" {
		t.Errorf("the close made the calls %v, want Close alone", *calls)
	}
}

// TestTheRouterClosesObserversMap: the app closes the map through the model
// the program ends with.
func TestTheRouterClosesObserversMap(t *testing.T) {
	d := mapDash(t, Config{})
	d, _ = pressKey(d, "g")
	r := NewRouter(d)
	r.CloseMap()
	if err := d.mapPane.m.Recentre(tuimaps.LonLat{Lon: -100, Lat: 40}); err == nil {
		t.Error("the map still moves after the router closed it")
	}
}
