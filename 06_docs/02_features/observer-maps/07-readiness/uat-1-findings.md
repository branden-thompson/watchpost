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
| U1-1 | S5 | SEEDED (80×24 golden, batch 11): the library's footer runs the scale bar into the credit - "├──────────┤ 50 kmOpenFreeMap (c) OpenMapTiles…" - no space between them. A go-tuiMaps footer question. | | |
| U1-2 | S5 | SEEDED (80×24 golden): the status line is cut mid-sentence - "This is a coarser picture: the map has no finer tiles for" - at 80 columns. | | |
| U1-3 | S10 | SEEDED: the map is drawn in the library's own colours, not the theme's; the theme's palette is W7's. | | |
| U1-4 | S12 | SEEDED (batch 9): "a refresh" was read as each layer's own fetches with nothing held, the basemap not counted; zones at a measured 10 KB. To be confirmed. | | |
| U1-5 | S1 | SEEDED (batch 11): the map window's frame costs 2,508 allocations a memo hit, against the severe window's 1,740 - pinned, not optimised (D-53). Only a finding if the window feels slow. | | |
