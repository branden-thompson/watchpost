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
| C-1 | **Fixed, batch 59 (D-157).** **Under the heavy workload the map never finishes.** Every priority snapshot re-asks the whole feed (`dashboard.go` `SnapshotMsg`), a newer generation drops the older answer (`applyMapFeed`), and the older ask is never cancelled — it runs to its 30 s limit | Baseline: 0 of 10 heavy opens reached M5; goroutines 484 → 550 | Verified |
| C-2 | **Fixed, batch 58.** **The Tides estimate is always 0**: the estimate is built without fetching, and only fetching fills `in.tides` (`mapmarine.go` `tideLayerCost`, `maplayers.go` `inputsFor`) | Code | Verified |
| C-3 | **Fixed, batch 58.** **The Overlays menu's picker blink has no tick of its own**: `tickNeeded` has no arm for `menuFlash` (Settings' has one) | Code; whether it shows depends on another tick being armed | Verified in code |
| C-4 | **Traced, batch 63 (D-162: zone geometry held once; −12 MB).** **~76 MB stays after the map closes** (heap 34 → 86 MB) | Baseline, both variants | Measured; cause not traced |
| C-5 | Two disk-cached providers (USGS near-field, NDBC) never `Forget` a body that failed to parse, so it is served again until its TTL | Auditor | **Fixed, batch 90**: both forget it - NDBC's station list and buoy files, USGS's every query - as HMS, WFIGS, FIRMS and NWS did; tested against a server that garbles its first answer |
| C-6 | A nil-pointer panic reachable only if the embedded index fails and no recent list is saved (`markFIRMS`, `fireFor`, … on `rp.asm`) | Auditor | **Fixed, batch 90**: `livePipelines.assemblers()`, the assemblers that exist, read by `fireFor`, `seismicFor`, `marineFor` and `markFIRMS` - four copies of the walk made one |

### Performance — the map's responsiveness and data fetching

