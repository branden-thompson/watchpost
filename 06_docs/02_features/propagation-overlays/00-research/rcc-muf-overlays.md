# RCC — MUF / foF2 / eSSN overlays (KC2G-style) for watchpost + go-tuimaps

**Ruled 0.19.0 (D-238, 2026-10-03): all of it - the overlays, the planner and the go-tuimaps additions
(v0.3.0). This is the seed of 0.19.0's DISCOVER, opened once 0.18.0 ships.**

Date: 2026-10-03. Read-only research while batch 132's gate ran. Sources: arodland/prop read through
the GitHub API (not copied here: the repo has no licence), live curl of prop.kc2g.com and GIRO, and a file-read-only audit
of go-tuimaps v0.2.0-rc.34 and watchpost. Inferences are marked [INFERRED].

## 1. The question

Does MUF need substantial go-tuimaps changes? If so, put them in v0.2.0 rather than a v0.3.0. Is a
Go port of KC2G's computation needed, as was done for the map library?

## 2. Findings: the data pipeline (arodland/prop)

- **No licence** (GitHub API 404). All rights reserved by default: the algorithms can be reimplemented
  from reading the code, but the code cannot be ported line for line. Contact: Andrew Rodland, through
  prop.kc2g.com/about.
- About 20 containerised services sharing Postgres; a scheduler every 15 minutes. The deployed code is
  ahead of the repo (the `diffusion` eSSN series and the esfi fields are not in it).
- **Stages and Go-port difficulty:**

