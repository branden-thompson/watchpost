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
