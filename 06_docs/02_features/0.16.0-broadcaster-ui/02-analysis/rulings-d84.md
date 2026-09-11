# D-84 — THE LINE-UP IS PLANNED BEFORE IT IS BROADCAST

**HUM LEAD, UAT 2026-09-11**, on being told the station had to go on air before the line-up would
fill:

> *"This is wrong, and it makes it impossible for the Human operator to preview and manage the lineup
> PRIOR to going on the air for the first time.  This MUST be fixed. …  being able to see, manage, and
> change the line up PRIOR to going on air is a fundamental requirement — otherwise the user might as
> well just use Observer."*

The expected behaviour, in his numbering:

1. entering Broadcaster puts the station in STANDBY (DEAD AIR);
2. Producers grab locations from the pool and propose reports;
3. the Director chooses and populates the LINE-UP;
4. **LIVE stays EMPTY** — and needs a designed empty state;
5. **UP NEXT gets the Composer**, because it is the first thing that goes on air when
   `SHIFT + ENTER` is pressed — *"and when the operator does, the line **should be ready to go** at
   that point."*

---

## DR-3 was not inverted, and my first draft said it was

Three places asked `advances(MainTrack)` before ACCEPTING a card, under the heading *"admission is a
promise to read (DR-3)"*.  I removed them and wrote "planning is not performing", which the HUM LEAD
pushed back on:

> *"I just saw 'admission is a promise to read' now is inverted — which seemed weird to me."*

**He was right, and the correction matters.**  The promise is intact: nothing admitted is dropped
unread.  What was wrong was the INFERENCE drawn from it — *"therefore do not admit while stopped."*
The line-up the operator builds on standby **is** the promise being kept; it goes to air when they
press the key.  Written the other way, a future reader could conclude that admitted cards may be
silently dropped, which would actually break DR-3.

> **Admission is a promise to read.  It is not a promise to read RIGHT NOW.**

**AND THE PACKAGE ALREADY SAID SO, one file along.**  `reconcileJoins`: *"IT IS NOT GATED ON
`advances`. These are consequences of cards the schedule has ALREADY admitted … A stopped station
still has a running order; it simply is not reading it."*  Admission now says the same thing, and
`advances` has ONE asker — `airOnce`, which is where reading happens.

---

## The defect the ruling creates, and the HUM LEAD named its fix

The Composer now works on standby.  A station can sit on standby for an hour.  At `StaleAfter`
(15 minutes) the prepared card is dropped as it takes the air (PD-3) and the listener's first words
are *"That report is out of date and has been dropped."*  **The operator sets up a line-up, goes for
coffee, presses the key, and the station opens by apologising.**

I had started building a refresh.  He arrived at the shape independently and stated it better:

> *"Do we need to 'toss' the card, or do we simply 're-hydrate' it?  It's still the same card, it's
> just like the composer 'filling it' for the first time, it's just stale. …  No need to 're-check
> admission' — it's already been admitted.  No need for the director to choose / move another card —
> it's still the same card … the order has been decided.  It just needs to have its data updated."*

So `refreshStandby` emits **the same `BuildCard` the first fill emits, for the same ID**.  No
proposal, no re-admission, no reorder, no movement in the line-up: `Built` comes home and
`WithScript` overwrites the words and the stamp in place.

**It lives at the bottom of `prepareNext`**, reached only when `toPrepare` finds nothing new to fill
— so the two reasons to ask the Composer have ONE owner, with an obvious precedence: a card with no
words at all comes before a card whose words have aged.

### `RefreshAfter = StaleAfter / 2`, derived rather than chosen

A rebuild is ~1.03 s of network, so half the window leaves 7.5 minutes of margin — four hundred
times the cost of the thing it races, which is what makes it impossible for a refresh to be the
reason a card goes stale.  Refreshing AT `StaleAfter` would race `takeTheAir`, which runs first in
the same settle and would drop the card on the very tick the refresh fell due.  It is derived from
`StaleAfter` rather than written beside it, so the two cannot drift.

### Three guards, and two of them were found by plants

- **The ask moves the stamp.**  The schedule ticks every second and a build takes a second or more;
  without it the same card is re-asked on every tick until its words come home — a build storm by
  construction.  `Built` overwrites the stamp with the true time when it lands.
- **Never while the rail holds anything.**  The merge property test caught this within a minute of
  the refresh being wired: `toPrepare` stops one card ahead, so a rail holding
  [standing-by, admitted] leaves the Composer idle — and the refresh would have spent that idle
  moment on a REPORT, putting the hazard's build behind it.  Property 5, *"the rail is prepared
  first, not merely aired first"*, exists because a plant that reversed it SURVIVED.
- **Only while the track cannot advance**, which bounds the new behaviour to the case that created
  it.  **The first test for this could not fail**, and a plant said so: one card on a running station
  is READ immediately, so there was no standing-by card left to refresh.  The test now puts one on
  the air and one behind it.

---

## LIVE is what is on the air, and on standby nothing is

`slotCard` applies a `liveOffset`: 1 while the station is not Running, 0 while it is.  The station
draws its line-up from UP NEXT down, with LIVE empty and waiting; the instant the operator goes on
air the head of the queue takes the air and everything moves up one — **the same movement D-40
already rules for a dropped card**, so the operator has seen it before.

**THE POWER DECIDES, NOT WHETHER A CARD HAPPENS TO BE PLAYING.**  A running station between two
reads — the next card still building — has nothing on the air for a moment, and keying the offset off
that would shunt every card down a row and back for the length of one build.  The power is stable;
"is something playing right now" is not.

---

## Still owed

**The LIVE slot's empty state is a blank box**, which is what the HUM LEAD flagged in point 4:
*"we'll need to design an empty-state for that slot."*  A blank box is at least not a false promise —
which is more than the eight slots below it managed before this batch, where `waiting for the
line-up …` promised cards that the schedule had deliberately decided not to produce.  **That second
problem disappears here**, because the line-up now genuinely fills: the shimmer became honest again
rather than being removed.
