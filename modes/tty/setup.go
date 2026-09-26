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

	// THE STATION'S OWN TWO (D-115). `query`, `hints` and `idx` above are SHARED
	// with this row's type-ahead and that is safe by construction: the listener's
	// default location is `scopeObserver` and the transmitter is
	// `scopeBroadcaster`, so the two are never on screen together and one
	// type-ahead is only ever filling one of them.
	//
	// THE RESOLVED REF IS NOT SHARED, though, and that is the half that matters:
	// they are different settings with different owners (D-72), and a save that
	// could write the station's epicentre into the listener's default location
	// would be the leak the whole scope table exists to prevent.
	txRef *snapshot.LocationRef

	// serviceMi is the miles buffer for the service radius, and serviceSeeded
	// says it still holds the STORED value — the same first-digit-replaces rule
	// the alert radius learned at UAT 2026-09-08, for the same reason.
	serviceMi     string
	serviceSeeded bool

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
	// radiusSeeded is true while radiusMi still holds the STORED value and the
	// listener has typed nothing. The first digit then REPLACES it instead of
	// appending (HUM LEAD, UAT 2026-09-08).
	//
	// Without this, opening a window that reads "[50] mi" and typing 20 — the
	// obvious way to change it — produced 5020, a five-thousand-mile radius,
	// and the listener reasonably read the result as "my choice was not saved".
	// It was saved; it was just not the number they entered.
	radiusSeeded bool

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

	// layerAt is the layers row's cursor: which layer space switches (0.18.0).
	layerAt int

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
	d = d.toggle(modalSetup).refreshMapCost() // the Maps tab's warning reads the estimate
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
	// THE WINDOW OPENS ON A ROW THIS SURFACE ACTUALLY DRAWS (D-92). `setupState{}`
	// focuses row zero, which is the default LOCATION — the listener's, and hidden
	// on the console. Opening focused on a row nobody can see is a window whose
	// first keystroke appears to do nothing.
	if !d.rowVisible(d.setup.focus) {
		d.setup.focus = nextRow(d.setup.focus, d.rowVisible)
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
		d.setup.radiusSeeded = true // the first digit typed replaces it
	}
	// THE STATION'S SERVICE RADIUS, SHOWN AS IT IS IN FORCE (D-115). Seeded the
	// same way and for the same reason: the first digit typed REPLACES it.
	if d.cfg.ServiceRadiusMi > 0 {
		d.setup.serviceMi = fmt.Sprintf("%d", d.cfg.ServiceRadiusMi)
		d.setup.serviceSeeded = true
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
		d = d.commitToModel() // before close(): it reads d.setup, which the reset below clears
		d = d.close()
		d.setup = setupState{}
		return d, apply
	case "tab": // D-62: tab and shift+tab switch tabs from any row
		d.setup.focus, d.setup.err = d.stepTab(1), ""
		return d.settled(), nil
	case "shift+tab":
		d.setup.focus, d.setup.err = d.stepTab(-1), ""
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
	case rowLocation, rowTransmitter:
		// ONE TYPE-AHEAD, TWO SETTINGS (D-115). They are never both on screen —
		// one is `scopeObserver` and the other `scopeBroadcaster` — so the query,
		// the hints and the index are shared and only the resolved ref is not.
		return d.setupLocationKey(key)
	case rowServiceRadius:
		return d.setupServiceKey(key)
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
	if d.setup.focus == rowMapClear && (key.String() == "space" || key.String() == "enter") {
		m, cmd := d.clearMapData()
		return m, cmd, true
	}
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
		if !d.rowTakesLeftRight() { // D-62: the arrows switch tabs, unless the focused row operates with them
			step := 1
			if key.String() == "left" {
				step = -1
			}
			d.setup.focus, d.setup.err = d.stepTab(step), ""
			return d.settled(), nil, true
		}
		if d.setup.focus == rowMapsOn {
			d.mapsOff = !d.mapsOff // 0.18.0: two states, so either arrow is "the other one"
			return d.uiTouched(), nil, true
		}
		if d.setup.focus == rowMapDesc {
			return d.cycleMapDesc(key.String() == "right"), nil, true
		}
		if m, ok := d.mapPrefArrow(key.String() == "right"); ok {
			return m, nil, true
		}
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
		if picker && !mapPickerRow(d.setup.focus) { // the map's pickers have nothing to preview
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
	if id < 0 || id >= setupRowCount {
		return false // outside the table: not a row, so not a visible one
	}
	// D-18's RULING, ASKED HERE (D-92). `stepRow` already walks only visible
	// rows and `setupBlock` already draws only visible rows — this seam was built
	// for exactly this and returned `true` for everything until now.
	return d.onFocusedTab(id) // D-62: a tab draws and walks its own rows; D-70: on every surface
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
	case rowMapsOn:
		d.mapsOff = !d.mapsOff // live: g says so at once (W1.8)
		return d.uiTouched()
	case rowMapDesc:
		return d.cycleMapDesc(true)
	case rowMapScale:
		return d.cycleMapScale(true)
	case rowMapNearby:
		return d.cycleNearby(true)
	case rowMapRadarSource:
		return d.toggleRadarSource()
	case rowMapDetailLevel:
		return d.cycleDetailLevel(true).uiTouched()
	case rowMapLayers:
		return d.toggleLayer()
	case rowMapDetailBorders, rowMapDetailWater, rowMapDetailRivers, rowMapDetailNames,
		rowMapDetailRoads, rowMapDetailRail, rowMapDetailParks:
		nd, _ := d.toggleDetailRow(id)
		return nd
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
	cmd := sequenceWrites(d.setupFinishCmd(strings.TrimSpace(d.setup.key)),
		d.uiApplyCmd(), d.radiusApplyCmd(), d.relayApplyCmd(), d.relayLangApplyCmd(),
		d.transmitterApplyCmd(), d.serviceRadiusApplyCmd())
	return d.commitToModel(), cmd
}

// commitToModel makes the MODEL agree with what closing the window just wrote.
//
// THE WRITE WAS NEVER THE PROBLEM. These three settings persist through a setter
// that returns nothing, so nothing wrote the new value back into d.cfg — and
// openSetup seeds the form FROM d.cfg, so re-opening showed the old choice and
// the listener reasonably concluded the save had failed. It had not: HUM LEAD,
// UAT 2026-09-08, saw the [w] window correctly trim its events to the new radius
// while Settings still displayed the previous one. The config file, the ticker
// pipeline and the severe window all had the new value; only the form did not.
//
// The display preferences never had this bug because uiApplyCmd returns a
// uiSavedMsg and applyUISaved writes the values back (setup_ui.go) — this is
// that same round trip, for the three settings whose setters cannot report an
// outcome to return.
//
// CALL IT AFTER THE CMDS ARE BUILT. applyIfChanged compares the new value with
// d.cfg, so updating d.cfg first would make every write look like a no-op and
// nothing would be saved at all.
//
// The guards match applyIfChanged's exactly. If they drift, the model and the
// file disagree about what is in force, which is a worse bug than this one.
func (d Dashboard) commitToModel() Dashboard {
	if d.cfg.SetAlertRadius != nil {
		d.cfg.AlertRadiusMi = d.setup.alertRadiusChoice()
	}
	if d.cfg.SetRelayDwell != nil && d.setup.relayDwell > 0 {
		d.cfg.RelayDwell = d.setup.relayDwell
	}
	// THE STATION'S TWO, ON THE SAME ROUND TRIP AND FOR THE SAME REASON: their
	// setters return nothing, so without this the window would re-open showing
	// the OLD epicentre and the operator would reasonably read the save as failed.
	// The guards match the ApplyCmds' exactly; if they drift, the model and the
	// file disagree about what is in force.
	if d.cfg.SetTransmitter != nil && d.setup.txRef != nil {
		ref := *d.setup.txRef
		d.cfg.Transmitter = &ref
	}
	if v := d.setup.serviceRadiusChoice(); d.cfg.SetServiceRadius != nil && d.inServiceRange(v) {
		d.cfg.ServiceRadiusMi = v
	}
	if d.cfg.SetRelayLang != nil && d.setup.relayLang != "" {
		d.cfg.RelayLang = d.setup.relayLang
	}
	return d
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
	// No validity predicate: 0 is "All (global)", a real choice (0.12.0).
	return applyIfChanged(d.cfg.SetAlertRadius, d.setup.alertRadiusChoice(), d.cfg.AlertRadiusMi, nil)
}

// transmitterApplyCmd writes the station's epicentre (D-115).
//
// IT WRITES ONLY WHAT THE OPERATOR CHOSE. `txRef` is nil until they pick a
// place, and a nil there means "unchanged" — NOT "clear it". A save that wrote
// the borrowed fallback back as the station's OWN transmitter would silently end
// the D-72 split: the station would stop following the listener's default
// location the first time anyone opened Settings and pressed enter.
func (d Dashboard) transmitterApplyCmd() tea.Cmd {
	set := d.cfg.SetTransmitter
	if set == nil || d.setup.txRef == nil {
		return nil
	}
	next := *d.setup.txRef
	if cur := d.cfg.Transmitter; cur != nil && snapshot.Key(*cur) == snapshot.Key(next) {
		return nil // it has not moved
	}
	return func() tea.Msg { set(next); return nil }
}

// serviceRadiusApplyCmd writes how far the station serves (D-115).
//
// THE BOUNDS ARE ENFORCED HERE, at the save, rather than at the keystroke: a
// field that refused "1" on the way to "100" would be fighting the operator over
// a number they had not finished writing. Out of range writes NOTHING and the
// stored value stands, which is the same "do nothing for an invalid value" rule
// `applyIfChanged` states.
func (d Dashboard) serviceRadiusApplyCmd() tea.Cmd {
	return applyIfChanged(d.cfg.SetServiceRadius, d.setup.serviceRadiusChoice(), d.cfg.ServiceRadiusMi,
		d.inServiceRange)
}

// serviceRadiusChoice is the buffer as a number, and zero when it is not one.
func (st setupState) serviceRadiusChoice() int {
	n, err := strconv.Atoi(strings.TrimSpace(st.serviceMi))
	if err != nil {
		return 0
	}
	return n
}

// applyIfChanged is the write-on-close shape THREE settings share: do nothing
// without a setter, do nothing for an invalid value, do nothing when the value
// has not moved — otherwise write it (metric D, 2026-09-08).
//
// IT COVERS THREE OF THE FIVE *ApplyCmd, NOT ALL FIVE, and that is deliberate.
// castApplyCmd and uiApplyCmd return a MESSAGE and handle a save error; forcing
// them through this would mean a second return path and a nil-able error, which
// is more shape than the duplication costs.
func applyIfChanged[T comparable](set func(T), next, cur T, valid func(T) bool) tea.Cmd {
	if set == nil || next == cur || (valid != nil && !valid(next)) {
		return nil
	}
	return func() tea.Msg { set(next); return nil }
}

// The rotation applies at once, not at the next launch: a listener who shortens
// it is usually shortening it to watch it work.
func (d Dashboard) relayApplyCmd() tea.Cmd {
	return applyIfChanged(d.cfg.SetRelayDwell, d.setup.relayDwell, d.cfg.RelayDwell,
		func(v time.Duration) bool { return v > 0 })
}

func (d Dashboard) relayLangApplyCmd() tea.Cmd {
	return applyIfChanged(d.cfg.SetRelayLang, d.setup.relayLang, d.cfg.RelayLang,
		func(v string) bool { return v != "" })
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
	// WHICH SETTING THIS TYPE-AHEAD IS FILLING (D-115). The station's transmitter
	// and the listener's default location are the same CONTROL and different
	// FACTS (D-72), so the keys are shared and the destination is not.
	into, cur, next := &d.setup.ref, d.currentDefault(), rowFIRMSKey
	if d.setup.focus == rowTransmitter {
		into, cur, next = &d.setup.txRef, d.currentTransmitter(), rowServiceRadius
	}
	switch key.String() {
	case "enter":
		if len(d.setup.hints) > 0 {
			ref := d.setup.hints[min(d.setup.idx, len(d.setup.hints)-1)]
			*into, d.setup.focus, d.setup.err = &ref, next, ""
			// AND THE QUERY IS SPENT. It is shared with the other row, so leaving
			// it behind would show one setting's search under the other's label
			// the next time the window opened on the other surface.
			d.setup.query, d.setup.hints, d.setup.idx = "", nil, 0
			return d, nil
		}
		if q := strings.TrimSpace(d.setup.query); q != "" {
			return d, d.resolveCmd(q, "setup")
		}
		if *into != nil { // a location already chosen this visit: keep it, move on (REVIEW C3)
			d.setup.focus, d.setup.err = next, ""
			return d, nil
		}
		if cur != nil { // a re-run keeps the default with a bare enter (UAT 111.2)
			*into, d.setup.focus, d.setup.err = cur, next, ""
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

// setupServiceKey is the station's service radius: digits only, no radio.
//
// HUM LEAD, 2026-09-13: it functions "like the Service alerts radius filer
// option in Settings just without the 'all alerts' option (so no radio button)".
//
// THE SAME FIRST-DIGIT-REPLACES RULE the alert radius learned at UAT 2026-09-08,
// and for the same reason: a field showing a number the operator did not type is
// a field they are about to type OVER, not one they are appending to. Without it
// a stored 25 and a typed 50 make 2550.
//
// THE BOUNDS ARE CHECKED AT THE SAVE, NOT AT THE KEYSTROKE. Typing "1" on the way
// to "100" would otherwise be refused as below the floor, which is a field that
// fights the operator over a number they have not finished writing.
func (d Dashboard) setupServiceKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		return d.setupAdvance()
	case "backspace":
		d.setup.serviceSeeded = false
		if r := []rune(d.setup.serviceMi); len(r) > 0 {
			d.setup.serviceMi = string(r[:len(r)-1])
		}
	default:
		if r := key.Text; r >= "0" && r <= "9" {
			if d.setup.serviceSeeded {
				d.setup.serviceMi, d.setup.serviceSeeded = "", false
			}
			if len([]rune(d.setup.serviceMi)) < 3 { // 100 is the ceiling; three digits hold it
				d.setup.serviceMi += r
			}
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
		// Editing is taking the field over as surely as typing is: after a
		// backspace the buffer is the listener's, so the next digit appends.
		d.setup.radiusSeeded = false
		if r := []rune(d.setup.radiusMi); len(r) > 0 {
			d.setup.radiusMi = string(r[:len(r)-1])
		}
	default:
		if r := key.Text; r >= "0" && r <= "9" {
			// A KEY THAT CHANGES NOTHING SELECTS NOTHING (VALIDATE red team,
			// 2026-09-08). Moving this out of the length guard let a digit typed
			// into a full buffer flip the radio to "Within" while leaving the
			// number alone — a press that appears to choose and does not.
			if d.setup.radiusSeeded || len([]rune(d.setup.radiusMi)) < 4 {
				d.setup.filtered = true // typing a distance means Filtered
			}
			// THE FIRST DIGIT REPLACES THE STORED VALUE, the rest append. A
			// field showing a number the listener did not type is a field they
			// are about to type over, not one they are appending to.
			if d.setup.radiusSeeded {
				d.setup.radiusMi, d.setup.radiusSeeded = "", false
			}
			if len([]rune(d.setup.radiusMi)) < 4 {
				d.setup.radiusMi += r
			}
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

// currentTransmitter is where the station transmits from right now — its own
// setting when it has one, and the listener's default location when it does not
// (config.Station's own fallback, said the same way here).
func (d Dashboard) currentTransmitter() *snapshot.LocationRef {
	if d.cfg.Transmitter != nil {
		return d.cfg.Transmitter
	}
	return d.currentDefault()
}

// serviceBounds is the radius' floor and ceiling, as the app handed them over
// (HUM LEAD, 2026-09-13: "min 2mi - Max 100 mi").
//
// A FLOOR, NOT A ZERO. The alert radius has an "All" that means no fence; a
// service radius does not — a station serves a region or it is not set up. Two
// miles is the smallest region a transmitter can usefully be the centre of.
//
// THEY WERE CONSTANTS HERE, and the comment above them said `modes/tty` "may not
// import `platform/config` (make lint-imports)". THAT WAS NOT TRUE — it compiles
// and the gate passes. The real reason is a convention nothing had written down:
// no package under `modes/` reads storage, because the UI is handed what it
// needs. So the numbers now arrive through `Config`, `platform/config` owns them
// alone, and the tie-test that stood between two copies is retired.
//
// UNSET REFUSES EVERYTHING, and that is deliberate. `ok` is false until the app
// supplies the bounds, and `inServiceRange` then admits no radius at all — a
// window that offered 1..∞ because nobody configured it would write a radius the
// storage clamps behind the operator's back. Fail closed, visibly.
func (d Dashboard) serviceBounds() (lo, hi int, ok bool) {
	lo, hi = d.cfg.ServiceRadiusMinMi, d.cfg.ServiceRadiusMaxMi
	return lo, hi, lo > 0 && hi >= lo
}

// inServiceRange is the one question the three sites ask.
func (d Dashboard) inServiceRange(v int) bool {
	lo, hi, ok := d.serviceBounds()
	return ok && v >= lo && v <= hi
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
