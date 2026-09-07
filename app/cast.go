package app

import (
	"context"
	"fmt"
	"maps"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/config"
)

// The seam between the typed config, the host, and the role registry.
//
// domains/radio/cast holds the RULE — who inherits from whom, which voice wins,
// why the requested one lost. This file holds the WIRING: it maps the config
// struct onto cast.Config, and it answers cast.Host's questions about this
// machine. Nothing here re-implements a decision.

// runtimeGOOS is the production seam for the platform. It is a package-level
// var rather than a test-file function so a test in any file can point the
// resolution at the other platform's namespace and walk the whole fallback
// matrix on one machine — the alternative is a matrix that only ever runs half
// of itself on the developer's box.
//
// It is written only by tests (t.Cleanup restores it) and read on paths that
// already hold no lock.
var runtimeGOOS = runtime.GOOS

// discoverTimeout bounds `say -v ?`. It has been seen to take seconds on a
// loaded machine; past this the curated list stands and discovery is simply
// not finished (UAT 85). A ceiling, not an expectation.
const discoverTimeout = 30 * time.Second

// castConfig is the ONE mapper from the persisted config to the role registry's
// view of it. Every consumer goes through it, so the nine roles are named in
// exactly one place and a role added to the registry fails to compile here
// until it is mapped.
func castConfig(cfg config.Config) cast.Config {
	v := cfg.Radio.Voices
	pairs := map[cast.Role]cast.Pair{
		cast.Alerts:     pairOf(v.Alerts),
		cast.Breaking:   pairOf(v.Breaking),
		cast.SevereRead: pairOf(v.SevereRead),
		cast.Standard:   pairOf(v.Standard),
		cast.Weather:    pairOf(v.Weather),
		cast.Maritime:   pairOf(v.Maritime),
		cast.Fire:       pairOf(v.Fire),
		cast.Seismic:    pairOf(v.Seismic),
		cast.Station:    pairOf(v.Station),
	}
	for role, pair := range pairs {
		if pair.Empty() {
			delete(pairs, role) // an empty pair is "inherit", not an assignment
		}
	}
	return cast.Config{
		Root:  cfg.Voice,
		Pairs: pairs,
		Mode:  cfg.Radio.Cast,
		Tones: cast.Tones{Mode: cfg.Radio.Tones.Mode, Muted: cfg.Radio.Tones.Muted},
	}
}

// castLoaded is the cast as everything that USES it should see it: mapped from
// the file, then reconciled against the window that presents it.
//
// castConfig stays a pure mapper — it must keep naming all nine roles, because a
// role it forgot would silently never resolve — and this is the separate step
// that decides which of them the app will honour.
func castLoaded(cfg config.Config) cast.Config { return reconcile(castConfig(cfg)) }

// settingsRoles are the roles the Settings window draws a picker for — its five
// correspondent rows. The root is not here: it is not a row, it is what an
// unassigned row inherits.
func settingsRoles() []cast.Role {
	return []cast.Role{cast.Alerts, cast.Weather, cast.Maritime, cast.Fire, cast.Seismic}
}

