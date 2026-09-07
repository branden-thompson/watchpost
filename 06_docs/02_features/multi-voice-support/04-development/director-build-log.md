# Build log — Station Director & the Lineup

**BUILD · SEV-0 · HUM LEAD · FULL TDD.** Plan: `03-architecture-design/director-build-plan.md`.
Requirements: `01-objectives/director-requirements.md` (DR-1…DR-24).

The **tracked** record of what was built, what was measured, and what each mutant proved. The p10
exemptions ledger is gitignored (`.gitignore:12`), so per PL-16 anything ratified at that gate is
mirrored here — this file is the only reviewable copy.

**No claim below has no command behind it** (anti-vacuity rule 5).

---

## Phase 0 — The net, and proof it can catch

**Status: complete.** Three pins through the real entry points, then mutation-validated.

| Task | What was pinned | Where |
|---|---|---|
| T0.1 | The watchlist advance: a live relay advances after `liveDwell`; a synth cycle end advances; a user Stop advances nothing | `app/director_pins_test.go` |
| T0.2 | The duck's single owner: an automatic advance does not lift the alert duck | same |
| T0.3 | The takeover's marquee/voice coupling: the cue precedes the words, **per event** | same |
| T0.4 | Mutation validation of all three | `06_docs/mutants/m48…m54` |

| Mutant | The rule it deletes | Verdict |
|---|---|---|
| m48 | the cue send | CAUGHT |
| m49 | the cue's ordering — it now follows the words | CAUGHT |
| m50 | the release becomes unconditional | CAUGHT |
| m51 | the advance ignores a user Stop | CAUGHT |
| m52 | the dwell arms for any mode | CAUGHT |
| m53 | the dwell restarts on every arm | CAUGHT |
| m54 | **the advance lifts the alert duck** — the UAT-2026-08-30 regression, recreated exactly | CAUGHT |

**Two of these tests were themselves wrong first, and both were found by mutants rather than by
reading.** T0.3's first ordering assertion exempted indices 0 *and* 1 when only the burst head is
index 0, so the first event's words slipped through and **m49 SURVIVED**; it was rewritten to assert
`index(cue) < index(words)` per event. T0.3b was **vacuous**: a pre-cancelled context never enters the
takeover, so "no release" would have been true of a perfect implementation too. It now interrupts
mid-sequence through the log's `onCue` hook, with guards asserting the takeover reached the air and
was genuinely interrupted.

---

## Phase 1 — The pure core

### T1.1 — `ReadRank` on the category registry

`platform/category`. A **third** per-category ordering beside tab order and `Rotation`, because the
three genuinely disagree: a disaster is read before a warning (R-2) while the tab bar places it
lower. `ReadOrder()` walks the ranks from 1 and refuses a gap.

The density check uses a two-value map lookup rather than `Lanes()`'s zero test, because Emergency is
`Category(0)` **and** rank 1 — comparing the value against zero would need a special case to survive,
and a special case in a density check is how the gap gets back in.

| Mutant | The rule it deletes | Verdict |
|---|---|---|
| m55 | the three orderings "harmonised" into one — the tidy-up RD-10 predicts | CAUGHT |
| m56 | Forecasts become readable as an alert | CAUGHT |
| m57 | Warnings outrank Disasters | CAUGHT |

