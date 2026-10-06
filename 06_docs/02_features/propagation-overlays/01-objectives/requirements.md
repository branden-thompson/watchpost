---
title: "0.19.0 — Propagation overlays — REQUIREMENTS"
date: 2026-10-06
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "DRAFT — for approval at the DISCOVER gate. On any conflict with the brief or the problem statement, this file wins once approved, and every change is a row in 02-analysis/rulings.md. Revised after the DISCOVER-exit red team, round 1 (08-reports/red-team-discover.md)."
---

# Requirements

Each functional requirement traces to a brief requirement (R-n) and to the ruling that makes it necessary.
**What, never how**: the how is PLAN's, and PLAN carries no code (0.18.0 D-13).

Each row's **Instrument** names what will hold it at BUILD exit: a test by its planned name, a gate, or a
HUM-graded metric. A planned name is a commitment PLAN may rename, never drop. A row whose instrument is a
document is marked **process commitment**: a document cannot fail, so those rows are kept by the phase
reports, not by a test.

**Order (D-4):**
- **O1** is MUF: the Propagation mode, the Broadcaster's map and readout, sources, storage, the words that ship with the layer (D-9). FR-6.5 comes first in BUILD (D-55).
- **O2** is 0.18.0's accessibility carry-over, held by a SHIP gate (D-63).
- **O3** is #12 and the `say` fixes (F-187, F-188), with O3's gate fixes allowed earlier where a flaky gate blocks O1 (D-15's reason).

## What the listener sees (D-70)

The user-facing rulings, in plain words. The rows below are normative; this section is the summary.

- **Observer.**
  - The map window gains a **Propagation mode** beside Radar and Forecast (D-21). It shows two layers, MUF(3000) and foF2, in MHz, over the whole world and across the sea, with a day/night line (D-24, D-32 to D-35).
  - It opens for any place, even outside the US regions; weather maps stay within their region (D-21).
  - It answers from the selected place to the continents by default, to any place found by city, ZIP or coordinates (found places are not kept), and to wherever the map is centred (D-25 to D-27, D-37, D-61).
  - Each HF band is called open, above the upper limit, absorbed by daytime or a disturbance, or no data (D-47).
  - Hours can be stepped up to a day ahead, labelled as forecasts (D-50).
  - Everything is also said in words without the picture, and the selected place is always said (D-9, D-64).
- **Broadcaster.**
  - The console gets its own regional map (tower, service radius, line-up) and a one-line propagation readout for the operator's picked HF bands, 80 m and 40 m by default; the readout is on by default (D-28 to D-30, D-60).
  - When a band is about to close, or has closed, a notice appears and stays until acknowledged. It is never put on the broadcast audio (D-48, D-59).
  - A key opens the Observer's Propagation mode from the tower (D-57). The map is described in words (D-58).
- **Both.**
  - The first time propagation is shown, a window says watchpost transmits nothing, the maps are reference, and anyone transmitting on HF must follow the law where they are. Closing it records it was seen; it blocks nothing (D-45, D-46).
  - Sources are credited, and GIRO's non-commercial terms are stated (FR-4).

