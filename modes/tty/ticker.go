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
	"strings"

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
type TickerCategory int

const (
	CatQuake    TickerCategory = iota // Severe Earthquakes (USGS)
	CatTropical                       // Tropical Cyclones (the live NHC storms)
	CatWarning                        // Warnings — all NWS warning products
	CatWatch                          // Watches — all NWS watch products
)

// tickerCatOrder is the rotation order (a function, not a global — P10-06).
func tickerCatOrder() []TickerCategory {
	return []TickerCategory{CatQuake, CatTropical, CatWarning, CatWatch}
}

// warnGlyph is the leading mark on the band (⚠, or ! under --ascii); the lane's
// count sits before it ([count] [glyph], HUM LEAD).
func warnGlyph(o render.Opts) string {
	if o.ASCII {
		return "!"
	}
	return "⚠"
}

// TickerItem is one active alert as the marquee shows it. The app composes Text
// ("Tornado Warning · the Oklahoma City area  declared 3:42 PM · expires
// 4:15 PM") from a globalfeed.Event; Category picks the lane and the colour.
type TickerItem struct {
	ID       string // the source event id
	Category TickerCategory
	Text     string
	Severity TickerSeverity // ordering within the lane (set by the app)
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
		content := render.TintRaw(centerText(it.Text, width), tones)
		blank := render.TintRaw(strings.Repeat(" ", width), tones)
		return blank + "\n" + content + "\n" + blank
	}

	right := tickerRightReserve
	mid := "  no active severe events"
	bg, fg := render.GroupSectionBG, render.TickerMutedFG // the muted band matches the RECENT/SEARCHED group header (HUM LEAD)
	if cats := d.tickerCategories(); len(cats) > 0 {
		cur := cats[d.tickerCatIdx%len(cats)]
		items := d.tickerLane(cur)
		left := fmt.Sprintf("  %d %s  ", len(items), warnGlyph(o)) // [count] [glyph] (HUM LEAD: the original left indicator); the lane is read by its band colour
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

// tickerLane is the Text of every alert in a lane, in the app's order.
func (d Dashboard) tickerLane(c TickerCategory) []string {
	var out []string
	for _, it := range d.ticker {
		if it.Category == c {
			out = append(out, it.Text)
		}
	}
	return out
}

// tickerCatBG is the band background for a lane — FIXED per category (HUM LEAD
// colour pass, 2026-08-27): Earthquakes = Red, Warnings = Orange, Watches =
// Yellow, Tropical = Blue.
func tickerCatBG(c TickerCategory) render.Token {
	switch c {
	case CatQuake:
		return render.TickerRedBG
	case CatWarning:
		return render.TickerOrangeBG
	case CatWatch:
		return render.TickerYellowBG
	default: // CatTropical
		return render.TickerBlueBG
	}
}

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
	var b strings.Builder
	for i := 0; i < width; i++ {
		b.WriteRune(loop[(off+i)%n])
	}
	return b.String()
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
	tape := strings.Join(d.tickerLane(cats[d.tickerCatIdx%len(cats)]), tickerBulletDot)
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
	d.tickerCatIdx = (d.tickerCatIdx + 1) % len(cats)
	d.tickerScroll = 0
}

// setTicker replaces the active-alert set. The lane index is kept valid as the
// present set changes; a lane whose alerts all expired simply drops out of the
// rotation on the next publish.
func (d *Dashboard) setTicker(items []TickerItem) {
	d.ticker = items
	if cats := d.tickerCategories(); len(cats) > 0 {
		d.tickerCatIdx %= len(cats)
	} else {
		d.tickerCatIdx = 0
	}
	if n := d.tickerLoopLen(); n > 0 {
		d.tickerScroll %= n
	} else {
		d.tickerScroll = 0
	}
}
