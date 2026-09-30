# workload-v1 cold default commit=2a6afd19 host=Darwin arm64 at=2026-09-30T02:33:26Z

## Resources by phase

| Phase | Minutes | Samples | CPU % | Footprint MB (median / max) | Heap MB (median) | Goroutines | Threads (max) |
|---|---|---|---|---|---|---|---|

## Render check

PASS: the map was drawn in every phase - no draw fell under its floor.

## Timings (ms)

| Phase | Trigger | Event | n | Median | p90 | Max |
|---|---|---|---|---|---|---|
| C1 open (cold) | key | key | 1 | 37 | 37 | 37 |
| C1 open (cold) | open | answered:feed | 1 | 5310 | 5310 | 5310 |
| C1 open (cold) | open | answered:radar | 1 | 1711 | 1711 | 1711 |
| C1 open (cold) | open | answered:temp | 1 | 5548 | 5548 | 5548 |
| C1 open (cold) | open | complete | 1 | 465 | 465 | 465 |
| C1 open (cold) | open | m5 | 1 | 7002 | 7002 | 7002 |
| C1 open (cold) | open | settled | 1 | 9943 | 9943 | 9943 |
| C2 open (cold) | key | key | 1 | 36 | 36 | 36 |
| C2 open (cold) | open | answered:feed | 1 | 5248 | 5248 | 5248 |
| C2 open (cold) | open | answered:radar | 1 | 1573 | 1573 | 1573 |
| C2 open (cold) | open | answered:temp | 1 | 2162 | 2162 | 2162 |
| C2 open (cold) | open | complete | 1 | 264 | 264 | 264 |
| C2 open (cold) | open | m5 | 1 | 6944 | 6944 | 6944 |
| C2 open (cold) | open | settled | 1 | 10025 | 10025 | 10025 |
| C3 open (cold) | key | key | 1 | 36 | 36 | 36 |
| C3 open (cold) | open | answered:feed | 1 | 5469 | 5469 | 5469 |
| C3 open (cold) | open | answered:radar | 1 | 1629 | 1629 | 1629 |
| C3 open (cold) | open | answered:temp | 1 | 2561 | 2561 | 2561 |
| C3 open (cold) | open | complete | 1 | 257 | 257 | 257 |
| C3 open (cold) | open | m5 | 1 | 7170 | 7170 | 7170 |
| C3 open (cold) | open | settled | 1 | 9936 | 9936 | 9936 |
| C4 open (cold) | key | key | 1 | 36 | 36 | 36 |
| C4 open (cold) | open | answered:feed | 1 | 5332 | 5332 | 5332 |
| C4 open (cold) | open | answered:radar | 1 | 1697 | 1697 | 1697 |
| C4 open (cold) | open | answered:temp | 1 | 2139 | 2139 | 2139 |
| C4 open (cold) | open | complete | 1 | 239 | 239 | 239 |
| C4 open (cold) | open | m5 | 1 | 7035 | 7035 | 7035 |
| C4 open (cold) | open | settled | 1 | 10065 | 10065 | 10065 |
| C5 open (cold) | key | key | 1 | 36 | 36 | 36 |
| C5 open (cold) | open | answered:feed | 1 | 5432 | 5432 | 5432 |
| C5 open (cold) | open | answered:radar | 1 | 1732 | 1732 | 1732 |
| C5 open (cold) | open | answered:temp | 1 | 2189 | 2189 | 2189 |
| C5 open (cold) | open | complete | 1 | 245 | 245 | 245 |
| C5 open (cold) | open | m5 | 1 | 7140 | 7140 | 7140 |
| C5 open (cold) | open | settled | 1 | 9931 | 9931 | 9931 |
