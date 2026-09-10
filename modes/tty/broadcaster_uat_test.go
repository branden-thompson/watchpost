package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// broadcaster_uat_test.go — the HUM LEAD's UAT of 2026-09-10, one test per
// symptom. They are kept together because they share ONE root cause and the
// grouping is the record of that: `TruncateCells` counted an escape sequence's
// bytes as display cells, the console clamps every row of its frame through it,
// and the masthead's wordmark carries a truecolor escape per rune.

// THE FRAME IS NEVER CUT THROUGH AN ESCAPE, AND NEVER LOSES CONTENT THAT FITS.
//
//	"1A. 'WATCHPOS     ' — the rest of the title missing
//	 1B. Top of the box draw missing
//	 1C. Updated Missing
//	 1D. API only shows ✔9 and missing the rest
//	 1E. Chips … change color as the terminal window expands / shrinks
//	 1F. GAIN/VOL control missing — only showing the left arrow"
//
// Six symptoms, one measure. The masthead is the strictest case in the frame —
// it is the row with the most escapes per cell — so it is what this asserts.
func TestTheFrameSurvivesColour(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	for _, w := range []int{100, 120, 133, 150, 200} {
		b := NewBroadcaster()
		b.width, b.height, b.version = w, 50, "0.16.0"
		rows := strings.Split(b.View().Content, "\n")
		for i, r := range rows { // every row, not just the masthead's
			if got := render.Width(r); got != w {
				t.Fatalf("at %d cols, row %d measures %d cells", w, i, got)
			}
			if strings.Contains(render.StripSGRForTest(r), "\x1b") {
				t.Fatalf("at %d cols, row %d was cut through an escape: %q", w, i, r)
			}
		}
		head := render.StripSGRForTest(rows[bcInsetRows])
		for _, want := range []string{"WATCHPOST", "Broadcaster", "v0.16.0"} {
			if !strings.Contains(head, want) {
				t.Errorf("at %d cols the masthead lost %q: %q", w, want, head)
			}
		}
		if !strings.Contains(render.StripSGRForTest(rows[bcInsetRows+1]), "API:") {
			t.Errorf("at %d cols the masthead lost its API summary: %q", w, rows[bcInsetRows+1])
		}
		// THE GAIN CONTROL KEEPS BOTH ENDS. It showed only its left arrow.
		station := strings.Join(rows[bcInsetRows+4:bcInsetRows+9], "\n")
		if !strings.Contains(render.StripSGRForTest(station), "+") {
			t.Errorf("at %d cols the gain control lost its right end:\n%s", w, render.StripSGRForTest(station))
		}
	}
}

// EVERY KEY THE MASTHEAD NAMES IS A CHIP.
//
//	"Chips dont render their bkg … tells me something about coloring and tokens
//	 are broken in broadcaster ui"
//
// Nothing was broken. The console had TYPED its keys as text, so there was no
// chip to paint — while the station section beside it, which uses `KeyCap`,
// painted its chips correctly. That difference is what the report describes.
func TestTheMastheadsKeysAreChips(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	b := NewBroadcaster()
	b.width, b.height, b.version = 150, 50, "0.16.0"
	o := b.opts()
	row := b.controlRow(o)
	for _, key := range []string{"s", "a", "S", "ctrl+o", "?", "q"} {
		if !strings.Contains(row, o.KeyCap(key)) {
			t.Errorf("the masthead names %q without a chip: %q", key, row)
		}
	}
}

