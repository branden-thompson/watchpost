---
title: "0.19.0 (with go-ionomaps and go-tuiMaps v0.3.0) — PLAN exit red team, round 1"
date: 2026-10-07
phase: PLAN exit
sev: SEV-0
authority: HUM LEAD
status: "ROUND 1 IN DISPOSITION — all eight reports in, all 'do not exit'; forks going to the HUM LEAD one at a time; fixes batched"
---

# PLAN exit — red team, round 1

## Summary

- **Eight blind reviewers (D-90). All eight say do not exit PLAN.**
- **Six reviewers independently found the same Critical problem:** the forecast ruled at D-103 cannot run.
  - Each forecast hour is half "the field at the same hour a day before".
  - Under D-39 (NOAA's newest grid only, 6 an hour), D-76 (nothing fetched until the mode opens) and D-78 (nothing recorded), no such field exists on any open, unless the process stayed open for the previous 24 hours.
  - The dry run scored it with the whole week on disk.
  - A catch-up would cost about 57 MB and about 4 hours at NOAA's cap, and about 0.9 s of CPU against D-98's 0.5 s.
- **A second Critical (Accessibility):** nothing places the terminal cursor on the new mode's focus, so a screen-reader user cannot follow it. M5b, graded by a sighted reader of text, would not catch it.
- **The record's other gaps:**
  - M1 and M4's instrument is planned nowhere.
  - About a third of the named tests are dropped between requirements and plans.
  - The public shape lacks fields the requirements promise.
  - The download and memory targets (D-95, D-98) don't add up.
  - The evidence behind about ten rulings lived only in a temp directory. It is now preserved outside the tree (below).
- **What reproduced:** every published number. Docs, the PLAN lens and Hygiene each re-ran `week.py`. Every code citation checked holds.

Reviewer notes: `reviewer-notes/plan-round-1.md`.

| Reviewer | Verdict | Fix first |
|---|---|---|
| Business Quality | Do not exit | B-F1: the forecast cannot run under D-39, D-76, D-78 |
| Accessibility | Do not exit until A-1 | A-1: the terminal cursor never follows the new mode's focus |
| InfoSec | Do not exit yet | I-1: the fetcher contract's `http.Header` breaks "opens no connection of its own"; I-2 no host allowlist |
| Docs Quality | Do not exit | D-F4: the forecast (as B-F1) |
| PLAN lens (Principal Architect) | Do not exit | L-F8: the forecast (as B-F1) |
| Performance | Do not exit | P-1: the forecast's inputs; re-rule D-95 and D-98 with P-2, P-3 |
| Project Hygiene | Do not exit | H-5.1: evidence only in a temp directory |
| Code Quality | Do not exit | C-M1: the forecast's cold start (as B-F1); also the `Snapshot` contract against R-3.3, and verifiers that would pass without verifying (C-E1, C-E3, C-E7, C-E8, the timing gates) |

## Where reviewers converged

| Finding | Reviewers | Route |
|---|---|---|
| **The forecast's "yesterday" has no source** (D-103) | B-F1, D-F4, L-F8, P-1, C-M1 (and B-F2, B-F3, D-F16, L-F4: its error figure has no stated source) | **HUM** (fork 1) |
| The terminal cursor; M5b by a sighted reader | A-1 | **HUM** |
| M1 and M4's instrument planned nowhere; no consequence on a miss | L-F1, H-4.1, B-F5, C-T2 | fix (task) + **HUM** (consequence) |
| Named tests dropped between requirements and plans | D-F8, L-F12, H-4.2, C-T1 | fix, with a docs-reading trace test |
| The public shape lacks promised fields; `Limit` a free string; `http.Header` in the contract; `Field.Values` NaN against R-3.3; removable fields | D-F7, L-F7, A-9, I-1, C-M3, C-D, C-N | fix (shape) |
| Download target: uncompressed bytes, no 304 for timestamped grids, the index's cost; MB undefined | B-F8, P-2, C-M4, C-T3 | **HUM** (re-rule D-95) |
| Memory budgets don't add up; "peak" undefined | P-3 | **HUM** (re-rule D-98, D-95's RSS) |
| FR-3's preamble promises the device Setting D-85 withdrew | B-F4, A-4, D-F5, H-4.3, L-F15 | fix |
| go-tuiMaps M1 and M6 have no target | B-F6, D-F3, L-F14 | **HUM** |
| NOTICE lacks IGRF-14 and the Apex sample | B-F11, D-F10, L-F13, H-4.5 | fix |
| G10.4 points at DISCOVER's `loo_hybrid.py`; inputs have no durable home | D-F9, L-F11, H-4.4, C-E7, C-U4 | fix |
| The update runs inside the per-ask command keypresses cancel; answers wait behind a 36 s burst | L-F9, P-4 | fix (architecture), bound to **HUM** |
| The Pacific: the hybrid's MUF worse than climatology | L-F2, D-F15 | **HUM** |
| A-26's Setting is UI (D-13 excludes it); "on demand" has no key | B-F9, A-2 | **HUM** |
| The scorer: common pairs not enforced where D-75 was read (C-E3); the F10.7 mean averaged 23-29 days (C-E2); the climatology chosen with test days in (C-E4); the disturbed-days argument optional (C-E1, D-F12, H-2.1) | C-E1 to C-E5, D-F12, H-2.1 | **fixed and re-scored before fork 1** |
| A third map mode on the `radarMode()` boolean (28 call sites, 15 negations) | C-M2 | fix (W2.0) |
| Timing targets gated on CI runners would flake or need machine checks (evasion lens) | C-10 | **HUM** (gate on proxies; reference-machine timing a release step) |
| W8.2's SHIP gate has nothing to check (FR-7 rows name no tests) | C-E8 | fix |
| `dry_fetch.py` keeps asking GIRO after giving up on a 429 | C-E6 | fix |
| Stale statements (RK-11, R-5.6, README, headers, plan-questions, "every day", kernel, 23-29-day mean) | D-F11, L-F15, H-4.6, H-5.2 to H-5.7, D-F6, D-F17 | fix, one batch |

