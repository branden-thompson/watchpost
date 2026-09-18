---
title: "0.16.0 — Broadcaster UI — PLAN REPORT"
date: 2026-09-09
phase: PLAN
report_template: plan-report
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
directives: FULL GIT; FULL DOCS; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD; FULL INST
status: "APPROVED — HUM LEAD 2026-09-09 (\"APPROVED; GO 4 BUILD\").  PLAN exited; BUILD opened at S0."
---

# 0.16.0 — Broadcaster UI — PLAN REPORT

## Bottom Line Up Front

**The plan is eight serial batches, and the red team made it materially better by proving two of my
claims wrong.**

I said seven subsystems were built for this release and left unwired. **There are five**, and one of
the two I got wrong — the alert fence — I had personally verified as live during DISCOVER and counted
as dormant anyway. I also diagrammed the give-way mechanism that already exists, which reacts to an
alert arriving, in place of the operator-pressed cut-over that is actually new — and wrote four test
rows that all exercised the shipped path.

**Both are corrected in place rather than quietly fixed**, because the direction of the error matters:
inflating a seam count makes a release look cheaper than it is.

**Every Critical and every High from the round is closed.** What remains is **five decisions that
belong to you**, and four of them the plan cannot make for itself.

**All five are now ruled.  Recommendation: enter BUILD, beginning with the spike.**

## What PLAN produced

| Artifact | Contents |
|---|---|
| `approach-options.md` | Three decisions with options and trade-offs; A gated B and C |
| `implementation-plan.md` | Eight batches, Tier B shapes for all of them, a derived sender inventory, test-case descriptions, error paths |
| `diagrams.md` | Four diagrams: the target wire, the router fan-out, the intent path, the three lanes |
| `red-team-plan.md` | 10 lenses, 48 findings, every one dispositioned |

## The decisions already ruled

| | Ruling |
|---|---|
| **D-22** | **A1 — the full wire.** *"This release IS the MVP."* The audio merge is in scope |
| **D-23** | **B1** — one placement event, so the console names an intent and the Director owns the order |
| **D-24** | **The medium decides.** Duck the relay because it cannot be paused; pause the rendered read, drain the rail, insert a transition, resume at full volume |

**D-24 turned out to be half-built.** The give-way for an arriving alert exists and is correct. The
operator-pressed cut-over does not, and that is the part this release adds.

## The eight batches

| # | Batch | Delivers |
|---|---|---|
| **P0** | The router | The router becomes the model; 11 send sites across 6 holders rewired; Observer identical; F-67 dispositioned |
| **P1** | The console shell | Three lanes read-only from `Publish`; breakpoints; the notice at 44 lines; the `--ascii` render; the frame measurement |
| **P2** | Station state | `Power` wired; Variant C banner; the "on the air" boundary; STANDBY-before-swap; the masthead |
| **P3** | **The main-track producer** | The rotation becomes cards; **the audio merge**; no double-speak |
| **P4** | Operator intent | The placement event, the reorder mutator, the card modal, mis-action recovery |
| **P5** | The bed and the cut-over | Transitions wired, duck-per-medium, the paused main track |
| **P6** | Settings, tower, radius, cadence | The additive config table and the two boundaries |
| **P7** | Instruments and gates | Runs throughout, closed here |

**Serial, on the standing instruction.** Each pushes on its first commit so CI sees both platforms
while the work is warm.

**The swap gate fails closed** between P1 and P2, the window where a second surface exists but the
power state is not yet wired.

**P3 carries three controls, not one**: its own red team, a UAT shared with nothing, a timing property
test over randomised interleavings, a dark path where the merged producer is observed without owning
the air, and an explicit go/no-go before P4 and P5 begin.

## Critical analysis

`red-team: SHIP-WITH-CONDITIONS · multi-agent · personas:[a11y, infosec, junior-dev, perf, safety-critical]`

Full artifact: `08-reports/red-team-plan.md`.

**48 findings. 4 Critical and 5 High, all closed. 19 fixed, 5 escalated, 2 deferred with rationale,
1 declined.**

The junior-developer lens **refused the merge batch outright as unstartable**, which was the correct
call: it was named the most dangerous batch and given the least detail of any. The safety lens found a
window between two individually-sound batches. The code lens found both of my headline errors.

**One finding declined:** that the smaller release shape was overridden without a re-costed risk
register. The override is your product ruling, made with the merge risk explicitly on the table. The
re-cost is fair, and it is folded into the batch-order escalation rather than treated as an unexamined
decision.

## Gates

