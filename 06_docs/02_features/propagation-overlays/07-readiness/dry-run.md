---
title: "0.19.0 PLAN — the dry run (FR-10.6, REFLECT L1)"
date: 2026-10-06
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "COMPLETE for PLAN — every FR-10.6 item measured or its gap named; targets ruled at D-93 to D-99 and D-108; the forecast at D-103; G1 and M1/M4 measured in BUILD"
---

# The dry run

## What the dry run found

- **The current hour is clearly better than the climatology**, most of all near stations. Within 500 km of a reporting station it is about half the climatology's error. In the mainland US, MUF is 2.1 MHz RMS against 2.9.
- **Hours ahead are the climatology, labelled typical** (D-109). Nothing tested beats it in a storm, and the one method that beat it on quiet days needs a field from the day before that 0.19.0 never holds.
- **The US floor is met** on the test days, with one US day missing on foF2.
- **The weak places are the storm and the Pacific**, where the chart is about as good as the climatology. It says so by distance (D-116).
- **The costs fit.** An update's compute is a few milliseconds; a cold open is about a third of a second; go-tuiMaps' frame is the one over budget (D-96).
- **Its limits:** one week, one solar level, one storm, four mainland-US stations.

REFLECT L1: each metric's instrument runs once before PLAN exits, and its number or its gap goes in the PLAN
report. FR-10.6 lists what this dry run measures. Requests follow D-39's throttle. The evidence scripts are
in go-ionomaps `06_docs/02_features/go-ionomaps/02-analysis/evidence/` (`dry_fetch.py` and the scoring
scripts).

## The measurements

