---
title: "0.15.0 — Pre-Broadcaster UI Improvements — DISCOVERY REPORT"
date: 2026-09-07
phase: DISCOVER
report_template: discovery-report
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
directives: FULL GIT; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD
status: "Step 2 complete; requirements restated against measured findings — awaiting HUM LEAD approval to enter PLAN"
---

# 0.15.0 — Pre-Broadcaster UI Improvements — DISCOVERY REPORT

## Bottom Line Up Front / BLUF

DISCOVER checked the brief's seven investigation areas against the code rather than against the
ledger, and the ledger was wrong more often than it was right.  Two requirements shrink
substantially, two grow, one is already satisfied, and one investigation produced a live defect that
shipped as 0.14.2 during the phase.

The locked problem statement survives unchanged and gained a tenth documented instance.

> **A person relying on Watchpost for hazard awareness cannot tell a quiet day from a station that
> has silently stopped telling them things.**

**The finding that reframes the release:** the diagnostic surface intended to answer that statement
already exists, is already correct about its hardest design decision, and **its "Emergency Order"
scenario injects a Tornado Warning.**  The instrument built to catch the class of defect that #15
turned out to be sends the wrong product.  Scope item 3 is not construction work.  It is repairing
an instrument that reports something it does not measure.

## What DISCOVER produced, by area

| Area | Result | Effect on scope |
|---|---|---|
| 1 — config writers | Confirmed.  **Six** owners; the ledger's "four siblings" is five.  `config.Save` is file-atomic but carries no mutex, so the read-modify-write is unserialised across owners | Unchanged |
| 2 — classifiers | F-2 wrong in three specifics; corrected in place at `f0ddddc`.  Yielded **#15**, shipped in 0.14.2 | Restated |
| 3 — memo keys | `modalKey` fully guarded: 11 of 11 modals, zero skips.  **F-30's premise is already satisfied** | **Shrinks sharply** |
| 4 — alert path | Seam exists and enters where a real alert enters.  Scenario payloads are wrong; bound is an hour | **Shrinks, then grows** |
| 5 — `ctrl+d` at 24 rows | Controls unreachable by any key at any focus.  Double-wrap at every width | **Grows** |
| 6 — Piper RSS | **Blocked on Arch hardware.**  No change | Blocked |
| 7 — docs tree | `06_docs` contains a compiled Go test package; 121 PII occurrences across 14 files | **Grows** |

Every probe written during this phase was deleted; the tree is clean at each commit.

## Functional Requirements

Restated against the findings.  Where a brief requirement survives unchanged it is marked as such;
where the evidence moved it, the movement is stated with its reason.

### FR-1 — Single ownership of shared rules  *(brief R1 — unchanged in substance)*

- **FR-1.1** Persisted configuration has exactly one write path.  Six independent
  `Load` → mutate → `Save` owners exist today: `setThemeHook`, `setUIHook`, `applySetup`,
  `saveCast`, `livePipelines.commit`, and `savePreference` — which documents itself as "the one
  path" while five siblings bypass it, not four.  `config.Save` is atomic at the file level and owns
  the `ticker_muted` mirror, so the target pattern is already demonstrated inside the function that
  needs to own the whole operation.
- **FR-1.2** One answer to "what category is this product."  `severe.Classify` and
  `globalfeed.LaneOf` both return a `category.Category` from the same product string and do not
  agree in the general case.
- **FR-1.3** The band's single-writer rule is enforced by a mechanism rather than a comment.
- **FR-1.4** The audio arbiter's hold, resume, and drop path is serialised against duck and restore.

### FR-2 — Producer/consumer completeness over closed sets  *(new — promoted out of R1)*

This was a sub-clause of the brief's R1.  The evidence promotes it to a requirement in its own
right, because it is the single mechanism that would have caught #15, and it generalises.

- **FR-2.1** Wherever a producer emits a value from a closed set and a consumer switches on it, a
  test walks the producer's set and fails if the consumer does not carry every member.
  `TestEveryFeedLaneSurvivesTheMarqueeMap`, written for 0.14.2, is the working template.
- **FR-2.2** `severeEvents()` is checked against both classifiers.  The nine products on the curated
  national query all name "Warning" or "Watch", or are the one civil-emergency product, which is the
  only reason `LaneOf`'s missing Advisory, Marine, Statement, and Forecast arms cannot diverge from
  `severe.Classify` today.  Adding one advisory to that list is a one-line edit that looks obviously
  safe and silently lands it on the band as a Warning.
