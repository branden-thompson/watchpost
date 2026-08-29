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
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// SnapshotMsg delivers a fresh priority snapshot (sent by app wiring).
type SnapshotMsg struct{ Snap *snapshot.Snapshot }

// RecentSnapshotMsg delivers the RECENT/SEARCHED pipeline's snapshot — a
// separate slow-cadence scheduler feeds the seeded major-city list (UAT
// session 2A) so the priority pipeline's M1 budget stays untouched.
type RecentSnapshotMsg struct{ Snap *snapshot.Snapshot }

// TickerMsg publishes the global event ticker stack (0.12.0); the app maps
// globalfeed events onto TickerItems and sends this on the ticker cadence.
type TickerMsg struct{ Items []TickerItem }

// TickerAdvanceMsg rotates the marquee to the next non-empty category lane
// (0.12.0). The ticker pipeline sends it every 90 s (HUM LEAD 2026-08-27) so
// the rotation keeps a steady wall-clock rhythm independent of the frame tick.
type TickerAdvanceMsg struct{}

// TickerBreakingMsg takes the marquee over with a single breaking event, shown
// centred in its lane colour (0.12.0). The pipeline sends one per event as it
// steps a breaking-news sequence; TickerBreakingDoneMsg ends the takeover and
// resumes normal rotation where it left off (HUM LEAD 2026-08-27).
type TickerBreakingMsg struct{ Item TickerItem }

// TickerBreakingDoneMsg ends a breaking-news takeover.
type TickerBreakingDoneMsg struct{}

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
	SeismicDays  int                                                   // 0.11.0: the [seismic] lookback window the section words ("last N days"; 0 = 7)
	Suggest      func(query string, limit int) []snapshot.LocationRef  // type-ahead hints for the Setup window (embedded index only; nil = enter resolves)
	Setup        func(def snapshot.LocationRef, firmsKey string) error // persist the default location (+ the FIRMS key when given) — the Setup window's finish (UAT 100)
	OpenSetup    bool                                                  // open the Setup window at launch (first run, no locations, or `watchpost setup`)
	FIRMSKey     func() string                                         // the stored FIRMS key's tail ("cdef"), "" when none — the Setup window shows it is there (UAT 111)
	Stats        func() Stats                                          // request/publish/dump counters for the [S] modal (quality pass Q0); nil = the rows are omitted
	ASCII        bool                                                  // --ascii: the row marks and legend in their ASCII forms (A11-10; quality pass Q3)
	TickerMuted  bool                                                  // 0.12.0: the persisted [M] state at launch
	MuteTicker   func(bool)                                            // 0.12.0: persist the [M] toggle and tell the ticker pipeline to stay silent; nil in tests
	NarrateEvent func(key string)                                      // 0.13.0: [space] in the severe window reads the focused event over the radio (UAT option B); nil = the chip mutes

	AlertRadiusMi  int       // 0.12.0: the Alert Notification Preference at launch — 0 = All (global), >0 = only alerts within N mi of the default location
	SetAlertRadius func(int) // 0.12.0: persist the radius and tell the ticker pipeline to re-scope; nil in tests
}

// Stats is what the app hands the [S] modal beyond the snapshots (quality
// pass Q0, plan §2.1): request counters merged across the app's clients,
// publish counters per pipeline, and the last diagnostic dump's outcome.
type Stats struct {
	Requests  httpx.RequestStats
	Pipelines [2]PipelineStats // [0] priority, [1] recent
	LastDump  string           // "" before the first dump; else "<ts> ok <dir>" or "<ts> failed: <reason>"
	DumpHint  string           // how to trigger a dump on this platform
}

