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

## 0.19.5 batch 6 — the small fixes that are byte-identical (plan Q3.1 first step, Q3.4 to Q3.6)

Each change is held to the code it replaces by an old-against-new test, with the shipped function kept in
the test as the reference. Mutants m725 to m729 are new; m713, m642, mAC8, mAC9, mRT2, mSC1 and mSC2 are re-pointed. Every
one is hand-vetted and killed.

- **The marquee (Q3.5, PF-5):** `scrollWindow` walks the tape's text and gap in place with a `tapeCursor`
  instead of copying the tape into a rune slice every frame. `TestTheMarqueeWindowIsByteIdentical`
  compares it with the shipped function at every offset from -3 to 259, at widths 0 to 120, with three
  gaps, wide runes and invalid UTF-8.
- **The console's finisher (Q3.4, PF-4):** `Broadcaster.clamp` writes the frame in one pass, truncating,
  insetting, padding and re-arming the theme's foreground row by row, through `writeArmed`. The shipped
  finisher joined the rows and then copied the frame again for the tint.
  - `TestTheConsolesFinisherIsByteIdentical` runs at 100x44, 133x44, 160x74 and 220x60, with width 0 and
    height 0. It covers colour off and on, the default and Watchpost Light themes, and an input taller
    than the terminal whose first row is wider than the frame.
  - mAC8 and mAC9 are re-pointed to the new place the foreground is chosen, and are still killed by
    `TestTheConsoleArmsTheThemesForeground`.
  - The comments over `clamp` are brought to D-11's rule, and name the two windowing functions as they
    are now called (`scheduledSpan`, `poolSpan`).
- **The router and dispatch (Q3.1 first step, PF-1):**
  - The five fall-through returns that answer "not handled" return no model, instead of boxing the
    Router into a `tea.Model` nobody reads.
  - `Router.View` draws the active surface through its own type, not through `surfaceView`. `Init` and
    `View` both ask `onBroadcaster`, so they still cannot disagree.
  - `Dashboard.dispatch` returns the Dashboard itself, so `Update` does not unbox it again.
  - The Router suites pin the router change. `TestARefreshKeepsTheReadersPlace` pins dispatch, and kills
    m729.
- **The window over the console (Q3.6, PF-11):** `Dashboard.overlay` works out the Observer's layout once
  for both the window and its cursor. `OverlayCursor` is gone, and `OverlayWindow` is the overlay
  without the cursor. `TestTheConsolesWindowIsTheSameLaidOnce` compares it with the shipped pair for
  Settings, Diagnostics, Diagnostics' confirmation and no window at all.
- **The pins, lowered to what landed** (each × 1.05):

  | Beat | Allocations before | Allocations now | Bytes before | Bytes now |
  |---|---|---|---|---|
  | Idle Observer | 462 | 456 | 156,520 | 113,562 |
  | Radio playing | 693 | 692 | 168,720 | 126,823 |
  | Console at 160x74 | 782 | 703 | 310,457 | 231,193 |
  | A no-change publish's frame | 5,708 | 5,711 | 423,847 | 408,378 |

  - The publish frame's three extra allocations sit inside the 5 % margin, and its bytes fell. It is
    pinned at what it measures now.
  - m713 is re-pointed: the marquee's builder reserving 64 KB is fewer allocations and is killed on bytes
    alone.
  - M2 is not met yet (114 / 127 / 231 KB against 80 / 90 / 140). The width measure (plan B6) and the
    memos (Q4) are the rest of what the prototypes measured.
- **P10 (D-27):** `make p10` reports no live finding. `modes/tty` measures 0.33 over the 54 functions
  this release has changed there, against its floor of 0.42. The functions this batch changed are bounded
  loops and one-pass writers with nothing further to check, so no check was added for the count. The
  figure goes to the HUM LEAD at REVIEW exit.

## 0.19.5 batch 7 — the width measure (plan Q3.2, PF-2)

Mutants m730 to m733, each hand-vetted and killed.

