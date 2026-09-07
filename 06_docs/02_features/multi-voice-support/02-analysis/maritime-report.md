# Maritime report — DISCOVER analysis (brief A-6 · R-6 · OQ-12)

Read-only investigation as of `main @ 186d97c` (2026-08-29). The report is a **full** one by ruling
(MVS-D-4: buoy observations, tides, currents); this document is the content inventory, the shape to copy and
the answers DISCOVER owes PLAN.

## 1. The data a location actually has

`Location.Marine *Marine` (`platform/snapshot/types.go:46`) — **nil inland**; SI internally, converted at render
(`types.go:1-8`). Fields (`types.go:58-113`): `SwellHeight` m · `SwellDirDeg` ° true · `WaveHeight` m
(significant) · `WavePeriod` s · `WindWaveHeight` · `SecondarySwellHeight/DirDeg/Period` · `WindSpeed`/`WindGust`
**m/s at the buoy** · `WaterTemp` **°C** (NDBC only) · `Buoy` id · `BuoyDistanceKM` · `ObservedAt` UTC ·
`TideLevel` m above MLLW (observed) · `Tides []TideEvent{Time, Height m, Type "H"|"L"}` (48 h, `interval=hilo`,
`datum=MLLW`) · `TideStation` **name** · `TideStationKM` · `Currents []CurrentEvent{Time, Speed m/s, Type
"flood"|"ebb"|"slack"}` (`interval=MAX_SLACK`) · `CurrentStation`. **No wind direction at the buoy.**

Providers (`app/dashboard.go:74`): `nws-marine` (the raw **gridpoint**'s eight wave/swell series, first non-null
value — `domains/weather/nws/marine.go:87,113-121`; coastal iff a positive swell or wave height, `:141-148`,
inland memoised 24 h), `ndbc` (buoys ≤ 150 km, `domains/marine/ndbc/ndbc.go:32`; obs TTL 10 min), `coops` (tides ≤ 60 km, currents
≤ 40 km, `domains/marine/coops/coops.go:32-34`; predictions cached to UTC midnight), `coops-obs` (water level, 10 min). Cadence:
`KindMarineObs` 10 min, `KindMarine` 30 min priority / 1 h recent (`app/pipelines.go:129-145`). Merge:
`harmonizeMarine` field-wise in provider order, `fillMarine` never replaces a set field, `fillTides` moves the
whole block with its station (`harmonize.go:189-229`).

**No coastal-waters forecast prose is fetched today** — `synth.Products` pulls HWO/SPS/NOW/ZFP only. *Superseded
by §10 (MVS-D-14): the CWF product is served by the same endpoint and is in scope.*

**The coastal predicate is exactly `loc.Marine != nil`** (`modes/tty/detail.go:53`); nothing else in the app
knows "coastal"; the watchlist table and `modes/report` render no marine data. And inside `Compose`,
`snap.Locations[0].Marine` is **always nil** — the deck's throw-away assembler fetches only `KindObs` and
`KindAlerts` (`app/radio.go:294-299`); fire and seismic arrive by hook, and maritime must too.

## 2. What the screen says today (the content inventory)

`modes/tty/detail_marine.go` (`maritimeRows :72-94`, `swellRows :111-132`, `tideRows :134-158`):
Observed (`fixedAgeTrim`, guarded `!ObservedAt.IsZero() && Buoy != ""` `:74`) · Conditions (`seaState()`
`:269-281`, Douglas bands: calm (glassy) < 0.1 m · smooth < 0.5 · slight chop < 1.25 · moderate chop < 2.5 ·
rough < 4 · very rough) · Water Temp `74ºF (buoy 46224, 6 mi)` · Swell `W 2.0 ft (period 14 s)` (16-pt
`compass()` `:258`) · Swell 2 · Wind Waves · Buoy Wind `11 mph (gusts 16 mph)` — **mph, not knots** · Tide
`Rising 3.7 ft (La Jolla, 24 mi)` (`tideTrend` `:162`; `stationNote` cuts the name at " (") · Next High
`19:40 5.7 ft` · Next Low `02:49 -0.1 ft` · Currents `Flood 1.4 kt (Slack 16:05)` (`currentRow` `:184-192`: last
extreme at/before now; note = next event). Pinned examples `detail_marine_test.go:26-30,62-68,77-82`.

## 3. The shape to copy

