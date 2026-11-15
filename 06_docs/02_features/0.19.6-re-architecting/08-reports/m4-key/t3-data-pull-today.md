# Answer key, T3: the UP NEXT card's DATA PULL stamp leaves out today's date

> **0.19.6 M4 answer key (D-24's method, JN P2-8), hidden from the blind reviewer.** Built on a scratch
> copy of `e57afeaf` (`git archive`, one commit `base`) by doing the task and leaving out one site at a
> time. Every run is in `<scratch>/plan-KB/runs.log`, its output in `<scratch>/plan-KB/logs/<label>.log`.
> Toolchain go1.27.4, `GOTOOLCHAIN=local`, no network. Lines "in the copy" are the finished change's.

Scratch copy: `<scratch>/plan-KB/t3` (base `6b939d2`). The complete change: `<scratch>/plan-KB/t3-complete.diff`;
every touched file: `<scratch>/plan-KB/t3-complete/`.

## The task, as given to the reviewer

> On the Broadcaster console, the UP NEXT card's DATA PULL line shows the date and the time of the pull,
> for example `DATA PULL: 09/12/26  16:02:45  (2 MIN AGO)`. When the pull was made today, show only the
> time and the age (`DATA PULL: 16:02:45  (2 MIN AGO)`); keep the date when it was made on an earlier
> day. The DATA PULLED line in the card's detail window is not part of this change.

## The change that was made

`cardPulledShort` writes the date only when the console's clock says the pull was made on an earlier
calendar day; with no clock (`now == nil`) it writes the date as before. "Today" is the pull's own
calendar day, read in the pull's own zone (`now().In(c.BuiltAt.Location())`), because the stamp is
written in that zone. A new test pins the three cases and the no-clock case. A CHANGELOG entry.

Why this instance: it is a change to what an existing console card line says, not a new line (0.19.5's
instance), so Walkthrough 8 routes it only in part; the card lives in the console package after 0.19.6.

Measured facts it depends on:
- **One caller.** `cardPulledShort` is called only from `readBody` (`modes/tty/broadcaster_slots.go:154`
  at `e57afeaf`); the detail window uses the long form, `cardPulled` (`broadcaster_detail.go:136`). So
  only the UP NEXT card changes (grep).
- **No golden holds a dated stamp.** All 12 console goldens draw the card with no `BuiltAt`, so their row
  is `DATA PULL: —` (for example `modes/tty/testdata/console-133x44-standby-plain.golden:19`). The
  finished change moved no golden: `go test ./modes/tty` passed with no re-recording (`t3-complete`).
- **The existing pins stay green.** `TestTheCardStampsFollowTheClock`
  (`modes/tty/broadcaster_manifest_test.go:214-233`) calls with no clock, so the date stays;
  `TestTheCardPaintsTheAgeWithItsRung` (`broadcaster_dataage_test.go:98-113`) checks only the age.
- **The width only shrinks.** The today form is 10 cells shorter (`09/12/26  `), so nothing new is cut
  at the console's floor (`bcMinCols`, 100 columns).
- **The card is not memoised.** The console memo covers the two tables only (`broadcaster_memo.go:3-17`);
  the UP NEXT box is drawn every frame (`broadcaster.go:810`), so the date drops and returns with the
  frames that already move the age. No memo key changes.

## Every site the change needed

| # | File | Symbol | What changes | Line in the copy |
|---|---|---|---|---|
| S1 | `modes/tty/broadcaster_manifest.go` | `cardPulledShort` | after the `now == nil` return: when `now()` in the pull's zone is the pull's calendar day, the stamp is `o.Clock.TimeSec(c.BuiltAt)` alone; the age is computed from the same `now()` | 140-144 |
| S2 | same | `cardPulledShort`'s doc comment | the two forms, the no-clock case, and why (it named only the dated form "from the reference") | 115-123 |
| S3 | `modes/tty/broadcaster_manifest_test.go` | `TestTheCardsStampLeavesOutTodaysDate` (new) | same day, next day, same day read in another zone, and no clock | 238-260 |
| S4 | `CHANGELOG.md` | a new `## [Unreleased]` / `### Changed` entry | the visible change (the heading does not exist at `e57afeaf`) | 3-9 |

**Counts: 4 sites, in 2 source files and the CHANGELOG; 0 goldens. NO GUARD: 3 (S2, S3, S4), and S1 has
none against the tests at `e57afeaf`.**

## Leave-one-out

Each row: that one site undone in the finished copy, `go test -count=1 ./modes/tty` whole, the file put
back from `t3-complete/` and the tree compared with the complete diff (`RESTORED OK` each time).

| Site left out | What fails (package `modes/tty`) | Run |
|---|---|---|
| S1 | `TestTheCardsStampLeavesOutTodaysDate` only: `the same day: the card's stamp is "09/12/26  16:02:45  (7 MIN AGO)"; want it to start "16:02:45  (7 MIN AGO)"`. **Against the tests at `e57afeaf`, NO GUARD**: without S3 nothing pins the new form (next row) | `t3-omit-S1` |
| S2 | **NO GUARD**: a comment; no test reads it. `make lint-authoring` judges comments and was not run (forbidden here) | none |
| S3 | **NO GUARD**: the whole package passes with S1 in and the new test out | `t3-omit-S3-test` |
| S4 | **NO GUARD**: no test reads `CHANGELOG.md` (grep) | none |

Variant, run alone: comparing `now().Date()` without `In(c.BuiltAt.Location())` fails only the new
test's "another zone" case (`t3-variant-rawzone`). In the app both are local time, so the variant is
harmless in production; the case is there so the rule the comment states is held.

The complete change: `go test -count=1 ./modes/tty` **ok**, no golden re-recorded (`t3-complete`).

## The docs that route a junior to each site today (`e57afeaf`)

| Site | Route | Gap |
|---|---|---|
| S1 | `docs/where-things-happen.md:75` ("The UP NEXT card is drawn"): `upNextBox` → `readBody`, "its rows: the data pull and the rest". `readBody`'s DATA PULL row calls `cardPulledShort` (`broadcaster_slots.go:154`). Walkthrough 8 step 2 (`docs/extending.md:237`) names `cardPulledShort` as the helper a new line sits beside | Routed. Walkthrough 8 is "add a line"; a change to an existing line has no walkthrough of its own, but these two lines reach the symbol |
| S2 | None (the comment beside S1) | NONE needed |
| S3 | Walkthrough 8 step 5 (`docs/extending.md:240`): a line's own tests in `broadcaster_upnext_test.go`, with "**Without them only the goldens fail**". For this change the goldens cannot fail (see above), so without S3 nothing fails | Routed, but its safety net is absent here |
| S4 | Walkthrough 8 step 7 (`docs/extending.md:242`): "An entry under `[Unreleased]`" | Routed to a heading that does not exist (finding 1) |

## Found along the way (at `e57afeaf`)

1. **`[Unreleased]` does not exist.** `docs/extending.md:242` says to write the entry "under
   `[Unreleased]`"; `CHANGELOG.md:5` begins at `## [0.19.5] — 2026-11-11`. A junior must create the
   heading, and nothing says in what form.
2. **No golden holds the card's stamp.** Walkthrough 8 step 4 (`docs/extending.md:239`) says to re-record
   the 12 console goldens and "read each diff first: the new line, and nothing else moving". Every console
   golden draws a card with no `BuiltAt` (`DATA PULL: —`), so no change to the stamp's words can ever move
   a golden. The step's safety claim does not hold for this row.
3. **The reference draws the date.** `cardPulledShort`'s comment says its form is "from the reference"
   (`broadcaster_manifest.go:115-117`); the v3 mock draws `DATA PULL: mm/dd/yy  23:59:59  (10m ago)`
   (`06_docs/02_features/0.16.0-broadcaster-ui/01-objectives/mock-broadcaster-v3.txt:40`). Leaving out
   the date departs from the reference: a choice to ask about before building, which Walkthrough 8's
   closing paragraph (`docs/extending.md:244-247`) points at in general terms only.
4. **A wall-clock fixture.** `broadcaster_upnext_test.go:65` sets `BuiltAt` from `time.Now()`. After
   this change that card's stamp drops its date except in the ten minutes after midnight. The test finds
   the row by its label, so it stays green, but its frame depends on when it runs.
5. **"Today" has two readings.** `render.Clock.Since` (`platform/render/clock.go:124-131`) compares
   `t.Date()` with `now.Date()` each in its own zone; the task needs one zone. The key reads both in the
   pull's zone and says so in the comment. No doc says which rule the console follows.

## After 0.19.6

S1 to S3 move with `modes/tty/console` (package map §2); S4 does not move. `where-things-happen.md:75`
and Walkthrough 8's file names must be re-pointed in the batch that moves the console.