- **`render.Width` sums the plain runs** between SGR sequences instead of measuring a stripped copy. The
  runs are substrings, so a styled row of the location table measures with no allocation, down from
  one allocation and 416 B, and in 286 ns instead of 360.
- **`plaintext` exports `SGRLen`,** so the stripper and the measure share one owner for what a sequence is.
- **Where it measures the stripped string whole, as before:**
  - **A rune touching a sequence that could join a cluster across it.** The lens listed the runes that
    join; this batch turns the list around. `standsAlone` names the ranges the frames draw from whose runes
    never join, and anything else beside a sequence falls back:
    - the ranges: ESC, printable Latin without the soft hyphen, punctuation without the joiners, arrows,
      technical symbols, enclosed alphanumerics, box drawing, blocks, shapes, symbols, dingbats, CJK
      ideographs;
    - why it is turned around: a rune missing from a list of joiners would be measured wrong, while a
      rune missing from this list is only measured slowly.
  - **ESC stands alone** because it is the next sequence's first byte: a cluster always breaks at a
    control, and that sequence checks the rune after it. Without it every row of sequences side by side
    fell back. m733 is that mutant, and `TestWidthAllocBudget` kills it.
  - **A string that is not valid UTF-8.** The first random run found this: runewidth measures an invalid
    byte by what follows it, so `"\xe2m"` measures 1 alone and 2 before a CJK ideograph. That difference
    holds with plain ASCII on both sides of the sequence.
- **The pins (RR-17):**
  - `TestWidthMatchesStripThenMeasure` covers a corpus of clusters split by a sequence, chains of
    sequences and invalid bytes, and 200,000 random strings on a fixed seed. The strings are drawn from
    the pieces of a sequence, whole and broken, every kind of joining rune and the runes the frames draw.
  - `FuzzWidthMatchesStripThenMeasure` is the same rule for the fuzzer. It ran 57.8 million inputs over
    five minutes, and three more minutes after ESC was admitted, with no difference. The random test
    also ran on ten further seeds, two million strings each, with no difference.
  - `TestWidthAllocBudget` holds the styled row at no allocation. `platform/render` gains the race flag
    that the allocation pins skip on.
  - The bench helper `colourOn` takes a `testing.TB`, and the declaration set is re-captured for
    `mayJoinAcross`, `standsAlone` and `plaintextEsc`.
- **The beat pins, lowered to what landed** (each × 1.05):

  | Beat | Before (allocs / bytes) | Now (allocs / bytes) |
  |---|---|---|
  | Idle Observer | 456 / 113,562 | 401 / 104,843 |
  | Radio playing | 692 / 126,823 | 626 / 112,447 |
  | Console at 160x74 | 703 / 231,193 | 537 / 183,812 |
  | A no-change publish's frame | 5,711 / 408,378 | 5,595 / 388,674 |

  M2 stands at 105 / 112 / 184 KB against 80 / 90 / 140. The memos (Q4) and the radio's split (Q3.3) are
  the rest.
- **The test's literals:** the invisible and combining runes in the test are written as `\u` escapes, as
  staticcheck's ST1018 asks. The first lint run caught the raw characters.
- **P10 (D-27):** `modes/tty` is unchanged at 0.33 over 54 changed functions; recorded for REVIEW.

## 0.19.5 batch 8 — the radio player is planned, then drawn once (plan Q3.3, PF-3)

Mutants m734 to m737, each hand-vetted and killed.

- **What changed:**
  - **The plan.** `radioPlan` is the first half of the player: the parts both players share (title,
    state, controls) and every decision the height follows from (the narrow breakpoint, the VOL ladder,
    whether the head takes its short form). It draws no marquee and no visualiser.
  - **The heights.** `radioPlan.rows(compact, viz)` counts either player's rows. `layoutWith` reads both
    heights from it, so `layoutRows` carries the plan rather than two rendered players.
  - **The render.** `layout` renders only the player of the mode in force, once, after the bands have
    given way. Before, it rendered the full and the compact player every frame and drew one of them.
  - **The two builders.** `radioRender` draws the player a plan describes. `radioMaxRows` reads its head,
    tail and room from the plan instead of laddering again. `radioLines` is plan then render, for the
    tests that call it.
  - **The plan's two steps in one.** Building the compact rows only when compact (step 1) is the case
    where the frame is compact: it falls out of the render.
