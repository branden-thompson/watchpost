# DISCOVER Lens L5 — go-studs seams & the render path

Read-only research lens, 2026-08-26. Measurements: Go 1.25, `go test -bench` with colour forced on,
`modes/tty` fixtures (133×44, 10 favourites + 50 RECENT, 10-row window). Scratch benchmarks and overlay
prototypes are reusable for BUILD (scratchpad `l5/`: `zz_l5_bench_test.go`, `zz_l5_render_bench_test.go`,
`overlay{,A,B,C,E}.json`, `cpu.out`, `mem.out`). Any change under `third_party/go-studs` needs HUM LEAD
approval (OQ-1); `NOTICE.md` still says "Do not edit here" and `scripts/sync-go-studs.sh:22` overwrites
local edits on resync (F19).

## Findings

| ID | Sev | Finding | Evidence | Recommendation | Approval |
|---|---|---|---|---|---|
| L5-F1 | **PERF** | go-studs opens `/dev/tty` and ioctls it **twice per frame**: `NewDataTable` → `NewTextFormatter` → `DetectTerminalCapabilities` → `GetTerminalSize` (`components/data_table_row.go:108`, `rendering/formatter.go:25-31,61`, `formatter_unix.go:15-36`); `DataTable` never reads the probed width | 32 µs each; **60 % of sampled CPU** of `View()` is `syscall.Open` kernel time; prototype A (`sync.Once` memo): 670 → 520 µs/frame (−22 %) | Lazy size probe (resolved on first `GetCapabilities()`), `RefreshCapabilities()` as the explicit path; replace the 70-line ioctl block with `golang.org/x/term.GetSize` (already imported) — retires the `unsafe` P10-09 exemption | HUM LEAD |
| L5-F2 | PERF | Width measurement strips ANSI with a regexp on every call, in both app (`render.go:1156-1180`) and kit (`text_utils.go:33-39`); every cell measured ≥ 3× | **18.9 % of all frame allocation**; styled row 3,656 ns / 8 allocs vs 1,522 ns / 0 allocs with `charmbracelet/x/ansi.StringWidth` (already a transitive dep; what ultraviolet lays cells out with); prototype C: 520 → 395 µs, 10,044 → 4,614 allocs, **all render + tty tests unchanged** | One width authority (`x/ansi.StringWidth`) in the kit; app `displayWidth` delegates or dies | HUM LEAD (kit half) |
| L5-F3 | PERF | `SGR()` allocates 5× per styled cell (`color_utils.go:147-155`: SplitSeq + append + Join + Sprintf) | 115.8 ns / 5 allocs vs 22.9 ns / 0 for the app's `sgrRaw`; ~400 per frame ≈ 2,000 allocs; prototype E (memo per param list): **345 µs / 322 KB / 3,216 allocs (−49 % time, −68 % allocs vs baseline)** | Cache classified escapes per param list, or precompute at `Tok()` registration | HUM LEAD |
| L5-F4 | **CORRECTNESS** | The table's header and every un-styled cell are painted by go-studs' own palette, gated by `$TERM`, not by the Watchpost theme (`data_table_row.go:192-195, 329-335, 816-861`) | With `TERM=xterm-256color`: header `38;5;135`, numbers/zip/cond `38;5;245`, unselected NAME `97`; with `TERM` unset the same cells are plain grey 250. The `t` chooser cannot restyle them (Monochrome still shows a purple header); tests never pin it (colour off under `go test`) | App sets `Column.Color`/`CellStyles` for every column from `Tok()` (new tokens `table.header`, `table.muted`, `table.name` defaulting to 135/245/97 so nothing changes) + a colour-on fidelity test; kit: `DataTableDefinition.NoAutoStyle` | HUM LEAD (visual) |
| L5-F5 | CORRECTNESS | `ColorSequence` mangles qualified composites: `SGR("1;38;5;220")` → `\x1b[1;38;5;38;5;38;5;220m`; the app survives by the `Tint`/`TintRaw` call-site convention (UAT 108), which user theme files cannot see — `"temp.hi": "38;5;208"` renders garbage | measured | Composite-aware `SGR` (consume `38;5;n`, `48;5;n`, `38;2;r;g;b`, `48;2;r;g;b` atomically); then `sgrRaw`/`TintRaw` collapse into `Tint`, background support arrives free, the UAT-108 test moves to the kit | HUM LEAD |
| L5-F6 | PERF | Layout facts recomputed ~8× per frame: `compact()` (`dashboard.go:2563-2578`) renders the full radio module + control row to count lines, called from `alertHeight`, `radioHeight`, `radioPanel`, `alertArea`; `windowSize` and `body` render them again | `radioLines` 7.6 % + `compact` 6.6 % + duplicates ≈ **15 % of frame allocation** thrown away | Per-frame `frameLayout{opts, compact, radioLines, alertH, radioH, controlRow, window}` computed once at the top of `View()` | app only |
| L5-F7 | APP-SEAM | `tableGeom`/`rowLen` (`render.go:396-438`) mirror the kit's fill/gutter math (`data_table_row.go:66-74, 892-979`); the kit's only exported width API runs a *different* algorithm than `Rows()` | kept in sync only by the mock goldens | Kit `DataTable.Geometry() []ColumnGeom`; app `groupHeader` consumes it; `tableGeom` deleted | HUM LEAD (API) |
| L5-F8 | UPSTREAM | `clampCells` (`render.go:656-663`) patches the kit returning over-wide fixed cells untouched (`formatCell :774-777`) | third per-cell width pass | `ColumnDefinition.Clamp` (default true for `Width > 0`) | HUM LEAD |
| L5-F9 | PERF | `Overlay` allocates a full cell canvas per modal frame (lipgloss compositor → `ultraviolet.NewBuffer`) | 15.2 % of allocation, 786 µs / 795 KB per composite; modal frame 1.55–1.87 ms — only under the 50 ms viz tick with a modal open (~3 % of a core) | Accept; ANSI-aware line splice if it bites | — |
| L5-F10 | INFO | `TintDefault` 25.6 µs / 2 allocs (~4 % of frame); `TitleGradient` 1.96 µs; `Tok()` 15 ns | measured | Leave, documented | — |
| L5-F11 | INFO | Frame cadence: the 300 ms shimmer `tick()` is armed unconditionally (`dashboard.go:287-289, 302, 331-333`) though `LoadingDots` only shows while a row is `Loading`; `vizTick` is correctly gated. bubbletea v2 calls `View()` after every Update but the renderer short-circuits on an unchanged view (`cursed_renderer.go:288`) and diffs by cell otherwise (`TestL5ViewStable`: byte-identical across ticks) | idle 0.67 ms × 3.3/s ≈ 0.2 % CPU, ~1.4 MB/s garbage; viz 1.3 %, ~8.7 MB/s | Arm the shimmer tick like the viz tick (while any row is `Loading` or `volFlash` is live) | app only |
| L5-F12 | INFO | Width vs terminals: runewidth, uniseg, displaywidth, x/ansi, kit and app all agree on every app glyph (◆ ▶ ⚠ ✔ ✘ º ∞ ↗ ↘ ♪ ■ … ━ █ ▲ ▼ │ ›) and CJK; the only disagreement is VS16 emoji (`⚠️`: 1 vs 2), terminal-dependent even in bubbletea | measured | None beyond F2; optionally strip variation selectors in `Plain` | — |
| L5-F13 | CORRECTNESS (kit, unused path) | Kit byte-width family is wrong for every non-ASCII glyph: `GetVisualWidth` = `len()` after strip (`"72ºF"` → 5), `PadRight/Left/Center`, `TruncateText` byte-slices (invalid UTF-8 split on CJK), `CreateBox`, `AdvancedDataTable.formatCell` — why the app hand-rolls `Panel`/`ScrollPanel` | measured | Delegate the `ANSIFormatter` family to `DisplayWidth`/`TruncateString` | HUM LEAD |
| L5-F14 | PERF | Kit truncation is O(n²) (`truncateText :747-768`, `TruncateString :129-155` shrink and re-measure per step); NAME is `Truncatable` | ~len(name) regexp passes per row per frame at narrow widths | `x/ansi.Truncate` (ANSI-preserving, O(n)) | HUM LEAD |
| L5-F15 | APP-SEAM | App duplicates of kit primitives: `render.WrapText` ≡ `WrapTextANSI`; `PadTo` ≡ `PadString`; `displayWidth` ≡ `DisplayWidth`. `WrapSegments`/`WrapLines`/`PadBetween`/`Band`/`Block`/`Module`/`ScrollPanel`/`KeyCap*` are app policy | — | Delete the three duplicates after F2/F13 make the kit versions trustworthy; `KeyCap` is a later upstream candidate | app |
| L5-F16 | PERF | DataTable row rendering allocates for wrapping it never does (`FormatEnhancedTableRow :237-295` builds `[][]string` per row; `formatSingleLineRow` re-measures every cell; `Rows()` appends without prealloc); app `layout.columns` rebuilds the spec per table per frame | 5.3 % + 7.5 % of allocation | Kit fast path when no column wraps; app caches the column spec per (width, days, extDates) | HUM LEAD (kit) |
| L5-F17 | INFO | Two colour gates (`$TERM` in the kit's `Style()`, NO_COLOR+tty in `WrapSGR`) and two theme systems coexist; each `NewDataTable` allocates a fresh `TokenRegistry` cache | measured `SupportsColor=true` with `ColorsEnabled=false` in one process | After F4 the kit registry is inert for Watchpost; note for upstream (one gate) | — |
| L5-F18 | INFO | The 76 "upstream-governed" kit exemptions (the brief's 57 undercounts): real and worth fixing locally — F1 (unsafe ioctl → `term.GetSize`), F2/F3/F14/F16 (perf), F5/F13 (correctness), three copies of the width algorithm and two of the style ladder in `data_table_row.go`; P10-06 mutable token maps with unlocked writers (a race only if `ApplyTheme` ran during render — Watchpost never does). Noise: the P10-02 loops, density, legacy build tags, most cyclomatic entries | — | Collapse into one package-level exemption per rule with the real items tracked as upstream candidates — the count drops without deleting a check (M4) | — |
| L5-F19 | INFO | Process seam: `NOTICE.md` "Do not edit here" and `sync-go-studs.sh` `cp` contradict the capture → approve → change basis; a local fix is lost on resync | — | Approved changes as `go-studs-patches/*.patch` (or a `LOCAL_CHANGES.md` ledger) the sync script re-applies; update NOTICE | HUM LEAD (policy) |
| L5-F20 | INFO | Stray empty `tea_debug.log` files (root, `modes/tty/`, `platform/render/`; untracked) | — | delete / ignore | — |

## Measured per-frame cost (baseline, colour on)

| Case | ns/frame | B/frame | allocs |
|---|---|---|---|
| 133×44, 10 fav + 50 recent (10-row window) | **670 µs** | 437 KB | 10,044 |
| same, colour off | 377 µs | 288 KB | 7,817 |
| same, help modal open (Overlay) | 1.55–1.87 ms | 1.29 MB | 13,064 |
| 200×60 (extended columns) | 1.42 ms | 918 KB | 20,036 |
| 133×44, 1 + 1 rows | 334 µs | 181 KB | 2,455 |

Where it goes (alloc_space, cum): `LocationTable` 41.7 % (kit `DataTable.Rows` 24.7 %); ANSI-strip regexp
18.9 %; `Overlay` 15.2 % (modal frames only); `radioLines` + `compact` ≈ 14 %; `layout.columns` 7.5 %;
`TintDefault` ≈ 4 % of time; `TitleGradient` 0.3 %. CPU: `/dev/tty` open 60 % of samples (F1). Terminal
writes are skipped when the view is unchanged and diffed by cell when not.

**Prototype ladder (all goldens/tests green):** A memo tty size → 520 µs; B + app width via x/ansi →
412 µs; C + kit width → 395 µs / 342 KB / 4,614 allocs; E + SGR memo → **345 µs / 322 KB / 3,216 allocs
(−49 % time, −68 % allocs)**; 200×60 1.42 → 0.70 ms.

## Top three by value/risk

1. **F1** — stop probing `/dev/tty` per table (kit, ~10 lines; −22 % frame time, −6 syscalls/s idle,
   −120/s under viz). Risk: kit components that re-read size after a resize (unused here) — keep
   `RefreshCapabilities()`.
2. **F2 + F3** — one allocation-free width authority (`x/ansi.StringWidth`) in kit and app, plus a
   memoised `SGR()` (−49 % / −68 % cumulative; zero golden drift measured).
3. **F6** — compute layout facts once per `View()` (app only; ~15 % of allocation).

Correctness to schedule alongside: **F4** (theme-owned table colours, `$TERM` independence) and **F5**
(composite-aware `SGR`, retiring `sgrRaw`/`TintRaw`) — both touch the kit and the accepted look.
