# Lens DC: documentation, artifacts and code comments (goal 4)

Tree: `watchpost` at `4c803d99` (v0.19.3). Read-only survey. The only files written are this report and
scratch files under `<lens scratch>/dc/`.

**Method.**
- Ran the existing checks first:
  - `go run ./tools/authoring`: "OK — 1002 Go file(s), no findings".
  - `go test ./tools/atlas`: ok (the atlas matches a fresh render). `tools/atlas` has no `-check` flag; `TestTheAtlasMatchesAFreshRender` is the check.
  - Everything below is beyond what those two catch.
- Wrote three throwaway AST scanners in the scratch dir:
  - `dc/symcheck`: names in comments that no declaration in the tree carries.
  - `dc/docbind`: doc blocks that carry a second declaration's doc, or name a missing one.
  - `dc/comments`: every comment line, then phrase scans for narration.
- Grepped every `make X` reference against the Makefile's targets.
- Three forks of this reviewer checked, against the code:
  - README / docs / Makefile;
  - the as-built maps and the top-level 06_docs;
  - follow-ups.md and the CHANGELOG.
- I merged and de-duplicated their findings here. Their raw reports are in `<lens scratch>/dc/fork-*.md`. Every finding names the file:line it was checked at.

Ranked most valuable first.

---

### DC-1: About 30 doc comments are bound to the wrong declaration; the SN-02 tool sees functions only, with a fixed verb list
- Where (production): each is a doc block whose first part belongs to another, or deleted, declaration.
  - `platform/snapshot/types.go:207`: "FireState holds hotspots and incidents (B5)." sits on `const MaxHotspots`. Exported `FireState` has no doc.
  - `domains/weather/nws/provider.go:45`: "Provider implements snapshot.Provider for api.weather.gov." sits on `const maxGrids`. `Provider` has no doc.
  - `domains/radio/synth/install.go:162`: the `EnsurePiper` contract ("installs Piper and the voice under dir when missing, verifying every artifact against the manifest; progress may be nil") sits on `var installMu`. `func EnsurePiper` at :167 has no doc.
  - `platform/render/units.go:124`: the `Glyphs` doc sits on `const HeadlineJoin`. `func (o Opts) Glyphs()` (:139) has no doc.
  - `modes/tty/dashboard.go:387`: the `Stats` doc sits on `type ZoneShapeStats`. `type Stats` (:398) has no doc.
  - `modes/tty/dashboard.go:584`: "RadioStatusMsg reports the player's condition (B4)." sits on `type VoiceNoteMsg`. `RadioStatusMsg` (:599) has no doc.
  - `app/mainread.go:149`: the 6-line `readCard` doc ("IT BLOCKS, DELIBERATELY") sits on `var errReadStopped`. `func readCard` (:160) has no doc.
  - `app/schedule.go:54`: the 21-line `startSchedule` doc ("THE LISTS ARE NOT ONE LIST", "`cutTo` DOES NOT RESOLVE AGAINST THE POOL") sits on `type bedSeams`. `func startSchedule` (:92) has no doc.
  - Same shape:
    - `domains/globalfeed/usgs.go:45`: the `usgsFeed` doc sits on `usgsFeature`.
    - `domains/severe/record.go:110`: the `capExtras` doc sits on `onsetOf`.
    - `modes/tty/memo.go:60`: the `bodyMemo` doc sits on `memoStats`.
    - `platform/render/table.go:399`: the `rowMarks` doc sits on `type Marks`.
  - Doubled docs, where the older half now contradicts the newer:
    - `app/radio.go:991-992`: "restore lifts the duck", then "restore lifts BOTH — a hold and a duck".
    - `app/radio.go:961`: a stale render/play paragraph above the current `render` doc.
    - Also `domains/globalfeed/stack.go:51` (`Lane`), `modes/tty/ticker.go:31` (`TickerCategory`), `domains/radio/synth/compose.go:246` (`Tail`), `modes/tty/broadcaster.go:455` (`heldEscalation`), `modes/tty/memo.go:159` (`rowLoading`) and `modes/tty/severe.go:230` (`severeWindowName`).
  - A deleted declaration's doc fused onto the next declaration:
    - `domains/marine/ndbc/ndbc.go:178`: "keepOr copies fallback ..." on `loadStations`.
    - `platform/snapshot/harmonize.go:109`: "keepOr returns have ..." on `KeepOr`.
    - `modes/tty/broadcaster.go:1348`: "cardRow is one lane row" on `type cardLane`.
    - `app/ticker.go:155`: a 6-line doc of a `tickerAudio` interface that is gone, on `startTicker`.
    - `modes/tty/setup.go:43`: `setupFocus` on `type setupState`.
  - Tests:
    - `cmd/watchpost/gates_test.go:49`: `verifyOnly` on `phaseExit`.
    - `app/mapairuv_test.go:82` and `domains/temperature/temperature_test.go:451`: the first line of the deleted tests `TestTheAirIsTheModelsGridsAskedWhileOn` / `TestOpenMeteoReadsTheAirQuality`, cut off mid-sentence and fused onto the next declaration.
    - `modes/tty/tick_test.go:18`: `tickOf` on `armedAfter`.
    - `modes/tty/bench_test.go:152`: `BenchmarkFrame_Help` on `BenchmarkFrame_133x44_Help`.
    - `platform/sched/sched_test.go:17`: `fakeClock`, documented twice.
