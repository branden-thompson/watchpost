---
title: "0.19.6 DISCOVER — lens TS: the seams in modes/tty"
date: 2026-11-14
phase: DISCOVER
lens: TS
status: "FILED — findings checked against e57afeaf"
---

# Lens TS: along which seams `modes/tty` splits, and what each cut costs

## Verdict

`modes/tty` has one clean seam and one large, expensive one. The clean seam is the Broadcaster console:
its 92 methods sit on its own type, it talks to the Router through messages and a mirror, and with a
41-declaration shared "kit" under it, it leaves without a cycle and without a new interface (TS-3,
TS-4). Everything else is the Observer, and the Observer is one value type, `Dashboard`, that carries
752 of the package's 1,394 functions as methods across 68 files (TS-1). Go only allows methods in the
package that declares the type. So no Observer cluster can leave by moving files. Each one needs a
sub-model type of its own, a narrow context passed in, and intents passed back (TS-5, TS-7). The map
(31 files, 10,528 lines, 381 `Dashboard` methods) is the only Observer cluster big enough and
self-owned enough to repay that work. **Recommended: Map II** (TS-13). `modes/tty` keeps the Router
and the Observer core, `modes/tty/console` takes the Broadcaster, a map package takes the map behind
a concrete sub-model, and a small kit holds what both surfaces draw. Build it in two steps: first the
moves, then the map seam. The strongest argument against it is that the second step is a redesign,
not a move. It re-homes 381 methods across a cut that 376 symbols cross today, in both directions,
and it rewrites the regression net in the same place the behaviour changes.

## What I checked, and how

- The shared graph (`q196/graph/`: `file-edges.tsv`, `symbols.tsv`, `test-reach.tsv`,
  `cross-package-edges.tsv`), read whole for `modes/tty`.
- **A declaration-level graph of my own**, because `file-edges.tsv` is file to file. The files that
  matter most mix a `Dashboard` method, a `Broadcaster` method and free functions in one file
  (`cursor.go`, `broadcaster_detail.go`, `timing.go`, `map_work.go`). So I built a scratch
  `go/packages` tool with full type information. For every use of a package-level symbol, method or
  field, it records the enclosing top-level declaration and, for each field, its owning struct. I ran
  it twice, once over the non-test files and once with tests. The output is in the scratchpad
  (`ts/out/`, `ts/outT/`), and nothing was written to the tree.
- **The candidate maps were evaluated by script**, not by eye. Each declaration is assigned to a
  package by its receiver, its file and named exceptions. The scripts count the distinct symbols
  that cross each boundary (and how many are unexported), the import direction, cycles (mutual
  reachability), and the test files whose unexported reach would straddle two packages. A
  dependency-closure pass found the smallest shared kit that reaches nothing on `Dashboard`.
