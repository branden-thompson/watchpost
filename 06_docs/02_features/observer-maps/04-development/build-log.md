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

## Batch 19 — W8 opens: radar from IEM and MRMS (2026-09-26)

**Rulings before code:** D-83 (MRMS by default, IEM for the lower 48 as an option, the source's chip)
and D-84 (an empty frame at an advertised time is a clear sky, not a missing one; the traps are shut at
the request; an all-empty loop is checked against the other source).

**W8.2, the fixtures, recorded before anything used them** (`domains/radar/testdata/`, manifest). IEM's time
list and a frame; an off-grid time (empty, 0 painted pixels); no time sent (2011's radar, 13,884 painted
pixels); MRMS's capabilities and a frame; a time past retention (empty). Every wave-1 trap reproduced.

**W8.3 and W8.3a, the sources and the boxes** (`domains/radar`).
- IEM covers the lower 48 alone (its AK/HI/PR/GU composites list no scans).
- MRMS covers the lower 48, Alaska, Hawaii, the Caribbean and Guam; neither source covers American Samoa.
- A source is asked for fixed boxes, never the view (D-47): each region's one box when the view is wide,
  and the lower 48's 4×2 grid boxes a closer view meets.
- **Found live, not in any document:** the library refuses an image over 250,000 pixels (its image cap,
  go-tuiMaps D-85/D-36; a host may lower it, never raise it). The boxes were first sized to the
  1,048,576-pixel header limit and every frame was refused. The boxes are resized and the test holds the
  true cap.

**W8.4 (FR-5.3 as D-84 amends it).** `Frame` refuses a time the source did not advertise, before anything
is sent; a time is always sent. `Check` reads the header first, refusing a picture past the cap undecoded,
and reports whether a frame paints.

**W8.5, W8.14, W8.14a, the radar client.** Its own `httpx` client: no cache directory, a 1 MiB body cap
enforced as it reads and never cached, a dial check that refuses loopback, private, link-local and
unspecified addresses after resolution, and https only. `bbox` is redacted from every error.
**Deviation, recorded:** the plan named per-request options (`BodyCap`, `MemoryOnly`). A dedicated client
is stronger, since no radar request can reach the disk tier at all, and simpler.

**W8.6, W8.7, W8.15, W8.15a, the loop** (`app/mapradar.go`).
- 24 five-minute slots over two hours: IEM's grid fills each exactly; MRMS's two-minute cadence fills each
  with its newest time within the step. A slot with none is a stated gap.
- Frames are fetched newest first; a failed or refused frame is a gap.
- A refresh fetches only new frames: held ones are answered from the client's memory.
- The estimate counts the boxes' frames; the Status window names both hosts.
- MRMS's approximate-colours note shows under the map.

**W8.8, W8.9a, W8.10, W8.12, the window** (`modes/tty/map_radar.go`).
- Radar is its own command, asked newest-only then whole, so the alerts never wait for it.
- The source's chip is on the second row's right. The library's time owns the first row (U1-26), so the
  legend moves down a row while the chip shows.
- The status line leads with the loop: the moment shown, the frame, stopped or playing, and the newest
  frame's age, stale past ten minutes.
- The playback keys are D-61's: space, `,` `.` and `n`, listed in Help. The map opens stopped on the
  newest frame at the slow rate.
- Switching radar off takes it away. The Maps tab has "Radar - MRMS / IEM (lower 48)".

**Live, end to end:** MRMS over the lower 48, IEM over southern California and MRMS over Hawaii, each a
24-frame loop, newest one to two minutes old, the whole loop in 6 to 8 seconds.

**Not in this batch:** the motion Setting's off and normal (W8.9, slow is built); W8.11's goldens over
specimen 29; M5's and M6's timings (W8.12, W8.13), measured at SHIP; the legend's radar classes (W9.10).

**Mutation verdicts** (targeted, 27). 26 were caught in the end: three first-round survivors were answered
with stronger tests (a close view in Alaska or Hawaii takes that region's box and never the grid; a
southern view takes southern boxes alone; the source toggles back as well as forth). One is **equivalent**
and says so here: dropping the frame fetch's error check in `radarLoop` changes nothing, because
`radar.Check` refuses a missing picture on the next line.

## Batch 20 — UAT-2's first pass: the loop preloaded; its timeline; Area Alerts closed on open (2026-09-26)

**U2-1, U2-2 (D-85).** Diagnosed live before any change: the library held the 24-frame loop, and the window
was at fault. Every later ask (each refresh, each settle) superseded the whole-loop ask before its 6-8 s
answer landed, so only the newest frame was drawn, each ask replacing it: the blink.
- One radar request at a time. A later want is kept and asked when the answer lands, and every answer is
  applied.
- The newest-only phase is gone. The loop is shown once it is in, with "Radar loading…" until then.
- New data asks again only after two minutes.
- A loop handed in is drawn once the library has prepared it, or at once when nothing is left to prepare.
  The map is not redrawn in between.

**U2-3 (D-86).** A timeline under the map while radar is on, its three rows held so the map's size never
waits on the loop:
- the frame's time above its mark;
- `[shift+←] ├──…█…┤ [shift+→]`;
- beneath, the oldest time, OBSERVED, and NOW with the newest time. With forecast frames, NOW is a tick
  and FORECAST follows it.

shift+← and shift+→ now step (D-61's `,` and `.` gave way to them). Help writes them ⇧← ⇧→, or S-left
and S-right under `--ascii`.

**U2-4 (D-87).** The Area Alerts box is closed when the map opens; A opens it.

**What the tests found.** A test helper had run the Work command and thrown its answer away, so the library
ran ahead of the drawn frame. The helper now sends every answer through `Update`. The window was right.

**Mutation verdicts** (targeted, 12), all caught in the end. One first-round survivor, the timeline's rows
not held in the map's height, is answered by `TestTheTimelineNeverMakesTheWindowScroll`: with radar on, the
window fits whole at 133×44, 100×30 and 80×24.

**Diagrams:** `as-built-map.md` (the one whole-loop ask, the keys and the timeline, the box closed on open);
atlas regenerated; the UAT guide's S18.

## Batch 21 — UAT-2's second pass: radar that loads (2026-09-26)

**U2-5, diagnosed live before any change.** Over Chicago and New York the radar stood at "loading" for good.
The library refused each loop: it charges a frame its PNG plus a byte a pixel, and a close box's two-hour
loop came to 6.8 and 7.1 MB against the 6 MiB budget. The whole-lower-48 test of batch 19 happened to fit.
The window ignored `Set`'s error, so a refusal read as loading.

**D-88.**
- The image budget is 24 MiB, the worst case of four boxes.
- A refused loop is said ("Radar could not be drawn: …"), with no chip.
- A loop as fetched is trimmed of its oldest frames, the same number from each box, until it fits four
  fifths of the budget.
- The boxes are smaller (the grid 400×373, the whole regions about 600 wide): each box's loop is lighter,
  and a county view still has twice a terminal's dots.
- Live: Chicago, New York and the whole lower 48 each load a 24-frame loop in about 6 s; a view across
  four boxes takes 18 s, the frames fetched one at a time. Speed and memory are F-183.

**Mutation verdicts** (targeted, 8). All caught but one, removed: an estimate that fetched only the frames
the budget would hold could never bind at 24 MiB, so it was dead code. The trim as fetched is the
safeguard, and its removal is caught.

**Diagrams:** `as-built-map.md`; atlas regenerated.

## Batch 22 — UAT-2's third pass: the rows under the map, as the HUM LEAD drew them (2026-09-26)

**U2-6 (D-89).** While radar is on, under the picture, in order:
- the radar's colour row, "RADAR LEGEND │ LIGHTER … HEAVIER", a swatch a class in the class's own colour
  from the library's legend (`render.Swatch`: the colour is the datum, not a theme's), its words where
  colour is off;
- the warning on one line, "Map may experience performance issues at this zoom level." in the list
  pointer's bold yellow (already AA-checked on the window's ground in every theme), then the advice;
- a blank, the timeline, and the status line: "Radar", the source's chip in the badge's colours, the loop,
  and "/ Est. MB / Requests" before anything the picture's status says;
- a blank, then the chips.

The legend box no longer lists the radar's values. Settings' Layers row keeps the full warning, estimate
included, since it has no status line. The rows are held while radar is on, so the window still never
scrolls (`TestTheRowsUnderTheMapAreTheMocks` checks the order, the blanks, the colours and the fit).

**Mutation verdicts** (targeted, 8), all caught in the end; one first-round survivor, the "loading" line's
chip, is answered by `TestALoopStillPreparingSaysLoadingWithItsSource`.

**What the gates found.** `lint` (QF1001) flagged a negated conjunction in a new test; it is written with
De Morgan's law.

## Batch 23 — UAT-2's fourth pass: maps that fill; room around Hawaii and the Caribbean; the badge (2026-09-26)

**U2-7 (D-90), in go-tuiMaps `v0.2.0-rc.12` (its L11.7, D-86).** The library drew nothing past 180°, as
its upstream did not, so Alaska at a wide window showed open sea where Chukotka is. The world now
repeats across the antimeridian:
- tiles are drawn per copy of the world;
- every overlay, marker and name is drawn once in each copy the view reaches, each whole;
- a first cut that chose each point's nearest copy tore lines across the frame, and the library's
  run-index test caught it.

No reference frame moved. watchpost at 190×50 over Alaska now draws Chukotka's coast.

**U2-8, U2-9 (D-91).** Hawaii is 172°W to 144°W by 10°N to 28°N, and the US Caribbean 78°W to 58°W by 11°N
to 23°N: about two levels wider each, so the weather around them is inside the region. Miami and Key West
stay in the lower 48.

**U2-10 (D-92).** The radar badge is three rows at the upper right: RADAR DATA, the source's chip, and the
frame's time in the listener's clock, STALE before it when old. It takes the library's top-row stamp
over (go-tuiMaps L11.8, `ShowStamp`, D-87), and the legend opens under it.

**What the tests found.** An edge test pressed keys on two copies of one Dashboard. The copies share the
library's map, so one copy's crossing moved the map under the other. It passed before only because the
Caribbean's old box was small. The test now presses in one line and says why.

**The go-tuiMaps bump:** `go get`, `go mod tidy`, the licence list regenerated.

**Mutation verdicts** (targeted, 6, and the library's 5 at L11.7 and L11.8), all caught in the end. The
Caribbean's box pushed north into the lower 48's survived at first: `RegionOf` looks in the lower 48 first,
so nothing it reports changed. The test now holds what D-91 promises: the two boxes do not overlap.

## Batch 24 — UAT-2's fifth pass: land past the antimeridian; Hawaii's radar reaches its hurricane (2026-09-26)

**U2-11, in go-tuiMaps `v0.2.0-rc.13` (L11.9).** Past 180° the coast drew but the land was painted sea.
The library's `World` painted ocean either side of the one world, and L11.7 had not moved it. L11.7's own
test counted line dots, not land. Beyond the world is now only above and below the poles; east and west
of it is the next copy. The upstream-parity test P-22 was brought to D-86, and it catches the old
painting.

**U2-12.** The radar's Hawaii box was cut to the old region, 17.5°N to 23°N, so the radar stopped in a
straight line south of the Big Island, where the hurricane is. Outside the lower 48 each box is now its
MRMS product's whole extent, from the services' capabilities:
- Hawaii 164°W to 151°W by 15°N to 26°N;
- the Caribbean 90°W to 60°W by 10°N to 25°N;
- Alaska 176°W to 126°W by 50°N to 72°N;
- Guam 140°E to 150°E by 9°N to 18°N.

Live: the newest Hawaii frame paints 4,141 pixels south of 17.5°N and 1,953 north of it.

**Mutation verdicts:** the library's `World` fix is caught by P-22; the boxes by
`TestEachBoxIsItsProductsWholeExtent`.

## Batch 25 — W10: temperature, and the map's two modes (2026-09-26)

**Rulings D-93 to D-98**, asked one at a time after live probes of the candidate sources:
- **D-93:** two sources join FR-3.8's closed list. NDFD (`graphical.weather.gov`) is the default, and Open-Meteo (`api.open-meteo.com`) is a Settings choice.
- **D-94:** the map has two modes. Radar mode is R on: everything drawn matches the frame's time. Forecast mode is R off: Now, Today, Tomorrow and Day 3 to Day 7, stepped. The chips gain R, and the Legend is on probation.
- **D-95:** over radar, temperature is isotherms plus a faint fill.
- **D-96:** Radar mode's temperature is always Open-Meteo, because NDFD has no past hours.
- **D-97:** a day draws its high, and `<` or `>` flips it to its low.
- **D-98:** alerts are drawn by their time.

**The probes that set the numbers.**
- NDFD answers only the first 100 points of a request, silently.
- NDFD reads an hour sent without its zone as each point's local time; it answered seven hours ahead.
- NDFD has no hour before the current one.
- Open-Meteo's free tier counts every point as a call, 10,000 a day. A lattice is therefore at most 80 points, which keeps four boxes refreshed hourly all day under the limit.

**go-tuiMaps `v0.2.0-rc.14`** (its D-88, L-15):
- **`Overlay.During`:** an overlay with a span is drawn only while the map's moment meets it, and stays prepared meanwhile.
- **`Map.ShowMoment`:** the host's moment while no loop is held.
- **The look:** a field that shares the map with an image is its labelled contours over faint bands.

Every hour's and every day's grid is therefore handed in at once, and a step or a frame moves only the moment. A grid swapped in at the step would blank until Work prepared it: U1-28's blink, once a step.

**The library gate's fuzzer found a third disagreement with the proven decoder** (L11.12, D-126's kind). This decoder read a point's second MoveTo command as more points. It is now refused (D-75), and the input is kept in the oracle's corpus.

**What the guards caught.**
- The Router carried neither new message back to Observer. With the console active, the temperature and Forecast mode's playback would have gone to the console and been lost.
- The frame's memo key did not cover the temperature or the step.
- Help's MAP group took two more rows and pushed Observer's row-mark legend out of its window. It now takes one: R and `<` `>` share a row.

**Every alert in effect is on now's frame.** An alert's span begins at its onset. An alert already in effect when it was issued after the newest radar frame, or inside the listener's hour, begins at the mode's now. Otherwise the map would hide a warning in effect until the next frame.

