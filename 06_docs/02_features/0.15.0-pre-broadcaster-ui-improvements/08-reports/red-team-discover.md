---
title: "0.15.0 — Pre-Broadcaster UI Improvements — RED TEAM, DISCOVER EXIT"
date: 2026-09-07
phase: DISCOVER (exit)
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
lenses: "4 axes + 5 personas + DISCOVER phase lens + Principal Engineer — 11, dispatched blind"
status: "NO-GO as written. Conditions below; six already discharged."
---

# Red team — DISCOVER exit

## Bottom Line Up Front / BLUF

Eleven lenses reviewed the discovery report, blind to one another, scoped to DISCOVER.  The verdict
is **NO-GO as written**, and the reason is not that the phase was shallow — it was unusually deep
where it looked.  The reason is that **a reader cannot tell scope reduction from scope
disappearance**, and the report made, in its own text, three instances of the error it was written
to indict.

Convergence did the ranking, which is why the lenses ran blind.  Seven of eleven independently found
the same Critical without being pointed at it.

## What convergence found

| Finding | Independent lenses | Verified |
|---|---|---|
| The brief's **R4 theme vanished** from the report with no descope line | **7 of 11** | Yes — `grep` returns 0 for `F-15`, `F-18`, `F-37`, `F-44`, `F-47`, `lint`, `staticcheck`, `ascii`, `AA` |
| The **brief's four Metrics of Success appear nowhere** in the report | 4 | Yes — 0 hits for `metric` |
| The **0.14.2 lane guard silently skipped lanes** | 2 | Yes — probe: 4 of 7 asserted, 3 skipped by `continue` |
| **#17 has no requirement**, only a risk row, while F-43 got one | 3 | Yes |
| **F-40's deferral** carries no exposure statement and no date | 3 | Yes |
| **More than two classifiers** over NWS product strings | 2 | Yes — four |
| The report **exits a phase it says cannot be exited** | 4 | Yes |
| The **counts do not reconcile** (PII, file totals, ledger rows) | 5 | Yes |

## The three self-inflicted findings

The report's thesis is that a green number is not coverage.  It then made that error three times.

1. **The lane guard.**  `TestEveryFeedLaneSurvivesTheMarqueeMap`, written that morning for #15 and
   named in the report as "the working template," walked `category.Lanes()` and `continue`d when its
   fixture helper disagreed.  It asserted **4 of 7 lanes** and reported green.  The three it skipped
   included **Advisories — the exact hazard FR-2.2 names.**  A `continue` where F-30 has a
   `t.Skipf`: the same defect, in the instrument written to catch that defect.
2. **F-30's coverage.**  The report read "11 of 11 modals, zero skips" as coverage and declared the
   row satisfied.  That number describes today's windows.  The guard `t.Skipf`s an unfixtured window
   and descends into three hard-coded nested structs.  Corrected in the report before the red team
   ran; found independently anyway.
3. **Issue #17's argument.**  "52 ms of work, unmoved when starved to one core, missed a 5 s bound —
   96×."  `engine.go:519` is a `time.NewTicker(50 * time.Millisecond)`.  The 52 ms is one poll
   interval, and a timer is invariant under CPU starvation **by construction**.  The measurement
   could not have moved regardless of what the code did.  Publicly corrected on the issue.

**All three are one error: reading a number without asking what produces it.**

## Findings that change the release

### C-1 — R4 must be reinstated or buried in writing  *(7 lenses)*

R4.1 (`make lint` — `golangci-lint` and `staticcheck` have never run as a gate here), R4.2 (the AA
register and `--ascii` completeness test), and R4.3 (the PTY precondition) are absent.  Only R4.4
survived, as NFR-3.  The locked problem statement names this theme explicitly — *"an AA register
that shrinks when a token is added, and `--ascii` scans that miss four windows, are how a surface
silently degrades"* — and the public issue #14 still promises it.

Two lenses measured live gaps behind it: **eleven declared tokens outside `aaPairs()`**, and
`TestEveryPaintedPairReadsAAInEveryTheme` iterating the hand-list its name claims completeness over.

**Disposition: reinstated as FR-8.**

### C-2 — There are four classifiers, not two  *(2 lenses, verified)*

FR-1.2 and FR-2.2 name `severe.Classify` and `globalfeed.LaneOf`.  Also present:

- `domains/radio/cast/tone.go:186` `Classify(product) Class` — **on the audio path**, no
  civil-emergency arm, so an Evacuation Immediate falls to the `ClassWarning` default.
- `platform/render/sgr.go:226` `AlertIsWarning(event, severity)` — `strings.Contains(event,
  "Warning")`, self-described as *"THE warning-vs-advisory classifier (single owner)"*.

The brief's handoff said the cross-table over all of them **was the requirement**.  It was not
produced.  The omitted one is on the path #7 broke.

### C-3 — The PII exposure is live, not prospective  *(1 lens, verified)*

The report says *"publication would leak PII."*  `gh repo view` → `PUBLIC`; `06_docs` is on
`origin/main`; 121 occurrences are **already published**, since 0.13.0.  A gate that blocks future
writes while the same content sits in every published tag is theatre.  A second lens found the scan
was pattern-limited to the maintainer's *name* and missed a **ZIP-level home locality in 35 files**.

