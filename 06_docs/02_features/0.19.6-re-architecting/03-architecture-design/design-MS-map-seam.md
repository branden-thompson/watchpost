---
title: "0.19.6 PLAN — design MS: the map seam (steps 2a and 2b)"
date: 2026-11-15
phase: PLAN
agent: MS
status: "FILED — checked by the lead against e57afeaf; its choices ruled D-21 to D-26 and A-19 to A-27"
---

# Design MS: the map seam, steps 2a and 2b

> **Filed from PLAN's design agent MS.** Its tables are in `design-data/MS/`. `<scratch>` names the
> session's working directory, which is not committed: the computed graph there is committed in batch 0 as
> its own module (A-12). Where a ruling chose differently from this design, the ruling stands.

## Verdict

The seam can be built inside the settled map (package map §2, D-9) with no interface, no new
message, no new lock and no growth of the models, if four things hold. **First**, 2a keeps the
memo guard and the tests' field paths exactly as they are by *embedding* an unexported `mapModel`
in `Dashboard` (it holds today's `mapPane`, the map's choices, its cost, close counter, release
delay and keymaps, and one pointer to its hooks). 2b turns the embedding into a named field and
gives the guard's walk one rule: a named sub-model is walked as an embedded struct is today.
**Second**, the map's hooks are held once, behind one pointer that `Config` hands over, never in
`Env`. **Third**, `Env` carries plain values only, except the Dashboard's own clock and tick hook,
which are func fields today and must stay functions (a ruling for the lead, C-2). **Fourth**, the
root keeps the window stack, the routing and the post-update chain. Where map code today opens or
closes a window, or moves the selection, and then reads the result, it is cut into two calls at
that point (11 split sites, MS-1).

The cost is the tests, not the code. 2a changes no assertion. It re-points 798 call references to
154 re-homed methods in 68 test files, and 173 hook references in 47. 2b, measured by rule: 408 test
functions reach the map's unexported code. 288 of them move with the code into the map package,
their setup rebuilt. **120 functions in 52 files cannot be served** without a test-only export: 53
assert across Settings and the map, and 36 on the frame the root draws (MS-7). That figure is the
one D-9 asks to be re-measured after 2a.

## What I computed, and how

All scripts and tables are in `design-data/MS/`. They read TS's declaration graph
(`<scratch>/ts/out`, `<scratch>/ts/outT`), the shared graph (`<scratch>/graph/`), the code at
`e57afeaf`, and `06_docs/mutants/*.py`. Nothing in the repository was written or run.

| Script | Output | What it does |
|---|---|---|
| `cross_raw.py` | `cross-raw.tsv` | re-runs TS's Map II projection (`ts/map2.py`), keeping each crossing symbol's declaring file and users: 254 root→map and 122 map→root, as TS found |
| `classify_crossings.py` | **`crossings-MS.tsv`** | MS-3: each of the 376 symbols with its disposition under this design |
| `methods_ms.py` | **`methods-MS.tsv`** | MS-1: the 387 methods 2a re-homes, their root callers, and the adapters kept |
| `layout.py` | — | this design's 2b layout over the declaration graph (map package, controls, kit, root) |
| `tests_ms.py` | `test-reach-MS.tsv` | every unexported symbol each test file reaches, by package under the 2b layout |
| `tests_fn_ms.py`, `classify_tests.py` | **`tests-fn-MS.tsv`** | MS-7 at the test-function level, each test's reach including its helpers' reach, transitively |
| `tests_by_file.py` | **`tests-MS.tsv`** | MS-7 by file (the worst class among its tests) |
| `mutants_ms.py`, `exports-2b.txt` | **`mutants-MS.tsv`** | MS-8: the 289 map anchors, whether 2a's re-homing changes their text, and 2b's renames |

## MS-1. 2a's map model type inside `modes/tty`

### The type and how `Dashboard` holds it

`mapModel` is a new unexported struct in a new file `modes/tty/map_model.go`. Every `Dashboard`
method in a `map_*.go` file moves onto it, keeping its value receiver (GP-9). Its fields:

| Field | Type | From today | Why it moves |
|---|---|---|---|
| `mapPane` (named) | `mapPane` | `Dashboard.mapPane` (`dashboard.go:687`; type at `map_pane.go:63-204`, 111 fields) | the map's state |
| `mapChoices` (embedded) | `mapChoices` | embedded in `Dashboard` (`dashboard.go:696`; type at `dashboard_choices.go:21-42`) | the map's 20 Settings. The map writes them from its own Overlays menu (`map_boxes.go:425-428`, `:531`, `:924`; `map_temp.go:482`) and reads them after the write in the same call, so they cannot be lent through `Env` |
| `mapCost` (named) | `MapCost` | `dashboard.go:701` | written by `refreshMapCost` (`map_prefs.go:158`) and read by the map's own draw (`map_pane.go:723`, `:790`, `:812`) in the same call |
| `mapCloses` | `uint64` | `dashboard.go:690` | the release generation (D-221) |
| `mapReleaseAfter` | `time.Duration` | `dashboard.go:692` | the release delay |
| `mapKeys`, `chartKeys` | `term.KeyMap` | `dashboard.go:702-703` | the map's and the chart's keymaps |
| `hooks` | `*mapHooks` | 15 func fields of `Config` (below) | held once (MS-2) |
| `ackSeen` | `int` | `Config.PropagationAck` (`dashboard.go:236`), written by the map at `map_prop_ack.go:63` | live state the map owns. `Config` keeps the launch value, as it does for the choices |

**`Dashboard` embeds it** (`mapModel` in place of `mapPane`, `mapCloses`, `mapReleaseAfter`,
`mapChoices`, `mapCost`, `mapKeys` and `chartKeys`). TS-9's options are embed, flatten, or a
recursive walk. Embedding is the one that changes nothing the guard sees:

- `walkAt` walks an embedded struct "as the model itself" (`memo_completeness_test.go:160-171`).
  `mapModel`'s named `mapPane` is then one level down, exactly as today (`:179-189`), and its
  embedded `mapChoices` is walked at the top with bare names. So `dashboardWriter`'s
  `mapTempNDFD`/`mapRainFull` cases still fire (`:109-113`). `mapCost` stays one level down, and the
  uint64 and keymap fields are perturbed exactly as today (not at all).
- Field promotion keeps every path compiling unchanged: `d.mapPane.x` (root and tests), `d.mapScale`
  and the other choices, `d.mapCost`, `d.mapKeys`. RR's 586 assertion references to `mapPane.`
  internals (63 files) change **zero** characters in 2a.
- In 2a the embedded type is unexported, so nothing leaks into `tty.Dashboard`'s exported API (CQ
  F-20 concerns 2b only).

In 2b's first commit the embedding becomes a named field with the same selector,
`mapModel mapview.Model` (MS-6). A named field from another package cannot be walked as embedded,
so the guard's engine gains one rule then, not now: **a field named in a test-held set of
sub-models is walked as an embedded struct is.** The rule is proved by a planted frame-reaching
scalar in the model that `MemoKey` omits. The guard must fail on it with the rule and pass silently
without it (INST-2). This is a change to `perturbEach`'s walk after D-15 moved it, so it is that
batch's instrument change, seen failing first. **Touches CK**, which owns the test-support
package.

### Which methods re-home and which stay (`methods-MS.tsv`)

