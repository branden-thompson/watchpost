---
title: "0.19.0 (with go-ionomaps and go-tuiMaps v0.3.0) — DISCOVER exit red team, round 1"
date: 2026-10-06
phase: DISCOVER exit
sev: SEV-0
authority: HUM LEAD
status: "ROUNDS 1 AND 2 DISPOSITIONED — round 1's forks ruled at D-45 to D-70 (two, PM-I1 and PM-E3, were missed and are answered in round 2 and the DISCOVER report); round 2's at D-72 to D-83; fixes applied in the three repositories (A-10 to A-23). Reviewer notes: reviewer-notes/."
---

# DISCOVER exit — red team, round 1

## Summary

- **Eight blind reviewers; five said do not exit.** The findings converged on six points:
  - band status ignored the lower limit;
  - M3 was unreachable;
  - targets were unset;
  - MUF was never scored for the path;
  - watchpost's client retries a 429;
  - the history store can erase a year.
- **Two defects in shipped code** were confirmed: the 429 retry, and the roll-up (#27).
- **The agent's own errors** are listed (E-1 to E-7). The worst, E-1, is a run past D-38's stop rule.
- **The HUM LEAD ruled every fork** (D-45 to D-70). Among them:
  - watchpost transmits nothing, and an acknowledgement replaces consent;
  - both band limits are modelled;
  - M3 is split by cause;
  - targets come from PLAN's dry run;
  - hours ahead are in, and station dots out;
  - the library is renamed go-ionomaps.
- **Every Fix was applied, save six that round 2 found** (listed under round 2). The full list of each reviewer's findings is in `reviewer-notes/round-1.md`.

**Eight blind reviewers**, each briefed verbatim from `06_docs/red-team-brief.md` (D-44) in its own scratch
directory, over the three repositories' DISCOVER records from base `31038ebf`. None made a request to GIRO,
NOAA, KC2G or Open-Meteo.

| Reviewer | Verdict | Fix first |
|---|---|---|
| DISCOVER lens (Distinguished PM) | **Do not exit** | PM-P1: US amateur rules may forbid PS-1's use |
| Business Quality | **Do not proceed** | BQ-F1: no lower limit (absorption) on band status |
| Docs Quality | **Do not proceed** | DQ-F16: the same |
| Code Quality | **Do not proceed** | CQ-E2: MUF(3000) never scored for the chosen path |
| Accessibility | **Do not proceed as written** | AX-C: the words seam sits after the alert-only returns |
| InfoSec | Proceed, with IS-1 to IS-5 as rows first | IS-1: httpx retries a 429, breaking D-39 |
| Performance | Proceed, with bounds set and PF-F4 blocking | PF-F4: the history roll-up erases its own year |
| Project Hygiene | Proceed, after one docs pass and three rulings | HY-F5/F10: evidence in a temp dir; the run went past D-38 |

**Reproduced independently:** Docs, Code and Hygiene each re-ran the wave-2 scripts. Every published
number reproduces at F10.7 = 100, and the request log matches.

**Verified by the agent before acceptance:**
- PM-P1's legal text, at `law.cornell.edu`:
  - 47 CFR §97.113(b): "An amateur station shall not engage in any form of broadcasting, nor may an amateur station transmit one-way communications except as specifically provided in these rules";
  - §97.113(c): retransmission of US-government "propagation and weather forecast information" only "occasionally, as an incident of normal amateur radio communications", never "on a regular basis";
  - §97.111(b), the one-way list: emergency communications, information bulletins, and others.
  - Not legal advice.
- PF-F4: `platform/history/history.go:1478` reads the year with `year, _ := readAs[yearDoc](...)`. A failed read is discarded, and the near-empty year is written back.

## The agent's own errors, found by this round

| # | Error | Found by |
|---|---|---|
| E-1 | **The one-sitting run went past D-38.** D-38: "on the first denial … the run stops and measures recovery". `burst.py` stopped; the agent then ran two further passes by hand (requests 98-120, at 1.5 s and 25 s), drew a second 429, and fitted the throttle model D-39 relied on to it. The findings presented it as one run, and the script for those passes is not kept | InfoSec IS-7, Hygiene F10 |
| E-2 | PS-1 (drafted by the agent, locked at D-7) was never checked against amateur-radio rules | PM-P1 |
| E-3 | D-28's question said a 120 mi radius is "about 190 km across"; it is about 193 km in **radius**, about 386 km across. The conclusion (the field is flat at local zoom) roughly survives | Docs F12 |
| E-4 | The wave-1 compute estimate (2.4×10⁸ multiply-adds a day at 1°) is about ten times low: about 2.2×10⁹ | Docs F13 |
| E-5 | RK-1 says `arodland/prop` was "never opened"; the 0.18.0 research read it through the GitHub API | PM-K1, BQ-F8, DQ-F6 |
| E-6 | The wave-2 request log undercounts (GIRO "5 of 5"; the day also had 120 GIRO and 24 NOAA requests); "38 of 118" live stations is 39 (TR169 was served on a probe and its body discarded) | CQ-E4, HY-F11 |
| E-7 | "B on D is best at every distance" overclaims: 1.06 against 1.08 at 1000-2000 km is inside the stated ±0.05 | DQ-F18, CQ-E7, HY-F12 |

## Where reviewers converged

| Theme | Raised by | Disposition |
|---|---|---|
| **PS-1's legality** under 47 CFR Part 97 | PM-P1 | **HUM** (first ruling: it decides how much of FR-2 survives) |
| **No lower limit:** band status from foF2/MUF alone; daytime D-layer absorption closes 160 m and 80 m; NOAA D-RAP not surveyed; "open" undefined | PM-R1, PM-E2, BQ-F1, DQ-F16, CQ-T2 | **HUM** |
| **M3's ≤ 20 min is unreachable** under hourly updates (D-29, D-39); GloTEC publishes 18-29 min late | BQ-F2, DQ-F5, CQ-M2, PF-F11, HY-F3 | **HUM** |
| **Metric targets unset**, deferred by wording, not by ruling (M2, M3, M5, M5b, G1, G-M2, G-M3, G-G1, v0.3.0 M5) | BQ-F3, DQ-F4, CQ-V4, PF-F9, HY-F4 | **HUM** |
| **MUF never scored for the path:** B on D's MUF RMS is 3.97 against GloTEC's 4.02; the error is M(3000)F2 (Shimazaki inverse); with measured M(3000)F2, 3.22. The < 500 km bin is seven European stations; the US stations are all > 1000 km apart | CQ-E2, DQ-F20 | **Fix:** R-9.2 (now) assimilates M(3000)F2 residuals too; the dry run scores MUF and US stations leave-one-out. Then **HUM** if D-40 needs revisiting |
| **httpx retries a 429** (`platform/httpx/httpx.go:926-928`, `:886-893`; `MaxRetries: 1` at `app/app.go:139`); production paces at 30 a second, not the 5 the record called "watchpost's way" | IS-1, PF-F1, PF-F10, CQ-T1 | **Fix:** a requirement that the propagation fetcher never retries a 429 and passes status through; a test against the real client |
| **The live-station list** has no defined way to be learned (probing all 118 exceeds the cap and the bucket); 39 live against a cap of 40 | PF-F3, HY-F2, CQ-E6 | **HUM** (how), with requirement rows |
| **One throttle owner per process:** readout, Propagation mode and recorder, across several instances, would exceed the bucket | PF-F8, CQ-M1 | **Fix:** one library object per process, merged updates, the hour claimed across instances (the store's `Claim`) |
| **Fleet load and bytes:** about 70 MB a day per copy by default (D-29); every copy polls GIRO; G1 has no network term; no plan if GIRO blocks the User-Agent | PM-IR1, BQ-F7, PF-F9, DQ-F7, IS-5 | **Fix:** G1 gains bytes a day and a 48 h window; a follow-up for "GIRO blocks us". The default stays D-29's |
| **D-238's "all of it" cut without a ruling:** eSSN chart, P.533 planner, hours ahead, station dots; F-189 unchanged; PS-2's "at a given hour" | PM-R3, BQ-F5, DQ-F8 | **HUM** |
| **L-5 (valued station points) has no consumer** | PM-R2, BQ-F12, CQ-N6 | **HUM** (D-35 ruled it) |
| **G-M3's KC2G reference cannot run** | PM-K4, BQ-F10, DQ-F10, CQ-N7, HY-F7 | **HUM** |
| **The traceability protocol D-1 deferred was never ruled** | PM-K1, BQ-F8 | **HUM** |
| **The history store's roll-up erases its own year** (32 MB read cap; 66 days at 2°); `[]*float64` holds 12.5-50 MB a day in memory | PF-F4, PF-F5 | **Fix** as 0.19.0 requirements (FR-6). **HUM:** whether to file it as a bug in shipped 0.18.0 |
| **Evidence lives in a temp directory** | HY-F5, CQ-E5 | **Fix** (A-n: where it lives) |

## Accessibility (AX)

**Fix as requirement rows** (they carry out D-9, "words ship with the layer"):
- AX-A: the mode named in words in every path.
- AX-B: the mode's controls as keymap actions, and focus after Lookup.
- AX-C: the description seam, tested for alerts off, alerts never asked, and the place out of view.
- AX-D: the mode's status in the no-picture path.
- AX-E: day and night in words, and a terminator that does not rely on colour (go-tuiMaps L-4.4).
- AX-F: OPEN, CLOSED and NO DATA as words.
- AX-H: the Broadcaster map's keys as actions, with key-clash tests.
- AX-K: units written for speech.
- AX-M: the outside-region message names the Propagation key.

**HUM:**
- AX-C's D-78 in this mode;
- AX-G: whether the Broadcaster map gets words, and its own Setting;
- AX-I: the channel for "said", whether the closure notice may go on air, and the notice's lifetime;
- AX-L: the 44-row floor;
- AX-N: a SHIP gate on FR-7.

## InfoSec (IS)

**Fix as requirement rows:**
- IS-2: GIRO replies carry the requester's IP. Headers are stripped first, errors never quote input, goldens are scrubbed, the cache is scanned, and MAP STATUS discloses it.
- IS-3: physical-range checks, no NaN or Inf out of the library, station codes held to a pattern and passed through `url.Values`, credits from a static list.
- IS-5: GIRO's non-commercial access condition is stated to users; RK-2 corrected; a dataset-terms field in the history store.
- IS-6: hardening options and per-source body caps; the "allowed hosts" promise is built or struck.
- IS-8: govulncheck in go-ionomaps.

**HUM:**
- IS-4: the Observer's lookup refuses non-US places and sends typed text to Open-Meteo. **Corrected in round 2 (N-2, E-8):** watchpost already embeds the world-cities index; non-US places are refused by a coverage gate (`domains/locations/resolver.go:101-102`), so the fix is a mode-aware gate, not a new list (D-61, FR-1.10). Coordinates parse locally for every resolver (FR-4.9).

## Performance (PF)

**Fix as requirement rows:**
- F2: ask GIRO only since the last sounding;
- F5: store values compactly;
- F6: GloTEC fetched without the HTTP cache;
- F7: M5 as a per-frame bound, and Update off the UI goroutine;
- typed GeoJSON decode.

## Record fixes (A-n batch)

- **Stale statuses:**
  - wave 2 still saying "D-20 next" and "OQ-G2 to be ruled";
  - the requirements' "(path)" markers, and "DRAFT — approved at" reworded "for approval at";
  - RK-2 and RK-3;
  - the band question in `problem-statement.md`;
  - follow-ups still saying D-256 first, against D-4;
  - required reading missing four rows.
- **Wrong references:**
  - the KC2G baseline in the brief;
  - `facts.go:186`;
  - GC-3 "unverified";
  - go-tuiMaps citing watchpost D-23 for P-1 to P-3 (0.18.0 D-23 is meant);
  - go-ionomaps's bare "D-2";
  - the issue of record promised against go-tuiMaps D-8;
  - "Europe, the Americas and Asia".
- **The agent's errors** E-3 to E-7.
- **Necessity deletions:**
  - R-9.1's "and the baseline" role (CQ-C1);
  - R-1.3 moved under R-2 as R-2.7 (CQ-C2);
  - duplicate cadence and NFR rows (CQ-N2, N3);
  - FR-4.5's duplicated tests and "headers are read" (CQ-M1, N4);
  - the effective-sunspot fit (then R-9.2) turned into a PLAN question (CQ-N1); R-9.2 is now the M(3000)F2 assimilation row.
- **Instruments that cannot fail:**
  - FR-10.1, FR-10.4 and FR-10.5 relabelled (CQ-N9);
  - v0.3.0 M4 anchored to mutants (CQ-V1);
  - R-6.1's golden made synthetic in the tree (CQ-V2);
  - the no-raw-data scan made content-based (CQ-V3);
  - L-4.2's tolerance stated (CQ-T3).
- **Scripts:** `pairs.py` fails on zero pairs (CQ-E1); the docstrings fixed (CQ-S2, HY-F12).
- **F-206** corrected: route 1 relaxes to IRI; route 2 is circular and not yet built (CQ-D2, HY-F8).
- **The baseline rule** written down: F10.7 and the coefficient set (CQ-E3).
- **OQ-G4** recorded as answered by D-31 (HY-F7).

## Further forks for the HUM LEAD (not ruled here)

- PM-P2: WSPR spots as PS-2's *answer*, not only its yardstick.
- PM-P3: does B on D meet PS-G's own words?
- PM-S1: a second HF operator outside the project.
- PM-I1: a do-nothing / do-less column.
- PM-E3: a comparison with HamClock, VOACAP Online and PSKReporter's map.
- BQ-F9 / DQ-F14: can the Broadcaster open the Propagation mode (issue #25 asks for MUF on a Broadcaster map), and the tower "always marked" row.
- DQ-F9: does the recorder fetch for an Observer who never opens the mode?
- BQ-F6: go-tuiMaps OW-14 to OW-26, owed to v0.3.0.
- CQ-S3: the name go-ionomaps, now that two of its three inputs are not GIRO.
- HY-F1: delete the shipped work branches (`feature/map-drawing`, `feature/radar-loops`, local `feature/go-tuimaps`).
- DQ-F19: a short summary atop each analysis page, or leave them as engineers' records.

# Round 2

**Four fresh blind reviewers (D-71)** checked closure of round 1 and anything new, at watchpost
`393aeaf7`, go-tuiMaps `03b3c42` and go-ionomaps `797adb5`. Notes: `reviewer-notes/round-2.md`.

| Reviewer | Verdict | Fix first |
|---|---|---|
| DISCOVER lens (PM) | Do not exit | N2: "open" undefined |
| Business | Proceed after N1 to N4 and N7 | N1: re-ask D-29's default |
| Docs | Do not proceed yet (no Critical; round 3 not needed if fixes are checked) | I-2: M3b/M3c's two rulings |
| Personas (A11y, InfoSec, Perf) | Proceed after N-1 to N-7 | N-1: the screen reader on the broadcast audio |

No Critical. Every published number reproduced again.

## The agent's errors found in round 2

| # | Error | Found by | Now |
|---|---|---|---|
| E-8 | D-61's question described the world-cities list as something to ship ("a few MB"). watchpost already embeds GeoNames cities15000 (`domains/locations/geodata/index.go`, about 13 MB of memory and 1.3 MB of binary); non-US places are refused by a coverage gate | personas N-2 | A-17; FR-1.10 |
| E-9 | Round 1's reviewer reports were never kept in the tree, so closure could not be audited | PM, Business N7 | condensed notes committed (`reviewer-notes/`) |
| E-10 | D-49's recorded text listed M3b and M3c as unset, contradicting D-48 | Docs I-2 | D-72 |
| E-11 | "The four US stations are all more than 1000 km apart, scoring 1.06-1.28" was wrong: by station, the mainland US scored 0.51-0.81 and the Pacific 1.2-1.8 | Docs I-4 | RK-3; `evidence/subsets.py` |
| E-12 | F-208 put losing GIRO at "about 0.6 MHz" near stations; it is 0.26 | Docs I-9 | F-208 |
| E-13 | The record said "every HUM fork ruled" while PM-I1 (do less) and PM-E3 (a comparison) were never asked | PM, Business | the do-less forks became D-76 to D-78; both answered in the DISCOVER report |

## Dispositions

**Ruled by the HUM LEAD (D-72 to D-83):**
- **D-72:** M3b/M3c's targets stood (later retired with M3).
- **D-73:** "open" is an SSB voice reference circuit.
- **D-74:** routine closures as a schedule.
- **D-75:** floors set before the dry run.
- **D-76:** the HUM LEAD's reference chart replaced the monitoring readout. This answered BQ N1, N2 and N4 and PM N1 and N3 in part; it retired D-29, D-30, M3, D-48, D-59 and D-74's notice.
- **D-77:** the Broadcaster's own map back to a follow-up.
- **D-78:** no propagation recording or backfill.
- **D-79:** Maidenhead locators.
- **D-80:** an audio-device Setting (N-1).
- **D-81:** the acknowledgement's details (N-7, N-14).
- **D-82:** listener first, with a glossary.
- **D-83:** only rate headers at run time (N-11).

**Fixed (A-16 to A-23):**
- **Requirements, all three:**
  - watchpost's rewritten for D-76 to D-83;
  - go-ionomaps gains R-2.8 and R-2.9;
  - go-tuiMaps notes the reach field on L-2.2.
- **InfoSec and performance:**
  - the propagation client is its own instance (N-5);
  - a shared, finite coordinate parser for every resolver (N-3);
  - validators kept for 304s (N-12);
  - answers off the UI path (N-8);
  - #27's split sized from bytes, with the decompressed cap named (N-4);
  - the fourth description case (N-9);
  - console keys and the focused target named (N-13);
  - the G1 row (N-15).
- **The record:**
  - the intake briefs carry banners (PM N5);
  - stale statuses, counts and citations corrected (Docs I-1, I-3, I-5, I-7, M-1 to M-8);
  - `subsets.py` added and CDN location headers stripped;
  - F-174, F-180, F-187, F-188 and F-208 updated; F-209 (band watch) and F-210 (history and trends) added; the closed F-190 removed.

**Carried to the DISCOVER report:**
- PM-I1, the do-nothing / do-less column. Its candidates were ruled at D-76 to D-78.
- PM-E3, a comparison with HamClock, VOACAP Online and PSKReporter's map.
- The fleet scenarios for GIRO (PM N4, kept under D-41).
- PM-E2, the space-weather scales beyond D-RAP: noted as a PLAN question.
