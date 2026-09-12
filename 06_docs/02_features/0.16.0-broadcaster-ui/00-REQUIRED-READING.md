---
title: "0.16.0 — REQUIRED READING, every session and after every compaction"
date: 2026-09-09
phase: ALL
sev: SEV-0
authority: HUM LEAD
status: "MANDATORY.  Re-read at session start and after any context compaction, for the life of 0.16.0."
---

# Read this before touching 0.16.0

**Why this file exists.**  On 2026-09-09 a ratified batch was built and reverted the same day over four
blockers, and every one of them was already answered in 0.14.0's documents.  The failure was not a gap
in the record.  **It was not reading the record**, across a session long enough that the context was
rebuilt from the code instead.

**This is a re-required read, not a reference.**  It survives compaction on purpose.

---

## The five roles (`multi-voice-support/03-architecture-design/role-model.md`)

| Role | Owns | Must not |
|---|---|---|
| **Card Producer** | what cards should exist, and when | what the card SAYS |
| **Card Composer** | what the card says, **at standby** — not at proposal | arrangement, and the UI |
| **Director** | the ORDER.  Its one additive act is the inter-card transition | perform anything.  It is a pure function |
| **Reader** | performing the LIVE card, **including tone and pauses** | decide the running order |
| **MasterControl** | the mechanics of what is broadcast, and it declares ON AIR / STANDBY | plan, or own the bed — it SILENCES the bed |

**Producer and Composer are two roles, split TEMPORALLY.**  A card is proposed with no words; the text
materialises **at standby**, not in the Up Next slot and not at proposal.

**There is no role called the ARBITER.**  The word means today's `app/director.go`, and 0.14.0 ruled
that it **dissolves**: *"its decision half — who gets the air, what suspends what — becomes state inside
the Director's step … `app/director.go` dissolves."*  **That is still unfinished**, and the wrong duck
decision was made in the half that never moved.

---

## The audio dispositions — READ BEFORE MOVING ANY AUDIO

**`multi-voice-support/01-objectives/director-charter.md:172-173`** is a disposition sheet.  Each row is
Absorb, Instruct, or Leave alone, and it is written to be **honoured or overturned item by item.**

| Thing | Disposition |
|---|---|
| `radioDeck.tune`, `SetMode`, `followMount`, **`startSynth`** | **Instruct** — it stays where it is |
| `engine.Suppress` / `Restore` / `giveWayLocked` | **Instruct** — *"the Director says an alert is on air; the engine decides how to yield from the source kind.  **Already the right shape**"* |

**BD-9** (`multi-voice-support/04-development/director-build-log.md:809`):

> *"A report's speak is **the engine Source adapter** — which **IS** the main-track absorb."*

**The end state, ruled and deferred** (`director-build-log.md:1086`):

> *"**The Director owns all ducking** — the arbiter stops dipping entirely.  **Rejected FOR NOW, and it
> is the end state.** … It arrives when everything reads through the Director (T3.2+)."*

---

## The give-way rule, and the design that was already rejected

**`engine.giveWayLocked` decides, from the SOURCE KIND, re-read every 50 ms:**

- a **relay** DIPS to 0.15 — live radio, and a paused relay resumes into audio that is minutes stale;
- a **rendered report** HOLDS — it has nowhere to be, so dipping loses its words for good.

**A give-way keyed on which RAIL is chosen was rejected, in a comment that records what it cost**
(`app/radio.go:666-667`):

> *"Asking the deck's mode here fixed an answer the audio could outlive."*

**`app/director.go`'s header, since 0.14.0:**

> *"The broadcast itself (relay or synth) is not a narration: it is what gets ducked."*

---

## What happens when an alert arrives while the main track is reading

**The read PAUSES and resumes mid-sentence.  The BED dips.**  Ruled three times:

- **D-24** (HUM LEAD, 2026-09-09): *"With the main rotation — we can PAUSE the read, let the alert rail
  drain, insert a transition read, then resume the read at normal volume.  The duck is important for the
  stream because that relay is fundamentally out of our control."*
- **DR-16** and **S-4**: a read resumes mid-sentence, optionally behind a transition card; a live bed
  ducks and normalises back up.
- **The alert never waits.**  The rail *"has to drain"* — read in order and in full, or cancelled by an
  operator.  Nothing else empties it.

---

## The rails, and the vocabulary trap

**Three LANES in the product model; TWO tracks in the type system.**  `platform/lineup/lineup.go:9-12`:

> *"The bed … is deliberately not here: it is a selectable resource the Director may cut over to, not a
> queue of cards, and **modelling it as a third track would invite something to be scheduled onto it**."*

