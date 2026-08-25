# B5 Build Log — Fire (HMS / WFIGS / FIRMS)

| Field | Value |
|---|---|
| Milestone | B5 (added at the 0.9.0 BUILD-exit review: "Build Exit NOT Approved; we need to include initial FIRMS (fire data) in 0.9.0") · BUILD · SEV-0 |
| Date | 2026-08-25 |
| Gate | Second BUILD-exit red-team pass (addendum to `08-reports/red-team-build.md` and `build-report.md`) |

## HUM LEAD decisions (2026-08-25)

- Keep HMS and WFIGS as the keyless default; FIRMS is the keyed upgrade — instructions to get a key
  and to test the feature are required (README "Fire", UAT 99.6).
- Fire is another alert type: a row mark plus a FIRE section in the location detail.
- Thresholds "fine for now — make it configurable": `[fire]` in config.toml, defaults filled in.

## Delivered (data first, then UI — per the standing order)

- `domains/fire`: `Rules` (+ `DefaultRules`, `Valid`), confidence rank, `Keep`, `Near`, `Bounds`,
  `Cluster` (~0.003° + UTC day, keep max FRP, nearest first), `Age`. Tests.
- `domains/fire/hms`: NOAA-NESDIS HMS `fireAllSats.kmz` — live-probed 2026-08-25 (1.4 MB, 10-min
  refresh, merged `kmls/hms_fire<date>.kml`, ~25,769 placemarks, description fields
  Lon/Lat/YearDay/Time/Satellite/Method/Ecosystem/FRP). One download serves the watchlist
  (client cache 10 min); streaming XML under a 200k-placemark cap and a 64 MB entry cap; every
  location gets a `FireState` (empty is data, not failure). Test builds a KMZ in-test.
- `domains/fire/wfigs`: NIFC `WFIGS_Incident_Locations_Current` — one national GeoJSON query
  (`IncidentTypeCategory='WF'`, 2000 rows), incidents inside the ring, largest first, cap 5;
  acreage IncidentSize → FinalAcres → DiscoveryAcres → InitialResponseAcres (young incidents carry
  none — verified live against the two near Oceanside). Test over a fixture.
- `domains/fire/firms`: NASA FIRMS area CSV API, VIIRS NOAA-20/21 NRT, one request per location
  per source per 10 min, header-driven CSV, confidence l/n/h + numeric. Registered only when a
  MAP_KEY is stored (`Enabled()`); the key is a path segment that httpx already redacts. Tests.
- `platform/snapshot`: fire kept per provider, `mergeFire` at Snapshot (cross-provider hotspot
  dedupe keeping max FRP nearest first; incidents by name largest first). Test.
- `platform/config`: `Fire` section + `WithDefaults`; unread fields (`Playlist`, `TTSCmd`,
  `StreamURLOverride`, `Provider.Enabled`) removed. Test.
- `app`: `fireProviders` / `fireRules` (one owner for the knobs; the bold threshold reaches the
  dashboard as `tty.Config.FireBoldMW`), fire tier 10 min (priority) / 15 min (RECENT), report
  wiring, attribution cases, credits (≤ 52 cells). Test: FIRMS listed only when keyed.
- `platform/render`: `LocationRow.Fire/FireHot`, `▲` in marks slot 5 (`FireMark` token: 208 /
  Monochrome 255 / Solarized 166), bold when hot, `^` under `--ascii`. Test.
- `platform/geo.BearingDeg` (+ test) — the compass the swell rows already use names it.
- `modes/tty`: `fireRows` — the FIRE section (always present), `fireHot`, `fireGlyph`,
  `thousands`; `Config.FireBoldMW`. Test with colour on.
- `modes/report`: `fireLines` (+ `alertLines` extracted to keep `RenderPlain` under P10-04's 40
  statements); parity fixtures extended.
- Docs: README "Fire" (+ FIRMS key instructions, `[fire]` reference, credits), CHANGELOG,
  architecture §11.10, UAT 99, this log; P10-05 density exemptions recorded for the four fire
  packages and `platform/config` (the ratified pure-parser pattern).

## Gates at close (2026-08-25, after the round-3 remediation — UAT 101.4)

`make verify` GREEN · P10 0 live · golangci-lint 0 · staticcheck 0 · `a2dh validate` 18/18 ·
live report and dashboard against real fires (Mineral Wells TX, Owyhee County ID). Details in the
addendum to `08-reports/build-report.md` and Round 3 of `08-reports/red-team-build.md`.

## Also in B5 (UAT 100–101)

- Setup became a window over the dashboard (`[s]`, first run, `watchpost setup`); the B2
  standalone wizard is gone; FIRMS keys take effect live; unkeyed FIRMS reads `off`.
- Location Details chip row consolidated (`+ Watchlist` / `− Watchlist` by membership).

## Carried forward

- FIRMS verified live 2026-08-25 with HUM LEAD's MAP_KEY (Mineral Wells, TX: 203 FIRMS + 166 HMS
  hotspots merged, `provider firms: ok`); the fixtures pin the request shape and the fail-soft paths.
- Fire-weather products (RFW) already arrive through the alerts feed; a dedicated fire-weather
  forecast (NWS FWF) is a candidate for 0.10.
- The `[fire]` knobs are configurable by file; an in-app editor was not asked for.
