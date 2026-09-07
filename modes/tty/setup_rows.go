package tty

// setup_rows.go — the Setup window's row table (0.14.0 P4 Task 4.4).
//
// Before this the window had three questions and a focus enum that was a small
// state machine: each key handler knew which question it was in and what came
// next. Twenty rows across four groups cannot be written that way — the
// keyboard rule, the focus order, the › mark and the scroll would each end up
// with their own idea of the order, and they would drift.
//
// So there is ONE TABLE. It says what every focusable row is, which group it
// belongs to, how it is operated and what it edits; the focus becomes an INDEX
// into it rather than a state. Everything else reads the table.

import (
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
)

// setupGroupID names a group of rows — the headings the mock draws.
type setupGroupID int

const (
	groupData setupGroupID = iota
	groupUI
	groupEvents
	groupTone
	groupCast
	groupRelay
)

// setupRowKind is how a row is operated. It decides which keys do anything on
// it, so the keyboard rule is written once and read from here.
type setupRowKind int

const (
	// rowInput takes typed text (the location query, the FIRMS key, the miles).
	rowInput setupRowKind = iota
	// rowRadio is one option of a mutually exclusive set: space selects it.
	rowRadio
	// rowCheck is an independent toggle: space flips it.
	rowCheck
	// rowPicker cycles a list with ←→ and previews with p.
	rowPicker
	// rowToggle is a two-state control shown as its STATE, not as a box:
	// ←→ or space flip it, and the row says which state it is in.
	rowToggle
)

// setupRowID names each row. The ORDER OF THESE CONSTANTS IS THE FOCUS ORDER:
// ↑↓ walk them, tab jumps to the next group's first row. Nothing else defines
// the order, so nothing else can disagree about it.
type setupRowID int

const (
	// DATA
	rowLocation setupRowID = iota
	rowFIRMSKey

	// WATCHPOST UI — the display preferences. The
	// theme chooser was a modal of its own; it is one picker row here, and the
	// two questions under it had no home at all: the units were a live-only
	// [f]/[c] toggle nothing remembered, and the clock was whatever each site
	// had hard-coded.
	rowTheme
	rowUnitsImperial
	rowUnitsMetric
	rowClock12
	rowClock24
	rowClockMil

	// ALERTS - EVENTS
	rowEventsAll
	rowEventsWithin

	// ALERTS - TONE. No mode radio: each class carries its own state, so the
	// row says whether that class will sound rather than leaving a listener to
	// combine a mode with a checkbox.
	rowClassDisaster
	rowClassWarning
	rowClassWatch
	rowClassAdvisory
	rowClassStatement
	rowClassStorm

	// WATCHPOST RADIO - CORRESPONDENTS. Five pickers, no mode radio and no
	// enabling checkbox: each row is a thing a listener hears and the voice
	// that reads it. A row with no assignment of
	// its own shows the voice it INHERITS, so every row always names whoever
	// will actually speak.
	rowCastAlerts
	rowCastWeather
	rowCastMaritime
	rowCastFire
	rowCastSeismic

	// WATCHPOST RADIO - RELAY REPLAY. One picker today, and a group of its own
	// because the rotation is a different question from who reads: the rows
	// above are voices, this is pacing. Asked for as a group rather than a
	// sixth correspondent so the next pacing setting has somewhere to land.
	rowRelayDwell
	rowRelayLang

	setupRowCount
)

// setupRow describes one focusable row.
type setupRow struct {
	id    setupRowID
	group setupGroupID
	kind  setupRowKind

	// picker is true when the row carries a voice picker in addition to its
	// own control. The Single Voice row is a radio AND a picker (the mock draws
	// "○ Single Voice │ System Voice │ ▾ │"); an override row is a checkbox and
	// a picker.
	picker bool

	// role is the cast role key this row assigns, "" when it assigns none.
	// The KEYS COME FROM THE APP (the registry's own words) — modes/tty may not
	// import a domain, so they arrive as strings and a parity test in app pins
	// them to the registry.
	role string

	// class is the tone-class key this row mutes, "" when it mutes none.
	class string
}

