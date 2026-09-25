package tty

// map_window_test.go — 0.18.0 W1.1, W1.3, W2.1, W2.2: the map window opens on
// its key, draws in Update, and never calls the library from View.

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"
	"github.com/branden-thompson/go-tuimaps/assets"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// embeddedMap is a map with the library's embedded tiles and no source: it
// draws a basemap and reaches nothing at all.
func embeddedMap(size tuimaps.Size) (*tuimaps.Map, error) {
	return tuimaps.New(tuimaps.WithSize(size.Cols, size.Rows), tuimaps.Embed(assets.Tile, assets.MaxZoom))
}

// placedSnap is the test snapshot with its one location placed where it is.
func placedSnap() *snapshot.Snapshot {
	s := snap()
	s.Locations[0].Lat, s.Locations[0].Lon = 33.2, -117.38 // Oceanside, CA
	return s
}

// mapDash is the dashboard at 133x44 with a placed location and maps on.
func mapDash(t *testing.T, cfg Config) Dashboard {
	t.Helper()
	if cfg.Version == "" {
		cfg.Version = "0.18.0-test"
	}
	if cfg.NewMap == nil {
		cfg.NewMap = embeddedMap
	}
	m, err := NewDashboard(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: placedSnap()})
	d := model.(Dashboard)
	t.Cleanup(d.closeMap)
	return d
}

func pressKey(d Dashboard, key string) (Dashboard, tea.Cmd) {
	msg := tea.KeyPressMsg{Code: rune(key[0]), Text: key}
	if key == "esc" {
		msg = tea.KeyPressMsg{Code: tea.KeyEscape}
	}
	m, cmd := d.Update(msg)
	return m.(Dashboard), cmd
}

// settleMap runs the map's work as Bubble Tea would - each Work command's
// message fed back through Update - until nothing more is pending.
func settleMap(t *testing.T, d Dashboard) Dashboard {
	t.Helper()
	for range 50 {
		cmd := d.mapWorkCmd()
		if cmd == nil {
			return d
		}
		m, _ := d.Update(cmd())
		d = m.(Dashboard)
	}
	t.Fatal("the map's work never settled")
	return d
}

// TestTheMapKeyOpensAndClosesTheWindow is W1.1 (FR-1.1): g opens the map
// window and closes it again, and esc closes it too.
func TestTheMapKeyOpensAndClosesTheWindow(t *testing.T) {
	d := mapDash(t, Config{})
	d, _ = pressKey(d, "g")
	if d.modal != modalMap {
		t.Fatalf("g opened %s, want the map", modalName(d.modal))
	}
	if !strings.Contains(stripANSITest(d.View().Content), "Map · Oceanside, CA") {
		t.Errorf("the window is not titled with the place:\n%s", stripANSITest(d.View().Content))
	}
	d, _ = pressKey(d, "g")
	if d.modal != modalNone {
		t.Errorf("a second g left %s open", modalName(d.modal))
	}
	d, _ = pressKey(d, "g")
	d, _ = pressKey(d, "esc")
	if d.modal != modalNone {
		t.Errorf("esc left %s open", modalName(d.modal))
	}
}

// TestHelpListsTheMapKey is W1.1 (FR-1.1): the key is a keymap action Help
// shows.
func TestHelpListsTheMapKey(t *testing.T) {
	d := mapDash(t, Config{})
	help := strings.Join(d.helpLines(d.opts()), "\n")
	if !strings.Contains(help, "g ") || !strings.Contains(help, "Map") {
		t.Errorf("Help does not list the map key:\n%s", help)
	}
}

// TestAKeysOverrideRebindsTheMap is W1.1 (FR-1.1): [keys] rebinds it.
func TestAKeysOverrideRebindsTheMap(t *testing.T) {
	d := mapDash(t, Config{KeyOverrides: term.KeyMap{actMap: {Keys: []string{"G"}, Help: "Map"}}})
	d, _ = pressKey(d, "g")
	if d.modal == modalMap {
		t.Error("g still opens the map after it was rebound")
	}
	d, _ = pressKey(d, "G")
	if d.modal != modalMap {
		t.Errorf("the rebound key opened %s, want the map", modalName(d.modal))
	}
}

