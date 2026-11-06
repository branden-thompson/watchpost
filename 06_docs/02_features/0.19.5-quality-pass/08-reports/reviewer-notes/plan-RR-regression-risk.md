# Regression risk — plan red team, 0.19.5

Reviewer: RR lens (every change pinned first; no output change without a ruling). Tree: `feature/quality-pass`
at `5363f3c9`, read-only. Evidence: one `go test -count=1 -coverprofile` run of `./modes/tty` (green, 93.5 %),
a parse of all 900 mutant anchors, and reading the code at each cited line. Scratch: `rr/` beside this file.

## Verdict: DO NOT EXIT

No finding is Critical as written. But four tasks cannot be built as specified, or would land unpinned:
Q2.1's seam, Q2.10's "tests only" work, Q6.4's builders and Q4.4's generation. Five more name a pin that
does not cover the code they change. Each costs a batch or a revert if found in BUILD. All are plan edits
of a sentence or two.

## Findings

### RR-1 [Important]: Q1.2 changes the console's clock with no ruling, and its units never reach the console's memo key
- Plan task(s): Q1.2 (DEF-2, D-5).
- Observed:
  - D-5 rules one defect: the console's `render.Opts` "sets no Units". Q1.2 also carries **Clock**.
  - The console's `opts()` (`modes/tty/broadcaster.go:577-579`) sets only Width, ASCII and Frame, so it
    draws `Clock12`, the zero value (`platform/render/clock.go:26`).
  - The console masthead goes through that `o`: `broadcaster_header.go:28` calls `mastheadStamp`, which
    writes `o.Clock.Stamp` (`masthead.go:53-54`). So for every 24-hour or military listener, "Updated: 8:00 PM"
    becomes "Updated: 20:00". The pin (`TestTheConsolesPoolWearsTheListenersUnits`) checks units only.
  - Separately, the console memoises the pool and running-order lines (`consoleMemo.sched, pool`,
    `broadcaster_memo.go:63-68`). Its key `consoleKey` (`:33-62`) holds no units and no clock.
  - Units are shared Settings that can be changed from the console (0.16.0 FR-6.2, D-92 forwarding).
- Consequence:
  - The clock change is an output change with no D-row (R1, M1).
  - Once units reach the console's `opts`, a units change made while the console is up leaves the pool
    in the old units until the next publish or selection change. That is the F-30 stale-frame class.
  - `broadcaster_memo_completeness_test.go` catches this only if units become a Broadcaster field its walk
    perturbs. Read through a shared config pointer, nothing catches it.
- Proposed fix to the plan: put the clock to the HUM LEAD as its own ruling (or as an explicit D-5 rider)
  before Q1.2. State that units and clock are mirrored into Broadcaster fields on every `Router.Update`, as
  the gain is (`router.go:452`), and are added to `consoleKey`. Add a pin that changes units with the
  console up and asserts the pool redraws.

### RR-2 [Important]: Q2.1's seam does not compile as listed, breaks three test files, and is unnecessary
- Plan task(s): Q2.1 (RN-1).
- Observed:
  - The plan's interface names 12 engine methods. The app calls 19. The seven missing are `Audition`
    (`app/voices.go:203`), `Fail` (`radio.go:681`, `:696`, `mainread.go:166`, `:185`), `StartSource`
    (`radio.go:712`, `mainread.go:210`), `Samples` (`radio.go:170`), `OnSilence`, `OnClipSpent` and `Trace`
    (`radio.go:152-154`).
  - RN-1's premise is that the engine "cannot be faked, so it is never run" and is "nil in every test".
    That is false. Three tests build a real `*player.Engine` over a fake `player.Output`:
    - `offlineDeck` (`app/director_pins_test.go:66-93`)
    - `liveDeck` (`app/mainread_live_test.go:72-82`)
    - `TestAUserStopDoesNotTakeAnAlertOffTheAir` (`app/radio_stop_test.go:62-70`)

    They call `d.engine.StartSource` and `d.engine.Restore` directly (`director_pins_test.go:186`, `:196`).
  - Preview and PreviewAside differ observably on a real engine: the tap flag (`player/engine.go:310`,
    `:320`) feeds `Samples`. The relay-fault chain starts in `reportSilence` (`engine.go:112`), which a
    recording fake bypasses. Of the plan's 12 methods, none is `OnSilence`, so the fake could not drive
    that chain at all.
