package tty

// setup.go — the Settings window's STATE and its KEYS: what is being edited,
// what each keypress does to it, and what is written when it closes.
//
// WHERE THE REST OF THE WINDOW LIVES. It was one 1,210-line file; it is now
// four, along the grain it already read in and following the naming the package
// already used (2026-09-06, a pure move):
//
//	setup.go         this — the window's state, its key handling, its saves
//	setup_layout.go  geometry: groups become blocks, blocks become columns,
//	                 and the body scrolls to keep the focused row on screen
//	setup_form.go    the three questions with no group file of their own —
//	                 default location, the FIRMS key, the alert radius
//
// and one file per question GROUP, as before: setup_rows.go (the row table),
// setup_cast.go, setup_ui.go, setup_relay.go, setup_tones.go.
//
// Originally split from dashboard.go by the quality pass (Q2); the map of where
// things happen is docs/where-things-happen.md.

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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
// The focus is now an INDEX INTO setupTable (setup_rows.go), not a state
// machine over three questions. Twenty rows across four groups cannot be
// enumerated by hand in four places without drifting.
type setupState struct {
	focus setupRowID

	// DATA
	query  string
	hints  []snapshot.LocationRef
	idx    int
	ref    *snapshot.LocationRef // the chosen (or kept) default
	key    string
	reveal bool

	// WATCHPOST RADIO - RELAY REPLAY
	relayDwell time.Duration
	// relayLang is the tie-break language for co-located relays.
	relayLang string // how long Watchlist holds a live relay

	// WATCHPOST UI
	themeIdx int          // the theme the picker is showing — applied live as it moves
	units    render.Units // Imperial / Metric
	clock    render.Clock // 12hr / 24hr / MIL
	uiDirty  bool         // a display preference changed and is not yet written

	// ALERTS - EVENTS
	filtered bool   // false = All locations, true = Within N mi
	radiusMi string // the miles buffer for the [    ] input (digits only)

	// ALERTS - TONE
	toneMode  string          // "" all tones on | "mute"
	toneMuted map[string]bool // class key -> muted

	// WATCHPOST RADIO - CORRESPONDENTS
	cast CastView

	// note is the line under the focused row, and noteRow whose row it belongs
	// to — a note follows its row rather than floating at the bottom, so a
	// listener reads the reason beside the thing it is about.
	note    string
	noteRow setupRowID
	// offered is the voice a preview has already asked about, for FR-4's
	// ask-once flow.
	offered string

	// flash blinks the chip a ←→ press landed on, the way the player's volume
	// chips do (UAT 41). The list wraps, so there is no inert end to mute
	// against and the blink is the only acknowledgement a key did anything.
	flash    pickerFlash
	flashEnd time.Time

	// castDirty marks the cast or tone state changed and not yet written.
	//
	// Those two groups AUTO-SAVE: they are toggles
	// whose effect a listener hears, and needing enter to "lock in" a choice
	// the screen already shows is not intuitive.
	//
	// It is DEBOUNCED rather than immediate, and that is the whole design.
	// Saving a cast re-casts the deck — a hard change, which hands the running
	// broadcast over at the spot reached. Writing on every ←→ press would make
	// the broadcast hand over on every keypress while a listener cycles
	// through voices looking for one: "This is Daniel, taking over for
	// Karen… This is Eddie, taking over for Daniel…". So the write happens
	// when the value SETTLES.
	castDirty bool

	err string

	// gen bumps on EVERY write. The body is built once per change and memoised
	// on it (Task 4.4); a writer that forgets to bump renders one keystroke
	// late, which reads as a broken keyboard.
	gen uint64
}

// touch marks the Setup state changed, so the memoised body is rebuilt. Every
// writer goes through it.
func (st setupState) touch() setupState { st.gen++; return st }

// openSetupAt opens Setup with a row already focused — what V does now that
// the voice chooser is retired (MVS-D-3): the listener presses the key they
// always pressed and lands on the correspondents.
func (d Dashboard) openSetupAt(at setupRowID) Dashboard {
	d = d.openSetup()
	if d.modal == modalSetup {
		d.setup.focus = at
		d = d.settled()
	}
	return d
}

