# The Lineup — the Director's schedule as a first-class object

**Source: HUM LEAD, 2026-09-01, answering L-1 … L-8 at DISCOVER.** Companion to
`director-charter.md` (what the Director is for) and `read-order-design.md` (the read order, one rule
that lives on this object). **Cold start? `README-director-handoff.md` first.**

**Status: SPECIFIED AT DISCOVER, NOT BUILT.**

The reaffirmed requirement that opened the RCC:

> The **LINEUP** (schedule) needs to be its own first-class concept and object that the Director
> owns, but other parts of the system related to radio / reading / saying / presenting need to
> reference in order to determine what things to pre-load / pre-build / sync.

## The tracks

Three, and they are not peers.

| Track | What it holds | Rules |
|---|---|---|
| **Main track** | The reads — location reports, event reads. The bulk of what the station broadcasts. | Cards are queued, **assigned a cast**, carry **individual settings**, and can be **moved up, moved down, or cancelled**. This is the track the rest of the broadcaster systems look at and reference. |
| **Alert rail** | Takeovers. | A priority lane that is **always there and usually empty**. Once it holds anything, those items are **read first, in order, and in full — it has to drain**. It drains one of two ways: the items are read, or a **human operator cancels** one. Nothing else empties it. |
| **The bed** | The live NOAA relays. | A **selectable bed**, always available, that the Director may **cut over to** on the human's configuration or command. There are multiple relays to choose from, so it is a selectable resource rather than a single fallback. |

**Resumption.** When the alert rail drains, the Director returns to either the main track or the
live bed — *"depending on the Director's discretion / configuration / previous state prior to the
alert."* The precise rule is open (S-4).

## Order, not clock

The Lineup is **mostly order**. It does not carry air times.

An individual card **may** carry its own duration constraint — the HUM LEAD's example: *"I'm doing a
Watchpost credits read — I can never go over :30s."* So duration is a **per-card cap**, optional,
not a schedule slot.

## The two proposers

Nothing writes the Lineup but the Director. Two things propose into it:

1. **Watchpost Observer** — the system that does the work today: fetches data, combines data +
   scripts into read reports, and understands every kind of event and alert.
2. **The Human Operator** — *"Read for me the location report at Oceanside CA"*, *"Play all reports
   rotating across my watchlist."*

## The Director's two functions

**1. It executes the will of the human operator.** It enforces every preference, filter, repeat rule
and other setting **as the human configured them**. When multiple things are proposed or queued at
once, the Director **pre-screens** them against those rules. The rules are the standing instruction;
the Director does not invent an arrangement, it applies one.

**2. It synchronises the watching and proposing systems.** Because the Director owns the lineup, the
other actors must at minimum tell it **what *kinds* of things they have available**, so it can build
and maintain a schedule. The Director therefore needs *some* understanding of each kind — but the
weight is on the other side:

> **The other systems need to understand what a scheduled slot in the lineup means FOR THEM.**

The HUM LEAD's illustration:

> The API layer sees a location report for 92057 is next — *"I need to make sure I have the most
> up-to-date data possible; time to schedule a focused lookup against our APIs."*

> It's like a real radio station: the Director is aware of all the kinds of things the station can
> broadcast, but does not necessarily know the ins and outs of HOW those things get produced or put
> together.

**The architectural consequence.** The Lineup is not a private queue the Director drains — it is a
**published, readable model that producer systems subscribe to and react to**. The Director owns
every write; everyone else reads what is coming and prepares their own half. That is the seam
D-C-6 asked for.

## Persistence — the split that matters

**The configuration that BUILDS the lineup persists. The lineup itself does not.**

The Lineup is an **ephemeral queue** of reads and slots, and the data inside those slots is whatever
Watchpost Observer has collected live. A slot-by-slot lineup does not carry from yesterday to today,
except where the items are simply evergreen rotations.

**In-the-moment operator edits do not persist.** *"Push this read down by two slots"* is a
moment-to-moment act. If a human wants that kind of change to stick, that is a signal that the
**Director's composition configuration** should change instead — the HUM LEAD's example: *"I now want
all location reports to end with Fire, not Seismic."*

