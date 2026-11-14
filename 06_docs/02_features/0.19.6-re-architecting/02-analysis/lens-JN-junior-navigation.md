---
title: "0.19.6 DISCOVER — lens JN: a junior maintainer's navigation, and how to measure principle 2"
date: 2026-11-14
phase: DISCOVER
lens: JN
status: "FILED — findings checked against e57afeaf"
---

# Lens JN — a junior maintainer's navigation

## Verdict

- **The documented tasks are well routed today, and the undocumented ones are not.** The walkthroughs
  (`docs/extending.md`), the flow map (`docs/where-things-happen.md`) and the two `doc.go` file maps
  are accurate and held by tests. A task without a walkthrough falls back to grep through 88 files.
- **What makes `modes/tty` hard is one hub, not file size.** The median file is 237 lines. But the
  Observer model is reached by 71 of the package's 87 other source files. It carries 752 methods in
  68 files, and `d.cfg` is read in 45 of them. Any method can touch any state, so no area can be read
  on its own.
- **The file prefixes look like boundaries and are not.** The map and Settings depend on each other
  in both directions. The app's shared picker widgets live in a Settings file. "Dashboard" names both
  the Observer and the host of every window, the console's windows included.
- **Principle 2 can be measured.** Eleven criteria follow (JN-8), each with an instrument, today's
  figure and a target. Several of them can be gamed: a split that cuts the hub's fan-in while every
  area still reads the whole model is one example. So each criterion is paired with a guard. The
  strongest instrument is a fresh blind trace in M5's style on the new layout. The cheapest are the
  graph figures used here.
- **A split can make navigation worse.** It brings more packages to learn, exports that leak
  internals, interfaces with one implementer, and tests that move or get exported hooks (JN-9). Each
  has a known way to avoid it.

## What I checked, and how

- **Read whole:**
  - `docs/where-things-happen.md`, `docs/extending.md` (all eight walkthroughs),
    `docs/architecture.md`, `06_docs/code-standards.md`;
  - `modes/tty/doc.go`, `app/doc.go`;
  - README's "Start here" (`README.md:624-636`);
  - the atlas's diagram list and its map as-built text;
  - 0.19.5's `m5-blind-review.md`, and the T3 key's method section, to understand the instrument
    only. No site list from the hidden key is used as evidence here;
  - 0.19.5's lens JR, to see which of its findings have since been fixed (most have: `doc.go` file
    maps, the arbiter rename, the vocabulary rows, Walkthrough 6).
- **Walked three tasks through today's tree by reading:** a key in the map's legend (no walkthrough),
  a line on a Broadcaster card (Walkthrough 8), and a Settings row (Walkthrough 6). For each I
  recorded what a junior would open, in what order, and where the route breaks. I ran no tests.
- **Figures come from the shared graph** (`file-edges.tsv`, `symbols.tsv`, `test-reach.tsv`,
  `cross-package-edges.tsv`) with awk. File and line counts come from `wc`, `find` and `grep`.
  Export counts come from `go doc -short` against `go doc -short -u`. Imports come from `go list`.
- **Not run:** `go test`, any gate, the app. Which goldens a legend change would move is
  UNVERIFIED (JN-3).

## Findings

### JN-1 — The Observer model is the hub every area reaches into

- **Claim.** In `modes/tty` nearly every file reaches into one struct and its 310-line `Config`. A
  junior cannot tell from a file what state it reads or changes.
- **Evidence.**
  - Fan-in of `modes/tty/dashboard.go`: 71 of the other 87 non-test files in the package use its
    symbols. Counting tests, 239 files use them (`file-edges.tsv`). The next file is `map_pane.go`
    at 35. In `app`, the largest is `app/dashboard.go` at 25.
  - `func (d Dashboard)` carries 752 methods in 68 non-test files. The `Broadcaster` type has 90 in
    12 files. (Counted with grep over `modes/tty/*.go`.)
  - `type Dashboard struct` (`modes/tty/dashboard.go:643`) runs 193 lines with 80 named fields and
    three embedded groups: `mapChoices`, `uiChoices` and `radioChoices`
    (`modes/tty/dashboard_choices.go`). Because they are embedded, they read as `d.<name>`
    everywhere. `type Config struct` runs from `modes/tty/dashboard.go:67` to `:377`.
  - Field reach (`symbols.tsv`, source users / test users):
    - `.cfg`: 45 / 59 files. A grep for `d.cfg.` confirms the 45 source files.
    - `.mapPane`: 30 / 63.
    - `.modal`: 24 / 45.
    - `.setup`: 21 / 27.
    - `.now`: 21 / 32.
  - Distinct symbols each area reads from `dashboard.go` and `dashboard_choices.go`, non-test
    (`file-edges.tsv`):
    - the map files (`map_*`): 72, of which 60 are fields;
    - Settings (`setup*`): 102, of which 93 are fields;
    - the console (`broadcaster*`): 13, of which 7 are fields.
  - `app` is not built this way. `livePipelines` (`app/dashboard.go:801`) has 108 methods in 21
    files, but no field of it is read by more than 4 source files (`symbols.tsv`).
