# B3 Infrastructure Ledger (UAT-driven)

> One row per infrastructure change made during B3 UAT, with the finding that caused it. The
> per-session narrative lives in `b3-uat-log.md`; the architecture truth in
> `03-architecture-design/architecture.md` §11 and `caching.md`.

| UAT | Trigger (what was on screen) | Infrastructure change | Where |
|---|---|---|---|
| 59 | Carlsbad `UNKNOWN n/a` | 4-station observation fallback chain + preferred station; hourly-forecast rehydration; fail-soft fragments; retry-before-cadence; bounded parallel fetch; singleflight points | `domains/weather/nws`, `platform/snapshot`, `platform/sched` |
| 59 | Maritime missing for SF / SD / Seattle / Miami | upper-case NDBC product ids; waves from true buoys + water temp from any station; fail-soft | `domains/marine/ndbc` |
| 60 | Station in the row | `WX STN` / `DIST` columns; station distance from `/stations` geometry; `platform/geo` | `platform/render`, `platform/geo` |
| 61 | Tides approved | `domains/marine/coops` (predictions, level, currents); TTL memo (`platform/memo`) | `domains/marine/coops` |
| 64 | Tides missing for favourites | priority lane in `httpx`; concurrent fan-out in every marine provider; publish per provider; CO-OPS on its own paced client; tide-station fallback | `platform/httpx`, `platform/sched`, `app` |
| 69 | Lookups re-requested the whole list | `Assembler.SetLocations`; `Scheduler.Update`; incremental `commit` | `platform/snapshot`, `platform/sched`, `app` |
| 71 | TODAY HI `n/a` after sunset | gridpoint HIGH/LOW fill with provenance; **URL-keyed HTTP cache** (caller TTL → server max-age → none; memory + disk; singleflight; negative cache); `platform/memo` retired | `platform/httpx`, `domains/weather/nws`, `app` |
| 77 | RADIO step 2 (Synth) | `domains/radio/synth` (products, normalizer, composer, PCM source, `Voice` seam: macOS `say` / Piper, pinned installer); `player.StartSource`; Live→Synth failover; setup + first-tune install | `domains/radio/synth`, `domains/radio/player`, `app/radio.go`, `app/setup.go` |
| 76 | RADIO step 1 | `tools/nwrtable` (NWS CCL.js → vendored transmitter table, 1,035 sites with coordinates); `domains/radio/stream` (directories, resolver); `domains/radio/player` (Icecast → go-mp3 → oto, pure Go, cross-OS); dashboard `Radio` hook + `RadioStatusMsg` | `tools/nwrtable`, `domains/radio`, `app/radio.go`, `modes/tty` |
| 74 | btop still 82–91 threads after UAT 73 | live-process profiling (`WATCHPOST_DEBUG_PPROF=1`): `time.LoadLocation` opened a zoneinfo file per location per publish under the assembler lock → `platform/tz` memo; publishes coalesced (50 ms) so a burst yields one snapshot; disk-cache reads bounded to 4; recent schedulers start 10 ms apart | `platform/tz`, `app`, `platform/httpx`, `platform/sched` |
| 73 | btop: 177 threads / 95 MB | pure-Go DNS resolver + 8 conns/host + in-flight caps (16 / 8 per lane); memory cache budget 8 MB with LRU, bodies > 2 MB disk-only; raw-body disk format with an async writer; 11 lint findings | `platform/httpx` |
| 72 | Rehydration cost evaluation | batched RECENT alerts (−25 req/min); NDBC `5day2` (−4 MB/h); inland grids remembered (−9 MB/h); `KindMarineObs` 10-min tier; `KindForecastHourly` on demand for RECENT rows (−8 MB/h) | `platform/sched`, providers, `app`, `modes/tty` |

## Measured baselines (2026-08-24)

| Product | Size | Server lifetime |
|---|---|---|
| NWS observation | 4 KB | 5 min |
| NWS alerts (batched) | 5–60 KB | 5 s |
| NWS daily forecast | 13 KB | ~next issuance |
| NWS hourly forecast | 162 KB | 1 h |
| NWS gridpoint | 228 KB | ~next issuance |
| NDBC realtime2 → 5day2 | 207 KB → 23 KB | 10 min |
| CO-OPS products | < 1 KB | none declared (`no-store`) |
| NWS points / stations | 4 KB / 10 KB | 1 day |

