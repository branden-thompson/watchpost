package tty

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestTheTapeTurnsOverAtMidnight is 0.19.5 Q4.3's pin, written before the tape
// memo: an alert's time reads as a time of day on the day it happened and as a
// date and time from the next day on, so the tape a frame draws changes at
// midnight with nothing else changing. A tape memo keyed without the day would
// keep yesterday's words.
func TestTheTapeTurnsOverAtMidnight(t *testing.T) {
	d := benchDash(t, 133, 44).(Dashboard)
	at := time.Date(2026, 8, 24, 23, 50, 0, 0, time.Local)
	before := at.Add(9 * time.Minute)
	d.now = func() time.Time { return before } // before setTicker, which measures the tape
	lane := d.ticker[0].Category
	d.setTicker([]TickerItem{{ID: "w1", Category: lane, Head: "Tornado Warning · Midnight County, KS", Verb: "issued", At: at}})
	o := d.opts()
	evening := strings.Join(d.tickerLane(o, lane), "")
	if band := stripANSITest(d.tickerMarquee(o)); !strings.Contains(band, "issued "+o.Clock.Time(at)) || strings.Contains(band, o.Clock.DateTime(at)) {
		t.Fatalf("before midnight the band should give the alert's time of day alone:\n%s", band)
	}
	_ = d.View().Content
	after := at.Add(11 * time.Minute)
	d.now = func() time.Time { return after }
	morning := strings.Join(d.tickerLane(o, lane), "")
	_ = d.View().Content
	if evening == morning {
		t.Fatalf("the tape reads %q either side of midnight; the day it was issued it is a time, the next day a date and time", evening)
	}
	if band := stripANSITest(d.tickerMarquee(o)); !strings.Contains(band, o.Clock.DateTime(at)) {
		t.Errorf("after midnight the band does not date the alert %q:\n%s", o.Clock.DateTime(at), band)
	}
}

// missEveryMemo empties every memo the frame reads, so an allocation count
// taken after it is the whole frame's cost (RR-17).
func missEveryMemo(d Dashboard) {
	d.memo.ok, d.memo.tape.ok = false, false
}

// TestTheTapeKeyCoversEverythingTheTapeShows is the tape memo's completeness
// guard (0.19.5 Q4.3, A-15), the body memo's property for the tape: if two
// models would draw different tapes for their lane, their keys differ. Every
// field the reflection walk can change, and by name the three inputs it
// cannot reach: the ticker's contents (through setTicker, its one writer),
// the clock a day on, and the other lane.
func TestTheTapeKeyCoversEverythingTheTapeShows(t *testing.T) {
	base := benchDash(t, 133, 44).(Dashboard)
	lanes := append([]TickerItem(nil), base.ticker...)
	for _, c := range tickerCatOrder() { // a second lane, so the lane is an input the guard can vary
		if c != lanes[0].Category {
			lanes = append(lanes, TickerItem{ID: "second", Category: c, Head: "Second lane · Somewhere", Verb: "issued", At: lanes[0].At})
			break
		}
	}
	base.setTicker(lanes)
	if base.tickerGen == 0 || len(base.tickerCategories()) < 2 {
		t.Fatal("the fixture's ticker is not set through setTicker over two lanes; the guard would check nothing")
	}
	drawn := func(d Dashboard) (string, tapeKey) {
		o, lane := d.opts(), d.tickerCategories()[d.tickerCatIdx%len(d.tickerCategories())]
		bare := d
		bare.memo = nil // the tape as built, not as remembered
		items, tape, loop := bare.tapeOf(o, lane)
		return fmt.Sprint(items, tape, loop), d.tapeKeyFor(o, lane)
	}
	before, baseKey := drawn(base)
	named := []perturbed{
		{field: "ticker (through setTicker)", model: func() Dashboard {
			d := base
			items := append([]TickerItem(nil), d.ticker...)
			items[0].Head += " · changed"
			d.setTicker(items)
			return d
		}()},
		{field: "the clock, a day on", model: func() Dashboard {
			d := base
			now := d.clock().Add(24 * time.Hour)
			d.now = func() time.Time { return now }
			return d
		}()},
		{field: "tickerCatIdx", model: func() Dashboard { d := base; d.tickerCatIdx++; return d }()},
	}
	reached := 0
	for _, p := range append(perturbations(t, base), named...) {
		after, key := drawn(p.model)
		if after == before {
			continue
		}
		reached++
		if key == baseKey {
			t.Errorf("changing %s changes the tape and NOT tapeKey: the tape memo (ticker.go, tapeOf) will draw the old tape; add what %s changes to tapeKeyFor, or have its writer bump tickerGen", p.field, p.field)
		}
	}
	if reached < 3 {
		t.Errorf("only %d perturbations reached the tape; the fixture stopped exercising it and this guard asserts nothing", reached)
	}
}