// PipelineStats counts one pipeline's publishes and the triggers its
// coalescing window folded.
type PipelineStats struct {
	Publishes int64
	Folded    int64
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
	Short    string // the station's short form for a narrow player ("EVENT · SPS · Palomar Mountain, CA"); "" = shorten the long one
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
		"severe":        {Keys: []string{"w", "ctrl+s"}, Help: "Severe Weather / Disaster Events"},
		"details":       {Keys: []string{"enter"}, Help: "Location Details"}, // the window shows what it has; the short label keeps the Help columns within 133 cols (UAT 2026-08-28)
		"add-location":  {Keys: []string{"ctrl+a"}, Help: "Add Location"},
		"status":        {Keys: []string{"S"}, Help: "API Status"},
		"remove":        {Keys: []string{"shift+delete"}, Help: "Remove from Watchlist"},
		"lookup":        {Keys: []string{"l"}, Help: "Lookup Location"},
		"theme":         {Keys: []string{"t"}, Help: "Choose Color Theme"},
		"setup":         {Keys: []string{"s"}, Help: "Setup"},
		"radio-play":    {Keys: []string{"space"}, Help: "Play/Pause · Read Event"},
		"radio-repeat":  {Keys: []string{"r"}, Help: "Repeat: Off / One / Watchlist"},
		"radio-viz":     {Keys: []string{"v"}, Help: "Visualizer"},
		"radio-mode":    {Keys: []string{"m"}, Help: "Radio Mode: Synth / Nearest Relay"},
		"voice":         {Keys: []string{"V"}, Help: "Correspondent Voice"},
		"radio-size":    {Keys: []string{"T"}, Help: "Toggle Player Size"},
		"ticker-mute":   {Keys: []string{"M"}, Help: "Mute Severe Alerts"},
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
	recentOff    int                   // scroll offset (interaction lands with tab section nav)
	modal        modal                 // the ONE open window (quality pass Q6, L3-F15): exclusivity by construction, not by ten reset sites
	addMode      string                // "add" | "lookup" (shared search modal, UAT 26.3/26.4)
	lookupRef    *snapshot.LocationRef // the location a lookup opened Details for, until its data lands (HUM LEAD UAT 2026-08-28: the modal showed the old top RECENT row meanwhile)
	addErr       string                // resolve failure surfaced in the modal
	voiceNote    string                // the Voice chooser's progress line (UAT 119): set by VoiceNoteMsg, "" when nothing is pending
	setup        setupState
	themeIdx     int // chooser cursor
	voiceIdx     int
	voiceErr     string
	voiceList    []string // snapshot of the hook's list, taken when the chooser opens (UAT 85: never from View)
	radioVoice   string   // the chosen correspondent (chip label)
	themeErr     string
	addQuery     string               // add-location search buffer
	modalScroll  int                  // shared scroll for floating modals (UAT 10.4)
	ticker       []TickerItem         // 0.12.0: the active global alerts (grouped into lanes by Category)
	tickerCatIdx int                  // which non-empty lane is showing (rotates every 90s)
	tickerScroll int                  // tape scroll offset within the current lane
	breaking     *TickerItem          // 0.12.0: a breaking-news takeover — one event centred, overrides the tape until done
	tickerMuted  bool                 // [M]: the P3 tone/narration muted (visual state here in P2)
	darkBG       bool                 // terminal mode (bubbletea BackgroundColorMsg)
	frame        int                  // animation phase (loading shimmer, UAT 18.2b)
	radioPlaying bool                 // [space] Play|Pause (UAT 39) — true while connecting/playing (B4)
	radioState   string               // B4: last RadioStatusMsg state ("" = never tuned)
	pendingCmd   tea.Cmd              // B4: a command a key handler queued (radio hook calls run off the update loop)
	radioStation string               // B4: resolved station label
	radioShort   string               // its short form for a narrow player (UAT 2026-08-28); "" = none
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
	radioMin     bool             // [T] Size: Min renders the two-row player
	vizBands     []float64        // the visualizer's latest frame (UAT 92); nil = blank rows
	vizTicking   bool             // a vizTick is in flight — never two
	tickArmed    bool             // a shimmer tick is in flight — never two (Q3: armed only while something animates)
	now          func() time.Time // the clock the header's "ago" reads (tests pin it)
	memo         *bodyMemo        // the body memo's single slot (Q3); allocated at construction, shared by every copy of the model
	mmemo        *modalMemo       // the modal memo's single slot (0.13.0, FR-10): the open window renders once per input change
	// 0.13.0: the Severe Weather / Disaster Events window (severe.go)
	severe          SevereMsg
	severeByTab     [severeNumTabs][]int // row indices per tab, in the app's sort
	severeTab       SevereTab
	severeRow       int
	severeDetail    bool      // the record replaces the table (esc backs out)
	lastBreaking    time.Time // the last breaking-news takeover: the window opens on its category for 10 min
	lastBreakingTab SevereTab
	severeReading   string // the key of the event being read over the radio ([space]); "" = none — the ▶ mark
}

