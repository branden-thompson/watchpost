# Requirements — Station Director & the Lineup

| Field | Value |
|---|---|
| Scope | **MAJOR SUB-FEATURE** of 0.14.0 `feature/multi-voice-support` |
| Phase | **DISCOVER** · LEVEL-1 · SEV-0 · HUM LEAD |
| Directives | FULL RCC · FULL PLAN · FULL DIAGRAMS · FULL TDD · FULL REPORTS · FULL GIT |
| Source | `director-charter.md` · `lineup-model.md` (L-1…L-8, S-1…S-8, T-1…T-5, R-1…R-6) · `read-order-design.md` · `README-director-handoff.md` · `08-reports/red-team-build-round2.md` |
| Problem (locked) | *What Watchpost reads aloud and what its news ticker shows are decided by separate code that agrees by coincidence, and the arrangement is a constant nobody can set — so a bound in one path can silence a hazard the other believes is being read, and no test can state the arrangement it means to assert.* |
| Supersedes on landing | F-17 · MVS-D-54 · `capBurst` / `maxBreaking` / `breakingAllowance` / `breakingCap` and the per-lane floor · round-2 defect 2's open item R-1 · the ten-rung scale in `read-order-design.md` |

Every requirement names how it is verified. **The instrument comes before the fix**
(`06_docs/remediation-review-loop.md`): a fixture is asserted *valid* before its behaviour is
asserted, and a test that has never failed proves nothing.

## 0. What changes for the listener (plain language)

Nothing they hear changes by accident. Deliberately:

- **The order alerts are read in is theirs to set**, and has a stated default, instead of being a
  constant three attempts failed to tune.
- **An evacuation order is always read in full**, however many there are — it is never one of the
  five that fit.
- **A stale disaster stops outranking live weather.** A landslide four days ago no longer leads a
  thunderstorm warning issued this morning.
- **A big quake far away can still reach them.** A significant disaster carries its own reach,
  because such a disaster has effects well past its point.
- **They are told what they did not hear.** *"For more details about these and N other alerts…"* —
  in Observer, `[w]`; on a Broadcaster station, a website.
- **The ticker's callout is a promise, not a coincidence.**

## Functional requirements

### The object

