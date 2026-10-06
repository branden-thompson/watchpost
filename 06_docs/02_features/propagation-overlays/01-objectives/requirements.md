---
title: "0.19.0 — Propagation overlays — REQUIREMENTS"
date: 2026-10-06
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "DRAFT — approved at the DISCOVER gate. On any conflict with the brief or the problem statement, this file wins once approved, and every change is a row in 02-analysis/rulings.md. Rows marked (path) wait for D-20's path ruling at DISCOVER exit."
---

# Requirements

Each functional requirement traces to a brief requirement (R-n) and to the ruling that makes it necessary.
**What, never how**: the how is PLAN's, and PLAN carries no code (0.18.0 D-13).

Each row's **Instrument** names what will hold it at BUILD exit: a test by its planned name, a gate, or a
HUM-graded metric. A planned name is a commitment PLAN may rename, never drop. REFLECT L1 requires each
metric's instrument to run once before PLAN exit.

**Order (D-4):**
- **O1** is MUF: the Propagation mode, the Broadcaster's map and readout, sources, storage, the words that ship with the layer (D-9).
- **O2** is 0.18.0's accessibility carry-over.
- **O3** is #12 and the `say` fixes (F-187, F-188), with O3's gate fixes allowed earlier where a flaky gate blocks O1 (D-15's reason).

