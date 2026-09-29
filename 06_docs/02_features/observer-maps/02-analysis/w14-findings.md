---
title: "0.18.0 W14 — the baseline and the audit's findings"
date: 2026-09-29
phase: BUILD
authority: HUM LEAD
status: "OPEN — every finding here is to be ruled or fixed before SHIP (W14). Nothing is changed until ruled."
---

# W14 — what was measured, and what was found

D-154 ruled the order: measure first, audit read-only, then batches — structure, performance,
resources — each re-measured and gated. This is the first two, as they stood on 2026-09-29.

## The baseline (standard workload v1, commit `ef088a21`, Darwin arm64)

Raw summaries: `06_docs/perf/workload-v1/baseline-2026-09-29/`. Method: `06_docs/perf-measurement.md`.

| | `default` | `heavy` |
|---|---|---|
| **Cold M5**, map opened at first data (n = 5) | median **23.2 s** (22.6–23.3) | **not reached in 45 s, 0 of 5** |
| Cold M5's first alerts answer | median 20.4 s | 28–43 s |
| Cold `complete` (the basemap whole) | median 328 ms | ~300 ms |
| Cold radar answered | median 1.6 s | 1.7 s |
| M5, station warm, map cold (session C, n = 1) | 7.9 s | not reached |
| M5 after a pan, with D-66's 600 ms settle (n = 39) | median 0.89 s, p90 1.5 s | 3 of 39 reached |
| Key to frame produced (n ≈ 85) | median 38 ms, p90 99 ms | 41 / 93 ms |
| The map clock's lateness, loop playing (n = 540) | median ≤ 1 ms, max 455 ms | ≤ 1 ms, max 382 ms |
| CPU: idle · radio on | 1.3 % · 1.3 % | 1.2 % · 1.5 % |
| CPU: map at the place, a pan every 15 s · national, loop playing | 7.7 % · **12.9 %** | 5.3 % · 3.5 % |
| Footprint: idle · map open · **after the map closed** | 90 · ~230 · **166 MB** | 92 · ~220 · **169 MB** |
| Heap: idle · after the map closed | 34 · **86 MB** | 34 · **86 MB** |
| Goroutines, idle → the run's end | 484 → 506 | 484 → **550** |

**Read with care.**
- **Cold M5 is 6.6× D-46's 3.5 s, and five cold opens sit within ±0.4 s** — a fixed queue, not the
  network.
- **The instrument's own two flaws**, found reading this run: `settled` after a *move key* fires
  before the move is marked (`handleMapKey` draws, then calls `viewMoved`), so the pan rows'
  `settled` are premature — their `m5` is sound; and `answered:*` under a trigger that outlives
  its ask (`map.play`) times later refreshes from the key press (the 300 s rows). Both are
  instrument fixes, owed before the next measurement.
- **The 12.9 % vs 3.5 % loop CPU is unexplained** — recorded, not interpreted.
- Warm opens are not in the table: their windows (35 s) were too short for M5 to be reached under
  the snapshot re-asks (finding P-1).

## The findings

Six read-only auditors (the map in `app/` and `modes/tty/`, the station's data fetching, the
library, P10's triage, DRY across the rest). **Every row marked Verified was read at its lines by
the builder after the audit; Reported rows are the auditor's, not yet read.** A claim that did not
hold is listed at the end.

### Correctness — defects, first in any order

| # | Finding | Evidence | Status |
|---|---|---|---|
| C-1 | **Under the heavy workload the map never finishes.** Every priority snapshot re-asks the whole feed (`dashboard.go` `SnapshotMsg`), a newer generation drops the older answer (`applyMapFeed`), and the older ask is never cancelled — it runs to its 30 s limit | Baseline: 0 of 10 heavy opens reached M5; goroutines 484 → 550 | Verified |
| C-2 | **The Tides estimate is always 0**: the estimate is built without fetching, and only fetching fills `in.tides` (`mapmarine.go` `tideLayerCost`, `maplayers.go` `inputsFor`) | Code | Verified |
| C-3 | **The Overlays menu's picker blink has no tick of its own**: `tickNeeded` has no arm for `menuFlash` (Settings' has one) | Code; whether it shows depends on another tick being armed | Verified in code |
| C-4 | **~76 MB stays after the map closes** (heap 34 → 86 MB) | Baseline, both variants | Measured; cause not traced |
| C-5 | Two disk-cached providers (USGS near-field, NDBC) never `Forget` a body that failed to parse, so it is served again until its TTL | Auditor | Reported |
| C-6 | A nil-pointer panic reachable only if the embedded index fails and no recent list is saved (`markFIRMS`, `fireFor`, … on `rp.asm`) | Auditor | Reported |

### Performance — the map's responsiveness and data fetching