// THE REGIONS BREATHE.
//
//	"the sections of the line up are missing their blank row between the
//	 sections like the mocks"
//
// BETWEEN REGIONS, NEVER BETWEEN CARDS — the reference draws its cards flush
// inside a region, and a gap between every card would say each card is its own
// section, which is the opposite of what the rail is for.
func TestOneBlankRowSeparatesTheRegions(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	rows := strings.Split(b.View().Content, "\n")
	// BETWEEN THE CARDS: the frame ends where the running order does, so
	// everything past the last card border is the terminal, not the frame.
	first, last := -1, 0
	for i, r := range rows {
		// A CARD's border, not the masthead's — that box draws `+` corners too,
		// and anchoring on the glyph alone put `first` on row 2.
		if strings.Contains(r, "|    +---") {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		t.Fatal("no cards drawn")
	}
	breaks := 0
	for _, r := range rows[first:last] {
		cells := []rune(r)
		if len(cells) < b.width {
			continue
		}
		// A BREAK CARRIES NOTHING LEFT OF THE FRAME'S RIGHT-HAND COLUMNS — no
		// card, and no rail. "The blank row in between sections needs to be
		// completely blank … the breaks in the mock were intentional."
		if strings.TrimSpace(string(cells[:b.width-bcRightChrome+3])) != "" {
			continue
		}
		breaks++
		// AND THE BREAK GOES ALL THE WAY ACROSS. The inner wall at 144 is part
		// of the same vertical line the left rail is; leaving it drawn would
		// break the rail on one side of the cards and not the other. The scroll
		// rail's own caps are the exception, and they are what a cap IS.
		switch c := cells[b.width-6]; c {
		case ' ', '^', 'v':
		default:
			t.Errorf("a break still draws %q in the rail column:\n%s", string(c), r)
		}
	}
	// ONE PER READ REGION (D-71). SCHEDULED and LINE UP are one stack of cards
	// the rail names in two halves — "this is one area they should be
	// continuous" — so the air goes after LIVE and after UP NEXT, and nowhere
	// else inside the order.
	want := 0
	for _, r := range bcRegions {
		if r.reads {
			want++
		}
	}
	if breaks != want {
		t.Errorf("%d breaks in the running order, want %d (one after each read region)", breaks, want)
	}
	// AND THE CARDS INSIDE A REGION STAY FLUSH: the LINE UP's five slots draw
	// twenty rows with nothing between them.
	lane := newCardLane(b.cardBoxWidth(), b.opts().Glyphs())
	r := bcRegions[len(bcRegions)-1]
	body := b.slotRows(r, nil, lane)
	if got, want := len(body), (r.upto-r.from)*bcFlatCardRows; got != want {
		t.Errorf("the last region draws %d rows for %d slots, want %d", got, r.upto-r.from, want)
	}
}

// THE READ CARDS ARE TALL AND THE ORDERED ONES ARE FLAT (D-68).
//
//	"the LIVE CARD should be bigger to support showing at least most the script
//	 being played … UP NEXT should also be bigger"
func TestTheReadCardsAreTallerThanTheOrderedOnes(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	lane := newCardLane(b.cardBoxWidth(), b.opts().Glyphs())
	for _, r := range bcRegions {
		got := len(b.slotRows(r, nil, lane)) / (r.upto - r.from)
		want := bcFlatCardRows
		if r.reads {
			want = bcReadCardRows
		}
		if got != want {
			t.Errorf("%s draws %d rows per card, want %d", r.label, got, want)
		}
	}
}

// AND A READ SLOT ON A STATION AT REST IS EMPTY, NOT SHIMMERING.
//
//	"when it's on standby — like it is on first open/play — that should be blank
//	 we should have an empty state for that live slot"
//
// A shimmer promises a read that is coming. Nothing is coming while the station
// is not on the air, so the promise would be false — which is the same fault as
// the dead end D-64 removed, wearing the opposite costume.
func TestAReadSlotIsEmptyWhileTheStationIsAtRest(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	lane := newCardLane(b.cardBoxWidth(), b.opts().Glyphs())
	rest := strings.Join(b.slotRows(bcRegions[0], nil, lane), "\n")
	if strings.Contains(rest, "waiting for the line-up") {
		t.Errorf("a stopped station shimmers in the LIVE slot:\n%s", rest)
	}
	b.power = lineup.Running
	live := strings.Join(b.slotRows(bcRegions[0], nil, lane), "\n")
	if !strings.Contains(live, "waiting for the line-up") {
		t.Errorf("a running station with nothing decided IS waiting:\n%s", live)
	}
	// AND THE SLOT IS THE SAME SHAPE EITHER WAY, or the frame jumps under the
	// operator at the moment the station goes on the air.
	if a, c := len(strings.Split(rest, "\n")), len(strings.Split(live, "\n")); a != c {
		t.Errorf("the read slot changes height with the station: %d at rest, %d live", a, c)
	}
}

// THE FRAME OPENS AND CLOSES ON THE APP'S OWN AIR.
//
//	"Universal 2 line inset like Observer" … "global 2 row inset"
func TestTheFrameKeepsItsInset(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	rows := strings.Split(stripANSITest(b.View().Content), "\n")
	for i := range bcInsetRows {
		if strings.TrimSpace(rows[i]) != "" {
			t.Errorf("row %d is inside the opening inset and is not blank: %q", i, rows[i])
		}
		if j := len(rows) - 1 - i; strings.TrimSpace(rows[j]) != "" {
			t.Errorf("row %d is inside the closing inset and is not blank: %q", j, rows[j])
		}
	}
	if strings.TrimSpace(rows[bcInsetRows]) == "" {
		t.Error("the masthead follows the inset immediately; a third blank row is not the design")
	}
}

// THE STATION BAND IS PAINTED, NOT WALLED (D-70).
//
//	"We can remove the lines from the playing section — since we'll use color for
//	 the differentiation. It should be the same grey taken as the
//	 'Recent/Searched Locations' on STANDBY and ALERT RED on 'ON AIR'"
//
// BOTH TONES ARE TOKENS THAT ALREADY EXIST, named by the HUM LEAD after the
// thing they already paint: `GroupSectionBG` IS the RECENT/SEARCHED band, and
// `TickerEmergencyBG` is what MVS-D-62 calls "THE red". A second red mixed here
// would be a second answer to what red means in this app.
func TestTheStationBandIsPaintedByItsState(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	b := NewBroadcaster()
	b.width, b.height = 150, 50

	_, bg := b.stationTone()
	if want := render.Tok(render.GroupSectionBG); bg != want {
		t.Errorf("a station at rest wears the RECENT/SEARCHED grey: %q, want %q", bg, want)
	}
	b.power = lineup.Running
	if _, bg = b.stationTone(); bg != render.Tok(render.TickerEmergencyBG) {
		t.Errorf("a station ON AIR wears THE red: %q", bg)
	}
	// AND THE BAND CARRIES THE TONE ON EVERY ONE OF ITS ROWS, including the
	// breathing rows above and below — a band painted only where there are words
	// is a stripe, not a region.
	fg, bg := b.stationTone()
	for i, r := range strings.Split(b.stationSection(b.opts(), fg, bg), "\n") {
		if !strings.Contains(r, bg) {
			t.Errorf("row %d of the band is unpainted: %q", i, r)
		}
		if strings.Contains(stripANSITest(r), "│") || strings.Contains(stripANSITest(r), "|") {
			t.Errorf("row %d of the band still draws a wall; colour is its edge: %q", i, stripANSITest(r))
		}
	}
}

// THE SCROLL CONTROL STARTS AND ENDS WHERE THE REFERENCE PUTS IT (D-70).
//
//	"The scroll line is still not right. Even if the LIVE and UP NEXT cards share
//	 the same width, the vertical control should start and end where the mock
//	 says."
//
// ▲ ON THE BREAK ABOVE THE FIRST SCROLLING CARD and ▼ on a row of its own below
// the last one — never across a card's border, which is where the down cap
// landed until the running order learned to close on a blank row.
func TestTheScrollControlCapsSitOnRowsOfTheirOwn(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	rows := strings.Split(b.View().Content, "\n")
	up, down := -1, -1
	for i, r := range rows {
		switch []rune(r)[b.width-6] {
		case '^':
			up = i
		case 'v':
			down = i
		}
	}
	if up < 0 || down < 0 {
		t.Fatal("the scroll control draws both of its caps")
	}
	if up >= down {
		t.Fatalf("the up cap is above the down cap: %d, %d", up, down)
	}
	for _, at := range []int{up, down} {
		cells := []rune(rows[at])
		if strings.TrimSpace(string(cells[:b.width-bcRightChrome+3])) != "" {
			t.Errorf("row %d carries a cap AND content; a cap sits on a row of its own:\n%s", at, rows[at])
		}
	}
	// AND THE SCROLLING REGION IS THE ONE THAT SCROLLS. The two read cards are
	// always the same two, so the control must not begin above them.
	firstScrolling := 0
	for i, r := range rows {
		if strings.Contains(r, chipFor("2")) {
			firstScrolling = i
			break
		}
	}
	if firstScrolling == 0 {
		t.Fatal("the first scheduled card must reach the frame")
	}
	if up > firstScrolling {
		t.Errorf("the control starts at row %d, below the first scrolling card at %d", up, firstScrolling)
	}
}

// THE LANE'S CAPTION IS A BARE ROW (D-71).
//
//	"This line: │ … STANDARD … │    │ should have no pipes / lines."
//
// It is a caption OVER the running order, not a row of it — so it is written
// before the frame's own right-hand columns are added rather than inside them.
func TestTheLaneCaptionCarriesNoLines(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	rows := strings.Split(stripANSITest(b.View().Content), "\n")
	at := -1
	for i, r := range rows {
		if strings.Contains(r, "STANDARD") && !strings.Contains(r, "•") && !strings.Contains(r, "*") {
			at = i
			break
		}
	}
	if at < 0 {
		t.Fatal("the lane names itself above its cards")
	}
	if strings.Trim(rows[at], " ") != "STANDARD" {
		t.Errorf("the caption row carries something other than its word:\n%q", rows[at])
	}
}

// AND THE STATION'S IDENTITY IS IN THE STATION'S SECTION (D-71).
//
//	"one more consolidation — we're going to take the station center out of the
//	 masthead"
func TestTheTransmitterIsNamedInTheStationSection(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	got := stripANSITest(b.stationSection(b.opts(), "", ""))
	for _, want := range []string{"TRANSMITTER:", bcPlaceholderLocation, "TOWER GPS", "SERVICE RADIUS"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q is missing from the station section:\n%s", want, got)
		}
	}
	// AND THE BED SAYS WHAT IT IS DOING IN A SENTENCE, at the right, in the
	// column the station's own transition hint occupies.
	if !strings.Contains(got, "BED IS") {
		t.Errorf("the bed's state reads as a sentence:\n%s", got)
	}
}
