---
title: "0.19.6 DISCOVER — lens PB: the performance and gate-time baseline"
date: 2026-11-14
phase: DISCOVER
lens: PB
status: "FILED — findings checked against e57afeaf"
---

# Lens PB: the baseline principle 1 is judged against, and where a split can move it

## Verdict

Principle 1 ("performance stays the same or improves") has a deterministic baseline. It is the
allocation pins: the three M2 beats, the no-change publish, and the frame and window pins (table A). Measured
today, every one of them equals the figure it was pinned at: idle 168 allocations and 75,988 bytes,
radio 327 and 68,475, console 377 and 122,387, publish 203 and 82,322. But the gate allows +5 %, so
a batch can raise a beat by 3.8 KB and still pass, which breaks "stays the same" (PB-1). Moving code
into packages costs nothing in itself. Go inlines across packages, there is no PGO profile to go
stale, and init is under 0.2 ms. The cost would come from what the cut forces: an interface or a
`tea.Model` return between the Router and a surface (at least 6.5 KB per beat, PB-2), or a context
of func fields where today 421 Dashboard calls in the map files inline (PB-3). The split does move
gate time. The sweep's cost per mutant is compile time, not suite time. TS's Map II would take
roughly 35-40 % off the tty fast path, about 6 minutes of a 66-minute sweep, and the large gain is
in the few mutants that run whole suites (PB-6). It barely moves `make verify`, whose
`mutant-check` step is package-blind, and `race` gains about 10 %, because `app` (201 s) is close
behind `modes/tty` (225 s) (PB-5). Three allocation pins never run in `make verify` at all,
one of them the Router-overhead pin a split leans on (PB-7). The tone-latency instrument, which
D-2's arbiter work would be judged by, reads 10x its last record and has to be reconciled first
(PB-8).

## What I checked and how

- **Tree.** `feature/re-architecting` at `fb338f4a`. Its code is `e57afeaf`:
  `git diff --stat e57afeaf HEAD -- '*.go' go.mod go.sum Makefile scripts` is empty.
- **Environment.** `go1.27.4 darwin/arm64` (`GOTOOLCHAIN=local`), Apple M5 Pro, 18 cores, 64 GB.
  Load average 6.8 at the start, falling to 1.9 by the beat benchmark. A live `watchpost` (the HUM
  LEAD's) and Defender were running throughout. **Allocation and byte figures are deterministic and
  are the evidence. Wall-clock figures are recorded with their load, not used as bounds.**
- **Runs**, one at a time, each `< /dev/null`, output in
  `<scratchpad>/q196/pb/`:

  | What | Command | Wall | Output |
  |---|---|---|---|
  | tty pins, verbose | `go test -count=1 -run 'AllocBudget' -v ./modes/tty` | 6.4 s | `alloc-tty.txt` |
  | every pin (what `make alloc-budget` runs) | `go test -count=1 -run 'AllocBudget' -v ./...` | 15.5 s | `alloc-all.txt` |
  | tty benchmarks | `go test ./modes/tty -run '^$' -bench . -benchmem -count 3` | 56 s | `bench-tty.txt` |
  | app benchmarks | `go test ./app -run '^$' -bench 'TimeToToneStart\|SevereDeckTrigger' -benchmem -count 10` | 222 s | `bench-app.txt` |
  | tty suite | `go test -count=1 -json ./modes/tty` | 60.9 s | `test-tty.json` |
  | app suite | `go test -count=1 -json ./app` | 135.4 s | `test-app.json` |
  | tty under race | `go test -race -count=1 -timeout 30m -json ./modes/tty` | 224.6 s | `race-tty.json` |
  | app under race | `go test -race -count=1 -timeout 30m -json ./app` | 201.5 s | `race-app.json` |
  | binary as `make build` builds it | `go build -trimpath -ldflags '-s -w -X main.version=pb' -o <scratch> ./cmd/watchpost` | — | `watchpost-make` |
  | cold build | `GOCACHE=<scratch>/gocache-cold go build -trimpath -o <scratch> ./cmd/watchpost`, with `-debug-actiongraph` | 4.2-4.7 s | `ag-cold.json` |
  | warm build | `go build -trimpath -o <scratch> ./cmd/watchpost` | 1.0 s | — |
  | inlining and escapes | `go build -gcflags='<pkg>=-m' -o /dev/null ./modes/tty` and `./app` | — | `m-tty.txt`, `m-app.txt` |
  | init cost | `GODEBUG=inittrace=1 <scratch>/watchpost-warm --version` | — | `init-watchpost.txt` |

- **A scratch copy** (`pb/wp/`, rsync of the tree without `.git`, `dist`, `06_docs`) carried what
  the tree must not:
  - a beat wall-time benchmark and a size probe (`wp/modes/tty/pb_scratch_test.go`);
  - four real mutants applied one at a time to time a sweep's compile steps (`pb/mutcost.sh`:
    `go build ./...`, `go vet <pkg>`, the killer test on the mutated tree, the killer on the
    restored tree, each with `-debug-actiongraph`).