## FR-1 — The Propagation mode (R-1; D-21, D-24 to D-27, D-37, D-50, D-61, D-64)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-1.1 *(O1)* | The map window has a **Propagation mode** beside Radar and Forecast, entered and left by a keymap action listed in Help and overridable in `[keys]`; **the current mode is named in words in every path** (picture, "Instead of the map", `--ascii`), not by a two-state chip | D-21; AX-A | `TestThePropagationModeIsAKeymapAction`, `TestTheMapKeysAreActionsHelpShows` (extended), `TestTheModeIsNamedInEveryPath` |
| FR-1.2 *(O1)* | The Propagation mode has **its own world-capable bound**; Radar and Forecast keep 0.18.0's region bound unchanged; no weather frame is ever wider than its region | D-21; 0.18.0 D-8, D-28 | `TestNoFrameIsWiderThanTheRegion` (extended: every weather-mode frame), `TestEachModeKeepsItsOwnBound` |
| FR-1.3 *(O1)* | The Propagation mode **opens for any place**, including one outside every US region; the outside-region gate gives way to this mode only, and its message names the Propagation key as the way forward | D-21; AX-M | `TestAPlaceOutsideEveryRegionOpensThePropagationMode`, `TestAPlaceInNoRegionIsAStatedState` (kept, for weather modes), `TestTheOutsideMessageNamesThePropagationKey` |
| FR-1.4 *(O1)* | Weather layers are off in the Propagation mode; propagation layers are off in the weather modes | D-21 | `TestEachModeDrawsOnlyItsLayers` |
| FR-1.5 *(O1)* | Two layers, **MUF(3000)** and **foF2**, each with its valid time and age, a legend in MHz, contour labels in MHz, and a day/night terminator marked without relying on colour | D-24; D-32 to D-35; AX-E | `TestBothPropagationLayersDrawWithTheirUnits`, `TestTheTerminatorIsDrawnForTheFieldsTime`; the library's v0.3.0 rows (L-3, L-4.4) |
| FR-1.6 *(O1)* | The fields **continue over the sea** | HR-1 (D-32) | `TestAPropagationFieldCrossesTheSea` (on the library's `TestHostCanFlipEither`) |
| FR-1.7 *(O1)* | Pan and zoom work as in the Observer's map; the view rehydrates with a loading indicator | D-25; 0.18.0 FR-1.10 | `TestThePropagationModePansAndZooms`, `TestTheWindowSaysWhatThePictureIs` (extended) |
| FR-1.8 *(O1)* | **Paths start at the selected place** in the Observer (at the tower when opened from the Broadcaster, FR-2.11); the origin is always marked | D-25, D-26, D-57 | `TestObserverPathsStartAtTheSelectedPlace` |
| FR-1.9 *(O1)* | **Continental reference points are the default targets**; the answer is given for each, by UTC hour and HF band, between both limits | D-25; D-47 | `TestTheContinentsAreTheDefaultTargets`, M4 |
| FR-1.10 *(O1)* | A location can be **found by city, ZIP or coordinates** (D-61): an offline world-cities list answers first (GeoNames cities15000, CC BY 4.0, credited); coordinates parse locally (finite, ±90, wrapped); ZIPs use the existing US index; only when nothing local matches does the online geocoder answer, non-US allowed in this mode, uncached and disclosed in MAP STATUS; the offline index loads on first use, with bounded memory; the found place is marked and answered and **never kept**; focus returns to the map and the found place is said | D-25, D-27, D-61; IS-4, AX-B | `TestAFoundPlaceIsAnsweredAndNotKept`, `TestALocalMatchSendsNothing`, `TestCoordinatesNeverGoOnline`, `TestTheOnlineFallbackIsUncached`, `TestFocusReturnsAfterAFind`; the index's load cost in the dry run |
| FR-1.11 *(O1)* | The answer for **the point at the map's centre** as the listener pans: MUF, foF2, and which HF bands open to it from the origin, on screen and in words | D-37 | `TestTheCentreIsAnsweredAsTheMapPans`, `TestTheCentreAnswerIsSaid` |
| FR-1.12 *(O1)* | The mode is a member of every closed-set window test (reachability at 80×24, memo completeness, margins, glyph survey, keys, leaving) | 0.18.0 FR-1.5 | the closed-set tests, each enumerating the mode |
| FR-1.13 *(O1)* | **Hours ahead:** the listener steps the fields up to 24 h ahead, one hour per key press (no automatic play), each forecast hour labelled a forecast with the time it was made, on screen and in words | D-50; AX-J | `TestTheHoursStepOnAKey`, `TestAForecastHourSaysItIsAForecast` |
| FR-1.14 *(O1)* | **Every control in the mode is a keymap action** Help lists: choosing MUF or foF2, stepping the hour, moving between targets, finding a place; Help fits its window with them | AX-B | `TestEveryPropagationControlIsAnAction`, `TestHelpFitsWithThePropagationKeys` |

## FR-2 — The Broadcaster's map and readout (R-2; D-28 to D-30, D-57 to D-60)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-2.1 *(O1)* | The Broadcaster gets **its own map** (F-174): the tower marked, the service-radius ring, the line-up's places; local or state zoom by default; pan and zoom as the Observer's; inside the US regions (0.18.0 D-8/D-28); its keys are console actions Help lists, clashing with no console key (`+`, `=`, `-`, `O` are taken) | D-28; AX-H | `TestTheBroadcasterMapShowsTheTowerItsRadiusAndTheLineUp`, `TestNoFrameIsWiderThanTheRegion` (extended to it), `TestTheConsoleKeysDoNotClash` |
| FR-2.2 *(O1)* | The Broadcaster's map follows the **tower**, never the Observer's selected place | D-28; `modes/tty/map_pane.go:948-963` | `TestTheBroadcasterMapFollowsTheTower` |
| FR-2.3 *(O1)* | A **propagation readout** on the console: foF2 at the tower, the near-vertical status of each amateur band 160 m to 10 m in words (open, above the upper limit, absorbed, no data: each naming the limit that decided it), and the data's age | D-28, D-24, D-47; AX-F | `TestTheReadoutSaysFoF2AtTheTowerAndEachBandsStatus`, M1, M2 |
| FR-2.4 *(O1)* | A **"Propagation readout"** Broadcaster-tab Setting, **on by default**; while on, the console fetches on the data's cadence (hourly; D-RAP every 10 minutes) with no map open; its cost, including downloads a day, is stated in the tab's notice area; MAP STATUS lists the hosts | D-29, D-48 | `TestTheReadoutSettingIsOnByDefault`, `TestTheReadoutStatesItsCost`, `TestMapStatusListsEveryPropagationHost` |
| FR-2.5 *(O1)* | A **"My HF bands"** Broadcaster-tab picker: a multi-pick of the amateur bands 160 m to 10 m from the existing picker component, defaulting to 80 m and 40 m, persisted in `config.Broadcaster` | D-30 | `TestMyHFBandsDefaultsTo80And40`, `TestTheBandsPickerIsTheSharedPicker` |
| FR-2.6 *(O1)* | **The closure notice:** for a picked band, a closure the model predicts is announced ahead with its expected time (M3a); a disturbance closure from D-RAP is said within 15 minutes (M3b); a closure seen in the measured field within 90 minutes (M3c); none picked silences it | D-30, D-48; M3 | `TestAClosedPickedBandRaisesTheNotice`, M3 |
| FR-2.7 *(O1)* | The readout is one compact line within the console's measured floor (44 rows, "fixed chrome plus one readable card"), expanded on a key, scrolling if needed; the floor does not rise | D-60; `modes/tty/broadcaster.go:621-647` | `TestEverySizeShowsAStatedStateAndNeverOverflows` (Broadcaster) |
| FR-2.8 *(O1)* | UI wording never uses the bare word "band" for HF, which already names the console's notice strips | D-30 | `TestTheConsoleSaysHFBandsNotBand` |
| FR-2.9 *(O1)* | **The closure notice is never on the broadcast audio:** a notice band on screen until acknowledged, logged in the Status window with its time | D-59; AX-I | `TestTheClosureNoticeNeverReachesTheBroadcastAudio`, `TestTheNoticeStaysUntilAcknowledged`, `TestTheNoticeIsLoggedInStatus` |
| FR-2.10 *(O1)* | **The Broadcaster map in words:** the tower, the service radius, which line-up places fall inside it, and the readout; "Instead of the map" applies to both modes | D-58; AX-G | `TestTheBroadcasterMapSaysItsGeography`, `TestInsteadOfTheMapSpansBothModes` |
| FR-2.11 *(O1)* | **A console key opens the Propagation mode** over the console, paths starting at the tower, the tower always marked; the key is a console action Help lists | D-57, D-25, D-26; AX-H | `TestTheConsoleOpensThePropagationModeFromTheTower` |

## FR-3 — In words (R-1.3; D-9)

"In words" and "said" mean **screen text a screen reader reads**; nothing propagation says ever goes to the
broadcast audio (D-59).

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-3.1 *(O1)* | With no picture (`--ascii`, "Instead of the map"), the Propagation mode says in words what MUF and foF2 are at the origin (the selected place, or the tower), always, wherever the view is (D-64), then the centre's answer (D-37), which bands open to each target, and the data's age | D-9, D-64; M5b | `TestThePropagationWordsAnswerWithoutThePicture`, M5b (UAT) |
| FR-3.2 *(O1)* | Words carry units and band names written for speech: "megahertz", "metres", each layer's full name once ("maximum usable frequency", "F2 critical frequency"), never a bare number | D-34; AX-K | `TestPropagationWordsCarryUnitsForSpeech` |
| FR-3.3 *(O1)* | The description gains a seam a layer adds its sentence to, reached **before** the alert-only early returns (`modes/tty/map_describe.go:65-70`); the alert sentences are unchanged | `wave1-findings.md` (Words); AX-C | `TestALayerAddsItsSentence` (three cases: alerts off, alerts never asked, the selected place out of view), `TestTheMemoKeyCoversEverythingTheFrameShows` (extended) |
| FR-3.4 *(O1)* | The Broadcaster's readout and notice are screen text a screen reader reads | D-28, D-59 | `TestTheReadoutIsScreenText` |
| FR-3.5 *(O1)* | Without the picture, the mode's status (loading, offline, a coarse picture, the data's age) is still said | AX-D (the mode's share of 0.18.0's AX-5) | `TestTheStatusIsSaidWithoutThePicture` |

