# The read lineup — a Director-owned, listener-configurable order

**Cold start? Read `README-director-handoff.md` first.**

> ## ⚠ PARTLY SUPERSEDED — read `lineup-model.md` first
>
> **This document is the original specification (HUM LEAD, 2026-08-31). Two of its central pieces were
> replaced at DISCOVER on 2026-09-01 and are WRONG as written here:**
>
> - **The ten-rung weighted scale below is superseded.** It predates Emergency Orders and Forecasts
>   and omits Marine entirely. The ratified order is in `lineup-model.md` — *The read order, ratified*.
> - **The per-category `Max` row and `TOTAL READS` are removed** (R-1). There is **one ordering and one
>   global Max**, not a budget per category.
> - **G-1 … G-11 below are mostly CLOSED.** G-9 was closed by reading the keymap; the rest were ruled
>   across R-1 … R-6 and T-5. `lineup-model.md` carries the answers.
>
> What remains accurate and useful here: the rationale for why the arrangement became a listener
> preference, the ALERTS - READ ORDER group's shape, and MVS-D-60's two per-product ordering rules.

**Status: SPECIFIED, NOT BUILT, and DEFERRED TO POST-BROADCASTER (MVS-D-79, HUM LEAD 2026-09-05).**

> **⚠ THE SHIPPING CONDITION BELOW IS SUPERSEDED.** MVS-D-56 held 0.14.0 for this; MVS-D-79 releases
> it. **0.14.0 ships without the SETTING.** What ships is the ORDERING — the ratified ladder, the
> significance fence, the global Max and the divert count are built and pinned (T1.1–T1.4, T4.3).
> What is deferred is the listener's ability to REARRANGE it, which lands after the Broadcaster UI.
>
> The HUM LEAD's reasons: 0.14.0 has already changed a great deal and this is a point release of its
> own; and *"I'm not sure if we want to give human users the ability to mess with this just yet."*
> The second is the weightier one — a knob on hazard ordering is a knob on which hazards a listener
> hears, and Broadcaster is the surface that will show what such a rearrangement actually does.
**Source: HUM LEAD, 2026-08-31.** Supersedes F-17, which was the backlog placeholder for it.

**The ruling.** 0.14.0 does not ship until this is built. There is no schedule pressure and the
requirement is that it *reliably works* — so it gets a **FULL RCC and a FULL PLAN**, not a patch.
Sequence: finish the remaining round-2 defects first, then open RCC on this.

Testability is a first-class requirement of the design, not something added afterwards. The reason
is on the record: this behaviour has been through two remediation rounds and each one moved the
failure boundary instead of removing it, because the ordering was a constant the tests asserted
against rather than a preference a test could set. A listener-set lineup is testable by
construction — the test states the arrangement it wants and asserts what is read.

## The Director runs the show

**The charter is `director-charter.md`** (HUM LEAD, 2026-09-01), and it is the wider document: the
read lineup below is *one* of the Director's responsibilities, not the whole of it. Read that first
— in particular that the Director is an **Assistant** Director, which decides how much of this it
may settle on its own.

**Architectural principle, and the reason this is worth a full PLAN.** The Director is the *sole*
owner of the arrangement:

- it builds the lineup — which alerts are read, in what order, and how many;
- it queues the reads;
- it inserts takeovers;
- and it keeps the **news ticker synced with the reads** as they happen, so what the band shows and
  what the listener hears are one schedule rather than two that agree by coincidence.

Nothing else decides any part of that. Today the arrangement is spread across `capBurst`,
`startTakeover`, `breaking`, `readBreaking` and the ticker's own rotation, which is precisely why a
bound in one of them could silence a hazard the others believed was being read.

**This is foundational for Broadcaster.** The planned WATCHPOST Broadcaster edition puts the
Director in charge of running a station — queuing reads, inserting takeovers, scheduling the whole
output. Getting that ownership right here is what makes that edition an extension rather than a
rewrite, so the bar is bullet-proof: properly architected, designed, guarded and tested.

Recorded here because the last three sessions have shown that a design living only in a chat log is
a design that gets re-litigated. This is the HUM LEAD's specification as given, followed by the gaps
in it that must be closed before anyone writes code.

## Why this replaces the current rule

0.14.0 currently reads a burst most-severe-first, bounded by count and by time. Two red-team rounds
have now found the same class of defect in that design: whichever hazards land at the tail of a
severity-ordered burst are the ones a bound silences, and they are silenced permanently under
sustained arrivals. Each fix moved the boundary rather than removing it — a per-lane floor, then a
derived time bound, and the reviewer's sweep still found a break at ~8.14 s per read.

The HUM LEAD's conclusion: *"there's too much 'what ifs' to approach this sensibly"* — the ordering
is not a constant to be tuned, it is a preference the listener sets.

## Default behaviour, with no listener input

**The Director owns the lineup.** Whatever schedules the read is the Director's job — it is the
component that already owns the air.

1. Take every alert from the last batch.
2. Sort by **severity**; within each severity band, sort by **proximity** to the Default Location.
   With no default location set, the application default is **Bonsall, CA** (an arbitrary but fixed
   choice, so the behaviour is defined rather than absent).
