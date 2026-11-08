# Answer key, T1: add an on/off toggle Setting ("Compact header")

> **0.19.5 M5 answer key (D-24), hidden from the blind reviewer.** Built on a scratch copy of the
> tree at batch 41 (`e25a9ea9`) by doing the task and leaving out one site at a time. Batch 42 acted on
> it: the doc findings below are fixed, and the guards it added are named in the build log
> (`TestEveryFetchKindHasADomain`, `TestAllAsksEveryFamily`, `TestEveryChipWordIsRead`,
> `TestReduceMotionsChipsNameIt`). The site lists stand as the key; a row's "NO GUARD" is as measured
> at batch 41. Scratch paths are written as `<scratch>/`.


Scratch copy: `.../q195/m5key/t1-toggle` (base commit `a9e86ea "base"`). The complete change, as a diff:
`.../q195/m5key/t1-toggle-complete.diff`. A copy of every touched file: `.../q195/m5key/t1-complete/`.
Omission logs, one per site: `.../q195/m5key/t1-omit/logs/<site>.log`.

The Setting: `compact_header = true|false` in `config.toml` (omitted when false), a row
"Compact header -  ← Off →" on the Settings window's **Watchpost UI** tab, directly under Reduce motion.
It is shaped as walkthrough 6 says: an on/off Setting drawn as a two-value picker
(`kind: rowPicker, picker: true`) with its own cycle function, like Reduce motion. It is stored and
restored only. Nothing reads it to change the header; that part is outside a Settings walkthrough.

Every guard below was checked by removing **that one site** from the finished change, running
`go vet` and then `go test -count=1 ./modes/tty ./app` whole, and putting the file back from the copy
(the tree was compared byte for byte with the complete diff after every run: all `RESTORED OK`).

## 1 and 2. Every site, and what fails without it

"Golden only" means the failure is a golden comparison, which fails only against a golden recorded
*with* the site. A junior who leaves the site out from the start and then runs `-update-golden`
(as step 10 tells them to) records the wrong output and **nothing fails**. Those rows count as
NO GUARD for a fresh change.

