---
title: "0.16.0 — the gate roster: every gate with a failure that was WATCHED"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "OPEN — grows per batch.  P0-P3 entered 2026-09-09; P4-P7 reconstructed 2026-09-13."
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

## P1(e) — NFR-3, the frame budget

**The review's objection was that NFR-3 was a PROMISE TO MEASURE, not a budget** — no threshold meant
nothing could fail it, and it had no owning batch until PLAN's remediation gave it one.  **These are
the measurements, taken and then pinned.**

| Measurement | Value |
|---|---|
| Observer's frame, direct | **218 allocations** |
| Observer's frame, through the Router | **219** |
| **The Router's overhead** | **1 allocation per frame** |
| The console's frame, 150x74, two cards | **13 allocations** |

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestRouterCostsObserverAlmostNothingPerFrame` | **NFR-1/NFR-3** — the Router does not regress Observer | **Plant 2026-09-09:** twenty escaping allocations per frame → **CAUGHT** (`through the router 239 · overhead 21 (budget 1)`).  **The first attempt did not build** — the import edit missed — which is not evidence; re-planted and it fired |
| `TestConsoleFrameAllocBudget` | The console's own frame cost | **Plant 2026-09-09:** twenty escaping allocations → **CAUGHT** (`allocates 33 per View(), budget 13`) |

**Both budgets are pinned AT the measurement, not at a comfortable margin.**  An overhead of 1 means a
second allocation fires the gate — deliberate, because the frame is drawn on every tick.

**The console's number WILL move**, and each later batch re-pins it deliberately with the reason in
the commit, the way the goldens are re-recorded.  A budget nobody re-pins is a budget nobody reads.

**The plant had to ESCAPE.**  A discarded `make` is deleted by the compiler before a gate can count
it — the bad plant recorded in 0.15.0's own roster.  These assign to a package-level sink.

## P2(a) — the station banner

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestTheBannerReadsTheDirectorsPowerNotALocalFlag` | **FR-5.1** — the state is the Director's, never a local flag | **Plant 2026-09-09:** the published state ignored and a local flag kept → **CAUGHT** (`got STOPPED`).  This is a SAFETY gate, not a display one: the swap gate depends on this value |
| `TestTheBannerIsVariantC` | **D-21** — a labelled field with the transition in parentheses | By the watched RED: the field existed, nothing rendered it, and the test named each missing part |
| `TestTheStateIsLegibleWithoutColour` | **FR-5.3** — the words carry the state, not the colour | By the watched RED under `--ascii` |
| `TestTheOnAirBoundaryIsStatedToTheOperator` | **FR-5.5** — the console says what ON AIR means | **Plant 2026-09-09:** the boundary line blanked → **CAUGHT** |
| `TestTheConsoleCarriesNoNonASCIIUnderASCII` **(strengthened)** | Now sweeps **every power state**, derived | **I PUT A NON-ASCII SEPARATOR IN THE ON AIR BANNER AND MY OWN GATE MISSED IT** — its fixture left the station STOPPED, whose line carries no separator.  The sweep now walks the power set by asking `Power.String()` where it ends.  **Plant 2026-09-09:** the literal restored → **CAUGHT**, and the message names the state: `--ascii (power=RUNNING): the frame carries "·"` |

**The lesson is the one INST-1 keeps making.**  A single fixture is a hand-written subject list of size
one.  The hole was not in the gate's logic; it was in what the gate was pointed at.

### The alloc budget fired for real, and I pushed past it

**`TestConsoleFrameAllocBudget` caught P2(a)'s banner: 14 allocations against a budget of 13.**  The
gate did exactly what it was pinned at the measurement to do.

**And I committed and pushed anyway.**  The test run and the commit were chained with `;` rather than
gated on the result, so a red gate reached `origin`.  That is the failure
*Verify-Then-Commit for Scripted Edits* names precisely — never chain edit-and-commit; verify, THEN
commit — and I had already recorded a variant of it earlier the same session when an `echo pushed`
followed a push whose exit code I had discarded.

**Re-pinned to 14 deliberately**, with the cause named (`stationLine`), which is the path the gate's
own failure message prescribes.  **Both the catch and the process failure are recorded**, because a
roster that only shows catches teaches the easy half.

## P2(b) — the swap gate, reading real power

**D-1's precondition stops failing closed and starts deciding.**

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestSwappingAwayIsRefusedWhileTheStationIsLive` | **FR-1.4** — a swap away from a RUNNING station is refused | **Plant 2026-09-09:** the refusal made unreachable (`&& false`, so the identifiers stay used) → **CAUGHT** |
| `TestSwappingAwayIsPermittedFromStandby` · `...WhenStopped` | STANDBY and STOPPED both permit it | **By the watched RED**: the fail-closed stub refused everything, and these two were the red half — which also proved the stub genuinely failed closed |
| `TestSwappingTOTheConsoleIsAlwaysPermitted` | Arriving is not the hazard the ruling bounds | Swept across **every power state, derived** from `Power.String()`'s end |
| `TestAnUndeclaredSurfaceIsRefused` | Fail closed on a corrupt value | **Plant 2026-09-09:** the default opened → **CAUGHT** |
| `TestThePowerPreconditionHasOneReaderInTheRouter` | **The rule has exactly ONE carrier** — an AST walk, derived, not a remembered file list | **Plant 2026-09-09:** a second read added in `View` → **CAUGHT** (`has 2 readers`).  Reports **"the check did not run"** rather than passing if the walk finds nothing (INST-2), and states its blind spot (INST-5) |

**Why the single-reader guard exists at all.**  The gate lives in the Router and not in the console's
key handler because if the surface decided, every future path that could request a swap would have to
re-implement it.  **Two carriers of one rule is the shape that produced the duck-lift bug**, and the
band and the config writer each already carry an AST guard for exactly this.

### Process, corrected

**This batch's commit was GATED on the results rather than chained after them** — `if ! go test; then
exit 1`.  P2(a) chained with `;` and pushed a red gate to `origin`.

## P2(c) — the bindings, and the chord that is not the only door

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestTheSwapActionsAreInTheKeyMapAndSoAreRebindable` | **FR-1.5** — the swap and the station toggle are ACTIONS, so a user can rebind them | **Plant 2026-09-09:** an action dropped from the map → **CAUGHT** (`not in the key map, so a user cannot rebind it`) |
| `TestTheSwapHasANonChordRoute` | **FR-1.6** — the chord is NOT the only door | **Plant 2026-09-09:** the plain key removed, leaving only `ctrl+o` → **CAUGHT**.  The accessibility lens leaned toward blocking on this at DISCOVER; it is now a gate |
| `TestPressingTheSwapKeyOnALiveStationDoesNotSwitch` | The binding goes THROUGH `canSwap` | **Plant 2026-09-09:** the gate asked and its answer ignored → **CAUGHT** (`the binding bypassed canSwap, which makes the key a second carrier of the D-1 rule`) |
| `TestPressingTheSwapKeyFromStandbySwitches` | Both routes work from STANDBY | Sweeps **every bound key** for the action, not just the first — testing only the chord would leave FR-1.6's own subject untested |

