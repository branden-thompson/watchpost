---
title: "0.19.0 — Propagation overlays — IMPLEMENTATION PLAN"
date: 2026-10-06
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "DRAFT for the PLAN gate. No code: signatures, shapes, test descriptions, file paths and order only (0.18.0 D-13)."
---

# Implementation plan

**Goal.** Build `01-objectives/requirements.md` (approved at D-84), test-first, in the order D-4 and D-55
set, on go-tuiMaps v0.3.0 and go-ionomaps v0.1.0 release candidates, shipping on their tags (D-3).

**Architecture.** `03-architecture-design/architecture.md`:
- the Propagation mode in `modes/tty`;
- `app/mapprop.go` names the domain and what draws;
- `domains/propagation` holds the one go-ionomaps object and its client;
- `platform/geo` parses coordinates and locators.

**Branch.** `feature/propagation-overlays`, squash-merged `release/0.19.0` at SHIP (D-5).

**Every task follows the same order** (0.18.0's, kept):
1. Write the named test, and watch it fail for the named reason.
2. Make the change.
3. Run the whole touched packages with `-race`, then `make verify` (`make verify-docs` for Markdown only). Read MAKE_EXIT and the gate's verdict line before committing; the commit is chained to the verdict.
4. Mutation check: disable what the test protects, watch it fail, restore by copy (never `git checkout`).
5. A task that wires something names its path from the composition root, and `make wires` sees it.
6. The record moves before the gate: the build log entry and the as-built map are written before `make verify` runs (REFLECT L5).

**Human-graded instruments.** M2, M5 and M5b are graded by the HUM LEAD alone (D-68). Their protocol is in
this plan's "UAT" section.

## Work packages

| WP | What | Requirements | Needs | Order |
|---|---|---|---|---|
| W0 | Foundations: the pre-push hook, the propagation client, the shared coordinate parser | FR-10.2, FR-4.4, FR-4.7, FR-4.9 | — | first |
| W1 | #27: the roll-up never erases a year | FR-6.5 | — | **first in BUILD (D-55)** |
| W2 | The Propagation mode's shell: the mode, its bound, opening anywhere, its keys, the acknowledgement | FR-1.1 to FR-1.4, FR-1.12, FR-1.14, FR-11 | W0 | O1 |
| W3 | The maps: the two layers over water, terminator, legend, words seam | FR-1.5, FR-1.6, FR-3.3 | W2, **go-tuiMaps v0.3.0 rc** | O1 |
| W4 | The data path: `domains/propagation`, `app/mapprop.go`, fetched only when open, honesty | FR-4.1 to FR-4.3, FR-4.5, FR-4.6, FR-4.8, FR-5 | W2, **go-ionomaps rc** | O1 |
| W5 | The reference chart: best bands, your frequency, the centre, hours ahead, finding a place | FR-1.7 to FR-1.11, FR-1.13, FR-1.15 to FR-1.18 | W3, W4 | O1 |
| W6 | Words without the picture | FR-3.1, FR-3.2, FR-3.5 | W5 | O1 (ships with the layer, D-9) |
| W7 | The Broadcaster: the key into the mode, the shared-device statement | FR-2.1, FR-2.2 | W5 | O1 |
| W8 | Accessibility carry-over, gated before SHIP | FR-7.1 to FR-7.5 | — | O2 |
| W9 | Memo keys and the `say` fixes | FR-8, FR-9 | — | O3 (FR-9 earlier if a flaky gate blocks O1) |
| W10 | Instruments: the journey, the G1 harness, the code-quality brief | FR-10.3, NFR-1 | W5 | through BUILD |

## W0 — Foundations

| # | Task | Files | Shape | Test first |
|---|---|---|---|---|
| W0.1 | A pre-push hook runs the docs-reading tests (REFLECT L5) | `scripts/hooks/pre-push`, `Makefile` (`hooks`) | — | a watched failure: a planted stale as-built entry stops the push |
| W0.2 | The propagation client: its own instance, no retry on 429, status and headers returned on every outcome, no shared pacing hold, HTTPS only, no private addresses, per-source body caps | `platform/httpx/` | `func NewPropagationClient(cfg Config) *Client` (name settled in BUILD) | `TestThePropagationFetcherNeverRetriesA429` (a counting server), `TestA429ReturnsItsHeaders`, `TestAGIRORefusalNeverPausesWeather`, `TestThePropagationClientIsHardened` |
| W0.3 | One finite-checked coordinate parser and a Maidenhead parser, used by every resolver, `watchpost report` included; the geocoder's `name` redacted; no error quotes typed text | `platform/geo/coords.go`, `platform/geo/maidenhead.go`, `domains/locations/resolver.go`, `app/app.go:169` | `func ParseCoordinates(s string) (LatLon, bool)`, `func ParseLocator(s string) (LatLon, bool)` | `TestEveryResolverParsesCoordinatesLocally`, `TestNaNCoordinatesAreRefused`, `TestTheGeocoderQueryIsRedacted`, `TestCoordinatesAndLocatorsNeverGoOnline` |

## W1 — #27, first

| # | Task | Files | Test first |
|---|---|---|---|
| W1.1 | A failed read of a year's roll-up refuses the write | `platform/history/history.go:1478` | `TestAFailedYearReadNeverRewritesTheYear` (the reviewer's reproduction: a year past the cap) |
| W1.2 | Roll-ups split below the decompressed read cap, sized from bytes; a byte bound per dataset; values stored compactly | `platform/history/` | `TestARollUpNeverPassesTheReadCap`, a 90-day global-grid soak (synthetic), the existing history tests |

