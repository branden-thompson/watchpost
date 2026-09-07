package app

import (
	"strings"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

var composeNow = time.Date(2026, 9, 4, 15, 0, 0, 0, time.UTC)

func ev(id, typ, loc string) globalfeed.Event {
	// Source is not decoration: the header names the agencies that declared the
	// alerts, and with none it composes to "" — which is correct, and which a
	// thin fixture silently turns into a passing test about nothing.
	return globalfeed.Event{ID: id, Type: typ, Location: loc, Source: "NWS", At: composeNow.Add(-time.Hour)}
}

// MVS-D-77 — A BURST IS ONE CARD, AND THIS IS WHAT IT SAYS.
//
// The Composer owns content and nothing else (S-7). These pin the boundary: it
// turns a selection into finished words, in read order, and decides nothing
// about which alerts are in the burst or when it is read.
func TestTheComposerTurnsASelectionIntoOneCardsWords(t *testing.T) {
	fresh := []globalfeed.Event{
		ev("a", "Tornado Warning", "Oceanside, CA"),
		ev("b", "Flood Advisory", "Vista, CA"),
		ev("c", "Fire Weather Watch", "Fallbrook, CA"),
	}
	sc := composeTakeover(nil, fresh, true, 0, render.Clock12, composeNow)

	lines := sc.Lines(lineup.PartLine)
	if len(lines) != len(fresh) {
		t.Fatalf("one line per alert, got %d for %d events", len(lines), len(fresh))
	}
	// IN READ ORDER, and carrying the event: the Reader cues the band from it,
	// so a line separated from its event puts the wrong callout on screen.
	for i, a := range lines {
		if a.Ref != fresh[i].ID {
			t.Errorf("line %d belongs to %q, want %q", i, a.Ref, fresh[i].ID)
		}
		if strings.TrimSpace(a.Text) == "" {
			t.Errorf("line %d has no words", i)
		}
	}
	if len(sc.Lines(lineup.PartHead)) != 1 || len(sc.Lines(lineup.PartTail)) != 1 {
		t.Errorf("a burst carries one header and one closing tail: %v", sc.Parts)
	}
	// AND THEY ARE IN THE ORDER THEY ARE SAID: head, then the lines, then tail.
	if sc.Parts[0].Kind != lineup.PartHead || sc.Parts[len(sc.Parts)-1].Kind != lineup.PartTail {
		t.Errorf("the script is in read order, got %v first and %v last", sc.Parts[0].Kind, sc.Parts[len(sc.Parts)-1].Kind)
	}
}

// A SINGLE EVENT CARRIES ITS OWN TAIL, so it gets neither a header nor a
// closing line. Adding them would make one alert sound like a bulletin.
func TestASingleEventComposesNoHeaderAndNoTail(t *testing.T) {
	sc := composeTakeover(nil, []globalfeed.Event{ev("a", "Tornado Warning", "Oceanside, CA")}, false, 0, render.Clock12, composeNow)
	if len(sc.Lines(lineup.PartLine)) != 1 {
		t.Fatalf("one line, got %d", len(sc.Lines(lineup.PartLine)))
	}
	if len(sc.Lines(lineup.PartHead)) != 0 || len(sc.Lines(lineup.PartTail)) != 0 {
		t.Errorf("a single event has no head and no tail: %v", sc.Parts)
	}
}

// THE TONE IS THE WORST EVENT'S, NOT THE FIRST'S (MVS-D-73).
//
// The rail orders by rung, so the first card can be the milder hazard — a fresh
// quake is rung 2 and a tornado warning rung 3. Taking the tone from the head of
// the list gave the listener the wrong chime for the burst they were about to
// hear, which is a rule that happened to hold rather than a rule.
func TestTheToneComesFromTheWorstEventNotTheFirst(t *testing.T) {
	mild := ev("a", "Flood Advisory", "Vista, CA")
	worst := ev("b", "Tornado Warning", "Oceanside, CA")

	lead := composeTakeover(nil, []globalfeed.Event{mild, worst}, true, 0, render.Clock12, composeNow)
	alone := composeTakeover(nil, []globalfeed.Event{worst}, false, 0, render.Clock12, composeNow)
	if lead.Tone != alone.Tone {
		t.Errorf("the burst's tone is the worst event's (%v), got %v", alone.Tone, lead.Tone)
	}
	// And it is NOT the mild one's, or this passes for the wrong reason.
	if mildAlone := composeTakeover(nil, []globalfeed.Event{mild}, false, 0, render.Clock12, composeNow); lead.Tone == mildAlone.Tone && alone.Tone != mildAlone.Tone {
		t.Errorf("the tone followed the first event (%v) rather than the worst", mildAlone.Tone)
	}
	if lead.Tone != cast.Classify("Tornado Warning").Key() {
		t.Errorf("the burst's tone is the tornado's, got %v", lead.Tone)
	}
}

// THE CARD KNOWS ITS WORDS BEFORE IT IS READ, which is why composition moved
// here at all: the Broadcaster shows a card's script in UP NEXT before it airs,
// and text that materialises mid-read cannot be displayed.
func TestTheCardIsFinishedBeforeItIsRead(t *testing.T) {
	fresh := []globalfeed.Event{ev("a", "Tornado Warning", "Oceanside, CA"), ev("b", "Flood Advisory", "Vista, CA")}
	sc := composeTakeover(nil, fresh, true, 0, render.Clock12, composeNow)
	for i, p := range sc.Parts {
		if p.Text == "" {
			t.Errorf("part %d has no words before the read", i)
		}
	}
	if sc.Empty() {
		t.Error("a composed burst says something")
	}
}
