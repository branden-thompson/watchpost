package tty

// motion_test.go — F-180, F-192 (FR-7.2, D-360, A-107 (10)): one Reduce
// motion switch for the whole app. On, nothing moves on its own: the map's
// loops and Forecast mode do not play, the ticker steps item by item, the
// visualiser and the loading dots hold still, the radio line is cut, and a
// pressed chip stays marked until the next key; the console follows.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/render"
)

// TestReduceMotionIsASettingOffByDefault is D-360: a row on the Watchpost UI
// tab, off by default, saved, and read back from the file.
func TestReduceMotionIsASettingOffByDefault(t *testing.T) {
	d := dash(t).(Dashboard)
	if d.reduceMotion {
		t.Fatal("Reduce motion is on by default")
	}
	if tabOfGroup(setupTable()[rowReduceMotion].group) != tabUI {
		t.Error("the row is not on the Watchpost UI tab")
	}
	d.setup.focus = rowReduceMotion
	ui, _ := d.uiLines(d.opts())
	if text := stripANSITest(strings.Join(ui, "\n")); !strings.Contains(text, "Reduce motion -") || !strings.Contains(text, "Off") {
		t.Errorf("the row does not say Reduce motion, Off:\n%s", text)
	}
	if d = d.cycleReduceMotion(true); !d.reduceMotion || !d.uiForSave().ReduceMotion {
		t.Error("the picker did not turn it on, or the save does not carry it")
	}
	if !mapDash(t, Config{ReduceMotion: true}).reduceMotion {
		t.Error("a file saying reduce_motion opens with it off")
	}
}

// TestReduceMotionStillsTheMap is D-360: the library's ReduceMotion is on, so
// the radar's loop does not play and says why; Forecast mode does not play.
func TestReduceMotionStillsTheMap(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked).cycleReduceMotion(true)
	d = d.renderMap()
	m, _, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	d = m
	if st := d.mapPane.m.Loop(); st.Playing || st.Off != tuimaps.OffReduceMotion {
		t.Errorf("space under Reduce motion left the loop %+v", st)
	}
	if row := stripANSITest(strings.Join(d.scrubRows(d.mapTextW()), "\n")); !strings.Contains(row, "REDUCE MOTION") {
		t.Errorf("the timeline does not say why the loop is still:\n%s", row)
	}
	var asks []MapAsk
	f := openTempMap(t, false, &asks).cycleReduceMotion(true)
	if m, cmd, _ := f.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}); m.mapPane.fcPlaying || cmd != nil {
		t.Error("space under Reduce motion played the forecast")
	}
}

// TestReduceMotionStepsTheTicker is D-360 (A-107 (10)): the tape does not
// scroll; each item of the lane is shown whole for 5 s, then the next.
func TestReduceMotionStepsTheTicker(t *testing.T) {
	items := []TickerItem{
		{ID: "tor", Category: CatWarning, Head: "Tornado Warning · the Oklahoma City area", Severity: TickerRed},
		{ID: "svr", Category: CatWarning, Head: "Severe Thunderstorm Warning · Cherry, NE", Severity: TickerOrange},
	}
	d := tickerDash(t, items, false).cycleReduceMotion(true)
	at := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	d.now = func() time.Time { return at }
	d.advanceTicker()
	if d.tickerScroll != 0 {
		t.Errorf("under Reduce motion the tape scrolled to %d", d.tickerScroll)
	}
	shown := func() string { return stripANSITest(d.tickerMarquee(render.Opts{Width: 200})) }
	first := shown()
	if strings.Count(first, "Warning ·") != 1 {
		t.Fatalf("the band shows more or less than one item whole: %q", first)
	}
	d.now = func() time.Time { return at.Add(tickerItemHold) }
	if second := shown(); second == first || strings.Count(second, "Warning ·") != 1 {
		t.Errorf("5 s on the band shows %q; want the next item, whole", second)
	}
}

// TestReduceMotionStillsTheVisualiserAndTheDots is D-360: the bars are not
// drawn and their clock does not run; the loading dots keep one phase.
func TestReduceMotionStillsTheVisualiserAndTheDots(t *testing.T) {
	d := dash(t).(Dashboard)
	d.radioViz, d.radioPlaying = true, true
	d.cfg.Spectrum = func() []float64 { return []float64{0.5, 0.7} }
	if !d.vizActive() {
		t.Fatal("the visualiser is not active with Viz on and the radio playing")
	}
	d = d.cycleReduceMotion(true)
	if d.vizActive() {
		t.Error("under Reduce motion the visualiser still moves")
	}
	before := d.opts().LoadingDots()
	m, _ := d.Update(tickMsg{})
	if after := m.(Dashboard).opts().LoadingDots(); after != before || before != "..." {
		t.Errorf("under Reduce motion the dots went %q to %q; want ... held", before, after)
	}
}

// TestReduceMotionCutsTheRadioLine is D-360: the voice's line is cut with an
// ellipsis rather than slid along as the voice speaks.
func TestReduceMotionCutsTheRadioLine(t *testing.T) {
	d := dash(t).(Dashboard).cycleReduceMotion(true)
	d.radioPlaying, d.radioState, d.radioSpoken = true, "playing", 4*time.Second
	d.radioDetail = strings.Repeat("The forecast for tonight is mostly clear with a light wind. ", 4)
	d.radioSince = time.Now()
	first := stripANSITest(d.marqueeTrack(60))
	d.radioSince = time.Now().Add(-2 * time.Second)
	if later := stripANSITest(d.marqueeTrack(60)); later != first {
		t.Errorf("under Reduce motion the line moved:\n%q\n%q", first, later)
	}
	if !strings.Contains(first, "The forecast for tonight") || !strings.Contains(first, "…") {
		t.Errorf("the line is not the text's start, cut: %q", first)
	}
}

// TestReduceMotionMakesTheBlinksSteady is D-360 (A-107 (10)): a pressed chip
// stays marked until the next key rather than blinking for 350 ms.
func TestReduceMotionMakesTheBlinksSteady(t *testing.T) {
	d := dash(t).(Dashboard).cycleReduceMotion(true)
	m, _ := d.Update(tea.KeyPressMsg{Code: '+', Text: "+"})
	d = m.(Dashboard)
	if d.volFlash != "+" || !d.volFlashEnd.After(time.Now().Add(time.Minute)) {
		t.Fatalf("under Reduce motion the volume mark ends at %v", d.volFlashEnd)
	}
	m, _ = d.Update(tickMsg{})
	if d = m.(Dashboard); d.volFlash != "+" {
		t.Error("a tick cleared the steady mark")
	}
	m, _ = d.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if d = m.(Dashboard); d.volFlash != "" {
		t.Error("the next key did not clear the mark")
	}
}

// TestTheConsoleFollowsReduceMotion is D-360: the console's loading dots
// hold still with the listener's choice, and move without it.
func TestTheConsoleFollowsReduceMotion(t *testing.T) {
	for _, on := range []bool{true, false} {
		d := dash(t).(Dashboard)
		if on {
			d = d.cycleReduceMotion(true)
		}
		r := NewRouter(d)
		r.active = SurfaceBroadcaster // the console's own tick reaches it while it is shown
		var m tea.Model = r
		for range 3 {
			m, _ = m.Update(tickMsg{})
		}
		if frame := m.(Router).broadcaster.frame; (frame == 0) != on {
			t.Errorf("Reduce motion %v: after three ticks the console's dots are at frame %d", on, frame)
		}
	}
}
