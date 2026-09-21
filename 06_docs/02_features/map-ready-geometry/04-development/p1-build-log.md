---
title: "p1 — the shape and the reader"
date: 2026-09-21
phase: BUILD
sev: SEV-0
release: 0.17.0
---

# p1 — the shape, and the reader

| Task | The test written first | What it taught |
|---|---|---|
| **1.1** `platform/geo` gains `Point`, `Ring`, `Shape` | a ring knows whether it is closed; a shape keeps its rings in order; the zero shape is usable | Nothing surprising. The types are deliberately thin - no simplification, no area, no containment - because the map library rules that hosts pass full detail and it does the simplifying (MG-6) |
| **1.2** A bounded GeoJSON reader | the three shapes the service sends; an absent geometry is not an error; **twenty thousand levels of nesting are refused**; damaged documents are errors, never guesses; a document of more than fifty thousand positions is refused | **An off-by-one in the depth accounting**, and it is the kind that would have passed a careless review: a position's array closes *at* the depth its own opening bracket counted, not one below it. The synthetic tests caught it because they assert vertex counts rather than "no error" |

## What is in the reader, and why

It streams tokens and **never recurses**. The interface decode path recurses
once per array level with no cap, so a hostile `coordinates` would overflow the
stack and take the process down - the red-team finding 0.12.0 P4 F2 that
`domains/globalfeed`'s point reader was written against. The depth here is a
counter with a ceiling of eight; GeoJSON needs four.

The depth positions sit at is taken **from the declared type**, not guessed by
looking. A `Polygon` read at a `MultiPolygon`'s depth would silently produce the
wrong shape rather than an error, which is the worst kind of wrong.

## Checked against something other than itself

The synthetic tests say the reader agrees with its author. These say it agrees
with the service:

| Fixture | Rings | Positions | Counted independently |
|---|---|---|---|
| `zone-dallas.geojson` (TXZ119) | 1 | 80 | 80 |
| `zone-glacier-bay.geojson` (AKZ320) | **32** | **12,004** | 12,004 |

Glacier Bay is the tail, not the median - a marine zone a hundred and fifty
times the size of a typical county - and it is committed precisely because it is
the case a store built on this has to survive. It reads at 0.6% of the map
library's per-overlay budget, which is why nothing here simplifies it.

## Gates

`gofmt`, `go vet`, `lint-imports` and the full test tree pass. `platform/geo`
carries no declaration set, so there is nothing to re-capture.
