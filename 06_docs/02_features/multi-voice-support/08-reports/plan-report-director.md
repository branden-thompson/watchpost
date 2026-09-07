# PLAN Report — Station Director & the Lineup

| Field | Value |
|---|---|
| Project | **watchpost** — terminal-native live weather station |
| Scope | **MAJOR SUB-FEATURE** of 0.14.0, branch `feature/multi-voice-support` |
| Classification | **LEVEL-1 · SEV-0 · HUMAN LEAD** |
| Directives | FULL RCC · FULL PLAN · FULL DIAGRAMS · FULL TDD · FULL REPORTS · FULL GIT |
| Phase | PLAN → **awaiting HUM LEAD approval to transition to BUILD** |
| Date | 2026-09-01 |
| Predecessor | `discover-report-director.md` (approved, `00395e3`) |
| Artifacts | `03-architecture-design/{director-architecture, plan-decisions, director-build-plan}.md` · `01-objectives/director-requirements.md` (DR-1…DR-24) · `02-analysis/director-risks.md` · `07-readiness/perf-protocol.md` §6 · `app/cutover_latency_test.go` |

## The architecture — Approach C, approved

`Step(Event) → (Director, []Effect)`: a **pure** function that describes work rather than performing
it, driven by one stateless pump that dispatches the effects and feeds their results back as events.

Three approaches were compared (`director-architecture.md` keeps all three, because the rejected ones
are the record of why C was chosen and **A is the named retreat** if BUILD finds C unworkable).

**The deciding reason is not elegance.** C is the only approach where *"a test states the arrangement
it wants and asserts what is read"* is true **by construction** — feed events, assert the effect list,
no goroutines, no sleeps, no clock plumbing. Roughly thirty remediation defects on this exact code
share one root: there was no artifact to assert against, only behaviour observed through timers. A
pure `Step` produces the artifact. It also deletes the deadlock class rather than managing it, makes
DR-20's determinism structural, and mirrors the shape `modes/tty/dashboard.go`'s `Update` already uses.

**The effect set is closed and enumerated** — `BuildCard`, `Speak`, `CueTicker`, `ReleaseTicker`,
`Duck`/`Restore`, `Tune`, `Publish`. Adding one is a deliberate change to that table and to `Step`'s
output type, which is what keeps the Director a coordinator rather than something that quietly grows
the ability to do work itself.

## The measurement that shaped it (P-1 / perf-protocol §6)

| Arrangement | Median (n=5) | Network | Cache |
|---|---|---|---|
| **Cold card build, sequential** (today) | **1.03 s** | 11 | 0 |
| **Cold, concurrent floor** | 0.96 s | 11 | 0 |
| Warm | 1–3 ms | 0 | 8 |

Against §1's ≈0.7 s output buffer, a cold build is about **1.5× the audible-silence threshold**, and
**concurrency does not remove it** — the calls funnel through a shared `/points` resolution that
serialises however they are issued. The warm figure is what makes standby a *removal* rather than a
relocation.

**A published number was wrong and is corrected on the record.** The first run reported **2.14 s**
because the instrument built its client without `RatePerSec`, taking httpx's default of 5/sec where
the app uses 30 — it timed its own misconfiguration, and the figure reached four documents before the
red team caught it (PL-17). The conclusion survived; the number did not.

## Decisions

| # | Disposition |
|---|---|
| PD-1 | Build a `running` state for **Observer's own** reason; nothing for callsign; service area already exists and DR-13 completes it |
| PD-2 | **Resolved by measurement** — pre-build the card; leave audio look-ahead alone (it already exists) |
| PD-3 | Implement the staleness check; window **15 min, ratified**. Observer cannot reach the case; Broadcaster can |
| PD-4 | **No Director seam** — the Composer reads config; the Director does not relay it |
| PD-5 | **Resolved by the architecture** — a split, not a rename. `Director` owns the Lineup; `mastercontrol` is the surviving effector |
| PD-6 | Defer to Broadcaster |

**One principle carried four of them:** a stub for a concept nobody consumes is dead code
(`AP-DEAD-01`). A seam is preserved by the thing that exists now being honestly named, not by an
unread field.

## Requirements added at PLAN

DR-1…DR-21 came from DISCOVER. PLAN adds three, each from a red-team finding:

| ID | Requirement |
|---|---|
| **DR-22** | The pump is **supervised** — a panicking effect is contained, reported as a `Fault`, and the schedule continues |
| **DR-23** | The lineup is **observable at runtime** — one `WATCHPOST_DEBUG_RADIO` line per card transition |
| **DR-24** | The cue and its release are **paired**, with the same guarantee the duck already has |

And three were amended: DR-7 (pre-build yields to the alert path), DR-11 (the emergency bound names
`globalfeed.MaxPerLane`), DR-21 (the modal carries AA, keyboard and focus obligations).

## Build plan

Five phases (`director-build-plan.md`), FULL TDD, each task naming its test-first.

| Phase | Content |
|---|---|
| **0 — The net, and proof it can catch** | Pin today's watchlist advance, the duck's single owner, the takeover's marquee/voice coupling — then **mutation-validate the pins themselves**. Nothing in Phase 1 starts until each pin catches its own mutant |
| **1 — The pure core** | `ReadRank`; the Lineup/Card values; the planner; the fence; `Step`/`Effect`; the `running` state. No goroutines, no clock |
| **2 — Pump and executors** | Wiring only. **Exit: every existing golden unchanged.** Effects are dispatched, never run inline; the pump is supervised |
| **3 — The absorb** | Highest risk. The alert rail; the main-track absorb; standby + staleness; the fault channel; the paired cue/release; observability. **One item at a time through the full remediation loop** |
| **4 — Settings and the spoken surface** | ALERTS - READ ORDER (one ordering, one Max); Bonsall as the shown default; the divert notice |
| **5 — Gates and exit** | verify · alloc · p10 (0 live, 0 unmatched, ratifications mirrored to the build log) · goldens (one deliberate move) · AA register · red team |

