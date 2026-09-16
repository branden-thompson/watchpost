---
title: "0.16.0 Broadcaster UI — BUILD report"
phase: "BUILD"
status: "PRESENTED for HUM LEAD approval, 2026-09-15"
sev: "SEV-0"
gates: "verify: ALL GATES GREEN — 16 required gates · mutant-check across 355 mutants"
---

# BUILD report — 0.16.0 Broadcaster UI

## The one-line version

**The console is built, the UAT round is closed, and a blind red team found two CRITICAL defects on
the hazard path that a green gate set and a 355-mutant corpus did not.**  Both are fixed, both were
reproduced independently before being touched, and both came from one root.  Three requirements are
genuinely open and named below rather than argued away.

## What this release is

The Broadcaster console: a second surface beside Observer that runs a station — a line-up the Director
orders and the operator may reorder, an alert rail that takes the air over it, a relay bed underneath,
and a read that goes out as audio.  0.15.0 shipped the listener; this ships the operator.

## What landed since the last report

**The UAT round (D-121 … D-138)** — twelve rulings from the HUM LEAD's own testing, each with the
defect it closed:

| | What was wrong | What it is now |
| --- | --- | --- |
| D-121 | `[P]`+digit+enter did not redraw the line-up | The card window's controls sit above the keymap; an open question owns the keyboard |
| D-122 | A zone-only alert bypassed the fence | `Fence.Tracked` — the zone arm asks what the station tracks |
| D-125..127 | Three live defects found by the architecture audit | Fixed with the audit's own evidence |
| D-128 | A window's ANSWER went to the wrong surface | `observerScoped` — a key is one half of a conversation |
| D-129/a | The console's lookup was unscoped; its caveat lost its colour on a wrap | Scoped by the station; wrapped to its own window's width |
| D-130 | "Valid" meant the 25-slot pool, not the SERVICE RADIUS | The radius is the test, behind `platform/debounce` |
| D-131 | A label outlived the control it named | PRESENTER removed |
| D-132/134/136 | The console's grounds and weights | UP NEXT bold white on the bands' blue; the report on the modal tile |
| D-133 | Both editions wore one colour | Broadcaster's wordmark in its own orange |
| D-135 | The help window answered for Observer, always | It answers for the surface you are on; `[r]` became a real binding |
| D-137 | Status and data age were uncoloured | Semantic `data.*` tokens, named for the data and never the colour |
| D-138 | The held-hazard notice was the least visible thing on the frame | A band |

**The red team's findings (D-139 … D-147)** are in Critical Analysis below.

## Gate evidence, on the committed tree

Every figure below was read from the gate's own log, never from a task notification — a distinction
this session had to learn twice (see *Honest assessment*).

