---
title: "Map-ready geometry — DISCOVER report"
date: 2026-09-21
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
release: 0.17.0
---

# DISCOVER — map-ready geometry

## What was done

The requirements for this release came from a place they usually do not: **a
working integration**. The terminal-map library was built into the Observer as a
spike, and what it could and could not draw is the requirement. That spike is on
`spike/tuimaps-first-host` and is kept as a reference until the integration
proper; it is never merged.

Then the data paths were read and **measured** - see
[the data, measured](../02-analysis/data-shape.md).

## Two beliefs this release started with, and both were wrong

**"The code documents that alert geometry is not carried."** It does not. The
line cited says no provider carries *sunrise and sunset* times and computes them
from solar geometry. **There is no record anywhere of a decision to drop alert
geometry**; the response struct simply never declared the field. An undocumented
omission is worse than a ruled one, and it is ruled now.

**"Alert polygons are enormous."** They are 7 to 13 vertices. The thousands-of-
vertices figure came from a zone shape in the map library's own scenario, which
is a different thing. Carrying alert polygons is nearly free; **carrying zone
shapes is the expensive half**, and an alert can name eighty-one zones.

Both were caught by measuring rather than by reading a call chain, which is P-6.

## Decisions taken

| | Decision | Why |
|---|---|---|
| **MG-1** | **Zone-only alerts must be drawable as areas**, not only the quarter that carry polygons | HUM LEAD: *"25% is not sufficient for integration once maps are available for a weather app, so it needs to work."* The cheap option was offered and declined |
| **MG-2** | **Zone shapes are keyed by id in their own store, not attached to the alert** | `snapshot.Alert` already carries `AffectedZones`. Keeping shapes out of it means no schema change for alerts, and a report that does not grow by hundreds of thousands of numbers for one outlook covering eighty-one zones |
| **MG-3** | **The geometry type lives in `platform/geo`; fetching and caching live in `domains/weather/nws/zones`; `app/` wires them; `platform/tuimap` consumes** | HUM LEAD's rule: data that spans domains belongs in `platform/`, data only one thing uses belongs in a domain. The import gate forces the *type* into `platform/` - neither `modes/` nor `platform/` may import a domain, and both must name it |
| **MG-4** | **Named for the data, not for its first use** | `domains/maps` was offered and set aside: zone shapes are NWS data from the same endpoint family as the alerts beside them, and they may later serve description rather than only the map. `domains/maps` stays available for concerns that are genuinely map-only |
| **MG-5** | **Earthquakes keep their epicentre** | The position already exists in the reader and is thrown away. This is a field, not a feature |

## Decided in PLAN, after the measurements

| | Decision | Why |
|---|---|---|
| **MG-6** | **Simplify at ingest, at 0.5 km, and keep only the simplified shape** | Half a kilometre is below what one braille dot resolves at any usable zoom, and it takes the worst zone measured from 12,004 vertices to 744 - the source carries collinear detail no terminal can draw. The cost is accepted: **the raw shape is gone**, so a later need for finer detail is a refetch. Zones are keyed by id, so that is a cache rebuild rather than a redesign |
| **MG-7** | **Fetch a zone on demand; seeding the country stays available** | The median alert names one zone and the 95th names five, at ~150 ms each and cached for good. Seeding all ~3,500 zones would be a few megabytes simplified, and remains the answer if on-demand proves wrong |
| **MG-8** | **The schema is bumped to 1.1.0-rc** | HUM LEAD's rule was additive where nothing is imposed on consumers, otherwise bump. `additionalProperties` is **false** on every object (`pkg/schema/schema.go:106`), so any new field fails a consumer validating new data against the old schema. That imposes, so it bumps - and at 0.x with an `-rc` suffix a bump is cheap |

## What is deliberately not decided yet

- **How much a zone shape is simplified, and where.** The map library simplifies
  for drawing anyway; storing detail no terminal cell can show is waste. The
  tolerance is a PLAN question and wants measuring, not guessing.
- **Whether zone shapes are fetched on demand or seeded.** An alert naming
  eighty-one zones on a cold cache is the case to answer.
- **Whether the schema needs a version bump** for the earthquake position. The
  policy says a new provider id is additive and a new top-level block is a
  change; a field on an existing object sits between them. Flagged for HUM LEAD
  rather than assumed.

## Risks carried into PLAN

| | Risk |
|---|---|
| **MG-R1** | An eighty-one-zone alert on a cold cache is eighty-one fetches. Unanswered, this is a stall on first draw |
| **MG-R2** | Zone shapes are the large kind. Held unsimplified, they are a memory problem; simplified too hard, a county stops looking like itself |
| **MG-R3** | The allocation budgets are gated, and the USGS memo path this work touches has one of the eight pins |
| **MG-R4** | Coordinates must never reach spoken text (UAT 81). A new geometry field must not become narratable by accident |
