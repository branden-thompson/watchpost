# JR: junior readability (goal 5)

Lens: could a junior maintainer who is new to this codebase find their way, understand what a piece
of code does, and change it safely? I read it the way that person would. I started with README.md,
docs/where-things-happen.md, docs/extending.md and 06_docs/code-standards.md. Then I traced four
first tasks: adding a Setting, adding a data source, changing a tty screen, and changing what the
radio reads. Commit `4c803d99`. The repository was not modified.

How I measured. I wrote three small `go/ast` programs in my scratch directory
(`<lens scratch>/jr/fmetrics`, `names`, `fused`), and their output is in `<lens scratch>/jr/*.tsv`:
- per-function lines, nesting depth, branch count and parameter count, over 4,282 non-test functions;
- per-file length and whether the file has a package doc;
- one-letter struct fields;
- SN-02's own "doc opens with another name" test, applied to types, values and struct fields.

Comment patterns were counted with grep over non-test Go, excluding `third_party/`. The command for
each count is described where the number appears.

## Headline

The code is small at the function level. P10 keeps functions short: only 48 of 4,282 exceed 60
lines, and only 37 have more than 15 branches. Most of the long ones are long because of comments,
not code. **What blocks a junior is reference and navigation**, not complexity:
- ruling IDs that cannot be resolved;
- doc comments bound to the wrong declaration;
- a 31k-line package whose only map is stale;
- vocabulary used everywhere and defined nowhere;
- no walkthrough for the most common first task (adding a Setting).

---

### JR-1: Bare ruling IDs cannot be resolved: `D-n` restarts in every release, and 2,817 comment cites name no release
- Where: across the tree. Examples:
  - `modes/tty/dashboard.go:83-85` (`MapProblem … (D-124)`)
  - `modes/tty/dashboard.go:89` (`MapRadar … (D-85)`)
  - `modes/tty/dashboard.go:671` ("legal under D-15")
  - `docs/extending.md:14` ("Keybindings are data (D-15)")
  - `modes/tty/router.go:3` ("(FR-1.1, D-2)")
- Observed:
  - **The volume.** 2,817 comment lines in non-test Go cite a `D-n`. Only 103 of them qualify it with
    a release (`grep -oE '[0-9]+\.[0-9]+\.[0-9]+('s)? D-[0-9]+'`).
  - **The collision.** Each release's rulings table restarts at D-1:
    - `observer-maps/02-analysis/rulings.md` holds 283 D-rows;
    - `propagation-overlays/02-analysis/rulings.md` holds 368;
    - `watchpost-cli` holds 28, `0.16.0-broadcaster-ui` 20, `multi-voice-support` 15.
  - **The same number means different things.** I checked four pairs:
    - D-124 is "errors need a path" in 0.18.0, and "evidence lived in a temp directory" in 0.19.0.
    - D-85 is "preload the radar" in 0.18.0, and "audio-output device" in 0.19.0.
    - D-142 is "radio-group tint" in 0.18.0, and "P10 sees a discarded ok" in 0.19.0.
    - D-15 has **four** meanings: keybindings are data, `[g]` opens the map, 0.19.0 scope, and
      0.16.0 R-8 cadence.
  - **Not just D.** About 45 ID families appear five or more times in comments: D 2,817, P10 648,
    FR 464, F 175, MVS-D 163, DR 143, U1/U2 135, AX 51, NFR 39, AI 36, SAM-D 21, BD 18, and more. The
    "Record IDs" table in `docs/where-things-happen.md:96-104` covers about 12 of them, and **not D,
    FR, F, MVS-D, DR or W at all**.
  - **FR has the same collision.** `actMap` cites "0.18.0 FR-1.1"; `router.go:3` cites a bare FR-1.1,
    which is 0.16.0's.
  - Some cites point at nothing in the tree: "ruling 2" (`app/pool.go:343`), "Option C"
    (`app/app.go:1`), "B3" and "M1" (`app/dashboard.go:1-3`).
- Cost:
  - A junior who greps `D-124` gets four files and may read the wrong ruling as the authority for
    the code in front of them. With 0.18.0 and 0.19.0 both on the map code, there is no reliable
    way to tell which is meant.
  - About 15% of all comment lines (4,260 of 28,798) cite an ID, so the problem is on most screens.
  - `06_docs/code-standards.md` already states the principle this breaks: "a reference they cannot
    reach makes every other reference less trustworthy".
- Proposed change:
  - (a) A generated index `06_docs/ids.md`. For each prefix+number it lists every release's row
    with its one-line subject, built from the rulings tables the same way the p10 ledger mirror is
    built. A junior then sees every candidate in one place.
  - (b) A short paragraph in where-things-happen "Record IDs" saying D/FR/F/DR are per-release, and
    pointing at the index.
  - (c) Going forward, an authoring rule: a new comment cites `0.19.0 D-142`, not `D-142`. Ask the
    lead before retro-qualifying the old 2,817.
