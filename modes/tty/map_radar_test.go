package tty

// map_radar_test.go — 0.18.0 W8 at the window: the radar is its own command,
// shown whole once its loop is in, one request at a time (W8.12, D-85); the
// loop's timeline (D-86); the source's chip in the
// upper right, MRMS on green and IEM on orange (D-83); the layer switched off
// takes it away; the playback keys drive the library's loop (D-61, W8.9a);
// the lower 48's source is a Setting (D-83).

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// radarFeed answers with a loop of n frames over southern California from
// the source named; asked records each ask's newestOnly and the IEM choice.
func radarFeed(t *testing.T, source string, asked *[]string) func(context.Context, MapAsk) MapRadar {
	t.Helper()
	png, err := os.ReadFile("testdata/radar-frame.png")
	if err != nil {
		t.Fatal(err)
	}
	return func(_ context.Context, ask MapAsk) MapRadar {
		word := "loop"
		if ask.RadarIEM {
			word += "+iem"
		}
		*asked = append(*asked, word)
		n := 12
		newest := time.Date(2026, 8, 24, 0, 55, 0, 0, time.UTC)
		var frames []tuimaps.LoopFrame
		for i := n - 1; i >= 0; i-- {
			frames = append(frames, tuimaps.LoopFrame{Valid: newest.Add(-time.Duration(i) * 5 * time.Minute), PNG: png})
		}
		o := tuimaps.RadarImage(RadarLayer+"/us-a", tuimaps.Image{Frames: frames, Provider: tuimaps.ProviderIEM,
			West: -126, South: 23, East: -65, North: 51, Projection: tuimaps.PlateCarree}, newest)
		return MapRadar{Overlays: []tuimaps.Overlay{o}, Source: source}
	}
}

// settleRadar runs the radar's asks and the work they start until nothing
// more is asked.
func settleRadar(t *testing.T, d Dashboard, cmd tea.Cmd) Dashboard {
	t.Helper()
	for range 6 {
		var next []tea.Cmd
		for _, msg := range msgsOf(cmd) {
			switch msg.(type) {
			case mapRadarMsg, mapTempMsg, mapWorkedMsg: // every answer goes back through Update, the work's too
				m, c := d.Update(msg)
				d, next = m.(Dashboard), append(next, c)
			}
		}
		if len(next) == 0 {
			break
		}
		cmd = tea.Batch(next...)
	}
	return settleMap(t, d)
}

func openRadarMap(t *testing.T, source string, asked *[]string) Dashboard {
	t.Helper()
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: radarFeed(t, source, asked), MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: true}}})
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC) }
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	return settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
}

func TestTheRadarIsShownWhole(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked)
	if strings.Join(asked, ",") != "loop" {
		t.Fatalf("the radar was asked %v; want the whole loop, once (D-85)", asked)
	}
	if o, ok := d.mapPane.radarGiven[RadarLayer+"/us-a"]; !ok || len(o.Image.Frames) != 12 {
		t.Fatalf("the loop was not handed in: %v", d.mapPane.radarGiven)
	}
	if st := d.mapPane.m.Loop(); st.Count != 12 || st.Playing {
		t.Errorf("the map opens on %+v; want the loop, stopped on the newest", st)
	}
	if !strings.Contains(stripANSITest(d.mapStatusLine()), "Radar  MRMS ") {
		t.Errorf("the status line does not say the loop: %q", stripANSITest(d.mapStatusLine()))
	}
}

func TestTheSourcesChipIsInTheUpperRight(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(rendering.ResetColorEnabledForTest)
	var asked []string
	for source, ground := range map[string]render.Token{"MRMS": render.MapRadarMRMSBG, "IEM": render.MapRadarIEMBG} {
		d := openRadarMap(t, source, &asked)
		row := strings.Split(d.mapWindow(d.opts()), "\n")[1] // the tab's row (D-120)
		plain := stripANSITest(row)
		if !strings.Contains(plain, "[  "+source+"  ]  ") || !strings.HasSuffix(strings.TrimRight(plain, " "), "│") {
			t.Errorf("%s: the tab's row is %q, no chip at its right", source, plain)
		}
		if !strings.Contains(row, render.Tok(ground)) {
			t.Errorf("%s: the chip is not on its own ground", source)
		}
	}
}

