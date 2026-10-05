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
| `TestTheConsoleShowsNothingItWasNotPublished` | It invents nothing | Passes on an empty lineup; the negative direction of the row above. **Plant 2026-09-17:** a card appended to the projection the schedule never published → **CAUGHT** |
| `TestTheConsoleNumbersTheMainTrackSlots` | The ten slots are addressable `0`-`9` (FR-2.4) | **Plant 2026-09-09, and THE FIRST ATTEMPT WAS INVALID.**  v1 deleted `strconv.Itoa(i)` and left `i` and the import unused → **BUILD FAILURE, which is not evidence either way** (the removed-a-use shape).  v2 computed the index, discarded it, and returned a constant handle → **CAUGHT** |
| ~~`TestTheConsoleShowsAtMostTenMainTrackCards`~~ → `TestTheConsoleShowsAtMostFifteenMainTrackSlots` | A rolling view of ten; an eleventh does not reach the frame (FR-3.1) | **Plant 2026-09-09:** the bound removed → **CAUGHT** (`an eleventh card reached the frame`) |
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
| ~~`TestTheBannerReadsTheDirectorsPowerNotALocalFlag`~~ → `TestEveryStationStateSaysWhatItIs` | **FR-5.1** — the state is the Director's, never a local flag | **Plant 2026-09-09:** the published state ignored and a local flag kept → **CAUGHT** (`got STOPPED`).  This is a SAFETY gate, not a display one: the swap gate depends on this value |
| ~~`TestTheBannerIsVariantC`~~ → `TestTheStationLineCarriesTheGainControl` | **D-21** — a labelled field with the transition in parentheses | By the watched RED: the field existed, nothing rendered it, and the test named each missing part |
| ~~`TestTheStateIsLegibleWithoutColour`~~ → `TestEveryStationStateSaysWhatItIs` | **FR-5.3** — the words carry the state, not the colour | By the watched RED under `--ascii` |
| ~~`TestTheOnAirBoundaryIsStatedToTheOperator`~~ → `TestEveryStationStateSaysWhatItIs` | **FR-5.5** — the console says what ON AIR means | **Plant 2026-09-09:** the boundary line blanked → **CAUGHT** |
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
| `TestSwappingTOTheConsoleIsAlwaysPermitted` | Arriving is not the hazard the ruling bounds | Swept across **every power state, derived** from `Power.String()`'s end. **Plant 2026-09-17:** arriving refused while the station is live → **CAUGHT** |
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
| `TestAnOverrideInTheKeyTableChangesTheConsolesChord` | **FR-1.5** — an override in the user's `[keys]` table changes the chord, the Router answers the new key, and the help PRINTS it | **Plant 2026-09-16:** the Router given the raw map → **CAUGHT** (`the console answers [ctrl+b B]; the operator's table says ctrl+g`).  Help given the raw map → **CAUGHT** |
| `TestAConsoleOverrideThatCollidesIsRefusedAtBuild` | **FR-1.5 / D-15** — a console override colliding inside the console's own scope is a build error, never a silent win | **Plant 2026-09-16:** the merge left unvalidated → **CAUGHT** |
| `TestAnOverrideForTheOtherSurfaceDoesNotBreakTheConsole` | **FR-14** — two scopes share one override table, so an entry meant for the other is DROPPED with a note, not refused | —. **Plant 2026-09-17:** the merge dropping the console's swap action → **CAUGHT** |
| ~~`TestTheSwapActionsAreInTheKeyMapAndSoAreRebindable`~~ → `TestEveryConsoleSwapActionHasKeysAndHelp` | **WITHDRAWN 2026-09-16 (D-158).**  It asserted the swap actions were IN the key map — a PROXY for FR-1.5, whose exit sentence is "an override in the user's key table changes the chord".  `broadcasterKeyMap()` reached the Router unmerged, so no `[keys]` entry could change any console binding: the requirement was unmet for the whole of 0.16.0 and this gate was green beside it.  The 2026-09-09 plant was real and caught what it aimed at; what it aimed at was not the requirement | **Red team round 3** (junior-dev lens) |
| `TestTheSwapHasANonChordRoute` | **FR-1.6** — the chord is NOT the only door | **Plant 2026-09-09:** the plain key removed, leaving only `ctrl+o` → **CAUGHT**.  The accessibility lens leaned toward blocking on this at DISCOVER; it is now a gate |
| `TestPressingTheSwapKeyOnALiveStationDoesNotSwitch` | The binding goes THROUGH `canSwap` | **Plant 2026-09-09:** the gate asked and its answer ignored → **CAUGHT** (`the binding bypassed canSwap, which makes the key a second carrier of the D-1 rule`) |
| `TestPressingTheSwapKeyFromStandbySwitches` | Both routes work from STANDBY | Sweeps **every bound key** for the action, not just the first — testing only the chord would leave FR-1.6's own subject untested. **Plant 2026-09-17:** only the first bound key honoured → **CAUGHT** |

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
| `TestARunningStationRaisesNoStandbyNotice` | It does not persist once the station is draining | Passes; the negative direction. **Plant 2026-09-17:** the notice drawn for RUNNING too → **CAUGHT** |
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
| ~~`TestTheRotationIsTheLowestClass`~~ → `TestAProposalAndARotationReadShareOneIdentity` | Everything interrupts the programme | **By the watched RED**: the class was added deliberately in the WRONG position first, so the failure was an assertion (`it is 2`) rather than a compile error.  **Plant 2026-09-09:** placed above the severe read → **CAUGHT**.  Walks the class type by its sentinel, so a class added later is covered without editing it |
| ~~`TestARotationReadIsSuspendedByASevereRead`~~ → `TestTheAlertRailDrainsBeforeTheMainTrack` | A severe read suspends the rotation, which then RESUMES | **Plant 2026-09-09:** the ordering inverted → **CAUGHT** (`got: duck,speak:rotation-line,speak:read-line,restore` — no pause at all) |
| ~~`TestARotationReadIsSuspendedByATakeover`~~ → `TestTheAlertRailDrainsBeforeTheMainTrack` | A takeover suspends the rotation | **WRITTEN WEAK, THEN FIXED — see below** |
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
| ~~`TestALocationReportIsSpokenAsTheRotationClass`~~ → `TestARotationCardIsTheDirectorsOwn` | The rotation reads as `narrateRotation` in the standard voice — **not as a takeover** | **Plant 2026-09-09:** the class selection made unreachable, so it read as a takeover → **CAUGHT** (`duck,aside:...` — the aside is a takeover's line, whose visualizer does not follow it) |
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
| `TestTheLocationCanBeReadAgainOnceItsCardHasLeft` | A rotation comes ROUND | Without it, a permanent refusal reads each location once and then plays nothing, and the duplicate gate above would still pass. **Plant 2026-09-17:** a permanent refusal (each location read once) → **CAUGHT** |
| ~~`TestNeedsReadOnAStoppedStationQueuesNothing`~~ → `TestANeedOnAStoppedStationIsQueuedButNotRead` | Admission is a promise to read (DR-3), so a track that cannot advance takes none | **Plant m1:** the gate deleted → **CAUGHT** |
| `TestARotationCardIsTheDirectorsOwn` | The origin is `FromDirector`, matching the stale-tune transition | **Plant m4:** origin set to Observer's → **CAUGHT** |
| `TestANeedWithNoHeadlineQueuesNothing` | A card is showable from proposal (DR-7) | **Plant m7b:** the headline replaced by the ref → **CAUGHT** |
| `TestTheMergeIsOffUntilItIsAskedFor` | An unrecognised `WATCHPOST_MAINTRACK` is **off**, never live | **Plant n1:** any non-empty value reads as live → **CAUGHT**.  Nine values checked, including `1`, `true`, `LIVE` and `" live"` |
| ~~`TestOnlyLiveOwnsTheAirAndDarkStillReports`~~ → `TestOnlyDarkReportsAndNothingElseChanges` | Dark observes the REAL producer and still does not own the air | **Plants n2, n3** → **CAUGHT** |
| `TestEveryStageNamesItself` | Every stage names itself, **walked from the registry** (INST-1), and a corrupt value says `undeclared` rather than reading as `off` | **Plants s1, s2, s3** → **CAUGHT** |
| ~~`TestTheDeckReportsTheNeedAndLeavesTheAirAlone`~~ → `TestTheDarkRunRecordsTheNeedItWouldHaveActedOn` | Live reports the fact and **does not also start audio** | **Plants n5, n6, n7** → **CAUGHT**.  The air half is proved by a blank `mode`: `setMode` is `startSynth`'s second statement |
| `TestAStaleNeedIsNotReported` | A need that arrived after the listener moved on queues nothing, **at every stage** | **Plant n4:** the epoch guard deleted → **CAUGHT** |
| ~~`TestADarkMainTrackCardIsDeclinedAtTheAirAndNeverReachesTheVoice`~~ → `TestOnlyDarkReportsAndNothingElseChanges` | The card stops one call short of the voice, and the decline is **Routed** | **Plants n8, n10** → **CAUGHT**.  Unrouted would raise a RELAY FAULT window every rotation turn while dark (I-2) |
| ~~`TestTheAlertRailReadsWhateverTheMainTrackStageIs`~~ → `TestAStoppedRadioStillReadsTheAlertRail` | **The rail is not staged** | **Plant n9:** the decline widened to every slot → **CAUGHT**.  This is the gate that stops the merge silencing hazards |
| `TestEveryPathToASynthesisedReadGoesThroughTheOneSeam` | `startSynth` has exactly one caller, **derived by walking the AST** (INST-1) | A fourth call site is correct in isolation and wrong in company, so no behavioural test sees it.  **Zero callers is a FATAL** here, not a pass (INST-2) — and at P3(d) zero becomes the right answer and this gate is rewritten to say so. **Plant 2026-09-17:** a second caller of `startSynth` in a throwaway file → **CAUGHT**, named by file and function |
| `TestTheNeedIsReportedFromTheOneSeam` | `lineup.NeedsRead` is constructed in exactly one place | A second site would queue the card under a staleness rule only one of them checks. **Plant 2026-09-17:** a second `lineup.NeedsRead` literal in a throwaway file → **CAUGHT** |
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
| `mAB1` | D-104 — one scroll control spanning BOTH tables | **THE RULE WAS RETIRED, NOT BROKEN.**  D-106 ruled the line-up does not scroll (*"Location Pool Scrolls, Line-up doesnt"*), and its detector ~~`TestOneScrollControlSpansBothTables`~~ → retired with D-106 (the line-up does not scroll) was deleted with it.  The mutant now defends a design the product deliberately abandoned |
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
| `mAA1` — the listeners location leaks to the console | **RETIRED 2026-09-25** | It guarded D-92 (a mode's own settings appear only in its mode). 0.18.0 **D-70** made every setting shown on every surface - *"settings show now be the same across ALL UI modes"* - so the mutant changed nothing a test could see. **RETIRED BY THE HUM LEAD 2026-09-25**, never self-issued. ~~TestTheConsoleDrawsOnlyTheSettingsThatApplyToIt~~ → `TestEverySurfaceOffersEverySetting`; each row still writes only its own values (the D-18 table's test) |
| `mAA2` — a group heading is drawn over nothing | **SURVIVED → CAUGHT** | nothing, until the helper stopped re-deriving the rule; then ~~TestTheConsoleDrawsOnlyTheSettingsThatApplyToIt~~ (gone with D-70); now `TestSetupGoldenEveryTab`, `TestEveryWindowClearsItsMargins` and the reachability test - a tab drawing another tab's headings |
| `mAA3` — the window opens on a row nobody can see | CAUGHT | TestTheWindowOpensOnARowThisSurfaceDraws |
| `mAA4` — tab lands in a group this surface hides | CAUGHT | TestTheKeyboardNeverFocusesAHiddenRow |
| `mAA5` — the window ignores the surface entirely | **RETIRED 2026-09-25** | It mutated `rowVisible` to draw every row on every surface - the behaviour 0.18.0 **D-70** now requires (*"settings show now be the same across ALL UI modes"*), and its line is gone. **RETIRED BY THE HUM LEAD 2026-09-25** (0.18.0 D-73), never self-issued. What `rowVisible` still decides - a tab draws its own rows - is held by `TestSetupGoldenEveryTab` and the reachability test. M4 ("settings bleed") is owed a new definition: a row writing another mode's values |
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

**SUPERSEDED 2026-09-15 — see the line below.  The figures in this paragraph describe the corpus as
it stood mid-release and are kept because the reasoning around them is still the record of how those
seven were triaged.**

> **CURRENT, 2026-09-18, on the COMMITTED tree `4d1cb84` (VALIDATE exit): `make mutant-verdicts` over all
> 380 mutants — 375 CAUGHT, 5 SURVIVED, 0 NO EVIDENCE**, 4 h 28 min. The survivors are the by-design
> five below. At the REVIEW-exit sweep on `73bce2b` (376 / 4 / 0) `mBM2` read as CAUGHT — by
> `TestOnlyMastercontrolWritesTheBand` "only against `./...`", an ordering-dependent catch and not a
> pin; on the VALIDATE-exit tree it SURVIVED again, which is its dispositioned state.
> **The number's blind spot, beside it (INST-5):** a detector that needs `-race` reads as SURVIVED under
> this sweep ("The sweep's own blind spot", below). The record is `mutant-verdicts.log` beside this file,
> which opens with the commit hash. The paragraph that follows is the BUILD-exit sweep, kept as history.
>
> **BUILD exit, 2026-09-17, on the COMMITTED tree `e956747`: `make mutant-verdicts` over all 379 mutants —
> 374 CAUGHT, 5 SURVIVED, 0 NO EVIDENCE**, 4 h 46 min wall clock. The five survivors are exactly the
> by-design set dispositioned at the end of this file — `m16`, `m43`, `mBM1`, `mBM2`, `mSC3` — and
> none is new. The per-mutant record is `mutant-verdicts.log` beside this file (403 lines). **A first
> run of the same day was VOID and was not recorded**: started on a tree still being edited, 372 of
> 379 SKIPPED (dirty tree); a second, on `6b1b621`, read the two known survivors as INVALID because a
> record file had grown to 15 MB in a bad edit and `TestNoTrackedBinaries` was red on the unmutated
> tree — the gate built two days earlier refusing its author's own commit. Stopped at 87, fixed, run
> a third time to completion.
>
> The 2026-09-15 figure, kept for the record: 355 mutants — 353 CAUGHT, 2 SURVIVED, 0 NO EVIDENCE.  The two survivors are `m16_scopeevents_drops_ok` and
> `m43_marine_narrowed`, the equivalents dispositioned at the end of this file.  The five that have
> since gone were retired or re-pointed as the code moved; the authoritative per-mutant record is
> `07-readiness/mutant-verdicts.log`, promoted by the target itself.
>
> **This is the first sweep run against a committed tree.**  `run.sh` refuses a dirty one, and until
> 2026-09-15 there was not one — which is why the filed record had been 22 mutants behind the corpus
> it described (red team, I3).

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
| `TestTheConsoleShowsAtMostTenMainTrackCards` (FR-3.1) | `29ddb8e` | `TestTheConsoleShowsAtMostFifteenMainTrackSlots` — the cap moved from 10 to 15; named at the `mAM2` row of the sweep table |
| `TestALocationReportIsSpokenAsTheRotationClass` | `92e288c` | `TestARotationCardIsTheDirectorsOwn`, `TestARotationCardIsNamedAfterItsLocationAndNothingElse` |
| `TestARotationReadIsSuspendedByASevereRead` | `92e288c` | D-82's give-way: `TestTheAlertRailDrainsBeforeTheMainTrack` |
| `TestARotationReadIsSuspendedByATakeover` | `92e288c` | as above |
| `TestADarkMainTrackCardIsDeclinedAtTheAirAndNeverReachesTheVoice` | `92e288c` | `TestOnlyDarkReportsAndNothingElseChanges`, `TestTheDarkRunRecordsTheNeedItWouldHaveActedOn` |
| `TestOnlyLiveOwnsTheAirAndDarkStillReports` | `92e288c` | as above |
| `TestTheAlertRailReadsWhateverTheMainTrackStageIs` | `92e288c` | `TestAStoppedRadioStillReadsTheAlertRail` |
| `TestTheDeckReportsTheNeedAndLeavesTheAirAlone` | `92e288c` | `TestTheDarkRunRecordsTheNeedItWouldHaveActedOn` |
| `TestTheRotationIsTheLowestClass` | `92e288c` | `TestAProposalAndARotationReadShareOneIdentity` and the tone-class tests in `domains/radio/cast` |
| `TestTheUpNextBoxWearsTheModalsGround` | D-134/D-136, 2026-09-15 | `TestTheUpNextBoxWearsTwoGrounds`, `TestTheUpNextBoxWearsTheSameBlueAsTheDirectionBand` — the ruling changed, and the gate changed with it |

**The corpus figure earlier in this file is also superseded** (the *"2026-09-13 corpus sweep"* section).  It reads *"over all 314 mutants — 307
CAUGHT, 7 SURVIVED"*.  The current corpus is larger and the survivor count is lower; the authoritative
record is `07-readiness/mutant-verdicts.log`, re-run and promoted at this exit, and the figure there
is the one to cite.

**The lesson, recorded rather than smoothed over.**  A roster is evidence, and evidence rots at the
rate the code changes.  Deleting a test is not the defect — the successors are real and were verified
to exist.  Not recording the deletion in the document that is *judged at exit* is.  `gates.md:402`
and `:440` already show the correct form; fourteen rows simply never got one.

## Survivors that survive BY DESIGN, added at this exit

Three mutants in the corpus are **tripwires**: they guard a rule that is currently held by a
*different* rule, so the mutation changes nothing any test can see and the sweep reports SURVIVED.
That is the D-42 shape `platform/lineup/air.go` already states in as many words — *"A TRIPWIRE, AND
ITS MUTANT SURVIVES BY DESIGN"* — and the reason to keep them is that the protecting rule can move.

**Each was CHECKED before being called a tripwire, not assumed.**  A mutant that survives because the
rule is unguarded and a mutant that survives because the rule is doubly guarded look identical in the
log, and the difference is the whole verdict.

| Mutant | Why it cannot fail today | What makes it fail later |
|---|---|---|
| `mSC3` | `routeKey` keeps its own Router when `keyAction` declines a key.  No fall-through path in that switch mutates, so taking the returned model instead is equivalent | The day a case mutates `r` *before* falling through, the surface below silently receives a Router already half-changed by a rule that declined to act |
| `mBM1` | `bodymemo`'s bound invariant only fires when eviction is already wrong.  Three tests drive the bound itself and stay green under this mutation, correctly | The day the eviction condition is edited — `!ok && len(m.items) >= m.max` is one `&&` away from never evicting — the cache grows without limit with every observable still reading correct |
| `mBM2` | `delete` on a key the map does not hold is a no-op in Go, and every real call site reaches `evictLocked` only when the map is occupied | The day `evictLocked` is called from a path that does not already know the map is non-empty. Read as CAUGHT once, at the REVIEW-exit sweep, by `TestOnlyMastercontrolWritesTheBand` "only against `./...`" — an ordering-dependent catch, not a pin; SURVIVED again at VALIDATE exit. By design stands |

**`mGL1` is recorded here for the opposite reason: its FIRST version was a bad mutant and was
replaced.**  It deleted the build-time guard that is itself a test — so it could not be caught *by*
that test, which is the tautology P-1 warns about: a test cannot be the instrument and the subject at
once.  It now mutates the `Glyphs` struct, which is the production change the guard exists to see,
and it is CAUGHT.

## The gate layer, consolidated (2026-09-16)

Eight gates over one model of the build system, and one registry over every exemption table. **Every
one's watched failure is BY CONSTRUCTION**: `cmd/watchpost/gateattacks_test.go` runs 36 attack
specimens on every invocation — each a plant against a synthetic Makefile, workflow and gate list —
and fails if any attack survives or any correct spelling is refused. The attack list was written and
committed BEFORE the model (`06_docs/gate-attack-list.md`), which inverts the order that failed three
remediation rounds. One attack was found by the real tree during implementation (A37) and added to
both files before the fix was trusted.

| Gate | Property | Evidence |
|---|---|---|
| `TestTheThreeGateListsAgree` | verify, CI and required-gates.txt agree, every difference declared | **By construction:** A17, A26 CAUGHT; base fixture PASSES |
| `TestEveryGateShapedTargetIsListedOrExempt` | a target that runs a check is listed or says why not | **By construction:** A1–A4 CAUGHT. **On the real tree:** found `build-diag` and `lint-update`, which the regex it replaced could not see |
| `TestNoRequiredGatesCIStepIsSilenced` | no step-level or job-level `if:`/continue-on-error/`\|\| true` on a required gate without a reason | **By construction:** A19–A25 CAUGHT — including `- if: false` as a step's first key, the spelling that defeated three rounds; B-ok1, B-ok2 PASS |
| `TestNoGateIsAnswerableFromCache` | no gate's `go test` omits `-count=1` | **By construction:** A11 CAUGHT |
| ~~`TestNoRequiredGateRecipeCannotFail`~~ → `TestEveryRequiredGateCanFail` | no required gate's recipe discards its exit status | **By construction:** A8 CAUGHT |
| ~~`TestEveryRequiredGatesCheckerHasAControl`~~ → `TestEveryControlIsReached` | every project checker a required gate runs is exercised by a self-test that RUNS | **By construction:** A5, A6, A7, A9, A10 CAUGHT; A-ok1 PASSES |
| ~~`TestEveryRequiredGateRunsACheck`~~ → `TestEveryRequiredGateCanFail` | a required gate's recipe runs something that can fail | **By construction:** A18 CAUGHT; A37 PASSES. **On the real tree:** its first run found the parser dropping every recipe line after a column-0 comment — the real `mutant-check` read as running nothing |
| `TestEveryBuildTargetTrimsThePath` | every `go build` carries `$(TRIMPATH)` | **By construction:** A12–A15 CAUGHT; A-ok3 PASSES |
| `TestTheAllocBudgetSelectsItsPins` | the alloc pattern reaches its pins; reaching nothing fails | **On the real tree:** eight pins counted; a pattern reaching none fails under the floor |
| `TestEveryExemptionRowIsRealAndStillNeeded` | every row of every registered table: real reason, subject exists, rule would still fire | **On the real tree:** its first run found three `identityExempt` rows that matched nothing — no-ops reading as considered exceptions — and they were deleted |
| `TestEveryExemptionTableIsRegistered` | the table list is derived from the package source; an unregistered table fails | **By construction:** the package's own `map[string]string` declarations are parsed, so a new table cannot be added without registering (A33) |
| the model's `mustRun` | an empty source is COULD-NOT-RUN, never pass (FR-11.3) | **By construction:** A34, A35, A36 |

### The Makefile half, executed (2026-09-16, round two)

A blind reviewer defeated the parsed model six ways and named the cause — a semantic question decided
by pattern, in a language whose global constructs the parser never reads. These three gates stop
parsing. Every checker is stubbed red and `make <gate>` is RUN; make and sh are the oracle.

| Gate | Property | Evidence |
|---|---|---|
| `TestEveryRequiredGateCanFail` | every required gate exits non-zero when its checks do | **By construction:** E1–E3, E5–E13, A18 CAUGHT on every run; E-ok PASSES. **On the real tree:** `make fmt` exited 0 with gofmt red — `test -z "$(gofmt -l …)"` is true when gofmt prints nothing, and a gofmt that failed to run prints nothing. Fixed to read gofmt's status |
| `TestVerifyCanFail` | `make verify` itself exits non-zero on a red gate | **By construction:** E15 (`-@` on the entry point) CAUGHT |
| `TestEveryControlIsReached` | every control goes red when its checker does — painted, then run | **By construction:** A5, A6, A9, E14 CAUGHT; E-ok PASSES. Sibling `_test.sh` controls are painted as controls, not as checkers |
| E4 `.SHELLFLAGS` | | **NOT APPLICABLE on GNU Make 3.81** (macOS); skipped by name, runs for real on CI's 4.x. The oracle's verdict is only as good as the make it runs under |

### Round three: the green control, the phony audit, stubs that answer by argument (2026-09-16)

| Gate | Property | Evidence |
|---|---|---|
| `TestEveryRequiredGateCanFail` (green half) | every required gate exits 0 with every stub green — the POSITIVE CONTROL; red under green is UNJUDGEABLE by name, never "sound" | **By construction:** H1–H4 report UNJUDGEABLE; H-ok PASSES both halves |
| `TestNoFileSilencesARequiredGate` (then named for the phony audit) | every required make target is `#  Phony target` in make's database; absent from the database is an error unless declared CI-only | **By construction:** J1 CAUGHT. **On the real tree:** four required gates were not in `.PHONY`; `touch lint-identity && make lint-identity` said "is up to date" and exited 0 having run nothing |
| `TestVerifyCanFail` (reach) | green everywhere → `make verify` exits 0; ONE checker red → exits non-zero, through treelock's delegation | **By construction:** H5 CAUGHT; E-ok PASSES |
| `TestEveryControlIsReached` (by argument) | the stub answers `--self-test` with the control status and anything else with the checker status; the proof paints the control alone and tries every carrier | **By construction:** K1, K2, K3, K4 CAUGHT; K-ok (`&&`) PASSES |
| the registry's nonce | `exists` is proved against a subject the REGISTRY chose, not the table | **By construction:** L1 CAUGHT; `satisfied` remains a declared ceiling |

