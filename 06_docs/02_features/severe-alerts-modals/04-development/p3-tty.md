# P3 — TTY: tokens, the window, the memo, budgets

Depends on P1/P2 for the message types' meaning, but compiles independently (UI types live here).
`modes/tty` imports `platform/*` only.

---

## Task 3.1 — Category tokens, `CategoryTone`, the guards

**Files:** `platform/render/theme.go`, `platform/render/themes.go`, `platform/render/sgr.go`,
`platform/render/theme_test.go` (MODIFY)

**Test first (RED):** `platform/render/theme_test.go` (append)
```go
// The category tints are theme-INDEPENDENT (SAM-D-7): every theme except
// Monochrome resolves them to the default. The planted override is the
// positive control — a guard that passes on an empty set proves nothing
// (red-team C-15a).
func TestEventCategoryTokensAreThemeIndependent(t *testing.T) {
	t.Cleanup(func() { SetTheme(DefaultThemeName) })
	def := defaultTheme()
	for _, name := range ThemeNames() {
		if name == "Monochrome" {
			continue
		}
		if !SetTheme(name) {
			t.Fatal(name)
		}
		for _, tok := range []Token{EventCatRedBG, EventCatOrangeBG, EventCatYellowBG, EventCatBlueBG} {
			if Tok(tok) != def[tok] {
				t.Errorf("%s overrides %s: %q (must inherit %q)", name, tok, Tok(tok), def[tok])
			}
		}
	}
	// Positive control: a registered theme that DOES override must be caught.
	RegisterTheme("planted-control", map[Token]string{EventCatRedBG: "48;2;1;2;3"})
	t.Cleanup(func() { UnregisterTheme("planted-control") })
	SetTheme("planted-control")
	if Tok(EventCatRedBG) == def[EventCatRedBG] {
		t.Fatal("the control override was not applied — the guard is vacuous")
	}
}

// White body text on every category tint, on both modal substrates, at AA.
func TestCategoryToneContrastAA(t *testing.T) {
	t.Cleanup(func() { SetTheme(DefaultThemeName) })
	for _, name := range ThemeNames() {
		SetTheme(name)
		for _, dark := range []bool{true, false} {
			for _, hue := range []Token{EventCatRedBG, EventCatOrangeBG, EventCatYellowBG, EventCatBlueBG} {
				fg, bg := CategoryTone(hue, dark)
				fl, ok1 := fgLuminance(fg)
				bl, ok2 := bgLuminance(bg)
				if !ok1 || !ok2 {
					t.Fatalf("%s %s dark=%v: unreadable tone %q / %q", name, hue, dark, fg, bg)
				}
				hi, lo := max(fl, bl), min(fl, bl)
				if ratio := (hi + 0.05) / (lo + 0.05); ratio < 4.5 {
					t.Errorf("%s %s dark=%v: %.2f:1 below AA", name, hue, dark, ratio)
				}
			}
		}
	}
}
```
`UnregisterTheme` does not exist yet (`themes.go` has Register/ThemeNames/SetTheme/ThemeName) — add it
beside `RegisterTheme`:
```go
// UnregisterTheme removes a registered theme (tests plant and remove control
// themes); the default is restored if the removed one was active.
func UnregisterTheme(name string) {
	themeMu.Lock()
	defer themeMu.Unlock()
	if name == DefaultThemeName {
		return
	}
	delete(themeTable, name)
	if themeName == name {
		themeName = DefaultThemeName
	}
	themeGen.Add(1)
}
```
(`themeMu`, `themeTable`, `themeName`, `themeGen` are the registry's identifiers — `themes.go:24-29`;
`Tok` resolves through `activeTable()`, so no separate "active" pointer exists.)

**Code — `theme.go`:** in the token block after `TickerMutedFG`:
```go
	// The severe-events window's category tints (0.13.0, SAM-D-7): fixed hues
	// keyed to the ticker lanes — Red disasters, Orange warnings, Yellow
	// watches/advisories/statements, Blue tropical — rendered by CategoryTone
	// onto the active modal substrate, so they read the same in every theme
	// (Monochrome greys them). Values: HUM LEAD's colour pass.
	EventCatRedBG    Token = "event.cat.red.bg"
	EventCatOrangeBG Token = "event.cat.orange.bg"
	EventCatYellowBG Token = "event.cat.yellow.bg"
	EventCatBlueBG   Token = "event.cat.blue.bg"
```
in `defaultTheme()` after `TickerMutedFG`:
```go
		EventCatRedBG:    "48;2;58;18;18",  // the Ticker red mixed 0.30 onto #1D2830 (HUM LEAD colour pass to tune)
		EventCatOrangeBG: "48;2;64;36;10",
		EventCatYellowBG: "48;2;60;52;14",
		EventCatBlueBG:   "48;2;14;34;64",
```
**`themes.go`** Monochrome overrides, beside the Ticker greys:
```go
			EventCatRedBG: "48;2;70;70;70", EventCatOrangeBG: "48;2;58;58;58", EventCatYellowBG: "48;2;46;46;46", EventCatBlueBG: "48;2;62;62;62", // 0.13.0: category by shade on monochrome; the tab glyph carries identity
```
**`sgr.go`** after `ModalTone`:
```go
// CategoryTone is the severe-events window's fg/bg pair for a category's
// tint token (the tab registry in modes/tty owns which tab wears which token):
// white body text on the fixed tint, mixed onto the active modal substrate so
// a light-background terminal is not handed a dark hole (RS-11).
func CategoryTone(hue Token, dark bool) (fg, bg string) {
	base := ModalBGDark
	if !dark {
		base = ModalBGLight
	}
	return Tok(AlertModalText), mixBG(Tok(hue), Tok(base), categoryBlend)
}

// categoryBlend is how much of the category tint shows over the modal
// substrate (the EventCat* values are already pre-darkened tints, so 0.6 of
// them reads as the plan's "~0.30 of the pure hue"). HUM LEAD tunes at UAT.
const categoryBlend = 0.6

// mixBG blends two "48;2;r;g;b" backgrounds: t of a over b.
func mixBG(a, b string, t float64) string {
	ar, ag, ab, ok1 := bgRGB(a)
	br, bg, bb, ok2 := bgRGB(b)
	if !ok1 || !ok2 {
		return a
	}
	m := func(x, y int) int { return int(float64(x)*t + float64(y)*(1-t) + 0.5) }
	return fmt.Sprintf("48;2;%d;%d;%d", m(ar, br), m(ag, bg), m(ab, bb))
}

// bgRGB reads a truecolor background value.
func bgRGB(v string) (r, g, b int, ok bool) {
	parts := strings.Split(v, ";")
	if len(parts) != 5 || parts[0] != "48" || parts[1] != "2" {
		return 0, 0, 0, false
	}
	r, _ = strconv.Atoi(parts[2])
	g, _ = strconv.Atoi(parts[3])
	b, _ = strconv.Atoi(parts[4])
	return r, g, b, true
}
```
(add `fmt`, `strconv` imports to `sgr.go` if absent.) **`theme_test.go`** add the bg reader the AA test uses:
```go
// bgLuminance reads a "48;2;r;g;b" or "48;5;n" background value.
func bgLuminance(v string) (float64, bool) {
	parts := strings.Split(v, ";")
	switch {
	case len(parts) == 5 && parts[0] == "48" && parts[1] == "2":
		r, _ := strconv.Atoi(parts[2])
		g, _ := strconv.Atoi(parts[3])
		b, _ := strconv.Atoi(parts[4])
		return luminance(r, g, b), true
	case len(parts) == 3 && parts[0] == "48" && parts[1] == "5":
		n, err := strconv.Atoi(parts[2])
		if err != nil || n > 255 {
			return 0, false
		}
		return luminance(xterm256RGB(n)), true
	}
	return 0, false
}
```

**Verify:** `go test ./platform/render -run 'TestEventCategory|TestCategoryTone|TestThemeTokens' -v`

---

## Task 3.2 — UI types and the tab registry

**File:** `modes/tty/severe.go` (CREATE — part 1: types)

**Test first (RED):** `modes/tty/severe_test.go`
```go
package tty

import "testing"

func TestSevereTabsAreSixInImportanceOrder(t *testing.T) {
	tabs := severeTabs()
	want := []string{"Warnings", "Watches", "Advisories", "Spec. Statements", "Sig. Quakes", "Tropical"}
	if len(tabs) != len(want) {
		t.Fatalf("%d tabs", len(tabs))
	}
	for i, w := range want {
		if tabs[i].Label != w || tabs[i].Short == "" || tabs[i].Tone == "" {
			t.Errorf("tab %d: %+v", i, tabs[i])
		}
	}
	if !tabs[SevereAdvisories].WatchlistHint || !tabs[SevereStatements].WatchlistHint || tabs[SevereWarnings].WatchlistHint {
		t.Error("watchlist hint on the wrong tabs")
	}
}
```

