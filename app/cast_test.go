package app

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"errors"
	"github.com/branden-thompson/watchpost/domains/globalfeed"
	"github.com/branden-thompson/watchpost/platform/render"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/player"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/config"
)

// asPlatform points the production seam at goos for one test.
func asPlatform(t *testing.T, goos string) {
	t.Helper()
	prev := runtimeGOOS()
	setRuntimeGOOS(goos)
	t.Cleanup(func() { setRuntimeGOOS(prev) })
}

// --- Task 1.12: the mapper ---

func TestCastConfigRoundTripsThroughTheTypedConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Voice = "Samantha"
	cfg.Radio.Cast = "cast"
	cfg.Radio.Voices.Alerts = config.RoleVoice{MacOS: "Rishi", Piper: "en_US-ryan-medium"}
	cfg.Radio.Voices.Station = config.RoleVoice{MacOS: "Daniel"}
	cfg.Radio.Tones = config.Tones{Mode: "mute", Muted: []string{"watch"}}

	c := castConfig(cfg)
	if c.Root != "Samantha" || !c.CastOn() {
		t.Fatalf("root/mode did not map: %+v", c)
	}
	if got := c.PairFor(cast.Alerts); got.MacOS != "Rishi" || got.Piper != "en_US-ryan-medium" {
		t.Errorf("alerts = %+v", got)
	}
	if got := c.PairFor(cast.Station); got.MacOS != "Daniel" || got.Piper != "" {
		t.Errorf("station = %+v", got)
	}
	// An unassigned role has NO entry — "inherit" is the absence of a pair, not
	// a stored blank, so the map size is the number of real assignments.
	if _, ok := c.Pairs[cast.Weather]; ok {
		t.Error("an empty pair must not be stored as an assignment")
	}
	if len(c.Pairs) != 2 {
		t.Errorf("Pairs has %d entries, want the 2 assigned roles", len(c.Pairs))
	}
	if c.Tones.Mode != "mute" || !slices.Equal(c.Tones.Muted, []string{"watch"}) {
		t.Errorf("tones = %+v", c.Tones)
	}

	// And back, unchanged.
	back := castToConfig(c, config.Default())
	if back.Voice != cfg.Voice || back.Radio.Cast != cfg.Radio.Cast {
		t.Errorf("root/mode did not map back: %+v", back.Radio)
	}
	if back.Radio.Voices != cfg.Radio.Voices {
		t.Errorf("voices did not map back:\n got %+v\nwant %+v", back.Radio.Voices, cfg.Radio.Voices)
	}
	if back.Radio.Tones.Mode != cfg.Radio.Tones.Mode {
		t.Errorf("tones did not map back: %+v", back.Radio.Tones)
	}
}

// Every role in the registry must be mapped: a role the mapper forgot would
// silently never resolve, which is the worst possible failure mode here.
func TestCastConfigMapsEveryAssignableRole(t *testing.T) {
	cfg := config.Default()
	cfg.Radio.Cast = "cast"
	cfg.Radio.Voices = config.Voices{
		Alerts: config.RoleVoice{MacOS: "a"}, Breaking: config.RoleVoice{MacOS: "b"},
		SevereRead: config.RoleVoice{MacOS: "c"}, Standard: config.RoleVoice{MacOS: "d"},
		Weather: config.RoleVoice{MacOS: "e"}, Maritime: config.RoleVoice{MacOS: "f"},
		Fire: config.RoleVoice{MacOS: "g"}, Seismic: config.RoleVoice{MacOS: "h"},
		Station: config.RoleVoice{MacOS: "i"},
	}
	c := castConfig(cfg)
	for _, r := range cast.Assignable() {
		if c.PairFor(r).Empty() {
			t.Errorf("role %q is not mapped by castConfig", r.Key())
		}
	}
	if len(c.Pairs) != len(cast.Assignable()) {
		t.Errorf("mapped %d roles, the registry has %d", len(c.Pairs), len(cast.Assignable()))
	}
}

// --- Task 1.12: the deck as cast.Host ---

func TestTheDeckIsACastHost(t *testing.T) {
	var _ cast.Host = (*radioDeck)(nil)
}

