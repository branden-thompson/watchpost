---
title: "0.16.0 — Functional and Non-Functional Requirements"
date: 2026-09-09
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "DRAFT for DISCOVER exit — derived from R-1..R-8 and rulings D-1..D-19.  Every FR carries a source and an exit condition."
---

# Requirements

**Every FR names the brief requirement or ruling it descends from, and an EXIT CONDITION — the
observable that says it is met.**  A requirement without an exit condition is a wish, and this release
runs under `FULL INST`, where the instrument that checks a thing is part of the thing.

---

## FR-1 — Two surfaces and the swap

| | |
|---|---|
| **Source** | R-1, D-1, D-2, D-3 |

- **FR-1.1** A thin **router** is the Bubble Tea model handed to the program.  `Dashboard` stays the
  Observer-only model.  *Exit: the program's model is the router; `Dashboard` has no surface field.*
- **FR-1.2** The router fans the **program-scoped messages** to the surfaces: window size, background
  colour, and key presses.  *Exit: a size change repaints both surfaces; a test drives a size change
  while Broadcaster is active and shows Observer correct on return.*
- **FR-1.3** All senders that today capture the program handle receive the **router's** send instead.
  *Exit: no production reference to `p.Send` outside the router's construction — a derived check, not
  a hand-written list.*
- **FR-1.4** Swapping to Observer is **refused unless Broadcaster is in STANDBY**, and the refusal is
  checked in the router's single swap branch.  *Exit: a planted second swap path is refused by the same
  gate; the check exists in exactly one place, proven by a single-owner AST check like the band's.*
- **FR-1.5** The swap chords are **rebindable**, and the defaults are chosen at PLAN against the
  multiplexer survey.  *Exit: an override in the user's key table changes the chord; the default is not
  a documented multiplexer prefix.*

## FR-2 — The three lanes

| | |
|---|---|
| **Source** | R-3, **D-11** |

- **FR-2.1** The console shows a **main track of ten cards** — the rotation — a **priority track** for
  takeovers, and the **bed**.  *Exit: a golden render at the ruled minimum size shows all three, with
  invariant assertions on lane identity, not only byte pins.*
- **FR-2.2** The priority track **always drains first**, including while the bed is playing.  *Exit:
  the existing rail-precedence property extended to the bed-playing case, with a mutant that reverses
  precedence and is caught.*
- **FR-2.3** The console reads its schedule from the **`Publish` effect**, not from a second source of
  truth.  *Exit: the console's model is populated only by a published lineup; no second read path
  exists.*
- **FR-2.4** The ten slots are addressable **`0`-`9`**, and the takeover layer has its own handles.
  *Exit: each handle opens the card it names, driven through the real key path.*

## FR-3 — Card management

| | |
|---|---|
| **Source** | R-3.3, R-3.4, D-11 |

- **FR-3.1** Selecting a slot opens a **modal for that card** showing its detail.
- **FR-3.2** The operator can **promote, demote and drop** a card.  *Exit: each action changes the
  order the schedule actually walks — asserted against `Next()`, never against a display field.*
- **FR-3.3** **An action must never be shown as taken unless the schedule took it.**  `Lineup.Set`
  refuses to reorder by design, so a promote routed through it would silently no-op.  *Exit: a mutant
  that makes promote update display state without reordering is CAUGHT.*  **This is the release's
  UI-integrity requirement and it exists because the failure is invisible.*
- **FR-3.4** An operator edit is **attributable**: the card records that a human placed it.
  `Origin.FromOperator` exists and is never constructed.  *Exit: an operator-placed card reports
  `FromOperator`.*
- **FR-3.5** Actions remain available **while the main track is paused** (FR-4.2).  *Exit: promote,
  demote and drop all succeed against a paused track.*
- **FR-3.6** A card the operator holds past the **fifteen-minute staleness bound** is dropped by the
  existing safety rule, and **the operator is told which card went**.  The listener is already told.
  *Exit: the operator-facing notice names the dropped card.*

## FR-4 — The bed and the cut-over

| | |
|---|---|
| **Source** | R-4.2, R-4.3, **D-11** |

- **FR-4.1** The operator **selects the bed's relay** from candidates bounded by the service radius.
  *Exit: a relay outside the radius is not offered — a hard distance filter, not a candidate count.*
- **FR-4.2** The operator can **cut the main track over to the bed**.  The main track then **pauses**:
  no card advances, and management actions stay live.  *Exit: a paused main track advances nothing over
  a dwell period while promote still reorders it.*
