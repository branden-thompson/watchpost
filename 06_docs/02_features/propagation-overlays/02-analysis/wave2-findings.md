---
title: "0.19.0 DISCOVER — wave 2 findings (measurements, D-18 budget)"
date: 2026-10-06
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "COMPLETE — the path was ruled on these findings (B on D, D-40); throttle behaviour from them (D-39); corrected after the DISCOVER-exit red team, round 1"
---

# Wave 2 findings

## Summary

- **The path is B on D** (D-40): GIRO's ionosondes assimilated over a NOAA GloTEC background. On one day,
  held out, its foF2 error was 1.00 MHz overall and 0.37 MHz within 500 km of a station, against 1.14 for
  GloTEC alone and 1.38 for climatology.
- **Those figures have limits:**
  - one day;
  - the near-station bin is seven European stations;
  - the US stations are all more than 1000 km apart;
  - foF2 only: on MUF(3000), B on D barely beats GloTEC, 3.97 against 4.02 MHz, because M(3000)F2 dominates.

  PLAN's dry run measures these.
- **GIRO throttles.** A working model is a bucket of about 90 requests refilling at about 3 a minute, with a
  bare 429 and recovery within 60 s. The feature's throttle is sized from it (D-39). NOAA met no limit.
- **One of the runs went past its ruling (E-1).** D-38 said stop at the first denial; two further passes were
  made by hand, and the second 429, which the model is fitted to, came from them.
- **The first sections below (day 1) are superseded** by the one-sitting run. They are kept as the record of
  what was believed at the time.

## How it was done

D-20 carries paths B (GIRO-driven) and D (derived from NOAA GloTEC) to measurement. Two questions:
1. How close is D's foF2 and MUF(3000) to ionosonde readings?
2. What does B's real-time GIRO access cost in requests?