// NewDashboard builds the model, merging user key overrides with validation
// (a conflicting override is a build error, never a silent win — D-15).
func NewDashboard(cfg Config) (Dashboard, error) {
	keys, err := term.Merge(defaultKeyMap(), cfg.KeyOverrides)
	if err != nil {
		return Dashboard{}, fmt.Errorf("key bindings invalid: %w", err)
	}
	d := Dashboard{cfg: cfg, keys: keys, units: render.UnitF, width: 80, height: 24, darkBG: true, radioVolume: 55, radioVoice: cfg.Voice, tickerMuted: cfg.TickerMuted, memo: &bodyMemo{}, mmemo: &modalMemo{}, now: time.Now}
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

// tickMsg drives the loading shimmer (UAT 18.2b) and, since Q3, every
// other wall-clock element of the frame: the marquee (when the visualizer
// tick is not already redrawing), the volume blink's clearing, the [S]
// ages and the Details labels. It runs only while one of them is showing
// (plan §2.5 tick predicate: tickNeeded).
type tickMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

// tickNeeded is the predicate (PF-2, R2-23): true while a frame would
// differ from the last one without any message arriving.
func (d Dashboard) tickNeeded() bool {
	switch {
	case d.volFlash != "": // pending or just expired — the tick after expiry clears it
		return true
	case d.modal == modalStatus || d.modal == modalDetails: // [S] ages; Details "N min ago" labels and LoadingDots
		return true
	case len(d.ticker) > 0: // 0.12.0: the marquee scrolls continuously while events are active
		return true
	case d.radioPlaying && d.radioDetail != "" && !d.vizTicking && !d.radioMin && (!d.radioLive || d.radioState != "playing"):
		return true // the marquee paces itself on the wall clock (UAT 83); the viz tick redraws faster when on; LIVE RADIO and the min player have none
	}
	return d.anyLoading() // the shimmer (UAT 18.2b)
}

// armTick starts the shimmer tick when the frame needs one and none is in
// flight — the shimmer twin of armViz. Called after every Update.
func (d Dashboard) armTick(cmd tea.Cmd) (tea.Model, tea.Cmd) {
	if d.tickArmed || !d.tickNeeded() {
		return d, cmd
	}
	d.tickArmed = true
	if cmd == nil {
		return d, tick()
	}
	return d, tea.Batch(cmd, tick())
}

// vizTickMsg drives the visualizer (UAT 92): 20 frames a second, only while
// there is something to draw — the shimmer tick is far too slow for bars.
type vizTickMsg struct{}

func vizTick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg { return vizTickMsg{} })
}

// Init implements tea.Model — asks the terminal for its background color
// so the window tint tracks light/dark mode (UAT 10.2). The animation tick
// arms itself from the first message that needs it (Q3).
func (d Dashboard) Init() tea.Cmd { return tea.RequestBackgroundColor }

// Update implements tea.Model: dispatch the message, then arm the shimmer
// tick if the resulting frame animates (Q3 tick predicate).
func (d Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := d.dispatch(msg)
	next, ok := m.(Dashboard)
	if err := invariant.Check(ok, "dispatch must return the dashboard model"); err != nil {
		return m, cmd
	}
	return next.armTick(cmd)
}