**The bed carries the SYNTH cycle too, not only the relay** — `live` distinguishes them, because a relay
dwells and a rendered cycle ends by itself.  **Today the synthesised rotation IS the bed.**

**Main and Bed are mutually exclusive** by product rule (D-11, FR-4.2) and by the engine having one
source.  **Nothing in `platform/lineup` models it** — a paused main track is *"a fourth condition the
`Power` enum does not yet express"* (FR-5.6).  That is P5's.

---

## ONE DECK, ONE AIR — and this is the question that keeps getting re-litigated

**Both surfaces run through the SAME `radioDeck`, on purpose.**  HUM LEAD, 2026-09-12:

> "Both Observer and Broadcaster do synth-based reads and relay streams, and so rather than make
> copies of code, and prevent the possibilities of 2 decks running at once, we made both modes run
> through the same deck, which would have its rotation arbitrated by the Director (in both modes — on
> purpose), and then gated from the air by MasterControl."

**Broadcaster is an EXTENSION of the product, not a standalone.**  All of 0.14.0 and 0.15.0 was spent
making the shared things genuinely shared — "one canonical way to do a thing" — because the business
problem is that Observer *can* broadcast a station if needed but is not suited to *operate* one.

**So the model is settled, and it is not a design question:**

| | |
|---|---|
| **one deck** | both modes' synth reads and relay streams |
| **the Director** | arbitrates the rotation in BOTH modes |
| **MasterControl** | declares the air, and gates the deck from it |

- `advances(MainTrack)` = `power == Running && air == AirProgramme && !bed.carries`
- `advancesMonitor()`   = `monitor && air == AirMonitor && !bed.carries`
- **mutually exclusive by construction** — `TestOnlyOneProgrammeCanEverAdvance` states it
- the ALERT RAIL is exempt from both; its FENCE follows the surface (D-73)

**In Broadcaster mode the Broadcaster ALWAYS has the air** (HUM LEAD, 2026-09-12): "nothing from
Observer should ever be able to re-tune, take over, or 'sneak under' to get on the air."  The deck
already has the predicate for this — `radioDeck.monitorHasTheAir()`, which ASKS THE EFFECTOR because a
flag of the deck's own would be a second carrier.

**WHY THIS PARAGRAPH EXISTS.**  A session called the shared deck "awkward" and proposed splitting it,
having rebuilt the model from `app/radio.go` instead of from `rulings-d74.md`.  That has now happened
more than once, across sessions, and the HUM LEAD has had to correct it each time.  **The split is not
an open question.**  If the deck looks wrong to you, the missing reading is `rulings-d74.md` and
`role-model.md`, not a new proposal.

---

## STOP is not STANDBY

|  | Observer STOP | Broadcaster STANDBY |
|---|---|---|
| the programme | stops | stops |
| **hazards** | **still read** | **do not read** |
| fresh alerts | consumed | **stay new**, read on resume; stale ones expire |

**And "ON AIR" never means the antenna is radiating** (FR-5.5).  Watchpost has no radio path: it
produces audio, and a human patches it into a transmitter.

---

## The process rules this release is under

`04-development/p3-flip-postmortem.md` carries P-1..P-10 in full.  The four that would have prevented
the revert:

- **P-1** — a seam a test cannot drive is **not a covered seam**.  A stubbed seam is a RED gate.
- **P-6** — **RUN IT** once before asking for a ruling on a shape.  Reading a call chain is not a spike,
  and a chain traced in one direction is not a measurement.
- **P-8** — **read the release that built the seam before wiring it**, disposition sheet included.
- **P-9** — **read the paragraph the cursor is in** before adding a member to a closed set.

---

## The UX rulings, and they are NOT re-derivable from the code

**`02-analysis/rulings-d37-d40.md` is a re-required read too.**  These are HUM LEAD rulings made in
conversation while reviewing the card mocks, and nothing in the tree implies them — a session that
rebuilt its context from the code would re-derive the WRONG answer for every one.

