---
title: "0.19.5 quality and performance pass — build log"
date: 2026-11-06
phase: BUILD
sev: SEV-0
status: "LIVE — what each batch built, and why"
---

# 0.19.5 build log

## Before BUILD: the prototypes D-22 names

On top of the PF lens's prototypes of the small fixes (PF-1's first step, PF-2 to PF-5), four pieces were
prototyped in a scratch copy of the tree. None of this code is in the tree; the prototypes are a
measurement and are kept, with the lens's, in the evidence folder outside it:

- PF-7: a one-entry masthead memo, one per surface;
- the console's `Projection`, cached per line-up generation;
- PF-8: one tape memo, keyed on a process-wide ticker generation bumped by `setTicker`;
- PF-1's second step, on the hot path only: the tick and visualiser messages pass through the Router and
  the Observer as concrete values, boxed once, on the way out of `Router.Update`.

Bytes per beat (`BenchmarkPF_Router{Tick,Viz,ConsoleTick}Beat`, three runs each, figures stable to the
byte):

| Beat | 0.19.3 | the small fixes | + masthead memo | + `Projection` cache | + tape memo | + PF-1 second step | D-4's target |
|---|---|---|---|---|---|---|---|
| Idle Observer, 133x44 | 156.5 KB | 104.9 | 96.8 | 96.8 | 83.8 | **62.0** | 80 |
| Radio playing, visualiser on | 169.7 KB | 112.5 | 104.4 | 104.4 | 97.9 | **76.1** | 90 |
| Console tick, 160x74 | 310.5 KB | 161.3 | 152.4 | 137.8 | 137.8 | **129.6** | 140 |

**All three of D-4's targets are met on the prototypes**, with 18, 14 and 10 KB to spare. PF-9 and PF-10
are not needed to meet them; they still land (Q4.1, Q4.4), because they cut the cost of a publish, which
the beat does not measure.

**One test failed on the prototypes, and it is the case the plan names.**
`TestTickAdvanceAllocBudgetWithATestEventOnTheTape` writes `d.ticker[0].Test = true` in place, and the
tape memo, keyed on a generation that only `setTicker` bumps, went on drawing the unmarked tape. That is
the stale-frame risk A-15 guards against. Q4.3 makes the ticker change only through `setTicker`, and its
guard perturbs the inputs. Every other test in `modes/tty` passed on the prototype copy.
