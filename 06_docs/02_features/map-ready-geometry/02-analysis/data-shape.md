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

## Measured against the live service, 2026-09-21

The fixture above is a trimmed sixteen. These are the real numbers, from
`api.weather.gov` with this application's own identifying user-agent.

### Every active alert in the country, in one request

1,516 KB in 511 ms, **337 active alerts**:

| | |
|---|---|
| carrying no polygon | **274 - 81%** |
| carrying a polygon | 63 |
| polygon vertices | median **7**, max 21 |
| zones named per alert | median **1**, 95th percentile **5**, max **42** |
| distinct zones in play nationally | 477 |

**MG-1 is now measured rather than argued.** Four alerts in five carry no
polygon. Drawing only the fifth would leave a weather map blank during most of
what it exists to show.

**And the median alert names one zone.** The eighty-one of the fixture was an
outlier; live, the 95th percentile is five. The tail is real but rare, and it is
the tail a design has to survive, not the case it should be built around.

### What a zone shape actually costs

| zone | bytes | fetch | rings | vertices |
|---|---|---|---|---|
| INZ018 Allen, Indiana | 9 KB | 163 ms | 1 | 74 |
| OHZ001 Williams, Ohio | 6 KB | 121 ms | 1 | 48 |
| TXZ119 Dallas, Texas | 10 KB | 110 ms | 1 | 80 |
| **AKZ320 Glacier Bay, Alaska** | **1,393 KB** | 224 ms | **32** | **12,004** |

A typical inland zone is as cheap as an alert polygon. A coastal Alaskan zone is
**a hundred and fifty times larger**. The tail is the design problem; the median
is not.

*(`/zones/county/<id>` answers 404 for a forecast-zone id - county zones carry
their own identifiers. Worth knowing before a fetcher is written against both.)*

### What simplification does to it

Douglas-Peucker, at tolerances below what one braille dot can resolve:

| zone | raw | 0.5 km | 1.0 km | 2.0 km | 5.0 km |
|---|---|---|---|---|---|
| Glacier Bay | 12,004 | **744** | 428 | 241 | 136 |
| Dallas | 80 | **6** | 6 | 6 | 6 |

**At half a kilometre the worst zone loses 94% of its vertices and the typical
one loses 92%**, because the source carries collinear detail no terminal can
draw. Simplifying at ingest is not an optimisation here; it is the difference
between storing megabytes and storing kilobytes.

A rough figure for the whole country, from these: the United States has on the
order of 3,500 forecast zones; at fifty simplified vertices each that is a few
megabytes in total. **Seeding the whole set is therefore possible**, which makes
the cold-cache risk MG-R1 a choice rather than a constraint.
