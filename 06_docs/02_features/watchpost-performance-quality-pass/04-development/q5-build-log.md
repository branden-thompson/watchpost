# Q5 build log — Network, bytes and fan-out (`v0.9.8`)

**Batch:** Q5 of the plan of record v3 (§2.2 conditional GETs, §2.6 fire fan-out, §3 Q5a/Q5b).
**Approval:** "Approved; go 4 0.9.7; Go 4 Q5" (2026-08-26).
**Branch:** `feature/watchpost-performance-quality-pass`, commit `8b66cda` (+ docs, this log).
**Status:** APPROVED 2026-08-26 ("Approved; GO for 0.9.8; GO 4 Q6") — the §4 freshness rows
initialled by HUM LEAD with the approval; `v0.9.8` cut from this tree; Q6 next. Follow-ups F-1/F-2 in
`follow-ups.md` (after the pass).

## 1. What changed, and why (junior-first)

Q5 is about asking for less over the network and asking once for what many locations share.

- **Conditional GETs.** Q1 stored `Last-Modified` / `ETag` with every cache entry; Q5 sends them.
  When an entry has expired, the miss goes out as `If-Modified-Since` (NWS honours it; its own
  ETags are mangled by its CDN) plus `If-None-Match` for hosts that honour that. A `304` renews the
  entry — in memory, and back from disk if the memory budget had evicted it — and moves the file's
  mtime on the writer goroutine; no body is downloaded. The counters record `NotModified` and
  `Bytes304`. A `304` to an *unconditional* request is a server fault: one bounded refetch, never a
  loop.
- **FIRMS by tile.** Every RECENT location's scheduler called `Fetch` on its own, so "merge the
  boxes" had nothing to merge. Now a request covers a fixed 5° tile of the globe: the tile URL is the
  cache key and the singleflight key, so every location inside a tile shares one request per
  satellite source; a location's 25 km box that straddles an edge fetches every tile it touches (at
  most four); membership is still decided by `fire.Near`, so the hotspots a location sees are
  byte-identical. Tiles are parsed once per body change (a memo bounded at 240 tiles); a source whose
  tile body passes 2 MB moves to 2.5° tiles.
- **One transport policy.** The 24 h soak's counters showed a TLS handshake per host per tick
  (FIRMS 178, CO-OPS 140, NDBC 91 in nine hours): the 90 s idle timeout let every connection die
  between 10-minute tiers. `httpx.NewTransport()` (11 min idle, HTTP/2, bounded connections, pure-Go
  resolver) is now the policy for the data clients, the ICY stream reader and the voice-model
  downloader.
- **Lifetimes that match the data.** Tide/current predictions are astronomical and the request
  window is keyed by the UTC date: cached to UTC midnight (one fetch per station per day, was 24).
  A location's NWS grid resolution is redone after a day and dropped when the location leaves the
  lists (`Retain`); the ~1 MB gridpoint that fills TODAY's HIGH/LOW after 6 PM is decoded once per
  body change, not once per location per tier.
- **The one-shot report** fetches its seven kinds in parallel through one client builder
  (`newDataClient`), the single owner of client policy for the dashboard and the report.

## 2. Files touched

| Area | Files |
|---|---|
| `platform/httpx` | `httpx.go` (`conditional`, `getOrRevalidate`, 304 in `attemptOnce`/`doAttempt`, `NewTransport`, 11 min idle), `cache.go` (`stale`, `revalidated`); tests `conditional_q5_test.go` (6 pins), transport pin |
| `domains/fire/firms` | `tiles.go` (new: grid, memo, split), `firms.go` (`hotspotsFor`, `near`, `MemoStats`); `tiles_test.go` |
| `domains/weather/nws` | `points.go` (`resolvedAt`, `gridTTL`, `Retain`), `forecast.go` (`gridExtremes`, `gridDoc`, `gridMemo`, `GridDecodes`), `provider.go`; `grid_q5_test.go` |
| `domains/marine/coops` | `predictionsLifetime` / `untilUTCMidnight`; test |
| `domains/radio` | `player/icy.go`, `synth/install.go` on `httpx.NewTransport()` (no behaviour change to the radio path — R6: smokes in §7) |
| `app` | `app.go` (`newDataClient`, parallel kinds), `dashboard.go` (`Retain` on commit), `stats.go` (gauges `firms.memo.tiles/parses`, `nws.grid.decodes`) |
| docs | `CHANGELOG.md`, `caching.md` (revalidation, tiles, lifetimes, the transport), `docs/where-things-happen.md` (+3 rows, +2 vocabulary), `follow-ups.md` (new) |
| records | `07-readiness/p10-q5.json`, `02-analysis/q5-soak-1h.csv`, `02-analysis/q5-counters/` |

