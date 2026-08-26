# DISCOVER Lens L2 — API connectivity & chattiness

Read-only research lens, 2026-08-26, over `feature/watchpost-performance-quality-pass` (= v0.9.4 tree).
Numbers are derived from code; "measure" marks where a live header or the instrumented run decides.
Assumptions: 10 favourites + 50 RECENT; ~20 of 60 locations coastal; FIRMS keyed.

## 1. Request-site inventory

| Site | Where | Lifetime rule | Triggered by |
|---|---|---|---|
| NWS `/points/{lat},{lon}` + `{stations}?limit=4` | `nws/provider.go:212,243` | server max-age (~1 day) on disk; **in-process `gridInfo` memo never expires** (`:172-196`, singleflight per key) | first Fetch of any kind per location; shared by all tiers |
| NWS `/stations/{id}/observations/latest` | `:364` | server max-age (obs 52 s observed / s-maxage 300 — AI-1 §37) | Obs tier (fav 90 s, RECENT 10 m); walks up to 4 stations (`:325-338`); also synth `segments()` |
| NWS `/gridpoints/…/forecast` | `:453` | server (~next issuance) | Forecast tier (30 m / 1 h); `report` |
| NWS `/gridpoints/…/forecast/hourly` (162 KB) | `:469` | server (~1 h) | Hourly tier fav 30 m; RECENT on demand (`hydrateCmd`, `tty/dashboard.go:502`) + once per lookup (`app/dashboard.go:645`) |
| NWS raw gridpoint (228 KB) | `:502`, `marine.go:127` | server; inland memo 24 h (`marine.go:28`) | Marine tier (coastal) **and** `fillDailyFromGrid` on any temp hole (every evening, all locations) |
| NWS `/alerts/active?status=actual&zone=…` | `:704` | server max-age 5 s | fav batched 20 s; RECENT batched 2 m (`app/dashboard.go:621`); synth per cycle (single-location URL) |
| NWS `/products/types/{T}/locations/{WFO}` ×4, `/products/{id}` | `synth/products.go:74,89` | TTL 10 m / 24 h | each synth cycle |
| NDBC `activestations.xml` (270 KB), `5day2/{ID}_5day.txt` | `ndbc.go:194,245` | TTL 24 h / 10 m | MarineObs tier (10 m); chain up to 4 buoys + 8 temp stations |
| CO-OPS 3× `stations.json` (6.6 MB, disk-only), `predictions`, `currents_predictions`, `water_level` | `coops.go:249,328,378,350` | TTL 24 h / **1 h** / 1 h / 10 m — own client, 5 req/s | Marine tier (30 m / 1 h), MarineObs (10 m) |
| HMS KMZ (1.4 MB) | `hms.go:95` | TTL 10 m; parse memo by hash | Fire tier fav 10 m, RECENT 15 m × 50 schedulers |
| WFIGS GeoJSON | `wfigs.go:87` | TTL 10 m | same |
| FIRMS `/api/area/csv/{key}/{src}/{bbox}/1` ×2 | `firms.go:135` | TTL 10 m; bbox unique per location | same, serial per location |
| Open-Meteo geocode | `geocode.go:54` | server headers | offline-index miss only |
| Relay directories (2 status-json.xsl) | `stream/directory.go:84` | TTL 5 m | every `Tune` / `SetMode` / Watchlist advance |
| ICY stream | `player/icy.go:55` | none; fresh `http.Transport` per Open | one per listener |
| Piper binary + voice (~70 MB) | `synth/install.go:250` | one-off; own `http.Client` | first synth on Linux/Windows |

Three `httpx.Client`s exist (dashboard at 30/s, CO-OPS at 5/s, `report` at default 5/s), each with its own
transport, memory tier and disk-writer goroutine; they share the disk directory.

## 2. Steady-state request budget (requests/hour, network, cache misses at TTL)

| Provider / product | Favourites (10) | RECENT (50) | Total | Cadence minimum | Note |
|---|---|---|---|---|---|
| NWS alerts (batched) | 180 | 30 | **210** | 210 (180 if one call covered all 60) | max-age 5 s: no cache help |
| NWS obs | 120–400 | 300 | **420–700** | same | fav depends on whether NWS returns the CDN's remaining lifetime (→120) or a flat ~52 s (→400). **Measure.** ×(1–4) on incomplete stations |
| NWS daily forecast | 20 | 50 | **70** | 70 | |
| NWS gridpoint (228 KB) | ~7 marine + ≤20 fill (evenings) | ~17 + ≤50 fill | **24 → ~94 evenings** | 24 | fill not deduped against inland memo; ~16 MB/h in the evening window |
| NWS hourly (162 KB) | 10–20 | ~0 | **10–20** | 10–20 | |
| NWS points/stations | 0 | 0 | 0 | 0 | memo for process lifetime |
| NWS products (synth playing) | 24 lists + ≤4 texts + 6–12 obs + 6–12 alerts | — | **~40–50 while playing** | ~24 | obs/alerts duplicate the pipelines |
| **NWS total** | ~350–650 | ~400 | **~750–1,050 (≈13–17/min)** | ~700 | matches the ledger's "~15 req/min" |
| NDBC buoy files | ≤4 × 1–3 × 6 | ≤17 × 1–3 × 6 | **~60–150** | same | |
| CO-OPS tide + current predictions | ~4 × 1.7 | ~17 × 1.7 | **~35** | **~1.5 (once per UTC day)** | astronomical data refetched 24×/day |
| CO-OPS water level | ≤4 × 6 | ≤17 × 6 | **≤126** | same | gauges report every 6 min |
| HMS / WFIGS | 6 / 6 | ~0 | **6 / 6** | 6 | HMS 8.4 MB/h |
| FIRMS (keyed) | 120 | 400 | **520** | **~12–30** with merged boxes | 60 distinct bboxes for one continental product |
| Relay directories | | | 0–24 | | within ToS |
| **All providers** | | | **~1,500–1,900/h (≈25–32/min)** | **~1,000/h** | |