- **Read, not run:** `06_docs/verify-runs.md` (112 green `verify` runs), `dist/mutant-verdicts.timings`
  (1,058 rows), `dist/mutant-check.log` (2026-11-11), `06_docs/mutants/verdicts/since-v0.19.5.log`,
  `06_docs/perf-measurement.md`, `06_docs/perf/workload-v1/`, and 0.19.5's brief, PF lens, rulings
  (A-25) and REFLECT.
- **Not run:** `make verify`, the sweep, `mutant-check`, `make property`, the workload driver
  (`scripts/quality/workload.sh` drives the live app on the network; required reading rule 5).

## The baseline, as measured

### A. Allocation pins: deterministic, gated by `make alloc-budget` (`Makefile:517-518`)

| Pin | Where | Measured today | Pinned measurement | Gate budget |
|---|---|---|---|---|
| Beat: idle Observer 133x44, tick | `modes/tty/beat_alloc_test.go:59` | 168 allocs, 75,988-75,991 B | 168, 75,990 | ×1.05 |
| Beat: radio playing 133x44, viz tick | `:60` | 327, 68,475-68,477 B | 327, 68,475 | ×1.05 |
| Beat: console 160x74, tick | `:61` | 377, 122,399-122,405 B | 377, 122,387 | ×1.05 |
| No-change publish's frame | `:106` | 203, 82,335-82,340 B | 203, 82,322 | ×1.05 |
| Frame hit / miss 133x44 | `bench_test.go:37-42`, `:55-60` | 155 / 5,525 | 155 / 5,525 | ×1.05 |
| Frame hit / miss 133x70 | same | 149 / 11,324 | 149 / 11,324 | ×1.05 |
| Frame hit / miss 200x60 | same | 152 / 14,236 | 152 / 14,236 | ×1.05 |
| Frame hit / miss 80x24 | same | 141 / 1,138 | 141 / 1,138 | ×1.05 |
| Severe window hit / miss | `bench_test.go:209` | 1,590 / 5,068-5,069 | 1,590 / 5,069 | ×1.05 |
| Settings 133x44 hit / miss | `bench_test.go:289` | 2,628 / 4,626 | same | ×1.05 |
| Settings 80x24 hit / miss | `bench_test.go:297` | 1,064 / 3,035 | same | ×1.05 |
| Settings Tab press, worst tab (Map Layers) | `bench_test.go:428` | 3,019 | 3,019 | 3,170 |
| Console frame 150x74 hit / miss | `broadcaster_alloc_test.go:102`, `:122` (logs) | 291 / 9,164 | — | 309 / 9,655 |
| Map window hit / miss | `map_alloc_test.go:70` (log) | 3,094-3,095 / 3,508 | — | 3,250 / 3,837 |
| Propagation window, plain / colour | `map_prop_alloc_test.go:112` (log) | 1,209-1,210 / 1,596; 1,581 / 3,091 | — | 1,270 / 1,758; 1,660 / 3,354 |
| `advanceTicker` per tick | `tick_alloc_test.go:31` | 3 | 3 | 3 |
| Outside the two packages (unaffected by a split, recorded): assembler publish of 25 locations | `platform/snapshot/bench_test.go:64` | 259 allocs, 75,594 B | — | 272, 79,374 |

The byte figures vary by at most 26 bytes between runs (0.02 % of the console beat). The counts do
not vary at all, except for a one-allocation flicker on the map, severe and Propagation windows.

### B. Wall time: recorded, never gated

