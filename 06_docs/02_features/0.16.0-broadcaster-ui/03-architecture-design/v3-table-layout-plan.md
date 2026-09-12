---
title: "0.16.0 — the v3 table layout: what survives, what retires, and the stages"
date: 2026-09-12
phase: PLAN (re-plan within BUILD)
sev: SEV-0
authority: HUM LEAD
status: "All five rulings settled 2026-09-12.  Nothing blocking.  Stage 1 next."
---

# The v3 layout — a re-plan

**The HUM LEAD's mock, 2026-09-12**, after UAT: the running order becomes go-studs tables matching
Observer, the pool gains weather, and the line-up grows.  *"This should help harmonize some of the
plumbing because now Observer and Broadcaster are actually closer in their presentation."*

---

## The two rulings that shaped it

**1 — Observer's watchlist STAYS AT 10.**  The mock proposed 10 → 15 for consistency.  It is not
needed: the Line-Up draws from the STATION'S POOL, not the listener's watchlist — D-72 split them so
the console was not limited to what the listener happens to watch.  **And growing it carried the one
real risk to the shipped product:** M1 is a *Primary* metric defined as *"≤ 3s warm / ≤ 8s cold, 10
locations"*, so 15 would have been a 50% increase in cold-start population against the number the
metric names.  Ruled: leave it.

**2 — the pool stays DERIVED, with intra-session pinning.**  `pool.go`'s rule holds — *"storing it
would create a second answer that could drift from the settings that produced it"* — and `[l]` on a
location inside the service radius but outside the pool PINS it to the top for the session.

**Eviction: drop the farthest; population breaks a distance tie.**  Confirmed 2026-09-12.

- **Distance alone is already the rule.**  `Near` is *"kept when it is inside the fence, sorted by
  distance, cut to the limit"*, so "who falls off at 26" is already defined.  No second ranking.
- **Population as a TIEBREAK ONLY**, because the compound "farthest AND smallest" has no answer for
  95 mi/500 people versus 99 mi/200,000.  A rule the operator cannot say in one sentence defeats
  `PoolCap`'s own rationale: *"a pool wider than the operator can hold in their head is a rotation
  they cannot predict."*
- **A pin is exempt from eviction** — otherwise looking up somewhere 99 miles out evicts the thing
  just asked for.
- **A re-derive clears the pins.**  Changing the transmitter or the radius recomputes from settings.
  Operator-visible, so it is stated rather than discovered.

---

## `PoolCap` is already 25, and Observer's tables are already go-studs

Two things the mock asks for that are already true:

- **`locations.PoolCap = 25`**, ruled by the HUM LEAD on 2026-09-10 (*"25 is approved"*).
- **`platform/render/table.go` — *"the location table on go-studs DataTable — the ONLY go-studs
  consumer in the app"***.  So "go-studs tables" and "like Observer" are ONE instruction, reached
  through `render.LocationRow`.  Reusing it gives label, colour and behaviour parity **by
  construction**, which is what "the user doesn't have to relearn" actually requires.

**And `broadcaster.go` is the odd one out today**: it calls `components.NewDataTableRow` directly,
bypassing the platform wrapper.  Moving the console onto `render.LocationRow` corrects that drift as a
side effect rather than as separate work.

---

## Survives — untouched

| | |
|---|---|
| **D-74 · D-91 · D-92** | air ownership, the guard boundary, settings scoping.  None of it renders |
| **D-90** | the relay selection; the RELAY BED row is still `←`/`→` |
| **D-88** | the card window.  `0 Full Read` / `1 Read / Manage` / `A Details` are the same window with better labels |
| **D-87 stage 2** | `Card.Contents`, `Segment.Source/Detail`, `contentsFromSegments`.  The READ MANIFEST box is what is already built |
| the lineup model, the Director, the five roles | |

## Reusable

`cardStatus` · `cardPulled` · `shortAgo` · `manifestHeading` · `manifestRow` · `manifestRows` ·
`render.HeavyBox` · `boxRule` · the D-86 category tints (now on alert rows, not card grounds) · the
rail tints as the LIVE NOW / UP NEXT label cells · Observer's shared pointer across two tables
(`d.selected` spanning Watchlist+Recent via `numPriority`) · `railify` for the shared scroll.

## Retires

- **D-89's standby box** — LIVE becomes one table row; the empty state needs a row-shaped form.
- **D-87 stage 1's ten manifest CARDS** — only UP NEXT keeps a card.
- **The vertical left rail** — `railColumn`, `bcRegions`, `slotRows`, `zipTracks`, `priorityColumn`,
  `Railify`-for-cards, `bcRail*` widths.
- **D-86 1C's per-origin card tint** — replaced by a `REQUESTED BY` column, which *says* Producer vs
  Station Operator instead of implying it in a shade.  Strictly better.

