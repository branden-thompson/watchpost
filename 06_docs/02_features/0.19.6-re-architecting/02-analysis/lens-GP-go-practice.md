---
title: "0.19.6 DISCOVER — lens GP: Go and Bubble Tea practice"
date: 2026-11-14
phase: DISCOVER
lens: GP
status: "FILED — findings checked against e57afeaf"
---

# Lens GP: Go and Bubble Tea practice for a program this size, within D-3

## Verdict

watchpost already composes its UI the way Charm's own larger programs do. There is one root
`tea.Model` (the Router). Its children are concrete types, and each child's update returns that
type, not `tea.Model`. That is the shape of glow's `ui` package and of every Bubbles component.
0.19.5's PF-1 measured why this matters here. The split should keep that shape. It should not adopt
the nested-`tea.Model` style that soft-serve and gh-dash use (interface slices put back by type
assertion), because that style brings PF-1's per-message boxing back (principle 1).

Three things decide whether a split of `modes/tty` is safe, and none of them is file layout:
- **Methods belong to their type's package.** 752 methods hang on `Dashboard`, so moving a seam
  means moving logic onto a sub-type first.
- **The order of the post-dispatch steps.** They run after every message, and the order must hold.
- **No new message hops.** A direct call turned into a command and a reply adds an Update and View
  cycle, and the test harness's FIFO drain would not notice.

Go practice also supplies the cost metric for each boundary: how many identifiers it forces to be
exported, and how many white-box tests lose their reach. A further hazard is the package-walking
guards. They read `"."`, so they go quiet when code leaves the package.

The Charm surface is narrow: Lip Gloss sits in two `platform/render` files, and only a small part of
the Bubble Tea API is used. The upgrade path D-3 protects has never been used, though: the pins have
not moved since the first release. The split must not widen that surface.

## What I checked and how

- **Code:**
  - `modes/tty/router.go` and `modes/tty/dashboard.go`, read whole where it mattered: types,
    `Update`, `update`, `View`, the scope predicates and the post-dispatch chain.
  - `modes/tty/view.go`, `map_work.go`, `map_prop_update.go` and `map_night.go`.
  - `app/dashboard.go` (program construction, `livePipelines`, `ttyConfig`), `app/pool.go`,
    `app/pipelines.go`, `app/ticker.go` and `app/radio.go`.
  - The test harness (`modes/tty/dashboard_test.go` `runCmd` and `drain`), and the guards that read
    the package's own source.
  - Every `file:line` below was read on `e57afeaf`.
- **The shared graph:**
  - `symbols.tsv` and `test-reach.tsv`: field users of `Dashboard`, shared test helpers, and the
    reach of the map tests.
  - `cross-package-edges.tsv`: what `app` names in `modes/tty`.
  - `file-edges.tsv`: map files to the rest of the package.
- **Charm at the pinned versions:**
  - Read from the local module cache: `charm.land/bubbletea/v2@v2.0.3` (`tea.go` event loop,
    `commands.go`).
  - Compared with `v2.1.1`, also in the cache, by diffing `go doc -all` output.
  - `go mod graph` with `GOPROXY=off`.
- **`go doc -short`** on `./modes/tty` and `./app`, for each package's exported surface.
- **External sources** (general topics only; nothing about this project was sent). Each one is cited
  where it is used:
  - the Go blog;
  - go.dev "Organizing a Go module";
  - Go Code Review Comments;
  - the Go spec;
  - Google's Go style guide;
  - Dave Cheney's "Practical Go";
  - Russ Cox on gradual code repair;
  - Louis Garman's "Tips for building Bubble Tea programs";
  - Charm's composable-views example, Bubbles' `textinput`, and the root models of glow, soft-serve
    and gh-dash. superfile was checked for its layout only.
- **Not done:** no tests, no benchmarks, no live app. Byte figures are 0.19.5 PF-1's, cited and not
  re-measured.

## The sources, in one place

