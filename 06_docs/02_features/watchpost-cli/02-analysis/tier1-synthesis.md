# DISCOVER Tier 1 — Cross-Cutting Synthesis

Date: 2026-08-23 · Inputs: `research/AI-1`, `AI-2`, `AI-4`, `AI-5`, `AI-6`, `AI-7`, `AI-8` · Phase: DISCOVER (SEV-0, HUMAN LEAD)

## A. Composition effects

1. **Radio v0.1 is feasible end-to-end (AI-4 + AI-5).** AI-4 found the real-world sources emit Icecast chunked-HTTP MP3 at ~32 kbps (113/117 mounts); AI-5 found `ebitengine/oto/v3` + `hajimehoshi/go-mp3` decode that pure-Go with zero C toolchain on macOS/Linux/Windows. Neither finding alone settles OQ-1+OQ-7; together they do. AAC/Ogg mounts (2/117) are "unsupported station" labels, not blockers.
2. **Alert correlation to radio via SAME codes (AI-1 + AI-4).** NWS `/points` already returns `county` UGC; NWR SAME codes are `"0"+county FIPS`. One lookup chain serves both alert zone-batching and transmitter selection: lat/lon → `/points` → {forecastZone, county} → {alert batch key, SAME → call sign → stream mount}.
3. **One normalized schema serves TTY, JSON, and the diff view (AI-2 + AI-1 + AI-6).** AI-2's SI-internal schema with a `source{provider, model_or_station, distance_km, issued_at}` block, fed by NWS (station obs, `wmoUnit:*`) and Open-Meteo (model grid), is exactly what go-studs' pure `Render() string` components consume and what `--json` emits — satisfying M5 parity by construction (RS-6 mitigated architecturally).
4. **Embedded geodata + location cache compose with the M8 budget (AI-8 + M8).** GeoNames cities15000 + US postal ≈ 7–10 MB decoded. That is 20–25 % of the 40 MB RSS target before any weather state; PLAN must lazy-load/compact the index (or use mmap-style binary search over the compressed blob) rather than materialize structs.
5. **go-studs width calc + bubbletea resize compose cleanly (AI-6 + AI-7 + OQ-12).** `rendering.GetTerminalSize()` (ioctl `/dev/tty` → stdin → `$COLUMNS` → 80) is called once for stdout mode; in TTY mode bubbletea's `WindowSizeMsg` is passed explicitly via `WithTerminalWidth`. the reference CLI never propagated resize to children — watchpost must.

## B. Convergence observations

