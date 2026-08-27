# Q2 build log — Structure and readability (`v0.9.6`, with Q3)

**Batch:** Q2 of the plan of record v3 (`03-architecture-design/quality-pass-plan.md` §3).
**Approval:** file map `04-development/q2-file-map.md` approved 2026-08-26 ("seems fine");
Q2 began once `v0.9.5` was green (release run `af2097b` → tag `v0.9.5` at `e2c5b6f`).
**Branch:** `feature/watchpost-performance-quality-pass`, commits `dac24c9` → this log.
**Status:** APPROVED 2026-08-26 ("Exemption approved; go for Q3 build") — `platform/declset` row ratified, M4 restated to ≤ 54.

## 1. What changed, and why (junior-first)

Q2 changes no behaviour. It moves code so that the next person can find it.

- **Five big files became twenty-six small ones.** `modes/tty/dashboard.go` (3,386 lines) is
  now 14 files named for what they hold (`detail_fire.go`, `status.go`, `setup.go`, …);
  `platform/render/render.go` (1,199) is `units / table / sgr / panel / text`; `app/dashboard.go`
  (881) and `app/radio.go` (753) are nine files; the NWS provider and the snapshot assembler each
  split along their natural seams. Every file starts with a three-line header that says what it
  holds and what it does not.
- **A proof that nothing moved but the text.** A new package, `platform/declset`, lists every
  declaration in a package (name, receiver, kind) and compares it with a golden file. The golden
  was captured *before* each split and checked *after*; a split that lost, renamed or duplicated a
  declaration fails `TestDeclarationSetUnchanged`. It stays in the tree so a future move has the
  same guard.
- **The go-studs seam is one file.** `platform/render/table.go` is the only file that imports
  go-studs (Q4 needs that isolation to patch or replace the table without touching colour, panels
  or text).
- **The one non-move.** `app/pipelines.go` now has `priorityTiers()` and `recentTiers()` — the tier
  tables were inline in the two `start*` functions (DISCOVER L3-F14). Behaviour is identical; the
  cadence table `app/testdata/cadences.md` pins every tier's cadence, retry and batch so it stays
  identical.
- **A flow map for the first read.** `docs/where-things-happen.md` answers "where does X happen"
  with `path/file.go:Func` for 27 events and defines the project's vocabulary (snapshot, fragment,
  tier, lane, memo, sweep, gauge, dump, deck, mount, synth, token, seam) and every record-ID
  prefix. A test opens every named symbol; the page cannot drift silently.
- **The report command tests without the network.** `cmd/watchpost/root.go` takes the report
  fetch as a parameter (`newRootCmdWith(fetch)`) instead of a package variable, so
  `report_test.go` covers plain / `--json` / `--verbose` / exit 2 / resolve-error against a fake.
- **Six ADRs written down.** The decisions the plan took in prose now have one page each
  (`03-architecture-design/adr-01…06.md`): dump trigger, disk tier, RECENT scheduler (C3 = A),
  go-studs patches, no `SetMemoryLimit`, targets.

## 2. Files touched

