# Answer key: T3, add one line to a Broadcaster card

> **0.19.5 M5 answer key (D-24), hidden from the blind reviewer.** Built on a scratch copy of the
> tree at batch 41 (`e25a9ea9`) by doing the task and leaving out one site at a time. Batch 42 acted on
> it: the doc findings below are fixed, and the guards it added are named in the build log
> (`TestEveryFetchKindHasADomain`, `TestAllAsksEveryFamily`, `TestEveryChipWordIsRead`,
> `TestReduceMotionsChipsNameIt`). The site lists stand as the key; a row's "NO GUARD" is as measured
> at batch 41. Scratch paths are written as `<scratch>/`.


Scratch copy: `scratchpad/q195/m5key/t3-cardline` (git, one commit `base`). Everything below was
checked against that copy. Toolchain go1.27.3 with `GOTOOLCHAIN=local`.

## The change that was made

The **UP NEXT card** (the console's read card, `modes/tty/broadcaster_upnext.go:upNextBox` →
`modes/tty/broadcaster_slots.go:readBody`) gets a `REPORTS:` line under `DATA PULL:`. It shows
which reports the card carries, in the running order's own words: `reportTypeOf(c)`
(`modes/tty/broadcaster_lineup.go:120`), which is `c.Reports.Describe()` ("NWS, FIRE",
"Location Report, Full") and falls back to the slot's title-cased name ("Location Report") when
the set is empty. An empty set means "the whole report" (`platform/lineup/card.go`, `Card.Reports`).

Why this line and this card:
- The card's **age is already drawn** (`DATA PULL: mm/dd/yy hh:mm:ss (N MIN AGO)`,
  `cardPulledShort`), so an "age" line would repeat it. Report kinds are new information here.
  The card's border shows only the slot ("LOCATION REPORT"), and the running-order table starts
  at slot 02, so slot 1's kinds were not drawn anywhere on the base frame.
- **LIVE NOW is a row, not a card** (`broadcaster_air.go` header, D-95): it sits in the air box
  beside RELAY BED. The card that has lines is UP NEXT.
- The value is already on the card (`Card.Reports`), so **no Producer, Composer, Director,
  Reader or MasterControl code changes**, and nothing in `platform/` or `app/`.

Measured facts the change depends on:
- **The card height stays the same.** `readBody` pads its rows to `bcReadCardRows-1` (13-1) and
  then adds the controls row. Before the change, 2 of those rows were padding. The new line
  takes one, so the frame keeps its height. One blank row is left: a second new line still
  fits, and a third makes the card one row taller (tested, see the table).
- **The narrow terminal.** The console refuses to draw below 100 columns (`bcMinCols`). At 100
  columns the UP NEXT card cell is 36 cells wide. The longest value the registry has,
  `REPORTS: Location Report, Full`, is 30 cells, plus a 3-cell inset. The manifest's own `room`
  (`lane.inner()-2*len(bcCardInset)`) is 28 at that width, which is too narrow for this line.
  The line is therefore held to `lane.inner()-len(bcCardInset)`, which is 31.
- **Truncation and `--ascii`.** The box truncates every row with `render.TruncateCells`, which
  adds no ellipsis. If a line is cut, `cardReports` adds `o.Glyphs().Ellipsis` itself: "…"
  normally and "..." under `--ascii`. The console's `--ascii` scan
  (`TestTheConsoleCarriesNoNonASCIIUnderASCII`) never draws a truncated line, so it cannot see
  a hard-coded "…".
- **The allocation budget.** The console hit-frame went from 291 to 300 allocs against a budget
  of 309 (`bcFrameAllocs`). The miss-frame went from 9164 to 9173 against 9655. No re-pin was
  needed, but the hit path now has 9 allocs of headroom where it had 18.

## 1. Every site the change needed

