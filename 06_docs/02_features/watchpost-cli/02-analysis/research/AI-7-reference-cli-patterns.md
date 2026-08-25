# AI-7: the reference CLI Patterns Research (for Watchpost CLI)

Source: `<local checkout>` (read-only). All paths below are relative to that root.

**Headline finding:** the reference CLI is NOT a bubbletea app at its core. The root interactive screen is a hand-rolled ANSI printer driven by `fmt.Scanf` (`internal/tui/models/root_command_interface.go:115-127`); bubbletea is used only inside the `create` plugin (`plugins/create/plugin.go:230-234`, `internal/tui/models/app.go`). go-studs is **vendored in-tree** at `pkg/go-studs` (import `the reference CLI/pkg/go-studs`, `cmd/the reference/studs.go:9-11`), not a `go.mod` dependency — confirmed: `go.mod` has no go-studs line. It is hash-pinned against upstream via `make resync-studs` (`Makefile:198-199`).

## 1. Layout & composition

| Concern | the reference CLI pattern | Cite |
|---|---|---|
| Package layout | `cmd/the reference` (main, router, theme), `internal/{config,setup,firstrun,prompts,tui,archtest}`, `pkg/{common,plugin,go-studs}`, `plugins/<name>/{plugin.go,cmd.go,subcommands,renderers,models}` | `ls` tree; `internal/plugins`, `internal/shared`, `internal/utils` are **doc.go-only stubs** |
| Cobra root | Root built in `main()`; persistent flags for verbosity (`MarkFlagsMutuallyExclusive`), `--light-mode/--dark-mode`, `--first-run`, `--dry-run` | `cmd/the reference/main.go:49-80` |
| Subcommands | Plugins registered explicitly (no discovery at runtime), `registry.CreateCommands()` -> `rootCmd.AddCommand` | `main.go:129-186` |
| Root TTY routing | `RootCommandRouter.ModifyRootCommand` wraps `rootCmd.RunE`; bare `the reference` -> first-run wizard or interactive selector; flags fall through to cobra | `cmd/the reference/root_router.go:106-201` |
| Theme init | `PersistentPreRunE` applies flags > cached theme.json > `$COLORFGBG` > `$TERM_PROGRAM` > dark | `cmd/the reference/theme.go:10-46` |
| TUI model | Single `AppModel` with a `state` enum; sub-state delegates to `promptOrchestrator.Update(msg)` (nested model) | `internal/tui/models/app.go:455-480` |
| Keymap/help | **No** `bubbles/key` or `bubbles/help`. Keys matched via `msg.String()` switch; root selector reads raw runes from `bufio` | `app.go:464-468`; `root_command_interface.go:441-470` |
| bubbles use | Only `bubbles/list` in feature/template selectors | `internal/tui/components/prompts/feature_selector.go:47,109` |
| Resize | `tea.WindowSizeMsg` stored on model (`m.width/m.height`) and **not propagated** to children | `app.go:459-462` |
| Program options | `tea.WithInput(os.Stdin)`, `tea.WithOutput(os.Stderr)` inline mode (no AltScreen); AAR printed to stdout after exit | `plugins/create/plugin.go:230-243` |

## 2. Config & secrets

| Concern | Pattern | Cite |
|---|---|---|
| Location | XDG `~/.config/the reference/config.yaml` with legacy `~/.the reference/config.yaml` fallback | `internal/firstrun/detector.go:23-31` |
| Format | YAML via `gopkg.in/yaml.v3`, header comment prepended | `internal/setup/config_writer.go:49-58` |
| Atomic write | temp file + `os.Rename`, cleanup on failure | `config_writer.go:62-74` |
| Permissions | **0644 file / 0755 dir everywhere** (`config_writer.go:46,64,100`, `internal/config/loader.go:100,114,133`, `theme_cache.go:87-90`) | AVOID for API keys |
| Secrets | None stored; mTLS cert paths only (`APP_CERT_DIR`) | `internal/private/certdiscovery.go:55` |
| Env overrides | Ad-hoc per package: `APP_VERBOSITY`, `APP_INTERACTIVE=false`, `APP_CREWS_*` | `internal/config/verbosity.go:154`, `setup/wizard.go:419`, `plugins/crews/config/config.go:150-161` |
| First-run detect | config missing or <10 bytes | `detector.go:52-65` |
| Wizard | Plain stdin `SimplePrompter` (`bufio`), not textinput/huh; 3 numbered paths; CI env vars force non-interactive | `wizard_optimized.go:75-107`; `prompts/simple.go:16-89`; `wizard.go:406-418` |
| Legacy residue | A second loader still points at `~/.dpx-web/config.yaml` | `internal/config/loader.go:20-21` |

## 3. Output modes

