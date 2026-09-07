---
title: "0.15.0 — Pre-Broadcaster UI Improvements — IMPLEMENTATION PLAN"
date: 2026-09-07
phase: PLAN → BUILD
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
revision: "2 — re-cut after the PLAN-exit red team returned NO-GO (red-team-plan.md)"
status: "Re-cut against eight verified Criticals — awaiting HUM LEAD approval to enter BUILD"
---

# Implementation plan — BUILD batches

**Revision 2.**  Revision 1 was returned NO-GO by an eleven-lens red team.  Eight Criticals, every
one a verifiable fact about the codebase that revision 1 asserted wrongly.  What changed, and why,
is in `08-reports/red-team-plan.md` and in the plan report's deviation table.

## The gate set, named once

Every batch exits on **`make verify` + `make alloc-budget` + `make test-platforms`**, and on the
batch's own conditions below.

Revision 1 wrote "Gates green" seven times and named the set once.  That was vacuous for allocations:
`make verify` (`Makefile:172`) does **not** include `alloc-budget`, and under `-race` the pins skip
themselves.  **B6 lands `alloc-budget` into `verify` so this stops being a discipline and becomes a
target** — until then, batches run it explicitly.

**Task decomposition happens at each batch's entry**, against the code as it then exists.  **Every
batch follows TDD**: a failing test that has been *watched* to fail, then the implementation, then
the gate set.  **Every batch pushes on first commit** (NFR-4).

---

## B1 · Population, classifiers, and the evacuation tone

**Lands:** FR-2.1a, FR-2.2, FR-2.3, FR-1.2, #18 · **Size:** L · **Depends on:** nothing

*Was B2.  It is now first, because `closedset` is deferred (PL-D-8) and the mechanism batch that
preceded it no longer exists.*

**Entry conditions**
- The **exclusion rule** for the closed-set survey is written down before counting begins:
  non-test, non-`third_party/`, non-`tools/`.  Revision 1 asserted "27 sets and 50 default arms"
  without one, and the count swings ~30% depending on whether the vendored go-studs patch stack is in
  scope.  A number with no stated boundary is the defect this release is about.
- FR-10's timebox is set and written down.

**Opening task** — the survey.  Classify every closed set and every `default:` arm into
*closed-set consumer needing a check* · *open set, a default is correct* · *decision stated, no check
owed*.  A reading pass; its output is a committed table with a count per bucket.

**Then, falling out of it:**
- The four-classifier coupling check.  **The work is one edit, not a new instrument:**
  `app/classifier_crosstable_test.go` already pins the divergence count at 7 across all four
  classifiers — its weakness is that its `products` list is hand-written.  Derive that list from
  `severeEvents()` and the coupling check exists.
- **#18's evacuation tone.**  Three repetitions of `PresetDualTone`, and the `category.Category` →
  `cast.Class` mapping declared in one place, with an unmapped category failing rather than falling
  through to `ClassWarning`.

**Exit conditions**
- The survey table is committed with its exclusion rule and a count per bucket.
- Every bucket-one member has a check or a written reason.
- An `Evacuation Immediate` sounds 3 × `PresetDualTone`; a category with no tone mapping fails a test
  that has been watched to fail.
- Gate set green.

**▸ The extraction decision happens here, at the end of B1.**  The tone map is the **second**
hand-written instance of "every member is carried or declared."  Put it beside the lane guard and
look: if they rhyme, extract `platform/closedset` then.  If they do not, record why and stop.  *This
is the standing rule — extract at the second caller — applied to two real call sites instead of the
five revision 1 counted, four of which did not exist.*

---

## B2 · Single ownership

**Lands:** FR-1.1, FR-1.3, FR-1.4 · **Size:** M · **Depends on:** B1 *(the survey tells FR-1.3 what
enforcement mechanisms the codebase already uses)*

***This batch did not exist in revision 1.***  FR-1.1, FR-1.3 and FR-1.4 had no batch at all while
the internal validation certified "10 of 10" — the release's first named scope item, designed in the
plan report and scheduled nowhere.  Four lenses found it independently.