- Observed:
  - `go doc ./platform/snapshot FireState` and `go doc ./domains/radio/synth EnsurePiper` print no description.
  - `tools/authoring` is green because `checkDocAttached` (`tools/authoring/rules.go:179`) inspects only `*ast.FuncDecl`, never type, const or var specs.
  - It also skips a block that opens with the declaration's own name, so doubled docs pass.
  - Its `godocOpener` verb list (`rules.go:218`) has no "renders", "fetches", "keeps", "lifts" or "installs". So `loadStations fetches …` and `cardLane renders …` are never recognised as openers.
  - The 0.16.0 red team already reported this class (`06_docs/02_features/0.16.0-broadcaster-ui/08-reports/red-team-review.md:147`). Its one named example is fixed; the class is not.
- Cost: hover, `go doc` and pkg.go.dev show no contract on `Provider`, `FireState`, `EnsurePiper`, `startSchedule` and `readCard`, and a wrong one on `MaxHotspots`, `installMu`, `bedSeams` and `errReadStopped`. A junior reads `restore`'s doc and cannot tell which half is true.
- Proposed change:
  - (1) A comment-only pass: split each block with a blank line, move the doc to its declaration, and delete the superseded half of each doubled doc.
  - (2) Widen SN-02:
    - Inspect `GenDecl` specs as well as functions.
    - Flag a block in which two lines open `// <Ident> <word>s`, any present-tense verb rather than a fixed list.
    - Add the shapes as `selftest.go` specimens.
  - `dc/docbind/main.go` is a working prototype: 38 fused and 17 orphan-name candidates, about 30 of them real.
- Regression risk: none for (1). For (2), possible new false positives, so run it over the tree first. Self-test is `go run ./tools/authoring -self-test`. UNPINNED for the new shapes.
- Size: M (pass S, tool M)
- Confidence: high for every listed site (each was read). The list may not be complete.

### DC-2: Comments that name a function, type or constant that does not exist
- Where and observed (each name grepped across `*.go`, including locals and fields; none is declared):
  - `app/bedfence.go:74`: "bedReachLine is the one-line form …". The method is `bedReach.Line`.
  - `app/burst_words.go:142`: "tapeText is one alert on the ticker tape" on `tapeHead`. Also `:160`: "the tape already strips at tapeText".
  - `domains/radio/synth/voice.go:31`: "hostileText guards the §10.5 contract" on `writeNarration`.
  - `modes/tty/broadcaster.go:1280`: "bedRow is the bed's selector" on `bedSelector`.
  - `modes/tty/broadcaster_lineup.go:195`: "scheduledLines is the SCHEDULED LINE-UP heading" on `scheduledSpan`. `scheduledLines` is also cited as the owner of the room calculation at `broadcaster.go:197`, `:705` (with `poolLines`, also gone) and `:871`.
  - `modes/tty/request.go:89` / `:146`: `requestValid` / `requestNote` on `valid` / `note`.
  - `platform/lineup/card.go:538`: "WithText fills in the script" on `WithScript`.
  - `modes/tty/broadcaster.go:650`: "THREE WIDTH OWNERS — `cardBoxWidth`, `orderWidth`, `bandWidth`". There is no `orderWidth`. The code has `frameWidth`, `tableWidth`, `cardBoxWidth`, `priorityWidth`, `upNextWidth`, `airBodyWidth` and `bandWidth`.
  - `modes/tty/broadcaster.go:1175` and `broadcaster_station_test.go:61`: "Not `sectionWidth()`". There is no such function.
  - `modes/tty/broadcaster_rail.go:77`: "a second list to keep in step with `bcRegions`". There is no `bcRegions`. The names live in the `railTone` switch (:86) and `bcRailForms` (:205).
  - `platform/lineup/projection.go:46` and `fence_view_test.go:14`: "`nextForAir` skips out-of-fence cards (D-75)". The skip is in `Lineup.Next` (`platform/lineup/lineup.go:251`).
  - `modes/tty/setup_scope_test.go:27`: "(setupScope.shownOn)". There is no `shownOn`. `:144`: "`stepGroup`" is `stepRow` (`setup_rows.go:452`).
  - `modes/tty/broadcaster_upnext.go:29`: "bcUpNextLabel and bcAlertLabel". There is no `bcAlertLabel`; that box's label is the literal "PRIORITY" (:195).
  - `platform/config/config.go:118`: "radioModes, castModes and toneModes are the closed sets Validate checks". There is no radioModes. `Validate` (:126-155) checks only `cast` and `tones.mode`. `[radio] mode` ("synth" | "relay", :48) is **not** validated, so the comment states a check the code does not make.
  - `app/director_pins_test.go:372,594,633` and `app/ticker_test.go:527`: `readBreaking` ("readBreaking sends the marquee message before …"). It does not exist, so these tests pin a contract by a name no reader can find.
  - `app/severe_read.go:155`: the name `Cancel` is justified by "radioDeck.Stop — which genuinely participates in a stopDwell cycle and carries its own ratified exemption". `stopDwell` and its ledger row are both deleted (F-140 CLOSED 2026-09-17).
  - `modes/tty/relayfault.go:122`: "relayFaultInset is that inset named once". There is none; the nearest is `relayFaultInsetLines` (:114).
  - `app/radio.go:1099`: "armDwell setting a five-minute timer, advanceQueue firing on it". Neither exists. It is meant as a counterfactual and should read as one.
  - `modes/tty/router.go:142`: "CutBed … — `[B]`". The bed key is `b` (`router.go:190`); `B` swaps to the Broadcaster.
  - `app/maintrack.go:66`: "rather than as one of the three". `stageNames()` (:59) has two stages, `off` and `dark`.
  - `app/pipelines.go:134`: seismic at 5 min is "the fire cadence", but fire is 10 min (:133). `:147`: RECENT seismic 15 min is "half the priority cadence", but 15 against 5 is a third.
  - `app/mapgeometry.go:14`: "the thing that will draw is under `platform/`". It is the external go-tuimaps module.
  - `app/platform_sim_test.go:18`: "(or `make test-linux`)". The Makefile has no such target.