### Round four: paint one thing, not the world (2026-09-16)

| Gate | Property | Evidence |
|---|---|---|
| `TestEveryRequiredGateCanFail` (per key) | for each stub a gate REACHES — checkers, toolchain sub-commands, prerequisites, `$(MAKE)` recursion — painted red ALONE, the gate goes red | **By construction:** M1–M4, M-ok. **On the real tree:** `mutant-check` with `exit $$rc` → `exit 0` CAUGHT; `lint-injector … \|\| true` CAUGHT — both had read as sound under whole-world painting |
| `TestVerifyCanFail` (per checker) | `make verify` goes red for each checker it reaches through its own delegation; green everywhere first; red-everywhere must reach the gates or it is UNJUDGEABLE | **By construction:** M7. **On the real tree:** `verify: lint` + `-@` CAUGHT |
| `TestNoFileSilencesARequiredGate` (then named for the phony audit) (walking) | every recipe-bearing node a required gate reaches is phony; absence from the database is an error unconditionally | **By construction:** N1, N2. **On the real tree:** rule deleted + `.DEFAULT` + a `ciOnly` row CAUGHT |
| the registry's nonce (per call) | `exists` is proved against fresh random hex, not a constant a table can name | **By construction:** N3 |
| the child environment | `MAKEFLAGS`/`MAKELEVEL` stripped, so a parent make's `-i` cannot reach the oracle | **By construction:** N4 |

