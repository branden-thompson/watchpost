# P3 build log — modes/tty: the window, the memo, the goldens, the budgets

**Batch:** P3 (Tasks 3.1–3.10) · **Date:** 2026-08-28 · **Gate:** `make verify` ALL GATES GREEN · `make p10` 0 live,
0 unmatched · `go test ./modes/tty ./platform/render` green (race and plain)

## What landed

| Task | Files | Notes |
|---|---|---|
| 3.1 | `platform/render/theme_test.go` | `TestEventCategoryTokensAreThemeIndependent` with the **planted-control** theme (`RegisterTheme`/`UnregisterTheme`) — the guard fails on an empty set; `TestCategoryToneContrastAA`: white body text on every tint × both substrates × every theme ≥ 4.5:1 |
| 3.3 | `modes/tty/dashboard.go`, `severe.go` | state (`severe`, `severeByTab`, `severeTab`, `severeRow`, `severeDetail`, `lastBreaking[Tab]`), `modalSevere`, the `severe` action (`w`, `ctrl+s`), `toggleSevere` (open / enter drills / esc backs out), `applySevere` (focus follows the KEY; a vanished record closes), `severeOpeningTab` (10-min breaking window), `SevereMsg` dispatch, `TickerBreakingMsg` records the tab, help NAVIGATE row |
| 3.4 | `modes/tty/severe.go` | `handleSevereNav`: ←/→ wrap through the six tabs (row resets), ↑/↓ clamp; in the record ↑/↓ scroll it |
| 3.5 | `platform/render/severe_table.go`, `modes/tty/severe.go`, `body.go` | `SevereTable` on go-studs DataTable (marks 7 · num 5 · EVENT · LOCATION 22 · DECLARED 15 · EXPIRES 15, gutter 2; the ladder drops EXPIRES, DECLARED, then squeezes LOCATION to 16); `bracketSpread` headers; `Railify` + `RailGlyphsFor` — the location table's `railify` now delegates (byte-identical goldens); browse renderer width-exact to `mock.py` at 120 / 100 / 80 and `--ascii`; the dead-source line; `[enter]` mutes in the empty state |
| 3.6 | `modes/tty/severe.go` | `severeDetailLines`: the [A] shape (bold title + meta · timing · area · paragraphs · chips `[esc] Back [esc esc] Close [↑↓] Scroll`), `render.Plain` on every field again at the frame |
| 3.7 | `modes/tty/memo.go`, `view.go`, `detail.go`, `status.go` | `modalKey` (one field per input any window reads; Setup / Status / Details extras projected only while open), `modalMemo` slot, `modalView` memoised over `renderModal`; `time.Now()` → `d.now()` in Details and Status so the clock is an input |
| 3.8 | `modes/tty/severe_test.go`, `modal_test.go`, `testdata/severe-*.golden` | openers/markers exclusivity now includes `severe`; four goldens each with the width invariant beside the pin; `TestSevereRailIsOneColumn` at 24 rows |
| 3.9 | `modes/tty/bench_test.go` | `severeBench` (60-row index), `BenchmarkFrame_133x44_Severe`, `BenchmarkOverlayOnly`, `TestSevereFrameAllocBudget` |
| 3.10 | `modes/tty/status.go`, `app/severe.go` | `severe index   N rows / 500`; the cap mirrored as `tty.SevereMaxRows` with a two-way compile-time equality assert against `severe.MaxRows` |

## Measurements (P3-9, `go test -bench`, no race)

| Path | allocs / View() | pinned budget |
|---|---|---|
| severe window, memo hit (every tick while open) | **3 067** | 3 220 (× 1.05) |
| severe window, memo miss (a publish, a key) | **8 061** | 8 464 |
| Help modal hit (reference) | 3 001 | — |
| `BenchmarkOverlayOnly` (memo hit; the compositor alone) | 3 074 | — |
| dashboard frame 133×44 hit / miss (unchanged) | 554 / 7 127 | 6 000 / 10 546 |
| `BenchmarkSevereDeckTrigger` (app, 580 rows, unchanged index) | 2 020 allocs · 0.58 ms | — |

The memo-hit cost is the overlay compositor (lipgloss Canvas/Layer) — ~3 000 allocs for any open window,
the Help modal included. The memo removed the render (≈ 5 000 allocs) from the tick path; the compositor
is the next lever (open, 0.13.0 follow-up: cache the composited frame while both the base and the modal
are memo hits).

## Deviations from the plan

1. **`tickNeeded` does not include `modalSevere`.** The plan listed it; nothing in the window is
   time-relative (the title stamp is absolute, the record's "(~15m)" is composed at publish), so a tick
   would only burn the memo-hit cost. Left out; the [S] gauge and Details keep their ticks.
2. **The rail keeps the header pinned** (▲ on the header row, the rows window beneath, ▼ on the last
   visible row); `mock.py` scrolled the header away with the rows. The location table's convention wins.
3. **The table window fills the panel budget** (`modalMax − 7` rows) like Details, so at 44 rows all
   nine mock rows show without a rail; the rail appears on shorter terminals (the 24-row tests).
4. **The empty state keeps the `[enter]` chip, muted** (`KeyCapIf`), as [A] does — the test asserting its
   absence was wrong and was changed.
5. **`TestStatusAgesAndDetailsLabelsMoveWithTheClock`** mutated a snapshot in place and expected a new
   frame; under the modal memo that is a legitimate stale hit (pointer identity = unchanged content, the
   body memo's contract). The test now swaps the pointer, as a publish does.
6. **`ASCII` chip labels** follow the mock (`[up/dn]`, `[lt/rt]`); the `—` and `·` typography stays under
   `--ascii` as it does across the app.
7. `toggleModal` reached complexity 17 with the severe branches (P10-04 live) → `toggleSevere` owns them.
8. `docs/where-things-happen.md` gained the "A severe event is listed" row and the modal-memo entry
   (`TestWhereThingsHappenNamesRealSymbols` caught the `modalView` move).
9. Declaration-set goldens re-captured for `modes/tty` and `platform/render` (intentional additions).

## BUILD-exit addendum (red-team round 3)

Changed after this log was written, on verified findings (ledger `08-reports/red-team-build.md`): the Red-tier
`⚠⚠`/`!!` glyph (B-02); the short-terminal chrome budget (B-01); `render.PlainLine` on one-line fields (B-04);
the rail thumb (B-05); `enter` inert in the record (B-06); the chip label ladder and `[up/down]`/`[left/right]`
under `--ascii` — deviation 6 above is **reversed** (round-2 Y5 stands); the open tab's category tint (C-03);
the hint gated on an empty watchlist (C-07); the detail title keeps the window name (C-14); the table paints
white-family tokens (B-03, pending the colour pass); the memo minute projected under Details only (B-09);
indexed row access (B-11 — hit 3 067 → 2 401); `splitEscFusion` for the pty's fused esc+key (P-1). Goldens
re-recorded deliberately for all of the above; `severe-120x20` and `severe-80x44-ascii` added; the ASCII frame
golden re-recorded for the RECENT rail's `--ascii` glyphs (C-10).
