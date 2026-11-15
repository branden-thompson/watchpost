# Answer key, T1: a Forecast days map Setting, in Settings and in the Overlays menu

> **0.19.6 M4 answer key (D-24's method, JN P2-8), hidden from the blind reviewer.** Built on a scratch
> copy of `e57afeaf` (`git archive`, one commit `base`) by doing the task and leaving out one site at a
> time. Every run is in `<scratch>/plan-KB/runs.log`, its output in `<scratch>/plan-KB/logs/<label>.log`.
> Toolchain go1.27.4, `GOTOOLCHAIN=local`, no network. Lines "in the copy" are the finished change's.

Scratch copy: `<scratch>/plan-KB/t1` (base `afbcf18`). The complete change: `<scratch>/plan-KB/t1-complete.diff`;
every touched file: `<scratch>/plan-KB/t1-complete/`. The omission script: `<scratch>/plan-KB/bin/t1omit.py`
(each site's exact text), driven by `<scratch>/plan-KB/bin/t1loo.zsh`.

## The task, as given to the reviewer

> Add a map Setting, *Forecast days*, that chooses which of each day's temperatures Forecast mode draws:
> *Highs* (the default, what the map draws today) or *Lows*. Save it in `config.toml`. Show it as a picker
> on the Settings window's Maps tab and as a picker in the map's Overlays window (`O`); changing it in
> either place changes the same Setting, and the map shows the change at once. The `<` and `>` keys keep
> flipping Forecast mode between highs and lows for the moment without changing the Setting; each time
> the map opens, it starts from the Setting.

## The change that was made

- **The file:** `map_forecast_lows = true` (omitted when false), a `bool` like `reduce_motion`.
- **The model:** `mapForecastLows` in `mapChoices`, read from `Config.MapForecastLows`, saved through
  `UIPrefs.MapForecastLows`, written by the app's `setUIHook` and handed in by `ttyConfig`. Shaped as
  Walkthrough 6's map Setting, its model the quake feed (`MapQuakeFeed`).
- **Settings:** row `rowMapForecastDays`, a two-value picker (`kind: rowPicker, picker: true`, group
  `groupMap`, scope `scopeObserver` as every weather-map row), drawn `Forecast days -  ← Highs →` directly
  under `Temperature -` on the Maps tab, its chips "Days".
- **The map:** `toggleForecastDays` moves the Setting and sets what Forecast mode draws
  (`mapPane.fcLow`) to it, redrawing when the map is built and in Forecast mode; it ends with
  `uiTouched`, which marks the Setting for saving. `toggleMap` sets `mapPane.fcLow` from the Setting on
  every open. `<` and `>` (`flipHighLow`, unchanged) flip `mapPane.fcLow` alone.
- **The Overlays menu:** a new row kind `menuDays`, one row `Forecast days:  [←] Highs [→]` in the
  OVERLAYS section after the tints, in the weather modes (the Propagation mode returns before it, as it
  does for every weather group). It is the app's picker (`pickerCellW` through the menu's `choice`,
  D-147 of 0.18.0); ←→ and space move it, and the pressed chip blinks as every menu picker's does.
- **Docs:** README's Maps-tab list; a CHANGELOG entry.

Measured facts the change depends on:
- **What `<` `>` did before.** `mapPane` outlives a close (`toggleMap`, `modes/tty/map_pane.go:306-344`
  at `e57afeaf`, builds the map only when `mapPane.m` is nil), and nothing reset `fcLow` on open, so a
  flip survived a close and reopen. After the change every open starts from the Setting, as the task asks.
- **Where the Overlays menu saves.** It writes through `uiForSave` with the Settings (`setup_ui.go:144-147`:
  "whichever window saves them (Settings on close, the map's Overlays menu)"), so the menu row needs no
  save of its own: `uiDirty` is enough.
- **The row renumbers.** `rowMapForecastDays` sits after `rowMapTempSource` in `setupRowID`, because the
  list's order is the focus order (`setup_rows.go:141-142`); every later row's number moves by one, which
  is most of `setup-arrows.golden`'s and `router-trace.golden`'s diffs.
- **The goldens that move** (from `t1-before-goldens`, the change with every code site and no
  re-recording): `model-from-config`, `prefs-saved`, `setup-arrows`, `setup-133x44-maps` and
  `router-trace`. Re-recorded with `go test ./modes/tty -run '<their five tests>' -update-golden`.
  Diffs read: `mapForecastLows=false/true` lines; the two prefs lines; every Maps-tab row's frame hash (from
  row 29, the tab gaining a line) and the new row 39, the rows after it renumbered; the Maps tab gains the row under Temperature and its frame one line; the trace's
  Settings frame hashes. Nothing else.

