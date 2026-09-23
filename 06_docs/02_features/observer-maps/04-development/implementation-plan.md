---
title: "0.18.0 — Observer maps — IMPLEMENTATION PLAN"
date: 2026-09-23
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "DRAFT — for internal review, then the PLAN red team, then HUM LEAD approval. No code: signatures, shapes, test descriptions, file paths and order only (D-13)."
---

# Implementation plan

**Goal.** Build 0.18.0's macro phase 1 (drawing and access) as `01-objectives/requirements.md`
states it, test-first, in the order go-tuiMaps' `radar-loops/03-architecture-design/integration-map.md`
fixes.

**Architecture.**
- The map is drawn in `Update` by one owner goroutine, and `View` only prints (D-41).
- Four guards keep an old frame from ever showing (D-41).
- A partial area is drawn as found, labelled, and its missing zones are named (D-42).
- The numbers are D-43's.
- Watchpost's structure holds: `domains/` are data sources, `platform/` is shared leaves, `modes/` are
  surfaces, and `app/` is the only composition root, checked by `lint-imports`.

**Branch.** `feature/map-drawing` → squash-merged into a `release/0.18.0` branch at SHIP.

**Every task follows the same order.**
1. Write the named test, and watch it fail for the named reason.
2. Make the change.
3. Run the lane the commit needs: `make verify` for any code.
4. Mutation check: revert the change, watch the test fail, then restore.

A task isn't done until its row in `requirements.md` names the test as its instrument.

## Where new code lives

| Package | Holds | Why there |
|---|---|---|
| `domains/radar` (new) | IEM and MRMS sources, one per file, registered and discovered (FR-9.3); frame validity (FR-5.3); fetch with a body cap (FR-5.7) | A data source is a domain |
| `domains/alerts` (existing) | Zone shapes, and their 7-day refresh (FR-4.6) | Zones are alert data |
| `platform/geo` (existing) | The region set and the bound check (FR-2.1, FR-2.5) | Shared and pure |
| `modes/tty/mapwin` (new) | The window, the map pane, messages, key handling, the description's layout | A surface |
| `app/maps.go` (new) | Wiring: the library, the sources, `livePipelines` methods, Settings | The composition root |

## Work packages

| WP | What | Requirements | Needs | Lands on |
|---|---|---|---|---|
| W1 | Window, size floor, degradation order, Settings rows | FR-1, FR-9.1, FR-9.4 | — | go-tuiMaps v0.1.0 |
| W2 | Draw in `Update`; the four stale guards; worker join | D-41, FR-8.2–FR-8.5, FR-3.3 | W1 | v0.1.0 |
| W3 | Basemap: source, caches, credits, offline, closed source list | FR-3 | W2 | v0.1.0 |
| W4 | The bound, held by the host | FR-2 | W2 | v0.1.0 |
| W5 | Alert areas, scope, partial areas, zone refresh | FR-4, D-42 | W2 | v0.1.0 |
| W6 | Fire and quakes | FR-6 | W5 | v0.1.0 |
| W7 | Without colour; the description (via `Describe`) | FR-7 | W5 | v0.1.0 |
| W8 | Radar loops and the motion Setting | FR-5 | W2, **go-tuiMaps v0.2.0 tag** | v0.2.0 |
| W9 | Move to v0.2.0: library bound, `Report`, fetch options, purge and retention | FR-2.4, FR-3.9, FR-3.10, FR-7.4, HR-6, HR-8 | W4, W7, **v0.2.0 tag** | v0.2.0 |

W1–W7 are **P1-a**: they need nothing from v0.2.0 and start at once. On their own they are the
ship-without-radar fallback (RK-4). W8–W9 are **P1-b**, and wait for the tag (FR-5.6).

---

## W1 — The window (FR-1, FR-9)

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W1.1 | `g` opens and closes the map window, visible in Help, overridable in `[keys]` (FR-1.1) | `modes/tty/mapwin/window.go`, keymap | A keymap action `map.toggle` | Help golden lists it; a `[keys]` override rebinds it |
| W1.2 | Centred on the selection, and following it (FR-1.2); selection gains a message (W1-C) | `modes/tty/nav.go`, `mapwin` | `type selectionMsg struct{ index int }` | Scripted PTY: move the selection, and the map's title and centre follow |
| W1.3 | No selection draws a stated state (FR-1.3) | `mapwin` | — | `selectedLocation()` nil → the stated text, never blank |
| W1.4 | The size floor, one owner for the number, the notice carrying the description; the degradation order and the legibility tier (FR-1.4, FR-1.9) | `mapwin/size.go` | `const minBody = Size{69, 12}` in one place | Size sweep 20×10 → 200×80: every frame is a map ≥ 69×12, or the notice with words, never overflow |
| W1.5 | The window joins every closed-set window test (FR-1.5) | `modes/tty/*_test.go` | — | The closed-set tests fail until it is registered |
| W1.6 | Maps off in Settings: `g` says so; nothing built or fetched (FR-1.6) | `mapwin`, `app/maps.go` | — | Setting off → zero map network calls |
| W1.7 | `--ascii` shows the description, never braille; the braille notice (FR-1.7, FR-1.8) | `mapwin` | — | `TestASCIIFramesCarryNothingButASCII` extended; chrome golden |
| W1.8 | The Settings rows: maps, scale, layers, alert scope, radar source, motion, description mode (FR-9.1) | `platform/config`, `modes/tty` settings | One row each, persisted, reachable without restart | Settings round trip per row |
| W1.9 | The in-app exposure notice: the rectangle goes to a named third party (FR-9.4) | `mapwin` | — | Golden of the notice on first open |
| W1.10 | A plugin registry for layers, sources and options (FR-9.3) | `app/maps.go`, `domains/radar` | Members register themselves and are discovered | An architectural test: add a fake source without editing the others |