## W2 to W7 — the feature

| # | Task | Files | Test first |
|---|---|---|---|
| W2.1 | The Propagation mode beside Radar and Forecast, named in words in every path | `modes/tty/map_prop.go`, `map_temp.go:167` | `TestThePropagationModeIsAKeymapAction`, `TestTheModeIsNamedInEveryPath` |
| W2.2 | Its own world bound; weather modes keep theirs | `modes/tty/map_pane.go` | `TestEachModeKeepsItsOwnBound`, `TestNoFrameIsWiderThanTheRegion` (extended) |
| W2.3 | Opens anywhere; the outside message names the key | `modes/tty/map_pane.go:466-477` | `TestAPlaceOutsideEveryRegionOpensThePropagationMode`, `TestTheOutsideMessageNamesThePropagationKey` |
| W2.4 | Every control a keymap action; Help fits | `modes/tty/` | `TestEveryPropagationControlIsAnAction`, `TestHelpFitsWithThePropagationKeys` |
| W2.5 | The acknowledgement: before the first fetch, Enter/Esc, cursor, versioned record | `modes/tty/map_prop_ack.go`, `platform/config/` | `TestTheAcknowledgementShowsBeforeTheFirstFetch`, `TestOnlyEnterOrEscCloseIt`, `TestAChangedAcknowledgementShowsAgain`, `TestTheCursorIsPlacedOnTheAcknowledgement` |
| W3.1 | The MUF and foF2 layers over water, terminator, legend in MHz | `app/mapprop.go`, go-tuiMaps rc | `TestBothPropagationLayersDrawWithTheirUnits`, `TestAPropagationFieldCrossesTheSea`, `TestTheTerminatorIsDrawnForTheFieldsTime` |
| W3.2 | The description's seam before the alert-only returns | `modes/tty/map_describe.go:58-70` | `TestALayerAddsItsSentence` (four cases) |
| W4.1 | `domains/propagation`: one object per process, the propagation client, nothing before the acknowledgement or the mode | `domains/propagation/`, `app/mapprop.go`, `modes/tty/dashboard.go:72` (`MapPropagation`) | `TestOneLibraryObjectPerProcess`, `TestThePropagationModeFetchesNothingUntilOpened`, `TestNothingIsFetchedBeforeTheAcknowledgement`, `TestTheUpdateNeverRunsOnTheUIGoroutine` |
| W4.2 | Credits, MAP STATUS, rate headers only, nothing raw kept, 304s | `app/credits.go`, `app/maps.go:213-225` | `TestEveryRegisteredSourceHasACredit`, `TestMapStatusListsEveryPropagationHost`, `TestOnlyRateHeadersAreRead`, `TestNoRawSourceDataIsStored`, `TestAnUnchangedGridCostsA304` |
| W4.3 | Honesty: age, stale, failed, fallback named, forecast never a measurement | `app/mapprop.go`, `modes/tty/` | `TestTheFieldsAgeIsAlwaysSaid`, `TestAStaleFieldIsSaidStale`, `TestTheFallbackIsNamed`, `TestAForecastHourSaysItIsAForecast` |
| W5.1 | Best bands now (near-vertical 400 km or the service radius; targets; open hours today) | `app/mapprop.go`, `modes/tty/map_prop.go` | `TestBestBandsNameTheirLimit`, `TestTheDaysOpenHoursAreShown`, `TestTheContinentsAreTheDefaultTargets` |
| W5.2 | Your frequency: entry, parse, reach drawn and said | as above | `TestYourFrequencyDrawsItsReach`, `TestYourFrequencyIsSaid`, `TestABadFrequencyIsSaidNotGuessed` |
| W5.3 | Finding a place: the existing index, a mode-aware coverage gate, locators, the uncached online fallback, focus back | `domains/locations/resolver.go:101-102` | `TestAFoundPlaceIsAnsweredAndNotKept`, `TestALocalMatchSendsNothing`, `TestTheOnlineFallbackIsUncached`, `TestFocusReturnsAfterAFind` |
| W5.4 | The centre's answer; hours ahead on a key; answers never block | as above | `TestTheCentreIsAnsweredAsTheMapPans`, `TestTheHoursStepOnAKey`, `TestAnswersNeverBlockTheUI` |
| W5.5 | Space weather named: raised R, S or G scales and NOAA's outlook in the chart; forecast hours during a storm say so; a missing feed said (D-88) | `app/mapprop.go`, `modes/tty/map_prop.go` | `TestARaisedScaleIsNamed`, `TestAForecastHourDuringAStormSaysSo`, `TestMissingScalesAreNotShownAsZero` |
| W6.1 | The words, in order, units for speech, status without the picture | `modes/tty/map_describe.go` | `TestThePropagationWordsAnswerWithoutThePicture`, `TestPropagationWordsCarryUnitsForSpeech`, `TestTheStatusIsSaidWithoutThePicture` |
| W7.1 | The console key from the tower; the shared-device statement | `modes/tty/router.go:159-224`, `help_about.go` | `TestTheConsoleOpensThePropagationModeFromTheTower`, `TestTheConsoleKeysDoNotClash`, `TestTheSharedDeviceIsSaid` |