## Every site the change needed

"Base guard" is what fails when the site is left out **and the change's own test edits are in**;
the leave-one-out table says which of those guards exist at `e57afeaf` (so catch a fresh change) and
which only the change's own new or edited tests supply.

| # | File | Symbol | What changes | Line in the copy |
|---|---|---|---|---|
| S1 | `platform/config/config.go` | `Config.MapForecastLows` | `bool` with `toml:"map_forecast_lows,omitempty"` | 339 |
| S1b | same | the field's comment | what true and false mean, which is the default (Walkthrough 6 step 1 asks for it) | 336-338 |
| S2a | `modes/tty/dashboard.go` | `Config.MapForecastLows` | the field the model is built with | 130-132 |
| S2b | same | `UIPrefs.MapForecastLows` | the field Settings saves | 466 |
| S3a | `app/dashboard.go` | `(*livePipelines).ttyConfig` | `c.MapForecastLows = cfg.MapForecastLows` | 448 |
| S3b | `app/themes.go` | `setUIHook` | `cfg.MapForecastLows = p.MapForecastLows` inside `config.Mutate` | 105 |
| S4a | `modes/tty/dashboard_choices.go` | `mapChoices` | `mapForecastLows bool` | 40 |
| S4b | same | `mapChoicesOf` | `mapForecastLows: cfg.MapForecastLows,` | 67 |
| S5a | `modes/tty/setup_ui.go` | `uiForSave` | `MapForecastLows: d.mapForecastLows,` | 169 |
| S5b | same | `applyUISaved` | the mirror into `d.cfg` | 211 |
| S6a | `modes/tty/setup_rows.go` | `setupRowID` | `rowMapForecastDays` right after `rowMapTempSource` (the drawn order) | 147 |
| S6b | same | `setupTable` | `{group: groupMap, scope: scopeObserver, kind: rowPicker, picker: true}` | 269 |
| S7 | `modes/tty/setup_layout.go` | `pickerChipWords` | `rowMapForecastDays: "Days",` | 378 |
| S8 | `modes/tty/map_settings.go` | `mapSettingLines` | `mapRow(... "Forecast days -", mapPicker(... daysLabel(d.mapForecastLows)))` under Temperature | 152 |
| S9 | `modes/tty/map_prefs.go` | `mapPrefArrow` | `case rowMapForecastDays: return d.toggleForecastDays(), true` | 607-608 |
| S10a | `modes/tty/map_temp.go` | `daysLabel` (new) | "Highs" / "Lows" | 538-544 |
| S10b | same | `toggleForecastDays` (new), beside `flipHighLow` | moves the Setting, and: | 546-557 |
| S10i | | | sets `mapPane.fcLow` to it, so what is drawn follows at once | 552 |
| S10ii | | | `setTemp().renderMap()` when the map is built and in Forecast mode | 553-555 |
| S10iii | | | ends with `d.uiTouched()` (marks for saving) | 556 |
| S11 | `modes/tty/map_pane.go` | `toggleMap` | `d.mapPane.fcLow = d.mapForecastLows` after `resetPlayback`, on every open | 333 |
| S12a | `modes/tty/map_boxes.go` | `rowKind` | `menuDays` | 545 |
| S12b | same | `overlayRows` | the row, after the tints, past the Propagation mode's early return | 591 |
| S12c | same | `menuRowLine` | `case menuDays`: `menuLine(... r.label+":", choice(daysLabel(...)))`, the app's picker | 763-764 |
| S12d | same | `hasChoice` | `|| r.kind == menuDays` (the pressed chip blinks, D-147 of 0.18.0) | 862 |
| S12e | same | `switchRow` (space) | `case menuDays: return d.toggleForecastDays()` | 963-964 |
| S12f | same | `chooseOnRow` (←→) | the same case | 997-998 |
| S12g | same | `forecastDaysKey` (new const) | the row's key, `"forecast-days"` | 865-866 |
| Ta | `modes/tty/config_reach_test.go` | `everySettingChanged` | `MapForecastLows: true` | 20 |
| Tb | `modes/tty/settings_kept_test.go` | `keptPrefs` | `MapForecastLows: true` | 31 |
| Tc | same | the `Config` literal of `TestASettingIsSavedAsItIsWhateverSavesIt` | `MapForecastLows: want.MapForecastLows` | 60 |
| Td | `app/settings_kept_test.go` | `want` in `TestASavedSettingIsTheSettingAfterARestart` | `MapForecastLows: true` | 24 |
| Te | same | its `back` map | `"MapForecastLows": got.MapForecastLows` | 48 |
| Tf | `modes/tty/map_menu_test.go` | `TestTheMenuIsTheHUMLEADsLayout` | `forecast-days` in its hand-written row list | 60 |
| Tg | same | `TestEveryBoxIsDrawnOnce` | `menuDays` joins the kinds that take a line of their own | 216 |
| Th | `modes/tty/map_forecast_days_test.go` (new) | `TestForecastDaysIsASetting`, `TestTheMapOpensItsDaysByTheSetting`, `TestTheOverlaysMenuMovesForecastDays` | the Settings row and its save; the open and `<`; the menu row and its save mark | 17, 41, 70 |
| G1-G5 | `modes/tty/testdata/` | `model-from-config`, `prefs-saved`, `setup-arrows`, `setup-133x44-maps`, `router-trace` | re-recorded with `-update-golden` | |
| S12h | `modes/tty/map_pane.go` | `overlaysMenuKey` | **added after the blind trace (M4 before):** its change test (`:1068`) compares only `mapLayerChoice`, `mapDetailChoice` and `mapDetailLevel`, so a menu move of the new Setting returns before `menuApplied` and is never saved by the menu (`uiApplyCmd`, `:1085`). The Setting joins the comparison. The key builder missed it; the reviewer found it by reading the caller of `handleOverlaysKey`. Not run by leave-one-out | 1068 |
| D1 | `README.md` | the Maps tab's MAP list | the Setting, also in the Overlays menu | 246-248 |
| D2 | `CHANGELOG.md` | a new `## [Unreleased]` / `### Added` entry | | 3-9 |