**Code:**
```go
package tty

// severe.go — the Severe Weather / Disaster Events window (0.13.0): UI types
// the app maps domains/severe onto (the snapshot-only rule — this package
// never imports the domain), the tab registry, the window's state helpers, and
// the browse / detail renderers. Mocks: 03-architecture-design/plan.md §5.

import (
	"time"

	"github.com/branden-thompson/watchpost/platform/render"
)

// SevereTab is the window's category, in the app's importance order.
type SevereTab int

const (
	SevereWarnings SevereTab = iota
	SevereWatches
	SevereAdvisories
	SevereStatements
	SevereQuakes
	SevereTropical
	severeNumTabs
)

// SevereMsg publishes the severe-events index (sent by the app's severe deck
// only when the row set changed). Totals count every row per tab BEFORE the
// cap, so the window can say "showing N of M".
type SevereMsg struct {
	Rows    []SevereRow
	Totals  [severeNumTabs]int
	Updated time.Time // the newest SUCCESSFUL source fetch (dataAsOf), zero when none yet
	Sources []SevereSource
	Gen     uint64 // bumps per publish — the modal memo's key field
}

// SevereSource is one feed's health as the window states it ("NWS unavailable").
type SevereSource struct {
	Name string
	OK   bool
}

// SevereRow is one event as the window lists it; the times arrive formatted in
// the right zone (FR-9) and the record composed (the app owns both).
type SevereRow struct {
	Key      string
	Tab      SevereTab
	Product  string // "Tornado Warning" · "Tropical Storm Dolly"
	Location string
	Declared string // "08/28 08:45 CDT"
	Expires  string // "" when none
	Severity TickerSeverity
	Record   SevereRecord
}

// SevereRecord is the [A]-shaped record of one row.
type SevereRecord struct {
	Title, Meta, Timing, Area string
	Paras                     []string
}

// severeTab is one registry row: adding a category is adding a row here plus
// its classification in the domain (red-team C-19).
type severeTab struct {
	Label, Short  string
	Tone          render.Token
	WatchlistHint bool // Advisories / Statements come from the tracked locations only (SAM-D-10)
}

// severeTabs is the registry (a function, not a global — P10-06).
func severeTabs() []severeTab {
	return []severeTab{
		{"Warnings", "Warn", render.EventCatOrangeBG, false},
		{"Watches", "Watch", render.EventCatYellowBG, false},
		{"Advisories", "Advis", render.EventCatYellowBG, true},
		{"Spec. Statements", "Stmts", render.EventCatYellowBG, true},
		{"Sig. Quakes", "Quakes", render.EventCatRedBG, false},
		{"Tropical", "Tropical", render.EventCatBlueBG, false},
	}
}

// severeBreakingWindow is how long a breaking event steers the opening tab
// (SAM-D-17): within it the window opens on that event's category.
const severeBreakingWindow = 10 * time.Minute
```
(Task 3.5 adds `fmt`, `strings`, `platform/term` as it uses them.)

**Verify:** `go test ./modes/tty -run 'TestSevereTabs' -v`

---

## Task 3.3 — State, enum, key, `toggleModal`, dispatch, help, `tickNeeded`

**Files:** `modes/tty/dashboard.go`, `modes/tty/help_about.go` (MODIFY); `modes/tty/severe.go` (state helpers)

**Test first (RED):** `modes/tty/severe_test.go` (append)
```go
func TestWOpensTheWindowAndEscEscCloses(t *testing.T) {
	m := dash(t)
	m, _ = m.Update(SevereMsg{Gen: 1, Rows: []SevereRow{{Key: "k", Tab: SevereWarnings, Product: "Tornado Warning", Location: "Olathe, KS", Declared: "08/28 08:45 CDT", Record: SevereRecord{Title: "TORNADO WARNING"}}}, Totals: [severeNumTabs]int{1}})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	d := m.(Dashboard)
	if d.modal != modalSevere || d.severeDetail {
		t.Fatalf("w did not open the browse view: %+v", d.modal)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.(Dashboard).severeDetail {
		t.Fatal("enter did not open the detail")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if d := m.(Dashboard); d.modal != modalSevere || d.severeDetail {
		t.Fatal("first esc must return to the table")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.(Dashboard).modal != modalNone {
		t.Fatal("second esc must close")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.(Dashboard).modal != modalSevere {
		t.Fatal("ctrl+s alias must open")
	}
}

func TestWIsInertWhileSetupOwnsTheKeyboard(t *testing.T) {
	m := dash(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 's', Text: "s"}) // Setup
	m, _ = m.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	if m.(Dashboard).modal != modalSetup {
		t.Fatal("w must be inert inside Setup")
	}
}

func TestApplySevereKeepsTheFocusedEventAndClosesAVanishedRecord(t *testing.T) {
	m := dash(t)
	rows := []SevereRow{{Key: "a", Tab: SevereWarnings}, {Key: "b", Tab: SevereWarnings}}
	m, _ = m.Update(SevereMsg{Gen: 1, Rows: rows})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // record for "b"
	m, _ = m.Update(SevereMsg{Gen: 2, Rows: []SevereRow{{Key: "c", Tab: SevereWarnings}, {Key: "b", Tab: SevereWarnings}, {Key: "a", Tab: SevereWarnings}}})
	if d := m.(Dashboard); d.severeRow != 1 || !d.severeDetail {
		t.Fatalf("the record must stay on b: row %d detail %v", d.severeRow, d.severeDetail)
	}
	m, _ = m.Update(SevereMsg{Gen: 3, Rows: []SevereRow{{Key: "a", Tab: SevereWarnings}}})
	if d := m.(Dashboard); d.severeDetail || d.severeRow != 0 {
		t.Fatal("a vanished event's record must close to the table")
	}
}

func TestOpeningTabFollowsARecentBreakingEvent(t *testing.T) {
	m := dash(t).(Dashboard)
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return now }
	m.lastBreaking, m.lastBreakingTab = now.Add(-5*time.Minute), SevereQuakes
	if got := m.severeOpeningTab(); got != SevereQuakes {
		t.Fatalf("within 10 min → the breaking tab, got %v", got)
	}
	m.lastBreaking = now.Add(-11 * time.Minute)
	if got := m.severeOpeningTab(); got != SevereWarnings {
		t.Fatalf("after 10 min → Warnings, got %v", got)
	}
}
```

**Code — `dashboard.go`:**
- `Dashboard` fields (after `breaking`):
```go
	severe          SevereMsg // 0.13.0: the severe-events index the window lists (Gen keys the modal memo)
	severeByTab     [severeNumTabs][]SevereRow // the index bucketed per tab, once per SevereMsg
	severeTab       SevereTab // the open tab
	severeRow       int       // the focused row within the tab
	severeDetail    bool      // the body shows the focused row's record (esc backs out, esc esc closes — SAM-D-9)
	lastBreaking    time.Time // when the last breaking takeover started — steers the opening tab (SAM-D-17)
	lastBreakingTab SevereTab
```
- enum: append `modalSevere // w / ctrl+s: the Severe Weather / Disaster Events window (0.13.0)` after
  `modalSetup`.
- `defaultKeyMap()`: add `"severe": {Keys: []string{"w", "ctrl+s"}, Help: "Severe Weather / Disaster Events"},`
  after `"alert-details"`.
- `toggleModal`: add `case "severe": return d.openSevere(), true` before `case "close"`, and change the
  close case to:
```go
	case "close":
		if d.modal == modalSevere && d.severeDetail {
			d.severeDetail, d.modalScroll = false, 0 // esc backs out of the record to the table (SAM-D-9)
			return d, true
		}
		return d.close(), true
```
- `handleKey`: before `if toggled, ok := d.toggleModal(act); ok {` add:
```go
		if d.modal == modalSevere && act == "details" { // enter on a row opens its record in place (FR-4)
			return d.openSevereDetail(), nil
		}
```
- `dispatch`: add `case SevereMsg: return d.applySevere(v), nil` after the ticker cases; in `handleTicker`'s
  `TickerBreakingMsg` case add `d.lastBreaking, d.lastBreakingTab = d.now(), severeTabOf(v.Item.Category)`.
