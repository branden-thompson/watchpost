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
	"context"
	"fmt"
	"sort"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	tuimaps "github.com/branden-thompson/go-tuimaps"

	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/report"
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

// Viewport padding (UAT 14.3: left 3 with the tables fixed; right 2 with the
// rail gutter beyond it).
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
	Version string
	// Timed is the timing instrument's ear (W14, D-154): nil, and nothing is
	// measured; the app sets it only under WATCHPOST_DEBUG_TIMING=1.
	Timed func(Timing)
	// MapFrame is the map's frame recorder (D-198): nil unless
	// WATCHPOST_DEBUG_MAPFRAMES names a file.
	MapFrame func(MapFrame)
	// MapClosed is told when the map window closes - really closes, not a
	// window opened over it (D-106): the app lets the zone store's memory go,
	// the disk tier serving it back on reopen (D-162).
	MapClosed    func()
	KeyOverrides term.KeyMap                                   // user [keys] table (validated at build)
	NewMap       func(size tuimaps.Size) (*tuimaps.Map, error) // 0.18.0: builds the map at its window's size (the library moves only a sized map); nil = maps off
	MapFeed      func(ctx context.Context, ask MapAsk) MapFeed // 0.18.0: the alerts the map draws, asked off the UI goroutine
	// MapRadar is the radar the map draws (W8): the whole loop, asked off the
	// UI goroutine, shown once it is in (D-85).
	MapRadar func(ctx context.Context, ask MapAsk) MapRadar
	// MapRadarSource is the file's word for the lower 48's radar: "iem", or
	// MRMS, the default (D-83).
	MapRadarSource string
	// MapRadarAhead is the file's hours ahead for the radar loop (D-114).
	MapRadarAhead int
	// MapQuakeFeed is the file's word for the quakes drawn (D-122): USGS's
	// feed by its name, M2.5+ over the past week by default.
	MapQuakeFeed string
	// MapProblem takes what went wrong with the map that the listener cannot
	// act on (D-124): the diagnostics', never shown. Nil drops it.
	MapProblem func(string)
	// MapTempSource is the file's word for the map's temperature, both modes:
	// "open-meteo", or NDFD, the default (D-93, D-190).
	MapTempSource string
	// MapRainDetail is the file's word for the rain's density past NDFD's
	// reach: "full", or coarse, the default (D-192).
	MapRainDetail string
	// MapFrameText asks the frame recorder for each frame's text as well (D-198).
	MapFrameText bool
	// MapUVCities is the file's count of cities UV asks EPA for: 8, 24 or
	// 48; anything else is 24, the default (D-202).
	MapUVCities int
	// MapTemperature is the temperature the map draws (W10): every hour's or
	// step's grids, each with its span, asked off the UI goroutine.
	MapTemperature func(ctx context.Context, ask MapAsk) MapTemperature
	Maps           string // 0.18.0: the file's words for the map's two Settings (UIPrefs')
	MapDescription string
	ClearMapData   func() MapCleared // 0.18.0 W3.8: the app empties what the live map cannot reach
	MapSources     []MapSource       // 0.18.0 W1.12, D-75: what the map contacts and sends, for the Status window (FR-9.4)
	MapRetention   string            // 0.18.0 W3.8: how long the map's data is kept, and the one stated total (FR-3.5, FR-3.9)
	MapScale       string            // 0.18.0 W4.3: the file's word for the scale the map opens at
	MapNearbyKm    int               // 0.18.0 W9.2: the file's nearby distance; 0 is the default
	MapLayerChoice map[string]bool   // 0.18.0 W1.11: the layers switched from their defaults, by key
	MapLayers      []MapLayer        // 0.18.0 W1.13: the app's registry of layers, in its order
	// MapCost is the app's estimate of one refresh with the layers as chosen
	// (0.18.0 W1.14, FR-9.2). Arithmetic over the snapshot: it fetches nothing.
	MapCost func(ask MapAsk, on func(key string) bool) MapCost
	// MapDetailChoice is the map's detail switched from Watchpost's defaults,
	// by key (D-65); MapAreaName names what is in view from the view's centre
	// and width, for the title (D-64). Nil draws the selected place's name.
	MapDetailChoice map[string]bool
	MapDetailLevel  string // D-67: the file's word for the detail level; Weather by default
	MapAreaName     func(centre tuimaps.LonLat, widthKm float64) string
	Resolve         func(query string) (snapshot.LocationRef, error)
	Commit          func(watch, recent []snapshot.LocationRef) error
	SetTheme        func(name string) error // live theme switch + persist (UAT 53)

	// 0.14.0 — the WATCHPOST UI group's three display preferences, written
	// together when Settings closes. One hook rather than three: they are one
	// group, they are saved at one moment, and three writes to the same file in
	// a row is three chances for two of them to land.
	SetUI        func(UIPrefs) error
	Units        string                  // "imperial" (default) | "metric" — the units at launch
	Clock        string                  // "12h" (default) | "24h" | "mil" — the clock at launch
	Voices       func() []string         // available correspondent voices (UAT 84)
	SetVoice     func(name string) error // choose + persist the ROOT voice (UAT 84)
	PreviewVoice func(name string)       // speak the sample line in a voice (UAT 86)
	Voice        string                  // the current root voice name

	// 0.14.0 — the cast, as strings. modes/tty may not import the registry
	// (make lint-imports), so the app supplies the role keys, the class list
	// and the hooks, and a parity test in app pins them to the registry.
	Cast           CastView               // who reads what, on this host
	Tones          ToneState              // the per-class mute state
	ToneClasses    []ToneClass            // the mutable classes, in draw order
	SetCast        func(CastView) error   // save the cast and re-cast the deck
	SetTones       func(ToneState) error  // save the tone mute; NOT a re-cast ([M] must not disturb the air)
	VoiceInstalled func(name string) bool // is this voice on the host? (the not-installed note)
	// OnSurface is called whenever the operator moves between surfaces (D-73).
	//
	// THE AIR FOLLOWS THE SURFACE. The alert rail is scoped to the listener's
	// filter on Observer and to the station's service area on the console — one
	// rail, one fence, and this is what moves it. The HUM LEAD's acceptance test
	// is exactly this: "when I switch to Broadcaster I don't hear alerts outside
	// my service radius … when I switch back to Observer, that alert track has
	// to re-adapt to whatever my filter settings dictate."
	// IT RETURNS A COMMAND, AND THAT IS LOAD-BEARING (D-79). Taking the air
	// STOPS THE MONITOR'S AUDIO, and halting the player calls back into the
	// program — so doing it inline would `Send` to a loop that is inside Update
	// and cannot receive, and the app would freeze. Observer does the same:
	// `withCmd(func() tea.Msg { radio.Stop(); return nil })`.
	OnSurface func(active Surface) tea.Cmd

	// StepBedRelay moves the bed's selection through the relays the station's
	// fence reaches (D-78) — the `←` / `→` controls the reference draws.
	// A COMMAND, FOR `OnSurface`'S REASON (D-79). Stepping the selection keeps
	// the relay it lands on and tells the console what it landed on, which
	// reaches the program, so doing it inline would send to a loop that is
	// inside Update and cannot receive, and freeze the app on an arrow press.
	// It plays nothing (D-215).
	StepBedRelay func(by int) tea.Cmd

	// ToggleBedRelay plays the bed's selected relay, or stops it (D-215): the
	// console's play key. A COMMAND, FOR `OnSurface`'S REASON (D-79): tuning
	// and halting the player reach the program.
	ToggleBedRelay func() tea.Cmd

	// StationArea is where the STATION transmits from and how far it reaches, at
	// launch (D-72). Changes arrive as `StationAreaMsg`; this is the value the
	// console opens with, because a message sent before the program's loop is
	// running has nobody to receive it — sent then, it deadlocks the whole app
	// at startup.
	StationArea StationAreaMsg

	Hydrate    func(ref snapshot.LocationRef) // on-demand hourly forecast for a RECENT row (UAT 72)
	Credits    []string                       // About "Data Provided by" lines — the app owns the list (UAT 75)
	MapCredits []string                       // About "Maps" lines (0.18.0 D-148): every credit in one window
	AboutNotes []string                       // About's closing lines after every credit: conditions of use, the safety framing
	Radio      Radio                          // NOAA Weather Radio playback (B4); nil = controls stay inert
	Spectrum   func() []float64               // the visualizer feed: the latest band levels 0..1 (UAT 92); nil = rows stay blank
	SaveRadio  func(RadioPrefs) error         // keeps the radio panel's choices (D-214); nil keeps nothing
	FireBoldMW float64                        // B5: FRP at which a hotspot reads emphasized (the app passes the configured rule; 0 = 50)

	// FireRadiusKm and FireIncidentRadiusKm are the two rings the fire section
	// reports against, and they are TWO because the data is two things: the
	// ring is satellite hotspots, and named incidents are admitted from a wider
	// one. The section states each ring beside its own list — a reader who is
	// told "none" needs to know what it was none of, and how far it looked
	// (UAT 2026-09-07: "Radio says none within your 16 mile fire ring - but
	// there are 2 hotspots at 11 miles", which were named incidents).
	FireRadiusKm, FireIncidentRadiusKm float64
	SeismicDays                        int                                                   // 0.11.0: the [seismic] lookback window the section words ("last N days"; 0 = 7)
	Suggest                            func(query string, limit int) []snapshot.LocationRef  // type-ahead hints for the Setup window (embedded index only; nil = enter resolves)
	Setup                              func(def snapshot.LocationRef, firmsKey string) error // persist the default location (+ the FIRMS key when given) — the Setup window's finish (UAT 100)
	OpenSetup                          bool                                                  // open the Setup window at launch (first run, no locations, or `watchpost setup`)
	FIRMSKey                           func() string                                         // the stored FIRMS key's tail ("cdef"), "" when none — the Setup window shows it is there (UAT 111)
	Stats                              func() Stats                                          // request/publish/dump counters for the [S] modal (quality pass Q0); nil = the rows are omitted
	ASCII                              bool                                                  // --ascii: the row marks and legend in their ASCII forms (A11-10; quality pass Q3)
	NarrateEvent                       func(key string)                                      // 0.13.0: [space] in the severe window reads the focused event over the radio (UAT option B); nil = the chip mutes
	EndEventRead                       func()                                                // 0.14.0 MVS-D-75: closing the window stops a read in progress; nil = it plays on

	AlertRadiusMi  int       // 0.12.0: the Alert Notification Preference at launch — 0 = All (global), >0 = only alerts within N mi of the default location
	SetAlertRadius func(int) // 0.12.0: persist the radius and tell the ticker pipeline to re-scope; nil in tests

	// THE STATION'S OWN TWO (D-115, F-87). Where it transmits from and how far it
	// serves — the settings the console's whole line-up is derived from, written
	// from the window rather than only by editing the config file.
	//
	// `Transmitter` IS A POINTER because "not set" is a real and DIFFERENT state
	// from "set to somewhere": a station with none borrows the listener's default
	// location, and the window says so rather than showing a choice nobody made.
	//
	// The setters persist AND re-derive the pool — a radius the operator changes
	// is a region the Producer must offer from on the very next cycle, which is
	// the re-derivation the HUM LEAD asked to be able to UAT.
	// THE OPERATOR'S TWO ACTS ON A SCHEDULED CARD (D-118). The schedule owns
	// both — `lineup.Moved` and `lineup.Dropped` model them — and FR-3.3 is why
	// they are events rather than setters: "an action must never be shown as
	// taken unless the schedule took it". Nil in tests, and on a build with no
	// schedule to tell.
	MoveCard func(id string, to int)
	DropCard func(id string)

	// LocateInRadius resolves what the operator typed into a location field and
	// says whether the station can broadcast about it (R4, D-130).
	//
	// THREE ANSWERS, NOT TWO. `found` is whether the place exists at all;
	// `within` is whether it is inside the station's SERVICE RADIUS — not
	// whether it is one of the 25 the pool happens to hold, which is the
	// distinction D-130 was filed for. A location outside is NOT a lookup
	// failure — HUM LEAD, 2026-09-14: it gets helper text saying "Observer
	// supports location lookup outside Broadcast Radius", which is only
	// possible if the two answers are kept apart.
	//
	// IT MAY REACH THE NETWORK, and therefore it is called ONLY from a command,
	// behind platform/debounce — never from a key handler or a render. The
	// embedded index does not hold the small places (Rainbow, CA is in neither
	// the city nor the zip table), so for them the geocoder is the only
	// authority there is.
	// FOUR ANSWERS, NOT THREE (D-151). `asked` is whether the question could be
	// PUT AT ALL: a 5-second timeout, a DNS blip or a cancelled context is not
	// the same as "no such place", and reporting it as one tells the operator a
	// real location does not exist AND disables the key that would retry it.
	LocateInRadius func(query string) (ref snapshot.LocationRef, within, found, asked bool)

	// RequestCard is the operator asking for a report at a position (R4).
	//
	// THE APP IS THE PRODUCER (D-40). This hands over WHAT was asked for; the
	// app composes the card, admits it and carries `Requested` to the Director.
	// The console does not mint cards.
	RequestCard func(ref snapshot.LocationRef, kinds report.Set, at int)

	Transmitter      *snapshot.LocationRef
	SetTransmitter   func(snapshot.LocationRef) // nil in tests
	ServiceRadiusMi  int
	SetServiceRadius func(int) // nil in tests

	// ServiceRadiusMinMi and ServiceRadiusMaxMi are the ruled bounds the window
	// validates against, HANDED IN rather than restated here.
	//
	// ONE CARRIER, NOT TWO. Holding them here as well as
	// `config.MinServiceRadiusMi`/`MaxServiceRadiusMi` would be two carriers of
	// one fact, kept honest only by a test importing both — and a test that
	// prevents drift is not the same as a fact with one owner (HUM LEAD,
	// 2026-09-13).
	//
	// THROUGH `Config` RATHER THAN BY IMPORTING `platform/config`, which would
	// compile and pass `lint-imports` and would still be wrong: nothing under
	// `modes/` reads storage. The UI is HANDED what it needs, and D-91's air
	// boundary is a classification of exactly these seams.
	//
	// ZERO IS NOT A DEFAULT, IT IS UNSET, and the window refuses every radius
	// while it is — see `serviceBounds`. There is deliberately no fallback
	// constant, because a fallback here would be the second carrier again,
	// wearing a different name.
	ServiceRadiusMinMi int
	ServiceRadiusMaxMi int

	// RelayDwell is how long Watchlist holds a live relay at launch, and
	// SetRelayDwell persists a change. Zero means the default (five minutes,
	// one NWR cycle); nil in tests.
	RelayDwell    time.Duration
	SetRelayDwell func(time.Duration)
	// History is the history's retention as the file holds it; SetHistory
	// writes a new one and applies it to the running store; ClearHistory
	// empties the store; HistoryUsage says what it holds and where (W18;
	// D-175, D-177).
	History      HistoryRetention
	SetHistory   func(HistoryRetention)
	ClearHistory func() error
	HistoryUsage func() string
	// RelayLang is which language wins when two relays share a transmitter
	// site, and SetRelayLang persists a change. "" means the default
	// (English). The listener's call, not the table's (HUM LEAD, UAT
	// 2026-09-04).
	RelayLang    string
	SetRelayLang func(string)

	// TuneRelay tunes to one named transmitter and ReadReport falls through to
	// the synthesized report — the two ways out of a relay that is up and
	// broadcasting nothing (MVS-D-76). Nil in a build without audio, where the
	// window cannot arise.
	TuneRelay  func(key string)
	ReadReport func()

	// InjectAlert fires a fabricated alert into the pipeline, and
	// DebugScenarios are what the ctrl+d window offers (F-21b). NIL WHERE THE APP
	// WIRES NO INJECTOR, and the window then has nothing to offer and says so.
	// Every fabricated alert is marked as a TEST EVENT wherever it reaches: an
	// UNMARKED hazard is never fabricated (D-152, NFR-2).
	InjectAlert    func(key string)
	DebugScenarios []DebugScenario
}

