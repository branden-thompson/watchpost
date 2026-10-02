package tty

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// timings is a Config.Timed that keeps what it is told.
type timings struct{ got []Timing }

func (tm *timings) record(t Timing) { tm.got = append(tm.got, t) }

// events is the events said after trigger, in order.
func (tm *timings) events(trigger string) []string {
	var out []string
	for _, t := range tm.got {
		if t.Trigger == trigger {
			out = append(out, t.Event)
		}
	}
	return out
}

// countOf is how many times x is in xs.
func countOf(xs []string, x string) int {
	n := 0
	for _, v := range xs {
		if v == x {
			n++
		}
	}
	return n
}

// ticking is a clock that moves a millisecond each time it is read, so every
// interval the instrument reports is measured, never zero.
func ticking() func() time.Time {
	at := time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC)
	return func() time.Time { at = at.Add(time.Millisecond); return at }
}

// openTimedMap is openRadarMap with the instrument on, then the lower 48 -
// the view the embedded tiles draw whole, so it reaches Complete with no tile
// host; zoomed in on the place, they would stay a stand-in.
func openTimedMap(t *testing.T, tm *timings) Dashboard {
	t.Helper()
	var timed func(Timing)
	if tm != nil {
		timed = tm.record
	}
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: radarFeed(t, "MRMS", new([]string)), Timed: timed,
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: true}}})
	d.now = ticking()
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	d = settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
	m, cmd = d.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	d = settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
	// The move's settle tick, as the real 600 ms timer delivers it: the view
	// stood still, so its alerts, radar and temperature are asked (D-66).
	m, cmd = d.Update(mapViewSettledMsg{gen: d.mapPane.viewGen})
	return settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
}

// TestTheMapTimesItsAsks is W14's instrument (D-154) for M5 and W8.12: from
// the ask - g, a map key - each answer as it lands, the first complete frame
// (M5's stop, "not the first frame") and the moment nothing is left to come,
// each once, in that order, each measured.
func TestTheMapTimesItsAsks(t *testing.T) {
	tm := &timings{}
	openTimedMap(t, tm)
	if got := strings.Join(tm.events("open"), ","); !strings.Contains(got, "answered:feed") || !strings.Contains(got, "answered:radar") {
		t.Errorf("opening did not time its answers: %s", got)
	}
	got := strings.Join(tm.events("map.region.1"), ",")
	// Each moment once per ask; answers as many as were asked.
	for _, want := range []string{"complete", "m5", "settled"} {
		if n := countOf(tm.events("map.region.1"), want); n != 1 {
			t.Errorf("the region key said %q %d times, want once: %s", want, n, got)
		}
	}
	// M5 is D-46's: the first complete frame holding every alert's area - so
	// after the moved view's own alerts land, which its settle tick asks.
	if strings.LastIndex(got, "answered:feed") > strings.Index(got, "m5") || !strings.HasSuffix(got, "settled") {
		t.Errorf("m5 comes after the view's alerts land, and settled is last: %s", got)
	}
	for _, e := range tm.got {
		if e.After <= 0 {
			t.Errorf("%s/%s measured %v: every interval is measured", e.Trigger, e.Event, e.After)
		}
	}
}

// TestASwitchIsTimedFromItsPress: an overlay switched on starts its own clock,
// and its answer and its settled frame are said against it.
func TestASwitchIsTimedFromItsPress(t *testing.T) {
	tm := &timings{}
	d := openTimedMap(t, tm)
	d.mapPane.clock.from = time.Time{} // the opening is done with
	d = d.timeFrom("overlay")
	d = d.renderMap()
	if got := strings.Join(tm.events("overlay"), ","); got != "complete,settled" {
		t.Errorf("a switch drawn from what is held says %q; want complete,settled - no m5, no alerts were asked", got)
	}
}

// TestTheMapClockSaysHowLateATickLanded is M6's first half: a tick the library
// asked for at a time, delivered after it, says by how much.
func TestTheMapClockSaysHowLateATickLanded(t *testing.T) {
	tm := &timings{}
	d := openTimedMap(t, tm)
	now := time.Date(2026, 8, 24, 2, 0, 0, 0, time.UTC)
	d.now = func() time.Time { return now }
	at := now.Add(-40 * time.Millisecond)
	d.mapPane.tickAt = at // the tick the map armed
	d.applyMapTick(mapTickMsg{at: at})
	var late []time.Duration
	for _, e := range tm.got {
		if e.Event == "tick" {
			late = append(late, e.After)
		}
	}
	if len(late) != 1 || late[0] != 40*time.Millisecond {
		t.Errorf("a tick 40 ms late said %v", late)
	}
}

// TestAKeyIsTimedToItsFrame is M6's second half: a key, from its arrival to
// the frame it produced, said once - a frame with no key before it says
// nothing.
func TestAKeyIsTimedToItsFrame(t *testing.T) {
	tm := &timings{}
	d := goldenDash(t, false)
	d.cfg.Timed = tm.record
	r := NewRouter(d)
	m, _ := r.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	r = m.(Router)
	_ = r.View()
	_ = r.View()
	keys := 0
	for _, e := range tm.got {
		if e.Event == "key" {
			keys++
		}
	}
	if keys != 1 {
		t.Errorf("one key and two frames said %d key timings, want 1", keys)
	}
}

// TestTheInstrumentChangesNothingItMeasures: the same opening with the
// instrument on and off draws the same frame - it only listens.
func TestTheInstrumentChangesNothingItMeasures(t *testing.T) {
	on, off := openTimedMap(t, &timings{}), openTimedMap(t, nil)
	if a, b := on.View().Content, off.View().Content; a != b {
		t.Error("the instrument changed the frame it measured")
	}
}