- `tickNeeded`: change the modal case to `case d.modal == modalStatus || d.modal == modalDetails || d.modal == modalSevere:` (the "Updated" stamp and the minute bucket — red-team P8).

**Code — `severe.go` (state helpers):**
```go
// openSevere opens the window on its opening tab (SAM-D-17) with the focus at
// the top and the record closed.
func (d Dashboard) openSevere() Dashboard {
	d = d.open(modalSevere)
	d.severeTab, d.severeRow, d.severeDetail = d.severeOpeningTab(), 0, false
	return d
}

// severeOpeningTab: the last breaking event's category when it broke within
// severeBreakingWindow, else Warnings.
func (d Dashboard) severeOpeningTab() SevereTab {
	if !d.lastBreaking.IsZero() && d.now().Sub(d.lastBreaking) <= severeBreakingWindow {
		return d.lastBreakingTab
	}
	return SevereWarnings
}

// openSevereDetail shows the focused row's record in place.
func (d Dashboard) openSevereDetail() Dashboard {
	if len(d.severeRows(d.severeTab)) == 0 {
		return d
	}
	d.severeDetail, d.modalScroll = true, 0
	return d
}

// applySevere takes a published index. The focus follows its ROW (by key),
// not its index: an open record stays on the same event when rows shift, and
// a record whose event vanished closes back to the table (red-team PLAN B-7).
func (d Dashboard) applySevere(v SevereMsg) Dashboard {
	var focusKey string
	if rows := d.severeRows(d.severeTab); d.severeRow < len(rows) {
		focusKey = rows[d.severeRow].Key
	}
	d.severe = v
	d.severeByTab = bucketSevere(v.Rows)
	rows := d.severeRows(d.severeTab)
	d.severeRow = 0
	found := false
	for i, r := range rows {
		if r.Key == focusKey {
			d.severeRow, found = i, true
			break
		}
	}
	if d.severeDetail && !found {
		d.severeDetail, d.modalScroll = false, 0
	}
	return d
}

// severeRows are a tab's rows, in the app's order (Declared DESC) — bucketed
// once per SevereMsg in applySevere, never filtered per frame (red-team PLAN P5).
func (d Dashboard) severeRows(tab SevereTab) []SevereRow { return d.severeByTab[tab] }

// bucketSevere splits the index into its tabs.
func bucketSevere(rows []SevereRow) [severeNumTabs][]SevereRow {
	var out [severeNumTabs][]SevereRow
	for _, r := range rows {
		if r.Tab >= 0 && r.Tab < severeNumTabs {
			out[r.Tab] = append(out[r.Tab], r)
		}
	}
	return out
}

// severeTabOf maps a ticker lane to the window's tab.
func severeTabOf(c TickerCategory) SevereTab {
	switch c {
	case CatQuake:
		return SevereQuakes
	case CatTropical:
		return SevereTropical
	case CatWatch:
		return SevereWatches
	}
	return SevereWarnings
}
```
**Code — `help_about.go`:** in `helpGroups()` NAVIGATE, insert `"severe"` after `"alert-next"`.

**Verify:** `go test ./modes/tty -run 'TestWOpens|TestWIsInert|TestOpeningTab' -v`

---

## Task 3.4 — Navigation inside the window

**File:** `modes/tty/nav.go` (MODIFY), `modes/tty/severe.go` (`handleSevereNav`)

**Test first (RED):** `modes/tty/severe_test.go` (append)
```go
func TestSevereNavTabsAndRows(t *testing.T) {
	m := dash(t)
	rows := []SevereRow{{Key: "a", Tab: SevereWarnings}, {Key: "b", Tab: SevereWarnings}, {Key: "c", Tab: SevereQuakes}}
	m, _ = m.Update(SevereMsg{Gen: 1, Rows: rows})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if d := m.(Dashboard); d.severeRow != 1 {
		t.Fatalf("down: row %d", d.severeRow)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // clamps at the last row
	if d := m.(Dashboard); d.severeRow != 1 {
		t.Fatalf("down past the end: row %d", d.severeRow)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if d := m.(Dashboard); d.severeTab != SevereWatches || d.severeRow != 0 {
		t.Fatalf("right: tab %v row %d", d.severeTab, d.severeRow)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft}) // wraps to Tropical
	if d := m.(Dashboard); d.severeTab != SevereTropical {
		t.Fatalf("left wraps: %v", d.severeTab)
	}
}
```

**Code — `nav.go`** in `handleNav`'s modal switch add `case modalSevere: return d.handleSevereNav(act)`.
**Code — `severe.go`:**
```go
// handleSevereNav: left/right change the tab (wrapping, focus to the top),
// up/down move the focused row in the table or scroll the record.
func (d Dashboard) handleSevereNav(act term.Action) Dashboard {
	n := int(severeNumTabs)
	switch act {
	case "alert-prev":
		if !d.severeDetail {
			d.severeTab, d.severeRow, d.modalScroll = SevereTab((int(d.severeTab)+n-1)%n), 0, 0
		}
	case "alert-next":
		if !d.severeDetail {
			d.severeTab, d.severeRow, d.modalScroll = SevereTab((int(d.severeTab)+1)%n), 0, 0
		}
	case "nav-up":
		if d.severeDetail {
			d.modalScroll = max(0, d.modalScroll-1)
		} else if d.severeRow > 0 {
			d.severeRow--
		}
	case "nav-down":
		if d.severeDetail {
			d.modalScroll = min(d.modalScroll+1, max(0, len(d.modalLines())-d.modalMax()))
		} else if d.severeRow < len(d.severeRows(d.severeTab))-1 {
			d.severeRow++
		}
	}
	return d
}
```
(import `github.com/branden-thompson/watchpost/platform/term` in `severe.go`.)

**Verify:** `go test ./modes/tty -run 'TestSevereNav' -v`

---

## Task 3.5 — The browse renderer

**File:** `modes/tty/severe.go` (renderers), `modes/tty/view.go` (`modalView`, `modalWidth`, `modalLines` cases)