- Struct sizes come from `go/types` `SizesFor("gc","arm64")`.
- Mutant anchors come from `06_docs/mutants/*.py` (each file's first `Path("modes/tty/…")`) and
  their killers from `06_docs/mutants/verdicts/since-v0.19.5.log`.
- The routing, the memos, the guards and the Router were read in full.
- Line counts are whole-file counts from `wc -l`. Where a map splits a file, the figure is marked
  approximate.
- I did not run `go test`, the app, or any benchmark.

---

## Findings

### TS-1: `Dashboard` is a god model, and that, not file count, is what makes the Observer hard to split

- **Claim.** Every Observer cluster is written as methods on one value type that reads fields owned
  by other clusters. The Observer's internal seams are conventions inside one type, not boundaries.
- **Evidence.**
  - Size of the type:
    - `type Dashboard struct` at `modes/tty/dashboard.go:643-835`: 84 fields, including three
      embedded Settings groups at `:696-698`.
    - It is 6,584 bytes, and `mapPane` (`map_pane.go:63`) is 3,248 of them.
    - 752 of the package's 1,394 functions and methods take a `Dashboard` receiver, in 68 of the 87
      non-doc files. The Broadcaster has 92 and the Router 32.
  - Hub fields, by non-test files that use them (`symbols.tsv`):
    - the type `Dashboard`: 68;
    - `.cfg`: 45;
    - `.mapPane`: 30;
    - `.modal`: 24;
    - `.now` and `.setup`: 21 each;
    - `Dashboard.opts` (`view.go`): 14;
    - `modalMap`: 14.
  - Hub files, by in-degree from non-test files:
    - `dashboard.go`: 71 files;
    - `map_pane.go`: 35;
    - `dashboard_choices.go`: 31;
    - `setup.go`: 23;
    - `map_temp.go` and `setup_rows.go`: 22 each.
  - Field reach of each cluster's `Dashboard` methods (declaration graph):
    - the map reads 35 `Dashboard` fields, including `modal`, `setup`, `surface`, `snap`,
      `selected`, `modalScroll` and `keys`;
    - Settings reads 30, of which 20 are the map's choices;
    - the Observer core reads 83.
  - With the files grouped by prefix, every pair of Observer clusters is coupled in both directions
    (for example observer→map 113 symbols and map→observer 70; setup→observer 115 and
    observer→setup 42). There are 1,021 distinct crossing symbols in all.
- **Consequence.**
  - Principle 2: a junior reading `map_temp.go` is reading `Dashboard`, not "the map".
  - Principle 4: an Observer split is a change of state ownership, and a change of ownership can
    change behaviour.
- **Recommendation.** Treat every Observer cut as a sub-model design with its own seam (TS-5), never
  as a file move. Plan each cut from the declaration graph, not from file prefixes.
- **Confidence.** High.

### TS-2: The file names mislead about where the seams are

- **Claim.** Several files named for the console hold Observer windows, and three files hold two
  models each.
- **Evidence.**
  - `request.go` is listed with the console (`doc.go:31`), but it holds 10 `Dashboard` methods, the
    Line-Up Request window, and no `Broadcaster` method.
  - `broadcaster_detail.go` holds 6 `Broadcaster` methods and 3 `Dashboard` methods
    (`Dashboard.showCard`, `cardLines`, `cardTitleOf`).
  - `broadcaster_manage.go` holds no `Broadcaster` method, only the card window's `Dashboard` prompts
    and `cardAction`.
  - The card and request windows are Dashboard windows whose content the console owns. The code
    rules that explicitly (`dashboard.go:1556-1573`, D-56).
  - `cursor.go` holds 5 `Dashboard` and 2 `Broadcaster` methods.
  - `timing.go` holds 5 `Dashboard` and 2 `Router` methods.
  - `map_work.go:237` holds `Router.CloseMap`.
- **Consequence.** A prefix-based map would put the Observer's card, request and cursor code in the
  console package and create a cycle. Principle 2's "traceable" means the package must match the
  owner, not the name.
- **Recommendation.**
  - In any map, `request.go`, the `Dashboard` half of `broadcaster_detail.go`, all of
    `broadcaster_manage.go`, and the `Dashboard` methods in `cursor.go` stay with the Observer.
  - Files that hold two models are split at the receiver.
  - The renames are A-rows.
- **Confidence.** High.

### TS-3: The console is the one seam that is a move, not a redesign

- **Claim.** `Broadcaster` (`broadcaster.go:127`, 472 bytes, 29 fields) can leave for
  `modes/tty/console` with no new interface and no cycle, once the shared kit (TS-4) sits under it.
- **Evidence** (Map I, below; declaration graph):
  - **console→root: 0 symbols.** console→kit: 28 (24 unexported).
  - **root→console: 41 symbols, 30 unexported.**
    - 17 are fields the Router writes:
      - at construction (`router.go:296-312`): `ascii`, `version`, `fireBoldMW`, `units`,
        `clockFmt`, `area`, `areaGen`;
      - on every update (`router.go:452-461`): `gain`, `units`, `clockFmt`, `frame`,
        `statusNote`;
      - in `scopedMessage` (`router.go:560`, `:572`): `listenerMuted`.
    - 15 are methods the Router calls: `Update`, `View`, `cardDetail`, `alertDetail`,
      `indexForSlot`, `liveOffset`, `scrollQueue`, `consoleCursor`, `heldCursor`, and others.
  - The console's 66 mutants (71 in `broadcaster*.go` and `request.go`, less `request.go`'s 5) are
    all killed by `broadcaster_*_test.go` files, plus `beat_alloc_test.go`.
  - Under Map I, 40 test files straddle the new boundary, with 114 unexported references outside
    their home package.
