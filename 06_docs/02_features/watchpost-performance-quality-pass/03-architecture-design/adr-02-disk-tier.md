# ADR-02 — The disk cache tier stays, bounded

**Status:** accepted (PLAN; built in Q1) · **Owner:** HUM LEAD

## Context
DISCOVER found the disk tier unbounded (1,376 files / 116 MB after two days, 593 orphans) and 95 % of
writes persisting entries that could never serve a relaunch (L4-F1, L4-F2).

## Options
1. Keep the tier; add a persistence floor, one retention rule, an allow-list sweep, a cap.
2. Replace it with an embedded size-capped store (bbolt) — a new dependency and failure modes.
3. Drop the tier except for the ≥ 2 MB station lists — loses warm relaunches for hourly forecasts.

## Decision
Option 1 (`platform/httpx/cache.go`): caller TTL > 5 min or `Persist()` to write; expired-with-validators
kept 24 h; stale memory entry skips the disk read; the sweep touches only names it wrote, refuses to run
outside a `watchpost` directory, caps at 256 MB oldest-first.

## Consequences
Disk writes fell from ~45k/day to ~10k/day in the Q1 gate run; the start sweep cut the live directory
from 1,429 files / 115 MB to 798 / 38 MB. Q5's conditional GETs renew from the retained entries.
