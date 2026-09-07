# Build plan — Station Director & the Lineup

**PLAN, 2026-09-01 · SEV-0 · HUM LEAD · FULL TDD.** Architecture: **Approach C**
(`director-architecture.md`, **corrected by `role-model.md` — MVS-D-77**). Decisions: `plan-decisions.md`. Requirements: DR-1…DR-21.

> **THIS IS A PLAN. NOTHING BELOW IS BUILT YET.** Every T-number is work for BUILD, including the
> Phase 0 pins — *"we build in the build phase"* (HUM LEAD, 2026-09-01). Tests that enforce
> requirements are welcome at PLAN; finished bodies and fixtures are not. The only Go produced during
> PLAN was `app/cutover_latency_test.go`, the card-build instrument (perf-protocol §6), which
> **measures** the existing system rather than implementing any part of this design.

## The ordering principle

Phase 0 exists because of a measured fact, not caution. `armDwell` / `advanceQueue` carry rulings
from UAT 83, 93 and 97, and the live bug fixed there was that the exported `Tune` lifted the alert
duck while the unexported `tune` did not — *"a rule that lives in the case of an identifier is a rule
waiting to be missed"* (`app/radio.go:121-138`). A listener heard the next location come up at full
volume over a breaking alert still reading.

**That rule is about to move.** So the regression net is built and **proven able to fail** before
anything is touched. Nothing in Phase 1 begins until Phase 0's pins catch their own mutants.

---

## Phase 0 — The net, and proof it can catch

| # | Task | Test first |
|---|---|---|
| **T0.1** | Pin today's watchlist advance through the real entry points: a live relay advances after `liveDwell`; a synth cycle end advances; a user Stop advances nothing (`d.mode == ""`). | Written against **today's** code; must pass now and after the absorb. |
| **T0.2** | Pin the duck's single owner: an **automatic** advance does not lift the alert duck; a **user** Tune does not either (0.14.0 reversed this deliberately). | The UAT-2026-08-30 regression, expressed as a test rather than a comment. |
| **T0.3** | Pin the takeover's marquee/voice coupling: the callout is sent **before** the line is spoken, and the ordering holds per event. | This is today's accidental "3…2…1"; DR-18 turns it into a contract, so its current shape must be recorded first. |
| **T0.4** | **Mutation-validate T0.1–T0.3** with `06_docs/mutants/run.sh`. | Delete each pinned rule; each deletion must produce `CAUGHT`. A pin that survives its own mutant is not a pin. |

**Exit:** three pins green, and each demonstrated to fail when the rule it guards is removed. **A pin
that has never failed proves nothing about the absorb it is protecting.**

---

## Phase 1 — The pure core

No wiring, no goroutines, nothing observable by a listener. Every task here is a pure value or a pure
function, which is the point of Approach C: this whole phase is testable with no clock and no
concurrency.

| # | Task | Test first | DR |
|---|---|---|---|
| **T1.1** | `category.Spec` gains **`ReadRank`** — the third per-category ordering, beside tab order and `Rotation`. | The registry test asserting the three orderings are **independently declared**, so a later "harmonisation" fails loudly (RD-10). | DR-10 |
| **T1.2** | The `Lineup`, `Card` and card states as values: id, slot type, subject, role, max duration, text (empty until standby), state, origin, locked. | Card-model tests; a card at ON AIR refuses edits. | DR-3, DR-6 |
| **T1.3** | **The planner** — read order, the emergency budget rule, Max, the divert count. | A golden of the planned lineup; the determinism test (same inputs twice → identical); the DR-11 cases at Max 5: `1 emergency + 20 warnings` → the emergency **and four warnings**, divert 16; `7 emergencies + 18 warnings` → all seven, divert 18. | DR-9…DR-15 |
| **T1.4** | **The fence** — hard radius plus the significance exception for disasters. **Its own commit, before or after the structural work, never inside it** (RD-5). | The HUM LEAD's own cases: M7.5 at ~120 mi admitted, M9.0 in Fairbanks not. Fixtures asserted **valid first** (see the test strategy). | DR-13 |
| **T1.5** | `Step(Event) → (Director, []Effect)` and the event/effect vocabulary, **decomposed into per-event handlers from the first commit**. | Feed an event sequence, assert the effect list. No goroutines, no sleeps. | DR-1, DR-2, DR-20 |
| **T1.6** | The `running` state (PD-1) — the main track does not advance while the radio is stopped. | The rule expressed **once**, replacing `d.mode != ""` plus the epoch counter as two carriers of one rule. | PD-1 |