| Instrument | Figure (median of runs) | Load |
|---|---|---|
| Beat, idle 133x44 (scratch `BenchmarkPBBeat`, `-count 5`) | 70.1 µs | 1.9 |
| Beat, radio playing 133x44 | 84.2 µs | 1.9 |
| Beat, console 160x74 | 173.8 µs | 1.9 |
| `BenchmarkFrame_133x44` / `_133x70` / `_200x60` (memo hit) | 52.1 / 72.1 / 96.4 µs; 50.6 / 75.0 / 103.4 KB | 6.8 |
| `BenchmarkFrame_133x44_Miss` | 325 µs, 385 KB, 5,530 allocs | 6.8 |
| `BenchmarkConsoleFrame` / `_Miss` | 126.6 / 431.5 µs | 6.8 |
| `BenchmarkFrame_133x44_Map` / `_MapPropagation` | 1.34 / 1.24 ms | 6.8 |
| `BenchmarkFrame_133x44_Severe` / `_Help` / `BenchmarkOverlayOnly` | 962 / 938 / 872 µs | 6.8 |
| `BenchmarkSetup_133x44` / `_80x24` / `_133x44_Miss` | 910 / 347 / 1,642 µs | 6.8 |
| `BenchmarkTimeToToneStart` (`ns/tone`, `-count 10`) | **15,761 ns**, max 17,164; 185 allocs, 21.7 KB | 6.2 |
| `BenchmarkSevereDeckTrigger` | 507 µs, 811 KB, 3,245 allocs | 6.2 |
| Startup init (`inittrace`, `--version`) | 18.4 ms over 149 packages; `modes/tty` 0.078 ms, `app` 0.10 ms; `platform/render` 8.9 ms, chroma 6.6 ms | — |
| Binary size, `make build`'s flags (`-trimpath -ldflags '-s -w -X …'`, `Makefile:24-29`) | 32,732,082 B (41,705,458 B without `-s -w`) | — |

### C. Build, test and gate time

| Measure | Figure | Source |
|---|---|---|
| Cold build of `./cmd/watchpost` (empty `GOCACHE`, std included) | 4.2-4.7 s wall, 26-27 s CPU; `modes/tty` compiles in 0.41-0.47 s, `app` in 0.30-0.39 s | `ag-cold.json` |
| Warm build (link only) | 1.0 s | run |
| `go test -count=1 ./modes/tty` | 60.9 s, serial: 1,083 top-level tests summing to 59.6 s; 0 call `t.Parallel` | `test-tty.json` |
| `go test -count=1 ./app` | 135.4 s, serial: 680 tests summing to 134.1 s; three of them 89 s (66 %) | `test-app.json` |
| `go test -race ./modes/tty` | 224.6 s wall (package 215.3 s). By test file family: `map_*` tests 166.7 s, `broadcaster_*` 1.1 s, the rest 45.6 s. Without race: 42.2, 0.1 and 17.4 s | `race-tty.json` |
| `go test -race ./app` | 201.5 s wall (package 196.8 s); the slowest test, `TestEveryLiveAlertIsEventuallyReadAloud`, 49.9 s alone | `race-app.json` |
| `make verify`, green | median 2,418 s over 52 runs since 2026-11-05 (2,061-4,003); the last on code, 2,061 s (`0519a868`, 2026-11-11) | `06_docs/verify-runs.md` |
| `make verify-docs`, green | 139-157 s | same |
| `mutant-check` inside `verify` | 1,572 s (2026-11-11) | `dist/mutant-check.log` (local) |
| Baseline sweep | 3,976 s wall; 14,101 mutant-seconds; tty 8,013 (57 %), app 3,213 (23 %) | `dist/mutant-verdicts.timings`, A-3 |

## Findings

### PB-1: The baseline is the pins' measurements, but the gate allows 5 % more, so "stays the same" needs its own bound

- **Claim.** Every allocation pin measures today exactly what it was pinned at. The gate passes
  anything up to ×1.05. A 0.19.6 batch could therefore add up to 3.8 KB to the idle beat, 3.4 KB to
  radio and 6.1 KB to the console, plus 5 % on every frame count, and `make verify` would stay
  green. Principle 1 would be broken.
- **Evidence.**
  - Budgets are "measured × 1.05" (`beat_alloc_test.go:53-62`, `:106`; `bench_test.go:37-60`,
    `:209`, `:281-298`, `:428`).
  - Today's figures equal the pinned measurements (table A). The byte jitter is at most 26 bytes.
  - 0.19.5's M2 verdict used these same figures (0.19.5 A-25: "idle 76.0 KB against 80, radio 68.5
    against 90, console 122.4 against 140").
- **Consequence.** Principle 1: the gate is the floor, but "the same" is tighter than the gate. The
  ×1.05 headroom exists so a feature can land without re-pinning. 0.19.6 is meant to change no
  behaviour, so it has no feature to spend that headroom on.
- **Recommendation.**
  - The 0.19.6 bound is the measurement, not the budget. Allocation counts no higher than the
    baseline. Bytes no higher than the baseline plus 0.1 % (about 10x the observed jitter).
  - At each batch, the verbose pin run's `t.Logf` lines (above) are compared with table A, and the
    comparison is recorded in the build log.
  - A count that falls is re-pinned down in the same batch, as 0.19.5 did.
  - Whether the budgets themselves are cut to ×1.00 for the release is a gate change. It is put to
    the HUM LEAD below.
- **Confidence.** High.

### PB-2: There is one model box per message today; an interface or a `tea.Model` between the Router and a surface would add 6.5 KB to each beat

