---
title: "0.18.0 W21 — the About window's credits, the HUM LEAD's mock"
date: 2026-10-02
phase: BUILD
sev: SEV-2
authority: HUM LEAD
status: "REFERENCE for W21. The mock as given; not yet built."
---

# The About window's credits — the HUM LEAD's mock (2026-10-02)

**HUM LEAD, 2026-10-02:** "I also want to clean up our [a] about modal to remove duplication of data
source credits. Here's the updated mock - make of note of it for when we're finished, this can also be
added later since this is largely a UI / modal content change."

**And:** "We can also use the same 'smart column' approach to break some of this up and turn it into a
two column layout on wider terminals if needed, or provide our vertical scroll control on the right
for shorter terminals."

The mock, verbatim:

```text
┌────────────────────────────────────────────────────────────────────────────┐
│              WATCHPOST    v. checkpoint/pre-w14-69-ge46d1673               │
│                                                                            │
│   ! NOT INTEDED AS A SUBSTITUTE FOR OFFICIAL WARNING SOURCES OR DEVICES    │
│   ! WEATHER RELAYS MAY BE DELAYED                                          │
│   ! NOT INTENDED FOR LIFE SAFETY USE                                       │
│                                                                            │
│   All sources free to use with attribution.                                │
│   ──────────────────────────────────────────────────────────────────────   │
│   DATA SETS PROVIDED BY:                                                   │
│                                                                            │
│   NATIONAL OCEANIC ATMOSPHERIC ADMINISTRATION (NOAA)                       │
│     NWS     - National Weather Service                   api.weather.gov   │
│     NDBC    - National Data Buoy Center                    ndbc.noaa.gov   │
│     CO-OPS  - Tides & Currents                                             │
│     HMS     - Wildfire Satellite Hotspots                  ospo.noaa.gov   │
│     NWR     - Transmitter List                                             │
│     MRMS    - Current Radar for Maps                                       │
│     HRRR    - Radar Ahead (forecast) for Maps                              │
│     NDFD    - National Weather Forecast Data       graphical.weather.gov   │ 
│                                                                            │
│   NATIONAL AERONAUTICAL AND SPACE ADMINISTRATION                           │
│     FIRMS   - Fire Hotspots, API Key Required         earthdata.nasa.gov   │
│                                                                            │
│   NATIONAL INTERAGENCY FIRE CENTER                                         │
│     WFIGS   - Wildfire Incidents                                nifc.gov   │
│                                                                            │
│   NATIONAL WEATHER RADIO                                                   │
│     wxradio.org & weatherUSA (community)                                   │
│                                                                            │
│   UNITED STATED ENVIRONMENTAL PROTECTION AGENCY                            │
│     AQI     - U.S. EPA AirNow                                              │
│                preliminary data, not fully verified                        │
│                                                                            │
│     UVI     - U.S. EPA (Envirofacts)                                       │
│                                                                            │
│   UNITED STATES GEOLOGICAL SURVEY                                          │
│     Earthquake Hazards Program                       earthquake.usgs.gov   │
│                                                                            │
│   IOWA ENVIRONMENTAL MESSONET                                              │
│     IEM     - Radar & Radar Ahead for Maps                                 │
│                                                                            │
│   OPEN-METEO                                                               │
│     geocoding (CC BY 4.0)                                                  │
│     Wind Data                                                              │
│     Supplemental Temperature Data                                          │
│     Supplemental UVIndex Data                                              │
│     Off-shore Wave Data, Interpolated                                      │
│     Rain and Snow total forecasts                                          │
│                                                                            │
│   ──────────────────────────────────────────────────────────────────────   │
│                                                                            │
│   Built with:                                                              │
│                                                                            │
│   GO 1.27.0 | BubbleTea | LipGloss | go-tuimaps                            │
│   Stylized Terminal UI Design System (STUDS)                               │
│                                                                            │
│   Built with ♥ by Branden R. Thompson                                      │
│   github: branden-thompson                                                 │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

## Sources the mock leaves out

**HUM LEAD, 2026-10-02:** "there might be a couple I missed as well (like OpenFreeMap, etc. those
should be added as a group with their contributions like the other data groups in the mock".

Found by listing every host the code names and every data set it embeds. Each one gets a group, or a
line in its agency's group, with what it supplies:

| Source | What it supplies | Where | Terms |
|---|---|---|---|
| **OpenFreeMap**, © OpenMapTiles, © OpenStreetMap contributors | the map's basemap tiles | `tiles.openfreemap.org` | **ODbL: the OpenStreetMap credit is required.** It is in `mapCredits` today |
| **GeoNames** | the offline place and ZIP index: cities15000 and US postal codes, embedded | `domains/locations/geodata` | **CC BY 4.0: attribution required** |
| **NOAA NHC**, National Hurricane Center | current storms, Atlantic and Eastern Pacific | `www.nhc.noaa.gov` | public domain; a line in the NOAA group |
| **Piper voices** (rhasspy), via Hugging Face | the offline speech voices, fetched on demand | `huggingface.co` | each voice model carries its own licence; check them |
| GitHub | the hourly "is there a newer Watchpost?" check | `api.github.com` | not a data set; perhaps a note, not a credit |

Hosts behind sources the mock already names:

- MRMS comes from `opengeo.ncep.noaa.gov`.
- CO-OPS comes from `api.tidesandcurrents.noaa.gov`.
- NIFC WFIGS comes from `services3.arcgis.com`. The mock shows `nifc.gov`.
- The NWR transmitter list is embedded, credited as "NOAA NWR transmitter list (weather.gov/nwr)".
- Open-Meteo's air-quality host (`air-quality-api.open-meteo.com`) is named in the code. Confirm
  whether it is still asked since D-193 put AirNow first.

## To settle when W21 opens

**Settled, D-220 (HUM LEAD, 2026-10-02):** "yes - fix typos, ensure we meet the licence requirements
while still respecting the intended format of the mock." The spellings below are corrected, and every
required credit is present in the mock's own format.

The points:

- **Spellings.** The mock reads "INTEDED", "AERONAUTICAL", "UNITED STATED" and "MESSONET". The usual
  names are "INTENDED", "National Aeronautics and Space Administration", "United States" and "Iowa
  Environmental Mesonet". Confirm before building.
- **The version line** is the build's own (`checkpoint/pre-w14-69-ge46d1673` was this build's),
  drawn from the binary, never a literal.
- **Credits each source's terms require** must still appear. Open-Meteo's CC BY 4.0 asks for a
  statement of change, and the map interpolates its data (`OpenMeteoCredit`). Check the AirNow, HMS
  and FIRMS terms the same way, and that every source the app contacts is listed (the closed lists:
  `mapSourceList`, the station's providers).
- **The "smart column" layout:** two columns where the window is wide enough, and the vertical
  scroll control on the right where it is short. Use the app's existing column and scroll components
  (D-147).
