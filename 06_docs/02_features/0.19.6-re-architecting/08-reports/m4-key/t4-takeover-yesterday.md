# Answer key, T4: a takeover says "yesterday" for an alert declared the day before

> **0.19.6 M4 answer key (D-24's method, JN P2-8), hidden from the blind reviewer.** Built on a scratch
> copy of `e57afeaf` (`git archive`, one commit `base`) by doing the task and leaving out one site at a
> time. Every run is in `<scratch>/plan-KB/runs.log`, its output in `<scratch>/plan-KB/logs/<label>.log`.
> Toolchain go1.27.4, `GOTOOLCHAIN=local`, no network. Lines "in the copy" are the finished change's.

Scratch copy: `<scratch>/plan-KB/t4` (base `cd2de1e`). The complete change: `<scratch>/plan-KB/t4-complete.diff`;
every touched file: `<scratch>/plan-KB/t4-complete/`.

## The task, as given to the reviewer

> When the station reads an alert aloud in a breaking takeover, alone or in a burst, it says when the
> alert was declared: `at Four Fifty PM` for today, and `on November 13 at Eleven Fifty PM` for an
> earlier day. Make it say `yesterday at Eleven Fifty PM` when the alert was declared the day before;
> keep the date for anything older.

## The change that was made

`spokenWhen` gains a branch between "today" and "any other day": when the event's local calendar date is
the day before `now`'s, it says `"yesterday at " + c.Spoken(local)`. The day before is taken with
`now.Local().AddDate(0, 0, -1)`, a calendar day, not 24 hours back. Its doc comment and the Composer's
midnight note are restated; a new test file; a CHANGELOG entry.

Why this instance: the words are composed in Go in the deck (`app/burst_words.go`, which the package map
puts in `app/internal/deck`), not in a script file, so it is not 0.19.5's instance (a new script part in
`domains/radio/synth`) and Walkthrough 7's "re-wording is a text edit" does not cover it.

