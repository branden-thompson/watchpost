---
title: "Map-ready geometry — objectives"
date: 2026-09-21
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
release: 0.17.0
---

# Map-ready geometry (0.17.0)

## Why this exists

A terminal map library has been built alongside this application and is ready for
its first tag. A spike integrated it into the Observer as a window, and it drew:
the basemap, the watched place, and this application's fire detections as points.
It works.

**What it could not draw is the thing the library was built for.** Its most
designed feature is the alert area - an outline with a tint, hatched where there
is no colour, labelled, and described in words. **No host has ever driven it with
real data**, because this application's alerts carry no geometry at all.

That is not a gap in the map. It is a gap here, and it is the reason this release
exists: **the data has to be ready before the map can be integrated properly.**

## What is wrong today, in one line each

| | |
|---|---|
| **Alerts** | The per-location fetcher models only the CAP `properties` of the response. The `geometry` beside them is discarded by `encoding/json` because no field is declared for it. There is no comment, no UAT reference and no ruling recording that choice: it is undocumented |
| **Earthquakes** | The USGS reader has `[lon, lat, depthKm]`, uses it to compute a distance and a compass word, and then throws the position away. An epicentre cannot be recovered from "134 km north-north-west of somewhere" |
| **Zones** | Three alerts in four carry no polygon at all - they name zones instead. Drawing those as areas needs the zones' own shapes, which nothing here has ever fetched |

## What "done" means

1. An alert that carries a polygon can be drawn as an area.
2. An alert that names zones and carries no polygon **can also be drawn as an
   area**, from the zones' own shapes. HUM LEAD ruled this in: *"25% is not
   sufficient for integration once maps are available for a weather app, so it
   needs to work."*
3. An earthquake can be drawn where it happened.
4. None of it reaches the narration path. Polygon coordinates are stripped from
   spoken text today (UAT 81) and must stay stripped.
5. The JSON report does not grow by hundreds of thousands of numbers.

## What this release is NOT

- It does not draw anything. The map window is the next release's.
- It does not change how alerts are matched to locations. That is zone-string
  matching today and stays so: this adds shape, not attachment.
