package tty

// broadcaster_script_test.go — the LIVE card's window onto the read (F-84,
// closed at D-83).
//
// THE HUM LEAD ASKED FOR THE CARD BY ITS CONTENT: "the LIVE CARD should be
// bigger to support showing at least 'most' the script being played." D-68 built
// the geometry; the words had been reaching the console since T3.8 and nothing
// drew them, so a card that was genuinely reading looked exactly like a broken
// one.

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

// reading is a card with words on it, the way one arrives at standby (DR-7).
func readingCard(t *testing.T, lines ...string) lineup.Card {
	t.Helper()
	parts := make([]lineup.Part, 0, len(lines))
	for _, l := range lines {
		parts = append(parts, lineup.Part{Kind: lineup.PartLine, Text: l})
	}
	c, err := lineup.Propose(lineup.Card{ID: "c", Slot: lineup.LocationReport,
		Subject: "oceanside", Headline: "LOCATION REPORT • OCEANSIDE, CA 92057"})
	if err != nil {
		t.Fatalf("proposing: %v", err)
	}
	c.Script = lineup.Script{Parts: parts}
	return c
}

// window is the card's script rows as the console draws them, at the reference
// width.
func window(t *testing.T, c lineup.Card) []string {
	t.Helper()
	b := Broadcaster{width: 150, ascii: true}
	lane := newCardLane(b.cardBoxWidth(), b.opts().Glyphs())
	return b.scriptWindow(b.opts(), lane, c)
}

func TestTheLiveCardShowsTheScriptItIsReading(t *testing.T) {
	c := readingCard(t, "Now, the weather for Oceanside.", "Currently sixty-one degrees and fair.")

	rows := window(t, c)

	if len(rows) != bcReadLines {
		t.Fatalf("the window is %d rows; got %d", bcReadLines, len(rows))
	}
	if !strings.Contains(rows[0], "Now, the weather for Oceanside.") {
		t.Errorf("row 0 is %q; the card's first line is not in it", rows[0])
	}
	if !strings.Contains(rows[1], "Currently sixty-one degrees") {
		t.Errorf("row 1 is %q; the card's second line is not in it", rows[1])
	}
	// EACH PART STARTS ITS OWN LINE. The script's shape is what the listener
	// hears — head, lines, tail — and running two parts together on one row
	// would show the operator a different arrangement from the one on air.
	if strings.Contains(rows[0], "Currently") {
		t.Errorf("two parts were run together on one row: %q", rows[0])
	}
	// THE INSET IS THE CARD'S OWN, the column its control sits at.
	for i, r := range rows[:2] {
		if !strings.HasPrefix(r, bcCardInset) {
			t.Errorf("row %d begins %q; the card's text is inset to %d cells", i, r, len(bcCardInset))
		}
	}
}

// THE HEIGHT NEVER MOVES. A card that grew when its words arrived would shift
// every card below it at the moment the operator is reading one — readBody's own
// rule, and the window is what would break it.
func TestTheScriptWindowIsTheSameHeightWithAnyScript(t *testing.T) {
	for _, tc := range []struct {
		name string
		card lineup.Card
	}{
		{"no card at all", lineup.Card{}},
		{"a card with no words yet", readingCard(t)},
		{"one line", readingCard(t, "Now, the weather for Oceanside.")},
		{"more lines than fit", readingCard(t, "one", "two", "three", "four", "five", "six", "seven")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if rows := window(t, tc.card); len(rows) != bcReadLines {
				t.Errorf("the window is %d rows; got %d", bcReadLines, len(rows))
			}
		})
	}
}

// A LONG LINE WRAPS RATHER THAN BEING CUT. A report's sentences are longer than
// a card is wide, and a window that truncated each one would show the operator
// the first half of five sentences instead of the opening of the read.
func TestALongLineWrapsInsideTheCard(t *testing.T) {
	long := "The National Weather Service in San Diego has issued a hazardous weather outlook " +
		"for southwestern California, including the coastal and valley areas of San Diego County."
	c := readingCard(t, long)

	rows := window(t, c)

	b := Broadcaster{width: 150, ascii: true}
	room := newCardLane(b.cardBoxWidth(), b.opts().Glyphs()).inner() - 2*len(bcCardInset)
	used := 0
	for _, r := range rows {
		if strings.TrimSpace(r) == "" {
			continue
		}
		used++
		if w := render.Width(r) - len(bcCardInset); w > room {
			t.Errorf("a row is %d cells of a %d-cell window: %q", w, room, r)
		}
	}
	if used < 2 {
		t.Errorf("a sentence longer than the card used %d rows; it must wrap onto the next", used)
	}
	if !strings.Contains(rows[0]+rows[1], "hazardous weather outlook") {
		t.Errorf("the wrap lost the middle of the sentence: %q / %q", rows[0], rows[1])
	}
}