Roughly the newest layout work; none of the plumbing.

---

## The stages

Each ends green and shippable.

| # | What | Retires | Notes |
|---|---|---|---|
| **1** | the console renders through `render.LocationRow` | `broadcaster.go`'s direct go-studs use | parity harness first: the same row data must render identically on both surfaces |
| **2** | **SCHEDULED LINE-UP** table, slots 2..N, with Observer's prefixes and alert tags | the rail, `bcRegions`, `slotRows`, `zipTracks` | the biggest single cut |
| **3** | **LOCATION POOL** table + `Population` on `LocationRef` + `[l]` pin and eviction + shared pointer | — | population is on `geodata.City` and NOT on `snapshot.LocationRef`; that plumbing is real work |
| **4** | the top **LIVE NOW / RELAY BED** table, the STATION AIR row and its elapsed timer, and the **gain split** | **D-89** | folds in the pending **stage C** (the relay-fault window's data flow) and corrects D-91's `SetVolume` row |
| **5** | **UP NEXT** and **ALERT** boxes side by side, both always present | D-87 stage 1's cards | the manifest content is unchanged |
| **6** | the per-card **PRESENTER** control | — | UNBLOCKED: `Card.ReadBy` already exists; the operator sets it, and it wins over the role cast |

---

## The three questions — RULED 2026-09-12

### 1 — FIFTEEN slots

`MainTrackSlots` 10 → **15**.  LIVE is 0, UP NEXT is 1, and the table draws **2..14 — thirteen rows**.

*The mock drew fourteen rows numbered `02`–`15`, which would be sixteen slots.  Fifteen is the ruling,
so the reference is redrawn to match it rather than the other way round.*

### 2 — the per-card presenter WINS, because Broadcaster thinks in beats

**HUM LEAD:** *"Observer is always '1 report at a time' whereas Broadcaster is 'what are the next 15
reads going to look like?'  In Broadcaster it's beats."*

That settles the precedence and explains it: on Observer a voice is a PREFERENCE, because there is only
ever one read in flight.  On the console the operator is composing a SEQUENCE, and varying who reads
which beat is part of composing it.  So:

```
the card's own presenter  >  the role cast (D-18 rows 6/7)  >  the root voice
```

**IT USES A FIELD THAT ALREADY EXISTS.**  `Card.ReadBy` is *"the display name of the voice that will
read this card, resolved against what THIS machine has"* — today it is resolved from the slot's cast
role.  The change is that the operator may set it.  No new field, no new resolution path.

**AND IT DOES NOT PERSIST, by construction rather than by decision:** a card is a slot in a running
order, not a setting.  It lives as long as the beat does.  Nothing reaches `config`.

### 3 — TWO stored integers, ONE component

**HUM LEAD:** *"I'm okay with a persistent gain value, it should re-use the code component so we don't
have two code copies of the same thing — but we can have two different integers stored for 'Observer
Volume' and 'Broadcaster Gain'.  That seems cheap — 2 copies of the same component code does not."*

**This RESTORES D-18 row 29** — `broadcaster.gain_pct`, persisted — and supersedes the 2026-09-12
"one volume for the app" reading, which was given against a survey that did not cite row 29.  Row 29's
own argument is the one that stands: *"sharing them means an operator's on-air level changes because
someone moved the listening volume."*

| | |
|---|---|
| **one** | the control, the bar, and `Engine.Volume` — no second copy |
| **two** | the stored integers: Observer's volume and the station's gain |
| **the surface picks** | which integer the control reads and writes, and which is applied on a swap |

**IT CHANGES 0.15.0's BEHAVIOUR, slightly and for the better:** Observer's volume is hard-coded to 55
at every launch today (D-18 row 29 notes it "is not persisted at all").  It becomes remembered.  That
is an improvement, and it is still a change to the shipped product, so it is written down rather than
slipped in.

**AND IT CORRECTS A ROW I SHIPPED YESTERDAY.**  D-91's `airBoundary` classifies `Radio.SetVolume` as
`airShared` with the reason *"one volume setting for the app"*.  That reason is now wrong.  The seam is
not shared — it is **routed**: reachable from both surfaces, writing whichever integer the current
surface owns.  The row's reason and possibly its bucket change with stage 4.

---

## Still open — nothing blocking

Stage 6 is unblocked.  Stage 4 now also carries the gain split, and D-91's `airBoundary` row for
`Radio.SetVolume` is corrected there.

## Two smaller notes

- The mock's READ MANIFEST numbers run `00, 02, 03, 04`; read as `01`–`04`.
- The mock shows real tower coordinates.  **F-66** keeps `<lat>, <lon>` in anything committed, because
  the repository is public — real coordinates render only on the operator's own screen.
