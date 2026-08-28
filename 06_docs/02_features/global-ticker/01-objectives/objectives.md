# Global event ticker — objectives (DISCOVER / RCC)

**Feature:** `global-ticker` · **Target:** `0.12.0` · **Status:** DISCOVER (context gathering) —
**awaiting HUM LEAD decisions (§6) before PLAN.**
**Owner:** HUM LEAD (Branden Thompson) · **Opened:** 2026-08-27 (right after 0.11.0 shipped)

## 1. Intent (deferred from the seismic 0.11.0 objectives)

> A global significant-quake ticker … belongs with a broader **"Global Severe / Catastrophic Weather
> Event ticker"** (hurricanes, typhoons, tropical storms, tornado warnings, damaging weather) that needs
> its own discovery — a **0.12.0** candidate.

The through-line: watchpost so far answers *"what's happening near MY tracked locations?"* The ticker
answers a different question — *"what major hazard events are happening in the world right now?"* — a
global awareness band, not tied to the watchlist.

## 2. The user need (to confirm with HUM LEAD)

A glanceable, rotating summary of the largest active hazard events worldwide — the significant
earthquakes and the major storms someone following weather/geology would want to know are happening,
independent of where they track. Reassurance-or-awareness at a world scale, the complement to the
per-location sections.

## 3. Data-source reconnaissance (DATA FIRST — live-probed 2026-08-27)

The word "global" is bounded by which authoritative, keyless feeds actually exist. What I found:

| Event class | Feed (keyless) | Coverage | Live sample |
|---|---|---|---|
| **Significant earthquakes** | USGS `summary/significant_week.geojson` (and `4.5_week`) | **Global** | 3 events: M5.8 Japan, M5.2 Nepal, M4.9 Afghanistan (2.5 KB) |
| **Tropical cyclones** | NHC `CurrentStorms.json` | **Atlantic + E-Pacific only** | 2 active: TS Dolly, TS Lala (8.7 KB, name/class/intensity/lat-lon) |
| **W-Pacific typhoons** | *not in NHC* — JTWC / JMA | W-Pacific | needs a separate, less-standard source (research needed) |
| **Tornado / severe / damaging wx** | NWS `alerts/active` | **US only** | NWS is a US agency; no global severe-weather feed exists |

**The finding that shapes scope:** three of the four intended classes have clean global-or-regional
keyless feeds; **"global severe/tornado weather" has no single authoritative source** (NWS is US-only,
and there is no worldwide tornado/damaging-wind feed). So a literally-global severe-weather ticker is
not buildable from authoritative free data. Realistic options are in §6.

## 4. Scope candidates (for HUM LEAD to choose in §6)

- **A — Quakes + tropical cyclones only, truly global-ish.** USGS significant (global) + NHC storms
  (Atlantic/E-Pacific), optionally a W-Pacific source later. Honest about coverage; no US-only bias.
- **B — A + US severe/tornado.** Add NWS tornado/severe warnings as a US layer, labelled as US. Mixes a
  global layer with a US layer (uneven coverage, but covers the intent's "tornado warnings").
- **C — Significant quakes only (v1), storms/weather later.** The smallest honest first cut, reusing the
  USGS provider; the weather layers become follow-on batches once sources are settled.

## 5. Open design questions (DISCOVER — to resolve before PLAN)

1. **UI form & placement.** What IS the ticker? A horizontal scrolling marquee (like the radio station
   name)? A rotating single-line banner in the header? A dedicated row above/below the table? A modal/
   view? How many events visible at once; rotation cadence?
2. **Ranking & rotation.** Order by severity, magnitude, recency, or a blended score? How many events
   before it cycles? Newest-first or largest-first?
3. **Global vs watchlist relationship.** Confirm the ticker is world-wide and NOT filtered to the
   watchlist (a distinct "world view" band), vs. a "nearby major events" variant.
4. **Radio (R6).** Is the ticker narrated in the synth broadcast (a "global events" segment), or
   display-only? (R6 gate applies if narrated.)
5. **Refresh cadence.** USGS significant updates ~minutes; NHC storms every few hours. One shared tier or
   per-source?
6. **Level / SEV / directives.** Seismic was LEVEL-1 / SEV-0 / FULL everything. Same for the ticker?

## 6. Decisions — RATIFIED by HUM LEAD (2026-08-27)

- **D1 — Scope B:** significant earthquakes (global, USGS) + tropical cyclones (NHC, Atlantic/E-Pacific)
  + **US severe/tornado warnings (NWS, US-only — that is fine)**. W-Pacific typhoons remain a later add.
- **D2/D3 — A single-row scrolling marquee** with:
  1. A **marquee cycle that rotates per severe-weather event category** (quakes · tropical · US severe).
  2. **Background colour by event type / severity — Red / Orange / Yellow** (most→least severe).
  3. An **`[M]` Mute/Unmute** control for the narration **and** the severe-alert tone.
  4. **Narration kept general** — the tone leads, then a two-second pause, then one template
     (HUM LEAD sharpened 2026-08-27):
     > `<tone><tone><tone>`  ⟨2 s⟩  "A(n) `<severe alert type>` has been `<declared | recorded |
     > reported>` for `<location>`. Play the Watchpost Radio Broadcast of that location for more details."
     - Verb by class: weather warning → *declared*, earthquake → *recorded*, storm → *reported*.
     - **A(n):** the article agrees with the alert type ("A Tornado Warning", "An Earthquake").
  5. A **severe-alert tone on a NEW alert only** → the ticker must **cache seen alerts for the last
     [x] days** so "new" is detectable across restarts.
  6. The marquee is a **STACK — most-recent-first; a new alert injects at the TOP and immediately
     interrupts the event currently scrolling** (a "breaking news" marquee).
  7. **Background colours are theme-independent** (the Red/Orange/Yellow are fixed) **except the
     monochrome theme** (which renders them greyscale, like every other mark).
- **D4 — LEVEL-1 · SEV-0 · FULL GIT · FULL DOCS · FULL REPORTS · FULL RCC · FULL PLAN · FULL TDD.**
- **D5 — One event, one ticker entry — dedup + location tying (HUM LEAD 2026-08-27).** A single global
  event (e.g. one earthquake felt by several tracked locations) must **not** repeat N times in the
  ticker. Tie each event to a single representative location, resolved in this order:
  1. the **highest location in the watchlist** it applies to (if any), else
  2. the **nearest explicit city named in the alert**, else
  3. a **fuzzy area phrase** — "the San Diego area", "the Los Angeles Metro Area" — when no single city
     is clean.
  The ticker de-duplicates on the underlying event id first (one entry per source event), and the tied
  location is what the narration and the marquee text name.

### Design details PLAN will propose (with HUM LEAD to direct)

The severity→colour mapping per class, the narration verb table, the cache window `[x]`, the tone's
character (a generated EAS-like tone vs. a short chime), the marquee's exact placement/row, the
new-alert identity keys per source, and the scroll/interrupt mechanics — all carried into PLAN as
proposals with an ASCII mock.

## 7. Non-goals (proposed)

Not a forecasting or tracking tool (no storm cones, no ShakeMaps); not alerting/notifications; not a
replacement for the per-location hazard sections; no paid or scraped sources. One glanceable world band.
