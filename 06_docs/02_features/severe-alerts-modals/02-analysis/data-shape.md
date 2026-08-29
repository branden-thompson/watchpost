# Data shape — severe-alerts-modals (DISCOVER A-1 / A-2)

**Feature:** `severe-alerts-modals` · **Target:** 0.13.0 · **Status:** DISCOVER working document (red-team
remediated 2026-08-28)
**Probed:** 2026-08-28 15:38Z — **one probe, one afternoon**; the samples are committed under
`domains/globalfeed/testdata/` (their one home — they are the BUILD fixtures; USGS and NHC verbatim; the
national severe query verbatim — empty; the unfiltered NWS pull trimmed to one alert per product with the
full counts in its `_note`). Product-mix statements
below are from that single probe and are labelled as such.

DATA FIRST. This document is the `COV` denominator (brief M2) and the population/de-dup rule the modal
must implement. Every claim cites the parser line or the live sample key.

## 1. What each parser declares vs keeps (today)

### 1.1 USGS `significant_week.geojson` — `domains/globalfeed/usgs.go`

| Declared | Line | Reaches `Event`? |
|---|---|---|
| `id` | :48 | `ID` (:79) |
| `mag` | :50 | tier input only (`quakeSeverity`, :81) — **numeric magnitude dropped** |
| `place` | :51 | `Place` (:83) |
| `time` | :52 | `At` (:74, :87) |
| `type` | :53 | `Type` (:82) |
| `tsunami` | :54 | tier input only — **flag dropped** |
| `geometry.coordinates` | :57 | `Lat`/`Lon` (:84-85) — **depth `[2]` dropped** |

### 1.2 NHC `CurrentStorms.json` — `domains/globalfeed/nhc.go`

| Declared | Line | Reaches `Event`? |
|---|---|---|
| `id` | :33 | `ID` (:59) + basin → `Place` (:62) |
| `name` | :34 | **declared, never read** — the storm's name ("Dolly") is parsed and discarded (`Fetch` :57-68) |
| `classification` | :35 | `Type` + `Severity` via `tropicalClass` (:49) |
| `latitudeNumeric` / `longitudeNumeric` | :36-37 | `Lat`/`Lon` (:63-64) |
| `lastUpdate` | :38 | `At` (:53, :66) |

### 1.3 NWS `alerts/active` (curated national) — `domains/globalfeed/nws.go`

| Declared | Line | Reaches `Event`? |
|---|---|---|
| `id` (feature) | :72 | `ID` (:161) — **the URL form** `https://api.weather.gov/alerts/urn:oid:…` |
| `event` | :74 | `Type` + `Severity` (:163-164) |
| `areaDesc` | :75 | `Place` (:165) |
| `onset` / `sent` | :76-77 | `At` (:144-148; sent is the fallback) |
| `ends` / `expires` | :78-79 | `Until` (:154-157; expires is the fallback) |
| `references[].@id` | :81 | collapsed to `Superseded bool` (:131-138, :169) — **referenced ids dropped** |
| `geometry.coordinates` | :85 | first vertex → `Lat`/`Lon` (:159) — **polygon dropped** |

### 1.4 Tracked-location alerts — `domains/weather/nws/alerts.go` → `snapshot.Alert`

Kept (`platform/snapshot/types.go:169-186`, populated at `alerts.go:89-110`): id (**`properties.id`, the bare
`urn:oid:` form**, :90), event, severity, urgency, certainty, messageType, sent, effective, onset, expires,
ends, references (ids, :102-104), affectedZones (last path segment, :108-110), areaDesc, headline,
description, instruction.

Not kept: `@id`, `@type`, `geocode` (SAME/UGC), `status`, `category`, `sender`, `senderName`, `response`,
`note`, `scope`, `code`, `language`, `web`, `eventCode`, the entire `parameters` map, geometry
(`alertProps` :19-39 declares none; the zone query drops the feature envelope :59-63).

