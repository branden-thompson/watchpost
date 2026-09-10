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
