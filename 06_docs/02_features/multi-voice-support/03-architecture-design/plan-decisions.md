# PLAN decisions — PD-1 … PD-6

**PLAN, 2026-09-01 · SEV-0 · HUM LEAD.** Each carries a recommendation with reasoning, as DISCOVER
promised. Architecture: **Approach C approved** (`director-architecture.md`).

**One principle runs through four of these, and it is worth stating once.** This codebase forbids
dead code (AP-DEAD-01), and a **stub for a concept nobody consumes is dead code**. A seam is not
preserved by adding an unread field — it is preserved by making sure the thing that exists now is
*named for its own job* and shaped so a later reason can join it. Every "don't build it" below rests
on that, not on schedule.

---

## PD-2 — Standby vs. line-by-line — **RESOLVED by measurement**

**The question as posed was the wrong one, and P-1 found out why.** The charter and S-5 were arguing
about pre-rendering *audio*, which `synth.Source` already does: `Open` renders one segment ahead on
its own goroutine and the writer is forbidden to synthesise. Between segments of a report there is no
gap.

**The gap is the card build, and it is data.** Measured (perf-protocol §6): **1.03 s median cold** over
11 network requests (n=5), against an output buffer of ≈0.7 s — about one and a half times the
audible-silence threshold, before `tune`'s own requests and before the first render. Warm: **1–3 ms**.

**Concurrency does not remove it (PL-3, measured).** The same calls issued in parallel floor at
**0.96 s** with the same 11 requests: they funnel through a shared `/points` resolution that
serialises however they are issued. So standby is the remedy, not one of two options.

**A correction is on the record here.** The first version of this measurement reported **2.14 s**,
because the instrument built its client without setting `RatePerSec` and took httpx's default of 5/sec
where the app uses 30 (`app/app.go:123`). Eleven requests at 5/sec is ~2.2 s — the instrument was
timing its own misconfiguration, and the figure looked entirely plausible. Caught by the PLAN red team;
the conclusion survives, the number did not.

**Recommendation — build the next card at standby, do not pre-render its audio.** The warm figure is
what makes this a removal rather than a relocation: a built card costs nothing to rebuild, so the
1.03 s does not reappear elsewhere. Audio look-ahead is left exactly as it is, since it already works
and R6's writer-starvation rules are hard-won.

**Already required as DR-7.** The requirement was right; what was missing was the number showing it
is worth 1.03 seconds.

---

## PD-5 — The rename — **RESOLVED by the architecture**

Holding this until the approaches existed was correct: **the concurrency choice answers it.**

Under C, `app/director.go` **splits rather than moves**. Its *decision* half — who has the air, what
suspends what — becomes state inside `Step`, where it is data a test can assert. Its *effector* half —
`duck`, `pause`, `resume`, `discard`, `restore`, plus the new ticker cue — is what remains.

**Recommendation — not a rename.** `Director` names the coordinator that owns the Lineup;
**`mastercontrol`** names the surviving effector, whose job is putting one card on the air across both
outputs at once. That is precisely the HUM LEAD's condition — *a system directly responsible for
ensuring the radio output and the news ticker are properly synced for takeover events* — and it now
exists by name rather than by promise. Nothing is renamed for its own sake.

---

## PD-1 — Stub or build station identity, service area, ON AIR / STANDBY?

**Recommendation: build none of the three as named Broadcaster concepts. Build the one thing Observer
genuinely needs, and name it for Observer's job.**

| Concept | Recommendation | Reasoning |
|---|---|---|
| **Station identity** (callsign) | **Nothing.** | No Observer path asks what station this is; the broadcast already says "Watchpost Radio". A callsign field nobody reads is dead code. |
| **Service area** | **Already exists — and DR-13 is building the rest.** | Observer has an origin (the default location, R-4) and a radius (`config.TickerRadiusMi`). DR-13's significance fence is in scope and is the substantive half. What Broadcaster adds later is *deriving a watchlist from the geometry*, which is a **producer** change, not a Director change. |
| **ON AIR / STANDBY** | **Build a `running` state, because Observer needs one on its own merits.** | The main track must not advance while the listener has stopped the radio. Today that is `d.mode != ""` plus an epoch counter — a rule carried in two places, which is the shape that produced the duck-lift bug (RD-2). Under C it is one field the `Step` reads. |

