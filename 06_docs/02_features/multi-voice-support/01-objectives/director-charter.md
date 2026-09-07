# The Station Director — charter

**Source: HUM LEAD, 2026-09-01.** Effective 0.14.0 onward. **Intent, not prescription**: the
architecture sketched here is illustrative. It describes what the Director is *for*, so a design can
be judged against it.

**Cold start? Read `README-director-handoff.md` first** — it orders these documents and lists the
approaches already tried and reverted.

Companion to `read-order-design.md`, which specifies one of the Director's responsibilities — the
read lineup. This document is the wider charter that lineup sits inside.

## It is an ASSISTANT Director

The code calls it the Station Director. It operates as the **Assistant** Director: it does the work
of running the station efficiently, and either **surfaces a decision** to the human or **acts on
decisions and preferences they have already given**.

In the Observer experience the human is the listener, and the Director acts on their stored settings.
In the planned Broadcaster experience the human is the **Station Owner/Operator**, and the Director
serves them the way an assistant director serves a producer.

This is the sentence to hold on to when the design gets hard: *the Director never decides what is
the human's to decide.* It prepares, it recommends, it executes what was already ruled.

## What it is

The central **coordinator-orchestrator** for everything in Watchpost concerning the real-time
presentation of audio programming, plus the limited UI that must move with it — the **News and
Alerts Ticker** in the Observer window.

The Broadcaster UI is neither designed nor built. In that scope the Director will additionally carry
station operations, scheduling, and casting for event reads.

## What it does today

- **Understands every audio source** — the live NOAA relay, and the synth layer (Say, Piper, …).
- **Owns and coordinates the schedule** — the "playlist": what is played or read, and in what order.
- **Controls interrupts and takeovers** — it may change the playlist in real time to pause, stop or
  duck normal programming so an urgent event reaches both the radio and the ticker.
- **Instructs the News Ticker** so it stays in sync with the notable events being read — Warnings,
  Watches, Advisories, Special Statements, Disasters, Marine.

## What it directs, and what it does not do itself

The Director is **not** responsible for the low-level work. It does not assemble observational data
into a script, and it does not apply pronunciation rules. It **instructs** the machinery that does.

**It knows:**

- every kind of report the station has or could have;
- every other audio input (here, NOAA relays; in a real station, ad spots and the rest);
- what is playing right now, if anything;
- the available voices — **which differ from station to station**, machine to machine;
- that a schedule/lineup/playlist exists and is its to run;
- that other parts of the station must be coordinated and in sync with a takeover;
- and, for each of them, what to say so it does its job at the right moment.

**It does not care:**

- how the voices work — that is Piper's, Say's, whatever's;
- how the general UI works, only *that* it does;
- where the weather data comes from;
- how the ticker is rendered, **only that it shows what it shows when it needs to show it.**

> The Director **directs** and **coordinates**. It is not in the business of doing work that is not
> coordination. Those roles belong to whatever does them best, and their job is to pass the right
> messages and information up so the Director can direct.

## The analog, as the HUM LEAD wrote it

This is the clearest statement of the intended behaviour, and the design should be readable against
it line by line.

> **Director:** "Alright, we're due to read a location weather report next. Lessac is going to read
> that report for Oceanside, CA 92057. Get that report ready and on standby so we can immediately cut
> over to it once the current report, read by Daniel for Rancho Peñasquitos, is done."
>
> *(Time passes. Lessac is now reading a report.)*
>
> **Director:** "We have a breaking news alert. System-voice, I need you to cut in with the report
> headline. News ticker — get ready to jump to the alert headline in 3 … 2 … 1 …"
>
> **System voice:** "The following alerts have been declared by the National Weather Service…"
> **News ticker:** shows each headline in its proper lane, in sync with the announcement.

And on failure:

> "This radio relay is down / not working." → **Director:** "Cut over to the Location Report.
> Lessac — you're up."

### What that analog establishes

Four things the current code does not have, drawn straight from the dialogue:

1. **Standby is first-class.** The next report is *prepared while the current one plays*, so the
   cutover is immediate rather than a render-then-speak gap. (`director.go` already separates
   `prepare` from `deliver` for exactly this reason on one path — the charter generalises it.)
2. **The ticker is INSTRUCTED, not inferring.** "Get ready to jump … in 3, 2, 1" is a cue with lead
   time, not the ticker noticing that an alert arrived. Today the band and the voice reach agreement
   independently, which is the coincidence MVS-D-56 named.
