# AI-9 — Terminal Compatibility Matrix (glyphs, color, frame budget, resize, width)

Builds on AI-6 (go-studs: raw SGR, 256-color bias, 50 ms default tick). Local experiments run in Ghostty 1.x on macOS (`TERM=xterm-256color`, `COLORTERM=truecolor`, `tput colors`=256, `$COLUMNS` unset in non-interactive shell), go-runewidth v0.0.28.

## 1. Glyph width and rendering matrix

go-runewidth widths (local experiment; "EAW" = `EastAsianWidth=true`, auto-enabled when `RUNEWIDTH_EASTASIAN=1` or locale is CJK):

| Glyph class | Sample | Default | EAW | Ambiguous? | Renders 1 cell in iTerm2/Terminal.app/Ghostty/Alacritty/kitty/WezTerm/WT/VS Code/tmux |
|---|---|---|---|---|---|
| Block elements U+2581–2588 | ▁▄█ | 1 | 2 | yes | All (but tmux/CJK locales may pad to 2 when EAW on) |
| Arrows U+2190–2199 | ↑↗ | 1 | 2 | yes | All; same EAW caveat |
| Arrow U+2B06 ⬆ | ⬆ | 1 | 1 | no | Risky: Emoji_Presentation-eligible; WT/iTerm2 may draw emoji glyph |
| Braille U+2800–28FF | ⠁⣿ | 1 | 1 | no | All; font-dependent (Menlo/Nerd OK; some Windows fonts substitute) |
| Box drawing U+2500–257F | ─┃╭ | 1 | 2 | yes | All; tmux may fall back to ACS (`U8=0`) with odd fonts |
| Text symbols | ☀☁☂❄ | 1 | 1 | no (☀) | 1 cell; WT and VS Code may auto-colorize as emoji |
| ⚡ U+26A1 | ⚡ | **2** | 2 | no | Emoji_Presentation=Yes: 2 cells everywhere |
| Emoji + VS16 | ☀️ | **1** | 1 | — | **Mismatch**: runewidth says 1, iTerm2/Ghostty/kitty/WezTerm render 2 → layout drift |
| Pictographic emoji | 🌧 🌤 | **1** | 1 | — | **Mismatch**: Emoji_Presentation=No, most terminals still draw 2 cells |
| Nerd Font PUA | U+F0C2 U+E30C | 1 | 2 | — | Needs patched font; otherwise tofu. Ghostty/kitty ship fallback glyphs, Terminal.app/WT do not |
| ° • · | ° | 1 | 2 | yes | 1 cell except EAW |

Avoid: any emoji (U+1F3xx, VS16 sequences, ⚡) in aligned layouts; U+2B06-style "heavy" arrows; Nerd Font icons without a fallback. Ambiguous-width glyphs (blocks, arrows, box drawing) are safe *unless* the user runs a CJK locale or sets `RUNEWIDTH_EASTASIAN=1`; offer `--ascii` to opt out.

## 2. Color

| Terminal | Truecolor | Sets `COLORTERM` | Notes |
|---|---|---|---|
| iTerm2, kitty, WezTerm, Alacritty, Ghostty, VS Code, Windows Terminal | yes | mostly yes (Alacritty/WT do not) | `38;2;` accepted |
| Terminal.app | only macOS 26+ | no | 256-color fallback essential |
| tmux | 2.2+ passthrough | no (inherits) | needs `terminal-features ",*:RGB"` (3.2+) or `terminal-overrides ",*:Tc"`; `tmux -T RGB` forces it; TERM should be `tmux-256color` |

`COLORTERM` is not forwarded over ssh/sudo: treat it as a *positive* signal only. Honor `NO_COLOR` (any value → no SGR). Background: lipgloss `HasDarkBackground()` uses termenv's OSC 11 query; run it **before** `tea.NewProgram` reads stdin; it fails silently (defaults dark) over tmux/ssh on some paths. bubbletea v1 exposes no background event; v2 adds `BackgroundColorMsg`. go-studs' 256-color bias means gradients quantize to the 6×6×6 cube: 5–8 step bands look fine, 20-step ramps band visibly. Supply `R:G:B` when `COLORTERM=truecolor`, or design ≤8-step palettes.

## 3. Frame budget

- v1.x standard renderer: ticker at `defaultFPS=60` (cap 120, `tea.WithFPS`); `flush()` short-circuits when `View()` string equals `lastRender`, then skips unchanged lines line-by-line; full repaint only on resize/alt-screen toggles. Cost is dominated by building the `View()` string, not the write.
- v2.0 ships the cell-buffer "Cursed" renderer (ncurses-style damage tracking, cell-level diff). Not needed at 200×60 if we keep unchanged-frame short-circuit.
- Estimate: 200×60 = 12k cells ≈ 30–60 KB styled string; a lipgloss `View()` costs ~0.3–1 ms. 10 fps ≈ ≤1% CPU; 20 fps ≈ 2%; 60 fps ≈ 5–8% (over budget). Verify with `pprof`; arithmetic supports **spinner/marquee 100 ms, gauges 250 ms, idle 0 fps**.
- Single tick source: one `tea.Tick(100ms)`; components derive phase via `frame % n`. Avoid `spinner.Tick` plus per-component ticks (each message forces a `View()`). Return `nil` cmd when nothing animates; cache the frame when the state hash is unchanged.