func TestSwitchingRadarOffTakesItAway(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked)
	m, cmd, _ := d.handleMapKey(tea.KeyPressMsg{Code: 'R', Text: "R"}) // D-94: R is the mode; the Overlays menu no longer lists radar
	d = settleRadar(t, m.(Dashboard), cmd)
	if len(d.mapPane.radarGiven) != 0 || d.radarChipText() != "" {
		t.Errorf("radar off left %v and the chip %q", d.mapPane.radarGiven, d.radarChipText())
	}
}

func TestThePlaybackKeysDriveTheLoop(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "IEM", &asked)
	d = shiftKey(d, tea.KeyLeft)
	if st := d.mapPane.m.Loop(); st.Index != 10 {
		t.Errorf("shift+← from the newest went to frame %d, want 10", st.Index)
	}
	d = shiftKey(d, tea.KeyRight)
	if d.mapPane.m.Loop().Index != 11 {
		t.Error("shift+→ did not step on")
	}
	d = shiftKey(d, tea.KeyLeft)
	d = pressCode(d, 'n', "n")
	if d.mapPane.m.Loop().Index != 11 {
		t.Error("'n' did not return to the newest")
	}
	m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !m.(Dashboard).mapPane.m.Loop().Playing {
		t.Error("space did not play")
	}
	m, _ = m.(Dashboard).Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if m.(Dashboard).mapPane.m.Loop().Playing {
		t.Error("space again did not stop")
	}
	var help []string
	for _, r := range mapHelpRows(defaultMapKeyMap(), false) {
		help = append(help, r.keys+" "+r.help)
	}
	if !strings.Contains(strings.Join(help, "\n"), "space, ⇧←, ⇧→, n Play") {
		t.Errorf("Help does not list the playback keys:\n%s", strings.Join(help, "\n"))
	}
}

func TestTheLower48sRadarSourceIsASetting(t *testing.T) {
	d, got := uiDash(t, rowMapRadarSource)
	body, _, _ := d.focusBody(d.opts())
	if text := stripANSITest(strings.Join(body, "\n")); !strings.Contains(text, "Radar -") || !strings.Contains(text, "MRMS") {
		t.Fatalf("the Maps tab has no radar row at MRMS:\n%s", text)
	}
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	if !d.mapRadarIEM || d.radarSourceLabel() != "IEM (lower 48)" || !d.mapAsk().RadarIEM {
		t.Errorf("→ gave %q; want IEM for the lower 48, in the ask", d.radarSourceLabel())
	}
	if back, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyLeft}); back.(Dashboard).mapRadarIEM {
		t.Error("← did not go back to MRMS")
	}
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if got.MapRadarSource != "iem" {
		t.Errorf("esc wrote %q, want iem", got.MapRadarSource)
	}
	if mapDash(t, Config{MapRadarSource: "iem"}).mapRadarIEM != true || mapDash(t, Config{}).mapRadarIEM {
		t.Error("the file's word does not open the window as chosen")
	}
	_ = geo.RegionContiguous
}

// TestTheNewestFramesAgeIsAlwaysSaid is W8.8 (FR-5.4, M3): the loop's line
// says how old the newest frame is, and marks it stale past ten minutes -
// never hidden; with MRMS the note under the map says its colours are
// approximate (W8.15a).
func TestTheNewestFramesAgeIsAlwaysSaid(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked)
	d.mapPane.radarNote = "MRMS radar's colours are approximate: its scale is read from its legend."
	d = d.renderMap()
	if line := stripANSITest(d.mapStatusLine()); !strings.Contains(line, "newest 5 min ago") || strings.Contains(line, "stale") {
		t.Errorf("five minutes on, the line is %q", line)
	}
	if !strings.Contains(bodyText(d), "approximate") {
		t.Error("MRMS's note is not under the map")
	}
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 30, 0, 0, time.UTC) }
	d = d.renderMap()
	if line := stripANSITest(d.mapStatusLine()); !strings.Contains(line, "35 min ago, stale") {
		t.Errorf("thirty-five minutes on, the line is %q", line)
	}
}

