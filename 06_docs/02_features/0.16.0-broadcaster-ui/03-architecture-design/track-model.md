---
title: "0.16.0 — the track model: foundation data and state for the three lanes"
date: 2026-09-09
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "BOTH DECISIONS RULED 2026-09-09.  Ruling 1 changed the shape from a derived predicate to INTENT, and that closes a real gap — see below."
---

# What this batch is

**Foundation data and state management for the tracks, before P4.**  All of it is `platform/lineup` and
all of it is pure — which is the point: P3's four blockers were every one of them in the join between a
pure state machine and an audio device, and three of the four gates this batch implies need no audio
device at all.

**What it must deliver**, each already ruled:

| | Ruled in |
|---|---|
| a **paused main track**, separately representable from STANDBY | **FR-5.6** |
| **main and bed are mutually exclusive**, in the MODEL and not only in the engine | **D-11**, **FR-4.2**, **D-32** |
| the **duck decision** as a pure function: a priority card on the air over a live bed | **D-32** |
| the **discard pile**: capped, trapdoor, Director-written, NOT a track | **D-35** |

---

## What is already settled, so it is not re-opened

**`Power` gains no fourth value.**  FR-5.6's exit criterion is that a paused main track and STANDBY are
*"separately representable and separately asserted"*, and two values on one enum are not separate.  The
diagnosis in `rulings-d11-d18.md` — *"a fourth condition the `Power` enum does not yet express"* — names
what is missing, not where to put it.

**The pause is not its own flag either.**  FR-4.2: *"The operator can cut the main track over to the
bed.  The main track then PAUSES."*  The pause is the CONSEQUENCE of the bed holding the programme, so
one piece of state carries both and there is no pair that can disagree (D-1).

---

## RULED — DECISION 1: a field on the bed, and the DIRECTOR sets it

**The state is one bit and its home is the argument.**

| | Shape | For | Against |
|---|---|---|---|
| **1a** | a field on `bed` — `d.bed.<name>` | it is a property OF the bed, beside `ref`, `live` and `since`; the struct's own comment is already *"what the broadcast rides on"* | the bed is *"deliberately not a track"*, and this makes it the thing that decides whether a TRACK advances |
| **1b** | a field on `Director` — `d.<name>` | the question is "which lane holds the programme", which is the Director's, not the bed's | a second place that describes the bed's role, next to `d.bed` |

**Naming is the live risk, not the placement.**  This project has now had **seven** defects from name
collisions, and the obvious words are all taken: `Holds` is the resource function, `held` is
mastercontrol's, `Programme` is an event, and `carrier` collides with the ON-AIR boundary language in
FR-5.5 (*"never that the antenna is radiating"*).  **Candidates that collide with nothing today:**
`d.bed.carries`, `d.bed.owns`, `d.onBed`.

> *"Field on the bed is fine, though I suspect the DIRECTOR is the 'setter' of that field (it says when
> the bed is active or not right now), while masterControl tells everyone 'air is clear to stream that
> bed to Audio out'."*

**RULED: 1a, named `carries`, and the Director is its only writer** — which it already is by
construction, since `bed` is a field of `Director` and DR-1 gives the lineup one writer.

### The ruling closed a gap I had not seen, and it is worth naming

**My own draft had `carries` as a DERIVED predicate** — `b.live && b.ref != ""`, "a relay is playing" —
which needs no new state and satisfies D-1 perfectly.  **It is also wrong, and the HUM LEAD's framing is
what exposes it.**

`live` is OBSERVED: `Tuned` is reported when audio actually plays, deliberately, because *"a resolve and
a connect can take seconds, and charging those to the listener's turn would cut every one short."*  So
between the operator pressing cut-over and the relay connecting, a derived `carries` is FALSE — and the
main track would advance a card into the gap the cut-over just created.

**`carries` is INTENT, not observation.**  The Director sets it when it moves the programme to the bed
and clears it when it moves back; MasterControl separately says whether the air is clear to stream.  The
bed already keeps `asked` and `askedAt` for exactly this distinction — *"the tune the Director has issued
and not yet seen land"* — so intent-beside-observation is the pattern the struct already uses.

| | Means | Set by | Drives |
|---|---|---|---|
| `carries` | **the Director has put the programme on the bed** | the Director, on a cut-over | whether the main track advances |
| `live` | **a relay is actually playing** | the deck, at `Tuned` | the dwell |

---

## RULED — DECISION 2: the model describes reality, and the split lands as early as it can

**This one is not cosmetic.**  Today `bed` carries EITHER a relay OR the synthesised rotation, and
`live` is what tells them apart — *"a relay, which dwells; the synth broadcast does not"*.

