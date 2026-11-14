---
title: "0.19.6 DISCOVER — lens RM: the role model and D-2"
date: 2026-11-14
phase: DISCOVER
lens: RM
status: "FILED — findings checked against e57afeaf"
---

# Lens RM — the role model, and what D-2 would take

## Verdict

**D-2 as ruled (the arbiter dissolves into the Director's step, and the Director owns all ducking)
cannot conform to D-1's principle 4 in 0.19.6, and should not land in it.** It changes what is
heard, not just where the code sits. The record's own precondition, *"it arrives when everything
reads through the Director"* (`director-build-log.md:1086`), is still unmet. One narration, the
`[space]`/`[w]` severe-event read, never reaches the Director (`app/executors.go:360` declines it;
`app/severe_read.go:264` runs it on the arbiter directly). Two audible beds the Director does not
duck are the Observer monitor and the on-air bed relay when it is not carrying. Taking over their
ducking changes Observer from one dip per card to one dip per drain, and moves the dip from the
start of the read to when the card is admitted. A test pins that Observer must *"sound exactly as
it did"* (`app/executors_test.go:868-870`). D-2 also needs new members in the closed effect set
(PL-6) and somewhere in the line-up for the monitor's own narration. Both are product rulings,
already tied to F-73 (P5's pause) and F-158 (P3(d)), and neither is a package move.

**What can land in 0.19.6 and conforms:**
- F-158's test, which the record says is owed and nobody has written;
- keeping the voice cluster (`arbiter.go`, `mastercontrol.go`, `read_script.go`) whole in the
  split's map;
- correcting the stale "Director"-means-arbiter wording.

**D-2 is separable from the split.** The split should land first, and D-2 should be ruled for a
later release, starting from the inventory in RM-1.

## What I checked and how

- **The record first, whole:**
  - 0.16.0's required reading;
  - `role-model.md`;
  - `director-charter.md` §§140-190, the disposition sheet;
  - `director-architecture.md` §§15-50 and 225-282;
  - `director-build-log.md` §§775-835, 1060-1132 and 2135-2232;
  - `rulings-d74.md`;
  - `as-built-air-and-deck.md`;
  - follow-ups F-158 (`06_docs/follow-ups.md:90`) and F-73 (`:145`);
  - 0.19.5 D-13, D-23 and A-23, and 0.19.6 `rulings.md` and its required reading.
- **Then the code:**
  - `app/arbiter.go` (whole), `app/mastercontrol.go` (whole);
  - `app/executors.go` (the Speak, Duck and Restore executors), `app/severe_read.go`,
    `app/read_script.go`, `app/pump.go` (dispatch and the lane);
  - `app/radio.go` (the deck as the arbiter's voice), `app/maintrack.go`, `app/bedrelay.go`;
  - `platform/lineup/bed.go` (`givingWay`, `giveOrTakeBack`), `director.go` (`settle`), `air.go`,
    `power.go`, `lineup.go`;
  - `domains/radio/player/engine.go` (`Suppress`, `Restore`, `giveWayLocked`, `watch`).
- **The pins:**
  - every test file that reaches the arbiter or MasterControl, from the shared graph's
    `file-edges.tsv`;
  - the mutants anchored in each file (`grep` over `06_docs/mutants/*.py`) and their verdicts in
    `06_docs/mutants/verdicts/since-v0.19.5.log`.
- **Nothing was run.** No test, benchmark or app.

## Findings

### RM-1 — What the arbiter still decides today, item by item against the disposition sheet

**Claim.** The sheet's row for the arbiter is *Absorb*, and the architecture made it precise:
*"decision half absorbed; effector half stays"* (`director-architecture.md` §§230-278, approach
C). The effector half has moved. Of the decision half, only the duck for the alert rail over the
programme has moved. Everything else is still decided inside `app/arbiter.go`, under its own lock.

**Evidence.**

| Piece | Who decides today | Where | Sheet |
|---|---|---|---|
| Duck, restore, pause or resume the line, discard a held line, stop the line, cue and clear the band (the effector half) | MasterControl | `app/mastercontrol.go:321-363`, `:422-457`, `:463-493` | Absorb (effector stays): **done** (T2.3) |
| Duck for the rail over a carrying bed or a main-track report | the Director: `givingWay` → `Duck`/`Restore` effects | `platform/lineup/bed.go:355-389`, `:392-402`; `director.go:859` | Absorb: **done** (D-82, D-139) |
| Class preemption: a takeover suspends a read | the arbiter, `admit` | `app/arbiter.go:316-324` | Absorb: **not moved** |
| Same-class queue (highest class first, then arrival order) | the arbiter, `first` | `app/arbiter.go:640-651` | Absorb: **not moved** |
| Resuming a suspended job, and which one | the arbiter, `settle` and `innermostResumable` | `app/arbiter.go:415-456`, `:491-502` | Absorb: **not moved** |
| The listener's pause and resume (MVS-D-74); a paused read holds nothing | the arbiter, `pauseRead`, `resumeRead`, `settle` | `app/arbiter.go:510-627`, `:457-467` | not on the sheet (built later): **arbiter** |
| Per-sequence duck: dip at admission, re-dip on resume, take back when idle | the arbiter → `mc.giveWay`/`takeBack` | `app/arbiter.go:345`, `:452`, `:466` | Absorb: **not moved** |
| When the bed may come back after the Director's own `Restore` | the arbiter, `releaseBed` → `settle` | `app/arbiter.go:675-683`; `app/executors.go:300-301` | Absorb: **not moved** (the lock order forces it; RM-5) |
| Cutting short the read on the air when its audio dies (FR-9) | the arbiter, `abandonOnAir` | `app/arbiter.go:396-409`; wired at `app/dashboard.go:558` | not on the sheet: **arbiter** |
| Dip or hold, from the SOURCE KIND, re-read every 50 ms | the engine | `domains/radio/player/engine.go:544-552`, `:615` | **Instruct**: holds as ruled, and nothing in D-2 touches it |
| `startSynth` and the fallback that reports up | the deck, directly; the report is dormant (`WATCHPOST_MAINTRACK` defaults to off) | `app/radio.go:618-660`; `app/maintrack.go:48-54` | **Instruct**: F-158 says the P3(d) ruling is owed |
| `startTakeover` and `capBurst` | the Director (the burst is handed to it) | `app/ticker.go:313-330` | Absorb: **done** (T3.10b) |

Two pieces are not decided anywhere else, and they are worth stating because they bound D-2:

- **Rendering the attention tone and the words at priority.** `synth.WithPriority`
  (`app/arbiter.go:277`) puts every arbiter job ahead of the broadcast's read-ahead.
- **Whether the visualiser follows the read.** `vizFor` (`:68`) does not follow a takeover.

**Consequence.** Most of the "decision half" the 0.14.0 ruling meant is still in `app`. This lens
keeps it out of 0.19.6, so the split moves it whole; it does not shrink in this release.

**Recommendation.** The package map should carry this table as the arbiter's current
responsibilities, so that what the arbiter decides is written down, not re-derived (principle 2).

**Confidence.** High.

### RM-2 — The arbiter's decision half exists because of one narration the Director does not schedule

**Claim.** The arbiter has two classes, and each has exactly one producer:

- `narrateBreaking` is the Director's `Speak` on the rail (`app/executors.go:408-409`);
- `narrateRead` is the `[space]`/`[w]` severe-event read (`app/severe_read.go:264`).

So "class preemption" today means one thing: **a card the Director scheduled outranks the one
narration it does not schedule.** The lineup has a slot for that read (`SevereRead`,
`platform/lineup/card.go:105`, `:171`), but its executor declines it by name: *"read by the severe
window's own reader, not the schedule"* (`app/executors.go:360`).

**Evidence.**
- `grep` finds no other `Run` call in production code (`app/executors.go:409`,
  `app/severe_read.go:264`).
- Within a class, the queue is rarely deep:
  - The Director keeps one card on the air per track.
  - `Toggle` cancels the old read before starting a new one, and `Read` does nothing while one is
    busy (`app/severe_read.go:93-110`, `:124-130`).
  - So more than one live job per class happens only while a cancelled job is still unwinding
    (medium confidence: inferred from the two callers, not measured).

**Consequence.** Before D-2 can make suspension "state inside the Director's step", the severe read
needs a home in the line-up. Neither track fits:

- **The main track:** `advances(MainTrack)` needs `power == Running && air == AirProgramme`
  (`platform/lineup/power.go:96`), so Observer never advances it.
- **The rail:** the rail must *drain*, and nothing but an operator empties it (0.16.0 required
  reading, "the alert never waits"). A read that `[esc]` cancels does not belong there.
- **A third track** is warned against in the type's own comment (`platform/lineup/lineup.go:8-11`).

The listener's pause is *"a fourth condition the `Power` enum does not yet express"* (FR-5.6), and
F-73 gives it to *"the release that builds P5's pause"* (`06_docs/follow-ups.md:145`). That is a
feature to design, not a re-architecture.

**Recommendation.** Do not start D-2 in 0.19.6. Record that D-2's first step is a home for the
severe-event read in the line-up, and that this step is the HUM LEAD's (product model and UX).

**Confidence.** High.

### RM-3 — "The Director owns all ducking" changes what is heard, in both modes

**Claim.** Today the arbiter's per-sequence duck is the **only** duck in three cases:

1. a rail card over the Observer monitor;
2. a rail card over the on-air bed relay that is not carrying;
3. the severe-event read.

The Director's `givingWay` covers only a rail over a carrying bed or a main-track report
(`platform/lineup/bed.go:369-389`), and that narrowness is pinned. For the Director to own all
ducking, `givingWay` must widen, and the widening is audible.

**Evidence.**
- **Observer.** `carries` is set only by `CutOver` (`bed.go:330-336`) and cleared by a fall-through
  (`bed.go:134-136`). Observer has no `CutOver`, so the Director never emits `Duck` there.
- **Broadcaster on air with the bed relay sounding.** `GoOnAir` plays the bed through
  `lp.playBed` → `tuneResolved` (`app/mastercontrol.go:138`, `app/bedrelay.go:230-236`), which
  sets no `carries`. So a hazard between main-track cards is ducked by the arbiter alone.
- **The pins that hold today's shape:**
  - `TestNothingDucksWhenTheBedIsNotCarrying` (`platform/lineup/duck_test.go:110-121`);
  - the unbracketed half of `TestTheBedIsDippedOnceForAWholeDrain`
    (`app/executors_test.go:850-871`): `duck,aside:first,restore,duck,aside:second,restore`,
    *"Observer must sound exactly as it did"*;
  - `TestArbiterDucksOnceAndRestoresOnce` (`app/arbiter_test.go:67`).
- **Mutants:** mE0, mE1, mE4, mE5 and mE7 on `app/mastercontrol.go`; mN4 on `bed.go`; mE3 and mM2
  on `app/executors.go`. mE4 and mN4 are CAUGHT in the baseline sweep
  (`since-v0.19.5.log`, entries 933 and 981).
- **The timing differs.** The Director's rising edge is at **admission** (`Arrived → duck · build ·
  publish`, `duck_test.go:76-78`); the arbiter dips when the `Speak` executor's job is admitted,
  just before the tone (`app/arbiter.go:340-346`). The falling edges differ too:
  - **Director:** at the last `Finished` of the drain, once the rail is empty.
  - **Arbiter:** when the job is released and nothing is waiting.
- The size of the gap between the two rising edges is UNVERIFIED. A takeover's `BuildCard`
  composes from events the producer already holds (`app/executors.go:316-330`), so the gap is a
  pump round trip rather than PD-2's 1.03 s, but it was not measured.

**Consequence.** Principle 4 is not met as byte- and timing-identical:

- two bursts back to back in Observer become one dip instead of two;
- the dip moves earlier, by the steps between admission and the read;
- the pins above have to be **re-authored**, not edited (role-model.md, T3.10a: *"Re-authoring a
  safety pin is not a rename"*).

The change might be judged an improvement (MVS-D-67 asks for one dip per drain). But "changes" is
not "regresses" only if the HUM LEAD rules the new sound as the requirement.

None of this touches the give-way rule. The Director would still say only "something speaks over
the broadcast", and `giveWayLocked` would still choose dip or hold from the source kind
(`engine.go:544-552`). A give-way keyed on the rail is never in question.

**Recommendation.** Do not move it in 0.19.6. If D-2 is scheduled later, put the Observer
one-dip-per-drain question to the HUM LEAD as its own ruling, with the trace re-run through the
executors as `director-build-log.md:2137-2160` did.

**Confidence.** High on the mechanism. The timing gap's size is UNVERIFIED.

### RM-4 — F-158: the arbiter covers for a Director `Restore` under a reading hazard, and the test is still unwritten

**Claim.** If the operator cuts back (`CutOver{ToBed:false}`) while a hazard is on the rail, the
Director emits `Restore`:

- `onCutOver` → `settle` → `giveOrTakeBack` (`bed.go:330-336`, `:392-402`);
- `givingWay` is now false (no carry, no report reading).

The bed stays down only because `releaseBed` defers to the arbiter's `settle`, which returns early
while a job is on the air (`app/arbiter.go:415-418`, `:675-683`). F-158 says the test is owed
*"before the direct path retires"* (`06_docs/follow-ups.md:90`). No such test exists:

- the only `CutOver{ToBed: false}` steps are in `platform/lineup/cutover_test.go:81` and
  `refence_test.go:149`, and neither asserts the duck;
- `app/bedrelay_kept_test.go:67` calls `CutBed(false)` without a hazard.

**Evidence.** As above. mE6 (`releaseBed` takes the bed back directly) is caught, by
`TestTheRailNeverLiftsTheBedOverALiveRead` (`app/executors_test.go:897-935`). That test covers a
`[space]` read, not the operator's cut-back.

**Consequence.** The first D-2 step to remove the arbiter's bed-return decision would lift the bed
under a tornado read, and no test would fail for the cut-back path. Today, through the whole chain,
a hazard reading keeps the bed down; the record has said so since 0.16.0 REVIEW, but nothing
asserts it.

**Recommendation.** **In 0.19.6, slice 1.** Write F-158's test at `app` level. A hazard on the
rail's air, a pressed cut-back, and the `Restore` performed through the executors must leave
`mc.givenWay()` true until the hazard's job ends. Under FULL TDD, show it failing under a new
mutant in which the `Restore` executor lifts the bed regardless (`x.mc.unhold(); x.mc.takeBack()`
in place of `releaseBed`). The behaviour does not change, so it is byte-identical. F-158 keeps its
ruling question (when the direct path and the switch retire); only its test half closes.

**Confidence.** High.

### RM-5 — The guarantees that matter are lock properties a pure step cannot hold

**Claim.** "Dissolve into the Director's step" can only ever move the **policy**. These guarantees
are atomicity under `arbiter.mu`, and they stay in an effector whatever D-2 becomes:

- a line starts under the lock, so a takeover admitted meanwhile finds it in flight and pauses it
  (`awaitAir`, `app/arbiter.go:232-250`);
- the suspension and `holdLine` happen in the same critical section as the admission (`:316-324`);
- the lock order `arbiter.mu → mc.mu` runs through `settle` → `takeBack`
  (`app/mastercontrol.go:390-406`, `app/arbiter.go:662-683`).

**Evidence.**
- **The pins:**
  - `TestALineNeverStartsUnderATakeover` (`app/arbiter_test.go:294`);
  - `TestArbiterALineRenderedUnderSuspensionWaitsForTheAir` (`:123`);
  - `TestATakeoversToneNeverSoundsOverALiveRead` (`app/takeover_read_test.go:28`);
  - `TestTheRailNeverLiftsTheBedOverALiveRead` with mE6;
  - F-D5's `TestTheBedIsNeverLiftedBetweenTheDipAndTheClaim` with mE7;
  - `TestTheArbiterIsRaceFreeWithoutAWrapperLock` (`app/arbiter_race_test.go:35`).
- **The pump** hands effects to workers and orders only the shared-output lane
  (`app/pump.go:155-176`, `:192-196`). A suspension the Director decided would reach the line in
  flight asynchronously, a step later, and *"the Director is open-loop about audio"* (0.15.0
  completion spike).
- **`TestNothingOutranksATakeover`** (`app/ticker_test.go:810`) protects `holdRest`'s
  wall-time-versus-air-time assumption (`app/read_script.go:274-283`). That assumption lives in
  the Reader, on top of the arbiter's `speaker`.

**Consequence.** Even done in full, D-2 leaves a mechanism in `app`: the speaker's gate on the air,
the suspension stack and the voice. Read correctly, the end state is approach C (a smaller arbiter
merged into MasterControl or the Reader), not "the arbiter is gone". A plan that promises its
deletion would be the "half that never moved" again.

**Recommendation.** Record it in the package map: D-2's end state keeps an air-gate effector; only
the policy (who outranks whom, when a held read returns, when the bed may lift) can become state in
the Director.

**Confidence.** High.

### RM-6 — D-2 grows the closed effect set and needs rulings already owed elsewhere

**Claim.** In the Director's step, suspension, the listener's pause and the severe-event read
would need new events (a read requested, paused, resumed or cancelled by the listener) and likely
new effects (suspending or resuming the line in flight). The effect set is closed and *"adding an
effect is a deliberate change to this table"* (`director-architecture.md` §§232-246, PL-6).
Today's set is walked in full at `app/executors.go:281-315`.

**Evidence.** As above. The rulings this depends on are open:

- F-158: P3(d), whether the switch and the direct path retire;
- F-73: P5's pause;
- MVS-D-74: the listener's pause, in the arbiter today.

**Consequence.** These are HUM LEAD rulings on behaviour and on the product model. Under D-4
every fork is the HUM LEAD's. Folding them into a re-architecting release would put product design
on the critical path of a split whose principle is "no behaviour changes".

**Recommendation.** Hold D-2 for the release that builds P5's pause (F-73's own "fixed by"). For
the record, the order that release would follow:

1. F-158's ruling (P3(d));
2. a home for the severe-event read in the line-up, with the listener's pause (F-73, MVS-D-74),
   and its vocabulary added to PL-6;
3. `givingWay` widened, with the Observer ruling from RM-3, and the duck_test and executors_test
   pins re-authored;
4. the arbiter's `giveWay`/`takeBack`/`releaseBed` and MasterControl's `held` retired, leaving
   `Duck`/`Restore` as the only path to `Suppress`/`Restore`;
5. the arbiter reduced to the air-gate effector (RM-5) and renamed for what is left.

Each step changes behaviour, so each needs its own ruling and its own pins first.

**Confidence.** High.

### RM-7 — D-2 is separable from the split, if the voice cluster moves whole

**Claim.** The split's `app` map does not depend on D-2's outcome, because D-2 changes the
cluster's insides and `platform/lineup`, not package edges. But the cluster cannot be cut:

- **The two halves call each other.** `arbiter.go` calls MasterControl's methods
  (12 uses, `file-edges.tsv`), and `mastercontrol.go` calls the `narrationVoice` interface declared
  in `arbiter.go` (8 uses). Two packages would form an import cycle.
- **A lock order crosses the pair:** `releaseBed` exists only for it (`app/arbiter.go:662-683`).
- **MasterControl, a role, is reached only through the arbiter**, which is not a role:
  `lp.arbiter.mc.MoveCard`, `DropCard` and `RequestCard` (`app/pool.go:92`, `:99`, `:490`);
  `nar.mc` (`app/ticker.go:166`); `.mc` from `app/airscope.go` and `app/schedule.go`.
- **The Reader runs on the arbiter.** `read_script.go` uses the arbiter's `speaker` (23 uses).
- **`narrationVoice` has only unexported methods**, implemented by `radioDeck`
  (`app/radio.go:908-997`). If the deck and the cluster land in different packages, the interface
  must be exported or adapted.

**Evidence.**
- 28 test files make 546 uses of `arbiter.go`, `mastercontrol.go` and `read_script.go`
  (`file-edges.tsv`); `severe_read_test.go` alone makes 179 into `arbiter.go`.
- Mutants anchored by file:

  | File | Mutants |
  |---|---|
  | `app/arbiter.go` | 11 (mE6, mJ0, mJ1, mK2, mK5, mK6, mL0 to mL4) |
  | `app/mastercontrol.go` | 9 |
  | `app/read_script.go` | 9 |
  | `app/severe_read.go` | 7 |

- The P10 mirror (`06_docs/p10-ledger.md`) has no row keyed on any of these files (grep). The
  declaration set (`app/testdata/declset.txt`) names their symbols and is re-captured on any move
  (TG).

**Consequence.** Split first, D-2 later is the order that holds D-1:

- the moves stay byte-identical, and the mutants are re-pointed once;
- D-2 is then designed against packages that will not move under it.

Doing D-2 first would reshape the cluster before the map could be drawn around it.

**Recommendation.**
- **In 0.19.6:**
  - The package map keeps `{arbiter, speaker, clip, narrationClass, narrationVoice,
    mastercontrol, readScript and its helpers}` together, with the deck or beside it.
  - If a boundary is drawn between the deck and the cluster, AS and GP rule the exported shape of
    the voice interface.
- **Not in 0.19.6:** reaching MasterControl through `livePipelines` directly instead of through
  `arbiter.mc`. It is a small API clean-up but changes a wiring path the 0.16.0 required reading
  draws; offer it to the HUM LEAD only as an A-row or a fork, not silently with the move.

**Confidence.** High. The cycle and the lock order were read; the graph supplied the counts.

### RM-8 — The vocabulary still calls the arbiter "the Director" in places, and one Reader comment is stale

**Claim.** 0.19.5 D-23(c) renamed the lower-case prose. The capitalised uses that mean the arbiter
survived:

- `app/arbiter.go:128` (`newDirector builds…`), `:161`, `:252`, `:274`, `:563`;
- `app/dashboard.go:548` (`buildDirector builds…`, over `buildArbiter`) and `:557`.

Separately, `app/read_script.go:5` says *"ONE READER, TWO CALLERS. The live takeover and the
Director's Speak executor"*. There is one caller (`app/executors.go:410`), because the live
takeover *is* the Director's `Speak` since T3.10b (`app/ticker.go:313`).

**Evidence.** As cited (grep, case-insensitive, over the four files).

**Consequence.** Principle 2: this is exactly the confusion D-13 was ruled to end. A junior reading
`arbiter.go:563` (*"EVERY LIVE READ THE DIRECTOR IS HOLDING"*) is told the role model's Director
holds reads, which it does not.

**Recommendation.** **In 0.19.6, slice 3**, with the doc pass of whichever batch moves these files:
the comment wording and the two stale function names in comments (an A-row, 0.19.5 D-13's class).
No identifier changes. Touches RC and JN.

**Confidence.** High.

### RM-9 — STOP versus STANDBY, and the alert rail's drain, are not in the arbiter and are not at risk from the split

**Claim.** The arbiter is indifferent to the power and the air. It performs whatever `Run` is
handed. STANDBY holds the rail in the Director (`advances`, `platform/lineup/power.go:96`), and
Observer's STOP leaves it reading. MasterControl's `declare` silences the programme and stops the
bed on standby (`app/mastercontrol.go:298-308`). The rail drains in the Director's schedule.

**Evidence.**
- `TestOnlyOneProgrammeCanEverAdvance` (`app/air_test.go:40`).
- The ten refused seams, `NarrateEvent` and `EndEventRead` among them, are guarded at the caller
  (as-built §4; `app/air_boundary_test.go`), so in Broadcaster the severe-event read never reaches
  the arbiter.

**Consequence.** A byte-identical move of the cluster cannot change STOP, STANDBY or the drain.
D-2 would: the severe-event read, once a card, would have to answer to `advances`, so its
behaviour under STOP and STANDBY becomes a ruling (today it is governed by the surface guard,
not by the power).

**Recommendation.** None for 0.19.6 beyond keeping the air-boundary test whole with the move (RX).
Add it to D-2's later ruling list.

**Confidence.** High.

### RM-10 — Set aside: a pure "policy core" for the arbiter inside `app`

**Claim.** The arbiter's policy could be pulled into a pure value type called under `d.mu`, with
the same lock and the same call order, so it would be byte- and timing-identical:

- `admit`'s preemption rule;
- `first`, `innermostResumable`, and `settle`'s promotion and take-back guard.

It would make the decision half testable as data, a step toward D-2 without the Director.

**Evidence.** The policy reads job pointers, the condition variable and `ctx.Err()` through `live`
(`app/arbiter.go:561`). A pure core would take liveness as an input. That is feasible, but it
re-points all 11 `arbiter.go` mutants and adds a type that is neither a role nor the end state.

**Consequence.** Principle 2's gain is doubtful (a fourth "settle"-shaped thing beside two already
named in `docs/where-things-happen.md:125`), and it builds an intermediate state the record warns
against.

**Recommendation.** Not in 0.19.6. It is listed so that D-7's reconciliation sees it was weighed.

**Confidence.** Medium.

## Recommended 0.19.6 slices (all hold D-1; none changes behaviour)

1. **F-158's test** (RM-4).
   - **Pins:** a new app-level test, seen failing under a new mutant at the `Restore` executor.
   - **Evidence that it holds D-1:** no production line changes.
2. **The voice cluster moves whole** (RM-7).
   - **Pins:** all 28 test files that reach the cluster move with it, or stay white-box beside it;
     the 36 mutants anchored in the four files are re-pointed and re-vetted; the baseline sweep is
     the judge (A-3).
   - **Evidence:** byte-identical, the race test green, and the declaration set re-captured.
3. **The vocabulary correction** (RM-8).
   - **Pins:** none needed, comments only; the fast docs lane does not apply, because these are
     `.go` files.
4. **The package map records** RM-1's table and RM-5's end-state statement, so a later D-2 starts
   from the record.

D-2 proper, as RM-6's five steps, goes to a later release with F-73.

## Open questions for the HUM LEAD

1. **Scope.** Is D-2 deferred to the release that builds P5's pause (F-73)? 0.19.6 would carry only
   slices 1-4, because every step of D-2 changes what is heard (RM-3, RM-6).
2. **Behaviour, when D-2 is scheduled.** Should Observer's alerts duck once per *drain*, as
   Broadcaster's programme does (MVS-D-67), or once per *card*, as today (RM-3)? Either answer is a
   ruling, not a regression.
3. **The product model, when D-2 is scheduled.** Where does the severe-event read live in the
   line-up, and what do STOP and STANDBY do to it (RM-2, RM-9)?

## Touches other lenses

- **AS:** the voice cluster's placement, and the exported shape of `narrationVoice` if the deck
  and the cluster are separated (RM-7).
- **RC:** the F-158 and F-73 rows, the stale vocabulary (RM-8), and the D-13 and D-23 lineage.
- **RX:** the duck and suspension pins and their mutants in RM-3, RM-4 and RM-5; whether the
  severe-read tests move with the cluster.
- **GP:** an exported interface whose implementation lives in another package (RM-7).
- **TG:** the declaration set and mutant re-pointing for the four files (RM-7).
- **JN:** "the Director" meaning two things in `arbiter.go`'s comments (RM-8).
