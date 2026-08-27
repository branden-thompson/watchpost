package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/branden-thompson/watchpost/domains/radio/player"
	"github.com/branden-thompson/watchpost/domains/radio/spectrum"
	"github.com/branden-thompson/watchpost/domains/radio/stream"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/domains/weather/nws"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
)

// radioDeck is the dashboard's Radio hook (B4, architecture §5): resolves
// a location to its NWR transmitter (county SAME from the cached NWS point
// → vendored table → relay directories), tunes the pure-Go player, and
// streams status back to the UI. Everything here runs on tea cmd
// goroutines — never the update loop.
type radioDeck struct {
	p        *tea.Program
	nws      *nws.Provider
	resolver *stream.Resolver
	engine   *player.Engine
	products *synth.Products
	units    render.Units
	voiceDir string             // Piper install dir (Linux/Windows)
	analyzer *spectrum.Analyzer // visualizer bands from the engine's tap (UAT 92)
	vizBuf   []float64          // one analysis window, reused per frame

	persistMode func(tty.RadioMode) error                   // saves the [m] pick (UAT 97); nil in tests
	fire        func(snapshot.LocationRef) synth.FireReport // the location's fire report for the broadcast (UAT 114); nil = skipped
	warn        func(snapshot.Warning)                      // a fresh relay-directory failure becomes a radio_unavailable warning (Q1); nil in tests
	dirDown     map[string]bool                             // relays already warned about, so an outage warns once (guarded by mu)
	mountOwner  map[string]stream.Station                   // the current tune list: mount URL → its station, so the label follows the mount that plays (guarded by mu)

	// tuneMu makes "check the epoch, then start the engine" one step, and
	// Stop's "bump the epoch, then halt" another (round 2 N-3): without it a
	// Stop landing between the check and engine.Start left audio playing.
	// Held only around those tails — never across resolving or a voice install.
	tuneMu sync.Mutex

	mu      sync.Mutex
	station string // label of the station being played
	detail  string
	mode    string // "live" | "synth" | ""
	ref     snapshot.LocationRef
	gen     uint64                 // tune epoch (red-team 0.9.0 C-3): Tune and Stop bump it; a slow Tune that lost the race must not start playback
	repeat  tty.RepeatMode         // [r] Off | One | Watchlist (UAT 83/93)
	pref    tty.RadioMode          // [m] Synth | Nearest Relay (UAT 97) — the source the user asked for
	queue   []snapshot.LocationRef // Watchlist mode's order (the favourites, from the dashboard)
	dwell   *time.Timer            // Watchlist on a live relay: advance after liveDwell
	source  *synth.Source          // the running synthesized broadcast, if any
	voiceID string                 // chosen correspondent (UAT 84); "" = the platform default
	voices  []string               // available correspondents, listed once in the background (UAT 85)
}

// newRadioDeck wires the player. A resolver failure (a broken vendored
// table) is a build bug: surfaced, and the hook stays nil (controls inert).
func newRadioDeck(p *tea.Program, client *httpx.Client, provider *nws.Provider, units render.Units) *radioDeck {
	r, err := stream.NewResolver(stream.NewDirectory(client, "", ""))
	if err != nil {
		return nil
	}
	d := &radioDeck{p: p, nws: provider, resolver: r, units: units, products: synth.NewProducts(client, ""), voiceDir: voiceDir()}
	d.engine, err = player.New(&player.OtoOutput{}, UserAgent, d.onStatus)
	if err != nil {
		return nil
	}
	d.analyzer, err = spectrum.New(player.OutputRate)
	if err != nil {
		return nil
	}
	d.vizBuf = make([]float64, spectrum.FFTSize)
	go d.listVoices() // `say -v ?` takes seconds: never on a key press, never on a render (UAT 85)
	return d
}

// Spectrum is the visualizer feed (UAT 92): the latest window of whatever
// plays — relay or synthesized — as band levels. The one hook that runs on
// the update loop: it is called on the 50 ms visualizer tick and does one
// 2048-point FFT (~20 µs, one allocation — BenchmarkBands), so the loop
// never waits on audio.
func (d *radioDeck) Spectrum() []float64 {
	n := d.engine.Samples(d.vizBuf)
	return d.analyzer.Bands(d.vizBuf[:n])
}

