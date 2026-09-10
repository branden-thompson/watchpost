---
title: "0.16.0 P3 — the main-track producer: the merge, staged"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "P3(a) LANDED DARK.  P3(b) and P3(d) are the next change, and they land together."
---

# P3(a) — the producer, wired dark

**The batch's defect is two owners of the audio device speaking at once, and the only reliable way to
not have that window is to never open one.**  So P3 lands in two changes rather than one, and the first
one leaves the station audibly identical.

## What landed

| Piece | Where | What it does |
|---|---|---|
| `NeedsRead` event | `platform/lineup/rotation.go` | a location wants a synthesised read.  Tenth member of the closed set |
| `onNeedsRead` | `platform/lineup/rotation.go` | proposes a `LocationReport` card onto `MainTrack`, then settles |
| `ReadID` | `platform/lineup/rotation.go` | the card's identity, **a pure function of the ref** |
| `needsRead` | `app/radio.go` | **the one seam.**  All three synth sites route through it |
| `mainTrackStage` | `app/maintrack.go` | off / dark / live.  **Temporary; deleted at P3(d)** |
| `refFor`, `composeFor` | `app/schedule.go` | a card's key becomes a place again; the deck composes its words |
| the dark decline | `app/executors.go` | a main-track card is refused at the air until the stage is live |

## Two decisions that deviate from the plan, and why

### 1.  The event is `NeedsRead`, not `Rotated`

**The plan named it `Rotated{Ref}` — "the rotation's turn came round".  That name fits ONE of the three
producers.**  `startSynth` had three production callers, and only the first is a rotation turn:

| Site | The fact |
|---|---|
| `radio.go:182` | nothing live carries this location — no relay in reach, or the listener prefers Synth |
| `radio.go:829` | the relay failed while playing |
| `radio.go:1014` | the relay went silent |

**Three facts, one consequence: nobody is carrying this location, so the station must read it.**  PL-6
says the effect and event sets grow *deliberately and once*, and `Rotated` would have forced either a
lie at two sites or three events for one consequence.  `NeedsRead` is what all three are true of.

**It is still a fact, not a request** — the event set's own rule.  The deck is the only thing that can
observe that nothing is carrying a location; whether that becomes a card, where it sits, and whether it
is read twice are all the Director's, and it refuses on a stopped station, refuses a duplicate, and
refuses a malformed need.

### 2.  The merge is staged, and the stage is a switch that dies at P3(d)

`app/maintrack.go` has three states and **the default is today's station**:

- **off** — the deck never tells the Director.  The rotation owns its own audio.  **A listener who has
  not asked for the merge cannot tell this commit happened.**
- **dark** — the need is reported, the card is queued, its words are composed, the console is published,
  and **the card is declined one call short of the voice**.  The producer's decisions are observable and
  comparable against the live path's, which is the observation the plan requires before the merge owns
  the air.  The cost, stated: a report is composed twice while dark.
- **live** — the card reads through the arbiter, and the deck's direct path is not taken.

**At P3(d) this file is deleted** along with `startSynth`'s direct path.  A switch that outlived the
merge would be a second way for the station to behave, which is the thing being removed.

## The instrument

**Twenty-five plants, on the rule rather than the assertion.**

| # | Plant | Verdict |
|---|---|---|
| m1 | the stopped-station gate deleted | CAUGHT |
| m2 | `ReadID` not a function of the ref | CAUGHT (after an INVALID first attempt that did not compile) |
| m3 | the card queued as a `BreakingAlert` | CAUGHT |
| m4 | the card queued as Observer's | CAUGHT |
| m5 | the queue result discarded | CAUGHT |
| m6 | the settle skipped | **SURVIVED — the gate did not exist** |
| m7 | headline `+ ""` | **BAD PLANT: no semantic change.  Redone as m7b** |
| m7b | the headline replaced by the ref | CAUGHT |
| m8 | the subject corrupted | CAUGHT |
| m9b | both proposal error guards removed | **SURVIVED, and BENIGN** |
| n1 | an unrecognised env value reads as live | CAUGHT |
| n2 | dark owns the air | CAUGHT |
| n3 | off reports too | CAUGHT |
| n4 | the epoch guard on the report deleted | CAUGHT |
| n5 | live starts audio as well | CAUGHT |
| n6 | the headline is the key | CAUGHT |
| n7 | the ref is the label | CAUGHT |
| n8 | the dark decline dropped | CAUGHT |
| n9 | the decline swallows the alert rail too | CAUGHT |
| n10 | the decline is unrouted | CAUGHT |
| c1 | the resolver returns the first entry always | CAUGHT |
| c2 | an unknown ref composes an empty report | CAUGHT |
| c3 | the nil-watchlist guard deleted | CAUGHT |
| c4 | the nil-deck guard deleted | CAUGHT |

