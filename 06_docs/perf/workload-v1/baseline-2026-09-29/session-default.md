# workload-v1 session default commit=ef088a21 host=Darwin arm64 at=2026-09-29T17:37:22Z

## Resources by phase

| Phase | Minutes | Samples | CPU % | Footprint MB (median / max) | Heap MB (median) | Goroutines | Threads (max) |
|---|---|---|---|---|---|---|---|
| A idle (radio off) | 10.0 | 28 | 1.3 | 90.1 / 94.0 | 33.7 | 484 | 33 |
| B radio on (synth) | 10.0 | 27 | 1.3 | 103.7 / 112.9 | 45.1 | 488 | 33 |
| C map open at the place (cold), a pan every 15 s | 10.1 | 27 | 7.7 | 197.7 / 236.9 | 78.6 | 502 | 34 |
| D map national (lower 48), loop playing | 10.1 | 27 | 12.9 | 231.0 / 246.2 | 98.7 | 499 | 34 |
| E map zoomed in three steps, a pan every 15 s | 10.0 | 28 | 7.9 | 223.1 / 280.4 | 86.3 | 500 | 34 |
| F map closed, idle | 10.0 | 27 | 1.3 | 166.3 / 210.2 | 86.8 | 498 | 34 |
| W2 open (warm) | 0.6 | 2 | 3.4 | 227.6 / 228.7 | 92.0 | 501 | 33 |
| W4 open (warm) | 0.6 | 2 | 3.5 | 233.5 / 236.1 | 94.2 | 504 | 33 |
| W5 open (warm) | 0.6 | 2 | 3.5 | 232.0 / 235.3 | 94.3 | 506 | 33 |

## Timings (ms)

