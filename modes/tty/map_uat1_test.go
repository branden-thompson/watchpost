package tty

// map_uat1_test.go — 0.18.0 UAT-1's first findings: the window at about 80%
// of the terminal (U1-13), the [A] Area Alerts box (U1-7, D-63), the controls
// box with its blinking chips (U1-11), the title that follows the view
// (U1-12, D-64) and the [O] Overlays menu with the map's detail (U1-9, U1-10,
// D-65).

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

// openMap is the window open at a size with one alert over Oceanside.
func openMap(t *testing.T, cfg Config, w, h int) Dashboard {
	t.Helper()
	if cfg.MapFeed == nil {
		cfg.MapFeed = boxFeed(-117.6, -117.1, false)
	}
	d := mapDash(t, cfg)
	m, _ := d.Update(tea.WindowSizeMsg{Width: w, Height: h})
	d, _ = pressKey(m.(Dashboard), "g")
	return feedAndSettle(t, d)
}

func bodyText(d Dashboard) string { return stripANSITest(strings.Join(d.mapBodyLines(), "\n")) }

// TestTheMapWindowLeavesTheDashboardInSight is U1-13: on a large terminal the
// window is about 80% of it each way, so the dashboard shows around it; on a
// small one it still leaves the 69×12 map room (FR-1.4).
func TestTheMapWindowLeavesTheDashboardInSight(t *testing.T) {
	d := openMap(t, Config{}, 133, 44)
	if w := d.modalWidth(); w < 133*75/100 || w > 133*82/100 {
		t.Errorf("at 133 columns the window is %d wide; want about 80%%", w)
	}
	if rows := len(strings.Split(stripANSITest(d.renderModal(d.opts())), "\n")); rows > 44*82/100 {
		t.Errorf("at 44 rows the window is %d tall; want about 80%%", rows)
	}
	small := openMap(t, Config{MapDescription: "off"}, 80, 24)
	if !small.mapFits() {
		t.Errorf("at 80×24 the map is %+v, under the floor", small.mapBodySize())
	}
}

// TestTheAreaAlertsBoxOpensWithTheMap is U1-7 and D-63: the description is a
// box over the map's upper left, open on every open, its words in the
// window's first rows; A closes and reopens it; the map around it is drawn.
func TestTheAreaAlertsBoxOpensWithTheMap(t *testing.T) {
	d := openMap(t, Config{}, 133, 44)
	lines := strings.Split(bodyText(d), "\n")
	if !strings.Contains(lines[0], "Area Alerts") {
		t.Fatalf("the first row is %q, not the Area Alerts box", lines[0])
	}
	first := -1
	for i, l := range lines {
		if strings.Contains(l, "Oceanside, CA - Currently:") {
			first = i
			break
		}
	}
	if first < 0 || first > 2 {
		t.Errorf("the description starts on row %d; D-63 keeps it in the first rows", first)
	}
	if !strings.ContainsFunc(lines[first], func(r rune) bool { return r >= 0x2800 && r <= 0x28ff }) {
		t.Errorf("the box's row carries no map beside it: %q", lines[first])
	}
	if !strings.Contains(stripANSITest(d.mapStatusLine()), "Area Alerts") {
		t.Error("the status line has no Area Alerts chip")
	}
	d = pressCode(d, 'A', "A")
	if strings.Contains(bodyText(d), "Oceanside, CA - Currently:") {
		t.Error("A did not close the box")
	}
	d, _ = pressKey(d, "g")
	d, _ = pressKey(d, "g")
	if !strings.Contains(bodyText(d), "Area Alerts") {
		t.Error("the next open did not open the box again (D-63: on every open)")
	}
	instead := openMap(t, Config{MapDescription: "instead"}, 133, 44)
	if strings.Contains(bodyText(instead), "Area Alerts") || !strings.Contains(bodyText(instead), "Oceanside, CA - Currently:") {
		t.Error("instead of the picture, the words are not the full text")
	}
}

// TestTheControlsBoxBlinksThePressedKey is U1-11: a box over the map's lower
// right shows the map's keys, and the one pressed blinks, as the other chips
// do, until the tick after its blink ends.
func TestTheControlsBoxBlinksThePressedKey(t *testing.T) {
	t.Setenv("TERM", "xterm-256color") // a blink is an inversion: it shows only in colour
	rendering.SetColorEnabledForTest(true)
	t.Cleanup(rendering.ResetColorEnabledForTest)
	d := openMap(t, Config{}, 133, 44)
	text := bodyText(d)
	for _, want := range []string{"Controls", "←", "→", "↑", "↓", "+", "-"} {
		if !strings.Contains(text, want) {
			t.Errorf("the controls box does not show %q", want)
		}
	}
	before := strings.Join(d.controlsBox(), "\n")
	m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	d = m.(Dashboard)
	if d.mapPane.flash != actMapPanRight || !d.tickNeeded() {
		t.Fatalf("→ blinks %q (tick needed %v)", d.mapPane.flash, d.tickNeeded())
	}
	if strings.Join(d.controlsBox(), "\n") == before {
		t.Error("the pressed chip is drawn as it was")
	}
	d.mapPane.flashEnd = time.Now().Add(-time.Millisecond)
	d = d.applyTick()
	if d.mapPane.flash != "" {
		t.Error("the blink did not end on the tick after it expired")
	}
}

