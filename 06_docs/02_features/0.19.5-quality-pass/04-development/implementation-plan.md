---
title: "0.19.5 quality and performance pass — IMPLEMENTATION PLAN"
date: 2026-11-06
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "APPROVED at the PLAN gate (D-25), after the plan red team (D-20; `../08-reports/red-team-plan.md`). No code: signatures, test descriptions, file paths and order only (0.18.0 D-13)."
---

# Implementation plan

**Goal.** Meet the brief (`../01-objectives/project-brief.md`, D-4, D-22, D-24) and carry out D-5 to D-24 and
A-2 to A-17 without a regression. Every task names the lens findings it closes (`../02-analysis/lens-*.md`)
and the tests that pin it.

**Branch.** `feature/quality-pass`, cut from `main` (`4c803d99`), squash-merged as `release/0.19.5` at SHIP
(A-1).

**No library patch is needed** (D-2): no finding lands in go-tuiMaps or go-ionomaps (A-12).

## Every task follows the same order

1. **Re-check** the finding against the code at its `file:line`. A finding that no longer holds is recorded
   in the build log as not holding, and nothing is changed.
2. **Pin first.** Write or name the pin, and see it fail for the named reason: on the shipped behaviour for
   a defect, or on a hand-applied mutant for a behaviour-preserving change. Restore by copy, never
   `git checkout`.
3. **Change**, applying D-12's all-caps rule to the declarations the task changes (A-13), and citing IDs
   with their release ("0.19.5 PF-7", D-11).
4. **Gate.**
   - Run every touched package whole with `-race`.
   - Run `make alloc-budget` where a pin moved.
   - Run `make p10` where functions are added, renamed or reshaped (A-14).
   - Re-capture a guarded package's declaration set (`-update-declset`) when its declarations change.
   - Run `make verify` once per batch, and read MAKE_EXIT and "ALL GATES GREEN" before committing; the
     commit is chained to the verdict.
   - Markdown-only commits take `make verify-docs`.
5. **Mutants.** New mutants, and every mutant whose anchor moved, are vetted by hand: applied, compiled,
   killed by their named test, restored by copy (A-14). `make mutant-anchors` is green in the same
   commit.
6. **The record moves with the code**, in the same commit:
   - the build log entry;
   - the as-built map and its diagrams, and the atlas (`go run ./tools/atlas`);
   - **the junior's map**: where-things-happen's rows, the vocabulary, extending.md's steps, and the
     Rules table for any rule a caller must obey (A-17).
7. **Push** only when no hosted run on the branch is in progress. Each batch's hosted run is read before
   the next push.

**No output changes without a D-row** (R1). A golden that changes names its ruling in the build log, and
is checked line by line to differ by that alone.

## Before BUILD

- **Prototypes** (D-22), in a scratch copy, measured with the beat benchmarks:
  - PF-1's second step;
  - PF-7's masthead memo, serving both surfaces;
  - PF-8's tape memo;
  - the console's cached `Projection`.

  A prototype that falls short of what M2 needs goes to the HUM LEAD with its figures before its batch.
