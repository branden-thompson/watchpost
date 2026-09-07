---
title: "0.15.0 — Pre-Broadcaster UI Improvements — DISCOVERY REPORT"
date: 2026-09-07
phase: DISCOVER
report_template: discovery-report
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
directives: FULL GIT; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD
status: "COMPLETE. All deliverables discharged, all HUM LEAD rulings recorded, gates green — awaiting approval to enter PLAN"
---

# 0.15.0 — Pre-Broadcaster UI Improvements — DISCOVERY REPORT

## Bottom Line Up Front / BLUF

DISCOVER checked the brief's seven investigation areas against the code rather than against the
ledger.  Every ledger row it sampled outside the seven has since also proved stale, so the honest
statement is not a score: **roughly forty rows have not been audited since 0.14.0.**  Two
requirements shrink, three grow, and one investigation produced a live defect that shipped as 0.14.2
during the phase.

**This report has been revised after an eleven-lens red team (`red-team-discover.md`), which
returned NO-GO as written.**  Seven lenses independently found that an entire brief theme had
vanished from the requirement set with no descope line; it is reinstated below as FR-8.  Three
findings were errors in this report or in code it cites, all of the same kind — a number read
without asking what produces it.  They are marked where they occurred rather than quietly fixed.

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
| 3 — memo keys | `modalKey` guarded for today's 11 windows, zero skips — but the guard **skips** an unfixtured window and hard-codes three nested structs, so F-30 **stands** | Reduced, not eliminated |
| 4 — alert path | Seam exists and enters where a real alert enters.  Scenario payloads are wrong; bound is an hour | **Shrinks, then grows** |
| 5 — `ctrl+d` at 24 rows | Controls unreachable by any key at any focus.  Double-wrap at every width | **Grows** |
| 6 — Piper RSS | **No longer blocking.**  §3–5 answer OQ-18 alone; they cannot measure FR-3.4, whose memos sit on the provider fetch path, not the frame | Unblocked (red team C-6) |
| 7 — docs tree | `06_docs` contains a compiled Go test package; and the 121 PII occurrences are **already published** — the repo is public and they have been on `origin/main` since 0.13.0 | **Grows** |

Probes written to measure a claim were temporary and deleted, with one deliberate exception: the
four-classifier cross-table is **retained** as `app/classifier_crosstable_test.go`, because it pins a
count and is therefore a gate rather than a snapshot.  The red team's complaint that this phase
destroyed every instrument it built is fair for the rest, and is the reason for the exception.

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
- **FR-1.2** No classifier over the NWS product-string set carries an unratified default arm.
  **There are four, not the two this report first named** (red team C-2, verified):
  `domains/severe/severe.go:105` `Classify` (partial — `(Tab, bool)`); `domains/globalfeed/stack.go:82`
  `LaneOf` (total, defaults to Warnings); `domains/radio/cast/tone.go:186` `Classify` — **on the audio
  path**, with no civil-emergency arm, so an Evacuation Immediate falls to `ClassWarning`; and
  `platform/render/sgr.go:226` `AlertIsWarning`, self-described as "THE warning-vs-advisory classifier
  (single owner)".  *Merging them is not the requirement and is not safely achievable — `Classify` is
  partial and `LaneOf` is total, so collapsing them either shows everything in the severe window or
  drops events off the band.  The deliverable is the four-way cross-table, produced below,
  plus FR-2.2's coupling check.*
- **FR-1.3** The band's single-writer rule is enforced by a mechanism rather than a comment.
- **FR-1.4** The audio arbiter's hold, resume, and drop path is serialised against duck and restore.

### The four-classifier cross-table  *(the brief's area-2 deliverable, produced 2026-09-07)*

The brief said *"that table IS the requirement."*  The first draft of this report returned one cell
instead.  It is now measured and retained as `app/classifier_crosstable_test.go`, which emits the
table and pins the divergence count, so it is a gate rather than a snapshot.

