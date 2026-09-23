---
title: "0.18.0 — Observer maps — REQUIREMENTS"
date: 2026-09-22
phase: DISCOVER (RCC)
sev: SEV-0
authority: HUM LEAD
status: "NORMATIVE (D-40): on any conflict with the brief or the problem statement, this file wins, and every change is a row in 02-analysis/rulings.md. APPROVED at the DISCOVER gate (D-37). Every requirement carries its phase (D-33); PLAN refines, it does not silently rewrite."
---

# Requirements

**Every requirement carries its macro phase (D-33).** *(P1)* is this release's drawing-and-access
work; *(P2)* is the interaction work that follows — pan and zoom, the map from the Details window,
layers and legends UI, size tiers. **Every requirement below is P1**: the P2 surface has no
requirements yet, by design, and gains them when that phase opens.

Each functional requirement traces to a brief requirement (R-n) and to the ruling or measurement
that makes it necessary. **What, never how** — the how is PLAN's, and PLAN carries no code (D-13).
Every "never" below is stated at the granularity it will be enforced, and each names the instrument
that enforces it; a requirement without one is marked **NO INSTRUMENT YET** so the gap is visible.

## FR-1 — The window (R-1)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-1.1 *(P1)* | `g` opens and closes the map window in the Observer, as a keymap action visible in Help and overridable in `[keys]` | D-15, R-1.1 | Help golden; keymap test |
| FR-1.2 *(P1)* | The map is centred on the selected location; when the selection changes while the window is open, the map follows | R-1.2; W1-C selection | Scripted PTY: move selection, assert title and centre |
| FR-1.3 *(P1)* | **No selection** (before the first snapshot, index out of range) draws a stated state, never a blank or a guess | W1-C: `selectedLocation()` returns nil | Unit test over the nil case |
| FR-1.4 *(P1)* | Below a 69×12 map body the window shows a notice naming the size needed and present, **and carries the FR-7.4 description** — a listener at a large font has fewer cells by definition, and "grow your window" is the one remedy they cannot take (R2 A11y F-3). **One owner** holds the number for the decision and the wording | D-16, R-1.3 | Size sweep: every size from 20×10 to 200×80, each frame either a map ≥ 69×12 or the notice **with words**, never overflow |
| FR-1.5 *(P1)* | The window is a member of every closed-set window test (reachability at 80×24, memo completeness, margins, `modalLines`, glyph survey) | R-1.4; W1-C §5 | The existing closed-set tests fail until it is |
| FR-1.6 *(P1)* | Maps switched off in Settings: `g` says so in one line; no map object is built and nothing is fetched | D-21, R-9.1 | Test: setting off → zero map network calls |
| FR-1.7 *(P1)* | Under `--ascii` the window shows the FR-7.4 description in place of the picture; never a braille cell | D-20, D-26 | `TestASCIIFramesCarryNothingButASCII` extended to the window |
| FR-1.8 *(P1)* | The window states, in text, that the picture is drawn with braille and names the remedy, because the terminal cannot be asked whether its font has it | red team A-3 | Golden of the window's chrome |
| FR-1.9 *(P1)* | A stated **degradation order** — picture, then description, then the size notice — so the window always shows the most useful thing that fits, including below 20×10 where a notice and a description cannot both be drawn (R3 A11y A-7). A **legibility tier** between the floor and comfortable: at a size that passes 69×12 but cannot carry the scale it is asked to draw, the window says what is degraded rather than drawing a blank or stretched frame (a county view at 69×12 with embedded tiles rendered entirely blank in testing) | red team A-6; W1-B | Size sweep asserting a stated state, never an empty body |

## FR-2 — The bound (R-2)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-2.1 *(P1)* | **No frame is drawn wider than the region holding the selected location** — the region set is every US region and territory the app's APIs cover, existing and added (contiguous US, Alaska, Hawaii, the Caribbean and Pacific territories, marine areas) — at any size, before or after placement, across resize, fit, pan, zoom and failure | D-8, **D-28**; W1-B (681°, 197°); red team B1 | **M2 instrument**: every rendered frame checked against the selected location's region, not a single longitude span |
| FR-2.2 *(P1)* | The view is placed before the first frame is drawn | W1-B: unplaced map is the whole world | Test: first frame's span within the bound |
| FR-2.3 *(P1)* | The default scale is a setting (regional / state / county); the regional bound is not | R-2.2 (D-24), D-28 | Settings round-trip test |
| FR-2.5 *(P1)* | A selected location in **no** known region is a stated state, never a silent fall-back to a wider view | D-28 | Unit test over an out-of-region point |
| FR-2.4 *(P1)* | Until go-tuiMaps v0.2.0 provides a bound (HR-3), watchpost enforces it at every view change; once v0.2.0 does, the host sets it once and the library holds it | HR-3; W1 synthesis §4 | M2 instrument, unchanged, both before and after |

