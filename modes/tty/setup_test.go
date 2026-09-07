package tty

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestSetupWindowOpensOnSAndAtLaunch(t *testing.T) {
	// UAT 100: [s] Setup at any time; first run / `watchpost setup` opens it
	// at once, over the dashboard like every other modal.
	m := dash(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	d := m.(Dashboard)
	if d.modal != modalSetup {
		t.Fatal("[s] must open the Setup window")
	}
	if view := stripANSITest(d.View().Content); !strings.Contains(view, "Settings") || !strings.Contains(view, "Default location:") {
		t.Fatalf("Settings window missing:\n%s", view)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.(Dashboard).modal == modalSetup {
		t.Fatal("esc closes it without saving")
	}
	first, err := NewDashboard(Config{Version: "t", OpenSetup: true})
	if err != nil || first.modal != modalSetup {
		t.Fatalf("OpenSetup opens the window at launch (%v)", err)
	}
	if view := stripANSITest(first.View().Content); !strings.Contains(view, "Default location:") {
		t.Fatalf("an empty dashboard with the Setup window must render:\n%s", view)
	}
}

func TestSetupFormNoKeyIsTheDefaultDataSet(t *testing.T) {
	// UAT 100 / 111.3: one form, both questions visible; enter on question 1
	// takes the pick and moves the focus to question 2; enter there with an
	// empty key saves — the default data set, no key.
	h := &setupHarness{}
	m, err := NewDashboard(h.config())
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()}) // an existing watchlist stays, below the new default
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	view := stripANSITest(model.(Dashboard).View().Content)
	for _, want := range []string{"› Default location:", "NASA FIRMS key:", "Key: ▌", "[tab] Next question", "[enter] Next"} {
		if !strings.Contains(view, want) {
			t.Fatalf("the form shows every question at once, missing %q:\n%s", want, view)
		}
	}
	// UAT 111.4: the chip row wraps by chip inside the inset — no line
	// runs to the window's edge, no chip is split.
	d0 := model.(Dashboard)
	for _, l := range d0.setupLines(d0.opts()) {
		if w := render.Width(stripANSITest(l)); w > d0.modalWidth()-7 {
			t.Fatalf("setup line runs past the inset (%d): %q", w, stripANSITest(l))
		}
	}
	// The reveal chip appears only when there is a TYPED key to reveal: the
	// stored key is never shown (the app hands this window only its tail).
	if strings.Contains(view, "ctrl+r") {
		t.Fatalf("nothing typed yet: no reveal chip\n%s", view)
	}
	model = typeText(model, "oce")
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "› Oceanside, CA (92057)") {
		t.Fatalf("type-ahead hint missing:\n%s", view)
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	view = stripANSITest(model.(Dashboard).View().Content)
	// The key row is a TEXT FIELD, so its chip reads Next: enter commits the
	// field and moves on. Saving from a field would make it unreachable
	// (UAT #12).
	if !strings.Contains(view, "Default location: Oceanside, CA (92057)") || !strings.Contains(view, "› NASA FIRMS key") || !strings.Contains(view, "[enter] Next") {
		t.Fatalf("enter accepts the pick and focuses the key row:\n%s", view)
	}

	// The key field commits and moves on; the next row is not a field, so
	// enter there saves.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter past the text fields saves")
	}
	model = drain(t, model, cmd)
	d := model.(Dashboard)
	if d.modal == modalSetup || h.setups != 1 || h.key != "" || h.def == nil || h.def.Zip != "92057" {
		t.Fatalf("no key: Setup hook once with the location and an empty key: %+v key=%q open=%v", h.def, h.key, d.modal == modalSetup)
	}
	if len(h.watch) != 1 || h.watch[0].Zip != "92057" || d.selected != 0 { // the fixture's only favourite IS Oceanside: kept once, on top
		t.Fatalf("the default leads the committed watchlist, never duplicated, focus on it: %+v", h.watch)
	}
	// tab moves between the questions both ways; saving without a location
	// on a first run sends the focus back with the reason.
	first, _ := NewDashboard(h.config())
	var fm tea.Model = first
	fm, _ = fm.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	// tab moves between GROUPS now (five stops), so the key row is reached
	// with ↓ within DATA; tab from DATA lands on the events group.
	fm, _ = fm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if !strings.Contains(stripANSITest(fm.(Dashboard).View().Content), "› NASA FIRMS key") {
		t.Fatal("down moves to the key row")
	}
	fm, _ = fm.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // the key field commits and moves on
	fm, _ = fm.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // → Save (no location on a first run)
	if fd := fm.(Dashboard); fd.setup.focus != rowLocation || !strings.Contains(stripANSITest(fd.View().Content), "choose your default location first") {
		t.Fatalf("saving without a location goes back to question 1 with the reason:\n%s", stripANSITest(fd.View().Content))
	}
	fm, _ = fm.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	// shift+tab from the FIRST group wraps to the LAST — derived from the
	// group list, not named, so the next group added moves the target here
	// rather than turning this into a failure about nothing.
	groups := setupGroups()
	last := firstOfGroup(groups[len(groups)-1])
	if got := fm.(Dashboard).setup.focus; got != last {
		t.Fatalf("shift+tab from the first group wraps to the last (%v), got %v", last, got)
	}
}