**Test first (RED):** `modes/tty/severe_test.go` (append)
```go
func severeFixture(t *testing.T, w, h int, ascii bool) tea.Model {
	t.Helper()
	m, err := NewDashboard(Config{Version: "test", ASCII: ascii})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: w, Height: h})
	var rows []SevereRow
	products := []string{"Extreme Heat Warning", "Tornado Warning", "Flash Flood Warning", "Severe Thunderstorm Warning", "Gale Warning", "Red Flag Warning", "Flood Warning", "Special Marine Warning", "High Wind Warning"}
	places := []string{"Wicomico, MD", "Olathe, KS", "Palomar Mountain, CA", "San Diego, CA", "Cape Cod Bay, MA", "Kern County Mtns, CA", "Wakefield, VA", "Chesapeake Bay, MD", "Laramie, WY"}
	for i := range products {
		sev := TickerOrange
		if i == 1 {
			sev = TickerRed
		}
		rows = append(rows, SevereRow{Key: fmt.Sprint(i), Tab: SevereWarnings, Product: products[i], Location: places[i], Declared: "08/28 11:20 EDT", Expires: "08/28 20:00 EDT", Severity: sev,
			Record: SevereRecord{Title: strings.ToUpper(products[i]), Meta: "[Extreme · Immediate · Observed]", Timing: "Declared 08/28 08:45 CDT   Expires 08/28 09:00 CDT   (~15m)", Area: "Area: Johnson County, KS · NWS Kansas City", Paras: []string{"At 845 AM CDT, a severe thunderstorm capable of producing a tornado was located near Olathe, moving northeast at 30 mph.", "Instructions: TAKE COVER NOW!"}}})
	}
	model, _ = model.Update(SevereMsg{Gen: 1, Rows: rows, Totals: [severeNumTabs]int{9}, Updated: time.Date(2026, 8, 28, 15, 38, 5, 0, time.UTC)})
	model, _ = model.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	return model
}

func TestSevereBrowseFramesMatchTheMocksAtEveryWidth(t *testing.T) {
	for _, w := range []int{80, 100, 120} {
		frame := stripANSITest(severeFixture(t, w, 44, false).View().Content)
		if !strings.Contains(frame, "SEVERE WEATHER / DISASTER EVENTS") || !strings.Contains(frame, "Warnings — 9 active") {
			// the ASCII variant reads "Warnings - 9 active" (TestSevereBrowseASCIIHasNoGlyphs covers it)
			t.Fatalf("%d cols: header/category line missing", w)
		}
		if !strings.Contains(frame, "› Warnings") && !strings.Contains(frame, "›Warnings") && !strings.Contains(frame, "›Warn") {
			t.Fatalf("%d cols: the open tab is not marked", w)
		}
		for _, line := range strings.Split(frame, "\n") {
			if render.Width(strings.TrimRight(line, " ")) > w {
				t.Fatalf("%d cols: a line overflows the terminal: %q", w, line)
			}
		}
		if w >= 120 && !strings.Contains(frame, "E X P I R E S") {
			t.Fatal("120: EXPIRES column missing")
		}
		if w == 100 && (strings.Contains(frame, "E X P I R E S") || !strings.Contains(frame, "D E C L A R E D")) {
			t.Fatal("100: EXPIRES must drop first and DECLARED stay")
		}
		if w == 80 && strings.Contains(frame, "D E C L A R E D") {
			t.Fatal("80: DECLARED must drop")
		}
		if !strings.Contains(frame, "9 Total Category Events") || !strings.Contains(frame, "[enter] Event Details") {
			t.Fatalf("%d cols: footer missing", w)
		}
	}
}

func TestSevereRailIsOneColumn(t *testing.T) {
	frame := stripANSITest(severeFixture(t, 120, 44, false).View().Content)
	col := -1
	for _, line := range strings.Split(frame, "\n") {
		for _, g := range []string{"▲", "│", "█", "▼"} {
			if i := strings.LastIndex(line, g); i >= 0 && strings.HasSuffix(strings.TrimRight(line, " │"), g) || (i >= 0 && strings.Count(line, "│") >= 2 && g != "│") {
				if col < 0 {
					col = render.Width(line[:i])
				} else if render.Width(line[:i]) != col {
					t.Fatalf("rail glyph %q at column %d, expected %d: %q", g, render.Width(line[:i]), col, line)
				}
			}
		}
	}
	if col < 0 {
		t.Fatal("no rail found")
	}
}

func TestSevereBrowseASCIIHasNoGlyphs(t *testing.T) {
	frame := stripANSITest(severeFixture(t, 120, 44, true).View().Content)
	start := strings.Index(frame, "SEVERE WEATHER")
	for _, r := range frame[start:] {
		if r > 127 && r != '·' { // the middot is typography the whole app keeps under --ascii (the ticker bullet is already substituted)
			t.Fatalf("non-ASCII glyph %q in --ascii frame", r)
		}
	}
	if !strings.Contains(frame, "> Warnings") || !strings.Contains(frame, "[up/down] Navigate") {
		t.Fatal("ASCII forms missing")
	}
}

func TestSevereEmptyState(t *testing.T) {
	m := dash(t)
	m, _ = m.Update(SevereMsg{Gen: 1})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight}) // Advisories
	frame := stripANSITest(m.View().Content)
	if !strings.Contains(frame, "Advisories — no active events") || !strings.Contains(frame, "tracks your watchlist") {
		t.Fatalf("empty state: %s", frame)
	}
}
```

**Design ruling (plan reviewer):** `platform/render/table.go:3` records that it is "the ONLY go-studs
consumer in the app". The severe table therefore lives beside `LocationTable` in `platform/render` as
`SevereTable`, taking plain cells — `modes/tty` composes chrome and never imports the kit. `railify`
(`body.go:293`) hard-codes `│`/`█` and its callers draw `▲`/`▼`; the window needs `--ascii` forms, so a
glyph-aware rail helper is added to `render` and **`railify` becomes a call to it** (one owner; the RECENT
table gains ASCII rail glyphs for free — FR-13). Upstream candidate logged in
`third_party/go-studs/LOCAL_CHANGES.md`: "DataTable: per-row prefix/mark column + cursor index".

**Code — `platform/render/severe_table.go`** (CREATE):
```go
package render

// severe_table.go — the severe-events window's browse table (0.13.0): the
// go-studs DataTable owns the data columns (the LocationTable pattern, UAT
// ruling: use the component structure); the marks prefix and the row number
// ride the first two columns; header cells are bracketed spread bands.

import (
	"fmt"
	"strings"

	studs "github.com/branden-thompson/watchpost/third_party/go-studs/components"
)

// SevereCell is one browse row's cells (already Plain'd by the caller).
type SevereCell struct {
	Product, Location, Declared, Expires string
	Red                                  bool // the red tier wears a double glyph (FR-15)
}

// Browse column widths (plan §3.5): marks 7 · num 5 · EVENT fill · LOCATION 22 ·
// DECLARED 15 · EXPIRES 15, gutters 2; EXPIRES drops first, then DECLARED, then
// LOCATION narrows, so EVENT keeps ≥ 22 cells (NFR-11).
const (
	severeMarksW   = 7
	severeNumW     = 5
	severeLocW     = 22
	severeLocMinW  = 16
	severeDateW    = 15
	severeGutter   = 2
	severeMinEvent = 22 // EVENT keeps at least this on every degrade rung (NFR-11)…
	severeMinEventFloor = 14 // …except below the last rung, where it takes what is left
)

type severeCol struct {
	name  string
	width int
}

// severeColumns picks the data columns that fit inner cells: one degrade
// ladder, widest first — full · no EXPIRES · no DECLARED · narrow LOCATION —
// the first rung that leaves EVENT ≥ severeMinEvent wins; below the last rung
// EVENT takes what is left (a sub-40-column terminal). fixed() omits the gutter
// before the first data column because the kit draws none before columns 0-2.
func severeColumns(inner int) (cols []severeCol, event int) {
	ladder := [][]severeCol{
		{{"LOCATION", severeLocW}, {"DECLARED", severeDateW}, {"EXPIRES", severeDateW}},
		{{"LOCATION", severeLocW}, {"DECLARED", severeDateW}},
		{{"LOCATION", severeLocW}},
		{{"LOCATION", severeLocMinW}},
	}
	fixed := func(cs []severeCol) int {
		n := severeMarksW + severeNumW
		for _, c := range cs {
			n += severeGutter + c.width
		}
		return n
	}
	for _, rung := range ladder {
		if inner-fixed(rung) >= severeMinEvent {
			return rung, inner - fixed(rung)
		}
	}
	last := ladder[len(ladder)-1]
	return last, max(severeMinEventFloor, inner-fixed(last))
}

// SevereTable renders the header band row and rows[lo:hi] at width cells;
// focus is the focused row's index (or -1). Rows are numbered from 1.
func (o Opts) SevereTable(rows []SevereCell, focus, lo, hi, width int) string {
	cols, eventW := severeColumns(width)
	g := o.Glyphs()
	defs := []studs.ColumnDefinition{
		{Name: "marks", Width: severeMarksW},
		{Name: "num", Width: severeNumW}, // the kit suppresses gutters before columns 0-2 (data_table_row.go:64-77) — severeColumns' fixed() relies on that same rule
		{Name: "event", Header: bracketSpread("EVENT", eventW), Width: eventW, Truncatable: true, TruncatedMinWidth: 14, TruncationTail: "…"},
	}
	for _, c := range cols {
		defs = append(defs, studs.ColumnDefinition{Name: c.name, Header: bracketSpread(c.name, c.width), Width: c.width, Truncatable: c.name == "LOCATION", TruncatedMinWidth: severeLocMinW, TruncationTail: "…"})
	}
	if o.ASCII {
		for i := range defs {
			defs[i].TruncationTail = "~"
		}
	}
	def := &studs.DataTableDefinition{Columns: defs, GutterWidth: severeGutter, NoAutoStyle: true}
	for i := lo; i < hi && i < len(rows); i++ {
		r := rows[i]
		data := []string{severeMarks(g, i == focus, r.Red), fmt.Sprintf("%03d.", i+1), r.Product}
		for _, c := range cols {
			switch c.name {
			case "LOCATION":
				data = append(data, r.Location)
			case "DECLARED":
				data = append(data, r.Declared)
			case "EXPIRES":
				data = append(data, r.Expires)
			}
		}
		styles := map[int]string{2: Tok(TableName)}
		if i == focus {
			styles[2] = Tok(FocusName)
		}
		def.Rows = append(def.Rows, studs.EnhancedTableRow{Data: data, CellStyles: styles})
	}
	dt := studs.NewDataTable(width, def)
	out := []string{strings.TrimRight(dt.Header(), " ")}
	for _, line := range dt.Rows() {
		out = append(out, strings.TrimRight(line, " "))
	}
	return strings.Join(out, "\n")
}

// severeMarks is the 7-cell prefix: pointer(0) · focus mark(3) · severity(5-6).
func severeMarks(g Glyphs, focused, red bool) string {
	m := []string{" ", " ", " ", " ", " ", " ", " "}
	if focused {
		m[0] = Tint(g.Pointer, Tok(FocusPointer))
		m[3] = Tint(g.Play, Tok(FocusPointer))
	}
	if red {
		m[5], m[6] = Tint(g.Alert, Tok(AlertDanger)), Tint(g.Alert, Tok(AlertDanger)) // ⚠⚠ / !! for the red tier (FR-15)
	} else {
		m[5] = Tint(g.Alert, Tok(AlertLabel))
	}
	return strings.Join(m, "")
}

// bracketSpread is the header band: "[  E V E N T  ]" fitted to w cells,
// unspreading when narrow (the centered/bracketTitle idiom).
func bracketSpread(name string, w int) string {
	label := strings.Join(strings.Split(name, ""), " ")
	inner := w - 2
	if displayWidth(label) > inner {
		label = name
	}
	if displayWidth(label) > inner {
		label = truncate(label, max(0, inner))
	}
	pad := inner - displayWidth(label)
	return "[" + strings.Repeat(" ", pad/2) + label + strings.Repeat(" ", pad-pad/2) + "]"
}

// RailGlyphs are a scroll rail's marks: top, bar, thumb, bottom.
type RailGlyphs struct{ Top, Bar, Thumb, Bottom string }

// Rail resolves the rail glyphs for the options (--ascii: ^ | # v).
func (o Opts) Rail() RailGlyphs {
	if o.ASCII {
		return RailGlyphs{"^", "|", "#", "v"}
	}
	return RailGlyphs{"▲", "│", "█", "▼"}
}

// Railify appends a scroll rail to each line of a table: the bar on every
// line with the thumb tracking lo/total over window rows (UAT 11.3). The
// caller draws Top on the line above and Bottom on the line below.
func Railify(table string, width, lo, total, window int, g RailGlyphs) string {
	lines := strings.Split(table, "\n")
	thumb := 0
	if maxLo := total - window; maxLo > 0 && window > 1 {
		thumb = lo * (window - 1) / maxLo
	}
	for i := range lines {
		glyph := g.Bar
		if i == thumb {
			glyph = g.Thumb
		}
		lines[i] = PadTo(lines[i], width-1) + glyph // PadTo, not PadBetween: a full-width row must not push the rail (UAT 6.6)
	}
	return strings.Join(lines, "\n")
}
```
**Code — `modes/tty/body.go`:** `railify` becomes `return render.Railify(table, width, lo, total, window,
d.opts().Rail())` — it needs the options; change its signature to a method
`func (d Dashboard) railify(table string, width, lo, total, window int) string` and update its one caller
(`body.go:262`); the `▲`/`▼` the RECENT section writes at `:232`/`:264` — and the empty-state `▼` at `:247` — become `o.Rail().Top`/`.Bottom`.