## W2 — Drawing in `Update`, and never an old frame (D-41, FR-8)

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W2.1 | The map pane and its key, as Dashboard fields (FR-8.4) | `modes/tty/dashboard.go`, `mapwin/pane.go` | `type mapPane struct{ m *tuimaps.Map; lines []string; key mapKey }`; `mapKey{changed, ticks uint64; size tuimaps.Size; selection int; look lookKey}` | **The memo-completeness guard is extended to `mapKey` first**: a key field that isn't a Dashboard field fails |
| W2.2 | Library calls only in `dispatch`; `Work` in a `tea.Cmd` (C-7, FR-3.3) | `modes/tty/dashboard.go`, `mapwin/work.go` | `mapWorkedMsg{changed, ticks}`; `mapTickMsg{at}` scheduled with `tea.Tick` at `NextCall` | A race test: no library call off the Bubble Tea goroutine except `Work` |
| W2.3 | **Guard 2:** every input the frame depends on causes a redraw | `mapwin/pane_test.go` | — | A table: data, frame advance, size, selection, playback, depth, theme, units, each map Setting → a redraw |
| W2.4 | **Guard 3:** the printed lines always equal a fresh `Render` | `mapwin/property_test.go` | — | Random message sequences, 10,000 runs |
| W2.5 | **Guard 4:** the stored frame's counters equal the library's before the lines are stored (needs go-tuiMaps WP-L3, L3.2) | `mapwin/pane.go` | On v0.1.0, `Changed()` alone; on v0.2.0, `Frame.Changed` and `Frame.Ticks` | A frame drawn before a `Set` and stored after is refused |
| W2.6 | Every map worker joined on close (FR-8.2); an allocation pin on the frame (FR-8.3); goldens carry width invariants (FR-8.5) | `mapwin` | — | A leak test; an alloc test; the golden helper checks widths at 80, 120 and 133 |

## W3 — The basemap (FR-3)

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W3.1 | The OpenFreeMap source, with the embedded tiles always passed (FR-3.1) | `app/maps.go` | — | Offline: a state view at 69×12 is `Complete` |
| W3.2 | No map request before the listener asks (FR-3.2) | `app/maps.go` | The map is built on first `g` | A network counter reads zero until `g` |
| W3.3 | Offline and failing states said in the window (FR-3.4) | `mapwin` | — | Golden per state |
| W3.4 | Caches under the OS cache dir; the memory cache sized for the largest view (FR-3.5, FR-3.6) | `app/maps.go` | `userCacheSubdir("map")`; `NewShared(4 MiB)` | Config test; a 149×38 state view doesn't refetch on pan-back |
| W3.5 | Credits at every width (FR-3.7) | `mapwin` | — | Golden at 69, 100 and 149 columns |
| W3.6 | The closed source list (FR-3.8) | `domains/radar`, `app/maps.go` | — | A test enumerates every source and refuses an unlisted one |

## W4 — The bound, held by the host (FR-2)

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W4.1 | The region set and the check (FR-2.1, FR-2.5) | `platform/geo/regions.go` | `func RegionOf(loc geo.Point) (Region, bool)` | Every US region and territory the APIs cover; an out-of-region point is a stated state |
| W4.2 | The view placed before the first frame (FR-2.2) | `mapwin/pane.go` | — | The first frame's span is within the bound |
| W4.3 | The default scale as a Setting; the bound is not (FR-2.3) | `platform/config` | — | Settings round trip |
| W4.4 | The host clamp at every view change until HR-3 (FR-2.4) | `mapwin/bound.go` | Wraps every view call and every `Render` | **M2 instrument:** every rendered frame in the suite and the PTY journeys checked against the region |