// reconcile folds a stored cast into the shape the Settings window can SHOW.
//
// A file can name roles the window has no row for. `standard` is the one that
// bit: it is the parent of the four report roles and of the station, so
// `[radio.voices.standard] macos = "Daniel"` made the maritime, fire and seismic
// reports — and the station's own lead and sign-off — speak in Daniel, while
// every one of those rows read "System Voice", because an unassigned row shows
// the ROOT and resolution walks the TREE. The window could not show it and could
// not clear it; a save carried it forward untouched (HUM LEAD, UAT 2026-08-30:
// "the app needs to know how to deal with that if the stored configs aren't
// matching the presented UI").
//
// Two steps, and the order matters:
//
//  1. PUSH DOWN. Every drawn role with no voice of its own takes the one its
//     ancestors would have given it — so nothing a listener hears changes, and
//     the row now SAYS what it will sound. A role whose only ancestor is the
//     root is left alone: "inherit" then means "follow the root", which is what
//     the row already shows.
//  2. DROP the rest. Any assignable role the window does not draw is removed,
//     so it cannot override a drawn row invisibly again. The station goes with
//     them and follows the root, which is what its row would have said if it had
//     one.
//
// Done HERE, in the one mapper from the file to the registry, so the DECK and
// the window reconcile identically. Reconciling only the view would leave the
// broadcast reading the file's version — the screen and the ear disagreeing,
// which is the bug itself.
func reconcile(c cast.Config) cast.Config {
	drawn := map[cast.Role]bool{}
	for _, r := range settingsRoles() {
		drawn[r] = true
	}
	for _, r := range settingsRoles() {
		if !c.Pairs[r].Empty() {
			continue
		}
		for p := r.Parent(); !p.IsRoot(); p = p.Parent() {
			if pair := c.Pairs[p]; !pair.Empty() {
				if c.Pairs == nil {
					c.Pairs = map[cast.Role]cast.Pair{}
				}
				c.Pairs[r] = pair
				break
			}
		}
	}
	for _, r := range cast.Assignable() {
		if !drawn[r] {
			delete(c.Pairs, r)
		}
	}
	// And the MODE is derived, exactly as assignRole derives it: any assignment
	// means a cast is in force.
	//
	// The same disagreement in a second form. Under Single Voice the resolver
	// ignores every pair and reads the root, so a file carrying assignments with
	// no `cast = "cast"` — which a 0.13.0 config, a hand edit, or a save from
	// before the mode radio was retired can all produce — showed a listener five
	// rows of names that nothing would ever use. The window has no mode control
	// any more; nothing should be able to leave the key disagreeing with the
	// rows it governs.
	c.Mode = cast.ModeSingle
	for _, r := range settingsRoles() {
		if !c.Pairs[r].Empty() {
			c.Mode = cast.ModeCast
			break
		}
	}
	return c
}

// castToConfig maps back, for Save. Only the halves this platform owns are
// written by the caller; the other half is carried through untouched, which is
// what lets one file serve a Mac and a Linux box (data-shape §2).
func castToConfig(c cast.Config, into config.Config) config.Config {
	into.Voice = c.Root
	into.Radio.Cast = c.Mode
	into.Radio.Tones = config.Tones{Mode: c.Tones.Mode, Muted: c.Tones.Muted}
	into.Radio.Voices = config.Voices{
		Alerts:     roleVoiceOf(c.Pairs[cast.Alerts]),
		Breaking:   roleVoiceOf(c.Pairs[cast.Breaking]),
		SevereRead: roleVoiceOf(c.Pairs[cast.SevereRead]),
		Standard:   roleVoiceOf(c.Pairs[cast.Standard]),
		Weather:    roleVoiceOf(c.Pairs[cast.Weather]),
		Maritime:   roleVoiceOf(c.Pairs[cast.Maritime]),
		Fire:       roleVoiceOf(c.Pairs[cast.Fire]),
		Seismic:    roleVoiceOf(c.Pairs[cast.Seismic]),
		Station:    roleVoiceOf(c.Pairs[cast.Station]),
	}
	return into
}

func pairOf(rv config.RoleVoice) cast.Pair { return cast.Pair{MacOS: rv.MacOS, Piper: rv.Piper} }
func roleVoiceOf(p cast.Pair) config.RoleVoice {
	return config.RoleVoice{MacOS: p.MacOS, Piper: p.Piper}
}

// --- the deck as cast.Host ---
//
// LOCK DISCIPLINE, stated once here and binding on every later task.
//
// Discovered and Installed take the deck's mutex. Therefore cast.Resolve and
// cast.Validate — both of which call into the host — MUST NOT be called while
// that mutex is held: the call re-enters the deck and deadlocks.
//
// The shape every caller uses is: SNAPSHOT UNDER THE LOCK, RESOLVE OUTSIDE IT,
// STORE UNDER THE LOCK. This was a real defect found by four independent
// red-team lenses at PLAN, not a hypothetical.

