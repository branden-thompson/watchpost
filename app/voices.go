package app

// voices.go — voices: discovery (macOS say, Piper catalogue), choice, preview, install. Split from radio.go by the quality pass (Q2, pure move).

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/player"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/modes/tty"
)

// listVoices fills the correspondent list once.
func (d *radioDeck) listVoices() {
	list := d.discoverVoices()
	d.mu.Lock()
	d.voices = list
	d.mu.Unlock()
}

// renderSlots is how many ORDINARY renders may run at once on this host.
//
// Three on macOS: `say` is cheap and the broadcast reads ahead while a takeover
// and a Setup preview can both be live. Two elsewhere: a Piper render is a
// ~63 MB model load per process, and a third concurrent one buys nothing a
// listener can hear while costing memory a small box may not have.
//
// The Station Director's two reserved slots are on top of this number, so the
// worst case is renderSlots()+2 processes (perf-protocol.md §3).
func renderSlots() int {
	if runtimeGOOS() == "darwin" {
		return 3
	}
	return 2
}

// voice picks the engine for this host: macOS `say`; Piper elsewhere,
// installed under the cache dir when missing (SHA-256-pinned artifacts).
//
// Every return is wrapped by the deck's limiter (FR-12): a caller cannot get an
// uncapped voice out of the deck, which is what makes "one owner of how many
// renders at once" true rather than merely intended.
func (d *radioDeck) voice() (synth.Voice, error) {
	v, err := d.rawVoice()
	if err != nil {
		return nil, err
	}
	return synth.Limited(v, d.limiter), nil
}

// rawVoice is the unbounded engine pick. It is unexported and has exactly one
// caller — voice, above — so the wrap cannot be forgotten.
func (d *radioDeck) rawVoice() (synth.Voice, error) {
	if runtimeGOOS() == "darwin" {
		d.mu.Lock()
		name := d.voiceID
		d.mu.Unlock()
		if name == "" {
			name = d.defaultVoice() // UAT 87: the first curated voice installed
		}
		if name == systemVoice {
			name = "" // `say` with no -v (UAT 88)
		}
		return synth.SayVoice{Voice: name}, nil
	}
	spec := d.piperSpec()
	if inst, ok := synth.FindPiperVoice(d.voiceDir, spec); ok {
		return synth.PiperVoice{Install: inst}, nil
	}
	// AN UNWIRED DECK DOES NOT DOWNLOAD. Installing here is a ~63 MB fetch whose
	// progress is reported back through the program; a deck with no program and
	// no engine is a test's, and it has nowhere to report and nothing to play.
	// Production is unaffected — a deck can only reach a tune once both are set.
	// Four Linux CI rounds of this release died on this path, each in a slightly
	// different dereference, because nothing stopped the download itself.
	if !d.canInstall() {
		return nil, fmt.Errorf("no voice for %s/%s: this deck is not wired to install one", runtimeGOOS(), runtime.GOARCH)
	}
	if !synth.PiperSupported() {
		return nil, fmt.Errorf("no voice for %s/%s: install Piper or use a relayed location", runtimeGOOS(), runtime.GOARCH)
	}
	// Serialize installs: the breaking-news goroutine and a Tune can both reach
	// here on a fresh host, and two concurrent ~63 MB downloads into the same
	// dir waste bandwidth and risk a corrupt model (red-team 0.12.0 P4). Re-check
	// under the lock so a waiter reuses the winner's install.
	d.installMu.Lock()
	defer d.installMu.Unlock()
	if inst, ok := synth.FindPiperVoice(d.voiceDir, spec); ok {
		return synth.PiperVoice{Install: inst}, nil
	}
	inst, err := d.installVoice(spec, d.setDetail) // first use of a voice downloads it, progress in the player (UAT 118)
	if err != nil {
		return nil, err
	}
	return synth.PiperVoice{Install: inst}, nil
}

