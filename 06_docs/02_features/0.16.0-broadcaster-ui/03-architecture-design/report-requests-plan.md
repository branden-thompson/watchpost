---
title: "Line-Up Requests — [r], and the report set that has to be able to grow"
date: 2026-09-14
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "RATIFIED 2026-09-14 — six rulings below.  Build approved."
---

# Line-Up Requests

**HUM LEAD, 2026-09-14:** *"this is part of the foundational functionality of Broadcaster so now is the
correct place to do this work, despite the risk."*

## The ruling that shapes the whole design

> *"while we have a set group of reports now, we WILL have more report types in the future, and
> ensuring Broadcaster is flexible enough for that list to grow and change WITHOUT having to completely
> re-architect the code and flow every time is critical.  It's worth the careful thinking and cost now
> vs. trying to bolt it on later when additional features may code us into a corner."*

**So the acceptance test for this design is not "does it show four reports".  It is: what does adding
a FIFTH cost?**  The answer this plan commits to is **one row in one registry**, and nothing else —
no new field, no new branch in the modal, no new case in the naming rule, no change to the card, the
event or the table.

That is the same shape `platform/category` already has, and its comment is the standard being copied:
*"A function rather than a package variable (P10-06), and the ONLY place a category is described."*

## The rulings

| # | Question | Ruling |
|---|---|---|
| 1 | How is the chosen set named? | **Both** name and label in the MODAL; **labels only** in the running order, comma-delimited: `02.  NWS, FIRE, QUAKE`.  All of them reads `Location Report, Full` |
| 2 | A location outside the service radius | Helper text under the field, Observer's own lookup-feedback pattern: *"Location not found in Pool."* and, in yellow italics, *"Observer supports location lookup outside Broadcast Radius"*.  Wording may shorten to fit |
| 3 | PRIORITIZE | **UP NEXT only, NEVER LIVE** |
| 4 | Push-down at a full track | **The last card falls off into the discard pile.**  The Producer may re-request a copy; the operator may also look the location up and re-place it by hand |
| 5 | Freshness | **Every card is checked as it enters UP NEXT**, already general — a pushed card uses the same mechanism as one arriving from slot 3 |
| 6 | May the Director refuse? | **NO, not the human operator** — not in 0.16.0.  It refuses only an invalid card.  *"The only thing that would supersede an operator action is a valid alert within the broadcast service radius"* |

## The design

### `platform/report` — the registry, and the only place a report type is described

```go
type Kind uint8                 // NWSForecast, Marine, Fire, Seismic, …
type Spec struct{ FullName, Label string; Order int }
func Of(k Kind) Spec            // the registry, a function (P10-06)
func All() []Kind
type Set uint32                 // one bit per Kind — room to grow without a shape change
```

**`Set` IS THE THING THAT TRAVELS.**  A bitset rather than a slice: it is comparable, so it works in a
memo key and in `==`; it has a zero value that means "nothing chosen"; and it costs the same at four
kinds as at twenty.  A slice would have needed an equality helper at every seam it crosses.

**The naming rule lives with the registry, not with the table** — `Set.Describe()`:

- every kind → `Location Report, Full`
- otherwise → the labels, comma-delimited, **in registry order** so the string is stable

A fifth kind changes none of that: it is a row, and `All()` grows.

### What each layer gains

| Layer | Change |
|---|---|
| `platform/report` | **new** — the registry, `Set`, and the naming rule |
| `platform/lineup` | `Card.Reports report.Set`; a new operator event `Requested`; and **`Insert`**, the mutator that does not exist today — push-down with the last card falling to the discard pile (ruling 4) |
| `app` | gathers **per kind** rather than unconditionally: the four sources `segments()` already collects become a table keyed by `Kind`.  **This is where the risk is** — see below |
| `modes/tty` | the modal, and the running order drawing `Set.Describe()` |

### The risk, stated before it is built

**`composer.Compose` has never been asked for a report without an NWS forecast.**  It is the backbone
of every location report today — the other three are `synth.Reports{Fire, Seismic, Maritime}` hung off
it.  A request for `FIRE` alone must produce a report that says something sensible and does not read
as a broken forecast.

**That is batch 2 and it is measured before the modal is written**, because if the composer cannot do
it the whole feature changes shape — and finding that out after building a modal for it would be the
expensive order.

## The batches

| # | Delivers | Why this order |
|---|---|---|
| **R1** | `platform/report`: registry, `Set`, `Describe` | Pure, no dependencies, and the naming rule is testable on its own.  **A fifth kind is added in its test to prove the cost is one row** |
| **R2** | The composer takes a `Set` | **The risk.**  Proven with a single-kind report before anything is built on it |
| **R3** | `Requested` + `Insert` + the discard fall-off | The schedule's half.  FR-3.3: an act is an EVENT, and the operator is never refused (ruling 6) |
| **R4** | The modal, and `[r]` bound | The cheapest part, and it needs the other three to be true first |
| **R5** | The running order draws the labels | `02.  NWS, FIRE, QUAKE` |

**`[l] Lookup Location from Pool` is ALSO drawn and unbound** — found while reading for this plan. It
is not in scope here and is recorded as a follow-up rather than fixed quietly, because a control that
is drawn and dead is the defect `[A]` just was.
