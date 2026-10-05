---
title: "0.18.0 — BUILD-exit red team"
date: 2026-10-05
phase: BUILD exit
sev: SEV-0
authority: HUM LEAD
status: "ROUND 1 DISPOSITIONED (batch 137) — every finding fixed, ruled by the HUM LEAD (D-250 to D-263), or carried to 0.19.0 with its reason (F-191 to F-202)."
---

# BUILD-exit red team

Four blind reviewers, one per axis of `06_docs/red-team-brief.md`, reviewed `dd3f2a52..feature/map-drawing`
at `a81fe3e4` (batch 136): 189 commits, 804 files. Each ran what its axis needed in a scratch directory of
its own: a clean clone built, tested and linted green; `make lint-authoring` clean over 842 files;
`make p10` 0 live; `app` and `modes/tty` green under the race detector; the binary's `--help` and
`--version`; all 119 tests the requirements name as instruments exist.

**The brief named the wrong base** (`e66e0ef2`, v0.14.2). The business reviewer caught it; the others were
corrected mid-run and re-scoped. No finding depended on the wrong base. The harness refused the reviewers'
report files, so their reports live in this page's ledger, one row a finding, in their words' substance.

All four said **do not ship yet**. Their first fixes: the false all-clear (business), the README and
CHANGELOG (docs), the stale P10 mirror (hygiene, code), the docs lane passing an empty change (code).

## Dispositions

- **Fix** — changes no behaviour, UI or API: done under the clearance, in the batch named.
- **HUM** — the HUM LEAD's: asked one at a time, with a recommendation and a counter-argument.
- **0.19.0** — recorded in `06_docs/follow-ups.md` with its reason.

## Business quality (BQ)

