package tty

// severe.go — the Severe Weather / Disaster Events window (0.13.0): UI types
// the app maps domains/severe onto (the snapshot-only rule — this package
// never imports the domain), the tab registry, the window's state helpers,
// its navigation, and the browse / detail renderers. Mocks:
// 03-architecture-design/plan.md §5 and 02-analysis/mocks/mock.py.

import (
	"fmt"
	"github.com/branden-thompson/watchpost/platform/category"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/term"
)

// SevereTab is the window's category, in the app's importance order.
type SevereTab = category.Category

// The window's tab names are the registry's categories. NOT a second ordering:
// these are aliases, so the two cannot drift — which they could when this was
// an enum of its own that had to agree with the domain's position by position
// (F-21).
const (
	SevereEmergency  = category.Emergency
	SevereWarnings   = category.Warnings
	SevereWatches    = category.Watches
	SevereAdvisories = category.Advisories
	SevereStatements = category.Statements
	SevereDisasters  = category.Disasters
	SevereMarine     = category.Marine
	SevereForecasts  = category.Forecasts
	severeNumTabs    = category.Count
)

// SevereMaxRows mirrors the domain's cap for the [S] gauge (the app asserts
// the two agree at compile time — this package cannot import the domain).
const SevereMaxRows = 500

// severeBreakingWindow: a breaking event within this window opens the window
// on its category (SAM-D-17); after it, Warnings.
const severeBreakingWindow = 10 * time.Minute

// SevereMsg publishes the severe-events index (sent by the app's severe deck
// only when the row set, a source's health or the fetch minute changed).
// Totals count every row per tab BEFORE the cap, so the window can say
// "showing N of M".
type SevereMsg struct {
	Rows    []SevereRow
	Totals  [severeNumTabs]int
	Updated time.Time // the newest SUCCESSFUL source fetch (dataAsOf), zero when none yet
	Sources []SevereSource
	Gen     uint64 // bumps per publish — the modal memo's key field
}

// SevereReadingMsg says which event the radio is reading ([space] in the
// window, UAT option B): the ▶ rides that row; Key "" when the read ends.
type SevereReadingMsg struct {
	Key string
	// Paused is a LISTENER'S hold, not a gap in the audio (MVS-D-74). The row
	// keeps its mark either way — a paused read is still the row's read — so
	// this is what tells the two apart on screen.
	Paused bool
}

// SevereSource is one feed's health as the window states it ("NWS unavailable").
type SevereSource struct {
	Name string
	OK   bool
}

// SevereRow is one event as the window lists it; the times arrive formatted in
// the right zone (FR-9) and the record composed (the app owns both).
type SevereRow struct {
	Key       string
	Tab       SevereTab
	Product   string // "Tornado Warning" · "Tropical Storm Dolly"
	Location  string
	Detection string // how the event was established: "Radar Indicated", "Observed", "Reviewed"; "" when not discernible
	Declared  string // "08/28 08:45 CDT"
	Expires   string // "" when none
	Severity  TickerSeverity
	Record    SevereRecord
}

// SevereRecord is the [A]-shaped record of one row.
type SevereRecord struct {
	Title, Meta, Timing, Area string
	Paras                     []string
}

// severeTabs is the window's view of the registry, in tab order. The rows are
// the registry's own (platform/category) — adding a category is adding it
// there, and nothing here.
func severeTabs() []category.Spec {
	out := make([]category.Spec, 0, category.Count)
	for _, c := range category.All() {
		out = append(out, category.Of(c))
	}
	return out
}

// --- state ---

// applySevere installs a published index. Focus follows the event's KEY, not
// its position (round-2 B-7): the focused row is re-found in the new set, and a
// focus whose event vanished returns to the top; an open record whose event
// vanished closes back to the table.
func (d Dashboard) applySevere(msg SevereMsg) Dashboard {
	var focusKey string
	if r := d.severeRowAt(d.severeRow); r != nil {
		focusKey = r.Key
	}
	d.severe = msg
	d.severeByTab = bucketSevere(msg.Rows)
	d.severeRow = 0
	found := false
	for i, idx := range d.severeByTab[d.severeTab] {
		if focusKey != "" && msg.Rows[idx].Key == focusKey {
			d.severeRow, found = i, true
			break
		}
	}
	if d.severeDetail && !found {
		d.severeDetail, d.modalScroll = false, 0
	}
	return d
}

