---
title: "0.18.0 — Observer maps — REQUIREMENTS"
date: 2026-09-22
phase: DISCOVER (RCC)
sev: SEV-0
authority: HUM LEAD
status: "DRAFT — for the DISCOVER-exit red team and HUM LEAD approval"
---

# Requirements

Each functional requirement traces to a brief requirement (R-n) and to the ruling or measurement
that makes it necessary. **What, never how** — the how is PLAN's, and PLAN carries no code (D-13).
Every "never" below is stated at the granularity it will be enforced, and each names the instrument
that enforces it; a requirement without one is marked **NO INSTRUMENT YET** so the gap is visible.

## FR-1 — The window (R-1)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-1.1 | `g` opens and closes the map window in the Observer, as a keymap action visible in Help and overridable in `[keys]` | D-15, R-1.1 | Help golden; keymap test |
| FR-1.2 | The map is centred on the selected location; when the selection changes while the window is open, the map follows | R-1.2; W1-C selection | Scripted PTY: move selection, assert title and centre |
| FR-1.3 | **No selection** (before the first snapshot, index out of range) draws a stated state, never a blank or a guess | W1-C: `selectedLocation()` returns nil | Unit test over the nil case |
| FR-1.4 | Below a 69×12 map body the window shows a notice naming the size needed and present; **one owner** holds the number for the decision and the wording | D-16, R-1.3 | Size sweep: every size from 20×10 to 200×80, each frame either a map ≥ 69×12 or the notice, never overflow |
| FR-1.5 | The window is a member of every closed-set window test (reachability at 80×24, memo completeness, margins, `modalLines`, glyph survey) | R-1.4; W1-C §5 | The existing closed-set tests fail until it is |
| FR-1.6 | Maps switched off in Settings: `g` says so in one line; no map object is built and nothing is fetched | D-21, R-9.1 | Test: setting off → zero map network calls |
| FR-1.7 | Under `--ascii` the window shows a notice in place of the picture; never a braille cell | D-20 | `TestASCIIFramesCarryNothingButASCII` extended to the window |

## FR-2 — The bound (R-2)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-2.1 | **No frame is drawn wider than the contiguous-US bound**, at any size, before or after placement, across resize, fit, pan, zoom and failure | D-8; W1-B (681°, 197°) | **M2 instrument**: every rendered frame's longitude span checked, over the test suite and the PTY journeys |
| FR-2.2 | The view is placed before the first frame is drawn | W1-B: unplaced map is the whole world | Test: first frame's span within the bound |
| FR-2.3 | The default scale is a setting (regional / state / county); the national bound is not | R-2.2 (D-24) | Settings round-trip test |
| FR-2.4 | Until go-tuiMaps v0.2.0 provides a bound (HR-3), watchpost enforces it at every view change; once v0.2.0 does, the host sets it once and the library holds it | HR-3; W1 synthesis §4 | M2 instrument, unchanged, both before and after |

## FR-3 — The basemap (R-3)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-3.1 | Network basemap from `https://tiles.openfreemap.org/planet`, with the embedded tiles always passed so a national or state view draws offline | D-21 (library), W1-B | Offline test: state at 69×12 `Complete` with no network |
| FR-3.2 | **No map request of any kind before the listener asks** — `g`, or map functionality from Details (phase 2). No warming, no prefetch | D-21; OpenFreeMap terms | Test: a full station start with maps on makes zero map requests |
| FR-3.3 | Nothing fetched on the draw path; the library's work runs off the UI loop | C-7, W1-C | Alloc/latency pin on the frame path |
| FR-3.4 | Offline or failing, the window says what it is showing: a county view that can only be a stand-in says the basemap is unavailable, rather than drawing a blank or stretched frame as current | W1-B offline render | Test over `Sharpening` + `tile-failed` |
| FR-3.5 | Caches live under the OS cache directory (`userCacheSubdir("map")`, `("radar")`) with **one stated total** across map, radar and the existing 256 MB HTTP cache | F-49 ruling; W1 synthesis §8 | Test: configured caps sum to the stated total |
| FR-3.6 | The memory tile cache is sized for the largest view the window can draw | W1-B: 909 KB needed vs 500 KB default | Measured at the largest size tier |
| FR-3.7 | Every drawn source is credited at every width; `SourceCredit()`'s HTML is never printed raw | R-3.3; W1-B | Golden at the narrowest width |