// TestEveryPlaybackEventDrawsTheFrame is W8.10, guard 2's playback rows: a
// playback key and the radar landing each draw, and the stored frame is the
// library's (W2.5's freshness).
func TestEveryPlaybackEventDrawsTheFrame(t *testing.T) {
	var asked []string
	for name, event := range map[string]func(Dashboard) Dashboard{
		"step back": func(d Dashboard) Dashboard { return shiftKey(d, tea.KeyLeft) },
		"play": func(d Dashboard) Dashboard {
			m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
			return m.(Dashboard)
		},
		"the radar lands": func(d Dashboard) Dashboard {
			r := radarFeed(t, "IEM", &asked)(context.Background(), d.mapAsk())
			r.Overlays[0].Image.Frames = r.Overlays[0].Image.Frames[1:] // a refresh: the loop changed
			m, cmd := d.applyMapRadar(mapRadarMsg{radar: r})
			return settleRadar(t, m.(Dashboard), cmd)
		},
	} {
		d := openRadarMap(t, "IEM", &asked)
		gen := d.mapPane.gen
		d = event(d)
		if d.mapPane.gen == gen {
			t.Errorf("%s did not draw the map", name)
		}
		assertFresh(t, d, name)
	}
}

// shiftKey presses shift and an arrow: the timeline's step keys (D-86).
func shiftKey(d Dashboard, code rune) Dashboard {
	m, _ := d.Update(tea.KeyPressMsg{Code: code, Mod: tea.ModShift})
	return m.(Dashboard)
}

// TestALaterAskWaitsForTheLoop is D-85: while the loop is being fetched, a
// later ask - a settle, new data - does not replace it; it is kept, and
// asked once the answer lands, which is applied.
func TestALaterAskWaitsForTheLoop(t *testing.T) {
	var asked []string
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: radarFeed(t, "MRMS", &asked), MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: true}}})
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	d = feedAndSettle(t, m.(Dashboard))
	if !d.mapPane.radarBusy || !strings.Contains(stripANSITest(d.mapStatusLine()), "Radar loading") {
		t.Fatalf("while the loop loads the line is %q (busy %v)", stripANSITest(d.mapStatusLine()), d.mapPane.radarBusy)
	}
	d, again := d.askRadar() // a settle while the first ask runs
	if again != nil || !d.mapPane.radarAgain {
		t.Fatal("a second ask started while the first was running")
	}
	var answer mapRadarMsg
	for _, msg := range msgsOf(cmd) {
		if r, ok := msg.(mapRadarMsg); ok {
			answer = r
		}
	}
	m, next := d.applyMapRadar(answer)
	d = m.(Dashboard)
	if len(d.mapPane.radarGiven) == 0 {
		t.Fatal("the answer was not applied")
	}
	if next == nil || !d.mapPane.radarBusy || d.mapPane.radarAgain {
		t.Error("the kept ask was not asked when the answer landed")
	}
	d.mapPane.radarBusy = false
	d.now = func() time.Time { return d.mapPane.radarAt.Add(time.Minute) }
	if _, c := d.refreshRadar(); c != nil {
		t.Error("new data a minute after a load asked for the radar again")
	}
	d.now = func() time.Time { return d.mapPane.radarAt.Add(3 * time.Minute) }
	if _, c := d.refreshRadar(); c == nil {
		t.Error("new data three minutes after a load did not ask again")
	}
}

