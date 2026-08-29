package tty

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

func TestSevereTabsAreSixInImportanceOrder(t *testing.T) {
	tabs := severeTabs()
	want := []string{"Warnings", "Watches", "Advisories", "Spec. Statements", "Sig. Quakes", "Tropical"}
	if len(tabs) != len(want) || len(tabs) != int(severeNumTabs) {
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

// severeFixture is the window's test model: a 9-row Warnings index (the
// plan §5 mock's rows) at a terminal size, opened with w.
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
	detections := []string{"Observed", "Radar Indicated", "Observed", "Spotter Reported", "Likely", "", "Observed", "Radar Indicated", "Possible"}
	for i := range products {
		sev := TickerOrange
		if i == 1 {
			sev = TickerRed
		}
		rows = append(rows, SevereRow{Key: fmt.Sprint(i), Tab: SevereWarnings, Product: products[i], Location: places[i], Detection: detections[i], Declared: "08/28 11:20 EDT", Expires: "08/28 20:00 EDT", Severity: sev,
			Record: SevereRecord{Title: strings.ToUpper(products[i]), Meta: "[Extreme · Immediate · Observed]", Timing: "Declared 08/28 08:45 CDT   Expires 08/28 09:00 CDT   (~15m)", Area: "Area: Johnson County, KS · NWS Kansas City", Paras: []string{"At 845 AM CDT, a severe thunderstorm capable of producing a tornado was located near Olathe, moving northeast at 30 mph.", "Instructions: TAKE COVER NOW!"}}})
	}
	model, _ = model.Update(SevereMsg{Gen: 1, Rows: rows, Totals: [severeNumTabs]int{9}, Updated: time.Date(2026, 8, 28, 15, 38, 5, 0, time.UTC)})
	model, _ = model.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	return model
}

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

func TestSevereBrowseFramesMatchTheMocksAtEveryWidth(t *testing.T) {
	for _, w := range []int{80, 100, 120, 133} {
		frame := stripANSITest(severeFixture(t, w, 44, false).View().Content)
		if w == 133 && !strings.Contains(frame, "EXPIRES") {
			t.Fatal("133: every column, EXPIRES included")
		}
		if !strings.Contains(frame, "SEVERE WEATHER / DISASTER EVENTS") || !strings.Contains(frame, "Warnings — 9 active") {
			t.Fatalf("%d cols: header/category line missing:\n%s", w, frame)
		}
		if !strings.Contains(frame, "› Warnings") && !strings.Contains(frame, "›Warnings") && !strings.Contains(frame, "›Warn") {
			t.Fatalf("%d cols: the open tab is not marked", w)
		}
		for _, line := range strings.Split(frame, "\n") {
			if render.Width(strings.TrimRight(line, " ")) > w {
				t.Fatalf("%d cols: a line overflows the terminal: %q", w, line)
			}
		}
		if w == 120 && (strings.Contains(frame, "EXPIRES") || !strings.Contains(frame, "DETECTION") || !strings.Contains(frame, "Radar Indicated")) {
			t.Fatal("120: EXPIRES drops first; DETECTION and DECLARED stay")
		}
		if w == 100 && (strings.Contains(frame, "DETECTION") || !strings.Contains(frame, "DECLARED")) {
			t.Fatal("100: DETECTION drops next and DECLARED stays")
		}
		if w == 80 && strings.Contains(frame, "DECLARED") {
			t.Fatal("80: DECLARED must drop")
		}
		if !strings.Contains(frame, "9 Total Category Events") || !strings.Contains(frame, "[enter]") || !strings.Contains(frame, "[space]") {
			t.Fatalf("%d cols: footer missing", w)
		}
	}
}

func TestSevereRailIsOneColumn(t *testing.T) {
	frame := stripANSITest(severeFixture(t, 120, 24, false).View().Content) // 24 rows: the table windows and the rail shows
	col := -1
	for _, line := range strings.Split(frame, "\n") {
		if !strings.Contains(line, "SEVERE") && !strings.Contains(line, "│ │") && !strings.Contains(line, "▲ │") && !strings.Contains(line, "█ │") && !strings.Contains(line, "▼ │") {
			continue
		}
		for _, g := range []string{"▲ │", "█ │", "▼ │", "│ │"} {
			if i := strings.Index(line, g); i >= 0 {
				c := render.Width(line[:i])
				if col < 0 {
					col = c
				} else if c != col {
					t.Fatalf("rail glyph %q at column %d, expected %d: %q", g[:len("▲")], c, col, line)
				}
			}
		}
	}
	if col < 0 {
		t.Fatalf("no rail found:\n%s", frame)
	}
}

func TestSevereBrowseASCIIHasNoGlyphs(t *testing.T) {
	d := severeFixture(t, 120, 24, true).(Dashboard) // 24 rows: the table windows, so the rail's ASCII forms render too
	frame := stripANSITest(d.renderModal(d.opts()))  // the window alone — the dashboard behind it keeps its own glyph policy
	for _, r := range frame {
		if r > 127 && r != '·' && r != '—' { // typography the mock keeps under --ascii (mock.py); the marks and rail do not
			t.Fatalf("non-ASCII glyph %q in --ascii frame:\n%s", r, frame)
		}
	}
	for _, want := range []string{"> Warnings", "[up/down] Navigate", " ^", " v", " #"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("ASCII form %q missing:\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "!") { // no severity glyph in the table (HUM LEAD UAT 2026-08-28)
		t.Fatalf("the table carries no severity glyph:\n%s", frame)
	}
}

func TestSevereEmptyState(t *testing.T) {
	bare, err := NewDashboard(Config{Version: "t"}) // no watchlist yet
	if err != nil {
		t.Fatal(err)
	}
	var m tea.Model = bare
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 44})
	m, _ = m.Update(SevereMsg{Gen: 1})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight}) // Advisories
	frame := stripANSITest(m.View().Content)
	if !strings.Contains(frame, "Advisories — no active events") || !strings.Contains(frame, "tracks your watchlist") {
		t.Fatalf("empty state without a watchlist: %s", frame)
	}
	full := dash(t) // a populated watchlist: the same empty tab, no hint (FR-14)
	full, _ = full.Update(SevereMsg{Gen: 1})
	full, _ = full.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	full, _ = full.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	full, _ = full.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if f := stripANSITest(full.View().Content); !strings.Contains(f, "Advisories — no active events") || strings.Contains(f, "tracks your watchlist") {
		t.Fatalf("a populated watchlist must not be told to add locations:\n%s", f)
	}
}

