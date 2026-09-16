---
title: "0.16.0 BUILD exit — red team"
status: "Two rounds, four blind agents, 2026-09-15.  Round 2's code verdict was NOT SAFE; every finding is dispositioned below."
---

# Red team — 0.16.0 BUILD exit

**Two rounds, two blind agents each, on the A2DH axes** — *code-quality + safety-critical* and
*docs-quality + project-hygiene*.  Every agent was given no prior context, told to verify before
asserting, told a clean report was a valid result, and fenced off the long gates so it spent its
budget reading.

**This file exists because round 2 found it did not.**  `05-debugging/README.md` cited it as the
source of record for round 1 while it had never been written, which made every red-team judgement
quoted in the BUILD report unverifiable against any artefact.

## Round 1 — against the working tree

| # | Axis | Finding | Disposition |
| --- | --- | --- | --- |
| C1 | docs | **The release was being judged on a WORKING TREE** — 88% of the P7 build log and a new platform package existed only on disk | FIXED — committed as `2d7c21e` |
| C1 | code | **An out-of-fence rail card stalled the alert rail permanently** — a hazard 5 miles from the transmitter never aired | FIXED — D-139 |
| C2 | code | **A rail of unreadable cards ducked the bed for ever** — 200 minutes of ticks, still ducked | FIXED — D-139; the `executors.go` comment asserting the path was inert was corrected |
| I2 | docs | `gates.md` cited 16 tests that no longer exist, 14 unannotated | FIXED — reconciled; round 2 then found 3 successors that do not carry the property |
| I3 | docs | The filed corpus record was 22 mutants behind | FIXED — re-run on the committed tree, 355 |
| I4 | docs | No BUILD report existed | FIXED — this release's `build-report.md` |
| I5 | docs | 13 FRs with no trace outside `requirements.md` | ANSWERED — traceability table in the BUILD report |
| I6 | docs | `05-debugging/` absent under SEV-0 | FIXED — populated |
| 3 | code | The held band counted CARDS and counted UNREADABLE ones | FIXED — D-142 |
| 4 | code | A queued hazard and one on the air render identically | **DEFERRED to REVIEW** — it changes what LIVE NOW means |
| 5 | code | `clipToWidth` counted ANSI bytes as cells | FIXED — D-141 |
| 6 | code | The console's scope was escapable by racing the debounce | FIXED — D-141 |
| 7 | code | A requested out-of-pool location vanished silently | FIXED — D-140 |
| 8 | code | `ClassEmergency` was silenceable by config migration | FIXED — D-143 |
| 9 | code | `blocker()` was a constant function; its two tests asserted the constant | FIXED — D-147 |
| M7–M10 | docs | Orphan follow-up row, mock GPS placeholder, F-66's overstated closure, a stray binary | FIXED |
| 12 minors | code | 6 fixed; 6 recorded as F-103 … F-108 because each needs a ruling | RECORDED |

## Round 2 — against the committed tree, aimed at the remediation

**Verdict: NOT SAFE to take to REVIEW.**  Both new criticals were in the FEATURE, not in the
remediation — on the default path of the release's headline control.

| # | Axis | Finding | Disposition |
| --- | --- | --- | --- |
| C1 | code | **A request at the window's own default slot (15) was silently dropped** — `scheduleIndex` refused any position past the last visible card, and the window had already closed.  FR-3.3's named trap | FIXED — D-149, clamped in `Insert` so `Reorder` stays strict |
| C2 | code | **`B` and `O` typed into either location field were eaten by the Router, and `O` swapped the surface mid-word** — "Oceanside" and "Bonsall" untypeable | FIXED — D-148 |
| C1 | docs | The report's headline **"104"** was a grep artefact, not a gate count | FIXED |
| C2 | docs | **"Every finding they raised is below" was false** — four round-1 findings missing, and this file did not exist | FIXED — restored, and this file written |
| C3 | docs | **"all nine are fixed"** contradicted by the report's own table | FIXED — eight fixed, one deferred |
| I3 | code | **The fence rule had a FOURTH call site** (`refreshStandby`) the remediation missed | FIXED — D-150 |
| I4 | code | A resolver failure was reported as "this place does not exist", and disabled the retry | FIXED — D-151, a fourth answer |
| I5 | code | The held-`enter` rule was taught to one of the two fields sharing `locateState` | FIXED — D-151 |
| I6 | code | `requestedCap`'s eviction order contradicted its justification | FIXED — D-152 |
| I1/I2 | docs | FR-6.1 cited a router test; FR-6.2 was CLOSED on a gate that cannot test it | FIXED — FR-6.1 re-cited, FR-6.2 now PARTIAL |
| I3 | docs | Three reconciliation successors do not carry the property; one hid **a real hole** | FIXED for two; **F-109** opened for FR-5.5, which has no test AND no implementation |
| I4 | docs | D-139 … D-147 had no development record | FIXED — this file and the build report |
| I5 | docs | The exposure numbers were derived before the commit they live in | Re-derived below |
| I6 | docs | **A 4.4 MB binary is in this branch's history; merging publishes it** | **RULED by the HUM LEAD: "No binaries in git branch history per standard best practices."** — history rewritten |
| I7 | docs | The P10 and `a2dh validate` rows had no filed evidence | Re-run and filed |
| I8 | docs | `06-key_learnings/` absent — SEV-0 folder completeness still unmet | FIXED — opened |
| I9 | docs | The headline evidence was uncommitted again | FIXED |
| minors | both | Stale line citations, "361 commits" vs 362, two garbled sentences, 6 baselined lint findings unmentioned, M7–M10 | Corrected in the report |

## What both rounds say about the process

**Round 2's most useful observation is that the remediation of round 1 was itself defective** — it
stopped one call site short of finishing the fence rule, and taught the held-`enter` rule to one of
two twins.  That is the shape the HUM LEAD named: *"catching consequences of my own remediation …
this is a very common pattern not just of this session, but of all agents I use."*

**And the axes earned their keep twice.**  In round 1 the docs agent found the tree was uncommitted —
no code axis would have.  In round 2 it audited the BUILD report and found three over-claims in the
document the human was about to approve.
