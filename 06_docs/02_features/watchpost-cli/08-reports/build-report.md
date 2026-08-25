---
title: "Build Report — Watchpost CLI (B0–B4, 0.9.0 candidate)"
date: 2026-08-25
phase: BUILD
sev: 0
collaboration: HUMAN LEAD
---

# Build Report — Watchpost CLI

## Executive Summary

Watchpost is a terminal weather station: a watchlist dashboard of National Weather Service data,
marine conditions and tides, and NOAA Weather Radio — live relays where they exist, and the
location's own forecast read aloud where they don't. Five builds (B0–B4) and 98 UAT sessions
produced the 0.9.0 candidate: the product designer drove every session from the terminal, and the
infrastructure (scheduler, cache, audio, voices) followed the UX findings. The BUILD exit ran two
adversarial rounds over six lenses; every finding is fixed or dispositioned below. The code is
ready to tag `v0.9.0`; the one gate that stays open by design is the Linux end-to-end run on a
second machine, which is the 0.9.0 → 1.0 validation.

## Context

| | |
|---|---|
| Project | watchpost — `github.com/branden-thompson/watchpost` (public, MIT) |
| Phase | BUILD (B0 scaffold · B1 data · B2 locations/setup · B3 TTY · B4 radio) → REVIEW |
| SEV | 0 — HUMAN LEAD; FULL GIT / DOCS / REPORTS / DIAGRAMS / TDD |
| Branch | `feature/watchpost-cli`, 145 commits, HEAD `eed7b3f` |
| Date range | 2026-08-23 (B0) → 2026-08-25 (exit) |
| Version | `0.9.0` (tag pending; stamped via `-ldflags` from `git describe`) |

## Implementation Log

### Platform (`platform/`)
**Files:** `snapshot/{types,assembler,foreach}.go`, `sched/sched.go`, `httpx/{httpx,cache}.go`, `config/config.go`, `render/{render,theme,themes,spectrum}.go`, `term/term.go`, `tz/tz.go`, `geo/geo.go`, `astro/`, `invariant/`
**Purpose:** the immutable snapshot the whole app reads (assembler with fail-soft fragments, last-good data, bounded warnings, location dedupe); the tiered scheduler (retry-before-cadence, incremental `Update`, per-provider publish); the HTTP client (token bucket with a priority lane, memory + disk cache honouring server headers, singleflight, bounded disk reads, body caps, redaction, self-healing cache entries); atomic 0600 config with favourites, the RECENT stack, theme, voice and radio mode; the render seam (the only go-studs importer — semantic theme tokens, validated theme files, the visualizer bars, `Plain` for outside text); memoized time zones; haversine.
**Status:** complete.

### Domains (`domains/`)
**Files:** `weather/nws/{provider,marine}.go`, `marine/{ndbc,coops}/`, `locations/{resolver,coverage,geodata,openmeteo}`, `alerts/` (replay harness), `radio/{stream,player,synth,spectrum}/`
**Purpose:** NWS observations / forecasts / hourly / alerts / products / coastal waters with a station fallback chain and county/zone lookups; NDBC buoys and CO-OPS tides, water levels and currents; offline city/zip geodata with NWS-coverage refusal for non-US places and an online geocoder fallback; the radio: a 1,035-site NWR transmitter table, community relay directories, an ICY/MP3 player with stall watchdog, mount failover and 403/404 honoured, a synthesized broadcast (products → narration rules → segments → `say` / Piper, render-ahead, repeat, mid-broadcast voice hand-over), a SHA-256-pinned Piper installer, and a spectrum analyzer feeding the visualizer.
**Status:** complete for 0.9.0; deferred items in *Known Technical Debt*.

### Modes (`modes/`)
**Files:** `tty/dashboard.go` (+ tests), `report/report.go`
**Purpose:** the dashboard (watchlist + RECENT stack, detail / alert / help / About / status / theme / voice / lookup / add / remove modals, compact layouts, radio player with Repeat: Off | One | Watchlist, Mode: Synth | Nearest Relay, visualizer, voice chooser with preview) built to the ASCII mocks; the one-shot plain / JSON report with a published schema. `modes/*` never imports `domains/*` (linted).
**Status:** complete.

### App (`app/`) and CLI (`cmd/watchpost`)
**Files:** `app/{app,dashboard,radio,setup,credits,themes}.go`, `cmd/watchpost/{main,root}.go`
**Purpose:** wiring — pipelines and tiers, the radio deck (tune epoch, dwell, fallback), setup wizard, credits with the safety framing, persisted preferences; cobra tree with typed exit codes; embedded tzdata.
**Status:** complete.

