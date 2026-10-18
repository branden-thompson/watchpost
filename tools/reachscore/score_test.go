package main

import (
	"errors"
	"testing"
	"time"

	ionomaps "github.com/branden-thompson/go-ionomaps"
)

// TestTheAnswerIsReadAsTheProductReadsIt is D-220 (C1): open and disturbed
// reach, every other status does not; and an answer agrees when it reaches
// on an open pair, or does not on a closed one.
func TestTheAnswerIsReadAsTheProductReadsIt(t *testing.T) {
	for s, want := range map[ionomaps.Status]bool{ionomaps.Open: true, ionomaps.Disturbed: true, ionomaps.Absorbed: false,
		ionomaps.Weak: false, ionomaps.AboveUpper: false, ionomaps.StatusNoData: false} {
		if got := reaches(ionomaps.BandStatus{Band: ionomaps.Band40m, Status: s}); got != want {
			t.Errorf("%v: reaches %t; want %t", s, got, want)
		}
	}
	var tl tally
	tl.add(open, true, false)
	tl.add(weak, false, false)
	tl.add(unheard, true, true)
	if n, agree := tl.total(); n != 3 || agree != 2 || tl.agree[unheard] != 0 || tl.noData[unheard] != 1 {
		t.Errorf("reaching an open pair, not a weak one, and reaching an unheard one: %+v; want 2 of 3 agreeing", tl)
	}
}

// TestAColumnFedOtherwiseFailsTheRun is the columns kept apart: the
// climatology's replay withholds GloTEC, and a snapshot fed other than its
// column says - a grid in the climatology's, a station's reading outside
// the shipped build's - fails the run.
func TestAColumnFedOtherwiseFailsTheRun(t *testing.T) {
	if replayFor(nil, climatology).grids || !replayFor(nil, background).grids || !replayFor(nil, shipped).grids {
		t.Error("the climatology's replay serves GloTEC, or another column's withholds it")
	}
	at := time.Date(2026, 9, 29, 0, 35, 0, 0, time.UTC)
	for _, c := range []struct {
		col column
		f   hourFacts
		bad bool
	}{
		{climatology, hourFacts{Grid: true}, true},
		{background, hourFacts{Grid: true, Stations: 4}, true},
		{climatology, hourFacts{}, false},
		{background, hourFacts{Grid: true}, false},
		{shipped, hourFacts{Grid: true, Stations: 4}, false},
	} {
		err := fedAsSaid(c.col, map[time.Time]hourFacts{at: c.f})
		if errors.Is(err, errReplayed) != c.bad {
			t.Errorf("%s fed %+v: %v", c.col, c.f, err)
		}
	}
}
