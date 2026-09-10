package app

import "testing"

// The switch is the one thing standing between a half-merged producer and two
// owners of the air, so its default and its unknown case are pinned rather
// than assumed.
func TestTheMergeIsOffUntilItIsAskedFor(t *testing.T) {
	for _, tc := range []struct {
		env  string
		want mainTrackStage
	}{
		{"", mainTrackOff},
		{"dark", mainTrackDark},
		{"live", mainTrackOff},     // D-33 retired the live stage; it is now just an unknown word
		{"1", mainTrackOff},        // truthy-looking, and NOT live
		{"true", mainTrackOff},     //
		{"LIVE", mainTrackOff},     // a case slip is not a licence
		{" live", mainTrackOff},    // nor a stray space
		{"off", mainTrackOff},      //
		{"anything", mainTrackOff}, //
	} {
		t.Setenv("WATCHPOST_MAINTRACK", tc.env)
		if got := mainTrack(); got != tc.want {
			t.Errorf("WATCHPOST_MAINTRACK=%q: want stage %d, got %d — an unrecognised value must never hand over the air", tc.env, tc.want, got)
		}
	}
}

// D-33 RETIRED THE LIVE STAGE, and with it the question this test used to ask.
// "live" meant the card was read through the ARBITER; the programme is not a
// narration, so no stage can hand it over that way. What is left is the one
// distinction that still means something: does the deck TELL the Director.
func TestOnlyDarkReportsAndNothingElseChanges(t *testing.T) {
	if mainTrackOff.reports() {
		t.Error("the default tells the Director nothing; a build nobody asked must be a no-op")
	}
	if !mainTrackDark.reports() {
		t.Error("dark observes the REAL producer, or it observes nothing")
	}
	// AND "live" IS JUST A WORD NOW. A shell profile left over from the
	// reverted merge must read as off, not as something the parser still
	// knows.
	t.Setenv("WATCHPOST_MAINTRACK", "live")
	if got := mainTrack(); got != mainTrackOff {
		t.Errorf("a stale WATCHPOST_MAINTRACK=live must read as off; got stage %d", got)
	}
}

// The dark run's log names the stage, and the registry is walked rather than
// listed — a fourth stage added without a name would fail here rather than
// print an empty string into the one log the comparison is made from.
func TestEveryStageNamesItself(t *testing.T) {
	seen := map[string]bool{}
	for s := mainTrackStage(0); s < numMainTrackStages; s++ {
		name := s.String()
		if name == "" {
			t.Errorf("stage %d has no name; the dark run's log is the only place the comparison is made", s)
		}
		if name == "undeclared" {
			t.Errorf("stage %d is inside the registry and must not name itself undeclared", s)
		}
		if seen[name] {
			t.Errorf("two stages are both called %q; the log could not tell them apart", name)
		}
		seen[name] = true
	}
	if len(seen) != int(numMainTrackStages) {
		t.Errorf("named %d stages of %d", len(seen), numMainTrackStages)
	}
	// A CORRUPT VALUE SAYS SO rather than reading as the safe default: a
	// diagnostic that quietly reports "off" would hide the one case worth
	// seeing.
	if got := mainTrackStage(-1).String(); got != "undeclared" {
		t.Errorf("an out-of-range stage names itself undeclared, got %q", got)
	}
	if got := mainTrackStage(numMainTrackStages).String(); got != "undeclared" {
		t.Errorf("an out-of-range stage names itself undeclared, got %q", got)
	}
}
