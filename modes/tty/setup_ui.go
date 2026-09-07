package tty

// setup_ui.go — the WATCHPOST UI group.
//
// Three display preferences that had no single home. The theme was a modal of
// its own, reached by [t] and doing nothing but choose; the units were a
// live-only [f]/[c] toggle that nothing remembered between runs; and the clock
// was not a preference at all — every site formatted times its own way.
//
// The theme is a PICKER (←→) and the other two are radio sets, because that is
// what each one is: a theme is one of many and the list wraps, while units and
// clock are small closed sets whose options are worth seeing side by side. Both
// shapes already exist in this window, so neither invents a keyboard.

import (
	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// uiLines draws the group, and returns the line the focused row is DRAWN on.
//
// Recorded as it builds rather than tabulated afterwards. The group's two
// question labels and the theme's support line are not focusable, so the
// offsets are not a row's index — a hand-written table of them was already
// wrong by one for the three clock rows, because it did not count the "Show
// Time in:" label. A layout is not something to write down twice.
func (d Dashboard) uiLines(o render.Opts) ([]string, int) {
	focus, at := d.setup.focus, 0
	lines := []string{
		"  " + setupMark(o, focus == rowTheme) + settingLabel("Theme -", focus == rowTheme) + "  " +
			pickerCell(d.themeName(), newArrowChips(o), d.pickerFlashFor(rowTheme)),
		"    Add yours: ~/.config/watchpost/themes/<name>.json",
		"",
		"  " + settingLabel("Show Units in:", false),
	}
	for _, u := range render.UnitsOrder() {
		id := unitsRow(u)
		if focus == id {
			at = len(lines)
		}
		lines = append(lines, "  "+setupMark(o, focus == id)+
			radioMark(d.setup.units == u, o.ASCII)+" "+settingLabel(u.Label(), focus == id))
	}
	// RADIO CONVENTION, not "Show Time in". Military
	// stopped being only a clock when it took the NATO alphabet with it: it reads
	// callsigns phonetically as well as writing times in four digits. The label
	// names the whole convention because that is what the row now picks.
	lines = append(lines, "", "  "+settingLabel("Radio Convention:", false))
	for _, c := range render.ClockOrder() {
		id := clockRow(c)
		if focus == id {
			at = len(lines)
		}
		lines = append(lines, "  "+setupMark(o, focus == id)+
			radioMark(d.setup.clock == c, o.ASCII)+" "+settingLabel(c.Label(), focus == id))
	}
	return lines, at
}

// unitsRow and clockRow are the row that selects a value, and back again.
func unitsRow(u render.Units) setupRowID {
	if u == render.UnitC {
		return rowUnitsMetric
	}
	return rowUnitsImperial
}

func clockRow(c render.Clock) setupRowID {
	switch c {
	case render.Clock24:
		return rowClock24
	case render.ClockMil:
		return rowClockMil
	}
	return rowClock12
}

// themeName is the theme the picker is showing — the PREVIEWED one while the
// group is open, which is the one on screen.
func (d Dashboard) themeName() string {
	names := render.ThemeNames()
	if len(names) == 0 {
		return "—"
	}
	return names[min(max(d.setup.themeIdx, 0), len(names)-1)]
}

// cycleTheme moves the picker and APPLIES the theme as it goes.
//
// Auto-preview: a theme is a thing you judge by
// looking at it, and a list of names tells a listener nothing about which one
// they want. So ←→ paint the whole app immediately and enter is what commits
// the choice to the file.
//
// Nothing is PERSISTED here. Closing the window writes what is showing, which
// is this window's rule for every group that auto-saves, and it is the right one
// for a theme too: whatever the app is painted in when you leave is what you
// kept. esc does not snap the colours back — the listener would have to press a
// key to find out what they had, which is exactly what auto-preview exists to
// avoid.
func (d Dashboard) cycleTheme(forward bool) Dashboard {
	names := render.ThemeNames()
	if len(names) == 0 {
		return d.settled()
	}
	step := 1
	if !forward {
		step = -1
	}
	d.setup.themeIdx = ((d.setup.themeIdx+step)%len(names) + len(names)) % len(names)
	render.SetTheme(names[d.setup.themeIdx]) // live, and only live
	return d.uiTouched()
}

// setUnits and setClock select one option of a radio set.
func (d Dashboard) setUnits(u render.Units) Dashboard {
	d.setup.units, d.units = u, u // live, as [f]/[c] always were
	return d.uiTouched()
}

func (d Dashboard) setClock(c render.Clock) Dashboard {
	d.setup.clock, d.clockFmt = c, c
	return d.uiTouched()
}

// uiTouched marks a display preference changed and not yet written. Like the
// cast and the tones, the group AUTO-SAVES on close rather than on every press:
// the effect is already on screen, and the file is not what the listener is
// looking at.
func (d Dashboard) uiTouched() Dashboard {
	d.setup.uiDirty = true
	return d.settled()
}

// uiForSave is what the window writes when it closes.
func (d Dashboard) uiForSave() UIPrefs {
	return UIPrefs{Theme: d.themeName(), Units: d.setup.units.Key(), Clock: d.setup.clock.Key()}
}

// uiApplyCmd writes the display preferences — and nothing else, for the same
// reason castApplyCmd does not: a half-typed FIRMS key must never reach the file
// because someone changed the clock.
func (d Dashboard) uiApplyCmd() tea.Cmd {
	if !d.setup.uiDirty || d.cfg.SetUI == nil {
		return nil
	}
	setUI, prefs := d.cfg.SetUI, d.uiForSave()
	return func() tea.Msg {
		if err := setUI(prefs); err != nil {
			return uiSavedMsg{err: err}
		}
		return uiSavedMsg{prefs: prefs}
	}
}

// applyUISaved records the write's outcome. What was WRITTEN becomes what the
// window opens with next time, so a re-open shows the file rather than the
// launch-time preferences (the same bug the cast had at UAT #2).
func (d Dashboard) applyUISaved(v uiSavedMsg) Dashboard {
	if v.err != nil {
		d.setup.err = "could not save: " + v.err.Error()
		return d.settled()
	}
	d.cfg.Units, d.cfg.Clock = v.prefs.Units, v.prefs.Clock
	return d
}

// uiSavedMsg is a display-preference write's outcome.
type uiSavedMsg struct {
	prefs UIPrefs
	err   error
}
