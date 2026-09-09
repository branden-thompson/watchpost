---
title: "Red-Team Verdict — 0.16.0 Broadcaster UI (DISCOVER exit)"
date: 2026-09-09
scope: "feature: 0.16.0-broadcaster-ui"
mode: multi-agent
report_template: red-team-report v1.0.0
---

# Red-Team Verdict — DISCOVER exit

`red-team: SHIP-WITH-CONDITIONS · multi-agent · scope:feature:0.16.0-broadcaster-ui · personas:[a11y, infosec, junior-dev, perf, safety-critical]`

## Verdict: SHIP-WITH-CONDITIONS

**DISCOVER may exit.**  The driving findings were a live self-contradiction inside a ruled document
about to become PLAN's input, and a missing instrument rule that would have let a builder believe
`FULL INST` was discharged while never validating an instrument — **both fixed before this report was
written.**  Two conditions remain and are named below; neither is a DISCOVER defect.

**Round composition:** 10 lenses in 5 sectioned dispatches — 4 axes (code solo, per the standing rule),
the DISCOVER phase lens, and 5 personas.  **Two lenses returned blocking verdicts** (phase lens
BLOCKED, safety+perf DO NOT PROCEED); both are dispositioned below, one accepted and one refuted on
its own arithmetic.

## Findings and dispositions

**46 findings across 10 lenses.  None dropped silently.**

### Fixed before this report (16)

| # | Lens | Finding | Sev | Disposition |
|---|---|---|---|---|
| RT-1 | Code | `config-field-table.md` asserted the WITHDRAWN D-12 unification alongside its D-20 correction | **Critical** | **FIXED.**  Two sites corrected, then grepped for the CLASS — a third survived in D-19 and is now struck, not deleted |
| RT-2 | Junior-dev | **INST-4 had no FR.**  Four of five instrument rules were mapped; the fifth was silently absent | **Critical** | **FIXED** — FR-11.6 |
| RT-3 | Phase | No requirement for operator-error recovery on the destructive DROP action | **Critical** | **FIXED** — FR-3.7.  PLAN chooses undo or confirm and **may not choose neither** |
| RT-4 | Safety | Two audio owners with no double-speak guard, and no requirement | **Critical** | **FIXED** — FR-2.5 |
| RT-5 | Docs | D-12 still read as current in its own file after D-20 amended it | **Critical** | **FIXED** — an AMENDED banner in place, plus a closure table |
| RT-6 | Business·Safety·Phase | **The mocked ON AIR does not answer the problem statement's third inability** | **Critical** | **FIXED, refined.**  See "the finding I reframed" below.  FR-5.5 |
| RT-7 | Code | 11 FRs had no Exit condition, violating the document's own founding rule | Important | **FIXED** — 10 written; the 11th is the withdrawn FR-6.5 |
| RT-8 | Code | FR-7.1's exit was satisfiable by `_ = term.BreakpointFor(w)` | Important | **FIXED** — retied to a mutant that moves the layout |
| RT-9 | Phase | No requirement for feed outage during a broadcast | Important | **FIXED** — FR-8.10 |
| RT-10 | Safety | No bound on how long a station may sit silent in STANDBY with a rail card held | Important | **FIXED** — NFR-7 |
| RT-11 | A11y | The surface switch was reachable only by a modifier chord | Important | **FIXED** — FR-1.6 |
| RT-12 | InfoSec | The tower is a real antenna position at metre precision with no stated storage boundary | Important | **FIXED** — FR-9.4 |
| RT-13 | Business | The state toggle had no requirement while promote and demote had full rigor | Important | **FIXED** — FR-5.4 |
| RT-14 | Junior-dev | The duck, the fence, the rail, dead air and STANDBY used throughout, defined nowhere in-folder | Important | **FIXED** — a glossary with citations |
| RT-15 | Business | All five metrics had an ungamed anti-solution path | Important | **FIXED** — five bounds added, each naming the hole it closes |
| RT-16 | Docs·Code | `open-questions.md` claimed 8 live blockers; all were ruled.  Rulings frontmatter said D-18, body ran to D-21.  Cross-reference paths did not resolve | Important | **FIXED** — all three |
| RT-17 | InfoSec | The plaintext clamp's reuse in the new lanes was assumed, not required | Minor | **FIXED** — FR-2.6 |

### Declined, with rationale (3)

| # | Lens | Finding | Why declined |
|---|---|---|---|
| RT-18 | Perf | *"data-cadence.md asserts affordability with no arithmetic"* — computed ~71 locations at 5 req/s and called the budget the constraint | **HALF-ACCEPTED, HALF-REFUTED.**  The missing arithmetic was real and is now shown.  **The lens's own number was wrong**: 5 req/s is the library default (`httpx.go:232`), and the app overrides it to `RatePerSec: 30` (`app/app.go:123`), with tides on a separate bucket.  ~430 locations, not 71.  The budget is **not** the binding constraint; source TTLs and unmeasured current utilisation are |
| RT-19 | Code | *"FR-9.2/9.3 justified by a planned map mentioned nowhere else — pure gold-plating, delete"* | **UNFOUNDED.**  The map appears five times in `project-brief.md` (R-7, C-11, Other Considerations) and is a **HUM LEAD ruling** — D-10, verbatim: *"That GPS is important because it will power the planned GO terminalMap visualization."*  The lens grepped only part of its own stated scope |
| RT-20 | A11y | Cross-surface key collisions on `s`, `a`, `S`, `A` are unresolved | **ALREADY ADDRESSED.**  FR-6.6 requires per-surface scoping, whose exit is *"the same key can mean different actions on the two surfaces"* — which is exactly the resolution.  What D-19 deferred was the *sharing* question, not the collision |

