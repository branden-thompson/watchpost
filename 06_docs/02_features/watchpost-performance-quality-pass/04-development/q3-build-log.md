# Q3 build log — Render path, app side (`v0.9.6`, with Q2)

**Batch:** Q3 of the plan of record v3 (`03-architecture-design/quality-pass-plan.md` §2.5, §3 Q3).
**Approval:** "Exemption approved; go for Q3 build" (2026-08-26, at the Q2 gate).
**Branch:** `feature/watchpost-performance-quality-pass`, commits `a56061a` → `50ba83f` (+ this log).
**Status:** APPROVED 2026-08-26 ("Accept deviations, approved for 0.9.6 and Go 4 Q4a") — §6.1–6.3
accepted; `v0.9.6` cut from this tree; Q4a is next.

## 1. What changed, and why (junior-first)

Before Q3 the dashboard redrew everything 3.3 times a second whether or not anything moved, and
every redraw rebuilt both tables from scratch. Q3 makes a frame cost what changed.

- **The tick runs only while something animates.** The 300 ms "shimmer" tick used to run from
  launch to quit. Now a predicate (`tickNeeded`) says when the frame would change on its own —
  a row still loading, a volume blink, the radio marquee (unless the visualizer's faster tick is
  already redrawing), the `[S]` ages, Location Details' "N h ago" labels — and the tick is armed
  after every message only while it holds. An idle dashboard renders nothing between events. The
  round-2 list of every `time.Now()` reachable from `View()` (R2-23) was re-taken after the Q2
  moves: `radio_panel.go:46/50/67` (blink), `:234` (marquee), `dashboard.go:534` (marquee clock),
  `status.go:31` (ages), `detail.go:55/58` (marine/fire rows) — each is a predicate term or a
  consequence of one, and each has a test that the frame moves with the clock.
- **The frame's geometry is resolved once.** Whether the terminal is "short" (compact mode), how
  tall the radio and alert modules are, how many RECENT rows fit — these were recomputed about
  eight times per frame, each time rendering the whole radio module to count its lines. `layout()`
  now computes them once per `View` and once per key event and hands the result down.
- **The two tables are remembered.** They are 42 % of a frame and only change when one of their
  inputs does. A single slot on the model (a pointer allocated at construction, because `View` is
  a value receiver) holds the last rendered pair with a key made of *every* input they read; a
  frame whose key matches reuses the strings. The key has one field per input and the test has
  one row per field (§3).
- **One RECENT snapshot per tier tick.** The fifty seeded locations each have a scheduler; when
  their 10-minute tier fires, the fetches land as a wave over a few seconds and the 50 ms window
  published ~47 times (Q1 soak: publishes 44 → 91 across one tick). The RECENT window is now 5 s.
  And scheduler tiers now fire on a fixed grid from start rather than "Every after the cycle
  ends", so fifty phases that started 10 ms apart stay 10 ms apart for days instead of drifting
  by their own fetch times (red-team PF-9).
- **The fire archive is parsed by a hand-decoded stream.** The 1.4 MB HMS archive used to be
  inflated into memory, then walked with a struct decode and a map per placemark. Now it streams
  through the token walk with an explicit element stack, the description is read field by field
  with `strings.Cut`, and satellite/method names are shared strings: 27.5k placemarks in 88 ms /
  33 MB / 605k allocations (was 104 / 75 / 1.05 M) and the inflated file is never held. The old
  parser lives in the test file as the equivalence oracle.
- **The WFIGS layer is decoded once per change**, like HMS, through a shared `fire.Memo[T]`.
- **A cached body is handed to its parser without a copy.** `GetText` returns the cache's own
  slice, documented read-only; every consumer package pins that its parser never writes into it.
- **The location index loads once** (was twice at launch: 36 ms / 19 MB / 500k allocations each).
- **`--ascii` is wired** (the B6 promise): the row marks and the Help legend read one glyph set
  from `Opts.Glyphs()`, so the table and the legend cannot disagree.

## 2. Files touched

