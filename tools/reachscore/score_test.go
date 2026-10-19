package main

import (
	"errors"
	"math/rand/v2"
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
	tl.add(open, true)
	tl.add(weak, false)
	tl.add(unheard, true)
	if n, agree := tl.total(); n != 3 || agree != 2 || tl.agree[unheard] != 0 {
		t.Errorf("reaching an open pair, not a weak one, and reaching an unheard one: %+v; want 2 of 3 agreeing", tl)
	}
}

// TestAColumnFedOtherwiseFailsTheRun is the columns kept apart, both ways:
// the climatology's replay withholds GloTEC, and a snapshot fed other than
// its column says - a grid in the climatology's or none in another's, a
// station's reading outside the shipped build's or none in it, a file the
// library could not read - fails the run.
func TestAColumnFedOtherwiseFailsTheRun(t *testing.T) {
	if !replayOptions(climatology).WithoutGloTEC || replayOptions(background).WithoutGloTEC || replayOptions(shipped).WithoutGloTEC {
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
		{background, hourFacts{}, true},                                // a missing grid: the background silently the climatology
		{shipped, hourFacts{Grid: true}, true},                         // no station: the shipped build silently the background
		{shipped, hourFacts{Grid: true, Stations: 4, Unread: 1}, true}, // a file the library could not read
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

// TestTheDefaultDrawIsByBand is D-224 (fork E): unless told otherwise the
// draw is by band alone, 80 of each outcome a band - the size B1's ten
// per band and hour made - and the strata are one of the two ruled.
func TestTheDefaultDrawIsByBand(t *testing.T) {
	f, err := parseScoreFlags([]string{"-inputs", "i", "-solar", "s", "-spots", "p", "-out", "o"})
	if err != nil || f.strata != strataBand || f.k != 80 {
		t.Errorf("the default draw: strata %q, k %d, %v; want %q and 80", f.strata, f.k, err, strataBand)
	}
	if _, err := parseScoreFlags([]string{"-inputs", "i", "-solar", "s", "-spots", "p", "-out", "o", "-strata", "hour"}); !errors.Is(err, errUsage) {
		t.Errorf("strata by hour alone: %v; want errUsage", err)
	}
}

// TestTheIntervalKeepsThePairing is D-228: each agreement and the shipped
// build's lead carry a 95 % paired bootstrap interval. Two columns that
// agree and disagree on the same scenarios have a lead of exactly zero on
// every draw; a lead of a tenth of the scenarios has an interval about it
// that holds it and excludes zero; the seed fixes the figures.
func TestTheIntervalKeepsThePairing(t *testing.T) {
	var same, ahead [][columns]bool
	for i := range 400 {
		same = append(same, [columns]bool{i%2 == 0, true, i%2 == 0})
		ahead = append(ahead, [columns]bool{i%2 == 0, false, i%2 == 0 || i%10 == 1})
	}
	rng := func() *rand.Rand { return rand.New(rand.NewPCG(7, 7)) }
	b := bootstrap(same, rng())
	if b.lead != (interval{}) || b.cols[climatology].lo > 50 || b.cols[climatology].hi < 50 {
		t.Errorf("identical columns: lead %+v, climatology %+v; want a lead of exactly 0 and 50 inside", b.lead, b.cols[climatology])
	}
	a := bootstrap(ahead, rng())
	if a.lead.lo <= 0 || a.lead.lo > 10 || a.lead.hi < 10 {
		t.Errorf("a lead of 10 points: %+v; want it held and zero excluded", a.lead)
	}
	if again := bootstrap(ahead, rng()); again != a {
		t.Errorf("the same seed gave %+v then %+v", a, again)
	}
}
