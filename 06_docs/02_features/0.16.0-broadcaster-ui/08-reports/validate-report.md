---
title: "0.16.0 — Broadcaster UI — VALIDATE REPORT"
date: 2026-09-18
phase: VALIDATE
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
directives: FULL GIT; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD; FULL INST
status: "Awaiting HUM LEAD approval to exit VALIDATE and enter SHIP"
---

# 0.16.0 — Broadcaster UI — VALIDATE REPORT

## Bottom Line Up Front

**VALIDATE found the release would have produced a tag with no release, and one loss of operator
intent on the safety surface — and fixed both before the branch is cut.** The release workflow's
20-minute cap was under this branch's measured 21.5-minute Linux verify; it is 45 now, derived from
the measurement. The Director replaced the whole bed on every `Tuned`, so an operator's cut-over was
forgotten the moment the relay stepped or fell through; it records only where the bed went now, pinned
for both ways a bed moves, under a fresh blind reviewer.

The install path was exercised end to end on this machine (the CI-only gates run locally: five
targets, the injector lint, the installer against a local server, the tamper control). The README was
audited whole against the code by a blind reader: three Important findings, every one in text written
this week, every one fixed — the privacy sentence now says exactly where the tower's position goes.
The HUM LEAD's UAT passed both surfaces, with one case carried.

**Recommendation: exit VALIDATE.** `make verify` ALL GATES GREEN and the sweep re-run on the final
code commit `ab1f04d` (figures in §1); `make quality` PHASE-EXIT GATES GREEN and `a2dh validate`
100 % (18/18) at the exit commit.

## 1. Exit gates

| Gate | Evidence |
|---|---|
| `deployment_verified` | `make install-test` locally: `release-matrix` 5 targets, `lint-injector` OK (6 artifacts), the installer end to end against a local server, checksum verified, tamper control fired; `strings` on two binaries: 0 build paths. The hygiene reviewer repeated it in a clean clone (16.5 s) |
| `no_regressions` | `make verify` ALL GATES GREEN on `73bce2b`, and again by the hygiene reviewer from a clean clone (18 m 59 s); every package green on every VALIDATE commit; HUM LEAD UAT (§4) |
| `readme_content_audited` | Blind narrative audit of the whole README and the three linked docs (§3) |
| `stakeholder_acceptance` | HUM LEAD UAT 2026-09-18 (§4) |
| `critical_analysis_complete` | Two blind agents on the docs-quality and hygiene + safety axes, plus a fresh reviewer on the bed fix (§2) |
| `make quality` / `a2dh validate` | PHASE-EXIT GATES GREEN (P10 0 live, 0 unmatched, 0 unratified) and 100 % (18/18), run on the exit tree |
| Sweep on the final code commit | *filled from the run on `ab1f04d`* |
| `report_published` | This document |

## 2. Critical analysis

**Agent 1 — docs quality, the whole README against the code.** Every Broadcaster claim chased to its
line; the privacy sentence tested by turning the diagnostic on and reading the file; the config
example paste-loaded. Three Important: "never sent anywhere" overclaimed (the tower's pair goes to
the National Weather Service in the same `/points` request every watched place makes — the sentence
and the settings form now say so); "every one is also a row in Settings" was false for `bed_radius_mi`
(config-only now, in words); the transmitter example without `lat`/`lon` was a silent no-op that read
as a setting (replaced by the rule). Minors: STOPPED is a third state; `P`/`k` and `ctrl+d` in the key
tables; a slot count; a caller list. The console had no picture in a README that shows every other
surface — five captures from the HUM LEAD's UAT build are in now.

**Agent 2 — hygiene + safety, from a clean clone.** Verify green from the documented commands; the
release path simulated; no tracked binaries, secrets or machine paths; ten CLOSED follow-ups and five
ratified P10 rows checked against the tree, all true. Two findings that mattered:

- **The release cap (Critical).** `release.yml` allowed 20 minutes; the last Linux verify on this
  branch took 21.5, the 0.15.0 release job 16.1 with a rising trend. With `fail_on_unmatched_files`,
  the tag would exist and the installer would 404. **45 minutes now, with the measurements beside it.**
- **The bed forgot the operator (Important, FR-4.2).** `onTuned` replaced the whole bed, wiping
  `carries`, `asked` and `ducked`; a relay step or a fall-through to synth resumed the main track over
  the relay the operator chose, with no `Publish` on that step. **Fixed at `d3538a9`** — three fields
  assigned, `TestACutOverSurvivesTheBedMoving` for both ways a bed moves, the old replacement planted
  on the committed tree and CAUGHT — and sent to a fresh blind reviewer (§2a).
