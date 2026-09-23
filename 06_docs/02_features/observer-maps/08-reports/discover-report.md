---
title: "0.18.0 — Observer maps — DISCOVERY REPORT"
date: 2026-09-23
phase: DISCOVER (RCC) — exit
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
directives: FULL GIT; FULL DOCS; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD; FULL INST
branch: feature/map-drawing
issue: "branden-thompson/watchpost#22"
status: "PRESENTED for the phase gate"
---

# 0.18.0 — Observer maps — DISCOVERY REPORT

## Bottom line up front

**DISCOVER is complete and recommends proceeding to PLAN.** The release draws maps in the Observer:
alert areas over the place a listener has selected, radar as a loop, fire and quake points, bounded
to the region that place sits in, with a text description wherever the picture cannot be drawn or
read.

Three things a reader should know before the detail:

1. **Every number in this record was measured, not assumed.** Three research waves made 42 live
   requests and ran the map library's own code: the scales it draws at, the five paths by which a
   view can widen, how often an alert resolves only partly, what a radar loop costs, and what each
   source reveals.
2. **Three adversarial rounds ran, and the third was the first to return mostly polish.** Seventeen
   reviewer-verdicts; the first two rounds were almost entirely negative. **Most of round 2's
   findings were defects in round 1's remediation**, which is the honest headline of this phase.
3. **Two defects were claims this record made about itself** — a requirement the ledger said existed
   and did not, and a fix whose test could not have caught its own absence. Both were found by asking
   reviewers to check the record against the tree rather than to read it.

## The problem, locked

> **"A person watching the weather in Watchpost is told that an alert applies to a place, but cannot
> see where that alert actually is: whether its area covers them, stops short of them, or lies to one
> side, so they judge how close the danger is from zone names and distances they have to picture for
> themselves."**

Locked at D-6, scored 4.5/5 with the half point recorded rather than polished away
(`01-objectives/problem-statement.md`).

## What DISCOVER produced

| Artefact | What it holds |
|---|---|
| `00-REQUIRED-READING.md` | What a cold session must know, and the fourteen rules that cost something |
| `01-objectives/project-brief.md` | The approved brief — issue #22's body; R-1..R-9, HR-1..HR-10, the egress table, 12 constraints |
| `01-objectives/problem-statement.md` | The locked statement, and seven measures hardened against the shortcut each invites |
| `01-objectives/requirements.md` | 56 functional requirements, each with its phase tag and the instrument that holds it; 6 non-functional; 15 risks |
| `02-analysis/wave1-findings.md` | Three surveys and a twelve-point cross-cutting synthesis |
| `02-analysis/rulings.md` | D-1..D-36, verbatim, with the counter-argument each ruling was given |
| `08-reports/red-team-discover.md` | Three rounds, every finding dispositioned — fixed, deferred with a row, or declined with a reason |

## Requirements, in one line each

**R-1 the window** (`g`, centred on the selected location, a 69×12 floor with one owner) · **R-2 the
bound** (never wider than the region the location sits in) · **R-3 the basemap** (network tiles,
nothing before the listener asks) · **R-4 alert areas** (through a reachable method; partial areas
never silently whole) · **R-5 radar** (both sources, as a loop) · **R-6 fire and quakes** (static
points) · **R-7 without colour** (and a text description that ships in phase 1) · **R-8 instruments**
(a learning becomes a failing test) · **R-9 choice and cost** (Settings, and the station says what a
choice costs and what it sends).

Ten host requirements ride on go-tuiMaps v0.2.0: loops, the MRMS table, a view bound, the contract
fix, the missing triage, a host-settable fetcher, a colour-independent pattern, cache retention and
purge, loop playback control, and tile-host confinement.

## Metrics of success

| | Metric | Type |
|---|---|---|
| M1 | **Where is it** — covers / stops short / lies to one side, from the picture alone | Primary |
| M1b | **Where is it, in words** — the same, from the description alone (D-36) | Primary |
| M2 | **Never global** — zero frames wider than the selected location's region | Primary |
| M3 | **Radar honesty** — the newest frame's age always visible; nothing stale drawn as current | Primary |
| M4 | **Partial honesty** — no alert shown, or withheld, without its missing part stated | Primary |
| M5 | **Time to picture** — target set at PLAN from wave-1's own numbers | Primary |
| M6 | **Loop smoothness** | Secondary |

Each carries the shortcut it forbids. M1 is the one this release is most likely to fail: most alerts
carry no polygon of their own — the field comment says four in five, measured nine in ten on
2026-09-22, marine-inflated and weather-dependent.

## What the research measured

- **Radar** is fetchable as a loop from both sources; both answer a bad request with **HTTP 200 and
  an empty image**, and one defaults to **2011** if the time is omitted. A refresh costs one request,
  not twelve, if frames are keyed by absolute time.
- **Scale**: the library's own fit puts a state at zoom 3.71 in a 69×12 box and 5.38 at 149×38;
  embedded tiles stop at zoom 3, so the default view has no offline floor.
- **The view widens on its own** by five paths — an unplaced map draws 681° of longitude; a placed
  one resized draws 197°.
- **Alerts**: 371 active, 333 without geometry, largest zone list 41, and 515 distinct zones
  nationally — three more than the store's own cap.
