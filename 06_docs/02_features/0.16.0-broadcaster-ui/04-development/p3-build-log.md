---
title: "0.16.0 P3 — the main-track producer: the merge, staged"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "P3 COMPLETE except its UAT.  The producer, the property test, the instrument, the red team and THE FLIP have all landed."
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


# P3(b)+(d) — the flip, as built

**Shape B, ratified 2026-09-09.**  The schedule decides WHEN a report is read; the source still decides
HOW it is read.  Full reasoning and the gap table: `p3-flip-design.md`.

## What moved

| | |
|---|---|
| `startSynth` → `readReport` | the same body, **one caller instead of three**, and it BLOCKS until the report ends |
| the caller | the **Speak executor**, inside `voice.Run` — the arbiter holds the air for exactly as long as the report plays |
| the words | the segments the BUILD composed, taken once from a bounded store keyed by card id |
| the end | the deck's own `cycleEnded` releases the waiter, carrying whether the report reached its sign-off |
| `app/maintrack.go` | **deleted**, with the direct path, exactly as it said it would be |
| `compose`, `readReport`, `held` | **REQUIRED seams now.**  The dated obligation recorded at P3(a) is discharged |

**`Speak` gained a `Subject`**, for the reason `BuildCard` carries its slot (BD-8): the reader is chosen
by what kind of read it is, and a location report's reader has to know WHICH location.  Looking it up
from the published lineup would race the publish, which runs in the same step.

## Three decisions taken while building it

**A repeat RE-COMPOSES.**  The composed segments are the first pass — what the console displayed is what
went out — and a loop fetches again.  Replaying a stale observation because the listener asked to hear
it again would be the station lying about the weather.

**`[M]` is the RAIL's rule and only the rail's.**  The mute check sat above the slot switch, so the
merge would have turned the mute key into a stop button: `[M]` means *"do not read me hazards"*, and it
has never silenced the broadcast.  Plants in both directions.

**`leaveTheAir` was extracted at the second caller.**  The rotation is played by its source and a
takeover is read line by line, and both have to end the card by DR-24's rule.  Two copies is two places
for it to drift, and the half that drifts is the half that fires rarely.

## The instrument

**Fifteen plants.  Thirteen CAUGHT on the first pass, two SURVIVED and both found real holes.**

| # | Plant | Verdict |
|---|---|---|
| d1 | the report is narrated line by line instead of played | CAUGHT (after one INVALID: the first form did not compile) |
| d2 | the report plays OUTSIDE the arbiter | CAUGHT |
| d3 | **the composed segments are never stored** | **SURVIVED**, then CAUGHT |
| d4 | the report is left in the store instead of taken | CAUGHT |
| d5 | mute stops the broadcast again | CAUGHT |
| d6 | mute no longer holds a hazard | CAUGHT |
| d7 | a card with no report airs silently | CAUGHT |
| d8 | the read's result is discarded and every read reports success | CAUGHT |
| d9 | the reason is never filed | CAUGHT |
| d10 | **the reason is never taken** | **SURVIVED**, then CAUGHT |
| d11 | the reason is not consumed, so it explains the next read too | CAUGHT |
| d12 | the station never says it is on its own broadcast | CAUGHT |

### d3 — nothing ran the two halves together

**Deleting the store's write changed no assertion.**  The build test looked only at the card's script;
the speak test put the segments in itself.  So the one wire that carries a report from where it is
composed to where it is voiced was untested, and a station with it cut would show a full lineup and read
none of it.

**Closed by `TestWhatTheBuildComposedIsWhatTheAirPlays`**, which builds and then speaks through the same
executors.  **This is the same shape as m6 and p1b**: a gate that watched the state and not the work.
Third instance this batch.

## Two P10 findings the flip raised, and what each one actually was

**`recursion (direct or via readReport)` and `(direct or via segments)` were the SAME false positive,
for the sixth time this release.**  P10 resolves by NAME, so the executors' `readReport` seam and the
deck's `readReport` method read as one node and the pair reported as a cycle.  **The seam is
`playReport` now**, and both findings cleared — which is also the evidence that they were name
collisions rather than real cycles.

**`newExecutors` crossed P10-04's branch bound**, and splitting the checks into `seamsPresent` merely
MOVED the count — the same sixteen branches under a different name, which the meter said plainly.  So
the chain became a TABLE the constructor walks: one branch, every message kept, and the contract now
reads as what it is.  Two plants — a row deleted, and the table walked without acting — are both CAUGHT.

**The bound was right and the first fix was not.**  A rule that only relocates the thing it measures is
not satisfied, and the meter is what said so.

### d10 — the reason had no gate because it had no seam

The detail line is the ONLY place a listener learns why the station is reading rather than relaying, and
it was written inside `readReport`, which resolves a voice — on a machine without one, a 63 MB download
inside a unit test.  **So it was extracted**: `announceReport` is the one carrier, takes the reason, and
needs no audio device to assert.  A rule that cannot be reached by a test is a rule with no gate.


# P3(d) — a regression I introduced, and found by reading what I replaced

**The flip dropped BOTH of `startSynth`'s staleness guards**, and that put a previously-fixed race back
on the path that now carries every ordinary broadcast.

| The old function had | What it was for |
|---|---|
| `if !d.epoch(gen)` at entry | *"a stale fallback must not relabel anything"* — a read the listener has moved on from must not rewrite the station row |
| `tuneMu` around a SECOND `epoch` check and the engine start | `tuneMu`'s own comment: *"without it a Stop landing between the check and engine.Start left audio playing"* (N-3, red team 0.9.0 C-3) |

**`readReport` carried neither.**  Found by diffing the new function against the one it replaced, which
is a step worth naming: a rewrite that keeps the BODY can still lose the GUARDS, and the guards are the
part with no visible behaviour to miss.

**Restored, with the lock held across the check and the start and NOTHING MORE.**  Holding it across the
wait — the report's whole length — would block Stop for minutes, which is the same
silence-that-will-not-stop by a different route.  A plant proved that direction too.

| # | Plant | Verdict |
|---|---|---|
| e1 | the entry epoch check deleted | CAUGHT |
| e2 | **the pre-start epoch check neutralised** | **SURVIVED, and recorded** |
| e3 | the tune lock held across the wait (`defer`) | SURVIVED first; CAUGHT once the position gate existed |
| e4 | the tune lock never taken | CAUGHT |
| e5 | the engine started outside the lock | CAUGHT |

## e2 is a limit, not an oversight, and it is stated rather than papered over

**Both halves of this rule are windows between two statements, and no fixture can stand in one.**  So
the rule is asserted as a POSITION: the check is inside the lock, the start is inside the lock, the wait
is outside.  That catches e3, e4 and e5.

**It cannot judge whether the CONDITION is honest.**  `if false && !d.epoch(gen)` leaves the call
exactly where the walk looks for it.  Tightening the gate to reject that shape would move the goalposts
to `gen == gen` and no further, so the gate stops here and says so.  **The old code had the same window
and the same untestability; the difference is that this is written down.**

## The collision that the walk found on its way

Scoping the walk to `readReport` failed against `livePipelines.readReport` in `dashboard.go` — the
dashboard's fetch cycle, nothing to do with audio, and no business holding a tune lock.  **The receiver
is part of the subject**, and the check says so now.  Seventh name collision this release.
