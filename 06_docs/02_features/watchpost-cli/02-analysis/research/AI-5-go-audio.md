# AI-5 — Pure-Go Audio Playback Feasibility (NWR Icecast radio)

Scope: playback + decode stack only. Stream sources are covered by AI-4. Assumption: MP3-over-Icecast primary, AAC secondary.

## 1. Output / playback layer

| Library | macOS | Linux | Windows | Last release | Notes |
|---|---|---|---|---|---|
| `github.com/ebitengine/oto/v3` | **No cgo** — AudioToolbox loaded via `ebitengine/purego` (verified: `driver_darwin.go` has no `import "C"`; go.mod requires purego v0.10.2) | **No cgo** — PulseAudio via pure-Go `jfreymuth/pulse` (native socket protocol), ALSA fallback `dlopen`s `libasound.so.2` at runtime; no `libasound2-dev` to build | **No cgo** (WASAPI/winmm syscalls) | v3.4.1 (2026-08-13); v3.5 alphas active | Ebitengine core project, well maintained. `Player` has `SetVolume`, `Play/Pause/Close`; one `Context` per process (cannot be closed/re-created — keep a single context, create/close players). Buffer size configurable (`BufferSize`), default ~ tens of ms; underruns → audible gaps, so feed from a goroutine, never from the Bubble Tea update loop. |
| `github.com/gopxl/beep/v2` (`speaker`) | same as oto | same as oto | same as oto | v2.1.1 (2025-01-05) | Thin layer over oto v3; adds `Streamer` composition, `effects.Volume`, `Ctrl` (pause), mixer. Bundles go-mp3/oggvorbis/flac decoders. Convenient but adds float64 resampling overhead; easy to use directly. |
| `hajimehoshi/oto` v2 | cgo | cgo (ALSA headers) | no cgo | superseded | Deprecated path → use v3. |
| `github.com/gen2brain/malgo` (miniaudio) | **cgo** | **cgo** (`-ldl`) | **cgo** | active | Excellent engine but requires a C toolchain everywhere. Disqualified by the zero-install rule. |

Runtime prerequisites for oto v3: Linux host must have a PulseAudio/PipeWire-pulse server *or* `libasound.so.2` (both present on stock Ubuntu Desktop; headless servers may have neither). macOS/Windows: nothing.

## 2. Decoders (pure Go)

| Format | Pure-Go decoder | Maturity | Verdict |
|---|---|---|---|
| MP3 | `github.com/hajimehoshi/go-mp3` v0.3.4 | Archived 2023-04 (read-only) but stable, widely used (beep, Ebiten). Accepts plain `io.Reader`; `Length()`=-1 and `Seek` errors on non-seekers — fine for live streams. Output: 16-bit stereo LE at source rate. | **GO** |
| AAC-LC / HE-AAC (ADTS) | `github.com/colespringer/waxflow/codec/aac` (MIT, AAC-LC + HE-AAC v1/v2, ADTS, "ffmpeg-differential verified", zero deps); also `skrashevich/go-aac` (LGPL), `thesyncim/goaac` (GPL), `arabian9ts/aac-go` | All 2025–26, ~0 stars, single-author, unproven. `Eyevinn/mp4ff` parses only. | **Experimental** — prototype behind a build tag; not a v0.1 commitment |
| Ogg Vorbis | `github.com/jfreymuth/oggvorbis` | Mature, pure Go, streaming | GO (rare for NWR) |
| Opus | `github.com/pion/opus` (pure Go decoder, RFC 6716; coverage of CELT/hybrid modes not stated — treat as SILK-focused); `hraban/opus` is cgo | Immature for music/broadcast | Defer |
| HLS | `github.com/grafov/m3u8` (playlist) + `asticode/go-astits` (TS demux) → still needs AAC decode | Plumbing fine; decode problem is AAC's | Defer |

## 3. Icecast / SHOUTcast specifics

