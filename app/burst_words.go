package app

// burst_words.go — the WORDS a takeover says, and the sentences the tape shows.
//
// SPLIT OUT OF ticker.go (2026-09-06), a pure move.
//
// THE ROLE MODEL SAID THIS ALREADY; THE FILES DID NOT. MVS-D-77 and S-7 give
// the Composer (app/compose_takeover.go) the card's CONTENT and give the
// Producer (app/ticker.go) what ARRIVED — "it turns a selection into finished
// words" against "it does not own words or pacing". But composeTakeover was ~18
// lines of structure over four helpers that lived in the Producer's file, so
// the four roles were a DOC split and not a FILE split (red team 2026-09-05,
// Junior-Dev 10). Now a reader who opens the Composer can follow it without
// leaving the concern.
//
// The Producer keeps the TAPE (itemsOf, laneItems, tapeItems, tickerCategory):
// it publishes the marquee, so those rows are its own output. What moved is the
// language — who declared an alert, how a headline reads, how a time is spoken.

import (
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/script"
	"github.com/branden-thompson/watchpost/platform/render"
)

// burstAgencies is the head's list of who declared the alerts: each agency ONCE,
// in the order its first alert appears, joined the way a person would say them.
//
// Deduplicated, because four alerts from two offices is two names, not four —
// and reading "the National Weather Service" twice would tell a listener there
// are two of it. A source with no spoken name is left out rather than guessed
// at.
func burstAgencies(evs []globalfeed.Event) string {
	var names []string
	seen := map[string]bool{}
	for _, e := range evs {
		n := e.Agency()
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
	}
	return spokenList(names)
}

// spokenList joins names the way a person says a list: "A", "A and B",
// "A, B, and C". Extracted at the second caller (the masthead's provider list,
// F-24) rather than written twice.
func spokenList(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	}
	return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
}

// burstHead is the one opening of a multi-event burst: who declared what is
// about to be read. "" when no source could be named, in which case the burst
// simply starts with its first alert — a head that named nobody would be worse
// than none.
func burstHead(lib *script.Library, evs []globalfeed.Event) string {
	agencies := burstAgencies(evs)
	if agencies == "" {
		return ""
	}
	return scriptText(lib, "breaking", "burst-head", map[string]string{"Agencies": agencies})
}

// burstTitle is one alert inside a burst: the TITLE, where, and when.
//
// Not the full sentence the single-event path reads. The head has already said
// these were declared, so repeating "has been declared for" once per alert turns
// four alerts into four copies of the same sentence — which is what an outbreak
// sounded like before the head existed.
func burstTitle(e globalfeed.Event, c render.Clock, now time.Time) string {
	s := e.Title() + " for " + e.Location + ", " + spokenWhen(c, e.At, now)
	if !e.Until.IsZero() {
		s += ", until " + c.Spoken(e.Until.Local())
	}
	return render.Plain(s)
}

// breakingLine is the line to speak for one event: a single event carries its
// own broadcast tail; a burst event's line has none (the tail comes once).
func breakingLine(lib *script.Library, e globalfeed.Event, burst bool, c render.Clock, now time.Time) string {
	if burst {
		return scriptText(lib, "breaking", "burst-line", map[string]string{"Line": burstTitle(e, c, now)})
	}
	return alertNarration(lib, e, c, now)
}

// worstOf is the most serious event of a burst — the one the tone warns about.
//
// SEVERITY FIRST, THEN THE LOUDER TONE (MVS-D-73). `Severity` is a three-value
// colour tier, so two hazards of different KINDS share one constantly — and
// ranking on it alone left the tone to whichever the rail happened to order
// first. A red hurricane and a red tornado warning would sound the low sweep or
// the EAS dual-tone depending on nothing a listener could reason about. The
// second key is cast.Class.ToneRank, the one carrier of "which sound says
// listen now".
//
// Deterministic to the end: with both keys equal the earlier card wins, and the
// rail's order is itself deterministic. The two classes sharing a preset sound
// identical anyway, so that last tie is inaudible either way.
func worstOf(evs []globalfeed.Event) globalfeed.Event {
	worst := evs[0]
	for _, e := range evs[1:] { // bounded by the burst (P10-02)
		if e.Severity != worst.Severity {
			if e.Severity > worst.Severity {
				worst = e
			}
			continue
		}
		if cast.Classify(e.Type).ToneRank() > cast.Classify(worst.Type).ToneRank() {
			worst = e
		}
	}
	return worst
}

