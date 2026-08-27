package tty

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

func TestThemeChooserAppliesLiveAndFooterDropsTab(t *testing.T) {
	// UAT 53: [t] floats the chooser; ↓ + enter applies the theme live via
	// the app hook; esc closes. The footer no longer carries [tab].
	defer render.SetTheme(render.DefaultThemeName)
	applied := ""
	m, err := NewDashboard(Config{Version: "t", SetTheme: func(n string) error { applied = n; return nil }})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	if strings.Contains(model.View().Content, "Navigate Sections") {
		t.Fatal("[tab] chip must be gone (UAT 53.1)")
	}
	if !strings.Contains(model.View().Content, "[t] Theme") {
		t.Fatal("[t] Theme lives in the header now (UAT 57)")
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	v := model.View().Content
	if !strings.Contains(v, "Themes apply live") || !strings.Contains(v, "High Contrast") {
		t.Fatalf("theme chooser missing:\n%s", v)
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if applied != render.ThemeNames()[1] {
		t.Fatalf("enter must apply the highlighted theme via the hook, got %q", applied)
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if strings.Contains(model.View().Content, "Themes apply live") {
		t.Fatal("esc must close the chooser")
	}
}

func TestVoiceChooserUAT84(t *testing.T) {
	// [V] opens the correspondent chooser after [v] Viz; ↓ + enter applies
	// through the hook (persisted by the app) and the chip shows the name.
	var chosen, previewed string
	hookCalls := 0
	m, err := NewDashboard(Config{Version: "t", Voice: "Samantha",
		Voices:       func() []string { hookCalls++; return []string{"Samantha", "Alex", "Daniel"} },
		SetVoice:     func(name string) error { chosen = name; return nil },
		PreviewVoice: func(name string) { previewed = name }})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	v := stripANSITest(model.View().Content)
	if i, j := strings.Index(v, "[v] Viz: Off"), strings.Index(v, "[V] Voice: Samantha"); i < 0 || j < 0 || j < i {
		t.Fatalf("[V] Voice: Samantha must follow [v] Viz:\n%s", v)
	}
	m2, _ := model.Update(tea.KeyPressMsg{Code: 'V', Text: "V", Mod: tea.ModShift})
	d := m2.(Dashboard)
	if !d.showVoice {
		t.Fatal("V opens the chooser")
	}
	if mv := stripANSITest(d.View().Content); !strings.Contains(mv, "Correspondent Voice") || !strings.Contains(mv, "✔ Samantha") || !strings.Contains(mv, "Daniel") || !strings.Contains(mv, "[p] Preview") || !strings.Contains(mv, "[enter] Select Voice") || !strings.Contains(mv, "[esc] Cancel") {
		t.Fatalf("chooser lists the voices with the current one marked and the chip controls:\n%s", mv)
	}
	for range 5 {
		_ = d.View() // UAT 85: rendering never calls the hook
	}
	if hookCalls != 1 {
		t.Fatalf("the voice hook must run once per open, got %d", hookCalls)
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if mv := stripANSITest(m3.View().Content); !strings.Contains(mv, "› ") || !strings.Contains(mv, "›   Alex") && !strings.Contains(mv, "›  Alex") {
		t.Fatalf("↓ must move the pointer to Alex:\n%s", mv)
	}
	if _, cmd := m3.Update(tea.KeyPressMsg{Code: 'p', Text: "p"}); cmd == nil { // UAT 86
		t.Fatal("[p] must queue the preview")
	} else {
		runCmd(cmd)
	}
	if previewed != "Alex" {
		t.Fatalf("preview must speak the highlighted voice, got %q", previewed)
	}
	m4, cmd := m3.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter runs the voice hook off the update loop (red-team 0.9.0 C-5)")
	}
	runCmd(cmd)
	d4 := m4.(Dashboard)
	if chosen != "Alex" || d4.radioVoice != "Alex" || d4.showVoice {
		t.Fatalf("enter applies the highlighted voice: chosen=%q chip=%q open=%v", chosen, d4.radioVoice, d4.showVoice)
	}
	if v := stripANSITest(d4.View().Content); !strings.Contains(v, "[V] Voice: Alex") {
		t.Fatalf("chip must show the new voice:\n%s", v)
	}
}

func TestVoiceChooserShowsTheWaitAndItsProgress(t *testing.T) {
	// UAT 119: p says at once what is about to happen; the deck's notes
	// (download progress, loading) replace it; "" clears; opening the
	// chooser starts clean.
	previewed := ""
	m, err := NewDashboard(Config{Version: "t", Voices: func() []string { return []string{"Lessac", "Amy"} }, PreviewVoice: func(n string) { previewed = n }})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.KeyPressMsg{Code: 'V', Text: "V"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model, cmd := model.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "… preparing Amy… (a first use downloads the voice") {
		t.Fatalf("the wait must be explained at once:\n%s", view)
	}
	_ = drain(t, model, cmd)
	if previewed != "Amy" {
		t.Fatalf("preview hook called with %q", previewed)
	}
	model, _ = model.Update(VoiceNoteMsg{Text: "installing Amy voice… 40% (25 MB)"})
	if view := stripANSITest(model.(Dashboard).View().Content); !strings.Contains(view, "… installing Amy voice… 40% (25 MB)") || strings.Contains(view, "preparing Amy") {
		t.Fatalf("the deck's progress replaces the first line:\n%s", view)
	}
	model, _ = model.Update(VoiceNoteMsg{Text: ""})
	if strings.Contains(stripANSITest(model.(Dashboard).View().Content), "… ") {
		t.Fatal("an empty note clears the line")
	}
	model, _ = model.Update(VoiceNoteMsg{Text: "loading Amy…"})
	model, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	model, _ = model.Update(tea.KeyPressMsg{Code: 'V', Text: "V"})
	if strings.Contains(stripANSITest(model.(Dashboard).View().Content), "loading Amy") {
		t.Fatal("reopening the chooser starts clean")
	}
}