| # | Finding | Evidence | Status |
|---|---|---|---|
| P-1 | **Fixed, batch 59 (D-156).** **The map's requests wait behind the station's launch burst.** The map's work contexts are `context.Background()` (`map_work.go`), so alerts in view, zone shapes, fire, quakes, AirNow go on the data client's normal lane — the RECENT pipeline's launch burst's lane (~375–600 requests at 30/s, FIFO). Only favourites use `WithPriority` | The ±0.4 s spread of cold M5 | Verified (lane); burst size estimated from code |
| P-2 | **Fixed, batch 59.** **Every move key asks the whole feed at once, and again 600 ms later.** `handleMapKey` batches `mapFeedCmd` on every pan, zoom and region key; D-66's settle tick then asks again. The per-key ask predates D-66 (batch 5) and survived it (batch 15) | History; code | Verified |
| P-3 | **Fixed, batch 60.** **Fire and quakes are fetched on every feed ask whatever their switches** (`inputsFor` → `fireIn`, `mapQuakes.fetch`); buoys, tides and air check theirs | Code | Verified |
| P-4 | **Fixed, batch 60 (D-160).** Zone shapes resolved for alert categories switched off (the estimate skips them, D-149; the feed does not) | Auditor | Reported |
| P-5 | **Fixed, batch 61.** The feed's inputs are fetched one after another — view alerts, fire (perimeters box by box), quakes, buoys, up to 20 serial tide predictions, AirNow — then the zones | Auditor, two independently | Reported |
| P-6 | **Fixed, batch 60.** **Seven overlays are stamped `Valid: now`** (fire ×3, AirNow, buoys, tides, quakes), so no answer compares unchanged and each is handed in again | Code | Verified |
| P-7 | **Fixed, batch 60 (AirNow, perimeters; NDBC left).** Big bodies re-parsed on every ask though cached (AirNow 1.9 MB, perimeters, NDBC) | Auditor | Reported |
| P-8 | **Fixed, batch 58.** `time.LoadLocation` (a zoneinfo read) per alert sentence in the map's description; `platform/tz` exists for this | Code | Verified |
| P-9 | **Fixed, batch 67.** Double renders: `retime` → `setFeed` → `renderMap`, then the caller's own — every forecast step and playback tick | Auditor; confirmed in code, and the mode's switch too; ~3 ms a draw once the report is kept | `retimeDrawn`: once |
| P-10 | The library repaints fully when the blink phase flips, though watchpost places no blinking marker | Code (`frame.go` compares the phase unconditionally; watchpost sets no `Blink`) | Verified mechanism; cost unmeasured |
| P-11 | **Report half fixed, batch 65 (go-tuiMaps rc.27).** The library's report memo is cleared by the wall clock every frame, and `Legend()` rebuilt every Update | Auditor; measured: 9.8 ms and 11 MB a redraw with a twelve-frame loop, all the loop's motion | Report kept by the stale set: 0.02 ms. `Legend()` not yet measured |
| P-12 | **Decode half fixed, batch 66 (rc.28).** The library's loop step: 128 raster samples a cell with the projection recomputed; the radar PNG decoded a pixel at a time through two interface conversions | Auditor; measured: a 600x275 frame 2.7 ms and 165,049 allocations; an advance 6.4 ms, 43 % the basemap repainted, 23 % the rain sampled | Frames read from their pixels: 1.3 ms, 50 allocations. The advance left: the loop plays only on space, ~0.3 % of a core, and the change is the renderer's core |
| P-13 | The station pool and the watchlist can fetch one place twice; `Retain` drops the pool's resolutions on every commit; one gridpoint document decoded twice a cycle | Auditor | **Fixed, batch 102** (D-208): a watched place leaves the recent pipeline, so each place is fetched once; the pool's rows find a place's weather in the recent snapshot, else the priority one, and the console memo is keyed on both; `Retain` keeps the pool's resolutions; the marine read and the fill share one decode of the gridpoint document |
| P-14 | UV's cities are asked again on every pan, zoom and refresh - up to 24 EPA requests, four at a time - though each city's forecast for today does not change through the day (HUM LEAD, D-211) | HUM LEAD | **Fixed, batch 116**: each city's EPA hours are kept for its local day, so a pan, zoom or refresh asks EPA only for the cities not yet read today; a city EPA refused is asked again. W20 keeps them across sessions |
| P-15 | HRRR's frame URL names the minute, not the run, and is cached 2 h: a new run's times draw the last run's pictures (D-212) | Audit | **Fixed, batch 107**: the frame's cache entry is keyed by its run (the run in the address's fragment, never sent), so a new run's minute is fetched afresh |
| P-16 | The temperature answer is rebuilt on every ask though it is the same within the hour - NDFD's XML parsed again (about 32 a lower-48 pan), Open-Meteo's up to three times a box, every grid interpolated, waves and rain built with their layers off (D-212) | Audit | **Fixed, batch 110**: a whole answer is kept for its hour by region, boxes, source, quota, mode, units, rain density, hours ahead and hour - a second ask, or a pan within the same boxes, asks no source; a partial answer is asked again; UV and air are added to a copy |
| P-17 | The radar loop is rebuilt on every ask: every frame's PNG decoded again by `radar.Check` (24-96 a pan) (D-212) | Audit | **Fixed, batch 111**: each frame's check is kept by the frame - source, region, box, time (and run, for HRRR's hours ahead) and size - so a pan decodes no frame it has seen and a refresh only the new one |
| P-18 | The history store reads and decompresses a day's files on every Get and Put, with no memory layer: about 2,000 file reads a pan in Forecast mode (D-212) | Audit | **Fixed, batch 112**: a day's hours are kept decoded while its files stand - its document's time and size, its bucket directory's time and count, its buckets' newest time and bytes - so a replay reads no file it has read, and a write by this instance or another is read at once |
| P-19 | AirNow's contours (2 MB of KML) parsed and rasterised on every ask while Air is on (D-212) | Audit | **Fixed, batch 113**: the file is parsed once for its hour (`bodymemo`, as the reporting areas are) and each box's grid kept by the box and the file's hour |
| P-20 | Open-Meteo Marine's URL shifts as land is learned, so the second ask in an hour misses the cache and is billed again; the quota gate refuses before the cache is read; the map clients' 8 MB memory tier is smaller than a lower-48 hour's bodies (D-212) | Audit | Reported - D-212 step 3 |
| S-12 | Six hand-written keyed, time-bounded memos (`areaMemo`, `aheadFetch`, the zone store's, NWS points', CO-OPS stations', HMS's) and two body-hash memos beside `platform/bodymemo` (`fire.Memo`, `globalfeed.sourceMemo`) (D-212) | Audit | **Body-hash half fixed, batch 108**: `fire.Memo` and `globalfeed.sourceMemo` are `platform/bodymemo` - its keep-errors form for the fire archives, `Last` for a body known unchanged. **Time-bounded half fixed, batch 109**: `platform/agememo` - fresh, one fetch a key shared, a stand-in on failure, a key cap - is the area alerts' memo (now the last eight sets of areas, a pan back asking nothing) and HMS's coalescing (its 30-minute last good now tested); the CO-OPS lists, the zone store, NWS points and `aheadFetch` stay, each for a stated reason (build log, batch 109) |

### Structure — P10, DRY, simplification

| # | Finding | Evidence | Status |
|---|---|---|---|
| S-1 | **Fixed, batch 58 (61 → 31 live).** **61 live P10 findings, not 100** (39 exempted). 29 of them are one false edge: the `scrolls` closure in `windowKeysOf` (`window_keys.go`), which the analyzer reads as a call; a method expression, as the neighbouring rows use, dissolves the 32-function "cycle" | Two auditors independently; the closure read | Verified |
| S-2 | **The P10 checker now keys methods `Recv.Name`**; ledger rows keyed by a bare method name stopped matching — 9 live now (Router's, `Dashboard.View`, `Engine.watchClip`), more as their code is touched | Auditor, from the harness source | Reported — a ledger edit, the HUM LEAD's |
| S-3 | **Fixed, batch 58.** One real 2-cycle: `layerOn` ↔ `radarMode`, bounded at depth 2 | Two auditors | Reported |
| S-4 | Six loops bounded in fact, in a form the checker cannot see | Auditor | Reported |
| S-5 | **Fixed: app half batch 69, tty half batch 70** (`mapWorkers.cmd`, `reconcile`; `setFeed` differs and stays). **App half fixed, batch 69** (`hourGrids`, `forecastDays`, `linedGrid`, `askAnchor`). The map's Forecast-mode step loop written 6×, its hourly loop 4×, grid building 3×, the radar loop fetch 2×; the tty's overlay reconcile 3×, its worker-command wrapper 4× | Auditor | Reported |
| S-6 | Unit conversions (°C/°F, km/mi, m/s) in ~20 places across three layers; the km/mile constant 6×; two spoken wordings already disagree ("kilometres" / "kilometers") | Auditor | **Fixed, batch 100**: `platform/units`, a leaf every layer reads - temperature, distance, speed, feet, inches, knots, mph, each by its definition; ~45 sites moved, six copies of the mile constant and two wrappers gone; spoken "kilometers" throughout; and found on the way, the detail view's feels-like difference said in °F to a metric listener - now in the listener's units |
| S-7 | Control components hand-built where the app's exist (the Request window's checkbox — D-147); `cycleIn` bypassed 4× | Auditor | **Fixed, batch 101**: the Request window's report box is Settings' `checkMark`, its pointer the list's `ListMark`, its position `radioMark`; every picker that steps through a list of choices goes through `cycleIn` - seven bypasses (map scale, detail level, fire, the voice picker, the Settings tabs, the history presets) and the second helper `nextChoice` gone. Index wraps (a menu cursor, the ticker's category, the theme index, the bed's relay pick) step an index, not a choice, and stay |
| S-8 | Code no production path reaches: F-24's on-air masthead, `castProblems`, `ellipsize`, `colorEnabled`, `bedReachFor`, `withForecast` | Auditor | **Fixed, batch 91 (D-197)**: F-24's masthead went in batch 68 (D-158); `castProblems` (its deadlock guards moved to `setCast`), `withForecast`, `term.colorEnabled` and `ellipsize` removed by D-163's rule; `bedReachFor` kept, unwired, owed to the Broadcaster as F-185 |
| S-9 | Functions past ~60 lines: `handleMapKey` 102, `overlaysBox` 96, `mapFeedWith` 86, `geojson.walk` 77, `ttyConfig` 74, `parseDWML` 72 | Auditor | **Fixed, batch 103** (D-209): the six and the two map builders grown since (`buildTemperature`, `withWaves`) split at seams they had - the map's keys by window, menu and view; the menu by warning, rows and note; the feed by source; the config by owner; the GeoJSON walk a `walker` with a method a bracket; NDFD's DWML by winds and temperatures; the temperature and the waves a builder each with fetch, Radar mode, Forecast mode and the chips. Each now under 60 lines, no behaviour changed. Data tables (the themes) stay whole |

### Found by re-measuring

| # | Finding | Evidence | Status |
|---|---|---|---|
| C-7 | **The map drops under its floor at mid-size terminals**: at 149×38 the window's status, notes and radar timeline left the map 11 rows, so the notice showed in its place. Pre-existing and intermittent at the checkpoint (probed on its own binary, even `default`); steady once answers landed reliably | Render probe, checkpoint vs batch 59 | **Fixed, batch 59 (D-159)** |
| R-1 | An occasional ~0.6 s pan freeze under the full session (1 of 42 pans in `default`, 2 of 42 in `heavy`: the pan's own frame, `complete` at the same moment); the baseline's worst was 112 ms | Session runs after batch 59. **Not batch 59's**: a pan A/B, batch 58 against 59, 82 pans each on one workload, gave median 37 / 36 ms, p90 59 / 58, max 61 / 60, none over 300 ms | Open: watched in the next sessions, a CPU profile route added so one can be caught |
| CPU-1 | One burst of ~60 CPU-seconds in ~44 s at +60 min of the `default` session (a warm reopen); absent in `heavy` at its own +60 | A targeted reproduction (the same sequence, 10 min closed, then reopen) did not trigger it: a normal 2-4 s at ~70 % of a core | Open, cause unknown: watched, with the profile route |
| T-1 | One `go test -race ./app` run failed while batch 62 was prepared; its output was discarded by the builder's own `tail -1` | Not reproduced: 5 whole-package `-race` runs and 30 stress runs of the likeliest suspect (the timing-bound `TestTheFeedsInputsAreAskedTogether`, loaded to eight cores) passed | **Explained, batch 62**: batch 61 made two tests' fake servers run on several goroutines at once, their handlers appending unlocked; the next gate's `race` named `TestTheSeasStationsAreAskedOnlyWhileOn`. Locked; 30 runs clean |
| C-8 | **The city scan on the UI goroutine**: the map's estimate reads the state under 25 points of the view, a full scan of the 34k-row city table each, re-parsing every row's coordinates - 31 ms and 1.02 M allocations a view, on every open, feed landing, Settings open and switch. The map window's idle cost measured beside it: ~10 frames a second open or closed (the marquee), 2.2 % of a core open against 0.5 % closed; after the fix 1.8 % - the title's scan was a quarter of the open map's cost | macOS `sample` of the open map; a benchmark on the real index; a 90 s probe each side with the frame-rate instrument | **Fixed, batch 64**: coordinates parsed once - 2.0 ms, 300 allocations; the same answers, tested over a grid |
| C-9 | **Each job's landing redrew the report at 28 ms**: after a feed ~24 library jobs land one by one, each followed by `renderMap`, whose `Report` measured the selected place against every alert's areas again (and again on every pan) - the ~0.65 s from answer to M5 on a warm open | Scratch instrument: Work ~2 ms, `Report` 28 ms, `renderMap` 31 ms a landing | **Fixed, batch 66 (rc.28)**: the measure kept per overlay, alert and place - answer to M5 0.01-0.25 s |
| C-10 | **D-85's radar guard cannot fire**: the radar landing draws only once nothing is left to prepare (`removed \|\| !set \|\| m.Pending() == 0`), but the library counts a job pending only after a `Render` plans it - straight after `Set`, `Pending()` is 0, so the landing always draws | Found by batch 70's mutants; logged calls and `Pending` | **Fixed, batch 93 (D-199, go-tuiMaps rc.31 L11.32)**: a refreshed loop's old pictures stand in until its new ones land; the blink seen live - one frame with no radar a refresh - gone |
| Q-1 | **Open-Meteo's free quota is spendable in ordinary use**: 10,000 calls a day, **every point of a request billed as a call** (its maintainer, open-meteo issues #1295 and #438: weight ~ max(1, variables/10 x a days factor) x locations) - measured (W18.6, batch 83): a whole lower-48 view 265 a refresh in Radar mode, 343 in Forecast mode; California (two boxes) 544 and 704; Alaska 695 - each hour the map is open, and each view into new boxes; temperature, rain and marine at ~80 calls each a region-hour, and every view change in a new region asks again. Spent in UAT (W14's measurement runs from the same IP), every Open-Meteo layer drew nothing, said only in the diagnostics | Live 429s: "Daily API request limit exceeded"; the pricing page | **W18** (D-165 to D-169): the refusal said and held (batch 71), NDFD drawing where it can (batch 71), the history store, UV's and rain's fallbacks to come; the day's cost to be measured (W18.6); **D-185 to D-187: keyless official sources first, Open-Meteo supplemental** (W19) |
| R-2 | **Radar draws significantly slower** (HUM LEAD, UAT of batch 71's build, 2026-09-30) | Reported in UAT; not yet measured. In range: batches 64-71. Candidates to measure first: go-tuiMaps rc.28's direct pixel reads (batch 66) and its per-alert measure memo; batch 70's shared reconcile and worker commands; batch 71's NDFD grids stretched under the loop (two grids meeting in one span while Open-Meteo refuses) and the quota notice spliced every frame | **D-196 (2026-10-01): no A/B - today's build to UAT first; measured only if the HUM LEAD still finds it slow.** Since reported: W14 batches 64-70 and W19 changed the map's workload (keyless sources, fewer Open-Meteo grids). **Measured, batch 99 (D-204)**: against `checkpoint/pre-w14`, five builds live, no regression - radar pixels idle 1.9-3.6 s against 3.2-3.3, under full load a median 3.4 s against 3.2; the observed loop no longer waits for HRRR, and a failed HRRR is asked again after 30 s |
| S-10 | The station identification on going ON AIR (F-24's `mastheadLine`) is built and tested but wired to nothing | Code | **Fixed, batch 68 (D-158)**: removed; the seam is `GoOnAir`'s, the words kept and tested |
| S-11 | F-27's spoken transitions are unwired too: the line-up's `ProgrammeReturn` and `Announcement` are never set in production (`app/schedule.go`), so no "returns to its regularly scheduled programming" is heard; `programmeReturnLine` is called by nothing | Code: the one `lineup.Settings` built outside tests sets `Max` and `Depth` alone | **Fixed, batch 72 (D-163)**: removed; the seam is the line-up's construction, the words kept and tested |

### Claimed, and did not hold

| Claim | What was found |
|---|---|
| `geo.CompassIndex` returns -1 for a negative bearing and callers panic | True of the arithmetic below about −67°, but every caller passes `[0, 360)` — `BearingDeg` normalises, the feeds' directions are compass degrees. A hardening, not a live defect |
| The library repaints twice a second with nothing blinking | Its clock already schedules no blink tick then; the waste is confined to draws that happen for other reasons (P-10) |