// setupTable is the table. It is built rather than declared as a package var
// because a package-level table would be mutable process-wide state (P10-06)
// and every caller wants its own copy anyway — it is twenty small structs.
func setupTable() [setupRowCount]setupRow {
	return [setupRowCount]setupRow{
		rowLocation: {rowLocation, groupData, rowInput, false, "", ""},
		rowFIRMSKey: {rowFIRMSKey, groupData, rowInput, false, "", ""},

		rowTheme:         {rowTheme, groupUI, rowPicker, true, "", ""},
		rowUnitsImperial: {rowUnitsImperial, groupUI, rowRadio, false, "", ""},
		rowUnitsMetric:   {rowUnitsMetric, groupUI, rowRadio, false, "", ""},
		rowClock12:       {rowClock12, groupUI, rowRadio, false, "", ""},
		rowClock24:       {rowClock24, groupUI, rowRadio, false, "", ""},
		rowClockMil:      {rowClockMil, groupUI, rowRadio, false, "", ""},

		rowEventsAll:    {rowEventsAll, groupEvents, rowRadio, false, "", ""},
		rowEventsWithin: {rowEventsWithin, groupEvents, rowRadio, false, "", ""},

		rowClassDisaster:  {rowClassDisaster, groupTone, rowToggle, false, "", "disaster"},
		rowClassWarning:   {rowClassWarning, groupTone, rowToggle, false, "", "warning"},
		rowClassWatch:     {rowClassWatch, groupTone, rowToggle, false, "", "watch"},
		rowClassAdvisory:  {rowClassAdvisory, groupTone, rowToggle, false, "", "advisory"},
		rowClassStatement: {rowClassStatement, groupTone, rowToggle, false, "", "statement"},
		rowClassStorm:     {rowClassStorm, groupTone, rowToggle, false, "", "storm"},

		rowCastAlerts:   {rowCastAlerts, groupCast, rowPicker, true, roleAlerts, ""},
		rowCastWeather:  {rowCastWeather, groupCast, rowPicker, true, roleWeather, ""},
		rowCastMaritime: {rowCastMaritime, groupCast, rowPicker, true, roleMaritime, ""},
		rowCastFire:     {rowCastFire, groupCast, rowPicker, true, roleFire, ""},
		rowCastSeismic:  {rowCastSeismic, groupCast, rowPicker, true, roleSeismic, ""},

		// picker:true is what makes ←→ cycle this row. The field's name says
		// "voice picker", but setup.go gates the arrow keys on it for EVERY
		// picker — the theme row sets it for the same reason. False here draws a
		// perfect, inert control.
		rowRelayDwell: {rowRelayDwell, groupRelay, rowPicker, true, "", ""},
		rowRelayLang:  {rowRelayLang, groupRelay, rowPicker, true, "", ""},
	}
}

// The role keys, as the registry spells them. They are duplicated here as
// STRINGS because modes/tty may not import domains/radio/cast
// (make lint-imports); TestSetupRoleKeysMatchTheRegistry in app pins them, so
// a rename in the registry fails a test rather than silently unassigning a row.
const (
	roleRoot     = "voice"
	roleAlerts   = "alerts"
	roleWeather  = "weather"
	roleMaritime = "maritime"
	roleFire     = "fire"
	roleSeismic  = "seismic"
)

// setupGroupTitle is the heading the mock draws above a group. The Tone group
// names the key that toggles it, as the mock does.
func setupGroupTitle(g setupGroupID) string {
	switch g {
	case groupData:
		return "DATA"
	case groupUI:
		return "WATCHPOST UI"
	case groupEvents:
		return "ALERTS - EVENTS"
	case groupTone:
		// "ALERTS - TONE", not the sketch's bare "ALERTS": there is an
		// "ALERTS - EVENTS" group directly above it, and two adjacent groups
		// both called ALERTS would be ambiguous. The "( [M] toggles )" note is
		// gone — the rows now say Enabled or MUTED outright, so there is
		// nothing left for it to explain.
		return "ALERTS - TONE"
	case groupCast:
		return "WATCHPOST RADIO - CORRESPONDENTS"
	case groupRelay:
		return "WATCHPOST RADIO - RELAY REPLAY"
	}
	return ""
}

// firstOfGroup is the row tab lands on for each group — the five tab stops.
func firstOfGroup(g setupGroupID) setupRowID {
	table := setupTable()
	for id := setupRowID(0); id < setupRowCount; id++ {
		if table[id].group == g {
			return id
		}
	}
	return rowLocation
}