- **Consequence.**
  - Principle 1: neutral. The Router already calls both surfaces through their concrete types, and
    `Router.View` refuses the interface "because it would box the whole model on every frame"
    (`router.go:502-505`; `surfaceView` at `:844-852`).
  - Principle 2: improved. The console's state becomes unreachable from the Observer by
    construction, which is what D-56 and the "HANDED, NEVER REACHED FOR" comment
    (`dashboard.go:739-742`) already state in prose.
- **Recommendation.**
  - Move the `Broadcaster` declarations.
  - Replace the Router's 17 field writes with one exported value type the console takes whole,
    applied in one place. Its shape: `func (b Broadcaster) Mirrored(m console.Mirror) Broadcaster`,
    where `Mirror` carries gain, units, clock, Reduce motion, listener mute and status note. Make the
    construction-time ones `console.New(console.Opening{…}) Broadcaster`.
  - Export the Router's 15 method calls under their current meaning.
- **Confidence.** High.

### TS-4: A 41-declaration kit breaks the only cycle the console cut would create

- **Claim.** Without a shared package the console cut forms a cycle, because the console draws the
  Observer's masthead, row helpers, tick and memo counters. The closure of what the console needs is
  small, and none of it reaches `Dashboard`.
- **Evidence.**
  - console→root (before the kit): 27 symbols, 23 unexported:
    - `mastheadSlot`, `mastheadKeyOf`, `mastheadStamp`, `mastheadTitle`;
    - `memoStats`, `rowLoading`, `weatherRow`, `agoWords`, `apiSummaryOf`;
    - `tick`, `tickMsg`, `armShimmer`;
    - `cursorSlot`, `surfaceMark`, `centerText`, `levelControl`, `testEventMark`,
      `fireBoldDefaultMW`;
    - `SnapshotMsg` and `RecentSnapshotMsg` (`dashboard.go`).
  - Its dependency closure is 41 declarations. That adds `mastheadKey`, `cursorLook`,
    `providerCounts`, `plural`, `mmdd`, `trend`, `tickEvery`, `staleAfter` and so on. The closure
    stopped at no `Dashboard`, `Router` or `Broadcaster` symbol: there are zero blocked edges.
  - The console's detail body also calls `indentBody` (`broadcaster_detail.go`), which joins the
    kit too.
- **Consequence.** The kit is the console's prerequisite, and the masthead memo that both surfaces
  share (`masthead.go:74`, `:102`; its guard `masthead_memo_test.go:53`) gets one owner by
  construction.
- **Recommendation.**
  - Create the kit with exactly this closure and nothing speculative.
  - `SnapshotMsg` and `RecentSnapshotMsg` are app-facing, so whether `app` imports them from the kit
    or the root aliases them is AS's call.
  - The name is an A-row; candidates are `modes/tty/frame` and `modes/tty/kit`.
- **Confidence.** High.

### TS-5: The map is the Observer's one cluster worth cutting, and the cut needs a new seam

- **Claim.** The map and Propagation files are the biggest cluster, at 31 files, 10,528 lines and
  381 `Dashboard` methods (half of them). Most of the state they touch is their own. Their coupling
  to the rest of the Observer is wide but definable: a context in, intents out, and a memo-key
  fragment.
