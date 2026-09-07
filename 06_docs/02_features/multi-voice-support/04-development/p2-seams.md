# P2 — the broadcast and narration seams (multi-voice-support, 0.14.0)

```
Goal:         Every read speaks in its role's voice: segments carry a role, the Source resolves and caches per
              voice, correspondents hand over by name, and the Station Director forwards role and class so the
              alert path stays find-only with its tone from constants.
Architecture: plan.md §2.3–2.5; data-shape.md §5 (the seam, not the code)
Branch:       feature/multi-voice-support
Gate:         07-readiness/gates.md §1
```

Task shape only; code at BUILD (`AP-PLANCODE-01`). The round-4 lenses assembled this batch's sketches over the
real package and ran them — **17/19 and 12/14 green under `-race`**, including both hand-over paths, the
writer-starvation property and announce-failure non-fatality. Those sketches are in `prior-art/p2-seams-code.md`
with their two known contradictions (D-1, D-2 there), both settled as contracts below.

## The contracts this batch is built on

1. **Two generations.** A *soft* change (`Invalidate` — a host fact landed: discovery, an install) takes effect
   at the next segment **rendered**; with one segment of look-ahead that is two segments later, and the segment
   already rendered plays as it is. A *hard* change (`Recast` — the listener saved a cast) hands the running
   segment over at the spot reached. **The writer goroutine never renders for a soft change** (D-R3-2/D-R4-2).
2. **One failure rule.** A **segment**'s render failure ends the broadcast with its reason (0.13.0's behaviour —
   there is no audio to play). A **hand-over line**'s failure never does: it is reported on the marquee and the
   segment plays, because FR-5's "never silence" is better served by continuing (D-R4-1). The two are different
   failures and the tests must not assert one rule of the other.
3. **The hand-over is rendered ahead of the air.** The render-ahead goroutine sees the voice change between two
   segments and renders the scripted line then, cached per (from → to, line text) under the incoming voice; the
   writer only writes it. The one render the writer performs is the listener's own mid-segment `Recast`, folded
   with the remainder into a single `Say`.
4. **No launch-time pre-warm ships in 0.14.0.** A `SetResolver`-time warm of every ordered voice pair costs
   V(V−1) serial renders on one ordinary slot (~10 s each on Linux: 60 s at three voices, 420 s at seven) and
   warms pairs the cycle may never cross. If Linux row 2.3 shows a first-cycle gap at a boundary, the remedy is a
   **per-cycle** warm — render this cycle's boundary lines when `next()` returns — decided with the measurement
   in hand, not before (this retires PERF3-4).

---

### Task 2.0 — the tone as parameters

**Files:** `domains/radio/synth/tone.go`, `tone_test.go`.

**Contract:** `Preset{Name, Freqs, Pulses, PulseDur, GapDur, Sweep, Decay, Amp}`, `Classic()` carrying today's
constants, `PresetByName`, `AlertTone(preset, rate)`, and `ToneRate = 22050` moved here from P1's `limit.go`
(one declaration). `AlertTone` validates its own edge — a non-positive rate, pulse count or duration returns
nil, never a panic — and fills gaps within its pre-sized buffer rather than allocating per gap.

**Test intent:** today's classic tone is unchanged in length, rate and envelope; the existing `AlertTone(0)` /
`AlertTone(-1)` pins move to the new arity; a malformed preset returns nil.

**Verify:** `go test ./domains/radio/synth -run 'Tone|Preset' -count=1`

---

### Task 2.1 — `Segment.Role`, the tags, the tail, `HandoffLine`

**Files:** `domains/radio/synth/compose.go`, `fire.go`, `seismic.go`, `synth_test.go`, `seismic_test.go`.

**Contract:** `Segment` gains `Role cast.Role`; `Compose` tags each section (lead/tail → Station,
conditions/alerts/products → Weather, each report → its role). The tail is keyed `"tail"` and carries
`VoiceToken` — the voice belongs in the *cache key*, not the segment key (the latent `tail:{{voice}}` bug).
`HandoffLine(from, to)` reads the script tree's `handover/line` and falls back to the built-in
`"This is <to>, taking over for <from>."` — a broken override must never become silence. `Composer.Tail`'s
empty-name branch is deleted: `spokenName` owns the "your correspondent" fallback now.