// TestNoSelectionIsAStatedState is W1.3 (FR-1.3): before the first snapshot
// there is no selection, and the window says so rather than drawing a blank
// or a guess; no map is built for it.
func TestNoSelectionIsAStatedState(t *testing.T) {
	built := 0
	m, err := NewDashboard(Config{Version: "t", NewMap: func(size tuimaps.Size) (*tuimaps.Map, error) { built++; return embeddedMap(size) }})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	d := model.(Dashboard)
	d, _ = pressKey(d, "g")
	out := stripANSITest(d.View().Content)
	if !strings.Contains(out, noSelectionText) {
		t.Errorf("with no selection the window does not say so:\n%s", out)
	}
	if built != 0 {
		t.Errorf("a map was built %d times with nothing to centre it on", built)
	}
}

// TestTheMapIsDrawnInUpdate is W2.1 and W2.2 (D-41, FR-8.4, C-7): the map is
// rendered in Update and its lines stored on the Dashboard; View only prints
// them, calling the library not at all; Work runs as a command and what it
// lands is drawn by the Update its message reaches.
func TestTheMapIsDrawnInUpdate(t *testing.T) {
	d := mapDash(t, Config{})
	d, _ = pressKey(d, "g")
	if len(d.mapPane.lines) == 0 {
		t.Fatal("opening the window drew nothing into the pane")
	}
	calls := &[]string{}
	d.mapPane.calls = calls
	_ = d.View()
	if len(*calls) != 0 {
		t.Errorf("View called the library: %v", *calls)
	}
	before := strings.Join(d.mapPane.lines, "\n")
	d = settleMap(t, d)
	if len(*calls) == 0 {
		t.Fatal("settling called nothing, so this proves nothing")
	}
	for _, c := range *calls {
		if c == "Work" {
			continue
		}
		if !strings.HasPrefix(c, "Render") && c != "Pending" && c != "NextCall" {
			t.Errorf("an unexpected library call: %s", c)
		}
	}
	after := strings.Join(d.mapPane.lines, "\n")
	if after == before {
		t.Error("the tiles work landed were never drawn")
	}
	if !strings.ContainsFunc(stripANSITest(d.View().Content), func(r rune) bool { return r > 0x2800 && r <= 0x28ff }) {
		t.Error("the window prints no braille after the map settled")
	}
	if d.mapPane.changed == 0 {
		t.Error("the pane did not store the counter its lines were drawn at")
	}
}

// TestTheMapIsCentredOnTheSelection is the first half of W1.2 (FR-1.2): the
// window opens on the selected place.
func TestTheMapIsCentredOnTheSelection(t *testing.T) {
	d := mapDash(t, Config{})
	d, _ = pressKey(d, "g")
	at, _ := d.mapPane.m.Centre()
	if absf(at.Lat-33.2) > 0.01 || absf(at.Lon+117.38) > 0.01 {
		t.Errorf("the map opened at %+v, want Oceanside, CA", at)
	}
}

func absf(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// TestTheMapReleasesOnClose is W2.6 in part (FR-8.2): the map is closed with
// the window's owner, and a closed pane builds again on the next open.
func TestTheMapReleasesOnClose(t *testing.T) {
	d := mapDash(t, Config{})
	d, _ = pressKey(d, "g")
	m := d.mapPane.m
	d.closeMap()
	if _, err := m.Render(tuimaps.Size{Cols: 10, Rows: 5}, time.Time{}); err == nil {
		t.Error("the map still renders after the pane was closed")
	}
}

// TestMapTestsReachNoSocket is W0.2's owed check, landing with the first map
// package: no map test opens a listener or dials; they draw from the embedded
// tiles and recorded fixtures only.
func TestMapTestsReachNoSocket(t *testing.T) {
	names, err := filepath.Glob("map_*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("no map test files found, so this proves nothing")
	}
	for _, name := range names {
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range file.Imports {
			path, _ := strconv.Unquote(spec.Path.Value)
			if path == "net" || path == "net/http/httptest" || path == "net/http" {
				t.Errorf("%s imports %s: a map test reaches no socket", name, path)
			}
		}
	}
}