| Stage | Language / deps | Port |
|---|---|---|
| Ingest: GIRO through a **private FTP account** (`lftp -u kc2g`), BoM SAO zips, INGV | Perl, SAO-4/SAOXML parsers | easy–moderate; GIRO's public FastChar text endpoint avoids SAO parsing |
| Scheduler | Perl Mojolicious + Minion | trivial |
| eSSN: the SSN in [−20,200] minimising weighted IRI-vs-measurement error | Python scipy, ppigrf, **IRI Fortran by pipe** | hard (needs IRI) |
| Per-station time GP (14 days, fixed kernel) | `george` C++ GP | moderate (gonum Cholesky; match george's kernel conventions) + IRI |
| IRI background map, 181×361 × 25 lead times | **IRI-2020 Fortran** + CCIR/URSI coefficients, IGRF, WMM; HDF5 with the SZ filter | **hard** (the subset KC2G uses is moderate [INFERRED]; NASA PyIRI, MIT, pure NumPy, is a cleaner reference) |
| Spatial assimilation: GP on the unit sphere, MUF through M(D) | george, scipy spline | easy–moderate given a background |
| Rendering | matplotlib / cartopy | easy in a terminal (we draw the grid) |

## 3. Findings: published outputs (no terms stated, no API documentation, CORS `*`)

- **`https://prop.kc2g.com/api/iongrid.bin`**: 12,545,472 bytes of raw little-endian float32,
  laid out `[foF2, M(D)][UT hour 0–23][lat −90..90 at 1°][lon −180..180 at 1°]`. MUF(3000) = foF2 × M(D).
  This is the full assimilated product, ideal for a go-tuimaps Grid. Fetch at most hourly and cache it.
- `api/essn.json?days=N` returns the 24h, 6h and diffusion series, every 15 minutes, history back to
  2020-06 (no cap on `days`).
- `api/stations.json` (filter out stale stations by time) and `renders/current/mufd-normal-now_station.json`.
- `renders/current/{mufd,fof2}-normal-{now,1h..24h}.geojson`: contour LineStrings (`level-value`).
- `hfprop/planner.json?txloc=FN21&rxloc=JO62&txant=ISOTROPIC&rxant=ISOTROPIC&txpow=100`: ITU-R P.533
  (ITURHFProp, not VOACAP) fed with KC2G's iongrid; a 24 h × 10-band table. This is the Broadcaster
  frequency-planning screen's data.
- Avoid `assimilated.h5`: the SZ filter has no pure-Go decoder [INFERRED].

## 4. Findings: GIRO

- Licence CC BY-NC-SA 4.0 (Rules of the Road): non-commercial; cite Reinisch & Galkin 2011
  (doi:10.5047/eps.2011.03.001); some stations require their own acknowledgement text. The SA clause
  says raw values are shared only with LGDC account holders; derived products are not restricted.
- `DIDBGetValues` is retired (404). The live endpoint is FastChar: `lgdc.uml.edu/fastchar/getbest?...`
  (text). **Returned 429 on the first probe**: rate limited, and the HUM LEAD's shared IP is the same
  lesson as the Open-Meteo quota.
- GAMBIT/IRTAM coefficients withhold the latest 3 days (real time is a commercial subscription), and
  using them would need IRI anyway.

## 5. Findings: go-tuimaps v0.2.0-rc.34 (no build or test was run)

**Works today:**
- a whole-globe Grid (`FitWorld`, no Bound), wrapping across the antimeridian;
- host `Type{Breaks}` (up to 20 breaks) with `Lines: true` gives labelled class-boundary contours;
- station Points labelled "SITE 12.3", as watchpost draws the EPA UV cities;
- `Keeps` stale handling, `During` spans for loops.

**Missing (all additive):**
1. **Fields stop at the shore** (D-32). `render.Input.FieldsOverWater` exists but has no public
   setter. A world MUF map would show land only. **The biggest gap.**
2. **No fill for a host type.** `classInk` returns 0 for preset 0, and the legend has no colours
   (`facts.go:186`). Needs a host colour ramp through `CheckRamp`, or `muf`/`fof2` presets.
   Borrowing the temperature preset works visually but mislabels the legend and Describe.
3. Contour labels have no unit; every level is one ink; labels are placed only on east–west crossings.
4. No day/night terminator, and no role token to draw one with (the maths is easy in the host or
   the library).
5. No colour-by-value role for station Points (only AQI and UV have one).
6. No line or time-series chart anywhere (go-tuimaps, go-studs, watchpost). eSSN needs a new go-studs
   component; braille or fractional blocks as `platform/render/spectrum.go` uses.

**Adding a requirement:** watchpost D-nnn → library D-102 → `## L-30` in `requirements.md` (note
that `## L-28` is used twice) → plan task L11.36 → the next rc.

**v0.2.0's own remaining release work:**
- L10.3–L10.12, including M6 (five consecutive green gate runs; recent runs alternate),
  M4 (the 1 h soak) and M1 (the motion UAT);
- OW-12, F-3;
- the v0.2.0 release checklist, not yet written.

## 6. Findings: watchpost

- D-8/D-28 forbid any frame wider than the selected location's region, so a world MUF map needs an
  amending ruling: an exception for propagation layers, or a separate unbound "Propagation" view.
- D-179 already sized the history store for "an ionosonde's MUF every 5–15 minutes".
- D-224/D-226 (record it, store it; a source with its own history needs a ruling). KC2G serves
  eSSN history back to 2020; the iongrid is 24 hours ahead with no history.
- D-185 (keyless first): KC2G is keyless; GIRO direct is rate limited.

## 7. Options

| Path | What | Effort | Risk |
|---|---|---|---|
| **A** | Consume KC2G's published outputs (iongrid.bin, essn.json, stations, planner.json) | small: a fetcher, a decoder, a Grid | an undocumented hobby API with no terms; ask KC2G for permission and a polling rate |
| **B** | GIRO FastChar + our own spatial GP over a climatology background | moderate–hard | GIRO NC/SA terms, 429s, still needs a background model |
| **C** | Full Go port (B + PyIRI-subset port + eSSN fit + per-station GP) | **large**: a separate project like go-tuimaps, a validation burden | unlicensed reference code, coefficient upkeep |

## 8. Recommendation

- **The feature is 0.19.0.** Path C is a project of its own: a Go IRI port with its scientific
  validation. Path A needs KC2G's permission, and asking it is on the HUM LEAD's clock, not ours.
  Neither belongs at the end of a release already in BUILD and UAT.
- **The library half can be v0.2.0 if the HUM LEAD wants to avoid a v0.3.0.** Items 1–3 (fields over
  water, a host colour ramp, unit labels) are small, additive and generic: waves and future marine
  layers benefit too. Cost: about 1–2 library batches, a new rc and watchpost's pin bump, and they
  restart M6's five-green count. Items 4–6 (terminator, value-coloured points, line chart) can wait:
  the terminator and value roles are additive in v0.3.0 or the host, and the chart is go-studs.
- **Before 0.19.0's DISCOVER:** email KC2G for permission and a polling rate; decide A (consume) vs C
  (port); a ruling against D-8/D-28 (global or propagation view); a ruling on storing eSSN and iongrid
  snapshots (D-226); GIRO and station acknowledgements in About (W21 chips: KC2G, GIRO).

## 9. 0.19.0 kickoff seeds (LEVEL-1 / SEV-0)

- **P1:**
  - the MUF(3000) and foF2 map overlays from iongrid (the hour selectable, 24 h ahead), station dots
    with values, a terminator;
  - eSSN/eSFI chip and time-series chart (the go-studs line chart).
- **P2:**
  - Broadcaster frequency planner (planner.json: tx = the station's transmitter locator,
    rx = a picked place);
  - band-quality summary.
- **P3 (optional):** own computation (path B/C) as a separate repo, if KC2G dependence is ruled out.
