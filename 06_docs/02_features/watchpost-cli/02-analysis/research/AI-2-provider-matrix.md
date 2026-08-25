# AI-2: Non-NWS Provider Matrix (Watchpost CLI, DISCOVER)

Scope: global/paid providers only. NWS and geocoding are covered by sibling research docs. Claims verified against official pages 2026-08-23; items marked "(unverified)" could not be confirmed from a fetchable official page.

## 1. Provider Matrix

| Provider | Key? | Free quota | Commercial / attribution | Caching / redistribution | Current fields | Forecast horizon | Alerts | Geocoding | Coverage | Update cadence |
|---|---|---|---|---|---|---|---|---|---|---|
| Open-Meteo [1][2] | **No** | 600/min, 5k/hr, 10k/day, 300k/mo | Free tier non-commercial only; data CC-BY 4.0 (attribution) | No explicit cache limit beyond CC-BY | temp, RH, apparent, precip/rain/showers/snow, weather_code, cloud, pressure, wind 10m speed/dir/gust, is_day; hourly adds visibility, dewpoint, UV | 7d default, up to 16d hourly+daily | No | Yes (separate endpoint) | Global, multi-model | Per model: HRRR/GFS hourly, ICON 3h, ECMWF 6h |
| WeatherAPI.com [3][4] | Yes | 100k/mo; 3-day forecast; 1-day history | Commercial OK; "Powered by" link "appreciated" | Not stated on pricing page | temp, wind mph/kph + gust, vis, UV, precip, condition, feels-like | 3d free (up to 14d paid) | Limited on free; gov alerts w/ severity | Yes (search.json) | Global | Realtime endpoint |
| Weatherstack [5][6] | Yes | 100/mo; **no forecast on free** (Professional+) | Not stated on product page | Not stated | temp, wind, pressure, precip, humidity, vis, UV | 7d+ on paid only | No | Built-in location lookup | Global | "Real-time" (vague) |
| OpenWeatherMap [7][8] | Yes | 60/min, 1M/mo (legacy); One Call 3.0 1,000/day | Data CC BY-SA 4.0 / ODbL -> attribution + share-alike | CC BY-SA share-alike applies to redistributed data | temp, feels, pressure, humidity, dew, UV, clouds, vis, wind speed/deg/gust, rain/snow 1h | minutely 1h, hourly 48h, daily 8d | Yes (national alerts, One Call) | Yes (Geocoding API) | Global | ~10 min |
| Tomorrow.io [9] | Yes | Per-plan; not published in docs (commonly cited 500/day, 25/hr, 3/s - unverified) | Commercial plans; headers expose limits (enterprise) | Not stated | temp, wind, precip prob/intensity, UV, vis, cloud | 1m/1h/1d timelines (up to 15d) | Yes (paid events) | Yes | Global | Proprietary, frequent |
| Visual Crossing [10][11] | Yes | 1,000 records/day | Commercial + non-commercial OK; attribution "Weather Data Provided by Visual Crossing" required at free/Pro level | Not stated | ~100 elements: temp, feels, wind+gust, precip+prob, UV, vis, cloud, solar | 15d daily+hourly | Yes (events/alerts) | Built-in address resolve | Global | Hourly-ish |
| Pirate Weather [12][13] | Yes | 10k/mo (20k with $2/mo) | Open-source; attribution to NOAA models | Not stated | Dark Sky schema: temp, wind speed/gust/bearing, precipProbability/Intensity by type, UV, vis, nearestStorm, AQI | minutely 1h, hourly 48h, daily 7-8d | Yes (US) | No (coords; city string accepted) | Global (GFS/ECMWF) + US HRRR/NBM | Model cycles |

## 2. Keyless Global Choice: Open-Meteo

Open-Meteo is the only keyless option in the set and meets every v0.1 need:

- **Non-commercial clause**: free API "only for non-commercial purposes"; personal/home/research explicitly allowed [1]. A free OSS CLI on a user's own machine fits; a hosted SaaS front-end would not.
- **Rate limits**: 10k/day, 600/min [1] - ample for a single-user TUI polling every 5-10 min (~300/day).
- **Attribution**: CC-BY 4.0 -> show "Weather data by Open-Meteo.com" in footer/about [1].
- **Models**: `best_match` auto-selects the highest-resolution model per location; specific models (ECMWF IFS, GFS, ICON, HRRR, regional "seamless" blends) selectable [2]. Exposing the model name in the UI explains disagreement with NWS.
- **Dashboard fields** [2]: wind_speed_10m / wind_direction_10m / wind_gusts_10m (current+hourly); precipitation + precipitation_probability (hourly), daily precipitation_sum/probability_max; uv_index (hourly, daily max); visibility (hourly only, not `current`); weather_code (WMO); pressure_msl; cloud_cover; dew_point; apparent_temperature; is_day.
- Units/timezone params (`temperature_unit`, `wind_speed_unit`, `precipitation_unit`, `timezone=auto`, ISO or unix) [2].

Gaps: no alerts (NWS covers US; no keyless global alert source), visibility is hourly-only, no station observations (everything is model-derived).

## 3. Harmonization Inputs

