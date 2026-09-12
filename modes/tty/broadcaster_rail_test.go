package tty

// broadcaster_rail_test.go — the left rail (D-60).
//
// The reference names each region of the running order down the left edge, one
// letter per row: LIVE, UP NEXT, BED, SCHEDULED, LINE UP. It is what says WHICH
// PART of the schedule a card is in — and it is why the card itself carries no
// state strip: the rail already says it, and two carriers of one fact is the
// shape this release keeps removing.

import (
	"fmt"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
)

func rail(label string, rows int) []string {
	return railColumn(label, rows, render.Opts{ASCII: true}.Glyphs())
}

func TestTheRailSpellsItsSectionDownTheEdge(t *testing.T) {
	got := rail("LIVE", 6)
	if len(got) != 6 {
		t.Fatalf("the rail is as tall as the section it names; got %d", len(got))
	}
	var letters []string
	for _, r := range got {
		if c := strings.TrimSpace(strings.Trim(r, "|")); c != "" {
			letters = append(letters, c)
		}
	}
	if strings.Join(letters, "") != "LIVE" {
		t.Errorf("the rail spells the section; got %q", strings.Join(letters, ""))
	}
}

// A SPACE IN THE LABEL IS A BLANK ROW, which is how the reference draws
// "UP NEXT": U, P, blank, N, E, X, T.
func TestASpaceInTheLabelIsABlankRow(t *testing.T) {
	got := rail("UP NEXT", 7)
	if len(got) != 7 {
		t.Fatalf("want 7 rows; got %d", len(got))
	}
	if c := strings.TrimSpace(strings.Trim(got[2], "|")); c != "" {
		t.Errorf("the third row is the label's space; got %q", c)
	}
}

// EVERY ROW IS THE RAIL'S OWN WIDTH, so the cards beside it all start in the
// same column. A ragged rail would step the whole running order in and out.
func TestEveryRailRowIsTheSameWidth(t *testing.T) {
	for _, n := range []int{1, 4, 9, 20} {
		for i, r := range rail("SCHEDULED", n) {
			if w := utf8.RuneCountInString(r); w != bcRailWidth {
				t.Errorf("rows %d, row %d: the rail is %d cells, want %d\n%q", n, i, w, bcRailWidth, r)
			}
		}
	}
}

// A LABEL LONGER THAN ITS SECTION IS CUT, NOT OVERFLOWED. A rail that grew rows
// would push the card it names off the bottom of the frame.
func TestALabelTallerThanItsSectionIsCut(t *testing.T) {
	got := rail("SCHEDULED", 3)
	if len(got) != 3 {
		t.Fatalf("the section's height wins; got %d rows", len(got))
	}
}

// THE RAIL IS WALLED ON BOTH SIDES, which is what makes it a rail rather than a
// margin: the frame's own edge at the left, and the divider the cards begin
// after at the right.
func TestTheRailIsWalledOnBothSides(t *testing.T) {
	for _, r := range rail("BED", 3) {
		if !strings.HasPrefix(r, "|") || !strings.HasSuffix(r, "|") {
			t.Errorf("the rail is walled both sides; got %q", r)
		}
	}
}

// THE TRACKS LAND ON THE REFERENCE'S COLUMNS (D-87).
//
// READ OFF THE MOCK, NOT COPIED OUT OF IT. The numbers live in
// `mock-broadcaster-v2.txt` and this test finds them there, so the reference and
// the console cannot drift apart without the drift being the failure — which is
// the standing rule for this project's mocks and is exactly what a hand-copied
// 132 stopped being the moment the layout changed.
func TestTheTracksLandOnTheReferencesColumns(t *testing.T) {
	left, right := mockTrackColumns(t)
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true

	start := bcRailWidth + bcRailGap
	if start != left[0] {
		t.Errorf("the alert column starts at %d; the reference puts it at %d", start, left[0])
	}
	if got := start + b.priorityWidth() - 1; got != left[1] {
		t.Errorf("the alert column ends at %d; the reference ends it at %d", got, left[1])
	}
	main := start + b.priorityWidth() + bcColumnGap
	if main != right[0] {
		t.Errorf("the running order starts at %d; the reference puts it at %d", main, right[0])
	}
	if got := main + b.cardBoxWidth() - 1; got != right[1] {
		t.Errorf("the running order ends at %d; the reference ends it at %d", got, right[1])
	}
}

