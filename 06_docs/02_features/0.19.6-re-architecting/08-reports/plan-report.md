---
title: "0.19.6 PLAN report"
date: 2026-11-16
phase: PLAN
sev: SEV-0
status: "PLAN exit approved (D-33); BUILD opens on B0"
---

# 0.19.6 PLAN report

## Verdict

PLAN is closed. The implementation plan (`../04-development/implementation-plan.md`) is approved
(D-33): ten batches, B0 to B9, about 24 full gates and 16 hours. M3's targets are set (D-27), the P10
re-key table is ratified with the pilot's results (D-10, D-31, D-32), and the agent's choices A-19 to
A-33 stand. BUILD opens on B0, batch 0's instruments.

## What was done

- **Three design agents**, read-only, designed the seams within the settled package map:
  - MS, the map's 2a and 2b: `design-MS-map-seam.md`;
  - AP, `app`'s deck, map data and kit: `design-AP-app-split.md`;
  - CK, the console, the kit and the test support: `design-CK-console-kit.md`.

  Their tables are filed in `../03-architecture-design/design-data/`. The agent checked each design's
  load-bearing claims against the code, and drew `../03-architecture-design/architecture.md` from them.
- **M4's before.**
  - A key builder fixed the four tasks and built their keys by leave-one-out on scratch copies of
    `e57afeaf` (36 test runs, one at a time).
  - A blind reviewer then traced the tasks. Grade: 0 of 4. The code routes held; the docs around them
    did not (`m4-before.md`).
- **The A-7 pilot** moved `everyTick` into `app/internal/run` on a scratch copy and measured the
  move's P10, inlining and test-time behaviour (`pilot-report.md`).
- **Five blind plan reviewers** judged the plan NOT READY, each on plan text, not design. The agent
  dispositioned every finding (`red-team-plan.md`).

## The rulings

| # | Ruled |
|---|---|
| D-20 | `CLAUDE.md` names 0.19.6's required reading |
| D-21 | R9: the seam adds no func-field read; the console gets the Dashboard's clock |
| D-22 | D-14's unservable tests: the drawn-value class, same killers |
| D-23 | the memo guard engine's recursion, two P10-01 rows |
| D-24 | the GOOS seam stays in the deck |
| D-25 | the map owns its Settings rows; the controls get a leaf package |
| D-26 | three pre-existing test-only exports recorded |
| D-27 | M3's full target table; W6 for a map Setting rises to 4 packages, accepted |
| D-28 | the performance criteria: benchstat fails on a confirmed 2 % slowdown |
| D-29 | M4's midpoint trace after B1 |
| D-30 | the instruments as tests in verify; `make graph` a gate; two exempt probes |
| D-31 | a density row per new package, in its batch |
| D-32 | no method shares a bare name with one it calls; each `New` its row |
| D-33 | PLAN exits; BUILD opens |

## What BUILD owes the HUM LEAD

- Before B2: the console's nine non-drawn test references, singly (A-33).
- Before B5: 2b's residual after B4's re-measure, singly (A-33).
- In each batch that creates a package: its density row, and any `New` row (D-31, D-32).
- Any benchstat slowdown D-28 catches.
- After B1: M4's midpoint grade (D-29).
- F-238: the `1f9a34e7` purge, the HUM LEAD's request to make.

## Lessons recorded

- **A file added during `make verify` fails the gate** as "the tree changed", after 40 minutes. Edits
  are finished and new files added with `git add -N` before a gate, and nothing is written into the
  tree while it runs.
- **A plan rests on what is committed.** The designs' scripts stayed in scratch, so B0.12's committed
  tool must reproduce every figure before the figures bind.
- **A premise is measured before it is ratified.** D-10's density assumption was wrong, and only the
  pilot showed it.