**Code — `view.go`:** `modalView` add `case modalSevere: return d.severeModal(o)`; `modalWidth` add
`case modalSevere: return stretch(110) // the browse mock (plan §5); narrower terminals degrade columns`;
`modalLines` add `case modalSevere: raw = d.severeLines(d.opts())`.

**Code — `modes/tty/severe.go` (renderers; add imports `fmt`, `strings`, `platform/term`):**
```go
// hz is the title rule glyph (─, or - under --ascii); dash the category
// line's separator (—, or -).
func hz(o render.Opts) string {
	if o.ASCII {
		return "-"
	}
	return "─"
}

func dash(o render.Opts) string {
	if o.ASCII {
		return "-"
	}
	return "—"
}

// severeTitleChrome is the panel's title chrome: "┌── " + " " + rule gaps + " ┐" = 10 cells (PanelColored).
const severeTitleChrome = 10

// severeTitle composes the window title: name, an optional "Tab · n / N"
// crumb, a rule, and the Updated stamp (the detailsModal idiom).
func (d Dashboard) severeTitle(o render.Opts, w int) string {
	title := render.Tint("SEVERE WEATHER / DISASTER EVENTS", render.Tok(render.ModalTitle))
	if d.severeDetail {
		rows := d.severeRows(d.severeTab)
		title += " " + render.Tint(fmt.Sprintf("%s %s · %d / %d", strings.Repeat(hz(o), 3), severeTabs()[d.severeTab].Label, d.severeRow+1, len(rows)), render.Tok(render.ModalTitle))
	}
	stamp := "Updated " + d.severe.Updated.Local().Format("01/02/2006 15:04:05 MST")
	if d.severe.Updated.IsZero() {
		stamp = "Updated --"
	}
	fill := w - severeTitleChrome - render.Width(title) - render.Width(stamp)
	if fill > 1 {
		title += " " + strings.Repeat(hz(o), fill) + " " + render.Tint(stamp, render.Tok(render.ModalTitle))
	}
	return title
}

// severeModal renders the window: the browse table or the focused record, on
// the open tab's category tint (SAM-D-7) with the [A] modal's chrome (R-7).
func (d Dashboard) severeModal(o render.Opts) string {
	fg, bg := render.CategoryTone(severeTabs()[d.severeTab].Tone, d.darkBG)
	w := min(o.Width, d.modalWidth())
	title := d.severeTitle(o, w)
	if d.severeDetail {
		return d.floatModalToned(o, w, title, d.severeDetailLines(o), fg, bg)
	}
	o.Width = w // the browse view composes its own body: fixed tab row, category line, footer; the table rows carry the rail
	return o.Block(o.PanelColored(title, strings.Join(d.severeBrowseLines(o), "\n"), ""), fg, bg)
}

// severeLines is the record body for the scroll bounds (the browse view
// windows its own table and never scrolls the panel).
func (d Dashboard) severeLines(o render.Opts) []string {
	if d.severeDetail {
		return d.severeDetailLines(o)
	}
	return nil
}

// severeTabRow renders the six tabs, degrading spaced → tight → short so the
// row always fits (the alertCompactLine idiom); the open tab wears the pointer
// glyph, bold, and its tint (glyph carries state, colour carries category).
func (d Dashboard) severeTabRow(o render.Opts, inner int) string {
	g := o.Glyphs()
	tabs := severeTabs()
	label := func(form int, i int, t severeTab) string {
		open := SevereTab(i) == d.severeTab
		switch form {
		case 0:
			if open {
				return "[ " + g.Pointer + " " + t.Label + " ]"
			}
			return "[ " + t.Label + " ]"
		case 1:
			if open {
				return "[" + g.Pointer + t.Label + "]"
			}
			return "[" + t.Label + "]"
		}
		if open {
			return "[" + g.Pointer + t.Short + "]"
		}
		return "[" + t.Short + "]"
	}
	for form := 0; form < 3; form++ {
		plain := make([]string, len(tabs))
		for i, t := range tabs {
			plain[i] = label(form, i, t)
		}
		if render.Width(strings.Join(plain, " ")) <= inner || form == 2 {
			for i := range plain {
				if SevereTab(i) == d.severeTab {
					plain[i] = render.TintRaw(plain[i], "1;"+render.Tok(render.AlertModalText)+";"+render.Tok(tabs[i].Tone)) // the UNMIXED tint, so the chip stands off the mixed panel (A11y Y4)
				}
			}
			return strings.Join(plain, " ")
		}
	}
	return ""
}

// severeBrowseLines composes the browse body: tab row, category line, the
// table window with its rail, the total and the chips.
func (d Dashboard) severeBrowseLines(o render.Opts) []string {
	inner := o.Width - 4 // panel chrome; the rail lives inside the table lines
	rows := d.severeRows(d.severeTab)
	tab := severeTabs()[d.severeTab]
	total := d.severe.Totals[d.severeTab]
	totalLine := fmt.Sprintf("%d Total Category Events", total)
	lines := []string{"", "  " + d.severeTabRow(o, inner-2), ""}
	if len(rows) == 0 {
		lines = append(lines, "  "+tab.Label+" "+dash(o)+" no active events", "",
			"  No active "+strings.ToLower(tab.Label)+" events · "+d.severe.Updated.Local().Format("Updated 01/02 15:04 MST"))
		if tab.WatchlistHint && d.numPriority() == 0 {
			lines = append(lines, "  (tracks your watchlist — add locations with ctrl+a)")
		}
		return append(lines, "", render.PadTo("", inner-len(totalLine)-3)+totalLine, "", "  "+d.severeChips(o))
	}
	cat := fmt.Sprintf("  %s %s %d active", tab.Label, dash(o), len(rows))
	if total > len(rows) {
		cat += fmt.Sprintf(" · showing %d of %d", len(rows), total)
	}
	if down := d.severeDownSources(); down != "" {
		cat += " · " + down // "NWS unavailable" — a dead source is stated, never hidden (red-team PLAN A-1/B-7)
	}
	lines = append(lines, cat)
	// The table: the header line wears the rail's Top, the rows its bar/thumb,
	// the total line its Bottom (the RECENT idiom, body.go).
	fixedTop, fixedBottom := len(lines), 3
	win := max(2, d.modalMax()-fixedTop-fixedBottom-1) // rows visible under the header
	lo := 0
	if d.severeRow >= win {
		lo = d.severeRow - win + 1
	}
	hi := min(len(rows), lo+win)
	cells := make([]render.SevereCell, len(rows))
	for i, r := range rows {
		cells[i] = render.SevereCell{Product: render.Plain(r.Product), Location: render.Plain(r.Location), Declared: render.Plain(r.Declared), Expires: render.Plain(r.Expires), Red: r.Severity == TickerRed}
	}
	// The rail column is inner-1 (0-based) on EVERY line: Top on the header,
	// bar/thumb on the rows, Bottom on the total line (red-team PLAN-CQ #4 —
	// TestSevereRailIsOneColumn pins the alignment; a golden alone would freeze a skew).
	rail := o.Rail()
	railCol := inner - 1
	table := strings.SplitN(o.SevereTable(cells, d.severeRow, lo, hi, railCol-2), "\n", 2)
	lines = append(lines, render.PadTo(table[0], railCol)+rail.Top)
	if len(table) > 1 {
		lines = append(lines, strings.Split(render.Railify(table[1], inner, lo, len(rows), win, rail), "\n")...)
	}
	lines = append(lines, render.PadTo(totalLine, railCol)[max(0, railCol-len(totalLine)-2):]+rail.Bottom, "", "  "+d.severeChips(o))
	return lines
}

// severeDownSources names the feeds whose last fetch failed ("NWS, NHC unavailable"), "" when all are up.
func (d Dashboard) severeDownSources() string {
	var down []string
	for _, s := range d.severe.Sources {
		if !s.OK {
			down = append(down, s.Name)
		}
	}
	if len(down) == 0 {
		return ""
	}
	return strings.Join(down, ", ") + " unavailable"
}

// severeChips is the browse footer; the Event Details chip mutes when there
// is no row to open (a dead control is never announced live — A11y Y3).
func (d Dashboard) severeChips(o render.Opts) string {
	hasRows := len(d.severeRows(d.severeTab)) > 0
	nav, cat := "↑↓", "←→"
	if o.ASCII {
		nav, cat = "up/down", "left/right"
	}
	return o.KeyCap(nav) + " Navigate  " + o.KeyCap(cat) + " Category  " + o.KeyCapIf("enter", hasRows) + " Event Details  " + o.KeyCap("esc") + " Close"
}
```
Update `TestSevereBrowseASCIIHasNoGlyphs` to also allow `—` in "Updated --"? No — the ASCII stamp uses `--`;
the only non-ASCII runes an `--ascii` frame keeps are `—` (category line) and `·` (separators), as the test
states; the truncation tail is `~` under ASCII (set above).