// AND IT SAYS WHEN THERE IS MORE. A window with no sign of its own edge reads as
// a short report — on a station, the difference between "that is all it says"
// and "that is all it fits".
func TestTheWindowMarksAReadThatRunsPastIt(t *testing.T) {
	short := readingCard(t, "one", "two")
	if got := window(t, short); strings.Contains(strings.Join(got, ""), "...") {
		t.Errorf("a script that fits was marked as continuing: %v", got)
	}

	long := readingCard(t, "one", "two", "three", "four", "five", "six")
	rows := window(t, long)
	last := rows[bcReadLines-1]
	if !strings.HasSuffix(strings.TrimRight(last, " "), "...") {
		t.Errorf("the last row is %q; a read that runs past the window must say so", last)
	}
	// IT CUTS TO MAKE ROOM rather than overflowing: the box truncates what it is
	// given, so a tail appended past the edge is the one thing removed.
	b := Broadcaster{width: 150, ascii: true}
	room := newCardLane(b.cardBoxWidth(), b.opts().Glyphs()).inner() - 2*len(bcCardInset)
	if w := render.Width(last) - len(bcCardInset); w > room {
		t.Errorf("the marked row is %d cells of a %d-cell window: %q", w, room, last)
	}
}

// THE MARK MEANS "MORE BELOW", NOT "THIS LINE CONTINUES", and the two must not
// look the same.
//
// FOUND BY LOOKING AT THE RENDERED CARD, not by a test: the first draft set the
// mark flush in both cases and produced "…a high near sixty-eight tomorrow.…",
// which reads as a typo rather than as a window edge. A line that had to be CUT
// does continue, so there the mark is flush on purpose.
func TestTheMoreMarkIsSetOffFromAnUncutLine(t *testing.T) {
	tail := "..."
	overflow := []string{"one", "two", "three", "four"}

	clean := window(t, readingCard(t, append(overflow,
		"Looking ahead, expect patchy fog after midnight and a high near sixty-eight tomorrow.", "and more")...))
	if got := strings.TrimRight(clean[bcReadLines-1], " "); !strings.HasSuffix(got, ". "+tail) {
		t.Errorf("an uncut line reads %q; the mark must be set off from the words it follows", got)
	}

	long := "The National Weather Service in San Diego has issued a hazardous weather outlook for " +
		"southwestern California, including the coastal and valley areas of San Diego County and the Inland Empire."
	cut := window(t, readingCard(t, append(overflow, long, "more")...))
	got := strings.TrimRight(cut[bcReadLines-1], " ")
	if !strings.HasSuffix(got, tail) {
		t.Fatalf("a cut line reads %q; it must still say there is more", got)
	}
	if strings.HasSuffix(got, " "+tail) {
		t.Errorf("a line that was CUT reads %q; the mark belongs flush there, because the line really does continue", got)
	}
}

// UP NEXT GETS THE WINDOW TOO, and it is not an accident: D-68 gives slots 0 and
// 1 the tall box, and DR-7 composes a card's words AT STANDBY — so the operator
// reads one card ahead of the one on the air.
func TestTheUpNextCardShowsItsScriptBeforeItAirs(t *testing.T) {
	var reads []string
	for _, r := range bcRegions {
		if r.reads {
			reads = append(reads, r.label)
		}
	}
	if len(reads) != 2 || reads[0] != "LIVE" || reads[1] != "UP NEXT" {
		t.Fatalf("the regions that draw a script window are %v; want LIVE and UP NEXT", reads)
	}
	c := readingCard(t, "Standing by with the forecast for Bonsall.")
	if rows := window(t, c); !strings.Contains(rows[0], "Standing by") {
		t.Errorf("a standing-by card draws no words: %q", rows[0])
	}
}