func TestSevereDownSourceIsStated(t *testing.T) {
	m := severeFixture(t, 120, 44, false)
	_ = m.View().Content // prime the memo: the next publish must invalidate it
	m, _ = m.Update(SevereMsg{Gen: 2, Sources: []SevereSource{{Name: "USGS", OK: true}, {Name: "NHC", OK: false}}})
	if frame := stripANSITest(m.View().Content); !strings.Contains(frame, "NHC unavailable") {
		t.Fatalf("a dead source must be stated on the category line:\n%s", frame)
	}
}

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

func TestSevereWindowNeverForwardsProviderEscapes(t *testing.T) {
	// An OSC clipboard write, a CSI clear, a raw C1 CSI and DCS, and a newline
	// — in every field (NFR-5; R3-B-04: a newline split a row into three).
	evil := "x\x1b]52;c;aGVsbG8=\x07y\x1b[2Jz\u009bq\u0090r\nw"
	row := func(s string) SevereRow {
		return SevereRow{Key: "k", Tab: SevereWarnings, Product: s, Location: s, Declared: s, Expires: s, Record: SevereRecord{Title: s, Meta: s, Timing: s, Area: s, Paras: []string{s}}}
	}
	frames := func(r SevereRow) (browse, detail string, lines int) {
		m := dash(t)
		m, _ = m.Update(SevereMsg{Gen: 1, Rows: []SevereRow{r}, Totals: [severeNumTabs]int{1}, Sources: []SevereSource{{Name: r.Product}}})
		m, _ = m.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
		d := m.(Dashboard)
		browse = d.View().Content
		lines = strings.Count(d.renderModal(d.opts()), "\n")
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		detail = m.View().Content
		return
	}
	browse, detail, evilLines := frames(row(evil))
	_, _, cleanLines := frames(row("plain"))
	for _, view := range []string{browse, detail} {
		for _, bad := range []string{"\x1b]", "\x07", "\x1b[2J", "\u009b", "\u0090"} {
			if strings.Contains(view, bad) {
				t.Fatalf("a provider escape %q reached the frame", bad) // the OSC's text may survive as plain characters; its bytes never do
			}
		}
	}
	if evilLines != cleanLines {
		t.Fatalf("a newline in a field changed the window's line count: %d vs %d", evilLines, cleanLines)
	}
}

