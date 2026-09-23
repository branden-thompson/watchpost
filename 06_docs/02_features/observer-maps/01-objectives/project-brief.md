---
title: "0.18.0 — Observer maps — PROJECT BRIEF"
date: 2026-09-22
phase: DISCOVER (RCC)
report_template: project-brief v1.1.0
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
directives: FULL GIT; FULL DOCS; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD; FULL INST
branch: feature/map-drawing
paired_release: go-tuiMaps v0.2.0 (D-11)
issue: "branden-thompson/watchpost#22 — this brief is its body (D-19)"
status: "APPROVED by the HUM LEAD 2026-09-22 (D-17); amended and approved at D-24; amended again after the DISCOVER-exit red team rounds 1 and 2 (D-25..D-34) — the bound, the description, colour and motion, the phase boundary, the threat model.  Problem statement LOCKED (D-6).  Metrics ADOPTED (D-18).  Rulings in 02-analysis/rulings.md."
---

# New Major System Feature | `Observer-Maps`

**LEVEL-1; SEV-0; FULL GIT; FULL DOCS; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD; FULL INST**

## Summary & Intent

Watchpost can tell a listener that an alert applies to a place. It cannot show them where it is.

0.18.0 puts a map in the **Observer**. Pressing `g` opens a window, centred on the location the
listener has selected, that draws the alert areas over that place, the radar as a moving loop, and
fire and quake points, at a regional, state or county scale and **never wider than the region that
location sits in** — the contiguous US, Alaska, Hawaii, a territory or a marine area (D-28).

**Why now.** The pieces exist and were built for this release by name. go-tuiMaps v0.1.0 is
published and resolvable. 0.17.0 made alert geometry map-ready and deliberately drew nothing:
`resolveAlertAreas` returns every alert's ground and the parts of it that could not be fetched, and
has no production caller. A spike drew a map in an Observer window and found the library's first
defect. What remains is the integration itself, and the two things the spike could not reach: real
alert areas, and radar.

**Who benefits.** The **listener watching the weather**, who today reads zone names and pictures the
distances themselves. The Broadcaster's operator is a later beneficiary (F-174, likely 0.19.0) and
is out of this release.

**What happens if this is not built.** The library's most designed feature is never driven by a
real host, and a listener in a partially-covered county keeps judging from text whether the warning
reaches them.

**Two releases, one outcome (D-11).** Radar loops and the NOAA colour table are library work.
go-tuiMaps **v0.2.0** gets its own brief and ships first; this brief states what the host needs from
it (§ Host requirements). Alert drawing may proceed against v0.1.0 in parallel; radar lands after
v0.2.0 is tagged.

## Problem Statement — LOCKED

> **"A person watching the weather in Watchpost is told that an alert applies to a place, but cannot
> see where that alert actually is: whether its area covers them, stops short of them, or lies to one
> side, so they judge how close the danger is from zone names and distances they have to picture for
> themselves."**

**LOCKED by the HUM LEAD, 2026-09-22** (*"Problem Statement A approved"*, D-6). Every scope
argument in this release traces to this sentence. Radar is in scope as the view the HUM LEAD names
the most common (D-7); fire and quake points as supporting context.

## Requirements

Numbered so DISCOVER and PLAN can trace each one. **Macro phase 1 (drawing) and macro phase 2 (how
the maps work) are split in PLAN, not here (D-12).**

### R-1 — The map window

- **R-1.1** `g` opens the map window in the Observer (D-15), bound as a `defaultKeyMap` action so
  Help and `[keys]` overrides see it. The Broadcaster does not get it in this release.
- **R-1.2** The map is centred on the **selected** location (`selectedLocation()`), not the first
  one — the spike's location-0 centring was a finding, not a design.
- **R-1.3** The map body is never smaller than **69×12 cells** (D-16; go-tuiMaps NFR-7). Below it,
  the window shows a btop-style notice naming the size needed and the size present. **One owner**
  holds the number, used both to decide and to word the notice.
- **R-1.4** The window registers with every closed-set window test the Observer already has
  (reachability, memo completeness, `modalLines`, glyph survey).
- **R-1.5** A build or run with no map says so in one line; a map that fails to build never stops
  the station starting.

### R-2 — Scale bounds, owned by the host (D-8)

- **R-2.1** Watchpost **never** shows a view wider than the region holding the selected location —
  every US region and territory its APIs cover, marine areas included (**D-28**, superseding the
  single US-centred rectangle). No frame is ever drawn at global or continental scale, including
  after a resize, a fit, or a failure.
- **R-2.2** The default scale — regional, state or county around the selected location — is a
  **setting**; the US-national bound of R-2.1 is not (D-8 is a hard constraint, D-23 P-2).
- **R-2.3** The bound is **enforced by a test that fails**, not by a convention (D-11).

