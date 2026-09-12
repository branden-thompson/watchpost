# MVS-D-88 — the card's own window

**Status:** BUILT, 2026-09-11.  Closes **F-97**.
**Raised by:** HUM LEAD, UAT 2026-09-11 — *"Pressing [1] doesn't open the details modal."*

---

## THE RULING

A slot's number opens that slot's card.  The window carries the card's status, the age of its data,
who will read it, who proposed it, **the whole manifest**, and **the whole read**.

The HUM LEAD stated the requirement twice, a month apart in the record and hours apart in UAT:

> "The operator should be able to inspect the full text of the report by keying the number position
> of either the live card `[0]` or the UP Next `[1]` card."

> "The full script on the top level card doesn't make sense when I can *drill down* to read the whole
> thing.  What DOES MAKE SENSE for the operator is to get a SUMMARY of what the report contains
> *before* it goes on air."

The second sentence is what made the first one load-bearing.  **D-87 turned the card into a manifest
on the strength of a drill-down that did not exist** — so for one release the console promised a
place to go and had nowhere to send anyone, and the words a station was about to say aloud were
reachable from no surface at all.

---

## WHERE THE WINDOW LIVES, AND WHY IT IS NOT THE CONSOLE'S

**It is a Dashboard window.**  The console builds the content; the Dashboard draws it.

That is D-56's split, stated for the diagnostics window and true here for the same reasons:

> "ONE WINDOW, NOT TWO.  The ctrl+d window is entirely Dashboard methods; giving the console its own
> would be a second injector UI, a second confirmation, and two places for the TEST EVENT wording to
> drift."

**Four gates hang off the Dashboard's window set**, and a console-private window would have been
outside every one of them on the day it shipped:

| Gate | What it measures | What it found here |
|---|---|---|
| `TestEveryLineOfEveryWindowIsReachableAtTheFloor` (FR-5) | every line reachable by keyboard at 80x24 | **the arrows did nothing** — see below |
| `TestEveryWindowClearsItsMargins` | three clear columns inside both borders | led at 2, trailed at 2 |
| `TestTheMemoKeyCoversEverythingTheFrameShows` (F-30) | frame differs ⇒ key differs | forced the generation counter |
| `TestASCIIFramesCarryNothingButASCII` (FR-8) | no glyph without an ASCII form | **found a design flaw** — see below |
| `Dashboard.modal` is a single value (Q6) | exclusivity by construction | free |

Three of the five reported a real defect within a minute of the window being registered.  That is the
argument for joining an existing set rather than starting a parallel one, and it is the same argument
`modalLines` already makes in its own comment.

---

## WHAT THE GATES FOUND

### 1.  A hand-written list of "the scrolling windows" (`nav.go`)

`handleNav` routes arrow keys by a **hand-written list of modals**.  A window absent from it does not
merely fail to scroll — its arrows fall through to the **table underneath**, which is precisely the
D-58 violation ("the window on top owns the keys") that the reachability gate was built to catch.

The card window was absent on the day it was written.  The gate reported four unreachable lines at
80x24; the probe showed the scroll pinned at 0 through ten keypresses.

`add` and `remove` are absent **on purpose** — their own keys walk `selected` — and that is now
written down beside the list, so the next person to read it does not have to re-derive which
omissions are deliberate.

### 2.  A body built in one glyph vocabulary and drawn in another

The first design handed the window a **finished `[]string`**, built with the console's `Opts`.  The
`--ascii` parity gate flips `cfg.ASCII` **after** the fixture is built — and the window came back
carrying `•` in its title and `↑↓` in its chips, neither with an ASCII form.

In production the two cannot disagree: `NewRouter` copies `cfg.ASCII` into the console once, and it
never changes.  **The flaw was unreachable and it was still a flaw**, and this codebase's standard is
to remove the class rather than to argue the class cannot be entered.

So the window holds a **renderer, not a snapshot**:

```go
cardID   string
cardRows func(render.Opts) (string, []string)
cardGen  int
```

Title and body are both asked for **at draw time, with the window's own Opts**.  There is no moment at
which the two vocabularies can differ, because there is no stored artefact in either.

**And the identity became the card's ID rather than its rendered title** — which is what `Card.ID` is
for ("ID addresses the card for the life of the lineup").  Matching on a rendered title would lose the
open card the moment anything about its appearance moved, including the very glyph switch that
started this.

### 3.  `bcDetailRoom` is derived, not chosen

The first draft laid the tables out at **78** — the manifest's own number, off the v2 reference.  The
margin survey reported the window running into its right border by exactly one column.

The right number is **77**, and it is arithmetic rather than taste: `85 - 2 - 3 - 3` — the window
class's floor width, less its two borders, less the three clear columns the survey requires inside
each.  Written down that way, it cannot be adjusted by feel.

---

## WHAT THE WINDOW REFRESHES, AND WHAT IT MUST NOT

**It re-hands on every update, and its generation moves only on a real change.**

A card the operator is reading **re-hydrates under them** — `RefreshAfter` is half of `StaleAfter`,
so the window is open across a refresh routinely.  A window that kept its first frame would show a
stale `DATA PULLED` stamp and a stale manifest on the one surface whose entire job is to be trusted
*before* something goes on the air.  That is F-30's freeze with a read in it.

So `refreshCardWindow` runs on the same cadence as the gain mirror, for the same stated reason, and
`showCard` compares before it bumps:

- **matched by ID**, so a card promoted from `[2]` to `[1]` keeps its window;
- **compared at one agreed `Opts`**, so the generation moves when the *report* changes and not when
  the *terminal* does;
- **unchanged ⇒ no bump**, so the memo still hits for as long as nothing moves.

---

## WHAT THE WINDOW SHOWS THAT THE CARD CANNOT

The card draws **four** manifest rows (`bcReadLines`) because four is what a location report has and
because a card's height is fixed — "the height is the same whether there is a script or not", or every
card below it moves when one report gains a source.

The window has a scroll rail and no such obligation.  **A fifth source is visible there and nowhere
else**, and the tests assert exactly that difference: a fixture of four sources could not tell the two
surfaces apart.

---

## WHAT IS NOT BUILT

- **The priority track has no handles.**  A takeover card draws no `Report Details` chip, so there is
  nothing to open.  Whether a hazard burst should be inspectable the same way is a HUM LEAD ruling
  and is not assumed here.
- **The window is read-only.**  No promote, no discard, no re-order from inside it.  D-39's original
  sketch had sticky controls; what those controls *are* is the line-up management ruling and has not
  been made.