// TestTheTimelineShowsWhereTheLoopIs is D-86: under the map, the shown frame's
// time above its mark, the bar with the step keys at its ends and a mark
// that moves with the frame, the oldest time, OBSERVED and NOW beneath.
func TestTheTimelineShowsWhereTheLoopIs(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked)
	tl := d.mapPane.radarTimeline
	if len(tl) != 3 {
		t.Fatalf("the timeline is %q", tl)
	}
	bar, below := stripANSITest(tl[1]), stripANSITest(tl[2])
	for _, want := range []string{"├", "┤", "█", "shift+←", "shift+→"} {
		if !strings.Contains(bar, want) && !strings.Contains(bar, strings.ReplaceAll(want, "shift+", "")) {
			t.Errorf("the bar %q has no %q", bar, want)
		}
	}
	for _, want := range []string{"OBSERVED", "NOW", "12:00 AM"} {
		if !strings.Contains(below, want) {
			t.Errorf("beneath the bar %q there is no %q", below, want)
		}
	}
	if !strings.Contains(stripANSITest(tl[0]), "12:55 AM") {
		t.Errorf("above the bar %q, not the newest frame's time", stripANSITest(tl[0]))
	}
	was := strings.Index(bar, "█")
	d = shiftKey(d, tea.KeyLeft)
	d = shiftKey(d, tea.KeyLeft)
	now := stripANSITest(d.mapPane.radarTimeline[1])
	if strings.Index(now, "█") >= was {
		t.Errorf("two steps back left the mark where it was: %q then %q", bar, now)
	}
	if !strings.Contains(stripANSITest(d.mapPane.radarTimeline[0]), "12:45 AM") {
		t.Errorf("two steps back the time reads %q", stripANSITest(d.mapPane.radarTimeline[0]))
	}
	if !strings.Contains(bodyText(d), "OBSERVED") {
		t.Error("the timeline is not under the map")
	}
}

// TestTheTimelineNeverMakesTheWindowScroll is D-86 with U1-18: the timeline's
// rows are held in the map's height while radar is on, so the window still
// fits whole - the status and its chips in sight - at every size.
func TestTheTimelineNeverMakesTheWindowScroll(t *testing.T) {
	var asked []string
	for _, size := range [][2]int{{133, 44}, {100, 30}, {80, 24}} {
		d := openRadarMap(t, "MRMS", &asked)
		m, _ := d.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		d = settleMap(t, m.(Dashboard).renderMap())
		if n, most := len(d.modalLines()), d.modalMax(); n > most {
			t.Errorf("%dx%d: the window's body is %d lines against %d: it scrolls", size[0], size[1], n, most)
		}
	}
}

// TestARefusedLoopIsSaidNotLoading is UAT-2 U2-5 as D-124 amends it: a loop
// the library refuses is never read as "loading" for ever; the refusal is
// ours, not the listener's to act on, so it goes to the diagnostics, never
// the line.
func TestARefusedLoopIsSaidNotLoading(t *testing.T) {
	var asked, problems []string
	d := openRadarMap(t, "MRMS", &asked)
	d.cfg.MapProblem = func(p string) { problems = append(problems, p) }
	d.mapPane.call("SetImageBudget", func() { _ = d.mapPane.m.SetImageBudget(1000) })
	r := radarFeed(t, "MRMS", &asked)(context.Background(), d.mapAsk())
	r.Overlays[0].ID = RadarLayer + "/us-b" // a new box: it must fit the budget whole
	d.mapPane.radarGiven = nil
	m, cmd := d.applyMapRadar(mapRadarMsg{radar: r})
	d = settleRadar(t, m.(Dashboard), cmd)
	line := stripANSITest(d.mapStatusLine())
	if strings.Contains(line, "loading") || strings.Contains(line, "could not") || d.radarChipText() != "" {
		t.Errorf("a refused loop reads %q with the chip %q", line, d.radarChipText())
	}
	if len(problems) != 1 || !strings.HasPrefix(problems[0], "Radar") {
		t.Errorf("the diagnostics were told %q; want the radar's refusal", problems)
	}
}