- **The `stationLine` sample** (Q9.4's first essay move) is shown to the HUM LEAD during B2, so Q3's
  rewrites of `clamp` and `Router.Update` move their essays in the approved shape.

## Batch map

Fifteen full-gate batches (A-12). The risky changes ride alone, and the Markdown-only work rides
separately on the docs lane.

| Batch | Tasks | Why it stands alone, or with what |
|---|---|---|
| B1 | Q1 (all), Q7.7 | D-5's defects and the ruled behaviour changes; Q7.7 removes the hazard to 0.16.0's record before any sweep (A-13) |
| B2 | Q2.1 to Q2.8 | the regression net, tests only |
| B3 | Q2.9, Q2.10 | the goldens (A-14) |
| B4 | Q2.11, Q2.12, Q2.13 | the four declared seams, the beat pin, the minor list |
| B5 | Q3.1 (first step), Q3.4, Q3.5, Q3.6 | behaviour-preserving by construction or byte-identical |
| B6 | Q3.2 | every width measurement changes |
| B7 | Q3.1 (second step), Q3.3 | 49 handlers in 16 files; the radio player's split |
| B8 | Q4.1, Q4.2, Q4.3, Q4.6 | the memos and copies |
| B9 | Q4.4 | PF-9's stale-row risk, alone; then M2's first verdict |
| B10 | Q4.5, Q5 | the console's `Projection` cache (D-22), and consolidation |
| B11 | Q6.1, Q6.2 | one console change, one UAT look (D-18) |
| B12 | Q6.3 | the arbiter rename (D-13, D-23) |
| B13 | Q6.4 | the constructor's shape (D-14, A-16); then M2's second verdict |
| B14 | Q7.1 to Q7.6 | the checks and the sites each finds |
| B15 | Q8 and Q9, the Go-comment parts | docs in code and readability |

Q8's and Q9's Markdown-only parts (Q8.2, Q8.3's text, Q8.7's docs, Q9.2, Q9.6) go in separate `.md`-only
commits beside B14 and B15. Q8.5 cannot, because it regenerates the atlas.

---

## Q1: The ruled behaviour changes (D-5 to D-9, D-21) — B1

Each task's pin fails on the shipped code and passes after.

- **Q1.1 DEF-1 (RN-4): backspace erases a letter in every text field.**
  - Pin: `TestBackspaceErasesALetterInEveryField`, one table test that presses backspace on "Peñ" in
    every text field. It asserts the field and each sibling side effect: Settings' suggestions refresh,
    `serviceSeeded` resets, Change Position's `cardErr` clears. It is seen to fail on a mutant of one site
    (RR-4).
  - Pin: `TestTheRequestWindowsKeysDoWhatTheyClaim`. Up wraps to Position; tab skips report rows; space
    toggles a report and flips PRIORITIZE; digits fill the slot.
  - Change: one `eraseRune(s string) string` in `modes/tty` at the ten erase sites (`request.go:449`,
    `:453`, `modal_location.go:69`, `setup.go:801`, `:833`, `:857`, `:881`, `map_prop_find.go:102`,
    `map_prop_freq.go:252`, `router.go:1168`). Each site's side effect stays at the site.
- **Q1.2 DEF-2 and D-21: the console honours the units and clock Settings (0.16.0 FR-6.2).**
  - Pins:
    - `TestTheConsolesPoolWearsTheListenersUnits`: metric draws °C and km, imperial °F and mi.
    - `TestTheConsolesMastheadWearsTheListenersClock`, under a 24-hour clock.
    - `TestAUnitsChangeRedrawsTheConsolesPool`: units change with the console up, and the pool redraws.
  - Change: units and clock are mirrored into the Broadcaster on every `Router.Update`, as the gain is
    (`router.go:452`). `Broadcaster.opts()` carries them, and they join `consoleKey`. The manifest's
    hard-coded 24-hour stamps (`broadcaster_manifest.go:99`, `:129`) are checked against 0.16.0's
    rulings, and change only if no ruling fixes them.
- **Q1.3 DEF-3 (D-6): every metric spoken temperature says "degrees Celsius".**
  - Pin: `TestTheWaterTemperatureSaysTheListenersUnit` (`domains/radio/synth`).
  - Change: `tempWords` delegates to `degrees`.
- **Q1.4 D-7: the identity lines drop the language suffix.**
  - Pin: `TestAHandOffSaysTheVoicesNameWithoutItsLanguage` (`domains/radio/synth`), with
    "Eddy (English (US))".
  - Change:
    - `synth.spokenName` strips " (…)".
    - `handoffKey` and the two "a different correspondent" comparisons (`source.go:362`, `:443`) key on
      `v.Name()`, so two voices sharing a base name keep their own hand-offs (RR-15).
    - `app.spokenName` and its test go, and `app/testdata/declset.txt` is re-captured.
    - No golden carries a suffix, so none moves.
- **Q1.5 D-8: Setup decides "the same place" by `snapshot.PlaceID`.**
  - Pins (`app/setup_test.go`): `TestSetupKeepsTwoPlacesThatShareCoordinatesButNotAZip`,
    `TestSetupKeepsAZiplessEntryBesideTheNewDefault`, its mirror
    `TestSetupKeepsAZippedEntryBesideAZiplessDefault`, and
    `TestSetupFoldsCoordinatesEqualToFourDecimals`. The `i == 0` drop is kept and pinned (RR-16).
- **Q1.6 D-9: an unknown `[radio] mode` is refused at load.**
  - Pin: `TestAnUnknownRadioModeIsRefusedNamingTheKey`, beside `TestTheRadioModesWordsLoad`. The second
    loads "", "synth", "relay" and the testdata configs (RR-14).
- **Q7.7 rides in B1** (A-13): see Q7.

## Q2: The regression net — B2 to B4

Q2.1 to Q2.10 are tests only. Q2.11 lists the four production seams, each declared and pinned first
(A-14).

- **Q2.1 RN-1: the radio deck's voice and relay-fault side**, on a real `*player.Engine` over a recording
  `player.Output`. This extends `heldOutput` and `drainOutput` (`app/director_pins_test.go:66`); there
  is no new seam. The pins:
  - a muted class sounds no tone, and a tone plays aside (the tap, read through `Samples`);
  - a viz clip feeds the visualiser and a takeover clip does not;
  - the player line's failure words;
  - the relay-fault chain through the engine's own `reportSilence`, through candidates and the picked
    station's mounts, to the countdown's read;
  - `SetMode` always saves, and re-tunes only when the monitor has the air.
- **Q2.2 RN-2:** muting a tone class in Settings: the empty-set mute case, un-muting the last class, and
  the row words.
- **Q2.3 RN-3:** the seen store's cap is what trims, and which ids survive it; the alert store's cap at 201
  notes.
- **Q2.4 RN-5:** `marineReportOf` in the location's zone; `seismicReportOf` with a zero `AsOf`; one
  `Compose` with marine, fire and seismic.
- **Q2.5 RN-6:** the operator's card; `requestedByOf`'s three words; Change Position and its error line.
- **Q2.6 RN-7:** two providers on one host, both orders.
- **Q2.7 RN-8:** the favourites cap: note, chip, inert enter, refusal.
- **Q2.8 RN-12, RN-13; JM-8:**
  - `layerAnswerWords` with two answers per layer.
  - The marine Details rows in both unit systems.
  - `TestEveryArrowRowChangesSomething`: every `rowToggle` and `rowPicker` row changes something on left
    and on right.
- **Q2.9 RN-11's goldens** (A-14):
  - the console on STANDBY and ON AIR at 100x44, 133x44 and 160x74, plain and colour, with every grade
    and an operator card;
  - Details for a coastal location, Help, Status and Diagnostics at 133x44 and 80x24, plain and ASCII;
  - Help and About with the version masked.

  A golden failure names its first differing line (JM-14).
- **Q2.10:** radio-playing frame goldens at 133x44 (compact) and 133x70 (full), plain and colour.
- **Q2.11 The four declared seams** (RR-3), each pinned first and run `-race -count=20`:
  - **The publisher's clock** (RN-10). The pin is first an `awaitRuns`-style failure-bound test of "the
    first publish is immediate, a wave folds". Then the publisher reads the clock through the interface
    `platform/sched` offers, with a method added for `AfterFunc`. This is in the batch's own review.
  - **The Propagation tick** (RN-9). A named tick command type, so `propMsgs` classifies a command
    without a wall-clock wait.
  - **The history sweep's clock.** `platform/history/prune.go:357-363` and `claim.go:76-80` read
    `s.now()`.
  - **The favourites note** (RN-8): its number comes from `WatchCap`.
- **Q2.12 PF-6: the beat pin.** `TestABeatCostsWhatItCosts` runs `Router.Update(msg)` and then `View()`
  for:
  - an idle tick;
  - a radio-playing visualiser beat;
  - a console tick at 160x74.

  It pins allocations and bytes at today's figures ×1.05. Beside it, bytes pins for a no-change publish's
  frame and for the assembler's publish (A-15).
- **Q2.13 RN's minor list:**
  - the scroll mappings;
  - Diagnostics' scroll;
  - the three `waitFor` bounds;
  - the duck wait;
  - the replay's sleep;
  - `app/stats.go:210-231` `sources`;
  - `PreviewVoice`;
  - the duplicated `centre` closure.

## Q3: Performance, the small fixes — B5 to B7

Each task lands with an old-against-new equivalence test. The beat pin and the frame budgets are lowered
to what lands.

- **Q3.1 PF-1.**
  - First step: the router's fall-through returns stop boxing; `Router.View` calls the active surface;
    `dispatch` returns the concrete `Dashboard`.
  - Second step: `Router.update` returns `Router`, and an internal `Dashboard.update` returns
    `Dashboard`. The 49 handlers return `Dashboard`, and the public `Update` keeps `tea.Model`.
  - Pins: the Router suites, and `TestRouterUpdateIsTheSameUnderRandomMessages`, an old-against-new
    property test over random message sequences (RR-8). The ratified P10 rows for `Router.Update` and
    `Router.update` are re-read against the new shape.
  - The docs say which of the `Update`/`update` pair holds the logic, and where-things-happen's "A key is
    pressed" row follows it (JM-15).
- **Q3.2 PF-2:** `render.Width` sums the plain segments, with the cluster fallback. `plaintext` exports its
  SGR length. Pin: `TestWidthMatchesStripThenMeasure`, on a fixed seed, plus a fuzz target (RR-17).
- **Q3.3 PF-3:**
  - First step: the compact rows are built only when compact.
  - Second step: `radioMaxRows` splits into a plan and a render, pinned by
    `TestTheRadiosPlannedHeightIsItsRows` and Q2.10's radio goldens.
- **Q3.4 PF-4:** `Broadcaster.clamp` becomes one pass. Pin: `TestTheConsolesFinisherIsByteIdentical` at
  100x44, 133x44, 160x74 and 220x60, colour on and off, the Watchpost Light theme, an input taller than the
  height, a row wider than the frame, and width 0 (RR-8). `mAC9` is re-pointed and re-vetted.
- **Q3.5 PF-5:** `scrollWindow` walks the strings. Pin: `TestTheMarqueeWindowIsByteIdentical`.
- **Q3.6 PF-11:** `Router.View` computes the Observer's options once. Pin: the console-window cursor tests;
  `m642` is re-pointed and re-vetted.

## Q4: Performance, the memos and copies — B8 to B10

Every guard perturbs the inputs the frame reads, and its failure names the memo, the field and the function
to edit (A-15).

- **Q4.1 PF-10:** `harmonize` shares the provider's hourly copy. Then the assembler shares `sec.Hourly`
  under the contract "Apply replaces, never writes into", pinned in the style of the existing alias tests.
  `Daily` is left as it is.
- **Q4.2 PF-7 and D-22:** a one-entry masthead memo that serves both surfaces' mastheads (Observer's
  `header` and `Broadcaster.header`), keyed on everything the stamp reads.