This gives the design a clean rule: **anything a human wants to persist is a composition rule, not a
lineup edit.** Whether intra-report segment order is in the Director's scope is open (S-7).

## Broadcaster: ON AIR / STANDBY

Broadcaster is not an "official" radio station — it is meant to make it easy for someone to stand up
their own weather **FRS / GMRS / ham** station. It should be intuitive, but it does **not** need
dead-air controls. There are two states:

- **ON AIR** — broadcasting actively.
- **STANDBY** — not broadcasting.

**Sign-off is a script, not a control.** If the operator wants a sign-off tail before ceasing
operations, it is made available as a script and is **auto-inserted as the last queued item** before
the station drops to STANDBY.

### How the main track fills in normal operation

Two modes, and the second is the **HUM LEAD's preferred default**:

1. **A location loop** — similar to, or powered by, the Observer's watchlist.
2. **Station geometry (preferred).** The operator states:
   > *"My station call-sign is `BRT123`, I am located at `<GPS coords>`, I want to serve all
   > observational data / reports / etc. in a `<distance>`-mile radius."*

   Service radius is generally informed by wattage / broadcast power. From that geometry the station
   **auto-builds its location list** for automatic data collection, compilation and broadcast.

**Forward concept: Station Identity & Service Area** — callsign, origin coordinates, service radius.
It is a Broadcaster concept, but it decides that the main track's filler must be **pluggable**:
watchlist-driven or radius-driven, chosen by configuration.

## Naming

`app/director.go`'s `director` is today a voice arbiter, and the name now belongs to the coordinator
above it. The HUM LEAD accepted a rename **on one condition**: that some system is *directly
responsible for ensuring the radio output and the news ticker are properly synced for takeover
events.* Proposal pending ratification: rename it **`mastercontrol`** — in a real station, master
control is exactly the room that makes what goes out, audio and visual, match the log and switch on
cue — and give it the ticker cue as an explicit responsibility rather than a second implementation.

## Observer is "Broadcaster for Me" — and the harder problem

**HUM LEAD, 2026-09-01, answering S-1 … S-8.**

Observer's radio *is* Broadcaster with an audience of one. The human selects the locations, sets the
preferences, and issues the play / pause / stop commands; everything automatic in Observer exists to
collect, normalise and store data so the TUI can serve those requests.

> In Observer the watchlist and the settings are **the human's own proto-lineup**: *"I care about
> these locations. Give me weather and observational data about them … and when I want you to, read
> it to me, or give me the closest NOAA relays."*

**Observer is in some ways the harder initial problem**, because the data-table TUI has **no
geographic limit** beyond the wired APIs. Broadcaster is *narrower*: stating a location and a service
radius constrains how much data is collected, because the audience is whoever can pick the station up
on FRS / GMRS / ham — not a national one. The same underlying concepts serve both.

**The Broadcaster difference is a second user.** Observer has one person who is Operator *and*
Director *and* audience. Broadcaster has the **Operator**, who needs the UI to know what the station
is about to broadcast, and the **radio audience**, whose entire experience is the audio. The Operator
uses the UI to craft the listeners' experience — which is why Broadcaster *needs* the lineup, and why
Observer's is powered by configs and the watchlist instead.

### Scope for 0.14.0

Build the foundation, and make Observer's radio deck genuinely run on the Lineup and its concepts.

**Station identity, service area and ON AIR / STANDBY are Broadcaster extensions.** Whether any of
them must be **stubbed or minimally built now** — because the seam is load-bearing for Observer's
radio under the new architecture — is **an open PLAN decision**. The HUM LEAD has asked for a *clear
recommendation with the reasoning*, and will rule on it there.

### The answers, point by point

