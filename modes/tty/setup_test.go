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
	if view := stripANSITest(d.View().Content); !strings.Contains(view, "Setup") || !strings.Contains(view, "1. Your default location") {
		t.Fatalf("Setup window missing:\n%s", view)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.(Dashboard).modal == modalSetup {
		t.Fatal("esc closes it without saving")
	}
	first, err := NewDashboard(Config{Version: "t", OpenSetup: true})
	if err != nil || first.modal != modalSetup {
		t.Fatalf("OpenSetup opens the window at launch (%v)", err)
	}
	if view := stripANSITest(first.View().Content); !strings.Contains(view, "1. Your default location") {
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
	for _, want := range []string{"› 1. Your default location", "  2. NASA FIRMS key (optional", "Key: ▌", "[tab] Next question", "[enter] Next"} {
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
	if !strings.Contains(view, "[ctrl+r] Reveal key") {
		t.Fatalf("a chip must never split across lines:\n%s", view)
	}
	model = typeText(model, "oce")
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "› Oceanside, CA (92057)") {
		t.Fatalf("type-ahead hint missing:\n%s", view)
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	view = stripANSITest(model.(Dashboard).View().Content)
	if !strings.Contains(view, "Chosen: Oceanside, CA (92057)") || !strings.Contains(view, "› 2. NASA FIRMS key") || !strings.Contains(view, "[enter] Next") {
		t.Fatalf("enter accepts the pick and focuses question 2 (Next, not Save):\n%s", view)
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // key → the alert preference (Q3, the last)
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "› 3. Alert Notification Preference") || !strings.Contains(view, "[enter] Save") {
		t.Fatalf("focus moves to question 3 (the last — Save):\n%s", view)
	}
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on the last question saves")
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
	fm, _ = fm.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !strings.Contains(stripANSITest(fm.(Dashboard).View().Content), "› 2. NASA FIRMS key") {
		t.Fatal("tab moves to question 2")
	}
	fm, _ = fm.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // key → alert (Q3)
	fm, _ = fm.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // Q3 → Save (no location on a first run)
	if fd := fm.(Dashboard); fd.setup.focus != focusLocation || !strings.Contains(stripANSITest(fd.View().Content), "choose your default location first") {
		t.Fatalf("saving without a location goes back to question 1 with the reason:\n%s", stripANSITest(fd.View().Content))
	}
	fm, _ = fm.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if fm.(Dashboard).setup.focus != focusAlert {
		t.Fatal("shift+tab from question 1 wraps to the last question")
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
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // key → alert (Q3)

	// Default is All; a digit selects Filtered and builds the miles.
	view := stripANSITest(model.(Dashboard).View().Content)
	if !strings.Contains(view, "● All") || !strings.Contains(view, "○ Filtered to [    ]") {
		t.Fatalf("Q3 starts on All, empty distance:\n%s", view)
	}
	model = typeText(model, "50")
	view = stripANSITest(model.(Dashboard).View().Content)
	if !strings.Contains(view, "○ All") || !strings.Contains(view, "● Filtered to [50") {
		t.Fatalf("typing a distance selects Filtered:\n%s", view)
	}
	// Save persists the radius.
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
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // → Q3
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "● Filtered to [25") {
		t.Fatalf("a stored radius opens on Filtered [25]:\n%s", view)
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyUp}) // ↑ picks All
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
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // key → alert preference (Q3)
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})  // Q3 → Save
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
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // oce → key
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // key → alert (Q3)
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})   // Q3 → Save
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
	for _, want := range []string{"Current: Oceanside, CA (92057)", "2. NASA FIRMS key: stored (…cdef) — ✔ working", "empty keeps it"} {
		if !strings.Contains(first, want) {
			t.Fatalf("re-run first screen missing %q:\n%s", want, first)
		}
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // keeps the default, focus to the key
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "Chosen: Oceanside, CA (92057)") || !strings.Contains(view, "› 2. NASA FIRMS key") {
		t.Fatalf("bare enter keeps the default:\n%s", view)
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // key → alert preference (Q3)
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})  // Q3 → Save
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
