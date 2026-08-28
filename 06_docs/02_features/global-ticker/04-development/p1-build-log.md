# P1 build log — Global ticker data layer (DATA FIRST)

**Batch:** P1 of the global-ticker PLAN (`03-architecture-design/plan.md` §6). **SEV-0** · FULL TDD.
**Branch:** `feature/global-ticker`. **Status:** at gate — unit + live green; one exemption decision for
HUM LEAD (§5).

## 1. What landed (junior-first)

The data layer for the world-events ticker, before any UI or audio — a **separate pipeline** from the
per-location snapshot (`domains/globalfeed`), because these events belong to no tracked location.

- **One Event model across three feeds** (`event.go`): `Event { ID, Class, Severity, Type, Place,
  Location, Lat, Lon, At, Source }`. `Class` (Quake / Tropical / SevereWx) drives the narration `Verb()`
  (recorded / reported / declared); `Article()` picks A/An by the type's initial sound; `Severity`
  (Yellow < Orange < Red) is the bg tier and the stack tiebreak.
- **Three keyless fetchers**, each mapping its feed to `[]Event`:
  - `usgs.go` — the **significant-quakes** summary feed (global); severity by magnitude (≥6.5 or any
    tsunami → Red, 5.5–6.5 → Orange, else Yellow).
  - `nhc.go` — **tropical cyclones** (NHC CurrentStorms, Atlantic + E-Pacific); HU Red / TS Orange / TD
    Yellow; basin from the id prefix; post-tropical/lows skipped.
  - `nws.go` — **US severe/tornado** (national active alerts filtered to a curated event list); Tornado/
    Extreme-Wind/Hurricane Warning Red, Severe-Tstm/Flash-Flood Orange, watches Yellow.
- **The stack** (`stack.go`): `Merge` dedups by source id (one entry per event — a quake felt by many
  locations is still one USGS id, so the global feed never repeats it, **D5**), sorts most-recent-then-
  most-severe, caps at `MaxEvents = 30`, and splits **new vs seen** (the P3 tone/narration fire on new).
- **D5 location tying** (`locate.go`): resolve one representative place — the **highest watchlist
  location within 150 km**, else the feed's **named place** (cleaned after "… of"), else a **fuzzy
  "the <metro> area"** from an injected nearest-city fn. One event, one name.

## 2. Tests first (the pins)

| Pin | Guards |
|---|---|
| `TestVerbAndArticleByClass` | recorded/reported/declared by class; A/An by sound |
| `TestSeverityMaps` | quake/tropical/severe → Red/Orange/Yellow; post-tropical & unknown handled |
| `TestSortMostRecentThenSevere` | stack order: recency, then severity |
| `TestMergeDedupsAndSplitsNewVsSeen` | dedup by id, id-less dropped, cap, new-vs-seen split |
| `TestLocateTiesToHighestWatchlistThenPlaceThenArea` | the D5 three-step resolution |
| `TestUSGSFetchMapsSeverityAndFields` / `NHC…` / `NWS…` | each fetcher maps fields/severity against fixtures |
| `TestLiveGlobalFeeds` (`WATCHPOST_LIVE=1`, CI-skipped) | all three real feeds answer and map |

**Live proof (2026-08-27):** USGS 3 significant events (M-Nepal/Afghanistan/Japan), NHC 2 tropical storms
(Atlantic + Central Pacific), **NWS 40 severe events** (active US weather). The stack caps that to 30.

**NWS query — three probed quirks** (nailed live): the `severity=` multi-param 400s; **repeated `event=`**
params 400 (use one **comma-joined** param); **`event`+`limit` together 400** (drop `limit` — the event
filter bounds it); and NWS needs **`%20`** for spaces, not the `+` `url.Values` emits.

## 3. Performance (the mandate)

- No render/audio path touched (P1 is data only). Bounds stated: the stack caps at 30 (`MaxEvents`);
  each fetch is a single global feed on its own TTL (USGS 5 min / NHC 30 min / NWS 2 min), reusing
  `httpx` (cache, conditional GET, byte discipline). The per-hour byte floor is measured at P3.
- The NWS national feed can be large in an outbreak; the curated `event=` filter + the stack cap bound it.

## 4. Gate

| Check | Result |
|---|---|
| `go test ./domains/globalfeed/...` | green (unit + live proof) |
| `make verify` | ALL GATES GREEN |
| full suite | green |
| `make p10` | **0 live · 0 unmatched** after the one package exemption (§5); ledger 111 → **112** |

## 5. Decision for HUM LEAD (the one exemption)

One **package P10-05 (invariant-density)** row for the new `domains/globalfeed`: the three fetchers and
the Event mappings (severity/type/basin/verb/article, Sort, cleanPlace, Locate) are **pure decode-and-
classify** where a quota assertion asserts nothing; the real light invariants are the id/coord guards in
each `Fetch` and the dedup in `Merge`. This is the **pure-helper-with-a-guarded-path** pattern already
ratified for `domains/seismic`, `domains/fire/*`, `term`, `httpx`, `render`. Padding the mappers with
no-op checks is what P10-05's intent forbids. **Recommend ratify** and restate M-P10 to ≤ 57.

## 6. Carried forward

- **P2** — the single-row marquee (scroll, stack/interrupt, R/O/Y bg theme-independent-except-monochrome,
  `[M]` visual, persistent muted empty state); goldens; `alloc-budget`.
- **P3** — new-alert persistence (seen-cache, cold-start-quiet, prune) + the tone + narration (R6).
- **P4** — REVIEW + VALIDATE + release `0.12.0` + DEBRIEF.
