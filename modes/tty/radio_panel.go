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
	if act == "radio-play" && d.modal == modalSevere {
		return d.readFocusedEvent(), true // inside the window [space] reads the focused EVENT, never the location underneath (0.13.0 UAT option B)
	}
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

// radioPanel renders the player: the rows the layout already built for the
// mode in force, in the heavy box (facelift 2026-08-28).
func (d Dashboard) radioPanel(fl frameLayout) string {
	fg, bg := render.RadioBlockTone()
	return fl.o.Box(fl.radioRows, fg, bg) // a bordered box, the rows inset 3 (facelift 2026-08-28); the rows the layout already rendered for the mode in force ([T] Size: Min = two-row player)
}

// radioLines builds the player rows for a layout mode (UAT 35: width-
// responsive — the control rows wrap and centre, the VOL bar scales, the
// head shortens). ONE builder feeds both rendering and the height budget,
// so what is measured is what is drawn.
func (d Dashboard) radioLines(o render.Opts, compactMode bool) []string {
	inner := o.BoxInnerWidth() // the box's rows: borders and the 3-cell insets off
	parts := radioParts{
		title:    render.Tint("WATCHPOST WEATHER RADIO", render.Tok(render.RadioAccent)), // the facelift's capitals (HUM LEAD UAT 2026-08-28)
		clock:    "",                                                                     // the max player lays the marquee itself (UAT 90)
		state:    d.radioStateLabel(),
		controls: d.radioControlLines(o, inner),
	}
	if compactMode {
		parts.vol = d.volControl(o, 10) // UAT 41.1: fixed 10 cells so the scrub bar gets the room
		return d.radioCompactRows(inner, parts)
	}
	parts.vol = d.volControl(o, max(10, min(30, inner/5))) // scaled in the full player
	return d.radioMaxRows(o, inner, parts)
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
		segs[0] = render.Tint("WWRADIO", render.Tok(render.RadioAccent))
	}
	if fits(segs) > inner {
		// The name outranks the play bar (UAT 40.3): shorten it to the room
		// left rather than dropping it; only a hopeless width goes title-only.
		room := inner - fits(segs[:1]) - 2
		if loc := d.selectedLocation(); loc != nil && room >= 8 {
			name := truncateTo(loc.Label+" "+loc.Zip, room-3) + "…"
			segs = []string{segs[0], d.opts().Glyphs().Note + " " + render.Tint(name, render.Tok(render.RadioStation))}
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
		lines[i] = trackLine(l, inner) // inside the player's track, under the marquee (UAT 2026-08-28)
	}
	return lines
}

// The player's track (HUM LEAD UAT 2026-08-28): the marquee row and the
// visualizer rows sit between two │ rails spanning the module; an idle
// marquee is a ░ fill, the voice's window slides over it while it speaks.
// trackIdle is the marquee band's idle fill — through Glyphs (░, or . under --ascii).
func trackIdle(o render.Opts) string { return o.Glyphs().Fill }

// trackLine frames one row of content between the track's rails.
func trackLine(content string, inner int) string {
	return "│" + render.PadTo(content, max(0, inner-2)) + "│"
}