- Regression risk: none for behaviour; this is docs plus a generator. `TestWhereThingsHappenNamesRealSymbols`
  (cmd/watchpost/docs_test.go) still guards the page.
- Size: M (index generator and doc). Retro-qualifying the old cites would be L.
- Confidence: high on the counts and the four collisions. I did not try to work out which release
  each of the 2,817 cites means.

### JR-2: Doc comments bound to the wrong declaration: SN-02 checks only funcs, and 11+ type/value/field docs are fused or detached
- Where (each found by applying SN-02's own opener regex to non-func declarations; scan in `<lens scratch>/jr/fused`):
  - `app/schedule.go:54`: `startSchedule`'s doc (21 lines) is fused onto `type bedSeams`. `go doc -u ./app startSchedule`
    prints **no doc**, and `go doc -u ./app bedSeams` opens with "startSchedule builds the Director…".
  - `app/mainread.go:149`: `readCard`'s doc is fused onto `var errReadStopped`, and `readCard` has none.
  - `modes/tty/dashboard.go:387`: `Stats`'s doc is fused onto `ZoneShapeStats`. `type Stats` (`:398`) is undocumented.
  - `app/executors.go:57`: field `read` carries **two** docs that disagree. The first says it "returns
    whether the words ran out" (a bool); the second, and the type, say it returns an `error`.
  - `domains/radio/cast/tone.go:10`, `domains/globalfeed/usgs.go:45`, `domains/weather/nws/provider.go:45`,
    `modes/tty/dashboard.go:584`, `modes/tty/memo.go:60`, `platform/render/table.go:399`,
    `platform/render/units.go:124`: the same fusion, 1 to 5 lines deep.
  - Detached, not fused: `modes/tty/dashboard.go:274-284` is "THE STATION'S OWN TWO … `Transmitter` IS A
    POINTER because…". It sits glued to `MoveCard`/`DropCard`'s doc, **48 lines above** the fields it
    explains (`Transmitter`, `:322`), and those fields are undocumented.
- Observed: `tools/authoring/rules.go:179` loops `for _, d := range f.Decls { fn, ok := d.(*ast.FuncDecl)`.
  GenDecl (types, vars, consts, struct fields) is never checked. `go run ./tools/authoring` reports "no findings".
- Cost:
  - A junior reading `go doc` or hovering in an editor is told that `bedSeams` builds the Director,
    and is told nothing about `startSchedule`, the function that wires the schedule.
  - At `Transmitter` they never see why it is a pointer. That is exactly the nil-means-unset fact
    they need not to break.
  - At `executors.read` they are told the wrong return type.
- Proposed change:
  - Extend `checkDocAttached` to TypeSpec, ValueSpec and struct-field docs, with self-test specimens
    for each.
  - Fix the 11 sites by inserting the blank line or moving the block.
  - Move the Transmitter paragraph down to its fields.
  - Delete the stale first half of `read`'s doc.
- Regression risk: comments only, plus the checker. `go run ./tools/authoring -self-test` pins the
  checker. No behaviour changes.
- Size: S.
- Confidence: high. Every site was confirmed by reading it, and `startSchedule`/`bedSeams` with `go doc`.
  The opener regex misses blocks that open with an ALL-CAPS lead, so there are probably more
  (the Transmitter case was found by reading, not by the scan).

### JR-3: `modes/tty` (85 files, 31k lines, 739 `Dashboard` methods in 68 files) has one entry point, and it is stale and contradicts the code
- Where: `modes/tty/dashboard.go:1-10` (the package doc), `modes/tty/dashboard.go:649`, `modes/tty/router.go:3`
- Observed:
  - **The package doc** says: "Dashboard is the B3 build of mock rev2 (125-col): header, conditional
    alert module, radio player frame (static until B4), priority table … defaults per D-19: a/A/ctrl+a,
    f/c units, q quit." It describes the 0.9-era shape and cites build batches. It never mentions the
    Router, the Broadcaster, the map, Settings, or how the 85 files are grouped.
  - **The type doc** says `// Dashboard is the root TTY model.` (`:649`). But `router.go:3` says "the
    model the program holds", and extending.md says the Router is the model handed to the program.
  - **The scale.** `Dashboard` has 739 methods spread over 68 files; `Broadcaster` has 90 over 13. The
    struct is 202 lines and `Config` is 313 lines (`:73-386`). `dashboard.go` itself is 2,000 lines,
    and its first 650 are message types and Config before the type appears.
  - The file prefixes are a good implicit map: `setup_*`, `map_*`, `map_prop_*`, `broadcaster_*`,
    `detail_*`, `status*`. But nothing states that map.
- Cost: a junior told to "change the Settings window" or "change the map legend" opens the package
  doc and learns nothing current. They may believe Dashboard is the program root and put
  program-wide message handling there instead of in `Router.Update`.
- Proposed change: replace the package doc with a `doc.go` that has:
  - one paragraph on the shape (Router → two surfaces; Observer = `Dashboard`; Broadcaster);
  - a table of file prefixes and what each owns;
  - a pointer to where-things-happen.

  Fix `:649` to "Dashboard is the Observer surface; the Router holds it."
- Regression risk: comments only. UNPINNED, and no test needs to change.
- Size: S.
- Confidence: high.

### JR-4: The `app` package doc is three fragments that `go doc` joins, all from 0.9, describing about a tenth of a 20k-line package
- Where: `app/app.go:1-2`, `app/dashboard.go:1-4`, `app/setup.go:1-6`
- Observed: `go doc ./app` prints these as one doc:
  - "the composition root (Option C)";
  - "RunDashboard wires the live pipeline (B3): config locations → NWS provider → tiered scheduler…";
  - "M1 instrumentation…";
  - "Setup (UAT 100, …) … (B2 red-team #2)".

  There is nothing about the radio deck, the arbiter, mastercontrol, the schedule/executors/pump,
  the maps, or history, which make up most of the package's 75 files.
- Cost: a junior who is told "change what the radio reads" has no map from the package doc to
  `radio.go` / `mainread.go` / `executors.go` / `schedule.go`. "Option C", "B3" and "M1" are not
  defined anywhere they can reach (see JR-1).
- Proposed change: one package doc in `app/doc.go`, giving the composition root's job in plain
  words and a short file-group table:
  - pipelines and snapshot: `pipelines.go`, `dashboard.go`;
  - radio: `radio.go`, `mainread.go`, `cast.go`, `voices.go`;
  - narration: `director.go` (the arbiter), `mastercontrol.go`;
  - schedule: `schedule.go`, `executors.go`, `pump.go`;
  - maps: `map*.go`;
  - history.

  Turn the other two fragments into file headers (a comment after `package`).
- Regression risk: comments only. UNPINNED.
- Size: S.
- Confidence: high.

### JR-5: One name for two concepts, two names for one: Director/director, Dashboard/Observer, setup/Settings, report, lane
- Where and observed:
  - **Two "Directors".** `app/director.go` declares `type director` (`:117`), the narration *arbiter*.
    `platform/lineup/director.go` declares `Director`, the scheduler.
    - The file's own header admits it: "The two collide in the word and in the file name, and both
      have a `settle` (red team 2026-09-05, R-7/Junior-Dev 5)" (`app/director.go:5-8`).
    - where-things-happen uses both words within a few rows ("the arbiter `app/director.go:Run`",
      "the **Director** decides: `platform/lineup/bed.go`").
    - The vocabulary table defines only the lineup one.
    - 0.16.0's REQUIRED-READING says "There is no role called the ARBITER. The word means today's
      `app/director.go`." That page is not linked from docs/ (JR-6).
  - **Observer vs `Dashboard`.** Users and docs say Observer (324 comment mentions in app/ and modes/);
    the type is `Dashboard`. "the dashboard" also means the screen with no window open
    ("`modalNone` is the dashboard alone"). `app/dashboard.go`/`RunDashboard` is the composition root.
  - **Settings vs setup.** The window is "Settings" to users. In code it is `setup*.go`, `modalSetup`,
    `d.setup`, the `watchpost setup` command, and "Setup window" (34 comment hits) next to "Settings
    window" (20).
  - **report.** `platform/report` is "what a location report can be MADE OF" (radio report kinds).
    `modes/report` is the one-shot stdout/JSON renderer.
  - **lane.** It means:
    - the HTTP client's pacing queues (vocabulary table);
    - a ticker category's band (`platform/category.Lanes`);
    - the console's three columns (`Broadcaster.lanes`, `broadcaster.go:767`);
    - a gate lane (`verify-docs`);
    - and, as a local variable, a text **width**: `lane := b.bandWidth()` (`modes/tty/broadcaster.go:1179`).
- Cost:
  - A junior asked to fix "the Director's ducking" can open `app/director.go` and change the arbiter.
    That is the exact confusion behind the 2026-09-09 revert the common brief warns about.
  - A junior asked to add a "report" can start in the wrong package.
- Proposed change, smallest first:
  - (a) Add vocabulary rows for "the arbiter (`app/director.go`, type `director`), NOT the
    Director", "Observer = type `Dashboard`", "Settings = `setup*` in code", and the two report
    packages. Mark "lane" as overloaded.
  - (b) Rename the local `lane` in `stationLine` to `width`.
  - (c) Ask the lead: rename `app/director.go`/`director` to `arbiter.go`/`arbiter`. This is a
    rename, not a role change. The five roles stay. About 41 references in `06_docs/mutants/` would
    need re-pointing (memory: grep mutants before moving code).
- Regression risk: (a) and (b) none. (c) touches the mutant anchors (`make mutant-anchors`,
  `make mutant-check`) and `TestWhereThingsHappenNamesRealSymbols`. Behaviour is unchanged.
- Size: (a)(b) S. (c) M.
- Confidence: high on the observations. (c) is a ruling question.

### JR-6: Vocabulary a junior meets on every screen is defined nowhere they would look; the architecture doc is not linked
- Where: `docs/where-things-happen.md:72-94` (the vocabulary table); README.md; docs/extending.md:173,177
- Observed:
  - **Undefined terms.** Comment mentions (case-insensitive, whole-word) of terms **absent** from
    the vocabulary table:
    - card-pipeline roles: Producer 153, Composer 85, Reader 216 (the five roles);
    - radio and narration: duck 71, burst 212, programme 108, segment 156, settle 90, correspondent 52;
    - pool 253, surface 261, "HUM LEAD" 414.
  - **The five roles.** They are defined only in `06_docs/02_features/0.16.0-broadcaster-ui/00-REQUIRED-READING.md:25-29`,
    which README.md and docs/ never link.
  - **The architecture doc.** The system architecture is at
    `06_docs/02_features/watchpost-cli/03-architecture-design/architecture.md`, a feature folder named
    after the 0.9 release. README.md and docs/ never link it by path.
    - extending.md says "say in `architecture.md §1.1`" (`:173`). But two `architecture.md` files
      exist (watchpost-cli and propagation-overlays), and only the first has §1.1.
    - Nineteen code comments cite "architecture §n" without a path.
  - **The atlas.** `06_docs/architecture-atlas.html` is not linked from README or docs either.
- Cost: a junior has to rebuild the card pipeline's roles, and what "duck", "burst" and "pool"
  mean, from code. They cannot find the architecture overview without already knowing the folder
  name.
- Proposed change:
  - Add the missing terms to the vocabulary table: the five roles with one line each, plus duck,
    burst, pool, programme, segment, surface, correspondent/cast/role, and HUM LEAD (the project's
    human lead).
  - Add a "Start here" block to README's "For tinkerers / Building from source": architecture.md
    (full path), the atlas, where-things-happen, extending, code-standards, and the 0.16.0
    REQUIRED-READING for the Broadcaster.
- Regression risk: docs only. Any symbol added to where-things-happen is checked by
  `TestWhereThingsHappenNamesRealSymbols`.
- Size: S.
- Confidence: high on the counts. The term list is mine: these are the ones that blocked my own
  first read.

### JR-7: There is no walkthrough for adding a Setting, the most common first task; one boolean threads through about 12 sites in 9 files, including a 1,410-character line
- Where: traced from `reduce_motion` (0.19.3):
  - `platform/config/config.go:275`: the field;
  - `app/themes.go:91`: persist;
  - `app/dashboard.go:376`: into `tty.Config`;
  - `modes/tty/dashboard.go`: `:181` (Config), `:455` (UIPrefs), `:721` (Dashboard field), `:1000` (NewDashboard);
  - `modes/tty/setup_rows.go`: `:88` (row id), `:263` (table row);
  - `modes/tty/setup_layout.go:372`: label;
  - `modes/tty/setup.go:390`: arrow handling;
  - `modes/tty/setup_ui.go`: `:61-65` (draw), `:149` (collect), `:184` (apply).
- Observed:
  - docs/extending.md has five walkthroughs (detail row, provider, radio control, report kind, map
    layer) and none for a Setting. The house rule (Settings, not flags) makes Settings the default
    home for any user choice.
  - The constructor sets 39 fields in **one 1,410-character line** (`modes/tty/dashboard.go:1000`).
    A new Setting adds a 40th, and its diff is unreadable. 299 non-test lines exceed 160 characters.
    After this one, the next longest are `setup_ui.go:150` (708 characters) and `map_prefs.go:731` (521).
  - The rows table uses positional literals: `{rowMapScale, groupMap, scopeObserver, rowPicker, true, "", ""}`
    (`setup_rows.go:263-290`, 55 rows). The bare `true` is the `picker` field. Its doc says "carries a
    **voice** picker" (`:220`), but 22 rows set it, including map pickers.
  - The table's doc says "it is twenty small structs" (`:238`). There are 55.
- Cost:
  - A junior will miss one of the 12 sites. The guards `TestASavedSettingIsTheSettingAfterARestart`
    and `TestEverySettingsRowIsRuledForItsSurface` catch some misses, but not, for example, a
    missing arrow branch in `setupRowKey`.
  - They will copy a positional literal whose `true` they misread as "voice picker".
- Proposed change:
  - Write extending.md Walkthrough 6, "add a Setting", listing the sites above in order, with the
    guard tests to extend. It is pinned by the existing docs test once its symbols go in
    where-things-happen.
  - Break the `NewDashboard` literal and `setupTable` rows into keyed fields, one per line. gofmt
    keeps them stable.
  - Fix the `picker` doc and the "twenty" count.
- Regression risk:
  - The literal reformat is a mechanical no-op: `go build` plus `TestReduceMotionIsASettingOffByDefault`,
    `TestTheOtherKeptSettingsSurviveARestart` (modes/tty/settings_kept_test.go) and
    `TestASettingIsSavedAsItIsWhateverSavesIt` (app/settings_kept_test.go).
  - P10-04 counts lines: a keyed `NewDashboard` grows from 48 lines to about 85. That is a rule
    question for the lead (see JR-10).
- Size: M.
- Confidence: high on the sites (grep trace), medium that 12 is complete for every Setting kind. A
  picker with choices also touches `cyclePicker` or `mapPrefArrow`.

### JR-8: Long functions here are long because of essays, not code; the worst bury a 25-line body under 90 lines of prose
- Where: in-body comment share for functions of 40+ lines:

  | Function | Location | Comment lines / total |
  |---|---|---|
  | `Broadcaster.stationLine` | `modes/tty/broadcaster.go:1142` | 90/125 (72%) |
  | `Broadcaster.lanes` | `modes/tty/broadcaster.go:768` | 59/82 |
  | `executors.speak` | `app/executors.go:403` | 61/88 |
  | `Broadcaster.clamp` | `modes/tty/broadcaster.go:681` | 51/74 |
  | `Router.Update` | `modes/tty/router.go:421` | 44/65 |
  | `executors.run` | `app/executors.go:250` | 58/98 |
  | `startSchedule` | `app/schedule.go:92` | 76/137 |

- Observed:
  - `stationLine` holds about 20 ALL-CAPS lead paragraphs and 15 ID cites (D-62, D-56, D-71, F-79,
    F-109, FR-5.5, D-153, D-107, D-160, D-70, D-80) around four `rows = append(...)` statements.
  - Its comment at `:1170` ("A NOTICE IS THE ONLY PROSE THIS ROW CARRIES…") sits above `lane := b.bandWidth()`,
    90 lines from the `statusNote` code it describes. The same point is made again at `:1254`
    ("NO STANDING PROSE… A NOTICE STILL GETS ONE").
  - In `startSchedule`, 13 of the struct-literal fields carry multi-line caps paragraphs.
- Cost: to change "which rows the station band draws", a junior has to read 125 lines to find four
  statements. Comments that have drifted from their statements, as at `:1170`, then mislead them
  about which line is being explained. Splitting these functions would not help, since the code is
  already small. Moving the rationale would.
- Proposed change, for the top seven only:
  - Move design rationale into the function's doc as a short "Why" list, keeping the IDs.
  - Leave one-line WHAT comments inline.
  - Delete the duplicated paragraph at `stationLine:1170`.

  Do not split these functions.
- Regression risk: comments only. UNPINNED. `TestWhereThingsHappenNamesRealSymbols` is unaffected.
- Size: M.
- Confidence: high on the numbers. Choosing which sentences are rationale and which are WHAT is
  judgement, and the lead should see one rewritten function before the rest.

### JR-9: Comments that contradict themselves or the code, at places a junior will copy
- Where and observed:
  - `modes/tty/dashboard.go:296` says "THREE ANSWERS, NOT TWO" and `:309`, in the same doc block,
    says "FOUR ANSWERS, NOT THREE".
  - The implementation `app/pool.go:409` still says "THREE ANSWERS, NOT TWO" and returns **unnamed**
    `(snapshot.LocationRef, bool, bool, bool)` (`:412`). The order (within, found, asked) is stated
    only in the `tty.Config` field type, 300 lines away in another package.
  - Its 43-line doc (`app/pool.go:369-411`) never names the three bools in order.
  - `modes/report/report.go:1-2` says "schema v1.0-rc"; the constant is `"1.1.0-rc"` (`platform/snapshot/types.go:23`).
  - extending.md Walkthrough 3 says `radioDeck` "runs off the update loop (`withCmd`/`takeCmd`)".
    Those are `Dashboard` methods (`modes/tty/dashboard.go:1925,1931`), not the deck's.
  - extending.md's banner claims "everything named here exists and the steps are verified". No test
    reads extending.md: `cmd/watchpost/docs_test.go` checks where-things-happen and accepted-costs only.
- Cost: a junior returning `ref, true, true, true` in a new branch has to guess which bool is which.
  A swap of `found` and `asked` compiles and inverts the operator's "no such place" message.
- Proposed change:
  - Name the results in the implementation's returned closure.
  - Reduce the two "N ANSWERS" headings to one line naming the four values.
  - Fix the schema version and the `withCmd`/`takeCmd` attribution.
  - Point the existing docs test at extending.md's backticked symbols too, as it already does for
    where-things-happen.
- Regression risk: naming results is a no-op. Pinned by `TestARealPlaceOutsideTheRadiusIsNamedButNotWithin`,
  `TestANameThatResolvesToNothingIsNotFound` and `TestAStationWithNoTransmitterReachesNothing`
  (app/locate_radius_test.go). The extending.md test is new, and may first fail on drift I did not find.
- Size: S.
- Confidence: high.

### JR-10: P10 rules are cited 648 times in comments but defined nowhere in the tree (rule question for the lead)
- Where:
  - examples of inline cites: `// bounded by … (P10-02)` (729 occurrences of P10-02 in Go), and
    `app/read_script.go:64-66` ("P10's bound of 15");
  - `06_docs/code-standards.md` (defines AP-HIST-01, AP-DEAD-01, SN-02, AP-SHELL-01, AP-OK-01, and no
    P10 rule);
  - `06_docs/p10-ledger.md` (lists rule **names** as headings only);
  - `docs/where-things-happen.md:102` ("the harness's P10 skill (outside the public tree)").
- Observed:
  - A junior cannot learn from the tree what P10-04's line limit is, what P10-05 "invariant density"
    requires, or why every loop carries a "bounded by" comment.
  - The only in-tree hint of P10-04's threshold is a backlog row (`06_docs/tooling-backlog.md:43`,
    "`funlen` at 60").
  - `make quality` (p10) needs the out-of-tree `a2dh` CLI, so a newcomer cannot even run the check.