### Release and docs
**Files:** `scripts/{install.sh,install-test.sh,sync-go-studs.sh,lint-*.sh}`, `Makefile`, `.github/workflows/release.yml`, `.github/PULL_REQUEST_TEMPLATE.md`, `README.md`, `CHANGELOG.md`, `LICENSE`, `third_party/go-studs/`, `docs/extending.md`, `06_docs/…/{04-development,05-debugging,06-key_learnings,07-readiness}`
**Purpose:** `curl | sh` installer with SHA-256 verification and a local end-to-end test (tamper control included); the cross-compile matrix with version stamping; a tag-triggered release workflow with SHA-pinned actions; the UI kit carried in-repo under its MIT licence; the seven-folder documentation set.
**Status:** complete; tag and first workflow run pending.

## Test Results

`go test ./...` — 274 test functions across the module's 29 packages, all passing; `-race` clean; six fuzz
targets (seed corpus in every run; one crasher found and fixed at exit). Coverage by package
(statements): tty 90 %, player 90 %, spectrum 97 %, stream 88 %, nws 85 %, coops 88 %, ndbc 86 %,
httpx 87 %, render 82 %, snapshot 81 %, sched 80 %, synth 66 %, config 68 %, app 25 % (wiring —
exercised by the pty runs), cmd 30 %, geo/tz 100 %.

| Package | Tests | Status |
|---|---|---|
| modes/tty | 60 | pass (anatomy, every UAT behaviour, radio controls, fail-soft UX) |
| domains/radio/{player,synth,stream,spectrum} | 58 | pass (+ 6 fuzz targets) |
| platform/{httpx,snapshot,sched,render,config,term,tz,geo,astro} | 86 | pass |
| domains/{weather,marine,locations,alerts} | 47 | pass (fixture-backed; M2/M3 replay harness) |
| app, cmd, modes/report, pkg/schema, tools | 16 | pass |

## Build Status

`make build` → `dist/watchpost` (version stamped); `make release-matrix` → darwin/{arm64,amd64},
linux/{amd64,arm64}, windows/amd64 with `checksums.txt`, CGO off, ~4 s; `make install-test` →
install from a local server + tamper control, OK. Quality gates at exit: `make verify` ALL GATES
GREEN (fmt, vet, race, import direction, watermark, positive controls); P10 0 live findings
(exemptions listed for ratification in the red-team verdict, Appendix A); framework validate
18/18; `golangci-lint` 0; `staticcheck` 0 (the upstream-governed copy excluded from both).

## Performance (real TUI in a pty, 133×44, 10 favourites + 50 recent)

| | launch → full view | threads | RSS |
|---|---|---|---|
| Warm cache | 550 ms (target ≤ 3 s) | 23 | 78 MB |
| Cold cache | 1.1 s (target ≤ 8 s) | 20 | 79 → 97 MB while the cache fills |

## Deviations from Plan

| Planned | Actual | Reason |
|---|---|---|
| View registry + layered KeyMap (§6, D-15, T-K) | One dashboard model; keybindings as data with a config override layer | One view for v0.x; the D-15 promise (rebindable, `?` locked, truthful help) holds; the registry arrives with the playlist cycler (B6) |
| Dependency pins (bubbletea 2.0.9, lipgloss 2.0.6, bubbles 2.2.0, oto 3.4.1) | bubbletea 2.0.3, lipgloss 2.0.2, bubbles 2.0.0-rc.1, oto 3.5.0-alpha.11 | oto: pure Go on Linux (UAT 76). Charm: what B0 resolved against go-studs; 97 sessions ran on it. **Decision for HUM LEAD: accept as the 0.9.0 pin row, or re-pin at 1.0** |
| PD-4: two height breakpoints | `tableBreakpoint = 20` + a measuring `compact()` | HUM-LEAD-directed in UAT 34/47/49 |
| `platform/audio` | PCM path in `domains/radio/player`; directory removed | one consumer |
| Covering relay plays live by default (UAT 78) | Synth by default; `[m] Nearest Relay` is the explicit pick (UAT 97) | HUM LEAD: "should default to the synth path" |
| C-6″ no installs beyond the OS | Piper (~85 MB) installs on Linux/Windows at setup or first tune-in, pinned, with progress | The text-ticker / `--tts-cmd` escape hatches are 1.0 items |
| `[p] Pin` | Retired (UAT 93) | Repeat: Watchlist is how the player follows the list |
| go-studs via private module + RS-19 `go mod vendor` | `third_party/go-studs` folded into the main module (no sub-module, no `replace`); other deps from the module proxy; licence texts shipped | HUM LEAD 2026-08-25; the old `replace` was an absolute path on one Mac; a sub-module `replace` broke `go install` |
| The development harness lives in the repo | Untracked and ignored (`_a2dh/`, `AGENTS.md`, `CLAUDE.md`, copilot instructions, `.a2dh*`); generic PR template | HUM LEAD 2026-08-25: no private-employer addresses or machinery in the public tree |

