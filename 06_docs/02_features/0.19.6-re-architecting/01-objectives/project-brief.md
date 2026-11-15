---
title: "0.19.6 re-architecting — project brief"
date: 2026-11-14
phase: DISCOVER
sev: SEV-0
status: "APPROVED (D-13)"
---

# 0.19.6 — project brief

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
- **They dominate the gate.** Every mutant re-runs its package's whole suite, so the two packages
  took 57 % and 23 % of the baseline sweep's time (A-3).

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

Each metric is measured at the end of every batch and reported at PLAN, BUILD and REVIEW against
the baseline on `e57afeaf` (package map §5).

| ID | Principle | Metric | Target |
|---|---|---|---|
| M1 | 1, performance | **The allocation pins** (idle 168 allocations and 75,988 B, radio 327 and 68,475, console 377 and 122,387, publish 203 and 82,322, and every frame and window pin, PB-1) | A comparator holds each at or below its recorded 0.19.6 baseline: any rise in a count fails, and any rise in bytes over 0.1 % fails (C-8). The three race-skipped pins run under `make alloc-budget` (PB-7). No new model escape on `Update` or `View`, model sizes unchanged, and no fewer inlined calls on a moved file without a stated reason (PB-2, PB-3). The tone-latency instrument is reconciled before the deck moves, then held (PB-8). |
| M2 | 1, performance | **Gate time** | Recorded, not targeted: `make verify`'s seconds every batch, and the closing sweep's against A-3's 3,976 s. PB-6 projects about 6 minutes off a 66-minute sweep; the figure is reported, not promised. |
| M3 | 2, understandability | **The graph's figures** (JN P2-2, P2-3, P2-6), re-run on each batch's layout | Each falls from its baseline: the hub's fan-in (68 files of methods; 82 % of its package using its file), each area's reach into the model (map 72, Settings 102), and the files read and packages entered by walkthroughs W6 and W8 and a legend key. Each area's inputs are listed in its `doc.go`, and the console stays at 13 or below. The numeric target for each is fixed at PLAN from the graph projected on the planned layout and ratified at PLAN exit. |
| M4 | 2, understandability | **The blind trace** (JN P2-8, 0.19.5 D-24's method) | A REVIEW gate: four tasks fixed at PLAN, one with no walkthrough (a legend key) and one crossing a new package boundary (a map Setting), graded against a fresh hidden key by a reviewer with a junior persona. 4 of 4 pass first time; a fail is fixed before SHIP or ruled. |
| M5 | 2, understandability | **The guards against gaming M3 and M4** (JN P2-1, P2-4, P2-5, P2-7, P2-9, P2-10, P2-11) | Zero exported names whose only users outside their package are tests (P2-5). The package graph is a DAG drawn in the atlas, each edge with its reason (P2-7). Every docs-to-code check green, the file-map test deriving its packages (P2-9). Every new package has a vocabulary row and the user's or the role's word as its name (P2-10). Each package's size at the close is recorded and held by a ratchet (C-6); no file passes 1,000 lines that is under it today, and none over it grows (P2-11). The front doors of `modes/tty` and `app` grow only by the new packages' entry points (P2-4). |
| M6 | 3, best practice and P10 | **The gates that hold the house rules** | `make p10` green at the end of every batch (A-7), with D-10's re-key table ratified at PLAN exit and the new packages' floors at BUILD exit. The import rules gated (batch 0 item 9). `lint-authoring` green. No surviving alias (C-4); `internal/` for every package nothing outside its parent imports (C-5); no new lock (C-7). `govulncheck` green at the floor and at the toolchain (D-12). |
| M7 | 4, no regression | **The regression net** | `make verify` green at every batch, MAKE_EXIT and "ALL GATES GREEN" read, and hosted full CI green on both systems before SHIP. Every golden and both byte-identical traces unchanged (batch 0 item 6). The same test inventory with and without `-tags property` and `-race`, and the union declaration set conserved (item 5). No pin's assertions change; a straddling test changes its setup only (C-3, D-9). Each batch's mutants re-vetted by a scoped sweep compared mutant by mutant with A-3's record (C-12). The closing full sweep matches A-3's: 1,051 caught, the 7 equivalents surviving or re-ruled, none with no evidence. |

## Requirements

- **R1** No behaviour changes. A change that alters output, sound or timing is a regression in
  0.19.6, not a ruling candidate; the audible changes D-2 would make are 0.19.7's (D-11).
- **R2** Batch 0 lands before any code moves (package map §4), each instrument seen failing before it
  passes.
- **R3** Each code batch is two commits (RX-13): prep in place, then move only (`git mv`, package
  clauses and imports, mutants re-pointed by path, no comment edits). A move commit first shows the
  instruments failing on a deliberately partial move (C-11).
- **R4** Charm and Bubble Tea are used through their interfaces as they are (D-3); the Router stays
  the one `tea.Model`, and the post-update steps stay at the root in today's order.
- **R5** The shared deck is not split (K-3): it moves into `app/internal/deck` whole.
- **R6** Every batch redraws its diagrams and regenerates the atlas in the same commit, and moves its
  doc rows with its code.
- **R7** The live app is not run; performance is measured from the pins and benchmarks.

## Scope

**In:** `modes/tty` split into the root, `modes/tty/console`, a map package (2a and 2b, D-9) and a
kit; `app` split into its root, `app/internal/deck`, `app/internal/maps` and `app/internal/kit`
(package map §2). Batch 0's twelve instruments, A-2's sweep fix among them. The floor scan and go
1.26.11 (D-12). RM's slices 1 to 4: F-158's owed test, the voice cluster moved whole, the vocabulary
corrected, RM-1's table recorded (C-10). The docs, diagrams, gate tools and ledgers that name what
moves, and the doc-against-code disagreements and misleading names found by RC-7, RM-8 and JN-5, each
fixed in the batch that moves its code.

**Out:** D-2 proper and everything A-8 lists, which are 0.19.7 (D-11). Re-implementing any part of
Charm (D-3). A Charm or Bubble Tea upgrade (C-9). New features. Splitting any other package.
go-tuiMaps and go-ionomaps.