## 2. What the feeds actually carry (live, 2026-08-28)

**The curated national NWS query returned `"features": []`** at probe time (zero severe events nationally —
`domains/globalfeed/testdata/nws_active_severe.json`). Inventory for NWS therefore comes from an unfiltered live pull (265
alerts, `domains/globalfeed/testdata/nws_active_unfiltered_trimmed.json`) plus the weather-provider fixture
`domains/weather/nws/testdata/alerts.json`; the key set is the CAP/api.weather.gov schema and does not vary
by product. (`domains/globalfeed/testdata/` was created at PLAN exit for these samples; the package's
existing tests use inline `httptest` servers, `fetch_test.go:26`.)

### 2.1 USGS per-event `properties` (+ geometry)

| Key | Example | In parser? |
|---|---|---|
| `mag` | 5.8 | tier only |
| `place` | "55 km NW of Kodāri, Nepal" | yes |
| `time` / `updated` | 1787712730000 / 1787900310526 | time yes / **no** |
| `tz`, `url`, `detail` | null / event page / detail geojson | **no** |
| `felt`, `cdi`, `mmi` | 153 / 5.7 / 4.624 | **no** |
| `alert` (PAGER) | "green" · "yellow" · null | **no** |
| `status` | "reviewed" | **no** |
| `tsunami`, `sig` | 0 / 651 | tier only / **no** |
| `net`, `code`, `ids`, `sources`, `types` | "us", "7000tbwb", ",us7000tbwb," | **no** |
| `nst`, `dmin`, `rms`, `gap` | 0, null, 11.91, null | **no** |
| `magType` | "mww" | **no** |
| `type`, `title` | "landslide" / "M 5.2 Landslide - …" | type yes / title **no** |
| `geometry.coordinates[2]` (depth km) | 61 | **no** |

### 2.2 NHC per-storm object

| Key | Example | In parser? |
|---|---|---|
| `id`, `classification`, `latitudeNumeric`, `longitudeNumeric`, `lastUpdate` | "al042026", "PTC", 15, -46.9 | yes |
| `name` | "Dolly" | **parsed, discarded** |
| `binNumber` | "AT4" | **no** |
| `intensity` (kt, **string**) | "45" | **no** |
| `pressure` (mb, **string**) | "999" | **no** |
| `latitude` / `longitude` (text) | "15.0N" / "46.9W" | **no** |
| `movementDir` / `movementSpeed` | 280 / 25 | **no** |
| `publicAdvisory`, `forecastAdvisory`, `windSpeedProbabilities`, `forecastDiscussion`, `forecastGraphics` | `{advNum "005", issuance, fileUpdateTime, url}` | **no** |
| `forecastTrack`, `trackCone`, `initialWindExtent`, `forecastWindRadiiGIS`, `bestTrackGIS`, `earliestArrivalTimeTSWindsGIS`, `mostLikelyTimeTSWindsGIS`, `windSpeedProbabilitiesGIS` | zip/kmz URLs | **no** |
| `windWatchesWarnings`, `stormSurgeWatchWarningGIS`, `potentialStormSurgeFloodingGIS`, `peakSurgeKML` | null | **no** |

### 2.3 NWS per-alert `properties`