// TestTheTitleNamesWhatIsInView is U1-12 and D-64: the title names the view
// by scale, from the app's namer, with the selected place while it is in
// view.
func TestTheTitleNamesWhatIsInView(t *testing.T) {
	namer := func(_ tuimaps.LonLat, widthKm float64) string {
		switch {
		case widthKm < 80:
			return "Oceanside, CA"
		case widthKm < 700:
			return "Southern California"
		}
		return "California"
	}
	d := openMap(t, Config{MapAreaName: namer}, 133, 44)
	if got := d.mapTitle(); got != "Map · Southern California · Oceanside, CA" {
		t.Errorf("at the state scale the title is %q", got)
	}
	for range 3 {
		d = pressCode(d, '+', "+")
	}
	if got := d.mapTitle(); got != "Map · Oceanside, CA" {
		t.Errorf("zoomed in the title is %q", got)
	}
	for range 12 {
		m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
		d = m.(Dashboard)
	}
	if got := d.mapTitle(); strings.Contains(got, "· Oceanside, CA") && got != "Map · Oceanside, CA" {
		t.Errorf("panned away, the title still names the place: %q", got)
	}
	if got := openMap(t, Config{}, 133, 44).mapTitle(); got != "Map · Oceanside, CA" {
		t.Errorf("with no namer the title is %q, want the place", got)
	}
}

// TestTheOverlaysMenuSwitchesLayersAndDetail is U1-9, U1-10 and D-65: O opens
// a menu of the weather layers and the map's detail; while it is open it owns
// ↑↓ and space; the detail opens weather-first; a switch reaches the library
// and is written with the map's Settings.
func TestTheOverlaysMenuSwitchesLayersAndDetail(t *testing.T) {
	calls := &[]string{}
	d := mapDash(t, Config{MapLayers: alertLayers, MapFeed: boxFeed(-117.6, -117.1, false)})
	d.mapPane.calls = calls
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	joined := strings.Join(*calls, " ")
	for _, want := range []string{"SetDetail:weather", "Layers:roads:on", "Layers:minor-roads:on", "Layers:borders:on", "Layers:names:on"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the map was not built weather-first (D-67): no %s in %v", want, *calls)
		}
	}
	d = pressCode(d, 'O', "O")
	text := bodyText(d)
	for _, want := range []string{"Overlays", "Alert areas", "Detail: Weather", "Major roads", "Minor roads (at Full)", "Parks (at Standard)", "Place names"} {
		if !strings.Contains(text, want) {
			t.Errorf("the menu does not list %q", want)
		}
	}
	centre, _ := d.mapPane.m.Centre()
	var rows []string
	for i := range d.overlayRows() {
		rows = append(rows, d.overlayRows()[i].key)
	}
	for range indexOf(rows, "roads") {
		m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		d = m.(Dashboard)
	}
	if now, _ := d.mapPane.m.Centre(); now != centre {
		t.Error("↓ panned the map while the menu was open")
	}
	*calls = nil
	m, _ := d.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	d = m.(Dashboard)
	if !strings.Contains(strings.Join(*calls, " "), "Layers:roads:off") {
		t.Errorf("space on Major roads did not reach the library: %v", *calls)
	}
	if on, ok := d.uiForSave().MapDetail["roads"]; !ok || on {
		t.Errorf("the choice is not written with the map's Settings: %v", d.uiForSave().MapDetail)
	}
	d = pressCode(d, 'O', "O")
	if strings.Contains(bodyText(d), "Parks (at Standard)") {
		t.Error("O did not close the menu")
	}
}

func indexOf(xs []string, x string) int {
	for i, v := range xs {
		if v == x {
			return i
		}
	}
	return -1
}

// msgsOf runs a command, and a batch's commands, each for up to a moment -
// a tick's command waits, and is not what these tests look for.
func msgsOf(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	out := make(chan tea.Msg, 1)
	go func() { out <- cmd() }()
	select {
	case msg := <-out:
		if batch, ok := msg.(tea.BatchMsg); ok {
			var all []tea.Msg
			for _, c := range batch {
				all = append(all, msgsOf(c)...)
			}
			return all
		}
		return []tea.Msg{msg}
	case <-time.After(200 * time.Millisecond):
		return nil
	}
}

// TestADetailSwitchChangesThePictureAndIsSaved: switching a detail layer in
// the menu redraws the map without it, and the choice is written at once -
// there is no Settings window to close.
func TestADetailSwitchChangesThePictureAndIsSaved(t *testing.T) {
	var saved UIPrefs
	d := mapDash(t, Config{MapFeed: boxFeed(-117.6, -117.1, false)})
	d.cfg.SetUI = func(p UIPrefs) error { saved = p; return nil }
	d, _ = pressKey(d, "g")
	d = feedAndSettle(t, d)
	before := strings.Join(d.mapPane.lines, "\n")
	d = pressCode(d, 'O', "O")
	var keys []string
	for _, r := range d.overlayRows() {
		keys = append(keys, r.key)
	}
	d.mapPane.menuAt = indexOf(keys, "borders")
	m, cmd, ok := d.handleMapKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !ok {
		t.Fatal("space in the menu was not handled")
	}
	d = settleMap(t, m.(Dashboard))
	if strings.Join(d.mapPane.lines, "\n") == before {
		t.Error("switching the borders off left the picture as it was")
	}
	for _, msg := range msgsOf(cmd) {
		if _, ok := msg.(uiSavedMsg); ok && saved.MapDetail != nil && !saved.MapDetail["borders"] {
			return
		}
	}
	t.Errorf("the switch was not written: %+v", saved.MapDetail)
}
