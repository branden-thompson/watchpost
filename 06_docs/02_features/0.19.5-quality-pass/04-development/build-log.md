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

## 0.19.5 batch 11 — a publish that changed nothing draws no table (plan Q4.4, D-31); M2's first verdict

Mutants m751 to m753, each hand-vetted and killed.

- **The ruling.** The plan's mechanism was an Assembler content generation bumped by every mutator. Two
  things against it were found first:
  - RR-6's mutators outside `Apply` (`SetLocations`, `SetInactive`).
  - `finalize`, which re-derives each location from the forecast at publish time (`harmonize.go:15-17`),
    so a location can change with no mutator at all.

  The tables draw only from the two snapshots' `Locations` and fields the key already holds, so the
  HUM LEAD ruled the comparison on arrival instead (D-31). The plan's generation and its JSON-keys pin
  are not built.
- **What changed.**
  - `applySnapshot` and `applyRecent` compare the arriving snapshot's locations with the ones it
    replaces, after `sortAlerts`, so both sides are in drawn order.
  - The body key holds `sameContent.keyFor`'s stand-in instead of the pointer: the earliest snapshot
    with the same locations, or the snapshot itself.
  - A snapshot written straight onto the model, as fixtures do, stands for itself, so it is a fresh
    miss and never a stale hit.
  - The comparison costs 22 allocations and 72 µs a publish on the bench fixture, measured before the
    ruling.
