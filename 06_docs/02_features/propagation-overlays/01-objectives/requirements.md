---
title: "0.19.0 — Propagation overlays — REQUIREMENTS"
date: 2026-10-06
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "APPROVED at the DISCOVER gate (D-84), normative. On any conflict with the brief or the problem statement, this file wins once approved, and every change is a row in 02-analysis/rulings.md. Revised after the DISCOVER-exit red team (08-reports/red-team-discover.md), the reference-chart ruling (D-76), PLAN's dry run and rulings (D-85 to D-108) and the PLAN-exit red team (08-reports/red-team-plan.md, D-109 to D-127)."
---

# Requirements

## What the listener sees (D-70, D-82)

**Observer and Broadcaster alike open one place for propagation: the map window's Propagation mode.**
- **It shows two world maps of the ionosphere**, MUF(3000) and foF2, with a day/night line, over land and sea (D-21, D-24, D-32 to D-35). It opens for any place, even outside the US regions; weather maps stay within their region (D-21).
- **"Best bands now"** says which amateur HF bands work right now, and at which hours today:
  - for the area around you (short, near-vertical paths);
  - for the continents and any place you find, by city, ZIP, coordinates or a Maidenhead grid square (D-25, D-61, D-76, D-79).
- **"Your frequency"**: type a frequency or a band ("7.188", "40 m") and it shows how far it should reach today: the skip zone and the area reached, drawn on the map and said in words (D-76).
- **Stepping ahead**: hours can be stepped up to a day ahead, each labelled typical for that hour (D-50, D-109). The map's centre is answered as you pan (D-37).
- **Without the picture**, everything is said in words, and your place is always said (D-9, D-64).
- **From the Broadcaster**, a key opens it from the tower (D-57). Help says the broadcast shares the computer's default sound output with a screen reader (D-85).
- **The first time it opens**, before anything is fetched, a window says that:
  - watchpost transmits nothing;
  - the maps are reference;
  - anyone transmitting on HF must follow the law where they are;
  - GIRO and NOAA will see your IP address.
  Enter or Esc closes it (D-45, D-46, D-81).
- **Nothing is fetched until it is opened** (D-76). Sources are credited, and GIRO's non-commercial terms are stated (FR-4).

**Not in 0.19.0**, by ruling:
- the Broadcaster's own regional map (D-77);
- a monitoring readout and closure notices; a "band watch" is a follow-up (D-76);
- propagation history and backfill (D-78);
- station dots, the eSSN chart and the full ITU-R P.533 planner (D-50);
- an observed-spots layer (D-67).

**Glossary**

