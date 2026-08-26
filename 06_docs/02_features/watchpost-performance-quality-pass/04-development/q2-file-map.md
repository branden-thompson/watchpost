# Q2 file map — proposal for HUM LEAD (no move happens before approval)

Plan §3 Q2: pure structure, one package per commit, each move certified by a `go/parser`
declaration-set test (R2-22) and `go build && go test ./...`; exemptions re-keyed in the same
commit; per-file "how this fits" headers; `docs/where-things-happen.md`, the ADRs and the
`architecture.md` corrections ride with it. Rides with Q3 into `v0.9.6`.

HUM LEAD's note (2026-08-26): "dashboard.go is a 2000+ line file … ensure if it can be sensibly broken
up to ease understanding and maintenance, that we consider it." This is that consideration, file by file,
with the rule that decides each cut: **a file is a responsibility a newcomer can name in three words**,
and nothing moves across a package boundary (that would be a seam, Q6).

## 1. `modes/tty/dashboard.go` — 3,386 lines, 178 functions → 13 files

| Target file | Responsibility (three words) | Today's ranges (DISCOVER L3-F1, +Q0 rows) | ≈ lines |
|---|---|---|---|
| `dashboard.go` | model, keys, Update | `Config`, `Stats`, `Dashboard`, messages, `NewDashboard`, `Init`, `Update`, `handleKey`, tick/viz ticks `:1–560` | 560 |
| `modal_location.go` | add / remove modals | `:556–640`, `:799–914` (search, type-ahead, remove confirm) | 200 |
| `modal_chooser.go` | theme + voice choosers | `:642–798` | 160 |
| `nav.go` | selection, sort, scroll | `:915–1048` (`syncRecentView`, `windowSize`, `compact`) | 135 |
| `view.go` | frame assembly, modal geometry | `View`, `modalWidth`, `floatModal`, overlay order `:1049–1157` | 110 |
| `detail.go` | Location Details modal | `:1158–1213`, `:1381–1510`, `:2140–2182` | 250 |
| `detail_fire.go` | FIRE rows | `:1214–1380` | 165 |
| `detail_marine.go` | marine rows (tides, buoys) | `:1511–1761` | 250 |
| `alerts.go` | alert modal + alert area | `:1762–1994`, `:2476–2549` | 310 |
| `status.go` | `[S]` API status | `statusLines`, `requestLines`, `fixedAge`, issues, `pipelineLine` `:1995–2139` | 150 |
| `body.go` | header, tables, empty states | `:2183–2475`, `:2557–2579` (`apiSummary`, `LocationTable` calls, `thousands`, `compass`, `displayCond`) | 320 |
| `radio_panel.go` | radio panel + marquee | `:2550–2892` (`radioClock`, `radioLines`, `vizActive`, volume flash) | 340 |
| `help_about.go` | help + about | `:2893–2977` | 85 |
| `setup.go` | Setup window | `:2978–3255` | 280 |
| *(stays in `dashboard.go`)* | `With*` / `apply*` builders | `:3256–3332` | 80 |

`text.go` from L3-F1 (shared string helpers) is **not** created: the helpers it would hold (`thousands`,
`displayCond`, `compass`) are the Q6 duplicates (`render.Thousands`, `geo.CompassIndex`,
`render.DisplayCondition`); moving them twice is churn. They stay in `body.go` until Q6 collapses them.

Test file `dashboard_test.go` (2,591 lines) mirrors the split: each test moves beside the code it pins
(`status_test.go`, `detail_test.go`, `alerts_test.go`, …); shared fixtures (`snap()`, `dash()`, `f64`,
`stripANSITest`) stay in `dashboard_test.go`. `bench_test.go`, `race_test.go`, `norace_test.go` stay.

## 2. `platform/render/render.go` — 1,199 lines, 66 functions → 5 files

| Target | Responsibility | Contents |
|---|---|---|
| `units.go` | units and formatting | `Units`, `Opts`, `Temp`, `Distance`, `TideHeight`, `Knots`, `Wind`, `HealthGlyph`, `LoadingDots` |
| `table.go` | the go-studs table | `LocationRow`, `DayCell`, `baseColumns`, `layout`, `tableGeom`, `rowMarks`, `clampCells`, `LocationTable` — **the only go-studs consumer**, isolated for Q4 |
| `sgr.go` | colour | `sgrRaw`, `Tint`, `TintRaw`, `TintDefault`, `WrapSGR` callers, `KeyCap`, `TitleGradient`, `FireMark` |
| `panel.go` | panels and overlay | `Panel`, `Block`, `Overlay`, `PadBetween` |
| `text.go` | width and plain text | `Width`, `PadTo`, `displayWidth`, `stripANSI`, `Plain`, `HumanBytes`, `WrapSegments` |

