# REVIEW Report — Watchpost CLI 0.9.0

| Field | Value |
|---|---|
| Phase | REVIEW (li-A2DH: BUILD → **REVIEW** → VALIDATE → SHIP → REFLECT) · SEV-0 · HUM LEAD |
| Entry | BUILD exit approved 2026-08-25 ("reports approved"); HUM LEAD GO for REVIEW; SQUASH publish approved |
| Scope | The whole 0.9.0 tree at `2f85e11` (post fit-and-finish UAT 102–115), four independent lenses, refute-before-report |
| Exit commit | `04d1e27` + the release-lens commit that follows (code, docs, installer, workflow, protocol) |
| Verdict | **PASS — ready for VALIDATE** (publish `main` + tag `v0.9.0` so the Linux protocol can install from GitHub) |

## Executive Summary

REVIEW asked one question of the finished tree: does the record match the code, and would a stranger's
first hour go as the docs promise? Four lenses (docs-vs-code, machine contract & licensing, code since
the last red-team, release readiness) raised **40 findings — 2 HIGH, 8 MEDIUM, 30 LOW/INFO**. Every HIGH
and MEDIUM is fixed and re-verified; the LOW/INFO items are fixed where a line sufficed and recorded
otherwise. No code defect surfaced that a user would have hit in the dashboard; the two HIGHs were a
first-run doc that would have sent users typing `y` into the FIRMS key line, and a `--json` envelope
that failed its own published schema (per-provider marine arrays published `null`). Both are the kind
of thing REVIEW exists to catch before a public tag.

Gates at exit: `make verify` GREEN (fmt, vet, `-race`, import/watermark lints + controls) · P10 0 live ·
golangci-lint 0 · staticcheck 0 · `a2dh validate` 18/18 · `make release-matrix` OK · `make install-test`
OK (SHA verified, tamper control fired) · live `--json` (89 KB, Oceanside) **VALID** against the
generated schema.

## Method

Each lens was a fresh agent with no shared findings, told to refute before reporting and to cite
`file:line` on both sides of every claim. Lenses: **D** docs-vs-code (README, CHANGELOG, `?`/About,
`--help`, extending.md, the broadcast scripts vs the UAT record); **M** machine contract & licensing
(`watchpost schema` vs live `--json`, checked-in schema, enums, exit codes, THIRD_PARTY_LICENSES,
attributions, public-tree hygiene, workflows); **C** correctness & regression over
`eaf7ade..2f85e11` (UAT 102–115) with `go test -race`; **R** release readiness (installer under
`dash`/`sh`, release workflow, first-run on Linux, Piper pins re-hashed against upstream, version
stamping, the Linux protocol). Findings were fixed in three commits and re-gated after each.

## Findings and Dispositions

