package tty

// radio_panel.go — the radio panel: controls, marquee, visualizer rows, radio key handling. Split from dashboard.go by the
// quality pass (Q2, pure move); the map of where things happen is
// docs/where-things-happen.md.

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// toggleRadio flips player state (UAT 37: chip labels follow state - pin/
// repeat/visualizer/size). Audio itself lands with B4.
func (d Dashboard) toggleRadio(act term.Action) (Dashboard, bool) {
	switch act {
	case "radio-play":
		if d.cfg.Radio == nil {
			d.radioPlaying = !d.radioPlaying // no player wired: labels still follow state (UAT 39)
			break
		}
		return d.radioToggle()
	case "radio-repeat":
		d.radioRepeat = d.radioRepeat.next() // UAT 93: Off → One → Watchlist
		return d.pushRepeat(), true
	case "radio-mode":
		d.radioMode = d.radioMode.next() // UAT 97: Synth ↔ Nearest Relay
		if radio, mode := d.cfg.Radio, d.radioMode; radio != nil {
			return d.withCmd(func() tea.Msg { radio.SetMode(mode); return nil }), true
		}
	case "radio-viz":
		d.radioViz = !d.radioViz
		if !d.radioViz {
			d.vizBands = nil // off: nothing lingers for the next on
		}
	case "radio-size":
		d.radioMin = !d.radioMin
	case "radio-vol-up":
		d.radioVolume = min(100, d.radioVolume+5)
		d.volFlash, d.volFlashEnd = "+", time.Now().Add(350*time.Millisecond) // green blink (UAT 41)
		return d.radioVolumeCmd()
	case "radio-vol-dn":
		d.radioVolume = max(0, d.radioVolume-5)
		d.volFlash, d.volFlashEnd = "-", time.Now().Add(350*time.Millisecond) // red blink
		return d.radioVolumeCmd()
	default:
		return d, false
	}
	return d, true
}

// volControl renders "VOL [-]█████░░░░░[+] 55" (UAT 41): VOL bold white,
// the level white, [-]/[+] chips that blink red/green while a press is
// being acknowledged (the bar only steps at the 10s, so the blink is the
// per-press feedback).
func (d Dashboard) volControl(o render.Opts, width int) string {
	filled := width * d.radioVolume / 100
	// UAT 42.2: at the floor/ceiling the inert chip mutes like every other
	// control; a live press blinks (the blink outranks the resting state).
	minus, plus := o.KeyCapIf("-", d.radioVolume > 0), o.KeyCapIf("+", d.radioVolume < 100)
	if d.volFlash != "" && time.Now().Before(d.volFlashEnd) {
		if d.volFlash == "+" {
			plus = o.KeyCapWith("+", render.ChipFlashUp)
		} else {
			minus = o.KeyCapWith("-", render.ChipFlashDown)
		}
	}
	// UAT 42.1: the level always occupies 3 cells (0-100) so the row never jitters.
	return render.TintRaw("VOL ", "1;97") + minus +
		render.Tint(strings.Repeat("█", filled), render.Tok(render.RadioAccent)) + strings.Repeat("░", width-filled) +
		plus + " " + render.Tint(fmt.Sprintf("%3d", d.radioVolume), render.Tok(render.TextBright))
}

// radioPanel renders the mock's full-size player frame. STATIC LAYOUT MOCK
// until B4 wires audio (UAT-3.3: render it now to test the design; content
// marked pending). Width-bound through render.Panel (UAT-2E).
func (d Dashboard) radioPanel(fl frameLayout) string {
	fg, bg := render.RadioBlockTone()
	return fl.o.Module(fl.radioRows, fg, bg) // the rows the layout already rendered for the mode in force ([T] Size: Min = two-row player)
}

// radioLines builds the player rows for a layout mode (UAT 35: width-
// responsive - controls wrap, the VOL bar scales, the progress "play" line
// drops when there is no room). ONE builder feeds both rendering and the
// height budget, so what is measured is what is drawn.
func (d Dashboard) radioLines(o render.Opts, compactMode bool) []string {
	_, bg := render.RadioBlockTone()
	inner := o.ModuleInnerWidth(bg)
	parts := radioParts{
		title:    render.Tint("Watchpost Weather Radio", render.Tok(render.RadioAccent)),
		clock:    "", // the max player lays the marquee itself (UAT 90)
		state:    d.radioStateLabel(),
		controls: d.radioControlLines(o, inner),
	}
	if compactMode {
		parts.vol = d.volControl(o, 10) // UAT 41.1: fixed 10 cells so the scrub bar gets the room
		return d.radioCompactRows(inner, parts)
	}
	parts.vol = d.volControl(o, max(10, min(30, inner/5))) // scaled in the full player
	return d.radioMaxRows(inner, parts)
}

// radioParts are the styled player fragments shared by both layouts
// (split from radioLines, P10-04).
type radioParts struct {
	title, vol, clock, state string
	controls                 []string
}

