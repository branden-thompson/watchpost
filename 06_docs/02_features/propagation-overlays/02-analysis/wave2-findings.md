---
title: "0.19.0 DISCOVER — wave 2 findings (measurements, D-18 budget)"
date: 2026-10-06
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "IN PROGRESS — day 1 (D-18 budget) and the one-sitting run (D-38) recorded; the path ruling (D-20) next"
---

# Wave 2 findings

D-20 carries paths B (GIRO-driven) and D (derived from NOAA GloTEC) to measurement. Two questions:
1. How close is D's foF2 and MUF(3000) to ionosonde readings?
2. What does B's real-time GIRO access cost in requests?

Every request is logged below. The data stays in the agent's scratchpad, never in the tree (GIRO:
CC BY-NC-SA 4.0; acknowledge each station's provider).

## B's access: how GIRO serves readings

- The public form (`giro.uml.edu/didbase/scaled.php`, POST) answers with a redirect to FastChar:
  `lgdc.uml.edu/fastchar/getbest?ursiCode=<station>&charName=foF2,MUF(D),M(D),hmF2&DMUF=3000&fromDate=YYYY/MM/DD hh:mm:ss&toDate=...`
- **One station per request**, any time range. The reply is text, with a confidence score (CS) per sounding, at a 5-minute cadence.
- Three requests spaced 10-20 s apart drew no 429 today. The reply records the requester's IP.
- **What it costs a path-B client:** about 40 real-time stations, polled once an hour with one request each, is **about 960 requests a day for every running copy**.
  - That is the 429 risk the 0.18.0 research met, multiplied by the number of copies running.
  - GIRO's terms ("free online access … only for educational and non-commercial research purposes") would carry to every one.
  - So path B in each listener's watchpost does not scale. It needs either a shared service (watchpost has none) or very sparse polling. **This is a finding for D-20's ruling.**
- **What it costs a path-D client:** one public-domain file of about 2.5 MB per update from NOAA SWPC, keyless; 10-minute cadence, or as rarely as the host asks.

## D's accuracy: first comparison (n = 2, night in Europe)

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

## Against the baseline: climatology at the same stations

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
- This weighs against path D, and toward measurements of the peak itself: ionosondes (path B), or INGV's measured Europe map where it applies.
- Wave 2 continues: day and night, more latitudes, the three-way comparison at every pair.

## The one-sitting run (D-38), 2026-10-06 13:23-13:34Z, for the day 2026-10-05

watchpost's way: one request at a time, at most 5 a second (`platform/httpx/httpx.go:61`), every
response's status, headers, size and time logged (scratchpad `burst-2026-10-05/requests.jsonl`).

### GIRO's throttle, measured

| Phase | Pace | Result |
|---|---|---|
| All 118 stations in the form's list, a whole day each | as fast as replies came (about 1.7 a second) | **95 served in 55 s; the 96th refused (429)** at 13:23:58 |
| Recovery probe | after 60 s idle | served |
| 22 stations not yet asked | one every 1.5 s, after 112 s idle | **8 served; the 9th refused (429)**; recovered after 60 s |
| The last 13 | one every 25 s | **13 served, none refused** |

- The 429 is a bare nginx page: no `Retry-After` and no rate-limit headers, before or after a refusal.
- **A working model, fitted to both refusals and tested once:** a token bucket of about 90 requests that refills at about 3 a minute (about 180 an hour). The slow phase passed at 2.4 a minute, as the model predicts. Two refusals and one passing test are not a published limit, so the feature must still read a 429 as "back off", whatever the numbers.
- **What path B costs, corrected:** 38 of 118 stations had data for the day. One request per live station returns its whole day, so an hourly update is about 38 requests per copy (about 900 a day). That is inside the refill (about 4,300 a day) for one copy on one IP. It does not show what GIRO thinks of many copies, and its terms ("only for educational and non-commercial research purposes") still bind each one.

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

## Requests (D-18)

**The one-sitting run's requests (D-38):**
- GIRO: 120, of which 2 were refused (429) and 2 were recovery probes; about 0.5 MB.
- NOAA: 24 grids, 60 MB.
- Each is logged in the scratchpad (`burst-2026-10-05/requests.jsonl`) with headers. The run's data stays outside the tree (GIRO: CC BY-NC-SA).



| UTC | Source | Request | Result |
|---|---|---|---|
| 2026-10-06T02:30:46Z | NOAA SWPC | GloTEC geojson 01:55Z | 200, 2,502,441 B |
| 02:30:09Z, 02:30:2xZ | GIRO | `scaled.php` form page, twice (reading the form's fields) | 200 |
| 02:30:55Z | GIRO | form POST, BC840 (redirect not captured) | 302 |
| 02:31:25Z | GIRO | form POST, BC840 (redirect captured) | 302 to FastChar |
| 02:31:35Z | GIRO | FastChar BC840 01:30-02:30 | 200, 535 B, no data in that hour |
| 02:31:48Z | GIRO | FastChar PQ052 01:30-02:30 | 200, 1,773 B |
| 02:32:08Z | GIRO | FastChar JR055 01:30-02:30 | 200, 1,907 B |

Today's count, UTC 2026-10-06:
- **NOAA SWPC 5 of 5**: 3 by the wave-1 sources survey, 2 here (GloTEC 01:55Z; `json/f107_cm_flux.json` at 04:16:22Z, 22,838 B).
- **GIRO 5 of 5** data requests (two POSTs, three FastChar), plus two reads of the form page.