Steady state before UAT 72: ~40 requests/min, ~30 MB/h (60 locations). After: ~15 requests/min,
~9 MB/h, with the maritime observations refreshed every 10 min instead of hourly.

## Perf pass (UAT 73, headless wiring-exact probe, 60 locations, 90 s)

| Metric | Before | After |
|---|---|---|
| OS threads (peak, never retired) | 137 | **15** |
| Resident (`sys`) | 80.5 MB | **54.7 MB** |
| Live heap after GC | 29.2 MB | **14.6 MB** |
| Memory cache tier | 17.4 + 6.6 MB | **8.0 + 2.9 MB** |
| Goroutines (steady) | 211 | 214 (50 per-location schedulers × 4 tiers — cheap, by design) |

Root causes: the launch burst opened hundreds of connections at once, each blocking an OS thread in
a cgo DNS lookup or a synchronous disk-cache write (Go never retires threads); the memory tier was
unbounded and the disk format base64-inflated every body and copied it on read.

## Perf pass, part 2 (UAT 74, the real TUI process in a pty, warm cache)

| Metric | UAT 73 binary | After UAT 74 |
|---|---|---|
| Mach threads (steady, `ps -M`) | 144 | **20** |
| Resident (`ps rss`) | 89 MB | **76 MB** |

Why the headless probe missed it: its cache dir had been warmed by its own first run, spreading
publishes out; the real launch fired ~300 publishes in one second, each `Snapshot()` opening 60
zoneinfo files under the assembler lock (140 threads), and then — once that was fixed — 200
scheduler goroutines making short cache-file syscalls in the same 80 ms (90 threads). Lesson
recorded in `06-key_learnings/b3-ux-backwards.md` §2: measure the *process*, not the pipeline.

Diagnostics kept: `WATCHPOST_DEBUG_PPROF=1` serves `/debug/pprof/` on 127.0.0.1:6060 (threadcreate,
goroutine, heap). Loopback only, off by default.

## Radio dependency pins (UAT 76)

| Module | Version | Why |
|---|---|---|
| `github.com/ebitengine/oto/v3` | **v3.5.0-alpha.11** | the only release whose Linux driver builds with `CGO_ENABLED=0` (purego); v3.4.1 is cgo on Unix. Pre-release — re-pin to 3.5.0 when it ships. |
| `github.com/hajimehoshi/go-mp3` | v0.3.4 | archived upstream, stable; vendor/fork if a frame-walker fix is ever needed (AI-5). |

## Voice artifacts (UAT 77, SHA-256-pinned in `domains/radio/synth/install.go`)

