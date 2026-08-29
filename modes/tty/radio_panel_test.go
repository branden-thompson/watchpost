package tty

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/third_party/go-studs/rendering"
)

func TestRadioChipLabelsFollowState(t *testing.T) {
	// UAT 37: compact state-driven labels — [r] Repeat: Off|One|Watchlist,
	// [v] Viz: On|Off, [T] Size: Min|Max — and Size: Min renders the two-row
	// player even on a tall terminal. [p] Pin retired (UAT 93).
	m := dash(t)
	v := m.View().Content
	for _, want := range []string{"[space] Play", "Repeat: Off", "Mode: Synth", "Viz: Off", "Size: Max", "STOPPED"} {
		if !strings.Contains(v, want) {
			t.Fatalf("initial label %q missing:\n%s", want, v)
		}
	}
	if strings.Contains(v, "[p]") || strings.Contains(v, "Pin") {
		t.Fatalf("[p] Pin is retired (UAT 93):\n%s", v)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	v = m.View().Content
	for _, want := range []string{"[space] Pause", "PLAYING", "Repeat: One", "Viz: On"} {
		if !strings.Contains(v, want) {
			t.Fatalf("toggled label %q missing:\n%s", want, v)
		}
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'T', Text: "T"})
	d := m.(Dashboard)
	if !strings.Contains(m.View().Content, "Size: Min") || !d.radioMin || len(d.radioLines(d.opts(), true)) < 2 {
		t.Fatalf("Size: Min must render the two-row player")
	}
}

func TestCompactRadioRowSpansModuleAndKeepsName(t *testing.T) {
	// UAT 40: the compact row always spans the module (tail right-aligned),
	// VOL floors at 10 cells, the clock drops before the location name, and
	// the name reads bold bright yellow.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	narrow, _ := m.Update(tea.WindowSizeMsg{Width: 78, Height: 24}) // 70 before the box: its borders and 3-cell insets take 8 (2026-08-28)
	d := narrow.(Dashboard)
	o := d.opts()
	inner := o.BoxInnerWidth()
	row := d.radioLines(o, true)[0]
	plain := stripANSITest(row)
	if w := len([]rune(plain)); w != inner {
		t.Fatalf("compact row must span the module (%d), got %d: %q", inner, w, plain)
	}
	if !strings.HasSuffix(plain, "STOPPED") {
		t.Fatalf("tail must right-align: %q", plain)
	}
	if !strings.Contains(plain, "♪ Oceanside") || strings.Contains(plain, "00:00 / 00:00") {
		t.Fatalf("clock drops before the location name (name may shorten, never vanish): %q", plain)
	}
	if n := strings.Count(plain, "█") + strings.Count(plain, "░"); n != 10 {
		t.Fatalf("VOL must floor at 10 cells, got %d", n)
	}
	if !strings.Contains(row, render.Tint("Oceanside", render.Tok(render.RadioStation))[:len(render.Tint("Oceanside", render.Tok(render.RadioStation)))-4]) { // the token's SGR + the name; the reset follows the full label
		t.Fatalf("station name must be bold bright yellow: %q", row)
	}
}

func TestVolumeControlStepsAndBlinks(t *testing.T) {
	// UAT 41: VOL [-]bar[+] 55; +/- step the level by 5 (bar cells step at
	// the 10s) and the pressed chip blinks green/red as acknowledgement.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	v := stripANSITest(m.View().Content)
	if !strings.Contains(v, "VOL  -") || !strings.Contains(v, " 55") {
		t.Fatalf("initial volume control missing:\n%s", v)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: '+', Text: "+"})
	raw := m.View().Content
	if m.(Dashboard).radioVolume != 60 || !strings.Contains(stripANSITest(raw), " 60") {
		t.Fatalf("plus must step to 60: %d", m.(Dashboard).radioVolume)
	}
	if !strings.Contains(raw, "\x1b[1;97;48;5;28m + ") {
		t.Fatal("[+] must blink green on press")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: '-', Text: "-"})
	if raw := m.View().Content; !strings.Contains(raw, "\x1b[1;97;48;5;124m - ") || m.(Dashboard).radioVolume != 55 {
		t.Fatal("[-] must blink red and step back to 55")
	}
	short, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	if n := strings.Count(stripANSITest(short.View().Content), "█") + strings.Count(stripANSITest(short.View().Content), "░"); n != 10 {
		t.Fatalf("compact VOL bar must be 10 cells, got %d", n)
	}
}

