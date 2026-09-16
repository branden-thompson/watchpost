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

// ONE SURFACE'S REBIND DOES NOT BREAK THE OTHER (F-114).
//
// HUM LEAD, 2026-09-16: collisions are reconciled, and a key binding functions as
// expected. This test was the opposite assertion for one commit — it pinned the
// REFUSAL that D-158 introduced — and is inverted here rather than deleted,
// because the refusal was real and the record should show it was replaced.
//
// THE UPGRADE THAT BROKE. `lookup = "b"` was valid in 0.15.0: `b` was unused on
// Observer. 0.16.0 adds a console that binds `b` to the relay bed, and D-158
// applied `[keys]` to the console for the first time — so the listener's own
// override collided with a binding they had never seen and the app refused to
// start. `term.Merge`'s own doc names that outcome: "refusing to launch over one
// is a broken upgrade".
//
// WHAT HAPPENS NOW: the rebind applies where it fits and is withheld where it
// would collide. Both surfaces do what the operator expects, and the withholding
// is REPORTED — nothing is silently lost.
func TestARebindOnOneSurfaceDoesNotBreakTheOther(t *testing.T) {
	d, err := NewDashboard(Config{KeyOverrides: term.KeyMap{
		actLookup: {Keys: []string{"b"}, Help: "Lookup Location"},
	}})
	if err != nil {
		t.Fatalf("a config that 0.15.0 accepted must still launch: %v", err)
	}
	if got := d.keys[actLookup].Keys; len(got) == 0 || got[0] != "b" {
		t.Errorf("Observer lookup is %v; the operator's table says b", got)
	}
	if got := d.consoleKeyMap()[actLookup].Keys; len(got) == 0 || got[0] != "l" {
		t.Errorf("the console's lookup is %v; it should keep its own key, not take one that collides", got)
	}
	if got := d.consoleKeyMap()[actBedCut].Keys; len(got) == 0 || got[0] != "b" {
		t.Errorf("the console's bed lost its key to another surface's override: %v", got)
	}
	// AND IT IS NOT SILENT. D-15 refuses a conflicting override rather than
	// letting one win quietly; withholding is the reconciliation, and the
	// operator is told which entry did not reach the console.
	if len(d.keysWithheld) != 1 || !strings.Contains(d.keysWithheld[0], "bed-cut") {
		t.Errorf("the withheld override was not reported: %v", d.keysWithheld)
	}
}

// AND A CONFLICT THE OPERATOR MADE INSIDE ONE SCOPE IS STILL A BUILD ERROR.
//
// D-15: never a silent win. Withholding is for an override aimed at the OTHER
// surface; a console-only action rebound onto another console key has nowhere
// else to apply, so there is nothing to reconcile and the file is wrong.
func TestAConsoleOnlyConflictIsStillRefused(t *testing.T) {
	_, err := NewDashboard(Config{KeyOverrides: term.KeyMap{
		// `bed-cut` exists only on the console, and `r` is the request window's.
		actBedCut: {Keys: []string{"r"}, Help: "Bed"},
	}})
	if err == nil {
		t.Fatal("a console-only action rebound onto another console key was accepted; D-15 says that is a build error")
	}
}

// AND AN OVERRIDE THAT RESOLVES ITS OWN COLLISION IS HONOURED.
//
// If the operator moves the bed off `b` and lookup onto it in the same file,
// both apply: the collision they would have caused is one they also resolved,
// and refusing it would be the tool arguing with a decision already made.
func TestOverridesThatVacateAKeyAreBothApplied(t *testing.T) {
	d, err := NewDashboard(Config{KeyOverrides: term.KeyMap{
		actBedCut: {Keys: []string{"k"}, Help: "Bed"},
		actLookup: {Keys: []string{"b"}, Help: "Lookup Location"},
	}})
	if err != nil {
		t.Fatalf("an operator who resolved their own collision was refused: %v", err)
	}
	if got := d.consoleKeyMap()[actBedCut].Keys; len(got) == 0 || got[0] != "k" {
		t.Errorf("the bed did not move to k: %v", got)
	}
	if got := d.consoleKeyMap()[actLookup].Keys; len(got) == 0 || got[0] != "b" {
		t.Errorf("lookup did not take the key the bed vacated: %v", got)
	}
	if len(d.keysWithheld) != 0 {
		t.Errorf("nothing should have been withheld: %v", d.keysWithheld)
	}
}

// A ROTATION OF SHARED KEYS DOES NOT REFUSE THE CONSOLE.
//
// `about = "l"` and `lookup = "b"` are both legal on Observer and neither names
// a console binding the operator has ever seen. Deciding in one pass grants
// `about` the `l` that `lookup` was about to vacate, then withholds `lookup` —
// which returns it to `l` — and the console holds `l` twice, so the app refuses
// to start with a collision the tool created.
//
// THE FILE BINDS NOTHING TWICE. Any message blaming the operator here is wrong.
func TestARotationOfSharedKeysStillLaunches(t *testing.T) {
	d, err := NewDashboard(Config{KeyOverrides: term.KeyMap{
		actAbout:  {Keys: []string{"l"}, Help: "About"},
		actLookup: {Keys: []string{"b"}, Help: "Lookup Location"},
	}})
	if err != nil {
		t.Fatalf("a rotation of two shared keys must not refuse the console: %v", err)
	}
	// AND THE CONSOLE'S MAP HOLDS NO KEY TWICE.
	seen := map[string]term.Action{}
	for act, b := range d.consoleKeyMap() {
		for _, k := range b.Keys {
			if held, dup := seen[k]; dup {
				t.Errorf("the console binds %q to both %q and %q", k, held, act)
			}
			seen[k] = act
		}
	}
	// AND THE BED KEEPS ITS KEY, since `lookup -> b` is the override that gave way.
	if got := d.consoleKeyMap()[actBedCut].Keys; len(got) == 0 || got[0] != "b" {
		t.Errorf("the bed lost its key: %v", got)
	}
}