- **The pins:**
  - **`TestTheRadiosPlannedHeightIsItsRows`.** It covers 15 widths from 40 to 220 columns, across both
    breakpoint edges, playing a voice with a long station name or stopped, the visualiser on and off.
    In each case the plan's count equals the rows drawn, compact and full. At 24, 44 and 70 rows the
    frame's player is the rows of the mode in force. m734, m735 and m736 are killed by it.
  - **The radio goldens (Q2.10) and every layout suite.** They hold unchanged.
- **`TestLayoutOncePerViewAndKey`'s bound restated.** It allowed a frame up to twice a layout plus 100
  allocations.
  - What broke it: the layout fell from 127 allocations to 93, while the frame's own work beside its
    layout stayed at 195. The frame (288) then crossed the old bound (286) with no second layout.
  - The new bound is absolute: the frame minus one layout is at most 195 × 1.10. A second `layout()`
    adds a whole layout on top. m737 adds one to `View` and is killed.
- **The beat pins, lowered to what landed** (each × 1.05):

  | Beat | Before (allocs / bytes) | Now (allocs / bytes) |
  |---|---|---|
  | Idle Observer | 401 / 104,843 | 367 / 102,508 |
  | Radio playing | 626 / 112,447 | 463 / 104,634 |
  | Console at 160x74 | 537 / 183,812 | 537 / 183,813 |
  | A no-change publish's frame | 5,595 / 388,674 | 5,561 / 385,203 |

- **The frame, setup, map, Propagation, console, Tab and tick budgets** are lowered in one pass when Q3
  closes, after the handler batch. Their comments quote the figures they were measured at, and re-pinning
  them once per batch would rewrite those comments each time. Until then the beat pin carries the paths
  they watch.
- **M2** stands at 103 / 105 / 184 KB against 80 / 90 / 140.
- **P10 (D-27):** `modes/tty` measures 0.32 over the functions this release has changed there, against its
  floor of 0.42. `rows` and `radioRender` are decisions over a plan that hold no state to guard. Recorded
  for REVIEW.

## 0.19.5 batch 9 — the handlers return their own types; Q3 closes (plan Q3.1 second step, the budget sweep)

Mutants m738 to m740 are new; m142, m595, m600, m620, m622, m664, m729, mRT2, mSC1 and mSC2 are
re-pointed. Every one is hand-vetted and killed.

- **The trace, recorded first.** `TestRouterUpdateIsTheSameUnderRandomMessages` (RR-8) runs three seeded
  sequences of 300 messages through a Router built by `NewRouter` over the bench dashboard.
  - The messages: navigation, the windows, the player, the swaps, digits, typing, three sizes, and the
    shimmer, visualiser and ticker ticks.
  - After each one it records the surface on screen, whether a command came back, and a SHA-256 of the
    frame. The 900 steps hold 455 distinct frames, 296 of them on the console.
  - It was recorded on the tree before any handler changed (`testdata/router-trace.golden`) and is
    unchanged after.
  - **It made the blinks deterministic first.** Its first run under the race detector differed at step
    340: Settings' picker blink had expired. The pressed-chip blinks were set and ended on the wall
    clock, so a slow run outlived the 350 ms. They are now set and ended on the model's clock
    (`d.clock()`): `markEnds`, the tick that ends them, and the four places that draw them. The model's
    clock is `time.Now` in the app (`NewDashboard`), so nothing changes for a listener.
  - `TestABlinkEndsOnTheModelsClock` pins that seam, killing m738 and m739. The trace then held under
    race, under another time zone and on one CPU.
