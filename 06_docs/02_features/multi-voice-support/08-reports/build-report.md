---
title: "BUILD report — multi-voice-support (0.14.0)"
date: 2026-09-06
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "BUILD exit APPROVED 2026-09-06"
---

# BUILD report — multi-voice-support (0.14.0)

> **WRITTEN RETROSPECTIVELY, DURING REVIEW.** BUILD exit was approved on 2026-09-06 without this
> report, and the gap was found working the release checklist: every comparable feature in this repo
> has one (`severe-alerts-modals`, `watchpost-cli`) and this one did not, under a standing FULL
> REPORTS directive. Nothing here is reconstructed from memory — every claim cites the batch log,
> gate record or red-team report it comes from, and where a number was never taken this report says
> so rather than supplying one. The process finding is recorded in §7.

**Problem (locked at DISCOVER):** *A Watchpost listener who is not looking at the screen cannot tell,
by ear, that what has just started speaking is an alert rather than one of the routine reports.*

---

## 1. What was built

**Six batches plus a sub-project.** P0 measured first (`perf-protocol.md` §1–2); P1 the resolver and
the cast foundation; P2 the seams and the arbiter; P3 the maritime report and the class tones; P4
Settings, the `[S]` window and the spoken surface; P5 UAT, the performance pass and BUILD exit. The
**Station Director** ran as its own DISCOVER → PLAN → BUILD inside P3–P5, on **Approach C**: a pure
`Director.Step(Event) → (Director, []Effect)` over a closed effect set (PL-6).

**Four named roles, one owner each** (MVS-D-77, S-7) — the division that makes the pump testable:

| Role | Owner | Responsibility |
|---|---|---|
| Producer | `app/ticker.go` | notices arrivals; produces them, decides nothing |
| Director | `platform/lineup/` | pure decision; every outcome is an `Effect` |
| Composer | `app/compose_takeover.go` | turns a burst into a multi-part script |
| Reader | `app/read_script.go` | speaks a script, holds the pacing |

**What the listener gets** (the CHANGELOG carries the full list): a cast of correspondents with
per-role voices and named hand-overs; a tone per alert class before any word; a marine report for
coastal locations; a read order that is most-serious-closest-first against a cap of five, with the
remainder *spoken* rather than dropped; a relay-silence window with an offer; STANDBY, so a muted
station holds an alert instead of spending it.

**Deliberately not built,** and each is a ruling rather than an omission: the resident Piper backend
(E-1, deferred to 0.15.0 behind the §3 RSS measurement), the on-air chip (MVS-D-24), the `[V]` voice
chooser as its own window (MVS-D-3 — it became a Settings group), and `transition/`, whose script is
written and pinned but not yet on air.

## 2. Gate results

Per-batch, from `07-readiness/gates.md` §1 and the batch build logs.

| Gate | P1–P4 | P5 (exit) |
|---|---|---|
| Unit + race | green, `-race -count=2` | green |
| `make verify` | green each batch | **ALL GATES GREEN** |
| `a2dh validate` | passing/skipped-with-declaration | 17/18, one skipped with declaration |
| `make p10` | findings **8 · 9 · 11 · 11**, all covered by a ledger row | **0 live · 0 unmatched**; 1 new row presented |
| p10 ledger | P1 4 added (ratified 2026-08-30); P2 2 re-keyed, 1 deleted as dead; P3 0; P4 0 | 1 added (`app/release.go:start`), **presented, not self-approved** |
| `make alloc-budget` | green | green, pins re-taken against a live marquee |
| PTY | `make pty-severe` green | green |
| Docs | `TestWhereThingsHappenNamesRealSymbols`, `TestAcceptedCostsNamesRealSymbols` | green; NFR-8 grep re-run |
| Goldens | re-recorded per batch, reviewed against the mocks | re-recorded for the lane label and masthead |

**The `tree_hash` column was deleted deliberately**, and that is a finding rather than a tidy-up: the
P1 row recorded a value that is not a git object and does not match `p10-p1.json`. The JSON carries
the hash; a hand-copied column beside it was a second place to be wrong, and was.

## 3. Measurements

| What | Number | Where |
|---|---|---|
| Time-to-tone-start (code path) | 0.14.0 median ≤ 0.13.0 + 50 %, held | `perf-protocol.md` §1; `BenchmarkTimeToToneStart` |
| Settings alloc pin, 133×44 | hit **3,046** / miss **3,476** | §2, `TestSetupAllocBudget` |
| Settings alloc pin, 80×24 | hit **1,978** / miss **2,478** | §2 — the PLAN probe predicted 1,979; **the pin is the measurement, not the probe** |
| Per-Piper RSS, soak, disk | **not taken** | §3–5, Arch box, **post-release by HUM LEAD ruling 2026-09-06** |

