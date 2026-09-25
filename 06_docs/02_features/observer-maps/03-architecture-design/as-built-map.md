---
title: "0.18.0 Observer maps — AS BUILT: where the map lives"
date: 2026-09-25
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "LIVE — redrawn with every BUILD batch that moves a part. Batches 1–7 (W1.1–W1.8, W1.10, W1.15, W1.16, W2.1, W3.1–W3.7, W4, W5 with W9.1–W9.4 folded)."
---

# As built: where the map lives

**This page is kept in step with the code, batch by batch** (the build log, `04-development/build-log.md`,
says what each batch moved). The atlas draws from its diagrams (`go run ./tools/atlas`).

## The parts, and who may name whom

`app/` is the only package that may name a domain and what draws (the import rule,
`scripts/lint-imports.sh`), so the zone store and the weather service are reached from `app/`
alone, and the window is handed functions.

```mermaid
flowchart LR
  subgraph app["app/ — the composition root"]
    MB["maps.go · mapBuilder\nthe closed list's basemap (OpenFreeMap)\nwatchpost/‹version›, 256 MiB, 7 days\n4 MiB shared memory cache · nearby 15 km (M1's rule)\nseeds zone outlines on the first map"]
    MF["mapfeed.go · mapFeed\none overlay per alert, severity → role\npartial areas labelled, notes in words\nwhich alerts' missing zones hold the place"]
    MG["mapgeometry.go · resolveAlertAreas"]
  end
  subgraph domains["domains/"]
    ZS["nws/zones · Store\n512 at once, reported past it\nshapes refetched after 7 days"]
    WS["nws · Provider.ZonesFor\nthe place's own zone codes"]
  end
  subgraph tty["modes/tty — the Observer"]
    MW["map_pane.go · the map window\ng opens · owns its keys while open (D-61)\ndraws in Update, View prints (D-41)\nunits follow the station's"]
    MD["map_describe.go · the description\nReport joined to watchpost's alerts by id\ncovers · stops short · lies to one side (M1's words)\nunits and directions in words; never 'you'"]
    MK["map keymap scope\narrows pan · + − zoom · [ ] place · PgUp/PgDn scroll"]
    ST2["Settings, WATCHPOST UI\nMaps on/off · Map description: with / instead / off\nwritten with the group (config maps, map_description)"]
  end
  subgraph plat["platform/"]
    RG["geo · RegionOf\nsix regions with their waters"]
    HX["httpx · DiskCacheBytes\nin the one stated total"]
  end
  L["go-tuiMaps v0.2.0-rc.N\nSetBound · Source · Set/Remove · Work · Render · Warnings"]
  MB -- "Config.NewMap(size)" --> MW
  MF -- "Config.MapFeed(snap, place)" --> MW
  MF --> MG --> ZS
  MF --> WS
  MB -. "first g" .-> ZS
  MW --> RG
  MW --> MK
  ST2 --> MW
  MW -- "Report(place) at every draw" --> MD
  MW --> L
  MB --> L
  MB --> HX
```

## What the window's body is

```mermaid
flowchart TB
  S{"selected place?"} -- "none" --> N["stated: no location is selected (FR-1.3)"]
  S -- "in no region" --> O["stated: outside every region, nothing wider drawn (FR-2.5)"]
  S -- "maps off in Settings" --> OFF["stated: maps are off, and where to turn them on (FR-1.6)"]
  S -- "in a region" --> A{"--ascii, or the description instead?"}
  A -- "yes" --> T["the description in place of the picture (FR-1.7)\n(under --ascii, first: braille and the remedy, FR-1.8)"]
  A -- "no" --> F{"a 69x12 map fits? (mapMinBody, the one owner)"}
  F -- "no" --> FL["stated: the size needed and the size present,\nthen the description (FR-1.4)"]
  F -- "yes" --> DS{"description with the picture?"}
  DS -- "yes (the default)" --> D1["the description first in reading order (D-55)"] --> M
  DS -- "off" --> M["the map, bound to the region (FR-2.1)\nthe window 8 rows short of the terminal, full width less its frame"]
  M --> NT["the feed's notes: partial areas named in words (FR-4.4)"]
  NT --> ST["the status line: loading · offline · coarser · blank when whole (FR-3.4)\nPgUp and PgDn scroll the body when it is longer than the window"]
```

## Not built yet

The `NextCall` tick; the remaining Settings rows (W1.11: default scale, layers, alert scope, nearby); the exposure
disclosure, the registry and the cost warning (W1.12–W1.14); the legend (W1.17); the national
scope (W5.3's second half); the clear path on the library's `Purge` (W3.8 with W9.5); radar (W8).