| Area | Files |
|---|---|
| `modes/tty` | `dashboard.go` (tick predicate, `dispatch`/`armTick`/`applyTick`/`applyRecent`, `Config.ASCII`, the memo slot), `layout.go` (new), `memo.go` (new), `view.go` (+ `frameText`: trim, indent and the base-grey re-arm in one pre-sized pass — was three copies), `body.go` (`writeBody` into the frame's buffer), `alerts.go`, `radio_panel.go` (`indent` retired), `nav.go`, `help_about.go`; tests `tick_test.go`, `memo_test.go`, `golden_test.go` (+ 2 goldens), `bench_test.go` (hit/miss budgets), the test runner (`runCmd`/`drain` flatten batches) |
| `platform/render` | `themes.go` (`ThemeGeneration`), `units.go` (`Glyphs`), `table.go` (marks from the set) |
| `platform/snapshot` | `Key` via `strconv` (+ equivalence test) |
| `platform/sched` | `runTier` on a fixed grid (+ drift test) |
| `app` | `pipelines.go` (window per publisher, `recentPublishCoalesce`), `dashboard.go` (`Options`, geodata once), `resolve.go`, `refs.go`, `stats.go` (gauges `wfigs.memo.incidents`, `hms.memo.parses`, `wfigs.memo.parses`), `dump.go` (`total_alloc`, `mallocs`); tests |
| `domains/fire` | `memo.go` (new, `Memo[T]`), `hms/hms.go` (streaming walk, `kmlWalk`, `descFields`, interner, `byteCount`), `wfigs/wfigs.go` (`decodeLayer`, memo, `MemoIncidents`); tests incl. the reference parser |
| `platform/httpx` | `GetText` aliases (+ pin) |
| `cmd/watchpost` | `--ascii` persistent flag; `app.Options` |
| docs | `CHANGELOG.md` `[Unreleased]`, `README.md`, `docs/where-things-happen.md` (+3 rows, +4 vocabulary), `architecture.md` §6 / §11.2 / new §11.5a, `caching.md` |
| records | `07-readiness/p10-q3.json`, `02-analysis/q3-bench.txt`, `02-analysis/q3-soak-1h.csv`, `02-analysis/q3-alloc-1h.csv` |

## 3. Tests first (the pins)

| Pin | Guards |
|---|---|
| `TestTickArmedOnlyWhileAnimating` (9 rows) | the predicate, row by row: idle, loading, blink, marquee, marquee-under-viz, LIVE RADIO, min player, `[S]`, Details |
| `TestTickRearmsOnTheTransitionAndNeverTwice` | re-arm on a loading snapshot; never two tickers; stops when the row loads (BQ-12) |
| `TestVolumeBlinkClearsOnTheTickAfterItExpires` | the blink's clearing tick |
| `TestMarqueeAdvancesAcrossTicksWithVizOff`, `TestStatusAgesAndDetailsLabelsMoveWithTheClock` | the wall-clock elements the tick serves (PF-2, R2-23) |
| `TestBodyMemoInvalidatesOnEveryInput` (15 rows) + `…OnAThemeSwitch` | one row per key field incl. `[space]`, stop, `[r]`, `[T]`, `[v]`, Setup's bold rule, the loading frame, `[t]` (R2-4) |
| `TestBodyMemoHitsBetweenTicksAndMissesOnce` | hit/miss accounting; the slot is shared across model copies |
| `TestLayoutOncePerViewAndKey` | `AllocsPerRun`: a hit frame ≤ 2 × one layout; a navigation key ≤ 2 × layout + 50 (CQ-9, JD-7) |
| `TestFrameAllocBudget` (hit **and** miss per size) | hit ≤ 6,000 (plan §1), miss ≤ Q0 × 1.05 |
| `TestFrameGoldenColourOff`, `TestFrameGoldenASCII` | the 133×44 frame byte for byte (A11-10); the ASCII frame carries no ▶ ∞ ◆ ⚠ › ✔ ✘ |
| `TestFrameHonoursNoColorUnderColorTerm` | the known-failing pin: skipped with the measured escape count until Q4a-004; passes by itself the day the kit honours NO_COLOR |
| `TestThemeGenerationMovesOnSwitchAndRegistration`, `TestGlyphsSwapAsOneSetUnderASCII` | the memo's theme key; the one glyph set |
| `TestKeyMatchesTheSprintfForm` | `strconv` == `Sprintf` over a grid incl. −0 |
| `TestTierCadenceIsAFixedGrid` | 3 s of fetch time does not move the +20 s slot |
| `TestPublisherWindowFoldsAWave` | 20 triggers in one window → 1 publish, 19 folded; RECENT ≥ 1 s, priority 50 ms |
| `TestStreamingParserMatchesTheReference` (9 rows), `…RefusalPathsStillRefuse` | equivalence with the pre-Q3 parser; outage page, empty, whitespace, torn body, EOF mid-document, over-budget archive (CQ-11) |
| `TestSatelliteAndMethodStringsAreShared` | the interner and its bound; allocations per placemark < 70 % of the reference (measured 22 vs 38) |
| `TestMemoParsesOncePerBodyChange`, `TestLayerIsDecodedOncePerBodyChange` | `fire.Memo`; WFIGS one decode + one request for two fetches |
| `TestGetTextAliasesTheCachedBody` + `TestGetTextCallersMustNotMutate` ×4 (hms, firms, ndbc, wfigs) | the contract's two sides (CQ-12) |
| `TestGeodataLoadsOncePerEntryPoint` | one `geodata.Load` on the dashboard path, one on the report path |

## 4. Before / after

| Measure (133×44, colour on, canonical fixture) | Q0 | Q3 |
|---|---|---|
| frame, memo hit (every tick/marquee/viz frame) | 660 µs · 436 KB · 10,044 allocs | **100 µs · 62 KB · 546 allocs** (−85 % / −86 % / −95 %) |
| frame, memo miss (snapshot, key, resize) | same | 585 µs · 368 KB · 9,165 allocs (−11 % / −16 % / −9 %) |
| 133×70 hit / miss | 15,539 allocs | 480 / 14,569 |
| 200×60 hit / miss | 20,031 allocs | 476 / 19,061 |
| help modal frame | — | 997 µs · 981 KB · 3,633 allocs (the Overlay canvas, L5-F9 — declined) |
| idle tick | 3.3 fps forever | none |
| HMS parse, 27.5k placemarks | 104 ms · 75 MB · 1.045 M allocs | **88 ms · 33 MB · 605k allocs**; inflated body not held |
| WFIGS decodes per hour (60 locations) | ~200 | 6 (one per 10-min body change) |
| geodata loads at launch | 2 | 1 |
| RECENT publishes per tier tick | ~47 | 1–2 (window 5 s) — soak §8 |
| `snapshot.Key` | Sprintf, 2 allocs | strconv, 49 ns, 1 alloc |
| coverage tty / app / cmd / hms / wfigs / sched | 90.2 / 38.6 / 56.9 / 83.0 / 78.7 / 79.5 % | 91.2 / 38.7 / 58.2 / 89.4 / 80.3 / 79.3 % |

## 5. Bounds stated (§0.8)

- **Body memo:** one slot, the last key; the strings it holds are the two tables (≤ ~120 KB at
  200×60). Invalidation is by key equality; a missing key field is the only failure mode, and the
  test table is the guard. No package state.
- **Tick predicate:** a false negative freezes a moving label; every wall-clock element has a
  term and a movement test. Known imprecision: the compact (short-terminal) player has no marquee
  but the predicate treats "playing with narration" as animating unless `[T]` Min is set — a
  3.3 fps hit-path frame (~120 µs) on short terminals with the radio on, not a frozen label.
- **`fire.Memo`:** one body's parse per feed; a parse error is memoised until the body changes
  (Fetch forgets the cache entry, so the next fetch is a new body).
- **Interner:** ≤ 64 distinct strings per parse, allocated per parse.
- **HMS token walk:** `maxTokens` and `maxPlacemarks` as before; `maxFields` = 32 per description.
- **Grid cadence:** a cycle that overruns its slot fires again at once and restarts the grid from
  then; missed slots are never replayed (no catch-up storm).
- **GetText aliasing:** the hazard is a caller writing into the cache; four parsers are pinned;
  a new `GetText` consumer must add the same test (`caching.md` says so).

## 6. Decisions and deviations from the plan

1. **No package-level `layoutFor` memo.** The plan listed a single-slot memo of the table layout
   (CQ-8). With the body memo, `layoutFor` runs only on a table re-render, and a package-level
   slot would be exactly the P10-06 state the plan otherwise forbids. Dropped; the effect is
   delivered by the body memo. *Deviation recorded for HUM LEAD.*
2. **The frame's µs target.** Plan §1 expected ≤ 470 µs at 133×44 after Q3. The memo-hit frame
   — every tick, marquee and visualizer frame — is 100 µs; the miss (a table re-render on a
   snapshot, key or resize) is 585 µs, above the line. The miss is the go-studs path Q4 owns
   (width regexp, `SGR` allocations, the tty probe: L5-F1/F2/F3). Recorded as met on the hit path,
   open on the miss path until Q4a. *Deviation recorded.*
3. **HMS allocations.** PF-7 hoped for a large cut; 38 → 22 allocations per placemark (−42 %),
   bytes −56 %, time −15 %. The remainder is `encoding/xml`'s own element-name strings (eight per
   placemark). A bespoke byte scanner would remove them at the cost of re-implementing entity
   and CDATA handling for a parse that runs once per 10 minutes; not taken. *Deviation recorded.*
