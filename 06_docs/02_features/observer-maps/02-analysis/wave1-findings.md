---
title: "0.18.0 Observer maps — DISCOVER wave 1 findings"
date: 2026-09-22
phase: DISCOVER (RCC)
sev: SEV-0
status: "COMPLETE — W1-A radar, W1-B basemap, W1-C watchpost integration; synthesis at the end"
---

# Wave 1 — what was measured, not assumed

Three surveys, dispatched with non-overlapping scope. Every number below came from a live request or
from running the library's own code on 2026-09-22. **The measuring programs are not committed**: Go
source in this tree is subject to every gate, and these were throwaway probes. Each result names the
request or the library call that produced it, so it can be re-derived; if PLAN adopts a probe as a
standing instrument, it enters the tree through BUILD with its tests.

## W1-A — Radar loops (27 live requests, ~14:21 UTC)

| | IEM (`nexrad-n0q`) | NOAA/NCEP MRMS (`conus_bref_qcd`) |
|---|---|---|
| Past frames | 12 relative layers `-m05m`…`-m55m`, **and** WMS-T (`n0q-t.cgi`, layer `nexrad-n0q-wmst`) back to 2011 | WMS `TIME` dimension, 60 listed times |
| Cadence | 5 min, on the grid | ~120 s mean, irregular (95–145 s) |
| Retention | 1 h relative; years by absolute time | ~2 h |
| Newest frame age at check | ~1.5 min | ~2 min |
| Caching | `Cache-Control: max-age=300` | `expires` = date + 2 min |
| State 298×152 frame | 17.6–19.5 KB | 7.7 KB |
| 12-frame cold loop, state | ~220 KB, 12 requests | ~95 KB, 12 requests |
| Colour table | **Published, exact, 256 entries** | **None published** — styles carry no `ColorMapEntry`; only a 500×30 picture legend (−20…70 dBZ) |
| Terms | Public domain, credit appreciated; no stated limit | US government; `Fees none`; no stated limit |

**Traps — each is a requirement, not a note:**

1. **Both sources fail with HTTP 200 and an empty transparent image** — IEM for an off-grid time
   (`13:51`), MRMS for a time outside retention. A frame must be checked against the source's
   capabilities time list; a 200 is not success.
2. **IEM's timed service defaults to 2011-02-21** when `TIME` is omitted
   (`default="2011-02-21T19:30:00Z"` in its capabilities). An unset time draws radar fifteen years old
   with no error.
3. **Relative layers roll every 5 minutes.** A loop fetched across a rollover duplicates or skips a
   frame, and relative frames cannot be cached across refreshes. Absolute times are stable keys: after
   the first load a refresh is one request, not twelve.
4. `-m30m` and the same absolute time agreed on only 84% of pixels (cause UNVERIFIED — possibly archive
   reprocessing). Mixing the two forms in one loop is unsafe.

**Coverage differs by source:** in one scene IEM painted 25% of a state box, MRMS 5% — MRMS's quality
control appears to remove weak echo (inference). The two loops are not interchangeable pictures of the
same thing, which argues for one overlay per source rather than a blend.

## W1-B — Basemap scale, the view bound, tiles (library code run; 15 live requests)

**Zoom by scale**, from the library's own `FitTo`, margin 0 (`internal/project/fit.go:86`):

| View | 69×12 | 149×38 |
|---|---|---|
| Contiguous US (height-limited fit) | 1.09 — shows 91° of longitude | 2.75 — 62° |
| Contiguous US, fitted by width | 1.74 | 2.86 |
| Region, 15°×10° | 2.34 | 4.01 |
| State, 5°×4° | 3.71 | 5.38 |
| County, 0.7°×0.7° | 6.23 | 7.89 |
| Unplaced map (whole world) | −1.81 — 681° | −0.15 |

Tiles are drawn at `floor(zoom)` (`mercator.go:110-114`), so **below zoom 4.0 the embedded tiles
suffice offline**: at 69×12 that is the US, a region and a state; at 149×38 only the US.

**Every path that widens the view without the host asking** (re-run and reproduced):

| Path | Evidence | Observed |
|---|---|---|
| Map made with `WithSize`, not yet placed | `map.go:214-215, 259-260`; `placed` set only at `view.go:216` | whole world, 681° |
| Placed, then resized larger | `map.go:262-263` keeps zoom, so ground grows with the box | US at 69×12, rendered at 149×38: **197°** |
| View invalid at new size | `map.go:264-265` → `WholeWorld` | not reached in test (10001×1 left the view unchanged); trigger UNVERIFIED |
| `FitWorld`, `ZoomBy`, `PanCells`, `FitTo` | host calls; floors at `MinZoom = −8` | only when called |

