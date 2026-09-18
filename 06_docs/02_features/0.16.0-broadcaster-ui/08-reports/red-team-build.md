---
title: "0.16.0 BUILD exit — red team"
status: "Three rounds, nine blind agents, 2026-09-15 to 2026-09-16.  Round 2's code verdict was NOT SAFE and round 3's was DO NOT PASS; every finding is dispositioned below."
---

# Red team — 0.16.0 BUILD exit

**Three rounds. Rounds 1 and 2 ran two blind agents each, on the A2DH axes** — *code-quality + safety-critical* and
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

## Round 3 — five blind agents, told to be critical of the first two

**The brief, from the HUM LEAD:** blind agents on the A2DH axes and personas,
*"do not ad-lib"*; code-quality, Distinguished Engineer and junior-dev the most important; every
finding backed by evidence; **"no findings is also a finding"**; and the Distinguished Engineer axis
explicitly *"critical of the last 2 rounds of remediation"*.  The standard: survive an adversarial
review by an external model, and let a human engineer approach the code and contribute — this is a
public repository.

**Nine findings.  Six were defects and all six are fixed; two are dispositioned as NOT defects; one
was a corpus figure the sweep settled.  Three of the six were introduced or left standing by rounds 1
and 2** — which is what round 3 was asked to test, and it is upheld.

| # | Axis / persona | Finding | Disposition |
|---|---|---|---|
| SC-1 | safety-critical | A lapsed hazard never left the rail — a 30-minute tornado warning still there six hours later, invisible to staleness, counted by `heldNotice`, escalated to its loudest rung, offered the air | **FIXED, D-155.**  The report's framing was WRONG and corrected before fixing: it claimed the listener heard a stale read; `eventsFor` declines an expired alert, so the listener heard nothing.  The defect is the inverse — the console asserted a pending read the schedule could never take, forever.  FR-3.3 read backwards |
| SC-2 | safety-critical | The fence never re-scoped when the service radius changed | **FIXED, D-154.**  Root was deeper than reported: `d.settings.Fence` had ONE assignment in the package, behind the air-moved guard, so the settings trigger reached nothing at all.  Fixing the guard alone would have left it broken |
| F-110 | (found by the fix) | A lapsed hazard suppressed its live siblings — the composer declined the whole burst if any ref had lapsed | **FIXED by HUM LEAD ruling A**, 2026-09-16: *"valid alerts need to be read, expired alerts must never be"*, and the per-hazard half belongs to the composer |
| DE-1 / SC-3 | Distinguished Engineer + safety-critical, **independently** | The Line-Up Request window sent the typed SLOT where the schedule takes an INDEX | **FIXED, D-156.**  Two agents found it separately, which is what moved it from plausible to confirmed.  `Requested.To` is documented as "the same number `Moved.To` carries" and the move path has subtracted `liveOffset` since D-119 |
| DE-2 | Distinguished Engineer | `couldNotAsk` taught to one of two fields sharing `locateState` | **FIXED, D-157.**  Three sentences in one window disagreeing: helper said "press enter to try again", chip said "Choose a location", key did neither |
| JD-1 / JD-2 | junior-dev | **FR-1.5 was never met**, its gate green for the whole release | **FIXED, D-158.**  The most serious finding of all three rounds — see below |
| SC-5 | safety-critical / P10 | "P10 gate is RED — do not report P10 compliant" | **NOT A DEFECT.**  The BUILD report already says *"P10 is not in `verify`.  Its ledger is ratified at gates, never self-issued, so the 42 are presented here for the HUM LEAD rather than accepted by me."*  It never claimed compliance.  Closed as already-satisfied rather than made into work |
| CQ-n | code-quality | `radio.go`'s four `want.Has` branches are hard-coded | **REFRAMED, F-111.**  The branches are not the defect: each answers a different TYPED hook, and table-driving them would erase the types.  What was missing is that NOTHING NOTICES A FIFTH — `report.Kind` is a closed set the operator now picks from (FR-3.4), so a new kind would appear in the registry, the modal and the labels and compose to nothing.  A completeness guard was written; it counts rather than names, because a test listing the four by hand is a second closed set drifting from the first |
| — | project-hygiene | Corpus 358 on disk vs 355 promoted | **SETTLED by the sweep.**  `mCL1`/`mCL2`/`mCL3` — round 2's own remediation mutants — had never been run.  All three CAUGHT |

### FR-1.5 is the finding that should sting

The requirement's exit sentence is *"an override in the user's key table changes the chord"*.  The
gate asserted the swap actions **are in the key map** — and a map no override can reach satisfies
that perfectly.  `broadcasterKeyMap()` went to the Router raw, so no `[keys]` entry could change a
single console binding, including **`ctrl+b`, tmux's own default prefix** and the exact key the
multiplexer survey FR-1.5 cites says an operator will need to rebind.  The 2026-09-09 plant that
"caught" it was real and caught what it aimed at; **what it aimed at was not the requirement.**

The proxy gate is withdrawn in `gates.md` with its reasoning and renamed to what it does measure.

### What the sweep then found about the remediation itself

The 358-mutant sweep returned **352 CAUGHT, 2 SURVIVED, 4 NO EVIDENCE** — and every one of the four
unmeasured was **UNAPPLIED because this round's own fixes had moved its anchor**: `mAM1` and `mP6`
onto `indexForSlot`, `mAZ3` onto the restructured `requestSchedule`, `mCB2` onto the `onSubmit`
switch.  The same shape as `mE5` earlier in this release, caught by the same gate.  All four
re-pointed and re-verified CAUGHT; **eleven new mutants** added for D-154…D-158, plus two for F-109.

### Two things no agent found, recorded because they are real

- **`actStationToggle` has no modal guard** and is answered before D-58's *"the window on top owns
  the keys"* rule.  Believed deliberate — taking a station off air must never be captured by a form —
  but undocumented, and it is why the live offset is read per key rather than stamped at window-open.
- **The decline reason for a lapsed alert** read *"the producer holds no alert for this card"* when
  the producer held it fine.

## What both rounds say about the process

**Round 2's most useful observation is that the remediation of round 1 was itself defective** — it
stopped one call site short of finishing the fence rule, and taught the held-`enter` rule to one of
two twins.  That is the shape the HUM LEAD named: *"catching consequences of my own remediation …
this is a very common pattern not just of this session, but of all agents I use."*

**And the axes earned their keep twice.**  In round 1 the docs agent found the tree was uncommitted —
no code axis would have.  In round 2 it audited the BUILD report and found three over-claims in the
document the human was about to approve.
