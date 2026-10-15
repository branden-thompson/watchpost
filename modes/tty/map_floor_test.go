package tty

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// notedFeed is a feed with three notes, as the heavy workload's answers
// carry: what pushes the map under its floor at 149x38.
func notedFeed(n int) func(context.Context, MapAsk) MapFeed {
	return func(context.Context, MapAsk) MapFeed {
		var notes []string
		for i := range n {
			notes = append(notes, strings.Repeat("A note the feed carries about the alerts in view. ", 1)+string(rune('A'+i)))
		}
		return MapFeed{Notes: notes}
	}
}

// sizedMap opens the map at a terminal size with the radar timeline held and
// the feed's notes in.
func sizedMap(t *testing.T, w, h, notes int) Dashboard {
	t.Helper()
	d := mapDash(t, Config{MapFeed: notedFeed(notes), MapRadar: radarFeed(t, "MRMS", new([]string))})
	m, _ := d.Update(tea.WindowSizeMsg{Width: w, Height: h})
	d = m.(Dashboard)
	d, _ = pressKey(d, "g")
	return feedAndSettle(t, d)
}

// TestTheMapKeepsItsFloorWhenTheTerminalCanHoldIt is D-159 as D-194 sizes it:
// the window runs from the top inset to the bottom one, so its own status,
// notes and radar timeline never push the map under 69x12 where the terminal
// can hold it (W14, C-7).
func TestTheMapKeepsItsFloorWhenTheTerminalCanHoldIt(t *testing.T) {
	d := sizedMap(t, 149, 38, 3)
	if len(d.mapPane.notes) != 3 || !d.radarTimelineOn() {
		t.Fatal("control: the notes or the radar timeline are not in, so this proves nothing")
	}
	if !d.mapFits() {
		t.Fatalf("at 149x38 the map is %v, under its floor %v", d.mapBodySize(), mapMinBody)
	}
	if d.modalMax() != 38-topInset-bottomInset-2 {
		t.Errorf("the window is %d rows; its ceiling is the rows between the insets but its borders (32)", d.modalMax())
	}
	if strings.Contains(stripANSITest(d.View().Content), "The map needs") {
		t.Error("the window shows the floor's notice in place of the map")
	}
	// The ceiling holds them: here the notes still show in full, nothing collapsed.
	if notes := stripANSITest(strings.Join(d.noteLines(d.mapTextW()), "\n")); !strings.Contains(notes, "A note the feed carries") {
		t.Errorf("the notes collapsed where the window could grow to hold them:\n%s", notes)
	}
}

// TestTheMapWindowRunsInsetToInset is D-194: the Map window alone reaches
// the global inset at the top and the bottom - its top border on the frame's
// first row under the top padding, its bottom border on the last row above
// the bottom one - wherever the map is drawn. (Under the floor the window holds
// only the floor's notice, from the top inset down.)
func TestTheMapWindowRunsInsetToInset(t *testing.T) {
	for _, h := range []int{38, 44, 60} {
		d := sizedMap(t, 200, h, 3)
		if !d.mapFits() {
			t.Fatalf("control: at 200x%d the map is under its floor", h)
		}
		if want := h - topInset - bottomInset - 2; d.modalMax() != want {
			t.Errorf("at 200x%d the window's body is %d rows, want %d", h, d.modalMax(), want)
		}
		rows := strings.Split(stripANSITest(d.View().Content), "\n")
		top, bottom := -1, -1
		for i, r := range rows {
			if strings.Contains(r, "┌") && top < 0 {
				top = i
			}
			if strings.Contains(r, "┘") {
				bottom = i
			}
		}
		if top != topInset || bottom != h-bottomInset-1 {
			t.Errorf("at 200x%d the window runs rows %d to %d; want %d to %d", h, top, bottom, topInset, h-bottomInset-1)
		}
	}
}

// TestTheNotesYieldWhereTheWindowCannot is D-159's last step: where even the
// ceiling cannot hold the floor, the notes collapse to one line naming the
// Status window, before the map is given up.
func TestTheNotesYieldWhereTheWindowCannot(t *testing.T) {
	d := sizedMap(t, 149, 31, 3) // ceiling 25: 2 status, 9 timeline, 3 notes leave 11; one note line leaves 13
	if !d.mapFits() {
		t.Fatalf("at 149x31 the notes did not yield: the map is %v", d.mapBodySize())
	}
	notes := d.noteLines(d.mapTextW())
	joined := stripANSITest(strings.Join(notes, "\n"))
	if !strings.Contains(joined, "Status") || strings.Contains(joined, "A note the feed carries") {
		t.Errorf("the notes did not collapse to one line naming Status:\n%s", joined)
	}
	// AND THE POINTER IS TRUE: the Status window carries them, under the map's hosts.
	d.cfg.MapSources = []MapSource{{Name: "tiles", Layers: "Basemap"}}
	status := stripANSITest(strings.Join(d.statusBlocks(0).sources, "\n"))
	for _, n := range d.mapPane.notes {
		if !strings.Contains(strings.Join(strings.Fields(status), " "), strings.Join(strings.Fields(n), " ")) {
			t.Errorf("the notes point to the Status window, which does not say %q:\n%s", n, status)
		}
	}
}

// TestTheWindowNeverPassesItsCeiling: on a terminal too short even with the
// notes yielded, the window stops at the bottom inset (D-194) and the
// floor's notice says so honestly (FR-1.4) - the dashboard is never covered.
func TestTheWindowNeverPassesItsCeiling(t *testing.T) {
	d := sizedMap(t, 149, 26, 3) // ceiling 20: 2 status, 9 timeline, 1 note line need 24
	if d.modalMax() != 26-topInset-bottomInset-2 {
		t.Errorf("the window is %d rows; it stops at the ceiling, 20", d.modalMax())
	}
	if d.mapFits() {
		t.Fatal("control: the map fits here, so this proves nothing about the ceiling")
	}
}

// TestAMapNotDrawnIsReportedToTheInstrument is the render check W14's
// workload owes (the HUM LEAD: "everything that's supposed to render,
// ACTUALLY renders"): a draw that falls under the floor - the notice shown in
// the map's place - is said to the instrument, so a run that loses its map
// cannot pass for one that drew it.
func TestAMapNotDrawnIsReportedToTheInstrument(t *testing.T) {
	var got []Timing
	d := mapDash(t, Config{MapFeed: notedFeed(3), MapRadar: radarFeed(t, "MRMS", new([]string)), Timed: func(tm Timing) { got = append(got, tm) }})
	m, _ := d.Update(tea.WindowSizeMsg{Width: 149, Height: 28})
	d = m.(Dashboard)
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	under := 0
	for _, tm := range got {
		if tm.Trigger == "render" && tm.Event == "below-floor" {
			under++
		}
	}
	if d.mapFits() || under == 0 {
		t.Errorf("the map was not drawn (fits %v) and the instrument was told %d times", d.mapFits(), under)
	}
	got = nil
	d = sizedMap(t, 149, 38, 3)
	d.cfg.Timed = func(tm Timing) { got = append(got, tm) }
	d = d.renderMap()
	for _, tm := range got {
		if tm.Event == "below-floor" {
			t.Error("a map that was drawn was reported below its floor")
		}
	}
}
