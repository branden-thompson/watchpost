---
title: "0.19.0 (with go-ionomaps and go-tuiMaps v0.3.0) — BUILD exit red team, round 1"
date: 2026-10-17
phase: BUILD exit
sev: SEV-0
authority: HUM LEAD
status: "ROUND 1 IN DISPOSITION — eight blind reviewers (D-203), all eight: do not exit BUILD; two Criticals; forks put to the HUM LEAD one at a time; remediation batches follow"
---

# BUILD exit — red team, round 1

Up: [rulings](../02-analysis/rulings.md) · [build log](../04-development/build-log.md) · reviewer notes: [build-round-1](reviewer-notes/build-round-1.md)

## Summary

- **Eight blind reviewers (D-203), over watchpost `5012fae7`, go-ionomaps `d7e7f3c` and go-tuiMaps `b50fc86`. All eight say do not exit BUILD.**
- **Five reviewers found the same first fix: watchpost does not run on the go-tuiMaps it will ship with.** Its `go.mod` pins `9dc4ff3`; P5, P6 and the checklist fix (`b50fc86`) were built and gated in go-tuiMaps alone, so their colour-depth change, shade key, marks and dialer have never run in watchpost or been through UAT. go-ionomaps has no tag and no release checklist; both feature branches are behind released fixes on `main` (go-tuiMaps' merges with conflicts).
- **Two Criticals:**
  - **Accessibility AX-1:** away from the US regions, the Observer's words name the wrong place for the origin - the region last drawn, or "the map" - so a listener who pans or finds a place abroad is told about somewhere they are not looking (`modes/tty/map_describe.go:150-153`, verified).
  - **Business B-F1:** a place found on the Observer becomes the origin, not a target, while FR-1.9 (`requirements.md:77`) and the UAT protocol's M5 (`uat-protocol.md:35`) make found places targets beside the continents (verified).
- **Wrong answers in front of a listener today** (verified): the hours ahead carry the live D-RAP disturbance (`map_prop_ahead.go:94-97`, L-F2); the chart, centre and reach answers stay keyed to a snapshot the no-readings correction has rebuilt (`map_prop_chart.go:44-50`, L-F3); a frequency's reach ignores the lower limit the chart applies (go-ionomaps `reach.go:26-29`, D-F3); found coordinates are spoken "33.2000 megahertz" (`map_prop_words.go:264-282`, AX-2); each hour ahead reads out a developer citation (go-ionomaps `ahead.go:48`, AX-3); a deadline on a slow link is said as "asked too soon" with a wrong cause and a midnight time (go-ionomaps `update.go:110-113,147-148`, L-F4); a mode reopened while its last update is still cancelling joins that update and gets no field (`domains/propagation/propagation.go:111-135`, C-2).
- **The record's gaps:** four tests the requirements name do not exist, one described as "written first" (B-F4, D-F2, verified); the requirements still state FR-1.8's origin and D-77's scope that D-193 and D-195 overturned (D-F1, B-F5); the architecture's library shape is stale (D-F4); go-ionomaps' README says "no code yet" (D-F6); file and line citations have drifted and the check reads only that a line is given (C-10, D-F5).
- **Unmeasured or missed targets:** go-tuiMaps M5 missed 2x on the UI goroutine with the night and the reach drawn (P-3, measured by the reviewer: 33.6 ms against 16 ms at 400x110); G1, R-8.1 and M1/M4 have no number at BUILD exit (P-4, B-F3); FR-1.17's figure is a spike's (P-4, D-F9).
- **Hosted CI, which the agent had not been reading:** batch 28 failed on macOS (`TestThePropagationRowsFollowTheMock`, a wall-clock fixture the agent wrote, at a minute boundary) and batch 32 failed on both systems (the shell hook, since replaced); the tip, batch 34, is green on both.

## The verdicts

| Reviewer | Verdict | Fix first |
|---|---|---|
| Code Quality | Do not exit | C-19: the go-tuiMaps pin |
| InfoSec | Do not exit | I-1: the go-tuiMaps pin; I-2: positions in unrecognised forms reach the online geocoder |
| Accessibility | Do not exit | AX-1: the origin named wrongly abroad (Critical) |
| Performance | Do not exit | P-3: the M5 frame missed 2x on the UI goroutine |
| Docs Quality | Do not exit | D-F3: the reach ignores the lower limit |
| Project Hygiene | Do not exit | H-1: the pin, with H-2's merge from `main` |
| Business Quality | Do not exit | B-F1: a found place is the origin, not a target (Critical) |
| BUILD lens | Do not exit | L-F2: the hours ahead carry today's disturbance |

## The forks for the HUM LEAD, in order

Each is put singly, with its evidence, options, a recommendation and the strongest counter-argument, and recorded verbatim as a D-row.

1. **A found place on the Observer: a target row, or the origin** (B-F1, FR-1.9, M5).
2. **The reach and the lower limit: apply it, or record the gap** (D-F3, R-2.9, FR-1.16).
3. **The library pins: tag go-ionomaps v0.1.0-rc.1 and go-tuiMaps v0.3.0-rc.2 at merged tips, and the visible colour-depth change (L-7.2) in watchpost** (I-1, H-1, H-2, H-7, AX-14, D-F8).
4. **The metrics at BUILD exit: amend D-118 and D-132, or build G10.1/G10.2 and measure G1 inside BUILD** (B-F3, P-4).
5. **The M5 frame: P7's blend memo and the edge pass now, or record the miss** (P-3).
6. **The words' read order: grouping identical targets, the station said once, the weather heading in this mode** (AX-7).
7. **A reader's place on a refresh: anchor the scroll, reserve the lines, or the age on change** (AX-5).
8. **The answers seam: build it, or ratify the departure and correct the architecture** (L-F1).
9. **GIRO's throttle across restarts: persist it, or ratify the risk A-20 left** (L-F7).
10. **A GIRO-free option for go-ionomaps' commercial users, or correct the README** (B-F6).
11. **The P10 workarounds: fix the analysers, or ratify the pattern as a class** (C-7); **go-ionomaps' root and `internal/limits` ledger rows** (C-8, H-3, H-4), with go-ionomaps' A-7 to A-17 put for veto.
12. **The libraries' API corrections before v0.3.0 and v0.1.0 freeze:** `Snapshot.Hours` (C-11, L-F14), go-tuiMaps' two-class edge mark (C-18), its colour-count error kind (C-17, L-F13), and `CheckedDialerFor` adopted or dropped (C-15, I-7).
13. **PS-1's population: evidence, or accepted as a hypothesis** (B-F2).

## The agent's own remediation, without a ruling

Batched after the forks, each batch through its gate with hosted CI read before the next:

- **The ship-together state:** merge `main` into both feature branches (H-2), the libraries tagged as ruled, watchpost pinned and gated on them (I-1, H-1, H-7); the libraries' full hosted gates dispatched (H-10).
- **Wrong answers and words:** L-F2, L-F3, L-F4, L-F5, L-F6, C-2, AX-1, AX-2, AX-3, AX-4 (hours said), AX-6 (Help's hour row), AX-8 to AX-13, P-6, L-F12, I-2 (positions refused locally), I-3, I-4, I-5, I-6.
- **The record:** B-F4 and D-F2 (the four tests mapped or written; the trace check made to find each test in the code), D-F1 and B-F5 (the requirements amended under one A-row), D-F4, D-F5 and C-10 (citations re-pointed and resolved), D-F6, D-F7, H-6 (the READMEs), D-F9 to D-F15, H-5, H-8, H-12, H-13, C-1, C-23.
- **The code:** C-3, C-4, C-5 and L-F10, C-6 (functions past 60 lines), C-9, C-12, C-16, C-20 to C-22, C-24 to C-27, L-F8, L-F9 (one horizon), L-F11, P-1, P-2, P-5, I-8, H-9, H-11, and the batch 28 wall-clock test pinned to its own clock.
