package tty

// ticker.go — the global event ticker marquee (0.12.0): a three-row band above
// the radio panel that ticker-tapes the world's largest active hazard events.
// The active alerts are grouped into CATEGORIES (earthquakes, tropical
// cyclones, warnings, watches); one category's alerts scroll as a continuous
// •-separated tape, and the band rotates to the next non-empty category every
// 90 s (HUM LEAD 2026-08-27, #5/#6). The band background is a FIXED colour per
// category. A UI-level type — the app maps domains/globalfeed onto it, so
// modes/tty stays decoupled from the domain (the snapshot-only rule).

import (
	"fmt"
	"github.com/branden-thompson/watchpost/platform/category"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
)

// TickerSeverity is retained for ordering within a category (the app sorts
// most-recent-most-severe); the band colour is now per-category, not per-tier.
type TickerSeverity int

const (
	TickerYellow TickerSeverity = iota
	TickerOrange
	TickerRed
)

// TickerCategory is the marquee lane an event belongs to; the band rotates
// through the non-empty lanes in this declared order (HUM LEAD 2026-08-27).
// TickerCategory is a lane on the band — the same thing as a window tab, named
// for where it is shown. An alias, so the two cannot drift.
type TickerCategory = category.Category

const (
	CatDisasters = category.Disasters // "Disasters": quakes and the other non-weather hazards
	CatMarine    = category.Marine    // "Marine": NHC storms and the marine products beside them
	CatWarning   = category.Warnings
	CatWatch     = category.Watches
	CatAdvisory  = category.Advisories
	CatStatement = category.Statements
	CatEmergency = category.Emergency
)

// tickerCatOrder is the rotation — the registry's own, not a list kept beside
// it. A lane missing from a hand-kept order never reached the band however many
// alerts it held (F-21).
func tickerCatOrder() []TickerCategory { return category.Lanes() }

// TickerItem is one active alert as the marquee shows it. The app composes Text
// ("Tornado Warning · the Oklahoma City area  declared 3:42 PM · expires
// 4:15 PM") from a globalfeed.Event; Category picks the lane and the colour.
// The item carries the FACTS, not a finished line. It used to arrive
// pre-formatted from the app, which meant the times on it were written at the
// moment the ticker cycled — so changing the clock in Settings did nothing until
// the next cycle up to two minutes later, and the tape sat there in the old
// format (HUM LEAD, UAT 2026-08-30: "there's a delay").
//
// Formatting here fixes that by construction: the tape is composed from
// render.Opts every frame, so the clock preference reaches it the same way the
// theme does — immediately, because there is nothing older to repaint.
type TickerItem struct {
	ID       string // the source event id
	Category TickerCategory
	Head     string         // "<Type> · <Location>" — the part with no time in it
	Verb     string         // how it happened: declared · recorded · reported · issued
	At       time.Time      // when it happened
	Until    time.Time      // when its window ends; zero = none
	Severity TickerSeverity // ordering within the lane (set by the app)

	// Test marks an item the ctrl+d window fabricated (FR-4.4). The BAND says
	// so — see the lane chrome — because a photograph of the marquee showing a
	// tornado warning is indistinguishable from a real one otherwise.
	Test bool
}