| ID | Sev | Finding | Disposition |
|---|---|---|---|
| D1 | HIGH | README/CHANGELOG still described Setup's `y`/`n` step — `y` would land in the key line and fail `CheckKey` | **Fixed** (`6a4326e`): rewritten to the form (paste or leave empty, `tab`, `enter` saves, stored key shown) |
| M1 | HIGH | Live `--json` failed its own schema: `by_provider.*.marine.tides/currents` published `null` | **Fixed** (`04d1e27`): `normalizeMarine` on every provider copy; schema envelope test applies a marine fragment; live envelope validated |
| D2 | MED | "Piper, installed by `watchpost setup`" | **Fixed**: installed the first time you tune in |
| D3 | MED | `go run ./tools/nwrtable` did not execute as documented | **Fixed**: the curl + `-in CCL.js` form |
| D4 | MED | Named fires had no direction on air — `Incident` carried no point | **Fixed**: `Incident.lat/lon` from WFIGS geometry; "Timber is 12 miles east of your location…" |
| M2 | MED | Checked-in schema stale; `$id` pointed at a dead URL; nothing kept file and generator in sync | **Fixed**: `$id` = raw path of the checked-in file; `make schema`; `TestPublishedSchemaMatchesGenerator` byte-compares |
| M3 | MED | `providers[].status`, `warnings[].code`, radio `source/status`, provider `role` were free strings | **Fixed**: `enum:"…"` struct tags emitted into the schema; the assembler publishes `none`/`secondary`/`reference` defaults instead of `""` |
| M4 | MED | Exit codes undocumented in README; §10.2 said "≠ ok" though `off` is 0 by design | **Fixed**: README `--json` paragraph; §10.2 reworded |
| C1 | MED | Fire segments cached by position — under repeat a changed count replayed yesterday's audio | **Fixed**: content-keyed segments (`fire:<sha>`), notice included |
| R1 | MED | Linux validation protocol described the old wizard, a token-gated private repo, a setup-time voice install | **Fixed**: rewritten for the Setup window, first-tune Piper (paths, sizes, ceiling), masthead, fire mark and report, exit checks, no-ASCII note |
| R2 | MED | "replace directive" claim for go-studs in README, release.yml, checklist | **Fixed**: in-module packages with rewritten imports |
| C2 | LOW | Deck cloned whole pipeline snapshots to read one fire state | **Fixed**: `Assembler.FireFor` / `ProviderStatus` narrow reads; `fireReportOf` pure |
| C3 | LOW | Setup Q1: bare enter after a chosen hint replaced the choice | **Fixed** |
| C4 | LOW | Fire notice credited NASA FIRMS whenever a key was stored | **Fixed**: only when FIRMS answered ok |
| C5 | LOW | Masthead counted never-answered providers as ✔ on the first frames | **Fixed**: pending providers in the total only |
| C6 | INFO | Audio cache (24) smaller than a busy cycle | **Fixed**: 40 |
| C7 | INFO | NAME floor 24→19 on 115–119 cols | Accepted at UAT 110 (the marks block) |
| C8 | INFO | User theme files not validated for raw-SGR shape | Deferred to 1.0 |
| D5, D6 | LOW | CHANGELOG session count; shipped items unlisted | **Fixed** |
| D7 | INFO | Installer knobs `WATCHPOST_BASE_URL/REPO` not in README | Dev/test knobs; left |
| D8 | INFO | `°` vs `º` in the README key table | **Fixed** |
| M5–M8 | LOW | "replace" wording; machine paths in a spike; one employer mention in a quoted row; personal email in the brief | **Fixed** (`$GOMODCACHE`; paraphrased; "the personal address") |
| M9 | INFO | THIRD_PARTY_LICENSES.md over-includes test-only modules | Recorded (harmless) |
| M10 | INFO | CC BY 4.0 and FIRMS acknowledgement URLs absent | **Fixed**: README credits |
| M11 | INFO | Schema `1.0.0-rc` ratification still open | Recorded; README states `-rc`, additive only |
| M12, M13 | INFO | Harness hygiene confirmed; CI ran twice on PR branches | Confirmed; **Fixed** (concurrency group) |
| R3–R7, R13 | LOW | `WATCHPOST_VERSION=0.9.0` 404'd; no HTTPS/TLS pin; empty release could publish; PATH/`HOME` edge cases; musl binary reported as installed; cosmetic | **Fixed** (`install.sh`, `release.yml`: `fail_on_unmatched_files`, release notes, 20-minute timeout) |
| R8–R12 | INFO | Version reads `0.9.0`; no `--ascii` flag; workflow SHAs verified; Piper manifest re-hashed against upstream; tzdata embedded | Confirmed |

## Evidence

- Tests added at REVIEW: schema marine fragment, `TestPublishedSchemaMatchesGenerator`, `TestFireForAndProviderStatusReadNarrowly`, `TestFireReportOf` (FIRMS credit), fire-report direction golden, masthead pending semantics (fixture). `go test -race ./...` clean.
- Live: `watchpost report 92057 --json` validated against `watchpost schema` (89,499 bytes, VALID). `make install-test` under `dash -n`/`sh -n` clean; SHA verified; tamper control fired.
- Identity for publish: all 168 commits author as the personal address; `gh` active account `branden-thompson`; the `github.com-personal` SSH alias (missing until REVIEW) added and proven with `git ls-remote`; `main-publish` prepared as one orphan commit (`2a58319`, 246 files, no harness files).

## Conditions carried into VALIDATE / SHIP

1. Publish: push `main-publish` as `main`, tag `v0.9.0`, let the release workflow build and publish; then the Linux protocol on HUM LEAD's laptop (`07-readiness/linux-validation-protocol.md`).
2. M8 soak (1 hour, radio on) — at VALIDATE.
3. Schema `1.0.0-rc` → `1.0.0` ratification — HUM LEAD, any time before 1.0.
4. Anything the Linux run finds ships as `v0.9.1` (fix-forward), never a re-cut tag.

## Source Documents

`08-reports/build-report.md` (+ addendum), `08-reports/red-team-build.md` (rounds 1–3), `04-development/b3-uat-log.md` sessions 99–116, `07-readiness/release-checklist.md`, `07-readiness/linux-validation-protocol.md`, `pkg/schema/watchpost-report.v1.0.0-rc.schema.json`.
