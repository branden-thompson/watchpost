# Extending watchpost

> **Status (as of 0.9.0):** everything named here exists and the steps are verified against the
> shipped code (Documented-Commands-Execute rule). The planned `app/registry.go` / `View` interface
> was not built — one dashboard model plus keybindings-as-data covered v0.x; see architecture
> §11.9. When a second top-level view arrives (the playlist cycler, B6), the registry comes with it.

watchpost is organized so you can find a feature by its name (`domains/…`) and extend it by
touching **one folder plus one wiring line**. Two invariants keep everything honest — the import
linter in `make verify` enforces the first; `term.Merge` plus its tests enforce the second:

1. **Data flows one way:** domains write `platform/snapshot`; `modes/` (all rendering: TTY and
   `report`) read ONLY the snapshot — never a domain. Anything the dashboard needs from a domain
   arrives as an app-provided hook in `tty.Config` (radio, voices, hydrate, spectrum).
2. **Keybindings are data** (D-15): `modes/tty/dashboard.go` `defaultKeyMap()` is the only place a
   key is named; users override any of them via `[keys]` in config.toml; `?` is reserved for help,
   which renders from the merged map so it is always truthful.

---

## Walkthrough 1 — add a UV-index line to the location detail modal

Goal: show `Harmonized.UVIndex` in the detail view (`enter`).

| Step | File | What you do |
|---|---|---|
| 1 | `platform/snapshot/types.go` | Nothing — `Conditions.UVIndex *float64` already exists (§10.1). A nil pointer means "provider has no value" and renders `n/a`. |
| 2 | `platform/render/units.go` | Only if no formatter fits: add `UVIndex(v *float64) string` next to `Distance`/`Knots`/`TideHeight` — `platform/render` is the ONLY package that may import go-studs, and every formatter has an `n/a` path and a width-bound test. |
| 3 | `modes/tty/detail.go` | In `detailLines` (the modal body builder), add the row where the mock puts it, using the formatter from step 2. Rows are label/value pairs aligned by the modal's shared inset; follow the neighbours. |
| 4 | Tests | Extend the detail-modal test with a sentinel UV value (e.g. `7.3`); the report-mode parity test then asserts the same value appears in `--json` (M5). |

You never touch providers, the scheduler, or `modes/report`.

## Walkthrough 2 — add a keyed provider (the shipped pattern: NASA FIRMS)

Goal: a new source that needs a user key. `domains/fire/firms` is the worked example (B5).

| Step | File | What you do |
|---|---|---|
| 1 | `domains/<domain>/<name>/<name>.go` | Implement `snapshot.Provider`: `ID()`, `Domains()` (`["fire"]`, `["weather"]`, …) and `Fetch(ctx, req)` returning a `Fragment` with `PerLocation`. Use `platform/httpx` for every request — a 32-hex path segment or a `key`/`api_key` query value is redacted from every error and log line; cache with `httpx.TTL(d)` where the server sends no cache headers. |
| 2 | Key wiring | Take the key in the constructor **and** behind a lock with `SetKey` / `Enabled()` (`firms.go`): the Setup window keys a provider while the dashboard runs. Validate the key's shape before it is stored or used (`firms.CheckKey`) so the redaction never depends on a well-formed paste, and scrub it from transport errors yourself (`redactKey`). Users store it in the Setup window (`s`) or `[providers.<name>]\nkey = "…"` in config.toml (0600); `app` reads `cfg.Providers["<name>"].Key`. Keys never appear in output: the no-secret golden asserts it. |
| 3 | `app/fire.go` (or a sibling) + `app/dashboard.go` | Build the provider in one place (`fireProviders`), add it to the pipeline's provider set (`livePipelines.fire` / `marine` / `providers()`), give `newAssembler` its attribution case (About renders it), and add it to `credits.go` (≤ 52 cells). Register it **always**; mark it `off` while unkeyed with `Assembler.SetInactive` (`livePipelines.markFIRMS`) so the API status never says "ok" for a feed that contributes nothing. |
| 4 | Scheduler | Cadence is per-**kind**: an existing kind (weather, obs, marine) rides its tier for free; a **new** kind needs a `snapshot.Kind`, a tier line in `startPriority` and one in `newFor` (RECENT), and `Domains()` on the provider — B5 added `KindFire` at 10 / 15 minutes. |
| 5 | Tests | A fixture-backed `Fetch` test (httptest, recorded body) that also asserts the request shape; a key test (stored trimmed, malformed refused without echo); and, for a new kind, the assembler merge test. |

Schema note: `by_provider` is additive — a new provider id needs no schema version bump (schema
v1.0-rc policy, architecture §10.3). A new top-level block (like `fire`) is a schema change and
lands with its `modes/report` lines and parity fixtures.

## Walkthrough 3 — add a radio control

Controls live where they act: a key in `defaultKeyMap()`, a case in `toggleRadio`, a chip in
`radioControlLines`, a method on the `tty.Radio` interface, and its implementation on
`app/radio.go`'s `radioDeck` — which runs off the update loop (`withCmd`/`takeCmd`) and reports
back through `RadioStatusMsg`. `[m] Mode` (UAT 97) is the smallest complete example.

## Rules you inherit for free

- **Every colour is a theme token.** Views call `render.Tint(text, render.Tok(render.SomeToken))`;
  the tables' own palette is three tokens (`table.header`, `table.muted`, `table.name`) the
  seam applies through go-studs' `HeaderColor` / `CellStyles` with the kit's automatic styling
  switched off (`NoAutoStyle`), so `NO_COLOR` is honoured by one gate and a user theme file can
  restyle the whole frame. A new colour = a new token with a value in every built-in theme and a
  row that passes `TestThemeTokenContrastAA`.
- **A new window is one `modal` constant** plus a case in `modalView` (what it draws), `modalWidth`
  (how wide) and, if it scrolls, `modalLines`; a key that opens it goes through `toggleModal`.
  Exclusivity is the type's: the rendered-frame test (`modal_test.go`) adds the window to its
  table and proves nothing stacks.
- **go-studs is patched, never edited.** An approved change to the kit is a
  `third_party/go-studs/patches/NNN-name.patch` with a row in `LOCAL_CHANGES.md`;
  `scripts/sync-go-studs.sh` re-applies the stack on every sync and refuses a drifted upstream.

- `make verify` runs fmt, vet, `-race` tests, import-direction lint, watermark lint, and the
  gates' self-tests. Run it before every commit, then `golangci-lint run ./...` and `staticcheck ./...`.
- New render primitives are built **on demand** — add them when a view needs them, never
  speculatively (PD/§10.10.5).
- Every package carries a package comment stating its contract; follow the pattern in
  `platform/config`, `platform/term`, `platform/httpx`.
- **Recording alert fixtures:** `go run ./tools/alertrec -zones CAZ554,CAC073 -every 20s -out feed.jsonl -for 10m`
  polls the live NWS feed into a replay fixture; drop it in `domains/alerts/testdata/replays/` and
  the M2/M3 harness replays it (`domains/alerts/replay_test.go`). The committed `basic.jsonl` is
  synthetic.
- **Refreshing the transmitter table:** `curl -o CCL.js https://weather.gov/source/nwr/JS/CCL.js && go run ./tools/nwrtable -in CCL.js` regenerates
  `domains/radio/stream/transmitters.csv` from the NWS `CCL.js`.
- Guard stateful paths with `platform/invariant`; pure total functions may be
  left without checks when a test pins them.