// handleTicker applies one global-event-ticker message and re-arms the frame
// tick (0.12.0): the stack, the 90 s lane rotation, and the breaking-news
// takeover in/out. Split from dispatch to keep its complexity in budget.
func (d Dashboard) handleTicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case TickerMsg:
		d.setTicker(v.Items) // the active alerts, grouped into lanes
	case TickerAdvanceMsg:
		d.advanceTickerCategory() // the 90s lane rotation (driven by the pipeline)
	case TickerBreakingMsg:
		item := v.Item
		d.breaking = &item                                             // a breaking event takes the marquee centre
		d.lastBreaking, d.lastBreakingTab = d.now(), severeTabOf(item) // the window opens on this category for a while (SAM-D-17)
	case TickerBreakingDoneMsg:
		d.breaking = nil // resume normal rotation where it left off
	}
	return d.armTick(nil)
}

// dispatch routes one message to its handler.
func (d Dashboard) dispatch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		d.width, d.height = v.Width, v.Height
		return d, nil
	case SnapshotMsg:
		return d.applySnapshot(v)
	case RecentSnapshotMsg:
		return d.applyRecent(v), nil
	case TickerMsg, TickerAdvanceMsg, TickerBreakingMsg, TickerBreakingDoneMsg:
		return d.handleTicker(msg) // 0.12.0: the global event ticker's messages, one owner
	case SevereMsg, SevereReadingMsg:
		return d.handleSevere(msg), nil // 0.13.0: the severe window's messages, one owner
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
		return d.applyTick(), nil
	case vizTickMsg:
		return d.vizFrame()
	case voiceErrMsg:
		d.voiceErr = v.err.Error()
		return d.open(modalVoice), nil // alone on top (N-7; the Q6 exclusivity test)
	case tea.KeyPressMsg:
		return d.handleKeyPress(v)
	}
	return d, nil
}

// handleSevere applies one severe-window message (0.13.0): the published
// index, or which event the radio is reading (the ▶ follows it; "" ends it).
func (d Dashboard) handleSevere(msg tea.Msg) Dashboard {
	switch v := msg.(type) {
	case SevereMsg:
		return d.applySevere(v)
	case SevereReadingMsg:
		d.severeReading = v.Key
	}
	return d
}

// handleKeyPress routes a key press, un-fusing a lone esc first: a lone esc
// followed by a key reaches the model FUSED as alt+key — the terminal sends
// ESC then the byte, and the input layer has no ESC timeout to tell them
// apart (probed on a pty against 0.12.0 too: esc then `a` never opened
// About). No binding uses alt, so the only reading is the user's: esc, then
// the key (0.13.0 red-team, PTY).
func (d Dashboard) handleKeyPress(v tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	esc, key, fused := splitEscFusion(v)
	if !fused {
		return d.handleKey(v)
	}
	m, c1 := d.handleKey(esc)
	m, c2 := m.(Dashboard).handleKey(key)
	return m, tea.Batch(c1, c2)
}

// splitEscFusion recognises an alt+key that can only be a lone esc fused
// with the key that followed it, and hands back the two presses.
func splitEscFusion(k tea.KeyPressMsg) (esc, key tea.KeyPressMsg, fused bool) {
	if k.Mod&tea.ModAlt == 0 {
		return k, k, false
	}
	key = k
	key.Mod &^= tea.ModAlt
	switch {
	case k.Code == tea.KeyEnter || k.Code == tea.KeyTab || k.Code == tea.KeyEscape: // ESC CR / ESC HT / ESC ESC: the key itself (round 4, B-08)
	case k.Code >= 0x01 && k.Code <= 0x1a: // a raw control byte after the esc (ESC ^S): ctrl+letter
		key.Code, key.Mod = k.Code+0x60, key.Mod|tea.ModCtrl
	}
	// Every other alt chord — an arrow, backspace, delete, a non-ASCII rune —
	// is the same fusion (REVIEW R5-C-08: esc then ↓ lost both); no binding
	// uses alt, so nothing is shadowed. An UPPERCASE key after the esc arrives
	// as alt+shift+letter with no text: the letter is the key (VALIDATE
	// 2026-08-29 — esc then S/V/T/M/A were lost on a real pty).
	if key.Text == "" && key.Mod&^tea.ModShift == 0 && k.Code >= 0x20 && k.Code <= 0x7e {
		r := rune(k.Code)
		if key.Mod&tea.ModShift != 0 {
			r = unicode.ToUpper(r)
			key.Mod &^= tea.ModShift
		}
		key.Code, key.Text = r, string(r)
	}
	return tea.KeyPressMsg{Code: tea.KeyEscape}, key, true
}

