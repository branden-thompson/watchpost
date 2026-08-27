# USGS seismic data — shape, volume, and how it maps to Watchpost (DISCOVER analysis)

Live-probed 2026-08-27 from a checkout of `feature/seismic-data`. Every number below is reproducible
with the `curl` lines shown.

## 1. The endpoint

USGS **FDSN event query** — keyless, public-domain, HTTPS:

```
https://earthquake.usgs.gov/fdsnws/event/1/query
    ?format=geojson
    &latitude=<lat>&longitude=<lon>&maxradiuskm=<km>   # a circle around the location
    &minmagnitude=<m>
    &starttime=<YYYY-MM-DD>                             # the lookback window
    &limit=<n>
```

This is the **per-location model exactly** — a circle around a lat/lon, a magnitude floor, a time
window — the way `firms` requests a box and `nws` requests a point. A sibling `…/count?` endpoint
returns just the number (useful for a cheap "any activity?" probe).

**Caching:** the response carries `Cache-Control: public, max-age=60` and an `Expires` header, so the
httpx client's server-TTL path already covers it; a caller TTL (a few minutes) is the right floor —
quakes are not more current than the feed's own minute.

**Feeds (alternative):** USGS also publishes pre-bucketed GeoJSON feeds (`all_hour`, `2.5_day`,
`4.5_week`, `significant_month`, …) at `…/feed/v1.0/summary/`. These are global, not near-a-location,
so they fit a **ticker** (0.12.0), not the per-location Details section. The query API is the right
tool here.

## 2. One feature's fields (a M2.56 near Progreso, MX)

`features[].properties` keys: `mag, magType, place, time, updated, tz, url, detail, felt, cdi, mmi,
alert, status, tsunami, sig, net, code, ids, sources, types, nst, dmin, rms, gap, type, title`.
`features[].geometry.coordinates` = `[lon, lat, depth_km]`.

The fields Watchpost cares about:

| Field | Meaning | Use in the section |
|---|---|---|
| `mag`, `magType` | magnitude and its scale (`ml`, `mww`, …) | the headline number |
| `place` | human string, e.g. "52 km SSW of Progreso, B.C., MX" | context line |
| `geometry[2]` (depth km) | hypocentre depth | a shallow quake is felt more — worth showing |
| `time` (epoch ms) | origin time | "N ago" (the fixedAge helpers) |
| `tsunami` (0/1) | a tsunami message was issued | a flag worth surfacing on coastal locations |
| `alert` | PAGER level green/yellow/orange/red (often null) | emphasis for a damaging event |
| `felt`, `cdi`, `mmi` | Did-You-Feel-It count, community/instrumental intensity | "felt" signal (candidate) |
| `sig` | USGS significance 0–1000+ | a ranking key |
| `type` | earthquake / quarry blast / explosion / ice quake | filter to `earthquake` (objectives §7.4) |
| `status` | reviewed / automatic | an automatic solution may revise — show as-is, USGS does |

Distance and bearing from the tracked location are **not** in the payload — computed client-side with
`platform/geo.HaversineKM` and `geo.BearingDeg` (already used by fire), so the "12 km NE" line and the
graduated-radius filter are ours to compute.

## 3. Volume — how much data, where (the M2 / bounds concern)

| Location | query | count |
|---|---|---|
| San Diego, CA | M2.5+ / 300 km / 30 d | 17 |
| Ridgecrest, CA (active) | M2.5+ / 300 km / 30 d | 40 |
| Anchorage, AK (very active) | M2.5+ / 300 km / 30 d | 98 |
| Chicago, IL (stable interior) | M2.5+ / 300 km / 30 d | **0** |
| Ridgecrest | M1.0+ / 100 km / **24 h** | 7 |

Reading: at the objectives' defaults (M2.5 floor, 7-day window, ≤ 2,500 km only for M6.5+), a normal
location returns **single digits**; the busiest US regions return dozens over 30 days, so **well
under a hundred over 7 days**. Payloads are small (a few KB). Lowering the floor to M1.0 in an active
area multiplies the count — the reason the floor and the graduated radius exist. **Bound to state
(PLAN):** the per-location result is capped (top-N by significance/recency) so a swarm cannot grow the
section or the memo without limit.

## 4. The magnitude-graduated radius, grounded

The felt radius of an earthquake grows roughly an order of magnitude per ~2 magnitude units. The
objectives' tiers approximate that: M3 ~ tens of km, M5 ~ few hundred, M6.5+ ~ a thousand-plus. The
rule is a pure function `radiusFor(mag)` interpolated over the `radius_tiers` breakpoints, applied
after the fetch — so USGS is queried once at the **widest** radius (the top tier) and the **lowest**
magnitude (the floor), and the curve filters the result. One request per location (or per shared
box), one client-side filter; identical output whether or not requests are shared (the FIRMS-tile
equivalence property).

## 5. How it lands in the existing architecture

- **`domains/seismic/usgs`** — a new `snapshot.Provider` (like `domains/fire/hms`, `domains/weather/nws`):
  `Fetch(KindSeismic)` builds the query URL from the location + rules, decodes GeoJSON, applies the
  graduated filter and the cap, returns a `snapshot.SeismicState` (AsOf + []Quake) per location.
- **`platform/snapshot`** — a new `FetchKind` (`KindSeismic`), a `SeismicState`/`Quake` payload on
  `PartialData`, merged by the assembler like fire. `AsOf` distinguishes "no quakes" (feed answered,
  none matched) from "unavailable" (feed cold/down) — the FIRE `AsOf` precedent (red-team B5 P3).
- **`platform/sched`** — a `KindSeismic` tier in the priority and RECENT pipelines (the objectives'
  cadence; a few-minute `Every` given `max-age=60`).
- **`modes/tty/detail*.go`** — a SEISMIC section renderer beside `detail_fire.go` / `detail_marine.go`,
  goldens pinned; the row layout from a mock (feedback-mock-fidelity).
- **`app`** — wire the provider into the pipelines (`fireProviders` sibling), a gauge for any parse
  memo, the `[seismic]` config → rules mapping (`fireRules` sibling), the attribution credit.
- **Request sharing (PLAN, from Q5)** — the USGS box for two nearby locations overlaps almost
  entirely; a canonicalised-box request keyed like the FIRMS tile would collapse them to one call.
  Recommended from the start; the Q0-style transaction count decides the box size.

Nothing here is novel infrastructure — seismic is a fourth hazard domain beside weather, marine, fire,
riding the same snapshot/scheduler/detail seams the quality pass just hardened.

## 6. Risks / unknowns for PLAN

- **Swarms** (Ridgecrest-style aftershock sequences) can spike a location's count — the cap and the
  floor bound it; the section should summarise ("+37 more") rather than list all.
- **`type` filtering** — quarry blasts and explosions appear near populated areas; filter to
  `earthquake` (a knob).
- **`alert`/`tsunami`** are rare but high-value — the section should emphasise a PAGER-alert or a
  tsunami-flagged event (a warning tone), decided at PLAN with the mock.
- **Depth** matters for "felt" — a deep M5 is unremarkable; the section shows depth so the reader
  judges. Whether the graduated rule should also weight depth is a PLAN question (DISCOVER: keep the
  rule magnitude-only for v1, note depth as a future refinement).
- **Rate/politeness** — USGS asks for a descriptive User-Agent (we send one) and reasonable rates;
  the shared-box request and the client cache keep us well within courtesy.
