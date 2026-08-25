---
title: "Red-Team Verdict — Watchpost CLI (BUILD exit, 0.9.0)"
date: 2026-08-25
scope: "project (feature/watchpost-cli, B0–B4, HEAD at exit)"
mode: multi-agent (6 lenses over 2 rounds) + single-agent synthesis
---

# Red-Team Verdict — Watchpost CLI, BUILD exit

> red-team: SHIP-WITH-CONDITIONS · multi-agent · scope:project · personas:[InfoSec (security lens), Performance (perf pass), Linux/cross-platform (portability lens)]

## Verdict: SHIP-WITH-CONDITIONS

Every Critical and Important finding from both rounds is fixed and re-verified (round 2 re-verified round 1's fixes and found the two places they were incomplete); what remains is Minor and accepted, plus two conditions the human owns: (1) the Linux end-to-end run on a second machine (`07-readiness/linux-validation-protocol.md`) is the 0.9.0 → 1.0 gate — one Linux finding (headless audio) was fixed blind and must be seen to fail correctly; (2) the 1-hour soak with radio on (M8) is not measured — 90 s windows show a creep that is not isolated.

## Method

Round 1 (fresh lenses, no shared findings): plan-vs-record gap analysis; security & standing constraints (§10); concurrency & lifecycle; fail-soft & degraded environments; Linux / cross-platform / first-run install. Round 2 (fresh lenses given round 1's findings + fix commits): verify-the-fixes & attack-the-new-code; project hygiene / docs quality / release axes. Each lens was told to refute before reporting and to cite `file:line`. Reproductions were required for concurrency claims (the orphaned-stream race reproduced 50/50 before the fix; the fuzzer produced the crasher). Every finding carries a disposition: **Fixed** (commit), **Declined** (rationale) or **Deferred** (tracked).

Fix commits: `4a1d5c5` (ship prep), `fe2c649` (round 1), `ee6b935` (Linux), `efd1879` (round 2 correctness), `c15bc4d` (UAT 98 cache), and the round-2 hygiene/scrub commit that follows `c15bc4d`. Gates after each: `make verify` GREEN, `-race` clean, P10 0 live, lint 0, `a2dh validate` 18/18, release matrix + installer test OK.

## Findings

Severity: Critical / Important / Minor (the lenses' BLOCKER/HIGH → Critical/Important, MEDIUM → Important, LOW/INFO → Minor).

### Round 1 — Plan vs record (gap analysis)

| # | Axis | Finding | Severity | Evidence | Simplify/Delete? | Action / Status |
|---|---|---|---|---|---|---|
| G-1 | Project Hygiene | `go.mod` `replace` pointed at an absolute path on one Mac; no other machine could build; the release workflow could not have run | Critical | `go.mod:47` | — | **Fixed** `fe2c649`: go-studs carried in `third_party/` via a relative `replace` (D-25) |
| G-2 | Business | No LICENSE, README, CHANGELOG for a public release | Critical | repo root | — | **Fixed** `4a1d5c5`/`fe2c649`: MIT (D-24), README, CHANGELOG 0.9.0 |
| G-3 | Docs | `docs/extending.md` documented `app/registry.go`, which never existed | Important | `docs/extending.md:1-8` | Delete the registry walkthrough step | **Fixed** `fe2c649`: rewritten against the shipped code |
| G-4 | Docs | Pin row in the architecture (bubbletea 2.0.9 …) ≠ `go.mod` (2.0.3 …); PD-4 breakpoint ruling superseded in UAT; `platform/audio` empty | Important | `architecture.md:7`, `go.mod` | Delete `platform/audio` | **Fixed** `fe2c649`: §11.9 records each deviation; `platform/audio` removed; pin decision escalated to HUM LEAD (build report) |
| G-5 | Business | B4 exit evidence named a fuzz suite; none existed (RS-16) | Important | `architecture.md:155`, `grep func Fuzz` | — | **Fixed** `fe2c649`: 6 fuzz targets — and round 2's fuzzer found N-1 |
| G-6 | Hygiene | Personal email in the NWS User-Agent; private hostnames and local paths in `06_docs` (a DISCOVER SHIP-gate item) | Important | `app/app.go:27`, AI-11, AI-6/7, project-brief | — | **Fixed** `fe2c649`: project URL as contact; hostnames/paths redacted |
| G-7 | Hygiene | A2DH PR template absent (D-7) | Minor | `.github/` | — | **Fixed**: `a2dh pr-template install` |
| G-8 | Business | R-13 safety framing incomplete (no app-level "not a substitute for official warnings") | Important | `app/credits.go` | — | **Fixed**: About lines + README |
| G-9 | Business | Plan items not built: B5 (global provider, fire, schema v1.0), B6 (playlist, national summary, `--ascii`/`--no-animation`, `report --every`, soak), WAT tone, silence detector, text ticker, `--tts-cmd`, M-V6b | Important | gap table | Candidates to delete from 1.0 scope: `--no-animation` (no animation exists to disable beyond the shimmer) | **Deferred** to the 1.0 plan (build report *Known Technical Debt*); `report --every` first — it was ratified (D-18/G-9) |
| G-10 | Hygiene | FIRMS key collected by setup and read by nothing | Minor | `app/setup.go`, `domains/fire/.gitkeep` | Hide the prompt until B5 | **Closed at B5** (`470bd08`; addendum): the key feeds `domains/fire/firms` and the Setup window stores it (UAT 99–100) |
| G-11 | Hygiene | `tea_debug.log` / `.DS_Store` "committed" | — | `git ls-files` | — | **Refuted**: untracked and ignored (`.gitignore`) |
| G-12 | Hygiene | 22 P10 exemptions accumulated through B3/B4 without a recorded ratification | Minor | `.a2dh-p10-exemptions.yml` | — | **Presented** for ratification in this report (Appendix A) |

### Round 1 — Security & standing constraints

| # | Persona | Finding | Severity | Evidence | Simplify/Delete? | Action / Status |
|---|---|---|---|---|---|---|
| S-1 | InfoSec | Overlapping `Start`/`StartSource` orphan an engine goroutine → two relay connections from one listener (ToS), double audio (converges with C-1) | Important | `engine.go:97-107` | — | **Fixed** `fe2c649` (startMu); pinned `TestConcurrentStartsNeverOrphanAStream` |
| S-2 | InfoSec | 403/404 mounts retried 3× with backoff; ToS says honour them | Important | `engine.go:165-182`, `icy.go:60` | — | **Fixed**: `ErrMountRefused`, no retries on that mount; pinned `TestRefusedMountIsNotRetried` |
| S-3 | InfoSec | Tar symlink with an absolute `Linkname` can be followed by a later entry | Minor | `install.go:230` | — | **Fixed**: absolute / `..` link targets refused |
| S-4 | InfoSec | Installer follows redirects to any scheme; no size cap | Minor | `install.go:147,165` | — | **Fixed**: https-only redirects, `LimitReader(Size+1)`, 1 MB for unsized |
| S-5 | InfoSec | `FindPiper("")` resolves `./piper/piper` relative to CWD when `$HOME` is unset → executes a planted binary | Minor | `install.go:82`, `radio.go:564` | — | **Fixed**: `FindPiper`/`EnsurePiper` refuse empty or relative dirs |
| S-6 | InfoSec | Theme JSON values spliced into SGR unvalidated; relay/provider text unsanitized reaches the plain report and OSC-8 links | Minor | `render.go:231`, `report.go:79`, `icy.go:161` | — | **Fixed**: `themeValue` regex; `render.Plain` at every outside-text boundary; `icy.plain` |
| S-7 | InfoSec | `Radio.TTSCmd` / `StreamURLOverride` parsed, never read | Minor | `config.go:40` | Delete or mark reserved | **Fixed**: marked reserved (1.0 items) |
| S-8 | InfoSec | `LD_LIBRARY_PATH` duplicated; glibc keeps the first | Minor | `voice.go:155` | — | **Fixed**: replaced, not appended |
| S-9 | InfoSec | Actions pinned by tag with `contents: write` | Minor | `release.yml` | — | **Fixed**: pinned by commit SHA |
| S-10 | InfoSec | Cache dir 0755 / files 0644 | Minor | `cache.go:66,171` | — | **Fixed**: 0700 / 0600 |
| S-✓ | InfoSec | Verified HELD: narration never in argv (§10.5, temp file 0600 / stdin); config 0600/0700 atomic; SHA-256 before extract/exec; UA on every request; redaction; no TLS overrides; directory poll ≤ 5 min; no secrets or attribution in tracked files; installer verifies before `mv` | — | see lens report | — | — |

### Round 1 — Concurrency & lifecycle

| # | Axis | Finding | Severity | Evidence | Simplify/Delete? | Action / Status |
|---|---|---|---|---|---|---|
| C-1 | Code Quality | Two overlapping starts orphan a stream (reproduced 50/50) | Important | `engine.go:97-112` | — | **Fixed** (startMu; `arm()` returns its own `done`); 60-round pin |
| C-2 | Code Quality | `cancel()` deferred after `stopAll()` → quit waits through the launch burst / retries | Important | `app/dashboard.go:61,97` | — | **Fixed**: cancel before wait |
| C-3 | Code Quality | No tune epoch: a slow Tune or late dwell timer restarts playback after Stop | Important | `radio.go:102-130` | — | **Fixed**: `gen` epoch; Stop bumps + clears mode; round 2 made it atomic (N-3) |
| C-4 | Code Quality | Render failure indistinguishable from completion → Watchlist re-tune spin (converges with F-3) | Important | `source.go:161`, `engine.go:243` | — | **Fixed**: `Source.Err`; deck maps to Failed, never advances; round 2 closed the hand-over path (N-4) |
| C-5 | Code Quality | `SetVoice` (disk + subprocess) on the update loop; `Name()` execs `defaults` under the Source lock | Important | `dashboard.go:680`, `voice.go:63` | — | **Fixed**: tea.Cmd + `voiceErrMsg`; `systemVoiceName` memoized |
| C-6 | Code Quality | `rp.scheds` ranged by the starter while a commit writes it — fatal map race | Minor (fatal) | `app/dashboard.go:638` | — | **Fixed**: `rp.mu` + `started`; round 2 closed `stop()` (N-6) |
| C-7 | Code Quality | Singleflight leader's ctx/lane governs all waiters | Minor | `httpx.go:308` | — | **Declined** for 0.9.0: self-healing via the 10 s rehydration retry; recorded as 0.9.x |
| C-8 | Code Quality | Unbounded body reads (httpx, installer) | Minor | `httpx.go:410`, `install.go:165` | — | **Fixed**: 32 MB cap; installer `Size+1` |
| C-9 | Code Quality | Concurrent `EnsurePiper` write the same `.part` | Minor | `install.go:159` | — | **Fixed**: `installMu` |
| C-10 | Code Quality | ctx-closer goroutine outlives a naturally ended broadcast until the next Halt | Minor | `source.go:134` | — | **Declined**: bounded to one, not a leak |
| C-11 | Code Quality | `next()` failure loops silently every 30 s under PLAYING | Minor | `source.go:152` | — | **Fixed**: the marquee says "Waiting for the forecast products…" |

### Round 1 — Fail-soft & degraded environments

| # | Axis | Finding | Severity | Evidence | Simplify/Delete? | Action / Status |
|---|---|---|---|---|---|---|
| F-1 | Business | A failed radio shows only `✘ NO STREAM`; the reason never renders | Important | `dashboard.go:2363` | — | **Fixed**: the reason on the station line, both player sizes; pinned |
| F-2 | Business | Voice error overwritten by "no relay carries this transmitter" | Important | `radio.go:236`, `engine.go:97` | — | **Fixed**: `Engine.Fail(reason)` |
| F-3 | Business | Broken voice reads as "broadcast complete"; Watchlist hot-loops re-tuning; setup promises a text ticker that does not exist | Important | `source.go:161`, `radio.go:599` | Delete the promise | **Fixed** (with C-4); setup message corrected; text ticker deferred (§11.9) |
| F-4 | Business | A duplicate watchlist ref publishes an EMPTY snapshot forever (probe-confirmed) | Important | `assembler.go:39,175`, `setup.go:199` | — | **Fixed**: dedupe in `NewAssembler`, the add path, setup; pinned |
| F-5 | Business | Non-US lookups load forever (404 every tick) | Important | `resolver.go:97`, `nws/provider.go:213` | — | **Fixed**: refused with the reason; round 2 widened to NWS territories on both paths (N-2) |
| F-6 | Code Quality | Warnings append forever; copied every publish | Important | `assembler.go:105-158` | — | **Fixed**: bounded to 256 |
| F-7 | Business | "Last Updated" ticks while every row is stale | Important | `dashboard.go:1932` | — | **Fixed**: newest provider fetch |
| F-8 | Code Quality | Poisoned cache body served until expiry; shared `.tmp` name | Important | `cache.go:79,170` | — | **Fixed**: forget + refetch once; `CreateTemp` |
| F-9 | Business | Corrupt config → wizard runs, then fails circularly | Important | `setup.go:284` | — | **Fixed**: load first |
| F-10 | Business | Commit save errors vanish | Important | `dashboard.go:574` | — | **Fixed**: modal reopens with the reason, naming the action (round 2 N-7) |
| F-11 | Business | 4xx mounts retried (= S-2) | Important | — | — | **Fixed** |
| F-12 | Business | Backoff cancellation hides the transport cause | Minor | `httpx.go:372` | — | **Fixed** |
| F-13 | Business | Offline rows shimmer forever | Minor | `dashboard.go:2072` | — | **Deferred** 0.9.x (the header ✘ and `[S]` say why) |
| F-14 | Business | Raw oto error; preview failures silent | Minor | `output.go:37` | — | **Fixed**: actionable message; preview reports (Linux F7) |
| F-15 | Business | `Retry-After` ignored | Minor | `httpx.go:403` | — | **Deferred** 0.9.x |
| F-16 | Business | No `time/tzdata`; hosts without zoneinfo compute UTC sun times silently | Minor | `cmd/watchpost` | — | **Fixed**: embedded |
| F-17 | Docs | Error strings without a next step; alert times in machine-local time | Minor | list in lens report | — | **Partly fixed**: alert clocks in the location's zone; the string audit is 0.9.x |
| F-18 | Business | Report prints the same warning per fetch kind | Minor | `report.go:96` | — | **Fixed**: deduped |

### Round 1 — Linux / cross-platform / first run

| # | Persona | Finding | Severity | Evidence | Simplify/Delete? | Action / Status |
|---|---|---|---|---|---|---|
| L-1 | Linux | go-studs `replace` absolute path (= G-1); `third_party` uncommitted at review time | Critical | `go.mod:47` | — | **Fixed** `fe2c649` |
| L-2 | Linux | oto reports "no PulseAudio, no ALSA" only via `Context.Err()`, never read → PLAYING over silence on a headless box | Important | `output.go:37-46` | — | **Fixed** `ee6b935`; **condition**: verify on the Linux laptop (protocol §4) |
| L-3 | Linux | `defaultVoice` off-darwin returns "Samantha"; chooser stale after a lazy Piper install | Important | `radio.go:520`, `:77` | — | **Fixed** `ee6b935` |
| L-4 | Linux | Repo not yet on GitHub / private | Important | `gh repo view` | — | **Decided** D-23: public; create the repo before tagging (ship step) |
| L-5 | Linux | Piper needs glibc ≥ 2.29 + libstdc++6; not documented | Important | `install.go:50` | — | **Fixed**: README requirements |
| L-6 | Linux | Linux binaries are dynamically linked to glibc (purego dlopen) despite CGO off | Important | `file dist/watchpost-linux-amd64` | — | **Fixed**: documented (Alpine/musl unsupported); accepted |
| L-7 | Linux | Preview with no voice is silent | Minor | `radio.go:403` | — | **Fixed** |
| L-8 | Linux | Interrupted installs leave `.part` files | Minor | `install.go:111` | — | **Fixed**: cleared at the next install |
| L-9 | Linux | GNU-only `sed \n` in the installer's token path | Minor | `install.sh:73` | — | **Fixed**: `tr` |
| L-10 | Linux | Windows config path under `~/.config` (unidiomatic) | Minor | `config.go:73` | — | **Declined** for 0.9.0 (works; Windows untested end to end — CHANGELOG) |
| L-11 | Linux | Glyph inventory; no ASCII fallback path; `SupportsUTF8` constant | Minor | `dashboard.go`, go-studs | — | **Fixed** (documented: UTF-8 locale + font required); `--ascii` is a 1.0 item (G-9) |
| L-12 | Linux | Report mode emits truecolor verbatim on 256-colour terminals | Minor | `term.go:92` | — | **Declined**: terminals approximate; noted |

### Round 2 — Verify the fixes / attack the new code

| # | Axis | Finding | Severity | Evidence | Simplify/Delete? | Action / Status |
|---|---|---|---|---|---|---|
| N-1 | Code Quality | **Crasher** (from the new fuzz suite): a block starting with a bare state code (`"CA "`) slices `runes[-1:-1]` — untrusted NWS text could crash the render goroutine | Important | `normalize.go:115` | — | **Fixed** `efd1879`: guard; the fuzz corpus entry is the regression; pinned |
| N-2 | Business | F-5's filter refused NWS territories (geodata files Puerto Rico as `PR`); the online fallback applied no filter | Important | `resolver.go:99` | — | **Fixed**: `domains/locations/coverage` shared by both paths; pinned |
| N-3 | Code Quality | C-3's epoch check not atomic with the engine start; a stale fallback relabelled mode after Stop | Important | `radio.go:130,248` | — | **Fixed**: `tuneMu` around check→start and around Stop's bump→halt; epoch checked before any relabel |
| N-4 | Code Quality | C-4 incomplete: re-voice and hand-over render failures ended the stream without `Err` | Important | `source.go:205,220` | — | **Fixed**: `fail()` on every render path; pinned (broken voice mid-broadcast) |
| N-5 | Code Quality | `EnsurePiper` lacked `FindPiper`'s IsAbs guard → 90 MB download every tune-in | Minor | `install.go:104` | — | **Fixed** |
| N-6 | Code Quality | `recentPipeline.stop` ranged the map without `rp.mu` | Minor | `app/dashboard.go:588` | — | **Fixed** |
| N-7 | Business | Error modals reopen on top of others; a failed remove reopened an "Add" modal unnamed | Minor | `dashboard.go:2648` | — | **Fixed**: alone, naming the action |
| N-8 | Business | A truncated download reads as "checksum mismatch" | Minor | `install.go:193` | — | **Fixed**: "download incomplete (got X of Y)" |
| N-✓ | — | Re-verified HOLD: C-1, C-2, C-5, C-6, C-8, C-9, F-4, F-6–F-10, F-17, S-3–S-10 (every built-in theme value matches the validator), third_party copy, `-race -count=3` clean | — | lens report | — | — |

### Round 2 — Project hygiene / docs / release axes

| # | Axis | Finding | Severity | Evidence | Simplify/Delete? | Action / Status |
|---|---|---|---|---|---|---|
| R2-01 | Hygiene / Business | The public tree carried private-employer material: an employer-IP analysis of go-studs, an employer email, internal remotes and module namespaces, 68 lines across 11 docs — and the development harness itself | Critical | AI-11, tier2-synthesis, project-brief, `_a2dh/`, `AGENTS.md`, `CLAUDE.md` | Delete from the public tree | **Fixed** (HUM LEAD: "remove any/all references … no addresses or machinery"): harness untracked + ignored; AI-11 replaced by a provenance note; every doc, code comment and the copied kit scrubbed to generic wording; tracked-file scan clean. **Condition**: three historical commit messages name internal tools — publish `main` by squash and keep the feature branch unpushed (ship step) |
| R2-02 | Business | Apache-2.0 / BSD-3 dependencies (oto, purego, go-mp3, cobra, pflag, x/*) require licence text with a binary; none shipped | Important | release assets | — | **Fixed**: `scripts/third-party-licenses.sh` → `THIRD_PARTY_LICENSES.md` (44 modules, 0 without a file), attached to releases with LICENSE |
| R2-03 | Business | `go install …@v0.9.0` and `go get` fail: the sub-module + `replace` | Important | `go.mod:47` | Simplify | **Fixed**: go-studs folded into the main module (imports rewritten, no `replace`); the sync script derives the upstream path from its `go.mod` |
| R2-04 | Release | No CI on push/PR — a tag would be the first gate run | Important | `.github/workflows/` | — | **Fixed**: `ci.yml` (verify + matrix + installer test, ubuntu/macos, SHA-pinned) |
| R2-05 | Release | "pre-release" changelog vs `releases/latest` in the installer | Important | `CHANGELOG.md:6`, `install.sh:85` | — | **Fixed**: 0.9.0 ships as a normal release ("first public release, pre-1.0") |
| R2-06 | Hygiene | Every invocation wrote a 0-byte `tea_debug.log` into the cwd (a dependency's `init()`) | Important | ultraviolet `terminal_renderer.go:1306` | — | **Fixed**: ultraviolet bumped to a build without it; verified in a fresh cwd |
| R2-07 | Hygiene | `.gitignore` incomplete (`.claude/`, `.a2dh/`, coverage, profiles) | Important | `.gitignore` | — | **Fixed** |
| R2-08 | Docs | `toolchain go1.27.0` forced a toolchain download for Go 1.25 users | Important | `go.mod:5` | Delete | **Fixed**: removed; `go mod tidy` |
| R2-09 | Hygiene | `go.sum` not tidy | Minor | — | — | **Fixed** |
| R2-10 | Docs | Help text leaked internal IDs ("M-V6", "v1.0-rc", "T-C explicitness") | Minor | `root.go:31,99,100` | — | **Fixed** |
| R2-11 | Docs | Help overlay said "Repeat On/Off" (tri-state since UAT 93) | Minor | `dashboard.go:170` | — | **Fixed** |
| R2-12 | Docs | `º` (ordinal) vs `°` (degree) in help/units | Minor | `dashboard.go:177` | — | **Declined**: the mocks use `º` throughout (mock fidelity rule); consistent everywhere |
| R2-13 | Docs | Public docs referenced private tooling and workflow | Minor | `docs/extending.md`, PR template | Delete/trim | **Fixed**: `extending.md` rules rewritten; generic PR template; harness untracked |
| R2-14 | Docs | Schema `$id` URL 404s; `1.0.0-rc` label on a 0.9.0 binary | Minor | `schema.json:2` | — | **Fixed** (`raw.githubusercontent` URL); the `-rc` contract label stays until B5 ratifies (§10.3) — **Declined** to relabel |
| R2-15 | Docs | CHANGELOG counts verified; date must match the tag day | Minor | — | — | **Noted** |
| R2-16 | Tests | Stale test asserted "bare invocation must error until B3" | Minor | `main_test.go:13` | Rewrite | **Fixed**: bare invocation reaches the dashboard and fails actionably without a config |
| R2-17 | Tests | 7 s concurrency pin; wall-clock httpx assertion | Minor | `redteam_test.go`, `httpx_test.go:189` | — | **Fixed** (20 rounds); the httpx timing window is **Deferred** (uses a real clock; flake watch in CI) |
| R2-18 | Simplify | Installer's private-repo token branch dead once public | Minor | `install.sh:60-77` | Delete | **Fixed**: removed |
| R2-19 | Simplify | 15 `.gitkeep` placeholders; empty `domains/fire/` | Minor | tree | Delete | **Fixed** |
| R2-20 | Simplify | Duplicate geodata blobs (1.3 MB) under docs | Minor | `04-development/spikes/` | Delete | **Fixed** |
| R2-21 | Simplify | Unread config surface (`Playlist`, `TTSCmd`, `StreamURLOverride`, `Provider.Enabled`) | Minor | `config.go` | Delete | **Fixed**: removed (unknown keys still load) |
| R2-22 | Simplify | Four exported functions with zero callers; test-only exports | Minor | `render.go:645`, `term.go:92`, `install.go:364`, `invariant.go:32` | Delete | **Fixed** (the four deleted); test-only exports **Declined** (platform API used by tests) |
| R2-23 | Simplify | `cacheDir`/`voiceDir` duplicate bodies | Minor | `app/` | Simplify | **Fixed**: `userCacheSubdir` |
| R2-24 | Simplify | `domains/alerts` is a test-only package | Minor | tree | Move | **Declined** for 0.9.0: the replay harness is documented where it is |
| R2-25 | Docs | `fire`/`radio` JSON blocks always empty | Minor | schema | — | **Closed at B5** for `fire` (`470bd08`; parity fixtures carry it); `radio` stays a tuner-availability block by design (PD note) |
| R2-26 | Structure | `modes/tty/dashboard.go` 2.7k lines | Minor | — | Split | **Deferred** 0.9.x (mechanical split, not before a tag) |
| R2-27 | Release | `shasum` only; fixed sleep in the installer test | Minor | `Makefile:54`, `install-test.sh:13` | — | **Fixed**: `sha256sum` fallback; port polling |
| R2-28 | Hygiene | Personal e-mail / local paths in two docs | Minor | project-brief, G5 spike | Delete | **Fixed** with R2-01 |
| R2-29 | Docs | `completion` undocumented | Minor | README | — | **Fixed** |
| R2-✓ | — | Verified clean: no secrets/tokens/`.env`/coverage/editor files tracked; executable bits; `go mod verify`; README key table = `defaultKeyMap()`; install URL/paths/env vars match code; `extending.md` symbols exist; installer `sh -n`/`dash -n`; actions pinned; suite network-free | — | lens report | — | — |

### UAT 98 (during the exit) — wide-terminal synth report

| # | Axis | Finding | Severity | Evidence | Simplify/Delete? | Action / Status |
|---|---|---|---|---|---|---|
| U-1 | Performance | "Very wide terminals: long narrator pauses, CPU ramps" | Important (reported) | measured: 133 vs 400 cols, PLAYING throughout, +3–5 % CPU at 400 | — | **Fixed-by-others** (F-3 hot loop, C-1 orphans); **condition**: re-test on the exit build; pprof switch documented |
| U-2 | Performance | RSS +35 MB in 90 s of synth; 64 cached stereo segments | Important | infra ledger | — | **Fixed**: mono cache, cap 24; residual creep → 1-hour soak (M8) at VALIDATE |
| U-3 | Docs | The pty harness must drain (expect `sleep` blocks the app on terminal write) | Minor | debugging ledger D9 | — | **Fixed** (harness), recorded |

### Round 3 — B5 fire + Setup window (addendum, 2026-08-25; HUM LEAD held BUILD exit for fire)

Four fresh lenses over `470bd08` (fire data layer + UI), refute-before-report, `file:line` cited, reproductions where claimed (parse cost measured on the live 1.46 MB archive; the 50-scheduler launch shape benchmarked; FIRMS 400 probed live). Fix commit: `eaf7ade` (UAT 100–101). HUM LEAD's fit-and-finish UAT that followed the presentation (102–115: masthead, empty states, Synthwave '84, ◆ marks block, Setup form, radio lead / fire report / pauses / pronunciations) landed as `0c94e1d`…`bcf4399`, each under the same gates; approved with this addendum 2026-08-25. Gates after: `make verify` GREEN, P10 0 live, golangci-lint 0, staticcheck 0, `a2dh validate` 18/18.

| ID | Lens | Finding | Severity | Where | Disposition |
|---|---|---|---|---|---|
| S1 | Security | FIRMS key redaction shape-dependent; a mis-pasted key leaks through `frag.Err` → warnings → report/JSON/[S] | Important | `httpx.go:238`, `firms.go:46` | **Fixed**: `firms.CheckKey` (32 hex) refuses before store or use, reason never echoes the value; `redactKey` scrubs transport errors; setup and `SetKey` gate on it. Pinned. |
| S2 | Security | `maxPlacemarks` counted XML tokens → silent truncation at ~100k | Important | `hms.go:159` | **Fixed**: counts placemarks; `ErrTruncated` published as the fragment warning with what was read. Pinned (200,005 → 200,000 + error). |
| S3 | Security | `[fire] min_confidence` validated late, case-sensitive, kills HMS/WFIGS too | Minor | `config.go:63`, `fire.go:35` | **Fixed**: `Fire.Validate` at `config.Load` (closed set, case-folded, names the key). Pinned. |
| S4 | Security | No ceiling on the rings | Minor | `config.go:63` | **Fixed**: 0–500 km at load. Pinned. |
| S5 | Security | KMZ inflate budget per entry, no file cap | Minor | `hms.go:117-135` | **Fixed**: 16 files, one 96 MB budget. |
| S6 | Security | Any `<`-led body parses as "zero fires" | Minor | `hms.go:109` | **Fixed**: root must be `<kml>`; empty body is an error. Pinned. |
| S7 | Security | One bad placemark aborted the continent | Minor | `hms.go:172` | **Fixed**: skipped. Pinned. |
| P1 | Perf | HMS parsed once per RECENT scheduler: 50 parses / 15 min, 4.5 GB allocated, 616 MB heap peak | **Critical** | `hms.go:70`, `dashboard.go:641` | **Fixed**: parse memoized by SHA-256 of the body — one parse (~120 ms, ~90 MB) per archive change whoever asks. Pinned (same bytes → same slice). |
| P2 | Fail-soft | A failed FIRMS source published an empty state (prior hotspots wiped, never retried) | Important | `firms.go:76-96` | **Fixed**: the location stays unserved (retry + prior data kept). Pinned. |
| P3 | Fail-soft | "no hotspots nearby" asserted before any fire feed answered | Important | `report.go:116`, `dashboard.go:1171` | **Fixed**: `FireState.as_of` (max over feeds); "fire feed not yet available" / "fire: feed unavailable" until then. Pinned both surfaces. |
| P4 | Fail-soft | Bad key: HTTP 400 unactionable, 120 failing requests per cycle | Important | `firms.go:78` | **Fixed**: 400/401/403 → "FIRMS rejected the MAP_KEY — open Setup ([s])…", cycle stops at the first. Pinned. |
| P5 | Perf | = S2 | Minor | — | Fixed with S2. |
| P6 | Fail-soft | A corrupt cached archive served for its TTL | Minor | `httpx.go:294` | **Fixed**: `Client.Forget`; HMS/FIRMS forget a body their parser rejects. |
| P7 | Lifecycle | Quit waited behind 50 parses | Minor | `sched.go:302` | Falls out of P1. |
| P8–P11 | Perf | mergeFire cost, cache tiering, retry semantics, fire-map lifetime | Info | — | Refuted / confirmed sound; numbers in the lens report (mergeFire 60 loc × 3 × 200 = 3.7 ms). |
| U1 | UX | `strength n/a · GOES-WEST` glued to the age column (the common HMS case) | Important | `dashboard.go:1161` | **Fixed**: `n/a MW`; satellite omitted when empty. |
| U2 | UX | Zero `DetectedAt`/`Discovered` rendered `2562047h` / `Jan 1` | Important | `dashboard.go:1168`, `report.go:120` | **Fixed**: `age n/a` / `n/a`. |
| U3 | UX | ▲ took the last gutter cell: `⚠▲001.` | Important | `render.go:473` | **Fixed, then redesigned by HUM LEAD at UAT 110**: the marks block is now 11 cells `›  ▶ 3◆ 2⚠ 001.` — a counted orange ◆ for fire, two cells after the pointer, one between the rest. |
| U4 | UX | `--ascii` never reaches the dashboard; `…`/`·` not swapped | Important | `dashboard.go:2048`, `root.go` | **Partly fixed** (separators swap); the flag itself is the B6 1.0 item (pre-existing). |
| U5 | UX | Silent incident-name truncation | Minor | `dashboard.go:1182` | **Fixed**: ellipsis. |
| U6 | UX | 3-letter bearing at ≥ 100 km fills the column | Minor | `dashboard.go:1160` | **Fixed**: guaranteed gap. |
| U7 | UX | FIRMS satellites as raw codes (`N20`) | Minor | `firms.go:149` | **Fixed**: `NOAA-20` / `NOAA-21` / `Suomi NPP`. Pinned. |
| U8 | UX | No legend for ▲ (or any mark) | Minor | `dashboard.go:2695` | **Fixed**: row-marks legend in `?`. |
| U9 | UX | "hot" is bold-only in the row | Minor | `render.go:470` | **Declined**: the fact is the glyph; the MW figure is in the modal (R-12a met). Count-in-row is a HUM LEAD design call. |
| U10–U12 | UX | Parity holds; themes; refuted candidates (spacer, bearing, cap, exit 2) | Info | — | High Contrast `FireMark` set to 214. |
| D1 | Docs | Plan of record contradicted itself (§11.9 "not built", tier diagram, §11.10 before §11.9, G-10/R2-25 open, build log promising an addendum) | Important | `architecture.md`, reports | **Fixed**: all reconciled; this addendum is the promised one. |
| D2 | Docs | UAT 99.6 said `tab` reveals the key; code is `ctrl+r` | Minor | UAT log | **Fixed**. |
| D3 | Docs | `min_confidence = "high"` silently dropped every HMS point | Minor | `fire.go:44` | **Fixed**: analyst-curated outranks high. Pinned. |
| D4 | Docs | `docs/extending.md` walkthrough 2 described an unshipped key pattern | Minor | `extending.md:42` | **Fixed**: rewritten to the FIRMS pattern (constructor + `SetKey`/`Enabled`, `SetInactive`, per-kind tiers). |
| D5–D7, D9 | Docs | CHANGELOG intro; setup help string; orphaned `finalize` godoc; credits ambiguous when unkeyed | Minor | — | **Fixed** ("through B5 … 100 sessions"; "needs key" credit; godoc moved; setup Short rewritten with the window). |
| D8 | Docs | `report` now fails on a corrupt config | Minor | `app.go:52` | **Declined**: the error names the file and the fault; hiding a broken `[fire]` table or key from the report would be worse. |
| D10 | Docs | `?` did not explain ▲ | Info | — | Covered by U8; `enter` relabelled "Location Details (forecast, marine, fire)". |

Also from HUM LEAD during the pass (UAT 100–101): Setup rebuilt as a dashboard window (`[s]`; first run; `watchpost setup`), FIRMS keys live without relaunch, unkeyed FIRMS reads `off`; Location Details chip row consolidated. Per-location hotspots capped at 300 nearest (a mega-fire clustered into 941).

**Round 3 verdict:** 40 findings — 1 Critical (fixed), 9 Important (8 fixed, 1 partly — the `--ascii` flag stays the B6 item), 30 Minor/Info (fixed or declined with rationale). No open blocker. The round-1/2 conditions stand unchanged (Linux protocol run, M8 soak, squash-publish `main`).

## Axis Coverage

- **Code Quality** — concurrency, lifecycle, fail-soft, fuzz: 2 rounds; 24 findings, 22 fixed, 2 declined with rationale.
- **Project Hygiene** — public-tree scrub (no private-employer addresses or machinery), licences shipped, dependency carriage in-module, CI, ignore rules, placeholders: 2 rounds; all Critical/Important fixed.
- **Docs Quality** — README/CHANGELOG/extending/architecture §11.9 written or rewritten against the code; help strings and the key map re-verified against the code; two declines recorded (`º` per the mocks; `-rc` per §10.3).
- **Business Quality** — safety framing (R-13), US-only honesty, failure reasons visible, ToS honoured, third-party licences; 1.0 scope deferred explicitly.
- **BUILD phase lens** (Distinguished Engineer) — exit evidence: tests/race/fuzz/gates green; perf measured; deviations recorded; debt named with triggers.

## Summary

Rounds 1–2: Critical 4 (all fixed) · Important 41 (39 fixed, 2 escalated to HUM LEAD: pin row, FIRMS prompt — the second answered by B5) · Minor 43 (32 fixed, 11 declined/deferred with rationale) — 88 findings, 75 fixed. Round 3 (B5 addendum): 40 findings, 1 Critical fixed, 8 of 9 Important fixed, the rest fixed or declined with rationale — 128 findings over three rounds. **BUILD can exit and 0.9.0 can ship with conditions**: publish `main` by squash (history names internal tools in three commit messages); the Linux protocol run and the M8 soak at VALIDATE; the two HUM LEAD questions answered.

## Appendix A — P10 exemptions added since the last ratification (for HUM LEAD)

B3/B4 (22): `platform/tz` (memo + density), `platform/geo`, `domains/radio/player` (density; `Close`, `Read`, `NewPlayer`, `Read`/`Samples` name-graph), `domains/radio/synth` (`Rate`, density), `domains/radio/stream` (density), `modes/tty` (density), `domains/marine/coops`, `tools/nwrtable`, `domains/locations/geodata`, `platform/render/themes.go` (theme registry), … (full list: `.a2dh-p10-exemptions.yml`, entries dated B3/B4).
Exit (9 + the copy): `domains/radio/spectrum` (pure DSP), `domains/locations/coverage` (pure lookup), `synth/voice.go` (System-Voice memo), `synth/install.go` (`installMu`), `app/radio.go` (`Stop`/`stopDwell`/`SetVoice` name-graph), and `third_party/go-studs/*` — once the kit was folded into the main module every P10 rule scanned it (57 findings, all in upstream code), exempted per file and symbol under one reason: upstream-governed, never edited here. The exemptions file itself is now local-only (development harness, untracked).