| Concern | Pattern | Cite |
|---|---|---|
| `--json` | **None.** Only inbound use is shelling to `go-status --json` | `plugins/app-status/api/go_status_proxy.go:69` |
| `--no-color` | No flag; honors `NO_COLOR` + `term.IsTerminal(stdout)` in one gate (`colorEnabled`) | `pkg/go-studs/rendering/color_enabled.go:11-28` |
| TTY detect | Two methods: `os.ModeCharDevice` (`root_router.go:219-226`) and `x/term` | mixed |
| Width non-TTY | `/dev/tty` ioctl, then stdin, then default | `pkg/go-studs/rendering/formatter_unix.go:15-40`; wizard clamps `<40 -> 80` (`wizard_optimized.go:33-36`) |
| Drift control | Goldens per width (`testdata/*_w60.golden`), ANSI-stripped, overflow-line counts frozen; `TestMain` forces color on | `plugins/my/renderers/oncall_golden_test.go:17-72`, `zz_colormain_test.go` |

## 4. Plugin / modularity seams

`CommandPlugin` interface: `Name/Description/Version/Create(deps)/Initialize/Cleanup/RequiredServices/OptionalServices` (`pkg/common/interfaces/plugin.go:10-24`); `AdvancedCommandPlugin` adds metadata, `Validate`, hooks (`:77-93`). `BasePlugin` embeds defaults (`pkg/common/plugin_base.go:42-94`). Registry is a map + `InitializeAll/CreateCommands/Cleanup` (`pkg/plugin/registry/registry.go:44-135`), plus a `PluginValidator` (`validator.go:143-300`) and an **fsnotify hot-reload manager** (`hotreload.go:14-25`) that compiled-in Go plugins cannot actually use. `internal/plugins/` is empty (`doc.go` only).

Verdict: the *interface + BasePlugin + explicit registration* trio is a good template for watchpost `views` and `providers`; drop the validator/hot-reload/`TUIRegistry` (`pkg/ui/display/tui.go:53-223`, `map[string]interface{}` everywhere).

## 5. Quality surface

| Item | Detail | Cite |
|---|---|---|
| `make verify` | gofmt, `go vet`, `go test -race -count=1`, staticcheck version-pinned (SA only, prod code), `go mod tidy -diff`, walkthrough currency | `Makefile:172-188` |
| Gates | G1 func ≤80 lines, G2 ≤15 files/pkg, G3 godoc coverage, G4 doc.go present+current, G5 file ≤800, alias transparency, renderer golden, G8 walkthrough churn; each has a positive-control test | `internal/archtest/gates_test.go:10-200`; `scripts/verify-walkthroughs.sh:1-9` |
| Color gates | no inline ANSI codes outside palette | `CLAUDE.md:216,317-344` |
| Coverage | `make test-coverage` HTML only, **no threshold** | `Makefile:66-72` |
| PTY tests | None; `tests/ui` renders to string and asserts | `tests/ui/cert_refresh_ui_test.go:14-403` |
| Charm version risk | `lipgloss.Style.Copy()` used heavily (deprecated no-op in v1, `internal/aar/formatters.go` 20x, `styles/colors.go` 13x); `bubbles v0.16.1` list API needs bump with bubbletea v1; `tea.Quit` usage is v1-compatible | grep counts |

## 6. Opinion

**REUSE**
1. **Root router** (`root_router.go:106-157`): bare invocation -> wizard-if-first-run -> TUI; flags fall to cobra. Exactly watchpost's `watchpost` / `watchpost setup` split. *Counter:* it hijacks `RunE` and calls `os.Exit(0)` mid-flow (`:100,147`); implement without exits.
2. **Atomic config write** (`config_writer.go:40-76`) + XDG path with legacy fallback (`detector.go:23-31`). *Counter:* must change perms to 0600/0700.
3. **Plugin interface + BasePlugin + explicit registration** (`interfaces/plugin.go`, `plugin_base.go`, `main.go:129-171`) for views/providers. *Counter:* `Create(deps interface{})` is untyped; use a concrete `Deps` struct.
4. **Single color gate** `NO_COLOR`+isatty (`color_enabled.go:28`) and width-keyed goldens with overflow pins (`oncall_golden_test.go`). *Counter:* goldens pin bugs as "known"; keep pin lists empty.
5. **Fail-closed `make verify` with positive-control gates** (`gates_test.go:31-37`). *Counter:* G8 walkthrough churn is heavyweight for a solo project; keep G1-G5.

**AVOID**
1. **Hand-rolled input loops and `fmt.Scanf` UIs** (`root_command_interface.go:115-127,441-470`): no resize, no `?` help, arrow keys "simplified". Watchpost should be one `tea.Program` using `bubbles/key` + `bubbles/help`. *Counter:* the reference's approach is zero-dependency and trivially testable as strings.
2. **0644 config and scattered `APP_*` env reads** (`config_writer.go:64`; five packages). Watchpost needs one typed config with 0600 and a single env-prefix loader. *Counter:* the reference stores no secrets, so 0644 was fine for it.
3. **Speculative infra**: hot-reload (`hotreload.go`), `TUIRegistry` with `map[string]interface{}`, `AdvancedCommandPlugin` hooks, empty `internal/{plugins,shared,utils}`. Violates the ≤40 MB / simplicity budget for no gain. *Counter:* the seams exist and cost nothing at runtime if unused.

Note on go-studs: the reference consumes it via a vendored copy with thin compat wrappers injecting the app palette (`internal/tui/components/go_studs_compat.go:21-25`) and a cache-store seam for theme persistence (`theme.go:24-25`). If watchpost imports go-studs as a module instead, that palette-resolver seam is the integration point to look at.