func TestStatusModalGaugesTheSevereIndex(t *testing.T) {
	m := severeFixture(t, 120, 44, false)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	if frame := stripANSITest(m.View().Content); !strings.Contains(frame, "severe index   9 rows / 500") {
		t.Fatalf("[S] lacks the severe gauge:\n%s", frame)
	}
}

// TestSevereGoldens pins the browse frames at three widths and under --ascii,
// each with the width invariant beside the byte pin (a pin alone can freeze
// a defect — calibration "Byte Pins Ride With Invariant Assertions").
func TestSevereGoldens(t *testing.T) {
	local := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = local })
	cases := []struct {
		name  string
		w     int
		ascii bool
	}{{"severe-80x44", 80, false}, {"severe-100x44", 100, false}, {"severe-120x44", 120, false}, {"severe-133x44", 133, false}, {"severe-120x44-ascii", 120, true}, {"severe-80x44-ascii", 80, true}, {"severe-120x20", 120, false}}
	for _, c := range cases {
		d := severeFixture(t, c.w, 44, c.ascii).(Dashboard)
		d.now = func() time.Time { return time.Date(2026, 8, 28, 15, 40, 0, 0, time.UTC) }
		frame := d.View().Content
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

// --- the modal memo (FR-10) ---

func TestModalMemoHitsAcrossTicksWithALoadingRow(t *testing.T) {
	m := severeFixture(t, 120, 44, false).(Dashboard)
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return now }
	m.recent = &snapshot.Snapshot{Locations: []snapshot.Location{{Label: "Loading City"}}} // rowLoading == true
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
	now = now.Add(time.Minute) // the minute is Details' input, not the window's (R3-B-09)
	_ = m.View().Content
	if _, m2 := m.modalMemoCounts(); m2-m1 != 0 {
		t.Fatal("a new minute must not miss on the severe window")
	}
	m.modal, m.selected = modalDetails, 0
	_ = m.View().Content
	_, m3 := m.modalMemoCounts()
	now = now.Add(time.Minute)
	_ = m.View().Content
	if _, m4 := m.modalMemoCounts(); m4-m3 != 1 {
		t.Fatal("a new minute must miss once on Details (its ages)")
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

func TestModalMemoInvalidatesOnEveryInput(t *testing.T) {
	base := func() Dashboard {
		d := severeFixture(t, 120, 44, false).(Dashboard)
		fixed := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
		d.now = func() time.Time { return fixed }
		d.recent = &snapshot.Snapshot{Locations: []snapshot.Location{{Label: "Loading A"}, {Label: "Loading B"}}} // two loading rows: Details keys on the shimmer
		d.cfg.Stats = func() Stats { return Stats{} }
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
		{"breaking (the ▶ mark)", func(d *Dashboard) { d.breaking = &TickerItem{ID: "0", Category: CatWarning} }},
		{"reading (the ▶ mark)", func(d *Dashboard) { d.severeReading = "0" }},
		{"addQuery", func(d *Dashboard) { d.modal = modalAdd; d.addQuery = "x" }},
		{"setup", func(d *Dashboard) { d.modal = modalSetup; d.setup.query = "x" }},
		{"themeIdx", func(d *Dashboard) { d.modal = modalTheme; d.themeIdx = 1 }},
		{"voiceIdx", func(d *Dashboard) { d.modal = modalVoice; d.voiceIdx = 1 }},
		{"nvoices", func(d *Dashboard) { d.modal = modalVoice; d.voiceList = append(d.voiceList, "Z") }},
		{"units", func(d *Dashboard) { d.units = render.UnitC }},
		{"ascii", func(d *Dashboard) { d.cfg.ASCII = true }},
		{"frame (a loading row)", func(d *Dashboard) { d.modal = modalDetails; d.frame++ }},
		{"darkBG", func(d *Dashboard) { d.darkBG = false }},
		{"theme", func(d *Dashboard) { render.SetTheme("Monochrome") }},
		{"minute (Details' ages)", func(d *Dashboard) {
			d.modal = modalDetails
			d.now = func() time.Time { return time.Date(2026, 8, 28, 12, 1, 0, 0, time.UTC) }
		}},
		{"addMode", func(d *Dashboard) { d.modal = modalAdd; d.addMode = "lookup" }},
		{"addErr", func(d *Dashboard) { d.modal = modalAdd; d.addErr = "no such place" }},
		{"voiceNote", func(d *Dashboard) { d.modal = modalVoice; d.voiceNote = "downloading" }},
		{"voiceErr", func(d *Dashboard) { d.modal = modalVoice; d.voiceErr = "failed" }},
		{"radioVoice", func(d *Dashboard) { d.modal = modalVoice; d.radioVoice = "Z" }},
		{"themeErr", func(d *Dashboard) { d.modal = modalTheme; d.themeErr = "bad json" }},
		{"second ([S] ages)", func(d *Dashboard) {
			d.modal = modalStatus
			d.now = func() time.Time { return time.Date(2026, 8, 28, 12, 0, 1, 0, time.UTC) }
		}},
		{"stats ([S])", func(d *Dashboard) { d.modal = modalStatus; d.cfg.Stats = func() Stats { return Stats{LastDump: "x"} } }},
	}
	for _, c := range cases {
		d := base()
		if pre := d; c.name != "modal" {
			c.mut(&pre)
			if pre.modal != d.modal || strings.HasPrefix(c.name, "modalScroll") {
				// inputs that key only while their window is open: open it first, prime, then mutate the one input
				d.modal, d.severeDetail = pre.modal, pre.severeDetail
				_ = d.View().Content
			}
		}
		_, m0 := d.modalMemoCounts()
		c.mut(&d)
		_ = d.View().Content
		render.SetTheme(render.DefaultThemeName) // the theme case must not leak into the next
		if _, m1 := d.modalMemoCounts(); m1-m0 != 1 {
			t.Errorf("%s: %d misses after the change, want 1", c.name, m1-m0)
		}
	}
}

// A short terminal (R3-B-01): the chrome gives up its blank lines, then the
// total line, so the body never exceeds the panel budget and the panel never
// re-wraps the table — one rail, columns intact, at 20, 18 and 17 rows.
func TestSevereShortTerminalKeepsTheTableIntact(t *testing.T) {
	for _, h := range []int{24, 20, 18, 17} {
		d := severeFixture(t, 120, h, false).(Dashboard)
		body := d.severeBrowseLines(d.opts(), min(d.opts().Width, d.modalWidth()))
		if len(body) > d.modalMax() {
			t.Fatalf("%d rows: %d body lines for a budget of %d", h, len(body), d.modalMax())
		}
		frame := stripANSITest(d.renderModal(d.opts()))
		if strings.Count(frame, "▲") != 1 || !strings.Contains(frame, "[esc] Close") {
			t.Fatalf("%d rows: the window must keep one rail and its chips:\n%s", h, frame)
		}
		for _, line := range strings.Split(frame, "\n") {
			if strings.Contains(line, "001.") && !strings.Contains(line, "08/28 11:20 EDT") {
				t.Fatalf("%d rows: a row lost its columns: %q", h, line)
			}
		}
	}
}

// The thumb stays visible at the bottom of the list (R3-B-05).
func TestSevereRailThumbShowsAtTheBottom(t *testing.T) {
	m := severeFixture(t, 120, 24, false)
	for range 8 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	d := m.(Dashboard)
	if frame := stripANSITest(d.renderModal(d.opts())); !strings.Contains(frame, "█") || !strings.Contains(frame, "009.") {
		t.Fatalf("at the bottom the thumb must show beside the last rows:\n%s", frame)
	}
}

// A breaking event records its category for the opening rule end to end
// (R3-B-10: severeTabOf was only reachable by hand).
func TestBreakingEventOpensTheWindowOnItsCategory(t *testing.T) {
	m := dash(t).(Dashboard)
	m.now = func() time.Time { return time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC) }
	var mm tea.Model = m
	mm, _ = mm.Update(TickerBreakingMsg{Item: TickerItem{ID: "q", Category: CatQuake, Text: "M 6.1"}})
	mm, _ = mm.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	if d := mm.(Dashboard); d.severeTab != SevereQuakes {
		t.Fatalf("a breaking quake opens Sig. Quakes, got %v", d.severeTab)
	}
	for _, c := range []struct {
		cat  TickerCategory
		want SevereTab
	}{{CatTropical, SevereTropical}, {CatWarning, SevereWarnings}, {CatWatch, SevereWatches}} {
		if got := severeTabOf(TickerItem{Category: c.cat}); got != c.want {
			t.Errorf("%v → %v, want %v", c.cat, got, c.want)
		}
	}
}

// ↑/↓ scroll the record and a second enter keeps the scroll (R3-B-06).
func TestSevereRecordScrollsAndEnterIsInertInside(t *testing.T) {
	m := severeFixture(t, 120, 17, false) // a short budget so the record scrolls
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if d := m.(Dashboard); !d.severeDetail || d.modalScroll != 2 {
		t.Fatalf("down scrolls the record: detail %v scroll %d", d.severeDetail, d.modalScroll)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if d := m.(Dashboard); d.modalScroll != 2 {
		t.Fatalf("enter inside the record must not reset its scroll: %d", d.modalScroll)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if d := m.(Dashboard); d.modalScroll != 1 {
		t.Fatalf("up scrolls back: %d", d.modalScroll)
	}
}

// The open tab wears its category's tint (FR-2, round-2 Y4) — with colour
// on, the chip carries the unmixed hue token.
func TestSevereOpenTabWearsItsCategoryTint(t *testing.T) {
	d := benchDash(t, 133, 44).(Dashboard)
	var m tea.Model = d
	m, _ = m.Update(SevereMsg{Gen: 1})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	d = m.(Dashboard)
	if frame := d.renderModal(d.opts()); !strings.Contains(frame, render.Tok(render.EventCatOrangeBG)+"m[ › Warnings ]") {
		t.Fatalf("the Warnings chip must carry the orange tint:\n%s", frame)
	}
}

// The [A] alert module and its compact line strip escapes from every field
// (NFR-6, R3-C-01).
func TestAlertModuleAndCompactLineStripEscapes(t *testing.T) {
	evil := "x\x1b]52;c;aGVsbG8=\x07y\x1b[31mz\nq"
	for _, h := range []int{44, 24} {
		m := dash(t).(Dashboard)
		m.height = h
		s := *m.snap
		s.Locations = append([]snapshot.Location(nil), s.Locations...)
		s.Locations[0].Alerts = []snapshot.Alert{{Event: evil, Severity: evil, Headline: evil, Description: evil, Instruction: evil, AreaDesc: evil, Urgency: evil, Certainty: evil}}
		m.snap = &s
		for _, view := range []string{m.View().Content, func() string { m.modal = modalAlerts; return m.View().Content }()} {
			if strings.Contains(view, "\x1b]") || strings.Contains(view, "\x07") {
				t.Fatalf("%d rows: an alert field's escape reached the frame", h)
			}
		}
	}
}

// A lone esc then a letter arrives fused as alt+letter (the terminal's
// ambiguity; the input layer has no ESC timeout — probed on a pty, 0.12.0
// behaves the same): the model reads it as the user meant it, esc then the
// key. No binding uses alt, so nothing else can claim the press.
func TestLoneEscThenKeyIsNotLost(t *testing.T) {
	for _, b := range defaultKeyMap() {
		for _, k := range b.Keys {
			if strings.Contains(k, "alt+") {
				t.Fatalf("a default binding uses alt (%q): the esc-fusion rule would shadow it", k)
			}
		}
	}
	m := dash(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'w', Mod: tea.ModAlt}) // esc on the dashboard, then w
	if m.(Dashboard).modal != modalSevere {
		t.Fatal("esc+w fused must open the window")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'w', Mod: tea.ModAlt}) // esc closes, w re-opens
	if m.(Dashboard).modal != modalSevere {
		t.Fatal("esc+w inside the window closes and re-opens it")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModAlt | tea.ModCtrl}) // esc then ctrl+s
	if m.(Dashboard).modal != modalSevere {
		t.Fatal("esc+ctrl+s fused must read as esc then ctrl+s")
	}
	// An uppercase key after the esc arrives as alt+shift+letter with no text:
	// it is the letter (VALIDATE 2026-08-29 — esc then S was lost on a pty).
	if _, second, ok := splitEscFusion(tea.KeyPressMsg{Code: 's', Mod: tea.ModAlt | tea.ModShift}); !ok || second.String() != "S" {
		t.Fatalf("esc+shift+s reads as esc then S: %v %q", ok, second.String())
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModAlt | tea.ModShift}) // esc closes the window, S opens Status
	if m.(Dashboard).modal != modalStatus {
		t.Fatalf("esc then S must open Status, got %v", m.(Dashboard).modal)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'w', Text: "w"})
	// ESC CR / ESC HT / ESC ESC read as esc then the key itself — never ctrl+m /
	// ctrl+i (round 4, B-08: the ctrl-byte rule swallowed Enter).
	for _, code := range []rune{tea.KeyEnter, tea.KeyTab, tea.KeyEscape, tea.KeyUp, tea.KeyDown, tea.KeyBackspace, tea.KeyDelete, 'é'} { // R5-C-08: arrows and the rest un-fuse too
		first, second, ok := splitEscFusion(tea.KeyPressMsg{Code: code, Mod: tea.ModAlt})
		if !ok || first.Code != tea.KeyEscape || second.Code != code || second.Mod != 0 {
			t.Fatalf("esc+%q: %v %+v %+v", code, ok, first, second)
		}
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 0x13, Mod: tea.ModAlt}) // the same press as a raw control byte after the esc
	if m.(Dashboard).modal != modalSevere {
		t.Fatal("esc+^S fused must read as esc then ctrl+s")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModAlt}) // esc closes the window, a opens About
	if m.(Dashboard).modal != modalAbout {
		t.Fatalf("esc+a fused must open About, got %v", m.(Dashboard).modal)
	}
}

