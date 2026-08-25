# AI-1: NWS API (api.weather.gov) Research

Research date: 2026-08-23. Live header values below were observed with `curl` against the production API on that date.

## 1. Endpoint map (lat/long → data)

| Step | URL template | Key JSON paths |
|---|---|---|
| 0. Resolve point (cache ~1 day) | `GET /points/{lat},{lon}` | `properties.gridId`, `gridX`, `gridY`, `forecast`, `forecastHourly`, `forecastGridData`, `observationStations`, `forecastZone` (e.g. `.../zones/forecast/DCZ001`), `county` (`DCC001`), `fireWeatherZone`, `radarStation` |
| 1a. Station list | `GET /gridpoints/{wfo}/{x},{y}/stations?limit=3` | `features[].properties.stationIdentifier` (ordered nearest-first; e.g. KDCA, KCGS, KADW for DC) |
| 1b. Current obs | `GET /stations/{id}/observations/latest` | `properties.timestamp`, `textDescription`, `temperature{value,unitCode,qualityControl}`, `dewpoint`, `windSpeed`, `windGust`, `windDirection`, `barometricPressure`, `relativeHumidity`, `visibility`, `heatIndex`, `windChill`, `precipitationLastHour` |
| 2a. Daily (12-h periods, 7 days) | `GET /gridpoints/{wfo}/{x},{y}/forecast` | `properties.periods[]{name,startTime,endTime,isDaytime,temperature,temperatureUnit,probabilityOfPrecipitation.value,windSpeed,shortForecast,icon}` |
| 2b. Hourly (7 days) | `GET /gridpoints/{wfo}/{x},{y}/forecast/hourly` | same `periods[]` shape, 1-h steps |
| 2c. Raw grid (optional) | `GET /gridpoints/{wfo}/{x},{y}` | `properties.<element>.values[]{validTime (ISO8601 interval), value}`, `uom` |
| 3. Active alerts | `GET /alerts/active?point={lat},{lon}` or `?zone=Z1,Z2,...` | `features[].properties.{id,event,headline,severity,urgency,certainty,sent,effective,onset,expires,ends,messageType,status,references[],affectedZones[],areaDesc,description,instruction}` |

Source: points response fields and FAQ workflow at https://www.weather.gov/documentation/services-web-api ; path list from https://api.weather.gov/openapi.json.

## 2. Alerts

**Filters on `/alerts/active`** (openapi.json): `status[]` (actual/exercise/system/test/draft), `message_type[]` (alert/update/cancel), `event[]`, `code[]`, `area[]` (state/marine area), `point`, `region[]`, `region_type` (land/marine), `zone[]` (forecast or county UGC), `urgency[]`, `severity[]`, `certainty[]`. `point`, `zone`, `area`, `region` are **mutually exclusive**. Convenience paths: `/alerts/active/zone/{zoneId}`, `/alerts/active/area/{area}`, `/alerts/active/count`. `/alerts` (non-active) adds `start`, `end`, `limit`, `cursor` pagination.

**Time fields** (schema `Alert`): `sent` = "time of the origination of the alert message" (use this for M2 latency measurement); `effective`; `onset` (nullable, event start); `expires` = message expiry; `ends` (nullable, event end). Treat alert as active while `now < (ends ?? expires)`.

**Dedupe**: `id` is unique per message; `messageType` = update/cancel carries `references[]{@id, identifier, sender, sent}` pointing to prior messages — supersede those ids. `status=actual` filter should be applied to exclude tests.

**Push/streaming**: none. The API offers CAP (`application/cap+xml`) and ATOM (`application/atom+xml`) representations of the same endpoints, but they are pull feeds. No WebSocket/SSE. Polling only.

**Poll tolerance**: the alerts endpoint returns `cache-control: public, max-age=5, s-maxage=5` (observed), i.e. NWS's own CDN refreshes at most every 5 s. Docs state rate limits are non-public and blocked requests "may be retried after the limit clears (typically within 5 seconds)". A 15–30 s alerts poll is comfortably inside M2 ≤60 s once NWS-side propagation (typically well under a minute from issuance to API) is added.

## 3. Rate limits & ToS

| Item | Finding |
|---|---|
| User-Agent | Required; format `(app-name, contact@email)`. Observed: request with empty UA → **403**. |
| Rate limit | "Not public information"; exceed → error, retry after ~5 s. Proxies/shared IPs hit it sooner. |
| Caching headers | All responses carry `cache-control` + `expires` + weak `etag`. Observed: `/points` max-age ≈2124 s; forecast/hourly max-age ≈1400–2100 s (s-maxage 3600); obs latest max-age 52 s (s-maxage 300); alerts max-age 5 s. |
| Conditional GET | `If-None-Match` with the returned ETag yielded **200**, not 304 — do not rely on it; use `max-age`/`expires` for client caching. |
| Coverage | US states and territories (incl. PR, GU, AS, VI marine/land zones) only. Non-US points return 404 from `/points`. |
| Auth/cost | None; free, no key. |