## Known Technical Debt (the 1.0 plan)

| Item | Rationale | Resolution |
|---|---|---|
| B5 (remaining): keyless global weather provider (non-US), schema v1.0 ratification, `fill_from` staleness cutoff | 0.9.0 is US-only and says so. Fire (HMS/WFIGS/FIRMS) was pulled forward into 0.9.0 at HUM LEAD's direction (addendum; UAT 99–100) — the FIRMS prompt question is answered by building its consumer | 1.0 |
| B6: playlist cycler, national summary, `--ascii` / `--no-animation` flags, `report --every` (D-18/G-9, the screen-reader live surface), 1-hour soak, cross-terminal matrix | `report --every` was a ratified v0.1 requirement — first in line | 1.0 (first item) |
| Radio: WAT alert tone + interrupt, Live silence detector, text ticker, `--tts-cmd`, transmitter-table checksum + date, 150-entry abbreviation TSV | Deferred in §11.7/§11.9 | 1.0 |
| Great Lakes levels, Tahoe (HUM CALL), M-V6b setup-as-modal mock | Queued in the infra ledger / UAT log | as prioritized |
| Retry-After honoured in httpx; offline rows say "offline" instead of shimmering; singleflight leader ctx (C-7); orphaned `.tmp` cleanup | LOW findings accepted at exit, recorded in the red-team report | 0.9.x |
| M8 soak (1 h, radio on), M9 CPU windows | Instrumented, not measured over an hour | VALIDATE |

## Build Statistics

| | |
|---|---|
| Go files | 92 (+ 41 test files; + 35 in `third_party/go-studs`) |
| Lines | 13.9k source · 7.9k test |
| Test functions | 274 (+ 6 fuzz targets) |
| Commits (feature branch) | 145 |
| UAT sessions | 98 (B3 1–75, B4 76–98) |
| Red-team | 2 rounds · 6 lenses · 88 findings · 75 fixed · 13 declined / deferred with rationale |
| P10 exemptions | 48 through B4 UAT + 9 at exit + the upstream-governed copy (per file/symbol); the file is local-only |

## Decisions (this exit)

| # | Date | Decision | Prompt (verbatim) | Rationale |
|---|---|---|---|---|
| D-23 | 2026-08-25 | Repository public | "Repo - public is fine" | installer URL as written; internal hostnames/paths redacted from docs first |
| D-24 | 2026-08-25 | Code licence MIT | "Code License - MIT? (Free for use with clear attribution)" | MIT's notice-retention is the attribution |
| D-25 | 2026-08-25 | go-studs carried in-repo | "for go-studs, can make local copy internal to app for now" | `third_party/go-studs` (MIT, same author) via `replace`; refreshed by `scripts/sync-go-studs.sh` |
| D-26 | 2026-08-25 | Version 0.9.0 | "we'll likely be at a 0.9.0 tag" | tag on the release commit; the Linux run is the 1.0 gate |
| D-27 | 2026-08-25 | No private-employer references in the public tree | "Let's remove any/all private-employer references and if necessary port code over as generic components … we should not expose any private addresses or machinery" | development harness untracked; docs, comments and the copied kit scrubbed to generic wording; the access research replaced by a provenance note; `main` to be published by squash because three historical commit messages name internal tools |

| D-28 | 2026-08-25 | Reports approved; **BUILD exit NOT approved** — initial FIRMS fire data joins 0.9.0 (B5 starts next); a second exit pass adds addendums to these reports | "Reports Approved; Build Exit NOT Approved; we need to include initial FIRMS (fire data) in 0.9.0 to augment the comprehensive value add … Once done we'll do another build exit pass and add to these reports as addendums" | the fire feature builds on the remediated foundation; the FIRMS-prompt question (D-27 era) is answered by building the consumer |

Process note: a draft of this report was swept into commit `c15bc4d` by a `git add -A` before it
was presented. The version presented here supersedes it; the red-team verdict was never committed
before presentation.

## Critical Analysis (red-team)

> red-team: SHIP-WITH-CONDITIONS · multi-agent · scope:project · personas:[InfoSec, Performance, Linux/cross-platform]

Two rounds, six lenses, 88 findings: Critical 4 (all fixed) · Important 41 (39 fixed; 2 escalated
to HUM LEAD — the dependency pin row, the FIRMS prompt in setup) · Minor 43 (32 fixed; 11 declined
or deferred with rationale). The full findings table with severity, evidence and disposition is the
standalone artifact `08-reports/red-team-build.md` (the durable copy); its conditions: publish
`main` by squash; run the Linux validation protocol; measure the 1-hour soak (M8) at VALIDATE.

