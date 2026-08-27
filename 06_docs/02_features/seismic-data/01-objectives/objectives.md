# Seismic data — objectives (DISCOVER / RCC)

**Feature:** `seismic-data` · **Version:** `0.11.0` · **Level:** 1 · **SEV:** 0
**Directives:** FULL GIT · FULL DOCS · FULL REPORTS · FULL RCC · FULL PLAN · FULL TDD
**Owner:** HUM LEAD (Branden Thompson) · **Opened:** 2026-08-27 (while the quality pass soaks)

## 1. Intent (HUM LEAD, verbatim)

> Add SEISMIC data as appropriate for locations. USGS real-time GeoJSON. SEISMIC as a section in the
> location details. Main use case: **"did my area have a quake?"** Rules for magnitude — the lower
> the magnitude, the closer it has to be to my location to show up.

A global significant-quake ticker is **out of scope for 0.11.0** (HUM LEAD): it belongs with a broader
"Global Severe / Catastrophic Weather Event ticker" (hurricanes, typhoons, tropical storms, tornado
warnings, damaging weather) that needs its own discovery — a **0.12.0** candidate.

## 2. The user need

A person tracking a location wants to glance at its details and answer *"has the ground shaken near
here recently, and how much?"* — the seismic analogue of the FIRE section. Not a seismologist's tool:
a resident's reassurance-or-alarm signal, in the location's own detail view.

## 3. Scope (0.11.0)

- **In:** a SEISMIC section in Location Details, per tracked location, listing recent nearby
  earthquakes from USGS, filtered by a **magnitude-graduated distance rule** (§4); the same section
  the FIRE rows established as a pattern.
- **In:** a `[seismic]` config section with the rule's knobs (min magnitude, the magnitude→radius
  curve, the lookback window), defaulted so a fresh install is sensible — mirrors `[fire]`.
- **In (added by HUM LEAD 2026-08-27, reversing the earlier "details-only" call):** a **row-marks glyph
  on the main table** — the strongest recent quake's felt-band glyph (○/●/◉) only, no count, in the marks
  block between the play and fire marks (`›  ▶ ● 5◆ 3⚠`). One glyph keeps the table uncluttered while
  surfacing "did my area shake" at a glance.
- **Out (0.11.0):** a dedicated seismic view; a global/significant ticker; non-USGS sources;
  historical/archive browsing. (Radio narration is IN, in the final batch — D1.)
- **Deferred (0.12.0 candidate):** the global severe-weather + significant-quake ticker.

## 4. The magnitude-graduated distance rule (RATIFIED — HUM LEAD 2026-08-27, D2)

Seismic visibility scales with magnitude: **a small quake shows only if it is close; a large quake
shows from far away.** The rule is a **step function** `radiusFor(magnitude)` — the max distance a
quake of a given magnitude is shown from — grouped by felt-likelihood. Distances are in **miles**
(the config's and the UI's unit); the app converts to km for the haversine comparison.

| Band | Magnitude | Shown within |
|---|---|---|
| **Can't feel it** | < 1.0 | 3 mi |
| | 1.0 – 2.5 | 10 mi |
| | 2.5 – 3.5 | 20 mi |
| **Might feel it** | 3.5 – 4.0 | 40 mi |
| | 4.0 – 4.5 | 100 mi |
| **Almost certainly felt** | 4.5 – 5.0 | 150 mi |
| **Significant** | 5.0 – 6.0 | 400 mi |
| | 6.0 – 7.0 | 500 mi |
| | ≥ 7.0 | 1,000 mi |

A quake of magnitude *m* takes the radius of the band whose upper bound is the first greater than *m*
(so `m = 3.8` → 40 mi; `m = 4.2` → 100 mi; `m = 7.5` → 1,000 mi). There is **no separate magnitude floor** — the < 1.0
band (3 mi) is the floor: a tiny quake shows only if it is essentially under the reader's feet, which
is exactly "did *my area* shake". The lookback window is the last **7 days** (a `[seismic]` knob).

**Felt-likelihood is also a UI signal** (D5): the section can label or tint each quake by its band —
"you likely felt this" vs "recorded nearby, below feeling" — turning the rule into reassurance.

