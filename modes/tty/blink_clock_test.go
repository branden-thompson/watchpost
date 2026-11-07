package tty

import (
	"testing"
	"time"
)

// TestABlinkEndsOnTheModelsClock is 0.19.5 Q3.1 (the router trace's seam): a
// pressed chip's blink is set and ended on the model's clock, which is the
// wall clock in the app, so a test that pins the clock pins every blink. The
// volume blink is lit 349 ms after the press and cleared by the first tick at
// 350 ms; the Settings picker's blink the same.
func TestABlinkEndsOnTheModelsClock(t *testing.T) {
	at := time.Date(2026, 8, 24, 1, 2, 0, 0, time.UTC)
	d := benchDash(t, 133, 44).(Dashboard)
	d.now = func() time.Time { return at }
	d.volFlash, d.volFlashEnd = "+", d.markEnds()
	d.setup.flash, d.setup.flashEnd = flashLeft, d.markEnds()
	if want := at.Add(pickerFlashDur); !d.volFlashEnd.Equal(want) {
		t.Fatalf("the blink ends at %v; want %v, the model's clock plus the blink", d.volFlashEnd, want)
	}
	d.now = func() time.Time { return at.Add(pickerFlashDur - time.Millisecond) }
	if lit := d.applyTick(); lit.volFlash == "" || lit.setup.flash == flashNone {
		t.Error("a tick 1 ms before the blink ends cleared it")
	}
	d.now = func() time.Time { return at.Add(pickerFlashDur) }
	if done := d.applyTick(); done.volFlash != "" || done.setup.flash != flashNone {
		t.Error("the first tick once the blink ends left it lit")
	}
}
