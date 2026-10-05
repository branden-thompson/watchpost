package tty

// map_keys_test.go — 0.18.0 W1.15, W1.16 and the rest of W1.2, as D-61 ruled
// them: while the map window is open it owns the keyboard.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// twoPlaceSnap is the placed snapshot with a second watched location.
func twoPlaceSnap() *snapshot.Snapshot {
	s := placedSnap()
	second := s.Locations[0]
	second.Label, second.Zip, second.Lat, second.Lon = "Flagstaff, AZ", "86001", 35.2, -111.65
	s.Locations = append(s.Locations, second)
	return s
}

func pressCode(d Dashboard, code rune, text string) Dashboard {
	m, _ := d.Update(tea.KeyPressMsg{Code: code, Text: text})
	return m.(Dashboard)
}

// TestTheMapWindowOwnsItsKeys is W1.15 and D-61: with the map open, the arrows
// pan it, + and - zoom it, and none of them reaches what it does outside.
func TestTheMapWindowOwnsItsKeys(t *testing.T) {
	d := mapDash(t, Config{})
	d, _ = pressKey(d, "g")
	at, zoom := d.mapPane.m.Centre()
	selected := d.selected

	d = pressCode(d, tea.KeyRight, "")
	moved, _ := d.mapPane.m.Centre()
	if moved.Lon <= at.Lon {
		t.Errorf("→ did not pan east: %v then %v", at, moved)
	}
	d = pressCode(d, tea.KeyUp, "")
	up, _ := d.mapPane.m.Centre()
	if up.Lat <= moved.Lat {
		t.Errorf("↑ did not pan north: %v then %v", moved, up)
	}
	if d.selected != selected {
		t.Error("↑ moved the selection while the map owns the keys")
	}
	d = pressCode(d, '+', "+")
	if _, z := d.mapPane.m.Centre(); z <= zoom {
		t.Errorf("+ did not zoom in: %v then %v", zoom, z)
	}
	d = pressCode(d, '-', "-")
	d = pressCode(d, '-', "-")
	if _, z := d.mapPane.m.Centre(); z >= zoom {
		t.Errorf("- did not zoom out: %v then %v", zoom, z)
	}
	if d.modal != modalMap {
		t.Fatalf("a map key closed the window: %s", modalName(d.modal))
	}
	d = settleMap(t, d) // tiles the new zoom needs land first, or both frames are the blank one
	before := strings.Join(d.mapPane.lines, "\n")
	d = settleMap(t, pressCode(d, tea.KeyLeft, ""))
	if strings.Join(d.mapPane.lines, "\n") == before {
		t.Error("a pan was not drawn")
	}
}

// TestTheMapFollowsTheSelection is W1.2 (FR-1.2) with D-61's keys: [ and ]
// step to the previous and next location, and the map and its title follow.
func TestTheMapFollowsTheSelection(t *testing.T) {
	d := mapDash(t, Config{})
	m, _ := d.Update(SnapshotMsg{Snap: twoPlaceSnap()})
	d = m.(Dashboard)
	d, _ = pressKey(d, "g")
	d = pressCode(d, ']', "]")
	at, _ := d.mapPane.m.Centre()
	if absf(at.Lat-35.2) > 0.01 || absf(at.Lon+111.65) > 0.01 {
		t.Errorf("] left the map at %+v, want Flagstaff, AZ", at)
	}
	if !strings.Contains(stripANSITest(d.View().Content), "Map · Flagstaff, AZ") {
		t.Error("the title did not follow the selection")
	}
	d = pressCode(d, '[', "[")
	at, _ = d.mapPane.m.Centre()
	if absf(at.Lat-33.2) > 0.01 {
		t.Errorf("[ left the map at %+v, want Oceanside, CA", at)
	}
}

// TestTheMapKeysAreActionsHelpShows is W1.16 (FR-1.11): every map control is a
// keymap action, listed in Help, rebindable in [keys], and no two of the map's
// own keys collide. The legend, playback and scroll actions join with their
// tasks (W1.17, W8.9a, W1.6): a key bound to nothing yet would be a dead key.
func TestTheMapKeysAreActionsHelpShows(t *testing.T) {
	d := mapDash(t, Config{KeyOverrides: term.KeyMap{actMapZoomIn: {Keys: []string{"z"}, Help: "Zoom In"}}})
	seen := map[string]term.Action{}
	for act, bind := range d.mapKeys {
		for _, k := range bind.Keys {
			if other, dup := seen[k]; dup {
				t.Errorf("%q is bound to both %s and %s in the map window", k, other, act)
			}
			seen[k] = act
		}
	}
	for _, act := range mapActions {
		if _, ok := d.mapKeys[act]; !ok {
			t.Errorf("the map's action %s has no binding", act)
		}
	}
	help := strings.Join(d.helpLines(d.opts()), "\n")
	if !strings.Contains(help, "MAP") || !strings.Contains(help, "Pan") || !strings.Contains(help, "Zoom") {
		t.Errorf("Help has no MAP group listing the map's keys:\n%s", help)
	}
	d, _ = pressKey(d, "g")
	_, zoom := d.mapPane.m.Centre()
	d = pressCode(d, 'z', "z")
	if _, z := d.mapPane.m.Centre(); z <= zoom {
		t.Error("the [keys] override did not rebind zoom in")
	}
}
