# 0.16.0 rulings, D-59 … D-68

The layout phase and the UAT that followed it.  D-59 … D-65 were built and shipped with their
reasoning in the code and in the commits; they are stated here in short because a ruling that lives
only in a commit message is a ruling the next session rebuilds from the code.  D-66 and D-67 are
stated in full: they are the answers to the HUM LEAD's UAT of **2026-09-10**, and one of them is a
correctness defect that had been sitting under a comment for a release.

---

## The short ledger, D-59 … D-65

| # | Ruling |
|---|---|
| **D-59** | **ONE MASTHEAD, DRAWN BY BOTH SURFACES.**  *"why the masthead different than the Observer Masthead?"*  It should not have been — the console had dropped the version, invented a stamp and left out the API summary.  The ladders live in `masthead.go` and are called twice; the EDITION WORD and the console's identity row arrive as arguments, not as a second header. |
| **D-60** | **THE MAIN TRACK IS DRAWN AS NAMED REGIONS** — LIVE, UP NEXT, SCHEDULED, LINE UP — which is what the reference's left rail spells out.  A card carries no state strip of its own: the rail already says it, once per region rather than once per card. |
| **D-61** | **THE PRIORITY RAIL IS INVISIBLE UNTIL IT HAS SOMETHING.**  *"the PRIORITY Rail label ONLY shows up when a priority card 'sits on top' of the main rail — this gives the operator more space to view/manage the main rail during 'normal' operation."*  A `(clear)` row spends two rows of the running order saying a hazard is not happening, which is the state the station is in almost all of the time. |
| **D-62** | **THE BED MOVES INTO THE BROADCAST SECTION.**  *"the bed was yet another conveyer of the AIR STATE and we wanted to consolidate those … This way the ON AIR / STANDBY is all in one section — and user doesn't have to look to different parts of the UI to determine what is and is not ON AIR."* |
| **D-63** | **THE FRAME IS THE VIEWPORT, IN BOTH DIMENSIONS.**  It is padded to the terminal's height and width, because `render.Overlay` composites against the BASE — a ten-line frame in a seventy-four-line terminal pinned every window to the top rail, and a ragged right edge sent a centred window off the side. |
| **D-64** | **THE CONSOLE DRAWS ITS TEN SLOTS ALWAYS, SHIMMERING WHILE UNDECIDED.**  *"Broadcaster UI should adopt the same methodology as Observer — render the full Broadcaster Dashboard, including all 10 cards … Cards should 'shimmer' while the producers are proposing and the director is deciding."*  `(nothing scheduled)` was a dead end and is gone. |
| **D-65** | **EVERY CONTROL THE MASTHEAD ADVERTISES IS BOUND.**  *"None of the chip controls work — s / a / ? / q in broadcaster UI mode."*  They were printed and bound to nothing.  Forwarded to the Observer rather than rebuilt, and the fix GENERALISED D-58: the Router composites whichever window is open, not only DIAGNOSTICS. |

---

# D-66 — ONE MEASURE OF WIDTH, AND IT HAD TWO

**Ruled by the defect, 2026-09-10.**  The HUM LEAD's UAT listed six separate symptoms in the
masthead alone:

> 1A. `'WATCHPOS     '` — the rest of the title missing
> 1B. Top of the box draw missing
> 1C. Updated Missing
> 1D. API only shows ✔9 and missing the rest
> 1E. Chips dont render their bkg, and change color as the terminal window expands / shrinks — tells me something about coloring and tokens are broken in broadcaster ui
> 1F. GAIN/VOL control missing — only showing the left arrow

…plus *"Right hand lanes are off"* and *"Card chips not shown"*.  **Eight symptoms.  One cause.**

`render.TruncateCells` counted an escape sequence's characters as display cells.  The console clamps
every row of its frame through it; `render.TitleGradient` emits a truecolor escape **per rune**.  So a
150-cell masthead row was cut after ten visible characters and **through the middle of an escape** —
which the terminal printed as text, and which left the span open so it painted the padding a colour
that MOVED as the window resized.  Nothing was wrong with the tokens.  Nothing was wrong with the
chips.  One function measured bytes where `render.Width` has always measured cells.

