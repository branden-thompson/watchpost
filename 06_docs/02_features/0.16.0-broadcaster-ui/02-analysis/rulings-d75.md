# D-75 — THE RAIL RE-SCOPES WHEN THE AIR MOVES

**Asked by the HUM LEAD, 2026-09-11:** *"Does the alert rail correctly filter / expand itself based on
which mode (Observer / Broadcaster) is active today with these changes?"*

**The honest answer was half**, and the half that did not work was found by measuring rather than by
reading:

```
under the WIDE fence the rail holds 1
after the fence NARROWS the rail holds 2: [burst:far burst:near]
```

`far` is a hazard a hundred miles out, admitted under the listener's 150-mile filter.  Narrowing to
the station's 25-mile service area left it on the rail — and it would still have been read.

## What already worked, and why it was not enough

D-73 made the fence follow the surface, and both readers ask one function: `fence()`, which decides
the rail's ORDER, and `scopeToRadius`, which decides what reaches the rail at all.  A swap nudges a
cycle, so the tape re-scopes at once.

**But the Director learned a fence only on an ARRIVAL.**  `Arrived` carries `t.fence()`; nothing else
did.  So everything already admitted went on being read under the fence that admitted it, and the
window — between an alert being admitted and being read — is exactly when an operator swaps surfaces
to see what is happening.

## Held, not dropped — and it is the only answer that keeps both rules

Three options went to the HUM LEAD; he took the third.

| | |
|---|---|
| leave it | DR-3 stands, the window is narrow, and he would occasionally hear an out-of-area alert |
| drop on re-fence | matches his sentence exactly and **overrides DR-3** — *"nothing admitted is dropped unread"* |
| **hold, don't drop** | the card waits for a surface whose fence admits it.  **Both rules stay true.** |

So `Aired` carries the fence, `refence()` marks each rail card against it, and `Lineup.Next()` skips
what is held.

**IT RUNS BOTH WAYS**, which is the "expand itself" half of the question and the half that is easy to
leave out: widening releases what a narrower fence was holding.  A rail that only ever held would go
quiet and stay quiet.

## Four decisions inside it

**The card remembers its ARRIVALS, not a copy of their geometry.**  `Fence.Admits` takes an `Arrival`
and is the one owner of what "inside the fence" means; a parallel struct of coordinates would be a
second place for that rule to live, and the two would disagree the day the zone-only exception moved.
Bounded by `Settings.Max` — a card holds at most the alerts it reads.

**ANY, not ALL.**  A burst is one card carrying several hazards.  Holding it because a companion alert
was farther out would silence a hazard in the operator's own town.

**SKIPPED at the selection, not refused at the air.**  Refusing it at `airOnce` would let a held card
block every admissible card behind it.

**The fence is ASKED, not passed.**  `GoOnAir` is reached through `tty.Station`, whose signature is
the console's contract and has no business carrying a fence — and a fence passed by every caller is a
fence every caller could get wrong.  `mastercontrol` asks `tickerDeck.fence()`, which is the same
translation every arrival already carries, so the fence that re-tests a card and the fence that
admitted it cannot be two readings of one setting.

## Seven plants, six caught — and the seventh is a tripwire

`z4` — "a hazard already being read is cut off mid-sentence" — **SURVIVES BY DESIGN**, and it is the
D-42 shape stated again.  `airOnce` returns early while the air is busy and a card that has taken the
air is never offered twice, so marking it would change nothing anyone reads.  **The rule is held by a
DIFFERENT rule**, which is precisely why it is written down: it vanishes silently the day the other
one moves, and the day it does, a hazard gets cut off mid-sentence.
