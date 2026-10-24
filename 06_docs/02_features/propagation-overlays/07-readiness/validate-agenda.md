---
title: "0.19.0 — what VALIDATE decides: the agenda"
date: 2026-10-24
phase: REVIEW (for VALIDATE)
sev: SEV-0
authority: HUM LEAD
status: "OPEN — every item REVIEW sends to VALIDATE, in one place (D-255). An item leaves when it is ruled, its D-row named."
---

# What VALIDATE decides

Up: [rulings](../02-analysis/rulings.md) · [UAT protocol](uat-protocol.md) · [the M1/M4 instrument](m1-m4-instrument.md)

Each item is put to the HUM LEAD at VALIDATE with its evidence. They were scattered across the
instrument page, the build log and the G1 summary until the REVIEW red team said so (RB-7).

| # | Item | Evidence so far | Sent by |
|---|---|---|---|
| 1 | **M1 and M4 short of their targets.** M1 53.7 % against 57.4 % (the climatology 52.4 %), six of its ten bands at 50 % in every column; M4 68.4 % against 69.4 %. Whether M1 is re-scoped or retired for near-vertical areas, and whether the chart says its measured agreement | `m1-m4-instrument.md`; the graded run outside the tree | D-118, D-250 |
| 2 | **PS-1's claim in the PR**: an operator can ask from the station, M1's agreement beside it; the release ships with no accuracy floor | D-250 | D-250 |
| 3 | **G1, both comparisons**: met against the map open in Forecast mode (-0.17 %, -10.4 MB); missed against the map never opened (+3.42 %, +40.0 MB), the open map's own cost; and the Broadcaster arm, measured for 48 hours alone on GIRO | the 48-hour and hourly summaries; the Broadcaster run, to come | D-246 |
| 4 | **The default refresh**: every 10 minutes (13.94 MB an hour open) or hourly (2.66), with gm3's figure for the newest grid against one up to an hour old | the G1 runs; gm3's measurement, to come | D-253 |
| 5 | **The Pacific worse than the climatology** on the fresh days (foF2 1.47 against 1.22 MHz), said "about as good as typical here, or worse" beyond 2000 km | go-ionomaps' G-M3 on the fresh days | D-116, D-129 |
| 6 | **The memory growth's fix shown over days**: the fixed build beside 0.18.2, idle, 72 hours | the side-by-side run, under way | D-242 |
| 7 | **The changelog's known limits**, worded from this agenda's rulings | D-255 | D-255 |
| 8 | **M5 at 800 by 220**, recorded as a known limit, measured again after RP-4's fixes | go-tuiMaps' benchmark | D-261 |