| # | File | Symbol | Change | Guard that fails when it is left out (verified) | Gist |
|---|---|---|---|---|---|
| S1 | `platform/config/config.go` | `Config.CompactHeader` | `CompactHeader bool \`toml:"compact_header,omitempty"\`` beside `ReduceMotion` | compiler (`go vet`/build of `app`) | `app/dashboard.go:378 cfg.CompactHeader undefined`; `app/themes.go:92` the same |
| S1b | same | the field's comment | says what true and false mean and which is the default (step 1 asks for it) | **NO GUARD** (all tests pass; `make lint-authoring` not run, see below) | - |
| S2a | `modes/tty/dashboard.go` | `Config.CompactHeader` | the field the model is built with | compiler | `dashboard_choices.go:85 cfg.CompactHeader undefined`; `setup_ui.go:223 d.cfg.CompactHeader undefined` |
| S2b | same | `UIPrefs.CompactHeader` | the field Settings saves | compiler | `setup_ui.go:166 unknown field CompactHeader in struct literal of type UIPrefs` |
| S3a | `app/dashboard.go` | `(*livePipelines).ttyConfig` | `CompactHeader: cfg.CompactHeader,` in the `tty.Config` literal | `TestASavedSettingIsTheSettingAfterARestart` (app) | `CompactHeader came back as false; it was saved as true` |
| S3b | `app/themes.go` | `setUIHook` | `cfg.CompactHeader = p.CompactHeader` inside `config.Mutate` | `TestASavedSettingIsTheSettingAfterARestart` (app) | the same message |
| S4a | `modes/tty/dashboard_choices.go` | `uiChoices` | the model field `compactHeader bool` | compiler | `unknown field compactHeader in struct literal of type uiChoices`, `d.compactHeader undefined` |
| S4b | same | `uiChoicesOf` | `compactHeader: cfg.CompactHeader,` | `TestASettingIsSavedAsItIsWhateverSavesIt`; also `TestEveryConfigFieldReachesTheModel`, `TestThePrefsSavedAreTheSettingsInEffect` (golden only) | `CompactHeader was saved as false; on file it is true`; `model-from-config.golden differs ... line 124`; `prefs-saved.golden differs ... line 2` |
| S5a | `modes/tty/setup_ui.go` | `uiForSave` | `CompactHeader: d.compactHeader,` | `TestASettingIsSavedAsItIsWhateverSavesIt`; also `TestThePrefsSavedAreTheSettingsInEffect` (golden only) | `CompactHeader was saved as false; on file it is true` |
| S5b | same | `applyUISaved` | `d.cfg.CompactHeader = v.prefs.CompactHeader` (the mirror into `d.cfg`) | **NO GUARD** (the doc says this too) | - |
| S6a | `modes/tty/setup_rows.go` | `setupRowID` const list | `rowCompactHeader` right after `rowReduceMotion` | compiler | `undefined: rowCompactHeader` (setup_layout.go, setup.go, setup_rows.go, setup_ui.go) |
| S6b | same | `setupTable` | `rowCompactHeader: {id: rowCompactHeader, group: groupUI, scope: scopeShared, kind: rowPicker, picker: true},` | `TestEverySettingsRowIsRuledForItsSurface`, plus `TestSetupFormNoKeyIsTheDefaultDataSet`, `TestSetupFormKeyMasksAndStoresIt`, `TestSetupFormShowsAStoredFIRMSKeyAndItsHealth`, `TestSettingsFocusLineFollowsTheBalancedLayout`, `TestEveryArrowRowsOutcomeIsPinned`, `TestRouterUpdateIsTheSameUnderRandomMessages` | `row 11 is in the Settings window and nobody has ruled which surface it belongs to`; with the row missing, its zero value is a Data-tab input row, so enter stops saving |
| S6c | same | the row's `picker: true` | marks it as a picker, so the chip row shows "←→ Header" | `TestEveryArrowRowsOutcomeIsPinned`, `TestRouterUpdateIsTheSameUnderRandomMessages`, **both golden only, so NO GUARD for a fresh change** | `setup-arrows.golden differs ... line 5` (the chip row's hash) |
| S6d | same | the row's `scope:` | `scopeShared` | `TestEverySettingsRowIsRuledForItsSurface` | `row 11 ... nobody has ruled which surface` |
| S6e | same | the row's `group:` | `groupUI` | `TestEveryArrowRowChangesSomething`, `TestSettingsFocusLineFollowsTheBalancedLayout` (+ 2 goldens) | `row 11 (3) does not change on left/right`; `the focus mark is not on line 1` |
| S7 | `modes/tty/setup_layout.go` | `pickerChipWords` | `rowCompactHeader: "Header",` | `TestEveryPickerNamesWhatItsArrowsChange` (+ 2 goldens) | `picker row 11 (group 1) has no words in pickerChipWords: its chips would offer a voice` |
| S8a | `modes/tty/setup_ui.go` | `uiLines` | append the row's line (`setupMark` + `settingLabel(render.PadTo("Compact header -", uiLabelW))` + `pickerCell(...)`) | `TestEveryArrowRowChangesSomething`, `TestSetupGoldenEveryTab`, `TestSettingsFocusLineFollowsTheBalancedLayout` (+ 2 goldens) | `row 11 (3) does not change on left`; `setup-133x44-ui.golden differs ... line 29`; `row 11 spans [17,17] of 17 lines` |
| S8b | same | `uiLabelW` | widened to `len("Compact header -")` (the new label is wider than "Reduce motion -") | `TestAGroupsPickersLineUp` (+ ui, arrows and router goldens) | `WATCHPOST UI: a ← at column 22, another at 23` |
| S8c | same | `uiLines`, the `at` bookkeeping | `if focus == rowCompactHeader { at = len(lines) }` | `TestSettingsFocusLineFollowsTheBalancedLayout` (+ router golden) | `two columns: row 11 - the focus mark is not on line 4` |
| S9 | `modes/tty/setup_ui.go` (new) | `cycleCompactHeader` | flips `d.compactHeader` and **returns `d.uiTouched()`** | for the `uiTouched` part: `TestAnArrowOnASavedSettingMarksItForSaving` (+ arrows golden) | `row 11 moved on left and is not marked for saving: closing Settings would lose it` |
| S9a | `modes/tty/setup.go` | `cycleFocused` | `case d.setup.focus == rowCompactHeader: return d.cycleCompactHeader(forward).flashed(forward), true`, after `rowReduceMotion`'s case | `TestEveryArrowRowChangesSomething`, `TestAnArrowOnASavedSettingMarksItForSaving` (+ arrows golden) | `row 11 (3) does not change on left/right` |
| S9b | same | the case's `.flashed(forward)` | lights the pressed arrow | `TestEveryArrowRowsOutcomeIsPinned`, **golden only, so NO GUARD for a fresh change** | `setup-arrows.golden differs ... line 5` (`flash=0`) |
| S10a | `modes/tty/config_reach_test.go` | `everySettingChanged` | `CompactHeader: true` | `TestEveryConfigFieldReachesTheModel`, `TestThePrefsSavedAreTheSettingsInEffect`, **both golden only, so NO GUARD for a fresh change** | `model-from-config.golden differs ... line 124`, `prefs-saved.golden differs ... line 2` |
| S10b | `modes/tty/settings_kept_test.go` | `keptPrefs` | `CompactHeader: true` | `TestASettingIsSavedAsItIsWhateverSavesIt` | `UIPrefs.CompactHeader has no value other than its default here: the guard cannot see it kept` |
| S10c | same | the `cfg := Config{...}` literal in `TestASettingIsSavedAsItIsWhateverSavesIt` | `CompactHeader: want.CompactHeader` | `TestASettingIsSavedAsItIsWhateverSavesIt` | `CompactHeader was saved as false; on file it is true` |
| S10d | `app/settings_kept_test.go` | `want` in `TestASavedSettingIsTheSettingAfterARestart` | `CompactHeader: true` | that test | `UIPrefs.CompactHeader has no value other than its default here` |
| S10e | same | the `back` map in the same test | `"CompactHeader": got.CompactHeader` | that test | `UIPrefs.CompactHeader is not handed back to the window here` |
| G1 | `modes/tty/testdata/model-from-config.golden` | re-recorded | `compactHeader=false` / `=true` lines | `TestEveryConfigFieldReachesTheModel` | `differs from its golden at line 18` |
| G2 | `modes/tty/testdata/prefs-saved.golden` | re-recorded | `CompactHeader:false` / `:true` | `TestThePrefsSavedAreTheSettingsInEffect` | `differs ... line 1` |
| G3 | `modes/tty/testdata/setup-arrows.golden` | re-recorded | two new lines (row 11), and **every later row renumbered** (+1) with every UI-tab frame hash changed: 174 lines of diff | `TestEveryArrowRowsOutcomeIsPinned` | `differs ... line 1` |
| G4 | `modes/tty/testdata/setup-133x44-ui.golden` | re-recorded | the new row, and Theme and Reduce motion move one column right (`uiLabelW`) | `TestSetupGoldenEveryTab` | `differs ... line 16` |
| G5 | `modes/tty/testdata/router-trace.golden` | re-recorded | Settings frame hashes in the random-message trace (292 lines of diff) | `TestRouterUpdateIsTheSameUnderRandomMessages` | `router-trace.golden differs ... line 47` |

