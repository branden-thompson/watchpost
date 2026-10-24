---
title: "0.19.0 Propagation overlays — REQUIRED READING, every session and after every compaction"
date: 2026-10-05
phase: ALL
sev: SEV-0
authority: HUM LEAD
status: "MANDATORY. Re-read at session start and after any context compaction, for the life of 0.19.0."
---

# Read this before touching 0.19.0

**Why this file exists.** 0.19.0 is one of three paired projects (D-1, D-3): watchpost 0.19.0, the new
library go-ionomaps, and go-tuiMaps v0.3.0. 0.18.0's condition for running releases in parallel still
holds: **the record is kept current at every step, and every lesson that can be a failing test becomes
one** (0.18.0 D-11).

## What this release is

A **Propagation mode** on the map (`P`) draws the HF bands' upper limit, MUF(3000) or foF2, over the whole world, from NOAA's GloTEC grids and GIRO's ionosondes assimilated over a climatology; the **Best Bands chart** (`B`) gives each band's status from the place to the continents and the hours of the day each is typically open; a **frequency's reach** (`F`) is drawn from the place.
The Observer's listener gets it on the map they have; the Broadcaster's operator gets a map of their own on `g`, centred on the station, with the same mode (D-195).
Its answers agree with what was heard on the air little better than the climatology alone does, and the release says so; no accuracy floor rides on it (D-250).

## Where things are

