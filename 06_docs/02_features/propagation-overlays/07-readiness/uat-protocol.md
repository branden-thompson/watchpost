---
title: "0.19.0 — UAT protocol for M2, M5 and M5b"
date: 2026-10-07
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "DRAFT for the PLAN gate. The sitting is live (D-89)."
---

# UAT protocol: M2, M5, M5b

**Graded by the HUM LEAD alone (D-68).** The VALIDATE report says the three metrics were graded by one
person. Definitions are in `01-objectives/problem-statement.md`; targets are ruled from the dry run (D-49).

## What each sitting asks

| Metric | Surface | Ten questions of the form | Correct when |
|---|---|---|---|
| M2 (PS-1) | the Broadcaster: the console key into the Propagation mode, from the tower (D-57) | "Does *frequency* reach *place in or near the served area* now?" | the answer matches M1's reference for that path within ±1 h |
| M5 (PS-2) | the Observer's map window, Propagation mode, picture on | "Which bands are open now between *your place* and *target*?" | every band named matches M4's reference: open under the reference circuit (D-73) |
| M5b (PS-2) | the same, with no picture (`--ascii`, or "Instead of the map") | as M5 | as M5, from the words alone (FR-3.1) |

**Seconds each:** from the question shown to the answer stated. A prompter script shows each question and records the times on a key press; it is built with the instruments (W10).

**Without leaving:** any answer that needed another program, a browser or a link out is marked wrong (the anti-solution checks).

## The scenarios

- **Drawn in BUILD**, after the chart's answers exist, and fixed before the sitting (D-89).
- Each set holds both outcomes:
  - for M2, reaches and does not;
  - for M5 and M5b, at least two targets with no band open and two with three or more.

  A constant answer therefore fails (the M1 and M4 anti-solution checks).
- Targets come from the chart's defaults (the continents, FR-1.15) and from places found by city, ZIP, coordinates and locator (FR-1.10), at least one of each.
- **M5b has its own ten, drawn the same way as M5's**, so remembered answers from M5 cannot carry over (A-25).

## The reference (the answer key)

M1's and M4's reference (D-22, D-73):
- WSPR spots from wspr.live, with RBN as a cross-check;
- each spot normalised to 100 W from its reported power (WSPR's SNR is already given in 2.5 kHz);
- a path counts as open when normalised spots clear about +10 dB within ±1 h.

The threshold is checked against published practice in PLAN (D-73; `dry-run.md`). Paths with no spots in either direction are left out of the key, never scored as closed.

## How the sitting is held (D-89): live

- **The binary:** the release candidate, on live data at the sitting's hour.
- **Fixed before the sitting:**
  - the targets, bands and questions, chosen so outcomes likely differ;
  - the key's procedure.
- **The key** is computed after the sitting from that hour's spots, by that procedure.
- **Rescheduled** if GIRO is paused during the sitting, or if the computed key does not hold both outcomes. Nothing is scored from a set that fails its own anti-solution check.
- No replay build is made.

## Recording

- Each sitting's answers, times and key go in `07-readiness/uat-results.md`.
- Each verdict becomes a D-row, listed in the build log batch that carries it (FR-10.5, REFLECT L8).