**The fixture refused to fake a chord.**  `keyPress` fatals rather than guessing when it cannot express
a key name, so a test cannot quietly drive past the real key path — the seam the 0.15.0 build log
records being burned by twice.  It was extended to express the chord rather than the test being
weakened to avoid it.

## P2(d) — NFR-7, the bounded silent station

**The safety lens's finding, made into a gate.**  STANDBY holds every track *including the alert rail*
— correct, and it is what stops a station on standby putting a tornado warning to air.  The cost is a
hazard that is neither broadcast nor visible, and the operator is the only person who can end it.

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestAHeldAlertInStandbyRaisesANotice` | A hazard held while silent is visible | By the watched RED — the notice was a `nil` stub |
| `TestTheHeldNoticeEscalatesWithTime` | **NFR-7's exit** — it escalates rather than sitting unchanged | **Plant 2026-09-09:** the ladder flattened to one rung → **CAUGHT** (`reads identically at 30s and at 20m`) |
| `TestAnEmptyRailInStandbyRaisesNothing` | A station holding NOTHING is at rest, not a hazard withheld | **Plant 2026-09-09:** the empty-rail return made unreachable → **CAUGHT**.  Crying wolf here would train the operator to ignore the notice that matters |
| `TestARunningStationRaisesNoStandbyNotice` | It does not persist once the station is draining | Passes; the negative direction |
| `TestARepeatedStationMessageDoesNotRestartTheStandbyClock` | A repeat of the same power is a refresh, **not a transition** | **WRITTEN BECAUSE A PLANT SURVIVED — see below** |

### A plant survived, and this time the PLANT was right

**Plant C made every `StationMsg` restart the standby clock (`if true`), and nothing failed.**

INST-3 says a surviving plant indicts the plant first.  It was checked, and here the plant was sound:
**every fixture above sends exactly one station message**, so a clock that restarts on repeats never
manifests.  **The gate was blind, not the plant bad.**

The consequence it would have shipped: a station silent for an hour looks *freshly quiet* the moment
any other message arrives, and **the escalation never climbs past its first rung** — the notice exists
and quietly never gets louder, which is worse than not having it.

`TestARepeatedStationMessageDoesNotRestartTheStandbyClock` was written for it, verified to **PASS clean
and FAIL under the plant**.  **This is the first surviving plant this release that indicted the gate
rather than itself**, and it is the reason the rule says to check both.

## P2(e) — the console is wired to the real schedule

**`lineup.Publish` gets its first consumer.**  The executor was deliberately empty through 0.15.0 and
said so — the seam existed and had no reader, which is a different thing from a no-op pretending to be
wired.

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestPublishCarriesThePowerWithTheLineup` | The effect carries the schedule AND the power, so no reader holds a torn pair | **Plant 2026-09-09:** the power replaced with a constant → **CAUGHT** twice (`got STOPPED want RUNNING`, `want OFF AIR`).  **The test also caught its own vacuity first**: swept from the zero power, the first transition was a no-op that published nothing, and the `no Publish was emitted, so this asserted nothing` fatal fired.  Fixture fixed to make every transition real |
| `TestTheConsoleGetsItsOwnMessagesWhileInactive` | Console-scoped messages reach it **while it is inactive** | **Plant 2026-09-09:** the category removed so they fell through to the active surface → **CAUGHT**.  A console that only learns while on screen is stale the instant it is swapped to — and the stale things here are the running order and whether the station is on the air |

### An existing guard caught the new seam, and that is the system working

`TestExecutorsRefuseToBeBuiltWithoutTheirSeams` **walks the executors struct by reflection** and demands
every nil-able field be either checked at construction or **declared optional**.  Adding `publish`
failed it immediately:

> `seam "publish" is neither checked nor declared optional: a nil one would reach production unmeasured`

**Declared optional, with the reason**: nil means no surface is listening, which is every build before
the console existed.  **Refusing to build without it would make the schedule depend on a UI** — the
wrong direction entirely.

**This is a derived guard written by an earlier release catching a field added by this one**, with no
edit to the guard itself.  That is what INST-1 buys.

## P3(c) — the rotation's narration class