| Product | severe → tab | LaneOf → lane | cast → tone |
|---|---|---|---|
| **Evacuation Immediate** | Emergency Orders | Emergency Orders | **Warnings** |
| Tornado Warning | Warnings | Warnings | Warnings |
| Hurricane Warning | Warnings | Warnings | Marine |
| Civil Emergency Message | Disasters | Disasters | **Warnings** |
| Child Abduction Emergency | Statements | Statements | **Warnings** |
| Small Craft Advisory | Advisories | **Warnings** | Advisories |
| Flood Advisory | Advisories | **Warnings** | Advisories |
| Air Quality Alert | Advisories | **Warnings** | Warnings |
| Coastal Flood Statement | Statements | **Warnings** | Special Statements |
| Special Weather Statement | Statements | **Warnings** | Special Statements |
| Marine Weather Statement | Marine | **Warnings** | Special Statements |
| Hazardous Weather Outlook | *(Forecasts — blank label)* | **Warnings** | Warnings |
| Earthquake | Disasters | Disasters | Disaster Events |
| Nonexistent Product Type | NOT SHOWN | Warnings | Warnings |

**Three findings, none of which reading produced:**

1. **An `Evacuation Immediate` gets the WARNING tone** — `toneRank: 5`, identical to a Severe
   Thunderstorm Warning.  The tone taxonomy has six classes and **none is Emergency**
   (`cast/tone.go:27-32`).  C-2 taught the feed, the window and the read ladder; #15 taught the
   marquee in 0.14.2; the tone was never told.  Under the surface taxonomy ruling this is
   **listener-facing** — the tone is the entire signal before the words start.  **Filed as #18,
   and RULED the same day (HUM LEAD).  The lane was already decided at C-2 and was not in
   question; the TONE becomes three repetitions of the existing dual tone — a warning is 1 ×, an
   evacuation order is 3 × — on the NWS attention-signal pattern, so the count carries the urgency
   before a word is spoken.  It reuses `PresetDualTone` rather than inventing a fourth sound: the
   distinction is carried by repetition, the one dimension the six-class taxonomy does not use.**
   Eleventh instance of the release's shape.  **The fix stops this instance and nothing else** — the
   recurrence is FR-2's job: every `category.Category` that can reach a read maps to exactly one
   `cast.Class`, declared in one place, and a category with no mapping fails a test instead of
   falling through to `ClassWarning`.  No further ruling needed; the cross-table is the fixture.
2. **Seven of twenty-five products are filed differently by the window and the band** — the whole
   Advisory / Statement / Marine / Forecast family.  FR-2.2 called this latent; it is latent only for
   the *national query*.  These products reach the window through tracked locations, so the
   divergence is reachable today.
3. **`category.Forecasts` renders a blank label**, so a Hazardous Weather Outlook files into a tab
   with no name.  Minor, and noted rather than chased.

**One limitation of the instrument, stated:** `render.AlertIsWarning(event, severity)` is fixtured
with an empty severity, so its column exercises the string path only.  A real alert carries a
severity and would take the `severe`/`extreme` arm.  That column is not evidence of a defect and is
excluded from the table above.

### FR-2 — Producer/consumer completeness over closed sets  *(new — promoted out of R1)*

This was a sub-clause of the brief's R1.  The evidence promotes it to a requirement in its own
right, because it is the single mechanism that would have caught #15, and it generalises.

- **FR-2.1** Wherever a producer emits a value from a closed set and a consumer switches on it, a
  test walks the producer's set and fails if the consumer does not carry every member.  **Every member
  is accounted for explicitly: carried, or declared unreachable with a written reason.  A member in
  neither list fails the test.**
  *`TestEveryFeedLaneSurvivesTheMarqueeMap` is the template — but not as first written.  Its first
  version walked `category.Lanes()` and `continue`d when its fixture disagreed, asserting **4 of 7
  lanes** and reporting green, with Advisories — the hazard FR-2.2 names — among the three it skipped.
  A `continue` where F-30 has a `t.Skipf`, inside the instrument written to catch exactly that.  Found
  by two lenses, confirmed by probe, rewritten, and its three failure modes observed.*