- **Consequence for D-1.**
  - Principle 2: understanding any map or Settings change means knowing the whole model.
  - For the split: a Go method must be declared in its receiver's package. As long as an area's
    code is written as `func (d Dashboard)` and reads `d.cfg` and `d.mapPane`, it cannot leave the
    package. The split's real design work is to give each area a state type of its own and a
    declared set of inputs. That is TS's question, and GP's for the idiom.
  - The console's 13 symbols make it the cleanest area to move first.
- **Recommendation.** Measure the split by the hub's reach, not its file name: JN-8 rows P2-2 and
  P2-3, with their guards. Record each area's declared inputs in its package doc, so a junior can
  read what the area depends on in one place.
- **Confidence:** high. The figures come from the graph and were cross-checked with grep.

### JN-2 — The file prefixes are not dependency boundaries, and the shared widgets live in Settings

- **Claim.** The `doc.go` rows group files by prefix. The dependencies cross those groups in both
  directions, so a junior cannot read one group alone. A split by prefix would hit import cycles.
- **Evidence.**
  - File-to-file edges between groups, non-test, from `file-edges.tsv`:

    | Groups | One way | The other way |
    |---|---|---|
    | map → map_prop | 22 | 39 |
    | map → dashboard | 30 | 14 |
    | map → setup | 19 | 16 |
    | frame → map | 13 | 11 |

  - What crosses between map and Settings:
    - Settings calls into the map: `setup.go` uses `Dashboard.mapPrefArrow`, `refreshMapCost` and
      `setZone`. `setup_layout.go` uses `mapSettingLines`. `setup_ui.go` reads twelve map keys.
    - The map calls into Settings: `map_prefs.go` uses 20 `rowMap…` row IDs from `setup_rows.go`.
      Five map files use `Dashboard.uiTouched` or `uiApplyCmd`.
  - The app's picker and chip widgets are declared in the Settings row table's file,
    `modes/tty/setup_rows.go`:
    - `checkMark` (`:496`), `arrowChips` (`:521`), `newArrowChips` (`:523`), `pickerCellW`
      (`:574`), `pickerFlash` (`:580`), `itself` (`:605`), `cycleIn` (`:618`).
    - `cycleIn` is used by 9 files, including five `map_*` files. `newArrowChips` is used by 7,
      including `debug.go` and `map_boxes.go` (`symbols.tsv`).
    - The reuse rule (0.19.0 D-147) tells a maintainer to build from these. Nothing in their file's
      name says they are there.
- **Consequence.**
  - Principle 2: the prefix map tells a junior where code is written, not what it depends on.
  - Principle 3: Go permits these cycles inside one package and forbids them between packages.
- **Recommendation.**
  - Before any package move, give the shared controls a home of their own, named for what they
    are. The map and Settings can then both depend on it without depending on each other.
  - Draw the package graph as a DAG in the atlas, each edge with its reason (P2-7). TS owns where
    the line falls.
- **Confidence:** high.

### JN-3 — A task with no walkthrough: a key in the map's legend

- **Claim.** A junior asked to add a key to the map's legend finds no route in any Start-here
  document. The one comment that names the legend's place routes to code that no longer draws one.
  The real path runs through files whose names do not say "legend".