// tapeLine is one alert as the tape reads it, in the LISTENER'S clock and the
// LISTENER'S ZONE: "<Type> · <Location>  declared 3:42 PM · expires 4:15 PM".
//
// .Local() on both, and it is not optional. The feeds publish UTC, and the
// spoken line localises — so without it the band said "0026" for an event the
// radio was calling "Seventeen Twenty-Six Hours", the same instant seven hours
// apart. It went missing when the formatting moved
// here from the app, which localised.
//
// The Clock methods deliberately do NOT localise: the alert list formats in the
// LOCATION's zone, because a tide or an expiry happens where the weather is.
// Choosing the zone is the caller's job, which is why this one has to say so.
func (d Dashboard) tapeLine(o render.Opts, it TickerItem) string {
	// THE ONE PLACE THE TAPE'S TEXT CROSSES THE GLYPH BOUNDARY (F-47). Head is
	// built by the Producer (app/ticker.go:611), which composes DATA and has no
	// view options, so its separator is always the middot; the tape is where a
	// frame learns whether it is being drawn under --ascii. tickerBullet already
	// knew this — the separator BETWEEN items had both forms while the one INSIDE
	// an item did not.
	dot := o.Glyphs().Dot
	head := it.Head
	if o.ASCII {
		head = strings.ReplaceAll(head, "·", dot)
	}
	s := head + "  " + it.Verb + " " + o.Clock.Since(it.At.Local(), d.clock())
	if !it.Until.IsZero() {
		s += " " + dot + " expires " + o.Clock.Since(it.Until.Local(), d.clock())
	}
	if it.Test {
		return testEventMark + " " + s + " " + testEventMark
	}
	return s
}

// tickerBullet separates alerts on the tape (a middot; a plain * under --ascii).
// Both forms are the same cell width, so the scroll loop length is stable.
const (
	tickerBulletDot   = "   •   "
	tickerBulletASCII = "   *   "
)

func tickerBullet(o render.Opts) string {
	if o.ASCII {
		return tickerBulletASCII
	}
	return tickerBulletDot
}

// tickerRightReserve keeps a few cells clear at the right of the band, where
// the multi-alert circle viz will render (HUM LEAD 2026-08-27); the mute
// control lives in the header controls, not the band.
const tickerRightReserve = 4

// testEventMark is what the band says when what it shows was fabricated.
//
// PREPENDED AND POSTPENDED, ON THE ITEM (HUM LEAD 2026-09-07). The tape
// scrolls: a marker at one end only is off-window half the time, and a marker
// at both ends means the item cannot be on screen without one of them. It sat
// in the band's top row first, as lane chrome, on the argument that an 18-cell
// prefix per item at the 80-column floor makes the marker the majority of the
// tape — the ruling overrides that argument, and it is the ITEM that is
// fabricated rather than the lane it happens to be filed under.
//
// TWO ASTERISKS, NOT THREE, and the ruling's own wording is the reason twice
// over: it is what was written, and at 14 cells it fits the severe window's
// EVENT column ahead of a product where "*** TEST EVENT ***" did not — that
// column truncates, and a mark that pushes the product out of its own column
// is a mark that hides what it is marking.
const testEventMark = "**TEST EVENT**"

// tickerMarquee renders the ticker as a THREE-row band: a category-coloured
// blank row above and below the tape row, so the band breathes and absorbs the
// header/radio spacers rather than growing the frame. Empty ⇒ a persistent
// muted band (never hidden, so the layout never jitters). The current lane's
// label sits at the left; its alerts ticker-tape across the rest.
func (d Dashboard) tickerMarquee(o render.Opts) string {
	width := o.Width

	// A breaking-news takeover overrides the tape: one event, centred, in its
	// lane colour, until the sequence ends (HUM LEAD 2026-08-27).
	if d.breaking != nil {
		it := *d.breaking
		tones := render.Tok(tickerCatBG(it.Category)) + ";" + render.Tok(render.TickerFG)
		content := render.TintRaw(centerText(d.tapeLine(o, it), width), tones)
		blank := render.TintRaw(strings.Repeat(" ", width), tones)
		return blank + "\n" + content + "\n" + blank
	}

	right := tickerRightReserve
	mid := "  no active severe events"
	bg, fg := render.GroupSectionBG, render.TickerMutedFG // the muted band matches the RECENT/SEARCHED group header
	if cats := d.tickerCategories(); len(cats) > 0 {
		cur := cats[d.tickerCatIdx%len(cats)]
		items := d.tickerLane(o, cur)
		left := fmt.Sprintf("  %s  %d %s  ", cur.Label(), len(items), o.Glyphs().Alert)
		win := max(1, width-render.Width(left)-right)
		tape := strings.Join(items, tickerBullet(o))
		mid = left + scrollWindow(tape, d.tickerScroll, win, tickerBullet(o))
		bg, fg = tickerCatBG(cur), render.TickerFG
	}
	tones := render.Tok(bg) + ";" + render.Tok(fg)
	content := render.TintRaw(render.PadTo(mid, width), tones)
	blank := render.TintRaw(strings.Repeat(" ", width), tones) // the band's top and bottom rows
	return blank + "\n" + content + "\n" + blank
}