4. **RawToken with an explicit stack, not Token.** `Token()` costs eight more allocations per
   placemark for name-space translation the walk never reads; the stack restores the one thing
   `Token()` checked that matters — a torn body errors, so Fetch forgets the cached entry (P6).
5. **RECENT window 5 s.** "One publish per tier tick" needs both the grid (so waves stay waves)
   and a window longer than a wave; 5 s folds a wave and a seed row still fills within five
   seconds of its fetch landing (the favourites keep 50 ms; M1 is theirs).
6. **`--ascii` scope.** The marks (▶ ∞ ◆ ⚠ ›), the health glyphs (✔ ⚠ ✘) and the alert module's ⚠
   swap. Box-drawing (─ │ ━ █ ░ ▲ ▼) and the radio labels' ♪ ■ ▶ ✘ do not: they are drawing, not
   marks, and every terminal font carries them. The modal error lines' ⚠ also stay (they are not
   in the frame golden). Listed so the next pass can widen it if wanted.
7. **Test runner.** Since a hook's command may now ride in a batch with the tick, the tty tests'
   `drain`/`runCmd` flatten `tea.BatchMsg` and drop tick messages (a tick costs its 300 ms once).
8. **Compass 8 → 16** untouched (excluded by HUM LEAD).

## 7. Gate