**A VALUE, NOT A MECHANISM.**  The arbiter already suspends a lower class for a higher one and resumes
it — eight tests pinned that before this class existed — so the rotation joining the card path needed a
constant and nothing else.

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestTheRotationIsTheLowestClass` | Everything interrupts the programme | **By the watched RED**: the class was added deliberately in the WRONG position first, so the failure was an assertion (`it is 2`) rather than a compile error.  **Plant 2026-09-09:** placed above the severe read → **CAUGHT**.  Walks the class type by its sentinel, so a class added later is covered without editing it |
| `TestARotationReadIsSuspendedByASevereRead` | A severe read suspends the rotation, which then RESUMES | **Plant 2026-09-09:** the ordering inverted → **CAUGHT** (`got: duck,speak:rotation-line,speak:read-line,restore` — no pause at all) |
| `TestARotationReadIsSuspendedByATakeover` | A takeover suspends the rotation | **WRITTEN WEAK, THEN FIXED — see below** |
| The 5 pre-existing suspension pins | The shared arbiter did not regress | All still green.  **This is the regression evidence that matters** for a change to an arbiter three paths share |

### A second surviving plant, the same shape as the first

**Disabling the arbiter's suspension entirely (`&& false`) failed nothing** in the takeover test.

The reason: it asserted only that the rotation **finished**, and a rotation that is never paused
finishes just as happily as one paused and resumed.  **"It completed" was never evidence that anything
gave way.**

Strengthened to assert the `pause ... resume` trace, and verified to fail under the plant.

**Both surviving plants this release share one shape:** a test asserting an OUTCOME that occurs either
way, instead of the MECHANISM that was supposed to produce it.  Worth naming, because it is not the
same as a missing test — the test existed, ran, and passed for the wrong reason.

### A test that was wrong where the code was right

My first assertion expected the severe read to run as an **aside** over the suspended rotation.  It does
not, and should not: *aside* marks a **takeover's** line, the one whose visualizer does not follow it.
**The code was right and the expectation was wrong**, so the expectation changed.

## P3(a) — segments become a script

**The join S0 named.**  The rotation composes `[]synth.Segment` and plays it on the engine; a card
carries `lineup.Script` and is read through the arbiter.  Everything else was already in place, so
this translation is deliberately the ONLY new idea in P3(a).

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestSegmentsBecomeAScriptOfLines` | Every segment is a `PartLine`, the words survive, and **no tone opens it** | **Plant 2026-09-09:** a tone added → **CAUGHT** (`got "alerts"`).  A tone is a promise of a hazard; sounding one before an ordinary report teaches a listener to ignore the one that matters |
| `TestBlankSegmentsAreDroppedNotSpoken` | A blank segment is dropped, not spoken | **Plant 2026-09-09:** the drop made unreachable → **CAUGHT**.  A blank part is dead air under a callout the band has already promised |
| `TestAnEmptyRotationComposesAnEmptyScript` | An all-blank compose yields a script the executor declines | **PLANT SURVIVED — AND IT WAS BENIGN.  See below** |

### A third outcome: the plant was not a defect

Returning a script of **one blank part** instead of an empty one failed nothing.

Checked, per INST-3, before blaming the gate — and here the plant simply **is not a defect**:
`Script.Empty()` is defined on whether any part has TEXT, not on the part count
(`platform/lineup/script.go:98-105`).  A script of one blank part is empty by that definition, the
executor declines it identically, and nothing observable changes.

**Three outcomes are possible when a plant survives, and this release has now seen all three:** a bad
plant that never reached its subject (P0), a blind gate (P2, P3(c)), and a plant that was not a defect
(here).  Recording which is which is the skill the roster exists to teach.

## P3(a2) — the executors can build a location report