- **Q4.3 PF-8 (F-32):**
  - One tape memo, keyed on a process-wide ticker generation bumped in `setTicker`, the lane, ascii, the
    clock format and the local date.
  - The contract "the ticker changes only through `setTicker`" is written as a contract comment, and
    `tick_alloc_test.go:59` writes through `setTicker` (A-15).
  - `TestTickAdvanceAllocBudgetWithATestEventOnTheTape` resets every memo for its miss (RR-17).
  - The midnight rollover is pinned first.
- **Q4.4 PF-9:** a content generation, bumped by every `Assembler` mutator (`Apply`, `SetLocations`,
  `SetInactive` and any added later; the guard derives the set by reflection over the method set).
  - The generation is unexported, and a pin holds the report's top-level JSON keys.
  - Pins: a no-change publish redraws no table; a changed reading redraws its row; a removed or retagged
    place leaves the table (issues #11 and #12) (RR-6).
  - Then M2's first verdict.
- **Q4.5 D-22: the console's `Projection`**, cached per line-up generation behind `consoleMemo`'s
  completeness guard.
- **Q4.6 PF's minor list:**
  - the seen store's dirty flag;
  - `cfg` by value stays (aliasing through the `With*` builders; recorded);
  - `radioLines` is subsumed by Q3.3;
  - `mapModalKey`, `statsFingerprint` and the engine poll stay as measured and small (recorded).

**M2's verdicts:** after B9 and after B13, against 80, 90 and 140 KB. A miss goes to the HUM LEAD with the
figures.

## Q5: Consolidation that preserves behaviour — B10

Each copy's behaviour is pinned before the copies merge. Where copies' edges differ and the merge would
change one, the edge is recorded as unreachable with its evidence, or the copy stays (A-17).

- **Q5.1 DR-1:** `render.FeltBand(mag float64) int`, with each value's meaning documented. Its boundaries
  are pinned, and each consumer's table length is pinned to the band count (JM-15).
- **Q5.2 DR-5:** `severe.Stamp`.
- **Q5.3 DR-6 (A-4):** the window as a `time.Duration`. Pin: a table through the read: no window, 29 s,
  30 s, 59 m 30 s, 1 d 12 h ("1 day", as today), and an onset after the issue time (RR-7).
- **Q5.4 DR-7 (D-17):**
  - `geo.CompassIndex` normalises negatives.
  - `geo.Compass16(deg float64) string`.
  - The third arithmetic form, `domains/severe/record.go:41`, goes through `CompassIndex` (DV-7).
- **Q5.5 DR-2's remainder:** `conditions` calls `windWords`; `depthWords` calls `distanceWords`; the
  spoken-quantity helpers share one file.
- **Q5.6 DR-8 to DR-14, DR-16:**
  - `plaintext.ClampRunes`.
  - `geo.OnEarth`. `propanswers.Target.Near` stays latitude-only (RR-13).
  - `viewWidthKm`.
  - A-8.
  - `missingRow`.
  - `hourRow`.
  - `cast.Class.row()`.
  - `httpx.HostOf`, with the differing edges recorded.
- **Q5.7 DR's minor list and A-5's test helpers.** The stale ledger row is owned by Q7.2.

## Q6: Structural moves — B11 to B13

- **Q6.1 D-10:**
  - The card lane's row construction moves behind a render function.
  - `render/table.go:3`'s "only consumer" wording is corrected (DV-7).
  - Pins: the console goldens (Q2.9) and its allocation budget.
- **Q6.2 D-18:**
  - Render exports a top rule taking a tinted title and a badge, and the card boxes use it.
  - A scratch-rendered golden diff, the give-way at a narrow card included, is shown to the HUM LEAD
    before B11 (A-12).
  - The console goldens change by that alone.
- **Q6.3 D-13 and D-23:**
  - `app/director.go` becomes `app/arbiter.go` and the type `director` becomes `arbiter`.
  - The fields, locals and lower-case prose that mean the arbiter are renamed, each by hand
    (`lp.director` and its kin).
  - The 11 mutants are re-pointed and re-vetted; the 30 on `platform/lineup/director.go` do not move.
  - The ledger row is re-keyed as D-23 ratifies.
  - A dated note goes under the 0.16.0 required reading's arbiter paragraph.
  - The declaration set is re-captured.
  - The 0.16.0 as-built page and its diagram move.
- **Q6.4 D-14 and A-16:**
  - Pin first: `TestEveryConfigFieldReachesTheModel`, a reflection test that each `Config` field the model
    reads reaches `NewDashboard`.
  - The Settings fields move into the embedded `mapChoices`, `uiChoices` and `radioChoices`, each with a
    builder returning a keyed literal.
  - The memo guard walks embedded structs.
  - `uiForSave` and `mapPickerRow` are keyed.
  - `setupTable` becomes one keyed row per line with zero fields omitted.
  - Then M2's second verdict.

## Q7: The checks, and the sites each finds — B14 (Q7.7 in B1)

Each check is run over the tree first. It lands with:
- the fixes for every site it finds;
- good and bad self-test specimens;
- the authoring tool's Rule, Text and Why output;
- a line in code-standards' "Running the checks" giving its command (JM-15).

The rule text each check enforces lands with it (A-13).

- **Q7.1 SN-02** widened to types, values and struct fields, with any present-tense verb as an opener.
  It lands with the DC-1 and JR-2 sites and `executors.read`'s contradicting half. A **one-package-doc
  rule** rides with it (JM-10).
- **Q7.2 `tools/dupes`** fails on a stale ratified row, and names it. Row `7dbf0b6d0c66` goes, and the
  test count is refreshed (DC-12).
- **Q7.3 AP-HIST-01 over `#` comments** in the Makefile and `scripts/`, together with **all** the
  Makefile's comment work in one place (DV-8):
  - the narrations restated (DC-9);
  - the misplaced blocks moved and the mutant timings reconciled (DC-8);
  - the wires sentence fixed.
- **Q7.4 `docs/extending.md`'s symbols** (JM-3). Bare `pkg.Symbol` and `Symbol` spans, types, constants
  and fields included, are resolved against the files the page names, with a floor and a "the matcher
  narrowed" self-check. Its drift is fixed (DC-14, JR-9). An unpathed `architecture.md` citation is
  refused (JM-13).
- **Q7.5 the go-studs import rule** (D-10), over non-test files. The refusal names the route ("add a
  render function").
- **Q7.6 D-11 and D-12:**
  - `06_docs/ids.md`, generated and checked like the P10 ledger mirror (moved from Q8, A-13).
  - The citation rule over new citations at the merge-base with `main`, with `ci.yml` checking out full
    history.
  - The all-caps rule's text and its soft warning, reported on changed declarations only.
  - A-2's P10 section, with thresholds taken from the checker and set against `setupTable` as a
    specimen. It says plainly who runs P10 (JM-12).
- **Q7.7 D-16, in B1** (A-13):
  - `HYGIENE_KEEP`'s comparator is the latest tag.
  - `VERDICTS_RECORD` is `06_docs/mutants/verdicts/since-<last tag>.log`.
  - A test injects the tag and pins both.
  - 0.16.0's record is left as shipped.

## Q8: Docs and artifacts — B15 and the docs lane

- **Q8.1 DC-2, DC-17:** each missing symbol is re-pointed or restated; orphan comments are deleted.
- **Q8.2 DC-3, DC-10 (A-6):** `follow-ups.md` is cleaned, and so are the other "0.16.5" mentions:
  `code-standards.md:5`, `perf-measurement.md:202-209`, `tooling-backlog.md:39,40,78` (DV-7).
- **Q8.3 DC-5 (A-9):** README's Settings text and where-things-happen's Settings row. The HUM LEAD
  retakes the three screenshots after B11 (A-12).
- **Q8.4 DC-6:** the as-built maps' five errors; counts a test derives are dropped.
- **Q8.5 D-15 and A-3:**
  - `tools/atlas` renders the as-built maps and `docs/` only.
  - `docs/architecture.md` holds the program's shape as built.
  - Every unpathed "architecture §n" names its file (JM-13).
- **Q8.6 DC-8's paths, DC-11, DC-13, DC-15, DC-16 and DC's minor list.** The package docs belong to
  Q9.1 and the Makefile to Q7.3 (DV-8).

## Q9: Readability — B15 and the docs lane

- **Q9.1 JR-3, JR-4, DC-11:** `modes/tty/doc.go` and `app/doc.go` replace the old package comments,
  which become file headers. `Dashboard`'s type doc says it is the Observer surface the Router holds.
- **Q9.2 JR-5(a)(b), JR-6:**
  - Vocabulary rows with a code owner in the "Defined in" column. They cover the five roles, the arbiter,
    Observer as `Dashboard`, Settings as `setup*`, the two report packages, duck, burst, pool, programme,
    segment, settle, surface, correspondent and HUM LEAD (JM-15).
  - "lane" is marked as overloaded, and the width local is renamed.
  - A "Start here" block goes in README.
- **Q9.3 JR-7 and JM-8:** extending.md's Walkthrough 6, "add a Setting".
  - It opens with the "Settings, not flags" rule and covers a toggle, a picker and a map preference.
  - It is proven by adding a throwaway Setting of each kind on a scratch copy and recording which guard
    fails when each step is skipped.
  - A "When a golden or declset fails" paragraph (JM-14).
  - Walkthrough 2 gains `providerEndpoints` (JM-15).
- **Q9.4 JR-8:** the seven essay-bodied functions move their rationale into a short "Why" list in the doc,
  in the shape the HUM LEAD approved at B2. "Bounded by" comments stay on their lines.
- **Q9.5 JR-9:** the pool lookup's results are named, and the "N ANSWERS" headings become one line.
- **Q9.6 JR-11:** README's build section in four steps.
- **Q9.7 JR-13:** `setupRowKey`'s chain becomes `cycleFocused(forward bool) (Dashboard, bool)`.
  - Pin first: every left/right branch (maps-on, map description, a map picker, reduce motion, a tone
    toggle, theme, a voice picker, preview), with the flash side (RR-12).
  - A comment says the order matters.
  - A `flashed(forward bool)` helper.
  - `cyclePicker`'s non-cast rows move beside their owners.
- **Q9.8 D-12:** verification only. The rule has been applied task by task (A-13).
- **Q9.9 JR's minor list.**
  - The long-lived one-letter fields JR names (`p`, `v`, `d`, `s`, `h`, `x`) get words.
  - The 59 "0.x.y:" stamps go where they narrate.
  - The rest is recorded.

---

## After BUILD

- **`make mutant-verdicts`**, the full sweep, before REVIEW, written to its derived record (A-14, D-16).
- **REVIEW:** a red team over the whole diff. Then M5, graded as D-24 sets it: the agent builds the hidden
  answer key on a scratch copy, and a blind reviewer with a junior persona traces the four tasks from
  README's "Start here".
- **VALIDATE:** the HUM LEAD's UAT build, covering:
  - D-18's card rule;
  - D-7's identity lines;
  - D-6's water temperature;
  - DEF-2 and D-21 on a metric, 24-hour console;
  - the screenshots (A-9);
  - M2's verdicts.
- **SHIP:** `release/0.19.5`, the CHANGELOG dated, hosted runs green, the tag, the release job, the assets
  checked, the branches deleted.