- **Evidence** (Map II):
  - **root→map: 254 symbols, 238 unexported.** By root file:
    - `dashboard.go`: 81, the `route` dispatch of 15 `apply*` messages and the 9 post-Update arms
      at `dashboard.go:1156-1169`;
    - `memo.go`: 48, the modal memo key (`memo.go:385-391`, `mapModalKey` at `:453`);
    - `dashboard_choices.go`: 38;
    - `setup_ui.go`: 33, and `setup_notices.go`: 20 (Settings saving and noting the map's
      choices);
    - `timing.go`: 20;
    - `router.go`: 18, the map's reply types in `observerScoped`, `router.go:396-417`;
    - `view.go`: 16;
    - `help_about.go`: 12;
    - `window_keys.go`: 9.
  - **map→root: 122 symbols, 99 unexported.**
    - 44 from `dashboard.go`, mostly ~25 map hooks on `Config`;
    - 37 from `setup_rows.go`: the `rowMap*` row ids and the picker helpers;
    - the modal stack: `open`, `close`, `modalMap`, `modalProp*`;
    - layout: `modalMax`, `modalLines`, `mapCeiling`, `mapWindowCols`;
    - `opts`, `now`, nav (`selectedLocation`, `handleNav`), motion and the timing hooks.
  - The two directions form a cycle as a raw move. Keeping Settings' map files
    (`map_settings.go`, `map_prefs.go`, `map_prop_settings.go`) in the root barely changes it
    (253 and 123). Keeping `mapChoices` with Settings does not change it either (234 and 142).
  - The map's own state is most of the `Dashboard` bytes: `mapPane` 3,248 of 6,584, plus
    `mapChoices` 128, plus its keymaps, cost and close counter.
  - Tests: under Map II, 122 test files straddle, with 512 unexported references outside home.
    Under Map I it is 40 and 114. The heaviest are `map_prop_honesty_test.go` (67 reaches,
    26 outside), `map_prop_chart_test.go`, `map_station_test.go` and `map_radar_test.go`.
- **Consequence.** Principle 2 gains the most here, since this is the area the last four releases
  grew. Principle 4 carries the most risk here (TS-9).
- **Recommendation.** Cut it behind a concrete sub-model (shape below), never by moving files with
  their `Dashboard` receivers. The map's five windows stay in the Dashboard's one `modal` enum
  (TS-8). The map package exports a reply predicate, so the message types stay unexported and the
  "decidable from the type alone" argument at `router.go:390-395` gets stronger.
- **Confidence.** High on the figures. Medium on the API width after the seam, which I estimate at
  30-50 exported entry points but did not measure.

### TS-6: The Router alone at the root (Map III) costs more than it buys

- **Claim.** Moving the Observer out to `modes/tty/observer` and leaving only the Router at the
  root exports the Router's deliberate intimacy with the Observer, and does not shrink the largest
  package.
- **Evidence.**
  - root→observer: 53-73 symbols, 37-57 unexported:
    - the card window's fields `cardAct`, `cardErr`, `cardID`, `cardMoveTo` and
      `modal == modalCard` (`router.go:604`);
    - `liveOffset`, `surface`, `tickerMuted`;
    - the four Settings mirrors (`router.go:447-457`);
    - `overlay` and `ModalOpen` (`router.go:509-511`, `:646`);
    - 25 unexported reply types.
  - observer→root: 37 (Surface, the console's `act*` constants and `broadcasterKeyMap`, read by
    Help and `consoleKeyMap`; `Timing`), all of which must move down to the kit.
  - `app` uses 161 exported names from `dashboard.go` (`cross-package-edges.tsv`), and every one
    changes import path.
  - The observer package would still be about 25,900 lines.
  - Tests: 50 straddle, with 181-192 references outside home.
- **Consequence.** The cost lands on principle 2 (more exported surface, and two places to read one
  routing rule) and on AS, with no gain to sweep time or package size.
- **Recommendation.** Do not choose it. The Router and the Observer core stay in one package.
- **Confidence.** High.

### TS-7: No other Observer cluster repays a cut (Map IV, by window, is dismissed)

