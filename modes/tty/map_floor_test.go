package tty

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// notedFeed is a feed with three notes, as the heavy workload's answers
// carry: what pushed the map under its floor at 149x38.
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

// TestTheMapKeepsItsFloorWhenTheTerminalCanHoldIt is D-159: the window grows
// - only as far as the floor needs, up to the terminal's height minus 8 - so
// its own status, notes and radar timeline never push the map under 69x12
// where the terminal can hold it. At 149x38 the map fell to 11 rows and the
// window showed "The map needs…" in its place (W14, C-7).
func TestTheMapKeepsItsFloorWhenTheTerminalCanHoldIt(t *testing.T) {
	d := sizedMap(t, 149, 38, 3)
	if len(d.mapPane.notes) != 3 || !d.radarTimelineOn() {
		t.Fatal("control: the notes or the radar timeline are not in, so this proves nothing")
	}
	if !d.mapFits() {
		t.Fatalf("at 149x38 the map is %v, under its floor %v", d.mapBodySize(), mapMinBody)
	}
	if d.modalMax() > 38-8 {
		t.Errorf("the window is %d rows; its ceiling is the terminal's height minus 8 (30)", d.modalMax())
	}
	if strings.Contains(stripANSITest(d.View().Content), "The map needs") {
		t.Error("the window shows the floor's notice in place of the map")
	}
	// Growing is first: here the notes still show in full, nothing collapsed.
	if notes := stripANSITest(strings.Join(d.noteLines(d.mapTextW()), "\n")); !strings.Contains(notes, "A note the feed carries") {
		t.Errorf("the notes collapsed where the window could grow to hold them:\n%s", notes)
	}
}

// TestTheWindowStaysItsSizeWhenTheMapFits is U1-13 kept: where the map
// already fits, the window is the ~80 % it was - it grows only for the floor.
func TestTheWindowStaysItsSizeWhenTheMapFits(t *testing.T) {
	d := sizedMap(t, 200, 60, 3)
	if want := max(5, min(60-8, max(60*80/100-5, mapMinBody.Rows+1))); d.modalMax() != want {
		t.Errorf("at 200x60 the window is %d rows, want U1-13's %d", d.modalMax(), want)
	}
}

// TestTheNotesYieldWhereTheWindowCannot is D-159's last step: where even the
// ceiling cannot hold the floor, the notes collapse to one line naming the
// Status window, before the map is given up.
func TestTheNotesYieldWhereTheWindowCannot(t *testing.T) {
	d := sizedMap(t, 149, 33, 3) // ceiling 25: 2 status, 9 timeline, 3 notes leave 11; one note line leaves 13
	if !d.mapFits() {
		t.Fatalf("at 149x33 the notes did not yield: the map is %v", d.mapBodySize())
	}
	notes := d.noteLines(d.mapTextW())
	joined := stripANSITest(strings.Join(notes, "\n"))
	if !strings.Contains(joined, "Status") || strings.Contains(joined, "A note the feed carries") {
		t.Errorf("the notes did not collapse to one line naming Status:\n%s", joined)
	}
	// AND THE POINTER IS TRUE: the Status window's MAP STATUS carries them.
	d.cfg.MapSources = []MapSource{{Name: "tiles", Layers: "Basemap"}}
	status := stripANSITest(strings.Join(d.statusBlocks(0).maps, "\n"))
	for _, n := range d.mapPane.notes {
		if !strings.Contains(strings.Join(strings.Fields(status), " "), strings.Join(strings.Fields(n), " ")) {
			t.Errorf("the notes point to the Status window, which does not say %q:\n%s", n, status)
		}
	}
}

// TestTheWindowNeverPassesItsCeiling: on a terminal too short even with the
// notes yielded, the window stops at the terminal's height minus 8 and the
// floor's notice says so honestly (FR-1.4) - the dashboard is never covered.
func TestTheWindowNeverPassesItsCeiling(t *testing.T) {
	d := sizedMap(t, 149, 28, 3) // ceiling 20: 2 status, 9 timeline, 1 note line need 24
	if d.modalMax() != 28-8 {
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
