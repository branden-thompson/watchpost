---
title: "0.19.6 re-architecting — REQUIRED READING"
date: 2026-11-14
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "MANDATORY for the life of 0.19.6, at session start and after any compaction."
---

# Read this before touching 0.19.6

## What this release is

The re-architecting that 0.19.5 found and ruled out of its own scope (0.19.5 D-40). `modes/tty` and
`app` are split into smaller packages along their seams (D-1). Nothing of D-2, the arbiter's
dissolving into the Director's step, is in 0.19.6: it is 0.19.7 (D-11). Charm is not re-implemented
(D-3).

## Where things are

| | |
|---|---|
| Branch | `feature/re-architecting`, cut from `main` at `e57afeaf` (A-1) |
| Phase | **BUILD**, opened 2026-11-16 when PLAN exited (D-33). **B0, the instruments, closed 2026-11-20** on `d33fc847`, its `make verify-full` green over the whole mutant corpus (7,877 s). **B1, the doc errors and M4's midpoint, closed 2026-11-21** on `da224fc8`, its `make verify-full` green (7,654 s), its scoped sweep 10 of 10 CAUGHT (D-51), M4's midpoint graded 0 of 4 and 2 of 4 on sites and routes (`08-reports/m4-mid.md`). **B2, the kit and the test support, closed 2026-11-23** on `9a3baba4`, its `make verify-full` green (6,837 s), its scoped sweep 232 of 232 CAUGHT, its 11 reports the move's re-points and D-58's split (D-69, D-70). **B3, the console, closed 2026-11-24** on `ceee5635`, its `make verify-full` green (6,693 s), its scoped sweep 136 of 136 CAUGHT, its 7 reports D-52's and D-58's splits, the console's move and prep 1's cut (D-75); B4, the map's 2a, is next; PLAN opened 2026-11-15 when DISCOVER exited (D-19, `08-reports/discover-report.md`); DISCOVER opened 2026-11-14. LEVEL-1, SEV-0, HUMAN LEAD, the full directive set (D-4). The baseline sweep ran on `e57afeaf` before any code moved: 1,051 of 1,058 caught, the 7 survivors the kept equivalents (A-3) |
| Rulings | `02-analysis/rulings.md`, every ruling the moment it is made; minor ones as A-n rows for veto |
| Survey | nine lens agents over one computed dependency graph (D-6, A-4, A-5): `02-analysis/lens-*.md`, reconciled by the agent into one package map (D-7): `02-analysis/package-map.md`, the settled document every later finding is judged against |
| Brief | `01-objectives/project-brief.md`, APPROVED (D-13) and amended (D-18): the cost and end state, the problem, metrics M1 to M7 for the four principles, R1 to R9, the scope |
| Red team | DISCOVER exit, round 1: `08-reports/red-team-review.md`, its five forks ruled D-14 to D-18 |
| Plan | `04-development/implementation-plan.md`, APPROVED (D-33), drawn from `03-architecture-design/architecture.md` and its three designs; PLAN's forks ruled D-21 to D-26, the plan red team's (`08-reports/red-team-plan.md`) and the pilot's (`08-reports/pilot-report.md`) D-27 to D-32 |
| M4's before | `08-reports/m4-before.md`: the blind trace on `e57afeaf`, 0 of 4; its keys in `08-reports/m4-key/`, never in a blind reviewer's copy |
| Issue | GitHub #41, "v. 0.19.6 - Re-architecting": the brief as approved and amended |
| Previous release | 0.19.5, shipped; its REFLECT is `../0.19.5-quality-pass/08-reports/reflect-report.md` (0.19.5 D-60) |
| Follow-ups | `06_docs/follow-ups.md` only |

## The rules that cost something

1. **D-1's four principles bind every change:** performance stays the same or improves; the
   architecture becomes more maintainable, traceable and understandable; best practice holds, P10
   included; no behaviour or function regresses.
2. **Charm, Bubble Tea and Lip Gloss are used through their interfaces, never re-implemented**, so
   either can be upgraded as released (D-3). Their composite's cost stands (0.19.0 D-327) unless an
   improvement by orders of magnitude is shown to the HUM LEAD.
3. **The shared deck is not split.** One deck, the Director arbitrating in both modes, MasterControl
   gating the air (`../0.16.0-broadcaster-ui/00-REQUIRED-READING.md`). A package split moves the deck
   whole. When 0.19.7 takes D-2, its work follows the role model and the disposition sheet, item by
   item.
4. **Every move keeps its mutants.** Before code moves, `06_docs/mutants/` is searched for its anchors;
   a moved anchor is re-pointed and re-vetted, and the closing sweep is judged against A-3's.
5. **The live app is not run for measurement** (the Open-Meteo quota). Performance is measured from
   benchmarks, allocation pins and profiles of tests.
6. **Rulings one at a time**, with evidence (`file:line`), options, a recommendation and the strongest
   counter-argument, recorded verbatim.
