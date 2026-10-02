package app

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// TestTheProgrammeReturnsWordsAreKept is D-163's seam, as D-158's: F-27's
// spoken transition back to the programme is not wired - the line-up's
// ProgrammeReturn is set nowhere in production - and has no composer; its
// words stay in the script library, held here, for the day it is wired
// (app/schedule.go says where).
//
// The HUM LEAD's ruling: ONE script for both modes. A live relay keeps playing
// underneath the read, so the listener is rejoined to something already under
// way; a synth programme does not. Everything else about the sentence is the
// same, which is why it is one file and not two.
func TestTheProgrammeReturnsWordsAreKept(t *testing.T) {
	say := func(live bool) string {
		return scriptText(nil, "transition", "resume", map[string]any{"InProgress": live})
	}
	synth, relay := say(false), say(true)
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

// TestTheStationIDsWordsAreKept is D-158's seam: the station identification
// on going ON AIR (F-24) is not wired and has no composer, and its words stay in
// the script library for the day a use case needs it - where the station
// broadcasts from, what it covers, whose data it is, and the limitation a
// weather service is obliged to state. A seam whose words had rotted would
// not be one small change to add back.
func TestTheStationIDsWordsAreKept(t *testing.T) {
	got := scriptText(nil, "transition", "masthead", map[string]any{
		"Location": snapshot.DefaultOrigin().Label, "Coverage": "a 50 mile radius", "Providers": "the National Weather Service",
	})
	for _, want := range []string{"This is Watchpost Weather Radio", snapshot.DefaultOrigin().Label, "a 50 mile radius", "the National Weather Service", "not intended for life safety use"} {
		if !strings.Contains(got, want) {
			t.Errorf("the station ID's words do not say %q:\n%s", want, got)
		}
	}
}