// Tune implements tty.Radio: resolve, then play the first relayed station
// — or the synthesized broadcast when nothing relays this location (B4
// step 2: 89 % of transmitters).
func (d *radioDeck) Tune(ref snapshot.LocationRef) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	d.mu.Lock()
	d.ref = ref
	d.gen++
	gen, pref := d.gen, d.pref
	d.stopDwell()
	d.mu.Unlock()
	same := stream.SAMEFromUGC(d.nws.CountyUGC(ctx, ref))
	stations, statuses := d.resolver.ResolveWithStatus(ctx, ref.Lat, ref.Lon, same)
	d.noteDirectories(statuses)
	// UAT 78/97: Synth is the default — a neighbour's broadcast is a
	// neighbour's forecast. [m] Nearest Relay asks for the live station
	// instead: the covering transmitter when relayed, else the nearest
	// relayed one; none in reach still means Synth, with the reason.
	st, live := stream.Station{}, false
	if pref == tty.ModeRelay {
		st, live = chooseNearest(stations)
	}
	if !live {
		d.startSynth(ref, d.synthReason(same, ref, stations), gen)
		return
	}
	// The tune list spans every candidate station in the resolver's order
	// (Q1): a directory lists sources that may not be connected (a 404),
	// and with weatherUSA offered again a transmitter can be "relayed" by a
	// dead mount alone — the engine must fall through to the next live
	// station, as it did when that transmitter was simply not offered.
	urls, owners := tuneList(stations)
	d.tuneMu.Lock()
	defer d.tuneMu.Unlock()
	if !d.epoch(gen) {
		return // stopped or re-tuned while resolving: this tune is stale
	}
	d.mu.Lock()
	d.mountOwner = owners
	d.mu.Unlock()
	d.setMode("live", d.label(st), st.Mounts[0].Relay)
	d.engine.Start(urls, st.Callsign+" "+st.Site) // the dwell arms when the relay reports Playing (onStatus)
}

// tuneList flattens the candidate stations' mounts in order and remembers
// which station each mount belongs to, so the label can follow the mount
// that actually plays.
func tuneList(stations []stream.Station) ([]string, map[string]stream.Station) {
	var urls []string
	owners := map[string]stream.Station{}
	for _, st := range stations {
		for _, m := range st.Mounts {
			urls = append(urls, m.URL)
			owners[m.URL] = st
		}
	}
	return urls, owners
}

// followMount re-labels the deck when the engine has moved on to another
// station's mount (a candidate earlier in the list was refused). Returns
// the label now in force. Callers hold no lock.
func (d *radioDeck) followMount(mount string) {
	if mount == "" {
		return
	}
	d.mu.Lock()
	owner, ok := d.mountOwner[mount]
	current := d.station
	d.mu.Unlock()
	if !ok || d.label(owner) == current {
		return
	}
	relay := ""
	for _, m := range owner.Mounts {
		if m.URL == mount {
			relay = m.Relay
		}
	}
	d.setMode("live", d.label(owner), relay)
}

// epoch reports whether gen is still the current tune (no Stop or newer
// Tune since it began).
func (d *radioDeck) epoch(gen uint64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.gen == gen
}

// noteDirectories turns a relay directory's first failure into one
// radio_unavailable warning (Q1, DISCOVER LR-1: the weatherUSA directory
// was unreachable for every Go build since 1.22 and nobody could see it)
// and re-arms when the relay recovers, so an outage is said once.
func (d *radioDeck) noteDirectories(statuses []stream.Status) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.dirDown == nil {
		d.dirDown = map[string]bool{}
	}
	for _, st := range statuses {
		switch {
		case st.Err == nil:
			delete(d.dirDown, st.Relay)
		case !d.dirDown[st.Relay]:
			d.dirDown[st.Relay] = true
			if d.warn != nil {
				d.warn(snapshot.Warning{Code: snapshot.WarnRadioUnavailable, Provider: st.Relay,
					Message: fmt.Sprintf("relay directory %s unreachable — its transmitters are not offered until it answers: %v", st.Relay, st.Err)})
			}
		}
	}
}

// SetMode implements tty.Radio (UAT 97): [m] picks the source; a playing
// location re-tunes under the new mode at once.
func (d *radioDeck) SetMode(mode tty.RadioMode) {
	d.mu.Lock()
	d.pref = mode
	ref, playing, persist := d.ref, d.mode != "", d.persistMode
	d.mu.Unlock()
	if persist != nil {
		_ = persist(mode) // a failed save is not a playback failure; the pick still applies for this run
	}
	if playing && d.engine.Status().State != player.Stopped {
		go d.Tune(ref)
	}
}