// openSetup toggles the Setup window with fresh state (the alert preference
// seeded from config), alone on top.
func (d Dashboard) openSetup() Dashboard {
	d = d.toggle(modalSetup)
	// Seeded from config, so the window opens showing what is in force — the
	// cast and the tone state are copied so editing them cannot reach the
	// stored config before a save.
	d.setup = setupState{
		cast:      CastView{Mode: d.cfg.Cast.Mode, Names: map[string]string{}},
		toneMode:  d.cfg.Tones.Mode,
		toneMuted: map[string]bool{},
		units:     d.units,
		clock:     d.clockFmt,
	}
	for i, n := range render.ThemeNames() { // the picker opens on the theme in force
		if n == render.ThemeName() {
			d.setup.themeIdx = i
		}
	}
	for k, v := range d.cfg.Cast.Names {
		d.setup.cast.Names[k] = v
	}
	for _, k := range d.cfg.Tones.Muted {
		d.setup.toneMuted[k] = true
	}
	// THE ROTATION'S CURRENT SETTING, or the default when nothing is set. A zero
	// here would show "5m" and mean thirty seconds the moment the picker moved,
	// so it is resolved once on open rather than left to the renderer's
	// fallback.
	d.setup.relayDwell = d.cfg.RelayDwell
	d.setup.relayLang = d.cfg.RelayLang
	if d.setup.relayLang == "" {
		d.setup.relayLang = defaultRelayLang()
	}
	if d.setup.relayDwell <= 0 {
		d.setup.relayDwell = defaultRelayDwell()
	}
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
		// Closing APPLIES what was recorded. esc is not a cancel for the cast
		// and tone groups — the chip says Close rather than Cancel over them —
		// while the typed DATA rows are still enter-to-save and still
		// discarded here.
		//
		// esc and the enter-save are the window's only two exits: while it is
		// open it owns the keyboard, so `s` and `V` type rather than toggle it
		// shut. Both exits write through sequenceWrites, so no group can be
		// saved by one route and dropped by the other.
		apply := d.applyOnCloseCmds()
		d = d.close()
		d.setup = setupState{}
		return d, apply
	case "tab":
		d.setup.focus, d.setup.err = nextGroup(d.setup.focus), ""
		return d.settled(), nil
	case "shift+tab":
		d.setup.focus, d.setup.err = prevGroup(d.setup.focus), ""
		return d.settled(), nil
	}
	// ONE KEYBOARD RULE for the whole window (the batch's constraint): ↑↓ walk
	// a group's rows, space operates the focused control, ←→ cycle a picker,
	// enter accepts and moves on — and saves on the last row.
	//
	// bubbletea v2 names the space key "space"; a `" "` case never fires. That
	// is not a detail: today's setup.go has exactly such a dead case, and it is
	// why the alert radio could not be operated with space before now.
	if key.String() == "ctrl+r" {
		// A WINDOW-level key, not a row-level one. The chip that names it is
		// drawn beside the key field, and a listener reads a chip and presses
		// the key — they do not first check which row has the focus. Scoped to
		// the key row it simply did nothing from anywhere else, which reads as
		// broken. (UAT 2026-08-30.)
		d.setup.reveal = !d.setup.reveal
		return d.settled(), nil
	}
	if m, cmd, handled := d.setupRowKey(key); handled {
		return m, cmd
	}
	// The row's own key handling, with the generation bumped HERE rather than
	// in each handler.
	//
	// The body is memoised on that counter, so a writer that forgets to bump it
	// renders one keystroke late — which is not a stale cache to the person
	// typing, it is a keyboard that does not work. Trusting every writer to
	// remember was the wrong shape: the typed rows did not, and the window
	// stopped showing what was being typed into it. One place cannot forget.
	m, cmd := d.setupRowText(key)
	if next, ok := m.(Dashboard); ok {
		return next.settled(), cmd
	}
	return m, cmd
}

// setupRowText is the focused row's own handling of a key the window-level
// rules did not take.
func (d Dashboard) setupRowText(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch d.setup.focus {
	case rowLocation:
		return d.setupLocationKey(key)
	case rowFIRMSKey:
		return d.setupKeyKey(key)
	case rowEventsAll, rowEventsWithin:
		return d.setupAlertKey(key)
	}
	if key.String() == "enter" {
		return d.setupAdvance()
	}
	return d, nil
}

