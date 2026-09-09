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
- **FR-1.6** **The chord is not the only door.**  Switching surfaces is reachable without a modifier
  chord, because a terminal, multiplexer or assistive tool can intercept one and leave the operator with
  no path back.  *Exit: a surface switch is reachable by a non-chord route, driven through the real key
  path.*  *(Red team, accessibility lens: a chord-only switch has no fallback.)*

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
- **FR-2.5** **No report is ever read twice by two audio owners.**  The rotation drives the engine
  directly while the rail goes through the narrator arbiter, and nothing dedupes between them today.
  *Exit: a test drives both paths at the same subject and asserts exactly one read; a mutant removing the
  guard is CAUGHT.*  *(Red team, safety lens: the double-speak gap had no requirement.)*
- **FR-2.6** **All external text in the new lanes routes through the existing plaintext clamp** — the
  boundary that strips escape sequences and control characters from provider prose and relay titles.
  **One owner, no second path.**  *Exit: a hostile relay title and a hostile alert body render clamped
  in the bed row and the takeover card.*  *(Red team, infosec lens: the reuse was assumed, not stated.)*

## FR-3 — Card management

| | |
|---|---|
| **Source** | R-3.3, R-3.4, D-11 |

- **FR-3.1** Selecting a slot opens a **modal for that card** showing its detail.  *Exit: the modal
  opens for every one of the ten slots and the takeover handles, driven through the real key path, and
  its content is the card the handle names.*
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
- **FR-3.7** **A mis-action is recoverable — BOTH a confirm AND an undo (D-26).**  Dropping a card is
  the destructive one, and an operator under severe-weather pressure will eventually drop the wrong
  thing.  **The confirm guards the accidental keypress; the undo recovers the considered-but-wrong
  decision.  Two different failures, two different remedies.**  **The undo carries NO TIMER** — an
  expiring undo is a hidden clock, and a hidden clock under pressure is a trap.  The confirm must not
  delay a takeover.
  *Exit: a dropped card can be restored, or its drop required a confirmation; asserted for an ACTIVE
  warning specifically, which is the case that matters.*  *(Red team, DISCOVER phase lens: promote and
  demote got full rigor and destructive drop had no safeguard at all.)*

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
- **FR-5.3** **The station state is legible without reading**, via a background treatment in the manner
  of the ticker (D-21).  *Exit: the state is distinguishable at a glance; the chosen pair is measured by
  the contrast register like every other painted pair, and the `--ascii` and no-colour paths still carry
  the state in words, because a colour alone is not a carrier.*
- **FR-5.4** **The operator changes the state with a named control**, and the control is a requirement
  rather than a mock detail.  *Exit: the control moves the Director's power and the console reflects it;
  the binding is in the key map like every other action.*  *(Red team, business lens: promote and demote
  had FRs and the state toggle had none.)*
- **FR-5.5** **THE BOUNDARY OF "ON AIR" IS STATED, NOT IMPLIED.**  Watchpost has **no radio path** — it
  produces audio, and a separate transmitter the application cannot observe puts it over the air.  So
  "on the air" can only ever mean *the software is putting programme out*, never *the antenna is
  radiating*.  **The console must say so where the operator reads it**, because the locked problem
  statement's third inability is about confirming you are on the air, and an operator who believes the
  banner is confirming RF has been misled by us.  *Exit: the boundary is stated in the console's own help
  or About text, not only in a design document.*  *(Red team, three lenses converged here: business,
  safety and the phase lens each reached it independently.)*
- **FR-5.6** **A paused main track is a distinct condition from STANDBY.**  `OffAir` holds the rail;
  a paused main track does not.  *Exit: the two are separately representable and separately asserted.*

## FR-6 — Settings

| | |
|---|---|
| **Source** | R-2, D-6, **D-19** |

- **FR-6.1** Broadcaster has its **own settings modal**.  *Exit: it opens, it edits only the fields
  ruled Broadcaster's or shared, and it never presents an Observer-only field.*
- **FR-6.2** The split is **exactly as ruled** in the field table: shared theme, units, clock, update
  check, provider keys, cast and voices; Observer's watchlist, recents, fire and seismic thresholds;
  split tone mode and mute list; Broadcaster's own tower, relay and gain.  *Exit: a round-trip test
  asserts, per field, that a shared value survives a swap and a split value does not leak.*
- **FR-6.3** The migration is **purely additive**.  A 0.15.0 config decodes unchanged and the user
  reconfigures nothing.  *Exit: a captured 0.15.0 file loads, re-saves, and diffs to only the new
  table.*
- **FR-6.4** `broadcaster.tower` is a **single table, never an array of tables**, because the unknown-key
  preservation refuses to follow those.  *Exit: an old-binary save preserves it.*
- **FR-6.5** **WITHDRAWN by D-20.**  There is no rename and no unification: Observer's alert radius
  keeps its meaning and its name, and Broadcaster's service radius is a separate setting.  Recorded
  rather than deleted, because the withdrawn version was written into the record.