Nothing new runs per tick (the on-air chip was dropped, MVS-D-24); a per-tick miss is a failure.

## 4. Deviations from the plan

All ruled and recorded in the batch logs, none silent: Setup became **Settings**; the theme chooser
became a row rather than a window; `[V]` opens Settings at the correspondents (MVS-D-19) instead of a
chooser of its own (MVS-D-3); **MARITIME → MARINE** and **Sig. Quakes → Disasters**, in the windows
and on the air, both names being narrower than what they carry; `[M]` mutes tones only, where 0.13.0
silenced tone *and* narration.

## 5. The red team — three rounds, and what each round taught

**Round 1** (`red-team-build.md`, 2026-08-31, `e31f8a5..HEAD`, 51 commits / 136 files): ten lenses
dispatched **blind** — four always-on axes, the BUILD phase lens (Distinguished Engineer, nine
challenge areas), five personas. **~110 findings.** Nine verified independently, eight upheld, **one
refuted.** Verdict **SHIP-WITH-CONDITIONS**.

**Round 2** (`red-team-build-round2.md`): the convergence pass round 1 called for, and its result is
the most important sentence in this report — **roughly thirty of round 1's own remediations were
defective, and every one had passed `make verify`, `make p10`, the allocation pins and the goldens.
The gates could not see any of them.** That file also nearly did not exist: for most of the round the
findings lived only in a chat log, which is the same as not existing.

**Round 3** (`red-team-build-round3.md`, `0e15e82..HEAD` — **224 commits, 402 files, ~8,300
production lines**, ~1.93M tokens): **four Criticals and nine Importants.** Verdict **NO-GO for the
tag until the four Criticals are dispositioned**, three of which wanted a HUM LEAD ruling rather than
a patch.

| Critical | What it was |
|---|---|
| C-1 | Muting one tone class silenced **every spoken alert, permanently**, from the next launch |
| C-2 | Read rank 1 could never be occupied — **Emergency Orders were never spoken** |
| C-3 | BD-6's significance reach was implemented, pinned, mutant-guarded, and **connected to nothing** |
| C-4 | **Every hazard active at launch was silently marked read** |

**Convergence was the strongest signal**, as designed: `Duck`/`Restore` unreachable was found by four
independent lenses; the Director documented as inert by three; five more by two each.

**The shape under all three rounds: the wire is not pinned.** C-3, R-11 and part of R-3 are one
defect — *a rule implemented and pinned in one layer, not carried by the layer that would deliver
it*. The T3.10 round had already named this shape and fixed five instances; these were three more, in
the same release. F-41 and F-42, found later at UAT, are two more again. The proposed general answer
is recorded in round 3: a producer/consumer completeness check over the closed sets.

**What the rounds did not reach,** stated because a coverage gap is worth more than a coverage claim:
no lens ran the test suite (CPU discipline). Every finding is a reading claim with a citation, not a
measured failure.

## 6. UAT

Run by the HUM LEAD against live weather on real builds. It found what no round of reading found —
including the two defects that closed during REVIEW (F-41, F-42), the Alabama watchlist-scope
silence, and F-43. **M3, the ear test, was accepted on UAT evidence** rather than a staged A/B:
*"I hear the difference when a burst contains warnings vs. only contains watches … from a UAT
perspective I am satisfied with that functionality for this version."*

## 7. Process findings

Recorded in `p5-build-log.md` §4 and repeated here because they outlive the batch:

- **The rulings governing this build existed only as commit subjects.** `"HUM LEAD, UAT 2026-08-30"`
  appears **98 times** in the diff as the governing authority and resolved to nothing on disk; four
  separate lenses traced a dozen findings back to it independently.
- **A gate was recorded green while failing.** P4's log records NFR-8's grep at zero over a README
  that still taught the retired `T` key and shipped the screenshot NFR-8 names.
- **The p10 evidence chain was broken at P1** — see §2.
- **And this report was missing entirely**, which is the fourth instance of the same class: the
  record of a decision is not the decision, and a phase that exits without its report leaves the
  next phase to reconstruct it.

## 8. Open items carried into REVIEW

`06_docs/follow-ups.md` is the register. Carried at exit: **F-22 … F-29** (including F-22, F-30 and
F-18, which round 3 ruled become 0.14.0 exit conditions because this release is cut to start
Broadcaster), plus **F-40** (the Brengel Fire evacuation invisible — its own DISCOVER against the
fire APIs, deliberately not a BUILD-exit item) and **F-43** (a tone with no words).

## BUILD exit — APPROVED

**HUM LEAD, 2026-09-06: "BUILD-EXIT APPROVED; GO 4 REVIEW."** Given after the four Criticals and
eight of the nine Importants were fixed and verified, the fresh UAT build was cut and exercised
against live weather, and the P10 ledger rows were approved.
