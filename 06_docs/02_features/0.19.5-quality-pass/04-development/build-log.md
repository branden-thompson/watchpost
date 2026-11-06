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

## 0.19.5 batch 2 — the regression net, tests only (Q2.1 to Q2.8)

No production code changed. Every new pin was seen to fail on a hand-applied mutant (m675 to m707), all
killed.

- **Q2.1 (RN-1), the radio deck's voice and relay-fault side** (`app/deck_voice_test.go`), on a real
  engine over an output that plays each clip to its end, and a headless program whose messages the test
  reads, with no new seam (A-14). The pins:
  - a muted class sounds no tone, and a tone never feeds the bars;
  - a viz line feeds the bars and a takeover's does not;
  - a line that cannot be read says why on the detail line;
  - a spent clip is said only while a read was on the air;
  - the relay-fault window offers the other stations in order, and a pick's mounts are tried first;
  - [m] saves the pick whatever holds the air. The re-tune's refusal while the console holds the air was
    already pinned by `air_boundary_test.go`.

  Not pinned: the "no voice for …" words. They need a host with no voice at all, and on Linux reaching
  that path starts a real Piper download.
- **Q2.2 (RN-2):** enabling one class under an empty mute keeps the others muted; enabling the last
  muted class turns mute mode off.
- **Q2.3 (RN-3):** both stores' caps keep the newest entries, reached by the cap rather than the window.
- **Q2.4 (RN-5):**
  - the marine report reads tides in the location's zone;
  - a seismic state is unknown until the feed answers;
  - a composed broadcast orders maritime, fire, seismic and the sign-off, each report opening after
    two seconds, and the marine report carries its manifest line.
- **Q2.5 (RN-6):**
  - the operator's card is minted as the operator's, with their reports, admitted at their position;
  - a card says who asked for it in the operator's words;
  - Change Position says its range and its error.
- **Q2.6 (RN-7):** a shared host shows its worst feed and its oldest fetch, in either order.
- **Q2.7 (RN-8):** the watchlist stops at `WatchCap`: the note, the inert enter, the refusal, the
  unavailable chip.
- **Q2.8:**
  - RN-12: a layer says its best answer whichever came first. The fixture's radar answer is itself
    no-data, so the pin sets one, and it checks that each answer says different words.
  - RN-13: the marine section's second swell, wind waves and buoy wind, in both unit systems.
  - JM-8: `TestEveryArrowRowChangesSomething` presses ← and → on every toggle and picker row and requires
    the window to change, which catches a row whose branch in `setupRowKey`'s chain is never reached
    (m706, m707).
- `new(v)` replaces a pointer helper only for a float literal: `new(270)` is an `*int` (noted for A-5).
- **A leak the batch's race run found:** pressing an arrow on the Theme row applies the theme to the
  whole process, so `TestEveryArrowRowChangesSomething` left a different theme for the tests after it,
  and two layout tests failed (`TestModuleBlocksAlignWithTables`, `TestTomorrowAndExtendedBandsTouch`).
  The test now puts back the theme it found, as the other theme-changing tests do.

## 0.19.5 batch 3 — whole-frame goldens (Q2.9, Q2.10)

- **36 goldens for the surfaces that had none** (`modes/tty/golden_surfaces_test.go`), each checked to repeat
  across runs and to show what its name says:
  - **the console** on standby and on the air, at 100x44, 133x44 and 160x74, plain and colour. Its
    running order holds an operator's card and its rail a two-alert takeover.
    `loadedConsoleAt` pins the weather's age, so the fixture is the same on every run.
  - **Details, Help, About, Status and Diagnostics** at 133x44 and 80x24, plain and `--ascii`, with the
    Go version in About masked as `<go>`;
  - **the radio playing** with the visualiser on, compact at 133x44 and full at 133x70, plain and colour.
- **A golden that drifts names its first differing line** (`firstDiffLine`), with the want and got, instead
  of printing the whole frame (JM-14).
- **DEF-8 (D-28), found by the 80x24 goldens:** with a full watchlist the Observer's frame is 31 rows on a
  24-row terminal, and the alt screen shows only the first 24. The goldens capture today's frame; B4
  fixes it and they change by that alone.

## 0.19.5 batch 4 — the frame fits the terminal (DEF-8: D-28, D-30) and the card stamps' clock (D-29)