## FR-3 — The basemap (R-3)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-3.1 *(P1)* | Network basemap from `https://tiles.openfreemap.org/planet`, with the embedded tiles always passed so a national or state view draws offline | D-21 (library), W1-B | Offline test: state at 69×12 `Complete` with no network |
| FR-3.2 *(P1)* | **No map request of any kind before the listener asks** — `g`, or map functionality from Details (phase 2). No warming, no prefetch. **This includes NWS zone geometry**: `seedZoneShapes` is gated on the maps setting and the first map open (`app/dashboard.go:89` seeds it at station start today, which contradicted D-21 — red team F6) | D-21, **D-25**; OpenFreeMap terms | Test: a full station start with maps on makes zero map requests **of any source, zone geometry included** |
| FR-3.3 *(P1)* | Nothing fetched on the draw path; the library's work runs off the UI loop | C-7, W1-C | Alloc/latency pin on the frame path |
| FR-3.4 *(P1)* | Offline or failing, the window says what it is showing: a county view that can only be a stand-in says the basemap is unavailable, rather than drawing a blank or stretched frame as current | W1-B offline render | Test over `Sharpening` + `tile-failed` |
| FR-3.5 *(P1)* | Caches live under the OS cache directory (`userCacheSubdir("map")`, `("radar")`) with **one stated total** across map, radar and the existing 256 MB HTTP cache | F-49 ruling; W1 synthesis §8 | Test: configured caps sum to the stated total |
| FR-3.6 *(P1)* | The memory tile cache is sized for the largest view the window can draw | W1-B: 909 KB needed vs 500 KB default | Measured at the largest size tier |
| FR-3.7 *(P1)* | Every drawn source is credited at every width; `SourceCredit()`'s HTML is never printed raw. **Credit text is remote-controlled** — it arrives in the TileJSON — so it is length-bounded and neutralised: it may not occupy the chrome a status line uses, and may not read as one (R3 InfoSec S5) | R-3.3; W1-B | Golden at the narrowest width |
| FR-3.8 *(P1)* | Tile and radar sources are a **closed list**, enumerated here, each pinned to an exact scheme and host. No free-form address; a custom source is a later ruling with its own rules and credit line | D-31; red team F5; R2 InfoSec F-1 | **Positive** test: the built list equals this table, member for member — plus the negative test that no path accepts an arbitrary URL |

**FR-3.8's list.** The normative set; anything else is not a source in 0.18.0.

**The effective host is not the configured one.** `https://tiles.openfreemap.org/planet` is a TileJSON *document*; the tiles come from whatever host it names. The library confines that today — a source's tiles must come from the source's own host — and **HR-10** asks v0.2.0 to keep that confinement explicit, because watchpost's positive test checks its own list, not the host a request actually reaches (R3 InfoSec S3).

| Source | Scheme + host | Feeds |
|---|---|---|
| OpenFreeMap | `https://tiles.openfreemap.org` | basemap tiles, and the TileJSON document that names them |
| Iowa Environmental Mesonet | `https://mesonet.agron.iastate.edu` | radar frames |
| NOAA / NCEP | `https://opengeo.ncep.noaa.gov` | radar frames |
| NWS (existing) | `https://api.weather.gov` | alerts and zone geometry |