// piperSpec is the chosen catalogue voice (Linux/Windows): the root pick by
// name, else the first installed voice, else the catalogue default.
func (d *radioDeck) piperSpec() synth.VoiceSpec {
	d.mu.Lock()
	name := d.voiceID
	d.mu.Unlock()
	if v, ok := synth.VoiceByName(name); ok {
		return v
	}
	if installed := synth.InstalledVoices(d.voiceDir); len(installed) > 0 {
		return installed[0]
	}
	return synth.DefaultVoice()
}

// installVoice downloads Piper (once) and one catalogue voice, reporting
// progress in the player's detail line; a failure is actionable.
func (d *radioDeck) installVoice(spec synth.VoiceSpec, report func(string)) (synth.Install, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	inst, err := synth.EnsureVoice(ctx, d.voiceDir, spec, UserAgent, func(what string, done, total int64) {
		pct := 0
		if total > 0 {
			pct = int(done * 100 / total)
		}
		report(fmt.Sprintf("installing %s… %d%% (%d MB)", what, pct, done/1e6))
	})
	if err != nil {
		return synth.Install{}, fmt.Errorf("voice install failed: %w", err)
	}
	return inst, nil
}

// SetVoice chooses the ROOT correspondent (UAT 84) and hands a playing
// broadcast over so the change is heard at once.
//
// The chooser that called this retired at P4 (MVS-D-3); from now on the
// root is set in Setup like every other role, and a Setup save is a Recast. The
// mechanism is already the same one, which is why this survives P2 unchanged in
// behaviour: a root change IS a hard recast — the listener asked for it and is
// waiting to hear it.
func (d *radioDeck) SetVoice(name string) {
	d.mu.Lock()
	d.voiceID = name
	mode, ref, src := d.mode, d.ref, d.source
	d.mu.Unlock()
	if mode != "synth" || d.engine.Status().State != player.Playing {
		return
	}
	if src != nil {
		// UAT 94: hand the running broadcast over at the spot reached — no
		// restart. The Source re-resolves through the deck's resolver, which
		// has just been told the new root, so nothing needs to be passed in.
		src.Recast()
		return
	}
	go d.Tune(ref)
}

