// Package tty hosts the live TUI program (modes/ — reads ONLY
// platform/snapshot per the import lint; M5 is structural).
//
// Dashboard is the B3 build of mock rev2 (125-col): header, conditional
// alert module, radio player frame (static until B4), priority table,
// seeded RECENT/SEARCHED table, chip-styled footer. The viewport carries a
// 4-col padding all around and auto-resizes: the table drops column groups
// as the terminal narrows and grows EXTENDED FORECAST columns beyond 125
// (UAT session 2 B/C/D/E). All bindings are D-15 data — only '?' is locked
// (R-3); defaults per D-19: a/A/ctrl+a, f/c units, q quit.
package tty

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/geo"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
	zones "github.com/branden-thompson/watchpost/platform/tz"
)

// SnapshotMsg delivers a fresh priority snapshot (sent by app wiring).
type SnapshotMsg struct{ Snap *snapshot.Snapshot }

// RecentSnapshotMsg delivers the RECENT/SEARCHED pipeline's snapshot — a
// separate slow-cadence scheduler feeds the seeded major-city list (UAT
// session 2A) so the priority pipeline's M1 budget stays untouched.
type RecentSnapshotMsg struct{ Snap *snapshot.Snapshot }

// Viewport padding (UAT 14.3: left back to 3 now the tables are fixed -
// a deliberate reversion; right stays 2 with the rail gutter beyond it).
const (
	viewPadLeft  = 3
	viewPadRight = 2
)

// recentWindow is the RECENT/SEARCHED viewport floor when the terminal is
// short; the window expands to fill tall terminals (UAT 46.1).
const recentWindow = 3

// Config wires the dashboard. Resolve and Commit are app-provided hooks
// (modes cannot import domains — import lint): Resolve turns a typed query
// into a location ref; Commit persists the watchlist and rebuilds the live
// pipelines with the new watch/recent ref sets (UAT 26).
type Config struct {
	Version      string
	KeyOverrides term.KeyMap // user [keys] table (validated at build)
	Resolve      func(query string) (snapshot.LocationRef, error)
	Commit       func(watch, recent []snapshot.LocationRef) error
	SetTheme     func(name string) error                               // live theme switch + persist (UAT 53)
	Voices       func() []string                                       // available correspondent voices (UAT 84)
	SetVoice     func(name string) error                               // choose + persist the voice (UAT 84)
	PreviewVoice func(name string)                                     // speak the sample line in a voice (UAT 86)
	Voice        string                                                // the current voice name, for the [V] chip
	Hydrate      func(ref snapshot.LocationRef)                        // on-demand hourly forecast for a RECENT row (UAT 72)
	Credits      []string                                              // About "Data Provided by" lines — the app owns the list (UAT 75)
	Radio        Radio                                                 // NOAA Weather Radio playback (B4); nil = controls stay inert
	Spectrum     func() []float64                                      // the visualizer feed: the latest band levels 0..1 (UAT 92); nil = rows stay blank
	FireBoldMW   float64                                               // B5: FRP at which a hotspot reads emphasized (the app passes the configured rule; 0 = 50)
	Suggest      func(query string, limit int) []snapshot.LocationRef  // type-ahead hints for the Setup window (embedded index only; nil = enter resolves)
	Setup        func(def snapshot.LocationRef, firmsKey string) error // persist the default location (+ the FIRMS key when given) — the Setup window's finish (UAT 100)
	OpenSetup    bool                                                  // open the Setup window at launch (first run, no locations, or `watchpost setup`)
	FIRMSKey     func() string                                         // the stored FIRMS key's tail ("cdef"), "" when none — the Setup window shows it is there (UAT 111)
}

// Radio is the app-provided player (B4): the dashboard asks, the app
// resolves the station and streams; status comes back as RadioStatusMsg.
type Radio interface {
	Tune(ref snapshot.LocationRef)
	Stop()
	SetVolume(pct int)
	SetRepeat(mode RepeatMode, watchlist []snapshot.LocationRef) // UAT 83/93: [r] Repeat; the watchlist is the Watchlist mode's queue
	SetMode(mode RadioMode)                                      // UAT 97: [m] Synth / Nearest Relay; re-tunes what is playing
}

// RadioMode is the source [m] picks (UAT 97): Synth voices the location's
// own products; Nearest Relay plays the nearest relayed NWR transmitter
// (the covering one when it is relayed), falling back to Synth when none
// is in reach.
type RadioMode int

// The [m] cycle, in order.
const (
	ModeSynth RadioMode = iota
	ModeRelay
)

// String is the chip label.
func (m RadioMode) String() string {
	if m == ModeRelay {
		return "Nearest Relay"
	}
	return "Synth"
}

// next flips the mode.
func (m RadioMode) next() RadioMode { return (m + 1) % 2 }

// ParseRadioMode reads the persisted form ("synth" | "relay"); anything
// else is Synth.
func ParseRadioMode(s string) RadioMode {
	if s == "relay" {
		return ModeRelay
	}
	return ModeSynth
}

// Key is the persisted form.
func (m RadioMode) Key() string {
	if m == ModeRelay {
		return "relay"
	}
	return "synth"
}

// RepeatMode is what [r] cycles (UAT 93): Off plays one broadcast and
// stops; One loops the tuned location; Watchlist plays each favourite in
// turn — the tuned one to the end of its cycle, then the next, around the
// list — which is also how the player follows you across the watchlist.
type RepeatMode int

// The [r] cycle, in order.
const (
	RepeatOff RepeatMode = iota
	RepeatOne
	RepeatWatchlist
)

// String is the chip label.
func (m RepeatMode) String() string {
	switch m {
	case RepeatOne:
		return "One"
	case RepeatWatchlist:
		return "Watchlist"
	}
	return "Off"
}

// next is the [r] cycle: Off → One → Watchlist → Off.
func (m RepeatMode) next() RepeatMode { return (m + 1) % 3 }

// RadioStatusMsg reports the player's condition (B4).
// VoiceNoteMsg is the radio deck's word to the Voice chooser (UAT 119):
// what is happening between a preview or pick and the first sound — a
// download with its progress, the model loading — so a ten-second wait on
// Linux never reads as "broken". "" clears the line.
type VoiceNoteMsg struct{ Text string }

type RadioStatusMsg struct {
	State    string // stopped | connecting | playing | reconnecting | failed
	Station  string // "KEC49 Monterey CA 162.550 MHz · 78 mi (nearest relayed)"
	Detail   string // relay name, in-band title, or the failure reason
	Volume   int
	Live     bool                 // a relayed broadcast (no timeline to show) vs the synthesized one (UAT 79)
	Spoken   time.Duration        // how long Detail will be spoken — paces the marquee (UAT 83)
	Location snapshot.LocationKey // the location being played, so the ▶ row follows a Watchlist advance (UAT 93); "" = unchanged
}

// defaultKeyMap is the dashboard's D-19 default bindings — data, not code.
func defaultKeyMap() term.KeyMap {
	return term.KeyMap{
		term.HelpAction: {Keys: []string{"?"}, Help: "Help"},
		"quit":          {Keys: []string{"q", "ctrl+c"}, Help: "Quit"},
		"about":         {Keys: []string{"a"}, Help: "About"},
		"alert-details": {Keys: []string{"A"}, Help: "Alert Details"},
		"details":       {Keys: []string{"enter"}, Help: "Location Details (forecast, marine, fire)"},
		"add-location":  {Keys: []string{"ctrl+a"}, Help: "Add Location"},
		"status":        {Keys: []string{"S"}, Help: "API Status"},
		"remove":        {Keys: []string{"shift+delete"}, Help: "Remove from Watchlist"},
		"lookup":        {Keys: []string{"l"}, Help: "Lookup Location"},
		"theme":         {Keys: []string{"t"}, Help: "Choose Color Theme"},
		"setup":         {Keys: []string{"s"}, Help: "Setup"},
		"radio-play":    {Keys: []string{"space"}, Help: "Play/Pause Radio"},
		"radio-repeat":  {Keys: []string{"r"}, Help: "Repeat: Off / One / Watchlist"},
		"radio-viz":     {Keys: []string{"v"}, Help: "Visualizer"},
		"radio-mode":    {Keys: []string{"m"}, Help: "Radio Mode: Synth / Nearest Relay"},
		"voice":         {Keys: []string{"V"}, Help: "Correspondent Voice"},
		"radio-size":    {Keys: []string{"T"}, Help: "Toggle Player Size"},
		"radio-vol-up":  {Keys: []string{"+", "="}, Help: "Volume Up"},
		"radio-vol-dn":  {Keys: []string{"-"}, Help: "Volume Down"},
		"units-f":       {Keys: []string{"f"}, Help: "ºF"},
		"units-c":       {Keys: []string{"c"}, Help: "ºC"},
		"nav-up":        {Keys: []string{"up"}, Help: "Navigate"},
		"nav-down":      {Keys: []string{"down"}, Help: "Navigate"},
		"alert-prev":    {Keys: []string{"left"}, Help: "Previous Alert"},
		"alert-next":    {Keys: []string{"right"}, Help: "Next Alert"},
		"close":         {Keys: []string{"esc"}, Help: "Close"},
	}
}

// Dashboard is the root TTY model.
type Dashboard struct {
	cfg          Config
	keys         term.KeyMap
	snap         *snapshot.Snapshot
	recent       *snapshot.Snapshot
	width        int
	height       int
	units        render.Units
	selected     int
	alertIdx     int
	recentOff    int // scroll offset (interaction lands with tab section nav)
	showHelp     bool
	showDetails  bool   // enter: floating forecast details (UAT 10.6)
	showAdd      bool   // ctrl+a / l: search modal (UAT 16.3/26)
	addMode      string // "add" | "lookup" (shared search modal, UAT 26.3/26.4)
	addErr       string // resolve failure surfaced in the modal
	showRemove   bool   // shift+del: remove confirmation (UAT 26.2)
	showAlerts   bool   // A: alert details modal (UAT 22)
	showStatus   bool   // S: API status/diagnostics modal (UAT 24.2)
	showAbout    bool   // a: About window (UAT 68)
	showSetup    bool   // s: Setup window (UAT 100) — the first-run questions, over the dashboard like every other modal
	voiceNote    string // the Voice chooser's progress line (UAT 119): set by VoiceNoteMsg, "" when nothing is pending
	setup        setupState
	showTheme    bool // t: theme chooser (UAT 53)
	themeIdx     int  // chooser cursor
	showVoice    bool // V: voice chooser (UAT 84)
	voiceIdx     int
	voiceErr     string
	voiceList    []string // snapshot of the hook's list, taken when the chooser opens (UAT 85: never from View)
	radioVoice   string   // the chosen correspondent (chip label)
	themeErr     string
	addQuery     string               // add-location search buffer
	modalScroll  int                  // shared scroll for floating modals (UAT 10.4)
	darkBG       bool                 // terminal mode (bubbletea BackgroundColorMsg)
	frame        int                  // animation phase (loading shimmer, UAT 18.2b)
	radioPlaying bool                 // [space] Play|Pause (UAT 39) — true while connecting/playing (B4)
	radioState   string               // B4: last RadioStatusMsg state ("" = never tuned)
	pendingCmd   tea.Cmd              // B4: a command a key handler queued (radio hook calls run off the update loop)
	radioStation string               // B4: resolved station label
	radioDetail  string               // B4: relay / title / failure reason
	radioLive    bool                 // B4: relayed broadcast (UAT 79: "LIVE RADIO" instead of a timeline)
	radioKey     snapshot.LocationKey // B4: the location being played (UAT 80: green ▶ in its row)
	radioSpoken  time.Duration        // UAT 83: spoken length of radioDetail
	radioSince   time.Time            // UAT 83: when radioDetail started
	radioVolume  int                  // 0-100 (D-19); [+]/[-] step 5, bar cells step at the 10s (UAT 41)
	volFlash     string               // "+" | "-" while the press acknowledgement blinks
	volFlashEnd  time.Time
	radioRepeat  RepeatMode // [r] Off | One | Watchlist (UAT 93)
	radioMode    RadioMode  // [m] Synth | Nearest Relay (UAT 97)
	radioViz     bool
	radioMin     bool      // [T] Size: Min renders the two-row player
	vizBands     []float64 // the visualizer's latest frame (UAT 92); nil = blank rows
	vizTicking   bool      // a vizTick is in flight — never two
}

// NewDashboard builds the model, merging user key overrides with validation
// (a conflicting override is a build error, never a silent win — D-15).
func NewDashboard(cfg Config) (Dashboard, error) {
	keys, err := term.Merge(defaultKeyMap(), cfg.KeyOverrides)
	if err != nil {
		return Dashboard{}, fmt.Errorf("key bindings invalid: %w", err)
	}
	d := Dashboard{cfg: cfg, keys: keys, units: render.UnitF, width: 80, height: 24, darkBG: true, radioVolume: 55, radioVoice: cfg.Voice}
	if cfg.OpenSetup {
		d = d.openSetup() // first run: the questions come to the dashboard, not the other way round (UAT 100)
	}
	return d, nil
}

// resolvedMsg returns from the app Resolve hook (ELM: cmd out, msg in).
type resolvedMsg struct {
	mode string
	ref  snapshot.LocationRef
	err  error
}

// committedMsg returns from the app Commit hook.
type committedMsg struct {
	err  error
	what string // "add" | "lookup" | "remove": names the action in the error line (round 2 N-7)
}

// tickMsg drives the loading shimmer (UAT 18.2b).
type tickMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

// vizTickMsg drives the visualizer (UAT 92): 20 frames a second, only while
// there is something to draw — the shimmer tick is far too slow for bars.
type vizTickMsg struct{}

func vizTick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg { return vizTickMsg{} })
}

// Init implements tea.Model — asks the terminal for its background color
// so the window tint tracks light/dark mode (UAT 10.2) and starts the
// animation tick.
func (d Dashboard) Init() tea.Cmd { return tea.Batch(tea.RequestBackgroundColor, tick()) }

// Update implements tea.Model.
func (d Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		d.width, d.height = v.Width, v.Height
		return d, nil
	case SnapshotMsg:
		return d.applySnapshot(v)
	case RecentSnapshotMsg:
		if err := invariant.Check(v.Snap != nil, "nil recent snapshot published to the dashboard"); err != nil {
			return d, nil
		}
		d.recent = v.Snap
		sortAlerts(d.recent)
		return d, nil
	case tea.BackgroundColorMsg:
		d.darkBG = v.IsDark()
		return d, nil
	case resolvedMsg:
		return d.handleResolved(v)
	case committedMsg:
		return d.applyCommitted(v), nil
	case RadioStatusMsg: // B4
		return d.applyRadioStatus(v).armViz().takeCmd()
	case VoiceNoteMsg: // UAT 119
		d.voiceNote = v.Text
		return d, nil
	case tickMsg:
		d.frame++
		return d, tick()
	case vizTickMsg:
		return d.vizFrame()
	case voiceErrMsg:
		d.voiceErr, d.showVoice = v.err.Error(), true
		d.showAdd, d.showRemove, d.showTheme = false, false, false // alone on top (N-7)
		return d, nil
	case tea.KeyPressMsg:
		return d.handleKey(v)
	}
	return d, nil
}

