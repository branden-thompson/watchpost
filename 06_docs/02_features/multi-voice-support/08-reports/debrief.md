---
title: "Multi-Voice Support (0.14.0) — DEBRIEF (After Action Report)"
date: 2026-09-07
phase: REFLECT
sev: SEV-0
authority: HUM LEAD
status: "Shipped — v0.14.0 tagged on 64ae770, release run 34116932007 green, 8 assets published"
---

# Multi-Voice Support (0.14.0) — DEBRIEF

**Problem (locked at DISCOVER, 2026-08-29):** *A Watchpost listener who is not looking at the screen
cannot tell, by ear, that what has just started speaking is an alert rather than one of the routine
reports.*

**Shipped 2026-09-07.** Solved: every alert class opens with its own tone before a word is spoken,
and the station reads in a cast of correspondents who hand over by name.

---

## 1. What was delivered

A cast of correspondents with per-role voices; a tone per alert class; a read order that is most
serious, closest first against a cap of five, with the remainder **spoken** rather than dropped; a
marine report for coastal locations; a window when the radio goes quiet; STANDBY; and Setup grown
into Settings. Underneath it, a new pure Director (`platform/lineup`) on Approach C, with four named
roles owning one responsibility each.

**Four defects fixed that could each have cost a listener a hazard**, and this is the list worth
re-reading before 0.15.0: a tone mute that permanently silenced every spoken alert from the next
launch; Emergency Orders that could never be read because read rank 1 was unreachable; every hazard
active at launch marked as already-heard; and a significant quake outside the radius silently
dropped. **Every one of them fails silently.** That is the shape of this product's risk.

## 2. What went well

**Blind red-teaming keeps earning its cost.** Ten lenses, dispatched with no knowledge of each
other, and convergence did the ranking for us: `Duck`/`Restore` unreachable was found independently
by four; the Director-documented-as-inert by three. Nothing else this release ranked findings as
cheaply or as honestly.

**The mutation corpus grew teeth.** 171 mutants, and the harness now reports INVALID rather than a
verdict when a failure cannot be attributed — after it told us CAUGHT for a reason unrelated to the
rule under test.

**Rulings got written down where the decision lives.** MVS-D-33's "the storm class keeps the mock's
word" sits in `tone.go` at the line a reviewer will challenge. That is why the goldens review could
tell a deliberate divergence from a defect in one read.

**The record caught the record.** Auditing the P10 ledger by enumeration rather than by reading its
own summary found the summary wrong — 23 rows still naming gates that had passed.

## 3. What to carry forward

**The wire is not pinned.** *A rule implemented and pinned in one layer, not carried by the layer
that would deliver it.* Five instances at T3.10, three more at BUILD exit (C-3, R-11, R-3), then
F-41 and F-42 at UAT, then `voices.go` bypassing the platform seam at SHIP. **Nine instances in one
release.** Round 3's proposed answer — a producer/consumer completeness check over the closed sets —
is still unbuilt and is the single highest-value thing 0.15.0 could inherit.

**Almost every defect was an instrument that looked like it was measuring and wasn't.** Assertions
that matched the marquee instead of the window; a pin above the cache that could not see a cache
bug; a mutant CAUGHT by an unrelated flaky test; a "durable record" that was one line; a platform
simulation answering "(cached)" on its second use. The question that catches all of them: **what
would this check do if the thing it checks were broken?** Asked as a control, not as a thought.

**A green check is not a checked thing.** `TestHostPlatformFollowsTheSeam` existed all along and
passed on a Mac, because there the seam and the real OS agree. The check was written; the ability to
run it where it could fail was not. That needs a different fix from "add a test".

**Prose and pixels are outside every gate.** The README documented a window this release renamed,
taught two retired controls in its images, and shipped the maintainer's account name in a screenshot
since 0.13.0. Every gate reads Go. The widest-audience surface has no instrument at all.