// Stats is what the app hands the [S] modal beyond the snapshots (quality
// pass Q0, plan §2.1): request counters merged across the app's clients,
// publish counters per pipeline, and the last diagnostic dump's outcome.
// ZoneShapeStats is the zone-outline store's own tally, carried as numbers.
type ZoneShapeStats struct {
	Fetched int64 // asked of the service
	Failed  int64 // asked and not got
	Served  int64 // answered from what was already held
	Held    int   // shapes in hand now
}

type Stats struct {
	Requests httpx.RequestStats
	// MapRequests are the map's own clients' counters (0.18.0 D-150): the
	// radar's, the temperature's, the basemap's tiles'. The map's other
	// hosts share the station's client, and Requests holds them.
	MapRequests httpx.RequestStats
	Pipelines   [2]PipelineStats // [0] priority, [1] recent
	// MapProblems are the map's last problems the listener cannot act on
	// (D-124): the diagnostics' alone.
	MapProblems []string

	// ZoneShapes is what the zone-outline store has done since launch
	// (0.17.0). **A new path over the network with no counters is invisible**:
	// when an alert draws no area there is otherwise nothing to say whether
	// the shapes failed or were never asked for. Plain numbers, because this
	// package may name no domain (`scripts/lint-imports.sh`).
	ZoneShapes ZoneShapeStats
	LastDump   string // "" before the first dump; else "<ts> ok <dir>" or "<ts> failed: <reason>"
	DumpHint   string // how to trigger a dump on this platform

	// The window's own header row (0.14.0): how long this run has been up, what
	// it is, and whether there is a newer one. Latest is "" until the check has
	// an answer — a failed update check is not a failure of anything.
	Uptime  time.Duration
	Version string
	Latest  string
	Behind  bool
	// CheckEnabled reports whether the release check is switched on. Off, the
	// row says nothing about being current rather than claiming a freshness
	// nobody asked it to confirm.
	CheckEnabled bool

	// Endpoints maps a provider id to the HOSTS it talks to (0.14.0). The app
	// knows which client each provider was built with; the snapshot does not
	// carry it, and the attribution strings cannot supply it — FIRMS credits
	// earthdata.nasa.gov while its API is firms.modaps.eosdis.nasa.gov.
	//
	// Several providers share a host (nws and nws-marine are both
	// api.weather.gov), which is why [S] keys its table by ENDPOINT: the
	// request counters are per host, so a host has exactly one set of them and
	// a provider does not.
	Endpoints map[string][]string
}