func TestVolumeLevelReservedAndEdgeChipsMute(t *testing.T) {
	// UAT 42: the level is a fixed 3-cell field; at 100 the [+] chip mutes,
	// at 0 the [-] chip mutes.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	if v := stripANSITest(m.View().Content); !strings.Contains(v, "+   55") {
		t.Fatalf("level must render in a 3-cell field ('+   55'):\n%s", v)
	}
	for range 12 { // 55 -> 100
		m, _ = m.Update(tea.KeyPressMsg{Code: '+', Text: "+"})
	}
	d := m.(Dashboard)
	d.volFlash = "" // look past the blink
	if d.radioVolume != 100 || !strings.Contains(d.volControl(d.opts(), 10), "48;2;43;43;43m + ") {
		t.Fatalf("at 100 the [+] chip must mute (vol=%d)", d.radioVolume)
	}
	for range 25 { // 100 -> 0
		m, _ = m.Update(tea.KeyPressMsg{Code: '-', Text: "-"})
	}
	d = m.(Dashboard)
	d.volFlash = ""
	if d.radioVolume != 0 || !strings.Contains(d.volControl(d.opts(), 10), "48;2;43;43;43m - ") {
		t.Fatalf("at 0 the [-] chip must mute (vol=%d)", d.radioVolume)
	}
	if !strings.Contains(stripANSITest(d.volControl(d.opts(), 10)), "+    0") {
		t.Fatal("0 must still occupy the 3-cell field")
	}
}

func TestVizChipTogglesVisualizerRows(t *testing.T) {
	// UAT 51: [v] shows/hides the max player's two visualizer rows; in the
	// min player it inserts one visualizer row between status and controls.
	m := dash(t)
	d := m.(Dashboard)
	o := d.opts()
	if n := len(d.radioLines(o, false)); n != 3 || strings.Contains(strings.Join(d.radioLines(o, false), ""), "VISUALIZER") {
		t.Fatalf("viz off: max player is 3 rows (head, track, controls) without visualizer rows, got %d", n)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	d = m.(Dashboard)
	full := d.radioLines(o, false)
	if len(full) != 6 {
		t.Fatalf("viz on: the max player gains three visualizer rows inside the track (6 rows), got %d: %q", len(full), full)
	}
	for _, r := range full[2:5] { // the rows sit inside the track, blank while nothing plays
		row := stripANSITest(r)
		if !strings.HasPrefix(row, "│") || !strings.HasSuffix(row, "│") || strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(row, "│"), "│")) != "" || render.Width(row) != render.Width(stripANSITest(full[0])) {
			t.Fatalf("visualizer row sits inside the track spanning the module (facelift 2026-08-28): %q", row)
		}
	}
	mini := d.radioLines(o, true)
	if len(mini) != 3 || !strings.HasPrefix(stripANSITest(mini[1]), "│") || !strings.HasSuffix(stripANSITest(mini[1]), "│") {
		t.Fatalf("viz on: min player gets one visualizer row (in the track) between status and controls: %q", mini)
	}
}

