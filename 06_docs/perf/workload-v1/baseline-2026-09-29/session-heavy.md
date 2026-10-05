# workload-v1 session heavy commit=ef088a21 host=Darwin arm64 at=2026-09-29T18:40:46Z

## Resources by phase

| Phase | Minutes | Samples | CPU % | Footprint MB (median / max) | Heap MB (median) | Goroutines | Threads (max) |
|---|---|---|---|---|---|---|---|
| A idle (radio off) | 10.0 | 27 | 1.2 | 92.3 / 94.6 | 34.4 | 484 | 33 |
| B radio on (synth) | 10.0 | 27 | 1.5 | 105.0 / 115.4 | 45.7 | 488 | 33 |
| C map open at the place (cold), a pan every 15 s | 10.1 | 28 | 5.3 | 194.7 / 222.8 | 76.5 | 501 | 34 |
| D map national (lower 48), loop playing | 10.1 | 27 | 3.5 | 221.0 / 231.2 | 91.4 | 493 | 34 |
| E map zoomed in three steps, a pan every 15 s | 10.0 | 27 | 5.0 | 217.4 / 256.4 | 85.2 | 532 | 34 |
| F map closed, idle | 10.0 | 27 | 1.5 | 168.9 / 209.8 | 86.0 | 529 | 34 |
| W1 open (warm) | 0.6 | 2 | 2.3 | 220.6 / 221.9 | 86.4 | 540 | 34 |
| W3 open (warm) | 0.6 | 2 | 2.7 | 229.3 / 235.4 | 93.4 | 544 | 34 |
| W5 open (warm) | 0.6 | 2 | 3.0 | 230.0 / 237.6 | 93.8 | 550 | 34 |

## Timings (ms)

| Phase | Trigger | Event | n | Median | p90 | Max |
|---|---|---|---|---|---|---|
| B radio on (synth) | key | key | 2 | 0 | 0 | 0 |
| C map open at the place (cold), a pan every 15 s | key | key | 42 | 41 | 91 | 105 |
| C map open at the place (cold), a pan every 15 s | map.pan.right | answered:feed | 109 | 786 | 9426 | 13368 |
| C map open at the place (cold), a pan every 15 s | map.pan.right | answered:radar | 39 | 667 | 1888 | 3464 |
| C map open at the place (cold), a pan every 15 s | map.pan.right | answered:temp | 40 | 693 | 6053 | 14827 |
| C map open at the place (cold), a pan every 15 s | map.region.1 | answered:feed | 1 | 125 | 125 | 125 |
| C map open at the place (cold), a pan every 15 s | map.region.1 | answered:radar | 1 | 1841 | 1841 | 1841 |
| C map open at the place (cold), a pan every 15 s | open | answered:feed | 1 | 9091 | 9091 | 9091 |
| C map open at the place (cold), a pan every 15 s | open | answered:radar | 1 | 1816 | 1816 | 1816 |
| C map open at the place (cold), a pan every 15 s | open | complete | 1 | 503 | 503 | 503 |
| D map national (lower 48), loop playing | key | key | 3 | 6 | 96 | 96 |
| D map national (lower 48), loop playing | map.play | answered:feed | 37 | 304980 | 584145 | 585963 |
| D map national (lower 48), loop playing | map.play | answered:radar | 4 | 246671 | 524774 | 524774 |
| D map national (lower 48), loop playing | map.play | answered:temp | 1 | 3286 | 3286 | 3286 |
| D map national (lower 48), loop playing | tick | tick | 540 | 1 | 1 | 382 |
| E map zoomed in three steps, a pan every 15 s | key | key | 43 | 6 | 93 | 102 |
| E map zoomed in three steps, a pan every 15 s | map.pan.left | answered:feed | 109 | 667 | 11884 | 12963 |
| E map zoomed in three steps, a pan every 15 s | map.pan.left | answered:radar | 39 | 632 | 2366 | 3437 |
| E map zoomed in three steps, a pan every 15 s | map.pan.left | answered:temp | 39 | 709 | 4550 | 6126 |
| E map zoomed in three steps, a pan every 15 s | map.pan.left | complete | 3 | 1453 | 2317 | 2317 |
| E map zoomed in three steps, a pan every 15 s | map.pan.left | m5 | 3 | 1453 | 2317 | 2317 |
| E map zoomed in three steps, a pan every 15 s | map.pan.left | settled | 2 | 4 | 2317 | 2317 |
| E map zoomed in three steps, a pan every 15 s | map.zoom.in | answered:feed | 2 | 2256 | 11673 | 11673 |
| E map zoomed in three steps, a pan every 15 s | map.zoom.in | answered:radar | 2 | 127 | 4755 | 4755 |
| E map zoomed in three steps, a pan every 15 s | map.zoom.in | answered:temp | 2 | 96 | 8583 | 8583 |
| W1 open (warm) | key | key | 2 | 0 | 34 | 34 |
| W1 open (warm) | open | answered:feed | 2 | 1798 | 20160 | 20160 |
| W1 open (warm) | open | answered:radar | 1 | 791 | 791 | 791 |
| W1 open (warm) | open | answered:temp | 1 | 44 | 44 | 44 |
| W1 open (warm) | open | complete | 1 | 1149 | 1149 | 1149 |
| W2 open (warm) | key | key | 2 | 0 | 54 | 54 |
| W2 open (warm) | open | answered:feed | 3 | 5871 | 26075 | 26075 |
| W2 open (warm) | open | answered:radar | 1 | 58 | 58 | 58 |
| W2 open (warm) | open | answered:temp | 1 | 83 | 83 | 83 |
| W3 open (warm) | key | key | 2 | 0 | 33 | 33 |
| W3 open (warm) | open | answered:feed | 3 | 904 | 10722 | 10722 |
| W3 open (warm) | open | answered:radar | 1 | 548 | 548 | 548 |
| W3 open (warm) | open | answered:temp | 1 | 44 | 44 | 44 |
| W4 open (warm) | key | key | 2 | 0 | 34 | 34 |
| W4 open (warm) | open | answered:feed | 2 | 102 | 15906 | 15906 |
| W4 open (warm) | open | answered:radar | 1 | 38 | 38 | 38 |
| W4 open (warm) | open | answered:temp | 1 | 100 | 100 | 100 |
| W5 open (warm) | key | key | 2 | 0 | 32 | 32 |
| W5 open (warm) | open | answered:feed | 3 | 795 | 21155 | 21155 |
| W5 open (warm) | open | answered:radar | 1 | 540 | 540 | 540 |
| W5 open (warm) | open | answered:temp | 1 | 42 | 42 | 42 |
