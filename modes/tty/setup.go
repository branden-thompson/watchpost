package tty

// setup.go — the Setup window: default location and the FIRMS key. Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// Setup window (UAT 100, HUM LEAD 2026-08-25): "goes to the dashboard,
// immediately opens a setup modal like all the others, asks the questions;
// no → default data set, no key; [s] Setup at any time". One form, both
// questions on screen (UAT 111.3): the default location (type-ahead over
// the embedded index, or a full resolve on enter; a bare enter keeps the
// current default) and the optional NASA FIRMS key (masked; empty keeps a
// stored key, or means the keyless default set). Saving hands the answers to the
// app's Setup hook (persist) and then the watchlist to Commit (the default
// location on top, the rest kept) in ONE command, so the two writes never
// race. esc closes without saving — the dashboard simply has no rows until
// a location is chosen, and [s] reopens the window.
// setupFocus names the question the keys go to (UAT 111.3: every question
// is on screen at once; tab / shift+tab move between them).
type setupFocus int

const (
	focusLocation setupFocus = iota // 1. default location    (group: Data Access)
	focusKey                        // 2. NASA FIRMS key       (group: Data Access)
	focusAlert                      // 3. alert notification   (group: Severe Weather / Disaster Events)
	setupQuestions
)

type setupState struct {
	focus    setupFocus
	query    string
	hints    []snapshot.LocationRef
	idx      int
	ref      *snapshot.LocationRef // the chosen (or kept) default
	key      string
	reveal   bool
	filtered bool   // 3. Alert Notification Preference: false = All, true = Filtered to N mi
	radiusMi string // the miles buffer for the [    ] input (digits only)
	err      string
}

// openSetup toggles the Setup window with fresh state (the alert preference
// seeded from config), alone on top.
func (d Dashboard) openSetup() Dashboard {
	d = d.toggle(modalSetup)
	d.setup = setupState{}
	if d.cfg.AlertRadiusMi > 0 {
		d.setup.filtered = true
		d.setup.radiusMi = fmt.Sprintf("%d", d.cfg.AlertRadiusMi)
	}
	return d
}

// handleSetupKey owns keys while the Setup window is open: printable keys
// build the focused answer, so table/global bindings never fire mid-typing.
// tab / shift+tab move between the questions; enter accepts the focused
// one (and saves on the last); esc closes without saving.
func (d Dashboard) handleSetupKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		d = d.close()
		d.setup = setupState{}
		return d, nil
	case "tab":
		d.setup.focus, d.setup.err = (d.setup.focus+1)%setupQuestions, ""
		return d, nil
	case "shift+tab":
		d.setup.focus, d.setup.err = (d.setup.focus+setupQuestions-1)%setupQuestions, ""
		return d, nil
	}
	switch d.setup.focus {
	case focusLocation:
		return d.setupLocationKey(key)
	case focusKey:
		return d.setupKeyKey(key)
	default:
		return d.setupAlertKey(key)
	}
}

// setupLocationKey is question 1: type → hints; ↑↓ pick; enter takes the
// pick, keeps the current default when nothing was typed, or resolves the
// typed text when nothing matched offline — then moves to question 2.
func (d Dashboard) setupLocationKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		if len(d.setup.hints) > 0 {
			ref := d.setup.hints[min(d.setup.idx, len(d.setup.hints)-1)]
			d.setup.ref, d.setup.focus, d.setup.err = &ref, focusKey, ""
			return d, nil
		}
		if q := strings.TrimSpace(d.setup.query); q != "" {
			return d, d.resolveCmd(q, "setup")
		}
		if d.setup.ref != nil { // a location already chosen this visit: keep it, move on (REVIEW C3)
			d.setup.focus, d.setup.err = focusKey, ""
			return d, nil
		}
		if cur := d.currentDefault(); cur != nil { // a re-run keeps the default with a bare enter (UAT 111.2)
			d.setup.ref, d.setup.focus, d.setup.err = cur, focusKey, ""
			return d, nil
		}
		d.setup.err = "type a city or ZIP first"
	case "up":
		d.setup.idx = max(0, d.setup.idx-1)
	case "down":
		d.setup.idx = min(max(0, len(d.setup.hints)-1), d.setup.idx+1)
	case "backspace":
		if r := []rune(d.setup.query); len(r) > 0 {
			d.setup.query = string(r[:len(r)-1])
		}
		d = d.setupSuggest()
	default:
		if key.Text != "" {
			d.setup.query += key.Text
			d = d.setupSuggest()
		}
	}
	return d, nil
}

