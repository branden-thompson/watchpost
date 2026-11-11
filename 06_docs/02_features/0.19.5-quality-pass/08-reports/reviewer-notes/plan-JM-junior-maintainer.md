# JM: the junior maintainer — plan red team, 0.19.5

Read as the future junior and as the REVIEW-time reviewer, against `implementation-plan.md` at
`feature/quality-pass` (`5363f3c9`). I walked the four first tasks through README.md,
`docs/where-things-happen.md`, `docs/extending.md` and the code. Read-only; nothing in the repository was
changed and nothing was run except grep, awk, sed and `go doc`.

## Verdict: DO NOT EXIT

No Critical findings. Nine Important ones, and every one can be fixed by editing the plan text, with no
re-ordering of the work packages. The plan carries out JR-1 to JR-13 one finding at a time. Three gaps
would leave a junior about where they are today: it never says what a junior must be able to do at the
end (M5 has no pass criteria), its new checks are under-specified in ways that check nothing or block the
wrong edits, and it never makes the junior's map (where-things-happen and extending.md) move with the
Q3 to Q6 code.

## Findings

### JM-1 [Important]: The release-qualified citation rule (Q7.6) fires on the old citations that Q8 and Q9 move, and the index it needs lands after it
- Plan task(s): Q7.6, Q8.6, Q9.4, Q9.8, Q7.1, Q8.1.
- Observed: Q7.6 checks "release-qualified citations (D-11) on added comment lines" and lands in Q7. The
  index it depends on (`06_docs/ids.md`) does not land until Q8.6. After Q7, Q9.4 moves the essays of seven
  functions into their docs, "keeping the IDs". `Broadcaster.stationLine` alone carries 15 bare cites
  (D-62, D-56, D-71, F-79 …, JR-8). Q9.8 rewrites caps headlines, and Q7.1 and Q8.1 move or restate about
  55 docs. In a diff, every moved or rewritten line counts as added. `tools/authoring` reads files, not
  diffs (`tools/authoring/rules.go`), and the plan names no base for "added" (branch point? `origin/main`?
  `HEAD~`?).
- Consequence: D-11 says "the old citations stay, and the index resolves them". The rule as written
  forces every moved old cite to be qualified, which means resolving which release a bare D-15 means
  (four meanings, JR-1). JR-1 sized that work L and the ruling declined it. Otherwise the rule gets
  bypassed in the batch meant to be held by it. A junior who fixes a typo on a line that holds "D-124" gets
  the same refusal and must resolve 0.18.0 against 0.19.0 with no guidance. With an unnamed base, the
  local result and CI can disagree (a shallow CI clone may have no merge-base).
- Proposed fix to the plan: Define "new citation" as an ID that is not already present in the same file
  at the base (or in the removed lines of the hunk), and name the base: the merge-base with `main`, the
  same one `scripts/quality/p10-unmatched.sh` uses. Land `ids.md` (Q8.6) with or before Q7.6. Make the
  message name `ids.md` and say how to choose among candidates.

### JM-2 [Important]: The pass mints IDs that already collide in the tree, and the plan never says which ID families D-11's rule and index cover
- Plan task(s): Q7.6, Q8.6, A-11, every task that writes a comment.
- Observed: The plan's work packages are Q1 to Q9, and BUILD will cite PF-n, DR-n, RN-n, DC-n and JR-n.
  The tree already holds 210 `Q`-style cites ("Quality pass Q2 …", `app/declset_test.go:3`,
  `app/themes.go:3`), and `where-things-happen.md:125` defines "Qn | a quality-pass batch" as 0.9.5's. It
  holds 59 `PF-n` cites (`app/maps.go:145` "PF-4" is not 0.19.5's PF-4, which is the console's clamp), and
  `where-things-happen.md:127` lists PF as a 0.9.5 red-team prefix. It holds 276 `DR-n` cites. Q7.6 says
  only "(D-11)", and Q8.6 does not list its families. JR-1 counted about 45 families.
- Consequence: 0.19.5 comments written in Q1 to Q6, before Q7.6 exists, re-create the JR-1 defect with
  the pass's own IDs. A junior who sees "quality pass Q3" or "PF-4" cannot tell 0.9.5 from 0.19.5. M5's
  "ruling IDs can be resolved" has no defined scope, so the reviewer cannot grade it.