- Recorded, not fixed: a relay that dies under a carrying bed raises nothing at the Director (the
  deck's own fall-through is the recovery); "IN STANDBY MODE" on an ON AIR station's LIVE NOW row is
  the HUM LEAD's verbatim wording (D-89) — a HUM CALL; a station with no audio engine can show ON AIR
  with no cue (ratified design, D-91). The `tools/authoring` P10 row's function count is stale (15 →
  39; the reason holds) — refresh at the next ratification.

### 2a. The bed fix under review — two passes to LGTM

**Pass one: NOT LGTM, one Critical.** The whole-struct write the fix narrowed had been doing two jobs:
it was also the only place a landed tune cleared its pending ask. With it narrowed, every ordinary
Watchlist rotation raised a false stall ("asked to move to b and did not") thirty seconds after a
successful tune — and the suite could not see it, because the landing test ticked at ten times the
stall bound, past the dwell that issues a new tune and resets the clock. The reviewer also
constructed the `Tuned{Live:false}` case the first draft pinned and observed a station silent with the
programme paused until the operator acts — a fork, not a fix (F-164). **Pass two (`ab1f04d`): LGTM.**
A landing clears the ask; the landing test ticks inside the window and was watched red; the
fall-through keeps its standing behaviour (release the cut-over, main track resumes), pinned, with the
ruling recorded; three plants on the committed tree, each CAUGHT on its own FAIL line; the release cap
derived from the Makefile's own bound. One Minor observation left as is: `onTuned` does not settle, so
the console's bed row flips on the next tick — the shape it has always had. Reports verbatim in
`red-team-validate.md`.

## 3. README content audit

Read whole by a blind reader, not grepped; findings in §2. What the audit could not see: pixels. The
five console captures carry the demo transmitter's position in the station banner; **ruled fine by
the HUM LEAD (2026-09-18), with a recapture on the 0.16.0-stamped build if wanted, as 0.15.0 did.**

## 4. UAT

HUM LEAD, 2026-09-18, on a local build of `35de36a` (the REVIEW-exit records commit):

| Surface | Result |
|---|---|
| Observer pass (the signed-off functionality before BUILD exit) | good |
| Broadcaster pass (`p3-uat.md`, the audio signal cases) | good, except case 10 |
| Case 10 — the thirty-minute Watchlist rotation | **not signed off** — not run, or run and not held: the HUM LEAD's to say |

This pass is also the HUM LEAD's standing step before the release branch: a final local build and a
regression pass of everything signed off before BUILD exit.

## 5. Carried into SHIP

| Item | Why it is not a blocker |
|---|---|
| **`p3-uat.md` case 10** | The one UAT case that needs the clock; its state is the HUM LEAD's to rule, and a stall is what the dwell and the fault band now cover at the Director |
| **The `Tuned{Live:false}` fork** | With the cut-over surviving a fall-through to synth, the main track stays paused while the bed plays synth — the reviewer of the fix is asked to construct what follows; presented to the HUM LEAD with the sequence, not ruled here |
| **F-158, F-159, F-160…F-162, F-157, F-156, F-137** | 0.16.5's rows, each with the reason it is carried |
| **D-89's wording on an ON AIR LIVE NOW row** | A HUM CALL on an ON AIR variant |
| **`CHANGELOG.md` 0.16.0 entry** | Written and dated on the day it ships, checked at tag time |

## 6. The release's defect curve

| Round | Reviewer | Findings | Blockers |
|---|---|---|---|
| BUILD exit | 7 blind reviewers over three rounds | many | gate-layer Criticals, converged over nine oracle rounds |
| REVIEW | 8 blind reviewers, four axes | 12 asks | **5 convergent Criticals**, all fixed |
| REVIEW remediation | 3 fresh reviewers, 6 passes | 2 Critical, 6 Important | all fixed, iterated to LGTM |
| VALIDATE | 2 blind reviewers + 1 on the fix | 3 Important (docs), 1 Critical + 1 Important (hygiene) | fixed before the branch |

The curve did not reach a clean round: every round found the author's own remediation wanting in at
least one place, and the last two Criticals — a workflow cap and a struct replacement — were found by
reviewers reading the tree from outside, not by the author's plants. **That is the argument for
independent review as a phase gate, and it is also the reason 0.16.5 is a quality pass.**

## VALIDATE exit — recommendation

**Exit VALIDATE, enter SHIP**, on the HUM LEAD's word, with case 10 and the `Live:false` fork ruled or
carried explicitly.