// TestTheRowsUnderTheMapAreTheMocks is D-103 and D-105 (D-89 before them):
// under the picture - the colour row; the warning, one line in the list
// pointer's bold yellow; the picture's status with the estimate; the loop's
// row, RADAR [source] · FRAME · NEWEST · STOPPED in yellow, over the timeline,
// the controls beside them; a blank; MAPS: and the region keys; a blank; the
// chips, with no Legend.
func TestTheRowsUnderTheMapAreTheMocks(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(rendering.ResetColorEnabledForTest)
	var asked []string
	cost := func(MapAsk, func(string) bool) MapCost { return MapCost{Bytes: 2_300_000, Requests: 197} }
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: radarFeed(t, "MRMS", &asked), MapCost: cost,
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: true}}})
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC) }
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	d = settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
	lines := d.mapBodyLines()
	plain := make([]string, len(lines))
	for i, l := range lines {
		plain[i] = stripANSITest(l)
	}
	at := func(want string) int {
		for i, l := range plain {
			if strings.Contains(l, want) {
				return i
			}
		}
		t.Fatalf("no row says %q:\n%s", want, strings.Join(plain, "\n"))
		return -1
	}
	legend, warn, est, loop, bar, maps, chips := at("RADAR LEGEND"), at("Map may experience"), at("Est. 2.3MB / 197 Requests"), at("FRAME"), at("shift+"), at("MAPS:"), at("Area Alerts")
	order := []int{legend, warn, est, loop, bar, maps, chips}
	for i := 1; i < len(order); i++ {
		if order[i] <= order[i-1] {
			t.Errorf("the rows are out of the mock's order: legend, warning, estimate, loop, bar, maps, chips at %v", order)
			break
		}
	}
	if strings.TrimSpace(plain[maps-1]) != "" || strings.TrimSpace(plain[chips-1]) != "" {
		t.Error("no blank before the region row, or before the chips")
	}
	row := plain[loop]
	words := []string{"RADAR  MRMS ", "FRAME 12 / 12", "NEWEST 5 MIN AGO", "STOPPED"}
	last := -1
	for _, w := range words {
		i := strings.Index(row, w)
		if i <= last {
			t.Fatalf("the loop's row is %q; want %v in that order (D-105)", row, words)
		}
		last = i
	}
	if !strings.Contains(lines[loop], render.Tok(render.MapRadarMRMSBG)) || !strings.Contains(lines[loop], strings.Split(render.Tint("§", render.Tok(render.ListPointer)), "§")[0]+"STOPPED") {
		t.Errorf("the loop's row lacks the source's colours or a yellow STOPPED: %q", lines[loop])
	}
	if !strings.Contains(plain[loop], "Controls") || !strings.Contains(plain[bar+1], "place") {
		t.Error("the controls are not beside the timeline")
	}
	if !strings.Contains(plain[maps], "Alaska") || !strings.Contains(plain[maps], "Guam") { // the full names, or the short ones where they do not fit
		t.Errorf("the region row is %q", plain[maps])
	}
	if strings.Contains(plain[chips], "Legend") {
		t.Error("the chips still offer the legend (D-103)")
	}
	if !strings.Contains(plain[legend], "LIGHTER") || !strings.Contains(plain[legend], "HEAVIER") || !strings.Contains(lines[legend], "48;2;") {
		t.Errorf("the colour row is %q", plain[legend])
	}
	if !strings.Contains(plain[warn], "Switch off layers") || strings.Contains(plain[warn], "Est.") {
		t.Errorf("the warning is %q: one line, the estimate on its own row", plain[warn])
	}
	for i := 0; i < d.mapBodySize().Rows && i < len(plain); i++ {
		if strings.Contains(plain[i], "Controls") {
			t.Fatal("the controls box is still on the map")
		}
	}
	if n, most := len(d.modalLines()), d.modalMax(); n > most {
		t.Errorf("the window's body is %d lines against %d: it scrolls", n, most)
	}
}

// TestTheColourRowSaysItsClassesWithoutColour: where colour is off, each
// swatch carries its class's words, so the row still means something.
func TestTheColourRowSaysItsClassesWithoutColour(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked)
	row := d.radarLegendRow(120)
	if row == "" || !strings.Contains(row, "LIGHTER") || strings.Contains(row, "\x1b[") {
		t.Fatalf("without colour the row is %q", row)
	}
	var first string
	for _, e := range d.mapPane.legend {
		if e.Preset == "radar" && len(e.Classes) > 0 {
			first = e.Classes[0].Label
		}
	}
	if first == "" || !strings.Contains(row, first[:min(3, len(first))]) {
		t.Errorf("the row %q does not say its first class %q", row, first)
	}
}