// applySnapshot takes a published watchlist snapshot (split from Update,
// P10-04): alerts sorted, the focus kept in range, and — under Repeat:
// Watchlist — the player's queue re-sent when the list changed (UAT 93).
func (d Dashboard) applySnapshot(v SnapshotMsg) (tea.Model, tea.Cmd) {
	if err := invariant.Check(v.Snap != nil, "nil snapshot published to the dashboard"); err != nil {
		return d, nil
	}
	changed := !sameRefs(refsOf(d.snap), refsOf(v.Snap))
	d.snap = v.Snap
	sortAlerts(d.snap) // UAT 16.2: most severe first, everywhere
	if d.selected >= d.numPriority()+d.numRecent() {
		d.selected = 0
	}
	if changed && d.radioRepeat == RepeatWatchlist {
		d = d.pushRepeat()
	}
	return d.takeCmd()
}

// handleKey routes through the merged KeyMap (D-15: keys are data).
func (d Dashboard) handleKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if d.showSetup {
		return d.handleSetupKey(key)
	}
	if d.showAdd {
		return d.handleAddKey(key)
	}
	if d.showRemove {
		return d.handleRemoveKey(key)
	}
	if d.showTheme {
		return d.handleThemeKey(key)
	}
	if d.showVoice {
		return d.handleVoiceKey(key)
	}
	act, bound := d.keys.Lookup(key.String())
	if !bound {
		return d, nil
	}
	switch act {
	case "quit":
		return d, tea.Quit
	case "units-f":
		d.units = render.UnitF
	case "units-c":
		d.units = render.UnitC
	default:
		if act == "add-location" && d.showDetails {
			return d.addFocused() // detail view: straight to the watchlist (UAT 27 mock)
		}
		if toggled, ok := d.toggleRadio(act); ok {
			return toggled.armViz().takeCmd() // [v] on / [space] play may start the visualizer (UAT 92)
		}
		if toggled, ok := d.toggleModal(act); ok {
			return toggled, toggled.hydrateCmd()
		}
		d = d.handleNav(act)
	}
	return d, nil
}

// addFocused appends the viewed location to the watchlist from the detail
// view (inert when full or already watched - the chip mutes to match).
func (d Dashboard) addFocused() (tea.Model, tea.Cmd) {
	if !d.canAddFocused() {
		return d, nil
	}
	loc := d.selectedLocation()
	watch := append(refsOf(d.snap), snapshot.LocationRef{
		Label: loc.Label, Tag: loc.Tag, Zip: loc.Zip, Lat: loc.Lat, Lon: loc.Lon, TZ: loc.TZ})
	return d, d.commitCmd(watch, withoutRef(refsOf(d.recent), refOf(*loc)), "add") // UAT 106: promoted, not copied
}

// canRemoveFocused: the detail view's − Watchlist chip state — the focused
// location is a favourite (shift+del then opens the Remove confirmation).
func (d Dashboard) canRemoveFocused() bool { return d.selected < d.numPriority() }

// canAddFocused: the detail view's add-to-watchlist chip state.
func (d Dashboard) canAddFocused() bool {
	loc := d.selectedLocation()
	if loc == nil || d.watchlistFull() {
		return false
	}
	for _, r := range refsOf(d.snap) {
		if r.Zip == loc.Zip {
			return false
		}
	}
	return true
}

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

// hydrateCmd asks the app for the hourly forecast when Details opens on a
// RECENT row that has none (UAT 72): the seed list skips the 162 KB hourly
// product on its cadence; a row someone drills into earns it.
func (d Dashboard) hydrateCmd() tea.Cmd {
	if !d.showDetails || d.cfg.Hydrate == nil || d.selected < d.numPriority() {
		return nil
	}
	loc := d.selectedLocation()
	if loc == nil || len(loc.Hourly) > 0 {
		return nil
	}
	ref := refOf(*loc)
	hydrate := d.cfg.Hydrate
	return func() tea.Msg {
		hydrate(ref)
		return nil
	}
}

// toggleModal owns the open/close actions for every floating window (split
// from handleKey, P10-04). Opening one closes the others.
func (d Dashboard) toggleModal(act term.Action) (Dashboard, bool) {
	switch act {
	case term.HelpAction:
		d.showHelp, d.showDetails, d.modalScroll = !d.showHelp, false, 0
	case "details":
		d.showDetails, d.showHelp, d.showAlerts, d.modalScroll = !d.showDetails, false, false, 0 // UAT 10.6
	case "alert-details":
		d.showAlerts, d.showHelp, d.showDetails, d.modalScroll = !d.showAlerts, false, false, 0 // UAT 22
	case "status":
		d.showStatus, d.showHelp, d.showDetails, d.showAlerts, d.showAbout, d.modalScroll = !d.showStatus, false, false, false, false, 0 // UAT 24.2
	case "about":
		d.showAbout, d.showHelp, d.showDetails, d.showAlerts, d.showStatus, d.modalScroll = !d.showAbout, false, false, false, false, 0 // UAT 68
	case "theme":
		d = d.openTheme() // UAT 53
	case "voice":
		d = d.openVoice() // UAT 84
	case "setup":
		d = d.openSetup() // UAT 100
	case "add-location":
		d.showAdd, d.addMode, d.showHelp, d.showDetails, d.addQuery, d.addErr = !d.showAdd, "add", false, false, "", "" // UAT 16.3/26.1
	case "lookup":
		d.showAdd, d.addMode, d.showHelp, d.showDetails, d.addQuery, d.addErr = !d.showAdd, "lookup", false, false, "", "" // UAT 26.4
	case "remove":
		if d.selected < d.numPriority() {
			d.showRemove = true // UAT 26.2: confirm before touching the watchlist
		}
	case "close":
		d.showHelp, d.showDetails, d.showAdd, d.showAlerts, d.showStatus, d.showRemove, d.showTheme, d.showAbout, d.showVoice, d.showSetup, d.modalScroll = false, false, false, false, false, false, false, false, false, false, 0
	default:
		return d, false
	}
	return d, true
}

// handleAddKey owns keys while the add-location modal is open: printable
// keys build the query, so table/global bindings never fire mid-typing.
func (d Dashboard) handleAddKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		if d.addMode == "add" && d.watchlistFull() {
			return d, nil // chip is muted; the press is inert (UAT 26.3)
		}
		if q := strings.TrimSpace(d.addQuery); q != "" {
			return d, d.resolveCmd(q, d.addMode)
		}
		return d, nil
	case "esc", "ctrl+a":
		d.showAdd, d.addQuery, d.addErr = false, "", ""
	case "backspace":
		if r := []rune(d.addQuery); len(r) > 0 {
			d.addQuery = string(r[:len(r)-1])
		}
	default:
		if key.Text != "" {
			d.addQuery += key.Text
		}
	}
	return d, nil
}

// watchlistFull reports the 10-location priority cap (UAT 26.3).
func (d Dashboard) watchlistFull() bool { return d.numPriority() >= 10 }

// resolveCmd asks the app hook to turn the typed query into a ref.
func (d Dashboard) resolveCmd(query, mode string) tea.Cmd {
	res := d.cfg.Resolve
	return func() tea.Msg {
		if res == nil {
			return resolvedMsg{mode: mode, err: fmt.Errorf("search is not wired in this build")}
		}
		ref, err := res(query)
		return resolvedMsg{mode: mode, ref: ref, err: err}
	}
}

// handleResolved routes a resolved location into its flow (UAT 26.3/26.4).
func (d Dashboard) handleResolved(v resolvedMsg) (tea.Model, tea.Cmd) {
	if v.err != nil {
		d.addErr = v.err.Error()
		return d, nil
	}
	if v.mode == "setup" { // the Setup window's location question, answered by a full resolve (no hint matched)
		d.setup.ref, d.setup.focus, d.setup.err = &v.ref, focusKey, ""
		return d, nil
	}
	watch, recent := refsOf(d.snap), refsOf(d.recent)
	switch v.mode {
	case "add":
		if len(watch) >= 10 {
			d.addErr = "the priority list is full (10 locations)"
			return d, nil
		}
		for _, r := range watch { // F4: a duplicate would leave the assembler with nothing to publish
			if sameLocation(r, v.ref) { // the lists' identity: zip first (UAT 106)
				d.addErr = v.ref.Label + " is already on the watchlist"
				return d, nil
			}
		}
		watch = append(watch, v.ref)       // bottom of the watchlist (UAT 26.3)
		recent = withoutRef(recent, v.ref) // UAT 106: a promoted location leaves RECENT — never on screen twice
	case "lookup":
		recent = prependRef(recent, v.ref) // top of recent/searched (UAT 26.4)
		d.selected = len(watch)            // focus the looked-up location...
		d.showDetails, d.modalScroll = true, 0
	}
	d.showAdd, d.addQuery, d.addErr = false, "", ""
	return d, d.commitCmd(watch, recent, v.mode)
}

// commitCmd hands the new ref sets to the app hook (persist + rebuild).
func (d Dashboard) commitCmd(watch, recent []snapshot.LocationRef, what string) tea.Cmd {
	commit := d.cfg.Commit
	return func() tea.Msg {
		if commit == nil {
			return committedMsg{err: fmt.Errorf("watchlist changes are not wired in this build"), what: what}
		}
		return committedMsg{err: commit(watch, recent), what: what}
	}
}

// handleThemeKey owns the theme chooser (UAT 53): ↑↓ move, enter applies
// live (and persists via the app hook), esc closes.
func (d Dashboard) handleThemeKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	names := render.ThemeNames()
	switch key.String() {
	case "esc", "t":
		d.showTheme = false
	case "up":
		d.themeIdx = max(0, d.themeIdx-1)
	case "down":
		d.themeIdx = min(len(names)-1, d.themeIdx+1)
	case "enter":
		name := names[d.themeIdx]
		if d.cfg.SetTheme == nil {
			render.SetTheme(name) // no persistence hook in this build (tests)
			return d, nil
		}
		if err := d.cfg.SetTheme(name); err != nil {
			d.themeErr = err.Error()
		}
	}
	return d, nil
}

// openTheme toggles the theme chooser with the cursor on the active theme.
func (d Dashboard) openTheme() Dashboard {
	d.showTheme, d.showHelp, d.showDetails, d.showAlerts, d.showStatus, d.showVoice, d.themeErr = !d.showTheme, false, false, false, false, false, ""
	for i, n := range render.ThemeNames() {
		if n == render.ThemeName() {
			d.themeIdx = i
		}
	}
	return d
}

// openVoice toggles the voice chooser with the cursor on the chosen voice.
// The list is read from the hook ONCE here (UAT 85): rendering must never
// run it — on macOS it shells out to `say -v ?`.
func (d Dashboard) openVoice() Dashboard {
	d.showVoice, d.showHelp, d.showDetails, d.showAlerts, d.showStatus, d.showTheme, d.voiceErr, d.voiceNote = !d.showVoice, false, false, false, false, false, "", ""
	if !d.showVoice {
		return d
	}
	d.voiceList = d.voices()
	d.voiceIdx = 0
	for i, n := range d.voiceList {
		if n == d.radioVoice {
			d.voiceIdx = i
		}
	}
	return d
}

// voiceChip is the [V] control's label: the chosen voice, or "—" when none.
func (d Dashboard) voiceChip() string {
	if d.radioVoice == "" {
		return "—"
	}
	return render.Tint(d.radioVoice, render.Tok(render.RadioStation))
}

// voices lists the correspondent voices the app offers (empty without a hook).
func (d Dashboard) voices() []string {
	if d.cfg.Voices == nil {
		return nil
	}
	return d.cfg.Voices()
}

// handleVoiceKey owns the voice chooser (UAT 84): ↑↓ move, enter applies
// (the app persists and re-tunes), esc closes.
func (d Dashboard) handleVoiceKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	names := d.voiceList
	switch key.String() {
	case "esc", "V":
		d.showVoice = false
	case "up":
		d.voiceIdx = max(0, d.voiceIdx-1)
	case "down":
		d.voiceIdx = min(len(names)-1, d.voiceIdx+1)
	case "p":
		if len(names) > 0 && d.cfg.PreviewVoice != nil {
			preview, name := d.cfg.PreviewVoice, names[d.voiceIdx]
			d.voiceNote = "preparing " + name + "… (a first use downloads the voice, ~63 MB; loading takes a few seconds)" // UAT 119: said at once, before the deck reports
			return d, func() tea.Msg { preview(name); return nil }
		}
	case "enter":
		if len(names) == 0 {
			return d, nil
		}
		name := names[d.voiceIdx]
		d.radioVoice, d.showVoice, d.voiceErr = name, false, ""
		if set := d.cfg.SetVoice; set != nil {
			// Off the update loop (red-team 0.9.0 C-5): the hook saves the
			// config and hands the running broadcast over — disk and a
			// subprocess, never on the key press.
			return d, func() tea.Msg {
				if err := set(name); err != nil {
					return voiceErrMsg{err: err}
				}
				return nil
			}
		}
	}
	return d, nil
}

// voiceErrMsg reports a failed voice change: the chooser reopens with the
// reason so the user can pick again.
type voiceErrMsg struct{ err error }

// voiceLines is the chooser body: one row per voice, the chosen one marked.
func (d Dashboard) voiceLines(o render.Opts) []string {
	lines := []string{""}
	names := d.voiceList
	if len(names) == 0 {
		lines = append(lines, "  No voices found on this system.")
	}
	for i, n := range names {
		ptr, mark := "  ", "  "
		if i == d.voiceIdx {
			ptr = render.Tint("› ", render.Tok(render.FocusPointer))
		}
		if n == d.radioVoice {
			mark = render.Tint("✔ ", render.Tok(render.ProviderOK))
		}
		lines = append(lines, "  "+ptr+mark+n)
	}
	if d.voiceErr != "" {
		lines = append(lines, "", "  ⚠ "+d.voiceErr)
	}
	if d.voiceNote != "" { // UAT 119: the wait explained — download progress, model loading
		lines = append(lines, "", "  … "+render.Tint(d.voiceNote, render.Tok(render.TextBright)))
	}
	lines = append(lines, "", "  Your correspondent for the synthesized broadcast; the choice is saved.")
	return append(lines, "", "  "+o.KeyCap("↑↓")+" Move  "+o.KeyCap("p")+" Preview  "+o.KeyCap("enter")+" Select Voice  "+o.KeyCap("esc")+" Cancel")
}

// themeLines is the chooser body: one row per theme, the active one marked.
func (d Dashboard) themeLines(o render.Opts) []string {
	lines := []string{""}
	for i, n := range render.ThemeNames() {
		ptr, mark := "  ", "  "
		if i == d.themeIdx {
			ptr = render.Tint("› ", render.Tok(render.FocusPointer))
		}
		if n == render.ThemeName() {
			mark = render.Tint("✔ ", render.Tok(render.ProviderOK))
		}
		lines = append(lines, "  "+ptr+mark+n)
	}
	if d.themeErr != "" {
		lines = append(lines, "", "  ⚠ "+d.themeErr)
	}
	lines = append(lines, "", "  Themes apply live; add your own as ~/.config/watchpost/themes/<name>.json")
	return append(lines, "", "  "+o.KeyCap("↑↓")+" Select  "+o.KeyCap("enter")+" Apply  "+o.KeyCap("esc")+" Close")
}