// bucketSevere indexes the rows per tab, preserving the app's sort.
func bucketSevere(rows []SevereRow) [severeNumTabs][]int {
	var by [severeNumTabs][]int
	for i, r := range rows {
		if r.Tab >= 0 && r.Tab < severeNumTabs {
			by[r.Tab] = append(by[r.Tab], i)
		}
	}
	return by
}

// severeCount is the open tab's row count; severeRowAt its i-th row (nil out
// of range) — indexed through severeByTab, never copied (R3-B-11).
func (d Dashboard) severeCount() int {
	if d.severeTab < 0 || d.severeTab >= severeNumTabs {
		return 0
	}
	return len(d.severeByTab[d.severeTab])
}

func (d Dashboard) severeRowAt(i int) *SevereRow {
	if i < 0 || i >= d.severeCount() {
		return nil
	}
	if idx := d.severeByTab[d.severeTab][i]; idx < len(d.severe.Rows) {
		return &d.severe.Rows[idx]
	}
	return nil
}

// severeOpeningTab: the breaking event's category when one broke within the
// window, else Warnings.
func (d Dashboard) severeOpeningTab() SevereTab {
	if !d.lastBreaking.IsZero() && d.now().Sub(d.lastBreaking) <= severeBreakingWindow {
		return d.lastBreakingTab
	}
	return SevereWarnings
}

// openSevere opens the window on the table (w / ctrl+s); a second press
// closes it.
func (d Dashboard) openSevere() Dashboard {
	if d.modal == modalSevere {
		return d.close()
	}
	d = d.open(modalSevere)
	d.severeTab, d.severeRow, d.severeDetail = d.severeOpeningTab(), 0, false
	return d
}

// openSevereDetail replaces the table with the focused event's record (FR-4);
// a second enter inside the record is inert (its scroll stays — R3-B-06).
func (d Dashboard) openSevereDetail() Dashboard {
	if d.severeDetail {
		return d
	}
	if d.severeRow < d.severeCount() {
		d.severeDetail, d.modalScroll = true, 0
	}
	return d
}

// closeSevereDetail backs out to the table (esc); the second esc closes.
func (d Dashboard) closeSevereDetail() Dashboard {
	d.severeDetail, d.modalScroll = false, 0
	return d
}

// handleSevereNav: ←/→ browse categories (wrapping), ↑/↓ browse events; in
// the record ↑/↓ scroll it.
func (d Dashboard) handleSevereNav(act term.Action) Dashboard {
	if d.severeDetail {
		switch act {
		case "nav-up", "nav-down":
			return d.handleModalNav(act)
		}
		return d
	}
	switch act {
	case "nav-up":
		d.severeRow = max(0, d.severeRow-1)
	case "nav-down":
		d.severeRow = min(d.severeRow+1, max(0, d.severeCount()-1))
	case "alert-prev":
		d.severeTab, d.severeRow = (d.severeTab+severeNumTabs-1)%severeNumTabs, 0
	case "alert-next":
		d.severeTab, d.severeRow = (d.severeTab+1)%severeNumTabs, 0
	}
	return d
}

// --- rendering ---

// hz is the title rule's glyph per --ascii (the panel draws the frame; the
// title's inner rule is the window's own).
func hz(o render.Opts) string {
	if o.ASCII {
		return "-"
	}
	return "─"
}

// severeArrows are the chip labels per --ascii: the words, not the

// severeWindowName is the title; in the record it keeps the name and adds the
// crumb (plan §5.6 M-6): "NOTABLE EVENTS AND FORECASTS ─── Warnings · 2 / 9".
// severeWindowName covers weather, non-weather hazards (a landslide is not
// weather) and now products that are not warnings at all — a forecast or an
// outlook is notable without being bad news (MVS-D-59).
const severeWindowName = "NOTABLE EVENTS AND FORECASTS"

// severeTitleChrome is what the panel spends around a title with a
// right-aligned stamp: the corner + rule + space before, the space + rule +
// corner after, and the two spaces the join costs (the [details] convention).
const severeTitleChrome = 10

// severeTitle composes the title with the right-aligned Updated stamp (the
// [details] convention): bold white name and stamp, the rule in the tile's
// tone. Widths are display cells (round-2 CQ #17).
func (d Dashboard) severeTitle(o render.Opts, w int) string {
	title := severeWindowName
	stamp := "Awaiting first fetch"
	if !d.severe.Updated.IsZero() {
		stamp = "Updated " + o.Clock.Stamp(d.severe.Updated.Local())
	}
	if d.severeDetail {
		stamp = fmt.Sprintf("%s · %d / %d", severeTabs()[d.severeTab].TabLabel, d.severeRow+1, d.severeCount())
	}
	fill := w - severeTitleChrome - render.Width(title) - render.Width(stamp)
	if fill <= 1 {
		return render.Tint(title, render.Tok(render.ModalTitle))
	}
	return render.Tint(title, render.Tok(render.ModalTitle)) + " " + strings.Repeat(hz(o), fill) + " " + render.Tint(stamp, render.Tok(render.ModalTitle))
}