## 3. Tests first (the pins)

| Pin | Guards |
|---|---|
| `TestConditionalGETRenewsOnA304WithIfModifiedSinceFirst` | both validators sent; 304 serves the stored body; `NotModified`/`Bytes304` counted; the renewed entry is a cache hit |
| `TestConditionalGETTakesTheNewBodyOnA200` | a changed body replaces the stored one |
| `TestNoValidatorsMeansAPlainGET` | nothing to revalidate → no conditional headers |
| `TestA304ToAnUnconditionalGETIsRefetchedOnce` | the SC-8 invariant: one refetch, never a loop |
| `TestA304RenewsAnEntryEvictedToDisk` | PF-4: the disk entry's validators revalidate; the file's mtime moves (Chtimes queued); the entry returns to memory |
| `TestNewTransportOutlivesATenMinuteTier` | idle > 10 min, HTTP/2, per-caller copies |
| `TestTilesForCoversTheBoxWithAtMostFour` | inside / meridian / parallel / corner → 1 / 2 / 2 / 4 tiles; a pathological box is bounded |
| `TestLocationsInOneTileShareOneRequestAndSeeTheSameHotspots` | one request per source for two locations; the request is the tile; the same hotspot as the per-box request; one parse per (source, tile); no re-request, no re-parse |
| `TestStraddlingBoxFetchesEveryTouchedTile` | a corner box: 8 requests (4 tiles × 2 sources) |
| `TestTileMemoIsBoundedAndSplitsAnOversizedSource` | ≤ 240 entries; the split pitch per source |
| `TestGridResolutionExpiresAfterADayAndKeepsThePreferredStation`, `TestRetainDropsLocationsNoLongerTracked`, `TestGridExtremesDecodeOncePerBody` | Q5b-6/7 |
| `TestPredictionsAreCachedToUTCMidnight` | 20:30Z → 3h30m; the one-minute floor; a local clock read in UTC |

## 4. Freshness table (§0.3) — every lifetime this batch changed

| Kind | Before | After | Freshness argument | HUM LEAD |
|---|---|---|---|---|
| CO-OPS tide / current predictions | 1 h | to UTC midnight (≥ 1 min) | astronomical; the window is keyed by the UTC date, so the answer cannot change inside the day and the URL itself changes at midnight | BT 2026-08-26 |
| NWS grid resolution (`/points` + `/stations`) | forever (process life) | 24 h, then re-resolved (last-good kept on failure); dropped on removal | grids and station lists change rarely; a day bounds a re-grid's staleness; the client cache answers the request when nothing changed | BT 2026-08-26 |
| Idle connection timeout (all clients) | 90 s | 11 min | not a data lifetime: a session outlives the 10-minute tiers, so the tick reuses it (fewer handshakes, same requests) | BT 2026-08-26 |
| FIRMS request shape | per-location box | 5° tile (2.5° on a > 2 MB body) | same 10-min lifetime; identical hotspots per location; requests shared per tile | BT 2026-08-26 |

The per-provider request floor (M2) is re-derived in §8 against the counters.

## 5. Bounds stated (§0.8)

- Parsed-tile memo: ≤ 240 (60 locations × ≤ 4 tiles), LRU; owner `firms.Provider`; pinned.
- Grid cache: one entry per tracked location (`Retain`); gridpoint memo: one per live grid.
- A tile box walk: ≤ 16 tiles (a 25 km box touches ≤ 4).
- Conditional GET: at most one extra request per miss (the unconditional refetch), never re-entering `fetch`.
- `SGR`/transport: unchanged bounds; the idle timeout is per transport.

