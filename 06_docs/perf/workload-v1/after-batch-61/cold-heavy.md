# workload-v1 cold heavy commit=2a6afd19 host=Darwin arm64 at=2026-09-30T02:37:28Z

## Resources by phase

| Phase | Minutes | Samples | CPU % | Footprint MB (median / max) | Heap MB (median) | Goroutines | Threads (max) |
|---|---|---|---|---|---|---|---|

## Render check

PASS: the map was drawn in every phase - no draw fell under its floor.

## Timings (ms)

| Phase | Trigger | Event | n | Median | p90 | Max |
|---|---|---|---|---|---|---|
| C1 open (cold) | key | key | 1 | 38 | 38 | 38 |
| C1 open (cold) | open | answered:feed | 1 | 5019 | 5019 | 5019 |
| C1 open (cold) | open | answered:radar | 1 | 1814 | 1814 | 1814 |
| C1 open (cold) | open | answered:temp | 1 | 3427 | 3427 | 3427 |
| C1 open (cold) | open | complete | 1 | 238 | 238 | 238 |
| C1 open (cold) | open | m5 | 1 | 7304 | 7304 | 7304 |
| C1 open (cold) | open | settled | 1 | 7304 | 7304 | 7304 |
| C2 open (cold) | key | key | 1 | 37 | 37 | 37 |
| C2 open (cold) | open | answered:feed | 1 | 6540 | 6540 | 6540 |
| C2 open (cold) | open | answered:radar | 1 | 1505 | 1505 | 1505 |
| C2 open (cold) | open | answered:temp | 1 | 3247 | 3247 | 3247 |
| C2 open (cold) | open | complete | 1 | 254 | 254 | 254 |
| C2 open (cold) | open | m5 | 1 | 8529 | 8529 | 8529 |
| C2 open (cold) | open | settled | 1 | 15326 | 15326 | 15326 |
| C3 open (cold) | key | key | 1 | 36 | 36 | 36 |
| C3 open (cold) | open | answered:feed | 1 | 7557 | 7557 | 7557 |
| C3 open (cold) | open | answered:radar | 1 | 1788 | 1788 | 1788 |
| C3 open (cold) | open | answered:temp | 1 | 6308 | 6308 | 6308 |
| C3 open (cold) | open | complete | 1 | 268 | 268 | 268 |
| C3 open (cold) | open | m5 | 1 | 9554 | 9554 | 9554 |
| C3 open (cold) | open | settled | 1 | 14138 | 14138 | 14138 |
| C4 open (cold) | key | key | 1 | 38 | 38 | 38 |
| C4 open (cold) | open | answered:feed | 1 | 7468 | 7468 | 7468 |
| C4 open (cold) | open | answered:radar | 1 | 1928 | 1928 | 1928 |
| C4 open (cold) | open | answered:temp | 1 | 3172 | 3172 | 3172 |
| C4 open (cold) | open | complete | 1 | 525 | 525 | 525 |
| C4 open (cold) | open | m5 | 1 | 10061 | 10061 | 10061 |
| C4 open (cold) | open | settled | 1 | 14115 | 14115 | 14115 |
| C5 open (cold) | key | key | 1 | 36 | 36 | 36 |
| C5 open (cold) | open | answered:feed | 1 | 7378 | 7378 | 7378 |
| C5 open (cold) | open | answered:radar | 1 | 1546 | 1546 | 1546 |
| C5 open (cold) | open | answered:temp | 1 | 3111 | 3111 | 3111 |
| C5 open (cold) | open | complete | 1 | 246 | 246 | 246 |
| C5 open (cold) | open | m5 | 1 | 9375 | 9375 | 9375 |
| C5 open (cold) | open | settled | 1 | 13070 | 13070 | 13070 |