// tapeText is one alert on the ticker tape: the specific type and tied
// location, when it happened (the class verb — declared/recorded/reported), and
// its active window's end when it has one (HUM LEAD 2026-08-27, #1). The lane
// label already names the category, so the line names the specific alert.
func tapeHead(e globalfeed.Event) string {
	s := e.Title() + " · " + e.Location // a named storm reads by name (SAM-D-14)
	// Feed text reaches the terminal here — strip any escape/control sequences a
	// hostile or compromised feed could smuggle in (OSC-52 clipboard, title
	// spoof, CSI frame corruption), the same defence the snapshot path applies
	// (red-team 0.12.0 P4 F1 — S-F6). ONE line: a newline or tab in a feed
	// name would add rows to the frame (REVIEW R5-C-01); the tape's own
	// two-space gap stays (PlainLine would squeeze it).
	return strings.NewReplacer("\n", " ", "\t", " ").Replace(render.Plain(s))
}

// eventNarration is one event's spoken line: the sentence, when it happened,
// and (for an alert with a window) until when — no tail (HUM LEAD script).
// Through render.Plain: a provider-supplied storm NAME now reaches the
// synthesiser (0.13.0), and the tape already strips at tapeText — the speech
// path must too (S-F6). ExpandStates in AlertNarration reads "VA" as "Virginia".
func eventNarration(e globalfeed.Event, c render.Clock, now time.Time) string {
	s := e.Sentence() + " " + spokenWhen(c, e.At, now)
	if !e.Until.IsZero() {
		s += " until " + c.Spoken(e.Until.Local())
	}
	return render.Plain(s)
}

// alertNarration is a SINGLE event's full narration: its line in the
// "breaking.single" script (the tail directing the listener to the window —
// 0.13.0, SAM-D-26 N-1 — lives in the script file, not here).
func alertNarration(lib *script.Library, e globalfeed.Event, c render.Clock, now time.Time) string {
	return scriptText(lib, "breaking", "single", map[string]string{"Line": eventNarration(e, c, now)})
}

// burstClosingLine is the one tail after a multi-event burst ("breaking.burst-closing").
func burstClosingLine(lib *script.Library, divert int, alerts string) string {
	return scriptText(lib, "breaking", "burst-closing", map[string]any{"Divert": divert, "Alerts": alerts})
}

// scriptText renders a script part for the air (script.Library.Say: Plain'd;
// a missing or broken script reads as silence rather than a crash — the
// visuals still run).
func scriptText(lib *script.Library, report, part string, data any) string {
	return lib.Say(report, part, data)
}

// spokenWhen is when an event happened, READ ALOUD: "at Four Fifty PM", or "on
// August 27 at Four Fifty PM" when it was not today.
//
// The whole phrase, preposition included, because the preposition changes with
// the answer — a caller that wrote " at " itself would say "at on August 27".
//
// Composed AT NARRATION TIME, which is why this one still takes the clock: the
// line is spoken once, when the alert fires, so it reads the preference in
// force at that moment. The TAPE is the opposite case — it is on screen for
// minutes, so it carries the facts and is formatted every frame.
func spokenWhen(c render.Clock, t, now time.Time) string {
	local := t.Local()
	y1, m1, d1 := local.Date()
	y2, m2, d2 := now.Local().Date()
	if y1 == y2 && m1 == m2 && d1 == d2 {
		return "at " + c.Spoken(local)
	}
	return "on " + local.Format("January 2") + " at " + c.Spoken(local)
}
