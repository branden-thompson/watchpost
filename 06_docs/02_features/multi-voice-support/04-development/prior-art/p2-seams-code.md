# P2 — the broadcast and narration seams (multi-voice-support, 0.14.0) — v2 after the plan review

> **PRIOR ART — not the plan.** A verbatim snapshot of this batch document *before* the PLAN artefacts were
> stripped to task shape (`AP-PLANCODE-01`, `../../06-key_learnings/retro-notes.md` RN-1). The artefact of
> record is `../p2-seams.md`. Read `README.md` in this folder first: it says which of these blocks were actually
> compiled and run, and lists the known defects (D-1 … D-9) not to copy forward.



```
Goal:         Every read speaks in its resolved voice (FR-3): sections of one broadcast by role with a voice-keyed
              cache and a Source-time hand-over (FR-5), the Station role (FR-1), the narration seam with the
              arbiter renamed the Station Director (FR-12), the tone as constants on a find-only alert path
              (FR-9), background pre-install, previews on the app ctx.
Architecture: plan.md §2.3–2.4; data-shape.md §5. P1 (limiter, Reports{}, cast, config) is the base.
Tech Stack:   Go 1.27 · the recVoice / markVoice fakes (synth) · the scriptVoice / fakeBreakingAudio fakes (app)
Branch:       feature/multi-voice-support
Gate:         go test ./domains/radio/... ./app -race -count=2; make verify; make p10; make pty-severe
```

Shape after the plan reviews and the red-team round: the tone presets land first (2.0); `render` resolves a
segment's voice ONCE and returns a `renderedSeg`; `SetVoice` is gone — the listener's Setup save drives a
`Recast` through the resolver (a hand-over at the spot reached), an install landing drives `Invalidate` (the
next segment); the hand-over's argument order is `(from, to)` everywhere; the hand-over line is rendered AHEAD of the
writer and cached per (from → to) — the writer only writes — and a failed line is reported, never fatal; the cache is bounded in bytes; the `[S]` rows are memoised per cast generation; find-only applies to
the alert path and the per-role resolver, the root's tune keeps its blocking install; the preview goes through
the limiter; the resident Piper backend is cut from 0.14.0 (Task 2.11 records the ruling).

## File map

```
MODIFY: domains/radio/synth/tone.go, tone_test.go     — 2.0: Preset, ToneRate, Classic(), PresetByName, AlertTone(p, rate)
MODIFY: domains/radio/synth/compose.go               — 2.1: Segment.Role, role tags, the tail key/text, tagged(), HandoffLine
MODIFY: domains/radio/synth/fire.go, seismic.go       — 2.1: tagged(segs, role)
MODIFY: domains/radio/synth/source.go                — 2.2/2.3: resolver, resGen, voice-keyed cache, renderedSeg.voice, spokenName, Source-time hand-over
MODIFY: domains/radio/synth/synth_test.go, seismic_test.go — pins + new tests; recVoice gains name/spokeLine/count
CREATE: domains/radio/script/scripts/handover/line.txt
MODIFY: domains/radio/script/script_test.go          — the report lists (two places) + the convention data map + the parts list
MODIFY: domains/radio/script/scripts/weather-radio/tail.txt
RENAME: app/narrate.go → app/director.go; app/narrate_test.go → app/director_test.go
MODIFY: app/ticker_test.go, app/severe_read_test.go  — fakes' signatures; pins
MODIFY: app/radio.go, app/voices.go                  — cast on the deck, resolveVoice, tonePCM/tone(class), render(role), installs, previews, Close
MODIFY: app/ticker.go, app/severe_read.go, app/dashboard.go
CREATE: app/radio_roles_test.go                      — fakeVoice, runtimeGOOS, contains helpers; the M1 and FR-7/9 tests
```

---

### Task 2.0 — the tone as parameters: `Preset`, `ToneRate`, `Classic()`, `AlertTone(p, rate)` (P1's `ToneRate` in `limit.go` is deleted here — one declaration)

**File:** `domains/radio/synth/tone_test.go` (RED — edit the two existing calls, add one test)

- line 12: `pcm := AlertTone(rate)` → `pcm := AlertTone(Classic(), rate)` (the body of `TestAlertToneIsThreePulsesThenAPause` is unchanged: 2800 ms, stereo, pulses, the 2 s tail).
- line 55: `if AlertTone(0) != nil || AlertTone(-1) != nil {` → `if AlertTone(Classic(), 0) != nil || AlertTone(Classic(), -1) != nil {`.

```go
func TestPresetByNameAndTheToneRate(t *testing.T) {
	if PresetByName("classic").Name != "classic" || PresetByName("nope").Name != "classic" {
		t.Fatal("classic by name; an unknown name is the classic preset")
	}
	if ToneRate != 22050 {
		t.Fatal("the tone rate is a constant (FR-9): the tone never resolves a voice for its rate")
	}
}
```

**File:** `domains/radio/synth/tone.go` (GREEN — replace lines 10–57)

```go
// ToneRate is the attention tones' sample rate — a constant, so the tone
// never resolves a voice (FR-9; every voice speaks at 22 050 Hz today and
// the engine resamples per clip).
const ToneRate = 22050

// Preset is one attention signal (tones.md, MVS-D-11): pulses of summed
// sines, or a linear sweep, with an optional exponential decay, then the 2 s
// pause before narration every preset shares (the takeover holds for the
// whole buffer — app/ticker.go — so the pause is part of the pacing).
type Preset struct {
	Name     string
	Freqs    []float64     // summed ÷ len(Freqs)
	Sweep    [2]float64    // f0 → f1 linear chirp per pulse; Freqs ignored when Sweep[1] > 0
	Pulses   int
	PulseDur time.Duration
	GapDur   time.Duration
	Decay    time.Duration // exponential τ within a pulse; 0 = none
	Amp      float64       // of full scale
}

const (
	alertTailDur = 2 * time.Second
	alertEnvDur  = 8 * time.Millisecond
)

// Classic is today's tone (0.12.0): three enveloped ~1 kHz pulses. A
// function, not a package variable (P10-06): a preset is built when asked.
func Classic() Preset {
	return Preset{Name: "classic", Freqs: []float64{1000}, Pulses: 3, PulseDur: 200 * time.Millisecond, GapDur: 100 * time.Millisecond, Amp: 0.45}
}

// Presets lists every preset the cast can name (P3 adds the other four).
func Presets() []Preset { return []Preset{Classic()} }

// PresetByName is the preset the cast names (cast.ToneName); an unknown
// name is the classic tone.
func PresetByName(name string) Preset {
	for _, p := range Presets() {
		if p.Name == name {
			return p
		}
	}
	return Classic()
}

// AlertTone renders a preset — its pulses then the 2 s pause — as 16-bit LE
// STEREO PCM at rate. Pure and cheap (~1 ms): rendered per takeover. A
// non-positive rate yields no tone.
func AlertTone(p Preset, rate int) []byte {
	if rate <= 0 || p.Pulses <= 0 || p.PulseDur <= 0 || p.GapDur < 0 { // an exported edge trusts nothing (P10-07)
		return nil
	}
	samples := func(d time.Duration) int { return int(d.Seconds() * float64(rate)) }
	pulseN, gapN, envN, tailN := samples(p.PulseDur), samples(p.GapDur), samples(alertEnvDur), samples(alertTailDur)
	out := make([]byte, 0, (p.Pulses*pulseN+max(0, p.Pulses-1)*gapN+tailN)*4)
	for k := 0; k < p.Pulses; k++ {
		out = appendPulse(out, p, rate, pulseN, envN)
		if k < p.Pulses-1 {
			out = out[:len(out)+gapN*4] // silence: the pre-sized buffer's zero bytes, no temporary slice
		}
	}
	return append(out, make([]byte, tailN*4)...)
}

// appendPulse renders one pulse of the preset (split from AlertTone, P10-04).
func appendPulse(out []byte, p Preset, rate, pulseN, envN int) []byte {
	phase := 0.0
	for i := 0; i < pulseN; i++ {
		env := 1.0
		if envN > 0 && i < envN {
			env = float64(i) / float64(envN) // attack
		} else if envN > 0 && i >= pulseN-envN {
			env = float64(pulseN-i) / float64(envN) // release
		}
		if p.Decay > 0 {
			env *= math.Exp(-float64(i) / float64(rate) / p.Decay.Seconds())
		}
		var s float64
		if p.Sweep[1] > 0 {
			f := p.Sweep[0] + (p.Sweep[1]-p.Sweep[0])*float64(i)/float64(pulseN)
			phase += 2 * math.Pi * f / float64(rate)
			s = math.Sin(phase)
		} else {
			for _, f := range p.Freqs {
				s += math.Sin(2 * math.Pi * f * float64(i) / float64(rate))
			}
			s /= float64(max(1, len(p.Freqs)))
		}
		v := uint16(int16(p.Amp * env * s * math.MaxInt16))
		out = binary.LittleEndian.AppendUint16(out, v)
		out = binary.LittleEndian.AppendUint16(out, v)
	}
	return out
}
```

**File:** `app/radio.go:377` — `tone := synth.AlertTone(v.Rate())` → `tone := synth.AlertTone(synth.Classic(), v.Rate())`
(the rest of `tone()` changes in Task 2.7). **Verify:** `go build ./... && go test ./domains/radio/synth -run 'Tone|Preset' -count=1`

---

### Task 2.1 — `Segment.Role`, the tags, the tail, `HandoffLine` (data-shape §5; FR-5)

**File:** `domains/radio/synth/synth_test.go` (RED — one new test; three pins updated)

```go
func TestComposeTagsEverySegmentWithItsRole(t *testing.T) {
	now := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	loc := snapshot.Location{Label: "Oceanside, CA", Alerts: []snapshot.Alert{{ID: "a1", Headline: "Heat Advisory", Description: "Hot."}}}
	fire := FireReport{Known: true, RadiusKm: 25, Sources: []string{"NOAA's Hazard Mapping System"}, State: snapshot.FireState{AsOf: now}}
	segs := std.Compose(loc, []Product{{ID: "p1", Type: "ZFP", Text: ".TONIGHT...Clear.\n\n$$"}}, now, true, Station{}, Reports{Fire: fire})
	roleOf := map[string]cast.Role{}
	for _, s := range segs {
		roleOf[strings.SplitN(s.Key, ":", 2)[0]] = s.Role
	}
	for prefix, role := range map[string]cast.Role{"lead": cast.Station, "lead-span": cast.Weather, "alert": cast.Weather, "ZFP": cast.Weather, "fire": cast.Fire, "tail": cast.Station} {
		if roleOf[prefix] != role {
			t.Errorf("%s segments carry %s, got %s", prefix, role, roleOf[prefix])
		}
	}
	if last := segs[len(segs)-1]; last.Key != "tail" || !strings.Contains(last.Text, VoiceToken) {
		t.Fatalf("the tail is keyed 'tail' (the voice lives in the cache key — the latent tail:{{voice}} bug) and keeps the token: %+v", last)
	}
}
```

Existing pins to update (their new values):
- `synth_test.go:43` (`TestComposeReadsLikeNWR`): the expected tail text becomes `"This is " + VoiceToken + " for Watchpost Weather Radio. You can choose your correspondents in Watchpost's Setup window."` (the wording delta is Task 2.5; land both edits together).
- `synth_test.go:48`: `segs[len(segs)-1].Key != "tail:Samantha"` → `!= "tail"`.
- `TestReportsAreSeparatedByAir` (`synth_test.go:599-603`): `segs[len(segs)-1].Key != "tail:Samantha"` → `!= "tail"`.
- `seismic_test.go:57`: `strings.HasPrefix(segs[i].Key, "tail:")` → `segs[i].Key == "tail"`.
- P1's `TestComposeReportsZeroValueIsTodaysBroadcast` already pins `HasPrefix(Key, "tail")` — unchanged.

**File:** `domains/radio/synth/compose.go` (GREEN)

```go
// Segment is one narrated unit; Key identifies its content so rendered
// audio can be cached across cycles (§5: keyed on product issuance). Role
// says which correspondent reads it (0.14.0): the Source resolves the voice
// at render, and the cache key carries that voice's name.
type Segment struct {
	Key   string
	Text  string
	Pause time.Duration // extra silence after the text, beyond the standard gap (UAT 112.3)
	Role  cast.Role
}
```