- Cost:
  - A junior's new function or loop fails a phase-exit gate they cannot run, against a rule they
    cannot read.
  - The rule also pushes code into shapes that hurt reading, for example the one-line 39-field
    literal in JR-7 and splits made "to keep both inside the safety gate's decision ceiling"
    (`modes/tty/setup.go:335-336`).
- Proposed change:
  - Add a "P10 rules" section to code-standards.md: one paragraph per P10-nn, giving its threshold
    and the accepted forms (e.g. the "bounded by" comment, `invariant.Check`).
  - Ask the lead whether a keyed struct literal should count as one line for P10-04.
- Regression risk: docs only.
- Size: S.
- Confidence: high that the definitions are absent. Medium on exact thresholds, because the
  checker is out of tree and I could not read it.

### JR-11: Onboarding: the build section does not say how long the gate takes, what the fast loop is, or that `verify` writes a tracked file
- Where: `README.md:572-588` ("Building from source"), `Makefile:295-318`, `tools/runlog/main.go:1-6`
- Observed:
  - README packs the toolchain floor, every make target and the prerequisites into one paragraph.
  - It does not say:
    - that `make verify` takes **31-34 minutes**. The last six green runs in `06_docs/verify-runs.md`
      took 1,857 to 2,056 s;
    - that `go test ./<pkg>` is the inner loop;
    - that `modes/tty` alone runs 226 s under `-race` (`Makefile:28-34`);
    - that `make verify-docs` is the lane for a Markdown-only change. It is documented only in a
      Makefile comment;
    - that `make verify` **appends a line to the tracked `06_docs/verify-runs.md`**, so a newcomer's
      tree is dirty after verifying;
    - that a second `make verify` while one runs is **refused** (treelock), not queued.
  - There are 47 targets and no `make help`. There is no CONTRIBUTING file.
  - python3 is needed (`Makefile:546`) and README does mention it. GNU make ≥ 3.82 is explained in
    extending.md (`:203-207`), not where the build is described.
