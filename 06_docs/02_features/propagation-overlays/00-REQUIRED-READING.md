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
| Phase | **BUILD** (PLAN approved 2026-10-07, D-139; `08-reports/plan-report.md`). W0.0 (the wire cost), W1 (#27: roll-up parts, a byte bound per dataset, compact values), W2.0 (the map-mode type) W2.1 to W2.3 (the Propagation mode's key, bound and a place in no region) W2.5 (the acknowledgement), W4.1 (the data path), W3.1 and W2.4 (the fields drawn, L and U) are built (`04-development/build-log.md`, batches 1 to 8); UAT-1 (D-151) is under way, its rounds 1 to 4 built (batches 9 to 12, D-153 to D-167), then a quality pass across the three projects (batches 13 and 14, A-40, A-41) and the Status window grouped by what each host serves (batches 15 and 16, D-170 to D-173). **UAT-2 is the reference chart (D-174), upper limits only (D-175):** go-tuiMaps P3 and P4 and go-ionomaps G6.1 to G8.1 are built and pushed; W5.1 (best bands by continent, the chart window on B, the typical day, the radius Setting, the night on the map; D-176, D-177) is batch 17; go-tuiMaps P2.2 to P2.4 (the host ramp, the reach mark) and W5.2 (your frequency, D-178, D-179) batch 18; W5.4's hours ahead batch 19, its centre's answer (D-180) batch 20. UAT-2 is under way; batch 21 wires in go-ionomaps' stations (G9, G5), D-RAP's disturbed state, the per-place answers and the absorption term (D-181 to D-189), the build the HUM LEAD UATs against; go-ionomaps' G6.2 (the reference circuit's lower limit: Absorbed and Weak, D-186 to D-191) is batch 22, with the chart's cells (D-192) and the RX noise Setting (D-187); UAT-2's notes 1 and 2 are batch 23 - the Propagation mode by UI mode (D-193: Observer answers from the map's centre, the Broadcaster from its station with the centre a target, one roles table for a later Analyst) and the Map window inset to inset, 85% wide (D-194); UAT of batch 23 found the console's `g` unbound (FR-2.1, W7, unbuilt): batch 24 is the Broadcaster's own map - `g` on the console, on the station, its service area a red circle always drawn, tinted, stippled over the Propagation field (D-195, D-196), on go-tuiMaps' P9 (a host's zone). Batch 25 is the open items: the best frequency (D-197), each chart row's confidence by its nearest station (D-198), the offset compared and the no-readings and refresh Settings (W4.4, W4.5), the shared device and Settings (W7.2). Batch 26: Day / night as a map detail option in every map mode (D-199). **D-201: before BUILD's exit, the build-first list, in order** - batch 27 is W0.3 (one coordinate and locator parser in front of every lookup, A-55, D-202), batch 28 W0.2's per-source caps and FR-5.1/FR-5.2's honesty (the oldest input's age, stale past an hour, A-56), batch 29 W6.1 (the words in D-136's order, for speech, A-57), batch 30 W5.5 on go-ionomaps G2.5 (space weather named, A-58); then W3.1 and W3.2, W0.1 and W0.4, go-tuiMaps P5's tests, P6 and its checklist, go-ionomaps' missing tests and F-215, W5.3 and W9; the instruments (W10, go-tuiMaps P7/P8, go-ionomaps G10) a track of their own. UAT of batches 21 to 26 continues; W4.5/W4.6's confidence treatment awaits its mock-ups. PLAN's red team: two rounds (`08-reports/red-team-plan.md`). DISCOVER approved 2026-10-06 (D-84). O2 moved to 0.19.1 (D-122) |
| Brief | `01-objectives/project-brief.md`, APPROVED (D-16) |
| Problems and metrics | `01-objectives/problem-statement.md`: PS-1 (the Broadcaster operator), PS-2 (the Observer ham), LOCKED (D-7); M1, M2, M4, M5, M5b and G1 (D-8, D-9; M3 retired at D-76); "open" is the reference circuit (D-73); floors set before the dry run (D-75); targets from the dry run (D-49) |
| Rulings | `02-analysis/rulings.md`: every ruling lands here the moment it is made; minor items are A-n rows, batched for veto (D-13) |
| Requirements | `01-objectives/requirements.md` — wins on any conflict once approved at the DISCOVER gate; opens with "What the listener sees" |
| Findings | `02-analysis/wave1-findings.md` (desk surveys), `02-analysis/wave2-findings.md` (measurements; the path's evidence is in go-ionomaps `02-analysis/evidence/`) |
| PLAN | `03-architecture-design/architecture.md` (the three projects, the update, go-ionomaps' shape); `04-development/implementation-plan.md` (W0 to W10, the UAT); `07-readiness/dry-run.md` (what was measured; a plain verdict first); `07-readiness/uat-protocol.md` (M2, M5, M5b, live, D-89); `02-analysis/plan-questions.md` (Q-1 to Q-7, all ruled); targets in `01-objectives/problem-statement.md` ("Targets") |
| Evidence | go-ionomaps `02-analysis/evidence/` (`week.py`, `offset.py`, `plan-dry-run/`, D-124); the data itself stays outside the tree (GIRO's terms) in a local folder named in the PLAN report |
| Red team | `08-reports/red-team-discover.md` (DISCOVER, rounds 1 and 2); `08-reports/red-team-plan.md` (PLAN exit, round 1: eight reviewers, every fork ruled), notes in `08-reports/reviewer-notes/` |
| Research | `00-research/rcc-muf-overlays.md` (0.18.0 D-238) |
| go-ionomaps (formerly go-giro-data, D-56) | `github.com/branden-thompson/go-ionomaps`, public, MIT (D-2); branch `feature/discover`; its brief, PS-G and requirements in `06_docs/02_features/go-ionomaps/01-objectives/` (local checkout `../go-ionomaps`) |
| go-tuiMaps v0.3.0 | paired release (D-3); branch `feature/propagation-fields` (A-7); HR-1..HR-5 ruled (D-32 to D-36), its log `06_docs/02_features/propagation-fields/02-analysis/rulings.md`; its brief APPROVED (its D-8) |
| Previous release | 0.18.0, shipped; its REFLECT is `../observer-maps/08-reports/reflect-report.md` |
| Follow-ups | `06_docs/follow-ups.md` only |

## The rules that cost something

1. **MUF first, then accessibility (D-4)**, which amends 0.18.0 D-256. MUF's own words ship with the layer (M5b, D-9).
2. **The reference implementation has no licence.** go-ionomaps reimplements from the published science; nothing is copied from `arodland/prop` (brief C-5).
3. **Weather modes are never wider than the place's region** (0.18.0 D-8, D-28). Only the **Propagation mode** has its own world bound (D-21). The Broadcaster reaches it by a key, from the tower (D-57); its own regional map is a follow-up again (D-77).
4. **Live requests follow the feature's throttle (D-39, D-51, D-83):** GIRO bursts of at most 40 an hour, at most 2 a minute sustained, a 429 backs off 60 s doubling to 15 minutes, the live stations seeded and rotated; NOAA grids at most 6 an hour; only rate headers read at run time. GIRO's throttle is modelled from two refusals (D-38), as a bucket of about 90 refilling at about 3 a minute, not a published limit. **Since D-76 nothing is fetched until the Propagation mode is opened.** Open-Meteo is never probed for research; no kc2g.com request until asked.
5. **Rulings one at a time**, with evidence (`file:line` for every claim about the code, REFLECT L3), options, a recommendation and the strongest counter-argument, recorded verbatim.
6. **No code in PLAN** (0.18.0 D-13).