3. Build the readout lineup on a weighted scale — "most serious, closest first":

   | # | Bucket |
   |---|---|
   | 1 | **Close** (≤ 100 mi) Disasters |
   | 2 | Close Warnings |
   | 3 | Close Watches |
   | 4 | Close Advisories |
   | 5 | Close Special Statements |
   | 6 | Remaining Warnings |
   | 7 | Remaining Watches |
   | 8 | Remaining Advisories |
   | 9 | Remaining Special Statements |
   | 10 | Whatever is left |

4. **Schedule the burst**: decide what is read now and what is *diverted*.
   - The default is **5 alerts**, and it is a listener setting. Setting it to ALL is legitimate — a
     read may then take thirty minutes, but the listener has **opted into** that rather than having
     it chosen for them.
   - Anything past the limit is **diverted**, and the listener is told so:
     *"the remaining &lt;number&gt; severe weather events and details are accessible by pressing w in
     Watchpost."*

## The setting — ALERTS - READ ORDER

In Settings, alongside the existing ALERTS - EVENTS group whose distance filter it inherits.

```
ALERTS - READ ORDER
(inherits "all" or distance filter from "events")

> 01.  <- Category  ->  sorted by <- Dimension  ->, then <- Dimension ->, Max: [nn]
> 02.  <- Category  ->  sorted by <- Dimension  ->, then <- Dimension ->, Max: [nn]
...  one row per category
                                                          TOTAL READS: [nn]
```

- **Categories:** Disasters, Warnings, Watches, Advisories, Spec. Statements, Marine
- **Dimensions:** Distance, Severity, Recency, Expiration
- **TOTAL READS** auto-sums the Max values above it.
- Controls follow the correspondent-picker pattern already established in the cast group.

## Ordering rules ruled outside the default scale

**HUM LEAD, 2026-09-01, with MVS-D-60.** Two products carry ordering rules of their own. They are
recorded here rather than built as one-offs, because ordering is what this design owns — a
pin-to-top and a proximity weight bolted onto today's sort would be two more rules for the redesign
to unpick.

| Product | Rule |
|---|---|
| `Local Area Emergency` | **Higher priority within its lane the closer it is to the default location.** A proximity weight *inside* a lane — the default scale sorts by proximity within a severity band, so this is that rule applied within Disasters. |
| `Child Abduction Emergency` | **Always at the top of the modal.** A pin that overrides the normal sort outright, not a weight. |

The design must express both: a per-category **sort weight** and an absolute **pin**. If the
ALERTS - READ ORDER row model cannot say "this product pins to the top", it is not expressive enough.

## Gaps to close before building

Each of these is a real ambiguity, not a quibble. Guessing at any of them is how the last two rounds
produced defects.

| # | Gap |
|---|---|
| G-1 | **Marine is a category in the setting but absent from the default weighted scale.** Where does it sit — its own rung, or inside "whatever is left"? |
| G-2 | **Disasters appear only as "Close Disasters" (rung 1).** A distant significant quake therefore falls to rung 10, below a remote advisory. Intended? |
| G-3 | **Step 2 and step 3 are two different orderings.** Presumably step 2's severity-then-proximity sort orders *within* each weighted bucket. Confirm. |
| G-4 | **Proximity for a zone-only alert.** Such alerts carry no coordinates (this is open decision D-1). They cannot be sorted by distance — do they take their tracked location's distance, or sort last within their bucket? |
| G-5 | **Is TOTAL READS a cap or a readout?** If a listener sets per-category maxima summing to 40, is 40 the burst, or does a separate overall limit still apply? The default of 5 is described as one number, not six. |
| G-6 | **Does the divert line count against the budget**, and is it spoken once at the end of the burst or per diverted alert? |
| G-7 | **The 100-mile "close" threshold** — fixed, or does it follow the ALERTS - EVENTS radius the group says it inherits? |
| G-8 | **Bonsall, CA needs coordinates** baked in as the application default, and a decision on whether it is shown to the listener as their location or used silently. |
| G-9 | **The key is `[s]` Settings**; `[S]` is Watchpost Status. The spec says "[S] Settings" — assumed shorthand, worth confirming. |
| G-10 | **The time bound.** With the listener setting the count, does a wall-clock backstop survive at all, or does "ALL means all, even thirty minutes" retire it? |
| G-11 | **Ticker/read synchronisation.** The Director owns both. What does "synced" mean precisely — the band shows the alert currently being read, the band holds while a read is in progress, or the lineup and the rotation share one cursor? This is a design question the RCC must answer, not an implementation detail. |

## What this supersedes on landing

- **F-17** in `06_docs/follow-ups.md` — the placeholder for this work.
- **MVS-D-54** — the interim ruling that a burst is read most-severe-first with a derived time bound.
- `capBurst`, `maxBreaking`, `breakingAllowance` and `breakingCap` in `app/ticker.go`, and the
  per-lane floor built for them.
- Round-2 defect 2's open item R-1, which is the defect this design removes by construction rather
  than by tuning.
