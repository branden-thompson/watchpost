# Discovery Report — Watchpost CLI

| Field | Value |
|---|---|
| Report | discovery-report (FULL depth) |
| Feature / Branch | watchpost-cli · `feature/watchpost-cli` |
| Phase | DISCOVER exit · SEV-0 · HUMAN LEAD · Theme BRTOPS |
| Directives | FULL GIT · FULL DOCS · FULL REPORTS · FULL DIAGRAMS · FULL RCC · FULL PLAN · FULL TDD |
| Date | 2026-08-23 |
| Status | **APPROVED — HUM LEAD, 2026-08-23** ("G-8: Approved. Approved for Discover Exit.") |

## 1. Executive Summary

DISCOVER ran two research tiers (12 investigations, AI-1..AI-12, all evidence-cited, several live-probed), two cross-cutting syntheses, 19 HUM-LEAD-ruled open questions, and a full red-team round (4 axes + phase lens + 3 confirmed personas). Every technical unknown that blocked planning is resolved: the radio pipeline is feasible pure-Go (MP3), NWS supports batched sub-minute alert polling within ToS, the keyless provider set is Open-Meteo + NWS + HMS/WFIGS, geocoding is embedded-first GeoNames + online fallback, and the go-studs dependency is pivot-ready by decision (D-9). Red-team verdict: **SHIP-WITH-CONDITIONS** — conditions are seven gate items (G-1..G-7, §8) and the phase-entry conditions transferred to PLAN/BUILD/SHIP (§9). Recommendation: **proceed to PLAN** upon gate rulings.

## 2. Locked Problem Statement & Positioning

> "Terminal-centric people who monitor weather across one or more locations — especially those in severe-weather or wildfire regions — have no single glanceable surface for current conditions, forecasts, and live alerts, and so they juggle multiple apps, sites, and radio, seeing alerts late or not at all."

Ratified 5/5 (D-8). **HUM LEAD evidence of record (G-7, verbatim):** "Lack of a single, comprehensive, solution with lower overall resource consumption." Scope ruling (G-1): the statement's "not at all" clause is scoped to *while at the terminal* — v0.1 is a glanceable surface; OS notifications are a v0.2 candidate. Post-survey honesty (AI-12): the claim survives **narrowly** — no terminal tool combines live multi-location conditions, streaming alerts, and a cycling display — but the pieces exist separately. **Positioning of record:** *closest prior art is WeatherStar 4000+ (browser cycling dashboard) and termidar (live terminal radar with alerts); watchpost differs by being terminal-native, multi-location with a national summary, sub-minute alert surfacing for the full watch/advisory spectrum, and multi-provider diffing — with JSON output as table stakes, not a differentiator.* The strongest counter-argument of record (AI-12 §4): a ws4kp browser tab + phone WEA + `curl wttr.in` already delivers a cycling dashboard, life-safety pushes, and agent-readable JSON at zero build cost — watchpost's residual value is terminal-nativeness and multi-location alert breadth, serving a small population. For "take-shelter-now" at the user's current location, phone WEA is and remains the authoritative channel (§ safety framing, R-13 proposed); watchpost adds alert **breadth**, **remote-location monitoring**, and **desk presence**. Scope note (B-4): v0.1 solves the locked problem for **US** users; non-US users get conditions/forecasts without alerts — an acknowledged non-goal.

## 3. Requirements (final DISCOVER state)

Functional R-1..R-11 and technical T-A..T-L per the approved brief, with post-ruling sharpenings R-2′/R-9a/R-9b/T-B′/T-C′/T-E′/T-F′/C-3′/C-6′ (brief, "Sharpened requirements"). Changes from this exit:

| ID | New/changed | Source |
|---|---|---|
| R-9a′ | Fire hotspots v0.1 = NOAA HMS (keyless default) + NIFC WFIGS incidents; NASA FIRMS via optional user key, prompted in setup for fire-zone locations | D-10/OQ-19 |
| **R-12 (proposed, G-3)** | Accessibility family: (a) alert severity always rendered as text label + position — color additive, never sole channel; (b) `--no-animation` flag alongside `--ascii`; (c) `--report-only` documented as the screen-reader surface; (d) contrast minima per token pair with light-background fallback | Red-team A-1..A-5 |
| **R-13 (proposed, G-3)** | Safety framing: visible "not a substitute for official warnings (WEA/NOAA Weather Radio)" disclaimer + stale-data honesty indicators as v0.1 acceptance criteria | Red-team F-2; AI-4 source disclaimer |
| v0.2 deferrals | `--watch` JSON-Lines, `--fail-on-stale`, evac orders (R-9b), paid providers | OQ-11/18; D-10 |

## 4. Research Base (12 investigations)

| Doc | Verdict in one line |
|---|---|
| AI-1 NWS | Live-probed; batched zone alerts every 20s meet M2/M3; `expires`-driven caching; UA mandatory |
| AI-2 Providers | Open-Meteo sole keyless global; CC-BY + non-commercial; Pirate→OWM→WeatherAPI as keyed v0.2 |
| AI-3 Fire | HMS keyless default + WFIGS incidents; FIRMS keyed upgrade; evac via NWS CAP later |
| AI-4 NWR | No official stream; wxradio/weatherUSA volunteer Icecast MP3 ~32kbps; ~10% transmitter coverage; ToS risk RS-12 |
| AI-5 Audio | **GO**: oto/v3 + go-mp3 pure-Go, zero toolchain, ~5% CPU est.; AAC no-go v0.1 |
| AI-6 go-studs | Pure render-string lib, zero charm imports; width calc canonical; strip-test bans app tokens; 7 net-new components mapped |
| AI-7 the reference CLI | Reuse root-router/atomic-config/plugin-registry/verify gates; avoid hand-rolled loops, 0644 config |
| AI-8 Geocoding | Embedded GeoNames cities15000 + US postal, Open-Meteo online fallback; Nominatim forbids type-ahead |
| AI-9 Terminal | Emoji width hazard; `--ascii` needed; 1×100ms tick policy; alt-screen dashboard / inline mini-player; 40/60/80/120 breakpoints |
| AI-10 JSON | `Snapshot`-pivot contract; harmonized/by_provider/diffs; schema semver + CI validation |
| AI-11 go-studs IP | MIT/personal-name LICENSE vs "internal the employer library" README; enterprise-gated repo; `go install` ignores `replace` |
| AI-12 Alternatives | Existence claim survives narrowly; ws4kp/termidar closest; WEA wins take-shelter-now; positioning §2 |
| AI-13 Synth radio | **GO** — NWR is already TTS (CRS/BMH); RWR/ZFP/HWO/NOW/SPS/CAP plain-text reconstruct the cycle; OS-native TTS (say/SAPI/espeak-ng) via exec → WAV → oto; zero-install on macOS/Windows, mostly Linux; one obscure prior art (Windows/Python hobby app) — the cross-platform terminal-native LIVE+SYNTH combination is novel |

Syntheses: `02-analysis/tier1-synthesis.md`, `tier2-synthesis.md` (composition/convergence/contradiction + OQ/RS ledgers).

## 5. Constraints (final)

C-1..C-6 per brief with C-3′ (0600 keys file, no keychain prompts, no secrets in repo) and **C-6″ (G-8 approved): no installs beyond what the OS ships — playback pure-Go (oto/go-mp3); TTS via OS-native engines; minimal-Linux degrades to text ticker + install hint; optional `--tts-cmd` override.** Additions from red-team: no secret in any machine-mode output (schema constraint); https-preferred streams with warning on http; checksummed embedded datasets; `.gitignore` Go artifacts at BUILD entry.

## 6. Risk Register (authoritative as of DISCOVER exit)

P×I scoring pass is a PLAN-entry task (F-7); ratings below are severity-class carried from syntheses.