- **FR-2.3** A default arm that absorbs unknown members of a closed set is a defect unless the
  absorption is stated as a decision.  #15 was a missing arm, not a wrong one, and the default made
  the gap read as deliberate.

### FR-3 — Memo keys carry identity, not position  *(brief R2 — substantially reduced)*

Measured, not assumed.  `TestTheMemoKeyCoversEverythingTheFrameShows` derives fields by reflection,
walks the modal enum rather than a written list, states its property as an implication, and skips
loudly rather than passing quietly.  It reports **11 of 11 modals covered, zero skips.**

- **FR-3.1** `bodyKey` gains the guard `modalKey` already has.  18 fields against the frame that runs
  24/7, and nothing derives its coverage.  This is the side the 0.13.0 lookup defect and #11 both
  lived on.
- **FR-3.2** The six data-cache memos — `tileMemo`, `boxMemo`, `gridMemo`, `sourceMemo`, `hostMemo`,
  `failureMemo` — receive a **ruling**, not the same test.  Their failure mode is stale data, not a
  stale frame, and the frame guard's property does not transfer.
- **FR-3.3** ~~A cursor fixture for the completeness test (F-30).~~ **ALREADY SATISFIED.**  Every
  modal in the enum has a fixture rich enough to draw, and none skip.  F-30 should be closed by
  measurement rather than by work.
- **FR-3.4** `tileMemo` and `boxMemo` consolidated with frame cost measured.  Blocked on Arch
  hardware — see the constraints below.

### FR-4 — The station can be tested on demand  *(brief R3 — reshaped)*

**The seam exists and is right about the hard part.**  `app/inject_debug.go` enters immediately
after the source fetch and before `globalfeed.Active`, so an injected alert crosses the active-window
filter, the location tie, the severe index, the radius scope, fresh-detection, the rail's plan, the
Composer, and the Reader.  `app/inject_release.go` defines the queue as an empty struct that still
answers `take()`, so the capability is absent from the binary rather than disabled within it.  The
brief's technical constraint 2 is already implemented and already argued in the file.

What remains is narrower and more specific than the brief assumed:

- **FR-4.1** Scenario payloads match their labels.  The "Emergency Order (leads the rail, overruns
  Max)" scenario injects `"Tornado Warning"` and is byte-identical to the default scenario.  A
  Tornado Warning is not in `civilEmergencyProducts()`, so it cannot lead the rail — leading the rail
  is what being an emergency order means.  **Had this scenario injected an actual
  `Evacuation Immediate`, an operator pressing `ctrl+d` would have watched #15 happen.**
- **FR-4.2** One scenario per category the feed can produce, derived from the closed set rather than
  hand-listed, so a new lane arrives with a way to exercise it.  Same shape as FR-2.1.
- **FR-4.3** Test events expire within two minutes.  They currently carry
  `Until: now.Add(time.Hour)`, which is the contamination risk #9 names, at thirty times the stated
  bound.
- **FR-4.4** `*** TEST EVENT ***` treatment in the ticker and in `[w]` reports.  No implementation
  exists today; the only marking is `Location: "Injected Test Location"`.
- **FR-4.5** Diagnostic head and tail on test audio, on the American emergency-broadcast pattern.  No
  implementation exists today.
- **FR-4.6** A STOP ALL control returning Observer to no active reads or relays.
- **FR-4.7** `/debug/dump` checks its method before the debug surface is documented for users, which
  FR-4 effectively does.

### FR-5 — The diagnostics window is usable at 24 rows  *(brief R3.5 — promoted, and larger)*

Reproduced by probe at 80, 100, and 133 columns.  At 24 rows the window renders 14 lines where 44
rows renders 21.

- As opened, **none** of the window's own controls are visible.
- `nav-down` brings `INJECT AN ALERT:` into view, then removes it again; the viewport oscillates.
- **The scenario labels and the `›` cursor are never visible at any focus position, by any key.**
- `page-down` and `nav-end` do nothing: `modalDebug` is not among the scrolling windows in
  `handleNav`, and `handleDebugNav` moves focus without touching any scroll offset.
- The footer offers `[enter] Inject` throughout.  An operator would be firing a scenario they cannot
  see.
- The prose double-wrap — `It exists so / a / person can check` — occurs at **133 columns as well as
  80**.  A hard-wrapped string is being re-wrapped.  F-35 describes this as an 80-column problem; it
  is not width-specific.

**FR-5.1** The window's controls are visible and reachable at 24 rows at every supported width.
**FR-5.2** Prose wraps once, at render width.
**FR-5.3** This gates FR-4: the window becomes user-facing, and it is unusable today.

