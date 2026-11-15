---
title: "0.19.6 DISCOVER report"
date: 2026-11-15
phase: DISCOVER
sev: SEV-0
status: "DISCOVER exit approved (D-19)"
---

# 0.19.6 DISCOVER report

## Verdict

DISCOVER is ready to exit. The brief is approved and amended (D-13, D-18). The package map is
settled (A-18). Every fork the survey and the exit red team raised is ruled, D-1 to D-18. The
agent's choices, A-1 to A-18, stand: A-1 to A-8 with the brief (D-13), and A-9 to A-18 with no veto
on 2026-11-15 (recorded in D-14).

## What was done

- **The HUM LEAD set the release on 2026-11-14.** D-1 gives the four principles: performance the
  same or better; maintainability, traceability and understandability improved; best practice, P10
  included; no regression. D-3 rules Charm out of re-implementation, D-4 the full directive set
  (LEVEL-1, SEV-0), and D-7 the agent's duty to disposition disagreements before presenting.
- **The baseline sweep ran first**, on `e57afeaf` before any code moved (A-3): 1,051 of 1,058
  mutants caught, with the 7 survivors the kept equivalents, in 3,976 s. Its record is committed in
  `7c5e5772`, and every comparator reads it from there (A-9).
- **Nine read-only lens agents** surveyed one computed dependency graph (D-6, A-4, A-5): TS, AS, RX,
  PB, RM, TG, RC, GP and JN. Their reports are filed whole in `02-analysis/lens-*.md`.
- **The agent reconciled them** into one package map, `02-analysis/package-map.md`, dispositioning
  every disagreement there (C-1 to C-13, D-7).
- **The exit red team was five blind reviewers** (CQ, RR, PH, DQ, BQ), with their reports in
  `reviewer-notes/`. They confirmed the figures and the cuts, and found the record's rules
  inconsistent for the first code batch. `red-team-review.md` dispositions every finding: six
  disagreements, A-9 to A-18, and five forks put to the HUM LEAD (D-14 to D-18).

## What the survey found

- **The hard part is one type, not the file count.** `Dashboard` carries 752 methods in 68 files,
  and `livePipelines` 108 in 21. Go keeps a method in its type's package, so an area leaves only
  after its logic moves onto a type of its own (2a before 2b).
- **The net is whole-model tests.** 362 of 363 test files are white-box, so tests cannot simply move
  with their code. D-14 sets how they may change.
- **Moving code costs no performance by itself.** The cost is what a cut forces: a boxed model,
  func-field contexts that stop inlining, new message hops. R8, R9 and M1 hold each.
- **Many guards would go quiet, not red, after a move.** Batch 0 makes each loud first, seen
  failing.
- **The split barely shortens the gate.** The sweep runs each mutant's last killer first, so the
  saving is about 9 % of the sweep (B-1). The release's reason is principle 2, not gate time.

## The map

| | Packages |
|---|---|
| `modes/tty` (Map II) | the root (about 16,900 lines), `modes/tty/console`, a map package (2a in place, then 2b the move), a kit, and the memo guard's test-support package (D-15) |
| `app` (Map I) | the root, `app/internal/deck` (the deck whole), `app/internal/maps`, `app/internal/kit` |

The order is A-13's: batch 0; RM's slices 1 and 3; the `modes/tty` kit and console; 2a; the `app`
kit, deck and maps; 2b in three commits; the closing sweep and REVIEW. That makes eight code
batches and about 12 hours of gate (D-18).

## The rulings

| # | Ruled |
|---|---|
| D-1 | the split, under four principles |
| D-2 | the arbiter's dissolving in scope where it conforms (superseded by D-11) |
| D-3 | Charm not re-implemented |
| D-4 | LEVEL-1, SEV-0, the full directive set |
| D-5 | `CLAUDE.md` unchanged until PLAN |
| D-6 | lens agents over one computed graph, reconciled into one package map |
| D-7 | the agent dispositions disagreements before presenting |
| D-8 | the branch history rewritten to remove a leaked path |
| D-9 | the map's 2a and 2b both in |
| D-10 | P10's re-key table ratified at PLAN exit, new floors at BUILD exit |
| D-11 | nothing of D-2 in 0.19.6; D-2 proper is 0.19.7 |
| D-12 | the floor scan; go 1.26.11 in batch 0 |
| D-13 | the brief approved; A-1 to A-8 stand |
| D-14 | a test crossing a new boundary reads the same value through production API, proved by the same killers |
| D-15 | the memo guard's engine in one test-support package, its `unsafe` and `reflect` one ratified P10-09 row |
| D-16 | no new lock; `lp.mu` shared by pointer, with a static check and a deadlock test |
| D-17 | no package-size gate; the file rule holds, sizes reported at each REFLECT |
| D-18 | the brief amended, B-1 to B-12 |

## Owed to PLAN

From the rulings and `red-team-review.md`'s list:
- **Names:** the new packages' names (A-14), the test-support package's among them (D-15).
- **Targets and keys:** M3's numeric targets, and M4's four tasks with their before on `e57afeaf`
  (D-18).
- **P10:** the re-key table (D-10).
- **Seams:** the Settings and controls seam for 2b, and the GOOS seam after the deck moves, under
  D-14's rule.
- **The lock:** the inventory of every `lp.mu` section that touches a moving field (D-16).
- **Owed tests:** F-158's owed test and the `app` trace, written against the deck's entry point.
- **Costs and pilot:**
  - the case that `app/internal/maps` and the rest of `app/internal/kit` repay their cost, or they
    fold back;
  - A-7's pilot, with its same-name pairs.
- **The CLAUDE.md question** (D-5).

## Next: PLAN

An implementation plan, `04-development/implementation-plan.md`, orders the batches above. Each
batch names:
- the files it moves;
- the instruments it must first see failing;
- its mutants (A-11);
- its doc rows (package map §9);
- its re-key rows.

A red team reviews the plan before BUILD.