- **Evidence.**
  - **No route in the docs.** "legend" does not appear in `docs/where-things-happen.md`,
    `docs/extending.md`, `docs/architecture.md` or `06_docs/code-standards.md` (grep).
  - **A wrong route.** `modes/tty/map_boxes.go:7-8` says: "The legend over the upper right is
    map_pane.go's (D-44)." The map's as-built text says "no legend (retired, D-103)"
    (`06_docs/02_features/observer-maps/03-architecture-design/as-built-map.md:34`).
    `map_pane.go` draws nothing over the upper right (grep: `upper` appears only at `:951`, the
    Description box over the upper left). This is RC's for the record.
  - **The real path.**
    1. `map_pane.go:617` appends `d.scrubRows(width)`.
    2. `map_scrub.go:56` (`scrubRows`) picks a row by mode:
       - Radar: `map_radar.go:391` (`radarLegendRow`);
       - Forecast: `map_temp.go:802` (`tempLegendRow`);
       - Propagation: `map_prop_draw.go:170` (`propLegendRows`).
    3. The classes come from go-tuimaps' `Legend()`, copied into the pane at `map_pane.go:484`.
  - **Misleading names on that path.**
    - `map_scrub.go`'s header calls the legend "the colour row" (`:3-4`).
    - `tempLegendRow` lives in `map_temp.go` but also keys UV, air quality, feels-like and wind
      (`:803-816`). The order of its `switch` is the precedence, and it says so only as "the key of
      what is drawn" (`:803`).
  - **Size.** At least five files to read, `map_pane.go` alone 1,694 lines, for an edit of one or
    two lines in one file. A new Forecast preset also needs go-tuimaps to offer it, which is out of
    tree (UV's is `tuimaps.UVGrid`, `app/history.go:691`).
  - **Coverage.** `map_radar.go` and `map_scrub.go` are named in no Start-here document. 34 of
    `modes/tty`'s 87 source files (doc.go aside) are reached from those documents only through a
    `doc.go` wildcard row. 20 of them are not in the atlas either, among them `map_radar.go`,
    `broadcaster_lineup.go` (`reportTypeOf`) and `setup_relay.go`. Counted by grep of basenames.
  - UNVERIFIED: which goldens and tests such a change would move.
- **Consequence.** Principle 2: walkthroughs cover eight growth axes; every other task is a grep
  through the hub. The M5 blind trace only measured tasks that had, or got, a walkthrough.
- **Recommendation.**
  - Include a no-walkthrough task in the blind trace on the new layout (P2-8).
  - Make the map package's doc name its four under-the-map rows, using one word for the legend.
  - The stale `map_boxes.go:7-8` header goes with whatever batch touches that file. Under
    AP-HIST-01 and the "docs reflect current reality" rule it has to go eventually anyway.
- **Confidence:** high on the route and the stale header. Medium on "at least five files", which is a
  reading path rather than a measured trace.

### JN-4 — Documented tasks: routed well, broad by construction

- **Claim.** The documented tasks reach every site, but the sites spread across files and packages.
  A split will tend to raise the number of packages per task, which is the wrong direction for a
  naive metric.
- **Evidence.** Counted from each walkthrough, with every symbol checked in the tree:
  - **Walkthrough 6, an on/off Setting.**
    - Ten source files in three packages:
      - `platform/config/config.go`;
      - `modes/tty/` `dashboard.go`, `dashboard_choices.go`, `setup_ui.go`, `setup_rows.go`,
        `setup_layout.go`, `motion.go` and `setup.go`;
      - `app/` `dashboard.go` (`ttyConfig`, `:343`) and `themes.go` (`setUIHook`, `:82`).
    - Two more when the Setting applies: `router.go` for the mirror onto the console (`:454`), and
      `masthead.go` for its memo key (`:74`).
    - Three test files: `modes/tty/config_reach_test.go`, `modes/tty/settings_kept_test.go` and
      `app/settings_kept_test.go`. Five goldens.
  - **Walkthrough 8, a line on a card.**
    - Two files edited: `broadcaster_slots.go` (`readBody`, `:130`) and `broadcaster_manifest.go`
      (`cardPulledShort`, `:126`).
    - Three files read: `broadcaster_upnext.go` (`:80`), `broadcaster_lineup.go`
      (`reportTypeOf`, `:120`) and `broadcaster.go` (`cardLane`, `:1262`; `inner`, `:1386`).
    - One package, two test files, 12 goldens, `CHANGELOG.md`.
  - **Under a plausible split**, a map Setting would touch `platform/config`, the tty root (`Config`,
    `UIPrefs`), a Settings package, a map package and `app`: five packages instead of three.
  - **The instrument's baseline.** At 0.19.5 batch 43, M5 graded T1, T2 and T4 pass and T3 fail
    (3 of 10 sites reached). T3 passed after Walkthrough 8 was written (`m5-blind-review.md`).
- **Consequence.** Principle 2: "packages touched" rises under a split, even a good one. "Files
  touched" changes little. What should fall is how much of each package a junior must know.
- **Recommendation.**
  - Count per task: files edited, files read, and **packages whose unexported internals the
    change needs**, as distinct from packages used only through their documented API (P2-6).
  - Keep each walkthrough's "Skipped, this fails" column. Re-measure it by leaving out one site at a
    time on the new layout, as 0.19.5 batch 36 did. The count of "nothing fails" rows must not rise.
- **Confidence:** high on the counts. The five-package figure assumes one plausible split and is
  illustrative.

### JN-5 — Names that send a junior to the wrong place

- **Claim.** 0.19.5 fixed the worst collision (the arbiter rename, 0.19.5 D-13). Several still
  route wrongly, and a split will add package names to the set.
- **Evidence.**
  - **"Dashboard" hosts every window, the console's included.** The console's card window and its
    move and drop prompts are `Dashboard` methods:
    - `broadcaster_detail.go:285` (`showCard`), `:309`, `:318`;
    - `broadcaster_manage.go:44`, `:73`, `:94`;
    - `request.go:174-524` (eleven `Dashboard` methods for the Line-Up Request window).

    Their state is on the Observer's struct (`dashboard.go:738-767`, `cardID` to `cardGen`). The
    Router reaches into it at `router.go:606` and `:813` (`r.observer.modal`). The type doc
    (`dashboard.go:641`) and the vocabulary row (`where-things-happen.md:128`) say only "the
    Observer surface". The reason, one window set under four gates, is stated in
    `broadcaster_detail.go:18-23`. A junior finds it only after opening that file.
  - **Seven basenames exist in both `app` and `modes/tty`:** `dashboard.go`, `debug.go`, `doc.go`,
    `setup.go`, `severe.go`, `ticker.go`, `timing.go`. Each pair holds different things:
    `app/dashboard.go` is the composition root, `modes/tty/dashboard.go` the Observer.
  - **One receiver name, four types.** The receiver `d` is used on `Dashboard` (752 methods),
    `radioDeck` (88), `arbiter` (17) and `dumper` (8). Reading a `d.` body requires checking the
    receiver first.
  - **One act, four names.** The duck path is:
    1. `app/arbiter.go:345` calls `giveWay`;
    2. `app/mastercontrol.go:321` (`giveWay`) calls `:338` (`dip`);
    3. `dip` calls `app/radio.go:908` (`duck`);
    4. `duck` calls `domains/radio/player/engine.go:521` (`Suppress`).

    That is four names in four files and two packages. The vocabulary defines "duck" as `dip` and
    `giveWay` (`where-things-happen.md:121`). D-2's ducking is RM's.
  - **An ambiguous subject.** `app/doc.go:3-4`: "It imports no domain and not the live surface
    (modes/tty)". The sentence is about `cmd/watchpost`, which is true (`go list`). It reads as
    `app`, which imports 43 watchpost packages including `modes/tty`.
- **Consequence.** Principle 2: each of these costs a junior a wrong file or a re-read. A split that
  names packages after mechanisms ("scrub", "boxes") or keeps code names that differ from users'
  words (`setup` for Settings, `where-things-happen.md:131`) adds more of them.
- **Recommendation.**
  - Name each new package for the user's or the role's word, and give it a vocabulary row (P2-10).
  - Correct `dashboard.go:641` and `where-things-happen.md:128` to say that `Dashboard` hosts every
    window, wherever the window host ends up (TS).
  - Rewrite `app/doc.go:3-4` with `cmd/watchpost` as its named subject.

  Names are A-rows under 0.19.5 D-13.
- **Confidence:** high.

### JN-6 — Every test reaches internals, and the tree has no `internal/` and no test-export idiom

- **Claim.** Tests are white-box throughout. After a split, the easy route is to export whatever the
  tests reach, and that leaks internals into the API a junior reads.
- **Evidence.**
  - `test-reach.tsv`, `modes/tty`: all 200 test files reach unexported symbols. Median 13, p90 32,
    maximum 103. 1,237 distinct unexported symbols are reached.
  - `test-reach.tsv`, `app`: all 136 test files reach unexported symbols. Median 10, p90 26,
    maximum 52. 838 distinct symbols are reached.
  - One test file reaches the internals of 22 source files (`map_prop_chart_test.go`). 53 test
    files reach only one source file.
  - Only one external test package exists in the two packages (`modes/tty/one_band_writer_test.go`,
    `package tty_test`).
  - `find` returns no `internal/` directory outside `third_party/` and no `export_test.go` anywhere
    in the tree.
- **Consequence.**
  - Principle 2: a test that reaches 22 files' internals is hard to read. Moving it with its code is
    the clean route.
  - Principle 4: RX holds the net.
- **Recommendation.**
  - Move each test with the code it reaches (test-reach.tsv names what moves together).
  - Target zero exported identifiers whose only users outside their package are tests (P2-5).
  - Treat any `export_test.go` as a cost to count, not a free pass. GP owns the idiom; RX the net.
- **Confidence:** high.

### JN-7 — The docs-to-code guards will catch a split's moves, with two gaps

- **Claim.** The guards that keep the navigation docs honest fail whenever a named `file:symbol`
  moves, which is what a split needs. Two gaps would let the new layout's maps go unheld or
  uninformative.
- **Evidence.**
  - `TestWhereThingsHappenNamesRealSymbols` (`cmd/watchpost/docs_test.go:62`) parses each
    `` `file.go:Symbol` `` and checks that the file declares it. `TestExtendingNamesRealSymbols`
    (`cmd/watchpost/extending_symbols_test.go:249`) does the same for `extending.md`.
  - The path references that will need re-pointing:

    | Document | `modes/tty/...go` | `app/...go` |
    |---|---|---|
    | `where-things-happen.md` | 86 | 97 |
    | `extending.md` | 34 | 26 |
    | `accepted-costs.md` | 7 | 3 |

  - **Gap 1.** `TestEveryFileIsInItsPackagesMap` hard-codes its packages:
    `range []string{"modes/tty", "app"}` (`cmd/watchpost/docs_test.go:353`). A new package's
    `doc.go` file map is held by nothing.
  - **Gap 2.** A wildcard row satisfies that test whatever its size. `map_*` covers 31 files and
    10,528 lines in one row (`modes/tty/doc.go:31-32`).
- **Consequence.** Principle 2: the maps are a junior's first stop. A map nobody checks goes stale,
  which is how 0.19.5 JR-3 found the old package doc.
- **Recommendation.**
  - Derive the test's package list (every package under `modes/` and `app/`, or every package with
    a file-map table) instead of listing it.
  - Cap the files one row may cover (P2-9).

  TG owns the gate. RC owns the re-pointing of the record.