// UIPrefs is the WATCHPOST UI group's answer, as config words. Strings, not
// render types, because this crosses the app seam and the app is what writes
// them to the file.
type UIPrefs struct {
	Theme          string
	Units          string
	Clock          string
	Maps           string // "on" (the default) or "off" (0.18.0)
	MapDescription string // "with" (the default), "instead" or "off"
	MapScale       string // "state" (the default), "county" or "region"
	MapNearbyKm    int
	MapRadarSource string          // "iem", or MRMS by default (D-83)
	MapTempSource  string          // "open-meteo", or NDFD by default (D-190)
	MapRainDetail  string          // "full", or coarse by default (D-192)
	MapUVCities    int             // UV's EPA cities: 8, 24 or 48; 0 is the default, 24 (D-202)
	MapRadarAhead  int             // the radar loop's hours ahead (D-114)
	MapQuakeFeed   string          // the quakes drawn, USGS's feed by its name (D-122)
	MapLayers      map[string]bool // the layers switched from their defaults
	MapDetail      map[string]bool // the map's detail switched from its defaults (D-65)
	MapDetailLevel string          // "essential", "weather" (the default), "standard" or "full" (D-67)
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

// Key is the persisted form (D-214).
func (m RepeatMode) Key() string {
	switch m {
	case RepeatOne:
		return "one"
	case RepeatWatchlist:
		return "watchlist"
	}
	return "off"
}

// ParseRepeatMode is the persisted form read back: Off for anything else.
func ParseRepeatMode(s string) RepeatMode {
	switch s {
	case "one":
		return RepeatOne
	case "watchlist":
		return RepeatWatchlist
	}
	return RepeatOff
}

// next is the [r] cycle: Off → One → Watchlist → Off.
func (m RepeatMode) next() RepeatMode { return (m + 1) % 3 }

// RadioStatusMsg reports the player's condition (B4).
// VoiceNoteMsg is the radio deck's word about a voice (UAT 119): what is
// happening between a preview or pick and the first sound — a download with its
// progress, the model loading — so a ten-second wait on Linux never reads as
// "broken". "" clears the line.
//
// IT GOES TO SETTINGS (F-41). The Settings cast rows draw these words, under the
// row that asked — without a drawer they are sent and dropped on arrival, which
// is a preview that is silent when it works and silent when it fails.
//
// THAT IS ALSO WHY IT IS ON THE NFR-8 GREP LIST AND WHY THE LIST CANNOT READ
// ZERO. The name means "a note about a voice", which is what it is, and it
// serves the Settings cast rows rather than any retired window.
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
		"severe":        {Keys: []string{"w", "W", "ctrl+s"}, Help: "Severe Weather / Disaster Events"},
		// A CHORD, NOT A LETTER. [D] is printable and would type into any field
		// the window has; ctrl+d cannot collide with typing (F-21). Note it is
		// EOF in many terminals and some multiplexers claim it first — worth
		// confirming on a target setup before relying on it.
		"debug":        {Keys: []string{"ctrl+d"}, Help: "Diagnostics"},
		"details":      {Keys: []string{"enter"}, Help: "Location Details"}, // the window shows what it has; the short label keeps the Help columns within 133 cols
		"add-location": {Keys: []string{"ctrl+a"}, Help: "Add Location"},
		"status":       {Keys: []string{"S"}, Help: "Watchpost Status"},
		"remove":       {Keys: []string{"shift+delete"}, Help: "Remove from Watchlist"},
		"lookup":       {Keys: []string{"l"}, Help: "Lookup Location"},
		"theme":        {Keys: []string{"t"}, Help: "Choose Color Theme"},
		"setup":        {Keys: []string{"s"}, Help: "Settings"},
		"radio-play":   {Keys: []string{"space"}, Help: "Play/Pause · Read Event"},
		"radio-repeat": {Keys: []string{"r"}, Help: "Repeat: Off / One / Watchlist"},
		"radio-viz":    {Keys: []string{"v"}, Help: "Visualizer"},
		"radio-mode":   {Keys: []string{"m"}, Help: "Radio Mode: Synth / Nearest Relay"},
		"voice":        {Keys: []string{"V"}, Help: "Correspondents"}, // opens Setup at the cast (MVS-D-3)
		"ticker-mute":  {Keys: []string{"M"}, Help: "Alert Tones"},
		"radio-vol-up": {Keys: []string{"+", "="}, Help: "Volume Up"},
		"radio-vol-dn": {Keys: []string{"-"}, Help: "Volume Down"},
		"units-f":      {Keys: []string{"f"}, Help: "°F"},
		"units-c":      {Keys: []string{"c"}, Help: "°C"},
		"nav-up":       {Keys: []string{"up"}, Help: "Navigate"},
		"nav-down":     {Keys: []string{"down"}, Help: "Navigate"},
		"alert-prev":   {Keys: []string{"left"}, Help: "Previous Alert"},
		"alert-next":   {Keys: []string{"right"}, Help: "Next Alert"},
		"close":        {Keys: []string{"esc"}, Help: "Close"},
		actMap:         {Keys: []string{"g"}, Help: "Map"}, // 0.18.0 FR-1.1; the map's other keys wait for W1.16's one ruling
	}
}

