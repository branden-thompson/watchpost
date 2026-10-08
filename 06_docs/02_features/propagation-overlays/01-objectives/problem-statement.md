---
title: "0.19.0 Propagation overlays — problem statements"
date: 2026-10-05
phase: DISCOVER (intake)
sev: SEV-0
authority: HUM LEAD
status: "Statements LOCKED (D-7); metrics ruled (D-8, D-9), revised by D-73 (what open means), D-75 (floors) and D-76 (M3 retired); targets set from PLAN's dry run (D-49)"
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

Targets are set from PLAN's dry run, each by its own ruling, before PLAN exits (D-49, amending D-8), against
the floors set before it (D-75). M1 and M4 are measured against WSPR via wspr.live, with RBN as a
cross-check (D-22); only spots that, normalised to the reference circuit, would carry SSB voice at 100 W
count (D-73). Their baseline is the IRI climatology, not KC2G (D-23). The numbers proposed at intake are
kept as a starting point. **UAT (M2, M5, M5b) is graded by the HUM LEAD alone; the VALIDATE report says so
(D-68).**

| # | Name · symbol | Type | Definition | Measured in | Proposed at intake |
|---|---|---|---|---|---|
| M1 | Reach agreement · R₁ | accuracy | share of scenarios (the tower, a place in the served area, UTC hour, band) where the reference chart's answer ("best bands now" or "your frequency", D-76), reaches or does not, matches observed reception reports normalised to the reference circuit (D-73) for that path within ±1 h; the set holds both outcomes | % of scenarios | at least the IRI climatology's agreement plus N points (D-23; first proposed as KC2G's, less 5) |
| M2 | Answerable from the station · A₁ | task (UAT) | the operator answers the reach question for N scenarios from the station alone (the Propagation mode, opened from the console, D-57), without leaving watchpost, correctly per watchpost's own answer for that question (D-92; accuracy is M1's) | correct of N; seconds each | 8 of 10, each within 30 s |
| ~~M3~~ | ~~Told when it closes~~ | — | **Retired by D-76:** the reference chart replaced the monitoring readout and its closure notices; the operator learns by looking. An optional "band watch" is a follow-up, and M3 returns with it (D-48, D-72 and D-74 recorded its split and targets). | — | — |
| G1 | Station cost · C | guardrail (issue #25) | watchpost's added CPU time, resident memory and downloads with the Propagation mode in use against not, over at least 48 hours (D-49) | % CPU; MB; MB a day | at most 1% mean CPU; 50 MB RSS (target set from the dry run, D-49) |

Anti-solution check:
- M1: a constant answer fails on the mixed set, and so does a well-drawn map with wrong numbers.
- M2: a link out fails ("without leaving"). A spoken line passes as well as a map, so the metric does not prescribe a map.
- M3: retired (D-76). PS-1's last clause ("the operator does not learn of it") is left to the band-watch follow-up.
- G1: holds any solution to issue #25's resource limit; measured over at least 48 hours, with downloads a day (D-49).

Answered in DISCOVER:
- the reference: WSPR via wspr.live, with RBN as a cross-check (D-22);
- the operator's bands: every band is answered in the reference chart, and the operator's own frequency can be entered (D-76; D-30's picker retired); the served area: the configured tower and service radius (`config.Broadcaster`).

Note: watchpost transmits nothing; the Broadcaster makes audio on the computer, and any transmission is the
operator's, under the laws where they are (D-45, D-46).

### PS-2 (D-9)

Targets are set from PLAN's dry run, as for PS-1 (D-49), against D-75's floors. G1 is shared with PS-1.

| # | Name · symbol | Type | Definition | Measured in |
|---|---|---|---|---|
| M4 | Path agreement · R₂ | accuracy | share of scenarios (listener's place, target place, UTC hour, band) where watchpost's open or closed matches observed reception reports normalised to the reference circuit (D-73) for that path within ±1 h; the set holds both outcomes | % of scenarios |
| M5 | Answerable without leaving · A₂ | task (UAT) | the listener names, from watchpost alone, the bands open between their place and N target places at a given hour, correctly per watchpost's own answer (D-92; accuracy is M4's) | correct of N; seconds each |
| M5b | The same, in words · A₂ʷ | task (UAT) | M5 answered with no picture (`--ascii`, or "Instead of the map") | correct of N; seconds each |

Anti-solution check:
- M4: a constant answer fails, as for M1.
- M5: a link out fails.
- M5b: an answer that exists only as colour on a map fails. MUF's words path therefore ships with the layer (D-9).

## Targets (D-49)

Each target is ruled from PLAN's dry run, against D-75's floors, and listed here as it is ruled. **On a miss (D-118):** a floor cuts its feature (D-75); any other missed target comes to the HUM LEAD at VALIDATE with its evidence, never re-set silently. M1 and M4 become "no worse than climatology" if the climatology's measured agreement is above 90%.

**Measured on (D-F15):** every target was set from one week (2026-09-29 to 10-05; F10.7 92 to 100; one G1 to G2 storm; four mainland-US stations) and from spikes on one machine (an Apple M5 Pro). G-M3's near-station limits come from the whole week, tuning days included. The Pacific is no better than the climatology and is reported separately (D-116). M1 and M4's baseline is not yet measured (W10.3).

| Metric | Target | Ruling |
|---|---|---|
| M2 | 8 of 10 correct, each within 30 s | D-93 |
| M5 | 8 of 10 correct, each within 45 s | D-93 |
| M5b | 8 of 10 correct, each within 60 s | D-93 |
| G1 | added mean CPU ≤ 1% of one core; added RSS ≤ 25 MB (split, D-112: the library's live heap ≤ 15, the host's conversion ≤ 4, GC headroom the rest); downloads counted on the wire in decimal MB, **re-ruled in BUILD's first task from a measured gzip request with 25% headroom** (D-95's 15 and 3 MB an hour were decoded estimates), none closed | D-95, D-111 |
| go-tuiMaps M5 | a whole-globe 2° field frame after a pan (fill, labelled contours, terminator, night) ≤ 16 ms at 400 × 110, ≤ 8 ms at 200 × 56 | D-96 |
| M1 | agreement ≥ the IRI climatology's + 5 points, on the same ≥ 200 scenarios with both outcomes, over ≥ 3 days, by band | D-99 |
| M4 | as M1, for the listener's paths | D-99 |
| go-ionomaps G-M3 | ≥ 15% below climatology in the mainland US, ≥ 10% overall, foF2 and MUF; near stations foF2 ≤ 0.5, MUF ≤ 1.6 MHz | D-108 |
| go-ionomaps G-M2, G-G1 | see D-97, D-98 | D-97, D-98 |