// mockTrackColumns is where the v2 reference puts each track's borders, read
// from the row that carries both.
func mockTrackColumns(t *testing.T) (left, right [2]int) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "06_docs", "02_features",
		"0.16.0-broadcaster-ui", "01-objectives", "mock-broadcaster-v2.txt"))
	if err != nil {
		t.Fatalf("the reference mock is the source of these numbers: %v", err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.Contains(line, bcTakeoverTitle) {
			continue
		}
		r := []rune(line)
		var opens, closes []int
		for i, c := range r {
			switch c {
			case '┏':
				opens = append(opens, i)
			case '┓':
				closes = append(closes, i)
			}
		}
		if len(opens) == 2 && len(closes) == 2 {
			return [2]int{opens[0], closes[0]}, [2]int{opens[1], closes[1]}
		}
	}
	t.Fatal("the reference has no row carrying both tracks' borders")
	return
}

// A SHORT SECTION GETS A SHORTER WORD, NOT A CUT ONE. A rail reading "SCHEDULE"
// or "U P _ N" looks like rendering damage rather than like a short section —
// and the first version of this did exactly that.
func TestAShortSectionLaddersItsLabelRatherThanCuttingIt(t *testing.T) {
	for _, c := range []struct {
		label string
		rows  int
		want  string
	}{
		{"SCHEDULED", 9, "SCHEDULED"},
		{"SCHEDULED", 8, "SCHED"},
		{"SCHEDULED", 4, "SCH"}, // SCHED is five runes; four rows cannot hold it
		{"SCHEDULED", 3, "SCH"},
		{"UP NEXT", 7, "UP NEXT"},
		{"UP NEXT", 4, "NEXT"},
		{"UP NEXT", 2, "UP"},
		{"LINE UP", 4, "LINE"},
	} {
		var letters []string
		for _, r := range rail(c.label, c.rows) {
			if x := strings.TrimSpace(strings.Trim(r, "|")); x != "" {
				letters = append(letters, x)
			}
		}
		if got := strings.Join(letters, ""); got != strings.ReplaceAll(c.want, " ", "") {
			t.Errorf("%q in %d rows spells %q, want %q", c.label, c.rows, got, c.want)
		}
	}
}

