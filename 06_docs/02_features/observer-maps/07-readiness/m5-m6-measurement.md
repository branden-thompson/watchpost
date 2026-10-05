# M5 and M6 — measured in VALIDATE (D-251)

M5 is measured by D-46's protocol: cold, 149×38, p90 of 20 opens, over recorded responses. The
built binary cannot run that protocol offline, so it is measured in process, as D-280 rules. M6 is
measured as D-53 defines it: the intervals between delivered map ticks and key-to-response latency,
with the Observer live. Both read the dashboard's own timing instrument (`modes/tty/timing.go`,
`WATCHPOST_DEBUG_TIMING=1`). Nothing below is asserted in a gate, and nothing here recommends a
threshold.

## Conditions

| | |
|---|---|
| Date | 2026-10-05, 18:47–19:00 UTC (binary run), 19:00 UTC (in-process M6 run) |
| Tree | `feature/map-drawing` at `8c538828`, plus the measurement tests listed under "Reproduce" |
| Machine | Mac17,8, Apple M5 Pro, 18 CPUs, 64 GiB |
| OS | macOS 26.7 (Darwin 25.6.0) |
| Go | go1.27.1 |
| Load average (1, 5, 15 min) | binary run: 3.88 4.03 4.79 before, 4.37 4.22 4.45 after; in-process M6: 5.04 4.36 4.50 before, 4.55 4.35 4.48 after |
| Other load | fseventsd and Microsoft Defender's scanner near a core each, a browser, and the listener's own `./dist/watchpost` |
| Terminal | a pseudo-terminal at 149 columns × 38 rows |
| Network | none: the journey's guards (`cmd/watchpost/journey_test.go`). Every proxy variable names a local proxy that refuses with 403, and the binary runs under `sandbox-exec` with outbound connections denied except to loopback, which a probe proves first. Requests to Open-Meteo hosts reached the refusing proxy and went no further |

## M5 — time to picture

### In process (D-280)

**Result: 20 of 20 opens reached m5. The p90 is 67.3 ms, against the 3.5 s target.**

| | |
|---|---|
| Date and load | 2026-10-05, 19:12 UTC; load average 3.22 3.88 4.46 just after the run; same machine, OS and Go as below |
| Opens | 20 at 149×38, each with a new station and empty caches |
| m5, each open (ms) | 45.0, 26.7, 40.0, 42.4, 40.1, 164.2, 244.9, 44.4, 36.5, 33.9, 32.7, 36.8, 41.3, 40.1, 42.1, 36.3, 36.3, 67.3, 29.3, 39.2 |
| p50 / p90 / max | **40.0 ms / 67.3 ms / 244.9 ms** |
| Target (D-46) | ≤ 3.5 s, p90 |

**Method.** `app/m5_measure_test.go` runs each open as its own station:

- The real tty Dashboard runs under its Router in a Bubble Tea program at 149×38, wired by the app's
  own `mapConfig` to the app's feed (`livePipelines.mapFeed`, both lanes, D-268).
- Every cache starts empty: each open gets a new map builder over an empty tile directory, a new zone
  store and a new NWS provider.
- The recorded answers come from a local server. M1 scenario 01's Oak Ridge Flash Flood Warning, its
  times moved to now, answers every active-alerts ask. The place's `/points` and its stations are
  served too. Each open asked `/alerts/active?status=actual&area=NC,TN` once and nothing else.
- The basemap comes from memory through the map builder's existing transport parameter, with the
  `recorded` transport from `maps_basemap_test.go`. It serves the recorded OpenFreeMap TileJSON and
  an embedded tile's bytes for every tile asked. Each open asked one TileJSON (two in opens 6 and 7)
  and two tiles.
- m5 is read from the app's timing keeper, from `g` to the first complete frame holding every
  alert's area.

No production code changed.

**Blind spots of this method:**

- **The process.** The binary's other pipelines, radio and launch burst are not running, and the
  radar and temperature have no sources, so they answer at once.
- **The terminal.** The renderer writes to nowhere, so no terminal write is measured.
- **The tiles.** The tiles are stand-ins: an embedded tile's bytes decoded as any tile, not
  OpenFreeMap's real tiles at that zoom, which carry more to decode and draw.
- **The network.** Neither the network nor a first-ever zone fetch is measured. This alert carries
  its own polygon.

### The built binary, offline

**Result: no m5 in any of the 20 recorded-response opens.** The binary cannot produce this number on
the recordings the repository holds.