**Worked line budget (red-team PLAN B-8) — 120×44 terminal:** `modalMax() = max(5, 44-12) = 32` body lines.
Fixed top: blank · tab row · blank · category line = **4**. Table: header (wears `▲`) + `win` rows.
Fixed bottom: total line (wears `▼`) · blank · chips = **3**. So `win = 32 - 4 - 3 - 1 = 24` rows visible;
with 9 rows all show (`lo=0, hi=9`, rail thumb at row 0); with 500 rows and the focus on row 30,
`lo = 30-24+1 = 7`, `hi = 31`, the thumb sits at `7*(24-1)/(500-24) = 0` → first row. At 80×24:
`modalMax = 12`, `win = 12-4-3-1 = 4`. The mocks in plan §5 show 6 rows because the generator used
`maxl = 14` — the real height comes from the terminal; the goldens (Task 3.8) pin the real values.

**Verify:** `go test ./modes/tty ./platform/render -run 'TestSevereBrowse|TestSevereEmpty|TestRail' -v`

---

## Task 3.6 — The detail renderer

**File:** `modes/tty/severe.go` (append)

**Test first (RED):** `modes/tty/severe_test.go` (append)
```go
func TestSevereDetailShowsTheRecordAndItsChips(t *testing.T) {
	m := severeFixture(t, 120, 44, false)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	frame := stripANSITest(m.View().Content)
	for _, want := range []string{"Warnings · 2 / 9", "TORNADO WARNING", "[Extreme · Immediate · Observed]", "Declared 08/28 08:45 CDT", "Area: Johnson County, KS", "Instructions: TAKE COVER NOW!", "[esc] Back", "[esc esc] Close"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("detail frame lacks %q:\n%s", want, frame)
		}
	}
}
```

