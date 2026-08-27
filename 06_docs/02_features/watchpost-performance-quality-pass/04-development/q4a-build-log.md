# Q4a build log — go-studs correctness patches (`v0.9.7`)

**Batch:** Q4a of the plan of record v3 (§2.5 Q4a, §3 Q4a, ADR-04).
**Approval:** "Accept deviations, approved for 0.9.6 and Go 4 Q4a" (2026-08-26); patch 004
approved individually ("approved; Go to proceed").
**Branch:** `feature/watchpost-performance-quality-pass`, commits `e2…` (colour-on golden) →
`5bdd524` (+ this log).
**Status:** APPROVED 2026-08-26 ("Approved; go 4 0.9.7; Go 4 Q5") — deviations §6.1–6.3 accepted;
`v0.9.7` cut from this tree; the Q3 deviation §6.2 (miss-path µs) closes here; Q5 next.

## 1. What changed, and why (junior-first)

Watchpost draws its tables with go-studs, HUM LEAD's own MIT kit, carried in-tree. Q4a fixes four
things the kit did on Watchpost's path — each as a **patch beside the kit** that the sync re-applies,
never an edit inside it — and builds the machinery that makes that safe.

- **A patch stack, not edits.** `LOCAL_CHANGES.md` pins the upstream commit and lists every patch
  with its why, test, upstream status and removal condition. `scripts/sync-go-studs.sh` refuses a
  drifted checkout, copies to a temp dir, rewrites import paths, applies every listed patch in
  order (`git apply --check` first), swaps atomically, regenerates the licences file and re-runs
  the kit's P10 check — and stops **by patch name** with nothing changed. Its self-test builds a
  throwaway upstream and project and runs in `make gate-controls`.
- **The public-tree scrub is a machine-local patch.** The first real sync run put comment wording
  back into the tree that the 0.9.0 public-tree scrub had removed, and nothing would have caught
  it (`lint-watermark` checks AI attribution only). A tracked patch would carry the words it
  removes, so the scrub lives in the gitignored `third_party/go-studs/.local/`, and the sync fails
  loud when a listed patch is absent. Recorded as a memory note for future sessions.
- **004 — the theme owns the table's colours.** The kit painted column headers (`38;5;135`) and
  every un-styled cell (`245` / `97`) from its own `$TERM`-gated palette: the `t` chooser could not
  restyle them and `NO_COLOR=1` did not silence them (L5-F4, A11-1). Patch 004 adds
  `DataTableDefinition.NoAutoStyle` and `ColumnDefinition.HeaderColor`; the app sets both, a
  `CellStyles` entry on every non-blank cell, and three new tokens whose defaults equal the old
  colours — the colour-on golden captured *before* the patch is byte-identical after it.
- **001 — the terminal is probed once, lazily.** `NewTextFormatter` opened `/dev/tty` and ioctl'd
  it on every construction (twice per table frame; 60 % of `View()`'s sampled CPU in DISCOVER —
  L5-F1) through an `unsafe` block with a Windows twin. Now capability detection skips the size,
  `GetCapabilities` probes on first use via `golang.org/x/term` in one untagged file.
- **003 — composites survive `SGR`.** The kit re-classified every `;`-element, so `1;38;5;220`
  became `1;38;5;38;5;38;5;220` (L5-F5); the app survived by a raw-escape convention user theme
  files could not see. Qualified composites are consumed atomically and the escape is built in one
  pre-sized buffer (five allocations → two, L5-F3).
- **008 — bounded loops** on the truncation path and in `StripANSI` (P10-02).
- **`Plain` drops U+FE0E/U+FE0F** (A11-8): terminals disagree on those selectors' width.

## 2. Files touched

| Area | Files |
|---|---|
| kit patches | `third_party/go-studs/patches/{001-lazy-terminal-probe, 003-composite-sgr, 004-no-auto-style, 008-bounded-truncation}.patch`; `.local/000-public-tree-scrub.patch` (untracked); `LOCAL_CHANGES.md`, `NOTICE.md`, `sync-exclude.txt` |
| kit files (as patched) | `rendering/formatter.go`, `rendering/terminal_size.go` (new), `rendering/formatter_unix.go` + `formatter_windows.go` (deleted), `rendering/color_utils.go`, `rendering/ansi.go`, `components/data_table_row.go` |
| scripts | `sync-go-studs.sh` (rewritten + `--self-test`), `third-party-licenses.sh` (warms the module cache; tidies `go.sum` after), `Makefile` gate-controls |
| `platform/render` | `theme.go` (`TableHeader/TableMuted/TableName`), `themes.go` (per-theme values), `table.go` (`HeaderColor`, `NoAutoStyle`, `tableCellStyles`), `text.go` (`Plain`); tests `theme_test.go` (contrast, xterm-256 palette), `q4a_test.go` |
| `modes/tty` | `golden_test.go` (+ `frame-133x44-colour.golden`, captured before 004) |
| docs | `CHANGELOG.md` `[Unreleased]`, `README.md` (theme tokens), `docs/extending.md` (two rules), `architecture.md` §4 note, `THIRD_PARTY_LICENSES.md` (regenerated: 45 modules — six were missing) |
| records | `07-readiness/p10-q4a.json`, `02-analysis/q4a-bench.txt`; ledger −11 rows (retired by 001/008) |

## 3. Tests first (the pins)

