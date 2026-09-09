---
title: "0.16.0 DISCOVER — open questions awaiting HUM LEAD ruling"
date: 2026-09-09
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "OPEN — 8 items.  OQ-12 carries a descope lever that changes the release's shape."
---

# Open questions — restated as direct asks

Ordered by how much each changes the work.  Items 1-5 block FR enumeration; 6-8 can be ruled later in
DISCOVER or at PLAN.  Every recommendation names its strongest counter-argument.

## 1. OQ-12 — Does the main track ABSORB the rotation, or run beside it?  (biggest)

**ASK: absorb, build beside, or descope ordinary reads from 0.16.0 entirely?**

There are two paths to speech.  The alert rail goes through the narrator arbiter; the ordinary
rotation drives the audio engine directly and never touches a card.  The tune seam carries
*"IT MUST NOT LIFT THE DUCK"* because every tune is automatic, and nothing dedupes between the paths —
so if both ran, the same report could be read twice by two different audio owners.

| | Option | Consequence |
|---|---|---|
| **A** | **Absorb**, per T3.2b — *"the main track absorbs `armDwell` / `advanceQueue` / `stopDwell`"* | The riskiest work in the release.  Renegotiates the duck's ownership boundary |
| **B** | **A second producer beside the rotation** | Two rotations can disagree about position — the drift `bed.go`'s single state was built to remove |
| **C** | **DESCOPE ordinary reads.**  Ship the console displaying the schedule and controlling TAKEOVERS only; the rotation keeps its own path untouched | Much safer release.  The ten-slot stack shows what exists rather than everything |

**RECOMMENDATION: C for 0.16.0, A for 0.17.0.**  The `Publish` seam already lets the console *display*
the whole schedule today, and takeover control alone is a genuinely useful operator console.  Option A
is a release's worth of risk on the SAFETY path and deserves its own cycle with its own red team, the
way T3.10 got one.  **Counter-argument:** a console that cannot reorder ordinary reads may not satisfy
the locked problem statement's *"cannot change its order before it goes out"* — which is exactly why
this is the HUM LEAD's call and not mine.

## 2. OQ-11 — One Director, one Fence, two surfaces.  Which radius admits alerts?

**ASK: which radius does the Director fence by?**

`lineup.Fence` documents itself as *"the listener's service radius: a HARD boundary on what reaches
the lineup at all … What it keeps out is not read, not counted and not pointed at"*, fed from
`config.TickerRadiusMi`.  Broadcaster's service radius is the same mechanism, not a new one.

| | Option | Consequence |
|---|---|---|
| **A** | The **active surface's** radius | Switching surfaces changes what is admitted; already-admitted cards stay, so the lineup becomes a mix of two policies |
| **B** | **Broadcaster's always wins** when configured | Observer's personal preference silently stops applying |
| **C** | **The WIDER of the two** | More reaches the lineup; each surface filters for display |
| **D** | Two Directors | Contradicts the one-owner architecture |

**RECOMMENDATION: C.**  The fence is a *hard* boundary — what it excludes is never read, counted or
pointed at — so under-fencing loses safety information irrecoverably, while over-fencing costs only
display filtering.  **Counter-argument:** it makes Observer quietly do more work than the listener
asked for, and R-8's cadence makes that work non-trivial.

## 3. OQ-9 — Which responsive breakpoint vocabulary does Broadcaster use?  (F-68)

**ASK: platform, Observer's, or a third?**

`platform/term` defines `Breakpoint`, `BreakpointFor`, `HeightCompact` and `BreakTooNarrow` — and
**nothing calls any of it.**  Observer reflows through its own `radioBP` scale at 84 and 146, which
matches the platform boundaries nowhere.  Broadcaster would be the third implementation.

| | Option |
|---|---|
| **A** | Adopt `platform/term` and **redefine its boundaries** for an operator console |
| **B** | Adopt Observer's `radioBP` scale |
| **C** | Broadcaster defines its own |

**RECOMMENDATION: A.**  It is unused, so its boundaries are free to change today and expensive to
migrate later, and it turns a dead vocabulary into a real one.  **Counter-argument:** an operator
console and a radio panel may genuinely want different boundaries, and forcing one enum to serve both
is how a shared abstraction becomes a bad fit for everyone.

## 4. OQ-10 — Is the on-air credits read a licence obligation?

**ASK: obligation, courtesy, or out of scope?**

