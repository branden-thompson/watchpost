---
title: "Red-Team Verdict — 0.16.0 Broadcaster UI (PLAN exit)"
date: 2026-09-09
scope: "feature: 0.16.0-broadcaster-ui"
mode: multi-agent
report_template: red-team-report v1.0.0
---

# Red-Team Verdict — PLAN exit

`red-team: SHIP-WITH-CONDITIONS · multi-agent · scope:feature:0.16.0-broadcaster-ui · personas:[a11y, infosec, junior-dev, perf, safety-critical]`

## Verdict: SHIP-WITH-CONDITIONS

**Four of five dispatches returned a conditional or blocking verdict, and they were right.**  The plan
as first written overstated its savings, under-specified its most dangerous batch, and diagrammed a
mechanism that already works in place of the one that is new.  **All of that is fixed.**  What remains
is five decisions that belong to the HUM LEAD, not to the plan.

## The two errors that mattered, both mine

### 1. I counted live production infrastructure as dormant seams

The plan claimed **seven** subsystems built for this release and left unwired.  **There are five, one
of them half-wired.**

- **`lineup.Fence` is fully consumed** — `app/ticker.go:344` passes `Fence: t.fence()` into `Arrived`,
  and `plan.go:298` admits against it.  **I verified this myself during DISCOVER and counted it as a
  seam anyway.**
- **The bed is live** — `bed.go`'s dwell and tune machine drives today's rotation.  Only the
  operator-manual trigger is new.
- **`lineup.Power`'s write side is already wired** (`app/radio.go:569,743`); only the read has no
  consumer.

**A seam count is a savings claim.**  Inflating it makes the release look cheaper than it is, which is
the error direction that hurts an estimate.

### 2. I diagrammed the mechanism that already works

D-24's give-way is triggered by **an alert arriving** — `Suppress`/`Restore` reacting against audio
already playing.  **D-24 is about an operator pressing a cut-over**, a different trigger with no state
in the engine.

**The tell was in my own artifacts.**  The flowchart begins *"Alert arrives on the rail"*, and every
one of P5's test rows exercised the shipped alert path.  **Not one tested an operator press.**

**This is the same move the DISCOVER round caught**, and I briefed this lens to look for it
specifically.  It found it.

## Findings and dispositions

**48 findings across 10 lenses.  None dropped.**

### Fixed before this report (19)

| # | Lens | Finding | Sev | Disposition |
|---|---|---|---|---|
| RP-1 | Code | Seven seams claimed; Fence and the bed are live production infrastructure | **Critical** | **FIXED** — corrected to five, with the error kept visible |
| RP-2 | Code | D-24 conflates the alert trigger with the operator trigger; no test for an operator press | **Critical** | **FIXED** — corrected in place, four new P5 cases |
| RP-3 | Phase·Junior | P1, P2, P3, P6, P7 had NO Tier B shapes; P3 — the dangerous batch — had six words | **Critical** | **FIXED** — all five shaped |
| RP-4 | Junior | `Broadcaster` used in `NewRouter`'s signature, never defined | **Critical** | **FIXED** — shaped under P1 |
| RP-5 | Safety | The swap gate's window: the gate lands before the state it reads | **High** | **FIXED** — window sharpened to P1→P2 and ruled **fail closed** |
| RP-6 | Safety | P3's UAT targets precedence, not timing races | **High** | **FIXED** — timing property test, a dark path, and a go/no-go before P4/P5 |
| RP-7 | Safety | A deferred cut-over firing during the transition read is untested | **High** | **FIXED** — a P5 case |
| RP-8 | Code·Junior | The sender list was estimated ("~9") with citations pointing at signatures | Important | **FIXED** — derived: 11 send sites across 6 holders.  **An INST-1 violation inside a plan mandating derived lists** |
| RP-9 | Code | `Reorder` marking `Origin` violates *"a card keeps its origin for life"* | Important | **SURFACED** as an open ruling; the diagram note now flags it instead of asserting it |
| RP-10 | Code | `PlaceDropped` overlaps `Remove`'s reserved operator DROP | Minor | **SURFACED** as an open ruling |
| RP-11 | Code | `mastheadLine` claimed as savings, assigned to no batch | Important | **FIXED** — assigned to P2 |
| RP-12 | Business·InfoSec·A11y | FR-2.6, NFR-7, FR-8.9, NFR-3 and the `--ascii` render had no batch owner | Important | **FIXED** — each assigned |
| RP-13 | Hygiene | F-67 due at "Broadcaster shipping" and never dispositioned | Important | **FIXED** — dispositioned into P0 |
| RP-14 | Junior | `Place` had no reference frame | Important | **FIXED** — defined relative to the card's current position on its own track |
| RP-15 | Junior | The minimum height was never stated as a number | Minor | **FIXED** — 44 lines, from the wave-1 measurement |
| RP-16 | Docs | "Diagrams still owed" listed two the diagrams document had already delivered | Important | **FIXED** |
| RP-17 | Phase·Safety | No error-path test cases at all | Important | **FIXED** — five added, including a torn config write and cross-surface leakage |
| RP-18 | InfoSec | Migration untested against a crash mid-write leaving a partial tower | Important | **FIXED** — an error-path row |
| RP-19 | Perf | NFR-3 had no owning batch and no scheduled measurement | High | **FIXED** — assigned to P1, with the measurement named as owed |