| Area | Before | After (lines) |
|---|---|---|
| `modes/tty` | `dashboard.go` 3,386 | `dashboard.go` 583 · `view.go` 234 · `radio_panel.go` 376 · `alerts.go` 316 · `setup.go` 305 · `body.go` 292 · `detail_marine.go` 279 · `modal_location.go` 215 · `status.go` 190 · `detail.go` 189 · `detail_fire.go` 186 · `modal_chooser.go` 169 · `nav.go` 147 · `help_about.go` 88 — tests mirrored one-to-one |
| `platform/render` | `render.go` 1,199 | `table.go` 538 (the seam) · `panel.go` 207 · `sgr.go` 190 · `text.go` 159 · `units.go` 136 · `render.go` 16 (package doc) |
| `app` | `dashboard.go` 881 · `radio.go` 753 | `dashboard.go` 354 · `pipelines.go` 323 · `refs.go` 103 · `themes.go` 70 · `debug.go` 64 · `resolve.go` 49 · `radio.go` 414 · `voices.go` 265 · `radio_queue.go` 100 |
| `domains/weather/nws` | `provider.go` 768 | `provider.go` 174 · `forecast.go` 202 · `observations.go` 174 · `points.go` 145 · `alerts.go` 120 |
| `platform/snapshot` | `assembler.go` 664 | `assembler.go` 314 · `harmonize.go` 284 · `merge_fire.go` 80 |
| new | — | `platform/declset/{declset.go, declset_test.go}` · `docs/where-things-happen.md` · `cmd/watchpost/{docs_test, report_test}.go` · `app/{logic_test, cadence_test, wiring_test, pipelines_test}.go` + `testdata/cadences.md` · `modes/report/golden_test.go` + `testdata/plain-80.golden` · `platform/invariant/invariant_test.go` · `*/testdata/declset.txt` (tty, render, app, nws, snapshot) · ADR 01–06 |
| docs | — | `docs/extending.md` (paths) · `watchpost-cli/03-architecture-design/architecture.md` (§1 node, §4 as-built, §11) · `README.md` (Diagnostics) |
| commits | `dac24c9` tty · `065612c` render · `715b7a4` app · `e91d6e2` nws + snapshot | each one package, each a pure move certified by its declset golden |

Largest non-test file after Q2: `modes/tty/dashboard.go` at 583 lines (was 3,386). The mechanical
splitter (`splitgo`, a 200-line Go program driven by `range from to file.go` / `name Recv.Name file.go`
maps) lives in the session scratchpad, not the tree — it was single-use and the declset pin is the
durable guard.

## 3. Tests first (the pins)

| Pin | Guards |
|---|---|
| `TestDeclarationSetUnchanged` (tty, render, app, nws, snapshot) | the split moved every declaration once and only once |
| `TestWhereThingsHappenNamesRealSymbols` | every `path:Func` in the flow map is declared where it says |
| `TestTierCadences` + `testdata/cadences.md` | the tier extraction changed no cadence, retry or batch (L3-F14) |
| `TestReportPlainJSONVerboseAndExitCodes` | the report command's four output paths and both error paths |
| `TestPlainReportAt80Columns` + golden | the plain report's layout (JD-1) |
| `platform/invariant` 4-case table | the invariant helper's on/off/panic/log behaviour (was 0 % covered) |
| `app/logic_test.go` | `parseLatLon`, refs ↔ config round trip, `restoreRecent`, `fullyPopulated`, `toKeyMap`, stale warnings, deck label |
| `app/wiring_test.go`, `app/pipelines_test.go` | theme loading, assembler construction, debug address, resolver hooks' failure paths, cache dirs, `deriveTag`, `seedRecent`, tides client, pipeline nil-safety, timing report |
| theme-token pins (`body_test.go`, `radio_panel_test.go`) | now assert through `render.Tok(...)` instead of literal SGR, so a theme edit fails the right test |

## 4. Before / after

| Measure | Before Q2 (`2e3df8c`) | After Q2 |
|---|---|---|
| largest file | 3,386 lines | 583 |
| files > 500 lines (non-test) | 5 | 2 (`tty/dashboard.go` 583, `render/table.go` 538) |
| go-studs importers | 1 file, inside a 1,199-line file | 1 file, `table.go`, alone |
| coverage `app` | 29.0 % | **38.6 %** (target ≥ 35 %) |
| coverage `cmd/watchpost` | 32.8 % | **56.9 %** |
| coverage `platform/invariant` | 0 % | 100 % |
| coverage tty / render / nws / snapshot / report | 90.2 / 82.1 / 83.8 / 85.2 / 82.5 % | unchanged (pure moves) |
| frame allocation budget (`TestFrameAllocBudget`) | 10,044 / 15,539 / 20,031 | unchanged, green |
| symbols the flow map names | — | 27 events, 3 vocab tables, all checked |