// The ▶ mark rides only the event being read over the radio (UAT item 11):
// the focused row wears the pointer alone; the breaking item's row wears ▶.
func TestPlayMarkFollowsTheBreakingEvent(t *testing.T) {
	m := severeFixture(t, 120, 44, false)
	d := m.(Dashboard)
	if frame := stripANSITest(d.renderModal(d.opts())); strings.Contains(frame, "▶") {
		t.Fatalf("no radio narration: no ▶:\n%s", frame)
	}
	m, _ = m.Update(TickerBreakingMsg{Item: TickerItem{ID: "https://api.weather.gov/alerts/1", Category: CatWarning}}) // the feed's raw id; the window's key is "1"
	d = m.(Dashboard)
	lines := strings.Split(stripANSITest(d.renderModal(d.opts())), "\n")
	hits := 0
	for _, l := range lines {
		if strings.Contains(l, "▶") {
			hits++
			if !strings.Contains(l, "002.") {
				t.Fatalf("▶ must ride the breaking event's row (002): %q", l)
			}
		}
	}
	if hits != 1 {
		t.Fatalf("exactly one ▶, got %d", hits)
	}
}

// [space] inside the window reads the focused event through the app's
// narrator (UAT option B) — the dashboard's radio is not toggled; the ▶
// follows the SevereReadingMsg; without the hook the chip mutes and [space]
// is inert.
func TestSpaceInTheWindowReadsTheFocusedEvent(t *testing.T) {
	m := severeFixture(t, 120, 44, false).(Dashboard)
	var read []string
	m.cfg.NarrateEvent = func(key string) { read = append(read, key) }
	var mm tea.Model = m
	mm, _ = mm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	mm, cmd := mm.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if cmd == nil {
		t.Fatal("space must queue the read command")
	}
	cmd()
	d := mm.(Dashboard)
	if len(read) != 1 || read[0] != "1" || d.radioPlaying {
		t.Fatalf("read %v, radioPlaying %v — the location underneath must not start", read, d.radioPlaying)
	}
	if frame := stripANSITest(d.renderModal(d.opts())); !strings.Contains(frame, "[space] Read") {
		t.Fatalf("the Read chip:\n%s", frame)
	}
	mm, _ = mm.Update(SevereReadingMsg{Key: "1"})
	d = mm.(Dashboard)
	frame := stripANSITest(d.renderModal(d.opts()))
	for _, l := range strings.Split(frame, "\n") {
		if strings.Contains(l, "▶") && !strings.Contains(l, "002.") {
			t.Fatalf("▶ must ride the event being read: %q", l)
		}
	}
	if !strings.Contains(frame, "▶") {
		t.Fatal("▶ must show on the event being read")
	}
	mm, _ = mm.Update(SevereReadingMsg{})
	if d := mm.(Dashboard); strings.Contains(stripANSITest(d.renderModal(d.opts())), "▶") {
		t.Fatal("▶ clears when the read ends")
	}
	bare := severeFixture(t, 120, 44, false) // no hook: inert, chip muted
	bare, cmd = bare.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if cmd != nil || bare.(Dashboard).radioPlaying {
		t.Fatal("without the hook, space in the window is inert")
	}
}

