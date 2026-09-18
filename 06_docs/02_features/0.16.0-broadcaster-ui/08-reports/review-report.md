---
title: "0.16.0 — Broadcaster UI — REVIEW REPORT"
date: 2026-09-18
phase: REVIEW
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
directives: FULL GIT; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD; FULL INST
status: "REVIEW APPROVED by the HUM LEAD 2026-09-18 (\"REVIEW APPROVED; GO 4 VALIDATE\"); exit commit 35de36a"
---

# 0.16.0 — Broadcaster UI — REVIEW REPORT

## Bottom Line Up Front

**REVIEW found that the release could not be released, that its completeness claim was wrong by
method, and that its one safety-class fault path could not fire on a live station — and closed all
three.** Eight blind reviewers on the four axes returned *do not ship* against the BUILD-exit commit
`557a40f`. Where they converged measured the repository: the release path (`p10` in `verify` broke
the tag on a runner with no harness); the traceability method (FR IDs collide across releases, so
"60/60, none untraced" was wrong by construction); four verifiers that failed open; a release-facing
record that contradicted itself on every headline number; and F-150's fault class, which re-entered
the schedule at pump speed and escalated only when the schedule was empty — never, on a station the
Producer keeps topped off.

Twelve asks went to the HUM LEAD with the evidence and a recommendation each; every ruling was
executed as seven remediations, R1–R7, and every safety-class fix went to a fresh blind reviewer and
was iterated to LGTM — the F-150 fix twice, the privacy fix twice. The second round of each found what
the first had missed: a fault band with two owners, and a redaction that covered one of seven lines.

**Exit at `35de36a`: `make verify` ALL GATES GREEN; sweep 380 mutants — 376 caught, 4 survived by
design, 0 no evidence; P10 0 live / 0 unratified; requirements 31 traced by ID, 13 by a named test,
4 OPEN with rows. REVIEW APPROVED by the HUM LEAD, 2026-09-18.**

## 1. Gate results

| Gate | Result | Evidence |
|---|---|---|
| `make verify` | **ALL GATES GREEN** on `73bce2b` (code) and `35de36a` (records) | `dist/verify-review-73bce2b.log`, `dist/verify-review-35de36a.log` — local artefacts (`dist/` is git-ignored), read from the log, not the exit code; the roster's sweep summary in `gates.md` is the tracked record |
| `make quality` (P10, phase-exit) | 0 live, 0 unmatched, 0 unratified; 153 ratified rows | `dist/p10.json`, `06_docs/p10-ledger.md` |
| Mutation sweep | **380 — 376 CAUGHT, 4 SURVIVED (by design), 0 NO EVIDENCE** on `73bce2b`, 4 h 48 min; `mBM2` read as caught there only against `./...` and survived again at VALIDATE exit (by design, all five) | `07-readiness/mutant-verdicts.log`, opening with the commit hash |
| Blind spot beside the number | a detector that needs `-race` reads as SURVIVED | `gates.md`, "The sweep's own blind spot" |
| Exposure | re-derived at the tip; identity 137 files / 341, location 214 / 750, credential 2 distinct fixtures, path 4 / 5 placeholders | `07-readiness/exposure-statement.md` |

## 2. Requirements verification

Derived by roster test NAME scoped to this release (`build-report.md`, "Requirements traceability",
ruling 9): **31 traced by ID, 13 by a named test without the ID, 4 closed in the red team's own rows,
4 OPEN with follow-up rows (FR-3.6 F-161, FR-8.3 F-157, FR-8.8 F-162, FR-9.2 F-160), the rest
PARTIAL / superseded / withdrawn as the rows say.** The five collisions two reviewers sampled — FR-2.2,
FR-4.3, FR-6.4, FR-9.2, FR-9.3 — are adjudicated by name in the table. FR-8.4 gained the three-mile
fence test its exit criterion asked for. OPEN requirements carry as recorded scope (ruling 9).

## 3. What the red team found, and what was done