- Proposed fix to the plan: List the families Q7.6 checks and Q8.6 indexes: at least D, FR, F, Q, PF, DR,
  RN, DC, JR, plus 0.19.5's own lens prefixes. Require 0.19.5 comments from Q1 onward to cite
  "0.19.5 PF-7" and similar. Q7.6 then sweeps any bare cite added since the branch point.

### JM-3 [Important]: Q7.4's extending.md symbol check, built "like where-things-happen's", would match nothing on the page
- Plan task(s): Q7.4, Q9.3, M4.
- Observed: The where-things-happen check uses ``symbolRef = "`([a-z0-9_/]+(?:\.go)?):([A-Za-z_]…)`"``
  (`cmd/watchpost/docs_test.go:23`) and `declares` accepts only functions and methods. Run over
  `docs/extending.md` it matches **0** of the page's 256 backticked spans. The page names bare symbols
  (`detailLines`, `firms.CheckKey`, `livePipelines.markFIRMS`), types and fields (`snapshot.Provider`,
  `Conditions.UVIndex`, `mapInputs`) and constants (`numKinds`, `KindFire`). The existing test's floor
  (`len(refs) < 30`) would fail at once, or would be dropped for this page.
- Consequence: Two outcomes are likely, and both are bad. The check passes while checking nothing, which
  makes M4's "symbol check over docs/extending.md" and the page's "verified" banner untrue. Or BUILD
  discovers mid-Q7 that all five walkthroughs must be rewritten into `file.go:Func` form, a batch the plan
  does not budget. Walkthrough 6 (Q9.3), the walkthrough a junior most needs, is written after the check,
  in an unspecified form.
- Proposed fix to the plan: State the check's form. Either it resolves bare `pkg.Symbol` and `Symbol`
  spans against the files named in the same table row, with types, constants and fields declared, or the
  walkthroughs are rewritten to `file.go:Symbol` form inside Q7.4. Keep a floor and a "the matcher
  narrowed" self-check, as the existing test has.

### JM-4 [Important]: Q6.4's helper signatures do not match the code, collide with an existing type, and leave the other unreadable Setting lines alone
- Plan task(s): Q6.4, Q9.3 (D-14).
- Observed:
  - `mapPrefsFrom(cfg Config) mapPane`: `mapPane` (`modes/tty/map_pane.go:63`) is the runtime map holder,
    with the library map, the drawn lines, counters and the Propagation session. The map preferences are
    about 20 flat `Dashboard` fields (`mapsOff`, `mapDesc`, `mapRadarIEM` …, `dashboard.go:699-721`) with
    about 250 `d.` uses in `modes/tty`. Returning them from a helper means either moving them into
    `mapPane` (every `d.mapDesc` becomes `d.mapPane.mapDesc`) or embedding new struct types, and the plan
    names neither.
  - `uiPrefsFrom(cfg Config) uiPrefs` would sit beside the exported `UIPrefs` (`dashboard.go:451`, the save
    shape that `uiForSave` builds). Walkthrough 6 would then tell a junior to "add the field to `UIPrefs`
    and to `uiPrefs`".
  - Every UI or map Setting also touches `uiForSave` (`setup_ui.go:150`, 708 characters) and
    `mapPickerRow` (`map_prefs.go:607`, 426 characters). Q6.4 keys only `NewDashboard` and `setupTable`.
  - "setupTable's rows become keyed too" gives no form. The function is already 103 lines
    (`setup_rows.go:239-341`), with 55 rows of 7 fields. One field per line makes it about 500 lines and
    destroys the at-a-glance focus order the table exists to show.
- Consequence: BUILD has to improvise a structural change to 39 fields (D-14 said helpers, not a field
  regrouping), or ship signatures that lie. The "add a Setting" diff still edits a 708-character line, so
  the readability gain that D-14 bought is half delivered.
- Proposed fix to the plan: Give the real shape: which struct each group's fields live in (embedded or
  named), and the access-path change it implies. Choose group type names that do not shadow `UIPrefs`.
  Add `uiForSave` and `mapPickerRow` to Q6.4. Specify `setupTable` as one keyed row per line with
  zero-valued fields omitted.

