---
title: "0.19.0 PLAN — the dry run (FR-10.6, REFLECT L1)"
date: 2026-10-06
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "IN PROGRESS — measurements land here as they finish; targets are ruled from them (D-49), against the floors set before (D-75)"
---

# The dry run

REFLECT L1: each metric's instrument runs once before PLAN exits, and its number or its gap goes in the PLAN
report. FR-10.6 lists what this dry run measures. Requests follow D-39's throttle. The evidence scripts are
in go-ionomaps `06_docs/02_features/go-ionomaps/02-analysis/evidence/` (`dry_fetch.py` and the scoring
scripts).

## Summary so far

| Item (FR-10.6) | Result | State |
|---|---|---|
| GloTEC's availability | **100%**: 4,466 of 4,466 ten-minute grids from 2026-09-05 21:45Z to 2026-10-06 21:55Z, no gap; NOAA keeps about 31 days | measured |
| #27: can a shipped dataset reach the 32 MB read cap? (D-55) | **No.** The largest, NDFD hourly, holds about 14.5 KB a day of decompressed roll-up, about 5.3 MB a year. Two days of local data, projected linearly | measured; no 0.18.1 hotfix indicated |
| WSPR density at near-vertical range under the reference circuit (D-73) | **Dense in the continental US.** On 2026-10-05, spots under 400 km between US stations that clear +10 dB at 100 W: 1,204 (160 m), 30,165 (80 m), 1,120 (60 m), 82,564 (40 m), 7,567 (30 m), 10,795 (20 m), down to 7,569 (10 m); every band had pairs in all 24 hours (80 m: 1,579 station pairs; 40 m: 4,829). Ground wave under about 50 km is still to be filtered | measured (one day) |
| D-RAP's file | 42 KB text, a 2° × 4° latitude-longitude table plus valid time, recovery estimate and X-ray/proton messages; `Last-Modified`, `ETag`, `max-age=60` | measured |
| MUF(3000) and foF2 held out for B on D, by station and distance, with signed bias; mainland US and Pacific separately | a week of data (2026-09-29 to 10-05) is downloading under the throttle | in progress |
| The NRL refits (D-43), with the F10.7 rule | PyIRI 0.1.7 installed under Homebrew Python 3.12 (D-86); scored on the week | pending the week |
| The forecast hours' error at +3 h and +12 h (D-75) | from the week's 3-hourly grids | pending the week |
| The answers' cost per keypress (FR-1.17) | **Measured in a spike** (below): at 2°, an hour step (reach, best bands, paths, centre) costs about 4.4 ms and a frequency change about 3.8 ms, with no allocations; at 1°, about 19 ms and 17 ms. Reach is the only answer that costs real time (about 212 ns a cell) | measured (spike) |
| G1 over at least 48 hours | **The instrument is proven** (below): a 10-minute offline run of today's watchpost (89c6f5bf) gave RSS 55 to 58 MB, steady, and about 3% of one core. The 48-hour measurement needs a build with the mode, so it lands in BUILD (W10.2) with this harness | instrument dry-run done; measurement in BUILD |
| M2, M5, M5b (UAT) | the protocol is written (`uat-protocol.md`); its scenarios are drawn in BUILD once the answers exist; graded by the HUM LEAD alone (D-68) | protocol drafted; the sitting is live (D-89) |
| M1 and M4 (agreement with WSPR) | **a gap until BUILD:** the instrument needs the chart's path answer (go-ionomaps G6, G7). The WSPR density it needs is measured (above). Its target can be ruled now relative to the climatology baseline (D-23), with the absolute numbers measured by G10.4's instrument | gap, named |
| D-73's +10 dB in 2.5 kHz against published practice | a desk check of the published SSB voice thresholds | pending |

## Requests made

| UTC | Host | Request | Result |
|---|---|---|---|
| 2026-10-06T22:19:39Z | NOAA SWPC | GloTEC directory index | 200, 505,115 B |
| 22:20:41Z onward | GIRO | 39 stations, 2026-09-29 to 10-05, one request each (one burst, D-39) | all 200 |
| 22:21:28Z | wspr.live | one aggregate SQL query (near-vertical density) | 200, 314 B |
| 22:21:42Z | NOAA SWPC | `text/drap_global_frequencies.txt` | 200, 42,469 B |
| from 22:21Z, one every 10 minutes | NOAA SWPC | 56 GloTEC grids, 3-hourly over the week | in progress |
| 2026-10-07T00:50:40Z | NOAA SWPC | D-RAP's product page (for PLAN's Q-2) | 301 to spaceweather.gov |
| 00:51:12Z | NOAA SWPC | `products/noaa-scales.json` (Q-2) | 200, 1,107 B, `ETag`, `max-age=60` |
| 00:51:56Z | NOAA SWPC | D-RAP's product page at spaceweather.gov (Q-2) | 200, 80,868 B |

