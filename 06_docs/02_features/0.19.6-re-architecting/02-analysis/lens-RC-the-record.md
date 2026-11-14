---
title: "0.19.6 DISCOVER — lens RC, the record"
date: 2026-11-14
phase: DISCOVER
lens: RC
status: "FILED — findings checked against e57afeaf"
---

# Lens RC: the record

Read-only. Prefix RC. The question: which rulings, architecture documents and as-built records
constrain the shape of `modes/tty` and `app`, and which documents must move with the code.

## Verdict

1. **The record does not forbid splitting either package; it forbids a handful of shapes and prices
   the rest.** Forbidden: splitting the shared deck (HUM LEAD, settled); copying a window or a rule
   into a second package instead of sharing one owner (0.16.0 D-56); re-implementing Charm (D-3).
   Everything else is allowed. It is paid for in ratifications (P10), re-pointed mutants and documents
   moved in the same commit.
2. **The costliest constraints are in the gates, not in the prose.** Seven guards that enforce
   rulings are scoped to the package as it is today: a flat `tty.Config`, one level of `Dashboard`,
   one directory, or a hard-coded list of two packages. A move puts code outside them **with every
   test still green** (RC-3). Under D-4's INST-1, the list each guard checks has to be computed, and
   that work comes before the first move.
3. **Two tools turn a pure move into a cascade of findings.** AP-CITE-01 judges citations by
   directory, so every old bare ID in a moved file turns into a "new" one (RC-5). P10-05 counts every
   moved function as changed, and a new package has no floor, so each one needs a HUM LEAD
   ratification (RC-4).
4. **A split by surface (Observer and Broadcaster) runs into "forwarded, not rebuilt".** The console
   uses eight of Observer's windows, the Line-Up Request among them. `modes/tty/doc.go` groups files by
   topic, not by type (RC-2).
5. **The record and the code disagree in six places today**, the window stack the largest (RC-7).

## What was checked, and how

- Read whole: 0.19.6's rulings and required reading; 0.16.0's required reading; `docs/architecture.md`,
  `docs/where-things-happen.md`, `docs/extending.md`; `06_docs/code-standards.md`; the 0.19.5 rulings
  dispositions and its REFLECT; `modes/tty/doc.go`, `app/doc.go`.
- Read in part, at the cited lines: `docs/accepted-costs.md`, `06_docs/p10-ledger.md`,
  `06_docs/follow-ups.md`, `06_docs/quality-observations.md`, `06_docs/quality-plan.md`,
  `06_docs/duplicates-ratified.md`, the 0.16.0 rulings files (D-50, D-51, D-56, D-58, D-74, D-91),
  the as-built maps, the 0.18.0 rulings (D-23, D-30, D-41, D-46, D-61, D-62, D-106, D-107, D-147), the
  0.19.0 rulings (D-27, D-28, D-212, D-230, D-264, D-283, D-306, D-327), the multi-voice director
  charter's disposition sheet, the 0.9.5 quality pass's file-split precedent (L3, Q2, RP-1).
- Searched the record for prior split proposals ("subpackage", "import cycle", "package boundary",
  "new package", "own package") and for single-owner rules ("one writer", "one owner", "two carriers").
- Checked each claim against the code at `e57afeaf`: `go list` for imports, grep and reading for
  symbols, receivers and comment text, and the source of the tests and tools that read documents
  (`cmd/watchpost/docs_test.go`, `extending_symbols_test.go`, `record_test.go`, `tools/authoring/cite.go`,
  `tools/atlas`, `app/air_boundary_test.go`, `modes/tty/memo_completeness_test.go`). Nothing was run
  beyond `go list` and grep.

---

## The constraints every candidate package map must satisfy

**Hard** means a HUM LEAD ruling, or a gate that holds one. **Recorded** means a preference, a lesson
or a design reason the record states, which a map may depart from only by saying why.

