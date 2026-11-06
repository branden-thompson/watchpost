---
title: "0.19.5 quality and performance pass — IMPLEMENTATION PLAN"
date: 2026-11-06
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "DRAFT for the plan red team and the PLAN gate. No code: signatures, test descriptions, file paths and order only (0.18.0 D-13)."
---

# Implementation plan

**Goal.** Meet the brief (`../01-objectives/project-brief.md`, D-4) and carry out the rulings D-5 to
D-18 and A-2 to A-11, without a regression. Every task names the lens findings it closes
(`../02-analysis/lens-*.md`) and the tests that pin it.

**Branch.** `feature/quality-pass`, cut from `main` (`4c803d99`), squash-merged as `release/0.19.5` at
SHIP (A-1).

**Every task follows the same order** (0.19.0's, kept):
1. Re-check the finding against the code at its `file:line`. A finding that no longer holds is recorded
   in the build log as not holding, and nothing is changed.
2. Write or name the pin, and see it fail for the named reason: on the shipped behaviour for a defect,
   or on a hand-applied mutant for a behaviour-preserving change. Restore by copy, never
   `git checkout`.
3. Make the change.
4. Run every touched package whole with `-race`, then `make alloc-budget` where a pin moved, then
   `make verify`. Read MAKE_EXIT and "ALL GATES GREEN" before committing; the commit is chained to the
   verdict. Markdown-only commits take `make verify-docs`.
5. New mutants are vetted by hand: applied, compiled, killed by their test, restored by copy. Mutants
   whose anchors moved are re-pointed in the same commit (`make mutant-anchors`).
6. The record moves with the code: the build log entry, the as-built map and its diagrams, and the
   atlas (`go run ./tools/atlas`) in the same commit.
7. Push only when no hosted run on the branch is in progress. Each batch's hosted run is read before
   the next push.

**No output changes without a D-row** (R1). A golden that changes names the ruling that changed it in
the build log, and is checked line by line to differ by that alone.

## Work packages

| WP | What | Closes | Rulings | Order |
|---|---|---|---|---|
| Q1 | The ruled behaviour changes | DEF-1, DEF-2, DEF-3, DEF-5, DR-3, DR-4, RN-4 | D-5 to D-9 | first (D-5) |
| Q2 | The regression net | RN-1 to RN-3, RN-5 to RN-13, PF-6 and RN's minor list | A-10 | before Q3 to Q9 |
| Q3 | Performance: the small fixes | PF-1 to PF-5, PF-11 | — | after Q2 |
| Q4 | Performance: the memos and copies | PF-7 to PF-10 and PF's minor list | D-4 | after Q3 |
| Q5 | Consolidation that preserves behaviour | DR-1, DR-5 to DR-14, DR-16, DR's minor list | D-17, A-4, A-5, A-8 | after Q2 |
| Q6 | Structural moves | DC-4, DR-15, JR-5(c), JR-7 | D-10, D-13, D-14, D-18 | after Q5 |
| Q7 | The checks, and the sites each finds | DC-1, DC-9, DC-12, DC-14, JR-2, DEF-4 | D-10, D-11, D-12, D-16, A-7 | after Q6 |
| Q8 | Docs and artifacts | DC-2 to DC-17 left after Q7 | D-11, D-15, A-3, A-6, A-9 | after Q7 |
| Q9 | Readability | JR-1, JR-3 to JR-13 left after Q6 to Q8 | D-12, A-2 | after Q8 |

Q3 and Q5 may interleave: both need only Q2. Q6 waits for Q5 because the arbiter rename and the card
lane move are easier to review over consolidated code. The checks (Q7) land before the docs and
readability passes so that those passes are held by them.

---

## Q1: The ruled behaviour changes (D-5 to D-9)

Each task's pin fails on the shipped code and passes after; each is a behaviour change a D-row rules.

- **Q1.1 DEF-1 (RN-4): backspace erases a letter in every text field.**
  - Pins: one test per key in the Line-Up Request window, in `modes/tty/request_keys_test.go`: up wraps
    to Position; tab skips report rows; space toggles a report and flips PRIORITIZE; digits fill the
    slot; backspace erases one rune in each field, "Peñ" included. Named
    `TestTheRequestWindowsKeysDoWhatTheyClaim` and `TestBackspaceErasesALetterNotAByte`.
  - Change: one helper `eraseRune(s string) string` in `modes/tty`, used by the eight erase sites
    (`request.go:449`, `:453`, `modal_location.go:69`, `setup.go:801`, `:833`, `:857`, `:881`,
    `map_prop_find.go:102`, `map_prop_freq.go:252`, `router.go:1168`).
- **Q1.2 DEF-2: the console honours the units and clock Settings (0.16.0 FR-6.2).**
  - Pin: `TestTheConsolesPoolWearsTheListenersUnits` (`modes/tty`): a metric config draws °C and km in
    the pool table, and an imperial one °F and mi.
  - Change: `Broadcaster.opts()` carries Units and Clock from the shared configuration; the console
    takes them as Observer does. The manifest's hard-coded 24-hour stamps (`broadcaster_manifest.go:99`,
    `:129`) are checked against 0.16.0's rulings first, and changed only if no ruling fixes them.