The goldens were re-recorded with
`go test ./modes/tty -run '<those five tests>' -args -update-golden`.

**Counts:** 32 rows (27 code and test sites plus 5 goldens), in 12 source files and 5 goldens.
Sites with no guard at all: S1b, S5b. Sites guarded only by a golden, which is no guard for a
fresh change: S6c, S9b, S10a.

### Other shapes, tested as variants rather than omissions

| Variant | What fails (verified) | Meaning |
|---|---|---|
| V1: the row as `kind: rowToggle` (the task's word, and the kind Maps on and every map-detail row use) instead of `rowPicker` | only `setup-arrows.golden` and `router-trace.golden`, so **NO GUARD** | ←→ still work (the row's own case comes before the generic toggle case), but **space does nothing**. `rowToggle`'s contract (`setup_rows.go:51-52`) says "←→ or space flip it". `setupSpace` sends a `rowToggle` to `toggleClass`, which returns early for a row with no tone class. A probe test confirmed space left `compactHeader=false, uiDirty=false`. A real toggle would need a `setupSpace` case, which no doc mentions. |
| V2: the exact Reduce-motion shape, `kind: rowPicker` without `picker: true` (= S6c) | golden only | by reading `setupChips`: the chip row shows no ←→ at all, and the `pickerChipWords` entry is never read. Today `rowReduceMotion: "Motion"` (`setup_layout.go:374`) is a dead entry for this reason. |
| V3: `rowCompactHeader` declared *before* `rowReduceMotion` but drawn after it | golden only, so **NO GUARD** | by reading `stepRow` (not probed separately): ↓ jumps from the clock rows to Compact header (drawn below), skipping Reduce motion. `TestDownWalksAGroupAsItIsDrawn` walks only the Maps tab. |
| V4: the const added at the end of the list (after `rowMapLayers`) | `setup-arrows.golden` only | harmless: focus walks only the visible tab's rows, so the order on the tab is still right. |
| S9d: the `cycleFocused` case moved to the end of the switch | nothing fails | correct: with `picker: true` the row passes `!row.picker`. It is only the Reduce-motion shape (`picker` false) that must come before `case !row.picker`. |

