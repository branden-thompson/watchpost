# Performance & resource protocol — multi-voice-support (0.14.0)

The standing lens, instrumented (red-team D-2): what is measured, how, by whom, against which 0.13.0 number, and
which decision each measurement gates. Owner: HUM LEAD for the Linux rows (the Arch box); the agent for the
macOS rows and the PTY/recording-voice rows.

## 0. Baselines from 0.13.0 (`main @ 186d97c`)

| Measure | 0.13.0 | Source |
|---|---|---|
| Whole-app RSS | **116 MB at launch · 86–115 MB plateau** over the 1-hour soak, no trend | `06_docs/02_features/severe-alerts-modals/08-reports/validate-report.md` §4; `07-readiness/validate/soak-1h.csv` |
| Frame allocations | 133×44 hit 655 / miss 7 102 · 80×24 964 / 3 238 · severe window 2 031 / 7 637 (pins ×1.05) | `modes/tty/bench_test.go` (one owner) |
| Setup window allocations (today's form) | **133×44 hit 3,046 / miss 3,476 · 80×24 hit 1,978 / miss 2,478** — measured at **P0, 2026-08-30**, in the tree; pinned ×1.05 by `TestSetupAllocBudget`. Informational "before": P4 Task 4.11 re-measures the new window and re-pins | `modes/tty/bench_test.go` (one owner) |
| Synth PCM cache | 40 segments ≤ 29 MB mono | `synth/source.go:344-348`; `synth.pcm.cache` gauge |
| Takeover time-to-tone-start (code path), **0.14.0 P2 paired re-measure** | 0.13.0 **median 1,120 ns · p95 1,209** · 0.14.0 **median 1,628 ns · p95 2,076** → **+45.3 %, ceiling +50 % (1,681 ns) — PASS.** Both trees measured back-to-back on the same machine state, `-count=10` each. See the note below on why the paired figure is the one of record | `04-development/p2-build-log.md` |
| Takeover time-to-tone-start (code path), P0 baseline | **median 1,018.5 ns · p95 1,044 ns** (10 runs of `BenchmarkTimeToToneStart`, `ns/tone`; darwin/arm64, Apple M3 Pro) — recorded at **P0, 2026-08-30**, on a worktree of the tag `v0.13.0` (`baec8b0`) with `app/tone_latency_test.go` copied in unchanged | §1 item 1; `04-development/p0-build-log.md` |

**Why the paired figure is of record (0.14.0 P2).** The P0 baseline was taken hours before the P2 measurement. Re-running the *same* `v0.13.0` tree back-to-back with 0.14.0 gave **1,120 ns**, not 1,018.5 — a 10 % drift from machine state alone, and the instrument's own run-to-run spread reaches 20 % under load (0.13.0 ranged 1,085–1,288 ns across ten runs). On a ~1 µs number a ±50 % gate is therefore within a couple of runs of its own noise. **Comparisons must be paired** — both trees, back-to-back, same session — and a single stored baseline compared against weeks later means little. Against the stored P0 number 0.14.0 reads +60 %; against the paired one, +45 %. The paired number is the honest one, and the drift is the reason.

| Takeover time-to-tone-start (live) | **no 0.13.0 number exists and none is needed** — the release binary carries no `breaking:`/`tone:` log line; §1 item 2 is an absolute budget on 0.14.0 | §1 item 2 |
| Piper per-utterance | ~10 s model load on every run (UAT 119.1) | `watchpost-cli/04-development/b3-uat-log.md:829` |
| Per-Piper-process RSS | **unmeasured** (§3) | — |

## 1. Time-to-tone-start (M4 / NFR-2) — PLAN batch 0 records the 0.13.0 number

**The one owner of the instrument (Task 0.1, NFR-2, gates.md and the Linux protocol cite this paragraph).**
Two measures, two purposes:
1. **The code-path pin** — `app/tone_latency_test.go`'s `BenchmarkTimeToToneStart` (P2 Task 2.9b; the same file
   is run on the `v0.13.0` tree at P0): the ticker's `breaking()` over a Director whose narration-voice fake
   stamps its `tone()` call; the number is `stamp − entry` (`go test ./app -run '^$' -bench TimeToToneStart -count=10`,
   median and p95). It measures Go — the Director's admission — on both trees (`radioDeck.engine` is a concrete
   `*player.Engine`, so the fake cannot sit below the deck). Pass: the 0.14.0 median ≤ the 0.13.0 median + 50 %
   **as a regression pin on the path, not the audio**.
