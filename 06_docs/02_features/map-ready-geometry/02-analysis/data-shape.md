---
title: "Map-ready geometry — the data, measured"
date: 2026-09-21
phase: DISCOVER
sev: SEV-0
---

# The data, measured

**Everything here was measured or read, not assumed.** P-6 of the process rules
says to run it once before asking for a ruling on a shape, and the numbers below
overturned two beliefs this work started with.

## The two alert paths

There are two, and only one of them is blind.

| Path | Endpoint | Geometry |
|---|---|---|
| Per-location, zone-scoped - feeds `snapshot.Alert` | `/alerts/active?status=actual&zone=…` (`domains/weather/nws/alerts.go:74`) | **Never modelled.** The response struct declares `Features[].Properties` and nothing else (`alerts.go:69-73`), so the `geometry` sibling is discarded at decode |
| National severe feed - feeds the ticker | `/alerts/active?event=…&status=actual` (`domains/globalfeed/nws.go:56-73`) | **Parsed.** `geoPoint()` (`nws.go:120-152`) streams tokens over the GeoJSON and keeps the first vertex as a representative point |

**The prior art matters more than the gap.** `geoPoint` is deliberately
iterative: *"the interface decode path recurses per array level with no depth
cap, so a hostile deeply-nested `coordinates` would overflow the stack and crash
the process (red-team 0.12.0 P4 F2)"*. Any polygon reader added here inherits
that discipline - bounded, token-streaming, never `unmarshal into any`.

## How much geometry there actually is

From the committed fixture `domains/globalfeed/testdata/nws_active_unfiltered_trimmed.json`,
16 alerts:

| | |
|---|---|
| carrying no geometry at all (zone-only) | **12 of 16** |
| carrying a polygon | 4 of 16 |
| vertices in those polygons | **7 to 13**, median 11 |
| vertices across all 16 alerts | **39** |

**An alert polygon is tiny.** This work began braced for thousands of vertices;
that figure came from a *zone* shape in the map library's own test scenario, and
a zone is a different thing from an alert polygon. Carrying alert polygons is
nearly free.

The proportion is corroborated independently by this application's own
`CHANGELOG.md:191-194`: *"many NWS products carry no polygon, including watches
and most flood warnings"*.

**Caveat, stated rather than buried:** this is one trimmed fixture of 16 alerts.
The three-in-four proportion is soft. The vertex magnitude is the solid part.

## How many zones an alert names

Same fixture:

| | |
|---|---|
| zones per alert | median **2**, max **81** |
| distinct zones across all 16 alerts | 124 |

The 81 is a Hydrologic Outlook. **This is what makes zone shapes the expensive
half**: zone polygons are the large kind, and one alert can name eighty of them.
Whatever is built must answer that case rather than the median one.

## The earthquake path

`domains/seismic/usgs/usgs.go:185-187` declares
`Geometry.Coordinates []float64 // [lon, lat, depthKm]`, and it survives into the
reader's own feature type (`usgs.go:203, 222-227`). It is **consumed and dropped**
at `stateFor` (`usgs.go:306-322`): the position becomes `DistanceKm` and a compass
word, and `snapshot.Quake` (`platform/snapshot/types.go:264-277`) carries no
latitude or longitude.

The reduction is lossy and irreversible. The ticker path keeps the epicentre
(`domains/globalfeed/usgs.go:139`); the snapshot path does not.

## The constraints any design must satisfy

| Constraint | Where it is enforced |
|---|---|
| Neither `modes/` nor `platform/` may import `domains/*` | `scripts/lint-imports.sh:44,53`, in `verify-gates` |
| Every exported snapshot field needs a JSON tag | `platform/snapshot/snapshot_test.go:158` |
| The schema is generated from the structs and is byte-compared | `pkg/schema/schema_test.go:85-99`; regenerate with `make schema` |
| Allocation budgets are pinned and gated | eight `*AllocBudget` tests; `TestBoxMemoHitAllocBudget` sits on the USGS memo path this work touches |
| A new top-level declaration re-captures the declaration set | `-update-declset`; struct **fields** do not trip it |
| Polygon coordinates never reach spoken text | `domains/radio/synth/normalize.go:130-131`, asserted at `synth_test.go:255` (UAT 81) |

## What is absent, checked rather than assumed

- No point-in-polygon logic anywhere.
- No struct-size pins (`Sizeof` has no hits in the tree).
- No geometry field on any snapshot type.
- **No documented decision to exclude alert geometry.** The omission is silent.