**A standing constraint on T1.5.** A naive `Step` is one enormous switch and breaches **P10-04**
(≤60 lines / ≤40 statements). Handlers are split from the first commit, not after the gate complains.

---

## Phase 2 — The pump and the executors

Wiring only. At the end of this phase the Director exists and runs, and **the listener hears exactly
what they heard before** — the executors are adapters over the code that already works.

| # | Task | Notes |
|---|---|---|
| **T2.1** | The pump: stateless, one goroutine. **Effects are DISPATCHED, never run inline** (PL-1): `for ev := range events { d, fx = d.Step(ev); dispatch(fx) }`, where `dispatch` hands each effect to a worker and returns immediately. | Not the tea update loop — audio scheduling must not ride the render path. **The earlier draft of this line ran `run(fx)` inline**, which would have paid the 1.03 s card build on the pump and reintroduced Approach B's blocking class inside the approach chosen to avoid it. **Test: a long-running effect does not delay the next event.** |
| **T2.2** | Effect executors as adapters: `BuildCard` → `segments()`; `Speak` → the existing render/play; `CueTicker` → the existing `tea.Msg`; `Tune` → `radioDeck.tune`. **The effect set is CLOSED and enumerated before this task starts** (PL-6) — it is the architecture's real interface, and "adapters over existing code" is otherwise unbounded work in a vague sentence. | Each returns its result as an **event**, never a blocking return. |
| **T2.4** | **Pump supervision** (DR-22): an effect that panics is contained, reported as a `Fault`, and the schedule continues. | Today `startTakeover` guards a panic for one surface; the pump is every surface. |
| **T2.3** | **`mastercontrol`** — the effector half surviving `app/director.go`'s split: `duck`, `pause`, `resume`, `discard`, `restore`, plus the ticker cue. | PD-5. The decision half moves into `Step`. **Assert the symbol set before and after** (RD-8: a range slice deleted three follow-ups this release, and a stray `//` silently commented out three struct fields). |

**Exit:** every existing golden, declset and PTY journey green, unchanged. If anything moves here, it
is a defect — the deliberate deltas belong to Phase 3 and 4.

---

## Phase 3 — The absorb

**Highest risk in the release (RD-1, RD-2, RD-3). One item at a time, each through the full
remediation review loop**: failing test first through the real entry point and *watched to fail* →
fix → mutate your own change → a **fresh adversarial reviewer who must run something** → LGTM before
the next.

| # | Task | The thing that must not break | DR |
|---|---|---|---|
| **T3.1** | The **alert rail** replaces `capBurst` / `startTakeover`'s selection. Delete `maxBreaking`, `breakingAllowance`, `breakingCap` and the per-lane floor. | **No admitted card is ever dropped unread** — bounds at admission only. This is the defect's removal; it gets its own test and its own mutant. | DR-3, DR-14 |
| **T3.2a** | **Wire the pump and Director into production, at behavioural PARITY.** They run, tick and supervise; nothing a listener notices changes. | **Observer behaviour is identical** — that is the whole claim, and it is what the pins assert. The pump's goroutine has an owner in the shutdown wait set (the 0.12.0 ship lesson). | DR-22 |
| **T3.2b** | The **main track** absorbs `armDwell` / `advanceQueue` / `stopDwell` (`Tick` → `Tune`). | **Phase 0's pins stay green.** The duck keeps exactly one owner across the move. | DR-3, PD-1 |

