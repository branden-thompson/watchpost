---
title: "0.15.0 — Pre-Broadcaster UI Improvements — BUILD REPORT"
date: 2026-09-08
phase: BUILD
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
directives: FULL GIT; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD
status: "Awaiting HUM LEAD approval to exit BUILD and enter REVIEW"
---

# 0.15.0 — Pre-Broadcaster UI Improvements — BUILD REPORT

## Bottom Line Up Front

**Every FR is landed, the gate set is green, and the red team ran three times — the last two with
independent agents blind to each other and to me.** 33 commits since PLAN.

**The number that matters most in this report is not a defect count, it is a comparison.** My own
red team found 5 findings and 0 blockers. Three blind agents found ~20 and **3 blockers**. A third
round, scoped to the remediation, found **2 regressions that my own fixes had introduced**. The
author could not review the author's work, and this release has the numbers for it.

**Recommendation: exit BUILD.** One gate is deliberately red (`make journey`, F-58, HUM LEAD ruling)
and is documented as such. Nothing else is outstanding.

## 1. What was built

| FR | Outcome |
|---|---|
| FR-1 … FR-5 | Landed in B1–B5 (see the batch verdicts in `04-development/implementation-plan.md`) |
| **FR-6.1** (#13) | A location the feed cannot serve stops reading as "loading". Needed a new snapshot field, `Location.WeatherAsOf` |
| **FR-6.2** | No code — F-46 was already closed as *not a defect*; the requirement was answered by ruling it out |
| **FR-6.3** | The four config-only voice roles documented as deliberate. F-5 said three; the count was wrong |
| **FR-6.6** (F-43) | **Closed unexplained** on the HUM LEAD's ruling — one sighting, many UAT sessions since. The trap stays armed; reopen on a trace line, not a memory |
| **FR-7.1** (F-3) | `app/release.go` ruled OUT of the provider seam as a named exception, and bounded to one check at startup |
| **FR-7.2** (F-49) | The cache stays at `os.UserCacheDir()`, ruled on a property: a cache must be disposable and OS-reclaimable |
| **FR-7.3** (F-45) | The ledger gate F-45 asked for — and it found five more unratified exemptions, then 127 with no structured field |
| **FR-7.4** (F-12) | **Ruled the other way**: the attribution stamps STAY. They are citations into a live UAT log, 94% resolving, not "ephemeral" |
| **FR-7.5** | Exposure statement across tree, history, tags, artifacts and images. Largest fixable finding: `-trimpath` was set nowhere |
| **FR-8** | Every gate carries a watched failure; `--ascii` and AA completeness rebuilt |
| **FR-10** (F-40) | Dispositioned. Both halves conclude "cannot be fixed here" — the fix owed downstream is a SOURCE, not code |
| **Metric D** | Instrument built, committed, gated. 10 groups → 9 collapsed, 1 ratified, 0 unexplained |

## 2. Gate results

`make verify` — **ALL GATES GREEN** at `5b03a33`, 14 gates including `race` and the 171-mutant corpus.

| Gate | Evidence it can fail |
|---|---|
| fmt · vet · vet-tags · test-tags · tidy · vuln · race | Planted 2026-09-08, each CAUGHT. `vet-tags` fires where plain `vet` passes — its whole justification, now measured |
| lint | Both directions: a new finding, and a stale baseline row |
| lint-imports · lint-watermark · sync-go-studs · p10-unmatched | By construction — `gate-controls` runs each against known-bad input every build |
| gate-controls | **Was green while running no control at all.** Two scripts treated an unknown flag as "run normally, exit 0" |
| alloc-budget | CAUGHT on the second plant; the first never escaped and the compiler deleted it |
| dupes | Ratification is site-exact; a third copy joining a ratified group fails |
| mutant-check | 171 mutants, 155 distinct targets, **zero without an `assert`** |
| **journey** | **RED, deliberately** — 26 of 28. F-58, HUM LEAD ruling |

## 3. Measurements

- **171** mutants; every one can report `UNAPPLIED`, proven when `m44` was stranded by a collapse.
- **75** declared colour tokens, 64 registered, 11 excused with a verified mechanism (F-57).
- **Metric D: 0** unexplained duplicates in production. Test-code duplication is **17**, internal, not gated (F-59).
- **`-trimpath`**: 473 occurrences of the build path per binary → **0**.
- **Exposure**: `internal-url` and `credential` clean (2 distinct credential-shaped values, both hex-ramp fixtures).

## 4. Deviations from the plan

- **FR-6.6's timebox was never needed** — the HUM LEAD closed F-43 on the evidence instead.
- **FR-7.4 was ruled the OPPOSITE way** to how it was written. The premise ("ephemeral", "~200") was wrong on both counts: 1,135 occurrences, 94% resolving to a live log.
- **Metric D was pulled INTO the release** (HUM LEAD: "build it now when the surface area is smallest"), having been deferred at DISCOVER and carried as a gap at PLAN.
- **The `--ascii` and AA exit conditions were both worse than the plan predicted**, and the plan was already pessimistic.

## 5. The red team — three rounds, and the case for independence

| Round | Reviewer | Findings | Blockers | Regressions |
|---|---|---|---|---|
| 1 | **Me** (author) | 5 | 0 | 0 |
| 2 | 3 blind agents | ~20 | **3** | 0 |
| 3 | 3 blind agents, A2DH axes, scoped to the remediation | 15 | 0 | **2 — both mine** |

**Round 2's blockers.** (a) The #13 fix ran in `watchpost report` and nowhere a listener could see —
`platform/sched` is the dashboard's only refresh path and was the one call site that did not pass the
asked set. (b) Six live `--ascii` defects, behind states the scan never rendered. (c) The ledger gate
scored *"NOT ratified by anyone yet"* as ratified.

**Round 3's regressions, both introduced by my Round 2 fixes.** (a) I justified the stamp with a
comment asserting "reaching `Apply` means the provider responded" — **false**, proven by a probe
against `127.0.0.1:1`; a network outage made every row claim it had been answered. (b) The alerts
tier ended the shimmer before any weather arrived — a hazard I had *already guarded against
elsewhere, with the reason in a comment*.

**What the author structurally cannot find:** a false premise they wrote as justification; a hazard
they already guarded elsewhere; and a doc recording a limit their adjacent commit had closed. All
three occurred. This does not argue for more review — it argues that **an author cannot clear their
own fix**, which the A2DH red-team skill already states and which this release now has numbers for.

## 6. UAT

Run by the HUM LEAD on real builds against live feeds. Closed: Settings `[s]` esc-save, `[w]`
trim/repopulate, the Oceanside synth read, the injected tornado takeover, **a real TSTORM warning
taking over mid-read**, the fire report on screen and on air, and the quiet read on Lone Pine.

UAT found two defects the gates did not: Settings' stale model config, and a seeded radius field that
**appended** rather than replaced (typing `20` over `50` gave `5020`). Both are now pinned, and the
audit that followed traced every user-configurable setting from its row to the file and back.

## 7. Process findings

- **INST-1 to INST-5** are recorded in `quality-plan.md` as a **dimension of FULL TDD**, with the A2DH
  fold-in named as their destination. Review date: **0.17.0** — retire any that has not caught
  something, the way D-4 was.
- **Every red gate this release fired on code younger than an hour.** `lint` on `tools/dupes` the day
  it was added, the corpus on `m44` when a collapse stranded it, the CI/verify parity test when I
  added `dupes` to one side, and `lint` again on the AA parse. Gates earn their cost on the newest
  edit.
- **A gate is at its least trustworthy on the day it is added.** Six instruments were built here and
  four were wrong on their first run, each failure shaped like good news.

## 8. Open items carried into REVIEW

| Item | State |
|---|---|
| **F-58** | Lookup stalls on a freshly seeded install. `make journey` red at 26/28 by ruling; a 40-second probe is committed |
| **F-59** | 17 unexplained test-code duplicate groups — internal metric, not a release criterion |
| **F-57** | 11 unmeasurable colour tokens, each with a stated mechanism |
| Demo location | 67 fixture files still carry it; recorded API responses, a re-record not a replace. The shipped binary is clean |
| D-9 | Aspirational for three of four deciding scripts; stated rather than asserted |

## BUILD exit — recommendation

**Exit BUILD, enter REVIEW.** Gate set green; red team run three times with the last two independent;
every finding fixed, ruled, or recorded as a stated limit; one gate red by explicit ruling.
