---
title: "0.19.0 DISCOVER — wave 2 findings (measurements, D-18 budget)"
date: 2026-10-06
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "IN PROGRESS — day 1 of the D-20 measurements; continues within D-18's daily budget"
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

- All four GloTEC cells around both stations carry `quality_flag` 5. Its meaning is not yet found in NOAA's documentation; it may mark background-dominated cells.
- GloTEC's hmF2 is the same, 366 km, at both stations, against 346-348 km measured. That is a sign the F2 peak is model-led there.
- **Not a verdict.** n = 2, one hour, night, one region. Both values sit above the published near-station target (about 0.5 MHz, wave 1).
- Wave 2 continues: five stations a day across latitudes, each over a 24-hour window, paired with five GloTEC files spread over the day. That is about 25 pairs a day within D-18. `quality_flag` is to be read from NOAA's documentation.

## Requests (D-18)

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
- **NOAA SWPC 4 of 5**: 3 by the wave-1 sources survey, 1 here.
- **GIRO 5 of 5** data requests (two POSTs, three FastChar), plus two reads of the form page.
