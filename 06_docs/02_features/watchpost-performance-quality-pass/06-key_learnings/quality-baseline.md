# Quality baseline — every number the pass made, with its command and source

**Status:** DRAFT (Q7 in progress) — rows marked _7-day_ fill when the macOS soak ends (2026-09-03) and
_Arch_ when HUM LEAD's run returns. Everything else is final and reproducible from the record.

This is the corpus the plan promised (§1, C5, ADR-06): the plan's targets, the number before the pass,
the number after, the command that takes it, and where the raw record lives. Read
`reading-profiles-and-soak-logs.md` first if `pprof` and soak CSVs are new to you.

## 1. The §1 targets

| Target | Before (DISCOVER, 2026-08-25) | After (batch) | How to take it | Record |
|---|---|---|---|---|
| Frame, memo hit (every tick/marquee/viz frame), 133×44 colour on | 660 µs · 436 KB · 10,044 allocs | **121 µs · 61 KB · 504 allocs** (Q3, Q4a) | `make quality-bench` → `BenchmarkFrame_133x44` | `02-analysis/q4a-bench.txt` |
| Frame, memo miss (table re-render) | same | **445 µs · 350 KB · 8,445 allocs** (Q4a) — the ≤ 470 µs line met | `BenchmarkFrame_133x44_Miss` | same |
| Allocation pins (CI) | — | hit ≤ 6,000 / miss ≤ Q0 × 1.05 at three sizes | `make alloc-budget` | `modes/tty/bench_test.go` |
| Idle allocation rate, tick off, radio off | ≈ 94 MB/min (the tick ran forever) | **≈ 9 MB/min**, render's share 0 (no frame drawn) | soak `total_alloc` Δ/min | `02-analysis/q3-alloc-1h.csv` |
| Radio-on render | ≈ 140 MB/min | ≈ 12 MB/min on the tick; ≈ 74 MB/min with the visualizer (20 fps × 62 KB) | same | same |
| RECENT publishes | ≈ 670/h (Q0 24 h soak: 11,253 in 16.8 h) | **33–37/h** (one per tier wave) | `counters.json` `recent.publishes` | q3/q5/q6 soaks |
| HMS archive parse | 104 ms · 75 MB · 1.05 M allocs (27.5k points); the inflated file held | 88–93 ms · 33 MB · 605k; streamed | `go test ./domains/fire/hms -bench Parse27k` | `q3-bench.txt`, `q4a-bench.txt` |
| Requests/hour (10 + 50, healthy) | NWS 691 attempts / 689 net / 24.6 MB (Q1 hour) | NWS **652 / 537 / 22.2 MB with 102 renewals** (Q5 hour); FIRMS 270 on tiles (≈ 420/h per location before); re-derived floor NWS ≈ 790, FIRMS ≤ 324 | `counters.json` per host | `02-analysis/q5-counters/counters-1h.json` |
| Bytes saved by 304 | 0 (no send side) | NWS 1.54 MB + NDBC 1.50 MB per hour | `NotModified`, `Bytes304` | same |
| Disk cache | 1,376 files / 116 MB after two days, unbounded | 738–818 files / 35–39 MB, flat; ≈ 461 writes/h (was ≈ 45k/day) | `disk_files`, `disk_bytes`, `httpx.disk.writes` | q1/q5/q6 soaks |
| Warm launch → full view | 550 ms (one run) | **790 ms median of 10** (2026-08-27, machine shared with a soak) — _re-take pending_ | `WATCHPOST_DEBUG_TIMING=1` × 10 | q7-validate-log §2 |
| P10 | 56 non-kit / 76 kit exemptions, gate absent | **0 live · 0 unmatched · 54 non-kit / 55 kit** (11 kit rows retired by Q4a patches) | `make p10` | `07-readiness/p10-q*.json` |
| Threads | 30–32 | 23–25 (soaks), bound by construction ≈ 34 | soak `threads` | q5/q6/q7 soaks |
| Growth term — memory | unknown | _7-day_: `slope` verdict at day 7, σ and floor printed | `go run ./tools/slope -in soak.csv` | `02-analysis/q7-soak/` |
| Growth term — structures | disk tier, neg-cache, gridInfo unbounded | every gauge flat against a stated bound (below); _7-day_ table | `counters.json` gauges | same |
| Idle footprint / plateau | 78 MB / 98–175 MB | soaks: RSS 80–140 MB, footprint 95–160 MB (fragmentation check only); _7-day_ per phase | soak `rss_kb`, `footprint_kb` | same |

## 2. The measuring apparatus (Q0) and its one defect

`httpx.RequestStats` (8 host slots + other), `app.Stats` and the gauges, `counters.json` on `SIGUSR1`
/ `/debug/dump`, `/debug/counters` live, `[S]` REQUESTS/DUMPS rows, `report --verbose`,
`tools/slope`, `scripts/quality/{soak.sh, soak-phases.expect, dump.sh, p10-unmatched.sh}`, in-package
benchmarks and the allocation pins.

