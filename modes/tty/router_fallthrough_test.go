package tty

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/term"
)

// THE CONSOLE'S KEYS FALL THROUGH ON OBSERVER, AND THAT IS LOAD-BEARING (D-159).
//
// THIS IS THE CONTRACT THE `keyAction` EXTRACTION COULD HAVE BROKEN. `update`
// was cyclomatic 43 and the switch inside it was lifted out wholesale; several
// of its cases deliberately DO NOT return, and falling past the switch is how
// the key reaches the active surface — which on Observer is the listener's own
// navigation. `r` is their repeat, the arrows walk their table, `b` is theirs.
//
// AN EXTRACTION THAT TURNED ONE FALL-THROUGH INTO A RETURN would be invisible
// to every existing test — the console still works, the key still "does
// something" — and would silently take the listener's navigation away. So the
// `handled` bool is asserted here directly, per action, rather than trusted.
//
// IT DRIVES `keyAction`, NOT THE FRAME, deliberately: the frame has many other
// reasons to look unchanged, and a test that watched it could pass while the
// contract was broken.
func TestTheConsolesKeysFallThroughOnObserver(t *testing.T) {
	d, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range []term.Action{
		// The bed's controls and the queue's arrows: on Observer these walk the
		// listener's table (D-79, D-111).
		actBedCut, actBedPrev, actBedNext,
		actQueuePrev, actQueueNext,
		// `r` is the listener's REPEAT before it is the console's request
		// window (D-135).
		actRequest,
	} {
		r := NewRouter(d)
		r.active = SurfaceObserver
		k := tea.KeyPressMsg{Code: 'x', Text: "x"}
		if _, _, handled := r.keyAction(k, k, a); handled {
			t.Errorf("%q was ANSWERED on Observer; it must fall through to the listener's own surface", a)
		}
	}
}

// AND A LETTER TYPED INTO AN OPEN WINDOW IS NOT A SURFACE SWAP (D-148).
//
// THE OTHER FALL-THROUGH, AND THE ONE WITH A MEASURED DEFECT BEHIND IT. `O` and
// `B` swap surfaces, and the swap cases returned on them UNCONDITIONALLY — above
// the rule that a window on top owns the keys. So typing a place name into a
// location field lost those letters and swapped the surface mid-word:
//
//	"Oceanside" -> "ceanside", and the operator is on Observer
//	"Bonsall"   -> "onsall"
//
// Those are the HUM LEAD's own station and the hyper-local case D-130 exists
// for. The `break` that fixes it is exactly the kind of non-returning path an
// extraction can quietly turn into a return.
func TestALetterTypedIntoAWindowIsNotASwap(t *testing.T) {
	d, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	r := NewRouter(d)
	r.active = SurfaceBroadcaster
	r.observer = r.observer.openRequest()
	if !r.observer.ModalOpen() {
		t.Fatal("the fixture did not open a window")
	}
	for _, tc := range []struct {
		act term.Action
		key tea.KeyPressMsg
	}{
		{actSwapObserver, tea.KeyPressMsg{Code: 'O', Text: "O"}},
		{actSwapBroadcaster, tea.KeyPressMsg{Code: 'B', Text: "B"}},
	} {
		if _, _, handled := r.keyAction(tc.key, tc.key, tc.act); handled {
			t.Errorf("%q swallowed %q from an open field and swapped the surface mid-word",
				tc.act, tc.key.Text)
		}
	}
	// AND THE CONTROL CHORD IS STILL THE WAY OUT. It carries no text, so no
	// field can want it — the escape hatch an operator needs from a window.
	chord := tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl}
	if _, _, handled := r.keyAction(chord, chord, actSwapObserver); !handled {
		t.Error("ctrl+o no longer leaves an open window; the window has become a trap")
	}
}