func TestSetupAlertPreferenceTogglesAndPersists(t *testing.T) {
	h := &setupHarness{}
	m, err := NewDashboard(h.config())
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model = typeText(model, "oce")
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // location → key
	// The events group is its own tab stop, and its two options are two ROWS
	// rather than one line with two marks. TWO tabs: WATCHPOST UI sits between
	// DATA and the events (0.14.0).
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})

	view := stripANSITest(model.(Dashboard).View().Content)
	if !strings.Contains(view, "● All locations (Default)") || !strings.Contains(view, "○ Within [    ] mi") {
		t.Fatalf("the events group starts on All, empty distance:\n%s", view)
	}
	model = typeText(model, "50")
	view = stripANSITest(model.(Dashboard).View().Content)
	if !strings.Contains(view, "○ All locations") || !strings.Contains(view, "● Within [50") {
		t.Fatalf("typing a distance selects Filtered:\n%s", view)
	}
	// Save persists the radius: enter on the group's LAST row.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	drain(t, model, cmd)
	if !h.radiusSet || h.radius != 50 {
		t.Fatalf("the alert radius persists: set=%v radius=%d", h.radiusSet, h.radius)
	}
}

func TestSetupAlertAllPersistsZero(t *testing.T) {
	h := &setupHarness{}
	cfg := h.config()
	cfg.AlertRadiusMi = 25 // a stored radius: the form opens on Filtered [25]
	m, _ := NewDashboard(cfg)
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model = typeText(model, "oce")
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // → WATCHPOST UI
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // → the events group
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "● Within [25") {
		t.Fatalf("a stored radius opens on Filtered [25]:\n%s", view)
	}
	// space selects the focused radio (the one keyboard rule); ↑↓ move.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	drain(t, model, cmd)
	if !h.radiusSet || h.radius != 0 {
		t.Fatalf("switching to All persists 0 (global): set=%v radius=%d", h.radiusSet, h.radius)
	}
}

func TestSetupFormKeyMasksAndStoresIt(t *testing.T) {
	h := &setupHarness{}
	m, err := NewDashboard(h.config())
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model = typeText(model, "oce")
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	key := "0123456789abcdef0123456789abcdef"
	model = typeText(model, key)
	view := stripANSITest(model.(Dashboard).View().Content)
	if strings.Contains(view, key) || !strings.Contains(view, "Key: "+strings.Repeat("•", 32)) {
		t.Fatalf("the key is masked while typed:\n%s", view)
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "Key: "+key) {
		t.Fatalf("ctrl+r reveals it:\n%s", view)
	}
	// Enter on a text field commits it and moves on; enter on the next row —
	// which is not a field — saves. That is 0.13.0's flow, one step shorter.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // key field → the events group
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})  // → Save
	model = drain(t, model, cmd)
	if model.(Dashboard).modal == modalSetup || h.key != key {
		t.Fatalf("enter stores the key: %q", h.key)
	}
	// A refused key (the hook's word) keeps the window open with the reason.
	h2 := &setupHarness{}
	cfg := h2.config()
	cfg.Setup = func(snapshot.LocationRef, string) error { return fmt.Errorf("a FIRMS MAP_KEY is 32 hex characters") }
	m2, _ := NewDashboard(cfg)
	model = m2
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model = typeText(model, "oce")
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // oce → key field
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // key field → the events group
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})   // → Save
	model = drain(t, model, cmd)
	d := model.(Dashboard)
	if d.modal != modalSetup || !strings.Contains(stripANSITest(d.View().Content), "setup failed: a FIRMS MAP_KEY is 32 hex") {
		t.Fatalf("a failed setup stays open with the reason:\n%s", stripANSITest(d.View().Content))
	}
}