// handleRemoveKey owns the confirmation modal (UAT 26.2).
func (d Dashboard) handleRemoveKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		d.showRemove = false
		watch, recent := refsOf(d.snap), refsOf(d.recent)
		if d.selected >= len(watch) {
			return d, nil
		}
		removed := watch[d.selected]
		watch = append(watch[:d.selected:d.selected], watch[d.selected+1:]...)
		recent = prependRef(recent, removed) // top of recent (UAT 26.2)
		if d.selected >= len(watch) && d.selected > 0 {
			d.selected--
		}
		return d, d.commitCmd(watch, recent, "remove")
	case "esc":
		d.showRemove = false
	}
	return d, nil
}

// refsOf rebuilds the pipeline ref set from a snapshot's locations.
func refsOf(sn *snapshot.Snapshot) []snapshot.LocationRef {
	if sn == nil {
		return nil
	}
	refs := make([]snapshot.LocationRef, 0, len(sn.Locations))
	for _, l := range sn.Locations {
		refs = append(refs, refOf(l))
	}
	return refs
}

// refOf is a location's request identity (the one builder — three callers).
func refOf(l snapshot.Location) snapshot.LocationRef {
	return snapshot.LocationRef{Label: l.Label, Tag: l.Tag, Zip: l.Zip, Lat: l.Lat, Lon: l.Lon, TZ: l.TZ}
}

// recentCap is the RECENT / SEARCHED list size (UAT 48: 50 most-recent).
const recentCap = 50

// withoutRef drops ref (by location key) from refs, order kept — the
// RECENT / SEARCHED list when a location is promoted to the watchlist
// (UAT 106): later entries move up one, the list shrinks by one.
func withoutRef(refs []snapshot.LocationRef, ref snapshot.LocationRef) []snapshot.LocationRef {
	out := make([]snapshot.LocationRef, 0, len(refs))
	for _, r := range refs {
		if !sameLocation(r, ref) {
			out = append(out, r)
		}
	}
	return out
}

// sameLocation: by zip when either side has one (the identity the lists
// dedupe on), else by location key.
func sameLocation(a, b snapshot.LocationRef) bool {
	if a.Zip != "" || b.Zip != "" {
		return a.Zip == b.Zip
	}
	return snapshot.Key(a) == snapshot.Key(b)
}

// prependRef puts ref at the head, deduped by zip, capped at recentCap.
func prependRef(refs []snapshot.LocationRef, ref snapshot.LocationRef) []snapshot.LocationRef {
	out := []snapshot.LocationRef{ref}
	for _, r := range refs {
		if r.Zip == ref.Zip || len(out) == recentCap {
			continue
		}
		out = append(out, r)
	}
	return out
}

// removeLines is the confirmation modal body (UAT 26.2).
func (d Dashboard) removeLines(o render.Opts) []string {
	label := "this location"
	if sel := d.selectedLocation(); sel != nil {
		label = sel.Label
	}
	return []string{
		"",
		"  Remove " + label + " from the watchlist?",
		"",
		"  It will move to the top of the RECENT / SEARCHED list.",
		"",
		"  " + o.KeyCap("enter") + " Confirm   " + o.KeyCap("esc") + " Cancel",
	}
}

// addLines is the add-location modal body (UAT 16.3). The live type-ahead
// results need a search hook wired from app (modes cannot import domains -
// import lint); the hook lands with the M-V3 flow.
func (d Dashboard) addLines(o render.Opts) []string {
	lines := []string{""}
	if d.addMode == "add" && d.watchlistFull() {
		// UAT 26.3: cap note leads the modal when the watchlist is full.
		lines = append(lines, "  Only 10 locations are allowed in the priority list", "  for performance reasons.", "")
	}
	lines = append(lines, "  Search: "+d.addQuery+"▌", "")
	if d.addErr != "" {
		lines = append(lines, "  ⚠ "+d.addErr, "")
	}
	lines = append(lines, "  Type a city name or ZIP code.", "")
	verb := "Add"
	enabled := d.addMode != "add" || !d.watchlistFull()
	if d.addMode == "lookup" {
		verb = "Lookup"
	}
	return append(lines, "  "+o.KeyCapIf("enter", enabled)+" "+verb+"   "+o.KeyCap("esc")+" Cancel")
}

// handleNav routes selection/paging actions (split from handleKey, P10-04).
// The focus index spans BOTH tables (UAT 4.4): 0..numPriority-1 walks the
// priority rows, then the recent rows, auto-scrolling the recent window.
func (d Dashboard) handleNav(act term.Action) Dashboard {
	if d.showHelp || d.showDetails || d.showAlerts || d.showStatus || d.showAbout {
		return d.handleModalNav(act)
	}
	switch act {
	case "nav-up":
		if d.selected > 0 {
			d.selected--
			d.alertIdx = 0
		}
	case "nav-down":
		if d.selected < d.numPriority()+d.numRecent()-1 {
			d.selected++
			d.alertIdx = 0
		}
	case "alert-prev":
		if d.alertIdx > 0 {
			d.alertIdx--
		}
	case "alert-next":
		if loc := d.selectedLocation(); loc != nil && d.alertIdx < len(loc.Alerts)-1 {
			d.alertIdx++
		}
	}
	return d.syncRecentView()
}

// severityLevel orders NWS severities most-dangerous-first (UAT 16.2).
func severityLevel(sev string) int {
	switch strings.ToLower(sev) {
	case "extreme":
		return 0
	case "severe":
		return 1
	case "moderate":
		return 2
	case "minor":
		return 3
	}
	return 4
}

// sortAlerts orders every location's alerts most severe first — warnings
// outrank advisories within a tier — so index 0 (the module's default page,
// the name tint, the details view) is always the worst active alert. The
// snapshot is this consumer's own published copy; sorting in place is safe.
func sortAlerts(sn *snapshot.Snapshot) {
	if sn == nil {
		return
	}
	rank := func(a snapshot.Alert) int {
		r := severityLevel(a.Severity) * 2
		if !render.AlertIsWarning(a.Event, a.Severity) {
			r++ // advisory sorts after a warning of the same tier
		}
		return r
	}
	for i := range sn.Locations {
		sort.SliceStable(sn.Locations[i].Alerts, func(x, y int) bool {
			return rank(sn.Locations[i].Alerts[x]) < rank(sn.Locations[i].Alerts[y])
		})
	}
}

// handleModalNav owns navigation while a modal floats (split from
// handleNav, P10-04): up/down scroll the window (UAT 10.4); in the [A]
// modal, left/right page alerts without an esc round-trip (UAT 23.1).
func (d Dashboard) handleModalNav(act term.Action) Dashboard {
	switch act {
	case "nav-up":
		d.modalScroll = max(0, d.modalScroll-1)
	case "nav-down":
		d.modalScroll = min(d.modalScroll+1, max(0, len(d.modalLines())-d.modalMax()))
	case "alert-prev":
		if d.showAlerts && d.alertIdx > 0 {
			d.alertIdx--
			d.modalScroll = 0
		}
	case "alert-next":
		if loc := d.selectedLocation(); d.showAlerts && loc != nil && d.alertIdx < len(loc.Alerts)-1 {
			d.alertIdx++
			d.modalScroll = 0
		}
	}
	return d
}

func (d Dashboard) numPriority() int {
	if d.snap == nil {
		return 0
	}
	return len(d.snap.Locations)
}

func (d Dashboard) numRecent() int {
	if d.recent == nil {
		return 0
	}
	return len(d.recent.Locations)
}

// selectedLocation resolves the focus index across both tables.
func (d Dashboard) selectedLocation() *snapshot.Location {
	np := d.numPriority()
	if d.selected < np {
		return &d.snap.Locations[d.selected]
	}
	if i := d.selected - np; i < d.numRecent() {
		return &d.recent.Locations[i]
	}
	return nil
}

// syncRecentView keeps the focused recent row inside the visible window.
func (d Dashboard) syncRecentView() Dashboard {
	np := d.numPriority()
	if d.selected < np {
		return d
	}
	idx := d.selected - np
	window := d.windowSize(d.opts())
	if idx < d.recentOff {
		d.recentOff = idx
	}
	if idx >= d.recentOff+window {
		d.recentOff = idx - window + 1
	}
	return d
}

// View implements tea.Model — a transcription of dashboard-mock-rev2-125col
// (feedback-mock-fidelity: the mock IS the spec). The viewport is terminal-
// width aware: 4-col padding all around (UAT-2C), content resizing per
// UAT-2D/E is delegated to the render seam.
func (d Dashboard) View() tea.View {
	o := d.opts()
	var b strings.Builder
	b.WriteString("\n\n") // top padding: 2 blank lines (UAT 10.3, was 3 per UAT-3.1)
	b.WriteString(d.header(o) + "\n")
	b.WriteString("\n")                                                                     // UAT-3.2: blank line between header and alert section
	b.WriteString(d.body(o))                                                                // UAT 57: no footer - every control lives where it acts
	content := render.TintDefault(indent(strings.TrimRight(b.String(), "\n"), viewPadLeft)) // UAT 4.10: base grey; no stray trailing row (UAT 58)
	if d.showHelp {
		// UAT 8.3: help floats over the dashboard (lipgloss compositing).
		content = render.Overlay(content, d.helpModal(o), d.width)
	}
	if d.showDetails {
		content = render.Overlay(content, d.detailsModal(o), d.width) // UAT 10.6
	}
	if d.showAdd {
		title := "Add Location"
		if d.addMode == "lookup" {
			title = "Lookup Location" // UAT 26.4
		}
		content = render.Overlay(content, d.floatModal(o, d.modalWidth(), title, d.addLines(o)), d.width)
	}
	if d.showRemove {
		fg, _ := render.ModalTone(d.darkBG)
		content = render.Overlay(content, d.floatModalToned(o, d.modalWidth(), "Remove Location",
			d.removeLines(o), fg, render.Tok(render.ConfirmBG)), d.width) // UAT 26.2
	}
	if d.showAlerts {
		content = render.Overlay(content, d.alertDetailsModal(o), d.width) // UAT 22
	}
	if d.showStatus {
		content = render.Overlay(content, d.floatModal(o, d.modalWidth(), "API Status", d.statusLines()), d.width) // UAT 24.2
	}
	if d.showAbout {
		content = render.Overlay(content, d.floatModal(o, d.modalWidth(), "", d.aboutLines()), d.width) // UAT 68
	}
	if d.showTheme {
		content = render.Overlay(content, d.floatModal(o, d.modalWidth(), "Color Theme", d.themeLines(o)), d.width) // UAT 53
	}
	if d.showVoice {
		content = render.Overlay(content, d.floatModal(o, d.modalWidth(), "Correspondent Voice", d.voiceLines(o)), d.width) // UAT 84
	}
	if d.showSetup {
		content = render.Overlay(content, d.floatModal(o, d.modalWidth(), "Setup", d.setupLines(o)), d.width) // UAT 100
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.BackgroundColor = render.WindowBG(d.darkBG) // UAT 10.2: blue-grey window
	return v
}

// modalMax is the modal body height budget (UAT 10.4: expand to fit tall
// terminals, window + rail on short ones).
func (d Dashboard) modalMax() int { return max(5, d.height-12) }

// modalWidth is the open modal's width — ONE source for the render sites
// and the scroll bounds.
func (d Dashboard) modalWidth() int {
	// Content-heavy modals stretch to 60% of the terminal on wide screens
	// (UAT 31.2); their base widths are the floor.
	stretch := func(base int) int { return max(base, d.width*60/100) }
	switch {
	case d.showDetails:
		return stretch(85) // location-detail-mock.txt width
	case d.showAlerts:
		return stretch(76)
	case d.showStatus:
		return stretch(68)
	case d.showAbout:
		return aboutWidth
	case d.showVoice, d.showSetup:
		return 68 // the four chip controls fit on one line (UAT 86); the FIRMS address fits (UAT 100)
	case d.showAdd, d.showRemove, d.showTheme:
		return 56
	}
	return 56 // help
}

// modalLines is the open modal's full body, wrapped exactly as the
// component renders it — scroll bounds always match what is on screen.
func (d Dashboard) modalLines() []string {
	var raw []string
	switch {
	case d.showDetails:
		raw = d.detailLines()
	case d.showAlerts:
		raw = d.alertDetailLines()
	case d.showStatus:
		raw = d.statusLines()
	case d.showAbout:
		raw = d.aboutLines()
	default:
		raw = d.helpLines(d.opts())
	}
	o := d.opts()
	return d.wrapModal(raw, min(o.Width, d.modalWidth()))
}

// wrapModal wraps a modal body for a panel of width w exactly as the
// component will draw it (UAT 68): to the full content width (w-4) when
// everything fits without the scroll rail, else to the rail budget (w-7).
// Single owner — the renderer and the scroll bounds both use it.
func (d Dashboard) wrapModal(lines []string, w int) []string {
	if full := render.WrapLines(lines, w-4); len(full) <= d.modalMax() {
		return full
	}
	return render.WrapLines(lines, w-7)
}

// Detail view (location-detail-mock.txt): labeled section rows with a
// divider column, 10-day forecast, alert blocks with the mock's bullet
// rules. MARITIME renders only when marine data exists — the marine
// provider (NWS coastal-waters / Open-Meteo Marine) is queued work.
const detailLabelW = 10

// detailPrefixW is the width of a detail row's section prefix
// ("{LABEL:10} │ "): the single owner every row-budget derives from.
const detailPrefixW = detailLabelW + 3

// detailRow renders one "{LABEL} │ {content}" line (label right-aligned;
// empty label for continuations). No lead (UAT 65): the section label
// column starts flush with the modal's header label; the freed cells are
// spacing on the right.
func detailRow(label, content string) string {
	// UAT 30: 1-col breathing room each side of the divider (was 3) - the
	// reclaimed width goes to the right gutter beside the scroll rail.
	return fmt.Sprintf("%*s │ %s", detailLabelW, label, content)
}

func (d Dashboard) detailLines() []string {
	loc := d.selectedLocation()
	if loc == nil {
		return []string{"No location selected."}
	}
	o := d.opts()
	// Rows must fit INSIDE the modal wrap budget (width-7): WrapLines
	// collapses interior spacing on over-wide lines, which would tear the
	// divider column. 15 = the detailRow chrome left of the content.
	cw := min(o.Width, d.modalWidth()) - 7 - detailPrefixW
	lines := []string{""}
	lines = append(lines, d.currentlyRows(o, loc)...)
	lines = append(lines, detailRow("", ""))
	lines = append(lines, d.todayRows(o, loc, cw)...)
	lines = append(lines, detailRow("", ""))
	lines = append(lines, d.forecastRows(o, loc)...)
	if loc.Marine != nil {
		lines = append(lines, detailRow("", ""))
		lines = append(lines, maritimeRows(o, loc.Marine, locTZ(loc), time.Now())...) // coastal locations only (UAT 29)
	}
	lines = append(lines, detailRow("", ""))
	lines = append(lines, fireRows(o, loc, time.Now(), d.fireBoldMW())...) // B5: fire is another alert kind
	lines = append(lines, alertBlocks(loc, min(o.Width, d.modalWidth())-11)...)
	// UAT 101: one consolidated chip row; + / − Watchlist enabled by membership.
	controls := o.KeyCap("↑↓") + " Scroll  " + o.KeyCap("esc") + " Close  " +
		o.KeyCapIf("ctrl+a", d.canAddFocused()) + " + Watchlist  " +
		o.KeyCapIf("shift+del", d.canRemoveFocused()) + " − Watchlist"
	return append(lines, "", controls)
}

// fireRows is the FIRE section of the detail modal (B5, HUM LEAD 2026-08-25:
// "fire is another alert type and can be a section in the location
// detail"): the hotspots inside the ring — nearest first, each with its
// bearing, distance, strength, satellite and age — the named incidents
// with acres and containment, and the fire-weather alert when one is
// active. Always present, so "none nearby" is said, never implied.
func fireRows(o render.Opts, loc *snapshot.Location, now time.Time, boldMW float64) []string {
	fs := loc.Fire
	if fs.AsOf.IsZero() { // no fire feed has answered yet (cold launch, feeds down): never "none" (red-team B5 P3)
		return []string{detailRow("FIRE", gridRow("Hotspots", "fire feed not yet available", ""))}
	}
	head := "none within the fire ring"
	if n := len(fs.Hotspots); n > 0 {
		head = fmt.Sprintf("%s hotspot%s nearby", hotspotCount(n), plural(n))
	}
	out := []string{detailRow("FIRE", gridRow("Hotspots", render.Tint(head, fireTone(len(fs.Hotspots) > 0)), ""))}
	out = append(out, hotspotRows(o, loc, fs.Hotspots, now, boldMW)...)
	out = append(out, incidentRows(o, fs.Incidents)...)
	for _, a := range loc.Alerts {
		if ev := strings.ToLower(a.Event); strings.Contains(ev, "red flag") || strings.Contains(ev, "fire weather") {
			out = append(out, detailRow("", gridRow("Fire Wx", render.Tint(render.Plain(a.Event), render.Tok(render.AlertDanger)), "")))
			break
		}
	}
	return out
}

// hotspotCount words the count; at the cap it is "300+" (snapshot.MaxHotspots).
func hotspotCount(n int) string {
	if n >= snapshot.MaxHotspots {
		return fmt.Sprintf("%d+", snapshot.MaxHotspots)
	}
	return strconv.Itoa(n)
}

// fireSeparators are the glyphs the FIRE rows join with, per glyph set.
func fireSeparators(o render.Opts) (dot, more string) {
	if o.ASCII {
		return " - ", "..."
	}
	return " · ", "…"
}

// hotspotRows: up to three hotspots nearest first, then "… and N more".
func hotspotRows(o render.Opts, loc *snapshot.Location, hs []snapshot.Hotspot, now time.Time, boldMW float64) []string {
	dot, more := fireSeparators(o)
	var out []string
	for i, h := range hs {
		if i == 3 {
			out = append(out, detailRow("", gridRow("", fmt.Sprintf("%s and %d more", more, len(hs)-3), "")))
			break
		}
		brg := geo.BearingDeg(loc.Lat, loc.Lon, h.Lat, h.Lon)
		where := "  " + fireGlyph(o) + " " + strings.TrimSpace(o.Distance(h.DistanceKm)) + " " + compass(&brg) + " " // the trailing space keeps a gap when the label fills the column (red-team B5 U6)
		strength := "n/a MW"                                                                                         // an unmeasured point (HMS GOES often) — short, so the age column never collides (U1)
		if h.FRPMW != nil {
			strength = fmt.Sprintf("%.0f MW", *h.FRPMW)
			if *h.FRPMW >= boldMW {
				strength = render.Tint(strength, "1;"+render.Tok(render.FireMark))
			}
		}
		if sat := render.Plain(h.Source.ModelOrStation); sat != "" {
			strength += dot + sat
		}
		age := "age n/a"
		if !h.DetectedAt.IsZero() { // a point without a time is not "2562047h" old (U2)
			age = fixedAgeTrim(now.Sub(h.DetectedAt))
		}
		out = append(out, detailRow("", gridRow(where, strength, age)))
	}
	return out
}

// incidentRows: up to three named incidents, largest first.
func incidentRows(o render.Opts, ins []snapshot.Incident) []string {
	dot, _ := fireSeparators(o)
	var out []string
	for i, in := range ins {
		if i == 3 {
			break
		}
		facts := strings.TrimSpace(o.Distance(in.Source.DistanceKm))
		if in.Acres != nil {
			facts += dot + thousands(*in.Acres) + " ac"
		}
		contained := ""
		if in.PercentContained != nil {
			contained = fmt.Sprintf("%.0f%% contained", *in.PercentContained)
		}
		out = append(out, detailRow("", gridRow("  "+ellipsize(render.Plain(in.Name), 11, o.ASCII)+" ", facts, contained)))
	}
	return out
}

// fireCount is the row badge's number (UAT 110): the named incidents
// nearby when there are any, else 1 for unnamed hotspots, 0 for no fire.
func fireCount(fs snapshot.FireState) int {
	switch {
	case len(fs.Incidents) > 0:
		return len(fs.Incidents)
	case len(fs.Hotspots) > 0:
		return 1
	}
	return 0
}

// fireHot reports whether any hotspot reads emphasized.
func fireHot(hs []snapshot.Hotspot, boldMW float64) bool {
	for _, h := range hs {
		if h.FRPMW != nil && *h.FRPMW >= boldMW {
			return true
		}
	}
	return false
}

// fireTone: the count reads in the fire colour when there is fire.
func fireTone(on bool) string {
	if on {
		return render.Tok(render.FireMark)
	}
	return render.Tok(render.TextBase)
}

// ellipsize cuts a name to n cells with a visible ellipsis (U5: a silent
// cut hid that "Cottonwood Creek Complex" was cut at all).
func ellipsize(s string, n int, ascii bool) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if ascii {
		return string(r[:max(0, n-3)]) + "..."
	}
	return string(r[:n-1]) + "…"
}

