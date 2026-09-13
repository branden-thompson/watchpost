---
title: "0.16.0 — the architecture-conformance audit"
date: 2026-09-13
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "COMPLETE.  One live defect found and fixed (D-125); two notes recorded."
---

# The architecture-conformance audit

**Scope: NARROW, ratified by the HUM LEAD 2026-09-13** — verify the stated rules hold, and read the
seams 0.16.0 added.  A broad read (are the abstractions themselves right?) was offered and declined:
it re-opens design the HUM LEAD has already ruled on, in the last step before BUILD exit.

## The rules, and whether they hold

| Rule | Result |
|---|---|
| `modes/` may not import `domains/` | **holds** — none.  Gated by `lint-imports` |
| `platform/` is a leaf: no `domains/`, no `modes/` | **holds** — none, including the one-hop hole the gate closes |
| `modes/` may not import `app/` | **holds** — none |
| A domain owns its own business | **holds** — ONE cross-domain edge in production, and it is documented |
| Shared things live in `platform/` | **holds**, with one note |
| Every `ForTest` export is in `platform/` | **holds** — and was restored by D-124 the same day |
| Every `tty.Config` func seam is classified for the air (D-91) | **holds** — gated, and the gate is derived by reflection rather than listed |
| Every closed-set member names a writer and a reader, or is RATIFIED | **holds** — 84 members, 2 ratified, 0 unexplained |

### The one cross-domain edge is deliberate

`domains/severe` imports `domains/globalfeed` — `Event`, `Class`, `Severity` and the per-class
details.  That is a CONSUMER relationship and it is written down at `severe.go:123`: *"THE TABLE MOVED
TO globalfeed AND IS NO LONGER OURS (C-2, D-1) … this package is the one that imports globalfeed."*
globalfeed owns the hazard taxonomy; severe reads it.  Not a reach.

`domains/alerts` → `domains/weather` exists **only in a test file** (`replay_nws_test.go`).  No
production edge.

### `platform/`, package by package

Three packages have no production consumer at all — `closedset`, `declset`, `singleowner` — and that
is **correct rather than suspicious**: they are quality-gate infrastructure used by tests in `app`,
`modes/tty`, `platform/render` and `domains/weather`.  A helper five packages share cannot live in any
one of them.

**NOTE (minor, no action):** `platform/astro` has exactly one consumer, `platform/snapshot`.  It is
domain-neutral maths and could equally live inside `snapshot`.  Recorded rather than moved — the rule
is about where SHARED things go, not a prohibition on a neutral leaf, and moving it buys nothing.

`platform/config` and `platform/sched` are single-consumer in production (`app`), which is by design:
**no package under `modes/` reads storage**, and D-124 is the ruling that kept it that way.

## The seams 0.16.0 added

### D-125 — a live defect, found here, fixed

**`BedMsg` had three publishers and exactly one of them set `Relays`.**

| Publisher | What it sends |
|---|---|
| `app/bedrelay.go:125` — the resolver | `{Relay, Carrying, Relays: len(st)}` |
| `app/bedrelay.go:191` — the selector | `{Relay, Carrying}` |
| `app/executors.go:288` — the deck's state | `{Relay, Carrying}` |

The console stores the WHOLE message and `bedAvailable()` reads `!bedTold || bed.Relays > 0`.  So
either of the latter two **zeroed the count the resolver had established**, and the BED control
disabled itself while relays were streaming.

**D-117 FIRING BACKWARDS.**  That rule exists so the operator cannot choose something that will
broadcast dead air.  This told them there was nothing to choose while a relay was carrying — the same
wrong answer, in the direction nobody was watching for.

Fixed as **one fact, one message, one writer**: `BedRelaysMsg` is sent only by `setBedStations`.
Mutants mAT1 (the count rides on `BedMsg` again) and mAK3 (re-pointed) both CAUGHT.

### Every other message was checked for the same shape

All 22 console message types were swept for multiple writers.  Only two have more than one, and
neither is now a hazard:

- **`BedMsg`** — three publishers, and after D-125 **all three set both of its remaining fields.**
  The defect was never "two writers"; it was a field only one writer knew about.
- **`StationAreaMsg`** — `app/pool.go` is the only SENDER; `app/dashboard.go` sets the launch VALUE
  on `Config`, because the program's loop is not running when the pool is first derived (D-93).
  Different roles, both documented.

## Two records that lied about the code, corrected

Neither was a code defect, and both are the same class as the roster that stopped at P3.

1. **`modes/tty/setup.go` claimed "IN `platform/` BY THE ARCHITECTURE'S OWN RULE"** — from inside
   `modes/tty`.  It stated a rule and asserted compliance with it in the same breath, from the package
   the rule points away from.  It also claimed `modes/tty` "may not import `platform/config` (make
   lint-imports)", which is **false**: it compiles and the gate passes.  The real constraint is a
   convention nobody had written down, and D-124 wrote it down.

2. **`app/livesource.go` kept a CENSUS of `ForTest` exports** — "all four are in `platform/`".  D-115
   made it false (five, two outside).  I corrected the number, and **D-124 made my correction false
   within the hour** by moving those two back.  The count is gone now: a hand-kept census in a comment
   is a fact with two carriers, it has been wrong twice in both directions, and the rule it sits above
   does not need a number.

## The audit's own finding about audits

**The defect was found by asking a mechanical question — "which messages have more than one writer?"
— and then reading the two answers.**  It was not found by reading the code for correctness; the
code reads correctly at every one of the three publishers, and each is right about what it knows.
The defect lives in the SET, which no single file shows.