## FR-4 — Sources, terms, credit and egress (R-3; D-2, D-22, D-29, D-39, D-41, D-43, D-61, D-69)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-4.1 *(O1)* | Every propagation source is **credited in About** under its own terms: GIRO (CC BY-NC-SA 4.0; Reinisch & Galkin 2011; each station's provider; **non-commercial access** stated); NOAA SWPC GloTEC and D-RAP (public domain; no endorsement); PyIRI (MIT; Forsythe et al. 2024; the NRL refits and their CCIR/URSI lineage); GeoNames (CC BY 4.0). WSPR (wspr.live, WSPRnet) and RBN are credited in the reports that use them | R-3.1; G-M1; D-22, D-41, D-43, D-61, D-69; IS-5 | `TestEverySourceIsCreditedOnce` (extended), `TestEveryRegisteredSourceHasACredit` (new: ties sources to credits) |
| FR-4.2 *(O1)* | The Observer's Propagation mode fetches **nothing before it is opened**; the Broadcaster's readout is the only scheduled fetch (D-29); the recorder fetches nothing on its own (D-62) | R-3.2; 0.18.0 D-25 | `TestThePropagationModeFetchesNothingUntilOpened` |
| FR-4.3 *(O1)* | Every new host is listed in MAP STATUS with what it learns: GIRO and NOAA learn the IP (GIRO's replies echo it back); the online geocoder learns the typed text; no request names the station's place | R-3.3; 0.18.0 D-31; IS-2, IS-4 | `TestMapStatusListsEveryPropagationHost`, the egress table in PLAN |
| FR-4.4 *(O1)* | Requests go through watchpost's own client with its User-Agent; go-ionomaps takes the host's fetcher | GR-5 | `TestPropagationFetchesUseTheHostsClient` |
| FR-4.5 *(O1)* | **The throttle (D-39, D-48, D-51) is go-ionomaps'** (its R-5.3 to R-5.5); watchpost holds it by handing that library a fetcher that **never retries a 429** and passes every status and header through (`platform/httpx/httpx.go:926-928` retries a 429 by default; the app's clients pace at 30 a second, `app/app.go:132`) | D-39; IS-1, CQ-T1, PF-F1, PF-F10 | `TestThePropagationFetcherNeverRetriesA429` (the real client against a counting 429 server) |
| FR-4.6 *(O1)* | **Nothing raw is kept:** GIRO's replies (which carry the requester's IP) never enter the HTTP cache, the diagnostics or an error; GloTEC and D-RAP are fetched uncached, parsed and dropped (2.5 MB grids would otherwise sit in the cache's large tier for its 24-hour grace) | IS-2; PF-F6; D-31, D-41 | `TestNoRawSourceDataIsStored` (scans the history store, the HTTP cache directory and the diagnostics) |
| FR-4.7 *(O1)* | The propagation client refuses plain http and private addresses, with a body cap sized to each source | IS-6 | `TestThePropagationClientIsHardened` |
| FR-4.8 *(O1)* | **One go-ionomaps object per process**, shared by the readout, the Propagation mode and the recorder; across instances sharing a store, the hour's update is claimed (the store's `Claim`), so one machine makes one update an hour; the update runs off the UI goroutine | PF-F7, PF-F8 | `TestOneUpdateAnHourAcrossInstances`, `TestTheUpdateNeverRunsOnTheUIGoroutine` |