| Gate | Status | Evidence |
|---|---|---|
| `approaches_proposed` | **PASS** | Three decisions, options and trade-offs; A gated B and C |
| `plan_authored` | **PASS** | Eight batches, Tier B shapes for all, dependency ordering |
| `diagrams_produced` | **PASS** | Four, per FULL DIAGRAMS at PLAN |
| `tier_b_respected` | **PASS** | Signatures and shapes only; no bodies |
| `critical_analysis_complete` | **PASS** | 10 lenses; 4/4 Critical and 5/5 High closed |
| `validate_green` | **PASS** | `a2dh validate` **100% (18/18)** |
| `p10_clean` | **PASS** | `--base main`, zero findings |
| `branch_ci_green` | **PASS** | Both platforms |
| `no_watermarks` | **PASS** | Hygiene axis found zero |
| `report_published` | **PASS** | HUM LEAD approved 2026-09-09 — *"APPROVED; GO 4 BUILD"*.  **Presented BEFORE commit this time**, correcting the DISCOVER-exit inversion |
| `plan_exit` | **PASS** | Same approval; PHASE TRANSITION PLAN → BUILD |

## The five decisions — ALL RULED

| # | Decision | Ruling |
|---|---|---|
| **D-25** | The batch order | **Spike the merge FIRST**, before any batch.  *"we should have most of this infra built - the main thing is going to be the operator controls."*  The value inversion is corrected: the merge's cost is known before three batches are sunk, and operator control moves out from behind it |
| **D-26** | Undo or confirm | **BOTH.**  Stronger than my recommendation of undo alone.  The confirm guards the accidental keypress; the undo recovers the considered-but-wrong decision.  **No timer on the undo** |
| **D-29** | The provider key | **CLOSED with arithmetic, not ruled.**  See below — I should not have escalated it a third time |
| **D-27** | May a card's origin change? | **Never, while it is alive.**  New cards and copies are permitted; re-stamping is not.  The invariant stands unrelaxed and my diagram was wrong |
| **D-28** | Does placement carry drop? | **No — drop stays with `Remove`**, which reserved the job in its own comment.  D-27 decides it: moving a card and ending one are operations on different lifetimes |

### D-29 in full, because it is a correction of my own process

**I carried "the provider key" as an open ruling across three checkpoints without ever saying which key
or what the exposure was.**  Asked to be specific, the answer took one measurement:

**It is NASA FIRMS, the only keyed provider.**  Its quota is documented in the package: **5,000
transactions per 10 minutes**, two sources at one request each per location, cached ten minutes.

| | |
|---|---|
| Locations the quota supports | **~2,500 per window** |
| Today's 60-location watchlist | **120 — 2.4%** |
| A 500-location station plus that watchlist | **1,120 — 22%** |

**The concern does not survive its own numbers.**  The key stays shared and the question is closed.
What remains is a one-line note: a revoked key takes fire detections from both surfaces at once, and
the code already degrades gracefully to the unkeyed source.

**The reusable lesson:** the red team was right that the risk was unsized.  **The correct response was
to size it, not to forward it.**  That is twice this release that a concern evaporated once someone did
the arithmetic — the first was a request budget read from a library default the application overrides.
**An unsized risk is a question for whoever can measure it, not for whoever has to rule.**

## The batch order, re-cut

| # | Batch | Delivers |
|---|---|---|
| **S0** | **THE SPIKE** | One question — can the rotation's reads travel the card path without a window where both can speak?  **One session, its own worktree, three outputs, code deleted after.**  A "not answered in the box" result is a finding, not a failure |
| **P0** | The router | 11 send sites across 6 holders; Observer identical |
| **P1** | The console shell | Three lanes read-only; breakpoints; the notice at 44 lines; the plain-text render; the frame measurement |
| **P2** | Station state | Power wired; the banner; the boundary; standby-before-swap; the masthead |
| **P3** | The main-track producer | **Re-planned from S0's answer**, not from the row I first wrote |
| **P4** | **Operator intent** | The release's actual value, per D-25 |
| **P5** | The bed and the cut-over | Transitions wired, duck-per-medium, the paused main track |
| **P6** | Settings, tower, radius, cadence | The additive table and the two boundaries |
| **P7** | Instruments and gates | Throughout, closed here |

## Measurements owed, each blocking a batch

| Measurement | Blocks |
|---|---|
| Current requests per minute, idle and burst, on the main client | **P6** — the cap cannot be chosen without knowing what Observer already spends |
| Frame cost against the recorded baseline | **P1** closing — a full three-lane repaint is 11,100 cells at the reference size |

## Recommendation

**ENTER BUILD, beginning with S0 — the spike.**  Then P0, whose claim is the one a test can state
plainly: nothing changes.

**A note on this phase's own quality.** The red team found more in my plan than in my discovery work,
and the pattern is the same both times: I am most wrong when I am summarising what already exists,
and most reliable when I am reporting what I measured. The seam count and the give-way diagram were
both summaries. The derived sender list, the column budget and the arithmetic were measurements.

## Evidence

- `a2dh validate` — **100.00% (18/18)**, after refreshing a P10 record that goes stale on any worktree
  change. **Second occurrence; it is friction at every phase exit rather than a defect.**
- `a2dh p10 check --base main` — zero findings
- Branch CI green on macOS and Linux
- 10 red-team lenses in 5 sectioned dispatches; 22 subagents across the feature to date
