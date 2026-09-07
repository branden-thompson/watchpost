package render

import (
	"testing"
	"time"
)

// Since DATES anything that is not from today.
//
// A bare "3:42 PM" answers "when" only for something that happened today, and
// the ticker holds significant quakes for seven days and a storm for as long as
// the feed lists it. The rule is the timestamp, not the caller's idea of how
// long its events live — which is why it covers a multi-day winter-storm
// product nobody enumerated, and leaves today's warnings exactly as they were.
func TestSinceDatesAnythingNotFromToday(t *testing.T) {
	now := time.Date(2026, 8, 30, 9, 0, 0, 0, time.Local)
	for _, tc := range []struct {
		name string
		at   time.Time
		want string
	}{
		{"today's warning", time.Date(2026, 8, 30, 15, 42, 0, 0, time.Local), "3:42 PM"},
		{"three days back", time.Date(2026, 8, 27, 15, 42, 0, 0, time.Local), "08/27 3:42 PM"},
		{"last week", time.Date(2026, 8, 23, 6, 5, 0, 0, time.Local), "08/23 6:05 AM"},
		{"the small hours of today", time.Date(2026, 8, 30, 0, 15, 0, 0, time.Local), "12:15 AM"},
		{"the same date a year ago", time.Date(2025, 8, 30, 15, 42, 0, 0, time.Local), "08/30 3:42 PM"},
	} {
		if got := Clock12.Since(tc.at, now); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
	at := time.Date(2026, 8, 27, 15, 42, 0, 0, time.Local)
	for _, tc := range []struct {
		c    Clock
		want string
	}{{Clock12, "08/27 3:42 PM"}, {Clock24, "08/27 15:42"}, {ClockMil, "08/27 1542"}} {
		if got := tc.c.Since(at, now); got != tc.want {
			t.Errorf("%v: %q, want %q", tc.c, got, tc.want)
		}
	}
}

// SPOKEN times: twelve- and twenty-four-hour read the
// SAME, because they are two ways of writing one instant and a person says that
// instant one way. Military is its own reading out loud as well as on the page.
func TestSpokenReadsTwelveAndTwentyFourAlikeAndMilitaryApart(t *testing.T) {
	for _, tc := range []struct {
		h, m        int
		twelve, mil string
	}{
		// The HUM LEAD's own readings, in their order.
		{0, 0, "Twelve AM", "Zero Hundred Hours"},
		{1, 0, "One AM", "Oh One Hundred Hours"},
		{1, 15, "One Fifteen AM", "Oh One Fifteen Hours"},
		{5, 30, "Five Thirty AM", "Oh Five Thirty Hours"},
		{7, 59, "Seven Fifty-Nine AM", "Oh Seven Fifty-Nine Hours"},
		{10, 0, "Ten AM", "Ten Hundred Hours"},
		{11, 0, "Eleven AM", "Eleven Hundred Hours"},
		{12, 0, "Twelve PM", "Twelve Hundred Hours"},
		{14, 10, "Two Ten PM", "Fourteen Ten Hours"},
		{16, 30, "Four Thirty PM", "Sixteen Thirty Hours"},
		{20, 0, "Eight PM", "Twenty Hundred Hours"},
		{23, 59, "Eleven Fifty-Nine PM", "Twenty-Three Fifty-Nine Hours"},
		// And the two the examples did not cover.
		{16, 50, "Four Fifty PM", "Sixteen Fifty Hours"},
		{16, 5, "Four Oh Five PM", "Sixteen Oh Five Hours"},
	} {
		ts := time.Date(2026, 8, 30, tc.h, tc.m, 0, 0, time.Local)
		if got := Clock12.Spoken(ts); got != tc.twelve {
			t.Errorf("%02d:%02d 12h: %q, want %q", tc.h, tc.m, got, tc.twelve)
		}
		// The 24-hour clock reads the same as the 12-hour one — the whole point.
		if got := Clock24.Spoken(ts); got != tc.twelve {
			t.Errorf("%02d:%02d 24h: %q, want the 12-hour reading %q", tc.h, tc.m, got, tc.twelve)
		}
		if got := ClockMil.Spoken(ts); got != tc.mil {
			t.Errorf("%02d:%02d MIL: %q, want %q", tc.h, tc.m, got, tc.mil)
		}
	}
}

// SPOKEN IDENTIFIERS under military: NATO phonetics,
// digits one at a time, and NINER for nine — the digit the convention renames
// because over a poor signal "nine" and "five" are the same word.
func TestSpokenIDUsesNATOPhoneticsOnlyUnderMilitary(t *testing.T) {
	for _, tc := range []struct{ in, mil string }{
		{"KCEQ", "Kilo Charlie Echo Quebec"},
		{"KC21F9", "Kilo Charlie Two One Foxtrot Niner"},
		{"KEC62", "Kilo Echo Charlie Six Two"},
		{"WXL58", "Whiskey Xray Lima Five Eight"},
		{"kec62", "Kilo Echo Charlie Six Two"}, // case is not part of a callsign
		{"", ""},
	} {
		if got := ClockMil.SpokenID(tc.in); got != tc.mil {
			t.Errorf("MIL %q: %q, want %q", tc.in, got, tc.mil)
		}
		// The other two leave it alone: a synthesiser reads "KEC62" letter by
		// letter well enough, and spelling every callsign out in full would make
		// a routine lead read like a drill.
		for _, c := range []Clock{Clock12, Clock24} {
			if got := c.SpokenID(tc.in); got != tc.in {
				t.Errorf("%v %q: %q, want it untouched", c, tc.in, got)
			}
		}
	}
	// TITLE CASE, not the capitals the convention is written in: a synthesiser
	// that sees "KILO" may spell it back as four letters.
	if got := ClockMil.SpokenID("K"); got != "Kilo" {
		t.Errorf("a phonetic word is a word, not an initialism: %q", got)
	}
}