// radioCompactRows is the two-row (plus optional visualizer) player: the
// status row degrades short title -> drop clock -> shorten station, and
// ALWAYS spans the module with the tail right-aligned (UAT 36/40/41).
func (d Dashboard) radioCompactRows(inner int, p radioParts) []string {
	segs := []string{p.title, d.station()} // the min player has no marquee (UAT 90)
	tail := p.vol + "   " + p.state
	fits := func(parts []string) int { return render.Width(strings.Join(parts, "  ")) + 2 + render.Width(tail) }
	if fits(segs) > inner {
		segs[0] = render.Tint("WWRadio", render.Tok(render.RadioAccent))
	}
	if fits(segs) > inner {
		// The name outranks the play bar (UAT 40.3): shorten it to the room
		// left rather than dropping it; only a hopeless width goes title-only.
		room := inner - fits(segs[:1]) - 2
		if loc := d.selectedLocation(); loc != nil && room >= 8 {
			name := truncateTo(loc.Label+" "+loc.Zip, room-3) + "…"
			segs = []string{segs[0], "♪ " + render.Tint(name, render.Tok(render.RadioStation))}
		} else {
			segs = segs[:1]
		}
	}
	// UAT 89: no play line in the narrow player — there is nothing to scrub.
	rows := []string{render.PadBetween(strings.Join(segs, "  "), tail, inner)}
	if d.radioViz {
		rows = append(rows, d.vizRows(inner, 1)...) // UAT 51.B: ONE row between status and controls
	}
	return append(rows, p.controls...)
}

// vizRows draws the visualizer's bracketed rows (UAT 92): CLIAmp's bar
// spectrum over the latest band levels, blank frames while nothing plays.
func (d Dashboard) vizRows(inner, rows int) []string {
	lines := render.Spectrum(d.vizBands, max(0, inner-2), rows)
	for i, l := range lines {
		lines[i] = "[" + l + "]"
	}
	return lines
}

// vizActive: the visualizer has something to draw — Viz is on, the app
// feeds levels, and the player plays or the bars are still settling.
func (d Dashboard) vizActive() bool {
	if !d.radioViz || d.cfg.Spectrum == nil {
		return false
	}
	if d.radioPlaying {
		return true
	}
	for _, v := range d.vizBands {
		if v > 0.01 {
			return true
		}
	}
	return false
}

// armViz starts the visualizer tick when it has work and is not already
// running — one ticker, however many status updates arrive.
func (d Dashboard) armViz() Dashboard {
	if d.vizTicking || !d.vizActive() {
		return d
	}
	d.vizTicking = true
	return d.withCmd(tea.Batch(d.pendingCmd, vizTick()))
}

// vizFrame pulls the next frame of levels and re-arms while there is work.
func (d Dashboard) vizFrame() (tea.Model, tea.Cmd) {
	d.vizTicking = false
	if !d.vizActive() {
		d.vizBands = nil
		return d, nil
	}
	d.vizBands = d.cfg.Spectrum()
	return d.armViz().takeCmd()
}

// radioMaxRows is the full player (UAT 54 mock): title … VOL + state /
// station … clock / [3-row visualizer when on] / play line / controls.
func (d Dashboard) radioMaxRows(inner int, p radioParts) []string {
	// UAT 90: the location keeps its full width; after a 4-cell buffer the
	// marquee (or LIVE RADIO) fills the rest of the row.
	station := d.station()
	lines := []string{
		render.PadBetween(p.title, p.vol+"   "+p.state, inner),
		render.PadTo(station+strings.Repeat(" ", marqueeGap)+d.radioClock(inner-render.Width(station)-marqueeGap), inner),
	}
	if d.radioViz {
		lines = append(lines, d.vizRows(inner, 3)...) // UAT 54: three rows in the full player
	}
	if inner >= 70 {
		lines = append(lines, strings.Repeat("━", inner)) // play line (UAT 35.3)
	}
	return append(lines, p.controls...)
}

// station names the focused location for the player line (B4 plays it).
func (d Dashboard) station() string {
	if d.radioState == "failed" && d.radioDetail != "" { // red-team 0.9.0 F1: the reason, where the station was
		return "✘ " + render.Tint(truncateTo(d.radioDetail, 72), render.Tok(render.AlertDanger))
	}
	if d.radioStation != "" { // B4: the resolved transmitter once tuned
		return "♪ " + render.Tint(d.radioStation, render.Tok(render.RadioStation))
	}
	if loc := d.selectedLocation(); loc != nil {
		return "♪ " + render.Tint(loc.Label+" "+loc.Zip, render.Tok(render.RadioStation)) // UAT 40.4
	}
	return "♪ Station: --"
}

// radioClock is the player's second line: the narration or install
// progress while the player has something to say (B4 synth), else the
// clock placeholder.
func (d Dashboard) radioClock(width int) string {
	if d.radioLive && d.radioState == "playing" { // UAT 79: a relay has no timeline — say what it is
		return render.Tint("LIVE RADIO", render.Tok(render.StatePlaying))
	}
	if d.radioPlaying && d.radioDetail != "" && width >= 8 {
		return render.Tint(marquee(d.radioDetail, width, d.marqueeProgress(time.Now())), render.Tok(render.TextBright))
	}
	return "" // UAT 89: no timeline placeholder — there is never a timeline
}

