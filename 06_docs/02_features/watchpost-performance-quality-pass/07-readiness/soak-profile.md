# Soak profile — the workload every M1 number is taken under

Red-team RT-2 / RT-10: a soak that does not name its workload cannot be repeated, and a thread or
memory number without its phase cannot be attributed. This file is the declaration; `scripts/quality/soak.sh`
takes the samples; `tools/slope` reads them; `06-key_learnings/quality-baseline.md` records the results.

## Fixture

| Item | Value |
|---|---|
| Binary | `./dist/watchpost` built from the batch's tag (`make build`; version printed in the header) |
| Config | 10 favourites (the HUM LEAD watchlist: 5 coastal, 2 mountain, 3 inland), RECENT = the 50 seeds, FIRMS keyed, `[fire]` rules default, theme default, units °F |
| Terminal | 133×44 (the mock size), `TERM=xterm-256color`, colour on, dark background |
| Launch | `WATCHPOST_DEBUG_PPROF=1 ./dist/watchpost` under the pty harness (`02-analysis/discover-run/run.expect` lineage) so `/debug/counters` and `/debug/dump` are reachable |
| Sampling | `soak.sh <pid> <hours> soak.csv 300` — every 5 min: RSS, footprint (vmmap physical footprint on macOS, `/proc/<pid>/smaps_rollup` Pss on Linux), threads, CPU, post-GC heap (`heap_alloc`, `heap_inuse`, `heap_objects`), goroutines, fds, disk-cache files/bytes, RECENT publishes; a dump every hour |
| Statistic | `go run ./tools/slope -in soak.csv` (defaults: 6 h warm-up, 1 h per-hour minima, 30-day horizon, bar 5 % of the post-GC heap plateau) |

## Phase schedule (72 h)

| Phase | Hours | State | What it attributes |
|---|---|---|---|
| A idle | 0–2 | radio off, viz off, no modals; nothing but the pipelines | the idle footprint (§1 row 1), the Go thread count with no audio |
| B synth | 2–26 | Synth on, Repeat: Watchlist, volume 5, viz off | the radio-on plateau, the PCM cache bound, `say`/Piper subprocess threads |
| C viz | 26–30 | Synth on, viz on | the 50 ms tick path (render churn at its worst) |
| D relay | 30–54 | Nearest Relay (a wxradio mount, then a weatherUSA mount after Q1), viz off | the ICY path, stream client sockets, Apple audio threads (macOS: named by `sample`) |
| E storm | 54–56 | radio off; the disk cache emptied at 54:00 (`rm` the `http/` dir contents while running) so every tier re-fetches | the cache-miss burst: in-flight caps, disk writer, the OS-thread ratchet's true peak |
| F settle | 56–72 | radio off, viz off | the after-storm plateau: threads must return to phase A's count (bounded ratchet), heap slope over the full run |

The same schedule runs on macOS (this machine) and on Arch (HUM LEAD's laptop). Phase changes are logged by
hand into `soak-phases.log` next to the CSV (`<utc> <phase>`), so the rows can be split by phase.

## Thread bound by construction (RT-2)

`threadcreate` profiles record `<unknown>` stacks (Go's template thread), so threads are attributed by
construction and by phase, not by profile:

`GOMAXPROCS + 2 (sysmon, template) + 3 clients × (4 disk reads + 1 writer) + ≤ 2 (say / Piper) + one-offs (tz load, `defaults`)`

= 12 + 2 + 15 + 2 + ~3 ≈ **34 on a 12-core macOS**; phases A and F must agree and never exceed it; the Apple
audio threads (5) appear in B–D and are listed by `sample` in the baseline document.

## Warm launch (§1)

Median of 10 launches with the disk cache primed by a run ≥ 1 h old (every ≥ 5-min-TTL entry present):
`WATCHPOST_DEBUG_TIMING=1 ./dist/watchpost` + quit at first full view, recorded per platform.
