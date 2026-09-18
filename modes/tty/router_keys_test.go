package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/lineup"
	"github.com/branden-thompson/watchpost/platform/term"
)

// FR-1.6: THE CHORD IS NOT THE ONLY DOOR.
//
// The accessibility lens found the switch reachable only by a modifier chord
// and leaned toward blocking on it. A terminal, a multiplexer or an assistive
// tool can intercept a chord — tmux takes ctrl+b by default — and an operator
// who loses the chord loses the only route back.

// TestEveryConsoleSwapActionHasKeysAndHelp is a STRUCTURAL check, and it is
// named for what it measures (D-158).
//
// IT CLAIMED FR-1.5 AND DID NOT MEET IT. Under its old name —
// `TestTheSwapActionsAreInTheKeyMapAndSoAreRebindable` — "is in the key map"
// stood in for "a user can rebind it", and a map no override could reach
// satisfies it perfectly. `broadcasterKeyMap()` went to the Router raw for the
// whole of 0.16.0 with this gate green beside it. The requirement's own exit
// sentence is driven by `TestAnOverrideInTheKeyTableChangesTheConsolesChord`.
//
// IT IS KEPT BECAUSE WHAT IT DOES MEASURE IS REAL: an action with no key, or
// with no help text, cannot be discovered from the key row whatever the
// override table says.
func TestEveryConsoleSwapActionHasKeysAndHelp(t *testing.T) {
	km := broadcasterKeyMap()
	for _, a := range []term.Action{actSwapObserver, actSwapBroadcaster, actStationToggle} {
		b, ok := km[a]
		if !ok {
			t.Errorf("%q is not in the console key map at all", a)
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
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	}
	if rest, found := strings.CutPrefix(name, "ctrl+"); found {
		r := []rune(rest)
		if len(r) == 1 {
			return tea.KeyPressMsg{Code: r[0], Mod: tea.ModCtrl}
		}
	}
	// SHIFT + A NAMED KEY, not just enter (D-111): the bed's relay selector moved
	// to shift+←/shift+→ so the bare arrows are free for the card's PRESENTER.
	if rest, found := strings.CutPrefix(name, "shift+"); found {
		switch rest {
		case "enter":
			return tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift}
		case "left":
			return tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift}
		case "right":
			return tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}
		}
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

// FR-5.4 — THE OPERATOR CHANGES THE STATE WITH A NAMED CONTROL, and the control
// asks the Director rather than flipping a flag: from STANDBY the bound key asks
// for ON AIR, from ON AIR it asks for STANDBY, through the real key path.
func TestTheStationToggleAsksTheDirectorForTheOtherState(t *testing.T) {
	for _, tc := range []struct {
		from lineup.Power
		want lineup.Power
	}{{lineup.OffAir, lineup.Running}, {lineup.Running, lineup.OffAir}} {
		for _, k := range broadcasterKeyMap()[actStationToggle].Keys { // bounded by the bound keys (P10-02)
			st := &station{}
			r := routerAt(tc.from, SurfaceBroadcaster)
			r.keys = broadcasterKeyMap()
			r.station = st
			r.Update(keyPress(t, k))
			if n := len(st.asked); n != 1 || st.asked[0] != tc.want {
				t.Errorf("from %v, %q asked the Director for %v; want exactly one request for %v (FR-5.4)", tc.from, k, st.asked, tc.want)
			}
		}
	}
}

// REVIEW 2026-09-17 (ruling 7-ii) — THE STATION TOGGLE IS REFUSED WHILE A WINDOW
// IS OPEN. shift+enter put the station ON AIR behind an open Request, Help or
// About window: the toggle gated on the surface only. The swap keys already
// hold back while a window is open (D-130); the toggle holds to the same rule.
func TestTheStationToggleIsRefusedWhileAWindowIsOpen(t *testing.T) {
	for _, k := range broadcasterKeyMap()[actStationToggle].Keys { // bounded by the bound keys (P10-02)
		st := &station{}
		r := routerAt(lineup.OffAir, SurfaceBroadcaster)
		r.keys = broadcasterKeyMap()
		r.station = st
		r.observer.modal = modalRequest
		r.Update(keyPress(t, k))
		if len(st.asked) != 0 {
			t.Errorf("%q behind an open window asked the Director for %v; a window open is not the operator's hand on the station", k, st.asked)
		}
	}
}
