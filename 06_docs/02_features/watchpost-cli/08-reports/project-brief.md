# Project Brief — Watchpost CLI

| Field | Value |
|---|---|
| Report | project-brief v1.4.0 |
| Phase | pre-DISCOVER (collect-brief handoff) |
| Date | 2026-08-23 |
| Author of record | Branden Thompson (HUM LEAD) |
| Branch | `feature/watchpost-cli` |
| Status | APPROVED — HUM LEAD, 2026-08-23 |

---

## Summary & Intent

Watchpost is a live, terminal-native weather station. Invoked as `watchpost`, it gives a person a dense, continuously-updating picture of current conditions, forecasts, and — critically — active weather and fire alerts across every location they care about, from a national summary down to a single zip code, with optional NOAA Weather Radio playback. Behind a deliberately simple, keyboard-driven interface sits a robust data layer: multi-provider API ingestion, normalization, harmonization, correlation, caching, and audio streaming.

**Why now / why it matters.** People who live in severe-weather and wildfire regions depend on timely alerts, but the information is scattered across apps, websites, and radio. For people who already live in a terminal, there is no first-class, glanceable, always-on surface for this. Watchpost closes that gap — and doubles as the proving ground for go-studs charting and animated components that will be upstreamed.

**Who benefits.** Terminal-centric individuals (developers, operators, hobbyists, weather enthusiasts) monitoring one or many locations; people in high-risk regions who need alerts surfaced the moment they are issued; automated/agent workflows that need machine-readable weather output.

**If we do nothing.** Alerts continue to be seen late or missed; multi-location awareness stays a manual, multi-tab chore; go-studs never gains the charting/animation components this project would force into existence.

## Locked Problem Statement

> **"Terminal-centric people who monitor weather across one or more locations — especially those in severe-weather or wildfire regions — have no single glanceable surface for current conditions, forecasts, and live alerts, and so they juggle multiple apps, sites, and radio, seeing alerts late or not at all."**