## W5 — Alert areas (FR-4, D-42)

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W5.1 | `resolveAlertAreas` reachable from production through a `livePipelines` method (FR-4.1, C-1) | `app/pipelines.go` | — | `TestEveryLivePipelinesMethodIsReachedFromProductionCode` extended |
| W5.2 | One feature per area; severity to role; times and sender carried (FR-4.2) | `app/maps.go` | — | Round trip against the library's reading of rings |
| W5.3 | Scope: the station's locations plus national severe events in view; a Setting (FR-4.3) | `app/maps.go` | — | A count of drawn alerts per scope |
| W5.4 | **Partial areas, drawn as found, labelled, the missing zones named, and whether the place is in one** (FR-4.4, D-42) | `app/maps.go`, `mapwin/notes.go` | The label gets "(N of M zones)"; a note line under the map | **M4 instrument** over `Area.Missing` fixtures, including specimen 31's case: Fort Davis in the missing zone |
| W5.5 | The 512-zone cap reported, not dropped (FR-4.5) | `domains/alerts` | — | A test at 513 zones |
| W5.6 | Zone shapes fetched again on next use once 7 days old (FR-4.6) | `domains/alerts/zones.go` | — | Fixed clock: 6 days → no fetch; 8 days → fetch |

## W6 — Fire and quakes (FR-6)

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W6.1 | Hotspots, incidents and quakes as static point layers (FR-6.1) | `app/maps.go` | One layer each, registered (W1.10) | Golden with each layer on and off |

## W7 — Without colour, and the description (FR-7)

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W7.1 | Readable under `NO_COLOR`, Monochrome and Light (FR-7.1); colours from tokens (FR-7.2, FR-7.6) | `mapwin`, the contrast register | — | The no-literal guard; goldens per theme |
| W7.2 | **The description ships** (FR-7.4): alerts shown, and where the selected place is relative to each — on v0.1.0 through `Describe`, never saying "you" (D-29) | `mapwin/describe.go` | Words built from the library's data | **M1b instrument:** the recorded scenarios scored from the description alone |
| W7.3 | Coordinates never spoken; the description held to speech rules (FR-7.3) | `mapwin/describe.go` | — | The say-safe guard |
| W7.4 | Every palette passes the library's checker (FR-7.5) | `app/maps.go` | — | Checker test per theme |

## W8 — Radar (FR-5) · needs the go-tuiMaps v0.2.0 tag

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W8.1 | IEM and MRMS as registered sources; the source is a Setting (FR-5.1) | `domains/radar/iem.go`, `mrms.go` | `type Source interface{ Name() string; Times(ctx) ([]time.Time, error); Frame(ctx, t, box) ([]byte, error) }` | Registry test; Settings round trip |
| W8.2 | A frame is valid only if its time is advertised and its image isn't empty (FR-5.3) | `domains/radar/valid.go` | — | Recorded responses: off-grid, expired, empty, the 2011 default |
| W8.3 | Body capped as it is read (FR-5.7) | `platform/httpx` | A per-request cap | A 2 MiB body is cut at 1 MiB |
| W8.4 | One overlay per source, absolute times (FR-5.2) → `Image.Frames` (go-tuiMaps D-54) | `app/maps.go` | — | A loop across a 5-minute rollover has no duplicate or skipped frame |
| W8.5 | A refresh fetches only frames not held (FR-5.5) | `domains/radar` | — | A request count over two refreshes |
| W8.6 | The newest frame's age always visible; stale marked (FR-5.4) | `mapwin` | Uses `LoopState.Newest` and the library's frame time | **M3 instrument** |
| W8.7 | The motion Setting drives `SetPlayback`; the rates (FR-5.8, FR-5.9) | `mapwin`, `platform/config` | One call per loop | Setting → `LoopState.Playback`; reduce motion wins |
| W8.8 | `go.mod` requires the tagged v0.2.0, with no local replace (FR-5.6) | `go.mod` | — | A test reads `go.mod` |
| W8.9 | Recorded responses committed as fixtures before anything depends on them (FR-8.6) | `testdata/` | — | The M1 scenario set and FR-5.3's responses present |

## W9 — Moving to v0.2.0 · needs the tag

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W9.1 | The library's bound replaces the host clamp (FR-2.4 → HR-3) | `mapwin/bound.go` | `WithBound` once | The M2 instrument unchanged, green before and after |
| W9.2 | `Report` replaces `Describe` for the description; the "nearby" Setting (FR-7.4, go-tuiMaps D-43, D-57) | `mapwin/describe.go`, `platform/config` | `SetNearby` from the Setting | M1b green on the new source; a "nearby" test |
| W9.3 | Fetch options: watchpost's user-agent (HR-6, NFR-4) | `app/maps.go` | `SetFetchOptions{UserAgent: "watchpost/<ver>"}` | The user-agent names watchpost |
| W9.4 | Retention and purge enforced by the library (FR-3.9, FR-3.10, HR-8) | `app/maps.go` | `MaxAge(7 days)` for tiles; `Purge` behind a Settings action | A 7-day-old tile is fetched again; purge empties everything |
| W9.5 | The motion description (go-tuiMaps D-42) in the description's words | `mapwin/describe.go` | — | A two-frame fixture: where the heavier rain was, then is, and never a forecast |

## Deviations from DISCOVER, recorded

- **The description ships on `Describe` in P1-a and moves to `Report` in P1-b.** DISCOVER assumed a
  single description source; go-tuiMaps D-57 added `Report`.
- **FR-2.4's host clamp** is kept until W9.1, as DISCOVER foresaw.
