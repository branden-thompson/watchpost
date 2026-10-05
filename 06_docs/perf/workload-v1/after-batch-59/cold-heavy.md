# workload-v1 cold heavy commit=cf2e2a37 host=Darwin arm64 at=2026-09-29T21:51:23Z

## Resources by phase

| Phase | Minutes | Samples | CPU % | Footprint MB (median / max) | Heap MB (median) | Goroutines | Threads (max) |
|---|---|---|---|---|---|---|---|

## Render check

PASS: the map was drawn in every phase - no draw fell under its floor.

## Timings (ms)

| Phase | Trigger | Event | n | Median | p90 | Max |
|---|---|---|---|---|---|---|
| C1 open (cold) | key | key | 1 | 38 | 38 | 38 |
| C1 open (cold) | open | answered:feed | 1 | 7455 | 7455 | 7455 |
| C1 open (cold) | open | answered:radar | 1 | 1689 | 1689 | 1689 |
| C1 open (cold) | open | answered:temp | 1 | 5509 | 5509 | 5509 |
| C1 open (cold) | open | complete | 1 | 253 | 253 | 253 |
| C1 open (cold) | open | m5 | 1 | 10022 | 10022 | 10022 |
| C1 open (cold) | open | settled | 1 | 13420 | 13420 | 13420 |
| C2 open (cold) | key | key | 1 | 36 | 36 | 36 |
| C2 open (cold) | open | answered:feed | 1 | 7355 | 7355 | 7355 |
| C2 open (cold) | open | answered:radar | 1 | 1606 | 1606 | 1606 |
| C2 open (cold) | open | answered:temp | 1 | 5656 | 5656 | 5656 |
| C2 open (cold) | open | complete | 1 | 303 | 303 | 303 |
| C2 open (cold) | open | m5 | 1 | 10200 | 10200 | 10200 |
| C2 open (cold) | open | settled | 1 | 10200 | 10200 | 10200 |
| C3 open (cold) | key | key | 1 | 36 | 36 | 36 |
| C3 open (cold) | open | answered:feed | 1 | 7416 | 7416 | 7416 |
| C3 open (cold) | open | answered:radar | 1 | 1506 | 1506 | 1506 |
| C3 open (cold) | open | answered:temp | 1 | 7506 | 7506 | 7506 |
| C3 open (cold) | open | complete | 1 | 278 | 278 | 278 |
| C3 open (cold) | open | m5 | 1 | 10673 | 10673 | 10673 |
| C3 open (cold) | open | settled | 1 | 14237 | 14237 | 14237 |
| C4 open (cold) | key | key | 1 | 37 | 37 | 37 |
| C4 open (cold) | open | answered:feed | 1 | 7466 | 7466 | 7466 |
| C4 open (cold) | open | answered:radar | 1 | 1665 | 1665 | 1665 |
| C4 open (cold) | open | answered:temp | 1 | 5506 | 5506 | 5506 |
| C4 open (cold) | open | complete | 1 | 487 | 487 | 487 |
| C4 open (cold) | open | m5 | 1 | 10056 | 10056 | 10056 |
| C4 open (cold) | open | settled | 1 | 13572 | 13572 | 13572 |
| C5 open (cold) | key | key | 1 | 44 | 44 | 44 |
| C5 open (cold) | open | answered:feed | 1 | 7463 | 7463 | 7463 |
| C5 open (cold) | open | answered:radar | 1 | 1675 | 1675 | 1675 |
| C5 open (cold) | open | answered:temp | 1 | 6316 | 6316 | 6316 |
| C5 open (cold) | open | complete | 1 | 291 | 291 | 291 |
| C5 open (cold) | open | m5 | 1 | 10053 | 10053 | 10053 |
| C5 open (cold) | open | settled | 1 | 14236 | 14236 | 14236 |