| Gate | Result | Evidence |
| --- | --- | --- |
| `make verify` | **ALL GATES GREEN** — the 16 gates `06_docs/required-gates.txt` names | `dist/verify-remediated3.log` (the log's verdict line carries no count; an earlier draft of this report published "104", which is the number of `ok <package>` lines in it, not a gate count) |
| `mutant-check` | green across **355** mutants (491 s) | same log |
| **`mutant-verdicts`** | **355 — 353 CAUGHT, 2 SURVIVED, 0 NO EVIDENCE** | `07-readiness/mutant-verdicts.log`, promoted by the target |
| `mutant-anchors` | 355, every anchor matches the tip | same |
| `a2dh validate` | **100% (18/18)** | run at BUILD exit; it was 94.44% until `a2dh p10 check` cleared a stale run record, which is a hard gate on the exit sequence |
| `-race` | green on `app` and `platform/lineup` | run directly after each hazard-path fix |
| `dupes` | 3 groups, **3 ratified, 0 unratified** | after collapsing the one I introduced |
| `wires` | 91 members, 2 ratified-unwired, **0 unexplained** | same |
| `lint` | no new findings (**6 baselined**) | same — the baseline is pre-existing and unchanged by this release |
| `lint-watermark` | OK — **zero AI attribution** across every commit on this branch and every tracked file | verified independently before staging, and again after the tip commit, which the cited gate run predates |
| P10 | 66 findings, **24 with a recorded reason → 42 unratified** | `dist/p10.json`, re-run at BUILD exit.  An earlier draft quoted these figures while the artefact on disk was still the 2026-09-10 run — red team round 2 caught that the evidence did not exist.  Base is `merge-base:origin/main`, i.e. the whole release |

**The two survivors owe nothing.**  `m16_scopeevents_drops_ok` — the `ok` check is defence in depth on
the hazard path, and both builders of the tracked set go through `alertKeysOf`, so no unusable key can
be in the set to match.  `m43_marine_narrowed` — the Marine arm matches `Contains(product, "Marine")`
and the live catalogue holds exactly one such product.  Both are equivalents, kept rather than
retired, dispositioned at `gates.md`'s survivor dispositions (end of file).  **A retirement is RATIFIED, never self-issued**, which
is why the sweep exits non-zero and that is correct.

**Zero NO EVIDENCE is the figure worth reading twice.**  It means nothing in the corpus failed to
compile and nothing reported silence as a pass — the distinction FR-11.3 exists for — and the one that
two mutations at this exit initially got wrong, each reporting SURVIVED when it had simply failed to
compile.

**P10 is not in `verify`.**  Its ledger is ratified at gates, never self-issued, so the 42 are
presented here for the HUM LEAD rather than accepted by me.  They are the release's standing posture,
one lower than the mid-release record.

## Critical analysis — blind red team at the exit gate

**Two independent agents, no prior context**, on the A2DH axes: *code-quality + safety-critical* and
*docs-quality + project-hygiene*.  **Every finding from BOTH rounds is below, including the ones that were inconvenient.**  Round 1's
docs/hygiene findings were fixed before this report existed and were missing from its first draft —
red team round 2 caught that omission, and they are restored under *The record's own findings*.  Nine were actionable: **eight are fixed and one is deferred to REVIEW** (finding 4, below); twelve minors are dispositioned, six of
them recorded as follow-ups because each needs a ruling rather than an edit.

### CRITICAL — both on the hazard path, both fixed

**C1 — An out-of-fence rail card stalled the entire alert rail, permanently.**
*Evidence:* `Next` has skipped out-of-fence cards since D-75 and says why: *"refusing it there would
let it block every admissible card behind it, and a hazard in the operator's own town would wait on
one that is not."*  **`toPrepare` had no such skip** — it promoted the card to Standby, and a report
standing by stops the walk.  Reproduced through this release's own headline journey (Observer queues
under a wide fence → `ctrl+b` → Broadcaster re-fences): a hazard **five miles from the transmitter**
never aired, and never would.  Not self-clearing: `dropStale` is only reachable through the air.
*Action taken:* `toPrepare` asks the fence before both arms (D-139).  Two tests pin it, including one
asserting the air and the walk agree about the fence.

**C2 — A rail holding only unreadable cards ducked the bed for ever.**
*Evidence:* `givingWay` asked `len(tracks[AlertRail])` — the raw track — where every other rail reader
asks `Projection`, which drops out-of-fence cards *because they are not read*.  Measured at 200
minutes of ticks: still ducked, relay at duck gain, nothing able to speak.
*Action taken:* it asks the projection (D-139).  A companion test pins that a READABLE hazard still
ducks — fixing it by never ducking would be the same defect reversed.
*Second half:* the agent measured `effect lineup.Duck = duck()` live, **falsifying a comment in
`app/executors.go`** that told every reader the path was inert.  True when written; P5 wired it and
the comment was not revisited, so a critical section and a lock-order warning had never been reviewed
as live.  Corrected, the lock discipline inspected and found correct, and the Director's emission —
untested, where the executor half was covered — now pinned.

### IMPORTANT — eight fixed, one deferred

