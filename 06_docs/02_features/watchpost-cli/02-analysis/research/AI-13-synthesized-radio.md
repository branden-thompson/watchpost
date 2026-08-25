# AI-13: Synthesized Weather Radio Feasibility (TTS from NWS Text Products)

**Status:** Research complete | **Feeds:** G-2 ruling (NWR critical, ~10% stream coverage) | **Related:** AI-1 (NWS API), AI-4 (streams), AI-5 (oto/go-mp3 playback)

**Headline:** NWR itself is already TTS — since 1997 (CRS/DECtalk "Paul"), today BMH/NeoSpeech reads NWS text products aloud. We can rebuild the same pipeline: the source products are public, plain-text, and free. The only hard problem is the TTS engine under C-6′.

## 1. Source Text: What NWR Reads, and Which API Products Reconstruct It

A broadcast cycle (repeats ~5 min, varies by station): station ID → hazards/HWO → current conditions roundup → zone forecast → climate summary → marine/tropical when applicable. The actual BMH broadcast script is **not published** — no public CRS/BMH text feed exists — but it is assembled from these public products:

| Product | Endpoint (`api.weather.gov/products/types/{id}/locations/{wfo}`) | Cadence | Narration fit |
|---|---|---|---|
| **RWR** Regional Weather Roundup | confirmed live (e.g., `/RWR/locations/OHX`) | hourly (~:10) | Excellent — tabular obs, easy to verbalize |
| **HWR/BRT** Hourly (Weather Radio) Roundup | in `/products/types` list; sparse coverage | hourly | Excellent — written *for* radio where issued |
| **ZFP** Zone Forecast Product | per WFO | 4×/day + updates | Excellent — plain prose forecast, NWR's backbone |
| **HWO** Hazardous Weather Outlook | per WFO | daily + event-driven | Good — leads the cycle |
| **NOW** Short Term Forecast | per WFO | event-driven | Good |
| **SPS** Special Weather Statement | per WFO | event-driven | Good |
| **CWF** Coastal Waters Forecast | coastal WFOs | 4×/day | Good (marine segment) |
| **AFD** Area Forecast Discussion | per WFO | 2–4×/day | Poor — technical jargon; skip or "nerd mode" |
| **CAP alerts** | `/alerts/active?point=` (AI-1) | real-time | Excellent — `description` + `instruction` are read-aloud prose |

All return `productText` as plain text (fixed-width; needs de-wrapping and abbreviation expansion: "FL 250"→"flight level", wind "NW 10 MPH", etc.).

## 2. TTS Engines by Install Friction

| Path | Engines | Ships by default? | Go integration |
|---|---|---|---|
| **OS-native** | macOS `say` (AVSpeechSynthesizer); Windows SAPI via PowerShell `System.Speech` | **Yes, both** — present on every stock macOS/Windows | `os/exec`, no cgo. `say -o out.wav --data-format=LEI16@22050`; PowerShell `SetOutputToWaveFile` |
| **OS-native Linux** | espeak-ng / speech-dispatcher | **Partial** — GNOME desktops ship speech-dispatcher+espeak-ng (Orca a11y); servers/minimal don't | `os/exec espeak-ng --stdout` → WAV |
| **Pure Go, offline** | *(none exist)* | — | GitHub survey: every Go TTS lib is a network wrapper — htgo-tts (Google Translate, 218★, unofficial/ToS-gray), edge-tts-go (Edge endpoint), ElevenLabs clients. No flite/espeak port, no pure-Go ONNX runtime that can run VITS/piper models |
| **Neural local** | piper (C++), sherpa-onnx (cgo Go bindings) | No — user must install binary/model (~50 MB) | exec or cgo; best quality, violates C-6′ |
| **Cloud** | OpenAI/Google/Amazon/ElevenLabs | No — API key + network + cost | HTTP; nonstarter for a resilience tool |

**Interpretation of C-6′:** exec'ing a binary the OS ships is not an "external install" — the user installs nothing. That reading gives zero-install TTS on **macOS (always)**, **Windows (always)**, **Linux desktop (usually)**; Linux server/minimal needs `apt install espeak-ng` (one package, ~4 MB) or falls back to text-only mode.

## 3. Quality & Latency; Pipeline Fit