### m6 — the one that found a real hole

**Skipping the settle changed no assertion.**  The card was queued and nothing ever asked for its words:
the station would show a lineup and play silence.  The test had `_ = fx` in it — the effects were
returned and thrown away — which is the shape of a test that watches the state and not the work.

**Closed by asserting that the step returns a `BuildCard` for THIS card, with the right slot, and a
`Publish` after it.**  Re-planted: CAUGHT.

### m9b — benign, and recorded rather than fixed

Removing **both** error guards after `Propose` and `To(Admitted)` changes nothing: `Queue` runs
`c.check()` and refuses the malformed card itself.  So the guards are not the carrier — `Queue` is.

**They stay**, because a Go function that discards a returned error is worse than a redundant branch,
and `Propose`'s own comment already names this shape.  **What is recorded is that they carry nothing**,
so nobody later reads their presence as the reason a bad card cannot be queued.

## What this cannot see (INST-5)

- **The off and dark stages on a FRESH generation are not unit-tested.**  Those paths enter
  `startSynth`, which resolves a voice — and on a machine without one, installs Piper, minutes of
  network, inside a unit test.  They are covered **structurally** instead: `startSynth` has exactly one
  caller and `lineup.NeedsRead` exactly one construction site, both derived by walking the package AST
  rather than compared against a remembered list.  The behavioural coverage is the P3 UAT.
- **`composeFor`'s success path is not unit-tested** — composing a real report is eleven network
  requests.  What is pinned is the resolution either side of it, including the second-entry case.
- **Both AST walks are blind to a call made from outside package `app`, or through a function value.**
- **Nothing here exercises timing.**  The property test over randomised interleavings of cycle-end and
  alert-arrival is P3's, and it is still owed.

# P3(c) — the timing property test

**A scenario UAT targets precedence and duplication, not timing.**  That was the safety lens's
objection to P3's controls, and it is right: no fixed scenario reaches the case where a cycle ends at
the same tick an alert arrives.  `platform/lineup/merge_property_test.go` asserts the merge's rules over
**randomly interleaved** needs, arrivals, ticks, completions, failures and power changes — with a
driver that services effects at random moments too, because *"the build came home three steps late"* is
itself a timing case.

**~~Reach: 300 runs, 11,629 steps, 2,706 admissions, 772 readings — 157 reports and 615 hazards — and 38
locations legitimately read a second time.~~  THAT LAST FIGURE WAS WRONG AND THE RED TEAM MEASURED IT:
35 of the 38 were HAZARD cards re-aired after a repeat burst, not rotation reads.  The true figure was
3 — one per hundred runs — thin enough that a seed change could have taken it to zero with the gate
still green, on the one case property 3 exists to tell apart from a double read.**

**Corrected, derived from the same `isRead` the report counter uses, and the driver rebalanced until the
case is routine.  Reach now: 300 runs, 60,300 steps, 12,036 admissions, 10,207 readings — 6,671 reports
and 3,536 hazards — 899 locations read a second time, and 8,116 events stepped against a STOPPED
station, where there were none at all.**

| # | Property | Why it is the one that matters |
| 6 | **a stopped programme does not read** | added after the red team measured the power dimension as inert; asked of the DIRECTOR's own power, so a driver that lost track of the station cannot make it pass |
|---|---|---|
| 1 | at most one card holds the air | two would be two voices.  **Counted directly**, because `OnAir` returns "nobody" when its own invariant trips, and a violation would otherwise read as an idle station |
| 2 | no two cards share an identity | an ambiguous address is a card read twice |
| 3 | **no card takes the air more times than it was admitted** | FR-2.5 at the schedule level.  The rotation legitimately comes round to a location again once its card has left, so a bare "read once" would be wrong; the count is what separates the two |
| 4 | a main-track card only takes the air when no hazard is waiting | DR-3, the rail drains first |
| 5 | **a main-track card is only SENT TO BE COMPOSED when no hazard is waiting to be** | found by a plant that survived — see below |

