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
