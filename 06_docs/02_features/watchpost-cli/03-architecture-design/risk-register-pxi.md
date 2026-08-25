# PLAN Artifact 2 — Risk Register P×I Pass

| Field | Value |
|---|---|
| Phase | PLAN entry · SEV-0 |
| Purpose | Red-team F-7 condition: every risk rated Probability × Impact with evidence, not a bare severity label |
| Date | 2026-08-23 |
| Scales | P: Low <20% · Med 20–60% · High >60% (chance it materializes during v0.1) — I: Low (annoyance) · Med (feature degraded / rework) · High (metric broken, ship blocked, or user harmed) |
| Status | Approved with the architecture packet (D-16); RS-19..21 added by PLAN red-team — see architecture.md §10.9 |

| RS | Risk | P | I | Score | Evidence for rating | Mitigation state |
|---|---|---|---|---|---|---|
| RS-1 | go-studs IP/distribution blocks public release | **Med** | **High** | **P2** | README/LICENSE conflict + enterprise-gated repo are facts (AI-11); but HUM LEAD authored it and holds a personal first version (D-13 context), so an amicable path exists | Pivot architecture (D-9) + private repo (OQ-17); SHIP gate; **impact contained to release timing, not build** |
| RS-2 | Scope breadth stalls v0.1 | **Med** | **Med** | P3 | 5 domains + audio + synth for a solo builder; but phased roadmap + every domain now has a concrete, researched design | Roadmap is PLAN's next artifact; v0.2 deferrals recorded |
| RS-3 | Radio streams unavailable for user's location | **High** | **Low** | P3 | ~117/1000+ transmitters is a measured fact (AI-4) — *will* happen for most users | Impact demoted by G-2 SYNTH fallback: degraded to voice-quality difference, not absence |
| RS-5 | Provider rate-limit/ToS enforcement (NWS throttle, wxradio blocklist) | **Low** | **Med** | P4 | NWS load ≈30 req/min at 25 locations, far under observed tolerance (AI-1); wxradio risk is real but per-user connections are the intended use (AI-4 §3) | Token bucket + backoff + caching designed; single-connection rule; SYNTH fallback |
| RS-8 | M8/M9 budgets don't close | **Med** | **Med** | P3 | P-1 arithmetic: worst case leaves ~5MB headroom — genuinely uncertain until measured | **CLOSED by measurement**: S1 1.83% CPU / flat heap; S2 ~18MB total, ~22MB headroom (spikes S1/S2). Re-verify end-to-end in VALIDATE |
| RS-10 | Alert miss/staleness harms trust ("worse than none") | **Low** | **High** | **P2** | Dual-UGC matching + issuance-anchored M2 + 100%-coverage M3 replay harness are designed; residual: NWS 5xx windows (observed intermittent, AI-1 §5) | R-13 disclaimer + stale badges adopted (G-3b); last-good serving; `warnings[]` first-class |
| RS-12 | Volunteer-stream ToS action against watchpost | **Med** | **Low** | P4 | No third-party-client grant is a fact; but per-user direct listening matches stated intent; SYNTH removes sole-path dependence | OQ-14 operator contact pre-SHIP; graceful 403 handling |
| RS-13 | Open-Meteo non-commercial clause conflicts with future distribution | **Low** | **Low** | P5 | Personal OSS use squarely permitted (AI-2 §2); only bites on a commercialization pivot nobody plans | Documented in About (OQ-15) |
| RS-14 | Glyph-width drift breaks layouts | **Med** | **Low** | P4 | Emoji mismatch is measured (AI-9); mitigation is exclusion + `--ascii`, both cheap and designed | In-component fallback; runewidth version alignment at PLAN |
| RS-15 | Fire-zone user on delayed HMS misses fast fire | **Med** | **Med** | P3 | HMS analyst delay is hours (AI-3) vs FIRMS URT <1min — the gap is real for keyless users; re-rated UP from Low per F-7 | FIRMS-key setup prompt in fire zones (OQ-19); NWS Red Flag alerts remain real-time regardless |
| RS-16 | Hostile stream input exploits decoder | **Low** | **Med** | P4 | Requires compromised relay + decoder vuln; go-mp3 is battle-used though archived (AI-5) | Vendored decoder; ICY+MP3 fuzz gated at BUILD; https-preferred |
| RS-17 *(new)* | TTS exec-path fragility (PowerShell latency, macOS voice availability, Linux fragmentation) | **Med** | **Low** | P4 | Three per-OS exec paths is AI-13's own counter-argument; text ticker always renders | Per-OS adapters behind one interface; golden tests on normalizer; `--tts-cmd` escape hatch |
| RS-18 *(new)* | bubbletea-latest churn (v2 API instability) invalidates v1-derived designs | **Med** | **Med** | P3 | G-5 ruling adopts latest; v2's renderer/API differences are documented but our claims were v1-derived (AI-9) | **CLOSED**: G-5 source-verified v2.0.9; claims table records survives/restated; pins exact (architecture Pins row) |

**Register hygiene:** RS-4/6/7/9/11 closed or demoted as recorded in the Discovery Report §6. Priority order for PLAN attention: **P2: RS-1, RS-10** → P3: RS-2, RS-8, RS-15, RS-18 → P4/P5 monitored.
