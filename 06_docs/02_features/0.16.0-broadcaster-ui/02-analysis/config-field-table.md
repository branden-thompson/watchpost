---
title: "0.16.0 DISCOVER — every persisted setting, for per-field ruling (D-18)"
date: 2026-09-09
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "RULED — HUM LEAD approved every recommendation 2026-09-09 (\"All recommendations approved\").  Row 4 stays deferred by its own terms."
---

# Every setting — RULED

**HUM LEAD, 2026-09-09: *"All recommendations approved."*  Every row below is ruled as recommended,
and Table 2's five proposed Broadcaster settings are approved as proposed.**

**Two consequences carried forward, because "approved" did not resolve them:**

1. **Row 4 (`keys.<action>`) is not ruled S/O/B — it could not be.**  What is approved is its
   *disposition*: per-surface scoping is built first, and the S/O/B question is answered afterwards.
   Approving a recommendation that says "needs a namespace" approves the sequence, not a value.
2. **Row 27 STANDS — see D-20, which amends D-12.**  There are TWO radii with different meanings:
   Observer's bounds ALERTS over an unbounded location set; Broadcaster's is a HARD boundary on
   LOOKUPS, so it bounds the location set itself and constrains alerts transitively.  Row 25 keeps its
   meaning and **is not renamed**; `broadcaster.service_radius_mi` is a real, separate setting.


**52 persisted dotted paths**, grouped into **25 natural ruling units**.  Ruling codes:
**S** shared (one value, both surfaces) · **O** Observer's own · **B** Broadcaster's own ·
**SPLIT** both surfaces keep independent copies of the same setting.

`FirstRun` and `Unknown` are excluded — they carry `toml:"-"` and are never written.

## Table 1 — the existing settings

| # | Setting | Paths | What the user is choosing | Rec. |
|---|---|---|---|---|
| 1 | `locations[]` | 6 | The watchlist.  **Index 0 is the default location** every report, fire ring and seismic band measures from | **O** |
| 2 | `recent[]` | 6 | The most-recently-searched stack, newest first | **O** |
| 3 | `providers.<name>.key` | 1 | Per-provider API key (FIRMS today) | **S** |
| 4 | `keys.<action>` | 1 | Key-binding overrides, one action to one or more keys | **needs a namespace** |
| 5 | `radio.mode` | 1 | Tuner source: `synth` or `relay` — the `[m]` pick | **O** |
| 6 | `radio.cast` | 1 | Single Voice (the root reads everything) or Cast (per-role pairs) | **S** |
| 7 | `radio.voices.*` | 18 | One voice pair (macOS · Piper) for each of 9 roles: alerts, breaking, severe_read, standard, weather, maritime, fire, seismic, station | **S** |
| 8 | `radio.tones.mode` | 1 | All tones on, or mute mode | **SPLIT** |
| 9 | `radio.tones.muted` | 1 | Which tone classes are muted.  Empty under mute mode means every class | **SPLIT** |
| 10 | `fire.radius_km` | 1 | How close a hotspot must be to count | **O** |
| 11 | `fire.incident_radius_km` | 1 | How close a named incident must be to be listed | **O** |
| 12 | `fire.min_frp_mw` | 1 | Weaker detections are ignored below this power | **O** |
| 13 | `fire.bold_frp_mw` | 1 | Strong detections above this are read emphasised | **O** |
| 14 | `fire.min_confidence` | 1 | Detection confidence floor: low, nominal or high | **O** |
| 15 | `seismic.enabled` | 1 | Quake detection on or off | **O** |
| 16 | `seismic.lookback_days` | 1 | How many days of USGS history are queried | **O** |
| 17 | `seismic.types` | 1 | Which USGS event types are shown | **O** |
| 18 | `seismic.radius_bands_mi` | 1 | The magnitude-to-radius step function | **O** |
| 19 | `theme` | 1 | The active colour theme | **S** |
| 20 | `voice` | 1 | The root correspondent voice used in Single Voice mode | **S** |
| 21 | `units` | 1 | Imperial or metric | **S** |
| 22 | `clock` | 1 | 12-hour, 24-hour or military | **S** |
| 23 | `update_check` | 1 | Opt-in check for a new release at startup | **S** |
| 24 | `ticker_muted` | 1 | A **derived mirror** of `radio.tones.mode`; `Save` is its only writer | follows row 8 |
| 25 | `ticker_radius_mi` | 1 | **Observer's ALERT radius.**  It feeds `lineup.Fence`, which admits *arrivals* — alerts — not locations.  Observer's location set is unbounded.  0 means All | **O**, per **D-20** |

