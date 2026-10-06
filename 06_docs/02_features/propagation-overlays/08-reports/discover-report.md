---
title: "0.19.0 Propagation overlays, with go-ionomaps and go-tuiMaps v0.3.0 — DISCOVER report"
date: 2026-10-06
phase: DISCOVER exit
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
branches: "watchpost feature/propagation-overlays; go-ionomaps feature/discover; go-tuiMaps feature/propagation-fields"
status: "APPROVED by the HUM LEAD 2026-10-06 (D-84); PLAN opens"
---

# DISCOVER report — 0.19.0, go-ionomaps, go-tuiMaps v0.3.0

## Bottom line

**Recommended: exit DISCOVER, and approve the three requirements files.** The three projects have:
- locked problem statements (PS-1, PS-2, PS-G);
- a chosen path, with evidence and its limits;
- a definition of "open";
- floors set before any target;
- requirements traced to 83 rulings and 23 agent decisions;
- two red-team rounds dispositioned.

Nothing Critical is open.

**What 0.19.0 is now, after DISCOVER.** One place, the map window's **Propagation mode**, open to the Observer
and, by a key, to the Broadcaster from its tower. It shows:
- **two world maps**, MUF(3000) and foF2, with a day/night line;
- **"best bands now"** for the area around you and for the continents or any place you find;
- **"your frequency"**, the skip zone and reach of a frequency you type in, drawn and said;
- **forecast hours**, up to a day ahead, if they meet their floor.

Nothing is fetched until it is opened. Everything is said in words without the picture. watchpost transmits
nothing, and a first-open window says so.