// fireGlyph is the fire mark for the current glyph set (▲, or ^ under --ascii).
func fireGlyph(o render.Opts) string {
	if o.ASCII {
		return "^"
	}
	return "▲"
}

// thousands groups an acreage ("12,915").
func thousands(v float64) string {
	s := fmt.Sprintf("%.0f", v)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// fireBoldMW is the emphasis threshold for the FIRE rows (Config, default 50).
func (d Dashboard) fireBoldMW() float64 {
	if d.cfg.FireBoldMW > 0 {
		return d.cfg.FireBoldMW
	}
	return 50
}

// Content column grid (UAT 32): labels at 0, primary values at colVal
// (the FORECAST condition column), secondary values at forecastHiLoCol so
// CURRENTLY / TODAY / FORECAST / MARITIME share two vertical scan lines.
const colVal = 14

// gridRow places a label, primary value, and optional secondary value on
// the grid (values never overrun: a long primary pushes the secondary).
func gridRow(label, primary, secondary string) string {
	line := render.PadTo(label, colVal) + primary
	if secondary != "" {
		line = render.PadTo(line, forecastHiLoCol) + secondary
	}
	return line
}

// currentlyRows: condition + temp/trend; feels-like + delta; humidity
// aligned to the HIGH/LOW column.
func (d Dashboard) currentlyRows(o render.Opts, loc *snapshot.Location) []string {
	h := loc.Harmonized
	temp := render.Tint(strings.TrimSpace(o.Temp(h.Temp)), render.Tok(render.TextBright)) + o.TrendGlyph(trend(*loc))
	out := []string{detailRow("CURRENTLY", gridRow(prettyCond(h.Condition), temp, ""))}
	feels, hum := "", ""
	if h.Feels != nil && h.Temp != nil {
		feels = fmt.Sprintf("%s   (%+.0fºF)", strings.TrimSpace(o.Temp(h.Feels)), (*h.Feels-*h.Temp)*9/5)
	}
	if h.HumidityPct != nil {
		hum = fmt.Sprintf("Humidity  :  %.0f%%", *h.HumidityPct)
	}
	if feels != "" || hum != "" {
		label := "Feels Like"
		if feels == "" {
			label = ""
		}
		out = append(out, detailRow("", gridRow(label, feels, hum)))
	}
	// UAT 60.2: the observing station and its distance live here at every
	// width — the table's WX STN / DIST columns surface them only when there
	// is room; drilling in one level always reaches them.
	if st := h.Source.ModelOrStation; st != "" {
		dist := ""
		if d := strings.TrimSpace(o.Distance(h.Source.DistanceKm)); d != "" {
			dist = "Distance  :  " + d
		}
		out = append(out, detailRow("", gridRow("Station   :", st, dist)))
	}
	return out
}

// todayRows: today's condition + HIGH/LOW, sunrise/sunset in local time.
func (d Dashboard) todayRows(o render.Opts, loc *snapshot.Location, cw int) []string {
	if len(loc.Daily) == 0 {
		return []string{detailRow("TODAY", o.LoadingDots())}
	}
	_ = cw
	day := loc.Daily[0]
	out := []string{detailRow("TODAY", gridRow(prettyCond(day.Condition), "", hiLo(o, day)))}
	tz := time.Local
	if z, err := zones.Location(loc.TZ); err == nil {
		tz = z
	}
	if !day.Sunrise.IsZero() {
		out = append(out, detailRow("", gridRow("Sunrise:", day.Sunrise.In(tz).Format("1504")+"  Local Time", "")))
	}
	if !day.Sunset.IsZero() {
		out = append(out, detailRow("", gridRow("Sunset :", day.Sunset.In(tz).Format("1504")+"  Local Time", "")))
	}
	return out
}

// forecastHiLoCol is the content column where every HIGH/LOW pair starts
// (date 10 + 4 + cond 13 + 1 + "(nnn%)" 6 + 3): TODAY aligns to it so the
// pairs scan as one column (UAT 28.1).
const forecastHiLoCol = 37

// hiLo renders "HIGH  98ºF /  98ºF LOW" with fixed 5-cell temps so 2- and
// 3-digit values stay aligned (UAT 28.2).
func hiLo(o render.Opts, day snapshot.Daily) string {
	return fmt.Sprintf("HIGH %5s / %5s LOW", o.Temp(day.TempMax), o.Temp(day.TempMin))
}

// forecastRows: up to 10 upcoming days with precip probability.
func (d Dashboard) forecastRows(o render.Opts, loc *snapshot.Location) []string {
	out := []string{}
	label := "FORECAST"
	for i, day := range loc.Daily {
		if i == 0 {
			continue // today has its own section
		}
		if i > 10 {
			break
		}
		pp := " --%"
		if day.PrecipProb != nil {
			pp = fmt.Sprintf("%3.0f%%", *day.PrecipProb)
		}
		date := day.Date
		if t, err := time.Parse("2006-01-02", day.Date); err == nil {
			date = t.Format("01/02/2006")
		}
		row := fmt.Sprintf("%s    %-13s (%s)   ", date, truncateTo(displayCond(day.Condition), 13), pp) + hiLo(o, day)
		out = append(out, detailRow(label, row))
		label = ""
	}
	if len(out) == 0 {
		out = append(out, detailRow("FORECAST", o.LoadingDots()))
	}
	return out
}

// locTZ resolves a location's zone for local-time rows (local fallback).
func locTZ(loc *snapshot.Location) *time.Location {
	if z, err := zones.Location(loc.TZ); err == nil && loc.TZ != "" {
		return z
	}
	return time.Local
}

// MARITIME grid (UAT 63/64/66 mock): labels on the modal's shared value
// column (col 14, like CURRENTLY/TODAY), a first sub-column of 8
// (direction / trend / time / phase), a fixed 4-cell number + unit, and
// the provenance notes in ONE column 2 cells past the section's widest
// value (UAT 66/67 — scannable, yet never further right than the data
// needs, so the section never pushes toward the scroll rail). Every row
// fits the details modal's 78-cell wrap budget with its section prefix.
//
//	Observed      39m 22s ago
//	Conditions    Slight Chop
//	Water Temp    75ºF             (buoy 46224, 11 mi)
//	Swell         SSW      3.0 ft  (period 14 s)
//	Tide          Rising   3.7 ft  (La Jolla, 24 mi)
//	Next High     19:40    5.7 ft
//	Next Low      02:49   -0.1 ft
//	Currents      Flood    1.4 kt  (Slack 16:05)
const (
	marLabelW  = colVal
	marFirstW  = 8
	marNoteGap = 2                                                                 // UAT 67: 2 cells past the widest value (Los Angeles-length names)
	marNumW    = 7                                                                 // "%4.1f ft"
	marNoteMax = 78 - detailPrefixW - marLabelW - marFirstW - marNumW - marNoteGap // 32: wrap budget at the 85-col modal floor after the widest value
)

// marineCell is one MARITIME row before layout: label, value, note.
type marineCell struct{ label, value, note string }

// marineRow collects one row for layoutMarine.
func marineRow(label, value, note string) marineCell { return marineCell{label, value, note} }

// layoutMarine lays the rows on the grid: notes share one column,
// marNoteGap cells past the widest value in the section (UAT 67).
func layoutMarine(cells []marineCell) []string {
	widest := 0
	for _, c := range cells {
		widest = max(widest, render.Width(c.value))
	}
	noteCol := marLabelW + widest + marNoteGap
	out := make([]string, 0, len(cells))
	for _, c := range cells {
		line := render.PadTo(c.label, marLabelW) + c.value
		if c.note != "" {
			line = render.PadTo(line, noteCol) + c.note
		}
		out = append(out, line)
	}
	return out
}

// marinePair is the two-part value: first sub-column + fixed-width number.
func marinePair(first, num string) string { return render.PadTo(first, marFirstW) + num }

// maritimeRows renders the coastal-waters section in the mock's scan order
// (UAT 29/32/61/63): observation age, sea state, water temperature, swells,
// then tides and currents.
func maritimeRows(o render.Opts, m *snapshot.Marine, tz *time.Location, now time.Time) []string {
	rows := []marineCell{}
	if !m.ObservedAt.IsZero() && m.Buoy != "" {
		rows = append(rows, marineRow("Observed", fixedAgeTrim(now.Sub(m.ObservedAt))+" ago", ""))
	}
	if m.WaveHeight != nil {
		rows = append(rows, marineRow("Conditions", seaState(*m.WaveHeight), ""))
	}
	if m.WaterTemp != nil {
		rows = append(rows, marineRow("Water Temp", strings.TrimSpace(o.Temp(m.WaterTemp)), buoyNote(o, m)))
	}
	rows = append(rows, swellRows(o, m)...)
	rows = append(rows, tideRows(o, m, tz, now)...)
	laid := layoutMarine(rows)
	out := make([]string, 0, len(laid))
	for i, r := range laid {
		label := ""
		if i == 0 {
			label = "MARITIME"
		}
		out = append(out, detailRow(label, r))
	}
	return out
}

// buoyNote is the "(buoy id, distance)" provenance note.
func buoyNote(o render.Opts, m *snapshot.Marine) string {
	if m.Buoy == "" {
		return ""
	}
	note := "(buoy " + m.Buoy
	if d := strings.TrimSpace(o.Distance(m.BuoyDistanceKM)); d != "" {
		note += ", " + d // display units, one formatter (UAT 60.2)
	}
	return note + ")"
}

// swellRows: primary/secondary swell with direction + period, wind waves,
// and the buoy wind.
func swellRows(o render.Opts, m *snapshot.Marine) []marineCell {
	var rows []marineCell
	if h := firstOf(m.SwellHeight, m.WaveHeight); h != nil {
		rows = append(rows, marineRow("Swell", marinePair(compass(m.SwellDirDeg), o.TideHeight(h)), period(m.WavePeriod)))
	}
	if m.SecondarySwellHeight != nil {
		rows = append(rows, marineRow("Swell 2", marinePair(compass(m.SecondarySwellDirDeg), o.TideHeight(m.SecondarySwellHeight)), period(m.SecondaryPeriod)))
	}
	if m.WindWaveHeight != nil {
		rows = append(rows, marineRow("Wind Waves", marinePair("", o.TideHeight(m.WindWaveHeight)), ""))
	}
	if m.WindSpeed != nil {
		gust := ""
		if m.WindGust != nil {
			gust = "(gusts " + o.Wind(m.WindGust) + ")"
		}
		rows = append(rows, marineRow("Buoy Wind", o.Wind(m.WindSpeed), gust))
	}
	return rows
}

// tideRows renders tides and currents (UAT 61/63, NOAA CO-OPS): trend from
// the next predicted event, one row per next high / low, local hh:mm.
func tideRows(o render.Opts, m *snapshot.Marine, tz *time.Location, now time.Time) []marineCell {
	var rows []marineCell
	nh, nl := nextTide(m.Tides, "H", now), nextTide(m.Tides, "L", now)
	if m.TideStation != "" || m.TideLevel != nil || nh != nil || nl != nil {
		val := tideTrend(nh, nl)
		if m.TideLevel != nil {
			val = marinePair(val, o.TideHeight(m.TideLevel))
		}
		rows = append(rows, marineRow("Tide", val, stationNote(o, m.TideStation, m.TideStationKM)))
	}
	if nh != nil {
		rows = append(rows, marineRow("Next High", tideEvent(o, nh, tz), ""))
	}
	if nl != nil {
		rows = append(rows, marineRow("Next Low", tideEvent(o, nl, tz), ""))
	}
	if row, ok := currentRow(o, m.Currents, tz, now); ok {
		rows = append(rows, row)
	}
	return rows
}

// tideEvent: "19:40    5.7 ft" (local time, fixed-width height).
func tideEvent(o render.Opts, e *snapshot.TideEvent, tz *time.Location) string {
	return marinePair(e.Time.In(tz).Format("15:04"), o.TideHeight(&e.Height))
}

// tideTrend reads the direction from whichever extreme comes next.
func tideTrend(nh, nl *snapshot.TideEvent) string {
	switch {
	case nh != nil && (nl == nil || nh.Time.Before(nl.Time)):
		return "Rising"
	case nl != nil:
		return "Falling"
	}
	return ""
}

// nextTide is the first event of a type after now (events are time-ordered).
func nextTide(events []snapshot.TideEvent, typ string, now time.Time) *snapshot.TideEvent {
	for i := range events {
		if events[i].Type == typ && events[i].Time.After(now) {
			return &events[i]
		}
	}
	return nil
}

// currentRow: the phase in force (last predicted extreme before now, with
// its max speed) and the next predicted event as the note.
func currentRow(o render.Opts, events []snapshot.CurrentEvent, tz *time.Location, now time.Time) (marineCell, bool) {
	var cur, next *snapshot.CurrentEvent
	for i := range events {
		if !events[i].Time.After(now) {
			cur = &events[i]
		} else if next == nil {
			next = &events[i]
		}
	}
	if cur == nil && next == nil {
		return marineCell{}, false
	}
	val, note := "Slack", ""
	if cur != nil && cur.Type != "slack" {
		val = marinePair(titleWord(cur.Type), o.Knots(&cur.Speed))
	}
	if next != nil {
		note = "(" + titleWord(next.Type) + " " + next.Time.In(tz).Format("15:04") + ")"
	}
	return marineRow("Currents", val, note), true
}

// stationNote is the "(name, distance)" provenance note for the tide row;
// the name is cut at its parenthetical qualifier and capped so the note
// fits the grid's note budget.
func stationNote(o render.Opts, name string, km *float64) string {
	if name == "" {
		return ""
	}
	if i := strings.Index(name, " ("); i > 0 {
		name = name[:i]
	}
	dist := strings.TrimSpace(o.Distance(km))
	room := marNoteMax - 2 // parentheses
	if dist != "" {
		room -= len(dist) + 2
	}
	note := "(" + truncateTo(name, room)
	if dist != "" {
		note += ", " + dist
	}
	return note + ")"
}

func titleWord(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// period formats a wave period for the secondary column.
func period(s *float64) string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("(period %.0f s)", *s)
}

// firstOf returns the first non-nil value.
func firstOf(vals ...*float64) *float64 {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}

// fixedAgeTrim is fixedAge without the alignment padding.
func fixedAgeTrim(d time.Duration) string { return strings.TrimSpace(fixedAge(d)) }

// compass maps degrees true to a 16-point heading.
func compass(deg *float64) string {
	if deg == nil {
		return "--"
	}
	pts := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	return pts[int((*deg+11.25)/22.5)%16]
}

// seaState words a significant wave height (Douglas sea-state bands).
func seaState(m float64) string {
	switch {
	case m < 0.1:
		return "Calm (glassy)"
	case m < 0.5:
		return "Smooth"
	case m < 1.25:
		return "Slight Chop"
	case m < 2.5:
		return "Moderate Chop"
	case m < 4:
		return "Rough"
	}
	return "Very Rough"
}

// alertBlocks renders each alert in full with the mock's bullet rules,
// separated by dividers.
func alertBlocks(loc *snapshot.Location, w int) []string {
	if len(loc.Alerts) == 0 {
		return nil
	}
	// UAT 33.1/65/66: the divider and the alert text run from the text edge
	// (flush with the section labels) to 3 cells before the scroll rail:
	// w is the modal width less 11; the rail sits at w+5, so the right
	// edge is w+2.
	textW := w + 2
	divider := strings.Repeat("─", textW)
	lines := []string{}
	for _, a := range loc.Alerts {
		tone := modalAlertTone(a)
		lines = append(lines, "", divider, "", render.TintRaw("⚠ "+strings.ToUpper(render.Plain(a.Event)), "1;"+tone)) // bold title (UAT 28.5)
		for _, l := range formatAlertBody(render.Plain(a.Description), textW) {
			lines = append(lines, render.TintRaw(l, tone))
		}
	}
	return lines
}

// modalAlertTone: advisory #ACAE7D / warning #BE5454 text in modals
// (UAT 28.3/28.4) - theme tokens on the raw-SGR path.
func modalAlertTone(a snapshot.Alert) string {
	if render.AlertIsWarning(a.Event, a.Severity) {
		return render.Tok(render.AlertModalWarnFG)
	}
	return render.Tok(render.AlertModalAdvFG)
}

// formatAlertBody applies the mock's bullet rules to NWS alert prose:
// prose indents 2 and "*"-bullets 4 cols from the text edge (the flush
// section-label column, UAT 65); single-line bullets stack tight; a
// multi-line bullet gets one blank line above and below. textW is the
// full line budget — every line, indent included, ends by it (UAT 66).
func formatAlertBody(desc string, textW int) []string {
	out := []string{}
	prevMulti := false
	for _, item := range splitAlertItems(desc) {
		if !item.bullet {
			for _, l := range render.WrapText(item.text, textW-2) {
				out = append(out, "  "+l)
			}
			prevMulti = false
			continue
		}
		wrapped := render.WrapText(item.text, textW-6)
		multi := len(wrapped) > 1
		if (multi || prevMulti) && len(out) > 0 {
			out = append(out, "")
		}
		for j, l := range wrapped {
			prefix := "    - "
			if j > 0 {
				prefix = "      "
			}
			out = append(out, prefix+l)
		}
		prevMulti = multi
	}
	return out
}

// alertItem is one prose paragraph or bullet from an NWS description.
type alertItem struct {
	text   string
	bullet bool
}

// splitAlertItems parses NWS description text: paragraphs separated by
// blank lines; "* "-prefixed items are bullets.
func splitAlertItems(desc string) []alertItem {
	items := []alertItem{}
	for _, para := range strings.Split(desc, "\n\n") {
		para = strings.TrimSpace(strings.ReplaceAll(para, "\n", " "))
		if para == "" {
			continue
		}
		for i, piece := range strings.Split(para, "* ") {
			piece = strings.TrimSpace(piece)
			if piece == "" {
				continue
			}
			items = append(items, alertItem{text: piece, bullet: i > 0 || strings.HasPrefix(para, "* ")})
		}
	}
	return items
}

// prettyCond renders a condition code as title-cased prose ("Partly Cloudy").
func prettyCond(c string) string {
	words := strings.Fields(strings.ToLower(strings.ReplaceAll(c, "_", " ")))
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	if len(words) == 0 {
		return "--"
	}
	return strings.Join(words, " ")
}

// truncateTo hard-limits a plain string to n cells.
func truncateTo(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// alertDetailsModal floats ONE alert at a time (UAT 23.2): the tile tint
// follows the FOCUSED alert's class (yellow advisory / red warning) as you
// page, keeping attention on each individual statement.
func (d Dashboard) alertDetailsModal(o render.Opts) string {
	sel := d.selectedLocation()
	fg, _ := render.ModalTone(d.darkBG)
	if sel == nil || len(sel.Alerts) == 0 {
		return d.floatModalToned(o, d.modalWidth(), "ALERTS", d.alertDetailLines(), fg, render.Tok(render.AlertModalAdvBG))
	}
	a := sel.Alerts[d.alertIdx%len(sel.Alerts)]
	bg := render.Tok(render.AlertModalAdvBG)
	if render.AlertIsWarning(a.Event, a.Severity) {
		bg = render.Tok(render.AlertModalWarnBG)
	}
	title := fmt.Sprintf("ALERT %d / %d · %s", d.alertIdx+1, len(sel.Alerts), sel.Label)
	return d.floatModalToned(o, d.modalWidth(), title, d.alertDetailLines(), fg, bg)
}

// alertDetailLines renders the FOCUSED alert's full record plus the paging
// controls (chips mute per direction - UAT 23.1).
func (d Dashboard) alertDetailLines() []string {
	sel := d.selectedLocation()
	o := d.opts()
	if sel == nil || len(sel.Alerts) == 0 {
		return []string{"", "  No active alerts for this location.", "", "  " + o.KeyCap("esc") + " Close"}
	}
	n := len(sel.Alerts)
	idx := d.alertIdx % n
	wrapW := min(o.Width, d.modalWidth()) - 9 // breathing room beside the scroll rail (UAT 23.3)
	lines := []string{""}
	lines = append(lines, alertRecordLines(o, idx, sel.Alerts[idx], wrapW, alertClock(sel))...)
	controls := o.KeyCapIf("←", idx > 0) + " Previous  " + o.KeyCapIf("→", idx < n-1) + " Next   " +
		o.KeyCap("esc") + " Close   " + o.KeyCap("↑↓") + " Scroll"
	return append(lines, "  "+controls)
}

// dataAsOf is the header's "Last Updated" time (red-team 0.9.0 F7): the
// newest successful provider fetch — not the publish time, which keeps
// ticking while every row is stale last-good data offline.
func dataAsOf(sn *snapshot.Snapshot) time.Time {
	if sn == nil {
		return time.Time{}
	}
	at := time.Time{}
	for _, p := range sn.Providers {
		if p.FetchedAt.After(at) {
			at = p.FetchedAt
		}
	}
	if at.IsZero() {
		return sn.GeneratedAt
	}
	return at
}

// alertClock is the zone an alert's times are shown in: the location's
// own (F17 — a Miami alert is not on Pacific time), else the machine's.
func alertClock(loc *snapshot.Location) *time.Location {
	if loc != nil && loc.TZ != "" {
		if z, err := zones.Location(loc.TZ); err == nil {
			return z
		}
	}
	return time.Local
}

// alertRecordLines formats one alert's full record for the modal.
func alertRecordLines(o render.Opts, i int, a snapshot.Alert, wrapW int, in *time.Location) []string {
	tone := modalAlertTone(a)                      // UAT 28.3/28.4 modal text tones
	_ = i                                          // paging lives in the modal title now (UAT 23.2)
	head := strings.ToUpper(render.Plain(a.Event)) // provider text never addresses the terminal (S-F6)
	meta := fmt.Sprintf("[%s · %s · %s]", a.Severity, a.Urgency, a.Certainty)
	out := []string{"  " + render.TintRaw(head, "1;"+tone) + "  " + meta} // bold title (UAT 28.5)
	start, end := a.Effective, a.Expires
	if a.Onset != nil {
		start = *a.Onset
	}
	if a.Ends != nil {
		end = *a.Ends
	}
	timing := "  Starts " + start.In(in).Format("Mon 01/02 3:04 PM") // the location's clock (F17)
	if end.After(start) {
		timing += "   Ends " + end.In(in).Format("Mon 01/02 3:04 PM") +
			fmt.Sprintf("   (~%s)", end.Sub(start).Round(time.Hour))
	}
	out = append(out, timing)
	if a.AreaDesc != "" {
		out = append(out, wrapPrefixed(o, "Area: "+a.AreaDesc, wrapW)...)
	}
	if a.Description != "" {
		out = append(out, "")
		out = append(out, wrapPrefixed(o, render.Plain(a.Description), wrapW)...)
	}
	if a.Instruction != "" {
		out = append(out, "")
		out = append(out, wrapPrefixed(o, "Instructions: "+a.Instruction, wrapW)...)
	}
	// UAT 55: body text (everything below the toned title) reads white for
	// contrast - advisories and alerts earn it.
	for i := 1; i < len(out); i++ {
		if out[i] != "" {
			out[i] = render.Tint(out[i], render.Tok(render.AlertModalText))
		}
	}
	return append(out, "")
}

// wrapPrefixed wraps prose to the modal width with the 2-col text inset.
func wrapPrefixed(_ render.Opts, text string, w int) []string {
	wrapped := render.WrapText(text, w)
	for i, l := range wrapped {
		wrapped[i] = "  " + l
	}
	return wrapped
}

// statusLines is the [S] API diagnostics body (UAT 24.2/31.1): aligned
// provider rows with a fixed-width freshness age, pipeline snapshot ages,
// and warnings AGGREGATED by code+provider (count, locations hit, latest
// message) so a busy pipeline reads as a handful of issues, not a flood.
// True request latency needs httpx instrumentation - queued with B5.
func (d Dashboard) statusLines() []string {
	o := d.opts()
	lines := []string{"", "  PROVIDERS"}
	if d.snap == nil {
		lines = append(lines, "    awaiting first snapshot...")
	}
	for _, p := range providersOf(d.snap) {
		age := "    n/a"
		if !p.FetchedAt.IsZero() {
			age = fixedAge(time.Since(p.FetchedAt))
		}
		lines = append(lines, fmt.Sprintf("    %s %-9s %-4s fetched %s ago",
			render.PadTo(o.HealthGlyph(strings.ToUpper(p.ID), p.Status), 13), p.Role, p.Status, age))
	}
	lines = append(lines, "", "  PIPELINES")
	lines = append(lines, pipelineLine("Priority", d.snap), pipelineLine("Recent  ", d.recent))
	lines = append(lines, "", "  ISSUES")
	lines = append(lines, aggregateWarnings(d.snap, d.recent)...)
	lines = append(lines, "", "  Request latency metrics land with the multi-provider work (B5).")
	return append(lines, "", "  "+o.KeyCap("esc")+" Close   "+o.KeyCap("↑↓")+" Scroll")
}

// fixedAge formats a duration in a 7-cell right-aligned slot so ages line
// up: "59m 59s", " 1m  5s", "    55s", " 2h 05m".
func fixedAge(dur time.Duration) string {
	dur = dur.Round(time.Second)
	h, m, sec := int(dur.Hours()), int(dur.Minutes())%60, int(dur.Seconds())%60
	switch {
	case h > 0:
		return fmt.Sprintf("%2dh %02dm", h, m)
	case m > 0:
		return fmt.Sprintf("%2dm %2ds", m, sec)
	}
	return fmt.Sprintf("    %2ds", sec)
}

// issue is one aggregated warning class.
type issue struct {
	code, provider, latest string
	count, locations       int
}

// aggregateWarnings folds both snapshots' warnings into issue classes and
// renders them (UAT 31.1). Split fold/render for P10-04.
func aggregateWarnings(snaps ...*snapshot.Snapshot) []string {
	issues := foldWarnings(snaps)
	if len(issues) == 0 {
		return []string{"    none"}
	}
	return renderIssues(issues)
}

// foldWarnings groups by code+provider: count, distinct locations, latest
// message; provider errors sort first, then by count.
func foldWarnings(snaps []*snapshot.Snapshot) []*issue {
	byKey := map[string]*issue{}
	seenLoc := map[string]map[string]bool{}
	for _, sn := range snaps {
		if sn == nil {
			continue
		}
		for _, w := range sn.Warnings {
			k := w.Code + "|" + w.Provider
			it := byKey[k]
			if it == nil {
				it = &issue{code: w.Code, provider: w.Provider}
				byKey[k] = it
				seenLoc[k] = map[string]bool{}
			}
			it.count++
			it.latest = w.Message
			if w.Location != "" && !seenLoc[k][w.Location] {
				seenLoc[k][w.Location] = true
				it.locations++
			}
		}
	}
	issues := make([]*issue, 0, len(byKey))
	for _, it := range byKey {
		issues = append(issues, it)
	}
	sort.Slice(issues, func(i, j int) bool {
		pi, pj := issues[i].code == snapshot.WarnProviderError, issues[j].code == snapshot.WarnProviderError
		if pi != pj {
			return pi
		}
		return issues[i].count > issues[j].count
	})
	return issues
}

// renderIssues formats issue rows, capped at 8 with an overflow note.
func renderIssues(issues []*issue) []string {
	out := []string{}
	for i, it := range issues {
		if i == 8 {
			return append(out, fmt.Sprintf("    ... and %d more issue classes", len(issues)-8))
		}
		glyph := "⚠"
		if it.code == snapshot.WarnProviderError {
			glyph = "✘"
		}
		prov := it.provider
		if prov == "" {
			prov = "-"
		}
		scope := fmt.Sprintf("×%d", it.count)
		if it.locations > 0 {
			scope += fmt.Sprintf(" (%d locations)", it.locations)
		}
		out = append(out, fmt.Sprintf("    %s %-11s %-22s %s", glyph, strings.ToUpper(prov), it.code, scope))
		if it.latest != "" {
			out = append(out, "        latest: "+it.latest)
		}
	}
	return out
}

// providersOf guards the nil snapshot for the status view.
func providersOf(sn *snapshot.Snapshot) []snapshot.ProviderStatus {
	if sn == nil {
		return nil
	}
	return sn.Providers
}

// pipelineLine summarizes one pipeline's snapshot for the status view.
func pipelineLine(name string, sn *snapshot.Snapshot) string {
	if sn == nil {
		return "    " + name + "  awaiting first snapshot..."
	}
	return fmt.Sprintf("    %s  snapshot %s · %d locations", name,
		dataAsOf(sn).Local().Format("15:04:05"), len(sn.Locations))
}

// displayCond mirrors// displayCond mirrors the table's condition vocabulary for the modal.
func displayCond(c string) string {
	c = strings.ToUpper(strings.ReplaceAll(c, "_", " "))
	c = strings.ReplaceAll(c, "PARTLY ", "P.")
	return strings.ReplaceAll(c, "MOSTLY ", "M.")
}

// detailsModal renders the floating detail view (location-detail-mock.txt):
// title carries the location + a right-aligned Updated stamp; the body
// lengthens/shortens with terminal height via the ScrollPanel budget.
func (d Dashboard) detailsModal(o render.Opts) string {
	loc := d.selectedLocation()
	title := "Location"
	if loc != nil {
		title = loc.Label + " " + loc.Zip
	}
	if d.snap != nil {
		stamp := "Updated: " + dataAsOf(d.snap).Local().Format("01/02/2006 15:04:05 MST")
		fill := min(o.Width, d.modalWidth()) - 10 - len([]rune(title)) - len([]rune(stamp))
		if fill > 1 {
			title = title + " " + strings.Repeat("─", fill) + " " + stamp
		}
	}
	return d.floatModal(o, d.modalWidth(), title, d.detailLines())
}

// floatModal is THE floating-window renderer (help, forecast details, and
// the coming About/setup modals): scrollable panel body, blue-grey tile
// background per terminal mode (UAT 12.4), base-grey text.
func (d Dashboard) floatModal(o render.Opts, width int, title string, lines []string) string {
	fg, bg := render.ModalTone(d.darkBG)
	return d.floatModalToned(o, width, title, lines, fg, bg)
}

// floatModalToned renders a floating window with an explicit tile tone —
// the [A] alert modal carries its severity tint (UAT 22). Body lines WRAP
// to the modal width here, in the component (UAT 25: truncation is not a
// bug any caller can reintroduce).
func (d Dashboard) floatModalToned(o render.Opts, width int, title string, lines []string, fg, bg string) string {
	o.Width = min(o.Width, width)
	lines = d.wrapModal(lines, o.Width)
	// Block alone arms BOTH the base-grey text and the tile background and
	// re-arms them after every inner reset. Running TintDefault first was
	// the session-12 color bug: it consumed the resets Block re-arms on, so
	// every styled span (chips, temp tints) dropped the tile background for
	// the rest of its line.
	return o.Block(o.ScrollPanel(title, lines, d.modalScroll, d.modalMax()), fg, bg)
}

// opts sizes the layout: content width = terminal - 2x2-col padding, minus
// a 2-col gutter reserved for the recent rail. The NAME fill column makes
// the tables span this width exactly (UAT 11.1), so every section stays
// flush and aligned at any terminal size.
func (d Dashboard) opts() render.Opts {
	raw := max(d.width-viewPadLeft-viewPadRight, 40)
	return render.Opts{Width: raw - 2, Units: d.units, Frame: d.frame}
}

// windowSize is the height-aware recent viewport (UAT 8.2): the 10-row
// window shrinks on short terminals so the Showing line and footer stay
// visible. chrome = 35 fixed lines (top pad 3, header 2, blank 1, radio
// 9+1, alert 7+1, priority header 2, table headers 2, recent title 3,
// recent headers 2, showing 1, blank 1) + priority rows + footer lines.
func (d Dashboard) windowSize(o render.Opts) int {
	// Module heights follow the theme's bg visibility (UAT 19.1): 9 fixed
	// chrome lines (top pad 2, header 2, blank, module gaps 2, priority
	// headers 2 - the band/showing rows are counted with the window) + the
	// control row's lines + each module + priority rows. No footer (UAT 57);
	// measured so the content ends exactly 2 rows above the bottom (UAT 58).
	chrome := 9 + strings.Count(d.controlRow(o), "\n") + 1 + d.radioHeight(o) + 1 + d.alertHeight() + 1 +
		d.numPriority() // UAT 58: measured - content fills to exactly height-2 (2-row bottom inset)
	// UAT 46.1: the window EXPANDS to fill tall terminals - the footer stays
	// on screen and a 2-row bottom inset mirrors the top padding.
	return max(recentWindow, d.height-chrome-2)
}

// header is the two-line masthead (UAT 102 mock, redesigned for narrow
// terminals and any number of APIs):
//
//	W A T C H P O S T  <version>     Updated: <stamp>     API: ✔8 ⚠0 ✘0 /  8  [S] Status
//	[s] Setup  [a] About  [t] Theme  [?] Help  [q] Quit
//
// The stamp is centred in the gap between the title and the API summary;
// the total reserves two columns for growth. Narrow terminals shorten the
// stamp (time only, then no label, then gone — it lives in [S]) before the
// row could overflow; the header never exceeds the width.
func (d Dashboard) header(o render.Opts) string {
	title := render.TitleGradient("W A T C H P O S T") + "  v" + d.cfg.Version // UAT 4.9 gradient; UAT 41: no pipe
	api := d.apiSummary(o) + "  " + o.KeyCap("S") + " Status"                  // UAT 24.2
	stamps := []string{"awaiting first data..."}
	if d.snap != nil {
		at := dataAsOf(d.snap).Local()
		stamps = []string{"Updated: " + at.Format("01/02/2006 15:04:05 MST"), "Updated: " + at.Format("15:04:05"), at.Format("15:04:05")}
	}
	if render.Width(title)+render.Width(api)+2 > o.Width { // very narrow: the chip alone, then no label
		api = d.apiSummary(o) + " " + o.KeyCap("S")
	}
	if render.Width(title)+render.Width(api)+2 > o.Width {
		api = strings.TrimPrefix(d.apiSummary(o), "API: ") + " " + o.KeyCap("S")
	}
	gap := o.Width - render.Width(title) - render.Width(api)
	line1 := render.PadBetween(title, api, o.Width)
	for _, stamp := range stamps {
		if w := render.Width(stamp); w+2 <= gap {
			pad := (gap - w) / 2
			line1 = title + strings.Repeat(" ", pad) + stamp + strings.Repeat(" ", gap-pad-w) + api
			break
		}
	}
	line2 := o.KeyCap("s") + " Setup  " + o.KeyCap("a") + " About  " + o.KeyCap("t") + " Theme  " + o.KeyCap("?") + " Help  " + o.KeyCap("q") + " Quit" // UAT 56/57/100/102
	return line1 + "\n" + line2
}

// apiSummary counts the providers by health: ✔ ok · ⚠ degraded but has
// served (stale) · ✘ degraded and never served (down); providers that are
// not a source right now (off) count in none of them, and a provider that
// has not answered yet counts in the total only (the three never sum past
// it; a shortfall means "still loading"). The total is the active set,
// padded to two columns (UAT 102: "reserve 2 col for growth").
func (d Dashboard) apiSummary(o render.Opts) string {
	ok, stale, down, total := 0, 0, 0, 0
	for _, p := range providersOf(d.snap) {
		switch {
		case p.Status == snapshot.ProviderOff:
			continue
		case p.Status == snapshot.ProviderOK && p.FetchedAt.IsZero():
			// registered, not yet answered (the first frames): in the total, not yet ✔ (REVIEW C5)
		case p.Status == snapshot.ProviderOK:
			ok++
		case p.FetchedAt.IsZero():
			down++
		default:
			stale++
		}
		total++
	}
	gOK, gStale, gDown := "✔", "⚠", "✘"
	if o.ASCII {
		gOK, gStale, gDown = "OK", "!!", "XX"
	}
	return fmt.Sprintf("API: %s %s%d %s%d /%3d",
		render.Tint(gOK+strconv.Itoa(ok), render.Tok(render.ProviderOK)),
		render.Tint(gStale, render.Tok(render.AlertLabel)), stale,
		render.Tint(gDown, render.Tok(render.ProviderDown)), down, total)
}

func (d Dashboard) body(o render.Opts) string {
	var b strings.Builder
	b.WriteString(d.radioPanel(o) + "\n\n") // UAT 8.1: radio module first
	b.WriteString(d.alertArea(o) + "\n\n")  // then the alert area (blanks per UAT 6.1/6.2)

	days := d.sharedExtDays()
	// UAT 26/43: controls live where they act - ABOVE the watchlist's group
	// labels now (right-aligned to the table edge).
	b.WriteString(d.controlRow(o) + "\n")
	if d.snap == nil || len(d.snap.Locations) == 0 {
		b.WriteString(emptyState(o.TableRowLen(days), watchlistEmpty, "") + "\n") // UAT 104: the empty state stands where the table will
	} else {
		rows := make([]render.LocationRow, 0, len(d.snap.Locations))
		for i, loc := range d.snap.Locations {
			rows = append(rows, d.row(i, loc, d.selected < d.numPriority() && i == d.selected))
		}
		b.WriteString(o.LocationTable(rows, days) + "\n")
	}
	b.WriteString(d.recentSection(o, days))
	return b.String()
}

// controlRow is the watchlist's control line (UAT 56): [enter] Details,
// [ctrl+a] Add, [shift+del] Remove, [l] Lookup with [↑↓] Navigate right-
// aligned when the row fits; on narrow terminals it smart-wraps instead
// (same WrapSegments as the footer) so no row exceeds the width.
func (d Dashboard) controlRow(o render.Opts) string {
	segs := []string{
		o.KeyCap("enter") + " Details", // UAT 57
		o.KeyCap("ctrl+a") + " Add",
		o.KeyCapIf("shift+del", d.selected < d.numPriority() && d.numPriority() > 0) + " Remove",
		o.KeyCap("l") + " Lookup",
	}
	nav := o.KeyCap("↑↓") + " Navigate"
	line := strings.Join(segs, "   ")
	if render.Width(line)+render.Width(nav)+2 <= o.Width {
		return render.PadBetween(line, nav, o.Width)
	}
	return render.WrapSegments(append(segs, nav), o.Width, "   ")
}

// sharedExtDays pins ONE extended-forecast column count for the priority
// and recent tables (UAT session 3.1: if extended columns display in
// priority, recent must immediately match).
func (d Dashboard) sharedExtDays() int {
	days := 0
	for _, sn := range []*snapshot.Snapshot{d.snap, d.recent} {
		if sn == nil {
			continue
		}
		for _, loc := range sn.Locations {
			days = max(days, min(5, max(0, len(loc.Daily)-2)))
		}
	}
	return days
}

// recentSection renders the RECENT/SEARCHED table — seeded with the top-25
// US cities until real search history lands (UAT session 2A) — windowed to
// recentWindow rows with the mock's ▲│▼ scroll rail.
func (d Dashboard) recentSection(o render.Opts, days int) string {
	var b strings.Builder
	// UAT 43/45: a full-width section band in the group-label style, no
	// blank lines around it; the rail's ▲ rides the band now that the
	// recent table shows rows only.
	rail := o.TableRowLen(days) + 2 // UAT 9.2: one blank col between the last cell and the rail
	band := render.Band("R E C E N T   /   S E A R C H E D", "R E C E N T", o.TableRowLen(days), render.GroupSectionBG)
	b.WriteString(render.PadTo(band, rail-1) + "▲\n")
	total, base := 0, 0
	if d.recent != nil {
		total = len(d.recent.Locations)
	}
	if d.snap != nil {
		base = len(d.snap.Locations)
	}
	if total == 0 {
		b.WriteString(emptyState(o.TableRowLen(days), recentEmpty, render.PadTo("", rail-o.TableRowLen(days)-1)+"▼") + "\n") // UAT 104 fallback; the rail closes on its last row
		return b.String()
	}
	window := d.windowSize(o)
	lo := min(d.recentOff, max(0, total-window))
	hi := min(total, lo+window)
	rows := make([]render.LocationRow, 0, hi-lo)
	for i := lo; i < hi; i++ {
		r := d.row(i, d.recent.Locations[i], base+i == d.selected) // focus spans both tables (UAT 4.4)
		r.Index = base + i + 1                                     // numbering continues after the priority rows (mock: 004.)
		rows = append(rows, r)
	}
	// UAT 44.1/45: the band connects the tables - the recent table renders
	// rows only (both header rows dropped; the watchlist's headers apply).
	table := strings.SplitN(o.LocationTable(rows, days), "\n", 3)[2]
	b.WriteString(railify(table, rail, lo, total, window) + "\n")
	showing := fmt.Sprintf("Showing %d-%d of %d locations", lo+1, hi, total)
	b.WriteString(render.PadBetween("", showing+"  ▼", rail) + "\n")
	return b.String()
}

// Empty states (UAT 104): three to five rows — a blank, the message centred
// on the table span and wrapped for narrow terminals, a blank — standing
// where the table will once a location is added or searched.
const (
	watchlistEmpty = "Run 's' Setup, 'l'ookup a location, or 'ctrl+a' a searched location to add to your Watchlist"
	recentEmpty    = "NO RECENT LOCATION SEARCHED or DATA-SEEDING FAILED"
	emptyWrapAt    = 64 // the mock's two-line break on wide terminals
)

// emptyState renders the block; tail rides the last row (the rail's ▼).
func emptyState(span int, text, tail string) string {
	lines := render.WrapText(text, max(10, min(span-4, emptyWrapAt)))
	out := []string{""}
	for _, l := range lines {
		pad := max(0, (span-render.Width(l))/2)
		out = append(out, strings.Repeat(" ", pad)+l)
	}
	out = append(out, "")
	if tail != "" {
		out[len(out)-1] = render.PadTo("", span) + tail
	}
	return strings.Join(out, "\n")
}

// railify appends the scroll rail: ▲ on the column-header line, │ on each
// row line with a █ thumb tracking the scroll position (UAT 11.3), ▼ on the
// Showing line.
func railify(table string, width, lo, total, window int) string {
	lines := strings.Split(table, "\n") // rows only (UAT 45); ▲ rides the band, ▼ the Showing line
	thumb := 0
	if maxLo := total - window; maxLo > 0 && window > 1 {
		thumb = lo * (window - 1) / maxLo
	}
	for i := range lines {
		glyph := "│"
		if i == thumb {
			glyph = "█"
		}
		// PadTo, not PadBetween: full-width rows must not push the rail right
		// (UAT 6.6 off-by-one vs the Showing line's ▼).
		lines[i] = render.PadTo(lines[i], width-1) + glyph
	}
	return strings.Join(lines, "\n")
}

// row converts a snapshot location to a table row.
func (d Dashboard) row(i int, loc snapshot.Location, selected bool) render.LocationRow {
	row := render.LocationRow{
		Index: i + 1, Name: loc.Label, Tag: loc.Tag, Zip: loc.Zip,
		Station:    loc.Harmonized.Source.ModelOrStation,                                                           // WX STN / DIST (UAT 60)
		Playing:    d.radioPlaying && d.radioKey == snapshot.Key(snapshot.LocationRef{Lat: loc.Lat, Lon: loc.Lon}), // UAT 80
		Repeat:     d.radioRepeat != RepeatOff,                                                                     // UAT 83/93: ∞ when the row will come round again
		StationKM:  loc.Harmonized.Source.DistanceKm,
		Conditions: loc.Harmonized.Condition, // display mapping in the seam (P.CLOUDY etc)
		Now:        loc.Harmonized.Temp,
		Trend:      trend(loc),
		HasAlert:   len(loc.Alerts) > 0,
		AlertCount: len(loc.Alerts),                            // ⚠ badge (UAT 20.2)
		Fire:       fireCount(loc.Fire),                        // B5 / UAT 110: n◆
		FireHot:    fireHot(loc.Fire.Hotspots, d.fireBoldMW()), // B5
		Selected:   selected,
		// UAT 18.2: shimmer until this location's data lands (obs or daily
		// still pending); post-load nils stay honest "n/a".
		Loading: loc.Harmonized.Source.Provider == "" || len(loc.Daily) == 0,
	}
	for _, al := range loc.Alerts {
		if render.AlertIsWarning(al.Event, al.Severity) {
			row.WarnAlert = true // warning-grade outranks advisory (UAT 14.1)
			break
		}
	}
	if len(loc.Daily) > 0 {
		row.Hi, row.Lo = loc.Daily[0].TempMax, loc.Daily[0].TempMin
	}
	if len(loc.Daily) > 1 {
		row.TomorrowConditions = loc.Daily[1].Condition
		row.TomorrowHi, row.TomorrowLo = loc.Daily[1].TempMax, loc.Daily[1].TempMin
	}
	for _, day := range loc.Daily[min(2, len(loc.Daily)):] {
		if len(row.Extended) == 5 {
			break
		}
		row.Extended = append(row.Extended, render.DayCell{Date: mmdd(day.Date), Hi: day.TempMax, Lo: day.TempMin})
	}
	return row
}

// mmdd shortens an ISO date to the mock's mm/dd column header.
func mmdd(iso string) string {
	if len(iso) < 10 {
		return iso
	}
	return iso[5:7] + "/" + iso[8:10]
}

// alertArea renders the alert module at a FIXED height (UAT 5.2: the area
// stays reserved-but-blank when the focused location has no alert, so the
// UI never jumps). Borderless background block per UAT 5.4: warning-grade
// red-on-red-tint, advisory-grade yellow-on-yellow-tint; the focused
// location's name sits in the title line (UAT 5.3).
// Alert module content: title + blank + THREE body lines (UAT 15.2a) +
// blank + pager = 7; the seam adds bg padding when the tone is visible
// (UAT 19.1 global inset policy).
const alertContentLines = 7
const alertBodyLines = 3

func (d Dashboard) alertArea(o render.Opts) string {
	sel := d.selectedLocation()
	if sel == nil || len(sel.Alerts) == 0 {
		// Reserve the module's CURRENT height (tone visibility included) so
		// the layout never jumps when alerts appear (UAT 5.2 / 19.1).
		return strings.Repeat("\n", d.alertHeight()-1)
	}
	a := sel.Alerts[d.alertIdx%max(1, len(sel.Alerts))]
	fg, bg := render.AlertBlockTone(a.Event, a.Severity)
	mw := o.ModuleInnerWidth(bg)
	if d.compact() {
		return o.Module([]string{d.alertCompactLine(o, sel, a, mw)}, fg, bg) // UAT 34
	}
	title := "⚠ " + strings.ToUpper(a.Event) + " · " + sel.Label
	// UAT 15.2: wrap, never truncate; fixed 3-line body area (15.2a); the
	// MESSAGE alone carries a 4-col inset each side (UAT 19.1).
	body := render.WrapText(fmt.Sprintf("[%s] %s", a.Severity, a.Headline), mw-8)
	if len(body) > alertBodyLines {
		body = body[:alertBodyLines]
	}
	for i := len(body); i < alertBodyLines; i++ { // counter form (P10-02)
		body = append(body, "")
	}
	lines := []string{title, ""}
	for _, l := range body {
		lines = append(lines, "    "+l)
	}
	// UAT 21.1: paging chips mute when the press would do nothing.
	controls := o.KeyCap("A") + " Alert Details   " +
		o.KeyCapIf("←", d.alertIdx > 0) + " Previous  " +
		o.KeyCapIf("→", d.alertIdx < len(sel.Alerts)-1) + " Next"
	lines = append(lines, "", render.PadBetween(fmt.Sprintf("%02d / %02d Alerts", d.alertIdx+1, len(sel.Alerts)), controls, mw))
	return o.Module(lines, fg, bg)
}

// alertCompactLine is the one-row alert module (UAT 34):
// "nn/nn  ⚠ EVENT · Label    [sev] headline...   [A] Alert Details [←] Previous [→] Next".
func (d Dashboard) alertCompactLine(o render.Opts, sel *snapshot.Location, a snapshot.Alert, mw int) string {
	n := len(sel.Alerts)
	head := fmt.Sprintf("%02d/%02d  ⚠ %s · %s", d.alertIdx%n+1, n, strings.ToUpper(a.Event), sel.Label)
	controls := o.KeyCap("A") + " Alert Details  " + o.KeyCapIf("←", d.alertIdx > 0) + " Previous  " + o.KeyCapIf("→", d.alertIdx < n-1) + " Next"
	body := fmt.Sprintf("[%s] %s", a.Severity, a.Headline)
	room := mw - render.Width(head) - render.Width(controls) - 7 // 4-col gap + 3-col gap
	if room < 8 {
		body = ""
	} else if render.Width(body) > room {
		body = truncateTo(body, room-3) + "..."
	}
	// Progressive degrade (UAT 35): body, then chip labels, then the title
	// itself - the row never exceeds the module width.
	if render.Width(head)+render.Width(controls)+3 > mw {
		controls = o.KeyCap("A") + " " + o.KeyCapIf("←", d.alertIdx > 0) + " " + o.KeyCapIf("→", d.alertIdx < n-1)
	}
	if over := render.Width(head) + render.Width(controls) + 3 - mw; over > 0 {
		head = truncateTo(head, max(8, render.Width(head)-over-1)) + "…"
	}
	return render.PadBetween(head+"    "+body, controls, mw)
}

// alertHeight is the alert module's rendered height under the active theme
// (the advisory tone carries the module's resting visibility) and layout
// mode (UAT 34: one line when compact).
func (d Dashboard) alertHeight() int {
	_, bg := render.AlertBlockTone("", "minor")
	if d.compact() {
		return render.ModuleHeight(1, bg)
	}
	return render.ModuleHeight(alertContentLines, bg)
}

// radioHeight is the radio module's rendered height for the active mode.
func (d Dashboard) radioHeight(o render.Opts) int {
	_, bg := render.RadioBlockTone()
	return render.ModuleHeight(len(d.radioLines(o, d.compact() || d.radioMin)), bg)
}

// tableBreakpoint is the total table rows (favourites + recent window)
// the full layout must keep before the modules minimize (UAT 49).
const tableBreakpoint = 20

// compact reports whether the terminal is too short for the full modules
// (UAT 34/47): the footer controls must always stay on screen; the alert
// module collapses to one line and the radio to two (~12 rows reclaimed)
// before the recent table is asked to give up any rows.
func (d Dashboard) compact() bool {
	_, abg := render.AlertBlockTone("", "minor")
	_, rbg := render.RadioBlockTone()
	fullRadio := render.ModuleHeight(len(d.radioLines(d.opts(), false)), rbg)
	// UAT 49: the full modules stay while the table can show at least
	// tableBreakpoint rows (favourites + recent window, any split); the
	// table grows/shrinks row-by-row above that. Only when the full layout
	// cannot deliver the breakpoint do the modules minimize.
	rows := max(recentWindow, min(d.numRecent(), tableBreakpoint-d.numPriority()))
	full := 9 + strings.Count(d.controlRow(d.opts()), "\n") + 1 + fullRadio + 1 + render.ModuleHeight(alertContentLines, abg) + 1 +
		d.numPriority() + rows + 2 // recent rows + bottom inset (UAT 58)
	return d.height < full
}

// radioPanel renders the mock's full-size player frame. STATIC LAYOUT MOCK
// until B4 wires audio (UAT-3.3: render it now to test the design; content
// marked pending). Width-bound through render.Panel (UAT-2E).
func (d Dashboard) radioPanel(o render.Opts) string {
	fg, bg := render.RadioBlockTone()
	return o.Module(d.radioLines(o, d.compact() || d.radioMin), fg, bg) // [T] Size: Min = two-row player
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
	return strings.Split(render.WrapSegments(segs, inner, "  "), "\n")
}

// indent left-pads every non-empty line (4-col viewport padding, UAT-2C).
func indent(s string, n int) string {
	pad := strings.Repeat(" ", n)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = pad + l
		}
	}
	return strings.Join(lines, "\n")
}

