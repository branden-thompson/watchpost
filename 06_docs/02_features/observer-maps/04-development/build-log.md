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