// PreviewVoice speaks the sample line in a voice, mixed over the broadcast
// (UAT 86). Runs on a tea cmd goroutine.
func (d *radioDeck) PreviewVoice(name string) {
	if name == systemVoice {
		name = ""
	}
	var v synth.Voice = synth.SayVoice{Voice: name}
	if runtimeGOOS() != "darwin" {
		spec, ok := synth.VoiceByName(name)
		if !ok {
			spec = d.piperSpec()
		}
		inst, ok := synth.FindPiperVoice(d.voiceDir, spec)
		if !ok { // a preview of an uninstalled voice downloads it first (UAT 118) — the chooser shows the progress (UAT 119)
			var err error
			if inst, err = d.installVoice(spec, d.voiceNote); err != nil {
				d.voiceNote(err.Error())
				return
			}
		}
		v = synth.PiperVoice{Install: inst}
		d.voiceNote("loading " + spec.Name + "…") // Piper reads the model on every run: a few seconds (UAT 119)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pcm, err := d.composer.SamplePCM(ctx, v)
	if err != nil {
		d.voiceNote("preview failed: " + err.Error())
		return
	}
	d.voiceNote("")                                       // sound is on its way: the line clears
	_ = d.engine.Audition(v.Rate(), bytes.NewReader(pcm)) // a sample, never the narration's line in flight (R5-B-03)
}

// voiceNote tells the Voice chooser what the deck is doing (UAT 119); nil
// program (tests) is a no-op.
func (d *radioDeck) voiceNote(text string) {
	d.send(tty.VoiceNoteMsg{Text: text})
}

// VoiceName is the correspondent's label (UAT 91:
// "System Voice", never the spoken "the System Voice").
func (d *radioDeck) VoiceName() string {
	d.mu.Lock()
	name := d.voiceID
	d.mu.Unlock()
	if name != "" {
		return name
	}
	return d.defaultVoice()
}

// Voices returns the correspondents listed at startup (instant; the
// current voice alone until discovery finishes).
func (d *radioDeck) Voices() []string {
	d.mu.Lock()
	list := append([]string(nil), d.voices...)
	d.mu.Unlock()
	if len(list) == 0 {
		if name := d.VoiceName(); name != "" {
			return []string{name}
		}
	}
	return list
}

// discoverVoices lists the correspondents available on this host: macOS
// voices from `say -v ?` (argv only); elsewhere the curated Piper catalogue
// (UAT 118) — every entry can be chosen, an uninstalled one downloads on
// first use (~63 MB) with progress in the player.
func (d *radioDeck) discoverVoices() []string {
	if runtimeGOOS() == "darwin" {
		// Through the shared discoverMacVoices, not a second `say -v ?` of its
		// own: P4's `report --verbose` needs the same answer without a deck,
		// and two implementations of "which voices does this Mac have" is how
		// the screen and the ear start disagreeing. The ceiling turns a wedged
		// `say` into "not discovered yet" rather than a goroutine held forever.
		ctx, cancel := context.WithTimeout(context.Background(), discoverTimeout)
		defer cancel()
		if list := discoverMacVoices(ctx); len(list) > 0 {
			return list
		}
		return []string{defaultMacVoice}
	}
	if !synth.PiperSupported() {
		return nil
	}
	return piperVoiceNames()
}

// piperVoiceNames is the catalogue in chooser order.
func piperVoiceNames() []string {
	cat := synth.VoiceCatalog()
	names := make([]string, 0, len(cat))
	for _, v := range cat {
		names = append(names, v.Name)
	}
	return names
}

// systemVoice is the Mac's own selected voice — `say` with no -v. On
// modern macOS that is a Siri voice `say -v ?` cannot name, so it gets an
// entry of its own (UAT 88: "what was 'macOS voice'?" — this).
const systemVoice = "System Voice"

// macVoices is the curated macOS correspondent list (HUM LEAD, UAT 87/88):
// the system voice, then the voices suited to a radio script, in this
// order. Names are exactly as `say -v ?` reports them (Eddy and Reed
// exist in many languages; only their English (US) variants qualify).
func macVoices() []string {
	return []string{
		systemVoice,
		"Aman (English (India))", "Daniel", "Eddy (English (US))", "Karen", "Moira",
		"Reed (English (US))", "Rishi", "Samantha", "Tara (English (India))", "Tessa",
	}
}

// parseSayVoices reads `say -v ?` ("Daniel              en_GB    # Hello…")
// and keeps the curated voices that are installed, in curated order.
func parseSayVoices(listing string) []string {
	installed := map[string]bool{}
	for _, line := range strings.Split(listing, "\n") {
		i := strings.Index(line, "#")
		if i < 0 {
			continue
		}
		fields := strings.Fields(line[:i])
		if len(fields) < 2 {
			continue
		}
		installed[strings.Join(fields[:len(fields)-1], " ")] = true
	}
	var out []string
	for _, name := range macVoices() {
		if installed[name] || name == systemVoice {
			out = append(out, name)
		}
	}
	return out
}

// defaultMacVoice is the technical fallback when no curated voice is
// installed: the US English voice every macOS ships with.
const defaultMacVoice = "Samantha"

// defaultVoice is the Mac's own System Voice (always present; UAT 88/91 —
// what a fresh setup heard first), else the first installed voice.
func (d *radioDeck) defaultVoice() string {
	if runtimeGOOS() == "darwin" {
		return systemVoice
	}
	if installed := synth.InstalledVoices(d.voiceDir); len(installed) > 0 {
		return installed[0].Name // the voice that will actually speak
	}
	return "" // no voice yet (Linux before Piper installs): the chip shows "—", never a Mac voice name (Linux F3)
}

// voiceDir is where Piper and its voice live: the OS cache dir (they are
// re-downloadable, pinned artifacts).
func voiceDir() string { return userCacheSubdir("piper") }