## G1's instrument, dry run (REFLECT L1)

**The harness.**
- `expect` starts the binary in a 133 × 44 pseudo-terminal, under `sandbox-exec` with outbound network denied except loopback, and with a fresh `HOME`.
- It writes the process id to a file, drains the output, and ends the run with SIGTERM after a set time (SIGKILL 10 s later if needed).
- A sampler reads that id and records the UTC time, RSS, CPU time and command name with `ps` every 30 s.
- A `perl` alarm bounds the whole run.
- The scripts are in the session's scratch space; BUILD lands them as W10.2, with byte counts from the propagation client.

**Two faults found and fixed before this run:**
- the first sampler measured `expect`, not watchpost;
- `q` did not end watchpost under the harness, so a run hung for about 55 minutes.

Both are fixed: the process id comes from the spawned process, and the run ends by signal.

**The run:** 2026-10-07T00:49:24Z to 00:58:58Z, 20 samples, all of the watchpost process itself:

| Measure | Result |
|---|---|
| RSS | 55,408 KB at the first sample, 55 to 58 MB after, 57,808 KB at the last; no growth |
| CPU | 17.7 s over 574 s, about 0.93 s every 30 s: **about 3% of one core** |
| Screen | the Observer dashboard, first run, "awaiting first data..." |
| Terminal output | 1.95 MB in 10 minutes (the dashboard redrawing) |
| Downloads | none: outbound network denied |
| Ending | SIGTERM at 600 s; the terminal restored; no process left behind |

**What it does not show:**
- the cost of real fetching and parsing, since everything was offline;
- growth over hours;
- the Propagation mode.

The 48-hour run on a build with the mode, mode in use against not, is W10.2.

The idle 3% is today's baseline, before the mode. It is a candidate question for that measurement (is the redraw rate needed when nothing changes?), not a 0.19.0 change.

## The answers' cost per keypress, spike (FR-1.17)

**The spike.**
- Throwaway Go in the session's scratch space, never committed; the standard library only, `CGO_ENABLED=0`, one goroutine.
- Apple M5 Pro, go1.27.1, `-benchtime=2s -count=5`. The table gives medians.
- Synthetic smooth fields: foF2 2 to 14 MHz, M(3000)F2 2.5 to 3.8.
- **The basic-MUF and absorption formulas are placeholders of equivalent cost**: the same count of trigonometric, square-root and exponential calls per path as a P.533-style computation, not the published equations.
- One optimisation pass, as a real build would have: per-row and per-column sine and cosine tables, the solar zenith as a dot product, no allocations, band-independent work out of the band loop. A test holds the two versions to identical outputs.

| Answer | Before the pass | After |
|---|---|---|
| Reach, 2° (180 × 91 cells) | 5.49 ms; 16,381 allocations | **3.47 ms**; none |
| Reach, 1° (360 × 181) | 21.2 ms | **13.4 ms** |
| Best bands, 400 km, now | 226 µs | **9.5 µs** |
| Best bands, now and 24 hours | 5.49 ms | **170 µs** |
| Paths, 7 targets × 10 bands × 24 hours | 587 µs | **36 µs** |
| Centre (a point, bands now and 24 hours) | 120 µs | **3.7 µs** |
| A frequency change, 2° (reach and centre) | 5.67 ms | **3.77 ms** |
| An hour step, 2° (reach, best bands, paths, centre) | 7.14 ms | **4.43 ms** |
| An hour step, 1° | 31.7 ms | **18.7 ms** |

**Reading.** Every answer runs off the UI goroutine anyway (FR-1.17, FR-4.8), so the frame is never held.
- At 2°, the default, an hour step takes about 4% of a 100 ms response and fits inside a 16 ms frame.
- At 1°, it fits 100 ms but not a frame.

**What it does not show:**
- the real formulas: the spike computes the full worst case for every cell, with no early exit;
- the update's own cost: the Gaussian process over the stations, on two fields;
- the app's other load. The machine was shared during the runs (load 6 to 12), which widens the spread on the composites (up to 41% at 1°).

