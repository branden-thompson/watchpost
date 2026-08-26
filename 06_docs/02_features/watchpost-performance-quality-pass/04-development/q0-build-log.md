# Q0 build log — Instrumentation and the measuring apparatus

Written for someone who has never seen the code (plan §0.5). Q0 changes no user-visible behaviour
except two additions to the `S` modal and one flag; its job is to make every later batch measurable.

| Field | Value |
|---|---|
| Batch | Q0 (plan §3) · branch `feature/watchpost-performance-quality-pass` · commits `112fe9c` … see below |
| Ships | as tooling with `v0.9.5` (Q1) — no release of its own |
| Gate | see §7; the 1-hour instrumented run's σ / detection floor decides the §1 growth bar |

## 1. What was built, and why (junior-first)

**The question the pass answers** is "does memory grow, and by how much per day?". Q0 gives the
process a way to *tell us* instead of us guessing from the outside.

1. **Request counters** (`platform/httpx/stats.go`). Every HTTP client already existed; each now keeps a
   small table of what it did since launch, one row per host: attempts (retries included), bodies
   received, cache hits, negative-cache hits, bytes, HTTP/2 responses, TLS handshakes. Eight named
   rows and an `other` row, so the table can never grow (P10-03). Keyed by hostname — never by URL,
   because a URL can carry a key (red-team IS-7). `CacheStats` gained the negative-cache count.
2. **Publish counters** (`app/dashboard.go` `publisher`). The two pipelines (favourites, RECENT)
   count their publishes and how many triggers the 50 ms window folded; the last snapshot pointer is
   kept so a dump can size it *then* (marshalling every publish would be the churn we are removing).
3. **Gauges** (`app/stats.go`). Every bounded structure in the process reports its length and bytes
   through one accessor on its owner (`nws.CachedGrids`, `hms.MemoPoints`, `coops.CachedStations`,
   `synth.Source.Cached`, `tz.Cached`, `Assembler.Size`, the httpx tiers, three directories). "Flat"
   is proven per structure, not per footprint (red-team RT-5).
4. **The dump** (`app/dump.go`). `kill -USR1 <pid>` (Unix) or `GET /debug/dump` (the opt-in loopback
   server, every OS) writes `profiles/<UTC ts>/` with the four runtime profiles and `counters.json`:
   post-GC `MemStats`, goroutines / threads / fds, the counters above, the gauges. Bounds: one in
   flight, a minute apart, twelve kept, 0700/0600, never `debug.WriteHeapDump`. `/debug/counters`
   serves the same JSON live — that is what `soak.sh` reads every sample.
5. **The `S` modal** (`modes/tty/dashboard.go` `requestLines`) shows the REQUESTS table (63 cells so
   it fits the 68-wide modal without wrapping — UAT 25) and the DUMPS line; `report --verbose`
   appends one plain line per host (`modes/report` `RenderRequests`).
6. **The statistic** (`tools/slope`). Reads `soak.csv`, takes the per-hour *minimum* of the post-GC
   heap (so in-flight bodies and HMS-parse transients drop out), fits a line, and reports the slope
   with a Newey–West (autocorrelation-robust) 95 % interval, the 30-day projection of its upper edge,
   and the **detection floor** — the smallest 30-day growth this run could have certified. Verdicts:
   PASS · GROWTH · UNCERTIFIABLE (the floor is above the bar: say so, never wave) · INSUFFICIENT.
7. **The harness** (`scripts/quality/`): `soak.sh` (macOS vmmap / Linux smaps_rollup + counters, hourly
   dump), `dump.sh`, `p10-unmatched.sh`; `make p10` (fails loud without the CLI), `make alloc-budget`
   (malloc pins outside `-race`), `make quality-bench`, `make tidy` / `make vuln` in `verify`.
8. **Benchmarks and pins**: in-package `bench_test.go` for the frame (three sizes, colour on, `TERM`
   set), HMS parse and render primitives; `TestFrameAllocBudget` pins allocations per frame.

## 2. Files touched