## 6. Decisions and non-decisions

1. **Synth's assembler read (Q5b-8) — already narrow.** `radioDeck.segments` builds a one-location
   assembler and `fireFor` reads one location's fire state (`Assembler.FireFor`); verified, no change.
2. **One shared disk cache for the NWS and CO-OPS clients (Q5b-10, L4-F14) — not now.** Two clients
   share one directory with separate 8 MB memory tiers and writer goroutines; the sweep runs twice at
   launch (bounded ≤ 10k listed / ≤ 1k deleted each). Unifying needs a cache handle shared between
   clients (`httpx.Config{Cache}`); the measured cost (two sweeps, ≤ 16 MB memory ceiling) does not
   justify a new seam in a network batch. Backlog for Q6 if its seam work touches `httpx.Config`.
3. **Two alert schedulers as one (L2-F15) — keep.** Merging gives the RECENT list 20 s alert
   freshness at +30 requests/h; the plan chose the 2-minute batch for RECENT on purpose (UAT 72).
4. **C2 (obs cadence)** — NWS serves observations with `max-age=300`; the 90 s tier is coalesced
   server-side (a request every 90 s, a network body every 5 min). No cadence change.
5. **h2 / TLS resumption (L2-F10/F11)** — from the 24 h soak's counters (nine hours in): every
   response over HTTP/2 on every host; full handshakes NWS 4 (one connection lived), FIRMS 178, CO-OPS
   140, NDBC 91, HMS 42, WFIGS 33 — one per tick per host at a 90 s idle timeout. Hence the 11-minute
   policy; the Q5 soak in §8 re-takes the number.
6. **Tile pitch.** 5° from the start (a peak day's CONUS ≈ 50k detections ≈ 5 MB of CSV; a 5° tile is
   ~1/40 of it), with the 2.5° split as the runtime fallback rather than a Q0 count that never
   happened (the plan asked for one; the split rule makes the count unnecessary).
7. **HUM LEAD follow-ups F-1/F-2** (RECENT seed rows feel slow; some CA locations lost Nearest Relay)
   are recorded in `follow-ups.md` with their likely causes and first moves — first thing after the
   pass, by HUM LEAD's direction; not folded in here.

## 7. Gate

