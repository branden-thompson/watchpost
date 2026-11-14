---
title: "0.19.6 re-architecting — HUM LEAD rulings"
date: 2026-11-14
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "LIVE — every ruling is written here the moment it is made."
---

# 0.19.6 — rulings

Recorded verbatim. A ruling's disposition says what it binds; a correction to a ruling is a new row,
never an edit of an old one. Rulings of 0.19.5 are cited as "0.19.5 D-n"
(`../../0.19.5-quality-pass/02-analysis/rulings.md`); of 0.19.0 to 0.19.3 as "0.19.0 D-n"
(`../../propagation-overlays/02-analysis/rulings.md`).

The release, as the HUM LEAD set it: "We'll do the re-architecting as a 0.19.6 release immediately post
ship/debrief of 0.19.5" (0.19.5 D-40), and its principles (2026-11-14, D-1): "Performance must stay the
same or improve; Maintainability, traceability, understandability of the architecture must improve;
Maintain best practices, including P10 rules; All beahvior, and functionality must NOT regress".

| # | Date | Question | HUM LEAD verbatim | Ruling |
|---|---|---|---|---|
| D-1 | 2026-11-14 | What 0.19.6 re-architects. `modes/tty` (31,346 lines, 39,637 of tests, 88 files) and `app` (20,184 lines, 28,477 of tests, 76 files) are each over three times the next package (`platform/render`, 5,989), and every mutant re-runs its package's whole suite: in the baseline sweep on `e57afeaf` they took 57 % and 23 % of the sweep's time (A-3). Options: A, split both into smaller packages along the seams they have, behaviour byte-identical, Charm and Bubble Tea not re-architected; B, A and the end states ruled and deferred (the arbiter dissolves into the Director's step, the Director owns all ducking: 0.16.0's required reading, F-158); C, A and Charm's window composite (0.19.0 D-327, F-220). | "A. Yes - 0.19.6 should follow these conceptual principles: -> Performance must stay the same or improve -> Maintainability, traceability, understandability of the architecture must improve -> Maintain best practices, including P10 rules -> All beahvior, and functionality must NOT regress" | **0.19.6 splits `modes/tty` and `app` into smaller packages along their seams.** Four principles bind it: performance stays the same or improves; the architecture becomes more maintainable, traceable and understandable; best practice holds, P10 included; no behaviour or function regresses. |
| D-2 | 2026-11-14 | Asked with D-1: option B. | "B. This is available assuming it can confrom to the rules" | **B is in scope where it conforms to D-1's principles**: the arbiter's dissolving into the Director's step, and the Director owning all ducking, may land in 0.19.6 when each holds to them; one that cannot is left for a later release. |
| D-3 | 2026-11-14 | Asked with D-1: option C. | "C. I don't want us to be in the business of trying to re-implement charm, it will prevent us from upgrading easily when we need to bump either BubbleTea or LipGloss. I would prefer to keep this out of scope unless you can demonstrate why this is worth the effort. I would expect Orders of Magnitude improvements to justify." | **Charm is not re-implemented, and C is out of scope**: watchpost keeps to Bubble Tea's and Lip Gloss's interfaces so either can be upgraded as released. C returns only with evidence of an improvement by orders of magnitude, put to the HUM LEAD. |
| D-4 | 2026-11-14 | How 0.19.6 is classified. It moves about 51,500 lines of code and 68,000 of tests in the two packages that carry 80 % of the sweep (A-3), and D-2 may change how audio is arbitrated; 0.19.5, at about 26,000 lines, ran LEVEL-1, SEV-0. Options: A, LEVEL-1, SEV-0, HUMAN LEAD, every phase with a red team at its exit and full reports, the HUM LEAD ruling every fork and the mechanical choices as A-rows; B, SEV-1, COLLABORATE, the agent deciding forks inside a package map the HUM LEAD approves; C, the pure moves at SEV-1 and D-2's work at SEV-0. | "A. LEVEL-1, SEV-0, FULL GIT, FULL REPORTS, FULL DOCS, FULL DIAGRAMS, FULL RCC, FULL DISCOVERY, FULL TDD, FULL INSTRUMENTATION RULES, FULL P10" | **0.19.6 runs LEVEL-1, SEV-0, HUMAN LEAD, under the full directive set:** FULL GIT (branch, commit, merge and push); FULL REPORTS (a report at every phase exit); FULL DOCS; FULL DIAGRAMS (the architecture drawn at PLAN, BUILD and REFLECT, and redrawn with the code); FULL RCC and FULL DISCOVERY (FULL RCC is FULL DISCOVER's name in this theme: complete requirements and context); FULL TDD (each change behind a test first seen failing); FULL INSTRUMENTATION RULES (INST-1 to INST-5, `06_docs/quality-plan.md`, a dimension of TDD: 0.16.0's FULL INST); FULL P10 (`make p10` clean, its ledger changed only by ratification). Every fork is the HUM LEAD's; mechanical choices are A-rows for veto. |

## Agent decisions (A-n, 0.19.5 D-13)

Minor choices (references, wording, names, tool fixes) are the agent's, recorded here and batched for
the HUM LEAD's veto.

| # | Date | Decision |
|---|---|---|
| A-1 | 2026-11-14 | **Names:** the record is `06_docs/02_features/0.19.6-re-architecting/`, the work branch `feature/re-architecting`, cut from `main` at `e57afeaf` (0.19.5 and its REFLECT), squash-merged as `release/0.19.6`. For veto. |
| A-2 | 2026-11-14 | **The sweep finds its killers after a release.** `make mutant-verdicts` reads the last killers from `06_docs/mutants/verdicts/since-v<last tag>.log` (`Makefile:378`, `:436`); after a tag that file does not exist yet, and `scripts/quality/mutant-verdicts.sh:51` then starts every mutant at the whole package: about 19 hours instead of one. The fix falls back to the newest record under `06_docs/mutants/verdicts/`, pinned by a test, in 0.19.6's first code batch. For veto. |
| A-3 | 2026-11-14 | **The baseline sweep (0.19.5 REFLECT recommendation 1).** Run on `main` at `e57afeaf` before any code moves, with the 0.19.3 record as its killers (A-2): 1,058 mutants, 1,051 caught, 7 survived, which are the 7 kept equivalents (m16, m43, mAA3, mAA4, mBM1, mBM2, mSC3), 0 no evidence; 3,976 s, the baseline 138 s. Promoted to `06_docs/mutants/verdicts/since-v0.19.5.log`. By package, `modes/tty` took 8,013 s of the mutants' time and `app` 3,213 s, 57 % and 23 %. 0.19.6 ends with a sweep judged against this one. |
