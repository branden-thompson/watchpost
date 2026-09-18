---
title: "0.16.0 Broadcaster UI — the REVIEW red team, verbatim"
date: 2026-09-17
phase: REVIEW
sev: SEV-0
authority: HUM LEAD
status: "Record. Eight blind reviewers on the four axes at 557a40f, then three remediation reviewers iterated to LGTM. Dispositions are in review-report.md and follow-ups.md."
---

# The REVIEW red team — reports as handed back

**Dispatched with `06_docs/red-team-brief.md`, blind, each in its own clone of `557a40f`.** Four axes;
the Code Quality axis was split into a whole-tree pass and four slices so that Necessity — the primary
lens — was read at file level. The remediation reviewers that follow were fresh agents on the fix
commits, one finding at a time, iterated to LGTM (`feedback-remediation-review-loop`). Reports are
reproduced as handed back, with only harness scratch paths removed; every disposition is in
`review-report.md` and `06_docs/follow-ups.md`.

## Code Quality — whole tree

# 0.16.0 Broadcaster UI — Distinguished Engineer review (v0.15.0..557a40f)

Mechanical: `make lint-authoring`, `go vet ./...`, `make dupes`, `make wires`, `make gate-controls` all green in a fresh clone. `go test ./app` FAILED once (TempDir cleanup race) and passed on rerun. Line numbers are repo-relative at 557a40f.

## 1. Understandability
- `startSchedule` takes eleven parameters; `bedSeams`' own doc admits it was "an eleventh and twelfth argument to a function that already had ten". Important. `app/schedule.go:60-76`. Simplify Y. One seams struct.
- `broadcaster.go` is 62% comment lines (1135/1802), `router.go` 56%; a reader filters rulings and dates to find the code. Minor. HUM LEAD's style call.
- Director core is readable: `Step` dispatches, one handler per event, effect set closed. Clean.

## 2. Maintainability
- Miles/km conversion has four owners while `near.go:43` claims "ONE OWNER". Important. `domains/locations/geodata/near.go:44`, `domains/radio/stream/within.go:66`, `app/bedrelay.go:99,231`. Simplify Y. One constant; `dupes` misses it (below its node floor).
- Top-off ranking comparator is written twice — once for the sort, once inside the invariant that re-checks it; a ranking change made in one place silently pins the old order. Important. `platform/lineup/topoff.go:161,185`. Simplify Y. One `less`.
- `histOther` duplicates 13 of `histPhrases`' 16 entries; `godocOpener` is a hand-typed verb list, so "X judges/refuses/records" fuses invisibly (`tools/gateoracle/oracle.go:15,62` use exactly those verbs). Important. `tools/authoring/rules.go:142-153,203`. Simplify Y.
- `maintrack.go` is a self-declared TEMPORARY env switch, default `off`, so relay-failure `NeedsRead` never reaches the Director in production; its removal (P3(d)) is "blocked on a ruling". Important. `app/maintrack.go:5-12`, `app/radio.go:626,645`. Fork for HUM LEAD: ship dormant, or rule P3(d) before SHIP.

## 3. Complexity
- `refreshCardWindow` re-renders detail rows slot by slot on every message including ticks, then `sameCard` renders both bodies again, though `cardInWindow` already finds the card by id. Minor. `modes/tty/router.go:1204-1213`, `broadcaster_detail.go:300-304`. Simplify Y.
- `report.Empty()`/`Full()` allocate a slice to answer a comparison. Minor. `platform/report/report.go`. Simplify Y.
- Four `cmd/watchpost` tests each build a fresh clone + stub + green run (158 s). Minor. `gates_test.go:453-481`. One oracle per package.

