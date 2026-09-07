# P4 — the Setup window, the Radio panel, the retirements (multi-voice-support, 0.14.0)

> **PRIOR ART — not the plan.** A verbatim snapshot of this batch document *before* the PLAN artefacts were
> stripped to task shape (`AP-PLANCODE-01`, `../../06-key_learnings/retro-notes.md` RN-1). The artefact of
> record is `../p4-ui.md`. Read `README.md` in this folder first: it says which of these blocks were actually
> compiled and run, and lists the known defects (D-1 … D-9) not to copy forward.



```
Goal:         Assign the cast and the tone mutes in Setup per the HUM LEAD's mock (FR-4, FR-11, FR-13, MVS-D-3/10/25..28);
              retire [V] and [T] and re-lay the Radio panel per breakpoint (MVS-D-23/24); the `voice` action becomes
              a Setup deep-link and a removed action never breaks startup (FR-14); the [S] cast table (FR-7); help,
              README, CHANGELOG, docs, goldens, the PTY journey (M2), the Setup alloc pin (NFR-3), NFR-8.
Architecture: plan.md §2.7; 02-analysis/mocks/{setup,radio-panel}.md (reproduce exactly; colour is the HUM LEAD's pass)
Tech Stack:   bubbletea v2 · go-studs (vendored) · the frame goldens (modes/tty/testdata, -update-golden) · expect (PTY)
Branch:       feature/multi-voice-support
Gate:         goldens; make pty-severe + the journey; alloc pins; grep zero for [V]/[T]/radio-size; make verify; p10
```

## 0. In plain words

Setup grows two groups. On the left, under today's DATA and ALERTS - EVENTS questions, **ALERTS - TONE** lets
the listener keep every alert tone on or mute the classes they choose. On the right, **WATCHPOST RADIO -
CORRESPONDENTS** lets them keep one voice for everything or give the alerts and the reports their own
correspondents, with four optional per-report overrides; `p` auditions the focused voice. `[V]` and `[T]` leave
the Radio panel, `V` now opens Setup at the correspondents, and the panel takes one of three fixed layouts by
width. `[S]` shows who speaks for whom and why.

## Constraints every task honours

