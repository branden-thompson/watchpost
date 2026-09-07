---
title: "0.15.0 — Pre-Broadcaster UI Improvements — IMPLEMENTATION PLAN"
date: 2026-09-07
phase: PLAN → BUILD
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
status: "Batches defined with entry and exit conditions — task decomposition happens at each batch's entry"
---

# Implementation plan — BUILD batches

**Task decomposition happens at each batch's entry, against the code as it then exists** (PL-D-6).
Writing several hundred 2–5 minute tasks now would be several hundred lines of untested code in a
document, which the HUM LEAD rule forbids and which 0.14.0 did not do either — it ran P1–P4 with
per-batch task lists written at batch entry.  This document defines the batches, their order, what
each must be true to start and to finish, and the one task each opens with.

**Every batch follows TDD** (directive-mandated): RED — a failing test that has been *watched* to
fail — then GREEN, then the gate suite.  A test nobody has seen fail is not evidence.

**Every batch pushes its branch on first commit** (NFR-4).  The single biggest finding of 0.14.0 was
that the release ran on Linux for the first time in its own release PR.

---

## B1 · Mechanism — `platform/closedset`

**Lands:** FR-2.1, FR-2.3 · **Size:** S · **Depends on:** nothing

**Entry conditions**
- FR-10's timebox is set and written down (it is the release's only unsized item).