**AT PARITY. Nothing produces one yet.**  This is the T3.2a discipline that worked in 0.14.0: wire the
capability, claim *nothing changes*, then switch the producer over in its own commit.

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestALocationReportCardIsBuiltNotDeclined` | The decline that read *"read by the main track, which arrives with T3.2"* is gone | **By the watched RED**, which quoted the decline back: `got lineup.Failed — this is the decline that said 'read by the main track...'` |
| `TestALocationReportThatComposesNothingIsDeclinedNotAired` | A card that composed nothing is DECLINED, not aired | A card on the air with no words is silence under a callout the band has already promised |

### Two existing guards caught this change, and neither needed editing to do it

1. **The seam guard** (`TestExecutorsRefuseToBeBuiltWithoutTheirSeams`) fired on the new `compose`
   field: *"neither checked nor declared optional: a nil one would reach production unmeasured"*.
   **Declared optional WITH A DATED OBLIGATION**: it becomes REQUIRED at P3(d), because once the direct
   path is deleted a nil composer means location reports never reach the air at all — silence rather
   than a decline.  **Recorded in the guard rather than remembered.**

2. **`TestEffectsNotYetEmittedAreDeclinedNotHalfDone`** pinned the decline reasons by naming the task
   that would end them, and **failed the moment that task arrived**: *"want one line naming T3.2"*.
   Updated deliberately to the new truth — the build half declines only for a missing composer now,
   while the **speak half still names T3.2**, because the two halves are separate commits at parity.

**A pin that names the task which will retire it is a pin that tells you when it is stale.**  That is
worth copying.

## P3(a3) — the speak half

**Both halves of the T3.2 decline are now gone.**  Still at parity: nothing produces a
`LocationReport` card yet.

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestALocationReportIsSpokenAsTheRotationClass` | The rotation reads as `narrateRotation` in the standard voice — **not as a takeover** | **Plant 2026-09-09:** the class selection made unreachable, so it read as a takeover → **CAUGHT** (`duck,aside:...` — the aside is a takeover's line, whose visualizer does not follow it) |
| `TestEffectsNotYetEmittedAreDeclinedNotHalfDone` | An unknown slot is DECLINED, not silently spoken | **Plant 2026-09-09:** the decline made unreachable → **CAUGHT** |

### The pin fired at both halves, which is what it is for

This table names the task that will retire each decline.  It failed at **P3(a2)** when the build half
arrived, and again at **P3(a3)** when the speak half did.  Each time it was updated deliberately.

**The location-report SPEAK row is now GONE from the table** rather than reworded: it is no longer
declined at all, and a row asserting a decline that never happens would be a check that cannot fail.
The positive assertion lives in its own test instead.

## P3(a4) — the producer, and the staging that keeps the air single-owner

**The rotation's turn is now a card.**  The switch below is the reason this commit is inaudible to a
listener who has not asked for it, and every gate names the failure it was watched to catch.

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `TestASynthesisedReadBecomesAMainTrackCard` | A need becomes ONE `LocationReport` card on `MainTrack`, **and the step also asks for its words** | **Plant m6 2026-09-09:** the settle skipped → **SURVIVED first**, because the test threw the effects away with `_ = fx`.  Gate rewritten to require a `BuildCard` for that card and a `Publish` after it; re-planted → **CAUGHT** |
| `TestASecondNeedForTheSameLocationDoesNotQueueTwice` | FR-2.5 at the schedule level | **Plant m2:** `ReadID` made non-deterministic → **CAUGHT** |
| `TestARotationCardIsNamedAfterItsLocationAndNothingElse` | **The MECHANISM**: the id is a pure function of the ref, and does not collide with a burst's | The duplicate gate above would pass under a remembered-set implementation; this one would not.  Written because two earlier plants survived by asserting an outcome that occurs either way |
| `TestTheLocationCanBeReadAgainOnceItsCardHasLeft` | A rotation comes ROUND | Without it, a permanent refusal reads each location once and then plays nothing, and the duplicate gate above would still pass |
| `TestNeedsReadOnAStoppedStationQueuesNothing` | Admission is a promise to read (DR-3), so a track that cannot advance takes none | **Plant m1:** the gate deleted → **CAUGHT** |
| `TestARotationCardIsTheDirectorsOwn` | The origin is `FromDirector`, matching the stale-tune transition | **Plant m4:** origin set to Observer's → **CAUGHT** |
| `TestANeedWithNoHeadlineQueuesNothing` | A card is showable from proposal (DR-7) | **Plant m7b:** the headline replaced by the ref → **CAUGHT** |
| `TestTheMergeIsOffUntilItIsAskedFor` | An unrecognised `WATCHPOST_MAINTRACK` is **off**, never live | **Plant n1:** any non-empty value reads as live → **CAUGHT**.  Nine values checked, including `1`, `true`, `LIVE` and `" live"` |
| `TestOnlyLiveOwnsTheAirAndDarkStillReports` | Dark observes the REAL producer and still does not own the air | **Plants n2, n3** → **CAUGHT** |
| `TestEveryStageNamesItself` | Every stage names itself, **walked from the registry** (INST-1), and a corrupt value says `undeclared` rather than reading as `off` | **Plants s1, s2, s3** → **CAUGHT** |
| `TestTheDeckReportsTheNeedAndLeavesTheAirAlone` | Live reports the fact and **does not also start audio** | **Plants n5, n6, n7** → **CAUGHT**.  The air half is proved by a blank `mode`: `setMode` is `startSynth`'s second statement |
| `TestAStaleNeedIsNotReported` | A need that arrived after the listener moved on queues nothing, **at every stage** | **Plant n4:** the epoch guard deleted → **CAUGHT** |
| `TestADarkMainTrackCardIsDeclinedAtTheAirAndNeverReachesTheVoice` | The card stops one call short of the voice, and the decline is **Routed** | **Plants n8, n10** → **CAUGHT**.  Unrouted would raise a RELAY FAULT window every rotation turn while dark (I-2) |
| `TestTheAlertRailReadsWhateverTheMainTrackStageIs` | **The rail is not staged** | **Plant n9:** the decline widened to every slot → **CAUGHT**.  This is the gate that stops the merge silencing hazards |
| `TestEveryPathToASynthesisedReadGoesThroughTheOneSeam` | `startSynth` has exactly one caller, **derived by walking the AST** (INST-1) | A fourth call site is correct in isolation and wrong in company, so no behavioural test sees it.  **Zero callers is a FATAL** here, not a pass (INST-2) — and at P3(d) zero becomes the right answer and this gate is rewritten to say so |
| `TestTheNeedIsReportedFromTheOneSeam` | `lineup.NeedsRead` is constructed in exactly one place | A second site would queue the card under a staleness rule only one of them checks |
| `TestACardsKeyResolvesBackToTheLocationItNames` | A card's key resolves to the right location, **including the second entry** | **Plant c1:** the resolver returns the first entry always → **CAUGHT** |
| `TestComposingForAnUnknownLocationFailsByName` | A location the listener removed fails by NAME, not as an empty report | **Plants c2, c3, c4** → **CAUGHT**.  An empty report becomes a card on the air with nothing to say |
| `TestTheDarkRunRecordsTheNeedItWouldHaveActedOn` | The dark run's only instrument exists, names the stage **from the switch**, and carries `fresh=` | **Plants s4, s6, s7** → **CAUGHT**.  A dark run missing this line is not a quiet run, it is a run that proves nothing |
| `TestTheMergedStationHoldsItsPropertiesUnderRandomTiming` | Five properties over 300 randomised runs — one voice, one identity, **no card read more times than admitted**, the rail airs first, **and the rail is PREPARED first** | **Plant p1b:** `toPrepare`'s precedence reversed → **SURVIVED**, which is what property 5 was written for; re-planted → **CAUGHT**.  **Plant p6:** the producer made a no-op → **CAUGHT by the reach assertion**, not by a property, which is the instrument checking itself |

### Two gates were written because a plant survived, and both are recorded as such

**m6** and **p1b** are the release's second and third instances of a gate that watched the state instead
of the work.  m6's test had the effects in hand and discarded them; p1b's properties all watched what
took the air, and the defect was in what got COMPOSED, where nothing takes the air at all.

### One survival is recorded as BENIGN rather than fixed

**s5** — removing the `radioDebugOn()` guard at the call site changes no behaviour, because `debugLog`
is gated inside; only whether the line is BUILT.  **No allocation gate was added**: that guard earns its
place on the takeover path M4 measures, and `needsRead` runs a few times a minute.  A gate here would be
theatre, and the honest record is that this one is convention.

**m9b** — both proposal error guards removed, and `Queue`'s own `check()` refuses the card anyway.  The
guards stay (a discarded error is worse than a redundant branch) and the record says they carry nothing.

---

# P4 onward — reconstructed 2026-09-13

**THE ROSTER STOPPED AT P3(a4) ON 2026-09-09 AND THE WORK DID NOT.**  118 commits and rulings D-32
through D-123 landed with no roster entry.  Each was gated and mutant-covered at the time; none of it
was written here.  A roster that lags the code describes a release that no longer exists, and BUILD
exit is judged on this file.

**THE VERDICTS BELOW WERE RE-RUN ON 2026-09-13, not recalled.**  The whole two-letter corpus was
re-executed against the tip — 64 mutants outside `app`, 18 inside it — because a verdict remembered
from a session that has since been compacted is not evidence.  That re-run is itself the finding of
this pass: see "What a full re-run found that no gate could".

## What a full re-run found that no gate could

**Four mutants that PASS every gate this repository has, and measure nothing.**

`make mutant-anchors` proves each mutant still FINDS its line.  `make mutant-check` proves each still
COMPILES with its tests.  **Neither asks whether anything still FAILS when it is applied** — and that
is the only question that makes a corpus evidence rather than decoration.

| Mutant | What it guarded | Why it no longer measures |
|---|---|---|
| `mAB1` | D-104 — one scroll control spanning BOTH tables | **THE RULE WAS RETIRED, NOT BROKEN.**  D-106 ruled the line-up does not scroll (*"Location Pool Scrolls, Line-up doesnt"*), and its detector `TestOneScrollControlSpansBothTables` was deleted with it.  The mutant now defends a design the product deliberately abandoned |
| `mAA2` | D-92 — a settings heading is never drawn over no rows | The detector `TestNoSettingsGroupIsDrawnEmpty` still exists and reads rendered output, so this points at the FIXTURE rather than the rule |
| `mCA` | BD-8 — the speak names its lane | Escalated: see the verdict table |
| `mDC` | RD-2/D-82 — the bed is a shared resource | Escalated: see the verdict table |

**THE SHAPE, NAMED:** *a mutant whose RULE was superseded.*  It is not anchor drift — the anchor
matches perfectly.  It is a corpus entry that outlived the requirement it was written for, and the
only instrument that can see it is a full re-run.

**ADOPTED BY THE HUM LEAD 2026-09-13: `make mutant-verdicts`.**  A full corpus verdict sweep is owed
at BUILD exit and before SHIP, with the log kept — because nothing else in the toolchain can tell a
live rule from a dead one.  It is deliberately OUTSIDE `verify` (tens of minutes, one mutant at a
time), on the same standing as `make journey`.

**A SURVIVOR IS ESCALATED TO `./...` BEFORE IT IS BELIEVED**, and that step is the sweep's own
control: a mutant run only against the package it EDITS reads as SURVIVED when its detector lives
elsewhere.  On 2026-09-13 escalation turned two of four apparent survivors into CAUGHT (`mCA`, `mDC`)
— the check was watched reporting false survivors without it, which is why it is not optional.

**The sweep is a thin driver over `run.sh`**, whose own properties are already gated by
`mutant-check`: that it reports each verdict with the right exit code, refuses a red baseline and a
dirty tree, and calls a crash CAUGHT.  The escalation is the only NEW behaviour, and it is the one
with the watched failure above.

## The corpus as re-measured, 2026-09-13

**82 two-letter mutants, re-run against the tip.  78 CAUGHT on the first pass; two of the four
survivors were false — caught by a test in another package, and found only because a survivor is
escalated to `./...` before it is believed.  Of the two that were real, one was a live coverage
hole and is fixed; one guards a retired rule.**

| Mutant | Verdict | Detector |
|---|---|---|
| `mAA1` — the listeners location leaks to the console | CAUGHT | TestTheConsoleDrawsOnlyTheSettingsThatApplyToIt |
| `mAA2` — a group heading is drawn over nothing | **SURVIVED → CAUGHT** | nothing, until the helper stopped re-deriving the rule; now `TestTheConsoleDrawsOnlyTheSettingsThatApplyToIt` |
| `mAA3` — the window opens on a row nobody can see | CAUGHT | TestTheWindowOpensOnARowThisSurfaceDraws |
| `mAA4` — tab lands in a group this surface hides | CAUGHT | TestTheKeyboardNeverFocusesAHiddenRow |
| `mAA5` — the window ignores the surface entirely | CAUGHT | TestSetupAllocBudget |
| `mAB1` — the rail breaks between the tables | **RETIRED 2026-09-13** | It guarded D-104 (one control over BOTH tables).  D-106 retired that rule — *"Location Pool Scrolls, Line-up doesnt"* — and deleted its detector with it, so the mutant defended a design the product had abandoned.  **RETIRED BY THE HUM LEAD 2026-09-13**, never self-issued.  The rail's actual behaviour is pinned by `mAC2`, `mAC3` and `TestTheScrollControlIsThePoolsAlone` |
| `mAB2` — the pool is whatever is left over | CAUGHT | TestTheScrollControlIsThePoolsAlone |
| `mAB3` — the hazard touches the place | CAUGHT | TestTheHazardNeverTouchesThePlace |
| `mAB4` — the pointer inherits the digits ceiling | CAUGHT | TestEnterOpensTheRowThePointerIsOn |
| `mAB5` — the number stencil floats again | CAUGHT | TestTheNumberStencilHangsItsNumbers |
| `mAB6` — the focused slot is not painted | CAUGHT | TestTheFocusedSlotIsPaintedLikeTheFocusedLocation |
| `mAC1` — the control claims the running order | CAUGHT | TestThePointerWalksAndTheWindowFollows |
| `mAC2` — the pools headings scroll away | CAUGHT | TestThePoolTableDrawsTheStationsCandidates |
| `mAC3` — wx stn outranks zip | CAUGHT | TestThePoolGivesUpWxStnBeforeZip |
| `mAC4` — population sets against the gutter | CAUGHT | TestPopulationSetsUnderItsOwnHeading |
| `mAC5` — the bed is drawn twice again | CAUGHT | TestTheBedIsDrawnOnceInsideTheStationSection |
| `mAC6` — the air box leaves the station section | CAUGHT | TestTheAirBoxAlwaysDrawsBothRows |
| `mAC7` — the air boxs width gets a second owner | CAUGHT | TestGoingOnAirFillsTheLiveRow |
| `mAC8` — the console reads the terminals palette | CAUGHT | TestTheConsoleArmsTheThemesForeground |
| `mAC9` — the frames tint assumes a palette index | CAUGHT | TestTheConsoleArmsTheThemesForeground |
| `mAD1` — a fabricated card loses its mark | CAUGHT | TestAFabricatedCardStillSaysSoInItsRule |
| `mAD2` — the mark follows the headline | CAUGHT | TestAFabricatedCardStillSaysSoInItsRule |
| `mAD3` — an empty slot loses its way in | CAUGHT | TestAnEmptySlotStillCarriesItsHandle |
| `mAD4` — the manifest caption drifts off centre | CAUGHT | TestTheUpNextCardFollowsTheReferencesOrder |
| `mAD5` — the card stamp goes long again | CAUGHT | TestTheUpNextCardFollowsTheReferencesOrder |
| `mAE1` — the bed takes the bare arrows back | CAUGHT | TestTheBedsControlsReachTheStation |
| `mAE2` — the bed chip names the wrong key | CAUGHT | TestABedWithRelaysIsOffered |
| `mAF1` — the pool builds its own row | CAUGHT | TestThePoolsWeatherIsObserversWeather |
| `mAF2` — the pool falls out of the pipeline | CAUGHT | TestTheStationsPoolSurvivesACommit |
| `mAF3` — a moved station never fetches its pool | CAUGHT | TestARestationedPoolIsWhatGetsFetched |
| `mAG1` — the pool row loses its fire and seismic | CAUGHT | TestThePoolRowCarriesEveryMark |
| `mAG2` — the pool row opens observers selection | CAUGHT | TestEnterOnAPoolRowOpensThatLocation |
| `mAG3` — a second enter only closes a card | CAUGHT | TestEnterOnAPoolRowOpensThatLocation |
| `mAG4` — the recent pipeline stops fetching seismic | CAUGHT | TestCadenceTableIsTheDoc |
| `mAH1` — out of fence hazards stay on the frame | CAUGHT | TestAnOutOfFenceCardLeavesTheProjection |
| `mAH2` — the up next box loses its ground | CAUGHT | TestTheUpNextBoxWearsTheModalsGround |
| `mAI1` — the station settings leak to observer | CAUGHT | TestSetupAllocBudget |
| `mAI2` — the typeahead writes the listeners default | CAUGHT | TestTheTransmitterRowResolvesLikeTheDefaultLocation |
| `mAI3` — a bare save ends the stations fallback | CAUGHT | TestASaveWithNoChoiceLeavesTheStationBorrowing |
| `mAI4` — a new radius is stored but not derived | CAUGHT | TestSettingTheServiceRadiusReDerivesThePool |
| `mAI5` — the service radius ignores its bounds | CAUGHT | TestTheServiceRadiusRefusesWhatIsOutOfBounds |
| `mAI6` — a borrowed epicentre looks chosen | CAUGHT | TestABorrowedEpicentreSaysSo |
| `mAJ1` — the running order forgets the weather | CAUGHT | TestConsoleFrameAllocBudget |
| `mAJ2` — the running orders marks are not the places | CAUGHT | TestTheRunningOrderCarriesEachBeatsWeather |
| `mAK1` — the bed tunes through the listeners list | CAUGHT | TestSteppingTheBedTunesTheChosenRelaysOwnMounts |
| `mAK2` — the bed offers relays nothing streams | CAUGHT | TestTheBedFenceKeepsOutWhatTheResolverWouldOffer |
| `mAK3` — the bed is offered with nothing to carry | CAUGHT | TestABedWithNothingToCarryIsNotOffered |
| `mAK4` — a moved station keeps the old regions relays | CAUGHT | TestAMovedStationReResolvesItsRelays |
| `mAL1` — a live card is offered management | CAUGHT | TestALiveCardOffersNeitherControl |
| `mAL2` — an out of range move is sent anyway | CAUGHT | TestChangePositionRefusesWhatIsOutOfRange |
| `mAL3` — the drop needs no confirmation | CAUGHT | TestDropAsksAndExplainsWhatItDoes |
| `mAL4` — the question lets keys through | CAUGHT | TestChangePositionSendsTheMove |
| `mAL5` — the box title is not a modal title | CAUGHT | TestTheBoxTitleReadsLikeAModalTitle |
| `mAM1` — a move lands one slot low | CAUGHT | TestChangePositionSendsTheMove |
| `mAM2` — the running order stops at fourteen | CAUGHT | TestTheConsoleShowsAtMostFifteenMainTrackSlots |
| `mAN1` — the console builds day cells it never draws | CAUGHT | TestTheLineupRowsBuildNoDayCells |
| `mAN2` — the alloc budget measures an empty console | CAUGHT | TestConsoleFrameAllocBudget |
| `mAO1` — the keymap answers the card windows keys | CAUGHT | TestChangePositionSendsTheMove |
| `mAP1` — the fence trusts the arrivals own tie | CAUGHT | TestAPointlessAlertIsAdmittedOnlyByTheTrackedTie |
| `mAP2` — the fence is built without the tie set | CAUGHT | TestTheDecksFenceCarriesTheScopesTieSet |
| `mAP3` — the arrival carries its raw id | CAUGHT | TestTheArrivalsKeyAndTheTieSetsKeyAreTheSameKey |
| `mAP4` — the tie set ignores the scopes radius | CAUGHT | TestTheDecksFenceCarriesTheScopesTieSet |
| `mAQ1` — the console never inherits the fire threshold | CAUGHT | TestTheConsoleReadsTheOperatorsFireThreshold |
| `mAQ2` — the console pins the fire threshold again | CAUGHT | TestTheConsoleReadsTheOperatorsFireThreshold |
| `mAR1` — the memo forgets an input | CAUGHT | TestConsoleFrameAllocBudget |
| `mAR2` — the memo forgets the running order | CAUGHT | TestTheConsoleMemoKeyCoversEverythingTheTablesShow |
| `mAR3` — the writer forgets to bump the generation | CAUGHT | TestTheConsoleMemoKeyCoversEverythingTheTablesShow |
| `mAR4` — the pool is appended onto the cached lineup | CAUGHT | TestTheJoinNeverWritesIntoTheCache |
| `mAR5` — the index is built on every frame | CAUGHT | TestConsoleFrameAllocBudget |
| `mAR6` — the memo never hits | CAUGHT | TestConsoleFrameAllocBudget |
| `mCA` — the lane is dropped from the speak | CAUGHT | TestABurstSoundsOneToneForItsHighestSeverityEvent — **found only by escalating to ** |
| `mCB` — a hazard is played as the programme | CAUGHT | TestARailCardThatReachesTheProgrammesReaderIsRefused |
| `mCC` — a halted read comes home finished | CAUGHT | TestAReadHaltedAfterItStartedComesHomeFailed |
| `mCD` — a cards read moves the bed | CAUGHT | TestAMainTrackReadTellsTheDirectorNothingAboutTheBed |
| `mCE` — standby leaves the words going out | CAUGHT | TestTheOperatorSilencingTheStationStopsAReadAlreadyGoingOut |
| `mCF` — the air goes back and the programme reads on | CAUGHT | TestTheOperatorSilencingTheStationStopsAReadAlreadyGoingOut |
| `mDA` — a step publishes twice | CAUGHT | TestAStepPublishesOnceAndLast |
| `mDB` — a stranger publishes | CAUGHT | TestAnEventForACardTheLineupDoesNotHoldChangesNothing |
| `mDC` — the bed is not a shared output | CAUGHT | TestOnlySharedOutputWorkRidesTheLane — **found only by escalating to ** |
| `mDD` — the lane outlives the loop | CAUGHT | TestTheLaneRetiresWithTheLoop |
| `mDE` — a queued run is discarded on a coin flip | CAUGHT | the test binary did not survive it: panic: test timed out after 10m0s |
| `mDF` — an effect is dropped from its run | CAUGHT | TestALongRunningEffectDoesNotDelayTheNextEvent |

## The 2026-09-13 corpus sweep: seven survivors, and what each was owed

**`make mutant-verdicts` over all 314 mutants — 307 CAUGHT, 7 SURVIVED.**  Nothing from this release
survived; all seven predate 0.16.0.  **A survivor is not a defect and it is not a bad plant until it
has been read**, and these split three ways.

### Five owed a test, and have one

| Mutant | The rule nothing measured | Why it survived |
|---|---|---|
| `mS4` | the scroll thumb tracks the window | **A past defect with no pin.** `chromeAt` records it: "until D-87 … the rail drew a thumb that never moved."  The near miss was `TestTheScrollGutterCarriesOnlyTheThumb`, which asserts there is exactly ONE thumb — still true when it is frozen |
| `mS0` | a column never bleeds into its neighbour | **No caller could express the fault.**  Both boxes are built by `shell`, which already pads every row, so the pad in the join is a no-op.  Extracted to `joinColumns` and driven directly |
| `m42` | only the Air Quality Alert is an "Alert" by name | **A rule held by a DIFFERENT rule.**  The only other "… Alert" product is Blue Alert, which the civil-emergency table decides first.  Tested as a POLICY, with products from other programmes |
| `m25` | an alert already read aloud is not offered again | **A test that proves the door opens says nothing about whether it closes.**  The path is covered end to end and drives an UNREAD event; nothing drove one already said |
| `mK3` | two presses cannot race the same reader | **A FALSE SURVIVOR, and the cause is the SWEEP.**  Its detector was already there and documents its own reliability: "20/20 under -race, but only ~81/100 without it, with batch-to-batch swings from 50% to 92% … the guarantee is `make race`."  **`run.sh` does not use `-race`**, so the sweep read a probabilistic detector on its bad days.  A deterministic test was added anyway — see below |

### Two owe NOTHING, and the code already says so

**These are equivalent mutants, and recording them is the honest disposition** — the same standing
`s5` and `m9b` were given at P3.

| Mutant | Why no test can catch it |
|---|---|
| `m16` | `scopeEvents`' `ok` check is **defence in depth on the hazard path**.  Verified rather than taken on trust: BOTH builders of the tracked set go through `alertKeysOf`, which inserts a key only when `NormalizeID` returns ok — so no unusable key can ever be in the set to match.  The source states it in as many words, including "deleting it changes no behaviour, and no test can catch that, which is stated here rather than left to look like coverage" |
| `m43` | The Marine arm matches `Contains(product, "Marine")` and the live catalogue contains exactly ONE such product — Marine Weather Statement — because every other marine product names warning, watch or advisory and is decided above.  **Narrowing it changes nothing without inventing a product the Weather Service does not issue.**  This is the distinction from `m42`: that rule is about products that do not exist YET and is therefore testable as policy; this one is about the catalogue as it stands, and "whether marine products belong here at all is an open ruling" |

**Both are KEPT rather than retired.**  Each guards a rule that is true today because of a
*neighbouring* rule; if that neighbour moves — the civil-emergency table, `alertKeysOf`, the NWS
catalogue — the mutant becomes catchable and the next sweep says so.

### The sweep's own blind spot, found by using it

**`mK3` was never a coverage gap.**  Its detector existed, and the sweep could not see it work:
`run.sh` runs `go test <pkgs>` with **no `-race`**, and that test is only reliable under the race
detector — its own comment says so, with numbers.

**SO A MUTANT WHOSE ONLY DETECTOR NEEDS `-race` READS AS SURVIVED.**  That is a third way to get a
false survivor, beside the two already known:

| cause | fix |
|---|---|
| run against only the package it EDITS, detector lives elsewhere | escalate to `./...` — already done |
| the detector is probabilistic and needs `-race` | **not handled** |
| the rule was genuinely retired | HUM LEAD retirement |

**RECOMMENDED, NOT SELF-ISSUED:** the escalation step should re-run a survivor under `-race` as well
as against `./...` before reporting it.  Only survivors pay, so the cost is bounded by how many there
are — seven, this time, against 314 mutants.  Without it every sweep will keep reporting the same
false survivor and someone will keep spending an hour on it.

**AND `mK3` NOW HAS A DETERMINISTIC TEST REGARDLESS**, because a detector that needs a flag and a
coin-flip is a poor guarantee for a hazard path: `TestTwoPressesCannotRaceTheSameReader` stands in the
middle of the critical section at `send` and asks whether a second press can get in.  It catches the
mutant without `-race` and without repetition.

---

## Roster reconciliation — 2026-09-15, at BUILD exit

**This file states its own standard at the top — *"Every gate this release adds carries an evidence
line naming a failure that was actually WATCHED"* — and at the bottom that **BUILD exit is judged on
this file**.  Red team checked it: of the 141 `Test*` names it cites, **16 no longer exist**.  Two
were already recorded as retired (`TestOneScrollControlSpansBothTables` above, and
`TestNeedsReadOnAStoppedStationQueuesNothing`).  The other 14 were not, and this table is their
correction.

**This is roster drift, not a coverage hole.**  Every row below names the successor that carries the
same property, and each was checked to exist.  But the rows above still cite the dead names, and a
reader auditing this release would find 10% of its evidence pointing at nothing — which is precisely
the failure mode this file names at `gates.md:382-388` about the roster lagging the code by 118
commits, reproduced one layer up.

| Cited gate (no longer exists) | Retired by | What now carries the property |
| --- | --- | --- |
| `TestTheBannerReadsTheDirectorsPowerNotALocalFlag` — **labelled a SAFETY gate** | `19eae8f` | `TestEveryStationStateSaysWhatItIs`, `TestTheTransitionNamesTheStateItWouldReach` (`broadcaster_station_test.go`) |
| `TestTheStateIsLegibleWithoutColour` (FR-5.3) | `19eae8f` | `TestEveryStationStateSaysWhatItIs` — the words carry the state; `TestEachTickerLaneNamesItselfWithoutColour` holds the same rule for the tape |
| `TestTheOnAirBoundaryIsStatedToTheOperator` | `19eae8f` | `TestEveryStationStateSaysWhatItIs` (the boundary sentence is part of the state's own words) |
| `TestTheBannerIsVariantC` | `19eae8f` | `TestTheStationLineCarriesTheGainControl`, `TestTheTransitionHintIsAnchoredToTheRightEdge` |
| `TestTheConsoleShowsAtMostTenMainTrackCards` (FR-3.1) | `29ddb8e` | `TestTheConsoleShowsAtMostFifteenMainTrackSlots` — the cap moved from 10 to 15; named at `gates.md:489` |
| `TestALocationReportIsSpokenAsTheRotationClass` | `92e288c` | `TestARotationCardIsTheDirectorsOwn`, `TestARotationCardIsNamedAfterItsLocationAndNothingElse` |
| `TestARotationReadIsSuspendedByASevereRead` | `92e288c` | D-82's give-way: `TestTheAlertRailDrainsBeforeTheMainTrack` |
| `TestARotationReadIsSuspendedByATakeover` | `92e288c` | as above |
| `TestADarkMainTrackCardIsDeclinedAtTheAirAndNeverReachesTheVoice` | `92e288c` | `TestOnlyDarkReportsAndNothingElseChanges`, `TestTheDarkRunRecordsTheNeedItWouldHaveActedOn` |
| `TestOnlyLiveOwnsTheAirAndDarkStillReports` | `92e288c` | as above |
| `TestTheAlertRailReadsWhateverTheMainTrackStageIs` | `92e288c` | `TestAStoppedRadioStillReadsTheAlertRail` |
| `TestTheDeckReportsTheNeedAndLeavesTheAirAlone` | `92e288c` | `TestTheDarkRunRecordsTheNeedItWouldHaveActedOn` |
| `TestTheRotationIsTheLowestClass` | `92e288c` | `TestAProposalAndARotationReadShareOneIdentity` and the tone-class tests in `domains/radio/cast` |
| `TestTheUpNextBoxWearsTheModalsGround` | D-134/D-136, 2026-09-15 | `TestTheUpNextBoxWearsTwoGrounds`, `TestTheUpNextBoxWearsTheSameBlueAsTheDirectionBand` — the ruling changed, and the gate changed with it |

**The corpus figure at `gates.md:520` is also superseded.**  It reads *"over all 314 mutants — 307
CAUGHT, 7 SURVIVED"*.  The current corpus is larger and the survivor count is lower; the authoritative
record is `07-readiness/mutant-verdicts.log`, re-run and promoted at this exit, and the figure there
is the one to cite.

**The lesson, recorded rather than smoothed over.**  A roster is evidence, and evidence rots at the
rate the code changes.  Deleting a test is not the defect — the successors are real and were verified
to exist.  Not recording the deletion in the document that is *judged at exit* is.  `gates.md:402`
and `:440` already show the correct form; fourteen rows simply never got one.
