---
title: "0.16.0 DISCOVER — research wave 2, and a correction to wave 1"
date: 2026-09-09
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "Wave 2 — schedule producer, router sender audit, station state and audio bed.  Config field table pending."
---

# Wave 2 — and the correction that matters most

## 0. CORRECTION — the STANDBY state already exists, and I reported otherwise

**This is the most important item in wave 2 and it corrects my own reporting.**

At intake I told the HUM LEAD that ON AIR and STANDBY needed a primitive the codebase does not have,
and that D-1's standby-before-swap gate was the most expensive of the four audio items.  **The
station-state half of that is wrong.**

`platform/lineup/power.go` defines `Power` with three values — `Stopped`, `Running`, **`OffAir`** —
and `OffAir` is precisely the station STANDBY this release needs:

> *"OffAir is DEAD AIR: nothing is broadcast, including hazards, and the schedule HOLDS what it has
> not yet said (MVS-D-78).  IT IS NOT A LOUDER 'STOPPED', and the difference is the alert rail."*
> — `power.go:26-33`

It was built as the Broadcaster seam, deliberately and in the open:

> *"It is an ENUM RATHER THAN A BOOL, and that is the seam.  ON AIR / STANDBY is a second reason to
> be off, so a Broadcaster station adds a value here rather than a mechanism anywhere."*
> — `power.go:12-16`

**And it is enforced, not merely declared.**  `Director.advances(t Track)` (`power.go:96-119`) fails
closed on an out-of-registry power *before* the rail's exemption, then holds everything:

> *"Standby is unbypassable for the same reason: a corrupt power must not be a way around dead air."*

So the catastrophic case — **a station on standby putting a tornado warning to air** — is already
closed at the model level, including for the alert rail.

**What this changes:**

| Claim I made | Status |
|---|---|
| STANDBY needs a new primitive | **WRONG.**  `lineup.OffAir` exists, is invariant-checked and holds the rail |
| D-1's gate is the most expensive audio item | **WRONG on the state; right on the placement.**  The predicate is `Director.Power() == OffAir`, a discrete already-guarded field.  The cost is the router seam, not the state |
| ON AIR as a continuous carrier does not exist | **STILL TRUE.**  `Running` means "the Director is putting programme to air of its own accord", not a continuous carrier |
| There is no fade, drain or graceful stop | **STILL TRUE** |

**What is genuinely missing is the wiring, not the mechanism.**  `Director.Power()` is never called
from `app` or `modes`; `OffAir` appears there only in comments anticipating Broadcaster
(`executors.go:98,511`, `ticker.go:64`).  The state is modelled and unreached — the same shape as
`Publish` in wave 1 and `Origin.FromOperator` at intake.  **That is now four artefacts built for this
release and left deliberately unwired.**

---

## 1. There are TWO PATHS TO SPEECH, and the main track is a named, unbuilt task

### 1.1 The two paths

| | Alert path | Rotation path |
|---|---|---|
| Trigger | `Arrived` → `Plan` → `AlertRail` | `Programme` → dwell/`Ended` → `Tune` |
| Composition | `BuildCard` effect → `composeTakeover` | `startSynth` → `Composer.Compose` per cycle |
| Playback | `Speak` → the narrator arbiter `x.voice.Run` | `d.engine.StartSource` **directly** |
| Goes through `lineup` cards? | Yes | **No** |

Citations: `director.go:398-427`, `bed.go:123-261`, `app/executors.go:269,323,370-376`,
`app/radio.go:162,455-534`, `synth/source.go:314-315`.

### 1.2 The code says the main track is deferred, by name

`app/executors.go:277-278` and `:288-290` both refuse the ordinary slots with the same sentence:

> `return x.decline(v, v.ID, "read by the main track, which arrives with T3.2")`

**And T3.2 is a real, planned, HUM-LEAD-split task** from the 0.14.0 director build plan
(`multi-voice-support/03-architecture-design/director-build-plan.md:86-93`):

- **T3.2a** — wire the pump and Director into production at behavioural parity.  **Done.**
- **T3.2b** — *"The main track absorbs `armDwell` / `advanceQueue` / `stopDwell` (`Tick` → `Tune`).
  The duck keeps exactly one owner across the move."*  **Not done.**