// TestALoopStillPreparingSaysLoadingWithItsSource: the answer is in and the
// library has not prepared the loop yet - the line says loading, beside the
// source's chip (D-89), and never a frame count.
func TestALoopStillPreparingSaysLoadingWithItsSource(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(rendering.ResetColorEnabledForTest)
	d := openMap(t, Config{MapRadar: radarFeed(t, "IEM", &[]string{}), MapLayers: []MapLayer{{Key: RadarLayer, Label: "Radar", On: true}}}, 133, 44)
	d.mapPane.radarSource = "IEM"
	got := d.radarStatus()
	if !strings.Contains(stripANSITest(got), "Radar  IEM  loading") || !strings.Contains(got, render.Tok(render.MapRadarIEMBG)) {
		t.Errorf("a loop not yet prepared reads %q", got)
	}
}

// TestTheRadarBadgeIsATab is D-92 as D-120 redraws it: one line, a tab
// joined to the map frame's top right - the frame's top edge opens into it,
// its bottom edge closes into the frame's right side - reading RADAR DATA,
// the source's chip in its colours, and the frame's time in the listener's
// clock, STALE before it when the newest frame is old. The library's stamp is
// handed to it (go-tuiMaps D-87).
func TestTheRadarBadgeIsATab(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(rendering.ResetColorEnabledForTest)
	var asked []string
	calls := &[]string{}
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: radarFeed(t, "MRMS", &asked),
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: true}}})
	d.mapPane.calls = calls
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC) }
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	d = settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
	if !strings.Contains(strings.Join(*calls, " "), "ShowStamp:false") {
		t.Error("the library's stamp was not handed to the badge")
	}
	rows := strings.Split(d.mapWindow(d.opts()), "\n")
	plain := func(i int) string { return strings.TrimRight(stripANSITest(rows[i]), " ") }
	words := " RADAR DATA  [  MRMS  ]  12:55 AM "
	top, tab, foot := plain(0), plain(1), plain(2)
	at := strings.Index(tab, "│"+words+"│")
	if at < 0 || !strings.HasSuffix(tab, "│"+words+"│") {
		t.Fatalf("the tab's row is %q; want it to end │%s│", tab, words)
	}
	cells := []rune(top)
	col := len([]rune(tab[:at]))
	if col >= len(cells) || string(cells[col]) != "┬" || !strings.HasSuffix(top, "┐") {
		t.Errorf("the frame's top edge is %q; want it to open into the tab with ┬ above its left side", top)
	}
	if !strings.HasSuffix(foot, "└"+strings.Repeat("─", len([]rune(words)))+"┤") {
		t.Errorf("the tab's bottom is %q; want it to close into the frame's right side with ┤", foot)
	}
	if !strings.Contains(rows[1], render.Tok(render.MapRadarMRMSBG)) {
		t.Error("the badge's chip is not in the source's colours")
	}
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 30, 0, 0, time.UTC) }
	d = d.renderMap()
	if tab := stripANSITest(strings.Split(d.mapWindow(d.opts()), "\n")[1]); !strings.Contains(tab, "STALE 12:55 AM") {
		t.Errorf("an old loop's tab reads %q", tab)
	}
}

// aheadFeed is radarFeed with an hour ahead: four HRRR quarter-hours after
// the newest observed frame, a loop of their own, marked forecast.
func aheadFeed(t *testing.T) func(context.Context, MapAsk) MapRadar {
	t.Helper()
	observed := radarFeed(t, "MRMS", new([]string))
	png, err := os.ReadFile("testdata/radar-frame.png")
	if err != nil {
		t.Fatal(err)
	}
	return func(ctx context.Context, ask MapAsk) MapRadar {
		out := observed(ctx, ask)
		newest := time.Date(2026, 8, 24, 0, 55, 0, 0, time.UTC)
		var frames []tuimaps.LoopFrame
		for i := 1; i <= 4; i++ {
			frames = append(frames, tuimaps.LoopFrame{Valid: newest.Add(time.Duration(i) * 15 * time.Minute), PNG: png, Forecast: true})
		}
		fc := tuimaps.RadarImage(RadarLayer+"/fc-us-a", tuimaps.Image{Frames: frames, Provider: tuimaps.ProviderIEM,
			West: -126, South: 23, East: -65, North: 51, Projection: tuimaps.PlateCarree}, newest)
		fc.Keeps, fc.During = 6*time.Hour, tuimaps.Span{From: frames[0].Valid}
		out.Overlays[0].During = tuimaps.Span{Until: newest}
		out.Overlays, out.Ahead = append(out.Overlays, fc), "HRRR"
		return out
	}
}