- **Claim.** Every other Observer cluster is bidirectionally coupled to the core, and each is small.
- **Evidence.** Each cluster was cut alone from the Observer, with its own fields moved with it:

  | Cluster | Files | Lines | `Dashboard` methods | cluster→core (unexported) | core→cluster (unexported) |
  |---|---|---|---|---|---|
  | map (no Propagation) | 17 | 7,245 | 265 | 166 (148) | 272 (255) |
  | Propagation | 14 | 3,283 | 116 | 150 (140) | 107 (106) |
  | Settings | 11 | 3,840 | 107 | 195 (105) | 89 (85) |
  | Status, Help, About, ctrl+d | 5 | 1,971 | 39 | 93 (61) | 36 (32) |
  | Details, lookup, locate | 8 | 1,788 | 21 | 45 (37) | 44 (44) |
  | radio panel, relay fault | 2 | 909 | 32 | 42 (32) | 46 (45) |
  | ticker, alerts | 2 | 829 | 16 | 16 (15) | 20 (15) |
  | severe | 1 | 537 | 18 | 20 (16) | 26 (18) |

  Settings is a hub in its own right. It draws and saves every area's rows from one ordered id list
  (`setup_rows.go`; "the list's order is the focus order", `docs/extending.md:182`). Cutting it out
  would need a cross-package row registry.
- **Consequence.** Map IV would re-home about 750 methods and multiply the TS-5 seam eight times.
  That is a principle 4 risk with diminishing principle 2 return. Propagation and the map are coupled
  in both directions (map→Propagation 59, Propagation→map 102), so they cut together or not at all.
- **Recommendation.**
  - Keep Settings, the windows, the radio panel, the ticker and severe in the root with the Observer
    core.
  - Revisit a cut only when a cluster grows a state of its own the size of the map's.
- **Confidence.** High.

### TS-8: Key routing and the modal control priority must stay in one package

- **Claim.** The order of who owns a key is written once, across the Router and the Dashboard's
  window table. Splitting it would make the order a cross-package convention.
- **Evidence.**
  - `Router.update` handles scope first, then keys (`router.go:469-489`).
  - `routeKey` is "one ordered rule", kept in one function on purpose (`router.go:581-590`). Its
    order is:
    1. the card window's controls above the keymap (`:604`);
    2. the swap through `canSwap` (`:611-622`);
    3. slot digits below the keymap (`:636-640`);
    4. "the window on top owns the keys" last (`:646-651`).
  - On the Observer side, `handleKey` asks `routeWindowKey` first (`dashboard.go:1440-1443`).
  - `windowKeysOf` (`window_keys.go:59-101`) declares each window's claim (`claimAll`,
    `claimBound`, `claimActions`) against the single `modal` value (`dashboard.go:704`,
    enum at `:1540-1587`), and the stack is `open` and `openWindow` (`dashboard.go:1597-1622`).
  - Inside the map, the open box owns the arrows, space and esc (`map_boxes.go:884-887`, D-61), and
    the chart owns its up and down (`map_prop_chart.go:31-38`).
  - The table holds method expressions on purpose, because a closure creates a false P10 recursion
    edge (`window_keys.go:61-63`).
- **Consequence.** Keeping the Router, the enum, the stack and the table together keeps the D-61 and
  D-62 rules where they are, and that is the case in Maps I and II. In Map II the map's windows keep
  their enum values and table rows. Their handlers become one-line `Dashboard` adapters into the map
  model, written as method expressions, never closures, so P10 sees no new edges. The map's internal
  box priority moves whole inside the map's key handler.
- **Recommendation.** Never move `routeKey`, `windowKeysOf`, the `modal` enum or the stack apart. A
  window's "escape always closes" stays declared in its row and is held by `window_keys_test.go`, as
  it is now.
- **Confidence.** High.

### TS-9: The render memos and their completeness guards constrain the map seam

- **Claim.** The modal memo is keyed across every window, the map's included. Its guard walks the
  `Dashboard` by reflection to a fixed depth, and the same walker serves both surfaces. All three
  facts constrain any split.
- **Evidence.**
  - `modalKey` (`memo.go:228-366`) carries the map's fields, and `modalKeyFor` adds `mapModalKey`
    for the four map windows (`memo.go:368-391`, `:453`).
  - The guard perturbs every field and fails when "the frame differs and the key does not"
    (`memo_completeness_test.go:38-80`).
  - It walks embedded structs as the model itself (`:155-171`). It goes **one level down** into a
    named struct (`:179-189`), and at that level only scalar kinds (`walkNested`, `:193-198`).
    `mapPane`'s 111 fields are covered today because `mapPane` is one level down.
  - The same generic engine, `perturbEach` (`memo_completeness_test.go:115-124`), serves the
    console's guard (`broadcaster_memo_completeness_test.go:85`) and the masthead guard
    (`masthead_memo_test.go:100`). Its own comment rules out a second copy (`:115-120`).