// trend derives the NOW arrow from the next forecast hour vs current temp
// (mock: 888ºF↗; ±0.3ºC deadband so noise doesn't flicker the arrow).
func trend(loc snapshot.Location) string {
	if loc.Harmonized.Temp == nil || len(loc.Hourly) == 0 || loc.Hourly[0].Temp == nil {
		return ""
	}
	delta := *loc.Hourly[0].Temp - *loc.Harmonized.Temp
	switch {
	case delta > 0.3:
		return "up"
	case delta < -0.3:
		return "down"
	}
	return ""
}

// helpModal renders live from the merged KeyMap so it is truthful after any
// swap (D-15 guarantee 3).
func (d Dashboard) helpModal(o render.Opts) string {
	return d.floatModal(o, d.modalWidth(), "Watchpost Help", d.helpLines(o)) // UAT 8.3/10.1/10.4
}

// helpLines renders the merged KeyMap as modal body lines (truthful after
// any swap - D-15 guarantee 3).
func (d Dashboard) helpLines(o render.Opts) []string {
	lines := make([]string, 0, len(d.keys)+4)
	for act, bind := range d.keys {
		lines = append(lines, fmt.Sprintf("%-14s - %s", strings.Join(bind.Keys, ", "), orDefault(bind.Help, string(act))))
	}
	sort.Strings(lines)
	// Row marks legend (red-team B5 U8): the glyphs beside a location, in words.
	lines = append(lines, "", "Row marks: ▶ playing   ∞ on repeat   n◆ fires nearby (bold = burning hard)   n⚠ alerts")
	return append(lines, "", "  "+o.KeyCap("esc")+" Close   "+o.KeyCap("↑↓")+" Scroll") // chips like every other modal (UAT 68.2)
}