| Check | Result |
|---|---|
| `make verify` | ALL GATES GREEN |
| `make alloc-budget` | green — hit 546 / 429 / 425 (budget 6,000); miss 9,165 / 14,518 / 19,010 (budget Q0 × 1.05) |
| `make tidy` / `make vuln` (Go 1.27.0) | verified / no vulnerabilities |
| `make p10` | **0 live · 0 unmatched** · ledger unchanged (120 = 66 kit + 54 non-kit; no new exemption) · `07-readiness/p10-q3.json` |
| goldens (plain, ASCII); tick tests; declset re-captured (render, tty, app — intentional, listed in §2) | green |
| `a2dh validate` | 18/18 |
| frame ≤ 470 µs at 133×44 (local) | **hit 100 µs — met; miss 585 µs — open until Q4a** (§6.2) |
| 1 h soak: viz on, then viz-off idle sample (BQ-12) | §8 |
| allocation rate per phase; `publishes.recent`/h vs §1 (R2-7) | §8 |

**Deviations for HUM LEAD:** §6.1 (no `layoutFor` slot), §6.2 (miss-path µs), §6.3 (HMS
allocation count). None is a regression; each is stated so the gate is a decision.

## 8. The 1-hour soak (macOS, 133×44, `dist/watchpost` at `50ba83f`'s parent, 60 s samples)

