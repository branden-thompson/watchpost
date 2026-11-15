---
title: "0.19.6 re-architecting — project brief"
date: 2026-11-14
phase: DISCOVER
sev: SEV-0
status: "APPROVED (D-13), amended (D-18)"
---

# 0.19.6 — project brief

## In short

0.19.6 splits `modes/tty` and `app` into smaller packages along their seams, and users notice
nothing: no output, sound or timing changes (R1). The new packages' names, M3's targets, P10's
re-key table, M4's four tasks and the CLAUDE.md question (D-5) are ruled at PLAN. D-2 proper, and
everything A-8 lists, is out: it is 0.19.7.

**What it costs.** Batch 0, then eight code batches (package map §6): RM's slices, the `modes/tty`
kit, the console, the map's 2a, the `app` kit, the deck, the maps, and the map's 2b. Each code batch
is two commits, and 2b three (R3), so about 18 full gates. At the 2,408-second median of 113 green
`make verify` runs (`06_docs/verify-runs.md`), that is about 12 hours of gate. On top come batch 0's
instruments, the scoped sweeps, the partial moves (C-11), the PLAN pilot (A-7), the 66-minute
closing sweep and hosted CI. 0.19.7, with its audible changes (D-2, F-73's pause, MVS-D-74),
waits on this release.

**What it leaves.** The root `modes/tty` stays the largest package, at about 16,900 lines, still
about 2.8 times `platform/render`. `Dashboard` keeps 374 methods in 38 files, down from 752 in 68 (A-27).
The first problem below stays true for the root, but its areas are bounded by the compiler.

## The problem

Two packages hold half of watchpost's application code, and everything about them is harder than it
needs to be (`../02-analysis/package-map.md` §1):

- **They are too big to hold in one head.** `modes/tty` is 31,346 lines in 88 files and `app`
  20,184 in 76; the next package, `platform/render`, is 5,989.
- **One type is the whole program's hub.** `Dashboard` carries 752 methods in 68 files, and
  `dashboard.go` is used by 71 of `modes/tty`'s 87 other files (JN P2-2). `livePipelines` carries
  108 methods in 21 files (AS-2). Nothing but convention says which area may touch which field: the
  map reads 72 of the model's symbols and Settings 102 (JN P2-3).
- **The areas are tangled.** Inside `modes/tty` the map and Settings, the map and the Dashboard, and
  the map and Propagation call each other both ways (JN P2-7).
- **They dominate the gate.** The two packages took 57 % and 23 % of the baseline sweep's
  14,101 mutant-seconds, summed over 4 workers (A-3, A-9). The sweep runs each mutant's last killer
  first (`06_docs/mutants/run.sh:93`), so the split saves only about 9 % of the sweep and about 10 %
  of `race`, and `make verify` barely moves. The gate is not the release's reason.

**Problem statement (D-1):** `modes/tty` and `app` have grown into two packages whose shape no
longer says which code owns what, so a maintainer cannot tell what a change touches and the compiler
cannot hold any boundary between their areas. Splitting them is only safe if the split changes no
behaviour and costs no performance, and both can be proved batch by batch.

## Principles (the HUM LEAD's, 2026-11-14, D-1)

1. Performance must stay the same or improve.
2. Maintainability, traceability and understandability of the architecture must improve.
3. Best practices are maintained, P10's rules included.
4. No behaviour or functionality regresses.

## Metrics and targets

Each metric except M4 is measured at the end of every batch and reported at PLAN, BUILD and REVIEW
against the baseline on `e57afeaf` (package map §5). M4 is measured twice: before, at PLAN, and
after, at REVIEW.

| ID | Principle | Metric | Target |
|---|---|---|---|
| M1 | 1, performance | **The allocation pins** (idle 168 allocations and 75,988 B, radio 327 and 68,475, console 377 and 122,387, publish 203 and 82,322, and every frame and window pin, PB-1) | A comparator holds each at or below its baseline, the maximum over 10 runs on `e57afeaf`, recorded with the count of runs: any rise in a count fails, any rise in bytes over 0.1 % fails, and a baseline pin that produces no measurement fails (C-8, A-10). The map, severe and Propagation windows flicker by one allocation at baseline, and their byte figures are held beside the count. The three race-skipped pins run under `make alloc-budget` (PB-7). The tone-latency benchmark's 185 allocations are held as a pin; its wall-time reconciliation is 0.19.7's (A-17). No new escape, gated by function and type; model sizes unchanged; inlined calls counted per function, and any fewer needs a HUM LEAD ruling (PB-2, PB-3). **Wall time:** every benchmark in the two packages compared by benchstat, paired back to back at `-count 10`, each code batch, and mandatory at 2a and 2b. Startup (`GODEBUG=inittrace=1`) and binary size are recorded at open and close. |
| M2 | 1, performance | **Gate time** | Recorded, not targeted: `make verify`'s seconds every batch, and the closing sweep's against A-3's 3,976 s. PB-6 projects about 6 minutes off a 66-minute sweep; the figure is reported, not promised. |
| M3 | 2, maintainability: a change's reach is bounded and visible | **The graph's figures** (JN P2-2, P2-3, P2-6), re-run on each batch's layout by the committed graph tool (A-12) | Each falls from its baseline: the hub's fan-in, measured by type, not by a file ratio; each area's reach into the model by type (today the map reads 72 of the model's symbols and Settings 102), counted across packages the same way as within one, the fields reached through `Env` and the hooks included; and the files read and packages entered by walkthroughs W6 and W8 and a legend key. Each area's inputs are listed in its `doc.go`, and the console stays at 13 or below. The numeric target for each is fixed at PLAN from the graph projected on the planned layout and ratified at PLAN exit. |
| M4 | 2, traceability: a reader can follow a behaviour to its code | **The blind trace** (JN P2-8, 0.19.5 D-24's method) | A before: the same four tasks, blind, on `e57afeaf` at PLAN, graded against JN's leave-one-out key. A REVIEW gate: four tasks fixed at PLAN, one with no walkthrough (a legend key) and one crossing a new package boundary (a map Setting), graded against a fresh hidden key by a reviewer with a junior persona. 4 of 4 pass first time; a fail is fixed before SHIP or ruled. |
| M5 | 2, understandability: names, docs and APIs say what the code is | **The guards against gaming M3 and M4** (JN P2-1, P2-4, P2-5, P2-7, P2-9, P2-10, P2-11) | Zero exported names added whose only users outside their package are tests (P2-5); the three that exist today, `ChipKnown`, `SameOverlay` and `FireAll`, are held by a test that lists exactly them (D-26). The package graph is a DAG drawn in the atlas, each edge with its reason (P2-7). Every docs-to-code check green, the file-map test deriving its packages (P2-9). Every new package has a vocabulary row and the user's or the role's word as its name (P2-10). Each package's size at the close is recorded at REFLECT, and every later REFLECT reports against it (D-17); no file passes 1,000 lines that is under it today, and none over it grows (P2-11). The front doors of `modes/tty` and `app` grow only by the new packages' entry points (P2-4). |
| M6 | 3, best practice and P10 | **The gates that hold the house rules** | `make p10`, a local gate outside `required-gates.txt`, green at the end of every batch, with its summary in the batch's build log (A-7), with D-10's re-key table ratified at PLAN exit and the new packages' floors at BUILD exit. The import rules gated (batch 0 item 9). `lint-authoring` green. No surviving alias (C-4); `internal/` for every package nothing outside its parent imports (C-5); no new lock (D-16). `govulncheck` green at the floor and at the toolchain (D-12). |
| M7 | 4, no regression | **The regression net** | `make verify` green at every batch, MAKE_EXIT and "ALL GATES GREEN" read, and hosted full CI green on both systems before SHIP. Every golden and both byte-identical traces unchanged (batch 0 item 6). The same test inventory with and without `-tags property` and `-race`, and the union declaration set conserved (item 5). No pin's assertions change except to read the same value through production API (D-14): the same killers in the scoped sweep, an outcome-equal inventory, and each batch's table of changed test lines. Each batch's mutants re-vetted by a scoped sweep, its list derived (A-11), compared mutant by mutant with A-3's record read from `7c5e5772` (C-12, A-9). The closing full sweep matches A-3's: every A-3 mutant keeps its class or is re-ruled, every new mutant is caught, and none has no evidence. |

## Requirements

- **R1** No behaviour changes. A change that alters output, sound or timing is a regression in
  0.19.6, not a ruling candidate; the audible changes D-2 would make are 0.19.7's (D-11).
- **R2** Batch 0 lands before any code moves (package map §4), each instrument seen failing before it
  passes.
- **R3** Each code batch is two commits (RX-13), and 2b three (A-13): prep in place, then move
  only. A move commit may change only `git mv`, package clauses, imports and package qualifiers;
  the `doc.go` file maps; P10 ledger and baseline re-keys; mutants re-pointed by path; the paths that
  documents and comments name; and the atlas. The body-hash union check proves nothing else changed
  (A-12). A move commit first shows the instruments failing on a deliberately partial move (C-11).
- **R4** Charm and Bubble Tea are used through their interfaces as they are (D-3). The program's
  model is always the Router (`TestTheProgramsModelIsAlwaysTheRouter`, `app/router_owner_test.go:17`),
  and the post-update steps stay at the root in today's order, held by a test (A-12).
- **R5** The shared deck is not split (K-3): it moves into `app/internal/deck` whole.
- **R6** Every batch redraws its diagrams and regenerates the atlas in the same commit, and moves its
  doc rows with its code.
- **R7** The live app is not run; performance is measured from the pins and benchmarks.
- **R8** No new `tea.Msg` or command where the code calls directly today, held by a pinned set of
  message types across the tree (A-12).
- **R9** The seam adds no func-field read on the frame path, and the map's hooks are held once. The
  map's `Env` carries no func field but the Dashboard's own clock and tick hook; the one addition is
  `console.Opening.Now`, the Dashboard's clock handed to the console by `NewRouter` (D-21).

## Scope

**In:** `modes/tty` split into the root, `modes/tty/console`, `modes/tty/mapview` (2a and 2b,
D-9), the kit `modes/tty/internal/frame`, `modes/tty/internal/controls` (D-25) and the test support
`modes/tty/internal/ttytest` (D-15); `app` split into its root, `app/internal/deck`,
`app/internal/mapdata`, `app/internal/run` and the test helpers `app/internal/apptest` (package map
§2, A-20). Batch 0's twelve instruments, A-2's sweep fix among them. The floor scan and go
1.26.11 (D-12). RM's slices 1 to 4: F-158's owed test, the voice cluster moved whole, the vocabulary
corrected, RM-1's table recorded (C-10). The docs, diagrams, gate tools and ledgers that name what
moves, and the doc-against-code disagreements and misleading names found by RC-7, RM-8 and JN-5, each
fixed in the batch that moves its code.

**Out:** D-2 proper and everything A-8 lists, which are 0.19.7 (D-11). Re-implementing any part of
Charm (D-3). A Charm or Bubble Tea upgrade (C-9). New features. Splitting any other package.
go-tuiMaps and go-ionomaps.