// The chip row holds ONE line at the 80-col --ascii floor (round 4, B-03:
// the [space] chip re-flowed the row and the golden was recorded wrapped).
func TestSevereChipsHoldOneLineAtTheASCIIFloor(t *testing.T) {
	d := dash(t).(Dashboard)
	d.cfg.NarrateEvent = func(string) {}
	for _, ascii := range []bool{false, true} {
		o := d.opts()
		o.Width, o.ASCII = 80, ascii
		width := 69 // the content width at 80 cols (severeBrowseLines)
		row := d.severeChips(o, true, width)
		if render.Width(row) > width || strings.Contains(row, "\n") {
			t.Fatalf("ascii=%v: the chips exceed the content width (%d): %q", ascii, render.Width(row), stripANSITest(row))
		}
	}
}

// The renderer never trusts a focus past the tab (REVIEW R5-C-07); the
// category line says "showing N of M" when the cap hid rows (R5-A-05).
func TestSevereRendererClampsTheFocusAndSaysShowing(t *testing.T) {
	d := severeBench(t)
	d.severeRow = 999
	d.severe.Totals[d.severeTab] = 999 // more than the rows: the cap hid some
	d.mmemo.ok = false
	if v := stripANSITest(d.View().Content); !strings.Contains(v, "showing") {
		t.Fatalf("Totals above the rows read as showing N of M:\n%s", v)
	}
	short, _ := d.Update(tea.WindowSizeMsg{Width: 60, Height: 21})
	_ = short.View().Content // no panic at the floor with a wild focus
}