**Opening task** — migrate `TestEveryFeedLaneSurvivesTheMarqueeMap` onto the new helper, as its
**first** call site.  This is deliberate and is the batch's real test: the helper is derived from
that check, so if it cannot express it, the approach has failed cheaply, before three more sites
depend on it (PL-D-1's revisit condition).

**Exit conditions**
- `closedset.Check` fails, observed, in all three of its modes: a member neither carried nor
  declared; a fixture that does not produce its member; a consumer that disagrees.
- The lane guard passes through the helper with no loss of coverage — still 5 carried, 2 declared.
- `make verify` and `make test-platforms` green.

---

## B2 · Population — the triage, and its two first customers

**Lands:** FR-2.1a, FR-2.2, FR-1.2, #18's tone mapping · **Size:** L · **Depends on:** B1

**Entry conditions**
- B1's helper exists and the lane guard runs through it.

**Opening task** — classify all 27 closed sets and 50 `default:` arms into three buckets:
*closed-set consumer needing a check* · *open set, a default is correct* · *decision stated, no check
owed*.  This is a reading pass and its output is the artifact; it is not a coding task.

**Then, falling directly out of it:**
- The four-classifier coupling check over `severeEvents()` (FR-2.2), using the cross-table at
  `app/classifier_crosstable_test.go` as its fixture.
- **#18's tone mapping** — every `category.Category` that can reach a read maps to exactly one
  `cast.Class`, declared in one place, with a category having no mapping failing rather than falling
  through to `ClassWarning`.  **The three-dual-tone signal for an evacuation order lands here**, not
  as a separate item, because the mapping is what makes it impossible to lose again.

**Exit conditions**
- The triage is committed as a table, with a count per bucket.
- Every bucket-one member has a check or a written reason.
- #18's tone is 3 × `PresetDualTone`, and a category without a tone mapping fails a test.
- Gates green.

---

## B3 · Operator surface — usable before honest

**Lands:** FR-5 then FR-4 · **Size:** M · **Depends on:** B2 *(FR-4.2's scenarios consume the
mapping)*

**Entry conditions**
- B2's triage is done, so FR-4.2's "one scenario per category" has a defined set to derive from.
- **NFR-2's artifact-level absence check is designed** — not built, designed.  If it is not, FR-4
  does not land in this batch (RS-3).

**Order within the batch is fixed: FR-5 before FR-4.**  Adding scenarios to a window whose controls
cannot be seen produces an instrument nobody can operate, which is how this release started.

**Opening task** — a failing test that renders `modalDebug` at 80×24 and asserts the scenario labels
and the `›` cursor are visible.  It fails today; the probe measured it at three widths.

**Exit conditions**
- The window's controls are visible and reachable at 24 rows at every supported width; prose wraps
  once.
- The `"emergency"` scenario injects an `Evacuation Immediate`, and every scenario's payload matches
  its label.
- Test events expire within two minutes, and their persisted footprint in `seen.json` is bounded or
  self-identifying.
- `*** TEST EVENT ***` marking is per-line, not head-and-tail only — a listener tuning in mid-read
  must not hear an unmarked fabricated warning.
- NFR-2's check runs against a built artifact and has been observed failing against a debug build.
- Gates green.

### ▸ CHECKPOINT — after B3

**The plan's only scope lever.**  B1–B3 are the batches whose sizes are best understood.  If they
overran, the release is re-cut here, with the HUM LEAD, rather than at B7 when it is too late.
RS-1 has no other mitigation, because descoping was declined.

---

## B4 · Consumers — the memo guards

**Lands:** FR-3 · **Size:** M · **Depends on:** checkpoint · **Parallel with:** B5

**Opening task** — make `memo_completeness_test.go` **fail** on an unfixtured window instead of
`t.Skipf`, and watch it fail by removing a fixture.

**Exit conditions**
- An unfixtured window fails; exemptions are declared rows with written reasons.
- The nested-struct set is derived, not the three hard-coded names.
- `bodyKey` has an equivalent guard.  **It does not transfer as-is:** `bodyKeyFor` derives ten fields
  from `frameLayout` and four by computation, and `perturbations` walks a `Dashboard` — so the guard
  must perturb the layout and the derived sources too, or it will report coverage it does not have.
- `tileMemo`/`boxMemo` consolidation measured with `make alloc-budget` and the memo gauges — **not
  a frame pin**, which cannot see a provider-fetch change (FR-3.4).
- FR-3.2's ruling on the six data-cache memos is written (OQ-9).
- Gates green.

---

## B5 · Listener safety

**Lands:** FR-9 · **Size:** M · **Depends on:** checkpoint · **Parallel with:** B4

**Opening task** — eliminate the test double as #17's cause.  `fakePlayer`'s drain goroutine and
`Play()` are unordered; if the drain reaches EOF first, `Play()` sets `playing` true afterwards with
nothing to clear it, producing the reported status exactly.  **Cheap disproof before any hardware
time.**

**Exit conditions**
- A read completes, fails, or reports a fault within a stated bound, and the bound lives in
  `player.Engine.watch` (PL-D-5).
- A sounded tone with no words produces a user-perceivable fault, with its diagnostic **on by
  default** — today `read:gaveup:after-tone` needs `WATCHPOST_DEBUG_RADIO`, so F-43's own closing
  condition is unreachable in normal use.
- A broadcast cycle yielding nothing to say is reported as a fault (FR-9.3).
- #17 has a written disposition: cause found, or the test double eliminated and the failure still
  open with what is known.
- Gates green.

---

## B6 · Gates that can fail

**Lands:** FR-8 · **Size:** L · **Depends on:** B4, B5

**Opening task** — run `golangci-lint` and `staticcheck` once and **count**.  The number is unknown
and it determines the batch's shape.

**Exit conditions**
- `make lint` exists and is in `verify`, landing as **baseline + no-new-findings ratchet** (PL-D-7).
  A clean bill is not an exit condition.
- One completeness test covers the AA register and `--ascii`.  Eleven declared tokens sit outside
  `aaPairs()` today, and `TestEveryPaintedPairReadsAAInEveryTheme` iterates that hand-list — a test
  whose name asserts a completeness it cannot deliver.  **Family B: the ruling applies.**
- The PTY journey's read step establishes its own precondition.
- **`ci.yml` runs what `make verify` runs.**  It omits `vet-tags` and `mutant-check` today, so metric
  **G** is tracked against a gate set no second machine reproduces.
- Every gate in `gates.md` carries an evidence line naming a **watched** failure.  This is metric G's
  baseline and it can only be produced here.
- Gates green.

---

## B7 · Defects and rulings

**Lands:** FR-6, FR-7 · **Size:** M · **Depends on:** B6

Six independent defects and five rulings; low coupling, high count.  Ordered cheapest-first, except
FR-6.4 (the WCAG 2.2.1 Level A timeout) which leads because it is a conformance failure on the only
station-tuning control.

**Exit conditions**
- FR-6.1 … FR-6.6 landed or dispositioned in writing.
- FR-7.1 … FR-7.5 ruled, and the rulings written where the decision lives, not only in a report.
- FR-7.5 produces an **exposure statement** — the PII is already published, so the deliverable is a
  categorised scan plus a HUM LEAD remediation ruling, not a gate against a door that is open.
- Gates green.

---

## FR-10 · Feed investigation — parallel throughout

**Size:** unsized; **timeboxed at B1 entry** · **Depends on:** nothing · **Contends with:** nothing

The only item touching no file another batch touches.  It investigates external feeds, so it runs
from day one alongside B1.

**Its own DISCOVER, bounded** (HUM LEAD 2026-09-06): the feeds get examined rather than the reader.

- **FR-10.1** why no FIRMS hotspot for the Brengel Fire — radius, confidence filter, satellite-pass
  latency, or absent from the feed.  Eliminate, do not guess.
- **FR-10.2** whether a county or CAL FIRE evacuation can reach the NWS alerts feed at all.
- **FR-10.3** the boundary is stated to the listener and the operator.  **This ships regardless of
  what 10.1 and 10.2 conclude** — it is the requirement that made the roll-in necessary.

**Exit condition:** a written disposition inside the timebox, whether or not a cause is found.

---

## Ordering summary

```
B1 ──► B2 ──► B3 ──►[CHECKPOINT]──► B4 ──┐
                                    B5 ──┴──► B6 ──► B7

FR-10 ═══════════════════════════════════════════════►  (parallel, timeboxed)
```

**Critical path:** B1 → B2 → B3 → (B4 ∥ B5) → B6 → B7
**Only parallelism:** B4 with B5, after the checkpoint; FR-10 throughout.
**Only scope lever:** the checkpoint after B3.