// Platform implements cast.Host.
func (d *radioDeck) Platform() string { return runtimeGOOS }

// Discovered implements cast.Host: may this macOS voice name be spoken here?
//
// It is a CLOSED ALLOWLIST AT EVERY MOMENT — the curated list until `say -v ?`
// has answered, the intersection afterwards. There is deliberately no window in
// which an unknown name is trusted because discovery has not finished yet: the
// cost of trusting one wrongly is a SayVoice built from a name the host does
// not have, which is silence on an alert (RS-2).
func (d *radioDeck) Discovered(name string) bool {
	d.mu.Lock()
	discovered := d.voices
	d.mu.Unlock()
	return discoveredIn(discovered, name)
}

// piperInstallFor is the ONE way app turns a voice NAME (or key) into an
// installed Piper voice, and every caller that starts from a name must use it.
//
// FindPiperVoice locates the model by KEY — `<dir>/voices/<Key>.onnx` — so a
// name has to go through the catalogue first. A `synth.VoiceSpec{Name: name}`
// has an EMPTY Key, so it looks for `voices/.onnx` and answers no for every
// voice however plainly installed. Four call sites did exactly that, one of them
// copying another with the comment "find-only, exactly as the deck's is". On
// Linux it made every alert silent: the tone sounded, the ticker took over, and
// nothing was ever read (issue #7, an Arch box on 0.14.0).
//
// The correct form already existed in the voice-preview path and nothing else
// used it. This is that, with one owner.
func piperInstallFor(dir, name string) (synth.Install, bool) {
	spec, ok := synth.VoiceByName(name)
	if !ok {
		return synth.Install{}, false
	}
	return synth.FindPiperVoice(dir, spec)
}

// Installed implements cast.Host: is this Piper catalogue key on disk?
//
// FIND-ONLY — a stat, never an install (FR-9). The alert path calls this, and
// an alert may not wait on a 63 MB download.
func (d *radioDeck) Installed(key string) bool {
	if key == "" {
		return false
	}
	_, ok := piperInstallFor(d.voiceDir, key)
	return ok
}

// Default implements cast.Host: this machine's own last resort when even the
// root did not resolve.
func (d *radioDeck) Default() string {
	return defaultVoiceFor(runtimeGOOS, d.voiceDir)
}

// discoverMacVoices reads `say -v ?` and returns the curated voices that are
// installed, in curated order.
//
// A package-level function rather than a deck method because P4's `report
// --verbose` needs the same answer without a deck, and two implementations of
// "which voices does this Mac have" is how the screen and the ear start
// disagreeing. The context ceiling is the caller's to set; discoverTimeout is
// what both callers use.
func discoverMacVoices(ctx context.Context) []string {
	if runtimeGOOS != "darwin" {
		return nil
	}
	out, err := exec.CommandContext(ctx, "say", "-v", "?").Output()
	if err != nil {
		return nil // not answered: the curated list stands
	}
	return parseSayVoices(string(out))
}

// castChanged re-points the running broadcast's resolver at the new cast.
//
// The Source holds a function, not a snapshot, so it re-resolves each segment
// as it renders — which means there is nothing to push here beyond making sure
// the function is installed. Note the shape: the resolver closure calls
// resolveVoice, which takes d.mu itself, so this must NOT hold the lock while
// handing it over (the lock discipline above).
func (d *radioDeck) castChanged() {
	d.mu.Lock()
	src := d.source
	d.mu.Unlock()
	if src == nil {
		return
	}
	src.SetResolver(func(role cast.Role) (synth.Voice, error) {
		v, _, err := d.resolveVoice(role)
		return v, err
	})
}

// castProblems validates the current cast against this host.
//
// It exists to make the lock discipline unmissable at the one call site P1 has:
// the snapshot is taken under the lock, and cast.Validate — which calls back
// into Discovered and Installed — runs with the lock released.
func (d *radioDeck) castProblems(cfg config.Config) []cast.Problem {
	snapshot := castLoaded(cfg) // a value; nothing of the deck is read here
	return cast.Validate(snapshot, d)
}

