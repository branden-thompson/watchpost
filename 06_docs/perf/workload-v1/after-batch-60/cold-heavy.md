# workload-v1 cold heavy commit=c86d6e89 host=Darwin arm64 at=2026-09-30T01:51:12Z

## Resources by phase

| Phase | Minutes | Samples | CPU % | Footprint MB (median / max) | Heap MB (median) | Goroutines | Threads (max) |
|---|---|---|---|---|---|---|---|

## Render check

PASS: the map was drawn in every phase - no draw fell under its floor.

## Timings (ms)

| Phase | Trigger | Event | n | Median | p90 | Max |
|---|---|---|---|---|---|---|
| C1 open (cold) | key | key | 1 | 37 | 37 | 37 |
| C1 open (cold) | open | answered:feed | 1 | 7762 | 7762 | 7762 |
| C1 open (cold) | open | answered:radar | 1 | 1807 | 1807 | 1807 |
| C1 open (cold) | open | answered:temp | 1 | 3141 | 3141 | 3141 |
| C1 open (cold) | open | complete | 1 | 255 | 255 | 255 |
| C1 open (cold) | open | m5 | 1 | 9768 | 9768 | 9768 |
| C1 open (cold) | open | settled | 1 | 13108 | 13108 | 13108 |
| C2 open (cold) | key | key | 1 | 38 | 38 | 38 |
| C2 open (cold) | open | answered:feed | 1 | 7387 | 7387 | 7387 |
| C2 open (cold) | open | answered:radar | 1 | 1735 | 1735 | 1735 |
| C2 open (cold) | open | answered:temp | 1 | 3194 | 3194 | 3194 |
| C2 open (cold) | open | complete | 1 | 266 | 266 | 266 |
| C2 open (cold) | open | m5 | 1 | 9383 | 9383 | 9383 |
| C2 open (cold) | open | settled | 1 | 13192 | 13192 | 13192 |
| C3 open (cold) | key | key | 1 | 36 | 36 | 36 |
| C3 open (cold) | open | answered:feed | 1 | 7666 | 7666 | 7666 |
| C3 open (cold) | open | answered:radar | 1 | 1699 | 1699 | 1699 |
| C3 open (cold) | open | answered:temp | 1 | 6044 | 6044 | 6044 |
| C3 open (cold) | open | complete | 1 | 244 | 244 | 244 |
| C3 open (cold) | open | m5 | 1 | 9886 | 9886 | 9886 |
| C3 open (cold) | open | settled | 1 | 10350 | 10350 | 10350 |
| C4 open (cold) | key | key | 1 | 37 | 37 | 37 |
| C4 open (cold) | open | answered:feed | 1 | 7602 | 7602 | 7602 |
| C4 open (cold) | open | answered:radar | 1 | 1633 | 1633 | 1633 |
| C4 open (cold) | open | answered:temp | 1 | 3194 | 3194 | 3194 |
| C4 open (cold) | open | complete | 1 | 445 | 445 | 445 |
| C4 open (cold) | open | m5 | 1 | 9635 | 9635 | 9635 |
| C4 open (cold) | open | settled | 1 | 13488 | 13488 | 13488 |
| C5 open (cold) | key | key | 1 | 36 | 36 | 36 |
| C5 open (cold) | open | answered:feed | 1 | 7438 | 7438 | 7438 |
| C5 open (cold) | open | answered:radar | 1 | 1927 | 1927 | 1927 |
| C5 open (cold) | open | answered:temp | 1 | 3413 | 3413 | 3413 |
| C5 open (cold) | open | complete | 1 | 252 | 252 | 252 |
| C5 open (cold) | open | m5 | 1 | 9458 | 9458 | 9458 |
| C5 open (cold) | open | settled | 1 | 14083 | 14083 | 14083 |