| # | Finding | Action taken |
| --- | --- | --- |
| 3 | The held band counted CARDS (a burst of five read "1 HAZARD(S) HELD") and counted UNREADABLE ones, so a fence-excluded rail said *"Go ON AIR to read them"* about cards that going on air would not read | Counts arrivals across `Projection` (D-142) |
| 4 | Nothing distinguished a hazard queued from one on the air; LIVE NOW named the programme card during a takeover | **Carried to REVIEW** — see Deferred |
| 5 | `clipToWidth` counted ANSI bytes as cells, so every bold span cost 8 phantom cells; cuts landed mid-escape and left styling open.  D-138's defect one function later | Delegates to `render.TruncateCells`, the canonical owner; width **sweep** replaces four spot checks (D-141) |
| 6 | The console's scope was escapable by pressing enter inside the 300 ms debounce — the D-129 UAT defect still reachable | Enter asks the SCOPED hook at once and the press is HELD until the verdict lands (D-141) |
| 7 | A requested in-radius location outside the pool was shown as scheduled, then silently vanished — `Failed{Routed:true}` is treated as self-healing.  **FR-3.3's named trap** | Composer resolves pool ∪ requested; Producer unchanged (D-140) |
| 8 | `ClassEmergency` — the leave-now tone — was silenceable through a config migration, with no row to un-tick | Never muted (D-143) |
| 9 | `blocker()` was a CONSTANT FUNCTION: `requestState.ref` outlived D-130 and nothing assigned it.  **Its two tests asserted the constant and would pass with the switch deleted** | Dead field removed; six-case table plus a non-constancy guard (D-147) |

### MINOR — 12 raised, 6 fixed, 6 recorded

Fixed: the unsynchronised `ticker.scope` write (now under the mutex that already guarded its twin);
`alertWidths` returning a zero-width hazard column, which **fails open** because `truncate(s,0)`
returns the row unchanged; a dead `len() >= 0` invariant; the mock's non-standard GPS placeholder; a
stray 4.4 MB binary in the repo root; and the `dupes`/`mutant-anchors` knock-ons described below.

Recorded as **F-103 … F-108** because each needs a ruling rather than an edit: `scheduleIndex`
filtering differently from `Projection` (the same latent class as C1/C2, one surface along); a rail
card being the one card that can never re-hydrate, whose drop notice misattributes a hazard as a
report; `onRestored` dropping `From` and making a restored hazard fence-exempt for life; the
`TruncateCells` escape limit; `SpliceCells` with no callers and a false contract; and three surfaces
that claim more than the code does.

**One finding was fixed and then deliberately UN-fixed.**  `TruncateCells` ignores non-SGR escapes.
I fixed it — then found the fix broke the function's stated contract, *"it measures what `Width`
measures"*, since `Width` knows only SGR.  Teaching one half would make the cutter and the measurer
disagree: the class of defect behind the 2026-09-10 masthead failure.  It is also unreachable.
Reverted, the invariant that matters pinned instead, and recorded as **F-106** with the reasoning.

### The record's own findings (round 1, docs + hygiene axis)

**These were fixed before this report was written and were absent from its first draft.**  Round 2
caught the omission, which is the same failure the report describes elsewhere: a true-sounding
sentence — *"every finding is below"* — that nobody had measured.

| Finding | Action taken |
| --- | --- |
| **The release was being judged on a WORKING TREE** — 88% of the P7 build log, a new platform package, 14 test files and 22 mutants existed only on disk | Committed as `2d7c21e`; every gate figure in this report is from a run against the committed tree |
| `gates.md` cited **16 tests that no longer exist**, 14 with no retirement line — in a file whose own standard is that BUILD exit is judged on it | Reconciled: each names its successor, every successor verified to exist.  Round 2 then found **three successors that do not carry the property**, one of which hid a real hole (F-109) |
| The filed corpus record was **22 mutants behind** the corpus it described | Re-run on the committed tree — 355 — and promoted by the target itself |
| The release **shipped `exposure-scan.py` and never ran it on itself** | `07-readiness/exposure-statement.md` filed, re-derived rather than cited |
| `05-debugging/` absent under SEV-0's "ALL folders" | Populated; round 2 then found `06-key_learnings/` also absent — now opened |

### What the gates caught in my own remediation

**This is the pattern the HUM LEAD named as common across agents, and it happened three times here.**

1. `dupes` refused a structural duplicate I introduced **while fixing a data race**.  A reason is
   RATIFIED, never self-issued, so it was collapsed into `setUnder` beside `tellUnder` — the second
   caller is where the helper gets extracted.
2. `mutant-anchors` then caught the corpus drift that collapse caused: `mE5` anchored on a literal
   that no longer existed — an **unmeasured rule, not a passing one**.  Re-pointed; CAUGHT by four.