**Why this is a seam and not a stub.** ON AIR / STANDBY is *a second reason to be off*. Once the
Director holds a `running` state for its own reason, Broadcaster adds an enum value, not a mechanism.
The seam is preserved by the state existing and being honestly named, which is the opposite of
stubbing.

---

## PD-3 — An interruption that outlasts the report's validity

**First, a finding that changes the shape of the answer: Observer cannot reach this case.**

The alert rail's drain in Observer is bounded by Max (default 5) plus any emergency overrun. At the
project's own ≈8 s-per-read assumption, five reads is ~40 s, and even twelve emergency orders is
under two minutes. **A location report does not go stale in two minutes.** The case becomes reachable
in Broadcaster, where an operator can hold the rail, and at emergency counts Observer will not
produce.

**Recommendation: implement the check, not elaborate handling.** A card carries `builtAt`. On resume,
if `now − builtAt` exceeds a staleness window, the remainder is discarded and a transition card says
so — reusing the card kind T-3 already requires, so it costs one comparison and no new concept.

**Why implement it at all if Observer cannot reach it:** the alternative is that Broadcaster inherits
a latent stale-read on the safety path, discovered by a listener. And under C it is **cheap to test
without waiting** — feed a `Tick` far in the future and assert the effect list carries the discard and
the transition. The cost of the mechanism is small; the cost of finding it in the field is not.

**The staleness window is 15 MINUTES. RATIFIED, HUM LEAD 2026-09-01.** It sits far beyond any
Observer-reachable drain, so it never fires there, while bounding the Broadcaster case. NWS
observations update hourly, so 15 minutes is conservative against the data's own cadence.

---

## PD-4 — How the Operator helps the Composer compose

**Recommendation: no Director seam at all, and out of scope for 0.14.0.**

The example that raised it — *"I now want all location reports to end with Fire, not Seismic"* — is
intra-report segment order, which S-7 places squarely with the Composer. The Director names *which*
report; it does not reach inside one.

**And it needs no pass-through field.** A card already names its slot type and subject, and the
Composer reads the listener's preferences from config the way it reads everything else it needs. A
`Profile` field on the card relayed through the Director would be dead code today **and** would
wrongly imply the Director owns composition — encoding the opposite of the boundary S-7 drew.

**The rule to carry forward: the Composer reads config; the Director does not relay it.**

Recorded as a Composer-side follow-up for a later release, with its own scope.

---

## PD-6 — A Broadcaster service-radius cap (~150 mi)

**Recommendation: defer to Broadcaster, and record it.**

Nothing in the Director depends on the number. DR-13's fence takes whatever radius the configuration
holds, and bounding what a human may *set* is a product decision about FRS / GMRS / ham range — which
belongs with the edition that has a transmitter, not with Observer, whose whole character is that it
has **no** geographic limit (S-1).

**Not to be confused with DR-13**, which is in scope: the significance fence gives a disaster its own
reach and is a different rule from capping the user-settable radius.

---

## Summary

| # | Disposition |
|---|---|
| PD-1 | Build a `running` state for Observer's own reason; nothing for callsign; service area already exists and DR-13 completes it |
| PD-2 | **Resolved by measurement** — pre-build the card (1.03 s), leave audio look-ahead alone |
| PD-3 | Implement the staleness check; window **15 min, ratified** |
| PD-4 | No Director seam; Composer reads config directly; out of scope |
| PD-5 | **Resolved by the architecture** — a split, not a rename; `mastercontrol` is the surviving effector |
| PD-6 | Defer to Broadcaster |