- Consequence:
  - As specified, the seam either fails to compile or leaves two engine handles, so `d.engine != nil`
    (`radio.go:292`, `:887`, `cast.go:399`, `dashboard.go:707`) means different things on different paths.
  - Either way, the three existing pins must be edited in the same commit as the production change they
    guard, and that commit lands before any pin covers the deck (rule 1, R2).
- Proposed fix to the plan: drop the seam. Write RN-1's pins on a real engine over a recording `Output`,
  extending `heldOutput`/`drainOutput`. Assert the tap through `Samples`, and silence through the engine's
  own `reportSilence` path. Q2 then really is tests only.

### RR-3 [Important]: Q2 is not "tests only": three production edits hide in it, two of them on timer paths
- Plan task(s): Q2.10, Q2.12, Q2.7.
- Observed:
  - **Q2.10, the publisher.** "The launch-fill test takes `platform/sched`'s fake clock." The publisher
    uses `time.AfterFunc` and `time.Now()` directly (`app/pipelines.go:64-90`). `sched.Clock` has only
    `Now` and `After` (`platform/sched/sched.go:19-31`).
    - So the test needs a production change: either `AfterFunc` moves onto a goroutine reading `After`, or
      `sched.Clock` grows a method.
    - F-51's own row (`06_docs/follow-ups.md:167`) says a clock change here "deserves its own review", and
      F-51 was fixed *without* one.
  - **Q2.10, `propMsgs`.** It "classifies the command by type". A `tea.Cmd` is an opaque `func() tea.Msg`.
    The tick is `tea.Tick(every, …)` (`map_prop_update.go:206`), whose closure has no type to inspect.
    Classifying it means a production seam (a named tick command or an injectable tick).
  - **Q2.12, the history sweep.** Pinning "the history sweep's clock" means changing
    `platform/history/prune.go:357-363` from `time.Since` to `s.now()`, as RN's own minor list says.
  - **Q2.7.** Deriving the note's "10" from `WatchCap` (`modal_location.go:300-303`) is a production edit
    too. It is harmless, but it is not tests only.
- Consequence: Q2 is planned as risk-free and runs before everything. These edits would land in the batch
  with the least scrutiny: timer and goroutine changes in the publisher and history store, guarded only by
  the timing tests being rewritten in the same commit.
- Proposed fix to the plan: list each as a declared seam with its own pin first. For the publisher, that
  is F-51's `awaitRuns`-style failure-bound test of "the first publish is immediate, a wave folds" before
  the clock change, then `-race -count=20` on `app`. Keep "tests only" for what is.

### RR-4 [Important]: Q1.1 rewrites ten erase sites; six are run by no test, and Q1 lands before Q2
- Plan task(s): Q1.1 (DEF-1, RN-4).
- Observed:
  - The coverage profile of `./modes/tty` shows these erase blocks are never run:
    - `setup.go:800-803` (place query), `:831-834` (service radius) and `:856-858` (FIRMS key)
    - `map_prop_find.go:101-103` and `map_prop_freq.go:251-253`
    - `router.go:1167-1170` (Change Position)
    - `request.go:446-457` (both fields)
  - Only the Lookup window (`modal_location_test.go:29`) and `setup.go:880` (`setup_esc_save_test.go:163`)
    press backspace.
  - Several sites carry a side effect beside the erase, which a helper extraction can drop:
    `d.setup.serviceSeeded = false` (`:830`), `d.setupSuggest()` (`:803`), `r.observer.cardErr = ""` (`:1170`).
  - Q1.1's pins cover the Request window only.
- Consequence: seven sites change behind no pin, in the first batch, before Q2's net exists. A dropped
  `cardErr` reset leaves a stale error line in Change Position, and nothing fails.
- Proposed fix to the plan: Q1.1's pin becomes one table test that presses backspace on "Peñ" in every
  text field, asserting the field and each sibling side effect, seen to fail on a mutant of one site.