The split's own rationale is worth carrying forward, because it applies again now:

> *"Splitting isolates the risky step behind a claim a test can state: **nothing changes.**"*

**Two neighbouring rows in the same plan are also Broadcaster's:**

- **T3.8** — `Speak` carries the script's PARTS, *"and the Broadcaster displays the same parts as
  distinct lines"* — the console's card body, anticipated.
- **T3.9** — *"STOP is not STANDBY"*, whose rationale is the §0 hazard verbatim: *"Under one flag a
  Broadcaster on STANDBY would put a tornado warning to air."*  **This is what `power.go` is.**

**So 0.16.0 is not opening a new design.  It is completing a plan that named it.**

### 1.3 The obstacle, and it is a real one

`cutTo` — the seam the rotation tunes through — **must not lift the duck**, and the comment says why
(`executors.go:113-123`):

> *"Every tune the Director asks for is AUTOMATIC — the dwell elapsed, a cycle ended — and nobody
> pressed anything.  Lifting the dip here would bring the next location's report in at full volume
> over a breaking alert still reading, which is the defect … T2.3 removed by giving the duck one
> owner."*

A main-track producer that spoke through the narrator arbiter would **renegotiate that ownership
boundary**, not merely add cards.  That is the deepest technical question in the release.

### 1.4 Double-speak has no guard

`mark`/`readAloud` (`executors.go:69-100`) dedupes **alerts by id, for the rail only**.  Nothing
consults it from `runTune`/`startSynth`.  **If a new main-track producer queued cards while the
rotation kept driving the engine, nothing in the tree stops the same report being read twice by two
different audio owners.**

### 1.5 The recommendation, and the counter-argument that may beat it

Research recommends **a separate producer** rather than repurposing the rotation, because the
rotation is a continuous, pull-based loop owning its own engine, hand-off lines and per-segment voice
resolution — none of which fits the discrete `Propose → Admit → BuildCard → Speak` lifecycle.

**The counter-argument is strong and should be recorded as the live alternative:** a second producer
needs its own "whose turn is it" state beside `d.bed.ref`, and two rotations that can disagree about
position is exactly the drift `bed.go`'s single Director state was built to remove.  **T3.2b's own
wording favours the counter-argument** — it says the main track *absorbs* the dwell, not that a second
producer is built beside it.

**This wants a HUM LEAD ruling at PLAN, with T3.2b's text on the table.**

---

## 2. The router's true cost: about nine wiring points, all in one package

### 2.1 The sender inventory

Every holder of a program handle or a send function, verified confined to `app/`:

| Holder | Wired at | Sends |
|---|---|---|
| `radioDeck.p` | `app/dashboard.go:376` | `RadioStatusMsg`, `RelaySilentMsg`, `VoiceNoteMsg` |
| priority pipeline closure | `pipelines.go:170,178` | `SnapshotMsg` |
| recent pipeline closure | `pipelines.go:299,316` | `RecentSnapshotMsg` |
| `severeDeck.send` | `dashboard.go:177` | `SevereMsg` |
| `eventReader.send` | `dashboard.go:197` | `SevereReadingMsg` |
| `mastercontrol.send` | `dashboard.go:192,305` **and** `ticker.go:152` | `TickerBreakingMsg`, `TickerBreakingDoneMsg` |
| `tickerDeck.send` | `ticker.go:145,157` | `TickerAdvanceMsg`, `TickerMsg` |

**Nothing outside `app/` holds one** — verified by grep across `modes`, `platform`, `domains` and
`cmd`.

### 2.2 The estimate improves

**Every sender receives its send function at CONSTRUCTION time**, one level removed from
`tea.NewProgram`; none calls `tea.NewProgram` itself or holds program lifecycle logic.  So the change
is a **wiring change, not a message-shape change**: each constructor takes the router's send instead
of the program's.

**Roughly nine call sites, in four files.**  No message type changes, no `modes/tty` dispatch changes,
no domain changes.

### 2.3 The two hazards that are real

