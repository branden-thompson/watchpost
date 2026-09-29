---
title: "0.18.0 — the gate roster: gates added and retired, each with a failure that was WATCHED"
date: 2026-09-29
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "OPEN — grows per batch."
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
| `TestNoListComparesZips` (batch 55, #23) | No list in `app` or `modes/tty` decides "same place" by ZIP; `snapshot.PlaceID` is the one definition | **Run against batch 54's `app/refs.go` and `modes/tty/dashboard.go`: CAUGHT all five old sites.** Its control asserts the pattern still sees #23's two shapes |
| `TestTheInjectorIsInEveryBuild` (batch 56, #9) | The plain build supplies the injector and its scenarios; no file names the retired tag | **By the watched RED**: before the change it failed on the missing hook, the missing scenarios and each tagged file |
| `TestATestEventIsMarkedOnEverySurface` (batch 56, #9) | NFR-2 restated: every scenario, through the window's hook and one real cycle, is marked on the tape, the band, [w] and the read | **On its first run it CAUGHT a real defect**: the tape's Emergency items dropped the mark (`laneItems`). Plants 2026-09-29: the mark dropped in `itemsOf`, in [w]'s rows, in `laneItems`, and the injector minting unmarked events - each CAUGHT |
