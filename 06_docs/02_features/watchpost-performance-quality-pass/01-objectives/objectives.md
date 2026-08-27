# Watchpost Performance & Quality Pass — Objectives

Source of truth: [`../08-reports/project-brief.md`](../08-reports/project-brief.md) for the problem,
metrics and requirements; **numeric targets live in the plan of record §1**
([`../03-architecture-design/quality-pass-plan.md`](../03-architecture-design/quality-pass-plan.md), C5)
once ratified — the table below records the brief's provisional values.

## Problem Statement (locked, 5/5)

> People who leave Watchpost running for days cannot tell whether its slowly rising resource use is normal
> or a defect that will eventually degrade the terminal session they live in — and the record cannot tell
> them either.

## Metrics of Success

| ID | Metric | Type | Target (provisional — OQ-5) |
|---|---|---|---|
| M1 | Long-run resource stability (`M-STAB`) | Primary | no growth term (weeks/months of continuous operation, OQ-8); measured over multi-day runs on macOS and Linux; footprint ≤ 160 MB with radio on, ≤ 90 MB idle (provisional); threads ≤ 32 and attributed |
| M2 | Steady-state chattiness (`M-CHAT`) | Secondary | measured, then no more than cadence math predicts |
| M3 | Zero regression (`M-REG`) | Primary | gates, goldens, pty smokes, Linux re-run all PASS; warm launch ≤ 550 ms |
| M4 | Code-quality floor (`M-P10`) | Maintenance | P10 0 live; non-kit exemptions ≤ 54 (plan §1: 56 − 4 stale − 2 one-liners + 2 Q0 build-tag entries + 1 `tools/slope` density entry, ratified at the Q0 gate; + 1 `platform/declset` density entry, ratified at the Q2 gate), 0 unmatched ledger entries; coverage ≥ today |
| M5 | Record completeness (`M-DOC`) | Maintenance | every change/non-change has rationale + evidence + a pin |

## Requirements

R1 no functional regression · R2 P10/style, value-only exemptions · R3 junior-dev-readable documentation of
every change · R4 best practices · R5 no performance regression · R6 radio/synth cannot regress · R7 the
record is a corpus and a comparison point.

## Scope

In: every package outside `third_party/`; `third_party/go-studs` on the capture → document → approve →
change basis (HUM LEAD owns it). Out: `tools/` unless a finding lands there; new features.

## Baseline

`../02-analysis/baseline-pid67943.log` — samples of the v0.9.4 process HUM LEAD left running (started
2026-08-25 17:55 local). One hourly row at 2026-08-26 10:21 UTC, then a gap (the sampler died — red-team
DQ-1/RT-1); restarted 11:48 UTC and moved to a 5-minute cadence at 12:02 UTC for 24 h (289 rows). The
instrumented-run artefacts (`discover-run/`) are copied in at Q0 task 0.