Measured facts it depends on:
- **Two callers, both takeover lines.** `spokenWhen` is called by `burstTitle` (a burst's line,
  `app/burst_words.go:67` at `e57afeaf`) and `eventNarration` (`:163`), which `alertNarration` (a single
  event's line, `:173`) wraps; `breakingLine` (`:103`) chooses between them. The `[space]` event read
  says no dates by design (`app/severe_read.go:280-286`, "No dates or clock times" at `:284`), and the ticker tape does not speak
  (`tapeHead`, `:146`). So only takeovers change (grep).
- **No test reaches the date branch at `e57afeaf`.** `TestNarrationLinesSingleBurstAndClosing`
  (`app/ticker_test.go:308-333`) uses a same-day event; `TestNarrationPointsAtTheWindow`
  (`app/severe_test.go:138`) uses an event a day *after* `sameDay` and checks only a prefix and a
  suffix. No `burst_words_test.go` exists. The finished change kept the whole `app` suite green with
  no edit to an existing test (`t4-complete-final`).
- **"Yesterday" must be the calendar's.** A first test without the 46-hours case passed a
  `now.Sub(t) < 24*time.Hour` implementation too; the case `01:00` the day before, heard at `23:00`,
  separates them (`t4-variant-24h`).
- **Composed once, up front.** A takeover's words are composed before the card airs
  (`app/compose_takeover.go:15-22`), so "yesterday" is judged at compose time; the file's midnight note
  describes exactly that case and its example wording changes.

## Every site the change needed

| # | File | Symbol | What changes | Line in the copy |
|---|---|---|---|---|
| S1 | `app/burst_words.go` | `spokenWhen` | the "yesterday" branch, after the today return and before the dated one | 207-211 |
| S2 | same | `spokenWhen`'s doc comment | names the three readings (it named two) | 189-191 |
| S3 | `app/compose_takeover.go` | the file's header, the midnight note | its example: a line composed at 23:59 and read at 00:01 now says "at 11:59 PM" rather than "yesterday at 11:59 PM" (it said "on September 4 at 11:59 PM") | 21 |
| S4 | `app/burst_words_test.go` (new file) | `TestSpokenWhenSaysYesterday` | today, yesterday, 46 hours back, two days back, yesterday across a month and a year; and the single-event and burst lines carrying it | 14-44 |
| S5 | `CHANGELOG.md` | a new `## [Unreleased]` / `### Changed` entry | the spoken change (the heading does not exist at `e57afeaf`) | 3-9 |

**Counts: 5 sites, in 3 source files and the CHANGELOG; 0 goldens. NO GUARD: 4 (S2, S3, S4, S5), and S1
has none against the tests at `e57afeaf`.**

## Leave-one-out

Each row: that one site undone in the finished copy, `go test -count=1 ./app` whole, the file put back
from `t4-complete/` and the tree compared with the complete diff (`RESTORED OK` each time).

| Site left out | What fails (package `app`) | Run |
|---|---|---|
| S1 | `TestSpokenWhenSaysYesterday` only: `yesterday: "on November 13 at Eleven Fifty PM", want "yesterday at Eleven Fifty PM"` and the single-event and burst lines likewise. **Against the tests at `e57afeaf`, NO GUARD** (next rows) | `t4-omit-S1` |
| S2 | **NO GUARD**: a comment. `make lint-authoring` not run (forbidden here) | none |
| S3 | **NO GUARD**: a comment in another file; nothing routes to it but a search for `spokenWhen` | none |
| S4 | **NO GUARD**: the whole package passes with S1 in and no new test | `t4-omit-S3-test` (run label from the first numbering; it is this row) |
| S5 | **NO GUARD**: no test reads `CHANGELOG.md` (grep) | none |

The complete change: `go test -count=1 ./app` **ok**, 135 s (`t4-complete-final`), with the test's
final six cases and its two line checks.

## The docs that route a junior to each site today (`e57afeaf`)

| Site | Route | Gap |
|---|---|---|
| S1 | `docs/where-things-happen.md:18` ("A report's wording is chosen"): "the takeover its lines in `app/burst_words.go:breakingLine`". Walkthrough 7 step 2 (`docs/extending.md:217`) names `breakingLine` "for a takeover". From `breakingLine` (`:103`) to `alertNarration`/`eventNarration`/`burstTitle` to `spokenWhen`, by reading code in the one file. `app/doc.go:29`: "`burst_words.go` … the words and Composer for a takeover". The vocabulary row "burst" (`where-things-happen.md:122`) names the same file | Routed to the file; the last hop is code |
| S2 | beside S1 | NONE needed |
| S3 | None. Only a search for `spokenWhen` finds the note in `compose_takeover.go:17-22`; the vocabulary row (`where-things-happen.md:122`) names the file, not the note | **NONE** |
| S4 | Walkthrough 7 step 4 (`docs/extending.md:219`) says to write a test of a new segment; for a Go-worded phrase nothing names a test file. No `burst_words_test.go` exists to extend | **NONE** for this shape |
| S5 | Walkthrough 8 step 7 (`docs/extending.md:242`), the console's walkthrough; Walkthrough 7 has no CHANGELOG step | NONE for this task |

## Found along the way (at `e57afeaf`)

1. **An orphaned doc comment.** `app/burst_words.go:74-75`, "breakingLine is the line to speak for one
   event…", is separated from `breakingLine` (`:103`) by `testHead` and `testLine` and a blank line, so it
   binds to nothing and `go doc -u` shows `breakingLine` with no comment. The symbol the flow map routes
   to (`where-things-happen.md:18`) is the one with no doc.
2. **Walkthrough 7 assumes every spoken phrase is a script file.** Its intro
   (`docs/extending.md:206-209`) says re-wording is a text edit and "a test that pins the old wording
   fails". The takeover's time words are Go (`spokenWhen`, `burstTitle`, `eventNarration`), are in no
   script file, and no test pins the dated wording. README's list "What the correspondent says"
   (`README.md:539-552`) lists script parts only.
3. **The date is not in the Radio Convention.** `spokenWhen` writes the date as `local.Format("January
   2")` whatever the clock Setting (`burst_words.go:206`); only the time follows it (`c.Spoken`). Under
   the military convention the time is said in that convention and the date still as "November 13".
   Unruled; noted, not changed.
4. **The Composer's own caveat is the edge of this change.** `compose_takeover.go:15-22` already says a
   line composed before midnight and read after it speaks the stale day word. With "yesterday" the stale
   word is shorter but the same defect; a junior may think the change should fix it. It should not:
   the composition is up front by ruling (that header, MVS-D-77, S-7).

## After 0.19.6

S1 to S4 move to `app/internal/deck` with the deck (package map §2, AS Map I: `burst_words.go` and
`compose_takeover.go` are in its list). `where-things-happen.md:18`, `:122`, Walkthrough 7 step 2 and
`app/doc.go:29` must be re-pointed in the batch that moves the deck.