// marqueeTrack is the marquee row: LIVE RADIO centred on a relay, the
// voice's sliding window while it speaks (short text centred), idle
// otherwise. With colour on the track is a BAND — the section band's grey
// (GroupSectionBG, the RECENT / SEARCHED tone) under the group-band text,
// as the column group labels are drawn (HUM LEAD UAT 2026-08-28); with
// colour off it is the ░ fill.
func (d Dashboard) marqueeTrack(inner int) string {
	w := max(0, inner-2)
	idle := trackIdle(d.opts())
	if render.ColorOn() {
		idle = " " // the band's background carries the track
	}
	fill := func(text string) string {
		n := render.Width(text)
		if n >= w {
			return text
		}
		left := (w - n) / 2
		return strings.Repeat(idle, left) + text + strings.Repeat(idle, w-n-left)
	}
	band := func(content string) string {
		if !render.ColorOn() {
			return trackLine(content, inner)
		}
		return trackLine(render.TintRaw(content, render.Tok(render.GroupText)+";"+render.Tok(render.GroupSectionBG)), inner)
	}
	switch {
	case d.radioLive && d.radioState == "playing": // UAT 79: a relay has no timeline — say what it is
		return band(fill(" LIVE RADIO "))
	case d.radioPlaying && d.radioDetail != "" && w >= 8:
		text := marquee(d.radioDetail, w, d.marqueeProgress(time.Now()))
		if render.Width(text) <= w-2 { // air either side only when both fit — a w-1 line padded to w+1 broke the box (round 4, B-02)
			text = " " + text + " "
		}
		return band(fill(text))
	}
	return band(strings.Repeat(idle, w))
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

// radioMaxRows is the full player (HUM LEAD UAT 2026-08-28 facelift): the
// head — "WATCHPOST WEATHER RADIO • ♪ station" … VOL + state — over the
// marquee track, the visualizer rows inside the track when it is on (three
// wide, one narrow), then the controls. No play line: there was never a
// timeline to scrub. Narrow: the title goes first, then the station reads
// its short form, then it shortens.
func (d Dashboard) radioMaxRows(o render.Opts, inner int, p radioParts) []string {
	// The head keeps its title as long as it can: the VOL bar gives first
	// (the wide bar, then 20 cells, then 10 — the mock's widths), the title
	// only after that.
	head := p.title + " • " + d.station()
	tail := p.vol + "   " + p.state
	room := inner - 2 - render.Width(tail)
	for _, bar := range []int{20, 10} {
		if render.Width(head) <= room {
			break
		}
		tail = d.volControl(o, bar) + "   " + p.state
		room = inner - 2 - render.Width(tail)
	}
	narrow := render.Width(head) > room
	if narrow {
		head = d.narrowHead(room)
	}
	lines := []string{" " + render.PadBetween(head, tail, inner-1), d.marqueeTrack(inner)} // the head one cell in, the track flush with the inset (the mock)
	if d.radioViz {
		rows := 3 // UAT 54: three rows in the full player
		if narrow {
			rows = 1
		}
		lines = append(lines, d.vizRows(inner, rows)...)
	}
	return append(lines, p.controls...)
}

// stationText is the station line's plain text (for shortening).
func (d Dashboard) stationText() string {
	if d.radioStation != "" {
		return d.radioStation
	}
	if loc := d.selectedLocation(); loc != nil {
		return loc.Label + " " + loc.Zip
	}
	return "Station: --"
}

// station names the focused location for the player line (B4 plays it).
func (d Dashboard) station() string {
	g := d.opts().Glyphs()                               // ♪ / ✘, or ~ / x under --ascii (R5-C-13)
	if d.radioState == "failed" && d.radioDetail != "" { // red-team 0.9.0 F1: the reason, where the station was
		return g.Fail + " " + render.Tint(truncateTo(d.radioDetail, 72), render.Tok(render.AlertDanger))
	}
	if d.radioStation != "" { // B4: the resolved transmitter once tuned
		return g.Note + " " + render.Tint(d.radioStation, render.Tok(render.RadioStation))
	}
	if loc := d.selectedLocation(); loc != nil {
		return g.Note + " " + render.Tint(loc.Label+" "+loc.Zip, render.Tok(render.RadioStation)) // UAT 40.4
	}
	return g.Note + " Station: --"
}

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
		return render.Tint(d.opts().Glyphs().Fail+" NO STREAM", render.Tok(render.StateStopped))
	}
	if d.radioPlaying { // no player wired (tests / older builds): state follows the toggle
		return render.Tint("▶ PLAYING", render.Tok(render.StatePlaying))
	}
	return render.Tint("■ STOPPED", render.Tok(render.StateStopped))
}

// radioToggle: [space] plays the focused location, or stops when that
// location is already the one playing (HUM LEAD 2026-08-27: "play on A,
// navigate to B, space → play B, not stop"). Re-tuning to the focused
// location while another plays is the common case now that a person browses
// the list with the radio on.
func (d Dashboard) radioToggle() (Dashboard, bool) {
	radio := d.cfg.Radio
	loc := d.selectedLocation()
	if loc == nil {
		if d.radioPlaying { // no selection to tune: space still stops
			d.radioPlaying = false
			return d.withCmd(func() tea.Msg { radio.Stop(); return nil }), true
		}
		return d, true
	}
	ref := refOf(*loc)
	if d.radioPlaying && d.radioKey == snapshot.Key(ref) { // already playing THIS location: stop
		d.radioPlaying = false
		return d.withCmd(func() tea.Msg { radio.Stop(); return nil }), true
	}
	// Not playing, or playing a different location: tune the focused one.
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
	lines := render.WrapSegments(segs, inner-2, "  ")
	for i := range lines { // the controls two cells in (the mock); a wrapped continuation centres under the first row
		if i == 0 {
			lines[i] = "  " + lines[i]
			continue
		}
		if pad := (inner - render.Width(lines[i])) / 2; pad > 0 {
			lines[i] = strings.Repeat(" ", pad) + lines[i]
		}
	}
	return lines
}

// narrowHead is the player's head without the title: the station (or the
// failure reason), then its short form, then whichever of those was chosen
// shortened with an ellipsis (round 4, B-09: the short form shortens, the
// failure text stays a failure), and the mark alone when even that will not
// fit (B-10).
func (d Dashboard) narrowHead(room int) string {
	head := d.station()
	g := d.opts().Glyphs()
	failed := d.radioState == "failed" && d.radioDetail != ""
	mark, text, tone := g.Note+" ", render.Plain(d.stationText()), render.Tok(render.RadioStation)
	if failed {
		mark, text, tone = g.Fail+" ", render.PlainLine(d.radioDetail), render.Tok(render.AlertDanger)
	} else if d.radioShort != "" {
		text = d.radioShort
		if render.Width(head) > room {
			head = mark + render.Tint(text, tone)
		}
	}
	switch {
	case render.Width(head) <= room:
		return head
	case room >= 8:
		return mark + render.Tint(truncateTo(text, room-3)+"…", tone)
	}
	return strings.TrimSpace(mark)
}