// severeModal renders the window: the browse table or the focused record,
// on the open category's tint mixed onto the modal substrate.
func (d Dashboard) severeModal(o render.Opts) string {
	fg, bg := render.CategoryTone(severeTabs()[d.severeTab].Tint, d.darkBG)
	w := min(o.Width, d.modalWidth())
	title := d.severeTitle(o, w)
	if d.severeDetail {
		return d.floatModalToned(o, d.modalWidth(), title, d.severeDetailLines(o), fg, bg)
	}
	return d.floatModalToned(o, d.modalWidth(), title, d.severeBrowseLines(o, w), fg, bg)
}

// severeTabRow is the category row in the widest form that fits the rail
// budget: "[ › Warnings ] [ Watches ]", then "[›Warnings] [Watches]", then the
// short names. The open tab is marked with the pointer and reads bright.
func (d Dashboard) severeTabRow(o render.Opts, inner int) string {
	ptr := o.Glyphs().Pointer
	tabs := severeTabs()
	forms := []func(i int) string{
		func(i int) string {
			if SevereTab(i) == d.severeTab {
				return "[ " + ptr + " " + tabs[i].TabLabel + " ]"
			}
			return "[ " + tabs[i].TabLabel + " ]"
		},
		func(i int) string {
			if SevereTab(i) == d.severeTab {
				return "[" + ptr + tabs[i].TabLabel + "]"
			}
			return "[" + tabs[i].TabLabel + "]"
		},
		func(i int) string {
			if SevereTab(i) == d.severeTab {
				return "[" + ptr + tabs[i].TabShort + "]"
			}
			return "[" + tabs[i].TabShort + "]"
		},
	}
	plain := func(f func(int) string) []string {
		out := make([]string, len(tabs))
		for i := range tabs {
			out[i] = f(i)
		}
		return out
	}
	var cells []string
	for _, f := range forms {
		cells = plain(f)
		if render.Width(strings.Join(cells, " ")) <= inner {
			break
		}
	}
	for i := range cells {
		if SevereTab(i) == d.severeTab { // the open tab wears its category's UNMIXED tint under bold white (FR-2, round-2 Y4)
			cells[i] = render.TintRaw(cells[i], "1;"+render.Tok(render.AlertModalText)+";"+render.Tok(tabs[i].Tint))
		}
	}
	return strings.Join(cells, " ")
}

// severeDownSources states every dead source on the category line ("· NHC
// unavailable"), so an empty tab never reads as "all clear" (plan §5.9).
func (d Dashboard) severeDownSources() string {
	var down []string
	for _, s := range d.severe.Sources {
		if !s.OK {
			down = append(down, render.PlainLine(s.Name)+" unavailable")
		}
	}
	if len(down) == 0 {
		return ""
	}
	return " · " + strings.Join(down, " · ")
}

// severeChips is the control row; [enter] mutes when there is nothing to
// open. The long form is the mock's; when it does not fit the content width
// (the 80-col --ascii floor, R3-B-07) the labels shorten step by step
// instead of re-flowing.
func (d Dashboard) severeChips(o render.Opts, hasRows bool, width int) string {
	ud, lr := "↑↓", "←→" // KeyCap names them in words under --ascii (R5-C-13)
	canRead := hasRows && d.cfg.NarrateEvent != nil
	// Each form is built only when the wider one did not fit (the window's
	// frame budget); the floor — the 80-col --ascii chips, round 4 B-03 —
	// leaves the arrows unlabelled.
	// A STATE-DRIVEN LABEL, the same as the radio panel's (UAT 37,
	// radio_panel.go): the chip says what the key will DO next rather than
	// naming both halves. A paused read reads "Play", because that is the press
	// that resumes it.
	//
	// It also keeps the row inside the 80-column ASCII floor, which the earlier
	// constant "Play/Pause" did not — six columns wider than the "Read" it
	// replaced was enough to push the floor over the content width.
	read := "Play"
	if d.severeReading != "" && !d.severeReadPause {
		read = "Pause"
	}
	for _, f := range [][3]string{{"Navigate", "Category", "Event Details"}, {"Navigate", "Category", "Details"}, {"Rows", "Tabs", "Details"}} {
		if row := "  " + o.KeyCap(ud) + " " + f[0] + "  " + o.KeyCap(lr) + " " + f[1] + "  " + o.KeyCapIf("enter", hasRows) + " " + f[2] + "  " + o.KeyCapIf("space", canRead) + " " + read + "  " + o.KeyCap("esc") + " Close"; render.Width(row) <= width {
			return row
		}
	}
	return "  " + o.KeyCap(ud) + " " + o.KeyCap(lr) + "  " + o.KeyCapIf("enter", hasRows) + " Open  " + o.KeyCapIf("space", canRead) + " " + read + "  " + o.KeyCap("esc") + " Close"
}