3. The `TruncateCells` revert above.

**None of these reached the red team.  All three were automated.**  The write-up — nine failure
shapes as executable probes, each earning its place by a named catch — is in
`06_docs/quality-observations.md` for A2DH skill extraction.

## Deferred, with the trigger that reopens each

| Item | Why it waits | What makes it due |
| --- | --- | --- |
| **Finding 4** — a queued hazard and one on the air render identically; LIVE NOW names the programme card during a takeover | The machinery exists (`lineupRowOf` already sets `Playing`), but making the air box a term of `lineup.OnAir(AlertRail)` changes what the operator's primary status row MEANS. That is a ruling. | REVIEW, with the HUM LEAD's reading of what LIVE NOW should name mid-takeover |
| **F-103 … F-108** | Six minors whose fixes are design decisions, not edits | REVIEW |
| **0.16.5** — time-to-first-data, the scheduler's serial provider walk, the standard workload | The cadence rule: a feature release records perf findings and defers them | 0.16.5, or `accepted-costs.md` §4's own trigger firing |
| **F-102** — one unexplained `TempDir` cleanup failure | 245 clean runs, no sighting, mechanism never identified | A second sighting |
| **F-101, F-78 … F-94** | Carried from earlier releases | Unchanged |

## Requirements traceability

Red team found **13 FR IDs with no trace outside `requirements.md`**.  Each is dispositioned below;
the table is the answer to *"is this citation drift or open scope?"* and it is **both**.