`Composer.Tail`'s `""` branch (`compose.go:136-138`) is dead once every caller passes `VoiceToken` — delete it.
In `Compose`: `Segment{Key: "lead:" + …, Text: notice, Pause: leadPause, Role: cast.Station}`; the span,
`wx:`, `alert:` and product segments get `Role: cast.Weather`; the tail becomes
`Segment{Key: "tail", Text: c.Tail(VoiceToken), Role: cast.Station}`. `Reports.Voice` is **deleted** (dead once the tail carries the token; `Sample()` takes its own name) together
with `segments()`'s `voiceName` parameter (`app/radio.go:294`, its caller `startSynth` passes `synth.VoiceToken`) and the
`Voice:` in every P1 call site (`synth_test.go`, `seismic_test.go`, `soak_test.go`, the P1 zero-value test). Append:

```go
// tagged sets one role on every segment (the reports build theirs in bulk).
func tagged(segs []Segment, role cast.Role) []Segment {
	for i := range segs {
		segs[i].Role = role
	}
	return segs
}

// HandoffLine is the scripted hand-over (FR-5): the script tree's
// handover/line, else the built-in Handoff — a broken override must never
// turn a change of correspondent into silence, so this is the
// one part that falls back to the built-in text instead of to "".
func (c Composer) HandoffLine(from, to string) string {
	if line := c.say("handover", "line", map[string]string{"To": to, "From": from, "Voice": to}); line != "" {
		return line
	}
	return builtinHandoff(from, to)
}
```

`fire.go:75` `return segs` → `return tagged(segs, cast.Fire)`; `seismic.go:61` `return segs` → `return tagged(segs, cast.Seismic)`.
Import `cast` in `compose.go`. **Verify:** `go test ./domains/radio/synth -run 'Compose|Tags|Air|NWR' -count=1 && make lint-imports`

---

### Task 2.2 — the Source resolves a voice per segment; the cache is voice-keyed (AX-8)

**File:** `domains/radio/synth/seismic_test.go` (RED — `recVoice` gains a name and two helpers; the field `said` stays)

```go
type recVoice struct {
	mu   sync.Mutex
	name string // "" = "rec"
	said []string
}

func (v *recVoice) Name() string {
	if v.name != "" {
		return v.name
	}
	return "rec"
}

// spokeLine reports an exact line; count how many times it was said.
func (v *recVoice) spokeLine(line string) bool { return v.count(line) > 0 }
func (v *recVoice) count(line string) int {
	v.mu.Lock()
	defer v.mu.Unlock()
	n := 0
	for _, s := range v.said {
		if s == line {
			n++
		}
	}
	return n
}
```

**File:** `domains/radio/synth/synth_test.go` (RED — append)

```go
// twoVoices resolves Weather to one voice and everything else to another.
func twoVoices(weather, other Voice) func(cast.Role) (Voice, error) {
	return func(r cast.Role) (Voice, error) {
		if r == cast.Weather {
			return weather, nil
		}
		return other, nil
	}
}

func TestSourceRendersEachSegmentInItsRolesVoice(t *testing.T) {
	alpha, bravo := &recVoice{name: "Alpha"}, &recVoice{name: "Bravo"}
	segs := []Segment{{Key: "wx", Text: "Sunny.", Role: cast.Weather}, {Key: "fire:x", Text: "No hotspots.", Role: cast.Fire}}
	src, err := NewSource(alpha, func(context.Context) ([]Segment, error) { return segs, nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	src.SetResolver(twoVoices(alpha, bravo))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = io.ReadAll(src.Open(ctx))
	if !alpha.spokeLine("Sunny.") || alpha.spokeLine("No hotspots.") || !bravo.spokeLine("No hotspots.") {
		t.Fatalf("weather in Alpha, fire in Bravo: alpha=%v bravo=%v", alpha.said, bravo.said)
	}
	if n, _ := src.Cached(); n != 3 { // both segments under their voices, plus Bravo's hand-over line (rendered ahead, cached per from → to)
		t.Fatalf("both segments cached, each under its voice: %d", n)
	}
}

func TestSourceCacheKeyCarriesTheVoice(t *testing.T) {
	alpha, bravo := &recVoice{name: "Alpha"}, &recVoice{name: "Bravo"}
	seg := Segment{Key: "tail", Text: "This is " + VoiceToken + ".", Role: cast.Station}
	src, _ := NewSource(alpha, func(context.Context) ([]Segment, error) { return []Segment{seg}, nil }, nil)
	if _, err := src.render(context.Background(), seg); err != nil {
		t.Fatal(err)
	}
	src.SetResolver(twoVoices(bravo, bravo))
	if _, err := src.render(context.Background(), seg); err != nil {
		t.Fatal(err)
	}
	if !alpha.spokeLine("This is Alpha.") || !bravo.spokeLine("This is Bravo.") {
		t.Fatalf("the same key renders once per voice, each naming itself: %v %v", alpha.said, bravo.said)
	}
}

func TestMarqueeAndAudioAgreeOnTheVoiceName(t *testing.T) {
	// FR-5: the marquee's {{voice}} and the audio's resolve to the same name per segment.
	alpha, bravo := &recVoice{name: "Alpha"}, &recVoice{name: "Bravo"}
	segs := []Segment{{Key: "wx", Text: "Sunny.", Role: cast.Weather}, {Key: "tail", Text: "This is " + VoiceToken + ".", Role: cast.Station}}
	var mu sync.Mutex
	var shown []string
	src, _ := NewSource(alpha, func(context.Context) ([]Segment, error) { return segs, nil }, func(s Segment, _ time.Duration) {
		mu.Lock()
		shown = append(shown, s.Text)
		mu.Unlock()
	})
	src.SetResolver(twoVoices(alpha, bravo))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = io.ReadAll(src.Open(ctx))
	mu.Lock()
	defer mu.Unlock()
	if !bravo.spokeLine("This is Bravo.") || !slices.Contains(shown, "This is Bravo.") || slices.Contains(shown, "This is Alpha.") {
		t.Fatalf("audio %v; marquee %v", bravo.said, shown)
	}
}


func TestSpokenNameIsPlainAndBounded(t *testing.T) {
	// NFR-6: a config-supplied name that reaches a voice (the discovery-trust
	// window) is Plain'd and capped before it is spoken or shown.
	v := &recVoice{name: "\x1b[31mRishi\nfoo" + strings.Repeat("x", 80)}
	got := spokenName(v)
	if strings.ContainsAny(got, "\x1b\n") || len([]rune(got)) > 48 || !strings.HasPrefix(got, "Rishi") {
		t.Fatalf("spokenName: %q", got)
	}
}
```

**File:** `domains/radio/synth/source.go` (GREEN — the struct, constructor, resolver, render; `current()`/`Rate()` follow the rename)

```go
type Source struct {
	next    func(ctx context.Context) ([]Segment, error) // the next cycle's segments
	onSeg   func(Segment, time.Duration)                 // narration text + its spoken length, for the marquee
	gap     time.Duration
	resolve func(cast.Role) (Voice, error)  // the cast (0.14.0): which voice reads a role; the root voice for every role until SetResolver
	handoff func(from, to string) string   // the scripted hand-over line (FR-5); the built-in Handoff by default

	mu         sync.Mutex
	voice      Voice             // the root voice: the stream's rate, and every role's voice until SetResolver
	resGen     uint64            // bumps on SetResolver / Invalidate / Recast; audio rendered under an older generation is re-resolved before it plays
	hardGen    uint64            // the resGen of the last Recast: only the listener's change hands over mid-segment
	lastSpoken string            // the voice that spoke last (full name) — a different next voice hands over (FR-5)
	spokenNow  string            // its spoken form (the hand-over's "from" for the next voice)
	bytes      int               // mono bytes resident in the cache (the byte bound)
	repeat     bool
	cache      map[string][]byte // voiceName + "\x00" + segment key -> rendered mono PCM
	order      []string
	err        error
}

// NewSource builds a source; onSeg may be nil. Until SetResolver, every role
// reads in voice (0.13.0 behaviour, FR-2).
func NewSource(voice Voice, next func(context.Context) ([]Segment, error), onSeg func(Segment, time.Duration)) (*Source, error) {
	if err := invariant.Check(voice != nil && next != nil, "synth: voice and segment provider are required"); err != nil {
		return nil, err
	}
	if onSeg == nil {
		onSeg = func(Segment, time.Duration) {}
	}
	s := &Source{voice: voice, next: next, onSeg: onSeg, gap: 400 * time.Millisecond, cache: map[string][]byte{}}
	s.resolve = func(cast.Role) (Voice, error) { return s.root(), nil }
	s.handoff = builtinHandoff
	return s, nil
}

// root is the stream's root voice (its rate is the stream's rate).
func (s *Source) root() Voice {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.voice
}

// Rate is the PCM rate (the root voice's — fixed for the stream's life).
func (s *Source) Rate() int { return s.root().Rate() }

// generations are the current resolution generation and the last hard one.
func (s *Source) generations() (soft, hard uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resGen, s.hardGen
}

// SetResolver installs the cast: how each role's voice is found. Every
// resolved voice speaks at the root's rate (Setup validates that at
// assignment). Audio rendered ahead is re-resolved before it plays.
func (s *Source) SetResolver(resolve func(cast.Role) (Voice, error)) {
	if err := invariant.Check(resolve != nil, "synth: a resolver is required"); err != nil {
		return
	}
	s.mu.Lock()
	s.resolve = resolve
	s.resGen++
	s.mu.Unlock()
}

// Invalidate re-resolves every role at the NEXT segment (a voice arrived).
func (s *Source) Invalidate() {
	s.mu.Lock()
	s.resGen++
	s.mu.Unlock()
}

// SetHandoff installs the scripted hand-over line (Composer.HandoffLine).
func (s *Source) SetHandoff(f func(from, to string) string) {
	if f == nil {
		return
	}
	s.mu.Lock()
	s.handoff = f
	s.mu.Unlock()
}

// warmHandoffs renders, in the background and cache-only, the hand-over line
// for every ordered pair of distinct voices the cast resolves to (≤ 9 roles →
// ≤ 72 lines of ~0.1 MB), so the first cycle's boundaries never wait on a
// line render — on Linux a Piper load per boundary (the R6 first-cycle case).
// Ordinary slots; ends with ctx.
func (s *Source) warmHandoffs(ctx context.Context) {
	voices := map[string]Voice{}
	for _, role := range cast.Assignable() {
		if v, _, err := s.voiceFor(Segment{Role: role}); err == nil {
			voices[v.Name()] = v
		}
	}
	for _, from := range voices {
		for _, to := range voices {
			if from.Name() == to.Name() || ctx.Err() != nil {
				continue
			}
			s.renderHandoff(ctx, to, spokenName(from), spokenName(to))
		}
	}
}

// Recast tells the Source the listener changed the cast (a Setup save): the
// running segment hands over at the spot reached (UAT 94's behaviour, now
// driven by the resolver), and audio rendered ahead is re-resolved. A
// background install landing uses Invalidate instead — it takes effect at
// the next segment, never mid-sentence.
func (s *Source) Recast() {
	s.mu.Lock()
	s.resGen++
	s.hardGen = s.resGen
	s.mu.Unlock()
}

// voiceFor resolves a segment's voice and the generation it was resolved under.
func (s *Source) voiceFor(seg Segment) (Voice, uint64, error) {
	s.mu.Lock()
	resolve, gen := s.resolve, s.resGen
	s.mu.Unlock()
	v, err := resolve(seg.Role)
	if err != nil {
		return nil, gen, fmt.Errorf("synth: no voice for %s: %w", seg.Role, err)
	}
	if v == nil {
		return nil, gen, fmt.Errorf("synth: no voice for %s: the resolver returned none", seg.Role)
	}
	return v, gen, nil
}

type renderedSeg struct {
	seg     Segment
	pcm     []byte
	voice   Voice  // the voice that rendered it
	spoken  string // its spoken name (spokenName), for the marquee and the hand-over
	gen     uint64 // the resolution generation it was resolved under
	line    string // the hand-over line, rendered AHEAD by renderLoop when the voice changed ("" = none)
	handoff []byte // its audio (nil with a line = it could not be rendered; play reports it)
}

// cacheKey scopes a segment's audio to the voice that rendered it (RS-4):
// the voice's full name (two engines that share a spoken name stay apart).
func cacheKey(voice, key string) string { return voice + "\x00" + key }

// spokenName is a voice's name as it may be spoken or shown: Plain, one
// line, at most 48 runes (NFR-6 — a voice's name is text the host reported,
// not the config's; the cap keeps Piper's stdin and the marquee to one line
// either way). A nameless voice (the macOS sentinel on modern systems)
// reads "your correspondent" (FR-5) — the sign-off is never blank.
func spokenName(v Voice) string {
	name := render.PlainLine(v.Name())
	if name == "" {
		return "your correspondent"
	}
	if r := []rune(name); len(r) > 48 {
		name = string(r[:48])
	}
	return name
}

// spokenFor resolves VoiceToken to a spoken name — the one owner for audio
// and marquee (they can never disagree, RS-4).
func spokenFor(text, spoken string) string { return strings.ReplaceAll(text, VoiceToken, spoken) }

// render voices a segment as mono PCM in its role's voice (cached by voice
// and key; play widens it) and returns it with the voice that spoke and the
// resolution generation — resolved ONCE per segment; play never resolves
// again unless a generation changes. The cache key carries the voice, so a
// render that straddled a resolution change is still right and is stored.
func (s *Source) render(ctx context.Context, seg Segment) (renderedSeg, error) {
	v, gen, err := s.voiceFor(seg)
	if err != nil {
		return renderedSeg{seg: seg, gen: gen}, err
	}
	r := renderedSeg{seg: seg, voice: v, spoken: spokenName(v), gen: gen}
	key := cacheKey(v.Name(), seg.Key)
	s.mu.Lock()
	if cached, ok := s.cache[key]; ok {
		s.mu.Unlock()
		r.pcm = cached
		return r, nil
	}
	s.mu.Unlock()
	pcm, err := s.say(ctx, v, spokenFor(seg.Text, r.spoken))
	if err != nil {
		return r, err
	}
	r.pcm = pcm
	s.mu.Lock()
	s.store(key, pcm) // the key carries the voice, so a render that straddled a resolution change is still right
	s.mu.Unlock()
	return r, nil
}

// store caches one rendering and evicts the oldest past the byte bound.
// Callers hold s.mu. A key already cached is left alone (no double count).
// The eviction loop is counter-bounded by the cache's own size (P10-02).
func (s *Source) store(key string, pcm []byte) {
	if _, dup := s.cache[key]; dup {
		return
	}
	s.cache[key] = pcm
	s.order = append(s.order, key)
	s.bytes += len(pcm)
	for n := len(s.order); n > 1 && s.bytes > maxCachedBytes; n-- {
		oldest := s.order[0]
		s.bytes -= len(s.cache[oldest])
		delete(s.cache, oldest)
		s.order = s.order[1:]
	}
}

// renderHandoff renders the hand-over line an incoming voice speaks — ahead
// of the writer, cached under the incoming voice per outgoing name (a line
// is deterministic per (from → to); ≤ roles² entries of ~0.1 MB). nil when
// it cannot be rendered: play reports it and the segment plays anyway.
func (s *Source) renderHandoff(ctx context.Context, to Voice, from, toSpoken string) (line string, pcm []byte) {
	s.mu.Lock()
	handoff := s.handoff
	s.mu.Unlock()
	line = handoff(from, toSpoken) // never under s.mu: a script override is a template exec
	key := cacheKey(to.Name(), "handoff:"+from+"\x00"+line) // the line text is in the key: a re-scripted hand-over never serves old audio
	s.mu.Lock()
	cached, ok := s.cache[key]
	s.mu.Unlock()
	if ok {
		return line, cached
	}
	mono, err := s.say(ctx, to, line)
	if err != nil {
		return line, nil
	}
	s.mu.Lock()
	s.store(key, mono)
	s.mu.Unlock()
	return line, mono
}

// maxCachedBytes bounds the rendered-audio cache in BYTES — the quantity the
// soak measures (DISCOVER L1-F13: ≤ 0.73 MB a mono segment; one voice's
// cycle ≈ 29 MB). 40 MB covers a cycle in one voice with room for a second
// correspondent's sections; the synth.pcm.cache gauge reports the live size.
const maxCachedBytes = 40 << 20
```