- **FR-4.3** **The cut-over is operator-initiated, and the duck's behaviour on it is decided
  explicitly.**  Today's tune seam must not lift the duck because *"nobody pressed anything"* — that
  premise no longer holds for a pressed cut-over.  *Exit: the chosen behaviour is stated in the code at
  the seam, with a test for the alert-still-reading case.*
- **FR-4.4** The bed row shows callsign, site, state, frequency, and **distance in miles** from the
  tower.  *Exit: the rendered distance matches a computed value; the km-to-mile conversion is the
  existing one, not a new one.*

## FR-5 — Station state

| | |
|---|---|
| **Source** | R-4.1, D-1 |

- **FR-5.1** The station is **ON AIR or STANDBY**, and STANDBY is `lineup.Power.OffAir` — the existing,
  invariant-checked state that holds every track including the rail.  *Exit: the console's state is
  read from the Director's power, never from a UI flag.*
- **FR-5.2** ON AIR is **mock for this release** — it does not assert a continuous carrier, and the
  release says so where a user reads it.  *Exit: no claim of continuous broadcast in the README, the
  help, or the About window.*
- **FR-5.3** **A paused main track is a distinct condition from STANDBY.**  `OffAir` holds the rail;
  a paused main track does not.  *Exit: the two are separately representable and separately asserted.*

## FR-6 — Settings

| | |
|---|---|
| **Source** | R-2, D-6, **D-19** |

- **FR-6.1** Broadcaster has its **own settings modal**.
- **FR-6.2** The split is **exactly as ruled** in the field table: shared theme, units, clock, update
  check, provider keys, cast and voices; Observer's watchlist, recents, fire and seismic thresholds;
  split tone mode and mute list; Broadcaster's own tower, relay and gain.  *Exit: a round-trip test
  asserts, per field, that a shared value survives a swap and a split value does not leak.*
- **FR-6.3** The migration is **purely additive**.  A 0.15.0 config decodes unchanged and the user
  reconfigures nothing.  *Exit: a captured 0.15.0 file loads, re-saves, and diffs to only the new
  table.*
- **FR-6.4** `broadcaster.tower` is a **single table, never an array of tables**, because the unknown-key
  preservation refuses to follow those.  *Exit: an old-binary save preserves it.*
- **FR-6.5** The service radius is **one value**, renamed from its ticker-era name with a compatibility
  shim.  *Exit: an old file's radius arrives under the new name with the same value.*
- **FR-6.6** Key-binding overrides gain **per-surface scoping** before their sharing question is
  answered.  *Exit: the same key can mean different actions on the two surfaces, and a duplicate within
  one surface is still refused.*

## FR-7 — Layout and the minimum size

| | |
|---|---|
| **Source** | R-5, D-4, **D-13** |

- **FR-7.1** Broadcaster uses the **platform breakpoint vocabulary**, whose boundaries are redefined
  for an operator console.  *Exit: the platform breakpoint API has production callers.*
- **FR-7.2** The console **renders correctly at every supported class**, decomposing as ruled — the
  visual scrollbar becomes a counter, titles truncate, and the side-by-side stacks below the class
  where the takeover panel would wrap.  *Exit: a size sweep that DERIVES its widths from the breakpoint
  boundaries, unioned with a stride sweep so an interior width cannot pass while broken.*
- **FR-7.3** **Below the minimum, the console shows a clear notice and does not render past the
  terminal.**  Silent overflow is a defect.  *Exit: a planted overflow is caught; the notice is asserted
  at a size below the floor.*
- **FR-7.4** The minimum height is far below the mock's 74 lines.  *Exit: the fixed chrome plus one
  readable card renders within the ruled floor.*

## FR-8 — Service radius and freshness

| | |
|---|---|
| **Source** | R-8, **D-12**, **D-16** |

- **FR-8.1** The **service radius is the boundary**, and it is the existing fence — a hard boundary on
  what reaches the schedule at all.  *Exit: the fence is fed from the renamed single value.*
- **FR-8.2** Locations inside the radius are fetched at the **priority cadence**.  *Exit: a location
  inside the radius is on the fast tier; one outside is not.*
- **FR-8.3** A **hard cap** bounds how many locations reach the priority tier, filled
  **population-descending**.  *Exit: with more candidates than the cap, the admitted set is the densest
  N, and the rule is stated where an operator reads it.*