`theme.go` / `themes.go` stay as they are.

## 3. `app/dashboard.go` — 881 lines, 42 functions → 6 files

| Target | Responsibility | Contents |
|---|---|---|
| `dashboard.go` | composition root | `RunDashboard`, `livePipelines` and its methods, `attachRadio`, `newAssembler`, `newCoops`, `cacheDir`/`userCacheSubdir` |
| `pipelines.go` | publisher and pipelines | `publisher`, `pipeline`, `startPriority`, `recentPipeline`, `startRecent`, `startStaggered`, caps and delays |
| `themes.go` | theme files | `applyThemes`, `loadUserThemes`, `setThemeHook` |
| `resolve.go` | resolver hooks | `newResolver`, `resolveHook`, `suggestHook`, `deriveTag` |
| `debug.go` | debug server + timing | `startDebugProfiles`, `debugAddr`, `reportTiming` (the dump stays in `dump.go`) |
| `refs.go` | config ↔ ref | `refsFromConfig`, `configLocations`, `restoreRecent`, `seedRecent`, `fullyPopulated`, `toKeyMap` |

## 4. `app/radio.go` — 753 lines, 41 functions → 3 files

| Target | Responsibility | Contents |
|---|---|---|
| `radio.go` | deck lifecycle | `radioDeck`, `newRadioDeck`, `Tune`, `tuneList`, `followMount`, `Stop`, `SetVolume`, `SetMode`, `setMode`, `label`, `onStatus`, `startSynth`, `synthReason`, `noteDirectories` |
| `radio_queue.go` | watchlist queue and dwell | `:146–225`, `:574–587` (`advanceQueue`, `nextInQueue`, `chooseNearest`, `armDwell`, `stopDwell`, `liveDwell`) |
| `voices.go` | voices | `:286–348`, `:415–573` (`Voices`, `VoiceName`, `SetVoice`, `PreviewVoice`, `listVoices`, `parseSayVoices`, `voiceDir`) |

## 5. `domains/weather/nws/provider.go` — 768 lines, 32 functions → 5 files

| Target | Responsibility |
|---|---|
| `provider.go` | `Provider`, `New`, `ID`/`Domains`, `Fetch`, `SetLocations`, `CachedGrids`, `CountyUGC`, URL helpers |
| `points.go` | `gridInfo`, `resolve`, `resolvePoints`, `stationOrder`, `markPreferred` |
| `observations.go` | `fetchObs`, `stationObs`, `quantity`, `toSI`, `conditionCode`, `windFromText` |
| `forecast.go` | `fetchForecast`, `fetchHourly`, `fillDailyFromGrid`, `foldDaily` |
| `alerts.go` | `fetchAlerts`, `mapAlert`, zone helpers |

## 6. `platform/snapshot/assembler.go` — 664 lines, 28 functions → 3 files

| Target | Responsibility |
|---|---|
| `assembler.go` | the stateful assembler `:18–315` (`NewAssembler`, `SetLocations`, `Apply`, `Warn`, `Size`, `SetAttribution`, `SetInactive`, `ProviderStatus`, `FireFor`, `Snapshot`) |
| `merge_fire.go` | fire merge (`mergeFire`, `Incident` union) |
| `harmonize.go` | pure merge/harmonize (`harmonize`, `fillFrom`, `normalizeMarine`, `published`) — the density exemption then names this file |

## 7. Discipline (every commit)

1. `git mv`-free: files are created and functions moved verbatim; imports pruned; no signature, name or
   comment changes except the per-file header.
2. `go build ./... && go vet ./... && go test ./<package>/` after each file; the package's
   `TestDeclarationSetUnchanged` (new, `go/parser`) compares the sorted set of top-level declarations
   against a golden captured before the first move.
3. Exemptions re-keyed to the new file (same count); `make p10` at the end of each package.
4. Coverage ≥ baseline per package; `app` ≥ 35 % after the new pure-logic tables.
5. Docs in the same package commit: `docs/where-things-happen.md` rows for that package, `extending.md`
   symbol paths, `architecture.md` §4 lines.

## 8. What Q2 does **not** do

No behaviour change; no package boundary crossed; no helper deduplicated (Q6); no render optimisation
(Q3). If a move exposes a bug, it is recorded in the build log and fixed in its own commit after the
moves, never inside one.

**Decision requested:** approve the six maps (or amend names / groupings) — Q2 starts with
`modes/tty` on approval.