- **Claim.** Since 0.19.5's PF-1, the Router turns into a `tea.Model` once per message, and nowhere
  else does a surface model reach the heap. A split that holds a surface behind an interface, or
  hands one back as a `tea.Model`, adds one copy of that model to every beat.
- **Evidence.**
  - `-gcflags=-m` on `modes/tty`: `router.go:463:9: out escapes to heap` is `Update`'s return, the
    one box. `router.go:466-468` says so.
  - `Router.View` draws both surfaces through their concrete types, "not through surfaceView: the
    interface would box the whole model on every frame" (`router.go:502-512`). `surfaceView`
    (`router.go:849-852`) is used only by `Init` (`router.go:320`, the two
    `r.observer`/`r.broadcaster escapes to heap` lines).
  - Sizes, from `unsafe.Sizeof` in the scratch copy: Router 7,136 B, Dashboard 6,584 B (`mapPane`
    3,248 of it, `Config` 1,320), Broadcaster 472 B.
  - One extra Dashboard box is 8.7 % of the idle beat's 75,988 B, which is more than its 5 % budget
    headroom (3.8 KB). The pin would catch it. A smaller sub-model, such as a map model of about
    3.4 KB (TS-11), would pass under the budget, and only PB-1's bound would catch it.
- **Consequence.** Principle 1: TS-3's console move and TS-11's concrete map sub-model keep the
  property, provided the Router holds `console.Broadcaster` and the map model as concrete fields, and
  their `Update` returns the concrete type, as `Broadcaster.Update` does today
  (`broadcaster.go:498`).
- **Recommendation.**
  - No sub-model implements `tea.Model` for the Router's use, and none sits behind an interface on
    `Update` or `View` (with GP-1 and TS-11).
  - At each batch, the model-typed heap escapes in `-gcflags=-m` for the packages that hold the
    Router and the surfaces are listed and diffed against today's: `router.go:463` and the two
    `Init` lines. A new one has to be explained before the batch commits.
  - A sizes probe (`unsafe.Sizeof` of Router, Dashboard, Broadcaster and any new sub-model) is
    recorded at each batch, so a field that moves by value into two places shows up.
- **Confidence.** High.

### PB-3: The package boundary itself costs nothing; a context of func fields would

- **Claim.** Go inlines across packages, escape analysis crosses packages through export data, and
  watchpost has no PGO profile. Moving a function to another package does not change its
  compiled code. What the boundary forces can: a method that becomes a call through a func field
  (an `Env` of closures) is an indirect call that never inlines.
- **Evidence.**
  - Cross-package inlining already carries the frame. `modes/tty` has 892 inlined call sites into
    other packages, 526 of them into `platform/render` (`m-tty.txt`). `app` has 1,994, among them
    `tty.Dashboard.WithRadio`, `WithSpectrum`, `ParseRepeatMode` (`m-app.txt`).
  - `modes/tty`: 748 functions can inline, 4,040 call sites inline, devirtualisation 0.
  - There is no `*.pgo` file in the tree. `Makefile:24-29` builds with `-trimpath -ldflags '-s -w …'`
    and no `-pgo`.
  - **The map is where it matters.** 421 call sites in `modes/tty/map_*.go` inline a `Dashboard`
    method today, for example `Dashboard.mapMode` (63), `settled` (28), `uiTouched` (24), `mapRow`
    (19), `mapRole` (19), `opts` (14). Under TS's Map II these become the map model's own methods or
    reads of the context passed in.
- **Consequence.** Principle 1 holds across the cut if the context is plain values and small
  concrete methods. If it is closures called per row or per cell, the frame loses inlining at those
  sites, and the allocation pins would not see it (an indirect call allocates nothing).
- **Recommendation.**
  - TS-11's "context is a plain value" is the rule. The map's `Env` carries values (options, the
    clock reading, the role) and never func fields that are read on the frame path.
  - The beat and frame wall-time benchmarks (table B) are re-run at the map batch with
    `-count 10` on a quiet machine and compared with benchstat. A regression the pins cannot see
    shows up there.
  - The inlined-call count of the moved files is compared before and after (`-gcflags=-m`, the count
    of `inlining call to` in those files).
- **Confidence.** High for the mechanism and today's figures. The map batch's figures are for that
  batch to measure.

### PB-4: Init costs nothing, but `app`'s map layers are ordered by init, and a split can reorder them

- **Claim.** Init is not a performance risk: `modes/tty` takes 0.078 ms and `app` 0.10 ms of an
  18.4 ms start. Init does carry behaviour, though. The map layers that Settings lists are registered
  by nine `init` functions in `app/map*.go`, so their order is the compiler's file order. If the
  registrants and the registry end up in different packages, the order becomes package-dependency
  order.