// Dashboard is the root TTY model.
type Dashboard struct {
	cfg       Config
	keys      term.KeyMap
	snap      *snapshot.Snapshot
	recent    *snapshot.Snapshot
	width     int
	height    int
	units     render.Units
	clockFmt  render.Clock // how times of day are written (render/clock.go); `clock()` is the wall clock
	selected  int
	alertIdx  int
	recentOff int // scroll offset (interaction lands with tab section nav)
	// consoleKeys is the console's bindings with the user's [keys] overrides
	// folded in (FR-1.5, D-158). Merged here because this is where the merge
	// error is already actionable; the Router and the help view both read it, so
	// what the operator presses and what the help PRINTS cannot disagree.
	consoleKeys term.KeyMap

	// keysWithheld names [keys] overrides that apply on Observer but would have
	// collided on the console, so the console kept its own (F-114).
	//
	// IT IS DRAWN, and that is what makes withholding legal under D-15. A
	// conflicting override is never a silent win; the console's help window is
	// where the entry did NOT apply, so it is the window that says so. Held
	// without a reader, this field would be the silence D-15 forbids wearing a
	// name that denies it.
	keysWithheld []string

	// liveOffset is how far the CONSOLE'S line-up sits below its LIVE slot,
	// mirrored onto this surface by the Router on every update (D-156, D-160).
	//
	// OBSERVER DRAWS THESE WINDOWS; THE CONSOLE OWNS WHAT THEY ACT ON. The
	// Line-Up Request window turns the slot the operator typed into a
	// running-order index (D-119), and the arithmetic belongs to
	// `Broadcaster.indexForSlot` — which this surface cannot reach. Only the
	// Router holds both.
	//
	// ZERO IS THE RUNNING STATION'S ANSWER, which is also what an unset field
	// reads as — so `requestSchedule` is tested through the Router rather than
	// by calling it directly, or the wiring could be absent and nothing would
	// say so (D-156).
	liveOffset int

	mapPane mapPane
	mapsOff bool        // 0.18.0: the Setting; g says so and builds nothing (W1.8)
	mapDesc mapDescMode // 0.18.0: the description with the picture, instead of it, or off (W1.10)
	// The Maps tab's others (0.18.0 batch 9): the scale the map opens at, the
	// nearby distance, the layer choices as one comparable word, and the last
	// estimate of a refresh's cost, asked in Update and read by the frame.
	mapScale        mapScaleMode
	mapNearbyKm     int
	mapLayerChoice  string
	mapCost         MapCost
	mapDetailChoice string         // the map's detail choices as one comparable word (D-65)
	mapDetailLevel  tuimaps.Detail // how much of the basemap is drawn (D-67, go-tuiMaps D-82)
	mapRadarIEM     bool           // IEM for the lower 48's radar, else MRMS (D-83)
	mapTempNDFD     bool           // NDFD for the map's temperature, both modes, else Open-Meteo (D-93, D-190)
	mapRainFull     bool           // Open-Meteo's full density for the rain past NDFD's reach (D-192)
	mapUVCities     int            // how many cities UV asks EPA for: 8, 24 or 48 (D-202)
	mapRadarAhead   int            // the radar loop's hours ahead: 1, 3, 6 or 12 (D-114)
	mapQuakeFeed    string         // the quakes drawn: USGS's feed by its name (D-122)
	mapKeys         term.KeyMap
	modal           modal  // the ONE open window (quality pass Q6, L3-F15): exclusivity by construction, not by ten reset sites
	addMode         string // "add" | "lookup" (shared search modal, UAT 26.3/26.4)
	// addLocate is the DEBOUNCED answer about what has been typed into the
	// search box, kept only while the window is serving the CONSOLE (D-129,
	// D-130). On Observer it stays zero: the listener's lookup reaches anywhere
	// and has nothing to check.
	addLocate  locateState
	lookupRef  *snapshot.LocationRef // the location a lookup opened Details for, until its data lands (HUM LEAD UAT 2026-08-28: else the modal shows the old top RECENT row meanwhile)
	addErr     string                // resolve failure surfaced in the modal
	setup      setupState
	relayFault relayFaultState

	// request is the Line-Up Request window's own model (R4).
	request     requestState
	debug       debugState
	voiceIdx    int
	voiceList   []string        // snapshot of the hook's list, taken when the chooser opens (UAT 85: never from View)
	radioVoice  string          // the chosen correspondent (chip label)
	addQuery    string          // add-location search buffer
	modalScroll int             // shared scroll for floating modals (UAT 10.4)
	under       []stackedWindow // the windows under the one shown, nearest last (D-107)
	resumed     modal           // the window returned to, whose resume runs at the end of this Update (D-107)

	// surface is which surface the operator is looking at, mirrored by the
	// Router (D-92).
	//
	// THE DASHBOARD IS OBSERVER, AND IT STILL NEEDS THIS. `[s]` is forwarded to
	// this model from the CONSOLE, so the Settings window can be open over a
	// surface that is not the one that owns it — and D-18 rules that some rows
	// belong to one surface only. Without this the window has no way to know
	// which it is being asked for.
	//
	// MIRRORED, NEVER DECIDED. The Router owns which surface is active, exactly
	// as it owns the gain.
	surface Surface

	// cardID says WHICH Broadcaster card modalCard is open on and cardRows DRAWS
	// it, title and all, as the console handed them over (D-88). HANDED, NEVER
	// REACHED FOR: the Dashboard has no lineup and must not grow one — the same
	// rule the console follows for the power and the bed.
	//
	// A RENDERER AND NOT A SNAPSHOT, which the ASCII parity gate holds it to.
	// Anything built once in the console's glyph vocabulary is drawn later by a
	// window with a vocabulary of its own, and the two disagree as soon as
	// `--ascii` flips after the hand-over — a bullet in the title and arrows in
	// the chips, with no ASCII form. Asking the console at DRAW time, with the
	// window's own Opts, means there is no moment at which the two can differ —
	// and it is why the TITLE comes back from the renderer too rather than being
	// handed over as a finished string.
	//
	// THE IDENTITY IS THE CARD'S ID AND NOT ITS TITLE, which is what `Card.ID` is
	// for: "ID addresses the card for the life of the lineup." A rendered title
	// changes with the glyph set, so matching on one would lose the open card the
	// moment anything about its appearance moved.
	//
	// cardGen moves whenever the hand-over changes what would be drawn, and it is
	// what puts an uncomparable field into a comparable memo key — the stand-in
	// setupGen already makes for Setup's two maps.
	cardID string

	// THE CARD WINDOW'S OWN TWO QUESTIONS (D-118): change this card's position,
	// or drop it from the running order. Both are drawn OVER the card, so the
	// operator can still read what they are about to move or discard.
	cardAct    cardAction
	cardMoveTo string // the digits buffer for [P]
	cardErr    string
	cardRows   func(render.Opts) (string, []string)
	cardGen    int

	// cardGround is the tile tone `modalCard` floats on, "" for the standard
	// modal ground (D-127). A breaking alert's window wears the SAME category
	// tint its card does — HUM LEAD, UAT 2026-09-13: "if the Card on the layout
	// is the [w] orange … that modal should MATCH the tone, not be the blue that
	// is currently is."
	cardGround   string
	ticker       []TickerItem // 0.12.0: the active global alerts (grouped into lanes by Category)
	tickerCatIdx int          // which non-empty lane is showing (rotates every 90s)
	tickerScroll int          // tape scroll offset within the current lane
	// tickerScrolls holds each lane's tape offset across rotations, so a lane
	// resumes rather than restarts and no alert is unreachable. Shared by
	// reference across the model's copies, like the memos.
	tickerScrolls map[TickerCategory]int
	breaking      *TickerItem // 0.12.0: a breaking-news takeover — one event centred, overrides the tape until done
	// tickerMuted is the visual half of a mute the app does not have. NOT
	// SEEDED FROM CONFIG (red team 2026-09-05, C-1): ticker_muted is a 0.13.0
	// back-compat mirror, not this binary's state.
	// TestMuteDeepLinksRatherThanFlippingAHeaderChip sets it both ways to prove
	// no [M] Mute/Unmute label is drawn (MVS-D-48).
	tickerMuted  bool
	darkBG       bool                 // terminal mode (bubbletea BackgroundColorMsg)
	frame        int                  // animation phase (loading shimmer, UAT 18.2b)
	radioPlaying bool                 // [space] Play|Pause (UAT 39) — true while connecting/playing (B4)
	radioState   string               // B4: last RadioStatusMsg state ("" = never tuned)
	pendingCmd   tea.Cmd              // B4: a command a key handler queued (radio hook calls run off the update loop)
	radioStation string               // B4: resolved station label
	radioShort   string               // its short form for a narrow player; "" = none
	radioDetail  string               // B4: relay / title / failure reason
	radioLive    bool                 // B4: relayed broadcast (UAT 79: "LIVE RADIO" instead of a timeline)
	radioKey     snapshot.LocationKey // B4: the location being played (UAT 80: green ▶ in its row)
	radioSpoken  time.Duration        // UAT 83: spoken length of radioDetail
	radioSince   time.Time            // UAT 83: when radioDetail started

	radioVolume int    // 0-100 (D-19); [+]/[-] step 5, bar cells step at the 10s (UAT 41)
	volFlash    string // "+" | "-" while the press acknowledgement blinks
	volFlashEnd time.Time
	radioRepeat RepeatMode // [r] Off | One | Watchlist (UAT 93)
	radioMode   RadioMode  // [m] Synth | Nearest Relay (UAT 97)
	radioViz    bool
	vizBands    []float64        // the visualizer's latest frame (UAT 92); nil = blank rows
	vizTicking  bool             // a vizTick is in flight — never two
	tickArmed   bool             // a shimmer tick is in flight — never two (Q3: armed only while something animates)
	now         func() time.Time // the clock the header's "ago" reads (tests pin it)
	memo        *bodyMemo        // the body memo's single slot (Q3); allocated at construction, shared by every copy of the model
	mmemo       *modalMemo       // the modal memo's single slot (0.13.0, FR-10): the open window renders once per input change
	// 0.13.0: the Severe Weather / Disaster Events window (severe.go)
	severe          SevereMsg
	severeByTab     [severeNumTabs][]int // row indices per tab, in the app's sort
	severeTab       SevereTab
	severeRow       int
	severeDetail    bool      // the record replaces the table (esc backs out)
	lastBreaking    time.Time // the last breaking-news takeover: the window opens on its category for 10 min
	lastBreakingTab SevereTab
	severeReading   string // the key of the event being read over the radio ([space]); "" = none — the ▶ mark
	severeReadPause bool   // that read is PAUSED by the listener (MVS-D-74) — the mark stays, the glyph changes
}

// scopedOverrides decides which `[keys]` entries the CONSOLE can take without
// colliding with a binding it already has.
//
// THE TWO MAPS ARE SEPARATE SCOPES — D-56 is "one key, one meaning PER SURFACE" —
// and five actions appear in both: lookup, about, status, help and quit. A
// listener who rebinds one of those is aiming at the surface they use, so
// merging it into the console's scope as well can collide with a binding they
// have never seen.
//
// AN OVERRIDE FOR A CONSOLE-ONLY ACTION IS NEVER WITHHELD. It has no other scope
// to apply in, so a collision there is one the operator made inside a single
// surface, and D-15 says that is a build error rather than a silent win. It is
// passed through for `term.Merge` to refuse.
//
// AN OVERRIDE FOR AN ACTION THE CONSOLE DOES NOT HAVE is passed through too:
// `term.Merge` drops it with a note (FR-14), which is a different question that
// already has an answer.
//
// IT WITHHOLDS TO A FIXED POINT, and that is the part worth reading. Withholding
// one override returns its action to the console's own key — which may then
// collide with an override already granted. Deciding in one pass grants `about`
// the `l` that `lookup` was about to vacate, then withholds `lookup` so it keeps
// `l`, and the console ends with `l` twice. So each pass resolves ONE collision
// and the map is rebuilt from scratch, until a pass finds none.
func scopedOverrides(base, observer, over term.KeyMap) (term.KeyMap, []string) {
	if len(over) == 0 {
		return over, nil
	}
	withheld := map[term.Action]bool{}
	var notes []string
	// Bounded by the overrides: every pass adds one to `withheld`, and an action
	// is never withheld twice (P10-02).
	for range len(over) {
		act, key, held := firstClash(base, over, withheld)
		if act == "" {
			break
		}
		if _, shared := observer[act]; !shared {
			break // console-only: nothing to reconcile, let Merge refuse it
		}
		withheld[act] = true
		notes = append(notes, string(act)+": "+key+" is the console's "+string(held))
	}
	out := term.KeyMap{}
	for act, b := range over {
		if !withheld[act] {
			out[act] = b
		}
	}
	return out, notes
}