| ID | Requirement | Verify by |
|---|---|---|
| DR-1 | **The Lineup is a first-class object with exactly one writer.** The Director owns every mutation. Every other component — the synth layer, the fetch layer, the news ticker, the radio deck — **reads** it to decide what to pre-load, pre-build or sync (HUM LEAD: *"the other systems need to understand what a scheduled slot in the lineup means FOR THEM"*). | A test that builds a lineup and asserts its contents **directly**, with no timers and no goroutines — the property the current design lacks. A package boundary that makes mutation from outside a compile error, plus a declset test pinning the exported surface. |
| DR-2 | **Planning is separable from execution.** The planned lineup is a pure function of (proposals, settings, default location, clock). Executing it is a separate stage. | Determinism test: the same inputs twice produce byte-identical lineups. A golden of the planned lineup for a fixed arrival set. Mutation-tested with `06_docs/mutants/run.sh` — deleting any ordering rule must fail a test. |
| DR-3 | **Three tracks, with stated precedence.** **Main track** — the reads (location reports, event reads); cards are queued, cast-assigned, carry their own settings, and can be moved or cancelled. **Alert rail** — a priority lane, always present and usually empty; when it holds anything those items are read first, in order, and it drains before normal programming resumes. **NO ADMITTED CARD IS EVER DROPPED UNREAD** — bounds apply at *admission*, never mid-burst. This is the guarantee `breakingCap` violated, and stating it is what removes the defect rather than moving it (RT-9). **The bed** — the live NOAA relays, a *selectable* resource the Director may cut over to, not a queue. | Table-driven scheduling tests over {empty, non-empty} × {main track, alert rail} × {bed playing, bed stopped}; a test that no bound cuts a burst once it has started. |
| DR-4 | **Three card origins.** **Observer** proposes (it fetches, normalises and composes); the **Operator** proposes (play/pause/stop, "read Oceanside"); and the **Director itself** generates structural cards (heads, tails, transitions, the divert notice). | `origin` asserted per card across each path. |
| DR-5 | **The Director pre-screens every proposal against the human's settings** — the radius fence, alert preferences, mute state, Max — and enforces them; it never invents an arrangement. | A matrix over settings × arrivals asserting which proposals are admitted and which are refused, through the real entry point. |
| DR-6 | **The card carries what the Operator must see:** `id`; **slot type** (Location Report · Marine Report · Special Weather Statement · …); **read by** — a voice profile available **on this machine**, or `N/A`; **max duration** — `Full length` by default or a per-card number of seconds; **the text of the read**, a field that exists from creation but is **EMPTY until the card reaches standby** (DR-7 — the two rows are one rule seen at two times, RT-6); plus `state`, `origin` and a lock. Controls: `DROP`, `DELAY` (down the queue), `PROMOTE` (up the queue). | Card-model test; a test that a card at ON AIR is locked against edits; a test that `read by` never names a voice this host cannot resolve (the FR-7 fallback chain applies). |
| DR-7 | **A card's text materialises at standby.** From creation it carries a headline and shape (slot type, subject, estimated length); the full script fills in as the card nears the air, so a queued report is not composed from data that will be stale when it plays. **The pre-build yields to the alert path** (PL-8): both share one httpx token bucket (`RatePerSec: 30`, `app/app.go:123`), and `synth.WithPriority` gives Director work a private *render* lane with no HTTP equivalent — so a pre-build in flight must not delay an alert's own fetches. | A test that a card queued at T has text composed no earlier than standby, and that its data is fetched within the standby window; a test that an alert fetch issued during a pre-build is not delayed behind it. |
| DR-8 | **The lineup is ephemeral; the configuration that builds it persists.** Nothing restores a slot-by-slot lineup across restarts. An in-the-moment reorder does not persist — a change the human wants to keep is a change to the Director's **composition configuration**. | Restart test: settings restored, lineup rebuilt from scratch. |

### The read order

| ID | Requirement | Verify by |
|---|---|---|
| DR-9 | **One ordering and one Max** (R-1). ALERTS - READ ORDER carries category priority and the sort dimensions within each; `Max` is a single number for the whole burst across all categories. The per-category `Max` row and `TOTAL READS` are removed from the specified group. | The Settings golden; a config round-trip; a test that per-category budgets have no effect. |
| DR-10 | **The default order ships as configuration, not code**, and reproduces the ratified table: Emergency Orders (exempt from Max) · Close {Disasters, Warnings, Watches, Advisories, Spec. Statements, Marine} · Remaining {the same six} · **Forecasts are never read AS AN ALERT** — they stay part of the location report, which is most of the broadcast; the phrase means "never a takeover" and nothing more (RT-5). The per-category read rank lives in `category.Spec` beside `Rotation` — a **third** ordering, distinct from tab order and lane rotation. | A test that the shipped defaults reproduce the table rung by rung; a test that Forecasts never enters a burst; a registry test that the three orderings are independently declared, so harmonising them fails loudly. |
| DR-11 | **Emergency Orders are read first and always, and they SPEND the budget** (R-2, Q-1). They are placed at the head of the burst; the **remaining** budget is filled by read order. Only when the emergency orders **alone exceed Max** does the burst overrun — then all of them are read and everything else is diverted, because *"the sheer amount of DO THIS NOW means the warnings are comparatively less important than the listener knowing the entirety of the emergency situation."* They are **not** exempt from the service radius. The overrun is bounded **by the feed**, never by a constant — and the datum is named rather than asserted (PL-7): feed-sourced rows are already capped at `globalfeed.MaxPerLane` (**30**, `domains/globalfeed/stack.go:51`) and watchlist-sourced rows by the severe index's own cap, so the read is a `range` over an already-bounded slice, which is P10-02's required form rather than an exemption. | At Max 5: 1 emergency + 20 warnings reads the emergency **and four warnings**, diverting 16; 7 emergencies + 18 warnings reads **all seven**, diverting 18; an out-of-radius emergency order is absent; the P10 gate passes with no added cap. |
| DR-12 | **Disasters vs. Warnings is conditional on the fence** (R-2). With a radius set, Disasters outrank Warnings unconditionally. With no fence (`All`), a Disaster must also be **within 24 hours** (R-5, ratified) to outrank a Warning; past that it sorts into the remaining band. **The no-fence branch is the DEFAULT path**, not the edge case: `config.TickerRadiusMi` defaults to `0 = All` (`platform/config/config.go:260`), so every fresh install lands there (RT-7). | The four-cell matrix {fence, no fence} × {fresh, stale}, with the **no-fence** case as the primary fixture; a boundary case either side of 24 h. |
| DR-13 | **The radius is a hard fence with a significance exception** (R-3, in scope for 0.14.0). It filters what reaches the lineup at all, except that a **significant disaster carries its own reach**, because such a disaster has proximal effects. Cascades are not modelled: a derived hazard arrives as its own local alert inside the fence. | The HUM LEAD's own cases: an M7.5 at ~120 mi is admitted; an M9.0 in Fairbanks is not; the fixture is asserted to pass the classifier and the active window **before** its ordering is asserted. |
| DR-14 | **The burst is a snapshot, and the remainder is spoken.** Head naming the providers → the Max headlines → a divert notice carrying the **count** and the **destination**. The destination is the one edition-specific value: `[w]` in Observer, a website list on a Broadcaster station, joined the way a person says a list. **The count is `arrivals − alerts read`** (RT-8): 40 arrive and 5 are read → *"these and 35 other alerts"*; 21 arrive of which 1 is an Emergency Order, so the emergency and four others are read → *"these and 16 other alerts"*; 25 arrive of which 7 are Emergency Orders, blowing the budget, so all 7 are read → *"these and 18 other alerts"*. | Script test on the composed burst; a test that the spoken count equals exactly what was not read; a test that the two editions differ in **exactly** the destination. |
| DR-15 | **Structural cards do not count against Max** (G-6). Heads, tails, transitions and the divert notice are the Director's arrangement; Max counts **alert reads**. | A Max of 5 with a head, a tail and a divert notice reads five alerts. |