- **Evidence.**
  - `inittrace` (table B).
  - `registerMapLayer` appends to `mapLayers`, "every registered layer, in registration order"
    (`app/maplayers.go:185-200`).
  - `windowLayers` hands them to Settings in that order (`app/maplayers.go:205-211`).
  - The registrants are the `init` functions at `app/mapfeed.go:219`, `mapfire.go:27`,
    `mapmarine.go:27`, `mapquakes.go:31`, `maprain.go:32`, `mapradar.go:27`, `maptemp.go:37` and
    `mapwaves.go:31`. `maps.go:31` registers the basemap source, "the first is the basemap"
    (`maplayers.go:187`).
  - I found no test that asserts the sequence `windowLayers()` returns. `app/mapbadges_test.go:20`
    and `:81` iterate it. The tty Settings tests use their own `MapLayers`.
- **Consequence.** Principle 4: a move can reorder the MAP · LAYERS rows with no test going red.
- **Recommendation.** A test written first that pins `windowLayers()`'s order as it is today, before
  any `map*.go` file moves (RX, AS).
- **Confidence.** Medium. The mechanism is read from the code. "No test pins it" rests on a grep.

### PB-5: `make verify` barely moves under a split; `mutant-check` is package-blind, and a split's gain there is the `race` step

- **Claim.** Of a 2,061-2,418 s `verify`, `mutant-check` is about 1,570 s, and what it does per
  mutant does not depend on the package's size. The split's lever in `verify` is `race`, where
  `modes/tty` and `app` are serial suites that run as one unit each.
- **Evidence.**
  - `verify-gates` (`Makefile:297`) runs `race` and `mutant-check` and no plain `test`.
  - Per mutant, `mutant-check` copies the tree (`06_docs/mutants/mutants_test.go:82-112`), runs
    `go build ./...` (`:153`), then `go vet` of the target package (`:160`), four lanes at a time
    (`:219`).
  - Measured in the scratch copy after a real mutant (action graph): `go build ./...` takes 1.45 s
    for a tty edit, 1.11 s for an app edit and 1.53 s for a `platform/lineup` edit. The lineup edit
    recompiled `modes/tty` (0.42 s) and `app` (0.31 s) too. **Go's cache has no early cutoff** for a
    body change, so a mutant anywhere below `app` rebuilds it all. Only the `vet` part scales with
    the package: 0.5-1.35 s for tty against 0.18 s for lineup.
  - The suites are serial: 0 of 1,083 tty tests and 1 of 680 app tests call `t.Parallel`. `go test`
    parallelises across packages. Under race, `modes/tty` takes 224.6 s and `app` 201.5 s, so the
    two are `race`'s critical path today, and `app` is only 23 s behind. In `modes/tty`'s race run,
    the `map_*` test files take 166.7 s of 213 s, the `broadcaster_*` files 1.1 s and the rest
    45.6 s. In `app`, one test takes 49.9 s on its own, which no split of `app` can go below.
- **Consequence.** Principle 1, for the gate: do not promise a shorter `verify` from the split. A
  `race` step bounded by the largest sub-package is the honest expectation, offset by one more test
  binary to link per new package (0.1-0.3 s each). Under Map II, the map package alone carries
  about 167 s of race tests, so `race`'s critical path moves to `app` (about 200 s), unless `app` is
  split as well. Expect a `race` step about 10 % shorter, not halved.
- **Recommendation.** Record `verify`'s seconds at each batch, as `runlog` already does, plus the
  `race` step's per-package `ok` times, against this baseline. The `-timeout 30m` comment's "modes/tty,
  the slowest package" (`Makefile:35-41`) is re-measured when the slowest package changes (with
  TG-13).
- **Confidence.** High for `mutant-check` and the build cascade. Medium for the size of the `race`
  gain, which depends on the final map.

### PB-6: In the sweep, a mutant costs compile time, not suite time; Map II takes about 35-40 % off the tty fast path, and the big wins are the few mutants that run whole suites

- **Claim.** 504 of tty's 539 mutants end at step 1, the last killer run alone. They cost compile
  and link, about 7 s each. 35 mutants that fall through to whole-suite steps carry 52 % of tty's
  sweep seconds. A split shortens both, by different mechanisms.