**T3.2 was SPLIT (HUM LEAD, 2026-09-03).** The row as written absorbs the dwell into a Director that
nothing drives — `newPump`, `newExecutors` and `lineup.New` have no production callers, so moving the
dwell onto it would delete a working five-minute Watchlist advance and replace it with something
nothing ticks. Splitting isolates the risky step behind a claim a test can state: **nothing changes.**
Every later Phase 3 task needs T3.2a regardless.
| **T3.3** | **Standby pre-build** (the 1.03 s finding) and the **15-minute staleness check**. | Discard is cheap and safe — a discarded standby card leaves no audio, no marks, no files (NFR-D-7). | DR-7, PD-3 |
| **T3.4** | The **fault channel**: one escalation, re-route by default, re-plan only when undeliverable; **modal only for a fault that stops the schedule**. The modal registers its colour tokens in `aaPairs`, is keyboard-dismissible, and takes focus without trapping it. | Today's relay→synth fallback raises **no** modal. Escalating a self-healing failure would be a noise regression. | DR-21 |
| **T3.5** | **The paired cue/release** — any transition out of ON AIR emits a release effect. | Today `TickerBreakingDoneMsg` has five early returns above it that send nothing, while the audio side releases unconditionally. Under the Lineup a discarded or superseded card makes that ordinary. | DR-24 |
| **T3.7** | **The Producer and the Composer, named.** A takeover card is produced as ONE card (`role-model.md`, MVS-D-77) and composed into a finished script — tone category, header, the alert lines, the divert tail. `lineup.Plan` becomes the Producer's selection step it already is in practice; `BurstHead` and `DivertNotice` retire from the slot registry into intra-card content. **Nothing is wired to the Director yet**: the claim this task can state is *the live takeover sounds identical*, which is the T3.2a discipline that worked. | The per-alert cards are already vestigial — `ticker.go:335-345` unwraps them straight back into events — so this names roles around code that exists rather than inventing behaviour. | MVS-D-77, S-7, DR-11 |
| **T3.8** | **The script-carrying `Speak`.** The effect set grows, deliberately and once (PL-6): `Speak` carries the script's PARTS, and the Reader paces them per MVS-D-72 — tone, 2 s, header, 1 s, lines, 1 s, 2 s, tail — with the render overlap that removes the dead gap before each line. | Inferring pauses by splitting text works until an alert contains a newline, and the Broadcaster displays the same parts as distinct lines. | MVS-D-72, MVS-D-77, DR-18 |
| **T3.9** | **STOP is not STANDBY.** Split the one flag into two states. Observer STOP keeps today's meaning — the programme stops, hazards still read. Broadcaster STANDBY is dead air: nothing is broadcast, alerts are still produced and shown, and PD-3's staleness applies to them while they wait. MasterControl becomes the clock keeper that declares which, and the Director complies. | Under one flag a Broadcaster on STANDBY would put a tornado warning to air. T3.3's staleness table passed because row 5 asked the Observer question. | MVS-D-77, PD-1, PD-3 |
| **T3.10** | **WIRE THE ARRIVALS** — the Producer hands its composed card to the Director, and `ticker.railBurst`'s inline composition retires. **The riskiest swap in the release**: it replaces the working takeover path, so it carries its own red team, and it must not share a UAT with anything else or a regression cannot be attributed. **Its UAT needs F-21(b)** — a takeover cannot be exercised on demand today, and validating this against whatever weather occurs is not validation. | AP-DEAD-01: shipping a subsystem no human can exercise is worse than a stub, because the only evidence for it is tests its own author wrote. | DR-3, MVS-D-77, AP-DEAD-01 |
| **T3.6** | **Runtime observability** — one `WATCHPOST_DEBUG_RADIO` line per card state transition, in the existing format so one log reads as one timeline. | RD-1: ~30 gate-passing defects on this code. A scheduler with no runtime signal is how the next one goes unnoticed. | DR-23 |

---

## Phase 4 — Settings and the spoken surface

| # | Task | DR |
|---|---|---|
| ~~T4.1~~ | **DEFERRED to POST-BROADCASTER (MVS-D-79, HUM LEAD 2026-09-05).** ALERTS - READ ORDER — one ordering, one Max — is **not built in 0.14.0**. The ORDERING ITSELF SHIPS: the ladder, the fence, the Max and the divert count are all built and pinned (T1.1–T1.4, T4.3); what is deferred is letting a **human rearrange it**. Two reasons, both the HUM LEAD's: 0.14.0 has already changed a great deal and this is a point release of its own; and giving listeners a knob on hazard ordering is not obviously wanted yet. **This supersedes MVS-D-56's shipping condition** — 0.14.0 no longer holds for it. | DR-9 |
| **T4.2** | Bonsall, CA compiled in as the default origin and **shown in Settings as the Default**. First run stays "no config file exists" — nothing new is built (R-6). | R-4 |
| **T4.3** | The divert notice: head → headlines → *"For more details about these and N other alerts…"*, destination `[w]` in Observer. The provider list joins the way a person says one. | DR-14 |