### Gates not run (the task allows only `go test`, `go vet` and `go build`)

- `make lint-authoring` (`go run ./tools/authoring`): this is what would judge S1b's comment and the new
  comments (AP-HIST-01, AP-CITE-01). New comments name no ID, so AP-CITE-01 has nothing to judge.
- `make p10`: P10-04 on `setupTable`. The base has 59 code lines in the body, as
  `06_docs/code-standards.md:343-345` says. With this row it has **60, exactly the limit**, so the next row
  added fails. Counted by hand, not run. P10 is phase-exit only, and its checker is not in the public tree anyway.
- `make wires`: from reading `tools/wires/main.go:311`, it enrols only enums whose sentinel is named `num…`.
  `setupRowCount` is not, so `setupRowID` is not checked.
- `make mutant-anchors`: not run. Instead, a read-only script took the 41 literal anchors in mutants that
  edit the touched files and found each one still occurs exactly once (0 drifted).

## 3. Doc accuracy (`docs/extending.md`, walkthrough 6, lines 147-173)

| Line | Step | Routes to the right file and symbol? | Problem |
|---|---|---|---|
| 155-156 | intro | Partly | "On/off Settings are drawn as two-value pickers here" holds only for Reduce motion. Maps on (`rowMapsOn`), the nine map-detail rows and the tone rows are on/off rows of kind **`rowToggle`**, which the walkthrough never mentions. A junior told to add a "toggle" will reach for `rowToggle` and get a row space cannot flip, with no guard (V1). |
| 164 | 1 | Yes (`platform/config/config.go`) | Its guard is "the compiler, at step 3": true. The comment it asks for has no guard. |
| 165 | 2 | Yes (`Config`, `UIPrefs` in `modes/tty/dashboard.go`) | none |
| 166 | 3 | File yes, **symbol missing** | It names `app/dashboard.go` and `app/themes.go` but not the functions: `(*livePipelines).ttyConfig` and `setUIHook`. Its guard is right. |
| 167 | 4 | Yes (`uiChoices` / `uiChoicesOf`) | It lists two guards, which are golden only. The guard that actually catches a fresh omission is `TestASettingIsSavedAsItIsWhateverSavesIt`, which it does not list. |
| 168 | 5 | Yes (`uiForSave`, `applyUISaved`) | The mirror has no guard, as it says (confirmed). |
| 169 | 6 | Yes (`setupRowID`, `setupTable`) | (a) "`picker: true` for a picker" contradicts its own worked example: `rowReduceMotion` (`setup_rows.go:253`) is `kind: rowPicker` with no `picker: true`. (b) It does not say the const's **position is the focus order** and must match the drawn order (V3 has no guard). (c) It does not say that inserting the const in the middle renumbers every later row, which rewrites `setup-arrows.golden` almost entirely. (d) It never names the field values: `kind: rowPicker`, `group: groupUI`, `scope: scopeShared`. |
| 170 | 7 | Yes (`pickerChipWords`) | It is only true with `picker: true`. For the Reduce-motion shape the words are never read, and `TestEveryPickerNamesWhatItsArrowsChange` skips the row. The doc's own example entry, `rowReduceMotion: "Motion"`, is dead. |
| 171 | 8 | File yes, **symbols missing** | It says "drawn in `modes/tty/setup_ui.go`, beside Reduce motion" but names neither `uiLines` nor its `at` focus-line bookkeeping (guarded by `TestSettingsFocusLineFollowsTheBalancedLayout`) nor `uiLabelW` (which a longer label must widen, guarded by `TestAGroupsPickersLineUp`). It lists 2 guards; 3 more fire. |
| 172 | 9 | Yes (`cycleFocused`, `uiTouched`) | (a) It never says where the cycle function goes. Reduce motion's is `cycleReduceMotion` in `modes/tty/motion.go`, a file the walkthrough never names. (b) It does not mention `.flashed(forward)` (golden-only guard). (c) "whose golden gains the row's two lines" understates the change: every later row renumbers. (d) It does not mention `setupSpace` (`setup.go:514`), which a `rowToggle` row needs. |
| 173 | 10 | Yes (`everySettingChanged`, both `settings_kept_test.go`) | (a) Each settings-kept file needs **two** edits (app: `want` and the `back` map; tty: `keptPrefs` and the `cfg` literal). Missing any one of them is caught. (b) "re-record the goldens the new row changes (below)" sends the reader to a section that names none. Five goldens move, and **`router-trace.golden` (`TestRouterUpdateIsTheSameUnderRandomMessages`) appears nowhere in the doc**. |
| 185 vs 172 | goldens | Both forms work | `go test ./modes/tty -update-golden` (line 185) and `... -args -update-golden` (line 172) are both accepted. No error. |

