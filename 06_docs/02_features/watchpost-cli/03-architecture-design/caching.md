# Caching Strategy

> B3 UAT 71. Audience: engineers picking this up cold. One page, no surprises.

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
| **Memory** | `httpx.cache.mem`, every client | Same product, many locations: 60 coastal rows share a handful of buoys and gauges; the daily HIGH fill and `nws-marine` share one gridpoint download. **Budgeted at 8 MB** (UAT 73): expired entries go first, then least-recently-used; a body over 2 MB (the CO-OPS station lists) is disk-only — parsed once, re-read from disk if ever needed. |
| **Disk** | `$XDG_CACHE_HOME/watchpost/http/` (`~/Library/Caches/watchpost/http` on macOS) | A relaunch within a lifetime is warm: points + stations (1 day) are 2 of the ~5 calls per location; the hourly forecast is a third for an hour. One `.cache` file per URL (SHA-256 name): a one-line JSON header (redacted URL, `expires`) then the **raw body** — open it in any editor; read back with no decoding. Writes happen on one goroutine off the request path. Public weather data only; safe to delete at any time. Best effort: a read-only disk loses the warm relaunch, never a fetch. |
| **Negative** | memory, 30 s | A non-retryable 4xx (a buoy with no product, a gauge with no datum) is not re-hit every cycle, yet heals on the scheduler's 10/20/40 s retries. 429 is never negative-cached. |

Cache hits **never consume a pacing token**, so a warm launch is also a fast one.

## What is *not* in this layer

- **Singleflight** — identical concurrent GETs share one request (the four tiers all resolve a
  location at launch). Lives in the client, next to the cache; not a lifetime rule.
- **Semantic caches** — `nws.gridInfo` (resolved grid + station fallback chain + preferred
  station) and the parsed NDBC/CO-OPS station lists. These cache *decisions* derived from
  products, not products; the products beneath them are URL-cached like everything else.
- **Last-good data** — the assembler keeps a location's previous sections when a fetch fails
  (`provider_error` warning, status degraded). That is resilience, not caching.
- **Pacing & lanes** — the token bucket (30 req/s NWS/NDBC, 5 req/s CO-OPS) and the priority
  lane for the favourites are politeness controls; they decide *when*, the cache decides
  *whether*.
- **Resource ceilings** (UAT 73) — 16 requests in flight per client on the normal lane, 8 on the
  priority lane, 8 connections per host, a pure-Go DNS resolver. These bound OS threads and
  sockets during the launch burst; they are not caching either.

## Reading the numbers

Launch, cold, 60 locations: ~5 calls each (points, stations, obs, forecast, hourly) + marine
(gridpoint, buoy, tides). Launch, warm within an hour: points, stations, hourly and all
astronomical products come from disk — roughly half the calls, none of the big ones. Steady
state: one call per product per lifetime, however many locations share it.

## Adding a provider

1. Call `client.GetJSON`/`GetText`. Pass `httpx.TTL(...)` only when you know better than the
   server's headers, and say why in a comment next to the constant.
2. Do not add a memo. Do not add a disk file. Do not parse the same body twice per cycle if a
   parsed form is what you actually reuse — cache *that* (a semantic cache) with a stated
   lifetime, like the station lists.