- **FR-8.4** Cadences stay **bounded by each source's own refresh and the client's politeness limits**.
  No cadence is set faster than its source refreshes.  *Exit: every cadence carries its argument beside
  it, as every existing one does.*
- **FR-8.5** The operator can see **effective freshness** — when each kind last arrived.
- **FR-8.6** A **national-scope exemption** to the radius is investigated and either built or dropped
  with a recorded reason.  *Exit: DISCOVER states whether any product carries a national scope that
  would not also appear locally.*

## FR-9 — The tower

| | |
|---|---|
| **Source** | R-7, D-10 |

- **FR-9.1** The broadcast location's **name and coordinates are one fact with one source of truth**.
  *Exit: a test that changes the name and finds the coordinates follow, or refuses the change.*
- **FR-9.2** The coordinate pair is a **first-class value**, not a string assembled for the masthead,
  because the planned map consumes it.
- **FR-9.3** The **service radius and the relay selection both measure from that pair.**

## FR-10 — Gain

| | |
|---|---|
| **Source** | R-4.4, D-19 |

- **FR-10.1** Gain is **Broadcaster's own and persisted**, distinct from Observer's unpersisted
  listening volume.  *Exit: changing one does not move the other, across a restart.*

## FR-11 — The instruments (`FULL INST`)

| | |
|---|---|
| **Source** | the directive itself |

- **FR-11.1** **Every gate this release adds carries a watched failure** — a plant applied, compiled,
  and seen to turn it red — recorded with its date, or a by-construction control.
- **FR-11.2** **Every subject list is derived** (INST-1): the size sweep walks the breakpoint
  boundaries; the settings round-trip walks the config struct; the lane assertions walk the track enum.
  No hand-written list of things-to-check.
- **FR-11.3** **Silence is a distinct verdict** (INST-2): not-applicable, could-not-run and not-covered
  never render as passed.
- **FR-11.4** **A surviving plant indicts the plant first** (INST-3).
- **FR-11.5** **Every published count states its blind spot in its own output** (INST-5).

---

# Non-Functional Requirements

- **NFR-1 — Observer does not regress.**  The swap, the router and the settings split change nothing a
  listener notices in Observer.  *This is T3.2a's discipline, which worked: the claim a test can state
  is "nothing changes".*
- **NFR-2 — The upgrade is silent.**  A 0.15.0 user loses nothing and reconfigures nothing.
- **NFR-3 — The frame stays within budget.**  A second surface must not regress Observer's frame cost;
  the release measures against the recorded baseline rather than asserting.
- **NFR-4 — The branch pushes on its first commit** so CI runs on both platforms while the work is
  warm.  *Already satisfied: the branch has been green on macOS and Linux since intake.*
- **NFR-5 — Safety-critical rules hold.**  P10 exemptions are presented for ratification, never
  self-approved.
- **NFR-6 — Accessibility guidance is advisory, not a gate**, per the standing ruling — improve where
  cheap and clearly better; do not hold work.

---

# Risk register

| # | Risk | Sev | Mitigation |
|---|---|---|---|
| **RS-1** | **A promote that displays but does not reorder.**  `Lineup.Set` refuses reordering by design, so the naive implementation no-ops silently and the console lies about running order | **HIGH** | FR-3.3's mutant; assert against the schedule's own walk, never a display field |
| **RS-2** | **The operator cut-over lifts the duck** and brings a report in over an alert still reading | **HIGH** | FR-4.3 decides it explicitly at the seam; test the alert-still-reading case |
| **RS-3** | **A mocked ON AIR drifts from the real audio state** while the swap gate depends on it | MED | FR-5.1 reads the Director's power, not a UI flag |
| **RS-4** | **The breaking-alert message pair is split by the router**, leaving the marquee stuck | MED | The pair's ordering is a router test; flush before a swap |
| **RS-5** | **A dense service area multiplies fetch load** until the client throttles | MED | FR-8.3's cap, population-ordered |
| **RS-6** | **The settings split leaks or loses a value** on upgrade | MED | FR-6.3's captured-file round trip, per field |
| **RS-7** | **A sparse town inside the radius is excluded by the cap** and the operator cannot tell why | LOW | FR-8.3 states the rule where the operator reads it |
| **RS-8** | **Broadcaster becomes a third breakpoint implementation** | LOW | Closed by D-13 |
