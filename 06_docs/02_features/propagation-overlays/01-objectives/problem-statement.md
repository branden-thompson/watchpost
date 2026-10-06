---
title: "0.19.0 Propagation overlays — problem statements"
date: 2026-10-05
phase: DISCOVER (intake)
sev: SEV-0
authority: HUM LEAD
status: "Statements LOCKED (D-7); metrics not yet set"
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

Not yet set. Each is written as name · symbol · type · definition · measured in, and validated against
the anti-solution check before it locks.
