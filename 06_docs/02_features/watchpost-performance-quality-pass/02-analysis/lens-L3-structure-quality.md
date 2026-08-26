# DISCOVER Lens L3 — Code structure, code quality, architectural quality

Read-only research lens, 2026-08-26. Evidence is `file:line` against the current tree; coverage from
`go test -cover ./...`; P10 from `a2dh p10 check` (0 live, 132 loaded, 56 outside third_party).

## 1. Package / file structure (all pure moves — same package, no logic touched)

| ID | File (lines) | Responsibilities found | Proposed files |
|---|---|---|---|
| L3-F1 | `modes/tty/dashboard.go` (3,332) | model/Config/msgs/keymap/Update `:1–560`; add/remove modals `:556–640, :799–914`; theme+voice `:642–798`; nav/sort `:915–1048`; View + modal geometry `:1049–1157`; detail `:1158–1213, :1381–1510, :2140–2182`; fire rows `:1214–1380`; marine rows `:1511–1761`; alert modal + area `:1762–1994, :2476–2549`; API status `:1995–2139`; header/body/table/empty `:2183–2475, :2557–2579`; radio panel `:2550–2892`; help/about `:2893–2977`; setup `:2978–3255`; apply*/With* `:3256–3332` | `dashboard.go`, `modal_location.go`, `modal_chooser.go`, `view.go`, `detail.go`, `detail_fire.go`, `detail_marine.go`, `alerts.go`, `status.go`, `radio_panel.go`, `help_about.go`, `setup.go`, `text.go` (shared string helpers); mirror the 2,537-line test file |
| L3-F2 | `platform/render/render.go` (1,188) | units/Opts; table (`LocationRow`, `layout`, `tableGeom`, `rowMarks`, `LocationTable`) `:141–760` — the only go-studs consumer; SGR/tint/keycaps; panels/overlay; text/width | `units.go`, `table.go` (go-studs coupling isolated), `sgr.go`, `panel.go`, `text.go` |
| L3-F3 | `app/dashboard.go` (797) | composition root; publisher/pipelines; themes; resolver; profiling; config↔ref mapping | `dashboard.go` (wiring), `pipelines.go`, `themes.go`, `resolve.go`, `debug.go`, `refs.go` |
| L3-F4 | `app/radio.go` (681) | deck lifecycle; watchlist queue/dwell `:146–225, :574–587`; voices `:286–348, :415–573` | `radio.go`, `radio_queue.go`, `voices.go` |
| L3-F5 | `domains/weather/nws/provider.go` (760) | points/resolve; observations; forecast/hourly/grid fill; alerts | `provider.go`, `points.go`, `observations.go`, `forecast.go`, `alerts.go` |
| L3-F6 | `platform/snapshot/assembler.go` (655) | stateful assembler `:18–315` vs ~270 lines of pure merge/harmonize | `assembler.go`, `merge_fire.go`, `harmonize.go` — the density exemption can then name the pure file |
| L3-F7 | — | Real seams distinct from moves: modal enum (F15), shared text helpers (F8–F10), caps (F11). No new package warranted | moves first, seams second, one commit each |

## 2. Duplication and single-owner knobs

| ID | Finding | Recommendation | Risk |
|---|---|---|---|
| L3-F8 | `thousands` byte-identical in `tty/dashboard.go:1355` and `synth/fire.go:226` | second caller reached → `render.Thousands` (platform is the legal shared owner for modes and domains) | low |
| L3-F9 | Three compass implementations: tty 16-pt abbreviations `:1735`, synth 16-pt words `fire.go:191`, synth **8-pt** words `compose.go:159` | `geo.CompassIndex(deg, points)` for the math; consumers keep word tables; 8-vs-16 for wind is a documented decision or an in-pass UX nit (golden change) | low / medium |
| L3-F10 | `displayCond` (tty `:2131`) and `displayCondition` (render `:667`) identical; `prettyCond` differs | export `render.DisplayCondition`, delete `displayCond` | low |
| L3-F11 | `recentCap = 50` in `app/dashboard.go:595` **and** `tty/dashboard.go:838`; watch cap `10` as a literal at `tty:581`, `tty:608`, `app/dashboard.go:388` | `tty.RecentCap`, `tty.WatchCap` (tty is where they act; app imports tty) | low |
| L3-F12 | Key-cap control rows hand-assembled in 9 places with inconsistent order (`:775, :795, :1202, :1898, :1906, :2014, :2907, :3163`, radio `:2831–2860`) | one `controlsRow(o, pairs…)` helper; fixes the ordering nit (OQ-6) | low |
| L3-F13 | `WrapSegments` callers each re-split on `\n` (`:2314, :2860, :3166`) | return `[]string` | low |
| L3-F14 | Cadence tables: one owner (app) but two literal tables + a const, re-typed in architecture §11.1 | table-driven test that prints them so the doc is generated | low |

