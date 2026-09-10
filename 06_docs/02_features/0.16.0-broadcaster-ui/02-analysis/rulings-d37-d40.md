---
title: "0.16.0 — rulings D-37 to D-40: the overlay stack, the discard modal, and who fills the lineup"
date: 2026-09-10
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "RULED in conversation, 2026-09-09/10, while reviewing the card-type mocks."
---

# D-37 — THE DISCARD PILE IS A MODAL, OPENED FROM THE BASE UI

> *"Discard Pile normally should not be visible to the operator, it's discarded — so it's probably going
> to be a modal that opens with something like `[shift+U]`, and the modal shows a list of the 5 most
> recent cards that were discarded, with an option to choose and restore.  The `[w]` modal might be the
> closest analog since we MAY have different categories of discards in the future."*

**Patterned on the severe window**, which already has the table-and-detail split that categories would
need, so the future case costs nothing now.

**AND IT IS REACHED FROM THE BASE UI ONLY.**

> *"We'll make the discard pile accessible from the base UI first — if we need to adjust, we'll do it
> later and pay that cost, but I would only want to do so when I have a real need and user-flow
> identified, otherwise we're just guessing."*

**That decides a real architecture question.**  `Dashboard.modal` is a SINGLE VALUE, not a stack, and
nesting today is a hand-rolled boolean per window.  A discard modal openable FROM a card modal would
need a return path and that is the rewiring; opened from the base UI, the current shape holds untouched.

---

# D-38 — FOUR LEVELS OF ELEVATION, AND THREE ALREADY EXIST

> *"the Broadcaster UI needs to support at least FOUR levels of overlay, in order of elevation (bottom
> to top): Base UI → Priority Track overlay (semi-persistent) → Detail / Operator Action Modals →
> Confirmation / Destructive Action Modals."*

**Checked rather than assumed, and the base UI already composites three of them.**  `view.go:19` builds
the frame, overlays the open window, then overlays the confirmation ON TOP OF THAT — and the comment
records why it was built that way:

> *"A second layer rather than a swapped body: the mock shows the red box ON TOP of the diagnostics
> window, and the window underneath is unchanged."*

| Level | Exists? | Where |
|---|---|---|
| Base UI | yes | `view.go` writeBody |
| **Priority track overlay** | **NO** | it is semi-persistent, takes input, and MOVES as it drains, so it belongs in the BODY render, not the overlay stack.  Nothing today overlays something INTERACTIVE onto the body |
| Detail / operator modals | yes | `render.Overlay(content, overlay, …)` |
| Confirmation modals | yes | a second `render.Overlay` over that |

**So the corner to avoid is not the overlay stack — it is the single-value modal**, and D-37 keeps us
out of it.

---

# D-39 — THE CARD DETAIL MODAL IS A CENTRED OVERLAY

> *"Card Detail modal should be a centered overlay modal like settings — the location details modal
> might be the closest analog because a report *could* have a lot of content once it hits the 'Up next'
> position and the composer merges the script with actual data.  So scroll, sticky controls, all of that
> needs to be there."*

The pinned-footer path (`modalSetup`, `modalDebug`, `modalRelayFault`) already lays out at its own box
width with the scroll following the focus, so scroll and sticky controls come from an existing owner
rather than a new one.

---

# D-40 — THE PRODUCER PROPOSES, THE DIRECTOR CHOOSES

**The gap this closes was found by walking a user flow, not by a gate.**  Only two things queue a
main-track card — the deck reporting a location needs a read, and the operator's undo — and **nothing
reads the track's depth.**  So the lineup holds about ONE card while the mock draws TEN.  0.14.0's role
model predicted it exactly: *"Nothing produces location reports or credits; the rotation does that
implicitly."*

