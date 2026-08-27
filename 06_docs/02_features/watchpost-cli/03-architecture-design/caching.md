# Caching Strategy

> B3 UAT 71; revised by the performance & quality pass, Q1 (2026-08-26). Audience: engineers picking
> this up cold. One page, no surprises.

## The one rule

**Every outbound GET is cached by URL, in the HTTP client, for a lifetime decided in this order:**

1. **The caller said so** — `httpx.TTL(d)`. Product knowledge that outranks the server: tide
   predictions are astronomical (CO-OPS sends `no-store`), NDBC's station list turns over daily
   (it declares 60 s on a 270 KB file).
2. **The server said so** — `Cache-Control: max-age` (or `Expires`). NWS is a model citizen:
   points and station lists 1 day, hourly forecast 1 h, observations 5 min, alerts 5 s, forecast
   and gridpoint "until the next issuance" (~minutes).
3. **Nobody said** — no cache. (`httpx.NoCache()` forces a round trip for anything.)

That is the whole policy. Providers never keep their own product memos; they *state lifetimes*
and call the client. If you find yourself writing a map of things with timestamps, stop and
use `TTL`.

## Where things live

| Tier | Where | Why it exists |
|---|---|---|
| **Memory** | `httpx.cache.mem`, every client | Same product, many locations: 60 coastal rows share a handful of buoys and gauges; the daily HIGH fill and `nws-marine` share one gridpoint download. **Budgeted at 8 MB** (UAT 73): expired entries that nothing can renew go first, then least-recently-used; a body over 2 MB (the CO-OPS station lists) is disk-only — parsed once, re-read from disk if ever needed. |
| **Disk** | `$XDG_CACHE_HOME/watchpost/http/` (`~/Library/Caches/watchpost/http` on macOS) | A relaunch within a lifetime is warm: points + stations (1 day) are 2 of the ~5 calls per location; the hourly forecast is a third for an hour. One `.cache` file per URL (SHA-256 name): a one-line JSON header (redacted URL, `expires`, validators) then the **raw body** — open it in any editor; read back with no decoding. Writes happen on one goroutine off the request path. Public weather data only; safe to delete at any time. Best effort: a read-only disk loses the warm relaunch, never a fetch. |
| **Negative** | memory, 30 s, ≤ 1,024 entries | A non-retryable 4xx (a buoy with no product, a gauge with no datum) is not re-hit every cycle, yet heals on the scheduler's 10/20/40 s retries. 429 is never negative-cached. At the cap the soonest-expiring entry is dropped. |

Cache hits **never consume a pacing token**, so a warm launch is also a fast one.

## The four rules that keep it bounded over weeks (Q1)

DISCOVER found the disk tier at 1,376 files / 116 MB after two days, growing forever: 95 % of writes
persisted entries that could never serve a relaunch, and date-keyed URLs (CO-OPS `begin_date`, NWS
`/products/{id}`, alert zone sets) added files nothing removed. These rules replace that.

| Rule | What it says | Where |
|---|---|---|
| **Persistence floor** | A body is written to disk only when the caller's TTL is **more than 5 minutes**, or the caller passed `httpx.Persist()`. Observations (5 min), alerts (5 s) and anything the server capped short stay in memory. The relay directory (exactly 5 min, and the one short document that warms a relaunch) is the only `Persist()` caller. | `cache.put` |
| **One retention rule** | An expired entry that carries validators (`ETag` / `Last-Modified`, each ≤ 512 bytes, control bytes refused) is kept for **24 h past its expiry** — in memory as an ordinary LRU citizen, on disk untouched by the sweep — so a conditional GET (Q5) has a body to renew. It is never served as fresh. A 304 calls `renew`, which moves the expiry and refreshes the file's mtime. | `evictLocked`, `renew` |
| **Stale-memory skip** | When the memory tier holds an entry that has expired, the disk read is skipped: the file is the same entry or older. (Before Q1 this was ~45,000 reads a day of files known to be expired.) | `cache.get` |
| **The sweep** | Runs on the writer goroutine once at launch and then daily. It is an **allow-list**: it lists the directory (no recursion, at most 10,000 entries), touches only regular files (a symlink is skipped by name), and only names it wrote — `<sha256>.cache`, the pre-0.9 `<sha256>.json` orphans, and its own `*.cache.NNN.tmp` files older than an hour. A cache file is removed when `max(expires, mtime) + 24 h` is past; then, if the directory exceeds **256 MB**, the oldest by mtime go until it fits — at most 1,000 removals per pass. It **refuses to run** unless the directory path contains a `watchpost` element, so a misconfigured `$XDG_CACHE_HOME` can never point it at a home directory. | `cache.sweep` |