- **Re-homed onto `mapModel`: 387.** These are the 381 `Dashboard` methods in 30 `map_*.go` files
  (TS-5), plus six in root files that are the map's:
  - `timeFrom`, `timed`, `timeDraw`, `timeBelowFloor` and `timeTick` (`timing.go:50-136`), the map's
    W14 instrument. They move with `mapClock`, `seen*` and `answerSeen` to a new `map_timing.go`.
  - `applyMotion` (`motion.go:44`), which hands Reduce motion to the map.
- **Of the 387:** 316 are called only by map code; 64 are called by the root, which then calls the
  model directly; 7 keep a one-line `Dashboard` adapter.
- **The adapters, all in one new root file `modes/tty/observer_map.go`** (not `map_`-prefixed,
  because it stays at the root in 2b):
  - The five window-key handlers that `windowKeysOf` names as method expressions
    (`window_keys.go:85-96`): `handleMapKey`, `handlePropAckKey`, `handlePropChartKey`,
    `handlePropFindKey` and `handlePropFreqKey`. They stay method expressions, never closures
    (TS-8), so P10 sees no new edge.
  - `toggleMap` (split site S1). It opens or closes the window at the root, then calls the model.
  - `closeMap`, which `Router.CloseMap` calls. `Router.CloseMap` moves from `map_work.go:237` to
    `router.go`; TS-2 splits files at the receiver.
- **Two seam helpers in the same file:** `mapEnv`, which builds the `Env`, and `applyMapWant`, which
  applies a `Want`.
- **An adapter never shares its bare name with the model method it calls.** P10-01 finds recursion
  by bare name (`06_docs/code-standards.md:329`), so a `Dashboard.handleMapKey` calling
  `mapModel.handleMapKey` would read as recursion. The model's key entry is one method, `press`
  (MS-6). A-7's pilot checks the rule (CQ F-16).

**Why the root calls the model directly rather than through same-named adapters.** An adapter
with a value receiver copies the 6.5 KB `Dashboard` in and out, *and then* the 3.4 KB model. A
direct call copies only the model (MS-10).

### The split sites: where the map calls the root and then reads what changed

The map writes root state today. A write the map does not read again in the same call is
returned in `Want` and applied by the root after the call (lend and return). A root operation
whose result the map reads later in the same call is a **split site**. There the map function is
cut at the operation, the root runs the operation, and then it calls the model again. Found from
the graph (the nine arms reach no root mutator: 119 methods walked) and by reading each site:

| # | Site | Today | 2a |
|---|---|---|---|
| S1 | `toggleMap` | opens the window, then builds, zooms and draws the map (`map_pane.go:305-344`) | root `toggleMap` opens (or closes) the window, then calls `opened`, which starts with `timeFrom("open")` |
| S2 | the frequency entry's Enter | closes, then `setProp().renderMap()` (`map_prop_freq.go:239`, `:247`) | `Want{Close, Then: StepSetPropRender}` / `StepRender` |
| S3 | a find's answer | closes the finder, then recentres and settles (`map_prop_find.go:144-150`) | `Want{Close, Then: StepFoundRecentre}` |
| S4 | the acknowledgement's Enter or Esc | closes, then `askPropagation` (`map_prop_ack.go:62-64`), which reads `mapShown` and `modalPropAck` | `Want{Close, Then: StepAckAsk}` |
| S5 | the place keys in the map | `handleNav` moves the selection, then `followSelection().requestFeed()`, then the shared tail `holdWorld`, `viewMoved`, `renderMap` (`map_pane.go:1139`, `:1150-1153`) | `Want{Nav, Then: StepPlaceMoved}` |
| S6 | the window scroll keys in the acknowledgement and the chart | `handleModalNav` as the last act (`map_prop_ack.go:58`, `map_prop_chart.go:503-505`) | `Want{Nav}` (no continuation) |
| S7 | the map's own scroll | `modalScroll` written from `len(d.modalLines())` (`map_pane.go:678-682`, `:1102-1104`; `map_prop_place.go:185`) | `Env.Scroll` in, `Want.Scroll` out. The map wraps its own lines with the kit's wrap (MS-2) |
| S8–S10 | the Overlays menu and mode switch saving the UI | `uiApplyCmd()`, then `uiDirty = false` (`map_pane.go:1085-1086`; `map_prop.go:125-126`; `map_temp.go:483-485`) | `Want.Settings.Save`. The root runs `uiApplyCmd` after the call, putting its command first in the batch, so the test harness's FIFO flattening (`dashboard_test.go:151-177`) sees today's order |
| S11 | the Settings rows | `uiTouched()` / `settled()` as the last act (`map_prefs.go:221-398`; `map_prop_settings.go:85-152`; `map_settings.go:68`, `:250-279`; `map_depth.go:64`; `map_temp.go:203`, `:245`) | `SettingsWant.Touched` / `.Settled` |

`Then` is an enum of named continuations (`Step`). Each continuation is a direct method call by the
root, never a closure (R8, R9). **Confidence: medium.** The graph proves that no other map method
reached from the nine arms calls a root mutator. It sees method calls, not func values. Every
other `d.open`/`d.close`/`d.handleNav` call in map code is in the list above
(`grep` over the 31 files). Each site gets its own test in 2a's prep commit, seen failing on a
deliberately wrong order.

**Cost against D-1.**
- Principle 2: the map's state can no longer be written from outside except through `Want` and
  the API.
- Principle 4: the receivers change, and the eleven split sites change control flow without
  changing order.
- Principle 1: smaller copies (MS-10).

**Confidence: high on the shape, medium on the split-site list's completeness.**

## MS-2. The seam's value types

### `Env`: per call, by value, built by `Dashboard.mapEnv`

| Field | Type | Source today (`file:line`) | Read by the map today as |
|---|---|---|---|
| `Window` | `Window` | `d.modal` (`dashboard.go:704`), mapped by one root switch | `d.modal == modalMap` and the like, 26 uses |
| `Shown` | `bool` | `d.mapShown()` (`dashboard.go:1657`) | 4 uses |
| `Width`, `Height` | `int` | `dashboard.go:651-652` | `wordsAnchorNow` |
| `Rows` | `int` | `d.modalMax()` (`view.go:146`) | 8 uses |
| `Ceiling` | `int` | `d.mapCeiling()` (`view.go:215`) | `notesYield` |
| `Cols` | `int` | `d.mapWindowCols()` (`view.go:204`) | `mapCols` |
| `ModalWidth` | `int` | `d.modalWidth()` (`view.go:155`) | the map wrapping its own lines (S7) |
| `Opts` | `render.Opts` | `d.opts()` (`view.go:463`); it carries `Units`, `Clock`, `Frame` and `ASCII` | 14 `d.opts()` uses, plus `d.units` (6), `d.clockFmt` (11) and `d.cfg.ASCII` (8), all read from it |
| `ReduceMotion` | `bool` | `uiChoices.reduceMotion` (`dashboard_choices.go:75`) | 2 uses |
| `NoColor` | `bool` | `Config.NoColor` (`dashboard.go:126`) | `mapDepthHint` |
| `Surface` | `Surface` | `d.surface` (`dashboard.go:737`) | `mapRole`, `followRole`, `setZone` |
| `Pick` | `Pick{Snap, Recent *snapshot.Snapshot; Lookup *snapshot.LocationRef; Selected int}` | `dashboard.go:646`, `:647`, `:711`, `:653` | `selectedLocation` (`nav.go:158`), `numPriority` (`nav.go:109`), `d.snap`, `d.selected` |
| `Station` | `Station{Transmitter *snapshot.LocationRef; RadiusMi int}` | `Config.Transmitter` (`dashboard.go:314`), `Config.ServiceRadiusMi` (`:316`) | `stationPlace`, `serviceKm`, `zoneWords` |
| `Layers` | `[]MapLayer` | `Config.MapLayers` (`dashboard.go:145`); the root reads it too (`setup.go:508`) | 10 uses |
| `Keys` | `term.KeyMap` | `d.keys` (`dashboard.go:645`) | `handlePropAckKey`, `partlyCleared` |
| `Scroll` | `int` | `d.modalScroll` (`dashboard.go:722`), lent; `Want.Scroll` returns it | 15 uses |
| `Now` | `func() time.Time` | `d.now` (`dashboard.go:817`) | 30 uses (Choice C-2) |
| `Ticks` | `func(time.Duration, tea.Msg) tea.Cmd` | `d.ticks` (`dashboard.go:823`), nil in production; `d.after` (`dashboard.go:1067`) becomes the map's own, falling back to the kit's `tickEvery` | 6 uses (Choice C-2) |