- **The Observer.**
  - All 47 handlers that returned `(tea.Model, tea.Cmd)` return `(Dashboard, tea.Cmd)`, and the 5 that
    returned `(tea.Model, tea.Cmd, bool)` return `(Dashboard, tea.Cmd, bool)`.
  - `Update` is now one line that returns `update`, the one place the Dashboard becomes a `tea.Model`.
  - `dispatch` returns the Dashboard and its command. The invariant that the handler answered with a
    Dashboard is gone, because the compiler now guarantees it.
  - `windowKeys`' two handler fields and `routed` follow.
  - Four places that unboxed a handler's answer read it directly: the fused-esc key, the window toggle,
    the first-run save, and the typed-row settle.
- **The Router.**
  - Its seven internal methods return `Router`, and `Update` is the one place it becomes a `tea.Model`.
    The mirror block after `update` no longer type-switches.
  - The three places that reached Observer through its boxed `Update` call `update`.
  - The five not-handled returns hand back the router value itself, which costs nothing now that it is
    concrete; mRT2, mSC1 and mSC2 move back to that line. m740 drops the Observer's answer and is
    killed by the trace.
- **The tests:** 135 call sites that unboxed a handler's answer or re-assigned a Dashboard from `Update`
  are edited mechanically, by `.(Dashboard)` removed and `Update` read as `update`. `Update` is `update`
  boxed, so each test runs what it ran before.
- **The docs:**
  - The `surfaceView` comment and the `Update` comment say what each call returns.
  - where-things-happen's "A key is pressed" row names `update` as the routing and `Update` as the
    wrapper (JM-15).
  - The ledger rows for `Router.Update` and `Router.update` (P10-01) were re-read against the new shape.
    They still match their findings, and their claim holds: the cycle is a name match, and `Dashboard`
    holds no `Router`. But their proof names `r.surface()`, which only `Init` reaches now. The
    re-wording goes to the HUM LEAD at REVIEW with D-27's density figures, since the ledger changes only
    on ratification.
- **The budget sweep**, deferred from batch 8. Every allocation pin is lowered to the highest of three
  measurements on this tree, at its own rule:

  | Pin | Before | Now (measured) |
  |---|---|---|
  | Beat: idle / radio / console, bytes | 102,508 / 104,634 / 183,813 | 80,707 / 82,830 / 175,591 |
  | Beat: idle / radio / console, allocs | 367 / 463 / 537 | 364 / 460 / 536 |
  | A no-change publish's frame | 5,561 / 385,203 B | 5,560 / 378,756 B |
  | Frame hit, 133x44 / 133x70 / 200x60 / 80x24 | 365 / 383 / 386 / 377 | 288 / 282 / 285 / 289 |
  | Frame miss, the same sizes | 5,608 / 11,477 / 14,477 / 2,188 | 5,520 / 11,319 / 14,231 / 1,133 |
  | Severe window, hit / miss | 1,740 / 5,248 | 1,723 / 5,202 |
  | Settings 133x44, hit / miss | 2,849 / 5,119 | 2,761 / 4,759 |
  | Settings 80x24, hit / miss | 1,641 / 3,882 | 1,212 / 3,183 |
  | A Settings Tab press | 3,740 | 3,152 |
  | Map window, hit / miss | 3,541 / 4,011 | 3,228 / 3,787 |
  | Propagation window, hit / miss / colour hit / colour miss | 1,280 / 1,742 / 1,762 / 3,415 | 1,252 / 1,716 / 1,650 / 3,262 |
  | Console frame, hit / miss | 501 / 9,296 | 422 / 9,232 |

  - The console's hit keeps its narrow 4 % (439), so a `locIndex` back outside the memo (29 allocations)
    still fails it. Its miss is 9,694 at 1.05.
  - The ticker's step (68) did not move.
  - `docs/accepted-costs.md`'s console entry states the new pins and their slack.