func orDefault(s, alt string) string {
	if s == "" {
		return alt
	}
	return s
}

// About window (UAT 68/70 mock, 60 cols): title + version centred, the
// data providers and the build stack inset 3, the maker lines centred. Lines
// are composed on the mock's 58-cell interior and handed to the panel
// minus the two cells its chrome already draws, so every offset matches
// the mock exactly. Providers come from the live provider registry so a
// new data source lists itself.
const aboutWidth = 60

func (d Dashboard) aboutLines() []string {
	interior := aboutWidth - 2
	centre := func(text string) string {
		return strings.Repeat(" ", max(0, (interior-render.Width(text))/2)) + text
	}
	inset := func(text string) string { return "   " + text } // 3-cell inset (UAT 70): the NOAA line splits 3-52-3
	lines := []string{
		centre(render.TitleGradient("W A T C H P O S T")),
		centre("v " + d.cfg.Version),
		"",
		inset("Data Provided by:"),
		"",
	}
	for _, p := range d.cfg.Credits {
		lines = append(lines, inset(p))
	}
	lines = append(lines,
		"",
		inset(creditsNotice), // UAT 75
		"",
		inset("Built with:"),
		inset("GO "+strings.TrimPrefix(runtime.Version(), "go")+" | BubbleTea | LipGloss |"),
		inset("STUDS - Stylized Terminal UI Design System"),
		"",
		centre("Made with ♥ by Branden R. Thompson"),
		centre("github: branden-thompson"),
		centre("Make CLIs Great for Humans Again"),
	)
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, strings.TrimPrefix(l, "  ")) // the panel chrome draws these two cells
	}
	return out
}