| Artifact | Source | Size |
|---|---|---|
| Piper 2023.11.14-2 linux x86_64 / aarch64 / armv7l, windows amd64 | github.com/rhasspy/piper releases | 22–27 MB |
| Voice `en_US-lessac-medium` (.onnx + .json) | huggingface.co/rhasspy/piper-voices v1.0.0 | 63 MB |
| macOS | built-in `say` (Piper's macOS archive ships no libraries) | — |

Install dir: the OS cache dir (`~/.cache/watchpost/piper`, `%LOCALAPPDATA%\\watchpost\\piper`) — re-downloadable, safe to delete.

## 0.9.0 exit measurements (2026-08-25, real TUI in a pty at 133×44, 10 favourites + 50 recent)

| Run | launch → full view | threads (steady) | RSS | Notes |
|-----|--------------------|------------------|-----|-------|
| Warm cache | **550 ms** (target ≤ 3 s) | 23 | 78 MB flat over 90 s | the M1 number |
| Cold cache (`~/Library/Caches/watchpost/http` moved aside) | **1.1 s** (target ≤ 8 s) | 20 | 79 → 97 MB over 30 s as the memory tier fills and the disk tier writes | the warm-launch queue item from UAT 59 is closed by these two numbers |

Method: `expect` drives `dist/watchpost` with `WATCHPOST_DEBUG_TIMING=1`, samples `ps -o rss=` and
`ps -M … | wc -l` every 5 s, sends `q`; see `05-debugging/debugging-ledger.md` D1.

## Synth playing, 90 s (UAT 98, exit build, draining pty, Oceanside)

| Width | app CPU (Viz off → on) | `say` child | RSS |
|---|---|---|---|
| 133 cols | 0.4–3.6 % → 3–11 % | 4–49 % bursts per segment (macOS TTS) | 82 → 116 MB |
| 400 cols | 1–10 % → 5–11 % | same | 88 → 131 MB |

The first ~20 MB step (t≈15 s) is the oto audio context (S1: allocated on first use) plus the
`say` output buffers; the rendered-audio cache is now mono and capped at 24 segments (was 64
stereo — up to ~85 MB under Repeat), which bounds long sessions but was ≤ 10 MB of this 90 s
window. The remaining ~10 MB over 75 s is not isolated in a window this short (Go heap headroom,
the 8 MB memory cache tier filling); **the 1-hour soak with radio on is the M8 item at VALIDATE**.
Width costs a few percent of one core; no stall at 400 cols.

## Visualizer budget (UAT 92)

- Feed: `player.Tap` ring of 3,072 mono int16 (one 2048 window + ~70 ms slack); mutex-guarded copy
  on every audio read (~8 KB) and every UI frame.
- Analysis: one 2048-point radix-2 FFT per frame, no allocation past the frame's 10-float result;
  19.7 µs/op, 80 B, 1 alloc on this Mac (`go test ./domains/radio/spectrum -bench Bands`). The
  silence gate skips the FFT entirely when nothing plays.
- Cadence: 50 ms `vizTick` (20 fps) only while Viz is on AND (playing OR bars settling); Viz off or
  idle = zero extra wakeups. The 300 ms shimmer tick is unchanged.
- Style reference: CLIAmp (MIT) `ui/visualizer.go` / `ui/vis_bars.go` / `player/tap.go`, read
  2026-08-24; no code copied, no dependency added.

## Still queued

- Great Lakes water levels (CO-OPS lake datums), Tahoe (no free source)
- B4 radio audio + visualizers; the NWR transmitter callsign belongs to the player line
- Warm-launch measurement with the disk tier (M1 target warm ≤ 3 s)

## M8 soak — 2026-08-25 (VALIDATE, UAT 123)

Real dashboard in a pty (133×44), 5 favourites + RECENT, **Synth playing on Repeat: Watchlist for
60 minutes**, System Voice, macOS. Sampled every 5 min (`ps -o rss,pcpu`; threads via `ps -M`):

| t | RSS MB | CPU % | threads |
|---|---|---|---|
| 0 | 144 | 1.0 | 21 |
| 5 | 166 | 3.1 | 28 |
| 10 | 213 | 1.7 | 27 |
| 15 | 221 | 4.3 | 29 |
| 20 | 191 | 3.7 | 29 |
| 25 | 162 | 1.5 | 31 |
| 30 | 203 | 2.2 | 29 |
| 35 | 203 | 5.6 | 31 |
| 40 | 173 | 2.0 | 29 |
| 45 | 207 | 4.3 | 29 |
| 50 | 215 | 4.8 | 30 |
| 55 | 142 | 2.0 | 29 |
| 60 | 194 | 3.6 | 32 |

Reading: RSS **oscillates between 142 and 221 MB with no trend** (the rendered-audio cache filling to
its 40-segment bound and being released as voices/segments churn; the 55-minute sample is below the
first); CPU 1–6 % of one core; threads settle at ~30 (audio context + scheduler pool). The 90-second
"creep" recorded at UAT 98 does not continue over an hour — **no leak**. Clean exit on `q`.

Second hour (same setup, `WATCHPOST_DEBUG_PPROF=1`, dump-on-linger armed): RSS 146–202 MB, no trend
(156 → 202 peak at 30 min → 146 at 40 min → 191 at 60); CPU 2–8 %; 28–31 threads; **`q` exited in
0 s**, no dump taken. The first run's straggler did not recur — recorded as a harness artefact of the
first script (its `expect eof` window), not an app fault.