| # | Criterion | Score | Evidence |
|---|---|---|---|
| 1 | Bad Outcome | ✓ | "no single glanceable surface… seeing alerts late or not at all" |
| 2 | Affected Humans | ✓ | Terminal-centric people; those in severe-weather/wildfire regions |
| 3 | Tech Agnostic | ✓ | No API, language, library, or product named ("terminal" describes the humans' habitat, not a solution) |
| 4 | Non-prescriptive | ✓ | Does not say CLI, TUI, Go, or any provider; many solutions could address it |
| 5 | Verifiable | ✓ | Observe a user assembling a multi-location picture today; count surfaces used and time to alert awareness |

**Score: 5/5 — LOCKED.** Ratified by HUM LEAD 2026-08-23 (D-8).

**Refinement trace.** Raw input was solution-first ("I want to build a CLI to get live weather information…"). One refinement pass stripped the solution (CLI, Go, go-studs, charm), named the affected humans, and expressed the bad outcome (late/missed alerts, fragmented awareness).

## Metrics of Success

Each metric was checked against an anti-solution ("could we hit this number without solving the problem?") and refined where it could be gamed.

| # | Name | Symbol | Type | Definition (direction) | Measured in |
|---|---|---|---|---|---|
| M1 | Time to Situational Awareness | TTSA | Primary | Seconds from `watchpost` launch to a fully-populated multi-location view (all configured locations showing current conditions + active alerts). **Lower is better.** Target ≤ 3s warm cache, ≤ 8s cold, 10 locations. | Built-in timing instrumentation (`--debug-timing`), Go benchmarks |
| M2 | Alert Surfacing Latency | ASL | Primary | Seconds between an alert's official issuance timestamp (e.g. NWS `sent`) and its first on-screen render. **Lower is better.** Target ≤ 60s while running. Anti-solution guard: measured against issuance time, not poll time, so "poll faster and show stale data" cannot game it. | Alert ingestion log with issuance vs render timestamps |
| M3 | Alert Coverage | AC | Primary | % of alerts issued for a configured location (per provider's own feed) that Watchpost rendered. **Higher is better.** Target 100% for NWS. Guards M2 — low latency on a subset is not success. | Replay harness comparing provider feed to rendered-alert log |
| M4 | Multi-Location Scale | MLS | Secondary | Max configured locations with TTSA still ≤ target and M8/M9 still within budget. **Higher is better.** Target ≥ 25; stretch: national summary. | Go benchmarks + `pprof` in VALIDATE |
| M8 | Memory Footprint | MEM | Primary | Resident set size (RSS) of the running TTY process after 10 minutes steady state. **Lower is better.** Targets: ≤ 40 MB at 10 locations (no radio); ≤ 80 MB at 25 locations with radio playing. Heap must be flat over 1 hour (no growth > 5%) — anti-solution guard against "low at launch, leaks forever". | `runtime.MemStats` sampled by `--debug-stats`; `pprof` heap profiles; 1-hour soak test in VALIDATE |
| M9 | CPU Budget | CPU | Primary | Steady-state CPU as % of one core, sampled over 60s windows. **Lower is better.** Targets: ≤ 1% idle/no-animation view; ≤ 5% with animated components at default tick; ≤ 10% with radio decoding + animation. Renders must be event/tick-driven, never busy-looping — anti-solution guard: CPU at 0 locations must be ≈ 0. | `--debug-stats` footer; `pprof` CPU profiles; benchmark of render loop per tick |
| M5 | Machine-Mode Fidelity | MMF | Secondary | `--json` / `--report-only` output passes a published schema and contains the same data the TTY view renders (no TTY-only facts). **100% parity.** | Schema tests + golden fixture parity tests |
| M6 | Component Upstream Yield | CUY | Tertiary | *Superseded by G-4a (D-12):* components accepted into whichever shared component home D-9 resolves to. **Higher is better.** Measurable post-SHIP only (red-team B-3). | Target home's changelog / merged PRs |
| M7 | Correction Count | CC | Maintenance | Number of HUM LEAD corrections per phase (framework quality-floor metric). **Lower is better.** | REFLECT reports |

## Requirements

Verbatim from HUM LEAD intake, numbered for traceability. Sharpening notes (SH-n) follow.

**Functional (R-n)**

| ID | Requirement |
|---|---|
| R-1 | Users must be able to invoke watchpost in the terminal via the `watchpost` command. |
| R-2 | Users must be able to provide city names and/or zip codes to get real-time weather data for that location. |
| R-3 | Users must be able to quickly navigate the app through specific keybindings, with `?` reserved for a help pane. |
| R-4 | Users must be able to set up their watchpost via `watchpost setup` — initial locations/regions, plus opt-in additional API support using their own keys for non-free APIs. |
| R-5 | When available, users must be able to "tune in" to a weather radio channel. |
| R-6 | CLI must provide a streamlined, smaller interface for the radio service (CLIAmp-like, tuned to weather radio). HUM LEAD will provide a mock at that feature. |
| R-7 | Users must be able to "dive in" from a broad list of locations to a specific location for detailed "right now" data plus any forecast/look-ahead data the APIs support. |
| R-8 | Users must be able to see weather alerts/statements as they appear, in real time. Highest importance for severe-weather regions. |
| R-9 | Users should be able to get FIRE alerts / evacuation notifications (e.g. California). May require additional API access. |
| R-10 *(derived)* | Users must be able to run a cycling "Local on the 8s"-style playlist of views (local forecast, national forecast, travel/airport forecast, …). *Derived from Summary/Intent — listed so it is tracked, not lost.* |
| R-11 *(derived)* | Users must be able to view many locations at once, up to a national summary. *Derived from Summary/Intent.* |

**Technical (T-n)**

| ID | Requirement |
|---|---|
| T-A | Must be written in Go. |
| T-B | Must support streaming/animated/interactive TTY views. |
| T-C | Must have a non-interactive stdout mode, opt-in via flag (`--report-only`, `--json`), usable by humans and agents/automation. |
| T-D | Must use go-studs, bubbletea, runewidth, etc. — the stack `the reference CLI` uses. |
| T-E | Must pull from available weather APIs: NWS required; WeatherAPI, Open-Meteo, WeatherStack, etc. to be evaluated in PLAN. |
| T-F | In multi-API scenarios, must be able to harmonize, present side-by-side, and/or call out differences between providers. |
| T-G | Must let end users supply their own API keys for non-free APIs (CLIAmp/Spotify pattern). |
| T-H | Must ingest streaming audio (e.g. NOAA radio) for playback. |
| T-I | Must ingest fire-watch data via APIs (NASA FIRMS, OpenWeather FWI, Genasys Protect, etc.). |
| T-J | Publishes via the `branden-thompson` GitHub account (PERSONAL_PROJECTS git config — verified: commits author under the personal address). |
| T-K | Architecture must be modular enough to add views as the project grows. |
| T-M *(added at PLAN exit, D-17)* | Humans must be able to install watchpost via a one-line curl command (`curl -fsSL …/install.sh \| sh`), CLIAmp-style: OS/arch detection, prebuilt static binaries, checksum verification, clear PATH guidance. |
| T-L | All experiences reachable via `watchpost`; sub-commands are primarily for single-output/agent/automation use (`watchpost report 92057 --json`). |

**Sharpening notes**

- **SH-1 (R-8 / M2-M3):** "Real time" is made testable by M2 (≤ 60s from issuance) and M3 (100% NWS coverage). HUM LEAD to confirm the 60s target.
- **SH-2 (R-5, R-9, T-H, T-I):** These depend on third-party availability (NOAA radio streams are not an official NOAA service; fire/evacuation feeds vary by vendor). Treated as *must-when-available* — PLAN must define graceful degradation when a feed is absent.
- **SH-3 (T-D):** `the reference CLI` pins bubbletea v0.24.2 / lipgloss v0.8.0 / bubbles v0.16.1 and does **not** import go-studs as a module. go-studs is on go 1.24.4. Version alignment is a PLAN decision (OQ-3).
- **SH-4 (T-C vs T-B):** Two rendering targets (live TTY, static stdout) from one data model is an architectural seam — PLAN must ensure views are pure functions of normalized state so M5 parity is achievable.
- **SH-5 (R-2):** "City names" are ambiguous (Springfield ×30). Geocoding disambiguation UX is an open question (OQ-5).

## Technical Constraints

| ID | Constraint |
|---|---|
| C-1 | Go; bubbletea/lipgloss/bubbles/runewidth/cobra stack as in `the reference CLI`; go-studs component library. |
| C-2 | Must function with zero API keys (NWS + Open-Meteo are keyless) — paid providers are strictly additive. |
| C-3 | Never store credentials in the repo; user keys live in user config with OS-appropriate permissions (secure-by-default skill applies at BUILD). |
| C-4 | Binaries build to `./dist/`; SemVer tagged releases; TDD for all Go code (FULL TDD). |
| C-5 | Rate limits and terms of service of every provider (NWS requires a User-Agent with contact info; free tiers have quotas) must be respected by design (cache + backoff). |
| C-6 | Audio playback must not require a compiled C toolchain by default if avoidable (cross-platform install friction) — to be validated in DISCOVER (OQ-7). |

## Other Considerations

- **Prior art / form-factor reference:** CLIAmp (cliamp.stream) — compact "now playing" form factor, bring-your-own-keys model. The Weather Channel "Local on the 8s" — cycling, TV-style presentation.
- **Design system:** go-studs (`<local checkout>`) currently has: spinner, animated spinner, table, data-table rows, progress, status, declarative status, badges, separators, header/footer, message block, syntax highlighting, taxonomy, theme/tokens. **No charting or directional/animated-gauge components exist** — windspeed/direction arrow, precipitation bars, temperature sparklines are net-new and upstream candidates (M6).
- **Reference implementation:** `the reference CLI` (`<local checkout>`) — layout `cmd/`, `internal/{tui,config,setup,firstrun,plugins,…}`, `pkg/`, `plugins/`; carries a A2DH full install and a Makefile/verify surface worth mirroring.
- **Stakeholders:** Branden Thompson — HUM LEAD, sole approver, will supply UI mocks (main views, radio mini-view). Future: go-studs maintainers (upstream PRs); end users of a public `branden-thompson` repo.
- **Architecture principle (HUM LEAD, verbatim):** *"a simpler, intuitive interface disguised a very robust application."* Dense data, simple surface.
- **Ops / git protocol (HUM LEAD):** `main` is never pushed to directly — every merge to `main` goes through a PR whose body uses the A2DH PR template (`05_templates/pr-template.md`, provisioned via `a2dh pr-template` as `.github/PULL_REQUEST_TEMPLATE.md`). Feature branches off `main` (or `release/vX.Y.Z`). Template headings must be reproduced explicitly when the PR body is supplied programmatically (tooling bypasses auto-fill).
- **Timeline:** none stated. Phased delivery expected (data layer → core TTY → alerts → multi-provider → radio → fire → playlist).

## Discovery Handoff Package

### Areas to Investigate

| ID | Area | Concrete artifact / question |
|---|---|---|
| AI-1 | NWS API surface | `api.weather.gov` endpoints: `/points`, `/gridpoints`, `/alerts/active`, `/stations/…/observations`; rate limits; required User-Agent; alert polling vs any push option; coverage limits (US only). |
| AI-2 | Secondary provider matrix | Open-Meteo (keyless, global), WeatherAPI, WeatherStack, OpenWeather — free-tier quotas, ToS on caching/redistribution, fields offered, geocoding support. Output: a comparison table feeding the PLAN provider decision. |
| AI-3 | Fire / evacuation data | NASA FIRMS (key, hotspot lat/long), OpenWeather Fire Weather Index, Genasys Protect / Watch Duty / CAL FIRE / IPAWS-via-NWS — which expose programmatic, licensed-for-use feeds; which are scrape-only (reject). |
| AI-4 | NOAA Weather Radio streams | Stream directories (e.g. community-run NWR relays, Broadcastify), codec (MP3/AAC), legality/ToS, per-transmitter lookup by location, reliability. |
| AI-5 | Audio playback in Go | Pure-Go decoders/players (e.g. `hajimehoshi/oto` + `go-mp3`, `gopxl/beep`, `ebitengine/oto`) vs shelling to `ffplay`/`mpv`; CGO requirements per OS; latency; volume control. |
| AI-6 | go-studs capability gap | Read `components/*.go`, `rendering/`, `theme/`, `tokens/` to establish the component contract (View-model shape, golden-fixture test pattern) so new chart/wind components conform and are upstreamable. |
| AI-7 | the reference CLI TUI architecture | `internal/tui`, `internal/config`, `internal/setup`, `internal/plugins` — what patterns to reuse (model composition, keymap/help, first-run, plugin seams) vs. avoid. |
| AI-8 | Geocoding | City/zip → lat/long without a paid key: NWS has none; Open-Meteo geocoding, Nominatim ToS, US Census geocoder, offline zip DB. Disambiguation UX. |
| AI-9 | Terminal capability matrix | Animation frame budgets, Unicode/braille for charts, 256 vs truecolor, iTerm2/Ghostty/tmux quirks, `runewidth` east-asian handling — what the charts can assume. |
| AI-10 | Machine-mode schema | Prior art for stable JSON weather schemas (e.g. NWS/CAP, Open-Meteo shape) to inform a versioned `--json` contract (M5). |
| AI-11 | go-studs access model | Module path a private module path in a private org — can a public personal project depend on it? Fork/mirror/vendor/replace options and IP implications. |

### Stakeholders to Consider

| Who | Role | Mode |
|---|---|---|
| Branden Thompson | HUM LEAD; product owner; designer (mocks); approver of every gate | HUMAN LEAD (SEV-0) |
| go-studs maintainers (Branden, a private org) | Upstream recipients of new components; contract owners | Consulted at PLAN; informed at SHIP |
| End users (public repo) | Terminal-centric individuals; severe-weather/wildfire-region residents | Represented via personas in red-team |
| Automation/agent consumers | Consumers of `--json`/`--report-only` | Represented by M5 + schema tests |
| Data providers (NWS, Open-Meteo, NASA, vendors) | ToS / rate-limit constraints | Constraint source; AI-1..AI-4 |

### Risk Signals

| ID | Risk | Severity | Why it is a risk |
|---|---|---|---|
| RS-1 | go-studs hosted on a private org; public personal project dependency | **High** | Access (private/SSO?), IP ownership, and licensing could block publishing or force a fork. Must be resolved in DISCOVER, not discovered at SHIP. *Final disposition (D-9, tier2 §E): Medium-deferred — pivot-ready architecture + repo private (OQ-17) + SHIP gate "go-studs distribution decided".* |
| RS-2 | Scope breadth (11 functional + 12 technical requirements, 4 distinct data domains + audio) | High | Classic "everything v1" risk. Mitigation: PLAN must produce a phased roadmap with an MVP gate (NWS + single/multi-location + alerts + json). |
| RS-3 | Third-party feed availability (radio, fire/evac) | High | NOAA radio streams are unofficial; evacuation feeds are vendor-specific and may be unlicensed for redistribution. "Must when available" needs explicit degradation design. |
| RS-4 | Audio playback portability (CGO, OS audio backends) | Medium | Could make `go install` fail or require brew deps; undermines R-1 simplicity. |
| RS-5 | Provider rate limits / ToS on caching | Medium | Multi-location + cycling views multiply calls; naive polling gets keys banned. Caching/backoff are architectural, not afterthoughts. |
| RS-6 | Dual rendering targets (TTY + stdout) drift | Medium | Without a pure view-from-state design, JSON and TTY diverge (M5 fails). |
| RS-7 | Multi-provider harmonization semantics | Medium | "Harmonize vs side-by-side vs call out differences" needs a defined conflict policy (units, timestamps, station distance) or the UI lies. |
| RS-8 | Performance under many locations / animation / radio | Medium | bubbletea re-render cost × locations × animation tick, plus audio decode, could breach M8/M9 budgets. Mitigation: render only dirty regions, tick coalescing, bounded caches. |
| RS-9 | Dependency version drift (the reference CLI pins older charm versions) | Low | bubbletea v0.24 vs current v1.x/v2 — API differences affect go-studs compatibility. |
| RS-10 | Alert safety / trust | Medium | A tool that *might* miss an alert is worse than none in a wildfire zone; needs honest "last updated / stale" indicators and M3 coverage testing. |
| RS-11 | No-watermark enforcement | Low | Harness injects attribution trailers by default; calibration now in place; red-team checks each phase. |

### Open Questions — RESOLVED (HUM LEAD, 2026-08-23)

| ID | Question | Ruling (verbatim) | Recorded implication |
|---|---|---|---|
| OQ-1 | MVP cut | "must include radio" | v0.1 = NWS + locations + alerts + `--json` + **radio**. RS-2/RS-3 remain High; feed-degradation design is in PLAN scope. |
| OQ-2 | Geographic scope | "Global data; US only alerts + NWS data" | A keyless global provider (Open-Meteo or equivalent) is a **v1 requirement**; alerts are US/NWS-only. T-E sharpened (T-E′). |
| OQ-3 | Charm stack version | "current bubble tea ; we'll update go-studs" | Track current bubbletea/lipgloss/bubbles; go-studs upgrade is a tracked pre-work item. RS-9 closed by decision. |
| OQ-4 | go-studs dependency | "start local; then we'll determine what do depending on how we diverge or augment (may require up streaming back to go-studs first)" | `go.mod replace` → local path for now. RS-1 → Medium-deferred with a checkpoint at PLAN exit and again before SHIP. |
| OQ-5 | City disambiguation | "both; we'll do hints while typing; and we'll always show zip code alongside" | R-2 sharpened (R-2′): type-ahead hints; zip displayed alongside every location label. |
| OQ-6 | Alert latency ≤ 60s | "correct" | M2 ratified. OS-level desktop notifications: not required (assumed "no" absent a ruling). |
| OQ-7 | Audio playback | "if there's already a go package that handles audio we can use, want to avoid users having to install other stuff to make watchlist work" | C-6 is now firm: pure-Go playback; no external player dependency. AI-5 becomes a go/no-go evaluation. |
| OQ-8 | Config/key storage | "best practices (no secrets in code) and shouldn't require OS keychain logins every time" | Keys in user config file (0600, outside repo); no keychain prompts. C-3 sharpened (C-3′). |
| OQ-9 | Harmonization policy | "we should support all and let users choose if they want. NWS is always the definitive source of truth when in doubt." | All three modes (harmonize / side-by-side / diff call-out) are user-selectable; **NWS is the tie-break authority**. T-F sharpened (T-F′). RS-7 mitigated by policy. |
| OQ-10 | Playlist (R-10) | "architect to be configurable. We'll refine as we build the specific feature" | R-10 stays in scope as an architectural requirement (configurable view sequences + dwell); UX detail deferred to its feature. |
| OQ-11 | Fire/evac v1 source | "Start with hotspots, evac orders next" | v1 hotspots; evac v1.x (R-9 split into R-9a/R-9b). *Source superseded by D-10/OQ-19: v0.1 = HMS keyless default + WFIGS incidents, FIRMS as optional user key.* |
| OQ-12 | Terminal minimums | "needs to be terminal width aware and support dynamic resizing for default views, for single output it will follow the go-studs terminal width calculation at invocation time before printing to stout" | T-B′: live views respond to resize events; T-C′: stdout mode uses go-studs width calculation once at invocation. No fixed minimum declared — PLAN defines graceful floor. |

### Sharpened requirements (post-ruling)

| ID | Sharpened text |
|---|---|
| R-2′ | Users can enter a city name or zip; the app offers type-ahead hints while typing; every rendered location label always shows its zip code alongside. |
| R-9a | Users can see active fire hotspots for/near configured locations — NOAA HMS keyless default + WFIGS incidents; NASA FIRMS via optional user key (per D-10/OQ-19). *(v1)* |
| R-9b | Users can receive evacuation orders/notices where a licensed feed exists. *(post-v1)* |
| T-B′ | Live views are terminal-width aware and re-layout on dynamic resize. |
| T-C′ | Stdout/`--json`/`--report-only` output computes width once at invocation via go-studs' terminal-width calculation, then prints. |
| T-E′ | NWS required (US observations/forecast/alerts); a keyless global provider required for non-US current/forecast; paid providers additive. |
| T-F′ | Harmonize, side-by-side, and difference call-out are all supported and user-selectable; NWS is the definitive tie-break. |
| C-3′ | No secrets in code or repo; user keys in a 0600 config file; no OS-keychain prompts on launch. |
| C-6′ | Audio playback must be pure-Go; users must not install external players/codecs. |

### Context carried forward

- **Problem statement:** refined and locked (5/5) above — ratify at gate.
- **Sharpening observations:** SH-1..SH-5.
- **Backlog review:** no `06_docs/02_features/*/backlog.yml` exists yet (new project) — skipped.

## Brief Metadata

| Field | Value |
|---|---|
| Header | **NEW MAJOR PROJECT \| "Watchpost CLI"** |
| Scope adjective | Major |
| Project type | Project (CLI application) |
| Project name / branch | `watchpost-cli` / `feature/watchpost-cli` |
| LEVEL | LEVEL-1 |
| SEV | SEV-0 (HUMAN LEAD) |
| Phase instructions | FULL GIT; FULL DOCS; FULL REPORTS; FULL DIAGRAMS; FULL RCC (DISCOVER); FULL PLAN; FULL TDD |
| Theme | BRTOPS |

**Completeness scorecard**

```
PROJECT BRIEF — COMPLETENESS CHECK
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  [✓] Header              — Major Project | "Watchpost CLI"
  [✓] Directives          — LEVEL-1; SEV-0; 7 FULL directives
  [✓] Summary / Intent    — clear what/why; problem statement refined 5/5
  [✓] Requirements        — 9 functional + 2 derived; 12 technical
  [✓] Metrics of Success  — 9 metrics, anti-solution checked (agent-proposed; needs ratification)
  [✓] Tech Constraints    — 6 enumerated
  [✓] Considerations      — prior art, design system gap, stakeholders
  Required sections: 5/5 complete
  Overall: READY FOR DISCOVER — APPROVED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

BRIEF SUFFICIENCY CHECK
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  [✓] Agent can identify WHAT to investigate       — AI-1..AI-11
  [✓] Agent can identify WHO to consider           — 5 stakeholder groups
  [✓] Agent can assess WHERE to look               — NWS docs, provider docs, go-studs, the reference CLI
  [✓] Agent can generate targeted questions        — OQ-1..OQ-12
  [✓] Agent can identify risks                     — RS-1..RS-11
  Sufficiency: READY FOR DISCOVER
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Decision Log

| # | Timestamp | Decision | By | Rationale |
|---|---|---|---|---|
| D-1 | 2026-08-23 | Thin A2DH install; BRTOPS theme | HUM LEAD | "This is an empty directory, so we'll do a thin-install here." |
| D-2 | 2026-08-23 | SEV-0, LEVEL-1, all FULL directives | HUM LEAD | Intake directive line (verbatim in Brief Metadata). `.a2dh.yml` `sev_level` set to 0 accordingly. |
| D-3 | 2026-08-23 | No AI attribution watermarks in any output | HUM LEAD | "We NEVER include <authored by: {{agent}}> in any outputs." Canonical calibration copied into `_a2dh/CALIBRATIONS_a2dh.md` (bundle was stale). |
| D-22 | 2026-08-24 | B2 gate rulings R-1/R-2 | HUM LEAD | R-1: "How would a person 'know' which firezone they are in? We should, instead, use their locations that they input to infer appropriate firezones" — the wizard now infers the fire weather zone from the chosen location (NWS /points, async, degrades to generic prompt offline) and Q3 becomes zone-aware. R-2: "Mask + give user the option to unmask" — FIRMS key masked by default, ctrl+r toggles reveal. |
| D-21 | 2026-08-24 | Live UAT with HUM LEAD required for all view/mock implementations (B3+) | HUM LEAD | "once we start implementing mocks, we're going to also need to do live UAT with me to ensure it looks and behaves correctly." Standing rule: from B3 on, every milestone that renders UI closes ONLY after (1) scripted-PTY machine verification of mechanics AND (2) a live HUM LEAD UAT session for look/behavior — UAT findings dispositioned like red-team findings before the milestone gate. |
| D-20 | 2026-08-23 | B1 exit approved; 15 pending P10 exemptions RATIFIED (density pattern x8, analyzer false-positives x2, sched main loop, schema generator x4); B2 GO | HUM LEAD | "GO" at the B1 exit gate presentation. |
| D-19 | 2026-08-23 | Mock behavior rulings | HUM LEAD | Keys (defaults, still D-15-swappable): `a` About, `A` Alert Details, `ctrl+a` Add Location. Location services → v0.2 (v0.1 default location = first priority location). Units: `f`/`c` global LIVE-swappable display toggle (SI stays internal; --json stays SI). Recent-history cap ~50 with eviction approved. Header glyph per enabled provider (FIRS=FIRMS) approved. Volume UI 0–100 approved. |
| D-18 | 2026-08-23 | G-9(a): `report --every` ratified as R-12d surface (amends OQ-18 — plain-text live mode in v0.1; JSON `--watch` stays v0.2); PLAN exit approved | HUM LEAD | "G-9: a. Approved for Exit" |
| D-17 | 2026-08-23 | Install ergonomics requirement (T-M) + view-mock inventory M-1..M-6 | HUM LEAD | "I would ultimately like humans to be able to install this via a curl command like CLI amp… if we need to adjust the plan to account for this, now is the time." Mock inventory: dashboard, location detail, add-location modal, search modal, help modal, setup view — HUM LEAD supplies on request at the milestones needing them. |
| D-16 | 2026-08-23 | Architecture & roadmap approved (P-1: no internal/ wrap v0.1; P-2: omit diffs[] from JSON v1.0; P-3: single golden system; P-4: two height breakpoints) | HUM LEAD | "Approved" — architecture.md is the artifact of record; B0..B6 roadmap ratified. |
| D-15 | 2026-08-23 | Keybindings hot-swappable, not locked; only `?`=help fixed | HUM LEAD | "don't lock in key bindings just yet… keeping the architecture flexible enough for us to 'hot-swap' key-bindings as we discover / iterate." + "the only one for sure is '?' for help". Layered KeyMap (view→mode→global), data-driven, config-overridable, runtime-swappable. |
| D-14 | 2026-08-23 | Project structure: Option C (domain-first + thin platform; modes read only platform/snapshot; app/ composition root) | HUM LEAD | "Approved for Option C" — structure-proposal.md is the artifact of record. |
| D-13 | 2026-08-23 | G-8 approved (C-6″ restatement); DISCOVER exit approved | HUM LEAD | "G-8: Approved. Approved for Discover Exit." Plus PLAN directive: propose project structure for HUM LEAD review/approval first — "I want to review and provide input on potential structure to keep plugins modular and easy for humans to understand as they navigate the project structure." |
| D-12 | 2026-08-23 | DISCOVER exit gate rulings G-1..G-7 | HUM LEAD | G-1 approved (v0.1 = glanceable surface; OS notifications v0.2 candidate). G-2 NOT approved — radio is critical: "we need to find a way to make this work - might even consider downloading the NOAA radio statements (I think those are plain-text) and figure out a way to create our own 'radio' solution using voice. Either way this is a critical feature, and one I don't think anyone does comprehensively." → AI-13 synthesized-radio investigation. G-3a/b adopted (R-12, R-13). G-4a/b approved (M6 redefined; M8 re-derivation authorized). G-5 NOT approved — "Use latest bubbletea - might give us better capability and re can always re-tool app specific components for later go-studs usage." → latest bubbletea; v1-derived frame claims re-verified at PLAN. G-6 approved (config edited). G-7 evidence: "Lack of a single, comprehsive, solution with lower overall resource consumption." |
| D-11 | 2026-08-23 | Post-approval supersession edits to this brief (RS-1 disposition note, OQ-11/R-9a supersession notes per red-team Docs D-1..D-3); version bumped v1.1.0 → v1.2.0 | Agent (red-team remediation), pending HUM LEAD exit approval | Approved-document amendments must be version-bumped and logged (round-2 R2-3). |
| D-10 | 2026-08-23 | OQ-13..OQ-19 rulings (see tier2-synthesis.md §E) | HUM LEAD | Acknowledge charm reinterpretation; defer wxradio contact; attribution in About frame/modal (not footer); cities15000 + online fallback; repo private until go-studs distribution decided (SHIP gate); --watch/--fail-on-stale to v0.2; fire = HMS + WFIGS + optional FIRMS key. |
| D-9 | 2026-08-23 | go-studs pivot-ready architecture | HUM LEAD | "I am the author of go-studs… As long as we architect things to make that changeover/pivot easy, it becomes a secondary concern." Component/render layer behind an internal abstraction so go-studs can be swapped (personal fork / bubbletea-native / upstream) without touching views. RS-1 → Medium-deferred. |
| D-8 | 2026-08-23 | Brief approved; problem statement ratified; metrics M1–M9 ratified | HUM LEAD | "Approved; Ratify problem statement; ratify metrics" |
| D-7 | 2026-08-23 | All merges to `main` via PR using the A2DH PR template | HUM LEAD | "we always do PRs to push to main using our A2DH template." Template to be provisioned before first PR. |
| D-6 | 2026-08-23 | Add memory (M8) and CPU (M9) metrics | HUM LEAD | "I want some memory footprint and cpu usage targets to ensure our tool remains performant." Targets agent-proposed. |
| D-5 | 2026-08-23 | OQ-1..OQ-12 rulings (see Open Questions table) | HUM LEAD | Two reply batches, verbatim recorded. |
| D-4 | 2026-08-23 | Git initialized; `main` + `feature/watchpost-cli`; personal identity | Agent (FULL GIT mandate) | Branch-before-write. Identity verified: the personal address. No push. |