| | |
|---|---|
| Opens | 20, each a new process under a new HOME (zone and map caches empty) |
| Opens reaching m5 within 30 s of `g` | **0 of 20** |
| p50 / p90 / max | none: no samples |
| Target (D-46) | ≤ 3.5 s, p90 |
| Live first-ever cold opens (W14, batch 61, n = 5 each) | `default` 7.0 s (6.9–7.2), `heavy` 9.4 s (7.3–10.1) |

**Why no open reached m5.** m5 is the first frame that is complete: every alert's area and the
basemap. At the place, the basemap needs OpenFreeMap tiles above the embedded tiles' zoom 3, and two
things keep them away offline:

1. **No recording exists.** The repository records OpenFreeMap's TileJSON
   (`app/testdata/maps/openfreemap-tilejson.json`) but no tiles for Oak Ridge or for any M1 scene. The
   go-tuiMaps fixtures (Gulf z6, Midwest z5, four urban z14 tiles) do not cover this view either.
2. **There is nowhere to put one.** The library fetches the TileJSON and the tiles itself, over
   HTTPS, through its own transport. It reads nothing from the station's HTTP cache, which is where
   the journey's recorded answers live, and its own tile cache on disk holds tiles, never the
   TileJSON. Every open therefore asked `tiles.openfreemap.org` (44 refusals over the run), and the
   frame never became complete.

The radar has the same limit: the radar client keeps nothing on disk and dials only public HTTPS
hosts, so radar is refused too. That does not block m5, which never waits for radar.

**What each open did say.** Times from `g`, n = 20:

| Event | p50 | p90 | max |
|---|---|---|---|
| `answered:feed`: the alerts in view, drawn | 11.8 ms | 13.6 ms | 14.9 ms |
| `answered:radar`: refused | 977 ms | 1,250 ms | 1,410 ms |
| `answered:layers`: the other layers, refused | 1,899 ms | 2,241 ms | 2,616 ms |
| `answered:temp`: refused | 3,877 ms | 4,408 ms | 4,682 ms |
| `complete`, `m5`, `settled` | not said | | |

Each open's seconds to the alerts being drawn: 0.011, 0.010, 0.011, 0.013, 0.015, 0.014, 0.011,
0.014, 0.012, 0.013, 0.012, 0.011, 0.012, 0.012, 0.011, 0.013, 0.011, 0.012, 0.012, 0.013.

**The recorded alert.** The answers are the journey's (the place's `/points`, its stations, and M1
scenario 01's Oak Ridge Flash Flood Warning with its times moved to now). The measurement adds one
more: the same recorded alert as the answer to the map's "Alerts in view" ask at 149×38,
`/alerts/active?status=actual&area=NC,TN`. The alert carries its own polygon, so no zone is fetched.

**The blind spot.** The live 7.0–9.4 s figure measures what a recorded run cannot: the network, the
first-ever fetch of the alerts' zones (4.0–4.9 s of a cold `default` open, batch 62), and the tiles'
round trips. A recorded run, once it can complete, measures the station and the library with those
removed. Measured here, the alerts part of that is about 12 ms.

**What D-280 ruled.** The HUM LEAD chose the in-process method above. The binary's offline result
stays here as a blind spot: recording the tiles and adding a seam to serve them to the binary were
not chosen.

## M6 — loop smoothness

The built binary on recorded responses has no radar to play, so M6 was taken in two places.

### A. In process, with the loop playing (`modes/tty/m6_measure_test.go`)

The Router and its dashboard run in a real Bubble Tea program at 149×38. The renderer writes to
nowhere, the map uses the embedded basemap, and the radar answers with twelve recorded IEM frames
over the lower 48 (`radarFeed`). The Observer is live the way a publisher keeps it: the test snapshot
is sent once a second, and the dashboard's own clocks run. The sequence is `g`, then `1` (the lower
48, settled after 0.61 s), then space to play. The loop played for 60 s, with →, ←, →, ←, → pressed
ten seconds apart.

| Measure | n | p50 | p90 | p99 | max |
|---|---|---|---|---|---|
| Interval between delivered map ticks | 55 | 1,000.1 ms | 1,000.9 ms | 1,999.5 ms | 1,999.5 ms |
| Lateness of each tick against the moment the library asked for | 56 | 0.8 ms | 1.1 ms | 1.2 ms | 1.2 ms |
| Key to its frame (space and the five pans) | 6 | 2.6 ms | 5.4 ms | 5.4 ms | 5.4 ms |
| Time for 100 frames drawn | 3 | 21,368 ms | 21,998 ms | 21,998 ms | 21,998 ms |

