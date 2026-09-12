package tty

// broadcaster_air_test.go — D-95: what is on the air, and what is under it.
//
// THE BOX HOLDS BOTH BECAUSE THEY ARE ONE QUESTION. The line-up and the bed are
// mutually exclusive by product rule (D-11, FR-4.2) and by the engine having one
// source, so a box that holds the two and marks which is live says the
// exclusivity in its shape rather than in a sentence.
//
// IT SUPERSEDES D-89's STANDBY BOX. That box was thirteen rows tall because LIVE
// was one card of ten; as one row of two it says the same thing in the space it
// has, and the HUM LEAD's wording is unchanged.

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// airRows is the air box, found by its own label column.
func airRows(t *testing.T, frame string) []string {
	t.Helper()
	rows := strings.Split(frame, "\n")
	for i, r := range rows {
		if strings.Contains(r, "LIVE NOW") {
			return rows[max(0, i-1):min(len(rows), i+4)]
		}
	}
	t.Fatalf("the frame has no air box, so this test measures nothing")
	return nil
}

// THE TWO ROWS ARE ALWAYS BOTH THERE (HUM LEAD, 2026-09-12: "Up Next and Alert
// are always present"), and the same is true of the air: a station has a
// programme and a bed whether or not either is carrying.
func TestTheAirBoxAlwaysDrawsBothRows(t *testing.T) {
	b := bcWith(t, card(t, "a", "Oceanside, CA"))
	got := stripANSITest(b.View().Content)
	for _, want := range []string{"LIVE NOW", "RELAY BED"} {
		if !strings.Contains(got, want) {
			t.Errorf("the air box is missing its %q row", want)
		}
	}
}

// AT REST THE LIVE ROW SAYS SO, in the HUM LEAD's own words (D-89, carried over).
func TestTheLiveRowCarriesTheStandbyNoticeAtRest(t *testing.T) {
	b := bcWith(t, card(t, "a", "Oceanside, CA"))
	air := strings.Join(airRows(t, stripANSITest(b.View().Content)), "\n")
	if !strings.Contains(air, bcStandbyNotice) {
		t.Errorf("nothing is on the air and the LIVE row does not say so:\n%s", air)
	}
}

// AND IT NAMES NO REPORT. The row used to be a card built from `lineup.Card{}`,
// whose zero Slot IS a location report — so it drew `LOCATION REPORT •STANDARD•
// [0]`: a report that does not exist, graded, with a chip that opens nothing.
func TestTheLiveRowNamesNoReportAtRest(t *testing.T) {
	b := bcWith(t, card(t, "a", "Oceanside, CA"))
	rows := airRows(t, stripANSITest(b.View().Content))
	var live string
	for _, r := range rows {
		if strings.Contains(r, "LIVE NOW") {
			live = r
		}
	}
	for _, forbidden := range []string{"LOCATION REPORT", "STANDARD", chipFor("0")} {
		if strings.Contains(live, forbidden) {
			t.Errorf("the LIVE row carries %q while nothing is on the air:\n%s", forbidden, live)
		}
	}
}

// GOING ON AIR PUTS THE CARD THERE, and the notice goes.
func TestGoingOnAirFillsTheLiveRow(t *testing.T) {
	b := bcWith(t, card(t, "a", "Oceanside, CA"), card(t, "b", "Bonsall, CA"))
	live, _ := b.Update(StationMsg{Power: lineup.Running})
	got := stripANSITest(live.View().Content)

	if strings.Contains(got, bcStandbyNotice) {
		t.Errorf("the station is on the air and the LIVE row still says nothing is")
	}
	air := strings.Join(airRows(t, got), "\n")
	if !strings.Contains(air, "Oceanside") {
		t.Errorf("the head of the queue did not take the air:\n%s", air)
	}
	// AND IT OFFERS THE WAY IN. The full read is behind `0`, which is the chip
	// the reference draws on that row.
	if !strings.Contains(air, "Full Read") {
		t.Errorf("the LIVE row offers no way to read the whole report:\n%s", air)
	}
}

// THE MARK SAYS WHICH ONE IS CARRYING, which is the exclusivity drawn rather than
// described: exactly one of the two rows can be the programme.
func TestExactlyOneAirRowIsMarkedLive(t *testing.T) {
	count := func(b Broadcaster) int {
		// THE FRAME'S OWN GLYPHS, not a second Opts: `bcWith` renders in unicode
		// and a test that asked the ASCII set would look for a mark the frame
		// never draws — and would then pass for the wrong reason.
		g := b.opts().Glyphs()
		n := 0
		for _, r := range airRows(t, stripANSITest(b.View().Content)) {
			if strings.Contains(r, g.Live) && (strings.Contains(r, "LIVE NOW") || strings.Contains(r, "RELAY BED")) {
				n++
			}
		}
		return n
	}
	at := bcWith(t, card(t, "a", "Oceanside, CA"))
	if got := count(at); got != 0 {
		t.Errorf("a station at rest marks %d rows live; nothing is carrying", got)
	}
	on, _ := at.Update(StationMsg{Power: lineup.Running})
	if got := count(on); got != 1 {
		t.Errorf("a running station marks %d rows live, want exactly one", got)
	}
}
