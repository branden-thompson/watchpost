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

## P1(b) — the three lanes

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestTheConsoleShowsTheMainTrackItWasPublished` | The console shows the cards the schedule PUBLISHED | **By the watched RED**: the stub stored the lineup and rendered `BROADCASTER`, and the test named the missing headline |
| `TestTheConsoleShowsNothingItWasNotPublished` | It invents nothing | Passes on an empty lineup; the negative direction of the row above |
| `TestTheConsoleNumbersTheMainTrackSlots` | The ten slots are addressable `0`-`9` (FR-2.4) | **Plant 2026-09-09, and THE FIRST ATTEMPT WAS INVALID.**  v1 deleted `strconv.Itoa(i)` and left `i` and the import unused → **BUILD FAILURE, which is not evidence either way** (the removed-a-use shape).  v2 computed the index, discarded it, and returned a constant handle → **CAUGHT** |
| `TestTheConsoleShowsAtMostTenMainTrackCards` | A rolling view of ten; an eleventh does not reach the frame (FR-3.1) | **Plant 2026-09-09:** the bound removed → **CAUGHT** (`an eleventh card reached the frame`) |
| `TestAHostileHeadlineIsClampedInTheNewLanes` | **FR-2.6** — external text in the NEW lanes goes through the EXISTING clamp, not a second path | **Plant 2026-09-09, v1 ALSO INVALID** — removing the clamp call left its import unused → build failure.  v2 CALLED the clamp and discarded its result → **CAUGHT** (`escapes intact`) |

**Two invalid plants in one batch, same shape, and it is a documented one.**  A mutation that removes
a USE leaves the tree uncompilable and reports nothing.  Both were re-planted as the rule says — the
call still runs, its result is discarded — and both then fired.

## P1(c) — the size floor and the breakpoints

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestNoSizeRendersPastTheTerminal` | **Nothing ever renders past the terminal**, in either direction | **Caught the F-55 shape on its FIRST RUN, before any fix existed** — `19x5 rendered 22 cells wide`.  **Plant 2026-09-09:** the clamp computed and its result discarded → **CAUGHT** (`24 cells wide`).  Sweeps widths **DERIVED from the breakpoint boundaries** (INST-1) **unioned with a stride of 5**, because a boundary-only sweep verifies classification and not interior rendering — the PLAN red team's own counter-argument |
| `TestBelowTheFloorTheConsoleSaysSoRatherThanOverflowing` | Below the floor the operator gets a NOTICE, and **the notice itself fits** | **Plant 2026-09-09:** the notice computed and never shown → **CAUGHT**.  The fit assertion exists because a notice that overflows lets a sweep report zero overflows — M5's anti-solution, closed |
| `TestTheFloorIsFortyFourLines` | The height floor is the MEASURED 44, not a chosen number | **Plant 2026-09-09:** floor changed to 30 → **CAUGHT** (`got 30`) |
| `TestTheLayoutIsSelectedByThePlatformBreakpoint` | **FR-7.1** — the layout is SELECTED by the platform classifier | **Plant 2026-09-09 — THE REQUIREMENT'S OWN ANTI-SOLUTION.**  The original exit read *"the platform breakpoint API has production callers"*, which the red team showed was satisfiable by `_ = term.BreakpointFor(w)`.  Planted exactly that — classify, discard, one layout → **CAUGHT** (`two different breakpoint classes rendered an identical frame`).  The rewritten exit is proven to reject what the old one accepted.  The test also **voids itself** if both fixture widths classify the same, so it cannot pass by asserting nothing |

**F-68 moves.**  `term.Breakpoint` had **zero** production callers at DISCOVER — it was the dead half of
the two-vocabulary duplication.  It now has two.  Whether Observer's own scale migrates is **still not
ruled** (D-13 did not decide it) and P1 does not assume it does.

## P1(d) — the `--ascii` render

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestTheConsoleCarriesNoNonASCIIUnderASCII` | Under `--ascii` the frame carries no glyph that has an ASCII form, **and no non-ASCII rune at all** | **By the watched RED**: the field existed and nothing consulted it, and the test named the offending glyph — `the frame carries "•"`.  **Plant 2026-09-09:** the glyph set consulted and a literal used anyway → **CAUGHT**.  **The subject list is DERIVED** (INST-1): it asks the glyph set what the rich forms ARE rather than naming a few, so a glyph added later is covered without editing the test |
| `TestTheConsoleInheritsTheASCIIDecision` | One setting, one carrier — the console inherits Observer's `--ascii` | **Plant 2026-09-09:** the config read and not carried → **CAUGHT** (`one setting must not have two carriers`) |

**RT-23 is closed.**  The DISCOVER red team flagged that no `--ascii` rendering of the design existed,
so it was *"not shown to survive that mode"*.  It is now shown, by a derived assertion rather than a
screenshot.
