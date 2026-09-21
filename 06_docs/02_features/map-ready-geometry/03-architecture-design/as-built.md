---
title: "Map-ready geometry — as built"
date: 2026-09-21
phase: BUILD exit
sev: SEV-0
release: 0.17.0
---

# As built

**This is what 0.17.0 actually does**, drawn from the code rather than from the
plan. Where it differs from [the plan](plan.md), the difference is named.

## The whole of it

```mermaid
flowchart TB
    API[("api.weather.gov")]

    subgraph DOM["domains/weather/nws"]
      direction TB
      AL["<b>alerts.go</b><br/>alertsPayload now declares <b>geometry</b><br/>alertFrom reads it · mapAlert matches on the FULL zone list (R3-A-01)"]
      ZN["<b>zones/zones.go</b><br/>Zone · Zones · Seed<br/>one fetch per id, kept whole WITH its name<br/><i>no cache of its own</i>"]
    end

    subgraph SEIS["domains/seismic/usgs"]
      US["<b>usgs.go:stateFor</b><br/>keeps Lat and Lon beside the distance<br/>and compass word computed from them"]
    end

    subgraph PLAT["platform/"]
      direction TB
      GEO["<b>geo</b><br/>Point · Ring · Shape<br/><b>ReadGeometry</b> — token-streamed, never recursed, depth-capped"]
      HTTP["<b>httpx</b><br/>the freshness the server declares · disk tier · conditional refresh<br/><i>MG-11 and MG-13 were already here</i>"]
      SNAP["<b>snapshot</b><br/>Alert.Area · Quake.Lat/Lon<br/>schema <b>1.1.0-rc</b>"]
    end

    APP["<b>app/mapgeometry.go:resolveAlertAreas</b><br/>its own polygon, or its zones' outlines — the same thing either way<br/>the only place that may name a domain AND what draws"]
    NEXT["platform/tuimap — <b>0.18.0</b>, not in this release"]

    API --> AL --> SNAP
    API --> ZN
    API --> US --> SNAP
    AL -. "reads with" .-> GEO
    ZN -. "reads with" .-> GEO
    ZN -. "fetches through" .-> HTTP
    AL -. "fetches through" .-> HTTP
    SNAP --> APP
    ZN --> APP
    APP -. "hands plain shapes to" .-> NEXT
```

## Where an alert's area comes from

```mermaid
flowchart LR
    A["An alert"] --> Q{"Does it carry<br/>its own polygon?"}
    Q -- "yes — <b>one in five</b>" --> OWN["Use it.<br/>Nothing is fetched"]
    Q -- "no — <b>four in five</b>" --> Z["The zones it names"]
    Z --> S{"Held already?"}
    S -- yes --> USE["Use the outlines"]
    S -- "no" --> F["Fetch once per distinct id<br/>(the same zone is named by many alerts)"]
    F -- ok --> USE
    F -- "could not be got" --> NONE["The alert resolves to nothing.<br/><b>Not an error here</b>: whether to draw a<br/>partly-known area needs a view, and<br/>this layer has none (MG-10)"]
```

## What differs from the plan, and why

| Planned | As built | Why |
|---|---|---|
| `platform/geo` gains Douglas-Peucker; shapes simplified at ingest to 0.5 km | **No simplification anywhere** | The map library rules that hosts pass full detail and it simplifies per zoom; 0.5 km is visible from zoom 10 and it draws to 18. Red-team round 1 (MG-6) |
| The zone store keeps a disk cache, a freshness rule and a revalidation path | **It keeps only parsed shapes** | All three were already in `platform/httpx`. Found while building p3, not while reviewing |
| An alert resolves whole or not at all | **It reports what it has** | Whole-or-nothing is a drawing decision and had been put in the data layer, where there is no view. It also cancelled the seeding ruling. Red-team round 2 (MG-10) |

## What this release does NOT do

- **It draws nothing.** `resolveAlertAreas` is where 0.18.0 begins.
- It does not change how alerts are matched to locations: that is zone-string
  matching, unchanged.
- It does not resolve a zone id to a *county* zone; `/zones/county/<id>` answers
  404 for a forecast-zone id, and county zones carry their own identifiers.
