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

## Where things are

| | |
|---|---|
| Branch | `feature/propagation-overlays`, cut from `main` (31038ebf); squash-merged `release/0.19.0` at SHIP (D-5) |
| Phase | **DISCOVER** (intake closed 2026-10-05, D-16) |
| Brief | `01-objectives/project-brief.md`, APPROVED (D-16) |
| Problems and metrics | `01-objectives/problem-statement.md`: PS-1 (the Broadcaster operator), PS-2 (the Observer ham), LOCKED (D-7); M1-M5b and G1 (D-8, D-9), targets set in DISCOVER |
| Rulings | `02-analysis/rulings.md`: every ruling lands here the moment it is made; minor items are A-n rows, batched for veto (D-13) |
| Research | `00-research/rcc-muf-overlays.md` (0.18.0 D-238) |
| go-ionomaps (formerly go-giro-data, D-56) | `github.com/branden-thompson/go-ionomaps`, public, MIT (D-2); branch `feature/discover`; its brief and PS-G in `06_docs/02_features/go-ionomaps/01-objectives/` (local checkout `../go-ionomaps`) |
| go-tuiMaps v0.3.0 | paired release (D-3); branch `feature/propagation-fields` (A-7); HR-1..HR-5 ruled (D-32 to D-36), its log `06_docs/02_features/propagation-fields/02-analysis/rulings.md`; its brief is written in DISCOVER |
| Previous release | 0.18.0, shipped; its REFLECT is `../observer-maps/08-reports/reflect-report.md` |
| Follow-ups | `06_docs/follow-ups.md` only |

## The rules that cost something

1. **MUF first, then accessibility (D-4)**, which amends 0.18.0 D-256. MUF's own words ship with the layer (M5b, D-9).
2. **The reference implementation has no licence.** go-ionomaps reimplements from the published science; nothing is copied from `arodland/prop` (brief C-5).
3. **Weather modes are never wider than the place's region** (0.18.0 D-8, D-28). Only the **Propagation mode** has its own world bound (D-21). The Broadcaster's own map (F-174) is regional, with a propagation readout, not a field (D-28 of this release).
4. **Live requests follow the feature's throttle (D-39):** GIRO bursts of at most 40 an hour, at most 2 a minute sustained, a 429 backs off 60 s doubling to 15 minutes; NOAA at most 6 grids an hour. GIRO's measured limit is a bucket of about 90 refilling at about 3 a minute (D-38). Open-Meteo is never probed for research; no kc2g.com request until asked.
5. **Rulings one at a time**, with evidence (`file:line` for every claim about the code, REFLECT L3), options, a recommendation and the strongest counter-argument, recorded verbatim.
6. **No code in PLAN** (0.18.0 D-13).