func TestVisualizerAnimatesWhilePlayingAndSettlesAfter(t *testing.T) {
	// UAT 92: with Viz on and the player playing, the dashboard pulls a
	// frame of band levels from the app on a fast tick and draws CLIAmp-style
	// bars in the visualizer rows; when playback stops the bars follow the
	// feed down to rest and the tick ends. Viz off = no tick at all.
	levels := []float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	fr := &fakeRadio{}
	m, err := NewDashboard(Config{Version: "t", Radio: fr, Spectrum: func() []float64 { return levels }})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	model, cmd := model.Update(RadioStatusMsg{State: "playing", Station: "KEC49", Volume: 55})
	if cmd != nil {
		t.Fatal("Viz off: playing never starts the visualizer tick")
	}
	model, cmd = model.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	if cmd == nil {
		t.Fatal("Viz on while playing arms the visualizer tick")
	}
	model, cmd = model.Update(vizTickMsg{})
	d := model.(Dashboard)
	rows := d.radioLines(d.opts(), false)[2:5]
	if !strings.Contains(stripANSITest(rows[0]), "█") || !strings.Contains(stripANSITest(rows[2]), "█") {
		t.Fatalf("full bands draw solid blocks on every visualizer row: %q", rows)
	}
	if cmd == nil {
		t.Fatal("the tick re-arms while playing")
	}
	// Stop: the feed decays; the tick keeps going until the bars rest.
	levels = []float64{0.5, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	model, _ = model.Update(RadioStatusMsg{State: "stopped", Volume: 55})
	model, cmd = model.Update(vizTickMsg{})
	if cmd == nil || !strings.Contains(stripANSITest(model.(Dashboard).radioLines(d.opts(), false)[4]), "█") {
		t.Fatal("stopped with bars still up: one more frame, tick re-armed")
	}
	levels = make([]float64, 10)
	model, cmd = model.Update(vizTickMsg{})
	if cmd != nil {
		t.Fatal("bars at rest and nothing playing: the tick ends")
	}
	if row := stripANSITest(model.(Dashboard).radioLines(d.opts(), false)[3]); strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(row, "│"), "│")) != "" {
		t.Fatalf("at rest the rows are blank frames inside the track: %q", row)
	}
	// A second arm while already ticking never doubles the ticker.
	levels = []float64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	model, cmd = model.Update(RadioStatusMsg{State: "playing", Volume: 55})
	if cmd == nil {
		t.Fatal("playing again with Viz on re-arms")
	}
	if _, cmd = model.Update(RadioStatusMsg{State: "playing", Volume: 55}); cmd != nil {
		t.Fatal("already ticking: a status update does not add a second ticker")
	}
}