## 4. Necessity — removable elements (primary)
- Tautological invariants that assert what the line above just did: `director.go:855-861` (asserts the `Publish` it appended), `director.go:1037` (`State == Standby` after `To(Standby)`), `topoff.go:137` (`took <= need` after `break` at `>=`), `topoff.go:185` (`IsSortedFunc` on the sort's own predicate), `topoff.go:210` (`i < len` inside `for i := range`), `transitions.go:166`. Important — the mAA2 shape the sweep was supposed to purge. Delete Y, with their untestable `return nil` branches.
- Production code with no production caller, alive only via tests: `framed`, `railColumn`/`railColumnToned`/`railTone`/`bcRailLabel`/`bcRailForms`, `cardBoxWidth` (`modes/tty/broadcaster_rail.go:41-98,140,192-246`), `couldNotAsk` (`locate.go:163`), `consoleMemoCounts` (`broadcaster_memo.go:137`), `withSize` (`broadcaster.go:1799`), `treelock -status` (`tools/treelock/main.go:51,60-66`), `schedule.x` test-only field (`app/schedule.go:38`). Important. Delete Y; `trackArea:151` still reserves width for a rail nothing draws.
- Dead exemption: `g != "mutant-check"` is outside the registry `exemptions_test.go` exists to abolish AND dead — `ciGates()` parses the `name:` step. Important. `cmd/watchpost/gates_test.go:209-211`. Delete Y.
- `testFilesUnder` duplicates `docs_test.go`'s `testNames` with a second regex. Important. `gates_test.go:396-423`. Delete Y.
- Dead branches: `request.go:406-408` (`at != 0` after `at = 0`), `setup_rows.go:176`, second height cut `broadcaster.go:638-640`; zero-fill loops over fresh slices `broadcaster_slots.go:151`, `broadcaster_upnext.go:216`; `lookInPool` second bool (`app/pool.go:348-362`); `airScope.set` (`app/airscope.go:30`); `tableFor` one-line wrapper (`platform/render/table.go:551`); `Writers`/`Readers` counters duplicating `len()` (`tools/wires/main.go:82,102`); `itoa` (`rules.go:198`); `contains`/`sortedKeys` (`record.go:105,114`). Minor. Delete Y.
- Duplicate spellings of one rule: `deck.air` vs `monitorMayReachTheAir` (`app/dashboard.go:22`, `app/air_boundary.go:5`); `mainTrack()` re-implemented at `broadcaster_detail.go:58,107`; `throughToObserver` at `router.go:485,563`. Minor. Delete Y.

## 5. Name semanticism (SN-01)
- `proposeFrom`/`composeFor` name their parameter `watch` ("the listener's watched locations") but are wired to the STATION pool — the exact confusion D-72/D-140 fixed, preserved in the name. Important. `app/schedule.go:76,118,127,248,290`. Rename `pool`.
- `tools/wires/main.go:40-50` still calls `app/director.go` "the arbiter"; required reading: no such role. Minor.
- `withSize` also sets `ascii=true`; `keyState` returns a memo fingerprint 0-4; `toggleSevere` dispatches three windows. Minor. `broadcaster.go:1800`, `locate.go:283`, `dashboard.go:1366`.

## 6. Code documentation
`lint-authoring` is green, so everything here is the class it cannot see.
- WRONG: `app/compose_seam_test.go:103-104` says "A CLIENT THAT ANSWERS NOTHING"; `nws.New(client, "")` defaults to `https://api.weather.gov` (`domains/weather/nws/provider.go:69-71`). Important — see Q7.
- CONTRADICTS A RATIFIED DISPOSITION: `app/maintrack.go:10` and `app/radio.go:623-624` say `startSynth`'s path "retires at P3(d)"; the disposition sheet says Instruct — it stays. Important. HUM LEAD to say which is current; then fix the loser.
- Orphaned doc comments describing functions that do not exist or detached by a blank line (looks like lint placation): `reStationOnCommit` (`app/pool.go:279-285`), `tickerAlert` (`app/dashboard.go:428`), `narrateEvent` (`:912-917` vs `:1007`), `ForecastZone` (`provider.go:115-136`), `FeedLanes` (`domains/globalfeed/stack.go:118`). Important. Reattach or delete.
- Wrong: `isLookupFailure` claims a failed-build case it does not check (`app/pool.go:559-565`); `maintrack.go:64,75` "three stages" (two exist); `record.go:92-102` says "in order", code sorts; `broadcaster.go:632-635` "never truncated" under a truncation at `:612`; `rules.go:66-68` credits `histExempt` for the mutants skip `skipDir` does. Minor.
- History in substance (AP-HIST-01 regex missed): tombstones `broadcaster.go:578,1212`, `broadcaster_slots.go:212`, `broadcaster_pool.go:100`; "until now" narration `director.go:1029-1039`, `executors.go` Duck block, `tools/wires/main.go:404,459`, `tools/dupes` `fingerprintBody`. Minor. Lint pattern is narrower than the rule; say so in `code-standards.md`.

## 7. Testability
- A unit test reaches the public internet: the "answers nothing" fixture is a real `nws.New` over `api.weather.gov`; a cache-writer goroutine outlives the test and races `t.TempDir()` cleanup — observed FAIL on first run. Critical (it runs under `make race`). `app/compose_seam_test.go:108-112`. Stub the transport; join the writer.
- Request window drops non-ASCII: `len(key.String()) == 1` counts bytes; typing "Peña" yields "Pea" while Lookup (`modal_location.go:74`, `key.Text`) keeps it; 676 ñ/é names in `cities_trim.tsv.gz`, and hyper-local is the point. Important, untested. `modes/tty/request.go:415`. Use `key.Text`; add the test.
- shift+enter puts the station ON AIR behind an open Request/Help/About window: `toggleStation` gates on surface only. Important. `router.go:719-720,923-926`. No test, no ruling found — HUM LEAD fork.
- Producer pool is population-filtered nearest-first, so a short-range station's unprompted rotation reads the 25 biggest places (`domains/locations/pool.go:76-78`; admitted at `app/pool.go:399-403`). HUM LEAD fork, not mine.
- `platform/snapshot/assembler.go:326` reads `time.Now()` outside the seam. Minor.

## 8. P10 conformance
- P10-01: recursive `walk` closure in a package that annotates every loop bounded; `filepath.WalkDir` sits beside it. `cmd/watchpost/gates_test.go:399-418`. Minor.
- P10-04: raw lengths not in the ledger — `stationLine` 143 (`broadcaster.go:1101`), `startSchedule` 134 (`schedule.go:105`), `defaultTheme` 129 (`theme.go:271`), `keyAction` 131, `upNextBox` 105, `executors.run` 98, `speak` 89, `specimens_test.go:54` 393. The `p10` gate reports 0 live, so it measures something else; the metric and the count disagree and the ledger should say which counts. HUM LEAD to ratify rows or the metric.
- P10-07: swallowed read/stat errors `tools/gateoracle/oracle.go:118,155`, `identity_test.go:110-112`, `shell_test.go:57,69`. Minor.
- P10-02/05/09: bounded, annotated, clean.

## 9. Evidence soundness
- `lint-identity` is `go test -run PublishedTreeNames`; rename the test and it runs zero tests, exit 0. The alloc-budget selector got a guard (`gates_test.go:364`); this one did not. Critical. `Makefile:123`. Guard or drop `-run`.
- `mutant-anchors.sh` fails OPEN when the driver dies: I ran it with a mutant raising `SystemExit` beside a DRIFTED one — no `__COUNT__` line, `set -- $count` defaults to `0 0`, prints "0 mutant(s), every anchor still matches", exit 0. Important. `scripts/quality/mutant-anchors.sh:132-135`. Fail on a missing count or `total==0`.
- `specimens_test.go:55-57` skips ALL 112 specimens for one specimen's `/usr/bin/true`; `:443-448` counts `COULD NOT RUN` as caught. Important.
- `wires -json` on an empty walk exits 0 (silence check is in the text branch only); `authoring` prints "OK — 0 Go file(s)". Important. `tools/wires/main.go:159-170`, `tools/authoring/main.go:152-157`.
- `identity_test.go:107` exempts any path containing `testdata/`, unregistered. Important.
- `gates_test.go:112` ratifies `lint.sh`'s `|| true` fail-open (F-118) inside a test table — self-issued. Flag to HUM LEAD.

## 10. Evasion lens
- `release.yml:24` runs `make verify`; `verify-gates` now includes `p10` (`Makefile:299`), which fails loud without `a2dh` (`:504`); no step installs it; the gate-model tests read `ci.yml` only. The next tag's release job cannot pass. Critical. Either model `release.yml` and declare `p10` local, or install the CLI.
- No evasion found: no sourced scripts, no `-f`/`-C` hops, no `PATH` re-export, no CI predicate; `test-tags ./app` is justified and I verified `app/` is the only package with tagged tests; the `treelock` delegate is the recognised R4 form.

## Verdict
**Do not ship.** Fix first: `release.yml` + `p10` — the release pipeline is broken by construction at this commit. Then the two fail-open verifiers (`Makefile:123`, `mutant-anchors.sh:132`) and the network-bound unit test; the "Peña" defect and the ON AIR-behind-a-window fork go to the HUM LEAD with the evidence above.

## Code Quality — platform/ slice

Task: review the platform/ slice of v0.15.0..557a40f (lineup, render, report, config, debounce, bodymemo, snapshot, term, declset, singleowner, httpx) for Q1-Q8, Necessity primary. `go vet ./platform/...` and `go test ./platform/...` are green (20 packages).

**Q1 Understandability.** Director core reads well: Step is a thin dispatch, one handler per event, effect set closed. One trip: `director.go:713-714` carries two doc comments for `onFailed`, the first describing the pre-grade behaviour — a reader cannot tell which is current. Minor. Delete the stale line (Y).

**Q2 Maintainability.** `topoff.go:56-70` writes the ranking comparator twice (once for `SortStableFunc`, once inside an `IsSortedFunc` invariant); a change to the ranking must be made in both or the assertion silently pins the old order. Important. Hoist one `less` closure (Y). `lineup_table.go:139-146` and `:153-195` build `studs.ColumnDefinition` from `baseCol` in two places; a new column edits both. Minor (Y).

**Q3 Complexity.** `report.go`: `Empty()` allocates a slice via `Kinds()` to answer `s == 0`, and `Full()` allocates to answer `s == Everything()`. Minor, Simplify Y. `settle` (`director.go:840-843`) calls `takeTheAir` a second time after `prepareNext`; the comment justifies it but it is a hidden fixed-point loop of two — acceptable, noted.

**Q4 Necessity — deletable elements.**
- `director.go:855-863`: `last := fx[len(fx)-1]; _, published := last.(Publish); invariant.Check(published…)` asserts a fact established two lines above by the literal `append`. Tautology of the mAA2 shape. Important. Delete (Y).
- `topoff.go:64-70`: the `IsSortedFunc` invariant re-runs the sort's own predicate — it can only fail if `slices` is broken. Delete (Y).
- `topoff.go:40`: `took <= need` cannot be false given `if took >= need { break }` at `:29`. Delete (Y).
- `topoff.go:80`: `invariant.Check(i < len(d.settings.Watchlist))` inside `for i := range d.settings.Watchlist`. Delete (Y).
- `transitions.go:88`: `len(out) <= 2*len(view)` after at most two appends per element. Delete (Y).
- `table.go:551-553`: `tableFor` is a one-line wrapper with one caller (`tableForDated(…, nil)`). Delete (Y).
- `cadence.go:75-80` `lastReadOf` has one caller (`overdue`) and its own bounds check that `overdue:105` repeats; fold in (Y). Minor.
- `debounce.go` `Settled()`/`settled` field: one caller outside the package; verify it is load-bearing or drop (N until checked).
Each deleted invariant also removes a "// bounded" comment burden and an untestable branch (`return nil` on a check that cannot fail).

**Q5 Name semanticism.** `BreakOptima` (`term.go`) is opaque but ruled by `mock-card-types-v2.txt:25`; not a finding. `retry.go` `declined` is documented as "a ring, oldest evicted" (`director.go:462-465`) but is implemented as a slice re-sliced from the tail (`retry.go:99-101`) — a ring by behaviour only; Minor, rename the comment or the structure (N).

**Q6 Code documentation.** `director.go:1029-1039` narrates what `prepareNext` "REFUSED TO BUILD" before D-84 and what "the ruling overturns" — history in the code (AP-HIST-01 shape; lint-authoring did not catch it, so the lint's pattern is narrower than the rule). Minor. Rewrite to state the current rule only. Retry ring vs D-48's "nothing else" is ratified at `02-analysis/rulings-d59-d70.md:107` — clean.

**Q7 Testability.** Director is pure (`Step` takes an event, returns a Director and effects; no clock, no I/O — verified across director.go, operator.go, topoff.go, transitions.go, air.go, bed.go). Every handler has tests; property tests (`merge_property_test.go`) exist. The tautological invariants above are the one untestable class. `snapshot/assembler.go:326` reads `time.Now()` inside `Apply` — a clock outside the seam; Minor.

**Q8 P10.** D-45: `card.go:73` keeps `OnAir -> Discarded`, which `onFailed` needs; every operator path guards it (`operator.go:122`, `:196`) — conforming, not a defect. P10-02: every loop in changed lineup code is bounded and says so. P10-01: `takeTheAir` replaced mutual recursion with `for range 2` (`director.go:884`) — correct. P10-04 by raw 60-line count: `themes.go:43` (134, ratified in p10-ledger.md:83), `theme.go:271` (129, data literal, NOT in the ledger), `assembler.go:304 Apply` (79), `director.go:1000 prepareNext` (68), `operator.go:264 Insert` (65); excluding comments all are under ~45 code lines, and the `p10` gate reports 0 live — the metric and my count disagree, which the ledger's own `builtinOverrides` row already says. Recommend a `defaultTheme` row for consistency, HUM LEAD to ratify. P10-05/07: pervasive `invariant.Check` with checked returns. P10-03 not applicable in this slice.

Fix first: `topoff.go:56-70` duplicated comparator (a real drift hazard); the rest of Q4 is a mechanical sweep of five dead invariants.

## Code Quality — app/ + domains/ slice

Slice: app/ + domains/ (v0.15.0..557a40f). `go vet ./app ./domains/...` clean. `go test ./domains/...` green. **`go test ./app` FAILED on first run** (TestOnlyTheChosenKindsAreGathered/everything: "TempDir RemoveAll cleanup: directory not empty"); passed on rerun — flaky.

Required-reading checks: `startSynth` stays (app/radio.go:678, one caller). `giveWayLocked` still keys on source kind, not rail (domains/radio/player/engine.go:540-548). Text materialises at Standby (platform/lineup/card.go:564 invariant; app/executors.go build LocationReport). `app/director.go` still exists (dissolution owed since 0.14.0, acknowledged in RR — not new).

**Q1 Understandability**
- `proposeFrom`/`composeFor` take a parameter named `watch` and the doc says "the listener's watched locations", but they are wired to the STATION pool and `resolvable` (app/schedule.go:76,118,127,248,290; app/dashboard.go:305). The exact confusion D-72/D-140 fixed is preserved in the names. Important. Simplify Y. Rename to `pool`/`resolvable`, fix doc.
- `startSchedule` takes 11 parameters; `bedSeams` doc admits it was "an eleventh and twelfth argument to a function that already had ten" (app/schedule.go:60-76). Important. Simplify Y: pass one seams struct.

**Q2 Maintainability**
- Miles/km conversion has FOUR owners: `milesPerKM` (domains/locations/geodata/near.go:44, whose comment claims "ONE OWNER FOR THE CONVERSION"), `kmPerMile` (domains/radio/stream/within.go:66), literal `0.621371` twice (app/bedrelay.go:32, :99), plus `globalfeed.WithinMiles`. Important. Simplify Y: one owner (geo package), delete the rest; `dupes` gate missed it.
- `maintrack.go` is a self-declared TEMPORARY env switch (app/maintrack.go:5-12) whose removal (P3(d)) is "BLOCKED ON A RULING" (p3-build-log.md:237). Default `off` means relay-failure `NeedsRead` never reaches the Director in production (app/radio.go:626,645). Fork for HUM LEAD: ship with a dormant switch, or rule P3(d) before SHIP.

**Q3 Complexity**
- `deck.air` closure (app/dashboard.go:22) re-expresses `monitorMayReachTheAir` (app/air_boundary.go:5); two spellings of one rule. Minor. Simplify Y: delete air_boundary.go's 7-line file or use it in both places.

**Q4 Necessity (removable)**
- `lookInPool` returns two bools that are always equal (app/pool.go:348-362); caller discards one (:420). Delete the second return. Minor, Y.
- `schedule.x` field has no non-test reader (app/schedule.go:38,145). Test-only field on a production struct. Minor, Y.
- `airScope.set` is redundant with `radiusMi > 0` in every constructor (app/airscope.go:30,42,44). Minor, Y.
- `bedReach`/`bedAdvice` (app/bedfence.go) — 3 advice strings and a struct for one caller; fold. Minor, Y.

**Q5 Names** — covered by Q1 `watch`; `stationArea.followsDefault` and `transmitterOf` fine.

**Q6 Documentation (the class the lint cannot see)**
- Orphaned/detached doc comments: `reStationOnCommit` describes a function that does not exist (app/pool.go:279-285, referenced :143); `tickerAlert` comment with no function (app/dashboard.go:428-429); `narrateEvent` doc detached from its function by a blank line (app/dashboard.go:912-917 vs :1007); `ForecastZone` doc separated from its func by `ugcOfKind` inserted between (domains/weather/nws/provider.go:115-136); "EMERGENCY LEADS" paragraph detached from `FeedLanes` (domains/globalfeed/stack.go:118-120); `eventsFor` doc split by a blank line (app/executors.go ~:730). These blank-line detachments look like lint placation. Important. Action: reattach or delete.
- WRONG comments: `isLookupFailure` says "so is a resolver that failed to build" — code checks only ctx errors (app/pool.go:559-565). `maintrack.go:64` "one of the three" and `:75` "Dark and live both do" — only two stages exist. `civil.go:8-12` rewrite now states a hypothetical as current. Minor-Important.
- History narration surviving the lint: executors.go Duck block ("corrected 2026-09-15... The comment was not revisited"), `onTheRail` ("for one release"), build LocationReport ("This decline read ... from 0.14.0 until now"), speak ("admitted here for one release and it was wrong"). AP-HIST-01 shape. Minor.

**Q7 Testability**
- app/compose_seam_test.go:89-139 builds a real `nws.New(client, "")` and hits api.weather.gov from a unit test; a cache-writer goroutine outlives the test and races `t.TempDir()` cleanup → flaky FAIL. Critical for the gate (make race runs it). Action: stub the transport; wait for the writer.
- Producer pool is population-filtered nearest-first (domains/locations/pool.go:76-78; admitted at app/pool.go:399-403 "a pool ordered that way can never surface [Rainbow]"). For a short-range station the unprompted rotation reads the 25 biggest places. HUM LEAD fork, not mine.

**Q8 P10**
- Line-length hotspots: `startSchedule` 134 lines (app/schedule.go:76), `executors.run` 98 (app/executors.go:260), `speak` 89 (:418), `tickerDeck.cycle` 75, `ttyConfig` 74, `startPipelines` 73, `FireSegments` 72 — none in p10-ledger.md; the checker counts statements so cite as review, not gate failure. P10-02 bounds annotated throughout; P10-07 swallowed `config.Mutate` errors are commented (app/pool.go:159,174). No recursion found.

## Code Quality — modes/tty slice

Task: modes/tty slice of v0.15.0..557a40f, Q1-8, Necessity primary. `go vet ./modes/tty` clean, `go test ./modes/tty` ok, gofmt clean. Two probe tests were run in the scratch clone only.

**Q4 Necessity (primary)**
- Production-dead code kept alive only by tests. Severity: Important. Evidence: `framed` broadcaster_rail.go:246; `railColumn`/`railColumnToned`/`railTone`/`bcRailLabel`/`bcRailForms` broadcaster_rail.go:41-98,192-226; `cardBoxWidth` broadcaster_rail.go:140; `couldNotAsk` locate.go:163; `consoleMemoCounts` broadcaster_memo.go:137; `withSize` broadcaster.go:1799. No non-test caller for any. Delete? Y. Action: delete; move test-only helpers into _test files. Note `trackArea` (broadcaster_rail.go:151) still reserves `bcRailWidth+bcRailGap` for the rail nothing draws, so `priorityWidth`/`upNextWidth` inherit a dead layout.
- `clamp` cuts height twice; the second cut (broadcaster.go:638-640) cannot fire after the pad at 626-630 makes len==height. Minor. Delete? Y (one cut plus its comment).
- Zero-fill loops over freshly made string slices: broadcaster_slots.go:151-154, broadcaster_upnext.go:216-219. Minor. Delete? Y.
- One-caller wrappers: `requestRows` (request.go:73, alias of `report.All`), `note` (154), `requestHelperWidth` (167), `headerRows` (broadcaster_header.go:40), `minSize` (broadcaster.go:551, returns two consts), `chromeAt`/`chrome` (broadcaster_rail.go:264,268 over `railed`). Minor. Delete? Y.
- Duplicates of existing helpers: broadcaster_detail.go:58-61 and 107-110 re-implement `mainTrack()` (broadcaster_air.go:135); router.go:485-488 and 563-568 re-implement `throughToObserver` (826-832), the latter behind `if …; true {`; `centre` closure twice, broadcaster_manage.go:45-47/74-76. Minor. Delete? Y.
- Dead branches: request.go:406-408 tests `at != 0` right after setting `at = 0`; setup_rows.go:176 `case scopeUnruled: return true` before `return true`; `cutBed`/`stepBed` re-check the surface (router.go:963,979) that `consoleOwnsTheKeys` settled at 738. Minor. Delete? Y.

**Q7 Testability / behaviour found by probe**
- shift+enter puts the station ON AIR behind an open Request, Help or About window: `keyAction` router.go:719-720 → `toggleStation` 923-926 gates on surface only; `consoleOwnsTheKeys` is not consulted. Probe: all three windows → `GoOnAir` called. No test, no ruling found in the release docs. Important. Fork for HUM LEAD: ruled or defect.
- Request window Location field drops non-ASCII: request.go:415 `len(key.String())==1` (bytes) where modal_location.go:74 uses `key.Text`. Probe: typing P,e,ñ,a yields "Pea" in Request, "Peña" in Lookup. `geodata/data/cities_trim.tsv.gz` has 676 ñ/é names — hyper-local places are the point. Important. Untested. Action: use `key.Text`, add the test.

**Q6 Documentation** (lint-authoring is the parent's run; this is the class its regex misses)
- Tombstones for deleted symbols: broadcaster.go:578 (`laneWidth IS GONE`), broadcaster_slots.go:212 (`flatBody RETIRED`), broadcaster_pool.go:100 (`fillPoolWeather RETIRED`), broadcaster.go:1212 (`THE BED'S OWN ROW IS GONE`); "It was"/"until now" narration at broadcaster.go:48,72,85,110,1273,1399, broadcaster_air.go:105, broadcaster_rail.go:122-126. AP-HIST-01 by its own text. Minor. Delete? Y.
- WRONG comments: broadcaster.go:578-583 names `cardBoxWidth` a live width owner and broadcaster_rail.go:136-139 calls it "the ONE place that arithmetic lives" — no production caller. broadcaster.go:632-635 says height was "never truncated" while 612-614 above truncates it. Minor.

**Q5 Names**: `withSize` also sets `ascii=true` (broadcaster.go:1800). `keyState` returns magic 0-4 as a memo fingerprint, named like keyboard state (locate.go:283-295). `toggleSevere` now dispatches debug/relay-fault/severe (dashboard.go:1366-1380). Minor.

**Q1 Understandability**: broadcaster.go is 62% comment lines (1135/1802), router.go 56%; the code is short and legible once stripped, but a reader must filter rulings, quotes and dates to find it. Minor; HUM LEAD's style call.

**Q2 Maintainability**: Router mirrors facts both ways each message (router.go:431,440,459) — one line per shared fact, growing. Minor.

**Q3 Complexity**: `refreshCardWindow` (router.go:1204-1213) renders detail rows slot by slot on every message including ticks, then `sameCard` (broadcaster_detail.go:300-304) renders both bodies again; `cardInWindow` (1147) already finds the card by id. broadcaster_pool.go:49-53 converts km→mi→km. Minor.

**Q8 P10**: no unbounded loops, no recursion, vet/gofmt clean. Lengths not in the P10-04 ledger: `stationLine` 143 lines (broadcaster.go:1101), `upNextBox` 105 (broadcaster_upnext.go:80), `requestBody` 91 (request.go:193) — `keyAction`/`dispatch` are ratified, these are not; either the gate measures cyclomatic only (say so) or these need rows. Minor, HUM LEAD.

Fix first: request.go:415 (the non-ASCII drop); then the ON AIR-behind-a-window fork.

## Code Quality — tools/ + gate tests slice

**Slice: tools/* + cmd/watchpost gate tests.** `go test ./tools/... ./cmd/...` green (gateoracle 151s, cmd/watchpost 158s); `go vet` clean; all four self-tests pass.

**Q1 Understandability** — Clean. Headers state the rule and the defect it closes; the oracle's doc.go is a model. One trap: `record.go:92-102` doc says "deduplicated, in order" but `readLog` SORTS (line 101) and `run.reach` says "sorted" — the comment is wrong. Minor. Fix comment.

**Q2 Maintainability** — `godocOpener` (`tools/authoring/rules.go:203`) is a hand-typed verb list; a fused doc whose opener is "X judges/refuses/records/lays out" (verbs this very tree uses: `oracle.go:15,62`) is invisible to SN-02. Important. Match `^// Name ` + any lowercase word, or derive verbs from the tree. `histOther` (`rules.go:146-153`) duplicates 13 of `histPhrases`' 16 entries verbatim: two lists free to disagree, in the file whose job is refusing drift. Important; Simplify Y: build both from one slice, delete `otherHistPhrase` (`rules.go:142`).

**Q3 Complexity** — `cmd/watchpost/gates_test.go:453-481` builds a fresh clone + stub + every gate's green run FOUR times (one per Test). One oracle per package (`TestMain`/`sync.Once`) halves the 158s. Minor.

**Q4 Necessity (removable)** —
- `gates_test.go:209-211` `g != "mutant-check"` is an exemption outside the registry `exemptions_test.go` exists to abolish, AND dead: `ciGates()` (`gatemodel_test.go:411`) parses the `name:/if:/run:` step, so mutant-check IS in `ci`. Important; Delete Y.
- `gates_test.go:396-423` `testFilesUnder` and `docs_test.go` `testNames` walk the same tree for the same `func Test…` regex, two spellings (`Test\w+` vs `Test[A-Za-z0-9_]+`) — the "two regexes for one idea" `gatemodel_test.go:5-12` condemns. Delete `testFilesUnder`; `TestTheAllocBudgetSelectsItsPins` needs only names. Important.
- `contains` at `gateoracle/record.go:105` and `gates_test.go:442`, `sortedKeys` at `record.go:114`: go 1.25 module; `tools/wires/main.go:234` already uses `slices.Sorted(maps.Keys())`. Delete both. Minor.
- `tools/authoring/rules.go:198-199` `itoa` wraps `strconv.Itoa` with a false reason (strconv is already imported). Delete.
- `tools/treelock/main.go:51,60-66` `-status`: no caller in the tree; `-check` answers the same question with an exit code. Delete.
- `tools/wires/main.go:82-83,102-103` `Writers`/`Readers` ints duplicate `len(WriteAt)`/`len(ReadAt)`. Delete the counters.
- `tools/dupes/main.go` ~264-268: the four-line "name a person would write" comment survived extraction into `funcName` and now duplicates its doc. Delete.
- `artifacts_test.go:117` `isExecutable(t, …)` takes `t` only for `Helper()`. Drop the parameter.

**Q5 Names** — `wires/main.go:40-50` header still says "the arbiter" for `app/director.go`, which the required reading says has no such role. Minor. `treelock -status` vs `-check` (above).

**Q6 Documentation** — lint-authoring passes, so these are the class it cannot see: `dupes/main.go` `fingerprintBody` doc (~144-152) describes REMOVED parameters ("IT TOOK A FileSet AND SRC AND READ NEITHER"); `wires/main.go:404-406,459-462` narrate a refactor ("measured 26 against P10-04", "the EIGHTH collision"); `dupes` `reportText` doc "separated from main at the length ceiling". All AP-HIST-01 in substance. Minor; rewrite as current-state. `authoring/main.go:158-159` prints a scope that omits the `authoring` skip at `main.go:48`. `rules.go:66-68` claims `histExempt` exempts mutants; it does not (skipDir does).

**Q7 Testability** — Tools have no `_test.go`; self-tests carry the burden and do discriminate both directions. `exemptions_test.go:116` hands a nil `*testing.T` to table funcs; `fileExists` calls `t.Helper()` on it and would panic under a specimen. Minor.

**Q8 P10** — `gates_test.go:399-418` recursive `walk` closure: P10-01, in a package annotating every loop "bounded (P10-02)"; `docs_test.go` uses `filepath.WalkDir` beside it. `specimens_test.go:54` 393-line function, `gateattacks_test.go:78` 119, `gatemodel_test.go:499` 65: P10-04. `oracle.go:118,155`, `identity_test.go:110-112`, `shell_test.go:57,69` swallow read/stat errors: P10-07.

**Q9 Verifiers fail closed** —
- `Makefile:123` `-run PublishedTreeNames` has no counterpart to `TestTheAllocBudgetSelectsItsPins` (`gates_test.go:364`); rename the test and `lint-identity` runs zero tests, exit 0. Critical. Guard it (or drop `-run`).
- `specimens_test.go:55-57` `t.Skip` on missing `/usr/bin/true` skips ALL 112 specimens for one specimen's dependency. Important. Skip E3 alone.
- `specimens_test.go:443-448`: CAUGHT is `fired`, and `Fatalf("COULD NOT RUN…")` also fires — a broken oracle (old make, failed clone) counts every `caught:true` row as caught. Important. Refuse verdicts whose `said` begins COULD NOT RUN/UNJUDGEABLE.
- `wires/main.go:159-170`: the "silence is a verdict" check sits inside the non-JSON branch; `-json` on an empty walk exits 0. Important. Hoist it.
- `authoring/main.go:57-59,72-73,152-157`: walk errors and unparseable files are swallowed; zero files prints "OK — 0 Go file(s)" exit 0. Important. `files == 0` must fail.
- `identity_test.go:107` `strings.Contains(p, "testdata/")` exempts any path with that segment, unregistered and unaudited. Important. Make it a table row or narrow to `/testdata/` fixture types.
- `gates_test.go:112` ratifies `lint.sh`'s fail-open (F-118) inside a test table — self-issued, not HUM LEAD; a required gate (`lint`) is declared unable to fail closed. Flag to HUM LEAD.

**Fix first:** `Makefile:123` unguarded `-run`. Then the `mutant-check` dead escape.

## Project Hygiene

# 0.16.0 review — `557a40f`, read-only clone

Every gate I could run from the tree passed in a fresh clone: fmt, vet, vet-tags, test-tags, tidy, lint, lint-imports, lint-watermark, lint-identity, lint-authoring, gate-controls, alloc-budget, dupes(+selftest), wires(+selftest), mutant-anchors (379/379), treelock-selftest, `go build ./cmd/watchpost`, and `go test ./cmd/watchpost` (the four oracle gates included). Not run: race, p10, mutant-check, vuln, release-matrix, install-test.

## Q1 — Anything in the tree that should not be

Tip is clean: no tracked binary, no harness file (`CLAUDE.md`/`AGENTS.md`/`_a2dh` untracked), no secrets, `docs/accepted-costs.md` is 15 KB, the only executable under `06_docs/` is `06_docs/mutants/run.sh`, ledgered at `cmd/watchpost/shell_test.go:46`.

**F1. The 15 MB Markdown blob is still in the branch history.** `6b1b621` introduced blob `d57a9ac6` (15,235,221 B, `docs/accepted-costs.md`); `e956747` fixed the file but the object ships with the push. `handoff-0.16.0.md:33` claims "zero objects over 400 KB in the whole release range" — true when written, false now. Severity: Important. Simplify/Delete: Y. Action: the ruling at `handoff-0.16.0.md:31` says unpublished history is cleaned; rewrite `6b1b621` before push or accept the blob forever, and re-state line 33.

**F2. Home-relative personal paths survive `lint-identity`.** `06_docs/02_features/multi-voice-support/07-readiness/agent-uat-p1.md:45` and `.../uat-p4.md:19` carry `cd ~/<a personal desktop path>`. `cmd/watchpost/identity_test.go:40` only matches `/Users/...`. Severity: Minor. Action: scrub both lines; add a `~/` class to the pattern.

## Q2 — Build from a clean clone

Yes, every gate other than `p10` runs from the tree (list above). But:

**F3 (Critical). The release workflow can no longer pass.** `.github/workflows/release.yml:24` runs `make verify`; `Makefile:299` now lists `p10` in `verify-gates` (it was not at v0.15.0, `git show v0.15.0:Makefile` line 231); `Makefile:503` exits 1 without `a2dh`; nothing installs `a2dh` on the runner. The next `v*` tag fails at verify by construction. `README.md:362-365` documents the stop "by design" for a clean clone, and the release runner IS a clean clone. The oracle cannot see this (stubs put `a2dh` on PATH, `tools/gateoracle/doc.go:14-20`), and `gates_test.go:46` states "CI has no a2dh" while `release.yml` is CI. Simplify/Delete: N. Action: HUM LEAD ruling — either release.yml runs `verify-gates` minus `p10` via a declared `releaseOnly` exemption, or `p10` leaves `verify` and becomes a phase-exit gate. Do this before tagging.

## Q3 — Do the claimed gates run

The three lists agree; every difference is declared (`gates_test.go:31-104`) and `TestTheThreeGateListsAgree` passes here. `ci.yml:48-95` is one step per gate, unsilenced. `mutant-check` is conditional on `MUTANT_POLICY=push` (`Makefile:423`) and Linux (`ci.yml:88-95`), declared at `gates_test.go:96`.

The oracle's claim is stated honestly: `tools/gateoracle/ceiling.go:5-16` prints the not-judged list on every pass; `gate-attack-list.md:464-480` names drift vs evasion and admits evasion is unbounded; `gatemodel_test.go:487-493` labels the CI half "a ratchet, not a proof". One overstatement: `gate-attack-list.md:522-527` (round nine) says the p10 absence case "PASSES"; it does, as a gate, and the passing gate is what breaks F3. Absence-loud was proven; absence-in-release was never asked.

## Q4 — Owed items that stopped being true

**F4. The build report's gate table is stale against its own header.** `08-reports/build-report.md:73` says "the 16 gates required-gates.txt names" (it names 24, `required-gates.txt:14-43`); rows 74-76 quote 355/353/2 while line 61 says 379/374/5; line 85 says "The two survivors owe nothing" (there are five); line 79 says wires 91 (live: 92); line 83 quotes P10 "66 findings, 42 unratified" and line 98 says "P10 is not in verify", contradicting `Makefile:299` and `handoff-0.16.0.md:19` (0/0/0, 152 rows — the mirror says 153, `p10-ledger.md:12`). Evidence cited is `dist/verify-remediated3.log` and `dist/p10.json`; `dist/` is gitignored (`.gitignore:1`), so the evidence is not in the record. Severity: Important. Delete: Y (the table). Action: regenerate from `557a40f` logs promoted under `07-readiness/`.

Otherwise clean: 153 ledger rows all name existing files; the shell ledger covers every scripted file in the index; the five survivors in `mutant-verdicts.log:18,37,210,211,357` match the by-design set at `gates.md:567-568, 654-656`; no TODO/FIXME in Go. F-58..F-63 remain open from 0.15.0 with no 0.16.0 disposition (`follow-ups.md:168-174`) — noted, not excused.

## Q5 — Record vs intent

Mostly what happened: the sweep log matches its quoted figures; only two test files changed after the sweep commit (`git diff e956747 557a40f -- '*.go'`); handoff-cited commits (`d7523e6`, `1ed3a93`, `fa022cb`, `8a5c121`, `c48e4a9`, `59c8b38`) exist on the branch.

**F5. The scope commit's handoff says do not ship.** `handoff-0.16.0.md:3` at `557a40f`: "Status: BUILD, not exited. Do not ship." The commit after (`aaa5721`) rewrites it to BUILD EXITED. The record at the reviewed commit and the commit message disagree. Severity: Minor. Action: none if `aaa5721` is the ship candidate — but then say so.

**F6. `mutant-verdicts.log` names no commit.** `07-readiness/mutant-verdicts.log:1-5` carries an ETA of "~18 min" and no tree hash; the 4 h 46 min / `e956747` attribution exists only in prose (`gates.md:527`). Severity: Minor. Action: have `scripts/quality/mutant-verdicts.sh` print `git rev-parse HEAD` into the log header.

## Q6 — The push and the history

Facts from the source repo: `origin/feature/0.16.0-broadcaster-ui` = `66dc88d` (2026-09-12); merge-base with local = `3a13fd2`; local is 283 ahead / 87 behind; `v0.15.0`/`origin/main` (`fd761ab`) is an orphan publish line, not an ancestor. **The 87 published commits carry two `wires` binaries (4.4 MB each, in `8b46b69`, `bee02cc`) and are absent from local — so local already rewrote published history** (the first rewrite, tag `backup/pre-blob-removal` `53a33b7`, 2026-09-15, which `handoff-0.16.0.md` never mentions; it describes only the `authoring` rewrite, `handoff-0.16.0.md:30-34`). `handoff-0.16.0.md:36-38` ("they cannot reach a remote") is wrong: the remote already holds the `wires` blobs, and the `authoring` blobs never were pushed.

**F7 (Critical, process).** "Published history is never rewritten; fix forward" means the local branch cannot be force-pushed without breaking the ruling; there is no fast-forward path. Order for a maintainer:
1. Touch nothing destructive — keep `pre-blob-rewrite`, `refs/original/...`, `backup/pre-blob-removal`.
2. Fix F1 (or accept it) and F3 first; both change what gets published.
3. HUM LEAD ruling: is `origin/feature/...` "published"? If yes: `git merge -s ours origin/feature/0.16.0-broadcaster-ui` onto local, push as a fast-forward, and accept the `wires` blobs forever. If the feature branch is ruled unpublished: `git push --force-with-lease=refs/heads/feature/0.16.0-broadcaster-ui:66dc88d`, recorded as a ratified exception.
4. Re-run `make lint-identity` and `TestNoTrackedBinaries` on the pushed tip; read the CI log, not the notification.
5. Only then the cleanup at `handoff-0.16.0.md:40-44`, plus `git tag -d backup/pre-blob-removal`.
6. Do not tag `v0.16.0` until F3 is fixed; the tag triggers the failing verify.

## Verdict

**Do not ship.** The product tree is in good shape and the local gates genuinely pass, but the release path is broken by the project's own gate (F3), and the push cannot honour the project's own ruling (F7). Fix first: **F3** — the release workflow, because every other step ends at a tag that fails.

## Docs Quality

# 0.16.0 document-quality review — `557a40f`, read-only clone

Scope as briefed. Every claim below was checked in the clone; `go run ./tools/authoring` and `python3 scripts/quality/exposure-scan.py` were run; no `make verify`.

## 1. Does each document answer its title?
Mostly yes. `gates.md`, `where-things-happen.md`, `accepted-costs.md`, `code-standards.md`, the two P3 records and the required reading do. Two do not:
- **`build-report.md` answers "what is the state of BUILD" three different ways at once** (see §2).
- **`handoff-0.16.0.md:3`** says "BUILD, not exited. Do not ship" and `:167` says the sweep "was deliberately not run", while `:20` reports the sweep run and the build report recommends REVIEW. A handoff that disagrees with itself does not hand off. Severity: Important. Simplify: Y (strike the stale items). Action: rewrite §1/§4 to the 557a40f state.

## 2. Contradictions with the code or another document
**Gate count — four figures, none agree.** `build-report.md:6,:73` "16 gates"; `docs/extending.md:151` "21 of the 23"; `handoff-0.16.0.md:17` "24"; `06_docs/required-gates.txt` has 24 and `Makefile:299` runs 22. Severity: Important. Simplify: Y. Action: cite `required-gates.txt` by count in one place; delete the prose numbers.

**Mutant figures inside one report.** `build-report.md:6` "corpus 358"; `:74-76` "355 — 353 CAUGHT, 2 SURVIVED"; `:60` "379 — 374 / 5"; `:86` and the FR-11.4 row still say "the two survivors". The log agrees with 379/374/5 (`mutant-verdicts.log:400`, survivors m16, m43, mBM1, mBM2, mSC3). Severity: Important. Simplify: Y. Action: one figure, one table row.

**P10.** `build-report.md:83` "66 findings … 42 unratified" vs `handoff-0.16.0.md:19` "0 unratified, 152 ratified" vs `p10-ledger.md:14` "153 rows". Severity: Important. Action: re-run and publish one number with its date.

**README privacy sentence vs the radio log.** `README.md:186` "never … written to a debug dump". `app/radio.go:640` writes `needs-read … ref=<lat,lon>` (`snapshot.Key`) to `<cache>/watchpost/debug/radio.log` (`README.md:352`), and the transmitter is itself a pool member (`TestAPoolWithNoFenceIsJustTheStation`). FR-9.4's gate (`app/dump_test.go:221`) covers `app/dump.go` only; `setup_form.go:244` makes the same promise. Severity: **Critical** (a privacy claim on the designer/PM page, repository public). Simplify: N. Action: HUM LEAD fork — narrow the sentence to "the diagnostic dump", or redact `ref=` in the radio log and gate it.

**Requirements not amended with the rulings.** "Ten cards/slots" at `requirements.md:49,:58,:76,:308` — cap is fifteen (`broadcaster_lanes_test.go:88`, `gates.md:53`); RS-5 `:354` "cap, population-ordered" and RS-7 `:356` cite FR-8.3 for a cap that lives in FR-8.6, which was amended on 2026-09-17 to nearest-first; FR-10.1 `:270` superseded by D-56. Severity: Minor. Action: amend as FR-8.6 was, dated.

**Stale `extending.md`.** `:3-6` says a second top-level view "was not built … when [it] arrives"; it arrived (`modes/tty/router.go`). `:15` "`defaultKeyMap()` is the only place a key is named" — `broadcasterKeyMap()` at `router.go:162`. Severity: Minor. Simplify: Y.

**Citation drift.** `gates.md:622` cites `gates.md:489`; the row is `:492`. Glossary rows `requirements.md:308-314` cite `lineup.go:171-188` (`held`), `card.go:233` (`handsBack`), `plan.go:188` (`TrackedAs`), `executors.go:113` (`alert`) — none is the thing named. Severity: Minor.

**`where-things-happen.md:110`** — an Event row ("A location is typed…") sits inside the Record-IDs table; `:14` "A key is pressed → `dashboard.go:handleKey`" omits the Router; the Vocabulary (`:70-97`) has no card, line-up, rail, bed, fence or Director. Severity: Minor.

## 3. Unverified claims in verified-reading documents
**Traceability "60/60, none untraced" (`build-report.md:277-341`).** The method — "a test file that cites the ID" — greps FR IDs across all test files, and IDs collide across releases. Sampled ten "TRACED" rows; five cite another release's requirement: FR-1.1 → `platform/config/one_writer_test.go:10` (config write atomicity); FR-4.4 → `modes/tty/severe_test.go:854` (fabricated-event mark, "HUM LEAD 2026-09-07"); FR-6.4 → `relayfault_test.go:716` (a ten-second clock); FR-9.2 → `app/cast_test.go:518` (a fault seam); FR-9.3 → `platform/lineup/bed_test.go:280` (a tune that never lands). The other five (FR-3.2, 4.1, 6.3, 8.10, 9.4) hold. Severity: **Critical** — the release's completeness sentence rests on the method. Simplify: N. Action: scope the derivation to this release's test files or to citations carrying "0.16.0"; re-derive; re-state the count with the collision blind spot beside it.

**Sweep figures.** 379/374/5/0 match `mutant-verdicts.log` exactly; the five survivors are the by-design set. Clean.

**Exposure table.** `exposure-statement.md:24-31` is exact at `59c8b38` (re-run: identity 137/339, host 12/28, path 4/5). At `557a40f` the same scanner reports identity **224/645**, host **18/34**, path **5/6** — because `:90` writes `/Users/you/…` and `exposure-scan.py:70-71` harvests every `/Users/<x>` in the tree as a username, so "you" now matches prose everywhere. The statement's own text moved its numbers, four commits before the tip it is judged on. Its prose also disagrees with its table: `:35` "+64 files" vs "+73"; `:46` "+93" vs "+114"; `:76` "true on 2026-09-15" vs `:14` "2026-09-17". Severity: Important. Simplify: Y. Action: spell the example as `~/…`, re-run at the tip, fix the three deltas.

**The live UAT.** `p3-uat.md:7,:10` say the live UAT "is `modes/tty/broadcaster_uat_test.go`". That file is ten rendering tests for one `TruncateCells` root cause (`:12-30`); none is the ten audio signal cases (`p3-uat.md:64-75` — dead air, two voices, thirty-minute rotation). No document records those cases run against the live console; F-144 was closed on this pointer (`follow-ups.md:255`). Severity: Important. Action: run and record cases 2, 6, 10, or state that they were not.

## 4. Audience order
`README.md:175-188` reads for a designer/PM: plain words, the STANDBY/ON AIR meaning, the storage boundary. Two issues: the privacy sentence above, and the section names promote/demote/drop and the bed cut-over but not one console key — the Observer sections each carry their key table. A HUM LEAD layout fork, presented with evidence: `router.go:164-221` holds 15 console bindings; the README shows two. It does not claim the `[shift+U]` discard-pile modal, which is unbuilt (`follow-ups.md:187`, F-108). Severity: Minor.

## 5. Numbers without blind spots (INST-5)
- The sweep line `build-report.md:60` and `gates.md:459` carry survivors-by-design but not the sweep's own recorded blind spot — a detector that needs `-race` reads as SURVIVED, "not handled" (`gates.md:578-590`).
- P10 "42 unratified" and the 60/60 table carry none.
- `exposure-statement.md:96` states one only for `path`.
- Good examples: `accepted-costs.md:234` (pins 520/9760 verified at `broadcaster_alloc_test.go:215-216`), `tools/authoring`'s banner, `code-standards.md:145-148` — though `code-standards.md:3` "every rule is checked by a tool" sits beside F-137's ~123 undetected AP-HIST-01 lines deferred to 0.16.5 (`follow-ups.md:248`; e.g. `modes/tty/broadcaster.go:1102-1105`). Severity: Minor. Action: name the phrase-list ceiling next to the claim.

## 6. The required reading
Every file/line/quote citation lands — `director-charter.md:172-173`, `director-build-log.md:809,:1086`, `app/radio.go:875-876`, `app/director.go:24`, `lineup.go:9-12`, `app/air_test.go:40`, `app/radio.go:1328`, `broadcaster.go:565`, P-1..P-10 at `p3-flip-postmortem.md:112-121`. Four near-misses: `:230` "glossary, risk register" are sections (`requirements.md:301,:346`), not documents, and F-34 lives in `follow-ups.md:139`, not `01-objectives/`; `:201` "`laneWidth()` is that seam" — `broadcaster.go:578` says "laneWidth IS GONE (D-80)"; `:187` presents D-37's `[shift+U]` modal without saying it is unbuilt (F-108); `:190` "~1 card while the mock draws 10" — the cap is 15. It answers the second half well: the deck split (`:110-149`) and the fourteen UX rulings (`:179-207`) are exactly what a reader rebuilding from code would get wrong. Severity: Minor. Simplify: N. Action: mark D-37 unbuilt, fix D-51, point `:230` at the sections.

## Verdict
**Do not ship as documented.** The code-facing record (sweep, gates roster, accepted costs, required reading) is strong; the release-facing record (build report, handoff, exposure statement) contradicts itself on every headline number and its completeness claim does not survive sampling.

**Fix first:** the traceability derivation (§3) — it is the one claim the exit rests on, it is wrong by method not by typo, and the README privacy sentence (§2) should go to the HUM LEAD in the same sitting.

## Business Quality

# 0.16.0 Broadcaster UI — independent review at 557a40f

Scratch clone: `<scratch>`. Repository untouched. Cited tests run: `domains/locations`, `platform/lineup`, `modes/tty`, `app` subsets — all green. Two throwaway tests written in the clone (deleted after) are the evidence for findings 1 and 4; each had a control that passed.

## Q1 — The requirement written down, or the one remembered?

**The traceability table is a string-match over FR IDs, and IDs collide across releases.** "TRACED" rows cite tests for a different release's FR of the same number:
- FR-4.3 (cut-over must not lift the duck; RS-2 HIGH) → `app/inject_scenarios_test.go:5,78` — that file's "FR-4.3" is *"it expires within two minutes"*, the test-event injector.
- FR-2.2 → `app/ticker_test.go:1055` says *"0.15.0 FR-2.2"*; `classifier_crosstable_test.go:36` is the curated query.
- FR-9.2 → `app/cast_test.go:518`; FR-9.3 → `platform/lineup/bed_test.go:280` ("a tune that never lands"); FR-6.4 → `modes/tty/relayfault_test.go:716` ("the clock stops when the listener chooses").

**Severity: Important. Evidence:** `08-reports/build-report.md:298-366` vs the files above. **Simplify? Y** — derive by test *name* from `gates.md`, not by ID grep. **Action:** re-derive; every collision row becomes "TRACED (no ID)" or OPEN.

**"None is untraced" (`build-report.md:284`, commit message) reads as "all met"; the same table holds FR-3.6 OPEN, FR-8.3 OPEN, FR-8.8 OPEN, four PARTIAL, FR-10.1 SUPERSEDED with the requirement text never updated, and FR-2.1 still says ten cards where the console draws fifteen (`build-report.md:301`).** Minor, but it is the headline a defender will be quoted on.

The ten I checked against code and cited test: FR-1.4, FR-3.2, FR-3.3, FR-5.4, FR-2.2 (`cutover_test.go:68` is real) — **met**. FR-3.3 is met the right way: `broadcaster_manage_test.go:239-286` stubs the schedule and moves the display only on a `LineupMsg`. FR-8.6 — amended, see Q3. FR-8.3, FR-8.4, FR-4.3 — see Q2/Q5. FR-8.10 — citation legitimate (`composer_test.go:99`), not re-run.

## Q2 — What the operator loses

**Finding 1 (Critical). F-150's fault class re-enters the schedule instantly and never raises the window while the line-up is topped off.** `escalation()` fires only when `held()==0` (`platform/lineup/fault.go:41,45-62`); the Producer refills on every publish (D-54), so on a station with a 25-place pool the schedule is never "stopped". Worse, `onFailed` records the cool-off only when `ev.Routed` (`director.go:728-729`), so a *non-routed* fault — compose error, no composer, empty report — is the one class `retry.go` does not rate-limit, although its header claims it does (`retry.go:16-25`). Proven in the clone: ten consecutive non-routed faults on a 3-deep topped-off track → re-admitted 10/10 times, 0 Escalate; control with `Routed:true` → held out, track empties by iteration 3. Operator consequence: **ON AIR banner, line-up churning at pump speed, dead air, and nothing on the console** — `report` goes to `radioDebugLog` only (`app/schedule.go:133-135`). This is the HUM LEAD's own UAT defect of 2026-09-10 ("FLYING through locations") reintroduced for the class F-150 just created. **Evidence:** `app/executors.go:644-665`; `fault.go:41-62`; `director.go:728`; `retry.go:16-25`. **Simplify? Y** — note the cool-off on every `Failed`, routed or not. **Action:** drop the `ev.Routed` guard at `director.go:728`; add a consecutive-fault escalation (N faults with no Finished between) since `stopped()` cannot be reached on a live station; test with a topped-off track, not `failing(t,1)` (`fault_test.go:68` uses one rail card and an empty main track — a state Broadcaster never occupies).

**Finding 2 (Important). The line is not drawn where an operator would draw it.** "The voice could not render a line" is classed Routed at `executors.go:541-546` — the station *cannot perform* (no voice, Piper install failure → `mainread.go:157-160` returns false via `engine.Fail`), yet it is graded deliberate. And the rail's mute decline (`executors.go:455-456`) is the *listener's* `[M]` from Observer silently swallowing every hazard on a station the operator put ON AIR, with no console indication (`grep muted modes/tty/broadcaster*.go` finds only a comment). **Evidence** as cited. **Simplify? N. Action:** move "voice could not render" to `fault`; surface Observer's mute on the Broadcaster air row or refuse ON AIR while muted — HUM LEAD's ruling either way.

**Finding 3 (Minor).** When the window *does* fire it is `RelaySilentMsg` — "the relay is silent" with tune candidates (`app/radio.go:1291-1298`) — for a station whose fault was "no composer". Wrong words on the one safety surface.

## Q3 — What was cut, and is it findable?

- **FR-8.6 amendment is a correction, not a lowering:** "population-descending" was never built and contradicts the station-context ruling; nearest-first is what `pool.go:71-97` does. But the amended text names only tier two ("from the population-filtered table") and omits tier three, the zip places the header calls *"the point"* — and the exit *"the rule is stated where the operator reads it"* is unmet: no console text states nearest-first or the 25 cap (`grep` of `modes/tty` finds only comments). Minor; add the sentence to the amendment and the pool header row.
- **FR-6.5** withdrawn in its own text — clean.
- **FR-3.6, FR-8.3, FR-8.8 OPEN rows exist only inside the build report's table; `06_docs/follow-ups.md` has no row for any of them** (grep: none). The 0.16.5 list there is tooling only (F-133/137/139/156, `follow-ups.md:404-407`). The next reader looks in follow-ups and will not find three open requirements. Important; add rows.

## Q4 — Cheaper way to the same outcome

Yes, for Finding 1: no new window. The console already has the NFR-7 band (`broadcaster.go:275-312`, `heldBand`); route consecutive non-routed faults to that band ("3 CARDS FAILED TO COMPOSE — check provider keys") and reuse the cool-off. For FR-8.3: `poolRoom()` already knows the pool size (`broadcaster_pool.go:129-140`); one line "1 PLACE IN REACH AT 3 MI" is the honest console, no mechanism.

## Q5 — Three-mile or fifty-mile?

**Measured with the real index around Bonsall:** 3 mi → pool of **1** (home only); 5 mi → 1; 10 mi → 7 (2 cities, 5 zip places incl. San Luis Rey); 50 mi → 25, **every one a city-table entry** reaching Lake Elsinore and Laguna Niguel — tier three, "the big value", gets zero slots because tier two fills the cap first (`pool.go:77-87`). So the release serves a ~10-mile station well, a 50-mile station as a city list, and a 3-mile station as one card read every dwell. Whether one card on loop is "usable" (FR-8.3's exit) is the HUM LEAD's call; I present the fork with the number. `TestATightFenceStillKnowsWhereTheStationIs` (`pool_test.go:93-104`) already pins ≤3 at 2 mi and calls the rest "the console's to say honestly (F-83)" — the console does not say it.

**FR-8.4's cited test is not the Broadcaster's.** `TestAZoneOnlyAlertTheAppIsTrackingSurvivesTheRadius` (`app/severe_test.go:431-460`) calls `scopeEvents(..., 100, ...)` — Observer's alert scope at 100 miles — not `lineup.Fence` and not 3 miles. The mechanism (`ticker.go:500`, `severe.go:421-436`) keys on tracked locations within the radius, which at 3 miles is Bonsall alone, so the county warning arrives only if Bonsall's own snapshot carries it. Plausible; **unasserted**, and the report itself says so (`build-report.md:336`). Important; write the 3-mile fence test the requirement names.

**Finding 4 (Important). FR-4.3's own case is untested and the Director contradicts it.** `executors.go:161-164` still states the premise the FR says no longer holds ("nobody pressed anything"). At the Director level a pressed cut-back (`router.go:966` → `CutOver{ToBed:false}`) while a tornado is on the rail emits `Restore` (`bed.go:384-402`; reproduced in the clone: duck=0 restore=1 with the rail holding the hazard). Audio survives only because the arbiter's `releaseBed` re-settles and `mc.giveWay()` re-dips (`app/director.go:684-692, ~457`) — the half 0.14.0 ruled dissolves, RS-2's exact shape. **Action:** an app-level test of pressed cut-back under a reading hazard asserting the dip holds; decide at the seam.

Clean, against the questions that cover them: FR-3.3 and FR-3.7's confirm; FR-5.4/FR-1.4 door and refusal; NFR-7's ladder; FR-5.5's boundary at every width.

## Verdict

**Do not ship.** Fix first: Finding 1 — `director.go:728`'s `Routed` guard on the cool-off, plus an escalation that can actually fire on a live station. The release's own words: an action shown as taken that was not taken — ON AIR, over dead air, with no window and a spinning line-up.

## Remediation review — R2 (F-150 fault class), round one

## R2 remediation review — b51cdd6 + d9b7fb5 (fault run / cool-off / band / read seam)

Baseline at tip: `go test ./platform/lineup ./app ./modes/tty` green (0.8s / 110s / 5.6s). `make mutant-anchors` exit 2 — see F3. Every construction below was RUN in a fresh clone; verdicts are what I observed.

### F1 — The escalation is dropped in the one configuration where every card faults. **Critical**
The band is SET through the deck and CLEARED through the console: `app/radio.go:1289` `if d == nil || d.p == nil { return }` before `d.send(tty.StationFaultMsg{...})`, while the clear goes out via `x.publish` at `app/executors.go:557-558`. A nil deck is a supported build (`app/dashboard.go:494-496` returns it; `:305` starts the schedule with it and a live `p.Send`), the station control is still published (`app/schedule.go:235`), so ON AIR is reachable (`modes/tty/router.go:931`). Then `x.read == nil` faults every main-track card (`app/executors.go:542-543`, "no reader for the main track"), the run reaches 3, `Escalate` fires, `deck.escalate` returns silently. That is the reported defect verbatim — ON AIR over dead air, nothing on the console — and the reason string written for it never arrives. Evidence is structural (no bench builds a nil-deck executor with a console).
**Edit:** in `executors.run` `case lineup.Escalate:` publish `tty.StationFaultMsg{Run: v.Run, Reason: v.Reason}` through `x.publish` (the channel the clear already uses); keep `x.escalate` for the debug log only. One owner for one message.

### F2 — `NeedsRead` walks past the cool-off. **Important**
Only `admit` asks `sittingOut` (`platform/lineup/topoff.go:241`); `onNeedsRead` (`platform/lineup/rotation.go:52-70`) queues straight in. RUN: `offering(3,a..d)` → fault `a` → `Offered(a..d)` gives `[b c d]` (refused, correct) → `NeedsRead{Ref:"a"}` gives `[b c d a]` — re-admitted inside its five minutes. Producers of `NeedsRead`: `app/radio.go:252`, `:1073` (`go d.needsRead` on every relay `Failed` status), `:1268` (relay silent). A location whose report cannot compose AND whose relay is failing re-enters on every relay failure — the UAT 2026-09-10 loop through the second door.
**Edit:** `if d.sittingOut(ev.Ref) { return d, nil }` before `Queue` in `onNeedsRead`, with a test on that path.

### F3 — mutant mCC drifted; the halted-read rule is unmeasured. **Important**
`06_docs/mutants/mCC_a_halted_read_comes_home_finished.py` anchors on `r.finish(false)`, gone since the seam returns `error`. `make mutant-anchors` → `DRIFTED mCC … 1 of 379`, exit 2. Re-pointed plant (`app/mainread.go:146` `r.finish(errReadStopped)` → `r.finish(nil)`): **CAUGHT** by `TestAReadHaltedAfterItStartedComesHomeFailed` and `TestAReadSessionEndsOnce…`, so re-pointing restores the measurement.
**Edit:** re-point old/new to the `player.Stopped` case; add a sibling for `:144` (`Failed` → `errReadStopped`, "a dead voice comes home routed") since P7 shows that grade held only at the executor, not at the seam.

### F4 — Two of the claimed properties are not held by the tests. **Important**
- `platform/lineup/fault_test.go:190-191` says "a Finished between faults resets the run"; nothing exercises it. Plant P1 (delete `platform/lineup/director.go:718` `d.faultRun = 0`): **SURVIVED**. Consequence unmeasured: the run never resets and every fault after the third escalates for the life of the process.
- `modes/tty/broadcaster_fault_test.go:26-29` "STANDBY did not clear" passes because `faultNotice` hides on `power != Running` (`modes/tty/broadcaster.go:353`). Plant P5 (delete `:504` `b.fault = StationFaultMsg{}`): **SURVIVED**; under P5 my construction Running→fault 3→OffAir→Running showed the stale band **reappear** (tip: false; P5: true).
- Also SURVIVED: P9 `Run: 0` in `Escalate` (`Describe` at `director.go:307-308` omits Run, so no lineup test can see the count) and P8 delete the clear at `executors.go:557-558` (unobserved at the executor).
**Edit:** assert `faultRun == 0` after `Finished` between faults; assert no "FAILED" after OffAir→Running; a bench assertion on the published clear.

### F5 — The Director keeps counting across standby; the console does not. **Important**
RUN: 2 faults → `Powered{OffAir}` → `Powered{Running}` → 1 fault → `escalate(read:c)` with Run 3. The console cleared on standby as "the operator acting on it" (`broadcaster.go:504`); the Director (`power.go:148-162`) never touches `faultRun`. The operator sees "3 CARD(S) FAILED" after one.
**Edit:** zero `faultRun` in `onPowered` on leaving `Running`, with a test.

### F6 — Two authors of one count. **Minor**
The band clears only on a MAIN-TRACK finish (`executors.go:557`), the Director resets on ANY `Finished` including the rail's (RUN: rail finish → `faultRun=0`, band untouched). The counter also moves before `leave` reports the card was held (RUN: `Failed{ID:"ghost"}`×3 → `escalate(ghost)`; `Finished{ID:"ghost"}` → reset). Both defensible — a real read did finish, a real fault did occur — but suggest carrying the run on `Publish` so the console has one source, which also removes F1's split and the executor's clear.

### F7 — Band-set vs band-clear can invert. **Minor**
`Escalate` has no `Holds` case (`director.go:365ff`) so it runs on a free goroutine (`app/pump.go:158-160`), concurrent with the broadcast worker's clear. Sequence: A reading; B,C,D fault → `Escalate` dispatched; A finishes → clear sent, `Finished(A)` queued; the escalate goroutine then draws "3 CARD(S) FAILED" over a run the Director has just zeroed, until the next main-track finish. Not deterministic to run; structural. Folds into F6.

### Items answered without a defect
- **Interleaved declines** (RUN f,m,f,m,f): routed declines neither reset nor count; escalates on the third fault. Right — a decline delivered nothing either; a `Finished` is the only proof the station performed.
- **Rail** (RUN 2 main + 1 rail fault → `escalate(burst:h00)`): a rail fault counts, and should — the station could not perform a hazard. `Arrived` leaves the counter alone.
- **Read seam**, every return: nil/empty → "nothing to say" fault; `readCast` err → fault (engine.Fail first; session not armed so no double report); `StartSource` err → fault; `ended` → nil; `player.Failed` → fault; `player.Stopped` → routed; `rctx.Done` → Halt → routed. Engine: EOF → `Stopped`+`EndedTitle` (`engine.go:566`), stream error → `Failed` (`:568`), `halt()` → `Stopped` (`:219`) whose callers are displacement (guarded by `begun`, F-95) and the read's own halt. A relay dying is `Failed` on the relay path (`:166`, `:281`), never this session. `Stopped` is deliberate at the schedule level. Residual: `mainread.go:213-227` selects at random when `done` and `rctx` are both ready, so a `Failed` landing as standby is pressed can grade routed — acceptable.
- **Words/count**: "3 CARD(S) FAILED — the station could not perform them: the report could not be composed: no key"; RUN 3,4 → Finished → new run 3. Right.
- **Plants CAUGHT**: P2 limit→300, P3 no increment, P4 routed-only cool-off (lineup test); P6 message ignored, P10 never hides (tty test); P7 every read error routed (app test).

**NOT LGTM.** Critical 1 / Important 4 / Minor 2. Fix first: F1 — publish the `Escalate` through `x.publish`, the channel the clear already uses, so the band has one owner and a nil deck cannot swallow the one message this remediation exists to deliver.

## Remediation review — R2 round two (three passes to LGTM)

### Pass 1 of 3

## REVIEW R2b — `3c67d88..4748f5d` (ad30d79, 4748f5d)

Clone at `<scratch>/review-r2b/wp`, checked out `4748f5d`. Baseline `go test ./platform/lineup ./app ./modes/tty`: all ok.

### Does it close F1–F5?

| Prior finding | Closed? | Evidence |
|---|---|---|
| F1 band set via deck, cleared via publish; nil deck swallowed the set | **Yes** | `app/executors.go:356-358` publishes `StationFaultMsg{Run,Reason}`; `:558-560` publishes the clear; `radioDeck.escalate` and the `escalate` seam deleted (`app/radio.go`, `app/schedule.go:173`). One owner. |
| F2 `onNeedsRead` bypassed `sittingOut` | **Yes** | `platform/lineup/rotation.go:68-70`. Plant P1 CAUGHT. |
| F3 mCC drifted | **Yes** | `06_docs/mutants/mCC…py` matches `app/mainread.go:145-146`; P8 CAUGHT. New mCL4 CAUGHT (P7). |
| F4 four unheld claims | **Yes** | Finished reset (`fault_test.go:252`), standby clears-not-covers (`broadcaster_fault_test.go:32-38`), published clear (`executors_test.go:1461-1466`), Run in Describe (`director.go:308`). All four plants CAUGHT. |
| F5 Director counted across standby | **Partly** — see S4 | `power.go:158-160` zeroes only on leaving `Running`. |

### (1) Plants — all CAUGHT

| Plant | Result / FAIL line |
|---|---|
| P1 delete NeedsRead guard `rotation.go:68-70` | CAUGHT `fault_test.go:247: a place in cool-off was re-admitted through NeedsRead: read:pa` |
| P2 delete standby zeroing `power.go:158-160` | CAUGHT `fault_test.go:297: one fault after standby escalated…` |
| P3 delete fault publish `executors.go:356-358` | CAUGHT `executors_test.go:1079`, `:1482`, `fault_grade_test.go:61` |
| P4 delete Finished reset `director.go:718` | CAUGHT `fault_test.go:266: the first fault after a finished read escalated…` |
| P5 delete finished-read clear `executors.go:558-560` | CAUGHT `executors_test.go:1465: a finished read published []` |
| P6 delete tty standby clear `broadcaster.go:520` | CAUGHT `broadcaster_fault_test.go:37` |
| P7 mCL4 | CAUGHT `mainread_live_test.go:246` |
| P8 mCC | CAUGHT `mainread_live_test.go:224,227` |

### (2) Adversarial sequences (in-package, observed)

- **S1 — `Finished` for a card NOT held resets the run.** Two faults, `Finished{ID:"read:ghost"}`, third fault: `faultRun=0`, no escalation. `director.go:718` zeroes before `leave` decides `left` (`:722-727`), whose own comment says an unheld completion "must not disturb" anything. A stale Finished from a superseded read wipes the count exactly in the churn where faults happen. **Important.** Not tested.
- **S2 — cool-off expiry.** `Tick` at `retryAfter-1s`: still out; at exactly `retryAfter`: re-admitted (`retry.go:114` uses `<`). Clean.
- **S3 — fault/decline/fault/decline/fault** → escalates `run=3`. Consistent with `director.go:469`, but "in a row" is a fork: HUM LEAD to rule whether a decline breaks a run.
- **S4 — faults while OFF AIR.** Three non-routed `Failed` after `OffAir`: `escalate(read:ba run=3)` emitted off air; `faultRun` stays 3 through `→Running`; next fault escalates at run=4. On the console the standby clear (`broadcaster.go:520`) fires on the transition, the later `StationFaultMsg{Run:3}` lands after it, hidden until `Running` — then shows. `advances` (`power.go:112`) holds the rail off air too, so realistically one in-flight compose fault, not three, lands this way; still, F5's zeroing is one-sided. **Minor**, fork: zero on any transition, or gate `faultRun++` on `power==Running`.
- **S5 — rail + main mixed** count one run (1 main, 1 rail, 1 main → `run=3`). By design; note only.
- **T1 — `Run:0` escalation shows nothing.** `StationFaultMsg{Run:0, Reason:"…asked to move…"}` while `Running`: band absent (`broadcaster.go:358`). `bed.go:158` constructs exactly this (`Escalate{Reason}`, no Run, no ID). The tune-failure escalation is delivered by the executor and dropped by the console. **Important**; predates the change, not excused.

### (3) Nil-deck path, end to end

Yes. `app/dashboard.go:305` passes `p.Send` as `publish` unconditionally (no deck gate between `:261-305`); `app/schedule.go:116` hands it to `newExecutors`; `executors.go:356-358` publishes; `broadcaster.go:511` stores; `:357-362` renders while `Running`. With a nil deck `readerFor` is nil, so every Speak faults at `executors.go:545` and the band reads "3 CARD(S) FAILED … no reader for the main track" — delivered.

### Stale references / duplication / helpers

- No `deck.escalate`/`escalate:` in code, mutants, or `declset.txt`; only `06_docs/…/red-team-build-round3.md:154` (a record; leave).
- `fault_test.go:165-178,193-197` re-implement `topOff` and `escalated` inline — the helpers were extracted at the second caller and the first was left. Fold.
- `station_test.go:75-79` is a second `StationFaultMsg` filter beside `bench.faults` (`executors_test.go:1082-1096`), with a different rule (drops the zero message). Two harnesses, two definitions of "an escalation".
- `topOff`, `faultOnce`, `escalated`, `bench.faults` earn their place; `offlineClient` predates this range (3c67d88), two callers — out of scope.

### Code Quality axis

1. **Understandability.** `app/schedule.go:169-172` — a four-line comment ("DR-21's one escalation channel. It reuses the relay-fault window…") now sits above nothing and describes a design that no longer exists. Important. Evidence `schedule.go:169-172`. Delete Y. Action: delete.
2. **Maintainability.** S1 above (`director.go:718`). Action: `if _, ok := d.find(ev.ID); ok { d.faultRun = 0 }` before `leave`, plus a test.
3. **Complexity.** Nothing found; the change removes a seam and an invariant.
4. **Necessity.** Delete: `fault_test.go:141 var _ = time.Second` (dead import keeper); the inline loops at `fault_test.go:165-178,193-197` (use `topOff`/`escalated`); the `station_test.go:75-79` filter (route through one helper). Minor each.
5. **Name semanticism.** `executors_test.go:1476 newBench(t, nil) // no audio at all: the build whose deck is nil` — the nil is a `*scriptVoice`; the executors never hold a deck. The comment names a proof the test does not perform. Minor. Action: reword ("the executors do not ask the deck").
6. **Code documentation.** History narration: `executors.go:349-354` ("The band was set through the deck and cleared here… R2 review F1"); `rotation.go:64-68` ("this door alone re-admitted… R2 review F2"); `mainread_live_test.go:232-235` ("held only at the executor until mCL4 SURVIVED"). Stale: `director.go:469` says a run ends only on Finished (standby now ends it too); `fault.go:27-29` says Run is "0 when the schedule simply stopped" — a stopped-by-fault schedule carries `Run≥1`; only `bed.go:158` is 0. Minor each. Action: state current rule; move the "why" to the record.
7. **Testability.** Adequate; S1 and S4 are untested interleavings worth one test each.
8. **P10.** Loops bounded and annotated (`fault_test.go:205,225,246,277`); no recursion; scope minimal. Clean.
9. **Evidence soundness.** (a) `executors_test.go:1067-1080` asserts an `Escalate{ID:"c1"}` (Run 0) "reaches a person" — the console drops Run 0 (T1); the test proves the seam, not the person. Important. (b) `fault_test.go:188-189` comment claims "a Finished between faults resets the run" for a control block that only tests a routed decline. Minor. Action: remove the claim (now held at `:252`).

### Verdict

**LGTM with conditions** — F1–F4 are closed with plants that fail for the stated reason; F5 is closed for the interleaving the tests state. Fix first: **S1 / `director.go:718`** — a ghost `Finished` resets the fault run, which silently defeats the count the whole remediation exists to deliver; one-line fix, one test. T1 (Run 0 dropped at `broadcaster.go:358`) should follow as F-16x with a HUM LEAD ruling on whether the tune-failure escalation belongs in this band.

### Pass 2 of 3

## REVIEW R2b — verification of `4748f5d..b0c1d70`

Fresh clone `<scratch>/review-r2b/wp2` at `b0c1d70`, read-only. Baseline `go test ./platform/lineup ./app ./modes/tty`: all ok.

### Conditions, verified

| Condition | Held? | Evidence |
|---|---|---|
| S1 ghost `Finished` no longer resets the run | **Yes** | `platform/lineup/director.go:722-724` gates on `d.find(ev.ID)`. Plant P1 (`if true`) CAUGHT: `fault_test.go:311: a Finished for a card the schedule never held reset the run…`. Plant P3 (`if false`) CAUGHT by the older test `fault_test.go:259`, so the two tests hold opposite edges. |
| S4 faults OFF AIR do not count | **Yes** | `director.go:748-749` `!ev.Routed && d.power == Running`. Plant P2 (gate removed) CAUGHT `fault_test.go:327: a fault while OFF AIR escalated…`. Plant P4 (gate inverted) CAUGHT by three tests (`:171,175,267` and `TestStandbyEndsTheFaultRun`). |
| Q1 orphaned `schedule.go` comment | **Deleted** | `app/schedule.go:166-169` now ends at `propose:`. |
| Q4 inline loops folded, `time` keeper gone, one filter | **Yes** | `fault_test.go:160-168` uses `faultOnce`/`escalated`; import list `:3-8` has no `time`; `executors_test.go:1092-1103 faultsIn` is the one definition, `station_test.go:190-198 escalated()` derives from it. Plant P5 (filter inverted) CAUGHT `fault_grade_test.go:61`. |
| Q5 bench comment | **Reworded** | `executors_test.go:1482`. |
| Q6 history narration | **Current** | `executors.go:349-352`, `rotation.go:64-66`, `mainread_live_test.go:231-233`, `director.go:468-471`, `fault.go:26-28` all state the rule, not the draft. |
| Q9(a)/(b) | **Yes** | `executors_test.go:1077` raises `Run: 1`; `fault_test.go:177` control claim narrowed. |

### What the fix moved, observed

- **S7 (new).** Single held card, `OffAir`, non-routed `Failed`: `escalate(read:x run=0)`. The S4 gate leaves `faultRun` at 0, `stopped()` is true, and the escalation now carries `Run 0` — which `broadcaster.go:358` drops. Before this fix the same event carried `Run 1` and showed on the way back ON AIR. This is a real narrowing: a compose fault that empties the schedule while on standby is now told to nobody. Realistic? One in-flight compose across the transition, with a one-card schedule — rare, but the producer refills only on the next publish. **Minor**, and it belongs in the same ruling as T1. Evidence `director.go:748-749`, `fault.go:64-73`, `broadcaster.go:358`. Delete? N. Action: fold into the T1 follow-up; the cheapest fix is the console showing `Run 0` escalations with the reason alone (the band already has a reason-only shape via `noticeBand`).
- **S8/S9.** A `Finished` for a held-but-queued card and a rail `Finished` both reset the run (observed `run=0`). Both are inside F-159's recorded scope; not new.

### Findings

1. **F-163 is not in the tree.** The coordinator's message says T1 and S3 are recorded as F-163 with rulings owed; `grep -rn F-163 06_docs` is empty at `b0c1d70`, and no added line in the range mentions the Run 0 drop or the decline question (`06_docs/follow-ups.md` gains F-118's close and F-159–F-162 only). Two Important findings from R2b (T1 at `broadcaster.go:358`, S3 fork) have no record. **Important** — the standing rule is that a deferred defect is recorded before REVIEW exits, and the evidence for "recorded" is the row. Evidence `06_docs/follow-ups.md:386-390`. Action: add the row; include S7 above.
2. `app/fault_grade_test.go:59` still raises `Escalate{ID:"burst:a", Reason:"the schedule stopped"}` with `Run 0` and asserts at `:60-61` that "DR-21's one channel delivered" it — the same shape Q9(a) fixed in `executors_test.go:1077`, and the console drops it. **Minor.** Delete? N. Action: `Run: 1`, or assert against what the console shows.
3. `station_test.go:190-198 escalated()` re-derives "an escalation" as `f != zero` while `faultsIn`'s doc (`executors_test.go:1093-1095`) calls itself "the ONE definition". Two harnesses still hold two rules (one counts the clear, one does not). **Minor.** Delete? Y. Action: have `faultsIn` take the filter, or give it a sibling `escalationsIn` used by both.

### Code Quality, in order

1 Understandability — clean. 2 Maintainability — finding 3. 3 Complexity — clean; two one-line gates. 4 Necessity — `station.escalated()` is the only removable element (finding 3). 5 Names — clean; `faultOnce`'s doc now says which place fails (`fault_test.go:202-203`). 6 Documentation — clean; every comment I flagged states current rule. 7 Testability — clean; S7 is the untested edge. 8 P10 — loops annotated (`fault_test.go:208,216,323`), no recursion. 9 Evidence soundness — finding 2; and the coordinator's claim "recorded as F-163" is unsupported by the tree (finding 1).

### Verdict

**NOT LGTM**, on one condition only: the F-163 row does not exist at `b0c1d70`, so two Important findings are neither fixed nor recorded. The code changes for S1 and S4 are correct, plant-verified, and their comments are current. Fix first: write the F-163 row (T1 Run-0 drop, S3 decline fork, and S7 above), then this is LGTM without a further code change.

### Pass 3 of 3

## REVIEW R2b — final verdict at `9e318ac` (range `b0c1d70..9e318ac`, with `620bce5`)

Fresh clone `<scratch>/review-r2b/wp3`, read-only. Baseline `go test ./platform/lineup ./app ./modes/tty`: all ok.

**Condition — F-163 recorded.** `06_docs/follow-ups.md:386` now holds one row with (i) T1 (`Run: 0` from `bed.go` dropped at `faultNotice`), (ii) S3 (does a decline break a run; today fault/decline/fault/decline/fault escalates at run=3), (iii) S7 (a one-card schedule emptied by a compose fault on standby escalates with `Run: 0`), status **OPEN — HUM CALL**, and the reason-only band named as the cheapest close for (i) and (iii). It says what the code does, not more. Held.

**Finding 2.** `app/fault_grade_test.go:59` raises `Run: 1`; the test now asserts an escalation the console would show. Held.

**Finding 3.** `app/executors_test.go:1106-1114 escalationsIn` is the one filter; `station_test.go:188-194 escalated()` derives from it and drops its own `tty` import; `bench.faults()` keeps the clears for `executors_test.go:1465` which asserts one. Plant (filter inverted) CAUGHT: `fault_grade_test.go:61: DR-21's one channel delivered 0 escalations, want 1`. Held.

**In-range change not on my list.** `f7fe5fa` moves the console's power transition into `Broadcaster.powered` (`modes/tty/broadcaster.go:518-529`) for P10-04. Behaviour is unchanged line for line; the standby clear now sits at `:526`. Re-planted (clear deleted) — CAUGHT: `broadcaster_fault_test.go:37: the fault band came back ON AIR after standby…`. Clean.

Nothing else in the range touches production code. No stale references, no duplicated assertions, no history-narrating comments introduced.

**LGTM.**

## Remediation review — rulings 10/12/F-163 and FR-9.4 at the writer (two passes to LGTM)

### Pass 1 of 2

## REVIEW R10 — commit 9944b5c (0963b36..9944b5c), fresh clone, read-only

Baseline at 9944b5c: `modes/tty` ok, `platform/lineup` ok, **`app` FAIL** (two tests; both green at 0963b36).

### (1) Privacy — the coordinates still reach the log, on every Director step
**`app/pump.go:314-319`** with `platform/lineup/director.go:306,1140`, `platform/lineup/rotation.go:42`, `platform/lineup/director.go:1157-1180`, `app/executors.go:353`. Severity **Critical**.
Evidence (scratch test in the clone, tower `33.2887,-117.2253` as a pool member, log read from disk under `WATCHPOST_DEBUG_RADIO`):
```
LEAK director:ev:tuned(33.2887,-117.2253)
LEAK director:fx:tune(33.2887,-117.2253)
LEAK director:fx:build(read:33.2887,-117.2253)
LEAK director:fx:escalate(read:33.2887,-117.2253 run=1)
LEAK director:cards:MAIN TRACK=[read:33.2887,-117.2253:ADMITTED]
LEAK schedule:escalate:the station was asked to move to 33.2887,-117.2253 and did not
ok   needs-read stage=dark fresh=true place="Bonsall, CA" why=why
```
A card's ID is `"read:"+snapshot.Key(ref)`, `Tune.Ref`/`Tuned.Ref` are the key (`app/radio.go:1106`), and `trace()` writes `Describe`/`DescribeEvent`/`Lineup.Trace()` for every effect. The commit redacted one line of seven; the README sentence is false at this commit. Also on-screen: the bed reason (`platform/lineup/bed.go:158`) puts the pair in the new STATION FAULT band. Simplify/Delete? N. Action: the key is identity and must stay in the Director; redact at the one describer boundary (a label lookup by key when the trace is written, or `speak(<label>)`), and the bed reason names the relay's label. Not excused by predating the change.

**Test is not a gate.** `app/radio_debuglog_test.go:143-158` — Important. `TestTheRadioDiagnosticNamesPlacesNotCoordinates` calls the pure `needsReadLine`; it never reads a log. Plant A1 (call site at `app/radio.go:641` reverted to `snapshot.Key(ref)`, function untouched): **SURVIVED** — `ok app 0.494s` while the file carried the pair. Not enough for the README sentence. Action: assert over the file after `d.needsRead(...)` (as `maintrack_deck_test.go:117` already does) and after one `trace()` step with the tower on the main track — one sweep over the whole log for the tower's digits.

**Red suite shipped.** `app/maintrack_deck_test.go:127` and `app/declset_test.go:34` — **Critical**. `TestTheDarkRunRecordsTheNeedItWouldHaveActedOn` still wants `ref=33.1959,-117.3795` (fails ×3 stages); `TestDeclarationSetUnchanged` reports `added: [func needsReadLine]`. Both green at the parent. `go test ./app/` was not run before the handoff wrote "done". Action: retarget the dark-run instrument's assertion to `place=`, refresh the declaration baseline, and record the run.

Other cache-dir files: HTTP cache is `sha256(url).cache` (`platform/httpx/cache.go:610`), profiles dump carries counters only, ticker `seen` — no coordinates found. Only the radio log leaks.

### (2) Label-only loses identity
`app/radio.go:1137-1140` — Important. Geodata has 138 duplicate `(name, admin1)` labels; two in CA (`Brentwood, CA` ×2, `Vincent, CA` ×2). The ZIP is used only when the label is empty; `Tag` never. Two pool members with one label produce identical lines and the dark-vs-live comparison the line exists for (`maintrack_deck_test.go:103-109`) cannot tell them apart. Action: `place=%q tag=%q` (the operator's 5-char tag, which is not a position) — HUM CALL on granularity.

### (3) Console sequences (driven through `Broadcaster.Update`, band read from `View`)
- Running, `{0,"the relay could not be tuned"}` → `!!! STATION FAULT — the relay could not be tuned`; then `{}` → no band.
- OffAir, `{0,"emptied on standby"}` → no band; then Running → `!!! STATION FAULT — emptied on standby` (the standby escalation surfaces on the next ON AIR — correct for S7; note it).
- `{3,"voice down"}` after Run 0 → `!!! 3 CARD(S) FAILED — the station could not perform them: voice down`.
- `{0, bed reason}` after Run 3 → STATION FAULT with the coordinate pair on screen (see 1).
- STANDBY then ON AIR → cleared.
`{Run:0, Reason:""}` as a fault: not today — `platform/lineup/fault.go:70-73` defaults an empty reason and `bed.go:158` is literal. It is a convention, not a type; acceptable, record it.

**Comment contradicts the rule.** `modes/tty/broadcaster.go:46-49` — Important. The type doc says "Run 0 clears it"; `faultNotice` now shows Run 0 with a reason. Delete? N. Action: "the zero message clears it; a Run of 0 with a reason is a fault with no count."

### (4) README table vs keymap (`README.md:190-201`, `modes/tty/router.go:162-222`)
- Missing from the table: **`A`** (`router.go:1031`, the takeover box's handle, an address like the digits — the HUM LEAD asked for it at UAT 2026-09-13); `ctrl+b`/`B` (bound, a no-op on the console — omission fine, but the table claims to be the console's keys).
- In the table, not in the keymap: `0`-`9` and `A`-less — by design (`router.go:630`). Correct.
- Wording: `README.md:192` "(held while a window is open)" — `router.go:929-935` *refuses* the toggle behind a window; "held" reads as key-held. Minor. Say "ignored while a window is open". `0`-`9` "the first ten" — true (`MainTrackCap = 16`, `operator.go:243`; digits address ten). Gain "mirrored with the Observer's volume" — true (`router.go:804-806` forwards to Observer).

### (5) Plants
- A1 call-site revert: **SURVIVED** (above).
- A2 function body revert: **CAUGHT** — `radio_debuglog_test.go:150: the line does not name the place: "needs-read stage=dark fresh=true ref=33.2887,-117.2253 …"` (+ :154 ×2, :158).
- B delete the Run-0 branch: **CAUGHT** — `broadcaster_fault_test.go:97: an ON AIR station with a fault and no run shows no "STATION FAULT": !!! 0 CARD(S) FAILED — …` and `:101 a fault with no run was shown as a count of cards`.

### (6) Necessity / comments — Minor
`app/radio.go:1132-1136` and `modes/tty/broadcaster.go:362-364` narrate the defect ("wrote the operator's antenna position…", "a band keyed on the count dropped it"); the test comments at `radio_debuglog_test.go:141-145` and `broadcaster_fault_test.go:85-89` repeat it. State the rule; the history is in the record. Nothing else removable.

## Verdict: NOT LGTM
Fix first: the Director trace (`app/pump.go:314-319`) — the tower's pair is written on every step the diagnostic records, and the gate that would have shown it reads a string, not the file. The two red `app` tests ride along with that fix.

### Pass 2 of 2

## REVIEW R11 — commit 000d61a (9944b5c..000d61a), fresh clone, read-only

**Baseline at 000d61a:** `app` ok (111s), `modes/tty` ok, `platform/snapshot` ok, `platform/lineup` ok. The two tests red at 9944b5c are green.

**Leak construction re-run** (tower `33.2887,-117.2253` as a pool member; a `trace()` step with Tuned, Tune, BuildCard, Escalate, Publish; the bed reason; `needsReadLine`; log read from disk). What the file carries:
```
director:ev:tuned(place:b2245476)
director:fx:tune(place:b2245476)
director:fx:build(read:place:b2245476)
director:fx:escalate(read:place:b2245476 run=1)
director:fx:publish(rail=[] main=[read:place:b2245476])
director:cards:MAIN TRACK=[read:place:b2245476:ADMITTED]
schedule:escalate:the station was asked to move to place:b2245476 and did not
needs-read stage=dark fresh=true place="Bonsall, CA" id=place:b2245476 why=why
```
No pair, and one place is followable across every line by one id — `app/radio.go:1241` rewrites at the single writer, so the seven lines that leaked at 9944b5c are covered by construction. `needs-read` carries `place=` and `id=` (`app/radio.go:1141`), which answers Q2 without a HUM CALL. The band rewrites a known key to the pool's label and an unknown one to `place:…` (`modes/tty/broadcaster.go:366-389`). The type doc (`broadcaster.go:46-50`) now states the rule; the README row for `A` and "ignored while a window is open" (`README.md:192-194`) match `router.go:929-935` and `:1031`.

**Plants, run myself:**
- P1 delete `line = snapshot.ReplaceKeys(...)` at `app/radio.go:1241`: **CAUGHT** — `radio_debuglog_test.go:187: the diagnostic names where the operator is (FR-9.4)`.
- P2 `reason := b.fault.Reason` at `broadcaster.go:366`: **CAUGHT** — `broadcaster_fault_test.go:121` and `:125`.

**One finding, Minor.** `platform/snapshot/redact.go:11` — the fence is `Key`'s shape (exactly four decimals, comma-joined). Two probe lines I wrote through the writer escaped it: `%.6f,%.6f` → `33.288700,-117.225300`, and `%v` of a `LocationRef` → `{Bonsall, CA  92003 33.2887 -117.2253  0}`. I swept every real writer: none composes either shape today; the only free text reaching the trace is `Failed.Reason` (`platform/lineup/director.go:1127`), which `app/pump.go:289` fills from a recovered panic value with `%v`. Evidence: probes above; sweep of `radioDebugLog(`/`debugLog(`/`x.fault(` arguments. Simplify/Delete? N. Action: state the bound in `ReplaceKeys`'s doc ("a pair in Key's form; free text from an error or a panic value is outside it"), or widen `keyPattern` to any two signed decimals joined by comma or whitespace — HUM CALL; not a blocker, because no writer produces those shapes and a panic value that carries a ref is a defect with its own gate.

Nothing removable; no comment narrates history.

## LGTM
Fix first (post-review, at the HUM LEAD's call): the `keyPattern` bound at `platform/snapshot/redact.go:11`.