// spokenName is the name the identity lines use for a resolved voice: the
// sentinel has no name a listener would recognise, so it introduces itself as
// "your correspondent" (UAT 88).
func spokenName(resolved string) string {
	if resolved == "" || resolved == systemVoice {
		return ""
	}
	if i := strings.Index(resolved, " ("); i > 0 {
		return resolved[:i] // "Eddy (English (US))" reads as "Eddy"
	}
	return resolved
}

// --- Task 2.7: the deck resolves by role ---

// maxUnattendedInstalls is how many Piper voices this session will fetch in the
// background without being asked (MVS-D-44, E-10).
//
// A hand-edited config can name six missing voices; installing them all on
// launch is ~380 MB the listener never requested. Past the cap the roles simply
// resolve up the tree and [S] says "install on next Save". A Setup SAVE is an
// explicit act and is not capped — it clears the counter, because at that point
// the listener has asked.
const maxUnattendedInstalls = 2

// installRetryAfter is how long a failed install is left alone. Without it an
// offline host would re-download 63 MB for every segment.
const installRetryAfter = 10 * time.Minute

// castState is everything the deck knows about who reads what. It is guarded by
// d.mu and is only ever replaced wholesale, so a reader takes a consistent view.
type castState struct {
	cfg         cast.Config
	resolutions map[cast.Role]cast.Resolution
	problems    []cast.Problem
	installs    int                  // unattended installs started this session
	failedAt    map[string]time.Time // voice key -> when its install last failed
}

// resolveVoice is the ONE resolution path: cast.Resolve for who, buildVoice for
// how. It is FIND-ONLY (FR-9) — it never takes the install mutex, so an alert
// can never wait on a 63 MB download. A missing Piper key starts at most one
// background install, subject to the session cap, and the read happens now in
// whatever voice the tree resolves to.
func (d *radioDeck) resolveVoice(role cast.Role) (synth.Voice, cast.Resolution, error) {
	d.mu.Lock()
	cfg := d.cast.cfg
	d.mu.Unlock()

	// Resolve OUTSIDE the lock: cast.Resolve calls back into Discovered and
	// Installed, which take it (P1's lock discipline).
	res := cast.Resolve(role, cfg, d)
	if want := cast.WantsInstall(role, cfg, d); want != "" {
		d.startBackgroundInstall(want)
	}
	if res.Silent() {
		return nil, res, fmt.Errorf("no voice is available on this host for %s", role.Key())
	}
	v, err := d.buildVoice(res.Spoken)
	if err != nil {
		return nil, res, err
	}
	d.recordResolution(role, res)
	return v, res, nil
}

// buildVoice turns a resolved NAME into an engine, capped by the deck's limiter
// (FR-12). It constructs, it never installs.
func (d *radioDeck) buildVoice(name string) (synth.Voice, error) {
	if runtimeGOOS == "darwin" {
		if name == systemVoice {
			name = "" // `say` with no -v (UAT 88)
		}
		return synth.Limited(synth.SayVoice{Voice: name}, d.limiter), nil
	}
	inst, ok := piperInstallFor(d.voiceDir, name)
	if !ok {
		return nil, fmt.Errorf("piper voice %q is not installed", name)
	}
	return synth.Limited(synth.PiperVoice{Install: inst}, d.limiter), nil
}

// startBackgroundInstall fetches a named-but-missing Piper voice, once, without
// blocking anything. It is the only place the session cap and the failure
// memory are applied.
// canInstall reports whether this deck can carry out a background install: it
// needs something to PLAY the voice on and something to REPORT progress to.
//
// Both halves were learned from CI, one per round. The engine came first: a test
// deck reached resolveVoice on Linux and the download's progress callback
// dereferenced a nil engine. With that guarded, the next Linux run panicked one
// line further on — a deck WITH an engine and no program, in Program.Send. The
// predicate was half a predicate, and only the platform it was never run on
// could say so.
//
// Production always has both by the time this can be reached: newRadioDeck
// returns nil unless the engine was built, and an install can only start through
// the source's resolver (radio.go:458), which is set during a tune — long after
// attachRadio assigns the program. A unit test that downloads 63 MB is a defect
// whatever it dereferences.
func (d *radioDeck) canInstall() bool { return d.engine != nil && d.p != nil }