// setupRowKey applies the ROW-LEVEL half of the one keyboard rule: ↑↓ move,
// space operates, ←→ cycle a picker, p previews. It reports whether it handled
// the key, so the caller can fall through to the row's own typing.
//
// Split out of handleSetupKey to keep both inside the safety gate's decision
// ceiling (P10-04) — and because "which keys move the focus" and "what this
// particular row does with text" are two different questions.
func (d Dashboard) setupRowKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	picker := setupTable()[d.setup.focus].picker
	switch key.String() {
	case "up":
		if !d.rowTakesArrows() {
			d.setup.focus = prevRow(d.setup.focus, d.rowVisible)
			return d.settled(), nil, true
		}
	case "down":
		if !d.rowTakesArrows() {
			d.setup.focus = nextRow(d.setup.focus, d.rowVisible)
			return d.settled(), nil, true
		}
	case "space":
		if setupTable()[d.setup.focus].kind == rowInput {
			return d, nil, false // a text field takes the space as text
		}
		return d.setupSpace(), nil, true
	case "left", "right":
		if setupTable()[d.setup.focus].kind == rowToggle {
			// A two-state control: ←→ and space all do the same thing, because
			// there is nothing to cycle THROUGH — there are two states and
			// either key means "the other one".
			forward := key.String() == "right"
			d = d.toggleClass(d.setup.focus)
			d.setup.flash, d.setup.flashEnd = flashLeft, time.Now().Add(pickerFlashDur)
			if forward {
				d.setup.flash = flashRight
			}
			return d.settled(), nil, true
		}
		if picker {
			forward := key.String() == "right"
			if d.setup.focus == rowTheme {
				d = d.cycleTheme(forward) // auto-preview: the theme is applied as the picker moves
			} else {
				d = d.cyclePicker(d.setup.focus, forward)
			}
			d.setup.flash, d.setup.flashEnd = flashLeft, time.Now().Add(pickerFlashDur)
			if forward {
				d.setup.flash = flashRight
			}
			return d.settled(), nil, true
		}
	case "p":
		if picker {
			m, cmd := d.setupPreview()
			return m, cmd, true
		}
	}
	return d, nil, false
}

// castTouched marks the cast or tone state changed. Every writer of either
// goes through it; the write itself happens when the window closes.
func (d Dashboard) castTouched() Dashboard {
	d.setup.castDirty = true
	return d.settled()
}

// castApplyCmd writes the cast and the tone state — and NOTHING else. It is
// deliberately not setupFinishCmd: the location and the FIRMS key are typed
// values, and a half-typed key must never be committed because a checkbox
// elsewhere in the window moved.
func (d Dashboard) castApplyCmd() tea.Cmd {
	if !d.setup.castDirty {
		return nil
	}
	setCast, cast := d.cfg.SetCast, d.castForSave()
	setTones, tones := d.cfg.SetTones, ToneState{Mode: d.setup.toneMode, Muted: d.setup.mutedClassKeys()}
	if setCast == nil && setTones == nil {
		return nil
	}
	return func() tea.Msg {
		if setCast != nil {
			if err := setCast(cast); err != nil {
				return castSavedMsg{err: err}
			}
		}
		if setTones != nil {
			if err := setTones(tones); err != nil {
				return castSavedMsg{err: err}
			}
		}
		return castSavedMsg{cast: cast, tones: tones}
	}
}

// applyCastSaved records an apply-on-close's outcome.
func (d Dashboard) applyCastSaved(v castSavedMsg) Dashboard {
	if v.err != nil {
		d.setup.err = "could not save: " + v.err.Error()
		return d.settled()
	}
	// What was WRITTEN becomes the config the window opens with, so a re-open
	// shows the file rather than the launch-time cast (UAT bug #2).
	d.cfg.Cast, d.cfg.Tones = v.cast, v.tones
	return d
}

// castSavedMsg is an apply-on-close outcome.
type castSavedMsg struct {
	cast  CastView
	tones ToneState
	err   error
}

// settled bumps the body's generation. EVERY writer of the Setup state calls
// it, or the window renders one keystroke late — which reads as a broken
// keyboard, not as a stale cache.
func (d Dashboard) settled() Dashboard { d.setup = d.setup.touch(); return d }

// rowTakesArrows reports whether the focused row consumes ↑↓ itself. Only the
// location row does, for its hint list; every other row lets ↑↓ move the focus.
func (d Dashboard) rowTakesArrows() bool {
	return d.setup.focus == rowLocation && len(d.setup.hints) > 0
}