// setupKeyKey is question 2: the FIRMS key line — type to paste, ctrl+r
// reveals, enter moves on to question 3 (the alert preference), keeping the
// typed key in state for the final save.
func (d Dashboard) setupKeyKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		d.setup.focus, d.setup.err = focusAlert, ""
		return d, nil
	case "ctrl+r":
		d.setup.reveal = !d.setup.reveal
	case "backspace":
		if r := []rune(d.setup.key); len(r) > 0 {
			d.setup.key = string(r[:len(r)-1])
		}
	default:
		if key.Text != "" {
			d.setup.key += key.Text
		}
	}
	return d, nil
}

// setupAlertKey is question 3: the Alert Notification Preference — ↑↓ picks All
// vs Filtered; a digit selects Filtered and builds the miles; enter saves the
// whole form. An empty or zero radius under "Filtered" reverts to All.
func (d Dashboard) setupAlertKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		if d.setup.ref == nil {
			if cur := d.currentDefault(); cur != nil {
				d.setup.ref = cur
			} else {
				d.setup.focus, d.setup.err = focusLocation, "choose your default location first"
				return d, nil
			}
		}
		return d, d.setupFinishCmd(strings.TrimSpace(d.setup.key))
	case "up", "down", "left", "right", " ":
		d.setup.filtered = !d.setup.filtered
	case "backspace":
		if r := []rune(d.setup.radiusMi); len(r) > 0 {
			d.setup.radiusMi = string(r[:len(r)-1])
		}
	default:
		if r := key.Text; r >= "0" && r <= "9" && len([]rune(d.setup.radiusMi)) < 4 {
			d.setup.filtered = true // typing a distance means Filtered
			d.setup.radiusMi += r
		}
	}
	return d, nil
}

// alertRadiusChoice reads the form's alert preference as the miles to persist:
// 0 when All (or Filtered with an empty/zero/invalid distance).
func (st setupState) alertRadiusChoice() int {
	if !st.filtered {
		return 0
	}
	mi, err := strconv.Atoi(st.radiusMi)
	if err != nil || mi < 0 {
		return 0
	}
	return mi
}

// currentDefault is the watchlist's first location (the stored default),
// nil on a first run.
func (d Dashboard) currentDefault() *snapshot.LocationRef {
	if d.snap == nil || len(d.snap.Locations) == 0 {
		return nil
	}
	ref := refOf(d.snap.Locations[0])
	return &ref
}

// setupSuggest refreshes the hints from the app hook (embedded index only —
// never the network per keystroke).
func (d Dashboard) setupSuggest() Dashboard {
	d.setup.hints, d.setup.idx, d.setup.err = nil, 0, ""
	if q := strings.TrimSpace(d.setup.query); q != "" && d.cfg.Suggest != nil {
		d.setup.hints = d.cfg.Suggest(q, 5)
	}
	return d
}

