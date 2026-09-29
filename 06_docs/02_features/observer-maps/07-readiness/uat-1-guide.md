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
| go-tuiMaps' final `v0.2.0` tag | after UAT (the station builds on the newest release candidate, `v0.2.0-rc.10` since batch 15) |

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
| Map window | `1` to `6` | Snap to a region: 1 the contiguous US, 2 Alaska, 3 Hawaii, 4 the Caribbean, 5 American Samoa, 6 Guam and the Northern Marianas (D-77); at a region's edge an arrow shows a chip naming the region beyond, and the same arrow again crosses to it (D-81) |
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
| S4 | Keys | Pan, zoom in to the limit, out to the region, `[` `]` across places in different regions; `1`-`6`; pan past each edge | Never wider than one region (D-28); an edge crosses to its neighbour in the arrangement (D-77); the map follows the selection; the keys the window owns do not reach the dashboard (D-61) |
| S5 | Sizes | Resize from full screen down to 80×24 and below, and back | Below 69×12 the notice names the size needed and the size present, and carries the description (FR-1.4); nothing overflows; nothing stale is left on screen |
| S6 | The description | Settings → Maps → Map description: with the picture, instead of it, off; then `--ascii` | Reading order (words first, D-55); the words themselves - never "you", no coordinates, units as the station's |
| S7 | Places outside the regions | Select a place in no region | The window says so, and draws nothing wider (FR-2.5) |
| S8 | Alaska, Hawaii, Puerto Rico, Guam, Samoa | Reach each by `1`-`6` and across the edges, and by selecting a place in each | The region bound holds each; marine waters are in view; the Area Alerts box names the view, not the selected place, when the place is out of view (D-78) |
| S9 | Offline | Disconnect, `g`; reconnect | What the window says it is showing (FR-3.4); the embedded national picture; recovery on reconnect |
| S10 | Settings - Maps tab | Every row: maps on/off, description, opens at, nearby, layers, detail, clear | Each change shows on the next `g`; each is still set after a restart; `←` `→` and `tab` behave as D-62 ruled |
| S11 | Alerts in view | Zoom out over a busy area; pan and zoom around the country (D-66, D-76) | Every alert in view is drawn, wherever the view is; the Area Alerts box lists them, most severe first, and "And N more in view" when full (D-78) |
| S12 | The cost warning | Switch layers on over a busy view | When it appears, what it says, whether it helps a choice |
| S16 | Alert categories and earthquakes | `O`: switch Advisories, Marine and the rest off and on over Hawaii; Earthquakes (D-80) | Each category takes only its own alerts; the choice is kept after a restart; significant quakes in view drawn as circles sized by magnitude; zooming out with categories off is quicker |
| S18 | Radar (W8, UAT-2 opens) | `g` anywhere in the lower 48, then Hawaii (3); Settings → Maps → Radar: MRMS / IEM (lower 48); `space` plays, `shift+←` `shift+→` step, `n` returns to now (D-83 to D-86) | The radar appears whole, never one frame or blinking (D-85); the timeline under the map moves with the frame, its time above the mark (D-86); the chip in the upper right names the source (MRMS green, IEM orange); the status line gives the moment, the frame and the newest frame's age, stale past ten minutes; MRMS's approximate-colours note; American Samoa says no radar covers it; alerts appear before the radar and never wait for it |
| S19 | Temperature and the two modes (W10, D-93 to D-98) | `g` in the lower 48 with radar on; `O` → Temperature on (off by default, D-99: it draws at once, held while the map is open); then `R`; in Forecast mode `space`, `shift+←` `shift+→`, `n`, `<` / `>`; `O` → Temperature off and on; Settings → Maps → Temperature: Open-Meteo (the default, D-101) / NDFD (NWS) | **Both modes:** temperature as labelled isotherms over faint bands, the key row under the map in the same faint colours with words that read on each (D-102, U2-18). **Radar mode:** the echoes in full colour over it (D-95); stepping the loop, the temperature moves on the hour and an alert issued mid-loop is absent from the earlier frames (D-96, D-98); the note credits Open-Meteo. **The rows under the map (D-103, D-105):** the colour row; the notes; the picture's status; the loop's row RADAR [source] · FRAME · NEWEST · STOPPED (yellow) or PLAYING (green) over the timeline, the Controls box beside them; MAPS: 1-6; the chips A · R · O (no Legend; L unbound); the map carries no controls box. **`R` with temperature off:** temperature turns on for Forecast mode alone and a chip at the top centre says FORECAST MODE: TEMP ENABLED until a key; back in Radar mode it is off again (D-104). The Overlays box reads MAP DETAILS / OVERLAYS with white headers and blank rows between groups. **`R`:** the chip reads Radar Off, the badge FORECAST / [O-METEO] or [ NDFD ] / the step in capitals, e.g. FRI HIGHS (U2-19); with NDFD chosen, nothing drawn past its US grid and no square patches, and Hawaii drawn from Open-Meteo, said (D-101); the bands fill, the steps scrubber replaces the loop's timeline, and the row under the map keys the temperature's colours. **Forecast mode:** Now, Today, Tomorrow and Day 3 to Day 7 step without a blink (every grid handed in at once, L-15.1); `<` or `>` flips the days between high and low, Now unchanged; a day NDFD no longer has (today's high after its daytime) is filled from Open-Meteo, the badge's chip saying so (D-100); moving through the Overlays menu never flickers the map or loses the radar (U2-13, U2-14); a watch for tomorrow is on Tomorrow and not on Now; earthquakes on Now alone. The Overlays menu no longer lists radar |
| S35 | UV and air quality (W15, D-137 to D-140) | `O` → UV on (over the lower 48 in the afternoon), then Air quality on; play the loop; `R` into Forecast mode, step the days; zoom to a city, then out; Settings → Maps → Temperature: NDFD; `S` for Status | Both off by default, and each turns Temperature, Feels like and the other off: one tint. UV: faint bands, green through yellow, orange and red to violet, contours at 3, 6, 8, 11 - 0 at night; the key row UV │ LOW ... EXTREME; Forecast mode Now and each day's highest (badge NOW UV, FRI UV). Air quality: bands green to maroon, contours at 51, 101, 151, 201, 301; AirNow's reporting areas as dots in their category's colour, "Anchorage 11" while thirty or fewer are in view; in Forecast mode today's and tomorrow's steps show AirNow's forecast, e.g. "Atascadero Good"; the badge row AIR QUALITY [  O-METEO  ]/[  AIRNOW  ]. Status names EPA AirNow and Open-Meteo Air Quality with their credits. The cost line counts AirNow's 1.9 MB file |
| S34 | Wind gusts (W13.15, D-136) | `O` → Wind on over a windy day (the West or the Plains), in Radar mode; play the loop; `R` into Forecast mode and step the days; Settings → Maps → Temperature: NDFD, then Open-Meteo; switch the units | An arrow's speed reads e.g. "15G30" where the gust beats the sustained wind by 10 mph (16 km/h), "15" where it does not - never a gust barely over the wind. Each radar frame its own hour's; Forecast mode Now's, and each day its peak sustained with its peak gust. In km/h with metric units. No new row: gusts are Wind's |
| S33 | Place names under every layer (W13.14, U2-38, D-135) | Every layer on - Buoys, Tides, Fire, Earthquakes (M1.0+), Temperature, Waves - over the lower 48, then zoom to a coast | The state and city names drawn with every layer off are still drawn with every layer on; alert words still win over them; buoy and tide readings, quake labels and contour values fill the room the names leave, and where there is none a marker or line draws without its words |
| S32 | The badge row (W13.12, U2-36, D-131 to D-133) | Every layer on in Radar mode, then `R` into Forecast mode; narrow the terminal to 80 columns; `S` for Status (the MAP block shows while a map is open; `esc` first if the map holds the key) | Under the colour row one row of badges, e.g. ALERTS [  NWS  ]    FIRE [  NIFC  ]/[  HMS  ]    QUAKES [  USGS  ] ... - **every chip, here and in the badge tab and the loop's row, "  NAME  " in bold on a colour of its own, the words black or white as reads best (D-134); try the light theme and a no-colour one too** - a layer off has none, RAIN [O-METEO] in Forecast mode alone; a second row only when narrow, no badge cut. No credit sentence under the map, no "approximate" note: the radar chip reads MRMS≈. Over the thresholds one line: "Map may experience performance issues at this zoom level. Adjust layers/zoom to improve experience."; the estimate under the timeline's right end. Status's MAP block: under Open-Meteo its credits and that each radar frame draws its own hour; under MRMS that its colours are approximate |
| S31 | The basemap always shows (W13.10, U2-34, D-129) | Every layer on - Buoys, Tides, Fire, Earthquakes (Settings → Maps → Quakes: M1.0+ past week), Waves - then `1` for the lower 48; open the map cold after Settings → Maps → Clear map data; play the loop | The basemap draws first and always - coasts, borders and names under everything, never markers over an empty frame; while it loads the frame says "Loading the map…" and nothing a programmer would read. Every buoy, tide station, fire and quake still drawn. A quake in the loop's two hours past is labelled NEW, e.g. "NEW M3.1 2:14 PM", in the past hour's colour; older ones as before. **U2-35 (D-130): the radar appears in about 1.5 s cold, 2 s at most - and `1`, `2`, `1` in quick succession draws the region last pressed, without waiting out the ones left** |
| S30 | Buoys and tides (W13.9, D-127, D-128) | `O` → Buoys on, then Tides on, over Southern California; zoom out to the coast of several states, then in to a harbour | Both off by default. Buoys: each station that read in the last two hours a pink marker labelled e.g. "4ft 73°" (waves and water), or "12kt" where it reads no waves; in metres and Celsius with metric units. Tides: each tide station in view a green marker; zoomed in to twenty or fewer, each labelled with its next high or low, e.g. "H 3.9ft 4:01 PM"; zoomed out, markers alone. Neither is ever an alert. The Status window names NOAA NDBC and CO-OPS |
| S29 | Wave height (W13.6, D-125, D-126) | `O` → Waves on over the West Coast, the Gulf and Hawaii; in Radar mode play the loop; `R` into Forecast mode, step the days | Waves off by default. On, the sea alone is tinted - never the land - in a wave scale of its own, faint, with labelled contours (2, 4, 6, 9, 13 ft, or 0.5, 1, 2, 3, 4 m by the units); near shore NDFD's numbers, farther out Open-Meteo's, the credit under the map saying both. Each radar frame its own hour; Forecast mode Now and each day's highest. The Status window's MAP block names Open-Meteo Marine |
| S28 | Errors (W13.5, U2-29, D-124) | Earthquakes on over the West with M2.5+ past week; Settings → Maps → Temperature: NDFD, then `R` into Forecast mode; `ctrl+d` for the diagnostics | The quakes draw - no note under the map. No message under the map that offers no way to fix it; where one is said it names its Setting (e.g. "Settings → Maps → Temperature: Open-Meteo draws it instead"). What the listener cannot act on is in the diagnostic dump's `map_problems`, never on the screen |
| S27 | Quakes as USGS draws them (W13.4, D-122, D-123) | Open the map over the West; zoom in and out; Settings → Maps → Quakes: M2.5+ past day, M1.0+ past week and day; `O` → Earthquakes off and on | Each quake a braille ring round its epicentre, the same size at every zoom, larger with magnitude; coloured by age - the past hour, the past day, older - labelled e.g. "M3.1 2:14 PM", or "M3.1 Sun 7:00 PM" when not today, in the listener's clock. M2.5+ over the past week by default; the other feeds as chosen, M1.0+ over a week raising the cost line (~918 KB). The Status window's MAP block names USGS |
| S26 | Fire (W13.3, D-121) | Open the map over a region with active wildfires (the West in season); zoom to a fire; `O` → Fire off and on; `R` into Forecast mode | Fire is on by default: each active perimeter an outline in fire's red-orange, holes left open; each named incident a marker labelled e.g. "Timber 12,915 ac, 26%"; the satellite hotspots as dots, the strongest bright, the rest fainter. Nothing of it reads as an alert over a place. In Forecast mode it is on Now alone. The Status window's MAP block names NIFC WFIGS and NOAA HMS |
| S25 | The badge a tab (W13.2, D-120) | Open the map in each mode; step the days; toggle Feels like; narrow the terminal to 80 columns; let the loop go stale | One line joined to the frame's top right - `┬` in the top edge above it, `└…┤` closing into the right side - reading RADAR DATA / RADAR FCST, the chip as before, the moment (STALE before it when stale), or FORECAST, the chip, the step; as wide as its words. At 80 columns the window's title ends in … before the tab, never under it |
| S24 | Feels like (W13.1, D-119) | `O` → Feels like on, then Temperature on; in Radar mode play the loop; `R` into Forecast mode, step the days, `<` `>`; Settings → Maps → Temperature: NDFD | Feels like on turns Temperature off, and Temperature on turns Feels like off: never both. Drawn as temperature is - isotherms over faint bands - each radar frame its own hour's; in Forecast mode Now and each day's feels-like high, or low with `<`. The badge's step reads e.g. TODAY FEELS HIGHS and the colour row FEELS LIKE. With NDFD chosen, Now's feels-like is Open-Meteo's (NDFD's begins at the next hour), credited |
| S23 | Rain and snow ahead beyond the lower 48, and Forecast mode's days (W12.2, W12.3, D-115 to D-118) | `[`/`]` to Alaska or Hawaii in Radar mode, `shift+→` past NOW; then `R` into Forecast mode and step the days; `O` → Rain & snow off and on, Temperature off | Beyond the lower 48 the loop runs on past NOW a frame an hour, the badge's chip O-METEO, the loop's row FORECAST O-METEO, and a note that the hours ahead are Open-Meteo's model rain, not radar. At a forecast frame the loop's row still gives the newest radar frame's age, STALE when it is. In Forecast mode, Rain & snow is on by default, over temperature's faint tint: Now the current hour, each day its heaviest hour in radar's colours, its total marked beside the arrows' spacing - rain in inches or mm, snow with a * in inches or cm. The colour row under the map reads MODEL RAIN · NOT RADAR; the credit names Open-Meteo. With Rain & snow off the row is temperature's; with temperature off the badge's step reads e.g. TODAY RAIN. Radar mode has no Rain & snow row |
| S22 | Rain and snow ahead, the loop's controls (W12.1, D-113, D-114) | Radar mode in the lower 48; `space` to play; `shift+→` past NOW; `n`; Settings → Maps → Radar ahead: 1 / 3 / 6 / 12 hours | The loop plays the past two hours and runs on past NOW into the next three at quarter-hours, in the timeline's FORECAST half; at a forecast frame the badge reads RADAR FCST with a purple HRRR chip and the loop's row leads FORECAST HRRR; `n` returns to the newest observed frame; a longer horizon lengthens the forecast half and raises the cost line. Outside the lower 48 the loop still ends at NOW until batch 34 |
| S21 | Wind (W11, D-108 to D-110) | `O` → Wind on, with Temperature on and off; `R` into Forecast mode; step the days; Settings → Maps → Temperature: NDFD | Braille arrows on an even spacing, pointing where the wind blows to, longer and brighter as it strengthens, every other with its speed in mph or km/h as the units are; over radar and over temperature's faint bands alike. In Radar mode each frame its own hour's wind; in Forecast mode Now the current wind, each day its peak (the badge's step reads e.g. THU PEAK with temperature off) and the row under the map keys the wind with temperature off. Forecast mode with wind on turns nothing else on. With NDFD, a day it lacks is Open-Meteo's, the chip saying so |
| S20 | The way back (D-106, D-107) | On the map, `]` to a place, `enter`; then `esc`; `enter` twice; `?` over the map then `g`; `esc` from the map | The place's details open over the map, and `esc` or `enter` returns to the map where it was, drawn and current; Help over the map and then `g` is the same map, not a second; `esc` from the map with nothing under it is the dashboard. The controls row above the table reads g Maps left of ↑↓ Navigate |
| S17 | Detail as a preset | Settings → Maps → Detail, then each switch; the same in `O` (D-79) | A level sets every switch; a switch on is drawn at any level (rail from county zoom, parks from state zoom, as they say); a switch changed reads "Custom" |
| S13 | ~~The legend~~ *retired (D-103): the colour rows under the map and the digits on the outlines say it* | `L` with one, then several severities drawn | It keys only what is drawn, sits over the map, `L` closes it |
| S14 | Clear map data | Settings → Maps → Clear map data, then `g` | What it says it removed; the next open is cold again |
| S15 | Long running | Leave the map open through an alert's expiry and a new one's arrival | The expired area goes; the new one arrives without a key press; nothing stale remains |

## Owed to the HUM LEAD alongside UAT-1

- **The M1 answer key's confirmation** (W0.2): the recorded scenarios' expected words, which
  `TestTheDescriptionAnswersTheM1Key` holds the description to.
- **M1b re-scored on this description** (W9.2's test, D-53).
- **"A refresh", as built for the cost warning** (build log, batch 9): each layer's own fetches with
  nothing held; the basemap not counted.