## 5. Bounds stated (§0.8)

- No behaviour changes; the only structural edit (`priorityTiers` / `recentTiers`) is pinned by
  the cadence golden and the app declset golden was re-captured for it, with the note in the commit.
- `platform/declset` is test-support code (imported only by `_test.go` files); it adds no runtime
  surface and no dependency.
- The report seam is a constructor parameter; `newRootCmd()` is unchanged for callers.

## 6. Decisions and non-decisions

- **Splitter stays out of the tree.** Single-use; the pin is what matters.
- **`--ascii` stays Q3's.** Approved as "wire in Q3, default; flagged for veto at the Q3 gate".
- **Compass 8 → 16 excluded** from the pass (HUM LEAD); only via a specific approval request.
- **Coverage floor.** `app` was raised by testing its pure helpers, not by exercising
  `RunDashboard`; the composition root itself remains the untested 60 % and is exercised by the pty
  smoke and the soaks, not unit tests. That is the honest shape of the number.
- **Synth untouched (R6).** The one diff in `domains/radio/synth` is a comment correcting a stale
  size estimate to the measured figure (L1-F13); no code.

## 7. Gate

| Check | Result |
|---|---|
| `make verify` (fmt, vet, tidy, vuln, race, lint-imports, lint-watermark, gate-controls) | ALL GATES GREEN |
| `make tidy` | `go.sum` pruned of stale lines; verified |
| `make vuln` (Go 1.27.0) | no vulnerabilities |
| `make alloc-budget` | green — `modes/tty` 2.9 s, budgets unchanged |
| `make p10` | **0 live · 0 unmatched** · ledger 120 = 66 kit + **54 non-kit** · snapshot `07-readiness/p10-q2.json` |
| `a2dh validate` | 18/18 |
| declset goldens (5 packages) | green — every split certified |
| flow-map symbol test | green — 27/27 |
| coverage `app` ≥ 35 % | **38.6 %** |
| docs touched for every moved symbol | `docs/extending.md`, `architecture.md` §4, `where-things-happen.md`; grep for old paths finds only historical records |

**Decision for HUM LEAD (the one deviation).** Non-kit ledger entries stand at **54 against the
ratified 53**. The extra row is `platform/declset` (P10-05 invariant density, package-scoped):
`Set`/`Write`/`Compare` carry the real checks; `declsOf`/`recvType`/`Diff`/`Golden` are pure AST
listing and set arithmetic where a quota check would assert nothing — the same pattern as the
ratified `term`/`httpx` pure-helper rows. Options: **(a) ratify** and restate M4 to ≤ 54 (recommended:
the package exists to make future moves safe, and padding it with no-op invariants would be
exactly what P10-05's intent forbids); **(b) refuse** — I inline the four helpers into `Set` and
`Compare`, which meets the meter and makes the package harder to read. Every other Q2 addition
landed with zero new exemptions.

## 8. Carried forward

- 24 h idle soak (Q0 apparatus, PID in the session scratchpad, port 6060) ends ~2026-08-27 15:55
  UTC; its σ result and the Q0 5-min sampler's 24 h series go into the Q3 log.
- Arch half of the relay proof: HUM LEAD's `WATCHPOST_LIVE=1 go test ./app -run LiveRelay -v`,
  deferred ≤ Q7 (RP-10).
- `domains/radio/synth` under `go test -race -count=3` exceeds 600 s (count=1 is 3 s); not CI's
  shape; noted for Q5's test-hygiene pass, not touched (R6).
- Q3 next (render path, app side): tick predicate, `d.layout()`, body memo, colour-off + ASCII
  goldens, `TestFrameHonoursNoColorUnderColorTerm` known-failing pin, HMS streaming parse,
  geodata once, publish consolidation, `--ascii` wiring. `v0.9.6` ships Q2 + Q3 together.
