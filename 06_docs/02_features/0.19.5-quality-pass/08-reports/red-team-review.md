---
title: "0.19.5 REVIEW red team"
date: 2026-11-09
phase: REVIEW
sev: SEV-0
status: "Round 1: every finding fixed in batch 50 or deferred by A-28, for the HUM LEAD's veto"
---

# 0.19.5 REVIEW red team, round 1

The REVIEW red team was four blind reviewers over the whole diff, `7ca44489..ef9b9c27` (702 files).
Each had one axis and the same brief: the release's goals, that a behaviour change needs a ruling
row, and no live app or network. Their reports are filed whole in `reviewer-notes/`:

| Axis | Report | Verdict | Findings |
|---|---|---|---|
| RG, regressions and unruled behaviour | `review-RG-regressions.md` | EXIT | none; 3 observations |
| DC, documents, comments, junior readability | `review-DC-docs-comments.md` | DO NOT EXIT | 3 Important, 16 Minor |
| PF, consolidation and performance | `review-PF-consolidation-performance.md` | DO NOT EXIT | 1 Important, 4 Minor |
| GT, gates, checks and tooling | `review-GT-gates-tooling.md` | DO NOT EXIT | 2 Important, 7 Minor |

None found a Critical issue. RG rendered about 8,500 frames at both commits and found every
difference traced to a ruling (D-5, D-18/D-32, D-28/D-30, D-29, D-36), and 400,000 random strings
measured alike by the old and new `Width`.

## What the agent checked before acting

- **PF P-1**, the beat pin re-running one message on one model: `beatCost` never fed `next` back
  (`modes/tty/beat_alloc_test.go:39-43`). Threaded, the idle beat measured 75,990 B, as the reviewer
  said; v0.19.3, measured the same way, 172,954 B.
- **PF P-3**, the publish pin: measured on v0.19.3, a prebuilt publish read only 113 KB, because
  that version's memo keys on the snapshot pointer and hit from the second iteration. The pin
  therefore publishes a new snapshot each time; v0.19.3 then measures 423,842 B and HEAD 82,322 B.
- **GT I-1**, the interrupted sweep promoted: `promote-verdicts` checked only that the log was not
  empty and named a commit (`Makefile:398-405`). After the fix, a real `make mutant-verdicts`
  interrupted during its baseline stopped with "Interrupt: 2" and left the record as it was, and the
  script interrupted with four `run.sh` in flight died by the signal and left no process, directory
  or worktree behind.
- **GT I-2**, a load flake credited as a catch: the attribution check re-runs the failing test on
  the unmutated tree only (`06_docs/mutants/run.sh:119-124`). `TestACatchMustRepeatAlone` reads
  CAUGHT without the new repeat and INVALID with it.
- **DC I-1**: D-33, D-36 and A-22 were not in the CHANGELOG.
- **DC I-2**: the last two green gates took 3,812 and 4,003 s; batch 48's hosted `mutant-check`
  took 131 minutes on Ubuntu (run 42700377465).

## Disposition

The agent's error the round found: the rewritten sweep's INT trap (batch 48) let `make` go on to
promote an interrupted log, and its comment claimed load could not turn a survivor into a catch.

| Finding | Disposition |
|---|---|
| PF P-1, P-2, P-3 | fixed, A-25 |
| GT I-1, M-1 | fixed, A-26 |
| GT I-2 | fixed, A-27 |
| PF P-4, C-1; DC I-1 to I-3, M-1 to M-11, M-13 to M-16 | fixed, A-28 |
| GT M-2, M-3, M-4, M-7 | deferred to `06_docs/tooling-backlog.md`, A-28 |
| GT M-5 | stands as built: SN-02's narrower opener is deliberate, against false positives on wrapped prose |
| GT M-6 | A-18's floor of 0.00, for P10's re-ratification |
| DC M-12, RG O-2 | A-9's screenshot retake at VALIDATE: the tabbed Settings, and the 12-hour stamp on a narrow card |
| RG O-1 | recorded: two configured voices sharing a first name in two locales would announce each other by the same name; the curated list has none, and D-7 accepts it |
| RG O-3 | recorded: below about 22 rows the Observer frame still runs past the terminal (25 rows at 60x20, from 32), outside D-28's 80x24 |
| DC, outside scope | the untracked local `CLAUDE.md` names 0.19.0 as the release in flight; the HUM LEAD's file |
