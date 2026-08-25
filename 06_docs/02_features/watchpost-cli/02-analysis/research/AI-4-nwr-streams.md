# AI-4: NOAA Weather Radio (NWR) Audio Stream Research

Scope: stream sources, transmitter lookup, ToS, reliability, fallbacks. Decoders/players are out of scope (see companion research); formats are reported so they can be matched. Probed 2026-08-23 with `curl`.

## 1. Stream sources

| Source | Discovery | Format / codec | Bitrate | HTTPS | ICY metadata | URL stability |
|---|---|---|---|---|---|---|
| **NWS (official)** | None. NWS only publishes station listings, coverage maps, outages; no audio stream [1][2] | n/a | n/a | n/a | n/a | n/a |
| **wxradio.org / noaaweatherradio.org** (Saratoga-Weather) | Icecast `status-json.xsl` lists every mount (117 sources at probe time; `listenurl` like `http://wxradio.org:8000/CO-Denver-KEC55`); mount name = `ST-City-CALLSIGN` [3][4] | Icecast: 113 `audio/mpeg`, 2 `audio/mp3`, 1 `audio/ogg`, 1 `audio/aacp` (probe) | mostly 32 kbps; some 24–320 | Yes: `https://wxradio.org/<mount>` via nginx proxy returns `Content-Type: audio/mpeg`, `icy-br`, `icy-name`, `Access-Control-Allow-Origin: *` (probe) | Yes (`icy-*` headers) | Good: mount names are deterministic |
| **weatherUSA radio** | HTML table by state/city/call sign; hosted pattern `https://radio.weatherusa.net/NWR/<CALLSIGN>.mp3`, plus links to wxradio.org, Zeno, private Icecast hosts [5] | Icecast 2.4.3, `audio/mpeg` (probe); some third-party links are `.ogg` [5] | 32–128 kbps (probe) | Yes (Icecast directly serves TLS; status JSON lists 117 mounts) | Yes | Mixed: call-sign mounts are stable, but not every transmitter is hosted (KEC55 returned 404) |
| **Broadcastify / RadioReference** | Web directory only; programmatic access forbidden [6] | proprietary player | – | – | – | Unusable (ToS) |

Note: no HLS anywhere observed; all community sources are plain Icecast HTTP chunked MP3 (rare Ogg Vorbis / AAC+).

## 2. Transmitter lookup

| Dataset | URL | Format | Fields |
|---|---|---|---|
| Station listing by state | `weather.gov/nwr/station_listing` -> state pages [1] | HTML (dynamic tables; not scrape-friendly via plain fetch) | site, call sign, frequency, power |
| County coverage by state | `weather.gov/nwr/county_coverage?State=XX` [7] | HTML | County, Partial County, SAME code, Station ID, Frequency, Status |
| SAME code list | `weather.gov/source/nwr/SameCode.txt` [8] | CSV `0SSCCC,County, ST` (SAME = "0"+FIPS) | SAME, county, state |
| NWR/EAS partial-county polygons | `weather.gov/gis/NWRPartialCounties` -> `cs18mr25.zip` [9] | Shapefile | FIPS, CWA, LON/LAT centroid, ENTIRESAME/AREA_SAME; **no transmitter points** |
| Find My Station | `weather.gov/nwr/station_search` [10] | HTML form (address/zip) | nearby call signs, coverage links |
| Outages | `weather.gov/nwr/outages` [11] | HTML table + RSS | State, Transmitter, Freq, WFO, Callsign, Status |

Mapping flow: lat/long -> county FIPS (NWS `/points` already gives this) -> `county_coverage` row -> call sign -> stream mount by call sign. NWS publishes no machine-readable transmitter CSV with coordinates, so ship a **vendored table** (call sign, freq, lat/lon, covered SAME codes) scraped once at build time. SAME codes are useful: they equal "0"+county FIPS [8], so NWS alert `geocode.SAME` arrays correlate directly to the transmitter's county list.

## 3. Legal / ToS

| Source | Third-party client | Relay/re-stream | Attribution / other |
|---|---|---|---|
| wxradio.org | Streams "intended for direct listening by end users"; direct playback in a personal client is consistent; heavy use -> IP blocklist [3] | "Harvesting/relaying ... not permitted without express permission" [3] | No API; don't poll `status-json.xsl` aggressively; carries "not for protection of life or property" disclaimer [3][4] |
| weatherUSA | No ToS page found (404); only disclaimer re: delays [5] | Unknown; assume same as wxradio | Show delay disclaimer |
| Broadcastify | **Prohibited**: scripts, headless clients, ffmpeg, API clients; commercial license required [6] | Prohibited [6] | Exclude entirely |
| NWS | Public-domain data; no stream | – | – |

Exposure: wxradio is a hobbyist service; a popular CLI could be seen as "harvesting". Mitigation: one connection per user, no server-side proxy, contact operator for permission, honor 403/blocklist gracefully.

## 4. Reliability

- wxradio lists 117 mounts vs. ~1,000+ NWR transmitters; coverage is partial and volunteer-fed; status JSON drifted 161 -> 151 -> 117 across three probes in one hour [3][4].
- Buffer latency up to 30 s (wxradio) or 10 s–2 min (weatherUSA) [3][5].
- Detection: treat HTTP 404/400 from Icecast as "mount down"; treat no bytes for >15 s, or decoded RMS below threshold for >60 s, as "silent" (NWR never goes silent by design). Re-check `status-json.xsl` (cached >=5 min) and fail over: wxradio HTTPS -> weatherUSA `NWR/<CALL>.mp3` -> next-nearest transmitter -> text fallback.
- NWS transmitter outages are separately listed with RSS [11]; surface them so users know silence is upstream.

## 5. Alternatives when no stream exists

- NWS text products via api.weather.gov (`HWO`, `AFD`, `NOW`) rendered in the TUI.
- NWR "broadcast schedule"/on-air text is not published by NWS as a product.
- Local TTS (e.g. `say` on macOS) of warnings — optional, not pure-Go on all platforms; note only.

## 6. Opinion

**Primary:** wxradio.org via `https://wxradio.org/<ST-City-CALL>` (HTTPS, ICY metadata, CORS, deterministic mounts, JSON directory). **Fallback:** weatherUSA `https://radio.weatherusa.net/NWR/<CALL>.mp3`, then a user-supplied URL. Lookup: location -> county FIPS -> vendored transmitter table -> call sign -> mount lookup in cached `status-json.xsl` -> play. Decoder requirements to match: Icecast chunked HTTP, MP3 (MPEG-1 Layer III, 32 kbps typical), optional Ogg Vorbis/AAC+; ICY header parsing.

**Strongest counter-argument:** both sources are volunteer services whose terms bar "relay/harvesting" and lack an explicit third-party-client grant; the operator could block the app's traffic at any time, and coverage is only ~10% of transmitters. If HUM LEAD wants guaranteed v0.1 availability, tune-in must be framed as best-effort with a text fallback, and the operator should be contacted before release.

## Sources

1. https://www.weather.gov/nwr/station_listing
2. https://www.weather.gov/nwr/Maps
3. https://wxradio.org/
4. https://wxradio.org/status-json.xsl (and https://wxradio.org:8443/status-json.xsl)
5. https://weatherusa.net/radio/?tune=KEC55
6. https://www.broadcastify.com/terms/
7. https://www.weather.gov/nwr/counties
8. https://www.weather.gov/source/nwr/SameCode.txt
9. https://www.weather.gov/gis/NWRPartialCounties
10. https://www.weather.gov/nwr/station_search
11. https://www.weather.gov/nwr/outages
12. https://noaaweatherradio.org/