// marqueeGap separates the location from the marquee on the max player.
const marqueeGap = 4

// marqueeProgress is how far through the current line the voice is (0..1).
func (d Dashboard) marqueeProgress(now time.Time) float64 {
	if d.radioSpoken <= 0 || d.radioSince.IsZero() {
		return 0
	}
	return min(1, max(0, float64(now.Sub(d.radioSince))/float64(d.radioSpoken)))
}

// marquee shows a width-cell window of text that follows the voice
// (UAT 83): the word being spoken sits about a third of the way in; short
// lines are static; the window never runs past the end.
func marquee(text string, width int, progress float64) string {
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	spoken := int(progress * float64(len(runes)))
	offset := min(len(runes)-width, max(0, spoken-width/3))
	return string(runes[offset : offset+width])
}

// radioStateLabel renders the player state (B4): STOPPED grey, PLAYING
// bold green (UAT 7.2e), CONNECTING / RECONNECTING accent, NO STREAM grey.
func (d Dashboard) radioStateLabel() string {
	switch d.radioState {
	case "playing":
		return render.Tint("▶ PLAYING", render.Tok(render.StatePlaying))
	case "connecting":
		return render.Tint("… CONNECTING", render.Tok(render.RadioAccent))
	case "reconnecting":
		return render.Tint("↻ RECONNECTING", render.Tok(render.RadioAccent))
	case "failed":
		return render.Tint("✘ NO STREAM", render.Tok(render.StateStopped))
	}
	if d.radioPlaying { // no player wired (tests / older builds): state follows the toggle
		return render.Tint("▶ PLAYING", render.Tok(render.StatePlaying))
	}
	return render.Tint("■ STOPPED", render.Tok(render.StateStopped))
}

// radioToggle: [space] tunes the focused (or pinned) location, or stops.
func (d Dashboard) radioToggle() (Dashboard, bool) {
	radio := d.cfg.Radio
	if d.radioPlaying {
		d.radioPlaying = false
		return d.withCmd(func() tea.Msg { radio.Stop(); return nil }), true
	}
	loc := d.selectedLocation()
	if loc == nil {
		return d, true
	}
	ref := refOf(*loc)
	d.radioPlaying, d.radioState, d.radioKey = true, "connecting", snapshot.Key(ref)
	return d.withCmd(func() tea.Msg { radio.Tune(ref); return nil }), true
}

// pushRepeat sends the repeat mode and the Watchlist queue (the favourites,
// in order) to the player — on every [r], and again when the watchlist
// changes under Watchlist mode (no-op when unwired).
func (d Dashboard) pushRepeat() Dashboard {
	if d.cfg.Radio == nil {
		return d
	}
	radio, mode, queue := d.cfg.Radio, d.radioRepeat, refsOf(d.snap)
	return d.withCmd(func() tea.Msg { radio.SetRepeat(mode, queue); return nil })
}

// sameRefs reports whether two location lists name the same places in the
// same order.
func sameRefs(a, b []snapshot.LocationRef) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if snapshot.Key(a[i]) != snapshot.Key(b[i]) {
			return false
		}
	}
	return true
}

// radioVolumeCmd pushes the volume to the player (no-op when unwired).
func (d Dashboard) radioVolumeCmd() (Dashboard, bool) {
	if d.cfg.Radio == nil {
		return d, true
	}
	radio, pct := d.cfg.Radio, d.radioVolume
	return d.withCmd(func() tea.Msg { radio.SetVolume(pct); return nil }), true
}

// radioControlLines wraps the player controls to the module width (UAT
// 35.1) - the same smart wrap the footer uses; B4 wires the handlers.
func (d Dashboard) radioControlLines(o render.Opts, inner int) []string {
	// UAT 52: an "On" state reads emphasized - repeat yellow bold, viz green bold.
	onOff := func(b bool, tok render.Token) string {
		if b {
			return render.Tint("On", render.Tok(tok))
		}
		return "Off"
	}
	size := "Max"
	if d.radioMin {
		size = "Min"
	}
	repeat := d.radioRepeat.String() // UAT 93: Off | One | Watchlist; any repeat reads emphasized
	if d.radioRepeat != RepeatOff {
		repeat = render.Tint(repeat, render.Tok(render.RepeatOn))
	}
	// UAT 37: compact, state-driven labels.
	play := "Play"
	if d.radioPlaying {
		play = "Pause"
	}
	segs := []string{
		o.KeyCap("space") + " " + play,       // UAT 39: action label follows state
		o.KeyCap("r") + " Repeat: " + repeat, // [p] Pin retired (UAT 93): Repeat: Watchlist is how the player follows the list
		o.KeyCap("m") + " Mode: " + render.Tint(d.radioMode.String(), render.Tok(render.RadioStation)), // UAT 97
		o.KeyCap("v") + " Viz: " + onOff(d.radioViz, render.VizOn),
		o.KeyCap("V") + " Voice: " + d.voiceChip(), // UAT 84
		o.KeyCap("T") + " Size: " + size,
	}
	return render.WrapSegments(segs, inner, "  ")
}