| Pin | Guards |
|---|---|
| `TestFrameGoldenColourOn` (captured before 004, CQ-4) | the 133×44 frame with colour on, byte for byte, through every patch |
| `TestFrameHonoursNoColorUnderColorTerm` | **now passes** (was the Q3 known-failing skip) |
| `TestThemeTokenContrastAA` | every theme's table and text tokens ≥ 4.5:1 on its own window |
| `TestGlyphsSwapAsOneSetUnderASCII`, plain/ASCII goldens | unchanged |
| `TestTerminalProbeIsLazyAndFallsBackToTheEnvironment` | detection does not probe; first `GetCapabilities` does; `COLUMNS`/`LINES` then 80×24 when there is no terminal (exercised on CI) |
| `TestSGRConsumesQualifiedComposites` (11 rows) | composites pass through; bare codes classify exactly as before; malformed composites classify element by element as before; ≤ 2 allocations |
| `TestTintAcceptsCompositesSinceTheKitConsumesThem` | `Tint` == `TintRaw` for composites; raw stays raw for `39`/`49` compositions |
| `TestStripANSIIsBoundedByTheEscapeCount` | a torn trailing escape cannot loop |
| `TestPlainDropsVariationSelectors` | `⚠️` → `⚠` |
| `scripts/sync-go-studs.sh --self-test` (in `make verify`) | pin refused, `--allow-drift`, exclusion list, import rewrite, a local patch, a listed-but-missing patch named, a failing patch leaves the kit untouched |

## 4. Before / after

| Measure (133×44, colour on) | Q3 | Q4a |
|---|---|---|
| frame, memo miss (table re-render) | 630 µs · 354 KB · 9,164 allocs | **447 µs · 350 KB · 8,445 allocs** — the plan's ≤ 470 µs line, open since Q3, is **met** |
| frame, memo hit | 100 µs · 62 KB · 546 allocs | 102 µs · 61 KB · 504 allocs |
| `/dev/tty` opens per table re-render | 2 | 0 |
| `SGR` allocations per escape | 5 | 2 |
| `unsafe` in the kit | 1 file (+ a Windows twin on `syscall`) | none |
| P10 ledger | 120 = 66 kit + 54 non-kit | **109 = 55 kit + 54 non-kit** (11 kit rows retired: 7 by 001, 3 by 008 + the file-wide P10-02 row) |
| NO_COLOR under `TERM=xterm-256color` | escapes present (kit palette) | **zero** |

## 5. Bounds stated (§0.8)

- The patch stack is ≤ 5 (ADR-04): four tracked + one local. Every patch names its removal condition.
- `maxTruncationPasses` = 65,536 (a pass trims ≥ 1 cell); `StripANSI` iterates once per escape present.
- `SGR`'s element loop is bounded by the parameter's length (every element consumes ≥ 1 byte).
- The terminal size is probed at most once per formatter unless `RefreshCapabilities` is called.

## 6. Decisions and deviations

1. **`Tok()`-time precompute (plan Q4a.3) — tried and dropped.** Classifying every theme value at
   registration changed values the app composes raw: the module-hidden background `49` (the kit's
   D-28 rule classifies the SGR defaults 39/49 as 256-palette codes) and `TintDefault`-style
   `"0;38;5;"+Tok(…)` compositions broke five tests and the module heights. The gain it targeted
   (L5-F3) is delivered by the one-buffer `SGR` instead; token values stay as the theme wrote them.
   *Deviation recorded.*
2. **`TintRaw` stays.** The plan expected it to collapse into `Tint`; it does for qualified
   composites, but compositions carrying `39`/`49` still need the raw path (the kit rule above).
   Upstream candidate: treat 39/49 as the SGR defaults they are.
3. **Q4b stays upstream-first.** Its trigger was "Q3 + Q4a miss ≤ 470 µs at 133×44"; measured 447.
   No local performance patch lands; 002/005/006/007/009/010 are HUM LEAD's upstream candidates.
4. **Public-tree scrub as `.local/`** — see §1; the alternative (a tracked patch) would carry the
   words.
5. **Licences file** — the regeneration now lists 45 modules (`go list -m all` over-approximates
   what the binary links; six modules were silently missing before because they were not in the
   module cache). Over-listing licences is the safe side.
6. **Upstream candidates** are HUM LEAD's to open against the kit (the copyright holder); the
   patches and their tests are the proposals, listed in `LOCAL_CHANGES.md`.

## 7. Gate

| Check | Result |
|---|---|
| `make verify` (incl. the sync self-test) | ALL GATES GREEN |
| goldens: colour-on (pre-patch capture), colour-off, ASCII | byte-identical |
| fidelity + NO_COLOR + contrast tests | green |
| `make p10` | **0 live · 0 unmatched** · 109 rows (−11 kit, itemised in §4) · `07-readiness/p10-q4a.json` |
| `make alloc-budget` | green (hit 504 / miss 8,445 at 133×44) |
| `a2dh validate` | 18/18 |
| frame recorded (`make quality-bench`, count 10) | miss **445 µs** · hit 121 µs · help modal 977 µs (`02-analysis/q4a-bench.txt`) |
| Synth / Relay pty smokes (local, `dist/watchpost` at `5bdd524`) | first data → synth **PLAYING in 4 s**; `[m]` → Nearest Relay **PLAYING** in 21 s; quit clean. `WATCHPOST_LIVE=1 go test ./app -run LiveRelay`: ok (9 s) |
| the real sync reproduces the tree | yes — `scripts/sync-go-studs.sh <upstream>` at `3e85e77` yields no diff beyond the patches |

## 8. Carried forward

- 24 h idle soak ends ~2026-08-27 15:55 UTC → Q5's log. Arch relay proof: HUM LEAD, ≤ Q7.
- Q5 next (network, bytes, fan-out; `v0.9.8`): conditional GETs on the stored validators, FIRMS
  tile canonicalisation with the straddle rule, `httpx.NewTransport()` for ICY/Piper, h2/TLS
  counters read, CO-OPS/gridpoint/gridInfo lifetimes, the parse-spike attribution from the Q3 soak.
