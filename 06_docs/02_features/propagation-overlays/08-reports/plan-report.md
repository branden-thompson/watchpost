---
title: "0.19.0 Propagation overlays, with go-ionomaps and go-tuiMaps v0.3.0 — PLAN report"
date: 2026-10-07
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "APPROVED by the HUM LEAD 2026-10-07 (D-139); the batched decisions stand (D-140); BUILD opens with W0.0 then W1"
---

# PLAN report — 0.19.0, go-ionomaps v0.1.0, go-tuiMaps v0.3.0

## Bottom line

- **The plan is ready to build, and the evidence behind it reproduces** (with one correction, below).
  - Three implementation plans hold every test the requirements name. A mechanical trace found none missing: watchpost 102, go-ionomaps 73, go-tuiMaps 30 in scope.
  - The architecture gives the three projects one shape.
  - Every target is ruled from measurements, each with what it was measured on.
- **The chart's value is the current hour.** On PLAN's week, held out:
  - in the mainland US it beat the climatology by about a fifth on foF2 and a quarter on MUF (0.66 against 0.82 MHz; 2.08 against 2.88);
  - near a reporting station it halved the error.
- **Hours ahead are the climatology, labelled "typical for this hour"** (D-109). Nothing tested beats it in a storm. The one method that beat it on quiet days needs a field from the day before, which 0.19.0 never holds.
- **The weak places are said, not hidden.**
  - In a storm, every hour ahead says typical conditions may not hold (D-88).
  - In the Pacific, and anywhere far from a station, the answer says it is "about as good as typical here, or worse" (D-116, D-129).
  - A screen reader cannot yet follow the mode's focus, and Help says so (D-133); 0.19.1 fixes it.
- **The PLAN red team ran two rounds.**
  - Round 1: eight reviewers, two Criticals, both closed.
  - Round 2: four reviewers, no Critical.
  - Thirty rulings came out of them (D-109 to D-138), with two fix batches.
- **The agent made six errors in PLAN** (E-14 to E-19), each found and corrected. The worst two:
  - E-15: a forecast method was put to the HUM LEAD without saying it could not run.
  - E-19: two forks quoted figures from the wrong climatology run.
- **0.19.0 is O1 and O3.** O2, the accessibility carry-over, becomes 0.19.1, the very next release (D-122).

## What was planned