**How this compares to the DISCOVER recommendation:** HUM LEAD's tiers are finer and, for small/mid
quakes, tighter — a M2 shows within 10 mi here vs ~31 mi in the coarse draft, and the high end is more
conservative (a M6.5 within 500 mi vs ~1,553 mi). Two consequences, both intended: (a) sub-M2.5
near-field events now show (the coarse draft floored them out), so a reader learns of a genuinely
local small quake; (b) most locations most of the time will read "no recent activity" unless
something close or regionally large happened — the honest answer to the use case.

## 5. Success criteria (measurable, verified at REVIEW/VALIDATE)

- A location with a recent nearby quake shows it in Details with magnitude, place, distance +
  bearing, depth, and "N ago"; a quiet location shows an honest "no recent seismic activity" (never a
  blank or a fake zero — the FIRE `AsOf` precedent: a cold/unavailable feed reads "unavailable", not
  "none").
- The graduated rule is enforced and unit-pinned: a M2.6 at 200 km is hidden; the same quake at 40 km
  shows; a M6 at 800 km shows.
- The request is **per tracked location** on a cadence consistent with the data (USGS `max-age=60`);
  requests are shared across nearby locations where the boxes overlap (PLAN: the FIRMS-tile precedent),
  and bytes/requests are measured against the quality pass's counters apparatus (no regression to the
  M2 request floor).
- No secret, no key (USGS is keyless/public-domain); attribution line carried (USGS Earthquake Hazards
  Program, public domain).
- R6 respected: if radio narration is included (§7), the Synth/Relay smokes + soak gate apply; if not,
  the radio path is untouched.
- Every quality-pass standard holds: P10 0 live, bounds stated for any new memo/cache, junior-first
  docs, goldens for the new section, `a2dh validate`, the publish protocol.

## 6. `[seismic]` config (proposed shape — PLAN finalises)

```toml
[seismic]
enabled = true
lookback_days = 7            # how far back "recent" reaches
types = ["earthquake"]       # USGS event types shown; extendable to "quarry blast", "explosion" (D4)
# the magnitude→radius step function as ascending (upper-magnitude, miles) bands;
# a quake shows iff its distance ≤ the miles of the first band whose magnitude > the quake's
radius_bands_mi = [[1.0, 3], [2.5, 10], [3.5, 20], [4.0, 40], [4.5, 100], [5.0, 150], [6.0, 400], [7.0, 500], [99, 1000]]
```

## 7. Decisions (RATIFIED by HUM LEAD 2026-08-27 — "Approved; Go 4 PLAN")

1. **D1 Radio narration — YES, but last.** Seismic *is* narrated in the synth broadcast; it lands in
   the **final** batch, after the data is wired, loaded and displaying correctly (HUM LEAD provides
   the scripts). R6 applies to that batch (Synth/Relay smokes + soak).
2. **D2 The graduated rule — ratified** as the §4 table (HUM LEAD's finer, mileage-based bands).
3. **D3 Request sharing — approved** (shared canonicalised boxes from the start, the Q5 FIRMS-tile
   precedent).
4. **D4 Earthquake only — approved,** but the `type` filter is config-driven (`types`) so blasts /
   explosions extend with one config line, no code change.
5. **D5 Section layout — approach approved;** PLAN carries an ASCII mock for HUM LEAD to direct, with
   the felt-likelihood band as a UI signal.

**Performance mandate (HUM LEAD):** this feature must not regress the quality pass's gains. The PLAN
carries the quality-pass disciplines as explicit gates — frame allocation budget unchanged, one
shared-box request with a bounded parsed memo + gauge, server `max-age`/conditional-GET respected, the
counters/soak byte-and-request check, and every new structure bound-stated (P10).

## 8. Non-goals (explicit)

Not a seismology tool; no focal mechanisms, no magnitude-type nuance in the UI, no aftershock
forecasting, no ShakeMap rendering, no historical catalog browsing, no alerting/notifications, no
non-USGS or global-ticker surface (0.12.0). One section, one question answered.
