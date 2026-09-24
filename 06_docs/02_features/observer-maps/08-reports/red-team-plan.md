---
title: "0.18.0 — PLAN red team"
date: 2026-09-23
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "ROUND 1 DISPOSITIONED — every finding ruled or applied (D-44 … D-55)."
---

# PLAN red team

**The round's record lives in go-tuiMaps**, because it reviewed both implementation plans and the
integration map together: `go-tuimaps/06_docs/02_features/radar-loops/08-reports/red-team-plan.md`,
with the three reviewer reports filed verbatim beside it in `reviewer-reports/plan-*.md`. Its finding
numbers (P-1 … P-9, B-1 … B-10) are used here.

**Watchpost's rulings from the round land in this feature's `02-analysis/rulings.md`** as they are
made, and are listed below.

| Finding | Watchpost ruling | What changed here |
|---|---|---|
| P-1 — the redraw signal | **D-45** (with go-tuiMaps D-66) | The map renders on every event it owns; D-41's memo key and guard 4 withdrawn; a freshness property replaces them (W2.1–W2.5, W8.10, W9.6; approach 1's diagram) |
| P-2 — M5 on the cold path | **D-46** | M5 keeps "complete" as every alert's area, target ≤ 3.5 s p90 (W2.9); W3.9 reverted to the existing bound of six |
| P-3 — what a radar request reveals | **D-47** | Radar requested per fixed state-regional region (W8.3, W8.3a); disclosure says so (W1.12); `BBOX` redacted (W8.14a); "region" reading awaiting confirmation |
| P-4 — playback semantics | **D-48** (with go-tuiMaps D-67) | The listener's control flow: open on "right now", play from the oldest through observed to forecast, stop, reset, ←/→ scrub, a loading indicator (W8.9, W8.9a, W8.9b); position by valid time. Pan and zoom's phase awaiting confirmation |
| P-5 — loop size and MRMS | **D-50** (with go-tuiMaps D-68) | Two hours of loop, 24 frames at 5 minutes, per region, 6 MiB budget; MRMS at 5 minutes by default; the map source and radar step are Settings (FR-9.5) |
| P-6 — integrating early | **D-51** (with go-tuiMaps D-69) | P1-b builds on go-tuiMaps release candidates; ships on the final tag (FR-5.6, W8.1) |
| P-8 — voice and motion default | **D-52** | The description is text in 0.18.0; speaking it joins one voice pass with the HUM LEAD's other voice bugs (F-181); the motion Setting defaults to slow (W8.9) |
| P-7 — API simplification | (go-tuiMaps D-70) | W9.1 calls `SetBound` on each change of region; W9.4 uses `SetCacheMaxAge` |
| P-9 — metrics | **D-53** (with go-tuiMaps D-71) | M1b re-scored on the shipping description, with an answer-key test; M6 measures ticks and key latency; timings recorded at SHIP (W2.9, W7.2, W8.12, W8.13, W9.2) |
| Readings confirmed | **D-54** | The legend ships in phase 1 with a key for everything shown (W1.17, W9.10); radar regions start state-regional, the scale set in UAT |
| Batch B-1 … B-10 | **D-55** (with go-tuiMaps D-72) | Applied; RK-5 goes to the HUM LEAD in the PLAN report |
| — (follow-up to D-48) | **D-49** | Pan and zoom move into 0.18.0 (FR-1.10, W1.15); every map key control evaluated together before bindings are fixed (FR-1.11, W1.16) |
| — (from the D-17 specimen, go-tuiMaps D-65) | **D-44** | A contextual picture-in-picture legend, `shift+L`, one `[ L ] Legend` chip; ships with the severity digits in P1-b (phase clause awaiting confirmation) |