B4 carries the two behaviour changes the HUM LEAD ruled after B3. The rest of the plan's B4 (Q2.11 to
Q2.13) moves to B5, and each later batch's number moves up one.

- **DEF-8 (D-28, D-30).** The layout gave way only in the RECENT window, so with ten favourites its fixed
  rows came to 30 (measured at 80x24, 80x30 and 100x30) and the frame needed 31.
  - `fitPriority` is now the last thing to give, after the compact player and the thin bands: the
    priority table shows as many favourites as fit, at least three. Its window is derived from the focus
    (`priorityLo`), so it holds no state of its own, and it draws RECENT's rail (`priorityRail`).
  - Writing the pin found a second shortfall: the empty watchlist's message is three to five rows, which
    the layout counted as none (`priorityHeight`).
  - Pin: `TestTheFrameNeverExceedsTheTerminal`, over five sizes, watchlists of none, three and ten, and
    every focus.
  - The twelve 80x24 goldens change from 31 rows to 24. Their diffs were read line by line: the priority
    table shows three favourites with the rail, RECENT one row, and the window overlays recentre on the
    shorter frame. No golden at another size moved.
- **D-29: the card manifests' stamps follow the clock Setting**, in the window and on the card. Pin:
  `TestTheCardStampsFollowTheClock`. Two tests that read the stamp under the default 12-hour clock now
  expect its 12-hour form.
- Mutants m708 to m712, each hand-vetted and killed.
- **mAD5 re-pointed** at the card stamp's new line (D-29) and re-vetted: it is killed by
  `TestTheCardStampsFollowTheClock`. 0.16.0's `gates.md` records `TestTheUpNextCardFollowsTheReferencesOrder` as its
  killer then; that record stays as shipped.

## 0.19.5 batch 5 — the declared seams, the beat pin, RN's minor list (plan Q2.11 to Q2.13)

Mutants m713 to m724, each hand-vetted and killed.

- **The beat pin (Q2.12, PF-6):** `TestBeatAllocBudget`, named so `make alloc-budget` selects it; the plan
  called it `TestABeatCostsWhatItCosts`.
  - It runs a message through the Router and draws the frame for the three beats M2 measures: idle
    Observer 462 allocations and 156,520 B, radio 693 and 168,720 B, console at 160x74 782 and
    310,457 B. Each is pinned at its measurement × 1.05, bytes as well as allocations.
  - m713, a tape rune slice copied twice, keeps the count under budget and is killed on bytes alone.
    That is the case PF-6 named.
  - Beside it: `TestPublishAllocBudget` (a no-change publish's frame, 5,708 allocations and 423,847 B)
    and `platform/snapshot`'s `TestAssemblerPublishAllocBudget` (336 and 998,625 B). The package gains
    the race flag the pins skip on.
- **The four declared seams (Q2.11), each pinned first and run `-race -count=20`:**
  - **The publisher's clock.** The publisher has its own `now` and `after`, so `sched.Clock`'s interface
    does not change (a narrower shape than the plan's). `TestFirstPublishIsImmediateAndLaunchFillsProgressively`
    now steps a `windowClock` instead of sleeping. `TestPublisherWindowFoldsAWave` held across the change.
  - **The Observer's ticks.**
    - Every timed message the Observer arms goes through `Dashboard.after`, made by the model's
      unexported `ticks` maker when a test sets one. `armShimmer` takes the tick to arm, so each
      surface passes its own, and the unused `vizTick` is gone.
    - The Propagation tests' fixture answers ticks at once as a `heldTick`. So `propMsgs` tells an armed
      tick from an answer by type, and a command that still blocks fails loudly after 10 s, naming the
      fix.
    - With the waits gone, `modes/tty` runs in 62 s, down from 84.
  - **The history sweep** judges a temp file's age, and an unreadable claim's, by the store's clock,
    as it already judged a readable claim.
  - **The favourites note** names `WatchCap`.
- **RN's minor list (Q2.13):**
  - the console's ↑/↓ signs and Diagnostics' scroll bounds;
  - the player tests' `waitFor` raised to the 10 s the other packages use;
  - the duck test's wait also asks that the new player never played;
  - Diagnostics finds each fire and quake provider;
  - a preview is refused while the station holds the air.

  Not done:
  - The alert replay's 5 ms sleep stays. Waiting for every woken tier to wait again was tried and was
    not enough, because the alert renders after that. Filed as F-223.
  - The console prompts' duplicated `centre` closure is a consolidation, so it goes with plan Q5.