// applyRecent takes a published RECENT snapshot: alerts sorted on the
// tty's own copy (L3-F16).
func (d Dashboard) applyRecent(v RecentSnapshotMsg) Dashboard {
	if err := invariant.Check(v.Snap != nil, "nil recent snapshot published to the dashboard"); err != nil {
		return d
	}
	d.recent = v.Snap
	sortAlerts(d.recent)
	if i := d.lookupIndex(); i >= 0 {
		d.selected, d.lookupRef = d.numPriority()+i, nil // the looked-up location's data arrived: the focus follows it, Details reads it from here on
	}
	return d
}

// applyTick advances the animation phase and clears an expired volume
// blink; armTick re-arms the tick only while the predicate holds.
func (d Dashboard) applyTick() Dashboard {
	d.tickArmed = false
	d.frame++
	d.advanceTicker() // 0.12.0: the ticker scrolls on the wall clock
	if d.volFlash != "" && !time.Now().Before(d.volFlashEnd) {
		d.volFlash = "" // the blink clears on the first tick after it expires (UAT 41)
	}
	return d
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
	sortAlerts(d.snap)                                                     // UAT 16.2: most severe first, everywhere
	if d.selected >= d.numPriority()+d.numRecent() && d.lookupRef == nil { // a pending lookup keeps its focus (R5-C-02)
		d.selected = 0
	}
	if changed && d.radioRepeat == RepeatWatchlist {
		d = d.pushRepeat()
	}
	return d.takeCmd()
}

