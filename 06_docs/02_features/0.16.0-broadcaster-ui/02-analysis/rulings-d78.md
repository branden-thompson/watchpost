# D-78 — THE BED'S CONTROLS, AND THE EVENT THAT HAD NEVER FIRED

**Closes F-79** (the bed's state is not published) and the control half of **F-89**.

---

## `lineup.CutOver` had been modelled for a release and never once emitted

```
$ grep -rn 'CutOver' app/ modes/ --include='*.go' | grep -v _test
(nothing)
```

It is ruled at FR-4.2, D-11 and D-32 — *"the operator can cut the main track over to the bed; the main
track then pauses"* — and `bed.carries` governs a real gate in `advances(MainTrack)`.  **With no
producer, that flag was false for the life of every process, so the pause it governs had never
happened.**  `[b]` is its first caller.

**The wires ledger is what closed it, not me.**  `Event.CutOver` carried a ratified exemption saying
"the control that presses it is P4's"; the moment the event was wired the row went STALE and the gate
refused the build.  Removing it is the obligation being discharged, not a note being tidied.

## The bed said INACTIVE because nothing had ever told it otherwise

`Publish` carried the line-up and the power.  So the region D-62 built to answer *"what is on the
air"* answered two thirds of it, and the third was a constant that happened to be true at launch —
`(no relay tuned)`, `○ INACTIVE`.

The bed rides on the same effect now, for the reason the other two already do: **D-62 consolidated
all three into ONE region so the operator reads the air state in a glance**, and three facts drawn
together and published apart would undo that at the seam.

## `B` was already a door

The reference draws `[ B ] BED:`, and `B` is the Broadcaster swap's non-chord key — FR-1.6 requires
one, because an operator whose multiplexer eats `ctrl+b` would otherwise have no route to the console
at all.  **One key means one thing (D-56), so the bed takes `b`** and the chip reads `[ b ]`.

Stated rather than quietly re-bound: the mock is the record, and a deviation from it is a decision.

## Two publishers, one answer

The Director publishes what the bed is CARRYING; the selector publishes what the operator has
SELECTED, and those are different facts until they cut to it.  A selector that GUESSED at `Carrying`
would flicker the row between the two writers — so it is RECORDED from the Director (`noteBed`) and
never decided locally.

## And the selector walks the STATION's fence

Nearest first, wrapping both ways.  **It wraps because a selector that stopped at the ends leaves the
operator pressing a key that does nothing and wondering which of the two reasons it was** — and with
one relay in reach it stays on the one relay, which is the honest answer to a fence that reaches one
thing, and what D-77's reach advice warns about before they get here.

Tuning goes through `deck.tuneCallsign`, the same path the relay-fault window already answers with
(MVS-D-76), so there is no second place for the duck to be lifted.

## Seven plants, six caught

`b7 — the bed is published as nothing` **SURVIVED**.  The console's tests feed it a `BedMsg` and prove
it DRAWS one; nothing proved the schedule ever SENDS one, so deleting the bed from the effect left
every test green and the row a constant again — **which is the exact defect F-79 was filed for.**

That is the fourth vacuum this release, and the shape has not changed: the half of a seam that is easy
to reach gets asserted, and the half that carries the fact does not.