- **The pins (RR-6):**
  - `TestANoChangePublishRedrawsNoTable` runs three repeated publishes, the watchlist's and RECENT's. It
    kills m752.
  - `TestAChangedPublishRedrawsTheTables` covers a changed reading, a place removed (#11) and a place
    renamed (#12), each looked for by its table row. It kills m751.
    - The rename stands in for the plan's retag, because a tag is not drawn at 133 columns.
    - The row is looked for by its number, because the radio's station line, which is not memoised,
      names the focused place too. The first version matched that line and would have passed with a
      stale table.
  - `TestAWrittenSnapshotStandsForItself` kills m753.
  - The body completeness guard holds unchanged.
- **The publish pin.** It now publishes a fresh copy of the locations, as the assembler does, where it
  shared the slice and let the comparison short-circuit. A no-change publish's frame falls from 5,426
  allocations and 362,318 B to 201 and 72,317 B, pinned × 1.05.
- **The docs:**
  - where-things-happen's "A snapshot arrives" and "body memo" rows say how a repeat publish is seen.
  - The CHANGELOG has the line.
- **The gate's first run** failed where-things-happen's symbol check, which resolves functions and
  methods, on the type `sameContent`. The row names its method `arrive` instead.
- **P10 (D-27):** `app/pipelines.go`'s row is still 0.50 against 0.61, for REVIEW. `modes/tty` is back
  over its floor.
- **M2's first verdict** (plan, after Q4.4). Measured by `TestBeatAllocBudget` on this tree, the bytes of
  one beat, the message through the Router and its frame:

  | Beat | Target (D-4) | First pinned (batch 5) | Now | Verdict |
  |---|---|---|---|---|
  | Idle Observer, 133x44 | 80 KB | 156.5 KB | 59.6 KB | met |
  | Radio playing, 133x44 | 90 KB | 168.7 KB | 68.5 KB | met |
  | Console, 160x74 | 140 KB | 310.5 KB | 166.7 KB | not yet |

  - The console has 26.7 KB to go.
  - D-22 counts the `Projection` cache (plan Q4.5, about 14 KB) toward it, and it is the next batch.
    The console's other costs are the railed boxes, about 33 KB in D-22's profile. A miss after Q4.5
    goes to the HUM LEAD with the figures, as D-22 rules.

## 0.19.5 batch 12 — the console's line-up and tables, kept (plan Q4.5, D-22, A-20)

Mutants m754 to m756 are new; mAR3 is re-pointed. Each is hand-vetted and killed.

The plan's batch 10 paired Q4.5 with Q5. Q5 is its own batch (13): it touches eight unrelated places,
and the console's M2 figure was owed first.

- **The `Projection` cache (Q4.5, D-22).**
  - `Broadcaster.projection` works the line-up out once per line-up the console is handed. Before, the
    frame's eight readers each called `lineup.Projection`, so it was worked out eight times.
  - One slot in `consoleMemo` holds both tracks, keyed on `lineupGen`. `lineupGen` is now a
    process-wide count (`lineupGens`), so two copies of the console sharing one memo never share a
    generation. That is RR-17's point about per-copy counters, applied here; mAR3 moves to the new
    assignment.
  - The cards are shared by every reader of one line-up. No reader writes into them, checked at all
    eight sites and in `slotCard` and `scheduledSpan`.
  - `TestTheConsolesProjectionFollowsItsLineup` interleaves reads between two sibling copies handed
    different line-ups. It kills m754 (a per-copy count) and m755 (the generation ignored).
  - The console beat falls from 166.7 KB to 152.1, which is D-22's estimate of about 14 KB, and is still
    12 KB short of 140.
- **The railed rows (A-20).**
  - Profiled at 152 KB, the console beat's bytes split three ways:
    - finishing the frame, 41 %, most of it the frame string itself;
    - `railed`, 22 %;
    - the station section, 14 %.
  - `railed` rebuilt every row of both tables with its rail mark on every frame, from the spans, the
    width and the glyph set, all of them the console key's. `tableRows` now keeps the railed rows in
    the memo beside the spans. `railTables` is the rail half, so `spans` is gone.
  - `TestTheConsolesTableRowsAreTheRowsItWouldDraw` runs the reflection walk over the console. With the
    memo filled for the unchanged console first, every changed copy reads the rows it would draw itself.
    It kills m756.
  - The console beat falls to 122.5 KB.
- **The pins.**
  - The console beat is 383 allocations and 122,527 B, × 1.05.
  - The console frame's hit is 297, pinned at its narrow × 1.04 (309); its miss is 9,195 (9,655).
  - `docs/accepted-costs.md`'s console entry states them.
  - The other pins did not move.
- **The gate's first run** found mAC1, mAR5 and mAR6 drifted. They were anchored on the scroll owner and
  the memo's store, which moved into `railTables` and `tableRows`. The rule is to search the mutants
  before moving code, and that search was not run here. All three are re-pointed, unchanged in what they
  break, and killed by their tests (`TestThePointerWalksAndTheWindowFollows`,
  `TestConsoleFrameAllocBudget`).
- **P10 (D-27):** `app/pipelines.go`'s row is unchanged at 0.50 against 0.61, for REVIEW.
- **M2, after Q4:**

  | Beat | Target (D-4) | First pinned (batch 5) | Now | Verdict |
  |---|---|---|---|---|
  | Idle Observer, 133x44 | 80 KB | 156.5 KB | 59.6 KB | met |
  | Radio playing, 133x44 | 90 KB | 168.7 KB | 68.5 KB | met |
  | Console, 160x74 | 140 KB | 310.5 KB | 122.5 KB | met |

  The second verdict follows Q6.4, as the plan says.

## 0.19.5 batch 13 — consolidation, first half (plan Q5.1 to Q5.5)

Mutants m757 to m767 are new; m665 is re-pointed. Each is hand-vetted and killed. Each task's copies were
pinned before they merged, with the shipped function kept in the test as the reference where the copies
are replaced. Q5.6 and Q5.7 follow as batch 14.

- **Q5.1, the felt bands (DR-1, JM-15).**
  - **Pinned first,** either side of 3.5, 4.5 and 5.0, with the shipped code:
    - `render.SeismicLevel` (`TestTheSeismicLevelAtEveryBoundary`);
    - the Details label (`TestTheSeismicLabelAtEveryBoundary`);
    - the radio's likelihood (`TestTheSpokenLikelihoodAtEveryBoundary`).
  - **One owner.** `render.FeltBand` holds the boundaries, with each band's meaning written beside it,
    and `render.FeltBands` is the count.
  - **The words stay with each surface,** as tables indexed by band: `seismicLevels` (render),
    `feltLabels` (tty), `feltLikelihoods` (synth).
  - **The table lengths.** A test in each package holds its table's length to the band count, since a
    `[N]string` literal one short would still compile, with a blank entry.
  - **Mutants.** m757 moves a boundary in the owner, and the Details label test catches it in another
    package. m758 is killed by the render pin.
- **Q5.2, the severe clock (DR-5).**
  - `TestTheSevereRowsTimesReadInThePlacesZone` pins the row's declared and expiry times, which were
    unpinned: in the place's zone, the zone named, blank with no clock.
  - `severe.Stamp` is exported and the row calls it.
  - The one edge where the two copies differed is a nil zone: `Stamp` reads local time, where the app's
    copy would have panicked. It is unreachable, because `rowClock` always returns a zone (A-17). m759
    and m760 are killed.
- **Q5.3, the alert window as a value (DR-6, A-4, RR-7).**
  - `TestTheAlertWindowIsSpokenAsTheRecordShowsIt` was written first and passed on the shipped code. It
    drives a domain row through `toSevereRow` and the spoken read for each case:
    - no window;
    - 29 s, still "0 minutes", as it was;
    - 30 s;
    - 59 m 30 s;
    - 1 h 30 m;
    - 1 d 12 h ("1 day");
    - an onset after the issue time.
  - **The window travels as a value.** `timing` returns the window beside the line. `Record.Window` and
    `SevereRecord.Window` carry it, and `spokenWindow` words that duration instead of parsing "(~…)"
    out of the display text.
  - **One owner for the rounding.** `severe.WindowParts` rounds to the minute and keeps whole days alone.
    `shortDur` writes its shorthand from it, and `spokenWindow` says it from it, so the two cannot round
    apart.
  - **What changed in the tests.** The old string-parsing cases in `TestEventScriptReadsTheAlertItself`
    take durations, and its fixture carries its window. No spoken output changed (A-4). m761 to m763
    are killed.
- **Q5.4, the compass (DR-7, D-17, DV-7).**
  - `geo.Compass16` is the one table of the 16 abbreviations. usgs's `bearing`, synth's `compassPoint`
    and tty's `compass` call it, and synth's own rounding is gone.
  - `TestCompass16MatchesTheTablesItReplaces` checks it against both shipped forms at every hundredth of
    a degree from 0 to 360.
  - **Negatives (D-17).** `CompassIndex` takes a negative or over-a-turn bearing round the rose first,
    pinned by `TestCompassIndexNormalisesANegativeBearing`.
  - **The one edge that changes.** An exact negative half-sector, such as -11.25°, is where synth's
    rounding half away from zero named the previous point. It is recorded as unreachable: NDBC sends 0
    to 360 (A-17).
  - **The storm heading.** severe's 8-point `compass` goes through `CompassIndex`.
    `TestTheTropicalHeadingMatchesTheShippedForm` holds it to the integer form for every whole degree
    from -720 to 720.
  - m764 and m765 are killed.
- **Q5.5, the spoken units (DR-2's remainder).**
  - `conditions` says the wind through `windWords`, pinned in both units by
    `TestTheConditionsSayTheListenersUnits`. `depthWords` is `distanceWords(km, false)`, pinned first by
    `TestADepthIsSaidInWholeKilometres`.
  - The ten spoken-quantity helpers live in `domains/radio/synth/spoken_units.go`: `degrees`,
    `heightWords`, `tempWords`, `windWords`, `knotWords`, `decimalWords`, `depthWords`, `distanceWords`,
    `ringWords` and `durationWords`. Each keeps its comment. m665 follows `tempWords` there.
  - m766 and m767 are killed.
- **The gate's first run** failed the declaration-set pins in two packages. Both changes were the
  batch's own, and both sets are re-captured:
  - render added `FeltBand`, `FeltBands` and `seismicLevels`;
  - app swapped `durationUnits` for `countWord`.
- **P10 (D-27):** three rows measure under their floors, for REVIEW:
  - `platform/geo` 0.50 against 1.17, from `Compass16` and the normalising lines;
  - `domains/severe` 0.22 against 0.50, from `Stamp`, `WindowParts` and `timing`'s second answer;
  - `app/pipelines.go` 0.40 against 0.61.

  Each changed function is a pure mapping with nothing further to guard, so no check was added for the
  count.

## 0.19.5 batch 14 — consolidation, second half, part one (plan Q5.6: DR-8, DR-9, DR-10, A-8)

Mutants m768 to m772 are new; m313 is re-pointed. Each is hand-vetted and killed. The mutant anchors were
searched before the gate this time.

- **DR-8, the rune clamp.**
  - Pinned first: `TestAnIDIsBoundedInRunes` and `TestAStationNameIsBoundedInRunes`. The lens found
    both bounds unpinned for length.
  - `clampProse`, `clampID` and `capName` stay as one-line names for what each bounds, and each calls
    `plaintext.ClampRunes`. Synth's inline copy in `spokenName` calls it too.
  - `capName`'s literal 48 is the named `maxStationNameRunes`.
  - m768 and m769 are killed.
- **DR-9, a place on Earth.**
  - `geo.OnEarth` is the one test, written so NaN fails it. `TestOnEarthRefusesWhatIsNoPlace` pins the
    edges, NaN and infinities.
  - The callers are AirNow's rows and contours, WFIGS's perimeters and the Open-Meteo geocoder, each
    deleting its copy, plus `ParseCoordinates`, the GeoJSON walker, the solar times, and reachscore's
    `onGlobe`.
  - **Astro.** Its `invariant.Check` takes the result through a local, so P10 still counts the check.
  - **The fuzz tests.** AirNow's and WFIGS's fuzz targets call `geo.OnEarth`.
  - **The one edge that changes.** `ParseCoordinates` and the GeoJSON walker used the negative form,
    which accepts NaN, and now refuse it. That is unreachable, because `ParseCoordinates` rejects NaN
    before the bound and the JSON decoder cannot produce one (A-17).
  - **What stays.** `propanswers.Target.Near` stays latitude-only (RR-13, A-17).
  - m313 follows the latitude bound into `ParseCoordinates`' call. m770, the negative form, and m771,
    the poles, are killed.
- **DR-10, the view width.**
  - `viewWidthKm` is the one width the map's title and its spoken description hand the area namer, at
    111.32 km a degree as before.
  - `TestTheTitleAndTheDescriptionNameOneArea` holds the two to the same width, and m772, the
    description measured at the equator, is killed.
- **A-8 (DR-11).** The four domain files that imported `render` only to sanitise text call `plaintext`
  directly. They are the radio script, synth's marine words, and severe's record and detection. severe
  no longer imports `render`, and its package comment names `plaintext.Text`. `render.Plain` and
  `render.PlainLine` stay as the modes' spelling.
- **P10 (D-27):** four rows measure under their floors, for REVIEW:
  - `tools/reachscore` 0.00 against 1.29: its one changed function, `onGlobe`, now delegates;
  - `platform/geo` 0.80 against 1.17;
  - `domains/severe` 0.31 against 0.50;
  - `app/pipelines.go` 0.40 against 0.61.

  No check was added for the count.

## 0.19.5 batch 15 — consolidation, the rest (plan Q5.6: DR-12, DR-13, DR-14, DR-16; Q5.7)

Mutants m773 to m778 are new; m425 is re-pointed. Each is hand-vetted and killed. With this batch every DR
finding is landed or recorded (M3).

- **DR-12, the missing row.**
  - `missingValues` is gone, and the temperature map calls `missingRow`.
  - No test held the row to NaN, so a row of zeros (a real 0 °C) would have passed.
    `TestAMissingRowIsAllMissing` pins it, and m773 is killed.
- **DR-13, the hour index.**
  - `hourRow` takes its rows variadically, so the temperature series passes its six measures and
    shares the rain's and the waves' one rule.
  - `TestAnHourGoesInItsPlace` holds, and kills m774, which gives a new hour one measure only.
- **DR-14, the class fallback.**
  - `cast.Class.row` is the one fallback to ClassWarning, and `ToneRank`, `Key`, `String` and `ToneName`
    read their field of it. `ToneRepeats` keeps its own rule.
  - The existing pin checks every accessor's out-of-range answer, and kills m775.
- **DR-16, the host of a URL.**
  - `httpx.HostOf` is the one answer, in the `url.Parse` form, returning the URL itself where it has
    no host. Every caller calls it directly: the failure columns, the Status window's map sources (app)
    and the temperature quota.
  - `TestHostOfNamesTheHostEveryCopyNamed` holds it to all three shipped copies on every URL those
    callers pass.
  - **The edges that differed are recorded as unreachable (A-17):**
    - app's "" for a URL with no host;
    - a URL that does not parse;
    - userinfo, which the slicing form kept and the Status column no longer would.
  - m776 and m777 are killed.
- **Q5.7, DR's minor list.**
  - **Landed:**
    - `speedWords` calls `units.MilesOf`.
    - The console's pool reads kilometres from `kmFromTower`, instead of converting miles back. The
      line-up keeps `milesFromTower`, and both read `towerKm`, so each allocates only its own answer;
      the line-up rows' allocation pin held them to it.
    - severe's `abs` is `math.Abs`. They differ at -0 only, which no feed sends.
    - The uncalled `fire.Age` is gone.
    - The hour, day and fall-back counts use `plural`.
    - Four membership loops are `slices.Contains`.
    - `tools/dupes` and `tools/authoring` skip `.claude`, so an agent worktree is not read as a second
      copy of the tree. Each skip list is pinned (`TestTheToolSkipsWhatIsNotOurs`), and m778 is killed.
  - **Kept, with the reason:**
    - `radarAheadLabel` and `hoursWords` stay two, now sharing `plural`. One is a Settings label that may
      shorten, the other is prose.
    - `notesYieldLine` keeps its singular, since its verb changes with the count.
    - `mapDepthByKey` and `propRefreshByKey` are reverse map lookups, not membership.
    - The sleep idiom, as the lens advised.
    - The age wording, a UX ruling rather than a consolidation.
    - The stale ledger row belongs to Q7.2.
- **A-5, the test helpers.**
  - The twelve pointer helpers (`f64`, `fx`, `pf`, `fp`, `f64ptr`) are `new(v)`. Each untyped integer
    constant is `new(float64(n))`: the compiler named all 108 sites, and none was left as an integer.
  - `indexOf`, `indexIn`, `containsExact`, `equalStrings`, the category `same` and the gate's `keys`
    are `slices` and `maps` calls.
  - The per-package fixtures stay.
- **The declaration set** in app is re-captured for `hostOf` and `missingValues` gone.
- **P10 (D-27):** seven rows measure under their floors, for REVIEW. This batch's changed functions are
  one-line delegations and table edits with nothing to guard:
  - `app` (`mapfire.go`'s row) 0.33 against 0.61;
  - `domains/severe` 0.36 against 0.50;
  - `domains/temperature` 0.17 against 0.50;
  - `platform/geo` 0.80 against 1.17;
  - `platform/httpx` 0.33 against 0.64;
  - `tools/authoring` 0.00 against 0.50;
  - `tools/reachscore` 0.00 against 1.29.

  At REVIEW exit each goes to the HUM LEAD for re-ratification with its figure.

## 0.19.5 batch 16 — go-studs stays behind render (plan Q6.1, D-10)

Mutant m779, hand-vetted and killed.

- **The card's title row is a render type.**
  - `render.CardRow` holds the two go-studs rows, plain and marked. `NewCardRow` builds the inset or
    mark column, the left-aligned truncating headline and the right-anchored badge. `Render` draws a
    row, with the mark in place of the inset when there is one.
  - The component's comments move with it: why two rows, why a column and not `SetPrefix`, why the
    mark never truncates.
  - The card lane keeps what a card means: the takeover's empty title row, the chip and its handle,
    and the refusal to draw a mark it cannot show whole.
  - `modes/tty` no longer imports go-studs, so `platform/render` is the only package that does, as
    `render.go`, where-things-happen and extending.md already said.
- **`table.go`'s line** no longer calls the location table the only go-studs consumer (DC-4, DV-7). It
  names the package as the one importer, and its four consumers.
- **The pins:**
  - Every console golden holds unchanged.
  - The console's allocation pins hold: the beat at 383 allocations, the frame hit at 297, the miss at
    9,170 (from 9,195), the line-up rows at 260.
  - m779 drops the mark in the render type, and `TestAFabricatedCardIsMarkedInTheLineUp` kills it.
  - The render declaration set is re-captured for `CardRow`.
- **The check that refuses a go-studs import elsewhere** is Q7.5's, as planned.

## 0.19.5 batch 17 — the cards draw the masthead's rule (plan Q6.2, D-18, D-32)

Mutant m780 is new; mAL5 is re-pointed. Both are hand-vetted and killed.

- **What changed.** render exports its titled rule as `TitledRule`. `BoxTitled` draws with it, and so does
  the console's `boxRule`, which keeps only what is the card's own:
  - the title's modal tint (D-118);
  - the badge's place at the right, which gives way first (D-110).

  The card's rule now has two marks where it had three, two spaces before the badge and two marks
  after it, and render's exact-fit give-way: the badge goes first, then the title.
- **The UAT look (A-12, D-32).** The changed lines were shown to the HUM LEAD before commit, and ruled to
  ship as shown:
  - the wide console's rules at 160x74;
  - the narrow ones at 100x44;
  - a scratch comparison of the give-way at inner widths 33 to 56. A narrow card now keeps its title,
    and then its badge, where the old rule dropped them. At the exact fit the two sit three spaces
    apart, as the masthead's stamp does.
- **The goldens.** Only the twelve console goldens changed, by the rule alone, and are re-recorded. The
  router trace's digests are re-recorded with them, since it draws the console.
- **The tests.** `TestTheBoxsRuleCarriesTheGrade` and `TestTheBoxsRuleMatchesTheReferenceGeometry`
  pinned the v3 reference's three marks, and now pin D-18's two, with their comments saying so.
- **The mutants.** mAL5 follows the title's tint into its new line. m780 drops the badge and is killed.
- **The CHANGELOG** says what an operator sees.
- **The gate's first run** failed render's declaration-set pin, `boxRule` having become `TitledRule`. It
  is re-captured.
- **P10 (D-27):** `platform/render` measures 0.10 against 0.11, for REVIEW, beside batch 15's seven rows.

## 0.19.5 batch 18 — the arbiter is named for what it is (plan Q6.3, D-13, D-23)

No new mutants. The 11 that cite the file are re-pointed and each re-vetted, killed by its own test.

- **The file and the type.** `app/director.go` is `app/arbiter.go` and its test `app/arbiter_test.go`. The
  type `director` is `arbiter`, with `newArbiter`, `buildArbiter` and the test helper `testArbiter`.
- **The identifiers, renamed by a `go/ast` pass.** It renamed by exact name only, never in comments or
  strings: 140 identifiers in 40 files. Every one outside the arbiter's own file was read and is the
  arbiter, among them the pipelines' field `lp.arbiter`, the decks' `voice`, and the schedule's and
  reader's `nar`. Lower-case `director` in `app` code always meant the arbiter, and the role is written
  `Director`.
- **The tests that test the arbiter** take its name: the five `TestDirector…` in its test file, and
  `TestReadCancelledWhileSuspendedDoesNotWedgeTheArbiter`. Tests that name the lineup's Director stay,
  such as `TestAPumpRefusesADirectorThatWasNeverBuilt`, and so does `director_pins_test.go`, which pins
  the Director's deck facts.
- **The prose, sorted by hand (D-23 c).** 24 lines in `app` say "arbiter" where they meant it,
  `app/arbiter.go` where they named the file, and the header says what the arbiter is not. Kept as they
  are:
  - the design documents' file names (`director-build-plan.md`, `director-charter.md`);
  - `executors.go`'s `director.go:842`, which is the lineup's file;
  - the pump's debug-log tags, which name the Director's events and are read by a test.
- **The records (D-23 a, b).**
  - The P10 ledger row `awaitAir` is re-keyed to `app/arbiter.go`, its text unchanged but for a
    re-key note citing D-13 and D-23. It matches its finding.
  - A dated note under the 0.16.0 required reading's arbiter paragraph says the file's name now. Its
    quotations keep the name they were written with, as do 0.16.0's rulings and findings, the spike
    result and the 0.14.0 build log, which are records.
  - The 0.16.0 as-built page and its diagrams name no arbiter file, so nothing in them moves. The atlas
    regenerates unchanged, since it lists diagram documents, not code files.
- **The map.** where-things-happen's four references name `app/arbiter.go`, and its vocabulary gains
  "the arbiter": what it is, that it is not one of the five roles nor the Director, and that 0.14.0
  ruled it dissolves into the Director's step. `quality-observations.md` and open follow-up F-158 name
  the file and the renamed test as they are now.
- **The declaration set** in app is re-captured for the rename alone.
- **The gate's first run** found the ledger's mirror, `06_docs/p10-ledger.md`, still keyed on the old
  file, so `TestEveryRatifiedP10RowNamesCodeThatExists` refused it. `make p10` regenerates the mirror
  but was run before the re-key. The mirror is regenerated now.
- **P10 (D-27):** `app`'s row measures 0.52 against 0.61 (the file the checker anchors it at moved to
  `airscope.go`), with batch 15's other rows and render's 0.10, for REVIEW.

## 0.19.5 batch 19 — the Settings in three groups (plan Q6.4, D-14, A-16); M2's second verdict

Mutants m781 and m782 are new; m297, mAI1 and mM4 are re-pointed. Each is hand-vetted and killed.

- **The pins, first.**
  - `TestEveryConfigFieldReachesTheModel` builds the model from the zero `Config` and from one with every
    Setting changed (`everySettingChanged`). It flattens the model's own fields into a golden, through any
    embedded struct and under their promoted names, so moving a field changes no line. Recorded on the
    shipped constructor, it showed all 24 Settings reaching a model field.
  - `TestThePrefsSavedAreTheSettingsInEffect` records what `uiForSave` writes for both configs.
  - Both goldens are unchanged after the move. m781 drops the quake feed from its builder and m782 from
    the save, and each is killed.
- **The move (A-16).**
  - The 28 Settings fields left `Dashboard` for three embedded, unexported structs in
    `modes/tty/dashboard_choices.go`, each with its comments. The map's twenty are `mapChoices`; the
    UI's three (units, clock, Reduce motion) are `uiChoices`; the radio's five (the correspondent, the
    volume, repeat, mode, the visualiser) are `radioChoices`.
  - Each group has a builder, `mapChoicesOf`, `uiChoicesOf` and `radioChoicesOf`, returning a keyed
    literal one field per line. `d.mapDesc` and the rest read as before through promotion.
  - `NewDashboard`'s literal is keyed, one field per line.
  - `mapCost` is no Setting and stays on the model. The comment it shared with three Settings is split
    between the two places.
- **Keyed, as A-16 asks.**
  - `uiForSave` is one field per line.
  - `mapPickerRow` is a switch, one row per line.
  - `setupTable`'s 57 rows are keyed literals with zero fields omitted. The table and the picker rows'
    answers were captured before and after the rewrite, and compare byte for byte.
- **The memo guard walks embedded structs (A-16).** `walk` reaches a field by its index path, and walks an
  embedded struct as the model itself, so its fields keep their own names. All 206 perturbations carry
  their promoted names, and a writer or excuse keyed on a field's name (`mapTempNDFD`, `mapRainFull`)
  still finds it.
- **The walkthrough (D-14).** `docs/extending.md` gains "add a Setting": the file's key, `Config` and
  `UIPrefs`, the app's hand-off and save, the group's struct and builder, `uiForSave`, the Settings row,
  and the two pins to extend.
- **Mutants.**
  - m297, mAI1 and mM4 anchored on `setupTable`'s positional rows, and follow them into the keyed form.
  - mAI1 then survived the whole package. Since D-70 every row is drawn on every surface, and nothing in
    the app reads a row's scope; the scope is D-18's ruling's record. Its old kill came from an
    allocation count that moved by chance. `TestTheStationRowsAreRuledTheConsoles` pins the ruled value
    (D-115) and kills it.
- **M2's second verdict** (plan, after Q6.4), measured by `TestBeatAllocBudget` three times:

  | Beat | Target (D-4) | First pinned (batch 5) | Now | Verdict |
  |---|---|---|---|---|
  | Idle Observer, 133x44 | 80 KB | 156.5 KB | 59.6 KB | met |
  | Radio playing, 133x44 | 90 KB | 168.7 KB | 68.5 KB | met |
  | Console, 160x74 | 140 KB | 310.5 KB | 122.4 KB | met |
- **The gate.**
  - Its first run lost the network part-way: `proxy.golang.org` did not resolve, so the pinned lint, P10
    and vuln tools could not be fetched. On the re-run everything passed.
  - P10 reports no live finding; `NewDashboard`, keyed one field per line, is inside P10-04's length.
  - The density rows under their floors are batch 15's and render's, unchanged, for REVIEW (D-27).

## 0.19.5 batch 20 — a ratification of nothing fails (plan Q7.2, DC-12)

Mutant m783, hand-vetted and killed.

- **`tools/dupes` fails on a stale ratified row.** `staleRows` names each row whose fingerprint matches no
  duplicate group the scan finds, with its sites, and `make dupes` fails on it, as `tools/wires` fails on
  its own stale rows. Its self-test gains a third control, a dead fingerprint named alone beside one that
  still matches. `TestAStaleRowIsNamed` pins it as a unit test, so m783 can be vetted.
- **The tree's one stale row goes.** Run over the tree first, it named row `7dbf0b6d0c66`
  (`Director.stepBed` · `Director.stepOperator`), whose two dispatch tables no longer share a shape.
  The row is deleted, as the plan (approved at D-25) says.
- **`06_docs/duplicates-ratified.md`:**
  - It says a stale row fails the gate.
  - Its test-duplication figure is today's: 38 unexplained across 40 groups, dated. Its "17 across 18"
    had doubled, as DC-12 found.
  - It names today's largest group, and points at the command that prints the figure.
- **The "Running the checks" list** in `06_docs/code-standards.md` names `make dupes` and what fails it
  (JM-15).

## 0.19.5 batch 21 — go-studs stays behind render, enforced (plan Q7.5, D-10)

- **The check.**
  - `scripts/lint-imports.sh` gains `check_studs`: a non-test file outside `platform/render` that
    imports go-studs fails `make lint-imports`. go-studs' own tree in `third_party`, `dist` and the
    agents' worktrees under `.claude` are not judged.
  - The refusal names the route: add a render function that does what the file needs, and call it.
  - Tests may import go-studs; dozens in `modes/tty` use its colour switch, and the rule is over non-test
    files, as the plan says.
- **The sites.** Run over the tree first, the rule found none: batch 16 moved the console's card row
  behind `render.CardRow`, which was the only non-test import outside render.
- **The self-test** gains two controls, four in all. A mode's file importing go-studs is refused, while
  render's file and a mode's test are not. `make gate-controls` runs them inside `verify`. The
  script's first form filtered with `\|`, which BSD grep's basic regex reads as a literal, so the
  good-specimen control refused the tests. That is how the control was seen to fire before it was
  trusted, and the filter is an extended regex now.
- **No mutant.** The script is run by the self-test that `gate-controls` runs, not by a Go test, so the
  mutant harness has no test to name. Its four controls are the proof.
- **The rule's text.** `06_docs/code-standards.md`'s checks list names `make lint-imports` and both rules
  it holds (JM-15). `render.go`, where-things-happen's "the seam" row and extending.md already state the
  rule.

## 0.19.5 batch 22 — a doc comment belongs to one declaration, and a package has one (plan Q7.1, DC-1, JR-2, JM-10)

Mutants m784 to m786 are new, each hand-vetted and killed.

- **SN-02 widened** (`tools/authoring/rules.go`).
  - **What it judges.** `docSubjects` is every function, type, constant, variable and struct field with
    a doc. A GenDecl's doc counts for its spec when it holds only one.
  - **The opener.** The subject's name, then `is`, `has` or any word ending in `s`. The fixed list of
    25 verbs had no "renders", "fetches", "keeps", "lifts" or "installs".
  - **Two failures.** A doc that names its subject only after line 0 is fused. One that opens with
    the subject and opens again later is doubled.
  - **Prose that starts with the name.** The first run over the tree reported 40, and six were prose:
    wrapped lines such as "A nil / frameChecks checks every frame", and new paragraphs opening with the
    name, as godoc's do. So a later line counts only when the line above it ends a sentence and is not
    a paragraph break, or when the block opens by naming another declaration. That brought it to 32,
    and every one was read and found real.
- **The 32 sites, and JR-2's two the rule cannot see.** All are comment-only changes.
  - Thirteen blocks moved to their own declarations, which had no doc of their own: `readCard`,
    `startSchedule`, `usgsFeed`, `EnsurePiper`, `capExtras`, `Provider`, `RadioStatusMsg`, `Stats`,
    `bodyMemo`, `Glyphs`, `FireState`, and two tests' docs. `TestPublisherWindowFoldsAWave`'s now
    opens with its name. `go doc` shows `FireState`, `EnsurePiper` and `startSchedule` their contracts
    again.
  - Ten docs of deleted or already-documented declarations were deleted:
    - `tickerAudio`, `verifyOnly`, two `keepOr` and `cardRow` are gone from the code;
    - the cut-off first line of a deleted test;
    - `rowMarks`' stale copy, `TestGivingUpAfterTheToneIsTraced`'s duplicate, and the first lines of
      `fakeClock` and `restore`.
  - Seven doubled docs became one: `executors.read`, `render`, `Lane`, `Tail`, `heldEscalation`,
    `railAt` and `TickerCategory`. `executors.read` keeps the half that matches its `error` return,
    so the bool its first half described is gone (DC-1, JR-2).
  - `setupKeyLines` and `severeWindowName` were one doc in two halves that agree, and each half became
    a paragraph.
  - `cast.Class` now opens with its name. The `Transmitter` paragraph moved down 48 lines to the fields
    it explains: it had been fused onto `MoveCard`'s doc.
- **SN-03, one package comment per package (JM-10).** `go doc` joins every package comment in a
  package. `checkPackageDocs` reports a package with more than one among its non-test files.
  - The tree had two such packages, `app` and `platform/render`. The fragments in `app/dashboard.go`,
    `app/setup.go` and `platform/render/theme.go` became file headers, detached by a blank line.
  - `modes/tty` has one, the stale one, and Q9.1's `doc.go` replaces it.
- **Proof.**
  - The self-test gains six specimens: a type, a constant and a field fused or doubled; a block that
    opens with another name and wraps onto its own; and the two prose forms that pass. It also gains
    SN-03's controls, in both directions.
  - `TestTheSelfTestDiscriminates` runs it all as a unit test, so a rule's mutant is killed by
    `go test`. m784 judges functions only again, m785 lets two package docs pass, and m786 counts a
    wrapped name as an opener.
- **The rule text.** `06_docs/code-standards.md`'s SN-02 says what it judges and how prose passes,
  with a field specimen. SN-03 is new, and the checks list names both. `06_docs/tooling-backlog.md`
  says what the detector decides and what still needs a reader.
- **The gate.**
  - Its first run failed lint on staticcheck's QF1001 in `ownOpeners`' condition. The condition is
    written out by De Morgan now, and the mutants were re-vetted on it.
  - P10 reports no live finding. `tools/authoring/main.go`'s density rose from 0.00 to 0.38, against
    its floor of 0.50. It joins batch 15's rows for REVIEW (D-27).
  - Its second `make verify` failed in `lint-authoring`. The cause was a process failure of mine:
    Q7.3's next change to the authoring tool was written while that gate ran, so `verify` ran a tool
    that was not this batch's. The FAILED row stays in the gate record. This batch's tree was put
    back exactly and gated again, with nothing edited while it ran.

## 0.19.5 batch 23 — the gates explain what they guarantee, not how they came to (plan Q7.3, DC-8, DC-9, DV-8)

Mutant m787, hand-vetted and killed.

- **AP-HIST-01 over `#` comments.**
  - `tools/authoring` now reads every `Makefile`, `.sh` and `.gitignore` in the tree (22 files) and
    checks their `#` comment lines with the phrase test the Go rule uses. `narrates` is that one
    predicate, shared by both passes.
  - A script's shebang and a recipe's commands are not judged.
  - Its self-test gains a control in both directions, run by `TestTheSelfTestDiscriminates`. m787
    inverts the predicate for `#` lines.
  - Run over the tree first, it found the two "used to" lines DC-9 named: `Makefile`'s schema target
    and `scripts/lint.sh`.
- **The reader pass (DC-9).** About thirty more lines narrated dated incidents that no phrase list
  decides. Each is restated as what the target or script guarantees now, and its reason is kept.
  - In the Makefile: `property`'s tag, `mutant-check` (its shell history, its log rule, its timeout),
    `hygiene` (the full disk, the `*.log` rule, the sweep's timings), `journey`, `mutant-verdicts`,
    `promote-verdicts`' `make -n` warning, the sweep's own clean-up, `wires`, `alloc-budget` and the
    schema.
  - In the scripts: `lint.sh`, `lint-imports.sh` and `p10-unmatched_test.sh` (the unknown-flag rule),
    `ledger-ratified.sh`, `mutant-verdicts.sh` (five sites), `mutant-anchors.sh` (three) and
    `workload.sh`; and `.gitignore`.
  - The incidents themselves stay in git, in the commits that wrote them.
  - Rulings and measurements keep their dates as evidence.
- **The Makefile's comment work, in one place (DV-8, DC-8).**
  - The attribution-gate line headed `lint-authoring`; it now heads `lint-watermark`.
    `lint-authoring`'s own line says "a syntax tree can decide", not "a grep", and names the `#` pass.
  - The allocation-pin paragraph headed `pty-severe`; it now heads `alloc-budget`, joined with that
    target's own block.
  - **The mutant timings are reconciled to one measured set.** The comments had "~140s", "about 400s",
    "seven minutes" and "261 s on twelve cores" for the same gate. They now give this batch's own
    measurement: 2,253 s for 1,032 mutants on the developer's 18 cores, from batch 22's gate
    (2026-11-07). The hosted run's 93 minutes for 835 mutants (2026-10-29) is kept beside it. The
    two cheaper gates say "seconds" and point at the Makefile for the figure.
  - The `wires` paragraph said five rows were "RATIFIED AS OWED". The ledger holds none, and it now
    says so.
  - `pty-severe` and `journey` join `.PHONY`.
  - `HYGIENE_KEEP` and `VERDICTS_RECORD` moved with the release in batch 1 (Q7.7, D-16), and their
    text already says so.
- **The rule's text.** `06_docs/code-standards.md`'s AP-HIST-01 says it covers the gates' `#` comments,
  what the tool decides there and what still needs a reader. `06_docs/tooling-backlog.md`'s row names
  the `#` pass.
- **The gate.** Green at first run. P10 reports no live finding. `tools/authoring/main.go`'s density is
  0.26, against its floor of 0.50. The new walk adds lines without checks, and the file's row
  stays for REVIEW (D-27).

## 0.19.5 batch 24 — extending.md names what exists (plan Q7.4, JM-3, JM-13, DC-14, JR-9)

Mutants m788 to m790 are new, each hand-vetted and killed.

- **The check** (`cmd/watchpost/extending_symbols_test.go`). JM-3 found that where-things-happen's
  pattern matches none of extending.md's spans, so this one indexes the tree instead.
  - **The index.** Every Go file is parsed, tests included, because the page names the tests to
    extend. go-studs is included too, because the page names its table API. The index holds each
    package's top-level names, each type's methods and fields (embedded ones by their type), and
    every file path.
  - **What must resolve.** A backticked span shaped like a Go symbol: `pkg.Symbol` in that package,
    `Type.Member` on that type, a field first (`Harmonized.UVIndex`) through declared names,
    `platform/geo.ReadGeometry` in that directory, a bare `Symbol` anywhere, and a call such as
    `httpx.TTL(d)` by its name.
  - **What must exist.** A span naming a `.go` file, by path or by name, and any path to another kind
    of file.
  - **What is left to the reader,** each class named in `checkable`: lowercase words and snake_case,
    which are config keys and JSON fields; ALL-CAPS environment variables; dotted keys like
    `table.header` that start with neither a package nor a type; and bare non-Go file names, such as
    the `CCL.js` a command downloads.
  - **JM-13.** An "architecture §n" citation must name its `architecture.md` on the same line, since
    the tree has several.
  - **The floor.** It checks 183 spans, and fails under 150, so a narrowed matcher cannot pass by
    checking a handful.
  - **Its own control.** `TestTheSymbolCheckRefusesWhatIsNotDeclared` runs the same index over spans
    that must be refused and spans that must pass. They include `Conditions.CheckKey`, a real name that
    is not a member of that type. It also checks that the words it skips stay skipped and that
    citations are told apart.
  - m788 lets a package name anything, m789 lets a type have every member, and m790 lets an unpathed
    citation pass.
- **The page's drift, fixed.** Run first, the check also caught the unpathed citations.
  - Walkthrough 2 (DC-14): the provider's attribution reaches the report's provider lines, not About.
    The credit goes in `creditGroups` (`app/credits.go`), and About draws it as `CreditLine` rows. No
    52-cell bound exists. The tiers are `priorityTiers` and `recentTiers`, the one owner of both lists;
    `startPriority` and the RECENT pipeline only read them.
  - Walkthrough 3 (JR-9): `withCmd` and `takeCmd` are the Dashboard's methods, not the deck's.
  - Walkthrough 4: `want.Has(report.X)` is quoted as the `if` it is.
  - Five unpathed architecture citations (§10.1, §10.3, §1.1 twice, and "PD/§10.10.5") now link to
    `06_docs/02_features/watchpost-cli/03-architecture-design/architecture.md`, whose §10.10 item 5 is
    the on-demand rule. The 29 unpathed citations in Go comments are Q8.5's (JM-13).
  - The schema note names `snapshot.SchemaVersion` rather than a version that drifts.
  - `modes/report`'s package comment said "schema v1.0-rc" against the constant's 1.1.0-rc (JR-9). It
    now names the constant.
- **The banner** names the test that holds its claim. `06_docs/code-standards.md`'s checks list gives
  its command (JM-15).

## 0.19.5 batch 25 — every ID resolvable, every new citation dated (plan Q7.6, D-11, A-13, A-21)

Mutants m791 and m792, hand-vetted and killed.

- **`06_docs/ids.md`, generated** (`tools/ids`; `go run ./tools/ids -write`).
  - It lists, for every ID the record defines, each feature's row: its release where the record states
    one (in the folder's name or its brief's title), the file, and its subject.
  - A row defines an ID when the ID is the whole of its first cell, and so does a heading that opens
    with one. 0.16.0 rules in headings, and without them its `D-55` was missing.
  - The families live in `tools/ids/idset`, the one list the index and the citation rule share.
  - `D-124` shows both its rows, 0.18.0's and 0.19.0's, which is D-11's example.
  - **How it is checked.** `TestTheIndexIsCurrent` regenerates the index and compares it, like the P10
    mirror. It runs in the docs lane too, since a docs change is what moves it.
    `TestARowDefinesAnIDByItsFirstCell` pins what is read: a definition, a heading, a later reference,
    a mention and a date.
- **AP-CITE-01** (`tools/authoring/cite.go`). `make lint-authoring` passes `-base auto`.
  - It flags a bare ID in a comment that the package's Go files did not carry at the merge-base with
    `main`. One `git grep` reads the base's IDs for the whole tree.
  - A release number qualifies every ID after it to the end of its sentence, across a comment's lines,
    and a release's prefix (`MVS-D-80`) qualifies too.
  - The self-test's control covers both directions. m791 stops a release qualifying anything.
  - `ci.yml`'s verify job checks out with `fetch-depth: 0`, so the base exists on the hosted runners
    (A-13). The release workflow already did.
- **Run over the branch first.**
  - By file, as A-13 first worded it, 127 citations were new. Most were in comments moved within a
    package, a file renamed (`director.go`), or the second line of a sentence that names its release.
  - A-21 records the rule as built: package scope and sentence scope. It is the agent's choice,
    for veto.
  - That left 25, each qualified with its release:
    - 0.19.5's `D-17`, `D-30`, `A-17`, `JM-13`, `JM-15` and `RR-17`;
    - 0.18.0's `D-248`, from the 0.18.0 commit that wrote it into `domains/airquality`;
    - 0.18.0's follow-up `F-190`;
    - 0.16.0's `D-55` and `D-87` in `render.CardRow`, which moved from `modes/tty` in batch 16.
  - 0.16.0 has no row that defines `D-87`, only plans that refer to it. The citation still names its
    release, and the index cannot list it.
  - The idset package's own doc names families, not IDs.
- **The scheme, written down.** `docs/where-things-happen.md`'s Record IDs section says how to read an
  old citation (the index, then the file's history) and how to write a new one. `06_docs/code-standards.md`
  gains AP-CITE-01, and its checks list names both commands.
- **The gate.**
  - `make verify` was green at first run. P10 reported two live P10-05 findings, both for the two
    new packages, which have no ledger row.
  - `tools/ids/idset`'s one function, a hand-written `strings.Join`, is now that call, so the
    package has no function to measure.
  - `tools/ids` measures 0.88 across 8 functions. No check is added for the count (D-27). A new
    package's row is the HUM LEAD's to ratify, so it goes to REVIEW with the density rows under their
    floors. The gate ran again on the tree as committed.


## 0.19.5 batch 26 — a capitals headline is for a contract (plan Q7.6, D-12, A-13)

Mutant m793, hand-vetted and killed.

- **The rule** (`06_docs/code-standards.md`, AP-CAPS-01). An all-caps headline is kept for a contract a
  caller must obey; rationale is written in sentence case; and no paragraph opens with an all-caps AND,
  BUT or SO. "Touched" is A-13's: a declaration the branch changed, its doc included, plus the whole of
  a file a task rewrites.
- **The soft warning** (`tools/authoring/caps.go`).
  - `-base auto` reads the branch's changed lines from one `git diff -U0`. A caps continuation that
    opens a paragraph inside a declaration touching one is reported.
  - It is printed under "soft warnings", which never fail the gate (`soft`, `split`). JM-11 counted
    147 standing hits over the whole tree.
  - The self-test's control covers a changed declaration, an unchanged one, a sentence-case "And" and a
    contract headline. m793 stops a paragraph's opening line being seen.
- **Applied to what 0.19.5 has touched, so Q9.8 only verifies.**
  - Run over the branch, it reported 76 caps continuations in changed declarations, 29 files from
    `app/` to `tools/dupes`. Each is rewritten in sentence case, its leading AND, BUT or SO dropped.
  - A first scripted pass was read line by line before it stood. Its allowlist kept "ON", "AIR" and
    "GOES" in capitals, and it carried a caps run past the end of its sentence. Both were fixed, and
    the files were restored from HEAD and the pass run again.
  - Role names (Director, MasterControl), state names (ON AIR, STANDBY), the RECENT and PRIORITY
    pipelines and IDs keep their form.
  - **Fragments rewritten whole.** Twelve paragraphs had made sense only as a list's next item, and
    each is now a whole sentence:
    - in the Router: "The console carries the same build", "The console takes the same fire
      threshold", "The console opens on the station it serves", "Observer is told which surface is
      active", "Only then is a key routed to whoever owns it", and the bed keys;
    - "Reduce motion is one switch for both surfaces";
    - `arbiter`'s claim;
    - the debug window's confirmation;
    - the console's RECENT snapshot and pool;
    - two test docs, which now open with their tests' names.
  - Labels on a struct field or a call ("What to call it on the console") read as a description of the
    line below them, and stay as labels.
  - Whether each remaining capitals headline is a contract is a reader's judgement. Q9.8 verifies it
    over touched code.
- **The gate.**
  - Its first run failed `mutant-anchors`: m622 anchored on the Reduce motion line's trailing comment
    in `router.go`, which the rewrite changed. The corpus was not grepped before the comments
    moved, which the anchors checklist asks. m622 follows the line, and is re-vetted and killed by
    `TestTheConsoleFollowsReduceMotion`.
  - Every other anchor holds (1,039 mutants).
  - P10 is unchanged: `tools/ids` only, for REVIEW.


## 0.19.5 batch 27 — P10 written down (plan Q7.6, A-2, JR-10, JM-12)

- **`06_docs/code-standards.md` gains "P10 — the safety-critical rules".** JR-10 counted 648 comments
  citing P10 rules that the tree defined nowhere.
  - **One row per rule, P10-00 to P10-10:** what it limits, its threshold, and the forms the checker
    accepts. A-2 asks for these three. A row per rule lays them out side by side; a paragraph per rule
    would give the same three things in prose.
  - **The thresholds are the checker's own,** read from the harness's pinned configuration and its
    analysers. Functions: 60 lines, 40 statements, and cyclomatic complexity under 15. Density: 2.0
    checks per changed function, or a ratified floor. Pointers: dereference depth 1.
  - **What the checker does not read.** This tree's `// bounded by … (P10-02)` comment is its own
    convention; the checker accepts the range or counter form.
- **Who runs it (JM-12).** `make p10` runs the harness's check. It is not in `make verify` or CI, whose
  runners have no harness. It runs at each batch and at BUILD and REVIEW exit, under `make quality`,
  and its figures go in the build log. Exemptions are the HUM LEAD's to ratify, and the public mirror
  is regenerated by the same target.
- **The specimen is `setupTable`.**
  - It is a function rather than a package variable (P10-06). Its body is one statement of 57 keyed
    rows.
  - It runs 101 lines, 59 of code and 34 of comments, and P10 reports nothing on it although batch 19
    changed it. So the comments are not counted against P10-04's 60, as measured, not as read from
    the checker.
  - The route at the limit is named: split it by group, or have the literal ratified. A package
    variable is not a route, since it trades P10-04 for P10-06.
- **`docs/where-things-happen.md`'s P10-nn row** points at the section, as well as at the harness.
- **The P10 figures this release carries to REVIEW (D-27).** Each package below its ratified floor:

  | Package | Floor | Now |
  |---|---|---|
  | `app/airscope` | 0.61 | 0.53 |
  | `domains/severe/record` | 0.50 | 0.36 |
  | `domains/temperature/quota` | 0.50 | 0.17 |
  | `platform/geo/coords` | 1.17 | 0.80 |
  | `platform/httpx` | 0.64 | 0.33 |
  | `platform/render/card_row` | 0.11 | 0.10 |
  | `tools/authoring/main` | 0.50 | 0.26 |
  | `tools/reachscore/spots` | 1.29 | 0.00 |

  `tools/ids` is new and has no row; it measures 0.88 across 8 functions.

## 0.19.5 batch 28 — the comments name what exists (plan Q8.1, DC-2, DC-17)

This batch changes comments only. mAG4 follows its line, and is re-vetted.

- **DC-2: names that do not exist.** Each was grepped again against today's tree, and re-pointed or
  restated.
  - **Re-pointed.** Each now names its current owner:
    - `bedReach.Line`, `tapeHead` (twice), `bedSelector` and `scheduledSpan` (with its two
      references in `broadcaster.go`);
    - `requestState.valid` and `.note`, `Card.WithScript`, `bcRailForms`, `Lineup.Next` (twice),
      `Dashboard.shownOnSurface` and `stepRow`;
    - `relayFaultPad`, `whenTail` (twice), `slotCard`.
  - **Moved since 0.14.0.** `readBreaking` became `readScript`, and its retry is `reader.readPart`.
    Three test docs now name them; the fourth says the counterfactual without a name.
  - **The layout's widths.** The comment names `tableWidth` where it named `orderWidth`, and no
    longer counts three owners where the code has seven.
  - **`writeNarration`'s doc** said what a test value guards. It now says what the function does, and
    names the §10.5 contract's file (JM-13).
  - **Said in words.** `sectionWidth()` has no successor, so its two sites describe the region in
    words. The `armDwell` counterfactual reads as one.
  - **`stopDwell` and its ledger row are gone.** `Cancel`'s reason is restated without them.
  - **Facts corrected.**
    - The bed's key is `[b]` (four sites); `B` is the door to the Broadcaster.
    - `maintrack` has two stages, not three.
    - Seismic polls twice as often as fire in PRIORITY, and a third as often in RECENT.
    - What draws the zone outlines is the go-tuimaps library.
    - The platform test points at `make test-platforms`.
  - `platform/config/config.go:118` needed nothing. `radioModes` and its check now exist, so the
    comment is true.
- **DC-17: orphans.** The fragments of the deleted `severeArrows`, `tickerAlert` and `tideTrend` are
  gone.
- **AP-CAPS-01 on what this batch touched.** Eight caps continuations sat in the declarations these
  edits changed, and each is rewritten to stand alone. The soft warning is clear.
- **mAG4** anchors on the RECENT seismic tier's comment, and follows it. It is killed by
  `TestTheRecentTiersFetchWhatTheMarksAreDrawnFrom` and `TestCadenceTableIsTheDoc`.

## 0.19.5 batch 29 — follow-ups.md carries only open work (plan Q8.2, DC-3, DC-10, A-6)

Markdown only, through the docs lane.

- **Fewer rows.** The file goes from 442 lines to about 210. The rows that remain are open, each saying
  what is left.
- **Deleted, per the file's own rule** ("close a row by deleting it in the commit that does the work"):
  - the rows marked CLOSED, FIXED, ANSWERED, RULED or DISPOSITIONED with nothing left owed;
  - the twelve the DC lens found done in the code: F-1, F-2, F-3, F-15, F-18, F-35, F-36, F-80, F-87,
    F-89, F-100 and F-173;
  - F-47 and F-64, whose remainders are done. The Makefile's timeout comment records the mutant
    timing and says where it is re-checked;
  - two sections that held nothing open, the 0.16.0 final round's and "Closed at 0.14.0".
  - Their text is in git.
- **Moved, because it is a release's record and not a follow-up.** The 0.16.0 gate layer's red-team
  rounds two to nine and its BUILD-exit paragraph were held only here. They move verbatim to
  `06_docs/02_features/0.16.0-broadcaster-ui/08-reports/gate-layer-rounds.md`, ceilings and
  dispositions included. Their open rows come back to the Open table: F-156 to F-162.
- **Cut to what remains:**
  - F-44 (the `space` read's 11.4 s and two journey gaps);
  - F-48 (the published 0.13.0 image, a HUM LEAD call);
  - F-55 (Observer's half of the size contract);
  - F-67 (three program messages no surface handles);
  - F-68 (Observer's own breakpoints);
  - F-73 (the paused main track);
  - F-90 (`verify` drives no console pty; `journey` does);
  - F-107 (`SpliceCells` has a caller; its wide-rune contract is unmeasured since);
  - F-138 (one `t.Skip` left).
- **Rewritten.** F-94 describes the console as built, the manifest and the detail window, with
  `scriptWindow` gone. F-105's three dropped fields fold into F-108(a), the restore path that would
  write them again.
- **Citations name symbols, not lines that move:** F-62 (`setupAlertLines`), F-103 (`Projection`,
  `scheduleIndex`, `visible`), F-104 (`Director.refreshStandby`), and F-211 (`release-matrix`'s
  `CGO_ENABLED=0`, oto v3.5.0-alpha.11 as `go.mod` pins it).
- **DC-10: work deferred to 0.16.5**, a release that was never cut.
  - In this file: F-121, F-133, F-137, F-139 and F-157 to F-162 say so, and each names what makes it
    due instead.
  - The two perf items `perf-measurement.md` carried there become F-224 and F-225.
  - `code-standards.md` sends F-137's sites to Q8.6.
  - `tooling-backlog.md`'s two deferred rows and its `mutant-check` cost are restated. The cost is
    now the Makefile's measured figures.
  - The 0.16.0 reports, the CHANGELOG and the 0.16.0 handoff are records, and keep the words they
    were written with.
- **The checks.** `TestFollowUpsCiteTestsThatExist` still finds its floor of citations, and every one
  exists. `06_docs/ids.md` is regenerated, since F-rows are IDs.

## 0.19.5 batch 30 — the README's Settings as built (plan Q8.3, DC-5, A-9)

Markdown only, through the docs lane.

- **The FIRMS key.** The README's step said "press `s`, `tab` to the key line". `tab` moves between
  Settings' tabs (`setup.go`), so it moves away from the key, which sits on the Data tab
  (`setup_rows.go`, `rowFIRMSKey`, group Data). The step now reads "`↓` to the key line on the Data
  tab", matching what the README says of the keys further up.
- **Two captions describe the window as built.** Settings has its tabs across the top, and the open
  tab in one column (0.19.3 D-364). The console's Settings has the station's two rows beside the tabs
  it shares. The old captions described two columns and a right-hand half.
- **where-things-happen's Settings row** described "two columns when they fit". It now names the tab
  row (`setupTabRows`) and the one-column page (`setupPage`, through `setupBody`). The symbol test
  holds all three names.
- **The images are the HUM LEAD's** (A-9). `docs/img/setup.png`, `broadcaster-settings.png` and
  `help.png` predate the tabs and the `g` and swap keys. The agent does not run the live app, so they
  are retaken before SHIP, and the captions above describe what the new ones will show.

## 0.19.5 batch 31 — the as-built maps match the code (plan Q8.4, DC-6)

Documents and the atlas only. The atlas is regenerated in the same commit (59 diagrams).

- **The air-and-deck map** (`0.16.0-broadcaster-ui/03-architecture-design/as-built-air-and-deck.md`):
  - **The guard boundary's counts are dropped,** where they had drifted. It said 53 seams (48 and 5)
    and airNone 34, while `TestEverySurfaceSeamIsClassifiedForTheAir` derives 59 and 40. The page now
    names the test that holds each class's count, and its six edges carry the class alone.
  - **The state subgraph** was titled "what MasterControl holds". It is the Director's state (power,
    monitor, air); MasterControl declares ON AIR / STANDBY. The title now says that, which is the
    five-role split the page exists to keep.
- **The map's as-built** (`observer-maps/03-architecture-design/as-built-map.md`):
  - The map's quakes come from `lp.mapQuakes.fetch`, the feed the map chose (D-122), not a
    `severeDeck.feedCopy` that does not exist.
  - Settings' map tabs are two, Maps and Map Layers, each one column (0.19.3 D-364). One "side by
    side" tab was drawn before.
  - `limitsOf` is `propanswers.LimitsOf`.
  - The window stack's "forty-three places", now 83, is said without a count.
  - The status line names 0.19.3 and 0.19.5.
- **map-ready-geometry's as-built** named a `platform/tuimap` that never existed. The drawer is the
  external go-tuimaps module.

## 0.19.5 batch 32 — the program's shape as built, and an atlas of the code (plan Q8.5, D-15, A-3, JM-13)

Mutant m794, hand-vetted and killed.

- **`docs/architecture.md`, new (A-3).** It has the program's shape as a diagram:
  - the external sources, as About credits them;
  - `domains/`, `platform/` and `modes/`, with the one enforced arrow;
  - `app` as the composition root;
  - the libraries: go-tuimaps, go-ionomaps, Bubble Tea, oto, and `say` or Piper.

  Beside it are the rules the shape keeps (no domain imports; render alone imports go-studs; one deck,
  one air; the immutable snapshot) and a map of the packages.
  - Each claim was read from the package docs: the global feed's three sources; propagation's own
    client, not `httpx`; and the scheduler asking the location providers only.
  - README's build section and where-things-happen link to it.
- **The atlas draws the code as built (D-15).** `tools/atlas` reads `docs/` and each feature's
  `as-built*.md`, so it has 14 diagrams, down from 59. It opens with the new shape. A plan's or a
  report's diagram stays readable in its own document. The page's own framing says so.
  - `TestOnlyTheAsBuiltDiagramsAreDrawn` pins the selection. `TestAChangedDiagramChangesTheAtlas`'s
    fixture is now an as-built map, since that is what the atlas reads.
  - The old watchpost-cli shape names espeak-ng and System.Speech. It stays, unchanged, as the
    approved design it was.
- **`docs/architecture.md` is held like `docs/extending.md`.** `TestArchitectureNamesRealSymbols`
  runs the same resolver over it, with a floor of 25 against the 32 spans it checks.
  - Both pages now also check that a named directory exists, such as `platform/snapshot`. A span
    counts as a directory only when its first segment is one of the tree's top-level directories, so
    `n/a` stays a word.
  - The control test gains the directory case.
  - On its first run the check refused this page's own citation, whose path sat on the next line, and
    then a path written relative to `02_features`. Both are fixed.
- **Every unpathed "architecture §n" names its file (JM-13).**
  - The 32 Go comments citing the original design now read "watchpost-cli architecture §n". Every
    cited section (§1 to §11.1) was checked to exist there.
  - `docs/architecture.md` says once which file that is, and which contracts its sections hold.
  - extending.md's five were done in batch 24.
- **AP-CAPS-01 on what this batch touched:** one continuation in `release_test.go`, rewritten.

## 0.19.5 batch 33 — what is left of the docs (plan Q8.6, DC-11, DC-13, DC-15, DC-16, DC's minor list)

Mutant m795, hand-vetted and killed. mAG4's anchor was untouched, and all 1,040 anchors hold.

- **DC-16, F-137's class: a review round's finding cited as the reason.**
  - 55 comment lines cited "(round 4, B-07)" and the like, from the 0.9.0 and 0.19.0 review rounds.
    41 lost the citation mechanically: each sentence already gave the reason, and every changed line
    was read. 14 were rewritten by hand, where the citation carried the reason after a colon or
    across a wrap. One line, broken by the first pass, was caught in the read.
  - `record_test.go`'s doc names the findings that define its test, which is authority. It keeps
    them, with their release.
  - **The class stays gone mechanically.** AP-HIST-01's phrase list gains `round N, X-n`, with a
    specimen each way. Its first run found a capital-R "Round 2 N-1" the hand sweep had missed.
  - **The other narrations DC-16 named,** restated as what holds now:
    - `app.go`'s "read 0.9 for five releases";
    - `KeepOr`'s "held a byte-identical copy";
    - the map key that "waits for W1.16" (it is `defaultMapKeyMap`'s);
    - `scratch_test.go`'s batch number;
    - the held band's "first draft shipped";
    - three "found by a plant that survived" notes, each now the reason the assertion exists;
    - `maintrack.go`'s "live IS GONE".
  - F-137's Go sites are done with this batch.
- **DC-11.** `app`'s package doc names the two packages `cmd/watchpost` imports itself. `ReportOnce`
  loses its build-era scope note. `coverage.go`'s package doc loses its bug story. The `modes/tty`
  package doc is Q9.1's (DV-8).
- **DC-13: the gate documents name today's tests.**
  - `tooling-backlog.md`: `TestEveryControlIsReached`, `TestNoRequiredGatesCIStepIsSilenced` and no
    `makeLines`. `lint.sh`'s closed `|| true` and the deleted `declset_test.go` are gone, and "the four
    ways" becomes a pointer to `gate-attack-list.md`. Both rule lists name all eight authoring rules.
  - `required-gates.txt`'s header is in the present tense, naming `TestTheThreeGateListsAgree`.
  - `red-team-brief.md` says which parts of questions 6 and 9 a tool decides, and that question 5's
    SN-01 has none.
  - The authoring tool's own doc says "most of the rules". Its specimens are
    `06_docs/code-standards.md`'s, not a private harness file. `gate-attack-list.md` drops its stale
    specimen count. F-124 no longer narrates its first version.
- **DC-15: the Settings code describes the window as built.**
  - `setupState`'s doc was the two-question form, fused with a `setupFocus` that does not exist. It
    is now the table, its groups, the tabs and the focus index.
  - `setup_rows.go` loses "twenty rows across four groups", and "tab jumps to the next group".
  - `setupTable`'s doc loses "twenty small structs".
  - `map_settings.go` loses "the map's two Settings".
  - No count is hard-coded in their place.
- **DC's minor list.**
  - accepted-costs names `RecentCap` by symbol. Its check, `TestAcceptedCostsNamesRealSymbols`, now
    resolves any top-level declaration by parsing the file (`declaredIn`), not only functions. m795
    pins it.
  - README:
    - `--verbose` prints the cast and tone answers too;
    - the first run opens on the Data tab, with its HISTORY rows;
    - the AA exemptions are the test's list, not "eleven" (it holds 40, the map's badge grounds
      among them);
    - the console's `space` plays or pauses the bed's relay;
    - `WATCHPOST_DEBUG_MAPFRAMES` (and `_TEXT`), `WATCHPOST_WATCHLIST_DWELL` and
      `WATCHPOST_MAINTRACK` are documented from their code.
  - extending.md: two steps need the network.
  - perf-measurement.md:
    - names `g1b`;
    - lists `answered:layers`, `frames100` and `below-floor` beside the events it had;
    - drops the soak's column count, which is 15;
    - paths `perf-protocol.md`.
  - `Timing.Event`'s comment points there rather than listing events that drift.
  - quality-plan.md:
    - dates the D-9 row's line counts (as of 2026-09-08);
    - paths `build-methodology.md`;
    - says the INST review due at 0.17.0 has no record, which is now F-226.
  - The backlog drops F-156's stale "23".
  - The map as-built's MAP rows follow `map_settings.go`'s order, with "RX noise". The atlas and
    `ids.md` are regenerated.
- **Not changed, and why.** `pkg/schema`'s v1.0.0-rc file is kept deliberately (CHANGELOG). The local
  `watchpost/CLAUDE.md` still names 0.19.0 as the release in flight. It is git-ignored and protected:
  changing it is the HUM LEAD's call.

## 0.19.5 batch 34 — the two big packages have a front door (plan Q9.1, JR-3, JR-4, DC-11, JM-10)

This batch changes comments, plus one test. Mutant m796 is hand-vetted and killed.

- **`modes/tty/doc.go`** says what the package is now:
  - the Router (`router.go`), the program's model, which owns the program's keys and swaps two
    surfaces;
  - the Observer, which is the `Dashboard` type;
  - the Broadcaster console;
  - the import rule as it stands (no domain; go-studs only through render), where JR-3 found
    "reads ONLY platform/snapshot".

  It also maps the package's files by what they own, in twelve rows, and points at
  where-things-happen and the architecture page. The old package comment, "the B3 build of mock
  rev2", is now a file header for `dashboard.go` that says what that file holds.
- **`Dashboard`'s type doc** called it "the root TTY model". It is the Observer surface, one of the
  two the Router holds.
- **`app/doc.go`** gives the composition root's job in plain words and what it builds:
  - the two pipelines;
  - the ticker as the producer;
  - the one deck;
  - the arbiter and MasterControl;
  - the schedule, its executors and the pump;
  - the maps;
  - the history recorder.

  It also maps the package's files in ten rows. `app.go`, `dashboard.go` and `setup.go` keep file
  headers. Those say what each file holds; the build-era labels (B3, M1, "B2 red-team #2") are gone.
  SN-03 holds both packages to one package comment.
- **The maps are held (JM-10).** `TestEveryFileIsInItsPackagesMap` checks that every non-test file in
  each package matches a row's name or prefix, and that every row names a file that exists. A file
  added without a row fails, and so does a row left after its files go. m796 drops a file from the
  map.

## 0.19.5 batch 35 — the words a newcomer meets, defined where they look (plan Q9.2, JR-5, JR-6, JM-15)

Three mutants are re-pointed, each re-vetted and killed. One local is renamed; the rest is docs.

- **The vocabulary.** `docs/where-things-happen.md` gains eighteen rows, each with a code owner in its
  "Defined in" column; the symbol test holds every `file.go:Func`.
  - The five roles, as a row of their own linking 0.16.0's required reading, and then one row each for
    the Producer (`startTicker`, `executors.offer`), the Composer (`composeFor`,
    `compose_takeover.go`), the Reader (`readScript`) and MasterControl (`dip`). The Director already
    had its row.
  - duck, burst, programme, segment, pool, and correspondent and cast.
  - settle, said as two functions with one name: the Director's and the arbiter's.
  - Observer as the `Dashboard` type, the Broadcaster, surface, and Settings as `setup*`.
  - The two report packages, and HUM LEAD.
- **"lane" is marked overloaded** in its own row: the HTTP lanes, a ticker category's band, the
  console's columns, and a gate lane.
- **The width local.** `stationLine`'s `lane := b.bandWidth()` is now `width`: a text width, not a
  lane (JR-5(b)). mAC5, mFR55 and mFR56 anchored on the old name and follow it. They are killed by
  `TestTheBedIsDrawnOnceInsideTheStationSection`, `TestTheBedRidesInTheStationSection` and
  `TestTheGainControlReflows`, among others.
- **README's "Start here"** (JR-6) is a five-step reading order for someone new to the code:
  architecture.md and the atlas, where-things-happen and its vocabulary, extending.md,
  code-standards, and, before the Broadcaster, 0.16.0's required reading. It replaces the two-line
  pointer batch 32 added.

## 0.19.5 batch 36 — "add a Setting", proven step by step (plan Q9.3, JR-7, JM-8, JM-14, JM-15, D-33, A-22)

Mutants m797 to m799, hand-vetted and killed.

- **The proof JM-8 asked for.** A throwaway Setting of each kind went onto a scratch worktree, outside
  the tree: a Watchpost UI on/off, drawn as a two-value picker as Reduce motion is, and a map picker.
  With each complete and green, every step that compiles when forgotten was left out alone, and
  `modes/tty`, `app` and `platform/config` were run whole.
  - **Caught where the walkthrough said:**
    - the app's hand-off and save: `TestASavedSettingIsTheSettingAfterARestart`;
    - the builder line: the two config goldens and the save test;
    - `uiForSave`: the prefs golden and the save test;
    - the table row: `TestEverySettingsRowIsRuledForItsSurface`;
    - the drawing: `TestEveryArrowRowChangesSomething` and the Settings goldens;
    - the arrow branch or case: `TestEveryArrowRowChangesSomething`.
  - **Two steps the walkthrough was missing,** each caught when skipped, and now in it: the new field
    in both `settings_kept_test.go` fixtures, and `uiTouched` at the end of a change.
  - **Three steps that nothing caught,** each answered:
    - **The chip words** (`pickerChipWords`). A picker without them offers "←→ Voice  p Preview".
      Two shipped rows did: the relay replay's rotation and language. They now name their own
      change (A-22), and `TestEveryPickerNamesWhatItsArrowsChange` holds every non-voice picker.
    - **`uiTouched`.** A change that returns without it is drawn, at best, and lost when the window
      closes. The UI throwaway was built that way and passed every test. Now
      `TestAnArrowOnASavedSettingMarksItForSaving` holds every row of the groups Settings saves on
      close, and m798 drops the quake picker's call.
    - **The `d.cfg` mirror in `applyUISaved`.** Nothing reads `d.cfg` for a Setting after launch,
      so skipping it changes nothing anyone can see. The walkthrough says so rather than naming a
      guard that does not exist.
  - **A fourth, found by the map run: `mapPickerRow`.** It was the hand-kept list of map pickers that
    `p` must not preview on, and leaving a row off it failed nothing. The probe that followed found
    `p` previewing the default voice on five shipped rows that are not voices: Theme, the two HISTORY
    pickers, and the relay replay's two. Ruled **D-33: `p` previews only on a voice row.**
    - `TestOnlyAVoiceRowPreviews` was written first and seen failing on exactly those five.
    - The condition is now "a cast row", and `mapPickerRow`, which only it read, is deleted.
    - m799 restores the old condition.
- **Walkthrough 6, rewritten (`docs/extending.md`).**
  - It opens with "Settings, not flags".
  - Its ten steps each name what fails when they are skipped, from the proof, for both kinds: the
    UI's drawing and branch in `setup_ui.go` and `setup.go`, and the map's `mapRow` and
    `mapPrefArrow` case.
  - It says a change ends in `uiTouched`.
- **"When a golden or a declset fails" (JM-14).** Read the diff first. Re-capture with the package's
  flag, given after the package path: the proof met `go test` refusing it before, so each package's
  flag is named. Say why in the build log.
- **Walkthrough 2 (JM-15)** gives a provider its host in `providerEndpoints`, held by
  `TestEveryProviderHasAnEndpoint`.

## 0.19.5 batch 37 — seven essays moved into "Why" lists (plan Q9.4, JR-8, D-34)

This batch changes comments only. mE3 and mC9 follow their lines and are re-vetted.

- **The sample, owed at B2.** The plan put the `stationLine` sample in front of the HUM LEAD during
  B2, and the record holds no sign that it was. It was rewritten and shown now. D-34 approved the
  shape: the rationale moves into the function's doc as a short "Why" list with its IDs, the body
  keeps one-line WHAT comments, and no code changes.
- **The seven,** as JR-8 named them. Each body's lines are given before and after, with the
  in-body comment lines first:

  | Function | Before | After |
  |---|---|---|
  | `Broadcaster.stationLine` | 90 of 125 | 0 of 32 |
  | `Broadcaster.lanes` | 55 of 68 | 0 of 11 |
  | `executors.speak` | 61 of 86 | 0 of 25 |
  | `Broadcaster.clamp` | 31 of 70 | 0 of 39 |
  | `Router.Update` | 46 of 63 | 0 of 17 |
  | `executors.run` | 58 of 96 | 0 of 38 |
  | `startSchedule` | 76 of 135 | 0 of 59 |

  Trailing one-line comments stay at the statements that need them. That includes the contracts a
  maintainer might otherwise "tidy" away: Restore's release through the arbiter for the lock order,
  and the control's send on its own goroutine.
- **The code is unchanged.** A comparison of each file against HEAD, with comments stripped, is
  empty. `startSchedule` differs only in gofmt's field alignment, which the moved comments had been
  breaking.
- **Corrected in the move.**
  - `speak`'s doc said it came home "with nothing" when a read ended early. It comes home a routed
    `Failed` (DR-24), which its own body explained, and the doc now says so.
  - `Router.Update` told Observer the active surface "so Settings can keep one surface's rows from
    the other". Since D-70 every row is drawn on both. What reads it now is Help, About and the map,
    and the Why list says that.
  - `run`'s line citations (`bed.go:392`, `director.go:842`) are symbols now.
  - `stationLine`'s duplicated "a notice is the only prose" paragraph is gone. It was JR-8's drifted
    comment, 90 lines from its code.
- **Mutants.** mE3 (`x.mc.hold()`) and mC9 (the rail's class) anchored on lines that now carry their
  one-line comments, and follow them. mE3 is killed by `TestTheDuckAndItsRestoreReachTheEffector`,
  and mC9 by `TestSpeakReadsThroughTheNarratorAndComesHomeFinished`.

## 0.19.5 batch 38 — the pool lookups name their answers, and what a comment is for (plan Q9.5, JR-9)

mCC2 follows its line and is re-vetted.

- **Named results (JR-9).**
  - `locateInRadius`'s closure returns `(ref, within, found, asked)`, the names
    `Config.LocateInRadius` already gives them. `lookInPool` returns `(ref, found, inPool)`.
  - JR-9's risk is gone: a junior returning `ref, true, true, true` had to find the order in a field
    type 300 lines away, in another package, and a swap of `found` and `asked` compiled.
  - Naming `within` made its `:=` an assignment, and mCC2 follows that line. It is killed by
    `TestARealPlaceOutsideTheRadiusIsNamedButNotWithin`. The lookups' own tests pass unchanged.
- **One line, not two headings.**
  - `Config.LocateInRadius`'s doc had "THREE ANSWERS, NOT TWO" and then "FOUR ANSWERS, NOT THREE".
    It now opens with one sentence naming the four values in order, and gives their reasons in
    sentence case.
  - `locateInRadius` and `lookInPool` say what they answer the same way.
  - `locateInRadius`'s doc also loses "for 0.16.5", since F-157 is re-targeted.
  - Its three caps continuations, in a touched declaration, are sentence case (AP-CAPS-01).
- **What a comment is for (HUM LEAD, 2026-11-08).** Approving the sample, the HUM LEAD restated the
  four rules every comment holds to: it describes the current code; it is not mechanical; it is about
  the problem the code solves; it warns about the gotchas a junior would not see. `06_docs/code-standards.md`
  stated only the first (AP-HIST-01). It now opens with "What a comment is for", giving all four and
  D-34's Why-list shape, and says which rules a tool checks and which are a reader's.

## 0.19.5 batch 39 — building from source, in four steps (plan Q9.6, JR-11)

Markdown only, through the docs lane.

- **README's "Building from source"** was one paragraph holding the toolchain floor, every target and
  the prerequisites. It is now four steps: what you need, the inner loop, before a pull request, and a
  Markdown-only change. The other targets follow.
- **What JR-11 found missing, now said:**
  - how long `make verify` takes. That is measured, not estimated: about 40 minutes, from the last
    eight green runs in `06_docs/verify-runs.md`, 2,342 to 2,481 s. JR-11's 31 to 34 minutes is older,
    from a smaller corpus;
  - that it appends to that tracked file, and that a FAILED line is never removed;
  - that a second run is refused, not queued, and `make tree-free` says so;
  - that `go test ./<package>` is the inner loop, and `modes/tty` is about three and a half minutes
    under `-race` (214 s, batch 36's run);
  - that `make verify-docs` is the lane for a Markdown-only change, at about two and a half minutes
    (146 to 162 s);
  - that macOS needs GNU make's `gnubin` first on the `PATH`, with the line to do it.
- **Kept from the old paragraph:** the toolchain floor and its reason, the vendored go-studs, `hooks`,
  `release-matrix`, `install-test`, `quality-bench`, and `quality`'s P10 gate.
- **No `make help`.** It was optional in JR-11. The four steps name the targets a newcomer needs, and a
  new target would also need an exemption in the gates test.

## 0.19.5 batch 40 — the Settings arrows, one ordered dispatch (plan Q9.7, JR-13, RR-12)

- **Pinned first (RR-12).**
  - `TestEveryArrowRowsOutcomeIsPinned` (`modes/tty/arrow_rows_test.go`, golden
    `testdata/setup-arrows.golden`) was written and recorded before any code moved. For each of the
    43 toggle and picker rows, on ← and on →, it records the flash side, the two save marks and a hash
    of the window as drawn: 86 lines, stable over three runs.
  - RR-12's unrun branches are in it: maps-on, reduce motion, the toggle rows. The `p` preview was
    already pinned by `TestOnlyAVoiceRowPreviews` (batch 36).
  - After the move the golden matched unchanged.
- **The move (JR-13).**
  - `setupRowKey`'s ←→ arm is one call to `cycleFocused(forward) (Dashboard, bool)`. Its doc says
    the order is the dispatch, and why: Reduce motion and the map preferences each have their own
    cycle, and would never be reached after the generic case.
  - The three copies of the flash idiom are one `flashed(forward)`.
  - `cyclePicker` cycles voices only. The relay pickers go to `cycleRelayRow` in `setup_relay.go`
    and the HISTORY pickers to `cycleHistoryRow` in `setup_history.go`, dispatched by row id, not by
    group, so a new row in either group is not cycled as one of them by accident.
  - `setupRowKey` is 53 lines, from 93.
- **Mutants.**
  - m706 and m707 follow their lines into `cycleFocused`. Both are killed by
    `TestEveryArrowRowChangesSomething` and the pin.
  - New: m800, the flash turns the other way, is killed by the pin alone. m801, Theme falls to the
    voice cycle, is killed by both.
- **Docs.** extending.md step 9 names `cycleFocused`, its order and the pin's golden.
  where-things-happen says `cyclePicker` moves a voice picker.

## 0.19.5 batch 41 — words for the one-letter fields, the release stamps gone, JR's minors (plan Q9.8, Q9.9)

- **Q9.8, the all-caps rule, verified.**
  - `go run ./tools/authoring -base <merge-base with main>` reports no AP-CAPS-01 finding, so every
    declaration the release changed is clean.
  - This batch's own edits touched eight more declarations with an "AND …" continuation. Each now
    says in sentence case what it adds.
- **The one-letter fields (A-23).**
  - The renames are `livePipelines.p` and `radioDeck.p` → `program`, `arbiter.v` and
    `mastercontrol.v` → `voice`, `speaker.d` → `arbiter`, `reader.s` → `speaker`, `reader.h` →
    `hooks`, and `schedule.x` → `executors`.
  - They were renamed by type with gopls (v0.20.0, installed outside the tree), so every use site
    follows and no comment or string changed.
  - Eight mutants follow the new names: m22, m48, m49, mE0, mE1, mG2, mG3, mH2. Each was applied,
    vetted and killed against the whole `app` package, then restored by copy.
- **The release stamps.**
  - 54 "0.x.y:" stamps on 53 lines were removed from Go comments: `// 0.x.y: `, `; 0.x.y: ` and
    `(0.x.y: `. JR counted 59 with a looser pattern that also caught IP addresses.
  - The script asserted its own count, and the assertion caught the `(` form it had been told to
    expect at 52.
  - Three restated lines read wrong without their stamp and were rewritten: theme.go's lane tokens, a
    "; and" join, and cast.go's zero value ("is 0.13.0:" now "is one voice").
  - mAG1's anchor carried one, and follows.
- **JR's minor list, the rest of it.**
  - "ruling 2" and "ruling 3" name their source: 0.16.0's `report-requests-plan.md` rows 2 and 3.
  - "HUM LEAD ruling 5", beside NFR-7 in executors.go, resolves to no recorded ruling, so the comment
    cites 0.16.0 NFR-7 alone. Nothing is lost: the reason is in the sentence.
  - schedule.go's `executors` doc no longer quotes pool.go's comment; it says the remedy and names
    `livePipelines.producer`.
  - `recentOff` describes the window it keeps, not the work it once waited on.
  - setup.go names setupLines' file.
  - setup_rows.go's 20-line note on why `setupScope` has no sentinel is now five lines: the reason
    and the test.
  - The ten files over 500 lines that had no header now have one, so where-things-happen's "the
    per-file headers say what a file holds" is true. The ten: app/radio.go, app/cast.go,
    platform/lineup/director.go and plan.go, platform/httpx/cache.go,
    domains/radio/player/engine.go, domains/radio/synth/source.go, domains/temperature/sources.go,
    platform/snapshot/assembler.go and modes/tty/map_prop_chart.go. Each header's claims were checked
    against its declarations, and three were corrected before the gate.
  - "Option C" in app.go was already gone.

## 0.19.5 batch 42 — the M5 answer key, and what it found (D-24, D-35, D-36)

- **The key (D-24).**
  - Four tasks were done on scratch copies of batch 41, one site left out at a time, each omission
    run against the touched packages whole. The tasks were a toggle Setting, a keyless provider, a
    line on a console card and a spoken phrase.
  - The keys are in `08-reports/m5-key/`, which the blind reviewer may not read. They give each
    task's sites, the guard that fails when one is missed (or NO GUARD), the docs' wrong routes and
    the IDs that did not resolve.
  - Site counts: the toggle 32, the provider 47 (16 NO GUARD), the card line 7 plus 3 sub-sites,
    the phrase 9.
- **IDs (D-35).**
  - The multi-voice release's rulings were never tabled. 85 MVS-D IDs are cited and none resolved.
  - `tools/ids` now indexes each by its best mention, in this rank order: its own file's heading;
    a bold statement of what it ruled in the multi-voice docs (the project brief's Rulings row and
    amendment table give 1 to 46); one elsewhere; any line in the multi-voice docs; any line elsewhere.
  - A bold ID inside a citation is a mention, not a statement.
  - `TestAnMVSRulingIsIndexedByItsBestMention` pins the order (m802), and
    `TestEveryMVSRulingTheCodeCitesIsIndexed` holds every MVS-D in a Go file to a row (m803).
  - The other IDs the M5 inputs cite that did not resolve are restated:
    - F-111, whose row batch 29 retired, now names its guard, `TestEveryReportKindReachesTheComposer`.
    - "0.15.0 FR-7.1": 0.15.0 has no requirements file in the tree. It is now MVS-D-52 and the HUM
      LEAD's 2026-09-08 ruling.
    - R3-A-01 now names its file.
  - A script over the M5 inputs (README, `docs/`, code-standards) now finds no ID the index lacks.
- **The provider's silent sites, two of them now guarded.**
  - `domainFor` decides whether any tier asks a provider, and a kind with no case was fetched
    nowhere, silently. `TestEveryFetchKindHasADomain` (platform/sched) reads the kinds from
    snapshot's const block and fails one with no domain and no reason in `onDemand` (m804). In
    `platform/sched` alone, m804 is caught only by it.
  - The live pipelines, the one-shot report and `TestEveryProviderHasAnEndpoint` each built the
    provider families by hand. The test claimed "the PRODUCTION construction" and was not using it.
    They are now one constructor, `newProviderFamilies`, with `providerFamilies.all` the one order,
    embedded in `livePipelines` so `lp.marine` and the rest still read.
  - `TestAllAsksEveryFamily` reads the struct's fields by reflection and fails a family `all` leaves
    out (m805).
  - Eight test literals name the embedded struct, because go.mod's 1.26 refuses a promoted field in
    a literal.
  - `app`'s declaration set gains exactly `type providerFamilies`, `func newProviderFamilies` and
    `func (providerFamilies).all`.
  - The rest of the provider's silent sites are routed, not guarded: `reportFetch`'s kinds, the
    attribution case, `ChipGrounds`, the assembler's four sites, and `detailLines`. Walkthrough 2's
    steps 3, 4, 4a and 4b now name every one, and say plainly where nothing fails.
- **Reduce motion's chips (D-36).**
  - Its row was not a picker, so the chip row never said what its arrows change, and its chip words
    were never read. It is now `picker: true`, and the chip row says "[←→] Motion".
  - Two goldens moved: `setup-arrows` (row 10's two frames, the same flash and save marks) and
    `router-trace` (the six frames with the focus on Reduce motion).
  - New: `TestEveryChipWordIsRead` fails a word for a row that is not a picker, and
    `TestReduceMotionsChipsNameIt` reads the drawn chip row. Both are killed by m806.
- **The docs.**
  - extending.md:
    - The intro no longer says "one folder plus one wiring line".
    - Walkthrough 2 is complete, from the provider key.
    - Walkthrough 4's script step no longer says no gate reads the tree.
    - Walkthrough 6 is corrected from the toggle key: `rowToggle` is the tone rows' kind;
      `ttyConfig` and `setUIHook` are named; the real guard on the builder line; focus order and
      the `setupTable` P10 limit; `uiLines`, `at` and `uiLabelW`; `.flashed`; two lines per
      settings-kept fixture; the five goldens.
    - New Walkthrough 7, add a spoken phrase, from the phrase key.
    - The golden flag before the path is shown as what it is: `go test` reads the path as the
      flag's value and fails with "no Go files", not a refusal.
  - where-things-happen:
    - `Pronounce` runs in `Source.say`, not the library's `Say`.
    - `Compose` asks for the cycle's phrases.
    - `domainFor` decides what a tier fetches.
    - `mergeLocationLocked` stores the state.
    - The air box and `lanes` are each placed in their file.
    - A row for the UP NEXT card's body (`upNextBox` → `readBody`).
  - architecture.md names the five roles and the spoken-phrase walkthrough. README's "Start here"
    lists every walkthrough.
  - code-standards' first sentence names the out-of-tree P10 checker.
  - script.go's package doc no longer says "the tree IS the wiring" or that adding a phrase is adding
    a file.
  - The 0.16.0 required reading puts `cardBoxWidth` in `broadcaster_rail.go`, and says F-34 was
    answered.
  - The watchpost-cli design's stale cadence copy is a pointer to the generated table, plus the
    notes it does not carry.
- **The doc-symbol check** now indexes file names under `testdata/` (their Go is not parsed), so a
  doc can name a golden by its path.
- **The console's `--ascii` sweep** now draws the floor width too, with a long second label. The card
  key's "the console is missing from the scan" was only half right: the console has its own sweep,
  which drew only the wide layout. A probe showed the long label is not cut with an ellipsis at the
  floor, so the sweep's comment claims only what it covers.

## 0.19.5 batch 43 — the keys stop defining the IDs they report (docs lane)

- **The problem.** `tools/ids` reads a table row whose first cell is an ID, and nothing else, as that
  ID's definition. The card and phrase keys each list the IDs that did not resolve in a table led by
  the ID. So the index gained rows for F-111, F-164 and F-34 that point at a report saying they have
  no row.
- **Found how.** The blind reviewer's copy was checked for any path into `m5-key/`, and `ids.md` had
  four.
- **The fix.** Those cells read "F-111 (cited)" and so on, and the index has no `m5-key/` row.
- **The check.** The M5-input ID scan was re-run on the corrected index and still finds none
  unresolved. F-111 is no longer cited there: Walkthrough 4 names its guard instead.

## 0.19.5 batch 44 — the blind M5 review, and its fixes (D-24, D-37)

- **The review.** A blind reviewer with D-24's persona traced the four tasks from README's "Start
  here". It worked on a copy of batch 43 with this release's folder deleted, reading only what the
  docs name. It is graded against the keys in `08-reports/m5-blind-review.md`.
  - T1, T2 and T4 pass, with notes.
  - **T3, a line on a console card, fails.** Seven of its ten sites had neither a route from the
    docs nor a guard: the words, the width, the ellipsis and the goldens' test. The card's 0.16.0
    rulings resolved to other releases' rows.
- **T3's fix.**
  - extending.md gains Walkthrough 8, add a line to a Broadcaster card, from the card key. It covers:
    - `readBody` and the card's fixed height;
    - the words through `reportTypeOf`;
    - the width `lane.inner()-len(bcCardInset)`, and the glyph set's ellipsis;
    - `TestConsoleGoldens` and its 12 goldens;
    - the line's own tests;
    - `bcFrameAllocs` and `make alloc-budget`.
  - It says that whether a line is drawing or what the card says is not ruled, so ask.
  - A second blind trace of T3 alone, after this batch's gate, checks the fix routes.
- **D-37, 0.16.0's untabled rulings.**
  - The HUM LEAD was first asked with "extend D-35 and read git" recommended, and answered so. The
    agent then found the commit-only rulings are on the pre-squash feature branches, not in main's
    history, so the hosted run cannot see them, and asked again with that corrected. The second
    answer, "Mentions only", stands.
  - `tools/ids` indexes each D-ID that 0.16.0's docs mention and its tables do not define, at its
    best mention there, marked "Mentioned:": 35 rows. A mention another release qualifies is that
    release's (`TestAnUntabledRulingIsIndexedByItsMention`, m807).
  - The 13 commit-only rulings (D-85 to D-126) are restated without the ID. That is 78 comment lines,
    each attributed to its release by the commit that wrote it: 72 were 0.16.0's, and 6 were console
    lines rewritten this release. The map and propagation code's citations of 0.18.0 and 0.19.0
    rulings with the same numbers are left alone.
  - Every restated sentence already said what was ruled, so the tag was all that came off. One line
    that began with its tag (`// (D-103). The reference …`) needed its full stop moved by hand.
  - The 24 "AND …" continuations in the declarations this touched are sentence case (AP-CAPS-01).
- **ids.md reads two more forms.**
  - A first cell that carries a priority ("FR-3.8 (P1)") defines its ID. 0.18.0's requirements now
    resolve to what they require, not to their traceability rows (m809).
  - A row's subject is its first cell with a word in it, not a cell of references only ("W1.13,
    W3.7"), m808.
  - Both are in `TestAPriorityStillDefinesAndAReferenceIsNoSubject`. 207 lines of the index changed.
- **The other routes.**
  - Walkthrough 2:
    - `defaultTheme` is in `theme.go`, and Monochrome is an entry of `builtinOverrides`;
    - seismic models neither the attribution case nor the report line, so copy fire's;
    - a new block rides an `-rc` schema version, and moving it is the HUM LEAD's call;
    - a station provider joins no host list but `providerEndpoints`;
    - the merge test's model is named.
  - Walkthrough 6 gains three notes: a row's surface is ruled; a Setting both surfaces read goes
    through `Router.Update`; a memoised region's Setting belongs in its memo's key.
  - Walkthrough 7 names `TestTheAppsPartsExist`'s file, and notes that `Compose` also composes every
    console card.
  - extending.md's intro says how to run a named test alone.
  - where-things-happen's Qn row says a release-qualified Q is that release's plan task.
  - code-standards' `setupTable` count says it is the body's.
  - The 0.16.0 required reading's two quotation sources are current: the duck reasoning now in
    `platform/lineup/duck_test.go`, and `app/arbiter.go`'s header. `duck_test.go` no longer credits
    radio.go.
  - The cadence test's failure asks for the freshness reason in the build log, not in a plan section
    no reader can find.