### Round five: observe, don't guess (2026-09-16)

A fifth blind adversary found six Criticals with one cause: `reach()` was regexes over recipe text —
round one's defect, in the one function execution had not replaced. In the SHIPPED tree, `p10`'s
`$(A2DH) p10 check` had never been painted red alone. The stubs now RECORD what ran, and nothing in
the executed half reads a recipe.

| Gate | Property | Evidence |
|---|---|---|
| `TestEveryRequiredGateCanFail` (recorded) | a gate's reach is what its stubs RECORDED during the green run — through `$(VAR)`, `$$(…)`, `$(CURDIR)/…`, `sh -c`, `$(MAKE) -s`, pattern rules — and each recorded key is painted red alone | **By construction:** O1–O9. **On the real tree:** `p10` is judged for `a2dh` for the first time (recorded reach: `a2dh python3 ledger-ratified.sh lint-ledger.sh p10-ledger-mirror.py p10-unmatched.sh`); **and the first recorded run CAUGHT a live discard the regex oracle could not see** — `release-matrix`'s `(… && sha256sum … \|\| shasum …)` dropped a failed `sha256sum` (O10). Fixed |
| `TestVerifyCanFail` (recorded) | `make verify` goes red for EVERY key its own delegation recorded — 34 on the real tree — not only the checkers | **By construction:** H5, M7, E15 still CAUGHT; E-ok passes |
| `TestNoFileSilencesARequiredGate` (replaces the phony audit) | for every node make itself reports considering (`make -n --debug=v`), a file of that name is created and the gate re-run green; if it RECORDS LESS, that file silences it. No database walk, no `.PHONY` parse | **By construction:** J1–J3, N1, N2, O6; O-ok (an order-only directory prerequisite is NOT a silencer — the file makes `mkdir -p` red, which is not silence) |
| `TestEveryControlIsReached` (recorded) | carriers are the required gates whose green run RECORDED the control | **By construction:** K1–K4, A5–A10 still CAUGHT |
| the child environment | `MAKEFILES`, `GNUMAKEFLAGS`, `MAKE`, `MAKEOVERRIDES` stripped as well; a `MAKEFILES` that sets `.SHELLFLAGS := -ec` would turn every `;` discard red and certify a neutered gate (the QUIET direction) | **By construction:** O7 — E8 is run with `MAKEFILES` set in the parent and must still be CAUGHT |
| the one list left | the toolchain stubs (`go gofmt a2dh python3 expect golangci-lint govulncheck shasum sha256sum`) are enumerated. A command not on it is real: red under green (loud), except a real command that exits 0 on an empty tree under `\|\| true`, which is not judged | **DECLARED** in the attack list; O10 is what forgetting one looked like |