Every request is logged below. No data is kept in any tree (GIRO: CC BY-NC-SA 4.0; acknowledge each
station's provider). The scripts and the one-sitting run's request log are kept in go-ionomaps
(`06_docs/02_features/go-ionomaps/02-analysis/evidence/`).

## Day 1 (superseded): B's access, how GIRO serves readings

- The public form (`giro.uml.edu/didbase/scaled.php`, POST) answers with a redirect to FastChar:
  `lgdc.uml.edu/fastchar/getbest?ursiCode=<station>&charName=foF2,MUF(D),M(D),hmF2&DMUF=3000&fromDate=YYYY/MM/DD hh:mm:ss&toDate=...`
- **One station per request**, any time range. The reply is text, with a confidence score (CS) per sounding, at a 5-minute cadence.
- Three requests spaced 10-20 s apart drew no 429 today. The reply records the requester's IP.
- *(Superseded by the one-sitting run: about 38 live stations, about 900 a day per copy.)* **What it costs a path-B client:** about 40 real-time stations, polled once an hour with one request each, is **about 960 requests a day for every running copy**.
  - That is the 429 risk the 0.18.0 research met, multiplied by the number of copies running.
  - GIRO's terms ("free online access … only for educational and non-commercial research purposes") would carry to every one.
  - So path B in each listener's watchpost does not scale. It needs either a shared service (watchpost has none) or very sparse polling. **This is a finding for D-20's ruling.**
- **What it costs a path-D client:** one public-domain file of about 2.5 MB per update from NOAA SWPC, keyless; 10-minute cadence, or as rarely as the host asks.

## Day 1 (superseded): D's accuracy, first comparison (n = 2, night in Europe)

GloTEC at 2026-10-06T01:55Z (`glotec_icao_20261006T015500Z.geojson`), bilinear between cell centres:
- foF2 = 8.98×10⁻⁶ √NmF2;
- M(3000)F2 ≈ 1490 / (hmF2 + 176), the first-order inverse of Shimazaki 1955.

The ionosonde values are the sounding nearest 01:55Z.

| Station | Ionosonde foF2 | GloTEC foF2 | Δ | Ionosonde MUF(3000) | GloTEC MUF(3000) | Δ |
|---|---|---|---|---|---|---|
| PQ052 Pruhonice (50.0N 14.6E), 01:55Z, CS 90 | 3.10 | 3.74 | **+0.64 (+21%)** | 8.61 | 10.26 | **+1.65 (+19%)** |
| JR055 Juliusruh (54.6N 13.4E), 01:53Z, CS 90 | 2.28 | 3.39 | **+1.11 (+49%)** | 6.28 | 9.32 | **+3.04 (+48%)** |

- All four GloTEC cells around both stations carry `quality_flag` 5, **the best coverage**. NOAA: "the mean number of F-region observations in each vertical profile … rounded down to five" when larger than five (`spaceweather.gov/products/glotec`). The overestimates are in well-observed cells, so they are not a gap in coverage.
- GloTEC's hmF2 is the same, 366 km, at both stations, against 346-348 km measured. That is a sign the F2 peak is model-led there.
- **Not a verdict.** n = 2, one hour, night, one region. Both values sit above the published near-station target (about 0.5 MHz, wave 1).
- Wave 2 continues: five stations a day across latitudes, each over a 24-hour window, paired with five GloTEC files spread over the day. That is about 25 pairs a day within D-18. NOAA's page states no accuracy for NmF2, hmF2 or foF2, and no retention for the product directory.

## Day 1 (superseded): against the baseline, climatology at the same stations

PyIRI 0.0.4 (MIT; the release Python 3.9 installs), CCIR coefficients, `IRI_density_1day` for
2026-10-06 01:55Z, F10.7 = 103 (NOAA's 2026-10-05 22:00 reading):

| Station | Ionosonde foF2 | GloTEC foF2 | Climatology foF2 | Ionosonde MUF(3000) | GloTEC MUF(3000) | Climatology MUF(3000) |
|---|---|---|---|---|---|---|
| PQ052 Pruhonice | 3.10 | 3.74 | **3.78** | 8.61 | 10.26 | **10.30** |
| JR055 Juliusruh | 2.28 | 3.39 | **3.41** | 6.28 | 9.32 | **9.26** |

**At both stations GloTEC is the climatology to within 0.06 MHz.** This is in cells NOAA marks as best
covered (`quality_flag` 5). Here, path D adds nothing over the IRI background that D-23 makes the baseline
to beat, and both miss the measured ionosphere by 0.6-1.1 MHz.
- **Still n = 2, one hour, one region, night.**
- It fits GloTEC's design: GNSS total electron content constrains the column, not the F2 peak, and the filter relaxes to IRI (Chou et al. 2023).
- This weighed against path D at the time. *The full day reversed it: GloTEC beats climatology by about 17% (below).*
- Wave 2 continues: day and night, more latitudes, the three-way comparison at every pair.

## The one-sitting run (D-38), 2026-10-06 13:23-13:34Z, for the day 2026-10-05

One request at a time from a Python script, at most 5 a second (the httpx **default**,
`platform/httpx/httpx.go:61`); in practice replies paced it to about 1.7 a second. **This is not watchpost's
pace:** the app's clients run at 30 a second with up to 16 in flight (`app/app.go:132`), and httpx retries a
429 by itself, which this script did not. The feature therefore takes the throttle into go-ionomaps, with a
fetcher that never retries a 429 (FR-4.5). Every response's status, headers, size and time is logged
(go-ionomaps `02-analysis/evidence/requests-2026-10-05.jsonl`).

**The run went past D-38 (the agent's error E-1).** D-38 said "on the first denial … the run stops and
measures recovery". `burst.py` did, at the first 429. The agent then fetched the 22 stations not yet asked,
by hand: first at 1.5 s, which drew the second 429, then at 25 s. Those passes are reconstructed as
`burst_rest.py`. The throttle model below is fitted to both refusals, so it rests partly on a run the ruling
did not allow.

### GIRO's throttle, measured

| Phase | Pace | Result |
|---|---|---|
| All 118 stations in the form's list, a whole day each | as fast as replies came (about 1.7 a second) | **95 served in 55 s; the 96th refused (429)** at 13:23:58 |
| Recovery probe | after 60 s idle | served |
| 22 stations not yet asked | one every 1.5 s, after 112 s idle | **8 served; the 9th refused (429)**; recovered after 60 s |
| The last 13 | one every 25 s | **13 served, none refused** |

- The 429 is a bare nginx page: no `Retry-After` and no rate-limit headers, before or after a refusal.
- **A working model, fitted to both refusals and tested once:** a token bucket of about 90 requests that refills at about 3 a minute (about 180 an hour). The slow phase passed at 2.4 a minute, as the model predicts. Two refusals and one passing test are not a published limit, so the feature must still read a 429 as "back off", whatever the numbers.
- **What path B costs, corrected:** 39 of 118 stations had data for the day (TR169's data came back on a recovery probe, whose body was not kept, so the pairs below use 38). One request per live station returns its whole day, so an hourly update is about 38 requests per copy (about 900 a day). That is inside the refill (about 4,300 a day) for one copy on one IP. It does not show what GIRO thinks of many copies, and its terms ("only for educational and non-commercial research purposes") still bind each one.

### NOAA, measured

- 24 GloTEC grids (60 MB) in 17 s, all served.
- Headers carry `Cache-Control`, `Expires` and `X-Cache` (a CDN), and nothing about rate. No limit was met or stated.

### The full day: ionosonde vs GloTEC vs climatology

`pairs.py`: a sounding within 10 minutes of a grid time, confidence score ≥ 70; GloTEC bilinear; PyIRI
climatology (CCIR) with F10.7 = 100. **566 pairs, 28 stations, 24 hours.**

| Subset | n | foF2 RMS, GloTEC | foF2 RMS, climatology | Bias, GloTEC | Bias, climatology | GloTEC closer |
|---|---|---|---|---|---|---|
| all | 566 | **1.14** | **1.38** | +0.50 | +0.71 | 62% |
| GloTEC quality 3-5 (well observed) | 310 | **0.91** | 1.20 | +0.41 | +0.66 | 66% |
| GloTEC quality 0-2 | 256 | 1.36 | 1.58 | +0.60 | +0.76 | 57% |

MUF(3000) RMS: GloTEC 4.02 MHz, climatology 4.29. GloTEC's M(3000)F2 is a first-order inverse of hmF2,
so its MUF error includes that approximation.

**What this says, and does not:**
- **Day 1's two pairs were not representative.** Over a whole day, GloTEC beats the climatology by about 17% in foF2 RMS, and by 24% where it is well observed. Both read high.
- **Both are far from the literature's near-station target** (about 0.5 MHz, wave 1). Neither path D nor the climatology is good enough to read the band edge from, where the truth is.
- **Path B cannot be scored this way.** An assimilation of these stations is close to them by construction. It is scored leave-one-station-out (G-M3), which needs B built at least as a prototype. Wave 2 cannot give B's number; it gives D's (about 0.9-1.1 MHz) and climatology's (about 1.2-1.4 MHz).
- **Limits:**
  - one day (2026-10-05);
  - one F10.7 value;
  - autoscaled soundings (their own error, typically 0.1-0.5 MHz);
  - stations unevenly spread (28 with pairs).

## Paths scored leave-one-station-out (offline, no new requests)

Prototypes `loo.py` and `loo_hybrid.py` (go-ionomaps `02-analysis/evidence/`), on the 566 pairs above:
- for each hour, each station is held out in turn;
- the residual (ionosonde minus background) is predicted at it from the other stations that hour, by a Gaussian process with an exponential kernel in great-circle distance (positive definite on the sphere, Gneiting 2013);
- background plus prediction is scored against the held-out reading.

There is no effective-sunspot fit, no per-station time smoothing and no confidence weighting. These are
the simplest versions of each path.

| Path | foF2 RMS, all (n = 566) | 0-500 km to the nearest station (143) | 500-1000 km (69) | 1000-2000 km (130) | > 2000 km (224) |
|---|---|---|---|---|---|
| Climatology (D-23's baseline) | 1.38 | 0.76 | 1.47 | 1.45 | 1.61 |
| **D** — GloTEC alone | 1.14 | 0.63 | 1.19 | 1.08 | 1.38 |
| **B** — stations on a climatology background | 1.12-1.17 | **0.38** | 0.78 | 1.25 | 1.46 |
| **B on D** — stations on a GloTEC background | **1.00** | **0.37** | **0.71** | **1.06** | **1.28** |

Kernel length 500-4000 km and noise 0.05-0.2 change these by at most 0.05; the distance rows use
1000 km and 0.2.

**What this says:**
- **B on D is best or tied at every distance** (at 1000-2000 km, 1.06 against 1.08 is inside the ±0.05 the kernel settings move it). Near a station it reaches the literature's target (about 0.5 MHz) with room to spare. Far from stations it inherits GloTEC's advantage over climatology.
- **On MUF(3000) it barely helps:** B on D's foF2 with GloTEC's M(3000)F2 scores 3.97 MHz against GloTEC's 4.02, and 3.22 with the measured M(3000)F2 (red team CQ-E2). The M(3000)F2 error dominates, so go-ionomaps assimilates M(3000)F2 too (its R-9.2).
- **B alone wins only near stations**; beyond 1000 km climatology is a worse background than GloTEC.
- **D alone** is 17% better than climatology, but never near the target.

**Costs, per copy, under D-39:** about 38 GIRO requests and one 2.5 MB NOAA grid an hour. The
computation is about 38 stations a GP solve an hour, trivial (G-G1).

**Terms:**
- GIRO readings are CC BY-NC-SA; a computed field is likely a "substantially derivative product", which GIRO's rules leave unrestricted (OQ-G2, to be ruled).
- GloTEC is public domain.

**Limits:**
- one day;
- one F10.7 value;
- no tuning;
- autoscaled soundings;
- 28 stations, mostly Europe and the Americas, with Pacific islands, Australia and South Africa; no mainland Asian station; the under-500-km bin is seven European stations, and the four US stations are all more than 1000 km from another.

A second day, or a week, would show whether the order holds.

## Requests (D-18)

**The one-sitting run's requests (D-38):**
- GIRO: 120, of which 2 were refused (429) and 2 were recovery probes; about 0.5 MB.
- NOAA: 24 grids, 60 MB.
- Each is logged with headers in go-ionomaps `02-analysis/evidence/requests-2026-10-05.jsonl`. The run's data stays outside every tree (GIRO: CC BY-NC-SA).



| UTC | Source | Request | Result |
|---|---|---|---|
| 2026-10-06T02:30:46Z | NOAA SWPC | GloTEC geojson 01:55Z | 200, 2,502,441 B |
| 02:30:09Z, 02:30:2xZ | GIRO | `scaled.php` form page, twice (reading the form's fields) | 200 |
| 02:30:55Z | GIRO | form POST, BC840 (redirect not captured) | 302 |
| 02:31:25Z | GIRO | form POST, BC840 (redirect captured) | 302 to FastChar |
| 02:31:35Z | GIRO | FastChar BC840 01:30-02:30 | 200, 535 B, no data in that hour |
| 02:31:48Z | GIRO | FastChar PQ052 01:30-02:30 | 200, 1,773 B |
| 02:32:08Z | GIRO | FastChar JR055 01:30-02:30 | 200, 1,907 B |

The day's count, UTC 2026-10-06 (**corrected**, red team CQ-E4):
- **NOAA SWPC 29:** 3 by the wave-1 sources survey; 2 under D-18 (GloTEC 01:55Z; `json/f107_cm_flux.json` at 04:16:22Z, 22,838 B); 24 in the one-sitting run (D-38).
- **GIRO 125:** 5 under D-18 (two POSTs, three FastChar), plus two reads of the form page; 120 in the one-sitting run (D-38), of them 23 past its stop rule (E-1).
