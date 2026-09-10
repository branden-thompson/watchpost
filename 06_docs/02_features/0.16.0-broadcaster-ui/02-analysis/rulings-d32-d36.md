---
title: "0.16.0 — rulings D-32 to D-36: the duck, the discard pile, and what [space] becomes"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "RULED in conversation, 2026-09-09, after the P3 revert.  Recorded here because a ruling that lives only in a transcript is a ruling the next session does not have."
---

# D-32 — ONE DUCK, AND IT IS THE PRIORITY RAIL OVER THE BED

> *"the main / bed logic described makes sense; only 1 can be active at a time, and the priority ducks
> the bed only."*

**Main and Bed are mutually exclusive.  A priority-rail card ducks the bed.  Nothing else ducks
anything.**

| What is playing | What arrives | What happens |
|---|---|---|
| the **bed** (a live relay) | a **priority** card | the bed **DUCKS** to 0.15 and plays on underneath |
| the **bed** | a **main-track** read | the bed **STOPS**.  Main replaces it — see D-33 |
| the **main track** | a **priority** card | **nothing ducks.**  The main track gives way (P5's paused main track) |
| the **main track** | another main-track card | it waits.  A card takes the air only when the air is free |

**Why the duck survives for exactly one case**, and it is the HUM LEAD's own reasoning from this
morning: *"the duck is important for the stream because that relay is fundamentally out of our control
(we can 'pause' the stream, only turn it on or off)."*  A duck lifts in 50 ms and the relay is where it
was.  Restarting a relay is a resolve and a connect, which the deck's own comment says *"can take
seconds"*.  **Putting that gap after every hazard is dead air at the moment the station has just said
something urgent.**

**Three things can trigger a duck today and only one of them should.**  That is what this ruling
removes.

**AND IT IS A DECISION THE PURE DIRECTOR CAN HOLD.**  Is a priority card on the air, and is the bed
live.  Both are answerable from state the Director already has, which makes the whole rule testable
with no audio device — the thing P3 could not do.

---

# D-33 — A CHOSEN READ REPLACES THE BED; IT DOES NOT PLAY OVER IT

> *"kill the duck entirely — if you choose a location with a synth read, you are choosing to listen to
> that read.  The Director can then determine, 'ah, this read is done, the user was listening to the
> relay before, so let's do a quick transition card' … and then resume the relay."*

**Scoped to the CHOSEN read** (D-32 keeps the duck for the priority rail).

**What it buys, and each item is a thing removed rather than added:**

- **No exception to main/bed exclusivity.**  The rule stays absolute.
- **The return transition becomes honest.**  Its one conditional clause turns on whether the programme
  kept running underneath; a stopped relay simply is not *"already in progress"*, so this is a clean
  third case rather than a clause that lies.
- **It sidesteps the staleness argument instead of working around it.**  The engine dips rather than
  pauses a relay because *"a paused relay resumes into audio that is minutes stale."*  A relay that
  STOPS and RESTARTS rejoins live.
- **The transition card covers the reconnect.**  The gap the resolve-and-connect costs is exactly what
  the card is filling, which is a card doing real work rather than decoration.

---

# D-34 — THE RETURN TRANSITION NAMES THE RELAY, AND ITS ANCHOR DIFFERS BY SURFACE

> *"we now return to `<Relay Name>` broadcasting on a frequency of `<frequency>` approximately
> `<distance>` from this station's location"*

> *"focused location in Observer, station location in Broadcaster.  That's fine."*

| Surface | The distance is measured from |
|---|---|
| **Observer** | the **focused location** |
| **Broadcaster** | the **station's tower location** (FR-6.4) |

**One script, one anchor that differs** — the same shape the existing return transition already uses,
where one script serves both modes and only a clause differs.  **Recorded now rather than discovered
when the line reads oddly**, because the tower location is a Broadcaster setting that does not exist in
Observer at all.

---

# D-35 — THE DISCARD PILE: A CAPPED STACK WITH A TRAPDOOR

> *"refused cards can pile up if we're not careful, since technically once it's refused, it's in a final
> state that is temporally unbounded.  We can create a Discard pile … the discard pile needs to act as a
> stack with a trapdoor — we cap the number of cards the discard pile can hold, and the oldest (at the
> bottom) fall through the trapdoor when another discarded card is set on top … perhaps 5."*

**And the Director is its writer**, per the same conversation: *"while anyone can PROPOSE a card, it's
the Director's job to say 'yep, it's in' or 'nope, it's out'."*  That is DR-1's one writer, applied to
refusal.

**THE CAP IS NOT OPTIONAL AND THE REASON IS MEASURED, NOT AESTHETIC.**  `stopped()` is
`lineup.held() == 0`, and `held()` counts every card in every track.  **A refused card parked in a track
means the schedule never reads as stopped and the relay-fault window can never fire.**  So the pile is
not a track, and the cap is what keeps it from becoming one.

**Precedent for the shape:** `bandRecord` is a bounded ring with a named cap whose comment says it
*"never grows with the day"*.  The cap here wants the same treatment — a stated reason rather than a
round number.

## The one collision, and how it resolves

The card model says, in as many words: *"REFUSED, DONE and DISCARDED lead nowhere: **a card that could
be revived is a card that can be read twice**."*

**An undo that revives the original card breaks that invariant.**  It reconciles with the HUM LEAD's own
earlier ruling — *"We can make **new** cards, or use a card to make a copy … but while it's alive its
origin should not change"* — so:

> **UNDO MINTS A NEW CARD from the discarded one's CONTENT, with a new identity.**  The pile holds
> content for restoration, not cards awaiting resurrection.

---

# D-36 — `[space]` BECOMES AN OPERATOR-INITIATED PROMOTE

> *"Could this be considered a 'special' (Observer only) kind of Human Operator initiated promote?
> Where it effectively: 1. Paused the current lineup — including the card being read; 2. Pushes down all
> items in the line by 1; 3. Begins the card read of the initiated Operator directive."*

**RULED, and it retires three open items rather than adding work.**

| It consumes | Which was |
|---|---|
| the **paused main track** | already required (**FR-5.6**), already missing, already P5's |
| the **promote** | already planned (**FR-3.3**, P4's reorder) |
| **`Origin.FromOperator`** | an unwired member with no writer — this is its writer |

**AND IT REMOVES THE SECOND PRODUCER OF SPEECH.**  `narrationClass` exists to referee speech attempts
from producers the Director does not own, and there is exactly one: the `[space]` read, which has **zero
references to `platform/lineup`**.  Make it a card and the arbiter has one producer, at which point the
Director's own track precedence IS the priority.

## What the implementation must honour, from what `[space]` does today

1. **It reads an EVENT in the severe window, not a location.**  The `SevereRead` slot exists and the
   schedule declines it today — *"read by the severe window's own reader, not the schedule"*.  **That
   decline retires**, and it is a change to a shipped path, so it carries its own care.
2. **A second press PAUSES and a third RESUMES.**  It does not stop.  So the toggle needs suspend and
   resume of an ON-AIR card, which is the capability that does not exist (P3's fourth blocker, the
   fourth `Power` condition).  **The model cannot be built before it.**
3. **A press on a DIFFERENT row cancels the old read at once** — *"the old read is stale the moment this
   one is asked for"*, and waiting was measured as *"up to two seconds of a key doing nothing"*.  Under
   the card model that is a drop plus a promote, which makes it **the discard pile's first customer**.
4. **Latency is not a risk here.**  An event read composes from a row already in memory (1–3 ms), not
   from the eleven network requests a location report costs.

---

## What these five rulings leave open

**`narrationClass.narrateRotation`'s disposition is now DECIDABLE but not yet DECIDED.**  With one
producer of speech the class has nothing left to arbitrate, and the header three lines above it already
says *"the broadcast itself (relay or synth) is not a narration: it is what gets ducked."*  Retirement
follows — but it follows from D-36 landing, and D-36 depends on P5's paused main track.  **It is not
ruled here, and it should not be assumed.**
