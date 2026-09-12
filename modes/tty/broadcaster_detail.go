package tty

// broadcaster_detail.go — the card's own window (D-88, F-97).
//
// HUM LEAD, UAT 2026-09-11: "Pressing [1] doesn't open the details modal."
//
// IT IS THE OTHER HALF OF THE MANIFEST, and D-87 is what made it load-bearing.
// The card stopped being a transcript on the HUM LEAD's own reasoning — "the full
// script on the top level card doesn't make sense when I can 'drill down' to read
// the whole thing" — so the drill-down is the thing the manifest defers TO. A
// manifest with nowhere to go promises a place that does not exist, and the words
// a station is about to say aloud would be reachable from nowhere at all.
//
// THE RULING IT SERVES IS OLDER THAN D-87 (HUM LEAD, 2026-09-11): "the operator
// should be able to inspect the full text of the report by keying the number
// position of either the live card [0] or the UP Next [1] card."
//
// THE CONSOLE BUILDS THE BODY; THE DASHBOARD DRAWS IT. That split is D-56's,
// stated for the diagnostics window and true for the same reasons here: "ONE
// WINDOW, NOT TWO … giving the console its own would be a second injector UI, a
// second confirmation, and two places for the wording to drift." Four gates hang
// off the Dashboard's window set — reachability at the floor, the margin survey,
// the memo-completeness walk and the single-value exclusivity of `Dashboard.modal`
// — and a console-private window would have been outside every one of them on the
// day it shipped.

import (
	"slices"
	"strconv"
	"strings"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

// bcDetailRoom is the column budget the window's tables are laid out in.
//
// 77, DERIVED FROM THE WINDOW AND NOT CHOSEN: the card's window class is 85 wide
// at its floor (modalWidth), two of those columns are its borders, and the margin
// survey requires three clear columns inside each border — 85 - 2 - 3 - 3. The
// first draft used the manifest's own 78 and the survey reported the window
// running into its right border by exactly the one column this arithmetic
// accounts for.
//
// THE TABLE DOES NOT STRETCH, and that is deliberate: a contents list whose
// columns moved with the terminal is a list the operator re-learns on every
// machine. The prose below it wraps, because prose has no columns to keep.
const bcDetailRoom = 77

// cardDetail is the window for the card at a slot handle: its name, its body,
// and whether that slot holds a card at all.
//
// AN EMPTY SLOT OPENS NOTHING. The handle is drawn on an undecided slot because
// the slot is addressable (slotRows), but there is no report to inspect — and a
// window that opened onto "waiting for the line-up" would be a way of asking the
// operator to read the absence of a card.
func (b Broadcaster) cardDetail(handle int) (string, func(render.Opts) (string, []string), bool) {
	main := b.lineup.Projection(lineup.MainTrack)
	if len(main) > MainTrackSlots { // the rolling window, as the frame draws it (FR-3.1)
		main = main[:MainTrackSlots]
	}
	c, decided := b.slotCard(main, handle)
	if !decided {
		return "", nil, false
	}
	// THE ID IS THE IDENTITY AND EVERYTHING DRAWN IS A FUNCTION OF THE WINDOW'S
	// OPTS — including the title, whose separator is a glyph (cardTitle) and whose
	// ASCII form is therefore not knowable here.
	return c.ID, func(o render.Opts) (string, []string) {
		return cardTitle(c, o.Glyphs()), b.detailBody(o, c)
	}, true
}

// detailBody is everything the window says about one card, in the order an
// operator asks it: what it is doing, how fresh it is, who says it, what it is
// made of, and then the words themselves.
//
// THE HEAD ROWS ARE THE CARD'S OWN (D-87). `cardStatus` and `cardPulled` are the
// same two functions the card draws with, called with more room — so the window
// and the card it opened from cannot disagree about the state or the age of the
// report, which is the one thing the drill-down exists to be trusted about.
func (b Broadcaster) detailBody(o render.Opts, c lineup.Card) []string {
	rows := []string{
		"STATUS:  " + cardStatus(c, true),
		cardPulled(o, c, b.clock, bcDetailRoom),
		"READ BY:  " + detailReadBy(c),
		"PROPOSED BY:  " + detailOrigin(c),
	}
	rows = append(rows, "", "READ CONTENTS", manifestHeading(bcDetailRoom))
	rows = append(rows, detailContents(c)...)
	rows = append(rows, "", "FULL READ")
	rows = append(rows, detailScript(c)...)
	rows = append(rows, "", o.Controls("   ",
		render.Ctl("esc", "Close"), render.Ctl("↑↓", "Scroll")))
	return indentBody(rows)
}

// detailReadBy is the voice that will say this, or that nothing has resolved one.
//
// `N/A` IS THE CONSOLE'S OWN WORD FOR AN UNRESOLVED VOICE (Card.ReadBy: "Empty
// means unresolved — the Broadcaster surface renders that as `N/A`"), so this
// says what the surface already says rather than inventing a second phrasing for
// the same absence.
func detailReadBy(c lineup.Card) string {
	if c.ReadBy == "" {
		return "N/A"
	}
	return c.ReadBy
}

// detailOrigin is who proposed the card, in the registry's own words.
//
// IT IS ON THE WINDOW AND NOT ON THE CARD because the card already says it in
// COLOUR — "Operator requested cards get a slightly different Tint then Producer
// created cards" (HUM LEAD, 2026-09-11, D-86) — and a tint tells the operator
// which pile a card came from at a glance while saying nothing they can quote.
// The window is where a glance becomes a fact.
func detailOrigin(c lineup.Card) string {
	if name := c.Origin.String(); name != "" {
		return name
	}
	return "N/A"
}

// detailContents is the WHOLE manifest, which is the difference between this and
// the card.
//
// THE CARD SHOWS FOUR (bcReadLines) because four is what a location report has
// and because the card's height is fixed — "the height is the same whether there
// is a script or not", or every card below it moves when one report gains a
// source. The window has a scroll rail and no such obligation, so a fifth source
// is visible here and nowhere else.
func detailContents(c lineup.Card) []string {
	if len(c.Contents) == 0 {
		// A CARD WITH NO MANIFEST IS WAITING ON THE COMPOSER, and saying so is
		// not the same as drawing an empty table. `Contents` is empty until the
		// words come home (D-87 stage 2), which is a moment the operator watches
		// pass — so it reads as a state rather than as a fault.
		return []string{"(the Composer has not reported the contents yet)"}
	}
	rows := make([]string, 0, len(c.Contents))
	for i, m := range c.Contents { // bounded by the manifest (P10-02)
		rows = append(rows, manifestRow(pad2(i+1), m.Name, m.Detail, bcDetailRoom))
	}
	return rows
}

// pad2 is a two-digit index, matching the card's own `%02d.` column.
//
// SHARED WITH NOTHING, DELIBERATELY: the card formats its number inline with
// `fmt.Sprintf` and this is the only other place that needs the form, so the
// second caller is what earns the helper (modularity standard) and both now go
// through it.
func pad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n) + "."
	}
	return strconv.Itoa(n) + "."
}