| FR | Exit criterion (abridged) | Disposition | Evidence |
| --- | --- | --- | --- |
| **FR-3.6** | *"the operator-facing notice names the dropped card"* | **OPEN** | `dropStale` → `readInstead` (`director.go:946`) queues a **listener**-facing notice; the requirement itself says *"The listener is already told."* No operator-facing notice names the card. See also **F-104**, which found the listener's notice misattributes a rail hazard as a report. |
| **FR-6.1** | Broadcaster has its own settings modal, editing only its own or shared fields | **CLOSED** | `TestTheConsoleDrawsOnlyTheSettingsThatApplyToIt` (`setup_scope_test.go:82`).  A first draft cited `TestAnUndeclaredSurfaceIsRefused`, which is a ROUTER surface-swap test and has nothing to do with the settings modal — corrected by red team round 2 |
| **FR-6.2** | *"a **round-trip test** asserts, per field, that a shared value survives a swap and a split value does not leak"* | **PARTIAL** | `TestEverySettingsRowIsRuledForItsSurface` asserts every row carries a ruled scope — it does NOT read the field table, and asserts nothing about values surviving a swap.  **No per-field round-trip test exists.**  A first draft marked this CLOSED on that gate; red team round 2 showed the gate cannot test the property |
| **FR-7.4** | Fixed chrome + one readable card within the ruled floor | **CLOSED** | `TestAboveTheFloorTheConsoleRendersLanes`, `TestBelowTheFloorTheConsoleSaysSoRatherThanOverflowing` |
| **FR-8.3** | Console usable and schedule non-empty at a **3-mile** radius | **OPEN** | No test exercises a 3-mile radius. The mechanism exists (D-122's zone-only arm, `Fence.Tracked`) and `TestAStationWithNoEpicentreOffersNothing` covers the degenerate end — but the HUM LEAD's own example is untested. |
| **FR-8.5** | Inside the radius is fetched at the priority cadence; outside does not exist | **PARTIAL** | The pool derivation is closed (`TestARestationedPoolIsWhatGetsFetched`); the CADENCE half is asserted by `TestCadenceTableIsTheDoc` for the table, not for the station's own pool. |
| **FR-8.7** | *"every cadence carries its argument beside it"* | **PARTIAL** | `TestCadenceTableIsTheDoc` diffs a generated table against `testdata/cadences.md` and is self-updating with `-update-cadences` — a change-detector.  The generated table has no argument column, so the ARGUMENT half is unasserted (red team round 2) |
| **FR-8.8** | Console shows a **per-kind** last-arrival that moves and visibly ages | **OPEN** | The card shows one per-CARD stamp (`DATA PULL … (n MIN AGO)`, now colour-coded by D-137's ladder). There is no per-KIND last-arrival. |
| **FR-9.1** | *"a test that changes the name and finds the coordinates follow, or refuses the change"* | **PARTIAL** — structurally closed, not tested as written | One source: `config.Station()` → `stationFrom` → `refsFromConfig` (`app/pool.go:49`). `TestARestationedPoolIsWhatGetsFetched`, `TestABorrowedEpicentreSaysSo`, `TestAMovedStationReResolvesItsRelays` |
| **FR-10.1** | Gain is Broadcaster's own and persisted, **distinct** from Observer's | **SUPERSEDED** | D-56 ruled the opposite: *"THE LEVEL IS MIRRORED, NEVER OWNED TWICE"* (`router.go:410`), and `air-reachability-survey.md:51` records `SetVolume` in bucket 5 — *"shared deliberately."* `TestGainPressedOnTheConsoleMovesObserversOwnLevel` asserts the mirror. **The requirement was never updated to match the ruling.** |
| **FR-11.3** | Silence is a distinct verdict; not-applicable / could-not-run never render as passed | **CLOSED** | `TestABlindMatcherCannotPass`, `TestAnAssumedBaselineStillReachesAVerdict`; `mutant-verdicts.sh`'s NO-EVIDENCE counter fails the run |
| **FR-11.4** | A surviving plant indicts the plant first | **CLOSED by practice** | Both survivors (`m16`, `m43`) are dispositioned at `gates.md`'s survivor dispositions (end of file) as equivalent mutants, HUM-LEAD-kept. Two mutations at this exit reported SURVIVED and were re-examined as INVALID (build failures). |
| **FR-11.5** | Every published count states its blind spot in its own output | **CLOSED** | `dupes`, `wires` and `mutant-anchors` each print their scope line (*"A FLOOR, not a total…"*) |

**Four are genuinely open — FR-3.6, FR-5.5, FR-8.3, FR-8.8 — three are PARTIAL, and one is
SUPERSEDED.**

**FR-5.5 was not in red team's list of 13 and is the most serious of them.**  Its boundary sentence —
*"audio out of this program; Watchpost does not observe a transmitter"* — is assigned per state and
**cannot render**: the row that would draw it is gated on `statusNote`, which has already replaced
it.  So the requirement has **no test and no implementation**, behind a `gates.md` citation that
looked like a retirement.  Restoring it as a row breaks the HUM LEAD's ruled layout (two rows of
text, a band of seven), so **it needs a ruling and is recorded as F-109** with four options.  A test
pins the dead end and fails the day the sentence renders.  FR-10.1 requires gain
to be *distinct* from Observer's; D-56 ruled the opposite and the code, the test and the architecture
survey all follow the ruling.  **The requirement was never updated to match it.**  That is a document
contradicting a ruling, and it is recorded here rather than left to read as a gap.

## Honest assessment

**What is strong.** The concurrency work: the red team's own words were that it is the strongest part
of the diff — `mastercontrol`'s lock discipline, the F-95 session guard and D-122's fence correction
are right, `-race` is green, and it could construct no path where a hazard is double-read or
misattributed.  The record is honest: it names invalid plants, a void measurement, a retracted
optimisation, a red gate that reached `origin`, and a roster that lagged the code by 118 commits —
in its own words, before anyone asked.

**What is weak, stated plainly.**
1. **Five of my own tests could not fail**, and three were found by strangers rather than by me.  The
   counter is one command — re-apply the defect and watch the new test fail — and I did not run it
   by default.  It is now written down as a probe.
2. **The two criticals shared a root that a gate cannot express**: a policy taught to two callers and
   not to three others.  No amount of coverage finds that; a reader asking *"who else asks this
   question?"* does.
3. **The release was judged on a working tree** until the red team said so.  88% of the build log
   existed only on disk.  That is a process failure, not a code one, and it invalidated nothing —
   but a reviewer reading `git log` would have seen a different release.

**Recommendation: proceed to REVIEW.**  The two hazard-path defects are fixed and pinned, the gate set
is green on the committed tree, and the three open requirements are named with their evidence rather
than closed by assertion.  Finding 4 and F-103 … F-108 are the agenda REVIEW inherits.