| RS | Risk | Rating | Disposition |
|---|---|---|---|
| RS-1 | go-studs IP/access | Medium-deferred | D-9 pivot architecture + private repo (OQ-17) + SHIP gate "distribution decided"; employer-IP/OSPO questions deferred to SHIP gate (F-8) |
| RS-2 | Scope breadth | High→managed | Phased roadmap is PLAN's first deliverable; v0.2 deferrals recorded |
| RS-3 | Feed availability (radio) | Medium | Best-effort framing + text-product fallback; ~10% coverage disclosed |
| RS-5 | Rate limits/ToS | Medium | NWS batching+caching designed; wxradio blocklist risk residual (→G-2) |
| RS-8 | Performance | Medium | Frame policy + spikes S1/S2; S2 **gates PLAN exit** (P-1) |
| RS-10 | Alert trust | Medium | Dual-UGC matching, stale indicators, `warnings[]` first-class; R-13 proposed |
| RS-12 | Volunteer-stream ToS | Medium→Low | G-2 ruling: synthesis de-soles the volunteer streams — a blocklisted/dead stream degrades to `[SYNTH]`, not silence; OQ-14 revisit stands pre-SHIP |
| RS-13 | Open-Meteo non-commercial | Low | Documented; About attribution (OQ-15) |
| RS-14 | Glyph-width drift | Medium | `--ascii` in-component fallback; runewidth version alignment at PLAN |
| RS-15 | HMS latency in fire zones | Low (re-rate at PLAN) | FIRMS-key setup prompt (OQ-19) |
| **RS-16** | Attacker-controlled audio input (hostile stream → archived decoder) | Medium | ICY+MP3 fuzz = gated BUILD requirement; go-mp3 vendored |
| RS-6 | TTY/JSON drift | Mitigation designed | Closes when the Snapshot parity tests land (they are PLAN-entry conditions, §9) |
| RS-7 | Harmonization semantics | Mitigated by policy | NWS tie-break + `source{}` block (OQ-9) |
| RS-11 | Watermarks | Demoted | Process checklist; round-1 sweep clean |
| Closed | RS-4 (audio portability), RS-9 (charm drift) | — | Resolved by verified evidence (AI-5 source-verified; AI-6 zero charm imports) |

## 7. Critical Analysis (red-team round 1)

`red-team: SHIP-WITH-CONDITIONS · multi-agent (4 sectioned dispatches) · scope:feature · personas:[infosec, a11y, perf]`

Full findings and the 47-row disposition ledger: **`08-reports/red-team-discover.md`**. Section verdicts: Code SHIP-WITH-CONDITIONS · Hygiene CONDITIONAL · Docs REVISE (fixed this commit) · Business CHALLENGE · Phase SHIP-WITH-CONDITIONS · InfoSec CONDITIONAL · A11y **FAIL** (→G-3) · Perf CONDITIONAL. Watermark sweep clean. Key meta-finding: risks named in research were dropped at synthesis — remediated by RS-16 registration, S2 gating, R-12/R-13 proposals, and this report as the authoritative register. Round 2 (fresh lenses, Step 9) runs on the remediated corpus before PLAN work begins.

## 8. Gate Items — RULED (HUM LEAD, 2026-08-23; D-12)