- **Tiers:** espeak-ng = robotic (ironically period-authentic — sounds like 1997 CRS "Paul"); macOS `say`/SAPI = clearly intelligible, decent; piper = near-human. Real NWR listeners *expect* synthetic voices — quality bar is uniquely low here.
- **Critical:** all three OS engines can **render to WAV file/stdout instead of speaking to the device**. Decode WAV (stdlib-level) → PCM → existing oto/v3 player. One audio path, our volume control, no device contention, and we can mix in tones. Latency: seconds to render a cycle; render ahead of playback.

## 4. Narration Pipeline (Proposed)

```
fetch (RWR + ZFP + HWO + NOW/SPS + /alerts/active)  [reuse AI-1 client+cache]
  → normalize: de-wrap, expand abbreviations, strip headers/UGC codes
  → script assembly: "This is Watchpost synthesized weather radio for {area}…"
      [alerts first w/ 1050 Hz WAT tone] → conditions → forecast → outlook → repeat marker
  → TTS to WAV (per-segment, cached by product issuance ID)
  → PCM → oto ring buffer, looping
```

- **Cadence:** regenerate a segment only when its product's `issuanceTime` changes (poll per AI-1 cadence); loop the assembled cycle continuously like NWR's ~5-min rotation. New warning → interrupt loop, play tone + alert immediately.
- **1050 Hz WAT tone:** trivial — synthesize a 10 s sine in ~10 lines of Go straight into the PCM stream. (SAME FSK bursts also synthesizable later; cosmetic.)
- **Cache:** per-segment WAV keyed on product UUID; only changed segments re-render.

## 5. Differentiation Check (Honest)

GitHub search found: one hobby **"NOAA Weather Radio Simulator"** (ruserprooyuncu-hub, Python, Windows-only desktop app, 0 stars, Aug 2026) doing CRS-style TTS with alert cycling — so the idea is *not* literally unprecedented. Also `asl_weather_announce` (ham-radio TTS announcements) and `ws4kp` (WeatherStar 4000 visual emulator — music bed, no NWS-product narration). **Verdict:** HUM LEAD is essentially right — nothing cross-platform, nothing terminal-native, nothing that merges *real streams where available + synthesis everywhere else*, and nothing in Go. The combination is genuinely novel; the raw concept has one obscure precedent.

## 6. OPINION (AI-4 recommendation)

**Architecture:** **Real stream when available → synthesized everywhere else**, presented as one seamless "Radio" mode with a visible `[LIVE]` / `[SYNTH]` badge. Real streams carry the authentic BMH voice and local color; synthesis is the universal fallback that turns 10% coverage into 100% — and also the *offline-ish* fallback (API-cached products still narrate when streams die). Synthesized-first would discard the authenticity users specifically want from NWR.

**TTS path:** macOS → `say`; Windows → PowerShell System.Speech; Linux → espeak-ng if present, else offer one-line install hint + text-only ticker fallback. All via `os/exec` → WAV → oto. **C-6′ must be restated:** from "no external installs" to **"no installs beyond what the OS ships"** (and on minimal Linux, graceful degradation, never a hard requirement). Escape hatch: optional `--tts-cmd` config so power users can plug piper for neural quality without us shipping it.

**Strongest counter-argument:** we now depend on three per-OS exec paths — PowerShell startup latency, macOS voice availability, Linux fragmentation — which is exactly the platform-matrix pain C-6′ existed to avoid, and abbreviation-expansion of NWS fixed-width text is a long tail of embarrassing mispronunciations ("NW wind" → "en-double-you"). Mitigation: segment-level golden-file tests on the normalizer, and the text ticker always renders even when audio fails.

## Sources

- https://api.weather.gov/products/types (ZFP/RWR/HWO/NOW/SPS/CWF/AFD/HWR/BRT/TWB confirmed)
- https://api.weather.gov/products/types/RWR/locations and `/RWR/locations/OHX` (hourly issuance verified live)
- https://en.wikipedia.org/wiki/NOAA_Weather_Radio (cycle content, CRS→VIP→BMH voice history, 1050 Hz/SAME spec)
- https://www.weather.gov/nwr/ (all-hazards scope)
- https://github.com/hegedustibor/htgo-tts · https://github.com/surfaceyu/edge-tts-go (Go TTS = network wrappers)
- https://github.com/ruserprooyuncu-hub/NOAA_Weather_Radio_Simulator (sole prior art, Windows/Python)
- https://github.com/pnearing/asl_weather_announce (adjacent prior art)