- **FR-6.6** Key-binding overrides gain **per-surface scoping** before their sharing question is
  answered.  *Exit: the same key can mean different actions on the two surfaces, and a duplicate within
  one surface is still refused.*

## FR-7 — Layout and the minimum size

| | |
|---|---|
| **Source** | R-5, D-4, **D-13** |

- **FR-7.1** Broadcaster uses the **platform breakpoint vocabulary**, whose boundaries are redefined
  for an operator console.  *Exit: the console's layout is SELECTED by the platform breakpoint function —
  proven by a mutant that changes a boundary and moves the rendered layout.  A call whose result is
  discarded does not satisfy this.*  *(Red team, code lens: the original exit was satisfiable by
  `_ = term.BreakpointFor(w)`.)*
- **FR-7.2** The console **renders correctly at every supported class**, decomposing as ruled — the
  visual scrollbar becomes a counter, titles truncate, and the side-by-side stacks below the class
  where the takeover panel would wrap.  *Exit: a size sweep that DERIVES its widths from the breakpoint
  boundaries, unioned with a stride sweep so an interior width cannot pass while broken.*
- **FR-7.3** **Below the minimum, the console shows a clear notice and does not render past the
  terminal.**  Silent overflow is a defect.  *Exit: a planted overflow is caught; the notice is asserted
  at a size below the floor.*
- **FR-7.4** The minimum height is far below the mock's 74 lines.  *Exit: the fixed chrome plus one
  readable card renders within the ruled floor.*

## FR-8 — The two boundaries, and freshness

| | |
|---|---|
| **Source** | R-8, **D-20** (which amends D-12), **D-16** |

**There are TWO radii and they bound different things.**  Conflating them deletes the stronger one.

- **FR-8.1** **Observer's alert radius is unchanged.**  It filters *arrivals* into an **unbounded**
  location set — the watchlist may hold anywhere.  *Exit: Observer's behaviour is identical; this is an
  NFR-1 claim, and the test states "nothing changes".*
- **FR-8.2** **Broadcaster's service radius is a HARD boundary on LOOKUPS.**  It bounds which locations
  may exist for the station at all, and therefore constrains alerts transitively.  **Nothing today
  bounds the location set**, so this is new work rather than a reuse.  *Exit: a location outside the
  radius cannot be added, and none appears in the rotation.*
- **FR-8.3** **A hyper-local station is a supported case, not an edge.**  Three miles is the HUM LEAD's
  own example.  *Exit: the console is usable, and the schedule non-empty, at a 3-mile radius.*
- **FR-8.4** **A tiny radius must still receive county and zone products.**  The mechanism already
  exists: a zone-only alert with no point to measure is admitted by being *"one the app is already
  tracking at a watched location"*.  *Exit: a 3-mile station receives the county warning for the
  location it tracks — asserted, because this is what makes FR-8.3 viable.*
- **FR-8.5** Locations inside the service radius are fetched at the **priority cadence**.  *Exit: inside
  is fast, outside does not exist for the station at all.*
- **FR-8.6** A **hard cap** bounds the priority tier, filled **population-descending**, and **behaves
  when the radius admits very few locations — possibly one.**  *Exit: the cap is correct at N=1 and at
  N greater than the cap; the rule is stated where the operator reads it.*
- **FR-8.7** Cadences stay **bounded by each source's own refresh and the client's politeness limits**.
  *Exit: every cadence carries its argument beside it, as every existing one does.*
- **FR-8.8** The operator can see **effective freshness** — when each kind last arrived.  *Exit: the
  console shows a per-kind last-arrival time that moves when data arrives and visibly ages when it does
  not.*
- **FR-8.10** **The station behaves defined-ly when a feed fails while broadcasting.**  Visibility of
  freshness is not a behaviour.  The station must not read stale data as current, and must not go
  silently quiet.  *Exit: with a provider forced to fail, the console shows the degradation and the
  spoken output either says the data is unavailable or omits it — never presents it as current.*
  *(Red team, DISCOVER phase lens: the worst day is severe weather with a failing feed, and no
  requirement covered it.)*
- **FR-8.9** A **national-scope exemption** is investigated and either built or dropped with a recorded
  reason.  **The HUM LEAD's own expectation is that it manifests as local alerts anyway**, and FR-8.4's
  zone mechanism is why.  *Exit: DISCOVER states whether any product carries a national scope that
  would not also arrive locally.*

## FR-9 — The tower

| | |
|---|---|
| **Source** | R-7, D-10 |

- **FR-9.1** The broadcast location's **name and coordinates are one fact with one source of truth**.
  *Exit: a test that changes the name and finds the coordinates follow, or refuses the change.*
- **FR-9.2** The coordinate pair is a **first-class value**, not a string assembled for the masthead,
  because the planned map consumes it.  *Exit: the pair is a typed value with one owner; no consumer
  parses it back out of rendered text.*
- **FR-9.3** The **service radius and the relay selection both measure from that pair.**  *Exit: both
  compute from the same origin; changing the tower moves both.*
