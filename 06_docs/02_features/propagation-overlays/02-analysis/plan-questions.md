---
title: "0.19.0 PLAN — the questions DISCOVER left for PLAN"
date: 2026-10-07
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "IN PROGRESS. Each question goes to the HUM LEAD one at a time; the ruling lands in rulings.md the moment it is made."
---

# PLAN questions

The DISCOVER report (`08-reports/discover-report.md`, "PLAN questions") left six questions for PLAN. Three
need the week's scores (`07-readiness/dry-run.md`), and they wait for them. Each question below gives its
evidence, its options, a recommendation and the strongest counter-argument.

| # | Question | From | State |
|---|---|---|---|
| Q-1 | How the 40-request cap and the rotation probe share an update | go-ionomaps R-5.3; D-39, D-51 | **ruled D-87: A**, the probe inside the 40 |
| Q-2 | The space-weather scales beyond D-RAP | PM-E2; D-47 | **ruled D-88: B**, named, not modelled |
| Q-3 | A bias correction for a false "open" | RK-11 | waits for the week's signed bias |
| Q-4 | The effective-sunspot fit | CQ-N1; go-ionomaps requirements, after R-9.4 | waits for the week's fallback scores |
| Q-5 | The 400 km near-vertical radius | A-18 | **ruled D-100**: 400 km the default of a Setting (200, 400, 600) |
| Q-7 | The climatology's magnetic coordinates: a table neither R-4.3 nor the design names | found in PLAN's dry run | waits for the week's raw-against-refit scores |
| Q-6 | Can a shipped dataset reach #27's cap? | D-55 | **answered by measurement: no** (`dry-run.md`); no 0.18.1 hotfix; #27 is still fixed first in BUILD |

## Q-1 — The 40-request cap and the rotation probe

**Evidence.**
- D-39 caps one update's GIRO requests at a burst of at most 40, at most hourly, and at most 2 a minute sustained outside a burst.
- D-51 adds one probe of a not-live station per update, and rotation once more than 40 stations are live (40 an update, those left out asked first next time).
- With exactly 40 live stations, an update would make 41 requests. R-5.3 names the conflict.
- Today 39 stations are live (the week's fetch: 39 requests, all 200), so the conflict starts with the next station that comes up.
- Each update asks only for the readings since the last one held (R-5.2). A station skipped for one update therefore loses nothing: its readings arrive an hour later.

**Options.**
- **A. The probe counts inside the 40.** With 40 or more live, an update polls 39 live stations and makes the probe; the live station left out goes first next time. One number bounds every update, and `TestAnUpdateNeverExceedsItsBurst` stays one assertion.
- **B. The probe runs after the burst**, at least 30 s later, under D-39's 2 a minute sustained. An update makes at most 41 requests, but never 41 in one burst.
- **C. No probe while 40 or more are live.** New stations are found only when one drops out.

**Recommendation: A.** It is the simplest rule to state and to test, and it never exceeds D-39 in any reading of it. Under R-5.2 its cost is at most one live station an hour late.

**Strongest counter-argument.** A holds one live station back every hour once 40 are live. B keeps every live station current, and is still within D-39's words, since the probe falls outside the burst. But B's "within D-39" depends on reading the sustained rate as separate from the burst, and the bucket model (about 90, refilling about 3 a minute) has only two refusals behind it (D-38).

## Q-2 — The space-weather scales beyond D-RAP

**Evidence.**
- D-47 models both limits. The lower limit comes from daytime absorption, computed, and from NOAA SWPC's D-RAP for disturbances.
- D-RAP's product page (spaceweather.gov, fetched 2026-10-07T00:51:56Z) says its components are "driven by one-minute GOES X-ray flux data and by five-minute GOES proton flux data". So it covers flares (NOAA's R scale) and solar proton events (the S scale). **No geomagnetic input is named.**
- A geomagnetic storm (the G scale) is the third kind of disturbance. It lowers foF2 over hours to days (the upper limit), and it brings absorption at high latitudes.
- The measured fields see a storm as it happens: GIRO's readings and GloTEC's grid. The climatology does not. The hours ahead carry the background forward, with the station corrections decaying toward climatology (go-ionomaps R-1.3), so nothing in them foresees a storm's onset or its recovery.
- NOAA's scales feed, `services.swpc.noaa.gov/products/noaa-scales.json` (fetched 2026-10-07T00:51:12Z):
  - 1,107 bytes, with `ETag`, `Last-Modified` and `max-age=60`;
  - R, S and G now, the last 24 hours (`-1`), and probabilities or scales for the next three days.

**Options.**
- **A. D-RAP only, as ruled.** The words and the legend say that a geomagnetic storm shows only through the measured fields, and that the hours ahead do not foresee its course.
- **B. A plus NOAA's scales feed, named but not modelled.**
  - One small request per update, with validators, alongside D-RAP; credited, and named under the acknowledgement's "NOAA".
  - The chart and the words name any R, S or G level above 0.
  - While G is 1 or more, each forecast hour says it does not foresee the storm's course.
  - The physics is unchanged.
- **C. B plus storm modelling:** a storm-time foF2 correction and auroral absorption. More science to port and validate; best as a follow-up.

**Recommendation: B.** It is honest about the one disturbance the chart cannot see coming. It costs about 1 KB an update, and it adds no new model to validate.

**Strongest counter-argument.**
- B adds a source: a credit, a parser with its fuzzer, a fault case, and another thing to fail.
- The scales are coarse. "G0" read as "all clear" can mislead while D-RAP already shows trouble.
- During a storm the assimilated fields already carry its effect where stations report. B mainly helps the hours ahead and the places far from any station.

## Q-7: the climatology's magnetic coordinates (found in PLAN)

**Evidence** (PyIRI 0.1.7, as installed for the dry run, D-86):
- **The refits.** NRL's spherical-harmonic coefficients (`sh_library.py`, `coefficients/SH/foF2_*.nc`, `M3000F2.nc`, about 1.9 MB each) are in quasi-dipole latitude and magnetic local time. For geographic input, `sh_library.py:121-136` converts through `Apex(...)`, which reads `coefficients/Apex/Apex.nc` (2.8 MB). That table holds spherical-harmonic fits of apexpy's coordinates for each year from 1900 to 2030 (`sh_library.py:9-11`); outside that range the nearest year is used (`:1611`).
- **The raw tables.** CCIR and URSI (`main_library.py`) use the modified dip, computed from IGRF at 300 km (`main_library.py:137-141`, `igrf_library`). It reads `coefficients/IGRF/IGRF13.shc`, whose IGRF-13 coefficients are defined to 2025, so 2026 is extrapolated. IGRF-14 (IAGA, December 2024) covers 2025 to 2030.
- go-ionomaps' R-4.3 allows one kind of third-party data, the NRL refits. The design's provenance table says "spherical harmonics in modip", which is right for the raw tables, not for the refits.
- Either way the climatology needs a magnetic-coordinate table, and that table has a date limit.

**The week's scores decide which path is worth its table.** The partial week (2026-09-29 to 10-01) has the refit and raw CCIR level, at foF2 0.98 and 0.98 MHz and MUF 3.34 and 3.36 MHz.

**To rule after the scores:** which climatology path go-ionomaps ports (refits with Apex, or raw tables with IGRF-14), R-4.3 amended to name the table, and how the library says when its table's years run out.

