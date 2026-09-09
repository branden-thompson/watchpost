---
title: "0.16.0 — the gate roster: every gate with a failure that was WATCHED"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "OPEN — grows per batch.  P0 entered."
---

# Gate roster

**Every gate this release adds carries an evidence line naming a failure that was actually WATCHED**
(FR-11.1).  A gate nobody has seen fail is a gate nobody has measured.

**"Watched" means one of two things, and the table says which.**  A **plant** is a defect introduced
deliberately, the gate run, and seen to fire — dated, and named so it can be re-run.  **By
construction** means the gate's failure path runs on every invocation against known-bad input.

**A plant that SURVIVED is recorded too**, and those rows are the most useful in the table: telling a
real hole from a bad plant is the skill the roster exists to teach.

## P0 — the router

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestRouterRendersObserverByteForByte` | The Router renders Observer unchanged — P0's whole claim | **Plant 2026-09-09, and the FIRST ONE WAS BAD.**  v1 replaced `"Watchpost"` in the frame → **SURVIVED**.  Per INST-3 that indicts the plant, and rightly: the frame renders `WATCHPOST` in capitals, so the plant never reached its subject.  v2, one character of `WATCHPOST` → **CAUGHT at byte 22, with the frame still 6146 bytes** — which is the distinction that matters: it catches a SUBTLY wrong frame, not merely an empty one |
| `TestRouterStartsOnObserver` | A station comes up on Observer | **Plant 2026-09-09:** the constructor set `SurfaceBroadcaster` → **CAUGHT** (`active = 1, want 0`).  Recorded because it passed on first write and was therefore UNPROVEN until planted |
| `TestRouterDelegatesInitToTheActiveSurface` | Observer's background-colour request is not dropped | **By the watched RED:** the stub returned `nil` and the test failed with its own message before the implementation existed |
| `TestRouterPassesAKeyToTheActiveSurface` | A keypress reaches the active surface | **By the watched RED**, same stub |
| `TestTheProgramsModelIsAlwaysTheRouter` | Every `tea.NewProgram` in `app/` is handed a Router — **DERIVED by AST walk, not a remembered list** | **Two plants 2026-09-09, both directions.**  (a) one site handed a bare model → **CAUGHT**, naming file and line.  (b) **the walk's own matcher broken** → **CAUGHT as "the check did not run, which is not the same as passing"** — INST-2 proven by plant rather than asserted.  States its blind spot in its own output (INST-5) |
| `make pty-severe` | The real binary still drives through a PTY with the Router as its model | **By construction, every run**: eight keystroke assertions against the shipped binary.  Green after the wiring |
| `TestDeclarationSetUnchanged` | A batch does not change the declaration set unnoticed | **By construction.**  It fired on P0's additions and was re-captured deliberately — 11 added, **0 removed** |

## P1(a) — the second surface and the fan-out

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestSizeReachesTheINACTIVESurface` | A resize reaches BOTH surfaces, not just the active one | **Two plants 2026-09-09, both CAUGHT.**  (a) the fan-out's result discarded → `the inactive one has 0x0, want 171x61`.  (b) **`WindowSizeMsg` removed from the program-scoped set** → same catch.  The second is the one that matters: it proves the gate watches the SET, not just the plumbing |
| `TestObserverStillGetsItsOwnSizeToo` | Fanning out does not STEAL the message from the active surface | **Plant 2026-09-09:** the active surface's update dropped → **CAUGHT** (`observer width = 133, want 171` — it kept its old size) |
| `TestBackgroundColourReachesTheINACTIVESurface` | The terminal's ground reaches both | By the watched RED: the stub did not fan and the test failed with its own message |

**Why these could not exist before P1.**  With one surface, a fan-out and a delegation are the same
operation and no test can tell them apart.  FR-1.2 waited for the surface that makes it observable —
which is why it was deferred rather than skipped.