### On the air

| ID | Requirement | Verify by |
|---|---|---|
| DR-16 | **Resumption is per mode** (S-4). A **read** resumes mid-sentence — optionally behind a transition card ("we now return to our regularly scheduled report in progress"). A **live bed** ducks and normalises back up, which is the give-way the engine already implements by source kind (`domains/radio/player/engine.go:391 Suppress`, `:399 Restore`, `:414 giveWayLocked`). | The existing suspend/resume tests extended through the lineup's entry point; a test that a read interrupted mid-line resumes at the same line. |
| DR-17 | **The news ticker keeps its own job** (T-5, closing G-11). It shows a **centred callout** of the alert being read, up to the Max, then resumes rotating between categories with its own marquee. Its rotation is **not** lineup-driven. The contract is narrow and one-directional: *"a notable event has been declared — when I call it out, show it."* | A test that the callout matches the card on air; a test that the rotation is unchanged when the lineup is idle; the existing ticker goldens stay green. |
| DR-18 | **The cue is fire-and-trust with a post-hoc record** (S-8). The voice never blocks on the ticker. The ticker records that it complied, and that record is both a debug surface and what tests assert on. | A test asserting the cue record for a takeover; a test that a ticker which never complies does not delay or fail the read. |
| DR-19 | **Nothing else changes.** Every 0.13.0/0.14.0 behaviour not named above is byte-identical: the broadcast's segment order, text and pauses; the narration lines; the tape's rotation; the tones; the cast resolution. **One deliberate delta:** DR-13's significance fence modifies `scopeToRadius`, which feeds `tapeItems` — so a distant significant disaster now appears on the tape and in `[w]` where it did not before (RT-4). It gets its own golden. | The existing goldens, declsets and PTY journeys stay unchanged and green; the tape golden moves once, for the reason above. Any further golden that must move is listed here first, with its reason. |
| DR-20 | **The clock is injected.** DR-2's determinism is unachievable otherwise: `app/ticker.go:cycle` calls `time.Now()` five times, `readBreaking` calls it per event, and both the 24-hour freshness window and the active-window filter read it. A clock seam is a **prerequisite of the design**, not an implementation detail (RT-3). | The determinism test in DR-2 runs on a fixed clock; no assertion path calls the real clock. |

