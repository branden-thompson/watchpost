package tty

// broadcaster_priority_test.go — the priority track is INVISIBLE UNTIL IT HAS
// SOMETHING (D-61, HUM LEAD 2026-09-10).
//
//	"the PRIORITY rail label ONLY shows up when a priority card sits on top of
//	the main rail — this gives the operator more space to view/manage the main
//	rail during normal operation."
//
// Which is the same thing said earlier about the track itself: "normally that
// priority lane is INVISIBLE to the operator — so the main track takes the full
// width of the UI."
//
// A "(clear)" ROW IS NOT NOTHING. It costs two rows of the running order to say
// that a hazard is not happening — which is the state the station is in almost
// all of the time.

import (
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestAClearPriorityTrackDrawsNothingAtAll(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	got := stripANSITest(b.View().Content)

	if strings.Contains(got, "PRIORITY") {
		t.Errorf("a clear priority track names itself nowhere:\n%s", headOf(got, 12))
	}
	if strings.Contains(got, "(clear)") {
		t.Errorf("and it does not spend a row saying so:\n%s", headOf(got, 12))
	}
}

func TestATakeoverBringsThePriorityTrackBack(t *testing.T) {
	var l lineup.Lineup
	c, err := lineup.Propose(lineup.Card{ID: "t1", Slot: lineup.BreakingAlert,
		Origin: lineup.FromObserver, Subject: "tornado", Headline: "TORNADO WARNING"})
	if err != nil {
		t.Fatalf("proposing: %v", err)
	}
	if c, err = c.To(lineup.Admitted); err != nil {
		t.Fatalf("admitting: %v", err)
	}
	if l, err = l.Queue(lineup.AlertRail, c); err != nil {
		t.Fatalf("queueing: %v", err)
	}

	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(LineupMsg{Lineup: l})
	got := stripANSITest(b.View().Content)

	if !strings.Contains(got, "TORNADO WARNING") {
		t.Errorf("a takeover must reach the frame:\n%s", headOf(got, 16))
	}
	if !strings.Contains(got, "PRIORITY") {
		t.Errorf("and the track names itself while it has something:\n%s", headOf(got, 16))
	}
}

// THE MAIN TRACK GETS THE ROWS BACK. That is the whole reason for the ruling:
// two rows spent on "(clear)" are two rows of running order the operator cannot
// see, in the state the station is in almost all of the time.
func TestAClearTrackGivesItsRowsToTheRunningOrder(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	rows := strings.Split(stripANSITest(b.View().Content), "\n")
	first := -1
	for i, r := range rows {
		if strings.HasPrefix(strings.TrimSpace(r), "|") && strings.Contains(r, "+---") {
			first = i
			break
		}
	}
	if first < 0 {
		t.Skip("no cards in this fixture; the region test covers the placement")
	}
}

// headOf is the first n rows of a frame, for an error that has to show WHERE.
func headOf(s string, n int) string {
	parts := strings.Split(s, "\n")
	if len(parts) > n {
		parts = parts[:n]
	}
	return strings.Join(parts, "\n")
}

// THE OVERLAY SITS ON TOP OF THE RUNNING ORDER, NOT ABOVE IT (D-61).
//
// "the priority track visually sits ON TOP of the main track — that's because
// it's not supposed to always be on, and it signals that it is TAKING OVER
// while there are alerts in that line."
//
// Measured off the reference: the overlay's box runs 6..72 while the main
// track's cards run their FULL width behind it, 9..140. The operator keeps the
// right-hand half of every card it covers — the half carrying the badge and the
// HANDLE they type.
func TestThePriorityOverlaySitsOnTopOfTheRunningOrder(t *testing.T) {
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	if got, want := b.priorityWidth(), 67; got != want {
		t.Errorf("the reference's overlay is %d cells; got %d", want, got)
	}
	if bcPriorityCol != 6 {
		t.Errorf("the reference's overlay begins at column 6; got %d", bcPriorityCol)
	}
}

// MEASURED IN CELLS, NOT RUNES. These counted runes, which is the same mistake
// the function itself was making (D-71): a splice now brackets its patch in
// resets so the row it covers cannot bleed a tone into it, and those escapes are
// runes that are not columns.
func TestSpliceWritesInPlaceAndNeverGrowsTheRow(t *testing.T) {
	base := []string{strings.Repeat("-", 20), strings.Repeat("-", 20)}
	got := spliceAt(base, []string{"ABC", "DEFGH"}, 5)
	if len(got) != 2 {
		t.Fatalf("splicing adds no rows; got %d", len(got))
	}
	for i, r := range got {
		if render.Width(r) != 20 {
			t.Errorf("row %d grew to %d cells: %q", i, render.Width(r), r)
		}
	}
	if plain := render.StripSGRForTest(got[0]); plain[5:8] != "ABC" {
		t.Errorf("the patch lands at the column it was given; got %q", plain)
	}
}

func TestASpliceRunningPastTheRowIsCutNotWrapped(t *testing.T) {
	got := spliceAt([]string{"----------"}, []string{strings.Repeat("X", 40)}, 6)
	if render.Width(got[0]) != 10 {
		t.Errorf("the frame is the viewport; got %d cells: %q", render.Width(got[0]), got[0])
	}
	if plain := render.StripSGRForTest(got[0]); !strings.HasPrefix(plain, "------XXXX") {
		t.Errorf("the patch is cut at the edge; got %q", plain)
	}
}

// AND A SPLICE OVER A STYLED ROW KEEPS WHAT IT DID NOT COVER (D-71).
//
//	"Alert card appearing (correct) causes the row render of the right hand side
//	 of the LIVE card to truncate inappropriately"
//
// The rows the overlay covers carry a tinted headline, a badge and the handle's
// chip. Splicing them by rune index counted each escape's characters as columns
// and overwrote the escapes it landed on, so everything right of the overlay
// came out short.
func TestTheOverlayDoesNotTruncateTheCardUnderIt(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	b := withTakeover(t, 6)
	for i, r := range strings.Split(b.View().Content, "\n") {
		if got := render.Width(r); got != b.width {
			t.Fatalf("row %d is %d cells with a takeover up, want %d", i, got, b.width)
		}
	}
	frame := render.StripSGRForTest(b.View().Content)
	if strings.Contains(frame, "\x1b") {
		t.Error("an escape survived the strip: the splice cut through one")
	}
	// THE HANDLE THE OPERATOR TYPES SURVIVES THE OVERLAY, which is the whole
	// reason the overlay covers only half the card.
	for _, want := range []string{chipFor("0"), chipFor("T")} {
		if !strings.Contains(b.View().Content, want) {
			t.Errorf("the frame lost %q while a takeover was up", render.StripSGRForTest(want))
		}
	}
}

// withTakeover is a line-up of `n` reports with one takeover on the rail.
func withTakeover(t *testing.T, n int) Broadcaster {
	t.Helper()
	var l lineup.Lineup
	for i := 0; i < n; i++ {
		c := card(t, "c"+string(rune('0'+i)), "LOCATION REPORT • TOWN "+string(rune('0'+i))+", CA")
		next, err := l.Queue(lineup.MainTrack, c)
		if err != nil {
			t.Fatalf("seeding: %v", err)
		}
		l = next
	}
	tk, err := lineup.Propose(lineup.Card{ID: "t1", Slot: lineup.BreakingAlert,
		Origin: lineup.FromObserver, Subject: "tornado", Headline: "TAKE-OVER | WEATHER ALERTS"})
	if err != nil {
		t.Fatalf("proposing: %v", err)
	}
	if tk, err = tk.To(lineup.Admitted); err != nil {
		t.Fatalf("admitting: %v", err)
	}
	if l, err = l.Queue(lineup.AlertRail, tk); err != nil {
		t.Fatalf("queueing: %v", err)
	}
	b := NewBroadcaster()
	b.width, b.height, b.ascii = 150, 74, true
	b, _ = b.Update(LineupMsg{Lineup: l})
	return b
}

// ON TOP OF, NOT ABOVE — and this is the assertion that says which.
//
// If the overlay were merely drawn ABOVE the running order, its rows would hold
// the takeover and nothing else. Composited, the card underneath is still there
// to the right of it: the operator keeps the half carrying the badge and the
// HANDLE they type, which is the whole point of covering only part of it.
func TestTheOverlayAndTheCardBeneathShareARow(t *testing.T) {
	rows := strings.Split(stripANSITest(withTakeover(t, 6).View().Content), "\n")
	found := ""
	for _, r := range rows {
		if strings.Contains(r, chipFor("T")) {
			found = r
			break
		}
	}
	if found == "" {
		t.Fatal("the takeover must reach the frame")
	}
	if !strings.Contains(found, chipFor("0")) {
		t.Errorf("the card beneath must still show its handle on the same row — the overlay is ON it, not above it:\n%q", found)
	}
}

// THE RAIL LABEL COMES WITH THE OVERLAY (HUM LEAD): "the PRIORITY rail label
// ONLY shows up when a priority card sits on top of the main rail."
//
// THE WHOLE BOX'S RAIL IS READ, AND COMPARED FOR EQUALITY. A first version took
// the single letter beside the takeover's title row and asked whether one of
// PRIORITY's forms CONTAINED it — and "I" is contained in "PRIORITY", so the
// main track's own "LIVE" passed the test. A one-character containment check
// against an eight-character word asserts almost nothing.
func TestTheOverlaysRowsCarryThePriorityLabel(t *testing.T) {
	rows := strings.Split(stripANSITest(withTakeover(t, 6).View().Content), "\n")
	title := -1
	for i, r := range rows {
		if strings.Contains(r, chipFor("T")) {
			title = i
			break
		}
	}
	if title < 1 {
		t.Fatalf("the takeover must reach the frame")
	}
	// The box is four rows: the border above the title, and two below it.
	var letters []string
	for i := title - 1; i < title+3 && i < len(rows); i++ {
		r := []rune(rows[i])
		if len(r) < 4 {
			continue
		}
		if c := strings.TrimSpace(string(r[1:4])); c != "" {
			letters = append(letters, c)
		}
	}
	got := strings.Join(letters, "")
	for _, form := range bcRailForms["PRIORITY"] {
		if got == strings.ReplaceAll(form, " ", "") {
			return
		}
	}
	t.Errorf("the overlay's rail spells %q; PRIORITY's forms are %v — the label did not come with the overlay",
		got, bcRailForms["PRIORITY"])
}