**A host clamp is sufficient in v0.1.0** if applied before every `Render` and wrapped around every view
call, with the view placed before the first `Render`: pre-clamped to 2.75, a 149×38 render stayed at
62°. **The counter-argument is strong and is the reason HR-3 stays:** a SEV-0 invariant resting on
caller discipline across five call paths fails the first time one path is missed, and the library
itself advertises −8 as valid. A floor and bounding box enforced inside the library make the rule
structural and testable.

**Tiles and caches.** One frame needs 1–2 tiles at 69×12 and 2–4 at 149×38. OpenFreeMap tiles are
64–221 KB on the wire, 99–395 KB decoded; a state view at 149×38 needs ~909 KB of memory cache against
a **500 KB default cap** (`internal/tiles/cache.go:17`) — the cap is reported, not enforced, and panning
back refetches. `NewShared(bytes)` raises it (~4 MB suggested). Cold start: one TileJSON request (19 KB,
~260 ms) plus ~50–100 ms per tile. The 256 MB disk cache (LRU, no expiry) holds ~700–2,500 tiles; a
session moving between a few places needs well under 50.

**OpenFreeMap terms** (fetched 2026-09-22): no limits, no keys; **automated collection without
permission is forbidden** (no prefetching or cache warming); the service "may discontinue it at any
time without notice"; attribution "OpenFreeMap © OpenMapTiles Data from OpenStreetMap" (already drawn
by the library, `facts.go:174`). Pass `https://tiles.openfreemap.org/planet` (TileJSON; resolves to a
dated tile path, zoom 0–14). `SourceCredit()` returns HTML — never print it raw.

**Offline, rendered:** before the first settle, `NoTiles` and a notice. Embedded only: a state at
69×12 is `Complete`; a county is `Sharpening` forever (the zoom-3 ancestor stretched 8×), and over
central Kansas it was **entirely blank**. A source with no network: the same stand-in plus a
`tile-failed` warning, retried later.

**Side finding (library):** `Fetcher = fetch.Func` takes an internal `fetch.Request`, so a host cannot
write a Fetcher without reflection. A contract gap for v0.2.0.

## W1-C — Watchpost integration (read-only on `e54df7d`; 23 live NWS requests, ~14:33 UTC)

**How often alerts only partly resolve, measured.** `/alerts/active`: 371 alerts, **333 (90%) with no
geometry** — most of them marine (Small Craft Advisory 201, Gale Warning 42). Zones per no-geometry
alert: median 1, mean 2.2, largest **41** (a WV/OH Flood Watch); 515 distinct zones nationally.
23 of 23 zone fetches, six at a time as the store does (`zones.go:37`), returned 200 in 0.16–0.36 s;
marine zones run to 640 KB. **In fair weather `Area.Missing` is almost always empty.** It fills when
the network fails or is slow (30 s timeout), under an outbreak burst (UNVERIFIED — not provoked), for
a zone over the vertex caps (rare: largest measured 15,194 of 50,000), and **for any national-scope
resolve: 515 distinct zones today exceed `maxAtOnce = 512` (`zones.go:67`)**. Resolving only the
watched locations' alerts stays far under it. A cold 41-zone watch costs ~2 s.

**The update path.** Snapshots are sent with `p.Send` from the publisher's timer goroutine
(`app/pipelines.go:178`), routed by `Router.Update` → `scopedMessage` → `Dashboard.dispatch`
(`dashboard.go:975`). **`View` is a value receiver** (`view.go:19`); memo slots are pointers for that
reason (`memo.go:9-10`). The one owner for library calls is the Bubble Tea loop: a new case in
`dispatch` beside `SnapshotMsg`. **The spike mutated the map on three goroutines** — `Show` on the
publisher's timer goroutine, two `Work` goroutines, and `Render(size)` from inside `View` — and its
`MapChangedMsg` did nothing (`63903d7 dashboard.go:977`). Its memo key read `Map.Changed()`, which is
not a Dashboard field, so **the F-30 guard (`memo_completeness_test.go:39`) could not see it** — the
spike's own comment says so.

**Selection has no event.** `nav-up`/`nav-down` change `d.selected` and nothing else
(`nav.go:38-47`); a resolved lookup sets it in `handleResolved` (`modal_location.go:163-169`).
**`selectedLocation()` returns nil** before the first snapshot and when the index is out of range
(`nav.go:170-185`) — both with the Observer on screen.