| DR-22 | **The pump is supervised** (PL-2). Under Approach C the pump is the single owner, so a panic in an effect executor would wedge **both tracks and the bed** — a strictly larger blast radius than today, where `startTakeover` guards a panic precisely because one "would wedge the marquee for the life of the process" (`app/ticker.go:245-251`) and that is *one* surface. An effect that panics is contained, reported as a `Fault`, and the schedule continues. | A test that a panicking effect executor does not kill the pump: the next event is still processed and the card is failed rather than the station going silent. |
| DR-23 | **The lineup is observable at runtime** (PL-4). Not the Broadcaster UI — S-1 ruled Observer's rail not inspectable — but a **diagnostic** surface, because a scheduler with no runtime signal is how the next defect goes unnoticed on a path that has already produced ~30 gate-passing ones (RD-1). Minimum: one `WATCHPOST_DEBUG_RADIO` line per card state transition, in the existing format. The emitter already exists and P-1 proved its value. | A test asserting the transition lines for one card's full life; the format matches the existing `segment`/`cycle-end` entries so one log stays readable as one timeline. |
| DR-24 | **The cue and its release are paired, with the same guarantee the duck already has** (PL-10, HUM LEAD ratified). Today `TickerBreakingDoneMsg` is sent on **one** path (`app/ticker.go:426`) with **five early returns above it** that send nothing, while the audio side is released unconditionally by the director's `release`/`settle` — which is why the comment there reads "the director still restores": it restores the audio, and nothing restores the band. Today only shutdown reaches it; **under the Lineup a discarded, cancelled or superseded card makes it ordinary.** Therefore: **any transition out of ON AIR emits a release effect** — a property of `Step`'s output, not a discipline about call sites. | A test over every exit from ON AIR (read in full · discarded · superseded · cancelled · context ended) asserting a release effect in each; the ticker is never left holding a stale callout with its rotation frozen. |
| DR-21 | **Failures reach the Director only when they change the schedule** (D-C-4, RT-2, ratified Q-4: *"the Director only cares when it affects their world"*). **One escalation channel.** Producers own their own retries. The Director **re-routes** by default — relay → synth, voice → fallback voice — and **re-plans** only when a card cannot be delivered at all. **Surfacing is graded:** a modal only for a fault that stops the schedule; a fault the Director routes around keeps today's treatment — the reason on the player's detail line plus a one-time `snapshot.WarnRadioUnavailable`. Escalating a self-healing relay failure to a modal would be a noise regression. **The modal is a new user-facing surface and carries the obligations of one** (PL-12): its colour tokens are registered in `platform/render/contrast.go:aaPairs` — an unregistered token is never measured and silently "passes" AA (F-18) — it is dismissible from the keyboard, and it takes focus without trapping it. | The four fault kinds (relay dies mid-play, voice cannot render, fetch fails, text never materialises) each asserted for route-vs-replan and for whether a modal is raised; a test that today's relay→synth fallback raises no modal; the AA gate measures every token the modal paints. |

## Designed for, not built in 0.14.0

Recorded so the seams are right, and so nobody builds them on their own judgement.

| Concept | Why it is out |
|---|---|
| **The burst-during-burst ladder** (`< 4` ride · `≥ 5` divert · the counter reset) | Broadcaster machinery. **Observer's rule is hard and fast** — each burst evaluated on its own, read the Max, divert the rest to `[w]` — and that handles large bursts and bursts-of-bursts alike (Q-2, Q-3). |
| **BURST PANIC COOLDOWN** — the circuit breaker | **Broadcaster-era, confirmed.** Observer needs no cooldown because `[w]` is always available. |
| **Station identity, service area, ON AIR / STANDBY** | PD-1: stub-or-build is a PLAN decision with a recommendation owed. |
| **Operator card editing** (`DROP` / `DELAY` / `PROMOTE`) | The card model carries the fields; the controls arrive with the Broadcaster UI. |