**Deterministic hygiene beats a threshold.** 274 GB of build cache took the volume to 1.2 GiB free
and corrupted measurements — a 14.2 s read that was 11.4 s on a clean disk. `make hygiene` now runs
as a step, not a judgement call.

## 4. Process review

**What the process caught:** the four Criticals, all before UAT. The P10 gate, which forced every
exemption to be presented rather than taken. The goldens-vs-mocks review, which found a missing
control and an `--ascii` defect that passing goldens could not.

**What it did not catch, and why:**

**The release ran on Linux for the first time in its own release PR.** The branch was local-only by
design and the last CI of any kind was 0.13.0, eight days earlier. An eight-thousand-line SEV-0
release with four red-team rounds, 171 mutants and a 127-row ledger was stopped by nobody having run
it on the other platform. **This is the finding of the release.** `make test-platforms` exists now,
but the durable fix is cheaper: push the branch early enough that CI runs on both platforms while
the work is still being done.

**UAT found four defects no amount of reading found** — F-40, F-41, F-42 and the Alabama
watchlist-scope silence — three of which were fixed here. Reading and running are not substitutes.

**Sequencing the Arch box after the release was right for the hardware and wrong as a proxy for
Linux.** The ruling was about installing a cut release on real hardware; CI then showed the *code*
did not pass on Linux at all. Those are different questions and 0.15.0 should not conflate them.

## 5. Follow-ups carried

**Open and material:** F-43 (a tone with no words — instrumented, unreproduced, undiagnosed) ·
F-40 (the Brengel Fire evacuation invisible — its own DISCOVER against the fire APIs) · F-44 (the
journey's read step still depends on its tab having an event) · F-45 (23 ledger rows now ratified;
the check that fails when a row names a passed gate is still unbuilt) · F-46 (the flattened cast is
ruled; residue removed) · F-48 (the account name published since 0.13.0 — a decision, not a bug) ·
F-49 (`~/.watchpost/` as one cache root — 0.15.0, with a migration).

**Closed at SHIP:** F-47, F-50, F-51, F-52 — three flaky or environment-bound tests, all fixed by
turning success bounds into failure bounds and by telling "this box cannot speak" apart from "our
input broke it".

**Owed before 0.15.0 DISCOVER:** `perf-protocol.md` §3 on the Arch box. 0.15.0's resident-Piper
decision (MVS-D-17 / OQ-18) has no input until it is measured.

## 6. By the numbers

- **790 commits**, 620 files, **+66,736 / −3,251**; 41 CHANGELOG bullets; 81 MVS-D decisions.
- **Red team: 7 reports** across DISCOVER, PLAN and BUILD. Round 1 ~110 findings; round 2 found
  ~30 of round 1's own remediations defective, **every one of which had passed every gate**; round 3
  (224 commits, 402 files, ~8,300 production lines, ~1.93M tokens) produced 4 Criticals and 9
  Importants and a NO-GO until dispositioned.
- **171 mutants**; the corpus guard reports CAUGHT / SURVIVED / INVALID / UNAPPLIED / SKIPPED.
- **P10:** findings 8 · 9 · 11 · 11 across P1–P4, 0 live and 0 unmatched at exit; 127 ledger rows,
  0 now naming a gate they wait for.
- **Ship:** PR #5 (plus #6 for the date), **five CI rounds** to green — four Linux failures and one
  macOS environment failure, none of which existed on the developer's machine. Release run
  34116932007 green in 13m42s; 8 assets; `v0.14.0` on `64ae770`.
- **Disk:** 1.2 GiB free → 275 GiB after `go clean -cache`; 274 GB was mutation-run build cache.

## Source documents

`08-reports/{project-brief,discover-report,plan-report,build-report,review-report,red-team-*}` ·
`07-readiness/{gates,release-checklist,goldens-vs-mocks,perf-protocol,pr-body}.md` ·
`06_docs/{follow-ups,quality-observations}.md` · `CHANGELOG.md` §0.14.0.