| # | Ruling |
|---|---|
| **S-2** | The human is the Operator. Observer is inert under a second `[space]` today because the human is *both* Operator and Director — the system serves data, and they act on it. Queuing is the Broadcaster-era concern, where a second user exists. |
| **S-3** | **A diverted alert is NOT discharged, and should not be.** The alert rail is powered by the human's own alert preferences — "show me all of them", "mute disaster tones", "nothing past 20 miles" — so the **settings are the opt-in**, and the pre-screen happens at *admission*. In Broadcaster it differs, because the Operator is not the intended consumer of the audio: there they may tell master control *"this is the third time that warning has been issued — don't broadcast it again, move to the location details instead."* |
| **S-4** | **A read resumes mid-sentence.** A **transition read** may be introduced with it — *"we now return to our regularly scheduled report in progress."* A **live stream** ducks and then normalises back up. This mirrors the per-mode give-way already adopted in the engine. No staleness rule. |
| **S-5** | One-ahead rendering is acceptable, and **line-by-line with no pre-render at all may be enough** — what the code does today. Separately: Broadcaster should **not** offer the inter-report transitions Observer allows, which would be jarring to a listening audience. |
| **S-6** | The card's fields, as the Operator must see them (below). |
| **S-7** | **The Composer owns report contents.** The general principle is accepted. Open follow-on: how the Operator gets *some* control over helping the Composer compose. |
| **S-8** | **Fire-and-trust with a post-hoc confirmation** that the cue actually happened, rather than a blocking guarantee. The "3 … 2 … 1" was illustrating that the Director's job is coordinating a synchronised start, not specifying a handshake. Open to suggestions. |

### The card, as the Operator sees it (S-6)

> The human Operator needs to see all relevant data for what's going to be on air.

| Field | Values |
|---|---|
| `id` | — |
| **Slot Type** | Location Report · Marine Report · Special Weather Statement · … |
| **Read by** | an available voice profile **on this machine**, or `N/A` |
| **Max Duration** | `Full length` by default, or a configurable number of seconds **for this card** |
| **Text content of the read** | the script itself |

**Basic controls:** `DROP` · `DELAY` (push down the queue) · `PROMOTE` (move up the queue) · possibly
others.

Two things follow. **Read by** is resolved against what *this station* has, confirming the charter's
station-local voices. And **the text is on the card**, which means a card carries its composed script
— see T-4 for when that composition happens.

## The tracks in Observer, and the ticker's contract

**HUM LEAD, 2026-09-01, answering T-1 … T-5.**

### T-2 — the main track IS built at 0.14.0, and it is invisible

The watchlist rotation becomes main-track cards. From the listener's side nothing changes:

> He just hears the audio, it corresponds to his watchlist, and alerts take over when they take
> over — and if there are a bunch of them, Observer tells him where to go to hear and read the rest.

This is D-C-7's **absorb** for `armDwell` / `advanceQueue`: the timers in `radioDeck` stop being the
schedule and the Lineup becomes it.

### T-1 — the overflow remedy differs by edition

**Observer:** *"Play the Max, and if there's more left over we direct the listener to `[w]`, where
they can hear each detailed read at their own discretion — they see a table of ALL the alert
events."* The `[w]` window is the remedy, and Observer's rail is **not** inspectable.

**Broadcaster:** the listeners have no such control, so it falls to the Operator — either a standing
rule that master control executes, or manual intervention. Broadcaster's rail **is** inspectable and
queued in the UI, which is most of the difference.

**Broadcaster is also naturally bounded.** Its auto distance filter constrains the volume, and a
candidate constraint was floated: *Broadcaster is designed for FRS / GMRS / ham, so cap the service
radius at ~150 miles.* A 40-alert backlog is then the exception, not the common case. **Candidate,
not yet ruled.**

Whether a diverted alert remains queued for a later takeover or is discharged to `[w]` is the one
question still open on this — see the RCC's rail-persistence fork.

### T-3 — transitions: both, split by where they sit

> **Intercard transitions are cards themselves. Intra-card transitions are utterances controlled by
> the Composer.**

This lands exactly on the S-7 boundary: the Director owns *arrangement*, the Composer owns *content*.
A transition between two cards belongs to neither card and needs its own slot, where the Operator can
see and drop it. A handoff inside a report is the Composer's, which already owns report contents.