| Area | Files |
|---|---|
| httpx | `stats.go` (new), `stats_test.go` (new), `httpx.go` (counting points, `attemptResult`, `handshakeTrace`), `cache.go` (`Stats.Negative`) |
| app | `stats.go`, `dump.go`, `dump_unix.go`, `dump_windows.go`, `dump_test.go` (new); `dashboard.go` (publisher counters, wiring, `startStaggered` split, `/debug/counters`, `/debug/dump`); `app.go` (`ReportOnceWithStats`, `reportFetch`) |
| accessors | `platform/snapshot/assembler.go` `Size`; `domains/weather/nws/provider.go` `CachedGrids`; `domains/fire/hms/hms.go` `MemoPoints`; `domains/marine/coops/coops.go` `CachedStations`; `domains/radio/synth/source.go` `Cached`; `platform/tz/tz.go` `Cached` |
| tty / report / cmd | `modes/tty/dashboard.go` (`Stats`, `PipelineStats`, `Config.Stats`, `requestLines`, `pipelineLine`); `modes/report/report.go` `RenderRequests`; `cmd/watchpost/root.go` `--verbose`; `platform/render/render.go` `HumanBytes` |
| benches | `modes/tty/{bench,race,norace}_test.go`, `domains/fire/hms/bench_test.go`, `platform/render/bench_test.go` |
| tools / scripts | `tools/slope/{main,main_test}.go`, `scripts/quality/{soak,dump,p10-unmatched}.sh`, `Makefile`, `.github/workflows/ci.yml` |
| docs | `README.md` (Diagnostics), `CHANGELOG.md` (`[Unreleased]`), `04-development/infra-ledger.md`, `07-readiness/soak-profile.md`, `07-readiness/p10-q0.json`, this log |
| record | `02-analysis/discover-run/`, `lens-benches/`, `red-team-perf/`, `red-team-round2/` (artefacts out of the session scratchpad — BQ-5) |

**Docs touched** (plan §0.5): README diagnostics paragraph replaced; CHANGELOG `[Unreleased]` added;
`docs/caching.md` unchanged (Q1 rewrites it); `architecture.md` unchanged (Q2).

## 3. Tests first (the pins)

| Test | Pins |
|---|---|
| `httpx.TestRequestStatsCountByHost` | attempts / net / cache / negative per host; keys carry no `/` |
| `httpx.TestRequestStatsCountEveryAttempt` | retries count as attempts, one net success |
| `httpx.TestRequestStatsSlotsOverflowToOther`, `…BusiestFirstOtherLast`, `TestMergeRequestStats` | the bound, the order, the merge |
| `httpx.TestRequestStatsSeeHTTP2AndTLSHandshakes` | h2 and handshake counters over a TLS httptest server |
| `app.TestDumpWritesProfilesAndCounters` | five files, 0700/0600, counters.json content, the note |
| `app.TestDumpRateBound`, `…KeepsOnlyTheNewest`, `…RequiresADirectory` | a minute apart; twelve kept; no dir = error |
| `app.TestDirGauge`, `TestPublisherCountsPublishesAndFoldedTriggers` | directory sizing; publish/folded counts |
| `tty.TestStatusModalShowsRequestAndDumpRows`, `TestStatusModalWrapsNeverTruncates` (updated) | the rows, "none yet", absent without a hook; no truncation |
| `report.TestRenderRequestsIsOneLinePerHost` | the plain trailer |
| `render.TestHumanBytesFitsSixCells` | the byte column |
| `slope.TestQuietFlatHeapPasses`, `…NoisyFlatHeapIsUncertifiable`, `…OneMegabytePerDayIsGrowth`, `…BucketMinimumIgnoresTransients`, `…InsufficientDataIsSaidOutright`, `…TQuantileTable`, `…OLSRecoversALine` | the statistic's four verdicts and its parts |
| `tty.TestFrameAllocBudget` | allocations per frame at three sizes (non-race) |

## 4. Before / after (Q0 changes no hot path; the numbers are the baseline every later batch diffs)

| Quantity | DISCOVER (L5, Go 1.25) | Q0 measurement (Go 1.27, `make quality-bench` fixture) | Δ |
|---|---|---|---|
| frame 133×44, colour on | 670 µs / 437 KB / 10,044 allocs | **681 µs / 437 KB / 10,044 allocs** | +1.6 % time, allocs identical (gate: within 10 %) |
| frame 133×70 | — | 15,539 allocs | new pin (× 1.05) |
| frame 200×60 | 724 µs (ladder E, L5) | 20,031 allocs | new pin (× 1.05) |
| HMS parse 27.5k | 1.05 M allocs / ~75 MB (L1) | 104.4 ms / 75.0 MB / 1,045,401 allocs | reproduces |
| Overlay (modal compositor) | 786 µs / 795 KB (L5-F9) | 776 µs / 797 KB / 2,092 allocs | reproduces |
| `NewTextFormatter` (the per-frame `/dev/tty` probe, L5-F1) | — | 32.8 µs / 208 B / 7 allocs | the Q4a-001 target |
| `LocationTable` 50 rows w129 / w196 ext | — | 727 µs / 388 KB / 15,579 allocs · 1,413 µs / 725 KB / 26,161 allocs | the Q3 body-memo target |
| frame 133×44 + Help modal | — | 1,630 µs / 1,323 KB / 13,071 allocs | modal path |
| `Assembler.Snapshot` 25 locations | RS-20 | 163 µs / 995 KB / 312 allocs | the per-publish cost |