**Counts: 44 rows with S12h (43 as built): 28 code sites (in 13 source files across `platform/config`, `app` and `modes/tty`),
8 test sites, 5 goldens, 2 docs. NO GUARD for a fresh change: 14 (S1b, S5b, S10i, S10ii, S11, S12b,
S12c, S12d, S12e, S12f, Ta, Th, D1, D2).** The whole Overlays-menu half (S12b to S12f) is unguarded at
`e57afeaf`.

## Leave-one-out

Each row: that one site undone in the finished copy (`bin/t1omit.py <site>`), `go vet ./modes/tty
./app ./platform/config`, then, if it compiles, the site's package tested whole; the tree is restored
from `t1-complete/` and compared with the complete diff ("restored OK" after every row in `runs.log`).
A site in `modes/tty` was run against `modes/tty` alone and a site in `app` against `app` alone
(`modes/tty` does not import `app`; that an `app` test catches a `modes/tty`-only omission was not
measured). New tests: `map_forecast_days_test.go`, the change's own. Goldens: G1-G5.

| Site | What fails, by name | Base guard at `e57afeaf`? | Run |
|---|---|---|---|
| S1 | compiler: `app/dashboard.go:448 cfg.MapForecastLows undefined` | yes | `t1-omit-S1` |
| S1b | NO GUARD (a comment; `make lint-authoring` not run) | **NO** | none |
| S2a | compiler: `dashboard_choices.go:67`, `setup_ui.go:211` | yes | `t1-omit-S2a` |
| S2b | compiler: `setup_ui.go:169 unknown field MapForecastLows in struct literal of type UIPrefs` | yes | `t1-omit-S2b` |
| S3a | `TestASavedSettingIsTheSettingAfterARestart` (app): `MapForecastLows came back as false; it was saved as true` | yes (with Td, Te) | `t1-omit-S3a` |
| S3b | `TestASavedSettingIsTheSettingAfterARestart` (app) | yes | `t1-omit-S3b` |
| S4a | compiler: `dashboard_choices.go:66`, `map_boxes.go:764`, `map_pane.go:333` | yes | `t1-omit-S4a` |
| S4b | `TestASettingIsSavedAsItIsWhateverSavesIt` (`MapForecastLows was saved as false; on file it is true`); goldens G1, G2; new `TestForecastDaysIsASetting` | yes (with Tb, Tc) | `t1-omit-S4b` |
| S5a | `TestASettingIsSavedAsItIsWhateverSavesIt`; G2; new test | yes | `t1-omit-S5a` |
| S5b | nothing | **NO** (as Walkthrough 6 step 5 says) | `t1-omit-S5b` |
| S6a | compiler: `setup_layout.go:378`, `map_prefs.go:607`, `map_settings.go:152 undefined: rowMapForecastDays` | yes | `t1-omit-S6a` |
| S6b | `TestEverySettingsRowIsRuledForItsSurface` (`row 39 … nobody has ruled which surface it belongs to`), `TestEveryChipWordIsRead` (`"Days" for row 39, which is not a picker`), `TestSettingsFocusLineFollowsTheBalancedLayout`; G3, G5; new test | yes | `t1-omit-S6b` |
| S7 | `TestEveryPickerNamesWhatItsArrowsChange` (`picker row 39 (group 7) has no words in pickerChipWords`); G3 | yes | `t1-omit-S7` |
| S8 | `TestEveryArrowRowChangesSomething` (`row 39 (3) does not change on left`), `TestDownWalksAGroupAsItIsDrawn` (`↓ from row 38 (line 13) went to row 39, drawn above it`), `TestSettingsFocusLineFollowsTheBalancedLayout`, `TestSetupGoldenEveryTab` (G4); G3, G5; new test | yes | `t1-omit-S8` |
| S9 | `TestEveryArrowRowChangesSomething`, `TestAnArrowOnASavedSettingMarksItForSaving`; G3; new test. (Without the case, `mapPrefArrow` falls to `toggleDetailRow`, `map_prefs.go:630` at `e57afeaf`) | yes | `t1-omit-S9` |
| S10a, S10b, S12a, S12g | compiler: each is referenced by a later site (`map_settings.go:152`, `map_prefs.go:608`, `map_boxes.go:591`, `:764`) | yes | by reading, not run |
| S10i | new `TestForecastDaysIsASetting`, `TestTheOverlaysMenuMovesForecastDays` only: the Setting moves, what is drawn stays until the next open | **NO** | `t1-omit-S10i` |
| S10ii | nothing: no test looks at the map redrawn after the menu's move in Forecast mode | **NO** | `t1-omit-S10ii` |
| S10iii | `TestAnArrowOnASavedSettingMarksItForSaving` (`row 39 moved on left and is not marked for saving`), `TestEveryArrowRowChangesSomething`; G3; new tests | yes for Settings; the menu's save mark only by the new test | `t1-omit-S10iii` |
| S11 | new `TestTheMapOpensItsDaysByTheSetting` only (`with lows chosen, the map opened on the days' highs`) | **NO** | `t1-omit-S11` |
| S12b | `TestTheMenuIsTheHUMLEADsLayout`, but only as edited by Tf; new test. With the test as at `e57afeaf` a missing row passes it | **NO** | `t1-omit-S12b` |
| S12c | new `TestTheOverlaysMenuMovesForecastDays` only. Without the case the row is drawn as a checkbox, `[ ] Forecast days`; `TestEveryBoxIsDrawnOnce` as at `e57afeaf` counts it as a box drawn once and passes | **NO** | `t1-omit-S12c` |
| S12d | nothing: ←→ still move the row, only the pressed chip's blink (0.18.0 D-147's feedback) is lost | **NO** | `t1-omit-S12d` |
| S12e | nothing. Worse than silence: `switchRow`'s fall-through (`map_boxes.go:958` at `e57afeaf`) treats the row as a detail switch, so space writes `forecast-days=on` into the map's detail word and marks it for saving | **NO** | `t1-omit-S12e` |
| S12f | new `TestTheOverlaysMenuMovesForecastDays` only | **NO** | `t1-omit-S12f` |
| Ta | `TestEveryConfigFieldReachesTheModel` (G1), `TestThePrefsSavedAreTheSettingsInEffect` (G2): golden only | **NO** for a fresh change | `t1-omit-Ta` |
| Tb | `TestASettingIsSavedAsItIsWhateverSavesIt`: `UIPrefs.MapForecastLows has no value other than its default here` (the derived check) | yes | `t1-omit-Tb` |
| Tc | `TestASettingIsSavedAsItIsWhateverSavesIt` | yes | `t1-omit-Tc` |
| Td | `TestASavedSettingIsTheSettingAfterARestart` (app), the derived check | yes | `t1-omit-Td` |
| Te | `TestASavedSettingIsTheSettingAfterARestart` (app): `not handed back to the window here` | yes | `t1-omit-Te` |
| Tf | `TestTheMenuIsTheHUMLEADsLayout`: `the rows are temperature uv air forecast-days …` | yes | `t1-omit-Tf` |
| Tg | `TestEveryBoxIsDrawnOnce`: `the box "Forecast days" is drawn 0 times` | yes | `t1-omit-Tg` |
| Th | nothing: the whole package passes with every production site and no new test | **NO** | `t1-omit-Th-newtests` |
| G1-G5 | each its own test: `TestEveryConfigFieldReachesTheModel`, `TestThePrefsSavedAreTheSettingsInEffect`, `TestEveryArrowRowsOutcomeIsPinned`, `TestSetupGoldenEveryTab`, `TestRouterUpdateIsTheSameUnderRandomMessages` (the change before re-recording) | yes, but a golden re-recorded without its site records the wrong frame | `t1-before-goldens` |
| D1 | nothing reads README's list (`go test ./cmd/watchpost` passes with it, `t1-complete-cmd`) | **NO** | none |
| D2 | nothing reads `CHANGELOG.md` | **NO** | none |