## Property 5 exists because a plant survived

**Reversing the precedence in `Next` is caught by property 4.  Reversing it in `toPrepare` was not** —
and it is the more dangerous of the two.

Preparation is the expensive step (a cold build is ~1 s of network), and **an unready rail card blocks
the air entirely**: `airOnce` takes `Next()` or nothing.  So composing a report ahead of a waiting
hazard does not merely reorder the reads — **it holds the whole station silent** while the tornado
warning queues behind a weather report.  Property 4 cannot see it, because nothing takes the air at all.

## The instrument, checked

| # | Plant | Verdict |
|---|---|---|
| p1a | `Next`'s precedence reversed | CAUGHT (property 4) |
| p1b | `toPrepare`'s precedence reversed | **SURVIVED, then CAUGHT** once property 5 existed |
| p1c | both reversed | CAUGHT |
| p2 | the "already on the air" guard removed | CAUGHT (property 1: two cards named in the failure) |
| p4 | duplicate identities allowed into the lineup | CAUGHT (property 2) |
| p6 | **the producer made a no-op** | CAUGHT **by the reach assertion**, not by a property — 0 reports aired.  This is the instrument checking itself: every property above passes trivially against a station that never reads anything |

**Two plants that did not apply are recorded as INVALID, not as verdicts.**  The first `p1` edit matched
two lines and the script refused it; a "SURVIVED" printed there would have been a lie about a plant
that was never made.

# P3 — the dark run's instrument

**The dark stage was landed with nothing to observe it with.**  The live path already records its
engine transitions and its segments; the NEED that produced them was recorded nowhere at all, so a dark
run would have compared one half of a pair against nothing.

`needs-read stage=<off|dark|live> fresh=<bool> ref=<lat,lon> why=<reason>` now goes to the existing
`WATCHPOST_DEBUG_RADIO` log, **before the staleness check and carrying its answer** — because *"the live
path started a read here and the producer did not"* has two possible causes, and `fresh=false` is what
tells them apart.  The protocol is `07-readiness/p3-dark-run.md`.

**The stage names itself from a registry**, walked by a test rather than listed in one (INST-1), and an
out-of-range stage names itself `undeclared` rather than reading as the safe default: a diagnostic that
quietly reports `off` for a corrupt value hides the one case worth seeing.

| # | Plant | Verdict |
|---|---|---|
| s1 | two stages share a name | CAUGHT |
| s2 | a stage left unnamed | CAUGHT |
| s3 | an out-of-range stage reads as `off` | CAUGHT |
| s4 | the record deleted | CAUGHT |
| s5 | **the `radioDebugOn()` gate at the call site removed** | **SURVIVED, and BENIGN** |
| s6 | the record omits the staleness verdict | CAUGHT |
| s7 | the record names the location by label instead of key | CAUGHT |

**s5 is benign by design and is recorded rather than fixed.**  `debugLog` is already gated inside — an
unset `WATCHPOST_DEBUG_RADIO` gives `radioDebugPath()` an empty path and nothing is written — so
removing the outer guard changes no behaviour, only whether the line is BUILT.  That is what
`radioDebugOn`'s own comment says it is for.  **No allocation gate was added**: that guard earns its
place on the takeover path M4 measures, and `needsRead` runs a few times a minute.  A gate here would be
theatre, and the honest record is that this guard is convention, not a load-bearing rule.

**Two plants did not apply and are recorded as INVALID, not as verdicts** — `p1` and `s4` each matched
two lines on the first attempt and the script refused them.

# P3 — the red team, and what it cost

**It found a BLOCKER, and it found it in the one place the staging could not look:** the `live` stage
could never start the station at all, and `dark` is structurally green on it.  Full disposition of all
twelve findings, and the fixes, in `08-reports/red-team-p3.md`.

**Three of my own gates were wrong and are recorded as such:** both seam AST walks skipped package-level
declarations, the property test's power dimension was inert, and the reach figure above was measuring
the wrong thing.  Each was found by the red team, not by me, and each is now caught by a plant.