**Re-run at T1.2 and still CAUGHT** (anti-vacuity rule 4: carry the previous round's mutants forward).

### Import guard — `platform/` is a leaf too

Not a plan task; found while placing T1.2's package, and load-bearing once the HUM LEAD ruled the
Lineup into `platform/`. `scripts/lint-imports.sh` was a **textual grep over `modes/` only**, not a
transitive one, so `modes/broadcaster → platform/lineup → domains/radio/cast` would have passed while
violating exactly what the `modes/` rule exists to enforce.

`check_tree platform` was added with a second self-test control, and **proved to fire against a real
violation planted in the tree**, not only against the fixture:

```
lint-imports self-test: both controls fired (OK)
lint-imports: FORBIDDEN import of domains/* from platform:
platform/lineup/probe.go:3:import _ ".../domains/radio/cast"
exit=1
--- restored ---
lint-imports: OK
```

### T1.2 — The Lineup and the Card as pure values

`platform/lineup`. **Placement ruled by the HUM LEAD**: both Observer and a Broadcaster station
schedule against it, and `platform/` is where a third surface could reach it too. **Cards stay pure**
— a card names a `Slot`, and the radio domain maps that to a `cast.Role` and thence to a voice from
the cast settings it already owns (`cast.Resolve`, `app/cast.go:350`); `ReadBy` is the plain display
name resolution puts back, and empty means unresolved.

Three build decisions, each a deviation from a literal reading of the plan, recorded rather than
made quietly:

| # | Decision | Reasoning |
|---|---|---|
| **BD-1** | **The lock is derived, not stored.** T1.2's row lists `locked` as a card field; `Locked()` is a method returning `State == OnAir` | A stored flag is a second carrier of one rule, and the two can disagree. That is precisely how the duck came to be lifted by one spelling of `tune` and not the other (`app/radio.go:121-138`). The lock is enforced in `Lineup.Set`, because an edit is real when it reaches the schedule |
| **BD-2** | **`OnAir → Discarded` is declared**, and the architecture's state diagram is corrected | DR-24 enumerates five exits from the air and four are one transition. With only `Done`, a superseded takeover is unexpressible and the paired release has nothing to hang on. The lock is against **edits**, never this transition — a lock that could hold a card on the air would wedge the station |
| **BD-3** | **Card fields are exported; the Lineup's storage is not** | A Card is what the Operator must see (DR-6), so it reads as data. Editing a local copy of a value harms nobody; `Cards()` hands out copies so a reader cannot become the second writer DR-1 forbids. It also keeps P10-05's meter measuring real logic instead of ten one-line accessors |

**Only the slots 0.14.0 actually produces are declared** — Location Report, Severe-event Read,
Breaking Alert, Burst Head, Transition, Divert Notice. A Marine Report as its own card, an
operator-requested read and a station identification arrive with the Broadcaster surface that
proposes them; a slot nobody proposes is dead code (AP-DEAD-01).

`Propose`'s DR-7 rule names the structural slots **literally** rather than calling
`Slot.textAtStandby()`, because P10-05 counts only call-free conditions.
`TestOnlyAReportComposesItsTextAtStandby` walks every slot in the registry, so the two drifting apart
fails a test rather than passing silently. The same trade is made once more, for the same reason, in
`Lineup.Set`'s lock condition; both sites carry the reason inline.

| Mutant | The rule it deletes | Caught by | Verdict |
|---|---|---|---|
| m58 | the alert rail no longer drains first | `TestTheAlertRailDrainsBeforeTheMainTrack` | CAUGHT |
| m59 | `Next`'s positive state test becomes a negative one | `TestACardOnTheAirIsNotOfferedAgain` | CAUGHT |
| m60 | the ON AIR lock removed | `TestACardOnTheAirRefusesEditsButMayStillLeaveTheAir` | CAUGHT |
| m61 | admission stops being the door | `TestTheLineupTakesAdmittedCardsAndNothingElse` | CAUGHT |
| m62 | the air has one exit | `TestTheCardStatesAdvanceOnlyAlongTheDeclaredPath` | CAUGHT |
| m63 | `Cards()` hands out the storage | `TestAReaderCannotReachTheLineupsOwnStorage` | CAUGHT |
| m64 | `Queue` appends in place | `TestTwoLineupsBuiltFromOneNeverShareStorage` | CAUGHT |
| m65 | DR-7 relaxed — text may arrive at admission | `TestAReportsTextMaterialisesAtStandbyAndNowhereElse` | CAUGHT |
| m66 | a card takes the air with nothing to say | `TestACardTakesTheAirWithItsWordsAlreadyOnIt` | CAUGHT |
| m67 | a card may change what it is under a live identity | `TestACardKeepsItsSlotAndItsOriginForLife` | CAUGHT |
| m68 | the duplicate check asks the wrong question | `TestTheLineupRefusesACardItCannotAddress` | CAUGHT |

**11/11 CAUGHT, each by the test that names its rule** — no mutant was caught only by an unrelated
test, which would have meant the rule itself was unpinned.

#### The instrument was validated, not trusted (anti-vacuity rule 7)

P10-05's density meter reported nothing for `platform/lineup`, and **silence from a meter that never
ran looks exactly like a pass** — this is RD-4, the risk that has already fired twice this release.
So the meter was made to fail on these exact files first: stripping every `invariant.Check` from
`card.go` and `lineup.go` produces

```
P10-05: invariant density 0.52 < 2.0 across 23 changed function(s)
```

The files were restored from a copy and re-verified. The meter sees the package, counts 23 functions
in it, and its silence on the real code is measured coverage.

#### Gates at T1.2

Run **serially on a clean tree**, not package-scoped:

```
go test ./...          44 packages ok, 0 failures
gofmt -l .             clean
go vet ./...           clean
go mod tidy -diff      clean
lint-imports           self-test fired both controls, then OK
a2dh p10 check --json  20 findings, ALL exempted, 0 live; 7/7 tools RAN
p10-unmatched.sh       0 unmatched (103 dormant, outside the diff)
git status             clean
```

**No exemption was added by this task.**

### T1.3 — The planner

`platform/lineup/plan.go`. `Plan(arrivals, settings, now) → Burst` is **pure** (DR-2): the same inputs
produce the same lineup, and — the stronger property, and the one actually tested — the arrival order
does not reach the output at all. The clock is a parameter because the freshness rule is the whole
reason DR-20 asks for one; no assertion path reads a real clock.

**The ladder is derived, not written out again.** The thirteen ratified rungs compose from
`category.ReadOrder()` and the two bands, so the Director does not become a fifth list of categories
(NFR-D-6 — F-21 is what happens when it does). The Remaining band is one rung shorter because
**Emergency Orders are never demoted**: only Disasters carry the freshness rule, so there is nothing
that could sort an evacuation order below a warning and no rung for it to sort to.

**Emergency orders lead and spend the budget** (DR-11, Q-1), in the HUM LEAD's own two cases at Max 5:

| Arrivals | Read | Diverted |
|---|---|---|
| 1 emergency + 20 warnings | the emergency **and four warnings** | 16 |
| 7 emergencies + 18 warnings | **all seven**, nothing else | 18 |

The overrun is bounded **by the feed**, never by a constant: the read is a `range` over an
already-capped slice (`globalfeed.MaxPerLane` = 30), which is P10-02's required form rather than an
exemption.

#### Two readings recorded rather than assumed — HUM LEAD, for ratification

Both are places the ruling does not quite reach, and counts and ordering are HUM LEAD's. Each is
implemented in the direction stated below, tested, and mutation-checked; either can be reversed in a
line if the ruling goes the other way.

| # | The question | Implemented as | Why this direction |
|---|---|---|---|
| **BD-4** | **Does a Forecast count as a diverted alert?** DR-14 says the count is `arrivals − alerts read` but does not say which side a forecast falls on | **Not read and not counted.** A burst of one forecast and one warning that reads the warning reports **0** diverted | The notice is spoken: *"…these and N other **alerts**."* An outlook the listener never lost is not one of them, and it stays part of the location report, which is most of the broadcast |
| **BD-5** | **Is a disaster exactly 24 hours old fresh?** R-5 ratified the window, not its boundary | **Inclusive** — at exactly the window it is still fresh; past it, it sorts down | *"Within 24 hours"* reads that way, and it is the direction that demotes a hazard later rather than sooner. Both sides are tested, and the boundary has its own mutant |

**Rungs 9–13 are unreachable in 0.14.0 by construction**, and that is not an oversight. Only Disasters
are ever demoted, so a Remaining Warning has no producer. The ladder declares all thirteen because the
ratified table does, and because the close threshold that would fill them is Broadcaster-era: G-7
already says the split *"simply collapses where the fence is tighter than the threshold"*, which today
it always is.

Within a rung the order is **today's, unchanged** (`app/ticker.go:542 byBreakingPriority`): severity,
then recency — then the identity, which is what makes the order **total**. Without that last term two
alerts alike in every dimension keep whatever order they arrived in, and the arrival order reaches the
listener.

**The golden** (`platform/lineup/testdata/plan.golden`) is one fixed arrival set spanning every
readable category, both bands, the forecast exclusion and the Max boundary. It shows the ruling
working: `slide-stale` carries severity 80 against `quake-fresh`'s 70, and is nonetheless read
nowhere, because 96 hours with no fence puts it on rung 8 below everything close.

| Mutant | The rule it deletes | Caught by | Verdict |
|---|---|---|---|
| m69 | the ladder composed Remaining-first | `TestTheShippedOrderReproducesTheRatifiedLadder` | CAUGHT |
| m70 | emergencies stop spending the budget | `TestEmergencyOrdersLeadAndSpendTheBudget` | CAUGHT |
| m71 | the overrun removed — emergencies bounded by Max | same | CAUGHT |
| m72 | the divert count taken from raw arrivals | `TestForecastsNeverEnterABurst` | CAUGHT |
| m73 | the freshness rule dropped | `TestDisastersVsWarningsIsConditionalOnTheFence` | CAUGHT |
| m74 | the 24-hour boundary tightened by one instant | `TestTheFreshnessBoundaryIsTwentyFourHours` | CAUGHT |
| m75 | the tie-break by identity dropped | `TestTiesAreBrokenByIdentity` | CAUGHT |
| m76 | recency and severity swapped inside a rung | `TestWithinARungTheWorstIsReadFirst` | CAUGHT |
| m77 | the rung-zero skip dropped — a forecast becomes a takeover | `TestForecastsNeverEnterABurst` | CAUGHT |
| m78 | the freshness rule lands on Warnings instead of Disasters | `TestDisastersVsWarningsIsConditionalOnTheFence` | CAUGHT |

**10/10 CAUGHT — but m71 came back `INVALID` first**, and is recorded rather than quietly fixed.
Deleting only the `!emergency` term left the variable declared and unused, so the tree did not
compile: *not evidence either way*, which is exactly the verdict the harness exists to distinguish
from a pass. This is the second time this release a mutation has failed that way (m51 was the first),
and both times the tell was a mutation that removed a *use* rather than a *rule*. The mutation now
removes the emergency branch entirely, which is the overrun genuinely being deleted.

**T1.2's eleven carried forward and all still CAUGHT** (anti-vacuity rule 4).

#### Gates at T1.3

Serial, clean tree: **44 packages ok**; **44 tests and 26 subtests ran, none skipped** (checked
explicitly — a golden test that silently skips for a missing fixture would pass); gofmt, `go vet`,
`go mod tidy -diff` clean; lint-imports self-test fired both controls then passed; p10 **20 findings,
all exempted, 0 live, 7/7 tools RAN, 0 unmatched**. **No exemption added.**

### T1.4 — The fence

`platform/lineup/fence.go`, **its own commit** (RD-5), with nothing structural riding along.

`Fence.Admits` governs **entry** (DR-13): what it keeps out is not read, not counted and not pointed
at. The rule, in order — no radius admits everything (`0 = All`, the default path); a radius with no
origin admits **nothing**, which is today's behaviour rather than a silent fallback to the global
stack the UI says is scoped away; a zone-only alert with no point gets in only by the tracked tie,
said the same way `scopeEvents` says it; then the radius; then the exception.

The exception is for a **Disaster**, because such a disaster has proximal effects. An evacuation order
is exempt from the Max, **not** from the radius (R-2). Cascades need no modelling: the tsunami warning
behind that M9.0 arrives as its own local alert inside the fence.

**`Settings.Fenced` is gone**, replaced by `Settings.Fence`. Whether a radius is in force was a
boolean kept beside the radius that could disagree with it; it is now one question with two askers —
admission, and DR-12's ordering rule.

#### BD-6 — the significance scale, RATIFIED by the HUM LEAD 2026-09-02

`QuakeReachMi` doubles per magnitude point.

| M5.5 | M6.5 | M7.0 | **M7.5** | M8.0 | M8.5 | **M9.0** | M9.5 |
|---|---|---|---|---|---|---|---|
| 32 mi | 64 mi | 91 mi | **128 mi** | 181 mi | 256 mi | **362 mi** | 512 mi |

The margin was put up before ratification — an M7.5 reaches 128 mi against the ruling's ~120, eight
miles of headroom — with the question that decides it: was ~120 an example or a floor?

> **"120 was an example to illustrate the correlation between distance and urgency / importance."**

**An example.** So the scale stands as built, and what the code now records is the *principle* rather
than the number: **a hazard matters less the further away it is** — which is what the fence encodes —
**and a big enough hazard pushes the point at which it stops mattering further out.** Reach is how far
one hazard's own significance carries that, and doubling per magnitude point is the shape a
logarithmic scale asks for.

The 120 is still asserted, and the reason is worth stating because it is not "the ruling requires it":
**a scale that cannot reproduce the example the rule was explained with is not encoding the rule.** A
change that moved it would be a change to the shape rather than a broken contract, and the failure
message says exactly that so the next reader is not told a number is binding when it is not.

#### The fixture is not the ruling

Los Angeles measures **79 mi** from Bonsall by great circle. The ruling says "~120 mi", which is what
a person means by how far away Los Angeles is. The first test claimed the LA fixture *was* the ~120,
which would have been a false statement in a comment and a scale pinned eight times looser than it
looked. The scale is now pinned against **the number**, with the Los Angeles fixture asserted beside
it.

#### The mutants found two rules unpinned, and one bad invariant

| Mutant | The rule it deletes | Verdict |
|---|---|---|
| m80 | the fence stops governing entry | CAUGHT |
| m81 | reach granted to every category | CAUGHT *(INVALID first)* |
| m82 | the magnitude ceiling removed | CAUGHT |
| m83 | reach below the strong-quake threshold | **SURVIVED first** |
| m84 | a fence with no origin falls back to the global stack | CAUGHT |
| m85 | point-less alerts walk through without the tracked tie | CAUGHT |
| m86 | the radius boundary becomes exclusive | **SURVIVED first** |
| m87 | reach compared in kilometres against miles | CAUGHT |

**m83 — an invariant was a second carrier of the rule it checked.** `QuakeReachMi` guarded "below M5.5
there is no reach" and then asserted `reach >= quakeReachBase`, which is the same sentence. Deleting
the guard left the invariant returning 0 in its place, so the deletion changed nothing observable.
**A check that can stand in for the rule it is checking is not an invariant** — it is the rule written
twice, with a recovery path that hides the first copy going missing. This is the P10-05 meter's own
documented limit ("two mechanical nil-checks satisfy the meter") arriving in a real function.

The rule was also **invisible against a wide fence**: an M5.0's would-be reach is 23 mi, inside a
50 mi radius anyway, so no test built on the listener's own fence could ever see it. It is visible
against a small service area — the case a Broadcaster station actually has — and that is now the
fixture, with its own control: the same quake one notch above the threshold *does* cross, so the
refusal is the threshold and not the fixture.

**m86 — the boundary test compared the fence against itself.** Asserting that `Admits` agrees with the
same `<=` expression cannot catch that expression being flipped, because both sides flip together.
The fence is now built **from** the fixture — the radius *is* the measured distance — and the test
asserts the km/mi round trip is exact before it asserts admission, so it fails loudly rather than
pinning nothing if that ever stops holding.

**m81 — INVALID first**, the third of that kind this release (m51, m71), and the same shape each time:
the mutation removed a **use** rather than a **rule**, leaving an identifier or import unused. It now
keeps the condition and makes it unreachable, which is how a rule usually stops applying in practice.
**Three occurrences is a pattern, not three accidents** — a mutation that deletes a `return` or a
term should be written as one that makes it unreachable.

**A ninth mutant was written and deleted before commit.** "Fenced-out alerts are counted as diverted"
is character-for-character the same edit as m72, which already exists for the forecast reading of the
same line. Two mutants applying one patch measure one thing twice and report it as two.

**m58–m79 carried forward: 22/22 still CAUGHT.**

#### Gates at T1.4

Serial, clean tree: **44 packages ok**; **55 tests and 26 subtests, none skipped**; gofmt, `go vet`,
`go mod tidy -diff` clean; lint-imports both controls fired then passed; p10 **20 findings, all
exempted, 0 live, 0 unmatched**. **No exemption added.**

### T1.5 — `Step`, and the event/effect vocabulary

`platform/lineup/director.go`. `Step(Event) → (Director, []Effect)` is pure: no I/O, no clock, no
goroutines. A test states the events and reads back the work, with nothing to synchronise with
(NFR-D-1) — the property three previous attempts could not get, and the reason every fixture written
against them was vacuous. The dispatch is a small switch; the work lives in one handler per event,
**from the first commit** rather than after P10-04 complains.

**DR-7 is structural, not scheduled.** `settle()` takes the air *first* and prepares next *second*, so
the following card's build starts in the same step the current one takes the air — not when it
finishes, which is exactly one build too late. The 1.03 s finding is answered by the order of two
lines rather than by a timer.

Three rules Phase 0 had to pin by hand are now properties of a returned list, which no call site can
get wrong:

| Rule | How it stopped being a discipline |
|---|---|
| The cue precedes the words (DR-18, T0.3, m49) | It is the order of the slice `takeTheAir` returns |
| Every exit from the air releases the band (DR-24) | `onFinished` and `onFailed` both go through `leave()`, the only exit a card has — and it releases only if it cued, because the pairing is with the **cue**, not the card |
| One card holds the air | `Lineup.OnAir` is **derived** from the cards, so the Director has no second carrier to disagree with |

**The effect set is closed and enumerated** (PL-6) before Phase 2 starts. `Duck`, `Restore` and `Tune`
are declared and **not yet emitted** — the bed is a selectable resource rather than a queue, and they
arrive with T1.6 and T3.2. Each type records that, so its absence reads as a schedule rather than an
oversight.

#### Two design calls the density gate pushed on, both taken on the merits

The P10-05 meter opened at **1.27 across 73 changed functions**. It counts Go's interface ceremony as
if it were logic, and this is a pure-value package: many small types, each needing a marker method and
a renderer. The temptation was to pad. What was done instead:

| Change | Why it is right independently of the meter |
|---|---|
| `Describe(Effect)` replaced eight `String` methods | **One owner for the format.** DR-23 asks for a timeline that reads as one timeline, and a format defined in eight places drifts in eight places — F-21 is what that costs. It also collapsed four character-identical bodies, which were plain repeated code |
| `isEvent` / `isEffect` embedded markers | The closed sum type now costs **two** functions instead of thirteen, at the same strength: a mistyped event is still a compile error, not a silent no-op on the hazard path |

**The markers' cost was measured, not estimated.** Removing all thirteen and re-running the gate
reported **1.93 across 54 functions** against 1.55 across 67 — so they were 0.38 of the gap, and
dropping them for `any` (bubbletea's shape) would still not have reached 2.0 while giving up
compile-time closure. Keeping them was the right call and the measurement is what said so.

The last four invariants were chosen for being **load-bearing**, not cheap: a clone that lost a track;
a card that left the schedule but is still held; the readers being told last; and — the best of them —
**every candidate was either read or diverted**, which is the arithmetic the listener is actually told,
stated as a conservation law instead of trusted to a subtraction. Final: **2.0+, no exemption added.
The gate was met, not negotiated with.**

#### Mutants

| Mutant | The rule it deletes | Verdict |
|---|---|---|
| m90 | the words precede the cue | CAUGHT |
| m91 | the next card waits for this one to finish | CAUGHT |
| m92 | the release is conditional again — only a full read frees the band | CAUGHT |
| m93 | the release fires without a cue | CAUGHT |
| m94 | a second card takes the air | CAUGHT |
| m95 | publish comes first | CAUGHT |
| m96 | a card is built twice | CAUGHT |
| m97 | the clock runs backwards | CAUGHT |
| m98 | a stranger's completion disturbs the schedule | CAUGHT |
| m99 | an arriving burst replaces the rail | **SURVIVED first** |

**m99 — DR-3's own guarantee was unpinned, and it defeated the post-condition meant to catch it.**
Clearing the rail before queueing a new burst is `breakingCap`'s defect wearing a new hat: cards
already promised a read, dropped unread. `onArrived` already asserted *"queueing a burst never
shortens the rail"* — and the mutant walked past it by clearing the rail **before the count was
taken**, which makes the assertion true.

**An invariant the mutation can reorder itself around is not a pin**, and no amount of strengthening
that particular check would have helped: the rule needed a test that names what was promised and looks
for it afterwards. That is now `TestABurstArrivingWhileTheRailDrainsAddsToIt`, and it also covers the
burst-during-burst case Q-2/Q-3 rule on — each burst evaluated on its own, the ladder deliberately not
built.

**Attribution was checked, not assumed.** `run.sh` reports only the first failing test, so m94, m96 and
m98 were each re-run against the single test named for their rule, and each was caught by it. A mutant
caught only by an unrelated test means the rule itself is unpinned and the pass is a coincidence.

**m58–m87 carried forward: 30/30 still CAUGHT.**

#### Gates at T1.5

Serial, clean tree: **44 packages ok**; **71 tests and 28 subtests, none skipped**; gofmt, `go vet`,
`go mod tidy -diff` clean; lint-imports both controls fired then passed; p10 **20 findings, all
exempted, 0 live, 0 unmatched**. **No exemption added.**

### T1.6 — The running state

`platform/lineup/power.go`. PD-1: the main track does not advance while the listener has stopped the
radio. Today that rule is `d.mode != ""` **plus** a tune epoch — one rule in two places, which is the
shape that produced the duck-lift bug (RD-2). Here it is one field and one predicate, and
`advances(Track)` is the only thing that asks.

**An enum, not a bool, and that is the seam.** ON AIR / STANDBY is a second reason to be off, so a
Broadcaster station adds a value rather than a mechanism. **`Stopped` is the zero value**, because that
is how a station starts — today's empty mode. A director that came up running would put a report to
air nobody asked for.

#### The asymmetry, which is today's behaviour and not a change

**Stopping the radio stops the programme, not the hazards.** `Stop` halts the engine, which silences
the broadcast, while a takeover reads through the narrator's own path and is never asked about the
deck's mode (`app/ticker.go:390` asks only whether the voice is silent or muted). **Mute** is the
control for *"do not speak to me"*; **stop** is for *"do not play me a programme"*. So a stopped radio
still reads the alert rail, and the rail keeps draining — nothing admitted is dropped unread (DR-3).

Stopping takes the programme off the air at once and releases the band with it. It is **not** a
discard: the rest of the rotation waits. And while stopped the main track is not **built** either — a
build costs 1.03 s of network, and a stopped programme has no cutover for it to be ready for.

#### A T1.5 test was rewritten rather than patched

`TestTheRailDrainsBeforeTheMainTrackThroughStep` expected the report's build *after* the alert
finished. The build had in fact started when the radio did — DR-7 working. The replacement asserts the
sharper property it was reaching for: **a report that is READY still waits behind an alert that is
not**, because `takeTheAir` looks at the **head** of the queue and stops if it is not ready rather than
looking past it for something that is. Skipping ahead would read the schedule out of order, which is
the one thing the rail exists to prevent. That property now has its own mutant (mA8).

#### Mutants

| Mutant | The rule it deletes | Verdict |
|---|---|---|
| mA0 | the programme runs while stopped | CAUGHT |
| mA1 | stopping silences the alert rail too | CAUGHT |
| mA2 | stopping cuts an alert short | CAUGHT *(INVALID first)* |
| mA3 | stopping leaves the programme on the air | CAUGHT |
| mA4 | a new director comes up running | CAUGHT |
| mA5 | the stopped programme is still built | CAUGHT *(INVALID first)* |
| mA6 | a repeated command acts twice | **SURVIVED first** |
| mA7 | an unknown power reads as running | **SURVIVED first** |
| mA8 | the head of the queue is skipped for something ready | CAUGHT |

**mA6 — the test asked too little.** It checked that a repeated command produced no *release*, when
what it must produce is **nothing at all**. Without the guard the handler still publishes, and every
effect is dispatched — so a no-op command re-notifies every subscriber about a lineup that did not
change, on every idle keypress.

**mA7 — the guard is unreachable through any public path**, because `onPowered` already refuses to
store an out-of-range power. It is now pinned where it *can* be observed, by setting the field
directly in-package, and the test says so plainly: **a guard nothing can exercise is decoration rather
than defence**. It is kept because `advances` is the single predicate deciding whether programme goes
to air, and one guard should not be the only thing between a corrupted state and audio nobody asked
for. **Recorded rather than quietly justified: the guard was added while meeting the density gate, and
it earns its place on the fail-closed argument, not on that one.**

#### The INVALID pattern is now five, and it has a fix

m51, m71, m81, mA2, mA5. **Every one removed a *use* rather than a *rule*** — deleting a `return` or a
term left a variable or an import unused, so the tree did not compile and the run reported `INVALID`:
*not evidence either way*, which is the verdict the harness exists to distinguish from a pass.

**A deletion mutant should make the rule UNREACHABLE, keeping the identifiers that hold the tree
together** — `if x == neverPossible` rather than deleting the branch. Five occurrences is a habit, not
five accidents, and it is worth carrying into the round-2 review as a rule about how mutants are
written.

**m58–m99 carried forward: 40/40 still CAUGHT.**

#### Gates at T1.6

Pre-commit, clean tree: `make verify` **ALL GATES GREEN**; `make alloc-budget` **exit 0**; p10 **20
findings, all exempted, 0 live, 0 unmatched**; **80 tests and 28 subtests, none skipped**; 44 packages
ok. **No exemption added.**

Density held at 2.0+ without padding: `Power.String` is written the way `Track.String` is — two members
do not need a registry, and a table only one caller reads is a table nobody can see is complete.

---

## Phase 1 — complete

| # | Task | Mutants |
|---|---|---|
| T1.1 | `ReadRank` on the category registry | m55–m57 |
| T1.2 | The Lineup and the Card as values | m58–m68 |
| T1.3 | The planner | m69–m79 |
| T1.4 | The fence | m80–m87 |
| T1.5 | `Step` and the vocabulary | m90–m99 |
| T1.6 | The running state | mA0–mA8 |

**53 mutants over the pure core, all CAUGHT.** Five survived first and were closed; five came back
INVALID and were rewritten. No goroutines, no sleeps, no real clock anywhere in the assertion path.

**Next: Phase 2 — the pump and the executors.** Its exit condition is that every existing golden,
declset and PTY journey is green and unchanged, and a **second red-team round follows it** (Step 9).

---

# Phase 2 — The pump and the executors

Wiring only. At the end of this phase the Director exists and runs, and the listener hears exactly
what they heard before.

### T2.1 + T2.4 — The pump, supervised

`app/pump.go`. **Built as one commit, deviating from the plan's task boundary, and the reason is
PL-2.** That finding was a **Critical**: under Approach C the pump is the single owner, so an
unguarded panic takes both tracks and the bed — and in Go an unrecovered panic on a goroutine kills
the *process*, not just the surface. A pump that exists unsupervised even for one commit contradicts
the disposition that closed the finding. The two tests the plan asks for are both delivered.

**It lives in `app/`** because it connects the pure core to the executors and both are here. A generic
pump package with one consumer would be a package for its own sake; if Broadcaster needs its own, that
is a move, not a rewrite.

#### The rule: it never blocks (PL-1)

`dispatch` hands a step's effects to workers and returns; the loop goes straight back to the select.
The earlier draft of this loop ran each effect inline, which would have paid the 1.03 s card build
**on the pump** and reintroduced the blocking class inside the approach chosen to avoid it. The test
holds one effect open indefinitely and requires the schedule to carry on around it.

#### BD-7 — effects about one card keep Step's order; everything else runs at once

**A design decision the plan did not anticipate, and it is load-bearing.** DR-18's "the cue precedes
the words" is guaranteed by `Step` **as a list order**. Dispatching every effect to its own goroutine
would throw that guarantee away the moment it left the pure core — the band would promise a callout
after the read had already started, which is the exact defect Phase 0 pinned by hand (T0.3, m49).

`byCard` groups *consecutive effects naming the same card* into one serial run and leaves everything
else free. That preserves precisely the ordering that means something and constrains nothing else, so
the next card's 1.03 s build never waits behind a read. `lineup.CardOf` is the single owner of "which
card does this effect concern", and the pump uses it for both questions it has — what must stay
ordered, and which card to fail when an executor panics.

**It carries its own control.** "The cue precedes the words" would *also* be satisfied by a pump that
ran everything in sequence — which would pass that assertion and fail the one that matters.
`TestEffectsAboutDifferentCardsRunAtOnce` holds a read open and requires the next card's build to
proceed regardless, so the grouping cannot quietly become a global lock.

#### Supervision (DR-22)

`contained` recovers, reports **every** containment to the fault sink, and additionally **fails the
card** when the effect named one — so the schedule re-plans rather than waiting for a completion that
will never come. A panic in an effect that names no card (a publish) is recorded and fails nothing:
inventing a card would take a read off the air for a fault that never touched it. The recover is
scoped to the one function, following `platform/config/keep.go`'s idiom, and the comment says what it
must never swallow.

The test asserts what happens **after** the panic — the card leaves the air, the band is released, the
next card is built, and the pump still answers a later event — not merely that nothing crashed.

#### Two of my own tests were wrong first

| Test | What was wrong |
|---|---|
| The drain test | It asserted `stop` blocks while an effect is in flight — but the fake executor honoured cancellation, so after the cancel **nothing was in flight** and the test proved nothing. It now uses a hold that ignores the context, the way a write already under way does, which is the case R5-B-07 exists for |
| The panic test | It waited for a **count** of effects and then asserted on that snapshot — a race dressed as a test, and it failed on the first run because the release had not arrived yet. `awaitContains` waits for the named effect instead |

Both were found by running, not by reading. The second is the more useful lesson: **waiting for a
count and asserting on the snapshot asserts against whatever happened to have arrived.**

#### The declset moved, additively

`app/testdata/declset.txt` gained eleven symbols and lost none — `removed: []`, which is the half
RD-8 actually cares about. Phase 2's "every golden unchanged" exit condition is about *behaviour*
goldens; a declaration set that records `app`'s symbols must change when a pump is added, and T2.3
will assert the same property across `app/director.go`'s split.

#### P10-02 exemption — RATIFIED by the HUM LEAD, 2026-09-02

**The ledger is gitignored (`.gitignore:12`), so this is the only reviewable record (PL-16).**

```
app/pump.go · loop · P10-02-BOUNDED-LOOPS
```

> The Director's pump (0.14.0 T2.1): a select on `ctx.Done()` (stop) and the event channel — an
> unbounded event loop by nature, ended by context cancellation, the same shape and the same reason as
> the `app/ticker.go` `run` row ratified at the global-ticker P3 gate, the `platform/sched` tier loops,
> and the `app/release.go` `start` row. There is no meaningful iteration count to bound it by: it runs
> for as long as the dashboard does. **RATIFIED by the HUM LEAD, 2026-09-02.**

**No honest alternative form exists, and that was checked before asking rather than asserted.** A
`range` over the event channel needs the channel closed while producers may still send — a
send-on-closed panic waiting to happen. A condition-only `for ctx.Err() == nil` is flagged in its own
right (`app/director.go:281` is exactly that, and is itself exempted). A counter would be a fabricated
bound on a loop that genuinely does not have one, and faking a bound to turn a gate green is worse
than the finding.

**This is the first exemption added by the Director work.** T1.1–T1.6 added none.

#### Mutants — and three findings in the instruments, not the code

| Mutant | The rule it deletes | Verdict |
|---|---|---|
| mB0 | effects run inline on the pump | CAUGHT |
| mB1 | the grouping inverted — a cue and its words race | CAUGHT *(after the pin was fixed)* |
| mB2 | everything joins one serial run | CAUGHT |
| mB3 | the panic is not contained | CAUGHT *(by the harness fix)* |
| mB4 | a contained panic fails nothing | CAUGHT |
| mB5 | a card-less panic fails a card | CAUGHT *(after two rewrites)* |
| mB6 | stop abandons effects in flight | CAUGHT |
| mB7 | a fault goes unreported | CAUGHT |

**Finding 1 — the harness had a wrong verdict in it, for the whole release.** `run.sh` decided CAUGHT
by grepping for `--- FAIL`, but **a panic on a goroutine takes the test process down**, so no test ever
prints that line. mB3 — the pump's `recover` deleted — was reported **SURVIVED**. It is the inverse of
the false RED the script was written to prevent: it *understates* coverage. Still a wrong verdict, and
a SURVIVED drives real work. The exit code is now the signal, since `go build` and `go vet` already
gate a bad tree above that point. **Validated in all three directions before being trusted**: CAUGHT
still fires (m58, and that branch is tested first so no existing verdict moves), SURVIVED still fires
(a control edit nothing asserts on), INVALID untouched.

**Finding 2 — a concurrency pin was probabilistic.** mB1 splits a cue and its words onto separate
workers. `TestEffectsAboutOneCardKeepTheOrderStepGaveThem` compared their positions in the executor
log, and under mB1 it **passed at `-count=1` and failed at `-count=20`** — catching the race only when
the scheduler went the wrong way.

**A pin that reports CAUGHT or SURVIVED depending on the scheduler is not a pin**, and on the hazard
path it is the worst kind: it will be green on the run that matters. It now asserts by **blocking** —
the cue is held open and the words must not have been asked for — which cannot pass unless the
grouping is intact, whatever the scheduler does. **20/20 caught after the change.** Nothing about
reading the test suggested this; only the mutant did.

The pure-function pin (`TestByCardGroupsOnlyWhatMustStayInOrder`) was already sound and catches mB1 on
its own. The gap was specifically the claim that involves real goroutines.

**Finding 3 — a misfiring mutant found dead code.** The rewritten mB5 anchored on `if !ofCard {`,
which appears **twice**; it patched `byCard`'s copy instead of `contained`'s and reported SURVIVED
against a rule it was not testing. The misfire was worth having: `byCard`'s
`if !ofCard { last = "" }` **could not fail**, because `CardOf` returns `""` for every card-less
effect, so the line only re-assigned what `last = id` had just assigned. Dead code (AP-DEAD-01),
removed, with the reason kept where it stood so nobody restores it for symmetry.

Re-anchored, mB5 then exposed the real gap: removing `contained`'s guard emits `Failed{ID: ""}`, which
the Director ignores because `find("")` holds nothing — **the rule is enforced in two layers and only
the far one was observable.** The guard is kept, and not for symmetry: an event naming no card is
garbage on the wire, it would appear in the DR-23 timeline as a failure of nothing, and its
harmlessness depends on a detail of the Director's lookup the pump does not control. It now has a test
asserting the **pump's own** contract, with a control for the card-carrying case.

**Attribution was checked per-mutant, not taken from `run.sh`**, which reports only the first failing
test — that is how Finding 2 surfaced at all.

#### One unexplained failure, recorded rather than dismissed

During this work a run of the concurrency set **failed once** and took 50 s where it normally takes
0.7 s, consistent with the 5 s `awaitLog` deadlines expiring. It has not recurred: **70 consecutive
runs since, 20 of them under `-race`, all green.** It was not reproduced and no cause is claimed.

Flagged for the round-2 red team, because a deadline-based wait that fires once is the same shape as
the probabilistic pin mB1 exposed — and this file's assertions are the ones standing between the
schedule and a silent station.

#### Gates at T2.1

`make verify` **ALL GATES GREEN** · `make alloc-budget` **exit 0** · p10 **exit 0, 21 findings, all
exempted, 0 live** · **0 unmatched** · `go test ./app -race -count=10` on the pump tests **clean** ·
44 packages ok.

### T2.1 — the verification the disconnect cut short (2026-09-02)

The previous session ended without notice after the T2.1 verdicts commit (6083559). Two things
every earlier task recorded are absent from that section: the carried-forward re-run of the previous
mutants, and any smoke. Both were run here against the committed tip on a clean tree, with the
T2.2 work in progress parked in a named stash so the harness could have the tree.

#### Gates at 6083559 (clean tree, serial)

| Gate | Result |
|---|---|
| fmt · vet · tidy · vuln · lint-imports · lint-watermark · gate-controls | green |
| `go test -race -count=1 ./...` | **one FAIL on the first run**, then 4/4 green; a further `-race -count=5 ./app` (375 s) green — nine clean runs since, the failure unreproduced |
| `make alloc-budget` | exit 0 |
| p10 | 21 findings, all exempted, 0 live, 0 unmatched — unchanged from T2.1 |
| live relay (macOS half of R6) | PASS — wxradio WXL78 and weatherUSA EW6308 both play; five weatherUSA mounts 404 and the engine falls through, as designed |
| `make pty-severe` | **RED, then green** — F-D3 below |
| validate-journey (fresh HOME) | 19 PASS / 7 FAIL — all seven attributed to stale 0.13.0 assertions, below |

**The race-gate failure is recorded, not explained.** Its output was lost to the operator's own
`tail`, so nothing can be said about which test. It is the same shape as the one T2.1 flagged (a
deadline-based wait firing once), now seen twice on two days. Ten runs of the app package under
`-race` since have all been green. Flagged for the round-2 red team beside T2.1's; the two together
are a pattern, not two accidents. **Every gate's full output is captured from here on** — a gate
that is tailed is a gate whose one failure cannot be read.

#### F-D3 — the severe-window pty smoke had been red since the rename

`make pty-severe` failed with *"w did not open the window"*. The transcript shows the window opening
on `w` — under its 0.14.0 name, **NOTABLE EVENTS AND FORECASTS** (`modes/tty/severe.go`
`severeWindowName`, renamed at 349401c on 2026-09-01). The script asserted the old title three
times. The last recorded green run is the P5 build log (2026-08-31), the day before the rename, and
**no gate table on the Director branch (T0.1 through T2.1) listed the smoke**, so it stayed red for
eleven commits unnoticed. Fixed at 202cedc; green end to end afterwards.

**The lesson is the gate table, not the title.** `README-director-handoff.md` names `make
pty-severe` as a gate; the per-task tables above did not carry it. A smoke absent from the table
is a smoke nobody runs. From this task on the table names every smoke.

#### The journey's seven failures, attributed

The fresh-HOME journey (`validate-journey.expect`, the 0.13.0 VALIDATE core tier) is a
HUM LEAD-owned validation artifact; it was **run, not edited**. Each failure was reproduced against
the current UI strings:

| Step | Expects | The screen shows | Attribution |
|---|---|---|---|
| P0 first data (a temperature) | `\d+ºF` (U+00BA, ordinal) | `75°F` (U+00B0, degree) | ca96fd9 "the degree sign" — stale glyph; the 180 s timeout is spent on a character the UI no longer draws |
| P0 w opens the severe window | `SEVERE` | the window, open, titled NOTABLE EVENTS AND FORECASTS | 349401c — stale title |
| P0 the tab row | `Warnings\|Watches` | the window's rows | cascade: expect's 2000-byte buffer had scrolled past the tab row during the previous timeout; "right moves to Watches" PASSED immediately after, which needs the tab row |
| P1 the Sig. Quakes tab | `Sig. Quakes\|Quakes` | the Disasters tab | the category registry's ratified label — stale |
| P1 t opens the theme chooser | `Color Theme` | Setup, footer "←→ Theme (live)" | `[T]` retired into Setup (P4) — stale by ruling |
| P1 Status gauges the severe index | `severe index\|rows / 500` | `SEVERE INDEX … n/500` | e15f41c "[S]: three aligned tables" — stale label |
| P1 the cast is on and a voice is picked | `Correspondent Cast` | WATCHPOST RADIO - CORRESPONDENTS, five pickers | P4 Setup redesign — stale |

**No defect was found by the journey.** Its assertions need re-authoring against 0.14.0's ratified
strings — VALIDATE's work, and the "F-6 M2 journey re-run" the handoff already lists as owed.
Recorded so VALIDATE starts from an attributed list rather than a red run.

#### Every mutant in `06_docs/mutants`, re-run — 92 of them

Scoped to the package(s) each mutant patches, which is where the log's own attribution says each
rule is pinned (T1.2, T1.5). **89 CAUGHT · 3 SURVIVED · 0 INVALID · 0 UNAPPLIED.** m48–mB7, the
whole Director series, all still CAUGHT.

| Survivor | Disposition |
|---|---|
| m16 `scopeEvents`' `ok` | **Deliberate, recorded** — red-team-build-round2.md and the comment at the guard both say it is defence in depth that cannot be observed to fail |
| m43 the Marine arm narrowed | **Deliberate, recorded** — fb65c51: across all 111 products every other marine product is decided above, so no test can tell the two apart |
| m42 the Air Quality Alert matched loosely | **F-D4 — a pin hollowed by a later commit.** See below |

#### F-D4 — m42's pin passes without proving anything

fb65c51 pinned "the Air Quality Alert is matched EXACTLY, not by `Alert`" with **Blue Alert** as the
control, which the classifier then refused — a real pin. 50c7c84 (MVS-D-60) routed Blue Alert to
Statements in the exact-match block **above** the sweep and replaced the control with **Avalanche
Warning**, which the Warning arm decides before the Air Quality arm is reached. From that commit the
test passes against `strings.Contains(product, "Alert")` as readily as against the exact match, and
the comment beside it still claims exactness. Re-run against every package (`./...`): still
SURVIVED, so no other pin covers it.

No product in the catalogue containing "Alert" now reaches that arm, so the exactness is
**unobservable across the catalogue** — the same situation fb65c51 recorded for m43, and an
invented product would be exactly what that commit refused. **Proposed disposition, for
ratification:** the same as m43 — the mutant is recorded as a deliberate survivor, the test's
comment stops claiming what it cannot prove, and the equivalence is written at the code. It is the
"asserted what the code does not do" failure mode from `06_docs/remediation-review-loop.md`, wearing
a test comment instead of a commit message.

### T2.2 — The alert rail's executors

`app/executors.go`. Each executor is an adapter over the code that already works, and the set
performs nothing the Director did not describe. **The closed set is walked in full**: every member
is performed or declined by name with the task that brings its executor, and nothing falls through
to a default.

| Effect | Executor | Comes home as |
|---|---|---|
| `BuildCard` (Breaking Alert) | today's `breakingLine` over the event the producer holds, composed at standby against the injected clock (DR-7, DR-20) | `Built{id, text}` · `Failed` when the producer cannot account for the card or the script renders nothing |
| `Speak` (the rail's slots) | today's narration arbiter, in the takeover's class — the duck keeps its one owner across Phase 2 (RD-2); with no voice the fixed hold still elapses so the callout is readable (P4 F10) | `Finished{id}` — or nothing when the context ended mid-read: the pump is stopping, and a Finished for words never finished would tell the schedule a read happened that did not |
| `CueTicker` · `ReleaseTicker` | the existing takeover messages, fire-and-trust, with a bounded post-hoc record (DR-18, DR-24) | nothing; a band that cannot be cued is reported and the words still read |
| `Publish` | the readers | nothing |
| `Duck` · `Restore` · `Tune` | **declined** — T2.3, T3.2 | reported |
| Build/Speak for Location Report, Severe Read, Transition | **declined** — T3.2 | `Failed{id}`, so the schedule re-plans (DR-21) |

**The producer seam.** `alertProposal` is the producer's record of the alert a card is about — the
event, and whether the card reads under a burst head, which decides the line's form as it does in
today's takeover. The card stays domain-free (DR-1); whoever proposed it keeps what it takes to
build it, the way the `[space]` read keeps its row (`eventReader.row`). T3.1 supplies the lookup.

#### Two build decisions, for ratification

| # | Decision | Reasoning |
|---|---|---|
| **BD-8** | **`BuildCard` and `Speak` carry the card's `Slot`.** | An executor is chosen by what kind of read the card is, and the effect is the description of that work. Looking the slot up from the published lineup instead would race the dispatch: the publish for the same step runs concurrently with the build (BD-7). Pinned in `platform/lineup` (`TestEffectsAboutACardCarryItsSlot`) and through a real step in `app` |
| **BD-9** | **Report cards and `Tune` are declined until T3.2, not built halfway.** The plan's T2.2 row names `BuildCard → segments()` and `Tune → radioDeck.tune`; this deviates deliberately | A report's speak is the engine Source adapter — which *is* the main-track absorb. The build half without the speak half is a rack of prepared segments nobody reads (AP-DEAD-01), and `Tune`'s result event does not exist in the vocabulary. An executor for an effect that cannot occur, asserting a result nobody defined, would be the plan's letter and not its intent |

**An empty script fails the card rather than building silence.** Today an empty line is a silent
hold. Under the lineup a `Built` with no words trips the Director's own invariant and the card
would sit at standby for ever — so the executor fails it, loudly, with a test whose fixture is a
broken script override (the one way `Say` renders "", round 4 A-13) rather than a `Skip`.

**Two invariants were removed before commit** for restating the guard beside them — the m83 shape,
a check that can stand in for its rule and hides the rule going missing.

#### Found while designing the executors — for the round-2 red team and T3.1

None of these is T2.2's to fix, and each would be found later at more cost:

| # | Finding | Where |
|---|---|---|
| **F-D1** | **`prepareNext` wedges on a structural card.** A card whose text is fixed at proposal (Burst Head, Transition, Divert Notice) reaches Admitted *with* text; `prepareNext` asserts `next.Text == ""` and returns `d, nil`, so the card never reaches Standby and `Next()` keeps offering it — the schedule stalls behind it. Unreachable today because the planner emits only Breaking Alert cards; reachable the moment T3.1 proposes a head. One line (a card already carrying its words goes straight to Standby), with its own test and mutant, in the pure core | `platform/lineup/director.go` `prepareNext` |
| **F-D2** | **The band is a shared resource across cards, and `byCard` does not know it.** `leave` returns `[release(N), cue(N+1), speak(N+1), …]`; `byCard` puts `release(N)` and `cue(N+1)` in separate runs, dispatched concurrently — so the release can land *after* the cue and clear the next card's callout. Confirmed by reading the two functions; observable only now that the executors are real. **The first fix keyed the grouping on the resource and was rejected on review — see the F-D1/F-D2 section below for what it took** | `app/pump.go` `byCard` |
| **Q-D1** | **RULED, MVS-D-65 — see the rulings below.** **The tone is not in the closed set.** Today one tone per burst precedes the first line, chosen by the highest-severity event. Nothing in the vocabulary sounds it. Adding `Tone{class}` is a deliberate change to the closed set (PL-6); T3.1 must decide before the rail goes live | `platform/lineup/director.go` |
| **Q-D2** | **RULED, MVS-D-66 — see the rulings below.** **The 1 s gap between a burst's events** (`breakingGap`) has no home: `Finished` puts the next card on the air at once | T3.1 |
| **Q-D3** | **RULED, MVS-D-67 — see the rulings below.** **The duck across a burst.** One `Run` per card ducks and restores per card; today one `Run` per burst ducks once. With T3.1 before T3.2 the bed would pump between cards. Either `Duck`/`Restore` around the rail land before T3.1 goes live, or the tasks reorder — a decision about what the listener hears, so the HUM LEAD's | T3.1 / T3.2 ordering |
| **Q-D4** | **RULED, MVS-D-68 — see the rulings below.** **Who marks a read alert as seen.** Today `readBreaking` marks each event after its read; under the lineup `Finished` reaches the Director and the producer learns nothing — a Done card is *removed* from the lineup, so a `Publish` reader sees only absence | T3.1 |
| **P10-05** | **The density meter measures `app/` as one package** (161 changed functions, exempted 0.52 under D-20), so a file added to it is never measured on its own. Validated by stripping the executors' six checks: the meter did not fire. By hand, from the committed file: 6 `invariant.Check` + 10 call-free guards (and 9 switch arms) over 11 functions. How the meter would score the file alone is not known, which is exactly why the count is written here instead of left as a silent pass. Recorded rather than trusted to the gate's silence (RD-4) | gate |

#### Mutants

| Mutant | The rule it deletes | Verdict |
|---|---|---|
| mC0 | a read the context ended still comes home Finished | CAUGHT |
| mC1 | an empty script is built as silence | CAUGHT |
| mC2 | the cue leaves no record | CAUGHT |
| mC3 | a declined card-effect fails nothing | CAUGHT |
| mC4 | a report is read as an alert | CAUGHT |
| mC5 | the band record is unbounded | CAUGHT |
| mC6 | the slot is dropped from the build (BD-8) | CAUGHT |
| mC7 | a cue that cannot be built fails the card | CAUGHT |
| mC8 | a silent read holds for nothing | CAUGHT |
| mC9 | the read follows the bars | CAUGHT |
| mCA | the slot is dropped from the speak (BD-8) | CAUGHT |

**11/11 CAUGHT, and 27 carried forward (m90–m99, mA0–mA8, mB0–mB7) — 38 in all, every one CAUGHT.**
Attribution was checked per mutant rather than taken from `run.sh`, which names only the first
failing test.

**Two of the carried-forward pins had stopped applying, and the wrapper hid it.** BD-8 put the
card's `Slot` on `BuildCard` and `Speak`, which moved the two lines **m90** and **m96** anchor on —
so the run measured 36 of the 38 it launched, breaking anti-vacuity rule 4 silently, in the same
task that relied on it. The wrapper took the FIRST line of `run.sh`'s output; an unapplied mutant
prints a Python traceback on stderr before `run.sh` reports `UNAPPLIED`, so the traceback was
recorded and the tally — counting only verdict-prefixed lines — read *"36 caught, 0 survived"*.
**That is the false-coverage bug `run.sh` exists to prevent, one layer up in its own wrapper.**

Fixed at `29f20a5`: the wrapper takes the LAST verdict line and asserts the count equals the number
launched. **Then every mutant was checked, not only the two that failed** — all 103 were applied to
a throwaway copy of the tip, and exactly m90 and m96 were stale (enumerate the consumers before
fixing). Both re-anchored, both re-run, both CAUGHT by the test that names their rule.

**And `run.sh` itself had a data-loss bug**, found by tripping it: the restore trap was armed ABOVE
the clean-tree check, so it fired on the `SKIPPED` path too and reverted the uncommitted work of
whoever ran it, seconds after telling them nothing would be done. It destroyed this very
re-anchoring on the first attempt. The trap is now armed only once the tree is known clean.
Control, with three modified files in the tree: SKIPPED, exit 2, three files still modified.

#### Gates at T2.2

`make verify` **ALL GATES GREEN** (full output captured) · `make alloc-budget` **exit 0** · p10 **21
findings, all exempted, 0 live, 0 unmatched** — **no exemption added** · `make pty-severe` **green**
· declset **+15 / −0**.

### F-D1 and F-D2 — the two defects T2.2 found, and the review rounds they took

Both were latent: nothing wires the pump or the executors yet, so neither shipped. Both were in this
branch's own code. **Every review that has returned rejected the fix in front of it** — F-D2 needed
two attempts, F-D1 three. The rounds are recorded in full because the find-rate is the argument for
having run them: three reviews have reported, all three returned ADDITIONAL FIXES REQUIRED, and one
of the findings was a defect the fix itself introduced. **A fourth is outstanding** — round 2 of
F-D2, against the band lane — and this section is written before its verdict, which is why it does
not speak for it.

#### F-D2 — the band is one resource, shared across two cards

`leave` emits `[release(N), cue(N+1), speak(N+1), …]`: the release belongs to the card going off the
air, the cue to the card coming on. Dispatched by card, those two landed on separate workers and
raced, and a release landing second **clears the callout the cue has just put up** — the stale-callout
failure DR-24 exists to prevent, wearing the opposite sign.

| # | What happened |
|---|---|
| Fix 1 | Effects declare the resources they hold — a card, and the band for a cue or a release — and consecutive effects sharing one keep their order. Green, and **wrong** |
| Review 1 | **Rejected, Critical.** Ordering held only WITHIN a step, and the two halves of a band pair routinely fall in DIFFERENT steps: a card finishes while the next card's 1.03 s build is still out, so its release goes out alone and the next cue arrives a step later on a new goroutine. Reproduced deterministically. Also: the grouping compared each effect only against the one before it, and an effect holding nothing — `Publish` today, `Duck` and `Restore` from T2.3 — sat between a cue and its words and split them, **which the test suite asserted as correct** |
| Fix 2 | Everything touching the band runs on ONE LANE, enqueued at dispatch on the pump's own serial goroutine, so the order outlives the step. Grouping is simpler for it: a run is all the effects about one card, joined connectedly rather than by adjacency |

**I had seen the cross-step gap while designing round 1, decided to record it as a finding, and did
not.** It reached neither the record nor the commit message. The review found it in twenty minutes.
That is the whole argument for the loop: the author's own scoping decision evaporates unless it is
written down, and a fresh reader does not inherit it.

#### F-D1 — the rail wedges on a card that arrives with its words

A burst head, a transition and the divert notice carry their words from proposal (DR-7).
`prepareNext` asserted a card awaiting preparation has no words, so it refused exactly those cards:
they never left ADMITTED, only a STANDBY card can take the air, and `Next` offered the same card for
ever — **every card behind it went unread**, which is DR-3 failing from the other side.

| # | What happened |
|---|---|
| Fix 1 | Such a card reaches standby, describes no build, and takes the air in the same step. Green, and **wrong in three ways** |
| Review 1 | **Rejected.** The MIRROR wedge was still open — a structural card proposed WITHOUT words was accepted, handed a build it can never absorb, and stranded at standby. The branch tested the SYMPTOM (`Text != ""`) where the registry already answers the question, and the invariant it removed was replaced by nothing. The one-ahead build was still lost behind TWO structural cards, which is how a real burst opens |
| Fix 2 | The door closes in `Propose`, the branch asks the registry, and preparation walks past cards needing no build. **Wrong again** |
| Review 2 | **Rejected.** The walk looked past every prepared card, so each completed build started the next: three cards built and a fourth building with the first still on air, across tracks. **DR-7's freshness traded away by the fix meant to protect the schedule.** The door was only on `Propose` while the commit claimed "where every card enters" — `Queue` is a door too, and took a wordless head. And the registry walk pinned only one direction, so the second literal list could drift silently |
| Fix 3 | The walk stops at a report standing by; the rule moved into `check`, which every write goes through; the registry walk pins both directions |

#### What the rounds cost me in claims, recorded because a wrong claim is not self-correcting

| Claim | What is true |
|---|---|
| *"It carries its own control"* (F-D2 round 1) | The blocking test passes under the mutant it was named against; the control is a different test plus a table row |
| *"exactly what `leave` emits"* | `leave` emits five effects, not the three quoted |
| *"a step never describes two reads"* | **Vacuous** — `takeTheAir` returns at most one read and the second call is guarded, so it cannot fail. Its mutant SURVIVED. The same shape as a check deleted two paragraphs earlier in the same commit, for the same reason |
| *"cost 0.06 of the density floor, 1.94 against 2.0"* | Wrong: a number measured BEFORE four invariants were added, reported as if it described the committed state. Re-measured at the tip both ways — the literal form has no density finding, the registry form reports **1.97 < 2.0** and goes live |
| *"closed where every card enters"* | Entry is `Queue`; the check was in `Propose` |
| *"the test walks every slot so the two cannot drift apart"* | It pinned one direction of two |

#### The instruments, twice

**The sweep wrapper reported a clean run over mutants that never ran.** BD-8 moved the lines m90 and
m96 anchor on; the wrapper took the FIRST line of `run.sh`'s output, and an unapplied mutant prints a
Python traceback before `run.sh` reports `UNAPPLIED`, so the tally read *"36 caught, 0 survived"* over
38 launched. It takes the LAST verdict line now and asserts the count equals the number launched.

**Every mutant is now applied to a throwaway copy of the tip before any sweep runs** — seconds against
the hour a sweep costs. It has caught a stale anchor on every round since, including one I missed
while re-anchoring its two neighbours.

**`run.sh` itself destroyed uncommitted work.** Its restore trap was armed ABOVE the clean-tree check,
so it fired on the `SKIPPED` path and reverted the tree of whoever ran it, seconds after telling them
nothing would be done. It ate a re-anchoring before it was caught. Armed after the check now, with a
control: three modified files in the tree, SKIPPED, exit 2, three files still modified.


#### The full sweep, and what it cost to run it wrong

**119 launched, 119 verdicts, 110 CAUGHT.** Every one of the eight survivors and the one INVALID was
an instrument defect or a recorded deliberate; **none was an unpinned rule in the product.**

| Survivor | What it actually was |
|---|---|
| mDD | **A real gap.** The lane's retirement was proved with a throwaway probe and the probe deleted, so the rule had no pin. Now a test, asserting on the CHANNEL — a receive from a closed channel is always ready, an open empty one blocks, so no timing enters the assertion |
| mD7 | The rule written twice: moving it into `check` left the copy in `Propose` enforcing what `check` already did, so deleting either changed nothing. Copy removed, mutant retired |
| mDB | INVALID — written as `if false`, leaving an identifier unused so the tree would not compile. **The sixth of that shape this release** |
| mDF | A mutant that could never be caught: it weakened an ASSERTION to its vacuous predecessor instead of breaking the rule, so both forms passed |
| mD0, mDC | **Scope, not coverage.** Both delete rules from the effect vocabulary in `platform/lineup` whose pins live in `app/pump_test.go`, and the sweep ran neither package's tests. Re-run with `./app`: both CAUGHT |
| m16, m42, m43 | The recorded deliberates — a defence-in-depth guard that cannot be observed to fail, a rule equivalent across all 111 products, and F-D4's hollowed pin |

**Then the fix for the scope defect cost more than the defect.** Adding `./app` to every mutant meant
**65 x ~180 s — 3.2 hours — to change two verdicts**, because `./app` takes ~90 s and `run.sh` runs a
baseline before each mutant. The HUM LEAD asked why the tests were taking so long, which is the wrong
person to be asking:

> *"if your approach routinely takes HOURS for something that used to not take that kind of time —
> the human (me) shouldn't have to be the one to raise the flag that we might be doing something in
> an inefficient or 'token heavy' manner."*

**The shape that was wanted is ESCALATE ON SURVIVAL:** run the cheap scope, and re-run wider only on
SURVIVED. A CAUGHT is valid at any scope — a wider scope cannot unpin a rule — while SURVIVED is both
the only verdict that drives work and the only one a too-narrow scope can get wrong. The five mutants
that actually needed settling took **minutes**, and four of them never paid the wide scope at all.
The running job was killed rather than waited out.

**Why `./app` is 90 s, since it is the multiplier behind all of this:** real-time waits in the
product, not sleeps in the tests, whose own declared sleeps total about two seconds. A five-second
hold per breaking event with no voice, a one-second gap between burst events, 100 ms hold steps in
the narration path — exercised by tests that install the REAL sleep rather than an injected clock.
DR-20 makes the clock injectable in the Director; the same treatment on the narration tests is what
would shrink this gate, and it is worth a task of its own.


### T2.3 — mastercontrol: one owner for the band, one for the bed

`app/mastercontrol.go`. The effector half of the old director's split (PD-5). The decision half is
already a pure function in `platform/lineup`; what remains here is the half that PERFORMS, and its
job is putting one card on the air across BOTH outputs at once — which is the thing the HUM LEAD's
condition asked for.

**Two carriers went in, one came out.** The band's takeover message was constructed in
`app/ticker.go` for the live takeover AND in `app/executors.go` for the Director's; the duck was a
bool on the arbiter beside a nil-check on the deck. Both are single-owner now (D-1). This codebase
has already paid for two carriers of exactly the duck rule: it was lifted by one spelling of `tune`
and not the other, and a listener heard the next location come up at full volume over a breaking
alert still reading.

`Duck` and `Restore` have executors, which **MVS-D-67 makes a prerequisite for the rail**: the bed
gives way once for a whole drain and is taken back only after the tail. `giveWay` is idempotent, so
the arbiter's own dip and the Director's cannot stack.

#### Three attempts at the ownership, recorded because the first two were wrong

| # | Shape | Why it failed |
|---|---|---|
| 1 | The ticker reaches through the arbiter for the effector | A deck with no arbiter is LEGAL — the visual half runs without a voice — and `t.voice.mc` dereferenced nil, taking the suite down |
| 2 | The ticker holds its own effector | Two instances is two owners, which is the thing being removed |
| 3 | One instance, constructed once and injected into both | Tests go through one builder, so a call site cannot forget it |

**Renamed rather than exempted.** `mastercontrol`'s methods shared names with the voice methods they
delegate to, and P10-01's name-based analyser reads that as recursion — the codebase already carries
one ratified exemption for exactly that shape (`radioDeck.Stop` / `stopDwell`).
`giveWay`/`takeBack`/`holdLine`/`resumeLine`/`dropHeld` collide with nothing and read better, so no
exemption was needed. **The bool returns from `giveWay` and `takeBack` are gone** (P10-07): nothing
asked which call won, and a return nobody reads is a promise to a caller that does not exist.

#### The pin the mutants demanded

`TestTheTickerCuesThroughTheOneOwnerNotItsOwnSend`. Asserting that a cue *reaches the band* cannot
see a second writer, because in production both writers reach the SAME band — so the deck's own
`send` and its effector's band are **different captures**, and the cue must land on the effector's.
Without it, the mutation that puts the second writer back would have survived, and the property this
task exists to create would have been unpinned by the tests that claim it.

#### Two findings on the way, neither caused by this task

| # | Finding |
|---|---|
| **F-D5** | **A test had a time bomb, and it went off this morning.** `TestSeenStoreColdStartPruneAndPersist` marked ids at a fixed `2026-08-27` while `loadSeen` prunes against the REAL clock over a seven-day window — so on **2026-09-03**, exactly seven days later, the reload pruned precisely what the test then asserted was still there. It had been passing for a week for a reason unrelated to what it pins. Anchored to the real clock |
| **F-D6** | **A silent failure in the fence.** NaN compares false against everything, so an unparseable coordinate fails the radius test AND the significance exception, and the hazard is **fenced out rather than admitted** — a feed that misparsed a point would quietly empty the burst. It has an invariant and a test whose fixture is asserted admissible first |

#### P10 — one live finding, PRESENTED FOR RATIFICATION, not self-approved

```
platform/lineup · package · P10-05-INVARIANT-DENSITY
invariant density 1.98 < 2.0 across 66 changed function(s)
```

**Recorded in `.a2dh-p10-exemptions.yml`, and `make p10` is 0 live, 0 unmatched with it.**

**The ledger is gitignored, so this is the only reviewable record (PL-16).**

The package was at or above the floor before this task and T2.3 changes nothing in it but the fence
invariant above, which raised the count rather than lowering it. The changed-function set grew by
one when a test fixture added earlier the same day — `platform/lineup/harness_control_test.go`, the
old shell controls' target — was correctly deleted along with the shell guards it existed for.

**Why no more checks were added to close it.** Every remaining candidate asserts a condition its own
branch already guarantees: the walk returns a card that is ADMITTED because the branch tested
`State == Admitted`; `find` returns a card whose ID matches because the loop compared it; a registry
lookup is complete because `String` already checks that at the point of use, and a second copy would
be D-1's two carriers. **That is the vacuous-invariant class, five of which shipped yesterday and
three of which were added to satisfy this very meter.** D-2 says name the input that makes a check
fail before writing it, and for these there is none.

**RATIFIED by the HUM LEAD, 2026-09-03** — "P10 ratified". The exemption stands on the reasoning
above: no check was added to close it because every remaining candidate asserts a condition its own
branch already guarantees, and padding a density meter with checks that cannot fail is the class
that produced five vacuous invariants the day before.

### The bed across a rail drain — traced, and the ruling made achievable

**The HUM LEAD asked for the trace and a recommendation, with Observer's behaviour not to regress.**
The trace was run rather than reasoned, driving two cards through the executors exactly as the rail
will and recording what the bed heard:

| Path | What the bed heard |
|---|---|
| Arbiter only (today) | `duck · first · restore · duck · second · restore` |
| Bracketed by the Director's `Duck`/`Restore` | `duck · first · restore · duck · second · restore` — **identical** |

**The Director's effects were powerless.** `giveWay`'s idempotence collapses the second dip, so the
first `duck` is shared — but the arbiter's own take-back after card one LIFTS the Director's duck,
card two dips again, and the closing `Restore` is a no-op because the bed is already up. The
listener hears the broadcast surge back between two alerts of one burst, which is exactly what
MVS-D-67 forbids.

**Four shapes were considered.**

| Option | Verdict |
|---|---|
| **A counter** — `giveWay` increments, `takeBack` decrements, lift at zero | **Rejected.** The arbiter's calls are not balanced: it gives way per admitted audible job but takes back once per *quiet moment*, so a counter would leak the dip or over-release. Making them balanced means changing the suspend/resume logic that came out of red-team round 4 and UAT |
| **One arbiter sequence per drain** — the whole rail inside one `Run` | **Rejected.** The executor would hold a `Run` open across cards, which is the blocking class PL-1 removed from the pump |
| **The Director owns all ducking** — the arbiter stops dipping entirely | **Rejected FOR NOW, and it is the end state.** It is one owner and one decision-maker, but the `[space]` read and today's takeover are not Director-driven yet, so it would regress Observer today. It arrives when everything reads through the Director (T3.2+) |
| **A HOLD that outranks the per-sequence take-back** | **Adopted** |

`Duck` now HOLDS the bed and `Restore` releases it. While held, a per-sequence take-back is refused;
on release the bed comes back only if the arbiter is idle — and `idle` is *asked of the arbiter*
rather than copied, because "when may the bed come back" already has an owner and a second carrier
would disagree with it the way the duck's two spellings once did (D-1).

| Path | What the bed hears now |
|---|---|
| Arbiter only (today) | `duck · first · restore · duck · second · restore` — **unchanged, and asserted** |
| Held for the drain | `duck · first · second · restore` — **one dip, one lift** |

**Why Observer cannot regress:** nothing emits `Duck` yet, so `held` is never set on any existing
path and every one of them sounds exactly as it did. The unbracketed row above is a test, not a
claim — it is the more important half of `TestTheBedIsDippedOnceForAWholeDrain`.

#### The release decided across two locks — found by review, and my claim was false

**A fresh adversarial review rejected the first version of the hold, and the mutant encoding the
failure was surviving.** Releasing the hold asked the arbiter whether it was idle and then acted on
that answer, with the question answered **outside the effector's lock**. A job admitted between the
question and the lift had the bed restored out from under it: the broadcast surging to full volume
over a read in progress, which is the exact class the single-owner work exists to prevent.

**The obvious fix would have deadlocked.** Holding the effector's lock across the question inverts
the order every other path takes — `settle` calls into the effector while holding the arbiter's lock
— so the decision moved to where that order already runs. `releaseBed` takes the arbiter's lock,
clears the hold and defers to `settle`, which already owns *when may the bed come back* and is the
only thing that should. `mastercontrol.release` and `director.idle` are gone with the defect; `idle`
existed only to be asked across a lock boundary, and a predicate whose only caller was that mistake
is not worth keeping.

| Mutant | Rule | Before | After |
|---|---|---|---|
| mE4 | a held bed is lifted between cards | CAUGHT | CAUGHT |
| mE5 | the hold is never cleared, so the bed never returns | CAUGHT | CAUGHT |
| mE6 | the rail lifts the bed over a live read | **SURVIVED** | CAUGHT, by the test named for it |

**Two claims of mine were false and are corrected here:** *"the mechanism is done and pinned"* and
*"`Restore` lifts only if the arbiter is idle"*. It was neither pinned nor race-safe.

**And the process miss, which is the more useful half.** I wrote mE4, mE5 and mE6 and **never swept
them** — I read the corpus guard's green as coverage, when all it checks is that a mutant APPLIES and
COMPILES. That is the same confusion between *the instrument ran* and *the rule is pinned* this
release has already paid for twice. The reviewer swept them because the dispatch told it to; I had
not. **A guard that proves a mutant is well-formed is not a guard that proves a rule is pinned, and
the two greens look identical.**

Also closed from the same review: `startTicker` built an effector unconditionally and discarded it
on every production path. One instance, none built to be thrown away.

**What is left for T3.1:** emitting the pair at the right two moments — `Duck` when the rail takes
the air, `Restore` after the tail and only once the rail is clear. The mechanism is pinned now, and
what remains is the schedule deciding when.




#### UAT of the T2.3 build — two rulings, neither a T2.3 regression

The HUM LEAD ran `dist/watchpost` at `843f926`. **What worked and needed nothing: the duck, play,
pause, mode and repeat.** Two changes were ruled, and both are older than this task.

| # | Ruling |
|---|---|
| **MVS-D-69** | **No tone on a `[space]` read.** *"<tone> is used for ticker notification to grab the users attention. When a user presses [w] - their attention is already secured."* Arrived earlier in 0.14.0, which opened this read with the row's class tone the way a takeover does |
| **MVS-D-70** | **The bed dips to 15 %,** from 25 %. *"still loud enough to be distracting, now loud enough for me to know it's still playing. Suggest we reduce that volume by another 5-15%"* |

**The four seconds the HUM LEAD measured were mostly silence, and it was silence built for a
different caller.** Every tone carries a two-second trailing silence (`synth.alertTailDur`) so a
takeover has a beat between the signal and *"…has been declared"* — and the read held for the whole
buffer:

| Preset | Buffer | Audible pulses |
|---|---|---|
| classic | 2.8 s | 0.8 s |
| soft-chime | 3.2 s | 1.2 s |
| low-sweep | 3.6 s | 1.4 s |
| dual-tone · 1050 Hz | 4.0 s | 0.6-1.2 s |

Then the render, about a second. So: tone, four seconds, words. Removing it takes the whole 2.8-4.0 s
out and leaves the render — inside the 2-3 s the HUM LEAD would have accepted *with* a tone. **The
takeover still sounds one:** the ruling is about who asked for the read, not about tones, and an
unattended alert still has to fetch the listener.

**MVS-D-70's figure was read as PERCENTAGE POINTS**, and the reading is stated because the two differ
audibly: 5 % *of* 0.25 is 0.2375, which no ear separates from 0.25, so the range is 0.10-0.20 and
0.15 is its midpoint. **The depth could not be changed quietly** — it was ratified on 2026-08-27 with
a test asserting it, so `TestTheDipDepthIsPinned` carries the number, the reason it is not zero, and
now the reason it moved. The constant's own comment moved with it rather than drifting away from it.

#### F-D5 — the dip and the claim were not atomic

**Found by a fresh review in round 2, and it is the ACQUIRE-side twin of mE6.** `hold` dipped the bed
under the lock, released it, and took it again to set `held`. `hold` runs on the executor's goroutine
and `takeBack` on the arbiter's, sharing only that mutex, so a take-back landing in the window read
`held=false` over a ducked bed and lifted it — and `held` was then set over a broadcast already back
at full volume. **The rail reads its whole drain against an undipped bed, which is the inverse of
MVS-D-67.** Latent today because `Duck`/`Restore` are not emitted until T3.1, which is exactly when
it would have gone live, presenting as "ducking sometimes stops working mid-burst".

The dip became its own carrier (D-1) so `hold` can dip AND claim without letting go.

**THE PIN WAS BUILT THREE TIMES, AND ONLY THE THIRD ONE MEASURED.** This is the entry worth reading
twice, because in both failures the test was green and the report would have said "closed":

| Attempt | What it did | Why it was thrown away |
|---|---|---|
| 1 | Waited to OBSERVE the dip, then lifted | **Passed against the defective code.** The goroutine never ran before `hold` had finished |
| 2 | Kept a take-back in flight; 300 rounds | Caught it four runs in five — **a coin toss, not a gate**. Round count then sized from the measured hit rate to 5000 |
| 3 | Same, but the test fixes its own GOMAXPROCS | Attempt 2 **passed on defective code at GOMAXPROCS=1** — one P gives `hold` no preemption point, so the window is never observable. A yield in the spin cured the resulting timeout and dropped detection to one run in five; fixing the parallelism instead keeps both |

**Final measurement: the defective code fails 13 of 13** across default P, `GOMAXPROCS=1` and `-race`;
the fixed code passes all three, in 0.4 s (1.6 s under race). An independent reviewer reproduced this
separately at 16 of 16 rather than taking the numbers on trust.

#### Gates at T2.3

**Every smoke is named here, which is the F-D3 lesson: a smoke absent from the table is a smoke
nobody runs.** Run on the final tip `2b3ead8`.

| Gate | Result |
|---|---|
| `make verify` | **ALL GATES GREEN** (fmt · vet · tidy · vuln · race · lint-imports · lint-watermark · gate-controls · mutant-check) |
| `make alloc-budget` | exit 0 |
| `make p10` | **0 live, 0 unmatched** (103 dormant) — see the binary note below |
| `make mutant-check` | **128** mutants apply and compile, none stale, none inert |
| `make pty-severe` | ok (w opens, right to Watches, enter/esc/esc, w re-opens, esc, ctrl+s opens, esc, q quits) |
| Live relay (macOS half of R6) | PASS — both directories play |
| Declaration set | **+17 / −2** across the task. The 17th is `mastercontrol.dip`, the new carrier; the two removed are `executors.cue`/`executors.release`, **renamed** to break a P10-01 name cycle rather than deleted |

**The T2.3 mutants, every one run to a verdict:**

| Mutant | Verdict | Caught by |
|---|---|---|
| mE0 the duck is not idempotent | CAUGHT | `TestDirectorBreakingSuspendsAndResumesARead` |
| mE1 the bed is taken back when nobody dipped it | CAUGHT | `TestDirectorMutedAndSilentNeverDuck` |
| mE2 the ticker writes the band itself | CAUGHT | `TestTheTickerCuesThroughTheOneOwnerNotItsOwnSend` |
| mE3 the duck effect never reaches the bed | CAUGHT | `TestTheDuckAndItsRestoreReachTheEffector` |
| mE4 a held bed is lifted between cards | CAUGHT | `TestTheBedIsDippedOnceForAWholeDrain` |
| mE5 the held bed is never given back | CAUGHT | `TestTheDuckAndItsRestoreReachTheEffector` |
| mE6 the rail lifts the bed over a live read | CAUGHT | `TestTheRailNeverLiftsTheBedOverALiveRead` |
| **mE7 the dip and the claim are not atomic** | **CAUGHT** | `TestTheBedIsNeverLiftedBetweenTheDipAndTheClaim` |
| mF0 the [space] read sounds a tone again | CAUGHT | `TestEventReaderDucksSpeaksRestoresAndOverlaysThePanel` |
| mF1 the bed barely dips | **CAUGHT — after a scope escalation** | `TestTheDipDepthIsPinned` |

**mF1 first reported SURVIVED, and the verdict was the instrument's fault, not the code's.** It was
run against `./app` while the constant it mutates lives in `domains/radio/player` and its pin lives
there too. Escalating the scope caught it immediately. **This is the case escalate-on-survival exists
for**: the old habit would have sent someone hunting for a hole that was never there.

**The corpus guard earned its place three times in this task alone**, catching m48, m49, mC2, mE3,
m35 and m36 going stale from my own edits — each before a sweep rather than during one.

#### An environment finding, not a code finding: `make p10` failed for the wrong reason

`make p10` first exited 2 printing **"p10: live findings — see dist/p10.json"**, which is the
Makefile's generic failure handler firing because the `a2dh` on PATH (`~/.local/bin`, 26 Jul) predates
the `p10` subcommand entirely. The framework's own build
(`<framework build>`, 18 Aug) has it, and against that binary the gate is **0 live, 0 unmatched**.

**The message is the defect.** A missing subcommand and a real P10 violation are opposite situations
and cannot share an error string — one is "your toolchain is stale", the other is "your code broke a
rule". Two upstream items follow, both A2DH-side rather than watchpost-side:

1. `make p10` should distinguish "the binary cannot run this check" from "the check found something",
   because as written a stale install reads as a code failure.
2. The thin install on PATH drifts from the canonical build with nothing reporting it, which is the
   same class as the stale-calibration problem already recorded.


## T3.1 — the alert rail

**The defect is removed rather than bounded.** `capBurst` and the read-time budget are gone: selection
is now `lineup.Plan`, the pure ladder that already existed in `platform/lineup/plan.go`, and NOTHING
bounds a read once the card is admitted (DR-3). What is past the Max is DIVERTED and counted, and the
count is kept because it is spoken aloud.

| Deleted | Added |
|---|---|
| `capBurst` and its per-lane floor | `tickerDeck.railBurst` — translation only; the choosing is `lineup.Plan`'s (D-1) |
| `maxBreaking`, `breakingAllowance`, `breakingCap` | `defaultBurstMax = 5` (read-order-design.md), the ONE bound, applied at admission |
| the `elapsed >= breakingCap` cut in `readBreaking` | `tickerDeck.fence`, `subjectOf`, `diverted` |
| `byBreakingPriority` inside `breaking` | — the rail sets read order; `breaking` performs it |

**Two defects found while wiring it, both mine, both caught by the suite:**

1. **One untied alert would have silenced the whole burst.** `Plan` refuses a batch containing an
   arrival with an empty Subject — deliberately, so a malformed arrival is caught before the takeover
   rather than halfway down it. I mapped Subject to `Event.Location`, which is the TIED location and
   is empty for an alert the locator could not tie. A single such alert would have made `railBurst`
   return nothing and the takeover read NOTHING — worse than the defect this task removes. `subjectOf`
   falls back to the feed's own place text, then to the hazard type.
2. **`breaking` indexed `fresh[0]` with no guard.** Unreachable from production now that the caller
   returns early on an empty burst, but it is one line and the panic would take the marquee down.

**The pin was validated against the defect before it was trusted (D-11), and the first version of it
had a hole.** Comparing a fast read against a slow one catches a TIME-dependent bound but not a
constant one — a loop that simply stopped after three would read three both ways and look perfectly
consistent. It now also asserts that the number read equals the number the rail ADMITTED, asked
against the rail rather than against a literal so that moving the Max cannot make it go quietly green.
`mG0` reintroduces the constant form and is caught.

**Two mutants RETIRED rather than re-anchored, and the corpus guard is what found them.** `m23`
guarded the count bound (`capBurst(fresh, maxBreaking)`) and `m24` the time bound
(`breakingAllowance`). Both rules were DELETED by this task, so there is nothing left to anchor to —
a mutant for a rule that no longer exists is not stale, it is finished. `mG0` guards what replaced
them: nothing bounds a read once the card is admitted. **Corpus 128 → 127.**

`readBreaking` also lost its `ctx` parameter, which went dead when the inter-alert pause moved from
`sleepCtx` to `s.hold`. That is a small improvement rather than a neutral move: the speaker's hold
honours SUSPENSION as well as cancellation, so a takeover that parks this read no longer has the gap
counting down underneath it.

## T3.2a — the schedule runs, and a listener notices nothing

**T3.2 was SPLIT (HUM LEAD, 2026-09-03).** The plan row absorbed the Watchlist dwell into a Director
that nothing drove: `newPump`, `newExecutors` and `lineup.New` had **no production callers at all**.
Moving the dwell onto it would have deleted a working five-minute advance and replaced it with
something that never fires. T3.2a wires the schedule in at PARITY — a claim a test can state — and
T3.2b does the absorb.

`app/schedule.go` performs through the arbiter and effector the ticker ALREADY has (`nar`, `nar.mc`).
It is the easiest place in the tree to reintroduce the two-owner defect T2.3 removed, and it takes
them rather than building either.

**The leak pin did not measure anything, twice.** It cancelled the context itself right after
stopping, so the goroutines exited on that rather than on `stop` — green with `stop` gutted entirely.
With `stop` as the only shutdown path, gutting it goes from 2 goroutines to 5. **And what it does NOT
pin is written into it:** removing `stop`'s wait for the tick goroutine leaves it green, because
`pump.send` already gives up on cancellation. The wait is still right; the coverage claim would not be.

### MVS-D-74 / MVS-D-75 — a `[w]` read can be paused, and closing the window stops it

**Found at UAT, and correctly called a design gap rather than a bug** (HUM LEAD, 2026-09-03): *"I have
no way to stop a `[w]` read once I start it… we haven't accounted for a play/pause of a notable
weather event read."* `End()` already existed and cancelled a read cleanly — wired to exactly one
caller, **shutdown**. The machinery was there and nothing could reach it.

**A second thing sat beside it:** closing the window left the read running, so the station kept
talking about a row the listener could no longer see. That is a defect, not a gap, and the same
missing wire.

| Ruling | |
|---|---|
| **MVS-D-74** | `[space]` on the reading row **pauses and resumes** it. A different row ends the running read and starts that one — ignoring the press would leave a key doing nothing on a row that looks ready, which is the complaint one level down |
| **MVS-D-75** | Closing the window **stops** the read, and a read cut short is **not marked as read**, because it was not heard |

**A LISTENER'S PAUSE IS NOT A SUSPENSION, though it uses the same stack.** A suspended job waits for
something to finish and resumes on its own; a paused one waits for a person. Two consequences, both
found by writing the pins:

1. **`settle` must never promote it.** They share the suspended stack with jobs a takeover displaced,
   and those DO resume on their own — so a paused read would have started speaking again the moment
   an unrelated alert ended, the listener's hold undone by something they had nothing to do with.
   `innermostResumable` skips them.
2. **A paused read HOLDS NOTHING.** The bed comes back up while it waits, because nobody is speaking
   over it. Counting it would have left the broadcast dipped for as long as the pause lasted — press
   pause, and the station goes quiet instead of coming back.

**And the resume had to re-dip.** The first pin caught it: `resumeLine` resumes the LINE, and the
ordinary suspension path never lifted the dip in the first place, so nothing put it back. `settle`
now asks `giveWay` on the resume path — a no-op for a takeover-suspended job, and the difference
between a resumed read and a resumed read speaking over a broadcast at full volume.

**The glyph:** ▶ becomes ‖ on the same row in the same colour (`*` → `=` under `--ascii`). Same token
deliberately — a paused read is still that row's read, and recolouring would read as a different KIND
of state rather than the same one held.

**The footer cost a column.** "Play/Pause" is six wider than the "Read" it replaced and did not fit
the 80-column ASCII floor, so the label now degrades on the same ladder that shortens "Event Details"
— reaching `Play` at the floor, the word a listener needs before they have anything to pause.

### The chip names the next press (HUM LEAD, 2026-09-03)

*"What about conditionally changing the label between 'Play' and 'Pause' depending on state — we do
this with other chip controls, so it should be okay here too."* It is the radio panel's own form
(`modes/tty/radio_panel.go`, "UAT 37: compact, state-driven labels"), so the window now matches the
panel instead of inventing a second convention.

**It also dissolved a problem rather than working around it.** The constant "Play/Pause" was six
columns wider than the "Read" it replaced and blew the 80-column ASCII floor, which I had patched by
degrading the label down the ladder. "Play" and "Pause" are each short enough that no degradation is
needed — and the 80-column row now reaches the THIRD rung (`Rows · Tabs · Details`) instead of falling
to the bare-arrows fallback, so the narrow terminal reads BETTER than before the feature.

Pinned across all three states — nothing reading, reading, paused — because a label that only ever
showed one of them would look right in a screenshot and be wrong in use. Validated by making it a
constant and watching the pin fail.

### T3.2a UAT — PARITY CONFIRMED (HUM LEAD, 2026-09-03, build `4c713bc`)

T3.2a's only claim was that a listener notices nothing, and the UAT covered every path the wiring
could have disturbed: **Observer location reads work · the synth correctly pauses its read when `[w]`
starts · play/stop on a location works · `q` works · the tone and the ticker synchronisation work.**

**Resources, measured under three loads** — the protocol's three scenarios (perf-protocol.md §8):

| Scenario | Threads | MemB |
|---|---|---|
| Startup, no radio or stream playing | 21 | 113 MB |
| After a weather alert + ticker takeover | 28 | 133 MB |
| High activity (play/pause/switch/`[w]` modal) | 28 | 175 MB |

**No regression from the schedule.** The pre-T3.2a range was 90–120 MB idle and ~170 MB under the
same heavy pattern, so the three constant goroutines and the one-per-second tick cost nothing
measurable. The idle-to-active spread remains ADR-03's documented per-location scheduler cost, not a
leak — the same conclusion the 2026-09-03 investigation reached, now confirmed with the schedule
running.

### PL-16 — one P10 exemption PRESENTED for ratification, replacing two

**`app/every.go` · `everyTick` · P10-02-BOUNDED-LOOPS.** A clock-driven loop cannot state a bound in
iterations — that is what a clock is — so every poller needed its own exemption and the count grew
per task. **HUM LEAD: *"generally there are 3 numbers I care about: 0 / 1 / n. Once we get to n, then
I think it's worth considering a shared helper."*** At n, so the helper exists.

| Before | After |
|---|---|
| `app/release.go` · start (row, unratified) | **deleted** — its poller calls `everyTick` |
| `app/schedule.go` · tick (would have needed a row) | never needed one |
| — | `app/every.go` · `everyTick` — **one row, pending ratification** |

Net for this shape: a row per poller becomes ONE, and Phase 3's remaining tasks add none. The pump's
event loop and the director's condition-variable wait are **deliberately not folded in** — different
shapes, and forcing them through one helper would be worse than two honest exemptions.

**Status: RATIFIED by the HUM LEAD, 2026-09-03.** Presented rather than self-approved, and granted on
its own terms — a clock loop has no honest iteration bound, and any number written there would have
been fabricated to satisfy a checker. The row now carries the explicit ratification rather than
"Ratify at the 0.14.0 BUILD gate", which moves it off the PL-14 list and leaves **six** entries still
awaiting reconciliation at BUILD exit.

#### Gates at T3.1 / T3.3

Run on `92fb43f`. **Every smoke is named, which is the F-D3 lesson.**

| Gate | Result |
|---|---|
| `make verify` | **ALL GATES GREEN** (fmt · vet · tidy · vuln · race · lint-imports · lint-watermark · gate-controls · mutant-check) |
| `make p10` | **0 live, 0 unmatched** — `breaking` hit complexity 16 mid-task and was SPLIT (`openBurst`) rather than exempted |
| `make alloc-budget` | exit 0 |
| `make pty-severe` | ok |
| `make mutant-check` | **133** mutants apply and compile |
| Live relay (macOS half of R6) | carried — unchanged by this task |

**The task's mutants, every one run to a verdict:** mG0 (a bound returns mid-read), mG1 (the tone
follows the running order), mG2 (the render moves back into the pause), mG3 (the first alert renders
in the head's pause), mG4 (the tone tie falls back to position), mH0 (nothing-left-to-wait skips the
check), mH2 (the cue goes up without checking the air) — **all CAUGHT**. mH1 **RETIRED**: it mutated a
TEST, which can never be evidence.

**Round 4 found the same defect as round 3 through a door round 3's pin did not cover** — the retry
path reaches a cue through a second render with no hold between, so `holdRest`'s air check never
fired there. Fixed at the loop head, and BOTH pins kept: removing the check fails the new one while
the older stays green, which is how they were shown to be complementary rather than redundant.

**Re-anchored rather than retired**, each after a restructure moved its target: m22 (which had become
INVALID, not merely stale — it deleted a rule AND broke the build), m49, m50, mG2, mG3. **Retired**:
m23 and m24, whose rules T3.1 deleted outright.

**Three review rounds, and the shape of them is the point.** Round 1 found a behaviour regression
(the tone). Round 2 found a false premise (the 400 ms) and a missed boundary (the head). Round 3
found **nothing wrong with the behaviour at all** — it measured every gap against the ruling and they
matched — and five problems with the GUARDS, including a guard that could not catch the one case it
was written for. The defects moved from the product to the instruments and then to the guards on the
instruments, which is the direction that should be expected as a task settles.

### MVS-D-73 — a severity tie is broken by the louder tone

**RULING (HUM LEAD, 2026-09-03): "Higher tone severity wins."** Raised by review as latent, not live:
`worstOf` picked the burst's tone by `Severity`, a three-value colour tier that two hazards of
different kinds share constantly, and broke ties by POSITION — the exact mechanism the tone fix had
just replaced. A red hurricane and a red tornado warning sounded the low sweep or the EAS dual-tone
depending on nothing a listener could reason about. The reviewer swept every tied pair it could build
from real feed events and found **zero** wrong chimes today, because the ladder happens to order the
louder class first — so it was fragility guarded by an unrelated ordering, which is the coupling that
was supposed to be gone.

**`cast.Class.ToneRank` is a NEW carrier, deliberately, and that is the point of the entry.** Three
orderings over the same classes already existed and none of them means this:

| Ordering | Question it answers |
|---|---|
| `cast.Classes()` / `ClassKeys()` | the order **Setup draws its checkboxes** |
| `category.ReadRank` | what is **read first** — and MVS-D-71 ruled that is about PROXIMITY as much as hazard |
| **`ToneRank`** | which **SOUND says "listen now"** |

Borrowing one of the first two would have been the same mistake as reusing `globalfeed.LaneOf` for
the read order, which is what put hurricanes at rank 7 (MVS-D-71). The rank follows the ratified
presets (tones.md §1) loudest first: dual-tone EAS, NOAA 1050 Hz, classic pulses, soft chime, low
sweep. **Disaster and Warning share the dual-tone**, so the gap between their two ranks is inaudible
and exists only to make the choice deterministic — pinned as the inaudible case, with an assertion
that fails if they ever stop sharing a preset.

`mG4` puts the positional tie-break back. One superseded assertion was updated rather than deleted:
`TestTheBurstsToneComesFromItsWorstHazardNotItsFirst` asserted "ties go to the first", which was the
old rule.

### MVS-D-72 — the burst's pause structure, and T3.3's pre-build pulled forward

**Found by the HUM LEAD on a REAL alert, not by a gate** (UAT 2026-09-03): *"the pauses between the
tone, header, alert-n, tail is noticeably long — feels like 2/3 s… Right now the uniform 2-3 s pause
in between every sentence feels like something is 'broken' if I were just listening through the
radio."* The ruling gives the structure:

	<tone>  2 s  <header>  1 s  <alert-1>  1 s  …  <alert-n>  2 s  <tail>

**The cause was not a constant, and no constant would have fixed it.** Each gap was
`synth.SegmentGap` (400 ms) + **a synchronous render (~1 s)** + `breakingGap` (1 s) ≈ 2.4 s. `s.line`
is render-then-play, and the render ran only after the previous line had finished — so every pause in
the burst contained a model load. `prepare`/`deliver` have existed as separate calls precisely to
overlap a render with the sound before it, and `app/director.go:151` says so in as many words; the
takeover never used them.

**HUM LEAD ruling: pull T3.3's standby pre-build forward**, ahead of finishing T3.1's loop, because
it is the actual defect and the plan's order was set before anyone had heard it.

| Change | Effect |
|---|---|
| The head renders DURING the tone; only the remainder of the tone is held | tone → header 3 s → **2 s** |
| Each alert renders during the alert before it | alert → alert 2.4 s → **1 s** |
| The tail renders during its own pause | last alert → tail ~0.4 s → **2 s** |
| `burstHeadGap`, `burstTailGap` added | the constants ARE the pauses |

**~~`synth.SegmentGap` is exported for this (D-1)~~ — WITHDRAWN, the premise was false.** It read:
*"every spoken clip already ends with 400 ms of silence inside its own buffer, so a caller pacing to a
ruling must subtract it."* That is true of the BROADCAST segment stream, where `synth.Source.write`
pads between segments, and false of the takeover, which renders through `speaker.prepare` →
`radioDeck.render` → `synth.AlertNarration` — `Say` plus a stereo conversion, padding nothing. So
every gap came out 400 ms SHORT of the ruling: 0.6 s where MVS-D-72 says 1 s.

**The premise was written into two comments and an exported constant, and checked against neither
path.** `waitFor` and the export are both reverted; the constants are the pauses. A later reviewer
measured all four boundaries against the real tone buffer and they land on the ruling exactly — 2 s
after the tone (from the tone's own tail, not a wait), 1 s to the first alert, 1 s between alerts, 2 s
before the tail.

**A pin failed for the right reason and was corrected rather than relaxed.** `TestT03TheCuePrecedes
TheWordsForEveryEvent` (DR-18) went red, because `cueVoice` logged `render` as "say". DR-18 is that the
band is told before the words are HEARD, and the render is silent — a faithful stand-in only while
rendering and playing were adjacent. The instrument now records `play`. **It was re-validated against
the defect it exists for** (the cue moved after the words) before being trusted, per D-11, and m49's
verdict re-checked.

`TestNoRenderSitsBetweenACueAndItsWords` pins the arrangement by ORDER, not by elapsed time — a
stopwatch pin would fail on a slow machine and pass on a fast one while proving nothing. `mG2` puts
the render back in the pause.

### MVS-D-71 — the per-lane floor is retired on the merits, not by omission

The floor guaranteed each lane's worst event a place in the burst. Removing it means a hurricane —
which `globalfeed.LaneOf` categorises as **Marine, read rank 7**, below Warnings (3), Watches (4),
Advisories (5) and Statements (6) — is diverted behind a tornado outbreak at Max 5. That was
presented as a read-order fork with three options.

**RULING (HUM LEAD, 2026-09-03): leave as is, and re-evaluate against real data.** *"Hurricanes follow
the same kind of Quake logic — a severe T.STORM warning locally is far more important than a hurricane
that is 500 miles away. I think most of the time a local hurricane 'making landfall' has one or more
additional notable event products that fall into the Warning / Disaster lane: Evacuation Orders, Flash
Flooding Warnings, Storm Warnings."*

**This corrected the recommendation put to it.** The proposal ranked by hazard TYPE — treat a tropical
system as a Disaster — on the assumption that a hurricane outranks a thunderstorm warning. The ruling
is that PROXIMITY is the signal, and that a landfalling hurricane brings its own Warning- and
Disaster-lane products that lead the read on their own merits. The two tests pinning the floor were
retired: `TestTheTimeBoundFitsTheWholeBurst` deleted with the bound it pinned, and
`TestATakeoverDuringAnOutbreakStillReadsTheQuietLanes` re-authored as
`TestEveryArrivalIsEitherReadOrCounted` — the protection is no longer "every lane gets a slot" but
"the listener is never left unaware of what they did not hear".

**Carried, not acted on:** `bandOf` returns `CloseBand` for everything except Disasters, so the
ladder's Close/Remaining split does not yet separate a nearby warning from a distant one for any other
category. With the default radius of All there is nothing to measure from, so it is inert today —
but the ruling above is explicitly about proximity, and T4.1 is where that surface arrives.

## HUM LEAD rulings, 2026-09-02 — the four questions T2.2 raised

| # | Amends | Ruling |
|---|---|---|
| **MVS-D-65** | Q-D1 (T2.2); MVS-D-12's one-tone-per-burst | **The tone is not the Director's, and the closed effect set does not grow.** *"The rule for alerts is 'most serious event in the burst sets the tone' — so that should be determined by whatever mechanism collates the burst and assembles the head and tail scripts, not the director (which is just the 'pump' to push these through). I don't know if we've ever formally defined this role in our system but I generally refer to that as the **card composer**."* The tone rides with the producer's record the executors already read, on the **first card of the burst** — the head where there is one, the single alert where there is not. A second reason it cannot ride on the card: `cast.Class` is a `domains/` type and `platform/lineup` may not import `domains/*`, so a card carrying a tone class would breach the import guard the Lineup was placed under. **The `card composer` is named as a role for the first time here**; today its work is split between `app/ticker.go` (`capBurst`, `burstHead`, `burstClosingLine`) and the planner, and T3.1 is where it becomes one thing. |
| **MVS-D-66** | Q-D2 (T2.2); `breakingGap` | **The pause between the events of a burst is ONE SECOND, and it belongs to the burst script.** *"That should be part of the burst script which the director uses to coordinate the synchronization of the ticker for the centered headline — but I would assume the standard pause between headlines is either .5 or 1 second — whatever it currently is in Observer today."* Observer's today is `app/ticker.go:348 breakingGap = time.Second`, so **1 s, unchanged**. The Director does not invent pacing: the composer puts the pause on the card, and the Director uses it to synchronise the band's centred headline with the read. |
| **MVS-D-67** | Q-D3 (T2.2); the T2.3 / T3.1 order | **One duck per RAIL DRAIN, never per card.** *"Duck across a burst — stays at its lowered volume until the rail is cleared and then either the 'press [w] in watchpost' (for Observer) or the 'for more information go to <website>' tail for Broadcaster plays. Keep it simple and deterministic."* The bed ducks when the rail takes the air and is restored only after the tail has played and the rail is empty. **Consequences, both binding:** `Duck` and `Restore` — declared in the closed set and not yet emitted — are emitted by the Director at T3.1, and `mastercontrol` (T2.3) owns them over `engine.Suppress`/`Restore`, so **T2.3 lands before T3.1** (the plan's order already). And T2.2's `speak` runs each card through the narration arbiter, which ducks and restores per `Run` — **a per-card duck that must be replaced when `Duck`/`Restore` land**, or the bed pumps up and down between the events of one burst. |
| **MVS-D-68** | Q-D4 (T2.2) | **Not an architectural gap: the executor marks the read, at the moment it mints `Finished`.** *"The audio layer likely needs to mark something as read after its tail, or we need to wire something that can tell when the script/read is fully rendered and has played in the radio. The director could also observe when a read is done since they own the schedule… I'm open to suggestions if this turns out to be a legitimate architectural or design gap."* **Recommended, pending ratification** — nothing depends on it before T3.1. |

### Why the executor, and not the Director or the audio layer (MVS-D-68)

1. **It is exactly the moment.** `speak` returns true only when the words played and the hold elapsed; today's `readBreaking` marks at the same instant, after the hold. The semantics are not approximated — they are the same instant, moved.
2. **The Director cannot.** It is domain-free by construction: `platform/lineup` may not import `domains/*`, and a seen store is a domain concern. Marking reads from the Director would put domain knowledge in the pure core, which is the one thing the architecture forbids.
3. **The audio layer cannot.** A rendered clip does not know which alert it is. The executor is the only place that holds both the card's identity and the audio's outcome — which is what an adapter is for.
4. **It preserves today's negative, which is the part that matters.** A burst cut short leaves its unread events unmarked, so a later takeover may still read them. Under the lineup, a cut-short read returns no `Finished` — so it marks nothing, for the same reason and by the same path.

**Cost:** one callback on the seam T2.2 already built — `markRead(id)`, symmetric with the `alert(id)` lookup the build executor uses. Persisting stays the producer's business, and can debounce the way `seen.save()` does today.
---

## Owed to the HUM LEAD

| # | Decision |
|---|---|
| **BD-8** | `BuildCard` and `Speak` carry the card's `Slot`, so an executor is chosen by what the card IS rather than by a lookup that races the dispatch |
| **BD-9** | Report cards and `Tune` are declined until T3.2 rather than built halfway — the plan's T2.2 row names both, and this deviates from it deliberately |
| **MVS-D-68** | The executor marks a read alert as seen, at the moment it mints `Finished` — **recommended**, and nothing depends on it before T3.1 |
| F-D4 | m42's disposition: record as a deliberate survivor beside m43 and correct the test's comment (recommended), or restore observability another way |
| — | The journey's seven stale assertions are re-authored at VALIDATE, against 0.14.0's ratified strings (recommended), or now |
| — | BD-1…BD-7 remain ratified or recorded as build decisions |

## T3.2b — RELAY REPLAY, the rotation as a Settings row (2026-09-04, `5de0216`)

Built to the HUM LEAD's spec verbatim: `WATCHPOST RADIO - RELAY REPLAY`, one picker reading
`Repeat: Watchlist - Rotate every`, seven choices (30s/45s/1m/1m 30s/2m/3m/5m) wrapping at both ends,
with `Shortest: 30 seconds / Longest: 5 minutes` on its support line. Its own group, per the stated
reason — a place for the radio settings that follow.

**Three defects, one pin.** The row was drawn correctly and was inert: `setup.go` gates ←→ on the
row's `picker` field and the row set it false, so every press was ignored. Three tests passed over it
because all three called `cycleRelayDwell` directly and none pressed a key.
`TestTheChosenRotationIsSavedToTheRadio` enters where the listener does — open, walk to the group,
press →, save — and caught the inert row plus two more seams (a close path that could drop the value,
a store that never re-tells the Director). Same shape as the Watchlist regression found at UAT:
right units, missing wire. Recorded in `quality-observations.md` as candidate rule **D-12**.

**Goldens re-recorded** (`setup-133x44`, `setup-133x44-ascii`, `setup-80x24`): the new group is drawn.
Both declsets recaptured for the group's declarations and the deck refactor below.

**The window nearly stopped fitting.** The new block measured 61 columns against the widest other at
57, and no split of the six groups fits two columns in the 126 available — `columnPlan` returned
false and the window went to 54 lines against a 32-line budget, i.e. a scroll rail. The cause was the
picker cell: `pickerNameW` is 17, sized for `Samantha (Enhanced)`, carrying `1m 30s`. Sizing the cell
to its own values (`relayDwellW = 6`) took the block to 50 and the window balances again. **The HUM
LEAD's wording is untouched** — the eleven columns came out of padding, not out of the spec.

**Two P10 findings, two restructures, no exemptions.** `relayDwellLabel` fell back by calling itself
with the default duration, terminating only because the default happens to be in the list — one edit
from a stack overflow; it now finds the fallback in the same walk. The Settings override was a
package-level `atomic.Int64` whose own pin needed a save/restore `t.Cleanup` to avoid leaking into the
next test; that was the diagnosis rather than the workaround, so it moved onto the deck as
`radioDeck.relayDwell` and the global is gone. `envWatchlistDwell` keeps the variable half.

Two tests that named today's last group/row were rewritten to **derive** the ends from
`setupGroups()`/`setupRowCount`, so the next group added does not fail them for the one reason they
do not care about (D-8).

Corpus: **mM4** (the row is drawn but inert) and **mM5** (the new rotation never reaches the
Director) — both CAUGHT. Gates at `5de0216`: `verify` ALL GATES GREEN, `p10` 0 live / 0 unmatched,
`alloc-budget` ok, `pty-severe` ok, full suite green.

## T3.7 — the Composer, and the claim that could only be settled by ear (2026-09-04, `1d476b7`)

The task's whole contract was **the live takeover sounds identical**. Every pin passed, which proves
the code still does what the tests describe — it cannot prove the listener hears the same thing.

**A real alert takeover was heard on `v1d476b7` and reported indistinguishable** (HUM LEAD, UAT
2026-09-04): *"sounded good, no noticable problems or regressions."* That is the evidence the task was
structured to obtain, and it is why composition was extracted BEFORE anything was rewired: a
behaviour-preserving change validated by ear, then the risky swap against a known-good baseline.
Reversing that order would have put two changes in front of one UAT with no way to attribute a fault
to either.

The disclosed difference stands and was not exercised: a burst spanning MIDNIGHT while reading an
event declared the day before now says "at 11:59 PM" rather than "on September 4 at 11:59 PM". It
needs both the boundary and a stale event, and a burst lasts seconds.

## F-21b — the injector, and a P10 exemption awaiting ratification (2026-09-05)

`ctrl+d` opens a diagnostics window; in a **debug build** it also offers three fabricated scenarios.
The capability is compiled out of release binaries rather than switched off, because a screenshot of
a fabricated tornado warning is indistinguishable from a real one.

**The seam is the whole value.** Injected events join the cycle immediately after the source fetch
and before `globalfeed.Active`, so they cross every stage a real alert does — the active window, the
D5 location tie, the severe index, the radius scope, fresh-detection, the rail's plan, the Composer
and the Reader. `TestAnInjectedAlertCrossesTheWholePipeline` drives a real cycle and asserts all
three consumers saw it, because an injector that shortcut any of them would validate the machinery
and say nothing about the wiring — the D-12 failure with a UI on it.

**One P10 exemption is PRESENTED, NOT TAKEN** (PL-16): `app/inject_release.go`'s `//go:build`
directive, under P10-08. The reason is recorded in `.a2dh-p10-exemptions.yml` and mirrored here
because the ledger is gitignored: it is a SAFETY split rather than a portability one, and it is
removable only by giving up the guarantee that a shipped binary cannot fabricate an alert.

Two defects the D-11 pass found, neither of them the feature. `scopeToRadius` dereferenced `t.radius`
with no nil guard while `fence()` two functions down guards the same field with a comment explaining
why — a panic in the ticker cycle would take the process. And a probe that "survived" had simply not
run: the test lives behind the debug tag and the probe omitted it, which is an instrument reporting
a verdict about a test it never executed.

## T3.10a — the schedule holds one card per burst (2026-09-05)

**DONE.** `Plan` gains `Takeover`, the ONE card the rail receives for a burst, alongside `Cards` —
the Producer's ORDERING, unchanged, because forty pins govern it and collapsing it would leave them
with nothing to look at. `onArrived` queues the takeover. **No production behaviour changed: nothing
outside tests sends `Arrived` yet**, so the Director's rail is still unfed and T3.10b is the switch-on.

**Two owners, both named.** `BurstID(lead)` is the one owner of how a burst is named, exported so the
PRODUCER can call it: the app must find the same burst again when the card's words are asked for, and
it cannot see a `[]placed`. `Burst.HasTakeover()` is DERIVED from the card's own identity rather than
stored beside it — the WIP carried a `Has bool`, and the invariant written to keep it honest was the
smell that it was a second carrier of one rule (D-1).

**Thirty-two pins re-authored, across seven files.** Most were mechanical — `a00` becomes
`BurstID("a00")`. The ones that were not are every pin that used a burst's alerts AS the several cards
it needed: DR-7's one-ahead, the rail draining before the main track, DR-21 row 1's "routed around",
DR-24's discarded-before-the-air, the trace, the pump's cue/words grouping and its band ordering. Those
now use several BURSTS, because the rule they pin is between cards and a burst is one card. Fixtures
carry the shape: `rail(t, "a", "b")` in the Director's tests, `rail(ctx, p, "a1", "b1")` in the pump's.

**The vacuous ones are the reason this was not a rename.** Four tests were GREEN after the change and
measuring nothing — `TestOneCardHoldsTheAirAtATime`, `TestTheSameEventsProduceTheSameEffects` and two
others drove scripts of `Built{ID: "a00"}` against a schedule that no longer held `a00`, so every event
was a no-op and the assertions passed over an empty run. A green test whose fixture stopped matching is
worse than a red one: nothing reports it.

**One pin where the human starts (D-12).** `TestABurstOfManyAlertsIsOneCardOnTheRail` asserts what the
operator sees — one panel, one slot number, a headline that says how many alerts follow the lead — and
it is the pin that starts where MVS-D-77 starts rather than at a schedule internal.

**The seam moved, and it is pinned as such.** `TestARealStepsEffectsAreBuiltNotDeclined` used to assert
that a real step's build comes home BUILT. It cannot: the build now names a burst and the executors'
producer record is per-alert, so composing a burst's script is T3.10b's Composer. Rather than fake the
lookup, the test pins what is true — the effect carries the card's own slot, the producer is asked by
the CARD's id (which is what T3.10b must answer to), and the build is DECLINED BY NAME and reported.
**The alternative was worse than red**: keying the fixture on the burst id would have made the executor
read the LEAD ALERT ALONE and call the burst done, which is a listener never told about the other two.

**Instrument validated (D-11).** The per-alert model was re-applied by hand against the finished tests
and CAUGHT by twenty of them, the new MVS-D-77 pin among them. Two mutants now carry it durably:
`mT0_the_rail_holds_one_card_per_alert` and `mT1_the_takeover_hides_what_follows_the_lead` — the second
because a headline that drops "+ N more" understates the hazard on the panel while the AUDIO stays
correct, so nothing else contradicts it.

**Gates.** `fmt` `vet` `tidy` `vuln` `lint-imports` `lint-watermark` `gate-controls` `mutant-check`
`p10` `alloc-budget` `pty-severe` all PASS. `race` failed ONCE on `platform/sched`'s
`TestTierCadenceIsAFixedGrid` and passed clean on re-run; it is **pre-existing and unrelated** —
1 failure in 25 here, **2 in 25 on the clean tip `588e724`**, 0 in 25 without `-race`, and
`go list -deps ./platform/sched` has no edge to anything T3.10a touched. Recorded as **F-29** with
the diagnosis, not waved through: the test waits 2 s of REAL time for a FAKE clock's advance to be
observed, so it races its own harness.

## T3.10b — the rail goes live (2026-09-05)

**DONE.** The alert rail is the Director's, end to end. The producer states what arrived
(`ticker.startTakeover` -> `Arrived`), the Director schedules it, the Composer writes it
(`executors.build` -> `composeTakeover` over the card's `Refs`) and the ONE Reader performs it
(`speak` -> `readScript`). `tickerDeck.breaking` and `railBurst` are retired, and with them the
`running` slot, the `breakers` wait set and the dead `diverted` counter.

**What each retirement cost, checked rather than assumed.** The `running` slot enforced "one takeover
at a time" by DROPPING a burst that arrived while another read; the schedule enforces it by QUEUEING
that burst (DR-3), which is what MVS-D-56 asked for. `breakers` waited for a read on a goroutine the
ticker owned; the read is an effect now and the pump drains every dispatched effect before it stops.
`diverted` was written every cycle and read by NOBODY — DR-14's notice is a `DivertNotice` card at
T4.3, so the counter was a stub for a concept with no consumer (AP-DEAD-01) and is gone rather than
carried.

**Three seams, and each exists because one predicate was doing two jobs.**

`Card.Refs` carries the burst's alerts in read order, and rides on `BuildCard` (BD-8) so the Composer
never has to re-plan or read a lineup the publish is concurrently replacing. The alternative — asking
the producer for the CARD's id — would have resolved to the lead alert and read a burst of five as a
burst of one, with nothing in the audio to say so.

`muted` split from `audible`. One was answering both "the listener said do not speak to me" and "this
machine has no sound card", and the two want OPPOSITE answers: a muted read must be declined so the
alerts stay new, and a voiceless read must go ahead so the band still shows the hazard. One predicate
would have swallowed a hazard or blanked the marquee, depending which way it was pointed.

`mark` is the producer's, not the schedule's. The Director's record of a read is the card's state,
which ends when the lineup lets the card go; the seen-store is what stops the next cycle offering the
same alert. Without it every burst repeats for ever.

**`[M]` was nearly made station-wide, and was not.** `OffAir` exists and MVS-D-78 reads like it fits.
But `[M]` today means "no new takeovers" — a read in progress continues and a tuned relay plays
straight through it — and making it silence the station is **F-26 (Stop All)**, which is unbuilt and
waiting on a HUM LEAD key-binding decision. So mute keeps its meaning and `speak` declines a muted
read instead, leaving the alerts new. `platform/lineup/power.go` was left untouched.

**A cut-short read comes home FAILED, not silent** — and the first version got this wrong. Returning
nothing when the context ended left the card ON AIR for ever with the band holding a callout for a
read that had stopped, which is DR-24's original defect reintroduced by the change meant to generalise
it. The pairing is unconditional now: deciding here whether the schedule is still worth telling is
exactly the judgement DR-24 exists to remove.

**The wiring is pinned where it lives.** `startSchedule` connects the producers itself rather than
leaving it to the caller. It was one forgettable line: without it the rail receives nothing and NO
HAZARD IS EVER READ, while every test stays green because each builds its own schedule. That is
D-12's shape with a far worse blast radius, so `TestTheProducersReachTheSchedule` asserts the wiring
function itself, and the pin was seen failing on its own defect.

**Twenty-five call sites re-authored through a HARNESS, not a shim.** `station_test.go` drives the
REAL path — producer, Director, Composer, Reader — with the pump's concurrency replaced by a loop,
which is what Approach C is for. A test-only `breaking` kept alive beside the shipped path would have
been a pin on code nobody runs.

**Gates.** `verify` (all steps) `p10` `alloc-budget` `pty-severe` `mutant-check` PASS. **One live P10
finding, fixed rather than exempted**: `alertStore.capOldest` used a condition-only loop, bounded in
fact and not in shape; it is the sort-and-trim form `seenStore.capOldest` already uses.

**THE CORPUS GUARD FOUND ELEVEN STALE MUTANTS, which is what it is for.** Nine re-anchored — the rules
survived and the code moved. TWO RETIRED, because their rules were DELETED rather than moved: `m32`
guarded releasing the takeover slot, and there is no slot; `mE2` guarded the ticker not writing the
band itself, and the ticker does not write the band at all now. `mE2`'s rule is real and now has
nothing observable to violate, which is **F-22**'s finding exactly — it wants a gate, not a test.
`mG4` was the interesting one: it applied but left the tree UNCOMPILABLE, because deleting the
ToneRank branch made `cast` an unused import once the takeover's read left this file. A verdict from
an uncompilable tree is no evidence either way, so it disables the branch instead.

**Five new mutants** carry T3.10b's load-bearing rules: `mU0` the producer never tells the Director,
`mU1` the Composer reads the lead alone, `mU2` a muted read is performed anyway, `mU3` a cut-short
read says nothing, `mU4` nothing is recorded as read aloud.

**A process failure worth recording.** Mid-task the five new mutants were run by hand in a shell loop
with `git checkout -- app/` between them, to revert each mutation. That reverts the whole DIRECTORY to
HEAD, and it destroyed every uncommitted T3.10b change under `app/` — four production files and six
test files. `06_docs/mutants/run.sh` exists precisely to prevent this: it refuses a dirty tree and
arms its restore trap only after confirming the tree is clean, with a comment recording that the same
mistake cost real edits at T2.2. Calling the mutants directly bypassed that guard. The work was
rebuilt from the session record and the tree is green, but the rule is: **mutants are run through
`run.sh`, and `git checkout -- <dir>` is never used against a tree holding uncommitted work.**

## T3.10b UAT round 1 (HUM LEAD, 2026-09-05)

**The swap itself passed.** Location radio, `[w]` alerts, the `location -> [w] -> [esc]` fall-back and
an **alert burst** were all exercised and clean. That is the whole claim of T3.10b confirmed by ear.

**THE RELAY-FAULT WINDOW WAS DEAD IN THE HAND**, and all three symptoms were one shipped window with
three defects — every one of them invisible to a green test suite.

*The countdown never ran, and the frame never redrew.* `tickNeeded()` decides whether a 300 ms tick is
armed, and it did not list `modalRelayFault`. So `stepRelayFault` was never called: the clock sat at
`<10>` for ever, the auto-fall-through could never fire, and nothing redrew between key presses — which
is what "reactions were slow" was. A window that ACTS BY ITSELF must keep the clock that acts.

*The arrows worked and the listener could not see it.* Focus moved correctly; the row was marked with a
bare `>` and NO COLOUR — the raw rendered line carried no ANSI at all. `platform/render/list.go` exists
precisely so every list-shaped surface marks focus the same way (D-1), and this window was marking it
its own way. It goes through `o.ListMark` and `render.ListLabel` now. The mock's geometry is unchanged:
`" " + ListMark + " "` puts the pointer and the colons in exactly the columns the HUM LEAD drew. The
aligned-colon style is left alone — that is the mock's, and changing it is a UX ruling, not a fix.

**WHY NOTHING CAUGHT IT, AND IT IS D-12 FOR THE THIRD TIME THIS RELEASE.** Both existing tests reached
PAST the keyboard: the wrap test calls `handleRelayFaultNav` directly, the clock test calls
`stepRelayFault` directly. Both green, both blind to a window that is dead in the hand. The new pin
starts at `tea.KeyPressMsg` and asserts the RENDERED FRAME changes. A second pin turns colour ON
deliberately — with colour off, every tint assertion is satisfied by the bare glyph that shipped, so a
tint pin that cannot see tint is not a pin. Both were seen failing on their own defect, and both now
carry mutants (`mV0`, `mV1`).

**`mU2` SURVIVED its first run**: the muted-read decline had no test at all. Pinned now, with an
UNMUTED CONTROL so it cannot pass against a read that never works. `mU0`, `mU1`, `mU3`, `mU4` CAUGHT.

**THE MUTANT RUNNER ITSELF IS FIXED.** Its restore trap was `git checkout -- $(git diff --name-only)` —
every modified file in the tree. The clean-tree check guards the START of a run, not the whole of it, so
when a backgrounded run finished while `modes/tty` was being edited it silently reverted that work. It
now records exactly what the mutant touched, at the moment it is applied, and restores only that. This
is the second time this instrument destroyed work in one session; the first was calling the mutants by
hand and bypassing the guard entirely.

**ROUND 2: THE ARROWS WERE STILL BROKEN, and the first two fixes were not the cause.**

The cursor WAS moving. Something was moving it back. `silenceReader` fires once per STREAM, and when a
mount goes quiet the engine falls through to the next on the tune list — a new mount, a new reader, a
new detection, a new `RelaySilentMsg`. A station broadcasting silence on EVERY mount, which is the
weatherUSA outage this window exists for, reports about every five seconds. `openRelayFault` rebuilt
its state on each one, so the cursor snapped back to the first row and the clock restarted at ten.
Arrows that do not work and a countdown that never counts, from one line.

A repeat is not a second event now, the same rule `onPowered` and `onArrived` already carry. Once the
window has been ANSWERED — which closes it — the next report raises a fresh one, because the other
half of the rule would be a station that goes silent and says nothing the second time.

**WHY IT WAS MISSED, AND IT IS NOT THE SAME LESSON AS THE FIRST TWO.** The first round's pins were
right about their own defects: the tick genuinely was never armed and the focus mark genuinely had no
colour. Both were real, both needed fixing, NEITHER was what the listener hit. Every test in the file
— including the two new ones — opens the window ONCE. Model-level, key-level and render-level were all
correct and all blind, because the reproduction needs a SECOND message and none of them sent one.
Reproducing the reported sequence beats reproducing the sequence the tests can already do.

`mV2` carries it.

**ROUND 3: THE FRAME WAS MEMOISED, and that was the defect all three rounds were reporting.**

The decisive clue was the HUM LEAD's: *"the modal did disappear on its own and the channel fell through
to synch"*. The clock ran, reached zero and fired — so the state machine was perfect and only the
DISPLAY was frozen.

`modalKeyFor` builds the modal memo's key from the model, with a `switch d.modal` naming what each
window shows that MOVES: Setup's generation, Status's second, Severe's row, Details' minute. There was
no case for `modalRelayFault`. The key never changed while the window was open, so it rendered once and
the memo replayed that identical frame. The arrows moved the cursor, the countdown counted, the
fall-through fired on time, and the listener saw a still picture.

**`modalDebug` HAD THE IDENTICAL HOLE**, found while fixing this one: the `ctrl+d` window's arrows were
equally dead. A rule with one instance is an anecdote, so the pin is a TABLE.

**AND IT IS WHY EVERY PIN PASSED.** They all called `renderModal` — the memo's MISS path. `modalView`
is what the app paints from. A pin that starts below the cache cannot see a cache bug, exactly as a pin
that starts below the keyboard could not see the tick, and a pin that opens the window once could not
see the repeat.

**FOUR REAL DEFECTS IN ONE WINDOW, and three fixes reported before the reported one was found.** The
tick, the tint and the repeat-reset were all real and all needed. All three were MASKED by the memo:
with the frame frozen the listener would have seen nothing regardless. The through-line is that each
pin sat one layer below where the bug lived — past the keyboard, then past the second message, then
past the cache — and each round's fix was correct about a defect nobody had reported.

Recorded as **F-30**: `modalKeyFor` is hand-maintained and an omission is SILENT, the same shape as
F-18's `aaPairs`. A new window with a cursor is frozen by default and nothing fails. Broadcaster's
operator surface is all cursors.

`mV3` and `mV4` carry the two cases.

**ROUND 4: THE MARGINS, AND THE TRUNCATION THAT SHOULD NEVER HAVE BEEN WRITTEN.**

The content ran into the right border. Two causes: the footer sat at two cells either side while every
line above it sat at three, and the ROWS had no right bound at all — the mock's placeholders are short,
so every line happened to clear the border and nothing measured whether it had to. A real NOAA mount
reads `KEC62 - San Diego, CA (NOAA Weather Radio All Hazards, 162.400 MHz)` and ran straight into it.

**The first fix was TRUNCATION, and that was wrong on a ruling that was already written down.**
`render.WrapLines` carries it in as many words: *"THE modal-content guarantee: floating windows wrap,
never truncate — callers cannot reintroduce the truncation class of bug (UAT 25)."* In THIS window it
is worse than untidy: what gets cut is the address of the station the listener is being told to tune
to, in the one window that exists because something is already wrong.

**`render.WrapHanging` is the other half of that guarantee.** WrapLines keeps a modal's PROSE inside
the box and preserves a line's own indent; a LABELLED row needs its continuation under the VALUE, or
the wrap reads as another way out. The rows are separated by a blank line, to the HUM LEAD's shape:

	›  Recommended:  Tune to WNG712 Coachella / Spanish CA
	                 162.525 MHz (81 mi)

**`PlainLine` TRIMS, and it cost two passes.** It measured `" [esc] Close"` as 11 instead of 12, and
the row's head — which begins with the mark's padding and ends with two spaces — four cells short, so
the row was allowed four cells too many. Any string whose own padding is load-bearing must be measured
with `Plain`.

**`modalInset` has one owner.** `columnMargin` already carried the ruling in its own comment ("3,
matching the 3-cell inset on the left", UAT 2026-08-28), with ad-hoc `"   "` literals in the windows
that remembered and nothing at all on the right.

**A mutant SURVIVED and the test was fixed rather than the verdict shrugged at.** Removing the body's
bound failed nothing, because the prose lines are literals that cannot be too long — a guard nobody can
falsify (D-2). It is asserted by handing the helper an input that does not fit, and the assertion is
that every WORD survives, which is the difference between wrapping and cutting.

**Open, and NOT actioned — they are HUM LEAD content rulings, not wrapping:** `Fall-Back` vs
`Fall-Thru` and its two-line wording; and `162.525 MHz (81mi)` vs today's `· 81 mi`, which is
`radioDeck.label`'s format and shared with other surfaces. The language token needs nothing: NWR
records it in the SITE NAME, so "Coachella / Spanish" is already what the label produces.

`mV5` and `mV6` re-anchored to the wrap — both had guarded the truncation this round removed. `mV1`
re-anchored to the row's new head.

**Gates.** `verify` `p10` `alloc-budget` `pty-severe` all PASS.

**Next: UAT and the red team** (HUM LEAD). F-21b's injector exists so the UAT need not wait for
weather.

**Design settled on the way (HUM LEAD, 2026-09-05).** The operator's view is a PROJECTION of the one
Lineup, not a second schedule; the "machinery tier" is the closed EFFECT SET, which is deliberately not
schedule entries — putting them there would make the Director schedule its own work, which Approach C
exists to prevent. The one-card model REDUCES the gap between what the operator sees and what the
machine holds, rather than creating it.

**Raised, not yet ruled:** "once a card is LIVE, only a takeover may interrupt it". The code does not
express this — `Card.CanBecome` allows ON AIR → Discarded from any cause. Wants a decision when
operator editing arrives.

## T3.10 RED TEAM (2026-09-05) — 32 findings, three lanes

Scope: `588e724..a3ad075`, ~2,300 insertions. Framed at SIMPLIFICATION AND QUALITY rather than
regression hunting, because the UAT had already covered behaviour by ear. Three fresh reviewers, one
per surface — seams and the role model, test quality, the TTY modal and render — none carrying the
author's context.

**THE HEADLINE WAS A CLASS NOBODY ASKED ABOUT: THE WIRE IS NOT PINNED.** Five places where the thing
could be DELETED and everything stayed green.

| Deleted | Result |
|---|---|
| `schedule.carry`'s body | **`go test ./app` passes.** Every producer event dropped, the rail dead |
| `onTick`'s body | `./modes/tty` passes. The countdown never moves, auto-close never fires |
| The `enter` branch for the fault modal | `./modes/tty` passes. Enter tunes nothing, reads nothing |
| The producer's mute gate (MVS-D-78) | **`go test ./...` fully green** |
| The slow-voice swap in `TestNoAdmittedAlertIsDroppedUnread` | Passes — the two runs were the SAME run, compared with itself, for 14 s |

The first is the sharpest lesson of the release. `TestTheProducersReachTheSchedule` was written two days
earlier FOR exactly this — its own comment says "a rail nothing ever reaches, reading no hazard at all,
silently" — and it asserted `emit != nil`. **A NON-NIL POINTER TO A NO-OP IS THE SAME SILENCE.** The
D-12 pin was built and then pointed at a pointer instead of at a hazard. It drives a real burst from the
producer and waits for the seen-store mark now: nothing short of the whole path turns that flag true.

**FIVE SHIPPED DEFECTS THE UAT COULD NOT REACH.** At 80x24 — the documented floor — every way out of the
fault window was below the fold, because `floatModalFooter` scrolled by SETUP's body whatever window was
open; the arrows moved the cursor and the screen did not change, which is the reported symptom reached by
geometry rather than by the memo, and the blank lines between rows had made it worse. F-30 had a THIRD
window (`severeReadPause`). The margin fix held only at exactly 84 columns — at 90 the footer's inset was
2 against the body's 3, the same defect one size down. `mark` took an event and re-looked-up an id it
already had, so an evicted record meant an alert READ TWICE. `runCue`'s live branch was unreachable, so
the band record logged releases with no cues.

**THE SIMPLIFICATION, which was the brief.** `Burst.Cards` DELETED — renaming it and building proved it
had no production consumer, while every ordering pin asserted it; reversing the takeover's read order
was invisible before and fails **18 tests** now. `cardsOf` went with it (a Propose and a To(Admitted)
per alert, per burst, for a field only tests read). `BurstHead` and `DivertNotice` retired — nothing
proposed either, against the registry's own AP-DEAD-01 comment, which is what T3.7 promised and did not
do. The `publish` seam dropped (a no-op behind a nil-guard, dispatched every second); the effect stays,
because `settle`'s "readers are told last" depends on it. A vacuous `Divert >= 0`, two unfalsifiable
guards, and three redundant tests gone; `runCue` folded into `cueFor`.

**F-30 RETIRED AS A CLASS.** A differential guard: reflect over every field, perturb one at a time,
assert `renderModal(a) != renderModal(b) => modalKeyFor(a) != modalKeyFor(b)`, over every modal DERIVED
from the enum. Two things about building it are the point. It first reported a dozen failures that were
the TEST'S OWN artefacts — it held `render.Opts` fixed where production recomputes it per frame, and
wrote Setup's state without the `touch()` every real writer calls. And on the first pass it caught the
relay-fault and debug bugs and MISSED THE SEVERE ONE, because the fixture had no read in progress: a
fixture that does not exercise the state is a hole shaped exactly like coverage.

**ONE FINDING WAS WRONG, AND MEASURING IT WAS WORTH MORE THAN FIXING IT.** The reviewer reported the
`[S]` footer at 1 cell against its rows at 3. It is 3/3, at 133x44 and at 80 — the claim had read
`" "+o.Controls(...)` without counting the panel's own two, which is the arithmetic that has tripped
everyone in these files all release. Measuring EVERY window instead found four real ones nobody had
reported: help, details, remove and debug. Debug is fixed (it is this work's sibling); the other three
are appearance calls, LISTED as visible exceptions in `TestEveryWindowClearsItsMargins`, which fails if
one starts clearing its margins while still listed. A margin claim is now answered by running a test.

That fix produced what the reviewer actually asked for: **`insetModalLines`, a shared HELPER rather than
another constant.** `modalInset` alone could never be the single owner — each window that remembered the
ruling did its own arithmetic against it, in the panel's coordinates instead of the body's, and each got
a different answer (2, 3 and 4 across the app, with the right margin unenforced). The number is not the
rule; APPLYING it is. P10 then caught the helper's own `panelInset` parameter, which both callers passed
2 to: it is `panelSide = panelFrame / 2` now, the panel's number rather than each window's.

**The corpus guard found ELEVEN more stale mutants across the round** — five in the first pass, six after
the simplification — every one re-anchored. `mT0` could no longer compile once `Burst.Cards` was gone, so
it expresses "one card per alert" by splitting the takeover back into one card per ref.

**F-25 WAS A DUPLICATE OF F-14**, found when the race gate hit it a third time. The output is captured at
last and confirms F-14's diagnosis exactly rather than merely matching it: the take-over came back as the
introduction ALONE. 0 failures in 20 isolated runs under `-race`; it fails only under full-suite load.
Folded into F-14; the fix is its own `domains/radio/synth` task, deliberately not attempted here.

**Left open:** `Escalate.ID` may be write-only (SUSPECTED, unverified by the reviewer); DR-14's divert is
unwired end to end and that is T4.3; help/details/remove margins are HUM LEAD appearance calls.

**Gates.** `verify` `p10` `alloc-budget` `pty-severe` `mutant-check` all PASS.

## Phase 4 (2026-09-05) — two built, one DEFERRED

### T4.2 — the application's default origin

`snapshot.DefaultOrigin()` is Bonsall, CA, compiled in and SHOWN in Settings as the Default. R-4's
two halves fail in opposite directions and only one is visible, so both are pinned: it is shown, with
its zip, and SAID to be the application's rather than the listener's — and it is **not used**, reaching
neither the watchlist nor the snapshot. A station that quietly scoped a listener's hazards to somebody
else's town would look like it was working.

A function rather than a package variable, so it cannot be written (P10-06), and the "not yours yet"
note is on the SUPPORT line rather than tinted onto the question, per the group's own convention.

### T4.3 — the divert notice, and DR-14 finally wired

The burst now closes with what it left out: *"For more details on these and 4 other alerts, press W in
Watchpost."*

**THIS NUMBER HAS BEEN COMPUTED SINCE T1.3 AND NOTHING EVER CARRIED IT.** The ticker's `diverted` field
was written every cycle and read by nobody; the `DivertNotice` slot was proposed by nothing. Both were
retired as dead at the red team — this is the carrier they were standing in for.

**The count travels ON THE CARD** (`Card.Divert`, `BuildCard.Divert`). Only the Director knows it — it
is the difference between what the fence admitted and what the Max allowed — and a Composer that
recomputed it would be a second planner disagreeing with the first about a number that is SPOKEN.

**It counts CANDIDATES, not raw arrivals.** DR-14's wording says "arrivals − alerts read", and m72
already ruled the resolution: *"a forecast the listener never lost is spoken as an alert they did."*
The existing arithmetic was right; it is now asserted through the planner rather than against a
literal, because DR-14's own check is "the spoken count equals exactly what was not read".

**ONE SCRIPT, NOT TWO (HUM LEAD, 2026-09-05).** The first pass wrote a second file for the divert
reading, reasoning that "these and 0 other alerts" is unsayable. Harmonising the opening removes the
need: the two readings differ only in the middle clause, the count guards its own clause, and a second
file would restate the opening and the destination and let the pair drift. Singular agreement stays in
CODE — the grammar rule is the same in every script and does not belong in the wording tree.

### T4.1 — DEFERRED to post-Broadcaster (MVS-D-79)

**THE ORDERING SHIPS; THE KNOB DOES NOT.** The ladder, the significance fence, the global Max and the
divert count are built and pinned. What is deferred is the listener's ability to REARRANGE it.

The HUM LEAD's reasons: 0.14.0 has already changed a great deal and this is a point release of its own;
and *"I'm not sure if we want to give human users the ability to mess with this just yet."* The second
is the weightier — a knob on hazard ordering is a knob on which hazards a listener HEARS, and
Broadcaster is the surface that will show what a rearrangement actually does.

**THIS SUPERSEDES MVS-D-56'S SHIPPING CONDITION.** Four documents asserted "0.14.0 does not ship until
this is built"; each now carries the supersession rather than being left to contradict the plan.
Recorded as **F-31**, which also carries the two rules the row model must be able to express before it
is built (MVS-D-60's proximity weight and absolute pin) — neither built, neither yet in the deferred
scope's estimate. **F-17**, the original backlog placeholder, is closed into it.

**What T4.1's absence leaves in place:** `category.ReadRank` stays the compile-time source of the
ladder and `defaultBurstMax = 5` the fixed Max. Both are ratified defaults, not placeholders.

## Phase 5 — the P10 ledger reconciled (PL-14, 2026-09-05)

**MIRRORED HERE BECAUSE THE LEDGER IS GITIGNORED (PL-16).** "Presented for ratification" is meaningless
without naming where, and `.a2dh-p10-exemptions.yml` is not in the tree; this section is the reviewable
record.

**PL-14's arithmetic was wrong, which is the first finding.** It carried "seven entries reading *Ratify
at the ⟨X⟩ gate* against the six carrying an explicit ratification". The ledger holds **127 entries**,
of which **six** are pending and **four** carry a ratification. The counts were never re-derived after
being written down.

**AND EVERY PENDING ENTRY NAMES A GATE THAT HAS ALREADY PASSED.** Four say "the 0.13.0 UAT/REVIEW gate",
one "the 0.13.0 REVIEW gate", one "the P1 gate". **0.13.0 SHIPPED** (`v0.13.0`, 2026-08-29). These were
carried through the gate that was supposed to settle them, and the ledger has said so in plain text
ever since. Nothing self-approves them here.

### RATIFIED, HUM LEAD 2026-09-05 ("exemptions ratified")

All seven carry `RATIFIED by the HUM LEAD 2026-09-05` in the ledger, and **zero entries now read
"Ratify at the ⟨X⟩ gate"**. PL-14 is closed: the reconciliation found the arithmetic wrong, found every
pending entry naming a gate that had already passed, and settled all of them at this one.

| # | Where | Rule | What it is |
|---|---|---|---|
| 1 | `app/cast.go` (`runtimeGOOS`) | P10-06 minimal scope | The PRESCRIBED production seam for the platform (0.14.0 P1 Task 1.12), written only by tests which restore it via `t.Cleanup`. It exists so the M5 fallback matrix can walk BOTH platform namespaces on one machine; without it half the matrix never runs |
| 2 | `app/director.go` (`admit`) | P10-02 bounded loops | The Station Director's turn wait: a `sync.Cond` loop, one Wait per iteration, ended when the job is first with nothing on air or its context ends, checked on every wake |
| 3 | `app/director.go` (`awaitAir`) | P10-02 bounded loops | The suspended sequence's wait for the air — the same shape as `admit` |
| 4 | `domains/radio/script` | P10-05 invariant density | A file loader over an embedded tree; the listing and lookup functions assert nothing, and the guards live where wrongness harms (template syntax never reaches speech; a read never leaves the tree) |
| 5 | `domains/radio/pronounce` | P10-05 invariant density | A table loader over five embedded files, same pattern |
| 6 | `platform/plaintext` | P10-05 invariant density | The plain-text boundary (0.13.0 REVIEW R5-C-05): one pure rune filter over provider and file text, pinned by the escape corpus, bidi and one-line tests |
| 7 | `app/inject_release.go` | P10-08 directive discipline | **The one this release added, and the only one where the exemption IS the safety property.** A `//go:build` split that is a SAFETY boundary rather than a portability one: it is removable only by giving up the guarantee that a shipped binary cannot fabricate an alert. Presented at F-21b, ratified here |

**Ratified, for contrast:** `app/every.go` and `app/pump.go` (P10-02), `platform/category` and
`platform/lineup` (P10-05) — the last carrying the explicit "RATIFIED by the HUM LEAD 2026-09-03".

### The rest of Phase 5

**AA register — CLEAN.** Twenty-six render tokens were added this release and every one is in
`contrast.go:aaPairs`, including `ListPointer` and `ListFocus`, which the relay-fault window began
using at the T3.10 UAT. F-18's gap is not open for 0.14.0.

**Goldens.** Twenty-odd moved across the release. DR-19 requires each to be listed with its reason
rather than re-litigated; the two this session moved are `relay-fault.mock` (the footer's inset and the
title's true centring, both HUM LEAD rulings) and `plan.golden` (DR-13's fence, the one deliberate move
the plan named in advance). The remainder belong to earlier tasks — the wordmark, the key rebinds, the
window rename, the Settings groups.

## The transition scripts (F-27, F-24 — HUM LEAD wording, 2026-09-05)

**WRITTEN AND PINNED; NOT YET WIRED.** Both render and both are asserted. Nothing calls either, and
that is stated here so the entry is not read as done.

### The resume transition (F-27)

> *"Watchpost Radio now returns to its regularly scheduled programming."*
> *"Watchpost Radio now returns to its regularly scheduled programming, which is already in progress."*

**ONE SCRIPT FOR BOTH MODES**, HUM LEAD's ruling. What differs is only whether the programme kept
running underneath: a LIVE RELAY did, so the listener is rejoined to something already under way; a
synth read did not. The pin asserts the two readings differ by EXACTLY that one clause, so a later edit
cannot quietly turn them into two sentences.

It is what makes the return audible. Without it a read simply stops and the next thing begins in the
same voice — reported at UAT as jarring in synth mode, and a human station never does it. The seams it
serves are the HUM LEAD's own two: `<location>` → `[w]` read → `[esc]`, and `<location>` → `<alert>` →
`<location>`.

### The masthead (F-24)

> *"This is Watchpost Weather Radio. Broadcasting from Bonsall, CA and covering weather observations,
> alerts, fire data, and seismic reports in a 50 mile radius. Data is compiled from the National
> Weather Service, the United States Geological Survey, and NASA FIRMS. Data may be delayed or
> incomplete from these providers and is not intended for life safety use."*

**ITS LENGTH IS THE POINT.** Coming back on air the first card must be composed and rendered before
anything can be said, so the moment a listener says "go" is otherwise the moment of the longest
silence. Every clause is still load-bearing: where, what area, whose data, and the limitation a weather
service is obliged to state.

**`0` READS AS "ALL LOCATIONS", not "a 0 mile radius"** — the ALERTS - EVENTS `0 = All` reaching the one
place that value is SPOKEN. A station announcing a nought-mile radius states the opposite of what it
covers.

**`spokenList` was EXTRACTED, not written twice.** `burstAgencies` carried the "A", "A and B",
"A, B, and C" joiner inline; the masthead is the second caller.

### What is not wired, and why

- **The resume transition** needs the Director to detect the seam and insert a `Transition` card —
  now the ONE structural slot, with a working precedent in `stale.go`. The wrinkle: `platform/lineup`
  cannot import `domains/`, so the words must reach it through a seam the app installs, exactly as
  `synth.SetHandoffLine` already does.
- **The masthead** needs a STANDBY → ON AIR trigger, and `OffAir` is emitted by nothing: mute stays "no
  new takeovers" deliberately (F-26, awaiting a key-binding ruling). Its trigger arrives with
  Broadcaster, which is where the HUM LEAD placed the need.

**A verification failure worth recording.** The scripts were added and checked with
`go test ./app -run "Transition|Masthead"`, which was green — and the new `transition` report broke two
exact-match report lists in `domains/radio/script` plus a declset. The gate sweep caught all three.
Running the PACKAGE THAT CHANGED, not the package being looked at, would have caught it in seconds; it
is the same shape as the memo bug one layer out.

## Still open

| # | Task | Note |
|---|---|---|
| **F-D1** | The rail wedges on a card that arrives with its words | **Fixed** — three attempts, two review rounds |
| **F-D2** | The band is shared, and the pump's grouping does not know it | **Fixed** — two attempts, two review rounds |
| ~~T2.3~~ | `mastercontrol` — the effector half of `app/director.go`'s split | **DONE, green on `2b3ead8`.** It owns `Duck`/`Restore` over `engine.Suppress`/`Restore`, as MVS-D-67 requires |
| **F-D5** | The dip and the claim were not one critical section | **Fixed** — one attempt on the defect, three on the pin |
| — | `holdLine`/`resumeLine`/`dropHeld` touch the voice with **no lock at all** | **Carried, unverified.** Safe from a data race (the voice is immutable after construction), but those calls are not serialised against `dip`'s `duck()` and `takeBack`'s `restore()`, which are locked. Raised by review as a suspicion it did not exercise; **pre-existing, not introduced by T2.3**, so it is not being fixed speculatively at a task boundary |
| — | `holdLine`/`resumeLine`/`dropHeld` — now also **F-34** in `06_docs/follow-ups.md` | The row above lived only here, which by that file's own preamble means it was not carried at all (BUILD-exit red team, R-6) |

Phase 1 (T1.1–T1.6) is complete; the earlier "still open in Phase 1" table that listed T1.6 was
stale from the moment T1.6 landed. **T3.1 was in this table too, and was wrong** — it has its own
completed section above and T3.10b took the rail live; removed at BUILD exit (R-6), one line under
the sentence congratulating this table for catching exactly that.

## The rulings this release made and did not write down (BUILD-exit red team, R-2)

`06_docs/` is **the record**. These two were cited in code as HUM LEAD rulings and appeared nowhere
in it, which is the practice that produced MVS-D-29…46 lapsing on a SEV-0 audio surface.

**MVS-D-76 — the relay-fault window** (HUM LEAD, UAT 2026-09-05). Cited 18 times in code
(`domains/radio/player/engine.go`, `modes/tty/relayfault.go`, `modes/tty/dashboard.go`,
`app/radio.go`) and in no document. The ruling, as the code implements it: **five seconds** of a
relay producing no audible audio is a fault, not a hiccup (`engine.go` — "Five seconds, ratified by
the HUM LEAD"); the window offers **three ways out** — the recommended next-best relay, an alternate,
and a fall-through to a synth read of the Watchpost report; and with nobody at the keyboard it
**auto-closes after ten seconds and takes the fall-through by itself** (`relayFaultSeconds = 10`).
The auto-fall-through is the load-bearing half: the station retunes without the listener, which is
why the window is a safety surface and not a notification. Its accessibility consequences are **F-33**
(the window is entirely inaudible) and **F-36** (the ten seconds is a hard timeout on the app's only
station-tuning control, WCAG 2.2.1).

**MVS-D-80 — the inter-card transition rule** (HUM LEAD, 2026-09-05). Recorded in `follow-ups.md`
F-27 and named in a commit subject, but with no entry here. Fires when something that interrupted the
programme leaves the air: a takeover, and a `[w]` read including one cut short by `[esc]`. A **ducked
relay gets one**, with the "already in progress" clause. It does **not** fire location-to-location
("the location scripts already announce their location") nor between alerts inside a burst.
**Placement is at the duck** (option A).

Carried from PLAN: **PL-14** — the ledger entries still reading "Ratify at the ⟨X⟩ gate". **The
arithmetic in this log was wrong twice** (BUILD-exit red team, R-5). `:2108` asserted "zero entries
now read 'Ratify at the ⟨X⟩ gate'"; five still do, and the re-derivation at `:2098` ("six are
pending") was short by three. The three unaccounted rows are **0.14.0's own**, added by this release
and naming a gate that passed weeks ago:

| Ledger row | Gate named | Ratified? |
|---|---|---|
| `domains/radio/cast/cast.go` — P10-06 | "ratify at the P1 gate" | **RATIFIED 2026-09-06** |
| `domains/radio/cast/tone.go` — P10-06 | "ratify at the P1 gate" | **RATIFIED 2026-09-06** |
| `domains/radio/cast` — P10-05 | "ratify at the P1 gate" | **RATIFIED 2026-09-06** |

**Ratified by the HUM LEAD, 2026-09-06 ("P10 ledger rows approved").** All three are the
immutable-registry shape already ratified twice on this ledger (`platform/render/themes.go`,
`platform/tz`) and are in fact *stronger* than those two: those guard genuinely mutable state under a
mutex, and these have no run-time writer at all. The stale gate name is struck from each row and
replaced with the ratification, so **zero rows now read "ratify at the P1 gate"** — a claim this log
made once before and could not support.

**And the last one.** `domains/globalfeed` — P10-05 read "ratify at the 0.13.0 BUILD gate" and
carried no ratification of any kind. Presented separately, because it was not in the table above and
folding it in silently would have been self-approval. **RATIFIED by the HUM LEAD 2026-09-06.** Same
pure-decode-helpers shape as its neighbour `domains/severe`, which was explicitly ratified at the
0.13.0 REVIEW.

**The fifteen rows in the table above are ratified, and none of THEM names a gate it is waiting for.**

**AND THE SENTENCE THAT USED TO STAND HERE CLAIMED MORE THAN THAT.** It read "the ledger is now fully
ratified … none names a gate it is waiting for", which is true of this table and false of the ledger.
Enumerated at REVIEW exit, 2026-09-06: the ledger holds **127 rows**, and **23 of them still name a
gate they are waiting for** — "ratify at gate", "ratify at B1a gate", "presented for ratification" —
with no ratification of any wording, at gates (B1a, the 0.13.0 BUILD gate, the quality pass's Q0)
that passed releases ago. The claim was written from the table in front of me rather than from the
file, which is the same shape as the `tree_hash` column and the NFR-8 grep recorded green while
failing: **a record asserting a property of something wider than what it checked.**

**This is pre-existing debt, not 0.14.0's.** Every row this release added or changed carries an
explicit ratification with a date (P1's four on 2026-08-30, `platform/category` on 2026-09-01, the
`everyTick` row on 2026-09-03 which REPLACED two, `app/inject_release.go` on 2026-09-05, and R-5's on
2026-09-06). So `release-checklist.md`'s row is satisfied **for this release's scope**, which is what
it is ticked against. The 23 are carried as **F-45** and are NOT swept into a 0.14.0 ratification:
the HUM LEAD ratified what was presented, and presenting 23 rows they were never shown by folding
them into that answer would be self-approval wearing someone else's signature.

**AND THEY WERE THEN PRESENTED, AND RATIFIED.** HUM LEAD, 2026-09-06 ("Ratified"), on the enumerated
23-row list. Each row now carries the ratification **in the ledger itself**, not only in a release
document — which is the actual fix for how this went unnoticed: the ledger was the thing being
audited and the ratification lived somewhere else. After the annotation the file parses to **127
rows, 0 malformed, 0 naming a gate they are waiting for**, and 58 carry an explicit ratification
(35 before). The remaining rows without one are the vendored `third_party/go-studs` kit's and their
neighbours, which never claimed to be awaiting a gate.

**Still wanted, and NOT built here:** a check that fails when a ledger row names a gate that has
already passed. Without it this accrues again and is found only by someone enumerating by hand.

**The ledger is gitignored (PL-16), so this table is the tracked record.**
