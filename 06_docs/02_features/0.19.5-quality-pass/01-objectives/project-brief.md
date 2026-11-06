---
title: "0.19.5 quality and performance pass — project brief"
date: 2026-11-06
phase: DISCOVER
sev: SEV-0
status: "APPROVED (D-4), M2 stretched"
---

# 0.19.5 — project brief

## The problem

Ten releases since the last pass (0.9.5) took watchpost to about 100,000 lines of Go, and the lens
survey (`../02-analysis/discover-findings.md`) found what that growth left behind:

- **The record and the code have drifted.** About 30 doc comments describe the wrong declaration,
  about 25 comments name symbols that do not exist, a third of `follow-ups.md` is closed rows, and the
  as-built maps and the atlas show things the code does not have.
- **Some rules are written in several places, and some copies already disagree.** Examples: the felt
  bands for earthquakes, the spoken temperature, "same place", the compass table.
- **Each frame and message copies more than it draws.** On the bench fixture an idle beat costs
  156 KB, a radio beat 170 KB and a console beat 310 KB, much of it copies of the model and of the
  frame.
- **A new maintainer cannot find their way.** Package front doors are stale, ruling IDs can't be
  resolved, core vocabulary is defined nowhere, and the most common first task (adding a Setting) has
  no walkthrough.
- **The regression net has gaps exactly where tidying would cut.** The radio deck's voice side, the
  Request window's keys, the cap helpers, and five surfaces without a whole-frame golden.

**Problem statement (D-4):** watchpost's code, comments and documents no longer agree with
one another, it does work each frame it then throws away, and a junior maintainer cannot safely find
or change the code a task needs. Cleaning that up is only safe once the behaviour it touches is
pinned.

## Goals (the HUM LEAD's, 2026-11-05)

1. No regressions.
2. Performance where it can be had without re-architecting Charm or Bubble Tea.
3. Functions consolidated where it is appropriate.
4. Every document, artifact and comment up to date.
5. Code a junior maintainer can follow.

## Metrics and targets (D-4)

| ID | Metric | Kind | Target |
|---|---|---|---|
| M1 | **No regression** | Primary | `make verify` green at every batch and hosted CI green on both systems. Every golden frame is byte-identical unless a D-row changes it. Every change lands behind a pin that was first seen failing on a mutant. The confirmed defects (DEF-1 to DEF-5) are fixed or ruled. |
| M2 | **Bytes per beat** (a message through the Router, then the frame), on the bench fixture | Primary | Idle Observer at most 80 KB (156.5 today). Radio playing with the visualiser at most 90 KB (169.7). Console at 160x74 at most 140 KB (310.5). Stretched by D-4 to take the medium memos (PF-7 to PF-10) as well as the small fixes, whose prototypes measured 104.9, 112.5 and 161.3; the console's share comes from its own masthead memo and a cached `Projection` (D-22). Each memo lands behind a completeness guard over the inputs it reads. M2 is measured after Q4 and again after Q6. A new beat pin holds them in bytes and allocations, and the existing frame pins are lowered to what lands. CPU time is not targeted: it is not measurable on a shared machine. |
| M3 | **Consolidation** | Secondary | Every confirmed DR finding has either landed or been ruled to stay two, with the reason recorded. `make dupes` at 25 nodes shows no unratified group and no stale ratified row. |
| M4 | **Docs current** | Secondary | Every DC finding is closed. `follow-ups.md` holds only open work. The as-built maps match the code. The new checks pass: SN-02 over types, values and fields, and a symbol check over `docs/extending.md`. |
| M5 | **A junior can find their way** | Secondary | `modes/tty` and `app` each have a current front door (`doc.go`). The vocabulary defines the five roles and the terms JR-6 lists. Ruling IDs can be resolved (`06_docs/ids.md`). There is an "add a Setting" walkthrough, and README says what the gate costs. **Graded at REVIEW (D-24)** by a blind reviewer with a junior persona (knows Go and the Bubble Tea README; not the harness, the ID schemes or the record), working from README's "Start here" and what it links, against a hidden answer key built by doing four tasks on a scratch copy (a toggle Setting, a keyless provider, a line on a console card, a spoken phrase). A task passes when every site is reached from the docs or its omission fails a guard that names it, no sentence routes to a wrong or missing file or symbol, and every ID met resolves. A fail is fixed before SHIP or ruled. |

## Requirements

- **R1** No behaviour changes without a D-row. A change that alters output is ruled first, and its
  pin shows the new output.
- **R2** Pins first. A batch writes or names the test that covers the code it touches before it
  changes that code.
- **R3** Charm and Bubble Tea are used as they are (0.19.0 D-327). No renderer, compositor or loop
  replacement.
- **R4** Consolidation is judged: copies are merged only when they answer the same question. The
  role split and the shared deck stay (0.16.0's required reading).
- **R5** The house rules hold: P10, AP-HIST-01, the other code standards, and no history in comments.
  A rule that hurts readability is changed by a ruling, not worked around.
- **R6** The live app is not run for measurement (the Open-Meteo quota). Performance is measured from
  benchmarks and allocation pins.
- **R7** A library finding is fixed in the library as a patch release and pinned (D-2).

## Scope

In: every package in watchpost, its docs, its Makefile and scripts, and its gate tools.
`third_party/` follows 0.9.5's capture, document, approve, change basis.

Out: new features, and a sweep of go-ionomaps and go-tuiMaps (D-2).