**Deliberately not built**, recorded so nobody builds it on their own judgement: the burst-during-burst
ladder, BURST PANIC COOLDOWN, station callsign, watchlist-from-geometry, operator card controls, a
composition `Profile` field, a Broadcaster radius cap, and any change to F-16's parked offsets.

## Critical analysis (SEV-0, mandatory)

`red-team: SHIP-WITH-CONDITIONS (was NO-GO; both Criticals closed) · single-agent · scope:feature ·
personas:[safety-critical, perf, junior-dev, a11y]`

Full artifact: `red-team-plan-director.md`. Four axes, the **PLAN lens** (Principal Architect), the
**DISCOVER lens** folded in retrospectively — the DISCOVER gate had been run ad-hoc without the
framework's structure — and four personas. **InfoSec skipped with the nominal-risk judgement stated
and evidenced**, per HUM LEAD.

**Tool evidence:** `a2dh p10 check --json` read from `findings[]`, not the exit code — **20 findings,
all exempted, 0 live**; 7/7 tools RAN, none skipped; run record current; **no exemption added by this
scope**.

**Seventeen findings: 3 Critical, 8 Important, 6 Minor. All dispositioned; none dropped.**

| Severity | Findings |
|---|---|
| **Critical** | PL-1 (the pump ran effects inline, contradicting the architecture) · PL-2 (unsupervised pump, wider blast radius than today) · PL-17 (**the PD-2 baseline measured a misconfiguration**) |
| **Important** | PL-3 · PL-4 · PL-5 · PL-6 · PL-7 · PL-8 · PL-9 · PL-10 · PL-11 |
| **Minor** | PL-12 · PL-13 · PL-14 · PL-15 · PL-16 |

**Two are self-reported errors** and are recorded rather than quietly fixed, because a wrong claim in
a phase document is not self-correcting: **PL-15** (a commit message claiming something the commit
could not evidence) and **PL-17** (the instrument that timed its own misconfiguration). A third, from
DISCOVER, was the over-claim that D-C-1…D-C-6 were all answered when D-C-4 was not.

**The convergence worth naming:** PL-1 and PL-3 were the same blind spot found from two directions —
the plan treated the card-build cost as a fact to route around rather than a number to attack, and
then wrote a pump that would have paid it on the critical path anyway.

## Risks

`director-risks.md` now rates **likelihood × impact** (PL-9), not impact alone.

| # | L | I | Risk |
|---|---|---|---|
| RD-1 | **High** | High | ~30 gate-passing defects on this code, twice |
| RD-2 | Med | **High** | The absorb moves a rule with UAT history (the duck-lift bug) |
| RD-3 | Med | **High** | Four goroutines consolidate to one owner |
| RD-4 | **High** | Med | Measurement unreliability — **recurred this phase** (PL-17) |
| RD-5 | Med | Med | The significance fence is new safety-path behaviour |
| RD-6 | Low | Med | Emergency overrun has no constant bound |
| RD-7 | Med | Low–Med | Broadcaster scope leaking in |
| RD-8 | Med | Med | Bulk renames have destroyed content twice |
| RD-9 | **Low** | Med | **Downgraded** — PD-2 settled by measurement |
| RD-10 | Low | **High** | Three orderings invite a silent "harmonisation" |
| RD-11 | Low | Low | "Text at standby" validated only at Broadcaster |
| RD-12 | Low | Med | This work edits the ticker, where F-16 is unfixed |

## Gate report

```
QUALITY GATE REPORT | watchpost — Station Director & the Lineup | SEV-0 | PLAN exit
-------------------------------------------------------------
  [PASS] architecture_documented: 3 approaches, 8 diagrams, Approach C approved
  [PASS] approach_selected: HUM LEAD, "Approved for C" (2026-09-01)
  [PASS] task_breakdown: 5 phases, T0.1..T4.3, each naming its test-first
  [PASS] decisions_resolved: PD-1..PD-6, each with reasoning; 2 resolved by
         measurement/architecture rather than judgement
  [PASS] risks_assessed: RD-1..RD-12 with likelihood AND impact
  [PASS] critical_analysis_complete: 17 findings (3 Critical), all dispositioned;
         both Criticals closed; 2 self-reported errors recorded
  [PASS] p10: 0 live, 7/7 tools RAN, no exemption added by this scope
  [PASS] tree: gofmt clean, go vet clean, go mod tidy clean
  [----] human_approval: AWAITING
  [----] report_published: AWAITING (this document)
-------------------------------------------------------------
  OVERALL: PARTIAL PASS — blocked only on human approval
```

## Recommendation

**Proceed to BUILD, starting at Phase 0.**

Two conditions, both already agreed and recorded rather than outstanding: the build plan's Phase 0
pins are **mutation-validated before Phase 1 begins**, and a **second red-team round runs after
Phase 2** — this round's find-rate and the foundational scope both call for it under Step 9, and
PL-1's dispatch plus PL-2's supervision add surface that has not had its own adversarial pass.

**What I would watch.** RD-4 is the risk that has now fired twice — once in the previous release's
mutation harness, once in this phase's own instrument — and both times it produced a *plausible*
number rather than an obvious failure. The seventh anti-vacuity rule added to the build plan
addresses the specific mechanism, but the general shape is that a measurement which cannot fail
honestly is worse than none, and BUILD will produce many measurements.