## FR-5 — Honesty (R-5)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-5.1 *(O1)* | The data's age is always said, on screen and in words | R-5.1 | `TestTheFieldsAgeIsAlwaysSaid` |
| FR-5.2 *(O1)* | Stale, partial or failed data is said as such, never drawn as current; a failure has a path to resolve or goes to the diagnostics (0.18.0 D-124); the fallback background (climatology) is named when it is in use | R-5.1; G-M4; D-40 | `TestAStaleFieldIsSaidStale`, `TestAFailedFetchIsSaidWithItsPath`, `TestTheFallbackIsNamed` |
| FR-5.3 *(O1)* | A forecast hour is never shown as a measurement | D-50 | `TestAForecastHourSaysItIsAForecast` |

## FR-6 — Storage (D-31, D-55, D-62)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-6.1 *(O1)* | The computed global MUF(3000) and foF2 grids, and the readout's values at the tower, are recorded hourly **from what propagation fetches anyway**; nothing is fetched for recording alone | D-31, D-62 | `TestPropagationGridsAreRecordedHourly`, `TestTheTowerReadingsAreRecorded`, `TestTheRecorderNeverFetchesAlone` |
| FR-6.2 *(O1)* | The history store opens whenever any recorded source is on, not only NDFD | D-31; `app/history.go:147-156` | `TestTheStoreOpensWithoutNDFD` |
| FR-6.3 *(O1)* | Each dataset may keep its own retention; the Data tab's value is the default; a zero or malformed retention deletes nothing | D-31; `app/history.go:496-505` | `TestADatasetKeepsItsOwnRetention`, `TestABadRetentionDeletesNothing` |
| FR-6.4 *(O1)* | Raw source data (GIRO readings) is never stored | D-31; GIRO terms | `TestNoRawSourceDataIsStored` |
| FR-6.5 *(O1, first in BUILD)* | **The roll-up never erases a year (#27):** a failed read of the year's roll-up refuses the write; roll-ups are split below the read cap (for example by month); each dataset has a byte bound beside its time retention; values are stored compactly, not one allocation each (12.5 MB a day held at 2° today) | D-55; PF-F4, PF-F5 | `TestAFailedYearReadNeverRewritesTheYear`, `TestARollUpNeverPassesTheReadCap`, a 90-day global-grid soak |
| FR-6.6 *(O1)* | **Backfill on request:** past hours are recomputed from archived inputs (GIRO's past readings; GloTEC's archive where it exists, else the climatology background) and recorded, under the throttle; the control is ruled in PLAN | D-62 | `TestABackfillFillsTheGap`, `TestABackfillHoldsTheThrottle` |
| FR-6.7 *(O1)* | A dataset's record carries its terms (for propagation: GIRO's non-commercial condition), so any later reader inherits them | IS-5 | `TestADatasetCarriesItsTerms` |