// AND IT REACHES THE SCREEN, which is a different claim from "the window builds
// the right rows" (P-1: a seam a test drives is not the delivery).
//
// WRITTEN BECAUSE A PLANT SURVIVED. `mN5` blanks `readBody`'s call to the window
// and every test above stayed green, because they all call `scriptWindow`
// directly. The wiring between the two was covered by nothing — which is exactly
// the state F-84 was in for two releases: a window built to the right size, with
// nothing putting words in it.
func TestTheScriptIsOnTheConsolesScreen(t *testing.T) {
	c := readingCard(t,
		"Now, the weather for Oceanside, California.",
		"Currently sixty-one degrees and fair.")
	admitted, err := c.To(lineup.Admitted)
	if err != nil {
		t.Fatalf("admitting: %v", err)
	}
	var l lineup.Lineup
	l, err = l.Queue(lineup.MainTrack, admitted)
	if err != nil {
		t.Fatalf("seeding: %v", err)
	}
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(LineupMsg{Lineup: l})

	got := stripANSITest(b.View().Content)
	for _, want := range []string{
		"Now, the weather for Oceanside, California.",
		"Currently sixty-one degrees and fair.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the console drew the card and not its words; %q is not on the screen", want)
		}
	}
}

// LIVE IS WHAT IS ON THE AIR, AND ON STANDBY NOTHING IS (D-84, HUM LEAD
// 2026-09-11).
//
//	"1. When the user enters Broadcaster Mode, the station is in STANDBY (DEAD
//	 AIR) … 3. The director should be choosing and populating the line-up. 4.
//	 LIVE should remain EMPTY … 5. The UP NEXT card should be getting the
//	 attention of the Composer … it will be the first thing that goes ON AIR when
//	 the human operator hits SHIFT+ENTER."
func TestOnStandbyTheLineUpStartsAtUpNextAndLiveIsEmpty(t *testing.T) {
	var l lineup.Lineup
	for _, id := range []string{"first", "second", "third"} {
		c := readingCard(t, "the words for "+id)
		c.ID, c.Subject = id, id
		c.Headline = "LOCATION REPORT • " + strings.ToUpper(id)
		admitted, err := c.To(lineup.Admitted)
		if err != nil {
			t.Fatalf("admitting %s: %v", id, err)
		}
		if l, err = l.Queue(lineup.MainTrack, admitted); err != nil {
			t.Fatalf("seeding %s: %v", id, err)
		}
	}
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(LineupMsg{Lineup: l}) // power is STOPPED: the station has not gone on air

	// LIVE holds nothing…
	if c, decided := b.slotCard(l.Projection(lineup.MainTrack), 0); decided {
		t.Errorf("LIVE holds %q on a station that is not on the air", c.ID)
	}
	// …and the head of the queue is UP NEXT, where the operator can see the words
	// that will go out first.
	if c, decided := b.slotCard(l.Projection(lineup.MainTrack), 1); !decided || c.ID != "first" {
		t.Errorf("UP NEXT holds %+v; want the head of the line-up", c)
	}
	got := stripANSITest(b.View().Content)
	if !strings.Contains(got, "the words for first") {
		t.Error("the operator cannot see what will go out first; UP NEXT drew no words")
	}
}

// AND GOING ON AIR MOVES EVERYTHING UP ONE. That is the same movement D-40
// already rules for a dropped card — "everything below moves up one" — so the
// operator has seen it before.
func TestGoingOnAirMovesTheHeadOfTheLineUpIntoLive(t *testing.T) {
	var l lineup.Lineup
	c := readingCard(t, "the words for first")
	c.ID, c.Subject, c.Headline = "first", "first", "LOCATION REPORT • FIRST"
	admitted, err := c.To(lineup.Admitted)
	if err != nil {
		t.Fatalf("admitting: %v", err)
	}
	if l, err = l.Queue(lineup.MainTrack, admitted); err != nil {
		t.Fatalf("seeding: %v", err)
	}
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(LineupMsg{Lineup: l})
	b.power = lineup.Running

	if c, decided := b.slotCard(l.Projection(lineup.MainTrack), 0); !decided || c.ID != "first" {
		t.Errorf("on air, LIVE holds %+v; want the head of the line-up", c)
	}
}

// THE OFFSET FOLLOWS THE POWER, NOT WHETHER A CARD HAPPENS TO BE PLAYING.
//
// A running station between two reads — the next card still building — has
// nothing on the air for a moment. Keying the offset off that would shunt every
// card down a row and back for the length of one build, which the operator would
// read as the console losing its place.
func TestTheLiveSlotDoesNotShuntWhileARunningStationIsBetweenReads(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b.power = lineup.Running
	if got := b.liveOffset(); got != 0 {
		t.Errorf("a running station with nothing on the air offset by %d; the power is what decides", got)
	}
	b.power = lineup.Stopped
	if got := b.liveOffset(); got != 1 {
		t.Errorf("a stopped station offset by %d; LIVE is empty and the queue starts at UP NEXT", got)
	}
}