// severeBrowseLines is the browse body for a panel w wide (the mock, width-
// exact): blank · tabs · blank · category line · the table with its
// one-column rail · the total · blank · chips. The table window is whatever
// the panel budget leaves, and the chrome gives up its blank lines on a short
// terminal (R3-B-01), so the body never exceeds the budget — the panel never
// re-wraps the table.
func (d Dashboard) severeBrowseLines(o render.Opts, w int) []string {
	inner := w - 7 // the rail budget: chrome (4) + rail col + gap
	tab := severeTabs()[d.severeTab]
	n := d.severeCount()
	// ONE STATEMENT OF THE COUNT, not two. "Advisories — 14 active" sat above
	// "14 Total Category Events" and the two were the same number: the heading
	// only differed when the tab was capped, which needs more than five hundred
	// rows in one category (HUM LEAD, UAT 2026-09-01). So the total line carries
	// everything the heading did — the cap when there is one, and any dead
	// source — and the heading is gone.
	total := d.severe.Totals[d.severeTab]
	totalText := fmt.Sprintf("%d Total Category Events", total)
	if n > 0 && total > n {
		totalText = fmt.Sprintf("Showing %d of %d Total Category Events", n, total)
	}
	totalText += d.severeDownSources()
	totalLine := render.PadTo("", inner-len(totalText)) + totalText
	if n == 0 {
		return d.severeEmptyLines(o, w, inner, tab, totalLine)
	}
	head := []string{"", "  " + d.severeTabRow(o, inner), ""}
	foot := []string{totalLine, "", d.severeChips(o, true, w-4)}
	budget := d.modalMax()
	if budget < len(head)+len(foot)+2 { // a short terminal: the blanks go first, then the total line
		head, foot = head[1:2:2], foot[0:1:1]
		foot = append(foot, d.severeChips(o, true, w-4))
		if budget < len(head)+len(foot)+2 {
			foot = foot[1:]
		}
	}
	win := max(2, budget-len(head)-len(foot))   // header + at least one row
	window := win - 1                           // rows visible
	d.severeRow = max(0, min(d.severeRow, n-1)) // the renderer never trusts a focus past the tab (R5-C-07)
	lo := 0
	if d.severeRow >= window {
		lo = d.severeRow - window + 1
	}
	hi := min(n, lo+window)
	cells := make([]render.SevereCell, 0, hi-lo)
	for i := lo; i < hi; i++ {
		r := d.severeRowAt(i)
		cells = append(cells, render.SevereCell{
			Num: i + 1, Event: render.PlainLine(r.Product), Location: render.PlainLine(r.Location), Detection: render.PlainLine(r.Detection), Declared: render.PlainLine(r.Declared), Expires: render.PlainLine(r.Expires),
			Focused: i == d.severeRow, Playing: d.severePlaying(r.Key), Paused: d.severeReadPause && d.severeReading == r.Key,
		})
	}
	table := o.SevereTable(cells, inner, tab.Tint)
	if n > window { // more rows than the window: the rail
		table = severeRailed(o, table, inner, lo, n, window)
	}
	return append(append(head, table...), foot...)
}