## FR-7 — Accessibility carry-over (R-6; D-4, D-63)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-7.1 *(O2)* | F-191 (FR-1.8): the window says its picture is braille and names the remedy | 0.18.0 D-244 | as F-191 |
| FR-7.2 *(O2)* | F-192, F-193, F-194, F-180: radar's motion Setting with reduce motion; the loop's rates and hold; the playback speed Setting | 0.18.0 D-245, D-255 | as each row |
| FR-7.3 *(O2)* | F-196, F-197, F-198: the Monochrome theme hint; the colour-literal guard; the `CheckRamp` matrix | 0.18.0 D-247 | as each row |
| FR-7.4 *(O2)* | F-205: the cursor placed for a screen reader; the description beyond alerts; the edge warning without a picture; the state words; the Maps tab; the outline key; "With the map" | 0.18.0 D-273 | as F-205's findings (AX-3 to AX-9) |
| FR-7.5 | **SHIP is blocked while any FR-7 row lacks its passing test**; dropping a row takes a ruling | D-63 | a SHIP-gate check over FR-7's rows |

## FR-8 — Memo keys carry identity (R-7; #12, D-14)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-8.1 *(O3)* | Every memo key is audited for identity against position; each position field either becomes an identity or carries its reason in a comment | #12 | the audit table in PLAN; `TestEveryPositionFieldHasAReason` |
| FR-8.2 *(O3)* | The app-side keys (`tempKey`, `airGridKey`, `uvDayKey`, `frameKey`, the area memo, `feedKey`) gain a completeness guard | `wave1-findings.md` (#12) | `TestTheAppMemoKeysCoverTheirInputs` |

## FR-9 — The gates stay reliable (R-8; D-15)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-9.1 *(O3)* | A normal exit, or an interrupt or termination signal, leaves no `say` child behind | F-187 | `TestNoSayOutlivesAnOrderlyExit` |
| FR-9.2 *(O3)* | A `say` left by a watchpost killed outright is found and ended at the next start, by process, not by its temp directory's name | F-187; `wave1-findings.md` | `TestAnOrphanedSayIsEndedAtStart` |
| FR-9.3 *(O3)* | `make test-say` cannot hang the gate: it has a bound, and a hang is diagnosed (the hung process sampled) rather than retried blind | F-188 | the gate's own bound; `06_docs/required-gates.txt` |

## FR-10 — Instruments (R-9; REFLECT L1-L8)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-10.1 | Each metric's instrument runs once before PLAN exit, its number or gap in the PLAN report | L1 | **process commitment**: the PLAN report |
| FR-10.2 | A pre-push hook runs the docs-reading tests | L5 | the hook, and a watched failure |
| FR-10.3 | The code-quality brief asks for any cross-layer ruling without an end-to-end test; the scripted-PTY journey (0.18.0 D-271) gains the Propagation mode and the Broadcaster's map | L2 | `06_docs/red-team-brief.md`; the journey |
| FR-10.4 | Ruling questions cite `file:line` for every claim about the code | L3 | a docs-reading test: every ruling row naming code carries a `file:line` |
| FR-10.5 | A batch carrying UAT verdicts lists their D-rows | L8 | **process commitment**: the build log |
| FR-10.6 | **PLAN's dry run measures** (D-49, the red team's limits): MUF(3000) leave-one-station-out for B on D; the US stations' scores and signed bias by distance; the NRL refits (D-43) with the F10.7 rule written down; GloTEC's availability and archive reach (D-62); the forecast hours' accuracy (D-50); D-RAP's file; WSPR's density at Broadcaster distances (under 200 km), before M1's scenarios are built; the offline cities index's load cost (D-61); G1 over at least 48 hours with downloads a day; every target D-49 left to it, each ruled | D-49; CQ-E2, DQ-F20, PF-F9, PM-K3, BQ-F4 | **process commitment**: the PLAN report |

