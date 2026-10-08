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