## The part that matters more than the bug

**It was known.**  `platform/render/status_table.go` carried this comment:

> `splitCells`, not `TruncateCells`: the lines are STYLED, and `TruncateCells` counts an escape
> sequence's bytes as content and will cut through the middle of one — which the terminal then
> prints as text.

A correct diagnosis of a shared function, written at the ONE call site that had been bitten, and
routed around locally instead of fixed.  Every later caller inherited the defect and no comment
warned them, because the warning was in the caller that had already escaped.  **A comment is not a
fix**, and a known-wrong shared function with a local detour around it is precisely the shape D-56
exists to stop.

`TruncateCells` is now ANSI-aware — escapes cost no cells, a cut never lands inside one, and a span
still open at the cut is closed — and `status_table.go`'s detour is gone.  The fitting case still
returns the string itself and allocates nothing, so the frame budget did not move for it.

## What else was one line of the same ruling

| Symptom | Cause |
|---|---|
| chips have no background | the console **typed** its keys as text; `o.Controls`/`KeyCap` is the one thing that knows `[ X ]` is a chip control, and it was never called.  Observer's own control row now builds through the same function |
| right-hand lanes off | the console drew an inner wall at 144 **and** a scroll rail at 145.  The reference has ONE column: `│` on an ordinary row, the thumb standing where the bar was |
| no air between the sections | *"the sections of the line up are missing their blank row between the sections like the mocks."*  One blank row between REGIONS — never between cards, which are flush inside a region in the reference |

---

# D-67 — A CARD THAT FAILED SITS OUT BEFORE IT IS OFFERED AGAIN

**Ruled by the defect, 2026-09-10.**  The HUM LEAD:

> the lineup is FLYING through locations rapidly even on standby — so it seems like cards are
> constantly getting discarded and proposed / accepted

It is, and his reading of it is exactly right.  The loop, confirmed at the model level:

```
decline → Failed{Routed: true} → the Director discards the card → the step PUBLISHES
→ the publish executor asks the producer to top the line-up off → the producer offers the same
watchlist → ReadID gives the failed location the identity it had a microsecond ago
→ admit → build → the same fault → decline
```

at pump speed, for as long as the fault lasts.  **Every executor refusal reaches it**: a muted
listener, a report that would not compose, a script that rendered nothing to say.

The executor's own comment said the work *"will be offered again on the producer's next cycle"*.  The
intent was right and DR-21's route-around-the-fault is right.  What nobody noticed is that
**a publish IS a cycle**, so "next" meant "now".

## Where the limit lives, and why

**On the Director.**  It is a decision about the SCHEDULE — the Director already refuses a duplicate
identity and already weighs how long since a kind of card was read (D-48); *"this one just failed"* is
the same sort of fact.  A producer holding it would be a second author of what the line-up may
contain, which D-40 divided precisely to avoid.

**A rate limit, not a blacklist.**  The memory is a TIME, so a fault that clears does not silence a
location for the rest of the run.

**Bounded, for D-48's ruled reason** — *"read history is too overweight"* — a fixed ring of 32,
oldest evicted, one entry per ref.  It cannot grow; there is nothing to cap, own or clear.

## The number is the HUM LEAD's

`retryAfter` is **five minutes**, which is the dwell the station already uses for its rotation — a
pace this project has reasoned about once rather than a number invented here.  What the constant must
do is stop a busy loop, and any value above zero does that; what it trades is how quickly a TRANSIENT
fault recovers against how often a PERSISTENT one is retried for nothing.  **Flagged for ruling, not
presented as settled.**  The rest of the watchlist keeps filling the depth meanwhile, so a location
sitting out never leaves the line-up short.

---

## Ten plants, ten caught

`s1` escapes counted as cells again · `s2` a cut leaves the span open · `s3` the masthead types its
keys instead of capping them · `s4` the regions run together · `s5` the rail column loses its bar ·
`s6` the scroll rail moves back off the wall · `r1` the cool-off is not consulted · `r2` a routed
failure is never recorded · `r3` the cool-off never expires · `r4` the ring is unbounded.

---

