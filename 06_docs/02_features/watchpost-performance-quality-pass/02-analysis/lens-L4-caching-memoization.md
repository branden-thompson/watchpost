# DISCOVER Lens L4 — Caching & memoization inventory

Read-only research lens, 2026-08-26. Method: every cache/memo site read; provider headers probed live;
the live disk cache inspected at `~/Library/Caches/watchpost/http`; scratch benchmarks via `go test
-overlay` (repo untouched). Numbers from those runs (`-benchmem`).

## Inventory

| # | Cache / memo | Where | Keyed by | Bound | Eviction | Evidence | Verdict |
|---|---|---|---|---|---|---|---|
| 1 | httpx memory tier | `httpx/cache.go:23-33,55-61,79-160` | full URL | 8 MB / 4096 entries / entry ≤ 2 MB; TTL = caller `TTL()` else `max-age`/`Expires` | expired first, then LRU (O(n) scan); `forget()` | hit 39 ns / 0 alloc; **miss with expired mem+disk 157 µs / 42 KB** | KEEP / TIGHTEN miss path |
| 2 | httpx disk tier | `cache.go:25,63-75,102-114,163-184` | SHA-256(URL) → `<dir>/<hex>.cache` 0600 | **none: no size/count cap, no sweep** | overwrite per URL; `forget()` only | live: **1,376 files / 116 MB after ~2 days; 794 expired (80 MB); 593 orphan `.json` files (76 MB, 0644) from the pre-UAT-73 format**; writer 473 µs/entry (5 syscalls) | **UNBOUNDED** |
| 3 | httpx negative cache | `cache.go:44-47,219-237` | URL → err, 30 s | distinct 4xx URLs | lazy | fine | KEEP |
| 4 | httpx singleflight | `httpx.go:64,326-344` | URL | in-flight | n/a | correct | KEEP |
| 5 | Cache-header honouring | `cache.go:247-269`, `httpx.go:408-414` | — | — | **no ETag / Last-Modified stored, no `If-None-Match` sent** | NWS sends ETag on obs (`max-age=52`), alerts (`max-age=5`), forecast/hourly/gridpoint (`max-age=3600`); HMS/NDBC ETag + Last-Modified | **MISSING** |
| 6 | `Client.CacheStats` | `httpx.go:470` | — | — | — | **no callers** | REMOVE or wire into [S] |
| 7 | HMS parsed-archive memo | `hms.go:59-68,126-136` | SHA-256(body) → points | one archive | replaced on change | parse 7.3 ms today (1,766 pts); 60-location scan 3 ms, 0 alloc | KEEP |
| 8 | WFIGS layer | `wfigs.go:72-117` | httpx URL only | — | — | **re-decodes 208 KB JSON per Fetch**: 0.98 ms / 278 KB / 4k allocs × ~200 Fetches/h ≈ 57 MB/h garbage | MISSING (memo like HMS) |
| 9 | FIRMS | `firms.go:130-168` | URL per location×source | — | `Forget` on bad CSV | 124 disk files here | KEEP |
| 10 | Synth PCM cache | `synth/source.go:36-37,301-324,335` | content `Segment.Key` → mono PCM | 40, FIFO | cleared on `SetVoice` | ≤ ~16 MB | KEEP |
| 11 | Products feed | `synth/products.go:74,89` | list 10 min, body 24 h | — | — | each product ID = a new disk file (~15/day/office) | KEEP (see F1) |
| 12 | Piper install | `synth/install.go` | files | catalogue-bounded | none | absent on macOS | KEEP |
| 13 | `say -v ?` list | `app/radio.go:83,98-103` | once, background | 11 | never | correct | KEEP |
| 14 | `systemVoiceName` | `synth/voice.go:73-88` | `sync.Once` | 1 | never | C-5 fix | KEEP |
| 15 | Relay directory | `stream/directory.go:25,84` | httpx TTL 5 min (server `no-store`) | — | — | ToS floor honoured | KEEP (but see live finding LR-1) |
| 16 | Transmitter table | `stream/table.go:48-77` | parsed once | ~1k rows | immutable | `Nearest` sorts per Tune — fine | KEEP |
| 18 | Geodata index | `geodata/index.go:74-108` | process | ~13 MB | immutable | **Load = 36.5 ms / 19.3 MB / 500k allocs, called TWICE at launch** (`app/dashboard.go:469`, `:571`) | TIGHTEN |
| 22 | tz memo | `platform/tz/tz.go:15-35` | zone name | few hundred | never | fixes the 140-thread launch | KEEP |
| 23 | astro sun times | `assembler.go:537-555` | per Daily row per Snapshot | — | — | inside 0.44 ms / 1.76 MB per 60-location Snapshot | OK |
| 24 | Theme registry / `Tok()` | `render/theme.go:162`, `themes.go` | token → SGR (RWMutex) | 5 + user | `SetTheme` | **17.7 ns, 0 alloc** | KEEP |
| 25 | `TitleGradient` | `render.go:254-284` | per frame | — | — | 2.2 µs / 786 B | OK |
| 26 | `Width()` | `render.go:1146,1156,1180` | regex strip + runewidth per call | — | — | 666 ns / 4 allocs, hundreds per frame | INFO (render lens) |
| 27 | NWS `gridInfo` | `nws/provider.go:51-53,172-196` | LocationKey → grid/stations/zones | **grows with every location ever seen; never evicted, never refreshed** (doc promises daily) | — | ~1 KB/entry — a true growth term against OQ-8 | TIGHTEN |
| 28 | NWS-marine inland memo | `nws/marine.go:22,28,36-47` | grid URL, 24 h | distinct grids | re-marked | saves 228 KB/grid/cycle | KEEP |
| 29 | CO-OPS / NDBC station lists | `coops.go:235-260`, `ndbc.go:188-215` | once / 24 h | 4 lists | daily | > 2 MB → disk-only, parsed once | KEEP |
| 30 | Snapshot publish | `app/dashboard.go:129-153`, `assembler.go:253-309` | — | — | — | 0.44 ms / 1.76 MB per publish; ≤ 200 publishes/h ≈ 350 MB/h garbage, no retention | OK (memory lens) |
| 31 | TTY per-frame work | `dashboard.go:1049-1098,2276-2296,2336-2371` | none — tables, header, radio, alerts rebuilt every frame | — | — | idle frame (10+10 rows) **0.71 ms / 446 KB / 10k allocs**; modal open 1.6 ms / 1.35 MB; 50 ms viz tick → ~9 MB/s garbage | TIGHTEN (low) |
| 34 | Config Load/Save | `config.go:146,175` | per user action | — | — | atomic | KEEP |
| 35 | DNS / connections | `httpx.go:84-97` | Go resolver (no cache), idle 90 s | — | — | 10-min-cadence hosts re-resolve + re-handshake each tick | INFO |