// synthReason explains the Synth default: the unrelayed covering
// transmitter, plus the nearest live broadcast when there is one.
func (d *radioDeck) synthReason(same string, ref snapshot.LocationRef, stations []stream.Station) string {
	reason := d.unrelayedLabel(same, ref)
	if len(stations) > 0 {
		o := render.Opts{Units: d.units}
		km := stations[0].KM
		reason += fmt.Sprintf(" · nearest live: %s %s %s", stations[0].Callsign, stations[0].Site, strings.TrimSpace(o.Distance(&km)))
	}
	return reason
}

// startSynth voices the location's NWS products (architecture §5 Synth):
// the voice is the built-in `say` on macOS, Piper elsewhere — installed on
// first use with progress shown in the player (HUM LEAD: first-run install).
func (d *radioDeck) startSynth(ref snapshot.LocationRef, why string, gen uint64) {
	if !d.epoch(gen) {
		return // a stale fallback (a relay that failed after the user stopped) must not relabel anything
	}
	d.setMode("synth", "Watchpost Synth · "+ref.Label, why)
	voice, err := d.voice() // may install Piper (minutes): never under tuneMu
	d.tuneMu.Lock()
	defer d.tuneMu.Unlock()
	if !d.epoch(gen) {
		return // stopped while the voice was being found/installed
	}
	if err != nil {
		d.setMode("synth", "Watchpost Synth · "+ref.Label, err.Error())
		d.engine.Fail(err.Error()) // the reason, in the player (F2)
		return
	}
	// The sign-off names whichever voice reaches it (UAT 94: the voice may change mid-cycle).
	src, err := synth.NewSource(voice, func(ctx context.Context) ([]synth.Segment, error) { return d.segments(ctx, ref, synth.VoiceToken) },
		func(seg synth.Segment, spoken time.Duration) { d.setDetailTimed(seg.Text, spoken) })
	if err != nil {
		d.engine.Fail(err.Error())
		return
	}
	d.mu.Lock()
	src.Loop(d.repeat == tty.RepeatOne) // Watchlist ends the cycle too — then advances (UAT 93)
	d.source = src
	d.mu.Unlock()
	d.engine.StartSource("Watchpost Synth ("+voice.Name()+")", src.Rate(), src.Open)
}

// segments composes one broadcast cycle: the location's current
// observation and alerts (from the provider, served by the client cache)
// plus the office's latest products.
func (d *radioDeck) segments(ctx context.Context, ref snapshot.LocationRef, voiceName string) ([]synth.Segment, error) {
	asm := snapshot.NewAssembler([]snapshot.LocationRef{ref}, []string{d.nws.ID()})
	for _, kind := range []snapshot.FetchKind{snapshot.KindObs, snapshot.KindAlerts} {
		if frag, err := d.nws.Fetch(ctx, snapshot.FetchReq{Kind: kind, Locations: []snapshot.LocationRef{ref}}); err == nil {
			asm.Apply(frag)
		}
	}
	snap := asm.Snapshot()
	if len(snap.Locations) == 0 {
		return nil, fmt.Errorf("no location")
	}
	office := d.nws.Office(ctx, ref)
	products, _ := d.products.Latest(ctx, office) // a product outage still leaves the observation and alerts to read
	zone, county := d.nws.ForecastZone(ctx, ref), d.nws.CountyUGC(ctx, ref)
	for i := range products {
		products[i].Text = synth.FilterUGC(products[i].Text, zone, county) // this location's blocks only (UAT 81)
	}
	now := time.Now()
	if z, err := time.LoadLocation(ref.TZ); err == nil && ref.TZ != "" {
		now = now.In(z)
	}
	var fire synth.FireReport
	if d.fire != nil {
		fire = d.fire(ref)
	}
	return synth.Compose(snap.Locations[0], products, now, d.units == render.UnitF, voiceName, d.stationFor(county, ref), fire), nil
}

// stationFor names the NWR transmitter the lead points listeners to (UAT
// 112): the one covering the county, else the nearest; none when the table
// is missing.
func (d *radioDeck) stationFor(countyUGC string, ref snapshot.LocationRef) synth.Station {
	if d.resolver == nil {
		return synth.Station{}
	}
	var tx *stream.Transmitter
	for _, c := range d.resolver.CoveringTransmitters(stream.SAMEFromUGC(countyUGC)) {
		tx = c
		break
	}
	if tx == nil {
		tx = d.resolver.NearestTransmitter(ref.Lat, ref.Lon)
	}
	if tx == nil {
		return synth.Station{}
	}
	return synth.Station{Callsign: tx.Callsign, Site: tx.Site, State: tx.State, FreqMHz: tx.FreqMHz}
}