**`Pick` carries the ingredients, not the answer.** `selectedLocation` builds the RECENT list,
which allocates, whenever a RECENT row is selected (`nav.go:145-172`). Computing it for every
`Env` would add an allocation to every message while such a row is selected, even with the map
shut. The nine arms build an `Env` on every message. Instead, `selectedLocation`'s logic becomes
one pure function over `Pick` in the kit. The root's `selectedLocation` calls it and so does the
map, so it allocates exactly when it does today, with one owner (K-8). **Touches CK.**

**Not in `Env`:**
- **Settings state.** It goes only to the Settings calls, as `SettingsView` (MS-4).
- **Any hook.** Hooks are held once, below.
- **Any interface.**

`Env` is about 200 bytes (an estimate from the field types; the batch's sizes probe records it).

### `Want`: what the map asks of the root

| Field | Type | Replaces (`file:line`) |
|---|---|---|
| `Open` | `Window` | `d.open(...)` (`map_prop_ack.go:48`, `map_prop_chart.go:470`, `map_prop_freq.go:210`) |
| `Close` | `bool` | `d.close()` (sites S2 to S4) |
| `Nav` | `term.Action` | `d.handleNav` (S5), `d.handleModalNav` (S6) |
| `Scroll` | `int`, with `ScrollSet bool` | the `modalScroll` writes (S7) |
| `Then` | `Step` (uint8 enum: `StepNone`, `StepRender`, `StepSetPropRender`, `StepFoundRecentre`, `StepAckAsk`, `StepPlaceMoved`) | the continuations after a stack or selection change |
| `Settings` | `SettingsWant` (MS-4) | `uiTouched`, `settled`, `uiApplyCmd`, and the `setup.uiDirty`, `setup.layerAt` and `setup.note*` writes |

Every field is a plain value. Commands come back as a separate `tea.Cmd`, as they do today.
`withCmd` (`dashboard.go:1936`), which the map calls once (`map_pane.go:343`), becomes the returned
command of `opened`.

### `Window`, the memo-key fragment, the reply predicate

- **`Window`** is a `uint8` enum: `WindowNone`, `WindowMap`, `WindowPropAck`, `WindowPropChart`,
  `WindowPropFind`, `WindowPropFreq`. The root's `modal` enum keeps its own five values and their
  `windowKeysOf` rows (TS-8). `mapEnv` maps them in one switch, and `applyMapWant` maps them back in
  one switch.
- **The memo-key fragment is `Key`**, a comparable struct holding every field
  `modalKeyFor`/`mapModalKey` take from the map today (`memo.go:385-388`, `:453-485`):
  - the choices part, set for every window: `mapsOff`, `mapDesc`, `mapScale`, `mapNearby`,
    `mapPropRadius`, `mapLayers`, `mapCost`, `mapDetail`, `mapLevel`;
  - each map window's part.

  `modalKey` (`memo.go:233`) holds one `mapKey Key` field in place of about thirty. With
  `WindowNone`, `MemoKey` returns the choices part only, so the key's projection per window is
  today's.
- **The reply predicate is `Answers(tea.Msg) bool`:** a type switch over the map's 18 unexported
  reply types (`router.go:404-413`, `mapPanicMsg` included). `observerScoped` (`router.go:396`) keeps
  its nine root types and asks `Answers` for the rest. The types stay unexported, which makes
  `router.go:390-395`'s "decidable from the type alone" a compiler rule.

### The hooks, R9 and PB-3

**Which func fields the map reads on the frame path today** (computed from the graph: every method
reachable from each entry point, and the `Config` func fields and `d.now` they read):

| Path | Func fields read (examples, `file:line`) |
|---|---|
| View, on a memo miss (`mapWindow`, the four Propagation windows, `mapModalKey`, `tickNeeded`) | `MapAreaName`, called (`map_describe.go:147-150`); `PlaceNear` (`propCentreNamed`); `MapFeed`, `MapRadar` and `PropagationUpdate` as nil checks; `d.now`, called (for example `map_night.go:91`) |
| the draw in Update (`renderMap`) | `d.now` (`map_pane.go:472`); `MapAreaName` (`map_boxes.go:1008-1010`); `MapFrame` (`map_framelog.go:66-81`); `Timed` (`timing.go:85`, `:121`); `MapProblem` on failure (`map_pane.go:1470-1476`) |
| the nine post-update arms (every message) | the same as `renderMap` (`followNight` draws), plus `PropagationUpdate` (`askPropagation`) |

So R9 read literally ("no func field is read on the frame path") is false of today's code, and no
seam can make it true without changing behaviour. This design reads R9 as RR F7 wrote it, and as
PB-3 argues it: **the seam adds no func-field read and moves none onto the frame path.** Concretely:

1. **`Env` holds no hook, and no func field except the Dashboard's own clock (`Now`) and test tick
   hook (`Ticks`).** Both are func fields today, read through the same indirect call. A static test
   over `Env`'s field types pins exactly these two (Choice C-2).
2. **The map's hooks are held once**, as one value behind one pointer, `mapHooks` (2b:
   `mapview.Hooks`). Its fields are `NewMap`, `MapFeed`, `MapRadar`, `MapTemperature`,
   `ClearMapData`, `MapDisk`, `MapCost`, `MapAreaName`, `PlaceNear`, `FindPlace`,
   `PropagationUpdate`, `SavePropagationAck`, `MapFrame`, `MapProblem` and `MapClosed`, plus the
   frame recorder's `MapFrameText` and the map's `Timed`. Today they are `Config` fields
   (`dashboard.go:71-164`, `:237`, `:244`).
   - **2a:** `Config` replaces the 15 func fields and `MapFrameText` with one field,
     `Map *mapHooks`, which app fills once (**touches AP**: `app/dashboard.go:424-466`'s assignments
     become one keyed literal, and `FindPlace` leaves the `ttyConfig` literal at `:365`).
   - `NewDashboard` gives the pointer to the model, allocating an empty one when it is nil, and sets
     `d.cfg.Map` to nil.
   - A static test fails on any read of `cfg.Map` outside `NewDashboard`, and on any write of a hook
     through anything but the model.
