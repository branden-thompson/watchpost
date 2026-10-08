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
| M2 (PS-1) | the Broadcaster: the console key into the Propagation mode, from the tower (D-57) | "Does *frequency* reach *place in or near the served area* now?" | the answer matches watchpost's own answer (D-92) |
| M5 (PS-2) | the Observer's map window, Propagation mode, picture on | "Which bands are open now between *your place* and *target*?" | the bands named are exactly the bands watchpost calls open (D-92) |
| M5b (PS-2) | the same, with no picture (`--ascii`, or "Instead of the map") | as M5 | as M5, against the words alone (FR-3.1). **It measures a sighted reader of the words**: the terminal cursor does not yet follow focus (F-205, D-110), and VALIDATE says so |

**Seconds each:** from the question shown to the answer stated. A prompter script shows each question and records the times on a key press; it is built with the instruments (W10).

**Without leaving:** any answer that needed another program, a browser or a link out is marked wrong (the anti-solution checks).

## The scenarios

- **Drawn in BUILD**, after the chart's answers exist, and fixed before the sitting (D-89).
- Each set holds both outcomes:
  - for M2, reaches and does not;
  - for M5 and M5b, at least two targets with no band open and two with three or more.

  A constant answer therefore fails (the M1 and M4 anti-solution checks).
- Targets come from the chart's defaults (the continents, FR-1.9) and from places found by city, ZIP, coordinates and locator (FR-1.10), at least one of each.
- **M5b has its own ten, drawn the same way as M5's**, so remembered answers from M5 cannot carry over (A-25).

## The answer key (D-92)

**Scored:** watchpost's own answer to each question at the sitting's hour, read from its words and recorded by the prompter as the sitting runs (the words path, FR-3.1, so it holds for M5b too).

**Reported beside it, not gated:** M1's and M4's reference (D-22, D-73, D-91):
- WSPR spots from wspr.live, with RBN as a cross-check;
- each spot normalised to 100 W from its reported power (WSPR's SNR is already given in 2.5 kHz);
- a path counts as open when normalised spots clear +13 dB within ±1 h (D-91, ITU-R F.339-8).

Paths with no spots in either direction are left out of the key, never scored as closed.

## How the sitting is held (D-89): live

- **The binary:** the release candidate, on live data at the sitting's hour.
- **Fixed before the sitting:**
  - the targets, bands and questions, chosen so outcomes likely differ;
  - the key's procedure.
- **The scored key** is watchpost's own answer, captured during the sitting. **The WSPR key** is computed after it from that hour's spots, by the procedure fixed before, and reported.
- **Rescheduled** if GIRO is paused during the sitting, or if watchpost's answers do not hold both outcomes. Nothing is scored from a set that fails its own anti-solution check.
- No replay build is made.

## Recording

- Each sitting's answers, times and key go in `07-readiness/uat-results.md`, **redacted (D-115)**: places as a region or a 4-character locator, no callsigns. The raw prompter logs and WSPR rows stay outside the tree, with the PLAN evidence.
- Each verdict becomes a D-row, listed in the build log batch that carries it (FR-10.5, REFLECT L8).