1. **The breaking pair.**  `TickerBreakingMsg` and `TickerBreakingDoneMsg` must arrive as a matched
   pair; `handleTicker` clears the marquee on `Done` with no guard
   (`modes/tty/dashboard.go:551-556`).  **A router that dropped or delayed one half leaves the
   marquee stuck showing a breaking item with no way to clear it** — and it is ordering and
   completeness, not mere delivery, that is load-bearing.
2. **The severe re-find.**  `applySevere` (`modes/tty/severe.go:119-134`) re-finds a row by key
   against index state built from the *previous* publish.  A buffered or dropped `SevereMsg` lands
   against a stale index.

**Neither is created by the router.**  Both are pre-existing assumptions that a router's
drop-or-hold policy would expose.

### 2.4 No message carries a routing tag

None of these messages is self-describing — `RadioStatusMsg`, `SevereMsg` and `TickerBreakingMsg`
carry payload only.  A router must therefore switch on type by name.  **Retrofitting a surface tag
onto every message struct is strictly more surface area than one switch in the router**, so the
switch is the recommendation.

---

## 3. The station, the bed and the gain

### 3.1 Service radius needs a hard cutoff, and `Nearest` cannot provide it

`Resolver.order` (`resolve.go:52-74`) adds the SAME-covering transmitter first, then walks
`Nearest(lat, lon, table.Len())` appending relayed stations until it hits `directoryCandidateCap = 3`.
**The cap is a COUNT, not a DISTANCE** — a transmitter two thousand miles away can fill a slot when
the nearer three are not relayed today.

`Table.Nearest(lat, lon, n)` **does not suffice**: it takes a candidate count and returns the `n`
closest however far they are (`table.go:122-149`).  A radius must be a hard filter layered on top.

**The repo already has the right shape twice**, and Broadcaster should follow it rather than invent:
`coops.nearestN` filters on `geo.HaversineKM(...) <= radiusKM` (`domains/marine/coops/coops.go:284-289`),
and `seismic.Rules.Keep` on `distanceKm <= r.RadiusMiFor(mag)*mileKm` (`domains/seismic/rules.go:160-162`).

### 3.2 The bed row, decoded field by field

| Field in the mock | Today |
|---|---|
| `KIG78`, `Coachella`, `CA`, `162.400 MHz` | **Available now** from `Transmitter` (`table.go:28-37`) |
| `41mi` | **Value exists as kilometres.**  `geo.HaversineKM` returns km; the mock shows miles.  The conversion pattern exists (`render/units.go:190`, `synth/fire.go:233,249`) but is not applied to a station |
| `from TOWER GPS` | **New.**  All distance maths today is from the listener's location, never a separate fixed operator position.  **This is R-7's requirement seen from the audio side** |
| `[ ← ] [ → ]` | **New.**  `Resolver` returns an ordered list; there is no cursor or selection state |
| `○ STANDBY` on the bed row | **New.**  No per-stream marker exists |

### 3.3 GAIN is Volume renamed — and it should NOT be shared

`Engine.Volume(pct int)` is the only level control (`player/engine.go:223-231`); `radioDeck.SetVolume`
is a pass-through (`app/radio.go:578`).  There is no second gain stage.  It is **not persisted** —
constructed at a hardcoded 55 (`engine.go:151`) and never loaded from config.

**Recommendation: a station-scoped, persisted gain, separate from Observer's listening volume.**
Broadcaster's gain sets the level of a signal going over the air; Observer's volume is a personal
listening preference.  Sharing one control means **an operator's on-air level changes silently
because someone adjusted the listening volume** — the "two carriers of one rule" shape
`mastercontrol.go:14-18` exists to prevent.  This is a D-6 field-table entry.

### 3.4 The mock's banner is ambiguous, and the CODE predicted exactly this ambiguity

The banner reads `{ *** ON AIR | BROADCASTING *** : STANDBY (DEAD AIR) }` and the control reads
`[ SHIFT + ENTER ] GO TO STANDBY`.

**Preferred reading (A):** the station is currently ON AIR, and the control names the state it would
move *into*.  Plain English supports it — an imperative "GO TO STANDBY" names a destination.