| Check | Result |
|---|---|
| `make verify` | ALL GATES GREEN |
| `make p10` | 0 live · 0 unmatched · `07-readiness/p10-q5.json` |
| `a2dh validate` | 18/18 |
| goldens, alloc budgets | unchanged / green |
| declsets re-captured (nws, app — intentional, §2) | green |
| Synth / Relay smokes (R6: the radio's HTTP client moved to the shared transport) | synth PLAYING in 4 s; `LiveRelay` passes on both relays (§8) |
| 1 h soak: counters ≤ the re-derived floor; bytes on the 304-able kinds; FIRMS tiles and bytes per tile; TLS handshakes; C2 | **met** — NWS 652 ≤ ≈ 790, FIRMS 270 ≤ 324; 130 renewals saved 3.0 MB; 54 tile parses; handshakes recorded (one hour cannot resolve the change); C2 `max-age=300` (§8) |

## 8. The 1-hour soak (macOS, idle dashboard at 133×44, `dist/watchpost` at `8b66cda`, 60 s samples)

The hour since launch (the launch burst included, as in Q1's table), from `counters-1h.json`:

| Host | attempts | net (200) | cache | **304** | bytes saved by 304 | bytes net | TLS handshakes | h2 |
|---|---|---|---|---|---|---|---|---|
| api.weather.gov | 652 | 537 | 931 | **102** | **1.54 MB** | 22.2 MB (41 KB avg) | 5 | all |
| firms (tiles) | 270 | 270 | 187 | 0 (no validators) | — | 0.03 MB | 19 | all |
| api.tidesandcurrents | 142 | 106 | 147 | 0 | — | 0.06 MB | 9 | all |
| www.ndbc.noaa.gov | 101 | 73 | 268 | **28** | **1.50 MB** | 3.5 MB | 14 | all |
| arcgis (WFIGS) | 4 | 4 | 147 | 0 | — | 0.87 MB | 4 | all |
| ospo (HMS) | 0 | 0 | 257 | — | — | — | 0 | — |

**Against the plan's gate lines.**

- **Requests ≤ the re-derived floor (M2).** Steady-state ceiling per hour for 10 favourites + 50
  RECENT from the cadence table: NWS alerts 180 + 30 (batched), observations ≤ 420 net (server
  `max-age=300` coalesces the 90 s tier — C2), forecast/hourly/marine ≈ 160 → **≈ 790 attempts**;
  measured **652 attempts / 537 net** (Q1's healthy hour: 691 / 689). FIRMS: 27 distinct tiles × 2
  sources × 6 expiries = **324 ceiling; measured 270** (the 24 h soak, per-location: ≈ 420/h; the
  per-location ceiling was 60 × 2 × 6 = 720). CO-OPS 142 with predictions now to midnight.
- **Bytes on the 304-able kinds.** NWS renewed 102 bodies without downloading them (1.54 MB, 7 % of
  its hour); NDBC 28 (1.5 MB, 30 % of its hour) — NDBC honours both validators. FIRMS and CO-OPS
  send none. Net bytes NWS 22.2 MB vs 24.6 MB in Q1's hour.
- **FIRMS tiles touched and bytes per tile.** 54 (tile, source) pairs parsed once each
  (`firms.memo.tiles` = `firms.memo.parses` = 54); ~0.1 KB per tile body this hour (few detections
  in the tracked tiles); the 2 MB split never armed.
- **TLS handshakes.** NWS 5, FIRMS 19, NDBC 14, CO-OPS 9, WFIGS 4 in the hour, all HTTP/2. The 24 h
  soak's per-hour rates at the 90 s idle timeout were ≈ 20 / 10 / 15 (FIRMS / NDBC / CO-OPS); one hour
  including the launch burst does not resolve the change — Q7's 7-day soak re-takes it.
- **C2 recorded:** NWS observations arrive with `Cache-Control: max-age=300`; no cadence change.
- **Publishes:** RECENT 35/h (Q3 parity), priority ≈ 4/min. **Heap after GC** 36–51 MB sawtooth, no
  trend; RSS 80–89 MB; goroutines 273 flat; disk files 818 flat; disk writes 461 in the hour (Q1: 404
  — the tiles' 10-min entries persist; each is ~0.1 KB).
- **Gauges:** `nws.gridinfo` 58, `nws.grid.decodes` 66 (once per grid per body change — the previous
  shape decoded once per location per tier), `hms.memo.parses` 1, `wfigs.memo.parses` 6.
- **Caveat.** The Q0 24 h soak was still running on the same cache directory: HMS and WFIGS bodies were
  warm from the other process, so this hour's ospo/arcgis rows measure the shared disk tier, not the
  network. The other rows are unaffected (their entries expire inside the hour).

R6 smokes on the Q5 binary (the stream client moved to the shared transport): synth **PLAYING in 4 s**;
`LiveRelay` **passes** (wxradio WNG522 128 kbps, weatherUSA WXK63 32 kbps play through the player). The
pty smoke's `[m]` Nearest Relay for Oceanside reached no state in 90 s: weatherUSA answers 404 for most
of the mounts its directory advertises (KEC62 included — `curl` reproduces it outside the app), which
is the confirmed external factor behind follow-up F-2, not a Q5 change.

Files: `02-analysis/q5-soak-1h.csv`, `02-analysis/q5-counters/counters-1h.json`.

## 9. Carried forward

- 24 h idle soak (Q0 apparatus, port 6060) ends ~2026-08-27 15:55 UTC → Q6's log.
- Q6 next (seams, `v0.10.0`): the rendered-frame modal exclusivity test → `type modal int`;
  duplicates/knobs; C3 re-check against the counters. Then Q7 (proof + baseline document).
- F-1, F-2 after the pass.
