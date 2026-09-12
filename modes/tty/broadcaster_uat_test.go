package tty

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
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
		if strings.Contains(r, "   +---") && strings.HasPrefix(r, "|") {
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
		switch []rune(r)[b.width-2] {
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

// THE LANE NAMES ITSELF OVER ITS OWN COLUMN (D-87).
//
// IT USED TO CARRY NO LINES AT ALL (HUM LEAD, UAT 2026-09-10: "this line …
// should have no pipes / lines"), because it was written outside the frame's
// columns. The v2 reference puts it INSIDE them — `│   │ … MAIN SCHEDULE
// (ROLLING WINDOW)` — so it wears the rail's walls like every other row of the
// running order, and what it must not carry is a CARD's border.
func TestTheLaneCaptionSitsOverTheRunningOrder(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	rows := strings.Split(stripANSITest(b.View().Content), "\n")
	at := -1
	for i, r := range rows {
		if strings.Contains(r, bcLaneLabel) {
			at = i
			break
		}
	}
	if at < 0 {
		t.Fatal("the lane does not name itself above its cards")
	}
	if strings.Contains(rows[at], "+--") {
		t.Errorf("the caption row carries a card's border:\n%q", rows[at])
	}
	// AND IT IS CENTRED OVER THE CARDS, which is what makes it read as naming
	// the column rather than the frame.
	main := bcRailWidth + bcRailGap + b.priorityWidth() + bcColumnGap
	lead := strings.Index(rows[at], bcLaneLabel) - main
	trail := b.cardBoxWidth() - lead - len(bcLaneLabel)
	if d := lead - trail; d > 1 || d < -1 {
		t.Errorf("the caption is not centred over the cards: %d left, %d right", lead, trail)
	}
}

// AND THE STATION'S IDENTITY IS IN THE STATION'S SECTION (D-71).
//
//	"one more consolidation — we're going to take the station center out of the
//	 masthead"
func TestTheTransmitterIsNamedInTheStationSection(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	// THE STATION IS TOLD WHERE IT IS (D-72). It was two hard-coded constants
	// until the station had settings of its own.
	b, _ = b.Update(StationAreaMsg{
		Transmitter: snapshot.LocationRef{Label: "Bonsall, CA", Lat: 33.2881, Lon: -117.2256},
		RadiusMi:    25,
	})
	got := stripANSITest(b.stationSection(b.opts(), "", ""))
	for _, want := range []string{"TRANSMITTER:", "Bonsall, CA", "TOWER GPS", "33.2881", "-117.2256", "SERVICE RADIUS: 25 Miles"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q is missing from the station section:\n%s", want, got)
		}
	}
	// AND A STATION THAT HAS NOT BEEN TOLD SAYS SO, rather than naming a place
	// it was compiled with.
	bare := stripANSITest(NewBroadcaster().withSize(150, 74).stationSection(b.opts(), "", ""))
	if !strings.Contains(bare, bcNoTransmitter) {
		t.Errorf("a station with no epicentre says so:\n%s", bare)
	}
	// AND THE BED SAYS WHAT IT IS DOING IN A SENTENCE, at the right, in the
	// column the station's own transition hint occupies.
	if !strings.Contains(got, "BED IS") {
		t.Errorf("the bed's state reads as a sentence:\n%s", got)
	}
}

// THE BED SAYS WHAT IT IS ACTUALLY CARRYING (F-79, closed at D-78).
//
// IT SAID `(no relay tuned)` AND `○ INACTIVE` UNCONDITIONALLY, because the
// schedule published the line-up and the power and nothing about the bed — so
// the one region D-62 built to answer "what is on the air" answered two thirds
// of it, and the third was a constant that happened to be true at launch.
func TestTheBedDrawsWhatWasPublished(t *testing.T) {
	b := NewBroadcaster().withSize(150, 74)

	// BEFORE IT IS TOLD, it says the true thing it can say.
	bare := stripANSITest(b.stationSection(b.opts(), "", ""))
	if !strings.Contains(bare, bcNoRelay) || !strings.Contains(bare, "BED IS INACTIVE") {
		t.Errorf("an untold bed says so:\n%s", bare)
	}

	b, _ = b.Update(BedMsg{Relay: "Oceanside, CA", Carrying: true})
	got := stripANSITest(b.stationSection(b.opts(), "", ""))
	if !strings.Contains(got, "Oceanside, CA") {
		t.Errorf("the bed names what it is carrying:\n%s", got)
	}
	if !strings.Contains(got, "BED IS ACTIVE") {
		t.Errorf("and says that it is:\n%s", got)
	}
	if strings.Contains(got, bcNoRelay) {
		t.Errorf("the placeholder outlived the answer:\n%s", got)
	}

	// AND IT GOES BACK. A bed that only ever learned to say ACTIVE would be the
	// same constant one state along.
	b, _ = b.Update(BedMsg{})
	if back := stripANSITest(b.stationSection(b.opts(), "", "")); !strings.Contains(back, "BED IS INACTIVE") {
		t.Errorf("cutting the bed away says so:\n%s", back)
	}
}

// THE STATION BAND KEEPS THE SAME INSET ON BOTH SIDES (D-80).
//
//	"since it's a colored bkg, let's ensure we have a consistent inset for
//	 content: 1 line top/bottom inset (good); 3 col left/right (currently 4 col
//	 left; 0 col right)" — HUM LEAD, UAT 2026-09-11
//
// ASSERTED WITH COLOUR ON, because that is the only way the RIGHT-hand inset
// exists: `Block` trims trailing spaces when there is no colour to paint, so a
// test without it measures a band that has no edges to be inset from.
func TestTheStationBandIsEvenlyInset(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	b := NewBroadcaster().withSize(150, 74)
	b.power = lineup.Running
	fg, bg := b.stationTone()
	rows := strings.Split(b.stationSection(b.opts(), fg, bg), "\n")

	// ONE BLANK ROW TOP AND BOTTOM — the vertical half, which was already right.
	if n := len(rows); n != 6 {
		t.Fatalf("the band is a blank row, four of content and a blank row; got %d", n)
	}
	for _, at := range []int{0, len(rows) - 1} {
		if strings.TrimSpace(stripANSITest(rows[at])) != "" {
			t.Errorf("row %d is the band's air and carries text: %q", at, stripANSITest(rows[at]))
		}
	}
	for i, r := range rows {
		if got := render.Width(r); got != b.width {
			t.Errorf("row %d is %d cells; a painted band is a rectangle", i, got)
		}
		p := stripANSITest(r)
		if strings.TrimSpace(p) == "" {
			continue // an air row is inset from nothing
		}
		// THE INSET IS THE FIRST THREE CELLS, not where the text happens to
		// begin: the status row's label column is reserved and empty (D-62), so
		// its words start nineteen cells in and its INSET is still three.
		if head := p[:len(bcSectionInset)]; strings.TrimSpace(head) != "" {
			t.Errorf("row %d has no left inset: %q", i, head)
		}
		if lead := len(p) - len(strings.TrimLeft(p, " ")); lead < len(bcSectionInset) {
			t.Errorf("row %d begins %d cells in, want at least %d", i, lead, len(bcSectionInset))
		}
		if trail := len(p) - len(strings.TrimRight(p, " ")); trail < len(bcSectionInset) {
			t.Errorf("row %d ends %d cells short of the edge, want at least %d", i, trail, len(bcSectionInset))
		}
	}
}

// A WINDOW ON TOP OWNS THE KEYBOARD, AND THE CONSOLE'S OWN CONTROLS STAND ASIDE
// (D-58, enforced at D-87).
//
// THE BED'S ARROWS HAD THIS WRONG SINCE D-78. The Router looks a key up BEFORE
// the check that hands an open window its input, so `←`/`→` stepped the relay
// while the operator was moving through the status window — and nothing saw it,
// because no gate drove those keys with a window up. Adding the queue's `↑`/`↓`
// is what surfaced it: the modal-reachability gate drives exactly those.
func TestTheConsolesControlsStandAsideForAnOpenWindow(t *testing.T) {
	d, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	r := NewRouter(d)
	r.active = SurfaceBroadcaster

	if !r.consoleOwnsTheKeys() {
		t.Fatal("with no window open the console owns its own keys")
	}

	// THE OBSERVER'S WINDOW IS THE ONE THAT COMPOSITES OVER THE CONSOLE, which
	// is why it is the one asked.
	open := r
	open.observer = openAModal(t, r.observer)
	if !open.observer.ModalOpen() {
		t.Skip("no modal could be opened in this fixture")
	}
	if open.consoleOwnsTheKeys() {
		t.Error("the console acted on a key meant for the window over it")
	}

	// AND ON OBSERVER THE CONSOLE NEVER OWNS THEM, window or not: those arrows
	// walk the listener's table.
	obs := r
	obs.active = SurfaceObserver
	if obs.consoleOwnsTheKeys() {
		t.Error("the console claimed the arrows while Observer was the surface")
	}
}

// openAModal puts the status window up, which is the one the reachability gate
// drives.
func openAModal(t *testing.T, d Dashboard) Dashboard {
	t.Helper()
	d.width, d.height = 150, 74
	m, _ := d.Update(keyPress(t, "S"))
	if out, ok := m.(Dashboard); ok {
		return out
	}
	return d
}
