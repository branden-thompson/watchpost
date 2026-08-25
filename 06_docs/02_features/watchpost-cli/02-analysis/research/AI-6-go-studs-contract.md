# AI-6 — go-studs Component Contract (research)

Repo: `<local checkout>` (read-only). All paths below are relative to it.

## 1. Dependency + version state

| Item | Value | Cite |
|---|---|---|
| Go | `go 1.24.4` | `go.mod:3` |
| bubbletea / lipgloss / bubbles | **not present — zero charm imports anywhere** (`grep charmbracelet` = 0 hits) | `go.mod:5-11` |
| runewidth | `github.com/mattn/go-runewidth v0.0.19` | `go.mod:7` |
| Other | chroma/v2 v2.20.0 (syntax), testify v1.11.1, x/term v0.6.0 (TTY gate), yaml.v3 (golden fixtures) | `go.mod:6-10` |
| Tags | v0.1.0 … v0.4.0; HEAD is `[Unreleased]` the reference convergence | `CHANGELOG.md:8,31` |

**Nature:** go-studs is a **pure render-string library**. Every component exposes `Render() string` (e.g. `components/progress.go:143`, `smart_separator.go:112`, `table.go:157`); there is no `tea.Model`, no `Init/Update/View`. Animation is self-driven via `time.Ticker` goroutines (`rendering/animation.go:46-50`), not tea ticks.

**What a bump to bubbletea v1 / lipgloss v1 breaks:** nothing inside go-studs — there is no charm code to break. The "move to current charm" ruling therefore applies to **watchpost only** (and to the the reference CLI side, out of scope here). The real integration risk is the opposite: go-studs emits raw SGR bytes and detects width/TTY itself, so a bubbletea host must (a) pass explicit widths from `tea.WindowSizeMsg` instead of letting components call `GetTerminalSize()`, and (b) force color on, because `colorEnabled()` checks `term.IsTerminal(os.Stdout)` (`rendering/color_enabled.go:28`) — fine in a real TTY, but tests/pipes strip escapes. A pre-existing data race in `SetFramerate` vs `animationLoop` is logged (`06_docs/.../backlog.yml` item `fix-data-race-in-renderinganimationgo-se`).

## 2. Component contract

| Concern | Contract | Cite |
|---|---|---|
| Shape | Struct + `NewX()` constructor + fluent `WithY()` builders returning `*X` + `Render() string`; plus a single-call `CreateX(...)` convenience with optional variadic `width ...int` | `progress.go:45-120,537`; `message_block.go:30`; `doc.go:7-12` |
| Width | Field `width`; `getTerminalWidth()` returns explicit width if `>0`, else `rendering.GetTerminalSize()`, else `$COLUMNS`, else 80 | `smart_separator.go:226-247`; `progress.go:328-338` |
| **Canonical width calc** | `rendering.GetTerminalSize()` — order: ioctl on `/dev/tty` (works when stdout is piped), then ioctl on stdin, then `COLUMNS`/`LINES`, then **80x24** | `rendering/formatter_unix.go:15-71`; doc `formatter.go:66-77` |
| Non-TTY behavior | `/dev/tty` still answers if a controlling terminal exists (pipe to `less` keeps real width); in CI/no-tty -> env -> 80. Color is gated separately: `NO_COLOR` or non-TTY stdout => plain text | `formatter_unix.go:17,59`; `color_enabled.go:28` |
| Responsive | Optional `ResponsiveRender()` with narrow (<40) and medium (<60) modes | `progress.go:394-414` |
| Theming | Three layers: semantic `tui.*` -> design `color.*` -> raw (`tokens/registry.go:26-66`). Components call `formatter.Style(text, "tui.X")` (`smart_separator.go:116`). Raw formats: ANSI code, `#hex`, or `R:G:B` (`tokens/colors.go:101-110`) | |
| 256 vs truecolor | `ColorSequence` classifies 3-digit / palette-band codes to `38;5;N` (`rendering/color_utils.go:117-130`); `R:G:B` -> `38;2;` only via `ApplyColor` (`color_utils.go:62-66`); `WrapSGR` cannot express truecolor (`color_enabled.go:51-54`). `DetectTerminalCapabilities` reads `TERM`/`COLORTERM` (`formatter.go:34-58`). Dark/light via `tokens.ApplyTheme` (`tokens/themes.go:144`) and `theme.Detect` (`theme/theme.go:80`) | |
| Animation | `AnimatedSpinner` holds `currentFrame`; caller drives `NextFrame()`/`CurrentFrame()` (`animated_spinner.go:120-125`). Shared `AnimationController` ticker default **50 ms (20 fps)**, recommended 100 ms, bounded by `SetFramerate` (`rendering/animation.go:33,225-229`). Demo loop `AnimateFor` prints `\r\033[K` itself (`animated_spinner.go:300-322`) — not usable under bubbletea; use frame-stepping API + `tea.Tick` | |
| Taxonomy | `ComponentClassification{Capability, Type}` + `GetXClassification()` per component | `taxonomy.go:33-117`; `progress.go:618` |

## 3. Testing conventions

| Convention | Detail | Cite |
|---|---|---|
| Color forced on | Per-package `TestMain` calls `rendering.SetColorEnabledForTest(true)` | `components/zz_colormain_test.go:13-16` |
| Golden A (byte-pin) | `testdata/legacy_pin_<name>.golden`; regenerate with `go test ./components -update-golden` | `data_table_row_test.go:13,209-224` |
| Golden B (spec fixture) | YAML `docs/02_features/progress-bar-improvement/fixtures/golden.yaml`, ANSI-stripped compare | `golden_fixtures_test.go:36-60` |
| Invariant helpers | `internal/testutils`: `StripANSI`, `GetDisplayWidth`, `AssertNoANSI`, `AssertContainsANSI`, `AssertWidthEquals/LessThanOrEqual/GreaterThanOrEqual` | `internal/testutils/testutils.go:12-53` |
| Coverage | Script asserts >=90% (`scripts/coverage.sh:24`); project target 90%+ (`docs/02_features/go-studs-tests/README.md:16`) | |
| Gate | `make verify` = gofmt + vet + test (**no `-race` yet**, backlog item) | `Makefile:7-22` |

