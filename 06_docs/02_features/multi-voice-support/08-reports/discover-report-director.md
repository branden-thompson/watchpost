# DISCOVER Report — Station Director & the Lineup

| Field | Value |
|---|---|
| Project | **watchpost** — terminal-native live weather station |
| Scope | **MAJOR SUB-FEATURE** of 0.14.0, branch `feature/multi-voice-support` |
| Classification | **LEVEL-1 · SEV-0 · HUMAN LEAD** |
| Directives | FULL RCC · FULL PLAN · FULL DIAGRAMS · FULL TDD · FULL REPORTS · FULL GIT |
| Phase | DISCOVER → **awaiting HUM LEAD approval to transition to PLAN** |
| Date | 2026-09-01 |
| Artifacts | `01-objectives/lineup-model.md` · `01-objectives/director-requirements.md` · `02-analysis/director-risks.md` · `01-objectives/director-charter.md` · `01-objectives/read-order-design.md` |

## Problem (locked)

> *What Watchpost reads aloud and what its news ticker shows are decided by separate code that agrees
> by coincidence, and the arrangement is a constant nobody can set — so a bound in one path can
> silence a hazard the other believes is being read, and no test can state the arrangement it means
> to assert.*

**Why it earns a full RCC and PLAN.** Two red-team rounds found the same class of defect and each fix
**moved** the failure boundary rather than removing it. The last measurement put a starvation edge at
~8.14 s per read: past it a quiet lane goes unspoken permanently under sustained arrivals. The
HUM LEAD's conclusion (MVS-D-56) was that ordering is not a constant to be tuned but a preference the
listener sets, owned by a Director that also keeps the ticker in step.

## The design in one page

**The Lineup is a first-class object with exactly one writer.** The Director owns every mutation;
every other system — synth, fetch, ticker, radio deck — **reads** it to decide what to pre-load,
pre-build and sync. That inverts today's arrangement, where each component decides for itself, and it
is the seam that makes Broadcaster an extension rather than a rewrite.

**Three tracks.** The **main track** carries the reads. The **alert rail** is a priority lane, usually
empty, that drains before normal programming resumes — and **no admitted card is ever dropped
unread**, which is precisely the guarantee `breakingCap` violated. The **bed** is the live NOAA
relays, a selectable resource rather than a queue.

**Planning is separable from execution.** The planned lineup is a pure function of arrivals,
settings, default location and clock. That is what makes the design testable by construction: a test
states the arrangement it wants and asserts what is read. Nothing in the current design permits that,
which is why every fixture written against it this release was vacuous.

**Observer is Broadcaster with an audience of one**, and is the harder problem — its data has no
geographic limit, where a Broadcaster station is constrained by service radius.

## Requirements

Full text in `01-objectives/director-requirements.md`, each naming how it is verified.

| ID | Requirement |
|---|---|
| DR-1 | The Lineup is a first-class object with exactly one writer; everything else reads it |
| DR-2 | Planning is separable from execution; the plan is a pure function of its inputs |
| DR-3 | Three tracks with stated precedence; **no admitted card is ever dropped unread** |
| DR-4 | Three card origins: Observer, the Operator, and the Director itself |
| DR-5 | The Director pre-screens every proposal against the human's settings |
| DR-6 | The card carries what the Operator must see, incl. text and a per-card max duration |
| DR-7 | A card's text materialises at standby, so a queued report is not composed from stale data |
| DR-8 | The lineup is ephemeral; the configuration that builds it persists |
| DR-9 | One ordering and one Max; the per-category `Max` row is removed |
| DR-10 | The default order ships as configuration; read rank lives in `category.Spec` |
| DR-11 | Emergency Orders lead and spend the budget, overrunning it only when they exceed Max |
| DR-12 | Disasters vs. Warnings is conditional on the fence; 24 h freshness with no fence |
| DR-13 | The radius is a hard fence with a significance exception for disasters |
| DR-14 | The burst is a snapshot; the remainder is spoken with a count and a destination |
| DR-15 | Structural cards do not count against Max |
| DR-16 | Resumption is per mode: a read resumes mid-sentence, a bed ducks and normalises |
| DR-17 | The news ticker keeps its own rotation; it defers only for the callout |
| DR-18 | The cue is fire-and-trust with a post-hoc record |
| DR-19 | Nothing else changes, with one deliberate delta (the fence changes the tape) |
| DR-20 | The clock is injected — a prerequisite of DR-2, not a detail |
| DR-21 | Failures reach the Director only when they change the schedule; surfacing is graded |

