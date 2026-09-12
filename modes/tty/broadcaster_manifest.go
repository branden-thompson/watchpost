package tty

// broadcaster_manifest.go — a card as a MANIFEST (D-87).
//
// HUM LEAD, 2026-09-11: "The full script on the top level card doesn't make
// sense when I can 'drill down' to read the whole thing. What DOES MAKE SENSE
// for the operator is to get a SUMMARY of what the report contains *before* it
// goes on air."
//
// SO THE CARD ANSWERS THREE QUESTIONS and stops: what is this doing right now,
// how fresh is what it will say, and what is in it. Everything longer is behind
// the handle.

import (
	"fmt"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

// cardStatus is what the card is doing, in the operator's words.
//
// DERIVED FROM THE STATE, NEVER STORED. The card already carries the only fact
// this needs, and a status written onto it at some earlier moment is a status
// that can disagree with the schedule.
//
// THE ON-AIR LINE SAYS WHAT THE OPERATOR MAY STILL DO, which is the reference's
// own wording: a live card is locked to management (D-45: only GO TO STANDBY or
// catastrophe changes it) and is still open to a takeover, because the rail
// interrupts the programme (D-82). Both halves are rules the operator would
// otherwise have to have read a document to know.
func cardStatus(c lineup.Card, decided bool) string {
	if !decided {
		return ""
	}
	switch c.State {
	case lineup.OnAir:
		return "READING  (Management Locked; Can be Taken-Over)"
	case lineup.Standby:
		return "Ready for Read-Out"
	}
	return "Scheduled; Awaiting Data"
}

// cardPulled is when the card's data was fetched, and how long ago.
//
// THE STAMP AND THE AGE TOGETHER, because they answer different questions: the
// stamp says which cycle this is, and the age is what tells the operator whether
// to trust it before putting it on the air. `BuiltAt` is the one fact behind
// both — the same fact PD-3 judges staleness on — so the card cannot show a
// fresh age beside a stale read.
func cardPulled(o render.Opts, c lineup.Card, now func() time.Time, room int) string {
	if c.BuiltAt.IsZero() {
		// NOT AN EM-DASH. Every mark on this console goes through the glyph set
		// so `--ascii` needs no special case anywhere, and a literal here was the
		// one rune the parity gate found.
		return "DATA PULLED:  " + o.Glyphs().Dash
	}
	at := c.BuiltAt
	stamp := "DATA PULLED:  " + at.Format("Monday January 2, 2006 @ 15:04:05")
	if now == nil {
		return stamp
	}
	// THE AGE IS ANCHORED RIGHT, which is the reference and is also what makes
	// it scannable: the stamps differ in length card to card, and an age that
	// floated after them would sit in a different column on every row — so the
	// one number the operator sweeps the list for would be the one thing they
	// had to hunt.
	age := "(" + shortAgo(now().Sub(at)) + ")"
	if gap := room - render.Width(stamp) - render.Width(age); gap > 0 {
		return stamp + strings.Repeat(" ", gap) + age
	}
	return stamp + " " + age
}

// shortAgo is the reference's own form of an elapsed span: "2 MIN AGO".
//
// IT SHARES THE DASHBOARD'S THRESHOLDS, NOT ITS WORDS. `agoWords` decides when
// "just now" stops being true and when minutes become hours; this decides how
// to say it on a console that shouts. Two functions with two ladders would be
// two answers to how old a thing is — so the ladder is asked once, through the
// one that already owns it, and only the wording differs.
func shortAgo(d time.Duration) string {
	words := strings.ToUpper(agoWords(d))
	words = strings.ReplaceAll(words, "MINUTES", "MIN")
	words = strings.ReplaceAll(words, "MINUTE", "MIN")
	words = strings.ReplaceAll(words, "HOURS", "HR")
	return strings.ReplaceAll(words, "HOUR", "HR")
}

// manifestHeading is the contents table's header row.
//
// TWO COLUMNS, AND THE SECOND IS DELIBERATELY VAGUE. The reference calls it
// "RANGE / INCIDENTS" because the sources answer different questions — a
// forecast covers a span of days, a fire report counts hotspots — and a heading
// that named one would be wrong for the others.
func manifestHeading(room int) string {
	return manifestRow("##.", "NAME", "RANGE / INCIDENTS", room)
}

// manifestRow lays one contents line out: a number, a name that gives way, and
// the detail right where the heading promised it.
//
// THE NAME IS WHAT TRUNCATES. The number is an index and the detail is the
// answer; a name cut short is still recognisable, and the other two are not.
func manifestRow(no, name, detail string, room int) string {
	if room < 20 {
		return render.TruncateCells(no+" "+name, max(0, room))
	}
	// THE DETAIL COLUMN IS THE REFERENCE'S: the name runs to column 46 of the
	// card's text and the detail begins there.
	nameRoom := max(8, room*46/78-4)
	head := no + " " + render.PadTo(render.TruncateCells(name, nameRoom), nameRoom)
	return render.PadTo(head+" "+detail, room)
}

// manifestRows is what the card's read contains, one line per source.
//
// EMPTY UNTIL THE COMPOSER REPORTS IT (D-87 stage 2). The Composer builds a
// report's segments and keeps what it counted to itself — "3 Hotspots / 10
// incidents" is computed while composing and thrown away — so the manifest is a
// new output of it rather than something the console can derive. The table is
// built to the reference's shape now so the rows have somewhere to land, which
// is the same order F-84's window was built in.
func (b Broadcaster) manifestRows(c lineup.Card, room int) []string {
	rows := make([]string, 0, bcReadLines)
	for i, m := range c.Contents { // bounded by the manifest (P10-02)
		if i >= bcReadLines {
			break
		}
		rows = append(rows, bcCardInset+manifestRow(fmt.Sprintf("%02d.", i+1), m.Name, m.Detail, room))
	}
	for len(rows) < bcReadLines { // bounded by the card's height (P10-02)
		rows = append(rows, "")
	}
	return rows
}