- **Q1.3 DEF-3 (D-6): every metric spoken temperature says "degrees Celsius".**
  - Pin: `TestTheWaterTemperatureSaysTheListenersUnit` (`domains/radio/synth`).
  - Change: `tempWords` delegates to `degrees`; its comment then states the one rule.
- **Q1.4 D-7: the identity lines drop the language suffix.**
  - Pin: `TestAHandOffSaysTheVoicesNameWithoutItsLanguage` (`domains/radio/synth`), with "Eddy (English (US))".
  - Change: `synth.spokenName` strips " (…)"; `app.spokenName` and its test are deleted;
    `app/testdata/declset.txt` is re-captured. The cache key keeps the full name.
- **Q1.5 D-8: Setup decides "the same place" by `snapshot.PlaceID`.**
  - Pins: `TestSetupKeepsTwoPlacesThatShareCoordinatesButNotAZip`,
    `TestSetupKeepsAZiplessEntryBesideTheNewDefault` and `TestSetupFoldsCoordinatesEqualToFourDecimals`
    (`app/setup_test.go`).
  - Change: `applySetup` compares `snapshot.PlaceID` of the two entries.
- **Q1.6 D-9: an unknown `[radio] mode` is refused at load.**
  - Pin: `TestAnUnknownRadioModeIsRefusedNamingTheKey` (`platform/config`).
  - Change: `Validate` checks the mode's closed set; `config.go:118`'s comment names the set the code
    holds.
- **Record:** the build log's batch entry, the CHANGELOG's Fixed and Changed lines, and README where it
  states a wording.

## Q2: The regression net (no behaviour change)

Tests only, except one seam (Q2.1). Each new pin is seen to fail on a hand-applied mutant.

