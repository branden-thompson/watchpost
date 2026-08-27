# ADR-05 — `debug.SetMemoryLimit` is not a lever (non-decision recorded)

**Status:** declined (red-team round 1, RT-3) · **Owner:** HUM LEAD

## Context
The plateau targets (≤ 160 MB radio on) could be met by a GC ceiling: `debug.SetMemoryLimit` makes the
runtime collect harder as the limit nears.

## Decision
Not used. A ceiling caps footprint without proving the absence of a growth term — the problem statement
is "can the record tell them whether growth is normal or a defect", and a limit hides the answer. The
levers are live-heap ones (geodata loaded once, HMS interning and streaming parse, the body memo) and
the per-structure gauges that prove "flat".

## Consequences
If a future soak shows footprint above the plateau target with every gauge flat, this decision is the one
to revisit — with the gauges as evidence, not instead of them.