## FR-11 — The acknowledgement (D-45, D-46)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-11.1 *(O1)* | The first time propagation data is shown (the Propagation mode, or the Broadcaster's map or readout), a window says that watchpost does not transmit, that MUF and foF2 are provided as reference, and that anyone who transmits on HF is responsible for following all applicable laws where they are | D-45, D-46 | `TestTheAcknowledgementShowsBeforeTheFirstPropagationView` |
| FR-11.2 *(O1)* | Closing it, with any of its keys, records that it was seen; it never gates the data or the fetch | D-46 | `TestClosingTheAcknowledgementRecordsItSeen`, `TestTheAcknowledgementGatesNothing` |
| FR-11.3 *(O1)* | The record keeps the wording's version; a changed wording is shown once more | A-9 | `TestAChangedAcknowledgementShowsAgain` |
| FR-11.4 *(O1)* | The window is drawn as text, readable without the picture, and named in Help | D-9 | `TestTheAcknowledgementReadsWithoutThePicture` |

# Non-functional requirements

| # | Requirement | Source | Instrument |
|---|---|---|---|
| NFR-1 | **Cost (G1):** watchpost's added CPU, memory and downloads a day with propagation on, measured over at least 48 hours, stay within G1's target, set from PLAN's dry run. The library's own cost is go-ionomaps' R-8.1 (G-G1) | issue #25; D-49 | G1's measurement |
| NFR-3 | **Live requests follow the feature's throttle** in every phase (D-39); never Open-Meteo for research; no kc2g.com request until asked | D-39 | the request log, with headers, kept beside the findings (go-ionomaps `02-analysis/evidence/`) |
| NFR-4 | **P10 clean** at every phase exit (`make p10`) | standing | `make p10` |
| NFR-5 | **Both CI platforms green** before merge, the mode named (quick or full) | standing | the hosted run |