## 4. Scaling to 25+ locations

Requests per location per cycle if naive: 1 obs + 1 hourly + 1 daily + 1 alerts = 4 (plus a one-time `/points` + `/stations`). For 25 locations at a 60 s cycle that is 100 req/min — likely fine but wasteful.

Batching: alerts accept a comma list of zones: `/alerts/active?zone=DCZ001,MDZ014,VAZ054` (observed 200). Group all locations' `forecastZone` + `county` UGCs (from `/points`) into one call (keep URL < ~2 KB, so ~50–80 zones per request), or one `?area=XX` call per state (TX returned 133 KB — larger payload but one request per state). Forecasts are per-gridpoint and not batchable, but locations in the same 2.5 km cell share a URL — key the cache by `{wfo}/{x},{y}`. Observations are per station; multiple locations may share a station.

| Data | Honor cache for |
|---|---|
| `/points`, station list | 24 h (docs: grid/office "may occasionally change", so re-resolve daily) |
| forecast, hourly | per `expires` (~30–60 min) |
| observations/latest | 60 s minimum (max-age 52 s; stations report hourly anyway) |
| alerts | 15–30 s poll, batched by zone |

## 5. Known pitfalls

- **500/503**: Well-known intermittent errors on forecast/gridpoint endpoints (documented widely in NWS GitHub issues). Implement retry with backoff and serve last-good data.
- **Observation staleness**: ASOS reports hourly (~:52); `latest` may be 1–2 h old or have `value: null` for fields (QC flag in `qualityControl`; prefer `V`/`C`). Fall back to the next station in the `/stations` list when `temperature.value` is null or `timestamp` > 2 h old. Nearest ≠ freshest.
- **Null grid values**: `forecastGridData` elements can be null/missing; `probabilityOfPrecipitation.value` is nullable in period forecasts.
- **Units**: obs use `wmoUnit:degC`, `wmoUnit:km_h-1`, `wmoUnit:Pa`, `wmoUnit:percent`, `wmoUnit:m`; period forecasts use `temperatureUnit: "F"` and text wind speeds ("10 mph"). Normalize on ingest.
- **Grid drift**: gridX/gridY may change; refresh `/points` daily and on 404.
- **Alert geometry**: many alerts are zone/county based with null `geometry`; match via `affectedZones`, not polygons.
- **403 on missing UA**, and CDN may return stale alert lists up to 5 s.

## 6. OPINION: recommended strategy

Use a two-tier scheduler: (1) **alerts** — one batched `/alerts/active?zone=...&status=actual` call every **20 s** covering all configured locations' forecast+county zones (≤80 zones/request, shard if more); dedupe on `id`, supersede via `references`, expire on `ends ?? expires`. This gives 3 polls per minute and worst-case detection ≈ 20 s + CDN 5 s, meeting M2 with margin and M3 via zone coverage. (2) **weather** — per-gridpoint forecast/hourly refreshed by `expires` (~30–60 min), obs every 60–120 s per unique station, all keyed by gridpoint/station so duplicate locations cost nothing. Single `http.Client` with a global token bucket (~5 req/s) and jittered backoff on 429/5xx. Net load at 25 locations: ~3 alert req/min + ~25 obs req/min + ~1 forecast req/min — trivial for M4/M8/M9.

**Strongest counter-argument**: the rate limit is undocumented and enforced per IP; a 20 s alert poll plus 25 obs polls from a shared office/VPN IP could be throttled, and `point`/`zone` alert filtering relies on correct UGC derivation — a location straddling a county boundary could miss a county-only alert (M3 risk). Mitigation: include both `forecastZone` and `county` UGCs, fall back to `?area=` state polling if zone coverage is in doubt, and degrade poll interval adaptively on any 429/5xx.

## Sources

- https://www.weather.gov/documentation/services-web-api (User-Agent, rate limit statement, cache approach, /points flow, formats)
- https://api.weather.gov/openapi.json (paths, alert params, Alert/Observation/QuantitativeValue schemas)
- Live probes 2026-08-23: `/points/38.8894,-77.0352`, `/stations/KDCA/observations/latest`, `/gridpoints/LWX/97,71/forecast{,/hourly}`, `/alerts/active?zone=...`, `?area=TX` (headers and status codes quoted above)
