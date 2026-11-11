---
title: "0.19.5 plan red team"
date: 2026-11-06
phase: PLAN
sev: SEV-0
status: "Round 1 closed: every finding ruled or carried into the plan"
---

# 0.19.5 plan red team, round 1

The plan red team (D-20) was three blind, read-only reviewers. Each read the brief, the rulings, the
lens files and the plan. Their reports are filed whole in `reviewer-notes/`:

| Lens | Report | Verdict | Findings |
|---|---|---|---|
| RR, regression risk | `plan-RR-regression-risk.md` | DO NOT EXIT | 12 Important, 5 Minor |
| DV, delivery | `plan-DV-delivery.md` | DO NOT EXIT | 6 Important, 5 Minor |
| JM, the junior maintainer | `plan-JM-junior-maintainer.md` | DO NOT EXIT | 9 Important, 6 Minor |

None found a Critical issue, and every finding is fixed by editing the plan. Before ruling, the agent
checked the claims that decided a fork:
- the real engine over a fake output (`app/director_pins_test.go:84`);
- the mutant counts: 11 on `app/director.go`, 30 on `platform/lineup/director.go`;
- the ratified ledger row (`06_docs/p10-ledger.md:64`);
- the depth-1 hosted checkout (`.github/workflows/ci.yml:39`);
- the console masthead's clock (`modes/tty/masthead.go:53`);
- the docs test's pattern (`cmd/watchpost/docs_test.go:23`).

All of them hold.

## The agent's errors the round found

- **The console's M2 target (DV-1).** At D-4 the agent proposed a console beat of at most 140 KB without
  checking that the console runs no Observer code, so the medium memos could not reach it. Ruled at D-22.
- **The mutant count at D-13.** The agent's grep matched both `director.go` files: 11 mutants move, not
  41. Corrected at D-23.
- **Q2 as "tests only" (RR-2, RR-3).** RN-1's premise, an engine that cannot be faked, was not checked
  before the plan named a seam. Three tests already fake its output. Carried by A-14.

## Disposition

| Finding | Where it went |
|---|---|
| RR-1, DV-6 (the console's clock) | D-21 |
| DV-1 (console M2) | D-22 |
| JM-5, DV-5's ledger row, RR-10's count (the rename's scope) | D-23 |
| JM-9 (M5's criteria) | D-24 |
| DV-2 (batch map), DV-9 (the HUM LEAD's steps) | A-12; plan "Batch map", "Before BUILD" |
| DV-3, DV-4, JM-1, JM-2, JM-11, RR-17's Q7 parts (rule order and definitions) | A-13; plan Q7.6, Q7.7 |
| RR-2 to RR-4, RR-7 to RR-12, RR-14 to RR-16, DV-5, DV-10, JM-8, JM-14 (the nets and pins) | A-14; plan Q1, Q2, Q3, Q5.3, Q9.7 |
| RR-6, RR-17's Q4 parts, JM-7, DV-11 (memo guards) | A-15; plan Q4 |
| RR-5, JM-4 (D-14's shape) | A-16; plan Q6.4 |
| DV-7, DV-8, JM-3, JM-6, JM-10, JM-13, JM-15, RR-11, RR-13 (the rest) | A-17; plan step 6, Q5, Q7, Q8, Q9 |

The amended plan is `../04-development/implementation-plan.md`.
