---
title: "0.19.0 DISCOVER — wave 1 findings (desk)"
date: 2026-10-05
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "COMPLETE — all four wave-1 surveys recorded; one budget breach (GitHub API) reported to the HUM LEAD"
---

# Wave 1 findings

## Summary

- **The map library can't yet draw a global MUF map** a reader can read: fields stop at the coast, a field of the host's own kind gets no colour, labels and words carry no units, and there is no day/night line. Two of these were promises the library made and never built. All are now go-tuiMaps v0.3.0's (D-32 to D-36).
- **Outside the US, watchpost shows no map and no words**, and it knows no operator's HF band; "band" already names the console's notice strips. Hence the Propagation mode with its own bound (D-21) and the "My HF bands" picker (D-30).
- **No openly licensed, global, ionosonde-driven MUF map exists.** INGV's Europe map is CC BY 4.0, and NOAA's GloTEC is public domain but publishes no MUF. PS-G was amended to fit (D-19).
- **The method is published** end to end. NASA's PyIRI (MIT) is the reference to port. IRI's own Fortran may not be redistributed.
- **The history store** opens only with NDFD and applies one retention to every dataset (fixed in FR-6).
- **One budget breach** (about 35 GitHub API calls) and one compute estimate about ten times low are recorded below.

## How it was done

Four read-only surveys (D-18):
- sources and their terms, with `giro.uml.edu` added at the HUM LEAD's request;
- the building blocks' licences and the published method;
- watchpost's code;
- go-tuiMaps' gaps.

Every claim cites `file:line` or a source. A claim the agent spot-checked is marked **checked**.

## go-tuiMaps v0.2.0 (origin/main `d1c4d5e`)

All five intake gaps (HR-1 to HR-5) are real. Two of them are promises the library already wrote down and
never built, and the research missed one gap.

| # | Finding | Evidence |
|---|---|---|
| HR-1 | Fields stop at the shore: `render.Input.FieldsOverWater` is set only by an internal test. `Map.Render` (`map.go:431`) builds `Input` through `paint` and `draw`, neither of which sets it. `ImagesMaskedByWater` has no public setter either. **checked** | `internal/render/frame.go:92`, `:96`; `field.go:182`; `field_test.go:77` |
| HR-1 (new) | **A broken promise.** FR-12 and D-87 say "a host can flip either, per overlay"; the plan names `TestHostCanFlipEither`, and the build log claims the flip. Neither the public path nor the test exists. | `requirements.md:38`; `rulings-discover.md:83`; `implementation-plan.md:355`; `build-log.md:191` |
| HR-2 | A host type has no fill: `classInk` returns 0 for preset 0. The legend's "no ramp" code is `rampToken`, not `facts.go:186` as the research said. The legend **does** carry the unit (`facts.go:97`), which corrects the research. | `internal/render/field.go:88-108`; `facts.go:179`, `:189-207` |
| HR-2 (new) | **A broken promise.** FR-15 promised a ramp for a host's own type from the `low`, `middle` and `high` tokens. The tokens are exported, but they have no default colour and nothing reads them. | `requirements.md:42`; `constants.md:109`; `internal/colour/token.go:58-60`; `preset.go:258-290` |
| HR-2 (new) | A host-type grid without `Lines` draws **nothing** at truecolor and 256 colours, and nothing warns. No test renders a host-type grid. | `internal/render/field.go:146-163` |
| HR-3 | Contour labels have no unit, use one ink, and sit only on east-west crossings, every 24th dot row. MUF's contours run east-west, so they would mostly go unlabelled. `Grid.Marks` (8 a field) can carry "14MHz" today. | `internal/overlay/field.go:167-176`, `:37-41`; `render/field.go:310-323` |
| HR-4 | There is no terminator or night code. A terminator line could use an existing role today, but `track` means a storm track. Night shading is impossible: only alert roles fill areas. | `render/basemap.go:206-208`; `store.go:278` |
| HR-5 | Colour by value exists only through `AirQualityRole` and `UVRole`. Point overlays get no legend. OW-17 already owes a non-colour mark. | `overlays.go:218-240`; `facts.go:129-136` |
| Words (new) | A field's `Answer` has an empty `ValueUnit` except for temperature, so a MUF answer has no "MHz" and no band words. **checked** | `internal/describe/answer.go:138-141`; fix at `describe.go:185`, as wind does |
| Globe | `FitWorld` frames 84°N to 56°S, so the far south can fall off. **checked** No test renders a whole-globe grid; "works today" rests on reading the code. A 2°×2° globe is 16,200 cells against a cap of 1,048,576. | `internal/project/fit.go:158-162`; `overlay/field.go:13` |
| Bound | The library imposes no regional limit. `SetBound(Bound{})` lifts a map's bound, and a separate propagation Map can share caches (`SharedCaches`). Only watchpost's D-8/D-28 forbid the wider frame. | `view.go:224-269`, `:257-258`; `shared.go:52` |
| Safe ramps (new) | `SafeRamps` covers only the tokens from `AlertExtremeOutline` to `High`. The wind, wave, UV, AQI and QPF tokens come after `High`, so they keep host colours with `SafeRamps` on, and `Warnings` stops reporting them. New MUF tokens would inherit this. **checked** | `internal/colour/palette.go:72`, `:99-101`; `sixteen.go:35` |

