---
title: "0.18.0 Observer maps — AS BUILT: where the map lives"
date: 2026-09-25
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "LIVE — redrawn with every BUILD batch that moves a part. Batches 1–13 (W1.1–W1.8, W1.10–W1.17, W2, W3.1–W3.9, W4, W5 with W9.1–W9.5 folded; Settings in tabs, D-62; the layer registry, the cost warning, the alert scope) - P1-a complete, UAT-1 open; batches 12 and 13 are its first two passes (D-63 to D-71); go-tuiMaps v0.2.0-rc.9."
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
    MK["map keymap scope\narrows pan · + − zoom · [ ] place · PgUp/PgDn scroll\nA Area Alerts · O Overlays · L legend"]
    BX["map_boxes.go · over the map (UAT-1)\nArea Alerts, upper left (D-63) · Controls, lower right, the pressed chip blinks\nOverlays menu: weather layers + the map's detail (D-65, Map.Layers)\nthe title names the view by scale (D-64)"]
    LG["the legend (D-44)\na box over the map's corner, from Legend()\nonly the severities drawn, with their digits"]
    ST2["Settings, the Maps tab (D-62), two columns (U1-25)\nMAP: Maps on/off · what the map sends (D-69: said here alone) · Map description · Default scale · Nearby · Alerts\nMAP - LAYERS AND DETAIL: Layers (+ the cost warning) · Detail level (D-67) · the detail list · Clear map data · the retention"]
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
  MW --> BX
  NM["mapnames.go · mapAreaNamer\nthe city index: a town · part of a state · the state · the region"] -- "Config.MapAreaName" --> BX
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
  DS -- "yes (the default)" --> D1["the description in the Area Alerts box on the first rows (D-55 as D-63 keeps it)"] --> M
  DS -- "off" --> M["the map, bound to the region (FR-2.1)\nthe window about 80% of the terminal each way (U1-13), never under what 69x12 needs"]
  M --> BXS["the boxes over it: Area Alerts, the Overlays menu, the controls, the legend"] --> NT["the feed's notes: partial areas named in words (FR-4.4)"]
  NT --> CW["the cost warning, when the layers on would cost more than 2 MB or 40 requests a refresh (FR-9.2)"]
  CW --> ST["the status line: loading · offline · coarser · blank when whole (FR-3.4)\nPgUp and PgDn scroll the body when it is longer than the window"]
```

## The map's commands, its clock and its close (W2)

```mermaid
flowchart LR
  U["Update (the Bubble Tea goroutine)\nevery library call but Work"] -- "Pending > 0" --> W["Work command\nadmitted by mapWorkers"]
  U -- "the feed asked" --> F["feed command\nadmitted by mapWorkers"]
  W -- "mapWorkedMsg" --> U
  F -- "mapFeedMsg" --> U
  U -- "after every Update: NextCall" --> T["one tick outstanding\n(50 ms floor)"]
  T -- "mapTickMsg at the time still wanted" --> U
  Q["the program ends\ncloseOnExit → Router.CloseMap"] --> C["closeMap\ncancel · join (≤ 2 s) · Close"]
  C -. "refuses what starts after" .-> W & F
```

## Settings, in tabs (D-62)

```mermaid
flowchart LR
  subgraph S["Settings: one window, the tab is the focused row's"]
    DA["Data (D-71)\nDATA · ALERTS - EVENTS"]
    UI["Watchpost UI (D-71)\nWATCHPOST UI"]
    R["Watchpost Radio\nALERTS - TONE · CORRESPONDENTS · RELAY REPLAY"]
    B["Broadcaster\nSTATION: transmitter · service radius"]
    M["Maps\nMAP | MAP - LAYERS AND DETAIL, side by side"]
  end
  O(["Observer"]) --> DA & UI & R & B & M
  C(["the console"]) --> DA & UI & R & B & M
  N["D-70: every surface shows every tab and row;\neach row still writes only its own values"] -.-> O & C
  K["keys: ←→ switch tabs unless the focused row is a picker or toggle\ntab / shift+tab switch tabs from any row · ↑↓ walk a tab's rows"] -.-> S
```

Each tab fits unscrolled at 133×44; the window is as wide as its widest tab on every tab. Within a column the
labels and the pickers' values are padded to one width, so the arrows line up (U1-24).

## Not built yet

The theme's palette and the colour-depth hint (W7), which bring W2.3's theme and depth rows; radar (W8),
which registers its sources and its layer and brings the frame-advance rows.
