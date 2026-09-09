---
title: "0.16.0 DISCOVER — HUM LEAD rulings D-11 through D-21"
date: 2026-09-09
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "D-11..D-21 ruled.  Nothing outstanding.  D-12 is AMENDED by D-20 — read them together."
---

# Rulings D-11 to D-21

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

## D-12 (OQ-11) — ⚠️ **AMENDED BY D-20 — DO NOT READ THIS SECTION ALONE**

> **This ruling was recorded correctly but I MISREAD it as unification — one radius.  D-20, below,
> carries the correction: there are TWO radii with different meanings.  Read D-20 before acting on
> anything here.**  *(Flagged after the DISCOVER-exit red team found this section still reading as
> current — RT-5.)*

### The ruling as originally recorded — The service radius is the boundary, with a possible National exemption

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

## D-19 — Every settings recommendation approved

> *"All recommendations approved."*

**RULED.**  All 25 rows of `02-analysis/config-field-table.md` take their recommended value, and the
five proposed Broadcaster settings in Table 2 are approved as proposed.

**Three things this settles that were previously open:**

1. **The five NEEDS-RULING fields are closed.**  The cast, the nine role voices and the root voice are
   **SHARED** — one station, one sound — which D-11 made answerable by settling that Broadcaster drives
   the same reads.  The tone mode and mute list are **SPLIT**, because a personal comfort mute must not
   silence a tone the station is meant to transmit.  Provider keys are **SHARED**, with the rate budget
   carried as a risk and D-16's cap as its mitigation.
2. ~~**D-12's unification is confirmed by silence.**~~  **WITHDRAWN BY D-20.**  I inferred unification
   from an uncorrected reading; the HUM LEAD then corrected it directly.  There are **two radii**:
   Observer's bounds ALERTS over an unbounded location set, Broadcaster's bounds LOOKUPS.
   `broadcaster.service_radius_mi` **stands**, and `ticker_radius_mi` is **not** renamed.
   *(Struck rather than deleted: the wrong inference is part of the record, and the lesson is that
   silence is not confirmation — see the disposition ledger, RT-5.)*
3. **Gain is Broadcaster's own and persisted**, separate from Observer's unpersisted listening volume.

**STILL OUTSTANDING, and deliberately not assumed:** the STANDBY wording pick (D-17).  Three variants
are rendered at 150 cells; "all recommendations approved" does not choose among them, because what I
recommended was the *principle* — the station keeps STANDBY and the bed row gets a different word —
and all three variants satisfy it.  **A, B or C is still needed.**

## D-20 — **D-12 AMENDED.**  Two radii, two different mechanisms.  My unification was wrong.

> *"Row 27 should stand - Observer's filter is for alert radius, but is unbounded for location.
> Broadcaster service radius is a HARD Fence for lookups, and may have different values.  Example,
> someone may want to run a 'hyper-local' station with a service radius of 3mi.  In this case - the
> reports and the alerts are constrained to that radius - which generally already includes
> warnings/advisories for the county."*

**RULED, and it corrects me.**  I read "service radius is the boundary" as unification — one value.
It is not.  There are **two boundaries operating at two different levels**, and conflating them would
have deleted the more important one.

### The distinction, confirmed against the code

| | Observer | Broadcaster |
|---|---|---|
| **What it bounds** | **ALERTS only** | **LOOKUPS** — the location set itself |
| **Location set** | **Unbounded.**  The watchlist may hold anywhere | **Bounded by the radius** |
| **Alerts** | Filtered by the alert radius | Constrained **transitively**, because every location is inside the radius |
| **Level** | Filters arrivals into an unbounded world | Defines how big the world is |

**The code confirms Observer's half exactly.**  `Fence.Admits(a Arrival)` (`fence.go:129`) takes an
*arrival* — an alert — not a location.  Nothing in the fence bounds the watchlist.  So today's radius
is an alert filter over an unbounded location set, which is precisely the HUM LEAD's description.

**And the county nuance is already load-bearing in the code.**  A zone-only alert has no point to
measure, and the fence admits it *"only by being one the app is already tracking at a watched
location"* (`fence.go:136-141`).  **That is the mechanism that makes a 3-mile station viable**: it
still receives the county and zone products for the location it tracks, so a hyper-local service
radius does not starve the station of warnings.  The HUM LEAD's *"generally already includes
warnings/advisories for the county"* is a property the tree already has.

### What this changes

1. **Row 27 STANDS.**  `broadcaster.service_radius_mi` is a real, separate setting.  Row 25 keeps its
   own meaning as Observer's alert radius and **is not renamed after all** — D-19's rename is
   withdrawn.
2. **FR-8 is rewritten** with two boundaries rather than one.  The Broadcaster radius is a **lookup
   boundary**, which is new work: nothing today bounds which locations may exist.
3. **The one-Director question dissolves rather than resolving.**  The two radii were never competing
   for the same fence.  Observer's feeds the alert fence; Broadcaster's bounds the location set that
   feeds everything.
4. **A hyper-local station is an explicit supported case.**  Three miles is a stated example, not an
   edge, so the minimum radius must be small and the cap of D-16 must behave sensibly when the radius
   admits very few locations — possibly one.

## D-21 (D-17 closed) — Variant C, with a coloured background for state

> *"Variant C accepted - we'll also color the background kind of like the ticker to make state
> IMMEDIATELY obvious as well."*

**RULED: Variant C.**  The station state is a labelled field with the transition in parentheses:

```
   STATION:  *** ON AIR · BROADCASTING ***                    ( SHIFT + ENTER  →  STANDBY )
```

and the bed row's third use of the word is replaced: `○ OFF BED  [ B ]`.

**Plus a background-colour treatment**, in the manner of the ticker, so the station's state reads
instantly rather than by reading words.  **The palette is the HUM LEAD's own pass**, per the standing
rule that colour is directed rather than chosen here; what DISCOVER records is the *requirement* that
state be legible without reading, and the constraint that whatever pair is chosen is measured by the
contrast register like every other painted pair.

**Two obligations this creates, both recorded now so they are not discovered late:**

- **The colour is not the only carrier.**  A background alone fails the `--ascii` path and any
  non-colour terminal, so the label carries the state in words as well — which Variant C already does.
- **The new pair enters the AA register.**  The completeness gate crosses the token vocabulary against
  the measured pairs, so a station-state background that is not registered fails the gate rather than
  passing silently.