**Mutation verdicts** (targeted): 22 in watchpost and 10 in the library. One watchpost mutant survived, and it showed dead code: a loop trimming the lattice to 80 points could never run, since the rows are 80 divided by the columns. The loop is gone; the columns are capped instead, so a box very wide for its height still keeps two rows. Three library mutants survived the first tests and were caught by new ones:
- the frame reuse after a span change (a day's field swapped for another);
- `Changed` on `ShowMoment`;
- the contours, whose number the loop's stamp had supplied.

## Batch 26 — UAT-2's sixth pass: temperature's first (2026-09-26)

**U2-13, U2-14: every arrow in the Overlays menu asked the radar and the temperature again.**
- The temperature's grids stated their currency from the clock, so no two answers compared equal. Every press handed every grid in again, and each blanked until Work prepared it: U1-28's blink.
- A re-sent radar loop that the library refused took the loop already drawn off the map with it. Only a burst of presses could make that happen.

Now:
- An arrow moves the cursor and makes no library call. Only a switch touches the map, and the menu no longer asks the radar at all (R owns it, D-94).
- An answer within one hour is the same answer: currency counts from the hour's start.
- A refused refresh keeps the loop or grid drawn, and says so.

**U2-15, D-100: Today was blank because NDFD has nothing for today after its daytime** (live: none of 72 points at 20:06 PDT). Open-Meteo is asked only when a day is empty, and fills it. The badge's chip names Open-Meteo on that step and its credit is said. A day neither source has is still said.

**D-99: temperature is off by default.** While the map is open it is fetched for the mode shown and held, so switching it on in the Overlays menu draws at once. Nothing is fetched with no map open.

**Mutation verdicts** (targeted, 12), all caught. One survived at first: a day Open-Meteo also lacked was still marked filled, so the badge would have named it on an empty step. The test now holds that case.

## Batch 27 — UAT-2's seventh pass: one temperature look; Open-Meteo by default (2026-09-26)

**U2-17 and U2-20 had one cause: NDFD covers the US and its immediate waters, and the map shows more than that.**
- In the lower 48, a lattice cell with no US point around it stayed empty while its neighbours extrapolated. That left the square patches over Mexico and Canada; it was not the tiles.
- In Hawaii, NDFD refuses a whole request (HTTP 404) whose points straddle its grid's edge. One point inside plus one outside is refused; a lone outside point is answered, empty.

**D-101:** Forecast mode defaults to Open-Meteo, the same source as Radar mode. NDFD stays a Setting, and is drawn only where a cell's nearest point has a value. A box NDFD refuses is Open-Meteo's, said and credited; when every box is, the chip names it.

**D-102: one look in both modes.** Temperature is labelled isotherms over faint bands. go-tuiMaps `v0.2.0-rc.15` (L11.13, its D-89) lets a grid ask for that look (`Grid.Lines`), and the legend keys it as drawn. The frame and the legend read one faint strength, `colour.FaintField`.

**U2-18:** the key row's words are black or white, whichever reads on the band: `render.SwatchText`, never under 4.5:1.

**U2-19:** the badge is the HUM LEAD's layout. The test caught an 11-character step ("TODAY HIGHS") that overflowed the badge by a cell: `PadTo` of zero width is still one space.

**The library gate's fuzzer found a fourth disagreement with the proven decoder** (L11.14), again not this batch's. A feature carrying both `name_en` and `name:en` kept whichever tag came last. Upstream's order is `name_<lang>` first, and it now is here too. The input is kept in the oracle's corpus.

**Mutation verdicts** (targeted): 10 in watchpost and 3 in the library, all caught. One survived at first, the Today step's words; the badge test now walks Today.

## Batch 28 — UAT-2's eighth pass: the rows under the map; the parts shared (2026-09-27)

**D-103, D-104, D-105**, with the HUM LEAD's mock.
- **Blank Forecast mode:** entering Forecast mode with no main overlay on turns temperature on for Forecast mode alone. A chip at the map's top centre says so until a key. Back in Radar mode it is off; nothing is saved.
- **Overlays box:** it reads MAP DETAILS / OVERLAYS, in Settings' heading style, with a blank row between groups and before its keys.
- **Legend:** the box is retired and L is unbound.
- **Controls:** the box leaves the map for the rows under it, beside the timeline. The region keys are a row of their own, MAPS:.
- **Loop row:** RADAR [source] · FRAME · NEWEST (FR-5.4) · STOPPED in yellow or PLAYING in green. Forecast mode's reads FORECAST · STEP · state. The picture's own status and the estimate have a row that is blank when whole, so the map's size never moves with it.

**The parts, consolidated** (the HUM LEAD: "consolidated into re-usable components (like the scrub control) ... future layout changes might be less complicated/painful"). `map_parts.go` holds one of each, and the modes hand them their words:
- `chipBox`: the edge chip and the mode's chip;
- `mapBadge`: radar's badge and forecast's;
- `scrubber`: radar's loop and Forecast mode's steps.

`map_scrub.go` lays the rows under the map from them, the same in both modes. Radar's and Forecast mode's timelines were two copies of the same drawing; each is now a `scrubber` value.

**Mutation verdicts** (targeted, 10), all caught. Two survived at first:
- Forecast mode's second visit never re-showed its chip. The test now enters Forecast mode twice.
- The controls box drawn on the map as well: the test read the raw picture, not the window. It now reads the window.

## Batch 29 — the window stack; g Maps (2026-09-27)

**D-106, D-107.** From the map, the HUM LEAD opened a place's details with enter, and esc landed on the dashboard: `modal` was one value, and opening a window closed the one shown. The same gap had left D-37's discard-pile window unbuilt (F-108).

The windows are now a stack:
- A window opened from another opens over it, and esc, or the key that opened it, returns to the one below. That window resumes at the end of the Update, whoever closed what was over it: the map is drawn again and asks for its work, radar and temperature.
- A window already in the stack is returned to, never doubled.
- A search or confirmation window is replaced by what it opens, never returned to.
- Each window's scroll is kept while it is under.

Evaluated before building:
- `modal` is assigned only in `open`, so the forty-three reads of the window shown read the top of the stack unchanged.
- Nothing else in the tree changed; the only golden that moved is the controls row.

Key capture stays each window's own; F-184, next, moves it into the stack.

**D-106's first item:** the controls row above the table gains g Maps, left of ↑↓ Navigate.

**Mutation verdicts** (targeted, 7), all caught.

## Batch 30 — the stack routes the keys (F-184, 2026-09-27)

**Key capture was in three places:**
- `handleKey`'s switch named the windows that take every key, and the map, which takes the keys it binds.
- `handleNav`'s hand-written list named the windows whose own scroll the arrows walk. A window once missing from it let its arrows reach past it to the table underneath.
- Help named the map's group by its title.

Each window now declares, once, how much of the keyboard it claims (`window_keys.go`). The routing, the windows' nav and Help read the declaration; only the window shown is asked. The change is a consolidation: behaviour is unchanged, and every existing test passed unaltered.

**The guards** (`window_keys_test.go`) derive the windows from the constants and hold every one to its declaration:
- declared, and what takes its keys named;
- its arrows its own, never the table's;
- left by esc (D-62);
- no key bound twice;
- its Help group listed;
- the window shown takes its keys before the Observer's;
- a form - the four D-107 names - taking every key: a stray g, q or ? never leaves it.

**Mutation verdicts** (targeted, 7). One survived at first: Setup declared as claiming no keys passed every guard that read the declarations. `TestAFormTakesEveryKey` names the form windows as policy and holds them by behaviour; it catches it.

**The mutant sweep's anchor gate caught a drift.** mW4, "the arrows reach past the card window", was anchored to the hand-written list this batch replaced. It is re-pointed at the card window's declaration, and both FR-5's reachability test and the new guard catch it.

## Batch 31 — W11: wind (2026-09-27)

**D-108 to D-110.** Wind works as temperature does:
- Radar mode draws each frame's own hour.
- Forecast mode draws Now's wind and each day's peak with its dominant direction.

Wind is braille arrows with speeds and no fill, drawn over radar and temperature's faint bands alike. It is off by default, loaded with temperature, and counted by D-103 as a main overlay.

**The data rides temperature's requests.** Open-Meteo adds four variables; its free-limit cost is unchanged. NDFD adds wind speed and direction to both requests. Its day answer starts at the next hour, so the current hour's request carries wind too, and a test now watches for it. NDFD has no daily wind, so each day's peak is its strongest hour, with the direction then.

**Direction is interpolated as a vector**, by its east and north parts: as a number, 350 and 10 degrees would meet at 180, the wind turned round.

**go-tuiMaps `v0.2.0-rc.16`** (its D-90, L-16; FR-8, deferred since v0.1.0):
- `Grid.From` makes a vector grid; `WindGrid` builds one in a call. It is drawn as arrows pointing downwind, length and colour by class, every other labelled.
- Six wind classes, round in each unit. Six tokens, `wind.1` to `wind.6`, come after every other, so none moves.
- The ramps were searched for against the library's checker as line work, on both grounds and at both depths.
- A place's answer gives the speed and the compass word it blows from, for M1's wind scenario.

**Mutation verdicts** (targeted): 10 in watchpost and 9 in the library, all caught. Two survived at first:
- The wind unit's breaks: the legend test now reads the calmest class's words.
- NDFD's current-hour wind: the request test now reads the query.

## Batch 32 — every region draws its fields whole (D-111, 2026-09-27)

**U2-26.** In Hawaii the wind stopped at a rectangle inside the map. Temperature and wind had been asked for the radar's fixed boxes, and Hawaii's radar box is MRMS's extent, narrower than the region D-91 widened. The Caribbean and Guam had the same mismatch in smaller measure. American Samoa, with no radar box, drew neither.

The HUM LEAD: "all of our regions should have some kind of data because we say watchpost's pre 1.0 release is for US (and territories) users."

Temperature and wind now have fixed boxes of their own (`fieldBoxes`). The lower 48 keeps the radar's boxes. Elsewhere each box is the whole map region, still fixed and never the view (D-47). Alaska is split at the antimeridian, which a grid cannot cross: two boxes, one more request there. Hawaii's points are coarser, about 2.6 degrees apart where they were 1.4. The map's view never leaves its region, so a region's box always covers it.

**Mutation verdicts** (targeted, 2), both caught.

## Batch 33 — the loop's hours ahead (W12.1, D-112 to D-116, 2026-09-27)

**D-112** set the order of the overlays to come. **D-113 to D-116** ruled rain and snow ahead:
- both modes;
- three hours ahead by default, a Setting;
- Open-Meteo's model rain outside the lower 48;
- Forecast mode's days in radar's colours with totals.

This batch builds the part that changes the controls, UAT'd first as the HUM LEAD asked.

**Probed live.** IEM, on the closed list, publishes NCEP HRRR forecast reflectivity with each run's start beside it. Its images read clean with IEM's existing colour table: the library raised no unmatched-colour warning. No new host and no new table.

**The loop.** A forecast loop a box, of HRRR's quarter-hours after the newest observed frame and up to the horizon:
- The observed loop is drawn until its newest frame and the forecast from its first (go-tuiMaps L-15.1), so no moment shows both.
- The forecast frames fit what the observed loops leave of the image budget; the farthest are dropped first.
- At a forecast frame the badge reads RADAR FCST with HRRR's chip, purple, a colour of its own because a model is not radar, and the loop's row leads FORECAST.

**A deviation from D-114's words.** The frames are fetched at half the radar box's size, about 8 km, not "at HRRR's resolution": a radar box is drawn at about 4 km, and twelve frames at full size would crowd the budget the observed loop needs.

**A library defect found by the first test, go-tuiMaps L11.16.** "Right now" became the far end of the forecast. Each loop offered its own newest frame, and a loop of forecast frames alone offered a forecast. Right now is now the newest observed frame of every loop, as L-1.10d says.

**Mutation verdicts** (targeted, 9), all caught.

## Batch 34 — rain and snow ahead beyond the lower 48, and Forecast mode's days (W12.2 to W12.4, D-115 to D-118, 2026-09-27)

**D-117 and D-118** were ruled at the start:
- **D-117:** Forecast mode's rain and snow is an Overlays row, on by default, so Forecast mode mirrors Radar mode. Temperature and the rest are the tints under it, and the map says in words that it is a model's rain, not radar.
- **D-118:** the rain and snow is Open-Meteo's everywhere and on every day, whatever the temperature's source.

**Beyond the lower 48 (W12.2, D-115).** The loop's hours ahead are Open-Meteo's hourly precipitation, a frame an hour:
- Each frame follows the newest observed frame, up to the horizon.
- Marshall and Palmer's relation (Z = 200 R^1.6) converts each rate to radar's scale.
- Each frame is painted as a PNG with an exact table of the app's own, a colour for each half dBZ. The library reads the classes from it and draws them in its own radar colours (go-tuiMaps D-45).
- A dry hour is transparent: no echo, never radar's lightest class.
- The frames join the observed loop as HRRR's do (`joinAhead`, shared).
- The chip reads O-METEO, and a note says the hours ahead are a model's rain, not radar.
- The request is a field box's lattice for the hours ahead, about 32 KB. The radar's cost line counts it in place of HRRR's frames.

**Forecast mode's days (W12.3, D-116).** A request a field box, seven days hourly with each day's rain, showers and snowfall, gives:
- Now's hour and each day's heaviest hour, each on the point's own local date, as grids in radar's scale, drawn as rain (go-tuiMaps L-17).
- Each day's total marked at the arrows' spacing. A day with snow marks its snow, apart with a `*`; otherwise its rain. Inches, or mm of rain and cm of snow. A trace marks nothing.
- The Overlays row appears in Forecast mode alone, on by default. D-104's temperature still turns on under it as the tint.
- The colour row under the map is radar's colours, headed MODEL RAIN · NOT RADAR (D-117's words on the map). The credit is under the map.
- With temperature off, the badge's step reads e.g. TODAY RAIN.

**The fixtures.** Open-Meteo was recorded over south-east Alaska, a lattice across the border with British Columbia. Its points answer in two zones an hour apart, and the test holds each day's heaviest hour to the point's own date.

**D-118's measure, corrected.** The ruling said about 250 KB a box. Measured with the request as built, it is 4.7 KB a point, about 375 KB for a lattice of 80, and the cost line counts that.

**A library change, go-tuiMaps L-17 (its D-91), rc.18.**
- **Rain grids:** a grid in radar's scale is drawn as rain: full colour, over the sea, over any field, never lined. A field beside it takes its lines.
- **Marks:** `Grid.Marks` carries the day's totals, written a cell apart before any value a field writes. Where a total is, it takes the place of a wind speed beside it.
- **A legend defect found building it (L11.18):** the legend had keyed radar's classes one colour off from the frame, the heaviest in a temperature colour past the ramp's end. That was true of every radar legend, images' too.

**W12.1's defect, found building W12.2 (W12.4).** With the hours ahead in the loop, the newest frame's age read the forecast's far end. The loop row said "NEWEST -55 MIN AGO" and could never say STALE (FR-5.4). The row, the status line and the badge now read the newest observed frame, the library's `LoopState.Now`.

**Mutation verdicts** (targeted, 20 in watchpost and 9 in go-tuiMaps), all caught. Three survived at first, and each was a real gap the tests closed:
- **Library, rain beside temperature:** a temperature field beside rain kept its full bands, because the whole-frame rain hid them. Rain over the sea alone now shows the land's temperature lined.
- **A newest frame on the hour:** a forecast frame at that same hour would have shown the moment twice. The test now puts the newest observed frame on the hour.
- **Back into Radar mode:** Forecast mode's held rain stayed drawn over the radar until the next answer. The test now checks at the switch.

## Batch 35 — feels like, and the badge a tab (W13.1, W13.2, D-119, D-120, 2026-09-27)

**D-112's second stage opens.** D-119 ruled on feels-like: its own row of the Overlays menu, "exclusive". Switching it on turns Temperature off, and the reverse, because the two share one tint.

**Probed live.**
- **Open-Meteo:** gives apparent temperature hourly and each day's high and low.
- **NDFD:** gives it (`appt`) hourly, then every few hours to seven days, with no daily value. Each day's high and low are worked out from its hours on the local date, as wind's peaks are.
- **Both:** it rides the requests temperature already makes, and it costs nothing of its own.

**NDFD ignores the hour asked for feels-like.** Its answer begins at the next hour, a trap recorded in the fixture. Forecast mode's Now is filled from Open-Meteo, credited, as D-100 fills a missing day. A day the source lacks is filled the same way.

**At the window.**
- Feels-like is drawn as temperature is: isotherms over faint bands.
- In Radar mode, each frame draws its own hour. In Forecast mode, Now and each day's high, or its low with `<`.
- With feels-like on, Forecast mode does not turn temperature on (D-104).
- The colour row reads FEELS LIKE, and the badge's step e.g. TODAY FEELS HIGHS, or NOW FEELS LIKE.
- The badge's step no longer says PEAK when wind is on beside feels-like.

**The badge is a tab (W13.2, D-120).** Feels-like's step would not fit the three-row badge's 12 cells, and the HUM LEAD redrew it: "stay on 1 line and widen/shrink as needed".
- The badge is one line, e.g. `RADAR DATA  [  MRMS  ]  12:55 AM` or `FORECAST  [ NDFD ]  TODAY HIGHS`: the mode, the source's chip as it was, and the moment, STALE before it when stale.
- It's a tab joined to the frame's top right. The top edge opens into it (┬), and its bottom edge closes into the frame's right side (└…┤).
- The window is built as a panel, the tab is spliced into its frame rows, and then the window's tones are laid.
- The three-row splice over the map's body is gone.
- In a narrow window the title gives way, never the tab: the title is shortened with an ellipsis to end before it, because the tab carries the moment and STALE (FR-5.4).

**Test-first, broken once and repaired.** The domain's code was written before its test. The tests were written next against recorded answers, and each change was then mutated away to see it caught.

**Mutation verdicts** (targeted, 24): 23 caught. The one survivor is equivalent: the line that gives a feels-like hour its local day. NDFD sends feels-like and wind in one time layout, and wind's line already sets every hour. It stays, because it matters where feels-like is asked without wind.

## Batch 36 — fire (W13.3, D-121, 2026-09-27)

**D-121** ruled "All three, on": the active perimeters, the named incidents and the satellite hotspots, one Fire row, on by default as alerts and quakes are (D-76).

**What it reads.**
- **Incidents:** WFIGS's national layer, the query and memo the places' fire already makes, now exported to the map (`wfigs.Incidents`).
- **Hotspots:** HMS's archive, the places' own coalesced read (`hms.Points`). A truncated archive is served as read.
- **Perimeters:** the one new request, WFIGS's interagency perimeters on the same host. It is asked for each field box the view is in, generalised to about a thousandth of the box's width. A perimeter in two boxes' answers, as in Alaska across the antimeridian, is drawn once.
- **The Status window:** its MAP block names NIFC WFIGS and NOAA HMS.
- **FIRMS:** it is keyed and asked by box for the places. It is not on the map yet; HMS stands for the satellites.

**At the map.**
- Perimeters are outlines with their holes, unlabelled.
- Each incident in view is a marker labelled with its name, acres and containment.
- Hotspots are dots, filtered by the places' rules: the strong (the rules' bold, 50 MW) bright, the rest fainter.
- All of it is timed as a thing so now: through the loop, and on Now alone in Forecast mode.
- The perimeters' cost is each box's share of the lower 48's measured 378 KB.

**A library change, go-tuiMaps L-18 (its D-92), rc.19.** The library's feature roles were the alerts' (a severity, reported over a place), the track's (the earthquakes' yellow in watchpost), and three field roles that resolve to nothing on their own. Fire has two roles of its own: `fire` (red-orange) and `fire.faint` (fainter), placed after every other token so none moves. A feature in them is never an alert.

**Mutation verdicts** (targeted, 16 in watchpost and 4 in go-tuiMaps), all caught. Three survived at first, and each was closed with a test:
- a missing default colour for a fire token;
- a truncated HMS archive served with its error;
- perimeters left un-de-duplicated across boxes.

## Batch 37 — quakes as USGS draws them (W13.4, U2-27, U2-28, D-122, D-123, 2026-09-27)

**UAT on fire (U2-27).** The fire works, and there is a lot of it: at national zoom it crowds the radar and runs the drawing over budget. It is carried to D-112's fourth stage as sub-filters and a cap on what the map draws, not patched here.

**Quakes (U2-28).** The map drew the ticker's "significant this week" feed, which held one quake worldwide and none in the US when asked.
- **D-122:** the map reads a USGS summary feed of its own. The past week is the HUM LEAD's default, with the past day a Setting. Magnitude is M2.5+ by default (the HUM LEAD's "M2.5"), with M1.0+ a Setting. One row, "Quakes -", offers the four. Its file word is the feed's own name, e.g. `2.5_week`, and the cost line counts the feed measured: 33 KB to 918 KB.
- **D-123:** each quake is drawn as USGS draws it:
  - a braille ring round the epicentre, the same size on the screen at every zoom, half again larger with each magnitude (3 dots at M2.5, 8 at M5, 17 at M7, capped at 48);
  - coloured by age: the past hour, the past day, older;
  - labelled with its magnitude and local time in the listener's clock, e.g. "M3.1 2:14 PM", and the weekday when not today.
- **The Status window:** its MAP block names USGS, which the map now asks itself.

**Library, go-tuiMaps L-19 (its D-93), rc.20.**
- A circle may be sized on the screen (`RadiusDots`, 1 to 48), drawn by the markers' own circle, with its label beginning beside it.
- Three quake roles by age.

**The library's gate had a defect of its own (L11.21).** A gate run stopped with TERM was logged "green" after five seconds, a line M6 would count as clean. The false row was taken out by hand, and the gate now traps INT, TERM and HUP: a stopped run exits non-zero and is logged INTERRUPTED. Its test reproduced the defect first.

**Mutation verdicts** (targeted, 14 in watchpost and 6 in go-tuiMaps), all caught. Three survived at first, and tests now catch each:
- a ring drawn as a dot;
- a ring's label drawn over it;
- the ring's cap, untested below M9.55.

## Batch 38 — errors with a path, or to the diagnostics (W13.5, U2-29, D-124, 2026-09-27)

**UAT-2 U2-29.** With quakes on, the map filled with "An alert could not be drawn: tuimaps: bad-currency ..." until it was pushed off the window.
- **The cause:** every quake was handed in current for eight days, and the library keeps a thing current for at most seven. The old significant feed was nearly always empty, so this never showed.
- **Made worse:** the window wrote a note for every refusal, each called "an alert".
- **The test:** every quake a feed can hold, from a minute to a full week old, is now handed to a real map and accepted. It reproduced the refusal first.

**D-124, the HUM LEAD's rule:** "We should never show error messages to the end user unless we give them a path to resolve it." "Diagnostics only" for the rest. The map's messages were audited against it:

- **Refusals, which are ours to fix:** the diagnostics alone, once a layer. That covers the feed's overlays, the radar's loop and the temperature's grids; `tempRefused` is gone.
- **The map failing to start or draw:** one sentence with its way back, "Close it with esc and press g to open it again", and the detail to the diagnostics.
- **A source that did not answer, where a Setting offers another:** said with that Setting. This is NDFD's temperature, and the lower 48's radar (IEM or MRMS).
- **A source that did not answer, with nothing to change:** the diagnostics alone. That is radar outside the lower 48, the hours ahead (HRRR or Open-Meteo), rain and snow, and Open-Meteo's temperature.
- **Kept, because they say what is drawn and are not errors:**
  - an alert's missing zones (FR-4.4);
  - NDFD's gap filled from Open-Meteo, credited;
  - the offline basemap;
  - MRMS's approximate colours;
  - the description's missing outline.

**The diagnostics.** The window hands such problems to `Config.MapProblem`. The app keeps the last fifty, time-stamped, in `mapProblems`, and the diagnostic dump writes them as `map_problems` in `counters.json`.

**Mutation verdicts** (targeted, 12), all caught. One survived at first: the window dropping the app's problems. A test now catches it.

**The dupes gate** found `mapProblems.last` repeating `bandRecord.recent`, a lock and a copy. Both now use the app's `lockedCopy`. `bandRecord`'s bound, mutant mC5's anchor, is untouched.

## Batch 39 — wave height (W13.6, D-125, D-126, 2026-09-28)

**Rulings.**
- **D-125, the source:** "NDFD, Open-Meteo beyond". The NWS's own coastal numbers where NDFD reaches, which is where boaters are. Open-Meteo Marine fills beyond, on its own host, marine-api.open-meteo.com, which joins the closed list.
- **D-126, the look:** "Sea bands, row off". The waves are the sea's alone, in a wave scale of their own with labelled contours, as temperature's are.

**Probed and recorded.**
- **NDFD** answers `waveh` in feet, hourly from the next hour to six days, offshore to 121°W, and nil ashore. The current hour is therefore Open-Meteo's, as feels-like's is.
- **Open-Meteo Marine** answers in metres and puts its sea points in a fixed UTC−8 zone, not California's daylight time. That trap is recorded in the fixture.

**The merge** works point by point, hour by hour and day by day: NDFD's value where it has one, Open-Meteo's where not. NDFD's point matching and time layouts came out of `parseDWML` so the wave reader shares them.

**At the map.**
- A Waves row, off by default, loaded in the background like the rest (D-99).
- Radar mode draws each hour; Forecast mode draws Now and each day's highest. Values are in feet or metres by the listener's units, and the credit names both sources.
- A failure goes to the diagnostics (D-124): no Setting offers another source.
- The cost line counts two requests a field box, about 295 KB for 80 points.

**Library, go-tuiMaps L-20 (its D-94), rc.21.**
- A `waves` preset in feet or metres, with six classes set round in each unit and tokens `wave.1` to `wave.6`.
- The colour scale was searched for against the library's checker on each ground's water, at both depths. Higher waves are lighter on the dark ground and darker on the light.
- A wave grid is drawn over the sea alone, the inverse of every other field's shore.

**A gap in the record, closed.** FR-3.8's closed list had not been brought up to date for fire (D-121) or quakes (D-122). The table now lists NIFC WFIGS, NOAA HMS, USGS and Open-Meteo Marine. The closed-list test also holds every host the Status window names to it, so the map cannot contact a host the list lacks.

**Mutation verdicts** (targeted, 11 in watchpost and 5 in go-tuiMaps), all caught. Two survived at first:
- waves painted over land, because the test's land box lay outside the test's view;
- Open-Meteo overwriting NDFD in an hour both answer.

Each is now caught.

**The dupes gate** found the waves' `hourIndex` repeating the rain's. Both now call one `hourRow`.

## Batch 40 — wave height's UAT pass (W13.7, U2-30 to U2-32, 2026-09-28)

**U2-30, the waves stopping at a straight line in the Pacific.** The lower 48's field boxes were the radar's, 126°W to 65°W, but its region reaches 130°W to 64°W. So temperature and wind stopped there too, unseen because they don't draw over the sea; waves made the strip visible. The outer boxes now reach the region's edges at any view. D-111's test had waved the lower 48 through ("the radar's boxes, as they were") and now holds it to the edges too.

**U2-31, bands short of the coast.** A sea cell whose nearest lattice point lay ashore was left blank. That is D-101's rule, right for a source's reach but wrong for waves, which the library already keeps to the sea. `Lattice.InterpolateOut` first gives a point without a value the mean of its neighbours, round after round, so the cells by the coast carry the sea's value. This also removes the blocks the blanks made.

**U2-32, the bands coming and going as the loop played.** Radar mode's hourly fields (temperature, wind, feels-like, waves) stopped at the current hour. Since D-113 the loop plays on into the hours ahead, so every frame past the hour had none.
- The fields are now drawn to the loop's horizon.
- Open-Meteo is asked for thirteen hours ahead, not two: the longest horizon, twelve, and the hour it ends in.
- An hour ahead is stamped with the current hour, as the forecast days are. Stamped with its own hour, a grid twelve hours ahead would have had no currency left, the library's refusal of U2-29's kind. A test hands every hour to the horizon to a real map.

**The cost line** was re-measured with the thirteen hours:
- Open-Meteo's temperature request: 139 KB for 80 points. It had been counted at 80 KB since before feels-like was added, so the old figure was already short.
- The marine request: 72 KB.

**Mutation verdicts** (targeted, 7), all caught. Three survived at first, and tests now catch each: the waves not filled, an hour ahead stamped with its own hour, and the thirteen hours not asked.

## Batch 41 — the scrubber on one axis (W13.8, U2-33, 2026-09-28)

**UAT-2 U2-33.** "The 'NOW' hatch deosnt seem to properly align with the correct 'now' frame ... when the map first loaded in, it looked like the first frame rendered was in the forecast zone. This one is important to get right because the scrub controls also provide 'legend like' functionality."

**The cause, a defect of W12.1's making.** The scrubber drew its NOW mark, its clock times and its OBSERVED and FORECAST halves by time, but its cursor by frame number. Before the hours ahead, every frame was five minutes apart and the two agreed. Since D-113 the frames are five minutes apart observed and fifteen ahead: 24 frames over two hours, then 12 over three.
- The newest observed frame is two-thirds along the frames but two-fifths along the time. The map opened on it, correctly, and drew it in the FORECAST half.
- The frame under the NOW mark was about an hour old.

The test reproduced it first: the cursor at 0.733, NOW at 0.478.

**The fix.** One axis, time. The cursor is where the frame shown lies in time along the loop, as NOW and the clock times at the bar's ends already were. At now the cursor sits on NOW; a forecast frame lies past it; stepping into the hours ahead moves the cursor in longer strides, as the frames are farther apart.

**Mutation verdict.** The cursor by frame number again is what the test caught first.

## Batch 42 — the sea's stations (W13.9, D-127, D-128, 2026-09-28)

**Rulings.**
- **D-127, "Row off, waves+water":** a station that read in the last two hours is a marker labelled with its wave height and water temperature, "4ft 73°", or its wind in knots, "12kt", where it reads no waves.
- **D-128, "Next tide, zoomed":** every tide station in view is a marker; when twenty or fewer are in view, each is labelled with its next high or low, "H 3.9ft 4:01 PM"; a wider view shows the markers alone.

**What they read.**
- **Buoys:** NDBC's one national file of latest observations, 103 KB every ten minutes, about 870 stations; its "MM" reads as nothing.
- **Tides:** CO-OPS's tide-station list, the one the places' tides read daily. Each station's next tide comes from its predictions, the places' own request and cache.

**Asked only while on.** The tides are a request a station, so both rows fetch only while switched on: the ask says whether each is on, and switching a row asks the feed again. Waves and temperature, which cost one request a box, are still loaded in the background (D-99). This is a deliberate difference.

**A deviation from D-128's words.** The ruling said an answer kept "six hours". The predictions are cached as the places' are: to midnight UTC, at most an hour at a time. That still asks each station at most once an hour.

**At the map.**
- Their own roles, go-tuiMaps L-21 (its D-95), rc.22: `buoy` in pink and `tide` in green, apart from every other feature role and the waves' scale, and never an alert.
- The Status window names NOAA NDBC and CO-OPS, and both join FR-3.8's closed list.
- The cost line counts NDBC's one file, and a request for each tide station in view while twenty or fewer are.

**Mutation verdicts** (targeted, 11 in watchpost and 1 in go-tuiMaps), all caught.

## Batch 43 — the basemap always shows (W13.10, U2-34, D-129, 2026-09-28)

**The finding.** With Buoys and Tides on, `1` for the lower 48 drew markers over an empty frame
(U2-34). Each station was an overlay of its own, hundreds at a national view. The library's queue
holds 256 jobs; full, it dropped its oldest - the view's tiles - and ran the overlays in the order
asked. The frame's notice, meanwhile, told the listener to "call Settle, or run Work".

**In the library, go-tuiMaps L-22 (its D-96), rc.23.**
- Past the cap the queue drops a job of a view no map shows first, then the oldest that is no tile,
  a tile last; within a view it runs the tiles first. The basemap cannot be starved by overlays,
  however many a host hands in.
- While the tiles load the frame says "Loading the map…" and names no call (D-124): the host learns
  it from the `NoTiles` status.

**In watchpost, D-129 ("whichever option gives us better performance").**
- Every layer of many things is one overlay: the quakes, the buoys, the tide stations, fire's
  perimeters and its incidents. The same view that handed the map over 1,200 overlays at batch 42
  hands it five.
- A quake no longer comes in on its own frame of the loop; one within the loop's two hours past is
  labelled NEW ("NEW M3.1 2:14 PM"), beside USGS's age colour (D-123).
- Alerts stay one an alert: each has its own times (D-98).

**U2-35, the radar slower to load**, is most likely the same flood - hundreds of overlay jobs ahead
of the radar's - and is re-checked in UAT (S31). The performance, structure and quality pass the HUM
LEAD required before SHIP is plan W14, for both repositories.

**A time bomb from batch 42, found on the way.** `TestTheSeasStationsAreAskedOnlyWhileOn` served
NDBC's recorded file as it was recorded; two hours later every reading was past D-127's two hours and
no buoy was drawn, so the test failed on the clock alone - as CI would have. The file is now served
with its readings ten minutes old.

**Mutation verdicts** (targeted; 5 in go-tuiMaps, 2 in watchpost), all caught. One survived at
first - the drop of a view no longer shown - and gained `TestAViewNoLongerShownGoesFirst`. The
wiring test `TestEveryLayerOfManyThingsIsOneOverlay` was run against batch 42's code and failed
there, 1,200 overlays for five.

## Batch 44 — the radar in 1.5 s cold (W13.11, U2-35, D-130, 2026-09-28)

**The finding.** After batch 43 the radar was still the last thing drawn, "at about 15s" (U2-35).

**Measured, live, the lower 48 cold.**
- The observed loop took 4.8 s: 24 frames one after another, and the radar client at the default
  pace of five requests a second - a frame every 200 ms. The time was the client's own pacing,
  not the network.
- HRRR's hours ahead took 2.7 s more, and were begun only once the observed loop was whole.
- The library prepared all 36 frames in 0.1 s: not the cause.
- `1` pressed while another region's loop was in flight waited for it (one ask at a time, D-85),
  then fetched the new region's: two cold loops, about 15 s.

**D-130, "Fix both now".**
- The frames are fetched six at a time, as the zones are (D-46, NFR-4), each in its place in the
  loop; the radar client's pace is 30 a second, so six can run.
- HRRR's hours ahead are fetched alongside the observed loop, and joined to it once it is whole: the
  join still needs the room the observed loop leaves (D-114).
- A change of region cancels the loop in flight for the region left; its answer, if it lands, is not
  handed to the map, and the new region's loop is asked at once. A pan or zoom within a region still
  waits for the loop in flight (D-85).
- Measured again, live: 1.26 s without the hours ahead, 1.45 s with three - all 36 frames. D-46's
  loop arm is ≤ 5.0 s.

**Not changed.** Beyond the lower 48 the hours ahead are Open-Meteo's model rain (D-115), a request
or two a field box, still asked after the observed loop. W14 measures it with the rest.

**Mutation verdicts** (targeted, 6), all caught. One survived at first - six in flight changed to
one - because the test compared the peak with the constant it was changing; it now holds the six
D-130 ruled.

## Batch 45 — the badge row (W13.12, U2-36, D-131 to D-133, 2026-09-28)

**The finding.** With every layer on, the notes under the map ran to six lines: MRMS's approximate
colours, three credit sentences, that each radar frame draws its hour's temperature, the cost
warning and the estimate (U2-36). The HUM LEAD drew one badge row in their place.

**Rulings.**
- **D-131, "Chips + full in Status":** the badge row credits in short chips; the full credit lines
  are in the Status window's MAP block. CC BY 4.0 allows credit "in any reasonable manner based on
  the medium", including pointing to where the details are.
- **D-132, "Fixed ones move":** the two notes that never change go to the Status window; MRMS's
  chip reads MRMS≈ (MRMS~ in `--ascii`). Notes that come and go keep a row, only while one applies.
- **D-133, "True sources":** FIRE [NIFC]/[HMS], not the mock's [FIRMS], which the map's fire does
  not read.

**As built.**
- One badge row under the colour row, in the Overlays menu's order: a layer off, one not yet drawn,
  and the radar have none. A second row only when the window cannot hold one; no badge is cut.
- Layers whose sources never change say them as they register (`MapLayer.Chips`). Temperature,
  feels-like and wind say theirs as drawn - [O-METEO], [NDFD], or [NDFD]/[O-METEO] where Open-Meteo
  filled what NDFD lacked - and waves and rain beside them (`MapTemperature.Chips`).
- The Open-Meteo, Open-Meteo Marine and MRMS entries in the Status window carry the full lines
  (`MapSource.Notes`).
- The cost warning is one line in the HUM LEAD's words; the estimate sits under the timeline's
  right end, off the status row.

**U2-35 re-checked by the HUM LEAD:** "much better - radar showed up in < 3s".

**Mutation verdicts** (targeted, 10), all caught. The three app tests were first run with the app's
new fields in place, so each was held RED by a mutant instead.

## Batch 46 — a chip a source (W13.13, U2-37, D-134, 2026-09-28)

**The finding.** Batch 45's chips were three kinds:
- MRMS, IEM and HRRR had grounds of their own, in white;
- O-METEO and NDFD borrowed IEM's orange and MRMS's green;
- the rest were plain.

They were also padded one space or two, depending on where they were drawn. The HUM LEAD: one
format, "  NAME  ", bold, a ground each, black or white words by contrast (D-134).

**As built.**
- `tty.chipFace` draws every chip, in the badge tab, the loop's row, the status line and the badge
  row.
- `render.ChipTones` reads the theme's ground and picks bold black or bold white, whichever has the
  higher contrast, so a yellow chip is never white-on-yellow. It replaces the one fixed white.
- Eight new grounds:
  - the default theme: saturated, a hue a source round the wheel;
  - the light theme: pale, as its test requires;
  - the no-colour theme: eleven greys, each its own.
- `TestEveryChipIsItsOwnAndReadsInEveryTheme` measures every chip in every theme, for uniqueness and
  AA. The AA register excuses the grounds by name, because its fixed-foreground model cannot hold a
  chosen one, and points to that test.
- The app's test holds every source it names to a chip of its own, so a new source cannot be drawn
  plain.

**Mutation verdicts** (targeted, 5), all caught.

## Batch 47 — place names before the data (W13.14, U2-38, D-135, 2026-09-28)

**The finding.** With every layer on, the place names were lost (U2-38). The library placed its
words in an order set when the only overlay words were an alert's: every overlay's label - now a
buoy's reading, a tide, a quake's magnitude and time, a fire's name - and every contour's value
claimed the map's cells before the basemap's names, which found none left.

**D-135, "Names before data"; go-tuiMaps L-23 (its D-97), rc.24.** The order is now:
1. the host's places;
2. an alert's words, and a marker's;
3. the alerts' digits;
4. the basemap's names, within their budget;
5. every other overlay's words;
6. the contours' values.

A reading or value with no room left goes unlabelled; its marker or line still draws. This
overturns the library's D-124, which put a field's values before the names.

**The test** was run against rc.23's code first, and reproduced the finding: Minnesota, Iowa,
Washington, California, Idaho and Michigan, all drawn with the field alone, were lost under its
values and three hundred stations.

**Mutation verdicts** (targeted, 5 in go-tuiMaps), all caught. Watchpost's change is the library's
version.

## Batch 48 — wind gusts (W13.15, D-136, 2026-09-28)

**D-136, "On the wind arrows".** Gusts are the Wind layer's. An arrow's label reads "15G30" -
sustained, G, gust, in the listener's unit - where the gust beats the sustained wind by 10 mph
(16 km/h), the METAR rule, and "15" elsewhere. Radar mode has each hour's; Forecast mode has Now's,
and each day's peak sustained with its peak gust.

**What it reads.** No new request - the gusts ride the requests already made for temperature and
wind (D-108):
- **NDFD:** `wgust`, hourly in knots, asked with the days and with the current hour; each day's
  strongest worked out, as its peak wind is.
- **Open-Meteo:** `wind_gusts_10m` hourly and `wind_gusts_10m_max` daily.
- **The fixtures:** recorded live through the sources' own requests (2026-09-28T23:09Z), and the
  tests hold every request to asking for the gusts.

**The library, go-tuiMaps L-24 (its D-98), rc.25.**
- A wind grid may carry `Gusts`, one a value, NaN where none is said, and the arrow reads "15G30".
- The library applies no rule. Watchpost decides which gusts are worth saying.

**Filled days.** A day whose wind NDFD lacks takes Open-Meteo's peak (D-100), and its gust comes
with it.

**A missing value made every answer new.** U2-13's test caught it on the way: the window hands in
only an overlay that changed, and judged "changed" by `reflect.DeepEqual`. A gust not worth saying
is NaN, and NaN is never equal to itself, so every wind grid would have been handed in again at
every answer, and blinked - as would any grid with a value missing, which the uniform fixtures had
never had. `tty.SameOverlay` now judges it, a missing value matching a missing one, at all three
places the window hands overlays in.

