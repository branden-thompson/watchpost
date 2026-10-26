---
title: "0.19.0 — the gate roster: gates added and changed, each with a failure that was watched, and P10 at each phase exit"
date: 2026-10-24
phase: REVIEW
sev: SEV-0
authority: HUM LEAD
status: "OPEN — grows per batch. BUILD-exit P10 recorded; REVIEW-exit P10 to come (D-252)."
---

# Gate roster

Up: [build log](../04-development/build-log.md) · [rulings](../02-analysis/rulings.md)

**A gate this release adds or changes carries a failure that was watched**, as 0.16.0's and 0.18.0's
rosters do: a **plant** (a defect introduced, the gate run, seen to fire), a **mutant** killed, or **by
construction**. A gate leaves `06_docs/required-gates.txt` only by a HUM LEAD ruling recorded here.
**P10 is recorded here at each phase exit (D-252)**: the ledger lives on one disk, so this table is where a
reader finds a dated run.

## Added and changed

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| The pre-push hook (`make hooks`, `tools/prepush`, `tools/prehook`; W0.1, A-60, A-71) | The tests that read the record run before a push, from the tree the push carries | Build log batch 32: watched failures, each restored by copy and compared; batch 42 (H-9, I-8) |
| `tools/runlog` (H-8, A-65) | Each `make verify` and `make verify-docs` run is a line in `06_docs/verify-runs.md`, its tree and result | Build log batch 36 |
| `tools/runlog`, the tree before and after (RC-3, D-245) | The tree is taken before the run; one changed during it makes the line FAILED and the run's exit not 0 | `TestATreeChangedDuringTheRunIsNoPass`; its mutant (batch 59) |
| `make verify-docs`, a refusal not logged (RD-12) | `tools/docslane` refuses before `runlog` starts, so the record holds runs, never refusals | By construction: the recipe's order |
| `perfsum -g1` (W10.2, A-79, A-80, A-83) | G1's arms against its targets; the hourly variant at 3.3 MB an hour | Build log batches 51, 52 and 58; mutants m495 to m507, m521 to m523 |
| `perfsum -g1`, failing closed (RC-1, D-273) | An RSS that does not read refuses the run; a cycle the driver could not confirm is a miss; a default run shorter than 48 hours by more than one 2-minute sample exits missed, one within it taken as 48 and said so; `workload.sh` exits 1 when a driver failed | `TestAG1RunThatCannotBeJudgedFails`; its mutants (batch 59) |
| `tools/finaltag`, in the release workflow (RH-9, D-3) | A final tag fails while `go.mod` requires a paired library's pre-release | `TestAFinalTagIsRefusedOnAReleaseCandidate`; run on the tree at batch 59, it refused v0.19.0 on rc.5 and rc.3 |
| `tools/p10floor`, in `make p10` (D-264) | Each density row whose reason states `FLOOR x.xx` fails when its package measures under it; a figure that does not read fails closed | `TestARowUnderItsFloorFails`; its `--self-test` in `gate-controls`; mutants m537, m538 (batch 61); run on a copy of the report with `modes/tty` lowered to 0.42, it failed naming the row |
| `make test-say`, a mute host (D-254) | A host whose `say` renders nothing fails under the leg, unless the run declares it mute, when the recipe prints NOT RUN | Its mutant (batch 59); the hosted macOS step's NOT RUN line |
| `tools/finaltag` on the go tool's parser (R2H-3) | go.mod is read by `go mod edit -json`, so a `replace`, CRLF endings or a trailing space cannot hide a requirement; under a final tag any `replace` of a paired library is refused | `TestAFinalTagIsRefusedOnAReleaseCandidate`'s four cases; mutants m553 and m554 (batch 66) |
| The release workflow runs `finaltag` (R2H-3) | `release.yml` runs `finaltag` on its tag before `make verify` | `TestTheReleaseRunsFinaltag`; mutant m555, the step deleted (batch 66) |
| `tools/p10floor`, a floorless row (D-277) | A firing density row whose reason states no floor fails, named with what it measures | `TestARowUnderItsFloorFails`; the `--self-test`; mutant m543 (batch 66) |
| `make test-say`, one diagnosed retry (D-276) | A render killed at its 2-minute bound has its main thread logged and is tried once a run, the retry printed; a second hang fails; the leg's 11 minutes are its renders' own | `TestAHungRenderIsTriedOnceMore`; mutants m545 and m546 (batch 66) |
| `tools/runlog`, a run with no record (RC-3) | A run whose line cannot be written, whose tree cannot be read again at its end, or that runs outside a repository exits failing | `TestARunWithNoRecordIsNoPass`; mutants m556, m565 and m566 (batches 66 and 67) |
| The run-selector check, build tags (RC-9) | A `-run` counts a test only where the recipe's `-tags` and `-race` build its file | `TestTheRunSelectorGuardCatchesAnEmptySelection`'s tag and race cases; mutants m557 and m567 (batches 66 and 67) |
| The libraries' gates, a refusal (D-280) | go-ionomaps' and go-tuiMaps' `scripts/gate` refuse the docs lane with nothing staged, or with a file that is not Markdown, before any leg, and log nothing | go-ionomaps `TestADocsLaneRefusalIsNoRun` (its m038); go-tuiMaps `TestEveryRunIsLogged` (its m089) |

## Phase-exit runs

`make p10` is not in `verify` or CI; it runs at each phase exit, with the ledger on the HUM LEAD's
machine, and its result is recorded here.

| Exit | Date | Result | Rulings |
|---|---|---|---|
| BUILD | 2026-10-18 | `make p10`: 0 live, 0 unmatched, 0 unratified (`08-reports/build-report.md`) | D-233 |
| REVIEW | | to come | D-252 |
