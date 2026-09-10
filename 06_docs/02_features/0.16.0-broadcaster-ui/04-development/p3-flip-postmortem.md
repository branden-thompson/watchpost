---
title: "0.16.0 P3(b)+(d) — the flip: what broke, and how it passed every gate"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "REVERTED to the dark state on the HUM LEAD's instruction.  The approach question is OPEN and is the HUM LEAD's."
---

# The flip did not work, and twenty gates were green over it

**Ratified, built, and reverted the same day.**  A fresh adversarial reviewer found four blockers, every
one of them in the join between the narration arbiter and the audio source.  **The station read
nothing.**

**This document is not the bug report.  It is the account of how a change this broken passed FULL TDD,
fifteen plants, twenty gates, a property test and a first red team.**  That is the part worth the cost.

## What was wrong

| # | Defect | Mechanism |
|---|---|---|
| **1** | **The arbiter suppresses the report it is about to play** | `director.Run` → `admit` calls `mc.giveWay()` for every audible job, BEFORE the sequence runs.  `giveWayLocked` **HOLDS** a rendered source rather than dipping it, so `StartSource` runs with the engine suppressed and the report never plays.  **The rotation IS the bed; ducking for it ducks the thing being played** |
| **2** | **Every report is failed the instant it starts** | `StartSource` calls `halt()` first, and `halt()` emits `Status{Stopped}` **synchronously**.  That lands on the channel armed one line earlier, so the wait returns false before a word plays |
| **3** | **After a takeover the report never comes back** | `settle` **returns** as soon as it promotes a job; `takeBack` sits past that return.  `Suppress`/`Restore` is the only pair that pauses and resumes a source, and only the pause half is on the resume path |
| **4** | **A hazard cannot preempt a report** | nothing in the lineup can take an on-air card off early, and the blocking `Speak` holds the pump's single serial lane, so the takeover's cue and words queue behind it.  With `[r]` repeat-one the source never ends and the card is **on air for ever** |

**1, 2 and 3 are mine.  4 is FR-5.6 and P5 arriving early**, because a main-track card that holds the
air for minutes needs the paused-main-track concept that release does not have yet.

## Why every gate was green: four causes, in the order they mattered

### 1. Every test drove the seam, and the seam was a stub

`app/executors_test.go`'s bench wired `playReport` to a function returning `true` immediately.  So all
fifteen plants asked **which function is called, with what arguments, in what order** — and none asked
**what happens when the real one runs**.

**The reviewer's mutant proves the size of it: delete the ENTIRE release mechanism — nothing ever
releases the wait — and `go test ./app/` stays green.**  Re-run here: SURVIVED.

**I tested the wiring and called it the behaviour.**  `TestALocationReportIsPlayedByItsSourceUnderTheArbiter`
asserts the voice log contains `duck` and reads that as *"the arbiter took the air"*.  For a rotation the
duck is *"the engine will hold this source"* — **the assertion pins blocker 1 in place.**

### 2. A stated blind spot on the batch's central mechanism was treated as a disclosure

The build log says, in my own words, that `readReport` *"resolves a voice, which on a machine without one
is a 63 MB download inside a unit test"* — and uses that as the reason not to test it.  **That sentence
identifies the untested core of the batch and then moves past it.**

**INST-5 says state your blind spots.  It does not say a blind spot on the thing the batch exists to
change is acceptable.**  It was a STOP signal and I filed it as a footnote.

**And the fixture existed.**  `heldOutput` and `endlessSilence` in `app/radio_stop_test.go` drive a real
`player.Engine` with no network and no voice install.  The reviewer used them to prove all four blockers
in a worktree.  I did not look for them.

### 3. Every plant hit code I wrote; none hit the code I depended on

All four blockers live in `director.admit`, `engine.giveWayLocked`, `engine.halt` and `director.settle`.
**I mutated none of them.**  I *read* them and summarised them in a design document — and that is where
the second failure compounds the first.

### 4. The design claim that won the ruling was half a trace

`p3-flip-design.md` argues *"Shape B has no cost"* over a five-call chain: `dip` → `duck` → `Suppress` →
`giveWayLocked` → hold.  **Every link is correct and the chain is half the loop.**  It never traces the
RELEASE side, which is precisely where it is broken (blocker 3).

**A ruling was asked for on the strength of that.**  The trace was presented as a measurement — *"the S0
discipline applied again, answered by reading five live call sites with zero code written"* — and S0's
actual discipline is the opposite: **S0 RAN eight tests.**  Reading five call sites in one direction is
not a spike.

### And the batch ran far too long between check-ins

Between the ruling and the reviewer's report: **five commits and roughly two thousand lines**, with no
intermediate confirmation.  **A single end-to-end play-through would have ended it in a minute** — the
HUM LEAD named this before the blockers were known.

## What must change in the process, stated so it can be checked

| # | Rule | The failure it answers |
|---|---|---|
| **P-1** | **A seam a test cannot drive is NOT a covered seam.**  If the fixture stubs it, the batch is uncovered and that is a RED gate, not a note | cause 1 |
| **P-2** | **A stated blind spot on the mechanism the batch exists to change is a STOP.**  Escalate it; do not disclose it and continue | cause 2 |
| **P-3** | **Before claiming a fixture is impossible, grep the package for one.**  `heldOutput` and `endlessSilence` were already there | cause 2 |
| **P-4** | **Plant the code you DEPEND on** — at minimum every call named in the design's own reasoning | cause 3 |
| **P-5** | **A "costs nothing" claim over a call chain must trace BOTH directions** — the take and the give-back — or it is not a measurement | cause 4 |
| **P-6** | **RUN IT once before asking for a ruling on a shape.**  Reading is not a spike; S0 ran tests | cause 4 |
| **P-7** | **Check in when the shape is proven, not when the batch is finished.** | the cadence |

**Every one of these is a rule I had already written down somewhere.**  `quality-observations.md`, in a
commit made the SAME DAY, names *"a gate that watches the STATE and not the WORK"* three times and says
the tell is that the test can see the result and does not ask for the step.  **This is the fourth
instance, committed hours after writing the rule.**  Knowing a shape and being unable to see yourself in
it are different skills, and the instrument is what closes the gap — which is the argument for P-1 and
P-4 being gates rather than habits.

## The open question, and it is the HUM LEAD's

**Was the arbiter the right approach at all?**

The arbiter's model is *"a narration speaks OVER the bed"*.  **The rotation is not a narration over the
bed — it IS the bed.**  Blocker 1 is that mismatch expressed as a duck; blocker 4 is the same mismatch
expressed as a lane.  Both shapes — A and B — put the rotation through a mechanism built for
interruptions.

**That question is not answered here, deliberately.**  What this document owes is the account of the
failure and the process changes; choosing the approach is a SEV-0 architecture ruling, and the last one
was asked for on the strength of half a trace.