- **`modes/tty` imports no `domains/` package** (`scripts/lint-imports.sh`, `make lint-imports`). The cast reaches
  the TUI as strings: `CastView` keys roles by `cast.Role.Key()`'s words (`voice alerts standard weather maritime
  fire seismic`) and the tone classes arrive as `[]ToneClass{Key, Label}` from the app (one owner: `cast`); a
  parity test in `app/` pins the TUI's keys to the registry.
- Every UI row is the mock's, character for character; the mock's two spellings ("Maritme", "Hostpots") ship
  corrected. The mock draws the `›` focus slot on every row; the slot is **blank on unfocused rows** (today's
  rule, UAT 111.3 — the mark is the focus). Widths: the Help modal's two-column rule (`twoColumnsWidth` +
  `panelChromeFor`, `modes/tty/columns.go`) decides two columns vs one; 80×24 is the single-column case.
- **Keyboard rule (one rule for the whole window):** `tab`/`shift+tab` move between *questions* (the first row
  of each group — location · key · events · tone · correspondents; five stops); `↑↓` move among the rows *inside*
  a group (in DATA, `↑↓` pick a location hint while one is open, else move); `space` selects the focused radio or
  toggles the checkbox; `←→` cycle the focused picker; `enter` accepts the row and moves to the next, and on the
  **last row saves**; `esc` cancels. The ALERTS - EVENTS rows follow the same rule (two radios, `space` picks;
  the radius digits type on the *Within* row). The chips are the mock's four, **pinned under the scroll window**
  (the mock draws them on the frame's last row beside the rail's `▼`); `←→  Voice` and `p  Preview` join them
  while a picker is focused (MVS-D-27; OP-1); OP-4 asks the HUM LEAD about the `↑↓  Pick` wording.
- **Notes sit under the row they explain** (the not-installed / no-audio / progress notes under the focused
  picker; the "nothing ticked = every class" line under *Mute:*) and the scroll keeps the row *and its note* on
  screen. A note is **wrapped to `noteWidth` = 44 cells** (`render.WrapLines`) into as many `nil` rows as it
  needs — never trimmed, and never wider than the picker rows (52 cells), so focusing a picker never flips the
  two-column layout. Every other Setup line is trimmed to its column (`truncateTo`) so the wrapped body and the
  focus-row arithmetic never diverge. The footer is ≤ 3 rows (two chip rows + the blank) at 80×24: the frame is
  title + 12 body + 3 footer + rule = 17 of 24 rows; `TestSetupFrameFitsAt80x24` pins `frame height ≤ d.height`
  with the widest footer (a picker focused, `--ascii`).
- Reuse, never re-roll: `sideBySide`/`twoColumnsWidth`/`widest`/`panelChromeFor` (columns.go), `radioMark`
  (setup.go:365), `render.WrapSegments`, `o.KeyCap` (already the mock's chip on every surface: the `KeyChip`
  ground with colour, `[key]` without, `asciiKey` words under --ascii), `render.PadTo`/`PadBetween`,
  `render.PlainLine`, `o.Glyphs()` for every mark (`Pointer`, `OK`, `Fail`, `Alert`, `Down`, `Rail`, `Arrow`),
  the modal memo, the `wrapModal` rail.

## File map

```
CREATE: modes/tty/setup_rows.go          — the row table (setupRow), focus ids, the keyboard rule's helpers
CREATE: modes/tty/setup_cast.go          — the CORRESPONDENTS group: CastView, pickers, rows, keys, notes
CREATE: modes/tty/setup_tones.go         — the ALERTS - TONE group: ToneClass rows, mode radio, the six checkboxes
CREATE: modes/tty/setup_helpers_test.go  — newTestDashboard · plainLines · containsLine · indexOfLine · openSetupAt · press
CREATE: modes/tty/setup_cast_test.go, setup_tones_test.go, setup_layout_test.go, setup_golden_test.go
CREATE: modes/tty/modal_theme.go (+ modal_theme_test.go) — the theme chooser, moved whole from modal_chooser.go
DELETE: modes/tty/modal_chooser.go, modal_chooser_test.go (the voice half)
MODIFY: modes/tty/setup.go               — the body built once (setupBody); focus-following scroll; setupSave; group headers; EVENTS as two rows
MODIFY: modes/tty/view.go                — "Setup / Configs"; floatModalFooter (pinned chips); modalWidth → setupWidth; modalVoice gone
MODIFY: modes/tty/dashboard.go           — Config hooks; `voice` → openSetupAt; radio-size gone; voice* fields gone; SetupNoteMsg; setupGen
MODIFY: modes/tty/memo.go                — the voice* keys gone; setupGen in the key
MODIFY: modes/tty/radio_panel.go         — radioBreakpoint (outer columns); radioLines(o, bp); the head; controls without V/T
MODIFY: modes/tty/layout.go              — layoutRows{full, compact} built once from radioLines; radioMin gone
MODIFY: modes/tty/body.go                — the [M] label: "Alert Tones: On / Muted (n of 6) / Muted (all)"
MODIFY: modes/tty/help_about.go          — the RADIO group without radio-size
MODIFY: modes/tty/status.go, dashboard.go — Stats.Cast / Stats.ConfigNotes; the CAST and CONFIG blocks (Reason printed)
MODIFY: platform/render/units.go         — Glyphs.Down (▾/v), Rail (│/|), Arrow (→/->), Ellipsis (…/...), Bullet (•/*)
MODIFY: platform/render/panel.go         — ScrollPanel's rail through RailGlyphs (ASCII form)
MODIFY: platform/term/term.go            — Merge tolerates an override for an unknown action (returns notes)
MODIFY: app/dashboard.go, app/cast.go (+ cast_test.go), app/stats.go, app/dump.go, app/voices.go (listVoices → hostFacts) — castViewOf, saveCast, saveTones, ToneClasses, castRowOf
CREATE: app/hostfacts.go (+ hostfacts_test.go) — hostFacts (one cast.Host for the deck and the report), CastReport
MODIFY: cmd/watchpost/root.go (+ report_test.go) — the `cast:` block and `tones:` line under --verbose
MODIFY: modes/tty/{dashboard,radio_panel,status}_test.go — the RED tests Tasks 4.2/4.3/4.8 append
MODIFY: platform/render/contrast.go      — aaPairs × modal (TextBright, ProviderOK, ProviderDown) + the composite self-check
MODIFY: scripts/quality/validate-journey.expect — the cast step (M2, counted; pinned ≤ 11)
MODIFY: modes/tty/bench_test.go          — TestSetupAllocBudget re-pinned (spike-then-pin)
MODIFY: README.md, CHANGELOG.md, docs/where-things-happen.md, docs/extending.md, docs/img (HUM LEAD captures)
MODIFY: modes/tty/testdata/*.golden      — re-recorded once (-update-golden); new frame-setup-{80x24,133x44,133x44-ascii}.golden
```

---

### Task 4.0 — the test scaffolding (every later RED test uses it)

**File:** `modes/tty/setup_helpers_test.go`

```go
package tty

import (
	"strings"
	"testing"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// newTestDashboard is the package's model for tests: no hooks, 80×24 (the
// constructor body_test.go and bench_test.go use).
func newTestDashboard(t *testing.T) Dashboard {
	t.Helper()
	d, err := NewDashboard(Config{Version: "t"})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// plainLines strips styling from every line.
func plainLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = render.Plain(l)
	}
	return out
}

// containsLine reports whether any plain line contains text.
func containsLine(lines []string, text string) bool { return indexOfLine(lines, text) >= 0 }

// indexOfLine is the first plain line containing text, or -1.
func indexOfLine(lines []string, text string) int {
	for i, l := range lines {
		if strings.Contains(render.Plain(l), text) {
			return i
		}
	}
	return -1
}

// press sends one key to the model — a rune: printable keys AND the special
// keys, which bubbletea v2 names as runes (tea.KeyTab, tea.KeyUp, …) and
// which carry no Text — and runs any command it returns, feeding the message
// back (the shape the existing Setup tests use for committedMsg).
func (d Dashboard) press(t *testing.T, key rune) Dashboard {
	t.Helper()
	msg := tea.KeyPressMsg{Code: key}
	if unicode.IsPrint(key) {
		msg.Text = string(key)
	}
	m, cmd := d.Update(msg)
	d = m.(Dashboard)
	if cmd != nil {
		if out := cmd(); out != nil {
			m, _ = d.Update(out)
			d = m.(Dashboard)
		}
	}
	return d
}
```

(`openSetupAt` is production code the `voice` action needs — Task 4.2 defines it in `setup.go`.) Probe, recorded
in the P4 build log: `tea.KeyPressMsg{Code: ' ', Text: " "}.String() == "space"`; `o.KeyCap("space")` with colour
off is `[space]` — the strings every RED test below pins. Today's `setup.go:171` carries the same dead `" "` case;
Task 4.4 fixes it with the EVENTS rewrite. **Verify:** `go vet ./modes/tty`

---

### Task 4.1 — `term.Merge` tolerates an override for an action that no longer exists (FR-14; RS-21)

**File:** `platform/term/term_test.go` (RED — append)

```go
func TestMergeIgnoresOverridesForUnknownActions(t *testing.T) {
	defaults := KeyMap{"setup": {Keys: []string{"s"}}, "voice": {Keys: []string{"V"}}}
	merged, notes, err := Merge(defaults, KeyMap{"radio-size": {Keys: []string{"T"}}, "voice": {Keys: []string{"x"}}})
	if err != nil {
		t.Fatalf("an override for a retired action is ignored, never an error: %v", err)
	}
	if merged["voice"].Keys[0] != "x" || merged["radio-size"].Keys != nil {
		t.Fatalf("known overrides apply, unknown ones are dropped: %+v", merged)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "radio-size") {
		t.Fatalf("the dropped override is noted for [S]: %v", notes)
	}
}
```

**File:** `platform/term/term.go` (GREEN) — `Merge(layers ...KeyMap) (KeyMap, []string, error)`: the first layer
is the defaults; before `maps.Copy` of each later layer, actions absent from the first layer are dropped into
`notes` (`"[keys] " + plaintext.Line(string(action)) + ": no such action in this version — ignored"` — the
action name is config text) instead of copied. Every caller (`modes/tty/dashboard.go:314`, the term tests)
takes the notes; `NewDashboard` keeps them on `d.keyNotes []string` for `[S]` (Task 4.8).
**Verify:** `go test ./platform/term ./modes/tty -run 'Merge|Keys' -count=1`

---

### Task 4.2 — the `voice` action opens Setup at the Correspondents group; `[T]` retires (FR-14; MVS-D-19/23)

**File:** `modes/tty/dashboard_test.go` (RED — append)

```go
func TestVoiceKeyOpensSetupAtTheCastOnScreen(t *testing.T) {
	d := newTestDashboard(t) // 80×24
	d = d.press(t, 'V')
	if d.modal != modalSetup || d.setup.focus != focusCastSingle {
		t.Fatalf("V opens Setup focused on the Correspondents group: modal=%v focus=%v", d.modal, d.setup.focus)
	}
	if row := d.setupFocusRow(); row < d.modalScroll || row >= d.modalScroll+d.modalMax() {
		t.Fatalf("the focused row is on screen at 80×24: row %d, window [%d,%d)", row, d.modalScroll, d.modalScroll+d.modalMax())
	}
	if _, ok := defaultKeyMap()["radio-size"]; ok {
		t.Fatal("[T] Size is retired (MVS-D-23)")
	}
}
```

**File:** `modes/tty/dashboard.go` (GREEN) — `defaultKeyMap()`: `"voice": {Keys: []string{"V"}, Help: "Correspondents (Setup)"}`,
delete `"radio-size"`; `toggleModal`: `case "voice": return d.openSetupAt(focusCastSingle), true`. Delete the
`radioMin` field, `modalVoice` (the enum value, its `view.go:64/93` cases, the `handleKey` route at `:573`),
`voiceIdx`, `voiceErr`, `radioVoice`, `voiceErrMsg`, `WithVoices`' `current` and `set` parameters
(`WithVoices(list func() []string, preview func(string))`), `cfg.SetVoice`, `cfg.Voice`, and `deck.VoiceName`
(app; no caller remains). **Keep** `voiceList` (the snapshot the pickers read — UAT 85: taken when Setup opens,
never in View) and `voices()`. `VoiceNoteMsg` → `SetupNoteMsg{Voice, Text string}` (the deck's word about one voice, drawn under the picker
that shows it; an empty `Text` clears): `case SetupNoteMsg: d.setupNote = SetupNoteMsg{Voice: render.PlainLine(v.Voice), Text: render.PlainLine(v.Text)}; d.setupGen++; return d.scrollSetupToFocus(), nil`
(the mirror of `RadioStatusMsg`'s `:786`; the scroll brings a note landing under the window's last row into
view); `d.voiceNote string` → `d.setupNote SetupNoteMsg`. `radio_panel.go:45-46` (`radio-size`) goes.
`help_about.go:164`'s RADIO group loses `"radio-size"`. `memo.go:147-148,175-176`: drop `voiceNote, voiceErr,
radioVoice, voiceIdx, nvoices`; the `setup` projection (`memo.go:181`, a `%+v` of the whole state) becomes
`setupGen uint64` on the model — bumped in `handleSetupKey` after every handled key, in `openSetup`, on
`SetupNoteMsg` and on resize — carried in the memo key instead of the string.

**File:** `modes/tty/setup.go` — `openSetupAt(f setupFocus) Dashboard { d = d.openSetup(); d.setup.focus = f; return d.scrollSetupToFocus() }`.
**Order:** this task's RED test needs `focusCastSingle`/`setupFocusRow`/`scrollSetupToFocus` (4.4) and the cast
rows (4.6); the deletions and the theme move land now, `openSetupAt` and the test are verified at 4.6's gate.
**File:** `modes/tty/modal_theme.go` — `handleThemeKey`, `openTheme`, `themeLines` moved verbatim from
`modal_chooser.go` (with the theme cases of `modal_chooser_test.go` → `modal_theme_test.go`); then
`modal_chooser.go` is deleted with its voice half (`openVoice`, `voiceChip`, `handleVoiceKey`, `voiceLines`,
`voiceErrMsg`). **Verify:** `go build ./... && go test ./modes/tty -run 'VoiceKey|Theme|Help|Keys' -count=1`

---

### Task 4.3 — the Radio panel per breakpoint (MVS-D-23/24; `02-analysis/mocks/radio-panel.md`)

**File:** `modes/tty/radio_panel_test.go` (RED — append)

```go
func TestRadioPanelBreakpoints(t *testing.T) {
	// The mock, in OUTER columns: wide (146) = head + 3 viz rows + marquee + controls; medium (84) = head + 1 viz +
	// marquee + controls; narrow (66) = head + marquee + keys-only controls; [V] and [T] gone; v follows the viz area.
	for _, c := range []struct {
		width, rows int
		bp          radioBreakpoint
		controls    string
		viz         bool
	}{
		{146, 6, radioWide, "[space]  Play   [r]  Repeat: Off   [m]  Mode: Synth   [v]  Viz: On", true}, // KeyCap with colour off (tests) is [key]; with colour on, the chip
		{110, 6, radioWide, "[space]  Play", true},
		{109, 4, radioMedium, "[space]  Play", true},
		{84, 4, radioMedium, "[space]  Play   [r]  Repeat: Off   [m]  Mode: Synth   [v]  Viz: On", true},
		{80, 4, radioMedium, "[space]  Play", true},
		{79, 3, radioNarrow, "[ space ] [ r ] [ m ]", false},
		{66, 3, radioNarrow, "[ space ] [ r ] [ m ]", false},
	} {
		d := newTestDashboard(t)
		d.width, d.radioViz = c.width, true
		if got := d.radioBreakpoint(false); got != c.bp {
			t.Fatalf("%d cols: breakpoint %v, want %v", c.width, got, c.bp)
		}
		lines := plainLines(d.radioLines(d.opts(), c.bp))
		if len(lines) != c.rows {
			t.Fatalf("%d cols: %d rows, want %d:\n%s", c.width, len(lines), c.rows, strings.Join(lines, "\n"))
		}
		last := lines[len(lines)-1]
		if !strings.Contains(last, c.controls) || strings.Contains(last, "Voice:") || strings.Contains(last, "Size:") {
			t.Fatalf("%d cols controls: %q", c.width, last)
		}
		if strings.Contains(last, "v") != c.viz {
			t.Fatalf("%d cols: the v control follows the visualizer area", c.width)
		}
		if !strings.Contains(lines[0], "VOL") || !strings.Contains(lines[0], "STOPPED") {
			t.Fatalf("%d cols head: %q", c.width, lines[0])
		}
	}
	d := newTestDashboard(t)
	d.width = 146
	if d.radioBreakpoint(true) != radioNarrow {
		t.Fatal("a height-compact frame takes the narrow player whatever the width (the 0.12.0 rule kept)")
	}
}
```

**File:** `modes/tty/radio_panel.go` (GREEN)

```go
// radioBreakpoint is the player's layout for the terminal width in OUTER
// columns, the way the mock states them (MVS-D-23): wide from 110 columns,
// medium from 80, narrow below. Each has a standard vertical size — [T] Size
// is retired; a height-compact frame (layout.go) takes the narrow player
// whatever the width.
type radioBreakpoint int

const (
	radioWide radioBreakpoint = iota
	radioMedium
	radioNarrow
)

const (
	radioWideCols   = 110
	radioMediumCols = 80
)

func (d Dashboard) radioBreakpoint(compact bool) radioBreakpoint {
	switch {
	case compact:
		return radioNarrow
	case d.width >= radioWideCols:
		return radioWide
	case d.width >= radioMediumCols:
		return radioMedium
	}
	return radioNarrow
}

// barWidthFor is the volume bar per breakpoint (no map literal per frame).
func barWidthFor(bp radioBreakpoint) int {
	switch bp {
	case radioWide:
		return 20
	case radioMedium:
		return 10
	}
	return 6
}

// vizRowsFor is the visualizer area's height per breakpoint (0 = no area).
func vizRowsFor(bp radioBreakpoint) int {
	switch bp {
	case radioWide:
		return 3
	case radioMedium:
		return 1
	}
	return 0
}

// radioLines builds the player rows for a breakpoint. ONE builder feeds both
// rendering and the height budget, so what is measured is what is drawn.
func (d Dashboard) radioLines(o render.Opts, bp radioBreakpoint) []string {
	inner := o.BoxInnerWidth()
	lines := []string{d.radioHead(o, bp, d.volControl(o, barWidthFor(bp)))}
	if rows := vizRowsFor(bp); rows > 0 { // the area exists at wide and medium; the viz fills it or it stays blank
		if d.radioViz {
			lines = append(lines, d.vizRows(inner, rows)...)
		} else {
			lines = append(lines, d.blankTrackRows(inner, rows)...)
		}
	}
	lines = append(lines, d.marqueeTrack(inner))
	return append(lines, d.radioControlLines(o, inner, bp)...)
}

// radioHead is the header row: the station name (wide only) · the on-air
// detail (the product as its code at medium and narrow, the place as its
// ZIP at narrow) · the volume bar · the state. No voice name (MVS-D-24).
func (d Dashboard) radioHead(o render.Opts, bp radioBreakpoint, vol string) string {
	inner := o.BoxInnerWidth()
	head := d.onAirDetail(bp)
	if bp == radioWide {
		head = render.Tint("WATCHPOST WEATHER RADIO", render.Tok(render.RadioAccent)) + " • " + head
	}
	tail := vol + "   " + d.radioStateLabel()
	if room := inner - 2 - render.Width(tail); render.Width(head) > room {
		head = d.narrowHead(room) // the existing shortener: short form, then ellipsis, then the mark alone
	}
	return " " + render.PadBetween(head, tail, inner-1)
}

// onAirDetail is the station line, or — while a read is on the air below
// wide — its short form (the product code; the place as its ZIP at narrow,
// which the app's short form already carries).
func (d Dashboard) onAirDetail(bp radioBreakpoint) string {
	if d.radioShort != "" && bp != radioWide {
		return d.opts().Glyphs().Note + " " + render.Tint(d.radioShort, render.Tok(render.RadioStation))
	}
	return d.station()
}

// blankTrackRows are the visualizer area's rows while the viz is off.
func (d Dashboard) blankTrackRows(inner, rows int) []string {
	out := make([]string, rows)
	for i := range out {
		out[i] = trackLine("", inner)
	}
	return out
}
```

`radioControlLines(o, inner, bp)`: the segments become `space  Play`, `r  Repeat: …`, `m  Mode: …`, and — when
`vizRowsFor(bp) > 0` — `v  Viz: …`, each `o.KeyCap(key) + "  " + label` (the chip with colour on; `[key]` with
colour off — what the tests and the goldens pin); at narrow the segments are keys only, the mock's literal form
`"[ " + key + " ]"` joined by a space (`[ space ] [ r ] [ m ]`, the same under --ascii). The
`V`/`T` segments go. `radioParts`, `radioCompactRows`, `radioMaxRows` are deleted (`radioLines` is the one
builder; `narrowHead`, `vizRows`, `marqueeTrack`, `station`, `radioStateLabel`, `volControl` stay).
`radio_panel_test.go:21` (`"[space] Play"`) is updated to the two-space form. `layout.go:37-47,76-80`: the pre-build stays — `built := layoutRows{full: d.radioLines(o, d.radioBreakpoint(false)), compact: d.radioLines(o, radioNarrow)}`
once per frame, then `fl.radioRows = built.full; if fl.compact { fl.radioRows = built.compact }` (no `radioMin`
branch; `radioLines` is never called inside `layoutWith`, which runs twice per frame). **Verify:**
`go test ./modes/tty -run 'RadioPanel|Radio|Layout' -count=1` (goldens re-record in Task 4.12, not here)

---

### Task 4.4 — the row table, the focus order, the body built once, two columns, focus-following scroll (RS-19; the Help rule)

**File:** `modes/tty/setup_layout_test.go` (RED)

```go
func TestSetupRowsAreTheMocksOrder(t *testing.T) {
	rows := setupRows()
	if len(rows) != 20 || rows[0].focus != focusLocation || rows[len(rows)-1].focus != focusCastSeismic {
		t.Fatalf("20 rows, location first, the seismic override last: %+v", rows)
	}
	if q := questionRows(); len(q) != 5 || q[1] != focusKey || q[3] != focusToneOn || q[4] != focusCastSingle {
		t.Fatalf("five questions (tab stops): %v", q)
	}
}

func TestSetupIsTwoColumnsWideAndOneNarrow(t *testing.T) {
	d := newTestDashboard(t)
	d.width, d.height = 133, 44
	d = d.openSetup()
	body := d.setupBody(d.opts())
	if !containsLine(body.lines, "DATA") || !containsLine(body.lines, "WATCHPOST RADIO - CORRESPONDENTS") {
		t.Fatalf("both groups: %v", plainLines(body.lines))
	}
	if !body.twoCol || indexOfLine(body.lines, "WATCHPOST RADIO - CORRESPONDENTS") != indexOfLine(body.lines, "DATA") {
		t.Fatal("at 133 cols the two column groups share a row (the Help rule)")
	}
	d.width, d.height = 80, 24
	d = d.resized() // the resize path's helper: bumps setupGen
	body = d.setupBody(d.opts())
	if body.twoCol || indexOfLine(body.lines, "WATCHPOST RADIO - CORRESPONDENTS") <= indexOfLine(body.lines, "ALERTS - TONE") {
		t.Fatal("at 80 cols the right column rolls under the left")
	}
}

func TestEverySetupRowIsOnScreenWhenFocusedAt80x24(t *testing.T) {
	d := newTestDashboard(t)
	d = d.openSetup()
	for f := setupFocus(0); f < setupRowCount; f++ {
		d.setup.focus = f
		d.setupGen++
		d = d.scrollSetupToFocus()
		row := d.setupFocusRow()
		if row < d.modalScroll || row >= d.modalScroll+d.modalMax() {
			t.Fatalf("row %v (%d) outside the window [%d, %d) (RS-19)", f, row, d.modalScroll, d.modalScroll+d.modalMax())
		}
	}
}
```

**File:** `modes/tty/setup_rows.go` (GREEN)

```go
package tty

// setup_rows.go — the Setup window's row table (the HUM LEAD's mock,
// 2026-08-29): every focusable row in the mock's order, which group it
// belongs to and what kind of control it is. The keyboard rule, the focus
// order, the › mark and the focus-following scroll all read this one table.

// setupFocus identifies a focusable row.
type setupFocus int

const (
	focusLocation setupFocus = iota // DATA: default location
	focusKey                        // DATA: NASA FIRMS key (its own tab stop — a question of its own)
	focusAlertAll                   // ALERTS - EVENTS: ○ All locations (Default)
	focusAlertWithin                // ALERTS - EVENTS: ○ Within │    mi of Default Location (the digits type here)
	focusToneOn                     // ALERTS - TONE: All Tones On
	focusToneMute                   // ALERTS - TONE: Mute:
	focusToneClass0                 // … six class checkboxes follow in the app's class order
	focusToneClass1
	focusToneClass2
	focusToneClass3
	focusToneClass4
	focusToneClass5
	focusCastSingle   // CORRESPONDENTS: Single Voice (+ the root picker)
	focusCastOn       // CORRESPONDENTS: Correspondent Cast:
	focusCastAlerts   // Alerts / Takeovers picker
	focusCastStandard // All Reports picker
	focusCastWeather  // [ ] Location Report override + picker
	focusCastMaritime // [ ] Maritime Report
	focusCastFire     // [ ] Fire/Hotspots
	focusCastSeismic  // [ ] Seismic — the last row: enter saves
	setupRowCount
)

// setupGroupID names a settings group; tab moves between the questions.
type setupGroupID int

const (
	groupLocation setupGroupID = iota
	groupKey
	groupEvents
	groupTones
	groupCast
)

// rowKind is what the row's keys act on.
type rowKind int

const (
	rowInput  rowKind = iota // typed text (location, key)
	rowRadio                 // one of a set (space selects)
	rowCheck                 // a checkbox (space toggles)
	rowPicker                // a voice picker (←→ cycle, p previews); the override rows carry the checkbox AND the picker
)

// setupRow is one focusable row.
type setupRow struct {
	focus setupFocus
	group setupGroupID
	kind  rowKind
	key   string // the role key (cast rows); "" otherwise — the class rows read their class from setupState.classes
}

// ToneClassCount is the number of class rows the mock draws; the app's list
// is pinned to it (app/cast_test.go: TestToneClassesMatchTheSetupRows).
const ToneClassCount = 6

// setupRows is the table, in the mock's order (left column, then right).
func setupRows() []setupRow {
	rows := []setupRow{
		{focusLocation, groupLocation, rowInput, ""},
		{focusKey, groupKey, rowInput, ""},
		{focusAlertAll, groupEvents, rowRadio, ""},
		{focusAlertWithin, groupEvents, rowRadio, ""},
		{focusToneOn, groupTones, rowRadio, ""},
		{focusToneMute, groupTones, rowRadio, ""},
	}
	for i := 0; i < ToneClassCount; i++ {
		rows = append(rows, setupRow{focusToneClass0 + setupFocus(i), groupTones, rowCheck, ""})
	}
	return append(rows,
		setupRow{focusCastSingle, groupCast, rowPicker, castKeyRoot},
		setupRow{focusCastOn, groupCast, rowRadio, ""},
		setupRow{focusCastAlerts, groupCast, rowPicker, castKeyAlerts},
		setupRow{focusCastStandard, groupCast, rowPicker, castKeyStandard},
		setupRow{focusCastWeather, groupCast, rowPicker, castKeyWeather},
		setupRow{focusCastMaritime, groupCast, rowPicker, castKeyMaritime},
		setupRow{focusCastFire, groupCast, rowPicker, castKeyFire},
		setupRow{focusCastSeismic, groupCast, rowPicker, castKeySeismic},
	)
}

// rowOf is a focus id's row; a zero row out of range (never a panic).
func rowOf(f setupFocus) setupRow {
	if f < 0 || f >= setupRowCount {
		return setupRow{}
	}
	return setupRows()[f]
}

// questionRows are the tab stops: the first row of each group.
func questionRows() []setupFocus {
	var out []setupFocus
	last := setupGroupID(-1)
	for _, r := range setupRows() {
		if r.group != last {
			out, last = append(out, r.focus), r.group
		}
	}
	return out
}

// nextQuestion / prevQuestion move between groups (tab / shift+tab).
func nextQuestion(f setupFocus) setupFocus { return stepQuestion(f, 1) }
func prevQuestion(f setupFocus) setupFocus { return stepQuestion(f, -1) }

func stepQuestion(f setupFocus, dir int) setupFocus {
	q := questionRows()
	cur := 0
	for i, s := range q {
		if s <= f {
			cur = i
		}
	}
	return q[(cur+dir+len(q))%len(q)]
}

// nextInGroup / prevInGroup move among a group's rows (↑↓); they stop at
// the group's edges (the mock's ↑↓ never leaves a group).
func nextInGroup(f setupFocus) setupFocus {
	if f+1 < setupRowCount && rowOf(f+1).group == rowOf(f).group {
		return f + 1
	}
	return f
}

func prevInGroup(f setupFocus) setupFocus {
	if f > 0 && rowOf(f-1).group == rowOf(f).group {
		return f - 1
	}
	return f
}

// nextRow is enter's move: the next row in the table; the last row saves.
func nextRow(f setupFocus) (setupFocus, bool) {
	if f+1 >= setupRowCount {
		return f, true
	}
	return f + 1, false
}
```

**File:** `modes/tty/setup.go` (GREEN) — `setupState` gains `toneMode string`, `muted map[string]bool`,
`classes []ToneClass`, `cast CastView`, `overrides map[string]bool`, `armed string` (the voice a first `p`
offered to download); `openSetup` seeds them from `d.cfg` (`Tones`, `ToneClasses` clamped to `ToneClassCount`,
`Cast` — by `maps.Clone`, with `Names` **always non-nil**) and takes the voice snapshot `d.voiceList = d.voices()`.
The old `setupQuestions` constant goes; `focusAlert` becomes the two EVENTS rows. `handleSetupKey` is the
keyboard rule:

```go
func (d Dashboard) handleSetupKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	d.setupGen++
	d.setupNote = SetupNoteMsg{} // a note belongs to the row that raised it
	switch key.String() {
	case "esc":
		d = d.close()
		d.setup = setupState{}
		return d, nil
	case "tab":
		d.setup.focus, d.setup.err = nextQuestion(d.setup.focus), ""
		return d.scrollSetupToFocus(), nil
	case "shift+tab":
		d.setup.focus, d.setup.err = prevQuestion(d.setup.focus), ""
		return d.scrollSetupToFocus(), nil
	}
	switch rowOf(d.setup.focus).group {
	case groupLocation:
		return d.setupLocationKey(key)
	case groupKey:
		return d.setupKeyKey(key)
	case groupEvents:
		return d.setupEventsKey(key)
	case groupTones:
		return d.setupTonesKey(key)
	}
	return d.setupCastKey(key)
}

// setupSave is the one owner of the save rules the last row reaches
// through nextRow: the typed FIRMS key travels; with no location chosen or
// kept the form bounces to the location row with the reason (today's rule,
// setup.go:161-169).
func (d Dashboard) setupSave() (tea.Model, tea.Cmd) {
	if d.setup.ref == nil {
		if d.setup.ref = d.currentDefault(); d.setup.ref == nil {
			d.setup.focus, d.setup.err = focusLocation, "type a city or ZIP first"
			return d.scrollSetupToFocus(), nil
		}
	}
	return d, d.setupFinishCmd(strings.TrimSpace(d.setup.key))
}
```

`setupLocationKey`/`setupKeyKey` keep their bodies; where they moved the focus by `(focus+1)%setupQuestions`
they call `nextRow` and, on `save`, `d.setupSave()`; in DATA, `↑↓` pick a location hint while the hint list is
open and are otherwise inert (the location and the key are one-row groups — `tab` moves between them; the rule
as built). `setupEventsKey`: `↑↓` move between the two radios, `space` selects the focused one
(`filtered = focus == focusAlertWithin`), a digit selects *Within* **and** types into `radiusMi` (today's rule,
`setup.go:176-180`, kept — pinned in the EVENTS test), backspace edits on the *Within* row, `enter` = `nextRow`. The body is built **once** per change and cached on a pointer the model shares
(`d.setupMemo *setupBodyMemo`, keyed by `setupGen`, `width`, `height` — the modal memo's pattern):

```go
// setupBody is the window's body, built once per change: the lines, the
// rows each line belongs to, whether two columns were used, and the width.
type setupBody struct {
	lines  []string
	rows   [][]setupFocus // per line: the focus ids drawn on it (a paired class row has two)
	twoCol bool
	width  int
}

type setupBodyMemo struct {
	mu            sync.Mutex
	gen           uint64
	width, height int
	body          setupBody
	ok            bool
}

// setupBody returns the cached body or builds it. The memo pointer is
// allocated in NewDashboard beside memo/mmemo; a model built without it
// (a test literal) simply rebuilds.
func (d Dashboard) setupBody(o render.Opts) setupBody {
	m := d.setupMemo
	if m == nil {
		m = &setupBodyMemo{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ok && m.gen == d.setupGen && m.width == d.width && m.height == d.height {
		return m.body
	}
	left, right := d.setupColumns(o)
	lw, rw := widest(texts(left)), widest(texts(right))
	body := setupBody{width: 78}
	if w := twoColumnsWidth(lw, rw, panelChromeFor(1+max(len(left), len(right)), d.modalMax())); w <= o.Width {
		body.twoCol, body.width = true, w
	}
	body.lines, body.rows = compose(left, right, body.twoCol, lw)
	m.gen, m.width, m.height, m.body, m.ok = d.setupGen, d.width, d.height, body, true
	return body
}

// compose lays the columns out — side by side (row i of either column is
// row i) or stacked — and records the focus ids per line; every line is
// trimmed to its column so the wrapped body never drifts from this index.
func compose(left, right []setupLine, twoCol bool, lw int) (lines []string, rows [][]setupFocus) {
	if !twoCol {
		for _, l := range append(left, right...) {
			lines, rows = append(lines, l.text), append(rows, l.focus)
		}
		return lines, rows
	}
	n := max(len(left), len(right))
	lines = sideBySide(texts(pad(left, n)), texts(pad(right, n)), lw)
	for i := 0; i < n; i++ {
		var f []setupFocus
		if i < len(left) {
			f = append(f, left[i].focus...)
		}
		if i < len(right) {
			f = append(f, right[i].focus...)
		}
		rows = append(rows, f)
	}
	return lines, rows
}

// setupLine is a body line and the rows drawn on it (nil = none).
type setupLine struct {
	text  string
	focus []setupFocus
}

func group(title string) []setupLine {
	return []setupLine{{"", nil}, {setupGroup(title), nil}, {"", nil}}
}

func pad(lines []setupLine, n int) []setupLine {
	for i := len(lines); i < n; i++ { // counter form (P10-02)
		lines = append(lines, setupLine{})
	}
	return lines
}

func texts(lines []setupLine) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.text
	}
	return out
}

// setupColumns are the two column groups, each line tagged with its rows.
func (d Dashboard) setupColumns(o render.Opts) (left, right []setupLine) {
	left = append(left, group("DATA")...)
	left = append(left, d.setupLocationLines(o)...)
	left = append(left, d.setupKeyLines(o)...)
	left = append(left, group("ALERTS - EVENTS")...)
	left = append(left, d.setupEventsLines(o)...)
	left = append(left, group("ALERTS - TONE  ( [M] toggles )")...)
	left = append(left, d.setupTonesLines(o)...)
	right = append(right, group("WATCHPOST RADIO - CORRESPONDENTS")...)
	right = append(right, d.setupCastLines(o)...)
	return left, right
}

// setupLines is the Setup window body (the HUM LEAD's mock, 2026-08-29): the
// air under the title, then the composed columns. The chips are the footer
// (floatModalFooter), pinned under the scroll window.
func (d Dashboard) setupLines(o render.Opts) []string {
	return append([]string{""}, d.setupBody(o).lines...)
}

// setupWidth is the window's width (modalWidth's case for modalSetup).
func (d Dashboard) setupWidth() int { return d.setupBody(d.opts()).width }

// setupFocusRow is the body line index of the focused row (1 = the air line).
func (d Dashboard) setupFocusRow() int {
	body := d.setupBody(d.opts())
	for i, rows := range body.rows {
		if slices.Contains(rows, d.setup.focus) {
			return 1 + i
		}
	}
	return 1
}

// scrollSetupToFocus keeps the focused row — and the note line under it,
// when one exists — inside the modal window (RS-19).
func (d Dashboard) scrollSetupToFocus() Dashboard {
	row, maxl := d.setupFocusRow(), d.modalMax()
	last := row
	if d.noteUnderFocus() {
		last = row + 1
	}
	switch {
	case row < d.modalScroll:
		d.modalScroll = row
	case last >= d.modalScroll+maxl:
		d.modalScroll = last - maxl + 1
	}
	return d
}

// setupChipRows is the footer: the mock's four chips; ←→ Voice and p Preview
// join while a picker is focused (MVS-D-27; p only when previews exist).
func (d Dashboard) setupChipRows(o render.Opts) []string {
	action := "Next"
	if d.setup.focus == setupRowCount-1 {
		action = "Save"
	}
	segs := []string{o.KeyCap("tab") + "  Next question", o.KeyCap("enter") + "  " + action, o.KeyCap("↑↓") + "  Pick"}
	if rowOf(d.setup.focus).kind == rowPicker {
		segs = append(segs, o.KeyCap("←→")+"  Voice")
		if d.cfg.PreviewVoice != nil {
			segs = append(segs, o.KeyCap("p")+"  Preview")
		}
	}
	if d.setup.focus == focusKey {
		segs = append(segs, o.KeyCap("ctrl+r")+"  Reveal key") // today's chip, on the row it serves
	}
	segs = append(segs, o.KeyCap("esc")+"  Cancel")
	inner := min(o.Width, d.modalWidth()) - 7 - 2
	rows := []string{""}
	for _, row := range render.WrapSegments(segs, inner, "    ") {
		rows = append(rows, "  "+row)
	}
	return rows
}
```

The three existing `*Lines` builders return `[]setupLine` with their rows tagged (`{text, []setupFocus{focusLocation}}`
on the question line, `nil` on support lines) and take the mark from `st.mark(f, o)` / `st.markWide(f, o)` (the
`›` slot: `"› "`/`"  "` before a checkbox, `"›  "`/`"   "` before a radio or a picker label — the mock's spacing
— through `o.Glyphs().Pointer` so --ascii draws `>`). The DATA rows are re-worded to the mock (`Default
location: │…`, `City Name, "City", or Zip`, `NASA FIRMS key: stored (…d9a6) — ✔ working`, `Paste new key to
replace — empty keeps`, `Key:`, `ctrl+r  Reveal key`; `✔`/`✘`/`⚠` through `Glyphs.OK/Fail/Alert`); the EVENTS
rows to `○ All locations (Default)` / `○ Within │    mi of Default Location` as two focusable rows. `view.go`:
`case modalSetup: return d.floatModalFooter(o, d.modalWidth(), "Setup / Configs", d.setupLines(o), d.setupChipRows(o))`
— `floatModalFooter` is `floatModal` with the footer rows drawn under the scroll window inside the frame (one
owner in `view.go`; Help and Status may adopt it later); `modalLines()` (the scroll bound) counts the body only;
`modalWidth`'s `case modalSetup: return d.setupWidth()`. **Who bumps `setupGen`** (the memo is only as good as its writers — every site that changes what the body reads):
`handleSetupKey` (every key), `openSetup`, the `SetupNoteMsg` case, `handleResolved` (`modal_location.go:73-74`,
which writes `d.setup.ref/focus/err` after a resolve), `applyCommitted` (`dashboard.go:759-762`, the "setup
failed: …" error), `applySnapshot`/`applyRecent` (the DATA rows read `firmsHealth`/`currentDefault` from the
snapshot) and the resize path — `tea.WindowSizeMsg` is handled inline in `dispatch` (`dashboard.go:424-426`); that
case gains `d.setupGen++`, and the test helper `resized()` is `func (d Dashboard) resized() Dashboard { d.setupGen++; return d }`
in `setup_helpers_test.go`. `modalLines()` (`view.go:107-125`, the scroll bound) gains `case modalSetup: raw = d.setupLines(d.opts())`
— today it falls through to `helpLines`, so `nav.go:94`'s clamp would bound Setup by Help's length.
`flipMute(mode string) string` (Task 4.7) lives in `setup_tones.go`. **Verify:** `go test ./modes/tty -run 'SetupRows|SetupIsTwoColumns|EverySetupRow|Setup' -count=1`

---

### Task 4.5 — the ALERTS - TONE group (FR-11; MVS-D-26/28)

**File:** `modes/tty/setup_tones_test.go` (RED)

```go
func testToneClasses() []ToneClass {
	return []ToneClass{{"disaster", "Significant Quakes & Disasters"}, {"warning", "Warnings"}, {"watch", "Watches"},
		{"advisory", "Advisories"}, {"statement", "Special Statements"}, {"storm", "Maritime"}}
}

func TestSetupTonesRowsAndKeys(t *testing.T) {
	d := newTestDashboard(t)
	d.cfg.ToneClasses = testToneClasses()
	d.cfg.Tones = Tones{Mode: "mute", Muted: []string{"warning"}}
	d = d.openSetupAt(focusToneOn)
	got := plainLines(texts(d.setupTonesLines(d.opts())))
	want := []string{ // the mock's slots: › only on the focused row; radio rows "›  ○", checkbox rows "› [ ]"; class cells 18 wide
		"  ›  ○ All Tones On (Default)",
		"     ● Mute:",
		"       [ ] Significant Quakes & Disasters",
		"       [x] Warnings      [ ] Watches",
		"       [ ] Advisories    [ ] Special Statements",
		"       [ ] Maritime",
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("line %d:\n got %q\nwant %q", i, got[i], w)
		}
	}
	d = d.press(t, tea.KeyDown) // Mute:
	d = d.press(t, tea.KeyDown) // the first checkbox
	d = d.press(t, ' ')
	if !d.setup.muted["disaster"] {
		t.Fatalf("space toggles the focused class: %v", d.setup.muted)
	}
	if got := plainLines(texts(d.setupTonesLines(d.opts()))); got[2] != "     › [x] Significant Quakes & Disasters" {
		t.Fatalf("the focused checkbox carries the mark: %q", got[2])
	}
	d = d.press(t, tea.KeyDown) // Warnings
	d = d.press(t, tea.KeyRight)
	if d.setup.focus != focusToneClass2 {
		t.Fatalf("→ moves to the second cell of a paired row (Watches): %v", d.setup.focus)
	}
	if got := plainLines(texts(d.setupTonesLines(d.opts()))); got[3] != "       [x] Warnings    › [ ] Watches" {
		t.Fatalf("the second cell's mark: %q", got[3])
	}
	for i := 0; i < 4; i++ { // Watches → Warnings → Significant… → Mute: → All Tones On
		d = d.press(t, tea.KeyUp)
	}
	d = d.press(t, ' ')
	if d.setup.toneMode != "" {
		t.Fatal("space on All Tones On clears the mute mode (the set is kept)")
	}
}

func TestMuteWithNothingTickedSaysSo(t *testing.T) {
	d := newTestDashboard(t)
	d.cfg.ToneClasses = testToneClasses()
	d.cfg.Tones = Tones{Mode: "mute"}
	d = d.openSetupAt(focusToneMute)
	if got := plainLines(texts(d.setupTonesLines(d.opts()))); got[2] != "       nothing ticked = every class" {
		t.Fatalf("OP-3 line: %q", got[2])
	}
}
```

**File:** `modes/tty/setup_tones.go` (GREEN)

```go
package tty

// setup_tones.go — the ALERTS - TONE group of the Setup window (the HUM LEAD's
// mock, MVS-D-26/28): a radio "All Tones On (Default)" / "Mute:" and a
// checkbox per tone class; [M] on the dashboard flips the same mode.

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// ToneClass is one mutable alert class as the app names it (the key is the
// config's word; the label the mock's). The app supplies the list — the
// registry lives in the radio domain, which this package may not import.
type ToneClass struct {
	Key, Label string
}

// Tones is the tone mute as configured (mirrors config.Tones without the import).
type Tones struct {
	Mode  string   // "" all tones on | "mute"
	Muted []string // class keys; empty with Mode "mute" = every class
}

// toneCellWidth is the mock's column for a class cell ("› [ ] Warnings" + 4).
const toneCellWidth = 18

// toneRows lays the classes as the mock does: one on the first row, then
// two per row, in the app's order.
func toneRows(classes []ToneClass) [][]ToneClass {
	if len(classes) == 0 {
		return nil
	}
	rows := [][]ToneClass{classes[:1]}
	for i := 1; i < len(classes); i += 2 {
		rows = append(rows, classes[i:min(i+2, len(classes))])
	}
	return rows
}

// setupTonesLines renders the group; the focused row carries the › mark; a
// paired row carries both cells' ids so the scroll finds either.
func (d Dashboard) setupTonesLines(o render.Opts) []setupLine {
	st := d.setup
	lines := []setupLine{
		{"  " + st.markWide(focusToneOn, o) + radioMark(st.toneMode == "", o.ASCII) + " All Tones On (Default)", []setupFocus{focusToneOn}},
		{"  " + st.markWide(focusToneMute, o) + radioMark(st.toneMode == "mute", o.ASCII) + " Mute:", []setupFocus{focusToneMute}},
	}
	if st.toneMode == "mute" && len(st.tones().Muted) == 0 { // OP-3: the rule, said where it applies (untinted: today's support-line convention)
		lines = append(lines, setupLine{"       nothing ticked = every class", nil})
	}
	i := 0
	for _, row := range toneRows(st.classes) {
		var cells []string
		var ids []setupFocus
		for _, c := range row {
			f := focusToneClass0 + setupFocus(i)
			cells = append(cells, render.PadTo(st.mark(f, o)+checkbox(st.muted[c.Key])+" "+c.Label, toneCellWidth))
			ids = append(ids, f)
			i++
		}
		lines = append(lines, setupLine{"     " + strings.TrimRight(strings.Join(cells, ""), " "), ids})
	}
	return lines
}

// checkbox is "[x]" / "[ ]" — a glyph-free mark, the same under --ascii (FR-13).
func checkbox(on bool) string {
	if on {
		return "[x]"
	}
	return "[ ]"
}

// setupTonesKey owns keys while a tone row is focused: ↑↓ move among the
// mode radios and the class rows, ←→ move between the two cells of a
// paired row, space selects/toggles, enter moves on.
func (d Dashboard) setupTonesKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	f := d.setup.focus
	switch key.String() {
	case "up":
		d.setup.focus = prevInGroup(f)
	case "down":
		d.setup.focus = nextInGroup(f)
	case "left", "right":
		if n, ok := d.setup.pairedCell(f, key.String() == "right"); ok {
			d.setup.focus = n
		}
	case "space": // bubbletea v2 names the space key "space" (its Text " " is not the String); a `" "` case never fires
		switch f {
		case focusToneOn:
			d.setup.toneMode = ""
		case focusToneMute:
			d.setup.toneMode = "mute"
		default:
			if c, ok := d.setup.classAt(f); ok {
				d.setup.muted[c.Key] = !d.setup.muted[c.Key]
				d.setup.toneMode = "mute" // ticking a class means Mute
			}
		}
	case "enter":
		next, save := nextRow(f)
		if save {
			return d.setupSave()
		}
		d.setup.focus = next
	}
	return d.scrollSetupToFocus(), nil
}

// pairedCell is the other cell of a two-class row (classes 1↔2, 3↔4).
func (st setupState) pairedCell(f setupFocus, right bool) (setupFocus, bool) {
	i := int(f - focusToneClass0)
	if i < 1 || i >= len(st.classes) {
		return f, false
	}
	if right && i%2 == 1 && i+1 < len(st.classes) {
		return f + 1, true
	}
	if !right && i%2 == 0 {
		return f - 1, true
	}
	return f, false
}

// classAt is the class a checkbox row stands for.
func (st setupState) classAt(f setupFocus) (ToneClass, bool) {
	i := int(f - focusToneClass0)
	if i < 0 || i >= len(st.classes) {
		return ToneClass{}, false
	}
	return st.classes[i], true
}

// tones is the group's value as Setup saves it (sorted keys; empty under
// Mode "mute" means every class — cast.Muted's rule).
func (st setupState) tones() Tones {
	var muted []string
	for k, on := range st.muted {
		if on {
			muted = append(muted, k)
		}
	}
	slices.Sort(muted)
	return Tones{Mode: st.toneMode, Muted: muted}
}
```

`radioMark(selected, ascii)` is `setup.go:365`'s existing mark (`●`/`○`, `*`/`o` under --ascii) — one owner for
the three groups' radios; not lifted. `st.mark(f, o)` returns `o.Glyphs().Pointer + " "` / `"  "`;
`st.markWide(f, o)` the same with two spaces (the mock's radio/picker spacing).
**Verify:** `go test ./modes/tty -run 'SetupTones|MuteWithNothing' -count=1`

---

### Task 4.6 — the WATCHPOST RADIO - CORRESPONDENTS group (FR-4; MVS-D-25/27; the pickers; the preview; the notes)

**File:** `modes/tty/setup_cast_test.go` (RED)

```go
func TestSetupCastRowsFollowTheMock(t *testing.T) {
	d := newTestDashboard(t)
	d.cfg.Voices = func() []string { return []string{"System Voice", "Rishi"} } // a voice list AND a preview hook: either missing draws a note under the focused row
	d.cfg.PreviewVoice = func(string) {}
	d.cfg.Cast = CastView{Mode: "cast", Names: map[string]string{castKeyRoot: "System Voice", castKeyAlerts: "Rishi", castKeyFire: "System Voice"}}
	d = d.openSetupAt(focusCastAlerts)
	got := plainLines(texts(d.setupCastLines(d.opts())))
	want := []string{
		"     ○ Single Voice │ System Voice     │ ▾ │ (Default)",
		"     ● Correspondent Cast:",
		"     ›  Alerts / Takeovers - │ Rishi            │ ▾ │",
		"        All Reports        - │ System Voice     │ ▾ │",
		"",
		"        ...except when there's a:",
		"       [ ] Location Report - │ System Voice     │ ▾ │",
		"       [ ] Maritime Report  - │ System Voice     │ ▾ │",
		"       [x] Fire/Hotspots    - │ System Voice     │ ▾ │",
		"       [ ] Seismic          - │ System Voice     │ ▾ │",
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("line %d:\n got %q\nwant %q", i, got[i], w)
		}
	}
}

func TestSetupPickerCyclesFromTheDisplayedVoiceAndPreviews(t *testing.T) {
	d := newTestDashboard(t) // cfg.Cast unset: openSetup seeds every map non-nil and the root from the first voice
	d.cfg.Voices = func() []string { return []string{"System Voice", "Rishi", "Samantha"} }
	var previewed string
	d.cfg.PreviewVoice = func(name string) { previewed = name }
	d = d.openSetupAt(focusCastAlerts)
	d = d.press(t, tea.KeyRight) // the picker shows the inherited "System Voice"; → moves to the entry after it
	if d.setup.cast.Names[castKeyAlerts] != "Rishi" || d.setup.cast.Mode != "cast" {
		t.Fatalf("→ cycles from the displayed voice and turns the cast on: %v mode=%q", d.setup.cast.Names, d.setup.cast.Mode)
	}
	d = d.press(t, 'p')
	if previewed != "Rishi" {
		t.Fatalf("p previews the focused picker's voice (MVS-D-27): %q", previewed)
	}
	chips := plainLines(d.setupChipRows(d.opts()))
	if !containsLine(chips, "[p]  Preview") || !containsLine(chips, "[←→]  Voice") { // KeyCap with colour off
		t.Fatalf("the picker chips show while a picker is focused: %v", chips)
	}
	d = d.press(t, tea.KeyTab) // the DATA question: no picker
	if containsLine(plainLines(d.setupChipRows(d.opts())), "[p]  Preview") {
		t.Fatal("the chip hides off a picker")
	}
}

func TestSetupNotesSitUnderTheFocusedRow(t *testing.T) {
	d := newTestDashboard(t)
	d.cfg.Voices = func() []string { return []string{"System Voice", "Ryan"} }
	d.cfg.VoiceInstalled = func(name string) bool { return name != "Ryan" }
	d.cfg.PreviewVoice = func(string) {}
	d = d.openSetupAt(focusCastAlerts)
	d = d.press(t, tea.KeyRight)
	got := plainLines(texts(d.setupCastLines(d.opts())))
	if got[3] != "        Ryan not installed — downloads on Save (~63 MB);" || got[4] != "        the alert tone never waits for it" { // wrapped at noteWidth, never trimmed
		t.Fatalf("the note sits under the focused picker, wrapped:\n%s", strings.Join(got, "\n"))
	}
	d = d.press(t, 'p') // a first p on an uninstalled voice asks (FR-4)
	if got := plainLines(texts(d.setupCastLines(d.opts()))); got[3] != "        press p again to download Ryan (~63 MB)" {
		t.Fatalf("the preview asks once before a download: %q", got[3])
	}
	d = d.press(t, tea.KeyDown) // moving the focus clears the note
	if got := plainLines(texts(d.setupCastLines(d.opts()))); strings.Contains(got[3], "press p") {
		t.Fatal("a note belongs to the row that raised it")
	}
}

func TestSetupWithoutAudioOrVoicesSaysSo(t *testing.T) {
	d := newTestDashboard(t) // PreviewVoice nil = no audio device; Voices nil = none found
	d = d.openSetupAt(focusCastAlerts)
	got := plainLines(texts(d.setupCastLines(d.opts())))
	if got[3] != "        no voices found — assignments still save" {
		t.Fatalf("the group says so: %q", got[3])
	}
	d.cfg.Voices = func() []string { return []string{"System Voice"} }
	d = d.openSetupAt(focusCastAlerts)
	if got := plainLines(texts(d.setupCastLines(d.opts()))); got[3] != "        no audio — previews off; assignments still save" {
		t.Fatalf("no audio: %q", got[3])
	}
}

func TestSetupSaveWritesTheCastAndTones(t *testing.T) {
	d := newTestDashboard(t)
	d.cfg.Voices = func() []string { return []string{"System Voice", "Rishi"} }
	d.cfg.ToneClasses = testToneClasses()
	var savedCast CastView
	var savedTones Tones
	d.cfg.SetCast = func(v CastView) error { savedCast = v; return nil }
	d.cfg.SetTones = func(t Tones) error { savedTones = t; return nil }
	d.cfg.Setup = func(snapshot.LocationRef, string) error { return nil }
	d = d.openSetupAt(focusToneMute)
	d.setup.ref = &snapshot.LocationRef{Label: "Oceanside, CA"}
	d = d.press(t, ' ')          // Mute:
	d = d.press(t, tea.KeyTab)   // the correspondents
	d = d.press(t, tea.KeyDown)  // Correspondent Cast
	d = d.press(t, ' ')          // on
	d = d.press(t, tea.KeyDown)  // Alerts / Takeovers
	d = d.press(t, tea.KeyRight) // Rishi (from the seeded root "System Voice")
	for i := 0; i < 6; i++ {     // enter through the remaining rows; the last saves (11 keypresses from V — M2)
		d = d.press(t, tea.KeyEnter)
	}
	if savedCast.Mode != "cast" || savedCast.Names[castKeyAlerts] != "Rishi" || savedCast.Names[castKeyFire] != "" || savedTones.Mode != "mute" {
		t.Fatalf("saved cast %+v tones %+v", savedCast, savedTones)
	}
}
```

**File:** `modes/tty/setup_cast.go` (GREEN)

```go
package tty

// setup_cast.go — the WATCHPOST RADIO - CORRESPONDENTS group of the Setup
// window (the HUM LEAD's mock, MVS-D-3/10/25/27): Single Voice (one picker,
// the root) or Correspondent Cast (the Alerts / Takeovers and All Reports
// pickers, and four optional per-report overrides). A picker is
// "│ <name> │ ▾ │": ←/→ cycle the host's voices, p previews (Audition,
// ducked by the Station Director). Switching to Single Voice keeps the
// cast's names (MVS-D-25).

import (
	"maps"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
)

// The role keys, as cast.Role.Key() names them. The app pins these to the
// radio domain's registry (app/cast_test.go: TestCastViewKeysMatchTheRegistry)
// — this package may not import it.
const (
	castKeyRoot     = "voice"
	castKeyAlerts   = "alerts"
	castKeyStandard = "standard"
	castKeyWeather  = "weather"
	castKeyMaritime = "maritime"
	castKeyFire     = "fire"
	castKeySeismic  = "seismic"
)

// CastView is the cast as Setup edits it and the app persists it: the mode
// and the voice NAME chosen per role on THIS platform ("" = inherits). An
// override is "on" exactly when its name is set — nothing else to persist.
// Names are plain, one line (NFR-6) — the app supplies them so.
type CastView struct {
	Mode  string            // "" single | "cast"
	Names map[string]string // by role key; never nil once Setup holds it
}

// castLabel is the row label as the mock draws it (padded to the mock's column).
func castLabel(key string) string {
	return map[string]string{castKeyAlerts: "Alerts / Takeovers", castKeyStandard: "All Reports       ",
		castKeyWeather: "Location Report", castKeyMaritime: "Maritime Report ", castKeyFire: "Fire/Hotspots   ", castKeySeismic: "Seismic         "}[key]
}

// picker is the "│ <name> │ ▾ │" control, the name padded to 16 cells.
func picker(g render.Glyphs, name string) string {
	return g.Rail + " " + render.PadTo(render.PlainLine(name), 16) + " " + g.Rail + " " + g.Down + " " + g.Rail
}

// isOverride: the four per-report rows.
func isOverride(key string) bool {
	return key == castKeyWeather || key == castKeyMaritime || key == castKeyFire || key == castKeySeismic
}

// shown is the name a picker displays: the role's own, else the root's.
func (st setupState) shown(key string) string {
	if n := st.cast.Names[key]; n != "" {
		return n
	}
	return st.cast.Names[castKeyRoot]
}

// setupCastLines renders the group; the focused row carries the › mark; the
// row's note (castNote) is drawn directly under the focused row.
func (d Dashboard) setupCastLines(o render.Opts) []setupLine {
	st, g := d.setup, o.Glyphs()
	var lines []setupLine
	add := func(text string, f setupFocus) {
		lines = append(lines, setupLine{text, []setupFocus{f}})
		if f == st.focus {
			for _, l := range render.WrapLines(d.castNote(), noteWidth) { // "" wraps to nothing
				lines = append(lines, setupLine{"        " + l, nil})
			}
		}
	}
	add("  "+st.markWide(focusCastSingle, o)+radioMark(st.cast.Mode != "cast", o.ASCII)+" Single Voice "+picker(g, st.shown(castKeyRoot))+" (Default)", focusCastSingle)
	add("  "+st.markWide(focusCastOn, o)+radioMark(st.cast.Mode == "cast", o.ASCII)+" Correspondent Cast:", focusCastOn)
	for _, f := range []setupFocus{focusCastAlerts, focusCastStandard} {
		key := rowOf(f).key
		add("     "+st.markWide(f, o)+castLabel(key)+" - "+picker(g, st.shown(key)), f)
	}
	lines = append(lines, setupLine{"", nil}, setupLine{"        ...except when there's a:", nil})
	for _, f := range []setupFocus{focusCastWeather, focusCastMaritime, focusCastFire, focusCastSeismic} {
		key := rowOf(f).key
		add("     "+st.mark(f, o)+checkbox(st.overrides[key])+" "+castLabel(key)+" - "+picker(g, st.shown(key)), f)
	}
	return lines
}

// noteWidth wraps a note under the picker rows' width (52 cells minus the
// 8-cell inset): a note never widens the column, so focusing a picker never
// flips the two-column layout.
const noteWidth = 44

// castNote is the text under the focused row: the deck's word about the
// voice this picker shows (installing, auditioning, a failure — SetupNoteMsg
// carries the voice name and is drawn only where that name is); the "press
// p again" offer; "not installed" for the focused picker's voice; the
// no-voices / no-audio notes; "inherits" for a group picker left empty.
// Empty when nothing needs saying.
func (d Dashboard) castNote() string {
	st := d.setup
	row := rowOf(st.focus)
	if row.kind != rowPicker {
		return ""
	}
	name := st.shown(row.key)
	if d.setupNote.Text != "" && d.setupNote.Voice == name {
		return d.setupNote.Text
	}
	if len(d.voiceList) == 0 {
		return "no voices found — assignments still save"
	}
	switch {
	case st.armed != "" && st.armed == name:
		return "press p again to download " + name + " (~63 MB)"
	case d.cfg.VoiceInstalled != nil && name != "" && !d.cfg.VoiceInstalled(name):
		return name + " not installed — downloads on Save (~63 MB); the alert tone never waits for it"
	case d.cfg.PreviewVoice == nil:
		return "no audio — previews off; assignments still save"
	case st.cast.Names[row.key] == "" && row.key != castKeyRoot:
		return "inherits — the voice above reads this until you pick one"
	}
	return ""
}

// noteUnderFocus reports whether a note line follows the focused row.
func (d Dashboard) noteUnderFocus() bool {
	return rowOf(d.setup.focus).group == groupCast && d.castNote() != ""
}

// setupCastKey owns keys while a cast row is focused: ↑↓ move, space selects
// a mode or toggles an override, enter moves on (the last row saves); the
// picker keys are setupPickerKey's.
func (d Dashboard) setupCastKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	f := d.setup.focus
	row := rowOf(f)
	switch key.String() {
	case "up":
		d.setup.focus, d.setup.armed = prevInGroup(f), ""
	case "down":
		d.setup.focus, d.setup.armed = nextInGroup(f), ""
	case "left", "right", "p":
		if row.kind == rowPicker {
			return d.setupPickerKey(key, row)
		}
	case "space":
		switch {
		case f == focusCastSingle:
			d.setup.cast.Mode = ""
		case f == focusCastOn:
			d.setup.cast.Mode = "cast"
		case isOverride(row.key):
			d.setup.overrides[row.key] = !d.setup.overrides[row.key]
		}
	case "enter":
		next, save := nextRow(f)
		if save {
			return d.setupSave()
		}
		d.setup.focus, d.setup.armed = next, ""
	}
	return d.scrollSetupToFocus(), nil
}

// setupPickerKey: ←→ cycle the focused picker from the voice it DISPLAYS
// (an inherited name included); p previews — a first p on a voice that is
// not installed only offers the download (FR-4: "asks once"), the second
// proceeds; a group picker's list starts with "" (inherit).
func (d Dashboard) setupPickerKey(key tea.KeyPressMsg, row setupRow) (tea.Model, tea.Cmd) {
	st := &d.setup
	switch key.String() {
	case "left", "right":
		list := d.voiceList
		if row.key != castKeyRoot {
			list = append([]string{""}, list...) // "" = inherit; shown as the voice above
		}
		st.cast.Names[row.key] = cycleVoice(list, st.shown(row.key), key.String() == "right")
		if isOverride(row.key) {
			st.overrides[row.key] = st.cast.Names[row.key] != "" // choosing a voice turns the override on; inheriting turns it off
		}
		if row.key != castKeyRoot {
			st.cast.Mode = "cast" // choosing a correspondent is choosing the cast: a pick under Single Voice would be silent (a11y round 3)
		}
		st.armed = ""
	case "p":
		if d.cfg.PreviewVoice == nil {
			break
		}
		name := st.shown(row.key)
		if d.cfg.VoiceInstalled != nil && !d.cfg.VoiceInstalled(name) && st.armed != name {
			st.armed = name // asked once; the next p proceeds
			break
		}
		st.armed = ""
		preview := d.cfg.PreviewVoice // the deck sends the "preparing…" note at once (one owner, P2 Task 2.7)
		return d.scrollSetupToFocus(), func() tea.Msg { preview(name); return nil }
	}
	return d.scrollSetupToFocus(), nil
}

// cycleVoice steps through list from the entry the picker displays (its own
// name, or the inherited one it shows); a name not in the list starts from
// the first entry. The "" entry (inherit) sits at index 0 of a group list
// and displays as the inherited name, so the two never collide on screen
// when castNote says "inherits".
func cycleVoice(list []string, shown string, forward bool) string {
	if len(list) == 0 {
		return shown
	}
	i := -1
	for j, v := range list {
		if v == shown && v != "" {
			i = j
			break
		}
	}
	switch {
	case i < 0:
		return list[0]
	case forward:
		return list[(i+1)%len(list)]
	}
	return list[(i+len(list)-1)%len(list)]
}

// castForSave is the cast as Setup hands it to the app: an override that is
// unticked saves an empty name (inherits), whatever its picker shows.
func (st setupState) castForSave() CastView {
	out := CastView{Mode: st.cast.Mode, Names: maps.Clone(st.cast.Names)}
	for _, key := range []string{castKeyWeather, castKeyMaritime, castKeyFire, castKeySeismic} {
		if !st.overrides[key] {
			out.Names[key] = ""
		}
	}
	return out
}
```

`setupState.cast` is seeded from `d.cfg.Cast` in `openSetup` **by copy** (`maps.Clone`; a nil `Names` becomes
an empty map; **an empty root name is seeded from `voiceList[0]`** so every picker shows a name — Setup edits
must not write the model's config until Save) and `setupState.overrides` from it (`overrides[key] = Names[key] != ""`).
`SetupNoteMsg{Voice, Text string}`: the deck names the voice its word is about; the model keeps it as
`d.setupNote` (both fields PlainLine'd) and `castNote` draws it only under the picker showing that voice — a note
that arrives while the focus is elsewhere is not lost: `[S]`'s CAST `State` carries the same install/audition
word. The `SetupNoteMsg` case also calls `scrollSetupToFocus()` so a note landing under the window's last row
brings it into view. `picker()` cuts the name to 16 cells (`render.TruncateCells`) so `▾` and `(Default)` never
leave the column. `Glyphs.Down` (`"▾"` / `"v"` under --ascii), `Glyphs.Rail` (`"│"` /
`"|"`) and `Glyphs.Arrow` (`"→"` / `"->"`) are added to `platform/render/units.go` beside `Dash`/`Dot`; the
picker and the DATA input rails use `g.Rail` (FR-13). The Linux picker lists Piper voices by their catalogue
**names** (`deck.Voices` already returns names); `saveCast` maps a name to its key (Task 4.7). The deck sends
`SetupNoteMsg{""}` when an audition ends (P2 Task 2.7), so a "preparing…" note never outlives it. **Verify:**
`go test ./modes/tty -run 'SetupCast|SetupPicker|SetupNotes|SetupWithout|SetupSave' -count=1`

---

### Task 4.7 — the hooks: `SetCast`, `SetTones`, `ToneClasses`, `VoiceInstalled`; `[M]` flips the mode (MVS-D-26)

**File:** `modes/tty/dashboard.go` — `Config` gains (and loses `SetVoice`, `Voice`, `TickerMuted`, `MuteTicker`):

```go
	Cast           CastView               // the cast as configured (seeds Setup)
	Tones          Tones                  // the tone mute (seeds Setup; [M] flips Mode)
	ToneClasses    []ToneClass            // the mutable classes, in the app's order (one owner: the radio domain)
	SetCast        func(CastView) error   // Setup's save: the app maps it onto config.Radio and re-casts the deck
	SetTones       func(Tones) error      // Setup's save and [M]
	Voices         func() []string        // the host's voices for the pickers (kept from 0.13.0)
	VoiceInstalled func(name string) bool // nil = every listed voice is installed (macOS)
	PreviewVoice   func(name string)      // p on a picker (kept from 0.13.0; Audition, ducked); nil = no audio
```

`setupFinishCmd(key)` captures `tones, view := d.setup.tones(), d.setup.castForSave()` and the two hooks, and
calls, in order: `setup(def, key)` → `SetTones(tones)` → `SetCast(view)` → `setRadius` → `commit` (each
nil-safe; an error becomes `committedMsg{err, what: "setup"}` as today). The `[M]` handler (`ticker-mute`,
`dashboard.go:587`):

```go
	case "ticker-mute": // 0.14.0: tones only (MVS-D-26); the words always read; the ticked set is kept
		d.cfg.Tones.Mode = flipMute(d.cfg.Tones.Mode)
		d.tickerMuted = d.cfg.Tones.Mode == "mute" // the masthead's state word
		if set := d.cfg.SetTones; set != nil {
			tones := d.cfg.Tones
			return d, func() tea.Msg { return committedMsg{err: set(tones), what: "tones"} } // off the update loop
		}
```

with `func flipMute(mode string) string { if mode == "mute" { return "" }; return "mute" }` in `setup_tones.go`.
The masthead label (`body.go:85-103`) words the state truthfully in **every** form of its width ladder: form 0
`Alert Tones: On` · `Alert Tones: Muted (all)` · `Alert Tones: Muted (2 of 6)`; form 1 (the label drops) keeps
the count on the chip — `[M] Tones 2/6 muted` / `[M] Tones muted` / `[M] Tones on`; form 2 (the chip drops) leaves
the state to `[S]`'s TONES line (Task 4.8) — so the state is readable on some surface at 80 cols. The chip's
verb names what it does: `Mute Alert Tones` when on; `Unmute Alert Tones` when every class is muted; `Mute ticked
tones (2)` when a partial set is kept.

**File:** `app/cast.go` (the P1 file) gains the view mapping; `app/dashboard.go`'s `ttyConfig` fills
`Cast: castViewOf(cfg)`, `Tones: tonesView(cfg.Radio.Tones)`, `ToneClasses: toneClasses()`,
`SetCast: lp.saveCast`, `SetTones: lp.saveTones`, `VoiceInstalled: installedHook(deck)` (nil on darwin, else
`deck.Installed` over `storedName`):

```go
// castViewOf is the cast as Setup edits it — this platform's half of each pair, by name.
func castViewOf(cfg config.Config) tty.CastView {
	v := cfg.Radio.Voices
	names := map[string]string{"voice": render.PlainLine(cfg.Voice)}
	for key, rv := range map[string]config.RoleVoice{"alerts": v.Alerts, "standard": v.Standard, "weather": v.Weather, "maritime": v.Maritime, "fire": v.Fire, "seismic": v.Seismic} {
		names[key] = render.PlainLine(displayName(platformHalf(rv)))
	}
	return tty.CastView{Mode: cfg.Radio.Cast, Names: names}
}

// toneClasses is the registry's class list for Setup, in its order.
func toneClasses() []tty.ToneClass {
	out := make([]tty.ToneClass, 0, len(cast.Classes()))
	for _, c := range cast.Classes() {
		out = append(out, tty.ToneClass{Key: c.String(), Label: c.Label()})
	}
	return out
}

// saveCast writes Setup's cast onto this platform's half of the pairs (the
// other OS's half survives — FR-8) and re-casts the deck.
func (lp *livePipelines) saveCast(view tty.CastView) error {
	err := savePreference(func(cfg *config.Config) {
		cfg.Voice = view.Names["voice"]
		cfg.Radio.Cast = view.Mode
		set := func(rv *config.RoleVoice, name string) { *platformHalfPtr(rv) = storedName(name) }
		set(&cfg.Radio.Voices.Alerts, view.Names["alerts"])
		set(&cfg.Radio.Voices.Standard, view.Names["standard"])
		for key, field := range map[string]*config.RoleVoice{"weather": &cfg.Radio.Voices.Weather, "maritime": &cfg.Radio.Voices.Maritime, "fire": &cfg.Radio.Voices.Fire, "seismic": &cfg.Radio.Voices.Seismic} {
			set(field, view.Names[key]) // "" = the override is off (Setup's castForSave cleared it)
		}
	})
	if err != nil || lp.deck == nil {
		return err
	}
	return lp.deck.recast()
}

// saveTones persists the tone mute (Save mirrors ticker_muted for 0.13.0
// readers — one owner, P1) and hands the deck the new mute — no reload, no re-cast.
func (lp *livePipelines) saveTones(t tty.Tones) error {
	tones := config.Tones{Mode: t.Mode, Muted: t.Muted}
	if err := savePreference(func(cfg *config.Config) { cfg.Radio.Tones = tones }); err != nil {
		return err
	}
	if lp.deck != nil {
		lp.deck.setTones(cast.Tones{Mode: tones.Mode, Muted: tones.Muted})
	}
	return nil
}
```

The four helpers, as code (`app/cast.go`), the role keys taken from the registry, never spelled:

```go
// platformHalf is this host's half of a pair; platformHalfPtr the same, to write.
func platformHalf(rv config.RoleVoice) string {
	if runtimeGOOS() == "darwin" {
		return rv.MacOS
	}
	return rv.Piper
}

func platformHalfPtr(rv *config.RoleVoice) *string {
	if runtimeGOOS() == "darwin" {
		return &rv.MacOS
	}
	return &rv.Piper
}

// displayName is what Setup shows for a stored value: a Piper key becomes
// its catalogue name; a macOS name is itself; "" stays "".
func displayName(stored string) string {
	if stored == "" || runtimeGOOS() == "darwin" {
		return stored
	}
	if spec, ok := synth.VoiceByName(stored); ok { // accepts keys and names (install.go:97)
		return spec.Name
	}
	return stored
}

// storedName is the reverse: a catalogue name becomes its key off darwin; an
// unknown name is stored as typed (Validate cannot know the host's voices —
// FR-7's fallback covers it).
func storedName(name string) string {
	if name == "" || runtimeGOOS() == "darwin" {
		return name
	}
	if spec, ok := synth.VoiceByName(name); ok {
		return spec.Key
	}
	return name
}

// setupRoles are the roles Setup edits, keyed as the registry keys them —
// the one owner of the words the TUI's castKey* constants repeat.
func setupRoles() map[string]cast.Role {
	return map[string]cast.Role{cast.All.Key(): cast.All, cast.Alerts.Key(): cast.Alerts, cast.Standard.Key(): cast.Standard,
		cast.Weather.Key(): cast.Weather, cast.Maritime.Key(): cast.Maritime, cast.Fire.Key(): cast.Fire, cast.Seismic.Key(): cast.Seismic}
}
```

`castViewOf`/`saveCast` iterate `setupRoles()` (a `pairOf(cfg, role) *config.RoleVoice` switch maps a role to its
field) instead of literal `"alerts"…` maps.
`TestCastViewKeysMatchTheRegistry` (`app/cast_test.go`) pins the seven `castKey*` words to
`{cast.All.Key()} ∪ {r.Key() for the six Setup roles}` — a subset of `Assignable()` plus the root (Breaking,
SevereRead and Station have no Setup row, FR-8); `TestToneClassesMatchTheSetupRows` pins
`len(toneClasses()) == tty.ToneClassCount`; `TestCastViewRoundTripsPiperKeys` (Linux shape, `runtimeGOOS`
stubbed) saves `"Ryan"` and reads back `piper = "en_US-ryan-medium"` then displays `"Ryan"` again. `recast()`
(P2 Task 2.10) runs `cast.Validate` through `setCast`; `ttyStats` maps `deck.Problems()` into `ConfigNotes`
(Task 4.8) — the class keys' only validator reaches the screen. **Verify:**
`go test ./app ./modes/tty -run 'Cast|Tones|Setup' -count=1 && make lint-imports`

---

### Task 4.8 — the `[S]` cast table, the config notes, and the non-TUI surface (FR-7; NFR-5)

**File:** `modes/tty/status_test.go` (RED — append)

```go
func TestStatusShowsTheCastTableAndConfigNotes(t *testing.T) {
	d := newTestDashboard(t)
	d.cfg.Stats = func() Stats {
		return Stats{
			Cast:        []CastRow{{Role: "Breaking takeover", Requested: "Rishi", Spoken: "Samantha", Link: "root", State: "installing", Reason: "Rishi is not installed"}},
			ConfigNotes: []string{"[keys] radio-size: no such action in this version — ignored", "[radio.voices.wether]: unknown table — ignored", "[radio.tones] muted: \"thunder\" is not a tone class"},
		}
	}
	d.cfg.Tones, d.cfg.ToneClasses = Tones{Mode: "mute", Muted: []string{"warning", "storm"}}, testToneClasses()
	lines := plainLines(d.statusLines())
	if !containsLine(lines, "CAST") || !containsLine(lines, "ROLE") || !containsLine(lines, "Breaking takeover  Rishi") || !containsLine(lines, "→ Samantha") || !containsLine(lines, "(root)  installing — Rishi is not installed") { // the test model renders the Unicode set (ASCII false)
		t.Fatalf("the cast table, header and reason included:\n%s", strings.Join(lines, "\n"))
	}
	if !containsLine(lines, "TONES") || !containsLine(lines, "Alert Tones: Muted (2 of 6) — warning, storm") {
		t.Fatalf("the tone state on a surface that fits at 80 cols:\n%s", strings.Join(lines, "\n"))
	}
	for _, w := range []string{"CONFIG", "radio-size", "wether", "thunder"} {
		if !containsLine(lines, w) {
			t.Fatalf("missing %q", w)
		}
	}
}
```

**File:** `modes/tty/dashboard.go` — `Stats` gains `Cast []CastRow` and `ConfigNotes []string`;
`CastRow{Role, Requested, Spoken, Link, State, Reason string}` (the TUI projection of the domain's resolution;
strings only; `State` = `""` · `installing` · `install failed — retries at hh:mm`). **File:**
`modes/tty/status.go` — `statusSections` gains `cast, tones, config`; `statusBlocks` builds `CAST`
(`statusHeader("CAST")`, a header row `ROLE  REQUESTED  SPOKEN  VIA` in the same columns, then one row per
role: `render.PadTo(role, 18) + render.PadTo(requested, 16) + g.Arrow + " " + render.PadTo(spoken, 16) + "(" + link + ")  " + state`
and — when set — `" — " + Reason`; cells padded by `PadTo`, never `%-18s`), `TONES` (one line:
`Alert Tones: On` / `Muted (all)` / `Muted (n of N) — <keys>` from `d.cfg.Tones` and `ToneClasses`) and `CONFIG`
(one row per note; the ignored `[keys]` overrides from `d.keyNotes` prepended; **capped at 32 rows, de-duplicated**
— `capNotes(notes, 32)`, one helper for every feed: `Merge`'s notes, `cfg.Unknown`, `cast.Validate`'s problems,
whose unknown-class-key problem is reported once with a count); `statusLines` stacks them after `dumps`. **File:** `app/stats.go` —
`ttyStats` fills `Cast` from `lp.deck.Resolutions()` mapped by `castRowOf(res)` — every string through
`render.PlainLine`; `Spoken` on Linux shown as its catalogue name via `displayName`; `State` from
`deck.installState(res.Requested)` — and `ConfigNotes` from `cfg.Unknown` (already plain, P1 1.11) followed by
`deck.Problems()` (`"[" + p.Key + "] " + p.Message`, PlainLine'd). The hostile assertion lives here: an `app`
test loads `hostile-name.toml` + `quoted-escape-key.toml`, builds the `Stats`, renders `statusLines()` through a
`tty` model and asserts no line carries `\x1b`/`\n`. **Non-TUI surface (FR-7):** `watchpost report --verbose` gains a `cast:` block and a `tones:` line. The host
facts have ONE owner the deck and the report share (`app/hostfacts.go`):

```go
// hostFacts is cast.Host without a deck: the platform, the macOS voice list
// (the curated list until `say -v ?` has answered — the same discovery the
// deck runs, under the same 30 s ceiling), and Piper's install check.
type hostFacts struct {
	voiceDir string
	mu       sync.Mutex
	voices   []string
	seen     bool
}

func (h *hostFacts) Platform() string { return runtimeGOOS() }

func (h *hostFacts) Discovered() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.seen {
		return macVoices()
	}
	return append([]string(nil), h.voices...)
}

func (h *hostFacts) Installed(key string) bool {
	spec, ok := synth.VoiceByName(key)
	if !ok {
		return false
	}
	_, ok = synth.FindPiperVoice(h.voiceDir, spec)
	return ok
}

// discover reads the host's voices once (macOS: discoverMacVoices; elsewhere nothing to read).
func (h *hostFacts) discover(ctx context.Context) {
	list := discoverMacVoices(ctx) // the package func the deck's listVoices calls too: exec.CommandContext, 30 s
	h.mu.Lock()
	h.voices, h.seen = list, true
	h.mu.Unlock()
}

// CastReport is the [S] cast table for a shell: one plain line per assignable
// role, `role: requested → spoken (link) — reason`, every field PlainLine'd.
func CastReport(ctx context.Context, cfg config.Config) []string {
	h := &hostFacts{voiceDir: voiceDir()}
	h.discover(ctx)
	c := castConfig(cfg)
	out := []string{"cast:"}
	for _, r := range cast.Assignable() {
		res := cast.Resolve(r, c, h)
		line := "  " + r.String() + ": " + render.PlainLine(res.Requested) + " → " + render.PlainLine(displayName(res.Spoken)) + " (" + res.Link.String() + ")"
		if res.Reason != "" {
			line += " — " + render.PlainLine(res.Reason)
		}
		out = append(out, line)
	}
	return append(out, "tones: "+tonesLine(cfg.Radio.Tones))
}
```

The deck embeds a `*hostFacts` (its `Platform/Discovered/Installed` methods forward to it; `listVoices` becomes
`h.discover` + `castChanged`), so there is one discovery, one ceiling, one allowlist. `cmd/watchpost/root.go`
(the `report` command lives there — there is no `report.go`) appends the block under `--verbose` beside
`RenderRequests`; `--json` carries the rows JSON-escaped as `app/dump.go` already does;
`TestReportVerboseListsTheCast`. The diagnostic dump carries the same rows. **Verify:** `go test ./modes/tty ./app ./cmd/... -run 'Status|Cast|Dump|Report' -count=1`

---

### Task 4.9 — help, README, CHANGELOG, `where-things-happen`, glyph parity, NFR-8

- `help_about.go` RADIO group (Task 4.2); the `Help` strings: `voice` → "Correspondents (Setup)", `ticker-mute`
  → "Mute Alert Tones"; `body.go`'s masthead label (Task 4.7).
- **Glyph parity (FR-13):** the Setup focus mark, `⚠` error lines and `✔ working` go through `o.Glyphs()`
  (`Pointer`, `Alert`, `OK`/`Fail`) in `setup.go:263,304,310,338,357,396`; `ScrollPanel`'s rail (`panel.go:211-221`)
  takes `RailGlyphs` from `Opts` (the severe table's type, `severe_table.go:234` — ASCII `^ # v |`); `[S]`'s `→`
  is `Glyphs.Arrow`.
- `platform/render/contrast.go` `aaPairs`: add `{TextBright, ProviderOK, ProviderDown} × modal` — the Setup
  window grows from 3 rows to ~25 on the modal ground. **Not `KeyChip`:** it is a composite token (its own
  background); `fgRGB` cannot read it, `Contrast` returns 0 and `deepenToAA` would blacken every theme's modal
  ground. The AA gate gains a composite self-check instead (fg vs the token's embedded `48;…` bg). The OP-3 support
  line is untinted (`ModalFG`).
- `—` in the re-worded DATA rows and `…` in `stored (…d9a6)` / the deck's `preparing …` go through `Glyphs.Dash`
  and a new `Glyphs.Ellipsis` (`…` / `...`); the FIRMS key mask `•` becomes `Glyphs.Bullet` (`•` / `*`) — the
  ASCII golden (4.12) pins them; the `v` for `▾` shares the Viz key's letter and the trend-down mark on purpose
  (a dropdown arrow, noted in the golden review).
- `README.md`: the Radio section's voice paragraph → the Correspondents group in Setup (single voice or a cast;
  alerts / reports / overrides; the tone mutes; `[M]` = tones only); the maritime report paragraph; the Radio
  panel per breakpoint; the config keys `[radio] cast`, `[radio.voices.<role>]`, `[radio.tones] mode/muted` in
  the config section with the MVS-D-20 note ("comments are dropped on save; unknown keys are kept — inside
  tables; not inside `[[locations]]`/`[[recent]]` entries"); the "For tinkerers" scripts paragraph
  (`README.md:234-238`) gains `handover/` and `maritime-report/`; `docs/extending.md` Walkthrough 3 gains "where
  a voice is chosen" (the `cast` package, `resolveVoice`); screenshots re-captured by the HUM LEAD
  (`docs/img/setup.png`, `radio.gif`, a new `maritime.png`); `README.md:62` (`docs/img/voices.png`) and
  `radio-min.png` go.
- `CHANGELOG.md` `[0.14.0] — Unreleased`: Added (the cast; the maritime report incl. the coastal forecast; the
  five tones; per-class mute; the `[S]` cast table; `report --verbose`'s cast block) · Changed (Setup two columns;
  the Radio panel per breakpoint; `[V]`/`[T]` retired, `V` opens Setup; **`[M]` mutes tones only — a 0.13.0
  listener who muted the alerts hears the words again, by design (MVS-D-26)**; the tail's wording; unknown config
  keys preserved on save; `Merge` ignores overrides for retired actions) · Fixed (the `tail:{{voice}}` cache
  key; unbounded render concurrency; the alert tone waiting on an install).
- `docs/where-things-happen.md`: the `handover` and `maritime-report` script folders (`:15`); rows "Setup saves
  the cast — `app/cast.go:saveCast`", "The Setup window is laid out — `modes/tty/setup.go:setupBody`",
  "The player's layout is chosen — `modes/tty/radio_panel.go:radioBreakpoint`"; the `[S]` row names
  `modes/tty/status.go:statusBlocks`. (The P2 rows for the Director and the resolver landed at the P2 gate.)
- **NFR-8 gate (the one pattern; objectives NFR-8 cites this task):**
  `grep -rn '\[V\]\|\[T\]\|radio-size\|radio-min\|radioMin\|modalVoice\|VoiceNoteMsg\|Severe Alerts\|voices\.png' --include='*.go' --include='*.md' --include='*.expect' . | grep -v '^./06_docs\|^./CHANGELOG.md\|^./third_party'`
  → zero (`Voice:` is not in the pattern — `SayVoice{Voice: …}` is a field, not the retired control).
**Verify:** the grep; `go vet ./...`; `go test ./cmd/... -run WhereThingsHappen`

---

### Task 4.10 — the PTY journey (M2) and the M3 arms

`scripts/quality/validate-journey.expect`: after the Setup step, a cast step — `V` (Setup at the
correspondents) · `↓` (Correspondent Cast) · `space` · `↓` (Alerts / Takeovers) · `→` (the next voice) · `enter`
× 6 (through the remaining rows; the last saves) = **11 keypresses**, asserting "Correspondent Cast" on screen
after `V` and, after the save, `[S]` showing `Breaking takeover` with the chosen name. The script counts its
own `send` calls for the step and fails above **11** (`set presses 0; proc press {k} {…; incr presses}`) — the
pin is the measured path; M2's row in the brief says so (AM-18). The M3 arms (`gates.md` §2; two trials, A/B
against the 0.13.0 binary) are the HUM LEAD's UAT, recorded in `07-readiness/validate/m3.md`.
**Verify:** `make pty-severe` and `HOME=$(mktemp -d) expect scripts/quality/validate-journey.expect dist/journey.log`
(the script takes its log path as `argv[0]`, `validate-journey.expect:9-14`).

---

### Task 4.11 — the Setup-open alloc pin (NFR-3)

`modes/tty/bench_test.go`: `TestSetupAllocBudget` (landed at P0 with the 0.13.0 window's numbers as an
*informational* row in `perf-protocol.md` §2) is re-measured on the new window — Setup open at 80×24 and
133×44 with the correspondents focused, hit and miss — and re-pinned ×1.05 of the measurement (spike-then-pin,
as `severeAllocBudget` did). The body-built-once rule (Task 4.4) is what keeps the miss path near the P0 number;
if the miss exceeds 2× the P0 miss, the pin is not raised — the builders are fixed. **Verify:**
`go test ./modes/tty -run 'AllocBudget' -v`

---

### Task 4.12 — goldens, once

`modes/tty/setup_golden_test.go`: `TestSetupGoldens` renders the Setup window open at the correspondents with
the fixture cast of `TestSetupCastRowsFollowTheMock` at 80×24 (`frame-setup-80x24.golden`), 133×44
(`frame-setup-133x44.golden`) and 133×44 `--ascii` (`frame-setup-133x44-ascii.golden`) through the same
`checkGolden` the frame goldens use (`golden_test.go:35-52`), honouring `-update-golden`. Then re-record every
frame golden that carries the Radio panel and the severe goldens (`go test ./modes/tty -run 'Golden' -update-golden`),
review the diff row by row against the two mocks, and commit. The `--ascii` golden (FR-13): `[ ]`/`[x]`,
`o`/`*` radios, `>` focus marks, `v` for `▾`, `|` for `│`, `->` for `→`, `^ # v |` for the rail.
**Verify:** `go test ./modes/tty -run Golden -count=1`

---

### Task 4.13 — P4 gate (the batch-exit checklist)

```
go test ./... -race -count=2 -timeout 600s
make verify
a2dh validate && make alloc-budget && golangci-lint run ./... && staticcheck ./...   # gates.md §1
A2DH=<framework build> make p10             # the PINNED framework build (gates.md batch record)
cp dist/p10.json 06_docs/02_features/multi-voice-support/07-readiness/p10-p4.json
make pty-severe && HOME=$(mktemp -d) expect scripts/quality/validate-journey.expect dist/journey.log
go test ./modes/tty ./app -run DeclarationSet -update-declset       # the packages whose declsets P4 moves (gates.md §1 is the owner)
make build && ./dist/watchpost   # UAT: Setup at 80×24 and 133×44; the three Radio breakpoints; V; M; the cast on the air
git commit -m "multi-voice-support P4: Setup Correspondents + Tones groups, the Radio panel per breakpoint, [V]/[T] retired, the [S] cast table, docs, goldens"
```

- **P10 ledger:** rows only from `dist/p10.json`; delete the `modal_chooser.go` rows. `setupCastKey` and
  `setupPickerKey` are each under 15 decisions (the split is unconditional); `pad` is counter-form; expected:
  reason refreshes on the `modes/tty`/`app`/`render`/`term` package rows.
- **Build log:** `04-development/p4-build-log.md`. No attribution trailers. The three readiness documents
  (`07-readiness/{gates,release-checklist,linux-validation-protocol}.md`) are checked off here before SHIP.

## Open points for the HUM LEAD (at the PLAN exit)

- **OP-1 — the `←→  Voice` chip.** The mock's chip row is `tab · enter · ↑↓ · esc`; MVS-D-27 added `p  Preview`
  while a picker is focused. The pickers cycle with `←→`, which nothing on screen says; this plan shows
  `←→  Voice` beside `p  Preview` under the same rule. Keep, or drop to the mock's row?
- **OP-2 — the DATA row wording.** The mock re-words today's two questions (`Default location:`, `NASA FIRMS key:
  stored (…d9a6) — ✔ working`); this plan ships the mock's words. Confirm.
- **OP-3 — "Mute:" with nothing ticked.** `[M]` and a bare "Mute:" mean *every* class (what the screen says is
  what the ear gets). The mock has no words for it; this plan draws a support line `nothing ticked = every class`
  under "Mute:" only while that is the state, and the masthead reads `Alert Tones: Muted (all)` / `Muted (n of 6)`.
  Keep, or leave the rule to the README?
- **OP-4 — the chip wording, the EVENTS rule, and M2's number.** (M2 is 11 keypresses on this design and the
  journey pins ≤ 11 — E-9 asks the HUM LEAD to ratify that as the metric of record; a Save chord would lower it.) With one keyboard rule for the window (`↑↓` move, `space`
  pick), the mock's `↑↓  Pick` chip is true only for the location hints. Reword to `↑↓  Move · space  Pick`, or
  keep the mock's word? Also: Save lives on the last row (11 keypresses on the journey; a listener changing only
  their location presses `tab` ×4 then `enter` ×8) — add an always-present `ctrl+s  Save` chip, or keep the mock?
- **OP-5 — the chip row as a footer.** The mock draws the chips on the frame's last row beside the rail; this
  plan pins them under the scroll window (they never scroll away at 80×24). Confirm that reading.