- **The gate's first run** stopped on two things:
  - mSC3, a tripwire mutant that converted `keyAction`'s returned model back to a `Router`. That no longer
    compiles once the model is a `Router`. It is re-pointed to take the value directly, and still
    survives by design.
  - `platform/plaintext`'s density row, which stated no floor. Its floor is written at the measured
    1.50 over 2 changed functions (A-19, for the HUM LEAD's veto).
- **P10 (D-27):** `modes/tty` measures 0.43 over 110 changed functions, at its floor of 0.42 again. The
  handlers whose signatures changed carry the guards they always had, so they now count among the
  changed functions. All seven firing density rows are at or over their floors.
- **M2 after Q3:** idle 80.7 KB against 80, radio 82.8 against 90, console 175.6 against 140.
  - The radio is met.
  - Idle is 0.7 KB over.
  - The console is 36 KB over. D-22's console work, the masthead memo and the Projection cache, is plan
    Q4.

## 0.19.5 batch 10 — the memos and copies (plan Q4.1, Q4.2, Q4.3, Q4.6)

Mutants m741 to m750, each hand-vetted and killed.

- **The tape memo (Q4.3, PF-8, F-32 closed).**
  - The pin came first. `TestTheTapeTurnsOverAtMidnight` gives an alert's time as a time of day before
    midnight and as a date and time after it, with nothing else changing.
  - `tapeOf` is a lane's items, its tape as the band draws it, and its loop length as the scroll counts
    it. The frame and every tick read it from one slot in the body memo.
  - **The key.** `tapeKeyFor`: a process-wide ticker generation, the lane, `--ascii`, the clock format,
    and the year and day of the model's own clock, as `Clock.Since` compares them. The generation is
    `tickerGens`, bumped by `setTicker`, so two copies of the model holding different tapes never share
    a key (RR-17).
  - **The contract.** "The ticker changes only through `setTicker`" is stated on the field. The tick
    pin and the map menu's blink test write through it, where they wrote into the ticker directly.
  - **The loop length** is counted from the items as the shipped code counted it, with the dot bullet.
    The prototype joined with the frame's bullet instead, but the two bullets are the same seven runes,
    so either counts the same.
  - **The guard (A-15).** `TestTheTapeKeyCoversEverythingTheTapeShows` runs the reflection walk over the
    Dashboard, plus by name the three inputs it cannot reach: the ticker's contents through `setTicker`,
    the clock a day on, and the other lane. A failure names `tapeKeyFor` and `tickerGen`.
  - **A flaw the vetting found.** m741, the day dropped from the key, first survived the midnight pin.
    `setTicker` measures the tape, so the slot had been filled under the real clock, which already dates
    the alert. The pin now sets its clock before the ticker, and checks the band before midnight too.
  - **The tick budget pin.** `TestTickAdvanceAllocBudgetWithATestEventOnTheTape` resets every memo for
    its miss, through `missEveryMemo` (RR-17).
  - **What it saved.** A tick falls from 68 allocations to 3. The tick pin's header now says what the
    tick costs and why. F-32 is struck in `follow-ups.md`, and F-16's parked offsets are untouched.
- **The masthead memo (Q4.2, PF-7, D-22).** One `mastheadSlot` type serves both surfaces: the Observer's
  in `bodyMemo`, the console's in `consoleMemo`.
  - **The key.** `mastheadKeyOf` holds what each masthead is drawn from, as values: the options without
    the animation phase, the edition and version, the data's as-of instant, the providers' counts, the
    age words and staleness, the theme generation and the colour switch. The controls inside the box are
    fixed for each edition.
  - **Not the snapshot pointer.** The prototype keyed on it. The values let a snapshot changed in place
    still miss, and they leave out the header row the prototype's key did not cover.
  - **The shared counter.** `countProviders` is the counting both the key and `apiSummaryOf` use.
  - **The guard.** `TestTheMastheadKeyCoversEverythingTheMastheadShows` runs the walk over both models,
    plus six named inputs on both surfaces: the clock two minutes on, the clock past `staleAfter`, a
    provider's health, a provider fetched later, the theme and the colour switch.
  - **A flaw the vetting found.** m747 (the colour switch) first survived. Undoing the theme input calls
    `SetTheme`, which moves the generation, so every later input differed by the theme alone. Each named
    input now takes its own baseline.
- **The hours shared (Q4.1, PF-10).**
  - `Assembler.Snapshot` and `harmonize` share each section's hours instead of copying them twice. The
    rule "Apply replaces a section's hours and never writes into them" is stated at the copy site.
    `TestAPublishedForecastIsNeverWrittenAfterwards` pins it: a snapshot published before a new forecast
    keeps its hours. m748 writes the new hours into the old slice and is killed.
  - Every write to `Hourly` in the tree is a replacement, checked by search. The days are still copied,
    because harmonize fills them in place.
  - The assembler's publish falls from 336 allocations and 998,625 B to 259 and 75,594 B.
- **Q4.6, PF's minor list.**
  - **The seen store is written only when it has changed.** It loads dirty, so the first save of a
    session still writes and tightens the file's mode (R3-D-01). A failed write stays dirty.
    `TestTheSeenStoreWritesOnlyWhatChanged` pins it.
  - **What stays, as PF measured it:**
    - `cfg` stays a value, since the `With*` builders assign into it and a pointer would alias every copy;
    - `radioLines` is subsumed by batch 8;
    - `mapModalKey`, `statsFingerprint` and the engine poll are small.
- **The pins, lowered to what landed.** Each is the highest of three runs, × 1.05, with the console hit at
  × 1.04:

  | Pin | Before | Now |
  |---|---|---|
  | Beat: idle / radio / console, bytes | 80,707 / 82,830 / 175,591 | 59,557 / 68,476 / 166,744 |
  | Beat: idle / radio / console, allocs | 364 / 460 / 536 | 166 / 327 / 475 |
  | A no-change publish's frame | 5,560 / 378,756 B | 5,426 / 362,318 B |
  | Frame hit, 133x44 / 133x70 / 200x60 / 80x24 | 288 / 282 / 285 / 289 | 155 / 149 / 152 / 141 |
  | Frame miss, the same sizes | 5,520 / 11,319 / 14,231 / 1,133 | 5,387 / 11,186 / 14,098 / 985 |
  | Severe, Settings 133x44 and 80x24, hit / miss | 1,723 / 5,202; 2,761 / 4,759; 1,212 / 3,183 | 1,590 / 5,069; 2,628 / 4,626; 1,064 / 3,035 |
  | Tab press; map hit / miss | 3,152; 3,228 / 3,787 | 3,019; 3,095 / 3,654 |
  | Propagation hit / miss / colour hit / colour miss | 1,252 / 1,716 / 1,650 / 3,262 | 1,210 / 1,674 / 1,581 / 3,194 |
  | Console hit / miss | 422 / 9,232 | 388 / 9,198 |
  | A ticker step | 68 | 3 |
  | The assembler's publish | 336 / 998,625 B | 259 / 75,594 B |

  - `TestLayoutOncePerViewAndKey`'s own-work bound is re-pinned at 62 × 1.10. The frame's own work fell
    from 195, and at the old figure a second layout would have fit under it. m737 is killed again.
  - `docs/accepted-costs.md` states the hit pins and the console's pins.
  - The quality baseline's bounds table names the two new memos.
- **The gate's first run** stopped on m714, anchored on the hourly copy this batch removed. It is
  re-pointed to bring that copy back, and `TestAssemblerPublishAllocBudget` kills it.
- **P10 (D-27):** `modes/tty` measures 0.41 against 0.42. `app/pipelines.go`'s row measures 0.50 against
  0.61, from the publisher's clock seam (batch 5) and the seen store's flag. Neither adds a check for the
  count; both go to the HUM LEAD at REVIEW.
- **M2 now:** idle 59.6 KB against 80 (met), radio 68.5 against 90 (met), console 166.7 against 140. The
  console's `Projection` cache (Q4.5) is plan batch 10, and M2's first verdict follows batch 11's content
  generation (plan B9).