### JM-5 [Important]: The D-13 rename covers the type and the file, but not the names and records a junior actually reads
- Plan task(s): Q6.3 (D-13), Q9.2.
- Observed: Q6.3 renames `app/director.go` to `arbiter.go` and the type `director` to `arbiter`, with
  tests, docs, where-things-happen and the 41 mutants. It does not cover:
  - the fields and locals that hold the arbiter, still named `director`: `lp.director` (22 uses,
    `app/dashboard.go:290,301,305,335`, `app/pool.go:89-99,488-491`; field at `app/dashboard.go:849`);
  - lower-case prose that means the arbiter (`app/radio.go:895` "The radioDeck is the director's voice";
    `domains/radio/player/retention_test.go:129`);
  - the ambiguous `platform/lineup/projection.go:15` ("director.go queues …");
  - the 0.16.0 REQUIRED-READING (the settled design, brief item 7), which says three times "The word means
    today's `app/director.go`";
  - the P10 ledger row `app/director.go | awaitAir` (`06_docs/p10-ledger.md:64`), ratified by the HUM LEAD
    and keyed by file. It was re-keyed once before, at the 0.14.0 rename.
- Consequence: After the rename, `lp.director.Run(...)` still reads as "the Director". That is the exact
  confusion D-13 exists to remove (JR-5: "a junior asked to fix the Director's ducking can open
  `app/director.go`"). The required reading points a junior at a file that no longer exists, and the P10
  phase-exit gate reports `arbiter.go:awaitAir` as unratified.
- Proposed fix to the plan: Extend Q6.3 to fields, locals and every lower-case "director" that means the
  arbiter, each sorted by hand. Add a one-line dated note under the 0.16.0 required reading's arbiter
  paragraph saying the file is now `app/arbiter.go`, leaving the verbatim quotes untouched. Re-key the
  ledger row with the HUM LEAD, citing the rename, as at 0.14.0.

### JM-6 [Important]: The junior's map is not in the per-task "record moves with the code" step, so Q3 to Q6 leave it describing the old code
- Plan task(s): the every-task order step 6, Q3.1, Q4.1 to Q4.4, Q5.1 to Q5.5, Q6.1, Q6.2.
- Observed: Step 6 lists "the build log entry, the as-built map and its diagrams, and the atlas". It does
  not list `docs/where-things-happen.md`, `docs/extending.md` or the vocabulary. Only Q6.3 names
  where-things-happen. The batches that move responsibility:
  - Q3.1 gives `Dashboard` an `Update`/`update` pair, as `Router` already has (`router.go:421`, `:489`).
  - Q4.2 to Q4.4 add a masthead memo, a tape memo and a content generation in a different package. The
    "A frame is built" row names only `memo.go:tables`, and the vocabulary defines only the body and modal
    memos.
  - Q4.1 adds the rule "Apply replaces, never writes into", and Q4.3 the contract "the ticker changes only
    through `setTicker`".
  - Q6.1 and Q6.2 move console card rows and the top rule into `platform/render`.
  - Q5 creates new single owners (`FeltBand`, `Compass16`, `OnEarth`, `HostOf`, `ClampRunes`).
  `TestWhereThingsHappenNamesRealSymbols` only proves that named symbols exist. It passes while a row
  describes a responsibility that moved.
- Consequence: Task 3 ("change a screen") on a console card now starts in `modes/tty` and ends in
  `platform/render` with no row saying so. A junior adding a fourth compass table or felt-band switch
  cannot find the owner. All of this lands in Q9 or never, and Q9 is last.
- Proposed fix to the plan: Add to step 6: "where-things-happen rows, the vocabulary and extending.md
  steps for any responsibility the batch moves or creates; any new rule a caller must obey goes in the
  'Rules, and where they are stated' table". Name the rows for Q3.1, Q4.1 to Q4.4, Q5 and Q6.1 to Q6.2.

### JM-7 [Important]: The Q4 memo guards are specified "over the key", which cannot catch the junior's likely mistake, and the generation contracts are unstated
- Plan task(s): Q4.2, Q4.3, Q4.4 (D-4).
- Observed: D-4 and Q4 say "a completeness guard over its key". The pattern the plan cites,
  `memo_completeness_test.go:37`, guards **inputs**, not the key: it perturbs every model field by
  reflection and fails when "the frame changes and NOT the key", naming the field and F-30. A guard over
  the key's fields cannot see a new input left out of the key. That is exactly what a junior does when
  they add a badge to the masthead or a column to the table (task 3).
  - PF-9's "drawn data", the set of fields whose change bumps the generation, lives in
    `platform/snapshot` while what is drawn lives in `platform/render/table.go:LocationTable` and
    `modes/tty`: a cross-package coupling.
  - PF-8's generation silently freezes the tape if the ticker is written in place, and an existing test
    already does that (`tick_alloc_test.go:59`, `d.ticker[0].Test = true`).
  - The "cache table" the memos are listed in is quality-baseline §4, inside 0.9.5's feature folder,
    which no junior doc links.
- Consequence: A junior adds a column, the table memo keys on a generation that the new field does not
  bump, and the row goes stale: the issue #11 and #12 class the plan names as the danger. Nothing reads
  "add your field to X in assembler.go".
- Proposed fix to the plan: Require every Q4 guard to perturb the inputs the frame reads (snapshot
  `Location` fields for PF-9, `Dashboard` and `Opts` for PF-7 and PF-8), with a failure that names the
  memo, the field and the function to edit. State `setTicker`-only and "Apply decides drawn data" as
  contract comments (D-12's own case), and add them as rows in the Rules table. Link the cache table from
  where-things-happen, or move it under `docs/`.

### JM-8 [Important]: Walkthrough 6 is not checked against a real Setting, and the one miss with no guard stays unguarded
- Plan task(s): Q9.3, Q9.7, Q2.
- Observed: JR-7 traced about 12 sites for one boolean (`reduce_motion`) and was "medium that 12 is
  complete for every Setting kind": a picker also touches `cyclePicker` or `mapPrefArrow`. It also found
  that no guard catches a missing arrow branch in `setupRowKey`. The existing guards are
  `TestEverySettingsRowIsRuledForItsSurface`, `TestEverySurfaceOffersEverySetting` and the
  reflection-based saved-setting tests (`app/settings_kept_test.go:19`, which fails clearly per field).
  Q9.7 adds only "a comment saying the order matters". The rule "Settings, not flags" is written nowhere
  in `docs/` or README.
- Consequence: The walkthrough a junior most needs can be written from the reduce-motion trace and still
  be wrong for a picker or a map Setting. The arrow branch, which a junior can place after the generic
  picker branch where it is never reached (JR-13), fails silently. The M5 reviewer finds the gap at REVIEW.
- Proposed fix to the plan: In Q9.3, cover the three kinds (toggle, picker, map preference). Prove the
  walkthrough by adding a throwaway Setting of each kind on a scratch copy and recording, per step, which
  guard fails when the step is skipped. Open the walkthrough with the "Settings, not flags" rule. Add one
  test-only guard to Q2: every `rowToggle` and `rowPicker` row changes something on left and right.

### JM-9 [Important]: M5's REVIEW verdict has no pass criteria
- Plan task(s): After BUILD, REVIEW; the brief's M5.
- Observed: The brief and the plan both say only that "a fresh blind reviewer with a junior persona
  traces the four first tasks from the docs alone". Nothing states:
  - the persona;
  - which documents count as "the docs";
  - what the reviewer produces;
  - what counts as a failure, or what happens after one.
  M5's checklist half is also partly ungradeable: "current" front door, "the terms JR-6 lists" (Q9.2's
  list drops JR-6's "settle"), and which ID families must resolve (JM-2).
- Consequence: Any reviewer report can be read as a pass or a fail. Gaps found at REVIEW have no rule
  for whether they block SHIP.
- Proposed fix to the plan: Add these pass criteria to the plan.
  - **Persona.** Knows Go and has read the Bubble Tea README. Does not know the harness, the ID schemes
    or the record.
  - **Inputs.** README's "Start here" and everything it links: `docs/*.md`, `06_docs/code-standards.md`,
    `06_docs/ids.md`, `go doc` of every package, and the files the docs name. Not the 0.19.5 feature
    folder, not the plan or lenses, no repository-wide grep.
  - **Output per task.** The ordered files and symbols to edit, the tests to extend, how to check the
    change (commands), and every blocker quoted with the missing sentence.
  - **Answer key.** Built by the agent before REVIEW by doing the four tasks on a scratch copy:
    1. a toggle Setting;
    2. a keyless provider;
    3. a line on a console card;
    4. a spoken phrase change.
    The key records every site and which guard fails when each site is missed. It is hidden from the
    reviewer.
  - **Pass**, for each task: every site in the key is reached from the docs, or its omission fails a
    guard whose message names the site. No doc sentence routes to a wrong or missing file or symbol (for
    example, the Director against the arbiter). Every ID met on the path resolves through `ids.md`.
  - **Fail.** A fail in any task is fixed before SHIP, or ruled as a D-row. It is not filed as a
    follow-up.

### JM-10 [Minor]: Q9.1's doc.go will be concatenated with the stale fragments it replaces, and nothing keeps it current
- Plan task(s): Q9.1.
- Observed: `go doc ./app` joins three package-clause comments (`app/app.go:1`, `app/dashboard.go:1`,
  `app/setup.go:1`). `platform/render` has two. `modes/tty/dashboard.go:1-10` holds the stale "B3 build of
  mock rev2" doc. Q9.1 adds `doc.go` files but does not say the fragments become file headers, as JR-4
  proposed. Its file-group tables are the kind of map that went stale last time.
- Consequence: The new front door prints after or before "Option C … B3 … M1 instrumentation …", and in
  a few releases its table describes files that have moved.
- Proposed fix to the plan: Detach the fragments (a blank line before `package`). Add an authoring rule:
  one package doc per package. Add a test that every non-test file in `modes/tty` and `app` matches a
  prefix row in its `doc.go` table.

### JM-11 [Minor]: D-12's "touched code" is undefined, its rule text lands after its check, and the files on task 4's path stay as they are
- Plan task(s): Q7.6, Q9.8, Q9.9 (D-12).
- Observed:
  - **Granularity.** "Code that Q1 to Q9 touched" could mean the touched line, the function or the file.
  - **The densest files.** The heaviest caps-lead files are `modes/tty/broadcaster.go` (41 leads, 18
    AND/BUT/SO), `modes/tty/router.go` (38/17), `app/pool.go` (22/8) and `platform/lineup/director.go`
    (16/5). The first three are touched by Q1, Q3, Q6 and Q9.4. `platform/lineup/director.go`, `bed.go` and
    `card.go`, the code task 4 runs through, are touched only if Q9.9 adds a file header to
    `lineup/director.go` (it is on JR's list of headerless files), and whether that counts depends on the
    undefined granularity.
  - **Order.** The soft warning lands in Q7.6, but the code-standards rule it enforces lands in Q9.8.
  - **Noise.** Over the whole tree the soft warning has 147 standing hits.
- Consequence: The REVIEW reviewer cannot check Q9.8. A warning that cites an unwritten rule, then fires
  147 times on every run, teaches people to ignore it.
- Proposed fix to the plan: Define "touched" as the function or declaration the batch changed, plus the
  whole file for files Q9.4 or Q9.9 rewrite. Land the code-standards text with the check in Q7.6. Report
  the soft warning on changed declarations only.

### JM-12 [Minor]: A-2's P10 section must read its thresholds and counting rules from the checker, and say who runs it
- Plan task(s): Q9.8 (A-2).
- Observed: The only in-tree threshold is "`funlen` at 60" (`06_docs/tooling-backlog.md:43`), yet
  `setupTable` is 103 lines (`setup_rows.go:239-341`) with no P10-04 ledger row. D-14's premise was that
  one field per line breaks P10-04 at about 85 lines. `make quality` needs the out-of-tree `a2dh`.
- Consequence: A section that says "60 lines" beside a 103-line function that passes confuses the reader
  it is written for. A junior's PR can pass CI and `make verify` and then fail `p10` at a phase exit they
  cannot run.
- Proposed fix to the plan: Take each threshold and what it counts (lines, statements, literals) from the
  checker, checked against `setupTable` as a specimen. State plainly: "you cannot run this; the maintainer
  runs it at phase exit; stay inside these limits or use these accepted forms".

### JM-13 [Minor]: A third `architecture.md` arrives while 32 citations name "architecture §n" with no path
- Plan task(s): Q8.5 (A-3), Q7.4, Q9.2.
- Observed: Two `architecture.md` files exist (watchpost-cli and propagation-overlays). A-3 adds
  `docs/architecture.md`, which "Start here" will link. 29 Go comments and 3 doc lines cite
  "architecture §n" without a path. Among them are extending.md:154-155 ("say in `architecture.md §1.1`
  why it is not hazard data"), an instruction on the data-source path (task 2), and :67 ("architecture
  §10.3").
- Consequence: A junior opens the linked new file and finds no §1.1 or §10.3.
- Proposed fix to the plan: In Q8.5, re-point every unpathed "architecture §n" to its full path (or to
  the new doc where it now holds the content), and have Q7.4's check refuse an unpathed `architecture.md`.

### JM-14 [Minor]: Task 3 meets goldens and declsets with no doc, and Q2.8 adds about 24 more
- Plan task(s): Q2.8, Q9.3, Q9.6.
- Observed: Re-capture flags differ by package (`-update-golden`, `-update-declset`, `-update-plan`,
  `-update-cadences`), and none is in README or `docs/`. The frame golden's failure prints the whole
  frame and says "say why in the build log" (`modes/tty/golden_test.go:34`), a harness term. The declset
  failure says "a pure move must not" (`app/declset_test.go:33`) and gives the re-capture command only when
  no golden exists. Walkthrough 5 step 5 adds a top-level function in `app` and so meets that message.
  Q2.8 adds goldens for six surfaces at two sizes, plain and ASCII.
- Consequence: Every screen change, the third first task, fails a dozen goldens with a wall of escapes
  and no doc on how to re-capture or what to check.
- Proposed fix to the plan: Add a "When a golden or declset fails" paragraph to extending.md (Q9.3 or
  Q9.6). Make Q2.8's goldens report the first differing line and its number. Make the declset message
  name `-update-declset` for an intentional addition.

### JM-15 [Minor]: Smaller junior gaps the plan should name
- Q9.2: role rows need a code owner in the "Defined in" column, or task 4 cannot be traced from the
  vocabulary. The code is spread across `app/pool.go:producer`, `platform/lineup/director.go:stepProducer`,
  `app/burst_words.go`, `domains/radio/synth/compose.go`, `app/read_script.go:reader` and
  `app/mastercontrol.go`. Add "settle", which JR-6 lists and the arbiter shares with the Director.
- Walkthrough 2 (task 2) omits `app/endpoints.go:providerEndpoints`, whose guard
  `TestEveryProviderHasAnEndpoint` fails a new provider. DC-14 does not list it.
- Q7 (all checks): each new check uses the authoring tool's Rule/Text/Why output and has a section in
  code-standards "Running the checks" with its command. The go-studs refusal names the route ("add a
  render function"). The dupes stale-row failure names the row to delete.
- Q5.1: `FeltBand` returns a bare int that three tables index. Document what each value means, and pin
  each consumer's table length to the band count.
- Q3.1: the new `Dashboard.Update`/`update` pair gets docs saying which holds the logic, and
  where-things-happen's "A key is pressed" row follows it.

## What the plan gets right
- It takes JR's headline seriously: navigation, not complexity. It does not split the essay-bodied
  functions (Q9.4 moves rationale only), and it shows the first rewrite to the HUM LEAD before the other
  six.
- The checks land before the docs and readability passes (Q7 before Q8 and Q9), so the gains are held.
  SN-02 over types, values and fields is the right fix for the fused docs a junior meets in `go doc`.
- The test names are sentences a junior can read as a specification
  (`TestBackspaceErasesALetterNotAByte`, `TestTheRadiosPlannedHeightIsItsRows`).
- Q9.6's build section sets the 30-minute expectation, the tracked-file side effect and the treelock
  refusal, which are the three surprises JR-11 found.
- Q9.7 extracts `setupRowKey`'s chain, the one split JR said would help. Consolidation picks owners in
  natural homes (`geo.Compass16` beside `CompassIndex`).

## What I did not check
- The RN, PF and DC claims in the plan beyond what touches readability. Whether PF-9's generation can be
  made complete at all.
- The out-of-tree P10 checker's thresholds (JM-12 rests on the ledger and `setupTable`'s length).
- Every step of Walkthroughs 1, 3, 4 and 5 against the code. I spot-checked Walkthrough 2's symbols
  (they exist) and the host guards.
- The caps-lead counts use JR-12's grep heuristic, and the ID-family counts are grep over non-third-party
  Go.