## Addendum — B5 fire and the second BUILD-exit pass (2026-08-25)

HUM LEAD approved the reports above but held the BUILD exit: "we need to include initial FIRMS (fire data) in 0.9.0". B5 was built data-first (UAT 99), setup was rebuilt as a dashboard window and the fire UI landed (UAT 100), and a second red-team pass (Round 3, four lenses) was run and remediated (UAT 101).

**Delivered.** `domains/fire` (rules, geometry, clustering) + three `KindFire` providers — HMS (keyless NOAA KMZ, parsed once per change, memoized), WFIGS (NIFC incidents), FIRMS (NASA VIIRS, keyed; always registered, `off` until keyed, keyed live from the Setup window); assembler merge with `as_of`; `[fire]` config validated at load; ▲ row mark, FIRE section in Location Details (bearing, distance, MW, satellite, age; incidents; fire-weather alert), report lines, JSON `fire` object; Setup window (`[s]`, first run, `watchpost setup`); Location Details chip row; README Fire section with FIRMS key steps; CHANGELOG; architecture §11.10; B5 build log.

**Evidence.** Tests: every fire package over in-test fixtures (a KMZ built in the test), assembler merge, config validation, render mark, detail section (colour on), report parity, Setup window flows, FIRMS fail-soft, HMS memo/cap/root/skip. Live: Mineral Wells TX (300+ hotspots, Ross 50,000 ac; with HUM LEAD's key 203 FIRMS + 166 HMS points merged, `firms: ok`), Owyhee County ID (Flint 3,344 MW at 2 km). Perf: HMS parse 118–124 ms / 89 MB per archive change (was ×50 per 15 min before the memo); 60-location proximity loop 48 ms; mergeFire 3.7 ms at 200 hotspots per provider.

**Gates at the addendum.** `make verify` GREEN · P10 0 live (+5 density exemptions: `domains/fire`, `hms`, `wfigs`, `firms`, `platform/config` — the ratified pure-parser pattern; for ratification) · golangci-lint 0 · staticcheck 0 · `a2dh validate` 18/18.

**Round 3 red-team.** 40 findings (security 7, perf/fail-soft 11, UX 12, docs 10): 1 Critical fixed (HMS parsed per RECENT scheduler), 8 of 9 Important fixed (the ninth, the `--ascii` flag, remains the B6 item), the rest fixed or declined with rationale — table in `red-team-build.md` Round 3. Verdict unchanged: **SHIP-WITH-CONDITIONS** (Linux protocol run, M8 soak, squash-publish `main`).

**Approved 2026-08-25** ("reports approved") after HUM LEAD's fit-and-finish pass, UAT 102–115 (`0c94e1d`…`bcf4399`), each landed under the same gates.

**For HUM LEAD at UAT.** (1) The fire mark: settled at UAT 110 (`›  ▶ 3◆ 2⚠ 001.`, counted orange ◆). (2) The setup question D-28 asked ("keep or hide the FIRMS prompt") is answered by the Setup window. (3) Open from before: the dependency pin row.

## Source Documents

| Document | Location | Status |
|---|---|---|
| Objectives | `01-objectives/objectives.md` | found |
| Research AI-1…13, syntheses | `02-analysis/` | found |
| Architecture (+ §11 as-built addendum, §11.9 deviations), caching, risk register | `03-architecture-design/` | found |
| Build logs B0–B2, UAT log sessions 1–97, infra ledger | `04-development/` | found |
| Debugging ledger | `05-debugging/debugging-ledger.md` | found |
| Key learnings | `06-key_learnings/b3-ux-backwards.md` | found |
| Release checklist, Linux validation protocol | `07-readiness/` | found |
| Red-team verdict (this exit) | `08-reports/red-team-build.md` | found |

## Next Steps

1. HUM LEAD approves this report and the red-team verdict (or directs changes).
2. Decide the two open questions: the dependency pin row; the FIRMS prompt in setup.
3. Create the public repository; merge `feature/watchpost-cli` → `release/v0.9.0` → PR to `main`
   **squash-merged** (three historical commit messages name internal tools; the feature and
   release branches stay unpushed); tag `v0.9.0` on `main`; the release workflow publishes the
   binaries, `checksums.txt`, LICENSE and THIRD_PARTY_LICENSES.md.
4. VALIDATE: run `07-readiness/linux-validation-protocol.md` on the second laptop against the
   published installer; record the VALIDATE report; fix-forward as `v0.9.x`.
5. SHIP report, then REFLECT (M6 upstream candidates, M7 correction count, the UX-backwards
   method).