### Round six: the tree is the tree (2026-09-16)

A sixth blind adversary ran twelve attacks against the recorded oracle: 9 SURVIVED, 1 CAUGHT, 1
false positive, 1 correct pass. Every survivor was one cause — **the scratch tree was empty**, so
whatever a recipe asked of the tree answered the opposite of the repository. Round two had closed
that in the red direction only.

| Gate | Property | Evidence |
|---|---|---|
| the oracle's tree | a shared clone of the repository with the working tree copied over it — a real `.git`, `go.mod`, the real `scripts/` — and stubs by PATH ALONE, reached through each script's `#!/usr/bin/env` shebang; not one byte of the tree is rewritten. An executable under `scripts/` without an env shebang is refused (it would run for real, unrecorded) | **By construction:** P1 (`git diff --quiet && exit 0` skips in a clean repo — CAUGHT), P1-ok, P4 (`gate: go.mod` silenced by a file — CAUGHT because `go.mod` is real), P10-ok (`find \| xargs gofmt` records on BSD xargs because the tree has Go files). **On the real tree:** 16 `#!/bin/sh` and 4 `#!/usr/bin/expect -f` shebangs changed to `env` |
| `TestEveryRequiredGateCanFail` (per invocation) | every INVOCATION is a key — `go:test#1`, `go:test#2`, `scripts/x.sh#1` — painted alone | **By construction:** P2 (two `go test`, the first `\|\| echo` — CAUGHT), P11 (one script twice). **On the real tree:** 24 gates, 52 invocations |
| the `go build` stub | writes a recording stub at `-o`, so a compiled checker is a key (`built:out/x`) | **By construction:** P3 CAUGHT, P3-ok |
| `TestNoFileSilencesARequiredGate` | nodes from the REAL green run's `--debug=v`, never `-n`; one file at a time, then all at once | **By construction:** P5 (`ifeq (,$(findstring n,$(MAKEFLAGS)))` hiding a node from a dry run — CAUGHT) |
| `TestVerifyCanFail` (coverage) | `verify`'s record contains every key each non-CI-only gate records on its own | **By construction:** P6 (`verify-gates FAST=1`, `lint` skips under it — CAUGHT; three text layers had passed it) |
| `TestEveryControlIsReached` | a `go run ./tools/…` checker needs a control like a script does | **By construction:** P8 |
| P10 — a finding against the reviewer | `bad=$$(find \| xargs gofmt -l); test -z "$$bad"` was reported as a FALSE POSITIVE. The oracle was right: `;` throws away the substitution's status — the F-152 defect | **CAUGHT**, kept as a specimen |
| the ceiling, stated on every passing run (FR-11.5) | a toolchain by absolute path (A9), `env -i` (A12), a stub detected by its shebang (A8), `go build` without `-o`, any command not on the tool list | **DECLARED.** A recipe written to escape the oracle escapes it; that is a review finding |

