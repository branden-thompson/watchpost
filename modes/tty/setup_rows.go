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

// setupScope is which surface a row belongs to (D-92), and it is D-18's ruling
// made structural.
//
// HUM LEAD, 2026-09-12: "Settings that are unique and specific to their mode
// should only appear in the settings modal of their mode, and should not be able
// to leak into the mode."
//
// THE ZERO VALUE IS UNRULED, DELIBERATELY. A row added without a scope is a row
// nobody has decided about, and the gate fails on it — the same shape as
// `reachabilityBaseline`. It still RENDERS at runtime, because a settings row
// that vanishes silently is worse than one that appears where it should not:
// the first is invisible, the second is reportable.
type setupScope int

const (
	// scopeUnruled is "nobody has decided". Only the gate treats it specially.
	scopeUnruled setupScope = iota
	// scopeShared is both surfaces, ONE value — D-18's S.
	scopeShared
	// scopeObserver is the listener's own — D-18's O.
	scopeObserver
	// scopeBroadcaster is the station's own — D-18's B.
	scopeBroadcaster
	// scopeSplit is both surfaces, INDEPENDENT values — D-18's SPLIT. It renders
	// like scopeShared; the separate storage is D-18's own additive migration and
	// is not built yet.
	scopeSplit
)

// NO `numSetupScopes` SENTINEL, and that is a decision rather than an omission.
//
// `wires` enrols a closed set by its bound and then asks every member to name a
// production WRITER — and `noteComposite` counts a member listed in a table
// literal as a READ, deliberately: "a set written out as a literal … is naming it
// as one of the things to consider."  A STATIC CLASSIFICATION therefore never has
// a writer, because nothing in production ever decides to produce one; the table
// simply states it.
//
// The two closest analogues are in this very file — `setupRowKind` and
// `setupGroupID`, both static classifications, both without a sentinel — so this
// matches the convention rather than dodging the gate.  What the gate would have
// bought is a range check on values that can only come from the table below;
// what the completeness check actually needs is `scopeUnruled`, and
// TestEverySettingsRowIsRuledForItsSurface asks for that directly.

// shownOn reports whether a row of this scope is drawn on a surface.
func (sc setupScope) shownOn(s Surface) bool {
	switch sc {
	case scopeObserver:
		return s != SurfaceBroadcaster
	case scopeBroadcaster:
		return s == SurfaceBroadcaster
	case scopeUnruled:
		// SHOWN, AND THE GATE FAILS ON IT. A settings row that vanishes silently
		// is worse than one that appears where it should not: the first is
		// invisible and the second is reportable.
		return true
	}
	// scopeShared and scopeSplit: both surfaces draw them.
	return true
}

// setupRow describes one focusable row.
type setupRow struct {
	id    setupRowID
	group setupGroupID
	scope setupScope
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
		// THE DEFAULT LOCATION IS THE LISTENER'S (D-18 row 1). The station's
		// epicentre is a different fact with a different owner — D-72 split them.
		rowLocation: {rowLocation, groupData, scopeObserver, rowInput, false, "", ""},
		rowFIRMSKey: {rowFIRMSKey, groupData, scopeShared, rowInput, false, "", ""},

		// DISPLAY PREFERENCES ARE ONE APP'S (D-18 rows 19, 21, 22).
		rowTheme:         {rowTheme, groupUI, scopeShared, rowPicker, true, "", ""},
		rowUnitsImperial: {rowUnitsImperial, groupUI, scopeShared, rowRadio, false, "", ""},
		rowUnitsMetric:   {rowUnitsMetric, groupUI, scopeShared, rowRadio, false, "", ""},
		rowClock12:       {rowClock12, groupUI, scopeShared, rowRadio, false, "", ""},
		rowClock24:       {rowClock24, groupUI, scopeShared, rowRadio, false, "", ""},
		rowClockMil:      {rowClockMil, groupUI, scopeShared, rowRadio, false, "", ""},

		// OBSERVER'S ALERT RADIUS (D-18 row 25, per D-20): it bounds ARRIVALS over
		// an unbounded location set. The station's service radius is a HARD bound
		// on LOOKUPS and a separate setting — two radii, not one.
		rowEventsAll:    {rowEventsAll, groupEvents, scopeObserver, rowRadio, false, "", ""},
		rowEventsWithin: {rowEventsWithin, groupEvents, scopeObserver, rowRadio, false, "", ""},

		// TONES ARE SPLIT (D-18 rows 8, 9): both surfaces have them, with
		// INDEPENDENT values. They render on both today; the separate storage is
		// D-18's own additive migration and is not built yet.
		rowClassDisaster:  {rowClassDisaster, groupTone, scopeSplit, rowToggle, false, "", "disaster"},
		rowClassWarning:   {rowClassWarning, groupTone, scopeSplit, rowToggle, false, "", "warning"},
		rowClassWatch:     {rowClassWatch, groupTone, scopeSplit, rowToggle, false, "", "watch"},
		rowClassAdvisory:  {rowClassAdvisory, groupTone, scopeSplit, rowToggle, false, "", "advisory"},
		rowClassStatement: {rowClassStatement, groupTone, scopeSplit, rowToggle, false, "", "statement"},
		rowClassStorm:     {rowClassStorm, groupTone, scopeSplit, rowToggle, false, "", "storm"},

		// ONE STATION, ONE SOUND (D-18 rows 6, 7, settled by D-11): the main track
		// is the rotation, so Broadcaster drives the same reads through the same
		// voices. The cast is SHARED.
		rowCastAlerts:   {rowCastAlerts, groupCast, scopeShared, rowPicker, true, roleAlerts, ""},
		rowCastWeather:  {rowCastWeather, groupCast, scopeShared, rowPicker, true, roleWeather, ""},
		rowCastMaritime: {rowCastMaritime, groupCast, scopeShared, rowPicker, true, roleMaritime, ""},
		rowCastFire:     {rowCastFire, groupCast, scopeShared, rowPicker, true, roleFire, ""},
		rowCastSeismic:  {rowCastSeismic, groupCast, scopeShared, rowPicker, true, roleSeismic, ""},

		// THE WATCHLIST ROTATION'S PACING IS THE MONITOR'S. Not in D-18's table
		// because neither is persisted — they live on the deck — but the rotation
		// they pace is `advancesMonitor()`'s, which cannot advance at all while the
		// console holds the air (D-74). A pacing control for something that cannot
		// happen is a control that lies.
		//
		// picker:true is what makes ←→ cycle this row. The field's name says
		// "voice picker", but setup.go gates the arrow keys on it for EVERY
		// picker — the theme row sets it for the same reason. False here draws a
		// perfect, inert control.
		rowRelayDwell: {rowRelayDwell, groupRelay, scopeObserver, rowPicker, true, "", ""},
		rowRelayLang:  {rowRelayLang, groupRelay, scopeObserver, rowPicker, true, "", ""},
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
	id, _ := visibleRowOfGroup(g, func(setupRowID) bool { return true })
	return id
}