- **FR-2.1a** The population is sized before the mechanism is generalised: **26 `iota` closed sets and
  48 `default:` arms** in non-test source.  Generalising from one instance is how the first version of
  the template happened.
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
- **FR-3.3** The guard **fails** on an unfixtured window instead of skipping it, and derives the
  nested-struct set instead of naming it.  *(F-30 — and the first draft of this report got this
  wrong.  I read the run's 11/11 with zero skips as coverage and wrote that F-30 was satisfied.  It
  is not: that number describes today's windows and says nothing about the guard's behaviour when a
  window is added, which is the whole point of the row.  Both of its mechanisms are confirmed in
  source — `memo_completeness_test.go:49` `t.Skipf`s an unfixtured window rather than failing, and
  `:122` descends into exactly three named nested structs, `relayFault || debug || setup`, so a
  fourth is never perturbed.  Reading a green number as coverage is the precise error this report
  accuses the ledger of, and I made it here.)*
- **FR-3.4** `tileMemo` and `boxMemo` consolidated, **measured on the path they actually sit on.**
  *Corrected after red team C-6: they are fields on the FIRMS and USGS clients
  (`domains/fire/firms/firms.go:65`, `domains/seismic/usgs/usgs.go:80`) — provider fetch, on scheduler
  goroutines, **not the render frame.**  `perf-protocol.md` §3–5 cannot measure them, and a frame
  allocation pin would have returned a green false pass for a change the frame cannot see.  The
  instrument is `make alloc-budget` plus the memo hit/miss gauges, on this machine.  **This item is no
  longer blocked on Arch hardware.***

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
  - **The PII exposure is LIVE, not prospective** *(red team C-3, verified — this report had the
    tense wrong).*  `gh repo view` returns `PUBLIC`, `06_docs` is on `origin/main`, and the content has
    been published since 0.13.0.  A gate that blocks future writes while every published tag carries
    the same content defends a door that is open.  The scan was also **pattern-limited to the
    maintainer's name** and missed a ZIP-level **home locality in 35 files**.  What is owed is an
    exposure statement, a categorised re-scan (identity, location, host, credential, internal URL,
    path), and a HUM LEAD remediation ruling on the *published* copy — on the same footing F-48 already
    has.  OQ-8 is the wrong question: the documentation has already moved.
  - Counts for the record, **with the scope named** (the earlier figures silently measured
    `06_docs/02_features` while the bullet above them said `06_docs`): `06_docs/02_features` is 280
    files; all of `06_docs` is **460 tracked files**, which is the tree the ruling applies to.  Of
    `02_features`: 194 prose, 77 evidence records, 6 code-adjacent, 2 other.  The evidence is cited by
    the prose, so the two cannot be separated without rewriting citations.

### FR-8 — Every gate can fail  *(brief R4 — REINSTATED after red team C-1)*

**This theme was absent from the first draft of this report, with no descope line.  Seven of eleven
lenses found it independently.**  It is not a scope reduction that was decided; it is one that
happened.  The locked problem statement names it directly — *an AA register that shrinks when a token
is added, and `--ascii` scans that miss four windows, are how a surface silently degrades* — and the
public issue #14 still promises it.

