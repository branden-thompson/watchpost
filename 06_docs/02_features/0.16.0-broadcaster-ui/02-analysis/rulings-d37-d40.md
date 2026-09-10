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