**Around it:**
- 0.18.0's accessibility carry-over, gated before SHIP;
- the memo-key audit (#12);
- the `say` fixes;
- the history store's year-erasing bug (#27), fixed first;
- a Broadcaster audio-device Setting, so a screen reader need not speak into the broadcast.

**The libraries:**
- **go-ionomaps** (new; formerly go-giro-data) computes the fields: GIRO's ionosondes over NOAA's GloTEC,
  with a climatology fallback ported from NASA's PyIRI.
- **go-tuiMaps v0.3.0** learns to draw them.

## The problems, and how they will be judged

| | Statement (locked) | Judged by |
|---|---|---|
| **PS-1** (D-7) | A Broadcaster operator on HF cannot tell from the station whether their band reaches the area they serve | M1 reach agreement, M2 answerable from the station. **M3 retired (D-76):** the operator learns by looking; a band watch is F-209 |
| **PS-2** (D-7) | An Observer ham cannot tell, without leaving watchpost, which bands carry a signal to the places they want | M4 path agreement, M5 answerable, M5b in words |
| **PS-G** (D-11, amended D-19) | A developer has no global MUF/foF2 source they are free to build on and able to check | G-M1 free to build on (scored per layer, D-69), G-M2 reproducible, G-M3 fidelity, G-M4 fails out loud, G-G1 cost |

- **"Open"** means the reference circuit (D-73): SSB voice at 100 W with simple antennas, about +10 dB signal-to-noise. M1 and M4 are scored against WSPR spots normalised to it (D-22), with the IRI climatology as the baseline (D-23).
- **Floors are set before measuring (D-75):**
  - B on D must beat climatology on foF2 and MUF(3000), held out, in the US, or the background ships alone;
  - forecasts must be no worse than climatology, or hours ahead are cut.
- Every other target comes from PLAN's dry run, each by its own ruling (D-49).

## The path, and the evidence for it

**B on D (D-40):** GIRO station readings assimilated over a GloTEC-derived background; climatology as the
fallback. Chosen on one day's measurements (2026-10-05, 566 ionosonde pairs, held out):

| Path | foF2 RMS, all | within 500 km of a station | mainland US (4 stations) | Pacific (4 stations) |
|---|---|---|---|---|
| Climatology | 1.38 MHz | 0.76 | — | — |
| D: GloTEC alone | 1.14 | 0.63 | — | — |
| **B on D** | **1.00** | **0.37** | 0.51-0.81 | 1.2-1.8 |

**Blind spots:**
- one day;
- one F10.7 value;
- untuned kernels;
- the near-station bin is seven European stations;
- on **MUF(3000)**, B on D barely beat GloTEC (3.97 against 4.02 MHz), because M(3000)F2 dominates. go-ionomaps now assimilates M(3000)F2 too, and the dry run scores MUF first.

Every figure reproduces from `go-ionomaps/06_docs/02_features/go-ionomaps/02-analysis/evidence/`, and four
reviewers reproduced them.

## Doing nothing, or less (PM-I1)

| Option | What the operator gets | Cost | Status |
|---|---|---|---|
| **Do nothing** | a browser tab on prop.kc2g.com (no stated reuse terms) or a desktop tool; nothing in watchpost | none | rejected at intake (D-1, issue #25) |
| **A link out** | watchpost names a website | trivial | fails M2 and M5 ("without leaving") |
| **The background alone** (GloTEC + climatology, no GIRO) | maps and answers about 0.26 MHz worse near stations, no non-commercial input | smaller | **the pre-set fallback** if B on D misses its floor (D-75) |
| **Monitoring** (scheduled readout, band picker, closure notices) | a station that watches bands | a background fetch on every install, GIRO load, daily notices | **retired** for the reference chart (D-76); a band watch is F-209 |
| **Recording and backfill** | trends, once something reads them | the store's size cap (#27), grid storage | **deferred** (D-78, F-210) |
| **The Broadcaster's own map** | a geography map of the station's coverage | a new view | **deferred** (D-77, F-174) |
| **Station dots, the eSSN chart, the full P.533 planner** | more context | new components | **deferred** (D-50, F-189) |

**What remains** is the reference chart, its maps and words, and the fixes that ride with it.

## What else is out there (PM-E3)

| Tool | What it gives | Terms | Against 0.19.0 |
|---|---|---|---|
| **prop.kc2g.com** | global MUF and foF2 maps, eSSN, a planner | no licence on code; no stated reuse terms on output (wave 1) | the maps watchpost's users keep in a browser; go-ionomaps reproduces the method from published science |
| **VOACAP Online** (`voacap.com`) | "free professional high-frequency (3-30 MHz) propagation prediction": point-to-point and coverage maps, monthly-median predictions, with antennas and power | free to people; "all automated access … strictly prohibited unless agreed upon in advance" | a monthly-median model, not a nowcast; cannot be fetched by watchpost; the closest thing to "your frequency" for planning |
| **PSKReporter map** | live reception reports by band | no stated terms (wave 1) | evidence rather than estimate; the observed-spots idea is F-207 |
| **HamClock** | a desktop dashboard with propagation panels | unverified: its site refused connection on 2026-10-06 | not assessed |

**What 0.19.0 adds that none does:** live, measurement-driven maps and answers inside a terminal weather
station, said in words for a listener without the picture, with every source's terms stated.

## GIRO and the fleet (PM-N4, kept under D-41)

An update asks GIRO about 39 times (one request per live station) and happens only while the Propagation
mode is open (D-76), at most hourly (D-39). GIRO's throttle is modelled as a bucket of about 90 refilling at
about 3 a minute, per IP. It is not published.

| Copies with the mode open in one hour | GIRO requests that hour | Note |
|---|---|---|
| 1 | about 39 | inside the refill (about 180 an hour) |
| 10 | about 390 | many installs, many IPs: GIRO sees the total |
| 100 | about 3,900 | a load an academic service may refuse |
| 1,000 | about 39,000 | beyond any reasonable courtesy |

If GIRO refuses, go-ionomaps runs on its background (D-40, D-75), and F-208 records the trigger to revisit
contact (D-41) or a shared relay. The fleet is unknown; watchpost has no telemetry, by design.

## The red team

- **Round 1:** eight blind reviewers, five "do not exit".
- **Round 2:** four reviewers. No Critical; all four asked for dispositions, which D-72 to D-83 and the fix batches made.

Records: `red-team-discover.md`, `reviewer-notes/`.

**Two defects in shipped code were found and are in scope:**
- watchpost's HTTP client retries a 429 (FR-4.4);
- the history roll-up erases a year (#27, FR-6.5).

**The agent's own errors (E-1 to E-13)** are recorded there. Most serious:
- E-1: a measurement run went past D-38's stop rule, and the throttle model leans on it.
- E-8: a question to the HUM LEAD rested on a wrong premise (the world-cities list already ships).
- E-11: a wrong claim about US stations.

## What PLAN inherits

- **The dry run (FR-10.6), before any target:**
  - MUF and foF2 held out, by station and distance, with signed bias;
  - the NRL refits;
  - GloTEC's availability;
  - the forecast's error;
  - WSPR density at near-vertical range under the reference circuit;
  - the answers' cost per keypress;
  - G1 over 48 hours.

  Then each target is ruled, against D-75's floors.
- **PLAN questions:**
  - a bias correction (RK-11);
  - the effective-sunspot fit (CQ-N1);
  - how the 40-request cap and the rotation probe share an update (go-ionomaps R-5.3);
  - the space-weather scales beyond D-RAP (PM-E2);
  - the 400 km near-vertical radius (A-18);
  - whether a shipped dataset can reach #27's cap (D-55).
- **The provenance table** (D-53), and the build order: #27 first, then O1, O2 (gated), O3.

## Blind spots of this report

- **The evidence is one day.** The floors exist because of it.
- **Every stakeholder in the record is the HUM LEAD.** UAT is graded by one person (D-68).
- **Two figures are reviewers' general knowledge, not measured:** WSPR's decode threshold, and HamClock.
- **GIRO's throttle is a model** fitted to two refusals, one of them from a run past its ruling (E-1).

## The rulings asked

1. **Approve DISCOVER exit** for watchpost 0.19.0, go-ionomaps and go-tuiMaps v0.3.0, so PLAN opens.
2. **Approve the three requirements files** as the requirements PLAN designs against:
   - watchpost `01-objectives/requirements.md`;
   - go-ionomaps `01-objectives/requirements.md`;
   - go-tuiMaps `propagation-fields/01-objectives/requirements.md`.