| # | File | Symbol | What changes there |
|---|---|---|---|
| S1 | `modes/tty/broadcaster_slots.go` | `Broadcaster.readBody` (the `rows` literal, line 161) | One element after the DATA PULL row: `bcCardInset + cardReports(o, c, lane.inner()-len(bcCardInset))`. |
| S2 | `modes/tty/broadcaster_manifest.go` | new `cardReports(o render.Opts, c lineup.Card, room int) string` (line 149) | Returns `"REPORTS: " + reportTypeOf(c)`, cut to `room` cells with the glyph set's ellipsis when it is too long. The guard's condition is call-free (`w := render.Width(line); if w <= room`) so that P10-05 counts it. |
| S2a | same function | its words: `reportTypeOf(c)`, not `c.Reports.Describe()` | `Describe()` returns "" for an empty set, which is every card the Director makes. That would draw a bare `REPORTS:`. |
| S2b | same function | the ellipsis: `o.Glyphs().Ellipsis` | Not a literal "…", and not a bare `TruncateCells` cut. |
| S3 | `modes/tty/broadcaster_slots.go` (the call in S1) | the `room` argument | Must be `lane.inner()-len(bcCardInset)`. The function's local `room` (the manifest's width) cuts the full label at 100 columns. |
| S4 | `modes/tty/testdata/console-{100x44,133x44,160x74}-{standby,onair}-{plain,colour}.golden` (12 files) | `TestConsoleGoldens` | Re-record them with `go test ./modes/tty -run TestConsoleGoldens -update-golden` (flag AFTER the path; `-args -update-golden` also works). Each diff: rows 20-23 of the card. The REPORTS line replaces the blank row under DATA PULL, and the blank row, caption and heading each move down one. No other row in the frame moves. |
| S5 | `modes/tty/broadcaster_upnext_test.go` | new `TestTheUpNextCardNamesTheReportsItCarries`, `TestTheReportsLineEndsInTheGlyphSetsEllipsis`, `TestTheLongestReportsLineFitsAtTheFloor`, plus the `platform/report` import | They pin three things. The words for an empty set, a subset and the full set. The order (stamp, then REPORTS, then caption). The card's fixed height (`bcReadCardRows+2`). Also the ellipsis under `--ascii` and normally, and that the longest set fits uncut at `bcMinCols`. |
| S6 | `modes/tty/broadcaster_slots.go` | `readBody`'s comment above `rows` (lines 152-156) | Says that the REPORTS line is not in the D-110 reference and that it uses one of the fixed height's blank rows. Also how many rows are left. |
| S7 | `CHANGELOG.md` | `## [Unreleased]` → new `### Added` entry | A change the operator can see. Every console change in Unreleased has an entry. |
| — | `modes/tty/broadcaster_slots.go` | `bcReadCardRows` / `bcFlatCardRows` / `bcReadLines` | **No change.** Raising `bcReadCardRows` is the trap (see the table below). |
| — | `modes/tty/broadcaster_alloc_test.go` | `bcFrameAllocs = 309`, `bcFrameMissAllocs = 9655` | **No change** (300 and 9173 measured). Re-pin only if a heavier line goes over the budget. |
| — | `modes/tty/broadcaster_upnext.go` | `bcUpNextLabelW`, `upNextWidth`, the `body < 20` floor | **No change.** No width constant moves, because the line fits the existing cell. |
| — | declset / plan / cadence goldens | `app`, `platform/render`, `platform/snapshot`, `platform/history`, `domains/weather/nws` declsets; `platform/lineup/testdata/plan.golden` | **None apply.** `modes/tty` has no declset. Nothing in `platform/` changed (a helper put in `platform/render` would have needed `-update-declset`). |
| — | `modes/tty/testdata/router-trace.golden` | `TestRouterUpdateIsTheSameUnderRandomMessages` | **No change**, because the trace's console has an undecided UP NEXT, which returns before the new row. It does move if the card's height changes (see the table). |
| — | `06_docs/mutants/*` | the 24 mutants that name `broadcaster_slots.go` / `broadcaster_manifest.go` / `broadcaster_upnext*` / `broadcaster_lineup.go` | **No drift.** Each mutant's own `assert old in s` was run with its write disabled, the same method as `scripts/quality/mutant-anchors.sh`: 24 checked, 0 drifted. |
| — | docs (`docs/*.md`, atlas) | — | **No change needed.** No current-reality doc describes the card's rows. `tools/atlas` reads no Go. |