**Entry conditions**
- The `edit` callback's contract is settled: `edit func(*Config) error`, so `applySetup`'s validation
  and `commit`'s `invariant.Check` can abort inside the seam.  Revision 1's `func(*Config)` could not.

**Opening task** — a failing test asserting that `config.Save` has exactly one caller.

**Exit conditions**
- **FR-1.1** — one serialized write path.  Three specifics revision 1 left open, each of which BUILD
  would otherwise decide by accident:
  - `saveCast` needs the computed value **out** of the transaction for `deck.setCast`; the signature
    must carry it or the caller smuggles it through a closure.
  - `commit` holds `lp.mu` across its Load/Save, so the lock order `lp.mu` → config is fixed and
    written down, and `reloadCast`'s bare `config.Load()` must never appear inside an `edit` closure —
    the mutex is not reentrant.
  - The mutex is process-local.  `Save` is atomic rename, so this closes a lost-update window, not a
    corruption one.  Say so; two instances on one machine still last-write-wins.
  - `edit` runs under the lock, so a panic inside it must not strand the mutex: `defer`.
- **FR-1.3** — the band's single-writer rule is enforced by a **mechanism**.  Note the tension
  revision 1 missed: FR-1.1's own "one write path" is a doc comment while `config.Save` stays
  exported.  Whatever mechanism FR-1.3 chooses (unexported behind an internal, or a `lint-imports`
  style gate) applies to FR-1.1 too, or the release demands of the band what it does not do itself.
- **FR-1.4** — the arbiter's hold/resume/drop serialised against duck/restore.  **The documented lock
  order is `d.mu` → `m.mu`, and inverting it deadlocks** (`mastercontrol.go:165`, `director.go:640`).
  A wrapper mutex on those three methods is the change most likely to invert it, so the atomicity unit
  must be named — which sequence of voice calls must not interleave — and a reachable interleaving
  demonstrated under `-race` before the lock is added.
- Gate set green.

---

## B3 · Operator surface

**Lands:** FR-5, FR-4, FR-6.4 · **Size:** M · **Depends on:** B1

**Entry conditions — one of these is new and blocking**
- **A `test-tags` target exists and is in `verify`: `go test -tags watchpost_debug -count=1 ./...`.**
  Revision 1 put this in B6, three batches *after* the work that needs it.  Today `vet-tags` runs
  `go vet` — it **vets, it does not test** — so no gate in this repository has ever executed a
  debug-tagged test.  Every FR-4 assertion would have landed in a build nothing runs.  The Makefile
  already records this happening once: *"`inject_seam_test.go` — the test the whole injector stands
  on — stopped compiling at T3.10b and stayed dark."*
- **NFR-2's predicate is written down**, and it is **symbol-level, not string-level**.
  `modes/tty/debug.go` has no build tag, so its UI literals — `"INJECT AN ALERT:"` — compile into
  every release binary and a string grep fails a clean build.  The predicate is `go tool nm` over each
  `dist/watchpost-*` asserting zero `app.(*tickerDeck).Inject` and zero `debugScenarios` body symbol.
  It runs as a **`release-matrix` post-step**, not in `verify`, because `verify` runs before the
  published artifacts exist.
- **FR-4.4 must not remove `Location: "Injected Test Location"`** until NFR-2's replacement predicate
  is landed and observed.  F-38 proposes grepping artifacts for that literal; renaming it first
  silently retires the only gate proposal that exists.

**Order within the batch is fixed: FR-5 before FR-4.**  Adding scenarios to a window whose controls
cannot be seen produces an instrument nobody can operate.