- Cost:
  - A junior runs `make verify` as their first check and waits half an hour with no expectation set.
  - They are surprised by a modified tracked file, may revert it (memory: never drop a gate
    record), or start a second run and get a refusal they do not understand.
- Proposed change: split "Building from source" into short steps:
  - prerequisites (Go floor, GNU make with the PATH line, python3);
  - the inner loop (`go test ./pkg`, `make build`);
  - before a PR (`make verify` takes about 30 min and records a line in verify-runs.md; it refuses
    an overlapping run; `make tree-free`);
  - Markdown-only (`make verify-docs`).

  Optionally add a `make help` that prints the target table.
- Regression risk: docs only. `cmd/watchpost/gates_test.go` (`TestEveryGateShapedTargetIsListedOrExempt`)
  would need to exempt a `help` target if one is added.
- Size: S.
- Confidence: high.

### JR-12: ALL-CAPS lead sentences: a useful headline device, overused to the point of losing its signal
- Where: 618 comment lines open with a 3+-word ALL-CAPS sentence (`grep -E '^\s*// [A-Z]{2,}…( [A-Z]+…){2,}[.:]'`).
  About 2,400 lines carry a 3+-word caps run anywhere. 147 open with a caps **continuation**
  (`// AND …`, `// BUT …`, `// SO …`, `// WHICH …`).
