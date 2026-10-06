---
title: "0.18.0 and go-tuiMaps v0.2.0 — REFLECT"
date: 2026-10-05
phase: REFLECT
sev: SEV-0
authority: HUM LEAD
status: "APPROVED by the HUM LEAD 2026-10-05 (D-283); the 0.18.0 / v0.2.0 cycle is closed"
---

# REFLECT — watchpost 0.18.0, with go-tuiMaps v0.2.0

One release train: the library (go-tuiMaps v0.2.0, radar loops, its first reviewed release) shipped first,
and watchpost 0.18.0 pinned it and shipped on it (D-239, D-243). This page reviews both. Read it with:
- the decision logs: `02-analysis/rulings.md` here (D-1 to D-282), and go-tuiMaps'
  `06_docs/02_features/radar-loops/02-analysis/rulings.md` (D-0 to D-141);
- the transferable shapes, copied into `06_docs/quality-observations.md` (its 2026-10-05 section).

## The goal, and what was delivered

**The goal** (locked at DISCOVER): a listener can tell where a hazard is relative to their place - not only
whether one exists - from the map, or from its words when they cannot see the picture.

**Delivered.**
- The map window: alert areas with severity in words and digits.
- Radar loops with HRRR's hours ahead.
- Seven weather layers: temperature, feels-like, wind and gusts, rain and snow, UV, air quality, waves.
- Fire, quakes, buoys and tides.
- Radar and Forecast modes.
- A text description that loads honestly (D-266).
- A disclosure that names what leaves the machine (D-261).
- An hourly history recorder.
- A scripted-PTY journey on the real binary (D-271).

**Cut to 0.19.0, by ruling:** eight requirements, mostly accessibility (D-244 to D-247), and reduce motion
(D-255). The remaining accessibility findings went too (D-273). RK-12's residual is ordered first in 0.19.0
(D-256).

**One known gap:** an alert with no drawable zone is not named by the map's words (D-250, D-274).

**Scope moved both ways.** The W10-W22 work - the weather layers, the sea's stations, the history store,
the About credits, the Settings notices - came from UAT rulings with no requirement rows (D-254). In-scope
accessibility work was cut at the end. The BUILD-exit red team graded that trade Minor (BQ-9) as an
ordering observation. In hindsight it is the release's main cost, because it produced RK-12's residual
and the D-273 carry-over; it is anti-pattern 5 below.

## Found after every gate was green, and by what

| Defect | Found by |
|---|---|
| The false all-clear while alerts load | two REVIEW reviewers independently (QA-1, AX-1) |
| An Overlays menu that saved changes unseen without a picture | the Accessibility reviewer (AX-2) |
| Pruning that never reached past ~250 series | the QA and Performance reviewers (QA-4, PF-1) |
| `q` hanging the station | the scripted-PTY journey, on its first run (D-271) |
| D-98's times dropped since W10 | the agent, while fixing D-266 (D-275) |
| A false disclosure of what leaves the machine | a docs worker, while fixing the README (D-261) |
| MRMS times out of order | a fuzzer added in REVIEW (`FuzzMRMSTimes`) |
| A three-point ring the decoders close differently (library) | `FuzzAgree`, at go-tuiMaps' VALIDATE (its D-140) |

Blind reviewers found three of the eight. Instruments built late found three: the journey and two
fuzzers. Fixing work found two. The lesson is in that spread: each kind of looking found what the others
missed.

## What else went well

- **Rulings one at a time, each with options, a recommendation, a counter-argument and the HUM LEAD's own words.** The log lets a reader reconstruct the why, and the rejected alternatives, for 424 decisions: 282 here, and 142 in go-tuiMaps counting its D-0, which restates watchpost's D-11.
- **Tests first, mutants restored by copy.** New guards were proven by watching a hand mutant fail.
- **Parallel workers in isolated worktrees.** The REVIEW fix batch, batch 138, was split across five workers and merged by cherry-pick with one conflict, in `app/maps.go`.

## What went wrong, and the anti-patterns behind it

