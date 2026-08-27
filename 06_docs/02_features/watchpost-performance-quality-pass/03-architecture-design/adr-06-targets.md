# ADR-06 — Targets and the growth statistic (C5)

**Status:** ratified (HUM LEAD, 2026-08-26; restated at the Q0 gate) · **Owner:** HUM LEAD

## Context
DISCOVER found no growth term in code; the pass must prove it over weeks, not assert it (OQ-8).

## Decision
Plan §1 as restated at Q0: idle ≤ 90 MB, plateau ≤ 160 / peak ≤ 220 MB (fragmentation check only);
**growth** = post-GC `HeapAlloc` every 5 min, per-hour minimum, HAC OLS slope, upper 95 % CI × 30 d <
max(5 % of the post-GC heap plateau, the run's measured detection floor) — the floor printed beside
every verdict (`tools/slope`: PASS / GROWTH / UNCERTIFIABLE / INSUFFICIENT); every gauge flat at 24/48/72 h
is the primary evidence; threads ≤ a constructed bound and stable; allocations gated per frame (not
time); outage ≤ 3× healthy with alerts never delayed; warm launch ≤ 550 ms median of 10; non-kit P10
exemptions ≤ 53 with 0 unmatched. Q7's macOS soak runs 7 days (floor ≈ 3.8 σ), Arch 72 h.

## Consequences
A one-hour run proves the apparatus, never the bar (Q0 gate finding). Every later batch's build log carries
a before/after table against these numbers; the baseline document (Q7) carries them with commands.