func TestOnStateLabelsEmphasized(t *testing.T) {
	// UAT 52: 'Repeat: On' reads yellow bold, 'Viz: On' green bold.
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	m := dash(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	v := m.View().Content
	if !strings.Contains(v, "Repeat: "+strings.TrimSuffix(render.Tint("One", render.Tok(render.RepeatOn)), "\x1b[0m")) {
		t.Fatalf("Repeat: One must be yellow bold:\n%s", v)
	}
	if !strings.Contains(v, "Viz: \x1b[1;38;5;77mOn") {
		t.Fatalf("Viz: On must be green bold:\n%s", v)
	}
	if off := stripANSITest(dash(t).View().Content); !strings.Contains(off, "Repeat: Off") || !strings.Contains(off, "Viz: Off") {
		t.Fatal("Off states stay plain")
	}
}

func TestMaxPlayerLayoutPerMock(t *testing.T) {
	// UAT 54: row 1 = title … VOL + state; row 2 = station … clock; both
	// right-aligned to the module edge.
	m := dash(t)
	d := m.(Dashboard)
	o := d.opts()
	rows := d.radioLines(o, false)
	r1, r2 := stripANSITest(rows[0]), stripANSITest(rows[1])
	if !strings.HasPrefix(r1, " WATCHPOST WEATHER RADIO") || !strings.HasSuffix(r1, "STOPPED") || !strings.Contains(r1, "VOL") { // one cell in (the box's rhythm)
		t.Fatalf("row 1: %q", r1)
	}
	if !strings.HasPrefix(r2, "│░") || !strings.HasSuffix(r2, "░│") || strings.Contains(r2, "00:00 / 00:00") { // the idle track (facelift 2026-08-28); UAT 89: no placeholder clock
		t.Fatalf("row 2: %q", r2)
	}
	if !strings.Contains(r1, "♪ Oceanside") { // the station rides the head now
		t.Fatalf("row 1 carries the station: %q", r1)
	}
}

func TestRadioModeChipTogglesSynthAndNearestRelay(t *testing.T) {
	// UAT 97: [m] Mode: Synth | Nearest Relay — the persisted mode seeds the
	// chip; each press flips it and pushes it to the player; help lists it.
	fr := &fakeRadio{}
	m, _ := NewDashboard(Config{Version: "t", Radio: fr})
	var model tea.Model = m.WithRadioMode(ParseRadioMode("relay"))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	if v := stripANSITest(model.View().Content); !strings.Contains(v, "[m] Mode: Nearest Relay") {
		t.Fatalf("the persisted mode seeds the chip:\n%s", v)
	}
	model, cmd := model.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if cmd == nil {
		t.Fatal("[m] pushes the mode to the player")
	}
	runCmd(cmd)
	if fr.mode != ModeSynth || !strings.Contains(stripANSITest(model.View().Content), "[m] Mode: Synth") {
		t.Fatalf("one press: Synth, got %v", fr.mode)
	}
	model, cmd = model.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	runCmd(cmd)
	if fr.mode != ModeRelay {
		t.Fatalf("second press: Nearest Relay, got %v", fr.mode)
	}
	if ModeRelay.Key() != "relay" || ModeSynth.Key() != "synth" || ParseRadioMode("bogus") != ModeSynth {
		t.Fatal("persisted forms")
	}
	model, _ = model.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if v := stripANSITest(model.View().Content); !strings.Contains(v, "Radio Mode") {
		t.Fatalf("help lists [m]:\n%s", v)
	}
}

func TestRepeatCyclesOffOneWatchlistAndTheRowFollowsTheDeck(t *testing.T) {
	// UAT 93: [r] cycles Off → One → Watchlist → Off; Watchlist hands the
	// player the favourites in order as its queue; when the player advances
	// it reports the location and the ▶ row follows; a watchlist change
	// under Watchlist mode re-sends the queue; Off clears the ∞.
	fr := &fakeRadio{}
	m, _ := NewDashboard(Config{Version: "t", Radio: fr})
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	two := snap()
	two.Locations = append(two.Locations, snapshot.Location{Label: "Carlsbad, CA", Zip: "92008", Lat: 33.16, Lon: -117.35})
	model, _ = model.Update(SnapshotMsg{Snap: two})
	press := func(r rune) {
		var cmd tea.Cmd
		model, cmd = model.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		if cmd != nil {
			runCmd(cmd)
		}
	}
	press('r')
	press('r')
	if fr.repeat != RepeatWatchlist || !strings.Contains(stripANSITest(model.View().Content), "Repeat: Watchlist") {
		t.Fatalf("two presses: Watchlist, got %v", fr.repeat)
	}
	want := refsOf(two)
	if !sameRefs(fr.queue, want) || len(want) != 2 {
		t.Fatalf("Watchlist queue is the favourites in order: %v", fr.queue)
	}
	// The deck advances to the second favourite: the playing mark (∞ while
	// repeating, in the ▶ column) moves from row 001 to row 002.
	model, _ = model.Update(RadioStatusMsg{State: "playing", Station: "Watchpost Synth · " + want[1].Label, Location: snapshot.Key(want[1]), Volume: 55})
	rows := strings.Split(stripANSITest(model.View().Content), "\n")
	var first, second string
	for _, line := range rows {
		if strings.Contains(line, "001.") {
			first = string([]rune(line)[:12])
		}
		if strings.Contains(line, "002.") {
			second = string([]rune(line)[:12])
		}
	}
	if !strings.Contains(second, "∞") || strings.Contains(first, "∞") || strings.Contains(first, "▶") {
		t.Fatalf("the playing mark follows the deck to row 002: first %q second %q", first, second)
	}
	// A changed watchlist re-sends the queue while in Watchlist mode.
	fr.queue = nil
	grown := snap() // a fresh snapshot (the model holds `two` — never mutate what it sees)
	grown.Locations = append(append([]snapshot.Location(nil), two.Locations...), snapshot.Location{Label: "Julian, CA", Zip: "92036", Lat: 33.08, Lon: -116.60})
	var cmd tea.Cmd
	model, cmd = model.Update(SnapshotMsg{Snap: grown})
	if cmd == nil {
		t.Fatal("a watchlist change under Watchlist mode re-sends the queue")
	}
	runCmd(cmd)
	if len(fr.queue) != len(want)+1 {
		t.Fatalf("queue follows the watchlist: %d", len(fr.queue))
	}
	if _, cmd = model.Update(SnapshotMsg{Snap: grown}); cmd != nil {
		t.Fatal("an unchanged watchlist sends nothing")
	}
	press('r')
	if fr.repeat != RepeatOff || !strings.Contains(stripANSITest(model.View().Content), "Repeat: Off") {
		t.Fatalf("third press: Off, got %v", fr.repeat)
	}
	if strings.Contains(stripANSITest(model.View().Content), "∞") {
		t.Fatal("Off: no row wears ∞")
	}
	// Help lists no [p] any more.
	model, _ = model.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if v := stripANSITest(model.View().Content); strings.Contains(v, "Pin") {
		t.Fatalf("help must not list the retired Pin control:\n%s", v)
	}
}

func TestRadioSpaceTunesFocusedLocationAndStatusDrivesLabels(t *testing.T) {
	// B4: [space] asks the app to tune the focused location; the player's
	// status message drives the state label and the station line; [space]
	// again stops; [+]/[-] push the volume.
	fr := &fakeRadio{}
	m, err := NewDashboard(Config{Version: "t", Radio: fr})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	m2, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if cmd == nil {
		t.Fatal("space must queue the tune command")
	}
	runCmd(cmd)
	if len(fr.tuned) != 1 || fr.tuned[0] != "Oceanside, CA" {
		t.Fatalf("tune calls = %v", fr.tuned)
	}
	if v := stripANSITest(m2.View().Content); !strings.Contains(v, "CONNECTING") {
		t.Fatalf("state must read CONNECTING while the app resolves:\n%s", v)
	}
	m3, _ := m2.Update(RadioStatusMsg{State: "playing", Station: "KEC49 Monterey CA 162.550 MHz · 78 mi", Detail: "wxradio.org", Volume: 55, Live: true})
	v := stripANSITest(m3.View().Content)
	if !strings.Contains(v, "▶ PLAYING") || !strings.Contains(v, "♪ KEC49 Monterey CA 162.550 MHz") || !strings.Contains(v, "[space] Pause") && !strings.Contains(v, " space  Pause") {
		t.Fatalf("playing status must drive the label, station line and control:\n%s", v)
	}
	if !strings.Contains(v, "LIVE RADIO") || strings.Contains(v, "00:00 / 00:00") { // UAT 79: a relay has no timeline
		t.Fatalf("live relay must read LIVE RADIO instead of a timeline:\n%s", v)
	}
	// UAT 80: the playing location's row wears ▶ in the radio column; others do not.
	playingRows := 0
	for _, line := range strings.Split(v, "\n") {
		if strings.Contains(line, "001.") && strings.Contains(line, "Oceanside, CA") {
			if r := []rune(line); len(r) > 5 && !strings.Contains(string(r[:8]), "▶") {
				t.Fatalf("playing row must show ▶ in the radio column: %q", string(r[:12]))
			}
			playingRows++
		} else if strings.Contains(line, "▶") && strings.Contains(line, "ºF") {
			t.Fatalf("only the playing location shows ▶: %q", line)
		}
	}
	if playingRows == 0 {
		t.Fatal("playing row not found")
	}
	synthM, _ := m2.Update(RadioStatusMsg{State: "playing", Station: "Watchpost Synth · Oceanside, CA", Detail: "Tonight. Mostly clear.", Volume: 55})
	if sv := stripANSITest(synthM.View().Content); !strings.Contains(sv, "Tonight. Mostly clear.") || strings.Contains(sv, "LIVE RADIO") {
		t.Fatalf("synth shows the narration line:\n%s", sv)
	}
	m4, cmd := m3.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	runCmd(cmd)
	if fr.stops != 1 {
		t.Fatal("second space must stop")
	}
	_, cmd = m4.Update(tea.KeyPressMsg{Code: '+', Text: "+"})
	runCmd(cmd)
	if fr.vol != 60 {
		t.Fatalf("volume must push to the player: %d", fr.vol)
	}
	m5, _ := m4.Update(RadioStatusMsg{State: "failed", Station: "KEC62 San Diego", Detail: "not relayed", Volume: 60})
	if v := stripANSITest(m5.View().Content); !strings.Contains(v, "NO STREAM") {
		t.Fatalf("failed status must read NO STREAM:\n%s", v)
	}
}

func TestMarqueeFollowsTheVoiceAndRepeatWiresThrough(t *testing.T) {
	// UAT 83: the marquee window tracks spoken progress; [r] pushes repeat
	// to the player and the playing row wears ∞.
	text := "This is a long narration line that will not fit the window and must scroll with the voice as it is spoken aloud."
	if got := marquee(text, 30, 0); got != text[:30] {
		t.Fatalf("start: %q", got)
	}
	mid := marquee(text, 30, 0.5)
	if mid == text[:30] || len([]rune(mid)) != 30 || !strings.Contains(text, mid) {
		t.Fatalf("midway the window must have moved: %q", mid)
	}
	if got := marquee(text, 30, 1); got != text[len(text)-30:] {
		t.Fatalf("end: %q", got)
	}
	if got := marquee("short", 30, 0.7); got != "short" {
		t.Fatal("short lines are static")
	}
	fr := &fakeRadio{}
	m, _ := NewDashboard(Config{Version: "t", Radio: fr})
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	m2, cmd := model.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if cmd == nil {
		t.Fatal("[r] must push repeat to the player")
	}
	runCmd(cmd)
	if fr.repeat != RepeatOne {
		t.Fatal("one press: Repeat: One")
	}
	m3, cmd := m2.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	runCmd(cmd)
	m4, _ := m3.Update(RadioStatusMsg{State: "playing", Station: "Watchpost Synth · Oceanside, CA", Detail: "Tonight. Mostly clear.", Volume: 55, Spoken: 3 * time.Second})
	v := stripANSITest(m4.View().Content)
	for _, line := range strings.Split(v, "\n") {
		if strings.Contains(line, "001.") && strings.Contains(line, "Oceanside, CA") {
			if !strings.Contains(string([]rune(line)[:8]), "∞") {
				t.Fatalf("repeat on: the playing row wears ∞: %q", string([]rune(line)[:12]))
			}
		}
	}
}

func TestMaxPlayerMarqueeFillsTheRowAfterTheLocation(t *testing.T) {
	// UAT 90: max player line 2 = full location name + 4 cells + the marquee
	// filling the rest of the row; the min player carries no marquee.
	m, _ := NewDashboard(Config{Version: "t", Radio: &fakeRadio{}})
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 60})
	model, _ = model.Update(SnapshotMsg{Snap: snap()})
	long := strings.Repeat("Tonight mostly clear with patchy fog overnight and lows near seventy. ", 4)
	model, _ = model.Update(RadioStatusMsg{State: "playing", Station: "Watchpost Synth · Oceanside, CA", Detail: long, Volume: 55, Spoken: 20 * time.Second})
	d := model.(Dashboard)
	o := d.opts()
	_ = render.RadioBlockTone
	inner := o.BoxInnerWidth()
	rows := d.radioLines(o, false)
	l2 := stripANSITest(rows[1])
	if !strings.HasPrefix(l2, "│") || !strings.HasSuffix(l2, "│") || !strings.Contains(l2, "Tonight mostly clear") {
		t.Fatalf("the marquee rides the track (facelift 2026-08-28): %q", l2)
	}
	if w := len([]rune(l2)); w != inner {
		t.Fatalf("the track must fill the row exactly (%d of %d)", w, inner)
	}
	if h := stripANSITest(rows[0]); !strings.Contains(h, "♪ Watchpost Synth · Oceanside, CA") {
		t.Fatalf("the station rides the head: %q", h)
	}
	min := stripANSITest(strings.Join(d.radioLines(o, true), "\n"))
	if strings.Contains(min, "Tonight mostly") {
		t.Fatalf("the min player has no marquee:\n%s", min)
	}
}