| FR-3.9 *(P1)* | Caches state a **retention as a number** — map tiles 7 days, radar frames 2 hours (radar's own retention at source) — and can be cleared by the listener. The stated total covers map, radar and the existing 256 MB HTTP cache (FR-3.5 owns the byte total; this row owns age and clearing) | D-31; red team F4/F7/F8; R2 InfoSec F-4 | Test: the radar cache drops frames past its age; a clear path exists and empties both |
| FR-3.10 *(P1)* | **Until HR-8 lands, neither cache's retention is enforceable at the stated granularity, and the record says so.** The radar half is no better than the map half: radar goes through `platform/httpx`, whose disk cache keeps an entry to `max(Expires, mtime) + 24 h` under a byte cap with no per-caller age, and whose `<sha256>.cache` names are **a third copy of the viewed-rectangle record** that FR-3.9's clear path must therefore name too (R3 InfoSec S2). For the map half: the library's `CacheRoot(dir, capBytes)` takes bytes only — no age, no purge — so FR-3.9's map half is a byte cap plus a watchpost-side delete of the directory, and nothing finer | R2 InfoSec F-4 | Test: the clear path removes the tile directory |

## FR-4 — Alert areas (R-4)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-4.1 *(P1)* | Areas come from `resolveAlertAreas` through a `livePipelines` method, reachable from production | C-1 | `TestEveryLivePipelinesMethodIsReachedFromProductionCode` — **extended**, or a sibling, so a free function cannot escape |
| FR-4.2 *(P1)* | One feature per area, first ring outline and the rest holes; severity → role 1:1; valid times and sender carried | R-4.2; handoff §3.1 | Round-trip test against the library's own reading of rings (read the consumer, handoff §4.2) |
| FR-4.3 *(P1)* | Default scope: the station's locations' alerts plus the national severe events in view; scope is a setting | D-23 | Settings test; count of drawn alerts per scope |
| FR-4.4 *(P1)* | A partial area is **never silently** drawn as whole or silently withheld; how it is shown is ruled with the drawing on screen | R-4.3, M4 | **M4 instrument** over `Area.Missing` fixtures |
| FR-4.5 *(P1)* | A scope that would exceed the 512-zone cap is reported as missing, not dropped | D-23; W1-C (515 zones live) | Test at 513 zones |
| FR-4.6 *(P1)* | Held zone shapes are refreshed on a stated rule, or the gap is ruled on | C-2, R-4.4 | NO INSTRUMENT YET — pending ruling in PLAN |

## FR-5 — Radar (R-5)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-5.1 *(P1)* | Radar from IEM and NOAA/NCEP MRMS; the source shown is a setting | D-9, R-5.1 (D-24) | Settings test |
| FR-5.2 *(P1)* | Radar is a loop of frames at absolute times, one overlay per source, never blended | D-10; W1-A traps 3–4 | Fixture test: a loop across a 5-minute rollover has no duplicate or skipped frame |
| FR-5.3 *(P1)* | **A frame is valid only if its time is in the source's advertised times and its image is not empty** — an HTTP 200 is not success; an unset time is never sent | W1-A traps 1–2 | Recorded-response tests: off-grid, expired, empty, 2011-default |
| FR-5.4 *(P1)* | The newest frame's age is always visible; a frame older than its source's cadence is marked stale, never hidden | R-5.3, M3 | **M3 instrument** |
| FR-5.5 *(P1)* | After the first load, a refresh fetches only frames not already held | W1-A: 1 request vs 12 | Request-count test over two refreshes |
| FR-5.6 *(P1)* | Radar lands only against a tagged go-tuiMaps v0.2.0 | D-11 | `go.mod` requires a tagged version; no local replace |
| FR-5.7 *(P1)* | A radar fetch carries an explicit body cap (≤ 1 MiB against measured 7.7–19.5 KB frames), **enforced inside the HTTP client as it reads** — `platform/httpx` caps bodies at a 32 MB package constant and writes to the cache before returning, so a caller-side check fires after the bytes are already held. An over-cap body is refused and **never cached**. A byte cap is not a decode cap: the same clause bounds the decoded image's dimensions, because a megabyte of PNG decodes to an arbitrarily large bitmap (R3 InfoSec S4) | D-31; red team F3; R2 InfoSec F-3 | Test asserting the cache is **empty** after an over-cap response, not merely that the caller errored |
| FR-5.8 *(P1)* | **Map** motion is a Settings row of **off / slow / normal** (D-27, D-35), driven through the library's loop playback control (**HR-9**). Watchpost supplies the data and passes the listener's choice through; the library turns data into frames and plays them, so the rate is set there, not by withholding data from it. With a still form that keeps the newest frame and its age — **and freezes the description with it**, because a description re-derived on every frame advance is a talking surface that never stops (R3 A11y A-4). Switching it off never costs the listener the radar data. The row is labelled for the map, because the Observer animates elsewhere at 300 ms and 50 ms (`modes/tty/dashboard.go:835,897`) and this switch does not reach those — an app-wide motion setting is F-180 | D-27; red team A-7, R2 A11y F-5/F-6 | Settings test; a still-form golden |
| FR-5.9 *(P1)* | A **maximum loop frame rate**, stated as a number, and what each of FR-5.8's three states means in frames a second. **NO INSTRUMENT YET** — set at PLAN once HR-9's control exists: go-tuiMaps' NFR-21 is a *flash* ceiling (2.5 per second), not a loop rate, and loops arrive with v0.2.0 | R2 A11y F-5, R2 PM D1 | NO INSTRUMENT YET |

## FR-6 — Fire and quakes (R-6)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-6.1 *(P1)* | Hotspots, incidents and quakes drawn as static points, each a layer the listener can switch | D-10, R-9.2 | Settings test; golden |

## FR-7 — Without colour (R-7)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-7.1 *(P1)* | Under `NO_COLOR` and the Monochrome and Light themes the map is readable: meaning never by colour alone. **Watchpost hints the library's colour depth from `NO_COLOR` *and* the active theme**, and **never claims colour the terminal cannot be asked about** — the hint overrides the library's own detection, so over SSH into a low-colour terminal a careless hint would switch the hatch off (R2 A11y F-7; detection order is PLAN's) — the library's hatch runs only at `NoColour` depth, so a colour-on monochrome theme got tint and no second channel (red team A-4) | R-7.1, R-12a | Goldens colour-off; invariant: every alert area present has a hatch or label |
| FR-7.2 *(P1)* | Colours come from theme tokens | R-7.2 | Guard: no colour literal in the map's packages |
| FR-7.3 *(P1)* | Coordinates are never spoken, **and the FR-7.4 description is held to the same guard** — it is written to be read aloud by the voice already in the app, so its abbreviations, units and glyphs must speak as well as they read (R3 A11y A-3) | R-7.3, MG-R4, D-26 | The existing speech guard, extended over the description's text |
| FR-7.4 *(P1)* | **A text description ships in this release** (macro phase 1): where the picture cannot be drawn or read — `--ascii`, a font without braille, a screen reader — the window renders `Describe` data as an aggregate of the location's hazard and forecast facts, which the existing voice can speak. It is also the fallback under every failure mode FR-3.4 contemplates | D-26; red team A-1/A-5; go-tuiMaps D-122 | **Acceptance: M1b** — a listener answers the covers / stops short / lies to one side question from the description alone. Scored by M1b and never by M1, whose own rule hides the text and demands the picture answer |
| FR-7.5 *(P1)* | Every palette watchpost hands the library passes the library's colour-vision check **in watchpost's own tests**, stated as a matrix: **each palette × each theme ground × each colour depth watchpost can be in**. A check is relative to a ground and a depth, so "every palette passes" means nothing until those are named | D-27; red team A-2, R2 A11y F-4 | `CheckRamp(ramp, RampCheck{Ground, Depth, Lines, Midpoint})` per cell of that matrix — `Lines` and `Midpoint` decide which rules run (outline work and fills carry different contrast bars), so they are named, not defaulted — keeping findings whose rule is the vision rule (R3 A11y A-8) |
| FR-7.6 *(P1)* | Map colour tokens are declared where the existing contrast register can see them, or carry an exemption with a reason | red team A-9 | The existing token-completeness test |

## FR-8 — Instruments (R-8, `FULL INST`)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-8.1 *(P1)* | A `make verify` gate refuses implementation code in plan documents | D-13, D-14 | The gate itself, with a positive control |
| FR-8.2 *(P1)* | Every map background worker is joined on close; a leak test proves it | W1-C §5 | Goroutine-count test in the style of `TestAScheduleLeavesNoGoroutineBehind` |
| FR-8.3 *(P1)* | The map's frame has an allocation pin; a memo hit does not re-render | W1-C §5 | `AllocBudget` test |
| FR-8.4 *(P1)* | The map's generation is a real Dashboard field so the F-30 guard can see it | W1-C §2 | F-30 guard |
| FR-8.5 *(P1)* | Every golden of the window carries width invariants beside its byte pins (80, 120, 133) | calibration "Byte Pins Ride With Invariant Assertions" | Golden test |
| FR-8.6 *(P1)* | The recorded source responses behind FR-5.3's tests **and behind M1's scenario set** are committed as fixtures before anything depends on them — radar frames *and* the alert responses M1 is scored on. Wave 1's live measurements are not reproducible without them | red team H5; R2 Docs D3 | The fixtures' own presence; tests and the M1 protocol read them, never the network |

## FR-9 — Choice and cost (R-9)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-9.1 *(P1)* | Maps on/off, default scale, layers, alert scope, radar source, map motion **and the description's mode — with the picture, instead of it, or off** — are Settings options, persisted, with builder-chosen defaults. The description must be reachable **without restarting**: `--ascii` is a start-up flag, and a listener who needs words cannot be asked to relaunch (R2 A11y F-2) | D-21, D-23, D-24, D-26 | Settings round-trip; config migration test; a test that the description is reachable with no flag |
| FR-9.2 *(P1)* | When the chosen layers and scope would cost a lot of network, the station says so in plain words | D-23, R-9.3 | NO INSTRUMENT YET — the threshold is PLAN's to set from W1 measurements |
| FR-9.4 *(P1)* | The listener is **told, in the app**, that opening a map sends the rectangle they are viewing to a third party, and which one — the brief's egress table is internal, and FR-9.2 already commits to telling them about network cost; this is the same duty about exposure | R2 Business 3; D-31 | Text test: the disclosure appears on first map open and in Settings beside the maps row, naming each source the session will contact |
| FR-9.3 *(P1)* | A new layer, source or option plugs in without editing the others | P-3, R-9.4 | Architectural test in PLAN (e.g. a registry whose members are discovered, per "Discover Consumers, Don't Enumerate Them") |

# Non-functional requirements

| # | Requirement | Source |
|---|---|---|
| NFR-1 | Time to picture: target **set at PLAN** from wave 1's measured parts (D-29). The library's 1 s warm / 3 s cold is its own figure, not a host measurement, and D-21 forbids warming, so every session's first open is cold | go-tuiMaps NFR-5 (go-tuiMaps v0.1.0 D-30); D-29 |
| NFR-2 | The radar loop holds its configured rate with the Observer live (M6) | D-18 |
| NFR-3 | No map network activity is ever made on behalf of a listener who has not asked for a map | D-21; OpenFreeMap terms |
| NFR-4 | Politeness: requests to IEM, NCEP and NWS stay sequential or bounded, with a descriptive User-Agent; no source is polled faster than its own cadence. Radar and NWS go through `platform/httpx`, which already sets watchpost's own agent. **Basemap tiles are exempt until HR-6 lands, and the cause is narrower than first recorded**: the library *does* expose `Fetcher`, but its type alias resolves to an internal request type that cannot be named outside the module, so a host cannot supply one without reflection. In the interim, tiles go out as the library's own agent, naming neither watchpost nor a contact | W1-A §6; `zones.go:36`; D-31; R2 InfoSec F-5 |
| NFR-5 | Both CI platforms green before merge; `make verify` green before every commit | handoff §5 |
| NFR-6 | No AI attribution in commits, PRs or tracked files | calibration; handoff §5 |

# Risk register

| # | Risk | Severity | Likelihood | Mitigation | Status |
|---|---|---|---|---|---|
| RK-1 | **Correct parts, wrongly connected** — a map wired to nothing, or mutated off the owner goroutine, with tests green | High | High — it happened four times in 0.17.0 and three times in the spike | One owner (the UI loop); reachability extended to free functions (FR-4.1); F-30 field (FR-8.4); scripted PTY on the real binary | Active |
| RK-2 | A frame wider than the selected location's region ships, breaking D-8/D-28 | High | Medium — five paths can widen the view (W1-B) | M2 instrument (FR-2.1); HR-3 in v0.2.0 | Active |
| RK-3 | Radar that looks current and is not — an empty 200, a 2011 default, a stale loop | High | Medium | FR-5.3/5.4 with recorded-response tests; M3 | Active |
| RK-4 | go-tuiMaps v0.2.0 slips, and radar with it | Medium | Medium | Paired releases (D-11): alert drawing proceeds on v0.1.0; radar is the last thing to land. **Ship-without-radar criterion:** alert areas + the FR-7.4 description satisfy the locked problem on their own, so if v0.2.0 is not tagged when the rest is ready, 0.18.0 ships without radar and radar follows in 0.18.1 — a HUM LEAD call at the REVIEW gate, not a slip nobody planned for. **What the criterion costs beyond radar:** shipping on v0.1.0 also means HR-7 and HR-3 are absent, so FR-7.1's pattern invariant cannot hold at colour-on depth and FR-2.1's bound rests entirely on watchpost's own clamp. Both must be stated in the release's exposure record if that path is taken (R3 A11y A-6) | Active |
| RK-5 | **The basemap has exactly one source**, and it is donated infrastructure that may be discontinued at any time — a swap is a release, not a setting (FR-3.8). Radar has two. A degraded-network outbreak, this product's core moment, is when the listener most needs the map and most likely gets a notice | High | Low–Medium | Two radar sources (D-9); FR-3.4's honest offline state; embedded tiles cover a national view. **A basemap fallback is unresolved and is PLAN's first network question** | Active |
| RK-5b | A source changes or disappears — OpenFreeMap "may discontinue", IEM has no SLA, MRMS has no published table | Medium | Low–Medium | Two radar sources (D-9); the source is a setting from a closed list (FR-3.8). **The offline floor is narrower than first written**: embedded tiles stop at zoom 3, which covers a state only at 69×12 and never a county (W1-B), so the default regional/state/county view has no offline floor and FR-3.4 must say so rather than draw a stand-in | Active |
| RK-10 | **What leaves the machine.** A radar request is a bounding box — the rectangle the listener is looking at — and a tile sequence reconstructs a pan trail, sent to two new parties, timestamped | Medium | High (by design, every map open) | The brief's "what leaves the machine" table; nothing fetched before the listener asks (FR-3.2); a closed source list (FR-3.8); stated retention and a purge path (FR-3.9) | Active |
| RK-11 | **A planted source.** The transport policy this rests on is stated rather than assumed: https only, no scheme downgrade, same-origin redirects with the port compared, and private or link-local addresses refused after resolution. Basemap traffic does not pass through watchpost's own client (NFR-4), so the same policy must be asserted of the library's, and pinned by HR-10 (R3 InfoSec S6). Caps and TLS hold, so the harm is not code execution but a believable picture — clear skies drawn over a tornado, which is what this feature exists to prevent | High | Low (closed list) | FR-3.8's closed list; a custom source needs its own ruling | Active |
| RK-12 | **A class of listener excluded.** Braille, colour and motion decisions each remove a group; three of them were found only by a blind reviewer | High | Medium | FR-7.4 description, FR-7.5 vision check, FR-5.8 motion, FR-1.8 braille notice, FR-1.9 legibility tier | Active |
| RK-6 | Partial areas misread as "not me" during an outbreak — the case M1 exists for | High | Low in fair weather, higher in outbreaks (W1-C) | M4 instrument; the partial ruling made with the drawing on screen | Active |
| RK-7 | The Bubble Tea loop stalls under a fast tile or radar pump | Medium | Medium | Render placement decided in PLAN with both options measured (W1 synthesis §12); M6 | Open — PLAN |
| RK-8 | Cache footprint grows unnoticed (HTTP 256 MB + map 256 MB + radar) | Low | High | One stated total (FR-3.5) | Active |
| RK-9 | Scope creep across two SEV-0 releases, and lost context between sessions | Medium | Medium | D-11 discipline: rulings written as made; required reading; learnings as tests | Active |

## Metrics of success — ADOPTED (D-18), normative here (D-40)

M1, **M1b**, M2–M5 primary; M6 secondary. Hardened against gaming in `01-objectives/problem-statement.md` at DISCOVER.

- **M1 — Where is it** *(grader: HUM LEAD; recorded-scenario protocol in `problem-statement.md` §5 — D-29)*. From the map alone, a listener can say whether an active alert covers the
  selected location, stops short of it, or lies to one side.
- **M1b — Where is it, in words** *(adopted D-36; protocol in `problem-statement.md` §5)*. The same
  recorded scenarios as M1, with the picture hidden and the description shown, scored the same three
  ways — the metric FR-7.4 is held to, because M1's own rule hides text and demands the picture.
- **M2 — Never global.** Zero frames drawn wider than the region holding the selected location (D-28), measured by an
  instrument over the test suite and the scripted-PTY journeys.
- **M3 — Radar honesty.** The newest frame's age is always on screen; no frame older than its
  source's cadence is drawn as current.
- **M4 — Partial honesty.** Every alert that did not fully resolve is drawn with that fact stated.
- **M5 — Time to picture** *(target set at PLAN from wave-1 measurements — D-29)*. Seconds from `g`
  to the first **complete** frame. The library's 1 s warm / 3 s cold is its own figure, never measured
  in this host, and D-21 forbids warming, so every first open is cold.
- **M6 — Loop smoothness** *(secondary, D-18)*. The radar loop holds its frame rate without stalling
  the rest of the Observer — the one way D-10 could fail that M1–M5 would not catch.

**M1 is the one this release is most likely to fail**, because most alerts have no polygon of their
own and depend on zone resolution. The field comment says four in five; measured live on 2026-09-22
it was **nine in ten**, inflated by marine advisories and weather-dependent (W1-C). Either way zone
resolution is the ordinary path, not the fallback.

