package tty

import (
	"strings"
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

// A CONFIG THAT 0.15.0 ACCEPTED CAN NOW REFUSE TO LAUNCH (F-114).
//
// THIS RELEASE IS THE FIRST TO APPLY `[keys]` TO THE CONSOLE (D-158), and
// several actions live in BOTH scopes — lookup, settings, about, status, help,
// quit, the gain pair, diagnostics. A key free on Observer may already be taken
// on the console, so an override that was legal becomes a hard start-up failure:
// `lookup = "b"` collides with the bed, `settings = "r"` with the request window.
//
// THE COLLISION IS REAL and D-15 says a conflict is never a silent win — but
// `term.Merge`'s own doc argues the other way for exactly this case: "losing a
// binding is a nuisance; refusing to launch over one is a broken upgrade". WHICH
// ONE WINS IS THE HUM LEAD'S CALL, recorded as F-114.
//
// THIS TEST PINS THE BEHAVIOUR AS IT SHIPS so the ruling changes a gate rather
// than a surprise, and asserts the error EXPLAINS ITSELF: the operator did
// nothing, and a message naming only the conflict would read as their mistake.
func TestAnOverrideLegalOnObserverCanRefuseTheConsole(t *testing.T) {
	for _, tc := range []struct {
		act           term.Action
		key, collides string
	}{
		{actLookup, "b", "bed-cut"},
		{actSettings, "r", "request"},
	} {
		_, err := NewDashboard(Config{KeyOverrides: term.KeyMap{
			tc.act: {Keys: []string{tc.key}, Help: "x"},
		}})
		if err == nil {
			t.Errorf("%q=%q was accepted; it collides with %q on the console", tc.act, tc.key, tc.collides)
			continue
		}
		if !strings.Contains(err.Error(), tc.collides) {
			t.Errorf("the error does not name what it collides with: %v", err)
		}
		// IT SAYS WHY THIS IS NEW. Without that the operator reads a config they
		// did not change as a config they got wrong.
		if !strings.Contains(err.Error(), "first to apply") {
			t.Errorf("the error does not explain that this release is the first to apply "+
				"[keys] to the console, so a previously-valid file can now fail: %v", err)
		}
	}
}
