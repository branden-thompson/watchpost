# Spike S1 — Radio Pipeline CPU/RSS (measured)

**Date:** 2026-08-23 · **Machine:** macOS arm64, Go 1.24.4 · **Metrics under test:** M9 (radio ≤10% of one core), M8 contribution · **Provenance note:** subagent built the harness and completed the 120s full-pipeline run; it stalled before the decode-only variant and write-up, which were completed by the coordinator directly. All numbers below are measured, none estimated.

## 1. Setup

- Deps resolved: `github.com/ebitengine/oto/v3 v3.4.1`, `github.com/hajimehoshi/go-mp3 v0.3.4` (exactly the AI-5 recommendation).
- Builds: **both `CGO_ENABLED=0` and `=1` binaries built successfully** (`spike-cgo0`, `spike-cgo1`) — confirms AI-5's zero-toolchain claim on macOS arm64. Measurements ran on `spike-cgo1`.
- Stream: **LIVE** — `https://wxradio.org/CA-Monterey-KEC49` (HTTP 200, `icy-metaint=16000`, `icy-name="NOAA Weather Radio KEC49 162.55MHz"`, `icy-br=32`, MP3 22050 Hz). ICY metadata blocks parsed and stripped in-line (`StreamTitle` observed). Playback via oto at `SetVolume(0)` (silent).

## 2. Method

`measure.sh <mode> <dur> <url> <label>`: run the pipeline binary, skip first 15 s, then sample `ps -o %cpu,rss` every 5 s until exit; program logs `runtime.MemStats` every 10 s. Full pipeline 120 s (n=22 samples); decode-only (ICY strip + go-mp3 → `io.Discard`, no oto) 60 s (n=10).

## 3. Results

| Variant | CPU mean | CPU max | RSS mean | RSS max | HeapAlloc trend |
|---|---|---|---|---|---|
| Decode-only (no audio out) | **0.90 %** | 6.8 % (startup) | **20.2 MB** | 20.2 MB | flat |
| Full pipeline (decode + oto playback) | **1.83 %** | 2.7 % | **41.3 MB** | 41.7 MB | flat — HeapAlloc oscillates 0.9–3.4 MB across GC cycles over 120 s, `Sys` steady at 12.6 MB, no growth |

## 4. Verdict vs budgets

- **M9 radio CPU ≤10 %: PASS with 5× margin** — 1.83 % mean, 2.7 % max for the complete live pipeline (AI-5 estimated ~5 %; reality is better).
- **M8:** heap cost is trivial (≤3.4 MB); the standout is **RSS composition**: oto adds ~21 MB of resident footprint on macOS (41.3 vs 20.2 MB), which is AudioToolbox/CoreAudio framework mappings, not Go heap. In-app the marginal cost will be shared with the process, but ~20 MB of macOS audio-framework residency is a real line item for the 80 MB radio-on budget: projected stack-up ≈ runtime+TUI baseline + geodata 13 MB (S2) + audio ~21 MB + app state — inside 80 MB with margin, to be proven end-to-end in VALIDATE.
- **Leak guard (M8 anti-solution clause): PASS** — flat heap over the full window.

## 5. Surprises / caveats

- ICY metadata on this mount was present but empty (`StreamTitle=''`) — the TUI must not depend on ICY titles for station naming (use the transmitter table).
- CPU max 6.8 % on decode-only was the startup sample; steady state never exceeded 2.7 % in either variant.
- Single stream, single 32 kbps mount, macOS-only, `ps`-sampled at 5 s granularity; no reconnect/failure-path exercised. Linux/Windows RSS composition (no AudioToolbox) will differ — re-measure in VALIDATE.
- Volume-0 playback exercises the full output path but not audibility; a human listen-check belongs to BUILD smoke tests.

## 6. Recommendation

Adopt the AI-5 stack as-is (`oto/v3 v3.4.1` + vendored `go-mp3 v0.3.4`); keep the oto `Context` initialized **lazily on first tune-in** so the ~21 MB AudioToolbox residency is paid only when radio is used; keep the ICY-strip reader from this spike as the starting point for `domains/radio/stream`; add a reconnect/backoff wrapper (not exercised here) and the RS-16 fuzz harness at BUILD.
