---
title: "0.19.6 plan red team"
date: 2026-11-15
phase: PLAN
sev: SEV-0
status: "Round 1: every finding dispositioned (D-7); six forks ruled D-27 to D-32; the agent's fixes A-29 to A-33, for veto"
---

# 0.19.6 plan red team, round 1

The plan red team was five blind, read-only reviewers, each sent `06_docs/red-team-brief.md`'s
template with the PLAN record at `fbbfeb1a` as its scope and the code at `e57afeaf`. No reviewer ran a
test, because the A-7 pilot was timing on the same machine. Their reports are filed whole in
`reviewer-notes/plan-*.md`; the pilot's is `pilot-report.md`.

| Axis | Report | Verdict | Critical | Important | Minor |
|---|---|---|---|---|---|
| CQ, code quality | `plan-CQ-code-quality.md` | NOT READY | 0 | 13 | 8 |
| RR, regression and performance risk | `plan-RR-regression-performance.md` | NOT READY | 1 | 11 | 2 |
| PH, project hygiene | `plan-PH-project-hygiene.md` | NOT READY | 0 | 8 | 9 |
| DQ, docs quality | `plan-DQ-docs-quality.md` | NOT READY | 0 | 6 | 7 |
| BQ, business quality | `plan-BQ-business-quality.md` | NOT READY | 0 | 4 | 7 |

**What they agree on.** The decomposition is sound and BUILD can execute it batch by batch; every
code citation the reviewers checked holds; nothing needs a new design. **What they found:** batch 0
had items that could not pass as written, several proofs had no fail condition, the M3 table up for
ratification left out the brief's figures, and the designs' scripts were not in the tree.

## The claims the agent checked before ruling

Each of these decides a fork or a disposition, and each holds on the code:
- the router trace builds no map: `benchDash` is `NewDashboard(Config{Version: "bench"})`
  (`modes/tty/bench_test.go:89`, used at `modes/tty/router_trace_test.go:50`);
- `modalWidth()` lays out the open window, `setupWidth()` for Settings (`modes/tty/view.go:154-170`);
- `app/testdata/declset.txt` is pinned by `TestDeclarationSetUnchanged` (`app/declset_test.go:28-32`);
- `readRecord` calls `t.Fatal` on a missing record file (`cmd/watchpost/record_test.go:133-141`);
- `modes/tty/map_prop.go:126` batches its save last;
- `modes/tty/map_floor_test.go:148` sets `d.cfg.Timed` after construction;
- M3's baselines reproduce from the DISCOVER graph: the map's reach 72 and Settings' 102 (JN's scope).

## Agreements and disagreements between reviewers (D-7)

The reviewers did not contradict one another; many found the same hole from different axes. Where
they overlapped, one disposition covers them all:

| Finding | Reviewers | Disposition |
|---|---|---|
| M3's table left out fan-in, Settings, W6, W8 | BQ-1, DQ-12 | **Fork 1, ruled D-27**: projected and added; W6 for a map Setting rises 3 to 4 packages, accepted with D-25 as its reason |
| benchstat has no fail condition; pins fail only on rises; sizes have no direction | CQ-12, RR-5, RR-12, PH F-13, DQ-9 | **Fork 2, ruled D-28** |
| M4's pass would measure B1's doc fixes, not the split | BQ-8, DQ-13 | **Fork 3, ruled D-29**: a midpoint trace after B1 |
| New instruments have no gate home; the scoped sweep has no command; the nested graph module is outside every gate | PH F-10, F-11, CQ-18 | **Fork 4, ruled D-30** |
| B0.4 needs helpers B2 and B5 create | PH F-8, DQ-4, RR-14 | A-29: B0 creates the two helpers alone |
| The golden gate fails on B0's own goldens; the `app` declset must move | PH F-9, CQ-13, RR-8 | A-29: the gate's baseline is B0's close, with exact lists |
| The designs' scripts are not in the tree; M3 rests on a tool nobody can run | PH F-3, F-4, DQ-11 | A-29: B0.12's tool reproduces every figure and replaces the scripts |
| The record tests break when pointed at 0.19.6 | PH F-7 | A-29: B0.13's full scope |
| The router trace is blind to the map | RR-1 (Critical) | A-29: it builds a real map and runs its commands |
| `Env`'s derived sizes run a window's layout every message | CQ-7, RR-11 | A-31: deleted from `Env`; two new pins |
| The root's re-station and attach calls lose their pins | BQ-5, RR-3, CQ-17 | A-32: root tests through the deck's held surface; A-23's "unpinned wiring" withdrawn |
| The non-drawn test references and 2b's residual arrive mid-batch | CQ-1, BQ-9 | A-33: singly to the HUM LEAD before B2 and before B5 |
| M4's owed guards are in no batch | PH F-14, CQ-5 | A-33: each given its batch |
| B4 promises no assertion changes | DQ-3 | A-31: assertions on methods gaining a parameter change under D-14 |
| The re-vet list misses in-place batches; a mis-anchored mutant passes; test identity is unruled | RR-2, RR-13, CQ-14 | A-30 |
| The Router rows' proof is wrong for `Dashboard` | DQ-5 | A-33: re-proved per surface |

The pilot added two forks: **fork 5, ruled D-31** (density: D-10's prefix premise is false) and **fork
6, ruled D-32** (same-name calls under P10-01).

## Not taken, with the reason

- **CQ-9: unexport `ChipKnown`; `Env.Ticks` exports a test seam.** D-26 keeps `ChipKnown` as a
  recorded contract check, and D-21 carries the Dashboard's tick hook in `Env`. Both are the HUM
  LEAD's rulings; the finding is recorded here as a recommendation against them, not acted on.
- **CQ-10's `run` and `frame` names.** `run` stays (A-20), its shadowing local renamed (A-32);
  `frame`'s definition is widened to "what more than one package draws" (A-33) rather than a fourth
  small package.
- **DQ's audience order (§4).** Taken in part: the plan gains an "In short" with what the gate
  ratifies; the tables stay where the work reads them.
- **DQ-7, `mutants-AP.tsv`.** Not regenerated: it places mutants by file, AP-10's statement-level
  counts govern, and the design's header now says so.
- **PH F-1.** Pushes go by URL by design; each push's remote ref is now recorded in the build log.

## What is owed to BUILD, so it cannot be lost

- The nine non-drawn test references, singly, before B2 (A-33).
- 2b's residual after B4's re-measure, singly, before B5 (A-33).
- Each new package's density row and each `New` row, in its batch (D-31, D-32).
- M4's midpoint trace after B1 (D-29).
- F-237 (a lock of their own) and F-238 (the `1f9a34e7` purge), filed.
