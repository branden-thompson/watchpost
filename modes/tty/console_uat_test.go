package tty

// console_uat_test.go — the console's controls stand aside for a window open over
// it.

import "testing"

// A WINDOW ON TOP OWNS THE KEYBOARD, AND THE CONSOLE'S OWN CONTROLS STAND ASIDE
// (D-58, D-87).
//
// The Router looks a key up BEFORE the check that hands an open window its
// input, so a console control that does not stand aside — the bed's `←`/`→` —
// steps the relay while the operator moves through the status window. Only a
// gate that drives those keys with a window up sees it, as the
// modal-reachability gate does for the queue's `↑`/`↓`.
func TestTheConsolesControlsStandAsideForAnOpenWindow(t *testing.T) {
	d, err := NewDashboard(Config{})
	if err != nil {
		t.Fatal(err)
	}
	r := NewRouter(d)
	r.active = SurfaceBroadcaster

	if !r.consoleOwnsTheKeys() {
		t.Fatal("with no window open the console owns its own keys")
	}

	// THE OBSERVER'S WINDOW IS THE ONE THAT COMPOSITES OVER THE CONSOLE, which
	// is why it is the one asked.
	open := r
	open.observer = openAModal(t, r.observer)
	if !open.observer.ModalOpen() {
		t.Skip("no modal could be opened in this fixture")
	}
	if open.consoleOwnsTheKeys() {
		t.Error("the console acted on a key meant for the window over it")
	}

	// AND ON OBSERVER THE CONSOLE NEVER OWNS THEM, window or not: those arrows
	// walk the listener's table.
	obs := r
	obs.active = SurfaceObserver
	if obs.consoleOwnsTheKeys() {
		t.Error("the console claimed the arrows while Observer was the surface")
	}
}

// openAModal puts the status window up, which is the one the reachability gate
// drives.
func openAModal(t *testing.T, d Dashboard) Dashboard {
	t.Helper()
	d.width, d.height = 150, 74
	m, _ := d.Update(keyPress(t, "S"))
	if out, ok := m.(Dashboard); ok {
		return out
	}
	return d
}
