# DISCOVER — Live-run findings (instrumented process + HUM LEAD's process)

Two processes observed 2026-08-26: HUM LEAD's `~/.local/bin/watchpost` v0.9.4 (PID 67943, started
2026-08-25 17:55, sampled hourly → `baseline-pid67943.log`) and an instrumented run of the same tree
(`WATCHPOST_DEBUG_PPROF=1`, all traffic through a counting CONNECT proxy, Synth on Repeat: Watchlist at
volume 5, samples every 5 min, pprof heap/goroutine/threadcreate every 30 min → `discover-run/`).

## LR-1 — FUNCTIONAL DEFECT: the weatherUSA relay directory has never been reachable from Go

**Symptom.** The connection log showed 4 connections to `radio.weatherusa.net` every 5 minutes, spaced
0 / 2 / 4 s — the httpx retry ladder's signature. Reproduced with the app's own client
(`platform/httpx`, app UA, retries off): `remote error: tls: handshake failure` in 270 ms; `wxradio.org`
fine. `curl` succeeds.

**Root cause.** `radio.weatherusa.net` accepts only **RSA key-exchange** cipher suites (it negotiated
`TLS_RSA_WITH_AES_128_GCM_SHA256`, HTTP/1.0). Go 1.22 removed RSA key exchange from the default suite
list; the `tlsrsakex` GODEBUG that re-enabled it is **removed in Go 1.27** (`fatal error: removed GODEBUG
"tlsrsakex"`). Bisection: default / h2 forced / h2 off / TLS 1.0 min / TLS 1.2 max all fail; an explicit
RSA-only `CipherSuites` list succeeds. The mounts themselves are **plain HTTP** (`http://radio.weatherusa.net:80/NWR/*.mp3`)
and stream fine; only the `https://…/status-json.xsl` directory fails. `http://radio.weatherusa.net/status-json.xsl`
returns the same 30,629-byte document over plain HTTP.

**Effect.** Every 5 minutes while the radio is tuned: 4 failed TLS handshakes (retry ladder — L2-F1 in
the wild); weatherUSA's ~119 mounts are never offered, so "Nearest Relay" can only choose wxradio.org
mounts (HUM LEAD's Linux relay test passed on a wxradio mount). No warning reaches the user — the
directory contributes "nothing rather than an error" by design (`stream/directory.go:69`).

**Options for PLAN.** (a) Fetch the weatherUSA directory over plain HTTP — no security regression since
the audio it points to is plain HTTP already; one constant. (b) A directory-only transport with an
explicit RSA suite list (`httpx.NewTransport` variant) — keeps https but pins a suite Go deprecated.
(c) Both, with (a) as the fallback. Either way: surface a `radio_unavailable`-class warning when a
directory fails so this class of failure is visible in [S]. Ships as a point release regardless of the
pass (in-pass fix, OQ-6).

## LR-2 — Goroutine census: 258 of 279 goroutines are scheduler tier waiters

`goroutine?debug=1` 45 s after launch: 258 parked in `platform/sched.(*Scheduler).wait`
(`sched.go:188`) — one goroutine per tier per scheduler. The priority pipeline has 7 tiers; the RECENT
pipeline runs **one scheduler per location** (`app/dashboard.go newFor`) × its tiers ≈ 250. Bounded by
locations × tiers (~2 MB of stacks) — not a growth term — but the biggest structural lever for memory,
chattiness (L2) and cache churn (L4-F6), and the reason the launch burst is ~4× the code comment (L2-F9).
The remainder is as designed: two httpx cache writers, oto's two audio goroutines, the synth source
pipe, bubbletea's renderer/reader/tick, the signal loop. `threadcreate` total: 25.

## LR-3 — Thread inventory of the 9-hour process (PID 67943)

`sample` (non-destructive): 31 threads = 23 Go runtime threads created at launch + 5 Apple audio threads
created on first tune-in (`caulk.messenger.shared`, `caulk::deferred_logger`, `AQConverterThread`,
`com.apple.audio.IOThread.client`) + 2–3 macOS libdispatch **workqueue** threads that appear and retire
(`start_wqthread → __workq_kernreturn`, one serving the `AQClient` dispatch queue). An hour later: 30.
Reading: two bounded ratchets — Go never reaps idle OS threads (the count follows the peak of concurrent
blocking syscalls: `say` subprocess waits, DNS, disk-cache file I/O — L4-F2's 45k writes/day is the
biggest syscall source) and Apple's GCD pool breathes by ones. Not a leak on this evidence; the
multi-day attributed run (M1, OQ-8) is what proves it.

## LR-4 — Footprint

PID 67943: `ps` RSS 249–254 MB; `vmmap` physical footprint 175 MB → 116 MB across an hour (peak 290 MB);
malloc zones ≈ 10 MB, so the bulk is Go heap. L4-F15: the bounded caches cannot account for the swing;
it is GC/scavenger behaviour over the per-frame (0.71 ms / 446 KB at 50 ms with the visualizer on) and
per-publish (0.44 ms / 1.76 MB) garbage. The instrumented run's heap profiles attribute it.

## LR-5 — Instrumentation gaps found while measuring

- The shipped binary's pprof hook (`WATCHPOST_DEBUG_PPROF=1`) is opt-in at launch only; a running
  process cannot be attributed after the fact (the 9-hour process is opaque). A signal- or file-triggered
  dump (OQ-2 approved) closes this.
- No request counter exists (L2-F16); the proxy counts *connections* (TLS/keep-alive), not requests.
- The pty harness must wait for real data before pressing play and confirm PLAYING — the first
  "quiet" attempt had silently failed to tune (`♪ … FAILED`).

## LR-6 — Instrumented run, 30-minute heap attribution (addendum)

`heap-1828` (30 min, Synth on Repeat: Watchlist, volume 5): **in-use 34 MB** — `os.readFileContents`
16.7 MB (the two httpx memory tiers, NWS + CO-OPS, populated from disk-promoted bodies: the 8 MB budget
is really 16 MB across the two clients, L4-F14), `io.ReadAll` 4.8 MB (in-flight bodies), `geodata.Load`
5.9 MB (one copy retained), the rest < 1 MB each. Cumulative allocation since start 1.16 GB, of which
**853 MB (73 %) under `bubbletea.(*Program).render`** — the render churn L1/L5 measured, live.
Footprint 98–101 MB, threads 27–31, goroutines 281, flat across the first 30 minutes.
