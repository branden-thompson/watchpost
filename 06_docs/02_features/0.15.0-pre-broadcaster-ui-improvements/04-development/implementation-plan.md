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
- **NFR-2's predicate is written down** — ~~symbol-level, not string-level~~ **STRING-level, and the
  correction was found by measuring rather than reasoning (2026-09-07).**  Release artifacts are
  linked `-s -w`, which strips the symbol table: `go tool nm` reports *"no symbol section"* on linux
  and *"no symbols"* on windows, so a symbol predicate finds zero in a DEBUG matrix and zero in a
  clean one and passes everything.  Strings survive stripping; symbols do not.
  The anchor is **`watchpost-injected-`**, the id prefix `Inject` mints — 1/1/1 in a debug matrix,
  0/0/0 in a clean one, on every platform.  It is **not copy**, so FR-4.4 may rewrite every
  user-facing marking string without retiring the gate.  `"INJECT AN ALERT"` is unusable: it lives in
  the untagged `modes/tty/debug.go` and ships in every clean binary.  `debugScenarios` is unusable
  too: zero in both builds, because its release form inlines away.
  It runs as a **`release-matrix` post-step**, not in `verify`, because `verify` runs before the
  published artifacts exist.  *Landed as `scripts/lint-injector.sh`, self-tested against **stripped**
  builds in both directions, and validated end-to-end against a matrix built with the tag.*
- ~~**FR-4.4 must not remove `Location: "Injected Test Location"`**~~ **No longer a constraint.**  The
  gate anchors on the id prefix rather than on marking copy, so FR-4.4 is free.

**Order within the batch is fixed: FR-5 before FR-4.**  Adding scenarios to a window whose controls
cannot be seen produces an instrument nobody can operate.