| # | Constraint | Source | Kind |
|---|---|---|---|
| K-1 | Performance the same or better; architecture more maintainable, traceable and understandable; best practice and P10 hold; no behaviour regresses | 0.19.6 D-1, `02-analysis/rulings.md:24` | Hard |
| K-2 | Charm, Bubble Tea and Lip Gloss are used through their interfaces; the composite's cost stands | 0.19.6 D-3 `rulings.md:26`; 0.19.0 D-327 `propagation-overlays/02-analysis/rulings.md:348`; `docs/accepted-costs.md:20` | Hard |
| K-3 | **The shared deck is not split.** One `radioDeck` for both surfaces, the Director arbitrating both rotations, MasterControl gating the air; a split moves the deck whole | 0.19.6 `00-REQUIRED-READING.md:38-40`; 0.16.0 `00-REQUIRED-READING.md:113-152` (HUM LEAD 2026-09-12); `rulings-d74.md` | Hard |
| K-4 | The disposition sheet is honoured item by item: `tune`, `SetMode`, `followMount`, `startSynth` are Instruct and stay in the deck; `Suppress`/`Restore`/`giveWayLocked` stay in the engine | `multi-voice-support/01-objectives/director-charter.md:172-173`; 0.16.0 `00-REQUIRED-READING.md:44-61` | Hard |
| K-5 | Five roles (Producer, Composer, Director, Reader, MasterControl); the arbiter is not a role, and it dissolves only under D-2's conditions | 0.16.0 `00-REQUIRED-READING.md:21-40`; 0.19.5 D-13 `0.19.5-quality-pass/02-analysis/rulings.md:38`; 0.19.6 D-2 `rulings.md:25` | Hard |
| K-6 | The air's guard boundary is every `tty.Config` func field and every `tty.Radio` method, a closed set derived by reflection; the guard sits at the caller; the exported `Radio` methods are the monitor's surface, the lower-case internals shared | 0.16.0 D-91, `as-built-air-and-deck.md:124-127`, `:155-158`; `app/air_boundary_test.go:213-243` | Hard |
| K-7 | Only MasterControl writes the band (constructs the takeover message) | `docs/where-things-happen.md:181`; `app/executors.go:46-48`; unenforced, F-22 `06_docs/follow-ups.md:167` | Hard (by convention only) |
| K-8 | One canonical way to do a thing: a duplicate only where isolation is a real need, named where it lives. Observer's Settings, About, Status, Help, Lookup, map, diagnostics and Line-Up Request windows are forwarded to the console, never rebuilt | 0.16.0 D-56 `rulings-d37-d40.md:687-704`, D-58 `:738`; `modes/tty/router.go:80-95`, `:729-742` | Hard |
| K-9 | Metric D: no unexplained duplicate in production code; a copy made to dodge an import fails `make dupes` unless ratified | `06_docs/duplicates-ratified.md:1-32` (HUM LEAD 2026-09-08) | Hard |
| K-10 | Consolidation is judged, not counted: two carriers answering two questions stay two | 0.19.5 `00-REQUIRED-READING.md:40-41`; 0.16.0 `track-model.md:305` | Recorded |
| K-11 | One map component and one propagation service, shared by Observer and the Broadcaster, its bound set per mode; seams left for the future Analyst and Scout modes | 0.19.0 D-28 `propagation-overlays/02-analysis/rulings.md:49`, D-27 `:48` | Hard |
| K-12 | The map library is called in Update; View only prints | 0.18.0 D-41 `observer-maps/02-analysis/rulings.md:57`; `docs/where-things-happen.md:24` | Hard |
| K-13 | An open window owns its keys; a focused picker keeps ←→; tab and shift+tab always escape; windows are a stack; each window declares its keys once (`window_keys.go`); the Router keeps the surface's keys above the stack | 0.18.0 D-61, D-62 `observer-maps/02-analysis/rulings.md:77-78`, D-106, D-107 `:122-123`; F-184; `as-built-map.md:155-166` | Hard |
| K-14 | Keybindings are data: each scope's keymap is named once, in its owner; the keys inside a window over the map are fixed | `docs/extending.md:21-30`; 0.19.0 D-292 `propagation-overlays/02-analysis/rulings.md:313` | Hard |
| K-15 | Settings: one window with tabs; each row belongs to a surface, ruled by the HUM LEAD; exactly two exits, both write; Settings, not flags; the model is built from per-concern helpers (`mapChoices`, `uiChoices`, `radioChoices`) | 0.18.0 D-62; 0.16.0 D-92; `where-things-happen.md:176`; `extending.md:164-166`, `:195-197`; 0.19.5 D-14 `rulings.md:39` | Hard |
| K-16 | Shared Settings (units, clock, Reduce motion) are mirrored across the surfaces by the Router, never owned twice | `modes/tty/router.go:435-436`; 0.19.5 D-21 `rulings.md:46`; 0.19.0 D-360 | Hard |
| K-17 | Each render memo sits behind a completeness guard over its key. The guards derive the model's fields by reflection, walk embedded structs as the model, and descend one level; the walk is generic over both surfaces' models, and a second copy of it is "the one duplication that matters" | 0.19.5 D-4 `rulings.md:29`, D-22 `:47`, D-31 `:56`; F-30; `modes/tty/memo_completeness_test.go:116-121`, `:147-188` | Hard (D-4); walk shape Recorded |
| K-18 | Layout: the D-50 breakpoints are the classifier's (100/120/150); D-51's right-rail seam is `cardBoxWidth`, the one place card width is computed; each of the layout's widths has one owner (D-80) | `rulings-d37-d40.md:462-505`; `modes/tty/broadcaster_rail.go:129-140`; `modes/tty/broadcaster.go:658-670` | Hard |
| K-19 | `modes/` and `platform/` import no domain; only `platform/render` imports go-studs; a domain reaches the surfaces only as a hook in `tty.Config`; no `modes/` package reads `platform/config` | 0.19.5 D-10 `rulings.md:35`; `scripts/lint-imports.sh`; `docs/architecture.md:66-69`; `extending.md:18-20`; 0.16.0 `perf-and-structure-pass.md:87-90` | Hard (imports); Recorded (config) |
| K-20 | P10-05: density 2.0 per package, or a floor the HUM LEAD ratified; never self-issued; `modes/tty` FLOOR 0.42, `app` FLOOR 0.61 | `06_docs/code-standards.md:317-333`; `06_docs/p10-ledger.md:122`, `:129`; 0.19.0 D-283 `:304`, D-264 `:285` | Hard |
| K-21 | The P10 ledger is keyed by file path; a re-key is a ledger change, which D-4 allows only by ratification | 0.19.6 D-4 `rulings.md:27`; `p10-ledger.md:49-56` ("RE-KEYED … Ratify all ten") | Hard |
| K-22 | AP-CITE-01: a new comment cites its release; old bare citations stay, and their release is the commit that wrote the line | 0.19.5 D-11 `rulings.md:36`, D-37 `:62`; `code-standards.md:210-229`; `where-things-happen.md:147-149` | Hard |
| K-23 | Every move keeps its mutants: anchors searched before the move, re-pointed and re-vetted; the closing sweep is judged against A-3's | 0.19.6 `00-REQUIRED-READING.md:41-43`; 0.19.5 D-13 (41 re-pointed) | Hard |
| K-24 | Diagrams move with the code; the atlas draws `docs/` and the as-built maps only; 0.19.6 draws the architecture at PLAN, BUILD and REFLECT | `observer-maps/04-development/build-log.md:180-184`; `app/maps_test.go:51-83`; 0.19.5 D-15 `rulings.md:40`; 0.19.6 D-4 | Hard |
| K-25 | INST-1: the set a gate iterates is derived, never enumerated | `06_docs/quality-plan.md:136`, bound by 0.19.6 D-4 | Hard |
| K-26 | Comments state the code as it is now, with rationale as a short Why list; docs state the code as it is now | AP-HIST-01 `code-standards.md:57-105`; 0.19.5 D-34 `rulings.md:59` | Hard |
| K-27 | Product principles: build for performance and structure (P-1); user choice through Settings (P-2); never architect into a corner (P-3); fix foundationally, ignore sunk cost (P-4); honest and complete before fast (P-5) | 0.18.0 D-23 `observer-maps/02-analysis/rulings.md:39`, D-30 `:46`, D-46 `:62` | Hard (standing guidance) |
| K-28 | Before wiring a seam, read the release that built it (P-8). A pure move is proven by a declaration set, one package per commit, with exemptions re-keyed and docs moved in that same commit. A file is a responsibility a newcomer can name in three words | 0.16.0 `p3-flip-postmortem.md:119`; 0.9.5 RP-1 `watchpost-performance-quality-pass/03-architecture-design/quality-pass-plan.md:383`; `04-development/q2-file-map.md:3-11` | Recorded |
| K-29 | A ruling whose behaviour crosses a package boundary gets one assertion that drives both sides together | `06_docs/quality-observations.md:3148-3152` | Recorded |
| K-30 | A gate that has to know its own depth breaks when it moves: walk up to `go.mod` | `quality-observations.md:1173-1177` | Recorded |
| K-31 | Accepted costs stay accepted until their trigger is measured; their `ACCEPTED COST` site comments and the allocation pins (`bcFrameAllocs`; the Observer's 141–155 at ×1.05) move with the code | `docs/accepted-costs.md:57-75`, `:222-253` | Hard |

**The prior proposals.** The record holds no ruling against splitting `modes/tty` or `app`. Earlier
releases kept them whole for reasons of scope:
- 0.9.5 split files only: "nothing moves across a package boundary (that would be a seam, Q6)"
  (`q2-file-map.md:11`). Its DISCOVER lens L3-F7 found "No new package warranted"
  (`lens-L3-structure-quality.md:16`).
- 0.18.0 PLAN put the map window in package `tty`, giving a reason D-1 does not answer by itself:
  "the memo-completeness and closed-set guards are package-internal, and a subpackage would
  import-cycle with the Dashboard" (`observer-maps/04-development/implementation-plan.md:58`).
- What has been rejected again and again is splitting the shared deck (K-3).

D-1 overrides those scopes. The reasons they gave are still facts that a package map has to answer.

---

## Findings

### RC-1: The deck, the roles and the air are settled; a split may move them but not divide them

- **Claim.** In `app`, the record fixes what may not be separated:
  - `radioDeck` is one deck (K-3);
  - MasterControl is the one writer of the band and the effector of the duck (K-7, `app/mastercontrol.go:463-468`);
  - the arbiter decides and MasterControl performs (`app/arbiter.go:120`, `:345`);
  - the disposition sheet's Instruct items stay where they are (K-4).

  Putting the deck and its files in a package of their own is consistent with "a split moves the deck
  whole" (0.19.6 `00-REQUIRED-READING.md:38-40`). Splitting `radio*.go`, `bed*`, `livesource.go` or
  `cast.go` across packages is not.
- **Evidence.**
  - 0.16.0 `00-REQUIRED-READING.md:148-152` records that this split has been proposed "more than
    once, across sessions", each time from rebuilding the model out of `app/radio.go`.
  - `app/doc.go:31-36` groups the deck, the arbiter and MasterControl as separate owners.
  - F-22 (`follow-ups.md:167`): nothing enforces the band's single writer. Its mutant (m50) was
    retired because no behaviour can tell two writers apart.
- **Consequence.** Principle 4 (no regression) and 2 (traceability). A boundary drawn through the
  deck is a re-litigation, not a seam.
- **Recommendation.** The package map states for each deck file which owner it belongs to, citing
  K-3 to K-7. Where a package boundary can enforce a single-owner rule (F-22's "no file outside
  MasterControl constructs the takeover message"), the map says so: an unexported constructor in
  MasterControl's package is the gate F-22 asked for. RM owns the roles; this lens names only the
  record.
- **Confidence.** High.

### RC-2: A split by surface collides with "forwarded, not rebuilt", and `doc.go`'s rows are topics, not types

- **Claim.** Observer owns windows that the console opens: Settings, About, Status, Help, Lookup, the
  map and ctrl+d. The Router forwards them and composites them over the console (D-56, D-58, D-65).
  The Broadcaster's own Line-Up Request window is a `Dashboard` window too. So the type boundary
  does not follow the surface names, and `modes/tty/doc.go`'s file map, held by a test, groups by
  topic: it is not a package map.
- **Evidence.**
  - `modes/tty/router.go:80-82`: "FORWARDED, NOT REBUILT. Settings, About, Status and Help are
    Observer's own windows".
  - `:87-95`: Lookup forwards on the HUM LEAD's UAT ("Expected: Location Search Modal pops up").
  - `:729-742`: `r.observer.openRequest()` and `throughToObserver` for settings, about, status, help,
    quit, lookup, map and diagnostics.
  - `modes/tty/request.go:524` and all 10 methods in `request.go` take a `Dashboard` receiver, yet
    `modes/tty/doc.go:33` maps `request.go` with "the Broadcaster console".
  - `broadcaster_manage.go` declares 3 `Dashboard` methods and no `Broadcaster` method;
    `broadcaster_detail.go` declares 3 `Dashboard` and 6 `Broadcaster`; `cursor.go` declares 2
    `Broadcaster` methods.
  - 0.19.0 D-28 makes the map one shared component (K-11).
- **Consequence.** Principle 2. A map that cuts along `doc.go`'s rows or file prefixes leaves
  `Dashboard` methods in a Broadcaster package, which does not compile. A map that "fixes" this by
  rebuilding a window for the console breaks D-56 and D-28.
- **Recommendation.** Every candidate map lists the windows both surfaces open and gives each one
  owner that both import, or keeps it with the Router. It answers the 0.18.0 PLAN's stated objection:
  the window guards are package-internal, and a window subpackage would import-cycle with
  `Dashboard`. TS owns the cut.
- **Confidence.** High.

### RC-3: Seven guards that hold rulings are scoped to the package as it stands, and a move escapes them silently

- **Claim.** Each guard below encodes a ruling, and each would still pass after code moved out of
  its reach. D-4 binds INST-1 (K-25), so each one's subject list must be derived over the new
  package set before code moves.
- **Evidence.**

| Guard | The ruling it holds | Why a move escapes it |
|---|---|---|
| `TestEverySurfaceSeamIsClassifiedForTheAir` (`app/air_boundary_test.go:216-223`) | 0.16.0 D-91, the closed air boundary (K-6) | it reflects over `tty.Config`'s **top-level** func fields only; seams regrouped into a nested or embedded struct, or into an interface-typed field, are skipped and unclassified (`Config` today: 55 flat func fields, `modes/tty/dashboard.go:67`) |
| `TestTheMemoKeyCoversEverythingTheFrameShows` and the walk (`modes/tty/memo_completeness_test.go:147-188`) | F-30 / FR-3.3, 0.19.5 D-4 (K-17) | it descends one level into named structs, and embedded ones count as the model; state moved two levels down, into a type of another package, is never perturbed |
| `TestNoListComparesZips` (`app/logic_test.go:171`) | 0.19.5 D-8, one definition of "same place" | it reads exactly `.` and `../modes/tty`, not their subdirectories |
| `TestEveryFileIsInItsPackagesMap` (`cmd/watchpost/docs_test.go:351-353`) | 0.19.5 Q9.1 (JM-10), the packages' file maps | it reads exactly `modes/tty` and `app`; a new package's `doc.go` map is held by nothing |
| `TestTheAsBuiltMapIsKeptInStep` (`app/maps_test.go:62-65`) | diagrams move with the code (K-24) | it knows two build logs, `observer-maps` and `propagation-overlays`; 0.19.6's batches are not held |
| Package-directory scans: `TestNoProductionFileInAppReadsRuntimeGOOSDirectly` (`app/cast_test.go:722`), `TestNoProductionFileInAppBuildsANameOnlyVoiceSpec` (`:861`), `TestTheInjectorIsInEveryBuild` (`app/inject_test.go:20`), `TestGeodataLoadsOncePerEntryPoint` (`app/geodata_test.go:16`), `TestEveryLivePipelinesMethodIsReachedFromProductionCode` (`app/reachable_test.go:29`), `TestNoMapRequestBeforeTheListenerAsks` (`app/maps_basemap_test.go:158`), `TestNoTestSetsTheZoneMidRun` (`modes/tty/main_test.go:30`), `TestAnAgeIsReadAgainstAHeldClock` (`held_clock_test.go:18`), `TestMapTestsReachNoSocket` (`map_window_test.go:225`), `TestTheMapsFilesHoldNoColourLiteral` (`map_colour_guard_test.go:89`) | among others 0.18.0 NFR-3 (no map request before one is asked for), the find-only voice rule, the injector's build tags | each reads `.` or a glob in its own directory: a file moved to a subpackage leaves the scan |
| `tools/prepush` (`tools/prepush/main.go:28`) | the record's tests run before a push | it names `./app` for `TestTheAsBuiltMapIsKeptInStep`; the test moves with whatever package holds the map |

- **Consequence.** Principle 4. This is the 2026-09-09 failure in a new form: the rule was written
  down, and the gate that held it stopped looking. Every test stays green.
- **Recommendation.** First code batch, before any move: derive each subject list from the module.
  Walk `go list ./...` or up to `go.mod` (K-30), not a fixed pair. Each guard gets a planted control
  showing that it still fires on a file in a subpackage. RX and TG own the mechanics; this lens owns
  the list of rulings behind them.
- **Confidence.** High for the scoping, read in each test's source. Medium that no other guard has
  the same shape: the scan was by pattern (`os.ReadDir`, `filepath.Glob`, a literal package list).

### RC-4: Each new package is a P10 ratification, and the moved code counts as changed

- **Claim.** P10-05 counts the functions a branch changed since its merge-base
  (`code-standards.md:323-324`), and a moved function is a changed one. A new package has no ledger
  row, so it is judged at 2.0 until the HUM LEAD ratifies a floor. The packages hold 1,394
  (`modes/tty`) and 865 (`app`) production functions. Their floors were measured over 344 and 49
  changed functions.
- **Evidence.**
  - 0.19.0's precedent: "P10 found the new package at 0.50 checks a function: the code had moved
    from the window, under its density row, to a package with none"
    (`propagation-overlays/04-development/build-log.md:531`), then ratified at 0.94 (D-230, `:251`).
  - D-306 (`:327`) ratified a package of one-line helpers rather than let "the metric choose where
    code lives".
  - F-221 (`follow-ups.md:119`): the counter counts touched functions, not the package's.
  - Sixteen ledger rows are keyed on these packages' paths (`p10-ledger.md:18`, `:49-56`, `:63`,
    `:122`, `:129`, `:172`, `:190`, `:212-213`), and re-keying them is a ratification (K-21).
  - 0.19.5 D-27 (`rulings.md:52`): no guard is added for the count; floors go to the HUM LEAD at
    REVIEW exit.
- **Consequence.** Principle 3. Nothing is wrong with the code, but the cost of the ruling grows with
  the number of packages, and a split that chased density would add guards for the count, which D-27
  forbids.
- **Recommendation.** The package map states its number of new packages as a cost. PLAN brings every
  new floor and re-keyed row to the HUM LEAD as one list at the phase exit, as 0.19.0 D-264 did. P10
  is measured per batch so the list holds no surprises. When rows are re-ratified, the `app` row's
  reason is rewritten (RC-7, item 3).
- **Confidence.** High that the counter is the harness's and the precedents say so; the checker
  itself is out of tree and was not run.

### RC-5: AP-CITE-01 judges by directory, and the citation scheme depends on each line's history

- **Claim.** `tools/authoring` decides whether a citation is "new" by checking it against the IDs its
  **directory** held at the merge-base (`tools/authoring/cite.go:20-23`, `:82`, `known :=
  base[path.Dir(rel)]`). It does this for every comment in every `.go` file, tests included
  (`tools/authoring/main.go:99-101`). A file moved to a new directory starts with an empty set, so
  every bare ID it carries is reported. 0.19.5 D-11 rules that "the old citations stay, and the index
  resolves them". Separately, the record resolves an old bare citation through "the release whose
  commit wrote the comment, which `git log` on the file shows" (`where-things-happen.md:147-149`;
  `06_docs/ids.md:11-12`; 0.19.5 D-37). After a move, plain `git log` on the new path shows only
  0.19.6's squash commit.