`app/credits.go:16-20` says the list *"is a licence obligation, not a courtesy"* because GeoNames and
Open-Meteo are CC BY 4.0.  The mock draws a `WATCHPOST CREDITS READ` card, and the `Slot` enum's four
members do not include one that fits it.

| | Option | Consequence |
|---|---|---|
| **A** | **Obligation** | Becomes a required card with its own `Slot` **and a cadence rule** — how often it must air is then a second ruling |
| **B** | **Courtesy** | Optional card, possibly a `Transition` variant, droppable under time pressure |
| **C** | **Out of scope for 0.16.0** | The mock's card is not built; the obligation question is deferred |

**RECOMMENDATION: A, treated as an obligation by default**, because that is the safer error and the
code already says so for the on-screen list.  **I am not qualified to rule the legal question**, and
whether on-air attribution is required for CC BY 4.0 derived data over FRS/GMRS is genuinely yours.

## 5. R-8 — the cadence requirement I failed to write at intake

**ASK: approve R-8 as drafted, amend, or reject?**

Your intent said *"higher data request frequency"* and the brief contains no requirement for it.
Investigating it found the literal reading fails: alerts already fetch every **20 seconds**, and the
ticker's own comment says the feeds' TTLs ride the cache so a fast tick only hits the network for
sources due.  You cannot fetch fresher than a source refreshes, and on a 429 you get *less*.

**What the intent actually buys:** freshness is decided by which TIER SET a location is in, not by a
global rate.  A bounded service area makes the fast tier affordable for everything inside it.  **The
two halves of your parenthetical are one mechanism.**

R-8 as drafted, in four parts: locations inside the radius get the priority cadence; cadences stay
bounded by each source's refresh and the politeness limits; the cadence set keeps one owner; and the
operator can see effective freshness.  **It deliberately names no numbers** — every existing cadence
carries its argument in a comment beside it, and a new number without one would be the first.

## 6. OQ-8 — What bounds how many locations the radius admits to the fast tier?

**ASK: pick a bounding rule.**

A radius bounds an **area**, not a **count**.  A hundred-mile radius over a dense region admits far
more locations than a sparse one, and the client holds roughly five requests per second before it
throttles — which would degrade the station precisely where it is busiest.

| | Option |
|---|---|
| **A** | A hard cap on priority locations, number chosen at PLAN against the request budget |
| **B** | A cap derived from the budget arithmetic rather than picked |
| **C** | Admit **transmitter coverage** rather than arbitrary points |
| **D** | No cap; accept throttling |

**RECOMMENDATION: A.**  Simplest to reason about, simplest to gate, and a number with an argument
beside it matches how every other cadence in the file is justified.

## 7. OQ-13 — The word STANDBY appears three ways on one screen

**ASK: authorize me to render variants, then rule from the rendering.**

`power.go` renamed the code concept to `OffAir` precisely because *"A CARD on standby is READY TO AIR;
a STATION on standby is OFF air … The operator-facing word stays STANDBY; the code says which standby
it means."*  The console then uses STANDBY for the **station banner**, the **bed row marker**, and the
card state machine has its own **card standby** underneath.

**RECOMMENDATION: the station keeps STANDBY; the bed row gets a different word.**  But the standing
rule is that a rendered change is ratified from an actual rendering, never a description — so the
specific ask is: **authorize me to render two or three variants at 150 columns for you to choose
from.**

## 8. The four remaining settings fields (D-6 said rule these at DISCOVER)

**ASK: rule each SHARED, OBSERVER, BROADCASTER, or SPLIT.**

| Field | The tension | Recommendation |
|---|---|---|
| `Radio.Cast`, `Radio.Voices.*` (9 roles), `Voice` | Shared sends a personal voice identity over the air; split duplicates 18 fields for an unproven benefit | **SHARED for 0.16.0**, because R-4 scopes Broadcaster's audio to mock.  Revisit when it speaks |
| `Radio.Tones.Mode`, `Radio.Tones.Muted` | **Asymmetric stakes.**  Shared means a personal comfort mute silences an alert tone the station is meant to transmit | **SPLIT.**  A private preference must not censor what goes over the air |
| `Keys[action]` | Neither shared nor split is right — it needs the per-surface namespace `Merge` does not have | **Per-surface scoping**, which is work, not just a ruling.  Pairs with OQ-9's F-68 decision |
| `Providers[name].Key` | Looks obviously shared, but R-8's cadence can burn a rate-limited tier Observer was sized for | **SHARED**, with the rate budget recorded as a risk and OQ-8's cap as its mitigation |
