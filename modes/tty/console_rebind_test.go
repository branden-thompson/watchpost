package tty

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/term"
)

// AN OVERRIDE IN THE USER'S KEY TABLE CHANGES THE CONSOLE'S CHORD (FR-1.5).
//
// THE REQUIREMENT'S OWN EXIT SENTENCE, DRIVEN. The gate carried
// `TestTheSwapActionsAreInTheKeyMapAndSoAreRebindable`, which asserts the swap
// actions ARE IN the map — and a map an override can never reach satisfies that
// test perfectly. `broadcasterKeyMap()` went to the Router raw: no `[keys]`
// entry could change a single console binding, so the requirement was unmet for
// the whole of 0.16.0 with a green gate beside it.
//
// `ctrl+b` IS WHY IT MATTERS. It is tmux's default prefix — the survey FR-1.5
// names — so the operator most likely to need the rebind is the one the default
// chord fails for, and rebinding was the documented answer.
func TestAnOverrideInTheKeyTableChangesTheConsolesChord(t *testing.T) {
	// THE OPERATOR MOVES THE BROADCASTER SWAP OFF tmux's PREFIX.
	d, err := NewDashboard(Config{KeyOverrides: term.KeyMap{
		actSwapBroadcaster: {Keys: []string{"ctrl+g", "B"}, Help: "Broadcaster"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	r := NewRouter(d)

	if got := r.keys[actSwapBroadcaster].Keys; len(got) == 0 || got[0] != "ctrl+g" {
		t.Fatalf("the console answers %v; the operator's table says ctrl+g", got)
	}

	// AND THE KEY ITSELF WORKS, which is the half a map assertion cannot reach.
	m, _ := r.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl})
	rr, ok := m.(Router)
	if !ok {
		t.Fatal("the router did not answer the key")
	}
	if rr.active != SurfaceBroadcaster {
		t.Errorf("the rebound chord did not reach the console; the surface is %v", rr.active)
	}

	// AND THE HELP PRINTS WHAT THE OPERATOR ACTUALLY PRESSES. A help view
	// showing the default chord to someone who rebound it is worse than no help:
	// it is the one surface whose whole job is to say which key to press.
	if got := d.helpKeys()[actSwapBroadcaster].Keys; len(got) == 0 || got[0] != "ctrl+g" {
		t.Errorf("the help prints %v; the operator's table says ctrl+g", got)
	}
}

// AND AN OVERRIDE THAT COLLIDES INSIDE THE CONSOLE'S OWN SCOPE IS A BUILD ERROR.
//
// NEVER A SILENT WIN (D-15). The Dashboard's map has been validated since
// 0.13.0; the console's was not validated at all, because it was never merged —
// so a table that bound the bed and the Broadcaster swap to one key would have
// been accepted and one of them would simply have stopped working.
func TestAConsoleOverrideThatCollidesIsRefusedAtBuild(t *testing.T) {
	_, err := NewDashboard(Config{KeyOverrides: term.KeyMap{
		// `b` is already the bed's (D-56, D-78).
		actSwapBroadcaster: {Keys: []string{"b"}, Help: "Broadcaster"},
	}})
	if err == nil {
		t.Fatal("a key bound to two console actions was accepted; D-15 says it is a build error")
	}
}

// AND AN OVERRIDE FOR THE OTHER SURFACE IS DROPPED, NOT REFUSED (FR-14).
//
// TWO SCOPES SHARE ONE OVERRIDE TABLE, so each merge sees entries meant for the
// other. `Merge` drops an action the scope does not have with a note — the same
// tolerance that lets a binding retired in an upgrade cost a nuisance rather
// than a broken launch.
func TestAnOverrideForTheOtherSurfaceDoesNotBreakTheConsole(t *testing.T) {
	d, err := NewDashboard(Config{KeyOverrides: term.KeyMap{
		actSwapBroadcaster: {Keys: []string{"ctrl+g", "B"}, Help: "Broadcaster"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	// OBSERVER'S OWN MAP IS UNDISTURBED by an override aimed at the console.
	if _, claimed := d.keys[actSwapBroadcaster]; claimed {
		t.Error("a console-only action reached Observer's map")
	}
	// AND THE CONSOLE STILL HAS EVERY BINDING IT SHIPPED WITH.
	for act := range broadcasterKeyMap() {
		if _, ok := d.consoleKeyMap()[act]; !ok {
			t.Errorf("the console lost %q to the merge", act)
		}
	}
}