| # | Finding | Evidence | Status |
|---|---|---|---|
| P-1 | **The map's requests wait behind the station's launch burst.** The map's work contexts are `context.Background()` (`map_work.go`), so alerts in view, zone shapes, fire, quakes, AirNow go on the data client's normal lane — the RECENT pipeline's launch burst's lane (~375–600 requests at 30/s, FIFO). Only favourites use `WithPriority` | The ±0.4 s spread of cold M5 | Verified (lane); burst size estimated from code |
| P-2 | **Every move key asks the whole feed at once, and again 600 ms later.** `handleMapKey` batches `mapFeedCmd` on every pan, zoom and region key; D-66's settle tick then asks again. The per-key ask predates D-66 (batch 5) and survived it (batch 15) | History; code | Verified |
| P-3 | **Fire and quakes are fetched on every feed ask whatever their switches** (`inputsFor` → `fireIn`, `mapQuakes.fetch`); buoys, tides and air check theirs | Code | Verified |
| P-4 | Zone shapes resolved for alert categories switched off (the estimate skips them, D-149; the feed does not) | Auditor | Reported |
| P-5 | The feed's inputs are fetched one after another — view alerts, fire (perimeters box by box), quakes, buoys, up to 20 serial tide predictions, AirNow — then the zones | Auditor, two independently | Reported |
| P-6 | **Seven overlays are stamped `Valid: now`** (fire ×3, AirNow, buoys, tides, quakes), so no answer compares unchanged and each is handed in again | Code | Verified |
| P-7 | Big bodies re-parsed on every ask though cached (AirNow 1.9 MB, perimeters, NDBC) | Auditor | Reported |
| P-8 | `time.LoadLocation` (a zoneinfo read) per alert sentence in the map's description; `platform/tz` exists for this | Code | Verified |
| P-9 | Double renders: `retime` → `setFeed` → `renderMap`, then the caller's own — every forecast step and playback tick | Auditor | Reported |
| P-10 | The library repaints fully when the blink phase flips, though watchpost places no blinking marker | Code (`frame.go` compares the phase unconditionally; watchpost sets no `Blink`) | Verified mechanism; cost unmeasured |
| P-11 | The library's report memo is cleared by the wall clock every frame, and `Legend()` rebuilt every Update | Auditor | Reported |
| P-12 | The library's loop step: 128 raster samples a cell with the projection recomputed; the radar PNG decoded a pixel at a time through two interface conversions | Auditor | Reported; bears on the 12.9 % |
| P-13 | The station pool and the watchlist can fetch one place twice; `Retain` drops the pool's resolutions on every commit; one gridpoint document decoded twice a cycle | Auditor | Reported |

### Structure — P10, DRY, simplification

| # | Finding | Evidence | Status |
|---|---|---|---|
| S-1 | **61 live P10 findings, not 100** (39 exempted). 29 of them are one false edge: the `scrolls` closure in `windowKeysOf` (`window_keys.go`), which the analyzer reads as a call; a method expression, as the neighbouring rows use, dissolves the 32-function "cycle" | Two auditors independently; the closure read | Verified |
| S-2 | **The P10 checker now keys methods `Recv.Name`**; ledger rows keyed by a bare method name stopped matching — 9 live now (Router's, `Dashboard.View`, `Engine.watchClip`), more as their code is touched | Auditor, from the harness source | Reported — a ledger edit, the HUM LEAD's |
| S-3 | One real 2-cycle: `layerOn` ↔ `radarMode`, bounded at depth 2 | Two auditors | Reported |
| S-4 | Six loops bounded in fact, in a form the checker cannot see | Auditor | Reported |
| S-5 | The map's Forecast-mode step loop written 6×, its hourly loop 4×, grid building 3×, the radar loop fetch 2×; the tty's overlay reconcile 3×, its worker-command wrapper 4× | Auditor | Reported |
| S-6 | Unit conversions (°C/°F, km/mi, m/s) in ~20 places across three layers; the km/mile constant 6×; two spoken wordings already disagree ("kilometres" / "kilometers") | Auditor | Reported |
| S-7 | Control components hand-built where the app's exist (the Request window's checkbox — D-147); `cycleIn` bypassed 4× | Auditor | Reported |
| S-8 | Code no production path reaches: F-24's on-air masthead, `castProblems`, `ellipsize`, `colorEnabled`, `bedReachFor`, `withForecast` | Auditor | Reported — F-24 is a HUM LEAD question (wire or delete) |
| S-9 | Functions past ~60 lines: `handleMapKey` 102, `overlaysBox` 96, `mapFeedWith` 86, `geojson.walk` 77, `ttyConfig` 74, `parseDWML` 72 | Auditor | Reported |

### Claimed, and did not hold

| Claim | What was found |
|---|---|
| `geo.CompassIndex` returns -1 for a negative bearing and callers panic | True of the arithmetic below about −67°, but every caller passes `[0, 360)` — `BearingDeg` normalises, the feeds' directions are compass degrees. A hardening, not a live defect |
| The library repaints twice a second with nothing blinking | Its clock already schedules no blink tick then; the waste is confined to draws that happen for other reasons (P-10) |
