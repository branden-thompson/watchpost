package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/term"
)

// FR-1.5 / FR-1.6: the swap is rebindable, and THE CHORD IS NOT THE ONLY
// DOOR.
//
// The accessibility lens found the switch reachable only by a modifier chord
// and leaned toward blocking on it. A terminal, a multiplexer or an assistive
// tool can intercept a chord — tmux takes ctrl+b by default — and an operator
// who loses the chord loses the only route back.

func TestTheSwapActionsAreInTheKeyMapAndSoAreRebindable(t *testing.T) {
	km := broadcasterKeyMap()
	for _, a := range []term.Action{actSwapObserver, actSwapBroadcaster, actStationToggle} {
		b, ok := km[a]
		if !ok {
			t.Errorf("%q is not in the key map, so a user cannot rebind it (FR-1.5)", a)
			continue
		}
		if len(b.Keys) == 0 {
			t.Errorf("%q is bound to no key at all", a)
		}
		if strings.TrimSpace(b.Help) == "" {
			t.Errorf("%q has no help text, so it cannot be discovered from the key row", a)
		}
	}
}

func TestTheSwapHasANonChordRoute(t *testing.T) {
	km := broadcasterKeyMap()
	for _, a := range []term.Action{actSwapObserver, actSwapBroadcaster} {
		plain := false
		for _, k := range km[a].Keys {
			if !strings.Contains(k, "ctrl+") && !strings.Contains(k, "alt+") {
				plain = true
			}
		}
		if !plain {
			t.Errorf("%q is reachable ONLY by a modifier chord (FR-1.6). A multiplexer or an "+
				"assistive tool can take the chord, and then the operator has no route at all: keys=%v",
				a, km[a].Keys)
		}
	}
}

// The swap must go THROUGH the gate. A binding that switched surfaces without
// asking canSwap would be a second carrier of the D-1 rule.
func TestPressingTheSwapKeyOnALiveStationDoesNotSwitch(t *testing.T) {
	r := routerAt(lineup.Running, SurfaceBroadcaster)
	r.keys = broadcasterKeyMap()
	m, _ := r.Update(keyPress(t, firstKey(t, r.keys, actSwapObserver)))
	if got := m.(Router).active; got != SurfaceBroadcaster {
		t.Error("the swap key moved off a LIVE station — the binding bypassed canSwap, which makes " +
			"the key a second carrier of the D-1 rule")
	}
}

func TestPressingTheSwapKeyFromStandbySwitches(t *testing.T) {
	// BOTH ROUTES, because FR-1.6's whole point is that the chord is not the
	// only door — testing only the chord would leave the requirement's own
	// subject untested.
	for _, k := range broadcasterKeyMap()[actSwapObserver].Keys {
		r := routerAt(lineup.OffAir, SurfaceBroadcaster)
		r.keys = broadcasterKeyMap()
		m, _ := r.Update(keyPress(t, k))
		if got := m.(Router).active; got != SurfaceObserver {
			t.Errorf("from STANDBY the swap must work via %q; still on surface %d", k, got)
		}
	}
}

// keyPress turns a bound key NAME back into the message the program would
// deliver, so the test drives the REAL key path rather than calling a handler
// directly — the seam the 0.15.0 build log warns about driving past.
func keyPress(t *testing.T, name string) tea.KeyPressMsg {
	t.Helper()
	switch name {
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	}
	if rest, found := strings.CutPrefix(name, "ctrl+"); found {
		r := []rune(rest)
		if len(r) == 1 {
			return tea.KeyPressMsg{Code: r[0], Mod: tea.ModCtrl}
		}
	}
	if rest, found := strings.CutPrefix(name, "shift+"); found && rest == "enter" {
		return tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift}
	}
	r := []rune(name)
	if len(r) == 1 {
		return tea.KeyPressMsg{Code: r[0]}
	}
	t.Fatalf("the fixture cannot express the key %q — extend it rather than driving past the key path", name)
	return tea.KeyPressMsg{}
}

func firstKey(t *testing.T, km term.KeyMap, a term.Action) string {
	t.Helper()
	if len(km[a].Keys) == 0 {
		t.Fatalf("%q has no keys", a)
	}
	return km[a].Keys[0]
}