Full `make quality-bench` output (count 10): `02-analysis/q0-bench.txt`.
| P10 | 121 findings / 0 live / ledger 132 (56 non-kit) | 123 findings / 0 live / ledger 131 (**55 non-kit**) | +2 budgeted directive entries, +1 package-density entry on `tools/slope` (§7), −4 dead entries (`app/setup.go` View/Update/Save, `resample.go Read` — SC-5's stale four, pulled forward from Q1 because `p10-unmatched.sh` proved them dead) |
| `go.mod` | 4 direct deps with no importer | tidy (`make tidy` green) | PH-1 closed |
| vulnerabilities | unchecked | `govulncheck`: none | IS-9 closed |

## 5. Bounds stated (plan §0.8)

| Structure | Owner | Bound | Pinned by |
|---|---|---|---|
| `RequestStats` | `httpx.Client` | 8 hosts + other | `TestRequestStatsSlotsOverflowToOther` |
| dump directory | `app.dumper` | 12 dumps; ≥ 60 s apart; one in flight | `TestDumpKeepsOnlyTheNewest`, `TestDumpRateBound` |
| signal loop | `app/dump_unix.go` | 10,000 dumps per process | constant `maxDumpsPerRun` |
| publish counters | `app.publisher` | two atomics + one pointer | — |
| gauges | `app.diagSources` | fixed list; directories sized on demand | `TestDirGauge` |

## 6. Decisions and non-decisions

- **The DISCOVER instrumented run was stopped at 4 h 08 m** (of a planned 6 h) because the Q0 gate run
  needs the same loopback port (6060) and the new binary; its samples and profiles are in
  `02-analysis/discover-run/` (LR-6 covers what it showed). The Q0 soak supersedes it.
- **`tools/slope` is Go, not Python** (R2-17): one language, no scipy; the t-table is embedded.
- **`Stats` lives in `modes/tty`** so `app` can hand it in without a cycle; `app` imports `tty` already.
- **`profileNames` is a function** and the prune loop is bounded by the surplus: P10-06/P10-02 clean without
  exemptions. `startRecent` was split (`startStaggered`) to stay under 60 lines.
- **`HumanBytes` is six cells max** so the REQUESTS row stays at 63 cells and never wraps (UAT 25).
- **Pulled forward from Q1**: the four stale exemptions (`app/setup.go` View/Update/Save,
  `resample.go Read`) are deleted here — `p10-unmatched.sh` proved them dead and the Q0 gate requires
  unmatched 0. Q1 keeps the two one-liners and the kit-entry reason rewrite.
- **Platform-tagged ledger entries** (`*_windows.go`, `*_unix.go`) are checked on their own OS;
  `p10-unmatched.sh` says "skipped (checked on windows)" rather than calling them dead (SC-3).

## 7. Gate

| Check | Result |
|---|---|
| `make verify` (fmt, vet, tidy, vuln, race, lint-imports, lint-watermark, gate-controls) | ALL GATES GREEN |
| `make alloc-budget` (CI, non-race) | green — 10,042 / 15,539 / 20,031 allocs vs budgets × 1.05 |
| `make p10` (local) | 0 live · unmatched 0 (four dead entries deleted; `dump_windows.go` checked on Windows) · snapshot `07-readiness/p10-q0.json` |
| `a2dh validate` | 18/18 |
| bench reproduces DISCOVER within 10 % | yes — 681 µs vs 670 µs; allocs identical |
| ≥ 1 h instrumented run → σ, detection floor vs the §1 bar | apparatus proven; the floor question is answered honestly in §8: a 1 h run cannot certify a 30-day bar, and the 72 h bar must be restated from measured σ (decision for HUM LEAD) |

**Exemption delta for HUM LEAD (plan §1: any exception is a gate decision).** Budget: +2 directive
entries. Actual: +2 directive (`app/dump_unix.go`, `app/dump_windows.go`) **+1 package-density entry
on `tools/slope`** (the same pattern as `tools/alertrec` and `tools/nwrtable`: the checked paths carry
invariants, the arithmetic helpers do not) −4 dead entries. Non-kit count 56 → 55; Q1's two
one-liner deletions bring it to **53**, one above §1's 52. Options: (a) ratify the slope entry and
restate §1 as ≤ 53; (b) refuse it, and the slope tool gains quota checks on pure arithmetic (the
pattern the P10 skill itself warns against). Recommendation: (a). **HUM LEAD ratified (a),
2026-08-26: plan §1 and objectives M4 now read ≤ 53.**

## 8. The 1-hour run (Q0 local gate)

Idle dashboard (radio off, viz off, 10 favourites + 50 recent, 133×44), `WATCHPOST_DEBUG_PPROF=1`,
2026-08-26 14:49–15:51 UTC, PID 83436, 120 samples at 30 s (`02-analysis/q0-soak-1h.csv`), one dump
(`profiles/20260826T144922Z`, 5 files, 57 KB).

**What the apparatus showed (every column populated, every structure gauged):**

| Series | First hour |
|---|---|
| post-GC `heap_alloc`, 5-min minima | 37.9 · 33.8 · 34.8 · 35.7 · 36.9 · 36.8 · 36.4 · 35.3 · 38.7 · 35.0 · 34.2 · 37.7 MB — **flat from minute 5** (range 33.8–38.7, raw sd ≈ 1.6 MB) |
| post-GC `heap_alloc`, raw samples | 33.8–66.8 MB (the sawtooth between GCs and parse transients — what the minimum removes) |
| footprint (vmmap) | 89 → 99–104 MB, plateau ≈ 100 MB (RSS 142 MB) — the §1 idle row (≤ 90 MB) is **not met by the current code**, as expected before Q3 |
| threads | 24 for 29 min, then 26 (the ratchet LR-3 describes: +2 in an hour, radio off) |
| goroutines | 273–302, mode 274 (bounded: 7 + 50×5 + alerts + tea + I/O) |
| fds | 14–17 |
| disk cache | 1,428 → 1,429 files, 114.9 MB (one date-keyed URL per hour — L4-F1's slow growth, Q1 closes it) |
| RECENT publishes | 392 in the hour ≈ 6.5/min (the number §1's allocation row needed — R2-7) |
| dump | `counters.json` carries every gauge (`httpx.mem.entries`, `nws.gridinfo`, `hms.memo.points`, `tz.cache` 8, `disk.*` …) |

**What `tools/slope` said** (`-warmup 10m -window 5m`, 11 buckets):
`slope=+198 MB/day, se=144 (NW lag 2), 95 % CI [−128, +524]; plateau 36.4 MB, bar 1.8 MB; projection 15,719 MB, floor 9,784 MB → UNCERTIFIABLE (exit 3)`.

This is the tool working as designed, not a defect: a one-hour lever arm (x spans 0.04 days) cannot
resolve any 30-day slope; the floor formula `t · SE · 30 d` is honest about it. The Q0 gate line
"detection floor over ≥ 1 h below the bar" (plan Q0 gate, R2-1's amendment) was mis-sized — the
apparatus can be proven in an hour, the floor cannot.

**What this means for §1's growth bar (a HUM LEAD decision at this gate).** With hourly minima over
72 h (n ≈ 66, x spanning 2.75 d, Sxx ≈ 41.6 d²) the floor is ≈ 9.3 × σ_hourly-minimum MB per 30 days. If
σ is the 1.6 MB seen here, the 72 h floor is ≈ **15 MB / 30 d** against a 5 % bar of ≈ 1.8 MB on a
36 MB post-GC plateau: the 72 h soak would return UNCERTIFIABLE on a perfectly flat process. Options:

1. **Certify at the floor, say so** — the §1 criterion becomes "upper 95 % CI × 30 d < max(5 % of
   plateau, the run's measured floor)", the floor printed beside every verdict, and the per-counter
   zero-tolerance check (every gauge flat at 24/48/72 h) stays the primary evidence of "no growth
   term"; the slope is the secondary bound. Honest, and what the plan already half-says.
2. **Lengthen the macOS soak to 7 days** (floor ≈ 3.8 × σ ≈ 6 MB) — this machine can host it; Arch stays
   at 72 h.
3. Both. **Recommendation: 3** — restate §1 per option 1 now, and schedule Q7's macOS soak at 7 days.

A **24-hour idle soak is running now** (PID 69828, started 15:55 UTC, 5-minute samples, hourly dumps)
to measure σ of the *hourly* minima before Q1 ships; its number replaces the 1.6 MB estimate above in
the Q1 build log and sets the floor figure the restated criterion will carry.