| Phase | Trigger | Event | n | Median | p90 | Max |
|---|---|---|---|---|---|---|
| B radio on (synth) | key | key | 2 | 0 | 0 | 0 |
| C map open at the place (cold), a pan every 15 s | key | key | 42 | 38 | 99 | 112 |
| C map open at the place (cold), a pan every 15 s | map.pan.right | answered:feed | 112 | 680 | 10690 | 14519 |
| C map open at the place (cold), a pan every 15 s | map.pan.right | answered:radar | 39 | 696 | 2006 | 2163 |
| C map open at the place (cold), a pan every 15 s | map.pan.right | answered:temp | 40 | 708 | 2833 | 8160 |
| C map open at the place (cold), a pan every 15 s | map.pan.right | complete | 40 | 64 | 241 | 321 |
| C map open at the place (cold), a pan every 15 s | map.pan.right | m5 | 39 | 892 | 1508 | 6650 |
| C map open at the place (cold), a pan every 15 s | map.pan.right | settled | 39 | 64 | 1077 | 3430 |
| C map open at the place (cold), a pan every 15 s | open | answered:feed | 1 | 5192 | 5192 | 5192 |
| C map open at the place (cold), a pan every 15 s | open | answered:radar | 1 | 1541 | 1541 | 1541 |
| C map open at the place (cold), a pan every 15 s | open | answered:temp | 2 | 4519 | 4531 | 4531 |
| C map open at the place (cold), a pan every 15 s | open | complete | 1 | 535 | 535 | 535 |
| C map open at the place (cold), a pan every 15 s | open | m5 | 1 | 7913 | 7913 | 7913 |
| C map open at the place (cold), a pan every 15 s | open | settled | 1 | 7913 | 7913 | 7913 |
| C map open at the place (cold), a pan every 15 s | tick | tick | 1 | 3 | 3 | 3 |
| D map national (lower 48), loop playing | key | key | 2 | 6 | 76 | 76 |
| D map national (lower 48), loop playing | map.play | answered:feed | 40 | 296208 | 572650 | 594070 |
| D map national (lower 48), loop playing | map.play | answered:radar | 5 | 253358 | 533664 | 533664 |
| D map national (lower 48), loop playing | map.play | answered:temp | 1 | 6459 | 6459 | 6459 |
| D map national (lower 48), loop playing | map.play | complete | 2 | 75 | 127 | 127 |
| D map national (lower 48), loop playing | map.play | m5 | 1 | 13203 | 13203 | 13203 |
| D map national (lower 48), loop playing | map.play | settled | 2 | 75 | 9159 | 9159 |
| D map national (lower 48), loop playing | map.region.1 | answered:feed | 1 | 3009 | 3009 | 3009 |
| D map national (lower 48), loop playing | tick | tick | 540 | 0 | 1 | 455 |
| E map zoomed in three steps, a pan every 15 s | key | key | 44 | 7 | 99 | 561 |
| E map zoomed in three steps, a pan every 15 s | map.pan.left | answered:feed | 111 | 720 | 9562 | 14765 |
| E map zoomed in three steps, a pan every 15 s | map.pan.left | answered:radar | 39 | 638 | 1969 | 3337 |
| E map zoomed in three steps, a pan every 15 s | map.pan.left | answered:temp | 39 | 667 | 3377 | 7438 |
| E map zoomed in three steps, a pan every 15 s | map.pan.left | complete | 40 | 6 | 165 | 1399 |
| E map zoomed in three steps, a pan every 15 s | map.pan.left | m5 | 39 | 787 | 2174 | 5685 |
| E map zoomed in three steps, a pan every 15 s | map.pan.left | settled | 40 | 6 | 1828 | 4651 |
| E map zoomed in three steps, a pan every 15 s | map.zoom.in | answered:feed | 2 | 2648 | 9399 | 9399 |
| E map zoomed in three steps, a pan every 15 s | map.zoom.in | answered:radar | 1 | 5829 | 5829 | 5829 |
| E map zoomed in three steps, a pan every 15 s | map.zoom.in | answered:temp | 1 | 7417 | 7417 | 7417 |
| E map zoomed in three steps, a pan every 15 s | map.zoom.in | complete | 1 | 477 | 477 | 477 |
| E map zoomed in three steps, a pan every 15 s | map.zoom.in | m5 | 1 | 2957 | 2957 | 2957 |
| E map zoomed in three steps, a pan every 15 s | map.zoom.in | settled | 1 | 7513 | 7513 | 7513 |
| W1 open (warm) | key | key | 2 | 0 | 39 | 39 |
| W1 open (warm) | open | answered:feed | 3 | 8067 | 28071 | 28071 |
| W1 open (warm) | open | answered:radar | 1 | 797 | 797 | 797 |
| W1 open (warm) | open | answered:temp | 1 | 46 | 46 | 46 |
| W1 open (warm) | open | complete | 1 | 375 | 375 | 375 |
| W2 open (warm) | key | key | 2 | 0 | 59 | 59 |
| W2 open (warm) | open | answered:feed | 2 | 444 | 13522 | 13522 |
| W2 open (warm) | open | answered:radar | 1 | 63 | 63 | 63 |
| W2 open (warm) | open | answered:temp | 1 | 90 | 90 | 90 |
| W2 open (warm) | open | complete | 1 | 58 | 58 | 58 |
| W3 open (warm) | key | key | 2 | 0 | 95 | 95 |
| W3 open (warm) | open | answered:feed | 3 | 8769 | 18646 | 18646 |
| W3 open (warm) | open | answered:radar | 1 | 328 | 328 | 328 |
| W3 open (warm) | open | answered:temp | 1 | 102 | 102 | 102 |
| W3 open (warm) | open | complete | 1 | 94 | 94 | 94 |
| W4 open (warm) | key | key | 2 | 0 | 95 | 95 |
| W4 open (warm) | open | answered:feed | 3 | 3771 | 23772 | 23772 |
| W4 open (warm) | open | answered:radar | 1 | 100 | 100 | 100 |
| W4 open (warm) | open | answered:temp | 1 | 165 | 165 | 165 |
| W4 open (warm) | open | complete | 1 | 94 | 94 | 94 |
| W5 open (warm) | key | key | 2 | 0 | 95 | 95 |
| W5 open (warm) | open | answered:feed | 3 | 8766 | 29018 | 29018 |
| W5 open (warm) | open | answered:radar | 1 | 572 | 572 | 572 |
| W5 open (warm) | open | answered:temp | 1 | 102 | 102 | 102 |
| W5 open (warm) | open | complete | 1 | 93 | 93 | 93 |
