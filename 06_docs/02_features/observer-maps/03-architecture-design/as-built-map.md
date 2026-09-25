---
title: "0.18.0 Observer maps — AS BUILT: where the map lives"
date: 2026-09-25
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "LIVE — redrawn with every BUILD batch that moves a part. Batches 1–10 (W1.1–W1.8, W1.10–W1.17, W2.1, W3.1–W3.9, W4, W5 with W9.1–W9.5 folded; Settings in tabs, D-62; the layer registry, the cost warning, the alert scope)."
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
    MB["maps.go · mapBuilder\nthe registered basemap (OpenFreeMap)\nwatchpost/‹version›, 256 MiB, 7 days\n4 MiB shared memory cache\nseeds zone outlines on the first map"]
    REG["maplayers.go · the registry (W1.13)\nlayers and sources register from their own files\nalert areas (mapfeed.go) · OpenFreeMap (maps.go)\nrefreshCost: each layer's own, summed"]
    MF["mapfeed.go · mapFeed\none overlay per alert, severity → role\npartial areas labelled, notes in words\nwhich alerts' missing zones hold the place"]
    MG["mapgeometry.go · resolveAlertAreas"]
    MN["mapnational.go · nationalAlerts\nthe national scope: the ticker's severe events\nin the selected place's region, as alerts\n(by point, or by zone code)"]
  end
  subgraph domains["domains/"]
    ZS["nws/zones · Store\n512 at once, reported past it; six in flight (W3.9)\nshapes refetched after 7 days"]
    WS["nws · Provider.ZonesFor\nthe place's own zone codes"]
  end
  subgraph tty["modes/tty — the Observer"]
    MW["map_pane.go · the map window\ng opens at the chosen scale, nearby as chosen\nowns its keys while open (D-61)\ndraws in Update, View prints (D-41)\nunits follow the station's\na layer off: its overlays not set (key before the slash)"]
    MD["map_describe.go · the description\nReport joined to watchpost's alerts by id\ncovers · stops short · lies to one side (M1's words)\nunits and directions in words; never 'you'"]
    MK["map keymap scope\narrows pan · + − zoom · [ ] place · PgUp/PgDn scroll · L legend"]
    LG["the legend (D-44)\na box over the map's corner, from Legend()\nonly the severities drawn, with their digits"]
    ST2["Settings, the Maps tab (D-62)\nMaps on/off · what the map sends · Map description\nDefault scale · Nearby · Alerts (scope) · Layers (+ the cost warning)\nClear map data · the retention"]
  end
  subgraph plat["platform/"]
    RG["geo · RegionOf\nsix regions with their waters"]
    HX["httpx · DiskCacheBytes\nin the one stated total"]
  end
  L["go-tuiMaps v0.2.0-rc.N\nSetBound · Source · Set/Remove · Work · Render · Warnings"]
  MB -- "Config.NewMap(size)" --> MW
  MF -- "Config.MapFeed(ask: snap, place, scope)" --> MW
  MF --> MG --> ZS
  MF -- "national scope" --> MN
  SD["severe.go · severeDeck\nthe ticker's national feed (polygons kept)"] -- "nationalFeed()" --> MN
  MF --> WS
  MB -. "first g" .-> ZS
  MW --> RG
  MW --> MK
  MW --> LG
  ST2 --> MW
  ST2 -- "Config.ClearMapData" --> CL["app clearMapData\nPurge on a bare map (no source, no seed)\nzone store Forget + ForgetCached\n(httpx ForgetPrefix: zones only)"]
  MW -- "Report(place) at every draw" --> MD
  MW --> L
  MB --> L
  MB --> HX
  MF -. "registers alert areas" .-> REG
  MB -. "registers OpenFreeMap" .-> REG
  REG -- "Config.MapLayers · Config.MapCost" --> ST2
  REG -- "Config.MapCost" --> MW
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
  NT --> CW["the cost warning, when the layers on would cost more than 2 MB or 40 requests a refresh (FR-9.2)"]
  CW --> ST["the status line: loading · offline · coarser · blank when whole (FR-3.4)\nPgUp and PgDn scroll the body when it is longer than the window"]
```

## Settings, in tabs (D-62)

```mermaid
flowchart LR
  subgraph S["Settings: one window, the tab is the focused row's"]
    G["General\nDATA · WATCHPOST UI · ALERTS - EVENTS"]
    R["Watchpost Radio\nALERTS - TONE · CORRESPONDENTS · RELAY REPLAY"]
    B["Broadcaster\nSTATION: transmitter · service radius"]
    M["Maps\nMAP: on/off · description · scale · nearby · alerts · layers · clear"]
  end
  O(["Observer"]) --> G & R & M
  C(["the console"]) --> G & R & B
  K["keys: ←→ switch tabs unless the focused row is a picker or toggle\ntab / shift+tab switch tabs from any row · ↑↓ walk a tab's rows"] -.-> S
```

Each tab fits unscrolled at 133×44; the window is as wide as its widest tab on every tab.

## Not built yet

The `NextCall` tick and the rest of W2 (the goroutine record, the frame guards, join on close, the
allocation pin, the width goldens); radar (W8), which registers its sources and its layer.