### C-4 — #17 needs a requirement, not a risk row  *(3 lenses)*

Three lenses noted that F-43 was promoted to a requirement with a timebox while #17 — an
unexplained intermittent failure in the path that speaks alerts — was parked as a risk with an
activity for a mitigation.  One lens traced a plausible mechanism in shipped code: `watch()`
(`engine.go:518`) detects completion only by polling `IsPlaying()` and carries **no deadline on a
read at all**.

**Disposition: FR-9 — every spoken read completes, fails, or reports a fault within a stated bound.**
That requirement stands whether or not #17 is ever reproduced.

### C-5 — F-40's deferral has no exposure statement  *(3 lenses)*

The brief calls it "the purest instance of the locked problem statement in the whole ledger" and
defers it behind the largest release in the project's history, with no user count, no season, no
date, and no interim mitigation.  One lens noted the sharper half: F-40(b) is a **coverage**
question — a county or CAL FIRE evacuation order may never enter the NWS feed at all — so the
release ships a known blindness and tells the listener nothing.

### C-6 — RS-2 was blocking the wrong thing  *(1 lens, verified)*

`tileMemo` and `boxMemo` are fields on the FIRMS and USGS **clients** — provider fetch, not the
render frame.  §3–5 measure Piper RSS, app soak, and extracted disk bytes; none can measure a memo
consolidation in two fetch paths, and a frame allocation pin would have returned a **green false
pass**.  §3/§4 remain the right instrument for OQ-18, which is a real question.

**Disposition: RS-2 stops being a phase-exit gate.**

## Where the lenses agreed on deletion

Nine of eleven recommended deleting most of **FR-7**.  The arguments differ and converge:

- It is the only theme with **no failure mode at all**, loud or silent — which the release's own
  scope predicate is supposed to exclude.
- A ruling costs the same to write at 0.16.0 as today.
- FR-7.5 alone imported a PII gate, a `06_docs/mutants` relocation hazard, and OQ-8 — and its
  premise is wrong anyway (C-3).

Also repeatedly named for deletion: **FR-3.4** (blocked, and mis-instrumented — C-6), **FR-6.6**
(an unbounded hunt), and **FR-4.5** (the emergency-broadcast head/tail — the one FR-4 item that
*constructs* a new forgeable artifact rather than repairing an instrument that lies, landing on the
modality with no screen to disambiguate it).

## What survived adversarial check

Worth recording, because a red team that only lists failures is as useless as one that finds none.

- **FR-4.1 is correct and is the report's best finding.**  Four lenses independently confirmed the
  `"emergency"` scenario injects `"Tornado Warning"`, byte-identical to `default`, with
  `Until: now.Add(time.Hour)`.
- **FR-5 is correct and understated.**  Confirmed at three widths.
- **The six config writers** are six.
- **F-30's two mechanisms** are in the source exactly as the row claims.
- **The FR-3.3 self-correction** was called creditable by three lenses.

## Ledger staleness — worse than the report claimed

The report headlined *"the ledger lost three times out of seven."*  Two lenses attacked the
arithmetic and both were right: the denominator included an uncontested row and a substituted one,
and 3-of-7 is *winning* more than losing.  Sampling outside the seven then found more:

| Row | State |
|---|---|
| F-8 | **Already fixed.**  `httpx/memo.go:213` compares host *and* port |
| F-10 | **Fixed in 0.14.0** (`1abf27d`), carried open through two releases |
| F-1 | Says "four siblings"; the report proved five and did not correct the row |

All three corrected.  The honest statement is not a score — it is that **~40 rows have not been
audited since 0.14.0**, and every row sampled since has been stale.

## Conditions for GO

| # | Condition | State |
|---|---|---|
| 1 | R4 reinstated or deferred in writing; issue #14 reconciled | **Discharged** — FR-8 |
| 2 | The four-classifier cross-table produced; FR-2.2 rewritten to cover all four | **Open** — PLAN entry task |
| 3 | HUM LEAD ruling on listener-vs-operator scope, and on F-40's deferral with an exposure statement and a date | **Open — HUM LEAD** |
| 4 | Risk table gains probability × impact; #17 gets a disposition | **Discharged** — FR-9 |
| 5 | Metrics D, K, T each acquire an owning requirement or a written deletion | **Discharged** |
| 6 | The lane guard fails for every lane it does not cover | **Discharged** — rewritten, three failure modes observed |
| 7 | RS-2 unblocked; FR-3.4 re-instrumented | **Discharged** |

## Cost

Eleven lenses, ~1.09M subagent tokens, 284 tool calls, ~5 minutes wall-clock in parallel.

**Cost per defect:** seven Criticals, three of them defects in the reviewing artifact itself that no
amount of re-reading by the author had surfaced — including one in code shipped that afternoon.  The
blind-dispatch design is what produced the ranking: seven lenses reaching the same Critical without
coordination is a stronger signal than any single lens's severity label.