func TestSetupFormShowsAStoredFIRMSKeyAndItsHealth(t *testing.T) {
	// UAT 111: on a re-run the form shows the current default and the stored
	// key's tail with how the provider is doing, from the first screen; a
	// bare enter keeps the default, an empty key keeps the stored key.
	h := &setupHarness{}
	cfg := h.config()
	cfg.FIRMSKey = func() string { return "cdef" }
	m, err := NewDashboard(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	s2 := snap()
	s2.Providers = append(s2.Providers, snapshot.ProviderStatus{ID: "firms", Status: snapshot.ProviderOK, FetchedAt: time.Now()})
	model, _ = model.Update(SnapshotMsg{Snap: s2})
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	first := stripANSITest(model.(Dashboard).View().Content)
	for _, want := range []string{"Default location: Oceanside, CA (92057)", "NASA FIRMS key: stored (…cdef) — ✔ working", "empty keeps"} {
		if !strings.Contains(first, want) {
			t.Fatalf("re-run first screen missing %q:\n%s", want, first)
		}
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // keeps the default, focus to the key
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "Default location: Oceanside, CA (92057)") || !strings.Contains(view, "› NASA FIRMS key") {
		t.Fatalf("bare enter keeps the default:\n%s", view)
	}
	// Enter on a text field commits it and moves on; enter on the next row —
	// which is not a field — saves. That is 0.13.0's flow, one step shorter.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // key field → the events group
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})  // → Save
	model = drain(t, model, cmd)
	if h.key != "" || h.setups != 1 || h.def == nil || h.def.Zip != "92057" {
		t.Fatalf("empty key keeps the stored one; the kept default is saved: %q %+v", h.key, h.def)
	}
	// A rejected key says so, in the window, before the user hunts through [S].
	s3 := snap()
	s3.Providers = append(s3.Providers, snapshot.ProviderStatus{ID: "firms", Status: snapshot.ProviderDegraded})
	s3.Warnings = []snapshot.Warning{{Code: "provider_error", Provider: "firms", Message: "firms: FIRMS rejected the MAP_KEY — open Setup ([s]) and paste it again"}}
	model, _ = model.Update(SnapshotMsg{Snap: s3})
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "stored (…cdef) — ✘ rejected — replace it") {
		t.Fatalf("rejected key must be said in the window:\n%s", view)
	}
}

// UAT 2026-08-30 (bug #2): saving a cast, closing Setup and re-opening it
// showed the LAUNCH-TIME cast again.
//
// The file was always right — the window is seeded from cfg, and cfg is
// captured once when the app is built. So the save's outcome now carries what
// it wrote, and the model seeds the next open from that.
func TestReopeningSetupShowsTheCastThatWasSaved(t *testing.T) {
	h := &setupHarness{}
	cfg := h.config()
	var savedCast CastView
	cfg.Voices = func() []string { return []string{"Samantha", "Rishi", "Daniel"} }
	cfg.SetCast = func(v CastView) error { savedCast = v; return nil }
	cfg.SetTones = func(ToneState) error { return nil }
	m, err := NewDashboard(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})

	// V opens at the first correspondent row; → picks a voice for it. There is
	// no mode radio and no enabling checkbox any more.
	model, _ = model.Update(tea.KeyPressMsg{Code: 'V', Text: "V"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	picked := model.(Dashboard).setup.cast.Names[roleAlerts]
	if picked == "" {
		t.Fatal("the fixture must pick a voice, or this proves nothing")
	}

	// Save: enter on the group's last row.
	d := model.(Dashboard)
	d.setup.focus = rowCastSeismic
	_, cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = drain(t, d, cmd)

	if savedCast.Names[roleAlerts] != picked || savedCast.Mode != castModeOn {
		t.Fatalf("the save must carry what the rows showed: %+v", savedCast)
	}
	if model.(Dashboard).modal == modalSetup {
		t.Fatal("a successful save closes the window")
	}
	// RE-OPEN: the rows must show what was saved, not what was there at launch.
	model, _ = model.Update(tea.KeyPressMsg{Code: 'V', Text: "V"})
	got := model.(Dashboard).setup
	if got.cast.Mode != castModeOn || got.cast.Names[roleAlerts] != picked {
		t.Fatalf("re-opening Setup must show the saved cast, got %+v", got.cast)
	}
}

// UAT 2026-08-30 (bug #3): ctrl+r did nothing unless the key row happened to
// have the focus — but the chip that names it is read from anywhere.
func TestCtrlRRevealsFromAnyRow(t *testing.T) {
	h := &setupHarness{}
	m, _ := NewDashboard(h.config())
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // → the key row
	model = typeText(model, "abc123")
	if v := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(v, "Key: •••••") {
		t.Fatalf("the typed key is masked:\n%s", v)
	}
	// From the key row.
	model, _ = model.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if v := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(v, "Key: abc123") {
		t.Fatalf("ctrl+r reveals from the key row:\n%s", v)
	}
	// And from somewhere else entirely.
	model, _ = model.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl}) // hide again
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyTab})            // → the events group
	model, _ = model.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if v := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(v, "Key: abc123") {
		t.Fatalf("ctrl+r is a window key, not a row key:\n%s", v)
	}
}