### Round seven: the tree as CI has it, and the instrument's joints (2026-09-16)

A seventh blind adversary: 2 Critical, 4 Important, 4 Minor. The Criticals are round six one step
further — the clone was the tree but not the tree AS CI HAS IT — and a count the verify coverage
did not make. The Importants are joints in the instrument. **The Critical count fell 5 → 2 and the
findings no longer share one cause.**

| Gate | Property | Evidence |
|---|---|---|
| the oracle's tree | a shared clone with HEAD DETACHED at the commit, NO tags, and only the source's tracked and unignored files laid over it — what a depth-1 `actions/checkout` has. One function serves the real tree and every specimen | **By construction:** Q1 (skip when detached — CAUGHT), Q2 (skip without tags — CAUGHT), Q3 (skip when a git-ignored file is absent — CAUGHT) |
| `TestVerifyCanFail` (coverage by COUNT) | for every key, `verify`'s invocation count ≥ the sum of the non-CI-only gates' own counts | **By construction:** Q4 (`FAST=1` skipping ONE of two `go test` — CAUGHT; P6 had closed the whole-key case only). **Real tree:** 38 invocations under verify, counts match |
| the status encoding | `/` → `%2F`, reversible | **By construction:** Q5 (`scripts/quality_lint.sh` and `scripts/quality/lint.sh` collided under `tr / _` — CAUGHT) |
| the `go run` / `go build` key | the package is the first argument SHAPED like one; `<module>/x` is `./x`; `-o dir/` lands the stub at `dir/<basename>`; `built:` keys need a control | **By construction:** Q6 ×3, Q6-ok, Q9 (found SURVIVED once while building: `-o out/` had been taken as the package — fixed before trusting) |
| the shebang rule | anchored to end of line and DERIVED from the interpreter list — `python3.12` is a real interpreter | **By construction:** Q7 |
| the silence audit | the all-at-once set adds every non-phony rule make's database lists, so a `$(MAKE)` hop with its output hidden still has its node created | **By construction:** Q8 ×2 |
| `sh -c` / `bash -c` | run the REAL shell; the script inside records through its shebang | **By construction:** Q11-ok |
| the ordinal | a `mkdir` lock around read-count-append | **By construction:** Q13 |
| the ceiling (FR-11.5) | printed by ALL FOUR assertions, naming `ORACLE_*`, discards inside a script, and how git metadata is shaped | on every passing run |