// TestAMoveIsNotSettledBeforeItsViewIsAsked (W14): a pan that drew the map
// before marking the view moved would say "settled" at once - before the pan's
// own alerts are asked, which its settle tick does 600 ms later (D-66).
func TestAMoveIsNotSettledBeforeItsViewIsAsked(t *testing.T) {
	tm := &timings{}
	d := openTimedMap(t, tm)
	m, cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	d = settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
	if n := countOf(tm.events("map.pan.right"), "settled"); n != 0 {
		t.Fatalf("a pan was settled %d times before its settle tick asked for its view", n)
	}
	m, cmd = d.Update(mapViewSettledMsg{gen: d.mapPane.viewGen})
	settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
	if n := countOf(tm.events("map.pan.right"), "settled"); n != 1 {
		t.Errorf("once its view is asked and in, the pan is settled once; said %d", n)
	}
}

// TestAnAskStopsListeningOnceSettled: a trigger that outlived its ask (space,
// for the loop) would time every later refresh from the key press - 300 s
// "answers". An answer is timed once a kind, and nothing
// after the ask is settled.
func TestAnAskStopsListeningOnceSettled(t *testing.T) {
	tm := &timings{}
	d := openTimedMap(t, tm)
	settledAt := len(tm.got)
	for range 3 { // three refreshes, as new snapshots bring
		d = d.requestFeed()
		d = feedAndSettle(t, d)
	}
	_ = d.timed("answered:temp") // a kind not yet seen, landing after the ask settled
	for _, e := range tm.got[settledAt:] {
		if e.Trigger == "map.region.1" {
			t.Errorf("after settled, a later refresh was timed against the ask: %+v", e)
		}
	}
	for _, trigger := range []string{"open", "map.region.1"} {
		for _, kind := range []string{"answered:feed", "answered:radar"} {
			if n := countOf(tm.events(trigger), kind); n > 1 {
				t.Errorf("%s timed %s %d times; an answer is timed once a kind", trigger, kind, n)
			}
		}
	}
}

// TestM5IsReachedUnderAStreamOfData is D-157 in the instrument: with new data
// landing while the feed's ask runs, the answer drawn is one data generation
// behind the newest - and it still holds the alerts asked since the trigger,
// so it is M5. Settled waits for the re-ask the new data wanted.
func TestM5IsReachedUnderAStreamOfData(t *testing.T) {
	tm := &timings{}
	d := openTimedMap(t, tm) // the lower 48: the view the embedded tiles draw whole
	d = d.timeFrom("probe")
	d, ask := d.requestFeed().askFeed()
	m, _ := d.Update(SnapshotMsg{Snap: placedSnap()}) // new data while the ask runs
	d = m.(Dashboard)
	if !d.mapPane.feedAgain {
		t.Fatal("control: the new data did not mark the feed wanted again")
	}
	answers := feedAsks(t, ask)
	if len(answers) != 1 {
		t.Fatalf("the ask answered %d times", len(answers))
	}
	m, _ = d.Update(answers[0])
	settleMap(t, m.(Dashboard))
	got := tm.events("probe")
	if countOf(got, "complete") != 1 {
		t.Fatalf("control: the frame never completed, so this proves nothing: %v", got)
	}
	if countOf(got, "m5") != 1 {
		t.Errorf("the complete frame holding the alerts asked since the trigger was not M5: %v", got)
	}
	if countOf(got, "settled") != 0 {
		t.Errorf("settled with a re-ask still wanted: %v", got)
	}
}

// TestAWarmReopenReachesM5 is the sessions' warm opens (W14): a pan whose
// settle tick lands while the map is closed is dropped, so the reopened map
// would never be "still" and M5 never said - but opening the map asks for its
// view afresh. The open is the view's ask.
func TestAWarmReopenReachesM5(t *testing.T) {
	tm := &timings{}
	d := openTimedMap(t, tm)
	m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyRight}) // a pan...
	d = m.(Dashboard)
	d, _ = pressKey(d, "g")                                    // ...and the map closed before it settles
	m, _ = d.Update(mapViewSettledMsg{gen: d.mapPane.viewGen}) // its tick, dropped: the map is closed
	d = m.(Dashboard)
	if d.mapPane.viewAsked == d.mapPane.viewGen {
		t.Fatal("control: the dropped tick left nothing unasked, so this proves nothing")
	}
	m, _ = d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"}) // reopened: the open asks for its view
	d = m.(Dashboard)
	if d.mapPane.viewAsked != d.mapPane.viewGen {
		t.Error("the reopened map asked for its view, but it is not counted still - M5 can never be said")
	}
}

// TestTheFrameRateIsMeasured is W14's next measure: with the map open the
// station costs ~7-8 % of a core (the sampler's cumulative CPU; the Go profiler
// undercounted it ~15x on this machine), and redrawing the frame - the
// terminal's width and grapheme work on every line - was the largest part a
// native sampler saw. How often the frame is drawn is the number that
// decides it: every hundredth frame says how long the hundred took.
func TestTheFrameRateIsMeasured(t *testing.T) {
	tm := &timings{}
	d := goldenDash(t, false)
	d.cfg.Timed = tm.record
	r := NewRouter(d)
	for range 250 {
		_ = r.View()
	}
	n := 0
	for _, e := range tm.got {
		if e.Event == "frames100" {
			n++
			if e.After <= 0 {
				t.Errorf("a hundred frames took %v", e.After)
			}
		}
	}
	if n != 2 {
		t.Errorf("250 frames said %d hundreds, want 2", n)
	}
}
