package tty

// map_playback_test.go — F-192 to F-194 (FR-7.2, D-361): the map's playback
// speed, a Setting of 1000 to 200 ms a frame, 500 by default, the flash limit
// said past 400 ms; the radar's loop and Forecast mode both step at it.

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// TestPlaybackSpeedIsASettingOf500ByDefault is D-361: the row's choices, its
// default, the file's word for each, and the save.
func TestPlaybackSpeedIsASettingOf500ByDefault(t *testing.T) {
	d := mapDash(t, Config{})
	if d.mapPlayback != 500*time.Millisecond || playbackLabel(d.mapPlayback) != "500 ms a frame" {
		t.Fatalf("by default the speed is %v, %q", d.mapPlayback, playbackLabel(d.mapPlayback))
	}
	want := []time.Duration{1000, 750, 500, 400, 300, 200}
	for i, c := range playbackChoices {
		if c != want[i]*time.Millisecond {
			t.Errorf("choice %d is %v; want %v ms", i, c, want[i])
		}
	}
	if d = d.cycleMapPlayback(true); d.mapPlayback != 400*time.Millisecond || d.uiForSave().MapPlaybackMs != 400 {
		t.Errorf("one step on is %v, saved as %d", d.mapPlayback, d.uiForSave().MapPlaybackMs)
	}
	if d = d.cycleMapPlayback(false); d.uiForSave().MapPlaybackMs != 0 {
		t.Errorf("the default is saved as %d; want it left out", d.uiForSave().MapPlaybackMs)
	}
	for ms, speed := range map[int]time.Duration{0: 500 * time.Millisecond, 1000: time.Second, 200: 200 * time.Millisecond, 123: 500 * time.Millisecond} {
		if got := playbackByKey(ms); got != speed {
			t.Errorf("the file's %d reads as %v; want %v", ms, got, speed)
		}
	}
	if got := mapDash(t, Config{MapPlaybackMs: 750}).mapPlayback; got != 750*time.Millisecond {
		t.Errorf("a file saying 750 opens at %v", got)
	}
}

// TestThePlaybackSpeedReachesTheLoop is D-361: the library plays the radar's
// loop at the speed chosen, and a change reaches it at once.
func TestThePlaybackSpeedReachesTheLoop(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked)
	if got := d.mapPane.m.Loop().Step; got != 500*time.Millisecond {
		t.Fatalf("the loop steps every %v; want the default 500 ms", got)
	}
	d = d.cycleMapPlayback(false) // to 750
	if got := d.mapPane.m.Loop().Step; got != 750*time.Millisecond {
		t.Errorf("after the change the loop steps every %v; want 750 ms", got)
	}
}

// TestTheFastSpeedsSayTheFlashLimit is D-361 (0.18.0 D-59): the two choices
// past the library's 2.5-changes-a-second ceiling say so beside them.
func TestTheFastSpeedsSayTheFlashLimit(t *testing.T) {
	for _, c := range playbackChoices {
		past := strings.Contains(playbackLabel(c), "past the flash limit")
		if want := c < 400*time.Millisecond; past != want {
			t.Errorf("%v: says the flash limit %v; want %v (%q)", c, past, want, playbackLabel(c))
		}
	}
}

// TestForecastStepsAtThePlaybackSpeed is D-361: Forecast mode steps at the
// speed chosen and holds its last day 2 s whatever the speed.
func TestForecastStepsAtThePlaybackSpeed(t *testing.T) {
	for _, c := range playbackChoices {
		if n := forecastHoldSteps(c); time.Duration(n)*c < 2*time.Second || time.Duration(n-1)*c >= 2*time.Second {
			t.Errorf("at %v the last day is held %d steps, %v; want the least that makes 2 s", c, n, time.Duration(n)*c)
		}
	}
	var asks []MapAsk
	d := openTempMap(t, false, &asks)
	m, _, _ := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	d = m.(Dashboard)
	last := len(d.forecastSteps()) - 1
	for range last {
		mm, _ := d.Update(forecastTickMsg{gen: d.mapPane.fcGen})
		d = mm.(Dashboard)
	}
	held := 0
	for d.mapPane.fcStep == last && held < 10 {
		mm, _ := d.Update(forecastTickMsg{gen: d.mapPane.fcGen})
		d = mm.(Dashboard)
		held++
	}
	if held != forecastHoldSteps(500*time.Millisecond) {
		t.Errorf("at 500 ms the last day held %d ticks; want %d, 2 s", held, forecastHoldSteps(500*time.Millisecond))
	}
}