The full list, with each finding's evidence, is in `reviewer-notes/plan-round-1.md`. Each disposition is recorded below as it is made.

## The agent's errors found in round 1

| # | Error | Found by | Correction |
|---|---|---|---|
| E-14 | Commit `90caa8fa` went out after a failed docs lane. The atlas was staged, `git checkout --` restored the staged copy, and the commit was chained with `;` not `&&` (recorded at A-24; missing from this table until now) | H-5.6 | `make verify` green on `90caa8fa`; commits chained to the verdict since |
| E-15 | **D-103 was put to the HUM LEAD without saying the blend needs a field from the day before, which no ruled data path supplies.** The dry run scored it with the whole week on disk | B-F1, D-F4, L-F8, P-1 | fork 1 re-puts the forecast with what a host can run, scored |
| E-16 | The forecast's "measured typical error" was ruled with no stated source at run time | B-F3, D-F16 | fork 1 |
| E-17 | The dry run said the B on D floor is met "every day"; 10-03 in the US misses on foF2 (0.93 against 0.84) | D-F6, H-5.2 | reworded in the fix batch |
| E-18 | The PLAN red team summary to the HUM LEAD said all eight reviewers had reported when seven had | the agent | corrected in the next message |

## Evidence preserved (H-5.1)

The dry run's data and instruments are copied out of the session's temp directory to a local folder outside every repository. It holds:
- the week's GIRO and GloTEC data, request logs, F10.7 and geomagnetic indices;
- the spikes and the go-tuiMaps frame benchmark;
- the G1 harness;
- the SNR extracts and the WSPR queries;
- the reviewers' prompts and reports;
- a `pip freeze` of the PyIRI 0.1.7 environment.

Nothing from it is committed: GIRO's data is CC BY-NC-SA, and its replies carry the requester's IP. Whether the spikes, harness and queries are committed to go-ionomaps `evidence/` is a HUM LEAD question (some carry absolute local paths).

## Dispositions

| Fork / fix | Findings | Disposition |
|---|---|---|
| Fork 1: what an hour ahead is | B-F1, B-F2, B-F3, D-F4, D-F16, L-F8, P-1, C-M1 | **D-109:** the climatology for that hour, labelled typical, with its error measured on PLAN's week and that basis stated. Nothing new is fetched; D-98's cold open already counted the 25 hours. The floor holds by construction, which also closes B-F2's re-check concern |
| The scorer's soundness | C-E1 to C-E5, D-F12, H-2.1 | **fixed** (`week.py`): the storm days a required argument; every method scored on the same pairs, with missing pairs an error (all 1,476 were common, so no number moved); the shipped climatology pinned (CCIR refit, 30-day F10.7) rather than chosen on the data; the F10.7 window stated (23 to 29 days) and refused below 20; tuning on nothing fails. Every published figure reproduces |