**Total: 52 paths.**

### Notes on the three rows that are not simple preferences

- **Row 25** is settled by **D-12**: the service radius is *the* boundary, one value.  It is listed here
  for completeness, not for re-ruling.  Its name is now wrong for what it does and should change.
- **Row 24** is derived, not chosen.  Whatever rules row 8 rules this.
- **Row 4** cannot be ruled S, O or B as things stand.  `term.KeyMap` is one flat namespace whose merge
  rejects any key claimed twice across the whole map, and Broadcaster's own key row already collides
  with Observer on `s`, `a`, `S` and `A`.  **It needs per-surface scoping built first**; the ruling is
  then about whether a user's overrides apply to one surface or both.

### Where a recommendation changed after D-11

Rows 6, 7 and 20 (the cast, the nine role voices, the root voice) were previously *needs ruling*, on the
grounds that the answer depended on whether Broadcaster speaks cards at all.  **D-11 settles that** —
the main track is the rotation, so Broadcaster drives the same reads through the same voices.  They are
now recommended **SHARED**: one station, one sound.

## Table 2 — the new Broadcaster settings, proposed

| # | Proposed path | Type | Reuses | Note |
|---|---|---|---|---|
| 26 | `broadcaster.tower` | the existing `Location` struct | **Reuse, do not invent** | R-7 says the name and the coordinates are one fact; `Location` already shapes exactly that (label, zip, lat, lon, tz).  **Must be a single table, never an array of tables** — `keepUnknown` refuses to follow those because *"indices are not stable"* |
| 27 | `broadcaster.service_radius_mi` | float64 | none — a new mechanism | **STANDS (D-20).**  A HARD boundary on LOOKUPS: it bounds which locations may exist for the station, and constrains alerts transitively.  Nothing today bounds the location set, so this is new work.  **A 3-mile hyper-local station is a stated supported case**, so the floor must be small and D-16's cap must behave when the radius admits very few locations |
| 28 | `broadcaster.relay` | string | none | The chosen audio-bed transmitter.  `radio.mode = "relay"` names a *kind*, and carries no identity |
| 29 | `broadcaster.gain_pct` | int | `Engine.Volume` mechanically | **Recommend B, persisted.**  Observer's volume is a listening preference and is not persisted at all (hardcoded 55).  A station's gain sets the level of a signal going over the air, and sharing them means an operator's on-air level changes because someone moved the listening volume |
| 30 | priority-tier cap and ordering | int | `sched.Tier` | **D-16**: a hard cap filled population-descending.  R-8.3 says the cadence set keeps one owner, so this generalises the existing tier table rather than adding a field beside it |

## The migration, which is purely additive

Add a `[broadcaster]` table; **touch, rename or repurpose nothing else** — except row 25's name, if
D-12's unification is confirmed, which is a rename with a compatibility shim rather than a move.  A
0.15.0 file then decodes unchanged and **R-2.4 holds by construction**.

For any row ruled **SPLIT**, a one-shot shim shaped like `withToneCompat` **copies** — never moves — the
legacy value once, so a first Broadcaster launch is not blank and the two are independent thereafter.

An old binary saving the file afterwards is already safe: `mergeUnknown` copies unknown paths from the
on-disk file into the fresh document before writing, which is the machinery `keep.go` was built for.