### R-3 — The basemap

- **R-3.1** State and county views use the network basemap (OpenFreeMap, go-tuiMaps D-21) with the
  library's disk cache, because the embedded tiles stop at zoom 3.
- **R-3.2** Nothing is fetched on the draw path. Offline, the map says what it is showing rather
  than drawing stand-in tiles as if they were current.
- **R-3.3** Every source drawn is credited, in the window or its chrome, at every width the window
  can be.

### R-4 — Alert areas

- **R-4.1** Alert areas come from `resolveAlertAreas`, reached through a **`livePipelines`
  method**, so the reachability gate sees the wiring (C-1).
- **R-4.2** One feature per area, never one per hazard; the first ring an outline, the rest holes
  (`tuimaps.Rings` per polygon). Severity maps to role 1:1; valid times and sender carried through.
- **R-4.3** **The partial-area decision is made with the drawing on screen** (handoff §3.2, MG-10).
  The library has no channel for "incomplete", so whatever is decided is watchpost's to show.
- **R-4.4** Zone shapes held for a long session are not trusted forever (C-2): refreshed, or the
  gap ruled on.

### R-5 — Radar and precipitation, as a loop

- **R-5.1** Radar is drawn from **both** the Iowa Environmental Mesonet and NOAA/NCEP MRMS (D-9).
  Which source is shown is a **setting**; PLAN picks the default (D-23 P-2).
- **R-5.2** Radar is a **loop** (D-10). A single frame is not the product.
- **R-5.3** The age of the newest frame is visible; old radar never looks current.
- **R-5.4** Colour-to-intensity tables live in go-tuiMaps, including MRMS's (D-9).

### R-6 — Fire and quake points

- **R-6.1** Fire hotspots, named incidents and quakes are drawn as points. **Static** — no loops
  unless a real use case appears (D-10).

### R-7 — Reading the map without colour

- **R-7.1** Meaning never depends on colour alone (render rule R-12a): `NO_COLOR` and the Monochrome
  and Light themes all produce a readable map, with the colour-depth hint following the theme as well
  as the environment (D-27). **`--ascii` draws no braille at all** (D-20): it gets the R-7.4
  description instead, which is a path, not a lesser map.
- **R-7.4** A **text description ships in this release** wherever the picture cannot be drawn or read —
  `--ascii`, a font without braille, a screen reader — as an aggregate of the location's hazard and
  forecast facts, speakable by the voice already in the app (**D-26**).
- **R-7.5** Every palette watchpost hands the library passes its colour-vision check, per theme ground
  and colour depth, in watchpost's own tests (**D-27**).
- **R-7.6** Map colour tokens sit where the existing contrast register can see them.
- **R-7.2** Colours come from theme tokens, never literals.
- **R-7.3** Coordinates are never spoken (MG-R4).

### R-8 — Learnings become instruments (D-11, D-14)

- **R-8.1** A `make verify` gate refuses implementation code in plan documents (D-13, D-14).
- **R-8.2** Every lesson this release learns is recorded as a test or gate that fails where
  possible, and as prose only where it cannot be.

### R-9 — The listener chooses; the station says what it costs (D-21, D-23)

- **R-9.1** Maps on/off is a Settings option (`s`), persisted, default on (D-21).
- **R-9.2** Which layers are shown is the listener's setting: alert areas, radar, fire, quakes. So is
  which alerts: the station's locations only, plus national severe events in view, or more. The
  default is D-23's C.