// firstClash is the first key two actions would both claim once the overrides
// are applied, naming the one to withhold and the incumbent it collides with.
//
// IT READS THE EFFECTIVE MAP, not the base: an override that has already been
// withheld leaves its action on the console's own key, and that key is then
// taken by whoever holds it. Answering from the base instead grants a vacated
// key twice.
//
// THE CLASH IT REPORTS IS THE OVERRIDDEN ACTION, because that is the one with
// somewhere else to go. The incumbent keeps what it had.
func firstClash(base, over term.KeyMap, withheld map[term.Action]bool) (term.Action, string, term.Action) {
	eff := term.KeyMap{}
	for act, b := range base {
		eff[act] = b
	}
	for _, act := range sortedActions(over) {
		if _, console := base[act]; !console || withheld[act] {
			continue
		}
		eff[act] = over[act]
	}
	owner := map[string]term.Action{}
	for _, act := range sortedActions(eff) { // deterministic: one answer per run
		for _, k := range eff[act].Keys {
			if held, dup := owner[k]; dup {
				// PREFER TO WITHHOLD THE ONE THAT WAS OVERRIDDEN.
				if _, ok := over[act]; ok && !withheld[act] {
					return act, k, held
				}
				if _, ok := over[held]; ok && !withheld[held] {
					return held, k, act
				}
				return act, k, held
			}
			owner[k] = act
		}
	}
	return "", "", ""
}

// sortedActions gives `scopedOverrides` a deterministic order, so which entry
// wins a race between two overrides for one key is the same on every run.
func sortedActions(m term.KeyMap) []term.Action {
	out := make([]term.Action, 0, len(m))
	for a := range m {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// NewDashboard builds the model, merging user key overrides with validation
// (a conflicting override is a build error, never a silent win — D-15).
func NewDashboard(cfg Config) (Dashboard, error) {
	keys, _, err := term.Merge(defaultKeyMap(), cfg.KeyOverrides)
	if err != nil {
		return Dashboard{}, fmt.Errorf("key bindings invalid: %w", err)
	}
	// AND THE CONSOLE'S BINDINGS TAKE THE SAME OVERRIDES (FR-1.5, D-158),
	// SCOPED SO ONE SURFACE'S REBIND CANNOT BREAK THE OTHER (F-114, HUM LEAD
	// 2026-09-16: collisions are reconciled, and a binding functions as expected).
	//
	// FR-1.5's exit is "an override in the user's key table changes the chord" —
	// for every console binding, `ctrl+b` (tmux's own prefix) among them. A test
	// that finds only that the swap ACTIONS ARE IN THE MAP is satisfied
	// perfectly by a map no override can reach.
	//
	// AND APPLYING OVERRIDES TO BOTH SCOPES CAN BREAK AN UPGRADE. FIVE actions
	// live in both scopes — lookup, about, status, help and quit — so a key free
	// on Observer may already be taken on the console. Settings, diagnostics and
	// the gain pair LOOK shared and are not: each surface names its own, so an
	// override for one does not reach the other. `lookup = "b"` is valid on
	// Observer and `b` is the console's bed, so applying it to both scopes would
	// make a config the operator did not change refuse to launch. `term.Merge`'s
	// own doc names that outcome: "losing a binding is a nuisance; refusing to
	// launch over one is a broken upgrade".
	//
	// SO AN OVERRIDE IS APPLIED WHERE IT FITS AND WITHHELD WHERE IT WOULD
	// COLLIDE. The listener's `b` binds lookup on Observer; the console keeps `b`
	// for the bed and `l` for lookup, and both surfaces do what the operator
	// expects. Nothing is silently lost — the withheld entries are reported, and
	// a genuine conflict INSIDE one scope is still a build error (D-15).
	consoleOver, withheld := scopedOverrides(broadcasterKeyMap(), keys, cfg.KeyOverrides)
	console, _, err := term.Merge(broadcasterKeyMap(), consoleOver)
	if err != nil {
		return Dashboard{}, fmt.Errorf("console key bindings invalid: %w", err)
	}
	mapKeys, err := mapKeysFrom(cfg.KeyOverrides)
	if err != nil {
		return Dashboard{}, err
	}
	d := Dashboard{cfg: cfg, keys: keys, mapKeys: mapKeys, mapsOff: cfg.Maps == "off", mapDesc: mapDescByKey(cfg.MapDescription), mapRadarIEM: cfg.MapRadarSource == "iem", mapTempNDFD: cfg.MapTempSource != tempSourceOpenMeteo, mapRainFull: cfg.MapRainDetail == rainDetailFull, mapUVCities: UVCitiesByCount(cfg.MapUVCities), mapRadarAhead: radarAheadByHours(cfg.MapRadarAhead), mapQuakeFeed: quakeFeedByKey(cfg.MapQuakeFeed), mapScale: mapScaleByKey(cfg.MapScale), mapNearbyKm: mapNearbyByKm(cfg.MapNearbyKm), mapLayerChoice: layerChoiceKey(cfg.MapLayerChoice), mapDetailChoice: layerChoiceKey(cfg.MapDetailChoice), mapDetailLevel: detailLevelByKey(cfg.MapDetailLevel), consoleKeys: console, keysWithheld: withheld, units: render.UnitsByKey(cfg.Units), clockFmt: render.ClockByKey(cfg.Clock), width: 80, height: 24, darkBG: true, radioVolume: 55, radioVoice: cfg.Voice, memo: &bodyMemo{}, mmemo: &modalMemo{}, tickerScrolls: map[TickerCategory]int{}, now: time.Now}
	if cfg.OpenSetup {
		d = d.openSetup() // first run: the questions come to the dashboard, not the other way round (UAT 100)
	}
	return d, nil
}

// consoleKeyMap is the console's bindings as the OPERATOR has them.
//
// ONE OWNER, READ BY BOTH THE ROUTER AND THE HELP VIEW. Reaching for
// `broadcasterKeyMap()` directly, a rebound chord would be answered by the
// Router and mis-printed by the help — the surface whose entire job is to tell
// the operator which key to press.
//
// A HAND-BUILT DASHBOARD FALLS BACK TO THE DEFAULTS. Tests construct
// `Dashboard{}` literals, and a nil map would leave the Router with no bindings
// at all — which is F-72 exactly: with `keys` nil the swap branch is skipped and
// the console cannot be arrived at.
func (d Dashboard) consoleKeyMap() term.KeyMap {
	if d.consoleKeys == nil {
		return broadcasterKeyMap()
	}
	return d.consoleKeys
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
	what string // "add" | "lookup" | "remove" | "setup": names the action in the error line (round 2 N-7)

	// What a Setup save actually WROTE. The window is seeded from cfg when it
	// opens, and cfg is captured once when the app is built — so without this
	// the next open would show the launch-time cast and silently discard what
	// the listener had just saved (UAT 2026-08-30).
	cast  CastView
	tones ToneState
	saved bool
}

// tickMsg drives the loading shimmer (UAT 18.2b) and (Q3) every
// other wall-clock element of the frame: the marquee (when the visualizer
// tick is not already redrawing), the volume blink's clearing, the [S]
// ages and the Details labels. It runs only while one of them is showing
// (plan §2.5 tick predicate: tickNeeded).
type tickMsg struct{}

// tickEvery is the one way this model makes a clock (metric D, 2026-09-08).
// TWO CLOCKS ARE CORRECT — 300 ms for the shimmer and ages, 50 ms for the
// visualizer's bars, which the shimmer tick is far too slow to draw — so this
// collapses how one is BUILT, not how many there are.
func tickEvery(d time.Duration, m tea.Msg) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return m })
}

func tick() tea.Cmd { return tickEvery(300*time.Millisecond, tickMsg{}) }

