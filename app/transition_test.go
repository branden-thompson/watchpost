package app

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestTheResumeTransitionDiffersOnlyByWhetherTheProgrammeKeptRunning — F-27.
//
// The HUM LEAD's ruling: ONE script for both modes. A live relay kept playing
// underneath the read, so the listener is rejoined to something already under
// way; a synth programme did not. Everything else about the sentence is the
// same, which is why it is one file and not two.
func TestTheResumeTransitionDiffersOnlyByWhetherTheProgrammeKeptRunning(t *testing.T) {
	synth, relay := programmeReturnLine(nil, false), programmeReturnLine(nil, true)
	if synth == "" || relay == "" {
		t.Fatal("the resume transition rendered nothing; the listener gets a hard cut")
	}
	for _, got := range []string{synth, relay} {
		if !strings.HasPrefix(got, "Watchpost Radio now returns to its regularly scheduled programming") {
			t.Errorf("the transition reads %q, want the ratified opening", got)
		}
		if !strings.HasSuffix(got, ".") {
			t.Errorf("the transition reads %q; it is a sentence", got)
		}
	}
	if synth == relay {
		t.Error("both modes read the same; a live relay is rejoined mid-programme and must say so")
	}
	if !strings.Contains(relay, "already in progress") {
		t.Errorf("the relay transition reads %q, want it to say the programme is already in progress", relay)
	}
	if strings.Contains(synth, "already in progress") {
		t.Errorf("the synth transition reads %q; nothing was running underneath it", synth)
	}
	// THE DIFFERENCE IS ONE CLAUSE, not a second sentence. That is the whole
	// argument for one script: a second file would restate the opening.
	if strings.TrimSuffix(synth, ".") != strings.Split(strings.TrimSuffix(relay, "."), ",")[0] {
		t.Errorf("the two readings differ by more than the one clause:\n  synth %q\n  relay %q", synth, relay)
	}
}

// TestTheMastheadNamesWhereWhatAndTheLimits — F-24.
//
// It covers the resumption gap, so it must be long enough to be worth reading —
// but every part of it is load-bearing: where the station broadcasts from, what
// area it covers, whose data it is, and the limitation a weather service is
// obliged to state.
func TestTheMastheadNamesWhereWhatAndTheLimits(t *testing.T) {
	at := snapshot.DefaultOrigin()
	got := mastheadLine(nil, at, 50, []string{"the National Weather Service", "the United States Geological Survey", "NASA FIRMS"})
	for _, want := range []string{
		"This is Watchpost Weather Radio",
		at.Label,
		"a 50 mile radius",
		"the National Weather Service, the United States Geological Survey, and NASA FIRMS",
		"not intended for life safety use",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the masthead does not say %q:\n%s", want, got)
		}
	}
	// ALL LOCATIONS, NOT A NOUGHT-MILE RADIUS. `0` is the ALERTS - EVENTS
	// setting's "All", and this is the one place that value is SPOKEN — a
	// station announcing a 0 mile radius states the opposite of what it covers.
	all := mastheadLine(nil, at, 0, []string{"the National Weather Service"})
	if strings.Contains(all, "0 mile") {
		t.Errorf("an unfenced station announces %q", all)
	}
	if !strings.Contains(all, "all locations") {
		t.Errorf("an unfenced station must say it covers all locations, got:\n%s", all)
	}
	// ONE PROVIDER IS NOT A LIST — asserted on the PROVIDER CLAUSE alone. The
	// masthead's own fixed wording contains ", and seismic reports", so a naive
	// search over the whole line matches the sentence rather than the list.
	one := mastheadLine(nil, at, 25, []string{"the National Weather Service"})
	clause := func(line string) string {
		_, rest, ok := strings.Cut(line, "compiled from ")
		if !ok {
			t.Fatalf("the masthead names no providers: %q", line)
		}
		got, _, _ := strings.Cut(rest, ". ")
		return got
	}
	if got := clause(one); got != "the National Weather Service" {
		t.Errorf("a single provider reads as %q, want it alone", got)
	}
	if got := clause(got); !strings.Contains(got, ", and NASA FIRMS") {
		t.Errorf("three providers read as %q, want a spoken list", got)
	}
}