**Mutation verdicts** (targeted; 14 in watchpost, 4 in go-tuiMaps), all caught. One survived at
first: dropping `wgust` from the current hour's request, since the fake answered whatever was
asked. The test now reads each request.

## Batch 49 — UV and air quality (W15, D-137 to D-140, 2026-09-29)

**Rulings.**
- **D-137, "Own row, UV scale tint":** UV in its own row, as a faint tint with labelled contours.
- **D-138, "Both":** the model's US AQI for the picture, and AirNow's measured AQI over it.
- **D-139, "One row, both":** one Air quality row draws both.
- **D-140, colours:** the official colours failed the library's checker. Red-green colour vision
  could not tell UV's Low from its Very High. So the HUM LEAD asked for "a color band that's close
  but fits our standards", with a legend to read it by, and every scale's colours kept as tokens.

**What it reads.** None of it needs a key.
- **UV:** Open-Meteo's `uv_index` and `uv_index_max`, in the temperature request already made -
  free where Open-Meteo is temperature's source, asked only while UV is on where NDFD is.
- **Air, the model:** Open-Meteo's air-quality API, `us_aqi` hourly for five days, a request a
  field box. It has no daily value, so each day's worst hour is worked out on the point's own date.
- **Air, measured:** AirNow's `reportingarea.dat`, the whole nation in one 1.9 MB file. An area's
  AQI is its primary pollutant's.
- **A trap in AirNow's file:** a forecast's day offset counts from its issue, so yesterday's
  "tomorrow" is today, and both issues are in the file. Each forecast is placed by its valid date
  against the area's own today, and the latest issue is kept. Most forecasts give a category alone,
  drawn in that category's colour.

**At the map.**
- UV and Air quality are in the one-tint group with Temperature and Feels like.
- Switching either on asks again, since each is asked only while on.
- AirNow's areas are one overlay of dots in their category's colour. Their words are drawn in the
  markers' ink, so a pale class still reads on the light ground. They are labelled while thirty or
  fewer are in view.
- In Forecast mode, today's and tomorrow's steps show AirNow's own forecast.

**The library, go-tuiMaps L-25 (its D-99), rc.26.**
- The `uv` and `aqi` presets, their scales searched to pass the checker on both grounds.
- `UVGrid`, `AirQualityGrid`, and `AirQualityRole`.

**Also.**
- The waves' answer became a general `Measure`, shared with the air's.
- AIRNOW has a chip of its own (D-134).
- Status names EPA AirNow and Open-Meteo Air Quality, with their credits.

**Mutation verdicts** (targeted; 12 in watchpost, 5 in go-tuiMaps), all caught. Two survived at
first, both gaps in the fixtures:
- Anchorage, the monitor the test read, has no AirNow forecast, so the forecast steps' timing went
  unchecked. It is now checked over California.
- No area in the fixture had two issues disagreeing for one day. A case of its own now does.

## Batch 50 — the consolidated menu (W16, U2-39, D-141 to D-146, 2026-09-29)

**The design** is the HUM LEAD's own (U2-39). Six rulings filled what the mock left open:
- **D-141:** a Hazards group for Fire and Quakes; Tides and Rain & snow join Data Points.
- **D-142:** space on the chosen tint clears it.
- **D-143:** a disabled group hides its layers and keeps their ticks.
- **D-144:** the presets are Minimal, Standard and All. Standard and Full drew the same offered
  switches, so they are one.
- **D-145:** Fire gets a choice of All, Named or Hotspots, which closes U2-27. No overlay cap:
  D-129 keeps fire to three overlays at any zoom.
- **D-146:** the warning is said in the menu and stays under the map.

**As built.**
- **The rows:** radio, group, box, Fire, preset, detail switch, in the order ↑↓ moves. Boxes are
  laid two to a line.
- **The keys:** the menu owns ←→ while open; they change a row's choice and never pan the map.
- **The groups:** `layerOn` is a layer's tick gated by its group. Everything that draws, asks or
  costs reads `layerOn`; the menu's boxes read the tick.
- **Alert Areas** is the alert layer's own switch, as the mock implies.
- **Pluggability:** a registered layer no group names falls into Data Points, so a new layer
  still plugs in without edits (W1.13).
- **Saved state:** the Fire choice, the groups and Temperature's measure ride the saved layer
  choices, so no setting changed shape. A saved "standard" level reads as All.
- **The zoom hints** "(county zoom)" and "(state zoom)" no longer fit two to a line. They became
  one dimmed line under the detail switches, so a switch on but not yet seen still says why
  (U1-42).

**Found on the way.** The Settings window's Layers row switched a layer on beside another tint,
bypassing D-119's one tint since feels-like, and read the group-gated state. It now keeps one tint
and reads the tick.

**The Settings golden** changed by one word: Detail reads Standard where it read Weather (D-144).

**Mutation verdicts** (targeted, 12), all caught.

## Batch 51 — the menu's pickers are the app's (D-147, 2026-09-29)

**The HUM LEAD:** batch 50's choices were hand-typed "<- value ->". They looked like the app's
picker but did not act like it: the picker blinks the chip pressed, as every other modal does.

**Now:**
- Every choice row - Temperature's measure, the three groups, Fire, the preset - is Settings'
  `pickerCellW` over the shared `arrowChips`.
- The pressed chip blinks for `pickerFlashDur`, 350 ms, on that row alone. The tick after the
  window ends it and redraws the menu, as Settings' does.
- A blank line parts the detail switches from their zoom note, for scanning.

**The standing rule** (D-147, and memory): wherever possible, every part of the application is
built from the app's own control components, their behaviour as well as their look.

**Mutation verdicts** (targeted, 3), all caught.

## Batch 52 — every credit in About (U2-40, D-148, 2026-09-29)

**The finding.** Credits were said in two windows: About's "Data Provided by", the station's
sources; and since D-131, the Status window's MAP block, the map's. The HUM LEAD: one place,
About; Status about the sources' state.

**As built.**
- **About**, widened from 60 to 78, reads in this order:
  - "Data Provided by", the station's sources, from `credits()`;
  - "Maps", every source the map draws from, from `mapCredits()`: the basemap under the ODbL, the
    radar and HRRR, NDFD, each Open-Meteo product under CC BY 4.0, AirNow, fire, quakes, buoys and
    tides;
  - the relays' condition of use and the safety framing, from `aboutNotes()`, split out of
    `credits()` so they stay last;
  - the licence line.
  A long credit wraps under its own start.
- **Status's MAP block** keeps each host, what it is sent, and D-132's two notes about the data.
  It says no credit, and a test holds it to that.
- **The map keeps its short credits:** the badge row's chips, and the basemap's line on its frame,
  which the ODbL asks for where the map is shown.
- Open-Meteo's credit names what it now covers: temperature, feels-like, wind and UV.

**The dupes gate** found the Status window's two host lists - the radar's and the temperature's -
had become the same function once their credits were gone. `hostsFor` now takes each host's notes,
and the two are one.

**Mutation verdicts** (targeted, 4), all caught.

## Batch 53 — a warning that means it; MAP STATUS (U2-41, U2-42, D-149 to D-151, 2026-09-29)

**U2-41, the warning with no overlay on.** Measured: the estimate counted radar, the mode, at 37
requests a refresh against D-43's 40, and every alert's zones whatever categories were checked, so
it spoke with nothing chosen. D-149, "Chosen overlays, 3 MB / 25 req":
- The estimate counts the overlays switched on, and the alerts by their checked categories.
- Radar and the basemap are never counted.
- It warns past 3 MB or 25 requests. The defaults never warn; everything on at a national view
  does.

**U2-42, a Status section that read as credits.** D-150 made it MAP STATUS: the API STATUS table's
own component, the go-studs data table through `providerTable`, whose second column now takes its
heading. A row per map host:
- the layers it serves, joined where one host serves several;
- OK while its last answer is its latest word, FAIL while a failure is, IDLE (dimmed) before it is
  asked;
- when it last answered, and its tries, answers, cache and bytes.
The tiles held are on its header, and the notes about the data under it.

**What made it possible.**
- **The tiles:** the library fetches its own, so watchpost now hands it the library's own
  transport, rebuilt, wrapped to count them. Its private-address-refusing dialer (L-10.3), proxy,
  TLS floor and timeouts are unchanged.
- **The radar's and temperature's clients** were in no counter. They join the Status window's.
- **`httpx`** keeps each host's last answer and last failure. Its named host slots went from 8 to
  16: the station's client serves the map's hosts too, and eight would have folded some into
  "other".

**D-151, the disclosure.** The per-host text D-150 replaced was also FR-9.4's disclosure of what
the map sends, which D-75 put in this section. It is now one line under the table, in exact words
rather than the proposed "never your location": the tiles in view surround the place chosen.

**Mutation verdicts** (targeted, 7), all caught.

## Batch 54 — the endpoint tables are one shape (U2-43, 2026-09-29)

