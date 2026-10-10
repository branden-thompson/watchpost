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