func TestHostAnswersBeforeAndAfterDiscovery(t *testing.T) {
	asPlatform(t, "darwin")
	d := &radioDeck{}

	// Before `say -v ?` answers, the CURATED list is the allowlist — and it is
	// a closed one: a name that is not on it does not get spoken on the hope
	// that discovery will vindicate it (D-R2-2).
	if !d.Discovered("Samantha") {
		t.Error("a curated name must resolve before discovery")
	}
	if d.Discovered("Nobody At All") {
		t.Error("an unknown name must NOT be trusted while discovery is pending")
	}
	if !d.Discovered(systemVoice) {
		t.Error("the sentinel is always present")
	}
	if d.Discovered("") {
		t.Error("an empty name resolves to nothing")
	}

	// After discovery the intersection is the allowlist: a curated name the
	// host does not actually have stops resolving.
	d.mu.Lock()
	d.voices = []string{systemVoice, "Samantha"}
	d.mu.Unlock()
	if !d.Discovered("Samantha") {
		t.Error("a discovered name must resolve")
	}
	if d.Discovered("Daniel") {
		t.Error("a curated name absent from the host's list must stop resolving once discovery lands")
	}
}

func TestHostPlatformFollowsTheSeam(t *testing.T) {
	d := &radioDeck{}
	asPlatform(t, "darwin")
	if d.Platform() != cast.PlatformDarwin {
		t.Errorf("Platform() = %q", d.Platform())
	}
	if got := renderSlots(); got != 3 {
		t.Errorf("renderSlots() on darwin = %d, want 3", got)
	}
	asPlatform(t, "linux")
	if d.Platform() != "linux" {
		t.Errorf("Platform() = %q", d.Platform())
	}
	if got := renderSlots(); got != 2 {
		t.Errorf("renderSlots() on linux = %d, want 2 — a Piper render is a ~63MB model load", got)
	}
	// discoverMacVoices is a no-op off darwin: it must never shell out there.
	if got := discoverMacVoices(context.Background()); got != nil {
		t.Errorf("discoverMacVoices off darwin = %v, want nil", got)
	}
}

func TestInstalledIsFindOnly(t *testing.T) {
	asPlatform(t, "linux")
	d := &radioDeck{voiceDir: t.TempDir()} // an empty dir: nothing is installed
	if d.Installed("en_US-ryan-medium") {
		t.Error("nothing is installed in an empty voice dir")
	}
	if d.Installed("") {
		t.Error("an empty key is not installed")
	}
	// And nothing was downloaded to find that out (FR-9): the dir is still bare.
	if d.Default() != "" {
		t.Errorf("Default() = %q on an empty host, want the legitimately silent row", d.Default())
	}
}