3. **Failures flow UP.** A dead relay reports to the Director, which re-routes the schedule. Today
   fallback logic lives in the deck.
4. **Voices are station-local.** What is installed differs by machine, so the schedule must be built
   against what this station actually has.

## Questions this charter opens for the RCC

Recorded now so they are answered by design rather than discovered at the keyboard. They join
G-1…G-11 in `read-order-design.md`.

| # | Question |
|---|---|
| D-C-1 | **Does the Director own the location-report rotation too?** The analog says it schedules them ("we're due to read a location weather report next"). Today the watchlist advance and dwell live in `app/radio_queue.go`. If yes, the playlist is one queue holding reports *and* alerts, not an alert path beside a report path. |
| D-C-2 | **What is the ticker cue's contract?** "3, 2, 1" implies lead time and possibly acknowledgement. Does the Director wait for the ticker to confirm it is ready, or fire and trust? What happens if the ticker cannot comply — a modal is open, a takeover is already drawn? |
| D-C-3 | **What is the standby contract?** How far ahead is the next item prepared, what is discarded when the schedule changes underneath it, and what is the cost of preparing something never played? |
| D-C-4 | **How are failures reported up?** A relay dying, a voice missing, a fetch failing — one channel or several, and does the Director re-plan or only re-route? |
| D-C-5 | **What does "surfacing a decision" mean in Observer?** In Broadcaster the operator is present to answer. In Observer there is only the settings the listener already gave — so is a decision the Director cannot resolve from settings simply an accepted default, and where is it recorded? |
| D-C-6 | **What is the seam?** The Director must instruct the ticker, the voices, the scheduler and the radio without knowing how any of them work. Naming that interface *is* the design: it is what makes Broadcaster an extension rather than a rewrite. |
| D-C-7 | **Which existing owners does it absorb, and which does it merely instruct?** `capBurst`, `startTakeover`, `breaking`, `readBreaking`, the ticker's rotation, `radioDeck`'s fallback and the watchlist advance are all candidates. The charter says the Director coordinates rather than does — so most should stay where they are and gain an instruction interface. |

## The Broadcaster picture the schedule has to serve

**HUM LEAD, 2026-09-01 — illustrative, not prescriptive.**

The Broadcaster UI shows the schedule as a **flowing feed, bottom-up**: the queue of what is going to
be played or read, rising toward the top. Each item is a **card**.

- A card's options are **editable until it reaches the ON AIR position** at the very top. At that
  moment every decision on it is **locked in**.
- The only thing that supersedes a locked card is a **takeover** — a disaster alert.
- Each card's options are carried out by the function that owns them; the Director does not perform
  them.
- **The feed itself instructs those functions when to fetch, pre-fetch and pre-render.** This is how
  a real station runs: the lineup drives everything.
- **And the lineup is fluid.** When it changes, the staff — our processes — adjust. Sometimes that
  means discarding work: the operator deletes a card that was going to play next, the item behind it
  moves into *Up Next*, and the system re-prioritises getting that one ready.

Two consequences for 0.14.0, before any Broadcaster UI exists:

1. **The schedule is the product, not a detail of the audio path.** If it drives pre-fetch and
   pre-render, it must exist as a first-class thing that can be inspected, edited and re-ordered —
   which is also what makes it testable, and testability is a requirement of this design.
2. **Discard must be cheap and safe.** Anything prepared ahead can be thrown away, so preparation
   cannot have side effects that outlive it.

## D-C-7 — the candidates, and what they do today

Two facts established by reading the code, which the design has to reckon with:

**There are two independent paths to audio, and neither one is a schedule.**

- The **broadcast path**: `radioDeck` → `engine.Start` (a relay) or `engine.StartSource` (a
  synthesised location cycle), driven by `tune`, `advanceQueue` and `armDwell`.
- The **narration path**: `director.Run` → `speaker`, with exactly two callers —
  `app/severe_read.go` (the `[space]` read) and `app/ticker.go` (the breaking takeover).

They never meet as a queue. They meet at the **volume**, inside the engine, via
`Suppress`/`Restore`. That is why a bound in one could silence a hazard the other believed was being
read.

**Today's `director` is a voice ARBITER, not a schedule owner.** It holds `onAir`, `suspended`,
`waiting` and `ducked`, and decides who gets the air when two things want it. Nothing in it plans.

