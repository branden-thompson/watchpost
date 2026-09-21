---
title: "p2 — the data that already existed and was thrown away"
date: 2026-09-21
phase: BUILD
sev: SEV-0
release: 0.17.0
---

# p2 — the data that already existed and was thrown away

Nothing in this phase fetches anything new. Every number here was already
arriving and being discarded.

| Task | The test written first | What it taught |
|---|---|---|
| **2.2** the quake keeps its epicentre | the position survives into the snapshot, and distance and bearing are unchanged because they are computed from the same two numbers | The position reached `stateFor`, was used, and was dropped on the next line. Two fields and one line of assignment |
| **2.1** the alert keeps its own polygon | an alert with a polygon keeps its four positions and its ring is closed; **a zone-only alert carries none and that is not an error** | The response struct declared only `properties`, so the geometry was discarded by `encoding/json` with nothing recording it. Declaring the field is the whole fix |
| **2.3** the schema is regenerated and bumped | the published schema matches the generator; a real envelope validates | **See below - the bump was broken before it was used** |
| **2.4** an alert's area is never spoken | an alert with a polygon is narrated, and none of its coordinates' digits reach the air | The composer reads `Headline` and `Description` **by name**, so the area cannot reach speech today. The test is a guard against a later change that walked the fields, not a discovery |

## A seam a test could not drive

The decode happened inline inside `fetchAlerts`, so nothing could exercise it
without a network. **P-1 says a seam a test cannot drive is not a covered
seam**, so the decode became `decodeAlerts`, and the construction of one alert
became `alertFrom`. The fetcher is unchanged in what it does.

That split had to keep an invariant that is easy to lose: **the zone match runs
over the full list, while only the stored copy is bounded** (0.13.0 red-team
R3-A-01). A Winter Storm Warning can span eighty zones and a watched location's
may be the sixtieth, so matching on the clamped copy would drop the alert for
exactly the locations furthest down it. `mapAlert` now matches on the raw list
and `alertFrom` clamps what is kept, and the comment says why.

## The schema bump was broken before anyone used it

`make schema` wrote to a **hardcoded** `v1.0.0-rc` filename while the test reads
`"watchpost-report.v" + snapshot.SchemaVersion + ".schema.json"`. They agree
only until someone bumps the version - so the first bump wrote new content into
the old name and the test looked for a file that did not exist.

Fixed rather than worked around: the target now derives the name from the one
place the version is defined, and an assertion that hardcoded `1.0.0-rc` reads
the constant instead.

## Gates

| Gate | Result |
|---|---|
| `gofmt`, `go vet`, `lint-imports` | pass |
| `alloc-budget` | pass, **including `TestBoxMemoHitAllocBudget`**, which sits on the USGS memo path this phase changed |
| `declset` (`domains/weather/nws`) | re-captured: `alertFrom`, `decodeAlerts`, `alertFeature`, `alertsPayload` are the seam P-1 asked for |
| `pkg/schema` | regenerated at **1.1.0-rc** |
| `modes/report` golden | re-captured. **One line changed**: the header names the schema version |
| full test tree | pass |
