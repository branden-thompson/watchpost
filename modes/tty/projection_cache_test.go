package tty

import (
	"fmt"
	"testing"

	"github.com/branden-thompson/watchpost/platform/lineup"
)

// withOperatorCard is the console handed a line-up with the operator's card
// placed at slot, through the message the app sends.
func withOperatorCard(t *testing.T, b Broadcaster, id string, slot int) Broadcaster {
	t.Helper()
	c, err := lineup.Propose(lineup.Card{ID: id, Slot: lineup.LocationReport, Origin: lineup.FromOperator,
		Subject: "Fallbrook, CA", Headline: "Fallbrook, CA", State: lineup.Proposed})
	if err != nil {
		t.Fatal(err)
	}
	if c, err = c.To(lineup.Admitted); err != nil {
		t.Fatal(err)
	}
	l, err := b.lineup.Insert(lineup.MainTrack, c, slot)
	if err != nil {
		t.Fatalf("placing the operator's card: %v", err)
	}
	b, _ = b.Update(LineupMsg{Lineup: l})
	return b
}

// TestTheConsolesProjectionFollowsItsLineup is 0.19.5 Q4.5 (D-22): the console
// works the projection out once per line-up it is handed, and every reader
// sees the line-up the console holds. A new line-up is read at once, and two
// copies of the console from one parent, handed different line-ups, each read
// their own, however their reads interleave, since the copies share one memo.
func TestTheConsolesProjectionFollowsItsLineup(t *testing.T) {
	base := loadedConsole(t, loadedPoolSize)
	ids := func(b Broadcaster) string {
		out := ""
		for _, c := range b.projection(lineup.MainTrack) { // bounded by the track (P10-02)
			out += c.ID + " "
		}
		return out
	}
	want := func(b Broadcaster) string {
		out := ""
		for _, c := range b.lineup.Projection(lineup.MainTrack) { // bounded by the track (P10-02)
			out += c.ID + " "
		}
		return out
	}
	_ = ids(base)
	a := withOperatorCard(t, base, "operator-a", 1)
	b := withOperatorCard(t, base, "operator-b", 3)
	if a.memo != b.memo {
		t.Fatal("the two copies do not share a memo; this checks nothing")
	}
	for i, m := range []Broadcaster{a, b, a, base, b} {
		if got, exp := ids(m), want(m); got != exp {
			t.Errorf("read %d: the console projects %s; its line-up is %s", i, got, exp)
		}
	}
	if fmt.Sprint(ids(a)) == fmt.Sprint(ids(b)) {
		t.Error("the two line-ups project the same running order; the fixture cannot tell them apart")
	}
}

// TestTheConsolesTableRowsAreTheRowsItWouldDraw is 0.19.5 Q4.5's other half
// (D-22): the console's memo keeps the railed rows with the spans, so the rows
// it hands back must be the rows the console would rail now. For every field
// the walk can change, with the memo filled for the unchanged console first, a
// copy reads the rows it would draw itself and never the base's.
func TestTheConsolesTableRowsAreTheRowsItWouldDraw(t *testing.T) {
	base := loadedConsole(t, loadedPoolSize)
	base.width, base.height = 160, 74
	const used = 20
	fresh := func(b Broadcaster) string { return fmt.Sprint(b.railTables(b.buildSpans(used))) }
	if len(base.tableRows(used)) == 0 {
		t.Fatal("the fixture rails no rows; this checks nothing")
	}
	checked := 0
	perturbEach(t, base, consoleExcuse, consoleWriter, func(field string, next Broadcaster) {
		_ = base.tableRows(used) // the memo holds the base console's rows
		if got, want := fmt.Sprint(next.tableRows(used)), fresh(next); got != want {
			t.Errorf("changing %s: the console's memo hands back rows it would not draw now (broadcaster_memo.go, tableRows)", field)
		}
		checked++
	})
	if checked == 0 {
		t.Error("the walk perturbed nothing")
	}
}
