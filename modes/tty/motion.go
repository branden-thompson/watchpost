package tty

// motion.go — Reduce motion, one switch for the whole app (D-360): on,
// nothing moves on its own clock. The map's loops and Forecast mode do not
// play (go-tuiMaps ReduceMotion); the ticker steps item by item; the
// visualiser and the loading dots hold still; the radio line is cut; a
// pressed chip stays marked until the next key. The console follows.

import (
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
)

// tickerItemHold is how long the still ticker shows each item (A-107 (10)).
const tickerItemHold = 5 * time.Second

// steadyMark is how long a pressed chip stays marked under Reduce motion:
// until the next key clears it, which comes long before this.
const steadyMark = 24 * time.Hour

// reduceMotionLabel is the row's choice as the Settings say it.
func reduceMotionLabel(on bool) string {
	if on {
		return "On"
	}
	return "Off"
}

// cycleReduceMotion switches Reduce motion: on, a playing forecast stops
// where it is, the dots and the bars hold still at once, and the map takes it
// at its next frame.
func (d Dashboard) cycleReduceMotion(bool) Dashboard {
	d.reduceMotion = !d.reduceMotion
	if d.reduceMotion {
		d.mapPane.fcPlaying, d.mapPane.fcGen = false, d.mapPane.fcGen+1
		d.frame, d.vizBands = 0, nil
	}
	return d.applyMotion().uiTouched()
}

// applyMotion hands the map the listener's choice; the same answer again
// changes nothing.
func (d Dashboard) applyMotion() Dashboard {
	m := d.mapPane.m
	if m == nil {
		return d
	}
	on := d.reduceMotion
	d.mapPane.call("ReduceMotion", func() { m.ReduceMotion(on) })
	return d
}

// shimmerStep moves the loading dots one phase on the tick; under Reduce
// motion they hold still.
func (d Dashboard) shimmerStep() Dashboard {
	if d.reduceMotion {
		return d
	}
	d.frame++
	return d
}

// markEnds is when a pressed chip's acknowledgement ends: after the 350 ms
// blink, or under Reduce motion at the next key.
func (d Dashboard) markEnds() time.Time {
	if d.reduceMotion {
		return time.Now().Add(steadyMark)
	}
	return time.Now().Add(pickerFlashDur)
}

// keyClearsMarks ends every pressed chip's mark as a key arrives under
// Reduce motion, so the last one stays until the next key makes its own.
func (d Dashboard) keyClearsMarks() Dashboard {
	if !d.reduceMotion {
		return d
	}
	d.volFlash, d.mapPane.flash, d.mapPane.menuFlash, d.setup.flash = "", "", flashNone, flashNone
	return d
}

// cutToWidth is text whole when it fits, else cut with the glyph set's
// ellipsis: the still form of a line that would slide or scroll.
func cutToWidth(o render.Opts, text string, w int) string {
	n := render.Width(text)
	if n <= w {
		return text
	}
	dots := o.Glyphs().Ellipsis
	return truncateTo(text, max(w-render.Width(dots), 0)) + dots
}

// stillTickerItem is the one item of a lane the still ticker shows: each
// whole for tickerItemHold, then the next, by the wall clock.
func (d Dashboard) stillTickerItem(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[int(d.now().Unix()/int64(tickerItemHold/time.Second))%len(items)]
}
