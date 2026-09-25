---
title: "0.18.0 Observer maps — UAT-1 findings log"
date: 2026-09-25
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "OPEN — one row per finding, numbered, never renumbered. A disposition is the HUM LEAD's; a fix names its batch."
---

# UAT-1 findings

**How a row is written.** The number never changes. *Seen* is what was observed, in the HUM LEAD's words
where they are the HUM LEAD's. *Disposition* is the HUM LEAD's: fix now, fix later (with where), not a
defect, or ruled (with the D-number). *Fixed in* names the batch and its test.

**Rows marked SEEDED were found by the agent while building** and are listed so they are not lost; each
is a finding only once the HUM LEAD has seen it.

| # | Scenario | Seen | Disposition | Fixed in |
|---|---|---|---|---|
| U1-1 | S5 | SEEDED (80×24 golden, batch 11): the library's footer runs the scale bar into the credit - "├──────────┤ 50 kmOpenFreeMap (c) OpenMapTiles…" - no space between them. A go-tuiMaps footer question. | Fix in go-tuiMaps v0.2.0 (D-68) - agent, to confirm | |
| U1-2 | S5 | SEEDED (80×24 golden): the status line is cut mid-sentence - "This is a coarser picture: the map has no finer tiles for" - at 80 columns. | Fixed with U1-7's chips (agent, to confirm) | Batch 12: the status and the chips on two lines; the 80×24 golden reads whole |
| U1-3 | S10 | SEEDED: the map is drawn in the library's own colours, not the theme's; the theme's palette is W7's. | | |
| U1-4 | S12 | SEEDED (batch 9): "a refresh" was read as each layer's own fetches with nothing held, the basemap not counted; zones at a measured 10 KB. To be confirmed. | | |
| U1-5 | S1 | SEEDED (batch 11): the map window's frame costs 2,508 allocations a memo hit, against the severe window's 1,740 - pinned, not optimised (D-53). Only a finding if the window feels slow. | | |
| U1-6 | all | HUM LEAD, 2026-09-25: "Overall functionality is good." (first run, no feedback yet: "This is pretty impressive for 1st run … This is very good.") | Noted | - |
| U1-7 | S2, S6 | HUM LEAD: "The text on the top probably needs to go somewhere else, and likely should be 'legend-style' modal that can \"picture in picture\" over the map. I think it should be [A] Area Alerts and should PIP on the upper left (Legend is on upper right)" | Fix now; open by default (D-63) | Batch 12: `TestTheAreaAlertsBoxOpensWithTheMap` |
| U1-8 | S2 | HUM LEAD: "Tint is good (Looking at Beach Hazard Statement in Oceanside CA)" | Noted | - |
| U1-9 | S10 | HUM LEAD: "I should likely have a menu for overlays - like Radar" | Fix now; the [O] Overlays menu (D-65) | Batch 12: `TestTheOverlaysMenuSwitchesLayersAndDetail` |
| U1-10 | S4 | HUM LEAD: "Lots of extra details in the maps - which for watchpost may not be needed (reservation outlines, etc), we may need to provide some options to reduce some of that detail to make space for the weather / alerts / etc. This is a WATCHPOST focused feature, but may need tuiMaps to expose some controls to help with render details of 'roads' etc" | Fix now; weather-first detail defaults, switchable in [O] (D-65) - the library already switches its layers, so no go-tuiMaps change | Batch 12: `TestTheOverlaysMenuSwitchesLayersAndDetail`, `TestADetailSwitchChangesThePictureAndIsSaved` |
| U1-11 | S4 | HUM LEAD: "Control surfaces for the map (left/right/up/down +/-, etc) likely needs a 'control modal picture in picture' on the lower right of the map, which should provide visual feedback like our other chip controls (if I press <-, then that chip 'blinks' to show it was pressed)" | Fix now | Batch 12: `TestTheControlsBoxBlinksThePressedKey` |
| U1-12 | S4 | HUM LEAD: "I like the 'Map · <location>' title - however when I start to zoom out or focus on something else, that title should update to reflect where I am. If we zoom out to an area like 'Southern California' (I can see San Diego, Oceanside, LA, etc) - then the title should reflect that we're now looking at a region and not a specific city." | Fix now; by scale, from the city index (D-64) | Batch 12: `TestTheTitleNamesWhatIsInView`, `TestTheTitleNamesTheViewByScale` |
| U1-13 | S5 | HUM LEAD: "Right now the map takes the WHOLE screen. I'd like that to be reduced so that it fills approx. 80% of the terminal view so users know that the dashboard is still behind the maps modal. A user might be confused that this is a completely different view/UI apart of Observer, which is not the case." | Fix now | Batch 12: `TestTheMapWindowLeavesTheDashboardInSight` |
| U1-14 | S2 | HUM LEAD: "Also only looks like Oceanside CA alerts are being drawn - when I went to differnt locations like New York, NY and hit 'g' - none of the alert geometries were drawn" | Fix now (a defect) | Batch 12: `TestARecentPlacesAlertsReachTheMap` |
| U1-15 | S4, S11 | HUM LEAD: "I should also see alerts as I navigate around the map - which is what I human user would expect as they \"explore\" the weather map. This *could* mean a lot of alerts, which is why we want to prioritize drawing weather and geography over roads (this isn't a nav app, so major roadways are sufficient for bearing purposes only) and other details that might not be as relevant to helping a human user see the relevant weather data for an area." | Fix now: "Alerts in view", the default scope (D-66) | |
| U1-16 | S4 | HUM LEAD: "on the tuimaps side this again suggests I should be able to turn on/off levels of detail in a view so my map isn't overcluttered with 'less relevant' information for the purpose I want to use the map for." | Fix now: go-tuiMaps detail levels in v0.2.0 (D-67, go-tuiMaps D-82) | |