// detailScript is the words, one row per part, in the order they are said.
//
// THE PARTS AND NOT `Script.Text()`. A takeover's shape is what the listener
// hears — head, lines, tail (MVS-D-72) — and flattening it to one blob would show
// the operator a paragraph where the station will speak a structure. `Script`
// exists in parts for exactly this reason: "The parts also ARE the display …
// One representation serves both, so what a listener hears and what an operator
// reads cannot drift apart."
//
// THE PANEL WRAPS THEM. A sentence is emitted whole and `wrapModal` folds it to
// the window's width, which is the same path every other window's prose takes —
// wrapping here would fold it to a width the window may not be drawn at.
func detailScript(c lineup.Card) []string {
	if c.Script.Empty() {
		// THE TWO REASONS A CARD HAS NO WORDS ARE DIFFERENT, and the operator
		// acts differently on them: a report's script arrives at standby (DR-7),
		// so an empty one is a card still being composed; a card that is on the
		// air with nothing is a fault.
		if c.State == lineup.OnAir {
			return []string{"(on the air with no script — this is a fault)"}
		}
		return []string{"(the Composer has not written the read yet)"}
	}
	rows := []string{}
	for _, p := range c.Script.Parts { // bounded by the script (P10-02)
		if strings.TrimSpace(p.Text) == "" {
			continue
		}
		rows = append(rows, p.Text, "")
	}
	// THE TRAILING BLANK IS DROPPED: the body adds its own air before the chips,
	// and two blanks read as a missing row rather than as spacing.
	if n := len(rows); n > 0 {
		rows = rows[:n-1]
	}
	return rows
}

// showCard hands the Dashboard a Broadcaster card to draw, and opens the window.
//
// THE GENERATION MOVES ONLY ON A REAL CHANGE, which is what lets this be called
// on every update without defeating the memo. A card the operator is reading can
// re-hydrate under them — `RefreshAfter` is half of `StaleAfter` — and a window
// that kept its first frame while the data moved is exactly the freeze F-30 was
// filed for. Re-handing on every update is how the window cannot lag; comparing
// before bumping is how it does not redraw for nothing.
func (d Dashboard) showCard(id string, rows func(render.Opts) (string, []string), at render.Opts) Dashboard {
	if d.cardID == id && d.cardRows != nil && sameCard(d.cardRows, rows, at) {
		return d
	}
	d.cardID, d.cardRows, d.cardGen = id, rows, d.cardGen+1
	return d
}

// sameCard reports whether two renderers would draw the same window at `at`.
func sameCard(a, b func(render.Opts) (string, []string), at render.Opts) bool {
	at1, a1 := a(at)
	at2, b1 := b(at)
	return at1 == at2 && slices.Equal(a1, b1)
}

// cardLines is the open card's body, asked of the console at THIS window's opts.
//
// NIL-SAFE, because `modalCard` can be opened by a fixture that has not handed a
// renderer over — and a window that draws nothing is a failure the gates report
// as such (fixtureFor) rather than a panic in the middle of a frame.
func (d Dashboard) cardLines(o render.Opts) []string {
	if d.cardRows == nil {
		return nil
	}
	_, body := d.cardRows(o)
	return body
}

// cardTitleOf is the open card's title, in the window's own glyphs.
func (d Dashboard) cardTitleOf(o render.Opts) string {
	if d.cardRows == nil {
		return ""
	}
	title, _ := d.cardRows(o)
	return title
}

// indentBody gives every row the window's third column.
//
// THE PANEL SUPPLIES TWO AND THE SURVEY WANTS THREE — measured, not chosen:
// TestEveryWindowClearsItsMargins reported this window leading at 2 the first
// time it ran, which is the same finding it made against four windows in
// 2026-09-05. Applied HERE, once, rather than written into each row: a margin
// spelled at fifteen call sites is fifteen places for it to drift.
//
// AND THE TRAILING PAD COMES OFF. `manifestRow` fills its row to `room` because
// on a CARD the row is a cell of a box that has to be full; inside a window it is
// invisible except to the margin survey, which measured it as content and
// reported the window running into its right border. The same two facts with two
// different consumers, so the consumer that does not want the fill removes it.
func indentBody(rows []string) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows { // bounded by the body (P10-02)
		if r = strings.TrimRight(r, " "); r == "" {
			out = append(out, r)
			continue
		}
		out = append(out, " "+r)
	}
	return out
}