// tickNeeded is the predicate (PF-2, R2-23): true while a frame would
// differ from the last one without any message arriving.
// ACCEPTED COST — see docs/accepted-costs.md §2. `len(d.ticker) > 0` is true
// nearly always, so the frame draws every 300 ms for the life of the process.
// That is the ticker feature working, not a leak; what follows from it is that
// every per-frame cost is a 24/7 cost, which is why the memo-hit allocation pins
// are tight. A hit-path pin failure is a regression, not a pin to raise.
func (d Dashboard) tickNeeded() bool {
	switch {
	case d.volFlash != "": // pending or just expired — the tick after expiry clears it
		return true
	case d.mapPane.flash != "": // U1-11: the controls' blink, same rule
		return true
	case d.setup.flash != flashNone: // the picker's press blink, same rule
		return true
	case d.mapPane.menuFlash != flashNone: // the Overlays menu's picker blink (D-147), same rule (W14, C-3)
		return true
	case d.modal == modalStatus || d.modal == modalDetails: // [S] ages; Details "N min ago" labels and LoadingDots
		return true
	// ITS CLOCK RUNS DOWN ON ITS OWN AND ACTS AT ZERO (MVS-D-76). Without this
	// the window opens with no tick armed, so stepRelayFault is never called:
	// the countdown sits at <10> for ever, the fall-through never fires, and the
	// frame never redraws between key presses ("reactions were slow"). A window
	// that acts by itself must keep the clock that acts.
	case d.modal == modalRelayFault:
		return true
	case len(d.ticker) > 0: // 0.12.0: the marquee scrolls continuously while events are active
		return true
	case d.radioPlaying && d.radioDetail != "" && !d.vizTicking && (!d.radioLive || d.radioState != "playing"):
		return true // the marquee paces itself on the wall clock (UAT 83); the viz tick redraws faster when on; LIVE RADIO and the min player have none
	}
	return d.anyLoading() // the shimmer (UAT 18.2b)
}

// armShimmer is THE shimmer-tick arming rule, for whichever surface asks (D-56).
//
// WHAT DIFFERS BETWEEN THE SURFACES IS THE PREDICATE, NOT THE ARMING. Observer
// animates while something is loading; the console animates while a slot is
// still waiting on the Director. Those are two questions with one answer about
// what to do with the answer — so the question stays with each surface and this
// is called with it.
func armShimmer(armed, needed bool, cmd tea.Cmd) (bool, tea.Cmd) {
	if armed || !needed {
		return armed, cmd
	}
	if cmd == nil {
		return true, tick()
	}
	return true, tea.Batch(cmd, tick())
}

// armTick starts the shimmer tick when the frame needs one and none is in
// flight — the shimmer twin of armViz. Called after every Update.
func (d Dashboard) armTick(cmd tea.Cmd) (tea.Model, tea.Cmd) {
	d.tickArmed, cmd = armShimmer(d.tickArmed, d.tickNeeded(), cmd)
	return d, cmd
}

// vizTickMsg drives the visualizer (UAT 92): 20 frames a second, only while
// there is something to draw — the shimmer tick is far too slow for bars.
type vizTickMsg struct{}

func vizTick() tea.Cmd { return tickEvery(50*time.Millisecond, vizTickMsg{}) }

// Init implements tea.Model — asks the terminal for its background color
// so the window tint tracks light/dark mode (UAT 10.2). The animation tick
// arms itself from the first message that needs it (Q3).
func (d Dashboard) Init() tea.Cmd {
	if d.radioRepeat == RepeatOff || d.cfg.Radio == nil {
		return tea.RequestBackgroundColor
	}
	radio, mode := d.cfg.Radio, d.radioRepeat // a kept repeat reaches the player at launch (D-214)
	return tea.Batch(tea.RequestBackgroundColor, func() tea.Msg { radio.SetRepeat(mode, nil); return nil })
}

// Update implements tea.Model: dispatch the message, then arm the shimmer
// tick if the resulting frame animates (Q3 tick predicate).
func (d Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := d.dispatch(msg)
	next, ok := m.(Dashboard)
	if err := invariant.Check(ok, "dispatch must return the dashboard model"); err != nil {
		return m, cmd
	}
	next, resumeCmd := next.resume()  // D-107: a window returned to picks up where it was
	next, mapCmd := next.armMapTick() // 0.18.0 W2.2: the map's clock, armed after every Update
	return next.armTick(tea.Batch(cmd, resumeCmd, mapCmd))
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
		d.breaking = &item // a breaking event takes the marquee centre
		// The lane a takeover ran in IS the tab it belongs to — one registry
		// since F-21 — so the window opens on the category the listener just
		// heard about (SAM-D-17).
		d.lastBreaking, d.lastBreakingTab = d.now(), item.Category
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
		if d.modal == modalMap {
			d, settle := d.boundMap().viewMoved() // a new size is a new view: its alerts are asked once it settles (D-66); marked before it is drawn
			return d.renderMap(), settle          // the bound's least zoom depends on the size; drawn at the window's new size, in Update (D-41, D-45's size row)
		}
		return d, nil
	case SnapshotMsg:
		m, cmd := d.applySnapshot(v)
		if next, ok := m.(Dashboard); ok && next.modal == modalMap {
			next, feed := next.requestFeed().askFeed() // 0.18.0: new data, so the map's alerts are asked again (D-45's data row) - one ask in flight (D-157)
			next, radar := next.refreshRadar()         // and the radar, once its loop has stood two minutes (D-85)
			next, temp := next.refreshTemp()           // and the temperature, as it stands or the hour turns (W10)
			return next, tea.Batch(cmd, feed, radar, temp)
		}
		return m, cmd
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
	case locatePauseMsg:
		return d.handleLocatePause(v)
	case locateVerdictMsg:
		return d.handleLocateVerdict(v)
	case committedMsg:
		return d.applyCommitted(v), nil
	case RadioStatusMsg, VoiceNoteMsg, RelaySilentMsg:
		return d.handleRadio(msg) // the radio's messages, one owner
	case tickMsg:
		return d.onTick()
	case castSavedMsg, uiSavedMsg:
		return d.handleSettingsSaved(msg), nil // the Settings window's apply-on-close outcomes, one owner
	case vizTickMsg:
		return d.vizFrame()
	case historyClearedMsg:
		return d.applyHistoryCleared(v), nil
	case mapClearedMsg:
		return d.applyMapCleared(v), nil // 0.18.0 W3.8: Settings says what went
	case mapFeedMsg:
		return d.applyMapFeed(v) // 0.18.0: the alerts, set and drawn in Update (D-41)
	case mapRadarMsg:
		return d.applyMapRadar(v) // W8: the radar, set and drawn in Update (D-41)
	case mapRadarAgainMsg:
		return d.applyRadarAgain(v) // D-204: the hours ahead owed
	case mapTempMsg:
		return d.applyMapTemp(v) // W10: the temperature, set and drawn in Update (D-41)
	case forecastTickMsg:
		return d.applyForecastTick(v) // D-94: Forecast mode's playback
	case mapWorkedMsg:
		return d.applyMapWorked(v) // 0.18.0: what a Work command landed is drawn here, in Update (D-41)
	case mapTickMsg:
		return d.applyMapTick(v), nil // 0.18.0 W2.2: the library asked to be drawn now
	case mapViewSettledMsg:
		return d.applyViewSettled(v) // 0.18.0 D-66: the view stood still - its alerts are asked
	case tea.KeyPressMsg:
		return d.handleKeyPress(v)
	}
	return d, nil
}

// handleSettingsSaved applies one apply-on-close outcome from the Settings
// window — the cast and tones, or the display preferences.
//
// One case in dispatch rather than two, the way the ticker's four messages and
// the severe window's two are already grouped there: they are one window's
// writes landing, and dispatch is a routing table, not the place to enumerate
// every group Settings happens to have.
func (d Dashboard) handleSettingsSaved(msg tea.Msg) Dashboard {
	switch v := msg.(type) {
	case castSavedMsg:
		return d.applyCastSaved(v)
	case uiSavedMsg:
		return d.applyUISaved(v)
	}
	return d
}

// handleSevere applies one severe-window message (0.13.0): the published
// index, or which event the radio is reading (the ▶ follows it; "" ends it).
func (d Dashboard) handleSevere(msg tea.Msg) Dashboard {
	switch v := msg.(type) {
	case SevereMsg:
		return d.applySevere(v)
	case SevereReadingMsg:
		d.severeReading, d.severeReadPause = v.Key, v.Paused
	}
	return d
}