**Existing pins this changes** (rewrite each, with its new value): the composed tail text and key in
`synth_test.go`; the `tail:` prefix assertion in `seismic_test.go`; `std.Tail("")`; P1's zero-value test.

**Verify:** `go test ./domains/radio/synth -run 'Compose|Tail|Handoff' -count=1`

---

### Task 2.2 — the Source resolves a voice per segment; the cache is voice-keyed

**Files:** `domains/radio/synth/source.go`, `synth_test.go`.

**Contract:** `SetResolver(func(cast.Role) (Voice, error))`; a segment's voice is resolved **once**, at render,
and travels with its audio. The cache key carries the rendering voice's **full name**, so two engines that share
a spoken name stay apart and a render that straddled a resolution change is still correct — it is stored
unconditionally. The cache is bounded in **bytes** (40 MB; a cycle in one voice ≈ 29 MB, plus a second
correspondent's sections and ≤ 4.6 MB of hand-over lines), evicted oldest-first by a counter-bounded loop, with
a duplicate-key guard so the byte count cannot drift. `Cached()` reports count and bytes in O(1).
`spokenName(v)` is the one owner of a name that will be spoken or shown: `PlainLine`, one line, ≤ 48 runes,
"your correspondent" when empty. `SetVoice` is deleted with the chooser (MVS-D-3) — a root change is a Setup
save, i.e. a `Recast`.

**Test intent:** each segment renders in its role's voice; the cache key separates two voices; the marquee and
the audio agree on the name; a hostile name reaches neither Piper's stdin nor the frame. Note that the
hand-over line (Task 2.3) is a cache entry too — a two-segment, one-hand-over cycle holds **three**.

**Existing pins this changes:** the mid-segment hand-over test moves from `SetVoice` to a resolver swap plus
`Recast`; the rate-refusal pin goes with `SetVoice` (a new-rate catalogue voice arrives with a resampling
decorator — backlog); the render-failure pin keeps its **segment**-failure assertion (contract 2 above).

**Verify:** `go test ./domains/radio/synth -run 'Source|Cache|Voice|Marquee|SpokenName' -race -count=2`

---

### Task 2.3 — the hand-over, rendered ahead; `play` writes

**Files:** `domains/radio/synth/source.go`, `synth_test.go`.

**Contract:** the render-ahead goroutine detects the voice change between consecutive segments and renders the
line there, cached per (from → to, line text). `play` announces it and writes; on a soft-generation mismatch it
plays what it has (contract 1); on a hard one it re-renders the remainder and the line as a single `Say` at the
spot reached, non-fatal (contract 2). The context `play` renders under is **marked as the writer's**, so tests
can assert the property directly: *zero Says on the writer goroutine across hand-overs*.

**Test intent:** one hand-over per voice change, spoken by the incoming voice, naming the outgoing one; a
mid-segment `Recast` produces exactly one utterance carrying line + remainder at the word boundary; an
`Invalidate` produces **no** mid-segment hand-over (and, with look-ahead, no hand-over until the next rendered
segment — assert that, not a specific speaker); a failed line leaves `handoff:failed` on the marquee with the
segment still played and `Err()` nil; a failed *segment* render still ends the stream with its reason.
**R6:** with a paced reader at real time and a slow voice, the gap between writes stays inside the output
path's ≈ 0.7 s of slack **and** the writer-Say count is zero — the count is the property; the gap is the symptom.

**Verify:** `go test ./domains/radio/synth -run 'HandsOver|Recast|Invalidate|Starve|Remainder' -race -count=2`

---

### Task 2.4 — the hand-over script part (MVS-D-6; RAT-4)

**Files:** `domains/radio/script/scripts/handover/line.txt`, `script_test.go`.

**Contract:** the default wording is `"This is {{.To}}, taking over for {{.From}}."`; data is `.To`/`.From`
(both already Plain and capped). The convention test's data map gains `"To"` only — `"From"` exists. An override
that fails to render falls back to the built-in line.

**Verify:** `go test ./domains/radio/script -count=1`

---

### Task 2.5 — the tail script; 2.5b — the composed-cycle golden

**Files:** `scripts/weather-radio/tail.txt`; `domains/radio/synth/testdata/cycle.golden`, `synth_test.go`.

**Contract (2.5):** the tail names the correspondent who reaches it and points at Setup rather than `[V]`; the
wording change is one of FR-2's three deliberate deltas. **(2.5b):** a golden of the composed segment list
(keys, roles, pauses) so FR-2's "byte-for-byte except the named deltas" is machine-checked. The golden's flag is
the tree's `-update-golden`.

**Verify:** `go test ./domains/radio/synth -run Golden -count=1`

---

### Task 2.6 — the Station Director: the rename, and `Run(ctx, class, role, audible, seq)`

**Files:** `app/narrate.go` → `app/director.go` (+ its test file); `app/ticker.go`, `severe_read.go`,
`domains/radio/player/engine.go`, `modes/tty/severe.go`; `docs/where-things-happen.md`.

**Contract:** a rename by grep, not by list — every `narrator` identifier, parameter, test name and comment
becomes the Director; **keep** `narration*`, `narrateRead`, `narrateBreaking` (they name the thing spoken, not
the arbiter). `Run` gains the job's `role` and forwards the tone's `class`; the protocol (admission, ducking,
suspension, restore) is unchanged and its bodies travel verbatim. Every Director job carries `WithPriority`
(D-R3-1). `clip.role` is set once, by the deck's `render`.

**Docs (gated by `make verify`):** the rows naming `app/narrate.go:…` move to `director.go`; the row naming
`app/voices.go:SetVoice` and the one naming the voice chooser are re-pointed (both symbols are deleted this
release); add rows for where a role's voice is chosen and where a correspondent hands over.

**P10 ledger:** the two ratified symbol rows on `narrate.go` are re-keyed to `director.go` — P1 Task 1.0 taught
the scope helper to read a rename, and this gate is the first run that exercises it on a real one.

**Existing pins this changes:** every `Run(...)` call site gains its role; the tone fakes record the class; the
Director tests that assert a sequence gain the tone entry.

**Verify:** `go test ./app -run 'Director|Narrat' -race -count=2`

---

### Task 2.7 — the deck resolves by role; the tone from constants; find-only on the alert path

**Files:** `app/radio.go`, `app/voices.go`, `app/radio_test.go`.

**Contract:** `resolveVoice(role)` = `cast.Resolve` + `buildVoice`, **find-only** — it never takes the install
mutex (FR-9); a Piper key that is missing starts one background install per key, and the takeover speaks in the
fallback voice meanwhile, **subject to a per-session cap of two unattended background installs (MVS-D-44,
E-10)**: a hand-edited config naming six missing Piper voices may not pull ~380 MB on launch unasked. Past the
cap no further background install starts; the roles resolve up the tree and `[S]` reads "install on next Save".
A Setup **save** is an explicit act and is not capped — it clears the counter along with the failure memory. A
failed install records its time and is **not retried for ten minutes**, so an offline host cannot re-download
63 MB per segment. `tone(class)` renders from
constants at `ToneRate` and never resolves a voice; a muted class sounds nothing while the words always read.
`render(role, text)` records **why** it failed on the player's detail line — a silent takeover must never be
unexplained. The root's tune path keeps today's blocking first-run install (UAT 118), including the catalogue
default when the config names no voice. `setCast`/`castChanged` follow P1's lock discipline: **snapshot under
the lock, `cast.Validate` and `cast.Resolve` outside it, store under the lock**; `Resolutions()` and
`Problems()` are memoised until a cast or host fact changes and copy on return. `installState` (the `[S]`
wording) lands with its consumer in P4, not here.

**Test intent:** a config naming three missing Piper voices starts exactly two background installs and the third
role resolves up the tree with the "install on next Save" state; a Setup save resets the counter and the failure
memory; a role inherits its group's voice and the resolution is recorded; an unknown macOS name falls
back before any voice is constructed; `resolveVoice` completes while the install mutex is held; a `setCast` with
an assigned pair completes under a timeout; the tone path resolves no voice; a muted class yields no tone and
the words still read.

**Verify:** `go test ./app -run 'Tone|Resolve|Voice|InstallMutex|SetCast|InstallCap' -race -count=2 -timeout 60s`

---

### Task 2.8 — the broadcast uses the cast

**Files:** `app/radio.go`.

**Contract:** `startSynth` installs the deck's resolver and the composer's hand-over line on the Source. M1's
test: with distinct voices assigned to Alerts and Standard, each section of one broadcast renders in the voice
its role resolves to, and the hand-over is spoken once at each boundary by the incoming voice. Assert
**membership**, not index — render-ahead decides the order in which the line and the segment are produced.

**Verify:** `go test ./app -run 'Broadcast|Roles' -race -count=2`

---

### Task 2.9 — the ticker and the read pass their role and class (FR-11; MVS-D-12)

**Files:** `app/ticker.go`, `app/severe_read.go` (+ tests).

**Contract:** a breaking takeover sounds the class of its **highest-severity** event (one tone per burst) and
reads in the Breaking role's voice; a severe read sounds its row's class — new in 0.14.0 — and reads in the
SevereRead role's voice. `[M]` and the per-class set silence the **tone only**; the words always read
(MVS-D-26). The ticker logs `breaking:<key>` at entry and the deck logs `tone:<class>` before the tone — the
live M4 instrument (`perf-protocol.md` §1 item 2).

**Existing pins this changes:** the sequences these tests assert gain a tone entry, and the shared fake's
duration now also covers the tone — list each pin with its new value, including the one whose timing would
otherwise hang.

**Verify:** `go test ./app -run 'ToneClass|SevereRead|Breaking' -race -count=2`

---

### Task 2.9b — the M4 code-path pin

**Files:** `app/tone_latency_test.go`, plus the helpers it needs (this task defines them; nothing else does).

**Contract:** a benchmark from `breaking()`'s entry to the tone call, over a Director whose narration fake stamps
the moment `tone` is asked for. It measures the admission path in Go, not audio: the gate's real M4 measure is
the live budget (`perf-protocol.md` §1 item 2). Report a median and p95 across the run — collect the samples and
report once, not per iteration. **P0 runs a `v0.13.0`-shaped twin**, not this file: the 0.13.0 tree has no
`cast` package and its narration fake has the older `tone()` signature. Both files must report the same metric
name so the two numbers are comparable.

**Verify:** `go test ./app -run '^$' -bench TimeToToneStart -count=10`

---

### Task 2.10 — `attachRadio` carries the cast; saves re-cast

**Files:** `app/dashboard.go`.

**Contract:** the deck is built with the config's cast and the app's context; the Director is attached after it
is built. The `[M]` hook — built where the pipelines are in scope, not in the package-level ticker state — saves
`[radio.tones] mode` and calls the deck's `setTones` (no config reload, no re-cast, no install pass); `Save`
mirrors `ticker_muted`. A voice/tone save re-casts the deck by reloading the config once. Report a save error
the way the surrounding code already does.

**Verify:** `go build ./... && go test ./app -race -count=1`

---

### Task 2.11 — `ResidentPiper`: cut from 0.14.0 (E-1)

Four lenses converged: a warm-process backend, default-off, unrunnable on the developer's macOS, waiting on a
UAT measurement that arrives after the code exists. The seam AX-1 ratified already exists — `synth.Voice`, with
`buildVoice` as its one branch point. 0.14.0 ships process-per-utterance under the FR-12 cap; no `piper_mode`
key is added. MVS-D-17's measurement stays on the HUM LEAD's UAT list as 0.15.0 DISCOVER input. *Pending the
HUM LEAD's confirmation at the PLAN exit.*

---

### Task 2.12 — P2 gate

Run `07-readiness/gates.md` §1 (declsets before the test line; stage before `make p10`), plus `make pty-severe`.

- **P10 ledger:** the re-keyed Director rows are presented for re-ratification; rows only from `dist/p10.json`.
- **Build log** `p2-build-log.md`: the rename's grep count, the ledger decisions, the `where-things-happen` rows,
  deviations. No attribution trailers.

**UAT-able alone:** with `[radio] cast = "cast"` and `[radio.voices.fire] macos = "Rishi"` written by hand, tune
a location with fire data — the fire report hands over to Rishi and back, the sign-off names whoever reaches it,
and a takeover opens with its class tone (the other presets land in P3).