// UAT 2026-08-30 (bug #4): the picker's press blink stayed lit "for an extended
// period" — it had a tick to START it and none to END it, so it survived until
// something else happened to redraw the window.
func TestThePickerBlinkIsClearedByATick(t *testing.T) {
	h := &setupHarness{}
	cfg := h.config()
	cfg.Voices = func() []string { return []string{"Samantha", "Rishi"} }
	m, _ := NewDashboard(cfg)
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(tea.KeyPressMsg{Code: 'V', Text: "V"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyRight})

	d := model.(Dashboard)
	if d.setup.flash != flashRight {
		t.Fatalf("→ must blink the right chip, got %v", d.setup.flash)
	}
	// The frame must keep ticking while it is lit, or nothing will come back
	// to clear it.
	if !d.tickNeeded() {
		t.Fatal("a lit blink must keep the tick armed, or nothing clears it")
	}
	// After its window, the next tick clears it.
	d.setup.flashEnd = time.Now().Add(-time.Millisecond)
	if got := d.applyTick(); got.setup.flash != flashNone {
		t.Errorf("the tick after expiry clears the blink, got %v", got.setup.flash)
	}
	// ← blinks the other one.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if got := model.(Dashboard).setup.flash; got != flashLeft {
		t.Errorf("← must blink the left chip, got %v", got)
	}
}

// UAT 2026-08-30 (#8): the window RECORDS every cast change as it is made and
// APPLIES them when it CLOSES.
//
// Needing enter to "lock in" a choice the screen already shows is not
// intuitive; applying each change as it happens would be worse, because a save
// re-casts the deck and the broadcast would hand over on every ←→ press while
// a listener cycles through voices. Recording continuously and applying once
// is both.
func TestSetupAppliesTheCastWhenItCloses(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  tea.KeyPressMsg
	}{
		// esc and the enter-save are the only two exits: while the window is
		// open it owns the keyboard, so s and V type rather than toggle it shut.
		{"esc", tea.KeyPressMsg{Code: tea.KeyEsc}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &setupHarness{}
			cfg := h.config()
			var saved CastView
			var saves int
			cfg.Voices = func() []string { return []string{"Samantha", "Rishi"} }
			cfg.SetCast = func(v CastView) error { saved, saves = v, saves+1; return nil }
			cfg.SetTones = func(ToneState) error { return nil }
			m, _ := NewDashboard(cfg)
			var model tea.Model = m
			model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
			model, _ = model.Update(tea.KeyPressMsg{Code: 'V', Text: "V"})
			model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyRight}) // pick
			model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyRight}) // and again

			// Nothing has been written yet: the broadcast must not hand over
			// on every keypress while a voice is being chosen.
			if saves != 0 {
				t.Fatalf("no write before the window closes, got %d", saves)
			}
			picked := model.(Dashboard).setup.cast.Names[roleAlerts]

			_, cmd := model.Update(tc.key)
			drain(t, model, cmd)
			if saves != 1 {
				t.Fatalf("closing applies exactly once, got %d", saves)
			}
			if saved.Mode != castModeOn || saved.Names[roleAlerts] != picked {
				t.Errorf("the applied cast must be what the rows showed: %+v", saved)
			}
		})
	}
}