2. **The live budget** — the debug log (`WATCHPOST_DEBUG_RADIO=<path>`, a file path): the `breaking:<key>` entry
   (P2 Task 2.9) and the `tone:<class>` entry (P2 Task 2.7). Pass, absolute, on 0.14.0: `breaking:` → `tone:`
   **≤ 250 ms** on a host with the root voice installed **and** on a fresh Linux HOME with nothing installed (the
   find-only path FR-9 promises; 0.13.0's `tone()` went through `d.voice()` and could wait on an install — the
   thing this release removes). The 0.13.0 release binary carries no log line; no 0.13.0 live number exists, and
   none is needed for an absolute budget.
**The writer budget:** the output path holds ≈ 0.7 s of audio (oto's 0.5 s mux buffer + the 200 ms device
buffer); anything the Source's writer goroutine waits on longer than that is audible silence. Rules: hand-over
lines are rendered ahead and cached (and pre-warmed at `SetResolver`); a background change (`Invalidate`)
never renders on the writer — the rendered segment plays, the next re-resolves; the one accepted exception is
the listener's own Recast (a deliberate act: one render, non-fatal). `TestWriterNeverStarvesAcrossHandOvers`
(a paced reader, ≤ 700 ms between reads, **zero writer Says**) and `TestInvalidateNeverRendersOnTheWriter` pin
it. **The first-cycle inequality** (Linux, ~10 s a Say): with one segment of look-ahead the writer is fed only if
`audio(prev) + gap + 0.7 s ≥ render(next) + render(hand-over)` at every boundary — the pre-warm removes the
second term; the Linux row 2.3 listens for the first cycle specifically. Pass: 0.14.0 median ≤ 0.13.0 median
+ 50 ms (the tone starts at the same moment; the **words** may start up to 1.2 s later by the preset's length —
deliberate, `02-analysis/tones.md`).

## 2. Frame allocations (NFR-3)

The existing pins, plus **one new owner row: Setup open with the Correspondents group expanded** at 80×24 and
133×44 (spike-then-pin ×1.05, as 0.13.0's `severeAllocBudget` did — measured at P4 on the new window). **P0 is done (2026-08-30):** the window's hit/miss are **133×44 hit 3,046 / miss 3,476 · 80×24 hit 1,978 /
miss 2,478**, measured by `TestSetupAllocBudget` over `setupBench` (the fixture dashboard with `openSetup`), and
pinned at ×1.05. The PLAN probe predicted 3 046 / 3 476 and 1 979 / 2 478 — the 80×24 hit came in one allocation
lower and the pin is based on the **measurement** (1,978), not the probe. These are the **informational**
"before" numbers, not the release pin. Nothing new runs per tick (the on-air chip was dropped, MVS-D-24); per-tick misses are a failure.

## 3. Per-Piper-process RSS (MVS-D-17 — 0.15.0's resident decision input, OQ-18) — HUM LEAD, Arch box, at UAT on a **0.14.1 or later** build

*(Step 1 assigns two voices in Setup, which 0.13.0 cannot do. The 0.13.0 comparison row is §4's, on `v0.13.0`
with one voice.)*

> **The 0.14.0 window is excluded, and a run on it must be discarded rather than adjusted (amended
> 2026-09-07).** This section's headline numbers come from step 2's takeover-over-read overlap. On
> Linux, 0.14.0 never reached Piper for a takeover at all: issue #7 left `FindPiperVoice` holding an
> empty `Key`, so every installed voice read as missing and the alert fell silent. The location
> report still spoke, because it passed a full spec — so the app looks healthy while the measurement
> is wrong. A §3 run on 0.14.0 Linux therefore samples ONE voice's footprint and records it as two,
> and the overlap the section exists to capture never occurs. Both headline numbers understate, and
> §4's summed-RSS criterion inherits the error, since it is stated relative to this one.

1. Install two Piper voices (`Setup` → assign *Alerts* and *Standard* distinct voices).
2. Tune a coastal location with fire and seismic data so the broadcast runs long; trigger a takeover mid-read.
3. Sample every second for two minutes: `ps -o rss= -C piper` (sum and max) and `grep VmHWM /proc/<piper pid>/status`
   for each Piper process; note the count of simultaneous `piper` processes (`pgrep -c piper`).
4. Record: max simultaneous processes · peak RSS per process · peak summed RSS · the app's own RSS (`[S]` or
   `ps -o rss= -C watchpost`).