// handleKeyPress routes a key press, un-fusing a lone esc first: a lone esc
// followed by a key reaches the model FUSED as alt+key — the terminal sends
// ESC then the byte, and the input layer has no ESC timeout to tell them
// apart (on a pty, esc then `a` does not open About). No binding uses alt, so
// the only reading is the user's: esc, then the key (0.13.0 red-team, PTY).
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
	// is the same fusion (REVIEW R5-C-08: unsplit, esc then ↓ loses both); no
	// binding uses alt, so nothing is shadowed. An UPPERCASE key after the esc
	// arrives as alt+shift+letter with no text: the letter is the key (VALIDATE
	// 2026-08-29: unsplit, esc then S/V/T/M/A are lost on a real pty).
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
	if d.mapPane.flash != "" && !time.Now().Before(d.mapPane.flashEnd) {
		d.mapPane.flash = "" // U1-11: the controls' blink ends on the tick after it expires
	}
	if d.mapPane.menuFlash != flashNone && !time.Now().Before(d.mapPane.menuFlashEnd) {
		d.mapPane.menuFlash = flashNone // the menu's picker blink ends as Settings' does (D-147)
		d.mapPane.gen++
	}
	if d.setup.flash != flashNone && !time.Now().Before(d.setup.flashEnd) {
		// Without this the blink stays lit until something ELSE happens to
		// redraw the window ("it stays green for an extended period"). A blink
		// needs a tick to end it, not only one to start it.
		d.setup.flash = flashNone
		d.setup = d.setup.touch()
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
	if m, cmd, ok := d.routeWindowKey(key); ok {
		return m, cmd // F-184: the window shown takes what it declares it owns
	}
	act, bound := d.keys.Lookup(key.String())
	if !bound {
		return d, nil
	}
	switch act {
	case "quit":
		return d, tea.Quit
	case "units-f", "units-c":
		d.units = render.UnitF
		if act == "units-c" {
			d.units = render.UnitC
		}
		if d.modal == modalMap {
			d = d.renderMap() // 0.18.0 D-45's units row: the description's distances follow the units
		}
		d.setup.uiDirty = true // kept, as Settings' units are (U2-61)
		save := d.uiApplyCmd()
		d.setup.uiDirty = false
		return d, save
	case "ticker-mute":
		// [M] OPENS Settings at the tone rows rather than toggling them. The six
		// classes are separately mutable, and one key cannot mean six things, so
		// it takes a listener to the group that does — the same place the header
		// chip points them.
		return d.openSetupAt(firstOfGroup(groupTone)), nil
	default:
		if act == "add-location" {
			return d.addFocused() // [ctrl+a] Favorite: add the focused recent/searched location; inert on a watchlist row
		}
		if toggled, ok := d.toggleRadio(act); ok {
			return toggled.armViz().takeCmd() // [v] on / [space] play may start the visualizer (UAT 92)
		}
		if toggled, ok := d.toggleModal(act); ok {
			m, cmd := toggled.takeCmd()
			return m, tea.Batch(cmd, m.(Dashboard).hydrateCmd())
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
	ref := refOf(*loc)
	for _, r := range refsOf(d.snap) {
		if sameLocation(r, ref) { // #23: by ZIP only when there is one - a park or a lake has none
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
// L3-F15): opening a window closes whatever was open, by construction, where
// booleans kept exclusive by hand at reset sites drift apart (help left over
// Alerts, a voice error over Details); the exclusivity test asserts it on the
// rendered frame.
type modal int

const (
	modalNone modal = iota
	modalHelp
	modalDetails    // enter: floating forecast details (UAT 10.6)
	modalAdd        // ctrl+a / l: search modal — addMode says which (UAT 16.3/26)
	modalRemove     // shift+del: remove confirmation (UAT 26.2)
	modalAlerts     // A: alert details modal (UAT 22)
	modalStatus     // S: API status/diagnostics modal (UAT 24.2)
	modalAbout      // a: About window (UAT 68)
	modalSetup      // s: Setup window (UAT 100) — the first-run questions, over the dashboard like every other modal
	modalSevere     // w / ctrl+s: the Severe Weather / Disaster Events window (0.13.0)
	modalRelayFault // the relay is up and silent (MVS-D-76)
	modalDebug      // ctrl+d: diagnostics, and injection in a debug build (F-21)

	// modalCard is a Broadcaster card's full report (D-88, F-97) — the drill-down
	// D-87's manifest defers to.
	//
	// IT IS A DASHBOARD WINDOW THOUGH THE CONSOLE OWNS ITS CONTENT, which is
	// D-56's split: the console builds the body (broadcaster_detail.go) and hands
	// it over, and this is the window set that carries the reachability gate, the
	// margin survey, the memo-completeness walk and the single-value exclusivity
	// above. A console-private window would have been outside all four.
	modalCard

	// modalRequest is the operator asking for a report at a position (R4).
	//
	// A DASHBOARD WINDOW FOR THE SAME REASON `modalCard` IS (D-56): the console
	// owns what it is ABOUT — the pool it validates against, the slot it targets
	// — and this is the window set carrying the reachability gate, the margin
	// survey and the memo-completeness walk. A console-private window would be
	// outside all three.
	modalRequest

	// modalMap is the map window (0.18.0 W1.1).
	modalMap

	// numModals bounds the set; it is not itself a modal. It exists so the
	// memo-completeness guard can DERIVE the list of windows rather than carry
	// a hand-written one — a hand-written list of windows is the same shape as
	// the hand-written memo key it checks, and would miss a new window in
	// exactly the same way (F-30).
	numModals
)

// open shows m alone, scrolled to the top.
//
// THE WINDOWS ARE A STACK (D-106, D-107). A window opened from another opens
// over it, and closing it returns to the one below - Details opened from the
// map goes back to the map. A window already in the stack is returned to,
// never doubled. A search or confirmation window is replaced by what it
// opens: it is done, and never returned to.
func (d Dashboard) open(m modal) Dashboard {
	was := d.mapShown()
	d = d.openWindow(m)
	d.tellMapClosed(was)
	return d
}

// openWindow is open's work: the window shown, and the stack under it.
func (d Dashboard) openWindow(m modal) Dashboard {
	if m != modalDetails {
		d.lookupRef = nil // only Details waits for a lookup (R5-B-09)
	}
	switch at := d.stackIndex(m); {
	case m == modalNone:
		d.under = nil
	case m == d.modal:
	case at >= 0:
		d.modal, d.modalScroll, d.resumed = m, d.under[at].scroll, m
		d.under = d.under[:at:at] // the windows over it are left behind
		return d
	case d.modal != modalNone && !transient(d.modal):
		d.under = append(append(make([]stackedWindow, 0, len(d.under)+1), d.under...), stackedWindow{d.modal, d.modalScroll}) // a copy: Dashboards are values
	}
	d.modal, d.modalScroll = m, 0
	return d
}

// stackedWindow is a window under the one shown, and where its scroll was left.
type stackedWindow struct {
	modal  modal
	scroll int
}

// stackIndex is where a window is in the stack under the one shown, or -1.
func (d Dashboard) stackIndex(m modal) int {
	for i, s := range d.under {
		if s.modal == m {
			return i
		}
	}
	return -1
}

// transient reports whether a window is done once it opens another - a
// search or a confirmation - and so is replaced, never returned to (D-107).
func transient(m modal) bool { return m == modalAdd || m == modalRemove }

// close dismisses whatever is open.
//
// A CLOSING SEVERE WINDOW STOPS ITS READ (MVS-D-75). The read belongs to the
// window that started it, so nothing keeps talking about a row the listener can
// no longer see — and a read cut short is NOT marked as read, because it was
// not heard.
func (d Dashboard) close() Dashboard {
	was := d.mapShown()
	d = d.closeWindow()
	d.tellMapClosed(was)
	return d
}

// mapShown reports whether the map window is open: shown, or in the stack
// under the window shown (D-106), which returns to it.
func (d Dashboard) mapShown() bool { return d.modal == modalMap || d.stackIndex(modalMap) >= 0 }

// tellMapClosed tells the app the map has closed, when it was shown and now
// is not anywhere (D-162).
func (d Dashboard) tellMapClosed(was bool) {
	if was && !d.mapShown() && d.cfg.MapClosed != nil {
		d.cfg.MapClosed()
	}
}

// closeWindow is close's work: back to the window under it, or to none.
func (d Dashboard) closeWindow() Dashboard {
	if d.modal == modalSevere && d.severeReading != "" && d.cfg.EndEventRead != nil {
		d.cfg.EndEventRead()
	}
	d.lookupRef = nil             // a closed Details modal no longer waits for a lookup
	d.severeDetail = false        // a closed window forgets its record view (REVIEW R5-A-04)
	if n := len(d.under); n > 0 { // back to the window below (D-107), which resumes
		below := d.under[n-1]
		d.under = d.under[: n-1 : n-1]
		d.modal, d.modalScroll, d.resumed = below.modal, below.scroll, below.modal
		return d
	}
	return d.openWindow(modalNone) // close tells the app, once
}

// resume runs a window's return (D-107) at the end of the Update that
// returned to it, whoever closed what was over it: the map is drawn again,
// its work asked for, and what has stood - its radar, its temperature - asked
// again. Nothing else keeps state that goes stale under another window.
func (d Dashboard) resume() (Dashboard, tea.Cmd) {
	m := d.resumed
	d.resumed = modalNone
	if m != modalMap || d.modal != modalMap || d.mapPane.m == nil {
		return d, nil
	}
	d = d.renderMap()
	d, radar := d.refreshRadar()
	d, temp := d.refreshTemp()
	return d, tea.Batch(d.mapWorkCmd(), radar, temp)
}

// handleRadio owns the messages the radio sends the dashboard: the deck's
// status (B4), a note about a voice (UAT 119), and a relay that has gone silent
// (MVS-D-76). Grouped the way the ticker's and the severe window's messages
// already are — one owner per source rather than a case each, which is also
// what keeps dispatch under the P10 complexity bound.
func (d Dashboard) handleRadio(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case RadioStatusMsg:
		return d.applyRadioStatus(v).armViz().takeCmd()
	case VoiceNoteMsg:
		// THE DECK'S WORDS REACH THE ROW THAT ASKED (F-41). Landing them in
		// d.voiceNote would put them where nothing draws them, so a preview would
		// be silent while it worked and silent when it failed. castNote renders
		// exactly this ("the deck's own words: progress, or why it failed").
		//
		// AND IT TOUCHES THE GENERATION. The modal memo keys Settings on
		// setup.gen, so a note stored without a touch would be written and never
		// drawn — the still-picture defect (F-30).
		d.setup.note = v.Text
		d.setup = d.setup.touch()
	case RelaySilentMsg:
		return d.openRelayFault(v), nil
	}
	return d, nil
}

// onTick is the shimmer's 300 ms beat: the loading animation, and the relay
// fault window's countdown.
//
// THE COUNTDOWN RIDES THIS TICK rather than starting a timer of its own — one
// clock in the model, and a window that cannot outlive the loop that draws it.
// Split out of dispatch to keep it within the P10 complexity bound.
func (d Dashboard) onTick() (tea.Model, tea.Cmd) {
	// THE MODEL'S OWN CLOCK, not time.Now(). d.now is the clock every other
	// wall-clock element of the frame reads and the one tests pin; calling
	// time.Now() here would make this the one moving part of the frame a test
	// could not drive, leaving the countdown's WIRE unpinned while its
	// arithmetic has a test of its own.
	next, done := d.stepRelayFault(d.now())
	if done {
		return next.fallThroughRelayFault()
	}
	return next.applyTick(), nil
}

// toggle opens m, or closes it when it is the open one.
func (d Dashboard) toggle(m modal) Dashboard {
	if d.modal == m {
		return d.close()
	}
	return d.open(m)
}

// toggleModal owns the open/close actions for every floating window (split
// from handleKey, P10-04). Opening one opens it over the one shown, and
// closing it returns there (D-107).
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
	case actMap:
		return d.toggleMap(), true // 0.18.0 FR-1.1
	case "theme":
		// [t] opens Settings at the theme picker, the same way [V] opens it at
		// the correspondents. The key a listener already knows still goes where
		// the thing lives.
		return d.openSetupAt(rowTheme), true
	case "voice":
		// V keeps its binding and its place in Help's RADIO group, and opens
		// Setup SCROLLED TO the correspondents, which is where a voice is chosen
		// (MVS-D-3).
		return d.openSetupAt(rowCastAlerts), true // UAT 84 / FR-14
	case "setup":
		return d.openSetup(), true // UAT 100
	case "lookup":
		d = d.toggle(modalAdd) // UAT 26.4: search a location into RECENT and open its details
		d.addMode, d.addQuery, d.addErr = "lookup", "", ""
		d.addLocate = locateState{} // a fresh window has judged nothing yet
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

// toggleSevere routes the WINDOW actions: opening one, and the keys that belong
// to whichever one is open.
//
// SPLIT BY WINDOW AT THE COMPLEXITY CEILING (P10-04, D-159). Each window's
// keys have a function of their own, so a reader holds one window to follow
// it. Every case names its modal exactly, which is what makes the split
// ORDER-PRESERVING: a case that can only match one window lives in that
// window's function, and nothing that could match two is separated.
func (d Dashboard) toggleSevere(act term.Action) (Dashboard, bool) {
	switch act {
	case "severe":
		return d.openSevere(), true
	case "debug":
		return d.toggle(modalDebug), true
	}
	switch d.modal {
	case modalDebug:
		return d.debugAction(act)
	case modalRelayFault:
		return d.relayFaultAction(act)
	case modalSevere:
		return d.severeAction(act)
	}
	return d, false
}

// debugAction is the injection window's own keys.
func (d Dashboard) debugAction(act term.Action) (Dashboard, bool) {
	switch {
	case act == "details" && !d.debug.confirm:
		// enter ASKS. Nothing here injects: an injected alert cannot be stopped
		// once it is under way, and what it produces goes out over the
		// operator's own broadcast (HUM LEAD mock, 2026-09-07).
		return d.askDebugConfirm(), true
	case act == "details":
		next, cmd := d.chooseDebug()
		return next.withCmd(cmd), true
	case act == "close" && d.debug.confirm:
		// esc answers the question "no" and leaves the window open. Closing the
		// tool because a confirmation was declined would be the app deciding
		// what the operator meant.
		return d.cancelDebugConfirm(), true
	}
	return d, false
}

// relayFaultAction is the relay-fault window's own key.
//
// enter TAKES THE FOCUSED WAY OUT (MVS-D-76). Handled with the other windows'
// actions rather than in the nav switch: choosing is not navigating, and it ends
// the window.
//
// THE COMMAND IS CARRIED OUT, NOT DROPPED. A tune that never runs is the window
// doing nothing while looking like it worked.
func (d Dashboard) relayFaultAction(act term.Action) (Dashboard, bool) {
	if act != "details" {
		return d, false
	}
	next, cmd := d.chooseRelayFault()
	return next.withCmd(cmd), true
}

// severeAction is the severe-events window's own keys.
func (d Dashboard) severeAction(act term.Action) (Dashboard, bool) {
	switch {
	case act == "details":
		return d.openSevereDetail(), true
	case act == "close" && d.severeDetail:
		return d.closeSevereDetail(), true
	}
	return d, false
}

// applyCommitted records a commit hook error for the add modal.
func (d Dashboard) applyCommitted(v committedMsg) Dashboard {
	if v.what == "setup" { // the Setup window owns its own outcome (UAT 100)
		if v.err != nil {
			d = d.open(modalSetup)
			d.setup.err, d.setup.focus = "setup failed: "+v.err.Error(), rowFIRMSKey
			return d.settled()
		}
		if v.saved {
			// The saved cast becomes the config the window opens with, so a
			// re-open shows what is on disk rather than what was there at
			// launch. The file is the source of truth; this keeps the view
			// agreeing with it without a reload.
			d.cfg.Cast, d.cfg.Tones = v.cast, v.tones
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

// RadioPrefs are the radio panel's kept choices (D-214): the volume - the
// console's gain is the same number - the repeat and the visualizer.
type RadioPrefs struct {
	Volume int
	Repeat RepeatMode
	Viz    bool
}

// WithRadioPrefs opens the panel on the kept choices, and save keeps each
// change (nil keeps nothing).
func (d Dashboard) WithRadioPrefs(p RadioPrefs, save func(RadioPrefs) error) Dashboard {
	d.radioVolume, d.radioRepeat, d.radioViz = min(max(p.Volume, 0), 100), p.Repeat, p.Viz
	d.cfg.SaveRadio = save
	return d
}

// RadioPrefs is the panel's choices as they stand.
func (d Dashboard) RadioPrefs() RadioPrefs {
	return RadioPrefs{Volume: d.radioVolume, Repeat: d.radioRepeat, Viz: d.radioViz}
}

// saveRadioCmd keeps the panel's choices, beside whatever the press already
// asked of the player.
func (d Dashboard) saveRadioCmd() Dashboard {
	save := d.cfg.SaveRadio
	if save == nil {
		return d
	}
	p := RadioPrefs{Volume: d.radioVolume, Repeat: d.radioRepeat, Viz: d.radioViz}
	return d.withCmd(tea.Batch(d.pendingCmd, func() tea.Msg { _ = save(p); return nil })) // a failed save keeps the session's choice; the next press tries again
}

// WithSpectrum attaches the visualizer feed (UAT 92).
func (d Dashboard) WithSpectrum(feed func() []float64) Dashboard {
	d.cfg.Spectrum = feed
	return d
}

// WithVoices attaches the voice hooks (UAT 84/86).
func (d Dashboard) WithVoices(list func() []string, current string, set func(string) error, preview func(string)) Dashboard {
	d.cfg.Voices, d.cfg.SetVoice, d.cfg.PreviewVoice, d.cfg.Voice, d.radioVoice = list, set, preview, current, current
	return d
}

// WithCast attaches the 0.14.0 cast: the assignments, the tone state, the class
// list and the two save hooks.
func (d Dashboard) WithCast(cast CastView, tones ToneState, classes []ToneClass, setCast func(CastView) error, setTones func(ToneState) error, installed func(string) bool) Dashboard {
	d.cfg.Cast, d.cfg.Tones, d.cfg.ToneClasses = cast, tones, classes
	d.cfg.SetCast, d.cfg.SetTones, d.cfg.VoiceInstalled = setCast, setTones, installed
	return d
}