**One of my own fixes then broke a gate, and the instrument caught that too.**  Rewriting the identity
test to satisfy a linter lost the ADJACENCY that made it work: comparing across two passes let a
counter-based mutant through, because with three refs and a counter taken modulo three every ref got the
same suffix on both passes.  Two adjacent calls differ under any per-call variation, whatever its
period.  **A cosmetic rewrite of a test is a change to the instrument.**

## What P3 still owes

| Owed | State |
|---|---|
| P3(b) the two relay-failure fallbacks | **the seam is in place**; the flip is what routes them |
| P3(d) `startSynth`'s direct path retires | **BLOCKED ON A RULING** — see `p3-flip-design.md`.  Retiring the direct path also retires the rotation's end signal, the marquee, the handoff lines, repeat-one and the player row |
| the property test over arrival timing | **DONE** |
| the dark path RUN, and its comparison | **the mechanism, the instrument and the protocol are all in.  The RUN is the HUM LEAD's** — it needs a real station, a real relay and a period of use |
| P3's own red team | not started |
| a UAT shared with nothing | not started |
| the go/no-go before P4 | not reached |
| `"compose"` moves from `optional` to `strippers` in `executors_test.go` | at P3(d): it is wired now, but a station running off/dark still works without it |


# P3's plants become a corpus — the fix for "a measurement that happened once"

**Forty-odd plants ran across this batch and none of them entered the corpus.**  They were `for` loops
in a terminal, verified once and gone, and the next attempt at this work would have inherited nothing
from them.  That is recorded as **P-10** in the post-mortem and this is it being discharged.

**Ten entries, chosen for DISTINCT RULES rather than for coverage.**  The corpus is 172 → **182**.

| # | The rule it deletes | Verdict | Caught by |
|---|---|---|---|
| **m100** | the settle after queueing a rotation card | CAUGHT | `TestASynthesisedReadBecomesAMainTrackCard` |
| **m101** | `ReadID` as a pure function of the ref (FR-2.5) | CAUGHT | `TestTheMergedStationHoldsItsPropertiesUnderRandomTiming` |
| **m102** | a stopped station admits no read (DR-3) | CAUGHT | `TestNeedsReadOnAStoppedStationQueuesNothing` |
| **m103** | the rail drains first, in `Next` | CAUGHT | `TestTheRailDrainsBeforeTheMainTrackThroughStep` |
| **m104** | the rail is PREPARED first, in `toPrepare` | CAUGHT | the same property |
| **m105** | one card holds the air | CAUGHT | `TestTheNextCardIsBuiltWhileThisOneReads` |
| **m106** | no two cards share an identity | CAUGHT | `TestTheLineupRefusesACardItCannotAddress` |
| **m107** | a stopped programme does not read (PD-1) | CAUGHT | the property test |
| **m108** | **the programme can start at all** | CAUGHT | `TestTheDeckReportsThatTheProgrammeIsRunning` |
| **m109** | a stale need queues nothing (N-3 / C-3) | CAUGHT | `TestAStaleNeedIsNotReported` |

**Ten run, ten CAUGHT.**  The two that matter most are the two that once SURVIVED: **m100**, because the
test held the step's effects and discarded them, and **m104**, because every property watched what took
the air and this defect stops anything taking the air at all.

**m108 is the shape of a blocker that shipped.**  The Director begins Stopped on purpose; with nothing
telling it otherwise every card is refused, the lineup fills, and **no fault is raised** — nothing
failed and nothing was ever admitted.  A permanently silent station is the hardest defect to see, and
this is the one wire that prevents it.

## Why the corpus and not another shell loop

`make mutant-check` iterates every entry and asserts it still **applies and compiles**, and it runs in
CI.  **So a corpus mutant cannot silently rot:** edit `rotation.go` so m100's target text no longer
exists and the build fails.  That is the durability the loops never had, and it is what makes *"a plant
worth running is worth keeping"* a gate rather than an intention.

**The flip's own plants are NOT here**, and that is deliberate: d1–d12 and e1–e5 target code the revert
removed, so an entry for them could not apply.  They are recorded in the build log above and will be
re-planted against whatever replaces the flip.