// setupGroups is every group, in draw order.
func setupGroups() []setupGroupID {
	return []setupGroupID{groupData, groupUI, groupEvents, groupTone, groupCast, groupRelay}
}

// nextRow is ↓ and prevRow is ↑. Both WRAP: ↓ on the last row returns to the
// first, ↑ on the first goes to the last.
//
// A carousel, because the window is a ring of twenty rows over two columns and
// there is nothing below the last one to reach. Stopping dead at an end reads
// as a stuck key — the listener presses again, nothing moves, and they have no
// way to tell that from a hung app.
//
// It matters MOST in the two-column layout, where the reading order is not
// obvious from the screen: ↓ walks the left column to its end and then jumps
// to the top of the right one, which nothing on the frame announces. A
// listener who over-shoots must be able to keep pressing and come back round,
// rather than having to work out which key retraces their steps.
func nextRow(cur setupRowID, visible func(setupRowID) bool) setupRowID {
	return stepRow(cur, 1, visible)
}

func prevRow(cur setupRowID, visible func(setupRowID) bool) setupRowID {
	return stepRow(cur, -1, visible)
}

// stepRow walks one visible row in either direction, wrapping.
//
// It is counter-bounded rather than looping until it finds one (P10-02): with
// every row hidden — which nothing can currently produce, but a future
// rowVisible could — an unbounded walk would spin forever rather than simply
// leaving the focus alone.
func stepRow(cur setupRowID, step int, visible func(setupRowID) bool) setupRowID {
	n := int(setupRowCount)
	for i := 1; i <= n; i++ {
		id := setupRowID(((int(cur)+step*i)%n + n) % n)
		if visible(id) {
			return id
		}
	}
	return cur
}

// nextGroup is tab: the first row of the next group, wrapping to the first.
func nextGroup(cur setupRowID) setupRowID {
	table := setupTable()
	groups := setupGroups()
	for i, g := range groups {
		if g == table[cur].group {
			return firstOfGroup(groups[(i+1)%len(groups)])
		}
	}
	return rowLocation
}

// prevGroup is shift+tab.
func prevGroup(cur setupRowID) setupRowID {
	table := setupTable()
	groups := setupGroups()
	for i, g := range groups {
		if g == table[cur].group {
			return firstOfGroup(groups[(i-1+len(groups))%len(groups)])
		}
	}
	return rowLocation
}

// enterSaves reports whether enter on this row SAVES rather than advancing.
//
// The rule is one sentence: ENTER ON A TEXT FIELD COMMITS IT AND MOVES ON;
// ENTER ANYWHERE ELSE SAVES.
//
// The rule this replaces — "enter saves on the last row of its group" — was
// wrong for exactly the rows a listener types into. The FIRMS key row is the
// last row of DATA, so arriving there and pressing enter (the natural "let me
// into this field" gesture) saved and closed the window, and there was no way
// to reach the field at all. (UAT 2026-08-30 #12: "I can never change or enter
// a FIRMS key".)
//
// With this rule the DATA flow is what it was before 0.14.0: type a location,
// enter, type a key, enter, enter to save.
func enterSaves(cur setupRowID) bool { return setupTable()[cur].kind != rowInput }

// setupMark and settingLabel are the shared list-focus pattern
// (platform/render/list.go), which is also where the reason it differs from the
// dashboard table's focus is written down.
func setupMark(o render.Opts, focused bool) string { return o.ListMark(focused) }

func settingLabel(text string, focused bool) string { return render.ListLabel(text, focused) }

// checkMark is a checkbox: [✔] ticked, [ ] not.
//
// A TICK, not an ✘ or an x. These boxes say "this class IS muted" and "this
// report HAS its own correspondent" — they enable a thing. An x reads as
// crossing something out, which inverts the metaphor exactly where the two
// groups are least alike: one silences, the other assigns (HUM LEAD, UAT
// 2026-08-30).
//
// The mark carries the state without colour (R-12a), and it goes through the
// glyph set so --ascii has its own form.
func checkMark(o render.Opts, ticked bool) string {
	if !ticked {
		return "[ ]"
	}
	return "[" + o.Glyphs().OK + "]"
}

