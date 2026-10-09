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