1. **Docs written after the gate ran.** Batch 135's build-log entry was written after its local `make verify`, so the as-built map trailed the log and the hosted gate went red on both platforms (run 37275637194; the build log's batch 135 records it). *Anti-pattern: the record as an afterthought to the gate.*
2. **A claim from the wrong instrument.**
   - Batch 136 claimed "P10 clean" from a bare `a2dh p10 check`; `make p10` also regenerates the public mirror, which was left at 150 rows, nine short.
   - The library did the same at its D-128: "CI green" when only the quick gate had run.

   *Anti-pattern: quoting a tool that is not the gate.*
3. **A review brief with the wrong base.** All four BUILD-exit reviewers were handed v0.14.2 (`red-team-build.md`) as the base instead of the release's fork point. *Anti-pattern: a brief field filled from memory.*
4. **Parts tested, wiring not.** D-98's times were computed (tested) and drawn per step (tested), and dropped in between. The false all-clear was the same shape. RK-1 named the risk at DISCOVER, and its mitigation was built only at REVIEW (D-271). *Anti-pattern: a unit test standing in for a wiring test.*
5. **Accessibility last, by both parties.** Layers were built in response to UAT while the accessibility requirements waited. At the end:
   - the agent offered cuts;
   - the HUM LEAD sometimes chose more than was offered - D-247 cut all three where one option built two tests now, and D-273 carried every remaining finding against the agent's recommendation to fix the word-only ones.

   Both choices were made under end-of-release time pressure. *Anti-pattern: in-scope work for the excluded listener left to the end, where it is the cheapest to cut - by whoever is choosing.*
6. **A ruling question that carried an untrue clause.** D-245's question said reduce motion forces playback off; the ruling quoted it, and the code had no reduce motion (D-255). *Anti-pattern: a question's facts unchecked because it is "only a question".*
7. **Instruments first exercised at the end.**
   - M5's protocol (recorded responses) could not run on the built binary, which has no recorded tiles (D-280).
   - M6's thresholds, owed in BUILD (D-53), were set only at VALIDATE (D-281).
   - In the library, `FuzzAgree` found D-140 at VALIDATE.

   *Anti-pattern: a measure defined at PLAN and first run at the end.*
8. **Gates that passed on nothing.** The docs lane passed an unchanged tree; `-run` patterns could match no test; `test-say` skipped itself green (BUILD-exit red team CQ-9). *Anti-pattern: a gate that cannot fail.*
9. **A UAT decision never logged.** The HUM LEAD had decided, for performance, not to reveal the feed's overlays by time; it was never written as a ruling, so the code looked like a defect when it was found (D-275). *Anti-pattern: a verdict given out loud that changes a ruling, with no D-row.*
10. **Parallel workers' environment.** The worktrees started at `origin/main`, not the branch tip, and sat under the repository, where the main tree's whole-tree tests scanned them. *Anti-pattern: assuming an isolated worker inherits the caller's branch and lives out of the caller's way.*

## Lessons