// THE LABEL SITS IN THE MIDDLE OF ITS REGION, not at the top of it. A rail
// pinned to the first row reads as naming that CARD; centred, it reads as naming
// the whole region — which is what it is for, and why the card carries no state
// of its own.
func TestTheLabelIsCentredInItsSection(t *testing.T) {
	rows := rail("LIVE", 10) // four letters, six blanks
	first, last := -1, -1
	for i, r := range rows {
		if strings.TrimSpace(strings.Trim(r, "|")) != "" {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		t.Fatal("the label must be drawn")
	}
	above, below := first, len(rows)-1-last
	if above == 0 {
		t.Errorf("the label is pinned to the top rather than centred: %d above, %d below", above, below)
	}
	if d := above - below; d > 1 || d < -1 {
		t.Errorf("the label is not centred: %d rows above, %d below", above, below)
	}
}

// NO DOUBLE BLANKS BETWEEN REGIONS. A section carries its own breathing row;
// a second one appended by the frame made a double gap that reads as a
// rendering fault rather than as spacing (HUM LEAD, UAT 2026-09-10).
func TestTheFrameHasNoDoubleBlankRows(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	rows := strings.Split(stripANSITest(b.View().Content), "\n")
	// The frame is padded to the terminal, so trailing blanks below the content
	// are expected; only gaps BETWEEN drawn rows are the concern.
	last := 0
	for i, r := range rows {
		if strings.TrimSpace(r) != "" {
			last = i
		}
	}
	// FROM BELOW THE STATION BAND. The opening inset is two blank rows by design
	// (D-68), and the band's own closing breathing row sits directly above the
	// bare separator under it — both intended, and both render as whitespace
	// only because a test draws without colour: the band is a PAINTED region
	// (D-70), so that row is filled, not empty.
	for i := bcInsetRows + 10; i <= last; i++ {
		if strings.TrimSpace(rows[i]) == "" && strings.TrimSpace(rows[i-1]) == "" {
			t.Errorf("rows %d and %d are both blank — a section's spacing has two owners", i-1, i)
		}
	}
}

// EVERY ROW OF THE RUNNING ORDER OPENS THE FRAME (D-87).
//
// IT USED TO CLOSE IT TOO, and the v2 reference took that edge away: "Notice the
// line on the far right is gone." What is left on the right is three of air and
// the scroll, so what a row of the running order still owes is its LEFT wall —
// the rail's own — and its full width.
func TestEveryRowOfTheRunningOrderOpensTheFrame(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b.power = lineup.Running
	rows := strings.Split(stripANSITest(b.View().Content), "\n")

	// FROM THE FIRST CARD DOWN. Above it are the masthead's own box, the painted
	// station band (no walls by design, D-70) and the lane's bare caption (none
	// either, D-71) — three regions that close themselves or deliberately do not.
	first, last := -1, 0
	for i, r := range rows {
		if strings.TrimSpace(r) != "" {
			last = i
		}
		if first < 0 && strings.Contains(r, bcLaneLabel) {
			first = i + 1 // the running order begins under its caption
		}
	}
	if first < 0 {
		t.Fatal("the running order drew no caption, so there is nothing to check")
	}
	belowTheCards := false
	for i := first; i <= last; i++ {
		r := []rune(rows[i])
		if len(r) != b.width {
			t.Errorf("row %d is %d cells, want %d", i, len(r), b.width)
			continue
		}
		// A ROW WITH NOTHING IN THE CARD COLUMN OWES NOTHING. A region break is
		// completely blank, walls included (HUM LEAD, UAT 2026-09-10), and the
		// scroll rail's caps deliberately sit ON those breaks — so the question
		// is only ever asked of a row the running order actually drew.
		// THE WALL BELONGS TO THE CARD REGION, AND THE TABLE HAS NONE (D-94).
		// The reference draws the SCHEDULED LINE-UP wall-less, full width, under
		// a heading of its own — so the question stops being asked at the
		// heading, which is where the cards stop.
		if belowTheCards {
			break
		}
		if strings.Contains(rows[i], bcScheduledHeading) {
			belowTheCards = true
			continue
		}
		main := bcRailWidth + bcRailGap + b.priorityWidth() + bcColumnGap
		// THE CARD COLUMN ONLY, not everything right of it: the scroll rail's
		// caps sit ON the region breaks by design (D-70), so a row carrying just
		// a cap is still a break and still owes no wall.
		if len(r) < main+b.cardBoxWidth() || strings.TrimSpace(string(r[main:main+b.cardBoxWidth()])) == "" {
			continue
		}
		if r[0] != '|' {
			t.Errorf("row %d does not open the frame: starts %q\n%.40s", i, string(r[0]), rows[i])
		}
	}
}

// THE SCROLL GUTTER IS BLANK EXCEPT FOR THE THUMB. `RailGlyphsFor` draws a bar
// on every row, which is right for a window that has no edge of its own — here
// the inner wall is already beside it, and a bar there reads as a doubled
// border rather than as a rail.
func TestTheScrollGutterCarriesOnlyTheThumb(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	row := strings.Repeat("-", b.cardBoxWidth())
	body := b.zipTracks(rail("SCHEDULED LINE UP", 3), nil, []string{row, row, row})
	framed := b.framed(body, 2, 10)

	// THE RAIL IS COLUMN 148 (D-87), and it is the LAST thing on the row: the
	// frame's outer wall on this side is gone, because the cards are boxes with
	// their own borders and a wall around them was a second edge saying the same
	// thing. Every row carries the bar; exactly one carries the thumb.
	// ▲ OPENS IT AND ▼ CLOSES IT (D-70), which is `Railify`'s own contract —
	// "callers draw ▲/▼ themselves" — and the HUM LEAD's UAT: "the vertical
	// control should start and end where the mock says."
	if got := []rune(framed[0])[b.width-2]; got != '^' {
		t.Errorf("the rail opens on %q, want the up cap", string(got))
	}
	if got := []rune(framed[len(framed)-1])[b.width-2]; got != 'v' {
		t.Errorf("the rail closes on %q, want the down cap", string(got))
	}
	thumbs := 0
	for i, r := range framed[1 : len(framed)-1] {
		switch c := []rune(r)[b.width-2]; c {
		case '#':
			thumbs++
		case '|':
		default:
			t.Errorf("row %d draws %q in the rail column; a rail is a bar or the thumb:\n%.40s", i+1, string(c), r)
		}
	}
	if thumbs != 1 {
		t.Errorf("the rail carries exactly one thumb between its caps; it carries %d", thumbs)
	}
}

// THE MARK COLUMN BELONGS TO THE SCROLL RAIL, AND ONLY TO IT (D-85, HUM LEAD
// 2026-09-11): "We need to remove the extra lines on the right side of the UI
// next to LIVE and UP NEXT."
//
// It was the wall on EVERY row, scrolling or not — a second vertical beside two
// regions with nothing to scroll, which reads as a column that stopped rather
// than as one that was never there.
func TestTheReadRegionsDrawNoScrollRailBesideThem(t *testing.T) {
	b := Broadcaster{width: 150}
	g := b.opts().Glyphs()
	body := []string{"a card row", "another"}

	quiet := b.chrome(body, false, 0, 0)
	for i, r := range quiet {
		// The frame's own right wall is the LAST cell; anything before it in the
		// mark column is the rail that should not be there.
		trimmed := strings.TrimSuffix(r, g.Rail)
		if strings.Contains(strings.TrimRight(trimmed, " "), g.Rail) {
			t.Errorf("row %d carries a rail beside a region that does not scroll: %q", i, r)
		}
	}

	// AND THE SCROLLING REGION STILL HAS ONE, which is what makes the absence
	// above mean something.
	scrolled := b.chrome([]string{"a", "b", "c"}, true, 2, 10)
	if !strings.Contains(scrolled[0], render.RailGlyphsFor(false).Up) {
		t.Errorf("the scrolling region lost its rail: %q", scrolled[0])
	}
}

// THE LEFT RAIL NAMES ITS REGION IN COLOUR (D-86, HUM LEAD 2026-09-11): "Color
// BKGs for the left RAIL: LIVE = RED, UP NEXT = ORANGE, SCHEDULED LINEUP =
// BLUE."
//
// ASKED OF THE TOKENS, NOT OF A LITERAL. A test carrying the SGR values would
// pass while a theme changed underneath it, which is the opposite of what
// "themeable like Observer" means.
func TestEachRegionsRailCarriesItsOwnGround(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(func() { rendering.SetColorEnabledForTest(false) })
	g := render.Opts{}.Glyphs()

	for _, tc := range []struct {
		label string
		tok   render.Token
	}{
		{"LIVE ON AIR", render.RailLiveBG},
		{"UP NEXT", render.RailNextBG},
		// ONE GROUND FOR THE WHOLE QUEUE, which D-87 made one region as well —
		// SCHEDULED and LINE UP were one stack of cards the rail named twice.
		{"SCHEDULED LINE UP", render.RailQueueBG},
	} {
		t.Run(tc.label, func(t *testing.T) {
			rows := railColumn(tc.label, 6, g)
			want := render.Tok(tc.tok)
			if want == "" {
				t.Fatalf("%s has no ground token value", tc.label)
			}
			for i, r := range rows {
				if !strings.Contains(r, want) {
					t.Fatalf("row %d of %s is %q; it does not carry %s", i, tc.label, r, tc.tok)
				}
			}
		})
	}

	// AND THE REGIONS ARE TOLD APART. Three grounds that resolved to one colour
	// would draw a rail that says nothing, and would still pass a per-region
	// check.
	live, next, queue := render.Tok(render.RailLiveBG), render.Tok(render.RailNextBG), render.Tok(render.RailQueueBG)
	if live == next || next == queue || live == queue {
		t.Errorf("the rail's three grounds are not distinct: %q / %q / %q", live, next, queue)
	}

	// THE PRIORITY RAIL IS NOT A REGION and carries no region's ground: it is
	// the overlay's own, and it appears only while the rail has something.
	for _, r := range railColumn("PRIORITY", 4, g) {
		for _, tok := range []render.Token{render.RailLiveBG, render.RailNextBG, render.RailQueueBG} {
			if strings.Contains(r, render.Tok(tok)) {
				t.Errorf("the priority rail wears %s, which belongs to a region of the running order", tok)
			}
		}
	}
}

// THE QUEUE SCROLLS, AND THE THUMB TRACKS IT (D-87, HUM LEAD 2026-09-11):
// "that's why we have the vertical scroll bar so that works like Observer —
// that section just needs to be able to scroll up and down."
//
// THE CARDS OUTGREW THE TERMINAL. A card is a manifest now, so ten of them need
// about ninety rows against the reference's seventy-four — and a queue the
// operator cannot reach the bottom of is a line-up they cannot manage.
func TestTheQueueScrollsAndItsThumbFollows(t *testing.T) {
	base := NewBroadcaster()
	// A SHORT TERMINAL, BECAUSE THE TABLE FITS THE REFERENCE'S (D-94).
	//
	// This test was written when a card was a manifest and ten of them needed
	// ninety rows against the reference's seventy-four. Thirteen TABLE rows fit
	// in seventy-four with room to spare — which is the improvement — so the
	// scroll is now exercised where it actually matters: a terminal too short to
	// hold the running order.  MEASURED — at 150 wide, 64 rows fits all thirteen
	// and 60 fits nine, so sixty is where the window is genuinely a window.
	base.width, base.height, base.ascii = 150, 60, true
	base.power = lineup.Running

	// THE ROWS THAT SCROLL ARE THE TABLE'S NOW (D-94), not the cards'. The two
	// read cards never scrolled and still do not — they are the reason the rail
	// begins below them (D-68) — so what this walks is the numbered rows of the
	// running order.
	slots := func(b Broadcaster) []string {
		var out []string
		for _, r := range strings.Split(stripANSITest(b.View().Content), "\n") {
			if m := lineupRowNum.FindStringSubmatch(r); m != nil {
				out = append(out, m[1])
			}
		}
		return out
	}
	thumb := func(b Broadcaster) int {
		for i, r := range strings.Split(stripANSITest(b.View().Content), "\n") {
			if len(r) > b.width-2 && r[b.width-2] == '#' {
				return i
			}
		}
		return -1
	}

	top := slots(base)
	if len(top) < 3 {
		t.Fatalf("the console drew %d slots; there is nothing to scroll", len(top))
	}
	if thumb(base) < 0 {
		t.Fatal("the rail draws no thumb")
	}

	// SCROLLING SHOWS SLOTS THAT WERE BELOW THE FOLD.
	down := base.scrollQueue(6)
	if got := slots(down); equalStrings(got, top) {
		t.Errorf("scrolling changed nothing: %v", got)
	}
	if thumb(down) <= thumb(base) {
		t.Errorf("the thumb sat still while the window moved: %d then %d", thumb(base), thumb(down))
	}

	// THE BOTTOM IS REACHABLE, which is the whole point: the last slot must be
	// drawable or the operator cannot manage it.
	end := base
	for range 40 {
		end = end.scrollQueue(1)
	}
	// THE NUMBER, NOT A CHIP (D-94): the table addresses a slot by its `##.`
	// column, so the last slot is `14` and not `[14]`.
	last := fmt.Sprintf("%02d", MainTrackSlots-1)
	if got := slots(end); !slices.Contains(got, last) {
		t.Errorf("the last slot (%s) is unreachable; the bottom of the queue shows %v", last, got)
	}
	// AND THE THUMB IS STILL DRAWN THERE. A rail that loses its thumb at the end
	// of the list says the list is gone rather than that it is finished.
	if thumb(end) < 0 {
		t.Error("the rail lost its thumb at the bottom of the queue")
	}

	// CLAMPED, NEVER WRAPPED. A list that jumped from its last row to its first
	// would lose the operator's place.
	if got, want := slots(end.scrollQueue(10)), slots(end); !equalStrings(got, want) {
		t.Errorf("scrolling past the end moved the window: %v then %v", want, got)
	}
	if got, want := slots(base.scrollQueue(-5)), top; !equalStrings(got, want) {
		t.Errorf("scrolling above the top moved the window: %v", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// lineupRowNum matches a running-order row by its `##.` column — the address the
// table draws, which since D-94 is where a slot's number lives.
var lineupRowNum = regexp.MustCompile(`^\s+(\d\d)\.\s`)
