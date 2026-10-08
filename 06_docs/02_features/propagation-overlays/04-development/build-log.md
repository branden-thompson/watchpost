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