`FireReport{Known, State, RadiusKm, IncidentRadiusKm, Sources, Lat, Lon}` + `FireSegments` (`synth/fire.go:20-38`);
`SeismicReport{Known, State, Lat, Lon}` + `SeismicSegments` (`synth/seismic.go:14-32`). Five steps: guard → nil
(`fire.go:39-41`, `seismic.go:33-35`); `ExpandStates(location)` + head with a 2 s pause (`firePause`/`seismicPause`);
keys `"<report>:notice:" + contentKey(text)` (sha256/8, `fire.go:78-81`, REVIEW C1); render script parts,
skipping `""` (`script.go:100-109`); append `Segment{Key: "<report>:" + contentKey(piece), Text: piece}`.

Script parts: `fire-report/{head(.Location,.Sources), count(.Count,.Ring), strongest, incident, outside}`;
`seismic-report/{head, count, quake, felt, more, link}`; conventions `script.go:1-21` (tree = wiring; missing
part → `global/<part>` → silence; `missingkey=error` `:165`; first line a `{{/* */}}` doc comment; overrides
under `<config>/scripts/`, ≤ 64 KB).

Hooks: `app/radio.go:43-44` (`fire`, `seismic func(snapshot.LocationRef) …`), assigned `app/dashboard.go:256`
from `lp.fireFor/seismicFor` (`:86`, `:366-399`) via `attachRadio` (`:251`); called `app/radio.go:314-323`;
narrow reads `Assembler.FireFor`/`SeismicFor` (`platform/snapshot/assembler.go:89,107`) returning
`(state, lat, lon, ok)` — REVIEW C2, no snapshot clone per cycle; pure mappers `app/fire.go:38`, `app/seismic.go:16`.
**There is no `Assembler.MarineFor`** — a new narrow read that runs the same `fillMarine` merge.

Absence today is two-fold: the **whole report is skipped** (`FireSegments` nil when `!Known`; seismic nil when
empty; `Compose` splices only non-empty, `compose.go:69,73`) and a **zero-count sentence** inside a known
report (`fire-report/count.txt` "no hotspots…", `fire_test.go:53-56`). For R-6: inland = skip; coastal with an
empty sub-block = an in-report absence line.

## 4. Units and pronunciation

In the composer: `imperial` threads everywhere (`app/radio.go:323`); `degrees()` `compose.go:171`; wind
`compose.go:158-163` ("kilometres" — British; `distanceWords` says "kilometers" — an existing inconsistency);
`compass()` 8-point words `compose.go:178-184`; `bearingWords` 16-point `fire.go:168-172`; `bearingLong`
`seismic.go:112-119`; `distanceWords` `fire.go:143-156`; `durationWords` `fire.go:175-199`; `joinAnd` `fire.go:214`.
**Nothing speaks knots, feet, a negative height or a clock time.** `durationUnits()` is `app/severe_read.go:192`.

Pronunciation tables already carry `KT/kts → knots`, `FT/ft → feet`, `NM → nautical miles`, `MPH`, and every
16-point compass abbreviation (`pronounce/rules/abbreviations.txt`); `Pronounce` runs at render
(`normalize.go:275-310`, `source.go:330`). Clock times "3:42 PM" are already safe (`normalize.go:27,188`,
`normalize_test.go:54`). **Gap:** digit spelling exists only for callsigns and web addresses
(`normalize.go:360-373`) — `"46224"` reads as a number, `"SDBC1"` hits the all-caps path.

Needed new spoken forms: knots (screen convention: knots under **both** unit systems, `render/units.go:146-148`,
UAT 61) with singular "1 knot"; heights in feet/metres incl. **"minus 0.1 feet"**; clock time via
`t.In(tz).Format("3:04 PM")` (`ref.TZ` is loaded at `app/radio.go:310-313`); flow words "flooding / ebbing /
slack water"; sea state — **`seaState()` lives in a renderer the composer must not import** (import-direction
lint, `types.go:1-8`): duplicate in synth or lift to `platform/render` (which synth imports); buoy wind — screen
says mph, marine convention says knots (a PLAN decision).

## 5. Tests that grow a maritime sibling