| Item (FR-10.6) | Result | State |
|---|---|---|
| GloTEC's availability | **100%**: 4,466 of 4,466 ten-minute grids from 2026-09-05 21:45Z to 2026-10-06 21:55Z, no gap; NOAA keeps about 31 days | measured |
| #27: can a shipped dataset reach the 32 MB read cap? (D-55) | **No.** The largest, NDFD hourly, holds about 14.5 KB a day of decompressed roll-up, about 5.3 MB a year. Two days of local data, projected linearly | measured; no 0.18.1 hotfix indicated |
| WSPR density at near-vertical range under the reference circuit (D-73, D-91) | **Dense in the continental US on the near-vertical bands.** On 2026-10-05, spots between 50 and 400 km between US stations that clear **+13 dB** at 100 W (D-91; ground wave under 50 km removed): 656 (160 m, 67 station pairs), 19,642 (80 m, 791 pairs), 927 (60 m), 63,525 (40 m, 2,881 pairs), 2,723 (30 m), 1,973 (20 m), thinning above (12 m: 51); every band but 12 m had spots in all 24 hours. At the first count's +10 dB and no distance floor: 30,165 (80 m) and 82,564 (40 m), reproduced exactly by the second query | measured (one day) |
| D-RAP's file | 42 KB text, a 2° × 4° latitude-longitude table plus valid time, recovery estimate and X-ray/proton messages; `Last-Modified`, `ETag`, `max-age=60` | measured |
| MUF(3000) and foF2 held out for B on D, by station and distance, with signed bias; mainland US and Pacific separately | **Measured over the week** (below; 1,476 pairs, 30 stations): on the test days, mainland US, B on D 0.66 MHz foF2 and 2.10 MUF against climatology's 0.82 and 2.88, **so D-75's floor is met**; whole week, all stations, 0.90 and 3.41 against 1.08 and 3.77; signed bias +0.03 MHz foF2 (GloTEC +0.51, climatology +0.28). The Pacific stations are no better than climatology. A hybrid (M(3000)F2 assimilated over climatology, not GloTEC) cuts MUF to 3.26 overall | measured (a week, including a G1 to G2 storm) |
| The NRL refits (D-43), with the F10.7 rule | **Measured:** CCIR refit and raw CCIR are level (foF2 1.08 and 1.08 MHz, MUF 3.77 and 3.79); URSI is worse (1.14, 3.97). The 30-day mean F10.7 beats the day's value on MUF (3.77 against 4.14), with foF2 level (1.08 and 1.07) | measured; **ruled D-104**: the 30-day mean |
| The forecast hours' error at +3 h and +12 h (D-75) | **Measured:** R-1.3's decay toward climatology fails the floor at +12 h in the US (0.85 against 0.76 MHz foF2). A blend of yesterday's hybrid field and climatology (w = 0.5) is no worse than climatology at both leads on the test days. It is clearly better on quiet days (+12 h US 0.50 against 0.60), and level or slightly worse in the storm (MUF 4.44 against 4.29) | measured; **ruled D-103**: the blend, every forecast hour labelled low-confidence |
| The answers' cost per keypress (FR-1.17) | **Measured in a spike** (below): at 2°, an hour step (reach, best bands, paths, centre) costs about 4.4 ms and a frequency change about 3.8 ms, with no allocations; at 1°, about 19 ms and 17 ms. Reach is the only answer that costs real time (about 212 ns a cell) | measured (spike) |
| G1 over at least 48 hours | **The instrument is proven** (below): a 10-minute offline run of today's watchpost (89c6f5bf) gave RSS 55 to 58 MB, steady, and about 3% of one core. The 48-hour measurement needs a build with the mode, so it lands in BUILD (W10.2) with this harness | instrument dry-run done; measurement in BUILD |
| M2, M5, M5b (UAT) | the protocol is written (`uat-protocol.md`); its scenarios are drawn in BUILD once the answers exist; graded by the HUM LEAD alone (D-68) | protocol drafted; the sitting is live (D-89) |
| M1 and M4 (agreement with WSPR) | **a gap until BUILD:** the instrument needs the chart's path answer (go-ionomaps G6, G7). The WSPR density it needs is measured (above). Its target is ruled relative to the climatology baseline (D-99: plus 5 points), with the absolute numbers measured by G10.4's instrument | gap, named; target ruled |
| D-73's +10 dB in 2.5 kHz against published practice | **Within published practice for just-usable SSB voice** (below): the sources span +4 to +14 dB in 2.5 kHz. ITU-R F.339-8's J3E "just usable" is 47 dB-Hz (stable) and 48 (fading), PEP to noise in 1 Hz, which is +13 and +14 dB in 2.5 kHz. WSPR's SNR is in 2500 Hz (WSJT-X 2.6.1 User Guide §7.1) | checked; **+13 dB ruled (D-91)** |
| The climatology's evaluation (part of G-G1) | **Measured in the spike** (below): PyIRI's refit shape (900 spherical-harmonic terms × 11 Fourier terms, plus the Apex mapping), both fields: a first open of 25 hours at 2° takes 334 ms with each cell's QD coordinates cached (6.9 MB), against 3.4 s naive; each new hour 12.7 ms. A Legendre cache halves it again but costs 61 MB, over G1's 25 MB (D-95) | measured (spike) |
| go-tuiMaps M5: a whole-globe frame with a field (v0.3.0 P7.1) | **Measured on v0.2.0** (below): after a pan, fill and labelled contours take 5.7 ms at 200 × 56 and **22.2 ms at 400 × 110**, over a 16 ms frame; fill alone 2.4 and 9.1 ms; an unchanged frame about 2 µs | measured |
| The update's compute (go-ionomaps G-G1) and GloTEC's decode (R-8.2) | **Measured in the spike** (below): the Gaussian process over 39 stations, two fields, 6.0 ms at 2° and 22.6 ms at 1°; the typed decode of one real grid 4.87 ms and 740 KB, against 10.7 ms and 7.61 MB generic | measured (spike) |

## Requests made