// creditsNotice states the terms every listed source shares: NOAA data is
// public domain, GeoNames and Open-Meteo are CC BY 4.0 — all free to use
// with attribution (UAT 75).
const creditsNotice = "All sources free to use with attribution."

// Setup window (UAT 100, HUM LEAD 2026-08-25): "goes to the dashboard,
// immediately opens a setup modal like all the others, asks the questions;
// no → default data set, no key; [s] Setup at any time". One form, both
// questions on screen (UAT 111.3): the default location (type-ahead over
// the embedded index, or a full resolve on enter; a bare enter keeps the
// current default) and the optional NASA FIRMS key (masked; empty keeps a
// stored key, or means the keyless default set). Saving hands the answers to the
// app's Setup hook (persist) and then the watchlist to Commit (the default
// location on top, the rest kept) in ONE command, so the two writes never
// race. esc closes without saving — the dashboard simply has no rows until
// a location is chosen, and [s] reopens the window.
// setupFocus names the question the keys go to (UAT 111.3: every question
// is on screen at once; tab / shift+tab move between them).
type setupFocus int

const (
	focusLocation setupFocus = iota // 1. default location
	focusKey                        // 2. NASA FIRMS key
	setupQuestions
)

type setupState struct {
	focus  setupFocus
	query  string
	hints  []snapshot.LocationRef
	idx    int
	ref    *snapshot.LocationRef // the chosen (or kept) default
	key    string
	reveal bool
	err    string
}