Four of the 55 intervals are about 2,000 ms. Every other interval is between 999.1 ms and 1,000.9 ms.

**Why some intervals are 2 s: the library's pace, not a dropped frame.** go-tuiMaps v0.2.0 plays a
loop one frame a step and holds the last frame for at least two seconds before it repeats (D-76).
`playback.go` sets `lastHold = 2 * time.Second`, and `nextAdvanceLocked` asks for the next call at
`run + hold` once the run has played. That call comes 2 s after the last frame's.

The evidence, from a second 60 s run that logs what happened between two ticks:

- The 2 s intervals ended at 13.0 s, 26.0 s, 39.0 s and 52.0 s. That is one per 13 s: twelve frames
  make 11 one-second steps, then the 2 s hold.
- Nothing was said between those ticks, except a pan's events once and a hundred-frame count once.
  The pans fell at 10, 20, 30, 40 and 50 s, so they do not line up with the gaps.
- No tick landed late: the worst lateness was 1.2 ms.

So no frame was dropped. The library asked for its next call 2 s ahead, and the tick arrived on time.

### B. The built binary, with the Observer live and no radar

This was the last of the 20 opens above, held at the place for 60 s after space, with →, ←, →, ←, →
pressed ten seconds apart. The station's publishers ran on recorded and refused answers. With no
radar, no loop played: the map asked for one tick.

| Measure | n | p50 | p90 | p99 | max |
|---|---|---|---|---|---|
| Interval between delivered map ticks | 0 | — | — | — | — |
| Lateness of each tick | 1 | 1.1 ms | 1.1 ms | 1.1 ms | 1.1 ms |
| Key to its frame | 6 | 6.3 ms | 6.9 ms | 6.9 ms | 6.9 ms |
| Time for 100 frames drawn | 7 | 9,315 ms | 9,616 ms | 9,616 ms | 9,616 ms |

"Key to its frame" is the instrument's own: from the key's arrival in the Router to the next `View`,
not to the terminal's write.

## Reproduce

```sh
# M5 (20 cold opens) and M6 on the built binary; about 11 minutes
WATCHPOST_VALIDATE_M5=1 WATCHPOST_VALIDATE_OUT=/tmp/m5m6.json \
  go test -run '^TestMeasureM5M6$' -count=1 -v -timeout 30m ./cmd/watchpost

# M5 in process (D-280), 20 cold opens; about 12 seconds
WATCHPOST_VALIDATE_M5=1 WATCHPOST_VALIDATE_M5_OUT=/tmp/m5.txt \
  go test -run '^TestMeasureM5InProcess$' -count=1 -v -timeout 20m ./app

# M6 in process with the loop playing; about a minute
WATCHPOST_VALIDATE_M5=1 WATCHPOST_VALIDATE_M6_OUT=/tmp/m6.txt \
  go test -run '^TestMeasureM6InProcess$' -count=1 -v -timeout 10m ./modes/tty
```

Optional settings: `WATCHPOST_VALIDATE_OPENS` sets the number of opens for a trial run (20 by
default), and `WATCHPOST_VALIDATE_M6_SECS` sets how long the loop plays (60 by default). Without
`WATCHPOST_VALIDATE_M5=1`, both tests skip, so `make verify` and CI never run them. Do not run either
test while `make verify` runs.

The tests are `app/m5_measure_test.go`, `cmd/watchpost/measure_test.go` (on the journey's harness:
`startJourneySized`, `journeyHome`, the refusing proxy and the sandbox) and
`modes/tty/m6_measure_test.go`.

## The verdicts

- **M5: met, by the in-process protocol (D-280).** p90 67.3 ms against 3.5 s. Its blind spots are above: the network, the first-ever zone fetch, the process and the terminal. The live first-ever open (7.0-9.4 s) is what a listener meets on a first run, and is not what this number says.
- **M6: met, against the HUM LEAD's thresholds (D-281).**
  - A tick no more than 100 ms late at p99: measured 1.2 ms.
  - Every interval the step within 10%, the library's 2 s last-frame hold excepted: measured 1000.1-1000.9 ms, and the four 2 s holds.
  - Key to frame within 100 ms at p90: measured 5.4 ms in process, the binary's max 6.9 ms.

  M6 is recorded at SHIP, never asserted in the gate (D-53).