- **Evidence.**
  - **The counts.** By the sweep's own attribution (`scripts/quality/mutant-verdicts.sh:142`, which
    accepts either quote, as does `targetRE` at `06_docs/mutants/mutants_test.go:47`), `modes/tty`
    has 539 mutants and `app` 153 (with RX-1). Within tty, files `map_*.go` have 289 and
    `broadcaster*.go` 80. TS's "the map's 18" and "the console's 66" (TS-3, TS-13) and the brief's 145
    and 110 count only double-quoted anchors: 18 + 66 + 61 = 145.
  - **Where the seconds go** (`dist/mutant-verdicts.timings`):

    | | Mutants | Seconds | Median |
    |---|---|---|---|
    | tty, ≤ 15 s | 504 | 3,815 | 7 s |
    | tty, 16-27 s | 29 | 549 | — |
    | tty, ≥ 474 s | 6 | 3,649 | — |
    | app, ≤ 15 s | 134 | 914 | 6 s |
    | app, > 15 s | 19 | 2,299 (1,232 and 604 among them) | — |
    | `platform/lineup` (13,719 lines with tests), all | 103 | 497 | 4 s |

    Three of tty's six slowest are the kept equivalents `mAA3` (`setup.go`), `mAA4`
    (`setup_tabs.go`) and `mSC3` (`router.go`). They survive, so they pay steps 2-4 (the package
    whole, `./...`, the package under `-race`). `app`'s 1,232 s is most likely `m16` (`app/severe.go`),
    its one survivor. The timings file carries no mutant names, so that pairing is inferred.
  - **What one fast-path mutant costs** (scratch copy, real mutants, load ~2.5). The test step is
    `go test -run '^<killer>$'`, run twice on the mutated tree and once restored (`06_docs/mutants/run.sh:93`,
    `:110`, `:137`):

    | Mutant | `go build ./...` | `go vet` | test step, mutated | test step, restored |
    |---|---|---|---|---|
    | `m119` (`modes/tty/map_temp.go`) | 1.45 s | 1.35 s | 1.96 s | 1.92 s |
    | `m120` (`modes/tty/map_temp.go`) | 1.46 s | 0.49 s | 1.92 s | 0.79 s |
    | `m109` (`app/radio.go`) | 1.11 s | 0.54 s | 1.83 s | 1.74 s |
    | `m100` (`platform/lineup/rotation.go`) | 1.53 s | 0.18 s | 0.69 s | 0.69 s |

    The tty test step breaks down as: compile the package with its tests, 1.02 s; link, 0.23 s;
    `go test`'s vet, 0.22 s; run one test, 0.43 s, nearly all of it process start. The test itself
    takes 0.00-0.07 s (the verdict log).
  - **Killers sit with what they kill.** 275 of the 289 map mutants are killed by a test in a
    `map_*_test.go` file, and 71 of the 80 console mutants by a `broadcaster_*_test.go` file (the
    verdict log's killer, resolved to its file).
- **Consequence (estimate).**
  - The fast-path cost per mutant fits about 3.5 s + 5 s × f, where f is the package's share of
    tty's 71,000 lines with tests. That gives 8.5 s at f = 1, against 7-8.6 s measured, and 4.4 s at
    lineup's f = 0.19, against 3.8-4 s.
  - Under TS's Map II (console f ≈ 0.15, map f ≈ 0.34, root f ≈ 0.5), with killers placed as today,
    tty's fast path falls from 3,815 to about 2,300-2,500 mutant-seconds. On four workers that is
    about 330-380 s of a 3,976 s sweep, roughly 9 %.
  - The survivors' steps 2 and 4 shrink with their package. Under Map II, `mAA3`, `mAA4` and `mSC3`
    stay in the root, so they roughly halve. `app`'s `m16` gains only if `severe.go` leaves the
    package that holds `app`'s three slowest tests (`ticker_test.go:661`, 49 s; `mapcost_test.go:102`,
    21 s; `cast_test.go:542`, 18 s).
  - The `go build ./...` second and a half is fixed, whatever the map.
  - Principle 1 for development: a gain, smaller than the suite-size ratio suggests. TG-13's "step 2
    … 14.9 s per tty mutant" and AS-10's "each mutant re-runs its own package's suite" read the mean
    as the typical case. The typical tty mutant runs one test, and the mean is the tail.
- **Recommendation.**
  - Keep each moved test file with the code its tests kill, so step 1 compiles the small package
    (RX).
  - The closing sweep is compared with A-3's per package, using a per-package summary of
    `dist/mutant-verdicts.timings` (count, seconds, median, the ≥ 100 s tail).
  - Record mutant names in the timings, so the tail can be attributed rather than inferred (TG).
- **Confidence.** High for the measurements and counts. Medium for the estimate: it is a linear fit
  over four points, and it assumes the killers move with their code.

### PB-7: Three allocation pins never run in `make verify` or CI's `race`, among them the Router-overhead pin a split leans on