| | |
|---|---|
| Branch | `feature/propagation-overlays`, cut from `main` (31038ebf); squash-merged `release/0.19.0` at SHIP (D-5) |
| Phase | **REVIEW** (opened 2026-10-18): where it stands is the dated list below. What each batch built, and why, is in `04-development/build-log.md`; UAT's findings are D-rows. O2 moved to 0.19.1 (D-122) |
| Brief | `01-objectives/project-brief.md`, APPROVED (D-16) |
| Problems and metrics | `01-objectives/problem-statement.md`: PS-1 (the Broadcaster operator), PS-2 (the Observer ham), LOCKED (D-7); M1, M2, M4, M5, M5b and G1 (D-8, D-9; M3 retired at D-76); "open" is the reference circuit (D-73); floors set before the dry run (D-75); targets from the dry run (D-49) |
| Rulings | `02-analysis/rulings.md`: every ruling lands here the moment it is made; minor items are A-n rows, batched for veto (D-13) |
| Requirements | `01-objectives/requirements.md` — wins on any conflict once approved at the DISCOVER gate; opens with "What the listener sees" |
| Findings | `02-analysis/wave1-findings.md` (desk surveys), `02-analysis/wave2-findings.md` (measurements; the path's evidence is in go-ionomaps `02-analysis/evidence/`) |
| PLAN | `03-architecture-design/architecture.md` (the three projects, the update, go-ionomaps as watchpost uses it); `04-development/implementation-plan.md` (W0 to W10, the UAT); `07-readiness/dry-run.md` (what was measured; a plain verdict first); `07-readiness/uat-protocol.md` (M2, M5, M5b, live, D-89); `02-analysis/plan-questions.md` (Q-1 to Q-7, all ruled); targets in `01-objectives/problem-statement.md` ("Targets"); the M1/M4 instrument, `07-readiness/m1-m4-instrument.md` |
| Evidence | go-ionomaps `02-analysis/evidence/` (`week.py`, `offset.py`, `plan-dry-run/`, D-124); the data itself stays outside the tree (GIRO's terms) in a local folder named in the PLAN report |
| Red team | `08-reports/red-team-discover.md` (DISCOVER, rounds 1 and 2); `08-reports/red-team-plan.md` (PLAN exit, round 1: eight reviewers, every fork ruled), notes in `08-reports/reviewer-notes/`; `08-reports/red-team-build.md` (BUILD exit, rounds 1 and 2) and `08-reports/build-report.md`; `08-reports/red-team-review.md` (REVIEW exit, round 1: seven reviewers, D-245), the reports in full in `reviewer-notes/review-round-1.md` |
| Research | `00-research/rcc-muf-overlays.md` (0.18.0 D-238) |
| go-ionomaps (formerly go-giro-data, D-56) | `github.com/branden-thompson/go-ionomaps`, public, MIT (D-2); branch `feature/discover`; its brief, PS-G and requirements in `06_docs/02_features/go-ionomaps/01-objectives/` (local checkout `../go-ionomaps`) |
| go-tuiMaps v0.3.0 | paired release (D-3); branch `feature/propagation-fields` (A-7); HR-1..HR-5 ruled (D-32 to D-36), its log `06_docs/02_features/propagation-fields/02-analysis/rulings.md`; its brief APPROVED (its D-8) |
| Previous release | 0.18.0, shipped; its REFLECT is `../observer-maps/08-reports/reflect-report.md` |
| Follow-ups | `06_docs/follow-ups.md` only |

## Where REVIEW stands

- **2026-10-18** BUILD exited on `e889e72a` (D-233): its full gate and hosted CI green on both systems, P10 clean, the BUILD report approved (`08-reports/build-report.md`; PLAN approved 2026-10-07, D-139; DISCOVER 2026-10-06, D-84). Beside REVIEW, verdicts before VALIDATE (D-208): G1's 48-hour run, G10.4's fresh-day fetch and G-M3, and M1/M4's graded run on the same fresh days (D-221, D-224, D-228, D-232).
- **2026-10-12 to 10-17** the fresh days fetched. M1 and M4 graded, both short of D-99's targets (batch 52); G-M3 met and D-75's floor held, the Pacific worse than the climatology (batch 53). The instruments share go-ionomaps' `internal/hybrid` and `ionomapstest` (D-237).
- **2026-10-21** G1 met over its 48 hours against the map open in Forecast mode, -0.17 % CPU and -10.4 MB (D-236, D-241); against the map never opened, reported, both missed, +3.42 % and +40.0 MB, the open map's own cost (D-246). G1's hourly setting met on 2026-10-24, 2.66 MB an hour against 3.3 (A-83).
- **2026-10-22** the heap growth the soak showed, in 0.18.1 too, found in the history store's day cache and bounded (D-242); the fixed build runs beside 0.18.2 to show it.
- **2026-10-24** REVIEW exit's red team, round 1: seven blind reviewers (D-245), four do not exit, no Critical; its seventeen forks ruled (D-246 to D-262), the remediation under way (`08-reports/red-team-review.md`). The remediation's release candidates are go-ionomaps v0.1.0-rc.6 and go-tuiMaps v0.3.0-rc.5 (D-263 added `DaylightAt`), watchpost on both from batch 60.

## The rules that cost something

1. **MUF first, then accessibility (D-4)**, which amends 0.18.0 D-256. MUF's own words ship with the layer (M5b, D-9).
2. **The reference implementation has no licence.** go-ionomaps reimplements from the published science; nothing is copied from `arodland/prop` (brief C-5).
3. **Weather modes are never wider than the place's region** (0.18.0 D-8, D-28). Only the **Propagation mode** has its own world bound (D-21). The Broadcaster has its own map on `g`, centred on the station, its service area always drawn, and its Propagation mode on `P` (D-195).
4. **Live requests follow the feature's throttle (D-39, D-51, D-83):** GIRO bursts of at most 40 an hour, at most 2 a minute sustained, a 429 backs off 60 s doubling to 15 minutes, the live stations seeded and rotated; NOAA grids at most 6 an hour; only rate headers read at run time. GIRO's throttle is modelled from two refusals (D-38), as a bucket of about 90 refilling at about 3 a minute, not a published limit. **Since D-76 nothing is fetched until the Propagation mode is opened.** Open-Meteo is never probed for research; no kc2g.com request until asked.
5. **Rulings one at a time**, with evidence (`file:line` for every claim about the code, REFLECT L3), options, a recommendation and the strongest counter-argument, recorded verbatim.
6. **No code in PLAN** (0.18.0 D-13).