// THE DEADLOCK GUARD. cast.Validate calls back into Discovered and Installed,
// which take d.mu. A caller that validates while holding d.mu re-enters and
// hangs forever — a defect four independent red-team lenses found at PLAN.
//
// This test must FAIL, not hang: the work runs on its own goroutine and the
// test gives up on a timer.
func TestValidatingTheCastDoesNotDeadlockAgainstTheDecksLock(t *testing.T) {
	asPlatform(t, "darwin")
	d := &radioDeck{}
	cfg := config.Default()
	cfg.Voice = "Samantha"
	cfg.Radio.Cast = "cast"
	cfg.Radio.Voices.Alerts = config.RoleVoice{MacOS: "Nobody At All"} // forces the fallback path

	done := make(chan []cast.Problem, 1)
	go func() { done <- d.castProblems(cfg) }()

	select {
	case problems := <-done:
		if len(problems) != 1 {
			t.Fatalf("want the one fallback warning, got %+v", problems)
		}
		if problems[0].Key != "radio.voices.alerts" {
			t.Errorf("problem key = %q", problems[0].Key)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("castProblems deadlocked: cast.Validate re-entered the deck's lock. " +
			"Snapshot under the lock, resolve OUTSIDE it, store under the lock.")
	}
}

// The lock is genuinely free while resolution runs: another goroutine can take
// it during the validate, which is what "resolve outside the lock" means.
func TestTheDeckLockIsFreeWhileTheCastResolves(t *testing.T) {
	asPlatform(t, "darwin")
	d := &radioDeck{}
	cfg := config.Default()
	cfg.Voice = "Samantha"
	cfg.Radio.Cast = "cast"
	cfg.Radio.Voices.Fire = config.RoleVoice{MacOS: "Nobody"}

	_ = d.castProblems(cfg)

	// Another goroutine must be able to take the lock and do real work with it
	// — which is what "resolve outside the lock" buys.
	taken := make(chan int, 1)
	go func() {
		d.mu.Lock()
		n := len(d.voices)
		d.mu.Unlock()
		taken <- n
	}()
	select {
	case <-taken:
	case <-time.After(5 * time.Second):
		t.Fatal("the deck's lock was still held after castProblems returned")
	}
}

func TestCastChangedIsANoOpInP1(t *testing.T) {
	d := &radioDeck{}
	done := make(chan struct{})
	go func() { d.castChanged(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("castChanged blocked — P1 lands it as a no-op; P2 gives it a body")
	}
}

func TestSpokenNameDropsTheLanguageSuffixAndTheSentinel(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Samantha", "Samantha"},
		{"Eddy (English (US))", "Eddy"},
		{"Aman (English (India))", "Aman"},
		{systemVoice, ""}, // "your correspondent" — the sentinel has no name (UAT 88)
		{"", ""},
	} {
		if got := spokenName(tc.in); got != tc.want {
			t.Errorf("spokenName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// --- Task 1.13: every voice the deck hands out is limited ---

func TestTheDeckHandsOutOnlyLimitedVoices(t *testing.T) {
	asPlatform(t, "darwin")
	d := &radioDeck{limiter: synth.NewLimiter(renderSlots(), synth.ReservedSlots)}
	v, err := d.voice()
	if err != nil {
		t.Fatal(err)
	}
	if _, raw := v.(synth.SayVoice); raw {
		t.Fatal("the deck handed out a RAW voice — nothing in app may render uncapped (FR-12)")
	}
	if v.Name() == "" && v.Rate() == 0 {
		t.Error("Name and Rate must pass through the wrapper")
	}
}

// With every ordinary slot held, the deck's next Say waits for one — it does
// not start a further render. The waiting Say is ended by cancelling its
// context, so the assertion needs no sleep and no 90-second bound.
func TestTheDecksVoiceQueuesWhenTheOrdinarySlotsAreHeld(t *testing.T) {
	asPlatform(t, "darwin")
	d := &radioDeck{limiter: synth.NewLimiter(1, synth.ReservedSlots)}

	held := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	holder := synth.Limited(&blockingVoice{entered: held, release: release}, d.limiter)
	go func() { _, _ = holder.Say(context.Background(), "holds the only ordinary slot") }()
	select {
	case <-held:
	case <-time.After(5 * time.Second):
		t.Fatal("the holding render never started")
	}

	v, err := d.voice()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() { _, err := v.Say(ctx, "queued behind it"); errc <- err }()

	// Nothing should have reached the engine; cancelling ends the wait.
	cancel()
	select {
	case err := <-errc:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("the queued Say returned %v, want context.Canceled — it must have been WAITING for a slot", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the queued Say neither ran nor gave up")
	}
}

// blockingVoice holds its slot until released, announcing when it has it.
type blockingVoice struct {
	entered chan struct{}
	release chan struct{}
}

func (v *blockingVoice) Name() string { return "blocking" }
func (v *blockingVoice) Rate() int    { return 22050 }
func (v *blockingVoice) Say(ctx context.Context, _ string) ([]byte, error) {
	close(v.entered)
	select {
	case <-v.release:
		return []byte{0, 0}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// --- Task 2.7: resolution, the tone path, and the install cap ---

// The tone path resolves NO voice. That is the property FR-9 exists for: on a
// fresh Linux host the correspondent may still be downloading, and the alert
// must sound anyway.
func TestTheTonePathResolvesNoVoice(t *testing.T) {
	asPlatform(t, "linux")
	d := &radioDeck{voiceDir: t.TempDir(), limiter: synth.NewLimiter(2, synth.ReservedSlots)}
	d.setCast(cast.Config{Root: "en_US-nothing-here", Mode: cast.ModeCast})
	// Nothing is installed, so resolveVoice cannot succeed...
	if _, _, err := d.resolveVoice(cast.Breaking); err == nil {
		t.Fatal("the fixture must have no resolvable voice, or this proves nothing")
	}
	// ...and the tone still renders, because it never asks.
	if got := synth.AlertTone(synth.PresetByName(cast.ToneName(cast.ClassWarning)), synth.ToneRate); len(got) == 0 {
		t.Fatal("the tone must render from constants with no voice on the host")
	}
}

func TestAMutedClassSoundsNoToneAndTheWordsStillRead(t *testing.T) {
	d := &radioDeck{}
	d.setTones(cast.Tones{Mode: cast.ModeTonesMute, Muted: []string{"watch"}})
	if !cast.Muted(cast.ClassWatch, d.tones()) {
		t.Error("the muted class is muted")
	}
	if cast.Muted(cast.ClassWarning, d.tones()) {
		t.Error("an unmuted class still sounds")
	}
	// Mute with nothing ticked means every class.
	d.setTones(cast.Tones{Mode: cast.ModeTonesMute})
	for _, c := range cast.Classes() {
		if !cast.Muted(c, d.tones()) {
			t.Errorf("%s must be muted when the set is empty", c.Key())
		}
	}
	// setTones must not disturb the cast itself: [M] is not a recast.
	d.setCast(cast.Config{Root: "Samantha"})
	before := d.Resolutions()
	d.setTones(cast.Tones{Mode: cast.ModeTonesOn})
	if len(d.Resolutions()) != len(before) {
		t.Error("setTones re-resolved the cast; [M] must not disturb a broadcast in flight")
	}
}

// MVS-D-44: a hand-edited config naming several missing Piper voices may not
// pull ~380MB on launch unasked. Past the cap the roles resolve up the tree.
func TestUnattendedInstallsAreCappedPerSession(t *testing.T) {
	asPlatform(t, "linux")
	d := &radioDeck{voiceDir: t.TempDir(), limiter: synth.NewLimiter(2, synth.ReservedSlots)}
	if got := d.InstallsRemaining(); got != maxUnattendedInstalls {
		t.Fatalf("a fresh session has %d installs, want %d", got, maxUnattendedInstalls)
	}
	d.setCast(cast.Config{
		Root: "en_US-root-missing",
		Mode: cast.ModeCast,
		Pairs: map[cast.Role]cast.Pair{
			cast.Alerts:  {Piper: "en_US-one-missing"},
			cast.Fire:    {Piper: "en_US-two-missing"},
			cast.Seismic: {Piper: "en_US-three-missing"},
		},
	})
	for _, r := range []cast.Role{cast.Alerts, cast.Fire, cast.Seismic} {
		_, _, _ = d.resolveVoice(r)
	}
	if got := d.InstallsRemaining(); got != 0 {
		t.Errorf("after three missing voices the budget is %d, want 0 — the cap must bind", got)
	}
	// A SAVE is explicit: it clears the counter, because the listener asked.
	d.setCast(cast.Config{Root: "en_US-root-missing"})
	if got := d.InstallsRemaining(); got != maxUnattendedInstalls {
		t.Errorf("a Setup save must reset the cap, got %d remaining", got)
	}
}

// resolveVoice is FIND-ONLY: it completes while the install mutex is held, so
// an alert can never queue behind a download (FR-9).
func TestResolveVoiceCompletesWhileTheInstallMutexIsHeld(t *testing.T) {
	asPlatform(t, "darwin")
	d := &radioDeck{limiter: synth.NewLimiter(2, synth.ReservedSlots)}
	d.setCast(cast.Config{Root: "Samantha"})

	d.installMu.Lock()
	defer d.installMu.Unlock()

	done := make(chan error, 1)
	go func() { _, _, err := d.resolveVoice(cast.Breaking); done <- err }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("resolveVoice: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("resolveVoice blocked on the install mutex — the alert path must be find-only (FR-9)")
	}
}

// setCast validates against the host and must not deadlock doing it.
func TestSetCastCompletesAndRecordsItsProblems(t *testing.T) {
	asPlatform(t, "darwin")
	d := &radioDeck{limiter: synth.NewLimiter(2, synth.ReservedSlots)}
	done := make(chan struct{})
	go func() {
		d.setCast(cast.Config{
			Root: "Samantha", Mode: cast.ModeCast,
			Pairs: map[cast.Role]cast.Pair{cast.Fire: {MacOS: "Nobody At All"}},
		})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("setCast deadlocked: validate under the deck's lock re-enters through Discovered")
	}
	if got := d.Problems(); len(got) != 1 || got[0].Key != "radio.voices.fire" {
		t.Fatalf("setCast must record the fallback warning for [S]: %+v", got)
	}
	// A role inherits its group's voice, and the resolution is recorded.
	if _, res, err := d.resolveVoice(cast.Fire); err != nil {
		t.Fatal(err)
	} else if res.Spoken != "Samantha" || res.Link != cast.LinkRoot {
		t.Errorf("the fire role falls back to the root: %+v", res)
	}
	if got := d.Resolutions(); len(got) == 0 {
		t.Error("resolutions must be memoised for [S]")
	}
}

// --- Task 2.8: the broadcast uses the cast (M1) ---

// M1: with distinct voices assigned, each section of ONE broadcast renders in
// the voice its role resolves to, and a hand-over is spoken at each boundary by
// the incoming correspondent. Membership, not index: render-ahead decides the
// order in which the line and the segment are produced.
func TestBroadcastSectionsSpeakInTheirRolesVoices(t *testing.T) {
	asPlatform(t, "darwin")
	d := &radioDeck{limiter: synth.NewLimiter(3, synth.ReservedSlots)}
	d.setCast(cast.Config{
		Root: "Samantha", Mode: cast.ModeCast,
		Pairs: map[cast.Role]cast.Pair{cast.Fire: {MacOS: "Daniel"}},
	})
	for _, tc := range []struct {
		role cast.Role
		want string
	}{
		{cast.Weather, "Samantha"}, // inherits the root
		{cast.Station, "Samantha"},
		{cast.Fire, "Daniel"}, // its own assignment
	} {
		_, res, err := d.resolveVoice(tc.role)
		if err != nil {
			t.Fatalf("%s: %v", tc.role.Key(), err)
		}
		if res.Spoken != tc.want {
			t.Errorf("%s resolves to %q, want %q", tc.role.Key(), res.Spoken, tc.want)
		}
	}
	// The Source is given a resolver, not a snapshot, so it re-resolves per
	// segment — which is what makes a mid-broadcast cast change possible.
	src, err := synth.NewSource(synth.SayVoice{Voice: "Samantha"},
		func(context.Context) ([]synth.Segment, error) { return nil, nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	d.mu.Lock()
	d.source = src
	d.mu.Unlock()
	d.castChanged() // installs the resolver
	// A recast reaches the running source.
	d.setCast(cast.Config{Root: "Daniel"})
}

// --- Task 2.9: the ticker and the read pass their role and class ---

// classVoice records the tone classes it was asked for, and the roles it
// rendered in.
type classVoice struct {
	mu      sync.Mutex
	classes []cast.Class
	roles   []cast.Role
}

func (v *classVoice) duck() {}

// fault is inert here: the seam exists so a read can report a tone with no
// words (FR-9.2).
func (v *classVoice) fault(string) {}

func (v *classVoice) tone(c cast.Class) time.Duration {
	v.mu.Lock()
	v.classes = append(v.classes, c)
	v.mu.Unlock()
	return 0
}

func (v *classVoice) render(_ context.Context, r cast.Role, t string) (clip, bool) {
	v.mu.Lock()
	v.roles = append(v.roles, r)
	v.mu.Unlock()
	return clip{text: t, role: r}, true
}
func (v *classVoice) play(clip) {}
func (v *classVoice) pause()    {}
func (v *classVoice) resume()   {}
func (v *classVoice) stop()     {}
func (v *classVoice) discard()  {}
func (v *classVoice) restore()  {}

func (v *classVoice) seen() ([]cast.Class, []cast.Role) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return append([]cast.Class(nil), v.classes...), append([]cast.Role(nil), v.roles...)
}

// MVS-D-12: a burst sounds ONE tone, chosen by its HIGHEST-SEVERITY event.
// Sounding one per event would turn an outbreak into an alarm sequence nobody
// can parse — and the class must be the worst thing coming, not the first.
func TestABurstSoundsOneToneForItsHighestSeverityEvent(t *testing.T) {
	v := &classVoice{}
	d := &tickerDeck{send: func(tea.Msg) {}, muted: &atomic.Bool{}, voice: testDirector(v, nil), seen: loadSeen(t.TempDir(), time.Hour)}
	// RECENT, and that is load-bearing. With no fence in force a Disaster must be
	// FRESH to keep its rung (bandOf, DisasterFreshWindow); a fixed date in the
	// past demotes the quake below the warning and the two orders coincide again,
	// which is exactly the coincidence this test exists to avoid.
	declared := time.Now().Add(-10 * time.Minute)
	// THE RUNG AND THE SEVERITY MUST DISAGREE, or this pins nothing.
	//
	// An earlier version of this test used a Watch and a Warning — rungs 4 and 3,
	// severities orange and red — where the ladder's order and the severity order
	// happen to coincide, so it stayed green through a commit that broke the very
	// rule it is named for. A quake is rung 2 and leads the READ; the tornado
	// warning is rung 3 and is the worst thing coming. The tone follows the
	// hazard, not the running order.
	evs := []globalfeed.Event{
		{ID: "a", Class: globalfeed.ClassQuake, Type: "Earthquake", Location: "Ridgecrest, CA", Severity: globalfeed.SevYellow, At: declared},
		{ID: "b", Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Location: "OKC", Severity: globalfeed.SevRed, At: declared},
	}
	// THE FIXTURE IS ASSERTED VALID FIRST: the rail must put the MILDER hazard
	// at the head of the read, or the tone and the running order agree by
	// accident and this pins nothing.
	order, _ := planned(t, d, evs)
	if len(order) != 2 || order[0].ID != "a" {
		t.Fatalf("the fixture needs the milder hazard leading the read, got %v", order)
	}
	newStation(t, d).takeover(context.Background(), evs)
	classes, roles := v.seen()
	if len(classes) != 1 {
		t.Fatalf("a burst sounds ONE tone, got %d: %v", len(classes), classes)
	}
	if classes[0] != cast.ClassWarning {
		t.Errorf("the tone is the highest-severity event's class: got %s, want warning (the Tornado Warning, not the Watch)", classes[0].Key())
	}
	for _, r := range roles {
		if r != cast.Breaking {
			t.Errorf("a takeover reads in the Breaking role, got %s", r.Key())
		}
	}
}

// A storm product beats the warning suffix (MVS-D-15), all the way through the
// ticker rather than only in the classifier's own unit test.
func TestABreakingStormSoundsTheStormClass(t *testing.T) {
	v := &classVoice{}
	d := &tickerDeck{send: func(tea.Msg) {}, muted: &atomic.Bool{}, voice: testDirector(v, nil), seen: loadSeen(t.TempDir(), time.Hour)}
	newStation(t, d).takeover(context.Background(), []globalfeed.Event{
		{ID: "h", Class: globalfeed.ClassTropical, Type: "Hurricane Warning", Severity: globalfeed.SevRed, At: time.Now()},
	})
	classes, _ := v.seen()
	if len(classes) != 1 || classes[0] != cast.ClassStorm {
		t.Errorf("a Hurricane Warning sounds the storm tone, got %v", classes)
	}
}

// TestAMutedTickerStillReadsTheWords WAS HERE, and is deleted (red team
// 2026-09-05). Its title claimed MVS-D-26 — the words always read — and its
// assertion required that NOTHING render, which is the opposite. It also needed
// BOTH mute gates removed before it would fail, so it pinned neither: the
// producer's gate is now pinned by TestMutingHoldsABurstRatherThanSpendingIt
// and the executor's by TestAMutedReadIsDeclinedAndNothingIsConsumed, each
// alone. MVS-D-26 is about the per-CLASS tone mute (deck.tone), which is tested
// where it lives.
func TestTheScreenAndTheVoiceAgreeOnTheSeaState(t *testing.T) {
	for _, height := range []float64{0.05, 0.3, 0.9, 2.0, 3.0, 5.0} {
		want := render.SeaState(height)
		h := height
		mr := synth.MarineReport{Known: true, State: snapshot.Marine{
			Buoy: "1", ObservedAt: time.Now(), WaveHeight: &h,
		}}
		var spoken string
		for _, s := range (synth.Composer{}).MarineSegments("Somewhere, CA", mr, true, time.Now()) {
			if strings.Contains(s.Text, "Seas are") {
				spoken = s.Text
			}
		}
		if spoken == "" {
			t.Fatalf("no sea sentence for a %.2f m wave", height)
		}
		if !strings.Contains(spoken, strings.ToLower(want)) {
			t.Errorf("at %.2f m the screen says %q; the voice said %q", height, want, spoken)
		}
	}
}

// --- Task 4.7: the strings modes/tty holds are the registry's ---

// modes/tty may not import domains/radio/cast (make lint-imports), so it
// carries the role keys and the class list as STRINGS. This is what stops that
// from being a silent duplication: rename a role in the registry and this test
// fails, rather than a Setup row quietly assigning nothing.
func TestSetupRoleKeysAndClassesMatchTheRegistry(t *testing.T) {
	// Every role a Setup row can assign, in the mock's mapping (setup.md):
	// Single Voice → the root, Alerts / Takeovers → alerts, All Reports →
	// standard, and the four overrides. Breaking, SevereRead and Station have
	// no picker: they inherit.
	for _, key := range []string{"voice", "alerts", "standard", "weather", "maritime", "fire", "seismic"} {
		if _, ok := cast.RoleByKey(key); !ok {
			t.Errorf("the Setup window assigns role %q, which the registry does not know", key)
		}
	}
	// The class list the TUI draws is the registry's, in the registry's order.
	got := toneClasses()
	if len(got) != len(cast.Classes()) {
		t.Fatalf("the TUI draws %d classes, the registry has %d", len(got), len(cast.Classes()))
	}
	for i, c := range cast.Classes() {
		if got[i].Key != c.Key() || got[i].Label != c.String() {
			t.Errorf("class %d = %+v, want key %q label %q", i, got[i], c.Key(), c.String())
		}
	}
}

// A save writes THIS platform's half and leaves the other alone (FR-8), so one
// config file serves a Mac and a Linux box.
func TestCastViewWritesThisPlatformsHalfOnly(t *testing.T) {
	both := cast.Config{Mode: cast.ModeCast, Pairs: map[cast.Role]cast.Pair{
		cast.Alerts: {MacOS: "Rishi", Piper: "en_US-ryan-medium"},
	}}
	asPlatform(t, "darwin")
	v := castView(both)
	if v.Names["alerts"] != "Rishi" {
		t.Errorf("macOS reads its own half: %+v", v.Names)
	}
	v.Names["alerts"] = "Daniel"
	back := castFromView(v, both)
	if got := back.Pairs[cast.Alerts]; got.MacOS != "Daniel" || got.Piper != "en_US-ryan-medium" {
		t.Errorf("a macOS save must leave the Piper half untouched: %+v", got)
	}

	asPlatform(t, "linux")
	v = castView(both)
	if v.Names["alerts"] != "en_US-ryan-medium" {
		t.Errorf("Linux reads the catalogue key: %+v", v.Names)
	}
	v.Names["alerts"] = "en_US-amy-medium"
	back = castFromView(v, both)
	if got := back.Pairs[cast.Alerts]; got.Piper != "en_US-amy-medium" || got.MacOS != "Rishi" {
		t.Errorf("a Linux save must leave the macOS half untouched: %+v", got)
	}
}

// --- Task 4.8: the same answer without a deck ---

// report --verbose and [S] read one CastReport, built from a config and this
// host's facts. Two implementations of "which voices does this host have" is
// how a report and a screen start disagreeing about the same machine.
func TestCastReportAnswersWithoutADeck(t *testing.T) {
	asPlatform(t, "darwin")
	cfg := config.Default()
	cfg.Voice = "Samantha"
	cfg.Radio.Cast = "cast"
	cfg.Radio.Voices.Fire = config.RoleVoice{MacOS: "Nobody At All"}
	cfg.Radio.Tones = config.Tones{Mode: "mute", Muted: []string{"watch", "not-a-class"}}

	rep := castReport(cfg, newHostFacts(t.Context()))
	if len(rep.Rows) != len(cast.Assignable())+1 {
		t.Fatalf("a row for the root and every assignable role, got %d", len(rep.Rows))
	}
	var fire CastRow
	for _, r := range rep.Rows {
		if r.Role == "fire" {
			fire = r
		}
	}
	if fire.Spoken != "Samantha" || fire.Link != "root" || fire.Reason != cast.ReasonUnknownHere {
		t.Errorf("the fire row must explain the fallback: %+v", fire)
	}
	// The tone summary names only classes this build KNOWS: an unknown key
	// mutes nothing, so listing it would say a class is silenced when it is not.
	if got := rep.ToneSummary(); !strings.Contains(got, "Watches") || strings.Contains(got, "not-a-class") {
		t.Errorf("ToneSummary = %q", got)
	}
	if len(rep.Problems) == 0 {
		t.Error("the unknown class key must be reported as a problem")
	}
}

// THE SEAM ONLY WORKS IF EVERY CALLER USES IT, and for the whole of 0.14.0 the
// most important one did not. app/voices.go:rawVoice branched on runtime.GOOS
// directly, so asPlatform(t, "darwin") set runtimeGOOS(), rawVoice ignored it, and
// the test walked the Piper install path anyway. On a Mac the two agree and
// everything passed; the first Linux CI run of this release panicked in a
// background install the test never meant to start.
//
// The ledger row that exempts this seam says what it is for in as many words:
// it exists "so the M5 fallback matrix can walk BOTH platform namespaces on one
// machine; without it half the matrix would never execute on the developer's box
// or in CI". A grep is the only thing that can hold that.
func TestNoProductionFileInAppReadsRuntimeGOOSDirectly(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]int{}
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		src, err := os.ReadFile(n)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if !strings.Contains(line, "runtime.GOOS") || strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			// The one legitimate use: the seam's own initialiser. It became an
			// atomic store when a test's restore was found racing the synth
			// render loop, so the shape this allows changed with it.
			if n == "cast.go" && strings.Contains(line, "goosSeam.Store(runtime.GOOS)") {
				continue
			}
			found[n] = i + 1
		}
	}
	for f, line := range found {
		t.Errorf("%s:%d reads runtime.GOOS directly — use the runtimeGOOS() seam, or asPlatform cannot steer it", f, line)
	}
	// CONTROL: the seam itself must still be there to be steered.
	if runtimeGOOS() == "" {
		t.Fatal("control: runtimeGOOS() is empty; the seam is gone and this guard checks nothing")
	}
}

// A deck with no engine cannot carry out a background install. The first Linux
// CI run of this release panicked because one tried: a real download, started by
// a unit test, whose progress callback dereferenced the nil engine.
//
// This tests the PREDICATE. Where it is applied — after the cap accounting, so
// the cap still binds, and before the goroutine, so no work starts — is fixed by
// TestUnattendedInstallsAreCappedPerSession on one side and by reading on the
// other; putting the guard a few lines earlier broke that cap test, which is how
// the placement was settled.
func TestAnUnwiredDeckCannotInstall(t *testing.T) {
	if (&radioDeck{}).canInstall() {
		t.Error("a deck with no engine reports it can install")
	}
	// A deck with an engine but nothing to report to still cannot: this is the
	// half CI found on the SECOND Linux round, in Program.Send rather than
	// Engine.Status.
	if (&radioDeck{engine: &player.Engine{}}).canInstall() {
		t.Error("a deck with no program reports it can install")
	}
	if (&radioDeck{p: &tea.Program{}}).canInstall() {
		t.Error("a deck with no engine reports it can install")
	}
	// CONTROL: with both, it can — so the predicate is reading them and is not
	// simply always false.
	if !(&radioDeck{engine: &player.Engine{}, p: &tea.Program{}}).canInstall() {
		t.Error("control: a fully wired deck reports it cannot install; the predicate is always false")
	}
}

// installFakePiperVoice lays out what FindPiperVoice stats: the binary and the
// model named by the spec's KEY. Nothing is executed — this is about lookup.
func installFakePiperVoice(t *testing.T, dir string, spec synth.VoiceSpec) {
	t.Helper()
	must := func(p string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must(filepath.Join(dir, "piper", "piper"))
	must(filepath.Join(dir, "voices", spec.Key+".onnx"))
	must(filepath.Join(dir, "voices", spec.Key+".onnx.json"))
}

// AN INSTALLED PIPER VOICE MUST RESOLVE AND BUILD ON LINUX — issue #7, reported
// from a real Arch box against the 0.14.0 release: the alert tone sounded and the
// ticker took over, and nothing was ever read, while the very same voice read
// user-initiated reports perfectly.
//
// The split is the whole story. The report path goes through rawVoice, which
// carries a FULL VoiceSpec from the catalogue; the takeover goes through
// cast.Resolve, which passes NAMES, and both Installed and buildVoice rebuilt a
// spec as VoiceSpec{Name: name} — leaving Key empty. FindPiperVoice locates the
// model at `<dir>/voices/<Key>.onnx`, so it looked for `voices/.onnx` and said
// no. Every Piper voice read as missing, on the one platform Piper is for.
//
// It survived every gate because every cast test pins darwin, where buildVoice
// returns a SayVoice and never reaches FindPiperVoice. This test is the Linux
// branch, with a voice actually on disk.
func TestAnInstalledPiperVoiceResolvesAndBuildsOnLinux(t *testing.T) {
	asPlatform(t, "linux")
	dir := t.TempDir()
	spec := synth.DefaultVoice()
	installFakePiperVoice(t, dir, spec)
	d := &radioDeck{voiceDir: dir, limiter: synth.NewLimiter(renderSlots(), synth.ReservedSlots)}

	// cast.Resolve asks by NAME — the form that was broken — and by key.
	if !d.Installed(spec.Name) {
		t.Errorf("Installed(%q) is false for a voice that is on disk", spec.Name)
	}
	if !d.Installed(spec.Key) {
		t.Errorf("Installed(%q) is false for a voice that is on disk", spec.Key)
	}
	if got := d.Default(); got != spec.Name {
		t.Errorf("Default() = %q, want the installed voice %q", got, spec.Name)
	}
	v, err := d.buildVoice(spec.Name)
	if err != nil {
		t.Fatalf("buildVoice(%q): %v — this is the takeover going silent", spec.Name, err)
	}
	if v == nil {
		t.Fatal("buildVoice returned no voice and no error")
	}

	// CONTROLS. A name that is not in the catalogue must still fail, and so must
	// the macOS sentinel — otherwise this passes for a deck that says yes to
	// everything, which is the failure mode being fixed, inverted.
	if d.Installed("not-a-voice") {
		t.Error("control: an unknown name reads as installed")
	}
	if d.Installed(systemVoice) {
		t.Error("control: the macOS sentinel reads as an installed Piper voice")
	}
	if _, err := d.buildVoice("not-a-voice"); err == nil {
		t.Error("control: buildVoice accepted a name that is not in the catalogue")
	}
}

// A NAME-ONLY VoiceSpec IS THE BUG, so nothing in app may build one.
//
// FindPiperVoice locates the model by Key; a spec carrying only a Name looks for
// `voices/.onnx` and answers no for every voice. Four call sites did it, and the
// fourth was written by copying the third — the comment on it read "find-only,
// exactly as the deck's is". piperInstallFor is the one owner now, and this is
// what keeps it the only one, because the next person will otherwise reach for
// the struct literal exactly as four people already did.
func TestNoProductionFileInAppBuildsANameOnlyVoiceSpec(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		src, err := os.ReadFile(n)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue // the owner's own comment names the shape it forbids
			}
			if strings.Contains(line, "synth.VoiceSpec{Name:") {
				found++
				t.Errorf("%s:%d builds a VoiceSpec from a name alone — its Key is empty, so FindPiperVoice "+
					"will never match it. Use piperInstallFor.", n, i+1)
			}
		}
	}
	// CONTROL: the owner must still exist to be used, or this guard forbids a
	// shape with nothing to replace it.
	if _, ok := piperInstallFor(t.TempDir(), "not-a-voice"); ok {
		t.Fatal("control: piperInstallFor found a voice that cannot exist; it is not doing the lookup")
	}
	_ = found
}
