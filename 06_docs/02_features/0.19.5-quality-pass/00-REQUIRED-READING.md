---
title: "0.19.5 quality and performance pass — REQUIRED READING"
date: 2026-11-05
phase: SHIPPED
sev: SEV-0
authority: HUM LEAD
status: "MANDATORY for the life of 0.19.5, at session start and after any compaction."
---

# Read this before touching 0.19.5

## What this release is

The regularly scheduled *.5 pass (D-1): no new features. Its five goals, the HUM LEAD's: no
regressions; performance where it can be had without re-architecting Charm or Bubble Tea; functions
consolidated where it is appropriate; every document, artifact and comment current; code a junior
maintainer can follow. The pass before it was 0.9.5 (`../watchpost-performance-quality-pass/`).

## Where things are

| | |
|---|---|
| Branch | none open: built on `feature/quality-pass`, cut from `main` (`4c803d99`, 0.19.3) (A-1), squash-merged as `release/0.19.5` (PR #37, `905ee845`) and `release/0.19.5-go1.27.4` (PR #38, `b4ab7dfb`); their branches are deleted |
| Phase | **SHIPPED** 2026-11-11: watchpost v0.19.5 released from `b4ab7dfb`, built with Go 1.27.4. BUILD built the plan's Q1 to Q9 (`04-development/implementation-plan.md`) in 53 batches; what each built is in `04-development/build-log.md`. REVIEW exited on `b11a2f40` (D-42): its red team is `08-reports/red-team-review.md`, M5's grade `08-reports/m5-blind-review.md`. P10 was re-ratified (D-43 to D-45). VALIDATE's UAT passed every item (D-46 to D-54, `07-readiness/validate-agenda.md`). The first release job stopped on GO-2026-6859, so CI and the release moved to Go 1.27.4 and the tag to that fix's merge (A-32, A-33, D-55). 0.19.6, the re-architecting, opens next (D-40) |
| Brief | `01-objectives/project-brief.md`, APPROVED (D-4): M1 no regression, M2 at most 80 / 90 / 140 KB a beat (idle, radio, console), M3 consolidation, M4 docs current, M5 a junior can find their way |
| Findings | `02-analysis/discover-findings.md`: what was checked, the five defects, the streams A to F |
| Scope | all of watchpost; a library finding becomes a library patch, pinned (D-2) |
| Rulings | `02-analysis/rulings.md`, every ruling the moment it is made; minor ones as A-n rows for veto |
| Lenses | `02-analysis/lens-*.md`, filed whole; each finding checked against the code before it counts |

## The rules that cost something

1. **No regressions.** A change lands only where a test pins the behaviour it touches; where none
   does, the pin is written first and seen to fail on a mutant.
2. **Charm and Bubble Tea are not re-architected.** Their composite's cost stands (0.19.0 D-327).
3. **The live app is not run for measurement**: it shares the HUM LEAD's Open-Meteo quota. Performance
   is measured from benchmarks, allocation pins and profiles of tests.
4. **Rulings one at a time**, with evidence (`file:line`), options, a recommendation and the strongest
   counter-argument, recorded verbatim.
5. **Consolidation is judged, not counted**: two functions alike in shape but answering different
   questions stay two (0.16.0's "two carriers, two questions"); the shared deck is shared on purpose.