## FR-4 — Alert areas (R-4)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-4.1 | Areas come from `resolveAlertAreas` through a `livePipelines` method, reachable from production | C-1 | `TestEveryLivePipelinesMethodIsReachedFromProductionCode` — **extended**, or a sibling, so a free function cannot escape |
| FR-4.2 | One feature per area, first ring outline and the rest holes; severity → role 1:1; valid times and sender carried | R-4.2; handoff §3.1 | Round-trip test against the library's own reading of rings (read the consumer, handoff §4.2) |
| FR-4.3 | Default scope: the station's locations' alerts plus the national severe events in view; scope is a setting | D-23 | Settings test; count of drawn alerts per scope |
| FR-4.4 | A partial area is **never silently** drawn as whole or silently withheld; how it is shown is ruled with the drawing on screen | R-4.3, M4 | **M4 instrument** over `Area.Missing` fixtures |
| FR-4.5 | A scope that would exceed the 512-zone cap is reported as missing, not dropped | D-23; W1-C (515 zones live) | Test at 513 zones |
| FR-4.6 | Held zone shapes are refreshed on a stated rule, or the gap is ruled on | C-2, R-4.4 | NO INSTRUMENT YET — pending ruling in PLAN |

## FR-5 — Radar (R-5)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-5.1 | Radar from IEM and NOAA/NCEP MRMS; the source shown is a setting | D-9, R-5.1 (D-24) | Settings test |
| FR-5.2 | Radar is a loop of frames at absolute times, one overlay per source, never blended | D-10; W1-A traps 3–4 | Fixture test: a loop across a 5-minute rollover has no duplicate or skipped frame |
| FR-5.3 | **A frame is valid only if its time is in the source's advertised times and its image is not empty** — an HTTP 200 is not success; an unset time is never sent | W1-A traps 1–2 | Recorded-response tests: off-grid, expired, empty, 2011-default |
| FR-5.4 | The newest frame's age is always visible; a frame older than its source's cadence is marked stale, never hidden | R-5.3, M3 | **M3 instrument** |
| FR-5.5 | After the first load, a refresh fetches only frames not already held | W1-A: 1 request vs 12 | Request-count test over two refreshes |
| FR-5.6 | Radar lands only against a tagged go-tuiMaps v0.2.0 | D-11 | `go.mod` requires a tagged version; no local replace |

## FR-6 — Fire and quakes (R-6)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-6.1 | Hotspots, incidents and quakes drawn as static points, each a layer the listener can switch | D-10, R-9.2 | Settings test; golden |

## FR-7 — Without colour (R-7)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-7.1 | Under `NO_COLOR` and the Monochrome and Light themes the map is readable: areas hatched, meaning never by colour alone | R-7.1, R-12a | Goldens colour-off; invariant: every alert area present has a hatch or label |
| FR-7.2 | Colours come from theme tokens | R-7.2 | Guard: no colour literal in the map's packages |
| FR-7.3 | Coordinates are never spoken | R-7.3, MG-R4 | Existing speech guard extended |

## FR-8 — Instruments (R-8, `FULL INST`)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-8.1 | A `make verify` gate refuses implementation code in plan documents | D-13, D-14 | The gate itself, with a positive control |
| FR-8.2 | Every map background worker is joined on close; a leak test proves it | W1-C §5 | Goroutine-count test in the style of `TestAScheduleLeavesNoGoroutineBehind` |
| FR-8.3 | The map's frame has an allocation pin; a memo hit does not re-render | W1-C §5 | `AllocBudget` test |
| FR-8.4 | The map's generation is a real Dashboard field so the F-30 guard can see it | W1-C §2 | F-30 guard |
| FR-8.5 | Every golden of the window carries width invariants beside its byte pins (80, 120, 133) | calibration "Byte Pins Ride With Invariant Assertions" | Golden test |

