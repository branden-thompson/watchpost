---
title: "0.18.0 Observer maps — UAT-1 guide: alert areas, the description, the window (W1–W5)"
date: 2026-09-25
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "OPEN — P1-a built (batches 1–11). UAT is expected to be long and to change the feature; every finding goes in uat-1-findings.md with a number, and each disposition is the HUM LEAD's."
---

# UAT-1: what is in it, and how to run it

**UAT-1 is P1-a: the map window, the basemap, the region bound, alert areas and their notes, the
description, the window's keys, and the Maps tab in Settings (W1–W5, with W9 folded, D-60).** Radar is
UAT-2 (W8), with the rest of W9. The build log (`../04-development/build-log.md`) says what each batch
built, and `../03-architecture-design/as-built-map.md` draws it.

**Every finding goes in `uat-1-findings.md`**, numbered, in the HUM LEAD's words where they are the HUM
LEAD's. The agent has seeded the log with what the build itself showed, marked as seeded; none of those is
a finding until the HUM LEAD sees it.

## Not in UAT-1

| | Where it lands |
|---|---|
| Radar, its loop, playback keys, the radar step Setting | W8 (UAT-2) |
| The map in the theme's colours; the colour-depth hint; the colour-vision matrix | W7 |
| Fire hotspots, incidents and quakes as layers | W6 (the registry takes them without editing the others) |
| The map from the Details window | phase 2 (D-33) |
| go-tuiMaps' final `v0.2.0` tag | after UAT (the station builds on `v0.2.0-rc.8`) |

## Build and run

```sh
cd watchpost
git switch feature/map-drawing && git pull
make build
./dist/watchpost            # Observer
./dist/watchpost --ascii    # the description in place of the picture (FR-1.7, FR-1.8)
```

- **Settings are written to** `~/.config/watchpost/config.toml` (`maps`, `map_description`, `map_scale`,
  `map_nearby_km`, `map_alert_scope`, `map_layers`).
- **The map's tile cache is** `~/Library/Caches/watchpost/map` (256 MiB, 7 days). **Settings → Maps →
  Clear map data** empties it and the zone outlines; to start truly cold, clear it before a scenario.
- **Nothing is fetched for the map until `g`** (D-21, D-25): the first open of a session says what it sends,
  and to whom (FR-9.4).

## The keys

| Where | Key | What it does |
|---|---|---|
| Dashboard | `g` | Opens the map on the selected location; `g` or `esc` closes it |
| Map window | `←` `↑` `→` `↓` | Pan a quarter of the view |
| Map window | `+` / `=`, `-` | Zoom in, out (never wider than the region) |
| Map window | `[` `]` | Previous, next location |
| Map window | `PgUp` `PgDn` | Scroll the window's body when it is longer than the window |
| Map window | `L` | The legend over the map's corner (D-44) |
| Map window | `c` | Units, as on the dashboard |
| Settings | `s`, then `tab` / `shift+tab` or `←` `→` | The Maps tab (D-62); a focused picker keeps `←` `→` |

## Scenarios

Each has what to do and what to look at. **"Look at" is not the pass condition** — anything that reads
wrong, looks wrong or feels wrong is a finding.

| # | Scenario | Do | Look at |
|---|---|---|---|
| S1 | First open | Fresh start, clear map data, select a watched place, `g` | The disclosure above the map; how long until the picture is whole; the status line while loading |
| S2 | A place under an alert | Select a location with an active alert, `g` | The area drawn; its severity digit on the outline; the description's first sentence (covers / stops short / lies to one side, how far, which way, until when) |
| S3 | A partly resolved alert | A watch or warning whose zones did not all arrive (or go offline mid-fetch) | The note under the map naming what is missing, in words; whether the description says the place is covered by a zone that could not be drawn |
| S4 | Keys | Pan, zoom in to the limit, out to the region, `[` `]` across places in different regions | Never wider than the region (D-28); the map follows the selection; the keys the window owns do not reach the dashboard (D-61) |
| S5 | Sizes | Resize from full screen down to 80×24 and below, and back | Below 69×12 the notice names the size needed and the size present, and carries the description (FR-1.4); nothing overflows; nothing stale is left on screen |
| S6 | The description | Settings → Maps → Map description: with the picture, instead of it, off; then `--ascii` | Reading order (words first, D-55); the words themselves - never "you", no coordinates, units as the station's |
| S7 | Places outside the regions | Select a place in no region | The window says so, and draws nothing wider (FR-2.5) |
| S8 | Alaska, Hawaii, Puerto Rico, Guam, Samoa | Select a place in each | The region bound holds each; marine waters are in view |
| S9 | Offline | Disconnect, `g`; reconnect | What the window says it is showing (FR-3.4); the embedded national picture; recovery on reconnect |
| S10 | Settings - Maps tab | Every row: maps on/off, description, default scale, nearby, alerts scope, layers, clear | Each change shows on the next `g`; each is still set after a restart; `←` `→` and `tab` behave as D-62 ruled |
| S11 | National scope | Alerts → "Plus national severe events" on a day with severe weather | The region's warnings and watches appear, and nothing outside the region; the cost warning if the region is busy |
| S12 | The cost warning | Choose the national scope on a busy day | When it appears, what it says, whether it helps a choice |
| S13 | The legend | `L` with one, then several severities drawn | It keys only what is drawn, sits over the map, `L` closes it |
| S14 | Clear map data | Settings → Maps → Clear map data, then `g` | What it says it removed; the next open is cold again |
| S15 | Long running | Leave the map open through an alert's expiry and a new one's arrival | The expired area goes; the new one arrives without a key press; nothing stale remains |

## Owed to the HUM LEAD alongside UAT-1

- **The M1 answer key's confirmation** (W0.2): the recorded scenarios' expected words, which
  `TestTheDescriptionAnswersTheM1Key` holds the description to.
- **M1b re-scored on this description** (W9.2's test, D-53).
- **"A refresh", as built for the cost warning** (build log, batch 9): each layer's own fetches with
  nothing held; the basemap not counted.
