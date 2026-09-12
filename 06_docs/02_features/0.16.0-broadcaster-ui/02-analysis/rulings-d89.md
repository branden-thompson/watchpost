# MVS-D-89 — the LIVE slot's empty state

**Status:** BUILT, 2026-09-11.  Closes **D-84 point 4**, the one part of that ruling left undesigned.
**Ruled by:** HUM LEAD, 2026-09-11.

---

## THE RULING, VERBATIM

> "empty state needs to be a grey box with a centered text of:
>
> NO REPORTS READ OR ACTIVE IN STANDBY MODE"

D-84 had opened the question and deliberately left it open:

> "LIVE should remain EMPTY — we'll need to design an empty-state for that slot before when the
> Station is in STANDBY (DEAD AIR) mode."

---

## WHAT IT REPLACED, AND THE DEFECT THAT WAS HIDING THERE

The slot was handed a `lineup.Card{}` and drawn through the ordinary card renderer.  **A zero
`lineup.Card` has a zero `Slot`, and the zero `Slot` IS `LocationReport`** — so the empty box came out
titled:

```
┃   LOCATION REPORT                                           •STANDARD•  [0] ┃
```

A report that does not exist, graded `STANDARD`, with a chip offering to open it.  `[0]` opened
nothing, because `cardDetail` correctly refuses an undecided slot — so this was **F-97 in miniature**,
in the very release that closed F-97: a control drawn on a surface with nothing behind it.

Found by rendering the standby console while implementing the ruling, not by a test.

---

## WHAT IT DRAWS NOW

```
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃                                                                             ┃
┃                                                                             ┃
┃                                                                             ┃
┃                                                                             ┃
┃                  NO REPORTS READ OR ACTIVE IN STANDBY MODE                  ┃
┃                                                                             ┃
┃                                                                             ┃
┃                                                                             ┃
┃                                                                             ┃
┃                                                                             ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

**No title, no badge, no handle.**  There is no report in that slot to name, to grade, or to open.

**Centred in BOTH axes.**  A line pinned to the top of a box this empty reads as a heading for
contents that are not there.

**The full height of a read card.**  The box is the same shape whether the station is on the air or
not, so going on air does not shunt the running order up and back down — at exactly the moment the
operator is watching it.  It is the same reasoning `readBody`'s fixed height already carries.

---

## THE GREY IS A TOKEN

`CardEmptyBG` — `bc.card.empty.bg` — new, and defined in all thirteen themes.

**Grey because grey claims nothing.**  A slot painted in the card family reads as a card the operator
cannot make out; a slot painted in the rail's red says the station is live.  An empty slot is neither
a card nor a state.

**It is its own token and not `GroupSectionBG`**, which is the nearest grey already in the set.  That
token means "a section header on Observer", and borrowing it would mean the console's empty slot moved
whenever somebody retuned Observer's bands.  One meaning, one token.

**The Quattro themes derive it** — `mix(p.fg, p.darkBg, 0.22)` — from the palette's own ink and
ground rather than from a hue, which is what makes it grey in every theme without anybody choosing
seven more colours.

**The light theme goes DARKER, not lighter.**  On a light ground the dormant thing is the one that
recedes, and recede means grey-toward-ink.

**The AA lift was measured before it was written**, per the standing rule and the D-86 scar:
registering a new ground widens `CardText`'s pair set, and `withAA` lifts a foreground until it reads
on *every* ground it is registered against.  `CardText` is **unchanged in all thirteen themes** — the
new ground is inside the range its existing ink already clears.

---

## THE REGRESSION THE TESTS CAUGHT

The first draft gave the notice to **every read slot**, because the code it replaced did.  A station
that had just opened then reported *"NO REPORTS READ OR ACTIVE IN STANDBY MODE"* in the UP NEXT slot —
**the slot the Composer is working on right now**.

D-84 puts the Composer on UP NEXT precisely so the line is ready at `SHIFT+ENTER`.  Reporting an
absence where there is work in progress is the opposite of what that ruling asks for, and the
**shimmer** is what says the Director is still choosing.

So the notice is the **LIVE slot's alone**, identified as position 0 — which `liveOffset` exists to
guarantee: on standby the line-up is drawn from UP NEXT down precisely so nothing can be in slot 0
until the operator goes on air.

---

## FR-2.4, AND WHAT IT NOW MEANS

Two tests asserted **"the ten slots are addressable 0-9 (FR-2.4)"**, and both failed on this change.
They have been amended rather than deleted, and the amendment is recorded here because it narrows a
stated requirement:

> **FR-2.4's ten addresses are the ten a card can be IN.**  Slot 0 becomes one the moment the operator
> goes on air; while the station is at rest it is the one slot nothing can be put into, because
> `SHIFT+ENTER` fills it rather than any per-slot control.

Both tests now also assert the **negative** — that `[0]` is absent at rest — so the ruling cannot be
reverted silently in either direction.

**This narrows a requirement, and the HUM LEAD may want it the other way.**  The alternative is to
keep `[0]` on the standby box as a pure position marker that opens nothing.  That was not chosen,
because a chip that opens nothing is the exact defect reported hours earlier as F-97 — but it is a UX
call and it is recorded as one.

---

## WHAT WAS EXTRACTED

`cardLane.shell` — a box of the card's shape around arbitrary contents, painted on one ground.
Extracted **at the second caller**, which is the standing modularity rule: the standby box needed the
same six lines of border, rule and tint, and a second copy would be two places for a corner to drift.
What differs between a card and an empty slot is now the CONTENTS, and only the contents.