Expected disk-write rate after the floor (derived from the DISCOVER inventory, L4-F2 — the Q1 gate
checks the counter against it): alerts 0 (5 s), observations 0 (5 min, not more), FIRMS ≈ 12.8k/day
(10 min TTL, until Q5's tiles), the ≥ 30-min group (forecast, gridpoint, marine, points, stations,
tides) ≈ 10k/day → **≈ 23k writes/day**, down from 45k — and `httpx.disk.writes` in `counters.json`
is the number to compare.

## What is *not* in this layer

- **Singleflight** — identical concurrent GETs share one request (the four tiers all resolve a
  location at launch). Lives in the client, next to the cache; not a lifetime rule.
- **Semantic caches** — `nws.gridInfo` (resolved grid + station fallback chain + preferred
  station), the parsed NDBC/CO-OPS station lists, and the fire feeds' parse memos (`fire.Memo`:
  the HMS archive and, since Q3, the WFIGS layer — parsed once per body change by content hash,
  whoever asks; bound one body each). These cache *decisions* derived from products, not
  products; the products beneath them are URL-cached like everything else. Each states its bound
  and reports its size as a gauge in the diagnostic dump.
- **The read-only body** (Q3) — `GetText` returns the cache's own slice, no copy (the HMS
  archive is 1.4 MB and was copied on every fetch). Parsers read it and never write into it; each
  consumer package pins that with `TestGetTextCallersMustNotMutate`.
- **Last-good data** — the assembler keeps a location's previous sections when a fetch fails
  (`provider_error` warning, status degraded). That is resilience, not caching.
- **Pacing, lanes and the failure memo** — the token bucket (30 req/s NWS/NDBC, 5 req/s CO-OPS),
  the priority lane for the favourites, one retry per request, and the per-host failure memo
  (transport errors and repeated 5xx make the *background* lane fail fast for 20 s; the priority
  lane always tries and heals it) are politeness and resilience controls; they decide *when*,
  the cache decides *whether*.
- **Redirects** — every client follows at most three, same scheme and host only.
- **Resource ceilings** (UAT 73) — 16 requests in flight per client on the normal lane, 8 on the
  priority lane, 8 connections per host, a pure-Go DNS resolver. These bound OS threads and
  sockets during the launch burst; they are not caching either.

## Reading the numbers

Launch, cold, 60 locations: ~5 calls each (points, stations, obs, forecast, hourly) + marine
(gridpoint, buoy, tides). Launch, warm within an hour: points, stations, hourly and all
astronomical products come from disk — roughly half the calls, none of the big ones. Steady
state: one call per product per lifetime, however many locations share it. The `S` modal's
REQUESTS rows and `report --verbose` show the counters per host since launch.

## Adding a provider

1. Call `client.GetJSON`/`GetText`. Pass `httpx.TTL(...)` only when you know better than the
   server's headers, and say why in a comment next to the constant. Pass `httpx.Persist()` only
   for a document under 5 minutes that genuinely warms a relaunch — there is one today.
2. Do not add a memo. Do not add a disk file. Do not parse the same body twice per cycle if a
   parsed form is what you actually reuse — cache *that* (a semantic cache) with a stated
   lifetime and a gauge, like the station lists.