- **FR-9.4** **The tower's storage boundary is stated to the operator.**  The tower is a real person's
  antenna position at metre precision, persisted to disk.  The operator is told, once, what the
  application does and does not do with it — that it stays local, and whether it reaches debug dumps or
  exports.  *Exit: the statement exists where an operator configuring the tower will read it, and a
  debug dump is asserted not to contain the pair unless that is stated.*  *(Red team, infosec lens: the
  project already reasoned about exactly this class for a MOCK coordinate — F-66 — and never re-ran that
  reasoning for the real field.)*

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
  and seen to turn it red — recorded with its date, or a by-construction control.  *Exit: a roster in
  `07-readiness/gates.md` with an evidence line per gate, as 0.15.0 produced.*
- **FR-11.2** **Every subject list is derived** (INST-1): the size sweep walks the breakpoint
  boundaries; the settings round-trip walks the config struct; the lane assertions walk the track enum.
  No hand-written list of things-to-check.  *Exit: adding a breakpoint, a config field or a track adds
  sweep rows with no edit to the test.*
- **FR-11.3** **Silence is a distinct verdict** (INST-2): not-applicable, could-not-run and not-covered
  never render as passed.  *Exit: each instrument has a case per verdict, and a forced could-not-run
  reports as such rather than green.*
- **FR-11.4** **A surviving plant indicts the plant first** (INST-3).  *Exit: any SURVIVED verdict is
  recorded with the plant re-examined, as 0.15.0 recorded both its surviving plants.*
- **FR-11.5** **Every published count states its blind spot in its own output** (INST-5).  *Exit: the
  count's own output names what its method cannot see.*
- **FR-11.6** **An instrument answers a KNOWN case before it is believed about an unknown one**
  (**INST-4**).  *Exit: each new instrument is run against a case whose answer is already established,
  and that A/B is recorded beside its first published number.*  *(Red team, junior-dev lens: INST-4 was
  the one rule with no requirement — a builder discharging every other FR-11 item would have believed
  they had satisfied `FULL INST` while never A/B-ing an instrument at all.)*

---

# Glossary — the words this release borrows from earlier ones

**Written because a newcomer cannot act on these documents without it.**  Every term below is used
throughout the requirements and is defined only in a previous release's files.

| Term | What it means here | Defined in |
|---|---|---|
| **the main track** | The rotation: the queue of ordinary reads, shown as the ten-card stack | `platform/lineup/lineup.go:15` |
| **the (alert) rail** | The priority queue for severe-weather takeovers.  It **always drains first** | `platform/lineup/lineup.go:16,171-188` |
| **the bed** | The live NOAA relay stream the programme rides on.  Deliberately **not** a track — a resource the Director cuts over to, not a queue | `platform/lineup/lineup.go:9-12`, `bed.go:2-3` |
| **a card** | One scheduled read, with a state machine from proposed to on-air to done | `platform/lineup/card.go:233-307` |
| **a takeover** | The one card a severe-weather burst produces, queued to the rail | `platform/lineup/plan.go:188-200` |
| **the fence** | A **hard** boundary on which alerts reach the schedule at all.  What it excludes is *"not read, not counted and not pointed at"* | `platform/lineup/fence.go:95-97` |
| **the duck** | Dipping the relay's volume so a spoken alert is audible over it.  **One owner**, and lifting it is the defect a prior release removed | `app/mastercontrol.go:12`, `app/executors.go:113-123` |
| **dead air** | The station is on but broadcasting nothing.  Modelled as `Power.OffAir`; **it holds the rail too** | `platform/lineup/power.go:26-33` |
| **STANDBY** | **Overloaded, deliberately.**  A *card* on standby is READY TO AIR; a *station* on standby is OFF air.  The code says `OffAir` for the station to keep them apart; the operator-facing word stays STANDBY | `platform/lineup/power.go:36-42` |

*(Red team, junior-dev lens: the duck, the fence and the rail were used throughout and defined nowhere
in this feature's own folder.)*

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
- **NFR-7 — A silent station is bounded.**  STANDBY holds every track including the alert rail, which is
  correct and deliberate — but it means a station can sit silent with a severe-weather card held and
  nothing tells anyone.  **There is a bound**: either a duration ceiling, or an escalating operator-facing
  warning that grows while a rail card is held in standby.  *Exit: with a held rail card, the console
  escalates over time rather than sitting unchanged.*  *(Red team, safety lens: held is better than
  dropped, and neither is broadcast.)*

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
| **RS-9** | **A hyper-local radius starves the station.**  A 3-mile radius could admit one location and few alerts | MED | FR-8.4 — the zone mechanism already delivers county products to a tracked location.  **Asserted, not assumed**, because FR-8.3's viability rests on it |
| **RS-10** | **The location-set bound is new work with no precedent.**  Nothing today bounds which locations may exist; the fence bounds arrivals only | MED | FR-8.2 is scoped as new rather than as a reuse, so it is estimated honestly at PLAN |
