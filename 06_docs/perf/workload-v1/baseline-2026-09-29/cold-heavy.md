# workload-v1 cold heavy commit=ef088a21 host=Darwin arm64 at=2026-09-29T17:33:11Z

## Resources by phase

| Phase | Minutes | Samples | CPU % | Footprint MB (median / max) | Heap MB (median) | Goroutines | Threads (max) |
|---|---|---|---|---|---|---|---|

## Timings (ms)

| Phase | Trigger | Event | n | Median | p90 | Max |
|---|---|---|---|---|---|---|
| C1 open (cold) | key | key | 1 | 37 | 37 | 37 |
| C1 open (cold) | open | answered:feed | 2 | 28571 | 38890 | 38890 |
| C1 open (cold) | open | answered:radar | 1 | 1742 | 1742 | 1742 |
| C1 open (cold) | open | answered:temp | 2 | 3308 | 3328 | 3328 |
| C1 open (cold) | open | complete | 1 | 281 | 281 | 281 |
| C2 open (cold) | key | key | 1 | 37 | 37 | 37 |
| C2 open (cold) | open | answered:feed | 2 | 37459 | 42224 | 42224 |
| C2 open (cold) | open | answered:radar | 1 | 1566 | 1566 | 1566 |
| C2 open (cold) | open | answered:temp | 2 | 3257 | 3275 | 3275 |
| C2 open (cold) | open | complete | 1 | 290 | 290 | 290 |
| C3 open (cold) | key | key | 1 | 36 | 36 | 36 |
| C3 open (cold) | open | answered:feed | 1 | 43284 | 43284 | 43284 |
| C3 open (cold) | open | answered:radar | 1 | 1601 | 1601 | 1601 |
| C3 open (cold) | open | answered:temp | 2 | 5471 | 5492 | 5492 |
| C3 open (cold) | open | complete | 1 | 319 | 319 | 319 |
| C4 open (cold) | key | key | 1 | 37 | 37 | 37 |
| C4 open (cold) | open | answered:feed | 1 | 43242 | 43242 | 43242 |
| C4 open (cold) | open | answered:radar | 1 | 1766 | 1766 | 1766 |
| C4 open (cold) | open | answered:temp | 2 | 3119 | 3141 | 3141 |
| C4 open (cold) | open | complete | 1 | 527 | 527 | 527 |
| C5 open (cold) | key | key | 1 | 36 | 36 | 36 |
| C5 open (cold) | open | answered:feed | 4 | 27954 | 42476 | 42476 |
| C5 open (cold) | open | answered:radar | 1 | 1801 | 1801 | 1801 |
| C5 open (cold) | open | answered:temp | 2 | 3042 | 3063 | 3063 |
| C5 open (cold) | open | complete | 1 | 294 | 294 | 294 |