## W8 to W10

| # | Task | Test first |
|---|---|---|
| W8.1 | F-191 to F-194, F-196 to F-198, F-180, F-205, each as its follow-up row states | each row's named test |
| W8.2 | The SHIP gate over FR-7's rows | the check fails while a row lacks a passing test |
| W9.1 | #12's audit table; every position field becomes an identity or carries its reason; the app-side keys guarded | `TestEveryPositionFieldHasAReason`, `TestTheAppMemoKeysCoverTheirInputs` |
| W9.2 | `say`: no orphan on an orderly exit; orphans ended at start; `make test-say` bounded and diagnosed | `TestNoSayOutlivesAnOrderlyExit`, `TestAnOrphanedSayIsEndedAtStart`, the gate's bound |
| W10.1 | The scripted-PTY journey gains the Propagation mode and the console key | the journey on the real binary |
| W10.2 | The G1 harness: CPU, RSS and downloads a day over 48 hours, mode in use against not | the harness's own dry run on today's binary, then the measurement on a build with the mode |

## UAT (M2, M5, M5b)

Graded by the HUM LEAD alone (D-68). The protocol is `07-readiness/uat-protocol.md`:
- **M2:** ten reach questions from the station (the console key, then the chart);
- **M5:** ten band questions to targets from the Observer;
- **M5b:** ten more, drawn the same way, without the picture (A-25).

Each is scored against watchpost's own answer (D-92), with seconds; the WSPR key is reported beside it. The
scenarios are drawn in BUILD and fixed before the sitting, which is live (D-89).

## The trace

Every requirement row maps to its task above. `requirements.md`'s Instrument column is brought up to date
once, at BUILD exit, under one ruling row.
