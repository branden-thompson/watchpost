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

## What P3 still owes

| Owed | State |
|---|---|
| P3(b) the two relay-failure fallbacks | **the seam is in place**; the flip is what routes them |
| P3(d) `startSynth`'s direct path retires | with the flip, in one change |
| the property test over arrival timing | **not started** |
| the dark path RUN, and its comparison | the mechanism is in; the run is not |
| P3's own red team | not started |
| a UAT shared with nothing | not started |
| the go/no-go before P4 | not reached |
| `"compose"` moves from `optional` to `strippers` in `executors_test.go` | at P3(d): it is wired now, but a station running off/dark still works without it |