// TestTheLoopSaysWhenItIsAhead is D-113: stepped past now into the hours
// ahead, the badge reads RADAR FCST with HRRR's chip, and the loop's row
// leads FORECAST; back at now, the radar again.
func TestTheLoopSaysWhenItIsAhead(t *testing.T) {
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: aheadFeed(t),
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: true}}})
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC) }
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	d = settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
	if st := d.mapPane.m.Loop(); st.Count != 16 || st.Forecast {
		t.Fatalf("the loop is %d frames, at a forecast %v; want 12 observed and 4 ahead, opened at now", st.Count, st.Forecast)
	}
	badge := func() string { return stripANSITest(d.radarBadge()) }
	if !strings.Contains(badge(), "RADAR DATA") || !strings.Contains(badge(), "MRMS") {
		t.Errorf("at now the badge is %q", badge())
	}
	d = shiftKey(d, tea.KeyRight)
	if !d.mapPane.m.Loop().Forecast {
		t.Fatal("⇧→ from now did not step into the hours ahead")
	}
	if !strings.Contains(badge(), "RADAR FCST") || !strings.Contains(badge(), "HRRR") {
		t.Errorf("ahead of now the badge is %q; want RADAR FCST and HRRR", badge())
	}
	if row := stripANSITest(d.loopRow(d.scrubW())); !strings.Contains(row, "FORECAST  HRRR") {
		t.Errorf("ahead of now the loop's row is %q; want it to lead FORECAST HRRR", row)
	}
	m, _, _ = d.handleMapKey(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if d = m.(Dashboard); d.mapPane.m.Loop().Forecast || !strings.Contains(badge(), "RADAR DATA") {
		t.Error("n did not return to now")
	}
}

// TestTheHoursAheadAreASetting is D-114: 3 hours by default; → steps 6, 12,
// 1; the file's number opens as chosen; the ask carries it.
func TestTheHoursAheadAreASetting(t *testing.T) {
	d, got := uiDash(t, rowMapRadarAhead)
	body, _, _ := d.focusBody(d.opts())
	if text := stripANSITest(strings.Join(body, "\n")); !strings.Contains(text, "Radar ahead -") || !strings.Contains(text, "3 hours") {
		t.Fatalf("the Maps tab has no hours-ahead row at 3 hours:\n%s", text)
	}
	m, _, _ := d.setupRowKey(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	if d.mapRadarAhead != 6 || d.mapAsk().RadarAhead != 6 {
		t.Errorf("→ gave %d; want 6, in the ask", d.mapRadarAhead)
	}
	m, cmd := d.handleSetupKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	drain(t, m, cmd)
	if got.MapRadarAhead != 6 {
		t.Errorf("esc wrote %d, want 6", got.MapRadarAhead)
	}
	if mapDash(t, Config{MapRadarAhead: 12}).mapRadarAhead != 12 || mapDash(t, Config{}).mapRadarAhead != 3 || mapDash(t, Config{MapRadarAhead: 5}).mapRadarAhead != 3 {
		t.Error("the file's number does not open as chosen, or a number not offered is not the default")
	}
}

// TestTheNewestIsTheNewestObservedFrame is W12.1's defect, found building
// W12.2: with the hours ahead in the loop, its newest frame is the
// forecast's far end, and the loop's row said "NEWEST -55 MIN AGO" - and
// could never say STALE (FR-5.4). The newest is the newest observed frame,
// in the row, the status line and the badge.
func TestTheNewestIsTheNewestObservedFrame(t *testing.T) {
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: aheadFeed(t),
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: true}}})
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC) }
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	d = shiftKey(settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd), tea.KeyRight)
	if row := stripANSITest(d.loopRow(d.scrubW())); !strings.Contains(row, "NEWEST 5 MIN AGO") {
		t.Errorf("ahead of now the loop's row is %q; want the newest observed frame's age, 5 minutes", row)
	}
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 30, 0, 0, time.UTC) } // the newest observed 35 minutes old; the forecast's end still ahead
	if row := stripANSITest(d.loopRow(d.scrubW())); !strings.Contains(row, "STALE") {
		t.Errorf("35 minutes on, the loop's row is %q; want STALE", row)
	}
	if line := d.radarStatus(); !strings.Contains(line, "35 min ago, stale") {
		t.Errorf("35 minutes on, the status line is %q; want stale", line)
	}
	if badge := stripANSITest(d.radarBadgeTimeNow()); !strings.Contains(badge, "STALE") {
		t.Errorf("35 minutes on, the badge is %q; want STALE", badge)
	}
}

