package tty

// modal_chooser.go — the theme and voice choosers ([t], [V]). Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// handleThemeKey owns the theme chooser (UAT 53): ↑↓ move, enter applies
// live (and persists via the app hook), esc closes.
func (d Dashboard) handleThemeKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	names := render.ThemeNames()
	switch key.String() {
	case "esc", "t":
		d = d.close()
	case "up":
		d.themeIdx = max(0, d.themeIdx-1)
	case "down":
		d.themeIdx = min(len(names)-1, d.themeIdx+1)
	case "enter":
		name := names[d.themeIdx]
		if d.cfg.SetTheme == nil {
			render.SetTheme(name) // no persistence hook in this build (tests)
			return d, nil
		}
		if err := d.cfg.SetTheme(name); err != nil {
			d.themeErr = err.Error()
		}
	}
	return d, nil
}

// openTheme toggles the theme chooser with the cursor on the active theme.
func (d Dashboard) openTheme() Dashboard {
	d = d.toggle(modalTheme)
	d.themeErr = ""
	for i, n := range render.ThemeNames() {
		if n == render.ThemeName() {
			d.themeIdx = i
		}
	}
	return d
}

// openVoice toggles the voice chooser with the cursor on the chosen voice.
// The list is read from the hook ONCE here (UAT 85): rendering must never
// run it — on macOS it shells out to `say -v ?`.
func (d Dashboard) openVoice() Dashboard {
	d = d.toggle(modalVoice)
	d.voiceErr, d.voiceNote = "", ""
	if d.modal != modalVoice {
		return d
	}
	d.voiceList = d.voices()
	d.voiceIdx = 0
	for i, n := range d.voiceList {
		if n == d.radioVoice {
			d.voiceIdx = i
		}
	}
	return d
}

// voiceChip is the [V] control's label: the chosen voice, or "—" when none.
func (d Dashboard) voiceChip() string {
	if d.radioVoice == "" {
		return "-" // no voice chosen (an ASCII-safe dash)
	}
	return render.Tint(d.radioVoice, render.Tok(render.RadioStation))
}

// voices lists the correspondent voices the app offers (empty without a hook).
func (d Dashboard) voices() []string {
	if d.cfg.Voices == nil {
		return nil
	}
	return d.cfg.Voices()
}

// handleVoiceKey owns the voice chooser (UAT 84): ↑↓ move, enter applies
// (the app persists and re-tunes), esc closes.
func (d Dashboard) handleVoiceKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	names := d.voiceList
	switch key.String() {
	case "esc", "V":
		d = d.close()
	case "up":
		d.voiceIdx = max(0, d.voiceIdx-1)
	case "down":
		d.voiceIdx = min(len(names)-1, d.voiceIdx+1)
	case "p":
		if len(names) > 0 && d.cfg.PreviewVoice != nil {
			preview, name := d.cfg.PreviewVoice, names[d.voiceIdx]
			d.voiceNote = "preparing " + name + "… (a first use downloads the voice, ~63 MB; loading takes a few seconds)" // UAT 119: said at once, before the deck reports
			return d, func() tea.Msg { preview(name); return nil }
		}
	case "enter":
		if len(names) == 0 {
			return d, nil
		}
		name := names[d.voiceIdx]
		d = d.close()
		d.radioVoice, d.voiceErr = name, ""
		if set := d.cfg.SetVoice; set != nil {
			// Off the update loop (red-team 0.9.0 C-5): the hook saves the
			// config and hands the running broadcast over — disk and a
			// subprocess, never on the key press.
			return d, func() tea.Msg {
				if err := set(name); err != nil {
					return voiceErrMsg{err: err}
				}
				return nil
			}
		}
	}
	return d, nil
}

// voiceErrMsg reports a failed voice change: the chooser reopens with the
// reason so the user can pick again.
type voiceErrMsg struct{ err error }

// voiceLines is the chooser body: one row per voice, the chosen one marked.
func (d Dashboard) voiceLines(o render.Opts) []string {
	lines := []string{""}
	names := d.voiceList
	if len(names) == 0 {
		lines = append(lines, "  No voices found on this system.")
	}
	for i, n := range names {
		ptr, mark := "  ", "  "
		if i == d.voiceIdx {
			ptr = render.Tint("› ", render.Tok(render.FocusPointer))
		}
		if n == d.radioVoice {
			mark = render.Tint("✔ ", render.Tok(render.ProviderOK))
		}
		lines = append(lines, "  "+ptr+mark+n)
	}
	if d.voiceErr != "" {
		lines = append(lines, "", "  ⚠ "+d.voiceErr)
	}
	if d.voiceNote != "" { // UAT 119: the wait explained — download progress, model loading
		lines = append(lines, "", "  … "+render.Tint(d.voiceNote, render.Tok(render.TextBright)))
	}
	lines = append(lines, "", "  Your correspondent for the synthesized broadcast; the choice is saved.")
	return append(lines, "", "  "+o.KeyCap("↑↓")+" Move  "+o.KeyCap("p")+" Preview  "+o.KeyCap("enter")+" Select Voice  "+o.KeyCap("esc")+" Cancel")
}

// themeLines is the chooser body: one row per theme, the active one marked.
func (d Dashboard) themeLines(o render.Opts) []string {
	lines := []string{""}
	for i, n := range render.ThemeNames() {
		ptr, mark := "  ", "  "
		if i == d.themeIdx {
			ptr = render.Tint("› ", render.Tok(render.FocusPointer))
		}
		if n == render.ThemeName() {
			mark = render.Tint("✔ ", render.Tok(render.ProviderOK))
		}
		lines = append(lines, "  "+ptr+mark+n)
	}
	if d.themeErr != "" {
		lines = append(lines, "", "  ⚠ "+d.themeErr)
	}
	lines = append(lines, "", "  Themes apply live; add your own as ~/.config/watchpost/themes/<name>.json")
	return append(lines, "", "  "+o.Controls("  ", render.Ctl("↑↓", "Select"), render.Ctl("enter", "Apply"), render.Ctl("esc", "Close")))
}