# D-68 — THE CARDS THE OPERATOR READS FROM ARE NOT THE CARDS THEY ORDER

**HUM LEAD, UAT 2026-09-10**, with a reference mock drawn without the alert overlay to show it:

> the LIVE CARD should be bigger to support showing at least "most" the script being played — when
> it's on standby, like it is on first open/play, that should be blank, we should have an empty state
> for that live slot … UP next should also be bigger

**Two kinds of card, one drawer.**  A card the operator READS FROM needs the words on it: eleven
rows — border, title, the marks' row, a five-line window onto the script, a blank, the card's own
controls, border.  A card they are merely deciding the ORDER of needs its name and its handle, which
is four.  Everything else about them is identical, so `boxOf` takes the interior and `box` is the
flat case of it; a second box function would be a second place for the handle to drift.

**The empty state is EMPTY, and only where the promise would be false.**  A shimmer says the read is
coming.  Nothing is coming while the station is not on the air, so the LIVE and UP NEXT slots draw
their box with nothing in it at rest, and shimmer only once the station is running.  The slot is the
SAME HEIGHT either way, or the frame jumps under the operator at the moment they go on air.

**The window is built and there is nothing to put in it (F-84).**  `BuildCard` produces
`Built{ID, Script}` inside the executors; `Card` carries a headline and no words.  The geometry is
the half that could be built without a ruling; where the script rides is the half that needs one,
and it is worth ruling once for the bed's state (F-79) as well.

## The chrome the reference draws and the console did not

| | |
|---|---|
| **two blank rows above and below the whole frame** | *"Universal 2 line inset like Observer"* — the console had its masthead hard against the top of the terminal, the one place in the app that does not breathe |
| **a bare blank row between the station section and the running order** | annotated twice: *"Notice the blank line and how it separates the rail — this is intentional."*  It carries NO walls, because the two regions are closed boxes and the air between them belongs to neither |
| **the lane names itself, centred over the cards** | *"Notice the header line; this should be centered"* — it is the STANDARD lane, and the priority lane names itself the same way when it has something (D-61) |
| **a spacer row under the header** | *"Notice the spacer row"* — which is what a region gap already is, so it is the same row builder |
| **the scroll rail spans only what scrolls** | *"Notice the top of the scroll is here, and the left rail is separated."*  The two read cards are always the same two; a thumb beside them would say they move |

**ONE DEVIATION, STATED**: the reference draws the read cards five cells WIDER than the scheduled
ones, because it drops the scroll rail's columns entirely in that zone.  The console keeps ONE card
width for both and leaves the gutter blank instead — `cardBoxWidth` is D-51's single owner of that
arithmetic, and two card widths is the thing that seam exists to avoid.  Raised rather than assumed.

---

# D-69 — A REFUSED SWAP HAS TO SAY SO

> ctrl+o will soft lock randomly — so ctrl+o -> ctrl+b -> ctrl+o (doesn't work the 2nd time) …
> While radio is on standby I should flip back and forth easily.

**It is not a lock and it is not random.**  Visiting Observer TUNES, a tune tells the Director the
programme is RUNNING (which is mM3's own fix, and correct), so the station the operator left STOPPED
is ON AIR when they come back — and FR-1.4 refuses to let anyone walk away from a live console.  The
gate was right every time.

**What was missing is that `Router.refusal` was recorded and drawn nowhere**, under a comment saying
exactly why that must not happen: *"a refusal they cannot read is indistinguishable from a broken
control."*  Written down, not done — the same shape as D-66's, two files apart, in the same release.

It rides to the console the way the gain does — mirrored, never owned twice — and lands on the row
the reference reserves for it: *"<this then becomes a status message of something related to the
broadcast bar>"*.  **It dies with its reason**: the only refusal there is says the station is on the
air, so once it is not, the sentence is false and goes.

**THE DEEPER QUESTION IS STILL OPEN AND IT IS F-80's.**  Observer's playback and the console's ON AIR
are the same `Power`, so listening on one surface puts the station live on the other.  That may be
exactly right — they are one station — but it means the operator can be refused a swap they never
did anything to earn.  Ruling wanted.