| UTC | Host | Request | Result |
|---|---|---|---|
| 2026-10-06T22:19:39Z | NOAA SWPC | GloTEC directory index | 200, 505,115 B |
| 22:20:41Z onward | GIRO | 39 stations, 2026-09-29 to 10-05, one request each (one burst, D-39) | all 200 |
| 22:21:28Z | wspr.live | one aggregate SQL query (near-vertical density) | 200, 314 B |
| 22:21:42Z | NOAA SWPC | `text/drap_global_frequencies.txt` | 200, 42,469 B |
| from 22:21Z, one every 10 minutes | NOAA SWPC | 56 GloTEC grids, 3-hourly over the week | all 200, ending 07:32:11Z |
| 2026-10-07T07:32:11Z | NOAA SWPC | the week's fetch completed: 39 GIRO and 56 GloTEC requests | all 95 answered 200; no 429 |
| 07:43:18Z | NOAA SWPC | `text/daily-geomagnetic-indices.txt` (the week's activity) | 200, 3,877 B |
| 2026-10-07T01:41:41Z | NOAA SWPC | `text/daily-solar-indices.txt` (F10.7 by day, for the climatology's rule) | 200, 2,919 B |
| 02:00:33Z | wspr.live | one aggregate SQL query: near-vertical density at +13 dB, ground wave removed (D-91) | 200, 442 B |
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

### The update's compute and GloTEC's decode (the same spike)

| Benchmark | Before the pass | After |
|---|---|---|
| Gaussian process, 39 stations, foF2 and M(3000)F2, 2° | 39.8 ms | **6.0 ms**, no allocations |
| The same, 1° | 153.5 ms | **22.6 ms** |
| GloTEC decode, one real grid (5,184 points), typed | — | **4.87 ms**, 740 KB, 16 allocations |
| The same, generic (`map[string]any`) | — | 10.7 ms, 7.61 MB, 191,890 allocations |

- **Method:** kernel var·exp(−d/L) in great-circle distance, L = 4000 km and noise 1.0, the values an early partial run tuned; the full week tuned L = 2000 km and noise 0.5 (below). The cost does not depend on L or the noise. Two Cholesky solves at 39 × 39, then a prediction at every cell.
- **The pass:** station unit vectors, dot products for the distance, no allocations. Before and after agree within 1e-8.
- **Reading:** an update's compute (decode and assimilation) is about 11 ms at 2° and about 28 ms at 1°, off the UI goroutine.
- **Not measured here:** the network. The climatology's evaluation is measured below; since D-109 the hours ahead are the climatology's hours, so they add no cost of their own.
- The grid file was read from the session's scratch space, outside the timed loop, and not copied.

## D-73's threshold against published practice (desk check)

Every figure converted to dB in 2.5 kHz: a dB-Hz figure less 34.0 dB; a 3 kHz figure plus 0.8 dB.

| Source | Level | As published | In 2.5 kHz |
|---|---|---|---|
| ITU-R F.339-8 (02/2013), Annex 1 Table 1, J3E telephony, audio speech-to-noise (note 18) | just usable / marginally commercial / good commercial | 6 / 15 / 33 dB in 3 kHz | 6.8 / 15.8 / 33.8 |
| the same, RF, stable (PEP to noise in 1 Hz) | the same | 47 / 56 / 64 dB-Hz | **13.0** / 22.0 / 30.0 |
| the same, RF, fading, no diversity | the same | 48 / 61 / 72 dB-Hz | **14.0** / 27.0 / 38.0 |
| VOACAP Online manual (OH6BG, rev. 2025-03-25, §2.1, §2.3) | minimum (internal) | 38 dB-Hz | 4.0 |
| voacap.com input help, "Req'd SNR" | "reasonable" SSB | 45 dB-Hz | 11.0 |
| K1JT's "Weak-Signal S/N Limits", as quoted on voacap.blogspot.com (2018-04) | minimum | about +10 dB in 2500 Hz | 10.0 (secondary; the slide itself not seen) |

**WSPR's reference bandwidth** is 2500 Hz: WSJT-X 2.6.1 User Guide §7.1 ("a standard reference noise bandwidth of 2500 Hz"), §1 and §17.2.10. WSPR is a constant-envelope signal, so its power is its PEP. A WSPR spot normalised to 100 W therefore compares directly with F.339's PEP figures.

**Reading.**
- **Ruled at D-91: +13 dB**, F.339-8's stable "just usable".
- +10 dB is inside the published spread (+4 to +14).
- It is about 3 dB stricter than F.339's audio figure, and about 3 to 4 dB looser than its PEP figures.
- F.339-8 is the only primary, versioned standard among the sources.
- F.339's note 4 adds 11.5 dB for day-to-day fluctuation against monthly-median predictions. The chart answers the hour, not a monthly median, and M1 and M4 score against that hour's spots, so the note does not apply.

The extracts are in the session's scratch space (`snr/`).

## go-tuiMaps M5: a whole-globe frame with a field (v0.3.0 P7.1)

**Measured on v0.2.0** (the code on `feature/propagation-fields`):
- a scratch benchmark outside the tree, Apple M5 Pro, go1.27.1, Truecolor, no basemap, `-benchtime=2s -count=5`, medians;
- a synthetic 2° whole-globe field (180 × 91, values 2 to 14). v0.2.0's generic host type has no ramp (L-2.4's defect), so the `temperature` preset's ramp was used with breaks every 1 MHz, giving 11 contour levels;
- "pan" re-renders after a one-cell pan; "repeat" renders with nothing changed.

| Frame | 200 × 56 | 400 × 110 |
|---|---|---|
| pan, fill | 2.37 ms; 716 KB, 10,777 allocations | 9.05 ms; 2.74 MB, 39,561 allocations |
| pan, fill and labelled contours | 5.70 ms | **22.2 ms** |
| repeat | 1.1 µs | 2.1 µs |
| pan, no overlay | 0.14 ms | 0.51 ms |

**Reading.**
- A large terminal's frame after a pan does not fit 16 ms with contours, about 1.4 times over.
- The profile puts most of the cost in colouring each cell: a text-contrast check using `math.Pow`, and blending the faint bands (about 50 to 65%). Contours are about 14%. There is about one allocation per cell.
- These figures are 1.5 to 7 times the round-1 Performance reviewer's (3.9 and 14.8 ms with contours). The difference is not explained: possibly the colour depth or the field's resolution.
- watchpost renders the map on the UI goroutine (round 1, F7: `modes/tty/map_pane.go:346-363`), so a slow frame there is a slow keypress.

### The climatology's evaluation (the same spike)

**The model's shape**, confirmed against PyIRI 0.1.7's `sh_library.py` (MIT), with random coefficients and none of its data copied:
- 11 Fourier terms in time × 900 spherical-harmonic terms (lmax 29), for each of the two fields and both solar levels;
- each cell mapped to quasi-dipole coordinates by a 441-term fit (Apex);
- the Fourier terms combined once per hour; the solar-level blend folded into the coefficients, since the model is linear in them.

| Case, both fields | 2° | 1° |
|---|---|---|
| one hour, naive | 134 ms; 302 MB allocated | 549 ms |
| one hour, QD coordinates cached per day | **12.7 ms**; none | 49.9 ms |
| one hour, Legendre values also cached | 6.7 ms | 27.7 ms |
| the day's cache: QD only / Legendre float64 / float32 | **0.4** / 61.3 / 30.9 MB | 1.6 / 244 / 123 MB |
| first open, 25 hours, naive | 3.39 s | 13.4 s |
| first open, 25 hours, QD cached | **334 ms**; 6.9 MB | 1.29 s |
| first open, 25 hours, Legendre cached | 191 ms; 68 MB | 737 ms |

Every variant agrees with the naive one within 6e-14 (float32: 2e-7).

**Reading.**
- The Legendre cache is ruled out at 2° by G1's 25 MB (D-95). The QD-only cache fits.
- A first open then costs about a third of a second, off the UI goroutine.
- After that, the climatology's hour fields are kept for the day: 2 fields × 25 hours × 16,380 cells × 4 bytes is about 3.3 MB, and one new hour costs about 13 ms each hour.
- An update every 10 minutes (D-94) adds the decode and the assimilation, about 11 ms.
- Together that is roughly 0.1 s of CPU an hour, against G1's 1% (36 s an hour).

## The week, scored (FR-10.6)

**Run:** `week.py DIR DSI 2026-10-04,2026-10-05` (go-ionomaps `02-analysis/evidence/`; the storm days are a required argument), PyIRI 0.1.7 under Python 3.12 (D-86), over 2026-09-29 00:05Z to 10-05 21:05Z. The "30-day" F10.7 mean covered 23 to 29 observed days, because NOAA's file starts 2026-09-06; the scorer states the window and refuses fewer than 20. Every method is scored on the same pairs, and the climatology is the one that ships (the CCIR refit, A-28):
- 56 GloTEC grids, 3-hourly;
- 39 GIRO stations fetched, of which 30 had soundings at a confidence score of 70 or more;
- 1,476 held-out pairs.

**Tuning:**
- kernel length and noise tuned on the first three days, then scored on the last four (823 pairs): B on D L = 2000 km, noise 0.5; B on C L = 2000 km, noise 0.2;
- the forecast's decay constant and blend weight also tuned on the first three days.

**The week's activity** (NOAA daily geomagnetic indices):
- quiet to 10-03 (planetary A 2 to 8);
- **a storm on 10-04 (A 36, Kp to 5.67, about G2) and 10-05 (A 24, Kp to 5.33, G1)**.

So the tuning days were quiet and the test days run into the storm.

**Methods** (every one held out at the scored station):

| Method | What it is |
|---|---|
| GloTEC | the background alone: foF2 from NmF2, M(3000)F2 from hmF2 |
| B on D | the plan (R-9.2): foF2 and M(3000)F2 residuals assimilated over GloTEC |
| hybrid | foF2 from B on D; M(3000)F2 assimilated over the climatology instead of GloTEC |
| B on C | both assimilated over the climatology (the fallback when GloTEC is missing) |
| clim + mean | climatology plus the other stations' mean residual: a proxy for an effective sunspot number (CQ-N1) |
| clim | the best climatology: CCIR refit, F10.7 the 30-day mean |

**Held out, test days** (foF2 RMS / bias | MUF(3000) RMS / bias, MHz):

| Region (n) | B on D | hybrid | GloTEC | B on C | clim + mean | clim |
|---|---|---|---|---|---|---|
| all (823) | 0.96 / +0.03 \| 3.60 / +0.06 | 0.96 \| **3.44** | 1.12 / +0.53 \| 3.92 | 1.06 \| 3.79 | 1.13 \| 4.10 | 1.14 / +0.20 \| 4.04 |
| mainland US (119) | **0.66** / −0.09 \| **2.10** | 0.66 \| 2.08 | 0.76 / +0.39 \| 2.58 | 0.79 \| 2.56 | 0.87 \| 2.85 | 0.82 / −0.09 \| 2.88 |
| Pacific (118) | 1.37 \| 5.58 | 1.37 \| 5.17 | 1.48 \| 5.95 | 1.34 \| 4.97 | 1.38 \| 5.18 | 1.38 \| 5.09 |
| elsewhere (586) | 0.91 \| 3.33 | 0.91 \| 3.22 | 1.10 \| 3.62 | 1.04 \| 3.73 | 1.11 \| 4.07 | 1.15 \| 4.00 |

**By distance to the nearest other reporting station** (whole week, foF2 | MUF):

| Distance | B on D | hybrid | clim |
|---|---|---|---|
| under 500 km (373) | 0.33 \| 1.27 | 0.33 \| 1.27 | 0.72 \| 2.63 |
| 500 to 1000 km (202) | 0.56 \| 1.99 | 0.56 \| 1.92 | 0.88 \| 2.87 |
| 1000 to 2000 km (355) | 1.03 \| 3.93 | 1.03 \| 3.82 | 1.20 \| 4.39 |
| over 2000 km (546) | 1.14 \| 4.34 | 1.14 \| 4.09 | 1.27 \| 4.25 |

**By day** (all stations, foF2 | MUF):
- B on D beats climatology every day, storm days included. On 10-04 it gets 1.01 | 4.05 against 1.18 | 4.48; in the US, 0.68 | 2.74 against 1.02 | 3.40.
- The one exception is 10-03 in the US on foF2 (0.93 against 0.84).

**By station:** the mainland-US stations score 0.42 to 0.75 MHz on foF2 (AL945, EG931, IF843, MHJ45), and the Pacific stations 0.77 to 1.60 (EA653 Adak best, WA619 Wake worst). Full table: `week.py`'s section 5.

**Climatologies** (whole week, foF2 | MUF):

| Variant | F10.7 30-day mean | F10.7 the day's |
|---|---|---|
| CCIR refit | 1.08 / +0.28 \| 3.77 | 1.07 / −0.27 \| 4.14 |
| CCIR raw | 1.08 \| 3.79 | 1.07 \| 4.18 |
| URSI refit | 1.14 \| 3.97 | 1.13 \| 4.34 |
| URSI raw | 1.14 \| 4.01 | 1.14 \| 4.40 |

**The forecast**, scored on the same pairs for every method:

| | +3 h all | +3 h US | +12 h all | +12 h US |
|---|---|---|---|---|
| decay toward climatology (R-1.3; τ 6 h, 12 h) | 1.10 \| 4.05 | 0.79 \| 2.53 | 1.13 \| 4.05 | **0.85 \| 2.87** |
| persistence | 1.18 \| 4.41 | 0.86 \| 2.70 | 1.27 \| 4.58 | 1.09 \| 3.44 |
| yesterday | 1.16 \| 4.37 | 0.89 \| 2.88 | 1.18 \| 4.46 | 0.78 \| 2.94 |
| blend of yesterday's B on D and climatology (w = 0.5) | 1.07 \| 3.96 | 0.81 \| 2.69 | 1.09 \| 4.03 | 0.72 \| 2.75 |
| **the same with the hybrid** | **1.07 \| 3.93** | **0.81 \| 2.72** | **1.09 \| 3.99** | **0.72 \| 2.72** |
| climatology | 1.12 \| 3.98 | 0.81 \| 2.73 | 1.14 \| 4.02 | 0.76 \| 2.78 |

**Quiet against disturbed** (test days; 10-04 and 10-05 disturbed):

| | +3 h quiet | +3 h storm | +12 h quiet | +12 h storm |
|---|---|---|---|---|
| hybrid blend, all | 0.95 \| 3.45 | 1.19 \| 4.39 | 0.96 \| 3.51 | 1.22 \| 4.44 |
| climatology, all | 1.07 \| 3.69 | 1.18 \| 4.28 | 1.08 \| 3.75 | 1.20 \| 4.29 |
| hybrid blend, US | 0.70 \| 2.03 | 0.92 \| 3.39 | 0.50 \| 1.99 | 0.94 \| 3.45 |
| climatology, US | 0.75 \| 2.31 | 0.89 \| 3.17 | 0.60 \| 2.28 | 0.94 \| 3.33 |

**Reading.**
- **The current hour's floor (D-75) is met** on the test days in aggregate, in the US and overall. One US day misses on foF2: 10-03, 0.93 against the climatology's 0.84 (the agent's error E-17 had said "every day").
- **The forecast floor was met only by the hybrid blend, and by thin margins** (0.00 to 0.06 MHz). R-1.3's decay fails at +12 h in the US. The blend needs a field from the day before, which no ruled data path supplies (the red team's B-F1; the agent's error E-15), so **D-109 makes the hours ahead the climatology, labelled typical**.
- On quiet days the blend beats climatology clearly. In the storm, no forecast beats climatology: the measured fields catch a storm, but nothing here foresees one (D-88).
- The hybrid's gain is in MUF away from the US: GloTEC's hmF2-derived M(3000)F2 is the weak half.
- The bias that RK-11 feared, a false "open" from a high foF2, is gone from B on D (+0.03 MHz). The assimilation's mean term removes it.
- The effective-index proxy gains nothing over the assimilation.

**Its limits:**
- one week, at one solar level (F10.7 92 to 100);
- one storm, of G1 to G2;
- 30 stations, 4 of them mainland US and 4 Pacific;
- kernels and weights tuned on three quiet days;
- the hybrid and the blend were chosen after seeing four days of data (the interim runs), so their test scores flatter them slightly;
- GIRO's autoscaled values are the truth here, with their own error.

### Skill by lead (quiet test days; decay from now, its constant tuned per lead)

| Lead | US: forecast \| climatology | All: forecast \| climatology |
|---|---|---|
| now (the hybrid) | 0.71, 1.74 \| 0.74, 2.30 | 0.89, 3.09 \| 1.09, 3.75 |
| +3 h | 0.73, 2.00 \| 0.75, 2.31 | 1.01, 3.61 \| 1.06, 3.69 |
| +6 h | 0.75, 2.15 \| 0.73, 2.24 | 1.04, 3.57 \| 1.08, 3.69 |
| +9 h | 0.83, 2.30 \| 0.74, 2.31 | 1.08, 3.78 \| 1.11, 3.75 |
| +12 h | 0.70, 2.32 \| 0.60, 2.28 | 1.05, 3.71 \| 1.09, 3.78 |

Each cell is foF2, then MUF (MHz).

**Reading.**
- Today's departure from climatology fades within about 3 hours.
- What the blend keeps is recurrence: the same hour a day before.
- The current hour is where the chart is clearly better than climatology.

D-103 keeps 24 hours, labelled low-confidence.

### The live offset (Q-3, D-105)

At each grid time, the mean over the reporting stations of sounding foF2 minus GloTEC foF2: what the assimilation removes (`offset.py`).

| Day | Mean | Range | Stations |
|---|---|---|---|
| 09-29 | −0.39 | −0.54 to +0.00 | 26 to 29 |
| 09-30 | −0.45 | −0.64 to −0.26 | 26 to 29 |
| 10-01 | −0.63 | −0.80 to −0.43 | 26 to 29 |
| 10-02 | −0.57 | −0.77 to −0.32 | 25 to 28 |
| 10-03 | −0.51 | −0.84 to −0.22 | 25 to 27 |
| 10-04 (storm) | −0.54 | −1.11 to +0.10 | 23 to 29 |
| 10-05 (storm) | −0.51 | −0.87 to −0.15 | 23 to 25 |

**The week:** mean −0.51 MHz, SD 0.23.

A band of 2 SD (0.45 MHz) flagged 3 of 56 updates:
- 10-04 00:05Z (−1.11) and 06:05Z (+0.10), at the storm's onset;
- 09-29 06:05Z (+0.00), a quiet night.