- **Consequence.**
  - Principle 4: if the map's state became a named field holding a struct that holds `mapPane`, its
    fields would sit two levels down. The guard would stop perturbing them and keep reporting green.
    That is F-30's frozen-frame failure, with the guard's own coverage silently reduced.
  - Separately, a `console` package cannot reach a `_test.go` helper in `modes/tty`.
- **Recommendation.**
  - The map's sub-model is either embedded in `Dashboard` or flat (its fields one level down), or
    the walk is made recursive and proven by a planted two-deep field.
  - The map exports one comparable key fragment, `func (m Model) MemoKey(w Window) Key`, which
    `mapModalKey` folds in.
  - `perturbEach` moves to one importable test-support package, as `platform/declset` already is
    (RX and TG decide where), before the console moves.
- **Confidence.** High.

### TS-10: The shared deck's surface in `modes/tty` stays whole in the Router

- **Claim.** In this package the one deck is reached through the Router's station control, the
  Router's mirror, and one `Config` of hooks. A split must keep each as a single carrier.
- **Evidence.**
  - The control and the swap:
    - `Station` (`router.go:137-151`) and `StationControlMsg` (`router.go:124`), held by the
      Router "because the router is where the key is looked up" (`router.go:547-553`);
    - `canSwap`, "THE ONE PLACE THE D-1 PRECONDITION IS CHECKED" (`router.go:895-905`);
    - `toggleStation`, which reads the Director's power (`router.go:926-933`).
  - The mirror and the hooks:
    - the gain mirror `out.broadcaster.gain = out.observer.radioVolume` (`router.go:452`);
    - the bed hooks `onSurface`, `relays` and `playBed`, which come from the Observer's `Config`
      (`router.go:275-285`, `:313-314`).
  - The console holds no flag of its own for the air ("draws what it is told", `router.go:146-148`).
- **Consequence.**
  - In Maps I and II the station control, `canSwap` and the mirror stay in the root.
  - The console keeps asking through `Station`, which is already an interface. Its methods are
    called on key presses, never per frame.
  - No package gets a `Config` of its own for the deck's hooks.
  - The deck is not split, and no map here touches it.
- **Recommendation.**
  - `console` exports what the Router mirrors, never a setter for power or air.
  - The Router stays the one reader of `Station`.
  - RM decides any D-2 change.
- **Confidence.** High.

### TS-11: The seams need not cost performance, if they stay concrete

- **Claim.** None of the recommended seams needs interface dispatch or boxing on a per-frame path.
- **Evidence.**
  - Frame path:
    - the frame is `Router.View` → `Dashboard.View` or `Broadcaster.View`, through concrete types
      (`router.go:502-523`);
    - the map draws in Update, and View reads lines already drawn (`docs/architecture.md:85`,
      "draws in Update");
    - the modal memo key is a comparable struct (`memo.go:228`).
  - Copy sizes:
    - every `Dashboard` method has a value receiver and returns a `Dashboard` (6,584 bytes);
    - a map method on a `mapview.Model` copies the map's state (about 3.4 KB) plus a context value,
      not the whole `Dashboard`.
  - `tea.Msg` already arrives as an interface, so the reply predicates (`Answers(msg) bool`) are
    type switches on a value that is already boxed.
- **Consequence.** Principle 1 holds by construction if the context is a plain value, the map
  model is a concrete field, and the reply predicates are type switches. A per-area Settings
  interface would add an indirect call per row per Settings render. Settings is memoised, so that
  is small but nonzero, and Map II does not need it.
- **Recommendation.**
  - No interface on the frame path and no `tea.Model` boxing of sub-models (the 0.19.5 PF-1
    precedent).
  - The map's Settings rows are reached through concrete exported functions.
  - Whether the smaller copies show up as a gain is for PB to measure, not for this lens to claim.
- **Confidence.** Medium: this is reasoning from the code, not a measurement.

### TS-12: Guards that read "the package" change meaning when files move

