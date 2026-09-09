---
title: "0.16.0 DISCOVER — HUM LEAD rulings D-11 through D-18"
date: 2026-09-09
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "D-11..D-17 ruled.  D-18 (the settings table) is presented and awaiting per-field rulings."
---

# Rulings D-11 to D-18

## D-11 (OQ-12) — THE MAIN TRACK **IS** THE ROTATION.  My framing was wrong.

> *"The main track is the rotation - main track has 10 cards, there's a priority track for
> take-overs/alerts which always drains first, and a bed where the relay stream reside.  The operator
> can 'switch over' the main track to the bed - at which time the main track line-up 'pauses,' thought
> the operator can still perform management actions on them.  Priority track can still take over when
> the bed is playing and still drains first."*

**RULED, and it dissolves the question rather than answering it.**  I presented "absorb the rotation
into the main track" versus "build a producer beside it" as two options.  They were never two things:
**the main track is the rotation.**  Option C — descoping ordinary reads — is therefore also withdrawn,
because there is nothing to descope; the rotation is the stack.

### The model, and it maps onto the code exactly

| The operator's model | The code today |
|---|---|
| **Main track** — 10 cards, the rotation | `MainTrack` (`lineup.go:15`) — the rotation's queue |
| **Priority track** — takeovers and alerts, **always drains first** | `AlertRail` (`lineup.go:16`); *"THE ALERT RAIL DRAINS FIRST (DR-3)"* (`lineup.go:171-188`) |
| **The bed** — where the relay stream resides | Already modelled, and **deliberately not a track** |

`Track`'s own comment is the confirmation, and it was written before this ruling
(`platform/lineup/lineup.go:9-12`):

> *"The bed — the live NOAA relays — is deliberately not here: it is a selectable resource the Director
> may cut over to, not a queue of cards, and modelling it as a third track would invite something to be
> scheduled onto it."*

And `bed.go:2-3`: *"the live broadcast the programme rides on, and the one decision the Director makes
about it."*

**So the architecture already holds all three lanes, in the operator's own terms.**  What is missing is
the wiring, exactly as with `Publish`, `Power`, `Fence`, `Origin.FromOperator` and `term.Breakpoint`.

### What the ruling ADDS that was not in the code or the brief

1. **A switch-over: the operator moves the main track to the bed.**  There is no operator-initiated
   cut-over today — `Tune` is emitted automatically on dwell or on a cycle ending (`bed.go:178,215`),
   and `cutTo` carries *"IT MUST NOT LIFT THE DUCK … nobody pressed anything"* (`executors.go:113-123`).
   **An operator-initiated cut-over is a pressed thing**, so that comment's premise no longer covers
   every caller, and the duck's behaviour on a deliberate switch-over is now a design question.
2. **A PAUSED main track that still accepts management actions.**  This is a new state: the lineup
   holds, no card advances, and yet promote, demote and drop remain live.  `Power.OffAir` holds
   everything including the rail, so it is **not** the same state — pausing the main track while the
   bed plays and the rail still drains is a fourth condition the `Power` enum does not yet express.
3. **The priority track still takes over while the bed is playing, and still drains first.**  This is
   the safety property, and it is already true in the model: `advances(AlertRail)` returns true for any
   power except `OffAir` (`power.go:112-117`).

**These three are the release's real design work**, and they replace the "two paths to speech" framing
in `wave2-findings.md` §1, which is superseded by this ruling.

## D-12 (OQ-11) — The service radius is the boundary, with a possible National exemption

> *"Service radius is the boundary - with a *potential* exemption for anything that could be considered
> 'National' - though in that situation I expect it would manifest as local alerts."*

**RULED.**  One boundary, and it is the service radius.  `lineup.Fence` already is that mechanism —
*"a HARD boundary on what reaches the lineup at all … not a sort key"* (`fence.go:95-97`) — so this is
a naming and wiring question, not a new concept.

**My reading, stated for correction:** Observer's alert-notification radius and Broadcaster's service
radius are **one value**, not two that must be reconciled.  That resolves the one-Director conflict by
unification rather than by precedence.  **If the intent was instead that Broadcaster's radius overrides
Observer's while both exist, say so and D-12 is amended.**

**The National exemption is recorded as a candidate, not a requirement.**  The HUM LEAD's own
expectation is that a national-scope event manifests as local alerts anyway, so the exemption may never
need to exist.  DISCOVER should check whether any product in the feed carries a national scope that
would NOT appear locally; if none does, the exemption is dropped rather than built.

## D-13 (OQ-9) — Adopt the platform breakpoint vocabulary

> *"A is fins in conjunction with whatever our 'min size' floor is."*  (read as "A is fine")

**RULED: option A** — adopt `platform/term`'s `Breakpoint` and redefine its boundaries, tied to the
minimum-size floor from D-4.  It is uncalled today, so its boundaries are free to change now and
expensive to migrate later.  **F-68 is thereby dispositioned**, and Observer's `radioBP` becomes the
open question of whether it also migrates — which is NOT ruled here and should not be assumed.

## D-14 (OQ-10) — The credits read is NOT a licence obligation

> *"No, this is just something for us as a potential credit for watchpost card."*

**RULED.**  The on-air credits card is a **Watchpost self-credit**, optional, and carries no legal
weight.  The licence obligation that `credits.go:16-20` names is discharged by the **About window's**
on-screen list and is unaffected.

**Consequence:** the card needs no new `Slot` on obligation grounds.  If it ships it is a courtesy
card, and it is the first thing to drop under time pressure rather than the last.

## D-15 (R-8) — Approved

> *"Approved."*

**R-8 is a requirement of this release**, in the four parts drafted: the radius admits to the priority
cadence; cadences stay bounded by source refresh and politeness limits; the cadence set keeps one
owner; the operator can see effective freshness.  **It names no numbers, deliberately.**

## D-16 (OQ-8) — A hard cap, plus population-ordered admission

> *"A is fine - we can also plan for some kind of sort logic - prioritize areas of higher population
> first, then smaller / less dense cities, town, areas, etc."*

**RULED: a hard cap, with population-descending admission order.**  When the service radius admits more
locations than the cap allows, the cap is filled by population, densest first.

**This is buildable today.**  `geodata.City` already carries `Population` (`domains/locations/geodata/index.go:42`,
parsed at `:186-187`), and the Open-Meteo geocoder returns a `population` field
(`domains/locations/openmeteo/geocode.go:50`).  **No new data source is required.**

**Recorded as a risk to carry:** population-first admission means a sparsely populated area inside the
radius can be excluded by the cap while a denser one is admitted — which is defensible for a broadcast
audience and worth stating plainly in the operator-facing copy, because the operator will notice a
town missing and should not have to guess why.

## D-17 (OQ-13) — Authorized to render the STANDBY wording variants

> *"That's fine."*

**Rendered** at exactly 150 cells in `02-analysis/standby-wording-variants.txt`, three variants,
awaiting a pick.  Each resolves both problems at once: which half of the banner is the live state, and
the bed row's third use of the word STANDBY.

## D-18 — The settings table is presented; per-field rulings pending

> *"Print out a table of all settings and I'll provide rules for each"*

**52 persisted paths, presented at 25 natural ruling units** in `02-analysis/config-field-table.md`.
Awaiting a ruling per row.