## 4. Resize and alt-screen

- `tea.WindowSizeMsg{Width,Height}` is delivered once at start (bubbletea queries size) and on every resize; renderer calls `repaint()`. Unix: SIGWINCH listener; Windows: console size polled (bubbletea `handleResize`). Always size from the message, never `$COLUMNS`.
- Alt-screen: full-window, clean exit, no scrollback pollution — right for the dashboard. Inline mode: leaves output in scrollback, coexists with shell, no resize repaint of prior lines — right for a ~40×10 "mini radio" and for `--once` prints. Inline caveat: height > terminal rows causes scroll artifacts; clamp to `min(height, rows-1)`.
- Mouse: `WithMouseCellMotion` only (clicks/wheel); `AllMotion` floods Update with move events (dozens/sec) — measurable CPU. Bracketed paste is on by default; keep it.
- `WithoutSignalHandler`: you then own SIGINT/SIGTERM; forgetting to restore the terminal leaves raw mode/alt-screen on Ctrl-C. Prefer default handler plus `tea.Interrupt` handling.

## 5. Non-TTY and width

| Case | bubbletea v1 behaviour | Recommendation |
|---|---|---|
| stdout pipe, stdin TTY | Program runs, skips raw mode/cursor hiding; escape codes still emitted | Detect with `isatty(stdout)`; go stdout mode (plain text, no SGR unless `--color=always`) |
| stdin pipe, stdout TTY | bubbletea opens `/dev/tty` for input; TUI works | Fine; allow `--location` piped in |
| Both pipes (CI, cron) | No size query; width 0 | Use `$COLUMNS` → `COLUMNS` in `.zshrc` is often unset (local: `0`); fall back to 80 |
| tmux/ssh | Size correct via SIGWINCH; `$COLUMNS` stale after resize | Always message-driven |

Floors and breakpoints: minimum 40 columns (mini player), 60 (single-column dashboard), 80 (standard two-column), 120 (three panels + charts). Below 40, print "terminal too narrow" rather than wrap.

## 6. Opinion

| Topic | Recommendation | Strongest counter-argument |
|---|---|---|
| (a) Glyphs | Blocks ▁–█, arrows ↑↗→…, light box-drawing ─│╭╮; ASCII fallback `#`/`^>v</+-|` under `--ascii`, `TERM=dumb`, or EAW on. No emoji, no Nerd Fonts, no braille in v1 | Braille sparklines double resolution and render 1-cell everywhere; only font coverage on Windows is the risk |
| (b) Color | 256-color default (matches go-studs), truecolor opt-in when `COLORTERM=truecolor`; ≤8-step palettes; honor `NO_COLOR`; OSC 11 probe before program start with dark default | Truecolor-first is simpler and every 2025 terminal except Terminal.app <26 supports it |
| (c) Ticks | One 100 ms tick, only while something animates; 250 ms for gauges; `WithFPS(30)` | 50 ms looks smoother for marquee; cost ~2× |
| (d) Screen | Alt-screen for dashboard; inline for mini radio and `--once` | Alt-screen everywhere simplifies resize handling |
| (e) Breakpoints | 40 / 60 / 80 / 120 columns; height breakpoints 10 / 24 / 40 *(height set superseded by PD-4: two breakpoints — architecture.md §9)* | Fewer breakpoints (60/100) means fewer layouts to test |

## Sources

- go-runewidth source and local experiment: https://github.com/mattn/go-runewidth/blob/master/runewidth.go
- bubbletea v1.3.10 `standard_renderer.go`, `tea.go`: https://raw.githubusercontent.com/charmbracelet/bubbletea/v1.3.10/standard_renderer.go , https://raw.githubusercontent.com/charmbracelet/bubbletea/v1.3.10/tea.go
- bubbletea releases (v2 renderer): https://github.com/charmbracelet/bubbletea/releases
- lipgloss `HasDarkBackground`: https://pkg.go.dev/github.com/charmbracelet/lipgloss
- termstandard/colors truecolor matrix and `COLORTERM`: https://github.com/termstandard/colors
- tmux FAQ (RGB/Tc, `U8=0`): https://github.com/tmux/tmux/wiki/FAQ
- Windows Terminal emoji width issue: https://github.com/microsoft/terminal/issues/900