// handleKey routes through the merged KeyMap (D-15: keys are data).
func (d Dashboard) handleKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch d.modal { // windows that own the keyboard while open
	case modalSetup:
		return d.handleSetupKey(key)
	case modalAdd:
		return d.handleAddKey(key)
	case modalRemove:
		return d.handleRemoveKey(key)
	case modalTheme:
		return d.handleThemeKey(key)
	case modalVoice:
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
	case "ticker-mute":
		d.tickerMuted = !d.tickerMuted // 0.12.0: mute the tone/narration; the marquee keeps scrolling
		if d.cfg.MuteTicker != nil {
			d.cfg.MuteTicker(d.tickerMuted) // persist + tell the pipeline
		}
	default:
		if act == "add-location" {
			return d.addFocused() // [ctrl+a] Favorite: add the focused recent/searched location (UAT 2026-08-27); inert on a watchlist row
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

// hydrateCmd asks the app for the hourly forecast when Details opens on a
// RECENT row that has none (UAT 72): the seed list skips the 162 KB hourly
// product on its cadence; a row someone drills into earns it.
func (d Dashboard) hydrateCmd() tea.Cmd {
	if d.modal != modalDetails || d.cfg.Hydrate == nil || d.selected < d.numPriority() || d.lookupRef != nil {
		return nil // a pending lookup is already fetching its own location
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

// modal names the one floating window that can be open (quality pass Q6,
// L3-F15): before it, ten booleans kept exclusivity by hand at ten reset
// sites and the red team found them inconsistent (help left Alerts open
// underneath, a voice error reopened the chooser over Details). Now opening
// a window closes whatever was open, by construction; the exclusivity test
// asserts it on the rendered frame.
type modal int

const (
	modalNone modal = iota
	modalHelp
	modalDetails // enter: floating forecast details (UAT 10.6)
	modalAdd     // ctrl+a / l: search modal — addMode says which (UAT 16.3/26)
	modalRemove  // shift+del: remove confirmation (UAT 26.2)
	modalAlerts  // A: alert details modal (UAT 22)
	modalStatus  // S: API status/diagnostics modal (UAT 24.2)
	modalAbout   // a: About window (UAT 68)
	modalTheme   // t: theme chooser (UAT 53)
	modalVoice   // V: voice chooser (UAT 84)
	modalSetup   // s: Setup window (UAT 100) — the first-run questions, over the dashboard like every other modal
	modalSevere  // w / ctrl+s: the Severe Weather / Disaster Events window (0.13.0)
)

// open shows m alone, scrolled to the top.
func (d Dashboard) open(m modal) Dashboard {
	if m != modalDetails {
		d.lookupRef = nil // only Details waits for a lookup (R5-B-09)
	}
	d.modal, d.modalScroll = m, 0
	return d
}

// close dismisses whatever is open.
func (d Dashboard) close() Dashboard {
	d.lookupRef = nil      // a closed Details modal no longer waits for a lookup
	d.severeDetail = false // a closed window forgets its record view (REVIEW R5-A-04)
	return d.open(modalNone)
}

// toggle opens m, or closes it when it is the open one.
func (d Dashboard) toggle(m modal) Dashboard {
	if d.modal == m {
		return d.close()
	}
	return d.open(m)
}

// toggleModal owns the open/close actions for every floating window (split
// from handleKey, P10-04). Opening one closes the others.
func (d Dashboard) toggleModal(act term.Action) (Dashboard, bool) {
	if d, ok := d.toggleSevere(act); ok {
		return d, true // 0.13.0: the severe window's open / drill-in / back-out
	}
	switch act {
	case term.HelpAction:
		return d.toggle(modalHelp), true
	case "details":
		return d.toggle(modalDetails), true // UAT 10.6
	case "alert-details":
		return d.toggle(modalAlerts), true // UAT 22
	case "status":
		return d.toggle(modalStatus), true // UAT 24.2
	case "about":
		return d.toggle(modalAbout), true // UAT 68
	case "theme":
		return d.openTheme(), true // UAT 53
	case "voice":
		return d.openVoice(), true // UAT 84
	case "setup":
		return d.openSetup(), true // UAT 100
	case "lookup":
		d = d.toggle(modalAdd) // UAT 26.4: search a location into RECENT and open its details
		d.addMode, d.addQuery, d.addErr = "lookup", "", ""
		return d, true
	case "remove":
		if d.selected < d.numPriority() {
			d = d.open(modalRemove) // UAT 26.2: confirm before touching the watchlist
		}
		return d, true
	case "close":
		return d.close(), true
	}
	return d, false
}

// toggleSevere owns the severe window's actions (0.13.0, split from
// toggleModal for P10-04): w / ctrl+s open it, enter drills into the focused
// event (FR-4), esc backs out of the record — the second esc falls through to
// close like any window.
func (d Dashboard) toggleSevere(act term.Action) (Dashboard, bool) {
	switch {
	case act == "severe":
		return d.openSevere(), true
	case act == "details" && d.modal == modalSevere:
		return d.openSevereDetail(), true
	case act == "close" && d.modal == modalSevere && d.severeDetail:
		return d.closeSevereDetail(), true
	}
	return d, false
}

// applyCommitted records a commit hook error for the add modal.
func (d Dashboard) applyCommitted(v committedMsg) Dashboard {
	if v.what == "setup" { // the Setup window owns its own outcome (UAT 100)
		if v.err != nil {
			d = d.open(modalSetup)
			d.setup.err, d.setup.focus = "setup failed: "+v.err.Error(), focusKey
			return d
		}
		d = d.close()
		d.setup, d.selected = setupState{}, 0
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
		d = d.open(modalAdd) // alone on top (N-7)
		d.addErr, d.addMode = what+" failed: "+v.err.Error(), "add"
	}
	return d
}

// applyRadioStatus mirrors the player's status into the model (B4).
func (d Dashboard) applyRadioStatus(v RadioStatusMsg) Dashboard {
	if render.PlainLine(v.Detail) != d.radioDetail {
		d.radioSince = time.Now() // a new line starts its own marquee clock (UAT 83)
	}
	// Every field crosses the boundary here, once: a station name, a product's
	// short form or a spoken line never addresses the terminal (NFR-6, round 4 A-06).
	d.radioState, d.radioStation, d.radioShort, d.radioDetail, d.radioLive, d.radioSpoken = v.State, render.PlainLine(v.Station), render.PlainLine(v.Short), render.PlainLine(v.Detail), v.Live, v.Spoken
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