| | The ruling | The trap it keeps us out of |
|---|---|---|
| **D-37** | the discard pile is a **modal**, `[shift+U]`, patterned on the `[w]` window — and reached **from the base UI only** | `Dashboard.modal` is a SINGLE VALUE.  Opening it from a card modal needs a return path, which is the rewiring |
| **D-38** | **four** elevations: base → priority overlay → detail/action → confirm/destructive | three already exist.  What does NOT is an INTERACTIVE overlay on the body, which is what the priority track is |
| **D-39** | the card detail modal is a **centred** overlay like the location-details one — scroll, sticky controls | a report grows long once the Composer merges real data at Up Next |
| **D-40** | **the Producer proposes — several, cheap, name-only — and the DIRECTOR chooses** | nothing reads track depth today, so the lineup holds ~1 card while the mock draws 10 |
| **D-41** | a card the Operator CHOSE is `FromOperator`, and that origin **outranks** | scope is the CHOICE ranking; where a RESTORED card lands is still unruled |
| **D-42** | a transition is **never** `FromOperator` — it is the Director's one additive act | held by accident before it was stated: a different rule refused it for an unrelated reason |
| **D-43** | transitions **bookend** their card and **move with it** on promote/quash | **JOIN-owned**, ruled 2026-09-10: derived from the sequence, re-evaluated on every change.  Free, because a transition's words are fixed at proposal |
| **D-44** | the **line-up** is a PROJECTION of the **schedule** — ruled 2026-09-05, built 2026-09-10 | `Cards()` = the schedule; `Projection()` = what the operator sees, numbers and addresses.  `Moved` carries LINE-UP numbers |
| **D-45** | ON AIR is locked; only GO TO STANDBY or catastrophe changes it | closes an item the 2026-09-05 log left open *"when operator editing arrives"*.  Every caller conforms; the TYPE still permits it |
| **D-46** | `[shift+N]` makes a **request order**, not a card — the Producer turns it into a proposal | keeps the role split at the point most likely to break.  Carries a KIND (no slot for it yet — D-31) and a target slot in LINE-UP space |
| **D-47** | origin is a **term** in the ranking, not a trump; insistence is the escalation | reconciles D-41 with the cadence example, and makes D-40's choice modal a TRIGGERED thing |
| **D-48** | the Director remembers **one timestamp per slot**, and nothing else | read history ruled out by name.  `WeighLastRead` is the operator's switch, and every degraded path answers the same for everything |
| **D-49** | a transition is a property of the **KIND**, not the origin — `announced` / `handsBack` rows | **MVS-D-80 ruled the trigger in 2026-09-05 and the scripts are pinned.**  Does NOT fire location-to-location.  The words come from the script library through Settings, never from the Director |
| **D-50** | breakpoints **100 / 120 / 150**, floor 25 rows; below 100 the console REFUSES to draw | replaces `platform/term`'s 40/60/80/120 — broadcaster was its only caller.  `bcMinRows` stays 44 (MEASURED); the ruling's "25" was inside an example of the message |
| **D-51** | above 150 is a **right rail**, a LATER release — a SEAM is owed now, not a layout | `laneWidth()` is that seam: the lane's width comes from ONE place, so a right rail is a smaller number, not a second renderer |
| **D-53** | **product first, optimisation second** — a working baseline is the reason | measure it, record it, surface the number, then KEEP BUILDING.  Optimising before behaviour settles has nothing to prove it did not change behaviour |
| **D-54** | the Producer is wired through the **`Publish` executor's return**, no new effect | the chain is self-limiting by the DEPTH: a full line-up admits nothing, so `onOffered` emits nothing, so there is no next publish |

**D-40's operator flow, ruled in full:** card in the main track → `[#]` → detail modal → `[D]` Drop →
confirm ARE YOU SURE → `[enter]` → both modals close → the card leaves → **everything below moves up
one** → **slot [9] gets a new card from proposals**.  That last arrow is the gap the ruling closes.

**And the mock is measured at 130 cells, not 65.**  The priority track is an OVERLAY on a full-width
main track — normally invisible, taking over only while it has alerts.  A card drawn at the occluded
width is half a card.

---

## The reading list, ranked

1. `multi-voice-support/04-development/director-build-log.md` — §§ 783-830, 1063-1130, 2137-2230.  **The
   only document that traced the bed's behaviour by RUNNING cards through the executors.**
2. `multi-voice-support/01-objectives/director-charter.md` — §§ 148-181.  The two-paths fact and the
   disposition sheet.
3. `multi-voice-support/03-architecture-design/director-architecture.md` — §§ 19-44, 230-278.  The closed
   effect set and the ruling that `app/director.go` dissolves.
4. `multi-voice-support/03-architecture-design/role-model.md` — the five roles, in full.
5. `multi-voice-support/01-objectives/lineup-model.md` — the tracks, S-4, S-5, T-1..T-5.
6. `0.15.0-pre-broadcaster-ui-improvements/02-analysis/completion-spike.md` — **the Director is open-loop
   about audio**; `OnClipSpent` is a station condition, not a schedule event.
7. `multi-voice-support/01-objectives/director-requirements.md` — DR-3, DR-15, DR-16, DR-24, PD-3.
8. `multi-voice-support/08-reports/debrief.md` — *"the wire is not pinned"*, nine instances, and the
   **producer/consumer completeness check that is still unbuilt.**
9. This release's own `01-objectives/` — brief, requirements, glossary, risk register (**RS-3** and
   **F-34** are live).