**FR-6.4 joins this batch** — the relay-fault window's 10-second auto-close, a **WCAG 2.2.1 Level A**
failure on the only station-tuning control.  Revision 1 put it last, behind the only scope lever, in
the slot a re-cut removes first.  It touches one constant and one nav handler, contends with nothing,
and is surface work.  *(Deviation PD-6.  `handleRelayFaultNav` touches neither `.last` nor `.left`,
so 2.2.1's "extend" exception is architecturally absent, not merely unimplemented — OQ-2 must be ruled
before the fix, since "how long" answers none of 2.2.1's three mechanisms.)*

**Exit conditions**
- FR-5 is a **set-level** property, not one window: at 80×24 every modal's focused row and footer keys
  are on screen, or the modal is declared with a reason.  Ten of eleven windows have no assertion
  below 44 rows today; fixing only `ctrl+d` leaves the defect in the other nine.
- The `"emergency"` scenario injects an `Evacuation Immediate`; every payload matches its label.
- Test events expire within two minutes, and their persisted footprint in `seen.json` is bounded or
  self-identifying — `tickerSeenWindow` is 7 days and FR-4.3's bound does not reach it.
- **A real arrival pre-empts a test event, and fabricated events never consume the burst Max.**  The
  `burst` scenario injects six events against `defaultBurstMax = 5`, so an operator testing at the
  wrong moment can push a real hazard out of the read budget.
- `*** TEST EVENT ***` marking is per-line for the spoken read; **for the marquee it is lane chrome,
  not tape text** — the tape is one scrolling line, so an in-tape marker is off-window most of the
  time, and an 18-cell prefix per item at the 80-column floor makes the marker the majority of the
  tape.
- **`advanceTicker` stays within its 71-alloc budget with a marked-event fixture present.**  It runs
  at 68 today; `benchTicker` contains no test events, so the existing pin cannot see this change.
- NFR-2's check runs against built artifacts, has been observed **failing** against a debug build
  **and passing** against a clean one.
- Gate set green.

### ▸ CHECKPOINT — after B3

**Trip condition, stated:** B1–B3 not complete by the end of the third working session.

**What it may do, given that descope was declined:** it may **re-order** and it may **defer within the
release**, in this pre-ranked order — FR-7.4 (attribution stamps), FR-7.3 (ledger ratification),
FR-6.2, FR-6.3.  It may not cut FR-1, FR-2, FR-5, FR-9 or FR-10.3.

Revision 1 said only "re-plan if B1–B3 overran," with no trip condition and a stroke that had already
been ruled out.  A lever with no trigger is a calendar entry, and RS-1 was recorded as mitigated by it.

---

## B4 · Consumers — the memo guards

**Lands:** FR-3 · **Size:** L *(was M)* · **Depends on:** checkpoint · **Parallel with:** B5

**Entry conditions**
- **A provider-package allocation pin exists and has been watched to fail.**  Revision 1 named
  `make alloc-budget` as this batch's instrument; all four of its tests are frame pins in `modes/tty`
  and there is no pin in `domains/fire/firms` or `domains/seismic/usgs` where the memos live.  **The
  instrument must be built before the change it measures**, or B4 ships a green frame pin as evidence
  for something the frame cannot see.
- OQ-9 is ruled: what property a data-cache memo owes.

**Opening task** — *not* flipping `t.Skipf` to `t.Error`.  That branch fires **zero times** today
(11/11, no skips), so the flip is a ratchet that falsifies nothing.  The opening task is the `bodyKey`
perturbation design, which is this batch's real risk.

**Exit conditions**
- `bodyKey` has a guard, and it perturbs what reflection cannot reach: **`theme` is
  `render.ThemeGeneration()`, a package global with no `Dashboard` preimage**, and `frameLayout` is
  *derived* by `d.layout()`, so its fields have no independent setter.  A guard copied from `modalKey`
  reports coverage of fields it cannot touch.  *(Note the trap: perturbing a `frameLayout` directly
  manufactures states the app cannot reach — the very thing `memo_completeness_test.go` warns against.
  Perturb the `Dashboard`, re-derive the layout, compare.)*
- The unfixtured-member ruling is written **once**, in one place, and both B4 and B6 cite it rather
  than each encoding it.
- `tileMemo`/`boxMemo` consolidation measured on the **provider fetch path**, against the pin built at
  entry, with a stated threshold.  A measurement with no threshold cannot fail.
- **K's denominator is fixed before FR-3.2's ruling lands**, so the ruling cannot zero the metric by
  redefining the set.
- Gate set green.

---

## B5 · Listener safety

**Lands:** FR-9, FR-6.5 · **Size:** L *(was M)* · **Depends on:** checkpoint · **Parallel with:** B4

**FR-6.5 moves here from B7** — making the relay-fault window audible is a listener requirement, and
DISCOVER's own taxonomy says so.  *(Deviation PD-7.)*

**Entry conditions**
- **PL-D-5 is withdrawn and the bound is re-sited.**  Revision 1 put it in `player.Engine.watch`.
  `watch` has one caller — `playPCM` — reached from the **relay stream** and `StartSource`.  A spoken
  read goes `Preview`/`PreviewAside` → `playClip`, which **already carries a 10-minute P10-02 bound**.
  A bound in `watch` would time the radio stream, not the read, and would abort a station stream that
  has played for hours.
- **A completion-signal spike, timeboxed to one hour**, threading one completion event out of
  `playClip` to the Director.  The read's duration is open-loop today: `director.go` returns a
  computed `pcmDuration` and `speaker.hold` sleeps it — **nothing observes that a read finished.**
  The spike settles both the bound's home and this batch's size.

**Opening task** — eliminate the test double as #17's cause.  `fakePlayer`'s drain goroutine and
`Play()` are unordered; if the drain reaches EOF first, `Play()` stores `true` afterwards with nothing
to clear it, reproducing the reported status exactly.  **Cheap disproof before any hardware time.**

**Exit conditions**
- A read completes, fails, or reports a fault within a stated bound.  **The bound is held-excluded
  elapsed time and is never applied to the live stream** — `watch` pauses the player entirely when a
  report gives way to an alert, so a naive wall-clock deadline kills the read that ducking exists to
  protect.  The number is derived from expected script length plus Piper's measured ~10 s
  per-utterance load, and both the number and its derivation are written down.
- **The fail-safe direction is stated and tested:** abandonment cancels the job's context so the
  existing `Failed{Routed:true}` path advances the schedule and `restore()` runs.  Without it the bed
  stays ducked, the arbiter stays occupied, and the reader speaks into a closed player — a silent
  station produced by the safety feature.
- **FR-9.2** — a sounded tone with no words produces a user-perceivable fault.  The **fault** is a UI
  obligation and lands with FR-6.5's window.  The **diagnostic** stays opt-in behind
  `WATCHPOST_DEBUG_RADIO`, hardened `debugAddr()`-style: a fixed, size-capped, rotated 0600 file under
  the cache root, with the env var selecting a validated name rather than an arbitrary path.
  *Revision 1 said "on by default," which would have made an unvalidated append-anywhere file write
  the default on a 24/7 process.*
  The fault fires only when **no part could be rendered and `ctx.Err() == nil`** — an operator pressing
  `esc`, a takeover pre-empting, or the pump stopping are all legitimate tone-without-words today.
- **FR-9.3, reworded (HUM LEAD, 2026-09-07):** *the main-track rotation produces a read within a
  stated interval, or it reports* — **unless the operator has chosen silence.**  Dead air the operator
  chose is not a fault.  This leaves ruling **I-2** intact: a deliberate non-delivery is not a fault,
  and neither is deliberate silence.  Revision 1's wording — "a cycle that yields nothing to say is a
  fault" — reversed I-2 by accident and would have raised the fault window on a quiet day, since the
  Director declines every non-alert cycle by design.  *The masthead / mastTail feature (0.16.0,
  Broadcaster) is the downstream surface that makes chosen dead air legible; FR-9.3 is what tells it
  the difference.*
- #17 has a written disposition.
- Gate set green.

---

## B6 · Gates that can fail

**Lands:** FR-8 · **Size:** L · **Depends on:** B4, B5

**Opening task** — run `golangci-lint` and `staticcheck` once and **count**.

**Exit conditions**
- `make lint` exists and is in `verify`, as **baseline + no-new-findings ratchet**.  A clean bill is
  not an exit condition.
- **`alloc-budget` joins `verify`.**
- **`ci.yml` and `verify` run the same set, in both directions.**  Revision 1 stated this
  one-directionally; `verify` omits `vet-tags` and `mutant-check` from CI's view, but CI *also* runs
  `alloc-budget`, `release-matrix` and `install-test` which `verify` omits — so a naive convergence
  **deletes the repo's only automated allocation gate.**
- One completeness test covers the AA register and `--ascii`.  Two corrections to revision 1:
  **`--ascii` misses ten windows, not four** (the golden scan covers `dashboard` and `settings` only,
  against eleven modals); and **the AA gate is currently a tautology** — `withAA` lifts every pair in
  `aaPairs()` and the test iterates the same list, so it can only fail if the lifter fails to
  converge.  The producer is the set of pairs actually *painted at render call sites*, not the
  register.
- The PTY journey's read step establishes its own precondition.
- Every gate in a **single** `gates.md` carries an evidence line naming a **watched** failure.
  Creating that roster is a task in this batch, not an assumption — no 0.15.0 `gates.md` exists.
- Gate set green.

---

## B7 · Remaining defects and rulings

**Lands:** FR-6.1, FR-6.2, FR-6.3, FR-6.6, FR-7 · **Size:** M · **Depends on:** B6

*(FR-6.4 moved to B3, FR-6.5 to B5.)*

**Exit conditions**
- FR-6.1, 6.2, 6.3 landed.  **FR-6.6 (F-43) carries a timebox set at entry and a written disposition**
  — the "or dispositioned in writing" escape is removed from the others.
- FR-7.1 … FR-7.4 ruled, and the rulings written **where the decision lives**, not only in a report.
- **FR-7.5 produces an exposure statement covering the tree, the git history, the published tags, the
  README and its images, and `dist/watchpost-*`.**  Revision 1 scoped it to `06_docs`.  The categories
  are enumerated — identity, location, host, credential, internal URL, path, **artifact, image** — and
  the count is re-run with its scope named.  A scan that never looks at a built artifact or a
  published tag is scoped wrong.
- Gate set green.

---

## FR-10 · Feed investigation

**Size:** unsized; timeboxed at B1 entry · **Depends on:** nothing

- **FR-10.1** why no FIRMS hotspot for the Brengel Fire — radius, confidence filter, satellite-pass
  latency, or absent from the feed.  Eliminate, do not guess.  **Genuinely parallel.**
- **FR-10.2** whether a county or CAL FIRE evacuation can reach the NWS alerts feed at all.
  **Genuinely parallel.**
- **FR-10.3 — the boundary is stated to the listener and the operator.**  ***NOT parallel.***  It is
  UI text and read-script text, in the same files B3 and B5 touch, and no coverage-boundary surface
  exists to extend.  **It lands in B5**, with the listener work.  *(Deviation PD-8.  Revision 1's
  claim that FR-10 "contends with nothing for code" was true of 10.1 and 10.2 and false of 10.3.)*
- **A boundary statement can itself mislead**, and the design must say how it avoids it: a static
  disclosure cannot distinguish *"we do not watch this"* from *"we watch it and it has been failing
  for eight days"* — which is #7's exact shape.  FR-10.3 states coverage **and current feed health**.

**Exit condition:** a written disposition inside the timebox, whether or not a cause is found.

---

## Ordering

```
B1 ──► B2 ──► B3 ──►[CHECKPOINT]──► B4 ──┐
  │                                 B5 ──┴──► B6 ──► B7
  └─► FR-10.1/10.2 ═══════════════════►  (parallel, timeboxed)
                     FR-10.3 ─────────► lands in B5
```

**Critical path:** B1 → B2 → B3 → (B4 ∥ B5) → B6 → B7
**Parallel:** B4 with B5 after the checkpoint; FR-10.1/10.2 throughout.
**Scope lever:** the checkpoint after B3, with the trip condition and pre-ranked deferral list above.
**Deferred, conditional:** `platform/closedset`, extracted at the end of B1 if the tone map and the
lane guard rhyme.