// HUM LEAD 2026-08-27: play A, navigate to B, [space] plays B (does not
// stop); [space] on the location already playing stops. This was the
// "relay only works on the second press" bug — the first press was read as
// a stop because the previous location was still marked playing.
func TestSpaceRetunesToTheFocusedLocationNotStop(t *testing.T) {
	fr := &fakeRadio{}
	m, err := NewDashboard(Config{Version: "t", Radio: fr})
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 133, Height: 44})
	// Two favourites, so navigation moves the focus.
	s := snap()
	s.Locations = append(s.Locations, snapshot.Location{Label: "Reno, NV", Zip: "89501", Lat: 39.53, Lon: -119.81,
		Harmonized: snapshot.Conditions{Temp: f64(20), Source: snapshot.SourceInfo{Provider: "nws"}},
		Daily:      []snapshot.Daily{{Date: "2026-08-24", TempMax: f64(30), TempMin: f64(15)}}})
	model, _ = model.Update(SnapshotMsg{Snap: s})
	// Play A.
	m2, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	runCmd(cmd)
	m2, _ = m2.Update(RadioStatusMsg{State: "playing", Station: "A", Detail: "wxradio.org", Volume: 55, Live: true, Location: snapshot.Key(snapshot.LocationRef{Lat: s.Locations[0].Lat, Lon: s.Locations[0].Lon})})
	if len(fr.tuned) != 1 || fr.tuned[0] != "Oceanside, CA" {
		t.Fatalf("A tuned: %v", fr.tuned)
	}
	// Navigate to B and press space once: it must TUNE B, not stop.
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m4, cmd := m3.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	runCmd(cmd)
	if fr.stops != 0 {
		t.Fatalf("space on a different location must not stop (stops=%d)", fr.stops)
	}
	if len(fr.tuned) != 2 || fr.tuned[1] != "Reno, NV" {
		t.Fatalf("space on B tunes B in one press: %v", fr.tuned)
	}
	// Now playing B; space on B (still focused) stops.
	m5, _ := m4.Update(RadioStatusMsg{State: "playing", Station: "B", Detail: "wxradio.org", Volume: 55, Live: true, Location: snapshot.Key(snapshot.LocationRef{Lat: s.Locations[1].Lat, Lon: s.Locations[1].Lon})})
	_, cmd = m5.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	runCmd(cmd)
	if fr.stops != 1 {
		t.Fatalf("space on the location already playing stops: stops=%d", fr.stops)
	}
}

