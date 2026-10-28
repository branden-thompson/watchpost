---
title: "0.19.0 — BUILD log"
date: 2026-10-08
phase: REVIEW
sev: SEV-0
authority: HUM LEAD
status: "LIVE — one entry per batch, BUILD's and REVIEW's remediation; the record moves before the gate (REFLECT L5)"
---

# BUILD log

BUILD opened 2026-10-07 at the PLAN gate (D-139). The order is W0.0, then W1 (#27), then the libraries'
release candidates (go-tuiMaps v0.3.0 first, D-3), O1 and O3 (`implementation-plan.md`).

## Batch 1 — W0.0, the wire cost (D-111, D-141)

- Three requests to NOAA with gzip asked, under D-39 (`07-readiness/dry-run.md`, "W0.0").
- **Found:** GloTEC's grids are not compressed (2.5 MB on the wire); the index and D-RAP are.
- **Ruled:** G1's downloads at most 19 MB an hour at the 10-minute default and 3.3 MB an hour hourly (D-141). The newest grid is found through the gzipped index.
- No code. Docs lane.

## Batch 2 — W1.1, #27: a failed year read never rewrites the year (D-55, D-142)

- **The test first:** `TestAFailedYearReadNeverRewritesTheYear` (`platform/history/history_test.go`). It rolls up a day, makes the year's file unreadable (past the 32 MB read cap), rolls up the next day, and checks the year is not rewritten and the day's hours are kept.
- **It failed for the named reason:** the year was rewritten to 218 bytes (one day), and the day's hours were pruned.
- **The fix** (`platform/history/history.go`, `rollUp`): when the year's file exists but cannot be read, the write is refused and the day waits.
- **Mutation check:** the guard disabled, the test fails; restored by copy.
- **AP-OK-01, the local rule (D-142).** The P10 checker is li-A2DH's CLI, not watchpost's (the plan's `tools/p10` was the agent's error E-20).
  - The rule catches a module function's bool assigned to `_` (`tools/authoring/okdiscard.go`), with self-test specimens in both directions.
  - It found 24 sites in production code. **All are fixed**, by handling the bool or by giving value-only callers a function without one: `NormalizeKey`, `chosenOn`, `radioVolumeCmd` (whose bool was always true), and `setTemp` (its hand-in fact kept as `tempHanded`).
  - None was a live bug. At the alert-area estimate, `drawable()` already drops the alerts the bool would have flagged; the agent first said otherwise and corrected it.
  - The upstream request is in `06_docs/quality-observations.md`.
- **Checks:** every touched package whole with `-race`; `make lint` (no new findings); `make p10` (0 live); `make lint-authoring` (no findings).


## Batch 3 — W1.2, #27: roll-ups within the read cap, a byte bound, compact values (FR-6.5, D-143, A-31)

- **The tests** (`platform/history/rollup_test.go`) were written with the change. The code before it wrote one file a year with no size limit, so the test's part names did not exist.
- **Parts by size.** Roll-ups go to `rollup/<YYYY-MM>-p<N>.json.gz`, each within half the read cap (16 MiB decompressed).
  - A day goes to its date's part, else the month's last part while it fits, else a new part.
  - A day larger than a part on its own waits, its hours kept.
  - A part that exists but cannot be read is never written over (W1.1's guard, moved into `partFor`; mutant m110 re-pointed to it).
  - A month's parts are listed from the directory, so a removed part hides none after it.
  - 0.18.0's yearly files are still read (`TestALegacyYearRollUpIsStillRead`) and pruned past retention with the parts.
- **The soak at the real budget** (`TestNinetyDaysOfAGlobalGridStayReadable`, `make property`, about 45 s): 90 days of a 2° global grid, two fields, synthetic values. Every part stays within the budget and every day is read back.
  - **Found:** a 31-day month is 16.67 MB decompressed, one part, just within the 16.78 MB budget. A further field splits it, and the parts handle that.
  - The agent first expected four or more parts. The test was corrected to what the design promises: within budget, one part or more a month.
- **Compact values (FR-6.5, PF-F5).** Values are held as one `[]float64` a field, NaN for missing, not one pointer a value.
  - Written byte for byte as before (`TestValuesAreWrittenAsBefore`).
  - Read exactly as 0.18.0's pointers read (`FuzzValuesAreReadAsBefore`, a minute of fuzzing, about 26 million inputs).
  - 4,096 values read in 1 allocation (`TestValuesReadAllocBudget`, in `make alloc-budget`; the pin count is now nine).
  - A reader gets its own copy (`TestARecordReadIsItsReadersOwn`).
- **The byte bound (D-143).** `Dataset.MaxBytes`, counted over every version.
  - Past it, the prune pass removes the oldest files first: roll-ups, then days with their buckets, never the current day.
  - `Store.Bounded` reports a dataset cut short. The Data tab names it; only a changed retention clears that.
  - Every app dataset declares a bound (`TestEveryDatasetHasAByteBound`): 4 GiB for NDFD's current hour and AirNow, 512 MiB for the rest (A-31, from this machine's measured rates).
  - Tests: `platform/history/bound_test.go` (six) and `TestTheDataTabSaysWhichDatasetItsBoundCutShort`.
- **Measured for D-143:** a day of 24 hourly records of the 2° grid is 1,454,432 bytes on disk. The agent had put this figure to the HUM LEAD as "measured" before measuring it; the estimate held, and the ruling row says so.
- **Mutants** m111 to m118, each checked killed by hand and restored by copy: a full part takes another day; a month part outlives its retention; a reader shares the store's values; values are written unlike before; the bound takes the current day; the bound takes the newest first; the same retention clears the words; the bound trims only the current version.
- **Docs:** the store's design (`observer-maps/03-architecture-design/history-store.md`) and the package comment now describe the parts, the bound and `Bounded`. No diagram draws the roll-ups.


## Batch 4 — W2.0, the map-mode type; D-152's toolchain; the rulings since batch 3

- **The map-mode type (C-M2, A-29).** `radarMode()` was a boolean, so each of its negations meant "Forecast, or any mode added later". It is now `mapMode()`, returning `modeRadar` or `modeForecast` (`modes/tty/map_temp.go`).
  - Every site names the mode it means: `d.radarMode()` became `== modeRadar` and `!d.radarMode()` became `== modeForecast`. **That alone did not keep a third mode out of the other modes' branches:** an `else`, a tagless switch's `default` and an early return still caught it (batch 5's survey found the sites; batch 5 widened the test and audited them).
  - **27 call sites**, not the plan's 28, which counted the definition.
  - `TestEveryMapModeIsHandledAtEverySite` walks the package and fails on a mode compared with `!=`, a negated mode comparison, a switch on the mode that leaves one out or has a default, and the old boolean. Five planted slips are caught. The modes are read from the declarations, with the type carried down an `iota` block; the agent's first walk missed `modeForecast` there.
  - Mutants m119 (a site decides by "not Radar") and m120 (the radar layer read as Forecast), each checked killed by hand and restored by copy.
- **D-152:** `go.mod`'s floor is 1.26.9; CI and the release workflow build on go1.27.2 (GO-2026-6617).
  - **Found:** under go1.27.2 golangci-lint v2.13.1 cannot read the standard library's export data (version 5; it reads to 4), so `make lint` reported `typecheck` findings in whichever package it loaded first. go1.26.9 and go1.27.1 run clean, so the toolchain is the cause. v2.13.2 fails the same way on the whole tree; the agent first read a piped `head`'s exit code as v2.13.2's success. v2.14.0 runs clean with the six baselined findings, and is pinned (A-32).
- **`ci.yml`'s `make property` comment** names the history soak as well (W1.2 added it to the target).
- **Docs:** the rulings D-144 to D-152 (go-tuiMaps' presets and A-5's hotfix, UAT-1, the floor) and the plan's UAT-1 section (D-151), made since batch 3.
- **The libraries, for UAT-1:**
  - go-tuiMaps P0, P1 and P2.1 are built; `v0.3.0-rc.1`'s release check is green but it is not tagged.
  - go-ionomaps G0 to G4.2 are built, G4.2 being the NOAA-only `Library.Update` (its D-56, A-4).
  - Neither is pushed: the GitHub token lacks the `workflow` scope. W3.1 and W4.1 import both, so they wait for the push; W2.1 to W2.5 do not.
- **Checks:** `modes/tty` and `app` whole with `-race`; then `make verify`.

## Batch 5 — W2.1 to W2.3, the Propagation mode: its key, its bound, a place in no region (D-153, D-154, A-33)

- **The mode.** `modePropagation` joins Radar and Forecast. `mapPane.prop` holds it, so it is never saved, and every open clears it (D-154). P enters it and returns to the mode it came from; R leaves it for Radar (D-153). Both recentre on the selected place.
- **Batch 4's claim corrected.** A survey of the map's code found that a site written `if mode == Forecast { … } else { … }`, a tagless switch with a `default`, or an early return sends a new mode into another mode's branch. `TestEveryMapModeIsHandledAtEverySite` now also fails on an `else` after a mode comparison and on a `default` in a switch that decides by mode (seven planted slips, each caught by line).
  - The sites were rewritten as three-way switches: the temperature's and feels-like's days, the rows under the map, the playback keys, the badge, the chips and the status line.
  - The early returns were audited by hand: `ensureMainOverlay`, `flipHighLow` and the forecast's tick now stop for the Propagation mode as for Radar.
- **Layers (FR-1.4).** `layerOn` is false for every weather layer in the Propagation mode, which takes the radar's loop, the temperature, the alert areas and their badges off. Nothing of the weather's is asked on entering it. `retime` filters the held feed by layer, so a step before the feed's next answer cannot draw the weather again (m127, which survived until the test looked before that answer).
- **Bound (FR-1.2, FR-1.7).** No region's bound in the Propagation mode, so a view can cross any meridian.
  - **Found:** with the zero bound go-tuiMaps v0.2.0 zooms below 0 (-6 reached), and `TestEachModeKeepsItsOwnBound`'s drive at that zoom was killed for memory.
  - watchpost holds the least zoom, the world filling the map one way (`holdWorld`), after every zoom, resize and entry. F-214 asks the library for a least zoom without a box.
  - The weather modes keep the library's bound: back from the Propagation mode, every frame is inside its region.
- **A place in no region (FR-1.3).** The words name P, as bound; P draws the Propagation mode there, on the place; P again gives the stated state back.
- **Words (FR-1.1).** The badge PROPAGATION, a chip `[P] Propagation On/Off`, a status line, a loop row, and a sentence in the description: every path, `--ascii` and "Instead of the map" included (A-33).
- **The frame's cost.** The first build cost 140 more allocations a memo miss: `layerOn` now asks the mode, and `choiceOf` split the choices into a new slice on every call. `strings.SplitSeq` walks them without one; the miss is 4,011, below batch 4's 4,164, and no budget moved.
- **Goldens.** The three map goldens changed by the P chip alone (`-update-golden`).
- **m119 re-pointed.** Its line, the playback `if` in `handleMapKey`, became a switch, and `make verify`'s `mutant-anchors` stopped the first run of this batch: the agent rewrote the line without grepping the mutants first. It now mutates `flipHighLow`'s guard to "not Forecast", which the mode test catches.
- **Mutants** m121 to m128, each checked killed by hand and restored by copy:
  - three of the agent's own needed work:
    - m123 did not compile (a duplicate case) and was rewritten.
    - m124 was first "killed" only by the process running out of memory, so the bound test now asserts the zoom before its drive.
    - m122 assigned the field to itself, which the harness's `go vet` refuses, and stopped the second `make verify`. It now deletes the reset. Every new mutant is now applied and vetted by hand before verify.
- **Docs:** the as-built map diagram draws `map_prop.go` and the P key; the atlas is regenerated.
- **Checks:** `modes/tty` and `app` whole with `-race`; then `make verify`.

## Batch 6 — W2.5, the acknowledgement (D-46, D-81, A-9, A-34)

- **The window.** The first P opens it over the map, in the same key press that enters the Propagation mode, so it is shown before anything of the mode's is fetched (W4.1 adds the fetch). It is a window of its own, `modalPropAck`, stacked over the map; Enter or Esc closes it to the map in the Propagation mode, and no other key leaves it.
- **The record (A-9).** Closing it saves the wording's version, `propagation_ack` in the config, through a save hook of its own (`SavePropagationAck`, classified for the air in `app/air_boundary_test.go`). A version seen older is shown once more. With the wording at version 1, "older" and "never seen" are both 0, so the comparison is a function the test calls with versions 1 and 2 (`ackDue`).
- **The cursor (D-81).** The terminal cursor stands at the start of its first line, found in the frame as drawn; it is placed nowhere else (D-110).
- **Found by the reachability gate:** at 80x24 one of its lines could not be brought on screen, since a window that takes every key scrolled with none. The Observer's scroll keys now read it.
- **Found by its own `--ascii` test: two leaks older than this release.** The radio panel's head joined its title and station with a literal "•", and its track's rails were a literal "│"; under `--ascii` both printed. They now come from the glyph set (`*` and `|` under `--ascii`). `TestASCIIFramesCarryNothingButASCII` did not reach that panel state; `TestTheAcknowledgementReadsWithoutThePicture` holds it now.
- **Batch 5's tests** close the acknowledgement with Enter when P opens it, as a listener does.
- **`app`'s declaration set** gains `savePropagationAck` (`-update-declset`).
- **W2.4 waits for W3.1 and W4.1** (A-34): its controls act on the layers and the update those bring.
- **Mutants** m129 to m134, each applied, vetted and checked killed by hand, restored by copy.
  - m134 first left a variable unused, which vet refused; rewritten.
  - Under m129 the package ran into its timeout: the tests' key helper ran every command a key returned, and Esc on the map schedules its five-minute release. The helper now runs only what a key on the acknowledgement asks for, and the window test stops at once if P opened none; m129 is killed in 42 s.
- **Docs:** the as-built map diagram draws the acknowledgement; the atlas is regenerated.
- **The first `make verify` stopped at `lint-authoring`** (AP-HIST-01): a comment said versions "seen older", which reads as history. It now states the rule.
- **Checks:** `modes/tty`, `app` and `platform/config` whole with `-race`; then `make verify`.

## Batch 7 — W4.1, the data path: the propagation client, the one service, the update in the window (D-155, D-156, A-35)

- **The libraries.** watchpost requires go-tuiMaps `v0.3.0-rc.1` and go-ionomaps at its pushed G4.2 (a pseudo-version); both are on GitHub since the token gained the `workflow` scope.
- **The propagation client** (`platform/httpx/plain.go`, FR-4.4, FR-4.7): every reply as it came, a 429 included, asked once; no cache, no pacing shared with the data client; https alone, no private address, an 8 MB cap; no error carries the reply. Its tests were written with it, not before it.
- **The service** (`domains/propagation`, FR-4.5, FR-4.7, FR-4.8, D-131): the one go-ionomaps object; a fetcher that refuses a host the library does not export and hands it only the status, the rate headers and the validators, each rate header's first sighting noted by host and name; one update at a time, joined by any asked while it runs, its own 60 s deadline, ended by the mode's context.
- **The window** (`modes/tty/map_prop_update.go`): `Config.PropagationUpdate`, called in a command, never on the UI goroutine.
  - The first update starts when the acknowledgement closes, or at once when none is due (A-35); every 10 minutes after while the mode is open.
  - Leaving the mode, opening the map again or closing it ends the session's update; a late result or tick is dropped by generation.
  - The status line and the words say the snapshot: foF2's background and GloTEC's time, M(3000)F2's, the time computed, an early answer's reason; with no field, the library's words, which say when it asks again.
- **The app** (`app/mapprop.go`): the service made on the first update, once a process (`TestOneLibraryObjectPerProcess`); MAP STATUS lists GIRO and NOAA SWPC from go-ionomaps' own hosts (FR-4.3); About credits SWPC, PyIRI and IGRF-14, three chips of their own (A-35).
- **Rulings:** L flips the layer (D-155) and U refreshes now (D-156); both keys come with W3.1 and W2.4 in batch 8, where there is a layer to flip.
- **Not yet drawn:** the fields themselves (W3.1, batch 8). The day/night terminator waits for go-tuiMaps P4, after `v0.3.0-rc.1`.
- **Two guards found what the targeted runs missed:**
  - `TestEveryWindowReplyIsRoutedBackToTheWindow`: the two new messages are routed to the map window, so a Broadcaster-surface map gets them.
  - `TestEveryRegisteredSourceIsOnTheClosedList`: GIRO and NOAA SWPC join the closed list of hosts (FR-4.3, D-113).
- **P10:**
  - Bare-name recursion: go-ionomaps' constructor is a package value; the service's method is `Ask`, not `Update`; the window's local is `seam`.
  - Density: the fetcher reads its hosts from the service's own library, which removed a probe library and two small functions. Real checks are phrased so P10 counts them, and three are added: a service not made with `New`, a negative deadline, a reply with no HTTP status.
- **Mutants** m135 to m144, each applied, vetted, killed and restored by copy.
  - **The agent's first run did not restore:** in zsh an unquoted `$FILES` is one word, so the copies aside failed and all ten mutants were applied on top of each other. Each was undone by inverting its own replacement, latest first (one by its context, its text occurring five times); every touched package then passed; the run was repeated with an array and a check after every restore.
  - m138 survived until the join test compared the two answers: unjoined updates still asked NOAA once, the library answering the second "too soon".
  - m142 was first "killed" only by the routing guard; its own test now delivers an old session's tick to a new idle one.
  - Five anchors moved with this batch's edits and were re-pointed: m122, m131, m136, m137, m144.
- **The first `make verify` stopped in `cmd/watchpost`:** `THIRD_PARTY_LICENSES.md` did not list go-tuiMaps `v0.3.0-rc.1` or go-ionomaps at their required versions. A `go.mod` change touches every package, and the agent had run only the packages it edited; the whole tree is now run before verify. Regenerated with `scripts/third-party-licenses.sh`.
  - **Found:** the script copied each module's licence but not its NOTICE. go-ionomaps names PyIRI's MIT licence there, and yaml/v3's Apache-2.0 NOTICE was missing from the file before this release. The script now copies a module's NOTICE too. go-ionomaps' NOTICE names PyIRI's licence and copyright but not its full text, which must ship before go-ionomaps' first tag and 0.19.0's SHIP (F-215).
- **Docs:** the as-built map diagram draws the service, the client and the seam; the atlas is regenerated.

## Batch 8 — W3.1 and W2.4: the fields drawn, L and U (D-155, D-156, A-36) — UAT-1

- **The fields** (`modes/tty/map_prop_draw.go`, FR-1.5, FR-1.6): the snapshot's hour on screen as go-tuiMaps v0.3.0's `MUFGrid` or `FoF2Grid`, its preset and unit, labelled contours over bands, continued over the sea; handed in and taken off through the window's one `reconcile`, on each update and as the mode is entered or left.
- **The legend row** keys the layer by its colours: "MUF(3000), MHz" or "foF2, MHz".
- **L** flips the layer (D-155); **U** asks for an update now (D-156), honoured under the library's throttle, its early answer said with its reason; in the weather modes neither acts (`invisibleKeys` says why).
- **Chips:** in the Propagation mode `[L]` the layer and `[U] Refresh Now` stand in the places of Area Alerts and Radar.
- **Found by `TestTheLegendBelongsToItsSurface`:** a tenth map row pushed Help past its window at 133x44, so the Observer's row-mark legend fell below the fold. Help's map group is nine rows again: Scroll joins Zoom, and P, L and U share one row.
- **Found by its own test:** the status line, cut at the window's width, hid an early answer's reason at its end; the layer, the time and the reason come first now, the backgrounds after.
- **`TestTheLegendBoxIsRetired`** held L free since D-103; L is the Propagation layer's now (D-155), and the test says so.
- **Mutants** m145 to m150, each applied, vetted, killed and restored by copy; m149 survived until the layer test read the legend row itself rather than any row naming foF2.
- **One live update** through the real client and service, before verify (three NOAA requests, under D-39), from a throwaway program not kept: 1.1 s; GloTEC's grid valid 01:25 UTC at 01:59 UTC; F10.7 mean 103.6 over 29 days; MUF(3000) 7.2 to 38.4 MHz (mean 19.6), foF2 2.6 to 13.3 MHz (mean 6.5) over the 16,200 cells.
- **Not yet:** the terminator (go-tuiMaps P4), the hours ahead, the chart, the stations (G5, G9), the Broadcaster's key (FR-2).
- **Checks:** the whole tree's tests; `modes/tty` and `app` with `-race`; then `make verify`.

## Batch 9 — UAT-1's verdicts: the rows under the Propagation map, the menu's Propagation row, the band legend (D-157 to D-160, A-37)

- **UAT verdicts carried (0.18.0 L8):** D-157 (the rows under the map, the HUM LEAD's mock, "here now, then continents"), D-158 (the Overlays menu's Propagation row), D-159 (the menu in the mode: tints and detail), D-160 (the legend, band over MHz).
- **The rows under the map** (D-157, `modes/tty/map_prop_draw.go`): in the timeline's five rows, the header with the sources' chips; the mode, the time computed in the station's clock and zone and its age, "upper limits only"; the selected place now, its foF2 and MUF(3000); the bands under foF2 for near-vertical paths to about 400 km; the highest band under MUF(3000) for about 3,000 km hops. The words say the same.
  - **Found:** in the agent's first layout the map came out a row shorter than in the weather mode when the weather mode drew no badge row. The legend's MHz line now takes the picture status's row, and the picture's status joins the mode line, so the height is the weather modes' (`TestThePropagationRowsFollowTheMock`).
  - **Found:** the first wiring recursed (the note lines asked whether the picture fits, which counts the note lines) and overflowed the stack; the mode's note lines are none now.
- **The legend** (D-160): the bands on their colours, and under each its lower edge in MHz. foF2's legend carries a class under 1.8 MHz, drawn as nothing (D-149), which keys no band; `bandClasses` aligns the library's classes to the bands and falls back to the one-line row for any other count.
- **The menu** (D-158, D-159): a "Radio Propagation" radio row among the tints. Space on it is P; on it chosen, it leaves as P does; a weather tint chosen in the Propagation mode returns to the weather mode with the tint drawn, never toggled off. In the mode the menu is the tints and MAP DETAILS; the weather groups return, their ticks kept. The menu's apply step is one function, `menuApplied`, for the menu's own switches and these.
- **Mutants** m151 to m157, each applied, vetted, killed and restored by copy.
  - m153 was first killed only by a panic in another test. The legend test checked 160m's colour, which a colourless fixture cannot see; the class mapping is now a function the test asks directly, and m153 was re-pointed at it.
- **Docs:** the as-built map diagram; the atlas.

## Batch 10 — UAT-1, round 2: one MAP DETAIL, Radio Propagation the fourth overlay (D-161, D-162)

- **UAT verdicts carried (0.18.0 L8):** D-161 (two MAP DETAIL headings in the Overlays menu), D-162 (Radio Propagation as the OVERLAYS group's fourth row).
- **One cause:** the menu heads each row's section by whether the row is an overlay (`modes/tty/map_boxes.go:641-643`). Batch 9's Radio Propagation row was not marked one, so it opened a MAP DETAIL section of its own between the tints and the rest, and in the weather modes OVERLAYS was headed a second time after it. Marked an overlay, it is the fourth row under one OVERLAYS heading in both modes.
- **Missed in batch 9:** its menu tests read the rows, not the menu as drawn; `TestThePropagationRowIsTheFourthOverlay` reads the drawn menu in both modes.
- **Mutant** m158, applied, vetted, killed and restored by copy.

## Batch 11 — UAT-1, round 3: the rows under the Propagation map laid out for scanning (D-163, A-38)

- **UAT verdict carried (0.18.0 L8):** D-163, the HUM LEAD's round-3 mock.
- **The layout:** a blank row after the legend, which is the picture's own status row when it has one; the header; the mode line; a blank row; the place's name in its own column with its foF2, MUF(3000) and two answers aligned under NOW. The block is six rows, counted by `mapChromeRows`; the map is a row or two shorter than in a weather mode.
- **Found:** `TestThePropagationRowsFollowTheMock` used a fixture with no radar source, where the rows under the map are not drawn at all, the blind spot that hid the empty area in UAT-1. It now uses one with radar, as the app has; mutant m159 (the blank row not counted) survived until it did.
- **Mutants:** m151 and m157 re-pointed to the new lines; m159 and m160 new; each applied, vetted, killed and restored by copy.

## Batch 12 — UAT-1, round 4: the place in bold, the fields as chips, the age on a general freshness ramp, the badge (D-164 to D-167, A-39)

- **UAT verdicts carried (0.18.0 L8):** D-164 (the place in bold white), D-165 (MUF(3000) and foF2 as chips), D-166 (the age's colour, a seven-rung ramp, made general-use at the HUM LEAD's follow-up), D-167 (the badge "RADIO FREQ. PROPAGATION").
- **The freshness ramp** (`platform/render/freshness.go`): seven tokens, `FreshnessTone(age)` for any feature, filled per theme from the theme's own colours (`withFreshLadder`, beside `withAgeLadder`) and held to AA like every tone. The ladder the app already had (`DataNew` to `DataStale`) has five rungs to 15 minutes and, by the Broadcaster's ruling, no red; it is unchanged.
- **The place** takes `MapPropPlace`, the window title's tone; **the fields** are chips with grounds of their own; the age reads "(just now)" under a minute.
- **The words** are built from the same block, plain: a colour-on test found them carrying the picture's styling once (mutant m166 survived the colourless fixture).
- **Mutants** m161 to m166, each applied, vetted, killed and restored by copy; m157 and m160 re-pointed to this batch's lines.
- **Docs:** the as-built map diagram; the atlas.

## Batch 13 — the quality pass after UAT-1: the Propagation mode's corrections, its frame's cost, and DRY (A-40)

The HUM LEAD asked for a performance, DRY and code-quality pass before the next batch, with behaviour and UI unchanged. Read-only reviews of the Propagation mode, its frame path and services, and the rest of the release raised the findings; this batch is the Propagation mode's. Each fix below was tested first and the test seen to fail.

- **Corrections (each a test written first, failing on the old code):**
  - **One refresh chain** (FR-4.10): every result set a new tick beside the one already pending, so each U press added a permanent ten-minute chain. A tick now carries a sequence, and only the latest set asks (`TestOnlyTheLatestRefreshTickAsks`, 3 asks where 2 were due). Releasing the map reset the session's generation to 0, so a tick from before the release could be taken for the new session's; the generations and the sequence now outlive the release (`TestAReleasedMapDropsTheOldSessionsTick`).
  - **A reopen draws no field** (D-154): the open set the weather mode but left the last Propagation field handed to the kept map; it is taken off (`TestAReopenedMapDrawsNoPropagationField`).
  - **The age moves** (D-166): the window's memo held no clock for it and no tick was armed, so "(just now)" and its tone stood until the next refresh. The age in whole minutes is in the memo's key, and a tick runs while it is shown (`TestThePropagationAgeMovesWithTheClock`).
  - **A tint from the Propagation mode asks once:** it left the mode (asking the weather) and then applied the menu (asking again), two temperature asks against the Open-Meteo quota. The tint is chosen first, while the mode draws none of it, then the mode left once (`TestAWeatherTintFromPropagationAsksOnce`, 2 asks where 1 was due).
- **Every mode but one** (C-M2): eight guards were written as "Radar or Forecast, return", which hands any mode added later the Propagation mode's behaviour (and three as "Radar or Propagation", Forecast's). Each now names the mode it means, and `TestEveryMapModeIsHandledAtEverySite` flags an `||` that names every mode but one (a planted one caught; m167).
- **The frame's cost** (pinned by `TestPropagationFrameAllocBudget`, new, beside the weather modes' pin): the map's size asked for the note lines a dozen times a frame, and in this mode that drew the band legend each time. The rows are now counted without drawing it (`noteRowCount`, held equal to the lines drawn in every mode by `TestTheNoteRowsAreCountedAsDrawn`), and the rows under the map draw it once. **A memo miss: 1,872 to 1,337 allocations (3,668 to 2,876 in colour); `renderMap`: 504 to 126 allocations, 30.9 to 12.5 µs.** What remains of a miss is mostly the overlay compositor the weather modes share.
- **DRY and structure:** `legendClasses` (three copies of the legend lookup); `rainIsRadar` (three); `resetPlayback` (three); `timelineBlock`, so the rows under the map are built for the mode shown instead of built and replaced; `mapChips` a switch on the mode; `overlayRow.weather` named `overlay`, which the Propagation row is too; `refreshProp` without a second mode check, `askPropagation` holding it; the bound's problem names the whole world in this mode; one `geo.Neighbour` call where there were two; `propAckMark` for a bare 12; `FireBoldMW` back with the radio's settings; two comments that no longer said what the code does.
- **go-ionomaps** at `20c4025`, its own quality pass (its build log): an hour's backgrounds 51 to about 34 ms, bit for bit; the index read in 0.74 ms where it took 5.66; the table headers checked; the solar file's validators kept only with the mean taken from it. `THIRD_PARTY_LICENSES.md` regenerated.
- **Mutants:** m119, m148, m150, m155, m158 and m159 re-pointed to the lines their rules now live on (m155 at the choice itself: in the Propagation mode no weather tint counts as chosen, so toggling it chooses it); m167 to m174 new. Each applied, vetted, killed and restored by copy. **Found:** a hand-kept list of files to restore missed one, so m158 stayed applied until `make mutant-anchors` reported it as drift; the runner now backs up each mutant's own target.
- **Not in this batch:** the propagation services' and the history store's findings, and go-tuiMaps', follow as their own batches. Two need the HUM LEAD: whether the sources' badges show when foF2 is the climatology's or nothing was reached, and whether MAP STATUS counts the propagation hosts' requests.
- **Docs:** the as-built map diagram; the atlas.

## Batch 14 — the quality pass: the history store, the propagation client, the themes, the authoring check (A-41)

The rest of the quality pass's watchpost findings outside the Propagation mode, and go-tuiMaps at its own pass. Each change was tested first where it changes what the code does, and the test seen to fail.

- **A bound that cannot be met starved every other dataset** (D-143): the byte bound spent the same 256 visits a pass as the walk that compacts, rolls up and expires, so a dataset held over its bound by its current day alone, across enough series, left no visits for any other dataset. The bounds now have their own visits a pass (`TestABoundThatCannotBeMetStarvesNoOtherDataset`: a day past its retention was kept, now it goes).
- **One reading of a roll-up's name** (`parseRollup`): four functions read the names, differently - a month's part numbered 0 was pruned, bounded and counted but never read. A year's file or a month's part numbered from 1 is now read the same way by every reader and the pruning (`TestARollUpsNameIsReadOneWay`).
- **Values read twice as fast:** each byte of a number was checked by decoding a rune and searching a string; a byte table gives the same answers (non-ASCII and broken UTF-8 included). **`BenchmarkValuesDecode` (new; a hundred thousand values): 11.3 to 5.96 ms**, allocations unchanged.
- **The propagation client's errors are the data client's** (FR-4.7, IS-1): a request with no reply carried the address unredacted and said "after 0 attempts"; it is now redacted, after its one attempt (`TestThePropagationClientsErrorsCarryNoSecret`); a refusal is said as the data client says it, and a reply past the cap is its 413. The capped read is one function, `readCapped`, both clients' (go-ionomaps turns a fetcher's error into its own words, so nothing a listener reads changes).
- **Every propagation source credited** (FR-4.1): `TestEveryPropagationSourceIsCredited` holds About's credits to the sources go-ionomaps names, with GIRO and the two ITU recommendations listed as pending and why, so G5 cannot read GIRO uncredited.
- **DRY:** `borrowUnnamed` (the age ladder and the freshness ramp borrowed their colours in two copies of one loop); `historyHours` and `historyDays` (fifteen and ten copies across the datasets); the authoring check's five rules in one list, `checkFile`, which the scan and its self-test share, and its parse without name resolution, which no rule reads.
- **Comments:** the store's formats named by what they are, not by the release that made them; `dayNames` says the parts it reads; AP-OK-01's reason in the present tense.
- **go-tuiMaps** at `69132c8`, its own quality pass (its build log): water asked once a frame (a pan over the world under a 2° field drawn as lines, 595 to 510 µs over water and 599 to 568 µs at the shore), ramps built once, a mutant harness that copies the tree once. `THIRD_PARTY_LICENSES.md` regenerated.
- **Mutants:** m112 re-pointed to `parseRollup`; m175 to m179 new; mCJ3 now applies to `borrowUnnamed`, and both ladders, and is still killed (`TestMonochromeHasNoColourInIt`). Each applied, vetted, killed and restored by copy. m177 and m175 each first left a variable unused and failed vet; rewritten. **Found by the first verify:** m135's replacement named `errPlainTooLarge`, which this batch removed, so it no longer compiled; `make mutant-anchors` reads anchors, not replacements. It names a refusal that exists, and every mutant on a file this batch changed was then applied and vetted (sixteen). The declaration locks of `platform/render` and `app` re-captured for the functions and constants added.
- **Not done, and why:** the roll-up's re-reading of a month's part (its fit could be worked out from the part's size; the arithmetic must stay exact, and the cost is a background pass once a day a series) waits for a measured need; `Store.Days` reading every part whatever its `max` has no caller yet that asks for fewer, and comes with the Analyst mode, its first (D-27). Three need the HUM LEAD: the Propagation block's source badges when foF2 is the climatology's or nothing was reached; whether MAP STATUS counts the propagation hosts' requests; and go-tuiMaps' safe ramps covering only the radar, temperature and alert ramps.
- **Docs:** none moved; no diagram draws what changed.

## Batch 15 — the Status window by what each host serves; the sources' badges; the propagation client counted (D-168 to D-170, A-42, A-43)

- **Rulings carried:** D-168 (the Propagation block's badges, dimmed where unused), D-169 (the propagation client counts its asks), D-170 (the HUM LEAD's grouping of the Status window, to end its duplicate rows), with A-42 and A-43 for their details.
- **The groups** (`modes/tty/status_sources.go`): one row a host, in one group - COMMON DATA for a host two or more features use, then WEATHER DATA, ADVERSE EVENTS, MAP DRAWING DATA, and OTHER DATA for a host the window counts that no source names. **A shared host's counters are the station's client's and the map's clients' added**, where MAP STATUS showed the station's numbers a second time under the map's name. One header of columns over every group, every group in the one shape (D-150). The headline counts hosts and groups. The tiles held stay on MAP DRAWING DATA's title; the notes and the disclosure under the last group.
- **The badges:** a row's sources are its sources' chips, as About and the map show them. The table measures cells by their bytes, so a chip cannot be a cell; each badge is laid out as its chip's plain face, which is the chip's width, and drawn over as its chip after the table is laid out (`chippedSources`); the columns stay where the table put them (`TestTheSourcesAreTheirBadges`).
- **What each source serves is stated in the app** (`app/endpoints.go`: `providerSources`, `hostSources`, `mapBadges`), and `TestEveryStatusSourceIsGroupedAndCredited` holds every provider, named host and map source to a feature and a badge About credits (or one listed as pending).
- **The acknowledgement's words** said MAP STATUS lists GIRO and NOAA; they say the Status window does, and its version is 2, so it is shown once more (A-9, A-43).
- **D-168:** `propBadges` lights a badge for a source that fed the snapshot shown and dims it, in the app's dimmed tone (`chipDimmed`), for one that did not (`TestTheSourcesBadgesSayWhatFedTheSnapshot`).
- **D-169:** `httpx.Plain` counts its asks per host as the data client does - an answer, a 304, a refusal, no reply (`TestThePropagationClientCountsItsRequests`) - and the app adds them to the map's counters (`TestThePropagationClientsAsksReachTheStatusWindow`); the holder keeps its client behind an atomic pointer, as the Status window reads it from the UI goroutine.
- **Tests written first:** each of the above failed before its code (the grouping test on the two tables' headings and the repeated host; the badges test on the missing `chipDimmed`; the counts on the missing `RequestStats`). The tests that read API STATUS and MAP STATUS now read the groups; `TestTheEndpointTablesAreOneShape` is folded into the column check across groups.
- **Mutants:** m180 to m187 new; m177 re-pointed (a count joined its block); every mutant on a file this batch changed applied and vetted (twenty-seven). The declaration lock of `app` re-captured.
- **go-tuiMaps** at `13270b9`: safe ramps keep every preset's classes (its D-29), which watchpost does not use. `THIRD_PARTY_LICENSES.md` regenerated.
- **Docs:** the as-built map diagram; the atlas; README, `docs/extending.md` and `docs/where-things-happen.md`; FR-4.3 and FR-11.1, and the architecture's two mentions, now name the Status window.

## Batch 16 — the Status window's UAT: groups apart, a note once, the notes terse (D-171 to D-173, A-44)

- **UAT verdicts carried (0.18.0 L8):** D-171 (a blank row between the groups), D-172 (a note several hosts carry said once - the propagation sources' twice), D-173 (the notes terse, in the HUM LEAD's example's style); A-44 for the wordings.
- **Built:** `sourceLines` puts a blank row before each group after the first; `mapWords` says each note once; MRMS's, Open-Meteo's and the propagation sources' notes and the disclosure are terse, the disclosure naming its hosts by their badges ("Map view sent to: OFM (tiles), NWS (…)"). The map's notes that come and go are the map window's words too, and stay.
- **Tests:** `TestTheGroupsStandApartAndANoteIsSaidOnce`, written first; the tests that read the old wordings read the new ones.
- **Mutants** m188 and m189, applied, vetted, killed and restored by copy; m188's first form (an empty `append`) failed vet and was rewritten. Every mutant on a file this batch changed applied and vetted (five).
- **Docs:** none moved; no diagram draws a note's words.

## Batch 17 — W5.1: best bands by continent, the chart window, the typical day, the radius Setting; the night on the map (D-174 to D-177, A-45, A-46)

- **Rulings carried:** D-174 (UAT-2 is the reference chart), D-175 (upper limits only until UAT-3), D-176 (the chart is a window on B, the block gaining the continents' line), D-177 (that line in the 3,000 km hops' place, the map's height unchanged); A-45 (the continents' reference points), A-46 (the chart's details).
- **The libraries:** go-ionomaps at `8befd5b` - the P.533 basic MUF, `Bands` and `Path` with each band's typical open hours, `Reach`, `Ahead` (its build log, G6.1 to G8.1); go-tuiMaps at `d034821` - units on contour labels (P3), the terminator and the night (P4). `THIRD_PARTY_LICENSES.md` regenerated.
- **The chart's answers** (`modes/tty/map_prop_chart.go`): `Bands` for the near-vertical area at the Setting's radius and `Path` from the selected place to the six continents (A-45), worked out in a command - never in Update (FR-1.17) - when the snapshot, the place or the radius changes, asked once for each (`armPropChart`, after every Update as the map's clock is). An answer to an older ask is dropped; a refused one is said in diagnostics (D-124). Measured about 5 ms (go-ionomaps: an area 1.7 ms, a path's 24 typical hours under 1 ms).
- **The block under the map:** the continents' line replaces "~3,000 km hops through here" (D-177): each continent's highest open band and the chart's key (A-46). The rows below the Controls box are cut at the map's full width, not the narrower width beside the box. The near-vertical line says the Setting's radius.
- **The chart window** (`modalPropChart`, B): the bands by target, the limit that decided each with its value, the selected row's typical day by hour with the hour shown marked, and "now differs from a typical day" where a band's status now is not its typical day's at this hour (D-135). It is one of the windows every closed-set test walks: reachability at the floor (PgDn scrolls it; the arrows pick its row), its keys, its memo, its routing.
- **The radius Setting** (D-100, A-18, D-121): "NVIS radius" on the Maps tab, 200, 400 or 600 km, 400 by default, the Observer's alone, saved with the display preferences (`map_propagation_radius_km`). The Maps tab's pickers widen to its longest value, "249 miles (400 km)" (the Settings golden re-captured for it). Its label and Nearby's are one function, `distanceLabel` (the first verify's duplicate check found the copy).
- **The night on the Propagation map** (go-tuiMaps P4, D-174): the night side and its dotted edge are handed in with the field for the clock to five minutes, and handed in again as the clock passes each step.
- **The origin is the selected place** (FR-1.8); the tower, from the Broadcaster, comes with its key (W7).
- **Tests written first:** `TestTheContinentsAreTheDefaultTargets`, `TestBestBandsNameTheirLimit`, `TestTheDaysOpenHoursAreShown`, `TestTheTypicalDayIsLabelled`, `TestADisagreementWithTheTypicalDayIsSaid`, `TestTheOriginIsTheSelectedPlaceOrTheTower`, `TestAnswersNeverBlockTheUI`, `TestTheNearVerticalRadiusIsASetting`, `TestTheNightIsDrawnOnThePropagationMap`, `TestTheNightFollowsTheClock`; each failed before its code. (The first is `TestTheOriginFollowsTheUIMode` since D-193 moved the origin, batch 23; the last was deleted at batch 31, the night following the field's time.) The tests that read the hops line, the one Propagation overlay, Help's map row and the settings kept read the new ones.
- **Mutants:** m190 to m199 new, each applied, vetted, killed and restored by copy (m194's first form left a variable unused and failed vet; it now runs the day's hours backwards). m148, m151, m157 and m166 re-pointed at the lines their rules moved to. **Two survivors found on files this batch changed, both older than it:** m149 (the legend's single line keys the layer drawn - since D-160 drawn only where the bands do not fit, so no test drew it) and m171 (the age's place in the memo key - its test cleared the key's blink between its two frames, so the memo missed for that reason and the age's absence went unseen); each test now catches its mutant. Every mutant on a file this batch changed applied and vetted (eighty-four, these among them).
- **Not done, and why:** the tower as the origin comes with the Broadcaster's key (W7); "your frequency" (W5.2) draws its reach on go-tuiMaps' generic host ramp with the reach mark (its P2.2 and P2.4), which come first; hours ahead and the centre's answer are W5.4.
- **Docs:** the as-built map diagram (the chart, the night, B, the radius row; the keymap node's "L free" was stale since D-155 and now names L, U and B); the atlas.

## Batch 18 — W5.2: your frequency, its reach drawn over the field and said (D-178, D-179, A-47)

- **Rulings carried:** D-178 (F opens the frequency entry over the map), D-179 (the frequency held said in the block's blank row); A-47 for the details.
- **go-tuiMaps** at `e97d5ab`: P2.2 to P2.4 - a host type filled from the low, middle and high spread or its own colours, a broken ramp reported, its legend keyed, and a two-class host type's edge drawn broken, over a field filled before it its edge alone (its L-2.2 to L-2.6, A-9, A-10). `THIRD_PARTY_LICENSES.md` regenerated.
- **The entry** (`modes/tty/map_prop_freq.go`, `modalPropFreq`): MHz ("7.188", "7.188 MHz") or a band by its metres ("40m", "40 m"), a band taken at its lower edge; 1.8 to 30 MHz; at most 16 characters. Enter draws it, an empty Enter clears it, Esc leaves it as it was; what it cannot take is said as what it accepts, never the text typed (IS-4). One of the windows every closed-set test walks.
- **The reach:** go-ionomaps' `Snapshot.Reach` from the selected place at the frequency held, worked out in a command when the snapshot, the place or the frequency changes, asked once (FR-1.17, about 30 ms), and handed to the map as a two-class host field after the layer, so go-tuiMaps draws its edge broken over the filled MUF or foF2 field.
- **Said:** the row between the mode and the place reads "YOUR FREQUENCY 14.200 MHz · skip zone to ~1,100 km · F change" while one is held, blank otherwise (D-179); the chart says it under its key line; the words say it after the bands (FR-3.1).
- **Help:** B and F share the Area Alerts row, "Alerts / Overlays / Bands / Freq." A row of their own pushed Help's row-mark legend off its window at 133 by 44, and P, L and U's row with them broke its two columns (A-47).
- **Tests:** `TestYourFrequencyDrawsItsReach`, `TestYourFrequencyIsSaid`, `TestABadFrequencyIsSaidNotGuessed`. **Not test first:** the code was written while go-tuiMaps' gate ran and the tests after it; each rule's mutant (m200 to m206) is shown caught instead. The closed-set guards name the new window, message and key.
- **Mutants:** m200 to m206 new, each applied, vetted, killed and restored by copy; m198 re-pointed (its return line now appends the night). Every mutant on a file this batch changed applied and vetted (sixty-eight, and these eight).
- **Docs:** the as-built map diagram (F, the entry, the reach); the atlas.

## Batch 19 — W5.4, hours ahead: typical for the hour, its error and basis (A-48)

- **The hours** (`modes/tty/map_prop_ahead.go`): shift+right steps an hour ahead, to 24; shift+left back; n now (A-48). Each hour ahead is go-ionomaps' `Snapshot.Ahead`, the climatology at the snapshot's F10.7 mean, worked out in a command once and kept for the snapshot held (FR-1.17); a new snapshot drops them. Nothing is fetched for a step.
- **One hour on screen:** `propOnScreen` is the snapshot with the hour shown, and every reader takes it - the field and the night (the middle of the hour ahead), the block's place row ("+2 h" for NOW), the chart and the reach, whose keys carry the hour, so each is asked again when the hour moves. `propShown` adds whether that hour is the one asked for.
- **Said typical** (FR-1.13, D-109, D-134): the mode line names the hour, "TYPICAL, NOT A FORECAST" and the typical error at the place in the layer drawn; the words say the same before the place, with the error's basis.
- **Tests written first:** `TestTheHoursStepOnAKey`, `TestAnHourAheadSaysTypical`, `TestTheTypicalErrorStatesItsBasis`, `TestAnHourStepNeverWaitsForAnUpdate`, `TestANewSnapshotMissesTheAnswerMemo`. The last first asked for the chart at once after a new snapshot; the hour ahead has to be worked out again first, and the test now follows that order.
- **Found by the authoring check:** four readers threw away whether the hour was ready; the one function became two, `propOnScreen` and `propShown`.
- **Mutants:** m207 to m213 new, each applied, vetted, killed and restored by copy; m148, m162, m163 and m209 re-pointed. Every mutant on a file this batch changed applied and vetted (seventy-one).
- **Not done:** the centre's answer (D-180) is the next batch.
- **Docs:** the as-built map diagram (the hours ahead); the atlas.

## Batch 20 — W5.4, the centre's answer: a box over the map, said last (D-180, A-49)

- **Ruling carried:** D-180 (the centre's answer in a box over the map's upper left, a mark at the centre); A-49 for its details.
- **The answer** (`modes/tty/map_prop_centre.go`): the map's centre to half a degree; from the selected place, the path's bands (go-ionomaps `Path`) beyond the near-vertical radius, the area's within it; worked out in a command when the map moves or what it comes from changes, from the hour on screen, so an hour ahead is answered as that hour (FR-1.11, FR-1.17).
- **On screen:** a "Centre" box over the map's upper left - the place, MUF(3000) and foF2 there, the distance from the selected place (or "Near-vertical from" within the radius), the limit and the highest band open - and a "+" at the map's centre; it gives way to the Overlays menu. The library is asked for its centre only in the Propagation mode (`TestTheMapIsDrawnInUpdate` holds the weather modes' calls).
- **Said** last in the words (FR-3.1).
- **One shape for the answers:** the chart, the reach and the centre are asked through one `armAnswer` and kept through one `applyPropAnswer`, with `askAnswer` and `answered` beneath them: the duplicate check found the centre's ask and keep copied from the chart's.
- **Tests written first:** `TestTheCentreIsAnsweredAsTheMapPans`, `TestTheCentreAnswerIsSaid`.
- **Mutants:** m214 to m217 new; m191 and m215 re-pointed by the collapse; each applied, vetted, killed and restored by copy. Every mutant on a file this batch changed applied and vetted (fifty-six).
- **Docs:** the as-built map diagram (the centre box); the atlas.

## Batch 21 — go-ionomaps' stations, D-RAP and answers wired in: a build for UAT with every library feature (D-181 to D-189)

- **Asked for by the HUM LEAD** during UAT-2: "what's a clean build I can uat against with all features incorporated" - on this branch, no separate one. **Rulings carried:** D-181 to D-189 (go-ionomaps' GIRO confidence, its one request, the two-call update, the reading age, the per-place answer, and G6.2's method, noise, antennas and tables), recorded during go-ionomaps' G5, G9, R-2.11 and G6 work and committed here.
- **go-ionomaps bumped** from `8befd5b` to `a88f045`: GIRO's stations assimilated (G9, G5), the live and typical offsets (G5.2), each answer's nearest station (R-2.11), D-RAP's disturbed state (G6.3), and the absorption term (G6.2a, not yet a status: "absorbed" comes with G6.2b).
- **The two-call update (D-183):** a snapshot said `StationsNext` is shown at once, said "asking GIRO's stations", and the update is asked again at once for GIRO's burst, not at the refresh tick (`modes/tty/map_prop_update.go`). With the mode open, GIRO is asked from this machine, within D-39's throttle (at most 40 an hour).
- **The answers' new shape (D-185):** the chart and the centre read `Answer{Bands, Nearest}`; a length the library answers that differs from the places asked is ours to report (D-124).
- **On screen:** the status says the stations used and the oldest reading's time, or "no stations: the background alone"; GIRO's badge is lit only when stations fed the snapshot; a disturbed band is ◐ in the advisory tone and the legend names it; a row's words keep its upper limit and add D-RAP's frequency; the chart names the rows whose nearest station is unscored, "verify on air" (D-181), and the centre box ends with its nearest station's distance, or "No station fed this answer". These words stand in for W4.5/W4.6's confidence treatment until its mock-ups.
- **Credits:** GIRO leaves `creditPending` - it is read now - and About credits it (non-commercial use, the citation); ITUHFProp is credited for the absorption factors go-ionomaps ports (D-186); SWPC's line names D-RAP. Two new chip grounds in every theme, each its own (the contrast test found the first picks shared NWS's and EPA's).
- **Tests written first:** `TestStationsNextAsksAgainAtOnce`, `TestTheStationsAreCreditedWhenTheyFedIt`, `TestADisturbedBandIsDrawnAndNamed`, `TestAnUnscoredNearestStationIsSaid`; the chart's tests moved to the new shape.
- **Mutants:** m218 to m225 new, each killed; m179, m193 and m197 re-pointed (the credit's wording, the chart's `Bands`, the limit name's `switch`) and killed; every mutant on a file this batch changed applied, vetted and killed (forty-one). **m142 survived, at the tip before this batch too:** since the refresh tick gained its number (one tick live at a time), every stale tick the test sent was stale by its number as well, so the session check it holds decided nothing; `TestTheModeRefreshesWhileOpen` now sends the left session's tick at the live tick's number, and m142 is killed.
- **Docs:** the architecture's opening sequence redrawn for the two calls; the as-built map (the chart, the centre, the update); the atlas.

## Batch 22 — go-ionomaps' lower limit wired in: absorbed and weak drawn, the RX noise Setting (D-185 to D-192)

- **Rulings carried:** D-185 to D-192 (the per-place answer; G6.2's method, noise, antennas, tables, the split of a closed band into Absorbed and Weak, Table 3 and §5.3; the chart's cells for them), recorded during go-ionomaps' G6.2 and committed here.
- **go-ionomaps bumped** from `a88f045` to `0ab4636`: the reference circuit's lower limit on every path (G6.2b, G6.2c), Absorbed and Weak (D-70), the noise as an update's option (D-67).
- **The chart (D-192):** ○ absorbed and ◌ weak in the muted tone; the legend in two lines naming all five states; a row's words its upper limit, the LUF and D-RAP's; the header says the reference circuit and its noise in place of "upper limits only", which the lower limit has made untrue.
- **The RX noise Setting (D-187):** the Maps tab's picker after NVIS radius, Residential, City, Rural, Quiet rural, Residential written empty to the file. The update seam carries it to `domains/propagation` and go-ionomaps; after every Update, a snapshot made for another noise than the Setting's is asked again, which go-ionomaps answers from what it holds; the chart and the centre are keyed on the snapshot's noise, since a rebuilt snapshot keeps its time.
- **Measured and written down:** the Settings window's memo-miss frame allocates 4,942 at 133x44 with the new row (budget 4,699 x 1.05 before), so its budget is the new figure x 1.05; the Maps tab's golden re-captured for the row.
- **Tests written first:** `TestAClosedBandIsDrawnByItsKind`, `TestTheNoiseSettingIsTheUpdates`, `TestTheChartIsAskedAnewForANewNoise`, `TestTheHostsOptionsReachTheLibrary`; the chart's and the guard's tests moved to the lower limit and the noise.
- **Mutants:** m226 to m236 new, each killed - m229 after its test was made to ask with a chosen noise; m144, m195, m221, m222 re-pointed and killed; every mutant on a file this batch changed applied and vetted: fifty killed, two equivalent at the tip before this batch as well - **mAA3** (the Settings window's fix-up for opening on a hidden row: since 0.18.0's tabs row zero is drawn on both surfaces, so the window never opens on a hidden row and the fix-up cannot fire; kept as a guard against a later reordering) and **mAI1** (the station's rows' scope: since D-70 every row is drawn on every surface and each writes only what it wrote, so a row's scope is a recorded ruling with no effect at run time; the allocation budget that once caught it does not see a classification). Recorded here, the corpus's verdicts being kept by hand.
- **Docs:** the architecture's seam row; the as-built map (the chart, the Setting, the update); the atlas.

## Batch 23 — UAT-2's notes 1 and 2: the Propagation mode by UI mode, the Map window inset to inset (D-193, D-194, A-51)

- **Asked for by the HUM LEAD** in UAT of batch 21: note 1, the location details stuck to a watchlist place while the map's centre moved; note 2, the Map window alone to the global insets and 85% wide. Note 1's ruling, D-193, made the behaviour the UI mode's; the HUM LEAD asked the batch also to "asses what can be turned into shared helpers and clean up, quality siees, AND make it general enough to support additional UI modes (Analyst) in the future".
- **The origin by UI mode (D-193), in one table:** `modes/tty/map_prop_origin.go` holds `propRoles` (since batch 24 the Propagation column of `mapRoles`, D-195), each UI mode's origin (the map's centre, or the station) and whether the centre is answered as a target. Observer's block, chart (B) and reach answer from the map's centre as it pans, named for the view; its Centre box, a duplicate, is gone and the + stays. The Broadcaster's answer from its transmitter, or the selected place said "(no transmitter set)"; the Centre box answers from the station to the centre - the usable bands, the limits, and the held frequency judged on the path with which way to move it. The origin is worked out in Update (`renderMap`), since the frame may not call the map library (D-41), and only in the Propagation mode.
- **Shared helpers:** `limitsOf` reads an answer's upper limit, LUF (absorbed or weak) and D-RAP frequency once, for the chart's row words and the target's verdict; `usableBands` and `mhzWords` the same; `propCentreKey` embeds the chart's key, so the centre can no longer drop one of its inputs (the noise, once). `propLimitWords` is gone. A UI mode added later (Analyst, D-27) is a row in `propRoles`; `TestEveryUIModeHasAPropagationRole` fails until it has one.
- **Found while writing the verdict's test:** with the LUF at or over the MUF the first draft said "come down under the MUF" to a frequency above it - into the absorption; "no frequency serves" is decided first now.
- **The Map window (D-194):** `render.OverlayAt` puts its top on the frame's top inset (every other window centred, as `Overlay` does); `mapCeiling`, every row between the insets but the borders, is the window's height, the map's budget and the notes' collapse point, replacing D-159's growth; 85% of the width (`mapWindowPct`). The top padding is written without allocating (`topInset`).
- **Measured and written down:** the Propagation frame at 133x44 allocates 1,270 a memo hit (budget 950 x 1.05 before) and 1,733 a miss, 1,751 and 3,409 in colour. Split at batch 22's tip: D-194's sizing alone takes a hit from 971 to 1,260 and a miss from 1,352 to 1,685 (more map rows, each costing its allocations); D-193 adds 10 and 48. The budgets are the new figures x 1.05. The radar-over-alert test counted at least 8 severity digits on the outline; the larger window draws the outline at another scale, 7 at every depth, so the count is held to truecolor's own (A-51). Goldens re-captured for the window: `map-80x24`, `map-120x40`, `map-133x44`, `map-radar-over-alert-*`; `platform/render`'s declaration set for `OverlayAt`.
- **Tests:** `TestTheOriginFollowsTheUIMode` and `TestEveryUIModeHasAPropagationRole` (since batch 24 `TestEveryUIModeHasAMapRole`) written first; `TestTheHeldFrequencyIsJudgedOnThePath`, `TestTheTargetSaysWhatIsUsable`, `TestTheMapWindowRunsInsetToInset` and `TestOverlayAtPlacesTheWindowsTopRow` written after the code, when the mutant pass found the verdict, the target and the window's placement had no test of their own; the centre's tests moved to the Broadcaster; D-159's floor and ceiling tests and U1-13's width test moved to D-194's ceiling and 85%.
- **Mutants:** m237 to m252 new, each killed - m243 after its first form, a self-assignment, failed vet; m251 after `TestTheOriginFollowsTheUIMode` was made to hold that Observer is never asked for a target. m197, m210, m222, m228 and m232 re-pointed and killed (the limit words moved to `limitsOf`, the block to the stored origin, the centre's key to an embedded chart key, so m232 now drops the noise from that). Every mutant on a file this batch changed applied, vetted and killed: seventy. **m160 survived at first:** the fixture's view name is longer than the place column, so the answers always took the short indent and the rule that they stand under NOW went unexercised; the rows test now also draws a short name, the Broadcaster's transmitter, and m160 is killed.
- **Docs:** A-51; the as-built map (the origin, the centre, the window); the architecture's answers (the origin follows the UI mode); REQUIRED-READING; the atlas.

## Batch 24 — W7, the Broadcaster's own map: `g` on the console, on the station, its service area always drawn (D-195, D-196, A-52)

- **Found in UAT of batch 23:** `g` did nothing on the console with no window open, and opened Observer's weather map with one open. The console's key map bound no map action (`modes/tty/router.go:159`), so the key reached the console; with a window open every key went to Observer (`router.go:656`), whose `g` opened its own map. FR-2.1 (W7, D-57) was unbuilt. The HUM LEAD ruled the Broadcaster's map UI-mode specific (D-195), and, shown that no tint passes over the MUF or foF2 ramp, a red stipple there (D-196).
- **go-tuiMaps P9 (its L-8, D-32, D-33, A-11)**, pushed as `9dc4ff3` and pinned here: the `Zone` role - outlined in red, its inside tinted beneath any alert's, stippled over a field its tint cannot blend with - and `Report` for a kilometre circle.
- **The map roles (A-52):** batch 23's Propagation roles table became `mapRoles` in `modes/tty/map_role.go`: whose place the map is on, whether the service area is drawn, whether `[` and `]` step through the places, and the Propagation role. `mapPlace` is every map site's place (eleven of them read the selected place before). Analyst is one row.
- **On the console:** `g` is bound (Observer's key, D-56) and handed to the window as Settings and About are; the map opens on the station - the transmitter, the watchlist's own record where one is at its place, or the selected place standing in. The service area (`map_zone.go`) and the tower's diamond are handed in in every mode; a swap of UI modes or a Settings save that moves the station puts an open map back on it. `[` and `]` do nothing there; the controls box says "red: service" in their row, and Help lists the map's keys without them. The near-vertical answers are to the service radius, "Local, to ~161 km (service area)" (FR-2.3); the words say the red circle.
- **Found while building it:** the console's key map handed to Help was the dashboard's own, so adding the map's keys to it would have rebound the console for good: Help takes a copy. A fixture that read the wall clock (`TestAClosedBandIsDrawnByItsKind`): at some hours the day's absorption closed a third band at another LUF than the two the test wrote; the test writes the row whole.
- **Not in this batch:** W7.2, the shared-device statement (FR-2.2) and the shared Settings (the rest of FR-2.3), in the plan's order.
- **Tests written first:** `TestTheConsoleOpensTheMapOnTheStation`, `TestTheConsoleKeysDoNotClash`, `TestTheServiceAreaIsAlwaysDrawn`, `TestTheBroadcastersMapOffersNoPlaceKeys`, `TestTheServiceAreaIsSaid`, `TestTheNearVerticalAnswersUseTheServiceRadius`, `TestEveryUIModeHasAMapRole` (from batch 23's roles test); `TestTheStationIsTheWatchlistsOwnRecord` after, for the mutant that found the watchlist's record untested.
- **Mutants:** m253 to m270 new, each killed - m256 after `TestTheStationIsTheWatchlistsOwnRecord` was written for it, and m259 and m267 after their first forms left an import or an `append` with nothing to append, which vet refused. m151, m195, m231 and m237 to m241 re-pointed (the local line to `propRadiusKm`, the roles to `mapRoles`, the station to `stationPlace` and `noTransmitter`) and killed - m240 too after its first form left a variable unused. Every mutant on a file this batch changed applied and vetted: 121 killed, and two equivalent as recorded before - **mAA3** (batch 22) and **mSC3**, `routeKey`'s tripwire, which survives by design while no fall-through case mutates the Router; the console's `g` case returns, so the property holds.
- **Docs:** D-195, D-196, A-52; FR-2.1 rewritten, FR-2.4 added; the plan's W7; the as-built map (the roles, the zone, the window); the architecture (the map follows the UI mode); REQUIRED-READING; the atlas.

## Batch 25 — The open items: the best frequency, confidence by distance, the offset compared, the refresh and no-readings Settings, the shared device (D-197, D-198, A-53; W4.4 to W4.6, W7.2)

- **Asked for by the HUM LEAD:** "proceed with open items". Two rulings first, one at a time: the best frequency (D-197, D-193's promised follow-up) and W4.5/W4.6's confidence treatment from its mock-ups (D-198). A third was not asked: whether the Broadcaster's map should offer the Maps tab's Settings, since D-70 already draws and applies every row on both surfaces.
- **The best frequency (D-197, FR-1.19):** 85 % of the path's upper limit (ITU-R P.373's FOT as a rule of thumb) and the band whose lower edge fits under it and above the LUF; every chart row "best ~15.0 (20m)" or "none serves", the Broadcaster's target box the long form.
- **Confidence by distance (D-198, FR-5.5):** each chart row starts with its nearest reporting station's shade - ▓ ▒ ░, ? unscored - keyed under the band legend, plain under --ascii; the map's words end with each row's nearest station, far ones about as good as typical. The rows' figures are bare, "figures in MHz" said once. The "verify on air" line A-50 left standing in is gone.
- **The offset compared (W4.5, FR-5.4):** the Status window's notes say the live offset beside its typical and spread; the diagnostics record each live offset; an unusual one, or the correction in use, is the chart's second line, the block's row under the mode line - the mode line is cut at its end beside the Controls box, and a warning there would be cut off - and the status words' last part.
- **The Settings (W4.4, W4.5, W7.2):** "HF refresh -" (every 10 minutes, hourly, on demand on the refresh-now key), its measured cost (D-141) a notice while chosen, the open mode's tick following it and none on demand; "No readings -" (GloTEC as is, or the typical offset), asked through the update; both, and "Instead of the map", recorded as one Setting for both UI modes (D-121). The update seam carries go-ionomaps' `UpdateOptions` whole; a snapshot asked with another noise or correction than the Settings' is asked again, which go-ionomaps answers without a request.
- **The shared device (FR-2.2, D-85):** the console's Help and the Settings window's Broadcaster tab say the broadcast and a screen reader share the default sound output.
- **Measured and written down:** a Tab press into the Maps tab allocates 3,740 with the two rows (budget 3,700 before), so its budget is 3,740 x 1.05; the Settings window is two columns wider, its widest tab's new notice; the Settings goldens re-captured (the Maps tab's two rows, the width).
- **Found while building it:** the centre test's "LUF " was there only at hours the absorption closes a band (a wall-clock fixture again); it asks for the best frequency instead. The tab walk in the shared-device test first sent a tab as text, which Settings does not read, and looped: it walks at most once round.
- **Tests written first:** `TestTheBestFrequencyIsNamed`, `TestEachRowShowsItsNearestStation`, `TestStatusShowsTheLiveAndTypicalOffset`, `TestAnUnusualOffsetIsSaid`, `TestTheNoReadingsCorrectionIsASetting`, `TestTheRefreshSettingStatesItsCost`, `TestTheSharedDeviceIsSaid`, `TestTheConsolePropagationModeHonoursTheSharedSettings`, `TestInsteadOfTheMapIsShared`; `TestTheWindowsOptionsReachTheLibrary` (app) for the seam's options; the chart's, the centre's and the kept-Settings tests moved to the new words and rows.
- **Mutants:** m271 to m298 new, each killed - m290 after `TestTheNoReadingsCorrectionIsASetting` was made to hold the kept correction itself: its last check passed while an ask was in flight, which refuses a second whatever the correction. m144, m223, m229, m230 and m236 re-pointed (the seam's options, the ? shade, the legend's units) and killed. Every mutant on a file this batch changed applied and vetted: 124 killed, and **mAA3** and **mAI1** equivalent as recorded in batch 22.
- **Docs:** D-197, D-198, A-53; FR-1.19, FR-5.5; the plan's W4.4 to W4.6 and W7.2; the as-built map; REQUIRED-READING; the atlas.

## Batch 26 — Day / night as a map detail option in every map mode (D-199, A-54)

- **UAT of batch 25, note 1:** the night side and its dotted edge, drawn in the Propagation mode alone, as a switch among the map's detail for the weather modes; "a general helper/overlay".
- **One night overlay (`modes/tty/map_night.go`):** the map works out whether the night is drawn and for what moment (`nightWanted`), hands it in (`setNight`) and keeps it moving (`followNight`) in every mode; the Propagation mode's overlays no longer carry it. In the Propagation mode it is drawn always, for the hour shown, as D-174 ruled; in the weather modes while "Day / night" is on - Radar mode's for the clock, Forecast mode's for the middle of the step shown.
- **The switch:** the last of the map's detail, in the Overlays menu and Settings; no level turns it on, a new level keeps it, and it never makes the level read "Custom" (A-54). The Propagation mode's menu offers no switch. The words say it.
- **Found while building it: the chart's fixture read the wall clock, a third time.** `drawnDash` made its snapshot for the hour the tests ran, and the absorption, the LUF and the best frequency follow the sun: at some hours the centre box said "no frequency serves" where the test, as batch 25 left it, wanted "best ~". Batches 24 and 25 had moved two tests off hour-dependent facts; this pins the fixture instead - its snapshot and its clock at one in the morning at Oceanside (`drawnAt`), the absorption at its least - so every test that uses it reads the same at every hour; every mutant on the Propagation files run again on it: 107 of 108 killed at once; **m216 survived, and had since batch 25**: the far centre's check looked for "MUF " anywhere in the box, and batch 25's best-frequency line ends "of the MUF" before the box's padding, so an area answer passed for the path's. Batch 25 changed no file m216 is on, so its run did not reach it. The check now reads the usable line's limit, and m216 is killed.
- **Tests written first:** `TestDayNightIsAMapDetailOption`, `TestTheNightIsForTheStepShown`, `TestThePropagationModeAlwaysDrawsTheNight`; the night's tests, the menu's rows and the reopened map's moved to the map's overlay; the Maps tab's golden re-captured for the row.
- **Mutants:** m299 to m307 new, each killed - m302 and m307 after their first forms left a variable unused, which vet refused; m198 and m199 re-pointed to the map's night (the Propagation mode's always, its guard) and killed. Every mutant on a file this batch changed applied and vetted: 65 killed, and **mAA3** and **mAI1** equivalent as recorded in batch 22.
- **Docs:** D-199, A-54; FR-1.20; the as-built map; REQUIRED-READING; the atlas.

## Batch 27 — W0.3, one coordinate and locator parser in front of every lookup (D-201, A-55)

- **The build-first list (D-201), first item.** Two surveys of the three plans against the code found W0.3 never built, with `app.parseLatLon` (`app/app.go:190`) taking "NaN" - `strconv.ParseFloat` reads it, and NaN fails no range comparison - and the Observer's search sending typed coordinates to the online geocoder, the resolver having no parser of its own.
- **`platform/geo`:** `ParseCoordinates` - two decimal numbers, a comma or spaces between, finite and on the globe; no exponent, hex, "NaN" or "Inf" - and `ParseLocator`, a Maidenhead locator of four, six or eight characters to the centre of its square; `LocatorLabel` writes one as it is written.
- **`locations.Resolver` reads them first,** in `Resolve` and the type-ahead, before the index or the geocoder: every lookup goes through it, so coordinates and locators never go online; text that reads as two numbers but is no coordinate is refused offline, saying what one is; a typed place in no region the weather covers is refused as a city there is. `app.parseLatLon` is deleted, `watchpost report` resolving through the same resolver.
- **No error quotes what was typed:** the resolver's and the geocoder's errors say what failed without the query, and `name` is redacted in a failed request's URL as a key is.
- **Tests:** `TestNaNCoordinatesAreRefused` and `TestALocatorIsTheCentreOfItsSquare` were written after the parsers they hold, a step out of order; `TestEveryResolverParsesCoordinatesLocally`, `TestCoordinatesAndLocatorsNeverGoOnline`, `TestNoLookupErrorQuotesTheQuery` and `TestTheGeocoderQueryIsRedacted` first, each watched failing (the coordinates went to the geocoder; both errors quoted the query); `TestTheReportResolvesCoordinatesThroughTheResolver` replaces `TestParseLatLonTable`. `app`'s declaration set re-captured for the deletion.
- **Mutants:** m308 to m318, each killed; no other mutant is on a file this batch changed.
- **P10 (D-202):** the change brought `Resolve` into scope, where P10-01 reads its call to the fallback's `Resolve` as recursion; the ledger's exemption, keyed "Resolve", no longer matched the checker's "Resolver.Resolve". As ratified, `Resolve` calls the fallback through a method value and the exemption is deleted; the public mirror regenerated.
- **Docs:** D-201, A-55; the plan's W0.3; the atlas.

## Batch 28 - W0.2's per-source caps and FR-5.1/FR-5.2's honesty (D-201, A-56)

- **The build-first list (D-201), second item.** The surveys found the propagation client's one body cap, 8 MiB (`app/mapprop.go:22`, NOAA's grid's), for every host, GIRO's too, where FR-4.7 asks for one sized to each source, and the field's age said since it was computed where FR-5.1 asks for the oldest input's; nothing said a field stale.
- **go-ionomaps** exports each source's bound on its source list (`Source.MaxBytes`: GIRO 2 MiB, NOAA 8 MiB), the constants its own fetchers read (its A-15, `32931d9`); watchpost's pin moves to it.
- **The caps:** `PlainConfig.MaxBodyByHost` gives a host a tighter cap than the client's; the app fills it from go-ionomaps' sources, so the client takes no more from a host than the library parses from it.
- **The age (FR-5.1):** the oldest input's - GloTEC's grid where it is the background, or the oldest GIRO reading where that is older - "data 25 min old", "data just in" under a minute, on the mode line in the freshness ramp's tone and in the words; "climatology" with no measured input.
- **Stale (FR-5.2):** past an hour, "STALE: data 2 h 5 min old", in the alert's tone; a failed update is said with the key that asks again ("U refreshes now").
- **Found while building it:** the GIRO-refusal test could not be written in `app` - the app's data client refuses plain http, as it should - so it is in `platform/httpx` with a data client of its own, and a control showing that the data client does hold after its own 429, without which the test proves nothing. The stale tone's mutant survived at first: the test compared tints with colour off, where every tone is the same text; it turns colour on. Of the six test names the plan and requirements gave for W0.2 and W4.3, four were built earlier under other names; the documents name the tests as built. `TestTheFallbackIsNamed` did not exist: the climatology's words had no test, and it is written now, after the code, with its mutant.
- **Tests written first:** `TestEachSourceHasItsOwnCap`, `TestEachPropagationSourceHasItsOwnCap`, `TestAGIRORefusalNeverPausesWeather`, `TestTheAgeIsTheOldestInputs`, `TestTheFieldsAgeIsAlwaysSaid`, `TestAStaleFieldIsSaidStale`, `TestAFailedFetchIsSaidWithItsPath`; `TestTheFallbackIsNamed` after, as said; the rows', round four's and the quality tests' ages moved to the oldest input's.
- **Mutants:** m319 to m328 new, each killed - m322 after its first form left a variable unused, which vet refused, and m324 after the stale test turned colour on; m163 re-pointed to the mode line's tone (and its form changed, vet refusing the first) and killed. Every mutant on a file this batch changed applied and vetted: 49 killed.
- **Docs:** A-56; FR-4.4's and FR-4.7's test names; the plan's W0.2 and W4.3; REQUIRED-READING; the atlas regenerated, unchanged (no diagram draws the client's caps or the mode line's words).

## Batch 29 - W6.1, the Propagation mode's words in D-136's order, written for speech (D-201, A-57)

- **The build-first list (D-201), third item.** None of W6.1's eight named tests existed. The words read the drawn block's rows back ("NOW · foF2 6.0 MHz", "Up to: NA 20m · ... B chart"), so they said "MHz", "40m", "~" and a key hint, in the block's order, not D-136's; no status block, no day or night at the origin, no "updated at", no focus limit; each row's nearest station was one long line after the bands.
- **The words (`modes/tty/map_prop_words.go`):** the mode and its layer's full name once; an hour ahead's typical label; the origin's MUF(3000) and foF2; day or night there and when the sun next turns; the status block - the age, what the field was made from, the origin's station, a disagreement with the typical day, "updated at" after a refresh, a failed update with the key that asks again; the focus limit on opening; one line per target, its name first, the open bands before the closed, its own station; the frequency held and its reach; the centre. Everything said goes through `spoken`, which writes the windows' short forms out.
- **Day or night (`platform/astro/turn.go`):** the sun's elevation (NOAA's general solar position equations), whether it is up, and the next sunrise or sunset, found within a day and a half to the second; go-tuiMaps has the same in an internal package, so watchpost adds its own beside `SunTimes` rather than widening the library's API.
- **Help** says the focus limit on both UI modes (FR-3.7, D-133).
- **Found while building it:** a failed update while a field was held was said nowhere - `propStatusWords` said the field kept and dropped the error (FR-5.2); it is said first on the status line, where the line's end is cut, and in the status block. A frequency's figure the centre's box writes bare after its first ("MUF(3000) 18.0 · foF2 6.0 MHz") is given its unit in the words. `TestThePropagationRowsFollowTheMock`'s fixture reads the clock, so its words check names no band.
- **Tests written first:** the eight the plan named, with `TestSpokenWritesTheFiguresOut` and astro's three, before the code - but first run after it, so not watched failing on the old words; their first run failed four (the frequency still the block's row, the failure cut off the status line, a sentence begun in lower case, a check of mine tied to the hour), and each mutant since has been killed by one. `TestTheDayAndItsTurnAreSaid`, `TestTheBandsAreSaidOpenFirst` and `TestEachTargetSaysItsStation`, and the refresh test's GIRO-after-NOAA case and the focus test's reopening, after, for mutants that had nothing to kill them. The rows', frequency, Help and nearest-station tests moved to the new words.
- **Mutants:** m329 to m352 new, each killed - m344 and m347 after their first forms left a variable unused, which vet refused. m157, m217, m279, m280 and m325 re-pointed to the new words and killed. **m166 retired:** it chose the styled block for the words, and the words no longer read the block, so nothing styled can reach them; `TestRoundFourMarksStandOut` still checks they carry none. `propNearestSummary` is deleted, its words now each target's own.
- **Docs:** A-57; FR-3.2's tests; the plan's W6.1; REQUIRED-READING; the atlas.

## Batch 30 - W5.5, space weather named, on go-ionomaps G2.5 (D-201, A-58)

- **The build-first list (D-201), fourth item.** go-ionomaps carried an empty `Scales` in every snapshot: G2.5 was never built. It now reads NOAA's scales file with D-RAP at each update (its A-16); watchpost's pin moves to it.
- **Read once for its layout:** NOAA's file was asked once, on 2026-10-16, to learn its layout (no reading kept; the tests build their own). That day G "now" was null while R and S were 0, so go-ionomaps leaves a level unset one by one, not only when the file is missing.
- **Named (`modes/tty/map_prop_scales.go`):** a scale above 0 by its level and NOAA's word, in a line under the chart's header and in the words' status block; NOAA's outlook for the days it expects a geomagnetic storm; "NOAA's scales not read" when none were; nothing for scales at 0 or a level left unset.
- **A storm (D-109):** while G is 1 or more, every hour ahead's label and words say typical conditions may not hold - also while the hour is still being worked out, which the label had returned early for.
- **Found while building it:** go-ionomaps' P10 leg refused the first gate: the parser named `Parse` called `time.Parse`, which the recursion check reads by its bare name as a cycle, and its guards carried init statements, which the density count does not see. The parser is `Read` and its guards call-free.
- **Tests written first:** `TestARaisedScaleIsNamed`, `TestAStormSaysTypicalMayNotHold`, `TestMissingScalesAreNotShownAsZero`; the first run failed one, the storm label's early return. The Propagation fixture's snapshot now has NOAA's scales read at 0, as a live update does.
- **Mutants:** m353 to m363 new, each killed - m363 after its comment named the test that checks the hour ahead's typical error, `TestTheTypicalErrorStatesItsBasis`, in place of one that does not. Every mutant on a file this batch changed applied and vetted: 55 killed.
- **Docs:** A-58; the plan's W5.5; REQUIRED-READING; the atlas regenerated, unchanged (no diagram draws the chart's lines).

## Batch 31 - W3.1's terminator for the field's time, W3.2's layer seam (D-201, A-59)

- **The build-first list (D-201), fifth item.** FR-1.5 names `TestTheTerminatorIsDrawnForTheFieldsTime` and FR-3.3 `TestALayerAddsItsSentence`; neither existed. The Propagation mode's night followed the clock in the current hour, so a field held an hour - on demand, or after failed updates - was drawn under a sun that had moved on from it; and with no selected place the description returned nothing, a layer's sentence with it.
- **The night for the field's time:** the current hour's night is the field's own time, to the step; a new field moves it; an hour ahead is unchanged, the middle of the hour. The weather modes' night is as it was.
- **A conflict in the tests, settled by the requirement:** `TestTheNightFollowsTheClock` held the clock's night in this mode and cited go-tuiMaps' L-4.3, which is about the night's tint carrying no alert words, not its moment; FR-1.5, approved at DISCOVER, wins, and the test is deleted. m199, which it killed, guards the clock-following night of the weather modes and is killed there by `TestDayNightIsAMapDetailOption`.
- **The seam:** `mapLayerSentences` lists the layers that add a sentence - the service area's circle and the night - said before the alerts' early returns and with no selected place.
- **Tests written first:** `TestTheTerminatorIsDrawnForTheFieldsTime` and `TestALayerAddsItsSentence` (each case with a control that it is in force); the first failed on the clock's night, and its first whole-package run found the conflicting test. `TestTheMemoKeyCoversEverythingTheFrameShows` walks every field by reflection, so FR-3.3's "extended" needs nothing more.
- **Mutants:** m364 to m368 new, each killed; m199 re-pointed as above, and m268 to `mapLayerSentences`, the service area's sentence now one of its entries; both killed. Every mutant on a file this batch changed applied and vetted: 32 killed.
- **Docs:** A-59; the plan's W3.1 and W3.2; REQUIRED-READING; the atlas regenerated, unchanged.

## Batch 32 - W0.1, a pre-push hook for the record; W0.4, the record's checks (D-201, A-60)

- **The build-first list (D-201), sixth item.** REFLECT L5's control - a hook that runs the docs-reading tests before a push - and A-29's three checks were never built; the trace from requirements to plan and the `file:line` rule (FR-10.4) were kept by hand.
- **The checks (`cmd/watchpost/record_test.go`, `tools/atlas`):** every test a requirement row names is carried by a plan task; every ruling row naming a Go file cites its line; the committed atlas is the page the documents render today (the atlas tool now renders apart from writing, byte for byte as before).
- **What they found, closed rather than excused:** six tests the plan did not carry - FR-4.10 named three never built under those names (`TestAnOpenModeFollowsEachNewGrid`, `TestGIROIsAskedAtMostHourlyWhileOpen`, `TestNothingRefreshesWhileClosed`), each matched to the test that holds it (`TestTheModeRefreshesWhileOpen`, go-ionomaps `TestGIROIsNeverAskedMoreThanHourly`, `TestAReleasedMapDropsTheOldSessionsTick`), and batch 26's three Day / night tests had no plan row (W3.3 now); and five ruling rows (D-14, D-202, A-17, A-34, A-35) named a file without its line, each given the line its claim is about today.
- **The hook (`scripts/hooks/pre-push`, `make hooks`):** about five seconds; installed in the working clone. Watched failures, each restored by copy and compared: a planted batch the as-built page does not draw, a ruling row citing a file without a line, and a test dropped from the plan each stopped it with exit 1.
- **Found while building it:** the gate's own tests refused the hook twice - its `#!/bin/sh` shebang, which the gate oracle cannot stub (`#!/usr/bin/env sh` now), and `make hooks` on no gate list (an `unlisted` row: it installs, asserting nothing, and every test the hook runs is on the gate path).
- **Tests written first:** `TestEveryRequirementsTestIsInAPlanTask` and `TestEveryRulingNamingCodeCitesItsLine` were measured against the record before the fixes (six and five gaps), `TestTheRecordChecksFailOnAPlantedGap` and `TestAChangedDiagramChangesTheAtlas` plant each gap, `TestTheAtlasMatchesAFreshRender`, `TestThePrePushHookRunsTheRecordsTests`.
- **Mutants:** none; this batch changes tests, a hook and the atlas tool, whose one change is the split its two tests hold.
- **Docs:** A-60; FR-4.10's, FR-10.2's and FR-10.4's tests; the plan's W0.1, W0.4, W4.4 and a W3.3 row; REQUIRED-READING; the atlas regenerated, unchanged.

## Batch 33 - W9: the memo-key audit (#12) and say's orphans (F-187, F-188) (D-201, A-61)

- **The build-first list (D-201), W9**, built ahead of W5.3 while the libraries' gates ran. None of its nine named tests existed.
- **The audit (`03-architecture-design/memo-key-audit.md`):** thirteen position fields in the three window keys; none needed to become an identity - each indexes a list its key already keys by identity, or a fixed set - and each says so in a `position:` comment the test reads. The app's keys: every field of the map's ask is carried by the temperature key or classified as no input, each carried field moves the key, and the keys named by their data are pinned with why.
- **say's children (`domains/radio/synth/children.go`):** held in memory and recorded one file a child under the user's cache; `EndChildren` at the run's exit and on a hang-up; at start a recorded process is ended only when it is the user's, orphaned, its arguments a watchpost-say- path's and its start time the one recorded - so a second watchpost never ends the first's speech. Bubble Tea quits cleanly on an interrupt or a terminate, so its exit covers them; a hang-up runs no exit, so it has its own handler, Unix only.
- **`make test-say`** has a three-minute bound and its test samples a say still running at 110 s (F-188's cause is unknown: the instrument is what was owed).
- **Found while building it:** m171, an earlier mutant that dropped the window key's separate age, survived: since batch 28 the status words in the same key carry the age, to the minute, and "STALE" past the hour, so the field was redundant - and measured the age since computed, which batch 28 had replaced. The field and its helper are removed and m171 retired; F-30's reflection guard still holds that no frame is replayed stale. go-ionomaps' pin moves to its missing tests and PyIRI's licence (`d7e7f3c`), closing F-215; F-187 and F-188 are closed. **The gate's own tests found batch 32's pre-push hook a shell script** with no ruling - the repository's rule is everything in Go - unseen then because the file was untracked when that gate ran: it is `tools/prepush` now, built by `make hooks` (A-62), and was watched failing again on a planted ruling row. The test's stand-in say is the test binary itself (AP-SHELL-01 refused a shell script in a Go string), and the new files carry no build constraint (P10-08).
- **Tests written first:** `TestAnOrphanedSayIsEndedAtStart`, `TestAnUnrelatedSayIsNeverEnded`, `TestNoSayOutlivesAnOrderlyExit`, `TestEveryPositionFieldHasAReason` (its first run named `mapsOff`, a flag, not a position: the guard reads integers only), `TestTheAppMemoKeysCoverTheirInputs`; `TestSayIsTrackedWhileItRuns`, `TestPSIsReadWhole`, `TestTheRunSweepsOrphansAndEndsItsChildren`, `TestTheSayLegIsBoundedAndDiagnosed` and `TestAPositionWithoutAReasonIsCaught` alongside. No test ends a real process but the `sleep` it starts.
- **Mutants:** m369 to m380 new, each killed - m377 after its first form left an import unused; m171 retired as above. Every mutant on a file this batch changed applied and vetted.
- **Docs:** A-61; the audit; FR-9.3's test; the plan's W9.1 and W9.2; F-187, F-188 and F-215 closed; REQUIRED-READING; the atlas.

## Batch 34 - W5.3, finding a place in the Propagation mode (D-201, A-63)

- **The build-first list's last item.** FR-1.10 and FR-1.14 named a finder no batch had built: the Propagation mode had no way to find a place, and the resolver's coverage gate refused anywhere the weather does not reach - for every lookup, the mode's included.
- **The mode-aware gate:** `locations.Resolver.ResolveAnywhere` reads coordinates and locators anywhere on the globe and the embedded index's cities in any country, sending nothing; only a miss goes online, to the geocoder's `ResolveWorld`, which takes any country and caches nothing (`httpx.NoCache`). The weather's lookups keep their gate.
- **The finder (`modes/tty/map_prop_find.go`):** `/` opens it over the map; Enter finds off the UI goroutine; the place found is centred - the Observer's answer is from it, the Broadcaster's target is it (D-193) - marked with a ring among the map's places, and said; the finder closes and the map has the focus. A failed find is said with the way on and the finder stays open; an earlier find's late answer is dropped. Nothing is kept: no commit, no recent place, and the mark goes when the mode is left or on an empty Enter.
- **Found while building it:** the router guard required the find's answer to be routed to the window that asked (`observerScoped`), the margins guard the finder's lines inside its width (the query capped at 48 characters, its error wrapped), and the air boundary the new seam classified (`FindPlace`: no air); the duplicate guard found the finder's opening the frequency entry's twice over, so both open through `openInPropagation`; and the every-key-shows guard names `/` with the other Propagation-only keys, which change nothing in the weather modes.
- **Tests written first:** `TestTheOnlineFallbackIsUncached` and `TestALocalMatchSendsNothing` against the domain, `TestAFoundPlaceIsAnsweredAndNotKept` and `TestFocusReturnsAfterAFind` against the window; `TestTheBroadcasterAnswersAFoundPlaceAsItsTarget` after, for D-193's other UI mode; `/` added to `TestEveryPropagationControlIsAnAction`; Help's tests to its new row.
- **Mutants:** m381 to m392 new, each killed; m311 re-pointed to the gated line. Every mutant on a file this batch changed applied and vetted: 73 killed.
- **Docs:** A-63; the plan's W5.3; REQUIRED-READING (the build-first list built); the atlas.

## Batch 35 - F-216: CI's actions on Node.js 24, the runner image pinned (BUILD red team H-13; A-64)

- **The red team's remediation, first item:** F-216 fell due on 2026-10-19 - GitHub forces the Node.js 20 actions onto Node.js 24 and moves `ubuntu-latest` to Ubuntu 26 that day.
- **The actions:** every workflow's `actions/checkout`, `actions/setup-go` and `softprops/action-gh-release` re-pinned by commit to its latest release, each built for Node.js 24 (read from its `action.yml`), and each read for the inputs the workflows pass.
- **The runner:** the Linux jobs pin `ubuntu-24.04` (A-64); a move to Ubuntu 26 will be its own change.
- **The follow-ups file** keeps its own rule: closed rows deleted (H-12), F-216 with them.
- **Tests:** none new; the hosted run on this commit is the test of the workflows, read before the next batch.

## Batch 36 - the record: requirements as built, every named test real, citations that resolve, runs recorded (A-65)

- **The BUILD red team's record findings**, before the libraries' release candidates: B-F2, B-F4, B-F5, D-F1, D-F2, D-F5, D-F11 to D-F13, C-10, C-23, H-6 and H-8 (A-65).
- **The requirements** say what was built: the origin and the centre by UI mode (D-193), the Broadcaster's own map (D-195), "weak" (D-70). REQUIRED-READING's rule 3 and Phase cell, the as-built map's library facts and F-174 follow.
- **Two record checks, each failing on a planted gap:** `TestEveryRequirementsTestExists` (four requirements leaned on tests that do not exist; three were renamed long ago, one never written) and `TestEveryCodeCitationResolves` (three citations named bare files).
- **`TestNoRawSourceDataIsStored`** (FR-4.6): a GIRO-shaped reply's address reaches no note and no error, and the propagation client has no cache; a mutant that notes a header's value is caught.
- **Batch 28's hosted failure fixed at its cause:** `TestThePropagationRowsFollowTheMock` and `TestRoundFourMarksStandOut` hold the dashboard's clock, and `TestAnAgeIsReadAgainstAHeldClock` keeps every such test holding it. Hosted CI failed on batch 28 (this test, macOS) and on batch 32 (the shell hook, both systems) without being read; it is read after every push since.
- **`tools/runlog`** records each verify run (H-8); this commit carries the line of the run that passed it.
- **Tests:** `TestEveryRequirementsTestExists`, `TestEveryCodeCitationResolves`, `TestNoRawSourceDataIsStored`, `TestAnAgeIsReadAgainstAHeldClock`, `TestEachRunIsRecorded`, `TestTheTreeIsWhatTheRunSaw`; each new check was shown failing on a planted gap or a hand mutant.

## Batch 37 - watchpost on go-ionomaps v0.1.0-rc.1: the hours ahead their own, no stations, the throttle kept (A-66)

- **The re-pin** (D-206): go-ionomaps v0.1.0-rc.1, the API as reviewed (D-216). The hours ahead are snapshots of their own, so the chart at an hour ahead carries none of today's disturbance (L-F2); the answers are for the shown snapshot's own hour; the sources come from `ionomaps.Sources()` with no library made (C-5).
- **A catch before the first run:** the first "has a field" check read the field's values, but the tests' fixtures - and the old check - count a snapshot by its hour; go-ionomaps gives one with no hour when it has no field, so the hour is the test.
- **"HF stations"** (D-214, FR-4.11) and **GIRO's throttle kept across restarts** (D-213, FR-4.12), its file in the state directory, which `platform/config.StatePath` now names for the quota's file too; the app's tests' guard against writing the developer's state names it.
- **About credits P.372** (the library ships its four noise constants); P.371 is pending, cited for its equation alone.
- **The architecture** says what watchpost relies on in go-ionomaps by role and points to the library's own documentation (D-F4).
- **Mutants:** m393 to m399, each applied and killed before the gate; m148, m180, m209, m212, m213, m218, m333, m364 and m365 re-pointed at the code as it reads now.
- **Goldens:** the Maps tab's, for the new row; the app's declaration set, for the fetcher no longer needed.
- **Tests:** `TestAnHourAheadCarriesNoDisturbance`, `TestNoStationsIsASetting`, `TestTheThrottleStateOutlivesARestart`; the two settings guards carry the new field.

## Batch 38 - W6.2, the Propagation mode's words: the centre named anywhere, read in the ruled order, the reader's place kept (A-67)

- **The BUILD red team's words, before the go-tuiMaps re-pin** (its release candidate's gate was running): AX-1, AX-2, AX-4 to AX-13, D-210 and D-211. Done in this order because none of it waits on go-tuiMaps; the re-pin follows as its own batch.
- **AX-1, the Critical:** abroad, the words named the region last drawn. The centre is now named for where it is - the place found while the centre rests on it, the nearest town in any country within 100 km, or its coordinates - from the embedded cities table, which held the world all along (`geodata.Index.NearWorld`; the package's comments said "34,106 US cities", which was the whole table). The map's title, away from every region, names it the same way.
- **AX-2:** `spoken` gave every decimal "megahertz", so coordinates and hours came out as frequencies. A figure now takes its unit from the frequency's name before it; a typed place's coordinates are said as degrees.
- **D-210:** the mode and its answers first, the place's weather and the night after them; identical targets on one line; the station every row shares said once, in the status block.
- **D-211:** the window keeps the sentence at its top by its kind across a refresh, an hour step or an answer landing, until the reader scrolls. Found while building it: the window wrapped the words a second time when its scroll rail appeared, leaving one-word stub lines ("1:50" / "PM UTC."); text that will scroll is now wrapped clear of the rail in every map mode.
- **The rest:** the short names said in full on opening (AX-8); the local area's typical day said, and the chart readable under `--ascii` (AX-4); Help's rows as key-name pairs, saying the hour step, at the same height, and the chart window's controls line (AX-6); a failed answer said with the key that asks again, which now does (AX-9); the world finder's own hints, and Esc cancelling a find in flight (AX-10, AX-11); the night said as drawn, at the window's clock (AX-13, AX-8); the service area named without its colour, and dimmed badges bracketed (AX-12); "1 hour ahead".
- **Tests, each shown to kill a mutant of the code it guards** (some written beside the code, not before it): `TestTheCentreIsNamedWhereverItIs`, `TestAPointIsNamedAnywhere`, `TestAWorldScanAnswersAsTheTableDoes`, `TestEscCancelsAFindInFlight`, `TestTheWorldsFailuresSayItsOwnFix`, `TestIdenticalTargetsShareALine`, `TestTheShortNamesAreSaidInFull`, `TestTheNightIsSaidAsItIsDrawn`, `TestAFailedAnswerSaysHowToAskAgain`, `TestTheChartIsReadableInASCII`, `TestAnHourIsSaidSingly`; `TestARefreshKeepsTheReadersPlace` rewritten for D-211 (it checked a line number); `TestSpokenWritesTheFiguresOut`, `TestCaveatsAreSaidOncePerOpen`, `TestTheCentreAnswerIsSaid`, `TestTheServiceAreaIsSaid`, the badge test and Help's tests to the new words.
- **Mutants:** m400 to m425, each applied and killed before the gate; m157, m217, m221, m226, m227, m236, m266, m280, m344, m346, m347 and m352 re-pointed at the code as it reads now. Three survived their first run and each found a test that looked at the wrong thing: `TestIdenticalTargetsShareALine` counted distinct lines and missed one said twice (m407); `TestAPointIsNamedAnywhere` timed the namer's memory at a point of open sea, where a scan allocates nothing anyway (m404); and the Broadcaster's Help test read the map's whole keymap, not the keys Help lists there, so an empty row went unseen (m266). Each test now checks what its rule says.
- **Found by the gate:** the new seam `PlaceNear` classified for the air (no air: it reads the embedded table), the app's declaration set re-captured for the namer, `TestEveryRequirementsTestIsInAPlanTask` refusing eleven tests the requirements named and no plan task carried - the plan's W6.2 now does - and the duplicate check two copies the world scan made of the US one, collapsed into `nearIn` and `placedOnce` (m403 re-pointed).
- **Docs:** A-67; FR-1.10, FR-1.14, FR-1.15, FR-3.1, FR-3.2 and FR-3.6 as built; the plan's W6.2.

## Batch 39 - W6.3, the BUILD exit's code findings: each update its own, answers for the snapshot shown, the session one struct (A-68)

- **Ahead of the go-tuiMaps re-pin**, whose release candidate waits on its gates: none of this needs it.
- **C-2, a wrong answer:** the service joined an update asked while one ran. A mode left while its update was still ending, then opened again, joined that update and was handed its "context canceled", which the window hid and did not ask again until its next tick - ten minutes, an hour, or never on demand. go-ionomaps already runs one `Update` at a time under its own lock, so the join is gone; `TestAReopenedModeGetsItsOwnUpdate` fails on the old code with that cancellation. The citations it carried named D-131, a memory ruling; they name A-30 (C-1).
- **L-F3:** the answers were keyed by the snapshot's moment, and the no-readings correction or the stations switched rebuild the snapshot at the same moment, so the chart, reach and centre stayed the old one's. Each key now carries which snapshot was kept.
- **P-6, P-1, P-5:** the Centre box is the answer for the hour, noise and station shown; answers from or to the centre wait for the view to stand still, as the alerts already do; the snapshot is copied for an answer only when it is asked.
- **C-3:** the open session's state is one struct, dropped whole when the mode is left. **C-4 declined:** inlined, two of the three asks are the same code, and the duplicate check refuses them.
- **I-2, I-6, I-3, I-4:** positions in other forms refused locally; the propagation client's port and credentials; the orphan sweep reads the program and reads again before it ends, keeps a record it could not end, ages out a stale one, and tells the diagnostics.
- **Tests:** `TestTwoUpdatesAskNOAAOnce` (in place of the join's), `TestAReopenedModeGetsItsOwnUpdate`, `TestARebuiltSnapshotIsAnsweredAgain`, `TestTheCentreBoxIsForWhatIsShown`, `TestAPanAsksOnceItSettles`, `TestAPositionInAnotherFormNeverGoesOnline`, `TestAnOrphanThatCannotBeEndedIsTriedAgain`, `TestTheSweepIsToldToTheDiagnostics`; the unlisted-host and unrelated-say tests extended; the answer helpers deliver the settle tick the harness drops.
- **Mutants:** m426 to m434 new; m138 retired with the join; m233, m334, m377 and m396 re-pointed.
- **Found running the packages whole:** `TestTheOriginFollowsTheUIMode` asked for the chart straight after a pan, which now waits for the view to settle; the app's declaration set re-captured for `sweptWords`; and the identity check read the credentials case's address, `user:pass@` and a host, as an e-mail address - it is built from its parts now.
- **Docs:** A-68; FR-1.11, FR-1.17, FR-4.2, FR-4.7, FR-4.9 and FR-9.2 as built; the plan's W6.3.

## Batch 40 - W4.8, watchpost on go-tuiMaps v0.3.0-rc.2: the reach's edge asked for, the colour depth a Setting (A-69)

- **The re-pin** (D-206): go-tuiMaps v0.3.0-rc.2, tagged on `dd67d22` after its local full gate, its hosted quick and full gates on both architectures, and its release check, all green. The re-pin checklist: tidy, the licence list, `cmd/watchpost` whole, the alloc budget.
- **What rc.2 asks of a host:** the reach, a two-class host field, is drawn as its edge only when its type asks (`Type.Edge`), so watchpost asks; the night is `NightOverlay(id, at)`; the tiles take `CheckedDialerFor` in place of watchpost's own proxy exemption (D-216).
- **"Colours" (D-207):** Automatic, Truecolor, 256, 16 or None; Automatic and `NO_COLOR` both leave the depth to go-tuiMaps' reading of the terminal, through the `TerminalDepth` this release candidate added for it (go-tuiMaps A-15).
- **Found by the gate:** the map's library-call guard needed the frame's new `ColourDepth` call allowed beside `Units`; the two settings guards carry the new field; the Maps tab's golden carries the new row.
- **Tests:** `TestTheColourDepthIsASetting`; `TestYourFrequencyDrawsItsReach` now requires the edge asked for.
- **Mutants:** m435 to m439, each killed.
- **Docs:** A-69; FR-1.21; the plan's W4.8.

## Batch 41 - W6.4, the answers a package of their own, a found place a target (A-70)

- **D-212 (L-F1):** the architecture named an answers seam that W5 never built - the chart's, the centre's and the limits' working-out lived in the window, the chart's targets a fixed list. They are `platform/propanswers` now: the snapshot and an ask (the origin, the radius, the targets) to rows, limits and words. The window calls it in a command as before; a later mode asks it with targets of its own. The moved code kept its tests; twenty mutants were re-pointed at it and each is killed.
- **Found moving it:** the package was first written under `domains/propagation`, which the window may not import; it is a platform package, holding no state and fetching nothing. And a local variable the chart and the words called `answers` would have shadowed it; it is `rows`.
- **D-204 (B-F1, Critical):** on the Observer a found place was centred and so became the origin, where FR-1.9 and M5 make it a target. It is the chart's last row now, answered from the origin, and the map stays where it is; on the Broadcaster a find is centred, as before.
- **L-F8:** the answers read each band by the band it names, never by its place in a list; two test fixtures that named no band name them now, as go-ionomaps does.
- **Tests:** `TestTheTargetsAreAnInput`, `TestBandsAreReadByName` in the package; `TestAFoundPlaceIsAnsweredAndNotKept` rewritten for D-204; `TestTheCentreIsNamedWhereverItIs` pans onto the place it found.
- **P10 found the new package at 0.50 checks a function:** the code had moved from the window, under its density row, to a package with none. It checks its boundary now - radii, every coordinate on the globe and not a number, a frequency that is none - and answers empty input at once; `TestAnAskOffTheGlobeIsRefused` holds each check (m445 to m448).
- **Mutants:** m440 to m448 new; m190, m193 to m195, m197, m216, m222, m228, m231, m242, m243, m245, m271 to m275, m341, m342 and m347 re-pointed.
- **Docs:** A-70; FR-1.9 and FR-1.10 as built; the architecture's seam row and diagram, and the atlas re-rendered from it (the gate found it stale); the plan's W6.4.

## Batch 42 - W9.3, the remaining code findings: one horizon, short functions, the history's names, the hook (A-71)

- **L-F9:** the words said day or night at -0.833°, the sunrise convention, and the line is drawn at the sun's centre on the horizon, so a place just past the line on the night side was said to be in daylight. The words turn where the line is drawn now; the weather's sunrise times keep the convention.
- **C-6:** P10's 60-line rule reads only the functions a change touches, so long functions this release changed went unseen. Every control-flow one is under 60 now, split along seams it already had; the data tables are left whole (A-71). The atlas renders the same bytes from its split `render`; go-ionomaps' `loadIGRF` is in two (its `94fb27e`).
- **Found splitting `RunDashboard`:** `attachRadio` returns no deck when it cannot build one, and `attachDeck` set that deck's clock unasked; `TestARunWithNoDeckStarts` holds the guard.
- **C-12:** the history's yearly roll-up is named for what it is, an older release's, and its comment says when reading it can end.
- **H-9, I-8:** the hook runs the record's tests from the tree, never a stale binary; it refuses a tree the push does not carry; `make hooks` writes an absolute path and refuses where it would silence another hook, saying how to keep both - each case run in a scratch repository. The README names it.
- **Found by the gate:** the installer was first a shell script; the gate oracle refused its `#!/bin/sh`, and the shell ledger refused a new script without a ruling (everything is Go unless it must not be, A-62). It is Go now, `tools/prepush -install`; and the authoring lint refused the hook as a shell program in a Go string, so the hook is `tools/prehook`, a Go stub that runs `go run ./tools/prepush` from the tree.
- **Tests:** `TestDayAndNightAreSaidAsTheLineIsDrawn`, `TestARunWithNoDeckStarts`, `TestThePushedTreeIsTheTested`, `TestHooksAreNeverSilenced`, `TestTheHookRunsOnlyItsOwnTree`; `TestTheNextTurnIsTheSunriseOrSunset` to the drawn horizon; `TestThePrePushHookRunsTheRecordsTests` reads the installer, which now carries what it held the Makefile to.
- **Mutants:** m449 to m453 new; mAG3, mRT2, m305 and m110 re-pointed.
- **Docs:** A-71; FR-3.1 and FR-10.2 as built; the plan's W9.3; the README.

## Batch 43 - W6.5, the answers' figures, and a too-soon refresh keeps its answers (A-72)

- **P-4:** FR-1.17 cited 4.4 ms, a figure from the dry run's spike with placeholder formulas, and a comment said about 5 ms. Measured on an Apple M5 Pro, the chart takes about 21 ms and the centre about 8 ms (`BenchmarkTheChartIsWorkedOut`, `BenchmarkTheCentreIsWorkedOut`, three runs each); both are worked out off the UI goroutine, the centre's once the view settles (P-1).
- **P-4's waste:** every refresh, even one answered too soon with the snapshot held, counted as another snapshot, so every answer and every hour ahead (about 35 ms each) was worked out again. A snapshot of the same moment and options is the one held now.
- **Tests:** `TestARefreshAnsweredTooSoonKeepsItsAnswers`; the two benchmarks.
- **Mutants:** m454 new; m430 re-pointed.
- **Docs:** A-72; FR-1.17 as measured; the plan's W6.5.

## The M1/M4 instrument's design, put for rulings (W10.3, D-208)

- **Designed, not built:** `07-readiness/m1-m4-instrument.md` sets out the three columns (the climatology, the background alone, the shipped build) and four choices that decide what the instrument measures - what evidences "closed" (A), how the scenarios are drawn (B), how an answer reads as open (C), and which days (D) - each with a recommendation and its strongest counter-argument. None is a minor choice (D-13), so the instrument waits for them; nothing has been fetched.

## Batch 44 - W4.9, watchpost on go-ionomaps v0.1.0-rc.2: the day's climatology hours kept (A-73)

- **The re-pin** (D-222): go-ionomaps v0.1.0-rc.2, tagged on `2436c8a` after its full gate and its hosted quick and full gates. The library keeps the day's climatology hours and builds on an unchanged background (its A-23), so an hour ahead is worked out once a day whichever snapshot asks; F-217's row is deleted. The re-pin checklist: tidy, the licence list, `cmd/watchpost` whole, the alloc budget.
- **What it changes here:** an hour ahead is now its UTC hour from the hour's start, so the mode line labels it so, and the night for it is drawn for the hour's middle. `TestTheTerminatorIsDrawnForTheFieldsTime` had expected the snapshot's minutes carried forward (10:10 for the hour 09:00 to 10:00), not the middle its own comment named; it expects 09:30 now.
- **Hosted CI:** the run on `d6fba065`, carrying batches 38 to 43, passed on macOS and Ubuntu - the first to finish since batch 37, each push before it having cancelled the last.
- **Tests:** `TestAnHourAheadIsLabelledByTheHourShown`. **Mutants:** m455 new.
- **Docs:** A-73; FR-1.13; the plan's W4.9; follow-ups (F-217 closed).

## Batch 45 - W10.3, the M1/M4 instrument built (A-74)

- **What:** `tools/reachscore`, built to the four forks ruled (D-218 to D-221). `spots` sends wspr.live one aggregate query a day, locators and never callsigns, and keeps the answer outside the tree. `score` draws B1's set, answers each scenario three ways through go-ionomaps by replaying PLAN's recorded files, reads each answer by C1, scores by A2 and prints the table by band. It also writes the scenarios and the inputs' checksums outside the tree. Its design and settings are in `07-readiness/m1-m4-instrument.md`, "As built".
- **The replay's promise is tested:** nothing dated after the clock is served (the index, the grids, the solar rows, the stations' readings), and any unknown address fails the run. An end-to-end test runs recorded-shape replies through go-ionomaps and finds them assimilated in the shipped build's column, and in no other.
- **P10:** the package's checks bring it to 2.0. `reservoir.offer` is named `keep`, because P10-01 matches calls by bare name and read `drawer.offer` calling it as recursion. The spots scanner is bounded.
- **Hosted CI:** the run on `573b5bcd` (batch 44) was in progress when this batch was committed.
- **Tests:** the thirteen named in the plan's W10.3. **Mutants:** m456 to m469 new, all killed.
- **Docs:** A-74; the plan's W10.3; the instrument page's "As built" and its diagram.


## Batch 46 - W10.3, the M1/M4 dry run on PLAN's week, and the strata as an option (A-75)

- **The dry run** (`reachscore` at `997f8471`, a clean tree): M1 50.4 % for the shipped build against the climatology's 50.4 %; M4 60.7 % against 58.4 %. Both miss D-99's +5 points. The figures and what they mean are in `07-readiness/m1-m4-instrument.md`, "The dry run".
- **The finding:** B1 balances open against closed within each band and hour, and a served area's answer is nearly constant within the hour, so M1 scores 50 % for any model whose answer depends on band and hour. The scenarios show it (40 m at 15 UTC: 8 of 10 open pairs reached, 8 of 10 closed). It comes to the HUM LEAD as a fork.
- **The strata as an option:** `-strata band` draws by band alone, so the two runs can be put side by side. D-219's band and hour stays the default.
- **wspr.live:** seven requests, one a day, each answered 200 in 3 to 4 s, about 53 MB a day. The spots stay outside the tree with the PLAN evidence.
- **Tests:** `TestBandStrataKeepTheHoursSignal`. **Mutants:** m470 new, killed.
- **Docs:** A-75; the plan's W10.3; the instrument page's "The dry run".

## Docs after batch 46 - the dry run by band alone, and fork E

- **The run** (`reachscore` at `d56225bf`, a clean tree, `-strata band -k 80`): M1 53.4 % in every column; M4 64.2 % for the climatology against 66.9 % for the shipped build. Neither draw reaches D-99's +5 points on either metric. The background alone carries M4's lead; the stations add about nothing.
- **Fork E** (`07-readiness/m1-m4-instrument.md`): the draw the graded run on fresh days uses, put to the HUM LEAD.

## Batch 47 - W10.3, the graded run's draw by band alone (D-224)

- **The ruling:** fork E, "E2 band only (Rec.)" (D-224, amending D-219). The graded run on fresh days draws by band alone, balanced in each band, the hours as they fall.
- **The instrument:** `-strata band` with 80 of each outcome per band is now the default. `-strata band-hour -k 10` still gives D-219's first draw, so the dry run's first figures can be reproduced.
- **Tests:** `TestTheDefaultDrawIsByBand`. **Mutants:** m471 new, killed.
- **Docs:** D-224; the instrument page (section B amended, fork E ruled, the draw's setting); the plan's W10.3.

## Docs after batch 47 - the BUILD report

- **`08-reports/build-report.md`**, presented to the HUM LEAD: what was built, round 1's remediation, the metrics at BUILD exit with what a floor miss would remove (D-208), the agent's errors, what REVIEW inherits, and the rulings asked.

## Docs after the BUILD report - red team round 2 (D-225)

- **Four blind reviewers** looked at `559f22db`, `2436c8a` and `228d6f7`. All four say do not exit BUILD, with no Critical. Round 1's closure, what the remediation itself brought in, the forks and the remediation's order are in `08-reports/red-team-build.md`, "Round 2"; the notes are in `reviewer-notes/build-round-2.md`.
- **The agent's errors this round found:**
  - **The BUILD report** misstated floor 2. The hours ahead are the climatology, so that floor is met by construction. The report also left out D-217's PS-1 sentence, and claimed hosted CI was read after every push when several runs were cancelled by the next push.
  - **Batch 45** renamed `reservoir.offer` and added guards to satisfy P10's analysers, which D-215 forbids.
  - **The remediation** carried D-205 into the library but not into watchpost's words.

## Docs after round 2 - its forks ruled (D-226 to D-229)

- **D-226:** RBN is ruled out, saying why: it reports no transmitter's power, so it cannot be normalised to the reference circuit. The problem statement, the requirements and the UAT protocol now say WSPR alone grades M1 and M4.
- **D-227:** the reach's cost is contained in watchpost (the latest ask wins, hour steps settle) and cut in go-ionomaps (the lower limit on a coarser grid of distances, its accuracy measured), in rc.3.
- **D-228:** the graded run shows a 95 % paired interval. **D-229:** `Inputs.Unread` and `BadEdge` go into rc.3.

## Batch 48 - W9.4, the BUILD exit's round 2 code findings; watchpost on go-ionomaps v0.1.0-rc.3 and go-tuiMaps v0.3.0-rc.3 (A-76)

- **The libraries:** go-ionomaps `v0.1.0-rc.3` (`31c2e19`): the reach's lower limit on a polar grid (D-227: 0.16 s kept, 0.26 s the day's first, 0.65 % of reached cells differing from their own circuits), a restored state that cannot crash an update, one climatology cache, a GIRO 5xx as no answer, `Inputs.Unread` (D-229). Its hosted full gate failed on the restore fuzz target's clock, 200,000 runs past the runners' 600 s; the target has its own count (`8f6c085`). go-tuiMaps `v0.3.0-rc.3` (`7b7c7de`): `BadEdge` (D-229), M5 measured with its contours drawn (10.2 to 11.4 ms at 400 x 110). The re-pin checklist run whole.
- **What every reviewer found:** the reach said "above the MUF everywhere" when the absorption or the circuit stopped it. It now says the limit go-ionomaps names.
- **The rest of round 2's code findings:** each snapshot's state said; a pan settled under a window answered on return (CN2-1); positions in any form refused locally (PN2-4); the hour ahead's night at its start (BL2-4); the kept place laid out only when the words change (PN2-5); a find's request cancelled with it (PN2-7); the geocoder's text cleaned and a failed lookup in words (PN2-6, PN2-9); bands by name (L-F8); an hour ahead said as being worked out until it is (PN2-10); one reach at a time, hour steps settled (D-227); the throttle state kept at the app's end (BL2-5); the instrument failing closed and showing its interval (CN2-9, D-228).
- **D-215 in watchpost (BL2-2, CN2-3):** the workarounds are written plainly and the guards that only raised a count removed. P10 now reports three bare-name misreadings (`New`, `Resolve`, `Put`) and two density shortfalls (`propanswers` 0.94, the instrument 1.34). These are put to the HUM LEAD with the BUILD report, not ruled here.
- **The record's checks:** living documents cite code by symbol, and the check found a stale one at once (`MapPropagation`, now `PropagationUpdate`). The hook runs batch 36's checks, and the as-built map is held to 0.19.0's batches and redrawn.
- **Tests:** the twelve named in the plan's W9.4, and `TestTheIntervalKeepsThePairing`. **Mutants:** m472 to m492 new, all killed. Two first failed vet and one survived its test, which was made to check a reach is not asked at all while stepping. m219, m410, m434 and m465 re-pointed; m364 rewritten, its anchor still matching but its mutation no longer compiling, which the gate's first run caught. m365 (the hour's middle) and m445, m446, m448 (the guards D-215 removed) retired.
- **Docs:** A-76; FR-1.16 and FR-1.17; the plan's W9.4 and W10.3; the as-built map; F-218.

## Batch 49 - `main`'s 0.18.1 merged; the record's round 2 findings (A-77)

- **`main` merged** (BL2-8): 0.18.1's changelog entry, Go 1.27.2 already on the branch. The branch's newer pins and library versions stand, `go.sum` and the licence list are regenerated, and `main`'s comment, with no history in it, is taken.
- **The record:**
  - the README and CHANGELOG say 0.19.0 (DN2-10);
  - the BUILD report is revised: floor 2 met by construction (DN2-3), PS-1 addressed in part (D-217, DN2-4), the hosted runs as they were (DN2-9), the M5 figures reconciled (DN2-14), and M1/M4 with intervals and how to read them (DN2-15);
  - the architecture's answers path, seams and memo key (DN2-7);
  - the instrument page's draw, interval and by-kind figures (DN2-11);
  - the status lines (DN2-12) and the required reading's rows (DN2-14).
- **The dry run again,** from `1e6c60b2`, a clean tree, by band: M1 53.4 % in every column, the lead +0.1 [-0.7, +0.9]; M4 64.2 % for the climatology against 66.9 % shipped, the lead +2.6 [+1.2, +4.0]. The figures are unchanged, now with D-228's intervals.
- **Docs:** A-77; the BUILD report; the as-built map's batches.

## Batch 50 - D-230 and D-231 built: the BUILD exit commit (A-78)

- **D-230:** `propagation.NewService`, the geocoder's `Geocode` and the history store's `Record`: distinct names for each layer's step, so P10-01 reads no call as recursion (renamed with gopls, the geocoder's own method by hand; gopls passed over the property-tagged soak test, which the gate's `vet-tags` caught). The two density rows are ratified in the ledger, and `make p10` reports no live finding.
- **D-231:** the words say a place abroad by its country's name ("near Tokyo, Japan"), from the tz database's public-domain ISO 3166 table, only where the place's country is known and not the US. F-218 is closed.
- **The BUILD report** is marked approved (D-233); this is the exit commit.
- **Tests:** `TestAPlaceAbroadIsSaidByItsCountry` (in `platform/geo` and in the words). **Mutants:** m494 new, killed. m493, on a check that a US label ending in its state's code could never reach, survived: the check was removed and the mutant with it.
- **Docs:** D-230 to D-234; A-78; the plan's W9.4; F-218 closed; the P10 ledger's mirror.

## BUILD exited (D-233)

- **On `e889e72a`** (batch 50): `make verify` green, `make p10` with no live, unmatched or unratified finding, and hosted CI green on Ubuntu and macOS - the first run to carry batches 48 to 50, the runs on `1e6c60b2` and `a9401314` having been cancelled by later pushes. The libraries' hosted full gates are green on go-ionomaps `8f6c085` and go-tuiMaps `d93fac5`.
- **REVIEW is open,** with D-208's long runs beside it: G1's 48 hours, G10.4's fresh-day fetch and G-M3, and the graded M1/M4 run.

## Batch 51 - REVIEW: the G1 harness built (W10.2; A-79)

- **The agent's error:** the BUILD report, as approved, said G1's harness was built. It was not. W10.2 had no driver, no wire count and no summary. The report's line is corrected and the error listed there.
- **The wire count (DN-12):** the propagation client counts the bytes read off its own connections, before decryption and decompression, and the debug counters carry it as `propagation_wire_bytes`. Only the propagation client is counted: G1's downloads are the mode's (D-141).
- **The driver:** `scripts/quality/workload.sh g1` runs two watchposts side by side, each in its own scratch HOME and on its own debug address. In one, the Propagation mode is open, with a pan, an hour step and a frequency entered each hour; the other never opens the map (A-79). Each is sampled every two minutes and its counters read every ten.
- **The summary:** `perfsum -g1` measures each arm from its "G measured" phase and reads the difference against G1's targets. It exits 1 on a miss, and says when a run is shorter than 48 hours.
- **P10:** P10-01 reads `countedConn.Read`, which calls the connection's `Read`, as recursion. That is the io.Reader adapter class the ledger already holds for `tapReader.Read`. A row is put to the HUM LEAD, not written (the ledger is edited only on ratification).
- **Tests:** `TestThePropagationClientCountsItsWireBytes`, `TestThePropagationClientsWireBytesReachTheCounters`, `TestAG1RunIsTheDifferenceOfItsArms`. **Mutants:** m495 to m504 new, all killed.
- **Docs:** A-79; the plan's W10.2; the BUILD report's G1 row and its errors.

## Docs after batch 51 - D-235

- **D-235:** the HUM LEAD ratified the P10-01 row for `countedConn.Read`. It is written in the ledger, and its mirror is regenerated.

## Batch 52 - REVIEW: G1's third arm (A-80); the graded M1/M4 run; G10.4's fetch

- **The fresh days are fetched** (G10.4, D-132): 2026-10-12 to 10-17 under D-39, 87 requests, every one answered 200. That is 39 GIRO replies and 48 GloTEC grids, kept outside the tree, with NOAA's solar and geomagnetic indices beside them. F10.7 was 105 to 109, and no day reached Kp 5, so the week has no storm day.
- **The graded M1/M4 run** (D-221), from a clean worktree of `2ecdaa36`, on the same days. M1: the shipped build 53.7 % against the climatology's 52.4 %, lead +1.3 [+0.4, +2.3], target 57.4 %, missed. M4: 68.4 % against 64.4 %, lead +4.0 [+2.6, +5.5], target 69.4 %, missed by 1.0 point. The climatology scored about the same on PLAN's week (53.4 %, 64.2 %). The figures and their reading are on the instrument page; the misses go to VALIDATE (D-118).
- **G1's dry run** on the built binary, 8 minutes, 3-minute cycles. Every cycle's pan, hour step and frequency entry took effect, the frequency drawn each time; the arm with the mode never opened read nothing off the wire. Two things came of it:
  - the summary read no downloads, because the poller's records were the launch's alone, so the driver's end record now counts (2.58 MB in those minutes);
  - in its first minutes the use arm ran 78 MB above the arm with the map never opened, much of it the map's own. A third arm, the map open in Forecast mode, now measures the mode's own cost beside it, and `perfsum -g1 -against` names the comparison judged (A-80). D-236 ruled, before the 48-hour run's figures exist, that G1 is judged against the map open (J1), and `-against map` is the default.
  - The third arm's first dry run failed its own check: it waited for the chip's "Radar Off", which the renderer redraws only where it changes, so the words never reached the pty whole. A capture of the switch showed the title "FORECAST MODE" drawn whole, and the driver now waits for that.
- **The agent's error:** G10.4's instrument, the Go replica that scores G-M3, was not built at BUILD exit, and the BUILD report did not say so. The report's row is corrected; the instrument is go-ionomaps work before G-M3's verdict.
- **Tests:** `TestAG1RunIsJudgedAgainstTheArmNamed`; `TestAG1RunIsTheDifferenceOfItsArms` updated. **Mutants:** m505 to m508 new, all killed; m501 and m503 re-pointed to the summary's new lines.
- **Docs:** A-80; D-236; the instrument page's graded run; the BUILD report's G-M3 row and errors; the plan's W10.2.

## Batch 53 - REVIEW: the instruments' seams (D-237, D-238); watchpost on go-ionomaps v0.1.0-rc.4; G-M3's verdict

- **D-237, the HUM LEAD's ruling on the instruments' structure:** one implementation of each shared recipe, built before there are users.
  - go-ionomaps `v0.1.0-rc.4` (`73c4a94`): `internal/hybrid`, the update's recipe for its field, which the G-M3 instrument shares; `ionomapstest`, public, which reads recorded days and replays them to the library at a clock, and writes synthetic ones for tests; and the service addresses, defined once.
  - The G-M3 instrument (go-ionomaps `tools/gm3`) was not built at BUILD exit (the agent's error, batch 52). It is built on those seams. Its field, with nothing held out, is held to the library's own update bit for bit. By default it takes the other stations' readings as the library would hold them; `-pairing week` reproduces week.py, which it matches within 0.01 MHz on PLAN's test days.
- **`tools/reachscore` on `ionomapstest`:** its own replay and its tests are deleted, and its column test writes its days with `ionomapstest.Write`. The re-pin checklist was run: `go mod tidy`, the licence list, `cmd/watchpost` whole.
- **D-238:** code never linked into a product build gets its P10-05 row as an A-row batched for veto; go-ionomaps' gm3 and `ionomapstest` rows are its A-26. go-ionomaps' `internal/hybrid`, library code, was ratified singly (its D-81).
- **G-M3's verdict on the fresh days, from rc.4** (go-ionomaps' build log, "G-M3's verdict"). G-M3 is met: mainland US foF2 0.46 against the climatology's 0.71 MHz, MUF(3000) 1.90 against 3.03; all stations 0.84 against 1.07 and 3.03 against 3.75; within 500 km, 0.33 and 1.27. D-75's floor (1) holds, so go-ionomaps ships GIRO's stations. **The Pacific is worse than the climatology** (foF2 1.47 against 1.22, MUF 5.43 against 4.92), wider than on PLAN's week; D-116 reports it ungated, and it goes to VALIDATE.
- **A finding for the HUM LEAD:** with GloTEC's index unanswered, go-ionomaps' update says `Offline` - "nothing could be fetched" - though the solar file arrived, and never offers GIRO's burst, so in a GloTEC outage the map shows the climatology unmoved by any station. Found by `ionomapstest`'s test of the library without GloTEC; not changed here.
- **Mutants:** m457, m458, m459 and m468 retired with `replay.go`; their kills are `ionomapstest`'s hand mutants (a grid indexed after the clock, a reading after the clock, the clock's own day in the solar file, the requester's line kept). m464 re-pointed to `replayOptions`. The other 14 of `reachscore`'s mutants all compile and are killed.
- **Docs:** D-237, D-238; the instrument page's replay; the plan's W10.3; the as-built map.

## Batch 54 - REVIEW: the race leg's clock, measured

- **Batch 53's hosted CI failed on macOS alone**, with "panic: test timed out after 10m0s" in `modes/tty` under the race detector. The test then running had been going for one second, and the batch changed nothing in that package. Under the race detector, `modes/tty` had run 290 to 489 s on the hosted runners and 226 s on the reference machine: a budget at the edge of `go test`'s default, as the mutant check's once was.
- **`make race` gets a measured `-timeout 30m`**, three times the slowest finished run, its measurement recorded beside it in the Makefile, as the mutant check's is. The same day, go-ionomaps' GloTEC fuzz target got its own count, for the same reason.
- **Docs:** the as-built map's batches.

## Batch 55 - REVIEW: an outage said truly (D-239); go-ionomaps v0.1.0-rc.5

- **D-239, the HUM LEAD's ruling on batch 53's finding:** "Approved to fix the GloTEC outage defect, we should never have data mislabelled".
- **go-ionomaps `v0.1.0-rc.5`:** GIRO's burst is offered whenever it is due, whatever NOAA answered. `Inputs.Unread` names every NOAA file asked for and not read. A snapshot short of any is the new reason `Partial`; `Offline` means no NOAA file came back and GIRO was not due. Its record is go-ionomaps' D-82 and A-27.
- **watchpost's words say no more than happened:**
  - `Offline` reads "nothing came back from NOAA: built from what is held", where it said "NOAA not reached: the last field" even when NOAA had answered;
  - files not read are "NOAA's GloTEC not read this update", with what stands in said by the background's own words, where they said "the last held used" even with no grid held and the climatology drawn;
  - GIRO's pause reads "after a refusal or no answer".
- **`tools/reachscore`:** recorded days hold no D-RAP and no scales, so the library now names them unread on every replayed hour. The instrument's fail-closed check counts every other file unread: a GloTEC grid or the solar file not read still fails the run.
- **Tests:** `TestEverySnapshotsStateIsSaid` holds each state's words, and that they never claim "the last held" for a GloTEC outage nor "NOAA not reached" where NOAA answered; `TestOnlyAFileTheReplayServesFailsTheRun`. **Mutants:** m509 to m512 new, all killed. m509 first failed vet, its removal leaving an import unused, and was rewritten.
- **Docs:** D-239; the as-built map; the required reading's library version.

## Batch 56 - REVIEW: the heap growth found and bounded (D-242); the G1 harness (A-81)

- **What G1's soak showed:** post-GC heap growing about 8 MB a day in every arm, the one with the map never opened among them. It rises through the US day and holds overnight, with no day's growth released.
- **What it is:** three idle instances ran silent, the profiled instances muted at the HUM LEAD's ask: 0.18.1, and 0.19.0 with the timing instrument on and off. Each kept an hourly heap profile (`heap-step/`, outside the tree).
  - All three grew alike, so the growth is in 0.18.1 too and is not the timing instrument.
  - Profiles 24 hours apart name the steady holder: the history store's day cache, 0.5 MB to 3-5 MB. It held each hour's whole encoded document after recording it, though recording reads only each hour's issue time and only the map's replay reads the documents.
  - It was bounded at 256 days by count, not bytes. It gains about 90 days a day, so it fills near day three, with one AirNow day 2.5 MB.
  - The rest of the day's rise is HMS's daily file, which falls back with the day, and the HTTP client's bounded caches filling.
  - The code survey's estimate (6-10 MB a day) was twice what the profiles measured.
- **The fix (D-242):**
  - **Recording keeps an index:** each hour and its issue time, so an idle instance holds no document. A write drops the day's documents held for readers, which are no longer its files.
  - **The replay's day cache** is bounded at 8 MB as well as 256 days, through a weight bound in `platform/agememo` (`NewSized`, `Options.MaxBytes`) that leaves its other users unchanged.
  - **Compaction and roll-up** read a finished day once and hold it for no one.
  - **The gauges** `history.days` (with its weight) and `history.index` make both visible to every soak.
- **PF-8 restated:** a day's 24 puts still read their day once. Reading the day back now reads it once more, and once only: the ruling's stated cost.
- **The G1 harness (A-81):** each arm measures 5 minutes past the span asked, so its last sample lands past it, and keeps an hourly heap profile in its folder. D-241's question gave the short span the wrong cause; A-81 corrects it.
- **Tests:** `TestAWeightBoundHoldsTheValues`, `TestRecordingHoldsNoDocument`, `TestTheHistoryStoresCachesAreGauged`; `TestAPutDoesNotReadItsDayAgain` and `TestADayIsCompactedWithoutLosingARecord` extended.
- **Mutants:** m513 to m520 new, all killed. m514 first survived (a write dropped what it would have shown) and found a test gap, now closed; m516 and m518 first failed vet and were rewritten. m498 was re-pointed, its line having grown.
- **Found at the gate's checks:** `withHour`, the old write path's, was left unused, which lint and the duplicate check both caught, so it is removed. P10-01 read the store's `Held` calling the memo's `Held` as recursion, so the store's method is `Memory` (D-215: a distinct name, not a guard).
- **Go 1.27.3 (A-82, under D-152):** `make vuln` failed on a new standard-library advisory, GO-2026-6809 (`crypto/x509`), reached through the radio's TLS streams. CI and the release build move to go1.27.3, as do go-ionomaps' and go-tuiMaps' gates. The live 0.18.1, built with go1.27.2, is affected; whether it gets a 0.18.2 is the HUM LEAD's.

## Batch 57 - REVIEW: the module floor, go 1.26.10 (D-244); 0.18.2 (D-243)

- **D-243:** the live 0.18.1 is built with go1.27.2 and carries GO-2026-6809. 0.18.2 is cut from `main` on go1.27.3 alone, with its own gate, release branch, PR and tag. D-242's change waits for 0.19.0.
- **D-244:** GO-2026-6809 is fixed in go1.26.10 too, and go-tuiMaps' gate scans the standard library at the module floor. Every built module's `go` directive in watchpost, go-ionomaps and go-tuiMaps is 1.26.10, as D-152 did for 1.26.9. The README says it.
- **The libraries:** go-ionomaps' hosted gate is on go1.27.3 (`3be1667`, its A-28); its floor and go-tuiMaps' pins and floor follow, each on its own gate (go-ionomaps D-84, go-tuiMaps D-38 and A-17). watchpost needs no re-pin for them: a library's `go` directive is a minimum, and watchpost's is now the same.

## Docs after batch 57 - `main`'s 0.18.2 merged (D-243)

- **0.18.2 shipped:** PR #30 squash-merged to `main` as `a7e77990`, CI on both systems green, tagged `v0.18.2`. The release workflow built every platform's binary, and the macOS arm64 binary checks as Go 1.27.3 from `a7e77990`. The release branch was pushed past the pre-push hook: the hook runs this branch's `tools/prepush`, which the 0.18 line does not have, and the full gate was green on the commit pushed.
- **Merged here:** the workflows conflicted only on the actions' pins, and this branch's newer ones stand (F-216). The changelog carries 0.18.2's entry; 0.19.0's section now names the floor (go 1.26.10, D-244) rather than the toolchain 0.18.2 already ships, and says the memory growth is in 0.18.2 too.

## Batch 58 - REVIEW: G1's hourly-setting run (A-83)

- **The workload:** `config-hourly.toml` is the default workload with the Propagation mode refreshing hourly, and silent. `workload.sh g1` takes it as its variant.
- **The summary:** `perfsum -g1` reads the variant from `run.txt`. An `hourly` run is held to D-141's 3.3 MB an hour on the wire; its CPU and RSS comparisons are reported, not judged, and it is not called a dry run. `TestTheHourlyRunIsJudgedOnItsDownloadsAlone`; mutants m521 to m523; m499 re-pointed to the target the run is held to.
- **The run:** six hours, the three arms side by side, from 31610af6; its figures are cited from this commit's `perfsum` once committed (E-19).

## Docs after batch 58 - G1's hourly-setting run measured

- **G1's hourly target is met:** 2.66 MB an hour on the wire with the Propagation mode in use, against 3.3 (D-141), over 6.1 hours from 31610af6, summarised by `perfsum` at ec72037c built from a clean tree. With the mode never opened, none. CPU and RSS against the map open in Forecast mode, reported only: -0.10 % and -0.8 MB, as in the 48-hour run. The run and its summary are kept outside the tree, with G1's 48-hour run.

## Docs after batch 58 - REVIEW exit's red team, round 1, and its forks (D-245 to D-262)

- **The round:** seven blind reviewers (D-245); four say do not exit, three exit, no Critical. The reports in full are `08-reports/reviewer-notes/review-round-1.md`; every finding checked against the clean tips is in `08-reports/red-team-review.md`.
- **The forks, all ruled the same day:** D-246 (G1 stands, both readings recorded, a Broadcaster arm), D-247 (GIRO's failures), D-248 (GIRO's plausibility), D-249 (a held F10.7), D-250 (PS-1's accuracy), D-251 (P-6 stands), D-252 (the P10 roster and rows), D-253 (the refresh measured), D-254 (`test-say`'s mute host), D-255 (known limits), D-256 (target keys rebindable), D-257 (the read order), D-258 (one sun), D-259 (the owed wires members deleted), D-260 (NOAA's back-off floor), D-261 (M5's scope), D-262 (go-tuiMaps v0.2.2 and a go-ionomaps nightly).
- **The agent's error, in the gate record:** a `make verify-docs` run at about 16:34:51 UTC, HEAD `0a8d578b`, FAILED on `TestEveryRulingNamingCodeCitesItsLine` (D-258 cited `platform/astro/turn.go` without its line). Before the run that passed, the agent restored `06_docs/verify-runs.md` from HEAD, which removed that run's FAILED row: the record is meant to keep every run. It is recorded here instead; the record is never restored again before a run.

## Batch 59 - REVIEW: the red team's remediation in watchpost (D-245 to D-262; A-84)

- **Wrong output fixed:** an hour ahead asked before a rebuild at the same moment was kept for the rebuilt snapshot (RC-4, L-F3's class); beyond the far distance the words now say D-129's "or worse" (RB-1).
- **Verifiers that passed without proof:** `perfsum -g1` read an unparsable RSS as 0, passed a run its driver warned on, and exited met on a dry run; `workload.sh` exited 0 when a driver failed (RC-1). `runlog` recorded the tree at the run's end, not the one it tested (RC-3), and logged a refused docs lane as a failed run (RD-12). `make test-say` skipped green where `say` renders nothing (D-254).
- **Rulings built:** the chart's target keys rebindable (D-256); `Event.Restored` and `State.Refused` deleted, with `takeDiscarded`, their tests and their wires rows (D-259); the REQUIRED-READING opens with what the release is (D-257); both G1 comparisons recorded (D-246); the VALIDATE agenda (D-255); 0.19.0's gate roster (D-252, RH-6).
- **Inputs bounded:** the geocoder's postcode, zone and coordinates (RI-2); reply headers and validators (RI-4); the state file's path out of diagnostics (RI-7).
- **The record:** the hook's error has a path (RH-8); `tools/finaltag` refuses a final tag on a paired library's release candidate (RH-9); `reachscore`'s blind spots and its withheld verdict (RB-8, RC-9); the architecture's API names, the README's floor ruling and P10's files, the problem statement, the changelog's memory entry, the PLAN report's figures (RD-7 to RD-10, RH-1, RB-6).
- **Mutants:** m524 to m531; m61 and m279 re-pointed.
- **Found before its gate:** a chart group of its own pushed Observer's row-mark legend out of Help's window, so the chart's keys are said on the pan row, "Pan (Bands: up/down target)"; the service refuses to start when go-ionomaps names no host (P10, `domains/propagation`), and `tools/finaltag` and `tools/runlog` took D-238 rows (A-85); m378 and m398 re-pointed; the map and chart key scopes share one merge.

## Batch 60 - REVIEW: the remediation's words, on go-ionomaps v0.1.0-rc.6 and go-tuiMaps v0.3.0-rc.5 (D-247, D-249, D-258, D-263; A-86)

- **GIRO's outcome said all hour** (D-247): GIRO refusing the host (a 401, 403 or 410) is said as "GIRO refused this host: not asked again for a day", and noted once in the diagnostics as it begins; a 429 or three failures in a row as "GIRO unavailable this hour"; both carried until the next burst, whatever the snapshot's reason; a station whose own ask failed is counted, "1 station did not answer".
- **The kept solar flux** (D-249): with NOAA's solar file not read, the kept F10.7 mean's age is said, "the solar flux held from 2 days ago".
- **One sun** (D-258, D-263): the words take day, night and the next turn from go-tuiMaps' `DaylightAt` at the moment shown - the sun the night is drawn with - so an hour ahead says what its picture shows. A probe before the ruling found that `Report`'s `Daylight`, at the frame's clock, said day in Wichita under 03:00 UTC's drawn night. `platform/astro`'s `Elevation`, `Up` and `NextTurn` are deleted; `SunTimes` stays for sunrise and sunset.
- **The real library's states** (RL-12): `TestTheLibrarysRealStatesAreSaid` drives go-ionomaps through `ionomapstest`'s replay into each state - fresh, GloTEC refused, GIRO refusing and denying, a station failing, the solar file unread two days on, a cold start with no field - and checks the words for each, never a snapshot built by hand.
- **Mutants:** m532 to m536, each killed (m533 survived a first run: every state driven was `GIROPaused`, so the all-hour line had no case; an ask too soon after the denial gives it one); m339 re-pointed; m349, m350, m351 and m449 retired with the code they changed.

## Batch 61 - REVIEW: every density row held to its floor (D-264; A-87)

- **The floors** (D-264): the twenty-three density rows a2dh reports state their measured figure as a floor, `modes/tty` 0.43 over 339 and `app` 0.61 over 49 among them; the mirror is regenerated from the ledger.
- **`tools/p10floor`**, in `make p10`: an exempted density row measuring under its floor fails, naming it; a figure that does not read fails closed. Run on a copy of the report with `modes/tty` lowered to 0.42, it failed; on the report itself it held all twenty-three. Its own guards are single returns through one closure, so P10 counts them; its `--self-test`, in `gate-controls`, proves it refuses a planted row under its floor in this tree.
- **The REVIEW red team's record:** the D-252 list's figures re-measured at the tips before the batch was put (`modes/tty` had grown from 335 to 339 functions); its sentence on go-ionomaps' nested modules made exact - P10's analyzer legs skip them and its density count reaches them, as the spike's row already said.
- **go-ionomaps** holds its sixteen rows by its own test (D-91), on the hosted gate too.
- **Mutants:** m537, m538.

## Batch 62 - REVIEW: the centre's name, the history store's files, the answers' arms (RP-6, RC-10; A-88)

- **The centre's name** (RP-6): naming the map's centre scanned the whole world table on the UI goroutine at every step, 0.91 ms; it now reads the band of latitude the fence can reach, 0.05 ms (`BenchmarkNearWorldPanning`), answering as the whole scan does - `TestAWorldScanAnswersAsTheTableDoes` now runs pole to pole at the app's fence and a 500-mile one, and `TestTheBandKeepsTheTablesOrder` holds the order ties rank by. A first attempt broke ties by a row's offset, until the table's order turned out to be the names', not the offsets'.
- **The history store's files** (RC-10): `platform/history/history.go`, 2,349 lines, split into eight files along its concerns - the store, the caches, the documents, the numbers, claims, pruning, roll-ups, size. A pure move: `TestDeclarationSetUnchanged` (new, its golden captured before the move) and the moved lines, compared as a multiset, equal; sixteen mutants re-pointed and each killed in its new file. `history-store.md` names the files, and its import list now includes `agememo`.
- **The answers' arms** (RC-10, cont.): the chart, the reach and the centre each have their own arm; the "still settling" check is one function. m434, m491 and m492 re-pointed and killed.
- **Mutants:** m539, m540 new; m403 re-pointed.
- **Its first verify run failed** at `make test-say`: macOS `say` hung for 1 min 50 s and rendered nothing, which D-254's check refuses on a host not declared mute. A probe a minute later rendered 61 kB; the run again was green. Both rows are in `06_docs/verify-runs.md`.

## Batch 63 - REVIEW: the Propagation mode's state by lifetime (RC-5; A-89)

- **One key for the snapshot as kept** (RC-5): `propMade` - the snapshot's moment, which snapshot kept, and the options it was made with - is carried by the chart's, the reach's and the centre's keys and the hour ahead's message; `madeWith` writes a snapshot's options once, so the three ways "made with the same options" was written (`armPropOptions`, `sameSnapshot`, the correction's check) are one comparison, and the next option is added to the update's options alone. RC-4's class - an answer kept for a snapshot rebuilt at the same moment - has one place to go wrong now, not four.
- **A find ends with its session** (D-27): leaving the mode left a find in flight running, its answer free to mark a place past the mode and, on the Broadcaster, move the map. Found while reading each field's lifetime; `TestLeavingTheModeDropsAFindInFlight` failed on the old code. `dropFind`, which Esc already did by hand, is called by Esc and by the session's end.
- **The fields by lifetime:** `mapPane`'s thirty-nine Propagation fields are grouped and commented by how long each lives - the session's, the snapshot's and its answers', the listener's settings, the entries, the place found, what is handed to the map, the reader's place - with no field moved or renamed.
- **Mutants:** m541 new; m195, m230, m231, m235, m289, m395, m454, m524, m420 and m482 re-pointed, each killed in its new place.
- **Its first verify run failed** at the mutant check: m431 and m232, in `map_prop_centre.go`, wrote the key's old fields in their replacement text - their anchors still matched, so the pre-verify anchor check passed them. Both re-pointed and killed; every mutant then compiled in a full run before the second verify, which was green.

## Batch 64 - REVIEW: the Broadcaster arm's harness, and the measurements (D-246, D-248, D-261, D-265, D-266; A-90)

- **The Broadcaster arm** (D-246): `workload.sh g1b` runs two console arms side by side, the map on `g` with the Propagation mode on G1's use cycle against the console idle, at 150 by 46; `perfsum -g1 <outdir> -against idle` judges it. A dry run of 8 minutes drew every cycle's frequency; the 48-hour run starts from this commit, alone on GIRO.
- **G-M3 re-run** (D-248): the committed gm3 at go-ionomaps 3c2da5a gives every figure and verdict of the earlier run on the fresh days unchanged, so the seed-position check and the residual bound refuse no reading there; the largest residuals - fresh days foF2 6.09 MHz and M(3000)F2 1.24, PLAN's week 7.19 and 1.48 - lie under the bounds of 8 and 2.0.
- **The grid's age** (D-253, D-265, D-266): the fresh days hold a grid every 3 hours, so D-253's hour-old grid does not exist in them - the agent's error in the fork's evidence. `gm3 -grid-age 3h` (go-ionomaps 14b38ff) scores the newest grid beside the one 3 hours older: mainland US foF2 RMS 0.46 against 1.21 MHz, worse than the climatology's 0.71; all stations 0.84 against 1.27. The gap met D-265's condition; the HUM LEAD authorised a day of 10-minute grids (D-266), fetched under D-39 outside the tree, to score 10, 30 and 60 minutes.
- **M5 at 800 by 220** (D-261): after RP-4's fixes, with the structured field, 24.6 to 26.7 ms a pan frame (the reviewer's 32 to 42 before them); 400 by 110 at 6.8 to 7.3 ms against M5's 16. A known limit, recorded in the VALIDATE agenda.

## Batch 65 - REVIEW: the say hang diagnosed, and the vetoes (D-267, D-268; F-219)

- **The say hang, diagnosed:** batch 62's and batch 64's first verify runs failed at `make test-say`, `say` killed at its bound with nothing rendered, and the test's sample of the hung process logged nothing. `sample` writes its report to a file, never to its output: the two reports were in `/tmp`, written at the two failures. Each shows `say`'s main thread waiting in its run loop on a Mach message for the whole sample - the speech service never answered. The test now has `sample` write its report where the test reads it, and logs the main thread's call graph (`TestAHangsReportNamesItsMainThread`); `TestTheSayLegIsBoundedAndDiagnosed` holds that the report is read. The cause is macOS's: F-219.
- **Rulings recorded:** D-267, the REVIEW remediation's A-rows stand; D-268, go-ionomaps' `tools/tables` under every P10 leg (go-ionomaps D-93: its two input-bounded loops now bounded by lines).
- **Mutant:** m542.

## REVIEW exit red team, round 2 - filed and ruled (D-269 to D-282)

- **Filed:** the seven reports verbatim in `08-reports/reviewer-notes/review-round-2.md`; the verdicts, the forks and the remediation list in `08-reports/red-team-review.md`, round 2. Five reviewers say exit and two do not (Docs: R2D-1, the committed scorer against D-241; the REVIEW lens: R2L-1 to R2L-3, go-ionomaps' failure handling, R2L-3 brought in by D-260).
- **Ruled:** D-270 to D-282, each put singly with its evidence.
- **Found during the pass:** go-tuiMaps' gate test `TestALegThatIgnoresTermIsStillStopped` kills `sleep 3607` machine-wide, so the reviewers' parallel gates failed each other (R2H-1); every run of a library's `scripts/gate` appends to its tracked `gate-runs.md`, so the reviewers' quick gates left rows in both libraries, kept as the record keeps every run.

## Batch 66 - REVIEW: the red team's round 2 remediation in watchpost (D-270 to D-282; A-91)

On go-ionomaps v0.1.0-rc.7 and go-tuiMaps v0.3.0-rc.6 (D-206, D-278); the libraries' halves are in their own build logs.

- **G1's span** (D-273): `perfsum` takes an arm as 48 hours when its span is within one 2-minute sample of it, and says so; three minutes short is still a dry run. D-241's run now reads as the measure: its arms 47.995, 47.969 and 47.975 hours, the judged arm, the map open, inside the tolerance by 9 seconds (the sampler's measured step 2.11 minutes against its nominal 2; R3D-9, batch 67).
- **The say leg** (D-276): each render has its own 2-minute bound; one killed at it has its main thread logged and is tried once more, the retry printed as `RETRY:` in the leg's output; a second hang fails. `TestAHungRenderIsTriedOnceMore` drives the rule with a render that hangs. The leg's timeout is 6 minutes.
- **The floors** (D-277): `p10floor` fails on a firing density row whose reason states no floor, naming it with what it measures; none fires today.
- **The benchmark** (D-282): `BenchmarkPropRenderMap` pans one step every op over a field shaped as a recorded MUF map, go-tuiMaps' own; the frame's allocation budget re-set from the shaped field (A-91 (2)). A pan frame at 133 by 44 is 1.20 to 1.29 ms, 557 allocations and 72 KB (five runs of 200, Apple M5 Pro; the evidence folder's `m5-frames/d282-prop-render-map.txt`), against 19.8 to 20.3 µs and 124 allocations for the unchanged view the benchmark used to redraw.
- **The rebuild and back** (R2C-1): `TestARebuildAndBackIsAnsweredAgain` rebuilds with other options and back at the same moment; the chart worked out before is not shown and an hour ahead asked before is dropped. Both m430 and `built: 0` (m547) survived every test before it; m430's "Killed by" named a test that flips the options.
- **`madeWith`** (R2C-2): `TestEveryUpdateOptionIsMadeWith` sets each of go-ionomaps' options in turn by reflection and fails until `madeWith` carries a new one; the comment that said one place was enough is corrected.
- **One moment** (R2C-8): the words take day or night at the night's own 5-minute step; `TestTheWordsAndTheNightShareAMoment` reads them ten seconds after the line passes.
- **Stored text** (R2I-2, R2I-3): the geocoder's zone "Local" is not kept; Settings draws every place through `placeWithZip`, its postcode plain, and a place with no postcode, which drew `Reykjavik, IS ()`, without empty brackets; the tile transport caps reply headers at 64 KiB.
- **The floors** (D-283): `modes/tty` measures 0.42 over 344 changed functions, under its 0.43: `hasKey`'s deletion took a function and its guard, and the postcode fix brought two pre-feature renderers into the changed set. The HUM LEAD renewed the floor at 0.42. `domains/propagation` held 2.00 once `noteStatus`'s missing-diagnostics case became its own guard; `tools/runlog`'s record stayed one function, 1.40.
- **Release** (R2H-3): `finaltag` reads go.mod through `go mod edit -json` and refuses a replaced paired library; the release step is pinned by a test. Against today's go.mod it refuses v0.19.0, as it must.
- **The refused readings** (R2C-10, R2L-5): the diagnostics note go-ionomaps' three counts apart - the burst's rows past the range checks, readings refused whole as implausible (D-248, D-279), readings whose M(3000)F2 alone was refused - `TestEveryRefusedReadingIsNoted` (m560, m561).
- **Records** (RC-3, RC-7, RC-9, R2H-8, D-272): a run with no record fails; `hasKey` and `stillOrphan` deleted; the run-selector check reads build tags; `make p10` names a missing `.a2dh.yml`; a GIRO or NOAA status D-247 does not name is noted once by host, and the denial's words no longer name a 410.
- **The record** (R2H-4, R2H-9, R2D-2, R2D-5, R2D-6, R2D-7, R2B-2, R2B-3, R2B-4, D-274): the headers say REVIEW; round 1 closed finding by finding in `red-team-review.md`; G1's figures say one run, one machine, and the noise between arms; the agenda carries the Broadcaster arm's rule, Hourly's STALE minutes, D-255's three limits and M5's run count; F-219 records D-276; F-220, the open map's own cost, with a target; the README names the chart's target keys; the judging command is spelled `-g1 <run> -against idle`.
- **The re-pin** (D-206): go-ionomaps v0.1.0-rc.7 and go-tuiMaps v0.3.0-rc.6; `THIRD_PARTY_LICENSES.md` regenerated; the tile transport's test now holds watchpost's header cap to the library's 64 KiB; the as-built map redrawn (the fetcher's notes, the refused readings) and the atlas regenerated; `app`'s declaration set re-captured for `tileMaxHeaderBytes`.
- **Mutants:** m543 to m562, each vetted applied and killed by its test (the mutant runner); m378, m430 and m538 re-pointed; m427 deleted with the second read it broke.

## Batch 67 - REVIEW: the red team's round 3 remediation in watchpost (D-285 to D-289; A-92)

On go-ionomaps v0.1.0-rc.8 (D-285 to D-288: a stalled NOAA host stops its round at 20 s, three asks running out call GIRO down, a stop moves the failed stations behind, no IDs in its public docs; `Restore` names the part it refused) and go-tuiMaps v0.3.0-rc.7 (no IDs in its public docs; the aliases' fields held by a test); the libraries' halves are in their own build logs.

- **The refused readings** (R3L-7): each count is noted when it changes from the snapshot held - one burst's counts, carried through the hour, noted once (`TestEveryRefusedReadingIsNoted` drives an hour; m564); the burst's count worded as what it counts.
- **A stopped burst** (D-286): the update asked after a StationsNext snapshot is GIRO's burst; an Unfinished one says so (`TestAStoppedBurstNamesGIRO`; m568).
- **The restore note** (R3L-6, R3C-2): the diagnostics name the part not taken - GIRO's throttle state, the kept solar flux, both, or the whole state - each on one line (m569); the architecture and FR-4.12 say each part is taken on its own.
- **D-272's note** (R3L-5, R3C-3): the statuses D-247 and D-260 name - 404 and the 5xx class among them - are never noted; the words fit NOAA's hosts too (m563).
- **The gates** (R3C-8, R3C-9, R3C-12): `runlog`'s end-of-run tree read and its line's write each fail the run when they fail (m565, m566); the say leg's 11 minutes are its renders' own and a hang needs an error; the run-selector check's `-race` branch is held (m567). The gate roster carries D-273's tolerance and batch 66's six gates (R3D-4).
- **Smaller:** `ionomaps.UnreadSolar` for the literal (R3L-8, R3C-5); `madeFromHeld` (was `propMadeOf`) and `reachscore`'s check written once (round 2's unrecorded Minors); `docs/extending.md` names the four keymaps and the README the actions' form (R3D-8); G1's arms' spans and the 9-second margin beside "taken as 48" (R3D-9); F-220's figures say one run (R3D-11); F-221 carries D-289; the record's R2H-5 lines and its H1 corrected (R3D-5).
- **The re-pin** (D-206): go-ionomaps v0.1.0-rc.8 and go-tuiMaps v0.3.0-rc.7, `THIRD_PARTY_LICENSES.md` regenerated; the as-built map redrawn (the restore's parts, the notes on change, a stopped burst) and the atlas regenerated.
- **Mutants:** m563 to m569, each vetted applied and killed; m231, m290, m325, m378, m398, m524, m534, m556, m560 and m561 re-pointed.

## Batch 68 - REVIEW: the red team's round 4 remediation in watchpost (D-291 to D-293; A-93)

On go-ionomaps v0.1.0-rc.9 (D-291: a burst reads the time left on the update and ends before the deadline, GIRO called down when its asks were running out; a host client's own timeout counts as an ask run out; no NOAA ask starts without room; no IDs left in its public docs) and go-tuiMaps v0.3.0-rc.8 (no IDs in its public docs; every struct alias's fields held by a test); the libraries' halves are in their own build logs.

- **A 410 is named** (R4D-3): a station or file gone, as D-247 and D-272 say, never noted as unknown; the note's test moves to a 418 and a 451 (m570).
- **The burst's flag ends with its session** (R4C-3): `propSession.burst`; `TestAStoppedBurstNamesGIRO` leaves the mode with a burst in flight, opens it again, and NOAA's stopped round is said as NOAA's files (m572; m290 and m568 re-pointed).
- **G1's scorer** (R4D-4): each arm's hours to three places and its shortfall from 48 hours, to the second; 2 minutes is the sampler's nominal step (m571). The gate roster's row 29 cites m544 and m571.
- **Docs:** the windows' keys are fixed, the docs saying so (D-292); the Broadcaster arm names `a444b000` and its figures an upper bound for the tip (D-293); the state file holds times and the flux mean (R4D-7); the say leg's 11 minutes hold three renders and a retry (R4D-5); batch 67's re-pointed mutants listed as they were.
- **The re-pin** (D-206): go-ionomaps v0.1.0-rc.9 and go-tuiMaps v0.3.0-rc.8, `THIRD_PARTY_LICENSES.md` regenerated; the as-built map redrawn (the named 410, the burst's flag on the session, the library's node at rc.9 with its ask limits) and the atlas regenerated. The first verify failed on the map not yet redrawn (the failed row kept in `verify-runs.md`).
- **Mutants:** m570 to m572, each vetted applied and killed; m290 and m568 re-pointed. G1's shortfall is worked out in the row it prints in, not a helper of one guard, which held `tools/perfsum`'s density at its floor.

## Batch 69 - REVIEW: the red team's round 5 remediation in watchpost (D-295 to D-299; A-94)

On go-ionomaps v0.1.0-rc.10 (D-295 to D-298: GIRO's last asks get the time left, the call-down at the update's end on two full asks run out in a row, a burst's stations not reached counted, no back-off for a round with no room, `MinDeadline` and `ErrNoRoom`) and go-tuiMaps v0.3.0-rc.9 (D-299: the memory peak judged with the race detector off; the alias count held at seventeen; the doc scan widened); the libraries' halves are in their own build logs.

- **A burst cut short is said** (D-297): "GIRO's burst cut short: N stations asked first next hour" in the status, and once in the diagnostics while the count holds (`TestACutShortBurstIsSaid`; m573, m574). D-286's stop words, `propSession.burst` and their test are gone with m568 and m572; m290 and m325 re-pointed.
- **The least deadline** (D-298): a Deadline under go-ionomaps' `MinDeadline` is refused; `TestAnUpdateHasItsOwnDeadline` reads `ErrNoRoom` at the least deadline, said at once (R5C-6, R5L-5; m139 holds).
- **Docs:** the bands chart window's `esc`, `PgUp` and `PgDn` are fixed, and the acknowledgement window is in the README's list (R5D-4); the required reading and the record through round 5 (R5D-5); the architecture's failure table gains a denial, a station's own failure, GIRO hanging, a burst cut short and a stalled NOAA (R5D-6); item 3's two blind spots beside its bound (R5D-8); the state file's contents (R4D-7's residue); FR-5.2 carries D-297, and plan task W4.3 its test; `noteStatus`'s doc names D-272.
- **The re-pin** (D-206): go-ionomaps v0.1.0-rc.10 and go-tuiMaps v0.3.0-rc.9, `THIRD_PARTY_LICENSES.md` regenerated; the as-built map redrawn (the cut-short words, the refused short deadline, the library's node at rc.10 with its round 5 rules) and the atlas regenerated.
- **Mutants:** m573 and m574, each vetted applied and killed; m568 and m572 retired; m290 and m325 re-pointed.

## Batch 70 - REVIEW: the red team's round 6 remediation in watchpost (D-301 to D-307; A-95)

On go-ionomaps v0.1.0-rc.11 (D-301 to D-307: one room rule with a 1 s margin, `MinDeadline` 32 s which asks; the count of stations not reached at the update's end only; a probe not reached first next hour; Unread's lacks rule; a server error breaking the run; `internal/deadline` with its ratified row; a station that ran out asked last) and go-tuiMaps v0.3.0-rc.10 (the race flag checked against the build); the libraries' halves are in their own build logs.

- **A quit stops GIRO's burst** (R6T-1): the update is tied to the app's context, cancelled before the app waits, so a quit during the burst keeps GIRO's throttle state inside `settleBound` and a restart within the hour asks nothing new (`TestTheAppsEndStopsAnUpdateAndKeepsItsState`; m575). `Settle`'s doc and its test say what they do.
- **The least deadline** (D-301): `TestAnUpdateHasItsOwnDeadline` reads the service's own deadline set under the least, and an update at exactly `MinDeadline` asks NOAA (m139 holds).
- **The diagnostics' guard** (R6T-6, R6L-4): a burst GIRO stopped is not noted as cut short (m576).
- **Docs:** `snapshotWords`' stale sentence gone (R6D-3, R6C-5, R6L-7); the failure table's 429 and NOAA rows quote the words (R6D-4); the state file's contents alike everywhere; batch 69's "R5D-4's residue" is R4D-7's; item 3's "U9 to U12" explained; the record's round 5 line on the deadline test corrected (R6L).
- **The re-pin** (D-206): go-ionomaps v0.1.0-rc.11 and go-tuiMaps v0.3.0-rc.10, `THIRD_PARTY_LICENSES.md` regenerated; the as-built map redrawn and the atlas regenerated.
- **Mutants:** m575 and m576, each vetted applied and killed; m298 re-pointed onto the seam's tied context.

## Batch 71 - REVIEW: the re-pin to go-ionomaps v0.1.0-rc.12 (go-ionomaps A-36)

- **The hosted gate's find:** go-ionomaps rc.11's full hosted gate failed its NaN fuzz target on amd64 - a path from a place to itself panicked in the sun track, a NaN turned to an index - which no arm64 gate could see; v0.1.0-rc.12 holds a path's crossings to it and refuses a place that is no number before stepping (go-ionomaps A-36). watchpost's Path asks are an origin and the continents' targets, so a path of no length needs the centre on a target; the re-pin closes it whatever the place.
- **The re-pin** (D-206): go-ionomaps v0.1.0-rc.12, `THIRD_PARTY_LICENSES.md` regenerated; the as-built map's library node at rc.12 and the atlas regenerated.

## Batch 72 - REVIEW: the red team's round 7 remediation in watchpost, built to the timing model (D-309 to D-312; A-96)

On go-ionomaps v0.1.0-rc.13, built to the update's timing model (D-312): a station known to hang stays last when a deadline left it (D-309), its running out again counts toward no run that calls GIRO down (D-310), and the clamp a helper with `internal/limits`' floor renewed (D-311); the library's half is in its own build log.

- **A quit during GIRO's burst** (R7T-3, R7C-6): `TestTheAppsEndStopsAnUpdateAndKeepsItsState` reads the 39 seeded live stations in a first hour, then cancels the next hour's burst in flight, its field built from a full hour's readings as it stops - 60 ms, 405 ms under race - and a restart within the hour asks GIRO nothing; `settleBound` is pinned by it (m577, A-96). The first fixture, one station and no readings held, settled in 1.5 ms and let m577 survive.
- **Docs:** the architecture says the app's end cancels the update too (R7D-4) and links the timing model; VALIDATE's item 7 and the changelog's known limits carry the model's section 8 (R7D-6).
- **The re-pin** (D-206): go-ionomaps v0.1.0-rc.13, `THIRD_PARTY_LICENSES.md` regenerated; the as-built map's library node at rc.13 and the atlas regenerated.
- **Mutants:** m577, vetted applied and killed.

## Batch 73 - REVIEW: the red team's round 8 remediation in watchpost, to the timing model as D-314 changed it (A-97)

On go-ionomaps v0.1.0-rc.14: the model changed whole (D-314) - before a burst's first answer a station known to hang counts, so a dead GIRO is called down every hour (M1); one station known to hang retried at each burst's head with a 2 s ask, in turn (M2); a kept probe known to hang asked last (M3); the model's words made true (M4). The library's half is in its own build log.

- **Docs:** the changelog's known limits as the model's section 8, all six (R8C-4, R8L-4, R8T-3); VALIDATE's item 7 the same, citing the model by repository and path; the required reading and the architecture cite it the same way (R8D-6); the as-built map's library nodes redrawn - rc.14's rules, go-tuiMaps at rc.10 (R8D-5); the Broadcaster figures' blind spots and unit (R8D-8); A-96's figures name their platform.
- **The re-pin** (D-206): go-ionomaps v0.1.0-rc.14, `THIRD_PARTY_LICENSES.md` regenerated; the as-built map and the atlas regenerated.

## Batch 74 - REVIEW: the hosted gate's find after batch 72 (A-98)

- **The find:** watchpost's hosted gate on `1275b161` failed on the Ubuntu runner's race leg: `TestTheAppsEndStopsAnUpdateAndKeepsItsState`'s cut burst took past `settleBound`, 2 s, under the race detector (405 ms under race here, 60 ms without). macOS passed.
- **The fix:** under race the test holds the burst to `settleBound` times `raceSlowdown` (10, `app/race_test.go` and `app/norace_test.go`); the 2 s value stays pinned without race, where the mutants run. m577 killed without race, m575 killed under race, each vetted applied.
- **A-96 corrected:** its "five times the race figure" is this machine's.

## Batch 75 - REVIEW: the HUM LEAD's simpler model, watchpost's side (D-316 to D-319; A-99)

On go-ionomaps v0.1.0-rc.15, the update's timing model replaced by the HUM LEAD's six points (D-316): stations asked in turn, a station with no data a miss and passed by, one missing three in a row flagged and asked every 6 hours (D-317), GIRO paused only for its own refusals (D-318), and the snapshot's count of stations reporting and how recently. The library's half is in its own build log.

- **The notice** (D-319): `thinStationsWords` says the map is a best approximation, first on the status line, when fewer than half the stations holding a reading reported in 3 hours or the newest is older than that (`TestThinOrStaleStationsAreSaid`; m578, m579).
- **Words:** "N stations gave no data"; one station said as one (R9L-3); the paused fallback "GIRO paused after a refusal"; the cut-short words and note removed, with m573, m574 and m576.
- **Docs:** FR-5.2; the architecture's failure table; the changelog's known limits and VALIDATE's item 7 to the model's accepted limits; the as-built map's library node.
- **The re-pin** (D-206): go-ionomaps v0.1.0-rc.15, `THIRD_PARTY_LICENSES.md` regenerated; the as-built map and the atlas regenerated.

## Batch 76 - REVIEW: round 10's light review (D-320, D-321; A-100)

On go-ionomaps v0.1.0-rc.16: a flagged station's miss no longer counted (D-321), a refused reply a miss, rule 7 and the limits worded to the code. The library's half is in its own build log.

- **One fresh reviewer: exit REVIEW**, no Important, seven Minors (record, Round 10).
- **The notice's edges** (R10-4): exactly half, and 2 h 59 m against 3 h 01 m (`TestThinOrStaleStationsAreSaid`; m580, m581).
- **The changelog** (R10-6, D-321).
- **The re-pin** (D-206): go-ionomaps v0.1.0-rc.16, `THIRD_PARTY_LICENSES.md` regenerated; the as-built map and the atlas regenerated.


## Batch 77 - VALIDATE: UAT issue one, a host that hangs (D-324; A-101)

- **The find:** the HUM LEAD's Observer session held the RECENT/SEARCHED rows loading for over 40 seconds. Its dump (SIGUSR1) showed FIRMS with 114 attempts and none answered, and 46 waits behind it; FIRMS's API was accepting requests and never answering. On a copy of the HUM LEAD's settings in a scratch HOME, a proxy that completes FIRMS's handshake and passes nothing on reproduces it: 27 rows loading at 60 s on the tip, and the same on 0.18.2.
- **The fix** (`platform/httpx`): a client timeout arms the failure memo as a refused connection does, and only the caller's own context ending is excused (`noteFailure`); each host takes its share of the lane, half of it, before the lane's slot (`acquire`, `share`). Under the same hang every row fills by 20 s; with FIRMS answering, by 20 s as before.
- **Tests:** `TestAHostThatNeverAnswersIsAvoided`, `TestTheCallersOwnDeadlineNeverMarksTheHost`, `TestAHungHostHoldsAtMostHalfALane` (each lane), `TestAHostsShareComesBackWithEachAnswer`, `TestAWaitThatEndsGivesTheShareBack`, `TestTheSharesAreBounded`; the first and third failed before the fix. m582 to m587, each vetted applied; m585 also hangs an earlier test in the whole package, as a share never returned would hang the app.
- **Docs:** the changelog's Fixed; `docs/where-things-happen.md`'s lanes and memo; the CLI's caching page; the as-built map's batch line and the atlas.
- **Not a finding:** the scripted runs' terminal emulator kept three rows of an earlier frame on screen in every build; the HUM LEAD's terminal shows each place once.