- Send `Icy-MetaData: 1`; if response has `icy-metaint: N`, strip a metadata block after every N audio bytes (1 length byte ×16 then `StreamTitle='…';`). Implement as an `io.Reader` wrapper; ~40 lines. Title is useful for the TUI (station/alert text).
- Body is an infinite HTTP response (Icecast sends no `Content-Length`; SHOUTcast may reply `ICY 200 OK` — Go's `net/http` rejects that; handle with a raw `net.Dial` fallback or a `http.Transport` with `Proto` tolerance). No `http.Client.Timeout`; use a per-read idle deadline (e.g. 15 s) via `net.Conn.SetReadDeadline` or a watchdog goroutine.
- Reconnect: exponential backoff 1s→30s with jitter, cap attempts, resume on network change; surface state (`Connecting/Playing/Stalled/Reconnecting`) to the TUI.
- Buffering: 64 kbps = 8 KB/s. Pre-roll 2–3 s (24 KB) before starting the player; ring/pipe of ~32–64 KB compressed bytes between the HTTP goroutine and the decoder; oto's own PCM buffer ~100–200 ms. Total < 1 MB.
- go-mp3 decodes frame-by-frame from `io.Reader` and blocks on `Read`; pass it `io.Pipe`/bufio over the ICY-stripped body. It does not support mid-stream sample-rate changes (rare on relays; on error, re-create the decoder).

## 4. Cross-compilation / install story

| OS | `go install …@latest` with no C toolchain | `CGO_ENABLED=0` build |
|---|---|---|
| macOS (no Xcode CLT) | **Works** — oto v3 uses purego; no clang needed | Works (purego supports CGO_ENABLED=0 on darwin) |
| Ubuntu without `libasound2-dev` | **Works** — no headers needed; runtime needs Pulse/PipeWire or `libasound.so.2` | Works (purego fakecgo path on linux/amd64,arm64) |
| Windows | **Works** | Works |
| Cross-compile from CI | Works for all three from one host with `CGO_ENABLED=0` | `GOOS/GOARCH` matrix → `./dist/` |

Verdict: zero-toolchain install on all three OSes is achievable. The only cgo-dependent candidate (malgo) is excluded.

## 5. Resource cost (64 kbps MP3)

- No published benchmark for go-mp3 found; community reports and Ebiten usage put pure-Go MP3 decode at roughly 1–3 % of one modern core for 44.1 kHz stereo (decode is ~40–100× real-time). go-mp3 issue #28 ("100 % CPU") concerns an old oto busy-loop, fixed in later oto; not the decoder.
- oto v3 playback: callback-driven on macOS/Windows, pulse socket writes on Linux — <1 % typically.
- Memory: go-mp3 decoder state < 1 MB; ring buffers < 1 MB; oto context ~ few hundred KB. Go runtime + Bubble Tea baseline (~15–25 MB RSS) dominates. **Expect ~5 % CPU and ~30–40 MB RSS** — within the ≤10 % / ≤80 MB budget, to be confirmed by a spike benchmark.
- Risk: beep's float64 streamer pipeline adds ~2× decode cost; using go-mp3 → oto directly avoids it.

## 6. OPINION — GO / NO-GO

**MP3 Icecast: GO.** Pure-Go, zero-install playback on macOS, Linux, Windows is achievable today.

Recommended stack:
- `github.com/ebitengine/oto/v3 v3.4.1` — output
- `github.com/hajimehoshi/go-mp3 v0.3.4` — MP3 (archived; vendor/fork if fixes needed)
- `github.com/jfreymuth/oggvorbis` — optional Vorbis
- Own ~300-line Icecast client (ICY metadata, ICY-200 fallback, backoff)
- Skip beep unless mixing/effects are needed

**AAC: NO-GO for v0.1 commitment.** Pure-Go AAC decoders now exist (waxflow, go-aac) but are months old with no adoption; ship AAC as an experimental build-tagged option, and treat AAC-only stations as "unsupported — choose an MP3 relay" in v0.1.

Compromise: none required for the toolchain. The real residual constraint is Linux *runtime*: a headless box with no Pulse/PipeWire and no `libasound.so.2` cannot play — acceptable for a desktop TUI; document it.

Strongest counter-argument: the decoder is unmaintained (archived) and oto's purego path is newer than its cgo path — a subtle macOS/Linux ABI regression in oto or a malformed-frame crash in go-mp3 would land on us with no upstream. Mitigation: pin versions, vendor go-mp3, add a fuzz test on the ICY+MP3 path, and keep "external player" (`mpv`) as an opt-in escape hatch, never a requirement.

## Sources

- https://github.com/ebitengine/oto (README prerequisites; `driver_darwin.go`, `driver_unix.go`, `go.mod`)
- https://pkg.go.dev/github.com/ebitengine/oto/v3?tab=versions
- https://github.com/hajimehoshi/go-mp3 (archived; `source.go`; issues #21, #28)
- https://github.com/gopxl/beep ; https://pkg.go.dev/github.com/gopxl/beep/v2?tab=versions
- https://github.com/gen2brain/malgo
- https://github.com/pion/opus
- https://github.com/colespringer/waxflow ; https://pkg.go.dev/search?q=aac+decoder
- https://github.com/jfreymuth/oggvorbis ; https://github.com/jfreymuth/pulse
- https://github.com/grafov/m3u8 ; https://github.com/asticode/go-astits