| ID | Sev | Finding | Disposition |
|---|---|---|---|
| BQ-1 | Critical | An alert none of whose zones could be drawn adds no overlay, so the description says "No alert on the map covers or comes near" a place inside it; `InMissing` is set and never read on that path (`app/mapfeed.go:110`, `modes/tty/map_describe.go:70-79`). No 0-of-N scenario is tested. Confirmed in the code | HUM: **D-250, leave as is** (a known gap; the station's alert list names the alert) |
| BQ-2 | Important | FR-1.6, FR-3.2 and NFR-3 say maps off fetches nothing; the history recorder fetches hourly whatever the setting (D-172, D-234), and the requirements were not amended | HUM: **D-253, texts amended to D-172; no new UI** |
| BQ-3 | Important | Temperature, wind, rain, waves, UV, AQI, the history store, About and the Settings notices trace to UAT rulings, not FR rows | HUM: **D-254, the rulings stand as requirements** |
| BQ-4 | Important | M5 (≤ 3.5 s cold, p90) has no current result; live evidence is 7.0-10.1 s, n = 5, a different protocol. M6's thresholds (D-53) were never brought back | HUM: **D-251, measured in VALIDATE** |
| BQ-5 | Important | M1 and M1b have no scored record in `08-reports/` (FR-7.4) | HUM: **D-252, UAT stands for M1** (D-239) |
| BQ-6 | Important | The eight cuts (D-244 to D-247) have no follow-up rows; F-180 still says FR-5.8 adds a row; REQUIRED-READING is stale | Fix |
| BQ-7 | Minor | D-244's condition - the release notes name `--ascii` - has no 0.18.0 CHANGELOG entry to land in | Fix |
| BQ-8 | Minor | Eight history datasets with their own source archives are recorded and never read | HUM: **D-258, kept as ruled** (Analyst mode reads them) |
| BQ-9 | Minor | New layers were built while in-scope accessibility work was cut | HUM: **D-256** - 0.19.0 orders FR-7, FR-1.8 and motion before any new layer |

## Docs quality (DQ)

| ID | Sev | Finding | Disposition |
|---|---|---|---|
| DQ-1 | Critical | README has no map section and no `g`; CHANGELOG has no 0.18.0 entry and calls 0.17.0 unreleased | Fix |
| DQ-2 | Important | `docs/where-things-happen.md` and `docs/extending.md` do not cover the map | Fix |
| DQ-3 | Minor | The as-built page's status is ~1,000 words of batch history | Fix |
| DQ-4 | Minor | `gates.md` stops at batch 56; no P10 row | Fix (with HY-5a) |
| DQ-5 | Important | Zone seeding said to happen at start-up in two docs; it happens on the first map | Fix |
| DQ-6 | Important | As-built gives the cost warning's thresholds as 2 MB / 40; the code and FR-9.2 say 3 MB / 25 | Fix |
| DQ-7 | Important | As-built's "Not built yet" lists wind (built), line 101 puts the controls and legend over the map, a motion Setting is said built; F-182 says radar's next hours remain | Fix |
| DQ-8 | Important | REQUIRED-READING: BUILD open, rc.9, go-tuiMaps DISCOVER next, pan/zoom in phase 2, rulings D-1..D-56 | Fix |
| DQ-9 | Minor | FR-4.4's text says a line below the map names the missing zone; D-240 sent it to the diagnostics | Fix (text to the ruling) |
| DQ-10 | Minor | The UAT-1 guide teaches `L` and `map_alert_scope`, both retired | Fix (marked superseded) |
| DQ-11 | Minor | Stale statuses: the credits mock, the plan, the rulings' phase, CHANGELOG 0.17.0 | Fix |
| DQ-12 | Minor | The handoff flags its own wrong step rather than fixing it | Fix (marked closed) |
| DQ-13 | Important | D-245 and FR-9.6 say reduce motion forces playback off; watchpost never calls `ReduceMotion` | HUM: **D-255, the record corrected**; reduce motion to 0.19.0 |
| DQ-14 | Important | = BQ-6 | Fix |
| DQ-15 | Important | No M1b score, no BUILD-exit record | This page; M1b = BQ-5, D-252 |
| DQ-16 | Minor | "The radar in 1.5 s cold" rests on one run, no conditions | Fix |
| DQ-17 | Important | = BQ-4 | HUM: D-251 |
| DQ-Q4 | — | Audience order inverted: README empty on the map, REQUIRED-READING first, as-built opens on code | Fix A (the README section, DQ-1); fork B HUM: **D-260, as-built reordered**, the Instrument column kept |

## Project hygiene (HY)

| ID | Sev | Finding | Disposition |
|---|---|---|---|
| HY-1a | Minor | The public tag `checkpoint/pre-w14` is what `git describe` picks, so a source build reports `checkpoint/pre-w14-96-g…`; local tags hold rewritten blobs | HUM: **D-259, clean all** - the tag deleted, `--match 'v*'`; a work branch still describes from the last version tag that is its ancestor (v0.14.2: later releases were squash-merged), which tagged release builds do not meet |
| HY-1b | Minor | Stale branches: origin `release/v0.17.0`; local `main-publish`, `release/v0.17.0`, `feature/map-ready-data`; local `main` is the unsquashed history | HUM: **D-259** - origin's branch and the local stale refs deleted after a verified bundle outside the repo; local `main` reset at SHIP |
| HY-1c | Minor | The same radar PNG committed three times; another twice | Fix |
| HY-2a | Minor | README says the Go floor is 1.25; `go.mod` says 1.25.13 | Fix |
| HY-3a | Important | HEAD has no CI run; batch 135's run was red (`TestTheAsBuiltMapIsKeptInStep`: its build-log entry was written after its local verify) | Fix: push on the fix's green; a full two-OS run before `release/0.18.0` |
| HY-3b | Minor | `release.yml`'s 90-minute timeout rests on a 38-minute measurement; ubuntu now takes 45-58 | Fix |
| HY-4a | Important | The public P10 mirror holds 150 rows; the ledger 159 | Fix (regenerated by `make p10`) |
| HY-4b | Important | = BQ-6 | Fix |
| HY-4c | Important | RK-12 is Active with three of its five mitigations cut | HUM: **D-256, residual to 0.19.0** |
| HY-4d | Important | = BQ-7 | Fix |
| HY-4e | Minor | F-101 is moot after D-153; F-186 is closed and still listed | Fix |
| HY-5a | Important | The gate roster stops at batch 56: `test-say`, `lint-plan-code`, `property`, `verify-docs` and the structural tests have no row | Fix |
| HY-5b | Important | Batch 136 says "P10 clean" from `a2dh p10 check` run directly - not `make p10`, which regenerates the mirror; batch 135 does not say it landed red | Fix (the record corrected) |

## Code quality (CQ)

| ID | Sev | Finding | Disposition |
|---|---|---|---|
| CQ-1a | Important | `maptemp.go`'s header and `sourceFor` say Radar mode always uses Open-Meteo (D-96); the code follows D-190 | Fix |
| CQ-1b | Minor | `fetchAtOnce` and `maxAtOnce` (zones) mean different things | Fix: each comment says which is the concurrency and which the count. Not renamed: the ratified P10 row for the package (D-249) names `maxAtOnce`, and a rename would re-open it for a Minor naming point |
| CQ-2a | Important | The alert feed keeps its own reconcile, dropping a refused overlay where `reconcile` keeps the old one drawn (U2-14) | HUM: **D-257, keep last drawn**; `setFeed` through `reconcile` (batch 138) |
| CQ-2b | Important | `Series`' six parallel slices, inserted into by name; `hourIndex` repeats `hourRow` | 0.19.0 (a structural change across the temperature sources at the release's end; no defect today) - F-199 |
| CQ-2c | Minor | `viewBox` copies the library's projection, untested against it | Fix: `TestTheViewBoxIsTheLibrarysView` holds it to the library's `Report` at two zooms, corners and edges (three scale mutants caught); a view-bounds method from go-tuiMaps is F-202 |
| CQ-3a | Important | The zone store builds its own aged cache beside `agememo` | 0.19.0 (moving it changes coalescing and eviction - behaviour) - F-200 |
| CQ-3b | Minor | A registry for a closed list of one basemap | Kept: the seam P-5 asks for; the header says each layer registers from its own file and the one basemap where the map is built |
| CQ-4.1 | Important | `buildTemperature`, `withRainDays`, `withWaves` have no production caller and drop the `whole` flag nothing tests | Fix: the three move to a test helper (`app/whole_test.go`), production keeps the `*Whole` forms, and `TestTheWholeFlagSaysWhetherEverySourceAnswered` holds the flag (three mutants caught) |
| CQ-4.2 | Important | `radarLayerCost` never runs in production | Fix: deleted with its two constants; a nil cost now means nothing fetched of its own, so the radar, feels-like, wind and UV layers register none |
| CQ-4.3-4.11 | Minor | An unreachable `cost == nil` branch; an unread `removed`; an unread `ok`; `mapPane.calls`; `ChipKnown`, `SameOverlay` exported for tests; `AlertCategorySwitch` a rename; `HRRR.Name`, `httpx.Interactive`; `tempSources.rain`/`waves`; the zones recover comment | Fix, except as noted: the `cost == nil` branch is now reached (nil means none); `removed` and `take`'s `ok` deleted; `AlertCategorySwitch` is the one function; `rain`/`waves` are accessors (`openMeteo()`, `ndfdAPI()`); the recover comment names what it still guards. **Kept, with reason:** `mapPane.calls` (98 test uses read names, where `where` reads goroutines for W2.2); `ChipKnown`, `SameOverlay`, `httpx.Interactive` (called by `app`'s cross-package wiring tests, which an `export_test.go` seam cannot reach). **Not so:** `HRRR.Name` has a production caller (`app/mapradar.go`, `fetchForecast`) |
| CQ-4.12 | Minor | History's `Catalog`, `Extent`, `Days` have no production reader | HUM: **D-258, kept as ruled** |
| CQ-5 | Minor | `domains/temperature` holds rain, waves, wind, UV and the quota gate | 0.19.0 (a rename across the tree) - F-201 |
| CQ-6a | Minor | `app/maps.go:119` says the off switch arrives with W1.8 | Fix |
| CQ-6b | Minor | AirNow's `"ADT": -8` (AirNow's Alaska) and the -5 fallback uncommented | Fix: the comment, and `TestAlaskasRowsReadAlaskasToday` (an Atlantic-ADT mutant caught) |
| CQ-8a | Minor | `SetBound`'s error discarded silently | Fix: a refused bound goes to the diagnostics (`TestABoundTheLibraryRefusesGoesToTheDiagnostics`) |
| CQ-8b | Minor | The marine provider's and buoys' errors discarded; buoys vanish with no diagnostic (D-124) | Fix: the buoys' and the tide stations' failures go to the diagnostics (`TestTheSeasStationsThatDidNotAnswerGoToTheDiagnostics`) |
| CQ-9A | Important | The docs lane passes an empty change; `verify-docs` then says green whatever was committed | Fix (refuses an empty change) |
| CQ-9B | Important | `make p10` rewrites the mirror rather than failing on a stale one | Fix (fails when the committed mirror differs) |
| CQ-9C | Important | `property` and `test-say` select by `-run` with no check the pattern still matches | Fix (the guard `alloc-budget` has, for every `-run` gate) |
| CQ-9D | Minor | `test-say` is green on ubuntu by skipping | Fix (macOS-only step, declared; missing `say` fails under the leg) |

## Found while fixing

| ID | Sev | Finding | Disposition |
|---|---|---|---|
| FX-1 | Important | The Status window's disclosure said every host but the tiles, the NWS and CO-OPS "is asked for fixed regions or national files, never the view"; EPA is asked the largest cities in view, by name, for UV | HUM: **D-261** - the line is built from each source's `Sent`; `TestTheHostsSentTheViewAreSaid` |
| FX-2 | Minor | `g` with maps off sent the listener to WATCHPOST UI; the switch is on the Maps tab | HUM: **D-262** - "on the Maps tab" |
| FX-3 | Minor | W8.11's goldens of radar over alert areas were never built or ruled | HUM: **D-263** - built; four depths seen by the HUM LEAD and frozen |
| FX-4 | Minor | `tabsShown`'s comment said the console has no Maps tab; every tab is shown on both (D-70) | Fix |
