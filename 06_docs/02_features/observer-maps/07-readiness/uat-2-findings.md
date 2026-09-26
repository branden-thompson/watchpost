---
title: "0.18.0 Observer maps — UAT-2 findings (radar, W8)"
date: 2026-09-26
phase: BUILD
sev: SEV-1
authority: HUM LEAD
status: "OPEN - radar (W8) under UAT; each finding numbered, dispositioned and closed by a batch"
---

# UAT-2 findings

UAT-2 opened with batch 19 (`e2cd29c0`), radar from MRMS and IEM. Findings keep UAT-1's shape: the HUM
LEAD's words, where they land, and the batch that closes them.

| # | Scenario | Finding | Disposition | Fixed in |
|---|---|---|---|---|
| U2-1 | S18 | HUM LEAD, 2026-09-26: "We should preload the radar before we show it - right now it's blinking / hiding and "feels" broken" | Fix now (D-85) | Batch 20 (D-85): `TestTheRadarIsShownWhole`, `TestALaterAskWaitsForTheLoop` |
| U2-2 | S18 | HUM LEAD, 2026-09-26: "I only see 1 frame of the radar, MRMS says loading consistently (not sure if the full load is resolving)". Diagnosed live: every refresh and settle superseded the whole-loop ask before it landed | Fix now (D-85) | Batch 20 (D-85): `TestALaterAskWaitsForTheLoop`; the loop drawn once prepared |
| U2-3 | S18 | HUM LEAD, 2026-09-26: "Radar mode likely needs some controls/visualization of the loop, so users know where in the loop we are" | Fix now (D-86) | Batch 20 (D-86): `TestTheTimelineShowsWhereTheLoopIs`, `TestThePlaybackKeysDriveTheLoop` |
| U2-4 | S2 | HUM LEAD, 2026-09-26: "Let's hide the Area Aert text modal by default. Every single time I've UAT'd I've hiddent it to get rid of it." | Fix now (D-87) | Batch 20 (D-87): `TestTheAreaAlertsBoxWaitsForA` |
| U2-5 | S18 | HUM LEAD, 2026-09-26 (second pass): "Radar has not loaded after 30+s - says Radar (MRMS) loading... I zoomed into the Chicago area" / "Tried it at New York City / Central New Jersy zoom level as well - stuck". Diagnosed live: the library refused each loop over its 6 MiB image budget (6.8 and 7.1 MB), and the window swallowed the refusal | Fix now (D-88) | Batch 21 (D-88): `TestARefusedLoopIsSaidNotLoading`, `TestTheLoopFitsTheBudget`, `TestTheRadarAsksForAndKeepsWhatFits`; live: Chicago, New York, a four-box view and the whole lower 48 each hold a 24-frame loop |
| U2-6 | S18 | HUM LEAD, 2026-09-26 (third pass): "let's clean up that control area" - a mock of the rows under the map, and: "1. \"May may experience performance issues at this zoom level.\" -> Bold yellow text 2. Remove the radar values from the legend PIP since we now have a color row legend under the map 3. [ MRMS ] matches the badge/chip in the upper right of the map" | Fix now (D-89) | Batch 22 (D-89): `TestTheRowsUnderTheMapAreTheMocks`, `TestTheColourRowSaysItsClassesWithoutColour`, `TestALoopStillPreparingSaysLoadingWithItsSource` |
| U2-7 | S8 | HUM LEAD, 2026-09-26 (fourth pass): "Alaska view doesn't full the whole map window in wider terminals ... so the map looks broken. Maps should always draw to \"fill\" their window - even it its \"out of area\"". Cause: go-tuiMaps draws nothing past the antimeridian | Fix now (D-90; go-tuiMaps) | go-tuiMaps rc.12 (L11.7, D-86): `TestTheWorldRepeatsAcrossTheAntimeridian`, `TestAnOverlayAcrossTheSeamIsDrawnWhole`; watchpost batch 23 on rc.12 (Alaska at 190 columns draws Chukotka west of 180°) |
| U2-8 | S8 | HUM LEAD, 2026-09-26: "Hawaii should give me the ability to zoom out a bit more while remaining in that region. This is a good test new since there's a hurricade in the area just south of the big island." | Fix now (D-91) | Batch 23 (D-91): `TestHawaiiAndTheCaribbeanHoldTheirWeather` |
| U2-9 | S8 | HUM LEAD, 2026-09-26: "Same with Puerto Rico / US Carribbean - I should be able to zoom out a couple of levels to see the wider meterological picture of the area while remaining in the same region." | Fix now (D-91) | Batch 23 (D-91): `TestHawaiiAndTheCaribbeanHoldTheirWeather` |
| U2-10 | S18 | HUM LEAD, 2026-09-26: "Control area looks great. Well done." and "Let's make the upper right badge a little bigger/more prominent ... RADAR DATA / [ MRMS ] / 20:41" | Fix now (D-92) | Batch 23 (D-92): `TestTheRadarBadgeIsThreeRows`; go-tuiMaps rc.12 (L11.8, D-87) `TestAHostCanTakeTheStampOver` |