### RR-5 [Important]: Q6.4's builder signatures do not match the model, and the task names no pin
- Plan task(s): Q6.4 (D-14, JR-7).
- Observed:
  - `mapPrefsFrom(cfg Config) mapPane` returns `mapPane`, but the map preferences NewDashboard sets are
    top-level `Dashboard` fields: `mapsOff`, `mapDesc`, `mapRadarIEM`, `mapTempNDFD`, `mapRainFull`,
    `mapUVCities`, `mapProp*` (`modes/tty/dashboard.go:1000`). `uiPrefs` and `radioPrefs` do not exist
    (only `tty.UIPrefs`, a Config type, at `:451`).
  - Followed literally, Q6.4 moves about 25 fields into nested structs. That touches every access site.
  - Two things key on the field names:
    - The memo guard walks fields by reflection, one level deep, with name-keyed exceptions
      (`dashboardWriter` matches `"mapTempNDFD"`, `"mapRainFull"`, `memo_completeness_test.go:109-113`).
    - Ten mutants anchor on these fields (`m613`-`m622`, e.g. `.reduceMotion`).
  - The task lists no pin. JR-7's own pins check three Settings, not the 39-field mapping.
- Consequence: a field dropped while splitting the 1,410-character literal into three helpers is a Setting
  silently ignored at start-up, and no test listed catches it. If the fields do move, the guard's
  name-keyed exceptions and ten anchors break at once.
- Proposed fix to the plan:
  - Fix the signatures. Either each helper writes into `*Dashboard` and returns nothing, or it returns a
    struct that is embedded, and the plan says which.
  - Pin first with a reflection test: perturb each `Config` field the model reads, and assert
    `NewDashboard` reflects it, in the style of the memo guard.

