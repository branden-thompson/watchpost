# AI-3: Fire Hotspot & Incident Data Sources

Scope: v0.1 hotspots (HUM LEAD ruling OQ-11); evacuation orders deferred to v1.x. NWS alerts (Red Flag / Fire Weather Watch via `/alerts/active` + `fireWeatherZone`) are covered by AI-1 and only referenced here as a complement.

## 1. NASA FIRMS

| Item | Finding |
|---|---|
| Key model | Free MAP_KEY, email registration required. Quota **5000 transactions / 10 min**; large requests may count as multiple. Increases by request. |
| Area endpoint | `https://firms.modaps.eosdis.nasa.gov/api/area/csv/{MAP_KEY}/{SOURCE}/{west,south,east,north}/{days}[/{YYYY-MM-DD}]`. Days 1–5. `world` accepted as bbox. |
| Sources | `VIIRS_SNPP_NRT`, `VIIRS_NOAA20_NRT`, `VIIRS_NOAA21_NRT`, `MODIS_NRT`, `LANDSAT_NRT` (US/CA only), plus `_SP` standard-processing variants. |
| Fields (VIIRS) | `latitude, longitude, bright_ti4, bright_ti5, scan, track, acq_date, acq_time (UTC HHMM), satellite, instrument, confidence (l/n/h), version, frp (MW), daynight (D/N)`. MODIS: `brightness`, `confidence` 0–100, `type` (SP only: 0 veg fire, 1 volcano, 2 static land source, 3 offshore). |
| Latency | URT <1 min (direct readout, much of US/Canada); RT <30–60 min; NRT <3 h (LANCE global). NRT replaces URT records 1–3 h later. |
| Format | CSV only for `/api/area` (JSON not offered; parse CSV in Go). |
| License | NASA open data, "full and open sharing." Attribution: "NASA FIRMS" / LANCE. No caching restrictions stated; respect quota. |

Complement: FIRMS says *where* heat is; NWS Red Flag says *whether conditions favor spread*. Show both in one panel.

## 2. Keyless Hotspot Alternatives

| Source | Key? | Cadence | Format | Coverage | Fields |
|---|---|---|---|---|---|
| NOAA HMS fire (`ospo.noaa.gov/Products/land/hms/data/latest_fires_final.kml`; archive `satepsanone.nesdis.noaa.gov/pub/FIRE/web/HMS2/` as shp/txt) | No | Analyst-QC'd; online ~08:00 ET, updated through day | KML, Shapefile, ASCII text | North America / CONUS | lon, lat, yearday, time, satellite, method, ecosystem, FRP |
| NOAA HMS smoke (`latest_smoke_final.kml`) | No | Twice daily (11–12 ET, 19–20 ET) | KML, Shapefile | North America | satellite, start, end, density (light/medium/heavy) |
| NIFC WFIGS Incidents/Perimeters (ArcGIS FeatureServer) | No | Continuous (IRWIN sync) | JSON, GeoJSON, PBF | US | see §3 |
| Copernicus EFFIS (WMS/WFS) | No (license acknowledgement) | Daily layers (1/7/30-day, MODIS+VIIRS) | WMS, WFS (shp/SpatiaLite) | Europe, Middle East, N. Africa | active fires, burnt areas |
| USGS/LANDFIRE | No | Annual | Raster | US | Fuel/vegetation only; not real-time, skip |

Takeaway: HMS is the only keyless *hotspot* feed for the US, but it is analyst-curated and slower than FIRMS. HMS KML is a viable zero-key default; FIRMS is the "bring your own key" upgrade.

## 3. Incidents + Perimeters (NIFC WFIGS)

- Incidents (point): `https://services3.arcgis.com/T4QMspbfLg3qTGWY/arcgis/rest/services/WFIGS_Incident_Locations_Current/FeatureServer/0/query`
- Perimeters (polygon): `.../WFIGS_Interagency_Perimeters_Current/FeatureServer/0/query`
- Query: `?where=IncidentTypeCategory='WF'&geometry=west,south,east,north&geometryType=esriGeometryEnvelope&inSR=4326&spatialRel=esriSpatialRelIntersects&outFields=IncidentName,FireDiscoveryDateTime,PercentContained,POOState,ModifiedOnDateTime_dt&outSR=4326&f=geojson`
- Incident fields confirmed: `IncidentName, FireDiscoveryDateTime, PercentContained, POOState, IncidentTypeCategory (WF/RX/CX), ModifiedOnDateTime_dt`; `DailyAcres` is **not** in the Incidents layer — use `IncidentSize`/`CalculatedAcres`. Perimeters use `attr_*`/`poly_*` prefixes (`attr_IncidentName, poly_GISAcres, attr_PercentContained, attr_POOState, attr_FireDiscoveryDateTime, attr_IrwinID`).
- maxRecordCount 2000; no key; copyrightText empty (public domain, NIFC Open Data).
- InciWeb RSS (`inciweb.wildfire.gov/incidents/rss.xml`): title, link, description (with lat/lon and narrative embedded as text), pubDate; no geo tags. Useful for human-readable updates only.

