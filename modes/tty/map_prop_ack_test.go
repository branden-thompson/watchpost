package tty

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// ackDash is the map open on Oceanside, the acknowledgement last seen at a
// version, its saves recorded.
func ackDash(t *testing.T, seen int, ascii bool, saved *[]int) Dashboard {
	t.Helper()
	cfg := Config{ASCII: ascii, PropagationAck: seen, SavePropagationAck: func(v int) error { *saved = append(*saved, v); return nil }}
	return openMap(t, cfg, 133, 44)
}

// ackPress sends a key the window stack routes, and runs what a key on the
// acknowledgement asks for.
func ackPress(t *testing.T, d Dashboard, key string) Dashboard {
	t.Helper()
	m, cmd := d.Update(mapKeyMsg(t, key))
	if cmd != nil && d.modal == modalPropAck {
		runAll(cmd) // the save a close asks for; any other window's commands may wait on timers
	}
	return m.(Dashboard)
}

// runAll runs a command and any batch it returns, dropping their messages.
func runAll(cmd tea.Cmd) {
	msg := cmd()
	if b, ok := msg.(tea.BatchMsg); ok {
		for _, c := range b {
			if c != nil {
				runAll(c)
			}
		}
	}
}

// TestTheAcknowledgementShowsBeforeTheFirstFetch is W2.5 (FR-11.1, D-81):
// the first P opens the acknowledgement over the map, in the same key press
// that enters the Propagation mode, and it says what D-46 and D-81 rule.
func TestTheAcknowledgementShowsBeforeTheFirstFetch(t *testing.T) {
	var saved []int
	d := ackPress(t, ackDash(t, 0, false, &saved), "P")
	if d.modal != modalPropAck || d.mapMode() != modePropagation {
		t.Fatalf("P showed window %d in mode %d; want the acknowledgement over the Propagation mode", d.modal, d.mapMode())
	}
	words := strings.Join(propAckText, " ")
	for _, want := range []string{"does not transmit", "reference", "responsible for following all applicable laws", "GIRO", "NOAA", "IP address", "MAP STATUS", "Enter or Esc"} {
		if !strings.Contains(words, want) {
			t.Errorf("the acknowledgement does not say %q", want)
		}
	}
	if !slices.ContainsFunc(d.under, func(w stackedWindow) bool { return w.modal == modalMap }) {
		t.Error("the map is not under the acknowledgement")
	}
}

// TestOnlyEnterOrEscCloseIt is W2.5 (FR-11.2): no other key leaves it -
// not the map's, not the Observer's - and Enter or Esc returns to the map in
// the Propagation mode.
func TestOnlyEnterOrEscCloseIt(t *testing.T) {
	for _, closer := range []string{"enter", "esc"} {
		var saved []int
		d := ackPress(t, ackDash(t, 0, false, &saved), "P")
		if d.modal != modalPropAck {
			t.Fatalf("P showed window %d, not the acknowledgement", d.modal)
		}
		for _, k := range []string{"g", "q", "?", "P", "R", "O", "x", "up", "space"} {
			if d = ackPress(t, d, k); d.modal != modalPropAck {
				t.Fatalf("%s left the acknowledgement for window %d", k, d.modal)
			}
		}
		if len(saved) != 0 {
			t.Errorf("the acknowledgement was recorded before it closed: %v", saved)
		}
		if d = ackPress(t, d, closer); d.modal != modalMap || d.mapMode() != modePropagation {
			t.Errorf("%s gave window %d in mode %d; want the map in the Propagation mode", closer, d.modal, d.mapMode())
		}
	}
}

// TestClosingTheAcknowledgementRecordsItSeen is W2.5 (FR-11.2, A-9):
// closing it records the wording's version, so the Propagation mode opens
// without it after that, this session or the next.
func TestClosingTheAcknowledgementRecordsItSeen(t *testing.T) {
	var saved []int
	d := ackPress(t, ackPress(t, ackDash(t, 0, false, &saved), "P"), "esc")
	if !slices.Equal(saved, []int{propAckVersion}) || d.cfg.PropagationAck != propAckVersion {
		t.Fatalf("saved %v, held %d; want version %d once", saved, d.cfg.PropagationAck, propAckVersion)
	}
	if d = ackPress(t, ackPress(t, d, "P"), "P"); d.modal != modalMap || d.mapMode() != modePropagation {
		t.Errorf("the Propagation mode again showed window %d in mode %d", d.modal, d.mapMode())
	}
	if next := ackPress(t, ackDash(t, propAckVersion, false, &saved), "P"); next.modal != modalMap {
		t.Errorf("a recorded acknowledgement showed again next session: window %d", next.modal)
	}
}

// TestAChangedAcknowledgementShowsAgain is W2.5 (FR-11.3, A-9): one seen in
// an older wording is shown once more.
func TestAChangedAcknowledgementShowsAgain(t *testing.T) {
	var saved []int
	if d := ackPress(t, ackDash(t, propAckVersion-1, false, &saved), "P"); d.modal != modalPropAck {
		t.Errorf("an older wording seen: window %d; want the acknowledgement again", d.modal)
	}
	for _, c := range []struct {
		seen, current int
		due           bool
	}{{0, 2, true}, {1, 2, true}, {2, 2, false}, {3, 2, false}} {
		if got := ackDue(c.seen, c.current); got != c.due {
			t.Errorf("seen %d, wording %d: due %v, want %v", c.seen, c.current, got, c.due)
		}
	}
}

// TestTheCursorIsPlacedOnTheAcknowledgement is W2.5 (FR-11.4, D-81): the
// terminal cursor stands at the start of its first line, and nowhere once
// it closes.
func TestTheCursorIsPlacedOnTheAcknowledgement(t *testing.T) {
	var saved []int
	d := ackPress(t, ackDash(t, 0, false, &saved), "P")
	v := d.View()
	if v.Cursor == nil {
		t.Fatal("no cursor on the acknowledgement")
	}
	lines := strings.Split(stripANSITest(v.Content), "\n")
	if v.Cursor.Y >= len(lines) {
		t.Fatalf("the cursor is on row %d of %d", v.Cursor.Y, len(lines))
	}
	row := []rune(lines[v.Cursor.Y])
	first := []rune(propAckText[0])[:12]
	if v.Cursor.X+len(first) > len(row) || string(row[v.Cursor.X:v.Cursor.X+len(first)]) != string(first) {
		t.Errorf("the cursor is at %d,%d on %q; want the start of %q", v.Cursor.X, v.Cursor.Y, string(row), propAckText[0])
	}
	if c := ackPress(t, d, "enter").View().Cursor; c != nil {
		t.Errorf("the cursor stayed at %v after the acknowledgement closed", c)
	}
}

// TestTheAcknowledgementReadsWithoutThePicture is W2.5 (FR-11.4): under
// --ascii it is the same words in plain text.
func TestTheAcknowledgementReadsWithoutThePicture(t *testing.T) {
	var saved []int
	d := ackPress(t, ackDash(t, 0, true, &saved), "P")
	screen := stripANSITest(d.View().Content)
	flat := strings.Join(strings.Fields(screen), " ")
	for _, want := range []string{"does not transmit", "MAP STATUS", "Enter or Esc"} {
		if !strings.Contains(flat, want) {
			t.Errorf("--ascii does not say %q:\n%s", want, screen)
		}
	}
	for _, r := range screen {
		if r > 127 && r != '°' {
			t.Fatalf("--ascii carries %q", r)
		}
	}
}