| Project | Plan | Shape |
|---|---|---|
| watchpost 0.19.0 | `04-development/implementation-plan.md`: W0 to W10 (O2's W8 now 0.19.1), 37 tasks | `03-architecture-design/architecture.md`: the Propagation mode, a dedicated view (D-102); two seams, the update's one owner and pure answers (A-30); the go-ionomaps shape |
| go-ionomaps v0.1.0 | `04-development/implementation-plan.md`: G0 to G10, 37 tasks | `03-architecture-design/design.md`: the packages, one update, the provenance of every method (D-53) |
| go-tuiMaps v0.3.0 | `propagation-fields/04-development/implementation-plan.md`: P0 to P8, 17 tasks | fields over water, the MUF and foF2 presets, the terminator, the reach edge without colour (L-2.6) |

**How a listener's answer is made:**
- **The current hour is the hybrid (D-101):** GIRO's station residuals assimilated over GloTEC for foF2, and over the climatology for M(3000)F2.
- **The climatology** is NRL's CCIR refit (D-43, A-28), on a 30-day mean F10.7 (D-104), in magnetic coordinates go-ionomaps computes itself from IGRF-14 (D-107).
- **Disturbance** comes from D-RAP, and NOAA's storm scales are named (D-88).
- **Refresh:** fields follow each new GloTEC grid while the mode is open (D-94), a Setting with its cost (D-120). Nothing is fetched while it is closed, and the client talks only to the library's declared hosts (D-113).

**Order in BUILD:**
1. **W0.0** measures GloTEC's wire cost before any code (D-111).
2. **W1**, #27's roll-up fix, is the first code change (D-55).
3. Then the libraries' release candidates and O1 (D-3), and O3.

The size is about 70 batches across the three, scaled from 0.18.0's measured rate (RK-5).

## The dry run, and the targets set from it

The dry run (`07-readiness/dry-run.md`) opens with a plain verdict (D-126). It measured:
- a week of GIRO and GloTEC data (all 95 requests answered 200);
- the cost of every answer and of the update (spikes, placeholder physics);
- go-tuiMaps' globe frame;
- WSPR's near-vertical density at +13 dB (D-91, ITU-R F.339-8);
- G1's harness on today's binary.

| Metric | Target | Ruling |
|---|---|---|
| M1, M4 (agreement with WSPR) | the climatology's + 5 points; ≥ 200 scenarios, ≥ 3 days, by band; becomes "no worse than climatology" if the baseline exceeds 90% | D-99, D-118 |
| M2, M5, M5b (UAT, graded against watchpost's own answer, D-92) | 8 of 10; within 30, 45 and 60 s; a live sitting (D-89), the record redacted (D-115) | D-93 |
| G1 (station cost) | ≤ 1% CPU; ≤ 25 MB added RSS measured over 48 h; downloads on the wire, re-ruled from W0.0 | D-95, D-111, D-112, D-131 |
| go-ionomaps G-M3 | ≥ 15% below climatology in the US, ≥ 10% overall; near stations foF2 ≤ 0.5 and MUF ≤ 1.6 MHz; on ≥ 3 fresh days | D-108, D-132 |
| go-ionomaps G-G1 | cold open ≤ 0.5 s; refresh ≤ 50 ms; live heap ≤ 15 MB | D-98, D-112 |
| go-ionomaps G-M2 | bit-identical per architecture; ≤ 0.001 MHz across | D-97 |
| go-tuiMaps M5 | the full frame including the reach overlay ≤ 16 ms at 400 × 110, ≤ 8 ms at 200 × 56 | D-96, D-137 |
| go-tuiMaps M1, M6 | 10 places, 8 of 10 in both arms; 5 clean full runs | D-117 |

**How targets are enforced:**
- Timing is gated by machine-independent proxies; wall-clock is a release step on the reference machine (D-119).
- A miss of any target other than a floor comes to the HUM LEAD at VALIDATE, never re-set silently (D-118).

**Their blind spots:**
- one week, one solar level, one storm;
- four mainland-US stations;
- spikes on one machine;
- G-M3's near-station limits drawn from the whole week, tuning days included.

That is why G-M3 and the floors are judged on fresh days (D-132).

## The red team

| Round | Reviewers | Verdict | Outcome |
|---|---|---|---|
| 1 (D-90) | eight: four axes, the PLAN lens, Accessibility, InfoSec, Performance | all "do not exit"; two Criticals | 19 forks ruled (D-109 to D-127); a fix batch |
| 2 (D-128) | four: the PLAN lens, Docs, Code, the combined personas | "do not exit yet"; no Critical; both Criticals confirmed closed | 10 rulings and corrections (D-129 to D-138); a fix batch |

**The two Criticals:**
- **The forecast could not run.** Six of eight reviewers found it independently; D-109 made hours ahead the climatology, labelled typical.
- **The terminal cursor never follows focus.** D-110 left it in O2, and D-133 says the limit in Help.

**What the rounds changed most:**
- **The API shape.** go-ionomaps' response carries rate headers only (the import ban can hold). The snapshot carries the inputs' own times. Answers take one place or many, bounded. The limit is typed (A-29, A-30).
- **The update.** It has one owner and a deadline. A cancelled burst still spends GIRO's hour. A first picture comes from NOAA before the GIRO burst ends.
- **The record.** It now reproduces, and its stale statements are gone.

The ledger, every finding and its route, is `red-team-plan.md`; the reviewers' notes are in `reviewer-notes/`.

## The agent's errors in PLAN

| # | Error | Corrected |
|---|---|---|
| E-14 | A commit went out after a failed docs lane (the atlas) | `make verify` green on it; commits chained to the verdict since |
| E-15 | D-103's blend was put without saying it needed a field 0.19.0 never holds | D-109 |
| E-16 | The forecast's "measured typical error" had no source at run time | D-109, D-134 |
| E-17 | The dry run said the floor was met "every day"; 10-03 in the US misses on foF2 | reworded |
| E-18 | The agent said all eight reviewers had reported when seven had | corrected in the next message |
| E-19 | Forks 1 and 8 quoted figures from a run on the raw CCIR tables, which D-43 forbids shipping; the Pacific result flipped | D-129, D-130; sections 10 and 11 published |

**The lesson for BUILD:** a figure put to the HUM LEAD is pasted from the committed scorer's output at the committed version, never from an intermediate run. The ledgers already quote `week.py`'s section numbers for this reason.

## What BUILD inherits

- **First:**
  - W0.0, the wire-cost measurement;
  - W1, #27;
  - W0.4's docs checks (the trace, `file:line`, the atlas);
  - W2.0, the map-mode type before a third mode.
- **The fresh days for G10.4** (D-132): `dry_fetch.py` under D-39, at least 3 days, kept outside the tree.
- **The evidence:**
  - committed instruments in go-ionomaps `02-analysis/evidence/` (`week.py`, `offset.py`, `plan-dry-run/`, D-124);
  - the data, with the full reviewer reports and the PyIRI environment's `pip freeze`, in a local folder outside every repository (`~/watchpost-0.19.0-plan-evidence/`), since GIRO's terms keep it out of the tree.
- **A gap stated, not fixed:** the "no code in PLAN" check (`plancode`) runs over watchpost's records only. go-ionomaps' gate arrives at G0 with one; go-tuiMaps' hits are in its v0.1.0 design docs only.

## Blind spots of this report

- **The evidence is one week**, and the cost figures come from spikes with placeholder physics on one machine.
- **Every stakeholder in the record is the HUM LEAD**, and UAT is graded by one person (D-68). M5b measures a sighted reader of the words (D-110).
- **GIRO's throttle is a model** from two refusals; the week's fetch met no 429.
- **The agent wrote the plans and the scorer that grade them.** The fresh days (D-132) and the release steps (D-119) are the independent checks.

## The rulings asked

1. **Approve PLAN exit** for watchpost 0.19.0, go-ionomaps v0.1.0 and go-tuiMaps v0.3.0, so BUILD opens with W0.0 and W1.
2. **The agent's decisions batched for veto** (D-13); each stands unless vetoed:
   - **A-25:** M5b gets its own ten scenarios.
   - **A-27:** FR-1.17 is met off the UI goroutine.
   - **A-28:** the CCIR refit ships, not URSI.
   - **A-29:** round 1's design choices: the API shape, the update, the bounds, the constants' table, W0.4, W2.0.
   - **A-30:** round 2's design choices: the two seams, NOAA first, the cancelled burst spends the hour, the inputs' times, "measured" within 500 km, the removals.

   A-26 was ratified as D-120.

On approval, the exit commit regenerates `06_docs/architecture-atlas.html` under the full `make verify` (A-24).