**But the ambiguity is genuine**, and `power.go:36-42` warns about this exact word:

> *"NAMED OffAir BECAUSE STANDBY IS ALREADY TAKEN, and by the opposite meaning.  A CARD on standby is
> READY TO AIR; a STATION on standby is OFF air. … The operator-facing word stays STANDBY; the code
> says which standby it means."*

**So the console has a vocabulary problem the model already solved for itself**, and the mock's bed
row `○ STANDBY` is a third use of the word on the same screen.  **This is a UX ruling for the HUM
LEAD**, and it is the kind the standing rule says must be decided from a rendering rather than a
description.

### 3.5 Cost ranking, revised by §0

| Rank | Item | Why |
|---|---|---|
| 1 | **STANDBY state** | **`lineup.OffAir` already exists.**  Needs exposure and wiring, not invention |
| 2 | Gain | `Engine.Volume` does the work; new is a persisted station-scoped field |
| 3 | **Bed selection** | **Most likely to be underestimated.**  Looks like "add a radius parameter"; actually needs a fixed tower position, a genuine distance filter, a unit conversion at a new call site, and cursor state — none of which exist |
| 4 | D-1 swap gate | The predicate is cheap; the **router placement** is the cost |

### 3.6 The trap in mocking ON AIR while D-1 makes STANDBY load-bearing

**A mocked ON AIR that only DISPLAYS a state can drift from the real audio path**, and D-1 gates a
mode swap on it.  That is the same shape as wave 1's `Lineup.Set` hazard: a surface asserting
something the machinery does not follow.

**The counter-argument, which §0 strengthens considerably:** the predicate reduces to
`Director.Power() == OffAir`, a discrete enum transition already guarded by `advances`' fail-closed
check — arguably no less trustworthy than any other gate in the app, with audio-truth verification
addable later without moving the gate.  **On balance the counter-argument wins**, provided the gate
reads the Director's power rather than a UI flag.  **That belongs in the requirement.**

---

## 4. Cross-cutting synthesis — wave 2

1. **Composition — four unwired artefacts now, not three.**  `Publish` (display), `Origin.FromOperator`
   (edit provenance), `term.Breakpoint` (responsive), and now `lineup.Power` (station state).  Every
   one was built for this release and left deliberately unreached.  **The foundation claim is
   substantially honest, and "exists" still never means "works" — none has had a first consumer.**
2. **Convergence — the 0.14.0 plan already contains this release.**  T3.2b is the main track, T3.8 is
   the card body, T3.9 is the STANDBY split.  0.16.0 is **completing a named plan**, not opening a
   design.  Wave 3 should read the rest of that plan before enumerating FRs, because more of it is
   probably ours.
3. **Contradiction resolved — my own intake claim.**  §0 corrects it in the open.  The lesson is the
   one this release opened with: a capability claim decays invisibly, in both directions.  Intake
   under-claimed here as badly as the old notes over-claimed elsewhere.
4. **Risk update — M3 is CHEAPER than assessed, and still the top risk.**  The state exists and is
   unbypassable; the exposure is the router seam and the mocked ON AIR's possible drift.
5. **Risk update — a NEW top technical risk: two audio owners.**  §1.4's double-speak has no guard,
   and §1.3's duck boundary says the two paths were deliberately kept apart.  **Merging them is the
   riskiest work in the release**, and T3.10's precedent applies: it gets its own red team and its own
   UAT, shared with nothing.
6. **Risk update — D-2 is CHEAPER than assessed.**  About nine wiring points in four files, no message
   changes.  The residual risk is the breaking-pair ordering, not the wiring.
7. **Open question — the word STANDBY is used three ways on one screen.**  Station standby, card
   standby, and the bed row's marker.  `power.go` solved it in code by renaming; the console cannot.
   **A HUM LEAD UX ruling, from a rendering.**
8. **Implication for wave 3.**  Read the remainder of the 0.14.0 director build plan for rows that are
   silently 0.16.0's; settle the main-track producer question with T3.2b's text on the table; and take
   the two outstanding rulings — the breakpoint vocabulary and the credits obligation — which still
   gate FR enumeration.