**Acceptance kit for a new component:** `X.go` (builder + `CreateX` + `GetXClassification`), `X_test.go` using testify + testutils width/ANSI invariants at widths 40/60/80/120 incl. narrow path, a golden under `testdata/` (flag-regenerable), a `doc.go` section (`components/doc.go:30-74` pattern), a taxonomy var (`taxonomy.go:85-117`), CHANGELOG entry, gofmt/vet clean.

## 4. Gap analysis for watchpost

**Reuse as-is:** `DataTable`/`AdvancedDataTable` (location lists; `table.go:62-157`, `data_table_row.go:90`), `StatusIndicator`/`CreateDeclarativeStatus` (alert badges; `status.go:55`, `declarative_status.go:40`), `HeaderFooterComponent` (`header_footer.go:31`), `SmartSeparator`, `ProgressBar` (humidity/UV with `WithFillChar`; `progress.go:89`), `CreateMessageBlock` (wrapped alert text; `message_block.go:30`), `BadgeAligner` (`badge_aligner.go:15`).

| Net-new | Closest pattern to copy | Conventions |
|---|---|---|
| (a) Wind gauge (arrow + abbr + speed, animated) | `AnimatedSpinner` frame-step API + `StudsSpinnerType` enum (`animated_spinner.go:36-45,120`) | Frames = 8 arrow glyphs by bearing; `NextFrame()` for gust pulse; classification `Animated, Indicator` |
| (b) Precip bar chart | `ProgressBar.calculateLayout` + `MultiProgressBar.RenderAll` (`progress.go:213,483`) | Vertical `▁▂▃▄▅▆▇█` columns, width-bounded, `TerminalWidthAware, DataDisplay` |
| (c) Temp/pressure sparkline | same block-glyph scaling; `ResponsiveRender` narrow mode (`progress.go:394`) | Pure function `CreateSparkline(values []float64, width ...int)`; runewidth via `rendering.DisplayWidth` |
| (d) Humidity/UV meter | `ProgressBar` presets + `getStateFillColor` (`progress.go:310`) | Add preset, not new type; threshold color via `tui.*` tokens |
| (e) Alert banner/marquee | `CreateMessageBlock` wrapping + `AnimatedSpinner` frame offset | Marquee = offset index stepped by host tick; `Animated, Feedback` (type exists, unused) |
| (f) Radio "now playing" panel | `TextFormatter.CreateBox` (`formatter.go:161`) + `SmartSeparator` labels | Static `Layout`; inner spinner reused |
| (g) Responsive multi-column grid | No precedent; nearest is `AdvancedDataTable.calculateColumnWidths` (`table.go:254-304`) | New `Layout` component joining pre-rendered cells via `rendering.PadRight`; breakpoints mirror 40/60 thresholds |

## 5. Upstream path

Versioning: Keep-a-Changelog + SemVer, git tags v0.1.0–v0.4.0 (`CHANGELOG.md:5-6`). No formal CONTRIBUTING; work is tracked in `06_docs/02_features/A2DH-backlog/backlog.yml` via `a2dh backlog` (`README.md:355-359`) and the the reference convergence was done as backlog items with commit evidence (`backlog.yml` item `absorb-the reference CLI-ui-improvements-tokens`). Hard rule: **no app-semantic tokens upstream** — a strip test asserts zero `oncall.*`/`crew.*` keys (`CHANGELOG.md:18`; `tokens/app_semantic_strip_test.go`). Watchpost must therefore use generic `tui.*`/`color.*` names (e.g. `tui.WarningText`, not `weather.alert`). Per-component updates required: taxonomy var (`taxonomy.go:85`), `components/doc.go` section, CHANGELOG, README component list, backlog item.

## 6. Opinion

**Dependency strategy:** Do **not** block on a go-studs charm bump — there is nothing to bump. Pin go-studs at HEAD via `replace` and adopt bubbletea v1.x/lipgloss v1.x in watchpost directly. The integration contract is: bubbletea owns the tick and window size; go-studs components are called with explicit `WithTerminalWidth(msg.Width)` inside `View()`; stdout (non-TUI) mode calls `rendering.GetTerminalSize()` once at invocation, satisfying the HUM LEAD ruling. Call `rendering.SetColorEnabledForTest(true)` in tests only; in the TUI, rely on the real TTY gate.

**Authoring template for the chart family:** one file per chart with `type X struct{values []float64; width int; label string}`, `NewX(values)`, `WithTerminalWidth/WithLabel`, `Render()` + `ResponsiveRender()`, `CreateX(label string, values []float64, width ...int) string`, `GetXClassification()`; shared private helper `scaleToBlocks(values, height) []string` in a new `components/chart_scale.go`; tests with testutils width invariants at 40/60/80/120 plus a `testdata/legacy_pin_*.golden`.

**Strongest counter-argument:** go-studs' self-driven goroutine animation and process-global color/theme state (`tokens.ApplyTheme` mutates globals, `themes.go:155`) are at odds with bubbletea's pure `Update/View` model; mixing them risks races (already one open) and double-rendering. The clean answer would be a lipgloss-native component layer. Rebuttal: go-studs' frame-step APIs are already pull-based, so the host tick can drive them without the controller; the mandate to use go-studs and the upstreamability requirement outweigh the purity gain.