| Key | Example | Ticker path | Location path |
|---|---|---|---|
| `@id`, `@type` | wx:Alert | no | no |
| `id`, `areaDesc`, `references`, `onset`, `sent`, `ends`, `expires`, `event` | "Wicomico" | yes | yes |
| `effective`, `status`, `messageType`, `category`, `severity`, `certainty`, `urgency` | "Actual", "Alert", "Met", "Moderate", "Observed", "Expected" | no | severity/urgency/certainty/messageType/effective yes; status/category no |
| `geocode` (`SAME`, `UGC`) | `{"SAME":["024045"],"UGC":["MDZ022"]}` | no | no |
| `affectedZones` | zone URIs | no | last segment only |
| `sender`, `senderName` | "NWS Wakefield VA" | no | no |
| `headline` | "Special Weather Statement issued August 28 at 11:20AM EDT by …" | no | yes |
| `description`, `instruction` | "At 1120 AM EDT, Doppler radar…" (1–3 KB) | no | yes |
| `response`, `note`, `scope`, `code`, `language`, `web`, `eventCode` | "Execute", null, "Public", "IPAWSv1.0" | no | no |
| `parameters.*` — `maxWindGust`, `maxHailSize`, `eventMotionDescription`, `VTEC`, `NWSheadline`, `AWIPSidentifier`, `WMOidentifier`, `eventEndingTime`, `waterspoutDetection`, `expiredReferences`, `BLOCKCHANNEL`, `EAS-ORG` | `["60 mph"]`, `["/O.NEW.KPHI.MA.W.0160…/"]` | no | no |

Product mix in the unfiltered sample (265, **single probe 2026-08-28**): Small Craft Advisory 186, Extreme
Heat Warning 22, Air Quality Alert 13, Gale Warning 9, Flood Warning 7, Heat Advisory 7, Red Flag Warning 5,
Special Weather Statement 3, Flood Advisory 2, Special Marine Warning 2, Wind Advisory 2, High Surf
Advisory 2 — on *that* day the tracked-location path was dominated by advisories, and the "warning"
products present were **not** in the curated national list (`nws.go:28-35`). Seasonal variance (winter
products, hurricane season) is expected and not yet sampled (red-team B-F10, J8).

### 2.4 Richer-record URLs (listed; neither fetched nor rendered in v1 — out of scope)

USGS `detail` (full product set: ShakeMap, DYFI, moment tensor, nearby cities) and `url`; NHC
`publicAdvisory.url` / `forecastAdvisory.url` / `forecastDiscussion.url` / `windSpeedProbabilities.url`
(narrative advisories, forecast positions, wind radii) and the GIS zip/kmz cone/track; NWS `@id` and
`affectedZones[]` URIs.

## 3. Ruling D-14 (HUM LEAD, 2026-08-28): retain everything parsed

> "We should not throw away any data we're parsing — esp storm names — we can figure out where that data
> gets displayed (ticker for sure for names)."

Consequences (as remediated at red-team — E-2 in objectives §11 asks HUM LEAD to confirm the reading):
- **Retention rule:** every property the parser decodes is kept on the event (per-class detail); nothing
  declared is dropped again. In v1 the **declared set = the render list + the storm name + URLs kept but not
  rendered** (§4). The GIS/kmz product tail, seismic-network telemetry (`net/code/ids/sources/types/nst/
  dmin/rms/gap`) and the open-ended `parameters` map are **not decoded** in v1 — nothing parsed is thrown
  away, and nothing no requirement asked for is parsed (red-team A-F3, S6, S7).
- **Storm names reach the ticker** (tape + narration): "Tropical Storm Dolly" not "Tropical Storm" — a
  0.12.0 behaviour amendment carried as FR-12, via a `Name` field and a name-aware `Sentence()` (folding the
  name into `Type` breaks `Article()`, `event.go:86-92` — red-team C-3).
- **Long text (NWS `description` / `instruction`, 1–3 KB):** retained, not clamped to `maxFieldRunes`. The
  cost is not residency (~2 MB for 300 alerts) but **re-parse churn** — `cycle()` decodes every source every
  2 min (`app/ticker.go:53`, `:145-153`) even when the httpx body is unchanged. Mitigation for PLAN: a parse
  memo per source keyed on **httpx's own not-modified / served-from-cache fact** (conditional GETs already
  run — `platform/httpx/httpx.go:511-512`, `cache.go:222-232`; surface it from `GetJSON`), instantiating
  the existing generic `Memo[T]` (`domains/fire/memo.go:13-30`), sha256 only as fallback (red-team P7, S8).
  Rune-bounding stays for *hostile* input (P4 F5): a larger cap for prose (~4 000 runes) rather than 120,
  and `render.Plain` at **every** rendered field path (NFR-6).

