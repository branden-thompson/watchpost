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
| `perfsum -g1`, failing closed (RC-1) | An RSS that does not read refuses the run; a cycle the driver could not confirm is a miss; a default run under 48 hours exits missed; `workload.sh` exits 1 when a driver failed | `TestAG1RunThatCannotBeJudgedFails`; its mutants (batch 59) |
| `tools/finaltag`, in the release workflow (RH-9, D-3) | A final tag fails while `go.mod` requires a paired library's pre-release | `TestAFinalTagIsRefusedOnAReleaseCandidate`; run on the tree at batch 59, it refused v0.19.0 on rc.5 and rc.3 |
| `make test-say`, a mute host (D-254) | A host whose `say` renders nothing fails under the leg, unless the run declares it mute, when the recipe prints NOT RUN | Its mutant (batch 59); the hosted macOS step's NOT RUN line |

## Phase-exit runs

`make p10` is not in `verify` or CI; it runs at each phase exit, with the ledger on the HUM LEAD's
machine, and its result is recorded here.

| Exit | Date | Result | Rulings |
|---|---|---|---|
| BUILD | 2026-10-18 | `make p10`: 0 live, 0 unmatched, 0 unratified (`08-reports/build-report.md`) | D-233 |
| REVIEW | | to come | D-252 |
