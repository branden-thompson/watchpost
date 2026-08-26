# DISCOVER Lens L1 — Memory allocation & management

Read-only research lens, 2026-08-26. Method: code read on the brief's paths; overlay benchmarks on
realistic 10-fav/50-RECENT fixtures (repo untouched); cross-checked against the instrumented process
(PID 73160, 13.5 min up, radio used, pprof on): 283 goroutines, 22 threads, HeapAlloc 42 MB / HeapSys
94 MB / RSS 126 MB, TotalAlloc 1.7 GB (≈ 2.1 MB/s churn), 59 GCs.

## Findings

| ID | Sev | Title | Where | Evidence | Recommendation |
|---|---|---|---|---|---|
| L1-F1 | **CHURN (dominant)** | Every tea message re-renders the full frame; the 300 ms shimmer tick never stops | `tty/dashboard.go:287-289, 331-333` | `View()` 133×44 colour = **470 KB / 10.5k allocs / 0.78 ms per frame**; 200×60 = 977 KB; details modal 1.45 MB. Idle tick 3.3 fps → **≈ 94 MB/min**; viz on (50 ms) → ≈ 620 MB/min. Live pprof: render path ≈ 80 % of all bytes (`recentSection` 766 MB cum = 45 %, `LocationTable` 585 MB = 34 %) | Stop `tick()` when nothing animates (no `Loading` row, radio idle), or memoize the rendered body keyed on (snap ptr, recent ptr, width, height, frame-dependent fields). Not a leak; the #1 GC-pressure source and the cheapest big win |
| L1-F2 | CHURN | `recentSection` re-renders the radio panel + control row 3–5× per frame to measure heights | `dashboard.go:2193-2204, 2550, 2563-2575`; `compact()` called at 2489, 2543, 2552, 2582 | `RecentSectionOnly` 245 KB / 6.4k allocs; `RadioPanelOnly` 28 KB / 506 allocs each time | Compute `compact()`, `radioHeight`, `alertHeight` once per `View()`; ~25 % of frame bytes |
| L1-F3 | CHURN | Width measurement strips ANSI with a regexp on every call | `render.go:1156-1180`; go-studs `rendering/text_utils.go:18,33` | Live: `regexp.ReplaceAllString` = **273 MB / 15 %** of all allocation | Zero-alloc ESC scanner (or `ansi.StringWidth` from `charmbracelet/x/ansi`, already a transitive dep). go-studs seam → A6 |
| L1-F4 | CHURN | Table layout rebuilt per frame | `render.go:193, 306, 339` | `layout.columns` 83 MB cum (4.6 %); pure over (width, days, dates) | Single-entry memo of `layoutFor(width, days)` |
| L1-F5 | BOUNDED-OK | `Snapshot()` deep copy per publish — **refuted as the suspect** | `assembler.go:253-309`; publisher `app/dashboard.go:140-153` | 10 fav = 385 KB / 0.18 ms; 60 = 2.15 MB / 0.7 ms; cadence ≈ 4/min priority + 2–5/min RECENT ≈ 1.5–5 MB/min — < 5 % of render churn; not in the live top-25 | Leave (optionally skip `Hourly` for RECENT consumers) |
| L1-F6 | OK | Other `Snapshot()` callers: radio per cycle, report once; `FireFor` is the narrow read | `app/radio.go:350-356`, `app/app.go:80` | not per frame/key | Leave |
| L1-F7 | BOUNDED-OK | HMS memo retains ~6.3 MB, **4.2 MB pinned by substrings** (`Satellite`/`Method` are `TrimSpace` substrings of each description) | `hms.go:126-136, 256-287` | 27.5k-placemark synthetic: 6.3 MB (228 B/pt) → **2.1 MB (77 B/pt)** with interning; bound ≤ 46 MB at `maxPlacemarks` | `strings.Clone`/intern the two fields (≤ 6 distinct values) |
| L1-F8 | CHURN (transient) | HMS parse: 75 MB / 1.05 M allocs / 100 ms per archive change | `hms.go:151-195` (`io.ReadAll` of the 8.5 MB KML), `:256-262` (a map per placemark) | **~85 MB heap spike** per 10-min content change — a plausible contributor to the 290 MB peak footprint | Stream the zip entry into `xml.NewDecoder`; parse descriptions with `strings.Cut` loops |
| L1-F9 | CHURN (minor) | `GetText` copies the cached body on every call | `httpx.go:274-275` | HMS 1.4 MB copied + hashed per Fetch, ≈ 3.5×/min ≈ 5 MB/min | Return the cache's immutable slice, or memo on (ptr,len) |
| L1-F10 | **GROWTH-TERM (disk, slow)** | Disk cache tier has no sweeper | `cache.go:162-198` | ~36 h: 1,378 files / 116 MB, 770 expired (81 MB); date/ID-keyed URLs (CO-OPS `begin_date`, NWS `/products/{id}`, alert zone sets) keep adding | Sweep expired files at start-up and hourly on the writer goroutine (= L4-F1) |
| L1-F11 | GROWTH-TERM (theoretical, tiny) | Negative-cache map never evicts unrequested keys | `cache.go:219-237` | ~150 B/entry; grows only when a date-keyed URL 4xxs | Sweep in `evictLocked` or cap at 1024 |
| L1-F12 | BOUNDED-OK | httpx memory tier | `cache.go:55-61,116-160` | 8 MB / 4096 / 2 MB, enforced on every `remember`; live `readFileContents` 13.7 MB in use — confirm on the soak | Leave |
| L1-F13 | BOUNDED-OK (comment stale) | Synth PCM cache | `synth/source.go:301-324,333-335` | `maxCached = 40` but the comment says "24 … ~16 MB"; measured 14.6 KB/word mono, ≤ 280 chars/segment ≈ **0.73 MB/segment → ≤ 29 MB** resident; per play `monoToStereo` + `silence` allocate a 1.5 MB transient (61 MB cum in 13 min) | Fix the comment; optionally 24–30; write stereo in 100 ms chunks |
| L1-F14 | BOUNDED-OK | Assembler warnings trimmed to 256 at `Snapshot()`, grow only between publishes (≤ 20 s) | `assembler.go:39,295-298` | ≤ a few hundred | Leave (or trim in `Warn`) |
| L1-F15 | GROWTH-TERM (human-bounded) | NWS `points` cache keyed by every location ever resolved, never evicted | `nws/provider.go:52,172-196` | ~1 KB/entry; months ≈ hundreds ≈ < 1 MB | Evict on `SetLocations` removal (= L4-F7) |
| L1-F16 | BOUNDED-OK | RECENT / alert store / fire parts | `assembler.go:23-27,141-170` | `SetLocations` deletes removed keys; hotspots ≤ 300; RECENT ≤ 50 | Leave |
| L1-F17 | BOUNDED-OK | **258 parked schedulers** by design | `sched.go:98-106,187-196`; `app/dashboard.go:706-726` | 7 priority tiers + 50×5 RECENT + 1 alerts — exactly the wiring; ~2 KB stack each ≈ 0.5–1 MB; `time.After` in `wait` GC-safe on Go ≥ 1.23; every other `go` cancelled/joined | Leave, or one scheduler with 50 tickers (−250 goroutines) — a UX choice about per-row publish timing |
| L1-F18 | INFO | OS threads: no growth term in code; count = peak of concurrent blocking syscalls | `synth/voice.go:98-121` (`say` exec + temp files, ≤ 2 concurrent), `cache.go:60,92-94` (≤ 4 disk reads + 1 writer), pure-Go DNS, oto (context never closed — 5 Apple threads) | 22 threads at 13 min; nothing sets `GOMAXPROCS`/`SetMaxThreads` | Attribute on the soak via `threadcreate` |
| L1-F19 | INFO | Relay: a new `http.Transport` per `Open` | `player/icy.go:55,82-85` | cancelled via ctx; not a leak | Share one relay transport (= L2-F12) |
| L1-F20 | BOUNDED-OK | Spectrum / tap / preroll / viz buffers | `tap.go:17-29`, `spectrum.go:44-51`, `engine.go:81`, `app/radio.go:82` | fixed; `Bands` allocates 80 B per 50 ms frame | Leave |
| L1-F21 | BOUNDED-OK | Geodata index loaded twice at start (second copy garbage); theme registry filled once | `geodata/index.go:74-108`; `app/dashboard.go:469, 571` | live in-use 5.9 MB (one retained) | `sync.Once` / load once (= L4-F5) |
| L1-F22 | CHURN (minor) | `snapshot.Key` is `fmt.Sprintf`, per row per frame | `types.go:301-303`; `dashboard.go:2423` | 136 ns / 50 B / 3 allocs; ~200 calls/s | Compare Lat/Lon directly in `row()` |
| L1-F23 | INFO | `sortAlerts` per published snapshot | `dashboard.go:355,317` | 35 KB / 455 allocs per snapshot, ~7/min | Leave |
| L1-F24 | INFO | Publisher can hold a spare snapshot while `p.Send` blocks | `app/dashboard.go:147-152` | ≤ a couple of 2 MB snapshots in flight | Leave |
| L1-F25 | INFO | Large transient reads are capped (32 MB bodies, 96 MB inflate, LimitReader) | `httpx.go:121,431`, `hms.go:45-48`, `install.go:282` | JSON decodes run on the cached slice without copy | Leave |

## Verdict

**No unbounded memory growth term exists in the code as written.** Every long-lived structure has an
enforced bound, every goroutine is either the fixed 258-scheduler set or cancelled/joined, and the OS
thread count is a ratchet to peak concurrent syscalls, not a slope; the live process confirms it (42 MB
HeapAlloc at 13 min with the 283 goroutines the wiring predicts). True growth terms: the **disk** cache
tier (no sweeper — L1-F10 / L4-F1) and two human-bounded trickles (`points` map, negative cache) at
sub-MB per year. What "slowly rising" most plausibly reflects is **churn**: rendering allocates ~80 % of
all bytes (≈ 94 MB/min idle, ≈ 620 MB/min with the visualizer) and the HMS parse spikes ~85 MB every
10 minutes — together they set the GC's heap ceiling and the 116 → 175 MB footprint swings. The soak
should show a *plateau with sawtooth*; the highest-value fixes are render-path (stop the idle tick,
memoize heights and layout, drop the regexp width) — none touch the data model, the radio, or the schema.