## 3. Findings

| ID | Sev | Finding | Recommendation |
|---|---|---|---|
| L2-F1 | **RISK** | Retry-on-retry: httpx retries 4 attempts (`httpx.go:170-175,375-402,446`) × scheduler re-cycles 10/20/40 s (`sched.go:113-141`) × `fetchObs` walks all 4 stations on any error (`provider.go:325-338`). Estimated **~23,000 attempts/h during an NWS 5xx/transport outage** vs ~1,000 healthy. Negative caching covers 4xx only; no `Retry-After` on 429 | One retry layer (httpx `MaxRetries` 0–1 where a scheduler rehydrates, or skip rehydration for failures httpx exhausted); walk the station chain only on 404/incomplete data; per-host failure memo 30–60 s; honour `Retry-After` |
| L2-F2 | WASTE | FIRMS: 120 area requests/10 min for one continental product (`firms.go:130-137`); RECENT makes it 400/h | Merge overlapping/adjacent boxes into a few regional boxes at the same cadence; filter with `fire.Near` exactly as today → ~12–30/h |
| L2-F3 | WASTE | CO-OPS predictions refetched hourly for astronomical data (`coops.go:43`, URL carries `begin_date=today`) | TTL "until next UTC midnight" — pure TTL change; ~700 requests/day saved on the provider that 403'd list repeats |
| L2-F4 | WASTE | Disk tier churns on 5 s / 52 s lifetimes (`cache.go:103-114,79-100`): every alerts/obs body written to disk and read back — ~2,000 blocking syscalls/h that never warm a relaunch | Skip the disk tier below a lifetime threshold (~60 s); skip the disk read when the memory tier holds the same URL expired |
| L2-F5 | OK / measure | Obs tier 90 s vs NWS CDN `max-age=52, s-maxage=300` — if the header returns the CDN's remaining lifetime the cache already coalesces; if flat 52 s the tier fetches the same object up to 5× per 300 s for no freshness | **Measure the header on the instrumented run** before deciding; the explicit freshness question for HUM LEAD |
| L2-F6 | WASTE (bytes) | Evening gridpoint fill: every location fetches 228 KB per forecast tick after local 18:00 (`provider.go:492-504`) — ~16 MB/h for two numbers | Memoize `{gridURL → today's max/min}` until the next issuance, or fill only for TODAY |
| L2-F7 | WASTE (minor) | Synth re-fetches obs + alerts the pipelines hold (`app/radio.go:349-355`) — 6–12 calls/h while playing | Narrow assembler read (the `fireFor` pattern) |
| L2-F8 | INFO | `report` one-shot: serial fan-out, ~20–30 requests, ~9 MB cold, NWS at 5/s → 4–10 s cold | `errgroup` over the 7 kinds; one `newClient()` builder as the single owner of client policy |
| L2-F9 | INFO | Launch burst is ~4× the "~75 calls" comment (`app/dashboard.go:54`): cold ≈ 70 priority + 220–320 RECENT (+ FIRMS) | Update the comment; otherwise cadence-minimal |
| L2-F10 | INFO | Priority-lane guarantee rests on HTTP/2 (one `Transport`, `MaxConnsPerHost: 8`) | Verify `resp.ProtoMajor` in the run; separate transport if h1.1 |
| L2-F11 | INFO | Connection hygiene sound (pure-Go resolver, keep-alive 30 s, idle 90 s, h2); TLS resumption unverified | Set `ClientSessionCache` if the run shows full handshakes |
| L2-F12 | INFO | ICY (`icy.go:82-85`) and Piper install (`install.go:250`) build their own transports — outside the pure-Go-resolver rule | Export `httpx.NewTransport()`; one owner |
| L2-F13 | INFO | `/points` memo never refreshes — with weeks of uptime a moved grid persists until relaunch | Daily refresh (queued in B1) |
| L2-F14 | INFO | RECENT alerts URL can carry 100 zones (~800 chars) | Document or shard at 80 |
| L2-F15 | INFO | Two alert schedulers could be one (−30/h; RECENT gets 20 s freshness) | Optional |
| L2-F16 | INFO | **No request counter exists** — M2 needs per-host/lane atomic counters (attempts, network vs cache vs negative) in `do()`, surfaced in [S] and a debug dump | Prerequisite for confirming this table |
| L2-F17 | OK | Already right: singleflight + URL cache across pipelines; HMS parse memo; inland memo; batched alerts; per-lane pacing; in-flight/per-host caps; caller TTLs; negative caching; `Update` fetches newcomers only; hourly on demand |

## 4. Verdict

At steady state with healthy providers the app is **not chatty** (~1,000/h NWS, ~1,500–1,900/h overall vs
a ~1,000/h floor). Avoidable volume in order: outage amplification (F1, ~20×), FIRMS per-location boxes
(F2), disk-tier churn (F4), CO-OPS hourly astronomical refetch (F3), evening gridpoint bytes (F6), synth
duplicates (F7). The one cadence question for HUM LEAD is F5, to be settled by measurement; F16 (a counter)
lands first so the instrumented run can confirm the table.