// And a window nobody changed writes nothing at all.
func TestClosingAnUntouchedSetupWritesNothing(t *testing.T) {
	h := &setupHarness{}
	cfg := h.config()
	saves := 0
	cfg.SetCast = func(CastView) error { saves++; return nil }
	m, _ := NewDashboard(cfg)
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	drain(t, model, cmd)
	if saves != 0 {
		t.Errorf("an untouched window writes nothing, got %d", saves)
	}
}

// THE RELAY REPLAY GROUP: its own heading, one picker, seven choices.
//
// A group rather than a sixth correspondent (HUM LEAD, UAT 2026-09-04): the rows
// above are about who reads, this is about pacing, and the heading leaves
// somewhere for the next pacing setting to land.
func TestTheRelayReplayGroupCyclesTheRotation(t *testing.T) {
	if got := setupGroupTitle(groupRelay); got != "WATCHPOST RADIO - RELAY REPLAY" {
		t.Errorf("the group heading is %q", got)
	}

	// SEVEN CHOICES, shortest first, and the ends are what the support line
	// promises: 30 seconds to 5 minutes.
	list := relayDwells()
	if len(list) != 7 {
		t.Fatalf("want seven rotation choices, got %d", len(list))
	}
	if list[0].d != 30*time.Second || list[0].label != "30s" {
		t.Errorf("the shortest is %v (%q), want 30s", list[0].d, list[0].label)
	}
	if last := list[len(list)-1]; last.d != 5*time.Minute || last.label != "5m" {
		t.Errorf("the longest is %v (%q), want 5m", last.d, last.label)
	}
	for i := 1; i < len(list); i++ {
		if list[i].d <= list[i-1].d {
			t.Errorf("the choices are not shortest-first at %d: %v then %v", i, list[i-1].d, list[i].d)
		}
	}

	// THE PICKER WRAPS AT BOTH ENDS, like every other picker in the window.
	if got := cycleRelayDwell(5*time.Minute, true); got != 30*time.Second {
		t.Errorf("→ from the longest wraps to the shortest, got %v", got)
	}
	if got := cycleRelayDwell(30*time.Second, false); got != 5*time.Minute {
		t.Errorf("← from the shortest wraps to the longest, got %v", got)
	}
	if got := cycleRelayDwell(time.Minute, true); got != 90*time.Second {
		t.Errorf("→ from 1m is 1m 30s, got %v", got)
	}

	// A VALUE THE LIST DOES NOT CARRY reads as the default rather than blank —
	// a config written by hand, or one from a build with different choices.
	if got := relayDwellLabel(7 * time.Minute); got != "5m" {
		t.Errorf("an unknown duration labels as %q, want the default's label", got)
	}
}

// THE ROW DRAWS ITS LABEL AND THE SUPPORT LINE UNDER IT.
func TestTheRelayRowDrawsTheRotationAndItsRange(t *testing.T) {
	d := dash(t).(Dashboard)
	d.setup.relayDwell = 30 * time.Second
	d.setup.focus = rowRelayDwell
	lines := d.relayLines(d.opts())
	if len(lines) < 2 {
		t.Fatalf("want the picker row and its support line, got %d", len(lines))
	}
	row := stripANSITest(lines[0])
	if !strings.Contains(row, "Repeat: Watchlist - Rotate every") || !strings.Contains(row, "30s") {
		t.Errorf("the row reads %q", row)
	}
	if support := stripANSITest(lines[1]); !strings.Contains(support, "Shortest: 30 seconds / Longest: 5 minutes") {
		t.Errorf("the support line reads %q", support)
	}
}