// Stop implements tty.Radio.
func (d *radioDeck) Stop() {
	d.tuneMu.Lock() // one step with the halt: a Tune tail cannot slip in between (N-3)
	defer d.tuneMu.Unlock()
	d.mu.Lock()
	d.gen++     // any Tune still resolving is stale now (C-3)
	d.mode = "" // and no fallback or Watchlist advance follows a user's stop
	d.stopDwell()
	d.mu.Unlock()
	d.engine.Halt()
}

// SetVolume implements tty.Radio.
func (d *radioDeck) SetVolume(pct int) { d.engine.Volume(pct) }

// label: "KEC49 Monterey CA 162.550 MHz · 78 mi (nearest relayed)".
func (d *radioDeck) label(st stream.Station) string {
	o := render.Opts{Units: d.units}
	km := st.KM
	s := fmt.Sprintf("%s %s %s %s MHz · %s", st.Callsign, st.Site, st.State, st.FreqMHz, strings.TrimSpace(o.Distance(&km)))
	if !st.Covering {
		s += " (nearest relayed)"
	}
	return s
}

// unrelayedLabel names the covering transmitter nobody relays, so the
// row explains itself: "KEC62 San Diego — not relayed".
func (d *radioDeck) unrelayedLabel(same string, ref snapshot.LocationRef) string {
	for _, tx := range d.resolver.CoveringTransmitters(same) {
		return fmt.Sprintf("%s %s %s %s MHz — not relayed", tx.Callsign, tx.Site, tx.State, tx.FreqMHz)
	}
	return ref.Label + " — no NWR relay in reach"
}

func (d *radioDeck) setMode(mode, station, detail string) {
	d.mu.Lock()
	d.mode, d.station, d.detail = mode, station, detail
	d.mu.Unlock()
}

// setDetail updates the detail line and pushes it to the UI at once
// (install progress, the sentence being narrated).
func (d *radioDeck) setDetail(detail string) { d.setDetailTimed(detail, 0) }

// setDetailTimed also carries how long the line will be spoken, so the
// marquee can pace itself to the voice (UAT 83).
func (d *radioDeck) setDetailTimed(detail string, spoken time.Duration) {
	d.mu.Lock()
	d.detail = detail
	station, mode, ref, st := d.station, d.mode, d.ref, d.engine.Status()
	d.mu.Unlock()
	d.p.Send(tty.RadioStatusMsg{State: string(st.State), Station: station, Detail: detail, Spoken: spoken, Volume: st.Volume, Live: mode == "live", Location: snapshot.Key(ref)})
}

// onStatus forwards engine status to the dashboard; a relay that fails
// outright falls back to the synthesized broadcast (§5: Live → Synth).
func (d *radioDeck) onStatus(st player.Status) {
	d.followMount(st.Mount) // a later candidate's mount is playing: the label says which (Q1)
	d.mu.Lock()
	station, detail, mode, ref, repeat, src, gen := d.station, d.detail, d.mode, d.ref, d.repeat, d.source, d.gen
	d.mu.Unlock()
	state := st.State
	if st.Title != "" {
		detail = st.Title
	}
	if st.Err != "" && st.State == player.Failed {
		detail = st.Err
	}
	ended := st.State == player.Stopped && st.Title == player.EndedTitle
	if ended && src != nil && src.Err() != nil {
		// The stream ended because the voice could not render, not because
		// the broadcast finished (C-4/F3): say so, and never advance on it.
		state, detail, ended = player.Failed, src.Err().Error()+" — check the voice in [V], or reinstall it", false
	}
	if state == player.Stopped {
		station = "" // the row falls back to the focused location's name
	}
	d.p.Send(tty.RadioStatusMsg{State: string(state), Station: station, Detail: detail, Volume: st.Volume, Live: mode == "live", Location: snapshot.Key(ref)})
	// Off the engine goroutine (it is finishing this very status): Halt
	// inside Tune waits for it. Nothing follows a user's Stop (mode == "").
	if st.State == player.Failed && mode == "live" {
		go d.startSynth(ref, "relay unavailable — "+st.Err, gen)
	}
	if ended && mode != "" && repeat == tty.RepeatWatchlist {
		go d.advanceQueue(ref) // UAT 93: the cycle ended — next favourite
	}
	if st.State == player.Playing {
		d.armDwell(ref) // UAT 93: a live relay under Watchlist gets its dwell from the moment audio plays
	}
}
