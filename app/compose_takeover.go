package app

// compose_takeover.go — the Card Composer for a takeover (MVS-D-77, S-7).
//
// A BURST IS ONE CARD, and this is what it says. The tone's class, the header,
// one line per alert and the closing tail are that card's CONTENT — not a
// schedule of separate cards, which is what the code modelled before the
// Broadcaster mock settled it. The operator promotes or drops the burst; there
// is nothing inside it to address separately.
//
// The composer's boundary, per S-7: it owns CONTENT and nothing else. It does
// not choose which alerts are in the burst (the Producer, `lineup.Plan`), it
// does not decide when the card is read (the Director), and it does not pace it
// (the Reader, per MVS-D-72). It turns a selection into finished words.
//
// IT COMPOSES ONCE, UP FRONT, and that is a deliberate change. Each line used to
// be composed at the moment it was rendered. The Broadcaster displays a card's
// script in UP NEXT before it airs, so text that materialises mid-read cannot be
// shown — a card has to know what it says before it says it. The only
// observable difference is a burst that spans MIDNIGHT while reading an event
// declared the day before: `spokenWhen` compares the event's date against
// `now`, so a line composed at 23:59 and read at 00:01 says "at 11:59 PM" where
// it used to say "on September 4 at 11:59 PM". A burst lasts seconds; this needs
// both the boundary and a stale event to appear at all.

import (
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

// composeTakeover turns a selection into the card's words: the tone's class, the
// header, one line per alert and the closing tail, in the order they are said.
//
// The tone is asked of the WHOLE burst, not of its first alert (MVS-D-73): the
// rail orders by rung, so the first can be the milder hazard.
//
// Head and Tail are OMITTED when the card has none — a single event carries its
// own broadcast tail inside its line, so it gets neither. An absent part is
// absent, rather than present and empty, so the Reader has nothing to skip.
func composeTakeover(lib *script.Library, fresh []globalfeed.Event, burst bool, divert int, c render.Clock, now time.Time) lineup.Script {
	sc := lineup.Script{Tone: toneClassOfEvent(worstOf(fresh)).Key()}
	if allFabricated(fresh) {
		return testCard(lib, fresh, sc)
	}
	if burst {
		if head := burstHead(lib, fresh); head != "" {
			sc.Parts = append(sc.Parts, lineup.Part{Kind: lineup.PartHead, Text: head})
		}
	}
	for _, e := range fresh { // bounded by the admitted burst (P10-02)
		sc.Parts = append(sc.Parts, lineup.Part{
			Kind: lineup.PartLine,
			Text: breakingLine(lib, e, burst, c, now),
			Ref:  e.ID, // the Reader cues the band from the producer's record
		})
	}
	if burst {
		if tail := burstTail(lib, divert); tail != "" {
			sc.Parts = append(sc.Parts, lineup.Part{Kind: lineup.PartTail, Text: tail})
		}
	}
	return sc
}

// burstTail closes a burst, and SAYS WHAT WAS LEFT OUT when anything was
// (DR-14).
//
// THE COUNT AND THE DESTINATION ARE ONE LINE. The listener is being told two
// halves of one fact — that there is more, and where it is — and splitting them
// across two utterances puts a pause between a number and the thing it counts.
//
// ONE SCRIPT, NOT TWO (HUM LEAD, 2026-09-05). The two readings differ only in
// the middle clause — "any of these alerts" against "these and N other alerts"
// — so a second file would restate the opening and the destination and let the
// pair drift. "these and 0 other alerts" stays unsayable because the count
// guards its own clause, which is what made two files look necessary.
//
// SINGULAR IS AGREED HERE, not in the template. The script tree is WORDING; the
// grammar rule is the same in every one of them, so it belongs once in code
// rather than repeated in each file that ever counts something.
func burstTail(lib *script.Library, divert int) string {
	alerts := "alerts"
	if divert == 1 {
		alerts = "alert"
	}
	return burstClosingLine(lib, divert, alerts)
}

// allFabricated reports whether every event on this card came from the ctrl+d
// window.
//
// EVERY ONE, NOT ANY ONE (FR-4.5). A burst carrying a real hazard is about that
// hazard, whatever else arrived with it: it keeps the ordinary head and tail,
// and the fabricated lines say what they are inside it. Reading "this is a test
// of the alert events system" over a live tornado warning is the one outcome
// this whole feature exists to make impossible.
func allFabricated(evs []globalfeed.Event) bool {
	for _, e := range evs { // bounded by the burst (P10-02)
		if !e.Fabricated {
			return false
		}
	}
	return len(evs) > 0
}

// testCard is the diagnostic read: head, one title per fabricated alert, one
// explanation, tail (HUM LEAD 2026-09-07).
//
// ALERT-AGNOSTIC ON PURPOSE. The same words for every category, so exercising
// the machinery never waits on someone writing suitable content for each
// hazard — and the TONE is still the injected type's own, so what is under test
// is the tone, the takeover, the band, the window and the expiry rather than
// the copy.
//
// THE EXPLANATION IS SAID ONCE. Six fabricated alerts each carrying it is a
// seven-minute read, and saying it six times is not more informative than
// saying it once. It carries no Ref, so it cues no band item: it is about the
// test, not about any one alert.
func testCard(lib *script.Library, fresh []globalfeed.Event, sc lineup.Script) lineup.Script {
	say := func(part string, e globalfeed.Event) string {
		return scriptText(lib, "test-alert", part, map[string]string{"Type": e.Title()})
	}
	if head := testHead(lib); head != "" {
		sc.Parts = append(sc.Parts, lineup.Part{Kind: lineup.PartHead, Text: head})
	}
	for _, e := range fresh { // bounded by the burst (P10-02)
		sc.Parts = append(sc.Parts, lineup.Part{Kind: lineup.PartLine, Text: testLine(lib, e), Ref: e.ID})
	}
	if explain := say("explain", fresh[0]); explain != "" {
		sc.Parts = append(sc.Parts, lineup.Part{Kind: lineup.PartLine, Text: explain})
	}
	if tail := say("tail", fresh[0]); tail != "" {
		sc.Parts = append(sc.Parts, lineup.Part{Kind: lineup.PartTail, Text: tail})
	}
	return sc
}