> *"I'm fine with the producer 'topping off' the line-up.  It also means that they could in theory
> propose MULTIPLE cards — a location report for `<a>`, `<b>`, `<c>`, `<d>` — each being city/town/zips
> within the service radius.  At this point they're just templates with a location name so the cost is
> cheap, and it's the Producer's job to PROPOSE."*
>
> *"The DIRECTOR in that model, as the owner of the lineup, then is the one who gets to choose which card
> gets the slot.  That mirrors the analog station today, and it fulfills the charter of the Director
> being the executor of the Operator's will.  They ease the operational aspect by choosing, understanding
> what that choice means for the line-up ('Oh, this card now being in the line up means transition cards
> [invisible to the operator] need to bookend this card.') — and WHEN IN DOUBT may surface a choice modal
> to the Operator."*

**Three things this settles, and each maps onto a role that already exists:**

| | |
|---|---|
| **The Producer proposes, and may propose SEVERAL** | a proposal is cheap because it carries no words — DR-7 already says text materialises at standby, so a template with a location name is the whole cost |
| **The Director chooses which takes the slot** | it owns the lineup (DR-1), and choosing is arrangement, which is exactly what the role model gives it |
| **The Director knows what the choice COSTS** | *"transition cards need to bookend this card"* — the inter-card transition is its ONE additive act (role model), and MVS-D-80 already rules when one fires |

**The UX implication, and it is new work:** a **choice modal** — *"which of these do you want in the
line-up?  It will go in at slot [9]"* — which is a level-3 Detail/Operator Action modal under D-38.

**Open, and small:** a card the OPERATOR chose from a proposal is arguably `Origin.FromOperator` rather
than the Director's.  Not ruled; flagged so it is decided rather than defaulted.

---

# D-41 — `Origin.FromOperator` OUTRANKS

> *"That makes sense, and `Origin.fromOperator` automatically gets a higher rank."*

**Ruled in answer to the open item D-40 left**, which was whether a card the Operator picked out of a
choice modal is the Operator's or the Director's.  It is the Operator's, **and the origin then carries
rank** — which is the Director's charter reading on itself: a card the human chose outranks one the
Director chose on their behalf.

**SCOPE, stated so it is not over-applied:** this governs the **choice ranking** — which proposal takes
the slot.  It is NOT yet a ruling on where a RESTORED card lands in the running order (`onRestored`
appends today).  That is a separate UX question and is deliberately not defaulted here.

**Nothing is built for it yet, on purpose.**  `Proposal` carries a ref and a headline; no producer sets
an origin, because no producer can — the modal that would is UI work.  Adding the field now would be a
member of a closed set with no writer, which the wires gate exists to refuse and `AP-DEAD-01` names.

---

# D-42 — A TRANSITION IS NEVER THE OPERATOR'S

> *"transition cards are NEVER `Origin.fromOperator`."*

**BUILT, because it was one invariant and it was already reachable.**  `Card.check` now holds it, so
every door — `Propose`, `Queue` and `Set` — enforces it.

**It used to hold BY ACCIDENT**, which is the reason it was worth stating.  The only path that could
have constructed one is the operator's undo, and that failed for an unrelated reason: it drops the
words deliberately, and a wordless transition is refused by the guard above.  **A rule held by a
different rule** is the shape this package keeps having to un-split — it is how the duck came to be
lifted by one spelling of tune and not the other.

**The reason behind the ruling** is the role model's: the transition is the DIRECTOR'S one additive act.
The human asks for a report; the hand-off around it is the Director's consequence of that choice, never
the request itself.

---

# D-43 — TRANSITIONS ARE SOFT-ASSOCIATED, AND THEY MOVE WITH THEIR CARD

> *"they may need to have some kind of soft association at the Director level -> Location Report Card
> from Operator needs to be bookended by transition cards, those transitions cards 'move' with the
> operator's card in the event the operator then 'promotes' or 'quashes' that card up or down the
> rotation.  I don't know if the code accounts for this, but we it should be noted just in case we find
> out we have to build this later."*

**Checked, and the code does not account for it.**  Filed as **F-75** with the evidence:

- **Exactly ONE transition exists in production** — `stale.go:93`, the staleness drop's replacement
  read, `FromDirector`, fixed id, *"one at a time is all that can"* exist.
- **Nothing bookends anything.**  The Director's inter-card transition is not built beyond that case.
- **There is no companion concept at all.**  `Reorder` moves exactly one card and asserts the track
  length is unchanged; `Remove` takes exactly one; `discard` piles one.  Bookending transitions today
  would be stranded by the first promote — the report would move and its hand-offs would stay, reading
  a lead-in to something no longer next.

**NOT BUILT, per the ruling.**  What F-75 records for whoever builds it: a move, a quash and an undo are
**three doors onto one rule**, so DR-1 says the association gets ONE carrier that `Reorder`,
`onDropped` and the discard pile all read.

**And verifying it found a live defect** — F-74, fixed the same day.  See the follow-ups.

---

# D-44 — THE LINE-UP IS A PROJECTION OF THE SCHEDULE, AND IT IS OLDER THAN THIS RELEASE

> *"I thought we already built the distinction between the 'schedule' and the 'line-up' — a previous
> session was the one that highlighted that the line-up is the human-facing PROJECTION, while the
> schedule was the underlying thing all the other roles saw and used."*

**Correct, and it is on the record.**  `director-build-log.md:1945`, HUM LEAD 2026-09-05:

> *"The operator's view is a **PROJECTION** of the one Lineup, **not a second schedule**; the 'machinery
> tier' is the closed EFFECT SET, which is deliberately not schedule entries — putting them there would
> make the Director schedule its own work, which Approach C exists to prevent.  The one-card model
> **REDUCES the gap** between what the operator sees and what the machine holds, rather than creating
> it."*

**The reason no code carried it: it was the IDENTITY FUNCTION.**  One card per burst and nothing
invisible meant `Cards` already WAS the operator's view.  The Director's structural cards are the first
thing that makes it a real function.

**AND ONE HAS BEEN IN PRODUCTION SINCE 0.14.0.**  `director.go:749` queues the staleness notice — a
`Transition` — straight onto the main track, so a card the operator never asked for has been sitting in
their numbered running order the whole time.  That is what let this be built and tested against a real
card rather than an invented fixture.

## The vocabulary, because the Go type is named for the wrong half

| Call | Is | Read by |
|---|---|---|
| `Cards(t)` | **THE SCHEDULE** — every card, structural ones included | the Reader, the Composer, the staleness check |
| `Projection(t)` | **THE LINE-UP** — what the operator sees, numbers and addresses | the console, `Reorder`, the top-off's depth |

## Three things it fixes, one of them live

- **The console drew the schedule.**  `broadcaster.go` showed the staleness notice as a numbered slot.
- **`Moved` was a silent off-by-N waiting to happen.**  Its own comment says *"the console already knows
  every slot number it drew"* — those are LINE-UP numbers, and `Reorder` indexed the SCHEDULE.  Both are
  valid indices, so nothing would have errored: the operator moves a card and a different one moves.
  `scheduleIndex` is the one owner of the translation.
- **The depth was counted in the wrong space** — my own code from this morning.  Ten slots against a
  schedule counting structural cards tops off at about five real reports and leaves the rest empty.

## The debt this ruling creates, stated rather than hidden

**The 2026-09-05 rationale is that the projection SHRINKS the distance between the two.**  Every
structural card is a small payment against that, so `structural` is a **field on the slot registry**
rather than a property anything may claim: the set of cards the operator cannot see is CLOSED, and
adding to it is a deliberate act with a row to fill in.

**It also names a category that ruling did not contemplate.**  It said the machinery tier is the closed
EFFECT SET, *"deliberately not schedule entries"*.  A transition IS a schedule entry — it is read aloud,
it takes the air, DR-24 pairs its release — so it is a third thing: **machinery that must be performed.**
That is not a contradiction, but it is the reason the gap can now grow, and why it is bounded by a
registry field instead of a convention.

---

# D-45 — ON AIR IS LOCKED, AND ONLY STANDBY OR CATASTROPHE CHANGES IT

> *"we already agree that once a card is ON-AIR - it's locked from editing.  The ONLY thing that can
> change that would be a GO TO STANDBY from masterControl (Operator initiated) or some catastrophic
> error, which we purposefully can never account for and test."*

**This closes something the record left open.**  The same 2026-09-05 entry reads:

> *"**Raised, not yet ruled:** 'once a card is LIVE, only a takeover may interrupt it'.  The code does
> not express this — `Card.CanBecome` allows ON AIR → Discarded from any cause.  **Wants a decision when
> operator editing arrives.**"*

**Operator editing has arrived (P4), so the decision is due, and it is made.**

**Every caller already conforms**, checked rather than assumed:

| Writer of `Discarded` | On the air? | |
|---|---|---|
| `onDropped` | **refuses it explicitly** | ✓ |
| `silenceTheProgramme` (`power.go:173`) | takes it off | ✓ — **this IS the ruled exception** |
| `dropStale` | `firstStale` considers STANDBY only | ✓ |
| `onFailed` | the card did not happen | ✓ — the catastrophe bucket |

**What is NOT expressed is the type**, which is exactly what the 2026-09-05 note flagged: `CanBecome`
still permits ON AIR → Discarded from anywhere.  **Narrowing it would be wrong** — `silenceTheProgramme`
legitimately needs that edge — so the closure is a gate that walks the WRITERS, not a narrower type.

---

# D-46 — THE PROACTIVE REQUEST: `[shift+N]`, AND IT IS A REQUEST ORDER, NOT A CARD

> *"`[shift+n]` 'New' — 'I want to insert a fire report into slot [3]' -> Opens a modal that allows the
> operator to 'craft' what is effectively a 'request order' that the Producer will then use to propose
> to the Director ('Hey, Operator wants fire report for Oceanside, CA 92057' <so
> `Origin.fromOperator`>)"*

**THE OPERATOR DOES NOT MAKE A CARD.  They make a REQUEST**, and the Producer turns it into a proposal —
which keeps the role split intact at the one place it was most likely to be broken.  The operator's will
enters as an order; what cards should exist is still the Producer's, and the running order is still the
Director's.

**Three things it carries**, each with a consequence already ruled:

| The request carries | Consequence |
|---|---|
| a **kind** ("fire report") | **NO SLOT EXISTS FOR IT.**  The registry holds `LocationReport`, `SevereRead`, `BreakingAlert` and `Transition`, and says a slot nobody proposes is dead code (AP-DEAD-01).  D-31's card-types work |
| a **subject** (Oceanside, CA 92057) | the fence guards it (DR-13), which is where the HUM LEAD put it: *"we should have guards so you can't do that in the first place"* |
| a **target slot** ("slot [3]") | **LINE-UP space, not schedule space** (D-44) |

**AND IT IS EXPLICITLY HELD FOR UAT** (HUM LEAD): *"I'm not sure if there's any other use-case AT THE
MOMENT, this will likely need some UAT once the UI is built and I want to actually 'run it'."*  So the
mechanism is designed now and the shape is not frozen.

---

# D-47 — ORIGIN IS A TERM IN THE RANKING, NOT A TRUMP — AND THAT IS WHAT "IN DOUBT" MEANS

**This resolves a collision between D-41 and the HUM LEAD's own example.**  D-41 says `FromOperator`
*"automatically gets a higher rank."*  But:

> *"it's a low priority read (station credits) but we're due for a location report because we haven't
> had one in a while in the line-up, so the Director may say — 'cool, I hear you, but it's been <6
> minutes> since we've done a location report, so I'm gonna choose that **unless you absolutely tell me
> not to**.'"*

**Both hold only if origin is a TERM rather than a trump**, and the HUM LEAD's own last clause says so.
So a request carries an **insistence**:

| Level | The Director | |
|---|---|---|
| **Requested** (default) | origin is a strong term; **cadence can outrank it** | **and THIS is D-40's "when in doubt" — the choice modal belongs exactly here** |
| **Insisted** | does not choose.  It places it | *"unless you absolutely tell me not to"* |

**That collapses three rulings into one mechanism.**  D-40's choice modal was a surfacing with no
trigger; D-41 was a rank with no scale; the cadence rule was a criterion with nothing to weigh against.
Together they are one ranking with a defined tie-break and a defined escalation.

**THE COST, AND IT IS STRUCTURAL — see F-76.**  "It's been six minutes since a location report" is a
question **nothing can answer today**.

---

# THE OPEN CONTENT QUESTION

**What makes a join need something said?**  The mechanism is one function; the RULE is a HUM LEAD
content ruling and is not yet made.  Candidates raised and not chosen: a change of origin, a change of
subject, a report following an alert, elapsed time since the last transition.  Built with D-43's stated
rule — *an operator-originated card is bookended* — as its initial content, because that is concrete,
it is the HUM LEAD's own words, and it exercises all three of the leading/tailing/both cases.

---

# D-48 — WHAT THE DIRECTOR REMEMBERS, AND HOW LITTLE

> *"I would agree this is something only the director needs to see, and it's bounded by the types of
> cards/reports we have (so it's bounded) and it's just a 'since last read of this type' — **read
> history is too overweight and violates our 'done/discarded cards can pile up' concern**."*

**ONE TIMESTAMP PER SLOT, in an array bounded by the registry.**  It cannot grow, so there is nothing to
cap, nothing to evict, and no second owner — a log would have wanted all four, and the HUM LEAD named
exactly that objection.  It closes **F-76**.

**Written where a card LEAVES THE AIR READ IN FULL** (`takeOffTheAir`, `to == Done`), not where one is
removed.  A discarded card was taken away precisely so the listener would not hear it.

## The degradation requirement, and it is arithmetic rather than a branch

> *"I would also architect this in a way where the Director can still function and make other
> evaluations in the event this particular stack fails or is somehow disabled (maybe a broadcaster
> specific setting — where the Operator does/does not want 'last read' to be a factor in Director
> prioritization)."*

**`Settings.WeighLastRead` is the switch**, and `false` is the zero value — a station that never chose
gets the behaviour that existed before the term did.

**`overdue` ANSWERS THE SAME FOR EVERY CANDIDATE ON EVERY DEGRADED PATH**, and a term equal for
everything discriminates between nothing, so the ranking falls straight through to the watchlist:

| Path | Answer |
|---|---|
| the operator switched it off | `0` for everything |
| the slot is outside the registry | `0` — **fail soft: it costs the Director its opinion, not its ability to choose** |
| nothing of that kind has ever gone out | `neverRead` for everything, at a cold start |

**That is why it degrades without a branch anyone has to remember to write.**

## A test caught the rule being wrong, and it was the HUM LEAD's own example that showed it

**`neverRead` is the LARGEST duration, not zero**, and it was zero first.  *"We have never done one"* is
the strongest possible case that one is due — stronger than any elapsed time.  With zero, a severe read
put out six minutes ago outranked a location report that had **never** gone out, which is the exact
inversion of the ruling that motivated the whole term.

**And it does not break the cold start**, because then every kind answers `neverRead`, so the term
discriminates between none of them.

## What came with it

**`Proposal.Slot`** — the cadence term discriminates between KINDS, and *"we haven't had a location
report in a while"* is not a statement about a location.  The zero value is `LocationReport`, which is
what every proposal was before.  Two guards came with it: the top-off admits only what belongs on the
**main track** (a takeover drains on the rail, and admitting one here would spend a slot it never
filled), and **the Director's own structural cards are not the producer's to propose** — a tripwire,
because today that holds only by accident (D-42's shape again: a structural card's words are fixed at
proposal, `Proposal` carries none, and `check` refuses a wordless transition).

**STILL NOT BUILT, and deliberately:** the ORIGIN term and the INSISTENCE levels from D-47.  Neither has
a producer until D-46's request order exists, and a closed-set member with no writer is what the wires
gate refuses.

---

# D-49 — THE TRANSITION RULE IS A PROPERTY OF THE KIND, AND MVS-D-80 ALREADY RULED IT

**I built the wrong rule first, and the record already had the right one.**  An arm fired on
`Origin == FromOperator` and spoke words I invented.  **MVS-D-80** (HUM LEAD 2026-09-05,
`follow-ups.md` F-27):

> Fires when **something that INTERRUPTED THE PROGRAMME leaves the air and the programme resumes**: an
> alert takeover, and a `[w]` read including one cut short by `[esc]`.  Does **NOT** fire
> location-to-location — *"the location scripts already announce their location"* — nor between alerts
> inside a burst.

**And the words were already written and pinned** (`a3c495b`): `transition/resume.txt`, plus
`transition/masthead.txt` (F-24).  `app/transition.go` carries `programmeReturnLine` and `mastheadLine`,
both complete, both with zero production callers.  **My arm also fired location-to-location**, which
**S-5** independently calls *"jarring to a listening audience"*.

## The HUM LEAD's re-reading of D-43, which is what unlocked it

> *"It does not necessarily need transitions — it was an example to show the function of the DIRECTOR
> understanding how a card fits into the line-up and to determine appropriately if a card actually
> requires a transition or not."*

**So origin is not the trigger.  THE KIND IS** — and MVS-D-80's own reason generalises into the rule:

| Row | Means | Set by |
|---|---|---|
| **`announced`** | this kind does NOT introduce itself, so the listener is told what is coming | **nothing yet** — station credits will, and credits have no slot until D-31 |
| **`handsBack`** | this kind INTERRUPTED the programme, so the listener is handed back after it | **`BreakingAlert`** |

**Adding an inter-card card is therefore a ROW**, which is the requirement in the HUM LEAD's own words:
*"the system needs to be flexible enough that adding additional transition or inter-card cards is low
cost, and doesn't require a complete rewiring."*

## The words are not the Director's

**It owns ARRANGEMENT; the script library owns CONTENT**, which is the S-7 boundary T-3 draws.
`Settings.ProgrammeReturn` and `Settings.Announcement` are handed in by the app from the script library,
so the sentence has ONE owner and the Director never invents one.  **Empty means say nothing** — a
station that speaks a line nobody wrote is worse than one that moves on.

## A failing test moved the design, and F-27 had already said where

The hand-back is minted **while the interrupting read is still ON AIR**, not at admission.  F-27's design
note asks for exactly that — *"enqueue it while the interrupting read is still on air … the hazard to
design against is the DUCK-BOUNCE"* — and it also **tells "read" from "never played" with no stored
association**: a takeover dropped before it aired never had a hand-back to strand.  The first attempt
minted it at admission, and a dropped takeover left a stray *"we now return to our regularly scheduled
programming"* with nothing before it.

## PLACEMENT — RE-OPENED, AND RATIFIED PROVISIONALLY (HUM LEAD, 2026-09-10)

**MVS-D-80 placed the transition "at the duck"** in `mastercontrol`, because the lift *"is the one point
that sees the takeover, the `[w]` read and the relay ALIKE."*  **With the relay case ruled out of scope**
(HUM LEAD 2026-09-10), two of those three are cards — and T-3 says inter-card transitions ARE cards,
where the Director can adjust and remove them.  So the takeover's hand-back is **a card at the end of the
rail**.  If that is wrong it costs one registry row, which is what the design was for.

> **RATIFIED, HUM LEAD 2026-09-10:** *"MVS-D-80 place is ratified for now — we may change based on UAT
> if it doesn't work as expected."*

**HELD PROVISIONALLY, AND UAT IS THE INSTRUMENT.**  This is a change a listener HEARS and a reader
cannot check: whether a hand-back at the end of the rail lands where a human expects it is not
decidable from the schedule.  So it is ratified to be TRIED, and the cost of reversing it is a registry
row plus a placement — which is why it was safe to try.

## Two plants survive, and both are named rather than hidden

- **the lead arm is unexercised in production** — nothing sets `announced`, so a mutation to where a
  lead is placed changes nothing observable.  `insertBeside`'s two sides are pinned directly instead,
  which is what stops `before` silently meaning "after".  It closes when credits get a slot.
- **the on-air guard in the prune is equivalent FOR TAILS** — a hand-back at the head is kept by the
  position rule anyway.  It is a live precondition for the LEAD path, which has no row, so it stays.

## Also settled

**T-3's visibility clause is superseded.**  `lineup-model.md:229` said a transition needs a slot *"where
the Operator can see and drop it"*; `role-model.md:72` says the operator *"never manages transitions"*,
and the HUM LEAD confirmed 2026-09-10: *"Operator never deals with transitions, that's the Director's
job.  The Operator manages the 'main cards' of the lineup (Report types, credits, etc)."*

**The relay hand-back stays out of scope** (HUM LEAD) — it is not a card join, the programme being the
bed, and F-27 names its hazards: the duck-bounce and the F-D5 lock.  P5's.

---

# D-50 — THE BREAKPOINTS, AND THE FLOOR BELOW WHICH WE DO NOT DRAW

> *"< 100 col : Not supported — we adopt a 'btop' style — 'resize your terminal to 100 x 25 or larger'.
> >= 100 <= 120 'smaller' — this is where titles may or may not truncate.  >= 120 <= 150 'Optima' —
> this is the range of the mock.  >= 150 'large terminals' — this will be supported in LATER releases,
> once we have things like terminal-based maps — this will have a 'right rail'."*

| Class | Width | |
|---|---|---|
| **UNSUPPORTED** | < 100 cols, or < 25 rows | btop-style: *"resize your terminal to 100 x 25 or larger"* |
| **COMPACT** | 100 – 119 | titles may truncate |
| **OPTIMA** | 120 – 150 | the range the reference mock is drawn at |
| **LARGE** | > 150 | **LATER RELEASE** — a right rail.  A SEAM ONLY (D-51) |

**150 IS THE TOP OF OPTIMA, NOT THE BOTTOM OF LARGE.**  The ruling's two ranges overlapped at exactly
150; the reference mock is 150 wide and the ruling calls that Optima's range, so that is the reading.
Stated rather than silently chosen.

## It REPLACES a vocabulary rather than adding a third, and that was checked

**F-68 warned that two breakpoint vocabularies already exist and 0.16.0 must choose one of three
answers.**  Checked before writing anything: **`broadcaster.go` is the ONLY production caller of
`term.BreakpointFor`** (lines 251 and 258); Observer uses its own `radioBP` at 84/146 and touches the
platform enum nowhere.  So the platform vocabulary is already Broadcaster's in practice, and redefining
its boundaries is F-68's *"wire the platform one"* answer rather than its *"write a third"*.

**`BreakMini`, `BreakSingle`, `BreakStandard` and `HeightCompact`'s 12-row rule all go with it** — the
new floor is 25 rows, from the same ruling.

---

# D-51 — THE RIGHT RAIL IS A SEAM NOW AND A LAYOUT LATER

> *"this will have 'right rail' — nothing needed yet, but make a note and ensure the architecture has a
> seam to support this cleanly."*

**WHAT IS OWED NOW IS ONE NUMBER FROM ONE PLACE.**  A right rail is not a new card renderer; it is a
SMALLER LANE.  So the requirement is that the card takes its lane width as an argument and derives
everything from it — which is what the v2 mock's generator demonstrates rather than asserts: the same
function draws 150, 130 and 100, and a right rail is simply a fourth number.

**THE FAILURE IT AVOIDS IS THE ONE v1 HAD.**  A card drawn at fixed columns has to be redrawn for every
lane it ever appears in, and the first sign is a half-drawn card — which is exactly how v1 was wrong.

---

# D-52 — THE CARD TYPES, v2

**`01-objectives/mock-card-types-v2.txt`**, and **v1 is deleted rather than kept**: it was drawn at the
occluded width and would have been built from.

**NOTHING IN IT IS A HARD-CODED COLUMN.**  Every row is GENERATED from the anchoring rule at each
supported width, which is the only way to be sure the RULE is what gets built rather than the drawing:

| | |
|---|---|
| box | fills the lane (width − 9 left chrome − 9 right chrome) |
| title | centred on the box; truncates **kind-first, subject-last** |
| badge | right-anchored, 2 cells inboard of the handle |
| handle | right-most, fixed 5 cells, 2 cells inside the edge — **the address the operator types**, so it never truncates and never moves |

**VERIFIED AGAINST THE REFERENCE, NOT AGAINST MEMORY.**  The generated 150-column row lands its badge at
col 122, its handle at col 134 and its edges at 9/140 — the same columns as `mock-broadcaster-v1.txt`.

**A LONG TITLE FOUND A BUG IN MY OWN RULE.**  The first draft tested whether the title FIT BY LENGTH,
but the title is CENTRED — so a title can be short enough to fit and still run through the badge once
centred, which is what it did (`…(COASTAL)D•`).  The test is POSITION, not length.  It is in the mock
set permanently so the same mistake cannot be made silently in code.

**Types drawn:** `LOCATION REPORT`, `SEVERE-EVENT READ` (SevereRead exists and nothing mocked it),
`WATCHPOST CREDITS READ` (**no slot — D-31**), `WEATHER ALERT • BURST` (`•PRIORITY•`), and a long
location report to exercise truncation.

**Not drawn, on the HUM LEAD's ruling:** the transition card, which the reference put at slot [7].
**The badge spelling is corrected** — the reference reads `•STANRARD•` in all ten card rows while its
own header spells `STANDARD`.

---

# D-50's CODE HALF — AND ONE NUMBER HELD BACK FOR THE HUM LEAD

**Built:** `platform/term.Breakpoint` is now `BreakUnsupported` / `BreakCompact` / `BreakOptima` /
`BreakLarge` at 100 / 120 / 150, with `String()` on each.  `HeightCompact` is **retired** — its 12-row
rule had no caller but its own test, so it went with the vocabulary it belonged to.  `broadcaster.go`
asks `>= BreakOptima` for the wide frame, and its column floor is **100**.

**THE FLOOR AND THE CLASSIFIER ARE NOW ONE NUMBER.**  `bcMinCols` was 80 and pinned by NOTHING, so it
could drift from `BreakpointFor` with no test noticing — two carriers of one rule, the shape this
release keeps un-splitting.  `TestTheColumnFloorIsTheUnsupportedBoundary` asserts the floor is the first
drawable width and that one column below it is not, so the two cannot move apart.

## The 25 rows: NOT applied, and flagged rather than silently chosen

**The ruling's example said "100 x 25".**  The **100 is the breakpoint** and is applied.  The **25
arrived inside an example of the MESSAGE** — *"resize your terminal to 100 x 25 or larger — or something
like this"* — and `bcMinRows` is **44, MEASURED**: fixed chrome plus one readable card, counted off the
mock at DISCOVER wave 1.

**Lowering it to 25 would let the console draw a frame the terminal cannot hold and clamp the remainder
away** — which is **F-55**, the defect this floor exists to prevent, arriving through the notice meant to
prevent it.  So 44 stands and the notice reads "Broadcaster needs 100x44".

**HUM LEAD: if 25 was meant literally, the honest fix is to make the chrome shorter at COMPACT, not to
lower the floor under it.**  That is a layout task, not a constant.

## The btop screen was already there

`notice()` has said the requirement AND the current size since wave 1, which is what the v2 mock
proposed independently.  What changed is the number it names.  Seven plants, all caught — including the
floor drifting from the classifier, and 150 slipping from the top of OPTIMA to the bottom of LARGE.

---

# D-53 — PRODUCT FIRST, OPTIMISATION SECOND, AND THE BASELINE IS THE REASON

> *"we never over optimize on theoretical software — we need to make the product (Broadcaster UI) work
> FIRST, then once we are happy with the product behavior and function, THEN we can look at performance
> and optimization passes, because we'll have a working baseline to ensure expected behavior and
> function doesn't regress.  Attempting to do that now would not only be too soon, but may corner us and
> make intended functionality hard-to-impossible."*

**RULED IN ANSWER TO THE CARD ROW'S ALLOCATION COST** — 14 → 266 per frame at a full line-up (see
`docs/accepted-costs.md`).

**THE ARGUMENT IS ABOUT CORRECTNESS, NOT EFFORT.**  An optimisation made before the behaviour is settled
has **no baseline to prove it did not change behaviour** — and it constrains functionality that has not
been written yet.  Optimising here would be a correctness risk wearing a performance costume.

**What this DOES NOT relax:** the cost is still measured, still recorded, and still surfaced to the
HUM LEAD with a number.  *"Never make Branden be the one to notice"* is unchanged; what changes is what
happens next, which is nothing.

---

# D-54 — THE PRODUCER IS WIRED, AND THE LEDGER ROW DISCHARGED ITSELF

**`Event.Offered` has a writer.**  `executors.run`'s `Publish` case asks the producer and returns the
offer — **no new effect**, because `run` already returns whatever an effect learned, and a publish is
the moment the schedule has SETTLED, which is exactly when the producer can see what the line-up still
needs.  That was the HUM LEAD's option 2, unchanged.

**THE CHAIN IS SELF-LIMITING BY THE DEPTH, NOT BY A COUNTER.**  Publish → Offered → the track fills →
settle publishes → Offered again → nothing left to admit → `onOffered` returns **no effects**, so there
is no publish and the chain has nowhere to go.  Pinned by
`TestTheTopOffChainStopsOnceTheLineUpIsFull`, because a feedback loop fed from the effect that publishes
is the one thing here that could run away.

**THE DEPTH AND THE CONSOLE'S WINDOW ARE ONE NUMBER.**  `tty.MainTrackSlots` is exported and `app` sets
`Settings.Depth` from it.  A second constant would agree today and drift silently, and the failure reads
as a bug from NEITHER side: the station either holds cards the operator cannot address, or leaves slots
empty for ever.

**AND THE PROPOSALS KEY THE WAY THE ROTATION DOES** — `snapshot.Key`, the same ref
`radioDeck.needsRead` reports — so `ReadID` gives a location ONE identity across both paths, and the
lineup's own refusal of a duplicate is what stops a place being read twice (FR-2.5).

## The gate discharged its own row

**`wires` reported `STALE EXEMPTION Event.Offered` — "it is WIRED now"** — on the commit that wired it,
which is the self-expiring ledger doing exactly what it was built for.  The row is gone.

**Two other gates fired and both were right:** `TestExecutorsRefuseToBeBuiltWithoutTheirSeams` refused a
seam that was neither checked nor declared optional (`propose` is now explicitly optional — a station
with no producer still broadcasts, it just never tops itself off), and the declaration-set golden
caught the two new top-level functions.

**A STALE COMMENT WAS FIXED RATHER THAN LEFT** — `TestAPublishIsCarriedAndDeclined` still said the
publish is "EMITTED and nobody reads it".  The console has read it since P2 and the producer answers it
now.  That is the **F-69 shape**, caught in the same file it applies to.

---

# D-55 — THE INJECTOR SHIPS, AND THE CARD LEARNS TO SAY "TEST"

> *"all UI portions of the inject alert are labeled **TEST EVENT** … we also have a test script which
> repeatedly indicates that it's a TEST … the Station Operator NEEDS that functionality to test his
> machinery, and the visual and audio call outs make it clear.  There's also real-world precedent for TV
> stations and Radio stations to do this very activity while actively on air."*

**THE PREMISE WAS CHECKED SURFACE BY SURFACE, AND IT HELD EVERYWHERE EXCEPT THE ONE THIS RELEASE ADDS.**

| Surface | Marked | Where |
|---|---|---|
| Ticker band | ✅ `**TEST EVENT**` at **both** ends, so it cannot scroll off-window | `ticker.go:108` |
| Severe `[w]` table | ✅ leads the EVENT column | `severe.go:419` |
| `[w]` read audio | ✅ four separate "this is only a test" statements | `test-alert/{head,title,explain,tail}` |
| Takeover audio | ✅ `allFabricated` → the test scripts | `compose_takeover.go:46` |
| Band cue from a card | ✅ `Test: e.Fabricated` | `itemsOf` |
| **Broadcaster line-up card** | ❌ **NOTHING AT ALL** | `Card` had no `Test` field |

**`takeoverOf` built the card from the headline and DROPPED `Arrival.Test`**, which survived only into
`selectBurst`'s ordering.  So the console drew a fabricated takeover as an ordinary one — the exact
hazard the original "never ship" rule was written against, arriving on the surface that did not exist
when that rule was written.

**Two safeguards worth crediting**, because they are stronger than the record suggested: `selectBurst`
**reserves every real hazard's slot before a test event is offered one**, and a fabricated emergency
takes **no** Max exemption — *"a fabricated event taking that room is a fabricated event silencing a
real alert."*

## What was built

**`Card.Test`, rendered PER SURFACE.**  The severe window already made and stated this call — the mark
is added at render *"rather than to the row's Product: the `[w]` read speaks that field, and a product
with three asterisks in it would be read aloud as asterisks."*

**EVERY alert in the burst, not ANY.**  One card carries the whole burst (MVS-D-77), so a burst holding
one REAL hazard is not a test; marking it would hide a live alert behind a label that says to ignore it.
This is the same call `allFabricated` makes about the WORDS, and the two must not disagree.

**A card that cannot be labelled honestly is NOT DRAWN.**  A test caught the outer clamp chopping the
mark into `**TEST E` — the one output worse than no mark, because a reader takes a broken label for
rendering damage and the warning beside it for real.  Drawing it unmarked is the screenshot hazard
itself, so the only remaining choice is to draw nothing.

## A go-studs gap, sent upstream rather than patched

**`SetPrefix` reserves its width and never renders on a data row.**  `prefixWidth` is counted
(`data_table_row.go:521`) and only `RenderHeader` writes it (`:210`), so a prefix set for the mark
reserved its space and printed nothing.  **M6 upstream candidate.**  The mark is a real column instead —
and TWO rows are built per lane rather than one, because a fixed column present on every card would put
every real hazard 14 cells off-centre to make room for a label it never carries.

---

# D-56 — "ONE CANONICAL WAY TO DO A THING" IS A STANDING RULE, ACROSS ALL ENGINEERING

> *"the 'one way to do a thing' extends to ALL aspects of our engineering, not just the router layer.
> This is/was a big helper in consolidating methods and providers in previous releases."*

**RATIFIED.**  Duplicate ONLY when isolation is a genuine need, **and name the need where the duplicate
lives.**

**The worked example is the diagnostic window:** rather than give the Broadcaster its own `ctrl+d`, the
Router composites the Dashboard's existing one over whichever surface is active.  One owner, no second
injector UI, and the console stays on screen underneath so the operator can watch the takeover drain.

**It is the same rule behind most of this release's real defects**, each of which was one rule carried
twice: the duck lifted by one spelling of `tune`; `bcMinCols` drifting from the breakpoint classifier;
the console's window drifting from the Director's depth; and every surviving plant that turned out to be
a redundant guard.

---

# D-57 — THE TEST MARK GOES IN THE CARD'S CORNERS, AND THE BADGE KEEPS ITS LANE

> *"we have a UI where overlays happen — we can mark 'TEST' on the card corners of a fabricated
> event"*, with `TEST EVENT` leading the title row and `TEST` at the bottom corners.  Badge: **"Keeping
> 'PRIORITY' is fine."**

**IT IS BETTER THAN ONE MARK, AND THE REASON IS THE OVERLAY.**  A single mark has a single point of
failure — it can be truncated, and more importantly **the priority track sits ON TOP of the main
track**, so a takeover card is exactly the card most likely to be partly occluded.  A one-position mark
on the one card type that gets overlaid is the weakest possible placement.  Marks at fixed corners
cannot all be lost to truncation, occlusion, or a cropped screenshot.

**It is the ticker's own reasoning, one surface along** — `testEventMark` is prepended AND postpended
because *"the tape scrolls: a marker at one end only is off-window half the time, and a marker at both
ends means the item cannot be on screen without one of them."*  The card is the same argument in two
dimensions.

**THE BADGE STAYS `•PRIORITY•` (HUM LEAD).**  The two facts never compete for one slot: the lane badge
is the only thing that says WHICH lane the card is on, and for a takeover that is what the operator most
needs to see.

## It waits on the boxed card, and that is a real dependency

**The console draws FLAT ROWS today — there are no borders to put corners on.**  The reference mock
draws boxes throughout, so the boxed renderer is the next rendering step and the corners land with it.

**What is in now (D-55's single leading mark) is a correct interim FOR A ROW**, and it is superseded
rather than extended when the box arrives.  Its "a card that cannot be labelled honestly is not drawn"
backstop SURVIVES the change: it is about the mark being whole, not about where the mark sits.