## 4. Proposed detail field set (FR-5 / COV denominator)

Per-class detail structs referenced from `Event` (SAM-D-21; a single wide struct is the counter-proposal —
§7, and objectives E-1 re-opens it for PLAN). Types and bounds are PLAN's to finalise. **Render v1**
marks the frozen COV render list; **Keep** = decoded and retained but not rendered; **v2** = not decoded in
v1 (red-team A-F3/S6/S7 remediation, pending E-2).

> **Amended at BUILD exit (red-team R3-C-05):** the Render column now states the ratified record shape
> (plan §5.6–5.8, `domains/severe/record.go`), and `cov_test.go` iterates exactly this list. Moved to
> Keep: quake `Title` (the record's title is the product + place), CAP `Headline` and `MessageType` (the
> record's title is the product and its timing line carries what the headline restates), `NWSheadline`
> and `VTEC`. Moved to Render: tropical `BinNumber`, `LatText`/`LonText` (the Position line), `Basin`,
> `ForecastNum`/`DiscussionNum` (the Advisories line).

| Class | Field | Go type | Source key | Bound | Render v1 / Keep / v2 |
|---|---|---|---|---|---|
| Quake | Mag | **`*float64`** (parser is a pointer, `usgs.go:50` — absent ≠ 0.0, red-team C-9) | `mag` | −1..12, reject NaN | Render |
| Quake | MagType | `string` | `magType` | clamp 120 | Render |
| Quake | DepthKm | `float64` | `geometry.coordinates[2]` | 0..1000 | Render |
| Quake | Title | `string` | `title` | clamp 120 | Keep (amended R3-C-05) |
| Quake | Alert (PAGER) | `string` | `alert` | enum green/yellow/orange/red/"" | Render |
| Quake | Sig, Felt | `int` | `sig`, `felt` | ≥ 0 | Render |
| Quake | CDI, MMI | `*float64` | `cdi`, `mmi` | 0..12 | Render |
| Quake | Status | `string` | `status` | enum | Render |
| Quake | Tsunami | `bool` | `tsunami` | — | Render |
| Quake | UpdatedAt | `time.Time` | `updated` (ms) | epoch sanity (P4 F6) | Render |
| Quake | Net, Code, Ids, Sources, Types, Nst, Dmin, Rms, Gap, TZ, URL, Detail | typed | same | clamp / bound each (D-14: keep) | **v2** (telemetry; `URL`/`Detail` Keep, not rendered) |
| Tropical | **Name** | `string` | `name` | clamp 120 | Render (+ tape + narration) |
| Tropical | WindKt, PressureMb | `int` | `intensity`, `pressure` (**strings**) | Atoi; 0..250 / 800..1100 | Render |
| Tropical | MoveDirDeg, MoveSpeedKt | `int` | `movementDir`, `movementSpeed` | 0..360 / 0..100 | Render |
| Tropical | BinNumber, LatText, LonText, Basin | `string` | same | clamp | Render (title suffix; the Position line — amended R3-C-05) |
| Tropical | AdvisoryNum, AdvisoryAt, ForecastNum, DiscussionNum, AdvisoryURL | `string`/`time.Time` | `publicAdvisory.*` etc. | clamp; URLs kept, never fetched | AdvisoryNum/At/ForecastNum/DiscussionNum Render; URLs Keep |
| Tropical | GIS product URLs | `[]string` or struct | `forecastTrack.*` … | cap slice len | **v2** |
| SevereWx | Headline | `string` | `headline` | clamp 120 | Keep (amended R3-C-05) |
| SevereWx | Description, Instruction | `string` | same | **prose cap ~4 000 runes** (D-14) | Render |
| SevereWx | CAPSeverity, Certainty, Urgency, MessageType, Category, Response, Status | `string` | same | clamp (PAGER is the one validated enum; the others are bounded text — R5-A-09) | Render (Severity/Certainty/Urgency); Keep (rest, MessageType amended R3-C-05) |
| SevereWx | SenderName, Sender | `string` | same | clamp | Render (SenderName); Keep |
| SevereWx | UGC, SAME | `[]string` | `geocode.*` | cap len ~50 | **v2** |
| SevereWx | AffectedZones, References | `[]string` | same | cap len | Keep (References drive NFR-12) |
| SevereWx | Effective, Sent, Expires, Ends, Onset | `time.Time` | same | parsed already | Render |
| SevereWx | Parameters (MaxWindGust, MaxHailSize, EventMotion, NWSHeadline, VTEC, …) | `map[string][]string` or named fields | `parameters` | cap map size / clamp values | **Allowlist only** — MaxWindGust, MaxHailSize, EventMotionDescription Render; NWSheadline, VTEC Keep (amended R3-C-05); all other keys **not decoded** (S7) |
| SevereWx | Polygon | `[][2]float64` | `geometry.coordinates` | cap vertices | **v2** |

## 5. Population and de-dup (A-2)

### 5.1 The data path and where the full set lives

Fetch loop `app/ticker.go:145-153` — all three sources into one `events` slice; **the pre-cap, pre-radius
full set exists at :153**; `Active` (:155) then drops expired alerts (a filter, not a no-op). Then: radius
`Within` (:161-167 → `stack.go:19-33`, centred on `watch[0]`, `events = nil` with no location set);
`Locate` (:168-169) — **after** the radius branch, so the pre-radius set has **no `Location`** yet; the
modal's publish path must run `Locate` on the pre-radius set before filtering (red-team C-2);
superseded stripped into `display` (:173-179); `Merge` cap 30 (:180 → `stack.go:73-99`, `MaxEvents`
:54); publish post-cap (`:181`). **Retention: none** — only ids survive in the seen-store
(`:187`, `:191`; store `:423-480`, ids-only proof `:461-465`, `:477`). The modal needs a **new publish path**, not a lookup.

### 5.2 Id normalisation (confirmed live)

`feature.id` = `https://api.weather.gov/alerts/urn:oid:2.49.0.1.840.0.8b42…001.1` (ticker path,
`nws.go:161`); `properties.id` = `urn:oid:2.49.0.1.840.0.8b42…001.1` (location path, `alerts.go:90`).
Same alert, two strings. De-dup key = the `urn:oid:` suffix **validated against the OID grammar**
(`urn:oid:` + dotted digits); an id that fails validation keeps its raw string and never merges — a bare
suffix match would let `https://evil/…/urn:oid:<real-id>` collide and, under "prefer the location record",
overwrite a national warning (red-team S4). On a collision the trusted-source record wins and fields merge;
nothing overwrites.

### 5.3 Overlap cases and what "properly de-duped" means

| Case | Rule |
|---|---|
| Same alert in both sources (Tornado Warning at a tracked location) | one row keyed on the normalised id; prefer the **location record** (has headline/description/instruction/ends); merge the feed's lat/lon and tied label |
| National-only | keep; label from `globalfeed.Locate` |
| Location-only warning products (Winter Storm, Flood, High Wind, Extreme Heat, Red Flag, Gale…) | keep in Warnings — the "larger list for free" |
| Watch products beyond the 3 national ones | union; classify by `Watch` in the event name |
| Advisories / Special Weather Statements | location-only by ruling (D-10); no cross-source key needed |
| Multi-location alert (`mapAlert` appends the same alert to each matched location, `alerts.go:111-118`) | one row, all matched location labels, counted once |
| Superseded/UPDATE pair on the location path (`References` carried but **never consumed** today) | apply a **guarded** superseded rule to both paths: a reference suppresses a live alert only from the same sender + same product + newer `sent` — the national rule at `nws.go:131-138` is unguarded today and would let a crafted low-grade alert suppress a live warning (red-team S3; NFR-12) |
| Radius-filtered ticker | the modal consumes the **pre-radius** set or it under-reports |

### 5.4 Tab population (recommendation, for HUM LEAD)

| Tab | Source(s) | Include rule |
|---|---|---|
| Warnings | national curated ∪ all tracked locations' alerts | event contains "Warning" |
| Watches | same union | event contains "Watch" |
| Advisories | tracked locations only | event contains "Advisory" |
| Spec. Statements | tracked locations only | event == "Special Weather Statement" (exact) |
| Sig. Quakes | USGS only | `Class == ClassQuake` |
| Tropical | NHC storms (∪ location alerts naming Hurricane/Tropical Storm → Warnings, not here) | `Class == ClassTropical` |

Unclassified location products (Air Quality Alert, "… Statement" other than SWS) are **not shown** and not
counted — the honest count `nnn` = rows the tab renders (de-duped, superseded removed, multi-location
collapsed), never a sum of per-location slices and never the post-30 stack. Retained set hard-bounded
(A-6 proposes 500, most-recent-wins) with "showing N of M" when exceeded.

### 5.5 Composition layer

`TickerMsg{Items []TickerItem}` (`dashboard.go:34-36`; `TickerItem` `ticker.go:57-62`) is lossy —
pre-rendered strings. `d.snap` (`dashboard.go:244`) holds the per-location alerts at the same time. The modal
can be composed in the TTY layer from (a) **a new field on `TickerMsg`** carrying the pre-cap, `Locate`d
events with their detail, plus (b) `d.snap` — preserving the "the app maps globalfeed events onto TickerItems"
seam documented at `modes/tty/dashboard.go:33-35`. A-3's recommendation to reuse `d.ticker` as the data
source is **not** sufficient (lossy, post-cap); a new message *type* is not needed either (red-team C-17).

### 5.6 Distance readiness (deferred sort, OQ-4)

Default location = `cfg.Locations[0]` (`app/setup.go:57-64`, `config.go:139`); the radius is measured from
`watch[0]` (`app/ticker.go:161-163`). `Locate` returns the *first* watchlist entry within 150 km
(`locate.go:32-36`), not the nearest. `HasPoint=false` is zone-only NWS (`nws.go:159`); location-sourced
alerts carry no coordinates at all — a future distance sort falls back to the parent location's lat/lon
(`types.go:35-36`) and sinks point-less rows last.

## 6. Cadence and cost (A-6, for the standing perf lens)

One loop, `tickerEvery = 2 min` (`app/ticker.go:53` — the *app* ticker, not `modes/tty/ticker.go`); per-source freshness by httpx TTL — USGS 5 m
(`usgs.go:27`), NHC 30 m (`nhc.go:12`), NWS 2 m (`nws.go:15`). **The modal adds no fetch**; its only new
fetch-layer cost is retaining the pre-cap set past `Merge`. Seen-store (`seen.json`, id + timestamp only,
`:461`, `:476`) is **unchanged** by widening `Event`. Residency: ~150–270 KB for 300 alerts with clamped
text, ~0.6–1.8 MB with full descriptions; churn without a parse memo ≈ 650 MB/day at 3 KB × 300 × 720
cycles — the number the parse memo (§3) removes.

## 7. Counter-arguments recorded

- **Single wide `Event` vs per-class detail structs:** Go has no sum types; per-class structs force a
  type-switch and three nil-check branches in every consumer; a flat struct with "populated when Class==X"
  fields stays rangeable and diffable and wins if the detail view renders a generic key/value list. PLAN
  decides; the inventory is the same either way.
- **Pre-cap union vs the ticker's `MaxEvents` invariant:** the 30-cap is a P10-03 bound and the seen-marking
  contract assumes "every active event is marked". A second, larger event universe must carry its own hard
  cap and an honest "showing N of M" — never an unbounded list.