- **Claim.** `TestRouterCostsObserverAlmostNothingPerFrame`, `TestLayoutOncePerViewAndKey` and
  `TestTheLineupRowsBuildNoDayCells` skip under the race detector, "measured … (make
  alloc-budget)". But `alloc-budget`'s pattern `AllocBudget` does not select them, and `verify`
  has no plain `go test` step. They run only in the docs lane and the sweep's baseline.
- **Evidence.**
  - The skips: `modes/tty/broadcaster_alloc_test.go:64-67`, `memo_test.go:122-125`,
    `broadcaster_loaded_test.go:184-187`.
  - `alloc-budget: go test -count=1 -run 'AllocBudget' ./...` (`Makefile:517-518`).
  - `verify-gates` (`Makefile:297`) lists `race` and `alloc-budget`, and no `test`. CI runs
    `make race` (`.github/workflows/ci.yml:55`).
  - The plain `go test -count=1 ./...` appears in `verify-docs-gates` (`Makefile:314`) and in the
    sweep's baseline (`scripts/quality/mutant-verdicts.sh:214`).
  - `TestTheAllocBudgetSelectsItsPins` (`cmd/watchpost/gates_test.go:386-403`) counts at least nine
    selected pins. It does not check that every race-skipped allocation test is among them.
- **Consequence.** Principle 1: the pin on the Router's per-frame cost over the Observer
  (`routerOverheadAllocs = 1`, `broadcaster_alloc_test.go:20`) is the seam a split moves, and a code batch can break it with `verify`
  green. The docs lane would then find it on some later commit.
- **Recommendation.** Before any code moves, bring the three under `make alloc-budget`, by name or
  by selector, and extend the gate test so a race-skipped allocation test outside the selector
  fails. This is a test and gate change, behind a failing case first (TG).
- **Confidence.** High.

### PB-8: The tone-latency instrument, which D-2's work would be judged by, reads 10x its last record, and its span includes test wiring

- **Claim.** `BenchmarkTimeToToneStart` measures a median of 15,761 ns per tone today. The last
  recorded figure is 1,628 ns (0.14.0, on an M3 Pro). The disagreement is ten-fold, and it has not
  been reconciled: no figure is recorded since 0.14.0. The timed span includes `newStation`, which
  builds a Director and its executors for the test.
- **Evidence.**
  - `bench-app.txt`: ten `ns/tone` values from 15,469 to 17,164, 185 allocs and 21.7 KB per op.
  - `06_docs/02_features/multi-voice-support/07-readiness/perf-protocol.md:15-16` holds the 0.13.0
    and 0.14.0 figures. A search of `06_docs` and `docs` for `ns/tone` finds nothing later.
  - `start := time.Now()` is followed by `newStation(b, deck).takeover(ctx, fresh)`
    (`app/tone_latency_test.go:99-100`). `newStation` builds `lineup.New` and `newExecutors`
    (`app/station_test.go:51-75`). The file last changed in 0.19.5 (`905ee845`).
  - The deck in the benchmark is `testArbiter(voice, nil)` (`tone_latency_test.go:94`): the arbiter
    D-2 would dissolve.
  - The run takes 222 s wall and 135 s system time for 20 benchmark runs, because each iteration's
    untimed setup makes a `TempDir` and loads a seen store.
- **Consequence.** Principle 1, for D-2: this is the instrument that would show the arbiter's
  dissolving costing a takeover anything. Without a reconciled baseline, a D-2 batch has nothing to
  be "the same" as. `perf-measurement.md`'s own rule applies: "when a new measurement disagrees with
  a recorded one by an order of magnitude, the disagreement is the finding".
- **Recommendation.**
  - Today's figure (15.8 µs median, 185 allocs, 21.7 KB) is 0.19.6's baseline for this instrument.
    It is re-taken back-to-back with the D-2 batch's tree, both `-count 10`, as perf-protocol §1's
    paired method asks.
  - Before D-2, the instrument is reconciled: what moved the span from 1.6 to 15.8 µs (the 0.16.0
    Director, the harness wiring inside the span, or the machine), and whether the wiring belongs
    outside the timed region. That is a test-only change, made first.
  - The allocation count (185) is a deterministic bound to hold alongside it.
- **Confidence.** High for the figures and the span. The cause of the ten-fold gap is UNVERIFIED.

### PB-9: The 0.19.6 performance metrics, their instruments, baselines and bounds