// rowVisible reports whether a row can be focused right now. A row is hidden
// only when the group it belongs to cannot act on it — nothing else, so the
// focus order never depends on scroll position or width.
func (d Dashboard) rowVisible(id setupRowID) bool {
	switch id {
	case rowFIRMSKey:
		return true
	}
	return true
}

// setupSpace operates the focused control: select a radio, toggle a checkbox.
func (d Dashboard) setupSpace() Dashboard {
	switch id := d.setup.focus; id {
	case rowEventsAll:
		d.setup.filtered = false
	case rowEventsWithin:
		d.setup.filtered = true
	case rowUnitsImperial:
		return d.setUnits(render.UnitF)
	case rowUnitsMetric:
		return d.setUnits(render.UnitC)
	case rowClock12:
		return d.setClock(render.Clock12)
	case rowClock24:
		return d.setClock(render.Clock24)
	case rowClockMil:
		return d.setClock(render.ClockMil)
	default:
		switch setupTable()[id].kind {
		case rowToggle:
			d = d.toggleClass(id)
		}
	}
	return d.settled()
}

// setupAdvance is enter on a row that does not handle it itself: move to the
// next row, or SAVE on the last.
func (d Dashboard) setupAdvance() (tea.Model, tea.Cmd) {
	if enterSaves(d.setup.focus) {
		return d.setupSave()
	}
	d.setup.focus, d.setup.err = nextRow(d.setup.focus, d.rowVisible), ""
	return d.settled(), nil
}

// setupSave owns the save rules for EVERY group's last row — one owner, so the
// bounce when no location is chosen cannot differ between them.
func (d Dashboard) setupSave() (tea.Model, tea.Cmd) {
	if d.setup.ref == nil {
		if cur := d.currentDefault(); cur != nil {
			d.setup.ref = cur
		} else {
			d.setup.focus, d.setup.err = rowLocation, "choose your default location first"
			return d.settled(), nil
		}
	}
	// The display preferences write on this exit too. setupFinishCmd owns the
	// location, radius, cast and tones; the WATCHPOST UI group is uiApplyCmd's,
	// and leaving it out of this path meant enter saved four groups of five and
	// then discarded the fifth with the window state.
	return d, sequenceWrites(d.setupFinishCmd(strings.TrimSpace(d.setup.key)),
		d.uiApplyCmd(), d.radiusApplyCmd(), d.relayApplyCmd(), d.relayLangApplyCmd())
}

// sequenceWrites orders the window's config writes and drops the no-ops.
//
// SEQUENCED, NEVER BATCHED. Each write loads config.toml, changes its own keys
// and saves the whole file; two of them running concurrently both read the file
// before either writes it, so whichever finishes last silently discards the
// other's keys. tea.Batch makes no ordering promise and runs its commands on
// separate goroutines, which is exactly that race.
// radiusApplyCmd and relayApplyCmd are the SINGLE owners of those two writes.
//
// The window has two exits and they must agree. The esc case says so in as many
// words — "no group can be saved by one route and dropped by the other" — and
// the rotation was saved by enter and dropped by esc anyway, because the write
// was spelled out inside the enter path where esc could not reach it (HUM LEAD,
// UAT 2026-09-04). The alert radius had the same defect and nobody had tried it.
// One owner per setting, called from both exits, is what makes the promise
// checkable; TestBothExitsSaveTheSameSettings is what keeps it true.
//
// Neither writes when the value has not moved: closing a window you only looked
// at should not re-scope the ticker or restart a rotation.
func (d Dashboard) radiusApplyCmd() tea.Cmd {
	set, mi := d.cfg.SetAlertRadius, d.setup.alertRadiusChoice()
	if set == nil || mi == d.cfg.AlertRadiusMi {
		return nil
	}
	return func() tea.Msg { set(mi); return nil }
}

func (d Dashboard) relayApplyCmd() tea.Cmd {
	set, dwell := d.cfg.SetRelayDwell, d.setup.relayDwell
	if set == nil || dwell <= 0 || dwell == d.cfg.RelayDwell {
		return nil
	}
	// The rotation applies at once, not at the next launch: a listener who
	// shortens it is usually shortening it to watch it work.
	return func() tea.Msg { set(dwell); return nil }
}