// THE RELAY GROUP IS REACHABLE, and tab lands on it.
//
// Drawing the lines proves they render; it does not prove a listener can get
// there. The window scrolls, so the group sits below the fold at every width —
// which is exactly the state in which "it renders" and "it exists" look the
// same from a golden.
func TestTheRelayGroupIsReachableFromTheWindow(t *testing.T) {
	d := dash(t).(Dashboard)
	d = d.open(modalSetup)

	// tab walks the groups; the relay group is the last of them.
	seen := map[setupGroupID]bool{}
	for i := 0; i < len(setupGroups())*2; i++ { // twice round, so the wrap is covered
		seen[setupTable()[d.setup.focus].group] = true
		d.setup.focus = nextGroup(d.setup.focus)
	}
	if !seen[groupRelay] {
		t.Fatal("tab never reaches the RELAY REPLAY group; the setting is unreachable")
	}

	// And with it focused, the window draws its heading.
	d.setup.focus = rowRelayDwell
	body, _, _ := d.setupBody(d.opts())
	joined := stripANSITest(strings.Join(body, "\n"))
	if !strings.Contains(joined, "WATCHPOST RADIO - RELAY REPLAY") {
		t.Error("the window does not draw the group's heading when its row is focused")
	}
	if !strings.Contains(joined, "Repeat: Watchlist - Rotate every") {
		t.Error("the window does not draw the rotation row when it is focused")
	}
}

// THE CHOSEN ROTATION REACHES THE RADIO.
//
// Drawing the picker, cycling it and reaching the group all pass while the
// value goes nowhere — the window saves the cast, the tones and the radius
// through separate hooks, and a setting simply left out of setupFinishCmd
// looks perfect on screen and changes nothing. That is the shape of the
// Watchlist regression: every part correct except the wire.
func TestTheChosenRotationIsSavedToTheRadio(t *testing.T) {
	h := &setupHarness{}
	m, err := NewDashboard(h.config())
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model = typeText(model, "oce")
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Walk to the relay group by its identity, not by counting tabs: a group
	// added between here and there must not silently retarget this test.
	d := model.(Dashboard)
	for i := 0; i < len(setupGroups())+1; i++ {
		if setupTable()[d.setup.focus].group == groupRelay {
			break
		}
		d.setup.focus = nextGroup(d.setup.focus)
	}
	if setupTable()[d.setup.focus].group != groupRelay {
		t.Fatal("never reached the relay group")
	}
	model = d

	// One press right: five minutes wraps to the shortest.
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if got := model.(Dashboard).setup.relayDwell; got != 30*time.Second {
		t.Fatalf("the picker cycles to 30s, got %v", got)
	}
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	drain(t, model, cmd)

	if !h.dwellSet {
		t.Fatal("saving the window never told the radio the new rotation")
	}
	if h.dwell != 30*time.Second {
		t.Fatalf("the radio was told %v, the listener chose 30s", h.dwell)
	}
}

// BOTH EXITS SAVE THE SAME SETTINGS.
//
// The window's esc case has always claimed this — "no group can be saved by one
// route and dropped by the other" — and it was untrue for two settings at once.
// The rotation was saved by enter and dropped by esc (HUM LEAD, UAT 2026-09-04);
// the alert radius had been the same since 0.12.0 and nobody had pressed esc
// after changing it. A comment is not a guard.
//
// This drives the WINDOW, changing settings by keypress and leaving by each
// door in turn, so a setting whose write is spelled out inside one exit fails
// here rather than in a listener's session.
func TestBothExitsSaveTheSameSettings(t *testing.T) {
	// exercise opens the window, changes the radius and the rotation, and
	// leaves by the given key.
	exercise := func(t *testing.T, leave tea.KeyPressMsg) *setupHarness {
		t.Helper()
		h := &setupHarness{}
		m, err := NewDashboard(h.config())
		if err != nil {
			t.Fatal(err)
		}
		var model tea.Model = m
		model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
		model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
		model = typeText(model, "oce")
		model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // location → key

		// The alert radius: type a distance into the events group.
		d := model.(Dashboard)
		d.setup.focus = firstOfGroup(groupEvents)
		model = typeText(d, "50")

		// The rotation: one press right on the relay row.
		d = model.(Dashboard)
		d.setup.focus = rowRelayDwell
		model, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyRight})

		_, cmd := model.Update(leave)
		drain(t, model, cmd)
		return h
	}

	for _, tc := range []struct {
		door string
		key  tea.KeyPressMsg
	}{
		{"enter", tea.KeyPressMsg{Code: tea.KeyEnter}},
		{"esc", tea.KeyPressMsg{Code: tea.KeyEsc}},
	} {
		t.Run(tc.door, func(t *testing.T) {
			h := exercise(t, tc.key)
			if !h.dwellSet || h.dwell != 30*time.Second {
				t.Errorf("leaving by %s must save the rotation: set=%v dwell=%v", tc.door, h.dwellSet, h.dwell)
			}
			if !h.radiusSet || h.radius != 50 {
				t.Errorf("leaving by %s must save the alert radius: set=%v radius=%d", tc.door, h.radiusSet, h.radius)
			}
		})
	}
}