## Non-functional requirements

| ID | Requirement | Verify by |
|---|---|---|
| NFR-D-1 | **Testable by construction.** The test states the arrangement it wants and asserts what is read — no timing, no sleeps, no goroutine coordination in the assertion path. | Every DR above is verified without a real clock. |
| NFR-D-2 | **P10 holds.** Bounded loops (P10-02) including DR-11's data-derived emergency bound; ≤60 lines / ≤40 statements per function (P10-04); invariant density ≥2.0 on new functions (P10-05); no mutable package globals (P10-06) — the Lineup is an instance, never a package var. | `make p10 A2DH=<framework build>` — 0 live, 0 unmatched. Any exemption is **presented for ratification**, never self-approved. |
| NFR-D-3 | **The import direction holds.** Nothing under `modes/` imports `domains/*`. Shared vocabulary goes in a `platform/` leaf. | `scripts/lint-imports.sh` via `make verify`. |
| NFR-D-4 | **The frame budget holds.** The ticker renders 24/7, so the band is on the frame's hot path. | `make alloc-budget`; the existing frame pins. |
| NFR-D-5 | **Every new render token is registered** in `platform/render/contrast.go:aaPairs` (F-18 — an unregistered token is never measured and silently "passes" AA). | The AA gate, plus F-18's completeness test if it lands first. |
| NFR-D-6 | **The category registry stays the single source.** The Director must not become a fifth list (F-21). Read rank is a `category.Spec` field. | The registry's completeness tests; a mutation that removes a rank must fail. |
| NFR-D-7 | **Discard is cheap and safe.** Anything prepared ahead may be thrown away, so preparation has no side effects that outlive it. | A test that discarding a standby card leaves no audio, no marks and no files. |
| NFR-D-8 | **Voices are station-local.** The schedule is built against what this machine has. | The existing fallback matrix, applied to `read by`. |

## Constraints

- **`scripts/lint-imports.sh`** — the reason `platform/category` exists; the same rule binds anything the Director shares between layers.
- **The remediation review loop** (`06_docs/remediation-review-loop.md`) governs every fix on this path: failing test first through the real entry point, watched to fail; mutate your own change; a fresh adversarial reviewer who must run something.
- **Do not tune `maxBreaking` / `breakingCap` / `capBurst`.** Three attempts, three moved boundaries. This design replaces them.
- **Do not "fix" the ticker's parked-offset behaviour** (F-16). Two attempts, two reverts. It wants its own DISCOVER with the geometry modelled first.
- **Read order, counts, pacing and precedence are HUM LEAD rulings**, even mid-fix. Present the fork with evidence; do not choose.
- **A passing test is not coverage.** Roughly thirty remediation defects this release passed every gate.

## Decisions owed to the HUM LEAD at PLAN

Each gets a **recommendation with reasoning**, not a question.

| # | Decision |
|---|---|
| PD-1 | **Stub or build** station identity, service area and ON AIR / STANDBY (S-1) — only if the seam is load-bearing for Observer's radio under the new architecture. |
| PD-2 | **Standby vs. line-by-line rendering.** The charter makes standby first-class (*"get that report ready … so we can immediately cut over"*); S-5 says line-by-line with no pre-render may be enough. They disagree about an audible cutover gap. |
| PD-3 | **An interruption that outlasts the report's validity.** Resumption is mid-sentence with no staleness rule, but where the rail drains longer than the report stays valid, the transition read may need to acknowledge that and move to the next scheduled report. |
| PD-4 | **How the Operator helps the Composer compose** (S-7) — the Composer owns report contents, but the human wants some hand in it. |
| PD-5 | **Renaming `director` → `mastercontrol`**, with the ticker cue as its explicit responsibility. Proposed, awaiting ratification. |
| PD-6 | **A Broadcaster service-radius cap** (~150 mi, FRS/GMRS/ham) — floated at T-1, not ruled. |

## Still owed by the HUM LEAD before 0.14.0 SHIP

Unchanged and unrelated to this sub-feature: M3's two listening trials (which also settle the read-length
assumption), the Linux validation protocol, F-6's M2 journey re-run, and the README screenshots.