It also introduces a **third card origin**: the Director itself, alongside Observer and the Operator.

### T-4 — a card's text materialises at standby

Approved as the first approach, to be refined once real reads exist: the card carries a **headline and
shape** from creation (slot type, subject, estimated length), and its **full text materialises when
the card reaches standby**. The Operator's queue shows text for the cards near the top.

### T-5 — the news ticker keeps its own job

The news ticker is **its own thing**, and the sync contract is narrow:

> It syncs with a **centred callout** of the alert being read, up to the Max, and then resumes its
> normal function: rotating between notable event categories, each holding its own marquee of events
> that animate, onboarding and falling off as they are declared and expire.
>
> The ticker only cares about the lineup when it says *"a notable event has now been declared, so
> when I call it out, you need to show it"* — which is 0.13.0's behaviour, but largely by
> **coincidental agreement**.

So the rotation is **not** lineup-driven. The redesign's job is to make the existing behaviour a
**contract** instead of a coincidence — G-11 answered.

### Carried to PLAN

- **An interruption that outlasts the report's validity.** Resumption is mid-sentence with no
  staleness rule, but where the alert rail drains for longer than the report stays valid, it may be
  better for the transition read to **acknowledge the report has expired** and move to the next
  scheduled one. A PLAN item to explore.
- **Standby vs. line-by-line.** The charter's analog makes standby first-class ("get that report
  ready … so we can immediately cut over"); S-5 says line-by-line with no pre-render may be enough.
  These disagree about an audible cutover gap. PLAN decides, with reasoning.
- **Stub-or-build** for station identity, service area and ON AIR / STANDBY (S-1).
- **How the Operator helps the Composer compose** (S-7).

## Closed without a ruling

**G-9** — `[s]` is Settings, lowercase; `[S]` is Watchpost Status. Verified in
`modes/tty/dashboard.go:274`. The design doc's "[S] Settings" was shorthand.

## The burst is a snapshot, and the pointer is what differs

**HUM LEAD, 2026-09-01. This closes the rail-persistence fork.**

**A diverted alert is discharged from the audio path.** The burst reads the Max and then tells the
listener where the rest are. This is 0.13.0's behaviour and it stays.

**The Lineup does not fork by edition — one card's destination does.**

| Edition | The pointer |
|---|---|
| **Observer** | `[w]` — the listener has the window, with a table of ALL the alert events, and can deep-dive and read on demand. |
| **Broadcaster** | A website. *"Telling a listening consumer five miles away to press `[w]` in Watchpost doesn't make sense."* |

**The Broadcaster model is NOAA Weather Radio's**, and the HUM LEAD scripted it:

> *"The following alerts have been issued by `<ALL PROVIDERS>`"* → **read 5 headlines** → *"For more
> details about these and `<REMAINDER>` other alerts, please visit `<site>`, `<site>`, and
> `<site>`."*

The destination is a **list**, joined the way a person says one — the same rule `burstAgencies`
already applies to provider names.

The division of labour, in the HUM LEAD's words:

- **Observer** — *totally driven by settings and pre-defined limits*, because manual deep-dive and
  on-demand reads exist.
- **Broadcaster** — *configurable options with manual in-queue overrides* for edge-case bursts,
  inside a limited service radius.

## Max and radius

**G-5 — Max is ONE number for the whole burst, across all categories.** *"Max reads is ALL CATEGORIES
in the burst."* Read order is about **sorting** — *"most severe alerts first when I get a burst"* —
not about per-category budgets.

**G-6 — structural cards do not count against Max.** Proposed with T-3 and consistent with the
scripted burst above: the head, the divert notice, transitions and tails are the Director's
arrangement; Max counts **alert reads**.

**G-7 — the radius is a HARD FENCE**, a filter and constraint on what reaches the lineup at all —
with a general exception for **significant disasters, because such a disaster has proximal effects**.
The HUM LEAD's illustration:

> *"I care about an M7.5 quake in Los Angeles (~120 mi away). I don't think I'd notice an M9.0 in
> Fairbanks, Alaska — and the only time I might is if there were a tsunami warning for San Diego,
> which a local alert within my service radius would pick up."*

