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
func (b Broadcaster) lineupRows(cards []lineup.Card) []render.LineupRow {
	out := make([]render.LineupRow, 0, MainTrackSlots)
	for i := bcScheduledFrom; i < MainTrackSlots; i++ { // bounded by the track (P10-02)
		row := render.LineupRow{Slot: i, Num: fmt.Sprintf("%02d.", i)}
		c, decided := b.slotCard(cards, i)
		if decided {
			row = b.lineupRowOf(row, c)
		}
		out = append(out, row)
	}
	return out
}

// lineupRowOf fills one row from the card in that slot.
func (b Broadcaster) lineupRowOf(row render.LineupRow, c lineup.Card) render.LineupRow {
	row.ReportType = reportTypeOf(c)
	row.Location = plaintext.Text(c.Headline)
	row.Priority = priorityOf(c)
	row.RequestedBy = requestedByOf(c)
	row.Correspondent = detailReadBy(c)
	row.Marks = render.Marks{
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
func (b Broadcaster) scheduledLines(cards []lineup.Card, used int) []string {
	w := b.bandWidth()
	if w <= 0 {
		return nil
	}
	lines := []string{"", centerText(bcScheduledHeading, w), ""}
	lines = append(lines, strings.Split(b.opts().LineupTable(b.lineupRows(cards), w), "\n")...)

	off, room := 0, b.height-used-2*bcInsetRows
	if room < 0 {
		room = 0
	}
	if room < len(lines) {
		// THE OFFSET IS CLAMPED HERE, NOT WHERE IT IS SET — only the frame knows
		// how much room the table has, and it changes with the terminal.
		off = min(b.queueOff, len(lines)-room)
		lines = append([]string(nil), lines[off:off+room]...)
	}
	return b.chromeAt(lines, off, len(lines)+off)
}

// bcScheduledHeading is the caption the reference draws over the running order.
const bcScheduledHeading = "S C H E D U L E D    L I N E - U P"