`renderLoop` (replacing `source.go:173-184`):

```go
		prevName, prevSpoken := "", "" // the voice of the segment emitted before this one
		for _, seg := range segs {
			r, err := s.render(ctx, seg)
			if err != nil {
				s.fail(err)
				return
			}
			if prevName != "" && prevName != r.voice.Name() { // a change of correspondent: the line is rendered here, not on the writer
				r.line, r.handoff = s.renderHandoff(ctx, r.voice, prevSpoken, r.spoken)
			}
			prevName, prevSpoken = r.voice.Name(), r.spoken
			select {
			case out <- r:
			case <-ctx.Done():
				return
			}
		}
```

`synth_test.go` gains the imports `flag`, `fmt`, `path/filepath`, `slices` and `domains/radio/cast` (the file has
none of them today). `TestSourceReportsARenderFailureInsteadOfCompleting` (`synth_test.go:505-540`) calls
`src2.SetVoice(brokenVoice{})` — rewrite that step as a resolver swap + `src2.Recast()` to the existing
`brokenVoice` fake (the same assertion: the stream ends with the reason). Delete `SetVoice` (the chooser that called it is retired — MVS-D-3; `deck.SetVoice` goes with it in Task 2.7;
`TestSetVoiceRefusesADifferentRate` at `synth_test.go:542` and the same-voice no-op pin go with it (and `synth_test.go:505-540`'s `SetVoice(brokenVoice{})` step becomes a resolver swap + `Recast`) — the rate
rule moves to the catalogue: every voice is 22 050 Hz today, a new-rate voice arrives with a resampling
decorator, backlog), `current()`, `prev`, `gen`, `handoff()` (the method), `spoken()` and the old
`const maxCached = 40` (`source.go:348` — replaced by `store` and `maxCachedBytes`); `Cached()` reports
`len(s.cache), s.bytes`. The hand-over's argument order is `(from, to)` **everywhere**: the exported
`Handoff(newName, oldName)` becomes the unexported `builtinHandoff(from, to string)` (the pin at `synth_test.go:400`
becomes `builtinHandoff("Alpha", "Bravo") == "This is Bravo, taking over for Alpha."`). `source.go` imports `cast` and `platform/render`
(both allowed: `render` is already a synth import via `script.Say`'s `render.Plain`). **Verify:**
`go test ./domains/radio/synth -run 'Source|Cache|Voice|Marquee|SpokenName' -race -count=2`

---

### Task 2.3 — the Source-time hand-over, as one `Say` (FR-5; FR-12)

**File:** `domains/radio/synth/synth_test.go` (RED — two new tests; one existing pin rewritten)

```go
func TestSourceHandsOverBetweenCorrespondentsOnce(t *testing.T) {
	alpha, bravo := &recVoice{name: "Alpha"}, &recVoice{name: "Bravo"}
	segs := []Segment{{Key: "wx", Text: "Sunny.", Role: cast.Weather}, {Key: "fire:x", Text: "No hotspots.", Role: cast.Fire}, {Key: "fire:y", Text: "Stay safe.", Role: cast.Fire}}
	src, _ := NewSource(alpha, func(context.Context) ([]Segment, error) { return segs, nil }, nil)
	src.SetResolver(twoVoices(alpha, bravo))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = io.ReadAll(src.Open(ctx))
	// The DEFAULT hand-over names the incoming voice first.
	if bravo.count("This is Bravo, taking over for Alpha.") != 1 || alpha.spokeLine("This is Bravo, taking over for Alpha.") {
		t.Fatalf("the incoming voice speaks the hand-over, once: bravo=%v alpha=%v", bravo.said, alpha.said)
	}
}

func TestSourceHandsOverWithTheInstalledLine(t *testing.T) {
	alpha, bravo := &recVoice{name: "Alpha"}, &recVoice{name: "Bravo"}
	segs := []Segment{{Key: "wx", Text: "Sunny.", Role: cast.Weather}, {Key: "fire:x", Text: "No hotspots.", Role: cast.Fire}}
	src, _ := NewSource(alpha, func(context.Context) ([]Segment, error) { return segs, nil }, nil)
	src.SetResolver(twoVoices(alpha, bravo))
	src.SetHandoff(func(from, to string) string { return "Over to " + to + " from " + from + "." })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = io.ReadAll(src.Open(ctx))
	if bravo.count("Over to Bravo from Alpha.") != 1 {
		t.Fatalf("bravo=%v", bravo.said)
	}
}
```

```go
func TestRecastHandsOverMidSegmentAsOneSay(t *testing.T) {
	// UAT 94 kept, driven by the resolver: the listener's Recast hands the
	// running segment over at the spot reached — the hand-over line and the
	// remainder are ONE render (FR-12: the pair existed to hide a model load).
	alpha, bravo := &markVoice{name: "Alpha", mark: 1000, msPerChar: 20}, &markVoice{name: "Bravo", mark: 2000, msPerChar: 5}
	long := Segment{Key: "zfp", Text: strings.Repeat("word ", 40), Role: cast.Weather}
	src, _ := NewSource(alpha, func(context.Context) ([]Segment, error) { return []Segment{long}, nil }, nil)
	var mu sync.Mutex
	current := Voice(alpha)
	src.SetResolver(func(cast.Role) (Voice, error) { mu.Lock(); defer mu.Unlock(); return current, nil })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r := src.Open(ctx)
	_, _ = io.ReadFull(r, make([]byte, 22050*4)) // one second of stereo: playback is under way
	mu.Lock()
	current = bravo
	mu.Unlock()
	src.Recast()
	_, _ = io.ReadAll(r)
	said := bravo.texts()
	if len(said) != 1 || !strings.HasPrefix(said[0], "This is Bravo, taking over for Alpha. word") {
		t.Fatalf("one Say carrying the hand-over and the remainder: %v", said)
	}
}

func TestInvalidateNeverHandsOverMidSegment(t *testing.T) {
	// A background install landing (Invalidate) takes effect at the next
	// segment — never a mid-sentence hand-over.
	alpha, bravo := &markVoice{name: "Alpha", mark: 1000, msPerChar: 20}, &markVoice{name: "Bravo", mark: 2000, msPerChar: 5}
	segs := []Segment{{Key: "one", Text: strings.Repeat("word ", 40), Role: cast.Weather}, {Key: "two", Text: "second", Role: cast.Weather}}
	src, _ := NewSource(alpha, func(context.Context) ([]Segment, error) { return segs, nil }, nil)
	var mu sync.Mutex
	current := Voice(alpha)
	src.SetResolver(func(cast.Role) (Voice, error) { mu.Lock(); defer mu.Unlock(); return current, nil })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r := src.Open(ctx)
	_, _ = io.ReadFull(r, make([]byte, 22050*4))
	mu.Lock()
	current = bravo
	mu.Unlock()
	src.Invalidate()
	_, _ = io.ReadAll(r)
	// Bravo renders the hand-over line and "second" (render-ahead decides the
	// order, so membership, not index); never a remainder of "one".
	if said := bravo.texts(); len(said) != 2 || !slices.Contains(said, "second") || slices.ContainsFunc(said, func(t string) bool { return strings.HasPrefix(t, "word") }) {
		t.Fatalf("bravo: %v", said)
	}
}

func TestWriterNeverStarvesAcrossHandOvers(t *testing.T) {
	// R6 on the pipe: with a voice that takes 300 ms per Say and a cycle that
	// changes correspondent on every segment, the gap between successive
	// reads never exceeds the output buffer's slack (~0.7 s) once the first
	// segment is out — the hand-over lines are rendered ahead, never on the
	// writer (perf-protocol §1, the writer budget).
	alpha, bravo := &slowSayVoice{name: "Alpha", delay: 300 * time.Millisecond}, &slowSayVoice{name: "Bravo", delay: 300 * time.Millisecond}
	var segs []Segment
	for i := 0; i < 7; i++ {
		role := cast.Weather
		if i%2 == 1 {
			role = cast.Fire
		}
		segs = append(segs, Segment{Key: fmt.Sprintf("s%d", i), Text: "A short sentence.", Role: role})
	}
	src, _ := NewSource(alpha, func(context.Context) ([]Segment, error) { return segs, nil }, nil)
	src.SetResolver(twoVoices(alpha, bravo))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r := src.Open(ctx)
	// The reader is PACED at real time — an io.Pipe has no slack of its own, so
	// an unpaced loop would measure render-ahead latency, not the writer's gap.
	chunk := make([]byte, 22050*4/10) // 100 ms of stereo
	if _, err := io.ReadFull(r, chunk); err != nil {
		t.Fatal(err)
	}
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	last, worst := time.Now(), time.Duration(0)
	for range tick.C {
		if _, err := io.ReadFull(r, chunk); err != nil {
			break
		}
		now := time.Now()
		worst = max(worst, now.Sub(last))
		last = now
	}
	if worst > 700*time.Millisecond {
		t.Fatalf("the writer starved for %s (budget 700 ms)", worst)
	}
	if n := alpha.writerSays.Load() + bravo.writerSays.Load(); n != 0 { // the property itself: no Say ran on the writer goroutine
		t.Fatalf("%d Says ran on the writer; the hand-over lines are rendered ahead", n)
	}
}

// slowSayVoice takes a fixed time per Say, returns a second of silence, and
// counts the Says made on the writer goroutine (OnWriter).
type slowSayVoice struct {
	name       string
	delay      time.Duration
	writerSays atomic.Int64
}

func (v *slowSayVoice) Name() string { return v.name }
func (v *slowSayVoice) Rate() int    { return 22050 }
func (v *slowSayVoice) Say(ctx context.Context, _ string) ([]byte, error) {
	if OnWriter(ctx) {
		v.writerSays.Add(1)
	}
	select {
	case <-time.After(v.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return make([]byte, 22050*2), nil
}

func TestInvalidateNeverRendersOnTheWriter(t *testing.T) {
	// A background change (Invalidate) re-resolves the NEXT segment on the
	// render-ahead goroutine; the one already rendered plays as it is. Zero
	// writer Says; a Recast (the listener's act) is the one exception.
	alpha, bravo := &slowSayVoice{name: "Alpha", delay: 200 * time.Millisecond}, &slowSayVoice{name: "Bravo", delay: 200 * time.Millisecond}
	segs := []Segment{{Key: "a", Text: "one", Role: cast.Weather}, {Key: "b", Text: "two", Role: cast.Weather}, {Key: "c", Text: "three", Role: cast.Weather}}
	src, _ := NewSource(alpha, func(context.Context) ([]Segment, error) { return segs, nil }, nil)
	var mu sync.Mutex
	current := Voice(alpha)
	src.SetResolver(func(cast.Role) (Voice, error) { mu.Lock(); defer mu.Unlock(); return current, nil })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	r := src.Open(ctx)
	_, _ = io.ReadFull(r, make([]byte, 22050*4/10))
	mu.Lock()
	current = bravo
	mu.Unlock()
	src.Invalidate()
	_, _ = io.ReadAll(r)
	if n := alpha.writerSays.Load() + bravo.writerSays.Load(); n != 0 {
		t.Fatalf("%d Says on the writer after an Invalidate", n)
	}
}

func TestAnnounceFailureIsNotFatal(t *testing.T) {
	// A hand-over line that cannot render is reported on the marquee; the segment still plays.
	alpha := &recVoice{name: "Alpha"}
	flaky := &flakyVoice{recVoice: recVoice{name: "Bravo"}, failOn: "This is Bravo, taking over for Alpha."}
	segs := []Segment{{Key: "wx", Text: "Sunny.", Role: cast.Weather}, {Key: "fire:x", Text: "No hotspots.", Role: cast.Fire}}
	var mu sync.Mutex
	var shown []string
	src, _ := NewSource(alpha, func(context.Context) ([]Segment, error) { return segs, nil }, func(s Segment, _ time.Duration) { mu.Lock(); shown = append(shown, s.Key); mu.Unlock() })
	src.SetResolver(twoVoices(alpha, flaky))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = io.ReadAll(src.Open(ctx))
	mu.Lock()
	defer mu.Unlock()
	if src.Err() != nil || !slices.Contains(shown, "handoff:failed") || !slices.Contains(shown, "fire:x") { // the line could not render: reported, the fire segment still played
		t.Fatalf("err=%v shown=%v", src.Err(), shown)
	}
}

// flakyVoice fails one exact line and speaks everything else.
type flakyVoice struct {
	recVoice
	failOn string
}

func (v *flakyVoice) Say(ctx context.Context, text string) ([]byte, error) {
	if strings.HasPrefix(text, v.failOn) { // a prefix: the combined hand-over+remainder line matches too
		return nil, errors.New("flaky")
	}
	return v.recVoice.Say(ctx, text)
}
```

(`markVoice` is `synth_test.go`'s existing timing voice — the one `TestSetVoiceHandsOverMidSegmentAtTheSameSpot`
uses; `recVoice` its recording voice; `slices`/`errors` imported.)

Rewrite the existing `TestSetVoiceHandsOverMidSegmentAtTheSameSpot` (`synth_test.go:410-495`) as
`TestRecastHandsOverAtTheSameSpot` — the switch is `current = bravo; src.Recast()` through a resolver (as
above) instead of `SetVoice`; with the hand-over line and the remainder rendered as ONE utterance, Bravo renders **three** things — the combined
`"This is Bravo, taking over for Alpha. <remainder>"`, the next segment (re-voiced) and the tail — and the
marquee follows **four** entries: the line, the remainder, the next segment, the substituted tail:

```go
	said := b.texts()
	handoff := "This is Bravo, taking over for Alpha. "
	seen, remainder := map[string]int{}, ""
	for _, s := range said {
		switch {
		case s == "second segment here", s == "This is Bravo signing off.":
			seen[s]++
		case strings.HasPrefix(s, handoff):
			remainder = strings.TrimPrefix(s, handoff)
		}
	}
	if len(said) != 3 || len(seen) != 2 || remainder == "" {
		t.Fatalf("Bravo renders hand-over+remainder (one utterance), next segment, tail: %q", said)
	}
	// (keep the existing Remainder/`at` assertions from synth_test.go:470-482 here, verbatim)
	if len(spoken) != 4 || spoken[0] != long || spoken[1] != remainder || spoken[2] != "second segment here" || spoken[3] != "This is Bravo signing off." {
		t.Fatalf("the marquee follows: line, remainder, next, substituted tail — got %q", spoken)
	}
```

**File:** `domains/radio/synth/source.go` (GREEN)

```go
// play streams one rendered segment to the pipe in 100 ms chunks, then the
// inter-segment gap. Before it starts, a change of correspondent — this
// segment's voice is not the one that spoke last — is announced by the
// incoming voice with the scripted hand-over line (FR-5). Between chunks it
// watches for a resolution change (UAT 94: the root voice changed): the new
// voice opens with the hand-over line and the remainder in ONE render
// (FR-12), then playback continues there. A segment rendered ahead under an
// older resolution is re-rendered whole before it plays.
func (s *Source) play(ctx context.Context, pw *io.PipeWriter, r renderedSeg) bool {
	ctx = onWriter(ctx) // every Say made here is the writer's: the starvation test counts them
	if soft, hard := s.generations(); soft != r.gen {
		if hard > r.gen { // the LISTENER re-cast since this was rendered: the one render the writer waits for (a deliberate act); a failure plays the audio in hand
			if nr, err := s.render(ctx, r.seg); err == nil {
				r = nr
			} else {
				s.onSeg(Segment{Key: "handoff:failed", Text: "(re-cast skipped: " + render.PlainLine(err.Error()) + ")", Role: r.seg.Role}, 0)
				r.gen = hard
			}
		} else {
			r.gen = soft // a background change (an install or discovery landing): this segment plays as rendered; the NEXT one is re-resolved — never mid-sentence, never a writer render
		}
	}
	if last, lastSpoken := s.setLastSpoken(r.voice.Name(), r.spoken); last != "" && last != r.voice.Name() {
		if r.line == "" { // the change was not visible to renderLoop (a Recast landed since): once per (from → to), then the cache answers
			r.line, r.handoff = s.renderHandoff(ctx, r.voice, lastSpoken, r.spoken)
		}
		s.announce(pw, r, lastSpoken)
	}
	pcm := monoToStereo(r.pcm)
	text := spokenFor(r.seg.Text, r.spoken)
	s.onSeg(Segment{Key: r.seg.Key, Text: text, Role: r.seg.Role}, s.duration(pcm))
	chunk := s.Rate() / 10 * 4
	for written := 0; written < len(pcm); {
		if _, hard := s.generations(); hard > r.gen { // the LISTENER changed the cast (Recast): hand over at the spot reached (UAT 94)
			nv, gen, err := s.voiceFor(r.seg) // resolve only — never a render on the writer
			if err != nil {
				return s.fail(err)
			}
			r.gen = gen
			if nv.Name() != r.voice.Name() {
				if rest := Remainder(text, float64(written)/float64(len(pcm))); rest != "" {
					if next, ok := s.handOver(ctx, r.seg, r.spoken, nv, rest); ok { // one Say (FR-12) — the listener's own action, the one render the writer waits for
						text, pcm, written = rest, next, 0
						r.voice, r.spoken = nv, spokenName(nv)
						s.setLastSpoken(r.voice.Name(), r.spoken)
					}
				}
			}
			continue
		}
		n := min(chunk, len(pcm)-written)
		if _, err := pw.Write(pcm[written : written+n]); err != nil {
			return false
		}
		written += n
	}
	_, err := pw.Write(s.silence(s.gap + r.seg.Pause))
	return err == nil
}

// setLastSpoken records who speaks now (full name + spoken form) and returns
// who spoke before (name and spoken form).
func (s *Source) setLastSpoken(voice, spoken string) (prev, prevSpoken string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev, prevSpoken = s.lastSpoken, s.spokenNow
	s.lastSpoken, s.spokenNow = voice, spoken
	return prev, prevSpoken
}

// writerKey marks a context as the writer goroutine's (tests count the Says
// made under it: zero across hand-overs, one per deliberate Recast).
type writerKey struct{}

func onWriter(ctx context.Context) context.Context { return context.WithValue(ctx, writerKey{}, true) }

// OnWriter reports whether ctx is the writer's — for test voices.
func OnWriter(ctx context.Context) bool { v, _ := ctx.Value(writerKey{}).(bool); return v }

// announce writes the hand-over audio renderLoop (or play, after a Recast)
// rendered — the writer never renders here. A line that could not be
// rendered is reported on the marquee and the segment plays anyway: FR-5's
// "never silence" is better served by continuing than by ending a working
// broadcast.
func (s *Source) announce(pw *io.PipeWriter, r renderedSeg, from string) {
	if r.handoff == nil {
		s.onSeg(Segment{Key: "handoff:failed", Text: "(hand-over skipped)", Role: r.seg.Role}, 0)
		return
	}
	pcm := monoToStereo(r.handoff)
	s.onSeg(Segment{Key: "handoff:" + from + ":" + r.spoken, Text: r.line, Role: r.seg.Role}, s.duration(pcm))
	_, _ = pw.Write(pcm)
}

// handOver renders the hand-over line AND the remainder as one utterance in
// the new voice (FR-12: the pair existed to hide a model load), speaks it,
// and returns the audio to continue with; the marquee shows the remainder.
// A render failure is reported and the old voice's audio continues (ok =
// false) — never the end of the broadcast.
func (s *Source) handOver(ctx context.Context, seg Segment, from string, to Voice, rest string) ([]byte, bool) {
	s.mu.Lock()
	handoff := s.handoff
	s.mu.Unlock()
	line := handoff(from, spokenName(to)) // from is a spoken form; never under s.mu
	mono, err := s.say(ctx, to, line+" "+rest)
	if err != nil {
		s.onSeg(Segment{Key: "handoff:failed", Text: "(hand-over skipped: " + render.PlainLine(err.Error()) + ")", Role: seg.Role}, 0)
		return nil, false
	}
	pcm := monoToStereo(mono)
	s.onSeg(Segment{Key: seg.Key, Text: rest, Role: seg.Role}, s.duration(pcm))
	return pcm, true // the caller writes it chunk by chunk
}
```

(The hand-over's "from" travels on `renderedSeg.from` and the `setLastSpoken` return; no `prevSpoken` field.) Add a `flakyVoice` case to
`TestAnnounceFailureIsNotFatal`'s sibling: `TestHandOverFailureKeepsTheOldVoice` — a Recast to a voice whose
combined line fails: Alpha's audio plays to the end, the marquee shows `handoff:failed`, `src.Err() == nil`.

`play` is ≤ 60 lines and ≤ 15 decision points as written; keep it so. **Verify:**
`go test ./domains/radio/synth -run 'HandsOver|MidSegment|Remainder' -race -count=3`

---

### Task 2.4 — the hand-over script part (MVS-D-6; RAT-4)

**File:** `domains/radio/script/scripts/handover/line.txt`

```
{{/* handover/line: the incoming correspondent's first words when the voice changes between sections. Data: .To (the incoming voice), .From (the outgoing voice), .Voice (= .To, for older overrides). */}}
This is {{.To}}, taking over for {{.From}}.
```

**File:** `domains/radio/script/script_test.go` — three edits: (1) line 39's report list →
`"breaking,event-report,fire-report,global,handover,seismic-report,voice-preview,weather-radio"`; (2) line 114's list
→ `"breaking,event-report,fire-report,global,handover,my-report,seismic-report,voice-preview,weather-radio"`; (3) the
convention test's data map (`:20-26`) gains `"To": "Bravo"` only — `"From"` already exists at `:22` (a duplicate key is a compile error, PR2-4); P3 adds the maritime keys; (4)
`TestTheAppsPartsExist` (`:54-56`) gains `"handover/line"`.

**File:** `domains/radio/synth/synth_test.go` (append)

```go
func TestHandoffLineFallsBackToTheBuiltInNeverSilence(t *testing.T) {
	if got := std.HandoffLine("Alpha", "Bravo"); got != "This is Bravo, taking over for Alpha." {
		t.Fatalf("built-in handover/line: %q", got)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "handover"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "handover", "line.txt"), []byte("{{/* broken */}}\n{{.Nope}}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	broken := Composer{Scripts: script.New(dir)} // an override tree with a broken part (missingkey=error → "")
	if got := broken.HandoffLine("Alpha", "Bravo"); got != "This is Bravo, taking over for Alpha." {
		t.Fatalf("a broken override falls back to the built-in line, never silence: %q", got)
	}
}
```

(imports `os`, `path/filepath`, `script` in the test file if absent.) **Verify:** `go test ./domains/radio/... -count=1`

---

### Task 2.5 — the tail script (FR-2 deliberate delta; RS-7)

**File:** `domains/radio/script/scripts/weather-radio/tail.txt`

```
{{/* weather-radio/tail: the sign-off. Data: .Voice (the correspondent voice name; "your correspondent" when unnamed). */}}
This is {{.Voice}} for Watchpost Weather Radio. You can choose your correspondents in Watchpost's Setup window.
```

The pin at `synth_test.go:43` was already rewritten in Task 2.1 to the new wording. **Verify:**
`go test ./domains/radio/synth -run 'NWR|Tail' -count=1`

---

### Task 2.5b — the composed-segment golden (FR-2's three deltas)

**File:** `domains/radio/synth/synth_test.go` (append)

```go
// FR-2: with no role assigned the composed cycle is 0.13.0's — order, text and
// pauses — except the three deliberate deltas (the tail's key, its token, its
// wording). The golden pins the list; re-record only for a ruled change.
func TestComposedCycleGolden(t *testing.T) {
	now := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	loc := snapshot.Location{Label: "Oceanside, CA"}
	segs := std.Compose(loc, []Product{{ID: "p1", Type: "ZFP", Text: ".TONIGHT...Mostly clear.\n\n$$"}}, now, true, Station{Callsign: "KEC62", Site: "San Diego", State: "CA", FreqMHz: "162.400"}, Reports{})
	var b strings.Builder
	for _, s := range segs {
		fmt.Fprintf(&b, "%s|%s|%s|%s\n", s.Key, s.Role, s.Pause, s.Text)
	}
	golden := filepath.Join("testdata", "cycle.golden")
	if *updateGolden {
		_ = os.WriteFile(golden, []byte(b.String()), 0o600)
	}
	want, err := os.ReadFile(golden)
	if err != nil || b.String() != string(want) {
		t.Fatalf("composed cycle differs from the golden (run with -update-golden to re-record after a ruled change):\n%s", b.String())
	}
}
```

with `var updateGolden = flag.Bool("update-golden", false, "re-record goldens")` in the test file (the tree's
flag name, `modes/tty/golden_test.go:19`) and the golden recorded once at this task. **Verify:**
`go test ./domains/radio/synth -run Golden -update-golden && go test ./domains/radio/synth -run Golden`

---

### Task 2.6 — the Station Director: rename and `Run(ctx, class, role, audible, seq)` (FR-12; MVS-D-16)

`git mv app/narrate.go app/director.go && git mv app/narrate_test.go app/director_test.go`.

**File:** `app/director_test.go` (RED)

The three fakes change signature (the bodies otherwise as today):

```go
func (v *scriptVoice) tone(c cast.Class) time.Duration { v.rec("tone:" + c.String()); return v.dur }
func (v *scriptVoice) render(_ context.Context, r cast.Role, t string) (clip, bool) {
	return clip{text: t, dur: v.dur, role: r}, true
}
func (p *probeVoice) tone(cast.Class) time.Duration                              { return 0 }
func (p *probeVoice) render(_ context.Context, r cast.Role, t string) (clip, bool) { return clip{text: t, role: r}, true }
func (v *slowVoice) render(ctx context.Context, r cast.Role, t string) (clip, bool) — the existing body (`narrate_test.go:136-150`) with one edit: the returned clip gains `role: r`
```

Every `Run(ctx, narrateBreaking, …)` / `Run(ctx, narrateRead, …)` in the tests gains `cast.Breaking` /
`cast.SevereRead` after the class; every `s.attention()` becomes `s.attention(cast.Warning)` (`director_test.go:52,53`).
In `app/ticker_test.go`: `fakeBreakingAudio.tone(c cast.Class)` records `"tone:" + c.String()` and `render(_, r cast.Role, text)`;
the pin at `:116` `script[0] != "tone"` → `!= "tone:warning"` (the fixture event is a warning). New test:

```go
func TestDirectorForwardsTheRoleToRenderAndTheClassToTone(t *testing.T) {
	v := &scriptVoice{}
	d := newDirector(v)
	d.Run(context.Background(), narrateBreaking, cast.Breaking, true, func(ctx context.Context, s *speaker) {
		s.attention(cast.Storm)
		s.line("Hurricane Dolly has been reported.")
	})
	if !slices.Contains(v.calls, "tone:storm") || !slices.Contains(v.calls, "aside:Hurricane Dolly has been reported.") {
		t.Fatalf("calls: %v", v.calls)
	}
	var seen cast.Role
	pv := &probeVoice{onPlay: func(c clip) { seen = c.role }}
	newDirector(pv).Run(context.Background(), narrateRead, cast.SevereRead, true, func(ctx context.Context, s *speaker) { s.line("x") })
	if seen != cast.SevereRead {
		t.Fatalf("the role rides on the clip to play: %s", seen)
	}
}

```

**File:** `app/director.go` (GREEN — the full changed declarations; the arbiter's `admit/release/settle/unsuspend/innermostSuspended/first/remove` bodies are unchanged and stay in the file)

```go
// director.go — the Station Director (0.14.0, MVS-D-16; the 0.13.0 narrator
// renamed under the station metaphor): the one owner of who has the air.
// Every spoken sequence — a breaking takeover, an event read — runs through
// Run with a class and a ROLE; the Director serialises them, ducks the
// broadcast once for the whole run and restores it once at the end, lets a
// higher class take the air from a lower one (the lower is SUSPENDED and
// RESUMES afterwards), and forwards the role to the deck's render and the
// tone's class to the deck's tone — it never reads a voice itself (FR-3,
// FR-11). A Setup preview borrows its duck (Duck/Restore) without a job.
//
// Priorities (highest first):
//   narrateBreaking — always first, suspends a read
//   narrateRead     — ducks the broadcast, waits behind a takeover, is suspended by one

import (
	"context"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
)

type narrationClass int

const (
	narrateRead narrationClass = iota
	narrateBreaking
)

// clip is one rendered line: the audio, how long it plays, whether the
// visualizer follows it, and who spoke it.
type clip struct {
	text string
	pcm  []byte
	rate int
	dur  time.Duration
	viz  bool
	role cast.Role
}

func vizFor(class narrationClass) bool { return class != narrateBreaking }

// narrationVoice is what a sequence needs from the radio. The Director
// forwards the job's role to render and the tone's class to tone; the deck
// resolves them (cast.Resolve).
type narrationVoice interface {
	duck()
	tone(class cast.Class) time.Duration
	render(ctx context.Context, role cast.Role, text string) (clip, bool)
	play(c clip)
	pause()
	resume()
	discard()
	restore()
}

type narrationJob struct {
	class     narrationClass
	role      cast.Role
	audible   bool
	ctx       context.Context
	seq       uint64
	suspended bool
}

// director arbitrates the air.
type director struct {
	mu        sync.Mutex
	turn      *sync.Cond
	v         narrationVoice
	onAir     *narrationJob
	suspended []*narrationJob
	waiting   []*narrationJob
	ducked    bool
	next      uint64
	sleep     func(ctx context.Context, d time.Duration) bool
}

func newDirector(v narrationVoice) *director {
	n := &director{v: v, sleep: sleepCtx}
	n.turn = sync.NewCond(&n.mu)
	return n
}

func (n *director) silent() bool { return n == nil || n.v == nil }

type speaker struct {
	n     *director
	job   *narrationJob
	ctx   context.Context
	sleep func(ctx context.Context, d time.Duration) bool
}

const holdStep = 100 * time.Millisecond

// attention sounds the class's tone; 0 when silent, muted or ended.
func (s *speaker) attention(class cast.Class) time.Duration {
	if !s.live() {
		return 0
	}
	return s.n.v.tone(class)
}

// line renders and plays one line in the job's role's voice, returning how
// long it plays; 0 when silent, muted or ended. A line rendered while the
// job is suspended waits for the air and starts under the arbiter's lock.
func (s *speaker) line(text string) time.Duration {
	if !s.live() {
		return 0
	}
	c, ok := s.n.v.render(s.ctx, s.job.role, text)
	if !ok {
		return 0
	}
	c.viz = vizFor(s.job.class) // c.role is the deck's (render set it)
	if !s.awaitAir(func() { s.n.v.play(c) }) {
		return 0
	}
	return c.dur
}

// hold, awaitAir, live: carried by the git mv; no edit.

// Run runs seq as a narration of this class in this role: it waits for its
// turn, ducks the broadcast if audible, runs seq, restores when nothing
// waits. A breaking sequence's renders carry the reserved-slot priority
// (FR-12). False when ctx ended before the sequence finished.
func (n *director) Run(ctx context.Context, class narrationClass, role cast.Role, audible bool, seq func(ctx context.Context, s *speaker)) bool {
	if n == nil {
		seq(ctx, &speaker{ctx: ctx, sleep: sleepCtx})
		return ctx.Err() == nil
	}
	ctx = synth.WithPriority(ctx) // every Director job outranks render-ahead at the limiter (a takeover outranks a read here, by class)
	job := &narrationJob{class: class, role: role, audible: audible && !n.silent(), ctx: ctx}
	stop := context.AfterFunc(ctx, func() {
		n.mu.Lock()
		n.turn.Broadcast()
		n.mu.Unlock()
	})
	defer stop()
	if !n.admit(job) {
		return false
	}
	seq(ctx, &speaker{n: n, job: job, ctx: ctx, sleep: n.sleep})
	n.release(job)
	return ctx.Err() == nil
}

// Duck lets a Setup preview borrow the duck (MVS-D-16); Restore lifts it
// when nothing else holds the air.
func (n *director) Duck() {
	if n.silent() {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.ducked {
		n.v.duck()
		n.ducked = true
	}
}

func (n *director) Restore() {
	if n.silent() {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.onAir == nil && len(n.waiting) == 0 && len(n.suspended) == 0 && n.ducked {
		n.v.restore()
		n.ducked = false
	}
}
```

Rename by grep, not by list (`grep -rn 'narrat' app/ domains/ modes/ docs/`): every `narrator` identifier,
parameter (`nar` in `severe_read.go`, `ticker.go`), test name (`TestNarrator*` → `TestDirector*`) and comment in
`app/`, `domains/radio/player/engine.go`, `modes/tty/severe.go` becomes the Director; **keep** `narration*`,
`narrateRead`, `narrateBreaking`, `narrationVoice`, `narrationJob` (they name the thing spoken, not the
arbiter). The two production `Run` call sites gain their role in Task 2.9.
- **`docs/where-things-happen.md`** (pinned by `cmd/watchpost/docs_test.go:TestWhereThingsHappenNamesRealSymbols`
  — `make verify` fails at this gate otherwise): the rows naming `app/narrate.go:…` → `app/director.go:…`; row
  36's `app/voices.go:SetVoice` (deleted this batch) → `app/radio.go:resolveVoice`, and its
  `modes/tty/modal_chooser.go:handleVoiceKey` → `modes/tty/setup.go:handleSetupKey` (the chooser is deleted at
  P4; the row must not name it from this gate on); add rows "A role's voice is chosen —
  `domains/radio/cast/resolve.go:Resolve`, `app/radio.go:resolveVoice`" and "A correspondent hands over —
  `domains/radio/synth/source.go:renderHandoff`, `domains/radio/synth/source.go:announce`".
- **P10 ledger:** the two ratified symbol rows keyed `app/narrate.go` (`admit` P10-02, `awaitAir` P10-02; the
  density row is the package's) are re-keyed to `app/director.go` and presented for re-ratification at the P2
  gate. **The rename hazard is already handled:** git reports the move as `R099` and hides the unchanged
`admit`/`awaitAir` hunks, so without help the re-keyed rows read as *unmatched*. **P1 Task 1.0** taught
`scripts/quality/p10-unmatched.sh` to scope a renamed file's symbols against both paths — nothing to change
here; the re-key below simply relies on it, and the P2 gate is the first run that exercises it on a real rename.
- Re-capture `app/testdata/declset.txt` at the gate.
**Verify:** `go build ./... && go test ./app -run 'Director|Takeover|Read|Suspend' -race -count=2`

---

### Task 2.7 — the deck resolves by role; the tone from constants; find-only on the alert path; the root's tune keeps its install (FR-3, FR-7, FR-9, FR-11)

**File:** `app/radio_roles_test.go` (RED — with every helper it needs)

```go
package app

import (
	"context"
	"io"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
)

// fakeVoice records what it is asked to say (the app package's twin of synth's recVoice).
type fakeVoice struct {
	name  string
	mu    sync.Mutex
	lines []string
}

func (v *fakeVoice) Name() string { return v.name }
func (v *fakeVoice) Rate() int    { return 22050 }
func (v *fakeVoice) Say(_ context.Context, text string) ([]byte, error) {
	v.mu.Lock()
	v.lines = append(v.lines, text)
	v.mu.Unlock()
	return make([]byte, 22050/10*2), nil
}
func (v *fakeVoice) said(line string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, l := range v.lines {
		if l == line {
			return true
		}
	}
	return false
}

func runtimeGOOS() string { return runtime.GOOS }

func TestTonePCMNeverResolvesAVoice(t *testing.T) {
	d := &radioDeck{limiter: synth.NewLimiter(1, 1)} // no cast, no voices, no engine
	pcm := d.tonePCM(cast.Warning)
	if len(pcm) == 0 || pcmDuration(pcm, synth.ToneRate) < 2*time.Second {
		t.Fatal("the tone comes from constants alone (FR-9): the preset by class at synth.ToneRate")
	}
}

func TestResolveVoiceUsesTheCastAndRecordsTheResolution(t *testing.T) {
	if runtimeGOOS() != "darwin" {
		t.Skip("macOS names")
	}
	d := &radioDeck{limiter: synth.NewLimiter(1, 1), voices: []string{"Samantha", "Rishi"}, discovered: true}
	d.setCast(cast.Config{Root: "Samantha", CastMode: "cast", Pairs: map[cast.Role]cast.Pair{cast.Alerts: {MacOS: "Rishi"}}}) // the cast is in force (MVS-D-25)
	v, res, err := d.resolveVoice(cast.Breaking)
	if err != nil || v.Name() != "Rishi" || res.Link != cast.LinkGroup {
		t.Fatalf("breaking inherits Alerts = Rishi: %v %+v %v", v, res, err)
	}
	if got := d.Resolutions(); len(got) != 9 || got[0].Role != cast.Alerts {
		t.Fatalf("the cast table for [S]: %+v", got)
	}
}

func TestUnknownMacNameNeverReachesSay(t *testing.T) {
	if runtimeGOOS() != "darwin" {
		t.Skip("macOS names")
	}
	d := &radioDeck{limiter: synth.NewLimiter(1, 1), voices: []string{"Samantha"}, discovered: true}
	d.setCast(cast.Config{Root: "Samantha", CastMode: "cast", Pairs: map[cast.Role]cast.Pair{cast.Breaking: {MacOS: "Nobody"}}})
	v, res, err := d.resolveVoice(cast.Breaking)
	if err != nil || v.Name() != "Samantha" || res.Spoken != "Samantha" || res.Link != cast.LinkRoot {
		t.Fatalf("an unknown name falls back before any SayVoice is built: %v %+v %v", v, res, err)
	}
}

func TestResolveVoiceNeverTakesTheInstallMutex(t *testing.T) {
	d := &radioDeck{limiter: synth.NewLimiter(1, 1), voices: []string{"Samantha"}, discovered: true, voiceDir: t.TempDir()}
	d.setCast(cast.Config{Root: "Samantha"})
	d.installMu.Lock() // an install in progress
	defer d.installMu.Unlock()
	done := make(chan struct{})
	go func() { _, _, _ = d.resolveVoice(cast.Breaking); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("resolveVoice blocked on installMu — the alert path must be find-only (FR-9)")
	}
}
```

(`d.setCast` in these tests must not start the background installer on macOS: `ensureRoleVoices` returns at
once on darwin; on Linux the tests skip.)

**File:** `app/radio.go` (GREEN) — fields, and the narrationVoice implementation replacing `tone`/`render`:

```go
	cast       cast.Config      // the configured cast (data-shape.md); zero = one voice everywhere (FR-2)
	director   *director        // the Station Director, for the preview duck (set by attachRadio)
	ctx        context.Context  // the app's context: previews and installs end with the app; nil in tests
	castRows   []cast.Resolution // Resolutions() memoised until the cast or a host fact changes (nil = rebuild); never resolved per frame
	problems   []cast.Problem    // cast.Validate's findings for [S] (guarded by mu; rebuilt with castRows)
	onAir      cast.Resolution  // the last narration's resolution (guarded by mu; the [S] table reads it)
	installing map[string]bool  // Piper keys being installed in the background (guarded by mu) — one goroutine per key
	installFailed map[string]time.Time // Piper keys whose last install failed, and when: no retry before installRetryAfter (guarded by mu)
```

```go
// setCast installs the configured cast and re-resolves a running broadcast.
// LOCK DISCIPLINE (the deck as cast.Host): Discovered/Installed take d.mu,
// so cast.Resolve and cast.Validate — which call them — are NEVER invoked
// while d.mu is held. Snapshot under the lock, resolve outside, store
// under the lock (the shape Resolutions() uses).
func (d *radioDeck) setCast(c cast.Config) {
	problems := cast.Validate(c, d) // outside d.mu
	d.mu.Lock()
	d.cast = c
	d.castRows, d.problems = nil, problems
	clear(d.installFailed) // the listener's explicit act: a failed install may try again now
	src := d.source
	d.mu.Unlock()
	if src != nil {
		src.Recast() // the listener's change: hand over at the spot reached
	}
	go d.ensureRoleVoices()
}

// castChanged is the one tail every host-fact change runs — discovery
// landing, an install landing or failing: the [S] rows and problems
// rebuild, and a running broadcast re-resolves at its next segment (never
// mid-sentence, never on the writer).
func (d *radioDeck) castChanged() {
	d.mu.Lock()
	c, src := d.cast, d.source
	d.mu.Unlock()
	problems := cast.Validate(c, d) // outside d.mu
	d.mu.Lock()
	d.castRows, d.problems = nil, problems
	d.mu.Unlock()
	if src != nil {
		src.Invalidate()
	}
}

// Problems is cast.Validate's list for [S] (P4 maps it into ConfigNotes).
func (d *radioDeck) Problems() []cast.Problem {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]cast.Problem(nil), d.problems...)
}

// setTones changes only the tone mute (the [M] key, the Setup tone rows):
// no re-cast, no invalidation, no install pass.
func (d *radioDeck) setTones(t cast.Tones) {
	d.mu.Lock()
	d.cast.Tones = t
	d.mu.Unlock()
}

func (d *radioDeck) currentCast() cast.Config { // not castConfig(): that is the package func mapping config → cast (Task 1.12)
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.cast
}

// appCtx is the app's context, or Background in tests.
func (d *radioDeck) appCtx() context.Context {
	if d.ctx == nil {
		return context.Background()
	}
	return d.ctx
}

// resolveVoice is the one owner of "which voice reads this role" (AX-2):
// cast.Resolve over this deck's host facts, then the voice built from the
// resolution's Spoken name — find-only, never an install (FR-9). A Piper
// key the resolution wants is installed in the background, once.
func (d *radioDeck) resolveVoice(role cast.Role) (synth.Voice, cast.Resolution, error) {
	res := cast.Resolve(role, d.currentCast(), d)
	if key := res.WantsInstall(d); key != "" {
		d.installOnce(key)
	}
	if res.Spoken == "" {
		return nil, res, fmt.Errorf("no voice for %s on %s: install a Piper voice in Setup", role, runtime.GOOS)
	}
	v, err := d.buildVoice(res.Spoken)
	if err != nil {
		return nil, res, err
	}
	return d.limited(v), res, nil
}

// buildVoice makes a Voice for a resolved name: `say` on macOS (the sentinel
// is `say` with no -v), a found Piper install elsewhere — or its resident
// process when the knob is on. Never installs.
func (d *radioDeck) buildVoice(name string) (synth.Voice, error) {
	if runtime.GOOS == "darwin" {
		if name == cast.SystemVoice {
			name = ""
		}
		return synth.SayVoice{Voice: name}, nil
	}
	spec, ok := synth.VoiceByName(name)
	if !ok {
		return nil, fmt.Errorf("%q is not a catalogue voice", name)
	}
	inst, ok := synth.FindPiperVoice(d.voiceDir, spec)
	if !ok {
		return nil, fmt.Errorf("%q is not installed", name)
	}
	return synth.PiperVoice{Install: inst}, nil // a resident backend (AX-1 (C), 0.15.0) plugs in here
}

// Resolutions is the cast table for [S]: every assignable role's resolution,
// memoised until setCast or castChanged clears it — [S] is rendered per
// frame and a resolution costs os.Stat calls on Linux.
func (d *radioDeck) Resolutions() []cast.Resolution {
	d.mu.Lock()
	if d.castRows != nil {
		rows := append([]cast.Resolution(nil), d.castRows...)
		d.mu.Unlock()
		return rows
	}
	c := d.cast
	d.mu.Unlock()
	rows := make([]cast.Resolution, 0, 9)
	for _, r := range cast.Assignable() {
		rows = append(rows, cast.Resolve(r, c, d))
	}
	d.mu.Lock()
	d.castRows = rows
	d.mu.Unlock()
	return append([]cast.Resolution(nil), rows...)
}

// tonePCM is the class's preset rendered at the constant tone rate — from
// constants alone, never a voice (FR-9). Rendering is ~1 ms of arithmetic
// per takeover: no memo.
func (d *radioDeck) tonePCM(class cast.Class) []byte {
	return synth.AlertTone(synth.PresetByName(cast.ToneName(class)), synth.ToneRate)
}

// tone sounds the class's preset once, returning its length — nothing when
// the class is muted (MVS-D-26: [M] and the per-class set silence the tone,
// never the words).
func (d *radioDeck) tone(class cast.Class) time.Duration {
	if cast.Muted(class, d.currentCast().Tones) {
		return 0
	}
	pcm := d.tonePCM(class)
	_ = d.engine.PreviewAside(synth.ToneRate, bytes.NewReader(pcm)) // the attention tone never drives the bars
	return pcmDuration(pcm, synth.ToneRate)
}

// render turns a narration line into audio in the role's voice (the deck's
// limiter admits it; a breaking job's context carries the reserved slot).
func (d *radioDeck) render(ctx context.Context, role cast.Role, text string) (clip, bool) {
	v, res, err := d.resolveVoice(role) // the Limiter's bound is the one timeout on a render

	if err != nil {
		d.setDetail("no voice for " + role.String() + ": " + render.PlainLine(err.Error())) // a silent takeover leaves its reason on the player
		return clip{}, false
	}
	pcm, err := synth.AlertNarration(ctx, v, text)
	if err != nil {
		d.setDetail("voice cannot render: " + render.PlainLine(err.Error()))
		return clip{}, false
	}
	d.mu.Lock()
	d.onAir = res
	d.mu.Unlock()
	return clip{text: text, pcm: pcm, rate: v.Rate(), dur: pcmDuration(pcm, v.Rate()), role: role}, true
}
```

`newRadioDeck` sets `installing: map[string]bool{}, installFailed: map[string]time.Time{}` beside the limiter.
`tone()` logs `d.debugLog("tone:" + class.String())` (`debugLog` takes one line, `radio.go:550`) just before
`PreviewAside` — the live M4 instrument (the ticker's `breaking()` logs `"breaking:" + key` at entry, Task 2.9). `setDetail` is the existing on-air
detail setter the panel reads (`app/radio.go`'s `detail` field) — the tone still sounds; the reason is why the
words did not follow.

**File:** `app/voices.go` (GREEN) — `voice()` is the root's tune-path resolution, which KEEPS today's blocking
first-run install (UAT 118); only the Director's paths and the Source's per-role resolver are find-only.
`deck.SetVoice` (`voices.go:100-117`) is **deleted** with the chooser (Task 2.10 drops its caller; the root
voice is a Setup save → `recast`). `discoverVoices` runs `say -v ?` under `exec.CommandContext` with a 30 s
timeout (a wedged speech daemon must not hold the goroutine for the session); there is no trust window to close
— `Discovered()` answers with the curated list until the read lands (P1 Task 1.12):

```go
// voice is the broadcast's root voice for a tune — cast.All resolved on
// this host. Unlike the alert path (resolveVoice, find-only), a tune may
// install the root voice when nothing on this host speaks yet (UAT 118: the
// first-run download with progress in the player) — never under tuneMu.
func (d *radioDeck) voice() (synth.Voice, error) {
	v, res, err := d.resolveVoice(cast.All)
	if err == nil {
		return v, nil
	}
	key := res.WantsInstall(d)
	if key == "" && runtime.GOOS != "darwin" && res.Requested == "" {
		key = synth.DefaultVoice().Key // a fresh install names no voice: today's default is installed on first tune (UAT 118)
	}
	if runtime.GOOS == "darwin" || key == "" {
		return nil, err
	}
	if !synth.PiperSupported() {
		return nil, fmt.Errorf("no voice for %s/%s: install Piper or use a relayed location", runtime.GOOS, runtime.GOARCH)
	}
	spec, ok := synth.VoiceByName(key)
	if !ok {
		return nil, err
	}
	d.installMu.Lock()
	defer d.installMu.Unlock()
	if _, err := d.installVoice(spec, d.setDetail); err != nil {
		return nil, err
	}
	v, _, err = d.resolveVoice(cast.All)
	return v, err
}

// installOnce starts one background install per Piper key: the
// alert path never waits (FR-9); the takeover speaks in the fallback voice
// meanwhile and [S] reads "installing".
func (d *radioDeck) installOnce(key string) {
	d.mu.Lock()
	if d.installing == nil {
		d.installing, d.installFailed = map[string]bool{}, map[string]time.Time{} // a deck built in a test has no maps yet
	}
	if d.installing[key] || time.Since(d.installFailed[key]) < installRetryAfter { // one attempt per key at a time; a failed one waits (an offline host must not re-download 63 MB per segment)
		d.mu.Unlock()
		return
	}
	d.installing[key] = true
	d.mu.Unlock()
	go d.installInBackground(key)
}

func (d *radioDeck) installInBackground(key string) {
	defer func() {
		d.mu.Lock()
		delete(d.installing, key)
		d.mu.Unlock()
	}()
	spec, ok := synth.VoiceByName(key)
	if !ok {
		return
	}
	d.installMu.Lock()
	defer d.installMu.Unlock()
	if _, ok := synth.FindPiperVoice(d.voiceDir, spec); ok {
		return
	}
	if _, err := d.installVoice(spec, d.setDetail); err != nil {
		d.mu.Lock()
		d.installFailed[key] = time.Now()
		d.mu.Unlock()
		d.setDetail("voice install failed: " + render.PlainLine(err.Error()) + " — retrying after " + installRetryAfter.String())
		d.castChanged() // [S] reads "install failed — retries at hh:mm"
		return
	}
	d.mu.Lock()
	delete(d.installFailed, key)
	d.mu.Unlock()
	d.castChanged() // the assigned voice is here now: the next segment uses it (never mid-sentence)
}

// installRetryAfter is how long a failed background install waits before
// the next attempt (a Setup save resets it through setCast → ensureRoleVoices).
const installRetryAfter = 10 * time.Minute

// installState words a Piper key's background state for [S]: "installing",
// "install failed — retries at 15:04", or "".
func (d *radioDeck) installState(key string) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch {
	case d.installing[key]:
		return "installing"
	case !d.installFailed[key].IsZero():
		return "install failed — not before " + d.installFailed[key].Add(installRetryAfter).Format("15:04") + ", on the next alert or Setup save"
	}
	return ""
}

`TestSetCastNeverDeadlocksWithANamedPair` (darwin shape, 5 s timeout via a goroutine + `select`): a deck in cast
mode with `Alerts: {MacOS: "Rishi"}` runs `setCast`, `castChanged` and `Resolutions()` back to back — the test
fails on the timeout, never hangs the gate.

// ensureRoleVoices installs, in the background, every Piper voice the cast
// resolves to and this host lacks — the alert roles first (FR-9).
func (d *radioDeck) ensureRoleVoices() {
	if runtime.GOOS == "darwin" || !synth.PiperSupported() {
		return
	}
	for _, r := range cast.Assignable() { // the registry's list, one owner
		if key := cast.Resolve(r, d.currentCast(), d).WantsInstall(d); key != "" {
			d.installOnce(key)
		}
	}
}
```

`installVoice`: `context.WithTimeout(context.Background(), 15*time.Minute)` → `context.WithTimeout(d.appCtx(), 15*time.Minute)`.
`PreviewVoice`: `var v synth.Voice = synth.SayVoice{Voice: name}` and the Piper branch's `v = synth.PiperVoice{Install: inst}`
both become `v = d.limited(…)`; the Piper branch's unknown-name fallback `spec = d.piperSpec()` (`voices.go:129`) becomes
`spec, ok = synth.VoiceByName(cast.Resolve(cast.All, d.currentCast(), d).Spoken); if !ok { d.voiceNote(name + " is not a catalogue voice"); return }`;
the timeout context uses `d.appCtx()`; wrap the audition:
`d.director.Duck(); defer d.director.Restore()` before `d.engine.Audition` (`director` nil-safe via `silent()`).
`VoiceName()` becomes `return cast.Resolve(cast.All, d.currentCast(), d).Spoken` (the chooser's chip until P4
removes it). Delete `piperSpec` and `defaultVoice` (unused once resolution is the cast's; `systemVoice` = `cast.SystemVoice` — keep the
`cast` constant, delete the app one). **Existing pins in `app/radio_test.go` that read the deleted fields:**
`TestVoiceChipLabelIsTheChooserLabel` (`:88 d.voiceID = "Karen"`) → `d.setCast(cast.Config{Root: "Karen"}); d.voices, d.discovered = []string{"Karen"}, true` then
`VoiceName() == "Karen"`; `TestPiperVoiceChooserListsTheCatalogue` (`:107-115`) → keep the `discoverVoices()` assertion, replace the
`defaultVoice`/`piperSpec` lines with `d.voiceDir = t.TempDir(); if d.Installed("en_GB-alan-medium") { t.Fatal("nothing installed in a fresh dir") }; if _, err := d.buildVoice("Alan"); err == nil || !strings.Contains(err.Error(), "not installed") { t.Fatal(err) }`
(skip on darwin, where `buildVoice` never consults the dir). **Verify:** `go test ./app -run 'Tone|Resolve|Voice|InstallMutex' -race -count=2`

---

**One bound owner:** the Limiter's per-`Say` bound (P1 Task 1.1) is the only timeout on a render:
`PreviewVoice`'s own `context.WithTimeout` (`voices.go:142`) and the 2-minute wrap `render` carried go — a
wedged process is the Limiter's to end. **Previews (one owner — the deck):** the note the deck sends at once reads `preparing <name>… (the radio may be
rendering; a preview waits its turn)` — the TUI sets no note of its own; when the audition ends — played or failed — the deck
sends `SetupNoteMsg{""}` so the note never outlives it (a failure sends its reason first, PlainLine'd). The
tests `TestResolveVoice…` set `CastMode: "cast"` because single-voice mode ignores the pairs (MVS-D-25).

---

### Task 2.8 — the broadcast uses the cast: `startSynth` installs the resolver and the hand-over; the M1 test

**File:** `app/radio.go` — in `startSynth`, after `NewSource` succeeds:

```go
	src.SetResolver(func(r cast.Role) (synth.Voice, error) { v, _, err := d.resolveVoice(r); return v, err })
	src.SetHandoff(d.composer.HandoffLine)
```

and `d.engine.StartSource("Watchpost Synth ("+voice.Name()+")", …)` → `d.engine.StartSource("Watchpost Synth", src.Rate(), src.Open)`
(the stream's name no longer claims one voice — red-team risk 11).

**File:** `app/radio_roles_test.go` (append)

```go
func TestBroadcastSectionsSpeakInTheirRolesVoices(t *testing.T) {
	// M1 on the recording output: weather in the root voice, fire in the Fire role's.
	root, fire := &fakeVoice{name: "Root"}, &fakeVoice{name: "Fire"}
	segs := []synth.Segment{{Key: "wx", Text: "Sunny.", Role: cast.Weather}, {Key: "fire:x", Text: "No hotspots.", Role: cast.Fire}}
	src, err := synth.NewSource(root, func(context.Context) ([]synth.Segment, error) { return segs, nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	src.SetResolver(func(r cast.Role) (synth.Voice, error) {
		if r == cast.Fire {
			return fire, nil
		}
		return root, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = io.ReadAll(src.Open(ctx))
	if !root.said("Sunny.") || !fire.said("No hotspots.") || root.said("No hotspots.") || !slices.ContainsFunc(fire.lines, func(l string) bool { return strings.HasPrefix(l, "This is Fire, taking over for Root.") }) { // render-ahead orders the line and the segment; membership, not index
		t.Fatalf("root=%v fire=%v", root.lines, fire.lines)
	}
}
```

**Verify:** `go test ./app -run Broadcast -race -count=2`

---

### Task 2.9 — the ticker and the read pass their role and class (FR-11; MVS-D-12; the mute rule)

**File:** `app/ticker_test.go` (RED — append; a pure test, no ticker constructor needed)

```go
func TestToneClassIsTheHighestSeverityEventsClass(t *testing.T) {
	fresh := []globalfeed.Event{
		{Class: globalfeed.ClassSevereWx, Type: "Flood Watch", Severity: globalfeed.SevYellow},
		{Class: globalfeed.ClassSevereWx, Type: "Tornado Warning", Severity: globalfeed.SevRed},
	}
	byBreakingPriority(fresh)
	if got := toneClass(fresh[0]); got != cast.Warning {
		t.Fatalf("one tone, the highest-severity event's class (MVS-D-12): %s", got)
	}
	if toneClass(globalfeed.Event{Class: globalfeed.ClassTropical, Type: "Hurricane Dolly"}) != cast.Storm || toneClass(globalfeed.Event{Class: globalfeed.ClassQuake}) != cast.Disaster {
		t.Fatal("a tropical cyclone is the storm class; a quake is a disaster (MVS-D-28)")
	}
}
```

**File:** `app/ticker.go` (GREEN) — `breaking()`'s `audible` no longer folds `t.muted` (the mute is the deck's
`tone()`'s, per class — MVS-D-26); the existing `[M]` pin in `ticker_test.go` that expects a muted takeover to be
silent becomes "a muted takeover has no tone and still reads its line".

```go
	audible := !t.voice.silent()
	t.voice.Run(ctx, narrateBreaking, cast.Breaking, audible, func(ctx context.Context, s *speaker) {
		if !s.hold(s.attention(toneClass(fresh[0]))) { // byBreakingPriority put the highest-severity event first: one tone, its class (MVS-D-12)
			return
		}
```

`t.muted` (the `*atomic.Bool` from `tickerState`) now means "Tones.Mode == mute". Until P4 gives `[M]` its own
hook (Task 4.7), the 0.13.0 handler still persists `ticker_muted`; **Task 2.10 makes `tickerState`'s persist hook call
`deck.recast()` after `savePreference`** so `Load`'s `fromTickerMuted` (P1 Task 1.8) turns it into "mute every class"
and the deck's `tone()` honours it at once. A new `ticker_test.go` pin (none exists today — `muted` is only
ever constructed `false`) that a muted class (`grep -n muted ticker_test.go`) becomes: `f.script` contains no `tone:` entry **and** contains the line's
text — code: `if slices.Contains(f.script, "tone:warning") || !slices.Contains(f.script, line) { t.Fatalf("muted: no tone, the words read: %v", f.script) }`.

```go
// toneClass is the tone an event's takeover opens with (FR-11): the product
// string classified by the cast; a significant quake is a disaster, a cyclone a storm.
func toneClass(e globalfeed.Event) cast.Class {
	switch e.Class {
	case globalfeed.ClassQuake:
		return cast.Disaster
	case globalfeed.ClassTropical:
		return cast.Storm
	}
	return cast.Classify(e.Type)
}
```

**File:** `app/severe_read_test.go` (RED — append; the existing pattern at `:222-234`)

```go
func TestSevereReadSoundsItsClassTone(t *testing.T) {
	v := &scriptVoice{}
	reading := make(chan struct{}, 4)
	row := tty.SevereRow{Product: "Flood Watch", Location: "Oceanside, CA"}
	r := newEventReader(context.Background(), newDirector(v), nil, func(string) (tty.SevereRow, bool) { return row, true }, func(m tea.Msg) {
		if s, ok := m.(tty.SevereReadingMsg); ok && s.Key != "" {
			reading <- struct{}{}
		}
	})
	r.Read("k")
	<-reading
	waitUntil(t, "the read on air", func() bool { return strings.Contains(v.got(), "speak:") })
	r.End()
	if !slices.Contains(v.calls, "tone:watch") {
		t.Fatalf("a read opens with its class's tone: %v", v.calls)
	}
}

func TestMutedClassIsMutedInTheDecksCast(t *testing.T) { // the mute rule as the deck holds it; tone() itself needs the engine (the PTY smoke covers it)
	// MVS-D-26 at the deck: the tone is the mute's only subject.
	d := &radioDeck{limiter: synth.NewLimiter(1, 1)}
	d.setCast(cast.Config{Tones: cast.Tones{Mode: cast.MuteMode, Muted: []string{"watch"}}})
	if len(d.tonePCM(cast.Watch)) == 0 {
		t.Fatal("tonePCM is the preset regardless")
	}
	if cast.Muted(cast.Watch, d.currentCast().Tones) != true || cast.Muted(cast.Warning, d.currentCast().Tones) {
		t.Fatal("watch muted, warning not")
	}
}
```

(the deck's `tone(class)` returns 0 for a muted class — Task 2.7 — so `speaker.attention` holds nothing and
`line` reads as ever.)

(`newEventReader`'s real signature is `(ctx, dir *director, scripts *script.Library, row func(string) (tty.SevereRow, bool), send func(tea.Msg))` —
`severe_read.go:55`; `v.got()` exists on `scriptVoice`.)

**File:** `app/severe_read.go` (GREEN) — in `run` (no mute hook: the deck's `tone()` is silent for a muted class,
the words always read — MVS-D-26):

```go
	r.dir.Run(ctx, narrateRead, cast.SevereRead, true, func(ctx context.Context, s *speaker) {
		r.send(tty.SevereReadingMsg{Key: key})
		defer r.send(tty.SevereReadingMsg{})
		if !s.hold(s.attention(cast.Classify(row.Product))) { // FR-11: the read opens with its class's tone (0 when muted)
			return
		}
		dur := s.line(script)
		// (the rest of Read's body is the file's, untouched — the diff is the one line above)
```

`breaking()` logs `d.debugLog("breaking:" + key)` at entry — with `tone:<class>` (Task 2.7) the two timestamps
are the live M4 instrument (`perf-protocol.md` §1). **Verify:** `go test ./app -run 'ToneClass|SevereRead|Breaking' -race -count=2`

---

### Task 2.9b — `app/tone_latency_test.go`: the M4 regression pin (P0 wrote it on the 0.13.0 worktree; this lands it)

`radioDeck.engine` is a concrete `*player.Engine`, so the benchmark sits at the Director seam — the shape
`app/ticker_test.go`'s `fakeBreakingAudio` already has: a narration voice whose `tone()` stamps `time.Now()`.

```go
package app

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
)

// stampVoice is a narration voice that records when the tone was asked for.
type stampVoice struct {
	fakeBreakingAudio
	at atomic.Int64
}

func (v *stampVoice) tone(class cast.Class) time.Duration {
	v.at.Store(time.Now().UnixNano())
	return 0
}

// BenchmarkTimeToToneStart is the code-path pin for M4: from breaking()'s
// entry to the tone call, through the Director's admission. It measures Go,
// not audio — the +50 ms clause of the gate is the LIVE check
// (perf-protocol §1: the breaking:/tone: debug entries); this pin says the
// path did not grow. The same file runs on the 0.13.0 tree (P0 Task 0.1).
func BenchmarkTimeToToneStart(b *testing.B) {
	v := &stampVoice{}
	t := newTestTicker(b, v) // the ticker over a Director on v (2.9's helper)
	for i := 0; i < b.N; i++ {
		entry := time.Now()
		t.breaking(context.Background(), fixtureWarning())
		for v.at.Load() == 0 {
			time.Sleep(50 * time.Microsecond)
		}
		b.ReportMetric(float64(time.Duration(v.at.Load()-entry.UnixNano()).Microseconds()), "µs/tone")
		v.at.Store(0)
	}
}
```

`go test ./app -run '^$' -bench TimeToToneStart -count=10` (the `-run '^$'` keeps the other `app` benchmarks
out). **Verify:** the benchmark runs on both trees; the number goes into `perf-protocol.md` §0.

---

### Task 2.10 — `attachRadio` carries the cast; saves re-cast

**File:** `app/dashboard.go` — the signature and body:

```go
func attachRadio(ctx context.Context, model tty.Dashboard, client *httpx.Client, provider *nws.Provider, cfg config.Config, mode tty.RadioMode, fire func(snapshot.LocationRef) synth.FireReport, seismic func(snapshot.LocationRef) synth.SeismicReport) (*tea.Program, *radioDeck, func()) {
	deck := newRadioDeck(nil, client, provider, render.UnitF)
	if deck == nil {
		return tea.NewProgram(model), nil, func() {}
	}
	deck.pref, deck.fire, deck.seismic, deck.ctx = mode, fire, seismic, ctx
	deck.setCast(castConfig(cfg))
	deck.persistMode = saveRadioMode
	model = model.WithRadio(deck).WithRadioMode(mode).WithSpectrum(deck.Spectrum).WithVoices(deck.Voices, deck.VoiceName(), func(name string) error {
		if err := savePreference(func(cfg *config.Config) { cfg.Voice = name }); err != nil { // until P4 removes the chooser: the root voice, through the cast

			return err
		}
		return deck.recast()
	}, deck.PreviewVoice)
	p := tea.NewProgram(model)
	deck.p = p
	return p, deck, deck.Stop
}
```

(the call site at `dashboard.go:86` passes `ctx` and `cfg` instead of `cfg.Voice`.) After
`buildDirector()` runs (`dashboard.go:134`): `lp.deck.director = lp.director` (nil-safe when the deck is nil).
The `[M]` hook is built where `lp` is in scope — `RunDashboard`'s `ttyConfig` (`app/dashboard.go:152-171`),
not the package-level `tickerState` (which cannot reach the deck):

```go
	muteTicker := func(muted bool) { // 0.14.0: tones only (MVS-D-26); an empty set under "mute" = every class until P4's rows
		mode := ""
		if muted {
			mode = "mute"
		}
		if err := savePreference(func(c *config.Config) { c.Radio.Tones.Mode = mode }); err != nil { // Save mirrors ticker_muted (P1 1.8)
			lp.warn("mute: " + err.Error())
			return
		}
		if lp.deck != nil {
			lp.deck.setTones(cast.Tones{Mode: mode})
		}
		lp.ticker.muted.Store(muted) // the marquee's mark, as today
	}
```

(`lp.warn` is whatever the existing closure used to surface a save error — keep it.) No config reload, no
re-cast. The `ticker_test.go` pin on the persisted value reads `cfg.Radio.Tones.Mode`.

```go
// recast reloads the config and installs the cast — every voice/tone save
// calls it (Setup in P4, the chooser until then).
func (d *radioDeck) recast() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	d.setCast(castConfig(cfg))
	return nil
}

```

The `d.voiceID` field is deleted (the root is `cfg.Voice` via `recast`). **Verify:** `go build ./... && go test ./app -race -count=1`

---

### Task 2.11 — `ResidentPiper` — CUT from 0.14.0 (pending the HUM LEAD's confirmation at the PLAN exit)

Four independent lenses converged (Business, Code Quality, Principal Architect, Performance): a warm-process
backend shipped default-off, unrunnable on the developer's macOS, and waiting on a UAT measurement that arrives
after the code exists. The seam AX-1 ratified **already exists** — `synth.Voice` (`voice.go:21-25`) — and
`buildVoice` (Task 2.7) is the one branch point a resident backend plugs into. 0.14.0 ships
process-per-utterance + the cap (AX-1 (A)); `[radio] piper_mode` is **not** added to the config (nothing would
read it — the key returns with the measured policy). MVS-D-17's measurement (`perf-protocol.md` §3) stays on the
HUM LEAD's UAT list as 0.15.0 DISCOVER input.

**Plan edits this implies (applied):** `config.Radio.PiperMode`, its `Validate` branch, `cast.Config.PiperMode`,
the `piper_mode` fixture lines and `castConfig`'s mapping (P1) are dropped; `radioDeck.residents`,
`newResidentPool`, `deck.Close` and the `stopAll` line (this batch) go; `buildVoice` keeps the comment "a resident
backend plugs in here".

---

### Task 2.12 — P2 gate (the batch-exit checklist)

```
go test ./domains/radio/... ./app -race -count=2 -timeout 180s
make verify
a2dh validate && make alloc-budget && golangci-lint run ./... && staticcheck ./...   # gates.md §1
A2DH=<framework build> make p10             # the PINNED framework build (gates.md batch record)
cp dist/p10.json 06_docs/02_features/multi-voice-support/07-readiness/p10-p2.json
make pty-severe
go test ./app -run DeclarationSet -update-declset
git commit -m "multi-voice-support P2: roles on segments, voice-keyed cache, Source-time hand-over, the Station Director, find-only alert path, tone as constants"
```

- **P10 ledger:** re-key the two `app/narrate.go` symbol rows to `app/director.go` (Task 2.6, after the
  script learns renames); delete the `app/voices.go:SetVoice` row if one exists. Expected from the tool: reason
  refreshes on the `synth`/`app` package rows; `store`'s eviction loop is counter-form and raises nothing; rows are
  added only from `dist/p10.json`.
- **Build log:** `04-development/p2-build-log.md`; the `where-things-happen` rows (Task 2.6) land here, not in
  P4. No attribution trailers in the commit.

UAT-able alone: set `[radio] cast = "cast"` and `[radio.voices.fire] macos = "Rishi"` by hand, tune a location with fire data — the fire
report hands over to Rishi and back; the sign-off names whoever reaches it; a takeover opens with the classic
tone still (the other presets are P3).