(NFR-2, the library's cost, was folded into NFR-1 by citing go-ionomaps' R-8.1; CQ-N3.)

# Risk register

| # | Risk | Likelihood | Impact | Mitigation | Owner |
|---|---|---|---|---|---|
| RK-1 | The reference implementation has no licence; a reimplementation drifts toward its code | low | high | D-53: `arodland/prop` was read through the GitHub API in 0.18.0's research and is never opened again; no code copied; go-ionomaps is written from papers and PyIRI (MIT); every exported function cites its source; a provenance table in PLAN | agent |
| RK-2 | **GIRO load across the fleet:** each Broadcaster polls GIRO about 38 requests an hour by default (D-29); many installs, or several behind one IP, share GIRO's bucket; GIRO's non-commercial terms bind every copy | high | high | D-39's throttle per process, the hour claimed across instances (FR-4.8), the seed and rotation (D-51), the fallback background when GIRO refuses; F-208 if GIRO blocks the User-Agent | HUM LEAD (D-29, D-41) |
| RK-3 | **The chosen field is not accurate enough.** On 2026-10-05, B on D's foF2 RMS was 1.00 MHz overall and 0.37 MHz within 500 km of a station (seven European stations); the US stations are all more than 1000 km apart (1.06-1.28 MHz there); MUF(3000) is barely better than GloTEC's (3.97 against 4.02 MHz) because M(3000)F2 dominates | medium | high | go-ionomaps R-9.2 assimilates M(3000)F2 too; the dry run scores MUF and the US (FR-10.6); G-M3 against held-out ionosondes; M1/M4 against climatology (D-23) | agent |
| RK-4 | Compute or memory too heavy for a terminal app | low | medium | G1 over 48 hours; G-G1; 2° default; recompute on new data only; compact storage (FR-6.5); the update off the UI goroutine (FR-4.8) | agent |
| RK-5 | Three paired releases at SEV-0 cost time | high | medium | D-13 (minor rulings batched); one record | HUM LEAD, agent |
| RK-6 | Accessibility last again (D-4 order) | high | high | M5b and FR-3 ship words with the layer; the SHIP gate on FR-7 (D-63) | HUM LEAD |
| RK-7 | Two bounds in one window let a weather frame go global | low | high | FR-1.2's tests over every frame | agent |
| RK-8 | The readout fetches for stations that never use HF (D-29 on by default) | medium | low | the Setting turns it off; the cost is stated | HUM LEAD (D-29) |
| RK-9 | The history-store changes (FR-6.2, FR-6.3, FR-6.5) break 0.18.0's datasets | medium | medium | existing history tests; a retention test per dataset; the 90-day soak | agent |
| RK-10 | The library's label move (D-34) shifts every existing layer's labels | high | low | specimens redrawn in the same batch (`diagrams-move-with-code`) | agent |
| RK-11 | **False "open":** both backgrounds read foF2 high on 2026-10-05 (GloTEC +0.4 to +0.5 MHz, climatology +0.7; B on D's bias not yet measured), so the model errs toward the bad outcome PS-1 names, a band called open that is closed | medium | high | the dry run reports signed bias by distance (FR-10.6); a bias correction is a PLAN question; every answer states its limit and age | agent |
| RK-12 | **M1 may be unmeasurable at Broadcaster distances:** WSPR spots under 200 km are sparse and muddied by ground wave | medium | high | the dry run checks density first (FR-10.6); if too sparse, M1's reference returns as a ruling | agent |
| RK-13 | One wrong reading (a NaN, an extreme value) poisons the whole field | low | high | go-ionomaps R-3.3 (physical ranges, no NaN or Inf out) | agent |

## Metrics of success

Normative in `problem-statement.md`:
- M1-M3 (PS-1), M3 split by cause (D-48);
- M4, M5, M5b (PS-2);
- G1, shared;
- G-M1 to G-M4 and G-G1 (go-ionomaps).

M1 and M4 are measured against WSPR, with RBN as a cross-check (D-22); their baseline is the IRI
climatology (D-23). Every target is set from PLAN's dry run, each by its own ruling (D-49).