3. **`Timed` has two consumers:** the Router's frame and key timing (`timing.go:145-169`) and the
   map's instrument. `Config.Timed` stays for the Router. The map's copy lives in its hooks, and app
   wires both from one `lp.timings.hook()` value (`app/dashboard.go:452`). Each consumer holds its
   hook once. `Timing` (`timing.go:22`) moves to the map package, because W14 and M6 are the map's
   instrument; the Router names `mapview.Timing`.
4. **No method call that inlines today becomes a call through a func field.** The map's 421 inlined
   `Dashboard` calls (PB-3) become calls on the model or plain reads of `Env` fields. `d.opts()` (14
   sites) becomes a field read.

**Confidence: high on the inventory; the reading of R9 is the lead's to rule (C-2).**

## MS-3. Every crossing symbol classified (`crossings-MS.tsv`)

376 symbols (TS's Map II: root→map 254, of them 238 unexported; map→root 122, of them 99
unexported), each with direction, symbol, kind, declaring file, disposition and note.

| Disposition | root→map | map→root | All | Meaning |
|---|---|---|---|---|
| MOVES | 67 | 10 | 77 | moves with the map: the model's fields, the choices with their words in and out, `mapClock`, the map's timing methods, its construction values |
| API | 158 | — | 158 | reached through the map package's API (MS-6): the 15 `apply*` folded into `Answer`, the nine arms, the window lines, `MemoKey`, help and keys, the data types `app` names |
| SETTINGS | 20 | 38 | 58 | the Settings seam (MS-4): the map's Settings rows and notes; `rowMap*` and `setupRowID`; the `setupState` fields; `uiTouched`, `settled`, `uiApplyCmd`, `rowVisible` and `pickerFlashFor` |
| ENV | — | 32 | 32 | stays at the root, read through `Env` |
| CONTROLS | — | 16 | 16 | the D-147 controls, moving down to the controls package |
| HOOKS | — | 14 | 14 | `Config`'s map hooks, moving into `mapview.Hooks` |
| KIT | 4 | 3 | 7 | pure helpers moving down to the kit (`chipFace`, `tabCol`, `withTab`, `titleChrome`, `insetModalLines`, `debugProseWidth`, `refOf`) |
| WANT | — | 5 | 5 | `open`, `close`, `withCmd`, `handleNav`, `handleModalNav` |
| REMOVED | 5 | 4 | 9 | the `Dashboard` receiver, `Dashboard.cfg`, `Router.CloseMap`'s file, and `timeDraw`'s lane reads, which move with it |

