---
title: "0.19.5 DISCOVER report"
date: 2026-11-06
phase: DISCOVER
sev: SEV-0
status: "For the HUM LEAD: DISCOVER exit and the A-row veto batch"
---

# 0.19.5 DISCOVER report

## Verdict

DISCOVER is ready to exit. The brief is approved (D-4), every fork the survey raised is ruled (D-5 to
D-18), and the agent's choices are A-1 to A-11, batched below for veto.

## What was done

- The HUM LEAD set the pass's five goals on 2026-11-05: no regressions, performance without
  re-architecting Charm or Bubble Tea, appropriate consolidation, every document and comment current,
  and code a junior can follow. The version is 0.19.5 (D-1); it sweeps watchpost and fixes library
  findings as library patches (D-2).
- **Five read-only lens agents** surveyed `4c803d99`, one per goal (D-3). They returned 70 findings
  plus minor lists. Their reports are filed whole in `02-analysis/lens-*.md`.
- **The agent checked** every finding that is a defect or needed a ruling, and re-ran the PF
  benchmarks (`02-analysis/discover-findings.md`, "How the findings were checked"). The rest are
  checked again where they change, pins first.

## What the survey found

- **Five defects** (`discover-findings.md`):
  - DEF-1: the Line-Up Request's backspace cuts a byte.
  - DEF-2: the console's pool ignores the units Setting.
  - DEF-3: the metric water temperature's wording.
  - DEF-4: `make mutant-verdicts` overwrites 0.16.0's record.
  - DEF-5: an unvalidated `[radio] mode`.
- **Performance:** an idle beat costs 156.5 KB on the bench fixture, a radio beat 169.7 KB and a
  console beat 310.5 KB. The five small fixes measured 104.9, 112.5 and 161.3 KB. D-4's targets,
  80, 90 and 140 KB, also need the medium memos.
- **Consolidation:** 16 findings. Six of them were one question with two answers that already
  disagreed (felt bands, spoken temperature, the voice's name, same place, the alert window, the
  compass).
- **Docs:** about 30 doc comments are bound to the wrong declaration and about 25 comments name
  symbols that do not exist. A third of `follow-ups.md` is closed rows. The as-built maps have five
  errors, and the atlas shows plan-era diagrams.
- **Readability:** ruling IDs can't be resolved, the front doors of `modes/tty` and `app` are stale,
  the vocabulary is missing, there is no "add a Setting" walkthrough, and essays sit inside function
  bodies.
- **The net:** the radio deck's voice side never runs under test, the Request window's keys are
  unpinned, and five surfaces have no whole-frame golden.

## The rulings

| # | Ruled |
|---|---|
| D-1 | 0.19.5; 0.19.4 left free for a hotfix |
| D-2 | watchpost swept; library findings become library patches |
| D-3 | five lens agents; every finding checked against the code |
| D-4 | brief approved, M2 stretched: 80 / 90 / 140 KB a beat |
| D-5 | DEF-1 and DEF-2 fixed in batch 1 |
| D-6 | metric temperatures all say "degrees Celsius" |
| D-7 | the voice's name drops its language suffix |
| D-8 | Setup uses `snapshot.PlaceID` |
| D-9 | an unknown `[radio] mode` is refused at load |
| D-10 | go-studs stays behind `platform/render`, enforced |
| D-11 | an ID index; new citations name the release |
| D-12 | all-caps headlines for contracts only, in code the pass touches |
| D-13 | the narration arbiter is renamed `arbiter` |
| D-14 | `NewDashboard` built from grouped helpers |
| D-15 | the atlas shows as-built diagrams only |
| D-16 | the Makefile's release paths are derived |
| D-17 | one `geo.Compass16` |
| D-18 | card boxes use render's rule; UAT look before SHIP |

## For veto: A-1 to A-11

In `02-analysis/rulings.md`, "Agent decisions":
- A-1: names.
- A-2: P10's rules written into code-standards.
- A-3: `docs/architecture.md` with the program's shape.
- A-4: the alert window carried as a value, its wording unchanged.
- A-5: test helpers.
- A-6: `follow-ups.md` cleaned.
- A-7: the new checks.
- A-8: the domain packages call plaintext directly.
- A-9: the HUM LEAD retakes three screenshots before SHIP.
- A-10: the timing-sensitive tests.
- A-11: how BUILD runs.

## Next: PLAN

An implementation plan (`04-development/implementation-plan.md`) orders the batches:
- stream A's pins and the two defects first (D-5);
- then performance, consolidation, docs, readability and the checks;
- each task names the findings it closes and the tests that pin it.

Following 0.9.5, a red team reviews the plan before BUILD.