The complete change: `modes/tty` **ok** (`t1-complete-tty`), `app` and `platform/config` **ok**
(`t1-complete-app`), `cmd/watchpost` **ok**, including `TestExtendingNamesRealSymbols` and the doc-symbol
tests (`t1-complete-cmd`).

## The docs that route a junior to each site today (`e57afeaf`)

| Sites | Route | Gap |
|---|---|---|
| S1, S1b | Walkthrough 6 step 1 (`docs/extending.md:181`) | routed |
| S2a, S2b | step 2 (`:182`) | routed |
| S3a, S3b | step 3 (`:183`), files and symbols | routed |
| S4a, S4b | step 4 (`:184`): `mapChoices`, `mapChoicesOf` | routed |
| S5a, S5b | step 5 (`:185`), "the mirror: nothing" fails | routed |
| S6a, S6b | step 6 (`:186`); its scope "is ruled, not chosen" (`:195-197`) | routed; the scope needs a ruling |
| S7 | step 7 (`:187`) | routed |
| S8 | step 8 (`:188`): "a map Setting is a `mapRow` in `modes/tty/map_settings.go`, in the order the tab shows it"; example "the map's quake feed" (`:174`) | routed |
| S9 | step 9 (`:189`): "a case in `mapPrefArrow` (`modes/tty/map_prefs.go`) calling its own `cycle…`" | routed |
| S10 (all) | step 9 says the `cycle…` "ends with `d.uiTouched()`" and, for a UI Setting, goes "beside the state it moves"; it does not say where a map Setting's goes, nor that a Setting the map draws by must also move what is drawn (S10i, S10ii) | partly; **NONE** for S10i, S10ii |
| S11 | Walkthrough 6 never covers a Setting applied when the map opens. `docs/where-things-happen.md:24` names `modes/tty/map_pane.go:toggleMap` for "The map opens" | **NONE** in a walkthrough; the flow map names the function |
| S12a-S12g | No walkthrough for an Overlays-menu row. `docs/where-things-happen.md:25` names `map_boxes.go:handleOverlaysKey` (the menu's keys) and `:29` `map_boxes.go:overlayRows` (in a sentence about layers, which Walkthrough 5 says need no edit there). Nothing names `rowKind`, `menuRowLine`, `hasChoice`, `switchRow` or `chooseOnRow`, nor the D-147 rule that the row must be the app's picker: "D-147" appears in no Start-here document (grep of `docs/`, `06_docs/code-standards.md`, `README.md`: none) | **NONE** |
| Ta-Te | step 10 (`:190`), each fixture named | routed |
| Tf, Tg | none; the failing tests name themselves (`map_menu_test.go` is in no doc) | NONE (the guard routes) |
| Th | none for the menu or the open; step 10 asks for the fixtures only | **NONE** |
| G1-G5 | step 10 (`:190`) lists `model-from-config`, `prefs-saved`, `setup-arrows`, `setup-133x44-ui`, `router-trace`. For a Maps-tab row the golden that moves is `setup-133x44-maps`, not `-ui` (measured) | routed, one name wrong |
| D1 | none | **NONE** |
| D2 | Walkthrough 8 step 7 (`:242`), the console's | NONE for this task |

**What the task costs (P2-6), at `e57afeaf`:** 13 source files and 5 test files edited, 5 goldens, 2
docs; 3 packages (`platform/config`, `app`, `modes/tty`). After 0.19.6 the same sites span
`platform/config`, `app`, the `modes/tty` root (`Config`, `UIPrefs`, `setupRowID`, `setupTable`,
`pickerChipWords`, `uiForSave`) and the map package (`map_boxes.go`, `map_temp.go`, `map_pane.go`).
Which side `mapChoices`, `map_settings.go`'s rows and `mapPrefArrow` land on is MS's design:
UNVERIFIED here.

## Found along the way (at `e57afeaf`)

1. **Walkthrough 6 names the wrong golden for a map row.** `docs/extending.md:190` lists
   `setup-133x44-ui`; a Maps-tab row moves `setup-133x44-maps` (`TestSetupGoldenEveryTab`), and `-ui`
   does not move (`t1-before-goldens`).
2. **The Overlays menu has no walkthrough and no derived guard.** Its one layout test hand-lists the row
   keys (`TestTheMenuIsTheHUMLEADsLayout`, `modes/tty/map_menu_test.go:54-63`), so a missing row passes
   it and an added row fails it until the list is edited. The layout is the HUM LEAD's (0.18.0 D-141,
   U2-39, `06_docs/02_features/observer-maps/02-analysis/rulings.md:157`): a new row is a ruling to ask
   for, which no doc says.
