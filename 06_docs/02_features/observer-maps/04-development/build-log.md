---
title: "0.18.0 Observer maps — BUILD log"
date: 2026-09-25
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "LIVE — one entry per batch: what landed, the tests that hold it, the mutation verdicts, and what the gates found."
---

# BUILD log

## Batch 1 — the map window opens and draws in Update (2026-09-25)

**Tasks:** W1.1, W1.3, W1.7, W2.1, the first half of W1.2 and of W2.2, on go-tuiMaps
`v0.2.0-rc.8` (D-60). **Ruled during it:** D-60 (P1-a on the release candidate), D-61 (the map
window owns the keyboard).

**What landed.** `g` (`map.toggle`, in Help's NAVIGATE group) opens and closes the map window; esc
closes it. The composition root hands the window `Config.NewMap(size)` (`app/maps.go`), which the
window calls on the first `g` with the window's size — the library moves only a sized map. It names
no source: the embedded tiles draw the basemap and nothing is reached (FR-3.2). The map opens at
zoom 6 on the selection. It is drawn in `Update` into `mapPane` and `View` only prints (D-41);
`Work` runs as a command and its `mapWorkedMsg` is drawn by the `Update` it reaches. With no
selection the window says so and builds nothing (FR-1.3). Under `--ascii` it says the picture is
braille and names the remedy (FR-1.7, FR-1.8) until W1.4's description replaces that line.

**The closed-set guards found four real holes the moment the window joined the enum (W1.7):**
1. **A stale frame.** The window's memo key did not include the pane, so a frame drawn before the
   tiles landed was replayed after they landed — the failure D-45 exists to prevent. The pane now
   raises a generation on every draw and the key carries it.
2. The body ran into the border: the map is now drawn three cells inside each side.
3. The map was not redrawn on resize: `tea.WindowSizeMsg` redraws it (D-45's size row). The
   reachability guard used to assign the size fields directly; it now reaches 80×24 through the
   resize message, as a terminal does.
4. `mapWorkedMsg` had no route: it is an answer owed to Observer's window (`observerScoped`).

**Mutation verdicts** (each restored and compared):

| # | Mutation | Verdict |
|---|---|---|
| M1 | the memo key without the draw generation | caught — `TestTheMapIsDrawnInUpdate` |
| M2 | resize not redrawn | caught — reachability at 80×24 |
| M3 | a landing not drawn | caught — `TestTheMapIsDrawnInUpdate` |
| M4 | View draws the map | caught — `View called the library: [Render]` |
| M5 | a map built with no selection | caught (a crash in the no-selection test) |
| M6 | braille under `--ascii` | caught — `TestASCIIFramesCarryNothingButASCII` |
| M7 | the reply not routed to Observer | caught — `TestEveryWindowReplyIsRoutedBackToTheWindow` |
| M8 | not centred on the selection | caught — `TestTheMapIsCentredOnTheSelection` |
| M9 | no key bound | caught — `TestTheMapKeyOpensAndClosesTheWindow` |
| M10 | no inset | caught — the margin survey |
| M11 | the generation not raised | caught — `TestTheMapIsDrawnInUpdate` |

**What the gates found.** `TestDeclarationSetUnchanged` (a new top-level name in `app`, written
into its golden); `mutant-anchors` (mCA1's anchor, re-pointed at the case `mapWorkedMsg` joined).

**For UAT.** The library refuses to move a map that has no size yet (its P-56), so a host must
build the map at a size before it can place the view. Watchpost does; whether the library should
place a view given before the first size is a question for the paired release.

**Owed from these tasks:** W2.2's goroutine record and the `NextCall` tick; W2.6's join on close
(`closeMap` exists, the app does not call it yet).

## Batch 2 — the map window's keys, as D-61 ruled them (2026-09-25)

**Tasks:** W1.15 (pan and zoom), W1.16 (the keys evaluated and ruled: D-61), the rest of W1.2
(the map follows the selection).

**What landed.** While the map window is open it owns the keys it binds, from a keymap scope of its
own (`defaultMapKeyMap`, merged with the `[keys]` entries that name map actions): ←↑→↓ pan a
quarter of the view (`PanCells`), `+`/`=` and `-` zoom (`ZoomBy`), `[` and `]` step to the previous
and next location and the map and its title follow. Every other key still reaches the Observer, so
the window shadows only what it binds: today the radio's volume and the alert pager; `space`
(play/pause) is shadowed when the loop's play key lands with W8, as D-61 ruled.
Help lists the keys in a MAP group. The legend, playback and description-scroll keys join with their
tasks (W1.17, W8.9a, W1.6): a key bound to nothing yet would be a dead key.

**Found on the way.** `render.scrollBody` was a second copy of `scrollWindow` with the rail's
glyphs written in, so **every scrolling window drew ▲ █ │ ▼ under `--ascii`**. No window scrolled
at the size the ASCII survey draws until the MAP group made Help one. It now draws through
`scrollWindow`, and `TestAScrollingPanelIsASCIIUnderASCII` holds it directly. Help's
once-per-binding checks count per scope: the map's zoom keys render the same as the volume keys, by
ruling.

**Mutation verdicts** — all caught: keys not routed to the map (K1), east panning west (K2), zoom
inverted (K3), `]` not following (K4), a key press not drawn (K5), `[keys]` overrides ignored (K6),
no MAP group in Help (K7), the rail not ASCII (K8, by the render test alone), a pan of nothing (K9).

**Owed:** W1.2's scripted PTY journey on the real binary; W1.15's loading indicator and M2 over
every frame (with W4's bound).

## Batch 3 — the basemap (2026-09-25)

**Tasks:** W3.1, W3.2, W3.3, W3.4, W3.5, W3.6, W3.7, with W9.3 and W9.4 folded in (D-60).

**What landed.** `app/maps.go`'s `mapBuilder` builds the window's maps. Each names the closed
list's one basemap (`basemapSources`: OpenFreeMap's planet, FR-3.8), with the embedded tiles
always passed (FR-3.1), fetched by the library as `watchpost/<version>` and confined by it to that
host (HR-6, HR-10). Tiles are kept at `userCacheSubdir("map")` under a 256 MiB cap and a 7-day
maximum age, held by the library on disk (HR-8, FR-3.9); the stated total is that cap plus the
HTTP cache's (`httpx.DiskCacheBytes`, exported so the total reads it rather than a copy). The
memory tile cache is a 4 MiB shared set, kept across the session's maps (FR-3.6). **The zone
outlines are seeded by the first map the listener opens** and no longer at the station's start
(FR-3.2, D-25); the builder's seed is the only caller. The window's last line says what the
picture is while it is not whole: loading while work is pending, offline once a tile has failed
(until the picture is whole again), and coarser when the tiles cannot sharpen the view and
nothing is pending (FR-3.4, W1.15's indicator). Watchpost never prints the TileJSON's credit; the
library draws its own (FR-3.7).

**Found on the way.** The memo-completeness guard caught the status line's inputs missing from
the window's key. The first draft said "loading" forever on a view the embedded tiles cannot
sharpen: loading now means work is pending, and a third, stated case covers the rest. The
no-socket check of batch 1 forbade importing `net/http` at all, which is stricter than "no
socket" and blocked in-memory transports: it now forbids `net`, `net/http/httptest` and the
default client and transport.

**Process note.** The app tests of this batch were written before the builder but not run red
before it was written. The mutation run below stands in for that step: each behaviour was
removed and its test failed.

**Mutation verdicts** — 14 of 14 caught: another source (B1), no user-agent (B2), no disk cache
(B3), no maximum age (B4), the default memory cache (B5), zones seeded on every map (B6) and at
start (B7), a total missing the HTTP cache (B8), the cache outside the OS cache directory (B9),
offline never said (B10), loading never said (B11), coarse never said (B12), the memo missing the
offline note (B13) and the offline note never cleared (B14, which survived the first run and got
`TestTheOfflineNoteClearsWhenTheSourceReturns`).

**Owed:** W3.8's clear path (with W9.5's `Purge`), W3.9's zone politeness, FR-3.2's full
station-start instrument through a counting transport.

## Batch 4 — the bound (2026-09-25)

**Tasks:** W4.1, W4.2, W4.4 with W9.1 folded in (D-60: the library's `SetBound`, no host clamp).
W4.3's default-scale Setting lands with W1.11's rows.

**What landed.** `platform/geo/regions.go`: six regions, each boxed with its waters — the
contiguous US (the Gulf, the Great Lakes, both coasts), Alaska across the antimeridian to Attu,
Hawaii, Puerto Rico and the Virgin Islands, Guam and the Northern Marianas, American Samoa — and
`RegionOf`, which normalises any longitude. The map window holds the map inside the selected
place's region with the library's `SetBound`; its least zoom is the one at which the window's view
fits **inside** the region on both axes (a fit that merely *holds* the box would leave the view
wider than the region on one axis), set again on every resize and every change of place. A place
in no region is stated and nothing wider is drawn (FR-2.5).

**M2's instrument** (`TestNoFrameIsWiderThanTheRegion`): every frame the test draws — the first,
then twelve zooms out, forty presses each way, three resizes — for Oceanside, Adak (across the
antimeridian) and Hilo, each checked against its region.

**A coupling to state plainly.** The least zoom and M2's frame box are both computed in the
library's published scale (256-dot tiles, 2×4 dots a braille cell). If the library's scale moved,
both would move together. **For the paired release:** an accessor for a view's ground box (or a
bound mode that fits inside rather than holds) would let the host ask rather than recompute.

**Mutation verdicts** — 10 of 10 caught: the height fit ignored (R1) and the width fit (R2), the
antimeridian width (R3), resize keeping the old least zoom (R4), no bound (R5), an out-of-region
place drawn (R6), antimeridian containment (R7), longitudes not normalised (R8, which survived
until two wrapped-longitude cases were added), a territory missing (R9), and the memo missing the
out-of-region state (R10).

## Batch 5 — alert areas (2026-09-25)

**Tasks:** W5.1, W5.2, W5.4, W5.6, the station-locations half of W5.3; W5.5 found already held.
**And the HUM LEAD's rule of this date:** every batch brings the diagrams and artifacts with it.

**What landed.** `app/mapfeed.go`'s `mapFeed` (a `livePipelines` method, handed to the window as
`Config.MapFeed`) resolves the station's locations' alerts through `resolveAlertAreas` and the zone
store, and gives the window **one overlay per alert** (an alert on two places is drawn once), one
feature per area, CAP's severity mapped one to one onto the library's severity and role, the
alert's times carried and its id on every feature so `Report` joins back to it; the overlay is
kept until the alert expires. **A partial area is drawn as found**, its label says how much
("4 of 5 zones"), and a note under the map names the missing zones **in words**, taken from the
alert's area description, and says whether the selected place lies in them — decided by the
place's own zone and county codes, because a missing zone has no ground (FR-4.4, D-42). Where the
names cannot be matched to the zones, the note counts them rather than print a code. The window
asks for the feed off its goroutine on opening, on each snapshot and on each change of place; an
older answer is dropped; a gone alert is removed. The zone store fetches a held shape again once
it is seven days old (FR-4.6); offline, that zone is then missing and the alert a stated partial
area, rather than last week's line served silently. W5.5's cap was already built and held by
`TestAskingForMoreZonesThanExistIsBoundedAndSaysSo`.

**Diagrams, from this batch on, move with the code.** Redrawn: `approach-1-render-placement.md`'s
PLAN diagram, now AS BUILT; new `as-built-map.md` (the parts and who may name whom, and what the
window's body is); the system overview (`watchpost-cli/.../architecture.md`) gains the map
library, OpenFreeMap and the zone store. The atlas regenerated. `TestTheAsBuiltMapIsKeptInStep`
fails when the build log reaches a batch the as-built page does not draw.

**Process note.** The app tests were written first but run red after the code, as in batch 3; the
mutation run stands in. The window's feed tests ran red first.

**Mutation verdicts.** Caught: a severity mapped wrong (F2, after `TestEverySeverityHasItsRole`
was added — the recorded scenario had no Moderate alert), the partial label (F3), zones named by
code (F4), the place never in the missing zone (F5), a partial area withheld (F6), a stale feed
applied (F7), a gone alert kept (F8), new data not fed (F9), zones never ageing (F11), the overlay
kept past or short of its alert (F12, after the test learned to check it), duplicates (F1, after
`TestAnAlertOnTwoPlacesIsOneOverlay`). **Equivalent:** the notes' memo key (F10) — notes change
only through a feed, which always redraws and raises the generation the memo keys on; the key is a
second guard.

**What the gates found.** `vet-tags`: the test helper `scenario` clashed with a declaration in the
`watchpost_debug` build's tests, which only the tagged vet sees; renamed `m1Fixture`.

**Owed:** the national-severe half of W5.3 and its Setting (with W1.11).

## Batch 6 — the description (2026-09-25)

**Tasks:** W1.4 with W9.2 folded (D-60: on `Report`, the nearby distance set).

**What landed.** `modes/tty/map_describe.go`: at every draw the window asks the library's
`Report` for the selected place and keeps its answer; the description joins each alert's answer
to watchpost's own alert by id and says, **in M1's words**, whether it covers the place, stops
short of it or lies to one side, how far its nearest edge is and which way, how severe it is and
until when; an alert whose missing zone holds the place "covers it by a zone that could not be
drawn" (the feed now says which, `MapFeed.InMissing`). It opens with the place's conditions, and
says so when no alert is near. Units and directions are words, and follow the station's units —
the map is told them at every draw, and `f`/`c` with the map open redraws it (D-45's units row).
Under `--ascii` the description stands in place of the picture (FR-1.7). The builder sets the
nearby distance to 15 km, M1's own rule. `tty.Relation` is exported so the answer key can be
checked through the real parts.

**M1's answer key, through the real parts** (`TestTheDescriptionAnswersTheM1Key`): every recorded
scenario but the offline one — feed, the station's map, `Report`, the description's word — matches
the key. The key is computed from the recorded geometry; **the HUM LEAD's confirmation of it is
still owed**, and M1b is scored by the HUM LEAD on the description that ships.

**Found on the way.** The first render told the map nothing of the station's units, so a
Fahrenheit station's description said kilometres. `f` and `c` did not redraw an open map.

**Mutation verdicts** — 9 of 9 caught: "stops short" said as "lies to one side" (D1, by the
answer key), the missing zone ignored (D2), units never set (D3), `c` not redrawn (D4), the
default nearby (D5, which survived — no recorded edge lies between 10 and 15 km — until
`TestAnEdgeTwelveKilometresOffStopsShort`), the in-missing set not handed over (D6, by the answer
key), the empty map unstated (D7), no description under `--ascii` (D8), the join to watchpost's
alert broken (D9).

**Diagrams:** `as-built-map.md` redrawn (the description, the Report join, the nearby rule, the
`--ascii` path); atlas regenerated.

**Owed:** the description beside the picture, first in reading order when its mode is on, and
scrolling (W1.6, W1.10); the "nearby" Setting (W1.11).

## Batch 7 — the size floor, the description beside the picture, the map's Settings (2026-09-25)

**Tasks:** W1.5, W1.6, W1.8, W1.10.

**What landed.** Two rows in Settings' WATCHPOST UI group, Observer's only, written with the group
on close (`config` `maps`, `map_description`; empty reads as the default, an unrecognised word too):
**Maps** on or off (a toggle, like the tones), and **Map description** — with the picture (the
builder's default, for UAT to judge), instead of it, or off (a ←→ picker; one line each so the
group still fits unscrolled). With maps off, `g` says so in one line and no map is built (FR-1.6).
With the description on, it comes **first in reading order**, above the braille (D-55); the body
scrolls with PgUp and PgDn (D-61's keys, now bound) when it is longer than the window. **The size
floor has one owner**, `mapMinBody` (69 × 12): under it no map is rendered, and the window names
the size needed and the size present and carries the description (FR-1.4). Help's MAP group lists
its ten keys in four rows.

**Two layout decisions of mine, for UAT to judge.** (1) **The map window is 8 rows short of the
terminal** rather than the windows' usual 12, and (2) **it takes the terminal's width less its
frame** rather than the dashboard's content width, which reserves a rail the map window does not
have. Measured: at the documented 80 × 24 floor, the usual budget gave an 11-row map and the
dashboard's width a 65-column one — **the 69 × 12 floor (D-16) could never be met at 80 × 24**.
Now it is 70 × 12 there. The window's panel and the reachability guard both measure at that width.

**What the gates found.** The Settings goldens drifted, as they should: two new rows (updated,
reviewed, both colour and `--ascii`). A first draft with a label and three radio rows made the
group scroll, which the margin survey caught (a scroll rail trails at one cell). Help's MAP group
of ten rows pushed Help's row-mark legend out of its window, which `TestTheLegendBelongsToItsSurface`
caught. The memo-completeness guard caught both new settings missing from the window's key.

The lint gate (staticcheck QF1003) asked for the chip line's row test to be a switch.

**Process note.** RED was run after the code for most of this batch; the mutation run stands in.

**Mutation verdicts** — 15 of 15 caught: maps off ignored (S1), the file's word ignored (S2), the
settings not saved (S3), the description shown when off (S4), missing with the picture (S5), a map
rendered under the floor only to be thrown away (S6, survived until the floor test counted
`Render`), the floor not named (S7), the memo missing the settings (S8), not written to the file
(S9), the old height budget (S11, survived until `TestTheMapDrawsAtTheDocumentedFloor`), wider than
the terminal (S12), ← cycling forward (S13, survived until `TestTheDescriptionPickerGoesBothWays`),
the dashboard's width again (S14), the panel clamped to it (S15).

**Diagrams:** `as-built-map.md` (the body's new branches, the Settings, the scroll keys); atlas.

## Batch 8 — Settings in tabs (D-62), the legend, the disclosure, Clear map data (2026-09-25)

**Tasks:** W1.12, W1.17, W3.8 with W9.5 folded; and **D-62**, which the HUM LEAD ruled mid-batch
when the map's rows made Settings 35 lines against its 32-line budget at 133×44.

**Settings is one window in tabs** (`setup_tabs.go`): General (DATA, WATCHPOST UI, ALERTS -
EVENTS), Watchpost Radio (ALERTS - TONE, CORRESPONDENTS, RELAY REPLAY), Broadcaster (a new STATION
group: the transmitter and service radius, out of DATA), Maps (MAP). Observer shows General, Watchpost
Radio and Maps; the console General, Watchpost Radio and Broadcaster. **The tab is the focused row's**,
not state of its own, so every path that moves the focus - a save sent back to the location, a deep
link to the voices - lands on its tab unasked. ←→ switch tabs as in [w] unless the focused row is a
picker or toggle ("just works"); tab and shift+tab switch tabs from any row (the builder's addition,
recorded in D-62). The tab row takes the place of the blank line the first group opened with, so
tabs cost no height. The window is as wide as its widest tab on every tab. `stepGroup` is gone with
the "next question" it served.

**The map's Settings (Maps tab):** Maps on/off; what opening the map sends, directly under it
(FR-9.4), built from the closed list so it names exactly the hosts contacted; Map description;
**Clear map data** (space), which purges the window's live map through the library and asks the app
to empty the tile files through the library's `Purge` on a bare map (no source, no seed: clearing
never fetches), the zone store's shapes and the HTTP cache's zone entries (`httpx.ForgetPrefix` -
a prefix, not the plan's `PurgeHost`, so the station's forecasts stay cached); the retention and the
one stated total (FR-3.9). The first open of a session shows the disclosure above the map.

**The legend** (D-44, D-54): `L` lays a box over the map's top-right corner, keying only the
severities the feed drew, each with its digit; `[L] Legend` sits on the status line, naming the key
as bound; Help's MAP group gains its row.

**What the tests found, and what changed with them.** The single-page tests were written for the
page D-62 replaced; they were adapted, not deleted: `walkTo` reaches a row with ↓ where they pressed
tab; the scope helper reads **every** tab from the full tab list (not the tabs production shows, which
could not notice a tab wrongly hidden); the empty-heading check asks the surface, not the open tab;
the station's rows are now asserted on the Broadcaster tab (D-62 superseding "under DATA"); the fit
rule is per tab and per surface. New goldens pin the General and Maps tabs, which also restores the
PTY journey's "Theme -" to the corpus. **Settings' 80×24 reachability improved** (2 unreachable
lines to 1) and its baseline came down. **The setup hit budget was re-pinned** 2,630 → 2,817 (+7%):
the one-width rule lays each tab out per frame; the miss went down.

**Mutation verdicts** — caught: a page of every tab (T1), arrows never switching tabs (T2), every
tab on every surface (T3, then directly by `TestEachSurfaceHasItsTabs`), width following the open
tab (T4), the legend keying every severity (T5), the live map not purged (T7), every cached entry
forgotten (T8), zones not forgotten (T9), no chip (T11), `L` doing nothing (T12), files not reported
(T13), the clear path not wired (T14). **Equivalent, then removed:** a "disclosed" flag (T6) — the map
is built once a session, so the first build is the first open.

**Diagrams:** `as-built-map.md` (the legend, the clear path, the Maps tab, and a new diagram of
Settings' tabs); atlas regenerated.

**What the gates found.** `lint-authoring` (AP-DEAD-01): a `_ = loc` in a test; the variable is gone.
`mutant-anchors`: mAA4 (tab landing where the surface hides - re-pointed at the tab step's surface
check, `firstRowOfTab`) and mAI1 (the station's rows leaking - re-pointed at STATION); both applied
and caught again.

## Batch 9 — the Maps tab's scale, nearby and layers; the registry; the cost warning (2026-09-25)

**Tasks:** W1.11 (less alert scope, which is W5.3's and batch 10's), W4.3, W1.13, W1.14, and W9.2's
"nearby" Setting (folded by D-60).

**Test first.** `map_prefs_test.go`, `maplayers_test.go`, the config round trip and the app wiring test
were written before the code and failed to compile (RED); the code followed.

**The Maps tab gains three rows**, in focus order after Map description: **Default scale** (Region /
State / County; the file's `map_scale`, State by default), **Nearby** (5, 10, 15, 25 or 50 km;
`map_nearby_km`, 15 by default, named in the station's units - "9 miles (15 km)"), and **Layers** (a
box per layer the registry names; `map_layers`, keyed by layer, holding only the choices that differ
from the builders' defaults). All three save with the group on close, like the rest of the tab.

**Every open is at the chosen scale** (W4.3): State is zoom 6 (D-54's default), County 9, Region zoom 0,
which the region's bound raises to the least zoom that keeps the view inside the region. The bound is
not a Setting (FR-2.3). The open used to zoom only when the map was first built; it now zooms, and sets
the nearby distance, on every open, so a Setting changed in the meantime is what the next `g` shows.

**Nearby moved from the builder to the window.** `mapBuilder` no longer calls `SetNearby`; the window
sets it from its Setting on every open. The app's twelve-kilometre test
(`TestAnEdgeTwelveKilometresOffStopsShort`) moved with it: `TestTheNearbySettingReachesTheDescription`
puts an edge about 12 km off Oceanside, which stops short at the default 15 km (where the library's own
10 would say it lies to one side), and lies to one side at 5 km.

**The registry (W1.13, FR-9.3)** is `app/maplayers.go`. Layers and sources **register themselves from
their own files** in `init`: the alert areas from `mapfeed.go` (key `alert`, which `tty.AlertLayer`
owns), and OpenFreeMap from `maps.go` (`basemapSources` is gone; `mapSources` is the closed list as
registered). The window is handed the layers (`Config.MapLayers`) and the estimate (`Config.MapCost`),
and switches overlays by **the key before the slash in the overlay's id**, so a layer needs no
case anywhere else. **Options** are the Settings table's rows (D-62), which already take a new row
without editing the others; the registry does not duplicate them. R-9.2's layers are alert areas,
radar, fire and quakes - **the basemap is a source, not a layer**, so today the row has one box, and
with one box ←→ switch tabs (D-62: a row with nothing to walk to does not keep the arrows).

**The estimate and the warning (W1.14, FR-9.2, D-43).** Each layer carries its own cost; the estimate
sums the layers switched on. **What "a refresh" is, as built:** what one refresh of the map's data would
fetch if nothing were held - each layer's own requests. **The basemap is not counted**: it is fetched
once per view and kept 7 days, so it is not a refresh's cost. The alert areas cost one request per
distinct zone their alerts name (an alert with its own polygon costs nothing), at **10 KB a zone -
measured today, a mean of 9.6 KB over 159 cached zone responses** (wave 1 measured no mean). Over 2 MB
or 40 requests the station says, under the Layers row and under the map: "The map's layers would fetch
about X MB in N requests a refresh, more than the 2 MB or 40 requests this station warns at. Switching a
layer off costs less." At the station's own alerts this is rarely said; national scope (W5.3) and radar
(W8.15) are where it will be. **For UAT:** whether the reading of "a refresh" above is the one meant.

**With the alert areas off,** none is drawn, the partial-area notes go with them, and the description
says "Alert areas are switched off in Settings (s), on the Maps tab, so none is drawn or described." -
never that nothing is there.

**Deviations from the plan's signatures.** `refreshCost(snap, on)`, not `(layers, scope, step)`: scope
arrives with W5.3 and the step with radar (W8.15), each as a layer's own input. The warning is drawn by
`map_prefs.go`, not a `map_window.go` (the window is `map_pane.go`).

**Mutation verdicts** (targeted, 20 mutants, all caught): each threshold's comparison (three), a layer's
default, an unregistered key's default, the overlay filter, the notes filter, the estimate on a switch,
on new data and on opening Settings, the layers row keeping the arrows with one layer, the feed filter
in `applyMapFeed`, `SetNearby` and `Zoom` on open, the "off" description, County's zoom, the layers
cursor mark, the estimate skipping layers off, a polygon alert costing nothing, and a key registered
twice. Tests for ← on each picker, the notes going with the alert areas, and the estimate's triggers
were added for survivors found before the run.

**What the guards found.** `TestEverySurfaceSeamIsClassifiedForTheAir`: `Config.MapCost` classified
(arithmetic, no audio). The declaration set re-captured. `staticcheck` QF1011 in a new test.

**Diagrams:** `as-built-map.md` (the registry, the Maps tab's rows, the estimate); atlas regenerated.

## Batch 10 — the alert scope, with the region's national severe events; zone politeness (2026-09-25)

**Tasks:** W5.3's second half and its Setting (W1.11's last row, FR-4.3), W3.9 (NFR-4, D-46).

**Test first.** `mapnational_test.go`, `nws_area_test.go`, `TestRegionOfZone` and the scope row's tests
failed to compile before the code (RED). W3.9's bound was already in the store (`fetchAtOnce`, kept by
D-46), so its test passed on the code as it stood; a mutant stands in for its RED - and the first
mutant **survived**, because the test compared against `fetchAtOnce` itself and moved with it. It now
pins D-46's six as a literal, and both mutants (seven, one) are caught.

**The scope** is a picker on the Maps tab, **Alerts -**: "This station's places" (the default, and all
the map drew before) or "Plus national severe events" (`map_alert_scope`). The national events are
**the ticker's**: the national feed is already polled for the marquee and the severe window, and the
map reads the severe deck's copy of it (`severeDeck.nationalFeed`) - **no new request**. Each event is
turned into the alert the station's own reader would have made (`app/mapnational.go`), so it is
resolved, drawn, noted and described by exactly the station's path; it is kept to **the selected
place's region** (D-28) - by its point, or for a zone-only event by its first zone's code
(`geo.RegionOfZone`: a state's or a marine area's prefix; AMZ7xx lie off Puerto Rico). Superseded
events and other classes are left out; an alert the station holds too is drawn once. The station's
scope does not read the national feed at all.

**The national feed keeps each alert's polygon** (`SevereDetail.Area`), read by the same bounded
`geo.ReadGeometry` the station's alerts use; an unreadable shape is no shape and the alert stands. Its
point is still the first vertex of the geometry's **coordinates** (a bounding box that comes first is
not a vertex - a test pins it).

**The feed's and the estimate's inputs are one value** (`tty.MapAsk`: snapshot, place, scope; in the
app, `mapInputs` with the region's national alerts), so a later layer's input (fire, radar's step) has
room without another signature change. The estimate now counts the national zone-only events' zones in
the national scope; a polygon costs nothing. The warning's words end "Switching a layer off, or drawing
this station's alerts only, costs less."

**The description names a national alert in full.** The library's answer carries no end time, and the
description found an alert's name and end only among the selected place's own alerts; the feed now
hands the window the national alerts it drew (`MapFeed.National`), and the description falls back to
them.

**W3.9:** `TestZoneFetchesArePolite` - thirty zones, never more than six in flight and more than one,
each with the station's agent; the production store is built on the data client, whose agent names
watchpost.

**Mutation verdicts** (targeted, 19 plus 4 re-runs): caught - the region filter (by point, by zone), a
superseded event drawn, the national scope in the inputs, the snapshot copy, the ids made bare, the
national alerts handed to the window, the description's fallback, the notes filter carrying them, the
scope in the ask, the scope moving the estimate (after a test was added), the polygon kept, the zone
code's letter check (after a case was added), the bbox (after a case was added), the politeness bound
(after the literal). **Equivalent:** the production inputs skipping the national read in the station's
scope - the conversion is scope-gated too, so skipping it only saves a copy; kept, for the cost.

**What the gates found.** `dupes`: `severeDeck.nationalFeed` was `LaneRows` over another field - both
now read through one generic `lockedCopy`, collapsed rather than ratified. The declaration set
re-captured.

**Diagrams:** `as-built-map.md` (the scope, the national feed through the severe deck); atlas
regenerated.

## Batch 11 — the rest of W2: the clock, the join, the guards, the pins (2026-09-25)

**Tasks:** W2.2's second half, W2.3, W2.4, W2.5, W2.6, W2.7, W2.8, W2.9's gate half. With this batch
**P1-a's build is complete (W1–W5 with W9 folded, W2 whole), and UAT-1 opens.**

**Test first where there was something to build.** `map_work_test.go` (the goroutine record, the tick,
the join, the router's close) and `TestTheMapIsClosedWhenTheProgramEnds` failed to compile before the
code. The guards - W2.3's table, W2.4's property, W2.5's freshness, W2.7's pin, W2.8's goldens, W2.9's
count - hold behaviour already built, so each passed on arrival and **a mutant stands in for its RED**:
a handler that skips its draw, a draw that keeps a stale counter, a frame that keeps its old lines -
each caught.

**The clock (W2.2).** One tick is kept outstanding at the library's `NextCall` (a marker's phase, a
tile's retry, an overlay going stale), **armed after every `Update`** beside the dashboard's own tick,
so no path that draws has to remember it. A tick for a time no longer wanted is dropped; a closed
window keeps none; the delay has a 50 ms floor so a time still wanted after its draw cannot spin.
`mapTickMsg` is routed to Observer's map window (the routing guard named it the moment it existed).
**The goroutine record:** the call wrapper records each call's goroutine in tests; every call but
`Work` runs on the goroutine driving `Update` (`NextCall` included), and `Work` on another.

**The join (W2.6, FR-8.2).** The map's commands - `Work` and the feed - are admitted under a lock by
`mapWorkers`: a close cancels them, waits for them (bounded at 2 s, reached only by a call that ignores
its context), and a command that starts after the close touches nothing. The app closes the map with
the model the program ends with (`closeOnExit` in `runProgram`, through `Router.CloseMap`).

**Guard 2 and freshness (W2.3, W2.5):** one row per P1-a event - the feed landing, `Work` landing,
size, selection, pan, zoom, units, the library's tick, a map Setting on the next open - each draws, and
after each the stored `Changed`/`FrameTicks` are the library's. **Theme and colour depth are not rows
yet:** the map does not take the theme's palette until W7 (FR-7.1, FR-7.2), so a theme change has
nothing to redraw; the rows land with W7. Frame advance and playback land with W8.10.

**Guard 3 (W2.4):** random sequences of pan, zoom, resize, units, new data and ticks, each settled,
then the printed map compared with a second map built from the same inputs (bound, view, units,
nearby, overlays) rendered in the window's own order - render, work, render what the work did. **Its
first two versions were wrong, not the window:** the second map was rendered without the render that
asks for its tiles, then stopped at the tile the embedded set lacks, which the window does not. Every
`go test` runs 200 sequences (the race run and every mutant's included); **the 10,000 run under the
`property` tag in `make property`, a step of its own in `verify-gates`** (about two minutes), and
`vet-tags` vets the tagged file.

**The pin (W2.7, FR-8.3):** the map window at 133×44 over the benchmark fixture - **2,508 allocations a
memo hit and 2,919 a miss**, pinned at ×1.05; neither calls the library. The hit is above the severe
window's 1,740: the braille lines carry their colour spans and the compositor walks each. **Measured and
recorded, not optimised** (D-53).

**The goldens (W2.8, FR-8.5):** the window at 80×24, 120×40 and 133×44, each carrying the width
invariant: no line wider than the terminal, and the window's right border in one column from corner to
corner, so a map line a cell too wide fails. (Lines are not padded to the terminal's width - the frame's
convention, as every existing golden shows - so "exactly the width" was the wrong invariant, found on
the first run.) A self-test proves the invariant refuses a sheared border.

**M5's gate half (W2.9):** a cold open at 149×38 on Roswell, with its watch and warning, reaches its
first complete frame in **one TileJSON, two tiles and five zones** - bounded at one, six (the most tiles
a 278×116-dot view touches) and each zone once. "Never waits for radar" joins with radar (W8.12); the
p90 timing is the SHIP report's.

**What the 80-column golden shows, for UAT:** the library's footer runs the scale bar into the credit
("50 kmOpenFreeMap"), and the status line is cut mid-sentence ("…the map has no finer tiles for"). Both
are seeded in the UAT-1 findings log.

**Mutation verdicts** (targeted, 16): caught - the close check, the wait, the cancel, the close's join,
re-arming an outstanding tick, arming while closed, drawing a superseded tick, a tick that does not
draw, the record's goroutine, the router's close, the arming step in `Update`, the tick's dispatch, the
app's close, a skipped draw after `Work`, a stale counter, stale lines. **Not caught, recorded:** the
50 ms floor (a tick's delay is not observable without waiting it out).

**What the gates found.** `TestTheThreeGateListsAgree` and `TestEveryGateShapedTargetIsListedOrExempt`:
`make property` ran in `verify` and not in CI, and was on no list - "a gate that only runs on one
machine is a gate that passes on the other by not being asked". It is now in
`06_docs/required-gates.txt` and a CI step beside `alloc-budget`. The routing guard named `mapTickMsg`
(routed to Observer's map window); the declaration set gained `closeOnExit`.

**Diagrams:** `as-built-map.md` (the map's commands, clock and close); atlas regenerated.

## Batch 12 — UAT-1's first findings (2026-09-25)

**The HUM LEAD's first pass of UAT-1** (`07-readiness/uat-1-findings.md`, U1-6..U1-14): "This is pretty
impressive for 1st run … This is very good." Three rulings were asked one at a time and recorded: **D-63**
(the Area Alerts box opens by default), **D-64** (the title names the view by scale), **D-65** (the
Overlays menu; weather-first detail).

**Test first.** `map_uat1_test.go`, `TestARecentPlacesAlertsReachTheMap`, `mapnames_test.go` and
`states_test.go` failed before the code (RED, the recent-place test with the defect's own symptom).

**U1-14, a defect: a RECENT or SEARCHED place's alerts were never drawn.** The feed was asked with the
watchlist's snapshot alone; a place selected from RECENT keeps its alerts in the recent snapshot. The
ask now carries the selected place on a copy of the watchlist's snapshot (the shared one is never
written). The HUM LEAD saw it at New York, NY.

**U1-13, the window at about 80%.** The map window is 80% of the terminal each way, so the dashboard
shows round it, and never less than a 69×12 map needs (FR-1.4): at 80×24 it is as it was.

**U1-7 and D-63, the Area Alerts box.** The description is a box over the map's upper left - the legend
has the upper right - open on every open when the description's mode is "with the picture", `A`
closing and reopening it; with the mode off it waits for `A`. The session's first open says what the
map sends inside it; with the box closed, above the map, as before (FR-9.4 said either way). Longer
than two thirds of the map, it ends "… the rest: see Settings". "Instead of the picture" and `--ascii`
are the full text, as before. The map is no longer shortened for the description: it is the box's
ground. `TestTheDescriptionComesFirstWithThePicture` now holds D-55 as D-63 keeps it - the words on the
window's first rows.

**U1-11, the controls.** A box over the map's lower right shows the map's keys as chips; the one pressed
is drawn inverted (the Settings pickers' acknowledgement) and its blink ends on the dashboard's tick
after it expires. A map under 14 rows or 60 columns draws none (the keys are on the chip line and in
Help).

**U1-12 and D-64, the title.** The window names what is in view from the view's centre and width: under
80 km the nearest town of 5,000 or more (else the nearest place); under 700 km the part of the state -
where the centre sits in the state's extent, on the axis it is further out on, both only when far out
on both ("Northwestern"), "Central" near the middle; under 1,500 km the state; wider, the region. The
selected place is named with it while it is in view. The extents are each state's own cities', in one
pass (`geodata.StateExtents`); the names are `geodata.StateName`. A point in open water names the
region.

**U1-9, U1-10 and D-65, the Overlays menu and the map's detail.** `O` opens a menu over the upper left -
the registry's weather layers, then the map's detail (borders, water, rivers, place names, roads, rail,
parks and reserves) - which owns ↑↓, space and esc while it is open (D-61, modal control priority). A
switch reaches the library at once (`Map.Layers`, already in go-tuiMaps - **no library change was
needed**) and is written at once, as the Settings window writes its own (`map_detail`). The detail
opens weather-first: roads, rail and parks off. Settings' Maps tab has a **Map detail** row - one layer
between the arrows and how many are on (seven boxes on one row made Settings wider on every tab).

**U1-2, the status cut short, fixed with them.** Three chips beside the status cut it even at 133
columns, so the status and the chips are now two lines, both reserved whatever the status says (the
map's size must not depend on the last frame's status). At 80×24 the status now reads whole.

**The pin, re-pinned:** the map window's frame is 3,541 allocations a memo hit and 4,011 a miss (2,508 /
2,919 before): the boxes are more spans for the compositor. Recorded, not optimised (D-53).

**What the tests found while building.** The memo guard named `mapPane.title` (a frame that changed and a
key that did not); the first Map detail row widened Settings on every tab; with the description off the
box still showed the words, and then the first open's disclosure went unsaid; the boxes were placed
three cells in where the lines start at one; the controls' blink cannot be seen with colour off, so its
test draws in colour. `staticcheck` QF1001 in a condition.

**Mutation verdicts** (targeted, 26 and 3 re-runs), all caught: the recent place in the ask, the 80%
width and height, the width's floor, the box open by mode, `A`, the disclosure's place, the blink's
mark, its drawing, its end and its tick, the title's view test, its fallback, its drawing and its memo
key, the detail's library call (after a picture test), the weather-first defaults, the menu owning its
keys, the menu's save (after a save test), the status rows, the region name, the town preference (after
a unit test), "Central", the state name, and the app's namer.

**Diagrams:** `as-built-map.md` (what sits over the map, the title's namer, the detail); atlas
regenerated.

## Batch 13 — UAT-1's second pass; the detail level on go-tuiMaps rc.9 (2026-09-25)

**The HUM LEAD's second pass** (U1-17..U1-25), and U1-26, found in U1-19's screenshot. Rulings: **D-69**
(the map says nothing of what it sends; FR-9.4 amended to Settings alone), **D-70** (Settings shows every
tab and row on every surface, superseding D-92's visibility), **D-71** (General split into Data and
Watchpost UI), and go-tuiMaps **D-84** (ten reference frames rewritten on the footer row). **U1-21 is
not a defect: radar is W8, P1-b, reviewed at UAT-2.**

**Built on go-tuiMaps `v0.2.0-rc.9`** (WP-L11: `SetDetail`, major and minor roads apart, the footer's
scale mark and credit never touching - U1-1 fixed in the library).

**U1-19, a defect: the colour lost beside the Area Alerts box.** `render.SpliceCells` wraps a patch in
resets - and a row that set its colour once, before the span, drew every cell after the patch in the
terminal's default. The splice now remembers the row's tone (every escape since its last reset) and puts
it back after the patch; a tone the row ended stays ended. RED first: `TestSpliceCellsRestoresTheToneAfterThePatch`.
The legend now splices too (it truncated and appended, the same loss), one row down: the library writes
the stale word and the frame time along the top row.

**U1-17 and D-69: no disclosure on the map**, in any mode; Settings says it beside the maps row.
`TestTheFirstOpenSaysWhatIsSent` is now `TestSettingsSaysWhatIsSent`.

**U1-18 and U1-20: the picture's window never scrolls.** The disclosure above the map was what pushed
the status and the chips out of the window when the box was closed; with it gone, the body is the map,
its notes and the two status rows, exactly the window - `TestThePicturesWindowNeverScrolls` at four
sizes, box open and closed.

**U1-26, a defect: the controls covered the credit** (the attribution, FR-14). They stand above the
library's last row now.

**U1-22 and D-71:** Data (DATA, ALERTS - EVENTS - a filter on what arrives, the agent's placement) and
Watchpost UI (WATCHPOST UI) replace General. **U1-23 and D-70:** every surface shows every tab; each row
still writes what it wrote. The tests that held D-92's visibility were rewritten to D-70
(`TestEverySurfaceOffersEverySetting`, `TestTheStationsSettingsAreOnEverySurface`,
`TestEachSurfaceHasItsTabs`); the reachability baseline for Settings fell to 0 (WATCHPOST UI's header is
now its tab's first line).

**U1-24 and U1-25: the Maps tab in two aligned columns.** MAP (maps on or off with what it sends, the
description, the scale, nearby, the alerts) and MAP - LAYERS AND DETAIL (the layers and the cost warning,
the detail level, the whole detail list, clear map data) - two groups, which the window's existing
column plan lays side by side. Within a column the labels and the pickers' values are padded to one
width. The detail level's picker is as wide as its own longest value, not the first column's: at the
first column's width the two columns stopped fitting and the tab stacked and scrolled - seen in the
golden, fixed before commit.

**D-67 on rc.9: the detail level.** Weather by default (`map_detail_level`); a Settings row and a row
of the Overlays menu (space steps it); the map is told on every open and every change. The detail list is
borders, water, rivers, place names, major roads, minor roads, rail, parks and reserves - every switch on
by default, the level doing the thinning - and a layer the level does not draw says the level that would
("Minor roads (at Full)"). The Overlays menu is 40 cells wide for those words.

**Pins re-pinned, and why:** Settings' memo miss at 133×44 3,808 → 4,127 and at 80×24 2,800 → 3,127
(five tabs measured where three were, the two-column Maps tab, the detail list).

**Mutation verdicts** (targeted, 14 and 2 re-runs), all caught: the tone put back, a reset ending it
(after a test was added), the controls' row, the legend's row, the Watchpost UI tab, every row on every
surface (twice), the level reaching the library (after a picture test was added), Weather the default,
the "(at …)" hint, the menu's step, the Settings row writing, the save carrying the level, the level's row.

**Diagrams:** `as-built-map.md` (the tabs on every surface, the Maps tab's columns, the level); atlas
regenerated.

**What the gates found (batch 13).** `TestTheRosterCitesTestsThatExist`: 0.16.0's gates.md still named
the test D-70 replaced; mAA1's row is retired (**D-72, the HUM LEAD's**: its guarded behaviour is now
intended) and mAA2's names its catchers today. `TestThePublishedTreeNamesNoPersonOrMachine`: the
findings log quoted the screenshots' desktop paths; they read as placeholders now, the meaning kept.
`mutant-anchors` then found mAA5 drifted: it mutated the very line D-70 removed, to the behaviour D-70
requires. **Retired by the HUM LEAD (D-73)**; 0.16.0's M4 is owed a new definition and instrument.

## Batch 14 — the map flush to its borders; the flicker (2026-09-25)

**The HUM LEAD's third pass, first two findings** (U1-27, U1-28); "Alerts in view" (D-66) follows as its
own batch while the next pass runs.

**U1-28, a defect: an alert's area "periodically flickers".** Reproduced before any fix
(`TestTheSameAlertAgainNeverDropsFromTheFrame`, RED on the first refresh): every new snapshot asks the
feed again, the same alert comes back, and `applyMapFeed` handed it to the library again. **A `Set`
replaces what the library prepared for that overlay, and until a `Work` prepares it again the area is not
drawn** - the frame drawn as the answer landed had no area, and the next `Work` brought it back. The
window now keeps what it last handed in, by id, and does not hand in an unchanged overlay; a changed one
is still set (`TestAChangedAlertIsHandedInAgain`). **The library half is owed to go-tuiMaps v0.2.0
(D-68):** a replaced overlay should be drawn as it was until its replacement is prepared, so an alert
that does change never blinks either - WP-L11's L11.5, with the next batch.

**U1-27: the map window runs border to border.** The panel gains a flush form (`render.Opts.Flush`),
which the map window alone uses; every other window keeps its inset (`TestAFlushPanelRunsItsRowsBorderToBorder`,
`TestTheMapRunsBorderToBorder`). The map is the window's width less its two borders; the words (the
description instead of the picture, the notes, the status, the chips) keep one space and wrap one cell
inside it, so none is cut at the border. The modal's wrap learned the flush width too - at the inset
width every map row split in two and the window scrolled. **`TestEveryWindowClearsItsMargins` exempts the
map window by ruling and holds it flush** (an inset creeping back fails).

**Mutation verdicts** (targeted, 7), all caught: the unchanged-overlay skip both ways, the flush panel,
the map window's flush option, the flush wrap, the words' one-cell margin, the 80% width's floor.

**What the gates found.** `lint-authoring` (AP-DEAD-01): a `_ = m` in a new test; the dead value is gone.

**Diagrams:** `as-built-map.md` (the flush window, the feed's hand-in); atlas regenerated.

## Batch 15 — UAT-1's fourth pass; Alerts in view (2026-09-25)

**The HUM LEAD's fourth pass** (U1-29 to U1-36, D-74, D-75) **and "Alerts in view"** (U1-15, D-66), for
UAT together. watchpost moves to go-tuiMaps `v0.2.0-rc.10`, which brings L11.5: a replaced overlay is drawn
as it was until its replacement is prepared, the library half of U1-28.

**Alerts in view (D-66), the default scope.** The window's ask carries the view (`MapAsk.View`, from the
map's centre and zoom). The app reads the states and marine areas the view touches (a 5×5 sample: the
nearest town's state within forty miles, else the coarse NWS marine area), asks the Weather Service once
for all of them (`Provider.AlertsInAreas`, `/alerts/active?area=`), remembers the answer for two minutes,
and keeps to the view the polygons whose box meets it and every zone-only alert. The estimate, which runs
on the UI goroutine, reads the memory and never fetches. The window asks the feed again 600 ms after the
view stops moving, by generation, so a pan asks once, and only with the in-view scope. The scope picker
runs In view → the station's places → plus regional severe.

**U1-29 (D-74): the description's words.** "Oceanside, CA - Currently: 72°F, cloudy." then, per alert,
"‹event› in effect for this area / for nearby ‹areas› / for ‹areas› until ‹time on day›." The areas are the
alert's own area description, its first two places, generic words lowered. M1's relation is unchanged
underneath, so its answer key stands.

**Settings (U1-30 to U1-36).**
- A blank row under the tabs, where the height allows (U1-30).
- The transmitter's support line is the HUM LEAD's sentence on one line, so no wrap splits its tint (U1-31, U1-32).
- The borrowed-location note moves to a line of its own; a narrower window had wrapped it to the margin.
- The helper text under Maps is gone. The Status window gains a MAP block: each source the map contacts,
  its host and what it is sent (U1-33, U1-34, D-75).
- The map's detail is a row per layer in the "← Enabled →" pattern, toggled by ←, → or space (U1-35).
- The window is no wider than 80% of the terminal, or the one-column floor on a smaller one, and the
  column plan fits inside that (U1-36).
- To keep the Maps tab's two columns side by side at 133, its words are cut:
  - "Description" and "Opens at";
  - "Detail", as the Overlays menu says it;
  - "Parks";
  - "Add regional severe" and "Station's places";
  - "With the map" and "Instead of the map".
- The FIRMS address moves to its own line so the Data tab fits too.

**Pins.** `TestSetupAllocBudget` re-pinned, the measurement recorded beside it: the 133×44 hit is
2,817 → 3,046 and the miss 4,127 → 4,699, because the narrower window shows more of the dashboard. Forced back
to 126 cells, the same frame measures 2,605 / 4,258. The 80×24 miss is 3,127 → 3,319, from eight picker
rows where one list was.

**Mutation verdicts** (targeted, 25 over two rounds), all caught in the end. Seven first-round survivors
were answered in two ways:
- **Stronger tests** for:
  - the estimate fetching with nothing remembered;
  - the latitude half of the view's box;
  - the forty-mile reach offshore;
  - a detail row that could only switch off;
  - the first-two-places limit and the lowered words (`TestTheAreasAreTheServicesOwnFirstTwo`).
- **A correct filter**: the settle tick's generation and scope guards survived only because the round's
  `-run` filter skipped `TestTheFeedIsAskedOnceThePanningStops`; the whole package kills both.

One survivor was dead code: the detail rows' `mapPickerRow` entry. The rows are toggles, which that
check never reads, so it is removed.

**What the gates found.** `lint` (unused): `distanceWords`, which D-74 left behind when the words dropped
distance and bearing; removed.
`dupes`: `tty.MapView.Contains` repeated `geodata.Extent.Contains`. They collapse into one `platform/geo.Box`,
of which both are aliases.
`mutant-anchors`: mAI6 ("a borrowed epicentre looks chosen") drifted when the borrowed note moved to its own
line. The rule is unchanged, so it is re-pointed at the new line (not retired), and the tty tests still kill it.

**Diagrams:** `as-built-map.md` (Alerts in view and its settle tick, D-74's words, the Status window's
MAP block, the Maps tab's rows, the 80% rule); atlas regenerated.

## Batch 16 — UAT-1's fifth pass: what is real in view; every region (2026-09-25)

**The HUM LEAD's fifth pass** (U1-37 to U1-41) and three rulings (D-76, D-77, D-78). watchpost moves to
go-tuiMaps `v0.2.0-rc.11`, which brings L11.6.

**U1-37 and U1-38: only the watched places' alerts drew.** This was diagnosed before any change. The
listener's file held `map_alert_scope = 'station'`: every Settings save writes every row, so an earlier
default was pinned. Run live against the same view, the app's in-view feed returned 38 alerts and 36
overlays. **D-76: the map shows what is real in view, switched at the map ([O] Overlays), and the scope
setting is removed.**
- The ask carries no scope, and the app always reads the view's areas.
- The national scope is removed: `app/mapnational.go`, `severeDeck.nationalFeed` and
  `geo.RegionOfZone`, whose only reader it was.
- The window's and the app's "national" alerts are renamed "in view".
- `map_alert_scope` is read and ignored, so a file that has it reports no unknown key, and it is written
  empty, so the next save drops it (`TestTheRetiredAlertScopeLeavesTheFile`).
- The "alert areas are off" words now point at the Overlays menu.

**U1-39: water off took the sea** (go-tuiMaps D-85, L11.6). The library's `WaterLayer` is now lakes and
inland water; the sea and its coast are never switched. The Maps tab row is "Lakes".

**U1-40 (D-77): every region, two ways, on the HUM LEAD's arrangement.**
- `platform/geo/arrangement.go` holds the layout: `Neighbour` and `RegionNumbered`.
- `1` to `6` snap the map to a region, shown whole.
- A pan the region's edge holds still crosses to the neighbour.
- The Controls box shows "1-6 region", and Help names the six in two rows; six rows pushed Help past its
  window at 133×44.
- The region-bound test (M2) now holds each frame to the region it was bound to, not only the place's.

**U1-41 (D-78): the box follows the view.**
- While the selected place is in view: its conditions, then its own alerts in M1's words.
- Away from it: the view's name, from the namer or the region, and nothing of the place.
- Either way, every other alert whose outline meets the view, most severe first. The box ends "And N more
  in view." when full; the text-only description lists them all.
- The title no longer names the place when the view has left it, namer or not.

**Mutation verdicts** (targeted, 19), all caught in the end. Three first-round survivors were answered
with tests:
- the view box's latitude half: a flood warning due north, at the view's longitudes;
- the feed's in-view merge and its held/not-held split: `TestTheFeedDrawsTheViewsAlertsAndNamesThemOnce`,
  covering what the deleted national-scope tests used to.

**What the gates found.** `tidy`: go.sum still listed rc.10 beside rc.11; tidied. The licence list was
regenerated for rc.11 (`TestEveryRequiredModuleHasItsLicenceListed`). Both steps belong to every go-tuiMaps
bump.

**Diagrams:** `as-built-map.md` (the national scope gone; the view's alerts unconditional; the region keys,
the arrangement and the description following the view); atlas regenerated. The UAT guide's keys and S4,
S8, S10 to S12 now describe the region keys and alerts in view.

## Batch 17 — UAT-1's sixth pass: switches that switch; alert categories and earthquakes (2026-09-26)

**The HUM LEAD's sixth pass** (U1-42, U1-43) and two rulings (D-79, D-80).

**U1-42: Rail, Parks and Minor roads did nothing.** Diagnosed before any change:
- The listener's Detail level was Weather, whose library level draws neither rail nor parks (Standard) nor
  minor roads (Full), whatever their switches said.
- The style draws parks only from zoom 6, rail from 8 and minor roads from 10, which is city scale.

**D-79: the level is a preset; the switches are the truth.**
- The library is set to Full, and the switches alone thin it.
- Choosing a level sets every switch to its preset. A switch changed afterwards makes the level read
  "Custom".
- Minor roads are no longer offered and the library is told to draw none.
- Rail and Parks say when they first show: "county zoom" and "state zoom".

**U1-43 (D-80): alert categories and earthquakes.**
- Each alert overlay carries its category in its ID (`alert/<category>/<id>`), from the [w] window's
  own `severe.Classify`.
- The Overlays menu lists [w]'s categories under the alert areas: Emergency, Warnings, Watches,
  Advisories, Spec. Statements, Marine. Each is switched and saved with the layers (`alert-<category>`),
  and the window takes a switched-off category's overlays out.
- Forecasts, and products [w] does not show, are not drawn, and their zones are not fetched or costed.
- A new `quake` layer draws the significant quakes the ticker already holds, when they are in view. Each
  is a circle, 10 km at magnitude 4, doubling with each magnitude up to 400 km, labelled "M 6.2". It is
  drawn in the track's colour with no severity, so the library never reports it as an alert over a place.
- The forecast overlays the HUM LEAD described are filed as F-182.

**The scratch test, codified.** `TestNoScratchTestIsInTheTree` (cmd/watchpost) walks the files on disk, not
the index, and fails on a `zz_` test file or a `TestZZ` function. It is proven against a probe file.

**Mutation verdicts** (targeted, 15), all caught in the end. Two first-round survivors were answered:
- **The library given the listener's level instead of Full.** The tests read only the call's label, so the
  label is now built from the value passed; the mutant changes it, and the tests catch that.
- **The quakes not read from the deck.** `TestTheQuakesComeFromTheTickersFeed` builds the inputs from a
  severe deck holding one quake in view and one out.

**Diagrams:** `as-built-map.md` (the earthquakes layer, the categories in the alert overlays' ids, the level
as a preset); atlas regenerated. The UAT guide gains S16 (categories and earthquakes) and S17 (detail as a
preset).

## Batch 18 — UAT-1's seventh pass: the edge's chip; the cost warning's words (2026-09-26)

**U1-44 (D-81, amending D-77's edge).** A press the region's edge holds still no longer crosses. It shows a
chip at that edge naming the region beyond ("US CARIBBEAN →", "← HAWAII", "↑ ALASKA", "↓ CONTINENTAL
US"), and the next press the same way crosses. Any other key takes the chip away, and a different
direction pans as normal. The chip sits mid-side for east and west, top-middle for north, and above the
credit row for south, which is never covered (FR-14). The frame memo keys on the chip; the memo guard
caught that it did not.

**U1-45 (D-82).** The cost warning reads the HUM LEAD's words, the first sentence in bold: "Map may
experience performance issues at this zoom level." then "Est. 2.2MB / 211 Requests | Switch off layers or
zoom in for a better experience." It is the same under the map and beside the Layers row. The old words'
"drawing this station's alerts only" went with D-76.

**Diagrams:** `as-built-map.md` (the chip, the warning's words); atlas regenerated; the UAT guide's keys row.

**Mutation verdicts** (targeted, 7), all caught in the end. Two first-round survivors were answered:
- **A chip that crossed whichever way was pressed next.** `TestAChipCrossesOnlyTheWayItPoints` covers it:
  Samoa's east chip, then its west edge shows Guam's chip rather than crossing.
- **The warning's sentence not bold.** The thresholds test now turns styling on, so bold is visible to it.
