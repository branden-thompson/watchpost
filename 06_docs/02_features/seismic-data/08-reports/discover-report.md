# Seismic data — DISCOVER / RCC report

**Feature:** `seismic-data` → `0.11.0` · **SEV-0** · **Phase:** DISCOVER (RCC) · **Date:** 2026-08-27
**Status:** DISCOVER complete — **awaiting HUM LEAD gate to enter PLAN** (5 decisions below).

## 1. What was asked

Add USGS seismic data as a **section in Location Details**, answering *"did my area have a quake?"*,
with a **magnitude-graduated distance rule** (smaller quakes must be nearer). Global significant-quake
ticker deferred to **0.12.0** (bundled with a global severe/catastrophic-weather ticker that needs its
own discovery).

## 2. What DISCOVER found (see `01-objectives`, `02-analysis`)

- **USGS `fdsnws/event/1/query`** (keyless, public-domain, HTTPS, `Cache-Control: max-age=60`) is a
  circle-around-a-location query — the per-location model the app already uses for weather/marine/fire.
  Live-probed: San Diego 17, Ridgecrest 40, Anchorage 98, Chicago **0** quakes M2.5+/300 km/30 d —
  single-to-double digits at the proposed defaults, small payloads.
- The data carries everything the section needs (magnitude, place, depth, time, tsunami, PAGER alert,
  felt/intensity, significance, type); distance/bearing are computed with the existing `platform/geo`
  helpers.
- **Seismic is a fourth hazard domain** beside weather/marine/fire, riding the same snapshot →
  scheduler → detail seams the quality pass hardened. No novel infrastructure. Effort is a new
  `domains/seismic/usgs` provider, a `KindSeismic` snapshot payload + tier, a `detail_seismic.go`
  renderer with goldens, `[seismic]` config, and (optionally) request-sharing like the Q5 FIRMS tile.
- The **novel** piece is the graduated rule: `show iff distance ≤ radiusFor(magnitude)`, a pure
  interpolated curve over config breakpoints, applied client-side after one widest-radius fetch —
  identical output whether or not requests are shared.

## 3. Recommendation

Ship 0.11.0 as **near-location seismic in Details, display-only, USGS, graduated-radius rule,
request-shared boxes**, following the fire domain's shape end to end. It is a well-bounded SEV-0
feature that reuses the just-hardened seams; the only genuinely new logic is the graduated filter and
its config, both small and unit-pinnable.

## 4. Fit with the current state

- Branch `feature/seismic-data` cut from the quality-pass HEAD (all shipped through `0.10.2`). The
  7-day performance soak keeps running on its own binary/process — unaffected by this branch.
- 0.11.0 follows the quality pass in the release line; the publish protocol (orphan `main-publish`,
  tag) is unchanged.
- The quality pass's standards are the floor: FULL TDD, P10 0-live, stated bounds, junior-first docs,
  goldens, `a2dh validate`, the counters/soak apparatus for the byte/request check.

## 5. Decisions for the PLAN gate (HUM LEAD)

| # | Decision | DISCOVER recommendation |
|---|---|---|
| D1 | **Radio narration** — narrate quakes in the synth broadcast (R6: smokes+soak) or display-only for 0.11.0? | **Display-only** for 0.11.0; narration a clean follow-up. |
| D2 | **The graduated curve** — ratify the tiers (2.5→50, 3.5→150, 4.5→400, 5.5→1000, 6.5→2500 km) and the floor (M2.5) + window (7 days), or adjust. | Ship these defaults; all `[seismic]` knobs. |
| D3 | **Request sharing** — shared canonicalised boxes (Q5 FIRMS-tile idea) from the start, or per-location first? | **Share from the start** (quakes are regional; big overlap). |
| D4 | **Quake types** — `earthquake` only, or include quarry blasts / explosions / ice quakes? | **`earthquake` only** (a knob). |
| D5 | **Section fields / layout** — the mock for the SEISMIC rows (magnitude · place · dist+bearing · depth · age; tsunami/PAGER emphasis). | Provide an ASCII mock at PLAN for HUM LEAD to direct (feedback-mock-fidelity). |

## 6. Proposed PLAN batches (preview — for the PLAN artifact, not committed here)

- **P1 data layer first** (DATA FIRST): `platform/snapshot` `KindSeismic` + `SeismicState`/`Quake`;
  `domains/seismic/usgs` provider (query build, GeoJSON decode, graduated filter, cap); the rule as a
  pure `seismic.Rules` with the curve; unit tests incl. the equivalence and boundary cases.
- **P2 scheduler + pipelines**: the `KindSeismic` tier, cadence with a §0.3 freshness argument, the
  shared-box request (the FIRMS-tile precedent) + its parsed memo bound + gauge.
- **P3 detail section + config**: `detail_seismic.go` from the approved mock, colour-off/ASCII
  goldens; `[seismic]` config → rules; attribution credit; docs (`architecture.md`, `extending.md`,
  the flow map).
- **P4 review + release**: red-team pass (SEV-0), the byte/request check on the counters, `a2dh
  validate`, publish `0.11.0`; DEBRIEF.

## 7. Open, tracked for later

- **0.12.0 candidate:** the global significant-quake + severe/catastrophic-weather ticker (hurricanes,
  typhoons, tropical storms, tornado warnings, damaging weather) — its own DISCOVER; more data to
  shape (NHC, SPC, the USGS significant feed).
- Depth-weighting the graduated rule (v1 is magnitude-only).