5. Also check (red-team B-5): with `--json-input`, whether the process signals completion for `output_file` (or use `--output_dir`, which prints the path after close); how long a cancelled 4 000-rune utterance keeps the process busy (cancel = kill-and-respawn — the respawn cost is the NFR-2 price); that `stopAll` can close the pool (stdin EOF → bounded wait → kill) within the shutdown budget.
6. Decision rule for OQ-18 — **0.15.0's** (the resident backend is cut from 0.14.0, E-1): if a resident
   process holds < 200 MB and two resident processes + the app stay under the phase-B numbers (§4) on the Arch
   box, 0.15.0 DISCOVER may adopt the resident design; otherwise process-per-utterance + the FR-12 cap stays.

## 4. Soak (RS-6 / phase B)

The 0.13.0 procedure (`WATCHPOST_DEBUG_PPROF`, 1 hour, 5-minute samples, post-GC series) with two assigned
voices and Repeat: One on a coastal location. **Pass, in numbers:** the app's RSS (`ps -o rss= -C watchpost`)
≤ 115 MB × 1.10 = **126 MB** at every sample with no upward trend (0.13.0 plateaued 86–115 MB); the summed RSS
of `piper` processes during a takeover-over-read overlap (`ps -o rss= -C piper`, sum) ≤ **the 0.13.0 summed
figure × 1.10** — the 0.13.0 figure is measured once on `v0.13.0` with one voice during a takeover-over-read
(Linux row 1.3; one voice already spawns several `piper` processes today, AM-3) — and, absolutely, ≤ (N + 2) ×
the §3 per-process number; at most N + 2 `piper` processes at once (`pgrep -c piper`, FR-12: N ordinary + 2
reserved); the PCM cache gauge (`synth.pcm.cache`) ≤ 40 MB (the hand-over lines inside it: ≈ 2.2 MB for five
voices, ≤ 7.9 MB worst).

## 5. Disk (NFR-4)

`disk.voices` gauges **extracted** bytes; the download sizes (26.5 MB binary archive, 63.2 MB per model) are a
lower bound. Measure the extracted `piper/` directory once on the Arch box (`du -sh ~/.cache/watchpost/piper`)
after `EnsurePiper` and record it here; NFR-4's budget is stated in extracted bytes from then on.

## 6. Card build (PD-2 — the Station Director's standby decision)

**The one owner of the instrument.** `app/cutover_latency_test.go`'s `TestCardBuildCost` times
`radioDeck.segments()` for the application default location (Bonsall, CA — R-4) cold and then warm,
and attributes its requests through `httpx.Client.RequestStats()`. Live by design and opt-in:
`WATCHPOST_LIVE_PERF=1 go test ./app -run TestCardBuildCost -v -count=3`, take the median. `make
verify` skips it. It **validates itself before reporting** — a build that yields no segments, or that
never reaches the network on the cold pass, fails rather than reporting a zero that reads like a fast
path.

**What it measures, and what it deliberately does not.** Audio look-ahead *inside* a report already
exists — `synth.Source.Open` renders one segment ahead on its own goroutine and the writer is
forbidden to synthesise (the `writerKey` seam exists so a test can assert that as a call count rather
than a stopwatch). There is no gap between segments. The gap is at the **card boundary**, and it is
data assembly rather than speech: `segments()` fetches observation, alerts, office, products,
forecast zone and county **in a straight line with no concurrency**, after `tune()` has already done
its own county lookup and relay resolve — all of it after the previous report has gone quiet.

**A second instrument sits beside it.** `TestCardBuildConcurrentFloor` issues the same calls
concurrently — only `products` depends on `office` — to measure what concurrency alone would buy, and
what it costs in extra requests (a `/points` cache stampede is the plausible price). It self-validates
the same way: a run that resolves no office, zone or county fails rather than reporting a fast number
for doing no work.

**Recorded 2026-09-01, macOS (darwin 25.6.0), `-count=5`:**

