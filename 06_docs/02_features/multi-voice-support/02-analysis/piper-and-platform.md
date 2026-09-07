# Piper cost and platform parity — DISCOVER analysis (brief A-4 · A-8)

Read-only investigation as of `main @ 186d97c` (2026-08-29), on macOS (Darwin 25.6.0) with the Linux path
reasoned from code and the 0.9.x UAT record. The performance-and-resource lens applies throughout.

## A-4 Piper cost

### 1. The process-per-utterance contract

`PiperVoice.Say` (`domains/radio/synth/voice.go:179-194`): `exec.CommandContext(ctx, binary, "--model", model,
"--output-raw")` (`:183`), `cmd.Dir` = the binary's dir (`:184`), `LD_LIBRARY_PATH` *replaced* (`:168-176`,
red-team 0.9.0 S-F8), stdin = one line + EOF (`:186`), `cmd.Output()` buffers the whole s16le stream (`:189`).
Pinned release `2023.11.14-2` (`install.go:45`), archives for linux/{amd64,arm64,arm} and windows/amd64
(`install.go:51-56`); macOS deliberately absent (`install.go:23-27`, UAT 77.2).

**A long-lived process per voice is viable — the single biggest win available.** In that Piper release
`main.cpp` loads the voice **once**, then loops `while (getline(cin, line))`: the ~63 MB ONNX load is a
*startup* cost the current code pays on every line only because it re-execs. `--json-input` exists
(`{"text", "speaker_id", "output_file"}`). The catch is **delimiting**: `--output-raw` streams int16 with no
framing between utterances. Options: **A (recommended)** `--json-input` + a per-utterance 0600 temp WAV named
by the app, read back through the existing `wavPCM` (`voice.go:124-138`) and deleted — the `say` path already
works this way (`voice.go:98-108`) and §10.5 holds (text never in argv); B `--output_dir` (Piper prints each
file's path); C sentinel framing in `--output-raw` — fragile, rejected. A/B confine the change to `PiperVoice`
(a held `*exec.Cmd`, a per-voice **queue** — not a lock, see §4 —, health check, restart-on-crash, idle
reaper); the `synth.Voice` interface (`voice.go:21-25`) is unchanged.

### 2. What the record says about latency and memory

| Fact | Number | Source |
|---|---|---|
| Piper reads the ONNX on every run | **~10 s per utterance, even when installed** | `06_docs/02_features/watchpost-cli/04-development/b3-uat-log.md:829` (UAT 119.1); why the preview timeout went 20 → 60 s (`app/voices.go:142`) |
| macOS `say` per line | **≈ 1 s** | `b3-uat-log.md:631` (UAT 94.1); `app/radio.go:395-397`'s "say/piper blocks ~1 s per line" is macOS-accurate and **Linux-optimistic by 10×** |
| Concurrent-subprocess budget | **≤ 2** — by construction, in the docs | `watchpost-performance-quality-pass/07-readiness/soak-profile.md:37`; `02-analysis/lens-L1-memory-allocation.md:29` (L1-F18) |
| Whole-app RSS | 80–140 MB soak; 123–455 MB observed peaks | `06-key_learnings/quality-baseline.md:29,54,57` |
| PCM cache | 40 segments ≈ ≤ 29 MB mono | L1-F13; `source.go:344-348` |
| **Per-Piper-process RSS** | **unmeasured** — no figure anywhere; ≥ 63 MB model + onnxruntime arena, expect ~120–200 MB | gap — RS-6 pins a peak never observed |

**The ≤ 2 budget is not enforced and is already exceeded.** No semaphore exists in `app/` or `domains/radio/`
(the only limiters in the tree are `platform/httpx`'s); the two `installMu` serialise installs only. Paths that
can be inside `Say` at once: the Source's render-ahead (`source.go:174`), `handOver`'s **two parallel says**
(`source.go:245-250`), a just-suspended read still rendering (`app/narrate.go:129` renders **before** `awaitAir`
at `:134`), the takeover's own render (`app/radio.go:395-407`), a preview (`app/voices.go:144`). Worst case
**6 processes** (one Source alone reaches 3 — red-team B-2 measured the hand-over pair beside the play goroutine's re-voice) — invisible on macOS at ~1 s each; on Linux at ~10 s and ~150 MB each, a ~750 MB spike on
the most urgent path. **This is the headline A-4 finding.**

### 3. The install path, and what pre-install (R-9) touches

Entry `voice()` (`app/voices.go:29-63`): darwin → `SayVoice` at once; else `FindPiperVoice` → miss → install,
under `d.installMu` (`app/radio.go:49`, `voices.go:53-54`) **held across the whole download** (minutes), plus
the package `synth.installMu` (`install.go:165,184`, double-checked `:179,186`). Install dir
`os.UserCacheDir()/watchpost/piper` (`app/voices.go:274`, `app/dashboard.go:292-298`): `piper/piper`
(`install.go:138-144`), `voices/<key>.onnx` + `.onnx.json` (`:153`); gauged as `disk.voices` (`app/stats.go:108,190`).
Timeouts: 15 min per `EnsureVoice` (`voices.go:83`), 10 min per asset (`install.go:251`), 60 s per preview
(`voices.go:142`). Catalogue: Lessac (default `install.go:94`), Amy, Ryan, Joe, Alan, Alba (`install.go:76-91`).
Progress: `installing %s… %d%% (%d MB)` (`voices.go:85-91`) to the player line on tune-in (`:58`) or to the chooser
via `voiceNote` → `tty.VoiceNoteMsg` on preview (`:134,155-159`; `modal_chooser.go:143-145`).

**Sizes — an assumption to validate (done below).** `install.go:80` hardcodes `Size: 63201294` for **all six**
models; it is the `io.LimitReader` ceiling (`install.go:279`) and the completeness check (`:288`). Only Lessac is
installed on a typical host; pre-installing alert-role voices is exactly what would expose a mismatch as a
spurious "download incomplete".

**Pre-install does not exist today** — `EnsurePiper` (`install.go:167`) has no non-test caller; startup only does
`go d.listVoices()` (`app/radio.go:89`). A pre-install pass is a goroutine beside it that walks the resolved
alert-role voices. Constraints: it must **not** hold `d.installMu` for the duration (two serial voices = up to
30 min during which any takeover reaching `voice()` blocks — RS-1 exactly), and `PreviewVoice` already bypasses
`d.installMu` (`voices.go:132-137`) so it and a pre-install serialise only on the package mutex, which nothing
can time out of. **Mid-install behaviour (R-9, M5):** the alert path must be *find-only* — a non-blocking
`FindPiperVoice` probe (pure `os.Stat`, `install.go:148-160`) up the inheritance chain, first hit speaks, the
install proceeds in the background; `defaultVoice()` (`app/voices.go:262-270`) already models "what will
actually speak".

### 4. Rate and the hand-over's concurrency

Every voice is 22 050 Hz with no exception (`install.go:73,79`; `voice.go:91,204`); the engine resamples to
44 100 (`resample.go:10`, `engine.go:273`) — the `SetVoice` rate refusal (`source.go:87-89`) is dead code today.
`handOver` runs **two `Say` calls in the same voice at once** (`source.go:245-250`): with process-per-utterance
on Linux that is two simultaneous 63 MB loads of the same model (~20 s, ~300 MB) to cover a hand-over designed
to cost "one short render"; with a long-lived process it needs a per-voice **queue**, not a lock, or it deadlocks.

## A-8 Platform parity (macOS)

### 5. The curated voices vs this machine

`macVoices()` (`app/voices.go:224-230`): the `System Voice` sentinel + 10 names; `say -v ?` here lists 186 rows.
**All ten curated voices are installed** (Aman · Daniel · Eddy · Karen · Moira · Reed · Rishi · Samantha · Tara ·
Tessa; en_IN/GB/US/AU/IE/ZA). Ambiguities for per-role assignment: `Aman (English (India))` and
`Tara (English (India))` each appear **twice** (one engine says "Hello! My name is Aman.", the other "Hi, I'm
Siri!"); `parseSayVoices` collapses duplicates by name (`voices.go:245`), so `say -v` picks between two engines
undefined — the ear test (M3) could differ run to run. No Enhanced/Premium variants are listed here; when a user
downloads one it appears under the **same base name**, so identity on macOS is not stable across machines. The
other rows are novelty voices (Bad News, Bells, Boing, Zarvox…) — **keep the curation; do not widen it for roles.**

**The `System Voice` sentinel carries no identity on modern macOS.** `SayVoice.Name()` (`voice.go:60-68`) reads
`defaults read com.apple.speech.voice.prefs SelectedVoiceName` once (`voice.go:73-81`); on this machine the
domain **does not exist**, so the memo is empty and the station signs off *"This is the System Voice for
Watchpost Weather Radio"* (the UAT-88 fallback). For R-5/R-10 the sentinel has no resolvable name, and two
roles inheriting it are indistinguishable to `SetVoice`'s dedupe (`source.go:90`).

### 6. Config portability — and a silent-wrong-voice bug

`Config.Voice string` (`config.go:147`), no namespace, no version (`:138-152`). The two namespaces: macOS `say`
names vs Piper `Key`/`Name` (`install.go:84-104`, either accepted case-insensitively).

*macOS name on Linux* — `piperSpec()` (`voices.go:67-78`) misses → `InstalledVoices[0]` → Lessac: **silent
substitution.** *Piper name on macOS* — `voice()` (`voices.go:30-40`) does **no validation**: `SayVoice{Voice:
"Amy"}` → `say -v Amy`. Measured here: `say -v Amy`, `say -v ZZZNotAVoice123`, `say -v en_US-amy-medium` all
**exit 0 and speak in the default voice**, while `SayVoice.Name()` returns the bogus name unconditionally
(`voice.go:61-63`) — so `{{voice}}` and the sign-off announce *"This is Amy"* while Samantha reads. **An M1/M5
failure invisible at every layer** (no error, no `[S]` line, no chip change); older macOS did error, so the
exit code cannot be relied on.

**Proposal — a typed per-role value, validated before use:**

```toml
voice = "System Voice"          # unchanged: the All-reads root
[radio.roles.breaking]
macos = "Rishi"
piper = "en_US-ryan-medium"     # the Key, not "Ryan": keys are stable, names are ours
```

A single string cannot round-trip: Linux writes `"Ryan"`, the Mac reads it as Samantha and — on the next Setup
save — rewrites it as `"Rishi"`, destroying the Linux assignment. Per-namespace fields let each OS write only its
own; resolution reads `roles[r][ns]` → inheritance chain → root → platform default, **recording why** for `[S]`
(R-7); the macOS value is validated against the discovered list (`d.voices`, `app/radio.go:89`) **before**
constructing `SayVoice`, the Linux value with `FindPiperVoice`.

### 7. The preview path

`p` in the chooser (`modal_chooser.go:96-102`) → `PreviewVoice` (`voices.go:121-151`) → `SamplePCM`
(`compose.go:191-198`, script `voice-preview/sample`: *"This is {{.Voice}} for Watchpost Weather Radio."*) →
`engine.Audition` (`voices.go:150`). A per-role Setup preview reuses all of it (`SamplePCM`/`Audition` take a
`synth.Voice`, not a name); it needs a role → `synth.Voice` resolver, a progress line of its own (the
`VoiceNoteMsg` channel belongs to the chooser being removed), and a `{{Role}}` key in the sample script.

**`Audition` plays *over* a live broadcast, undecked** — `playClip` creates and starts a new player (`engine.go:277,287`)
mixed with whatever is playing; `PreviewVoice` never ducks (the narrator does, `narrate.go:238-241`); `tapped=true`
feeds the visualizer with the mix; `inFlight=false` keeps a takeover from pausing it (R5-B-03 — R-4's "never
pausable" for free). With ~6 role previews instead of one, decide explicitly: duck for the preview (the arbiter
owning the duck, not `PreviewVoice`) or gate previews while the radio plays. → **OQ-17**.

## Summary

| Dimension | Value |
|---|---|
| Per-voice disk (Piper) | **63.2 MB** model + ~4 KB json; binary 22–26 MB once; N voices ≈ 26.5 + 63.2·N MB (2 → 153 MB, 3 → 216 MB, all 6 → 406 MB); macOS 0 MB managed |
| Per-utterance load (Piper) | **~10 s on every run** (UAT 119.1) — the model reload; `say` ≈ 1 s |
| Per-process RSS (Piper) | **unmeasured** — must be measured on the Arch box before committing to resident processes |
| Concurrent processes | budgeted ≤ 2 (docs) · **enforced: none** · actual worst case **4–5** |
| Sample rate | 22 050 Hz everywhere; the rate refusal is dead code |
| Install serialisation | two mutexes held across whole downloads; `PreviewVoice` bypasses the deck one |
| Startup pre-install | does not exist |

## Recommendations carried to PLAN

1. **Cap concurrent synthesis first** — a weighted semaphore around every `Say` (2 on Linux/Windows, 3–4 on
   macOS) with a reserved slot for the takeover path so `handOver` does not re-introduce the gap it hides.
2. **A long-lived `piper` per assigned voice** (`--json-input` + per-utterance temp WAV): the ~10 s reload becomes
   a one-time warm-up, making 2–3 Piper voices *cheaper* than one is today; **measure one process's RSS on the
   HUM LEAD's Arch box first**; keep process-per-utterance as the fallback behind the same interface; a
   per-voice queue for `handOver`. If too much for 0.14.0: cap only (correctness now, latency later).
3. **Alert path find-only, pre-install in the background** (`resolveVoice(role)` walks the chain with `os.Stat`;
   `ensureRoleVoices()` beside `listVoices()`); `[S]` shows *installing*, not *broken*. **Verify the
   six model sizes** (below).
4. **Type the per-role voice as `{macos, piper}`**, Piper by Key, validated against the discovered list before use.
5. **Decide the preview-during-broadcast behaviour** (OQ-17); reuse `SamplePCM`/`Audition` unchanged.

## Assumption validated 2026-08-29 — the six model sizes

HEAD requests against the six pinned `.onnx` assets (`huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/…`):
lessac · amy · ryan · joe · alan · alba → **`Content-Length: 63201294`, all six.** The medium models share one
architecture and weight count, so `install.go:80`'s single constant is correct today. It stays a pin worth a
test (a catalogue change to a non-medium model would break the download ceiling silently), not a live bug.