- **Confidence:** high.

### JN-8 — Measurable criteria for principle 2

Each row has an instrument, today's figure, a target to put in the package map, whether the
criterion can be gamed, and the guard that stops it. "Graph" means the D-6 tool, re-run on the new
layout. Targets are recommendations. The package map sets the final ones (D-7).

| # | Criterion | Instrument | Today (`e57afeaf`) | Target | Gameable? Guard |
|---|---|---|---|---|---|
| P2-1 | Package size | `find`, `wc` per package, non-test | `modes/tty` 31,346 lines in 88 files. `app` 20,184 in 76. The next is `platform/render`, 5,989 in 22 | No package in the split over about 6,000 lines or 25 files, today's next-largest | **Yes**: cut into many small mutually dependent packages. Guard: judge with P2-3, P2-4 and P2-7; each package states a one-sentence contract in `doc.go`; a package under about 300 lines needs a stated reason |
| P2-2 | Hub fan-in | Graph: distinct non-test files using the largest type's declaring file, and files declaring methods on it | `dashboard.go` used by 71 of 87 files (82 %). 752 `Dashboard` methods in 68 files. `d.cfg` read in 45 | No type has methods in more than about 15 files. No file is used by more than a third of its package's files. `app/dashboard.go`'s 25 is a fair ceiling | **Yes**: move the struct into a shared "state" package every area imports (fan-in moves, it does not fall), or hide fields behind getters. Guard: measure by type, not file name, and count reach through accessors as reach |
| P2-3 | Area reach into the model | Graph: distinct symbols of the model each area reads | Map 72 (60 fields). Settings 102 (93). Console 13 (7) | Each area reads only its declared inputs, listed in its `doc.go`. The console stays at 13 or below | **Yes**: pass the whole model as one parameter, and the count drops to 1. Guard: count fields reached through the parameter, which the typed graph can do |
| P2-4 | Front door | `go doc -short` against `go doc -short -u` per package | `modes/tty` exports 89 top-level names of 942. `app` 10 of 655. 5 tty exports have no user outside the package (`AlertCategory`, `Broadcaster`, `Relation`, `StatusFeature`, `WatchCap`) | The front doors of `modes/tty` and `app` grow by no more than the new packages' entry points. Exports used only by sibling packages sit under `internal/` | **Yes**: export everything "to make it compile". Guard: count exported names used by one sibling package only, and names with no outside user; both are targets of 0 outside `internal/` |
| P2-5 | Test-only exports | Graph `test-reach.tsv` and `cross-package-edges.tsv`; grep for `export_test.go` | 362 of 363 test files in-package. Median reach 13 (tty) and 10 (app). No `export_test.go` | 0 exported names whose only outside users are tests. No test file reaches more source files than it does today | **Yes**: test hooks in `export_test.go`. Guard: count those exports as test-only exports |
| P2-6 | What a task costs | Each walkthrough, plus one no-walkthrough task, counted on the new layout | W6: 10 (+2) source files, 3 packages, 3 test files, 5 goldens. W8: 2 edited and 3 read, 1 package, 2 test files, 12 goldens. Legend key: at least 5 read, 1 edited, no doc route | Files read falls. Packages whose internals the change needs: at most today's. Each package a task crosses is named in its walkthrough | **Yes**: a registry or reflection that hides the wiring cuts the file count and raises the coupling a junior cannot see. Guard: the "Skipped, this fails" column re-proven by leave-one-out, with "nothing fails" rows not rising |
| P2-7 | A readable package graph | `go list -deps`, drawn in the atlas (FULL DIAGRAMS) | One package. Inside it, group cycles map↔setup (19/16), map↔dashboard (30/14), map_prop↔map (39/22) | A DAG in the atlas with each edge's reason. No package imports more than a handful of its siblings | **Yes**: break a cycle with an interface that hides the dependency. Guard: count interfaces introduced by the split that have one implementation; 0 unless ruled |
| P2-8 | Blind trace | M5's method (0.19.5 D-24) on the new layout, against a fresh hidden key | 0.19.5 batch 43: 3 of 4 passed first time; T3 reached 3 of 10 sites | 4 of 4 pass first time. At least one task has no walkthrough (a legend key), and one crosses a new package boundary (a map Setting) | **Yes**: a key written by the doc's author shares its blind spots, and tasks picked to match walkthroughs pass trivially. Guard: tasks fixed at PLAN, the key built by leave-one-out on a scratch copy before the docs are rewritten, the reviewer a fresh agent with no search |
| P2-9 | Docs-to-code checks | `TestWhereThingsHappenNamesRealSymbols`, `TestExtendingNamesRealSymbols`, `TestEveryFileIsInItsPackagesMap`, `TestTheAtlasMatchesAFreshRender` | Green. The file-map test is hard-coded to 2 packages, and one row covers 31 files | Green at every batch. The file-map test covers every package it derives. No row covers more than about 10 files | **Yes**: wildcard rows. Guard: the cap |
| P2-10 | Names | grep basenames and receivers; the vocabulary table | 7 basenames shared by app and tty. Receiver `d` on 4 types. `Dashboard` named "Observer" only. Settings is `setup` in code | Every new package has a vocabulary row and the user's or the role's word as its name. No new package is named for a mechanism | Low. Guard: a reviewer reads the vocabulary against `go list` |
| P2-11 | Long files | `wc -l` | 6 tty files over 1,000 lines (largest `dashboard.go`, 2,011). 2 in app (`radio.go`, 1,346) | None over about 1,000 lines | **Yes**: arbitrary file cuts. Guard: P2-9's file map, one owner per file |

