package tty

import (
	"slices"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	tuimaps "github.com/branden-thompson/go-tuimaps"
)

// A MAP CLOSED A WHILE IS LET GO (D-221): closing the map schedules its
// release; when it comes and the map is still closed, the library's map is
// closed and the window's hold on it - the pictures it was handed - goes, and
// the next open builds a new map.
func TestAMapClosedAWhileIsLetGo(t *testing.T) {
	builds := 0
	d := mapDash(t, Config{NewMap: func(s tuimaps.Size) (*tuimaps.Map, error) { builds++; return embeddedMap(s) }})
	d.mapReleaseAfter = time.Millisecond
	d, _ = pressKey(d, "g")
	if d.mapPane.m == nil || builds != 1 {
		t.Fatalf("the map was not built (%d builds)", builds)
	}
	var calls []string
	d.mapPane.calls = &calls
	d.mapPane.viewGen, d.mapPane.viewAsked, d.mapPane.feedGen, d.mapPane.lanes[laneAlerts].seq, d.mapPane.fcGen = 7, 7, 7, 7, 7 // asks made before the close
	d, cmd := pressKey(d, "esc")
	rel, ok := findRelease(t, cmd)
	if !ok {
		t.Fatal("closing the map scheduled no release")
	}
	m, _ := d.Update(rel)
	d = m.(Dashboard)
	if d.mapPane.m != nil || len(d.mapPane.given) != 0 || len(d.mapPane.radarGiven) != 0 || len(d.mapPane.tempGiven) != 0 {
		t.Error("the released window still holds its map or what it handed it")
	}
	if !slices.Contains(calls, "Close") {
		t.Error("the library's map was not closed")
	}
	if p := d.mapPane; p.viewGen < 7 || p.viewAsked < 7 || p.feedGen < 7 || p.lanes[laneAlerts].seq < 7 || p.fcGen < 7 {
		t.Errorf("a generation went back (view %d, asked %d, feed %d, seq %d, forecast %d): an old answer could be taken for a new one", p.viewGen, p.viewAsked, p.feedGen, p.lanes[laneAlerts].seq, p.fcGen)
	}
	d, _ = pressKey(d, "g")
	if d.mapPane.m == nil || builds != 2 {
		t.Errorf("the open after a release built %d maps in all; want a new one", builds)
	}
}

// A MAP REOPENED IN TIME IS KEPT (D-221): a release scheduled by a close is
// void once the map opens again, or closes again - only the newest close's
// release lets it go.
func TestAMapReopenedInTimeIsKept(t *testing.T) {
	builds := 0
	d := mapDash(t, Config{NewMap: func(s tuimaps.Size) (*tuimaps.Map, error) { builds++; return embeddedMap(s) }})
	d.mapReleaseAfter = time.Millisecond
	d, _ = pressKey(d, "g")
	d, cmd := pressKey(d, "esc")
	first, ok := findRelease(t, cmd)
	if !ok {
		t.Fatal("closing the map scheduled no release")
	}
	d, _ = pressKey(d, "g") // reopened before the release came
	m, _ := d.Update(first)
	d = m.(Dashboard)
	if d.mapPane.m == nil || builds != 1 {
		t.Fatal("a map open again was let go by the release its earlier close scheduled")
	}
	d, cmd = pressKey(d, "esc")
	second, _ := findRelease(t, cmd)
	m, _ = d.Update(first) // the first close's, after a second close
	d = m.(Dashboard)
	if d.mapPane.m == nil {
		t.Error("an earlier close's release let the map go")
	}
	m, _ = d.Update(second)
	if m.(Dashboard).mapPane.m != nil {
		t.Error("the newest close's release did not let it go")
	}
}

// findRelease runs a command's messages and finds the release among them.
func findRelease(t *testing.T, cmd tea.Cmd) (mapReleaseMsg, bool) {
	t.Helper()
	for _, msg := range msgsOf(t, cmd) {
		if r, ok := msg.(mapReleaseMsg); ok {
			return r, true
		}
	}
	return mapReleaseMsg{}, false
}