**The finding** (the HUM LEAD's screenshot):
- MAP STATUS's ENDPOINT did not fill, and its columns sat apart from API STATUS's above.
- The disclosure, wrapped at the terminal's width, stretched the Status window to 193 columns,
  where every content window's rule gives 120.

**Now.**
- The two tables share one shape (`endpointShape`): the columns' widths are taken over both
  tables' rows, and one width form is chosen that both fit.
- Both are filled to the same width, so ENDPOINT fills in each and every other column lines up
  down the window.
- The words under MAP STATUS - the data notes and the disclosure - wrap to the table's width and
  never widen the window.
- Both tests were run against batch 53's code first and failed there, the second at exactly the
  screenshot's 193.

**Mutation verdicts** (targeted, 2), all caught.

## Batch 55 — a place without a ZIP is its own place (#23, U2-44, 2026-09-29)

**The issue.** Lake Henshaw, CA looked up; ctrl+a (+ Watchlist) disabled. Places with a ZIP were
unaffected.

**The cause.** ctrl+a's check asked whether any watched place had the looked-up place's ZIP. The
watchlist held Guajome Park, which has none, so `"" == ""` made every place without a ZIP "already
watched". The code dates from 0.9.0; it showed once the first place without a ZIP was favourited,
not with 0.16.0. The same comparison was in two more lists:
- **RECENT** (`prependRef`): a second lookup without a ZIP dropped the first.
- **The launch's restore** (`restoreRecent`): with a watched place without a ZIP, every saved
  one was dropped at each start.

**Now.** `snapshot.PlaceID` is the one definition of "same place": the ZIP when there is one, else
the location key. `sameLocation`, ctrl+a's check, RECENT and the restore all read it.
`TestNoListComparesZips` fails on any ZIP compared or used as a key in `app` or `modes/tty`, unless
the same line proves it non-empty (`setup.go`'s, which does); it failed on the five old sites.

**Mutation verdicts** (targeted, 5), all caught.

## Batch 56 — the diagnostics in every build (#9, U2-45, D-152, D-153, 2026-09-29)

**The ruling.** ctrl+d's test alert ships in every build, behind its ARE YOU SURE (D-152). The
build tag was ruled before the confirmation and the test scripts existed; a station operator has
to test their alerts as a radio station does. NFR-2 now reads: nothing shipped may fabricate an
UNMARKED hazard.

**Now.**
- `app/inject.go` (was `inject_debug.go`) has no tag; `inject_release.go`, its empty stand-in, is
  gone, and so is its P10-08 ledger row (the mirror regenerated, 152 rows).
- The tagged tests run with every other. `test-tags`, `lint-injector` (and its self-test) and
  `build-diag` retired from the Makefile, CI and `required-gates.txt` (D-153); `vet-tags` keeps
  the `property` tag. F-101, `test-tags`' unexplained failure, closed by retirement.
- The ctrl+d window, the README's key tables and the code's comments say what ships.

**Found by the new guard.** `TestATestEventIsMarkedOnEverySurface` sends each scenario through the
window's own hook and one real cycle, and asks every consumer for the mark: the tape's items, the
band's, [w]'s rows, and the read's first and last lines. On its first run the Emergency scenario
failed: `laneItems`, which builds the tape's Emergency, Statement and Advisory items from [w]'s
rows, never copied the row's Test flag, so an injected evacuation order scrolled on the tape
unmarked. It is marked now.

**Mutation verdicts** (targeted, 6), all caught - one, the read speaking a real alert's line for a
test event, by the existing `test_event_test.go` rather than the new guard (an all-test card reads
its own script; that branch is a mixed card's).

## Batch 57 — W14 step 0: measure first (D-154, 2026-09-29)

**The checkpoint.** `checkpoint/pre-w14`, pushed on both repositories: watchpost `0e940ef2` (verify
green; CI green on macOS and Ubuntu, run 36588478972), go-tuimaps `1c0ca83` (`v0.2.0-rc.26`, its
gate green). Either can be restored exactly.

**The CI flake, measured before it was fixed.** CI's macOS `race` failed once on
`TestTheLoopSaysWhenItIsAhead` ("the loop is 0 frames"). The test helper `msgsOf` gave every
command 200 ms and dropped a late answer silently.
- Not reproduced locally in 80 runs, even with six cores busy.
- The only command that does not return is a `tea.Tick`; the radar's decode takes ~612 ms under
  `-race`.
- A window of 5 ms still passed: the decode dropped is redone by `settleMap`. A window no answer
  can meet reproduced CI's exact failure, 3 of 3: **a late radar answer, dropped**.
- Now a timer keeps the 200 ms; every other command runs to its end, and one past 30 s fails the
  test by name. It also ends a quieter race: ~20 map tests asserted while an abandoned decode was
  still writing the map. The honest wait costs `modes/tty`'s `race` +24 s (71.5 → 95.8 s),
  measured per test.

**The instrument** (`modes/tty/timing.go`, `app/timing.go`). Off unless `WATCHPOST_DEBUG_TIMING=1`;
`TestTheInstrumentChangesNothingItMeasures` holds the frame the same either way. Two definitions
corrected while it was built, each by a measurement:
- the library's `Complete` is the basemap whole; M5 (D-46) also needs every alert's area - hence
  `m5` apart from `complete`;
- a moved view is not in until its settle tick has asked for it (D-66): `settled` had fired 600 ms
  before a pan's own alerts were asked.

**The standard workload v1** - configs, driver, runner, `tools/perfsum` - recorded as the method in
`06_docs/perf-measurement.md`. **Found while building it:** the soak drivers waited for `º`
(U+00BA) where the app draws `°` (U+00B0); every earlier soak began on a silent timeout.

**First signal, n = 1, not yet a finding.** A cold open at first data: `default` reached M5 at
**19.0 s** against D-46's 3.5 s; `heavy` not in 45 s. In both the alerts feed is the late answer
(16-19 s). The baseline (n ≥ 5) comes next, then the audit.

**Mutation verdicts** (targeted, 7 on the instrument), all caught.

## Batch 58 — W14's verified low-risk fixes (D-155 step 1, 2026-09-29)

**The baseline and the audit** are `02-analysis/w14-findings.md` (commit `cb4a6cbb`). D-155 put
the verified low-risk fixes first.

**The instrument's two flaws**, both found reading the baseline and both RED first:
- `handleMapKey` (and the window's resize) drew the map, then marked the view moved, so `settled`
  was said before the move's settle tick had asked for its view. The view is now marked first -
  `viewGen` is read by the settle tick and the instrument alone, so nothing else moves.
- A trigger that outlived its ask (space, for the loop) timed every later refresh from the key
  press. An answer is now timed once a kind per ask, and an ask stops listening once settled.

**P10: 61 live findings to 31.**
- S-1: the `scrolls` closure in `windowKeysOf` is a method expression. The analyzer read the
  closure's body as a call made there, and that one edge closed a 32-function "recursion" through
  the layout code - 29 findings. Two auditors found it independently.
- S-3: the one real cycle, `layerOn` → `radarMode` → `layerOn`: the choice is read through
  `chosen`, the tick inside its group, which both use.
- `tools/perfsum`'s own loops are bounded, and its invariants mean something - phases in time
  order, cumulative CPU never falling in a phase - each with a test that fails without it. Its
  density (1.64) is left to the HUM LEAD as a dev-tooling exemption, as `tools/slope`'s was.

**C-2, the Tides estimate was always 0.** The estimate is built without fetching and only
fetching filled the stations. It now counts the stations the CO-OPS provider already holds in
view (`HeldTideStations`), asking nothing - one request each, as the feed makes. The test's view
holds some fixture stations and not others; the first form held them all and a mutant survived it.

**C-3, the Overlays menu's picker blink had no tick.** `tickNeeded` has its arm, as Settings'
picker has; a control asserts nothing else keeps a tick armed in the test.

**P-8, a zoneinfo read an alert sentence** on the description's path - through `platform/tz`'s memo
now, the rule B3 UAT 74 set.

**Found running the batch: a race in `modes/tty`'s tests.** Four helpers set `time.Local` and
restored it, while timers left sleeping by earlier tests (a `tea.Tick`'s command, which `msgsOf`
does not wait out) called `time.Now` on their own goroutines. `-race` reported it one run in three,
in `TestAVoiceNoteFromTheDeckIsDrawnUnderTheRowThatAskedForIt`. Every one wanted UTC: the package's
`TestMain` now pins it once, before any goroutine, and `TestNoTestSetsTheZoneMidRun` refuses a
mid-run write.

**Mutation verdicts** (targeted, 12): 10 caught at first; 2 survived and their tests were
strengthened until caught. The two structural changes are behaviour-identical by design; their
guard is the P10 checker, re-run: 61 → 31 live.

## Batch 59 — the feed, and the map kept at its floor (D-155 step 2a; D-156, D-157, D-159; 2026-09-29)

**D-156, the listener's lane.** The shared HTTP client has a third lane, interactive, for what the
listener has just asked for: the map's feed and the place lookup. Its own pacing and in-flight cap;
unlike the priority lane it keeps the per-host failure memo, proven through the client's own
request path. Found on the way: the lookup's resolve ran on the normal lane with a 5 s limit, so a
lookup in the launch's first seconds could spend its whole limit queued.

**D-157, one feed ask in flight.** New data while an ask runs marks the feed wanted again, and one
fresh ask follows its answer. A move or a change of place makes the ask stale: its settle tick
cancels it and asks for the new view, and a cancelled ask's answer (it carries an older number) is
never drawn. P-2: a move no longer asks the feed on its key - the settle tick asks, once (D-66).

**The first re-measure found a regression of this batch's own**, and it was fixed before anything
else: an answer was keyed on the view's box, which follows the map's size - and the size shrinks as
the notes and the description fill in, with no move at all. The answer was dropped and nothing
asked again: 2 of 5 cold `default` opens drew no alerts in 45 s. The view is now the listener's (a
move or a resize, `viewGen`); `TestAnAnswerIsDrawnWhenOnlyTheMapsSizeChanged` reproduced it first.

**D-159, the map kept at its floor.** Also found by re-measuring: at 149×38 with every overlay on,
the window's status, notes and radar timeline left the map 11 rows, under FR-1.4's floor, so the
notice showed in its place. A probe of the checkpoint's own binary showed it pre-existing - there
the map dropped out intermittently, even under `default` (117×7) - and 59a's reliable answers had
made it steady: by the HUM LEAD's rule, a regression. The window now grows, as far as the floor
needs and no further than the terminal's height minus 8, before the map yields; where even that
cannot hold it, the notes collapse to one line pointing to the Status window - which now carries
the map's notes (MAP STATUS), so the pointer is true. The map's width got its own function
(`mapWindowCols`): read through `modalWidth`'s switch, the new sizing closed a P10 cycle through
Help's width, and P10 had gone from 31 to 40 live; back to 31.

**The render check.** A draw under the floor tells the instrument (`below-floor`), and `perfsum`
names the phase and exits non-zero: a run that lost its map cannot pass (the HUM LEAD: "everything
that's supposed to render, ACTUALLY renders").

**Measured, cold opens at first data, n = 5 each** (`06_docs/perf/workload-v1/after-batch-59/`):

| | checkpoint | batch 59 |
|---|---|---|
| `default` M5 | 23.2 s | **7.6 s** (6.9-7.8) |
| `heavy` M5 | never, 0 of 5 | **10.1 s** (10.0-10.7), 5 of 5 |
| alerts answered, `default` · `heavy` | 20.4 s · 28-43 s | 5.3 s · 7.4 s |
| render check | not checked | PASS, both |

Still above D-46's 3.5 s: batch 59b takes the feed's inputs (serial, and fetched for layers off).

**Mutation verdicts** (targeted, 23): all caught, four only after their tests were strengthened -
the memo read through the real request path, an answer for a view left, the ceiling, and notes
shown in full where the window can grow.

## Batch 60 — what the feed asks for (D-155 step 2b; D-160; 2026-09-30)

**P-3.** Fire and Quakes were fetched on every feed ask whatever their switches, and dropped after.
The ask carries both rows, and each is fetched only while on - D-149's "nothing of it is fetched".

**P-4, D-160 ("unchecked = gone").** An unchecked alert category was not drawn, but its zones were
still resolved, and the window kept its notes and its "alerts in view", so the description could
name an alert of a category the listener had unchecked. The ask says what is switched OFF (an ask
that says nothing asks for everything), and the feed drops those alerts before their zones are
asked; with the layer off, it drops them all. The description's sentences already read only what
is drawn.

**P-6.** Fire, buoys, tides, quakes and AirNow were stamped `Valid` with the moment of each ask, so
the same data a minute later was a new overlay - handed to the library again, and prepared again,
on every answer. Stamped to a 10-minute step (`overlayStamp`), unchanged data compares the same;
currency moves at most a step early, well inside every `Keeps` (an hour at least). Ages - a buoy's
reading, a quake's NEW - still read the moment.

**P-7.** AirNow's national file (~1.9 MB) and a box's perimeters (up to ~378 KB) were parsed again
on every ask though served from the cache. Now through `platform/bodymemo`: AirNow keyed by the UTC
hour (its parse reads the moment only to the hour), perimeters by the box's URL; a bad perimeters
body is still forgotten - a test now holds it, which none did before.

**The instrument.** A warm reopen is its view's ask, so the view counts as still; a pan's settle
tick dropped while the map was closed had left warm opens never "still", and M5 unsaid. The debug
server takes a CPU profile (`/debug/pprof/profile`), so CPU-1's kind of burst can be caught.

**P10 held at 31.** Calling `bodymemo.New` inside a provider's own `New` reads, to the name-matching
call graph, as `New` calling itself; the constructor is referenced as a value instead.

**Measured, cold opens at first data, n = 5 each** (`06_docs/perf/workload-v1/after-batch-60/`):
`default` M5 7.3 s (7.1-9.7), `heavy` 9.6 s (9.4-9.9) - within batch 59's (7.6, 10.1): this batch
saves repeated asks and switched-off layers, which a cold open with the defaults has neither of.
Render check PASS both. The first answer's ~5.4 s is the serial inputs, the next batch's (P-5).

**Mutation verdicts** (targeted, 14): all caught, two after their tests were strengthened (the
switches read through `inputsFor`; a bad perimeters body).

## Batch 61 — the feed's inputs asked together (D-155 step 2c, P-5, 2026-09-30)

**P-5.** The view's alerts, the fire, the quakes, the buoys, the tide stations and AirNow do not
depend on one another, and the feed asked them one after another - six inputs of 300 ms took
2.45 s in the RED test. `fetchInputs` asks them together and joins them before the feed reads any:
six goroutines at most, one an input and never one a thing, each writing its own field; every
request still goes through its lane's pacing, so no host is asked faster. Clean under `-race`.

**Measured, cold, n = 5 each:** `default` M5 7.0 s (6.9-7.2), `heavy` 9.4 s (7.3-10.1) - a small
gain on batch 60 (7.3, 9.6). The alerts' answer still takes ~5.3 s: the other inputs were not what
held it. The chain left is the alerts in view, then their zones, then the overlays - measured stage
by stage next, rather than guessed at.

**Mutation verdicts** (targeted, 3): all caught - the feed not waiting for its inputs among them.

## Batch 62 — the feed timed stage by stage (2026-09-30)

**Why.** After batch 61 the alerts' answer still took ~5.3 s cold, and which stage held it was
being inferred. With the instrument on, a feed ask now keeps each input's time, the zones', the
overlays' and the whole's (`timingLog.stage`; one nil check a stage when off).

**Measured, first ask of a cold open, 3 runs each:**

| | inputs (together) | zones | overlays | whole |
|---|---|---|---|---|
| `default` | 0.26-0.59 s | **4.0-4.9 s** | ~0 | 4.6-5.2 s |
| `heavy` | 1.7-2.0 s (tides: up to 20 lookups at CO-OPS's 5 a second) | **3.0-4.1 s** | ~0 | 5.0-5.8 s |

**And relaunched with the cache kept** (the same HOME, `default`): zones 0.02 s - api.weather.gov
allows ~25 days and the disk tier keeps them - the alerts answered in 0.9-1.5 s, **M5 2.8-3.6 s**
against D-46's 3.5 s (23.2 s at the checkpoint). The first-ever zone fetch is left as it is
(D-161); the ~2 s from the answer to a complete frame is the library's (D-155 step 4).

**Found by this batch's gate: a race batch 61 left in two tests.** Asking the feed's inputs
together means a test's fake server is hit on several goroutines at once, and two tests' handlers
appended to a shared slice unlocked - `TestTheSeasStationsAreAskedOnlyWhileOn` (buoys and tides)
and `TestFireAndQuakesAreAskedOnlyWhileOn` (fire and quakes). Batch 61's gate passed by the luck of
timing; this one's `race` caught it, and it is the likeliest cause of T-1. The product's code was
not in either report. Both handlers lock; 30 runs of each under `-race` clean.

## Batch 63 — the map's memory held once (D-155 step 3; C-4, D-162; 2026-09-30)

**The trace** (heap profiles after GC, `pprof -base`): idle 36.7 MB; the map used and closed a
minute, 98.5 MB; closed ten minutes, 97.3 MB - bounded, no leak. Zone geometry was held three
ways - the raw bodies in the HTTP client's memory tier (capped), the zone store's parsed shapes,
the converted outlines - beside the warm-reopen outlines, radar frames (inside D-88's 24 MiB) and
grids. D-162: step 1 only - geometry held once, everything a warm reopen draws from kept.

**Now.** Closing the map window - really closing it: `mapShown` counts the map in D-106's stack,
so a window opened over it is not its closing - tells the app (`Config.MapClosed`), which lets the
zone store's memory go (`Forget`, the disk untouched). The next ask reads the zones back from the
disk tier (~0.02 s, batch 62's measure). `open` and `close` are thin wrappers now that compare
before and after; `close`'s fall-through calls `openWindow`, so a close tells the app once.

**Measured, the same probe:** closed a minute 84.5 MB (was 98.5), ten minutes 86.8 (was 97.3) -
the map's hold +50 MB where it was +62; the parsed zone shapes (`geo.walk`, 7.6 MB) gone from the
profile. Live alerts differ between the runs, so part of the difference is theirs.

**A test made to mean what it says.** Batch 60's D-160 test checked "no zones fetched when off" on
a fixture whose alerts carry their own polygons - it fetches no zones even when everything is on,
so that half proved nothing. It and this batch's test now use a fixture drawn from its zones, each
with a control that zones are asked when on; D-160's mutants were re-run and caught.

**Mutation verdicts** (targeted, 6): all caught.

## Batch 64 — the city scan parsed once (D-155 step 4, C-8; 2026-09-30)

**Measured first.** The map window's cost with nothing moving, a probe of 90 s each side (the
`default` workload, 149×38, ps cumulative CPU): the dashboard redraws about 10 frames a second
whether the map is open or not (a new `frame/frames100` instrument event, the time a hundred frames
take) - so the map pays no animation of its own - and the map open costs 2.2 % of a core against
0.5 % closed. The macOS sampler had named the city scan (`geodata.Index.Near`) on the UI goroutine:
the estimate (`refreshMapCost` → `mapCost` → `viewAlerts` → `viewAreas`) reads the state under 25
points of the view, one full scan of the 34k-row city table each, on every open, feed landing,
Settings open and layer switch - and every scan parsed each row's coordinates again.
`viewAreas` measured **31 ms, 1.02 M allocations and 3.9 MB a call**.

**Now.** The index parses the US cities' coordinates once, on first use (`placesCache`,
`usCities`), and every scan reads them: `viewAreas` **2.0 ms, 300 allocations, 6.6 KB** (15×).
The scan is still a full one, in the table's order, so every answer is the same one;
`StateExtents` walks the same parsed rows instead of a second copy of the filter and the parse.
Held, the coordinates cost about 0.8 MB. The map's area namer and the Producer's line-up read the
same scan and are faster by the same token.

**Tests.** `TestAScanDoesNotParseTheTable` pins a scan's allocations (40,940 before; 20 now,
pinned); `TestAScanAnswersAsTheTableDoes` answers a grid over the country and its waters at three
fences against the table parsed row by row - same cities, same order. `TestTheFrameRateIsMeasured`
for the instrument.

**Mutation verdicts** (targeted, 3): the cache rebuilt every scan, the US filter dropped, the
coordinates swapped - all caught.

## Batch 65 — the library's report kept while only the clock moves (D-155 step 4, P-11; go-tuiMaps rc.27; 2026-09-30)

**Measured first.** Watchpost asks the library for its report on every `renderMap` (the selected
place's answers, which the description reads) and passes its clock to every `Render`. The
library kept its last report keyed on that clock, so every map redraw worked it out again: in
the library's own benchmark, a redraw and a report with only the clock moved cost **9.8 ms and
11 MB**, against 0.02 ms for the redraw alone - all of it the radar loop's observed motion
(`describe.Track` over every frame), which the clock does not touch.

**Now** (go-tuiMaps L11.28, `v0.2.0-rc.27`): the report is kept by which overlays are stale -
the clock's one part in an answer - not by the clock: **0.02 ms and 5.6 KB**. Tested both ways
in the library: a later clock with nothing gone stale keeps the report; one overlay going stale
while another already is works it out again (a key on "anything stale" would miss it; the
mutant was caught). The library's idle `Render` is pinned too (63 allocations, 3.9 µs).

**What it changes at rest, measured:** nothing a probe can separate. A/B, batch 64 against this
build, the idle probe run alternately: map open 2.2 / 1.9 % against 1.6 / 1.8 %, closed 0.5 / 0.4
against 0.6 / 0.4 - within the runs' own spread (one earlier run of this build read 2.7 %). An
idle map redraws seldom; the saving is on each redraw - loop playback, Forecast steps, pans,
data landing. No regression either way.

**Batch 64's measure, corrected.** The title's per-frame area name was a quarter of the open
map's cost (2.2 → 1.8 %), not most of it as first read.

## Batch 66 — an alert's distance measured once; radar frames read from their pixels (D-155 step 4, P-12; go-tuiMaps rc.28; 2026-09-30)

**Measured first.** A new `warm` mode in the workload runner (`workload.sh warm`: one HOME kept,
run 1 fills the caches, runs 2 onward are a returning listener's) put a warm open's M5 at
1.18-1.27 s - batch 62 had 2.8-3.6 s - with a steady ~0.65 s from the alerts' answer to M5.
Batch 62's "~2 s, the library's" had mostly gone with batches 63-65. A scratch instrument (built,
read, removed) showed the rest: after the feed, the library had ~24 jobs; each Work took ~2 ms and
each landing's `renderMap` ~31 ms, **28 ms of it the library's `Report`** - worked out again at
every landing (its key holds the work landed) and every pan, and nearly all of that the nearest
edge of every alert's areas from the selected place, great circle by great circle.

**Now** (go-tuiMaps L11.29, `v0.2.0-rc.28`): a place's measure against an alert is kept per
overlay, alert and place while the overlays and the units stand; the view, the clock and the
work landing no longer pay it. **Warm opens, answer to M5: 0.01-0.25 s** (rc.27 back to back:
0.67-0.72 s); M5 0.45-1.26 s, the rest the network's answer. The same cost left every pan's
redraw: a candidate for R-1's occasional freeze, to be watched in the next session run.

**And P-12's decode half.** A radar PNG read through `image.Image` allocated for every pixel - a
600x275 frame 165,049 allocations, 2.7 ms. NRGBA and paletted pictures are now read from their
pixels: 50 allocations and 1.3 ms (paletted 288 and 0.97 ms), every other layout as before, each
tested against the general reading. The loop step's other cost - the basemap repainted on every
advance, 43 % of its 6.4 ms - is recorded, not changed: the loop plays only on space, at one frame
a second, about 0.3 % of a core, and the change would be to the renderer's core (P-12).

**Settled's spread is the temperature source's.** Settled equals `answered:temp` in every warm
run, 2.2-7.7 s across both builds; nothing in this batch touches it.

**Mutation verdicts** (library, targeted, 7): the overlay check, the units check, the place's
position in the key, the memo itself; the NRGBA path, its channels, the palette's index - all caught.

## Batch 67 — a Forecast step drawn once (D-155 step 4, P-9; 2026-09-30)

**Measured first.** With the library's report kept (batches 65, 66), a `renderMap` without it costs
~3.3 ms (batch 66's instrument: `Render` 1.1 ms, the title 0.2 ms, the rest). Forecast mode's
step and playback tick, and the mode's switch, drew it twice - `retime` sets the feed's overlays
again, which draws, and each caller drew once more: ~3 ms a second while Forecast mode plays.
Small; the change is also the simpler code.

**Now.** `retimeDrawn` is retime drawn once: the feed set again (which draws), or with no feed yet
the frame drawn alone - the case the second draw had covered. The step, the tick and the switch
use it. Radar's landing keeps `retime` as it was: it draws only once nothing is left to prepare,
by design, and that is not this batch's to change.

**Tests.** `TestAForecastStepIsDrawnOnce`: the switch, a step and a tick each call the library's
`Render` once, with the feed in and before any has landed (RED: twice with the feed in).

**Mutation verdicts** (targeted, 1): the no-feed draw dropped - caught (a step drew nothing).

## Batch 68 — the unwired station identification removed, its seam left (D-155 step 5, S-10, D-158; 2026-09-30; not the map's)

**D-158, as ruled.** `mastheadLine` and `coveragePhrase` (`app/transition.go`) - F-24's spoken
station identification on going ON AIR from standby, built and tested and called by nothing in
production - are removed with their test. **The seam:** `mastercontrol.GoOnAir`'s comment says
where a station ID is read and from what (`transition/masthead.txt`, rendered by `scriptText`,
before the power is declared), and the words stay in the script library, held by
`TestTheStationIDsWordsAreKept` so the seam cannot rot (a changed limitation sentence was caught).
A stray doc comment for a `spokenList` that no longer exists (`app/burst_words.go`), which named the
masthead as its second caller, went with it. Nothing a listener hears changes: it was never read.

**Found beside it (S-11), for the HUM LEAD.** F-27's spoken transitions have the same shape: the
line-up takes `ProgrammeReturn` ("Watchpost Radio now returns to its regularly scheduled
programming", from `transition/resume.txt`) and `Announcement` in its `Settings`, and the
production line-up (`app/schedule.go`) sets neither - so the Director arranges no transition and
the listener hears none. `programmeReturnLine` composes the words and nothing calls it. Queued as a
ruling: wire it, or remove it with a seam as D-158.

## Batch 69 — the map's grids built one way (D-155 step 5, S-5 app half; 2026-09-30)

**The duplication, located and read** (a read-only survey, then each copy read): the Radar-mode
hour loop 4 times (temperature and feels-like, wind, UV and air, waves), the Forecast-mode step
loop 6 times (temperature, feels-like, wind, UV and air, waves, rain), the lined grid 3 times
near-identical (UV and air, temperature, waves; wind's and rain's differ materially and stay
theirs), and the anchor's fallback 4 times.

**Now, three helpers and one:** `hourGrids` is the hour loop - the horizon, the id, the stamp, the
hour's span - taking a grid function (waves' copy gains the length guard the other three had);
`forecastDays` is Now's step and the days as far as both the steps and the sources reach, so the
six loops lose their `k+1 >= len(steps)` bookkeeping; `linedGrid` is interpolate, all-missing,
convert, lined grid, currency - `fieldGrid`, `tempGrid` and `waveGrid` are each a line over it;
`askAnchor` is the anchor. 99 lines in, 125 out; each copy's real difference (the high/low pairs,
the missing-day notes, the gusts, the rain's totals) stays at its call.

**A gap the extraction showed.** Two mutants of the shared code survived at first - the currency
counted from the grid's valid time instead of the anchor, and an unanchored ask left at the
minute - because neither rule was held by any test before either. `TestAGridIsCurrentThroughItsHourAndTheAnchorIsTheHour` holds both.

**Mutation verdicts** (targeted, 6): the horizon ignored, the hour's span, a day fewer, no unit
conversion, the currency, the anchor's hour - all caught.

## Batch 70 — the map window's commands and reconciles one way (D-155 step 5, S-5 tty half; 2026-09-30)

**Now.** `mapWorkers.cmd(stop, closed, f)` is the one shape of the map's Work, feed, radar and
temperature commands - admitted or refused once the map closes, `done` deferred, cancelled too when
`stop` ends (`context.Background` where nothing stops it). `reconcile` is the radar's loops' and the
temperature's grids' hand-in: the unchanged kept (U1-28), the changed set, a refused one keeping the
form drawn before it (U2-14) and said, the gone taken off. The feed's `setFeed` stays its own: it
drops a refused overlay rather than keeping the old, groups refusals by layer, and times each first.
57 lines in, 68 out. Lint caught one thing on the way: a nil `context.Context` passed for "no stop"
(SA1012) - `context.Background` now, tested by `Done() != nil`.

**Gaps the extraction showed.** Of six mutants of the shared code, four survived the map tests at
first: the radar's and temperature's "unchanged, not handed in again" was held only for the feed;
now `TestTheRadarAndTemperatureAreReconciled` holds it for both, with a control. Two stay, and are
equivalent: a closed map's Work command returning nothing rather than an empty `mapWorkedMsg`
(which draws nothing and asks nothing), and the radar landing's "a loop taken off draws at once" -
see C-10.

**Found (C-10): D-85's guard cannot fire.** The radar landing draws only "once nothing is left to
prepare" (`removed || !set || m.Pending() == 0`), so a loop handed in is not drawn half-ready. But
the library counts a job as pending only once a `Render` has planned it: straight after `Set`,
`Pending()` is 0, so the landing always draws. Existing behaviour, untouched here; queued to
investigate with the library (whether D-85's blink can recur, and whether the guard should ask the
library differently).

**Mutation verdicts** (targeted, 6): stop ignored, unchanged handed in again, refused dropping the
drawn form, nothing taken off - caught; the closed message and the removal's draw - equivalent.

## Batch 71 — a spent quota said, and NDFD drawing where Open-Meteo refused (W18.1, W18.2; D-165, D-166; 2026-09-30)

**Found in UAT.** In Radar mode temperature, feels-like, wind and UV drew nothing under their
badges; air quality's bands came after its reporting areas. Not a regression of W14's batches:
Open-Meteo answered **HTTP 429, "Daily API request limit exceeded. Please try again tomorrow."**
Its free tier is 600 calls a minute, 5,000 an hour, 10,000 a day, and each 80-point lattice counts
as many; W14's own measurement runs from this machine (about 20 instrumented launches with the map
open) spent most of it. Its marine API shares the quota (429 too); its air-quality API does not (it
answered - hence air's bands, not its areas, came late: they ride the temperature ask). Open-Meteo
publishes no reset time and sends no Retry-After. Rulings D-163 to D-169 recorded; the history
store's design (W18.3, D-169) written to `03-architecture-design/history-store.md` for the HUM LEAD.

**Now (W18.1).** The HTTP client keeps a failure's reason - a bounded, sanitised line of its body,
`StatusError.Reason`, never put in the error's words; `temperature.QuotaOf` names a spent quota by
its period and its reset (the period's end); a `QuotaGate` wraps Open-Meteo's client - a spent host
is refused at once without being asked, probed once an hour (an answer frees it), another host
never held for it. The answer carries `MapQuota`, and the map says it top centre on its own ground,
`MapNoticeQuotaBG` - dark orange (#C2410C) in the dark themes, a grey of its own in mono, and in
Watchpost Light the lightest orange its no-dark-ground rule allows (#FB923C, words black):
"! Daily Open-Meteo API Usage Exceeded. Resets 5:00 PM" - the reset in the listener's clock, with
the day when it is not today. The Forecast mode chip moves below it while both show.

**Now (W18.2).** A box Open-Meteo does not answer is asked of NDFD, wherever NDFD covers the region:
temperature, feels-like and wind draw, the chips say NDFD (both chips where only some boxes fell
back). NDFD has no hour before the current one, so in Radar mode its current hour lies under the
loop's earlier frames too (D-166's cold start), until the history store holds those hours. UV and
Forecast mode's rain have no fallback yet (W18.4, W18.5): the notice says why they are empty.
**Verified live**, Open-Meteo refusing: the answer said the daily quota (reset 00:00 UTC), drawn
from NDFD with its chip, three temperature, feels-like and wind grids - the current hour stretched
back an hour, and the two ahead.

**P10.** One new finding and it was false: the gate's `GetText` calling the wrapped getter's
`GetText` read as recursion by name; the wrapped call is a method value now. Held at 31.

**Mutation verdicts** (targeted, 10): the hold, the freeing answer, the 429 check, the daily reset,
the rescue, the stretch, the quota carried, the notice drawn, the reason's sanitising, the reason
read - all caught.

## Batch 72 — F-27's unwired transition removed, its seam left (S-11, D-163; 2026-09-30; not the map's)

**Verified unused first:** the line-up's `ProgrammeReturn` and `Announcement` are set by no
production code - the one `lineup.Settings` built outside tests (`app/schedule.go`) sets `Max`
and `Depth` alone - and `programmeReturnLine` was called by nothing but its test.

**Now, as D-158.** `app/transition.go` - `programmeReturnLine` alone after D-158 - is removed. Its
words stay in `transition/resume.txt`, held by `TestTheProgrammeReturnsWordsAreKept` (the one
clause between the modes, the ratified opening; a changed clause was caught). **The seam** is the
production line-up's construction in `app/schedule.go`: both fields left empty on purpose, and the
one line that would hand them in; the line-up's own transition machinery, tested, is unchanged, and
its two comments that said the app composes the words now say that none are handed in (D-163).
Nothing a listener hears changes.

## Batch 73 — map chips without brackets (D-174; 2026-09-30)

**The HUM LEAD's nit:** "badges in maps don't need the [ ] around them, just the bg color with
<space><space><label><space><space>" - "this also goes for the upper right chips as well". Four
places wrapped `chipFace` in brackets: the radar's badge and its hours-ahead badge (the tab's upper
right), Forecast mode's badge, and the badge rows, whose several chips were joined with `/`. Each is
the chip alone now; several chips sit side by side, their grounds dividing them (D-173's example put
them edge to edge), the padding dividing them where colour is off. Five tests held the old form and
hold the new (RED first); the history store's design draws its RECORDED chip the same way.

## Batch 74 — the history store's core (W18.3a; D-166, D-169 to D-180; 2026-09-30)

**The HUM LEAD's GO**, and four answers taken while the gate ran (D-175 to D-178), and two scope
directions while it was drafted: every API the app has or may have - an ionosonde's MUF the example
(D-179) - and a third UI mode later, an Analyst, reading what the store holds (D-180).

**Built: `platform/history`**, standard library, `platform/geo` and `platform/invariant` alone - it
knows nothing of weather. Datasets with their own step (a minute to a day), numeric fields over a
lattice or a point, and a bounded JSON document per record; bucket files compacted into a day's file
once the day is done, and rolled up into the year's past the hours' retention; claims for fetching a
bucket, compacting a day and rolling up a year; a manifest per dataset version, with Catalog, Series
and Extent for a reader that did not write it; bounded pruning. Not wired into the app yet: nothing
records or replays until W18.3b.

**Its first draft lost records, and the tests found it.** A day's hours in one file, each writer
reading, merging and replacing it: twelve concurrent writers kept three. A claim judged stale by the
file's real modification time against the store's clock; a year parsed from `2026.json.gz` with one
extension stripped. Rebuilt: a bucket per file, compaction and roll-up under exclusive claims, a
claim holding its own time. `TestADayIsCompactedWithoutLosingARecord` - two compactions at once and a
late bucket written during them - ten runs under the race detector, none lost.

**P10 at 31, met honestly.** The first cut raised it to 36: an unbounded loop (`trimExt`), three
recursions read by name (`os.Open` beside the store's own `Open`; `readDoc` now reads with
`os.ReadFile`), and invariant density 0.49 against 2.0. Density was met by structure, not padding: a
failure is one guard (`if bad { return s.note(tally, false) }`), each function's real preconditions
guarded (a nil store, an empty root, a bound not positive, shapes and lengths), trivial helpers
folded (four path functions into `filesAt`; `readDay` and `readYear` into one generic `readAs`, which
also cleared the one duplicate `dupes` reported).

**Mutation verdicts** (targeted, 14): buckets removed unheld, an older issue overwriting, a claim not
exclusive, a day expired without its roll-up, buckets unread, values unrounded, keys unchecked,
claims never stale, years never pruned, the catalog only of registered datasets, values over no
shape, a step unbounded below, a version mismatch read - all caught.

## Batch 75 — the history recorded (W18.3b, part 1; D-166, D-172, D-176; 2026-09-30)

**Now.** Whenever watchpost runs, in any mode (D-172), a recorder looks every five minutes - from a
random start in the first two, so instances seldom meet - for an hour to record: NDFD's current hour
(`NDFD.Hour`, Fetch's own second ask, so an hour the map fetched costs nothing) over every field box
of the station's region and of the region the map last drew. The dataset `ndfd-hourly` (v1:
temperature, feels-like, wind, gusts, wind direction) keeps 72 hours and rolls up into a month of
days for trends (D-176). One instance fetches each hour (the store's claim); an hour NDFD did not
answer is tried again once its claim goes stale; the store prunes once an hour.

**A box the map can draw is a box recorded.** The lower 48 draws its whole-region box at a wide
view and its eight grid boxes closer in; recording the whole region's alone would have left nothing
to replay at state zoom. `recordedBoxes` is all nine, each as `fieldBoxes` shapes it (the outer ones
grown to the region's edges), with `radar.GridBoxes` naming the grid.

**Not yet drawn.** Replay - a loop's past hours from the record, and the RECORDED chip - is part 2.

**Mutation verdicts** (targeted, 5): no claim, nothing put, the grid not recorded, a region
recorded twice, feels-like dropped - all caught.

## Batch 76 — the quota's notice where it belongs, and chips that say the truth (D-181 to D-183; 2026-09-30)

**The HUM LEAD's UAT nits, ruled in three steps.** The notice hid the north edge chip (a control);
it showed with no Open-Meteo overlay on; a layer with a fallback deserved its own words; the badges
should name what each overlay is drawn from now.

**Now.** The notice sits at the map's **lower right, on the row above the OpenFreeMap credit row**,
and gives way to every control: directly above the controls box while it shows, above an edge chip
it would meet; drawn beneath them all, and not at all on a map too short to hold it above them. The
Forecast mode chip is back at the top. It shows **only while an overlay drawing from Open-Meteo's
forecast or marine service is on** - temperature, feels-like, wind, UV, rain and snow in Forecast
mode, waves; never for Air quality, whose service has its own quota. Its words, the HUM LEAD's:

| The Open-Meteo overlays on | The notice |
|---|---|
| every one drawn from a fallback | `! OPEN-METEO: Quota Exceeded; Falling back to NDFD` |
| none drawn at all | `! OPEN-METEO: Quota Exceeded; Resets 5:00 PM` |
| some of each | `! OPEN-METEO: Quota Exceeded; Resets 5:00 PM // 2 Fall-backs active` |
| drawn from Open-Meteo still (its last answer, held) | nothing |

**The chips say what drew.** Waves name NDFD alone while Open-Meteo refuses (Open-Meteo alone where
NDFD does; both when both); UV's chip is the answer's, named only while it draws; temperature's,
feels-like's and wind's are dropped for a layer that drew nothing. The notice reads those same
chips, so the two cannot disagree.

**On the way.** Placing the notice around the controls and the edge chip drew their placements out
(`controlsPlace`, `edgeChipPlace`); `dupes` then found `withControls` and `withEdgeChip` one shape -
one `spliceAt` now. A measuring slip in the test (the splice's reset escape counted as cells) was
the test's, not the placement's.

**Mutation verdicts** (targeted, 9): the layers ignored, Open-Meteo's own chip counted a fallback,
the plural, the controls and the edge chip not given way to, the edge chip never drawn, the waves
always naming Open-Meteo, UV's chip missing, an empty layer keeping its chips - all caught.

## Batch 77 — a loop's past hours replayed from the history (W18.3b, part 2; D-166, D-173, D-178; 2026-09-30)

**Now.** Where Open-Meteo does not answer a box in Radar mode, NDFD draws its current hour and the
hours ahead (batch 71), and **the hours before the current one come from the history**: as many as
were recorded of the three Open-Meteo itself gives (`past_hours=3`), set into NDFD's series ahead of
its own hour (`withRecorded`), so they are drawn by the same path as a live hour - the same ids, each
during its own hour, handed in and kept as any other. **The stretch is now the cold start alone**:
the current hour lies under the loop's earlier frames only where nothing was recorded (D-166).

**Said by one chip.** A layer drawing anything replayed appends **`  RECORDED  `** after its source's
chip - one chip, one style, on its own ground (D-173, D-174): `MapChipRecordedBG`, the muted violet
D-178 chose - #5B4B8A in the dark themes, #DDD6FE in Light (the words black), a grey of its own in
mono - measured with every chip in every theme. The quota's notice does not count it a source:
"Falling back to NDFD", and with the history alone "Falling back to recorded data".

**A record of another shape is not drawn** - a box whose geometry changed since it was recorded
reads as unrecorded (tested).

**Mutation verdicts** (targeted, 6): no replay, the stretch over a replayed hour, no RECORDED chip,
any shape replayed, RECORDED counted a source, the chip without its ground - all caught.

## Batch 78 — the quota's hold shared by every instance (W18.3, design 4b; 2026-09-30)

**Why.** Open-Meteo's quota is the machine's, not a process's; batch 71's gate held a spent host in
its own memory, so each instance met the refusal for itself and probed on its own clock.

**Now.** The gate keeps its holds in `$XDG_STATE_HOME/watchpost/quota.json` (default
`~/.local/state/watchpost/quota.json`; `DefaultQuotaState`), written by temp file and rename. Every
gate takes the state as the machine's word, read at most every five seconds; a refusal met by one
instance holds them all; an answered probe frees them all - and `Refused`, which the notice reads,
reads the state too. **One probe for all:** an instance whose probe is due first reads the state -
a probe already moved forward is another's - then moves it forward under its own name and reads it
back, and asks only if its name stands. Where the state cannot be read or written, a gate holds by
its own memory, as before.

**Two tests the first draft of the test needed.** Mutants showed the shared test passing without the
sharing: after the other instance's answered probe, this one's own probe was also due, so it freed
itself; and a claimed probe was never tried by a second instance whose own reading said it was due.
`TestInstancesShareTheQuotasHold` now looks before its own probe; `TestAClaimedProbeIsNotAskedTwice`
has the second instance try during the first's probe.

**Mutation verdicts** (targeted, 5): the state never re-read, a refusal not shared, an answer not
shared, no check before claiming - caught; the claim's read-back never lost - equivalent in any
sequential test (it guards two writes landing between one read and one write), left to the design.

## Batch 79 — the Data tab's HISTORY group (W18; D-171, D-175, D-177; 2026-09-30)

**Where.** Settings already had a Data tab (`tabData`: DATA and ALERTS - EVENTS) - the HUM LEAD's
"[ Data ] Tab" is that one, so the history is a group on it, **HISTORY**, on every surface
(recorded in any mode, D-172).

**Now.** Built from the app's own parts (D-147):

- **Hourly detail** - a picker: 72 hours (default) / 7 days / 30 days / 1 year; **Trends** - 30 days
  (default) / 90 days / 1 year / 5 years (D-175). Written on close with every other setting
  (`applyIfChanged`, `history_hours`/`history_trends` in config.toml) and applied to the running
  store at once (`Store.Retain`, its manifest rewritten).
- **What it holds** - under the rows: "Holds 3.2 MB, in ~/.local/share/watchpost/weather/history"
  (`Store.Bytes`).
- **Clear history** - space opens the ARE YOU SURE on the red confirm tile, as ctrl+d's (its centring
  now one `confirmCentre` for both); while open it owns the keys; esc cancels; enter clears off the UI
  goroutine (`Store.Clear` - every entry of its root, nothing beside it) and the row says so (D-177).

**The goldens.** The Data tab's golden draws HISTORY under ALERTS - EVENTS in its right column. In
ASCII the arrows are `[left]`/`[right]`, and a picker as wide as the map's pushed the Data tab past
two columns; the history pickers are as narrow as their values (`historyValueW`), and ASCII's window
is 2 columns wider (99 to 101: every tab takes the widest's). The routing guard asked that Clear
history's answer be carried to the Settings window (`observerScoped`); the air boundary asked that
its three seams be classified (none reaches the audio); mCA1 was re-pointed to the router's line.

**Mutation verdicts** (targeted, 7, and mCA1 re-pointed and caught): no ARE YOU SURE, the question
not owning the keys, not written on close, the arrows reversed, trends ignored, not applied, Clear
doing nothing - all caught.

## Batch 80 — UV kept, and replayed when Open-Meteo refuses (W18.4, part 1; D-167; 2026-09-30)

**Now.** A second dataset, `openmeteo-uv` (v1: the UV index over each field box): whenever
Open-Meteo answers the UV, each hour it gave up to the current one is recorded - issued at its own
hour, so asking again rewrites nothing, and an hour ahead is never kept. Where Open-Meteo does not
answer, Radar mode draws the recorded UV for the current hour and the three before it, each in its
own hour; the UV badge says **RECORDED** alone (its source answered nothing), and the quota's notice
reads "Falling back to recorded data". The Data tab's retention is every dataset's alike (D-175).

**Still to come for UV (D-167):** the cold start - EPA's UV index for the cities in view as markers,
with its callout - a new source, its chip and credit, and a marker style the library may need.

**Mutation verdicts** (targeted, 5): hours ahead recorded, nothing recorded, nothing replayed, the
replay named O-METEO, UV keeping its own retention (caught once the Data tab's test held every
dataset) - all caught.

## Batch 81 — UV's cold start: EPA's index for the cities in view (W18.4, part 2; D-167; 2026-09-30)

**Now.** Where Open-Meteo gives no UV and nothing is recorded, the UV layer draws the U.S. EPA's
hourly UV forecast (Envirofacts, no key) for the largest cities in view - at most eight, from the
thousand largest US cities, ranked once - as markers in their UV band's colour, labelled with the
city and the value ("Vista 7"). Like the history's replay, each of the current hour and the three
before it is drawn during its own hour. The UV badge says **EPA**, on a forest-green ground of its
own, and a note under the map says the UV is EPA's forecast for the largest cities in view. Anything
recorded is drawn first: the cold start is the last tier before the notice alone.

**A new source, registered as every source is:** `domains/uv` (its hours read in each city's zone,
which the answer does not say); the credit in About's Maps list; `data.epa.gov` on the closed host
list; "EPA Envirofacts" in MAP STATUS; `MapChipEPABG` in all three themes (mono a grey of its own).
**go-tuiMaps rc.29** gives `UVRole`, the UV band a value falls in, for the markers (L11.30).

**EPA's answer, as it is:** its evening hours are dated the day before (a Sep/30 forecast runs
04 AM..04 PM Sep/30, then 05 PM..11 PM Sep/29). They are left unmatched, not guessed at - their index
is 0 or 1. What is refused is an index below 0 or past 50, or an hour a day or more from the first.
A strict ordering check was tried first and refused the recorded answer: the fixture caught it.

**P10:** `domains/uv` first read at density 1.25; `Host` became a constant and the answer gained the
two checks above - 31 live, unchanged.

**Mutation verdicts** (targeted, 15): the replay drawn first, the cap of eight, a city with no zone,
a city outside the view, the band, the badge, the hour read, the span's end (survived once - the test
took a zero end; tightened), the past hours; EPA's negative and too-high index, an hour a day before
and a day after, the hour's end, the zone the hours are read in - all caught.


## Batch 82 — rain and snow from the history and NDFD's totals (W18.5; D-168, D-184; 2026-09-30)

**Now.** Forecast mode's rain and snow no longer vanish when Open-Meteo refuses. While it answers,
each day it gave - its heaviest hour, its rain and its snowfall - is recorded (`openmeteo-rain-days`,
keyed at the day's local start, so a day in Guam is its own). Refused, a box draws its recorded days
as they were drawn, and for the days not recorded NDFD's daily totals: its six-hour amounts, the
rain liquid-equivalent (qpf) and the snowfall, summed on the local date each period starts - today
and three days on, NDFD's reach. NDFD is asked once a box, and only when a day is not recorded. Now
stays blank in a fallback: NDFD has amounts, not a rate.

**The totals' own scale (D-184):** go-tuiMaps **rc.30** (L11.31) adds a `qpf` preset in mm, the NWS
WPC's breaks at 0.01, 0.1, 0.25, 0.5, 1, 2 and 4 inches, nothing drawn under a trace (radar's rule,
shared as `FloorsFirst`), WPC's hues lightness-searched to pass the checker. Each day's total is
marked on it as Open-Meteo's are, snow apart ("*2.0in"). The colour row under the map reads
**NDFD TOTALS · NOT RADAR** and keys the totals' classes; wherever a model's rain draws too, it keys
radar's. The badge names what drew - O-METEO, NDFD, RECORDED - and a note says the totals are
NDFD's. The library's legend now writes a break with a second decimal where one would misstate it
(0.25, not 0.2).

**The fixture:** `ndfd-totals.xml`, a live NDFD answer over the Alaska Range on 2026-09-30, the
lower 48 holding no snow that day - rain and snow at five points of six, 3.19 inches of snow in one
period. The app's tests answer NDFD for whatever points they ask, as Open-Meteo's do.

**Mutation verdicts** (targeted, 13, plus the library's 6): inches to mm, snow summed as rain, a day
summed as its last period, NDFD asked every day, the history passed over, a day's record skipped, the
totals in radar's scale, snow unmarked, the NDFD chip, the RECORDED chip, a day's span, the colour
row's head - all caught. **Equivalent:** NDFD's days capped at four (its answer reaches no further).
The library's: `FloorsFirst` radar's alone, the class ink, the unit, the label's second decimal, the
legend's trace, the floors - all caught.

## Batch 83 — what the map costs Open-Meteo, measured and held to a budget (W18.6; D-185; 2026-09-30)

**Now.** `temperature.CallWeight` weighs an ask as Open-Meteo bills it - every location a call, its
variables over ten and its fortnights past one raising each (its maintainer, open-meteo issues #438
and #1295; the pricing page's examples). `TestTheMapsOpenMeteoWeightIsWithinItsBudget` runs the
map's own temperature pipeline - and Radar mode's model rain where HRRR is not - against a stand-in
that answers nothing and keeps every address, each distinct address weighed once, as the hour's cache
asks it. Every Open-Meteo row on:

| View | Radar mode | Forecast mode | Of which |
|---|---|---|---|
| The lower 48, whole | 265.2 | 343.2 | one box: forecast 109.2 (+78 rain days), marine 78, air 78 |
| California | 544.0 | 704.0 | two boxes |
| Alaska | 695.2 | 695.2 | two boxes; Radar mode's model rain ahead 158 |

A refresh is each hour the map is open and each view into boxes not yet asked. Eight hours on one
state is 4,400 to 5,600 of the 10,000 a day; a second instance on the IP doubles it. **The budgets are
these figures**: a change that spends more fails, and W19 lowers them as each layer moves to its
keyless source (D-185 to D-187).

**A correction.** D-185 was put with "~3,000 calls a refresh" - nine lower-48 boxes. A whole view of
the lower 48 is one field box; the figure was too high by about ten times there, about four for a
state. The ruling's ground stands (above), and its row now carries the measured figures.

**Mutation verdicts** (5): the per-location multiplier, the days factor, more points a box, more
variables an ask - caught. Equivalent: 81 points a box (the lattices are the same as 80's). One slip:
a mutant of `rain.go` was undone with `git checkout`, not by copy - it held no uncommitted change, and
nothing was lost.

## Batch 84 — temperature from NDFD first, both modes (W19.1; D-185, D-188 to D-190; 2026-09-30)

**Now.** Settings → Maps → Temperature governs both modes, **NDFD (NWS) its default** (D-190); the
file's word is "open-meteo" for Open-Meteo, anything else NDFD, so a file that held the old empty
default opens on NDFD. Choosing Open-Meteo says beneath the row that it is metered. D-96 and D-101
are overturned in this.

- **Radar mode on NDFD:** NDFD's current hour and hours ahead; the loop's past hours from the
  history - the recorder's NDFD hours - chips NDFD then RECORDED; nothing recorded, the current hour
  stretched under the loop (the cold start). Open-Meteo is not asked for temperature, feels-like or
  wind.
- **Now's feels-like (D-188):** NDFD answers it from the next hour, so the recorder now keeps NDFD's
  next-hour feels-like as that hour's record, merged in when the hour comes. Now draws the recorded
  hour's; on a cold start NDFD's next hour. The history's `Claim` now counts an hour recorded only by a
  record issued in its own time - a record kept ahead of it is not.
- **An empty Today (D-189):** in the evening NDFD has no Today high or low. The hours recorded since
  midnight fill it - their highest and lowest, feels-like and peak wind likewise - only where NDFD
  left it empty; else Open-Meteo for that day; else blank with the note it always had.

**The measure, rebased.** W18.6's stand-in now answers NDFD as NDFD answers - feels-like from the next
hour, an evening's days - on the clock the pipeline reads (it had answered on the test's date, and
every NDFD day fell outside the week). NDFD chosen, Open-Meteo's weight a refresh:

| View | Open-Meteo chosen | NDFD (default) |
|---|---|---|
| Lower 48, Radar mode, UV and air off | 187.2 | **78.0** (the waves' fill alone) |
| Lower 48, Forecast mode, UV and air off, the history warm | 265.2 | **156.0** (rain days, waves' fill) |
| California, Forecast mode, UV and air off, warm | 544.0 | **320.0** |
| Every row on | unchanged | unchanged - UV still asks the full forecast (W19.2) |

**The Settings focus order (HUM LEAD nit).** ↓ in the Maps tab went from Radar to Temperature, past
Radar ahead and Quakes: the rows were declared in another order than they are drawn. They are now
declared as drawn, and `TestDownWalksAGroupAsItIsDrawn` holds every group to it. The Maps golden
moves by one line: Temperature opens at NDFD (NWS).

**The memo guard** caught the metered note: it grows the Maps tab, and every tab takes the widest's
size, so `mapTempNDFD` reaches the Settings frame. Its one writer settles the window (bumps its
generation); the guard now models that writer for a top-level field as it did for `setup`.

**Mutation verdicts** (targeted, 15): the picker Forecast mode's alone, the history for NDFD's boxes,
a box Open-Meteo filled replayed, Today from the history, Now's recorded feels-like, the cold start's
next hour, the lowest and the highest, the history overwriting what NDFD gave (survived once -
nothing held Today's feels-like high to NDFD's; now held), the recorder's merge, the next-hour record,
`Claim`'s issued rule, the default, the metered note - all caught.

## Batch 85 — UV from EPA's cities first (W19.2; D-186, D-191; 2026-09-30)

**Now.** With UV on, the UV layer is EPA's forecast for the largest cities in view, as markers in
their bands' colours, labelled "City 7": in Radar mode the current hour and the three before, each in
its own hour; in Forecast mode Now's hour on Now and **the day's peak on Today** (EPA forecasts today
alone). The badge says EPA, a note says whose it is. Open-Meteo's UV grid is drawn beside them **only
over the boxes whose Open-Meteo forecast was asked for something else** - Open-Meteo chosen as the
temperature's source, or filling for NDFD (D-191) - so UV alone asks Open-Meteo nothing; the badge
then reads EPA, O-METEO. Recording and replay are as they were where Open-Meteo is asked and refuses.
`uvCold` is `uvCities`: no longer a cold start.

**The weight a refresh, every row on** (NDFD the default):

| View | Batch 83 | Now |
|---|---|---|
| Lower 48, Radar mode | 265.2 | **156.0** |
| California, Radar mode | 544.0 | **320.0** |
| Alaska, Radar mode | 695.2 | **474.0** |
| California, Forecast mode, the history warm | 704.0 | **480.0** |

What remains in Radar mode is air quality's model tint and the waves' fill (W19.4, W19.5); Forecast
mode on a cold history still asks Open-Meteo for Today (D-189's second tier), and UV's grid rides it.

**Mutation verdicts** (targeted, 9): Open-Meteo's grid where its forecast was not asked, EPA asked
with UV off, Today's peak, Now's hour, the peak's maximum, the EPA chip - caught; the boxes the filler
answered, the source's every box, and a refused box counted - survived the first run (the cost measure
cannot see a grid that rides a cached answer), now held by `TestUVsGridRidesWhereOpenMeteoAnswered`
through `uvAsked`, and caught. P10 read the filler's `Fetch` calling the wrapped `Fetch` as recursion,
as it read the quota gate's `GetText` (batch 78): the same fix, a method value.

## Batch 86 — Settings: a picker a layer, the pickers lined up (HUM LEAD's Settings nits, 2026-09-30)

**Now.** The HUM LEAD's Settings notes, taken together:

- **MAP - LAYERS is its own group**, a row a layer, each the "← Enabled / Disabled →" picker the
  detail rows already were: ↑↓ walk the layers and leave the group past either end (from above onto
  the first, from below onto the last); ←→ and space switch the layer under the cursor, through the
  same switch as before - one tint at a time (D-119, D-137, D-139). What the layers would cost is said
  under them. The old group is MAP - DETAIL. A build that registers no layer draws no group.
- **Smart columns.** MAP - LAYERS is drawn after MAP - DETAIL, so the planner balances the tab: MAP
  and MAP - DETAIL in the first column, MAP - LAYERS in the second, stacked where they do not fit -
  as every other tab is laid. The rows are declared in that order, so ↓ follows it.
- **A group's pickers line up.** A group's labels pad to its own longest: Data → History's Hourly
  detail and Trends were two cells apart. `TestAGroupsPickersLineUp` holds every group of every tab.

The Settings fixtures now register the app's layers, as the app always does: the Maps golden shows
MAP - LAYERS; the Data golden's History pickers line up. (The focus-order fix the same notes asked for
shipped in batch 84.)

**Mutation verdicts** (9): the last layer's edge, the first's (survived once), coming from below,
coming from above (survived once), ←→ switching, the layers' label width, History's, the empty group
hidden (survived once), → with one layer (survived once) - the survivors now held in
`TestALayersRowIsAPickerAndTheArrowsWalkThem`, and all caught.

## Batch 87 — rain from NDFD's totals first, Open-Meteo's coarser (W19.3; D-187, D-192; 2026-10-01)

**Now.** Forecast mode's rain and snow draw **NDFD's daily totals for today and three days on**, in
their own scale (D-184), whether or not Open-Meteo answers. Now - the hour's rate - and the days past
NDFD's reach are Open-Meteo's heaviest hour in radar's scale, each day's total marked, **asked on a
quarter of a box's points** by default (`temperature.LatticeOf`, ~18 to 20 a box against ~78);
Settings -> Maps -> **Rain Day 4+** offers the full density, and says beneath it that it asks about
four times the calls (D-23). Where Open-Meteo refuses, its days draw as recorded; where NDFD refuses,
Open-Meteo draws all seven. The badge names what drew; the note says the totals are NDFD's.

**The weight a refresh** (NDFD the default):

| View | Batch 85 | Now |
|---|---|---|
| Lower 48, Forecast mode | 343.2 | **283.2** |
| California, Forecast mode | 704.0 | **584.0** |
| Alaska, Forecast mode | 695.2 | **575.2** |
| Lower 48, Forecast mode, UV and air off, the history warm | 156.0 | **96.0** |
| California, Forecast mode, the history warm | 480.0 | **360.0** |

**Mutation verdicts** (10): Open-Meteo drawing NDFD's days, the full density, the coarse lattice, the
history drawing NDFD's days, NDFD's refusal drawing something (survived once - nothing tested NDFD
refusing while Open-Meteo answers; now tested), `LatticeOf` ignoring its points, the toggle, the
note, the ask - all caught.

## Batch 88 — air quality from AirNow's contours (W19.4; D-193; 2026-10-01)

**Now.** Air quality's tint is **AirNow's current-AQI contours** (`airnow/today/cur_aqi_combined.kml`,
keyless, about hourly, ~2 MB): each field box a grid in the AQI scale, each cell its contour's category
(`airquality.CategoryAQI`: the category's middle), none where no contour reaches - under AirNow's
monitors, through Radar mode's loop and on Forecast mode's Now; Today and Tomorrow keep AirNow's
reporting-area forecasts as markers. **Open-Meteo's air-quality API is not asked** (nothing else asks
it, so it is never "already asked"), and its path is gone: `temperature.AirQuality`, its host, credit
and fixture. The badge says AIRNOW; About credits AirNow for the monitors and contours; the layer's
cost estimate is AirNow's two files, whatever the view.

**The file.** The contours tile the country with holes - one national Good polygon of ~26,000
vertices, every other contour a hole in it, and contours nesting (Unhealthy inside
Unhealthy-for-Sensitive-Groups). A point is the highest category whose polygon holds it, holes and
all. A point test a cell took **100 ms** for the lower 48's box; `Contours.Raster`, a scanline that
fills between each row's crossings by the even-odd rule, takes **1.2 ms** - and agrees with the point
test at every cell of the fixture (`TestARasterIsItsCellsCategories`). The fixture is California's
twelve contours, the national polygon left out.

**The weight a refresh, every row on:**

| View | Batch 87 | Now |
|---|---|---|
| Lower 48, Radar mode | 156.0 | **78.0** (the waves' fill alone) |
| California, Radar mode | 320.0 | **160.0** |
| Alaska, Radar mode | 474.0 | **316.0** |
| Lower 48, Forecast mode | 283.2 | **205.2** |
| California, Forecast mode | 584.0 | **424.0** |
| California, Forecast mode, the history warm | 360.0 | **200.0** |

**Mutation verdicts** (12): holes, the highest category (survived once - the fixture's contours never
overlap; now an overlap test), the raster's highest (survived once, the same test), the scanline's
edge, an hour ahead (survived once; now tested), an unknown style, Now's step alone, a cell no
contour holds, the row off - all caught. P10's live 31 are the same 31 (checked against HEAD in a
worktree): the temperature package's density finding moved from the deleted `air.go` to `cost.go`.

## Batch 89 — the waves from NDFD first, Open-Meteo past its reach (W19.5; D-194, D-195; 2026-10-01)

**Now.** The waves follow temperature (D-194). The recorder keeps **NDFD's waves an hour a record**
(`ndfd-waves`): NDFD's start at the next hour, so its next hour is kept as that hour's, and when the
hour comes it is the current hour's. The map draws NDFD's waves, the **current hour and the loop's
three before it from the history**, and on a cold start NDFD's next hour stretched under the loop.
**Open-Meteo Marine is asked only for the points NDFD does not reach** (`OpenMeteo.WavesAt`, the
points named alone), and a point it answers nothing for - land - is remembered (`landPoints`, a box's
points, for the run) and never asked again. In Forecast mode a day NDFD gives nothing for anywhere -
past its six - is Open-Meteo's at every point but the learned land (D-195), and a note says so where
it gives nothing either. The badge names NDFD, O-METEO and RECORDED as they drew.

**The weight a refresh, its second** (the first learns the land, once a run - logged beside it):

| View, every row on | Batch 88 | Now |
|---|---|---|
| Lower 48, Radar mode | 78.0 | **6.0** |
| California, Radar mode | 160.0 | **32.0** |
| Lower 48, Forecast mode, the history warm, UV and air off | 96.0 | **24.0** |
| California, Forecast mode, the history warm | 200.0 | **72.0** |

The measure's stand-ins now answer as the sources do: NDFD's waves in a coastal band, nothing ashore
or past 127W; Open-Meteo Marine's past 127W, nothing ashore. Alaska's figures stay high in the measure
only because its stand-in puts all of Alaska's longitudes at sea.

**The measure, out of the race leg.** It ran 184 s under -race - sixty refreshes over stand-in answers
built and parsed each time - and it is one goroutine: `//go:build !race`, as the idle bench is, its
stand-ins apart in `mapcost_standin_test.go` for the history's tests, and NDFD's answers built once an
address. The app package's race run went from ~222 s to 126 s.

**Mutation verdicts** (12): the land skipped, NDFD's reach, the land learned, the history's hours,
the cold start's stretch, the empty day, its note, the recorder, the next hour, `WavesAt`'s scatter,
`SetHour` over the source's own (survived once - untested; now tested) - all caught.

## Batch 90 — two W14 defects: a cached garble, an absent assembler (C-5, C-6; 2026-10-01)

**Now.** W19 done, W14's structure and quality work resumes with its two real defects.

- **C-5:** NDBC's station list and buoy files, and USGS's every query, now **forget a body that does
  not parse**, so the next ask goes to the source rather than failing on the same garbage for the
  cache's lifetime - as HMS, WFIGS, FIRMS and NWS already did. Each tested against a server whose
  first answer is garbled.
- **C-6:** an empty RECENT list starts the recent pipeline with no assembler, and four readers -
  the radio's fire, quakes and sea, and the FIRMS status - dereferenced it. They now walk
  `livePipelines.assemblers()`, the assemblers that exist, favourites first: four copies of the walk
  made one, and none can reach a nil.

**R-2** waits for UAT (D-196, the docs commit before this batch).

**Mutation verdicts** (4): the station list's forget, the buoy file's, USGS's (the test fails without
it), the nil assembler listed - all caught.

## Batch 91 — code no production path reaches (W14 S-8; D-163, D-197; 2026-10-01)

**Now.** S-8's unreached code, each checked unreached first:

- **`castProblems`** - a stand-in for the cast's validation, called by tests alone; production
  validates in `setCast` and `softChanged`. Its two deadlock guards (validating under the deck's lock
  re-enters it and hangs) now drive **`setCast` itself**, the production path - proven by a mutant
  that validates under the lock and is caught.
- **`withForecast`** - a wrapper; production calls `fetchForecast` and `joinForecast` itself. Its two
  tests call them as production does.
- **`term.colorEnabled`** and **`ellipsize`** - nothing calls them; the fire row's test compared a
  name against `ellipsize`'s identity on names that never reach its width.
- **`bedfence.go` stays** (D-197): the bed fence's reach, measured wording and all, is owed to the
  Broadcaster's next release as **F-185**.

## Batch 92 — a Settings key press kept crisp (U2-48; 2026-10-01)

**Now.** The HUM LEAD found Tab in Settings laggy (U2-48), during a gate's run - "a good proxy for
older hardware". Measured: the harness's Tab press is ~2 ms, but each Settings frame read the
history's size **two or three times** (every tab is drawn to size the window), and each read walks
the store - a directory that grows every hour the recorder runs. **The size is now kept 30 seconds**,
and read again after Clear history. Two guards, counted not timed so a busy machine can neither fail
nor hide them: the size not read every frame, and **a Tab press into every tab** - the app's layers
registered, MAP - LAYERS among them - held to 3,700 allocations (3,631 measured, Maps the costliest).

**U2-47 investigated, not reproduced in the harness**: the loop at rest and stepped, with HRRR's hours
ahead joined and the clock past the newest frame, draws radar every time. A live run is next.

**Mutation verdicts** (2): the cache, Clear's re-read (survived once - the test's cached value was
small too; now a large store first) - both caught.

## Batch 93 — the radar's refresh blink, and a frame recorder (U2-47, C-10; D-198, D-199; go-tuiMaps rc.31; 2026-10-01)

**The live runs (D-198).** Four launches of today's build under a scratch home, a frame recorder on -
under the cap the HUM LEAD set, no 429. **U2-47 did not reproduce**: the radar was drawn at rest within
~2 s every time. Found instead, read from the recorded frames:

- **A one-frame radar blink** whenever a refreshed loop lands or the view zooms - 0 radar cells, then
  all of them 60 ms later. Traced into go-tuiMaps: a loop handed in again dropped its old pictures at
  once, and nothing of it drew until its job decoded the new version. **Fixed in rc.31 (L11.32)**: the
  old loop's frame at the moment stands in until the new pictures land - the picture form of L11.5's
  shape stand-in - held by `TestARefreshedLoopKeepsDrawing` through the public Map. This is C-10's
  symptom; D-85's guard no longer needs to fire.
- **Place names changing in a still view** while the loop plays (the label budget under an overlay
  re-placing names frame to frame) - a candidate for U2-46, next.
- **A slow terminal stalls the app**: bubbletea writes each frame holding its renderer lock, so a
  terminal slow to read blocks the event loop; keys queue and replay in a burst. Seen when the driver
  stopped reading; a candidate for U2-46's lag-then-jump under load, after the labels.

**The frame recorder.** `WATCHPOST_DEBUG_MAPFRAMES=<file>` appends every drawn map frame as a JSON
line - the loop's place, the view, the cells in the radar's colours, the place names, the library's
status. Without the switch the dashboard gets no hook and a frame pays one nil check; its seam is
classified for the air (none).

**Mutation verdicts** (9 library, 3 here): the stand-in taken, its lookups (two), kept across two
refreshes (survived once), dropped with the loop (survived once), dropped once the new land (survived
once) - the survivors now held - and the public test fails without the fix; the recorder's cells, its
hook, its call - all caught.

## Batch 94 — the names stand still as the loop plays (U2-46; D-200; go-tuiMaps rc.32; 2026-10-01)

**The cause.** The HUM LEAD saw the flicker with the loop playing (D-200). Read in the library: an alert
drawn by its time (D-98, L-15.1) was skipped on the frames outside its span, and its word - placed before
any place name - gave its room back on those frames, so the names around it came and went. Two more
movers found in the same read: an outline's severity digits (placed before names, D-65), and the name
budget - a radar frame on a gap, or a field outside the moment, counted the map bare and placed more names.

**The fix, in go-tuiMaps rc.32 (L11.33, L-28).** An overlay outside the moment is reserved: never drawn,
but an alert's word and digits hold their room from the names, after every drawn alert's, and a field or
image - or a loop on a gap - keeps the budget the one under it. Held by the public
`TestNamesHoldStillAsALoopPlays` (Kansas and South Dakota came and went without it) and
`TestAFieldOutsideTheMomentHoldsTheNames`, and five renderer tests. Accepted and written into L-28: two
alerts' words that collide may still trade places frame to frame.

**Mutation verdicts** (17, library): 16 caught; the survivor a redundant check (a reserved label is
always an alert's, already skipped there) - removed. Here: the bump alone; the wiring of an alert's span
(D-98) is unchanged.

## Batch 95 — the tab shown, each layer's notes, EPA's New York (U2-49, U2-51, U2-52; D-201 to D-203; 2026-10-01)

**UAT notes, four** (U2-49 to U2-52), traced by three read-only investigations in parallel, then ruled one at a
time: D-201 (NDFD's gaps: weight the valid corners, a denser NDFD lattice), D-202 (UV cities spread over
the view, the count a Setting), D-203 (air: AirNow's forecasts stay markers, a clear note). This batch is
what needed no ruling and D-203; D-202 and D-201 are their own batches.

- **U2-49.** The tab shown is drawn as a focused row is: the list's pointer and the focus yellow,
  brackets included (`setupTabRow`, through `render.ListLabel` and `ListMark` - the app's own, D-147).
- **The notes nobody saw (U2-52).** The map's notes were shown only while Temperature or Feels like was
  on - and Air, UV, Rain and Waves each turn those off, so their notes never showed. `MapTemperature`
  gains `LayerNotes` by layer; `tempNotes` says each while its layer is on, in one order. UV's, rain's
  and waves' notes moved there; Air says D-203's note - Radar mode "the current hour, drawn through the
  loop", Forecast mode "the current hour; Today and Tomorrow are its forecasts by area, as markers; none
  is published past Tomorrow".
- **U2-51's bug.** EPA answers "NEW YORK CITY" with an error, so the largest city in view drew nothing;
  asked as EPA names it, "New York" (`epaName`).

**Mutation verdicts** (7): the tab's label and pointer, a layer's note gated by its layer, Air among the
noted layers, the memo key seeing the notes, the note by mode, EPA's name - all caught.

**Goldens updated** (five Settings goldens, `-update-golden`): the tab row alone, its words unchanged -
the tab shown now carries the focus colour (U2-49). **The first gate failed on U2-48's pin**
(`TestSetupAllocBudget`, memo miss +2 at 133x44, +13 at 80x24: the tab tinted in four pieces, both
forms built every frame). Not raised: the tab is tinted in one piece and the narrow form built only
when the wide does not fit - now 4881 and 3459, under the pin (4934, 3485) with room. **P10** went to 32 with a one-line `epaName`
(`epa.go`'s density); the alias moved inside `Hourly`, back to 31.

## Batch 96 — UV's cities spread over the view, the count a Setting (U2-51; D-202; 2026-10-01)

**The spread.** `spreadInView` replaces `largestInView`: the view is cut into about n cells by its shape
on the ground (kilometres, not degrees), each cell's largest ranked city taken, then each cell's next,
round by round, until the count - cells over sea or beyond the border hold none, so the land's take a
second. No two closer than 100 km, or than half a cell's side where cells are smaller (a state's view
keeps its count: Southern California's 24 about 23 km apart). The ranking's order kept, so markers draw largest
first. The lower 48 at 24: New York, Los Angeles, Chicago, Houston, Phoenix, Philadelphia, San Antonio,
Dallas, Jacksonville, San Jose, Columbus, Indianapolis, Seattle, Denver, Oklahoma City, El Paso, Boston,
Portland, Detroit, Omaha, Cleveland, New Orleans, Tri-Cities and Billings. (GeoNames' "New South
Memphis" appears at 48; EPA may not know the name, and a city EPA refuses is left out silently, D-124.)

**Asked four at a time** (`uvAskers`): 24 asks one by one would hold the map's UV for seconds; the
answers are kept in the cities' order whatever order they land in.

**The Setting.** Settings -> Maps -> UV cities, 8 / 24 / 48 (default 24), file word `map_uv_cities`;
`MapAsk.UVCities`; `tty.UVCitiesByCount` the one rule for a count not offered, read by the app too. Its
cost - "about N keyless EPA asks an hour for a view; more cities show UV a little later" - is said under
the row while it is focused (D-23): said always, it cost Settings two lines and its alloc pin.

**U2-48's pin re-pinned, 80x24-miss 3_319 -> 3_491 (+5%)**: the window is measured on every tab for its
width, so a row on the Maps tab costs every Settings frame - after the tab row was built once a frame
(batch 95) and the note kept to its focus. 133x44 stays under its pin. The Maps golden gains the row.

**Mutation verdicts** (15): the spacing, a city a cell a round, the spacing's shrink (survived once -
California's cells were near 100 km; Southern California added), a zone needed, four at a time, the
order kept, the count forwarded, the ranking's order out (survived once - a second round's larger city
before a first round's smaller; a case added), the picker's direction, the ask, the default, the save,
the note on focus - all caught.

## Batch 97 — NDFD's temperature to the coasts and borders (U2-50; D-201; 2026-10-01)

**The cause** (traced read-only, then built): one ~5.5-degree lattice of 78 points for the lower 48;
NDFD answers xsi:nil for points over the sea, Mexico or Canada, and a cell whose *nearest* point was nil
was blank (D-101's rule, against U2-17's square patches) - so land beside a nil point went blank: the
Florida panhandle beside a Gulf point, Big Bend beside Coahuila, Boston, Cape Cod and Vermont beside the
Atlantic and Ontario.

**The weighting.** `Lattice.InterpolateWide` / `InterpolateWindWide`: a cell is blank only where all
four of its points are; otherwise weighed from those that answered. Temperature, feels like, wind and
the gusts only (`tempGrid`, `windGrid`); rain, UV and the waves keep the nearest rule, as ruled.

**NDFD's own lattice.** `NDFDLatticeFor` = `LatticeOf(.., 4 x MaxPoints)`, about twice as dense each way
(the lower 48 27x11, 297 points, was 13x6, 78); the drawn field split half as fine (`Lattice.fine`), so it is about as
many cells as before, not four times. NDFD answers only the first hundred points of a request, so
`pointAsks` cuts a lattice into asks of a hundred - the same for `Fetch` and `Hour`, so the recorder's
hour asks the very addresses the map's did and shares the cache; each answer is read into the one
series by its points' coordinates. A box: 8 asks a refresh (4 x the days and the hour), keyless, cached
to the hour; `tempLayerCost` now counts them (it said a request a box - a quarter of what NDFD fetched).
Open-Meteo keeps its 80 points (D-185): asked on its own lattice as source, fallback and filler; the
Open-Meteo budget measure is unchanged in every case.

**Where the two meet.** `fillDays` (D-189's "else Open-Meteo for that day") copied Open-Meteo's day
arrays whole into NDFD's series: on two lattices the lengths disagreed and the day would have drawn blank
- now `Lattice.Resample` puts them on NDFD's points (bilinear, the wide rule), the peak wind's direction
by `ResampleNearest` (350 and 10 degrees never blended to 180). The recorder records NDFD's hours on
its lattice - the shape the map's replay matches - and the waves on the box's own, where the map merges
them point by point with Open-Meteo Marine's. **On upgrade**: records of the old shape are passed over
by every replay (checked by shape, never misread), so the first hours after are a cold start, and the
current hour's old claim holds until the next hour; the history refills within its hourly retention.

**Mutation verdicts** (17): the wide rule, the half-fine split, the wind's, the resample's weighting
(survived once - a case with an empty point added), its copy, the nearest index, a hundred an ask
(survived once - the test read the constant it checked; now the literal), the hour every ask, the denser
lattice, NDFD on its lattice, Open-Meteo's fallback on its own (survived once - asserted now), the fill
resampled, the temperature's and wind's grids wide, the cost, the recorder's two lattices - all caught.
Not caught by a test of the wiring: the peak direction's nearest resample in `fillDays` (the fakes'
directions are uniform); `ResampleNearest` itself is held by its own test.

## Batch 98 — every frame of the loop drawn, the temperature sooner (U2-54 to U2-58; 2026-10-01)

**The report.** UAT of batch 97: temperature, feels like and wind missing on the loop's observed frames,
the wind on some frames and not others, the temperature slow - "sometimes work, sometimes draw" - and
suspected of the performance batches.

**How it was traced.** The frame recorder (D-198) learned to count each layer's cells, to tell what the
feed delivered by layer, and - with `WATCHPOST_DEBUG_MAPFRAMES_TEXT` - to keep each frame's text and the
feed's spans, chips and notes. Live runs under a scratch home with a copy of the HUM LEAD's history
(read-only), the loop playing: every frame before the current hour drew no contour and no wind; every
frame from it drew both; the temperature arrived 47 s after the map opened. Then offline, on the real
pipeline over the measure's stand-ins, the spans.

**The causes - none in the drawing, none in the performance batches.**
1. Radar mode's temperature has been NDFD's since W19 (D-190), and NDFD has no hour before the current
   one: the loop's past is the history's (D-166). With little recorded, the current hour was stretched
   **one** hour back under a loop two hours long; and a recorded hour with the next missing left frames
   of nothing between them.
2. Batch 97 made NDFD's lattice denser, and every replay passed over the hours recorded before it as
   another shape - so on the day of the upgrade nothing past was drawn.
3. NDFD's current hour can come back with no wind: a gap at the hour itself.
4. Batch 97's eight NDFD asks a box went one after another.

**The fixes.** `onLattice`: a record of the box's old lattice is put on the new one's points (bilinear;
the wind's direction by its nearest point), in every NDFD replay - the past hours, Now's feels-like
(D-188), Today from the hours (D-189), the recorder's own merge. `fillPast` replaces `stretchNow`: any
hour with no grid of its own - before the current one, or the current one itself - draws the next newer
grid, back to `pastHours` before the hour; for temperature, feels like, wind and the waves. NDFD's asks
go four at a time (`ndfdAskers`), read in the order asked; a lattice of one ask pair as before.
**Verified live**: the past hours replayed from the old-lattice history, every one of 36 frames drawn,
the temperature in 10 s cold (2.5 s warm), the wind's current hour drawn from the next.

**Tests reworded, not weakened**: four asserted the one-hour stretch or the refusal of another shape;
they now assert what was meant - every observed frame drawn, each recorded hour drawn as itself,
another *box* refused.

**Mutation verdicts** (13): the old lattice resampled, another box refused, the direction nearest, Now's
feels-like and Today from the old lattice, how far back, the gap filled, feels and wind filled, the
waves filled (survived once - the fake gave past hours; a cold-start case added), NDFD in parallel -
caught. Equivalent: ignoring an ask's error (the empty answer then fails to parse, the refusal still
returned). The recorder's text kept only when asked: tested.

**Comments describe the code now (AP-HIST-01, HUM LEAD 2026-10-01).** Every comment written in batches
93-98 and go-tuiMaps L11.32-L11.33 read against the rule: nineteen here and four in the library narrated
a defect or a batch ("was blank", "took half a minute", "were never seen", "batch 97 made ...") and are
rewritten in the present tense. `tools/authoring`'s AP-HIST-01 table now catches those shapes, its
self-test holding a specimen of each and a present-tense sentence of the same facts that must pass (the
self-test fails with a phrase removed - tried). Older narrative comments the table cannot decide are F-186.

**Still open**: U2-53 (radar slow) and the HRRR frames arriving late (U2-57's scrubber, "after 2
loops") - R-2, bisected next against `checkpoint/pre-w14`.

## Batch 99 — the observed loop first, HRRR's hours ahead joining (U2-53, U2-57; D-204; 2026-10-01)

**Measured first (R-2, as D-196 waited for).** Five builds - `checkpoint/pre-w14`, batches 63, 70, 84 and
98 - each launched live under a fresh home, radar only, the map opened on the lower 48. The loop's frames
(hours ahead included) were known in 0.4-2.2 s on every build. Radar *pixels* - timed by replaying each
run's terminal stream through a terminal emulator and counting map cells in the radar legend's own
colours - came in 1.9-3.6 s idle (the checkpoint 3.2-3.3 s, today 1.9-3.6 s); with temperature and wind
on, 1.1-2.6 s; under every core spun busy, five rounds each, a median 3.2 s at the checkpoint and 3.4 s
today (one run of today's at 9.0 s). No regression.

**Read in the code.** The observed loop was handed back only once every HRRR forecast frame had
downloaded - a slow HRRR held the radar the listener could already have - and a failed HRRR was asked
again only at the next 2-minute refresh: U2-57's scrubber with no hours ahead, the band "after 2 loops".

**D-204, built.** HRRR's hours ahead are fetched by `aheadFetch`, apart from any one ask: an ask starts
it and returns the observed loop at once, with `MapRadar.AheadIn` - when to ask again (3 s while HRRR is
on its way; what is left of 30 s after a failure, the failure told to the diagnostics once; zero when
joined). The map pane schedules that ask (`mapRadarAgainMsg`), for the region it was owed for and only
while the radar is on. A region or horizon left cancels its fetch. Live: the radar at 1.9 s, the loop
grown from 24 frames to 36 at 7.9 s.

**Mutation verdicts** (7): joined when landed, the 30 s retry, the failure told once, one fetch at a
time, the answer saying when, the tick scheduled, the region guard - all caught. The new message is
carried back to Observer's map window by the router (`observerScoped`) - held by
`TestEveryWindowReplyIsRoutedBackToTheWindow`, which refused it until it was.

**A measuring note.** The terminal-text timing read "FRAME n / N" from the stream; a renderer that
redraws only changed cells sends "36", not the label, when the count grows - so a count that grows
after the first draw is read from the frame recorder, not the stream.