Other docs read: README "Start here" (623-641), `docs/where-things-happen.md:56` and `:130`
(`setup_rows.go:setupTable` for the Settings layout: correct), and `06_docs/code-standards.md:341-349`
(the setupTable P10-04 specimen: accurate, and this change reaches the limit it describes).
`docs/architecture.md:104` correctly points to `docs/extending.md` for adding a Setting.

## 4. IDs met in the docs followed, against `06_docs/ids.md`

The index covers only the families in `tools/ids/idset/idset.go`: NFR, DEF, FR, DR, PF, RN, DC, JR, DV,
JM, RR, RS, OQ, D, A and F.

**Do not resolve:**
- `F-111`: `docs/extending.md:94`. It is in F's family but has no row. It appears only inside a table row
  whose first cell is `CQ-n` (`06_docs/02_features/0.16.0-broadcaster-ui/08-reports/red-team-build.md:87`),
  so the indexer never sees it defined.
- `0.15.0 FR-7.1`: `docs/where-things-happen.md:68`. The index has FR-7.1 only for 0.16.0, so the
  release named is wrong or unindexed.
- `R3-A-01` (where-things-happen:34), `MVS-D-3` (:54), `MVS-D-15` (:60) and `MVS-D-80` (:147;
  code-standards:225): release-prefixed IDs, which the index has no section for.
- `AP-HIST-01`, `AP-CITE-01`, `AP-CAPS-01`, `AP-DEAD-01`, `AP-SHELL-01`, `AP-OK-01`, `SN-02`, `SN-03`, and
  `P10-00` to `P10-10` (in code-standards.md, and where-things-happen:150 for AP-CITE-01). These are rule
  IDs. They are explained in `06_docs/code-standards.md` itself, but not in ids.md.

**Resolve, but ambiguously (a bare ID with several release rows):** in extending.md, `FR-1.1` (3 rows, L8),
`FR-1.2` (4 rows, L8), `D-15` (7 rows, L17; the intended one is watchpost-cli's "Keybindings
hot-swappable") and `D-61` (3 rows, L22; the intended one is 0.18.0's). Elsewhere: `FR-9` (9 rows),
`DR-21` (2), `D-124` (2), `D-10` (11), `D-160` (2), `D-56` (3) and `D-30` (4).

**Resolve uniquely, or match their named release:** `D-292` (0.19.0), `FR-3.8` (0.18.0),
`0.19.5 D-33` (extending.md:175: "p previews a voice only", correct), `F-21`, `FR-9.4`, `0.19.5 D-13`,
`0.19.5 D-4`, `0.19.5 D-11`, `0.19.5 D-10`, `0.19.5 D-34`, `0.19.5 D-30`, `JM-3`, `DC-14`,
`0.19.5 D-12`, `0.19.0 D-142`, `0.19.5 A-2` and `0.19.5 D-27`. Labels without a hyphen in extending.md
(`B5`, `M5`, `M2`, `M3`, `UAT 97`) are not ID-family shaped and the index does not cover them.

## Final run

With the complete change in place, `gofmt -l` is clean, `go vet ./modes/tty ./app ./platform/config`
is clean, and
`go test -count=1 ./modes/tty ./app ./platform/config ./cmd/watchpost` **passes**:
modes/tty 61.4s, app 136.0s, platform/config 1.5s, cmd/watchpost 79.0s. cmd/watchpost includes
`TestExtendingNamesRealSymbols`.
