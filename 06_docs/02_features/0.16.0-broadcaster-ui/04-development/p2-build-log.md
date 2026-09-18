---
title: "0.16.0 — P2 build log: station state and the swap gate"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "P2 COMPLETE.  Five commits.  Whole tree green, pty-severe green, gates GATED not chained."
---

# P2 — station state and the swap gate

## What it delivered

| Requirement | Status |
|---|---|
| **FR-5.1** the state is the Director's, not a local flag | **DONE** |
| **FR-5.3** legible without colour | **DONE** |
| **FR-5.4** the toggle is a rebindable action | **DONE** |
| **FR-5.5** the "on the air" boundary stated to the operator | **DONE** — in the frame, not a document |
| **FR-1.4** STANDBY before a swap away | **DONE** — the gate decides on real power |
| **FR-1.5 / 1.6** rebindable, and the chord is not the only door | **DONE** |
| **NFR-7** a bounded silent station | **DONE** |
| `lineup.Publish` consumed | **DONE** — its first reader |

## The four things worth carrying forward

**1. A surviving plant indicted the GATE, not itself — the first time this release.**  Making every
station message restart the standby clock failed nothing, because every fixture sent exactly one
message.  What it would have shipped is worse than no notice at all: a station silent for an hour looks
freshly quiet, and **the escalation never climbs past its first rung** while the operator trusts it.
The rule says check the plant first; it also says check the gate, and this is why.

**2. Two guards for two failure shapes, deliberately.**  The single-reader AST guard counts who reads
the power.  It would **not** catch a binding that swapped surfaces without reading the power at all —
so the key-path test exists beside it.  A guard that counts readers is blind to a path that reads
nothing.

**3. My own `--ascii` gate had a hole I put there.**  A non-ASCII separator in the ON AIR banner passed,
because the gate's fixture left the station STOPPED and only the ON AIR line carries a separator.
**A single fixture is a hand-written subject list of size one** — INST-1's mistake, made inside a test
written to honour INST-1.  Closed by asking `Power.String()` where the set ends.

**4. An existing guard caught a field this release added.**  The executors' seam check walks the struct
by reflection; `publish` failed it at once and had to be declared optional with a reason.  **Written by
an earlier release, catching a later one, with no edit to the guard.**

## Where the process failed, and what changed

**P2(a) chained the test run and the commit with `;` and pushed a RED gate to `origin`.**  The
allocation budget had fired correctly — the banner costs one extra allocation — and the commit ran
anyway.  That is *Verify-Then-Commit* named exactly, and it was the second instance in the session:
earlier an `echo pushed` followed a push whose exit code had been discarded and claimed success for a
push that had failed.

**Every commit from P2(b) onward is GATED**: `if ! go test; then exit 1`.  Both failures are recorded
in the roster beside the catches, because a roster that only shows catches teaches the easy half.

## Numbers

| | |
|---|---|
| Gates on the roster after P2 | **31**, each with a watched failure |
| Console frame budget | **14 allocations** (re-pinned from 13; the banner) |
| Plants this batch | **15 CAUGHT · 1 survived and found a blind gate · 1 invalid (removed-a-use)** |

## Next

**P3 — the main-track producer.**  Re-planned from S0's answer: the rotation's tune is **wiring**; the
**two relay-failure fallbacks are the real work**; the no-double-speak property is **asserted, not
built**, because the arbiter already provides it.  It carries its own red team, a UAT shared with
nothing, a timing property test, a dark path, and a go/no-go before P4 and P5.