// tickerCategories are the non-empty lanes, in rotation order.
func (d Dashboard) tickerCategories() []TickerCategory {
	var present []TickerCategory
	for _, c := range tickerCatOrder() {
		for _, it := range d.ticker {
			if it.Category == c {
				present = append(present, c)
				break
			}
		}
	}
	return present
}

// tickerLane is every alert in a lane, as the tape reads it, in the app's order.
func (d Dashboard) tickerLane(o render.Opts, c TickerCategory) []string {
	var out []string
	for _, it := range d.ticker {
		if it.Category == c {
			out = append(out, d.tapeLine(o, it))
		}
	}
	return out
}

// tickerCatBG is the band background for a lane — FIXED per category (HUM LEAD
// colour pass, 2026-08-27): Earthquakes = Red, Warnings = Orange, Watches =
// Yellow, Tropical = Blue.
func tickerCatBG(c TickerCategory) render.Token { return category.Of(c).BandTone }

// centerText centres text within width (clipping by DISPLAY WIDTH when it is
// wider) — the breaking-news event sits in the middle of the band. A
// non-positive width yields empty (a negative slice bound would panic — P4 F3).
func centerText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	w := render.Width(text)
	if w >= width {
		return clipToWidth(text, width)
	}
	left := (width - w) / 2
	return render.PadTo(strings.Repeat(" ", left)+text, width)
}

// clipToWidth truncates text to at most width display cells (wide runes count
// as their cell width, so the result never overflows the band).
func clipToWidth(text string, width int) string {
	var b strings.Builder
	used := 0
	for _, r := range text {
		cw := render.Width(string(r))
		if used+cw > width {
			break
		}
		b.WriteRune(r)
		used += cw
	}
	return b.String()
}

// scrollWindow returns a width-cell window into text at the given offset; a
// line that fits is left-aligned and padded, a longer line scrolls (text + a
// wrap gap, looping) — the ticker-tape mechanic. gap is what bridges the tape's
// end back to its start (a bullet, so the tape reads continuously).
func scrollWindow(text string, offset, width int, gap string) string {
	if width <= 0 {
		return ""
	}
	if render.Width(text) <= width {
		return render.PadTo(text, width)
	}
	loop := []rune(text + gap)
	n := len(loop)
	off := ((offset % n) + n) % n
	// The window is width CELLS, not runes: a wide-rune place name would
	// otherwise run to twice the terminal (REVIEW R5-C-01). Bounded by n
	// runes per pass (P10-02).
	var b strings.Builder
	cells := 0
	for i := 0; i < n && cells < width; i++ {
		r := loop[(off+i)%n]
		w := render.RuneCells(r)
		if cells+w > width {
			break
		}
		b.WriteRune(r)
		cells += w
	}
	return render.PadTo(b.String(), width)
}

// advanceTicker steps the tape one cell; the offset is kept bounded to the
// current lane's loop length so it never overflows.
func (d *Dashboard) advanceTicker() {
	if len(d.ticker) == 0 || d.breaking != nil {
		return // a breaking takeover overrides the tape — freeze the scroll so normal rotation resumes where it left off (P4 F5)
	}
	d.tickerScroll++
	if n := d.tickerLoopLen(); n > 0 {
		d.tickerScroll %= n
	}
}

// tickerLoopLen is the current lane's tape length plus its wrap bullet.
func (d Dashboard) tickerLoopLen() int {
	cats := d.tickerCategories()
	if len(cats) == 0 {
		return 0
	}
	tape := strings.Join(d.tickerLane(d.opts(), cats[d.tickerCatIdx%len(cats)]), tickerBulletDot)
	return len([]rune(tape)) + len([]rune(tickerBulletDot))
}