## 2. The guard for each site, verified by leaving it out

Method: the finished files were copied aside to `t3-aside/final/`. Each omission was applied by
an asserted script edit, the whole `./modes/tty` package was run (`go test ./modes/tty
-count=1`, about 61 s), and the files were restored by copy and `cmp`-checked. No `git checkout`
was used.

| Site left out | Guard that fails | Failure, in short |
|---|---|---|
| S1 (the row in `readBody`) | `TestConsoleGoldens`, `TestTheUpNextCardNamesTheReportsItCarries`, `TestTheLongestReportsLineFitsAtTheFloor` | golden: `console-100x44-standby-plain.golden differs ... at line 20`, want the `REPORTS: Location Report` row, got a blank one. Tests: "the card has no REPORTS line". |
| S2 (the `cardReports` function) | compiler (`go build` / `go vet`) | `broadcaster_slots.go:161:17: undefined: cardReports` |
| S2a (`Describe()` used in place of `reportTypeOf`) | `TestTheUpNextCardNamesTheReportsItCarries`, `TestConsoleGoldens` | `no set of its own: the line reads "REPORTS:", want "REPORTS: Location Report"`. Golden: got `REPORTS:` with nothing after it. |
| S2b-i (no fitting at all: `return line`) | only `TestTheReportsLineEndsInTheGlyphSetsEllipsis` (new) | `"REPORTS: Location Report, Full" is 30 cells; want at most 20`. **Without the new test, NO GUARD:** no golden or existing test draws a line long enough to cut. |
| S2b-ii (a bare `TruncateCells`, no mark) | only `TestTheReportsLineEndsInTheGlyphSetsEllipsis` (new) | `"REPORTS: Location Re" ... ending "..."`. **Without the new test, NO GUARD.** |
| S2b-iii (a literal "…" in place of the glyph set's) | only `TestTheReportsLineEndsInTheGlyphSetsEllipsis` (new), ascii case | `ascii=true: "REPORTS: Location R…"`. **Without the new test, NO GUARD:** `TestTheConsoleCarriesNoNonASCIIUnderASCII` (150x74, no cut) and `TestASCIIFramesCarryNothingButASCII` (whose `asciiSurfaces` does not include the Broadcaster console at all) both pass. |
| S3 (`room` = the manifest's width) | only `TestTheLongestReportsLineFitsAtTheFloor` (new) | `at 100 columns the line is cut: "REPORTS: Location Report,...", want "REPORTS: Location Report, Full"`. **Without the new test, NO GUARD:** the goldens' card has an empty set ("Location Report", 24 cells), which fits either way. |
| S4 (goldens not re-recorded) | `TestConsoleGoldens` | `console-100x44-standby-plain.golden differs from its golden at line 20`. It stops at the **first** differing file (`checkGolden` fails fatally), so the message names one file while all 12 differ. |
| S5 (the new tests) | **NO GUARD** | Whole package `ok`. Only the goldens still hold the line, at three widths with an empty set. The fitting, the ellipsis, `--ascii` and the floor width then have no guard (S2b, S3). |
| S6 (the `readBody` comment) | **NO GUARD** | Not run: `tools/authoring` matches phrases and caps headlines only, and cannot see a missing or stale comment (`06_docs/code-standards.md:3-5, 302-305`). |
| S7 (CHANGELOG entry) | **NO GUARD** | No test or tool reads `CHANGELOG.md` (the only `.go` file that mentions it is a comment in `platform/config/config.go`). |
| Trap A: also raise `bcReadCardRows` (`+3` → `+4`) | `TestConsoleGoldens` and `TestRouterUpdateIsTheSameUnderRandomMessages` | The card grows by a row and the takeover box beside it grows to match (golden line 28: got `10.`). `router-trace.golden ... line 305` frame hash changes. **The new test's height check does NOT fail**, because it measures against `bcReadCardRows+2`, which moves too. |
| Trap B: a third new line (overflows the padding) | `TestConsoleGoldens` and the new test's height check | `the card is 16 rows; want 15, its fixed height`. **Before this change only the goldens held the card's fixed height.** No existing unit test pins it (a second extra line fits silently in the last blank row). |
| (not a site) the allocation budget | `TestConsoleFrameAllocBudget` (`make alloc-budget`; skipped under `-race`) | Passes at 300/309. Recorded so a heavier line knows its margin is 9. |

Gates **not run** here, as instructed: `make verify`, `make mutant-*` (an equivalent read-only
anchor check was run instead), `make p10` (the harness CLI is outside the public tree,
`code-standards.md:312-317`), and `go run ./tools/{authoring,dupes,wires}` (`go run` is outside
the allowed `go test`/`vet`/`build`). The rules the code was written to:
- P10-02: no new loop in production.
- P10-04: `cardReports` is 8 lines.
- P10-05: one call-free guard.
- AP-CITE-01: the new comments cite no new IDs, and `D-110` is already in the package.
- AP-CAPS-01: the new paragraphs are in sentence case.
- dupes: the new body is not a copy of any other.

## 3. Doc accuracy

The route a junior gets is README "Start here" → architecture → where-things-happen → extending →
code-standards (→ the 0.16.0 required reading). **No doc routes to the UP NEXT card's body**
(`broadcaster_upnext.go:upNextBox` → `broadcaster_slots.go:readBody`, its lines built in
`broadcaster_manifest.go`). The card was found by grepping `DATA PULL`.

| Doc:line | Routes right? | Finding |
|---|---|---|
| `README.md:623-634` | yes | The five steps exist and point at real files. Step 2's "the five roles" is right (`where-things-happen.md:115-119`). |
| `docs/architecture.md` (whole) | partly | **Does not describe the five roles.** It names only the Director (`platform/lineup`) and MasterControl (`:70-73`, `:82`). Card Producer, Card Composer and Reader appear nowhere on the page; they are in `where-things-happen.md:115-119` and `00-REQUIRED-READING.md:21-29`. The task brief's description of architecture.md is wrong here. |
| `docs/architecture.md:91` | package only | `modes/tty` "the Broadcaster console". No file is named. |
| `docs/architecture.md:103-104` | missing step | Says extending.md covers "a detail row, a provider, a radio control, a report kind, a map layer or a Setting". There is **no Broadcaster card or console recipe**, and "a detail row" is Observer's Location Details (Walkthrough 1, `modes/tty/detail.go:detailLines`), the wrong surface for this task. |
| `docs/where-things-happen.md:70-75` | missing row | The Broadcaster rows cover the swap, the transmitter, a read need, a fault, a takeover and the STANDBY notice. **There is no row for "a console card is drawn".** |
| `docs/where-things-happen.md:74` | **wrong file for half** | "the console draws the air and the lanes ... in `modes/tty/broadcaster_air.go`". That file draws the air box only (`airBox`, `liveLine`, `bedLine`). The frame (`Broadcaster.lanes`) is in `broadcaster.go:801`, UP NEXT and the takeover are in `broadcaster_upnext.go`, the running order is in `broadcaster_lineup.go`. A junior who follows this lands on the LIVE NOW **row**, not the card. |
| `docs/where-things-happen.md:97` | **wrong symbol** | "the console's columns (`Broadcaster.lanes`)". `Broadcaster.lanes` builds the whole frame top to bottom. The column/width concept on the console is `cardLane` / `newCardLane` (`broadcaster.go:1263, 1277`). |
| `docs/where-things-happen.md:68` | ID mismatch | "(0.15.0 FR-7.1)". ids.md's only FR-7.1 row is 0.16.0's (breakpoints). See §4. |
| `docs/extending.md:116-119` | yes | `bcReadLines` is in `broadcaster_slots.go:108`. The card's height is fixed (`readBody` pads to `bcReadCardRows`). The detail test fixture is named "Watchpost Air Quality Report" (`broadcaster_detail_test.go:63`). All accurate, and the only doc pointer anywhere near the card. |
| `docs/extending.md:116-118` | missing step | "The card's height is fixed" is true, but no doc says that the fixed height **already has two blank rows**, or that a third added line grows the card with only the goldens to notice. |
| `docs/extending.md:124-126` | not checked | The `Spec.Label` width note is about the running-order column. It was not needed here; the card line was measured separately (it fits at 100 columns). |
| `docs/extending.md:185-186` | **inaccurate** | "Before the path, `go test` refuses a flag it does not know." Measured: `go test -count=1 -run TestConsoleGoldens -update-golden ./modes/tty` (and `-update-golden=true`) prints `no Go files in <repo root>` / `FAIL . [setup failed]`. Go treats the unknown flag and everything after it as test-binary arguments, so the package path is lost. It fails, but not with a refusal of the flag. The task brief's "silently does nothing" is also wrong: it fails visibly. The form after the path (`go test ./modes/tty -update-golden`, as written) works, and so does `-args -update-golden` (`:172`). |
| `docs/extending.md:186-189` | yes | The flag-to-package list matches each `flag.Bool(...)` (`modes/tty`, `modes/report`, `domains/radio/synth`; the declsets; `platform/lineup` plan; `app` cadences). |
| `docs/extending.md` (whole) | missing step | No walkthrough states the console's `--ascii` rule. The rule is real: the code routes every mark through `o.Glyphs()`. But the console is **missing from** `asciiSurfaces` (`modes/tty/golden_test.go:280-306`). Only `TestTheConsoleCarriesNoNonASCIIUnderASCII` (one size, no truncation) scans it. |
| `06_docs/code-standards.md:3` vs `:312-317` | contradiction | "Every rule below is checked by a tool in this repository ... every tool runs without anything outside it." Then: "The checker [P10] and its exemptions ledger live in the harness, outside this public tree", and `make p10` "fails loud ... when the CLI is absent." |
| `06_docs/code-standards.md` (whole) | missing step | Says nothing about goldens or the alloc budget's re-pin route. `:296` lists `make alloc-budget` but not that it is skipped under `-race`. Both routes are in extending.md `:177-190` and in `broadcaster_alloc_test.go`. |
| `06_docs/02_features/0.16.0-broadcaster-ui/00-REQUIRED-READING.md:204` | **wrong file** | "the seam is `cardBoxWidth` (`modes/tty/broadcaster.go`)". It is in `modes/tty/broadcaster_rail.go:135`. |
| `00-REQUIRED-READING.md:235-236` | **wrong file** | "**F-34** lives in `06_docs/follow-ups.md`". `follow-ups.md` has no F-34. |
| `00-REQUIRED-READING.md:34-37` | stale but labelled | "`app/director.go`". The note at `:39-40` says it is `app/arbiter.go` now. |

Where the code and the docs disagreed, I followed the code. The card the brief calls "LIVE NOW /
up-next" is two things in the code: the LIVE NOW **row** (D-95) and the UP NEXT **card**.

## 4. IDs met in the docs followed, against `06_docs/ids.md`

The check takes every hyphenated ID in the docs above and looks for an ids.md row whose first
cell is that ID.

**Resolve:**
- architecture: D-10.
- where-things-happen: D-11, D-13, D-124, D-4, DR-21, F-21, FR-9, FR-9.4.
- extending: FR-1.1, FR-1.2, D-15, D-61, D-292, D-33, FR-3.8.
- code-standards: D-10, D-12, D-27, D-30, D-34, D-56, D-142, D-160, A-2, DC-14, JM-3.
- required reading: D-11, D-13, D-23, D-24, D-31, D-33, D-37 to D-51, D-53, D-54, D-73, D-80,
  D-159, DR-3, DR-15, DR-16, DR-24, F-108, FR-4.2, FR-5.5, FR-5.6, RS-3.

The release-qualified ones (`0.19.5 D-13`, `0.19.5 D-33`, `0.19.5 D-10`, `0.19.5 D-34`,
`0.19.5 A-2`, `0.19.5 D-12`, `0.19.5 D-27`, `0.19.0 D-142`) each have a row in the release
they name.

**Do not resolve through ids.md:**

| ID | Cited at | Note |
|---|---|---|
| FR-7.1 (as "0.15.0 FR-7.1") | `where-things-happen.md:68` | ids.md has only a 0.16.0 FR-7.1 (breakpoints). The 0.15.0 one exists in prose and a plan table (`0.15.0-.../08-reports/plan-report.md:323`, "FR-7.1 … FR-7.5"), not as a defining row. |
| MVS-D-3 | `where-things-happen.md:54` | No MVS-prefixed row. ids.md never indexes multi-voice-support's D rulings by prefix. |
| MVS-D-15 | `where-things-happen.md:60` | same |
| MVS-D-80 | `where-things-happen.md:147`, `code-standards.md:225`, `00-REQUIRED-READING.md:202` | Defined only as a paragraph in `multi-voice-support/04-development/director-build-log.md:2225`. ids.md's own header (`ids.md:13`) uses it as its example. |
| F-111 | `extending.md:94` | No row. It appears in `0.16.0-broadcaster-ui/03-architecture-design/report-requests-plan.md` and `08-reports/red-team-build.md`, not in a table's first cell. |
| R3-A-01 | `where-things-happen.md:34` | No row. |
| AP-HIST-01, AP-DEAD-01, AP-SHELL-01, AP-CITE-01, AP-CAPS-01, AP-OK-01 | `code-standards.md:4, 109, 192, 210, 29/236, 267`; `where-things-happen.md:150` | Not ids.md families. Defined in `code-standards.md` itself (its `:8-12` says so). |
| SN-02, SN-03 | `code-standards.md:39, 124, 171` | same: defined in code-standards.md |
| P10-00 … P10-10 | `code-standards.md:311-338` | same: defined in code-standards.md's P10 table |
| BD-9 | `00-REQUIRED-READING.md:54` | No row. Cited to `director-build-log.md:809`. |
| F-164 | `00-REQUIRED-READING.md:206` | No row. |
| F-34 | `00-REQUIRED-READING.md:235` | No row, and not in follow-ups.md where the doc says it is. |
| P-1, P-6, P-8, P-9, P-10 | `00-REQUIRED-READING.md:171-178` | No rows. Cited to `04-development/p3-flip-postmortem.md`. |
| PD-3 | `00-REQUIRED-READING.md:231` | No row (also cited in code, `cardPulled`'s doc). |
| S-4, S-5, T-1, T-5 | `00-REQUIRED-READING.md:90, 228` | No rows (lineup-model.md's own labels). |
| SEV-0 | `00-REQUIRED-READING.md:5` | A front-matter severity, not a record ID. Listed for completeness. |

## Final run, with the complete change in place

- `gofmt -l modes/tty`: clean. `go vet ./modes/tty`: clean. `go build ./...`: clean.
- `go test ./modes/tty -count=1`: **ok** (61.5 s).
- `go test -race ./modes/tty -count=1`: **ok** (221.6 s; the allocation budgets skip under race).
- `TestConsoleFrameAllocBudget`: hit 300 / 309, miss 9173 / 9655.
- Touched packages: `modes/tty` only. The other change is `CHANGELOG.md`, which no package reads.