- **Claim.** Several in-package guards derive their set from the package directory, so a move
  shrinks what they see without failing.
- **Evidence.**
  - `TestEveryWindowReplyIsRoutedBackToTheWindow` reads `declset.Files(".")` for every unexported
    `*Msg` (`router_lookup_test.go:131-170`). Under Map II the map's 15 reply types leave its view.
  - Other tests read the directory the same way: `map_mode_test.go:164`, `router_swap_test.go:73`
    (`declset.Files`) and `main_test.go:34` (`os.ReadDir`).
  - `TestEveryFileIsInItsPackagesMap` (`cmd/watchpost/docs_test.go:347-351`) holds
    `modes/tty/doc.go`'s file table.
- **Consequence.** A green guard after a move could be measuring one package of three.
- **Recommendation.** Each such guard names the packages it covers, and fails on zero found, which
  the reply guard already does at `:166-168`. Inventory is TG's.
- **Confidence.** High on the reply guard; the other three were located but not read in full.

### TS-13: Recommendation, Map II, built in two steps

The three maps, side by side (lines are whole-file and approximate where a file splits):

| | Map I: console out | **Map II: console and map out** | Map III: Router alone |
|---|---|---|---|
| `modes/tty` | Router, Observer: ~27,100 | Router, Observer core, Settings, windows: ~16,900 | Router: ~1,400 |
| `modes/tty/console` | `Broadcaster`: ~3,900 | same | same |
| map package | — | map and Propagation: ~10,500 | — |
| `modes/tty/observer` | — | — | ~25,900 |
| kit | 41 declarations, ~300-600 | same | plus `Surface` and the console keymap |
| Crossing symbols (unexported) | root→console 41 (30); root→kit 33 (29); console→kit 28 (24) | Map I's, plus root→map 254 (238) and map→root 122 (99) before the seam | root→observer 53-73 (37-57), plus the console's 41 |
| Cycles | none | one, removed only by the seam | none, once `Surface` and the keymap go down |
| `Dashboard` methods re-homed | 0 | 381 | 0 |
| New seam | `console.Mirror` value | plus map `Model`, `Env`, `Want`, `MemoKey`, `Answers` | exported Router-facing API on `Dashboard` |
| Test files straddling (references) | 40 (114) | 122 (512) | 50 (181-192) |
| `app` imports to change | the console's message types (17) | plus the map's data types (~100) | plus all of `Config` and `Dashboard` (161+) |

**The Bubble Tea composition under Map II** (types and signatures only):

- `modes/tty` (the root) keeps the program's model. The Router is still the only `tea.Model`, and
  `Dashboard.update` still routes the Observer's messages.
  - `type Router struct{ observer Dashboard; broadcaster console.Broadcaster; active Surface; … }`
  - `func (r Router) Update(tea.Msg) (tea.Model, tea.Cmd)` and `func (r Router) View() tea.View`
  - `type Dashboard struct{ …; mapview.Model /* embedded, TS-9 */; … }`
