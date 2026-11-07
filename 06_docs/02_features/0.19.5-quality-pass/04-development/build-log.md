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