// TestALongTitleGivesWayToTheTab is D-120 in a narrow window: the tab keeps
// its words - the moment and STALE are never hidden (FR-5.4) - and the
// window's title, shortened if it must be, ends before the tab begins.
func TestALongTitleGivesWayToTheTab(t *testing.T) {
	var asked []string
	d := openRadarMap(t, "MRMS", &asked)
	d.width, d.height = 80, 24
	d.mapPane.title = "the contiguous United States " + d.opts().Glyphs().Dot + " Oceanside, California, far too long a name"
	rows := strings.Split(stripANSITest(d.mapWindow(d.opts())), "\n")
	top := []rune(strings.TrimRight(rows[0], " "))
	tee := strings.LastIndex(string(top), "┬")
	if tee < 0 || !strings.Contains(rows[1], "[  MRMS  ]") {
		t.Fatalf("at 80 columns the tab is gone:\n%s\n%s", rows[0], rows[1])
	}
	before := strings.TrimRight(string(top)[:tee], "─")
	if !strings.HasPrefix(string(top), "┌── Map") || !strings.HasSuffix(string(top)[:tee], "─") || strings.HasSuffix(before, "┬") {
		t.Errorf("the title runs into the tab: %q", string(top))
	}
	if !strings.Contains(before, "…") {
		t.Errorf("the shortened title %q does not say it was shortened", before)
	}
}

// TestTheScrubbersNowIsWhereNowIs is UAT-2 U2-33: the cursor was placed by
// frame number and NOW by time, and the loop's frames are five minutes apart
// observed and fifteen ahead - so the newest observed frame drew two-thirds
// along, in the FORECAST half, and the frame under the NOW mark was an hour
// old. The scrubber is one axis, time: at now the cursor is on NOW, a
// forecast frame lies past it, the oldest at the start and the newest at
// the end.
func TestTheScrubbersNowIsWhereNowIs(t *testing.T) {
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false), MapRadar: aheadFeed(t),
		MapLayers: []MapLayer{{Key: AlertLayer, Label: "Alert areas", On: true}, {Key: RadarLayer, Label: "Radar", On: true}}})
	d.now = func() time.Time { return time.Date(2026, 8, 24, 1, 0, 0, 0, time.UTC) }
	m, cmd := d.Update(tea.KeyPressMsg{Code: 'g', Text: "g"})
	d = settleRadar(t, feedAndSettle(t, m.(Dashboard)), cmd)
	s, ok := d.radarScrubber()
	if !ok || len(s.ticks) != 1 {
		t.Fatalf("the scrubber is %+v (%v); want NOW's tick", s, ok)
	}
	if math.Abs(s.cursor-s.ticks[0]) > 1e-9 {
		t.Errorf("at now the cursor is at %.3f and NOW at %.3f; want them one", s.cursor, s.ticks[0])
	}
	st := d.mapPane.m.Loop()
	want := float64(st.Now.Sub(st.Oldest)) / float64(st.Newest.Sub(st.Oldest))
	if math.Abs(s.ticks[0]-want) > 1e-9 {
		t.Errorf("NOW is at %.3f; want %.3f, its time along the loop's", s.ticks[0], want)
	}
	d = shiftKey(d, tea.KeyRight)
	if s, _ := d.radarScrubber(); s.cursor <= s.ticks[0] {
		t.Errorf("a forecast frame is at %.3f, NOW at %.3f; want it past NOW", s.cursor, s.ticks[0])
	}
}