// The player's facelift (HUM LEAD UAT 2026-08-28): the head reads
// "WATCHPOST WEATHER RADIO • ♪ station … VOL + state"; under it the marquee
// TRACK (│…│, ░ when idle, the voice's window while it speaks); the
// visualizer rows sit inside the track; no play line; a narrow player drops
// the title, then reads the station's short form; wrapped controls centre.
func TestRadioFaceliftHeadTrackAndControls(t *testing.T) {
	wide := benchDash(t, 200, 60).(Dashboard)
	wide.radioMin = false
	rows := wide.radioLines(wide.opts(), false)
	plain := make([]string, len(rows))
	for i, r := range rows {
		plain[i] = stripANSITest(r)
	}
	if !strings.HasPrefix(plain[0], " WATCHPOST WEATHER RADIO • ♪ ") || !strings.Contains(plain[0], "VOL") || !strings.HasSuffix(strings.TrimRight(plain[0], " "), "STOPPED") { // one cell in: the box's rhythm
		t.Fatalf("wide head: %q", plain[0])
	}
	if !strings.HasPrefix(plain[1], "│") || !strings.HasSuffix(plain[1], "│") || !strings.Contains(rows[1], render.Tok(render.GroupSectionBG)) { // colour on (benchDash): the band, not the ░ fill
		t.Fatalf("an idle track is the section band between the rails: %q", plain[1])
	}
	for _, r := range plain {
		if strings.Contains(r, "━") {
			t.Fatalf("no play line: %q", r)
		}
	}
	// The voice's window rides the track.
	wide.radioPlaying, wide.radioState, wide.radioDetail, wide.radioSpoken, wide.radioSince = true, "playing", "Watchpost Weather Radio forecasts may be delayed", 10*time.Second, time.Now()
	if track := stripANSITest(wide.marqueeTrack(80)); !strings.Contains(track, "forecasts may be delayed") || !strings.HasPrefix(track, "│ ") || !strings.HasSuffix(track, " │") {
		t.Fatalf("the spoken text rides the track centred on the band: %q", track)
	}
	wide.radioLive = true
	if track := stripANSITest(wide.marqueeTrack(40)); !strings.Contains(track, "LIVE RADIO") {
		t.Fatalf("a relay says so on the track: %q", track)
	}
	// Viz rows sit inside the track: three wide, framed by the rails.
	wide.radioViz = true
	rows = wide.radioLines(wide.opts(), false)
	viz := 0
	for _, r := range rows[2:] {
		if p := stripANSITest(r); strings.HasPrefix(p, "│") && strings.HasSuffix(p, "│") && !strings.Contains(p, "░") {
			viz++
		}
	}
	if viz != 3 {
		t.Fatalf("three visualizer rows inside the track, got %d:\n%s", viz, strings.Join(rows, "\n"))
	}
	// Narrow: the title goes, the station's short form reads, one viz row, the wrapped controls centre.
	narrow := benchDash(t, 96, 44).(Dashboard) // the box's inner at 96 cols holds the short station beside the 10-cell bar
	narrow.radioMin, narrow.radioViz = false, true
	narrow.radioStation, narrow.radioShort, narrow.radioState, narrow.radioPlaying = "EVENT · Special Weather Statement · Palomar Mountain, CA", "EVENT · SPS · Palomar Mountain, CA", "playing", true
	rows = narrow.radioLines(narrow.opts(), false)
	head := stripANSITest(rows[0])
	if strings.Contains(head, "WATCHPOST") || !strings.Contains(head, "EVENT · SPS · Palomar Mountain, CA") || !strings.Contains(head, "PLAYING") {
		t.Fatalf("narrow head drops the title and reads the short station: %q", head)
	}
	inner := narrow.opts().BoxInnerWidth()
	for _, r := range rows {
		if render.Width(r) > inner {
			t.Fatalf("a player row overflows the module: %q", stripANSITest(r))
		}
	}
	if viz := stripANSITest(rows[2]); !strings.HasPrefix(viz, "│") || !strings.HasSuffix(viz, "│") {
		t.Fatalf("one viz row in the narrow track: %q", viz)
	}
	controls := rows[3:]
	if len(controls) < 2 {
		t.Fatalf("the narrow controls wrap: %v", controls)
	}
	if last := controls[len(controls)-1]; !strings.HasPrefix(last, "   ") {
		t.Fatalf("a wrapped continuation centres: %q", stripANSITest(last))
	}
	// Narrower still: the short station itself shortens with an ellipsis, never vanishes.
	tiny := benchDash(t, 84, 44).(Dashboard)
	tiny.radioMin = false
	tiny.radioStation, tiny.radioShort, tiny.radioState, tiny.radioPlaying = narrow.radioStation, narrow.radioShort, "playing", true
	if head := stripANSITest(tiny.radioLines(tiny.opts(), false)[0]); !strings.Contains(head, "♪ EVENT · SPS") || !strings.Contains(head, "…") || strings.Contains(head, "WATCHPOST") {
		t.Fatalf("84 cols: the SHORT station shortens (round 4, B-09), the title stays gone: %q", head)
	}
	// Failed: the reason shortens, never replaced by the station (B-09).
	tiny.radioState, tiny.radioDetail = "failed", "voice cannot render: say: exit status 1 — check the voice in [V], or reinstall it"
	if head := stripANSITest(tiny.radioLines(tiny.opts(), false)[0]); !strings.HasPrefix(strings.TrimSpace(head), "✘ voice cannot render") || !strings.Contains(head, "…") {
		t.Fatalf("84 cols failed: the reason shortens: %q", head)
	}
}

// With colour on the marquee track is a band in the section grey — the
// RECENT / SEARCHED tone under the group-band text (HUM LEAD UAT 2026-08-28).
func TestMarqueeTrackIsASectionBand(t *testing.T) {
	rendering.SetColorEnabledForTest(true)
	defer rendering.SetColorEnabledForTest(false)
	d := dash(t).(Dashboard)
	d.radioPlaying, d.radioState, d.radioDetail, d.radioSpoken, d.radioSince = true, "playing", "Tonight mostly clear.", 10*time.Second, time.Now()
	track := d.marqueeTrack(60)
	if !strings.Contains(track, render.Tok(render.GroupSectionBG)) || !strings.Contains(track, "Tonight mostly clear.") || strings.Contains(track, "░") {
		t.Fatalf("the track carries the section band's grey under the text, no ░: %q", track)
	}
	d.radioPlaying = false
	if idle := d.marqueeTrack(60); !strings.Contains(idle, render.Tok(render.GroupSectionBG)) || strings.Contains(idle, "░") {
		t.Fatalf("an idle track is the band alone: %q", idle)
	}
}