### RR-6 [Important]: Q4.4's content generation is bumped only in `Apply`, but other assembler calls change the rows too
- Plan task(s): Q4.4 (PF-9, D-4's stale-row guard).
- Observed:
  - "The assembler keeps a content generation that `Apply` bumps." But `SetLocations`
    (`platform/snapshot/assembler.go:194`) adds, removes and reorders locations and carries their Label
    and Tag. `SetInactive` (`:177`) turns a provider off. Neither goes through `Apply`.
  - Today the table memo keys on the snapshot pointer (`memo.go:42`), so these reach the frame on the next
    publish. Keyed on a generation that ignores them, they would not.
  - The guard D-4 requires is "over its key", but the plan does not say which struct the reflection
    covers. The generation is computed in the assembler, not from a struct a reflection walk can perturb.
  - The plan does not say where the generation travels. `watchpost report` marshals the whole `Snapshot`
    (`modes/report/report.go:30`), and no test pins its JSON key set. An exported field changes the
    report's output silently.
- Consequence: removing a watched place, or editing its tag, leaves the old row drawn until some other
  location's data changes. That is issue #11 / #12 again, the case D-4 named.
- Proposed fix to the plan:
  - Every `*Assembler` mutator bumps the generation.
  - The guard derives the mutator set by reflection over the method set, as the modal guard derives
    windows from the enum, and asserts "table differs ⇒ generation differs".
  - The generation is unexported or `json:"-"`, with a pin on the report's top-level keys.

### RR-7 [Important]: Q5.3's single-row pin cannot hold "no spoken output changes" (A-4)
- Plan task(s): Q5.3 (DR-6, A-4).
- Observed: `shortDur` (`domains/severe/record.go:95-107`) has four behaviours a duration-based rewording
  must copy:
  - It rounds to the minute.
  - It **truncates days**: 1 day 12 hours reads "1d", so it is spoken "1 day".
  - It measures from the onset when that is later than the issue time (`:83-86`).
  - It writes no window at all when the expiry is not after that start (`:87`). `spokenWindow` then
    returns "" (`app/severe_read.go:334-338`) and the line is dropped.

  The plan's pin feeds one real row.
- Consequence: a natural duration worder says "1 day 12 hours" where the radio says "1 day" today, and
  speaks a "0 minutes" or negative window where today's read drops the line. Each is a spoken change
  without a ruling, and the one-row pin passes.
- Proposed fix to the plan: pin a table through the read before the change: no window, 29 s, 30 s,
  59 m 30 s, 1 d 12 h, and an onset after the issue time.

### RR-8 [Important]: Q3.4's equivalence pin never reaches clamp's height backstop, and Q3 names no pin for three tasks
- Plan task(s): Q3.4 (PF-4), Q3.1, Q3.3 step 1, Q3.6.
- Observed:
  - Clamp's truncation backstop (D-102, `broadcaster.go:682-684`, `:708-709`) is never run by any test
    (coverage). The pin's four sizes (100x44 to 220x60) all fit their lanes, so a one-pass rewrite that
    drops "cut to the viewport" passes.
  - The pin runs in one theme. `mAC9` anchors on the exact line PF-4 deletes
    (`render.TintKeeping(out, render.FgSGR(…))`, killed by `TestTheConsoleArmsTheThemesForeground`), and
    must be re-pointed to the new base computation.
  - Q3.1 (both steps), Q3.3 step 1 and Q3.6 name no pin. PF-1's step 2 (`Router.update` returns `Router`)
    was never prototyped. `m642` anchors on the `OverlayCursor` line Q3.6 rewrites (`router.go:529`).
- Consequence: the backstop that turns a mis-budgeted region into a missing row, rather than a broken
  frame, can be lost in a "byte-identical" rewrite.
- Proposed fix to the plan:
  - Q3.4's pin adds: an input taller than the height, a row wider than the frame, width 0, colour off,
    and the Watchpost Light theme.
  - Q3.1 and Q3.6 name their router pins, plus an old-against-new property test over random message
    sequences for step 2.

### RR-9 [Important]: Q2.8's goldens miss what Q3 to Q6 change: colour, the console's floor and the radio frame
- Plan task(s): Q2.8 (RN-11), against Q3.2, Q3.3, Q3.4, Q6.1 and Q6.2.
- Observed:
  - The console refuses below 100x44 (`bcMinCols = 100`, `bcMinRows = 44`, `broadcaster.go:637-647`), so
    the planned 80x24 console golden is the "TERMINAL TOO SMALL" notice. No console golden is planned at
    the 160x74 the beat pin uses.
  - All planned goldens are plain and ASCII.
    - PF-2 changes `Width` only on strings that carry SGR, and only `frame-133x44-colour.golden` has any.
    - PF-4 changes the tint.
    - D-10 moves tinted card rows into render.
    - D-18 changes a tinted title and the give-way order, which shows only at a narrow card width.
  - No golden has the radio playing (`benchDash` has none), yet Q3.3 step 2 rebuilds the player's parts.
    It is pinned only by a height-equals-rows property.
  - The About pane prints `runtime.Version()` (`help_about.go:375`), so a byte golden of it breaks on
    every toolchain bump.
- Consequence: the console's colour (PF-4, D-10) and the radio player's content (PF-3) change behind
  goldens that cannot see them. The D-18 frames shown to the HUM LEAD would not include the give-way.
- Proposed fix to the plan:
  - Console goldens at 100x44, 133x44 and 160x74, plain and colour, with cards of every grade and an
    operator card.
  - A radio-playing frame golden at 133x44 (compact) and 133x70 (full).
  - Help and About with the version masked.

### RR-10 [Important]: Re-pointed mutants are never re-vetted, and BUILD has no verdict sweep
- Plan task(s): step 5 of every task, Q6.3, After BUILD.
- Observed:
  - Step 5 vets *new* mutants and only "re-points" moved ones. `mutant-anchors` proves a mutant finds its
    line and `mutant-check` that it compiles. In the Makefile's words, "NEITHER ASKS WHETHER ANYTHING STILL
    FAILS WHEN IT IS APPLIED" (`Makefile:367-369`).
  - The full sweep, `make mutant-verdicts`, is "owed at BUILD exit and before SHIP", but appears nowhere in
    the plan's After BUILD. Q7.7 changes where its record goes.
  - Drift is certain:
    - 73 anchors include comment text, e.g. `m622`'s `// AND REDUCE MOTION` inside `Router.Update`, a Q9.4
      essay function and a D-12 target.
    - `mAC9`, `m642` and `mV5` (`displayWidth`) sit on lines Q3 rewrites.
  - D-13's "41 mutants" is 30 on `platform/lineup/director.go`, the scheduler, which is not renamed, plus
    11 on `app/director.go`. Only those 11 move.
- Consequence: a mutant re-pointed onto new code that no test kills any more is a pin silently lost, and
  no gate in `verify` notices.
- Proposed fix to the plan:
  - Step 5 re-vets every re-pointed mutant (apply, killed by its named test, restore by copy).
  - After BUILD runs `make mutant-verdicts` before REVIEW.
  - Q6.3 says 11 mutants, and that the 30 on lineup's `director.go` do not move.

### RR-11 [Important]: Q7.5's go-studs rule would flag 45 test files, including the golden harness
- Plan task(s): Q7.5 (D-10).
- Observed:
  - Outside `platform/render` and `third_party/`, `modes/tty/broadcaster.go` is the only production
    importer of go-studs. But 45 `modes/tty/*_test.go` files import it, `golden_test.go` among them:
    `rendering.SetColorEnabledForTest`, `:16`, `:26`.
  - The existing `scripts/lint-imports.sh` greps `--include='*.go'`, which includes tests.
- Consequence: written like its neighbour, Q7.5 forces edits to 45 test files in a checks batch. Those
  files include the colour switch every golden depends on, so pins change under a "checks" commit.
- Proposed fix to the plan: state that the rule covers non-test files. If tests are in scope, add a render
  test helper first and move the tests to it in their own commit, goldens unchanged.

### RR-12 [Important]: Q9.7 moves Settings key branches that no test runs, and names no pin
- Plan task(s): Q9.7 (JR-13).
- Observed:
  - The coverage profile shows these `setupRowKey` blocks are never run:
    - the maps-on arrow (`setup.go:381-383`)
    - the reduce-motion arrow and its flash (`:391-396`)
    - the toggle-row arrow and its flash (`:402-408`)
    - the `p` preview (`:424-427`)
  - Two of the three flash idioms that `flashed(forward)` would replace are among them. JR-13 says the
    chain is order-dependent: a row placed after the generic branch is never reached.
- Consequence: an extraction that reorders reduce-motion behind the generic toggle branch, or inverts the
  flash, ships green.
- Proposed fix to the plan: pin each left/right branch first (maps-on, map description, a map picker,
  reduce motion, a tone toggle, theme, a voice picker, preview), with the flash side, then extract.

### RR-13 [Minor]: Q5.5 merges copies whose edges differ, with no rule for when a pinned edge changes
- Plan task(s): Q5.5 (DR-9, DR-16).
- Observed:
  - `propanswers.Target.Near` checks latitude only (`platform/propanswers/answers.go:41`). Folding it into
    `geo.OnEarth(lat, lon)` adds a longitude refusal.
  - The three `hostOf`s differ on a missing host and on userinfo (`app/maps.go:246`,
    `domains/temperature/quota.go:90`, `platform/httpx/httpx.go:340`).
  - Q5 says "each copy's behaviour is pinned before the copies merge", but has no step for a pin the merge
    must break.
- Proposed fix to the plan: for each such merge, record the differing edges as unreachable, with the
  evidence, as an A-row; or leave `Near` as it is.

### RR-14 [Minor]: Q1.6 must pin the words that still load
- Plan task(s): Q1.6 (D-9).
- Observed: the legal set is "", "synth" and "relay". `mode` is `omitempty` (`platform/config/config.go:48`),
  so every config that never pressed `m` carries "". The plan pins only the refusal.
- Proposed fix to the plan: the pin also loads "", "synth" and "relay", and the testdata configs, so a
  closed set that forgets "" fails here, not at a listener's start-up.

### RR-15 [Minor]: Q1.4's strip moves the hand-off cache key, which the plan says keeps the full name
- Plan task(s): Q1.4 (D-7).
- Observed:
  - `handoffKey` builds its key from `spokenName` (`domains/radio/synth/source.go:378-380`).
  - The "different correspondent" tests (`:362`, `:443`) also compare `spokenName`.
  - The plan says "The cache key keeps the full name", but stripping inside `spokenName` changes all three.
    Two hand-configured voices that share a base name would then share hand-off audio, and would skip the
    hand-off line.
  - D-7's "the hand-off goldens change" has nothing to change: no golden carries a suffix (`cycle.golden`).
- Proposed fix to the plan: key `handoffKey` and the two comparisons on `v.Name()`, and say no golden moves.

### RR-16 [Minor]: Q1.5 omits the mirror divergence
- Plan task(s): Q1.5 (D-8).
- Observed: an existing entry *with* a ZIP, at the exact coordinates of a ZIP-less new default, is dropped
  today (`app/setup.go:63`) and kept under `PlaceID`. The plan pins only the opposite case. The new rule
  must also keep the `i == 0` drop.
- Proposed fix to the plan: add that pin.

### RR-17 [Minor]: Small gaps in Q3.2, Q4.3, Q7.6 and Q7.7
- Q3.2: the randomized width equivalence test needs a fixed seed, plus a fuzz target. Otherwise a rare
  failing cluster turns into a one-leg CI flake.
- Q4.3:
  - The memo slot is "shared by every copy of the model" (`dashboard.go:839`). A per-copy counter bumped in
    `setTicker` lets two sibling copies with different tapes share a key. Use a process-wide counter, as
    snapshot pointers are unique.
  - `TestTickAdvanceAllocBudgetWithATestEventOnTheTape`'s "frame miss" (`tick_alloc_test.go:65`) resets only
    the body memo, so after Q4.2 and Q4.3 it measures masthead and tape *hits*. Name that pin and reset all
    memos in it.
- Q7.6:
  - "Added lines" needs a diff base. The authoring tool reads the whole tree today, and hosted CI checks out
    at depth 1.
  - Q9.4 moves bare IDs, which appear as added lines. Say how moved lines are treated, and that
    qualifying an old ID uses Q8.6's index.
- Q7.7: on the work branch the tree's version is `0.19.3-N-g…[-dirty]` (`Makefile:13`), so a 0.19.5 sweep's
  record would be named after 0.19.3. Say which version names it.

## What the plan gets right
- The task order (re-check, pin first, change, `-race`, gate verdict) and "restore by copy" are the right
  regression discipline, and each Q1 defect has a pin that fails on the shipped code.
- Q4 puts a completeness guard on every memo, and makes the midnight rollover of `Since` a pin before the
  tape memo.
- Q3.2, Q3.4 and Q3.5 carry the PF lens's old-against-new equivalence tests, and PF-2's cluster fallback.
- Q6 waits for consolidated code, and the D-18 golden change goes to the HUM LEAD for a UAT look.
- M2's targets are not moved quietly: a miss goes to the HUM LEAD with figures.
- I spot-checked several things the lenses claim, and they hold:
  - PF-10's "no writes into `Hourly`" (none outside `platform/snapshot`).
  - The three compass tables are identical, and every `>=` felt-band boundary agrees.
  - The ClampRunes copies are identical.
  - Every Router fall-through returns a `Router`, and no Dashboard handler returns a nil model.

## What I did not check
- I did not run `-race`, benchmarks, the mutant harness, the app, or any package's tests other than
  `./modes/tty`'s coverage run.
- Coverage figures for `app/`, `platform/` and `domains/` are the RN lens's, not re-measured, except where
  I read the code.
- I did not trace every Settings row's surface scope to confirm that units can be changed with the console
  up. RR-1's memo point rests on FR-6.2 and D-92's forwarding.
- I did not inspect `broadcaster_memo_completeness_test.go`'s walk in full.
- Not reviewed for regression risk: Q8's document edits beyond Q8.5, Q9.1 to Q9.3, Q9.5, Q9.6 and Q9.9.
  They are documents or comments, and the only code risk I found there is RR-10's anchor drift.
- I did not check the out-of-tree P10 checker. The "bounded by" loop comments inside Q9.4's functions
  (`app/schedule.go:269`, `:377`, `broadcaster.go:1243`) must stay on their lines.