### Round eight: the threat model, and the instrument in Go (2026-09-17)

| Gate | Property | Evidence |
|---|---|---|
| `tools/gateoracle` (package) | the oracle is a library with a `doc.go`; the stub is `tools/gateoracle/stub`, built once per test process and uninstrumented, so `bin/go`, `bin/sh` … are symlinks to it and no Go file carries shell; every decision — `goKey`, `packageShaped`, `buildOutput`, `hasShellCommandFlag`, `scriptArg`, `isSelfTest`, `encodeKey`, `specialTarget` — is a function with a known-case-first unit test | `stub_test.go`; `make lint-authoring` (AP-SHELL-01) refuses shell in Go; `make dupes` caught the one duplicate the move introduced |
| the threat model | DRIFT enforced; EVASION declared on every passing run and carried by the brief's question 10 | the ceiling sentence, all four assertions |
| `AssertEveryRequiredGateCanFail` etc. (drift, round eight) | `-c` anywhere in leading flags; `-o=`; `go test -c`; dot-named nodes; database-only nodes singly and the harmless set together; local `go run` and `built:` keys under the control obligation; parse-time invocations subtracted from a database run with a goal that does not exist | **By construction:** R1–R8 (100+ executed specimens). **Real tree:** 24 gates / 52 invocations, verify 38, 10 controls — unchanged from round seven, in half the time |
| `TestEveryShellScriptHasALedgerRow` | every non-Go source under `scripts/` has a ratified row; derived from the tree | 23 rows, F-156 |
| ci.yml `gnu-make` step | macOS runner installs GNU make ≥ 3.82 so the oracle is not COULD-NOT-RUN on one OS | R7; `TestNoRequiredGatesCIStepIsSilenced` accepts the step's `if:` because it is not a gate |

