# Infra ledger — the measuring apparatus this pass adds and what each piece reads

Owned by the quality pass (red-team RT-14 asked for one owner; the 0.9.x ledger stays with `watchpost-cli`).
One row per instrument: where it lives, what it emits, who reads it, its bound.

| Instrument | Lives in | Emits | Read by | Bound / cost |
|---|---|---|---|---|
| `httpx.RequestStats` | `platform/httpx/stats.go` | per-host counters since launch: attempts, net bodies, cache hits, negative hits, 304s (Q5), bytes, h2 responses, TLS handshakes | `[S]` modal, `report --verbose`, `counters.json` | 8 host slots + `other`; one mutex; zero allocation on the request path beyond the closure |
| `httpx.CacheStats` | `platform/httpx/cache.go` | memory-tier entries/bytes, negative-cache entries | gauges | existing |
| publish counters | `app/dashboard.go` `publisher` | publishes and folded triggers per pipeline; the last snapshot pointer | `[S]` PIPELINES rows, `counters.json` (`snapshot_bytes` measured only at dump time) | two atomics + one pointer per pipeline |
| gauges | `app/stats.go` `diagSources.gauges()` | `httpx.mem.entries`, `httpx.neg.entries`, `nws.gridinfo`, `coops.stations`, `hms.memo.points`, `assembler.{priority,recent}.{locations,warnings}`, `synth.pcm.cache`, `tz.cache`, `disk.cache`, `disk.profiles`, `disk.voices` | `counters.json` | each owner exposes one accessor; directories are sized on demand (flat for the cache, recursive for profiles/voices) |
| runtime counts | `app/stats.go` `runtimeCounts()` | goroutines, OS threads ever created (`threadcreate` count), open fds (`/dev/fd`, −1 when absent) | `counters.json`, `soak.csv` | — |
| dump | `app/dump.go` + `dump_unix.go` / `dump_windows.go` | `profiles/<ts>/{heap,allocs,goroutine,threadcreate}.pb.gz` + `counters.json` | `pprof -base`, `slope`, the baseline document | one in flight; ≥ 60 s apart; newest 12 kept; 0700/0600; zero idle cost |
| triggers | SIGUSR1 (Unix) · `/debug/dump` and `/debug/counters` on the `WATCHPOST_DEBUG_PPROF=1` loopback server (all platforms) | — | `soak.sh`, `dump.sh`, a person | same-UID trust boundary; loopback only; opt-in |
| `soak.sh` | `scripts/quality/soak.sh` | `soak.csv` rows every N s (OS view + counters) and an hourly dump | `slope`, baseline document | reads `ps -o` columns and our own JSON only (never a raw process listing — RT-15) |
| `dump.sh` | `scripts/quality/dump.sh` | one dump, prints its directory | a person | — |
| `slope` | `tools/slope` | slope, HAC 95 % CI, 30-day projection, detection floor, verdict PASS / GROWTH / UNCERTIFIABLE / INSUFFICIENT | the Q0 and Q7 gates | stdlib only; per-window minima; Newey–West lag ⌊4(n/100)^{2/9}⌋ |
| benchmarks | `modes/tty/bench_test.go`, `domains/fire/hms/bench_test.go`, `platform/render/bench_test.go`, `platform/snapshot/bench_test.go` | ns/op, B/op, allocs/op at 133×44 / 133×70 / 200×60, colour on | `make quality-bench` (local, recorded) | never gated on time |
| allocation pins | `modes/tty/bench_test.go` `TestFrameAllocBudget` | allocs per `View()` vs budget | `make alloc-budget` (CI, non-race) | budget = Q0 measurement × 1.05, lowered by Q3/Q4b |
| `make p10` + `p10-unmatched.sh` | `Makefile`, `scripts/quality/p10-unmatched.sh` | `dist/p10.json`, unmatched ledger entries | every batch gate (local) | fails loud without the CLI |
| `make tidy` / `make vuln` | `Makefile` | tidy diff, module verification, vulnerability report | `make verify`, CI | — |

## Baseline instruments carried from DISCOVER

| Instrument | Where | Note |
|---|---|---|
| PID 67943 sampler | `02-analysis/baseline-pid67943.log` | v0.9.4 release binary; ps/vmmap/threads every 5 min from 2026-08-26 12:02 UTC (hourly row + gap before) |
| instrumented run | `02-analysis/discover-run/` | proxy connection log, 5-min samples, pprof sets every 30 min |
| lens benchmarks | `02-analysis/lens-benches/` (`.go.txt`) | the L1/L4/L5 sources the in-package benchmarks were ported from |