// setupFinishCmd persists the answers then commits the watchlist with the
// default location on top — one command, two hooks, in order.
func (d Dashboard) setupFinishCmd(key string) tea.Cmd {
	if d.setup.ref == nil {
		return nil
	}
	def, setup, commit := *d.setup.ref, d.cfg.Setup, d.cfg.Commit
	setRadius, radius := d.cfg.SetAlertRadius, d.setup.alertRadiusChoice()
	watch := []snapshot.LocationRef{def}
	for _, r := range refsOf(d.snap) {
		if !sameLocation(r, def) {
			watch = append(watch, r)
		}
	}
	recent := withoutRef(refsOf(d.recent), def)
	return func() tea.Msg {
		if setup == nil {
			return committedMsg{err: fmt.Errorf("setup is not wired in this build"), what: "setup"}
		}
		if err := setup(def, key); err != nil {
			return committedMsg{err: err, what: "setup"}
		}
		if setRadius != nil {
			setRadius(radius) // 0.12.0: persist the alert-notification radius + re-scope the ticker
		}
		if commit == nil {
			return committedMsg{what: "setup"}
		}
		return committedMsg{err: commit(watch, recent), what: "setup"}
	}
}

// setupGroup is a settings-group header: white like the questions, set off by
// blank lines above and below (the section pattern — see docs). More groups
// join as more configurability is added.
func setupGroup(text string) string {
	return "  " + render.Tint(text, render.Tok(render.TextBright))
}

// setupLines is the Setup window body — settings grouped by concern, every
// question on screen, the focused one marked › (UAT 111.3).
func (d Dashboard) setupLines(o render.Opts) []string {
	st := d.setup
	mark := func(f setupFocus) string {
		if st.focus == f {
			return "› "
		}
		return "  "
	}
	lines := []string{"", setupGroup("Data Access"), ""}
	lines = append(lines, d.setupLocationLines(o, mark(focusLocation))...)
	lines = append(lines, d.setupKeyLines(mark(focusKey))...)
	lines = append(lines, "", setupGroup("Severe Weather / Disaster Events"), "")
	lines = append(lines, d.setupAlertLines(o, mark(focusAlert))...)
	action := "Next"
	if st.focus == focusAlert {
		action = "Save"
	}
	// The chip row wraps by chip, inside the inset (UAT 111.4) — the same
	// WrapSegments the radio controls use, never mid-chip.
	segs := []string{o.KeyCap("tab") + " Next question", o.KeyCap("enter") + " " + action, o.KeyCap("↑↓") + " Pick", o.KeyCap("ctrl+r") + " Reveal key", o.KeyCap("esc") + " Cancel"}
	inner := min(o.Width, d.modalWidth()) - 7 - 2 // wrapModal's rail allowance, then the 2-cell inset
	lines = append(lines, "")
	for _, row := range render.WrapSegments(segs, inner, "   ") {
		lines = append(lines, "  "+row)
	}
	return lines
}

// setupLocationLines is question 1 of the form.
func (d Dashboard) setupLocationLines(o render.Opts, mark string) []string {
	st := d.setup
	lines := []string{"  " + mark + render.Tint("1. Your default location (city, \"City, ST\" or ZIP)", render.Tok(render.TextBright))} // questions read white (UAT 111.5)
	switch {
	case st.ref != nil:
		lines = append(lines, "       Chosen: "+render.Plain(st.ref.Label)+" ("+st.ref.Zip+")")
	case strings.TrimSpace(st.query) == "":
		if cur := d.currentDefault(); cur != nil {
			lines = append(lines, "       Current: "+render.Plain(cur.Label)+" ("+cur.Zip+") — "+o.KeyCap("enter")+" keeps it") // UAT 111.2
		}
	}
	if st.ref == nil || st.focus == focusLocation {
		lines = append(lines, "       Search: "+st.query+o.Glyphs().Cursor)
		for i, h := range st.hints {
			pick := "  "
			if i == st.idx {
				pick = "› "
			}
			lines = append(lines, "       "+pick+render.Plain(h.Label)+" ("+h.Zip+")")
		}
	}
	if st.err != "" && st.focus == focusLocation {
		lines = append(lines, "       ⚠ "+st.err)
	}
	return lines
}