func (d *radioDeck) startBackgroundInstall(key string) {
	d.mu.Lock()
	if d.cast.installs >= maxUnattendedInstalls {
		d.mu.Unlock()
		return // the cap: [S] reads "install on next Save"
	}
	if at, failed := d.cast.failedAt[key]; failed && time.Since(at) < installRetryAfter {
		d.mu.Unlock()
		return // an offline host must not re-download per segment
	}
	d.cast.installs++
	d.mu.Unlock()

	// THE ACCOUNTING ABOVE STILL HAPPENS; the WORK below does not. The cap is
	// about intent and is asserted by TestUnattendedInstallsAreCappedPerSession,
	// so skipping earlier would stop the cap binding — it did, and that test
	// caught it before this reached CI a third time.
	if !d.canInstall() {
		return
	}

	go func() {
		d.installMu.Lock()
		defer d.installMu.Unlock()
		if _, ok := piperInstallFor(d.voiceDir, key); ok {
			return // a concurrent caller won
		}
		spec, ok := synth.VoiceByName(key)
		if !ok {
			return // not a catalogue voice: nothing to install under that name
		}
		if _, err := d.installVoice(spec, d.setDetail); err != nil {
			d.mu.Lock()
			if d.cast.failedAt == nil {
				d.cast.failedAt = map[string]time.Time{}
			}
			d.cast.failedAt[key] = time.Now()
			d.mu.Unlock()
			return
		}
		// A host fact landed: a SOFT change. The segment already rendered plays
		// as it is; the next one rendered gets the new voice.
		d.softChanged()
	}()
}

// InstallsRemaining reports how many unattended installs this session may still
// start. [S] reads it in P4; the tests read it here.
func (d *radioDeck) InstallsRemaining() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return max(0, maxUnattendedInstalls-d.cast.installs)
}

// recordResolution memoises what each role resolved to, for [S].
func (d *radioDeck) recordResolution(role cast.Role, res cast.Resolution) {
	d.mu.Lock()
	if d.cast.resolutions == nil {
		d.cast.resolutions = map[cast.Role]cast.Resolution{}
	}
	d.cast.resolutions[role] = res
	d.mu.Unlock()
}

// Resolutions is what each role resolved to, copied on return.
func (d *radioDeck) Resolutions() map[cast.Role]cast.Resolution {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make(map[cast.Role]cast.Resolution, len(d.cast.resolutions))
	maps.Copy(out, d.cast.resolutions)
	return out
}

// Problems is what is wrong with the current cast on this host, copied on
// return. Memoised: it is recomputed only when the cast or a host fact changes.
func (d *radioDeck) Problems() []cast.Problem {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]cast.Problem(nil), d.cast.problems...)
}

// tones is the current per-class mute state.
func (d *radioDeck) tones() cast.Tones {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.cast.cfg.Tones
}

// setCast installs a new cast — the listener saved in Setup.
//
// It follows P1's lock discipline exactly, and the shape is load-bearing:
// SNAPSHOT under the lock, VALIDATE AND RESOLVE outside it, STORE under the
// lock. cast.Validate calls back into Discovered and Installed, which take
// d.mu; validating while holding it deadlocks. Four independent red-team lenses
// found that at PLAN, which is why it is written out here rather than assumed.
func (d *radioDeck) setCast(cfg cast.Config) {
	// Outside the lock, on a value nobody else can see.
	problems := cast.Validate(cfg, d)

	d.mu.Lock()
	d.cast.cfg = cfg.Clone()
	d.cast.problems = problems
	d.cast.resolutions = nil // every role re-resolves
	d.cast.installs = 0      // a SAVE is an explicit act: the cap and the
	d.cast.failedAt = nil    // failure memory both start over
	src := d.source
	d.mu.Unlock()

	if src != nil {
		src.Recast() // the listener is waiting to hear it: a HARD change
	}
	d.castChanged()
}