**Config and caches.** Config at `$XDG_CONFIG_HOME/watchpost/config.toml`. Caches at
`userCacheSubdir(name)` → `<OS cache dir>/watchpost/<name>` (`app/dashboard.go:611-618`). **F-49 is
ruled** (FR-7.2, 2026-09-08): caches stay in the OS cache directory. The HTTP cache already caps at
256 MB (`httpx/cache.go:106`); the library's own disk cap is another 256 MB. **There is no
`--no-map` and no offline flag** — the spike's comment names one that never existed.

**Instruments a map window must satisfy:** F-30 memo completeness; the margin test
(`memo_completeness_test.go:399`); reachability at 80×24 (`modal_reachability_test.go:107`); the
alloc-budget pins (`bench_test.go:167`, `tick_alloc_test.go:27`); goldens with width invariants
beside byte pins (`severe_test.go:327-338`); scripted PTY (`make pty-severe`, `make journey`); the soak
slope; `TestAScheduleLeavesNoGoroutineBehind` (hand-rolled, no goleak); and
**`TestASCIIFramesCarryNothingButASCII` (`golden_test.go:173`) — braille is not ASCII.**

## Cross-cutting synthesis

1. **Composition — the bound needs the owner.** W1-B shows the view widens on resize and before
   placement; W1-C shows the only safe place to call the library is the Bubble Tea loop. Together:
   the view clamp and every mutator live in one place, `dispatch`, driven by `WindowSizeMsg`,
   selection and snapshots. A clamp in any other goroutine is the spike's defect again.
2. **Composition — radar traps and the owner.** W1-A's silent-empty frames mean fetching is a
   domain concern (validate against capabilities, key by absolute time), while W1-C's owner rule means
   only finished, validated frames cross into the loop. The fetcher never touches the map.
3. **Convergence — the spike's wiring is the release's main hazard.** W1-C (three mutating
   goroutines, a memo key no guard sees) and C-1 (a free function the reachability gate cannot see)
   and handoff §4.1 (four defences wired to nothing) all say the same thing: **the failure mode of
   this codebase is correct parts, wrongly connected, with a guard that cannot see the connection.**
   Every new wire needs a guard that can.
4. **Convergence — HR-3 is real.** W1-B's 197° resize and the library's advertised −8 minimum make a
   host-only clamp a caller-discipline invariant at SEV-0. Watchpost clamps in v0.1.0; the library
   bound in v0.2.0 makes it structural.
5. **Contradiction — `--ascii` against braille.** *(RULED: D-20, then D-26/D-34 — `--ascii` gets the text description, never braille.)* R-7.1 says `--ascii` produces a readable map; the
   ASCII golden test forbids every braille cell. Either the map has a non-braille form under
   `--ascii`, or `--ascii` shows a notice in its place. **A HUM LEAD ruling** — see open questions.
6. **Contradiction, resolved — "four in five".** The field comment says four alerts in five lack a
   polygon (C-9); today it was nine in ten, inflated by marine advisories. Both say the same thing:
   zone resolution is the ordinary path. C-9 stands; the number is weather-dependent.
7. **Risk update — M1 (where is it).** Partly mitigated: `Area.Missing` is rare in fair weather, so
   most alerts draw whole. Still active for outbreaks, which is when M1 matters most.
8. **Risk update — cache footprint.** New: map tiles (256 MB) and radar frames on top of the
   existing 256 MB HTTP cache, in the same directory tree. Needs one stated total.
9. **Risk update — memory.** New: a state view at 149×38 needs ~909 KB of tile cache against a
   500 KB default; the host must size the shared cache (`NewShared`).
10. **Open questions answered.** HR-3: stays (4). Tile source address: `…/planet` (W1-B). Radar
    loops fetchable from both sources: yes (W1-A). Where caches live: `userCacheSubdir("map"|"radar")`
    (W1-C, F-49).
11. **Open questions raised for the HUM LEAD — all three since RULED:** `--ascii` → D-20 and D-26; a
    map off switch → D-21 (a Settings row, not a flag) and D-22; the 512-zone cap → D-23 and FR-4.5
    (a scope that would exceed it is reported, never silently dropped).
12. **Implications for PLAN:** render-in-`Update` versus render-in-`View` under a lock is a genuine
    design choice with a performance cost either way (W1-C's counter-argument) — PLAN presents both;
    selection needs a message; the map needs a drawn state for no selection; a goroutine-join test for
    the pump; an alloc pin for the map frame; goldens with width invariants at 80, 120 and 133.