6. **Three independent sources say "bubbletea owns the loop; go-studs renders strings."** AI-6 (no charm imports; frame-step APIs are pull-based), AI-7 (the reference CLI's hand-rolled loops are the anti-pattern), and the M9 budget (event/tick-driven only) all point to a single `tea.Program` with `tea.Tick`-driven animation and go-studs as a stateless render layer.
7. **Everything keyless converges on Open-Meteo (AI-2 + AI-8).** It is the only keyless global weather source *and* the best keyless type-ahead geocoder (prefix search, timezone, population, postcodes in one payload). One provider client, two endpoints.
8. **Caching is load-bearing everywhere (AI-1, AI-2, AI-4, AI-8).** NWS `expires`-driven caching, Open-Meteo 10-min cache, wxradio `status-json.xsl` ≥5-min cache, permanent geocode cache. A single cache subsystem with per-source TTL policy is a PLAN-level component, not a per-provider detail.

## C. Contradiction observations

9. **OQ-3 ruling vs. AI-6 fact.** HUM LEAD: "current bubbletea; we'll update go-studs." AI-6: go-studs has zero charm imports — there is nothing to update. **Resolution proposed:** the ruling's intent (modern charm stack) is honored by watchpost adopting bubbletea v1.x/lipgloss v1.x directly; go-studs pinned via `replace` at HEAD. No go-studs change required for v0.1. *Needs HUM LEAD acknowledgement.*
10. **"Must include radio" (OQ-1) vs. ~10 % transmitter coverage + volunteer ToS (AI-4).** Radio *capability* is v0.1; radio *availability* for a given location is best-effort. **Resolution proposed:** R-5's "when available" clause already covers this; the v0.1 UX must show "no stream for {call sign}" + NWS text-product fallback (HWO/AFD) rather than fail. Recommend contacting the wxradio operator pre-release. *Needs HUM LEAD ruling on contacting the operator.*
11. **OQ-8 "0600 file, no keychain" vs. AI-2 key-leak risk.** Not a conflict — AI-2 confirms the main leak vectors are URL query strings in logs and shell history, which 0600 storage plus log redaction address. Record as a BUILD requirement (SEC).
12. **the reference CLI as reference (T-D) vs. the reference CLI reality (AI-7).** T-D said "use what the reference CLI uses"; the reference CLI is mostly not a bubbletea app, has no JSON mode, vendors go-studs in-tree, and writes 0644 config. **Resolution:** T-D is satisfied by the *library set* (cobra, bubbletea, bubbles, lipgloss, runewidth, go-studs), not by copying the reference CLI's architecture. Reuse its root-router, atomic config write, plugin-interface + explicit registration, color gate, and `make verify` gates.

## D. Risk signal status

| RS | Status after Tier 1 | Evidence |
|---|---|---|
| RS-1 go-studs org/IP | **Still open** → Tier 2 AI-11 | OQ-4 defers mechanism; licence text not yet read |
| RS-2 scope breadth | **Partly mitigated** | Radio/geocode/provider paths are now concrete; PLAN can phase |
| RS-3 feed availability | **Escalated for radio, resolved for weather** | ~117/1000+ transmitters; volunteer ToS (AI-4). Weather: NWS + Open-Meteo solid |
| RS-4 audio portability | **Resolved** | oto v3 cgo-free on all 3 OSes (AI-5, verified in source) |
| RS-5 rate limits/ToS | **Partly mitigated** | NWS batching + `expires` caching (AI-1); Open-Meteo 10k/day (AI-2); wxradio blocklist risk remains |
| RS-6 TTY/stdout drift | **Mitigated by design** | Composition #3 |
| RS-7 harmonization semantics | **Mitigated** | `source{}` block + NWS tie-break (OQ-9) |
| RS-8 perf | **Partly mitigated** | Radio ≈5 % CPU est. (AI-5); geodata memory now a named cost (#4) |
| RS-9 charm drift | **Closed** | AI-6: no charm in go-studs |
| RS-10 alert trust | **Partly mitigated** | AI-1: match on forecastZone AND county UGC; stale-obs handling; 5xx last-good |
| RS-11 watermarks | Unchanged | Calibration active |
| **RS-12 (new) volunteer-stream ToS** | **Open — Medium** | wxradio "no relaying/harvesting"; no explicit third-party-client grant |
| **RS-13 (new) Open-Meteo non-commercial** | **Open — Low** | Blocks any future paid/ad-supported distribution; document |

## E. Open question status

| OQ | Status |
|---|---|
| OQ-1..OQ-12 | Ruled by HUM LEAD (brief D-5). Tier 1 *validated* OQ-1/7 (feasible), *sharpened* OQ-3 (#9), OQ-5 (hybrid embedded+Open-Meteo), OQ-12 (GetTerminalSize semantics). |
| **OQ-13 (new)** | Acknowledge OQ-3 reinterpretation: go-studs needs no charm bump; watchpost adopts bubbletea v1 directly? |
| **OQ-14 (new)** | Contact wxradio.org operator for third-party-client permission before v0.1 release? |
| **OQ-15 (new)** | Accept GeoNames CC-BY attribution + Open-Meteo CC-BY attribution line in footer/`--about`? (Required by licence; confirming UX placement.) |
| **OQ-16 (new)** | Bundle `cities15000` (≈26k cities, 3.3 MB) or `cities5000` (+2 MB) for offline type-ahead? Recommend 15000 + online fallback; measure miss rate in VALIDATE. |

## F. Implications for Tier 2 / PLAN

13. **Tier 2 dispatch set:** AI-3 (fire: FIRMS hotspots v1, evac later — per OQ-11), AI-9 (terminal capability matrix — now with concrete glyph needs from AI-6's chart family: `▁▂▃▄▅▆▇█`, 8 arrow glyphs, braille), AI-10 (JSON contract — seeded with AI-2's normalized schema + AI-1 alert fields), AI-11 (go-studs licence/IP read for RS-1).
14. **PLAN must own:** cache subsystem (#8), scheduler (AI-1's two-tier: 20 s alert batch / `expires`-driven weather / 60–120 s obs), provider interface (NWS, Open-Meteo, future keyed), geocoder interface (embedded → online), audio pipeline (HTTP/ICY → ring → go-mp3 → oto), view registry (explicit registration per AI-7), and a memory plan honoring #4.
15. **Pre-BUILD spike recommended:** a 1-day throwaway that plays one wxradio mount via oto+go-mp3 and measures CPU/RSS — converts AI-5's estimate into M8/M9 evidence before architecture is locked.