| Source | What it says that bears on 0.19.6 |
|---|---|
| Go blog, "Package names" (https://go.dev/blog/package-names) | "Packages named `util`, `common`, or `misc` provide clients with no sense of what the package contains"; contents' names "need not repeat the package name"; break up generic packages; if no good name fits, the boundary is probably wrong |
| go.dev, "Organizing a Go module" (https://go.dev/doc/modules/layout) | "Larger packages or commands may benefit from splitting off some functionality into supporting packages"; "It's recommended to keep packages in `internal` as much as possible", which leaves the code free to refactor its API |
| Go Code Review Comments (https://go.dev/wiki/CodeReviewComments) | "Go interfaces generally belong in the package that uses values of the interface type"; "Do not define interfaces before they are used"; return concrete types; avoid `util, common, misc, api, types, interfaces`; "If the receiver is a large struct or array, a pointer receiver is more efficient" |
| The Go spec, Method declarations (https://go.dev/ref/spec#Method_declarations) | A receiver base type "must be declared in the same package as the method" |
| Google Go style, Best practices: Package size (https://google.github.io/styleguide/go/best-practices#package-size) | Tightly coupled code belongs together; within a package it can share unexported detail "without polluting the public API"; putting an entire project in one package "would likely make that package too large"; test-helper packages take the production name plus `test` |
| Dave Cheney, "Practical Go" §4 and §5 (https://dave.cheney.net/practical-go/presentations/qcon-china.html) | Name a package for what it provides; "prefer fewer, larger packages", because every boundary forces identifiers public, giving a "wide, shallow, API surface"; prefer internal tests; use `internal` to reduce the public surface |
| Russ Cox, "Codebase Refactoring (with help from Go)" (https://go.dev/talks/2016/refactor.article), and Go 1.9 release notes (https://go.dev/doc/go1.9#language) | Gradual code repair needs an old and a new name for one type; type aliases shipped in Go 1.9 for moving a type between packages |
| Go stdlib `net/http/export_test.go` (https://github.com/golang/go/blob/master/src/net/http/export_test.go) | "Bridge package to expose http internals to tests in the http_test package". The bridge serves one package's own external tests, and no other package |
| L. Garman, "Tips for building Bubble Tea programs", 2024 (https://leg100.github.io/en/posts/building-bubbletea-programs/) | Keep Update and View fast and offload work to commands; messages from commands arrive in any order ("Embrace the chaos of concurrency"); use `tea.Sequence` or update the model directly when order matters; a non-trivial program becomes a tree of models, with the root routing and compositing |
| Charm `bubbletea/examples/composable-views` (https://github.com/charmbracelet/bubbletea/tree/main/examples/composable-views) | The parent holds `timer.Model` and `spinner.Model` as concrete fields, routes keys to the focused child, routes ticks by type, and joins the views |
| Charm Bubbles `textinput` (https://github.com/charmbracelet/bubbles/blob/main/textinput/textinput.go) | `func (m Model) Update(msg tea.Msg) (Model, tea.Cmd)`: a component returns its own concrete type, not `tea.Model` |
| Charm glow, `ui/ui.go` (https://github.com/charmbracelet/glow/blob/master/ui/ui.go) | One `ui` package; the root holds `stash stashModel` and `pager pagerModel` as concrete fields, routes by a `state` switch, and puts each child back without an assertion |
| Charm soft-serve, `pkg/ui` (https://github.com/charmbracelet/soft-serve/tree/main/pkg/ui) | Package per component and per page (`components/*`, `pages/*`, `common`); panes are a `[]common.TabComponent` interface slice, put back with an unchecked `m.(common.TabComponent)` |
| gh-dash, `internal/tui` (https://github.com/dlvhdr/gh-dash/tree/main/internal/tui) | Everything under `internal/`; about 25 component packages; the root's children are concrete (`m.sidebar, cmd = m.sidebar.Update(msg)`) but its sections are a `[]section.Section` interface; shared mutable state through one `*context.ProgramContext` given to every component |

## Findings

### GP-1: watchpost's composition is already Charm's own large-app pattern; nested `tea.Model`s would regress principle 1
- **Claim:**
  - **watchpost has one `tea.Model` boundary.** `Router.Update` is the one place the program's
    model becomes a `tea.Model`. Its routing `update` returns `Router`, and it delegates to
    `Dashboard.update` and `Broadcaster.Update`, which return their own concrete types.
  - **This is the shape Charm uses.** glow's root, Bubbles' components and the composable-views
    example are built the same way.
  - **The alternative costs here.** The nested-`tea.Model` alternative is soft-serve's interface
    slice with `m.(common.TabComponent)`, or gh-dash's `[]section.Section`. It boxes each child into
    an interface on every message, which is PF-1's cost again.
- **Evidence:**
  - **The Router.** `modes/tty/router.go:446` `func (r Router) Update(msg tea.Msg) (tea.Model, tea.Cmd)`.
    `:469` `func (r Router) update(msg tea.Msg) (Router, tea.Cmd)`. `:490-496` delegates to
    `r.observer.update` and `r.broadcaster.Update`.
  - **The two surfaces.** `modes/tty/broadcaster.go:498` `func (b Broadcaster) Update(msg tea.Msg) (Broadcaster, tea.Cmd)`.
    `modes/tty/dashboard.go:1151` is the Dashboard's one `tea.Model` adapter, and `:1156` its
    concrete `update`.
  - **The narrow interface.** `modes/tty/router.go:842-852` keeps `surfaceView` to `Init` and `View`,
    "so the Router needs no type assertion to put either back, and nothing copies a surface into an
    interface". `:502-512`: View draws through the concrete type and avoids the box (0.19.5 PF-1).
  - **The size of a box.** 0.19.5 PF-1
    (`06_docs/02_features/0.19.5-quality-pass/02-analysis/lens-PF-performance.md:54-110`) measured
    Dashboard at 6,536 bytes and Router at 7,072, and one interface box of either at about 6.5-7 KB
    per conversion. `TestRouterCostsObserverAlmostNothingPerFrame`
    (`modes/tty/broadcaster_alloc_test.go:64`) pins the View side.
- **Consequence:**
  - Principle 1: one box for each child turned into a `tea.Model`, on every message. The Map window,
    Settings and the console are each a candidate child.
  - Principle 4: soft-serve's unchecked assertion is a panic path that watchpost does not have
    today.
- **Recommendation:**
  - **A sub-package exports a concrete component:** its `Model` (or a named type), a constructor,
    `Update(msg tea.Msg) (Model, tea.Cmd)` returning its own type, and a view that returns a string
    or lines.
  - **Only the Router implements `tea.Model`**, and only the root produces a `tea.View`: v2's `View`
    carries cursor, background and alt-screen (`view.go:50-55`).
  - **No interface slices of children.** No `Component` interface package either: Code Review
    Comments says "do not define interfaces before they are used", and there is one consumer.
- **Confidence:** high.

### GP-2: A split keeps frames byte-identical only if it adds no message hop, and the test harness cannot tell
- **Claim:**
  - **One Update, then one View.** In Bubble Tea v2.0.3 each message is one `model.Update`, then one
    `model.View`.
  - **A batch has no order.** `tea.Batch` runs its commands in concurrent goroutines, with no
    ordering guarantee.
  - **Synchronous delegation keeps frames identical.** Splitting the root into sub-packages that the
    parent still calls synchronously, inside the same Update, gives the same state and the same frame.
  - **A message hop changes them.** A split that replaces a direct call with "the child returns a
    command, and its message is handled later" adds a whole Update and View cycle. It renders an
    intermediate frame, and its reply races other messages.
  - **The tests would not notice.** The test harness runs commands in FIFO order, synchronously,
    so a split like that would pass the suites while changing live behaviour.
- **Evidence:**
  - **The loop.** `$GOMODCACHE/charm.land/bubbletea/v2@v2.0.3/tea.go:872` `model, cmd = model.Update(msg)`,
    then `:880` `p.render(model)`, and `:886-889` render calls `model.View()` on every message.
  - **Batch and Sequence.** `commands.go:7-21`: Batch has "no ordering guarantees". `tea.go:918-957`
    gives each command in a batch its own goroutine. `:892-916`: Sequence runs in order.
  - **The harness.** `modes/tty/dashboard_test.go:151-177` `runCmd` flattens a batch into a FIFO list.
    `:181-192` `drain` feeds it back synchronously, so the suites see a deterministic order that the
    program does not have.
  - **The same advice from outside.** Garman's tips: use `tea.Sequence` or "update the model
    directly" when order matters.
- **Consequence:**
  - Principle 4 (no regression), and D-4's FULL TDD.
  - The regression net checks frames under a deterministic order. A new hop can be green there and
    still differ live.
- **Recommendation:**
  - **A PLAN rule for 0.19.6:** a split adds no `tea.Msg` type and no command where the code today
    calls directly. A child talks to its parent by return value, as `Broadcaster.Update` and
    `Dashboard.update` do now.
  - **Count the message types.** Pin the set of `*Msg` types across the split tree before and after,
    with `platform/declset`, run over the tree rather than one directory (see GP-7).
  - RX owns the net; this is the GP rule it would enforce.
- **Confidence:** high.

### GP-3: The post-dispatch steps run after every message and read across sub-states, so they stay at the root, in order
- **Claim:**
  - **What the steps are.** `Dashboard.update` follows each dispatch with a fixed chain of
    reconcilers: resume, map panic, map tick, night, chart, reach, ahead, centre, options and
    refresh.
  - **They run on every message.** That includes messages that are not the map's, such as a
    Settings save.
  - **They read state outside the map.** The open modal and the Settings choices are two examples.
  - **What a nested split would change.** If the map became a child whose `Update` receives only map
    messages, the chain would stop running on the other messages. A Settings change would then no
    longer re-ask the propagation snapshot.
- **Evidence:**
  - **The chain.** `modes/tty/dashboard.go:1156-1171`: the chain and one `tea.Batch` of its commands.
  - **The options step.** `modes/tty/map_prop_update.go:105-111` `armPropOptions` compares the held
    snapshot's options with `d.propOptions()`, which are the Settings.
  - **The map tick.** `modes/tty/map_work.go:188-203` `armMapTick` reads `d.modal`.
  - **The night.** `modes/tty/map_night.go:72-81` `followNight` reads `d.modal` and the clock.
  - **Order matters for state, not for commands.** The Batch's command order is not significant at
    runtime (GP-2), but each step reads the state the step before it produced.
- **Consequence:** principle 4. This is the most likely way a "behaviour byte-identical" split could
  regress silently: a re-ask that no longer happens after a Settings change.
- **Recommendation:**
  - **The step stays on every message.** A sub-package that takes over these steps exports one
    reconcile step, for example `(Model) Settle(in Inputs) (Model, tea.Cmd)` with the cross-state
    passed in, and the parent calls it after every dispatch, in today's position in the chain.
  - **Pin the order.** The chain's order is written down as a test-held list.
  - TS decides which seam carries it.
- **Confidence:** high on the mechanism (read in code). Medium on how many other chains exist like
  it. The Router's mirror-after-update at `router.go:446-466` is a second one at the Router level.

### GP-4: Go puts methods with their type, so splitting `modes/tty` means moving logic onto sub-types first, and that is two steps
- **Claim:**
  - **Most logic is on `Dashboard`.** Almost all of `modes/tty`'s logic is `Dashboard` methods.
  - **A method cannot change package.** The spec ties a method to its receiver's package, so no
    `Dashboard` method can move to another package.
  - **The sub-type comes first.** To split, each seam's logic must first become methods on a
    sub-type, such as the `mapPane` that already exists, with the few root fields it reads passed
    in. Only then can it move.
  - **Package by responsibility, in Go terms.** The responsibility is the sub-type.
- **Evidence:**
  - **The count.** 752 non-test methods have the receiver `Dashboard`, against 92 on `Broadcaster`
    and 32 on `Router`. They are spread over about 60 files (for example `map_pane.go` 51,
    `dashboard.go` 50, `map_temp.go` 42, `map_boxes.go` 40, `setup.go` 35). Source: grep of
    `^func (d Dashboard)`.
  - **The map's reach into the root.** Fields declared on `Dashboard` and used by non-test `map_*`
    files (from `symbols.tsv`): `.mapPane` 24 files, `.cfg` 18, `.now` 14, `.modal` 12, `.setup` 6,
    `.mapKeys` 6, `.mapCost` 4, `.snap` 3, `.surface` 2, `.modalScroll` 2, `.keys` 2, and `.width`,
    `.height`, `.selected` one each.
  - **The spec.** "must be declared in the same package as the method"
    (https://go.dev/ref/spec#Method_declarations).
- **Consequence:**
  - Principle 2: the sub-type gives the seam a name and a type a reader can find.
  - Principle 4: the receiver change is where logic actually changes hands. A pure file move cannot
    express it.
  - Principle 1: a handler that copies a 3.2 KB `mapPane` (PF-1's figure) in place of a 6.5 KB
    `Dashboard` may copy less; PB to measure.
- **Recommendation:**
  - **Two steps per seam.**
    1. In the same package, move the logic onto the sub-type behind tests. Each changed function is
       visible in review, and the mutants re-anchor here.
    2. Move the files to the new package. This step is a pure move, pinned by `declset` (re-keyed for
       the package qualifier) and by the import graph.
  - **Never both in one commit.** This keeps D-4's FULL TDD honest: the first step is where a test
    can fail.
- **Confidence:** high.

### GP-5: Name packages by what they provide, put what `app` does not name under `internal/`, and do not package by layer
- **Claim:**
  - **watchpost's names are already good.** Its existing packages follow the naming advice: no
    `util` or `common`, and names that say what is provided (`platform/singleowner`, `agememo`,
    `closedset`).
  - **No `internal/` directory exists**, so every package is importable by any module.
  - **The split should not package by layer.** Go practice argues against layer packages (`msgs`,
    `types`, `views`, `components`, `common`). It argues for packages by responsibility, with
    implementation kept under `internal/`.
  - **What `internal/` cannot do here.** `modes/tty/internal/x` can be imported only from under
    `modes/tty`. The types `app` names must therefore stay outside it.
- **Evidence:**
  - **No `internal/`.** `go list ./...` shows none outside `tools/internaltrees`, which is a tool's
    name, not a Go `internal` directory. No package is named util, common, misc, helpers or base.
  - **The exported surfaces.** `go doc -short ./app` lists 10 exported identifiers, the shape of a
    composition root. `go doc -short ./modes/tty` lists 102 top-level lines (66 types, 13 funcs,
    23 const lines).
  - **What `app` names.** Non-test `app` files name 344 distinct exported `modes/tty` identifiers,
    types and fields included (`cross-package-edges.tsv`). The most-named are the map's port types:
    `MapAsk` 27, `MapView` 23, `Region` 22, `Overlays` 22, `Chips` 15, `MapTemperature` 14.
  - **The sources.** go.dev layout: "keep packages in `internal` as much as possible". Code Review
    Comments: avoid `types` and `interfaces` names. Go blog "Package names": no stutter
    (`http.Server`, not `HTTPServer`).
- **Consequence:**
  - Principle 2: a junior finds a feature by name. A layer package scatters one feature across
    several.
  - Principle 3: best practice.
  - Unexported identifiers that a boundary forces into the export can stay invisible outside
    `modes/tty` when the package is internal.
- **Recommendation:**
  - **Name by responsibility.** Each new package is named for the responsibility it provides,
    matching the vocabulary in `docs/where-things-happen.md` (its row for Settings at `:242` already
    records that the code says `setup` for what users call Settings; JN's call).
  - **Use `internal/` where `app` has no business.** Packages that `app` does not import go under
    `modes/tty/internal/<name>`.
  - **Port types stay importable.** The port types `app` must name (the map's `MapAsk`/`MapView`
    family, `Config`, the messages `app` sends) stay in an importable package, either `modes/tty`
    itself or a named sibling.
  - **No stutter.** Inside a package, names do not repeat its name.
  - **Follow Bubbles for components:** `Model`, a constructor, `Update`.
- **Confidence:** high on the facts. Medium on where the port types should sit, which is TS's and
  AS's call.

### GP-6: The cost of a boundary is what it forces to be exported and what tests lose; the graph measures both, and `export_test.go` cannot bridge packages
- **Claim:**
  - **Cheney's counterweight.** "Prefer fewer, larger packages" holds against D-1. It does not
    overrule D-1, but it names the cost of every boundary: identifiers forced public, and white-box
    tests cut off.
  - **The map is a measured example.** Nearly every map test reaches internals outside the map
    files.
  - **The bridge does not exist.** Go's `export_test.go` serves one package's own `_test` package
    only. It cannot give a test in `modes/tty` access to `modes/tty/internal/mapwin`'s unexported
    state.
- **Evidence:**
  - **The map tests.** 66 of 67 `map_*` test files reach unexported symbols outside `map_*` files
    (`test-reach.tsv`): `dashboard.go:.mapPane` 57, `.cfg` 26, `view.go:Dashboard.opts` 23, `.now`
    16, `.modal` 10, `setup.go:Dashboard.setupRowKey` 9.
  - **Shared test helpers declared in `_test.go` files:** `stripANSITest` (used by 123 test files),
    `mapDash` 43, `pressKey` 31, `dash` 28, `feedAndSettle` 25, `snap` 23. Across the two packages,
    167 (tty) and 120 (app) test-declared symbols are used from other test files.
  - **Two kinds of helper.** `dash` drives the public path only (`NewDashboard`, `Update`;
    `dashboard_test.go:40-50`). `mapDash` reaches the unexported `closeMap`
    (`map_window_test.go:38-56`).
  - **Where white-box testing stands.** 362 of 363 test files are in-package (graph README).
- **Consequence:**
  - Principle 3 (best practice: in-package tests are what Cheney recommends).
  - Principle 4: RX's net. A boundary that strands 57 test files behind `.mapPane` forces each one
    to be rewritten, moved or re-pointed, and each rewrite is a chance to test less.
- **Recommendation:**
  - **Score every proposed boundary** before it is ruled: (a) the identifiers it forces exported,
    and (b) the test files that lose reach. Both come straight from the graph. Prefer the cut with
    the smallest pair.
  - **Tests move with their code.** Tests of a sub-type's logic move with it and test the sub-type
    directly.
  - **Tests through the root stay black-box.** Tests that drive the whole surface stay at the root
    and use messages and the frame, as `dash` does.
  - **Shared helpers get a test-support package.** Helpers that need only public API go in one
    `<name>test` package (Google style), under `internal/`. Helpers that need unexported state stay
    in-package.
- **Confidence:** high.

### GP-7: Guards that walk the package's own directory go quiet when code leaves it, and the Router's reply routing depends on unexported types
- **Claim:**
  - **The INST-1 guards read `"."`.** Several completeness guards read the package's own directory.
    After a split, they keep passing while they measure less.
  - **The sharpest case is the Router's reply guard.** `observerScoped` routes Observer's window
    replies by type switch. Its correctness argument is that the reply types are unexported, so only
    a Dashboard can have issued them. Its guard enumerates only the unexported `*Msg` types declared
    in `"."`.
  - **A moved reply escapes the guard.** If a reply type moves to a sub-package, the Router cannot
    name it unexported, and the guard no longer sees it. Its `found == 0` control does not fire,
    because other `*Msg` types remain.
- **Evidence:**
  - **The guard.** `modes/tty/router_lookup_test.go:131-170` (`declset.Files(".")` at `:139`; the
    `found == 0` control at `:167-169`).
  - **The argument it rests on.** `modes/tty/router.go:390-395` ("They are unexported replies to
    unexported commands ... what makes the routing decidable from the type alone"); the predicate is
    at `:396-421`.
  - **Other walkers of `"."`:**
    - `declset.Files(".")`: `modes/tty/map_mode_test.go:164`, `modes/tty/router_swap_test.go:73`,
      `app/maintrack_seam_test.go:71`, `app/maintrack_power_test.go:126`,
      `app/router_owner_test.go:21`.
    - `os.ReadDir(".")` or `filepath.Glob`: `modes/tty/main_test.go:34`,
      `modes/tty/held_clock_test.go:19`, `app/cast_test.go:723` and `:862`,
      `app/reachable_test.go:31`, `app/maps_basemap_test.go:177`.
    - The per-package golden in `app/declset_test.go:19-26`.
  - **Guards that fail loudly.** Parses by file name, such as
    `app/segments_completeness_test.go:39` on `"radio.go"`, stop with an error when the file moves.
- **Consequence:**
  - Principle 4: a guard that goes quiet is a regression net with a hole.
  - INST-1 (D-4): the iterated set must stay derived, and after the split it is derived from the
    wrong set.
- **Recommendation:**
  - **Walk the tree.** Every directory walker reads the package tree (for example
    `./modes/tty/...`), not `"."`.
  - **Name what moved.** Each walker gets a control that names one symbol known to have moved, and
    fails if the walk does not see it.
  - **Route moved replies by ownership.** Each sub-package exports an ownership predicate, for
    example `func Owns(msg tea.Msg) bool`, and the Router consults it. The Router does not
    type-switch on another package's types. The guard then checks every sub-package's `*Msg` types
    against their predicate.
  - RX and TG own the inventory; this is the Go mechanism.
- **Confidence:** high.

### GP-8: The Charm surface is narrow and the upgrade path is open but never used; the split must not widen the first, and 0.19.6 should not use the second
- **Claim:**
  - **Lip Gloss is behind one seam.** It is imported only by `platform/render`.
  - **Bubble Tea's surface is small.** About ten API names, plus a few message types.
  - **The pins have never moved.** They have not been re-pinned since the first build. Newer
    versions of both are in the local module cache, and the documented Bubble Tea API difference
    from v2.0.3 to v2.1.1 is additive.
  - **Two things widen the surface.**
    - (a) `app` passes `*tea.Program` through six signatures but uses only its `Send` outside
      `runProgram`. D-93's `func(tea.Msg)` seam is already the pattern in six places.
    - (b) The test harness recognises `tea.Sequence` by reflecting on the shape of an unexported
      Bubble Tea type.
- **Evidence:**
  - **Lip Gloss.** `platform/render/panel.go:8` and `platform/render/sgr.go:12` are the only non-test
    imports (grep over the tree).
  - **Bubble Tea, non-test.** 32 `modes/tty` files and 12 `app` files import it. Their API, by count:
    `tea.Cmd` 171, `tea.Msg` 90, `tea.KeyPressMsg` 38, `tea.Batch` 30, `tea.Program` 15, `tea.Model`
    9, `tea.View` 5, `tea.Cursor` 4, `tea.Tick` 3, `tea.Sequence` 1, plus the size, focus and colour
    messages.
  - **The pins.** `go.mod:5-7` pins bubbletea v2.0.3 and lipgloss v2.0.2.
    `06_docs/02_features/watchpost-cli/03-architecture-design/architecture.md:528` records that the
    Charm set was "what `go get` resolved at B0 ... never re-pinned".
  - **Newer releases.** The module cache holds `bubbletea/v2@v2.1.1` and `lipgloss/v2@v2.0.6`. Their
    `go doc -all` diff against v2.0.3 adds `ProgramStatus` and `RequestProgramStatusSupport` and
    rewords docs; no removal was seen.
  - **(a) The `*tea.Program` signatures:** `app/pipelines.go:185`, `:316`; `app/ticker.go:159`;
    `app/radio.go:143`; `app/dashboard.go:265`, `:499`. `Run` is used only at `app/dashboard.go:160`.
  - **(a) The send seam already exists:** `app/pool.go:230-234` ("IT TAKES A SEND SEAM, NOT A
    PROGRAM (D-93)"), and `send func(tea.Msg)` at `app/mastercontrol.go:47`, `app/severe.go:57`,
    `app/severe_read.go:60`, `app/ticker.go:100` and `app/executors.go:91`.
  - **(b) The reflection:** `modes/tty/dashboard_test.go:163-170`. `sequenceMsg []Cmd` is unchanged
    in v2.1.1 (`commands.go:30`).
- **Consequence:**
  - D-3: every package that names `tea.Program` is one more place a bump touches.
  - (b) breaks quietly on a bump that changes the type: the sequenced commands become invisible to
    tests. It has not happened yet.
  - Principle 1: none.
- **Recommendation:**
  - **The send seam.** When `app`'s actors leave `app`, each takes `send func(tea.Msg)`, and only
    the composition root names `*tea.Program`.
  - **Keep Lip Gloss behind `platform/render`.** No new package imports it (`make lint-imports`
    could hold that the way it holds go-studs; TG).
  - **A control for the harness.** Add a test that fails if `runCmd` stops unpacking a
    `tea.Sequence`. UNVERIFIED whether one exists: none was found.
  - **No Charm bump inside 0.19.6.** Principle 4 is judged against `e57afeaf`'s frames, and a
    renderer change would confound the comparison. Whether a bump follows the closing sweep is the
    HUM LEAD's (below).
- **Confidence:** high on the facts. Medium that v2.0.3 to v2.1.1 is fully additive: that is
  documentation-level, not a build.

### GP-9: Value receivers on a 6.5 KB model go against Code Review Comments; keep them in 0.19.6
- **Claim:**
  - **The guidance.** Code Review Comments advises a pointer receiver for a large struct.
  - **watchpost's choice.** It uses value receivers throughout, which is Bubble Tea's own example
    style. The memo slots are pointers shared by every copy of the model, so a copy costs a memmove
    and never a second render.
  - **Why not change it now.** Changing the receiver kind during a split would change aliasing:
    tests that hold an earlier copy would see later mutations. That is a behaviour change in the
    test suite and possibly in the program.
- **Evidence:**
  - **The model's size.** PF-1's sizes: `Dashboard` 6,536 bytes, `Router` 7,072.
  - **The shared slots.** `modes/tty/dashboard.go:820` `memo *bodyMemo` ("shared by every copy of
    the model") and `:824` `mmemo *modalMemo`.
  - **Other practice.** Garman: value receivers are the examples' style, and pointer receivers are
    allowed. gh-dash's root uses `func (m *Model) Update`.
- **Consequence:** principle 1 is unchanged by keeping them. Principle 4 is at risk if they are
  changed.
- **Recommendation:**
  - **Sub-types keep value receivers**, consistent with Bubbles.
  - **Receivers are a separate decision.** A pointer-receiver change is put to PB and the HUM LEAD
    on its own measured evidence, and is not bundled with a move.
- **Confidence:** medium. The aliasing risk is reasoned from how the tests are written; I did not
  inventory tests that keep two copies.

### GP-10: The interfaces already sit with their consumers; keep it so when `app` splits, and group `tty.Config`'s hooks by the package that consumes them
- **Claim:**
  - **Interfaces are few and consumer-side.**
    - `tty.Radio` and `tty.Station` are declared by `tty`, the consumer, and implemented by `app`.
    - `app`'s four interfaces are each declared beside their one consumer.
  - **The port is wide.** `tty.Config` has 108 field lines, 54 of them func hooks.
  - **`app` already groups the hooks.** It fills them by group, so the hooks already have seams that
    a split can follow.
- **Evidence:**
  - **`tty`'s interfaces.** `modes/tty/dashboard.go:479` (`Radio`), `modes/tty/router.go:137`
    (`Station`).
  - **`app`'s interfaces.** `app/arbiter.go:76` `narrationVoice`, `app/livesource.go:45`
    `liveSource`, `app/mapwaves.go:36`, `:41` `waveSource` and `marineSource`.
  - **The port.** `modes/tty/dashboard.go:67-380` (`Config`; counted excluding comment lines).
  - **The groups.** `app/dashboard.go:343` `ttyConfig`, `:395` `stationConfig` and `:424`
    `mapConfig`.
  - **The root.** `livePipelines` has 56 field lines (`app/dashboard.go:801-926`).
  - **The source.** Code Review Comments: "Go interfaces generally belong in the package that uses
    values of the interface type".
- **Consequence:**
  - Principle 2: a hook declared in the package that calls it is found where it is used.
  - Principle 3: best practice.
- **Recommendation:**
  - **Hooks move with their consumer.** When a `modes/tty` sub-package takes a responsibility, its
    hooks move with it as that package's own struct of funcs or a small interface. `tty.Config`
    then holds those structs.
  - **Interfaces move with their consumer.** When an `app` actor moves out (deck, arbiter, ticker,
    map feeds), it declares the interfaces it consumes.
  - **The root keeps only wiring.** `RunDashboard`, `livePipelines` and the `*Config` builders stay
    in the root.
  - **No shared `interfaces` or `types` package.**
- **Confidence:** medium-high. The hook-to-file grouping is TS's and AS's to settle.

### GP-11: Type aliases let a move and the re-pointing of its users be separate commits; none may survive the release
- **Claim:**
  - **The problem.** `app` names 344 `modes/tty` identifiers. A move that re-points every one of
    them in the same commit is a large, mixed diff.
  - **Go's tool for it.** Go added type aliases for moving a type between packages gradually.
  - **How it applies.** One commit moves a type and leaves `type X = newpkg.X` at the old path; a
    later commit re-points the users.
  - **The price.** While an alias exists, one type has two names, and that goes against principle 2.
- **Evidence:**
  - **The count.** 344 names, from `cross-package-edges.tsv` (GP-5).
  - **The toolchain.** `go.mod:3` `go 1.26.10`. Aliases exist since Go 1.9, and generic aliases
    since 1.24.
  - **The sources.** Go 1.9 notes and Russ Cox, both cited above.
- **Consequence:** principle 2 improves while a move is in review (smaller, pure diffs) and worsens
  if an alias outlives the release.
- **Recommendation:**
  - **Aliases are allowed only as a bridge** inside a batch.
  - **A gate at release.** The release fails if any alias that points into the split tree remains.
    The check is the AST's `TypeSpec.Assign`; TG.
- **Confidence:** medium. This is a mechanical choice and could be an A-row.

## Principles recommended for the 0.19.6 package map

| # | Principle | Source | watchpost evidence |
|---|---|---|---|
| P-a | One `tea.Model` boundary (the Router). Children are concrete types whose `Update` returns their own type. No interface slices of children | Bubbles `textinput`; glow `ui.go`; composable-views; 0.19.5 PF-1 | `router.go:446`, `:469`, `:842-852`; `broadcaster.go:498`; `dashboard.go:1151-1156` (GP-1) |
| P-b | No new message hops: a child reports by return value; the set of `*Msg` types across the tree is pinned before and after | Bubble Tea v2.0.3 `tea.go:872-889`, `commands.go:7-21`; Garman | `dashboard_test.go:151-192` FIFO harness (GP-2) |
| P-c | The post-dispatch reconcilers run at the root after every message, in today's order | Elm loop semantics (`tea.go:872-880`) | `dashboard.go:1156-1171`; `map_prop_update.go:105-111` (GP-3) |
| P-d | Per seam, first move logic onto a sub-type in the same package, then move the package as a pure move | Go spec, Method declarations | 752 `Dashboard` methods (GP-4) |
| P-e | Packages by responsibility, named for what they provide; no `util`, `common`, `types`, `msgs`, `components`, `interfaces`; no stutter; Bubbles-style `Model`, constructor, `Update` | Go blog "Package names"; Code Review Comments; Cheney §4 | the existing `platform/*` names (GP-5) |
| P-f | Everything `app` does not import goes under `modes/tty/internal/`; port types `app` names stay importable | go.dev "Organizing a Go module"; Cheney §5.1.3 | 344 names, `MapAsk`/`MapView` family (GP-5) |
| P-g | Every boundary is scored by identifiers forced exported and test files cut off, from the graph, before it is ruled | Cheney §5.1 "fewer, larger packages"; Google style "Package size" | 66 of 67 map test files; `stripANSITest` in 123 (GP-6) |
| P-h | Tests stay in-package and move with their code; tests through the root are black-box; shared public-API helpers go in one internal `<name>test` package | Cheney §5.1.2; Google style test-helper packages; `net/http/export_test.go` | `dash` vs `mapDash` (GP-6) |
| P-i | Source-walking guards read the tree, not `"."`, and each has a control naming a moved symbol; moved replies are routed by an exported ownership predicate | INST-1 (D-4) | `router_lookup_test.go:131-170`; `router.go:390-421` (GP-7) |
| P-j | Only the composition root names `*tea.Program`; actors take `func(tea.Msg)`. Lip Gloss stays behind `platform/render`. No Charm bump inside 0.19.6 | D-3; Code Review Comments (accept the narrowest dependency) | `app/pool.go:230-234`; six `*tea.Program` signatures (GP-8) |
| P-k | Value receivers kept; a receiver change is its own measured decision | Code Review Comments (Receiver Type), weighed against principle 4 | `dashboard.go:820-824` (GP-9) |
| P-l | Interfaces and hook structs live with their consumer; `tty.Config` holds per-package hook groups; no shared interfaces package | Code Review Comments (Interfaces) | `dashboard.go:479`; `router.go:137`; `app/dashboard.go:395`, `:424` (GP-10) |
| P-m | Type aliases only as a bridge inside a batch; none remain at release | Go 1.9 notes; R. Cox | 344 names to re-point (GP-11) |

## Open questions for the HUM LEAD

1. **A Charm bump after the closing sweep (scope).** The pins (bubbletea v2.0.3, lipgloss v2.0.2)
   have not moved since the first build (`architecture.md:528`), and newer releases exist. D-3's
   reason for keeping to Charm's interfaces is easy upgrades. Should 0.19.6 exercise that path once
   after its closing sweep, as its own step against its own frame comparison, or leave it to a later
   release? My recommendation is not inside the split itself (GP-8).
2. **Bubbles in D-3's scope (scope).** watchpost does not use Bubbles: its keymap
   (`platform/term`), text fields and scrolling are its own. D-3 names Bubble Tea and Lip Gloss. My
   recommendation is no change in 0.19.6, because adopting Bubbles would change behaviour. Whether
   D-3's "never re-implement Charm" covers Bubbles for later releases is the HUM LEAD's to say.

## Touches other lenses

- **TS:** GP-3 (which seam carries the reconcilers), GP-4 (the method-to-sub-type step), and GP-5 and
  GP-6 (scoring each cut; where the port types sit).
- **AS:** GP-8(a) (send seams as actors leave `app`) and GP-10 (interfaces moving with their
  consumers).
- **RX:** GP-2 (the FIFO harness cannot see a new hop) and GP-7 (guards that go quiet).
- **PB:** GP-1 (boxing per nested child), GP-4 (copy sizes of sub-type receivers) and GP-9.
- **TG:**
  - GP-7's walkers, GP-8's Lip Gloss import rule and GP-11's alias gate.
  - P10-05's per-package floors (`06_docs/p10-ledger.md`): a new package starts with no ratified
    floor.
  - P10-01's bare-name recursion rule (`06_docs/code-standards.md:329`): a package-level `New` that
    calls `sub.New` reads as recursion. UNVERIFIED whether same-name method calls on child fields are
    flagged. `Router.View` calling `observer.View` at `router.go:508-511` passes `make p10` today,
    which suggests not.
  - The ultraviolet pseudo-version in `go.mod` is newer than bubbletea v2.0.3 asks for; its reason
    is UNVERIFIED.
- **JN:** GP-5 (package names by what users call things: Settings vs `setup`).
- **RC:** `architecture.md:528` still lists `bubbles v2.0.0-rc.1` as built, but `go.mod` has no
  Bubbles requirement, so the as-built row is stale.