- Cost: a junior greps the name in the comment and finds nothing. For `nextForAir`, `scheduledLines` and `readBreaking`, they also cannot find the code that holds the rule being explained. The config comment promises validation that does not happen.
- Proposed change: re-point each mention to the current symbol, or restate it in plain words. For `config.go:118`, either drop `radioModes` or add the `[radio] mode` check. The check would be a behaviour change at config load (look at `tty.ParseRadioMode`'s fallback first), so it needs the HUM LEAD.
- Regression risk: comment-only, except the optional `Validate` change, which is UNPINNED for an unknown mode.
- Size: S
- Confidence: high

### DC-3: follow-ups.md, the deferred-work list every reviewer is sent to, is about a third closed rows, and a dozen "open" rows are already done
- Where: `06_docs/follow-ups.md` (441 lines, about 65 KB, too large for one `Read`). Its rule at :9 is "Close a row by deleting it in the commit that does the work".
- Observed:
  - About 58 of the 170 `| F-n` rows carry CLOSED, ANSWERED, FIXED, DISPOSITIONED, Superseded or strike-through, and still sit under "Open". Rounds two to nine (:277-:421) are almost all closed.
  - Open rows the code already satisfies, each verified:
    - F-1: `config.Mutate`, `platform/config/config.go:518`, is the only writer.
    - F-2: `app/classifier_crosstable_test.go` is the coupling check.
    - F-3: the update check runs once at startup (`config.go:359`).
    - F-15: `Makefile:489` `lint:` exists and is in verify.
    - F-18: `platform/render/aa_completeness_test.go`.
    - F-35: `modes/tty/debug.go:191-211`, pinned by the 80x24 tests at `debug_test.go:146,194`.
    - F-36: `relayfault.go:78` stops the countdown.
    - F-80: `router.go:897-908` → `app/airscope.go:117` `takeTheAir`.
    - F-87: `app/pool.go:149-172` through `config.Mutate`.
    - F-89: `app/bedrelay.go:5-12` walks the station's fence.
    - F-100: `app/air_boundary.go:53` `monitorMayReachTheAir`.
    - F-173: `ci.yml:15-17` concurrency group.
  - Half-done rows that read as wholly open:
    - F-67: focus and blur are handled at `router.go:327-335`.
    - F-90: the journey now drives the console, `validate-journey.expect:394-401`.
    - F-138: one `t.Skip` is left (`broadcaster_uat_test.go:345`); turning it into `t.Fatal` closes the row.
    - F-55: the Broadcaster half is built.
    - F-107: `SpliceCells` now has a caller (`map_boxes.go:39`).
  - Rows about code that is gone:
    - F-94 cites `scriptWindow` (absent; the card shows a manifest and the detail modal is built).
    - F-105 cites `onRestored` (deleted, as F-108 itself says).
  - Wrong line citations:
    - F-211: `Makefile:539` is `p10:`; cgo is at :557. The row also says "oto v3.5.0", but `go.mod` has `v3.5.0-alpha.11`.
    - F-103: `operator.go:306` is past the end of a 296-line file.
    - F-104: `stale.go:187` is now :270.
    - F-62: `setup_form.go:151` is now :146.
- Cost: the release brief sends every reviewer here as "known, deliberately deferred work". A planner cannot tell owed from done without reading 1-3 KB rows. Several rows (F-80, F-89, F-100) describe Broadcaster defects as live, inviting a second "fix".
- Proposed change: delete closed rows per the file's own rule, moving any measurement worth keeping to its release record. Delete the 12 done rows, cut the 5 half-done rows to the remainder, rewrite or fold F-94 and F-105, and fix the four citations.
- Regression risk: `cmd/watchpost/docs_test.go:159-165` reads this file and fails if it cites fewer than 10 tests. Deleting closed rows must leave at least 10 citations (or the threshold is changed deliberately). The same test checks that every cited `Test…` exists.
- Size: M
- Confidence: high for the done rows marked as read in the fork report (`dc/fork-c-followups-changelog.md`). Medium for F-35, F-87 and F-89 (write path read, rendering not exercised).

### DC-4: Five places say platform/render is "the ONLY package that may import go-studs"; the Broadcaster imports it directly
- Where:
  - The claim: `platform/render/render.go:1` ("the D-9 pivot seam: the ONLY package allowed to import go-studs"), `platform/render/table.go:3` ("the ONLY go-studs consumer in the app"), `platform/render/lineup_table.go:10`, `docs/where-things-happen.md:111` and `docs/extending.md:36`.
  - The violation: `modes/tty/broadcaster.go:25` imports `third_party/go-studs/components`, used at :1381-1441 (`components.DataTableRow`, `ColumnDefinition`, `NewDataTableRow`).
- Observed: `go list -f '{{.ImportPath}} {{join .Imports " "}}' ./... | grep go-studs` lists `modes/tty` and `platform/render`. Within render, `lineup_table.go` and `status_table.go` import it as well, so "table.go is the only consumer" is false too. `scripts/lint-imports.sh` checks only `domains/` imports, so nothing enforces the rule.
- Cost: `docs/extending.md` tells contributors to keep go-studs inside render while the console already reaches past it. The D-9 "pivot seam" (swap the kit in one package) is not true today.
- Proposed change: either keep the seam (move `cardLane`'s row construction behind a render function) or restate the five sites ("render, and the Broadcaster's card lane"). Either way, add an enforcing check (a `singleowner`-style test or a line in `lint-imports.sh`).
- Regression risk: none for docs only. A move touches the console frame: the golden tests in `modes/tty` and the 14-allocation budget the comment at `broadcaster.go:1351` cites (`make alloc-budget`).
- Size: S (docs) / M (move)
- Confidence: high

### DC-5: The README's Settings instructions and screenshot, and where-things-happen.md, describe the pre-tabs window
- Where: `README.md:58` (`docs/img/setup.png`), `README.md:339` (`docs/img/broadcaster-settings.png`), `README.md:375-376`, `docs/where-things-happen.md:54`.
- Observed:
  - `setup.png` (last committed 2026-09-07) shows two columns, no tab row, and a footer of `tab Next question` / `esc Cancel`. The code has six tabs (`setupTabs()`, `modes/tty/setup_tabs.go:29-35`) in one column (`setupPage`, `setup_layout.go:282-285`, D-364). `tab` switches tabs (`setup.go:269`).
  - README:375 says "press `s`, `tab` to the key line" for the FIRMS key. The key row is on the same Data tab (`setup_rows.go:254`), so `tab` moves away from it. README:94 states the current rule, so the README contradicts itself.
  - where-things-happen.md:54 says "the body is two columns when they fit". `TestWhereThingsHappenNamesRealSymbols` checks names only, so it passes.
- Cost: the one step-by-step instruction for the only key the app takes fails as written. The first screenshot a user sees is a different window from the one they get.
- Proposed change: "press `s`, `↓` to the key line". Retake `setup.png` and `broadcaster-settings.png` (and check `help.png`, from 2026-09-07, before the `g`/swap keys). Rewrite the where-things-happen row as "tab row (`setupTabRows`), then the open tab in one column (`setupPage`)".
- Regression risk: docs and images only.
- Size: M (screenshots), S (text)
- Confidence: high for setup.png and the text. Medium for broadcaster-settings.png and help.png (not opened).

### DC-6: The as-built maps, which the atlas renders and which promise to match the code, are wrong in five places
- Where and observed:
  - `0.16.0-broadcaster-ui/03-architecture-design/as-built-air-and-deck.md:124,132`: "53 seams: 48 `tty.Config` func fields and 5 `tty.Radio` methods", airNone 34. The table that `TestEverySurfaceSeamIsClassifiedForTheAir` derives (`app/air_boundary_test.go:103-204`) holds 59 (54 + 5), with airNone 40. The newer seams include `FindPlace`, `PlaceNear`, `PropagationUpdate`, `SavePropagationAck`, `HistoryCost` and `SaveRadio`.
  - `as-built-air-and-deck.md:64-67`: `subgraph state["what MasterControl holds"]` holds power, monitor and air. These are Director fields (`platform/lineup/director.go:438-449`); MasterControl declares them, it does not hold them (`app/mastercontrol.go:37-104`). This blurs the five-role split the page exists to keep.
  - `observer-maps/03-architecture-design/as-built-map.md:132`: `severe.go · severeDeck.feedCopy — the ticker's feed` feeds the map's quakes. `feedCopy` does not exist. The quakes come from `lp.mapQuakes.fetch(lctx, ask.QuakeFeed) // D-122: the feed chosen, not the ticker's` (`app/maplayers.go:139`).
  - `as-built-map.md:46`: one "Maps" tab with "MAP | MAP - LAYERS · MAP - DETAIL, side by side". The code has separate `tabMaps` and `tabMapLayers` in one column (`setup_tabs.go:35`); the page's own prose (:62-63) says so.
  - `map-ready-geometry/03-architecture-design/as-built.md:38`: names `platform/tuimap`. No such package ever existed; the drawer is the external go-tuimaps module.
  - Minor: `as-built-map.md:100` `limitsOf` is `propanswers.LimitsOf`. `:157` "forty-three places" is now 83 sites. `:7`'s status line stops at 0.19.2.
- Cost: these pages are the "read the record, not the code" maps that the required-reading discipline relies on. Each error sends a reader to a symbol that is not there, or to a role holding state it does not hold.
- Proposed change: fix each node or count, and drop counts that a test already derives. Run `go run ./tools/atlas` in the same commit.
- Regression risk: `TestTheAtlasMatchesAFreshRender` until the atlas is regenerated. No behaviour change.
- Size: S
- Confidence: high

### DC-7: The architecture atlas presents plan-era diagrams as architecture without saying so, its first diagram included
- Where: `06_docs/architecture-atlas.html`, generated by `tools/atlas/main.go` from every mermaid block under `06_docs/` and `docs/`.
- Observed:
  - Of 59 diagrams, 13 come from as-built maps and 46 from plans and reports: `director-architecture.md` 8, `multi-voice-support/plan.md` 5, `quality-pass-plan.md` 5, `severe-alerts-modals/plan.md` 4, `por-report.md` 4, `0.16.0 diagrams.md` 4, `plan-report.md` 3, and others.
  - The page's only framing is "The tracked documents are the source of truth" (`tools/atlas/template.html`).
  - The first diagram, "The shape of the program" (`watchpost-cli/03-architecture-design/architecture.md:14-49`), has `TTS[OS TTS engines<br/>say · System.Speech · espeak-ng]`. `grep -rln 'espeak\|System.Speech' --include=*.go .` finds nothing (the voices are `say` and Piper, `app/voices.go:42`). It also omits Propagation/go-ionomaps, the history store, radar, temperature, air quality, UV, seismic and the global feed. Yet it carries "0.18.0:" annotations, so it looks maintained.
  - "The Director and the voices", diagram 0 (`director-architecture.md:24`): `radioDeck · tune · advanceQueue · armDwell` and `tickerDeck.cycle · capBurst`. `advanceQueue`, `armDwell` and `capBurst` do not exist. The source opens "Today there are two paths to audio", a 2026-09-01 picture.
- Cost: the atlas is the visual front door. A junior starts from replaced designs and method names that are not in the code. The freshness test proves the page is current with the documents, not that the documents are current with the code.
- Proposed change: in `tools/atlas`, label each diagram by its source kind: as-built (`as-built*.md`, `docs/`) versus plan or record. Render records under a clearly titled "design history" section, or leave them out. Separately, fix the watchpost-cli shape diagram (TTS node, the 0.19 sources) or mark that file as a record.
- Regression risk: the generated page changes. Pinned by `TestTheAtlasMatchesAFreshRender` and `TestAChangedDiagramChangesTheAtlas`.
- Size: M
- Confidence: high on the named examples. I did not audit all 46 plan diagrams.

### DC-8: The Makefile's comments sit on the wrong targets, contradict each other on timings, and describe two pins that do not move
- Where and observed:
  - `Makefile:80-88`: "Human-Accountability Attribution gate: no AI watermarks …" heads `lint-authoring`. It describes `lint-watermark` (:92). The next line calls the authoring rules ones "a grep can decide", which contradicts `tools/authoring/main.go:3-9` ("WHY IT IS A TOOL AND NOT A GREP").
  - `Makefile:343-347`: the "Deterministic allocation pins" paragraph heads `pty-severe`. Its target is `alloc-budget` (:506). This is SN-02's shape in a Makefile.
  - Mutant timings (`:141-147`, `:152-153`, `:175-179`, `:188-193`, `:437-439`): the figures are "~140s", "about 400s", "seven minutes" and "261 s on twelve cores", next to "835 mutants ran 93 minutes on a hosted runner" and `-timeout 240m`. `06_docs/mutants` holds 900 mutants today.
  - `HYGIENE_KEEP` (still `watchpost-0.14.1`, :236-256) and `VERDICTS_RECORD` (still the 0.16.0 folder's `mutant-verdicts.log`, :379-382) are each documented as "MOVES WITH THE RELEASE". Neither has moved, so `make mutant-verdicts` overwrites 0.16.0's shipped record.
  - `Makefile:470-474`: "the five that remain are RATIFIED AS OWED in wires-ratified.md". That ledger's exemptions table is empty (`06_docs/wires-ratified.md:27-32`), and `make wires` reports 0 unratified.
  - `pty-severe` and `journey` are missing from `.PHONY` (:2).
- Cost: the Makefile is where a junior learns the gates. Paragraphs attached to the wrong target, and timing figures that differ tenfold, steer process-depth decisions. The verdict pin damages a shipped release's record.
- Proposed change: move the two blocks to their targets and keep one dated, sourced mutant timing. For the two pins, the lead decides: derive them from the release in flight, or say they are fixed. Fix the wires sentence.
- Regression risk: comments, except the pins. Changing `HYGIENE_KEEP` changes what `hygiene` deletes (UNPINNED).
- Size: S
- Confidence: high on the mismatches. I did not re-time the gate.

### DC-9: AP-HIST-01 is enforced on Go only; the Makefile and scripts narrate dated incidents, including the catalogue's own "used to"
- Where:
  - `Makefile:583-586`: "It used to be written here by hand, so a bump wrote new content into the old file's name … (0.17.0)".
  - `Makefile:157-158`: "All of that was shell until 2026-09-03, and 17 of one session's 34 defects were in that shell".
  - `Makefile:354-357`: "IT HAD NO TARGET UNTIL 2026-09-06 … stayed red, unnoticed".
  - Also `Makefile:371,412,467-470`; `scripts/lint.sh:35` ("which used to read as clean"); `scripts/lint-imports.sh:23-24,50`; `scripts/quality/p10-unmatched_test.sh:23-24` ("Found by planting a renamed flag, 2026-09-08"); `scripts/quality/ledger-ratified.sh:29-30` ("this rewrite … The first version scored …"); `scripts/quality/mutant-verdicts.sh:11,32,42,63,132`; `scripts/quality/mutant-anchors.sh:12`; `.gitignore` ("Two tool binaries reached commits while these lines stood").
- Observed: `tools/authoring` walks `*.go` only (`main.go:73`). I count about 30 narrating comment lines in these files. Two of them use "used to", which the Go tool would fail at once. `required-gates.txt:3-6` and `docs/accepted-costs.md:69-72,101-103,112-118,229-232` narrate their own corrections the same way.
- Cost: the gate files tell September's stories instead of what each target guarantees. The rule reads as "Go only", which `06_docs/code-standards.md` does not say.
- Proposed change: one reader pass that keeps every WHY, restated in the present tense, and moves the incidents to build logs. Optionally scan `#` comments in `Makefile` and `scripts/**` with the same phrase list, adding a non-Go self-test specimen.
- Regression risk: comment-only. `parseRequired` ignores `#` lines in `required-gates.txt` (pinned by `TestTheThreeGateListsAgree`).
- Size: S-M
- Confidence: high for the quoted sites. The count comes from a phrase grep.

### DC-10: Work deferred "to 0.16.5", a release that never existed, in four living docs; two perf items are tracked nowhere else
- Where: `06_docs/code-standards.md:5` ("the ~123 history-narrating lines … are F-137's, deferred to 0.16.5"); `06_docs/perf-measurement.md:202-209` ("Carried to 0.16.5"); `06_docs/tooling-backlog.md:39,40,78`; `06_docs/follow-ups.md` F-121, F-133, F-137, F-139.
- Observed: there is no 0.16.5 folder; the tree went from 0.16.0 to 0.17.0. perf-measurement's two carried items exist only there: geocoder warm-up at start-up (a measured ~700 ms on the first search, :204-206) and whether a partial name may resolve from the pool. `git grep -i 'warm.*geocod'` finds nothing, and follow-ups.md never mentions the geocoder.
- Cost: deferred work points at a release that will not come, so the 0.19.5 pass will not see it.
- Proposed change: re-target each to a follow-up row (add rows for the two perf items) or to 0.19.5.
- Regression risk: none.
- Size: S
- Confidence: high

### DC-11: The modes/tty and modes/report package docs state the wrong import rule and the wrong schema version
- Where and observed:
  - `modes/tty/dashboard.go:1-2`: "Package tty … reads ONLY platform/snapshot per the import lint". `go list -f '{{join .Imports "\n"}}' ./modes/tty` shows 15 platform packages (bucket, category, debounce, geo, httpx, invariant, lineup, plaintext, propanswers, render, report, snapshot, term, tz, units), plus go-studs, go-tuimaps and go-ionomaps. The lint forbids only `domains/`.
  - The same doc, :4-11: "Dashboard is the B3 build of mock rev2 (125-col) … radio player frame (static until B4)". This is a build-era snapshot narrating history.
  - `modes/report/report.go:1-2`: "--json (machine, schema v1.0-rc)". `platform/snapshot/types.go:23` is `SchemaVersion = "1.1.0-rc"`.
  - `app/app.go:1-2`: "the ONLY place that wires domains, platform, and modes". `cmd/watchpost` imports `modes/report` and `pkg/schema` itself (minor).
  - `app/app.go:35`: `ReportOnce` "(B1a scope; scheduler-driven live mode arrives B1b/B3)".
  - `domains/locations/coverage/coverage.go:5`: the package doc carries a bug story ("red-team 0.9.0 round 2 N-2: the offline path refused Puerto Rico while the online path accepted Paris").
- Cost: the package doc is the first thing `go doc` shows. "reads ONLY platform/snapshot" makes a junior refuse a legitimate `platform/units` import, or believe the tree breaks its own rule.
- Proposed change: state the real rule ("platform/* yes, domains/* never; lint-imports.sh"). Replace the B3/B4 narrative with what Dashboard is now. Cite `snapshot.SchemaVersion` instead of a literal.
- Regression risk: none
- Size: S
- Confidence: high

### DC-12: duplicates-ratified.md ratifies a duplicate that no longer exists, for a reason that is false; its test count has doubled
- Where: `06_docs/duplicates-ratified.md:42` (row `7dbf0b6d0c66`), `:17`, `:20-21`; `platform/lineup/director.go:591-615`.
- Observed: the row says `stepBed` and `stepOperator` "became identical only when `Requested` made the operator's switch a fourth case". `stepOperator` has three cases (`Moved`, `Requested`, `Dropped`; `Restored` left with D-259). `go run ./tools/dupes -json` reports only `bb46a41bc37f` and `3462cd9e225c`, so the row matches nothing. `tools/dupes` has no stale-row check (`tools/wires` does). `:17` says "-tests: 17 unexplained across 18 groups"; `go run ./tools/dupes -tests` reports "37 … groups; 2 ratified, 35 NOT".
- Cost: a dead exemption in the HUM LEAD's ledger, test duplication understated by half, and every future collapse can leave another dead row.
- Proposed change: delete the row and drop or refresh the hard count. Give `tools/dupes` the stale-row failure `tools/wires` has (with a self-test specimen).
- Regression risk: `make dupes` reads the table, and deleting a non-matching row cannot fail it. The stale-row check is new behaviour.
- Size: S / M
- Confidence: high

### DC-13: tooling-backlog.md, required-gates.txt and red-team-brief.md describe gates and tests that were replaced
- Where and observed:
  - `06_docs/tooling-backlog.md:36`: `TestEveryRequiredGatesCheckerHasAControl` (now `TestEveryControlIsReached`, `gates_test.go:488`).
  - `tooling-backlog.md:37`: `TestNoRequiredGatesCIStepIsSilentlyConditional` (now `…IsSilenced`, :282).
  - `tooling-backlog.md:41`: `makeLines` (absent).
  - `tooling-backlog.md:69`: lint.sh `|| true` (gone; F-118 CLOSED).
  - `tooling-backlog.md:79`: `declset_test.go` (deleted; F-122 CLOSED).
  - `tooling-backlog.md:85-88`: "four ways the three-list rule can still be defeated" cites `ciTargets`, `targetDeps` and `TestEveryRequiredGateIsStillRun` (all absent). Two of the four ways are CAUGHT as A17 and A18 in `gate-attack-list.md:39-40`.
  - `06_docs/required-gates.txt:3-6`: argues from `TestCIAndVerifyRunTheSameGates` (now `TestTheThreeGateListsAgree`, `gates_test.go:187`).
  - `06_docs/red-team-brief.md:79-80`: "Questions 5, 6 and 9 are mechanically checkable". Question 5 is SN-01 (names), which no tool checks. It, and `tooling-backlog.md:19,99`, list three authoring rules where `checkFile` runs five (`tools/authoring/main.go:177-183`).
  - `tools/authoring/main.go:4`: "Two of the three rules here", in the tool's own doc.
  - `tools/authoring/selftest.go:18-19`: "taken verbatim from `code-authoring-standards.md`". That is a private-harness file not in the tree; `06_docs/code-standards.md:8-12` exists precisely so public references don't point there.
  - `gate-attack-list.md:11`: "17-specimen self-test"; `-self-test` reports 36.
- Cost: the backlog sends contributors to fix closed defects and look up absent symbols. Blind reviewers are told to skip a lens no tool covers.
- Proposed change: point each cell at the current test, strike the closed items, and replace "four ways" with a pointer to gate-attack-list.md. Rewrite the required-gates header in the present tense. List all five authoring rules and say SN-01 is a reader's job. Fix the self-test reference to `06_docs/code-standards.md`.
- Regression risk: none (`parseRequired` ignores `#` lines).
- Size: S
- Confidence: high

### DC-14: extending.md, Walkthrough 2 ("add a provider"), gives the wrong steps
- Where: `docs/extending.md:50-51`.
- Observed:
  - "give `newAssembler` its attribution case (About renders it), and add it to `credits.go` (≤ 52 cells)". About draws `creditGroups` through `creditBlocks` (`modes/tty/help_about.go:344,435`). The assembler's attribution reaches the report (`modes/report/report.go:89`), not About. There is no 52-cell bound anywhere.
  - "a tier line in `startPriority` and one in `newFor`". The tiers live in `priorityTiers()` / `recentTiers()` (`app/pipelines.go:125,140`, "the one owner"); the other two only read them.
- Cost: the page says "the steps are verified against the shipped code". A contributor edits the wrong function, or works to a limit that does not exist.
- Proposed change: name `creditGroups` / `CreditLine`, the report as the attribution's destination, and `priorityTiers()` / `recentTiers()`.
- Regression risk: none. No test reads `docs/extending.md`; `cmd/watchpost/docs_test.go` checks only where-things-happen.md and accepted-costs.md. That gap is why this page drifted.
- Size: S
- Confidence: high

### DC-15: The Settings code still introduces itself as the two-question setup form
- Where:
  - `modes/tty/setup.go:32-46` (doc of `type setupState`): "One form, both questions on screen … the default location … and the optional NASA FIRMS key", fused with "setupFocus names …" (no such thing) and "Twenty rows across four groups".
  - `modes/tty/setup_rows.go:3-5`: "Twenty rows across four groups".
  - `setup_rows.go:57`: "tab jumps to the next group's first row".
  - `modes/tty/map_settings.go:3`: "the map's two Settings".
- Observed: `setup_rows.go:25-35` declares 11 groups and about 57 rows before `setupRowCount` (:165). `tab` switches tabs (`setup.go:269`). map_settings.go draws about 20 rows (:140-158).
- Cost: the central Settings types are introduced at a third of their size, with the wrong key model.
- Proposed change: describe the table, its groups and tabs, and the focus index, without hard-coded counts.
- Regression risk: none
- Size: S
- Confidence: high (row count approximate)

### DC-16: History narration in Go that the phrase list misses: mostly "round N" red-team citations used as the reason (F-137's class)
- Where (production, not tests): 38 lines. By package: modes/tty 13, app 8, platform/render 4, domains/radio/synth 3, domains/radio/script 3, domains/radio/player 2, and 1 each in platform/config, domains/severe, domains/radio/pronounce, domains/locations/openmeteo and domains/locations/coverage. Of these, 34 are "(round 4, B-01)" / "(red-team round 4, A-04)". Test files hold another 36.
- Examples:
  - `platform/render/panel.go:177` "(round 4, B-01)".
  - `domains/radio/synth/normalize.go:87` "(round 2 N-1: "CA" at index 0 sliced runes[-1:-1]); "CA-…" was handled above".
  - `app/app.go:30` "it read 0.9 for five releases".
  - `platform/snapshot/harmonize.go:113` "Exported because domains/marine/ndbc held a byte-identical copy".
  - `modes/tty/dashboard.go:645` "the map's other keys wait for W1.16's one ruling". They are all bound, in `defaultMapKeyMap` (`map_pane.go:940-970`).
  - `cmd/watchpost/scratch_test.go:3` "(0.18.0 batch 16)".
  - `broadcaster_heldband_test.go:137` "which is how the first draft shipped".
  - `broadcaster_standby_test.go:74` and `platform/lineup/rotation_test.go:42,128` "Found by a plant that survived".
- Observed: known as F-137 (`06_docs/follow-ups.md:261`), still marked "DEFERRED to 0.16.5" (see DC-10). The counts above come from my phrase set (`dc/narr.tsv`), not from F-137's 123.
- Cost: "round 4, B-07" means nothing without the 0.9.0 red-team file, and the sentence usually states the real reason already. This is cheap now: deleting the parenthetical is mechanical.
- Proposed change: one pass that deletes "round N," citations where the sentence already gives the reason and rewrites the half-dozen real narrations. Then optionally add `\bround \d+,? [A-Z]+\d*-\d+` to `histPhrases`.
- Regression risk: comment-only.
- Size: S-M
- Confidence: high on these counts. My phrases certainly miss some.

### DC-17: Orphan comments bound to nothing, one cut off mid-sentence
- Where:
  - `modes/tty/severe.go:228`: "// severeArrows are the chip labels per --ascii: the words, not the", a fragment followed by a blank line. No `severeArrows` exists.
  - `app/dashboard.go:537-538`: `tickerAlert` (gone).
  - `modes/tty/detail_marine.go:173`: `tideTrend` (gone).
  - `modes/tty/detail.go:327` and `detail_test.go:372`: "(whenCols)".
  - `modes/tty/broadcaster_detail.go:52`: "(slotRows)". Both names are gone.
- Cost: dead text that readers try to attach to the code around it.
- Proposed change: delete, or name the current owner.
- Regression risk: none
- Size: S
- Confidence: high

---

## Minor (one line each)

- `docs/accepted-costs.md:108`: `modal_location.go:49` for `RecentCap`. It is at :113; use `modal_location.go:RecentCap`, which the symbol test checks.
- `README.md:563-564` / `cmd/watchpost/root.go:122`: `--verbose` also prints the cast and tone answers (`root.go:103-107`), which neither text mentions.
- `README.md:90`: "asks three things, all on one screen". The Data tab also carries the three HISTORY rows.
- `README.md:384-386` against F-57: the README says eleven tokens sit outside the AA register; F-57's three classes sum to 10, and it excludes `radio.bg`.
- `README.md:341-353`: the Broadcaster key table omits `space` "Play/Pause Bed Relay" (`router.go:209`).
- `README.md:555-570`: `WATCHPOST_DEBUG_MAPFRAMES` / `_TEXT` (`app/framelog.go:27`), `WATCHPOST_MAINTRACK` and `WATCHPOST_WATCHLIST_DWELL` are read in production and documented only in follow-ups.
- `docs/extending.md:200`: `vuln` is "the one step that needs the network", but `lint` also fetches golangci-lint (`scripts/lint.sh:36`).
- `06_docs/perf-measurement.md:222,231,23`: the workload modes omit `g1b`, the events omit `answered:layers` (`map_pane.go:1377`), and the soak has 15 columns, not 14. `modes/tty/timing.go:24` omits `m5` and `answered:layers`.
- `06_docs/quality-plan.md:75`: line counts 115 / 228 are now 139 / 251. `:195`: the INST review "due at 0.17.0" has no record after 0.16.0.
- `06_docs/tooling-backlog.md:97` and F-156: "23" scripts ports; there are 24 non-Go sources today.
- `perf-measurement.md:3,67` and `quality-plan.md:162` cite `perf-protocol.md` / `build-methodology.md` with no path.
- `as-built-map.md:104`: the MAP row order differs from `map_settings.go:140-158` ("RX noise", with Colours and Playback after Opens at).
- `pkg/schema/watchpost-report.v1.0.0-rc.schema.json` is kept deliberately (CHANGELOG.md:252). It is not stale, just noting it.
- The local, git-ignored `watchpost/CLAUDE.md` still names 0.19.0 as "the release in flight" and makes its REQUIRED-READING mandatory at startup. Not public, but every agent session reads it first.

## What is in good shape (do not spend time here)

- **No TODO / FIXME / XXX / HACK markers** anywhere in the tree's Go, Makefile, scripts or living docs.
- **Every Go package has a package doc comment.** The only exceptions are the test-only `domains/alerts` (whose `alerts_test` package is documented) and the vendored `go-studs/tokens`.
- **The atlas is fresh** (`go test ./tools/atlas` ok). The problem is what it renders (DC-7), not drift.
- **CHANGELOG 0.19.0-0.19.3**: every sampled claim (keys, Settings tabs and rows, playback speeds and the flash ceiling, the throttle file, go.mod versions, GIRO rules, request share, the 8 MB history cache) matches the code. There is no Unreleased section.
- **README key tables and config blocks**: Observer, map, chart and Broadcaster keys, every config.toml key and default, themes, voice roles, seismic bands, floors and cost thresholds all match the code (except DC-5 and the minor items).
- **p10-ledger.md** (165 rows, every symbol resolves), **wires-ratified.md** (every symbol wired; `make wires` 95 members, 0 stale), the collapsed table in duplicates-ratified.md, and code-standards.md's targets and symbols are all correct.
- All Make targets that README, docs and 06_docs name exist. The only dead one is `make test-linux` in a test comment (DC-2).
- **Narrative-word hygiene in Go is mostly good**: "no longer", "now" and "previously" occur almost only in present-tense senses. The narration that remains is the round-citation class (DC-16).

## What I did not cover

- I did not read all 53k comment lines by hand. DC-1, DC-2, DC-16 and DC-17 come from AST and phrase scanners plus a read of every hit, so a comment that is wrong in *substance* (a wrong number or behaviour, with valid names) is found only where I or a fork happened to check it. That is the larger class, and it still needs readers per package.
- The 46 plan and report diagrams in the atlas were not audited node by node; only the examples in DC-7 were.
- `third_party/` comments were excluded (vendored).
- follow-ups rows that wait on a ruling rather than code (about 19 pre-0.16 rows) were checked only for symbol existence. The CHANGELOG before 0.19.0 was not checked.
- Screenshots other than `setup.png` were not opened.
- I counted narration with phrase sets. F-137's "~123" uses a different, unpublished method, so the numbers are not comparable.
