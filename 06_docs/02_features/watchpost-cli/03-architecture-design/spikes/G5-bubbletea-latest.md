# G-5: Bubbletea "Latest" Verification (2026-08-23)

All findings verified against downloaded module source (Go module cache, `MODCACHE=$GOMODCACHE`) and a compiling smoke-test module at the G5 scratchpad (`g5-bubbletea/main.go`, builds clean on Go 1.24.4).

## 1. What "latest" is

**v2 is stable.** `go list -m -versions`: bubbletea v2 final `v2.0.0` (published 2026-02-24, GitHub API), latest **v2.0.9** (2026-08-19); lipgloss **v2.0.6**; bubbles **v2.2.0**. v1 line ended at bubbletea v1.3.10 / lipgloss v1.1.0 / bubbles v1.0.0. **Critical: the v2 module path is the vanity domain `charm.land/bubbletea/v2`** (likewise lipgloss/bubbles) — `go get github.com/charmbracelet/bubbletea/v2` fails with a module-path mismatch. The v2.0.0 release notes position v2 (cursed renderer, declarative views, keyboard enhancements) as the way forward; the v2 README/upgrade guide assume new work is on v2 (https://github.com/charmbracelet/bubbletea/releases/tag/v2.0.0).

## 2. Renderer & CPU policy

v2 replaces the v1 line-diff "standard renderer" with the **cursed renderer** (`cursed_renderer.go`, ncurses-style cell-buffer diffing via `github.com/charmbracelet/ultraviolet`). AI-9's CPU model **still holds, and improves**:

- The render loop is still a ticker at the configured FPS (`tea.go` `startRenderer()`, `time.NewTicker(framerate)`), default 60, capped 120 (`renderer.go:13-14`).
- **Unchanged-frame short-circuit survives**: `cursedRenderer.flush()` returns immediately when `viewEquals(s.lastView, &view)` and bounds are unchanged (`cursed_renderer.go:320-323`) — idle cost is a mutex + struct compare, no terminal writes. "0 fps idle" effectively holds.
- Changed frames now emit **cell-level** damage (touched lines in a cell buffer), strictly cheaper than v1's whole-line diff — better than the arithmetic assumed.
- **`tea.WithFPS(n)` still exists** (`options.go:142`). Our one-100ms-`tea.Tick`-while-animating policy is unchanged; `WithFPS(30)` remains a valid belt-and-suspenders cap.

## 3. API surface check (all paths in `charm.land/bubbletea/v2@v2.0.9`)

| API | Status in v2.0.9 |
|---|---|
| `tea.WindowSizeMsg` | Exists unchanged (`screen.go:7`); `tea.WindowSize()` cmd renamed `tea.RequestWindowSize` |
| `tea.Tick`, `tea.Every` | Exist unchanged (`commands.go:154,102`) |
| `tea.WithAltScreen()` | **Removed.** Declarative: `view.AltScreen = true` in `View()`. Inline mode is the default and now auto-sizes to frame height (`cursed_renderer.go` flush resizes `frameArea` to `content.Height()`) — ideal for the ~40x10 mini radio player |
| `tea.WithInput` / `tea.WithOutput` | Exist (`options.go:40,30`); `WithInputTTY` removed (TTY auto-opened) |
| `tea.WithMouseCellMotion()` | **Removed** → `view.MouseMode = tea.MouseModeCellMotion`; `MouseMsg` is now an interface, events split into `MouseClickMsg`/`MouseReleaseMsg`/`MouseWheelMsg`/`MouseMotionMsg` |
| `BackgroundColorMsg` / OSC-11 | **Confirmed in core** (`color.go:67,13`): `tea.RequestBackgroundColor` cmd → `tea.BackgroundColorMsg` with `.IsDark()`. AI-9 called it right that it's v2-only |
| `Suspend`/`Resume` | Exist: `tea.Suspend()` cmd, `SuspendMsg`, `ResumeMsg` (`tea.go:572-586`); plus new `tea.Interrupt` |

New options: `WithColorProfile`, `WithWindowSize(w,h)` (testing), `WithFilter`, `WithContext`.

## 4. v1→v2 migration deltas to adopt day one

From `UPGRADE_GUIDE_V2.md` shipped in the module (bubbletea, lipgloss, bubbles each ship one):

- **Model interface**: `Init() Cmd` and `Update(Msg) (Model, Cmd)` are **unchanged** (beta-era `Init() (Model, Cmd)` did not ship). Only `View()` changed: returns `tea.View` (`tea.NewView(str)` + fields `AltScreen`, `MouseMode`, `Cursor`, `WindowTitle`, `BackgroundColor`…) instead of `string` (`tea.go:53-65`).
- **Program construction**: `tea.NewProgram(model)` unchanged; terminal-feature options/commands (`EnterAltScreen`, `EnableMouse*`, `SetWindowTitle`, `Show/HideCursor`) all removed in favor of View fields.
- **Keys**: `tea.KeyMsg` is now an interface; match `tea.KeyPressMsg`; `msg.Type→msg.Code`, `msg.Runes→msg.Text`, space is `"space"` not `" "`. `bubbles/v2/key.Matches` is generic over `fmt.Stringer` and works with `KeyPressMsg` (`bubbles/v2@v2.2.0/key/key.go:130`).
- **lipgloss v2**: `Color()` is a function returning `image/color.Color`; `TerminalColor` and the whole `Renderer` gone (Style is a pure value); `AdaptiveColor` → `compat` package or `LightDark(isDark)` fed by `tea.BackgroundColorMsg`; downsampling moved out of `Render()` to the output layer (Bubble Tea handles it). `Style.Copy` still present but deprecated — use assignment.
- **bubbles v2**: requires bubbletea v2 (`go.mod`: bubbletea v2.0.8, lipgloss v2.0.5); exported `Width`/`Height` fields → setters; `DefaultKeyMap` vars → functions.

## 5. go-studs coexistence (D-9 seam)

**Behavior change, seam holds.** v2 no longer passes View strings through verbatim: `flush()` wraps content in `uv.NewStyledString` and parses it cell-by-cell (`ultraviolet/styled.go:101` `printString`). Verified in source:

- **SGR (`CSI …m`) and OSC-8 hyperlinks are fully parsed** into cell styles (`styled.go:195-201`) — go-studs' raw-SGR output composes correctly inside `View()`; smoke test with a raw `\x1b[38;2;…m` string compiles and renders through this path.
- **Any other escape in content (cursor movement, etc.) is dropped** (`styled.go:193` `// TODO: Handle cursor movement`). go-studs must stay SGR-only — it is, per D-9.
- **Colors are re-emitted through the detected color profile** (`cursed_renderer.go:644` `scr.SetColorProfile`; `ColorProfileMsg` in `profile.go`): truecolor SGR from go-studs downsamples automatically on 256/16-color terminals — a net win v1 never gave raw strings.

## 6. VERDICT — pin these

```
charm.land/bubbletea/v2 v2.0.9
charm.land/lipgloss/v2  v2.0.6
charm.land/bubbles/v2   v2.2.0
```

**Survives**: CPU/tick policy (tick-driven renders, unchanged-frame short-circuit, `WithFPS`), `WindowSizeMsg`, `Tick`, `WithInput/WithOutput`, Suspend/Resume, go-studs SGR seam, `Init`/`Update` signatures.
**Restate**: import paths (`charm.land/*`); `View() tea.View`; alt-screen/mouse/title/cursor as View fields not options/cmds; key/mouse message types; lipgloss color system (`color.Color`, no Renderer, `LightDark`); bubbles setter methods.
**New capability to exploit**: core OSC-11 background detection (`RequestBackgroundColor` + `LightDark` for theme-correct palettes), auto-height inline mode for the mini player, cell-level damage rendering (cheaper SSH/tmux), keyboard enhancements (shift+enter etc.), `WithWindowSize`/`WithColorProfile` for deterministic tests, native `Cursor` control and OSC-52 clipboard.

## Claims table

| Claim (AI-9, v1.3.10-derived) | v1 fact | Latest (v2.0.9) fact | Survives? |
|---|---|---|---|
| Latest stable is v1.3.x | v1.3.10 | v2.0.9 stable since 2026-02-24, path `charm.land/bubbletea/v2` | **No — restate** |
| Renderer skips unchanged frames | line-diff std renderer | cursed renderer, `viewEquals` short-circuit + cell diff | **Yes (better)** |
| `WithFPS`, default 60 | yes | yes (`options.go:142`, max 120) | **Yes** |
| `WindowSizeMsg`, `Tick` | yes | yes | **Yes** |
| `WithAltScreen` option / inline mode | option/cmds | `view.AltScreen` field; inline default, auto-height | **Restate** |
| `WithMouseCellMotion` | option | `view.MouseMode`; MouseMsg now interface | **Restate** |
| BG detect (OSC-11) is v2-only | absent in v1 | `RequestBackgroundColor`/`BackgroundColorMsg` in core | **Yes (now usable)** |
| Suspend/Resume | yes | yes + `Interrupt` | **Yes** |
| `Init() Cmd` / `Update` signatures | yes | unchanged; only `View() tea.View` | **Yes (View restated)** |
| Raw SGR passes through View | verbatim | parsed to cells; SGR+OSC-8 preserved, profile-downsampled, other escapes dropped | **Yes, with caveat** |
| lipgloss `Style.Copy` removed | deprecated | still present, deprecated (`style.go:229`) | **Yes** |

## Sources

- Module cache: `$GOMODCACHE/charm.land/{bubbletea,lipgloss,bubbles}/v2@{v2.0.9,v2.0.6,v2.2.0}` — `UPGRADE_GUIDE_V2.md`, `tea.go`, `options.go`, `renderer.go`, `cursed_renderer.go`, `color.go`, `screen.go`, `commands.go`, `profile.go`, `style.go`, `key/key.go`
- `$GOMODCACHE/github.com/charmbracelet/ultraviolet@v0.0.0-20260811164956-006e29f97886/styled.go`
- https://github.com/charmbracelet/bubbletea/releases/tag/v2.0.0 ; GitHub API `releases/latest` (v2.0.9, 2026-08-19)
- Compile smoke test: scratchpad `g5-bubbletea/` (`go build` OK, Go 1.24.4)