**Non-functional:** testable without a real clock · P10 holds (incl. a data-derived bound for DR-11)
· the import direction holds · the frame budget holds · every new render token registered (F-18) ·
the category registry stays the single source (F-21) · discard is cheap and safe · voices are
station-local.

## Decision log

The sub-feature uses its own ruling series — **L-** (lineup object), **S-** (sharpening), **T-**
(tracks and ticker), **R-** (read order), **Q-** (final clarifications), **RT-** (red team). All were
given by the HUM LEAD on 2026-09-01 and are recorded with their verbatim context in
`01-objectives/lineup-model.md`.

**Proposed for the parent ledger: MVS-D-64 — "DISCOVER approved for the Station Director sub-feature,
adopting the L/S/T/R/Q/RT ruling series."** One entry pointing at the sub-feature's own log, rather
than ~35 numbers in the parent. *Flagged as a choice, not made silently.*

| ID | Ruling |
|---|---|
| L-1 | The live relays are a **selectable bed**, always available for the Director to cut over to |
| L-2 | **Main track** (the reads) + an **alert rail** that drains fully before normal programming resumes |
| L-3 | Mostly **order**, not clock; a card may carry its own duration cap |
| L-4 | Two proposers — Observer and the Operator. The Director executes the human's will and synchronises the producers; **the other systems must understand what a slot means for them** |
| L-5 | The alert rail is a priority lane; it drains by being read or by an operator cancelling |
| L-6/7 | Broadcaster is **ON AIR / STANDBY**; no dead-air controls; sign-off is an auto-inserted script; station geometry (callsign + coords + service radius) is the preferred default filler |
| L-8 | The lineup is **ephemeral**; the configuration that builds it persists |
| S-1 | 0.14.0 builds the foundation and puts Observer's radio on the Lineup; station identity / service area / ON AIR-STANDBY are extensions, **stub-or-build is a PLAN decision** |
| S-2 | The human is the Operator; Observer's `[space]` stays inert; queuing is a Broadcaster concern |
| S-3 | A diverted alert is **not discharged**; the settings are the opt-in and the pre-screen is at admission |
| S-4 | A read resumes **mid-sentence**, optionally behind a transition read; a bed ducks and normalises |
| S-5 | One-ahead rendering is acceptable; line-by-line may be enough |
| S-6 | The card's field list and controls |
| S-7 | **The Composer owns report contents** |
| S-8 | The cue is **fire-and-trust with post-hoc confirmation** |
| T-1 | The burst is a snapshot; the **pointer** differs by edition — `[w]` in Observer, a website on a station |
| T-2 | **The main track is built at 0.14.0** and is invisible to the listener |
| T-3 | **Intercard transitions are cards; intra-card transitions are the Composer's utterances** |
| T-4 | A card's text materialises at standby; refine once real reads exist |
| T-5 | The news ticker keeps its own rotation and defers only for the callout — **G-11 closed** |
| R-1 | **One ordering, one Max** |
| R-2 | Emergency Orders lead and spend the budget; **not** exempt from the radius. Disasters outrank Warnings in-fence; **recent** Disasters outrank Warnings with no fence |
| R-3 | **The significance fence is in scope for 0.14.0** |
| R-4 | Bonsall shown in Settings as the Default; Setup auto-opens on first run |
| R-5 | **The Disaster freshness window is 24 hours** |
| R-6 | First run stays "no config file exists"; a version-keyed marker was **declined** |
| Q-1 | Emergency Orders **spend** the budget and overrun it only when they exceed Max |
| Q-2 | The burst-during-burst boundary is `≤ 4 ride / ≥ 5 divert` |
| Q-3 | **BURST PANIC COOLDOWN is Broadcaster-era**; Observer's hard-fast rule handles bursts-of-bursts |
| Q-4 | Failures: one channel, re-route by default, re-plan only when undeliverable; **modal only for a fault that stops the schedule** |

**Closed without a ruling:** **G-9** — `[s]` is Settings, verified at `modes/tty/dashboard.go:274`.

## Critical analysis (SEV-0, mandatory)

Run single-agent against this phase's own output — the portable baseline per
`_a2dh/AGENTS_a2dh.md`. Nine findings; all are resolved.

