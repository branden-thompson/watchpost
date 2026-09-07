# P2 build log — the broadcast and narration seams (multi-voice-support, 0.14.0)

```
Batch:   P2 — roles on segments, the Source under a cast, the Station Director, the alert path
Date:    2026-08-30
Branch:  feature/multi-voice-support
Gate:    07-readiness/gates.md §1
Host:    darwin/arm64, Apple M3 Pro
CLI:     a2dh v1.17.0 (commit 7960f0e)
```

**Verifiable alone by an agent; the first batch with something a listener can HEAR** — a hand-written
`[radio.voices.fire] macos = "Rishi"` now changes who reads the fire report, and correspondents introduce
themselves at the boundary. The listener-facing judgement still belongs to P4, when Setup can assign a voice
without editing a file (RN-4).

## Tasks

| Task | What landed | State |
|---|---|---|
| 2.0 | `Preset`, `Classic()`, `PresetByName`, `AlertTone(preset, rate)`, `ToneRate` | ✅ |
| 2.1 | `Segment.Role`, the section tags, `tail:{{voice}}`, `Composer.HandoffLine` | ✅ |
| 2.2 | `SetResolver`, the voice-keyed byte-bounded cache, `spokenName` | ✅ |
| 2.3 | the hand-over rendered ahead; `play` writes; soft vs hard generations | ✅ |
| 2.4 | `scripts/handover/line.txt` + the built-in fallback | ✅ |
| 2.5 / 2.5b | the tail's new wording; the composed-cycle golden | ✅ |
| 2.6 | the Station Director rename; `Run(ctx, class, role, audible, seq)`; `WithPriority` on every job | ✅ |
| 2.7 | `resolveVoice` find-only, `tone(class)` from constants, the install cap, `setCast`/`setTones` | ✅ |
| 2.8 | `startSynth` installs the resolver and the hand-over line | ✅ |
| 2.9 | the ticker's one-tone-per-burst and the read's class tone; the M4 log entries | ✅ |
| 2.9b | `app/tone_latency_test.go` — the P0 twin is the version committed at `2b6377c` | ✅ |
| 2.10 | `attachRadio` carries the cast; the `[M]` hook where the deck is in scope | ✅ |
| 2.11 | `ResidentPiper` cut (E-1, confirmed as MVS-D-29) | ✅ |
| 2.12 | this gate | ✅ |

## The rename (Task 2.6)

`app/narrate.go` → `app/director.go`, by pattern rather than by list: **13 files, 65 lines**. `narration*`,
`narrateRead` and `narrateBreaking` survive untouched — they name the thing SPOKEN, not the arbiter. The
`grep -ri narrator` count over the tree is now **0**.

**The P10 ledger's first real rename.** P1 Task 1.0 taught the scope helper to read a rename, and this gate is
the first run that exercised it on one. It worked exactly as designed: the two ratified rows on `narrate.go`
(`admit`, `awaitAir`) came back as **unmatched, in scope** — correctly demanding a re-key rather than going
quietly dormant — and were re-keyed to `director.go` with their reasons refreshed.

A third row, `app/voices.go:SetVoice` (P10-01), came back unmatched and was **deleted as dead, not re-keyed**:
its reason said the finding came from calling `synth.Source.SetVoice`, and that method no longer exists
(MVS-D-3 retires the chooser). The deck's `SetVoice` now calls `Recast()`, so the finding cannot return.

## Two contracts discovered by making the tests pass

Both are in the code as comments, because both are non-obvious and both were wrong first:

1. **A recast must reach segments already rendered ahead.** With one segment of look-ahead, a recast can land
   after the next segment is already voiced by the outgoing correspondent. Without carrying the hard generation
   on the rendered segment, the listener hears the OLD voice again immediately after the change they just made.
2. **The introduction is spoken when the voice ON AIR changes** — not per queued segment. Several look-ahead
   segments may need to switch after one recast; only the first of them introduces anybody. Announcing per
   segment made three correspondents introduce themselves in a row, each to the same person.

## The M4 pin — it failed, then passed, and the reason matters

The first 0.14.0 measurement read **+69 %** against P0's stored baseline, over M4's +50 % ceiling. Two things
were true at once, and separating them was the work:

**A real regression, fixed.** Three things had been added to the path between `breaking()`'s entry and the tone:
`cast.Classify`, the `breaking:` debug line, and `WithPriority`. Bisecting the path (not the profile — the
profile is dominated by post-stamp work) attributed ~460 ns to the first two on real data, far more than a
constant-folded micro-benchmark suggested. Both were fixed on their merits, not to satisfy the gate:

- The debug line was **built on every takeover whether or not anybody was collecting the log**. `radioDebugOn()`
  now gates the concatenation. That is pure waste removed from the one path M4 measures.
- `Classify` lowered its input, **allocating on the alert path**, to compare against rules that are already
  lower-case. It now folds at the comparison (`hasSuffixFold` / `containsFold`) and allocates nothing.

**A measurement problem, named.** Re-running the *same* `v0.13.0` tree back-to-back gave **1,120 ns**, not the
1,018.5 ns stored at P0 — 10 % of drift from machine state alone, and the instrument's spread reaches 20 %
under load (0.13.0 ranged 1,085–1,288 across ten runs). On a ~1 µs number a ±50 % gate sits within a couple of
runs of its own noise.

**Paired result of record** — both trees, back-to-back, same session, `-count=10` each:

| | median | p95 | range |
|---|---|---|---|
| 0.13.0 | **1,120 ns** | 1,209 | 1,085–1,288 |
| 0.14.0 | **1,628 ns** | 2,076 | 1,532–2,136 |

**+45.3 %, ceiling 1,681 ns — PASS.** Against the *stored* P0 number it would read +60 % and fail. The paired
figure is the honest one, and `perf-protocol.md` §0 now says so: **comparisons must be paired**, and a stored
baseline compared against weeks later means little. Worth keeping in proportion either way — the whole path is
1.6 µs against the live budget's **250 ms**.

## Gate

| Gate | Result |
|---|---|
| Unit + race | `go test ./... -race` green |
| Verify | `make verify` — **ALL GATES GREEN** |
| Harness | `a2dh validate` 17/17 |
| Alloc pins | `make alloc-budget` green |
| P10 | **0 live, 0 unmatched** (113 dormant) · record at `07-readiness/p10-p2.json` |
| Declsets | `app` re-captured |
| PTY | `make pty-severe` green |
| Goldens | `domains/radio/synth/testdata/cycle.golden` captured |
| Docs | `where-things-happen` green; rows re-pointed and three added |
| M4 | paired, **PASS** (above) |

### P10 ledger changes — for HUM LEAD ratification

| File | Symbol | Rule | Action |
|---|---|---|---|
| `app/director.go` | `admit` | P10-02 | **re-keyed** from `app/narrate.go`, reason refreshed |
| `app/director.go` | `awaitAir` | P10-02 | **re-keyed** from `app/narrate.go`, reason refreshed |
| `app/voices.go` | `SetVoice` | P10-01 | **deleted as dead** — `synth.Source.SetVoice` no longer exists |

No new exemptions. Two live findings were **fixed rather than exempted**: an ineffectual assignment in
`source.go`'s play loop (`text` was carried forward after `takeOver` had already re-derived it), and the
condition-only loop finding, which was the re-keyed `awaitAir` row.

## Deviations from the plan

1. **`ToneRate` is declared in `tone.go`, not moved from `limit.go`.** P1 never put it in `limit.go`; the plan
   assumed it had. One declaration either way.
2. **The `[M]` mute hook lost its home in `tickerMuteState`.** The plan said the hook must be built where the
   pipelines are in scope; that meant `tickerMuteState` could no longer own it, since the deck does not exist
   when it runs. It now returns the flag alone, and `livePipelines.muteHook` is the hook. It also stopped
   writing `ticker_muted` — `config.Save` is the one owner of that mirror.
3. **A `radioDebugLog` package-level function was extracted.** The ticker writes the M4 `breaking:` entry and
   has no deck; the second caller is where the helper gets extracted, rather than a second implementation of
   the same file format.
4. **The composed-cycle golden pins keys, roles and pauses — not text.** Wording is the HUM LEAD's and changes
   without the structure changing; pinning it would make every script edit read as an architectural regression.
   It earned its place immediately: it caught the fire and seismic **notice** segments untagged (reading in the
   root's voice rather than their report's), which no other assertion noticed.

## Notes carried to P3/P4

1. **Only `Classic` exists as a preset.** `PresetByName` falls back to it, so between now and P3 Task 3.6 every
   class sounds the classic tone. Deliberate, and stated in the code — but it means P2's UAT cannot tell the
   classes apart by ear yet.
2. **`saveTones` and `reloadCast` are wired but the Setup UI is P4's.** `[M]` uses `saveTones` today; the
   per-class checkboxes will use the same path.
3. **`installState` (the `[S]` wording) is still unwritten**, as planned. `InstallsRemaining()` and `Problems()`
   are the data it will read.
4. **The M4 instrument's noise floor is now documented.** P4's re-measure must be paired the same way, and the
   Linux row should be too.