func (d Dashboard) relayLangApplyCmd() tea.Cmd {
	set, lang := d.cfg.SetRelayLang, d.setup.relayLang
	if set == nil || lang == "" || lang == d.cfg.RelayLang {
		return nil
	}
	return func() tea.Msg { set(lang); return nil }
}

// applyOnCloseCmds is THE list of what closing the window writes. Both exits
// use it, so adding a setting to the window means adding it here once.
func (d Dashboard) applyOnCloseCmds() tea.Cmd {
	return sequenceWrites(d.castApplyCmd(), d.uiApplyCmd(), d.radiusApplyCmd(), d.relayApplyCmd(), d.relayLangApplyCmd())
}

func sequenceWrites(cmds ...tea.Cmd) tea.Cmd {
	live := make([]tea.Cmd, 0, len(cmds))
	for _, cmd := range cmds {
		if cmd != nil {
			live = append(live, cmd)
		}
	}
	if len(live) == 0 {
		return nil
	}
	return tea.Sequence(live...)
}

// setupPreview is `p` on a picker: hear the voice this row will use. The first
// press on an uninstalled voice OFFERS the download; the second proceeds
// (FR-4's "asks once") — a preview must not quietly pull 63 MB.
func (d Dashboard) setupPreview() (tea.Model, tea.Cmd) {
	next, go_ := d.previewOffer(d.setup.focus)
	d = next.settled()
	if !go_ {
		return d, nil
	}
	// THE NOTE BELONGS TO THE ROW THAT ASKED. The deck answers asynchronously,
	// so binding it here rather than on arrival keeps it under the row a
	// listener pressed `p` on even if they have moved on since (F-41).
	d.setup.noteRow = d.setup.focus
	name := d.pickerName(d.setup.focus)
	return d, func() tea.Msg { d.cfg.PreviewVoice(name); return nil }
}

// setupLocationKey is question 1: type → hints; ↑↓ pick; enter takes the
// pick, keeps the current default when nothing was typed, or resolves the
// typed text when nothing matched offline — then moves to question 2.
func (d Dashboard) setupLocationKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		if len(d.setup.hints) > 0 {
			ref := d.setup.hints[min(d.setup.idx, len(d.setup.hints)-1)]
			d.setup.ref, d.setup.focus, d.setup.err = &ref, rowFIRMSKey, ""
			return d, nil
		}
		if q := strings.TrimSpace(d.setup.query); q != "" {
			return d, d.resolveCmd(q, "setup")
		}
		if d.setup.ref != nil { // a location already chosen this visit: keep it, move on (REVIEW C3)
			d.setup.focus, d.setup.err = rowFIRMSKey, ""
			return d, nil
		}
		if cur := d.currentDefault(); cur != nil { // a re-run keeps the default with a bare enter (UAT 111.2)
			d.setup.ref, d.setup.focus, d.setup.err = cur, rowFIRMSKey, ""
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
		return d.setupAdvance() // the key row ends DATA: this saves
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
		return d.setupAdvance()
	case "up", "down":
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
	// The cast and the tones are saved from the ROWS, not from the config the
	// window opened with: castForSave carries only what the rows actually
	// showed as assigned, so an override the listener unticked is cleared
	// rather than quietly kept.
	setCast, cast := d.cfg.SetCast, d.castForSave()
	setTones, tones := d.cfg.SetTones, ToneState{Mode: d.setup.toneMode, Muted: d.setup.mutedClassKeys()}
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
		// The cast is a re-cast: the listener is waiting to hear it. The tones
		// are not — [M] and the checkboxes must not disturb a broadcast in
		// flight. A failure on either is reported the way setup's is.
		if setCast != nil {
			if err := setCast(cast); err != nil {
				return committedMsg{err: err, what: "setup"}
			}
		}
		if setTones != nil {
			if err := setTones(tones); err != nil {
				return committedMsg{err: err, what: "setup"}
			}
		}
		// The outcome carries what was WRITTEN, so the model can seed the next
		// open from it rather than from the launch-time config.
		done := committedMsg{what: "setup", cast: cast, tones: tones, saved: true}
		if commit == nil {
			return done
		}
		if err := commit(watch, recent); err != nil {
			return committedMsg{err: err, what: "setup"}
		}
		return done
	}
}