// severeEmptyLines is the empty-state body: the category line with any dead
// source, the stamp, the watchlist hint only when there is no watchlist to
// track (FR-14), the total and the chips with [enter] muted.
func (d Dashboard) severeEmptyLines(o render.Opts, w, inner int, tab category.Spec, totalLine string) []string {
	// NO CATEGORY LINE WHEN THE TAB IS EMPTY. It read "Forecasts — no active
	// events" directly above "No active forecasts events" and "0 Total Category
	// Events" — three ways of saying nothing is here. The heading would earn its
	// place if it introduced sub-groups, and it does not (HUM LEAD, UAT
	// 2026-09-01). The dead-source note it carried moves to the stamp, which is
	// the other line about how current the tab is.
	lines := []string{"", "  " + d.severeTabRow(o, inner), ""}
	stamp := "no fetch yet"
	if !d.severe.Updated.IsZero() {
		stamp = "Updated " + o.Clock.DateTimeZone(d.severe.Updated.Local())
	}
	lines = append(lines, "  No active "+strings.ToLower(tab.TabLabel)+" events · "+stamp)
	// A WATCHLIST TAB SAYS WHY IT IS EMPTY IN BOTH CASES.
	//
	// This only spoke when the watchlist was EMPTY, which is the case where a
	// listener already knows why nothing is there. With locations set — the case
	// where the tab looks like a national view that has missed something — it
	// said nothing at all. Found at UAT 2026-09-06: a Special Weather Statement
	// for Alabama, seen in another app, absent here, and no way to tell from
	// this screen that the tab never looks past your own zones. The national
	// feed carries nine products and no statements (SAM-D-10), so Warnings is
	// nationwide and this tab is not — an asymmetry only this line can explain.
	if tab.Watchlist {
		if d.numPriority() == 0 {
			lines = append(lines, "  (tracks your watchlist — add locations with ctrl+a)")
		} else {
			lines = append(lines, "  (your watchlist locations only — these are not in the national feed)")
		}
	}
	return append(lines, "", totalLine, "", d.severeChips(o, false, w-4))
}

// severeRailed adds the one-column rail to a windowed table: ▲ on the
// header, the thumb over the rows above the last visible one, ▼ on that last
// row (the thumb never hides under it — R3-B-05).
func severeRailed(o render.Opts, table []string, inner, lo, n, window int) []string {
	g := render.RailGlyphsFor(o.ASCII)
	rows := table[1:]
	out := []string{render.PadTo(table[0], inner+1) + g.Up}
	out = append(out, render.Railify(rows[:len(rows)-1], inner+2, lo, n-1, window-1, g)...)
	return append(out, render.PadTo(rows[len(rows)-1], inner+1)+g.Down)
}

// severePlaying reports whether an event is the one being read over the
// radio right now — a [space] read, else the breaking takeover's item (the
// ▶ mark, UAT item 11).
// The tape carries the feed's raw id; the window's key is its normalised
// form (the OID, or the bare id), so a suffix match covers both.
func (d Dashboard) severePlaying(key string) bool {
	if key == "" {
		return false
	}
	if d.severeReading == key {
		return true
	}
	return d.breaking != nil && (d.breaking.ID == key || strings.HasSuffix(d.breaking.ID, key))
}

// readFocusedEvent is [space] inside the window: the focused event is read
// over the radio through the app's director (UAT option B) — never the
// dashboard's location underneath. Inert without the hook or a row.
func (d Dashboard) readFocusedEvent() Dashboard {
	r := d.severeRowAt(d.severeRow)
	if r == nil || d.cfg.NarrateEvent == nil {
		return d
	}
	key := r.Key
	return d.withCmd(func() tea.Msg { d.cfg.NarrateEvent(key); return nil })
}

// severeDetailLines is the focused event's record in the [A] shape (FR-5, FR-7):
// bold title + meta, timing, area, the paragraphs, then the chips.
func (d Dashboard) severeDetailLines(o render.Opts) []string {
	r := d.severeRowAt(d.severeRow)
	if r == nil {
		return []string{"", "  No event selected.", "", "  " + o.KeyCap("esc") + " Back"}
	}
	rec := r.Record
	wrapW := min(o.Width, d.modalWidth()) - 9 // breathing room beside the scroll rail (UAT 23.3)
	out := []string{"", "  " + render.TintRaw(render.PlainLine(rec.Title), "1;"+render.Tok(render.ModalTitle)) + "  " + render.PlainLine(rec.Meta)}
	if rec.Timing != "" {
		out = append(out, "  "+render.PlainLine(rec.Timing))
	}
	if rec.Area != "" {
		out = append(out, wrapPrefixed(o, render.Plain(rec.Area), wrapW)...)
	}
	for _, p := range rec.Paras {
		if p == "" {
			continue
		}
		out = append(out, "")
		out = append(out, wrapPrefixed(o, render.Plain(p), wrapW)...)
	}
	for i := 2; i < len(out); i++ { // body text white for contrast, as [A] reads (UAT 55)
		if out[i] != "" {
			out[i] = render.Tint(out[i], render.Tok(render.AlertModalText))
		}
	}
	ud, _ := "↑↓", "←→" // KeyCap names them in words under --ascii
	return append(out, "", "  "+o.KeyCap("esc")+" Back   "+o.KeyCap("esc esc")+" Close   "+o.KeyCap(ud)+" Scroll")
}