| ID | Measure | Instrument | Baseline (this lens) | Bound | Checked |
|---|---|---|---|---|---|
| P-1 | Bytes and allocations per beat: idle, radio, console | `TestBeatAllocBudget` log (`go test -count=1 -run AllocBudget -v ./modes/tty`, or the package that holds the Router after the split) | 168 / 75,988 B; 327 / 68,475 B; 377 / 122,387 B | allocs ≤ baseline; bytes ≤ baseline + 0.1 % | every code batch, figures in the build log (PB-1) |
| P-2 | No-change publish's frame | `TestPublishAllocBudget` log | 203 / 82,322 B | as P-1 | every code batch |
| P-3 | Frame and window pins, hit and miss (table A) | `make alloc-budget` logs | table A | count ≤ baseline | every code batch |
| P-4 | The three unselected pins (PB-7) | named run until they are selected | Router overhead within `routerOverheadAllocs`; layout once per view and key; lineup rows build no day cells | pass | every code batch |
| P-5 | Model boxes per message | `-gcflags=-m` on the Router's and surfaces' packages; model-typed escapes | 1 (`router.go:463`) plus the 2 `Init` lines | no new one on `Update` or `View` | every batch that moves a surface (PB-2) |
| P-6 | Model sizes | `unsafe.Sizeof` probe | Router 7,136; Dashboard 6,584; Broadcaster 472 B | no growth | every batch that moves state |
| P-7 | Inlining on moved files | `-gcflags=-m`, count of `inlining call to` in the moved files | e.g. 421 Dashboard-method sites in `map_*.go` | no fall without a stated reason | the batch that moves them (PB-3) |
| P-8 | Beat and frame wall time | the beat benchmark (to be added in the tree, tests only) and `BenchmarkFrame*`, `BenchmarkConsoleFrame*`, `-count 10`, benchstat | 70.1 / 84.2 / 173.8 µs; table B | no statistically significant regression | BUILD exit and the map batch; quiet machine |
| P-9 | Time to tone | `BenchmarkTimeToToneStart`, paired, `-count 10` | 15,761 ns median, 185 allocs (to reconcile first, PB-8) | allocs ≤ 185; median within the paired method's noise | the D-2 batch(es) and BUILD exit |
| P-10 | Start-up init | `GODEBUG=inittrace=1 watchpost --version` | tty 0.078 ms, app 0.10 ms, total 18.4 ms | new packages ≤ 0.1 ms each | BUILD exit |
| P-11 | Binary size | `make build` output | 32,732,082 B | ≤ +1 % | BUILD exit |
| G-1 | Sweep time, per package | `make mutant-verdicts`, the timings per package | 3,976 s; tty 8,013 / 539 (median 7 s); app 3,213 / 153 (median 6 s) | total ≤ baseline; tty fast-path seconds down (PB-6) | the closing sweep (A-3) |
| G-2 | `verify` time | `06_docs/verify-runs.md` (`runlog`) | median 2,418 s; last code run 2,061 s | ≤ baseline + 5 % (noise) | every code batch |
| G-3 | Per-package suite and race time | `go test -count=1 -json`, `-race -json` | tty 60.9 s / 224.6 s; app 135.4 s / 201.5 s | the largest package's time ≤ today's | PLAN, BUILD exit |

P-1 to P-7 and P-10 are deterministic and settle principle 1. P-8 and P-9 are wall clock and need a
quiet machine, so they are taken at fixed points, not every batch. G-1 to G-3 are the development
side of D-1's motivation, not the user's, and are reported, not used as bounds on correctness.

## Open questions for the HUM LEAD

1. **The pins' headroom for 0.19.6 (PB-1).** Options: A, the pins stay at ×1.05 and the release
   holds itself to the measurement in its build log; B, every pin is cut to its measurement for the
   life of 0.19.6, so the gate itself refuses any rise; C, as B, but the ×1.05 comes back at
   REFLECT. Recommendation: A for the gate, with the build-log comparison made mandatory per batch.
   The strongest counter-argument: a rule held in a log is held by discipline, and 0.19.5 L-rows show
   discipline slipping under load.
2. **Whether a beat wall-time benchmark lands in the tree (P-8).** Today the beat is pinned only in
   allocations and bytes, and its time lives in this lens's scratch file. This is a test-only
   addition. It is put here only because 0.19.6 is meant to add no instrument it does not need.

## Touches other lenses

- **TS:** the map and console mutant counts are 289 and 80, not 18 and 66 (PB-6). TS-11's "plain
  value context" is the condition PB-3 measures.
- **AS:** `windowLayers()`'s order is init order (PB-4). `m16`'s sweep cost depends on where
  `severe.go` and the three slow `app` tests land (PB-6).
- **RX:** RX-1's 692 agrees. A first test for the map layers' order (PB-4). Killer tests moving with
  their code (PB-6).
- **TG:** the three pins outside `alloc-budget` (PB-7). TG-13's step-2 mean (PB-6). Mutant names in
  the timings.
- **RM:** the tone instrument is D-2's yardstick and needs reconciling first (PB-8).
- **GP:** no `tea.Model` sub-models (PB-2 measures GP-1's claim).