| Candidate | What it does today | Proposed |
|---|---|---|
| `radioDeck.advanceQueue`, `armDwell`, `stopDwell` | Rotates the watchlist and times each location's dwell | **Absorb** — this *is* the schedule, for one kind of item |
| `tickerDeck.startTakeover`, `capBurst` | Chooses which alerts are announced and how many | **Absorb** — lineup selection, and where the read order lives |
| `director` (current) | Class preemption, suspend/resume, duck | **Absorb** — already coordination; becomes the Director's air-control half |
| PLAY / PAUSE / STOP (`radioDeck.Stop`, `Tune`, `SetRepeat`, the panel keys) | Transport, straight from the keyboard to the deck | **Relay through the Director** — commands arrive from the human or from their stored presets, and the Director acts on them |
| `radioDeck.tune`, `SetMode`, `followMount`, `startSynth` | Picks relay vs synth for a location; handles fallback | **Instruct**, but the **fallback reports UP**: "this relay is down" is a schedule change, and the Director re-routes |
| `engine.Suppress`/`Restore`/`giveWayLocked` | How the broadcast gives way — dip for a relay, hold for a report | **Instruct** — the Director says an alert is on air; the engine decides how to yield from the source kind. Already the right shape |
| `modes/tty/ticker.go` — `setTicker`, `advanceTicker`, `advanceTickerCategory` | The band's contents and rotation, derived independently | **Instruct** — the charter's cue with lead time replaces the derivation |
| `severeDeck.publish`, `setLaneRows`, `LaneRows` | Builds the window index and the tape's lane rows | **Instruct** — data assembly, not coordination |
| `tickerDeck.cycle`, `seenStore` | Fetches, scopes, and decides what is new | **Instruct** — fetching is not coordination; the Director asks what is new |
| `script`, `pronounce`, `synth`, `cast.resolveVoice` | Scripts, pronunciation, voices | **Leave alone** — explicitly outside the charter |
| `severe.Classify/Sort/Cap`, `globalfeed.Merge/capPerLane` | Classification and bounds | **Leave alone** |

The shape that falls out: the Director absorbs **the schedule and air control**, and gains an
instruction interface to everything else. Most of the code stays where it is.

## The convention the Director is built on — DONE

**HUM LEAD, 2026-09-01. Built the same day — `platform/category` is the registry.** The four enums
are now aliases of one type and the order lists are views of one table, so the Director does not have
to build this first and must not become a fifth list.

What follows is the problem it solved, kept because it is the reason the convention exists.

The same concept — an alert category — was declared four times, with
hand-kept order lists beside each. Adding a category means editing six places, and forgetting one
fails *silently*: a lane missing from `laneOrder` drops its alerts from the stack, one missing from
`tickerCatOrder` never reaches the band, and the two tab enums disagreeing files every row one tab
over. All three were live in this release.

The Director should be built on **one registry**, not become a fifth list. It needs, per category:
identity, label and short form, render token, lane membership and rotation position, whether the
national feed can produce it, whether it is watchlist-only — and now, from MVS-D-60, **per-category
ordering rules**: the proximity weight on `Local Area Emergency`, the pin-to-top on `Child Abduction
Emergency`. Those rules have nowhere to live today, which is why they were recorded against the
read-order design rather than built.

The constraint is real and enforced: `scripts/lint-imports.sh` forbids anything under `modes/` from
importing `domains/*`, which is exactly why the mirror enum exists. The registry therefore belongs in
a shared leaf package under `platform/` — the precedent is `platform/snapshot`, already imported by
both layers.

**Where the Director's own rules go.** Every ordering question the redesign must answer — the
weighted lineup, the lane floor, what the ticker shows against what is read — is a per-category rule,
and `category.Spec` is where one lives. MVS-D-60's two are the first candidates: the proximity weight
on `Local Area Emergency` and the pin-to-top on `Child Abduction Emergency`.

## Why the bar is what it is

The Broadcaster edition puts the Director in charge of running a station. Getting this ownership
right now is the difference between that edition being an extension and being a rewrite — which is
why MVS-D-56 held 0.14.0 for it, gave it a FULL RCC and PLAN, and made testability a requirement
(the SETTING is since deferred to post-Broadcaster by MVS-D-79; the ordering itself ships in 0.14.0)
of the design rather than something added afterwards.