| # | Ruling | Consequence |
|---|---|---|
| G-1 | **Approved** | v0.1 = glanceable surface; problem statement scoped to "while at the terminal"; OS notifications v0.2 candidate |
| G-2 | **Not approved — radio stays critical** | Descope rejected. New avenue ruled: reconstruct a "radio" experience from NWS plain-text products narrated by synthesized voice, alongside real streams where available — "a critical feature, and one I don't think anyone does comprehensively." → **AI-13 investigation (this exit); R-5″: radio = real stream when available + synthesized voice everywhere**; RS-12 partially mitigated (volunteer streams no longer the sole path). **AI-13 verdict: GO.** Architecture of record: one seamless Radio mode — real stream when available, synthesized everywhere else, visible `[LIVE]`/`[SYNTH]` badge; alerts interrupt with a synthesized 1050 Hz WAT tone; per-segment WAV cache keyed on product issuance ID; abbreviation-normalizer with golden tests; text ticker always renders when audio fails. **Requires constraint restatement C-6″ (G-8 below).** |
| G-3a | **Adopted** | R-12 (a11y family) is a v0.1 requirement |
| G-3b | **Adopted** | R-13 (safety framing) is a v0.1 requirement |
| G-4a | **Approved** | M6 = "components accepted into whichever shared component home D-9 resolves to" |
| G-4b | **Approved** | M8 re-derivation authorized pending spike S2 |
| G-5 | **Not approved — use latest bubbletea** | "might give us better capability and we can always re-tool app specific components for later go-studs usage." Consequence: AI-9's v1-derived renderer/frame-budget claims and the lipgloss/bubbles API surface must be **re-verified on latest bubbletea at PLAN entry** (new PLAN condition); code finding #1 dispositioned by HUM LEAD ruling |
| G-6 | **Approved** | `.a2dh.yml`: description set this commit. `languages: [go]` — approved but **activation deferred to BUILD entry**: P10 fails closed with no go.mod to certify (verified live: 4 tools refuse an empty module scope), and the phase-exit validate-100% gate takes precedence. Declared as a commented block with this rationale; BUILD-entry checklist item added §9. |
| G-7 | **Provided** | Evidence integrated into §2 |
| **G-8** | **Approved** — C-6″ adopted (§5) | AI-13: no pure-Go offline TTS exists; OS-native engines (macOS `say`, Windows System.Speech, Linux espeak-ng) cover the need via `os/exec` → WAV → our oto pipeline. Proposed: C-6′ "pure-Go audio, no external installs" → **C-6″ "no installs beyond what the OS ships; on minimal Linux, graceful degradation to text ticker + one-line install hint; optional `--tts-cmd` power-user override."** Playback stack stays pure-Go throughout. |

## 9. Conditions transferred to later phases

**PLAN entry:** P×I pass on risk register; spike S2 (geodata memory) gates PLAN exit; spike S1 (radio CPU/RSS) before architecture lock; **re-verify AI-9 frame/renderer claims on latest bubbletea (G-5)**; **AI-13 synthesized-radio architecture folded into the audio design (G-2: LIVE+SYNTH mode, normalizer golden tests, per-OS TTS adapters behind one interface)**; Snapshot immutability contract; parity sentinel fixtures; M2/M3 test designs (replay harness, teatest loops, `-race`); secret-redaction schema constraint; contrast minima; runewidth alignment; simplification list (diffs[], oggvorbis, dual goldens, compat machinery, height breakpoints); exit-code semantics; end-user personas.
**BUILD:** activate `languages: [go]` in `.a2dh.yml` + first `a2dh p10 check` once go.mod exists (G-6); severity-as-text rule; `--no-animation`; NextFrame()-in-Update-only; https-preferred streams; keys-file isolation + README warning; dataset checksums; ICY+MP3 fuzz (gated); `.gitignore` Go artifacts; FIRMS setup prompt.
**VALIDATE:** cross-terminal validation of AI-9's documented-not-validated cells; M8 soak; M3 replay.
**SHIP gate:** go-studs distribution decided (+ employer-IP/OSPO answers); scrub private strings from 06_docs before repo goes public; NWS API roadmap check; OQ-14 wxradio contact revisit.

## 10. Exit Gate Checklist (DISCOVER)

| Gate | Status | Evidence |
|---|---|---|
| requirements_documented | PASS | Brief + §3 (R-12/R-13 pending G-3) |
| constraints_identified | PASS | Brief + §5 |
| risks_assessed | PASS | §6 register (P×I pass scheduled at PLAN entry) |
| critical_analysis_complete | PASS (conditional) | §7; `red-team-discover.md` rounds 1+2 (near-converged); final convergence = HUM LEAD spot-check of G-rulings at this gate |
| report_published | PASS on commit | This document |
| a2dh validate 100% | PASS | 18/18 (re-run live during round 2) — carries standing FLAG MEDIUM: a2dh framework upgrade available (noted since INIT; framework maintenance, not a watchpost gate) |
| human_approval | **PASS** — G-1..G-8 ruled; exit approved (D-13, 2026-08-23) | G-1..G-7 + exit approval |

## 11. Recommendation

Proceed to **PLAN** upon G-1..G-7 rulings. PLAN's first deliverables: phased v0.1 roadmap (data layer → TTY core → alerts → radio → fire → playlist), architecture with the pivot-ready component abstraction (D-9), spikes S1/S2, and the FULL DIAGRAMS architecture set.