**After the seam, no map→root symbol remains.** Every one of the 122 is now an `Env` field, a
`Want` field, a hook, a `SettingsView`/`SettingsWant` field, or code that moved down or across
(UNCLASSIFIED: 0). Root→map becomes calls on MS-6's exported names: the root calls **about 45 of
them** (the model's methods and `Answers`, `Width` and `New`). The data types add about 40 names
that `Config`'s hook signatures and `app` use.

**No import cycle** (2b):

| Package | Imports (watchpost) |
|---|---|
| `modes/tty` | `modes/tty/mapview`, `modes/tty/internal/controls`, the kit, `modes/tty/console`, `platform/*` |
| `modes/tty/mapview` | `modes/tty/internal/controls`, the kit, `platform/*` |
| `modes/tty/internal/controls` | `platform/render`, `platform/term` only; the closure of its 17 declarations reaches nothing outside itself (computed) |
| the kit | `platform/*`; this design adds 14 declarations whose closure is blocked by nothing (computed: `Surface`, `tickEvery`, `chipFace`, `chipGrounds`, `tabCol`, `withTab`, `titleChrome`, `spliceBox`, `tabMinLeft`, `insetModalLines`, `debugProseWidth`, `modalInset`, `panelSide`, `refOf`), plus the selection function and the modal wrap |
| `modes/tty/console` | the kit |

No edge points back into `modes/tty`, so the graph is a DAG. `lint-imports` holds it once batch 0
extends it (package map §4 item 9).

## MS-4. The Settings and controls seam (CQ F-2)

**What crosses today.**
- The D-147 controls are declared in `setup_rows.go`: `checkMark` (`:496`), `arrowChips`/
  `newArrowChips` (`:521-523`), `pickerCellW` (`:574`), `pickerFlash` and `flashNone`/`Left`/
  `Right` (`:580-583`), `pickerFlashDur` (`:602`), `itself` (`:605`) and `cycleIn` (`:618`). The
  same file has `setupMark`, `settingLabel` and `settingSupport` (`:474-484`). `radioMark` is in
  `setup_form.go:157`.
- The map uses them in its Overlays menu (`map_boxes.go:425`, `:695`, `:737`, `:758`, `:775`, `:850`) and in its Settings rows.
- The map's Settings rows switch on 21 `rowMap*` ids (`setup_rows.go:135-166`; table at `:260-291`)
  and write `setupState` (`map_settings.go:247`, `:278`; `map_prefs.go:562-564`).
- Settings (root) calls the map's rows: `setup.go`, `setup_layout.go`, `setup_ui.go`,
  `setup_notices.go` and `setup_history.go` (33, 20, 12, 4 and 3 symbols, TS-5).

**Option A (recommended): the map owns its Settings' values and words; Settings owns the window.**
- **The controls** move down to their own leaf, `modes/tty/internal/controls` (JN-2's "a home of
  their own", C-5's `internal/` because only `modes/tty` uses them). Root and map both import it;
  it imports `platform/render` and `platform/term` only. The flash timing the root and the map
  share (`markEnds`, `motion.go:67`, and `steadyMark`, `:20`) moves with it as
  `controls.MarkEnds(now time.Time, reduce bool) time.Time`. `Dashboard.markEnds` stays as the
  root's one-line caller. The notice tones (`noticeDone`, `noticeInfo`, `noticeFail`) move with
  them, as `controls.Tone`.
- **The map's choices** live in the model (MS-1). The map package owns their words: `ChoicesOf`
  at `New` from the launch words, and `Saved() Words` for `uiForSave` (`setup_ui.go:148-175`),
  whose UIPrefs map fields the root fills from it.
- **The map's Settings rows** (`map_settings.go`, `map_prefs.go`, `map_prop_settings.go`,
  `map_depth.go`, `map_playback.go`, and `map_temp.go`'s two toggles) move with the map. They are
  reached through `SettingLines`, `SettingArrow`, `SettingStep`, `SettingClear`, `SettingNotes`,
  `RefreshCost` and `PartlyCleared` (MS-6).
- **Settings keeps** the ordered row ids ("the list's order is the focus order", `docs/extending.md:186`), the tabs,
  the window, focus, scroll and save.
  - The map declares its own `Row` enum (21 values).
  - `setupRow` (`setup_rows.go:207`) gains one column, `mapRow mapview.Row` (zero for non-map
    rows), so the root-to-map mapping is in one table.
  - `SettingsView{Focus Row; LayerAt int; Flash controls.Flash; FlashEnd time.Time; Visible RowSet}`
    goes in, and `SettingsWant{Touched, Settled, Save bool; LayerAt int; Note string; NoteRow Row; Tone controls.Tone}`
    comes out.
- **Import direction:** root → map → controls; root → controls. A map Setting (M4's
  boundary-crossing task) touches:
  - the map package: the choice field, its words, its row lines and arrow, and its `Row` value;
  - the root: one id in `setup_rows.go`'s list, its table row and its `mapRow`;
  - `app`: the launch word.

**Option B: Settings keeps every row; the map exports its choices' vocabulary.**
- The choices stay in `Dashboard`, and the map reads them from `Env`.
- **Rejected on evidence:** the map writes three of them from inside its own window and reads them
  in the same call (`withChoice`, `map_boxes.go:918-926`, then `refreshMapCost().requestFeed()`;
  `menuApplied`, `map_pane.go:1077-1088`). Each such write would become a split site. Two more need
  `refreshMapCost` to run between a map write and a map read.
- TS measured that keeping Settings' map files at the root barely changes the crossings: 253 and
  123 (TS-5).

**Option C: the controls go into the kit (CK's package) instead of a leaf of their own.**
- Fewer packages.
- The kit is sized as the console's closure, "nothing speculative" (`lens-TS-tty-seams.md:180`).
  The console uses none of the controls (grep: no `broadcaster*.go` or `request.go` code names one). A kit that grows by a second, unrelated responsibility is the `util` shape A-14 and GP-5
  warn against.

**Recommendation: A.** **Strongest counter-argument:** A adds about ten exported Settings entry
points and a `Row` enum that mirrors 21 root ids, so a map Setting is declared in two packages.
B would keep the whole of Settings in one package, at the price of five split sites inside the
map's own menu.

## MS-5. The post-update steps

Today's chain, `dashboard.go:1156-1168` (the package map cites `:1156-1171`), runs on every
message:
1. `dispatch`. It wraps `route` in `wordsPlaceNow`/`followReader` (`:1195-1198`), the map's D-211
   reader place.
2. `resume` (`:1694`).
3. The nine map arms, in this order:
   1. `mapPanicked` (`map_work.go:153`)
   2. `armMapTick` (`map_work.go:188`)
   3. `followNight` (`map_night.go:72`)
   4. `armPropChart` (`map_prop_chart.go:109`)
   5. `armPropReach` (`map_prop_freq.go:84`)
   6. `armPropAhead` (`map_prop_ahead.go:47`)
   7. `armPropCentre` (`map_prop_centre.go:76`)
   8. `armPropOptions` (`map_prop_update.go:105`)
   9. `armPropRefresh` (`map_prop_update.go:116`)
4. `armTick` (`:1129`), with one `tea.Batch` of the commands.

**Design.**
- `Dashboard.update` stays at the root, with the same eleven steps in the same order. Each map arm
  becomes a call on `d.mapModel` with one `Env`, built once after `resume`.
- Sharing one `Env` across the nine is equivalent because none of them, nor any of the 119 methods
  they reach, calls a root mutator or writes a root field (computed from the graph; the field
  writes the map makes to `modalScroll` and `setup` are all in functions the arms do not reach).
- The commands keep their positions in the batch.
- The arms keep their names, so their mutants' anchors and their killers keep theirs; in 2b they
  are capitalised (MS-6).
- This is GP-3's rule ("the parent calls it after every dispatch, in today's position") and R4's
  literal text: the steps stay at the root.

**The tests that hold it.**
1. **An order test** (batch 0, A-12's "test-held list of the post-update steps"). It parses
   `Dashboard.update`'s body with `go/ast` and compares the sequence of selector calls with a list
   held in the test: `dispatch`, `resume`, the nine arms (by name), `armTick`. It is seen failing
   on a swapped pair. 2b's first commit changes only the names in the list.
2. **A behaviour test** (batch 0; none exists: the arms are tested by direct calls, for example
   `map_prop_honesty_test.go:161`, `map_prop_lower_test.go:76`). A Settings save message, which
   is not the map's, still makes `armPropOptions` re-ask a snapshot made with other options. This
   is GP-3's silent regression, pinned through `Dashboard.Update`.

**Option set aside:** one `Settle(Env)` running the nine inside the map (GP-3's example, TS's
`After`). It costs one export instead of nine. It moves the order out of the root, against
package map §2 ("do not move") and RR F7. **Confidence: high.**

## MS-6. 2b's exported API

**The package:** `modes/tty/mapview` (MS-11). It is importable, not `internal/`, because `app`
names its data types (C-5). Every name is used by production code outside the package. None is
for a test (D-14).

**Types (12):**

| Name | Why |
|---|---|
| `Model` | the map's state (2a's `mapModel`) |
| `Hooks` | the map's hooks, held once (MS-2); `Config.Map *mapview.Hooks` |
| `Env` | per-call inputs |
| `Want` | what the map asks of the root |
| `Step` | `Want.Then`'s continuations |
| `Window` | the map's five windows |
| `Key` | the memo-key fragment |
| `Pick` | `Env`'s selection ingredients |
| `Station` | `Env`'s transmitter and radius |
| `Words` | the choices as the file has them, in (`New`) and out (`Saved`) |
| `Row` | Settings' map rows |
| `RowSet`, `SettingsView`, `SettingsWant` | the Settings seam |
| `ReaderPlace` | `WordsPlace`'s result (unexported fields), handed back to `FollowReader` |
| `HelpRow` | `windowKeysOf`'s and Help's rows (today `mapHelpRow`) |

(`RowSet`, `SettingsView` and `SettingsWant` share a row; listed with `ReaderPlace` and `HelpRow`,
the total is 17 type names.)

**Functions (4):**
- `func New(h *Hooks, w Words, overrides term.KeyMap) Model`: `NewDashboard`'s builder.
- `func Answers(msg tea.Msg) bool`: `observerScoped`.
- `func Width(w Window) int`: `modalWidth`'s widths for the four Propagation windows.
- `func ChoicesOf(w Words) Words`: normalises the launch words. *Folded into `New` if the pilot
  finds `New` is its only caller.*

**Methods on `Model` (about 40), each with the root caller it serves:**

| Signature | Root caller today |
|---|---|
| `Answer(msg tea.Msg, e Env) (Model, tea.Cmd, Want, bool)` | `routeMap` (`dashboard.go:1262`), the 17 messages |
| `MapPanicked(e Env) Model`, `ArmMapTick(e Env) (Model, tea.Cmd)`, `FollowNight(e Env) Model`, `ArmPropChart`, `ArmPropReach`, `ArmPropAhead`, `ArmPropCentre`, `ArmPropOptions`, `ArmPropRefresh` (each `(e Env) (Model, tea.Cmd)`) | `update` (MS-5): nine |
| `WordsPlace(msg tea.Msg, e Env) (ReaderPlace, bool)`, `FollowReader(msg tea.Msg, p ReaderPlace, keep bool, e Env) (Model, Want)` | `dispatch` |
| `Resized(e Env) (Model, tea.Cmd)`, `SnapshotLanded(e Env) (Model, tea.Cmd)`, `Resumed(e Env) (Model, tea.Cmd)` | `route`'s size and snapshot arms (`dashboard.go:1204-1219`), `resume` |
| `Opened(e Env) (Model, tea.Cmd)`, `Closed(e Env) (Model, tea.Cmd)`, `Released() Model` | `toggleMap` (S1), `tellMapClosed` (`dashboard.go:1661`), `closeMap` |
| `Press(w Window, k tea.KeyPressMsg, e Env) (Model, tea.Cmd, Want, bool)`, `Continue(s Step, e Env) (Model, tea.Cmd)` | the five key adapters, `applyMapWant` |
| `Lines(w Window, o render.Opts, e Env) []string`, `Title(e Env) string`, `Tab(e Env) string` | `mapWindow` (`view.go:432`), `modalLines` (`view.go:228`), `renderModal` |
| `MemoKey(w Window, e Env) Key` | `modalKeyFor` (`memo.go:368`) |
| `NeedsTick(e Env) bool`, `Tick(now time.Time) Model`, `ClearMarks() Model`, `MotionChanged(e Env) Model` | `tickNeeded` (`dashboard.go:1081`), `applyTick` (`:1394`), `keyClearsMarks`, `cycleReduceMotion` (`motion.go:33`) |
| `Notes(e Env) []string`, `Status(e Env) string`, `Built() bool`, `Cursor(e Env) bool` | `status_sources.go`, `setup.commitToModel`, `cursor.go` |
| `Off() bool`, `Saved() Words`, `Role(e Env) Role`, `PlaceKeys(e Env) bool` | `help_about.go`, `setup_ui.uiForSave`, `setup.go` |
| `Keys() term.KeyMap`, `ChartKeys() term.KeyMap`, `HelpRows() []HelpRow`, `HelpNotes(e Env) []string` | `window_keys.go:96`, `help_about.go` |
| `SettingLines(v SettingsView, o render.Opts, e Env) []string`, `SettingArrow(r Row, forward bool, v SettingsView, e Env) (Model, SettingsWant, bool)`, `SettingStep(down bool, v SettingsView, e Env) (Model, SettingsWant, bool)`, `SettingClear(v SettingsView, e Env) (Model, SettingsWant, tea.Cmd)`, `SettingNotes(e Env) []string`, `SettingWidths(o render.Opts) (row, picker int)`, `RefreshCost(e Env) Model`, `PartlyCleared(err error, e Env) (Model, SettingsWant)`, `SelectionMoved(e Env) Model`, `ZoneMoved(e Env) Model` | Settings (MS-4), `setup.commitToModel` |

Fields like `Role`'s are resolved in the pilot. **The count is 17 types, 4 functions and about 46
methods, about 67 new names**, inside TS's estimate of "30-50 entry points" once the 17 types are
set aside. Every method name is checked against P10-01's bare-name rule.

**What `app` imports from it** (`cross-package-edges.tsv`; 39 top-level names, 123 with fields):
- **Types:** `MapAsk`, `MapCost`, `MapLayer`, `FeedPart`, `MapFeed`, `MapRadar`, `MapTemperature`,
  `MapQuota`, `ForecastStep`, `TimedOverlay`, `MapCleared`, `MapFrame`, `PropagationResult`,
  `MapView` (an existing alias of `geo.Box`, `map_boxes.go:1020`), and `Timing`.
- **Functions:** `UVCitiesByCount`, `AlertCategoryKey`, `AlertCategorySwitch` and `ForecastSteps`.
- **Constants:**
  - layers: `AlertLayer`, `QuakeLayer`, `RadarLayer`, `AirLayer`, `BuoyLayer`, `FeelsLayer`,
    `FireLayer`, `RainLayer`, `TemperatureLayer`, `TideLayer`, `UVLayer`, `WaveLayer`, `WindLayer`;
  - parts and fire modes: `FeedAlerts`, `FeedLayers`, `FireHotspots`, `FireNamed`.
- **Three are used only by `app`'s tests:**
  - `ChipKnown` has no production user anywhere (`map_parts.go:50`);
  - `SameOverlay` (`map_pane.go:1623`) and `FireAll` (`map_prefs.go:480`) are used in production
    only inside the map.

  After the move each counts against M5's "zero exported names whose only users outside are
  tests". They are test-only exports today, which the split makes visible. **Touches AP; a ruling
  is needed** (C-6).
- The move commit changes the qualifier `tty.` to `mapview.` in the 14 `app` files that name
  them, and nothing else.

**`app`'s hook assignments** (`app/dashboard.go:424-466`, AP's lane):
- In 2a they become one keyed literal assigned to `c.Map`, which includes `FindPlace`, moved from
  the `ttyConfig` literal at `:365` (it needs `resolver` and `resolverErr` passed to `mapConfig`).
- `c.Timed` stays and is assigned the same instrument as `c.Map.Timed` (`:452`).
- In 2b the literal's type becomes `mapview.Hooks`, a qualifier change only.

**Name stutter.** `mapview.MapFeed` stutters (GP P-e). Renaming means a prep commit across 14
`app` files and the map's tests, because R3 allows no rename in a move commit. Recommendation: keep
the names in 0.19.6 and record the renames as a follow-up (C-7). **Confidence: medium on the exact
method list, which the pilot fixes; high on the types and on `app`'s list.**

## MS-7. Tests under D-14 (`tests-MS.tsv`, `tests-fn-MS.tsv`)

### After 2a

One package, so nothing straddles. What changes, all identifiers:

| What | Count | Change |
|---|---|---|
| Call references to the 154 re-homed methods tests call | 798 in 68 test files | the call goes to `d.mapModel` with `d.mapEnv()`, and the result is folded back. A setup or act line, never an assertion's value |
| Hook references (`Config{MapFeed: …}` and `d.cfg.MapFeed = …`) | 173 in 47 files (41 writes after construction) | `Config.Map`. The writes after construction go to the model's hooks. **Audit:** the hooks are now shared by every copy of the model (one pointer). A test that copies a model and then writes a hook would see the write in both copies. Each of the 41 is checked in the prep commit. UNVERIFIED that none does |
| Field paths in assertions (`d.mapPane.x`, `d.mapScale`, `d.mapCost`) | 0 | promotion through the embedded model |

### After 2b

The rule is applied per test function. A test's reach includes every test helper it calls,
transitively, which is why so few tests are "setup only".

- **C1, setup only:** every map-internal symbol it reaches is construction through exported
  `Hooks`/`Config`.
- **C2, the same value through production API:** every map-internal symbol it reaches is served
  value for value by an exported production entry point (the table in `classify_tests.py`, for
  example `mapBodyLines` → `Lines(WindowMap, …)`, `mapsOff` → `Off()`).
- **C3, moves with the code:** not C1 or C2, and its root reach is only what becomes `Env` or the
  path to the model (`opts`, `now`, `modal`, size, selection, `cfg`, `open`/`close`, `update`, the
  `modal*` values). It moves into `mapview`. Its setup is rebuilt on `New` and an `Env` fixture,
  and its assertions' identifiers are re-pointed (`d.mapPane.x` becomes `m.mapPane.x`). D-9
  explicitly allows this.
- **C4, cannot be served without a test-only export:** it reaches map internals no production API
  serves *and* root internals or root output no map-package test can stand in for: Settings' window
  and rows, the frame the Dashboard or the Router draws (`View`), Help, Status, the memo slot,
  the window stack, the tick predicate.

| Class | Test functions | Files (worst class) | Map-internal references |
|---|---|---|---|
| C1 setup only | 0 | 0 | — |
| C2 same value through production API | 0 | 0 | — |
| C3 moves with the code | **288** | 34 (another 33 files hold both C3 and C4 tests) | — |
| C4 cannot be served | **120** | **52** | — |
| All reaching the map's unexported code | 408 | 86 | 2,518 references from 82 files, counted directly |

**C4 by what keeps it at the root:**
- Settings window and rows: 53
- the frame (`Dashboard.View`, `renderModal`): 36
- Help: 7
- Router: 3
- tick predicate: 3
- window stack: 3
- Status: 2
- other: 13

The heaviest files:
- `map_prefs_test.go`: 9
- `cursor_test.go`: 8
- `map_radar_test.go`: 6
- `map_describe_test.go` and `map_window_test.go`: 5 each
- `map_prop_honesty_test.go`, `map_station_test.go`, `map_settings_test.go`, `map_floor_test.go`,
  `map_uat1_test.go`, `window_keys_test.go` and `window_stack_test.go`: 4 each

**How exact this is.** The class rule is exact and reproducible (`tests_fn_ms.py`). It is a rule
over the graph, which cannot tell a setup line from an assertion. Some C4 tests reach a map internal
only to force a state, for example calling `renderMap` before reading `View`. Reading each of the
120 may move those to "setup through production API": the same state reached by a message such as
`tea.WindowSizeMsg`. So **120 is the upper bound, by rule, on today's code**. D-9 asks for it to be
re-measured on 2a's code. Run that day, the same script gives the figure the HUM LEAD is asked
about, test by test. C3's 288 are D-14's heaviest change set: each is proved by the scoped sweep's
same killers and the outcome inventory (A-12). RR's 586 assertion references and TS's 122 files
counted differently (RR: `mapPane.` in assertions; TS: any straddle, console and kit included).
The figures do not contradict each other. **Confidence: high on the method, medium on the split
between C3 and C4.**

## MS-8. Mutants (`mutants-MS.tsv`)

Anchors are exact text, `assert old in s`, in each `06_docs/mutants/*.py`, with the target as
`pathlib.Path("modes/tty/…")` (for example `m122_the_propagation_mode_kept_across_opens.py`).
`run.sh` applies the file and checks it changed something (`06_docs/mutants/run.sh:78-83`).

| | Count |
|---|---|
| Mutants whose target is a `map_*.go` file | 289, one anchor each (A-6) |
| **2a's re-homing changes the anchor text** (it sits in a re-homed method and contains the receiver `d` or `Dashboard`) | **146** |
| 2b's first commit (capitalising the nine arms, `mapRole`, `mapHelpRows`, `propRadiusKm` and the like) changes the text | 7 (m255, m260, m263, m264, m266, m269, m492) |
| 2b's move changes the path | 289 |
| Root-file mutants 2a also touches | about 6: m613 (`motion.go`, `applyMotion`), m620 (`markEnds`), m729 (`dashboard.go`, `followReader`), m172 (`propAgeShown`), m262 and m265 (`mapRole`), m640 (`cursor.go`, `mapPane`), m781 (`dashboard_choices.go`, `mapChoicesOf`). The anchor scan also matched m599, mV1 and m780 on generic words; they are not affected |

2a's 146, by file:

| File | Mutants | 2a | File | Mutants | 2a |
|---|---|---|---|---|---|
| `map_prop_update.go` | 56 | 28 | `map_prop_centre.go` | 7 | 3 |
| `map_prop_words.go` | 31 | 13 | `map_role.go` | 7 | 1 |
| `map_prop_chart.go` | 25 | 13 | `map_zone.go` | 7 | 4 |
| `map_pane.go` | 24 | 17 | `map_temp.go` | 6 | 6 |
| `map_prop_draw.go` | 23 | 7 | `map_prop_scales.go` | 6 | 0 |
| `map_prop_ahead.go` | 13 | 6 | `map_layer_words.go` | 6 | 0 |
| `map_boxes.go` | 10 | 5 | `map_prop_ack.go` | 4 | 3 |
| `map_prop_find.go` | 10 | 9 | `map_prop_origin.go` | 4 | 1 |
| `map_prop.go` | 9 | 4 | `map_prop_settings.go` | 4 | 0 |
| `map_prop_freq.go` | 9 | 5 | `map_prefs.go` | 3 | 2 |
| `map_describe.go` | 8 | 7 | `map_depth.go` | 3 | 1 |
| `map_night.go` | 7 | 6 | `map_prop_place.go`, `map_scrub.go` | 2 each | 2 each |
| | | | `map_parts.go`, `map_radar.go`, `map_playback.go` | 1 each | 0, 1, 0 |

- CQ F-19 counted 131 by grepping for `d.`. This count also takes the anchor's enclosing function
  into account, and includes anchors that use `d` without a dot (`return d`, `d, cmd :=`).
- **The receiver name.** Naming the model's receiver `d` would leave every `d.mapPane.` anchor
  intact, but it would give a fifth type the receiver `d` (JN P2-10). The design uses `m` and
  re-points the 146 in 2a's prep commit. A-11 reviews them as an old-and-new table, and the scoped
  sweep re-vets them.
- Files keep their basenames in 2b (`modes/tty/mapview/map_pane.go`), so the move re-points by path
  only. **Confidence: high.**

## MS-9. M3 projections (B-5)

| Figure | Today (`e57afeaf`) | After 2a | After 2b |
|---|---|---|---|
| The map's reach into the model, JN's scope (symbols of `dashboard.go` and `dashboard_choices.go` the map reads; JN P2-3) | **72** (reproduced from `file-edges.tsv`) | **23**, counted through `Env` and `Want` as B-5 asks: `.ASCII` (in `Opts`), `.NoColor`, `.ServiceRadiusMi`, `.Transmitter`, `.clockFmt`, `.units` (in `Opts`), `.height`, `.width`, `.keys`, `.modal`, `.modalScroll`, `.now`, `.ticks`, `.reduceMotion`, `.selected`, `.snap`, `.recent`, `.lookupRef`, `.setup`, `.surface`, `open`, `close`, `mapShown` | 23 (the move changes no reach) |
| The same, by type over every file (members of `Dashboard`, `Config`, the three choice groups and `setupState`) | **93**: 43 `Dashboard`, 21 `Config`, 20 `mapChoices`, 6 `setupState`, 3 `uiChoices` | **40**: 20 fields and 6 methods through `Env`, 7 root operations and 5 `setupState` fields through `Want`, 2 more through `SettingsView` | 40 |
| `Dashboard` methods and files | 752 in 68 | **374 declared in 38**: 752 − 387 + 9. In 2a the embedded model's unexported methods are also promoted into `Dashboard`'s method set. They are not declared on it, and none is exported. 2b's named field ends the promotion. The 9 are the 7 adapters and 2 seam helpers in `observer_map.go`; 37 files keep methods (`timing.go` loses its five). The brief's 371 counted 381 re-homed and no adapters | 374 in 38 |
| The legend-key task (JN-3): files read, packages entered | at least 5 read, 1 package of 31,346 lines, no doc route (`lens-JN-junior-navigation.md:131-172`) | unchanged, unless 2a's comment pass fixes `map_boxes.go:7-8` | **3 read** (the map package's `doc.go`, which names the four under-the-map rows with one word for the legend, JN-3; `map_scrub.go:56`; the mode's row, for example `map_temp.go:802`), **1 package of about 10,500 lines**. This holds only if 2b's `doc.go` carries that row list |

The M3 targets proposed for ratification:
- the map's reach is at most 23 (JN's scope) and at most 40 (by type);
- `Dashboard` has at most 374 methods in 38 files;
- the legend-key task reads at most 3 files in 1 package.

**Confidence: high on the method; the 40 depends on MS-4 A.**

## MS-10. Performance

| Risk on `Update` or `View` | Where | How the design avoids it |
|---|---|---|
| A boxed model (+6.5 KB a beat, PB-2) | none: `Model` is a concrete field, its methods return `Model`, and nothing implements `tea.Model` | the batch's escape list (by function and type) shows no `mapview.Model` or `Env` |
| An `Env` escaping into a command's closure | every map command closure (for example `map_pane.go:1280`) | the map copies the values it needs into locals before the closure, as today. A capture of `e` or `m` shows up as a new escape of type `Env`/`Model`, which fails the gate |
| Lost inlining (421 sites, PB-3) | the `Dashboard` methods the map inlines today: `mapMode` (63), `settled` (28), `uiTouched` (24), `mapRow`, `mapRole` (19 each), `opts` (14) | `mapMode`, `mapRow` and `mapRole` keep their bodies on the model. `opts` becomes a field read. `settled` and `uiTouched` become `Want` field writes. Inlined calls are counted per function before and after (A-12). A body pushed over the budget by the new `Env` parameter is a finding for the lead (B-3) |
| Indirect calls | hooks: `m.hooks.X`, one pointer load more than `d.cfg.X`, at the same sites | no hook in `Env`, no new indirect call (MS-2) |
| An allocation per message | `selectedLocation` computed for every `Env` | `Pick` carries ingredients. The kit's function allocates exactly when today's does |
| An allocation per message | `Answers` | a type switch on a `tea.Msg` already boxed |
| Copy size | today every internal map call copies `Dashboard` in and out (6,584 B each way, PB-2) | it copies `Model` (about 3.6 KB: `mapPane`'s 3,248 plus 128 of choices plus about 50) and `Env` (about 200 B). This is a gain, held by M1's wall-time benchstat |
| Model size | `Dashboard` 6,584 B | `Config` loses 15 func fields and a bool (about 120 B) and gains one pointer; the model gains one pointer and `ackSeen`. `Dashboard` falls by about 100 B (estimate; the sizes probe records it at 2a) |

The idle pin's bytes would not see that change. The Router boxed per message, 7,136 B
(`router.go:463`), sits in the 8,192 B size class either way. Holding the hooks by value in the
model would instead grow `Dashboard` by about 120 B. The pin would not catch that either, so it is
the sizes probe (PB-2) that holds "sizes unchanged".

**Receivers.** Value receivers on `Model`, as on `Dashboard` (GP-9: changing the receiver kind
changes aliasing). `Env` is passed by value. A pointer would save about 200 B a call, but a pointer
to a stack `Env` that any of about 400 callees leaked would move it to the heap on every call.
**Confidence: high on the mechanisms; the sizes are estimates until the probe runs.**

## MS-11. Names (A-14)

**The package:**
1. **`modes/tty/mapview`: recommended.** It names what the package provides, the map's view with
   its Propagation windows and their state. It is not a mechanism. It shadows no standard-library
   package, and `maps` is avoided. It is not a keyword. It is the name TS used.
2. `modes/tty/mapwindow`: the user's word ("the Map window"), but the package holds five windows.
3. `modes/tty/observermap`: wrong, because the Broadcaster has a map too (D-195, `map_role.go:40-55`).

**The 2a type:**
1. **`mapModel`: recommended.** It is the Bubbles idiom, and it becomes `mapview.Model` by one
   rename in 2b's first commit. The embedded field's selector, `d.mapModel`, is the same in 2a and
   2b.
2. Keep `mapPane` and widen it: GP-4's suggestion, with less churn, but it changes what a name
   that already exists means.
3. `mapState`.

**The controls package:** `modes/tty/internal/controls`, D-147's own word ("control components").

**The root seam file:** `modes/tty/observer_map.go`.

**The two new 2a files:** `map_model.go` (the type, `mapEnv`'s fields and `Want`) and
`map_timing.go`. In 2b all four are whole files.

## Choices for the lead

- **C-1, the Settings seam (MS-4).**
  - Options: A, the map owns its Settings' values and words, with the controls in a leaf;
    B, Settings keeps every row; C, A with the controls in the kit.
  - Recommendation: **A**.
  - Strongest counter-argument: a map Setting is declared in two packages, through a mirrored
    `Row` enum.
- **C-2, the clock and R9 (MS-2).**
  - Options: a, `Env` lends the Dashboard's `now` and `ticks` (func fields today, read through the
    same indirect call), and a static test pins `Env`'s func fields to exactly these two;
    b, `Env.Now` as one `time.Time` reading per call. Under b, the timing instrument's intervals
    and the frame log's stamps are read before the draw instead of after it, which is a change to
    diagnostic output (R1).
  - Recommendation: **a**, with R9 recorded as "the seam adds no func-field read on the frame path;
    the hooks are read where and as today, held once".
  - Strongest counter-argument: R9's text says no func field at all, and a reads it as "none added".
- **C-3, the post-update arms (MS-5).**
  - Options: nine exported steps called at the root; or one `Settle`.
  - Recommendation: **nine**.
  - Strongest counter-argument: nine exports for one call site.
- **C-4, holding the hooks (MS-2, MS-10).**
  - Options: one shared pointer from `Config.Map`; by value in the model (about 120 B more per
    model copy); flat on `Config` and lent through `Env` (against R9).
  - Recommendation: **the pointer**.
  - Strongest counter-argument: every copy of the model shares one `Hooks`, an aliasing change for
    the 41 test writes made after construction (audited in 2a).
- **C-5, the C4 tests (MS-7).** 120 test functions in 52 files, by rule on today's code, to be
  re-measured on 2a's code (D-9). This is the ruling D-14 reserves for the HUM LEAD. The options to
  put, once re-measured:
  - a, split each test: the map side moves (C3), and the root side asserts what the root itself
    draws through `View`, which changes those assertions;
  - b, the root reads a small set of production accessors it needs anyway (`Off`, `Saved`,
    `Lines`), where the value is the same;
  - c, a ruled exception list.
- **C-6, three test-only exports `app` has today** (`ChipKnown`, `SameOverlay`, `FireAll`, MS-6).
  - Options: unexport, with `app`'s tests changed under D-14; or a ruled exception.
  - Recommendation: unexport `ChipKnown`, which has no production user, and rule the other two with
    AP.
- **C-7, stutter (MS-6).**
  - Options: keep `mapview.MapFeed` and the like in 0.19.6; or rename in 2b's first commit.
  - Recommendation: **keep**, and file a follow-up.
  - Strongest counter-argument: GP P-e, and a stutter that ships tends to stay.
- **Names (MS-11).** A-rows: `mapview`, `mapModel`, `internal/controls`.

## Touches other agents

- **CK (kit and test support).**
  - The kit gains `Surface`, `tickEvery`, seven window-chrome helpers with their closure (14
    declarations, MS-3), the selection function over `Pick`, and the modal wrap out of
    `wrapModalAt` (`view.go:286`).
  - The controls package, if C-1's C is taken instead of A.
  - The memo guard engine's "named sub-model" rule in 2b (MS-1).
- **AP (`app`).**
  - `Config.Map` in 2a, which rewrites `app/dashboard.go:424-466`'s assignments and moves
    `FindPlace` out of `:365`.
  - `Timed` assigned twice (`:452`).
  - `tty.` to `mapview.` in 14 files in 2b's move.
  - C-6's three exports.
- **KB (the keys).**
  - The legend-key task's route after 2b (MS-9): `mapview/doc.go`, `map_scrub.go:56`, the mode's row.
  - The map Setting task, which touches the map package, one root id and table row, and `app`'s
    launch word (MS-4).