- **Q2.1 RN-1: the radio deck's voice and relay-fault side.**
  - Seam: an unexported interface on `radioDeck` naming the engine methods it calls (`Suppress`,
    `Preview`, `PreviewAside`, `PausePreview`, `ResumePreview`, `StopPreview`, `DropHeld`, `Restore`,
    `Status`, `Start`, `Halt`, `Volume`), set from `*player.Engine` in `newRadioDeck`.
  - Pins, with a recording fake: a muted class sounds no tone; a tone plays aside; a viz clip plays
    through `Preview` and a takeover clip through `PreviewAside`; the player line's failure words; the
    relay-fault chain (silence, then candidates, then the picked station's mounts lead, then the
    countdown's read); `SetMode` saves always and re-tunes only when the monitor has the air.
- **Q2.2 RN-2:** muting a tone class in Settings: the empty-set mute case, un-muting the last class, and
  the row words.
- **Q2.3 RN-3:** the seen store's cap is what trims (a window longer than the oldest id), and which ids
  survive; the alert store's cap at 201 notes.
- **Q2.4 RN-5:** `marineReportOf` reads tide times in the location's zone; `seismicReportOf` with a zero
  `AsOf` is unknown; one `Compose` with marine, fire and seismic pins order, pauses and the manifest
  line.
- **Q2.5 RN-6:** the operator's card (origin, slot, reports, id); `requestedByOf`'s three words; one
  render of Change Position and its error line.
- **Q2.6 RN-7:** two providers on one host: the worse state and the older fetch win, in both orders.
- **Q2.7 RN-8:** the favourites cap: the note, the muted chip, the inert enter, the refusal. The note's
  number comes from `WatchCap`.
- **Q2.8 RN-11: whole-frame goldens** for the console on STANDBY and ON AIR, Details for a coastal
  location, Help, Status and Diagnostics, plain and ASCII, at 133x44 and 80x24, from the existing
  fixtures.
- **Q2.9 RN-12, RN-13:** `layerAnswerWords` with two answers per layer; the marine Details rows for a
  second swell, wind waves and buoy wind, in both unit systems.
- **Q2.10 RN-9, RN-10 (A-10):** `propMsgs` classifies the command by type; the launch-fill test takes
  `platform/sched`'s fake clock with F-51's.
- **Q2.11 PF-6: the beat pin.**
  - `TestABeatCostsWhatItCosts` (`modes/tty`) runs `Router.Update(msg)` then `View()` for an idle tick, a
    radio-playing visualiser beat and a console tick at 160x74.
  - It pins allocations and bytes (`testing.Benchmark`'s `AllocedBytesPerOp`) at today's figures ×1.05.
  - The radio fixture is promoted from the PF lens's scratch benchmarks.
- **Q2.12 RN's minor list:** the scroll key-to-direction mappings, Diagnostics' scroll, the three
  `waitFor` helpers' bounds, the duck wait's condition, the replay's sleep, and the history sweep's
  clock.

## Q3: Performance, the small fixes (PF-1 to PF-5, PF-11)

Each lands with the old-against-new equivalence test the PF lens prototyped. The beat pin and the frame
budgets are lowered to what lands.

- **Q3.1 PF-1:** the router's fall-through returns stop boxing the model; `Router.View` calls the active
  surface directly; `dispatch` returns the concrete `Dashboard`. Then, as a second step, `Router.update`
  returns `Router` and an internal `Dashboard.update` returns `Dashboard`; the public `Update` keeps
  `tea.Model`.
- **Q3.2 PF-2:** `render.Width` sums the plain segments between SGR sequences, falling back to
  strip-then-measure where a rune next to an escape could join a cluster. `plaintext` exports its SGR
  length so the grammar has one owner. Pin: `TestWidthMatchesStripThenMeasure`, a randomized
  equivalence test over escapes, CJK, emoji with modifiers, ZWJ, flags, Hangul and Devanagari.
- **Q3.3 PF-3:** the compact radio rows are built only when the layout is compact, and the full rows are
  reused at the narrow breakpoint. A second step splits `radioMaxRows` into a plan and a render, pinned
  by `TestTheRadiosPlannedHeightIsItsRows` across widths, visualiser and playing.
- **Q3.4 PF-4:** `Broadcaster.clamp` becomes one sized pass, as Observer's `frameText`. Pin:
  `TestTheConsolesFinisherIsByteIdentical` at 100x44, 133x44, 160x74 and 220x60, colour on.
- **Q3.5 PF-5:** `scrollWindow` walks the strings instead of building a rune slice. Pin:
  `TestTheMarqueeWindowIsByteIdentical` over every offset, widths 1 to 120, both gaps, wide runes and
  invalid UTF-8.
- **Q3.6 PF-11:** `Router.View` computes the Observer's options once and passes them to
  `OverlayCursor` and `OverlayWindow`.

## Q4: Performance, the memos and copies (PF-7 to PF-10; D-4's targets)

Every memo has a key that a reflection completeness guard covers (`memo_completeness_test.go`'s
pattern), and it is listed in the cache table with its bound.

- **Q4.1 PF-10:** `harmonize` shares the provider's hourly copy (S). Then the assembler shares
  `sec.Hourly` into the snapshot under a pinned rule, "Apply replaces, never writes into", in the style
  of the existing alias tests (M). `Daily` is left as it is.
- **Q4.2 PF-7:** a one-entry masthead memo beside `bodyMemo`. It is keyed on the version, the options
  with Frame zeroed, the snapshot pointer, the age words, the stale flag and the theme generation.
- **Q4.3 PF-8 (F-32):** one tape memo, keyed on a ticker generation bumped in `setTicker`, the lane,
  ascii, the clock format and the local date. Both the marquee and `tickerLoopLen` read it. The
  midnight rollover of the `Since` form is pinned first.
- **Q4.4 PF-9:** the assembler keeps a content generation that `Apply` bumps only when a location's
  drawn data changes, and the table memo keys on it instead of the snapshot pointer.
  - Pins: the generation's completeness guard; a no-change publish redraws no table; a changed reading
    redraws its row; the issue #11 and #12 stale-row cases.
  - The masthead's `Updated:` stamp still follows every publish.
- **Q4.5 PF's minor list:** the seen store's dirty flag; `tickerLoopLen` reads the frame's bullet.
- **M2's verdict:** the beat pin's figures against 80, 90 and 140 KB, recorded in the build log. A miss
  goes to the HUM LEAD with the measured figures; the targets are not quietly moved.

## Q5: Consolidation that preserves behaviour

Each copy's behaviour is pinned before the copies merge.

- **Q5.1 DR-1:** `render.FeltBand(mag float64) int`, whose boundaries are pinned at 3.49, 3.5, 4.49, 4.5,
  4.99 and 5.0. `SeismicLevel`, `seismicBand` and `feltLikelihood` index their own words by it.
- **Q5.2 DR-5:** `severe.Stamp` exported; `toSevereRow` calls it.
- **Q5.3 DR-6 (A-4):** the alert window as a `time.Duration` on the record. Pin: a real `shortDur` row
  read through `spokenWindow`'s wording.
- **Q5.4 DR-7 (D-17):** `geo.CompassIndex` normalises negative bearings, pinned for negatives;
  `geo.Compass16(deg float64) string` owns the abbreviations; usgs, synth and tty call it.
- **Q5.5 DR-8 to DR-14, DR-16:**
  - `plaintext.ClampRunes` at the four hand copies;
  - `geo.OnEarth(lat, lon float64) bool`, in the NaN-refusing form, at the seven sites;
  - one `viewWidthKm` for the map title and the description;
  - DR-11 per A-8;
  - `missingValues` folded into `missingRow`;
  - `hourRow` taking several row slices;
  - `cast.Class.row()`;
  - one `httpx.HostOf`.
- **Q5.6 DR's minor list and A-5's test helpers.**

## Q6: Structural moves

- **Q6.1 D-10:** the card lane's row construction moves behind a render function. The console's goldens
  (Q2.8) and its allocation budget hold it.
- **Q6.2 D-18:** render exports a top rule taking a tinted title and a badge, and the card boxes use it.
  The console goldens change by that alone, and those frames are shown to the HUM LEAD for a UAT look.
- **Q6.3 D-13:** `app/director.go` becomes `app/arbiter.go`, and the type `director` becomes `arbiter`,
  with its tests, docs, where-things-happen and the 41 mutants re-pointed and re-vetted. The 0.16.0
  as-built page and its diagram move in the same commit.
- **Q6.4 D-14:** `NewDashboard` calls grouped builders: `mapPrefsFrom(cfg Config) mapPane`,
  `uiPrefsFrom(cfg Config) uiPrefs` and `radioPrefsFrom(cfg Config) radioPrefs` (names are A-rows if
  they move). Each is a keyed literal, one field per line. `setupTable`'s rows become keyed too.

## Q7: The checks, and the sites each finds (A-7)

Each check is run over the tree first; it lands with the fixes for every site it finds and with
self-test specimens, good and bad.

- **Q7.1 SN-02 widened** to types, values and struct fields, with any present-tense verb as an opener.
  It lands with the DC-1 / JR-2 sites: about 30 fused, doubled or orphaned docs, and `executors.read`'s
  contradicting half.
- **Q7.2 `tools/dupes`** fails on a stale ratified row. It lands with `7dbf0b6d0c66` deleted and the test
  count in `duplicates-ratified.md` refreshed (DC-12).
- **Q7.3 AP-HIST-01 over `#` comments** in the Makefile and `scripts/`, with the Makefile and script
  narrations restated in the present tense (DC-9).
- **Q7.4 `docs/extending.md`'s symbols** are checked like where-things-happen's, and its drift is fixed
  (DC-14, JR-9).
- **Q7.5 the go-studs import rule** (D-10), in `scripts/lint-imports.sh` or a test.
- **Q7.6 release-qualified citations** (D-11) on added comment lines, and the all-caps continuation
  warning (D-12).
- **Q7.7 D-16 (DEF-4):** `HYGIENE_KEEP`'s comparator comes from `git describe`, and `VERDICTS_RECORD`
  becomes `06_docs/mutants/verdicts/<version>.log`. A test pins both derivations; 0.16.0's record is
  left as shipped.

## Q8: Docs and artifacts

- **Q8.1 DC-2, DC-17:** each comment naming a missing symbol is re-pointed or restated; orphan comments
  are deleted.
- **Q8.2 DC-3, DC-10 (A-6):** `follow-ups.md` is cleaned, keeping `TestFollowUpsCiteTestsThatExist`'s
  floor.
- **Q8.3 DC-5 (A-9):** README's Settings text and where-things-happen's Settings row are fixed. The HUM
  LEAD retakes the three screenshots before SHIP.
- **Q8.4 DC-6:** the as-built maps' five errors are fixed, and counts that a test derives are dropped.
- **Q8.5 D-15, A-3:** `tools/atlas` renders the as-built maps and `docs/` only. `docs/architecture.md`
  holds the program's shape as built. `TestTheAtlasMatchesAFreshRender` and
  `TestAChangedDiagramChangesTheAtlas` follow.
- **Q8.6 D-11:** `06_docs/ids.md`, generated and checked like the P10 ledger mirror; where-things-happen
  explains the scheme.
- **Q8.7 DC-8, DC-11, DC-13, DC-15, DC-16 and DC's minor list:**
  - the Makefile's misplaced blocks and timings;
  - the package docs;
  - tooling-backlog, required-gates, the red-team brief and gate-attack-list;
  - the Settings code's self-description;
  - the "round N" citations.

## Q9: Readability

- **Q9.1 JR-3, JR-4:** `modes/tty/doc.go` and `app/doc.go`, each with the package's shape and a table of
  file groups. `Dashboard`'s type doc says it is the Observer surface the Router holds.
- **Q9.2 JR-5(a)(b), JR-6:** vocabulary rows for the five roles, the arbiter, Observer as `Dashboard`,
  Settings as `setup*`, the two report packages, duck, burst, pool, programme, segment, surface,
  correspondent and HUM LEAD; "lane" marked as overloaded; the width local renamed. README's build
  section gains a "Start here" block.
- **Q9.3 JR-7:** extending.md's Walkthrough 6, "add a Setting", naming the sites Q6.4 leaves and the
  guards to extend. The `picker` field's doc and the "twenty" count are fixed.
- **Q9.4 JR-8:** the seven essay-bodied functions move their rationale into a short "Why" list in the
  doc, keeping the IDs and one-line WHAT comments inline. The first, `Broadcaster.stationLine`, is shown
  to the HUM LEAD before the other six (JR-8's own advice).
- **Q9.5 JR-9:** the pool lookup's results are named; the two "N ANSWERS" headings become one line.
- **Q9.6 JR-11:** README's build section in four steps: prerequisites, the inner loop, before a PR
  (`make verify` takes about 30 minutes, records a line, refuses an overlapping run), and Markdown only.
- **Q9.7 JR-13:** `setupRowKey`'s left/right chain extracted into `cycleFocused(forward bool)
  (Dashboard, bool)`, with a comment saying the order matters; a `flashed(forward bool)` helper; and
  `cyclePicker`'s non-cast rows moved beside their owners.
- **Q9.8 D-12, A-2:** code-standards.md gains the all-caps rule and the P10 section. Code that Q1 to Q9
  touched is brought to the all-caps rule.
- **Q9.9 JR's minor list.**

---

## After BUILD

- **REVIEW:** a red team over the whole diff, and M5's blind reviewer with a junior persona, who traces
  the four first tasks (add a Setting, add a data source, change a screen, change what the radio reads)
  from the docs alone.
- **VALIDATE:** the HUM LEAD's UAT build: D-18's card rule, D-7's identity lines, D-6's water
  temperature, DEF-2's metric console; the screenshots (A-9); M2's verdict.
- **SHIP:** `release/0.19.5`, the CHANGELOG dated, hosted runs green, the tag, the release job, the
  assets checked, the branches deleted.
