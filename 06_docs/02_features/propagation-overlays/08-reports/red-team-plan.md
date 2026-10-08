---
title: "0.19.0 (with go-ionomaps and go-tuiMaps v0.3.0) — PLAN exit red team, round 1"
date: 2026-10-07
phase: PLAN exit
sev: SEV-0
authority: HUM LEAD
status: "ROUNDS 1 AND 2 DISPOSITIONED — round 1: eight reviewers, two Criticals, forks D-109 to D-127; round 2: four reviewers, no Critical, both round-1 Criticals closed, forks D-129 to D-138; both fix batches applied; A-25 to A-30 batched for veto"
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
- **What reproduced:** every number published in `dry-run.md`; Docs, the PLAN lens and Hygiene each re-ran `week.py`. Every code citation checked holds. **But** the figures the agent put to the HUM LEAD in forks 1 and 8 came from an intermediate run and did not reproduce (E-19, found in round 2, corrected at D-129 and D-130).

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
| Fork 2: the terminal cursor; how M5b is sat | A-1 | **D-110:** left in O2 (F-205, AX-3); VALIDATE states M5b measured a sighted reader of the words and that a screen reader cannot yet follow the mode's focus; the protocol says so |
| Fork 3: the download figures | B-F8, P-2, C-M4, C-T3 | **D-111:** wire bytes, decimal MB; BUILD's first task (W0.0) measures a gzip grid and the index and re-rules D-95's downloads with 25% headroom, newest-grid discovery costed |
| Fork 4: memory | P-3 | **D-112:** the library's live heap ≤ 15 MB (amends D-98); G1's 25 MB added RSS split library 15 / host conversion 4 / GC headroom; only the shown hour handed to go-tuiMaps |
| Fork 5: host allowlist | I-2 (A-12) | **D-113:** go-ionomaps exports its hosts with their terms; the propagation client refuses any other; MAP STATUS and a test read the one set; A-12 withdrawn |
| Fork 6: reading the netCDF tables | I-6 | **D-114:** a one-time plain-text export from the pinned PyIRI environment with source checksums; `tools/tables` standard library only |
| Fork 7: privacy in the UAT record | I-5 | **D-115:** places as a region or 4-character locator, no callsigns; raw logs outside the tree |
| Fork 8: the Pacific | L-F2, D-F15 | **D-116:** every answer says its distance to the nearest reporting station; beyond 2000 km "about as good as typical"; G-M3 reports the Pacific and distance bands, not gated |
| Fork 9: go-tuiMaps M1 and M6 | B-F6, D-F3, L-F14 | **D-117:** M1 10 places, 8/10 picture and 8/10 words, HUM-graded, P8 task; M6 5 consecutive clean full runs |
| Fork 10: a missed target | B-F5, L-F10 | **D-118:** floors cut; any other miss is a ruling at VALIDATE with evidence, never re-set silently; M1/M4 baseline measured first, >90% means "no worse than climatology" |
| Fork 11: timing gates | C-10, P-7 | **D-119:** gates assert proxies; wall-clock targets a release step on the reference machine (fail when missing); a linux/amd64 run recorded |
| Fork 12: the refresh Setting and its key | B-F9, A-2 | **D-120:** A-26 ratified; a refresh-now keymap action in FR-1.14, honoured under D-39; retry names the key |
| Fork 13: the Broadcaster's words | A-3 | **D-121:** "Instead of the map", refresh and the no-readings correction shared across both modes; the radius Observer-only; FR-2.3, W7.2 |
| Fork 14: when O2 ships | L-F10 | **D-122:** O2 becomes 0.19.1, the next release; FR-7, W8 and D-63's gate move with it; 0.19.0 is O1 and O3 |
| Fork 15: the oracle's tolerance | L-F5 | **D-123:** D-107 stands (set in BUILD from the measured agreement); the agreement is reported in MHz of MUF beside the tolerance chosen |
| Fork 16: the evidence's home | H-5.1 | **D-124:** instruments committed to go-ionomaps `evidence/plan-dry-run/` (paths removed, placeholder headers); data and extracts stay local |
| Fork 17: the 1° option | P-10 | **D-125:** 2° only in v0.1.0; a typed choice, so 1° can be added later |
| Fork 18: the dry run's read order | D-F14 | **D-126:** a plain verdict first, then the table, then the detail |
| Fork 19: words and the reach mark | A-5, A-6, A-8, A-10 | **D-127:** all four: state said, one line per target, a refresh keeps the place, go-tuiMaps L-2.6 (also closes C-T1's L-2.5 gap via P2.4) |
| **The fix batch** | every finding routed "fix" | **Applied across the three records.** The details:<br>- **Trace:** every test the requirements name is in a plan task (watchpost 95, go-ionomaps 74, go-tuiMaps 30 in scope; L-5's two are out of v0.3.0), and W0.4 makes it a docs-reading check.<br>- **New tasks:** M1/M4's instrument (W10.3) and the UAT prompter (W10.4); W2.0's map-mode type (C-M2); W4.6 (D-116); R-3.5 and G7.2 (caller bounds, no NaN, no fetch in answers); R-9.8 and G10.5 (the constants' basis and expiry); P7.0 (whole-globe contrast before optimising).<br>- **The shape completed** (A-29) and the update separated from the answers, with the memo key and immutability.<br>- **InfoSec rows:** I-1 (import ban), I-4, I-7, I-8, I-10.<br>- **Corrected:** FR-3's preamble; RK-3, RK-11, RK-12 and a new RK-15; the floors text; FR-1.16's range; FR-1.17's caveat.<br>- **Brought current:** REQUIRED-READING and `plan-questions.md`; the dry run's verdict first (D-126), with "every day", the kernel, the window and the command corrected.<br>- **Evidence and record:** NOTICE (IGRF-14, the Apex sample, hmF2 removed); the README; R-5.6, R-7.3, R-9.1's role; NFR-1; `loo_b_on_d.py` (renamed, docstring fixed); `dry_fetch.py` stops GIRO after the full back-off; `day.sh` deleted; G10.4's oracle is `week.py`; `week.py`'s station message computed; F-208 refreshed; F-212 (storm modelling) and F-213 (the idle redraw) added; W1.1 teaches P10 to see a discarded `ok`; `parseLatLon` deleted in W0.3; W8.1's rows name their tests before 0.19.1.<br>- **Sizes:** estimated, about 44 batches for the train. |
| Not fixed, with the reason | C-C2 (`offset.py` re-runs `week.py`, 39 s); H-3.2 and C-E9 (`plancode` checks only watchpost) | C-C2: a one-off evidence script, and refactoring it risks the published numbers. H-3.2 and C-E9: go-ionomaps' gate arrives at G0 with a plan-code check; until then the gap is stated in the PLAN report |

# Round 2 (D-128)

**Four fresh blind reviewers.** No Critical; both of round 1's Criticals were confirmed closed, and the test
trace was re-run mechanically by three of them (watchpost 95 of 95, go-ionomaps 74 of 74, go-tuiMaps 30 of 32
with L-5's two out of scope). Notes: `reviewer-notes/plan-round-2.md`.

| Reviewer | Verdict | Fix first |
|---|---|---|
| Combined personas (Accessibility, InfoSec, Performance) | Do not exit yet; no Critical | N1: A-29's shape broke D-112's memory cap |
| PLAN lens (Principal Architect) | Do not exit yet; no Critical | LN1: the update's seam, deadline and cancelled-burst budget |
| Code Quality | Do not exit yet; no Critical | CN1: D-109's figures do not reproduce |
| Docs Quality | Do not exit yet; no Critical | DN-1: D-109, D-116 and RK-3's figures do not reproduce, and the Pacific result flips |

## The agent's errors found in round 2

| # | Error | Found by | Correction |
|---|---|---|---|
| E-19 | **Fork 1's band-answer figures and fork 8's Pacific and distance figures came from an intermediate run whose climatology was raw CCIR**, which D-43 forbids shipping. With the shipped climatology the Pacific hybrid is worse than the climatology (5.17 against 5.09), not better; the ledger's "every published figure reproduces" was false for these | CN1, DN-1 | D-129 (D-116 re-put; stands with honest words) and D-130 (D-109's figures corrected; conclusion unchanged); RK-3 and `dry-run.md` corrected; sections 10 and 11 published |

## Round 2 dispositions

| Fork / fix | Findings | Disposition |
|---|---|---|
| D-116's corrected premise | DN-1 (E-19) | **D-129:** D-116 stands; beyond 2000 km the words say "about as good as typical here, or worse"; fresh BUILD days re-check |
| D-109's figures | CN1, DN-1 (E-19) | **D-130:** corrected (76 against 56; 55 against 53); conclusion unchanged |
| Memory | persona N1, N2 | **D-131:** hours ahead share the climatology cache, one nearest-station table, no-data as a bitset (live about 7 MB, estimated); G1 judged on measured RSS |
| Grading on the same week | LN2 | **D-132:** G10.4 scores at least 3 fresh days fetched in BUILD |
| A screen reader in 0.19.0 | persona N6 | **D-133:** Help and the words say the focus limit (FR-3.7); RK-6 restated |
| The typical error | LN6, DN-7, CN2 | **D-134:** by distance band from the test days (0.77 \| 2.91 to 1.36 \| 4.58 MHz); `week.py` section 11; G10.5 asserts |
| The day's open hours | DN-6, LN7 | **D-135:** the climatology for every hour, labelled "typical day", beside the current hour; a disagreement said |
| The words' order | persona N8 | **D-136:** a status block said once per open; one line per target; the order in FR-3.1 |
| go-tuiMaps M5's case | DN-10 | **D-137:** the target includes the reach overlay |
| `Options.Grid` | Code (a recommendation) | **D-138:** deleted |
| **The fix batch** | every other round-2 finding | **Applied.** The details:<br>- **The architecture:** two seams (`PropagationUpdate`, `PropagationAnswers`) and the redrawn "Opening the mode" diagram; a 60 s deadline; a cancelled burst spends the hour; NOAA first, so a first picture comes before GIRO; the memo key on every input; the no-readings correction per update.<br>- **The shape:** the inputs' own times and D-RAP's state; "measured" defined as within 500 km; `NotChanged`, `Typical`, merging, the hmF2 check and `internal/forecast` removed (A-30).<br>- **Bounds and expiry:** `Retry-After` bounded and fuzzed; at most 64 places a call; R-9.8's expiry a release step.<br>- **Corrected text:** FR-5.1's age is the oldest input's; NFR-1 restated (D-111, D-112); the GloTEC 304 text corrected and the test renamed `TestAnUnchangedFeedCostsA304`; the SILSO row deleted; the go-ionomaps R-9 heading and floors current; `requirements.md`'s "forecasts" line, the dry run's stale lines, `plan-questions.md`, the UAT protocol's FR-1.9, RK-12's "one day"; "first in BUILD" made unambiguous (W0.0 then W1).<br>- **The O2 follow-ups:** rows at 0.19.1 (D-122), F-206 corrected, F-210 keeps the set-aside blend.<br>- **Evidence:** `week.py` checks the storm days and documents its sections; `offset.py`'s usage; `dry_fetch.py`'s back-off stated exactly and NOAA stops after giving up.<br>- **Tasks:** go-tuiMaps P7.2 (M6 and the release record); the release records' contents named; the size recalibrated to about 70 batches (RK-5); the trace complete again. |
| Not changed, with the reason | LN9 (the two project briefs still say A-12 struck the allowlist) | The briefs are approved records kept as written, with a banner that `requirements.md` is current and wins (A-23) |