**FR-6.4 joins this batch, as an IMPROVEMENT rather than a conformance gate** — HUM LEAD, 2026-09-07:
*"Watchpost is not a web application, so recommendations are useful, but never blocking
functionality, at least until I have data or users that insist otherwise."*  The relay-fault window's
10-second auto-close on the only station-tuning control is still worth fixing on its own merits — a
control that acts for you while you are reading it is poor regardless of any standard — but it does
not gate the batch and WCAG conformance is not claimed.  Revision 1 put it last, behind the only scope lever, in
the slot a re-cut removes first.  It touches one constant and one nav handler, contends with nothing,
and is surface work.  *(Deviation PD-6.  `handleRelayFaultNav` touches neither `.last` nor `.left`,
so 2.2.1's "extend" exception is architecturally absent, not merely unimplemented — OQ-2 must be ruled
before the fix, since "how long" answers none of 2.2.1's three mechanisms.)*

**Exit conditions**
- ~~FR-5 is a **set-level** property, not one window: at 80×24 every modal's focused row and footer
  keys are on screen, or the modal is declared with a reason.~~  **MET, `ac69622`.**
  `modes/tty/modal_reachability_test.go` walks the modal enum and asserts every line a window draws
  can be brought on screen at 80×24, driven through the real key path; the baseline ratchets both
  ways, so a window that improves takes its number down with it.  Two defects fixed rather than
  pinned: the offset counted in unwrapped coordinates while the panel scrolled wrapped ones, and the
  ctrl+d window a **release build ships** had no focus and therefore no scroll at all — five lines,
  all of what that window exists to say, unreachable by any key.  Three windows keep a non-zero
  count and it is one mechanism, filed as **F-55** for a HUM LEAD ruling.  The exit condition's own
  premise held: fixing only `ctrl+d` would have left it in setup and relay-fault.
- ~~The `"emergency"` scenario injects an `Evacuation Immediate`; every payload matches its label.~~  **MET, `c5bcd03`** — and generalised: the scenarios are DERIVED from `globalfeed.FeedLanes()`, one per lane the feed can produce, each label built from the payload it injects, and each round-tripped through `globalfeed.LaneOf` rather than against a second copy of the mapping.
- ~~Test events expire within two minutes~~ **MET, `c5bcd03`** (`testEventLife = 2 * time.Minute`).  The seen store keeps the id for its 7-day window as before, and the id is self-identifying (`watchpost-injected-`) — the persisted footprint is therefore bounded by identification, not by expiry.
- ~~**A real arrival pre-empts a test event, and fabricated events never consume the burst Max.**~~  **MET, `c5bcd03`.**  Read as *never DISPLACES*: every real arrival's slot is reserved before a test event is offered one, and a fabricated emergency takes no exemption from Max.  Unlimited fabricated reads would be the worse failure, so a test event still spends what is left of the budget.  Watched failing in both directions first.
- ~~`*** TEST EVENT ***` marking is per-line for the spoken read; **for the marquee it is lane chrome, not tape text**~~  **MET (`c5bcd03`, and completed on the 2026-09-07 rulings).**  The mark is `**TEST EVENT**` and it **prepends AND postpends the tape item** — the ruling overrides the plan's lane-chrome argument, on the grounds that the tape scrolls and it is the ITEM that is fabricated rather than the lane.  The severe window's row carries it leading the EVENT column, with `injected` in DETECTION.  Original wording: — the tape is one scrolling line, so an in-tape marker is off-window most of the
  time, and an 18-cell prefix per item at the 80-column floor makes the marker the majority of the
  tape.
- ~~**`advanceTicker` stays within its 71-alloc budget with a marked-event fixture present.**~~  **MET, `c5bcd03`** — 68 with a marked event, and the 133x44 memo-miss frame moves 5666 → 5668 against a 5888 budget.  Both are pinned in a test that fails if the fixture stops reaching the marked path.
- **FR-4.5 — the diagnostic read (HUM LEAD 2026-09-07).**  **MET.**  Not a head and tail: a whole
  ALERT-AGNOSTIC script, so exercising the machinery never waits on someone writing suitable content
  per hazard.  Head, one title per fabricated alert, one explanation, tail —
  `domains/radio/script/scripts/test-alert/`, with built-in fallbacks in Go for the two parts that
  identify the alert, because the scripts are user-editable and a marking an edit removes is not a
  marking.  The TONE is still the injected type's own, so what is under test is the machinery.  A
  card is read as a test only when EVERY event on it is fabricated: a burst carrying a real hazard
  keeps the ordinary head and tail, and its fabricated lines say what they are inside it.
- **FR-4.6 — STOP ALL.**  **DEFERRED to F-26, HUM LEAD 2026-09-07** ("Defer approved").  It needs a
  key-binding decision (OQ-3) and the obvious letters are taken.
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

**~~FR-6.5~~ RULED AND DEFERRED, 2026-09-08 (HUM LEAD).**  *"Potentially yes for Observer, NO for
Broadcaster — relay faults are the business of the Operator, not the reader."*  The requirement as
written asks for the wrong thing: **an operational error message over the air is confusing and the
listener can do nothing about it.**  The industry answer is infrastructure this app does not have —
a station cuts to a pre-recorded "we are experiencing technical difficulties", NOAA Weather Radio
says "this station is currently offline, please tune to <alternate>".

**What can be accounted for is a JOURNEY, and it is already F-27's:** if audio was running before the
fault, an intra-card transition carries the listener from the old audio to the fallback — relay →
switch location → fail → transition read → backup relay, the same shape as the ruled relay A → `[w]`
read → `[esc]`/finish → transition → relay A.  Catastrophic failovers cannot be accounted for.

**So FR-6.5 does not land in B5.**  It is not a descope by scope pressure: the thing it asked for is
refused for a broadcast audience, and the part that survives — Observer, where the listener IS the
operator and can retune — is a "potentially" whose wording is a HUM LEAD call and whose seam is the
duck path F-27 owns, with its own task and its own red team.  F-33 carries the ruling.

**Entry conditions**
- ~~**PL-D-5 is withdrawn and the bound is re-sited.**~~  **WITHDRAWN 2026-09-08, and it was
  REDUNDANT rather than merely mis-sited.**  `playClip`'s watcher already bounds the read: 12,000
  polls at 50 ms, and `if !held { i++ }` makes it held-excluded — the read's own player, never the
  live stream, which is exactly what the exit condition asks for.  What the read lacked was anyone
  being TOLD how it ended.  See `02-analysis/completion-spike.md`.  Original condition:  Revision 1 put it in `player.Engine.watch`.
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
- ~~A read completes, fails, or reports a fault within a stated bound.~~  **MET.**  The bound was already there and already the right shape — `playClip`'s watcher, held-excluded, on the read's own player — and what was missing was anyone being told how it ended.  `Engine.OnClipSpent` reports it, the deck ends the read on the air, and the derivation is written beside the constant.  Original condition:  **The bound is held-excluded
  elapsed time and is never applied to the live stream** — `watch` pauses the player entirely when a
  report gives way to an alert, so a naive wall-clock deadline kills the read that ducking exists to
  protect.  The number is derived from expected script length plus Piper's measured ~10 s
  per-utterance load, and both the number and its derivation are written down.
- ~~**The fail-safe direction is stated and tested:**~~  **MET.**  A job now carries its own cancel, a child of the caller's context; `abandonOnAir` unwinds the read onto the path a cut-short one already takes.  Watched failing.  Original condition: abandonment cancels the job's context so the
  existing `Failed{Routed:true}` path advances the schedule and `restore()` runs.  Without it the bed
  stays ducked, the arbiter stays occupied, and the reader speaks into a closed player — a silent
  station produced by the safety feature.
- ~~**FR-9.2**~~ **MET, with one condition removed for being unfalsifiable.**  The reader counts what reached the air after a tone and reports through `narrationVoice.fault` — the operator's surface, never spoken (HUM LEAD, 2026-09-08).  The diagnostic is hardened: a validated NAME, the cache root, 0600, rotated once past 8 MiB.  **The `ctx.Err() == nil` half of the trigger was removed:** every path to that line has already checked the air, so it was a branch with no failing input (D-2), and a planted mutation proved nothing could construct the state it excluded.  Original condition:  The **fault** is a UI
  obligation and lands with FR-6.5's window.  The **diagnostic** stays opt-in behind
  `WATCHPOST_DEBUG_RADIO`, hardened `debugAddr()`-style: a fixed, size-capped, rotated 0600 file under
  the cache root, with the env var selecting a validated name rather than an arbitrary path.
  *Revision 1 said "on by default," which would have made an unvalidated append-anywhere file write
  the default on a 24/7 process.*
  The fault fires only when **no part could be rendered and `ctx.Err() == nil`** — an operator pressing
  `esc`, a takeover pre-empting, or the pump stopping are all legitimate tone-without-words today.
- ~~**FR-9.3, reworded (HUM LEAD, 2026-09-07):**~~  **MET.**  The rotation's liveness is a `Tune` that never lands — the deck reports `Tuned` when audio actually plays, so a tune that never plays is silence nothing else sees.  Thirty seconds, derived and written down; once per ask; and a stop while a tune is in flight clears it rather than reporting, which is the rewording's whole point.  Original condition: *the main-track rotation produces a read within a
  stated interval, or it reports* — **unless the operator has chosen silence.**  Dead air the operator
  chose is not a fault.  This leaves ruling **I-2** intact: a deliberate non-delivery is not a fault,
  and neither is deliberate silence.  Revision 1's wording — "a cycle that yields nothing to say is a
  fault" — reversed I-2 by accident and would have raised the fault window on a quiet day, since the
  Director declines every non-alert cycle by design.  *The masthead / mastTail feature (0.16.0,
  Broadcaster) is the downstream surface that makes chosen dead air legible; FR-9.3 is what tells it
  the difference.*
- ~~#17 has a written disposition.~~  **MET** — posted on the issue: the stall was `fakePlayer`, not the completion path, and oto's own `playImpl` refuses to enter the playing state when its source is spent.  Closing it is the HUM LEAD's.
- Gate set green.

---

## B6 · Gates that can fail

**Lands:** FR-8 · **Size:** L · **Depends on:** B4, B5

**Opening task** — run `golangci-lint` and `staticcheck` once and **count**.

**Exit conditions**
- ~~`make lint` exists and is in `verify`, as **baseline + no-new-findings ratchet**.~~  **MET.**
  `scripts/lint.sh`, golangci-lint v2.13.1 pinned, fingerprinted by file+rule.  It fails in BOTH
  directions — a new finding of a baselined rule, and a baseline row matching nothing on disk.  Both
  watched: a ratchet that only tightens is half a gate, because a stale row is how a baseline
  outlives the finding it excused.
- ~~**`alloc-budget` joins `verify`.**~~  **MET**, and watched — **on the second plant.**  The first
  survived: `_ = make([]byte, 64)` never escapes, so the compiler removed the defect before the gate
  could count it.  Recorded in the roster, because a plant the optimiser deletes measures the
  optimiser.
- ~~**`ci.yml` and `verify` run the same set, in both directions.**~~  **MET.**
  `TestCIAndVerifyRunTheSameGates` carries `ciOnly`/`verifyOnly` and rots either way, so the naive
  convergence the plan warned about — the one that deletes the only allocation gate — fails the test
  rather than the release.
- ~~One completeness test covers the AA register and `--ascii`.~~  **MET, and both halves were worse
  than the plan said.**  `--ascii` missed ten windows AND fifteen glyphs: the scan checked the
  dashboard for seven named marks against eleven windows and a set of twenty-two.  It is now one scan
  over thirteen surfaces asking the only question that needs no list — is anything here outside
  ASCII? — which found six leaks in four windows, two of them (`Dot`, `Rail`) marks no list had.  The
  AA gate was the tautology the plan called it; the producer is now the 75 **declared** tokens
  crossed against the 64 registered, and each of the 11 exemptions carries a verified mechanism
  rather than a judgement (F-57).
- ~~The PTY journey's read step establishes its own precondition.~~  **MET.**  The recorded blocker
  was that it needed an absence check this harness cannot make.  It does not: a record's title stamp
  carries `· n / N` only while one is open, so one ARRIVING pattern proves the tab is populated and a
  row is focused.  `readableTab` walks the categories instead of naming one.  Passed against live
  feeds on both runs, and the `space` read passed with it.
- ~~Every gate in a **single** `gates.md` carries an evidence line naming a **watched** failure.~~
  **MET.**  `07-readiness/gates.md`, thirteen plants.  **It found a real hole:** `gate-controls` — the
  gate whose entire job is proving the others still fire — was GREEN while running no control at all,
  because two of its scripts ignored an unrecognised flag and ran their normal path to exit 0.
- **Gate set green.**  **MET WITH ONE DELIBERATE EXCEPTION.**  `make verify` is green.  `make journey`
  is **RED and stays red**: 26 of 28 steps pass, and the two Lookup steps fail on a freshly seeded
  install (**F-58**).  Widening its bound would convert a measured stall into a tick — the call F-44
  already made for the 11.4 s `space` read.  HUM LEAD 2026-09-08: document and proceed; the cause is
  not cheap to find.  A 40-second probe is committed so the next attempt starts from a measurement.

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

## UAT — required before REVIEW exit  *(HUM LEAD, 2026-09-07)*

**Two questions, and the second is not a formality.**

1. **Regression:** does anything behave differently from accepted behaviour?  The batches touch the
   config write path, the tone taxonomy, the classifier coupling, the arbiter, and the diagnostics
   window — none of which is supposed to change what a listener or operator experiences, except
   where a defect was being fixed.  **A structural change that alters behaviour is a defect until
   ruled otherwise.**
2. **Yield:** did the design changes produce anything beyond the defects they targeted?  Single
   ownership and good structure are a win on their own — the HUM LEAD's framing — but if collapsing
   six config writers or one tone mapping *also* removes a latent behaviour nobody had named, that is
   worth capturing rather than discovering later.

**What must be exercised, per batch:**

| Batch | Behaviour that must be unchanged | Behaviour that must be NEW |
|---|---|---|
| B1 | every alert class still sounds its ratified preset; the marquee lanes as before | an evacuation order sounds **three** dual tones |
| B2 | Settings saves, theme switches, location add/remove, cast changes all persist | nothing user-visible — this is the test |
| B3 | the diagnostics window opens and injects as before at 133×44 | it is usable at 80×24; test events expire in two minutes |
| B4–B7 | *(set at each batch's entry)* | |

**The B2 row is the interesting one.**  Six write paths became one, and the correct UAT result is
that a user cannot tell.  If they can, the refactor changed something it was not asked to.

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
