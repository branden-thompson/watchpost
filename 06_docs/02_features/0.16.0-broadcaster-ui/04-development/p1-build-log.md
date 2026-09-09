---
title: "0.16.0 — P1 build log: the console shell"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "P1 COMPLETE.  Five commits.  Whole tree green under -race, alloc-budget green, pty-severe green."
---

# P1 — the console shell

**Commits:** the second surface and the fan-out · the three lanes · the size floor · the `--ascii`
render · the frame budget.

## What it delivered

| Requirement | Status |
|---|---|
| **FR-1.2** the fan-out | **DONE**, and it is what P0's deferral was waiting for |
| **FR-2.1-2.4** the three lanes, read-only from `Publish` | **DONE** |
| **FR-2.6** the text clamp reused | **DONE**, wired not asserted |
| **FR-7.1** the layout selected by the platform breakpoint | **DONE** — and the rewritten exit provably rejects what the original accepted |
| **FR-7.2-7.4** the floor, the notice, no overflow | **DONE**, floor 44 lines / 80 columns |
| **NFR-3** the frame budget | **DONE** — it is a number now |
| **RT-23** the `--ascii` render | **CLOSED** |
| **F-67** the five orphaned messages | **structural half CLOSED**; handling still open |

## The three things worth carrying forward

**1. The deferral from P0 was right, and it proved itself.**  With one surface a fan-out and a
delegation are indistinguishable.  With two, a plant that removed `WindowSizeMsg` from the
program-scoped set was **caught** — which proves the gate watches the SET, not just the plumbing.
That test could not have existed a batch earlier.

**2. My gate caught F-55's shape before any fix existed.**  `19x5 rendered 22 cells wide` on its first
run.  The defect the requirement was written for, reproduced by its own instrument.

**3. FR-7.1's rewritten exit was proven, not just adopted.**  The review rewrote it because the
original — *"the platform breakpoint API has production callers"* — is satisfiable by a discarded
call.  Planting exactly that discarded call **fired the new exit**.  A rewritten requirement that
rejects what the old one accepted is a rewrite that earned itself.

## Where I went wrong, four times

| | What happened |
|---|---|
| **Two invalid plants in one batch** | Deleting the clamp call and deleting the slot index left their identifiers unused, so the tree would not COMPILE.  A build failure is not evidence either way — the documented removed-a-use shape, walked into twice |
| **A bad plant in P0** | Planted mixed-case text into a frame that renders capitals.  It never reached its subject |
| **A non-escaping plant** | Nearly repeated 0.15.0's own recorded mistake: a discarded allocation is deleted by the compiler before a gate can count it |
| **An import edit that missed** | A plant that failed to build, twice, before it fired |

**The pattern in all four: the instrument was fine and the PLANT was wrong.**  That is exactly what
INST-3 says to suspect first, and it earned its place again.

## The domain taught the fixture

The lineup refused three malformed cards before accepting one — *"every card carries a headline from
the moment it is proposed"*, then *"the lineup holds admitted cards only"*.  **The fixture now follows
the real lifecycle rather than side-stepping it.**  A test that seeded a fake card would have proven
nothing about a real one.

## What P1 does NOT claim

**Byte-exact fidelity to the mock.**  The lanes use the mock's vocabulary and the schedule's data; the
mock's 150-column budget, its side-by-side panels and its scrollbar are later work, and layout is the
HUM LEAD's pass.  The code says so where a reader will find it.

## Numbers

| | |
|---|---|
| Router overhead on Observer's frame | **1 allocation** |
| Console frame, 150x74, two cards | **13 allocations** |
| Size floor | **80 x 44** — 44 measured off the mock at wave 1 |
| Gates on the roster after P1 | **16**, each with a watched failure |

## Next

**P2 — station state.**  `Power` gets its first consumer, the Variant C banner lands with its
background treatment, the "on the air" boundary is stated where the operator reads it, and the swap
gate stops failing closed.