**New or repeated** says whether an earlier rule already covered it. A repeated lesson failed once under a
rule that relied on remembering it (D-11: "the rule an agent has to remember is a rule that will eventually
be passed over"), so its control must run by itself.

| # | Lesson | New or repeated | Control - what runs by itself | Owner | How we know it held |
|---|---|---|---|---|---|
| L1 | A metric's instrument runs before BUILD exit | new | a PLAN-exit check: each metric's instrument run once, its number or its gap in the PLAN report | the agent, at 0.19.0 PLAN | the 0.19.0 PLAN report shows each instrument run |
| L2 | A cross-layer ruling gets an end-to-end assertion | repeated (RK-1, 0.16.0 "the wiring is untested") | the code-quality reviewer's brief asks for any cross-layer ruling with no test through the composition; the PTY journey is the default home | the agent | 0.19.0's REVIEW finds none |
| L3 | A ruling question's facts cite the code | new | the ruling question carries a `file:line` for each claim about the code; no line, no claim | the agent | no 0.19.0 ruling withdrawn for a false premise |
| L4 | Accessibility first | new (the order); ruled D-256 | 0.19.0's plan orders F-191 to F-194, F-196 to F-198, F-180 and F-205 before any new layer (F-195, the radar step, is not in D-256's order) | the HUM LEAD's order | the 0.19.0 build log lands them before MUF (D-238) |
| L5 | The record moves before the gate | repeated (`diagrams-move-with-code`; `TestTheAsBuiltMapIsKeptInStep` since 2026-09-25) | the test held - the hosted gate caught it; what failed was ordering. Control: a pre-push hook runs the docs-reading tests (seconds) | the agent | no batch lands red for a trailing record |
| L6 | A claim names its gate and mode | repeated (go-tuiMaps D-128, `ci-mode-precision`) | `make p10` now fails on a stale mirror (batch 137); the mirror cannot go stale silently | the agent | no stale mirror at an exit |
| L7 | Every gate fails closed | repeated (0.15.0's watched failures) | `TestLaneRefusesAnEmptyChange` and `TestEveryRunSelectorSelectsATest` run in the gate (batch 137) | the agent | the gate roster has a watched failure for every gate |
| L8 | A UAT verdict that changes a ruling is a D-row before its batch closes | new | the build log's batch entry lists the UAT verdicts it carries and their D-rows | the agent, with the HUM LEAD | no 0.19.0 behaviour found "decided but unlogged" |

## The SEV, in hindsight

**SEV-0 was right for watchpost.** The release answers a safety-relevant question - is this alert over me? -
and both reviews found Critical defects in exactly that answer (BQ-1, QA-1/AX-1) and in its accessible form
(AX-2). It also changes what leaves the machine. There the findings were graded Important (IS-1 to IS-4),
and the false disclosure was found while fixing, not by a reviewer - so the egress half of the SEV was
earned by care, not by a near miss.

**SEV-0 was right for go-tuiMaps v0.2.0 too,** as its first reviewed release under a host that decides
safety words from it: its REVIEW found and fixed defects in the host contract and the fetch policy
(its D-129 to D-138).

**The process around minor items was heavier than needed.** The single-item minor rulings were not counted
for this page. Refs, wording and small cleanups were asked one at a time across many sessions, although
go-tuiMaps' D-48 and the agent's memory already said to batch them.

## Recommendations, in order

1. **Dry-run every metric's instrument at PLAN (L1).** Cheapest, most leverage: VALIDATE's surprises - M5's missing tiles, M6's unset thresholds, D-140 - become PLAN-time decisions.
2. **A pre-push hook for the docs-reading tests (L5)**, and the cross-layer question in the code-quality brief (L2). Both are small, and both replace a remembered rule with a running one.
3. **Accessibility first in 0.19.0 (L4, D-256)**, already ruled; listed so it is not lost.
4. **A minor-ruling class.** Refs, wording and small cleanups are decided by the agent under clearance and reported in a batch for veto, not asked singly. That needs the HUM LEAD's ruling at 0.19.0's DISCOVER.
5. **Upstream to li-A2DH**, written as items in this repository's `quality-observations.md` "Proposed shape for A2DH" section for the HUM LEAD to file:
   - the red-team brief computes its base;
   - isolated agent worktrees start at the caller's tip and live outside the repository;
   - subagents return reports as text, since the harness refuses their report files.

**Dropped as not worth its cost:** re-running all four axes at REVIEW (D-265). Batch 138's own regressions -
the M1 key's ordering and the macOS voice - were caught by the hosted gate (run 37342571721) and fixed
before REVIEW closed; D-279 recorded that batch 138 was otherwise read only by its authors and the gates.

## Where this is kept

- This page.
- `observer-maps/00-REQUIRED-READING.md`, which links here.
- go-tuiMaps' `radar-loops/08-reports/reflect.md`, which points here.
- `06_docs/quality-observations.md`, which holds its five transferable shapes.
- The agent's durable memory, which holds L5 to L8 and the worktree lesson.
- The 0.19.0 order: D-256, F-191 to F-205, `06_docs/follow-ups.md`.