| Convergence | Axes | Disposition |
|---|---|---|
| **The release path** — `p10` in `verify-gates`; `release.yml` runs `verify`; no `a2dh` on the runner | Hygiene, Code Quality | **R1** (`2bf599d`): `p10` is a phase-exit gate under `make quality`; declared in the three-list model |
| **The traceability method** — FR IDs collide across releases | Business, Docs | **R7** (`6ff8802`): derived by roster name scoped to this release; collisions adjudicated; three OPEN requirements got rows |
| **Verifiers that fail open** — `lint-identity -run`, `mutant-anchors.sh`'s missing count, the specimens' skip-all and COULD-NOT-RUN-as-caught, `wires -json`, `scripts/lint.sh`'s discarded exit code (F-118) | Code Quality, Hygiene | **R6** (`3c67d88`): each fails closed, each planted |
| **F-150's fault class** — cool-off skipped non-routed faults; escalation fired only on an empty schedule | Business, Code Quality | **R2** (`b51cdd6`…`9e318ac`): cool-off for every failure, a run limit of three, the station's own band; two blind rounds to LGTM (see §4) |
| **The record contradicts itself** — gate counts, three sweep figures, P10 numbers, the handoff, the exposure statement's own example | Hygiene, Docs | **R7** (`620bce5`, `0963b36`, `35de36a`): one figure per fact, each named with its commit |
| Product forks — "Peña" → "Pea"; ON AIR behind a window; the listener's mute on air; one place at three miles | Code Quality, Business | **R3** (`dfadc12`), **R4** (`34bf80b`): fixed as ruled; FR-8.6 amended; F-157 for tier three |
| Instruct vs P3(d) — two comments said `startSynth` retires | Code Quality | **R5** (`5355e3f`): Instruct is current (ruling 4); F-158 records the ruling owed |
| The README's privacy sentence vs the opt-in radio log | Docs (Critical) | **Ruling 10** (`9944b5c`, `000d61a`, `73bce2b`): every coordinate pair rewritten at the one file writer; the gate reads the file; the console band names places by label |
| The console's key table; a `Run: 0` escalation dropped; does a decline break a run | Docs, R2 reviewer | **Rulings 12, F-163** (`9944b5c`): the table; the reason-only band; a decline does not break a run |

The reports are reproduced verbatim in `red-team-review.md`.

## 4. The remediation loop, counted

| Fix | Reviewer round | Verdict | What the round found |
|---|---|---|---|
| R2 (F-150), first attempt | 1 | NOT LGTM — 1 Critical, 4 Important, 2 Minor | the band SET through the deck and CLEARED through the console seam; a nil deck swallowed the set. `NeedsRead` bypassed the cool-off. Three claims the tests did not hold |
| R2, round two | 2 | LGTM with conditions | a ghost `Finished` reset the run; faults OFF AIR counted |
| R2, conditions | 3 | LGTM | — |
| Rulings 10/12/F-163 | 1 | NOT LGTM — 1 Critical | the Director's trace wrote the coordinate pair on every step; the fix covered one line of seven and the gate never read the file; two red tests shipped in the commit |
| FR-9.4 at the writer | 2 | LGTM, one Minor taken | the redaction's bound, widened and stated |

Every round was a fresh agent on the fix commit, in its own clone, running its own plants. Two of the
five rounds found the author's own remediation wanting in a way the author's plants had not — which is
the argument for the loop.

## 5. Rulings given, and executed

1–9 (2026-09-17): R1–R7 above. 10, 12, F-163 (2026-09-17, "recommendations approved"): `9944b5c`
onward. 11: the live-audio UAT — see §6. The release mechanics ruling: a squash-merge release branch,
one commit onto `main`, never a rewrite of published history; the feature branch deleted on origin at
SHIP. And the HUM LEAD's standing step: a final local build and a regression pass of everything
signed off before BUILD exit, before the release branch is cut.

## 6. UAT and the regression pass (HUM LEAD, 2026-09-18)

On a local build after the REVIEW remediations: **the Observer pass — good. The Broadcaster pass —
good, except the thirty-minute rotation (`p3-uat.md` case 10), which is not signed off.** Cases 2 and
6 (dead air at a report's end; a relay killed mid-play) are covered by the Broadcaster pass. Case 10
is carried into VALIDATE with its state to be ruled: not run, or run and not held (§7).

## 7. Carried into VALIDATE

| Item | Why it is not a blocker for REVIEW exit |
|---|---|
| **`p3-uat.md` case 10** — the thirty-minute rotation, not signed off | A stall is dead air on a live channel; it is the one UAT case that needs the clock. Whether it was not run or did not hold is the HUM LEAD's to say; VALIDATE holds the slot |
| **F-158** — the P3(d) ruling; the pressed cut-back under a reading hazard | Audio holds today through the arbiter's re-dip; the test is owed before the direct path retires |
| **F-159** — the fault run's two authors and the free-goroutine race | Minor, self-correcting on the next finish; the `Publish` design is 0.16.5's |
| **F-160…F-162** — the OPEN requirements | Recorded scope, ruling 9 |
| **F-157** — tier three never reaches a slot | 0.16.5, its own red team |
| **F-156**, **F-137** — the shell port; the undetected history comments | 0.16.5 |

## REVIEW exit

**REVIEW APPROVED; GO 4 VALIDATE** — HUM LEAD, 2026-09-18, at `35de36a`.