3. **Two silent fall-throughs.** `switchRow` ends by treating any row it does not know as a detail
   switch (`map_boxes.go:958`), so a new row kind without its space case writes its key into the
   saved `map_detail` word (S12e, no guard). `mapPrefArrow` ends with `toggleDetailRow`
   (`map_prefs.go:630`); that one is caught by `TestEveryArrowRowChangesSomething`.
4. **D-147 is cited bare and resolves three ways.** The menu's picker comment says "(D-147)"
   (`map_boxes.go:736`); `06_docs/ids.md:1540-1542` has a 0.16.0, a 0.18.0 and a 0.19.0 row. The rule
   meant is 0.18.0's (reuse the app's picker), and it is stated in no Start-here document.
5. **`<` and `>` survived a close before.** `toggleMap` (`map_pane.go:306-344`) never reset `fcLow`, so
   a flip outlived a close and reopen. The task changes that, as it says; no doc described either.
6. **In Radar mode the menu row moves a Setting with nothing visible changing.** The row is drawn in
   both weather modes (the Settings row always); 0.18.0 U1-42 warns that "a switch that seems to do
   nothing reads as broken". Showing it in Forecast mode alone is a choice to ask about.
7. **README's Overlays-menu screenshot goes stale.** Its alt text lists the menu's groups
   (`README.md:210`); `cmd/watchpost`'s screenshot test matches by shape, not content (passed,
   `t1-complete-cmd`).
8. **The shared pickers live in Settings' row file** (`pickerCellW`, `newArrowChips`,
   `modes/tty/setup_rows.go:521-580`, JN-2). A junior asked to "use the D-147 controls" in the map menu
   finds them only through `menuRowLine`'s `choice` closure (`map_boxes.go:736`).
