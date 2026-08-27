# ADR-03 — RECENT scheduling stays one scheduler per location (C3 = A)

**Status:** accepted (PLAN, HUM LEAD 2026-08-26) · **Owner:** HUM LEAD

## Context
The RECENT list runs 50 schedulers × 5 tiers ≈ 250 parked goroutines (LR-2). DISCOVER proposed a single
heap scheduler (option B) for −250 goroutines and batchable fetches.

## Options
A. Keep 50 schedulers; record the ~1 MB cost.
B. One heap scheduler with per-location phases and a worker pool.

## Decision
A. Goroutine count is not a metric (BQ-4, RT-8); the batching benefit is delivered by the HMS/WFIGS memos
(Q3) and the FIRMS tiles (Q5) without a new scheduler; a single pool would let CO-OPS pacing starve
observation rows (PF-5); Q6 landing after Q5 would invalidate Q5's network gates (PA-3).

## Consequences
The publish consolidation (one publish per tier tick) moves to Q3. B is re-opened only if Q0 counters
after Q3 + Q5 show > 5 MB live or publish-coalescing failures attributable to the 50 schedulers — then as
the per-tier phased variant with per-kind lanes and PR-7's pins preserved.
