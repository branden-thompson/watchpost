---
title: "0.18.0 — the gate roster: gates added and retired, each with a failure that was WATCHED"
date: 2026-09-29
phase: BUILD exit
sev: SEV-0
authority: HUM LEAD
status: "OPEN — grows per batch. BUILD-exit P10 recorded (2026-10-05); REVIEW-exit P10 to come."
---

# Gate roster

**A gate this release adds carries an evidence line naming a failure that was actually watched**, as
0.16.0's roster does: a **plant** (a defect introduced, the gate run, seen to fire) or **by
construction**. **A gate leaves `06_docs/required-gates.txt` only by a HUM LEAD ruling recorded
here.**

## Retired

| Gate | Why | Ruling |
|---|---|---|
| `test-tags` | It ran `go test -tags watchpost_debug ./app`. With the injector untagged (D-152) its tests are ordinary tests, run by `race` and `test` with every other | **D-153**, 2026-09-29, HUM LEAD: "Retire both (Recommended)" |
| `lint-injector` (in `release-matrix`, and its self-test in `gate-controls`) | It failed any release artifact carrying the injector. Every artifact carries it by design now (D-152); `TestTheInjectorIsInEveryBuild` guards the other direction | **D-153**, as above; `build-diag`, which inverted it, retired with it |

## Added

| Gate | What it asserts | Evidence: a failure watched |
|---|---|---|
| `lint-plan-code` (W0.3, 2026-09-23) | No plan document carries implementation code: signatures and API shape only (D-13, FR-8.1). Its own self-test runs first. In `verify`, `verify-docs` and CI; in `required-gates.txt` | **By construction**: `go run ./tools/plancode -self-test` runs before every check and fails unless every planted body is refused and every planted signature passes |
| `verify-docs` (D-38, 2026-09-23) | A change that is Markdown alone runs every test uncached, `test-say`, `lint-authoring`, `lint-watermark` and `lint-plan-code`; `tools/docslane` refuses the change if any file is not Markdown. Local only, so not in CI; registered as unlisted in `cmd/watchpost/gates_test.go` with its reason | **Plants (D-38)**: the tool's own tests kill four mutations - untracked files ignored, unstaged changes ignored, the comparison dropped, one stray file allowed. **Watched in use, batch 122**: the lane failed on all nine ratified P10 rows when `TestEveryRatifiedP10RowNamesCodeThatExists` could not find `Router.Init` |
| `test-say` (D-58, 2026-09-24) | `TestSayVoiceOnDarwinNarratesHostileTextSafely` runs alone, with nothing else loading the machine, and passes with the listener's watchpost open; the parallel suite skips it. In `verify`, `verify-docs` and CI's macOS leg (it drives macOS `say`); in `required-gates.txt` | watched failure: not recorded. The failure that made the gate is D-58's: the test killed at its 120 s bound inside the parallel suite while passing alone in about 3 s (F-176) |
| `property` (batch 11, W2.4) | Guard 3 at its full 10,000 sequences: after random pan, zoom, resize, units, data and ticks, the printed map equals a fresh render of the same inputs (`TestThePrintedMapIsAFreshRender`, `-tags property`). Every `go test` runs 200. In `verify` and CI; in `required-gates.txt` | watched failure: not recorded for the window. Recorded: the guard's first two versions failed against a correct window - their second map skipped the render that asks for its tiles. `TestTheThreeGateListsAgree` then caught `property` in `verify` and on no other list (batch 11) |
| `TestTheAsBuiltMapIsKeptInStep` (batch 5, `app/maps_test.go`) | The as-built page's `Batches 1–N` names the build log's latest `## Batch N`: a batch that lands without its diagrams fails. Runs in `race` and every `go test` | **Watched, batch 135**: CI's run was red on it - the batch's build-log entry was written after its local verify (red team HY-3a) |
| `TestNoScratchTestIsInTheTree` (batch 17, `cmd/watchpost`) | No `zz_` test file and no `TestZZ` function on disk, tracked or not. Runs in `race` and every `go test` | **Plant (batch 17)**: proven against a probe file |
| `TestNoListComparesZips` (batch 55, #23) | No list in `app` or `modes/tty` decides "same place" by ZIP; `snapshot.PlaceID` is the one definition | **Run against batch 54's `app/refs.go` and `modes/tty/dashboard.go`: CAUGHT all five old sites.** Its control asserts the pattern still sees #23's two shapes |
| `TestTheInjectorIsInEveryBuild` (batch 56, #9) | The plain build supplies the injector and its scenarios; no file names the retired tag | **By the watched RED**: before the change it failed on the missing hook, the missing scenarios and each tagged file |
| `TestATestEventIsMarkedOnEverySurface` (batch 56, #9) | NFR-2 restated: every scenario, through the window's hook and one real cycle, is marked on the tape, the band, [w] and the read | **On its first run it CAUGHT a real defect**: the tape's Emergency items dropped the mark (`laneItems`). Plants 2026-09-29: the mark dropped in `itemsOf`, in [w]'s rows, in `laneItems`, and the injector minting unmarked events - each CAUGHT |

## Phase-exit runs

`make p10` is not in `verify` or CI; it runs under `make quality` at BUILD and REVIEW exit, and its
result is recorded here.

| Exit | Date | Result | Rulings |
|---|---|---|---|
| BUILD | 2026-10-05 | `make p10`: 0 live, 0 unmatched, 0 unratified; 159 ratified rows | D-248, D-249 |