## FR-9 — Choice and cost (R-9)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-9.1 | Maps on/off, default scale, layers, alert scope and radar source are Settings options, persisted, with builder-chosen defaults | D-21, D-23, D-24 | Settings round-trip; config migration test |
| FR-9.2 | When the chosen layers and scope would cost a lot of network, the station says so in plain words | D-23, R-9.3 | NO INSTRUMENT YET — the threshold is PLAN's to set from W1 measurements |
| FR-9.3 | A new layer, source or option plugs in without editing the others | P-3, R-9.4 | Architectural test in PLAN (e.g. a registry whose members are discovered, per "Discover Consumers, Don't Enumerate Them") |

# Non-functional requirements

| # | Requirement | Source |
|---|---|---|
| NFR-1 | Time to picture ≤ 1 s warm, ≤ 3 s cold (M5) | go-tuiMaps NFR-5 |
| NFR-2 | The radar loop holds its configured rate with the Observer live (M6) | D-18 |
| NFR-3 | No map network activity is ever made on behalf of a listener who has not asked for a map | D-21; OpenFreeMap terms |
| NFR-4 | Politeness: requests to IEM, NCEP and NWS stay sequential or bounded, with a descriptive User-Agent; no source is polled faster than its own cadence | W1-A §6; zones store's own rule (`zones.go:36`) |
| NFR-5 | Both CI platforms green before merge; `make verify` green before every commit | handoff §5 |
| NFR-6 | No AI attribution in commits, PRs or tracked files | calibration; handoff §5 |

# Risk register

| # | Risk | Severity | Likelihood | Mitigation | Status |
|---|---|---|---|---|---|
| RK-1 | **Correct parts, wrongly connected** — a map wired to nothing, or mutated off the owner goroutine, with tests green | High | High — it happened four times in 0.17.0 and three times in the spike | One owner (the UI loop); reachability extended to free functions (FR-4.1); F-30 field (FR-8.4); scripted PTY on the real binary | Active |
| RK-2 | A frame wider than the US ships, breaking D-8 | High | Medium — five paths can widen the view (W1-B) | M2 instrument (FR-2.1); HR-3 in v0.2.0 | Active |
| RK-3 | Radar that looks current and is not — an empty 200, a 2011 default, a stale loop | High | Medium | FR-5.3/5.4 with recorded-response tests; M3 | Active |
| RK-4 | go-tuiMaps v0.2.0 slips, and radar with it | Medium | Medium | Paired releases (D-11): alert drawing proceeds on v0.1.0; radar is the last thing to land | Active |
| RK-5 | A source changes or disappears — OpenFreeMap "may discontinue", IEM has no SLA, MRMS has no published table | Medium | Low–Medium | Two radar sources (D-9); embedded tiles for national/state; the source is a setting (P-3) | Active |
| RK-6 | Partial areas misread as "not me" during an outbreak — the case M1 exists for | High | Low in fair weather, higher in outbreaks (W1-C) | M4 instrument; the partial ruling made with the drawing on screen | Active |
| RK-7 | The Bubble Tea loop stalls under a fast tile or radar pump | Medium | Medium | Render placement decided in PLAN with both options measured (W1 synthesis §12); M6 | Open — PLAN |
| RK-8 | Cache footprint grows unnoticed (HTTP 256 MB + map 256 MB + radar) | Low | High | One stated total (FR-3.5) | Active |
| RK-9 | Scope creep across two SEV-0 releases, and lost context between sessions | Medium | Medium | D-11 discipline: rulings written as made; required reading; learnings as tests | Active |
