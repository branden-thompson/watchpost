---
title: "0.18.0 — Observer maps — IMPLEMENTATION PLAN"
date: 2026-09-23
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "DRAFT, revised after the internal plan check — for the PLAN red team, then HUM LEAD approval. No code: signatures, shapes, test descriptions, file paths and order only (D-13)."
---

# Implementation plan

**Goal.** Build 0.18.0's macro phase 1 (drawing and access) as `01-objectives/requirements.md`
states it, test-first, in the order go-tuiMaps' `radar-loops/03-architecture-design/integration-map.md`
fixes.

**Tech stack.** Go 1.25; Bubble Tea v2 and Lip Gloss v2 (existing); go-tuiMaps, first at the
`v0.1.0` tag (P1-a), then at the `v0.2.0` tag (P1-b). No other new module.

**Architecture.**
- The map is drawn in `Update` by one owner goroutine, and `View` only prints (D-41).
- The map renders on every event it owns, and a tested property keeps an old frame from ever showing
  (D-41, revised by D-45).
- A partial area is drawn as found, labelled, and its missing zones are named (D-42).
- The numbers are D-43's, with M5 re-set by D-46.
- Watchpost's structure holds: `domains/` are data sources, `platform/` is shared leaves, `modes/` are
  surfaces, and `app/` is the composition root, checked by `lint-imports`. **`app/` is also the only
  package that may name a domain and what draws** (0.17.0's `app/mapgeometry.go`), so turning alerts,
  fire and quakes into map overlays lives there, in `app/mapfeed.go`, beside the wiring in
  `app/maps.go`.

**Branch.** `feature/map-drawing` → squash-merged into `release/v0.18.0` at SHIP (D-2).

**Every task follows the same order.**
1. Write the named test, and watch it fail for the named reason.
2. Make the change.
3. Verify: the named test is green, and `make verify` is green (`make verify-docs` for a
   Markdown-only change).
4. Mutation check: delete or disable the thing the test protects, watch the test fail, then restore.
   For a task that extends a guard, the mutation is the omission the guard exists to catch (for
   example, a map pane field that is not a Dashboard field).
5. **A task that wires something** also names the path from the composition root and shows the
   reachability gates see it: `make wires` (`tools/wires`) and
   `TestEveryLivePipelinesMethodIsReachedFromProductionCode` (required reading rule 8).

**Human-graded instruments.** M1 and M1b are scored by the HUM LEAD (D-29, D-36). A task carrying
one is done when the score is recorded in `08-reports/`, not when a test passes.

**The trace.** The table at the end maps every requirement to its task and test. `requirements.md`'s
Instrument column is brought up to date once, at BUILD exit, under one ruling row (D-40 requires a
ruling for every change to it).

## Where new code lives

| Path | Holds | Why there |
|---|---|---|
| `modes/tty/map_*.go` (new files, package `tty`) | The window, the map pane and its key, messages, key handling, the size notice, the description's layout, the notes line | Every window is a file of package `tty` (`modal_location.go`, `detail_fire.go`); the memo-completeness and closed-set guards are package-internal, and a subpackage would import-cycle with the Dashboard |
| `app/maps.go` (new) | Wiring: the library, its options, the sources, the Settings gate, `livePipelines` methods | The composition root |
| `app/mapfeed.go` (new) | Alerts, fire and quakes → overlays; scope; labels; partial-area notes; the cost estimate | The only package that may name a domain and what draws |
| `app/dashboard.go` (existing) | The zone-seed gate (`:89`); new `livePipelines` methods (declared at `:653`) | Where both live today |
| `app/setup.go`, `modes/tty/setup_rows.go` (existing) | The map Settings rows and the "Clear map data" action | The Settings window (`s` opens setup) |
| `domains/radar/` (new) | IEM and MRMS sources, one per file, registered; frame validity | A data source is a domain |
| `domains/weather/nws/zones/zones.go` (existing) | The 512-zone cap report; the 7-day refresh | Where zone shapes are fetched (imported by `app/mapgeometry.go`) |
| `platform/geo/regions.go` (new) | The region set and the bound check | Shared and pure |
| `platform/httpx/` (existing) | A per-request body cap enforced as it reads; purge by host | The HTTP client every source uses |
| `platform/config/` (existing) | The map Settings fields and their migration | Settings persistence |
| `go.mod`, `THIRD_PARTY_LICENSES.md` | go-tuiMaps v0.1.0, then v0.2.0 | The dependency and its licence |
| `tools/plancode/` (new), `Makefile` | FR-8.1's gate | A parser, like `tools/authoring`, beside the other `lint-*` gates |
| `<package>/testdata/maps/` | Recorded alert, zone, M1-scenario and radar responses | Beside each consumer, as `domains/alerts/testdata` does |

## Work packages

| WP | What | Requirements | Needs | Lands on |
|---|---|---|---|---|
| W0 | Foundations: the library, the fixtures, the plan-code gate | FR-8.1, FR-8.6 (P1-a half) | — | go-tuiMaps v0.1.0 |
| W1 | The window, its words, its Settings | FR-1, FR-7.4 (renderer), FR-9 | W0 | v0.1.0 |
| W2 | Draw in `Update` on every event; the freshness guards; workers; M5's first arm | D-41, D-45, FR-8.2–FR-8.5, FR-3.3, M5 | W1 | v0.1.0 |
| W3 | Basemap: source, the request gate, caches, credits, closed list, clear path | FR-3, NFR-3, NFR-4 | W2 | v0.1.0 |
| W4 | The bound, held by the host | FR-2 | W2 | v0.1.0 |
| W5 | Alert areas, scope, partial areas, zones | FR-4, D-42 | W2 | v0.1.0 |
| W6 | Fire and quakes | FR-6 | W5 | v0.1.0 |
| W7 | Without colour; the description scored | FR-7 | W5 | v0.1.0 |
| W8 | Radar loops and the motion Setting | FR-5, FR-3.9 (radar), M3, M5, M6, NFR-2 | W2, **go-tuiMaps v0.2.0 tag** | v0.2.0 |
| W9 | Move to v0.2.0: bound, `Report`, fetch options, retention and purge, the frame's counters, labels, pattern | FR-2.4, FR-3.9, FR-3.10, FR-7.1, FR-7.4, HR-3, HR-6, HR-7, HR-8, HR-10, D-42 | W4, W7, **v0.2.0 tag** | v0.2.0 |

**D-60 (2026-09-25): W1–W7 build on the newest go-tuiMaps release candidate, not v0.1.0, and W9's tasks fold into their P1-a counterparts (the W9 section says where each went).** W0–W7 are **P1-a**: they need nothing from v0.2.0 and start at once. W8–W9 build against go-tuiMaps release candidates as their packages land (D-51). On their own they are the
ship-without-radar fallback (RK-4). W8–W9 are **P1-b**, and
wait for the tag (FR-5.6). The integration map uses these numbers.

---

## W0 — Foundations

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W0.1 | The module rules: every required module's licence is listed at its version, and `go.mod` has no `replace`. **go-tuiMaps `v0.1.0` is required by the first task that imports it** (W1.4, the description renderer), because `go mod tidy` drops a requirement nothing imports — a plan error found in BUILD | `cmd/watchpost/modules_test.go`, later `go.mod`, `go.sum`, `THIRD_PARTY_LICENSES.md` | — | Both tests fail on a planted unlisted module and a planted `replace` (mutation-checked) |
| W0.2 | The P1-a fixtures committed before anything uses them: nine M1 scenarios captured live — alerts, the zones they name, the place, an answer key for the HUM LEAD to confirm — the West Texas Flood Watch with TXZ277 withheld among them (FR-8.6) | `app/testdata/maps/m1/`, `app/maps_fixtures_test.go` | A folder a scenario and a manifest | A test fails on a file the manifest names that is missing, and on a set short of the protocol's kinds; the "no socket in map packages" check lands with the first map package |
| W0.3 | A gate refuses implementation code in plan documents (FR-8.1, D-13, D-14) | `tools/plancode/`, `Makefile` (`lint-plan-code` in `verify-gates` and `verify-docs`) | The Go parser judges each fenced Go block in every design and plan document (`03-architecture-design`, `04-development`): a function body, a function literal or a statement is refused; signatures, types and constants pass. Features shipped before D-13 are exempt by name, with their reasons | Its self-test: a positive control (a body) is refused, a negative control (a signature) passes |

## W1 — The window, its words, its Settings (FR-1, FR-7.4, FR-9)

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W1.1 | `g` opens and closes the map window, visible in Help, overridable in `[keys]` (FR-1.1) | `modes/tty/map_window.go`, `platform/term/term.go` (the keymap), `modes/tty/help_about.go` | A keymap action `map.toggle` | Help golden lists it; a `[keys]` override rebinds it *As built (batch 1): `map.toggle` in NAVIGATE; `TestTheMapKeyOpensAndClosesTheWindow`, `TestHelpListsTheMapKey`, `TestAKeysOverrideRebindsTheMap`. The window is built from `Config.NewMap(size)`, handed by `app/maps.go` (`TestTheAppHandsTheWindowItsMap`); the library moves only a sized map, so the constructor takes the window's size.* |
| W1.2 | Centred on the selection, and following it; selection gains a message (FR-1.2, W1-C) | `modes/tty/nav.go`, `modes/tty/map_window.go` | `type selectionMsg struct{ index int }` | Scripted PTY on the real binary: move the selection, and the map's title and centre follow |
| W1.3 | No selection draws a stated state (FR-1.3) | `modes/tty/map_window.go` | — | `selectedLocation()` nil → the stated text, never blank *As built: `TestNoSelectionIsAStatedState` — and no map is built for it.* |
| W1.4 | **The description renderer** (FR-7.4, D-26, D-29, D-55): the alerts shown — names, severity, times, from watchpost's own alert data — each with the library's answer for the selected place (one overlay per alert, so each answer names its alert), and the location's hazard and forecast facts from the snapshot. It names the place, never says "you", and every line passes the text cleaner | `modes/tty/map_describe.go` | `type shownAlert struct{ alert snapshot.Alert; answer tuimaps.Description }`; `func describeLines(shown []shownAlert, loc *snapshot.Location, width int) []string` | Unit tests over fixtures (W0.2): a place inside, outside and near an area; the forecast line present; the word "you" absent *As built (batch 6, W9.2 folded): on `Report`, in M1's words; `map_describe_test.go` and `TestTheDescriptionAnswersTheM1Key` over every recorded scenario. The type is not `shownAlert`: the pane keeps the `PlaceReport` and the join is by alert id.* |
| W1.5 | The size floor with one owner for the number; below it, the notice carries the description (FR-1.4) | `modes/tty/map_size.go` | `var mapMinBody = tuimaps.Size{Cols: 69, Rows: 12}`, the only place the number is written | Size sweep 20×10 → 200×80: every frame is a map ≥ 69×12 or the notice **with words**, never overflow *As built (batch 7): `mapMinBody` the one owner; `TestEverySizeShowsAStatedStateAndNeverOverflows`, `TestBelowTheFloorTheNoticeCarriesTheDescription`, `TestTheMapDrawsAtTheDocumentedFloor` (the window's height and width were changed so the floor fits at 80×24).* |
| W1.6 | The degradation order and the legibility tier (FR-1.9, D-55): the partial-area line ranks above the picture; **when the description's mode is on, it comes first in reading order**, above the braille; the description scrolls (keys from W1.16) | `modes/tty/map_size.go`, `modes/tty/map_describe.go` | — | Size sweep asserting a stated state, never an empty body, including below 20×10 and a county view at 69×12; **at every size every shown alert's name and relation is reachable**, by scrolling if need be *As built (batch 7): the description first in reading order when on (D-55), the body scrolling with PgUp/PgDn; the legibility tier is the status line's "coarser" and "offline" words (batch 3); `TestTheDescriptionComesFirstWithThePicture`.* |
| W1.7 | The window joins every closed-set window test (FR-1.5) | `modes/tty/modal_reachability_test.go`, `memo_completeness_test.go`, the margin, `modalLines` and glyph-survey tests | — | The closed-set tests fail until it is registered *As built: they did, all five, and each named a real hole: the memo key lacked the pane (a frame drawn before the tiles landed was replayed after — the stale frame D-45 exists to stop), the body ran into the border, the map was not redrawn on resize, and `mapWorkedMsg` had no route to Observer. The reachability guard now reaches 80×24 through the resize message, as a terminal does, not by assigning the fields.* |
| W1.8 | Maps off in Settings: `g` says so in one line; nothing built or fetched (FR-1.6, NFR-3). Wired from `app/maps.go` | `modes/tty/map_window.go`, `app/maps.go` | — | Setting off → the map constructor is never called and a counting transport sees zero requests *As built: `TestMapsOffSaysSoAndBuildsNothing` (the constructor is never called, so nothing can be requested).* |
| W1.9 | `--ascii` shows the description, never braille; the braille notice in the chrome (FR-1.7, FR-1.8) | `modes/tty/map_window.go` | — | `TestASCIIFramesCarryNothingButASCII` extended to the window; chrome golden |
| W1.10 | Settings rows: maps on/off, and the description's mode (with the picture, instead of it, off), reachable without restart (FR-9.1) | `platform/config/`, `app/setup.go`, `modes/tty/setup_rows.go` | One row each, persisted | Settings round trip; a config migration test from 0.17.0's file; a PTY test reaches the description with no flag *As built: WATCHPOST UI rows (toggle + picker); `TestTheMapSettingsMigrateAdditively` against the newest recorded file (0.15.0's); the description is reachable with no flag by default. The PTY journey is owed with W1.2's.* |
| W1.11 | Settings rows: default scale, layers, alert scope (FR-9.1, FR-2.3, FR-4.3) | same | — | Settings round trip per row |
| W1.12 | The exposure disclosure: on first map open **and** in Settings beside the maps row, naming each source the session will contact and what it is sent — for radar, a fixed region, not the view (FR-9.4, D-47) | `modes/tty/map_window.go`, `modes/tty/setup_rows.go` | — | Text test on both surfaces; the named sources equal FR-3.8's list for the layers switched on *As built (batch 8): on the session's first open above the map, and on the Maps tab directly under the Maps row (D-62); built from the closed list. `TestTheFirstOpenSaysWhatIsSent`, `TestTheDisclosureNamesEverySourceAndTheRetention`.* |
| W1.13 | A registry for layers, sources and options (FR-9.3). Members register from `app/maps.go`; FR-3.8's closed list is checked against what registered | `app/maps.go` | Members register themselves and are discovered | An architectural test adds a fake layer without editing the others; `make wires` sees every registration reached |
| W1.14 | The cost warning above 2 MB or 40 requests a refresh (FR-9.2, D-43) | `app/mapfeed.go`, `modes/tty/map_window.go` | `func refreshCost(layers []MapLayer, scope AlertScope, step time.Duration) (bytes int64, requests int)` from measured sizes | Estimates at, below and above each threshold; the warning's words golden |
| W1.15 | **Pan and zoom** (FR-1.10, D-49): the window's keymap actions drive the library's intents; the view rehydrates; the loading indicator (W8.9b's, brought forward) shows while the frame is not complete; the host clamp (W4.4) wraps every call | `modes/tty/map_window.go`, `modes/tty/map_pane.go`, `platform/term/term.go` | Keymap actions `map.pan.*`, `map.zoom.in`, `map.zoom.out` | Scripted PTY: pan each way and zoom in and out; the indicator shows then clears; M2 over every frame *As built (batch 2): the keys D-61 ruled — arrows pan a quarter of the view, `+`/`=` `-` zoom — held by `TestTheMapWindowOwnsItsKeys`. The loading indicator, the PTY journey and M2 are still owed (with W4).* |
| W1.16 | **The map's key controls evaluated together** (FR-1.11, D-49): every map action listed with the Observer's keymap, collisions found, default bindings proposed as one ruling | `platform/term/term.go`, `06_docs/02_features/observer-maps/02-analysis/` | A table of actions and proposed keys | A keymap test fails on any collision; the HUM LEAD's ruling recorded *As built: D-61; `TestTheMapKeysAreActionsHelpShows` refuses a collision within the map's scope, requires every map action bound, listed in Help's MAP group and rebindable.* |
| W1.17 | **The legend window** (D-44, D-54): a picture-in-picture window over the map, from a chip `[ L ] Legend` in the window's chrome (binding from W1.16); contextual — a key for everything on the map that needs one; in P1-a the alert severities as drawn on v0.1.0 | `modes/tty/map_legend.go`, `platform/term/term.go`, `modes/tty/help_about.go` | A keymap action `map.legend`, read from the library's `Legend()` | Help lists it; a `[keys]` override rebinds it; with alerts on the map the legend lists each severity shown; it joins the closed-set window tests *As built (batch 8): `L` (D-61), a box over the map's corner keying only the severities drawn; `[L] Legend` on the status line. `TestTheLegendOpensOverTheMap`, `TestTheChipNamesTheLegend`.* |

## W2 — Drawing in `Update`, and never an old frame (D-41, FR-8)

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W2.1 | The map pane as Dashboard fields: the stored lines and the counters they were drawn at (FR-8.4, D-45) | `modes/tty/dashboard.go`, `modes/tty/map_pane.go` | `type mapPane struct{ m *tuimaps.Map; lines []string; changed, ticks uint64 }`. On v0.1.0 `ticks` stays zero and `changed` is read from `Changed()` after the render | The memo-completeness guard sees the pane's fields: a pane field that is not a Dashboard field fails *As built: `mapPane{m, lines, changed, ticks, gen, failed}`; the window's memo keys on `gen`, raised by every draw, and on `failed`.* |
| W2.2 | Library calls only in `dispatch`; `Work` in a `tea.Cmd` (C-7, FR-3.3) | `modes/tty/dashboard.go`, `modes/tty/map_work.go` | `mapWorkedMsg{did bool}`; `mapTickMsg{at time.Time}` scheduled with `tea.Tick` at `NextCall`. Every library call goes through one wrapper that records the goroutine it ran on | A test drives messages and asserts, through the wrapper's record, that every call but `Work` ran on the Bubble Tea goroutine *As built (first half): every call goes through `mapPane.call`, which records names; `TestTheMapIsDrawnInUpdate` holds that View calls nothing and that only Render, Pending and Work are made. The goroutine record and the `NextCall` tick are still owed.* |
| W2.3 | **Guard 2 (D-45):** every event that can change the frame triggers a render — each mutation, each `Work` that did something, each tick. P1-a rows: data, `Work` landing a tile, size, selection, depth, theme, units, each map Setting. The frame-advance and playback rows land in W8.10 | `modes/tty/map_pane_test.go` | Render on every event; no memo key | A table, one row per event → a render |
| W2.4 | **Guard 3:** the printed lines equal a fresh render from a second map built from the same inputs (a fresh render on the same map would disturb what it checks) | `modes/tty/map_property_test.go` | — | Random message sequences, 10,000 runs |
| W2.5 | **The freshness property (D-45):** after every event, the stored frame's counters equal the library's | `modes/tty/map_pane_test.go` | — | A handler that skips its render after a `Set` or a landed tile fails the property |
| W2.6 | Every map worker joined on close (FR-8.2) | `modes/tty/map_work.go` | — | A goroutine-count test in the style of `TestAScheduleLeavesNoGoroutineBehind` |
| W2.7 | An allocation pin on the frame; a memo hit does not re-render (FR-8.3) | `modes/tty/map_pane.go` | — | An `AllocBudget` test |
| W2.8 | Every golden of the window carries width invariants at 80, 120 and 133 (FR-8.5) | the golden helper in `modes/tty` | — | The helper fails a golden whose line widths differ from the frame's |
| W2.9 | **M5's first arm** (D-43, D-46, D-53): first complete frame — every alert's area and the basemap — ≤ 3.5 s cold at 149×38, p90 of 20 opens | `modes/tty/map_timing_test.go`, the SHIP report | A latency profile taken from wave 1's measured parts (zones 0.16–0.36 s each, TileJSON 0.26 s, tiles 50–100 ms) | **In the gate:** the request count to the first complete frame, and that it never waits for radar. **At SHIP:** the p90 measured on the reference machine, plus one live-network sample |

## W3 — The basemap (FR-3)

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W3.1 | The OpenFreeMap source, with the embedded tiles always passed; built in `app/maps.go`, handed to `tty` (FR-3.1) | `app/maps.go` | — | Offline: a state view at 69×12 is `Complete`; `make wires` sees the constructor reached *As built (batch 3): `mapBuilder` in `app/maps.go`; a state view with the source answering is `Complete` (`TestTheBasemapIsTheClosedListsSource`). Offline, a state view cannot be `Complete` - the embedded tiles stop at zoom 3 - so W3.3's line says it is the coarser picture.* |
| W3.2 | **No map request of any kind before the listener asks, zone geometry included** (FR-3.2, D-21, D-25): `lp.seedZoneShapes` (`app/dashboard.go:89`) runs only when maps are on and on the first map open | `app/dashboard.go`, `app/maps.go` | The map, its sources and the zone seed are built on the first `g` | A full station start with maps on, through a counting transport: zero requests to any FR-3.8 host for map purposes, zone geometry included, until `g` *As built: the seed runs on the first map built (`TestNoMapRequestBeforeTheListenerAsks`), and an AST check refuses any other caller. The full station-start instrument is still owed.* |
| W3.3 | Offline and failing states said in the window (FR-3.4) | `modes/tty/map_window.go` | — | Golden per state over `Sharpening` and `tile-failed` *As built: a status line under the map - loading (work pending), offline (a tile failed, until whole again), coarser (nothing pending, no finer tiles) - held by `TestTheWindowSaysWhatThePictureIs` and `TestTheOfflineNoteClearsWhenTheSourceReturns`.* |
| W3.4 | Caches under the OS cache dir with **one stated total** (FR-3.5) | `app/maps.go` | `userCacheSubdir("map")`; the total written once | A test sums the configured caps (map, radar, the 256 MB HTTP cache) and compares them with the stated total *As built: `statedCacheBytes` = the map's 256 MiB + `httpx.DiskCacheBytes`; radar joins with W8.* |
| W3.5 | The memory tile cache sized for the largest view (FR-3.6) | `app/maps.go` | `NewShared(4 MiB)`, from W1-B's 909 KB at 149×38 | Measured at 149×38: a second open of the same view makes no tile request *As built: a 4 MiB shared set kept across the session's maps; go-tuiMaps measured about 800 KB for 200×60 (its L10.6).* |
| W3.6 | Credits at every width; credit text bounded and neutralised (FR-3.7) | `modes/tty/map_window.go` | A length cap and the existing text cleaner | A hostile TileJSON credit (overlong, escape bytes, reading as a status line) is cut and cleaned, and never lands in the status row; golden at 69, 100 and 149 columns *As built: watchpost never prints the TileJSON's credit; the library draws its own fixed one (`TestAHostileCreditNeverReachesTheWindow`). Goldens at three widths are owed with W2.8.* |
| W3.7 | The closed source list (FR-3.8) | `app/maps.go`, `domains/radar/` | The list as a table in code | **Positive:** the registered set equals FR-3.8's table, member for member. **Negative:** no path accepts an arbitrary URL *As built: `basemapSources`; the builder takes no address, and every request made is to the listed host (`TestTheBasemapIsTheClosedListsSource`). Radar's half joins with W8.3.* |
| W3.8 | **The P1-a clear path and stated retention** (FR-3.9, FR-3.10, D-55): a Settings action "Clear map data" closes the map, then deletes the tile directory and the HTTP cache's entries for map hosts **and for zone geometry** (zone fetches get an explicit TTL so they are clearable); the Settings text states the retention and, until HR-8, that it is a byte cap | `app/setup.go`, `modes/tty/setup_rows.go`, `platform/httpx/` | `func (c *Client) PurgeHost(host string) (removed int, err error)` | After a map session, the action leaves the tile directory absent and no `<sha256>.cache` entry for any map host or zone; **with a tile fetch in flight, nothing is written back after the clear** *As built (batch 8, W9.5 folded): on the Maps tab; the live map's `Purge`, then the app: `Purge` on a bare map, zone `Forget` and `ForgetCached` (`httpx.ForgetPrefix`, not `PurgeHost`: the forecasts stay). `TestClearMapDataEmptiesTheLiveMapAndAsksTheApp`, `TestClearingEmptiesWhatTheMapKept`.* |
| W3.9 | Politeness for zone geometry (NFR-4, D-46): bounded at six in flight (`zones.go:37`), with watchpost's user-agent | `domains/weather/nws/zones/zones.go` | — | A test over the transport: never more than six zone requests in flight; the agent names watchpost |

## W4 — The bound, held by the host (FR-2)

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W4.1 | The region set and the check (FR-2.1, FR-2.5) | `platform/geo/regions.go` | `func RegionOf(loc Point) (Region, bool)` | Every region the APIs cover: the contiguous US, Alaska (across the antimeridian), Hawaii, the Caribbean and Pacific territories, **the marine areas**; an out-of-region point is a stated state *As built (batch 4): `RegionOf(lat, lon)`; six boxed regions with their waters; `TestEveryCoveredPlaceHasARegion`, `TestAPlaceInNoRegionIsAStatedState`.* |
| W4.2 | The view placed before the first frame (FR-2.2) | `modes/tty/map_pane.go` | — | The first frame's span is within the bound *As built: the bound is set before the first frame; M2 checks it from frame 0.* |
| W4.3 | The default scale as a Setting; the bound is not (FR-2.3) | `platform/config/` | — | Settings round trip (row from W1.11) |
| W4.4 | The host clamp at every view change until HR-3 (FR-2.4) | `modes/tty/map_bound.go` | Wraps every view call and every `Render` | **M2 instrument:** every rendered frame in the suite and the PTY journeys checked against the region *As built with W9.1: no host clamp; the library's `SetBound` with a least zoom that fits the view inside the region on both axes, set again on resize. `TestNoFrameIsWiderThanTheRegion` checks every frame of a drive through every key and three sizes, for three regions. The PTY journeys join it when they exist.* |

## W5 — Alert areas (FR-4, D-42)

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W5.1 | `resolveAlertAreas` reachable from production through a `livePipelines` method (FR-4.1, C-1) | `app/dashboard.go`, `app/maps.go` | A `livePipelines` method | `TestEveryLivePipelinesMethodIsReachedFromProductionCode` extended; `make wires` *As built (batch 5): `lp.mapFeed`, handed as `Config.MapFeed` (`TestTheFeedIsReachedFromProduction`).* |
| W5.2 | **One overlay per alert**, one feature per area; severity to role; times and sender carried; **the severity word in the label from P1-a** (FR-4.2, D-55) | `app/mapfeed.go` | — | Round trip against the library's own reading of rings; each label carries its severity word *As built: `TestEveryAlertIsOneOverlay` (the library's `Report` reads it back with its severity), `TestEverySeverityHasItsRole`, `TestAnAlertOnTwoPlacesIsOneOverlay`; the window's half in `map_feed_test.go`.* |
| W5.3 | Scope: the station's locations plus national severe events in view; a Setting (FR-4.3) | `app/mapfeed.go` | — | A count of drawn alerts per scope *As built: the station's locations' alerts; the national-severe half and its Setting are owed with W1.11.* |
| W5.4 | **Partial areas, drawn as found, labelled, the missing zones named, and whether the place is in one** (FR-4.4, D-42). "In one" is decided by UGC code: the selected location's forecast zone and county (`nws.Provider.ForecastZone`, `CountyUGC`) against the missing ids, because a missing zone has no geometry to test against | `app/mapfeed.go`, `modes/tty/map_notes.go` | The label gets "(N of M zones)"; a note line under the map naming the missing zones **in words, from the alert's area description, never by code** (D-55) | **M4 instrument** over `Area.Missing` fixtures (W0.2), including specimen 31's case: Fort Davis in the missing zone TXZ277 *As built: `TestAPartialAreaIsDrawnAsFoundAndSaid` over the recorded Fort Davis scenario, both in and out of the missing zone.* |
| W5.5 | The 512-zone cap reported, not dropped (FR-4.5) | `domains/weather/nws/zones/zones.go` | — | A test at 513 zones *As built before 0.18.0: `TestAskingForMoreZonesThanExistIsBoundedAndSaysSo` asks the cap plus 40.* |
| W5.6 | Zone shapes fetched again on next use once 7 days old (FR-4.6) | `domains/weather/nws/zones/zones.go` | — | Fixed clock: 6 days → no fetch; 8 days → fetch; `make wires` sees the refresh reached *As built: `TestAHeldZoneIsFetchedAgainOnceAWeekOld` (6, 8 and 9 days).* |

## W6 — Fire and quakes (FR-6)

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W6.1 | Hotspots, incidents and quakes as static point layers (FR-6.1) | `app/mapfeed.go` | One layer each, registered (W1.13) | Golden with each layer on and off; Settings round trip |

## W7 — Without colour, and the description scored (FR-7)

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W7.1 | The colour-depth hint (FR-7.1). Order: `NO_COLOR` set → none; the Monochrome theme → none (so the hatch runs); otherwise the lower of the theme's depth and the terminal's reported profile (`tea.ColorProfileMsg`). Never a depth the terminal did not report | `modes/tty/map_pane.go` | `func mapDepth(noColor bool, theme Theme, profile colorprofile.Profile) tuimaps.Depth`, applied with `ColourDepth` | A table over env × theme × reported profile; the invariant "every alert area present has a hatch or a label" at each resulting depth |
| W7.2 | **M1b scored** (FR-7.4, D-53): the recorded scenarios answered from the description alone; **an answer-key test** per scenario (the place, each alert, the expected relation word present) | `08-reports/`, `modes/tty/map_describe_test.go` | — | The answer-key test in the gate; **human-graded:** done when the HUM LEAD's score is recorded |
| W7.3 | Coordinates never spoken; the description held to the speech guard (FR-7.3) | `modes/tty/map_describe.go` | — | The existing say-safe guard extended over the description's text |
| W7.4 | Every palette passes the library's checker, as a matrix (FR-7.5) | `app/maps.go` | Palette × theme ground (dark, light, monochrome) × depth (truecolor, 256, 16, none); outline ramps checked with `Lines: true`, fills with `Midpoint` set per ramp | `CheckRamp(ramp, RampCheck{Ground, Depth, Lines, Midpoint})` per cell; findings kept where the rule is the vision rule |
| W7.5 | Map colour tokens in the contrast register (FR-7.2, FR-7.6) | the contrast register, `modes/tty/map_*.go` | — | The no-literal guard; the token-completeness test |

## W8 — Radar (FR-5) · needs the go-tuiMaps v0.2.0 tag

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W8.1 | `go.mod` requires the newest go-tuiMaps release candidate holding the packages the next task needs, then the final `v0.2.0` at SHIP; no local replace (FR-5.6, D-51) | `go.mod`, `go.sum`, `THIRD_PARTY_LICENSES.md` | — | W0.1's test: a tagged version, never a replace; a SHIP check refuses an `-rc` tag |
| W8.2 | Radar fixtures committed before anything uses them (FR-8.6, radar half) | `domains/radar/testdata/` | Recorded time lists and frames: off-grid, expired, empty, the 2011 default | The manifest test from W0.2 |
| W8.3 | IEM and MRMS as registered sources; the source is a Setting (FR-5.1). MRMS relies on the library's MRMS table (**WP-L6**). **A source is asked for a radar region, never a view** (D-47) | `domains/radar/iem.go`, `domains/radar/mrms.go`, `app/maps.go`, `modes/tty/setup_rows.go` | `type Source interface{ Name() string; Times(ctx) ([]time.Time, error); Frame(ctx, t time.Time, r Region) ([]byte, error) }` | Registry test; Settings round trip; `make wires`; a test that no request carries the view's box |
| W8.3a | **The radar region table** (D-47, D-54): fixed state-regional boxes to start (the exact scale set in UAT), each sized so a county or state-regional view sits inside one, fetched at a fixed size within the library's per-image cap; a view crossing an edge takes the neighbour too | `domains/radar/regions.go` | `type Region struct{ Name string; W, S, E, N float64; Cols, Rows int }`; `func RegionsFor(view Box) []Region` | Every station location's default view lies inside one or two regions; each region's image is within the cap; a selection change inside a region fetches nothing new |
| W8.3b | **The radar step is a Setting** beside the source (FR-9.5, D-50): 5 minutes by default for every source, MRMS included; the listener may opt into a source's native cadence or back off to a longer step; two hours kept (24 frames at 5 minutes); the image budget set to fit the choice (6 MiB by default, go-tuiMaps D-68) | `platform/config/`, `modes/tty/setup_rows.go`, `app/maps.go` | The choices and their values set in BUILD | Settings round trip; the frames fetched match the step; a costlier choice raises FR-9.2's warning; the budget handed to the library covers the chosen loop |
| W8.4 | A frame is valid only if its time is advertised and its image isn't empty (FR-5.3) | `domains/radar/valid.go` | — | The W8.2 responses |
| W8.5 | **The body cap inside the HTTP client** (FR-5.7): refused as it reads, **never cached**; the decoded dimensions capped from the PNG header before anything decodes | `platform/httpx/`, `domains/radar/valid.go` | A per-request option `httpx.BodyCap(n int64)` checked inside the client's read, before its cache write; a header check against the library's per-image cap; **radar dials through a check that refuses loopback, private and link-local addresses after resolution, https pinned** (RK-11, D-55) | After a 2 MiB response the cache is **empty** and the caller has an error; a small PNG declaring huge dimensions is refused undecoded; a resolver answering 127.0.0.1 is refused; plain http is refused |
| W8.6 | One overlay per source, absolute times (FR-5.2) → `Image.Frames` (**WP-L2**); the advertised times trimmed to the newest that fit `MaxFrames` and the budget (D-55) | `app/mapfeed.go` | — | A loop across a 5-minute rollover has no duplicate or skipped frame; a source advertising more than fits hands in the newest that fit |
| W8.7 | A refresh fetches only frames not held; no source polled faster than its cadence; requests sequential (FR-5.5, NFR-4) | `domains/radar/` | — | A request count over two refreshes; a clock test that the cadence is honoured |
| W8.8 | The newest frame's age always visible; stale marked (FR-5.4, **WP-L4** `LoopState`) | `modes/tty/map_window.go` | Uses `LoopState.Newest` and the frame's time | **M3 instrument** |
| W8.9 | The motion Setting (off / slow / normal) sets the rate, **slow by default** (D-52); "off" disables play and drives `ReduceMotion` (FR-5.8, D-27, D-35, D-48, **WP-L4**). Rates slow 1 frame a second, normal 2, the last frame held 2 s (FR-5.9, D-43). The still form keeps the frame and its age **and freezes the description** | `modes/tty/map_pane.go`, `platform/config/`, `modes/tty/setup_rows.go` | One call per Setting change | The library's frame schedule at each setting matches the rates; a still-form golden; the description does not change across frame advances while stopped |
| W8.9a | **The listener's playback controls** (D-48): the map opens stopped on "right now" (the newest observed frame); **play** runs from the oldest held frame through "right now" and on through forecast frames if any, labelled forecast; **stop** holds the frame on screen; **reset** returns to "right now"; **←/→** scrub one frame, stopping playback. Keymap actions `map.play`, `map.stop`, `map.reset`, `map.back`, `map.forward`, listed in Help, overridable in `[keys]`; default bindings from W1.16's evaluation | `modes/tty/map_window.go`, `platform/term/term.go`, `modes/tty/help_about.go` | Driven only through the library's playback API (**WP-L4**, go-tuiMaps D-67) | A scripted PTY journey: open → "right now"; play → the oldest frame first, then each in time order; stop → held; ← and → → one frame each; reset → "right now"; a refresh while stopped keeps the same moment |
| W8.9b | A loading indicator while the map fetches for a view it cannot yet draw complete (D-48) | `modes/tty/map_window.go` | From the frame's status (`Sharpening`) and pending work | Golden: a cold open shows the indicator; it clears when the frame is complete |
| W8.10 | Guard 2's frame-advance and playback rows (from W2.3); the freshness property over frame advances | `modes/tty/map_pane_test.go` | — | The two rows → a render; after an advance, the stored `FrameTicks` equals the library's |
| W8.11 | Radar over alert areas: the blend, furniture never erased (**WP-L3**) | `app/mapfeed.go` | — | Goldens over specimen 29's scene at each depth; every outline, label and credit present |
| W8.12 | **M5's radar arms** (D-43, D-53): newest frame ≤ 3.0 s, whole loop ≤ 5.0 s, the first frame never waits for radar (**WP-L3**) | `modes/tty/map_timing_test.go`, the SHIP report | — | **In the gate:** request counts and "never waits". **At SHIP:** the p90s measured on the reference machine |
| W8.13 | **M6 / NFR-2 (D-53):** with the Observer live, the intervals between delivered map ticks and the key-to-response latency | `modes/tty/map_timing_test.go`, the SHIP report | — | Measured with the Observer's publishers replaying; thresholds measured in BUILD and brought to the HUM LEAD; recorded at SHIP, not asserted in the gate |
| W8.14a | `BBOX` redacted from `httpx` error text for radar hosts (D-47) | `platform/httpx/` | — | A radar error's text carries no box |
| W8.14 | Radar retention (FR-3.9, radar half): radar requests are cached in memory only, by an explicit request option `httpx.MemoryOnly()` (D-55), and frames past 2 hours leave the loop | `domains/radar/`, `app/mapfeed.go`, `platform/httpx/` | — | After a radar session, no radar entry on disk; a frame older than 2 hours is dropped |
| W8.15 | The cost estimate (W1.14) gains radar | `app/mapfeed.go` | — | The W1.14 table with radar on |
| W8.15a | **MRMS's unverified heavy end, told to the listener** (D-55): while the library's legend marks the source unverified, the note line says so and IEM stays the default source | `modes/tty/map_notes.go`, `app/maps.go` | Reads `LegendEntry.Unverified` | With MRMS chosen and unverified, the note line says so; the default source is IEM |

## W9 — Moving to v0.2.0 · needs the tag

**Folded by D-60.** Each row below is built with the P1-a task D-60 names (W9.1 → W4, W9.2 → W1.4, W9.3–W9.5 → W3, W9.6 → W2.1, W9.7 → W8, W9.8 → W5, W9.9 → W7, W9.10 → W1.17 and W8), and its test is that task's test. The rows stay here as the record of what each must do.

| # | Task | Files | Shape | Test first (RED) |
|---|---|---|---|---|
| W9.1 | The library's bound replaces the host clamp (FR-2.4 → HR-3, **WP-L7**) | `modes/tty/map_bound.go` | `SetBound` on every change of region (go-tuiMaps D-70) | The M2 instrument unchanged, green before and after |
| W9.2 | `Report` replaces `Describe` for the description; the "nearby" Setting (FR-7.4, go-tuiMaps D-43, D-57, **WP-L5**) | `modes/tty/map_describe.go`, `platform/config/` | `SetNearby` from the Setting | The answer-key test green on the new source; **M1b re-scored by the HUM LEAD on this description** (D-53); a "nearby" test |
| W9.3 | Fetch options: watchpost's user-agent (HR-6, NFR-4, **WP-L8**); tile-host confinement (HR-10) | `app/maps.go` | `SetFetchOptions(FetchOptions{UserAgent: "watchpost/<ver>"})` | The agent names watchpost; a TileJSON naming another host is refused |
| W9.4 | Tile retention by the library (FR-3.9, HR-8, **WP-L9**) | `app/maps.go` | `SetCacheMaxAge(7 * 24 * time.Hour)` (go-tuiMaps D-70) | A tile older than 7 days is fetched again |
| W9.5 | "Clear map data" moves to the library's `Purge` (FR-3.9, FR-3.10, **WP-L9**) | `app/setup.go` | The library's `Purge` behind the Settings action from W3.8, with `PurgeHost` kept for radar and zone entries (D-55) | The action empties the library's caches and the HTTP cache's map, radar and zone entries; the FR-3.10 record is updated to "enforced" |
| W9.6 | The pane stores `Frame.Changed` and `Frame.FrameTicks` from the frame itself, and `Changed()` now counts inputs (go-tuiMaps D-66, **WP-L3** L3.2, **WP-L4**) | `modes/tty/map_pane.go` | — | The freshness property green before and after the move; a tile landed by `Work` raises `Changed()` without a render |
| W9.7 | The motion description in the description's words (go-tuiMaps D-42, **WP-L5**) | `modes/tty/map_describe.go` | — | A two-frame fixture: where the heavier rain was, then is, and never a forecast |
| W9.8 | The named place's label kept (D-42's second clause; go-tuiMaps D-60, L-8.9, **WP-L3**) | `modes/tty/map_pane_test.go` | — | Specimen 31's scene at 69×12 and 149×38: "Fort Davis" is on the frame |
| W9.9 | The colour-independent pattern at colour-on depths (HR-7, FR-7.1, **WP-L3**) | `modes/tty/map_pane_test.go` | — | W7.1's invariant at every depth, now including colour-on |
| W9.10 | The legend gains P1-b's keys (D-54): the severity digits (go-tuiMaps D-65), the radar classes, and any value scale the map shows | `modes/tty/map_legend.go` | — | With radar and alerts on the map the legend lists the five digits and their words and the radar classes |

## The trace

| Requirement | Task | | Requirement | Task |
|---|---|---|---|---|
| FR-1.1 | W1.1 | | FR-5.1, FR-9.5 | W8.3, W8.3b |
| FR-1.2 | W1.2 | | FR-5.2 | W8.6 |
| FR-1.3 | W1.3 | | FR-5.3 | W8.4 |
| FR-1.4 | W1.5 | | FR-5.4, M3 | W8.8 |
| FR-1.5 | W1.7 | | FR-5.5 | W8.7 |
| FR-1.6 | W1.8 | | FR-5.6 | W0.1, W8.1 |
| FR-1.7, FR-1.8 | W1.9 | | FR-5.7 | W8.5 |
| FR-1.9 | W1.6 | | FR-5.8, FR-5.9 | W8.9, W8.9a |
| FR-1.10, FR-1.11 | W1.15, W1.16 | | | |
| FR-2.1, FR-2.5 | W4.1 | | FR-6.1 | W6.1 |
| FR-2.2 | W4.2 | | FR-7.1 | W7.1, W9.9 |
| FR-2.3 | W1.11, W4.3 | | FR-7.2, FR-7.6 | W7.5 |
| FR-2.4, M2 | W4.4, W9.1 | | FR-7.3 | W7.3 |
| FR-3.1 | W3.1 | | FR-7.4, M1b | W1.4, W7.2, W9.2 |
| FR-3.2, NFR-3 | W1.8, W3.2 | | FR-7.5 | W7.4 |
| FR-3.3 | W2.2 | | FR-8.1 | W0.3 |
| FR-3.4 | W3.3 | | FR-8.2 | W2.6 |
| FR-3.5 | W3.4 | | FR-8.3 | W2.7 |
| FR-3.6 | W3.5 | | FR-8.4 | W2.1 |
| FR-3.7 | W3.6 | | FR-8.5 | W2.8 |
| FR-3.8 | W1.13, W3.7 | | FR-8.6 | W0.2, W8.2 |
| FR-3.9 | W3.8, W8.14, W9.4, W9.5 | | FR-9.1 | W1.10, W1.11 |
| FR-3.10 | W3.8, W9.5 | | FR-9.2 | W1.14, W8.15 |
| FR-4.1 | W5.1 | | FR-9.3 | W1.13 |
| FR-4.2 | W5.2 | | FR-9.4 | W1.12 |
| FR-4.3 | W1.11, W5.3 | | NFR-1, M5 | W2.9, W8.12 |
| FR-4.4, M4, D-42 | W5.4, W9.8 | | NFR-2, M6 | W8.13 |
| FR-4.5 | W5.5 | | NFR-4 | W3.9, W8.7, W9.3 |
| FR-4.6 | W5.6 | | NFR-5, NFR-6 | every commit (`make verify`, `lint-watermark`) |
| D-41, D-45 | W2.1–W2.5, W8.10, W9.6 | | M1 | the HUM LEAD, at BUILD exit (D-29) |
| D-44, D-54 legend | W1.17, W9.10 | | | |

## Deviations from DISCOVER, recorded

- **The description ships on `Describe` in P1-a and moves to `Report` in P1-b.** DISCOVER assumed a
  single description source; go-tuiMaps D-57 added `Report`.
- **FR-2.4's host clamp** is kept until W9.1, as DISCOVER foresaw.
- **D-41's guards 1 and 4 are withdrawn** and replaced by the freshness property (D-45).
- **FR-3.9's map retention is a byte cap plus a directory delete until W9.4** (FR-3.10 records it).
- **The description is text only in 0.18.0** (D-52); speaking it is F-181's voice pass.
- **Pan and zoom came into 0.18.0** (D-49), which D-33 had put in phase 2.
- **Found in BUILD:** go-tuiMaps is required by the first task that imports it, not by W0.1 (`go mod tidy` drops an unused requirement); W0.3 is a parser tool, not a shell script.
