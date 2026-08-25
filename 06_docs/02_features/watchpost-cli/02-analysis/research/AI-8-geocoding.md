# AI-8: Geocoding Research (Watchpost CLI v0.1)

Scope: resolving city names / US zips to lat-long + metadata, with type-ahead hints and zip-alongside-label. Zero API keys. Weather data is out of scope (see provider research).

## 1. Keyless geocoder comparison

| Source | Key? | Rate limit | Desktop-client ToS | Prefix/type-ahead | Postal codes | Timezone | Coverage | Latency (est.) |
|---|---|---|---|---|---|---|---|---|
| Open-Meteo Geocoding | No (non-commercial) | <10k/day, 5k/hr, 600/min | CC-BY 4.0 attribution | **Yes** - "normalized prefix matching" at 3+ chars (2 chars = exact) | `postcodes[]` array (from GeoNames) | **Yes** (`timezone`) | Global (GeoNames-based) | ~100-300 ms |
| Nominatim (OSM) | No | "absolute maximum of 1 request per second"; custom User-Agent mandatory | **Forbids type-ahead**: "Auto-complete search: This is not yet supported by Nominatim and you must not implement such a service on the client side using the API." Also "Results must be cached on your side." | No (prohibited) | Yes (`address.postcode`, when OSM has it) | No | Global | ~200-600 ms |
| US Census Geocoder | No | Not published | Public domain | No (address matching only; city-only input unreliable) | Yes (address-level) | No | US only | ~300-1000 ms |
| Photon (komoot) | No | "reasonable limit"; "extensive usage will be throttled or completely banned"; no availability guarantee | Demo server; self-host recommended | **Yes** - search-as-you-type is a core feature | Yes (`postcode` when present) | No | Global (OSM) | ~100-300 ms |
| GeoNames web service | **Free account username** (a key in practice) | 10k credits/day, 1k/hr | CC-BY 4.0 | Yes (`name_startsWith`) | Via separate `postalCodeSearch` | Yes | Global | ~200-500 ms |
| Zippopotam.us | No | Not published | ODbL | No (exact zip or state/city) | Zip -> place and place -> zips | No | ~60 countries | ~100-300 ms |

Takeaway: only Open-Meteo and Photon permit per-keystroke prefix search cleanly; Open-Meteo is the only one that also returns timezone, population and postcodes in one response, which is exactly the disambiguation payload we need.

## 2. Type-ahead feasibility

Online: Open-Meteo fits. Per-keystroke at 3+ chars with a 150-250 ms debounce stays far below 600/min even with fast typists. Photon is a viable fallback but its "demo server" wording makes it a weak primary. Nominatim is out for type-ahead by explicit policy.

Offline bundled dataset (for zero-network hints and resilience):

- GeoNames `cities15000.zip` 3.3 MB (about 26k cities), `cities5000.zip` 5.5 MB, `cities500.zip` 13 MB (about 200k). CC-BY 4.0. Includes admin1, country, population, lat/long, timezone.
- US zips: USPS data is not redistributable. Alternatives: GeoNames postal `US.zip` 619 KB (about 41k rows; postal code, place name, state, lat/long; CC-BY 4.0); Census ZCTA gazetteer ~1 MB, public domain, but ZCTA-only (no city names); SimpleMaps free tier (CC-BY with attribution link; page returned 403 during research, verify before relying on it).
- Binary impact: cities15000 + US postal trimmed to needed columns, gzip-compressed via `go:embed`, adds roughly 2-3 MB to the binary. Decoded in memory: about 70k records x ~100 B = 7-10 MB, plus a sorted-slice prefix index (binary search over lowercased names) or a small trie; no external deps.

## 3. Zip alongside every label (HUM LEAD rule)

- User typed a zip: show it verbatim.
- US city/lat-long: derive "the" zip by nearest-centroid lookup in the GeoNames US postal table (haversine over ~41k rows is sub-millisecond), preferring a row whose place name matches the city. Open-Meteo's `postcodes[]` gives the same answer online (first entry), but it is ordered arbitrarily, so a deterministic rule is needed: pick the lowest-numbered zip whose place name equals the city, else nearest centroid. Mark the zip as representative ("62701"), since cities have many zips.
- Non-US: show the country postal code if GeoNames postal `allCountries` (19 MB, too big to bundle) or Open-Meteo `postcodes[]` provides one; otherwise show the ISO country code in the same slot, e.g. "Paris, Ile-de-France (FR 75001)" vs "Reykjavik, IS". Do not invent a US-style zip for non-US places.

## 4. Disambiguation fields

Per candidate: `name`, `admin1` (state/province; US two-letter via GeoNames admin1 code), `country_code`, `population` (rank ties, drop sub-1000 in hints), `latitude/longitude`, `timezone` (IANA, needed to render local time and "as of" timestamps), `feature_code` (filter PPL* only), and the representative postal code from section 3. Render: "Springfield, IL (62701)" / "Springfield, MO (65801)". All fields are present in Open-Meteo responses and in the GeoNames dumps.

## 5. Caching and offline

- Persist resolved locations in the config dir (`$XDG_CONFIG_HOME/watchpost/locations.json`): query string, canonical label, zip, lat/long, timezone, source, resolved_at. Cache keyed by normalized query; never expire geocodes (places do not move), but re-resolve on explicit user request.
- Offline: hints from the embedded index and previously resolved places must work with no network; only weather fetch fails, and it should show the last cached observation with a stale badge. Nominatim-style "must cache" is good hygiene regardless of source.

## 6. Opinion: recommended v0.1 architecture

**Hybrid, embedded-first.** Bundle GeoNames `cities15000` + GeoNames US postal `US.zip` (both CC-BY 4.0; attribution in `--about` and README) via `go:embed` for instant, offline, ToS-free type-ahead and zip derivation. Use Open-Meteo Geocoding online only when the embedded index has no prefix match (small towns, non-US postal codes), with the result written to the cache. Skip Nominatim entirely in v0.1 (no type-ahead, timezone absent, 1 rps). Photon as an optional secondary if Open-Meteo is down.

Strongest counter-argument: the embedded index adds 2-3 MB and a data-refresh chore, and cities15000 misses many US towns users will type (population under 15k), so the "online fallback" path will fire often enough that a pure Open-Meteo design with a local cache would be simpler and nearly as good. Mitigation: bundle `cities5000` (+2 MB) if fallback rate proves high in testing.

## Sources

- Nominatim Usage Policy: https://operations.osmfoundation.org/policies/nominatim/
- Open-Meteo Geocoding API: https://open-meteo.com/en/docs/geocoding-api
- Open-Meteo Terms: https://open-meteo.com/en/terms
- GeoNames export/web service credits: https://www.geonames.org/export/
- GeoNames dumps (sizes): https://download.geonames.org/export/dump/
- GeoNames postal codes (license, fields): https://download.geonames.org/export/zip/
- Photon: https://github.com/komoot/photon
- Zippopotam.us: https://api.zippopotam.us/
- US Census Geocoder API: https://geocoding.geo.census.gov/geocoder/Geocoding_Services_API.pdf
- Census Gazetteer (ZCTA): https://www.census.gov/geographies/reference-files/time-series/geo/gazetteer-files.html
- SimpleMaps US zips (not verified, HTTP 403): https://simplemaps.com/data/us-zips
