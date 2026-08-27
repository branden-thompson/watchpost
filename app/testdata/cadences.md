| Pipeline | Tier | Cadence |
|---|---|---|
| Priority (≤ 10 favourites, one batched scheduler) | alerts | 20s |
| Priority (≤ 10 favourites, one batched scheduler) | observations | 1m30s |
| Priority (≤ 10 favourites, one batched scheduler) | marine observations | 10m0s |
| Priority (≤ 10 favourites, one batched scheduler) | forecast (daily) | 30m0s |
| Priority (≤ 10 favourites, one batched scheduler) | forecast (hourly) | 30m0s |
| Priority (≤ 10 favourites, one batched scheduler) | marine forecast | 30m0s |
| Priority (≤ 10 favourites, one batched scheduler) | fire | 10m0s |
| Priority (≤ 10 favourites, one batched scheduler) | seismic | 5m0s |
| RECENT (one scheduler per location) | observations | 10m0s |
| RECENT (one scheduler per location) | marine observations | 10m0s |
| RECENT (one scheduler per location) | forecast (daily) | 1h0m0s |
| RECENT (one scheduler per location) | marine forecast | 1h0m0s |
| RECENT (one scheduler per location) | fire | 15m0s |
| RECENT (one scheduler per location) | seismic | 15m0s |
| RECENT (one batched scheduler for the list) | alerts | 2m0s |
| RECENT | forecast (hourly) | on demand (Details / lookup) |

Rehydrate on failure: 10 s / 20 s / 40 s (sched); publish coalescing window: 50ms.
