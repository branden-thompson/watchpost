# D-82 — THE AIR IS PER LANE, AND THE RAIL INTERRUPTS THE PROGRAMME

**HUM LEAD, 2026-09-11:**

> *"breaking alerts need to interrupt reports; Pause and resume approved; if that means fourth power
> variable, that's fine — 'it needs to work' — we can always come back later and look for performance
> or code structure optimizations; but this is NOT a license to do overtly silly things like blatant
> DRY violations, etc."*

**Approved: pause-and-resume, per D-24.**  The cheap alternative — drop the report and re-read it from
the top — was offered and not taken.

---

## Half of it was already built, and that is the headline

`admit` calls `mc.giveWay()` when an audible sequence takes the air; that reaches
`engine.Suppress()`; and `giveWayLocked` reads the SOURCE KIND every 50 ms:

- a **relay** DIPS — live radio plays on under the alert;
- a **rendered report** HOLDS — the player pauses and resumes where it stopped.

So D-24 — *"we can PAUSE the read, let the alert rail drain … then resume the read at normal
volume"* — has been implemented in the audio layer since 0.13.0.  **No fourth `Power` value was
needed, and none was added.**  What was missing was permission from the schedule.

**This is the third time this release that the answer was already in the tree.**  The pattern is
consistent enough to be a rule now: measure the existing mechanism before designing a new one.

---

## D-82 — `Lineup.OnAir` takes a lane

`OnAir()` held *"at most one card holds the air"* **across both tracks**, and `airOnce` asked it
BEFORE choosing a card.  So a rail card could not take the air while a report held it, and a hazard
waited out the weather — as long as one location report.

It was the right rule while one lane could speak.  It is now three lines:

| | before | after |
|---|---|---|
| `Lineup.OnAir` | one card, anywhere | one card, **on a lane** |
| `airOnce` | busy-check before `Next` | busy-check after `Next`, about that card's lane |
| `Next` | *(the doc claimed the drain rule; nothing enforced it)* | a rail card still READING stops the walk |

**TWO CARDS ON THE AIR IS NOT TWO VOICES.**  The rail speaks over the programme and the programme
holds underneath.  Two on ONE lane would still be two voices, and that is what the invariant now
says.

### The rule `Next` was documented to have and did not

*"THE ALERT RAIL DRAINS FIRST (DR-3). When it holds anything, those cards are read first and in
order; normal programming resumes only when it is dry."*  That paragraph has been on `Next` since
0.14.0.  Nothing in it enforced the last clause — the Director's *"is anything on the air
anywhere"* guard did, **by accident**, and that guard is exactly what had to go.

Two tests caught it within a minute of the change, and one of them was a fixture that had been
relying on the accident to age a card into staleness.  **Fourth instance this release of "a comment
is not a fix"**, and the first where the comment was describing an emergent property rather than a
known defect.

---

## The pump's half, and the schedule's half is worthless without it

`Holds(Speak)` claimed `TheBed` for every read, so every read rode the pump's **one lane**.  That is
right for the rail — its cards are read one after another and its duck must not be overtaken — and
wrong for a report, whose read BLOCKS for as long as the words take.  The hazard the Director had
just admitted could not be **asked for** until the weather finished: the operator would see the
takeover on the console and hear it minutes later.

**MEASURED, not argued.**  `TestAHazardsWordsDoNotQueueBehindAReportThatIsReading` was written
first, watched fail with the old rule, and passes with the new one.

So: **the band and the bed are the RAIL's outputs.**  `CueTicker` and `ReleaseTicker` carry the lane
for the reason `Slot` does (BD-8), and a report's run holds only its own card's order.  This is not
an exemption — it is the truth about what a report touches.

### F-71 closes as a consequence

`clearBand` carried this defect in an eight-line comment for two releases, with its trigger named
exactly: *"the moment P4 gives the band a second writer it is a live defect."*  What arrived instead
was a second card on the air.  A report's exit would clear the callout for a tornado warning still
being read over it.

It is fixed **where that comment said it had to be** — in the caller, from a fact on the effect, not
inside mastercontrol, which *"could not see the schedule that decides it."*

---

## The duck follows the programme

MVS-D-67 is *"one duck per RAIL DRAIN, never per card"* — a rail of two cards that dipped, lifted and
dipped again between them was MEASURED.  Its one condition was `bed.carries`, because the bed was the
only thing a hazard could be over.

A report can be underneath now.  The arbiter gives way per SEQUENCE and takes it back the moment
nothing is waiting, so between two hazards the engine would un-hold the report for the few
milliseconds the schedule takes to dispatch the next one — and **un-holding a paused report is not a
volume bounce, it is a fragment of a word.**

`givingWay()` asks whether the rail holds anything and whether there is a programme underneath it:
the bed carrying, or a report reading.

**THIS DOES NOT CONTRADICT D-32** (*"the priority ducks the bed only"*).  The Director says only that
the rail is speaking over the programme; the ENGINE picks dip-or-hold from the source kind, every
50 ms.  D-32 answered "what gets ducked" when the main track had no audio of its own; D-24 answers
the other half.  One predicate, because one `Suppress` already serves both.

---

## What was NOT built, deliberately

- **No fourth `Power` value.**  FR-5.6 anticipated one; the engine's give-way turned out to express
  the same thing, and a state nobody reads is a state that can disagree with the audio.
- **No `Suspended` card state and no resume path.**  The card never leaves the air, so there is no
  second `Speak` to emit and nothing to re-derive.  The worker performing the read is still blocked
  on the engine the whole time; the words simply stop and start.
- **Nothing in `app/mainread.go` changed at all.**

---

## Recorded

- **F-93 closes.**
- **F-71 closes.**
- `duck_test.go`'s "the ordering rule is unreachable" paragraph rested on main/bed exclusivity, which
  this relaxes.  **Re-checked rather than assumed**: both new ways a cue could fall beside a falling
  edge are closed by the schedule, and the comment now says which.