**Under the three-lane model the synth rotation is the MAIN TRACK.**  So `bed` describing it is the old
model's vocabulary surviving inside the new one, and it is exactly the kind of stale carrier this
release has already been bitten by twice.

| | Shape | Consequence |
|---|---|---|
| **2a** | **the bed means the RELAY, and nothing else.**  `live` retires; a synth read is main-track state | the honest model, and it makes `d.bed.carries` unambiguous.  **`Tuned{Ref, Live}`, `Ended`, `dwellElapsed` and `advanceBed` all need re-reading** — `Ended` exists because *"the synthesised broadcast ends on its own"*, which under 2a is a main-track card finishing |
| **2b** | **the bed keeps both**, and `carries` means "the bed holds the programme, whatever medium" | smaller change, and the model keeps a word that means two things — which is what the three-lane vocabulary was meant to end |

> *"I agree — we should ensure [the model] actually describes reality when possible.  But I think we
> should separate that as soon as possible to reduce blast radius."*

**RULED: 2a, and DECISION 1 is what makes "as soon as possible" mean NOW rather than later.**

`carries` is relay-only **from birth**, because the only thing that sets it is a cut-over and a cut-over
is to a relay.  So the separation is done in this batch and needs no rework: **`carries` is correct today
AND after the main track owns its audio.**

**What is left for later is a DELETION, not a redefinition.**  `live` keeps doing its one job — the
dwell — with a dated obligation: **it retires in the batch that gives the main track its own audio**,
because until then a synth read genuinely is the engine's main source and a model that denied it would
be lying about the code.

**Blast radius today: zero.**  Nothing emits a cut-over yet, so `carries` is always false and every
existing path behaves exactly as it does now.  That is the parity claim this batch opens with, and it is
the T3.2a discipline that worked in 0.14.0.

---

## The shapes (Tier B — signatures only, bodies are `// TBFI in BUILD`)

```go
// platform/lineup/bed.go
// carries says the BED holds the programme, so the main track is paused
// (FR-4.2, D-32).  ONE CARRIER: the pause is not a second flag.
// <name per DECISION 1>

// platform/lineup/power.go
// advances gains the second question.  Unchanged for the rail (DR-3).
func (d Director) advances(t Track) bool

// platform/lineup/director.go — the operator's cut-over, both directions.
// CutOver says the operator moved the programme between the lanes.
type CutOver struct { isEvent; ToBed bool }

// The duck, as a PURE decision (D-32): a priority card on the air, over a
// live bed, and nothing else.  Emitted from settle, paired.
func (d Director) givingWay() bool

// platform/lineup/discard.go — new (D-35).
// discardPile is a bounded stack: the newest on top, the oldest through the
// trapdoor.  NOT a track — `held()` counts tracks, and a parked card would
// mean the schedule never reads as stopped and the fault window never fires.
type discardPile struct { /* TBFI */ }

const discardDepth = 5

func (p discardPile) push(c Card) discardPile
func (p discardPile) restore() (Card, discardPile, bool)
```

---

## The gates this implies, and what each needs

| Gate | Audio device? |
|---|---|
| a cut-over to the bed pauses the main track, and the rail still drains | **no** |
| a cut-over back resumes it, and no card was lost | **no** |
| STANDBY and a paused main track are separately representable and separately asserted (FR-5.6) | **no** |
| a priority card over a LIVE bed emits `Duck`; over the main track it emits nothing | **no** |
| every `Duck` is followed by exactly one `Restore` | **no** |
| the discard pile drops the oldest through the trapdoor, and never grows | **no** |
| a restored card is a NEW card, because a revived one can be read twice | **no** |

**Seven gates, no audio device.**  That is the argument for doing this batch before P4.


---

# Built: the duck, and one rule that is enforced by something other than itself

**`givingWay()` is two questions and D-32 is that there is only one duck.**  Five plants, four caught at
once.  The fifth is the interesting one.

## The duck-before-the-cue rule has no reachable violation today

The effect set states it — *"a duck that landed after the read had started would be the duck-lift bug in
a new costume"* — and a plant that moves the duck after the cue **SURVIVES**.  Measuring the sequence
says why, rather than guessing:

```
Arrived  -> duck    · build   · publish
Built    -> cue     · speak   · publish
Finished -> release · restore · publish
```

**The duck's edges never fall in the same step as a cue.**  The rising edge is at admission, when the
card has no words and nothing can air.  The falling edge is at the last `Finished` — and while the bed
carries, **exclusivity means no main-track card can take the air** to be cued alongside it.

