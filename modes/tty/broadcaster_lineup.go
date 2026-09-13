package tty

// broadcaster_lineup.go — the running order, as rows for the table (D-94).
//
// THE CONSOLE JOINS; IT DOES NOT DERIVE. Every cell is a fact the schedule
// already holds, or a lookup against the pool the app published (D-93). `Card` is
// domain-free by DR-1 — it carries a subject and a headline, never a zip or a
// distance — so ZIP and DIST come from the pool entry the card's subject names.

import (
	"fmt"
	"strings"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/plaintext"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// lineupRows is every scheduled slot below the two the operator reads from.
//
// AN EMPTY SLOT STILL DRAWS ITS NUMBER, which is the reference's own idiom — the
// takeover box lists `07.` through `10.` with nothing beside them. A slot is an
// ADDRESS the operator can put something in, so it is shown whether or not the
// Director has filled it yet.
func (b Broadcaster) lineupRows(cards []lineup.Card, idx locIndex) []render.LineupRow {
	out := make([]render.LineupRow, 0, MainTrackSlots)
	for i := bcScheduledFrom; i < MainTrackSlots; i++ { // bounded by the track (P10-02)
		row := render.LineupRow{Slot: i, Num: fmt.Sprintf("%02d.", i)}
		row.Marks.Selected = i-bcScheduledFrom == b.lineupSelection()
		c, decided := b.slotCard(cards, i)
		if decided {
			row = b.lineupRowOf(row, c, idx)
		}
		out = append(out, row)
	}
	return out
}

// lineupRowOf fills one row from the card in that slot.
func (b Broadcaster) lineupRowOf(row render.LineupRow, c lineup.Card, idx locIndex) render.LineupRow {
	row.ReportType = reportTypeOf(c)
	row.Location = plaintext.Text(c.Headline)
	row.Priority = priorityOf(c)
	row.RequestedBy = requestedByOf(c)
	// THE POINTER SURVIVES THE FILL. This assigned a fresh `Marks` and threw the
	// caller's `Selected` away with it, so the pointer vanished on every row that
	// actually had a card in it — visible only once a card existed, which is why
	// the empty frame looked right.
	row.Marks = render.Marks{
		Selected:  row.Marks.Selected,
		Playing:   c.State == lineup.OnAir,
		HasAlert:  c.Slot == lineup.BreakingAlert,
		WarnAlert: c.Slot == lineup.BreakingAlert,
		// THE BURST'S OWN COUNT, which is what the operator is deciding about:
		// how many hazards this one beat will read.
		AlertCount: len(c.From),
	}
	if ref, ok := b.poolEntry(c.Subject); ok {
		row.Zip = ref.Zip
		row.DistMi = b.milesFromTower(ref)
		// AND THE WEATHER WHERE THE BEAT IS ABOUT (D-116). The join is FREE:
		// `producer()` may only offer locations from the station's pool, so every
		// card on this track is about a place the pool already holds — and the
		// pool's weather is already fetched (D-112). No new call, no new cadence.
		//
		// HUM LEAD, 2026-09-13, choosing this over wiring the correspondents:
		// "Useful information and doesn't require the correspondents wiring work."
		if loc := idx.at(ref); loc != nil {
			// NO EXTENDED DAYS: this table draws CONDITIONS and NOW, and the day
			// cells `weatherRow` would build are five allocations a row that
			// nothing on this console reads (D-120).
			w := weatherRow(loc, b.fireBold(), 0)
			row.Conditions, row.Now, row.Trend, row.Loading = w.Conditions, w.Now, w.Trend, w.Loading
			// AND THE PREFIX MARKS ARE THE PLACE'S (HUM LEAD: "Relevant Prefix
			// Alerts"). A report about a town with a warning, a fire and a quake
			// near it should say so where the operator is choosing what to read —
			// which is the same three marks the pool below draws for that town.
			//
			// A TAKEOVER KEEPS ITS OWN COUNT, because a burst's alerts are what
			// that card WILL READ rather than what is happening near a place.
			// Nothing puts a burst on this track today; the branch above says what
			// the answer is if anything ever does.
			if c.Slot != lineup.BreakingAlert {
				row.Marks.HasAlert, row.Marks.WarnAlert = w.HasAlert, w.WarnAlert
				row.Marks.AlertCount = w.AlertCount
			}
			row.Marks.Fire, row.Marks.FireHot, row.Marks.Seismic = w.Fire, w.FireHot, w.Seismic
		}
	}
	return row
}

// poolEntry is the published pool entry a card is about, by KEY.
//
// `Card.Subject` IS THE LOCATION KEY — `topoff.go` sets it from the proposal's
// `Ref`, which is `string(snapshot.Key(ref))`. So this is an identity match, not
// a name match: two Vistas with different centroids are different pool entries
// and the card names exactly one of them.
func (b Broadcaster) poolEntry(subject string) (snapshot.LocationRef, bool) {
	if subject == "" {
		return snapshot.LocationRef{}, false
	}
	for _, r := range b.area.Pool { // bounded by locations.PoolCap (P10-02)
		if string(snapshot.Key(r)) == subject {
			return r, true
		}
	}
	return snapshot.LocationRef{}, false
}

// reportTypeOf is what KIND of read this is, in the operator's case.
//
// FROM THE SLOT REGISTRY, NEVER INVENTED HERE. The registry shouts its names
// (`LOCATION REPORT`) because they are drawn as titles; a table column is read in
// running text, so it is title-cased — the NAME still has one owner, and D-31's
// finer taxonomy needs no change here when it lands.
func reportTypeOf(c lineup.Card) string {
	name := c.Slot.String()
	if name == "" {
		return ""
	}
	// EVERY WORD, not just the first: "LOCATION REPORT" is "Location Report", and
	// capitalising only the leading letter gave "Location report".
	words := strings.Fields(strings.ToLower(name))
	for i, w := range words { // bounded by the registry's own name (P10-02)
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// priorityOf is the badge the card wears.
//
// A PLACEHOLDER, AND SAYING SO IS THE POINT. Nothing in the schedule carries a
// priority today — the console draws the literal `STANDARD` on every card, and
// the reference's `Priority` rows are a model that does not exist yet (D-16's
// tiering, D-31's card types). A takeover is the one card that is genuinely not
// standard, so it is the one that reads differently until that model lands.
func priorityOf(c lineup.Card) string {
	if c.Slot == lineup.BreakingAlert {
		return "Priority"
	}
	return "Standard"
}

// requestedByOf is who asked for this beat, in the operator's words.
//
// THE ROLE'S NAME, NOT THE ENUM'S. `FromObserver` is the CARD PRODUCER — the
// role model's own first role, "decides that a card should exist" — and the
// operator has never heard of an origin called Observer while looking at a
// station. `FromOperator` is the person at the keyboard.
func requestedByOf(c lineup.Card) string {
	switch c.Origin {
	case lineup.FromOperator:
		return "Station Operator"
	case lineup.FromDirector:
		return "Director"
	case lineup.FromObserver:
		return "Producer"
	}
	return ""
}

// milesFromTower is how far a pool entry sits from the transmitter.
//
// NIL WHEN THERE IS NO TOWER, so the column is blank rather than reading a
// distance measured from the equator.
func (b Broadcaster) milesFromTower(ref snapshot.LocationRef) *float64 {
	tx := b.area.Transmitter
	if tx.Lat == 0 && tx.Lon == 0 {
		return nil
	}
	mi := geo.HaversineKM(tx.Lat, tx.Lon, ref.Lat, ref.Lon) * 0.621371
	return &mi
}

// bcScheduledFrom is the first slot the TABLE draws: the two above it are the
// cards the operator reads from (D-68), and they keep their own shape.
const bcScheduledFrom = 2

// scheduledLines is the SCHEDULED LINE-UP heading and its table, windowed to
// whatever height the frame has left (D-94).
//
// THE HEADING REPLACES THE RAIL. What the vertical rail spelled down the side in
// single letters, a centred caption says in one row — and the row it saves is a
// row of the running order.
//
// IT IS WINDOWED, NOT CLIPPED, for the reason D-87 gave when the cards outgrew
// the terminal: emitting every row and letting `clamp` cut the bottom loses the
// frame's own closing inset and runs rows past the edge, which FR-7.3 calls a
// defect rather than a degradation.
func (b Broadcaster) scheduledSpan(cards []lineup.Card, used int, idx locIndex) scrollSpan {
	w := b.tableWidth()
	if w <= 0 {
		return scrollSpan{}
	}
	slots := b.lineupRows(cards, idx)
	// THE CONTROLS SIT ABOVE THE HEADING, WHERE OBSERVER PUTS THEM (D-102, HUM
	// LEAD 2026-09-12): "control hints missing at the top of the table which
	// should be right above 'SCHEDULED LINE UP' (just like in Observer)".
	//
	// SAME SHAPE, SAME SPACING, SAME `[↑↓] Navigate` PUSHED RIGHT — `PadBetween`
	// is Observer's own, so the two rows cannot drift apart.
	lines := []string{""}
	// SPLIT, NOT EMBEDDED. `consoleControls` wraps on a narrow terminal, and a
	// multi-line string held as ONE element counted as one row in the height
	// budget and escaped the per-row padding — so the frame ran a row past the
	// terminal at 100x44 and a row measured 91 cells instead of 100. Two symptoms,
	// one cause.
	lines = append(lines, strings.Split(b.consoleControls(w), "\n")...)
	// THE HEADING IS A BAND, IN OBSERVER'S OWN TONE (D-102, HUM LEAD 2026-09-12):
	// "let's make the 'SCHEDULE LINE UP' 3 rows have the same background color as
	// Observer's 'RECENT / SEARCHED LOCATIONS' row (so it can be themed)".
	//
	// `GroupSectionBG` IS THAT TONE — `radio_panel.go` names it in as many words,
	// "GroupSectionBG, the RECENT / SEARCHED tone" — so the two bands are one
	// token and a theme moves both together.
	//
	// THREE ROWS: the air above, the words, the air below. A band is a region, and
	// its breathing room is painted or it is a stripe.
	band := render.Tok(render.GroupText) + ";" + render.Tok(render.GroupSectionBG)
	for _, r := range []string{"", centerText(bcScheduledHeading, w), ""} {
		lines = append(lines, render.TintKeeping(render.PadTo(r, w), band))
	}
	lines = append(lines, strings.Split(b.opts().LineupTable(slots, w), "\n")...)

	// THE TOTAL IS THE UNTRUNCATED COUNT, taken BEFORE the window is cut. Passing
	// the window's own length told `railed` there was nothing below it, and the
	// rail drew no caps at any height — the scroll worked and said it did not.
	total := len(lines)
	// THE POOL GETS ITS SHARE OF THE HEIGHT (D-104). The running order used to
	// take everything left and the pool drew in whatever remained, which is why
	// the operator could see twelve of twenty-five locations and no rail said so.
	off, room := 0, b.height-used-bcInsetRows-b.poolRoom()
	if room < 0 {
		room = 0
	}
	if room < len(lines) {
		// THE WINDOW FOLLOWS THE POINTER, AND IT IS COMPUTED HERE (D-101) because
		// only the frame knows how much room the table has — the same argument the
		// offset was already clamped here for. A `queueOff` moved at the keystroke
		// could not know the room, so it scrolled one way and never came back.
		dataAt := len(lines) - len(slots)
		if sel := b.lineupSelection(); sel >= 0 {
			at := dataAt + sel
			if at >= room {
				off = at - room + 1
			}
		} else {
			// THE POINTER IS IN THE POOL, so the running order shows its END: the
			// operator has walked past it and the row they came from is the last.
			off = len(lines) - room
		}
		off = max(0, min(off, len(lines)-room))
		lines = append([]string(nil), lines[off:off+room]...)
	}
	// THE RUNNING ORDER CARRIES NO CONTROL OF ITS OWN (D-106). Fifteen slots is
	// the whole list and it fits; the pool is what scrolls, so the pool is what
	// the control belongs to. If the frame is ever too short to hold the running
	// order this still windows it — silently, which is a fork worth raising
	// rather than a rail worth drawing beside a list that normally does not move.
	return scrollSpan{lines: lines, from: -1, off: off, total: total}
}

// scrollSpan is a region that scrolls, and where its window sits in the whole of
// it: the two facts a scroll control needs and the two a region knows about
// itself.
//
// IT EXISTS SO THE TWO TABLES CAN SHARE ONE CONTROL (D-104). The pointer already
// walks the running order and the pool as one list (`rowCount`), so the reference
// draws ONE rail beside both — and a rail over two regions has to be told where
// each of their windows sits rather than guessing from the rows it was handed.
type scrollSpan struct {
	lines []string
	// from is the row within `lines` the scroll control starts on, or -1 for a
	// region that does not scroll (D-106).
	from       int
	off, total int
}

// bcScheduledHeading is the caption the reference draws over the running order.
const bcScheduledHeading = "S C H E D U L E D    L I N E - U P"

// consoleControls is the running order's own control hints.
//
// WHAT THE OPERATOR CAN DO TO THE LIST THEY ARE POINTING AT, which is why it sits
// over the table and not in the masthead: the masthead's row is what the SURFACE
// offers, and this is what the ROW under the pointer offers.
func (b Broadcaster) consoleControls(w int) string {
	o := b.opts()
	segs := []string{
		o.KeyCap("l") + " Lookup Location from Pool",
		o.KeyCap("enter") + " Details / Manage Slot",
		o.KeyCap("r") + " Request for Line-Up",
	}
	nav := o.KeyCap("↑↓") + " Navigate"
	line := strings.Join(segs, "   ")
	if render.Width(line)+render.Width(nav)+2 <= w {
		return render.PadTo(render.PadBetween(line, nav, w), w)
	}
	// PADDED TO THE FRAME, like every other row. The wrap path returned ragged
	// lines, and the colour gate measures every row against the frame's width.
	wrapped := render.WrapSegments(append(segs, nav), w, "   ")
	for i, l := range wrapped { // bounded by the segments (P10-02)
		wrapped[i] = render.PadTo(l, w)
	}
	return strings.Join(wrapped, "\n")
}