Three things follow. The exception is **scaled by significance, not unbounded** — Fairbanks is the
counter-example. **Cascades need no modelling**: a derived hazard arrives as its own local alert.
And because the fence governs *entry*, the "close" threshold governs only the *ordering* of what got
in — where the fence is tighter than the threshold, the close/remaining split simply collapses, which
is harmless.

## The read order, ratified

**HUM LEAD, 2026-09-01, answering R-1 … R-4. This supersedes the ten-rung scale in
`read-order-design.md`, which predates Emergency Orders and Forecasts and omitted Marine.**

### One ordering, one Max (R-1)

The per-category `Max` row is **gone**. ALERTS - READ ORDER carries **one ordering** — category
priority, and the sort dimensions within each — and **one global Max** for the burst. Read order is
about **sorting**, not per-category budgets.

### Emergency Orders (R-2)

| Bound | Applies? |
|---|---|
| **Max** | **They spend the budget first, and may overrun it.** *"We have a Max of 5, which means we should live within that budget — the only time that's not true is when we have more than 5 emergency orders."* Emergency orders lead the burst; the remaining budget is filled by read order. When they alone exceed Max, all of them are read and everything else is diverted. **Clarified at Q-1** — an earlier reading had them consuming the whole burst regardless of count, which was wrong. |
| **Service radius** | **NOT exempt.** *"I don't care about a fire evac order in Los Angeles if I'm in San Diego — but the AIR QUALITY HAZARD STATEMENT for San Diego as a result of that fire is more relevant and important to me."* |

The second half is the **cascade principle again**: the consequence that reaches you arrives as its
own local alert, inside your fence. The design never has to model the link.

### Disasters vs. Warnings — the rule is conditional on the fence (R-2)

> **Disasters outrank Warnings when in service radius** — the Broadcaster use case.
> **RECENT Disasters outrank Warnings without a service radius** — the Observer use case:
> *"I care more about today's severe thunderstorm warning than a landslide that happened four days
> ago."*

Read against the existing ALERTS - EVENTS setting, where `0` means All: **when a radius fence is in
force, Disasters outrank Warnings unconditionally; with no fence, a Disaster must also be fresh to
outrank a Warning**, and a stale one sorts down into the remaining band.

The freshness window is an open number (RCC R-5). It is a genuine ordering knob, and **Recency** is
already one of the four dimensions the settings group offers.

### Forecasts

**Never spoken as an alert** — the same reason they have no ticker lane. The marquee and the burst
are both for what is *happening*.

### The default order

Rungs 2 … 13 sort within the burst; Emergency Orders sits above the Max entirely.

| Rung | Bucket | | Rung | Bucket |
|---|---|---|---|---|
| — | **Emergency Orders** — in-radius, exempt from Max | | 8 | Remaining Disasters † |
| 2 | Close Disasters | | 9 | Remaining Warnings |
| 3 | Close Warnings | | 10 | Remaining Watches |
| 4 | Close Watches | | 11 | Remaining Advisories |
| 5 | Close Advisories | | 12 | Remaining Spec. Statements |
| 6 | Close Spec. Statements | | 13 | Remaining Marine |
| 7 | Close Marine | | — | **Forecasts — never read** |

† subject to the significance fence, and to the freshness rule where no radius is set.

**Design note.** This is a **third** per-category ordering, beside tab order and ticker rotation. All
three are legitimately different jobs. Read order belongs in `category.Spec` as a named field beside
`Rotation`, so that nobody later "harmonises" the three into one and silently changes what is spoken.

### The significance fence is in scope for 0.14.0 (R-3)

> *"In scope now — we'll need it later, and it will improve Observer in 0.14.0 anyway."*

### The default location (R-4)

Bonsall, CA is **shown in Settings as the Default**, never used silently. Settings **auto-opens** on
`watchpost setup` or on first run, and the listener is told and prompted to set their own default
location.