- **FR-8.1** A `make lint` target wiring `golangci-lint` and `staticcheck`.  Neither has ever run as a
  gate in this repository, while `.github/PULL_REQUEST_TEMPLATE.md` already asks reviewers to tick
  that both are clean.  *(F-15; the ledger's own due column reads 0.15.0)*
- **FR-8.2** One completeness test covering the AA contrast register and `--ascii` together.  Measured
  during the red team: **eleven declared tokens sit outside `aaPairs()`**, and
  `TestEveryPaintedPairReadsAAInEveryTheme` iterates that hand-list — a test whose name asserts a
  completeness it cannot deliver, which is FR-2's shape on a different producer set.  `--ascii` is
  scanned for two frames only.  *(F-18, F-37, F-47)*
- **FR-8.3** The PTY journey's read step establishes its own precondition.  *(F-44)*
- **FR-8.4** Every gate in `gates.md` carries an evidence line recording a watched failure.
  *(formerly NFR-3)*

### FR-9 — A read completes, fails, or reports a fault within a stated bound  *(new — red team C-4)*

Three lenses noted that F-43 was promoted to a requirement with a timebox while #17 stayed a risk row
with an activity for a mitigation.  One traced the mechanism: `domains/radio/player/engine.go:518`
`watch()` detects completion only by polling `IsPlaying()`, and **carries no deadline on a read at
all.**

- **FR-9.1** Every spoken read completes, fails, or reports a fault within a stated bound.  A read
  that overruns is abandoned loudly and the schedule advances.
- **FR-9.2** A sounded tone not followed by a spoken part produces a user-perceivable fault, and its
  diagnostic is on by default.  *(Today `read:gaveup:after-tone` is emitted only when
  `WATCHPOST_DEBUG_RADIO` is set, so F-43's own closing condition is unreachable in normal use.)*

- **FR-9.3** A broadcast cycle that yields nothing to say is a **fault**, not a quiet moment, and is
  reported as one.

**FR-9 stands whether or not #17 is ever reproduced.**  A bound is a requirement about the product;
#17 is a question about one failure.

### The surface taxonomy — HUM LEAD ruling, 2026-09-07

> **Visual is the operator.  Audio is the listener.**

This settles the red team's strongest finding, and settles it as a mis-framing rather than a gap.
Three lenses argued that the locked statement names a *listener* while every requirement serves the
*operator*.  Under the ruling that is false: FR-9, FR-9.2, FR-6.5, F-43 and #17 are **listener**
requirements, and FR-4, FR-5, FR-7 and FR-8 are **operator** requirements.  Metric **T** measuring
"operator intent" is correct rather than a substitution, because `ctrl+d` is a visual surface and
therefore operator-facing by definition.  What was missing was never the requirements; it was this
sentence.

**A periodic audible liveness signal is OUT OF SCOPE (same ruling).**  The reason is structural and
is why FR-9.3 exists: the station always has something to report — a default location and a service
radius produce a rotation, on the NOAA weather-radio pattern, whether or not any hazard is active.
**So a listener CAN already tell a quiet day from a dead station, because a quiet day still speaks.**
Silence is not the absence of news; silence is the fault.

That makes "the station always has something to say" a **load-bearing invariant of the product's
safety argument**, and today nothing enforces it.  One path is already handled — a card whose script
renders nothing is declined (`app/executors.go:273`, *"the script rendered nothing to say"*) — but
being declined is not the same as the cycle producing a read.  A rotation in which every candidate
declines is silent, and under this ruling that silence is indistinguishable from a dead station to
the only sense the listener has.  FR-9.3 is what turns the ruling into something a test can fail.

### FR-10 — Hazard coverage is known, and its boundary is stated  *(F-40 — ROLLED IN, HUM LEAD 2026-09-07)*

**Reversal of the intake ruling, and the reason is FR-9's reason.**  At intake F-40 was deferred to
its own 0.15.x.  The red team then asked what the listener is told in the meantime, and the answer
was nothing — which under the surface taxonomy is the same failure shape as an unstated silence.  A
boundary nobody states is a boundary nobody can account for.  So the fix comes into 0.15.0 rather
than the disclosure being deferred with it.

The sighting is real, dated, and named: the Brengel Fire, Vista CA, morning of 2026-09-06.  An
evacuation order was issued and the hotspot appeared in the details for **neither** Oceanside 92057
nor Vista — the two nearest tracked locations.

It remains **two questions, and neither is the read path**:

- **FR-10.1 — the FIRE.**  Why no hotspot for a fire close enough to force evacuations in a
  neighbouring town.  Candidates, none yet eliminated: a FIRMS radius, a confidence filter, a
  satellite-pass latency, or the detection genuinely not being in the feed yet.
- **FR-10.2 — the ORDER.**  `Evacuation Immediate` is now fetched and leads the read (C-2, and #15
  fixed its marquee lane in 0.14.2), but that is the **NWS** alerts feed.  A fire evacuation issued
  by a county or by CAL FIRE may never appear there at all, **which makes C-2's fix necessary and not
  sufficient.**
- **FR-10.3 — the boundary is stated to the listener and to the operator.**  Whatever FR-10.1 and
  FR-10.2 conclude, the product says what it does not watch.  This is the requirement that made the
  roll-in necessary, and it is deliverable even if the other two conclude "cannot be fixed here."

**This keeps its own DISCOVER inside the release** (HUM LEAD, 2026-09-06, unchanged): the feeds get
examined rather than the reader, because guessing from one sighting is how the wrong thing gets
built.  It is scoped as a bounded investigation with a written disposition, not as an open hunt —
the same discipline RS-4 applies to F-43.

**Scope consequence, stated rather than absorbed:** this is the largest single item in the release
and RS-1 was already its top risk.  See the recommendation for the offset.

## Metrics of Success — carried forward, with owners

*Absent from the first draft; four lenses found that independently.  The brief hardened each against
an anti-solution at intake and this report discarded the hardening by omission.*

| Metric | Owning requirement | Baseline | Target |
|---|---|---|---|
| **T** — time to confirm the station works | FR-4 + FR-5 | Unbounded | ≤ 2 min |
| **D** — unexplained duplicate implementations | FR-1 + FR-2, gate landed by FR-8.1 | **Unmeasured — owed before PLAN exit** | 0 |
| **K** — memo keys with no completeness guard | FR-3.1 + FR-3.3 | 1 on the frame path (`bodyKey`); 6 data caches pending FR-3.2's ruling | 0 |
| **G** — gates never observed failing | FR-8.4 | **Unmeasured — needs a canonical gate roster** | 0 |

**Two hardening notes the red team added.**  K must stay an absolute count: FR-3.2's ruling on the six
data-cache memos removes them from the numerator by definition rather than by guarding them, which is
the anti-solution the brief hardened **D** against, re-imported.  And **G** has no denominator today —
`gates.md` exists per feature, not as one roster, and `ci.yml` omits `vet-tags` and `mutant-check`
from what `make verify` runs.

## Brief → requirement trace

*Added after red team C-1: the absence of this table is what let a theme disappear.*

| Brief | Report | Movement |
|---|---|---|
| R1 | FR-1 | Unchanged in substance; FR-2 promoted out of it |
| R2 | FR-3 | Reduced — the modal guard exists; `bodyKey` does not |
| R3 | FR-4, FR-5 | Reshaped — the seam exists; FR-5 promoted and enlarged |
| **R4** | **FR-8** | **Reinstated.  Absent from the first draft with no descope line** |
| R5 | FR-6 | Unchanged |
| R6 | FR-7 | Unchanged, premise corrected (C-3) |
| — | FR-2 | New — promoted to stand alone |
| — | FR-9 | New — from #17, red team C-4 |

## Non-Functional Requirements

- **NFR-1 — Nothing regresses the frame path.**  Memo work is measured against
  `docs/accepted-costs.md`, not assumed.  The frame runs 24/7.
- **NFR-2 — The injector is absent from release artifacts, provably.**  An artifact-level check, not
  a source-text assertion.  FR-4 makes the surface user-facing, which converts this from
  "if it is cheap" to a release gate.
- **NFR-3 — Every gate carries a watched failure.**  *Promoted to FR-8.4; it was the only part of the
  brief's R4 that survived the first draft, and demoting a theme to one NFR is how the rest went
  missing.*
- **NFR-4 — Both platforms run in CI while the work is in progress**, not at the release PR.
- **NFR-5 — A test event is indistinguishable in path and unmistakable in presentation.**
- **NFR-6 — WCAG 2.2.1 Level A**: no hard timeout on a control.

## Constraints & Dependencies

1. **HUM LEAD-owned, and NO LONGER PHASE-BLOCKING:** `perf-protocol.md` §3–5 on Arch hardware.  It
   is the right instrument for **OQ-18, the resident-Piper decision** — and only that.  FR-3.4 came off
   this dependency at red team C-6.  §3.6 already states OQ-18's default ("otherwise
   process-per-utterance + the FR-12 cap stays"), so not measuring costs a design that was not going to
   be built in 0.15.0 anyway.  §3 was amended at
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
| **RS-1** | Scope — **now the live risk, and unmitigated by descoping** | FR-1 to FR-10, nothing removed.  FR-3 shrank and FR-4 lost its construction half, but FR-8 was reinstated and FR-10 rolled in, so the release is larger than the brief's R1–R6, not smaller.  Descope was recommended (nine lenses on FR-7) and declined on precedent grounds | The scope predicate no longer helps: every remaining item passes it.  What is left is **sequencing and early detection** — FR-2 first, FR-5 before FR-4, a defined checkpoint after FR-1 to FR-5, and **FR-10 started immediately in parallel**, since it investigates feeds and does not contend with any other item for code.  PLAN owes a size per FR |
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
- **OQ-10** — ~~Should F-30 be closed and F-35 rewritten?~~ **RESOLVED 2026-09-07.**  F-35 amended
  (understated on both halves).  F-30 stands unchanged and carries a dated re-verification; the
  claim in this report that it was satisfied was mine and was wrong.

## Recommendation

**Proceed to PLAN**, with three qualifications.

1. **FR-2 leads.**  The producer/consumer completeness check is the one mechanism that generalises
   across FR-1, FR-3, and FR-4, and it already has a working instance from 0.14.2.  Building it first
   makes the rest of the release cheaper and gives every subsequent item a way to prove itself.
2. **FR-5 precedes FR-4.**  The window is unusable at 24 rows today.  Adding scenarios to a surface
   whose controls cannot be seen produces an instrument nobody can operate, which is how this release
   started.
3. **DISCOVER exits without RS-2.**  *Reversed after red team C-6 and F-A.*  RS-2 cannot measure
   FR-3.4 — the memos are on the provider fetch path, not the frame — and it is the right instrument
   for OQ-18 alone, whose default §3.6 already states.  Holding a SEV-0 phase on externally-owned
   hardware, with no date and no fallback, to answer a question whose answer is already the fallback,
   is a mitigation that restates the risk.  **OQ-18 and §3/§4 move to 0.16.0 DISCOVER**, where a
   Broadcaster workload justifies the paired v0.13.0 + 0.14.2 runs §4 actually requires.
4. **Both open conditions are now closed.**  The surface taxonomy is ruled (visual = operator, audio
   = listener) and F-40 is rolled in as FR-10.  The four-classifier cross-table is a DISCOVER
   deliverable owed by this report, not a ruling owed by the HUM LEAD; it was mis-assigned.
5. **No descope.  FR-7 stays whole — HUM LEAD, 2026-09-07: "better to solve now than later."**
   Nine of eleven lenses recommended deleting most of it, and the offset was recommended and
   declined.  The counter-argument the ruling rests on is sound and was in the recommendation: FR-7.1
   and FR-7.2 are **precedents**, FR-7.1 explicitly sets the pattern six new feeds will copy, and
   0.16.0 is when those feeds arrive — so ruling after Broadcaster starts is ruling too late.  The
   same "cheaper now than twice" logic the release rests on applies to them.

   **The consequence is stated, not absorbed:** scope now runs FR-1 to FR-10 with nothing removed,
   and the largest item in the release arrived after the red team called scope the top risk.  With
   descoping off the table, **the only remaining levers are order and early detection**, so PLAN owes
   two things it would otherwise have been able to skip — a sequence, and a size per FR.  The
   principal-engineer lens named this as the likeliest overrun driver and it is now the live one:
   nine open questions, six of them HUM LEAD rulings that block work, and not one FR carrying an
   estimate or a checkpoint definition.

**Ledger staleness is worse than this report first claimed, and not in the direction it claimed.**
F-35 was understated and is amended; F-30 was correct and this report was wrong (FR-3.3).  The red
team then sampled outside the seven and found **F-8 already fixed** (`httpx/memo.go:213` compares host
*and* port) and **F-10 fixed in 0.14.0**, carried open through two releases — plus F-1's "four
siblings," which this report disproved and did not correct.  All three now corrected.  The "three of
seven" framing is withdrawn: the denominator included an uncontested row and a substituted one, and
the real finding is that **~40 rows have not been audited since 0.14.0** and every row sampled since
has been stale.

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