- **R-9.3** Where a choice costs a lot of network, the station says so in plain words, before or as it
  happens. Where a hard limit binds (the 512-zone cap, a source's stated limit), it reports what was
  left out (M4) and never drops anything silently.
- **R-9.4** Defaults are chosen by the builders; the seams are built so a new layer, source or option
  plugs in without rework (P-3).

## Host requirements on go-tuiMaps v0.2.0

What watchpost needs from the paired release. v0.2.0's own brief decides how.

| # | Need | Why | From |
|---|---|---|---|
| **HR-1** | Image loops: an image overlay with several frames, each with its own valid time | R-5.2 | FR-37; contract §9 names it additive |
| **HR-2** | A published colour-to-intensity table for MRMS `conus_bref_qcd` | R-5.1, R-5.4 | D-9; plan-entry-checks §1 says none is published |
| **HR-3** | A way for the host to bound the view, or a guarantee the library never falls back to the whole world once the host has placed it | R-2.1 | `map.go:265`; `MinZoom` is a constant |
| **HR-4** | The contract document matches the code | a host reads the contract | `contract.md:103,114` name calls that do not exist |
| **HR-5** | The after-tag triage D-123 names, written down | nineteen items with no record | D-123; not found in the tree |
| **HR-6** | A host-settable fetcher: export the request type and its options (User-Agent token, allowed hosts, timeout, roots) | the library opens its own HTTP client (`tiles.go:52`), bypassing watchpost's single door, so NFR-4's User-Agent is unachievable and a host cannot supply a fetcher without reflection | red team F2; W1-B side finding |
| **HR-7** | An alert pattern that does not depend on colour depth, and Extreme/Severe distinguishable by more than hatch spacing | the hatch runs only at `NoColour` depth (`frame.go:513`), so a colour-on monochrome theme carries no second channel; Extreme and Severe share `╳` at stride 2 vs 3 | D-27; red team A-4 |
| **HR-8** | A host-settable **max age** and a **purge** call on the tile cache | `CacheRoot(dir, capBytes)` takes bytes only and the cache keeps tiles "without expiry", so FR-3.9's retention is unbuildable for the source that motivated it — and the file names are a record of where the listener looked | R2 InfoSec F-4 |
| **HR-9** | **Loop-rate control**, and a stated meaning for `ReduceMotion` over frames | `ReduceMotion` is a boolean about markers and the clock; loops arrive with FR-37, and nothing yet says what "reduce motion" does to a twelve-frame loop, so FR-5.9's ceiling has no mechanism | R2 A11y F-5; R2 PM D1 |

## What leaves the machine (D-31)

This is the first watchpost feature where a third party learns **where the listener is looking**,
continuously. Stated here because the risk register carried nine rows and none of them were security
or privacy (red team F1).

| Source | Request | What it reveals | When |
|---|---|---|---|
| OpenFreeMap | `{z}/{x}/{y}` vector tiles | IP, plus a tile path that at county zoom is a few kilometres square; a sequence reconstructs a pan trail | Only after the listener opens a map (FR-3.2) |
| IEM / NOAA MRMS | WMS `GetMap` with a `BBOX` | IP, plus **the exact rectangle on screen**, timestamped per frame | Only with radar on, after a map is opened |
| NWS (existing) | zone GeoJSON, alerts | IP, plus which zones the listener's locations sit in — coarser, and already true today | Zone geometry is gated on the maps setting (D-25) |

Not new ground: watchpost already fetches from NASA FIRMS, Open-Meteo geocoding and others, and the
credential redaction those use is the precedent this extends (red team F9's correction).

## Metrics of Success — ADOPTED (D-18)

M1–M5 primary, M6 secondary. Hardened against gaming in `01-objectives/problem-statement.md` at DISCOVER.

- **M1 — Where is it** *(grader: HUM LEAD; recorded-scenario protocol in `problem-statement.md` §5 — D-29)*. From the map alone, a listener can say whether an active alert covers the
  selected location, stops short of it, or lies to one side.
- **M2 — Never global.** Zero frames drawn wider than the US-national bound, measured by an
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

## Technical Constraints

**Measured against the tree at `17e40d4` (watchpost) and the `v0.1.0` tag `bbc039a` (go-tuiMaps), not
assumed.** (`1cac1ce` was cited first: it exists only on go-tuiMaps' local `release/v0.1.0` and is
not what a reader would check out — red team H4.)

### C-1 — `resolveAlertAreas` is unwired, and the guard named to catch that cannot see it
It has no production caller (`app/mapgeometry.go:28`; called only from `mapgeometry_test.go`). It is
a **free function**, and `TestEveryLivePipelinesMethodIsReachedFromProductionCode`
(`app/reachable_test.go:31`) checks only methods on `*livePipelines`. **The handoff's §6 step 3 is
wrong on this point** and is corrected here.

### C-2 — Held zone shapes are never refreshed while the program runs
`Zone()` returns a held shape without refetching (`domains/weather/nws/zones/zones.go:116-124`).
The 4.9-day `Cache-Control` rule (MG-11) applies only after eviction, so a session left running for
days draws day-one zones.

### C-3 — The library can fall back to the whole world, and has no host bound
A view that fails validation at a new size becomes `project.WholeWorld` (`go-tuimaps/map.go:265`);
`MinZoom` is a package constant (`go-tuimaps/view.go:16`). R-2 cannot be met by configuration today.

### C-4 — Offline tiles stop at zoom 3
The embedded assets are zoom 0–3 (go-tuiMaps D-33, `README.md:56`). Nothing is fetched until
`Source` is called (`tiles.go:30-34`).

### C-5 — Watchpost has no radar source, and v0.1.0 draws one frame
No package fetches radar imagery; "radar" appears only as alert detection wording
(`domains/severe/detection.go`). The library's `RadarImage` takes one image (`overlays.go:108`);
loops are deferred (contract §9).

### C-6 — The library has no way to say "incomplete"
`Answer` carries `NoData` and `Stale`, nothing for partial data; `Overlay` has no field for it.

### C-7 — The map cannot be mutated from `View`
Watchpost renders from a value receiver; every library mutator, and `Render` itself, is an owner
call made one at a time. The spike ran recentring on the publish goroutine and could not recentre
on selection (`63903d7:modes/tty/modal_map.go:37-46`).

### C-8 — The Observer has no minimum-size gate; the Broadcaster has one
`tooSmall()` and its notice exist for the Broadcaster (`modes/tty/broadcaster.go:643-664`); the
Observer has none. A modal body is `(W−11)×(H−12)` at most (`view.go:121,363`) — exactly 69×12 at
80×24.

### C-9 — Four alerts in five have no polygon
Stated at the field (`platform/snapshot/types.go:195-197`). Zone resolution is the ordinary path,
not the fallback.

### C-10 — Where the code may live
`modes/` and `platform/` may not import `domains/` (`scripts/lint-imports.sh`). A map wrapper may
live in `platform/`, the window in `modes/tty`, and zone resolution stays in `app/`.

### C-11 — The library is not yet a dependency
`go.mod` has no `require` for go-tuiMaps; the spike used an uncommitted local path, the era that hid
a defect for a day (handoff §4.2).

### C-12 — The library's documents have drifted from its code
`contract.md` names `Frame.Line`, `WriteTo`, `SetSize`, `Focused`, `BorrowCheck`; the public
surface (`07-readiness/public-surface.txt`) has none of them. The draw call is `Render(size, now)`.

## Other Considerations

- **The map's look belongs to PLAN, ratified from rendered specimens (D-32).** This record carries no
  mock on purpose: a hand-drawn mock of a braille picture is fiction, and the calibration requires a
  visual choice to be ratified from an actual rendering. The window follows the existing modal
  pattern, with control commands along the bottom.
- **Macro phase 2 is an input to PLAN, not a commitment here (D-12):** a map from the Details
  window, pan and zoom keys, layers and legends, a spoken description (`Describe`), settings, size
  tiers.
- **The Broadcaster's one map is F-174**, likely 0.19.0. R-7 of 0.16.0 already makes the tower's
  location a first-class value for it.
- **The spike is evidence, not a base.** Reusable: the two-goroutine pump with a coalescing wake,
  `p.Send` as the only redraw trigger, the change counter in the memo key, "draw to the box", the
  resize regression test. Throwaway: fires-only, `Zoom(6)`, location 0, recentring on every publish.
  `spike/tuimaps-first-host` is deleted once 0.18.0 ships.
- **Owed to the HUM LEAD from v0.1.0, and both land here:** the first-host review (WP-14.19), and
  the screen-reader half of WP-14.17 — which go-tuiMaps D-122 **redirected to 14.19**, "where a host
  renders `Answer` values to its platform's accessibility layer" (PL-AX-3, "a redirection, not a
  waiver"). So this release owes the accessibility test of `Describe` output, whichever macro phase
  PLAN puts `Describe` in.
- **Standing principles (D-23), for every user-facing feature:** **P-1** the builders optimise for
  performance and structure; **P-2** default to user choice and settings unless a hard constraint
  forbids it; **P-3** never architect into a corner — maintainability and extensibility are
  first-class, always. Interactive toggles live in Settings; a flag only on a one-shot CLI command —
  `--no-map` is F-175, for spot reports (D-22).
- **Standing rules carried:** rulings one at a time; silence is not consent; a gate is obeyed or
  ruled on; blind red-team; both CI platforms green before merge; no AI attribution; no code in
  PLAN (D-13).
- **Issue of record.** This brief is the body of issue #22 (D-19); the release PR closes it.

## Completeness Check

```
PROJECT BRIEF — COMPLETENESS CHECK
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  [✓] Header              — Scope, type, name, paired release
  [✓] Directives          — LEVEL-1, SEV-0, eight phase directives (D-5)
  [✓] Summary / Intent    — What, why now, who benefits, cost of not building
  [✓] Problem statement   — LOCKED (D-6)
  [✓] Requirements        — 9 families, R-1..R-9 (R-9 by amendment D-24)
  [✓] Host requirements   — HR-1..HR-9 for go-tuiMaps v0.2.0
  [✓] Metrics of Success  — M1–M5 primary, M6 secondary (D-18); hardened at DISCOVER
  [✓] Tech Constraints    — 12, measured at 17e40d4 / bbc039a (the v0.1.0 tag)
  [✓] Considerations      — phase 2 inputs, F-174, spike, owed items, issue of record

  [✓] Rulings             — D-1..D-32, verbatim, in 02-analysis/rulings.md
  [✓] Approval            — APPROVED as presented (D-17)
  [✓] Issue of record     — #22 (D-19)
  [✓] Outstanding         — none.  INTAKE CLOSED.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```