// setupKeyLines is question 2 of the form: the FIRMS key, with a stored
// key's tail and health when there is one (UAT 111).
func (d Dashboard) setupKeyLines(mark string) []string {
	st := d.setup
	hint := ""
	if d.cfg.FIRMSKey != nil {
		hint = d.cfg.FIRMSKey()
	}
	var lines []string
	if hint != "" { // UAT 111: a stored key is shown to be there, with how it is doing, and can be replaced
		lines = append(lines, "", "  "+mark+render.Tint("2. NASA FIRMS key: stored (…"+hint+") — ", render.Tok(render.TextBright))+d.firmsHealth(),
			"       Paste a new key to replace it — empty keeps it")
	} else {
		lines = append(lines, "", "  "+mark+render.Tint("2. NASA FIRMS key (optional — satellite fire detection)", render.Tok(render.TextBright)),
			"       Free key: firms.modaps.eosdis.nasa.gov/api/map_key",
			"       Empty = the default data set, no key")
	}
	shown := strings.Repeat("•", len([]rune(st.key)))
	if st.reveal {
		shown = st.key
	}
	lines = append(lines, "       Key: "+shown+d.opts().Glyphs().Cursor)
	if st.err != "" && st.focus == focusKey {
		lines = append(lines, "       ⚠ "+st.err)
	}
	return lines
}

// setupAlertLines is question 3: the Alert Notification Preference — a radio
// pick (All vs Filtered to N mi of the default location). The question reads
// white; the value line reads grey like the other supporting lines, the
// selection carried by the ●/○ marks (glyph, not colour — R-12a).
func (d Dashboard) setupAlertLines(o render.Opts, mark string) []string {
	st := d.setup
	lines := []string{"  " + mark + render.Tint("3. Alert Notification Preference", render.Tok(render.TextBright))}
	buf := st.radiusMi
	if st.focus == focusAlert && st.filtered {
		buf += o.Glyphs().Cursor // the miles cursor, only while editing a Filtered distance
	}
	field := "[" + render.PadTo(buf, 4) + "]"
	value := fmt.Sprintf("%s All   %s Filtered to %s Mi of my location", radioMark(!st.filtered, o.ASCII), radioMark(st.filtered, o.ASCII), field)
	lines = append(lines, "       Current:  "+value)
	if st.err != "" && st.focus == focusAlert {
		lines = append(lines, "       ⚠ "+st.err)
	}
	return lines
}

// radioMark is a radio-button glyph: ● selected / ○ not (or * / o under
// --ascii) — the mark carries the choice without colour (R-12a).
func radioMark(selected, ascii bool) string {
	switch {
	case selected && ascii:
		return "*"
	case selected:
		return "●"
	case ascii:
		return "o"
	default:
		return "○"
	}
}

// firmsHealth words the FIRMS provider's state for the Setup window (UAT
// 111): ✔ working (green), ✘ rejected (red), degraded, off, or not yet
// reported — glyph and colour together (R-12a: the glyph carries it alone).
func (d Dashboard) firmsHealth() string {
	if d.snap == nil {
		return "no report yet"
	}
	for _, w := range d.snap.Warnings {
		if w.Provider == "firms" && strings.Contains(w.Message, "rejected the MAP_KEY") {
			return render.Tint(d.opts().Glyphs().Fail+" rejected "+d.opts().Glyphs().Dash+" replace it", render.Tok(render.ProviderDown))
		}
	}
	for _, p := range d.snap.Providers {
		if p.ID != "firms" {
			continue
		}
		switch p.Status {
		case snapshot.ProviderOK:
			return render.Tint("✔ working", render.Tok(render.ProviderOK))
		case snapshot.ProviderOff:
			return "not active"
		}
		return render.Tint(d.opts().Glyphs().Fail+" degraded (see [S] Status)", render.Tok(render.ProviderDown))
	}
	return "no report yet"
}
