package tty

// map_playback.go — the map's playback speed, a Map Setting (D-361): how long
// each frame shows while the radar's loop or Forecast mode plays, from
// 1000 ms to 200 ms, 500 by default; past go-tuiMaps' ceiling of 2.5 changes
// a second the row says so (0.18.0 D-59).

import (
	"strconv"
	"time"
)

// playbackChoices are the row's choices, slowest first.
var playbackChoices = []time.Duration{1000 * time.Millisecond, 750 * time.Millisecond, 500 * time.Millisecond, 400 * time.Millisecond, 300 * time.Millisecond, 200 * time.Millisecond}

// defaultPlayback is the speed when none is chosen (0.18.0 D-59).
const defaultPlayback = 500 * time.Millisecond

// flashCeiling is the fastest step within the library's 2.5 changes a
// second; a faster one lifts it, for people that flashing harms.
const flashCeiling = 400 * time.Millisecond

// lastDayHold is how long Forecast mode holds its last day before playing
// again, as the radar's loop holds its newest frame (FR-5.9).
const lastDayHold = 2 * time.Second

// playbackByKey reads the file's milliseconds; anything not offered is the
// default.
func playbackByKey(ms int) time.Duration {
	want := time.Duration(ms) * time.Millisecond
	for _, c := range playbackChoices {
		if c == want {
			return c
		}
	}
	return defaultPlayback
}

// playbackKey is the file's milliseconds for a speed; the default is left out.
func playbackKey(c time.Duration) int {
	if c == defaultPlayback {
		return 0
	}
	return int(c / time.Millisecond)
}

// playbackLabel is a speed as the Settings say it.
func playbackLabel(c time.Duration) string {
	words := strconv.Itoa(int(c/time.Millisecond)) + " ms a frame"
	if c < flashCeiling {
		words += ", past the flash limit"
	}
	return words
}

// forecastHoldSteps is how many steps Forecast mode shows its last day at a
// speed: the fewest that make lastDayHold.
func forecastHoldSteps(step time.Duration) int {
	return int((lastDayHold + step - 1) / step)
}

// cycleMapPlayback moves the speed picker, and round again; the radar's loop
// takes it at once, Forecast mode at its next step.
func (d Dashboard) cycleMapPlayback(forward bool) Dashboard {
	d.mapPlayback = cycleIn(playbackChoices, itself, d.mapPlayback, forward)
	return d.applyPlayback().uiTouched()
}