**Already owed to v0.3.0:**
- OW-14 to OW-20 and OW-24 to OW-26;
- F-1 (the crowded root);
- F-2 / OW-6 (pluggable architecture).

None covers HR-1 to HR-5. OW-24 touches the same legend `Class` as HR-2, and OW-17 is the colour-only problem HR-5 would add to.

**What v0.3.0's brief must decide (for its own DISCOVER):**
- the water flip per overlay (FR-12's promise) or map-wide;
- MUF and foF2 presets, or the generic host ramp FR-15 promised;
- units on labels and answers;
- whether a terminator and night shading are the library's job.

## watchpost (`48616c69`; no Go file changed since the brief's `538f3dbb`)

| Area | Finding | Evidence |
|---|---|---|
| Layers | A layer registers itself (`registerMapLayer`) and reaches the window through `Config` funcs (`MapFeed`, `MapTemperature`, ...), because `modes/` cannot import `domains/`. Field fetches go box by box, and a region is split at the antimeridian, which a grid cannot cross. | `app/maplayers.go:157-203`; `modes/tty/dashboard.go:86-131`; `app/maptemp.go:637-682` |
| Bound | There are six fixed US region boxes. **A place outside every region gets no map and no description** (the outside branch returns before any words). **checked** A non-US ham gets nothing, so R-1.3's words path inherits the region gate as built. | `platform/geo/regions.go:30-52`; `modes/tty/map_pane.go:466-477` |
| Words | The description is alerts only (F-205). A layer cannot add a sentence today, only a note (`mapNotesNow`). M5b needs a new seam in `describeUpTo`, or in the notes, plus a `modalKey` input. | `modes/tty/map_describe.go:57-107`; `map_pane.go:581-587`; `memo.go:337-347` |
| Broadcaster | The console has no windows of its own: Observer windows are composited over it, and the Observer's map follows the Observer's selected place, not the tower (F-174's "not the Observer's"). The 44-row floor is "fixed chrome plus one readable card". The tower and service radius are already first-class (`config.Broadcaster`, `StationAreaMsg`). | `modes/tty/router.go:513-518`; `broadcaster.go:621-668`; `platform/config/broadcaster.go:17-47` |
| Bands | **watchpost knows no operator band or frequency.** The only MHz values are NOAA Weather Radio's relay transmitters. "band" already names the console's painted notice strips. | `domains/radio/stream/table.go:29-32`; `broadcaster.go:326-447` |
| Settings | Rows are constants whose order is the focus order. Each tab has a notice area (D-237); the Broadcaster tab has no notice case yet. | `modes/tty/setup_rows.go:59-277`; `setup_notices.go:113-159` |
| Credits | `creditGroups()` lists credits by hand. **No test ties a registered source to its credit.** | `app/credits.go:16-71`; `app/credits_test.go:29-67` |
| Egress | `mapSourceList()` builds MAP STATUS's disclosure; a new host needs an entry and a `Sent` value. httpx is the single client package, with a mandatory User-Agent and **no allowed-hosts list**. go-tuiMaps fetches with its own transport and User-Agent. | `app/maps.go:213-225`; `platform/httpx/httpx.go:58-74`; `app/maptiles.go:35-44` |
| History | The store takes a per-station series and a global grid per hour (no cell cap; its doc comments use MUF as the example). But it **opens only when an NDFD temperature source exists**, and **one retention from the Data tab overrides every dataset's own** (D-175, D-231). **checked** | `platform/history/history.go:55-125`, `:566-588`; `app/history.go:147-156`, `:496-505` |
| #12 | Memo keys and what the completeness guards cover: see below. | `modes/tty/memo.go:36-387`; `memo_completeness_test.go:37-245` |
| F-187/F-188 | `say` is found on PATH, not at `/usr/bin/say`. Its only kill is CommandContext's SIGKILL to one PID. No process group, `WaitDelay` or termination-signal handler exists. **checked** A watchpost killed outright leaves `say` and its temp dir behind. F-187's suggested sweep names the temp-dir pattern, not the process. `make test-say` has no `-timeout`. | `domains/radio/synth/voice.go:94-121`; `app/dump_unix.go:28`; `Makefile:102-103` |

**#12, memo keys:**
- Every guard checks the same property: a field that moves the frame must move the key. It perturbs only top-level scalars and pointers, one level deep, and never uint, slice or map fields.
- None asks the identity question #12 raises: can what sits at a position change while the position and the keyed identities stay the same?
- The keys holding positions are `bodyKey`, `modalKey`, `consoleKey` and `tempMemoKey`.
- The app-side keys (`tempKey`, `airGridKey`, `uvDayKey`, `frameKey`, the area memo, `feedKey`) carry identities only, and **have no completeness guard**.

**Minor:** `setup_rows.go:63-71` says the transmitter rows sit in DATA; they sit on the Broadcaster tab (`setup_tabs.go:50-51`).

## Sources and their terms (`giro.uml.edu` included at the HUM LEAD's request)

No request went to any kc2g.com address and no source file of `arodland/prop` was opened. The survey's
data requests stayed inside D-18's budget, logged below.

| Source | What | Terms | Commercial / redistribute | Method |
|---|---|---|---|---|
| **GIRO / LGDC** (DIDBase, FastChar) | Station values from about 70 Digisondes: foF2, MUF(3000), M(D), hmF2. Not maps. Keyless web form; FastChar returned 429 earlier | CC BY-NC-SA 4.0. NC: "Data are not used or shared for commercial purpose". SA: "Shared, modified, or annotated data retain their original license". **"Development of substantially derivative products based on the acquired GIRO data is not restricted."** Cite Reinisch & Galkin 2011, doi:10.5047/eps.2011.03.001. **checked** (`giro.uml.edu/didbase/RulesOfTheRoad.html`) | NC; SA on the data itself; substantially derivative products unrestricted | measurements |
| **GIRO IRTAM / GAMBIT** | Global real-time foF2 and hmF2 maps: GIF images, plus coefficients | Free tier: "open academic-use access", **the latest three days not available**. Real time is paid ($9,995 a year), and its clause F forbids sharing with third parties | No | published papers; coefficients plus IRI can rebuild a map; the assimilation code is not public |
| **Australia BoM SWS** | Global foF2 and global *vertical* MUF images every 10 minutes; regional maps | Bureau copyright: personal or in-organisation use; "must not supply it to any other person or use it for any commercial purpose" | No | inputs listed (including GIRO); method not published |
| **NOAA SWPC GloTEC** | **Global grid of NmF2 and hmF2** (2.5°×5°, every 10 minutes, keyless geojson). foF2 follows from NmF2 | NWS public domain (`weather.gov/disclaimer`) | **Yes** | published (Chou et al. 2023, doi:10.1029/2023SW003480): a Kalman filter on GNSS and COSMIC-2 over an IRI background; GNSS-driven, not ionosonde-driven; its accuracy at the F2 peak is unverified |
| **NOAA WAM-IPE** | A physics model's forecast: NmF2 and hmF2 in NetCDF; MUF in its plots | NWS public domain | Yes | physics forecast, not a nowcast; files not confirmed (NOMADS returned empty) |
| **INGV eSWua** | **MUF(3000)F2 nowcast map of Europe**, every 15 minutes, 0.5°, keyless JSON | CC BY 4.0, "even commercially"; cite doi:10.13127/ESWUA/HF, and acknowledge station owners (from the survey; the agent's own check was refused by the host) | **Yes** | described: stations into an IRI background, then kriging; code not public; **Europe only** |
| NICT, DLR IMPC, SANSA, HamQSL | Station values, TEC only, or widgets | all rights reserved, permission required, or GIRO's terms inherited | No | - |
| **PyIRI** | The IRI climatology in Python, global | MIT (GitHub licence API) | Yes | fully reproducible: a candidate open background |

**PS-G's claim (D-11) — returns as a ruling.**
- "The maps published today come with no licence or reuse terms, or for non-commercial use only, and none can be reproduced" is **refuted for a regional map**: INGV's Europe map is CC BY 4.0.
- It is **weakened for a global one**: NOAA GloTEC publishes public-domain global NmF2 and hmF2, from which foF2, and an estimated MUF(3000), can be derived.
- **What survives:** no global, ionosonde-driven MUF(3000) nowcast is published under terms that allow reuse, with a method that can be re-run end to end.

**For the name and path.** GIRO's NC/SA terms bind its data, but "substantially derivative products" are
unrestricted. A computed global field may qualify, and that needs a ruling. GloTEC (public domain) plus
PyIRI (MIT) would be a fully open global path, GNSS-driven.

## Reception reports (M1, M4 reference, OQ-1)

| Source | Access | Terms | Fit |
|---|---|---|---|
| **WSPR via wspr.live** | keyless ClickHouse SQL, 20 requests a minute; hourly band × grid aggregation in one query | "allowed to use … for your own research and projects, as long as the results are accessible free of charge for everyone. … not allowed … commercial" | the survey's primary choice: standard beacons, power and SNR reported |
| **Reverse Beacon Network** | daily CSV zips, archive from 2009, about 1 day late | "freely available for study and analysis"; asks that results be shared with the RBN community | cross-check |
| PSKReporter | last 6-24 h only, one query per 5 min; MQTT live feed | no stated terms | unsuitable without a long live capture |

**All three are one-sided evidence:**
- A spot proves a band was open; no spot proves nothing unless the area is densely monitored.
- Coverage is dense in Europe and North America, and sparse over the oceans and Africa.
- Ground wave, short skip and sporadic-E muddy short paths and 10 m.

## Data requests made (D-18 budget)

| Source | Requests | Bytes |
|---|---|---|
| NOAA SWPC (GloTEC indexes and one geojson) | 3 | about 3.0 MB |
| NOAA NOMADS (two directory listings, empty) | 2 | 0 |
| INGV (one map viewer page) | 1 | not measured |

No GIRO data, IRTAM, BoM API, wspr.live, RBN or PSKReporter data was requested.

**Unverified:**
- NOAA NCEI (HTTP 503) and NOMADS files;
- GloTEC's F2-peak accuracy;
- the IRI licence text;
- whether INGV takes the shared stations' data from their owners or through LGDC;
- ESA SWE, UKSSDC, VOACAP Online.

## Building blocks: licences

| Block | Licence | Port under MIT? | Ship its data under MIT? |
|---|---|---|---|
| **IRI-2020** (Fortran, coefficient and index files) | custom: "to use, copy, and modify" — **no right to distribute** **checked** (`irimodel.org/IRI-2020/00_iri-License.txt`) | **No** (a reimplementation from the published formulas is another matter) | No |
| **PyIRI** (NASA/NRL, Forsythe et al. 2024) | **MIT** **checked** (GitHub metadata); computes global foF2 and M(3000)F2 for every point and hour as a few matrix products | **Yes**, keeping the notice | **Probably**, with a caveat: its CCIR/URSI tables come from CCIR/ITU and URSI, and PyIRI states no provenance for them. Its own spherical-harmonic refits are the most defensible to ship |
| **PyIRTAM** | MIT; fetches GIRO's IRTAM coefficients at run time | Yes | No: the IRTAM coefficients are GIRO's (free tier three days late, D-19's finding) |
| **ITU-R P.533-14, P.1239-4** | ITU copyright; recommendations are free to download; non-commercial reuse of content | The method (formulas) can be implemented; the text and tables cannot be copied | **No** for ITU-sourced tables |
| **ITURHFProp** | ITU dedication: "free from any copyright assertions" for implementers; no LICENSE file | Likely yes | Its data files are not in the repository and fall under ITU terms |
| **NeQuick G** | EUPL-1.2 (copyleft) | **No** | No |
| Go implementations of IRI, NeQuick or P.533 | none found | - | - |

**Consequence for D-2 (MIT, "subject to" the licence check):**
- Port PyIRI's method and code (MIT), never IRI's Fortran.
- Ship coefficient data only after a provenance ruling, or fetch it at run time.
- Copy no ITU tables.

## The method, from published sources (G-R8)

The pipeline is published end to end, without `arodland/prop`.

| Step | Primary citations |
|---|---|
| GIRO and autoscaling confidence | Reinisch & Galkin 2011 (doi:10.5047/eps.2011.03.001; required); Pignalberi et al. 2018 (filtering bad autoscaling, doi:10.1186/s40623-018-0952-z) |
| Background | Forsythe et al. 2024, PyIRI (doi:10.1029/2023SW003739); ITU-R P.1239-4; Jones & Gallet 1965; Bilitza et al. 2022 (doi:10.1029/2022RG000792) |
| Effective sunspot number | **Secan & Wilkinson 1997** (doi:10.1029/97RS01350); Brown, Bilitza & Yiğit 2017 (doi:10.1016/j.jastp.2017.08.022) |
| Whole approach, closest prior art | **Galkin et al. 2012**, IRTAM (doi:10.1029/2011RS004952); Galkin et al. 2020, NECTAR (doi:10.1029/2020SW002463); **Pietrella et al. 2026**, effective R12 from ionosondes into IRI, MUF maps every 15 min (doi:10.1016/j.asr.2026.09.053) |
| Assimilation on the sphere | Gneiting 2013 (valid kernels on spheres, doi:10.3150/12-BEJSP06); Jeong, Jun & Genton 2017 (doi:10.1214/17-STS620); kriging in modified-dip coordinates, Liu et al. 2023 (doi:10.3390/atmos14091399) |
| MUF from foF2 and M(3000)F2 | ITU-R P.533-14 §3.4-3.5; Bradley & Dudeney 1973; Dudeney 1983 |

**Traps the papers name:**
- The ITU coefficients use R12 on the version-1 sunspot scale (k = 0.6), so SILSO's version 2 must be scaled.
- foF2 is capped at R12 = 160.
- The geomagnetic field is taken at epoch 1960, 300 km.
- The P.533 equations were rebuilt from a garbled PDF extraction and must be checked against the PDF before coding.

**Validation practice (G-M3):**
- Leave-one-station-out scoring, with error reported against distance to the nearest station. IRTAM improves IRI only near assimilated stations (Pignalberi et al. 2021, doi:10.3390/atmos12081003).
- Published foF2 errors:
  - about 0.2 MHz (night) and 0.5 MHz (day) at a held-out site 685 km away (McNamara et al. 2013);
  - 0.4-2.0 MHz in general (Gulyaeva et al. 2026);
  - IRI's MUF(3000) off by 1-15% (Seba & Poedts 2024).
- A starting target, to be checked against the full texts: about 0.5 MHz RMS near stations, 1-2 MHz far from them, IRI climatology as the baseline to beat.

## Compute (G-G1)

- The peak maps need about 1,400 multiply-adds per grid point and hour (foF2 988 coefficients, M(3000)F2 441), not IRI's full height profile.
- A 2° globe is 16,471 points; 1° is 65,341.
- **Estimate, not measured:**
  - a whole day at 1° is about 2.2×10⁹ multiply-adds (65,341 points × 24 hours × about 1,429); **corrected** from 2.4×10⁸, about ten times low (red team DQ-F13). The earlier "under a second on one core" rested on the wrong figure and is withdrawn; the cost is measured in PLAN's dry run (G-G1);
  - under 50 MB at 2° in float32;
  - a spatial fit over 40-100 stations is trivial (n³ ≈ 10⁶).
- Suggested default: 2° × 24 h, 1° as an option, recomputed only on new data (15 minutes).

## Budget breach (D-18)

The licences survey made **about 35 requests to the GitHub API** (repository searches, listings, README,
LICENSE and CITATION reads, and parts of PyIRI's and PyIRTAM's MIT sources), against D-18's limit of 5
per source.
- No coefficient or data file was downloaded.
- No `arodland/prop` file was opened.
- No kc2g.com address was requested.

Within budget:
- OpenAlex 5;
- Crossref 5;
- Semantic Scholar 2 (both 429);
- two ITU recommendation PDFs, as documents.

Reported to the HUM LEAD with these findings.

## Addendum, 2026-10-06: D-RAP for disturbance absorption (D-47)

A desk check of documentation only, with no data request (`spaceweather.gov/products/d-region-absorption-predictions-d-rap`).

| | |
|---|---|
| What | "the highest frequency affected by absorption of 1 dB due to either solar X-ray flux or SEP events or a combination of both"; global, plus polar maps at 10 dB |
| Grid | tabular values every 5° of latitude and 15° of longitude |
| Cadence | "update continuously, driven by one-minute GOES X-ray flux data and by five-minute GOES proton flux data" |
| Access | keyless ASCII, `services.swpc.noaa.gov/text/drap_global_frequencies.txt`; archive at NCEI |
| Terms | none stated on the page beyond NOAA/NWS's (`weather.gov/disclaimer`): public domain, no endorsement |

D-RAP covers **disturbance** absorption only (flares, solar energetic particles). Regular daytime absorption
is not in it; it is computed from the sun's position (D-47, go-ionomaps R-2.4). The fetch falls under D-39
(NOAA: at most 6 an hour), and the file's size and format are measured in PLAN's dry run.