### FR-6 — Silent defects with user-visible consequences  *(brief R5 — unchanged)*

FR-6.1 (#13) a never-resolving lookup is recognisable rather than perpetually "loading" ·
FR-6.2 (F-46) the *All Reports* picker appears in Settings · FR-6.3 (F-5) the three config-only cast
roles are reachable or documented as deliberately not · FR-6.4 (F-36) the relay-fault window's
ten-second auto-close, a **WCAG 2.2.1 Level A** failure on the only station-tuning control ·
FR-6.5 (F-33) the relay-fault window is audible · FR-6.6 (F-43) the tone with no words, chased under
a timebox.

### FR-7 — Rulings, not migrations  *(brief R6 — one item grew a precondition)*

FR-7.1 (F-3) rule on `app/release.go` as provider or not · FR-7.2 (F-49) rule on the cache root ·
FR-7.3 (F-45) ratify the 23 stale ledger rows · FR-7.4 (F-12) the ephemeral attribution stamps.

- **FR-7.5** The documentation survey reports **three** categories, not two, and a publication
  precondition:
  - **`06_docs` is not documentation alone.**  `06_docs/mutants/` holds `mutants_test.go`,
    `harness_test.go`, and 171 `.py` mutants, compiled by `make mutant-check`
    (`go test -tags mutants ./06_docs/mutants`).  A wholesale relocation breaks a quality gate.
  - **Publication would leak PII**: 81 occurrences of `/Users/bthompso`, 40 of `bthompso`, and one
    `/home/research`, across 14 files.  F-48 already covers one such leak in the README; publishing
    this tree multiplies it by 121.
  - Counts for the record: 279 files, 4.3 MB — 194 prose, 77 evidence records, 6 code-adjacent, 2
    other.  The evidence is cited by the prose, so the two cannot be separated without rewriting
    citations.

## Non-Functional Requirements

- **NFR-1 — Nothing regresses the frame path.**  Memo work is measured against
  `docs/accepted-costs.md`, not assumed.  The frame runs 24/7.
- **NFR-2 — The injector is absent from release artifacts, provably.**  An artifact-level check, not
  a source-text assertion.  FR-4 makes the surface user-facing, which converts this from
  "if it is cheap" to a release gate.
- **NFR-3 — Every gate carries a watched failure.**  One evidence line per gate in `gates.md`
  recording an observed failure against a deliberately broken input.
- **NFR-4 — Both platforms run in CI while the work is in progress**, not at the release PR.
- **NFR-5 — A test event is indistinguishable in path and unmistakable in presentation.**
- **NFR-6 — WCAG 2.2.1 Level A**: no hard timeout on a control.

## Constraints & Dependencies

1. **Blocking, HUM LEAD-owned:** `perf-protocol.md` §3–5 on Arch hardware.  FR-3.4 and the
   resident-Piper decision (MVS-D-17 / OQ-18) have no input until it is measured.  §3 was amended at
   `9ee6bdc` to exclude the 0.14.0 window: on Linux that build never reached Piper for a takeover
   (#7), so a run against it samples one voice's footprint and records it as two.  **0.14.1 or
   later.**
2. **`06_docs` contains source.**  Any documentation relocation must leave `06_docs/mutants/` where
   the module expects it.
3. **PII scrub precedes publication**, and must be a gate rather than a one-time edit, since the tree
   is written continuously.
4. Go 1.27.0 with `GOTOOLCHAIN=local` · any table is a go-studs table · no reimplementing lipgloss or
   go-studs · P10 exemptions presented for ratification, never self-approved · no AI attribution ·
   no PII in shipped artifacts.
5. **Nothing in 0.15.0 may require Broadcaster to exist.**
6. **Sequence:** 0.15.0 → 0.15.x (F-40, the fire feeds) → 0.16.0 (Broadcaster).

## Risk Assessment

| ID | Risk | Rationale | Mitigation |
|---|---|---|---|
| **RS-1** | Scope | FR-1 through FR-7 remains large, though smaller than the brief's R1–R6 after FR-3 shrank and FR-4 lost its construction half | The scope predicate — a loud failure is not 0.15.0 — plus a checkpoint after FR-1 to FR-5 |
| **RS-2** | Arch measurement is blocking and externally owned | FR-3.4 and OQ-18 have no input | HUM LEAD run; DISCOVER cannot honestly exit without it |
| **RS-3** | FR-4 raises the stakes on injection | The surface becomes user-facing for the first time | NFR-2 becomes a release gate; if it slips, FR-4 does not ship |
| **RS-4** | FR-6.6 is an unbounded hunt | F-43 has never been reproduced | Timebox and a written disposition, decided before it starts |
| **RS-5** | FR-1 and FR-3 are wide refactors carrying hazard information | SEV-0 | TDD is directive-mandated; safety is the tests that exist before the change |
| **RS-6** | **New — a gate reported green over a failed step** | A `$(MAKE)` chained with `;` inside a recipe does not propagate status.  Found and fixed for one instance in 0.14.2; nothing checks the others | Belongs to NFR-3 |
| **RS-7** | **New — an intermittent stall in the broadcast completion path** | #17.  52 ms of work, unmoved by starving to one core under `-race`, missed a 5 s bound: 96×.  One failure in three runs on identical source, same platform | Investigation on Linux hardware.  **The bound must not be widened** — that deletes the symptom and leaves the stall |

## Open Questions

- **OQ-1** — ~~Fire release numbering.~~ **RULED:** Broadcaster is 0.16.0, so the fire release is
  0.15.x.  Remaining half: does F-40's fix fit a point release, or does a new fire feed make it a
  minor?
- **OQ-2** — FR-6.4: how long should the relay-fault window stay open, and does a keypress reset it?
- **OQ-3** — FR-4.6: what key does STOP ALL take, and where does it live?
- **OQ-4** — FR-7.1: is `app/release.go` a provider?
- **OQ-5** — FR-7.2: `os.UserCacheDir()` or `~/.watchpost/`, and does the migration ride with the
  ruling?
- **OQ-6** — FR-6.6: F-43's timebox and its disposition if unreproduced.
- **OQ-7** — FR-3.4: does consolidating the two memo types cost frame time?  Blocked on RS-2.
- **OQ-8** — FR-7.5: does documentation move at all, given that `06_docs` holds source and 121 PII
  occurrences?
- **OQ-9** — **New.**  FR-3.2: what is the right property for a data-cache memo?  The frame guard's
  implication does not transfer, and inventing the wrong property is worse than having none.
- **OQ-10** — **New.**  Should F-30 be closed as satisfied by measurement, and should F-35 be
  rewritten?  Both rows describe a state that no longer matches the code — F-30 understates the
  coverage, F-35 understates the defect.

## Recommendation

**Proceed to PLAN**, with three qualifications.

1. **FR-2 leads.**  The producer/consumer completeness check is the one mechanism that generalises
   across FR-1, FR-3, and FR-4, and it already has a working instance from 0.14.2.  Building it first
   makes the rest of the release cheaper and gives every subsequent item a way to prove itself.
2. **FR-5 precedes FR-4.**  The window is unusable at 24 rows today.  Adding scenarios to a surface
   whose controls cannot be seen produces an instrument nobody can operate, which is how this release
   started.
3. **DISCOVER does not exit until RS-2 is measured.**  Everything else here was checked against the
   code rather than the ledger, and the ledger lost three times out of seven.  Accepting an
   unmeasured input after that would be inconsistent with the rest of the phase.

**Two ledger rows should be dispositioned before PLAN**, per OQ-10: F-30 is satisfied and F-35 is
understated.  Both are HUM LEAD calls, since a ledger row is a record and not only a task.

## Evidence

Findings were produced by probe rather than by reading wherever a claim could be measured.  Every
probe was temporary and deleted in the same session:

- `app/zz_discover_probe_test.go` — the lane divergence.  **Its first version was invalid**: it
  built a `globalfeed.Event` without a `Class`, and `ClassQuake` is the zero value, so an unset event
  is silently a quake.  The first result was discarded because it disagreed with the source.  That
  zero value is itself a latent hazard of the same family: `Class` has no invalid state, so any
  construction path that omits it produces a quake in the Disasters lane.  Both current call sites
  set it — safe by discipline, not by construction.
- `domains/radio/player/zz_probe_test.go` — the completion margin, 12 runs per condition.
- `modes/tty/zz_probe_test.go` — the `ctrl+d` geometry and key-reachability walk.

## Source documents

`08-reports/project-brief.md` · GitHub #9, #12, #13, #14, #15, #17 · `06_docs/follow-ups.md`
(47 open rows; F-2 corrected at `f0ddddc`) · `06_docs/quality-observations.md` ·
`multi-voice-support/08-reports/debrief.md` · `multi-voice-support/07-readiness/perf-protocol.md`
(§3 amended at `9ee6bdc`) · `CHANGELOG.md` §0.14.2.