// toggleCell draws a two-state control as `[←] STATE [→]` — the same chips as a
// voice picker, because it is the same gesture, and the STATE rather than a box.
//
// A checkbox made a listener combine two things to know an answer: the box's
// tick and the group's mode. The state word answers it outright, which is what
// the group is for.
func toggleCell(state string, c arrowChips, flash pickerFlash) string {
	left, right := c.pick(flash)
	return left + " " + render.PadTo(state, toggleStateW) + " " + right
}

// arrowChips are the ←→ key caps, rendered ONCE per frame and shared by every
// row that draws them.
//
// Thirteen controls draw two chips each, and a chip is a styled span: building
// twenty-six of them per frame — twenty-four of which are byte-identical —
// was the single largest thing this window allocated (when the
// tone toggles pushed the frame past its pin). The flashed pair is still built
// per press, which is one chip on one row.
type arrowChips struct{ left, right, litLeft, litRight string }

func newArrowChips(o render.Opts) arrowChips {
	return arrowChips{
		left: o.KeyCap("←"), right: o.KeyCap("→"),
		litLeft: o.KeyCapInverted("←"), litRight: o.KeyCapInverted("→"),
	}
}

func (c arrowChips) pick(flash pickerFlash) (left, right string) {
	switch flash {
	case flashLeft:
		return c.litLeft, c.right
	case flashRight:
		return c.left, c.litRight
	}
	return c.left, c.right
}

// toggleStateW holds the widest state word, so a column of toggles lines up.
const toggleStateW = 7 // "Enabled"

// pickerCell draws a voice picker as `[←] <name> [→]` — key chips, not the
// mock's `│ <name> │ ▾ │` dropdown.
//
// The mock drew a dropdown because it assumed a sub-panel would open. It does
// not: `←→` cycle the list in place. A control that LOOKS like a dropdown and
// is not is a promise the window cannot keep, and the chips say exactly which
// keys move it — the same shape as the player's volume control, which is the
// other place in the app where two keys step through a value in place.
//
// The name is padded to pickerNameW so a column of pickers lines up, and
// truncated there when it is too long — the full name goes in the row's note
// (Task 4.6), never lost. PlainLine here, once: a voice name can come from a
// hand-edited config, and this is the one place it is drawn (NFR-6).
func pickerCell(name string, c arrowChips, flash pickerFlash) string {
	// The press acknowledgement is an INVERSION, not a colour.
	//
	// The volume's chips blink green up and red down because those directions
	// mean something and either can be at its end. A voice list WRAPS: neither
	// direction is more than the other and neither can be exhausted, so a
	// colour would be saying something untrue. An inversion says only "that
	// landed" — and reads the same on a light terminal as a dark one, because
	// it swaps whatever the chip's own colours already are.
	return pickerCellW(name, c, flash, pickerNameW)
}

// pickerCellW is pickerCell at a stated width. The voice pickers all want the
// same wide cell so their chips line up down the column; a picker whose values
// are SHORT (a duration, say) wants a cell its values fit, because a cell
// sized for "Samantha (Enhanced)" carrying "30s" is eleven columns of nothing
// — and eleven columns is the difference between the window fitting in two
// balanced columns at 133 and needing a scroll rail.
func pickerCellW(name string, c arrowChips, flash pickerFlash, w int) string {
	left, right := c.pick(flash)
	return left + " " + render.PadTo(render.TruncateCells(render.PlainLine(name), w), w) + " " + right
}

// pickerFlash is which chip is blinking on a picker.
type pickerFlash int

const (
	flashNone pickerFlash = iota
	flashLeft
	flashRight
)

// pickerFlashFor is the blink state of one row's picker: only the FOCUSED row
// can blink, and only while its window is open.
func (d Dashboard) pickerFlashFor(id setupRowID) pickerFlash {
	if d.setup.focus != id || d.setup.flash == flashNone || !time.Now().Before(d.setup.flashEnd) {
		return flashNone
	}
	return d.setup.flash
}

// pickerNameW is the picker's name column: "System Voice" plus room for the
// longer catalogue names without widening the window.
const pickerNameW = 17

// pickerFlashFor's window, matching the volume chips' 350 ms (UAT 41).
const pickerFlashDur = 350 * time.Millisecond
