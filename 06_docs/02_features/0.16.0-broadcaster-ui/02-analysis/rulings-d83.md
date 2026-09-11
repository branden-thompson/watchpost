# D-83 — THE LIVE CARD SHOWS THE READ, AND THE WORDS WERE ALREADY THERE

**HUM LEAD, D-68:** *"the LIVE CARD should be bigger to support showing at least 'most' the script
being played … UP NEXT should also be bigger."*

**HUM LEAD, 2026-09-11:** *"The 5 line preview is fine — the operator should be able to inspect the
full text of the report by keying the number position of either the live card `[0]` or the UP Next
`[1]` card."*

---

## F-84's premise was half wrong, and the fix was a quarter of the size

The row read:

> *"`BuildCard` produces `Built{ID, Script}` inside the executors; `Card` carries `Subject` and
> `Headline` and no words … So the LIVE card's window onto the read has nothing to put in it."*

True when it was written.  **Overtaken at T3.8**, when `Card.Script` was added for the Reader —
`Projection` returns whole cards and `Publish` hands the console the lineup, so the words have been
arriving for two releases.  `readBody` emitted five empty strings under a comment still explaining
that they could not.

**So this was a RENDERING job, not a wiring one.**  Fifth "a comment is not a fix" this release, and
the first where the comment was simply overtaken rather than wrong when written — which is the
harder kind to notice, because it was accurate on the day it was committed.

---

## What it draws

`scriptWindow` wraps the card's parts to the box's interior less the card's own inset, and draws
`bcReadLines` of them.  Each part starts its own line: the script's shape is what the listener hears
— head, lines, tail — and running two parts together would show the operator a different arrangement
from the one going out.

**The height never moves.**  `readBody` has required that since D-68 — a card that grew when its
words landed would shift every card below it at the moment the operator is reading one, or typing a
slot number at one — and the window is the first thing that could have broken it.

**UP NEXT gets it too, and that falls out rather than being added.**  D-68 gives slots 0 and 1 the
tall box, and DR-7 composes a card's words AT STANDBY — so the operator reads one card ahead of the
one on the air.  Nothing was written for that.

### The edge of the window says it is an edge

A five-line view of a twelve-line report reads as a short report unless it says otherwise — on a
station, the difference between *"that is all it says"* and *"that is all it fits"*.  The mark is the
glyph set's ellipsis, so it degrades to `...` with the rest of the console rather than inventing one.

**IT IS SET OFF FROM THE WORDS WHERE THERE IS ROOM**, and that is not cosmetic.  The mark means
*there is more below*, not *this line continues*.  The first draft set it flush in both cases and
produced

```
   Looking ahead, expect patchy fog after midnight and a high near sixty-eight tomorrow.…
```

which reads as a typo.  **Found by rendering the card and looking at it**, not by a test — and a line
that genuinely had to be CUT still takes the mark flush, because there it is true.

---

## One measurement owner

`cardLane.inner()`.  Two things measure the box's interior now — the border drawer and the window
that has to fit inside it — and `bandWidth` was this exact lesson at D-80: a fourth number agreeing
with three others by coincidence is what put the station band three cells over its frame.

---

## A plant survived, and it was the same shape F-84 itself had been in

`mN5` blanks `readBody`'s call to the window.  **Every test written for this batch stayed green**,
because all of them call `scriptWindow` directly — the wiring between the two was covered by nothing.

That is P-1 (*a seam a test cannot drive is not a covered seam*) and it is **precisely the state F-84
described for two releases**: a window built to the right size, with nothing putting words in it.
Closed with a test that drives the whole console and looks for the words on the screen.

---

## Not built, and filed

The window shows the OPENING of the read, not the five lines around where the voice currently is.
Following the read needs the read position published to the console, and nothing publishes it.  For
a report it is usually the same thing; for a long one the operator watches a static opening while the
voice is elsewhere.  **F-94.**

The HUM LEAD's answer to "where is the rest" is the card detail modal, keyed `[0]` / `[1]` — drawn
since D-52 and inert since (D-39, D-31).  That is the next thing this window needs.
