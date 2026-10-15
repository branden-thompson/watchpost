---
title: "0.19.0 — BUILD log"
date: 2026-10-08
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "IN PROGRESS — one entry per batch; the record moves before the gate (REFLECT L5)"
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
- **Tests written first:** `TestTheContinentsAreTheDefaultTargets`, `TestBestBandsNameTheirLimit`, `TestTheDaysOpenHoursAreShown`, `TestTheTypicalDayIsLabelled`, `TestADisagreementWithTheTypicalDayIsSaid`, `TestTheOriginIsTheSelectedPlaceOrTheTower`, `TestAnswersNeverBlockTheUI`, `TestTheNearVerticalRadiusIsASetting`, `TestTheNightIsDrawnOnThePropagationMap`, `TestTheNightFollowsTheClock`; each failed before its code. The tests that read the hops line, the one Propagation overlay, Help's map row and the settings kept read the new ones.
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