**Code:**
```go
// severeDetailLines is the focused row's record in the [A] shape: bold title
// + meta, timing, area, paragraphs in AlertModalText, then the chips.
func (d Dashboard) severeDetailLines(o render.Opts) []string {
	rows := d.severeRows(d.severeTab)
	if len(rows) == 0 || d.severeRow >= len(rows) {
		return []string{"", "  No event selected.", "", "  " + o.KeyCap("esc") + " Back"}
	}
	rec := rows[d.severeRow].Record
	wrapW := min(o.Width, d.modalWidth()) - 9 // breathing room beside the rail (UAT 23.3)
	text := render.Tok(render.AlertModalText)
	// Plain at the point of use — the domain already stripped these; this is
	// the TTY's own last line of defence should a mapping ever bypass RecordOf.
	lines := []string{"", "  " + render.TintRaw(render.Plain(rec.Title), "1;"+text) + "  " + render.Tint(render.Plain(rec.Meta), text)}
	if rec.Timing != "" {
		lines = append(lines, "  "+render.Tint(render.Plain(rec.Timing), text))
	}
	if rec.Area != "" {
		for _, l := range render.WrapText(render.Plain(rec.Area), wrapW) {
			lines = append(lines, "  "+render.Tint(l, text))
		}
	}
	for _, p := range rec.Paras {
		lines = append(lines, "")
		for _, l := range formatAlertBody(render.Plain(p), wrapW) { // the [A] bullet rules (UAT 65/66)
			lines = append(lines, render.Tint(l, text))
		}
	}
	lines = append(lines, "", "  "+o.KeyCap("esc")+" Back  "+o.KeyCap("esc esc")+" Close  "+o.KeyCap("↑↓")+" Scroll")
	return lines
}
```
(`formatAlertBody` is `alerts.go:54`; it wraps prose and `* ` bullets at the text width — the record's
paragraphs are already `Plain`'d by the domain.)

**Verify:** `go test ./modes/tty -run 'TestSevereDetail' -v`

---

## Task 3.7 — The modal memo (family)

**Files:** `modes/tty/memo.go`, `modes/tty/view.go`, `modes/tty/dashboard.go` (`memo` allocation), `modes/tty/memo_test.go`

**Test first (RED):** `modes/tty/memo_test.go` (append)
```go
// The modal memo (FR-10): the open window's overlay is rebuilt only when an
// input changes. Positive control (red-team P4): a loading row must NOT miss
// the modal memo (shimmer is not in its key), and a frozen clock must hit.
func TestModalMemoHitsAcrossTicksWithALoadingRow(t *testing.T) {
	m := severeFixture(t, 120, 44, false).(Dashboard)
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return now }
	loading := &snapshot.Snapshot{Locations: []snapshot.Location{{Label: "Loading City"}}} // rowLoading == true
	m.recent = loading
	_ = m.View().Content
	h0, m0 := m.modalMemoCounts()
	for i := 0; i < 20; i++ {
		var mm tea.Model = m
		mm, _ = mm.Update(tickMsg{})
		m = mm.(Dashboard)
		_ = m.View().Content
	}
	h1, m1 := m.modalMemoCounts()
	if m1-m0 != 0 || h1-h0 != 20 {
		t.Fatalf("20 ticks: %d misses, %d hits (want 0 / 20)", m1-m0, h1-h0)
	}
	now = now.Add(time.Minute) // the minute bucket invalidates Status/Details ages
	_ = m.View().Content
	if _, m2 := m.modalMemoCounts(); m2-m1 != 1 {
		t.Fatal("a new minute must miss once")
	}
}

func TestEveryModalRendersByteIdenticalTwice(t *testing.T) {
	for name, open := range openers {
		m := modalFixture(t).(Dashboard)
		fixed := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
		m.now = func() time.Time { return fixed }
		var mm tea.Model = m
		mm, _ = mm.Update(open)
		d := mm.(Dashboard)
		o := d.opts()
		a := d.renderModal(o) // bypass the memo slot: an impurity lives in the renderer, and the slot would hide it
		b := d.renderModal(o)
		if a != b {
			t.Errorf("%s: two renders differ with a frozen clock (a hidden impurity)", name)
		}
	}
}
```
(`openers` must include `"severe"` — Task 3.8.)

**Code — `memo.go`:** append
```go
// modalKey is every input any open window's renderer reads (FR-10, the same
// discipline as bodyKey): one field per input; adding an input to a *Lines()
// renderer means adding a field here and a row in TestModalMemoInvalidates.
// Deliberately NOT keyed on the loading shimmer (red-team P4) — no modal shows
// it — and keyed on the MINUTE, not the tick, for the "N min ago" labels.
type modalKey struct {
	modal          modal
	width, height  int
	modalScroll    int
	selected       int
	alertIdx       int
	snap, recent   *snapshot.Snapshot
	severeGen      uint64
	severeTab      SevereTab
	severeRow      int
	severeDetail   bool
	addMode        string
	addQuery       string
	addErr         string
	setup          setupKey // a comparable projection of setupState (it holds a slice)
	themeIdx       int
	voiceIdx       int
	voiceErr       string
	voiceNote      string
	themeErr       string
	radioVoice     string
	nvoices        int // the Voice chooser's list length (the list is a slice)
	opts           render.Opts // the options handed to renderModal — Width/Units/Frame/ASCII/ThinBands all key (comparable struct)
	darkBG         bool
	theme          uint64
	minute         int64 // now.Unix()/60 — Status ages, Details "N min ago"
	statsGen       uint64
}

// setupKey is setupState without its slice: every field the Setup window
// renders, plus the hint count (the hints themselves are derived from query).
type setupKey struct {
	focus    setupFocus
	query    string
	nhints   int
	idx      int
	ref      snapshot.LocationRef // zero when none
	key      string
	reveal   bool
	filtered bool
	radiusMi string
	err      string
}

func setupKeyOf(s setupState) setupKey {
	k := setupKey{focus: s.focus, query: s.query, nhints: len(s.hints), idx: s.idx, key: s.key, reveal: s.reveal, filtered: s.filtered, radiusMi: s.radiusMi, err: s.err}
	if s.ref != nil {
		k.ref = *s.ref
	}
	return k
}

// modalMemo is the single slot for the composed overlay.
type modalMemo struct {
	mu           sync.Mutex
	ok           bool
	key          modalKey
	out          string
	hits, misses int
}

// modalKeyFor derives the key from the model and the frame's options.
func (d Dashboard) modalKeyFor(o render.Opts) modalKey {
	return modalKey{
		modal: d.modal, width: d.width, height: d.height, modalScroll: d.modalScroll, selected: d.selected, alertIdx: d.alertIdx,
		snap: d.snap, recent: d.recent, severeGen: d.severe.Gen, severeTab: d.severeTab, severeRow: d.severeRow, severeDetail: d.severeDetail,
		addMode: d.addMode, addQuery: d.addQuery, addErr: d.addErr, setup: setupKeyOf(d.setup), themeIdx: d.themeIdx, voiceIdx: d.voiceIdx,
		voiceErr: d.voiceErr, voiceNote: d.voiceNote, themeErr: d.themeErr, radioVoice: d.radioVoice, nvoices: len(d.voiceList),
		opts: o, darkBG: d.darkBG, theme: render.ThemeGeneration(), minute: d.now().Unix() / 60, statsGen: d.statsGen,
	}
}

// modalMemoCounts reports the slot's counters (tests and the dump).
func (d Dashboard) modalMemoCounts() (hits, misses int) {
	if d.mmemo == nil {
		return 0, 0
	}
	d.mmemo.mu.Lock()
	defer d.mmemo.mu.Unlock()
	return d.mmemo.hits, d.mmemo.misses
}
```
(`setupState.hints` is a slice — `setup.go:40-51` — hence the `setupKey` projection above; `snapshot.LocationRef`
is a struct of strings and floats, comparable.)

**The clocks the key must see (red-team PLAN P2):** `detail.go:55`, `:58`, `:60` call `time.Now()` directly;
route all three through `d.now()` (the field exists, `dashboard.go:291`) so the minute bucket is the ONLY
clock a renderer reads — otherwise the frozen-clock tests pass while the live modal reads an unkeyed input.

**`statsGen` (red-team PLAN P3):** the [S] window's counters change only when a request or publish lands.
Add `statsGen uint64` and `statsFP [3]int64` to `Dashboard`; in `applyTick` when `d.modal == modalStatus`:
```go
	if d.cfg.Stats != nil {
		st := d.cfg.Stats()
		var attempts int64
		for _, h := range st.Requests.Hosts {
			attempts += h.Attempts
		}
		fp := [3]int64{attempts, st.Pipelines[0].Publishes + st.Pipelines[1].Publishes, int64(len(st.LastDump))}
		if fp != d.statsFP {
			d.statsFP, d.statsGen = fp, d.statsGen+1
		}
	}
```
so a quiet [S] window hits the memo and a landed request misses it once.

**Code — `dashboard.go`:** `mmemo *modalMemo // the modal memo's single slot (0.13.0, FR-10)`; allocate
`mmemo: &modalMemo{}` in `NewDashboard`.

**Code — `view.go`:** rename the existing `modalView` to `renderModal` and add:
```go
// modalView returns the open window's overlay from the memo when nothing it
// reads has changed (FR-10 — the per-tick rebuild an open window paid for
// every 300 ms); "" when none is open.
func (d Dashboard) modalView(o render.Opts) string {
	if d.modal == modalNone {
		return ""
	}
	key := d.modalKeyFor(o)
	m := d.mmemo
	if m == nil {
		return d.renderModal(o)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ok && m.key == key {
		m.hits++
		return m.out
	}
	out := d.renderModal(o)
	m.ok, m.key, m.out = true, key, out
	m.misses++
	return out
}
```
The invalidation table — one row per key field, each mutation must miss exactly once:
```go
func TestModalMemoInvalidatesOnEveryInput(t *testing.T) {
	base := func() Dashboard {
		d := severeFixture(t, 120, 44, false).(Dashboard)
		fixed := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
		d.now = func() time.Time { return fixed }
		_ = d.View().Content
		return d
	}
	cases := []struct {
		name string
		mut  func(*Dashboard)
	}{
		{"modal", func(d *Dashboard) { d.modal = modalHelp }},
		{"width", func(d *Dashboard) { d.width = 100 }},
		{"height", func(d *Dashboard) { d.height = 30 }},
		{"modalScroll", func(d *Dashboard) { d.severeDetail = true; d.modalScroll = 1 }},
		{"selected", func(d *Dashboard) { d.modal = modalDetails; d.selected = 1 }},
		{"alertIdx", func(d *Dashboard) { d.modal = modalAlerts; d.alertIdx = 1 }},
		{"snap", func(d *Dashboard) { d.snap = &snapshot.Snapshot{} }},
		{"recent", func(d *Dashboard) { d.recent = &snapshot.Snapshot{} }},
		{"severeGen", func(d *Dashboard) { d.severe.Gen++ }},
		{"severeTab", func(d *Dashboard) { d.severeTab = SevereWatches }},
		{"severeRow", func(d *Dashboard) { d.severeRow = 1 }},
		{"severeDetail", func(d *Dashboard) { d.severeDetail = true }},
		{"addQuery", func(d *Dashboard) { d.modal = modalAdd; d.addQuery = "x" }},
		{"setup", func(d *Dashboard) { d.modal = modalSetup; d.setup.query = "x" }},
		{"themeIdx", func(d *Dashboard) { d.modal = modalTheme; d.themeIdx = 1 }},
		{"voiceIdx", func(d *Dashboard) { d.modal = modalVoice; d.voiceIdx = 1 }},
		{"nvoices", func(d *Dashboard) { d.modal = modalVoice; d.voiceList = append(d.voiceList, "Z") }},
		{"units", func(d *Dashboard) { d.units = render.UnitC }},   // reaches the key through opts
		{"ascii", func(d *Dashboard) { d.cfg.ASCII = true }},      // likewise
		{"frame", func(d *Dashboard) { d.modal = modalDetails; d.frame++ }}, // opts.Frame: the Details LoadingDots phase
		{"darkBG", func(d *Dashboard) { d.darkBG = false }},
		{"theme", func(d *Dashboard) { render.SetTheme("Monochrome") }},
		{"minute", func(d *Dashboard) { d.now = func() time.Time { return time.Date(2026, 8, 28, 12, 1, 0, 0, time.UTC) } }},
		{"statsGen", func(d *Dashboard) { d.statsGen++ }},
	}
	for _, c := range cases {
		d := base()
		_, m0 := d.modalMemoCounts()
		c.mut(&d)
		_ = d.View().Content
		render.SetTheme(render.DefaultThemeName) // the theme case must not leak into the next
		if _, m1 := d.modalMemoCounts(); m1-m0 != 1 {
			t.Errorf("%s: %d misses after the change, want 1", c.name, m1-m0)
		}
	}
}
```
(the model is a value type; the memo slot is a pointer shared by copies, so `base()` returns a warmed
model whose next differing key misses exactly once.)

**Verify:** `go test ./modes/tty -run 'TestModalMemo|TestEveryModalRenders' -v`

---

## Task 3.8 — Exclusivity, markers, goldens

**Files:** `modes/tty/modal_test.go`, `modes/tty/severe_test.go`, `modes/tty/testdata/`

**Code:** in `modal_test.go` add `"severe": "SEVERE WEATHER / DISASTER EVENTS"` to `modalMarkers` and
`"severe": tea.KeyPressMsg{Code: 'w', Text: "w"}` to `openers`. Goldens (`golden_test.go:19` defines the
`-update-golden` flag as `updateGolden`):
```go
// TestSevereGoldens pins the browse frames at three widths and under --ascii,
// each with the width invariant beside the byte pin (a pin alone can freeze
// a defect — calibration "Byte Pins Ride With Invariant Assertions").
func TestSevereGoldens(t *testing.T) {
	cases := []struct {
		name  string
		w     int
		ascii bool
	}{{"severe-80x44", 80, false}, {"severe-100x44", 100, false}, {"severe-120x44", 120, false}, {"severe-120x44-ascii", 120, true}}
	for _, c := range cases {
		frame := severeFixture(t, c.w, 44, c.ascii).View().Content
		for _, line := range strings.Split(stripANSITest(frame), "\n") {
			if render.Width(strings.TrimRight(line, " ")) > c.w {
				t.Fatalf("%s: line wider than the terminal: %q", c.name, line)
			}
		}
		path := filepath.Join("testdata", c.name+".golden")
		if *updateGolden {
			if err := os.WriteFile(path, []byte(frame), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: run with -update-golden first: %v", c.name, err)
		}
		if string(want) != frame {
			t.Errorf("%s: frame differs from the golden (re-record deliberately with -update-golden)", c.name)
		}
	}
}
```
(record once: `go test ./modes/tty -run TestSevereGoldens -update-golden`; review the four files by eye
against plan §5 before committing them.)

**Verify:** `go test ./modes/tty -run 'TestExactlyOneModal|TestEveryModalHasItsMarker|TestSevere' -v`

---

## Task 3.9 — Budgets (NFR-2)

**File:** `modes/tty/bench_test.go` (MODIFY)

**Code:** add `"80x24": 0` placeholders to be **measured first** (the spike rule): run
`go test ./modes/tty -run TestFrameAllocBudget -v` after the memo lands, read the printed hit/miss for the
new `Severe` rows, and pin at measured × 1.05:
```go
// BenchmarkFrame_133x44_Severe is the severe window's frame (0.13.0).
func BenchmarkFrame_133x44_Severe(b *testing.B) {
	m := severeBench(b, 133, 44)
	b.ReportAllocs()
	for b.Loop() {
		_ = m.View().Content
	}
}

// TestSevereFrameAllocBudget pins the open window's hit and miss paths.
func TestSevereFrameAllocBudget(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts are measured without the race detector (make alloc-budget)")
	}
	for size, budget := range severeAllocBudget {
		var w, h int
		if _, err := fmt.Sscanf(size, "%dx%d", &w, &h); err != nil {
			t.Fatal(err)
		}
		d := severeBench(t, w, h).(Dashboard)
		_ = d.View().Content
		hit := testing.AllocsPerRun(20, func() { _ = d.View().Content })
		miss := testing.AllocsPerRun(20, func() { d.mmemo.ok = false; _ = d.View().Content })
		t.Logf("severe %s: hit %.0f (budget %.0f) · miss %.0f (budget %.0f)", size, hit, budget.hit, miss, budget.miss)
		if hit > budget.hit || miss > budget.miss {
			t.Errorf("severe %s: hit %.0f / miss %.0f exceed %.0f / %.0f", size, hit, miss, budget.hit, budget.miss)
		}
	}
}

// severeAllocBudget: measured at BUILD P3-9, then pinned at measured × 1.05.
// The provisional expectations live in ONE place — 03-architecture-design/plan.md §0;
// the zeros here fail loudly until the measurement is taken and written in.
var severeAllocBudget = map[string]struct{ hit, miss float64 }{
	"133x44": {0, 0}, // measure first (plan §0), then pin
	"80x24":  {0, 0},
}
```
```go
// severeBench is the window's benchmark fixture: the canonical dashboard plus
// a full 500-row index (Warnings-heavy, every 4th row red) with the window open.
func severeBench(tb testing.TB, w, h int) tea.Model {
	tb.Helper()
	m := benchDash(tb, w, h)
	var rows []SevereRow
	for i := 0; i < 500; i++ {
		sev := TickerOrange
		if i%4 == 0 {
			sev = TickerRed
		}
		rows = append(rows, SevereRow{Key: fmt.Sprintf("k%d", i), Tab: SevereTab(i % 3), Product: "Severe Thunderstorm Warning", Location: fmt.Sprintf("Benchmark County %02d, KS", i%60),
			Declared: "08/28 11:20 CDT", Expires: "08/28 13:00 CDT", Severity: sev,
			Record: SevereRecord{Title: "SEVERE THUNDERSTORM WARNING", Meta: "[Severe · Immediate · Observed]", Timing: "Declared 08/28 11:20 CDT   Expires 08/28 13:00 CDT   (~1h40m)", Area: "Area: Benchmark County, KS", Paras: []string{strings.Repeat("At 1120 AM CDT, a severe thunderstorm was located near Benchmark, moving northeast at 30 mph. ", 8), "Instructions: Move to an interior room."}}})
	}
	m, _ = m.Update(SevereMsg{Gen: 1, Rows: rows, Totals: [severeNumTabs]int{167, 167, 166}, Updated: time.Date(2026, 8, 28, 15, 38, 5, 0, time.UTC)})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	return m
}
```

Also add the missing baseline the budgets derive from (red-team PLAN P6) — `render.Overlay` runs outside
the memo on every open-modal frame, so the hit budget = closed hit + Overlay + `modalKeyFor`:
```go
// BenchmarkOverlayOnly isolates the compositor's cost (platform/render/panel.go:206).
func BenchmarkOverlayOnly(b *testing.B) {
	m := benchDash(b, 133, 44).(Dashboard)
	base := m.View().Content
	modal := m.floatModal(m.opts(), 76, "Overlay", []string{"one", "two", "three"})
	b.ReportAllocs()
	for b.Loop() {
		_ = render.Overlay(base, modal, 133)
	}
}
```

**Verify:** `make alloc-budget` (or `go test ./modes/tty -run 'AllocBudget' -v`); record the numbers in
the P3 build log; also confirm `BenchmarkFrame_133x44_Help` drops from 4 422 allocs to the hit path.

---

## Task 3.10 — The [S] gauge for the index (NFR-4)

**File:** `modes/tty/status.go` (MODIFY)

**Test first (RED):** `modes/tty/severe_test.go` (append)
```go
func TestStatusModalGaugesTheSevereIndex(t *testing.T) {
	m := severeFixture(t, 120, 44, false)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	if frame := stripANSITest(m.View().Content); !strings.Contains(frame, "severe index   9 rows / 500") {
		t.Fatalf("[S] lacks the severe gauge:\n%s", frame)
	}
}
```

**Code:** in `statusLines()` after the provider rows (read `status.go:22-47` for the row format helper the
file uses — every row is `fmt.Sprintf("  %-14s %s", label, value)` or the file's equivalent):
```go
	lines = append(lines, fmt.Sprintf("  %-14s %d rows / %d · gen %d", "severe index", len(d.severe.Rows), 500, d.severe.Gen)) // NFR-4: the retained index against its cap
```

**Verify:** `go test ./modes/tty -run 'TestStatusModalGauges' -v`

**Batch exit:** `make verify`; goldens updated deliberately (never blindly); commit
`feat(severe): the window — tokens, browse/detail, nav, modal memo, budgets`.