**Not a criterion: the comment ratio.** Comment lines are 30 % of `modes/tty`, 31 % of `app` and 49 %
of `platform/lineup`. That says nothing about understandability, and a target on it would be met by
writing or deleting prose.

**Recommendation.**
- Adopt P2-2, P2-3, P2-6, P2-8 and P2-9 as the release's evidence for principle 2, measured at PLAN
  (the baseline above), at each package move, and at REVIEW.
- P2-1, P2-4, P2-5, P2-7, P2-10 and P2-11 are guards against gaming those five.
- Whether the graph tool becomes an in-tree check is A-4's PLAN question (TG).

**Confidence:** high on today's figures. The targets are judgement.

### JN-9 — What a split could make worse for a junior, and how to avoid it

1. **More packages to know.** Two packages become N, and a task crosses more of them (JN-4).
   - Avoid: shallow package paths under `modes/tty/` and `app/`, named in the vocabulary. One
     `doc.go` per package with its file map and declared inputs.
   - The walkthroughs name each package a task crosses.
2. **Exports that leak internals.** Siblings must export what they share, and `go doc` then shows a
   junior internals next to the API.
   - Avoid: `internal/` for sibling-only names (none exists in the tree today, JN-6), and P2-4's count.
3. **Indirection through interfaces.** A cycle broken by an interface sends "go to definition" to an
   interface with one implementer, which a junior then has to find.
   - Avoid: concrete types, and plain function values in a config struct, which is the tree's
     existing idiom (`tty.Config`'s hooks).
   - Use an interface only where there are two implementations or a Charm interface already
     stands (D-3). P2-7 counts them. GP owns the idiom.
4. **The whole model passed as a parameter.** An area function that takes the old struct whole meets
   P2-2 and keeps JN-1's problem.
   - Avoid: P2-3's guard. Each area's inputs are a named, documented type.
5. **Tests that move, or stay and reach through new exports** (JN-6).
   - Avoid: tests move with the code they reach, and P2-5 is held at zero.
6. **Goldens and their flags spread.** `modes/tty/testdata` holds 64 files and `-update-golden` is
   declared once (`modes/tty/golden_test.go:20`). A split spreads goldens across packages, and
   `extending.md:257-262` must list every package's flag.
   - Avoid: keep the frame goldens with the surface they draw, and keep the flag list in that one
     place.
7. **Two maps of the same thing drift.** During the moves `doc.go`, `where-things-happen.md`, the
   as-built maps and the atlas all change.
   - Avoid: each batch moves its doc rows and diagrams in the same commit, held by P2-9 (RC, TG).

## What is in good shape (do not spend 0.19.6 here)

- **The walkthroughs and the flow map.** Every symbol I followed in Walkthroughs 6 and 8 exists where
  named. The "Skipped, this fails" column of Walkthrough 6 is the best single navigation aid in the
  tree.
- **The `doc.go` file maps** of both packages, held by a test.
- **File size in general.** Median 237 lines in tty, 177 in app. The problem is the hub, not the files.
- **The console.** It reads 13 symbols of the Observer's model, and its card has a walkthrough.

## Open questions for the HUM LEAD

1. **Is principle 2 held by a pass/fail gate at REVIEW?** 0.19.5 D-24 made the M5 blind trace a
   ship criterion. The proposal is the same for 0.19.6 (P2-8): four tasks fixed at PLAN, one with no
   walkthrough and one crossing a new package boundary, graded against a fresh hidden key, with a
   fail fixed before SHIP. The alternative is to report it without a gate.
2. **Are the graph figures (P2-2, P2-3, P2-6) the release's evidence for principle 2?** They would be
   reported at PLAN, BUILD and REVIEW against the baseline above. The alternative is to judge
   principle 2 by the blind trace alone. (Whether the tool becomes an in-tree check is A-4's
   separate PLAN question.)

## Touches other lenses

- **TS:**
  - the hub's state and its 752 methods (JN-1);
  - the map↔Settings cycle and the widgets' home (JN-2);
  - where the window host lives, given that the console's windows are `Dashboard`'s (JN-5).
- **AS:** `app` is not hub-shaped; its largest fan-in is 25 files (JN-1). Its `dashboard.go`
  basename and `doc.go:3-4` wording (JN-5).
- **GP:**
  - `internal/` for sibling-only exports;
  - interfaces with one implementation;
  - method receivers and package location (JN-1, JN-9).
- **RX:** test reach and test-only exports (JN-6).
- **RC:**
  - the stale `map_boxes.go:7-8` route against `as-built-map.md:34` (JN-3);
  - about 250 path references in the docs to re-point (JN-7).
- **TG:**
  - `TestEveryFileIsInItsPackagesMap`'s hard-coded list (`docs_test.go:353`);
  - the graph tool as a check (A-4) (JN-7, JN-8).
- **RM:** the duck path's four names across four files (JN-5).