## 4. Evacuation Orders (v1.x planning)

| Source | Programmatic? | Licensed? | Verdict |
|---|---|---|---|
| Genasys Protect / Zonehaven | No public API found (docs host unreachable; site has no developer section) | Unknown; zone data is proprietary to agencies | Reject unless a partner API is offered |
| Watch Duty | No public API; ToS page not retrievable (404/blank) | Community-sourced, not redistributable | Reject (scrape-only) |
| CAL FIRE `fire.ca.gov/umbraco/api/IncidentApi/List` | JSON exists but returned 403 to automated fetch (bot-blocked) | No stated license | Fragile; treat as unofficial |
| IPAWS/WEA via NWS CAP | Yes — `/alerts/active` already returns CAP-derived `Evacuation Immediate` events relayed by NWS | Public domain | **Accept** (partial coverage, but real) |
| County ArcGIS evac layers | Yes, ad hoc per county | Public | Accept as opt-in registry |

Path: v1.x parses NWS CAP evacuation events first, then an optional per-county ArcGIS layer list.

## 5. Correlation / UX Inputs

- Radius: default 25 km ring, "nearby" <10 km, "watch" 10–50 km; user-configurable.
- Filter: VIIRS `confidence` in {nominal, high} by default; show low only with `--all`. FRP ≥ 5 MW to surface; ≥ 50 MW = bold.
- Dedupe: cluster detections within ~375 m (VIIRS pixel) and same `acq_date`; key = rounded lat/lon (3 dp) + date. Across satellites, keep max FRP per cluster and count passes.
- Day/night: `daynight=N` detections are cleaner (no sun glint); note "night detection" in UI.
- Smoke: overlay HMS smoke polygon density for the user's point ("heavy smoke overhead").
- False alarms: static sources (refineries, flares, cement plants) repeat daily at the same coordinates — suppress points seen ≥5 of last 7 days with no WFIGS incident nearby; RX incidents (`IncidentTypeCategory='RX'`) label as prescribed burn; agricultural burns are small FRP/short-lived, so FRP + persistence thresholds cover most.

## 6. OPINION

v0.1: HMS KML (keyless) as default hotspot source + WFIGS incidents for names/containment; FIRMS VIIRS (NOAA20/21) as optional keyed upgrade enabled in `watchpost setup`. Refresh: HMS/WFIGS every 15 min, FIRMS every 10 min (well under quota). Radius 25 km, nominal+ confidence, FRP ≥ 5 MW. v1.x evac: NWS CAP evacuation events + opt-in county ArcGIS layers; no Watch Duty/Genasys scraping.

Strongest counter-argument: HMS is analyst-delayed (hours) and North-America-only, so a keyless default may miss a fast-moving fire that FIRMS URT would catch in minutes. Mitigation: nag once for a free FIRMS key when the user is in a fire-weather zone.

## Sources

- https://firms.modaps.eosdis.nasa.gov/api/area/
- https://www.earthdata.nasa.gov/data/tools/firms/faq
- https://www.earthdata.nasa.gov/data/tools/firms/active-fire-data-attributes-modis-viirs
- https://www.ospo.noaa.gov/products/land/hms.html
- https://services3.arcgis.com/T4QMspbfLg3qTGWY/arcgis/rest/services/WFIGS_Incident_Locations_Current/FeatureServer/0
- https://services3.arcgis.com/T4QMspbfLg3qTGWY/arcgis/rest/services/WFIGS_Interagency_Perimeters_Current/FeatureServer/0
- https://inciweb.wildfire.gov/incidents/rss.xml
- https://forest-fire.emergency.copernicus.eu/applications/data-and-services
- Unverified (fetch blocked): https://www.fire.ca.gov/incidents, https://app.watchduty.org, https://protect.genasys.com