## 3. Dashboard modal model

| ID | Finding | Recommendation | Risk |
|---|---|---|---|
| L3-F15 | Ten `show*` booleans; exclusivity maintained by hand at 10 reset sites and **already inconsistent**: `voiceErrMsg :338` and `applyCommitted :3275` reset 3 of 9; `remove :544` resets none; help `:523` closes only details. Four parallel switch chains (`handleKey :367`, `View :1057`, `modalWidth :1110`, `modalLines :1131`). **No test pins exclusivity** | `type modal int` + `d.modal`; keep per-modal payload fields; `open(kind)`/`close()`; write the exclusivity table test first | medium (mechanical, compiler-checked) |
| L3-F16 | `sortAlerts` sorts the published snapshot's alert slices in place (`:961`) — safe only because `Assembler.Snapshot()` deep-copies | document the owned-copy contract at the publisher, or sort in the assembler | low |

## 4. P10 exemptions (56 non-third-party)

| ID | Class | Count | Disposition |
|---|---|---|---|
| L3-F17 | **Stale**: `app/setup.go` `View`/`Update`/`Save` — symbols gone since UAT 100; the `app` density reason cites "RunSetup … 5 wizard tests" | 3 | delete; rewrite the reason → 53 |
| L3-F18 | (a) valid name-graph false positives (`Close`, `Read`×2, `NewPlayer`, `Samples`, `Rate`, `Now`, `After`, `Resolve`, `Stop`, `SetVoice`, `stopDwell`) | 12 | keep (ratified D-20 class) |
| L3-F19 | (b) density on pure parsers/renderers | 31 | keep; splits sharpen reasons |
| L3-F20 | (c) removable by a small refactor: `pkg/schema` `timeType` global (`schema.go:44`) → func; `render/theme.go:88` `theme` global → `defaultTheme()` folded into the registry entry | 2 | do → 51 |
| L3-F21 | (d) valid by design (`tz` sync.Map, `runTier` loop, schema reflect/recursion, themes registry, `installMu`, voice memo) | 8 | keep |

Net **56 → 51** without deleting a check (M4; RS-4 honoured).

## 5. Tests and coverage

| ID | Finding | Recommendation |
|---|---|---|
| L3-F22 | `app` 16 %: mostly wiring, but untested pure logic — `parseLatLon`, `resolveQuery`, `staleWarnings` (`app.go:86–143`), `fullyPopulated`, `toKeyMap`, `refsFromConfig/configLocations`, `radioDeck.synthReason/label/unrelayedLabel/stationFor/onStatus` | after F3/F4 the pure functions sit in their own files; table tests lift app to ~35–40 % |
| L3-F23 | `cmd/watchpost` 33 %: the `report` run path (JSON vs plain, exit 2) untested | one test through `report` with a fake `ReportOnce` |
| L3-F24 | No byte golden for the plain report (the R-12d screen-reader surface) | width-80 golden on the parity fixture |
| L3-F25 | A tty test pins a theme implementation value (`48;2;86;86;86`, `dashboard_test.go:1760`) | assert via `render.Tok` |
| L3-F26 | pty smokes are **not in CI** — manual HUM LEAD sessions | record it; list the command in the checklist |
| L3-F27 | `platform/invariant` 0 % (the P10 keystone); `tools/alertrec` 0 % | 4-case table test for `Check` |

## 6. Errors, invariants, context

- L3-F28: `invariant.Check` used consistently (30 files); `_ =` sites are Close/Remove in defers except
  `app/radio.go:465` (silent preview failure → route through `voiceNote`) and `app/dashboard.go:348,351`
  (`SetInactive` drops → invariants).
- L3-F29: contexts/timeouts complete on every network path; error messages actionable. No action.

## 7. Documentation for juniors

- L3-F30: `architecture.md` stale beyond §11.9: §4 says `modes/report` does not import `platform/render`
  (it does, for `render.Plain`); §4 lists primitives that don't exist (`Sparkline/Bars/WindGauge/Meter/Grid`);
  §1 still shows `platform/audio`. Package docs present everywhere except the three `main`s. Add an as-built
  import graph (from `go list`) and a per-package file map after the splits.

## Proposed ordering (value ÷ regression risk)

1. F17, F20 — stale exemptions + two one-liners: 56 → 51, no behaviour change.
2. F1–F6 pure file moves, one commit per package, with F30 file-map comments.
3. F27, F24, F25, F23 — cheap tests that raise the floor before any seam (TDD order).
4. F11, F8, F10, F13, F12 — caps and identical helpers.
5. F15 modal enum — exclusivity test first, then the refactor (fixes the latent stacking inconsistencies).
6. F22 — app logic tests once the pure functions are in their own files.
7. F9 compass unification, F28 preview error — the only user-visible changes; last, with goldens, under OQ-6.

F26 is a record item, not code.