| Provider | Default units | Time handling | Why it legitimately disagrees |
|---|---|---|---|
| Open-Meteo | C, km/h, mm, hPa; switchable [2] | ISO-8601 local via `timezone`, or unix | Grid model point (1-11 km), no station; "current" = interpolated model |
| NWS (ref) | SI (wmoUnit) | ISO-8601 with offset | Nearest ASOS station obs (may be 20+ km away), forecast from WFO gridpoint |
| WeatherAPI | Both metric+imperial in one payload [4] | epoch + localtime string | Blended model/station; 15-min-ish |
| OWM | `units=standard/metric/imperial`; K default [7] | unix `dt` + `timezone_offset` | Proprietary blend, 10-min refresh [8] |
| Visual Crossing | `unitGroup=us/metric/uk` | local + epoch | Station-based current + model forecast [10] |
| Pirate Weather | `units=us/si/ca/uk` [13] | unix UTC + tz string | HRRR in US, GFS/ECMWF elsewhere; exposes `flags.sources` |
| Tomorrow.io | `units=metric/imperial` | ISO UTC | Proprietary model |

**Minimal normalized schema** (SI internal; convert at render): `observed_at` (RFC3339 UTC), `tz` (IANA), `temp_c`, `feels_c`, `dewpoint_c`, `humidity_pct`, `pressure_hpa`, `wind_ms`, `wind_dir_deg`, `wind_gust_ms`, `precip_mm_1h`, `precip_prob_pct` (forecast only), `cloud_pct`, `condition_code` (WMO-mapped enum), `is_day`, `source` {provider, model_or_station, distance_km, issued_at}. Hourly/daily arrays reuse the same fields plus `temp_max/min_c`, `uv_index`, `sunrise/sunset`.

**Partial-only fields**: `visibility_m` (Open-Meteo hourly only, NWS obs, OWM, WeatherAPI, Pirate), `uv_index` (not NWS obs), `precip_prob` (not NWS current), `station_distance_km` (NWS only), `model_name` (Open-Meteo/Pirate only), AQI/pollen (paid/partial). Diff view should render "n/a" rather than 0.

## 4. Paid-Tier Value (beyond NWS + Open-Meteo)

1. **Pirate Weather** - minutely precip (1h), US alerts, AQI, Dark Sky schema, open source; 10k/mo free [12]. Best value-per-key.
2. **OpenWeatherMap One Call 3.0** - minutely 1h, global national alerts, air pollution endpoint; 1k/day free [7]. Note CC BY-SA share-alike [8].
3. **WeatherAPI.com** - pollen (NA/EU), AQI with EPA/DEFRA indices, marine, astronomy; 100k/mo free but only 3-day forecast [3][4].
4. **Visual Crossing** - 15d forecast, 50y history, solar/agri elements, alerts; 1k records/day [10].
5. **Tomorrow.io** - richest (lightning? pollen, minute-cast) but quotas undocumented publicly [9]; low hobbyist value.
6. **Weatherstack** - 100/mo, no free forecast [5]; not worth a key.

Lightning and marine are not free anywhere in this set; treat as out of scope.

## 5. Risks

- **Open-Meteo non-commercial**: blocks anyone bundling Watchpost into a paid/ad-supported product; document in README [1].
- **OWM CC BY-SA**: share-alike could taint cached/exported data files if redistributed [8]; keep OWM opt-in.
- **Attribution obligations**: Open-Meteo (CC-BY), Visual Crossing (required text at free level) [11], WeatherAPI (requested) -> a mandatory provider credit line in the TUI footer/`--about`.
- **Weatherstack HTTPS**: product page says free includes HTTPS [5] but historically free was HTTP-only; verify before shipping.
- **Key leak**: keys go in URL query strings for WeatherAPI/OWM/Pirate [4][7][13] -> redact from logs/debug output; store in OS keychain or `$XDG_CONFIG_HOME` with 0600; never in shell history examples; support env vars.
- **Quota exhaustion**: cache responses per provider (Open-Meteo 10 min, OWM 10 min) and surface remaining-quota headers (Pirate `Ratelimit-Remaining` [13]).

## 6. Opinion

**v0.1 (keyless)**: NWS (US obs/forecast/alerts) + Open-Meteo `best_match` (global current+forecast, and the US second opinion for diff view). Show model name and station distance so disagreements are explainable.

**v0.2 (additive, key-gated)**: Pirate Weather first (minutely + alerts + AQI, permissive licence), then OpenWeatherMap One Call (global alerts, minutely) and WeatherAPI.com (pollen/AQI/astronomy). Skip Weatherstack and Tomorrow.io.

**Strongest counter-argument**: Open-Meteo's non-commercial clause and lack of alerts mean non-US users get no warnings and any future commercial distribution requires re-licensing; OWM One Call (1k/day free, global alerts) would close the alert gap at the cost of mandatory keys in v0.1. Ruling stands because keyless-first UX outweighs alert coverage for v0.1, and OWM's CC BY-SA is its own licence risk.

## Sources

1. https://open-meteo.com/en/terms
2. https://open-meteo.com/en/docs
3. https://www.weatherapi.com/pricing.aspx
4. https://www.weatherapi.com/docs/
5. https://weatherstack.com/product
6. https://weatherstack.com/
7. https://openweathermap.org/api/one-call-3
8. https://openweathermap.org/faq
9. https://docs.tomorrow.io/reference/rate-limiting.md
10. https://www.visualcrossing.com/weather-api/
11. https://www.visualcrossing.com/weather-data-editions/
12. https://docs.pirateweather.net/en/latest/
13. https://docs.pirateweather.net/en/latest/API/
