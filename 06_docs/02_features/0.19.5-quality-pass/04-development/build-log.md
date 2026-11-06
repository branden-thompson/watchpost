---
title: "0.19.5 quality and performance pass — build log"
date: 2026-11-06
phase: BUILD
sev: SEV-0
status: "LIVE — what each batch built, and why"
---

# 0.19.5 build log

## Before BUILD: the prototypes D-22 names

On top of the PF lens's prototypes of the small fixes (PF-1's first step, PF-2 to PF-5), four pieces were
prototyped in a scratch copy of the tree. None of this code is in the tree; the prototypes are a
measurement and are kept, with the lens's, in the evidence folder outside it:

- PF-7: a one-entry masthead memo, one per surface;
- the console's `Projection`, cached per line-up generation;
- PF-8: one tape memo, keyed on a process-wide ticker generation bumped by `setTicker`;
- PF-1's second step, on the hot path only: the tick and visualiser messages pass through the Router and
  the Observer as concrete values, boxed once, on the way out of `Router.Update`.

Bytes per beat (`BenchmarkPF_Router{Tick,Viz,ConsoleTick}Beat`, three runs each, figures stable to the
byte):

| Beat | 0.19.3 | the small fixes | + masthead memo | + `Projection` cache | + tape memo | + PF-1 second step | D-4's target |
|---|---|---|---|---|---|---|---|
| Idle Observer, 133x44 | 156.5 KB | 104.9 | 96.8 | 96.8 | 83.8 | **62.0** | 80 |
| Radio playing, visualiser on | 169.7 KB | 112.5 | 104.4 | 104.4 | 97.9 | **76.1** | 90 |
| Console tick, 160x74 | 310.5 KB | 161.3 | 152.4 | 137.8 | 137.8 | **129.6** | 140 |

**All three of D-4's targets are met on the prototypes**, with 18, 14 and 10 KB to spare. PF-9 and PF-10
are not needed to meet them; they still land (Q4.1, Q4.4), because they cut the cost of a publish, which
the beat does not measure.

**One test failed on the prototypes, and it is the case the plan names.**
`TestTickAdvanceAllocBudgetWithATestEventOnTheTape` writes `d.ticker[0].Test = true` in place, and the
tape memo, keyed on a generation that only `setTicker` bumps, went on drawing the unmarked tape. That is
the stale-frame risk A-15 guards against. Q4.3 makes the ticker change only through `setTicker`, and its
guard perturbs the inputs. Every other test in `modes/tty` passed on the prototype copy.

## 0.19.5 batch 1 — the ruled behaviour changes (Q1, Q7.7)

Every task's pin failed on the shipped code before its fix, and each fix has hand-vetted mutants
(m655 to m674), all killed.

- **DEF-1 (D-5, RN-4): backspace erases a letter in every text field.** One `eraseRune` (`modes/tty/textfield.go`)
  serves the ten erase sites; each keeps its own side effect (Settings' suggestions, the seeded radii,
  Change Position's error line). Pin: `TestBackspaceErasesALetterInEveryField`.
- **DEF-6 and DEF-7 (D-26), found by the Request window's new pin:** Bubble Tea names the space key
  "space", never " ", so the window's toggle never fired; and tab reused the row-walking `next()`, so
  it never left the reports. Space now toggles a report and flips PRIORITIZE, and still types a space
  into the location; tab moves by field (`requestState.nextField`). Pin:
  `TestTheRequestWindowsKeysDoWhatTheyClaim`. A grep found no other handler matching a bare " ".
- **DEF-2 (D-5) and D-21: the console wears the units and clock Settings.** The Router mirrors both
  into the Broadcaster on every update, as it does the gain; `Broadcaster.opts()` carries them, and the
  console's memo key holds them. Pins: `TestTheConsolesPoolWearsTheListenersUnits`,
  `TestTheConsolesMastheadWearsTheListenersClock`, `TestAUnitsChangeRedrawsTheConsolesPool`. The card
  manifest's stamps (`broadcaster_manifest.go:99`, `:129`) keep the 24-hour form the 0.16.0 reference
  mock draws (`mock-broadcaster-v2.txt`); they are put to the HUM LEAD at the batch's report.
- **DEF-3 (D-6): every metric spoken temperature says "degrees Celsius".** `tempWords` takes the
  conditions' `degrees`. Pin: `TestTheWaterTemperatureSaysTheListenersUnit`.
- **D-7: the identity lines drop the language suffix.** `synth.spokenName` strips " (…)"; the hand-off
  cache key and the "a different correspondent" checks use the full name (`voiceID`), so two voices
  sharing a first name keep their own hand-offs (RR-15). The dead `app.spokenName` and its test are
  gone, and `app`'s declaration set re-captured. No golden carried a suffix. Pin:
  `TestAHandOffSaysTheVoicesNameWithoutItsLanguage`.
- **D-8: Setup decides "the same place" by `snapshot.PlaceID`.** Pins: `TestSetupKeepsTwoPlacesThatShareCoordinatesButNotAZip`,
  `TestSetupKeepsAZiplessEntryBesideTheNewDefault`, `TestSetupKeepsAZippedEntryBesideAZiplessDefault`,
  `TestSetupFoldsCoordinatesEqualToFourDecimals`.
- **D-9: an unknown `[radio] mode` is refused at load.** `Radio.Validate` checks the closed set the
  comment already named. Its doc said a bad cast mode "is reported and read as the default"; `Load`
  refuses it, so the doc now says what the code does. Pins: `TestAnUnknownRadioModeIsRefusedNamingTheKey`,
  `TestTheRadioModesWordsLoad`.
- **D-16 (DEF-4, Q7.7): the Makefile's release paths are derived.** `LAST_TAG` is the newest release
  tag; `HYGIENE_KEEP` keeps its binary, and `VERDICTS_RECORD` is
  `06_docs/mutants/verdicts/since-v<LAST_TAG>.log`, which `promote-verdicts` refuses to write in a clone
  with no tags. 0.16.0's record is left as shipped. `gateoracle.RequireMake` is exported so the new test
  shares its make check. Pin: `TestTheReleasePathsAreDerivedFromTheLastTag`.
- **P10 (D-27):** `make p10` reports no live finding. One density row is under its floor: `modes/tty`
  measures 0.25 over the 16 functions this release has changed there, against 0.42 ratified over
  0.19.0's 344. Ten of the 16 are the backspace sites, one line each. No check was added for the count.
  `platform/config`'s `Validate` gained a real guard: the source mode's closed set is now the
  `radioModes` table its comment names, which holds that package at its floor of 1.00.