### Deferred to tracked items (5)

| # | Lens | Finding | Where it went |
|---|---|---|---|
| RT-21 | Code | `app/executors.go:195-196` says the Broadcaster consumer *"is 0.15.0's"* — stale, since 0.15.0 shipped without it | **F-69** — a real in-code staleness defect, AP-HIST-01 shape |
| RT-22 | Safety | The persona's own deterministic check had nothing to run against on a document-only diff; it reported its activation as not well-formed | **F-70** — and an upstream A2DH proposal: a hazard-analysis persona for document gates |
| RT-23 | A11y | No `--ascii` rendering of the mock exists, so the design is not shown to survive that mode | **PLAN task**, named in the report.  Cheap, and it needs the glyph set |
| RT-24 | InfoSec | Shared provider key: blast radius named but not sized or owned | **HUM LEAD ruling at PLAN.**  A shared key means one revocation or throttle degrades both surfaces |
| RT-25 | Business·Code | FR-4 and FR-8 are the largest, least-precedented blocks and are not required by the locked problem statement | **PLAN scope decision**, recorded.  Both are HUM-LEAD-ruled scope (D-11, D-15, D-20), so this is a size argument, not a defect |

### Accepted as-is (positives, 3)

Hygiene clean: commit messages match their diffs; **zero AI-attribution watermarks** anywhere;
follow-up items consistent; folder creation follows D-7.

## Axis coverage

| Lens | Examined | Result |
|---|---|---|
| **Code Quality** (solo) | 15 load-bearing citations spot-checked against source | **All directionally accurate.**  One in-code comment found stale (RT-21) |
| **Business Quality** | Problem-statement traceability, the five metrics, release size | 4 findings; the metrics all broken and now bounded |
| **Docs Quality** | Reader-misleading claims, superseded versions, cross-references | 5 findings, all fixed |
| **Project Hygiene** | Commits vs diffs, watermarks, follow-up consistency, folder ordering | **CLEAN — no findings** |
| **DISCOVER phase lens** | Problem understanding, completeness, alternatives, failure modes | 5 findings; 3 fixed, 2 deferred |
| **Safety-critical** | Hazard enumeration, the standby gate, the paused track, staleness, mocked ON AIR | 4 findings; 3 fixed |
| **Performance** | The cadence arithmetic, frame cost, contention, deferred work | 4 findings; 1 refuted on its own numbers |
| **Accessibility** (advisory) | Colour, minimum size, the keyboard model, `--ascii` | 6 findings; would otherwise have leaned NO-GO, **stated and not enforced** per the standing ruling |
| **InfoSec** | Credentials, location privacy, consent, input trust, config surface | 5 findings; 3 fixed |
| **Junior-dev** | Comprehension, `FULL INST`, vocabulary, actionability, fear | 4 findings; 3 fixed |

## The finding I reframed rather than accepted

**Three lenses independently reached the same conclusion**: the release does not answer the locked
problem statement's third inability, *"cannot confirm at a glance whether they are currently on the
air"*, because ON AIR is mocked.

**Convergence that strong is usually right, and here it is right about the gap and wrong about the
fix.**  Watchpost has **no radio path at all** — it produces audio, and a transmitter it cannot observe
puts it over the air.  So no version of this application can ever confirm RF output.  "On the air" can
only mean *the software is putting programme out*, and the console does answer that.

**The real defect is that the boundary was never stated.**  An operator who reads a confident ON AIR
banner and infers their antenna is radiating has been misled by us.  FR-5.5 now requires the boundary
be stated where the operator reads it, in the console — not in a design document.

## Summary

| Severity | Count | Fixed | Declined | Deferred |
|---|---|---|---|---|
| **Critical** | 6 | **6** | 0 | 0 |
| **Important** | 26 | 11 | 2 | 4 |
| **Minor** | 14 | 1 | 1 | 1 |

**Every Critical is closed.**  DISCOVER can exit, subject to two conditions:

1. **The duck's behaviour on an operator cut-over (FR-4.3) is decided at PLAN, before BUILD.**  The
   safety lens is right that it is the release's most safety-relevant undecided item.  It is an FR with
   an exit condition, so it is tracked rather than open-ended.
2. **The provider-key sharing question (RT-24) gets a HUM LEAD ruling at PLAN.**

**A note on the two blocking verdicts.**  The phase lens returned BLOCKED and the safety+perf dispatch
returned DO NOT PROCEED.  Both drove real findings, and the Criticals behind them are fixed.  The perf
half rested on arithmetic that did not survive checking.  **Recorded so the record shows the round was
adversarial and the verdicts were not softened** — they were answered.