| # | Severity | Finding | Disposition |
|---|---|---|---|
| RT-1 | **Blocking** | Nothing stated what happens to a burst arriving while the rail drains — the old defect's exact home | **Ruled** (Q-1…Q-3): Observer applies its hard-fast rule per burst; the Broadcaster ladder is designed-for, not built |
| RT-2 | **Blocking** | **The author over-claimed**: "D-C-1 … D-C-6 are all answered" was false. D-C-4 (failure escalation) was unanswered and D-C-3 only partly | **Ruled** (Q-4) and captured as DR-21. D-C-3 remains open as PD-2 |
| RT-3 | **Blocking** | DR-2's determinism has an unbuilt prerequisite: `time.Now()` is called five times in `cycle` alone | **Fixed** — added as DR-20 |
| RT-4 | Serious | DR-13 contradicted DR-19: the significance fence changes the tape | **Fixed** — named as a deliberate delta with its own golden |
| RT-5 | Serious | "Forecasts never read" read literally would delete forecast content from location reports | **Fixed** — "never read **as an alert**" |
| RT-6 | Serious | DR-6 and DR-7 contradicted each other on the card's text | **Fixed** — one rule at two times |
| RT-7 | Serious | The freshness rule is the **default** path, not an edge case: `TickerRadiusMi` defaults to `0 = All` | **Fixed** — the no-fence case is now the primary fixture |
| RT-8 | Serious | The divert count was unspecified | **Ruled** — `arrivals − alerts read`, with three worked examples |
| RT-9 | Note | "The rail drains" was near-tautological; the guarantee that matters was missing | **Fixed** — "no admitted card is ever dropped unread" added to DR-3 |

**The author's own correction (RT-2) is recorded here deliberately.** A wrong claim in a phase
document is not self-correcting — the next reader takes it as evidence, which is the failure mode
this release has already paid for four times.

## Risks

Full register in `02-analysis/director-risks.md`. Severities are about what reaches a listener.

| # | Sev | Risk |
|---|---|---|
| RD-1 | HIGH | This exact code closed 17 findings and opened ~30, all passing gates |
| RD-2 | HIGH | Absorbing the watchlist rotation changes audio with UAT history (the duck-lift bug) |
| RD-3 | HIGH | Four goroutines touch this; a single-writer Lineup serialises them |
| RD-4 | MED | Measurement on this path has been unreliable — the instrument is fixed, the habit is not |
| RD-5 | MED | The significance fence is new safety-path behaviour landing inside a rewrite |
| RD-6 | MED | Emergency-order overrun is the one read with no constant bound |
| RD-7 | MED | Broadcaster scope leaking into 0.14.0 |
| RD-8 | MED | Bulk renames have destroyed content twice this release |
| RD-9 | MED | PD-2 unresolved can produce an audible regression either way |
| RD-10 | LOW | Three per-category orderings invite a later "harmonisation" |
| RD-11 | LOW | "Text at standby" is validated only when Broadcaster is built |
| RD-12 | LOW | This work edits the ticker, where F-16 is unfixed |

**Dependencies:** `platform/category` **done** (F-21 closed) · F-18 AA register **open** · D-1
**open**, needs a live-feed count · M3 listening trials **owed by the HUM LEAD**.

## Decisions carried to PLAN

Each gets a written recommendation with reasoning, not a question.

| # | Decision |
|---|---|
| PD-1 | Stub or build station identity, service area, ON AIR / STANDBY |
| PD-2 | Standby vs. line-by-line rendering — the charter and S-5 disagree about an audible gap |
| PD-3 | An interruption that outlasts the report's validity |
| PD-4 | How the Operator helps the Composer compose |
| PD-5 | Renaming `director` → `mastercontrol`, with the ticker cue as its explicit responsibility |
| PD-6 | A Broadcaster service-radius cap (~150 mi) |

## Gate report

```
QUALITY GATE REPORT | watchpost — Station Director & the Lineup | SEV-0
-------------------------------------------------------------
  [PASS] requirements_documented: DR-1..DR-21 + 8 NFRs, each naming its verification
  [PASS] constraints_identified: 6 constraints; import direction, remediation loop,
         3 explicit do-nots carried from the reverted attempts
  [PASS] risks_assessed: RD-1..RD-12 with severity, evidence and mitigation;
         4 dependencies named
  [PASS] critical_analysis_complete: 9 findings, all resolved; 2 blocking findings
         ruled by the HUM LEAD, 1 self-correction recorded
  [----] human_approval: AWAITING
  [----] report_published: AWAITING (this document)
-------------------------------------------------------------
  OVERALL: PARTIAL PASS — blocked only on human approval
```

## Recommendation

**Proceed to PLAN.** Every architectural question that shapes the object model has been ruled by the
HUM LEAD, the two blocking red-team findings are closed, and the six decisions that remain are
genuinely PLAN-shaped: they need a recommendation grounded in measurement, not more discovery.

Two things I would carry into PLAN as first work rather than late work: **RD-3's concurrency model**,
because the choice determines the shape of everything above it, and **PD-2's measurement** — the
current cutover gap is measurable today via `WATCHPOST_DEBUG_RADIO` segment timings, and measuring it
before deciding is cheaper than arguing about it.