### Round nine: absence must be loud (2026-09-17)

The first round briefed to attack DRIFT under the written threat model: 1 Critical, 4 Important, 6
Minor; evasion reported separately. The Critical was a verdict the oracle had never asked for.

| Gate | Property | Evidence |
|---|---|---|
| `AssertEveryRequiredGateCanFail` (absent-alone) | for every stubbed tool a gate recorded — not `go`, not what the OS ships (`osShipped()`, derived) — the gate is run with that tool ABSENT from PATH and must go red, or record a fallback the green run did not | **By construction:** S1 ×3 CAUGHT, S1-ok ×3 (own-line skip is loud by its next line; the shipped checksum fallback). **Real tree:** 24 gates, 56 invocations (52 red-alone + 4 absent-alone) |
| `green()` | an `unjudged:<role>` in a green reach is UNJUDGEABLE; the shebang rule covers every tracked executable | **By construction:** S2 ×2 |
| the scratch between runs | `git reset --hard && git clean -fdx` before every run; the working tree committed in the scratch, so cleanliness answers as CI whatever is uncommitted here | **By construction:** S4, `TestADirtySourceTreeIsJudgedAsCIWouldHaveIt` |
| parse-time subtraction | once per makefile read the run reports | **By construction:** S3-ok |
| the instrument's own tests | eight more pure functions unit-tested; the ceiling's text pinned; three specimens renamed for the property that catches them | `stub_test.go` |