`synth/{fire,seismic,composer}_test.go` patterns; `synth_test.go:575 TestReportsAreSeparatedByAir` (the
composition-order/pause test — extend for a third report); **every `Compose` call site changes with the
signature** (`synth_test.go:568,583,601`, `seismic_test.go:50`, `soak_test.go:40`); `soak_test.go:21` the
env-gated 1-hour R6 soak pattern; `script/script_test.go:39,54-56,114` pin the report list
(`…,fire-report,global,maritime-report,seismic-report,…`) and every part the app asks for; `app/testdata/declset.txt`
(and snapshot's) re-captured for `marineFor`/`MarineFor`. No golden pins the broadcast order beyond these.

## 6. Order in the broadcast (OQ-12) — recommendation

Between the products and the fire report (`compose.go:66→69`): lead → conditions → alerts → products →
**maritime** → fire → seismic → tail. It mirrors Location Details (CURRENTLY · TODAY · FORECAST · **MARITIME** ·
FIRE · SEISMIC, `modes/tty/detail.go:47-60`), keeps "spoken consistent with the screen" (`seismic.go:81-87`),
reads as conditions continuing the weather rather than a hazard report, and needs only one new pause assertion
(products → maritime); the `compose.go:67-68` comment's "forecast → fire → …" is literally a placeholder for more
reports, and `pauseLast` only ever raises the last segment's pause, so any insertion is safe.

## 7. Proposed spoken content — SUPERSEDED for wording by `04-development/p3-maritime-tones.md` Tasks 3.2–3.3 (MVS-D-21: "above the low-water mark", 12-hour clock, wind in the listener's unit); the data mapping below still holds

| Part | Sentence (imperial shown) | Data |
|---|---|---|
| `head` (2 s) | "This is the Watchpost Maritime report for Oceanside, California. Data from the National Data Buoy Center and NOAA Tides and Currents; not for life-safety use." | `.Location`, `.Sources` = the providers that answered |
| `observed` | "Conditions at the nearest buoy, 6 miles offshore, observed 39 minutes ago." | `BuoyDistanceKM`, `ObservedAt` (guard exactly as the screen: `!IsZero && Buoy != ""`) — **drop the id** rather than spell it |
| `sea` | "Seas are slight chop, with a significant wave height of 3 feet." | `seaState(WaveHeight)`, `WaveHeight` |
| `swell` | "Primary swell from the west at 2 feet, with a dominant period of 14 seconds." (+ secondary) | `SwellDirDeg`→`compass()`, `SwellHeight ?? WaveHeight`, `WavePeriod`, `Secondary*` |
| `water` | "Water temperature 74 degrees." | `WaterTemp` → `degrees()` |
| `wind` | "Wind at the buoy 11 knots, gusting to 16." | `WindSpeed`, `WindGust` (no direction field — omit) |
| `tide` | "The tide is rising, currently 3.7 feet above mean lower low water, at La Jolla, 24 miles from you." | `tideTrend`, `TideLevel`, `TideStation` (name cut at " ("), `TideStationKM` |
| `tide-next` | "The next high tide is at 7:40 PM at 5.7 feet; the next low at 2:49 AM at minus 0.1 feet." | next H / next L after now, `3:04 PM` in the location's zone |
| `current` | "Tidal currents are flooding at 1.4 knots, with slack water at 4:05 PM." | last event ≤ now, next event, `Speed` → knots |
| `absence` | "There are no tide or current predictions available for your area." | `len(Tides)==0 && len(Currents)==0` (station out of range); inland = whole report skipped |
| `link` (optional) | "For coastal conditions in your area, visit https://www.ndbc.noaa.gov" | mirrors `seismic-report/link.txt` (no trailing slash) |

Length: ~10 sentences ≈ 45–60 s of Piper audio — about fire + seismic combined; a coastal location that also
carries fire and seismic could pass four minutes before the tail. In-pattern mitigations: speak only the next
high and low (as the screen does), fold sea state + swell into one sentence, a spoken cap like
`spokenQuakeCap` (`seismic.go:25`).

## 8. Files a PLAN would create / touch

New: `domains/radio/synth/marine.go` (`MarineReport{Known, State snapshot.Marine, TZ, Lat, Lon}`,
`MarineSegments`, `knotWords`/`heightWords`/`clockWords`/`seaStateWords`/`flowWords`) · `scripts/maritime-report/{head,
observed, sea, swell, water, wind, tide, tide-next, current, absence, link}.txt` · `synth/marine_test.go` ·
`app/marine.go` (`marineReportOf`). Modified: `synth/compose.go:48,66-69` (via the `Reports{}` struct from A-2.6,
**not** a tenth positional parameter — landing it positionally guarantees a conflict with the role work) ·
`app/radio.go:43-44,314-323` · `app/dashboard.go:86,249,256` + `marineFor` · `platform/snapshot/assembler.go`
(`MarineFor`) · the pinned lists in `script_test.go` · the `Compose` call sites and `TestReportsAreSeparatedByAir` ·
the declset pins · `docs/where-things-happen.md:15` · CHANGELOG.

## 9. Risks (into the risk register)

1. **Mostly predictions, and a contaminated age** — tides/currents are astronomical predictions cached to
   UTC midnight; `nws-marine` stamps `ObservedAt = time.Now()` (`marine.go:137`), so without the screen's exact
   guard the voice says "observed 0 minutes ago" for a forecast-only block. Tide clock times change the
   content key every cycle → cache churn on a repeat loop (compounds A-2's `maxCached` risk).
2. **Partial blocks are the common case** — tides only (no buoy in 150 km), buoy only (no station in 60 km —
   `detail_marine_test.go:85`), swell forecast only; currents rarest (40 km, type-W stations dropped
   `coops.go:230-238`). Every phrase guards itself; the absence line distinguishes "no data" from "inland".
3. **Units collide** — SI internally; screen ft/m for heights, kt for currents under both systems, mph/km/h for
   buoy wind; marine convention knots and feet. Negative heights need "minus". Wording: "three point seven
   feet" vs "about four feet".
4. **Length** (§7).
5. **Import direction** — `seaState/tideTrend/nextTide/currentRow` live in `modes/tty`; duplicating risks the
   screen and the voice drifting on the same data (the failure `seismic.go:81-87` guards); lifting to
   `platform/render` touches the render declset pin.
6. **Buoy id pronunciation** — drop the id ("the nearest buoy, 6 miles offshore").
7. **Signature churn** — `Compose` is being reworked by A-2 at the same time; the `Reports{}` struct is the
   seam both need, batch 1.
8. ~~No coastal-waters forecast~~ — superseded by §10 (MVS-D-14: in scope, free through the products endpoint).

## 10. Addendum (MVS-D-14, 2026-08-29) — the Coastal Waters Forecast is free

Probed live against the NWS API for the San Diego office (`SGX`):

- `GET /products/types/CWF/locations/SGX` — the **same endpoint shape `synth.Products` already uses** for
  HWO/SPS/NOW/ZFP (`domains/radio/synth/products.go:74`) — returned **14 issued CWF products**, the newest at
  21:13 UTC that day. Adding `"CWF"` to a marine product list is one more fetch per office at the existing
  10-minute TTL: **free in the sense of the ruling.**
- The product (`FZUS56 KSGX … CWFSGX`, 3 857 chars, 94 lines) is sectioned by marine zone with `$$`
  separators and UGC headers exactly like a ZFP: a **synopsis** block (`PZZ700-…`, "Synopsis for the far
  Southern CA coast…") followed by one block per coastal-waters zone (`PZZ740-…` "Coastal Waters from San
  Mateo Point to the Mexican Border and out to 10 nm", then the 10–60 nm zones) with `.TONIGHT… Wind W 10 kt
  … Seas 4 to 5 ft. Wave Detail: W 4 ft at 7 seconds…` periods.
- **`synth.FilterUGC` (`domains/radio/synth/ugc.go:20`) already cuts a product to named UGC blocks** and keeps the
  preamble; passing the location's marine zone (e.g. `PZZ740`) plus the synopsis zone (`PZZ700`) yields a spoken
  CWF of synopsis + the nearshore forecast — the same mechanism the ZFP uses at `app/radio.go:307-309` (UAT 81).
- **Resolving the marine zone is the one new piece.** The point response (`/points/{lat},{lon}`) carries
  `forecastZone`, `county`, `fireWeatherZone`, `cwa` — **no marine zone**; `/zones?type=marine|coastal&point=…`
  and `…&area=CA` returned nothing; but `/zones/coastal/PZZ740` returns the zone with its **Polygon geometry**.
  So: read the CWF's own UGC headers (the zones the office issues for), fetch each zone's geometry once (24 h
  TTL, a handful per office), and pick the nearshore ("out to 10 nm") zone whose polygon is nearest the
  location — a PLAN design item (A-6 → "marine zone resolver"), bounded and cacheable.
- **Length:** the synopsis + one nearshore zone block is ~15–20 lines ≈ 60–90 s spoken. With the observations,
  tides and currents the maritime report approaches 2–2.5 minutes; the spoken caps in §7 apply, and the CWF
  block should read only the first two or three periods (`.TONIGHT`, `.SUNDAY`, `.SUNDAY NIGHT`) — a
  `spokenPeriodsCap` in the style of `spokenQuakeCap` (`seismic.go:25`).
- The "UGC-filtered" phrase in §6 refers to this `FilterUGC` pass at `app/radio.go:309`, not to anything in
  `Compose` itself.