- `modes/tty/console` exports what the Router uses today:
  - `func New(console.Opening) Broadcaster`
  - `func (b Broadcaster) Update(tea.Msg) (Broadcaster, tea.Cmd)` and
    `func (b Broadcaster) View() tea.View`
  - `func (b Broadcaster) Mirrored(console.Mirror) Broadcaster`
  - `func Owns(tea.Msg) bool` (today's `consoleScoped`)
  - the card hand-over getters, for example
    `func (b Broadcaster) CardDetail(id string) (func(render.Opts) (string, []string), bool)`
- The map package exports:
  - `type Model struct` (the map's own state), and `type Env struct` (a value, built by the
    Dashboard for each call). `Env` carries the clock, `render.Opts`, the width and the window's
    room, the selected location, the snapshot, the units, clock and Reduce motion choices, the
    surface, and the map hooks.
  - `type Window uint8` (the map and its four Propagation windows), and
    `type Want struct{ Open Window; Close bool }` (what the map asks of the stack).
  - `func Answers(tea.Msg) bool`
  - `func (m Model) Update(tea.Msg, Env) (Model, tea.Cmd)`
  - `func (m Model) After(Env) (Model, tea.Cmd)`, the nine arms, in today's order
  - `func (m Model) Key(Window, tea.KeyPressMsg, Env) (Model, tea.Cmd, Want, bool)`
  - `func (m Model) Lines(Window, Env) []string`
  - `func (m Model) MemoKey(Window) Key`
- How messages and the modal reach the map:
  - The Router passes the map's replies to the Observer when `mapview.Answers` says so.
  - `Dashboard.update` hands them to `Model.Update` and then runs `Model.After`.
  - The modal stays the Dashboard's single value. The map is asked about a key only through its
    rows in `windowKeysOf`, and it opens or closes windows only by returning a `Want`, which the
    Dashboard applies through `open` and `close`.
  - The `Config`'s map hooks become an embedded `mapview.Hooks`. `app`'s assignments
    (`app/dashboard.go:426-431`) still compile by field promotion; `FindPlace`, set in the keyed
    literal at `app/dashboard.go:344-387`, would have to name the embedded struct (AS).

**Why Map II.**
- Map I alone moves 3,900 of 31,346 lines and leaves a 27,000-line package. That is the clean win,
  but on its own it does little for principle 2.
- Map II halves the root, gives the most-changed area a model whose state the rest of the Observer
  can no longer write by accident, and puts 18 map mutants on a smaller suite (PB).
- It keeps the routing, the stack and the memo key in one place (TS-8, TS-9), never exports the
  Router's coupling to the Observer (TS-6), and does not touch the deck (TS-10).

**Build order.**
1. The kit, then the console: pure moves, each behind its guards.
2. The map seam, as its own ruled batch. Write the guard changes (TS-9's depth, TS-12's
   package-reading guards) first, and see them fail before the move.

**The strongest argument against it.** The map seam is not a move. Its dangers:
- Size of the change: 381 methods change receiver, across a boundary that 254 and 122 symbols
  cross today in a cycle.
- Ordering: the nine post-Update arms (`dashboard.go:1157-1168`) depend on order and must survive
  intact.
- The memo key: it spans the map's windows, and its guard can lose coverage silently at the wrong
  depth.
- The regression net: 122 test files would straddle, so the tests change in the same place the
  behaviour could. Principle 4 is tested by a net that is itself being rewritten.

If the HUM LEAD rules that 0.19.6 is moves only, Map I is the map, and the map seam is the next
release's.

---

## Open questions for the HUM LEAD

1. **Scope:** does 0.19.6 include a change of state ownership (the map sub-model, Map II's second
   step), or only moves (Map I)? This decides whether D-1's "split along the seams" includes
   designing a seam that does not yet exist.
2. **API:** under Map I and Map II, `app` imports the console's message types, and under Map II the
   map's data types, from new paths. Or does the root keep type aliases so `app` reads `tty.X` as
   it does today? This is shared with AS.

## Touches other lenses

- **AS:** `app` uses about 400 exported tty names (`cross-package-edges.tsv`). Map I moves 17 of
  them (`broadcaster.go`), Map II moves about 100 more (`map_temp.go`, `map_prefs.go`,
  `map_radar.go`, `map_pane.go`), and Map III moves 161 from `dashboard.go` alone.
- **RX:** straddling tests are 40 (114 references) for Map I, 122 (512) for Map II and 50 (181-192)
  for Map III. `perturbEach` must become importable before the console moves (TS-9).
- **PB:** whether the console's 66 mutants and the map's 18 run measurably faster on their own
  suites, and whether the smaller copy per map method shows up (TS-11).
- **GP:** whether the concrete sub-model, with `Env` in and `Want` out, is the Bubble Tea practice
  GP endorses for a program this size, within D-3.
- **TG:** the guards that read the package directory (TS-12), `doc.go`'s file table, and the
  mutants' `Path("modes/tty/…")` anchors (145).
- **RM:** TS-10 is unchanged by every map here, and any D-2 change is RM's.
- **JN:** TS-2's misleading names, and `docs/extending.md`'s Setting walkthrough, which touches
  six `modes/tty` files and would cross a boundary under Map II for a map Setting.
