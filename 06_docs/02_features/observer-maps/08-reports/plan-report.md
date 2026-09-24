---
title: "0.18.0 — Observer maps — PLAN REPORT"
date: 2026-09-23
phase: PLAN — exit
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
branch: feature/map-drawing
issue: "branden-thompson/watchpost#22"
status: "FOR THE HUM LEAD'S APPROVAL — PLAN's exit artefact"
---

# 0.18.0 — Observer maps — PLAN REPORT

## Bottom line up front

**PLAN is complete and recommends proceeding to BUILD.** 0.18.0 is planned as ten work packages
(W0 … W9) and 85 test-first tasks, each naming its files, its shapes and the test it writes
first, with no implementation code (D-13). Every requirement is traced to a task
(`04-development/implementation-plan.md`, "The trace"). It is built beside go-tuiMaps v0.2.0; the two
meet at go-tuiMaps' `radar-loops/03-architecture-design/integration-map.md`.

What a reader should know before the detail:

1. **Two halves, one release.** P1-a (W0–W7) draws alert areas, the basemap, fire and quakes, the
   description, pan and zoom and the legend on the library's v0.1.0, and starts at once. P1-b (W8–W9)
   adds radar loops and moves to v0.2.0, building on its release candidates (D-51). **0.18.0 ships on
   the final v0.2.0**, or without radar if the HUM LEAD rules so at REVIEW (RK-4, whose full cost is now
   written down).
2. **The listener's control flow is the design** (D-48): the map opens on "right now"; play runs from
   the oldest frame through the present into forecast; stop, reset, ←/→ scrub; pan and zoom with a
   loading indicator (D-49, moved into 0.18.0). Every key is chosen together, once, against the
   Observer's keymap (W1.16).
3. **Never an old frame, by a tested property** (D-45): the map renders on every event it owns, and
   after every event the stored frame's counters equal the library's.
4. **Honest about what it sends and holds.** Nothing is fetched before the listener asks (zone
   geometry included); radar is requested per state-regional region, never per view (D-47); "Clear map
   data" covers tiles, radar and zones; the listener is told what goes where.
5. **Function before performance** (D-46, D-50): M5 keeps "complete" as every alert and moves to 3.5 s;
   radar keeps two hours of loop; memory and speed are shaved later, against measurements.

## What BUILD will make

| Package | What | Lands on |
|---|---|---|
| W0 | The library at v0.1.0, the recorded fixtures, the gate that refuses code in plans | v0.1.0 |
| W1 | The window, its description, Settings, the size floor and degradation order, the cost warning, pan and zoom, the key evaluation, the legend | v0.1.0 |
| W2 | Drawing in `Update` on every event; the freshness guards; workers joined; M5's first arm | v0.1.0 |
| W3 | The basemap: no request before the listener asks, caches with one stated total, credits cleaned, the closed source list, the clear path | v0.1.0 |
| W4 | The bound: never wider than the selected location's region | v0.1.0 |
| W5 | Alert areas: one overlay per alert, the severity word, scope, partial areas named in words, zone refresh | v0.1.0 |
| W6 | Fire and quakes as layers | v0.1.0 |
| W7 | Without colour; M1b scored | v0.1.0 |
| W8 | Radar: sources and regions, the step Setting, validity and caps, the listener's controls, the loading indicator, M3/M5/M6, MRMS's unverified note | v0.2.0 rc → final |
| W9 | Moving to v0.2.0: the library's bound, `Report`, fetch options, retention and purge, the frame's counters, labels kept, the severity digit, the legend's P1-b keys | v0.2.0 rc → final |

## What PLAN decided

| Ruling | Decision |
|---|---|
| D-39, D-40 | PLAN opened; `requirements.md` normative |
| D-41, D-45 | The map is drawn in `Update`; it renders on every event; a freshness property guards it |
| D-42 | A partial area is drawn as found, labelled, and its missing zones named |
| D-43, D-46 | Numbers: M5 ≤ 3.5 s p90 (every alert complete), radar newest ≤ 3.0 s, loop ≤ 5.0 s; zones refreshed after 7 days; the cost warning above 2 MB or 40 requests; loop rates 1 and 2 frames a second |
| D-44, D-54 | A contextual picture-in-picture legend from a `[ L ] Legend` chip, in phase 1, with a key for everything shown |
| D-47, D-54 | Radar per fixed state-regional region, the scale set in UAT |
| D-48, D-49 | The listener's control flow; pan and zoom in 0.18.0; all map keys evaluated together |
| D-50 | Two hours of radar; the map source and radar step as Settings, so a listener can opt into more or back off |
| D-51 | P1-b builds on release candidates; ships on the final tag |
| D-52 | The description is text in 0.18.0; speaking it joins one voice pass with the other voice bugs (F-181); motion defaults to slow |
| D-53 | M1b re-scored on the description that ships; M6 measures ticks and key latency; timings recorded at SHIP |
| D-55 | The red team's batch |

## For approval

**RK-5, a basemap fallback.** OpenFreeMap is the one network basemap source. If it is down, national
and state views still draw from the embedded tiles, and a county view says the basemap is unavailable
(FR-3.4). **Proposed: accept one network source for 0.18.0**, with the embedded fallback and the notice
as they stand; a second source is a later ruling under FR-3.8's closed list.

## Critical analysis

**Internal plan check** found a zone fetch that broke "nothing before the listener asks", a package
layout that would not compile, tasks in the wrong order and phase-1 gaps; fixed before the red team
(`74274ba`).

**PLAN red team**, one round, three reviewers across both plans and the map; the round's record is in
go-tuiMaps (`radar-loops/08-reports/red-team-plan.md`), this feature's rulings in
`08-reports/red-team-plan.md`. Every finding is ruled or applied (D-44 … D-55).

## Risks carried into BUILD

| Risk | Where it stands |
|---|---|
| RK-1 correct parts, wrongly connected | Reachability gates on every wiring task; the freshness property; PTY journeys on the real binary |
| RK-4 v0.2.0 slips | P1-a ships alone by ruling at REVIEW, its full cost written down |
| M5 on the cold path | 3.5 s by ruling; measured at SHIP with a live sample |
| Keys | One evaluation of every map control before bindings (W1.16) |

## Gates

```
QUALITY GATE REPORT | watchpost 0.18.0 | SEV-0 | PLAN exit
-------------------------------------------------------------
  [PASS] approach_selected          : render placement ruled (D-41, D-45)
  [PASS] design_documented          : approach doc, integration map, atlas (PLAN diagrams labelled)
  [PASS] implementation_plan        : 10 packages, 85 tasks, every requirement traced; no code (D-13)
  [PASS] critical_analysis_complete : internal plan check + 1 red-team round, every finding dispositioned
  [PASS] make verify                : green at a13914c (the last code change); docs lane at this report's commit
  [PEND] human_approval             : this report
-------------------------------------------------------------
```

## Recommendation

**Proceed to BUILD**, starting with W0 (the library, the fixtures, the plan-code gate), and accept RK-5
as proposed.

## Source documents

`01-objectives/requirements.md` · `02-analysis/rulings.md` · `03-architecture-design/approach-1-render-placement.md` ·
`04-development/implementation-plan.md` · `08-reports/red-team-plan.md` · go-tuiMaps' integration map
and red-team record · the atlas