## FR-1 — The Propagation mode (R-1; D-21, D-24 to D-27)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-1.1 *(O1)* | The map window has a **Propagation mode** beside Radar and Forecast, entered and left by a keymap action listed in Help and overridable in `[keys]` | D-21 | `TestThePropagationModeIsAKeymapAction`, `TestTheMapKeysAreActionsHelpShows` (extended) |
| FR-1.2 *(O1)* | The Propagation mode has **its own world-capable bound**; Radar and Forecast keep 0.18.0's region bound unchanged; no weather frame is ever wider than its region | D-21; 0.18.0 D-8, D-28 | `TestNoFrameIsWiderThanTheRegion` (extended: every weather-mode frame), `TestEachModeKeepsItsOwnBound` |
| FR-1.3 *(O1)* | The Propagation mode **opens for any place**, including one outside every US region; the outside-region gate gives way to this mode only | D-21 | `TestAPlaceOutsideEveryRegionOpensThePropagationMode`, `TestAPlaceInNoRegionIsAStatedState` (kept, for weather modes) |
| FR-1.4 *(O1)* | Weather layers are off in the Propagation mode; propagation layers are off in the weather modes | D-21 | `TestEachModeDrawsOnlyItsLayers` |
| FR-1.5 *(O1)* | Two layers, **MUF(3000)** and **foF2**, each with its valid time and age, a legend in MHz, contour labels in MHz, and a day/night terminator | D-24; HR-2 to HR-4 (D-33 to D-35) | `TestBothPropagationLayersDrawWithTheirUnits`, `TestTheTerminatorIsDrawnForTheFieldsTime`; the library's v0.3.0 contract rows |
| FR-1.6 *(O1)* | The fields **continue over the sea** | HR-1 (D-32) | `TestAPropagationFieldCrossesTheSea` (on the library's `TestHostCanFlipEither`) |
| FR-1.7 *(O1)* | Pan and zoom work as in the Observer's map; the view rehydrates with a loading indicator | D-25; 0.18.0 FR-1.10 | `TestThePropagationModePansAndZooms`, `TestTheWindowSaysWhatThePictureIs` (extended) |
| FR-1.8 *(O1)* | **Paths start at the selected place** in the Observer; the selected place is marked | D-26 | `TestObserverPathsStartAtTheSelectedPlace` |
| FR-1.9 *(O1)* | **Continental reference points are the default targets**; the answer is given for each, by UTC hour and HF band, between both limits (D-47) | D-25; R-1.2; D-47 | `TestTheContinentsAreTheDefaultTargets`, M4 |
| FR-1.10 *(O1)* | A location can be **found by city, ZIP or coordinates**, using the Observer's existing lookup (D-147), marked and answered; **found places are never kept**, and can always be found again | D-25, D-27 | `TestAFoundPlaceIsAnsweredAndNotKept`, `TestTheLookupIsTheObserversOwn` |
| FR-1.11 *(O1)* | The answer for **the point at the map's centre** as the listener pans: MUF, foF2, and which HF bands open to it from the origin, on screen and in words | D-37 | `TestTheCentreIsAnsweredAsTheMapPans`, `TestTheCentreAnswerIsSaid` |
| FR-1.12 *(O1)* | The mode is a member of every closed-set window test (reachability at 80×24, memo completeness, margins, glyph survey, keys, leaving) | 0.18.0 FR-1.5 | the closed-set tests, each enumerating the mode |

## FR-2 — The Broadcaster's map and readout (R-2; D-28 to D-30)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-2.1 *(O1)* | The Broadcaster gets **its own map** (F-174): the tower marked, the service-radius ring, the line-up's places; local or state zoom by default; pan and zoom as the Observer's; inside the US regions (0.18.0 D-8/D-28) | D-28 | `TestTheBroadcasterMapShowsTheTowerItsRadiusAndTheLineUp`, `TestNoFrameIsWiderThanTheRegion` (extended to it) |
| FR-2.2 *(O1)* | The Broadcaster's map follows the **tower**, never the Observer's selected place | D-28; `modes/tty/map_pane.go:948-963` | `TestTheBroadcasterMapFollowsTheTower` |
| FR-2.3 *(O1)* | A **propagation readout** on the console: foF2 at the tower, the near-vertical status of each amateur band 160 m to 10 m in words (open, above the upper limit, absorbed, no data: each naming the limit that decided it, D-47), the data's age | D-28, D-24, D-47 | `TestTheReadoutSaysFoF2AtTheTowerAndEachBandsStatus`, M1, M2 |
| FR-2.4 *(O1)* | A **"Propagation readout"** Broadcaster-tab Setting, **on by default**; while on, the console fetches on the data's cadence (at most hourly) with no map open; its cost is stated in the tab's notice area; MAP STATUS lists the host | D-29 | `TestTheReadoutSettingIsOnByDefault`, `TestTheReadoutStatesItsCost`, `TestMapStatusListsEveryPropagationHost` |
| FR-2.5 *(O1)* | A **"My HF bands"** Broadcaster-tab picker: a multi-pick of the amateur bands 160 m to 10 m from the existing picker component, defaulting to 80 m and 40 m, persisted in `config.Broadcaster` | D-30 | `TestMyHFBandsDefaultsTo80And40`, `TestTheBandsPickerIsTheSharedPicker` |
| FR-2.6 *(O1)* | **The closure notice:** for a picked band, a closure the model predicts is announced ahead with its expected time (M3a); a disturbance closure from D-RAP is said within 15 min (M3b; D-RAP fetched every 10 min); a closure seen in the measured field within 90 min (M3c); on screen and in words; none picked silences it | D-30; D-48; M3 | `TestAClosedPickedBandRaisesTheNotice`, M3 |
| FR-2.7 *(O1)* | The readout and notice fit the console's measured floor (44 rows, "fixed chrome plus one readable card"), or the floor is re-measured and ruled | `modes/tty/broadcaster.go:621-647` | `TestEverySizeShowsAStatedStateAndNeverOverflows` (Broadcaster) |
| FR-2.8 *(O1)* | UI wording never uses the bare word "band" for HF, which already names the console's notice strips | D-30 | `TestTheConsoleSaysHFBandsNotBand` |

## FR-3 — In words (R-1.3; D-9)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-3.1 *(O1)* | With no picture (`--ascii`, "Instead of the map"), the Propagation mode says in words what MUF and foF2 are at the selected place, which bands open to each target, and the data's age | D-9, M5b | `TestThePropagationWordsAnswerWithoutThePicture`, M5b (UAT) |
| FR-3.2 *(O1)* | Words carry units (MHz) and band names, never a bare number | HR-3 (D-34) | `TestPropagationWordsCarryUnits` (on the library's `Answer.ValueUnit`) |
| FR-3.3 *(O1)* | The description gains a seam a layer can add its sentence to; the alert sentences are unchanged | `wave1-findings.md` (Words) | `TestALayerAddsItsSentence`, `TestTheMemoKeyCoversEverythingTheFrameShows` (extended) |
| FR-3.4 *(O1)* | The Broadcaster's readout and notice have spoken and screen-reader forms | D-28, M3 ("in words") | `TestTheReadoutIsSaid` |

## FR-4 — Sources, terms, credit and egress (R-3; D-2, D-18, D-29)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-4.1 *(O1)* | Every propagation source is **credited in About** under its own terms (GIRO: CC BY-NC-SA, cite Reinisch & Galkin 2011, acknowledge station providers; NOAA: public domain, no endorsement; others as the path chooses) *(path)* | R-3.1; G-M1 | `TestEverySourceIsCreditedOnce` (extended), `TestEveryRegisteredSourceHasACredit` (new: ties sources to credits) |
| FR-4.2 *(O1)* | The Observer's Propagation mode fetches **nothing before it is opened**; the Broadcaster's readout is the only scheduled fetch (D-29), beside the history recorder | R-3.2; 0.18.0 D-25 | `TestThePropagationModeFetchesNothingUntilOpened` |
| FR-4.3 *(O1)* | Every new host is listed in MAP STATUS with what it learns; a request that names the station's place says so | R-3.3; 0.18.0 D-31 | `TestMapStatusListsEveryPropagationHost`, the egress table in PLAN |
| FR-4.4 *(O1)* | Requests go through watchpost's own client with its User-Agent; go-giro-data takes the host's fetcher | GR-5 | `TestPropagationFetchesUseTheHostsClient` |
| FR-4.5 *(O1)* | **Throttle behaviour (D-39):** GIRO updates ask only the live stations (list refreshed at most daily), in one burst of at most 40, at most hourly, never more than 2 a minute sustained; a 429 stops the update and backs off 60 s doubling to 15 minutes, keeping the last good field with its age; NOAA newest grid only, at most 6 an hour; every response's rate headers are read | D-39; D-38 | `TestAnUpdateNeverExceedsItsBurst`, `TestA429BacksOffAndKeepsTheLastField`, `TestRetryAfterIsHonouredWhenSent` |

## FR-5 — Honesty (R-5)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-5.1 *(O1)* | The data's age is always said, on screen and in words | R-5.1 | `TestTheFieldsAgeIsAlwaysSaid` |
| FR-5.2 *(O1)* | Stale, partial or failed data is said as such, never drawn as current; a failure has a path to resolve or goes to the diagnostics (0.18.0 D-124) | R-5.1; G-M4 | `TestAStaleFieldIsSaidStale`, `TestAFailedFetchIsSaidWithItsPath` |

## FR-6 — Storage (D-31)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-6.1 *(O1)* | The computed global MUF(3000) and foF2 grids are recorded hourly, and the readout's values at the tower | D-31 | `TestPropagationGridsAreRecordedHourly`, `TestTheTowerReadingsAreRecorded` |
| FR-6.2 *(O1)* | The history store opens whenever any recorded source is on, not only NDFD | D-31; `app/history.go:147-156` | `TestTheStoreOpensWithoutNDFD` |
| FR-6.3 *(O1)* | Each dataset may keep its own retention; the Data tab's value is the default | D-31; `app/history.go:496-505` | `TestADatasetKeepsItsOwnRetention` |
| FR-6.4 *(O1)* | Raw source data (GIRO readings) is never stored | D-31; GIRO terms | `TestNoRawSourceDataIsStored` |

## FR-7 — Accessibility carry-over (R-6; D-4)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-7.1 *(O2)* | F-191 (FR-1.8): the window says its picture is braille and names the remedy | 0.18.0 D-244 | as F-191 |
| FR-7.2 *(O2)* | F-192, F-193, F-194, F-180: radar's motion Setting with reduce motion; the loop's rates and hold; the playback speed Setting | 0.18.0 D-245, D-255 | as each row |
| FR-7.3 *(O2)* | F-196, F-197, F-198: the Monochrome theme hint; the colour-literal guard; the `CheckRamp` matrix | 0.18.0 D-247 | as each row |
| FR-7.4 *(O2)* | F-205: the cursor placed for a screen reader; the description beyond alerts; the edge warning without a picture; the state words; the Maps tab; the outline key; "With the map" | 0.18.0 D-273 | as F-205's findings (AX-3 to AX-9) |

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
| FR-10.1 | Each metric's instrument runs once before PLAN exit, its number or gap in the PLAN report | L1 | the PLAN report |
| FR-10.2 | A pre-push hook runs the docs-reading tests | L5 | the hook, and a watched failure |
| FR-10.3 | The code-quality brief asks for any cross-layer ruling without an end-to-end test; the scripted-PTY journey (0.18.0 D-271) gains the Propagation mode and the Broadcaster's map | L2 | `06_docs/red-team-brief.md`; the journey |
| FR-10.4 | Ruling questions cite `file:line` for every claim about the code | L3 | the rulings log |
| FR-10.5 | A batch carrying UAT verdicts lists their D-rows | L8 | the build log |

## FR-11 — The acknowledgement (D-45, D-46)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-11.1 *(O1)* | The first time propagation data is shown (the Propagation mode, or the Broadcaster's map or readout), a window says that watchpost does not transmit, that MUF and foF2 are provided as reference, and that anyone who transmits on HF is responsible for following all applicable laws where they are | D-45, D-46 | `TestTheAcknowledgementShowsBeforeTheFirstPropagationView` |
| FR-11.2 *(O1)* | Closing it, with any of its keys, records that it was seen; it never gates the data or the fetch | D-46 | `TestClosingTheAcknowledgementRecordsItSeen`, `TestTheAcknowledgementGatesNothing` |
| FR-11.3 *(O1)* | The record keeps the wording's version; a changed wording is shown once more | A-9 | `TestAChangedAcknowledgementShowsAgain` |
| FR-11.4 *(O1)* | The window is drawn as text, readable without the picture, and named in Help | D-9; AX | `TestTheAcknowledgementReadsWithoutThePicture` |

# Non-functional requirements

| # | Requirement | Source | Instrument |
|---|---|---|---|
| NFR-1 | **Cost (G1):** watchpost's added CPU and memory with propagation on stay within G1's target, set in DISCOVER | issue #25; D-8 | G1's measurement |
| NFR-2 | **The library's cost (G-G1):** one update's CPU and memory, measured in go-giro-data | D-12 | G-G1 |
| NFR-3 | **Live probes follow the feature's throttle** in every phase (D-39, superseding D-18's 5 a day); never Open-Meteo for research; no kc2g.com request until asked | D-39 | the request log, with headers, in each findings page |
| NFR-4 | **P10 clean** at every phase exit (`make p10`) | standing | `make p10` |
| NFR-5 | **Both CI platforms green** before merge, the mode named (quick or full) | standing | the hosted run |

# Risk register

| # | Risk | Likelihood | Impact | Mitigation | Owner |
|---|---|---|---|---|---|
| RK-1 | The reference implementation has no licence; a reimplementation drifts toward its code | low | high | G-R8: every algorithm cites published work; `arodland/prop` never opened | agent |
| RK-2 | GIRO's terms (NC/SA) and rate limits; path B costs about 960 requests a day a copy (`wave2-findings.md`) | high (if B) | high | D-20 measures it; path D or a sparse schedule | HUM LEAD at D-20 |
| RK-3 | The chosen field is not accurate enough (GloTEC overestimated foF2 at two stations in well-observed cells) | medium | high | wave 2's pairs; G-M3 against held-out ionosondes; M1/M4 against climatology (D-23) | agent |
| RK-4 | Compute or memory too heavy for a terminal app | low | medium | G1, G-G1; 2° default; recompute on new data only | agent |
| RK-5 | Three paired releases at SEV-0 cost time | high | medium | D-13 (minor rulings batched); one record | HUM LEAD, agent |
| RK-6 | Accessibility last again (D-4 order) | medium | high | M5b and FR-3 ship words with the layer; O2 is in scope, not optional | HUM LEAD |
| RK-7 | Two bounds in one window let a weather frame go global | low | high | FR-1.2's tests over every frame | agent |
| RK-8 | The readout fetches for stations that never use HF (D-29 on by default) | medium | low | the Setting turns it off; the cost is stated | HUM LEAD (D-29) |
| RK-9 | The history-store changes (FR-6.2, FR-6.3) break 0.18.0's datasets | medium | medium | existing history tests; a retention test per dataset | agent |
| RK-10 | The library's label move (D-34) shifts every existing layer's labels | high | low | specimens redrawn in the same batch (`diagrams-move-with-code`) | agent |

## Metrics of success

Normative in `problem-statement.md`:
- M1-M3 (PS-1);
- M4, M5, M5b (PS-2);
- G1, shared;
- G-M1 to G-M4 and G-G1 (go-giro-data).

M1 and M4 are measured against WSPR, with RBN as a cross-check (D-22). Their baseline is the IRI
climatology (D-23). Targets are set in DISCOVER and the PLAN-time dry run.
