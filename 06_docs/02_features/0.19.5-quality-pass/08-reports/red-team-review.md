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

# Round 2 (D-41)

One fresh blind reviewer on batch 50's diff, `ef9b9c27..55b10c57`, gates, tooling and measurement.
Report: `reviewer-notes/review-round-2-GT-batch-50.md`. **Verdict: EXIT**, no Critical or Important,
10 Minor. It reproduced the pins, the v0.19.3 figures (radio aside, below) and the interrupt handling
on macOS.

| Finding | Disposition |
|---|---|
| 1, the repeat reads a catch seen after another test as INVALID | fixed, A-29: the repeat is the same run; both shapes have a probe, each seen to fail on the other design |
| 2, a signal to the script alone does not stop `make` | fixed, A-29: the recipe stops on an exit of 128 or more |
| 3, `go-build` directories left after an interrupt | fixed, A-29: `GOTMPDIR` inside the run's directory |
| 4, a TERM during the baseline waits for the baseline | fixed, A-29: the baseline runs in the background; a TERM to the script alone now takes 1.9 s |
| 5, `missEveryMemo` leaves the cursor memo | fixed, A-29: re-pinned |
| 6, 7, the gate's and `mutant-check`'s times | fixed, A-29: stated as their measured ranges |
| 8, v0.19.3's radio beat | recorded: the agent measured 169,733 B, the reviewer 168,744 B; the port reads the wall clock, so the radio player's time string varies. The CHANGELOG's "about 170 KB" stands |
| 9, the promote test's cover | fixed, A-29: a commit-less summary, and the last real sweep's log as the script wrote it |
| 10, the repeat costs a step-1 run | kept, A-29: a lone test can lose a race under load; the SHIP sweep measures the cost |

**Found while checking the fixes:** the sweep's baseline read RED in batch 50's own interrupt checks.
`platform/sched`'s `TestRetriesAreBounded` advanced the fake clock as soon as a fetch was counted,
before the scheduler armed its retry wait, and under the baseline's load the advance was lost. With a
50 ms pause put before the retry wait, it fails every time; waiting for the wait before each advance,
it passes. Fixed in batch 51 (A-29). Of the interrupt checks, a TERM to the script alone during its
baseline passed (1.9 s; `make` stopped at 143, nothing promoted or left behind); the check among
the workers runs on batch 51's commit, whose baseline the fix makes reliable.