| Term | Meaning |
|---|---|
| MUF(3000) | maximum usable frequency: the highest frequency a signal reflects off the ionosphere over a 3000 km path |
| foF2 | the F2 layer's critical frequency: the highest frequency reflected straight up, which governs short, near-vertical paths |
| near-vertical (NVIS) | paths of a few hundred kilometres, the signal going up steeply and coming down nearby |
| upper limit | the highest frequency that works on a path (from MUF or foF2) |
| absorbed / lower limit | lower frequencies are soaked up by the D layer in daylight; below this limit a band is closed |
| disturbed | a solar flare or particle event is absorbing more than usual (NOAA's D-RAP); degraded, not necessarily closed |
| space-weather scales | NOAA's levels 0 to 5 for radio blackouts (R), solar radiation storms (S) and geomagnetic storms (G); named in the chart, not modelled (D-88) |
| reference circuit | what "open" means here: SSB voice at 100 W with simple antennas, needing +13 dB signal-to-noise in 2.5 kHz (D-73; ITU-R F.339-8's "just usable", D-91) |
| skip zone | the ring around a transmitter that a frequency jumps over: too far for the ground wave, too near for the sky wave |

## How to read the rows

Each functional requirement traces to a brief requirement (R-n) and to the ruling that makes it necessary.
**What, never how**: the how is PLAN's, and PLAN carries no code (0.18.0 D-13). Each row's **Instrument**
names what will hold it at BUILD exit: a test by its planned name, a gate, or a HUM-graded metric. A
planned name is a commitment PLAN may rename, never drop. A row whose instrument is a document is a
**process commitment**: a document cannot fail, so the phase reports keep it.

**Order (D-4):**
- **O1** is MUF: the Propagation mode, its reference chart and words (D-9), sources, the shared-device statement (D-85). #27's roll-up fix (FR-6.5) comes first in BUILD (D-55).
- **O2** is 0.18.0's accessibility carry-over, held by a SHIP gate (D-63).
- **O3** is #12 and the `say` fixes (F-187, F-188), allowed earlier where a flaky gate blocks O1 (D-15).

## FR-1 — The Propagation mode (R-1; D-21, D-24 to D-27, D-37, D-50, D-57, D-61, D-64, D-73, D-76, D-79)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-1.1 *(O1)* | The map window has a **Propagation mode** beside Radar and Forecast, entered and left by a keymap action listed in Help and overridable in `[keys]`; **the current mode is named in words in every path** (picture, "Instead of the map", `--ascii`) | D-21; AX-A | `TestThePropagationModeIsAKeymapAction`, `TestTheMapKeysAreActionsHelpShows` (extended), `TestTheModeIsNamedInEveryPath` |
| FR-1.2 *(O1)* | The Propagation mode has **its own world-capable bound**; Radar and Forecast keep 0.18.0's region bound unchanged; no weather frame is ever wider than its region | D-21; 0.18.0 D-8, D-28 | `TestNoFrameIsWiderThanTheRegion` (extended: every weather-mode frame), `TestEachModeKeepsItsOwnBound` |
| FR-1.3 *(O1)* | The Propagation mode **opens for any place**, including one outside every US region; the outside-region gate gives way to this mode only, and its message names the Propagation key | D-21; AX-M | `TestAPlaceOutsideEveryRegionOpensThePropagationMode`, `TestAPlaceInNoRegionIsAStatedState` (kept, for weather modes), `TestTheOutsideMessageNamesThePropagationKey` |
| FR-1.4 *(O1)* | Weather layers are off in the Propagation mode; propagation layers are off in the weather modes | D-21 | `TestEachModeDrawsOnlyItsLayers` |
| FR-1.5 *(O1)* | Two layers, **MUF(3000)** and **foF2**, each with its valid time and age, a legend in MHz, contour labels in MHz, and a day/night terminator marked without relying on colour | D-24; D-32 to D-35; AX-E | `TestBothPropagationLayersDrawWithTheirUnits`, `TestTheTerminatorIsDrawnForTheFieldsTime`; the library's v0.3.0 rows (L-3, L-4.4) |
| FR-1.6 *(O1)* | The fields **continue over the sea** | D-32 | `TestAPropagationFieldCrossesTheSea` (on the library's `TestHostCanFlipEither`) |
| FR-1.7 *(O1)* | Pan and zoom work as in the Observer's map; the view rehydrates with a loading indicator | D-25; 0.18.0 FR-1.10 | `TestThePropagationModePansAndZooms`, `TestTheWindowSaysWhatThePictureIs` (extended) |
| FR-1.8 *(O1)* | **The origin** is the selected place in the Observer, or the tower when opened from the Broadcaster (FR-2.1); it is always marked | D-25, D-26, D-57 | `TestTheOriginIsTheSelectedPlaceOrTheTower` |
| FR-1.9 *(O1)* | **Continental reference points are the default targets**; places found (FR-1.10) are answered too | D-25 | `TestTheContinentsAreTheDefaultTargets` |
| FR-1.10 *(O1)* | A location can be **found by city, ZIP, coordinates or a Maidenhead locator**: watchpost's embedded index (GeoNames cities15000, already shipped and credited, `domains/locations/geodata/index.go`) answers first, with a **mode-aware coverage gate** so non-US cities are accepted in this mode (`domains/locations/resolver.go:101-102` refuses them today); coordinates and 4- or 6-character locators parse locally (finite, latitude ±90, longitude wrapped) and never go online; only when nothing local matches does the online geocoder answer, uncached; no error quotes the typed text; the found place is marked, answered and **never kept**; focus returns to the map and the found place is said | D-25, D-27, D-61, D-79; IS-4, N-2, N-3, AX-B | `TestAFoundPlaceIsAnsweredAndNotKept`, `TestALocalMatchSendsNothing`, `TestCoordinatesAndLocatorsNeverGoOnline`, `TestTheOnlineFallbackIsUncached`, `TestFocusReturnsAfterAFind` |
| FR-1.11 *(O1)* | The answer for **the point at the map's centre** as the listener pans: MUF, foF2, and which bands work to it from the origin, on screen and in words | D-37 | `TestTheCentreIsAnsweredAsTheMapPans`, `TestTheCentreAnswerIsSaid` |
| FR-1.12 *(O1)* | The mode is a member of every closed-set window test (reachability at 80×24, memo completeness, margins, glyph survey, keys, leaving) | 0.18.0 FR-1.5 | the closed-set tests, each enumerating the mode |
| FR-1.13 *(O1)* | **Hours ahead (D-103, D-109):** the listener steps the fields and answers up to 24 h ahead, one hour per key press (no automatic play). **Each hour ahead is the climatology for that hour, and carries a prominent label** beside the source chip and in words: "typical for this hour", not a forecast of today, with its typical error for the place's distance band as measured on PLAN's week, and that basis stated (D-134). While NOAA's G scale is 1 or more, each hour ahead also says a storm is in progress and typical conditions may not hold (D-88). The current hour stays the default and the freshest field (D-94) | D-50, D-75, D-88, D-103, D-109; AX-J | `TestTheHoursStepOnAKey`, `TestAnHourAheadSaysTypical`, `TestTheTypicalErrorStatesItsBasis`, `TestAStormSaysTypicalMayNotHold` |
| FR-1.14 *(O1)* | **Every control in the mode is a keymap action** Help lists: choosing MUF or foF2, stepping the hour, moving between targets, finding a place, entering a frequency, **refreshing now** (D-120: honoured under D-39, an early ask answered from the last field with its age and the reason); the focused target is named in words, and **each control's current state is said: the layer (MUF or foF2), the hour, the frequency entry open and what it holds (D-127)**; Help fits its window with them | AX-B; N-13; D-120 | `TestEveryPropagationControlIsAnAction`, `TestHelpFitsWithThePropagationKeys`, `TestEachControlsStateIsSaid` |
| FR-1.15 *(O1)* | **"Best bands now"** (D-76): for the near-vertical area around the origin (within 400 km of the selected place by default, a Maps-tab Setting offering 200, 400 or 600 km, A-18, D-100; the service radius when opened from the tower) and for each target, each amateur band 160 m to 10 m is called open, above the upper limit, absorbed, disturbed, or no data, under the reference circuit (D-73), naming the limit that decided it; with **the hours today each band is typically open**: the climatology for every hour of the UTC day, labelled "typical day", beside the current hour's own answer; where they disagree the words say "now differs from a typical day" (routine closures as a schedule, D-74, D-135) | D-47, D-73, D-74, D-76, D-100; AX-F | `TestBestBandsNameTheirLimit`, `TestTheDaysOpenHoursAreShown`, `TestTheTypicalDayIsLabelled`, `TestADisagreementWithTheTypicalDayIsSaid`, `TestTheNearVerticalRadiusIsASetting`; M1, M4 |
| FR-1.16 *(O1)* | **"Your frequency"** (D-76): a frequency in MHz or a band name is accepted, checked against HF, 1.8 to 30 MHz, and parsed locally; for this hour (or an hour ahead, typical, D-109), the skip zone and the area reached from the origin are drawn on the map and said in words, with the limit that bounds them | D-76 | `TestYourFrequencyDrawsItsReach`, `TestYourFrequencyIsSaid`, `TestABadFrequencyIsSaidNotGuessed`; M1, M4 |
| FR-1.17 *(O1)* | The chart's answers (best bands, your frequency, the centre) never block a keypress: they are computed off the UI goroutine (A-27; an hour step measured at about 4.4 ms at 2° in the dry run's spike, with placeholder formulas of equivalent cost) | N-8; A-27 | `TestAnswersNeverBlockTheUI` |
| FR-1.18 *(O1)* | **Space weather named (D-88):** any of NOAA's R, S or G scales above 0 is named in the chart and in the words, with NOAA's outlook; while G is 1 or more, each hour ahead says a storm is in progress and typical conditions may not hold (D-109); a missing scales feed is said, not shown as level 0 | D-88; PM-E2 | `TestARaisedScaleIsNamed`, `TestAStormSaysTypicalMayNotHold`, `TestMissingScalesAreNotShownAsZero` |

## FR-2 — The Broadcaster (R-2; D-57, D-80)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-2.1 *(O1)* | **A console key opens the Propagation mode** over the console, its origin the tower, always marked; the key is a console action Help lists and clashes with no console key (`+`, `=`, `-`, `O` are taken) | D-57, D-25, D-26; AX-H | `TestTheConsoleOpensThePropagationModeFromTheTower`, `TestTheConsoleKeysDoNotClash` |
| FR-2.2 *(O1)* | **Help and the Broadcaster tab say plainly** that the broadcast plays on the computer's default output device, the same one a screen reader speaks on, so screen-reader speech can reach the broadcast audio. The device Setting is F-211 (D-85) | D-80, D-85; N-1 | `TestTheSharedDeviceIsSaid` |
| FR-2.3 *(O1)* | **The Broadcaster's Propagation mode honours the shared Settings (D-121):** "Instead of the map", the refresh Setting and the no-readings correction are one Setting each across both modes; the near-vertical radius is Observer-only (the Broadcaster uses its service radius) | D-58, D-121; A-3 | `TestTheConsolePropagationModeHonoursTheSharedSettings`, `TestInsteadOfTheMapIsShared` |

**Retired from FR-2 by ruling:** the regional Broadcaster map and its words (D-77; D-28, D-58), the readout,
its Setting, the "My HF bands" picker, the closure notice and the console line (D-76; D-29, D-30, D-48, D-59,
D-60, D-74).

## FR-3 — In words (R-1.3; D-9)

"In words" and "said" mean **screen text a screen reader reads**. The broadcast plays on the computer's
default output device, which a screen reader shares; FR-2.2 says so plainly, and the device Setting is a
follow-up (D-85, F-211).

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-3.1 *(O1)* | With no picture (`--ascii`, "Instead of the map"), the mode says in words, **in this order (D-136)**: the origin, always, wherever the view is (D-64); MUF and foF2 there; day or night there and when that changes (AX-E); **a status block**, said once per open: the data's age, the nearest station and its confidence (D-116), any space-weather scale above 0 (FR-1.18), an unusual offset (FR-5.4), "updated at" (FR-3.6), a disagreement with the typical day (D-135); "best bands now" (FR-1.15), **each target one line, its name first, open bands before closed (D-127)**, a far target's line carrying its own "about typical"; "your frequency", if entered (FR-1.16); and the centre's answer (D-37). The focus limit is in Help and said once on opening (FR-3.7) | D-9, D-64, D-76, D-127; M5b; AX-E, D-136 | `TestThePropagationWordsAnswerWithoutThePicture`, `TestTheWordsFollowTheRuledOrder`, `TestCaveatsAreSaidOncePerOpen`, M5b (UAT) |
| FR-3.2 *(O1)* | Words carry units and band names written for speech: "megahertz", "metres", each layer's full name once, never a bare number | D-34; AX-K | `TestPropagationWordsCarryUnitsForSpeech` |
| FR-3.3 *(O1)* | The description gains a seam a layer adds its sentence to, reached **before** the alert-only early returns (`modes/tty/map_describe.go:58-70`); the alert sentences are unchanged | AX-C; N-9 | `TestALayerAddsItsSentence` (four cases: alerts off, alerts never asked, the selected place out of view, no selected place), `TestTheMemoKeyCoversEverythingTheFrameShows` (extended) |
| FR-3.5 *(O1)* | Without the picture, the mode's status (loading, offline, a coarse picture, the data's age, the fallback in use) is still said | AX-D | `TestTheStatusIsSaidWithoutThePicture` |
| FR-3.6 *(O1)* | **A refresh keeps the reader's place (D-127):** scroll and focus are kept, and the words say once that they were updated, and when | D-127; A-10 | `TestARefreshKeepsTheReadersPlace`, `TestARefreshIsSaidOnce` |
| FR-3.7 *(O1)* | **The focus limit is said (D-133):** Help and the Propagation mode's words state that a screen reader cannot yet follow focus in this mode, and that 0.19.1 fixes it (F-205) | D-133; persona N6 | `TestTheFocusLimitIsSaid` |

## FR-4 — Sources, terms, credit and egress (R-3; D-2, D-22, D-39, D-41, D-43, D-69, D-76, D-81, D-83)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-4.1 *(O1)* | Every propagation source is **credited in About** under its own terms: GIRO (CC BY-NC-SA 4.0; Reinisch & Galkin 2011; each station's provider; **non-commercial access** stated); NOAA SWPC GloTEC, D-RAP, the space-weather scales and the daily solar indices (public domain; no endorsement); PyIRI (MIT; Forsythe et al. 2024; the NRL refits and their CCIR/URSI lineage); GeoNames (already credited). WSPR (wspr.live, WSPRnet) and RBN are credited in the reports that use them | G-M1; D-22, D-41, D-43, D-69; IS-5 | `TestEverySourceIsCreditedOnce` (extended), `TestEveryRegisteredSourceHasACredit` (new: ties sources to credits) |
| FR-4.2 *(O1)* | **Nothing propagation-related is fetched until the Propagation mode is opened**, and nothing before its acknowledgement (FR-11) has been shown; nothing is fetched in the background | D-76, D-81; 0.18.0 D-25 | `TestThePropagationModeFetchesNothingUntilOpened`, `TestNothingIsFetchedBeforeTheAcknowledgement` |
| FR-4.3 *(O1)* | Every new host is listed in MAP STATUS with what it learns: GIRO and NOAA learn the IP (GIRO's replies echo it back); the online geocoder learns the typed text; no request names the origin | 0.18.0 D-31; IS-2, IS-4 | `TestMapStatusListsEveryPropagationHost` (reads go-ionomaps' exported host set, D-113), the egress table in PLAN |
| FR-4.4 *(O1)* | **The propagation client is its own instance** (not the data client's), with no shared pacing hold and no host avoidance from other traffic; it **never retries a 429** and returns status and headers on every outcome, so go-ionomaps alone decides a back-off (`platform/httpx/httpx.go:926-928` retries a 429; `memo.go:138-149` holds every lane on a Retry-After; `httpx.go:838-847` drops headers on a non-200) | D-39; IS-1, N-5 | `TestThePropagationFetcherNeverRetriesA429`, `TestA429ReturnsItsHeaders`, `TestAGIRORefusalNeverPausesWeather` |
| FR-4.5 *(O1)* | At run time only rate-related headers are read (`Retry-After`, `X-RateLimit-*`, `RateLimit-*`); their first sighting goes to the diagnostics by host and header name | D-83 | `TestOnlyRateHeadersAreRead` |
| FR-4.6 *(O1)* | **Nothing raw is kept:** GIRO's replies (which carry the requester's IP) never enter the HTTP cache, the diagnostics, an error or an error's reason text (the propagation client keeps no reply text, I-8); GloTEC, D-RAP and the scales are fetched uncached and dropped after parsing, with their validators (ETag, Last-Modified) held in memory so an unchanged poll costs a 304 | IS-2; PF-F6; N-12 | `TestNoRawSourceDataIsStored` (scans the HTTP cache directory and the diagnostics), `TestAnUnchangedFeedCostsA304` (D-RAP, the scales, the solar indices; GloTEC grids are timestamped, D-111) |
| FR-4.7 *(O1)* | The propagation client refuses plain http, private addresses and **any host outside go-ionomaps' exported set** (D-113; this guards against a bug in the library, not a compromised one, which would export its own hosts; persona N9), with a body cap sized to each source | IS-6; D-113 | `TestThePropagationClientIsHardened`, `TestTheClientRefusesAnUnlistedHost` |
| FR-4.8 *(O1)* | **One go-ionomaps object per process**, shared by every caller; the update runs off the UI goroutine | PF-F7, PF-F8 | `TestOneLibraryObjectPerProcess`, `TestTheUpdateNeverRunsOnTheUIGoroutine` |
| FR-4.9 *(O1)* | **The coordinate parser is shared and finite-checked** in front of every resolver, the Observer's lookup and `watchpost report` included: coordinates never go online, `NaN` and `Inf` are refused, and the geocoder's `name` parameter is redacted in logs | N-3 (not excused by predating the change) | `TestEveryResolverParsesCoordinatesLocally`, `TestNaNCoordinatesAreRefused`, `TestTheGeocoderQueryIsRedacted` |
| FR-4.10 *(O1)* | **Refresh while open (D-94, A-26):** while the mode is open, the fields follow each new GloTEC grid (every 10 minutes by default); GIRO is asked at most hourly and its last readings are re-assimilated over each new grid; D-RAP and the scales come with each update. A Setting (Maps tab) chooses every 10 minutes (default), hourly or on demand (the refresh-now key, FR-1.14), each stating its measured wire cost while open (D-111, D-120). Nothing refreshes while the mode is closed | D-94, A-26; P-2 | `TestAnOpenModeFollowsEachNewGrid`, `TestGIROIsAskedAtMostHourlyWhileOpen`, `TestTheRefreshSettingStatesItsCost`, `TestNothingRefreshesWhileClosed` |

## FR-5 — Honesty (R-5)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-5.1 *(O1)* | The data's age is always said, on screen and in words: **the age of the oldest input in use** (GloTEC's grid or the oldest GIRO reading assimilated, from the snapshot's inputs; A-30), not the time the field was computed | R-5.1; A-30 | `TestTheFieldsAgeIsAlwaysSaid`, `TestTheAgeIsTheOldestInputs` |
| FR-5.2 *(O1)* | Stale, partial or failed data is said as such, never drawn as current; a failure has a path to resolve or goes to the diagnostics (0.18.0 D-124); the fallback background (climatology) is named when it is in use | G-M4; D-40 | `TestAStaleFieldIsSaidStale`, `TestAFailedFetchIsSaidWithItsPath`, `TestTheFallbackIsNamed` |
| FR-5.3 *(O1)* | An hour ahead is never shown as a measurement or as a forecast of today: it is said to be typical (D-109) | D-50, D-109 | `TestAnHourAheadSaysTypical` |
| FR-5.4 *(O1)* | **The offset, compared (D-105):** Status and the diagnostics show the live offset (GloTEC against the stations now) beside its typical value; when they differ by more than the threshold (2 SD; 0.45 MHz on PLAN's week), the chart's status label and the words say GloTEC is unusually far from the stations now. A Maps-tab Setting chooses, for when no station readings are held, GloTEC as is (default) or corrected by the typical offset; the state in use is named | D-105; RK-11 | `TestStatusShowsTheLiveAndTypicalOffset`, `TestAnUnusualOffsetIsSaid`, `TestTheNoReadingsCorrectionIsASetting` |
| FR-5.5 *(O1)* | **Confidence by distance (D-116):** each answer says its distance to the nearest reporting station, in the words and the status label; beyond 2000 km it says the answer is about as good as typical there, or worse (D-129) | D-116, D-129; L-F2 | `TestAnAnswerSaysItsNearestStation`, `TestAFarAnswerSaysAboutTypical` |

## FR-6 — Storage (D-55, D-78)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-6.5 *(first in BUILD)* | **The roll-up never erases a year (#27):** a failed read of the year's roll-up refuses the write; roll-ups are split so that no document passes the store's read cap, which is measured on the **decompressed** size (`platform/history/history.go:879`), the split sized from each dataset's bytes, not fixed by month; each dataset has a byte bound beside its time retention; values are stored compactly, not one allocation each | D-55; PF-F4, PF-F5; N-4 | `TestAFailedYearReadNeverRewritesTheYear`, `TestARollUpNeverPassesTheReadCap`; PLAN measures whether a shipped dataset can reach the cap (D-55) |

**Retired from FR-6 by D-78:** recording propagation grids and tower values, backfill, the store opening
without NDFD, per-dataset retention, and the dataset terms field (FR-6.1 to FR-6.4, FR-6.6, FR-6.7).

## FR-7 — Accessibility carry-over (R-6; D-4, D-63, D-122)

**Moved to 0.19.1 (D-122)**, the very next release; these rows are its requirements, and D-63's SHIP gate applies to 0.19.1's SHIP.

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-7.1 *(0.19.1)* | F-191 (FR-1.8): the window says its picture is braille and names the remedy | 0.18.0 D-244 | as F-191 |
| FR-7.2 *(0.19.1)* | F-192, F-193, F-194, F-180: radar's motion Setting with reduce motion; the loop's rates and hold; the playback speed Setting | 0.18.0 D-245, D-255 | as each row |
| FR-7.3 *(0.19.1)* | F-196, F-197, F-198: the Monochrome theme hint; the colour-literal guard; the `CheckRamp` matrix | 0.18.0 D-247 | as each row |
| FR-7.4 *(0.19.1)* | F-205: the cursor placed for a screen reader; the description beyond alerts; the edge warning without a picture; the state words; the Maps tab; the outline key; "With the map" | 0.18.0 D-273 | as F-205's findings (AX-3 to AX-9) |
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
| FR-9.2 *(O3)* | A `say` left by a watchpost killed outright is found and ended at the next start, by process, not by its temp directory's name; **only a process proven to be watchpost's is ended**: the same user, a `watchpost-say-` path in its arguments, and a process ID and start time watchpost recorded (I-4) | F-187; I-4 | `TestAnOrphanedSayIsEndedAtStart`, `TestAnUnrelatedSayIsNeverEnded` |
| FR-9.3 *(O3)* | `make test-say` cannot hang the gate: it has a bound, and a hang is diagnosed (the hung process sampled) rather than retried blind | F-188 | the gate's own bound; `06_docs/required-gates.txt` |

## FR-10 — Instruments (R-9; REFLECT L1-L8)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-10.1 | Each metric's instrument runs once before PLAN exit, its number or gap in the PLAN report | L1 | **process commitment**: the PLAN report |
| FR-10.2 | A pre-push hook runs the docs-reading tests | L5 | the hook, and a watched failure |
| FR-10.3 | The code-quality brief asks for any cross-layer ruling without an end-to-end test; the scripted-PTY journey (0.18.0 D-271) gains the Propagation mode and its key from the Broadcaster | L2 | `06_docs/red-team-brief.md`; the journey |
| FR-10.4 | Ruling questions cite `file:line` for every claim about the code | L3 | a docs-reading test: every ruling row naming code carries a `file:line` |
| FR-10.5 | A batch carrying UAT verdicts lists their D-rows | L8 | **process commitment**: the build log |
| FR-10.6 | **PLAN's dry run measures**, before targets are set (D-49) and against the floors set before it (D-75): MUF(3000) and foF2 leave-one-station-out for B on D, by station and by distance, with signed bias; the mainland-US and Pacific stations reported separately; the NRL refits (D-43) with the F10.7 rule written down; GloTEC's availability; the forecast hours' error at +3 h and +12 h (D-75); D-RAP's file; WSPR spots at near-vertical distances under the reference circuit (D-73), before M1's scenarios are built; the answers' per-keypress cost (FR-1.17); G1 over at least 48 hours with downloads a day; M3 is retired (D-76) | D-49, D-73, D-75; CQ-E2, DQ-F20, PF-F9, PM-K3, I-4 | **process commitment**: the PLAN report |

## FR-11 — The acknowledgement (D-45, D-46, D-81)

| # | Requirement | Source | Instrument |
|---|---|---|---|
| FR-11.1 *(O1)* | The first time the Propagation mode opens, **before its first fetch**, a window says that watchpost does not transmit; that MUF and foF2 are provided as reference; that anyone who transmits on HF is responsible for following all applicable laws where they are; and that GIRO and NOAA will learn the IP (MAP STATUS lists them) | D-45, D-46, D-81 | `TestTheAcknowledgementShowsBeforeTheFirstFetch` |
| FR-11.2 *(O1)* | **Enter or Esc** closes it and records that it was seen; it never gates the data | D-46, D-81 | `TestClosingTheAcknowledgementRecordsItSeen`, `TestOnlyEnterOrEscCloseIt` |
| FR-11.3 *(O1)* | The record keeps the wording's version; a changed wording is shown once more | A-9 | `TestAChangedAcknowledgementShowsAgain` |
| FR-11.4 *(O1)* | The window is drawn as text, readable without the picture, with the terminal cursor on its first line, and named in Help | D-9, D-81 | `TestTheAcknowledgementReadsWithoutThePicture`, `TestTheCursorIsPlacedOnTheAcknowledgement` |

# Non-functional requirements

| # | Requirement | Source | Instrument |
|---|---|---|---|
| NFR-1 | **Cost (G1):** watchpost's added CPU, memory and downloads a day with the Propagation mode in use, measured over at least 48 hours, stay within G1's target (Targets in `problem-statement.md`: added CPU at most 1%; added RSS at most 25 MB, judged on measured RSS (D-112, D-131); downloads counted on the wire and re-ruled from W0.0's measurement (D-111)). The library's own cost is go-ionomaps' R-8.1 (G-G1, D-98) | issue #25; D-49, D-95 | G1's measurement |
| NFR-3 | **Live requests follow the feature's throttle** in every phase (D-39); never Open-Meteo for research; no kc2g.com request until asked | D-39, D-83 | the request log, CDN location headers stripped, kept in go-ionomaps `02-analysis/evidence/` |
| NFR-4 | **P10 clean** at every phase exit (`make p10`) | standing | `make p10` |
| NFR-5 | **Both CI platforms green** before merge, the mode named (quick or full) | standing | the hosted run |

# Risk register

| # | Risk | Likelihood | Impact | Mitigation | Owner |
|---|---|---|---|---|---|
| RK-1 | The reference implementation has no licence; a reimplementation drifts toward its code | low | high | D-53: `arodland/prop` was read through the GitHub API in 0.18.0's research and is never opened again; no code copied; go-ionomaps is written from papers and PyIRI (MIT); every exported function cites its source; a provenance table in PLAN | agent |
| RK-2 | **GIRO's load and terms:** an update asks GIRO about 39 times; GIRO offers access "only for educational and non-commercial research purposes"; contact was ruled out (D-41) | medium | high | fetches only when the mode is opened (D-76); D-39's throttle in one object per process; the seed and rotation (D-51); the fallback background if GIRO refuses (D-40, D-75); fleet scenarios in the DISCOVER report; F-208 | HUM LEAD (D-41) |
| RK-3 | **The chosen field is not accurate enough.** PLAN's week (held out, test days): the hybrid's foF2 RMS 0.96 MHz overall and 0.66 in the mainland US against the climatology's 1.14 and 0.82; within 500 km of a reporting station 0.33; in the Pacific worse than the climatology (MUF 5.17 against 5.09; D-129) | medium | high | the hybrid (D-101); confidence said by distance (D-116); G-M3's targets (D-108) and their route on a miss (D-118) | agent |
| RK-4 | Compute or memory too heavy for a terminal app | low | medium | G1 over 48 hours; G-G1; 2° default; answers off the UI goroutine (FR-1.17, FR-4.8) | agent |
| RK-5 | Three paired releases at SEV-0 cost time: about 70 batches across the three, scaled from 0.18.0's measured 0.8 batches a task (round 2, LN8) | high | medium | D-13 (minor rulings batched); scope cut at D-76 to D-78; O2 moved to 0.19.1 (D-122) | HUM LEAD, agent |
| RK-6 | Accessibility last again (D-4 order): O2 moved to 0.19.1 (D-122), so 0.19.0 ships a mode a screen reader cannot follow by focus | high | high | M5b and FR-3 ship words with the layer; the limit is said in Help and the words (FR-3.7, D-133); FR-7's SHIP gate applies to 0.19.1 (D-122) | HUM LEAD |
| RK-7 | Two bounds in one window let a weather frame go global | low | high | FR-1.2's tests over every frame | agent |
| RK-9 | #27's fix changes a store 0.18.0's datasets rely on | medium | medium | existing history tests; the split and byte bound tested per dataset | agent |
| RK-10 | The library's label move (D-34) shifts every existing layer's labels | high | low | specimens redrawn in the same batch (`diagrams-move-with-code`) | agent |
| RK-11 | **False "open":** both backgrounds read foF2 high (PLAN's week: GloTEC +0.51 MHz, the climatology +0.28) | low | high | the assimilation's mean term removes it (the hybrid's held-out bias +0.03); the live and typical offsets compared and said (D-105); every answer states its limit, age and nearest station (D-116) | agent |
| RK-12 | **M1 may be unmeasurable at near-vertical distances** | low | high | measured: at +13 dB with ground wave removed, WSPR under 400 km is dense on 160 to 40 m in all 24 hours of one day, 2026-10-05 (`07-readiness/dry-run.md`); the instrument is W10.3 | agent |
| RK-13 | One wrong reading (a NaN, an extreme value) poisons the whole field | low | high | go-ionomaps R-3.3 (physical ranges, no NaN or Inf out) | agent |
| RK-14 | A blind operator's screen reader is mixed into the broadcast | medium | high | stated plainly (FR-2.2, D-85); the device Setting is F-211 | HUM LEAD (D-85) |
| RK-15 | **NOAA feeds on the main path:** GloTEC, D-RAP, the scales and the daily solar indices; the climatology's F10.7 (D-104) now feeds every hour's M(3000)F2 (D-101) and every hour ahead (D-109) | low | medium | each missing feed is said with its age (FR-5.2); the last F10.7 mean held is used with its age (R-9.5); the host set is fixed (D-113) | agent |

## Metrics of success

Normative in `problem-statement.md`:
- M1 and M2 (PS-1), M3 retired (D-76);
- M4, M5 and M5b (PS-2);
- G1, shared;
- G-M1 to G-M4 and G-G1 (go-ionomaps).

**"Open"** means the reference circuit (D-73). M1 and M4 are scored against WSPR spots normalised to it,
with RBN as a cross-check (D-22). Their baseline is the IRI climatology (D-23). **Floors are set before the
dry run (D-75):** the current hour (the hybrid, D-101) must beat the climatology in the US, and the hours
ahead must be no worse than it (met by construction since D-109). Every other target is set from the dry run, each by its own ruling (D-49), except M3b and M3c's, which
D-48 set and D-76 retired.