// advanceTickerCategory rotates to the next non-empty lane (the 90 s switch,
// driven by the app). One lane present ⇒ no rotation. A new lane starts its
// tape from the left.
func (d *Dashboard) advanceTickerCategory() {
	if d.breaking != nil {
		return // hold the rotation under a takeover (P4 F5)
	}
	cats := d.tickerCategories()
	if len(cats) <= 1 {
		return
	}
	// EACH LANE RESUMES WHERE IT LEFT OFF, so every alert on a tape is reachable.
	// A visit lasts tickerRotate at one cell per tick, which is far less than a
	// busy lane's tape; a lane that restarted each visit would never show its
	// tail, while the band's count still claimed it was there.
	cur, showing := d.showingLane()
	if showing {
		d.parkScroll(cur)
	}
	d.tickerCatIdx = (d.tickerCatIdx + 1) % len(cats)
	d.tickerScroll = d.tickerScrolls[cats[d.tickerCatIdx]]
}

// parkScroll records how far a lane's tape has run, for its next visit.
func (d *Dashboard) parkScroll(c TickerCategory) {
	if d.tickerScrolls == nil {
		d.tickerScrolls = map[TickerCategory]int{}
	}
	d.tickerScrolls[c] = d.tickerScroll
}

// showingLane is the lane on the band right now, and whether there is one.
func (d Dashboard) showingLane() (TickerCategory, bool) {
	cats := d.tickerCategories()
	if len(cats) == 0 {
		return 0, false
	}
	return cats[d.tickerCatIdx%len(cats)], true
}

// laneAfter is the index in cats of want, or — when want is no longer present —
// of the first lane that follows it in rotation order, wrapping to the front.
//
// Ranked through tickerCatOrder rather than by comparing the constants, which
// happen to be declared in the same sequence today: the rotation order is that
// function's to state, and a reader who reorders it should not have to know
// that the handover silently depended on the iota values agreeing.
func laneAfter(cats []TickerCategory, want TickerCategory) int {
	order := tickerCatOrder()
	rank := func(c TickerCategory) int {
		for i, o := range order {
			if o == c {
				return i
			}
		}
		return len(order) // not in the rotation at all: sorts last
	}
	w := rank(want)
	for i, c := range cats {
		if rank(c) >= w {
			return i // the lane itself, or the first one past where it was
		}
	}
	return 0 // it was the last lane present: wrap
}

// setTicker replaces the active-alert set, KEEPING THE SHOWING LANE ON THE LANE
// IT WAS SHOWING.
//
// tickerCatIdx is a position in the PRESENT lanes, and the present set changes
// on every publish as alerts arrive and expire. This used to keep the index in
// range with `idx %= len(cats)`, which silently teleports it the moment the set
// shrinks: on [Disasters, Marine, Warnings, Watches] showing Watches (3), a
// publish where Disasters has gone quiet leaves three lanes and 3 % 3 = 0 —
// Marine. Disasters and Marine come from the national feed, so they are almost
// always present AND first in the order, and every shrink dragged the band back
// onto them. It looked like the rotation was stuck on those two, which is what
// the HUM LEAD saw.
//
// So the lane is carried as an IDENTITY across the swap: still present, still
// showing. Only a lane that has actually emptied hands over, and it hands over
// forward — the next present lane in rotation order, wrapping — so a lane going
// quiet advances the rotation instead of resetting it.
func (d *Dashboard) setTicker(items []TickerItem) {
	was, showing := d.showingLane()
	d.ticker = items
	cats := d.tickerCategories()
	switch {
	case len(cats) == 0:
		d.tickerCatIdx = 0
	case !showing:
		d.tickerCatIdx = 0 // nothing was on the band: start at the front
	default:
		d.tickerCatIdx = laneAfter(cats, was)
	}
	if n := d.tickerLoopLen(); n > 0 {
		d.tickerScroll %= n
	} else {
		d.tickerScroll = 0
	}
}