- Observed (judged fairly):
  - **Where it helps.** As a one-line rule headline followed by the reason, it is good and is what
    a junior needs. `app/rotation_script.go:20-34` is an example: "EVERY SEGMENT IS A LINE." / "A
    BLANK SEGMENT IS DROPPED." then why.
  - **Where it blocks.**
    - (a) Continuation leads that only make sense after the paragraph above:
      `app/schedule.go:199` "AND MASTERCONTROL DECLARES ON AIR / STANDBY"; `modes/tty/router.go:298`
      "AND THE SAME BUILD."; `app/pool.go:379` "AND THE OFFLINE DATA CANNOT ANSWER IT ALONE." When a
      block is moved (JR-2 shows blocks do get moved), the "AND" now points at the wrong
      predecessor.
    - (b) Density. `stationLine` has about 20 caps headlines in one function, so none stands out.
    - (c) Caps used for emphasis on individual words inside prose ("a DELIBERATE subset", "IT MAY
      REACH THE NETWORK"). A junior cannot tell which caps phrase is the contract.
- Cost: a reader skimming for "what does this do" lands on a caps headline that answers "why is
  it not something else". Contracts like "SLOW BY CONTRACT, never from a key handler"
  (`app/pool.go:389-390`) are the same visual weight as 600 other headlines.
- Proposed change (a style question for the lead, not a defect):
  - Keep caps headlines for contracts a caller must obey (blocking, locking, never-from-render,
    nil meaning).
  - Write rationale headlines in sentence case.
  - Avoid opening a paragraph with a caps "AND/BUT/SO".

  This could be a soft `tools/authoring` warning on the continuation form, which a syntax tree can
  see.
- Regression risk: comments only.
- Size: M as a sweep. S as a written guideline plus the lint.
- Confidence: medium. The counts are grep heuristics, and "helps or blocks" is my judgement on
  about 40 samples.

### JR-13: The Settings key handler is the one function where a split would genuinely help
- Where: `modes/tty/setup.go:338` `setupRowKey`: 93 lines, 30 branches, depth 3. Its `left/right`
  arm is `:377-428`.
- Observed:
  - The `left/right` arm is an **order-dependent** chain of row special cases: `rowMapsOn`, then
    `rowMapDesc`, then `mapPrefArrow`, then `rowReduceMotion`, then any `rowToggle`, then any picker,
    with `rowTheme` special inside that.
  - The flash idiom `d.setup.flash, d.setup.flashEnd = flashLeft, d.markEnds(); if forward { d.setup.flash = flashRight }`
    is written out three times (`:392`, `:404`, `:417`).
  - Row-specific cycling lives in `cyclePicker`, which sits in `setup_cast.go:164` but handles relay,
    dwell and history rows that have nothing to do with the cast.
- Cost: a junior adding a picker row must choose where in this chain to add it. A row placed after
  the generic picker branch is never reached, and nothing says so.
- Proposed change:
  - Extract `cycleFocused(forward bool) (Dashboard, bool)` holding the row-specific chain, with a
    comment stating that order matters and why.
  - Add a `flashed(forward)` helper.
  - Move `cyclePicker`'s non-cast rows beside their owners, or rename the file section.

  Other high-branch functions do **not** need splitting:
  - `marineArea` (33 branches): it is a lookup table.
  - `Dashboard.route`/`routeMap` (22/17): readable routing tables.
  - `Geocoder.search` (26).
- Regression risk: Settings keyboard behaviour. Pinned by `TestSetupArrowsWrapAround`,
  `TestAVoicePickerStepsBothWays` (modes/tty/setup_test.go), `TestThePickerBlinkIsClearedByATick`
  (setup_tick_test.go), `TestReduceMotionIsASettingOffByDefault` (motion_test.go) and the
  map_settings/map_prefs tests.
- Size: S-M.
- Confidence: medium. I did not read every map picker's `mapPrefArrow` path.

---

## Minor

- `app/schedule.go:41-50`: the doc for field `x *executors` quotes another file's comment verbatim
  ("It is the same remedy `producer()` carries … for the same stated reason: '…'"). Quoting
  comments between files means two places to keep in sync.
- One-letter struct fields that live a long time: `livePipelines.p` (`app/dashboard.go:831`),
  `radioDeck.p` (`app/radio.go:36`), `director.v` (`app/director.go:120`), `mastercontrol.v`
  (`app/mastercontrol.go:42`), `speaker.d` (`app/director.go:148`), `reader.s`/`reader.h`
  (`app/read_script.go:68-69`), `schedule.x` (`app/schedule.go:51`). There are 77 one-letter fields
  in all. Most `io.Reader` wrappers' `r` are fine, but `p`→`program` and `v`→`voice` would read
  better at use sites.
- `app/pool.go:343`: "ruling 2" is unqualified, with no file. `app/app.go:1`: "Option C" is undefined.
- `modes/tty/dashboard.go:661`: `recentOff int // scroll offset (interaction lands with tab section nav)`
  is future tense about unbuilt work.
- `Makefile`: comments narrate history, e.g. `:494-499` "THE PATTERN IS NOT ANCHORED, and that is a
  fix… `AllocBudget$$` selected seven of the eight pins". AP-HIST-01 covers Go only.
- `modes/tty/setup_rows.go:180-210`: a 20-line essay explains why `setupScope` has *no* sentinel, in
  terms of the `wires` tool's internals. It is useful to a tool author and opaque to a junior; one
  sentence plus the test name would do.
- 10 files over 500 lines have no file header, against where-things-happen's claim that "the per-file
  headers say what a file holds": `app/radio.go` (1,344), `platform/lineup/director.go` (1,164),
  `platform/httpx/cache.go` (755), `domains/radio/player/engine.go` (732), `domains/radio/synth/source.go`
  (651), `app/cast.go` (629), `platform/lineup/plan.go` (567), `domains/temperature/sources.go` (556),
  `platform/snapshot/assembler.go` (509), `modes/tty/map_prop_chart.go` (504).
- `app/setup.go:2` points at "modes/tty setupLines". It exists (`setup_layout.go:80`), but the file
  is not named.
- 59 trailing "// 0.x.y:" version stamps (e.g. `modes/tty/dashboard.go:268-271`, "0.12.0: the Alert
  Notification Preference…") say when something arrived, not what it is. That is borderline under
  AP-HIST-01.

## What is in good shape (do not spend time here)

- **Function size and nesting.** 48 of 4,282 functions exceed 60 lines, 50 nest four or more levels,
  and 37 exceed 15 branches. Nearly every long one is long because of comments (JR-8), not logic.
  Deep nesting is rare and local (`NDFD.Waves` depth 6 is a straightforward XML walk).
- **docs/where-things-happen.md.** It is the best onboarding asset in the tree. It is event-to-symbol,
  pinned by `TestWhereThingsHappenNamesRealSymbols`, and has a "Rules, and where they are stated"
  table that answers exactly the questions a junior asks. Extend it (JR-1, JR-5, JR-6); do not
  replace it.
- **docs/extending.md walkthroughs.** I spot-checked about 40 named symbols and they exist (the one
  drift is in JR-9). Walkthrough 4 (report kind) and 5 (map layer) are exemplary: closed-set
  registries with guard tests (`TestEveryReportKindReachesTheComposer`,
  `TestAFakeLayerPlugsInWithoutEditingTheOthers`).
- **Routing tables.** `Dashboard.route`/`routeMap` (`modes/tty/dashboard.go:1195-1290`) and
  `executors` are readable dispatch tables.
- **Every package has a package doc** (no package lacks one).
- **AP-HIST-01 works.** Only about 52 history-flavoured phrases remain in comments, and most are
  legitimate ("no longer wanted" as present-tense logic).
- **code-standards.md** explains each rule with a why, a yes/no example, and a self-testing checker.
- **Files whose header explains the design** before any code is read: `app/mainread.go`,
  `app/maintrack.go` and `app/mastercontrol.go`. Junior-friendly, despite the ID density.

## What I did not cover

- I did not read tests for readability (112k lines). Test names are descriptive sentences, which
  is good, but I did not survey them.
- I did not read `third_party/`, `tools/` beyond `authoring`, or the 06_docs feature folders except
  to resolve IDs.
- I did not resolve which release each of the 2,817 `D-n` cites means (JR-1). Nor did I check
  whether every cited ruling actually says what the comment claims.
- The P10 checker is out of tree, so its thresholds (JR-10) are inferred, not read.
- The "helps vs blocks" judgement on ALL-CAPS comments (JR-12) rests on about 40 samples read in
  context, not all 618.
- I did not run any make target, and I did not time `go test` myself. The gate timings come from
  `06_docs/verify-runs.md` and the Makefile's own comments.