---

## Phase 5 — Gates and exit

| Gate | Note |
|---|---|
| `make verify` | fmt · vet · tidy · vuln · race · lint-imports · lint-watermark · gate-controls |
| `make alloc-budget` | The ticker renders 24/7; the band is on the frame's hot path |
| `make p10 A2DH=…/dist/a2dh` | 0 live, 0 unmatched. Any exemption **presented for ratification**, never self-approved — **and the reason plus its ratification are mirrored into `04-development/director-build-log.md`** (PL-16). The ledger is gitignored, so the tracked build log is the only reviewable record; "presented for ratification" is meaningless without naming where. Also reconcile the **seven entries still reading "Ratify at the ⟨X⟩ gate"** against the ones carrying an explicit "RATIFIED by the HUM LEAD ⟨date⟩" (PL-14). The Director work added exactly one row — `app/pump.go` `loop`, ratified 2026-09-02 — and it is already on the ratified side |
| Goldens | **One deliberate move**: DR-13's fence changes the tape (a distant significant disaster now appears). Every other golden unchanged |
| AA register | Any new render token registered in `contrast.go:aaPairs`, or it is never measured (F-18) |
| Red team | BUILD exit, mandatory at SEV-0 |

---

## Test strategy — the anti-vacuity rules

Five fixtures this release passed while proving nothing: an invented CAP id the OID grammar rejects,
an alert expiring exactly at `now`, a sweep measuring a 35-cell lane instead of 2,563, a churn model
that could not distinguish the two rules under test, and a product NWS does not issue. Each rule
below exists because one of those got through.

1. **Assert the fixture is VALID before asserting behaviour.** Every alert fixture is first asserted
   *accepted* by `severe.Classify` **and** inside its active window. Only then is its ordering,
   distance or read position asserted. This is the single most repeated mistake in the record.
2. **Watch every test fail before it passes.** A test that has never failed proves nothing about the
   defect it claims to cover.
3. **Mutate your own change before handing it over.** `06_docs/mutants/run.sh` — green baseline
   required, edit-applied asserted, build and vet gated. Verdicts are `CAUGHT / SURVIVED / INVALID /
   UNAPPLIED`, never a boolean.
4. **Carry the previous reviews' mutants forward verbatim.** Author-invented mutants probe only what
   the author already suspects; a reviewer's are by construction the ones that slipped past.
5. **No claim without a command.** Every behavioural sentence in a commit message or comment has a
   run behind it, or it does not go in. Four of nine checkable claims in one round this release were
   false.
6. **Enumerate the consumers before fixing.** *Fixed one of N paths* produced the blocking finding in
   both previous rounds.
7. **An instrument is built the way production is built, and its constructor carries the reason.**
   Added after PLAN, because the rule was missing and the gap was immediately expensive: the
   card-build instrument constructed its httpx client without `RatePerSec`, took the default of 5/sec
   where the app uses 30, and reported **2.14 s** for what is **1.03 s** — it timed its own
   misconfiguration, the figure reached four documents, and PD-2 was decided on it. Rules 1–6 all
   concern the *fixture*; none of them covers the *instrument's configuration*. The tell was in the
   data — three samples within 0.1 s of each other is a token bucket in lockstep, not a network — and
   it read as reassuring consistency.

## What is deliberately NOT built

Recorded so nobody builds it on their own judgement, and so a reviewer does not read its absence as
an oversight.

| Not built | Why |
|---|---|
| The burst-during-burst ladder (`≤4` ride · `≥5` divert · the reset) | Broadcaster machinery. Observer's rule is hard and fast (Q-2, Q-3) |
| BURST PANIC COOLDOWN | Broadcaster-era, confirmed (Q-3) |
| Station callsign; watchlist-from-geometry | PD-1 — a field nobody reads is dead code |
| Operator card editing (`DROP`/`DELAY`/`PROMOTE`) | The card model carries the fields; the controls arrive with the Broadcaster UI |
| A composition `Profile` field on the card | PD-4 — the Composer reads config; the Director does not relay it |
| A Broadcaster service-radius cap | PD-6 — defer |
| Any change to F-16's parked-offset behaviour | Two attempts, two reverts. It wants its own DISCOVER with the geometry modelled first |