- **Egress**: a radar request is the rectangle on screen; a tile sequence is a pan trail.

## Risks

Fifteen, led by **RK-1: correct parts, wrongly connected** — the failure mode this codebase has had
four times, where every unit test passes and the program does nothing. Then the region bound, radar
that looks current and is not, the paired release slipping (with a stated ship-without-radar rule),
a single-source basemap with no offline floor for its default view, partial areas misread during an
outbreak, the caches as a viewing record, a planted source, and a class of listener excluded.

## Critical analysis

Three rounds, dispatched blind from `06_docs/red-team-brief.md`, each in its own scratch directory.

| | Reviewers | Verdicts | What it found |
|---|---|---|---|
| Round 1 | 5 (4 axes + DISCOVER lens + 2 personas) | 6 of 7 negative | A consent promise the code contradicted; a bound excluding Alaska, Hawaii and the territories; three classes of listener with nothing; a metric with no grader; no threat model; a gate that failed open and printed a pass |
| Round 2 | 5, given round 1's fixes | 5 of 5 negative | **Most findings were in round 1's remediation**: three requirements that could not be built as written, a phase boundary that no longer matched, and a red gate at HEAD caused by the remediation report itself |
| Round 3 | 3, scoped to both remediations | mixed; code axis "convergence with one exception" | A requirement the ledger claimed and nobody wrote; a live false negative in a guard; a tail of one-sentence record edits |
| Verification | 2, claim-by-claim against the tree | Record **CONVERGED**; code found 3 unproved guards, since fixed and mutation-killed | — |

Every finding is dispositioned in `red-team-discover.md`: fixed, deferred with a follow-up row, or
declined with a reason. **Two round-1 declines were reversed** when a mutation test refuted them, and
the reversals are rows rather than edits.

## Gates

```
QUALITY GATE REPORT | watchpost 0.18.0 | SEV-0 | DISCOVER exit
-------------------------------------------------------------
  [PASS] requirements_documented     : 56 FRs, 6 NFRs, each with its instrument or NO INSTRUMENT YET
  [PASS] constraints_identified      : 12, measured at 17e40d4 / the v0.1.0 tag bbc039a
  [PASS] risks_assessed              : 15, each with a mitigation pointing at a requirement
  [PASS] critical_analysis_complete  : 3 rounds + a verification pass; every finding dispositioned
  [PASS] a2dh validate               : 100.00% (17/17)
  [PASS] make verify                 : ALL GATES GREEN at 9e65f10
  [PASS] p10                         : 0 live findings, all six analyzers running
  [----] human_approval              : AWAITING
  [----] report_published            : AWAITING (this document)
-------------------------------------------------------------
  OVERALL: PARTIAL PASS — the two open gates are the HUM LEAD's
```

## What PLAN inherits

1. **The render decision.** Whether the library is called from the update loop or from the view under
   a lock — both have a real cost, and wave 1 measured the arguments on each side.
2. **The partial-area ruling**, to be made with a drawing on screen, not before.
3. **M5's target**, re-derived from wave-1's numbers.
4. **Three requirements marked NO INSTRUMENT YET**: the zone-refresh rule, the network-cost warning
   threshold, and the loop rate ceiling.
5. **A structural cure for this phase's recurring defect**: the bound, the metric set and the phase
   boundary are each stated in three documents, and drift between those copies was found in every
   round. Making `requirements.md` normative and having the other two cite it is deferred to PLAN
   entry as a change to an approved artefact.
6. **go-tuiMaps v0.2.0's brief needs amending** — its host requirements grew from five to ten here,
   several from watchpost's accessibility and security review rather than from the library's own
   record.

## Recommendation

**Proceed to PLAN.** The record answers what is being built, for whom, at what cost, and how each
promise will be held. Where it cannot answer yet, it says so in the requirement rather than in a
footnote.

Two cautions carried forward. **First:** this phase's own defect rate was high, and the instruments
that caught it were the ones that read the artefact rather than the account of it — mutation tests,
ledger-versus-tree checks, and the gates. PLAN should keep that habit rather than trust a green
summary. **Second:** 0.18.0 now depends on a sibling release that has not started its own planning,
and the drop-dead rule in RK-4 is what keeps that dependency from becoming an open-ended wait.

## Evidence

- Commits: `17e40d4`, `d3a40d5`, `e54df7d`, `5c36042`, `ebdbcb1`, `35de739`, `86e1e93`, `e1cc972`,
  `af4bb6c`, `9e65f10`
- `a2dh validate` 100% and `make verify` green at HEAD; P10 clean with golangci-lint, staticcheck,
  go vet, govulncheck, the P10 analyzers and the density pass all running
- Wave-1 measurements, with every UNVERIFIED item marked as such
- Three red-team rounds and a verification pass, all dispositioned

## Source documents

`00-REQUIRED-READING.md` · `01-objectives/project-brief.md` · `01-objectives/problem-statement.md` ·
`01-objectives/requirements.md` · `02-analysis/rulings.md` · `02-analysis/wave1-findings.md` ·
`08-reports/red-team-discover.md` · `06_docs/follow-ups.md` (F-174..F-180) ·
`06_docs/handoff-0.18.0.md` · go-tuiMaps `06_docs/02_features/radar-loops/`