Driver: `run.expect` tuned the focused location, pressed `[v]`, held 30 min, pressed `[v]` and
`[space]`, held 30 min, quit. Two things the driver did not intend, both visible in the data and
stated here rather than smoothed over: the tuned broadcast ran with **Repeat: Off**, so the synth
finished its cycle and the player stopped by itself after ~5 min (the marks say "viz chip not
confirmed" — the chip is tinted, so the pattern missed — and "stop not confirmed" at 30 min,
because the `[space]` *re-tuned* a stopped player). The run therefore has four phases, which is
more than was asked for:

| Phase | What ran | Allocation rate (`total_alloc` Δ/min, median) | Frames |
|---|---|---|---|
| A1 22:50–22:55 | synth playing, **visualizer on** (50 ms tick) | **≈ 190 MB/min** | viz frames at 20 fps ≈ 1,200/min × 62 KB ≈ 74 MB/min of that; the rest is the synth rendering its cycle (PCM) |
| A2 22:56–23:19 | stopped, viz on but nothing to draw, **tick disarmed** — the BQ-12 idle sample | **≈ 9 MB/min** (6–12) | none — no message, no frame |
| B1 23:20–23:25 | synth playing, viz off — the marquee on the 300 ms tick | ≈ 47 MB/min | 200 frames/min × 62 KB ≈ 12 MB/min; the rest synth |
| B2 23:26–23:49 | stopped, idle | ≈ 9 MB/min | none |

Periodic bursts ride on the idle baseline: **≈ 105–260 MB/min for one minute every 15 minutes**
(23:04, 23:19, 23:34, 23:49 with ~100–230k mallocs; 23:09 and 23:39 with 1.6–2.2 M mallocs at the
30-minute forecast tier). The 15-minute spacing is the RECENT fire tier; the low malloc count with
high bytes says large buffers (a body download and its cache write, or the archive parse when the
10-minute TTL has lapsed), not the tables. The new `hms.memo.parses` / `wfigs.memo.parses` gauges
are in `counters.json` from this commit so the next soak attributes them by count; the fire
fan-out is Q5's (§2.6), where they are measured against the tile canonicalisation.

Against §1: the idle floor is **≈ 9 MB/min** with the render's share **0** (PF-1 estimated 16–23
before anything animates; the tick-off render target of ≤ 2 MB/min is met by construction — no
frame is drawn). Radio-on render: 12 MB/min on the tick (target ≤ 20 with the memo: **met**); with
the visualizer on, 20 fps × 62 KB ≈ 74 MB/min — the visualizer is a 20 fps animation and its
frame is now the 62 KB hit path (was 436 KB: ≈ 520 MB/min).

| Counter | Value over the hour |
|---|---|
| `publishes.recent` | **37** (0.6/min; the Q1 soak saw 44 → 91 across one 10-minute tick, ≈ 5–10/min) · folded 2,095 |
| `publishes.priority` | 242 (4/min: the 20 s alert tier + observations, unchanged by design — the favourites keep the 50 ms window) |
| heap after GC | 47–79 MB sawtooth, no trend across the hour (10-row series in `q3-soak-1h.csv`) |
| RSS / threads / goroutines | 92–163 MB · 31–32 · 276–280 — flat |
| disk cache | 738 → 752 files, 35.1 → 35.6 MB (the Q1 sweep holds) |

Files: `02-analysis/q3-soak-1h.csv` (the sampler's rows), `02-analysis/q3-alloc-1h.csv`
(`total_alloc`, `mallocs`, `heap_alloc`, `num_gc`, publishes per minute), `02-analysis/q3-bench.txt`.

## 9. Carried forward

- 24 h idle soak (Q0 apparatus, port 6060) ends ~2026-08-27 15:55 UTC; σ result → the next log.
- Arch half of the relay proof: HUM LEAD's run, ≤ Q7.
- Q4a next: go-studs correctness patches, each presented (004 `NoAutoStyle` first — the NO_COLOR
  pin flips green with it; 001 lazy tty probe; 003 composite `SGR`; 008 bounded truncation).
  `v0.9.6` ships Q2 + Q3 after this gate; `v0.9.7` is Q4a.