- **Evidence.** A line-level approximation: 1,604 production and 2,432 total comment lines in
  `modes/tty`, and 742 and 1,180 in `app`, carry an ID of the tool's families with no release number
  on the same line. The tool qualifies by sentence across lines, so the true count is lower, and it
  is UNVERIFIED as an exact figure.
- **Consequence.**
  - Principle 3: `make lint-authoring` fails on the first moved file.
  - Principle 2: qualifying thousands of citations by hand is a large, error-prone edit, and getting
    each line's release right needs its history.
  - Without rename-aware history, the moved citations resolve to the wrong release.
- **Recommendation.**
  - Fix the tool in the first code batch, as an A-row (a tool fix keeping D-11's intent, like A-2):
    known IDs are judged by the moved file's own history, or module-wide at the base.
  - Moves are whole-file renames wherever possible, so `git log --follow` and `git blame -C` keep each
    line's origin.
  - `where-things-happen.md:147-149` and `ids.md:11-12` (its generator's text) name `--follow`.
  - TG owns the tool.
- **Confidence.** High for the tool's behaviour, read in source. Medium for the history consequence,
  reasoned from the squash-merge workflow and not run.

### RC-6: The documents that move with the code

Split into those a test forces into the same commit and those that would drift silently.

**Held by a test: a move fails until the document follows.**

| Document | What in it names a path or file | Held by |
|---|---|---|
| `docs/where-things-happen.md` | 183 `modes/tty/…go:Func` and `app/…go:Func` spans: the event table (`:12-76`), the vocabulary's "Defined in" column (`:88-138`), the rules table (`:174-181`) | `TestWhereThingsHappenNamesRealSymbols` (`cmd/watchpost/docs_test.go:62`) |
| `docs/extending.md` | 60 paths: the keymaps' files (`:21-30`); Walkthroughs 1, 3–6 and 8's file columns; the `go test ./modes/tty` and `./app` commands and the per-package golden, declset and cadence flags (`:257-262`) | `TestExtendingNamesRealSymbols` (`extending_symbols_test.go:249`) |
| `docs/architecture.md` | the mermaid's `TTY` and `APP` nodes (`:39-43`), the package table's `app/` and `modes/tty` rows with their files (`:87`, `:97`), the rules (`:66-75`) | `TestArchitectureNamesRealSymbols` (`:255`), and the atlas (`TestTheAtlasMatchesAFreshRender`) |
| `docs/accepted-costs.md` | `modes/tty/ticker.go`, `bench_test.go`, `dashboard.go:tickNeeded`, `modal_location.go:RecentCap`, `app/pipelines.go`, `broadcaster_alloc_test.go` (`:40-253`) | `TestAcceptedCostsNamesRealSymbols` (`docs_test.go:144`) |
| `modes/tty/doc.go`, `app/doc.go` | every non-test file, by row | `TestEveryFileIsInItsPackagesMap`, for those two packages only (RC-3) |
| `06_docs/p10-ledger.md` | 16 rows keyed on these paths | `TestEveryRatifiedP10RowNamesCodeThatExists` (`docs_test.go:288`); re-keyed only by ratification |
| 0.19.0's four living documents | `04-development/implementation-plan.md` 24, `03-architecture-design/architecture.md` 3 and `01-objectives/requirements.md` 1 `modes/tty` or `app` `file.go:Symbol` citations | `TestEveryCodeCitationResolves` (`record_test.go:271`), through `releaseRecord` (`:20`) |
| `06_docs/architecture-atlas.html` | regenerated from `docs/` and the as-built maps | `TestTheAtlasMatchesAFreshRender` |
| `06_docs/mutants/*.py` | 255 anchors (145 tty, 110 app) | `make mutant-anchors`, `make mutant-check` (K-23) |

**Not held: these drift silently.**

| Document | What names a path or file |
|---|---|
| `observer-maps/03-architecture-design/as-built-map.md` | `subgraph app["app/ — the composition root"]` with `maps.go`, `maplayers.go`, `mapfeed.go`, `mapgeometry.go`, `mapinview.go`, `history.go`, `mapquakes.go`, `mapradar.go`, `maptemp.go` (`:78-95`); `subgraph tty["modes/tty — the Observer"]` with `map_pane.go`, `map_describe.go`, `map_parts.go`, `map_prop.go`, `map_temp.go`, `map_boxes.go`, `status.go` (`:97-107`); `setup_notices.go` (`:48`); `mapprop.go`, `mapnames.go` (`:128-142`); `window_keys.go` (`:161`). Only its batch count is tested |
| `0.16.0-broadcaster-ui/03-architecture-design/as-built-air-and-deck.md` | `app/radio.go` (`:16`), `tty.Config` and `tty.Radio` as the boundary (`:124-127`), `app/air_boundary_test.go` (`:126`) |
| `map-ready-geometry/03-architecture-design/as-built.md` | `app/mapgeometry.go:resolveAlertAreas` (`:37`) |
| `0.16.0-broadcaster-ui/00-REQUIRED-READING.md` | `modes/tty/broadcaster_rail.go` as D-51's seam (`:204`), `app/arbiter.go` (`:39`, `:77`), `app/radio.go` (`:73`, `:149`) |
| `06_docs/follow-ups.md` | 19 paths (F-22, F-68, F-107, F-108, F-138, F-158, F-185 among them); only test names are checked (`docs_test.go:188`) |
| `06_docs/code-standards.md` | the `setupTable` specimen (`modes/tty/setup_rows.go`, `:340`), `app/dump_windows.go` (`:328`) |
| `06_docs/quality-plan.md:151`, `06_docs/perf-measurement.md:54`, `:222` | `modes/tty/golden_test.go`, `bench_test.go`, `modes/tty/timing.go`, `app/timing.go` |
| `README.md:597` | "`modes/tty` is the slowest package: about three and a half minutes under `-race`" |
| Go comments | 49 comments in 35 files name a `modes/tty/…go` or `app/…go` path, 15 of the files outside the two packages (`platform/lineup` ×7, `platform/snapshot` ×2 and others), plus every `ACCEPTED COST` site |

**What 0.19.6 has to add, which nothing requires today:**
- an `as-built*.md` of its own under `03-architecture-design/`, which the atlas will render (K-24);
- a row for it in `TestTheAsBuiltMapIsKeptInStep`;
- a `doc.go` file map in each new package, held by a derived list;
- a row in `docs/architecture.md`'s package table for each new package. Nothing fails if one is
  missing, because the test checks that names exist, not that the table is complete.

- **Consequence.** Principle 2: the documents a newcomer reads first are the ones that go stale.
- **Recommendation.** Every move batch carries both tables' rows for the files it moves. The
  unchecked documents get a check, or a line in the batch checklist. The as-built maps are redrawn
  in the batch, not at REFLECT (K-24).
- **Confidence.** High for the held set, read in each test. Medium for the counts, which are by grep.

### RC-7: Six places where the record and the code disagree today

1. **The windows are a stack; three documents say one window.**
   - `docs/where-things-happen.md:15`: "one `modal` value, so opening one closes the rest".
   - The vocabulary row (`:95`): "the one floating window that can be open".
   - The code: `modes/tty/dashboard.go:723` (`under []stackedWindow`, D-107) and `:1591-1596`: "THE
     WINDOWS ARE A STACK (D-106, D-107). A window opened from another opens over it, and closing it
     returns to the one below". The doc line just above, `:1590`, still says "open shows m alone".
   - 0.16.0 `00-REQUIRED-READING.md:190` (D-37) still gives the single value as the trap.
   - `docs/extending.md:291-294`'s recipe for a new window omits declaring its keys in
     `window_keys.go`, which `TestEveryWindowDeclaresItsKeys` requires (`as-built-map.md:161-166`).

   A test fails on that last one, so it is a recipe gap, not a silent one.
2. **"Three owners."** 0.16.0 `00-REQUIRED-READING.md:204` says "the layout has three owners now".
   The code says each width has one owner (`modes/tty/broadcaster.go:658-662`). The 0.19.5 build log
   corrected the comment because "the code has seven" (`0.19.5-quality-pass/04-development/build-log.md:1322`).
3. **The `app` P10 row's reason names functions that are gone.** `p10-ledger.md:122` cites
   "ReportOnce/resolveQuery/Save/RunSetup" and "the bubbletea model plumbing
   (Init/Update/View/advance — behavior pinned by 5 wizard flow tests)". In `app` only `ReportOnce`
   (`app/app.go:37`) and `resolveQuery` (`:165`) exist. No `Save`, `RunSetup`, `Init`, `Update`,
   `View` or `advance` is declared there. The row is checked for its key, not its reason.
4. **`app/doc.go:2-6` reads as a claim about `app`.** "It imports no domain and not the live surface
   (modes/tty); beside app, only … modes/report, … pkg/schema and four platform helpers." That is
   exactly `cmd/watchpost`'s import set (`go list`: app, modes/report, pkg/schema, httpx, invariant,
   snapshot, term), and "It" refers back to `cmd/watchpost`. But in `go doc ./app` the sentence sits
   in app's own package comment, and `app` imports 26 domain packages and `modes/tty`. The wording is
   ambiguous rather than false. JN's.
5. **"The release in flight" is 0.19.0.** `cmd/watchpost/record_test.go:19-20`: "releaseRecord is the
   release in flight's feature folder", set to `propagation-overlays`. 0.19.6 is in flight. The
   living-document checks hold 0.19.0's plan to the code, and 0.19.6's own required reading and
   architecture are held by nothing. TG's.
6. **The doc maps are a topic map.** `modes/tty/doc.go:33` maps `request.go` with the Broadcaster
   console, but every method in it is a `Dashboard` method (RC-2). It is not wrong as a list of
   topics. It is wrong as a guide to where a package line can fall.

- **Consequence.** Principle 2; the 2026-09-09 class. A package map built from these documents
  inherits their errors, item 1 above all.
- **Recommendation.** Correct items 1 to 4 and 6 in the first docs batch, before PLAN's package map
  cites them. Item 5 is a PLAN question for TG (re-point `releaseRecord`, or hold both records).
- **Confidence.** High. Each item was read at both ends.

### RC-8: The record already describes how to move code safely; `modes/tty` lacks the proof it describes

- **Claim.** Three moves in the record follow one pattern:
  - 0.9.5 Q2: an approved file map before any move; one package per commit; a declaration-set
    golden captured before and checked after; exemptions re-keyed and docs moved in the same commit.
  - 0.19.0 D-212's `platform/propanswers` extraction from `modes/tty`: the moved tests kept; 20
    mutants re-pointed and each one killed; density found and ratified; the architecture diagram and
    the atlas redrawn in the batch (`propagation-overlays/04-development/build-log.md:525-533`). Its
    "found moving it" is the shape to expect: "first written under `domains/propagation`, which the
    window may not import".
  - 0.19.5 D-13's rename: 41 mutants re-pointed.

  `app` has a declaration set (`app/declset_test.go`); `modes/tty` has none (`extending.md:260-261`
  lists `-update-declset` for `app`, `domains/weather/nws`, `platform/history`, `platform/render`,
  `platform/snapshot`).
- **Evidence.** As cited. And the 0.13.0 severe pass found `severe_table.go` "0 % covered in its own
  package" after it moved behind the render seam (`severe-alerts-modals/08-reports/red-team-build.md:51`):
  coverage is measured per package, so moved code can lose its measured coverage even though the
  tests still exercise it.
- **Consequence.** Principle 4. Without a declaration set for `modes/tty`, nothing proves that a
  31,346-line move lost, renamed or duplicated nothing.
- **Recommendation.** Capture `modes/tty`'s declaration set before the first move. PLAN adopts K-28's
  pattern by name. Coverage is reported with `-coverpkg` across the old package's successors at each
  move. RX owns the net.
- **Confidence.** High.

---

## Open questions for the HUM LEAD

1. **The P10 cost of the package count (RC-4).** Each new package needs a ratified floor, or 2.0, and
   moved functions all count as changed. Should PLAN present the new floors and re-keyed rows as one
   list at BUILD exit, as 0.19.0 D-264 did, or should each new package's row be ratified in the batch
   that creates it?

Everything else above is mechanical (the agent's, as A-rows) or belongs to the package-map
reconciliation (D-7).

## Touches other lenses

- **RM:** RC-1. The deck stays whole (K-3, K-4); F-22's single band writer could become structural
  through a package boundary.
- **TS:** RC-2. Observer's windows are forwarded to the console; `doc.go`'s rows are topics, not
  types; `request.go` and `broadcaster_manage.go` are `Dashboard` code.
- **AS:** K-6. A regrouped `tty.Config` escapes D-91's reflection.
- **RX:** RC-3 and RC-8. The guards scoped to one package; no declaration set for `modes/tty`;
  coverage measured per package.
- **TG:** RC-3, RC-5 and RC-7 item 5. AP-CITE-01 keyed by directory; `releaseRecord`;
  `tools/prepush`; the two-feature as-built test; the ledger re-keys; `lint-imports` holds no
  boundary between the new packages (A-4's PLAN question).
- **JN:** RC-7 items 1 and 4, where a newcomer is told the wrong thing today.
- **PB:** K-31. The accepted costs and allocation pins move with their code.
- **GP:** K-2, Charm through its interfaces.
- **Not in any lens:** 0.19.5's REFLECT recommendation 3 (scanning at the go 1.26.11 floor) "is
  asked at 0.19.6's DISCOVER" (`0.19.5-quality-pass/08-reports/reflect-report.md:148-151`;
  0.19.5 D-60). No 0.19.6 row records it yet.
