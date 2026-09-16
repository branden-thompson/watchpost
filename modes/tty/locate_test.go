package tty

// locate_test.go — the debounced location check, driven the way the operator
// drives it.

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// settledLocate drives a field through the REAL path — edit, then the verdict
// for that edit — rather than building the state by hand. A fixture assembled
// field by field would keep compiling after the path stopped working, which is
// the whole reason these windows keep breaking in UAT and not in tests.
func settledLocate(f locateField, query string, ref snapshot.LocationRef, within, found bool) locateState {
	var st locateState
	st, _ = st.edit(f, query)
	// `asked: true` — this helper means THE HOOK ANSWERED. A check that could
	// not be made is its own state (D-151) and is built explicitly where it is
	// being tested, never inherited by every fixture that wanted a verdict.
	return st.apply(locateVerdictMsg{
		field: f, seq: st.gate.Seq(), query: query, ref: ref, within: within, found: found, asked: true,
	})
}

// TYPING DOES NOT ASK. The pause does. This is the rule the HUM LEAD ruled:
// "instead of trying to resolve on *every keystroke* let's set a timer to
// 'wait' for the user pause".
func TestTypingNeverReachesTheResolver(t *testing.T) {
	asked := 0
	d := goldenDash(t, false)
	d.surface, d.addMode = SurfaceBroadcaster, "lookup"
	d.cfg.LocateInRadius = func(string) (snapshot.LocationRef, bool, bool, bool) {
		asked++
		return snapshot.LocationRef{}, false, false, true
	}
	d = d.open(modalAdd)

	var m tea.Model = d
	for _, ch := range "Rainbow, CA" {
		m, _ = m.(Dashboard).handleAddKey(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	if asked != 0 {
		t.Fatalf("typing reached the resolver %d time(s): the debounce is not in the key path", asked)
	}
}

// AND A PAUSE THAT HAS BEEN TYPED OVER ASKS NOTHING EITHER. Eleven keystrokes
// arm eleven pauses; only the last one is still wanted.
func TestOnlyTheLastPauseAsks(t *testing.T) {
	asked := []string{}
	d := goldenDash(t, false)
	d.surface, d.addMode = SurfaceBroadcaster, "lookup"
	d.cfg.LocateInRadius = func(q string) (snapshot.LocationRef, bool, bool, bool) {
		asked = append(asked, q)
		return snapshot.LocationRef{Label: "Rainbow, CA"}, true, true, true
	}
	d = d.open(modalAdd)

	var m tea.Model = d
	var seqs []int
	for _, ch := range "Rainbow" {
		m, _ = m.(Dashboard).handleAddKey(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		seqs = append(seqs, m.(Dashboard).addLocate.gate.Seq())
	}
	// Every pause fires — tea.Tick cannot be cancelled — so deliver them ALL,
	// in order, exactly as the runtime would.
	for _, seq := range seqs {
		var cmd tea.Cmd
		m, cmd = m.(Dashboard).handleLocatePause(locatePauseMsg{field: locateLookup, seq: seq})
		if cmd != nil {
			cmd()
		}
	}
	if len(asked) != 1 {
		t.Fatalf("expected exactly one resolve for seven keystrokes, got %d: %v", len(asked), asked)
	}
	if asked[0] != "Rainbow" {
		t.Errorf("the resolve must carry the FINAL text; it carried %q", asked[0])
	}
}

// AND AN ANSWER THAT ARRIVES AFTER THE NEXT KEYSTROKE IS DROPPED. The resolve
// is already in flight and cannot be called back, so it has to be refused on
// arrival — otherwise the window draws a verdict about text that is gone.
func TestALateAnswerAboutOldTextIsDropped(t *testing.T) {
	d := goldenDash(t, false)
	d.surface, d.addMode = SurfaceBroadcaster, "lookup"
	d.cfg.LocateInRadius = func(string) (snapshot.LocationRef, bool, bool, bool) {
		return snapshot.LocationRef{}, false, false, true
	}
	d = d.open(modalAdd)

	m, _ := d.handleAddKey(tea.KeyPressMsg{Code: 'V', Text: "V"})
	stale := m.(Dashboard).addLocate.gate.Seq()
	m, _ = m.(Dashboard).handleAddKey(tea.KeyPressMsg{Code: 'i', Text: "i"})

	out, _ := m.(Dashboard).handleLocateVerdict(locateVerdictMsg{
		field: locateLookup, seq: stale, query: "V",
		ref: snapshot.LocationRef{Label: "Vista, CA"}, within: true, found: true,
	})
	if out.(Dashboard).addLocate.settled() {
		t.Error("an answer about text the operator has typed over was filed against the new text")
	}
}