### BUILD-exit batch A: the three product rows the red team left open (2026-09-17)

| Gate | Property | Evidence |
|---|---|---|
| `TestA0150ConfigRoundTripsUnchangedAndGainsOnlyDefaults` | FR-6.3 / RS-6: a file v0.15.0 itself wrote decodes unchanged, gains only 0.16.0 defaults, and survives a 0.16.0 Save byte for byte | **By the watched RED:** the captured fixture carried `cast = "on"`, which the loader refused — a capture error, and the test said so before anything else could |
| `TestTheTransmitterQuestionStatesTheStorageBoundary` | FR-9.4: the storage boundary is stated where the operator sets the tower | **By the watched RED:** three phrases absent before the support line existed |
| `TestADumpCarriesNoCoordinates` | FR-9.4: no JSON the dumper writes names a coordinate | **By the watched RED:** a `lat` field planted in `dumpRecord` was reported and reverted |
| `TestAStationThatCannotComposeFaultsAndAListenerWhoDeclinedIsRouted` | F-150: a station that cannot perform FAULTS (not routed) and DR-21's window is owed; a deliberate non-delivery stays routed | **By the watched RED:** "no composer" and "a card with nothing to say" were `Routed: true` before the split |

### F-142 closed: the roster names tests that exist, and every gate names a failure it watched (2026-09-17)

Eighteen live rows named tests that had been renamed or deleted; each is now struck through with its
successor beside it, and the reconciliation ledger is the one section where a gone name may stand
alone. `TestTheRosterCitesTestsThatExist` holds that from now on — the drift's third recurrence is the
one that got a gate. Eight rows carried no watched failure; each now records a plant of 2026-09-17,
every verdict read from the test's own `--- FAIL` line, with a plant that failed to compile named
INVALID and redone rather than counted.

### BUILD exit, the three rulings (2026-09-17)

| Gate | Property | Evidence |
|---|---|---|
| `TestTheStationToggleAsksTheDirectorForTheOtherState` | FR-5.4: the bound control asks the Director for the other state, from STANDBY and from ON AIR, through the real key path | **By the watched RED:** the toggle made inert, both rows failed |
| `TestEveryRatifiedP10RowNamesCodeThatExists` | every ratified row in the ledger mirror names a file that exists and a symbol it still holds (F-140) | **By the watched RED:** `app/radio_queue.go · stopDwell` — the one stale row — named before its deletion, and nothing else |
| FR-8.6, amended | the pool is nearest-first from the population-filtered table, capped at `locations.PoolCap`; `TestThePoolIsCapped` and `TestAPoolWithNoFenceIsJustTheStation` hold the cap at N > cap and N = 1 | the requirement now says what the code does |