**The defect (found at Q7):** `/debug/counters` read memory without a GC until `0.10.1`; only the dump
collected first. Every 5-minute sample before the fix is pre-GC; the hourly dumps are the post-GC
series. The Q0 24 h soak is kept as the record of it.

## 3. The Q0 24-hour soak, read correctly

_Final numbers at the run's end (2026-08-27 15:55 UTC)._ At 17 h: the post-GC heap across dumps
56.0 → 57.1 → 61.0 MB, stepping with `hms.memo.points` 77k → 81k → 95k and flat between steps; the
CSV's pre-GC series said GROWTH (+25.5 MB/day); the same series past the last step said +0.4 MB/day,
UNCERTIFIABLE. The sites (`hms.Parse` → `io.ReadAll`, `xml.copyValue`, `parseDescription`) are the
ones Q3 retired.

## 3a. The pre-pass process, sampled for 26 hours (the "before" the plan asked for — OQ-3)

HUM LEAD's own `v0.9.4` session (PID 67943, launched 2026-08-25, radio in use) was sampled every 5
minutes from 2026-08-26 10:21 UTC to 2026-08-27 12:09 UTC — 291 samples, `ps`/`vmmap` only, never a
process listing (RT-15): **RSS 123–455 MB (last 148), physical footprint 116–242 MB (last 160),
threads 30–39 (last 37)**. That is the footprint band the plan's §1 rows 1–2 were written against
(78 MB idle / 98–175 MB radio-on) and the thread ratchet the bound by construction (≈ 34) replaces.
The pass's soaks on `v0.10.x` run at RSS 80–140 MB, 23–25 threads. Record:
`02-analysis/baseline-pid67943.csv` (the `.log` is gitignored by pattern; the CSV is the committed copy).

## 4. Bounds, stated (every memo, cache, counter the pass introduced or touched)

| Structure | Owner | Bound | Gauge | Pin |
|---|---|---|---|---|
| HTTP memory tier | `httpx.cache` | 8 MB / 4,096 entries, expired-first then LRU; 24 h grace for validators | `httpx.mem.entries` | Q1 tests |
| HTTP disk tier | writer goroutine | > 5-min TTL or `Persist()`; sweep: allow-list, `max(expires, mtime)+24 h`, 256 MB, ≤ 10k/≤ 1k per pass | `disk.cache`, `httpx.disk.writes` | Q1 tests |
| negative cache | `httpx.cache` | 1,024, soonest-expiring out | `httpx.neg.entries` | Q1 |
| failure memo | `httpx.memo` | 16 hosts, 20 s; Retry-After ≤ 5 min | — | Q1 |
| body memo | `tty.bodyMemo` | one slot | — | Q3 memo tests |
| tick | `tty.tickNeeded` | runs only while animating | — | Q3 tick tests |
| HMS / WFIGS parse memo | `fire.Memo` | one body each | `hms.memo.points/parses`, `wfigs.memo.incidents/parses` | Q3 |
| FIRMS tile memo | `firms.tileMemo` | 240 tiles, LRU | `firms.memo.tiles/parses` | Q5 |
| grid cache | `nws.Provider` | one per tracked location, 24 h, `Retain` | `nws.gridinfo`, `nws.grid.decodes` | Q5 |
| RECENT publisher | `app.publisher` | 5 s window | `recent.publishes/folded` | Q3 |
| scheduler cadence | `sched.runTier` | fixed grid | — | Q3 |
| synth PCM cache | `synth.Source` | 40 segments ≤ 29 MB | `synth.pcm.cache` | B4 |
| dumps | `app.dumper` | ≥ 1 min apart, newest 12 | `disk.profiles` | Q0 |

## 5. The cost of C3 = A (recorded, not paid twice)

50 RECENT schedulers × 5 tiers = 250 parked goroutines ≈ 1 MB; goroutines flat at 273–280 in every
soak; RECENT publishes 33–37/h once the grid cadence and the 5 s window landed. The §2.4 threshold did
not trip (Q6 §6.1).

## 6. Accepted non-decisions

`SetMemoryLimit` (ADR-05); the shared disk cache for the two clients (Q5 §6.2); the two alert
schedulers (Q5 §6.3); compass 8 → 16 (excluded); `Tok()`-time precompute (Q4a §6.1); the `layoutFor`
package memo (Q3 §6.1); the bespoke HMS byte scanner (Q3 §6.3).

## 7. The 30-day extrapolation table

_7-day: per counter and gauge, the day-7 value, its slope over days 1–7, the 30-day projection, and
the bound it must stay under._

## 8. Reproducing any row

Every command above runs from the repository root on a checkout of the batch's tag; the soak commands
need a terminal of 133×44 and `WATCHPOST_DEBUG_PPROF=1`. The raw files are under
`06_docs/02_features/watchpost-performance-quality-pass/02-analysis/`.