// setTones changes only the per-class mute state. It is deliberately NOT a
// recast: no config reload, no re-resolution, no install pass — [M] must be
// instant and must not disturb a broadcast in flight.
func (d *radioDeck) setTones(t cast.Tones) {
	d.mu.Lock()
	d.cast.cfg.Tones = cast.Tones{Mode: t.Mode, Muted: append([]string(nil), t.Muted...)}
	d.mu.Unlock()
}

// softChanged records a host fact landing: discovery finished, an install
// completed. Nobody is waiting for it, so it takes effect at the next segment
// RENDERED and the writer never re-renders for it.
func (d *radioDeck) softChanged() {
	d.mu.Lock()
	cfg, src := d.cast.cfg, d.source
	d.mu.Unlock()

	problems := cast.Validate(cfg, d) // outside the lock

	d.mu.Lock()
	d.cast.problems = problems
	d.cast.resolutions = nil
	d.mu.Unlock()

	if src != nil {
		src.Invalidate()
	}
}

// --- Task 4.7: the cast, as the TUI holds it ---

// castView maps the deck's cast onto the strings the TUI shows: THIS
// platform's half of each pair, and catalogue names rather than keys.
//
// modes/tty may not import the registry (make lint-imports), so the role keys
// travel as strings. TestSetupRoleKeysMatchTheRegistry pins them, and a
// renamed role therefore fails a test rather than silently unassigning a row.
func castView(c cast.Config) tty.CastView {
	out := tty.CastView{Mode: c.Mode, Names: map[string]string{}}
	if c.Root != "" {
		out.Names[cast.All.Key()] = c.Root
	}
	for _, r := range cast.Assignable() {
		if name := halfFor(c.Pairs[r]); name != "" {
			out.Names[r.Key()] = name
		}
	}
	return out
}

// castFromView maps back, for a save. Only THIS platform's half is written;
// the other one is carried through from the config the caller passes in, so a
// file shared between a Mac and a Linux box keeps both (FR-8).
func castFromView(v tty.CastView, into cast.Config) cast.Config {
	out := into.Clone()
	out.Mode = v.Mode
	if root, ok := v.Names[cast.All.Key()]; ok {
		out.Root = root
	}
	if out.Pairs == nil {
		out.Pairs = map[cast.Role]cast.Pair{}
	}
	for _, r := range cast.Assignable() {
		pair := out.Pairs[r]
		setHalf(&pair, v.Names[r.Key()])
		if pair.Empty() {
			delete(out.Pairs, r)
			continue
		}
		out.Pairs[r] = pair
	}
	return out
}

// halfFor reads this platform's half of a pair.
func halfFor(p cast.Pair) string {
	if runtimeGOOS == cast.PlatformDarwin {
		return p.MacOS
	}
	return p.Piper
}

// setHalf writes this platform's half, leaving the other untouched.
func setHalf(p *cast.Pair, name string) {
	if runtimeGOOS == cast.PlatformDarwin {
		p.MacOS = name
		return
	}
	p.Piper = name
}

// toneView and toneFromView carry the per-class mute across the same seam.
func toneView(t cast.Tones) tty.ToneState {
	return tty.ToneState{Mode: t.Mode, Muted: append([]string(nil), t.Muted...)}
}

func toneFromView(v tty.ToneState) cast.Tones {
	return cast.Tones{Mode: v.Mode, Muted: append([]string(nil), v.Muted...)}
}

// toneClasses is the mutable classes, in the registry's draw order, with the
// labels the Setup window shows.
func toneClasses() []tty.ToneClass {
	out := make([]tty.ToneClass, 0, len(cast.Classes()))
	for _, c := range cast.Classes() {
		out = append(out, tty.ToneClass{Key: c.Key(), Label: c.String()})
	}
	return out
}