**So the ordering is enforced by exclusivity, not by the order of that list.**  A check asserting the
position would be the vacuous class this project has already shipped five of, so there is none; the
reasoning is recorded instead, in the test and here.

**IT BECOMES REACHABLE IF EXCLUSIVITY IS EVER RELAXED.**  Whoever allows a main-track card to air over a
live bed — the exception D-33 deliberately refused — re-opens this, and then the position needs a check.


---

# Built: the discard pile, and the line between a DROP and a FAILURE

**Six plants, five caught at once; the sixth was a bad plant of mine and was re-run properly.**

## It is not a track, and the assertion says why in the words that matter

`stopped()` is `held() == 0` and `held()` counts every card on every track, so a discarded card parked
in one would mean **the schedule never reads as stopped and the relay-fault window could never fire**.
The plant that makes `held()` count the pile is CAUGHT by a test whose name is the rule.

## The pile has a production writer TODAY, so it is not another owed row

**The staleness drop.**  PD-3 removes a card whose data has aged past the window, and the reason is
sharper than "the observation is old" — the window exists **to stop the station asserting something
untrue**.  That is a deliberate removal, so it belongs on the operator's pile.  **Before this the card
simply vanished** and nothing could say what had been taken away or why.

## A FAILED card is not a DROPPED card

**This is the line, and getting it wrong would have made the feature useless.**  A routed decline is the
schedule routing AROUND a fault (DR-21) — the producer offers the alert again and nobody chose anything.
Putting those on the undo pile would fill it with things the operator never did, **one per rotation turn
on some paths**, and an undo buffer full of noise is one nobody reaches for under pressure.

So the hook is at the staleness drop and at the operator's own DROP when it arrives — **not** at `leave`,
which would have caught every failure uniformly and looked tidier.

## The fixture had to drive the real path, and the first version did not

Staleness is *"a question about a card that is about to be READ"*, so it fires only when the air frees
and the next card has been standing by too long.  **The first test queued a report and ticked**, which
never reaches the check: the report takes the air immediately, and an on-air card is not a candidate.

The fixture now holds the air with a hazard while the report ages behind it, then finishes the hazard —
which is the sequence a real station produces, and the 0.15.0 build log's own lesson about **driving
through the seam rather than past it**.


---

# Built: P4's pure half — operator intent, and FR-3.3's own mutant

**Nine plants.  Five caught at once; four survived and every one pointed at a test that should have
existed.**  One of the four was **FR-3.7's own stated exit**, which I had not written.

## FR-3.3 is satisfied, by its own named mutant

> *"`Lineup.Set` refuses to reorder by design, so a promote routed through it would silently no-op.
> Exit: a mutant that makes promote update display state without reordering is CAUGHT."*

**Planted exactly that** — the promote routed through `Set` — and it is **CAUGHT**.  Every assertion in
the batch is against what `Next()` walks, never a field a console renders, which is what makes the
mutant visible at all.

## The four that survived, and what each was hiding

| Plant | Why it survived | The test that now exists |
|---|---|---|
| the on-air lock removed | `Next()` skips an on-air card either way, so the WALK cannot see it — and the harm is positional: the schedule describing a queue place for a card that has left the queue | asserts the card's POSITION, deliberately against the rule used everywhere else |
| the undo leaves its pile entry | restoring twice is refused by the identity check anyway, so nothing broke — but the console keeps offering an undo for a card already back | `TestRestoringSpendsThePileEntry` |
| every slot restores to the main track | **nothing restored a TAKEOVER** | `TestARestoredTakeoverGoesBackOnTheRail` — **FR-3.7's exit, in its own words: "asserted for an ACTIVE warning specifically, which is the case that matters"** |
| an on-air card can be dropped | nothing tried | `TestTheCardOnTheAirCannotBeDropped` — a drop mid-read strands the band's callout, which is DR-24's original defect arriving through the operator's control |

**The third is the one worth the batch.**  A restored warning going back to the programme instead of the
priority rail would read behind every weather report ahead of it, and the requirement had named that
exact case as its exit while the test did not exist.

## The gate closed an obligation and opened three, on the same run

`Origin.FromOperator` gained its writer — `onRestored` attributes the card to the human who put it back
— so its ledger row went **STALE** and the build stayed red until it was deleted.  It had existed for
two releases with nothing ever constructing it.

**And `Moved`, `Dropped` and `Restored` are ratified as owed to P4's UI half**, because the pure half
deliberately does not invent the CONFIRM that FR-3.7 requires: a confirm is a console control, and
inventing one here would be the surface deciding the operator's interaction from underneath.