// visibleRowOfGroup is the first row of a group that THIS SURFACE draws, and
// whether the group draws one at all (D-92).
//
// ONE FUNCTION FOR BOTH FACTS, because they are one walk. The first draft had
// `firstVisibleOfGroup` and `groupHasAVisibleRow` side by side and the `dupes`
// gate reported them as twins at 34 nodes — correctly: the loop was identical and
// only the return differed. A pair like that is two places for the visibility
// rule to drift.
//
// THE BOOL IS NOT REDUNDANT WITH THE ID. `rowLocation` is a real row AND the
// zero value, so "found rowLocation" and "found nothing" are indistinguishable
// without it.
func visibleRowOfGroup(g setupGroupID, visible func(setupRowID) bool) (setupRowID, bool) {
	table := setupTable()
	for id := setupRowID(0); id < setupRowCount; id++ { // bounded by the table (P10-02)
		if table[id].group == g && visible(id) {
			return id, true
		}
	}
	return rowLocation, false
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

// stepGroup walks to the next group that DRAWS something on this surface (D-92).
//
// COUNTER-BOUNDED, like stepRow and for the same reason: with every group hidden
// an unbounded walk would spin rather than leave the focus alone.
func stepGroup(cur setupRowID, step int, visible func(setupRowID) bool) setupRowID {
	table := setupTable()
	groups := setupGroups()
	at := 0
	for i, g := range groups {
		if g == table[cur].group {
			at = i
			break
		}
	}
	n := len(groups)
	for i := 1; i <= n; i++ { // bounded by the group set (P10-02)
		g := groups[((at+i*step)%n+n)%n]
		if id, ok := visibleRowOfGroup(g, visible); ok {
			return id
		}
	}
	return cur
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

// cycleIn moves one entry through a picker's list, wrapping at both ends.
//
// ONE OWNER FOR THE WRAP ARITHMETIC (metric D, 2026-09-08). The relay language
// and relay dwell pickers each carried their own copy of
// `((at+step)%len+len)%len` — the expression that makes -1 wrap to the end
// rather than panicking — and an off-by-one in one of them would be invisible
// in the other. The saving is not the six lines; it is that the arithmetic
// exists once.
//
// A value not in the list starts at index 0, which is what both copies did:
// a config written by hand can name a choice a later build removed, and the
// picker has to land somewhere.
func cycleIn[T any, K comparable](list []T, key func(T) K, cur K, forward bool) T {
	at := 0
	for i, it := range list { // bounded by the list (P10-02)
		if key(it) == cur {
			at = i
			break
		}
	}
	step := 1
	if !forward {
		step = -1
	}
	return list[((at+step)%len(list)+len(list))%len(list)]
}