// openSetup toggles the Setup window with fresh state, alone on top.
func (d Dashboard) openSetup() Dashboard {
	open := !d.showSetup
	d.showHelp, d.showDetails, d.showAdd, d.showAlerts, d.showStatus, d.showRemove, d.showTheme, d.showAbout, d.showVoice, d.modalScroll = false, false, false, false, false, false, false, false, false, 0
	d.showSetup, d.setup = open, setupState{}
	return d
}

// handleSetupKey owns keys while the Setup window is open: printable keys
// build the focused answer, so table/global bindings never fire mid-typing.
// tab / shift+tab move between the questions; enter accepts the focused
// one (and saves on the last); esc closes without saving.
func (d Dashboard) handleSetupKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		d.showSetup, d.setup = false, setupState{}
		return d, nil
	case "tab":
		d.setup.focus, d.setup.err = (d.setup.focus+1)%setupQuestions, ""
		return d, nil
	case "shift+tab":
		d.setup.focus, d.setup.err = (d.setup.focus+setupQuestions-1)%setupQuestions, ""
		return d, nil
	}
	if d.setup.focus == focusLocation {
		return d.setupLocationKey(key)
	}
	return d.setupKeyKey(key)
}

// setupLocationKey is question 1: type → hints; ↑↓ pick; enter takes the
// pick, keeps the current default when nothing was typed, or resolves the
// typed text when nothing matched offline — then moves to question 2.
func (d Dashboard) setupLocationKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		if len(d.setup.hints) > 0 {
			ref := d.setup.hints[min(d.setup.idx, len(d.setup.hints)-1)]
			d.setup.ref, d.setup.focus, d.setup.err = &ref, focusKey, ""
			return d, nil
		}
		if q := strings.TrimSpace(d.setup.query); q != "" {
			return d, d.resolveCmd(q, "setup")
		}
		if d.setup.ref != nil { // a location already chosen this visit: keep it, move on (REVIEW C3)
			d.setup.focus, d.setup.err = focusKey, ""
			return d, nil
		}
		if cur := d.currentDefault(); cur != nil { // a re-run keeps the default with a bare enter (UAT 111.2)
			d.setup.ref, d.setup.focus, d.setup.err = cur, focusKey, ""
			return d, nil
		}
		d.setup.err = "type a city or ZIP first"
	case "up":
		d.setup.idx = max(0, d.setup.idx-1)
	case "down":
		d.setup.idx = min(max(0, len(d.setup.hints)-1), d.setup.idx+1)
	case "backspace":
		if r := []rune(d.setup.query); len(r) > 0 {
			d.setup.query = string(r[:len(r)-1])
		}
		d = d.setupSuggest()
	default:
		if key.Text != "" {
			d.setup.query += key.Text
			d = d.setupSuggest()
		}
	}
	return d, nil
}

// setupKeyKey is question 2: the FIRMS key line — type to paste, ctrl+r
// reveals, enter saves the form (an empty key keeps a stored one, or means
// the default data set on a first run).
func (d Dashboard) setupKeyKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter":
		if d.setup.ref == nil {
			if cur := d.currentDefault(); cur != nil {
				d.setup.ref = cur
			} else {
				d.setup.focus, d.setup.err = focusLocation, "choose your default location first"
				return d, nil
			}
		}
		return d, d.setupFinishCmd(strings.TrimSpace(d.setup.key))
	case "ctrl+r":
		d.setup.reveal = !d.setup.reveal
	case "backspace":
		if r := []rune(d.setup.key); len(r) > 0 {
			d.setup.key = string(r[:len(r)-1])
		}
	default:
		if key.Text != "" {
			d.setup.key += key.Text
		}
	}
	return d, nil
}

// currentDefault is the watchlist's first location (the stored default),
// nil on a first run.
func (d Dashboard) currentDefault() *snapshot.LocationRef {
	if d.snap == nil || len(d.snap.Locations) == 0 {
		return nil
	}
	ref := refOf(d.snap.Locations[0])
	return &ref
}

// setupSuggest refreshes the hints from the app hook (embedded index only —
// never the network per keystroke).
func (d Dashboard) setupSuggest() Dashboard {
	d.setup.hints, d.setup.idx, d.setup.err = nil, 0, ""
	if q := strings.TrimSpace(d.setup.query); q != "" && d.cfg.Suggest != nil {
		d.setup.hints = d.cfg.Suggest(q, 5)
	}
	return d
}

// setupFinishCmd persists the answers then commits the watchlist with the
// default location on top — one command, two hooks, in order.
func (d Dashboard) setupFinishCmd(key string) tea.Cmd {
	if d.setup.ref == nil {
		return nil
	}
	def, setup, commit := *d.setup.ref, d.cfg.Setup, d.cfg.Commit
	watch := []snapshot.LocationRef{def}
	for _, r := range refsOf(d.snap) {
		if !sameLocation(r, def) {
			watch = append(watch, r)
		}
	}
	recent := withoutRef(refsOf(d.recent), def)
	return func() tea.Msg {
		if setup == nil {
			return committedMsg{err: fmt.Errorf("setup is not wired in this build"), what: "setup"}
		}
		if err := setup(def, key); err != nil {
			return committedMsg{err: err, what: "setup"}
		}
		if commit == nil {
			return committedMsg{what: "setup"}
		}
		return committedMsg{err: commit(watch, recent), what: "setup"}
	}
}

// setupLines is the Setup window body — one form, every question on
// screen, the focused one marked › (UAT 111.3).
func (d Dashboard) setupLines(o render.Opts) []string {
	st := d.setup
	mark := func(f setupFocus) string {
		if st.focus == f {
			return "› "
		}
		return "  "
	}
	lines := append([]string{""}, d.setupLocationLines(o, mark(focusLocation))...)
	lines = append(lines, d.setupKeyLines(mark(focusKey))...)
	action := "Next"
	if st.focus == focusKey {
		action = "Save"
	}
	// The chip row wraps by chip, inside the inset (UAT 111.4) — the same
	// WrapSegments the radio controls use, never mid-chip.
	segs := []string{o.KeyCap("tab") + " Next question", o.KeyCap("enter") + " " + action, o.KeyCap("↑↓") + " Pick", o.KeyCap("ctrl+r") + " Reveal key", o.KeyCap("esc") + " Cancel"}
	inner := min(o.Width, d.modalWidth()) - 7 - 2 // wrapModal's rail allowance, then the 2-cell inset
	lines = append(lines, "")
	for _, row := range strings.Split(render.WrapSegments(segs, inner, "   "), "\n") {
		lines = append(lines, "  "+row)
	}
	return lines
}

// setupLocationLines is question 1 of the form.
func (d Dashboard) setupLocationLines(o render.Opts, mark string) []string {
	st := d.setup
	lines := []string{"  " + mark + render.Tint("1. Your default location (city, \"City, ST\" or ZIP)", render.Tok(render.TextBright))} // questions read white (UAT 111.5)
	switch {
	case st.ref != nil:
		lines = append(lines, "       Chosen: "+render.Plain(st.ref.Label)+" ("+st.ref.Zip+")")
	case strings.TrimSpace(st.query) == "":
		if cur := d.currentDefault(); cur != nil {
			lines = append(lines, "       Current: "+render.Plain(cur.Label)+" ("+cur.Zip+") — "+o.KeyCap("enter")+" keeps it") // UAT 111.2
		}
	}
	if st.ref == nil || st.focus == focusLocation {
		lines = append(lines, "       Search: "+st.query+"▌")
		for i, h := range st.hints {
			pick := "  "
			if i == st.idx {
				pick = "› "
			}
			lines = append(lines, "       "+pick+render.Plain(h.Label)+" ("+h.Zip+")")
		}
	}
	if st.err != "" && st.focus == focusLocation {
		lines = append(lines, "       ⚠ "+st.err)
	}
	return lines
}

// setupKeyLines is question 2 of the form: the FIRMS key, with a stored
// key's tail and health when there is one (UAT 111).
func (d Dashboard) setupKeyLines(mark string) []string {
	st := d.setup
	hint := ""
	if d.cfg.FIRMSKey != nil {
		hint = d.cfg.FIRMSKey()
	}
	var lines []string
	if hint != "" { // UAT 111: a stored key is shown to be there, with how it is doing, and can be replaced
		lines = append(lines, "", "  "+mark+render.Tint("2. NASA FIRMS key: stored (…"+hint+") — ", render.Tok(render.TextBright))+d.firmsHealth(),
			"       Paste a new key to replace it — empty keeps it")
	} else {
		lines = append(lines, "", "  "+mark+render.Tint("2. NASA FIRMS key (optional — satellite fire detection)", render.Tok(render.TextBright)),
			"       Free key: firms.modaps.eosdis.nasa.gov/api/map_key",
			"       Empty = the default data set, no key")
	}
	shown := strings.Repeat("•", len([]rune(st.key)))
	if st.reveal {
		shown = st.key
	}
	lines = append(lines, "       Key: "+shown+"▌")
	if st.err != "" && st.focus == focusKey {
		lines = append(lines, "       ⚠ "+st.err)
	}
	return lines
}

// firmsHealth words the FIRMS provider's state for the Setup window (UAT
// 111): ✔ working (green), ✘ rejected (red), degraded, off, or not yet
// reported — glyph and colour together (R-12a: the glyph carries it alone).
func (d Dashboard) firmsHealth() string {
	if d.snap == nil {
		return "no report yet"
	}
	for _, w := range d.snap.Warnings {
		if w.Provider == "firms" && strings.Contains(w.Message, "rejected the MAP_KEY") {
			return render.Tint("✘ rejected — replace it", render.Tok(render.ProviderDown))
		}
	}
	for _, p := range d.snap.Providers {
		if p.ID != "firms" {
			continue
		}
		switch p.Status {
		case snapshot.ProviderOK:
			return render.Tint("✔ working", render.Tok(render.ProviderOK))
		case snapshot.ProviderOff:
			return "not active"
		}
		return render.Tint("✘ degraded (see [S] Status)", render.Tok(render.ProviderDown))
	}
	return "no report yet"
}

// applyCommitted records a commit hook error for the add modal.
func (d Dashboard) applyCommitted(v committedMsg) Dashboard {
	if v.what == "setup" { // the Setup window owns its own outcome (UAT 100)
		if v.err != nil {
			d.showSetup, d.setup.err, d.setup.focus = true, "setup failed: "+v.err.Error(), focusKey
			return d
		}
		d.showSetup, d.setup, d.selected = false, setupState{}, 0
		return d
	}
	if v.err != nil {
		// Show it (red-team 0.9.0 F10): the modal that asked is already
		// closed, so the location modal reopens with the reason instead of
		// failing silently — alone (N-7: never stacked under another modal),
		// naming the action, in add mode (a failed remove has no remove modal to return to).
		what := v.what
		if what == "" {
			what = "change"
		}
		d.addErr, d.showAdd, d.addMode = what+" failed: "+v.err.Error(), true, "add"
		d.showRemove, d.showVoice, d.showTheme = false, false, false
	}
	return d
}

// applyRadioStatus mirrors the player's status into the model (B4).
func (d Dashboard) applyRadioStatus(v RadioStatusMsg) Dashboard {
	if v.Detail != d.radioDetail {
		d.radioSince = time.Now() // a new line starts its own marquee clock (UAT 83)
	}
	d.radioState, d.radioStation, d.radioDetail, d.radioLive, d.radioSpoken = v.State, v.Station, v.Detail, v.Live, v.Spoken
	d.radioPlaying = v.State == "playing" || v.State == "connecting" || v.State == "reconnecting"
	if v.Location != "" {
		d.radioKey = v.Location // UAT 93: the ▶ row follows a Watchlist advance
	}
	if v.State != "" {
		d.radioVolume = v.Volume
	}
	return d
}

// withCmd queues a command for the caller to return with the model.
func (d Dashboard) withCmd(cmd tea.Cmd) Dashboard {
	d.pendingCmd = cmd
	return d
}

// takeCmd returns the model and any queued command, clearing it.
func (d Dashboard) takeCmd() (tea.Model, tea.Cmd) {
	cmd := d.pendingCmd
	d.pendingCmd = nil
	return d, cmd
}

// WithRadio attaches the player after construction (B4: the app builds the
// program from the model and the player needs the program).
func (d Dashboard) WithRadio(r Radio) Dashboard {
	d.cfg.Radio = r
	return d
}

// WithRadioMode sets the persisted source mode for the [m] chip (UAT 97).
func (d Dashboard) WithRadioMode(mode RadioMode) Dashboard {
	d.radioMode = mode
	return d
}

// WithSpectrum attaches the visualizer feed (UAT 92).
func (d Dashboard) WithSpectrum(feed func() []float64) Dashboard {
	d.cfg.Spectrum = feed
	return d
}

// WithVoices attaches the voice chooser hooks (UAT 84/86).
func (d Dashboard) WithVoices(list func() []string, current string, set func(string) error, preview func(string)) Dashboard {
	d.cfg.Voices, d.cfg.SetVoice, d.cfg.PreviewVoice, d.cfg.Voice, d.radioVoice = list, set, preview, current, current
	return d
}
