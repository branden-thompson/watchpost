---
title: "0.19.5 quality and performance pass — REFLECT"
date: 2026-11-11
phase: REFLECT
sev: SEV-0
authority: HUM LEAD
status: "PRESENTED to the HUM LEAD 2026-11-11, for approval (D-59)"
---

# REFLECT — watchpost 0.19.5, the quality and performance pass

The regularly scheduled *.5 pass (D-1): no new features, five goals, six days (the brief on 2026-11-05,
the release on 2026-11-11). Read it with:
- the decision log, `../02-analysis/rulings.md` (D-1 to D-59, A-1 to A-36);
- what each batch built, `../04-development/build-log.md` (53 batches, then SHIP);
- the red teams, `red-team-plan.md` and `red-team-review.md`, and M5's grade, `m5-blind-review.md`.

## The goal, and what was delivered

**The goal** (the HUM LEAD's, 2026-11-05): no regressions; performance without re-architecting Charm or
Bubble Tea; functions consolidated where it is appropriate; every document, artifact and comment current;
code a junior maintainer can follow.

| Metric | Target | Delivered |
|---|---|---|
| M1, no regression | `make verify` green at every batch, hosted CI green on both systems, goldens byte-identical unless a D-row changes them, every change behind a pin first seen failing, the shipped defects fixed or ruled | **Met.** Every committed batch passed `make verify` (92 runs over the release, 19 failed and were fixed before a commit; `06_docs/verify-runs.md`). The REVIEW regression reviewer rendered about 8,500 frames at both ends and traced every difference to a ruling. Eight shipped defects fixed: DEF-1 to DEF-5 found at DISCOVER, DEF-6 to DEF-8 found by the pins written first (D-26, D-28) |
| M2, bytes per beat | idle at most 80 KB, radio at most 90, console at most 140 | **Met:** 76, 68 and 122 KB, from 173, 170 and 310. What is drawn is byte for byte the same |
| M3, consolidation | every DR finding landed or ruled two; `make dupes` clean | **Met** (Q5, batches 13 to 15; D-17) |
| M4, docs current | every DC finding closed; follow-ups only open work; as-built maps match; the new checks pass | **Met** (Q7, Q8; D-15, D-38) |
| M5, a junior can find their way | the four tasks traced blind against a hidden key | **Met after a fix.** T1, T2 and T4 passed. T3, a line on a console card, reached 3 of 10 sites and failed; Walkthrough 8 fixed it and the re-trace passed (batches 44 and 46) |

**Also delivered.** The full mutant sweep went from 19 hours 36 minutes to 57 minutes: the last killer is
tried first, on four workers (D-39). The SHIP sweep judged 1,058 mutants: 1,051 caught, and the 7
equivalents kept (D-39, A-24). The release paths are derived from the last tag (D-16), so no release moves
them by hand. Every ruling ID resolves through `06_docs/ids.md` (D-11, D-35, D-37).

**Scope.** 713 files changed since 0.19.3, +26,446 and −4,868 lines; 22 of the 53 batches were documents,
comments and the checks that hold them (Q7 to Q9). Nothing was cut. The re-architecting that the sweep's cost exposed was moved to
0.19.6 by ruling (D-40), not absorbed.

## Found after every gate was green, and by what

| Defect | Found by |
|---|---|
| Four lost kills: tests that had stopped killing m250, m334, m389 and m402 | the first full sweep since 0.16.0 (batch 48) |
| T3: a console card's line reachable from the docs at 3 of 10 sites | the blind M5 reviewer (D-24) |
| Six Importants: the beat pin not feeding its model back, a partial sweep promoted, a load flake credited as a catch, three in the docs | REVIEW round 1's blind reviewers (A-25 to A-28) |
| `GOTMPDIR` set for the sweep broke the sweep's baseline | the sweep, which `make verify` does not run (batch 52 reverted it) |
| `TestRetriesAreBounded`'s fake-clock race, the cause of F-190 | the sweep's baseline reading red (A-29) |
| GO-2026-6859, published after PR #37's runs passed | the release job's `make vuln` (A-32) |
| A golden that hashed the Go release's name | the first gate on go1.27.4 (A-33) |
| A test that needed a commit a squash-merge had thrown away | PR #39's hosted runs, once the branches were deleted (A-34) |
| go-tuiMaps red on the same advisory, at both its scans | its own nightly gate (D-57) |

Three of these came from the release's own machinery: the mutant sweep, the M5 trace and the blind
reviewers. Three came from the outside world moving: a new advisory, a new toolchain, and branches
deleted. None was found by a human at UAT, which passed every item without a fix round.

## What went well

- **Pin first.** Three shipped defects (DEF-6 to DEF-8) were found by pins written before the change they
  guard, not by the change itself.
- **The sweep is affordable now.** At 57 minutes it can run every release. A catch must repeat on the
  mutated tree (A-27, A-29), which cost about 2 % (3,481 s against 3,421 s).
- **A blind grade against a hidden key.** M5's reviewer found a real routing gap that no checklist would
  have found, and the fix was checked by the same trace.
- **REVIEW sized to the change.** Four blind axes, then one light round (D-41), against 0.19.0's ten
  rounds. It exited with no Critical finding and every Important fixed.
- **Performance without a visible change.** M2's three targets were met with every frame byte-identical
  and every memo behind a completeness guard.
- **Precedent applied, not re-litigated.** The release-job failure followed 0.19.0 D-353 (fix by PR, move
  the tag), and the toolchain moves followed D-152, A-82 and go-tuiMaps v0.2.2.

## What went wrong, and the anti-patterns behind it

1. **No full sweep for three releases.** 0.18.0, 0.19.0 and 0.19.3 shipped with no full sweep recorded,
   because the sweep took most of a day. Four tests quietly stopped killing their mutants in that time.
   *Anti-pattern: an instrument too slow to run is an instrument that is not run.*
2. **A check that reads the outside world, trusted from an earlier run.** PR #37's `make vuln` was green.
   The advisory database changed before the tag, and the release job found GO-2026-6859.
   *Anti-pattern: a green PR treated as green at tag time for a check whose answer changes without a
   commit.*
3. **A test that held the toolchain, not the code.** The router trace hashed frames that show
   `runtime.Version()`, so a toolchain bump broke a test about routing. *Anti-pattern: a golden that
   captures the build environment.*
4. **A test that held history the workflow deletes.** The promote test asked git for the commit the
   sweep judged, which was on a work branch, squash-merged and then deleted. The same shape is latent in
   go-tuiMaps: `TestThePlanRecordsTheMapCommitItFollows` reads `9e333bf`, a work-branch commit kept alive
   only by the `v0.2.0-rc.14` to `rc.36` tags. *Anti-pattern: a work-branch hash used as a durable
   citation. This is repeated: the agent's own branch-workflow rule already says not to, and it was not
   applied to tests.*
5. **A change made for a tool the gate does not run.** `GOTMPDIR`, set for the sweep in batch 51, broke
   the sweep's baseline, and `make verify` could not see it. *Anti-pattern: changing a tool's environment
   and gating it with a run that does not use that tool.*
6. **A target set unchecked.** The console's 140 KB stretch was set at D-4 without measuring whether the
   console beat could reach it. The PLAN red team caught it (D-22), and two tasks were added. *0.18.0's L1
   held here: the miss was found at PLAN, not at VALIDATE.*
7. **Two permissions needed mid-step.** The safety classifier blocks tag pushes and deletes. Each one
   waited for the HUM LEAD at the moment it was reached: the first tag, then the re-tag after the release
   job failed (D-55). *Anti-pattern: asking for an authority when the step is reached rather than at the
   gate that opens it.*
8. **Verdicts pointed at, not presented.** The UAT verdicts were first put in a document. The HUM LEAD
   asked for them in the conversation, and they were then asked one at a time (D-47 to D-54).
   *Anti-pattern: making the human open a file to make a decision.*
9. **The release dated before it was released.** The CHANGELOG's 0.19.5 heading was dated 2026-11-10,
   when it was written, but the release was published on 2026-11-11 (A-36 corrects it).

## Lessons

**New or repeated** says whether an earlier rule already covered the lesson. A repeated lesson failed
under a rule that relied on someone remembering it, so its control must run by itself.

| # | Lesson | New or repeated | Control: what runs by itself | Owner | How we know it held |
|---|---|---|---|---|---|
| L1 | A full mutant sweep runs every release | new | the release check refuses a tag when `06_docs/mutants/verdicts/since-v<last tag>.log` is missing or judged a commit older than the release branch | the agent, 0.19.6 | 0.19.6 ships with its own sweep record |
| L2 | A check that reads the outside world runs again just before the tag | new | the release checklist runs `make vuln` within the hour before the tag; the release job's own `make vuln` stays as the backstop | the agent, at SHIP | no release job stops on an advisory |
| L3 | Goldens hold the code, not the build environment | new | `TestRouterUpdateIsTheSameUnderRandomMessages` normalises the Go release's name (A-33); the code standards name the rule; a grep found no other instance | the agent | a toolchain bump changes no golden |
| L4 | A test reads only commits that a squash-merge keeps | repeated (branch-workflow) | A-34's pattern: read a record at the commit that recorded it. go-tuiMaps' plan-map test moves to a commit on `main` or a release tag at its next release | the agent | the libraries' and watchpost's tests pass in a `--single-branch` clone of `main` |
| L5 | A change to a tool is gated by a run of that tool | new | a change under `06_docs/mutants/` or to the sweep's environment runs a `MUTANT_ONLY` sample before commit | the agent | no sweep baseline reads red from a gate-green batch |
| L6 | Tag authority is asked at the SHIP gate, with the release's ruling | new | the SHIP ruling's question names the tag push, and the re-tag that 0.19.0 D-353 allows, as part of what is approved | the agent, with the HUM LEAD | no SHIP step waits on a permission prompt |
| L7 | A decision is presented in the conversation | new | the agent's memory; a verdict request always carries its items inline | the agent | the HUM LEAD does not have to ask |

## Corrections (the quality metric)

From the record and this session: **one correction by the HUM LEAD**, "Present those verdicts to me
here" (item 8). The agent also re-asked one ruling after finding its own question's facts wrong (D-37),
and set one target unchecked that the red team caught (D-22). Corrections made in earlier sessions and
not written down cannot be recounted now. This page counts only what the record holds.

## The SEV, in hindsight

**SEV-0 was right for the code.** 26,000 changed lines with "no regressions" as the first goal found eight
shipped defects and four lost kills. Each was found only because the pin-first rule and the full sweep
ran at their full depth.

**The documents half could have run lighter.** Q7 to Q9 changed what a reader meets, not what the program
does. They went through the same gate, and the fast docs lane applied only where a batch was Markdown
alone; most were not, as each added the check that holds it.

## Recommendations, in order

1. **0.19.6 opens with a full sweep on `main`** before the re-architecting moves any code, and closes
   with one (L1). Mutants anchor to code that is about to move. The first sweep is the baseline that
   shows a moved test still kills what it killed.
2. **Two release-checklist rows:** `make vuln` within the hour before the tag (L2); tag authority in the
   SHIP ruling (L6).
3. **Should watchpost scan at its floor, as both libraries do (go-tuiMaps v0.2.0 D-133)?** watchpost scans only
   at its CI toolchain. A host building at its floor, go 1.26.10, carries GO-2026-6859, which the
   libraries' rule would have caught. Adopting the rule raises watchpost's floor to 1.26.11. It is for
   the HUM LEAD at 0.19.6's DISCOVER, because it reverses A-32's "floor stays".
4. **go-tuiMaps' next release** carries the floor go 1.26.11 (A-35) and moves the plan-map test's commit
   (L4). 0.19.6 pins it if the re-architecting needs a library change. Otherwise it waits for the next
   library need.
5. **Carry forward:** F-227, retaking the three Settings-era screenshots.

## Where this is kept

- This page, linked from `../00-REQUIRED-READING.md`.
- The rulings: D-56 to D-59, A-35 and A-36.
- The agent's durable memory: L2, L4, L6 and L7, and the state of 0.19.6.
- `06_docs/quality-observations.md`, which receives L1 to L5 once this page is approved.