**First run stays "no config file exists"** (R-6) — today's `config.FirstRun` semantics. A
release-versioned marker was considered and declined: *"no config file is also good versus 'new
version'."* It would have re-prompted a settled listener on every upgrade.

**So the mechanism already exists.** `cmd/watchpost/root.go:47` is the `setup` subcommand;
`app/dashboard.go:53` opens Setup when `cfg.FirstRun || len(refs) == 0`; `config.FirstRun` is set at
`platform/config/config.go:303`. **All that is new is Bonsall compiled in as the origin and shown as
the Default.**

### G-10 — the wall-clock backstop — RETIRED

**RULED, HUM LEAD 2026-09-01 (red-team PL-11).** Burst duration is bounded by **the number of events,
not the length of the reads**.

The reason is stronger than the one first proposed. The draft argued a time bound was *redundant*
beside the count. The HUM LEAD's reason is that it is **unsound**:

> *"Different voices on different machines may read at different rates, so a wall-clock time bound
> seems flakey in terms of what actually gets read."*

A time bound would make **which events a listener hears depend on how fast their machine's voice
speaks** — a slower voice would silently lose the tail of the burst. That is the same class of defect
as the ~8.14 s starvation edge MVS-D-56 exists to remove, reintroduced through the host rather than
through the sort.

**Broadcaster is flagged as where duration may still need its own answer** — *"less concerned for
Observer, more important for Broadcaster"* — where the audience cannot press `[w]`.

## A burst that arrives while the rail is draining

**HUM LEAD, 2026-09-01, answering red-team finding RT-1** — the old defect's exact home, and the one
case none of the earlier rulings covered.

**Observer: the rules are hard and fast**, because `[w]` is always available. Each burst is evaluated
on its own — read the Max, divert the rest to the window — and nothing negotiates.

**Broadcaster** has no `[w]`, so it needs conditional bounding:

| Condition | Behaviour |
|---|---|
| **Emergency Orders** in the new burst | **They follow immediately. No skipping, no negotiation.** |
| New burst of **4 or fewer** | Let them ride — read them. |
| New burst of **5 or more** | Do not read them. Say: *"During this time `<number>` new alerts have also been issued — go to `<website>` for details."* |

The boundary is `≤ 4 ride / ≥ 5 divert`, confirmed at Q-2 — the original wording left 4 and 5 in a gap.

**The counter resets.** After a second burst is handled, the logic resets: a third burst is evaluated
as "Burst 1" again.

### The circuit breaker — BURST PANIC COOLDOWN

For bursts upon bursts upon bursts, the Operator can intervene directly, **and** a cooldown is under
consideration, explicitly modelled on a stock-market circuit breaker — where trading halts when the
market moves past a threshold inside a window:

> **Broadcast NO non-Emergency-Order alert** for **1 minute or 1 main-track card, whichever is
> shorter**, and **purge the rail** for that period.

Emergency Orders are never subject to it. **Candidate, not ruled** — scope is an open question.

## Failures

**HUM LEAD, 2026-09-01, answering RT-2 / D-C-4, in part.**

- **Broadcaster** — failures must be accounted for in the UI.
- **Observer** — *"likely just some kind of alert or error modal."*

This settles how a failure is **surfaced**. It does not yet settle how one is **routed** — one
escalation channel into the Director or per-producer handling, and whether the Director re-plans or
only re-routes. See the RCC's open items.

## What this leaves open

**R-5 — the Disaster freshness window is 24 HOURS. RATIFIED, HUM LEAD 2026-09-01.** It matches
*"today's thunderstorm warning versus a landslide four days ago"*, and sits well inside the 7-day
USGS window the feed already keeps — so a stale disaster still appears on the tape and in `[w]`, it
merely stops outranking live weather.

**R-6 also confirms the upgrade case**: *"if we have a config file that has a location stored from a
previous version, we don't need to prompt the user again."* That is exactly today's `FirstRun`
semantics, so nothing new is built.

D-C-1 … D-C-6 are answered across the L-, S-, T- and R- rulings and are closed out in the DISCOVER
report.