| Arrangement | Samples | Median | Network | Cache |
|---|---|---|---|---|
| **Cold, sequential** (today) | 1.365 · 0.885 · 1.025 · 0.907 · 1.027 s | **1.03 s** | 11 | 0 |
| **Cold, concurrent floor** | 1.074 · 0.875 · 0.932 · 0.964 · 1.195 s | **0.96 s** | 11 | 0 |
| Warm | 1–3 ms | — | 0 | 8 |

**Three findings.**

1. **Against §1's audibility threshold** — the output path holds ≈ 0.7 s of audio, so anything the
   writer waits on beyond that is audible silence — a **1.03 s** card build is about **one and a half
   times the buffer**, before `tune`'s own requests and before the first render (≈ 1 s a Say on macOS;
   §1 records ~10 s on Linux). Every watchlist advance pays it.
2. **Concurrency does not help.** 0.96 s against 1.03 s is ~60 ms, within the run-to-run spread, at an
   identical request count. The calls are not 11 independent round-trips: they funnel through a shared
   `/points` resolution that serialises however they are issued. The build is **not
   latency-parallelisable**, so standby is the remedy rather than one of two candidates.
3. **The warm figure is the half that decides it.** 1–3 ms means a card, once built, costs essentially
   nothing to rebuild — so moving the build to standby *removes* the gap rather than relocating it.
   That is the distinction three earlier attempts on the burst bounds never made.

### A correction, recorded because a wrong number does not self-correct

**The first run of this instrument reported 2.14 s, and it was wrong.** The client was built without
setting `RatePerSec`, so it took httpx's default of **5/sec** (`httpx.go:232`) where the app uses
**30** (`app/app.go:123`). Eleven requests at 5/sec is ~2.2 s — **the instrument was timing its own
misconfiguration**, and the result looked like an entirely plausible network figure. The tell was
visible in hindsight: three samples within 0.1 s of each other, which is a token bucket in lockstep,
not a network.

It was caught by the PLAN red team asking whether the baseline was measured under realistic
conditions. The conclusion survived; the number did not. `cardBuildClient` is now a named constructor
carrying the reason, so the next reader cannot repeat it by omission.

## §8 — Memory is measured in THREE scenarios, not one

**HUM LEAD, 2026-09-03**, after a memory question that an idle measurement could not have answered:

> *"it is a good catch to test the three scenarios - app idle, app under consistent load, app under
> heavy action load (switching / location lookups / reads / alerts / w reads / etc)"*

| # | Scenario | What it is for | What it misses on its own |
|---|---|---|---|
| **1. Idle** | Launched, settled, nothing touched | The floor, and whether the marquee's 24/7 frame leaks | Everything paid per ACTION. An idle app never creates a second scheduler |
| **2. Consistent load** | Playing, left alone, sampled over an hour (`scripts/quality/soak.sh` + `tools/slope`) | Leak vs plateau over TIME — the slope, which is the only number that distinguishes a cost from a defect | Anything paid per distinct location or per read |
| **3. Heavy action** | Switching locations, `[space]` plays, `[w]` reads, stream toggles, in cycles | Costs paid PER ACTION: one scheduler per RECENT location touched, rendered speech, the severe index growing with locations | Slow growth over hours |

**Why all three.** On 2026-09-03 an idle-then-playing soak was started against a ~170 MB report and
was **stopped after eight samples as the wrong question**: the reported growth came from heavy
switching, and scenario 2 holds the location set still. Scenario 3 found it in twelve minutes — 309
scheduler goroutines across 52 schedulers, exactly ADR-03's documented per-location cost, bounded by
`RecentCap = 50`.

**The trap, stated because it cost real time.** RSS is too noisy for this. The SAME binary measured
**96.9 MB and 134.0 MB** on two runs of one script — a ±35 MB spread that swamps the ~10 MB signal
being looked for. Use **physical footprint** (what macOS's own Memory column reports) plus Go's
`Sys`, `HeapAlloc` and `Stack` from `WATCHPOST_DEBUG_PPROF`, and compare against the SAME scenario on
a prior tag rather than against a remembered figure.

**And match the process exactly.** `pgrep -f dist/watchpost` matches the `expect` wrapper too, which
reports 9 MB and looks like a triumph. Use `pgrep -x watchpost`.
