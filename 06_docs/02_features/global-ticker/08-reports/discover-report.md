# Global event ticker — DISCOVER report (RCC phase exit)

**Feature:** `global-ticker` → `0.12.0` · **Phase:** DISCOVER · **Date:** 2026-08-27
**Level 1 · SEV-0 · FULL GIT/DOCS/REPORTS/RCC/PLAN/TDD.** Decisions D1–D4 ratified (objectives §6).

## 1. What we're building

A single-row **scrolling marquee** — a "breaking news" band of the largest active hazard events in the
world, independent of the watchlist. It rotates through recent severe events (earthquakes, tropical
cyclones, US severe/tornado warnings), colours its background by severity (Red/Orange/Yellow), plays a
tone and a short narration on a **new** alert, and can be muted with `[M]`. New alerts inject at the top
of a most-recent-first **stack** and interrupt whatever is scrolling.

## 2. Data-source reconnaissance (DATA FIRST — live-probed 2026-08-27)

| Class | Feed (keyless) | Identity | Narration location | Severity signal |
|---|---|---|---|---|
| Significant quakes | USGS `significant_week.geojson` (global) | `id` (`us7000tbwb`) | `place` ("55 km NW of …, Nepal") | `mag` / `sig` |
| Tropical cyclones | NHC `CurrentStorms.json` (Atlantic + E-Pacific) | `id` (`al042026`) | `name` + basin | `classification` (TD/TS/HU) / `intensity` kt |
| US severe / tornado | NWS `alerts/active` (national query) | alert `id` | `areaDesc` | `event` + `severity` (Severe/Extreme) |

- All three are **keyless / public-domain**. USGS significant is tiny (2.5 KB, ~3 events); NHC ~8.7 KB
  (~2 storms); NWS national severe is bounded by a national query (refine the filter at PLAN — my probe
  400'd on the param form; **the app already parses NWS alerts** in `domains/weather/nws`, so the types
  and parsing are reusable).
- **Coverage is honest, not literally global:** quakes global, tropical Americas-basins, severe/tornado
  US-only (accepted, D1-B). W-Pacific typhoons (JTWC/JMA) are a later add.

## 3. The architectural finding (shapes PLAN)

These feeds are **global, not per-location.** The whole existing data spine — `snapshot.Provider`,
`FetchReq{Locations}`, the assembler keyed by `LocationKey`, the per-location detail sections — assumes
"data for a tracked location." The ticker's events belong to no watchlist location. **So the ticker is a
separate data pipeline**, parallel to the snapshot: its own fetch cadence, its own `[]Event` stack, its
own publish into the UI — reusing `platform/httpx` (cache, conditional GET, the byte disciplines) and
the audio engine, but **not** the assembler. This is the biggest design decision and PLAN will lay it
out with approaches.

## 4. Novel elements (each a PLAN design item + a risk)

| Element | The question PLAN answers |
|---|---|
| The **Event** model | one type across three very different feeds: class, severity tier, title, location, verb, stable id, time |
| The **stack + new-alert detection** | dedup by id; "new" = id not seen before; a **persisted seen-cache for the last [x] days** so a restart doesn't re-alert; inject-at-top + interrupt |
| The **marquee render** | single row, horizontal scroll, background colour by severity, "breaking news" interrupt; reuse the existing marquee machinery or a new scroller |
| **Colour theming** | fixed Red/Orange/Yellow that are **theme-independent except monochrome** — how that sits in the theme-token system |
| The **alert tone** | a new audio asset — generated (EAS-like vs. a short chime) and played once per new alert through the engine |
| **Narration (R6)** | tone ×3 → 2 s → the general one-line template ("A(n) … has been declared/recorded/reported for …"); `[M]` mute gates narration **and** tone |
| **Event → location tying (D5)** | one event = one entry; resolve the named location to the highest applicable watchlist location, else the nearest named city, else a fuzzy area ("the San Diego area") — reuses the geodata/geo resolver; avoids N repeats of one quake |
| The **`[M]` control** | a new keybinding + a persisted preference |

## 5. Risks / unknowns

- **NHC basin gap** — no W-Pacific typhoons from NHC (accepted; note in the UI wording so coverage isn't
  overclaimed).
- **NWS national query** — needs the right filter (event/severity) and could be large during a big
  outbreak; bound it (top-N by severity) and honour the byte disciplines.
- **New-alert cache persistence** — must survive restarts (disk), be pruned to `[x]` days, and be bounded
  (P10). A cold cache on first launch must not alert-storm (treat first-seen-at-launch as not-new, or
  seed quietly).
- **The tone** — must not fire on every refresh, only genuinely new alerts; must respect mute; must not
  collide with the radio audio (mixing / ducking).
- **R6** — narration + tone is radio-adjacent; the audio batch carries the R6 gate (smokes + soak).

## 6. Ratified decisions (objectives §6)

D1 **B** (quakes + tropical + US severe/tornado) · D2/D3 the single-row marquee spec (stack, R/O/Y bg,
`[M]` mute, tone-on-new, general narration, [x]-day cache, theme-independent-except-monochrome) · D4
LEVEL-1 / SEV-0 / FULL everything.

## 7. Next: PLAN

FULL PLAN will carry: the separate-pipeline architecture (with approaches), the Event model, the
severity→colour map, the narration verb table, the cache design + window proposal, the tone proposal,
the marquee mechanics + an **ASCII mock** for HUM LEAD to direct, and the TDD batch breakdown, all under
the same performance/P10/junior-doc gates the seismic feature held.