## Findings

| ID | Sev | Finding | Recommendation | Risk |
|---|---|---|---|---|
| L4-F1 | **UNBOUNDED** | The disk tier is never pruned and carries a dead-format graveyard: 1,376 files / 116 MB after two days; 593 `.json` orphans (76 MB, 0644) from the pre-UAT-73 base64 format the S-F10 hardening never touched. New files forever: CO-OPS URLs carry `begin_date` (~46 new files/day for a coastal watchlist); `/alerts/active?zone=…` keyed by the whole zone set (a new 32 KB file per RECENT commit, 20 variants already); synth product IDs (~15/day/office) | Sweep at `newCache` and daily on the writer goroutine: delete expired `.cache` and any non-`.cache` file; cap the directory (e.g. 256 MB) by oldest mtime | low (one function, `TempDir`-testable) |
| L4-F2 | TIGHTEN | ~95 % of disk writes persist entries that can never serve a relaunch: alerts (TTL 5 s) 4,320 writes/day; obs ~16,800; FIRMS ~12,800; fire/marine/forecast ~10k → **≈45k writes/day ≈ 0.5–0.7 GB/day** through CreateTemp+write+chmod+rename (≈21 s/day of blocking file I/O — the syscall class the brief names as a thread source) | Persist only when TTL ≥ ~5 min (one `if` in `put`) | very low |
| L4-F3 | TIGHTEN | The miss path reads an expired file it already knows is expired (157 µs / 42 KB per miss; ~21k wasted reads/day) | Skip the disk read when the memory tier holds a stale entry for the URL | very low |
| L4-F4 | **MISSING** | No conditional GETs although every NWS product carries an ETag: ~180 alert polls/h × 32 KB and ~700 obs polls/h × 4 KB are full downloads of mostly unchanged bodies; each hourly expiry re-downloads 162 KB/location | Store ETag/Last-Modified in the entry header; send validators on expiry; on 304 renew `Expires` in place. Cuts NWS bytes 60–80 % at unchanged request counts | medium (httpx core; well covered by tests) |
| L4-F5 | TIGHTEN | Geodata index loaded twice per launch (36.5 ms + 19.3 MB each) | Load once in `RunDashboard`, pass to both | nil |
| L4-F6 | MISSING | WFIGS re-decodes the 208 KB layer per per-location Fetch (~57 MB/h garbage) | Body-hash memo as HMS has — a shared `fire` helper | nil |
| L4-F7 | TIGHTEN | `gridInfo` never expires or refreshes; keys never dropped on `SetLocations` | `resolvedAt` + re-resolve after 24 h; drop keys on removal | low |
| L4-F8 | INFO | `CacheStats` is dead code | Wire Entries/Bytes + a request counter into [S], or remove | — |
| L4-F9 | TIGHTEN (low) | Frames rebuild pure content; the 50 ms viz tick costs 0.71 ms / 446 KB per frame (~9 MB/s garbage) — churn, not growth | Memoize rendered table strings keyed by (snapshot ptr, selection, width, days, radio key) — only if the render lens wants the CPU | moderate |
| L4-F10–12 | OK | `Tok()` 17.7 ns; HMS memo sufficient (a spatial index buys nothing); semantic caches bounded and threaded correctly | — | — |
| L4-F13 | INFO | Idle 90 s vs 10-min cadences: CO-OPS/NDBC/FIRMS/HMS/WFIGS pay DNS + TLS every tick | ~11 min idle timeout would keep a handful of sockets warm; connectivity lens decides | — |
| L4-F14 | INFO | Two clients (NWS, CO-OPS) + one per `report` run: separate 8 MB memory tiers and writer goroutines over one cache dir — the memory budget is really 16 MB | Document or unify | — |
| L4-F15 | INFO | Nothing here explains the 175 → 116 MB footprint swing: bounded caches sum to ~8+8+13(+13 transient)+≤16+3 MB; the swing is GC/scavenger behaviour over the per-frame and per-publish garbage | — | — |

## Verdict

The policy is right and well documented (`03-architecture-design/caching.md`): one URL-keyed layer with
stated lifetimes, singleflight, a negative cache, a short list of semantic caches. Its weaknesses are the
two seams the design called "best effort" and never bounded: **the disk tier** (no persistence floor, no
sweep, orphaned format, date-keyed URLs — the only truly unbounded structure in the app) and
**revalidation** (no ETag use). Best value/risk: (1) disk-tier hygiene (F1–F3, one file); (2) conditional
GETs (F4); (3) geodata once + WFIGS memo (F5, F6). `gridInfo` expiry (F7) and `CacheStats` wiring (F8) ride
along; F9 waits for the render lens's numbers.