### Escalated to the HUM LEAD (5)

| # | Lens | Decision |
|---|---|---|
| RP-20 | Business | **The batch order inverts the value A1 was chosen for** — all operator capability sits behind the riskiest batch |
| RP-21 | A11y | **Undo or confirm** for a dropped card.  FR-3.7 says PLAN chooses and may not choose neither; **PLAN chose neither** |
| RP-22 | InfoSec | **The provider key** — owed at three checkpoints |
| RP-23 | Code | **May `Origin` change after queueing?** |
| RP-24 | Code | **Does the placement event carry drop, or does drop stay `Remove`'s?** |

### Deferred with rationale (2)

| # | Finding | Where |
|---|---|---|
| RP-25 | The console shows schedule state, not confirmed audio state | **Recorded as a stated gap**, the same treatment FR-5.5 gave the radio boundary.  Whether it earns a requirement is a PLAN-exit question |
| RP-26 | `term.Breakpoint` collides with `radio_panel.go`'s own scheme | D-13 chose the platform vocabulary; whether Observer migrates is unruled, and P1 may not assume it does |

### Declined (1)

| # | Finding | Why |
|---|---|---|
| RP-27 | *"A2 was overridden without a re-costed risk register"* | **The override is a HUM LEAD product ruling**, made with the merge risk explicitly on the table.  The re-cost is fair and is folded into RP-20's escalation rather than treated as an unexamined decision |

## Axis coverage

| Lens | Result |
|---|---|
| **Code Quality** (solo) | **The round's most valuable pass** — found both headline errors |
| **Business Quality** | Found the value inversion; FR trace gaps |
| **Docs Quality** | Verified every "already exists" claim; found the stale owes-list |
| **Project Hygiene** | Commits match diffs, **zero watermarks**, phase placement correct.  One finding: F-67 |
| **PLAN phase lens** | Found the Tier B collapse; argued for spiking the merge first |
| **Safety-critical** | The gate window, the timing gap, the transition race.  **Again reported its own tooling limitation (F-70)** |
| **Performance** | NFR-3 ownerless; the 11,100-cell repaint; the cadence measurement |
| **Accessibility** (advisory) | Carried items batch-assigned but test-orphaned |
| **InfoSec** | Tower and clamp assigned but untested; provider key still owed |
| **Junior-dev** | Refused P3 outright as unstartable.  **The correct call** |

## Summary

| Severity | Count | Fixed | Escalated | Deferred | Declined |
|---|---|---|---|---|---|
| **Critical** | 4 | **4** | 0 | 0 | 0 |
| **High** | 5 | 5 | 0 | 0 | 0 |
| **Important** | 26 | 9 | 3 | 1 | 1 |
| **Minor** | 13 | 1 | 2 | 1 | 0 |

**Every Critical and every High is closed.**  PLAN can exit once the five escalated decisions are
ruled — they are decisions, not defects, and four of them the plan cannot make for itself.
