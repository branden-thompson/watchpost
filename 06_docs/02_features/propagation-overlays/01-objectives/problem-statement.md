---
title: "0.19.0 Propagation overlays — problem statements"
date: 2026-10-05
phase: DISCOVER (intake)
sev: SEV-0
authority: HUM LEAD
status: "Statements LOCKED (D-7); metrics ruled (D-8, D-9), targets set in DISCOVER"
---

# Problem statements

Two primary statements (D-6), locked as written (D-7). Each carries its own metrics.

## PS-1 — the Broadcaster operator

> A station operator relaying watchpost's broadcasts on amateur HF cannot tell, from the station,
> whether the band they transmit on will reach the area they serve at that hour, so a broadcast can go
> out on a band that carries it nowhere near its listeners, and the operator does not learn of it.

## PS-2 — the Observer ham

> An amateur HF operator using watchpost cannot tell, without leaving it, which bands will carry a
> signal between their place and the places they want to reach at a given hour, so they choose by habit
> or from a separate website, and call into bands that are closed or miss ones that are open.

## Scorecard

| Criterion | PS-1 | PS-2 |
|---|---|---|
| Bad outcome | a broadcast that reaches nowhere near its listeners, unnoticed | closed bands called, open ones missed |
| Affected humans | the Broadcaster station operator on HF | the Observer listener who operates HF |
| Tech agnostic | no technology named | no technology named |
| Non-prescriptive | a map, a table, a spoken line or a band chip could answer it | the same |
| Verifiable | the band chosen against observed propagation for that path and hour | the same |

Why HF only: MUF governs 3-30 MHz. FRS and GMRS, which the Broadcaster also serves (issue #25), are UHF,
where it has no bearing.

## Metrics

Each is written as name · symbol · type · definition · measured in, and validated against the
anti-solution check before it locks.

### PS-1 (D-8)

Targets are set in DISCOVER and the PLAN-time dry run (D-8). M1 and M4 are measured against WSPR via
wspr.live, with RBN as a cross-check (D-22), and their baseline is the IRI climatology, not KC2G (D-23).
The numbers proposed at intake are kept as a starting point.

| # | Name · symbol | Type | Definition | Measured in | Proposed at intake |
|---|---|---|---|---|---|
| M1 | Reach agreement · R₁ | accuracy | share of scenarios (station place, served area, UTC hour, band) where the station's answer, reaches or does not, matches observed reception reports for that path within ±1 h; the set holds both outcomes | % of scenarios | at least the IRI climatology's agreement plus N points (D-23; first proposed as KC2G's, less 5) |
| M2 | Answerable from the station · A₁ | task (UAT) | the operator answers the reach question for N scenarios from the station alone, without leaving watchpost, correctly per M1's reference | correct of N; seconds each | 8 of 10, each within 30 s |
| M3 | Told when it closes · L₁ | latency | from the first published update in which the operator's band stops reaching the served area, to the station saying so on screen and in words | minutes | at most 20 (one 15-minute update period and slack) |
| G1 | Station cost · C | guardrail (issue #25) | watchpost's added CPU time and resident memory with propagation on against off, over one hour | % CPU; MB | at most 1% mean CPU; 50 MB RSS |

Anti-solution check:
- M1: a constant answer fails on the mixed set, and so does a well-drawn map with wrong numbers.
- M2: a link out fails ("without leaving"). A spoken line passes as well as a map, so the metric does not prescribe a map.
- M3: it measures whether the operator learns, which is PS-1's last clause.
- G1: holds any solution to issue #25's resource limit.

Open for DISCOVER:
- ~~which reception reports are the reference~~: ruled, WSPR via wspr.live with RBN as a cross-check (D-22);
- how the station learns the operator's band and served area.

### PS-2 (D-9)

Targets are set in DISCOVER, as for PS-1. G1 is shared with PS-1.

| # | Name · symbol | Type | Definition | Measured in |
|---|---|---|---|---|
| M4 | Path agreement · R₂ | accuracy | share of scenarios (listener's place, target place, UTC hour, band) where watchpost's open or closed matches observed reception reports for that path within ±1 h; the set holds both outcomes | % of scenarios |
| M5 | Answerable without leaving · A₂ | task (UAT) | the listener names, from watchpost alone, the bands open between their place and N target places at a given hour, correctly per M4's reference | correct of N; seconds each |
| M5b | The same, in words · A₂ʷ | task (UAT) | M5 answered with no picture (`--ascii`, or "Instead of the map") | correct of N; seconds each |

Anti-solution check:
- M4: a constant answer fails, as for M1.
- M5: a link out fails.
- M5b: an answer that exists only as colour on a map fails. MUF's words path therefore ships with the layer (D-9).