// THE LANGUAGE CHOICE REACHES THE RADIO, by both doors.
//
// The rotation row was drawn perfectly and inert because nothing pressed a key
// at it; this is the same row shape, so it gets the same test — pressed from
// the window, saved by each exit, checked at the hook.
func TestTheLanguageChoiceIsSavedToTheRadio(t *testing.T) {
	for _, door := range []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{"enter", tea.KeyPressMsg{Code: tea.KeyEnter}},
		{"esc", tea.KeyPressMsg{Code: tea.KeyEsc}},
	} {
		t.Run(door.name, func(t *testing.T) {
			h := &setupHarness{}
			m, err := NewDashboard(h.config())
			if err != nil {
				t.Fatal(err)
			}
			var model tea.Model = m
			model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
			model, _ = model.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
			model = typeText(model, "oce")
			model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

			d := model.(Dashboard)
			if got := d.setup.relayLang; got != "en" {
				t.Fatalf("the window opens on English, got %q", got)
			}
			d.setup.focus = rowRelayLang
			model, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
			if got := model.(Dashboard).setup.relayLang; got != "es" {
				t.Fatalf("→ cycles to Spanish, got %q", got)
			}
			_, cmd := model.Update(door.key)
			drain(t, model, cmd)

			if !h.langSet || h.lang != "es" {
				t.Errorf("leaving by %s must save the language: set=%v lang=%q", door.name, h.langSet, h.lang)
			}
		})
	}
}

// The group draws both rows, with the language row's own note.
func TestTheRelayGroupDrawsBothRows(t *testing.T) {
	d := dash(t).(Dashboard)
	d = d.open(modalSetup)
	d.setup.focus = rowRelayLang
	body, _, _ := d.setupBody(d.opts())
	joined := stripANSITest(strings.Join(body, "\n"))
	for _, want := range []string{
		"WATCHPOST RADIO - RELAY REPLAY",
		"Repeat: Watchlist - Rotate every",
		"Preferred language",
		"English",
		"Used when two relays share a transmitter site.",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("the group must draw %q", want)
		}
	}
}

// TestTheApplicationsDefaultIsSHOWNAndNotUsed — T4.2 / R-4, both halves.
//
// "Bonsall, CA is SHOWN in Settings as the Default, NEVER USED SILENTLY" (HUM
// LEAD 2026-09-01). The two halves fail in opposite directions and only one of
// them is visible: a station that quietly scoped a listener's hazards to
// somebody else's town would look like it was working.
func TestTheApplicationsDefaultIsShownAndNotUsed(t *testing.T) {
	m, err := NewDashboard(Config{Version: "0.1.0-test"})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	d := model.(Dashboard).open(modalSetup)

	// SHOWN, with its zip, and SAID to be the application's rather than theirs.
	got := stripANSITest(d.modalView(d.opts()))
	origin := snapshot.DefaultOrigin()
	if !strings.Contains(got, origin.Label) || !strings.Contains(got, origin.Zip) {
		t.Errorf("the Default location row does not name %s (%s):\n%s", origin.Label, origin.Zip, got)
	}
	if !strings.Contains(got, "Watchpost's default") {
		t.Error("the row shows a location without saying it is the application's, not the listener's — " +
			"which reads as a location they already chose")
	}

	// AND NOT USED. The listener has set nothing, so nothing downstream may have
	// silently adopted it: no watchlist entry, and no location in the snapshot.
	if d.snap != nil && len(d.snap.Locations) > 0 {
		t.Errorf("the application's default reached the snapshot; it is shown, never used")
	}
	if len(refsOf(d.snap)) != 0 {
		t.Errorf("the application's default reached the watchlist as %v; it is shown, never used", refsOf(d.snap))
	}
}
