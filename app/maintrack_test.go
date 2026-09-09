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
		{"live", mainTrackLive},
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

func TestOnlyLiveOwnsTheAirAndDarkStillReports(t *testing.T) {
	if mainTrackOff.reports() {
		t.Error("the default tells the Director nothing; a commit before the flip must be a no-op")
	}
	if !mainTrackDark.reports() {
		t.Error("dark observes the REAL producer, or it observes nothing")
	}
	if !mainTrackLive.reports() {
		t.Error("live reports too")
	}
	if mainTrackOff.ownsTheAir() || mainTrackDark.ownsTheAir() {
		t.Error("the air stays with the rotation's own path until the flip — this is the double-speak window")
	}
	if !mainTrackLive.ownsTheAir() {
		t.Error("live is the merged station")
	}
}
