# Reading a heap profile and a soak log — a walkthrough

For someone who has never opened a `pprof` file or a soak CSV. Every example below is a real record
from this pass (the Q0 24-hour soak of 2026-08-26/27), so you can open the same files and see the same
numbers. Red-team RT-14 / R2-14 asked for this page.

## 1. What the soak produces

`scripts/quality/soak.sh <pid> <hours> soak.csv 300` writes one CSV row every 5 minutes and asks the
process for a **dump** once an hour. A row:

```
utc,elapsed,rss_kb,footprint_kb,threads,pcpu,heap_alloc,heap_inuse,heap_objects,goroutines,fds,disk_files,disk_bytes,publishes_recent
2026-08-27T08:04:22Z,16:08:53,213088,148684,23,1.9,83537056,119939072,844542,274,15,814,38922847,11253
```

| Column | What it is | What to look for |
|---|---|---|
| `rss_kb`, `footprint_kb` | what the OS says the process holds (RSS; on macOS the vmmap physical footprint) | a *fragmentation* check only — it sawtooths with the Go heap's growth and shrink and is not the growth series |
| `threads` | OS threads | must match the bound by construction (soak-profile.md: ≈ 34 here) and be the same at 24/48/72 h |
| `heap_alloc` | bytes of live Go objects **after a garbage collection** (since 0.10.1; before it, the 5-minute samples were pre-GC — see §5) | the growth series |
| `heap_objects` | live objects | tracks `heap_alloc`; a rise in objects with flat bytes means many small things |
| `goroutines`, `fds` | parked goroutines, open files/sockets | flat: 273–280 goroutines (50 RECENT schedulers × 5 tiers + the rest), 15–20 fds |
| `disk_files`, `disk_bytes` | the HTTP disk cache | flat after the start sweep (Q1) |
| `publishes_recent` | RECENT snapshots published since launch | ≈ 35/h since Q3 (it was ≈ 670/h before) |

A **dump** is a directory under the cache dir's `profiles/` named by its UTC time, holding four
profiles (`heap`, `allocs`, `goroutine`, `threadcreate`) and `counters.json` — the same numbers as
the CSV row plus the request counters per host and the gauges (every bounded structure's size).

## 2. The growth statistic

```
$ go run ./tools/slope -in soak.csv
n=11 buckets (203 samples) sigma=3.55 MB
slope=+25.523 MB/day  se=10.157 (NW lag 2, t=2.26)  95% CI [+2.548, +48.499]
plateau=49.8 MB  bar=2.5 MB (30-day growth)
projection(upper)=1455.0 MB  floor=689.3 MB
GROWTH
```

Read it top to bottom:

- **n, sigma** — the first 6 h are warm-up and dropped; the rest is bucketed into hours and each hour's
  *minimum* `heap_alloc` is the series (minima, because a GC can land anywhere in the 5 minutes).
  `sigma` is that series' scatter.
- **slope, CI** — an ordinary least-squares line through the hourly minima, with Newey–West standard
  errors (the samples are correlated in time). The 95 % interval is the honest range of the slope.
- **plateau, bar** — the plateau is the series' median; the bar is 5 % of it: the growth over 30 days
  that would count.
- **projection(upper), floor** — the upper CI edge times 30 days, and the smallest growth this run
  *could* have resolved. When the floor is above the bar the run cannot certify the bar; it says so.
- **Verdict** — `PASS` (upper projection under the bar), `GROWTH` (the whole CI is positive: a term is
  there), `UNCERTIFIABLE` (the CI spans zero but the floor is above the bar: not enough hours),
  `INSUFFICIENT` (fewer than the minimum buckets).

This run said **GROWTH**. That is a claim to *attribute*, not a conclusion. The next two sections show
how.

## 3. Attributing growth with `pprof -base`

Take two dumps hours apart and subtract them: what grew, by site.

```
$ go tool pprof -sample_index=inuse_space -top -nodecount=14 \
    -base profiles/20260826T235743Z/heap.pb.gz profiles/20260827T075921Z/heap.pb.gz
      flat  flat%   sum%        cum   cum%
 4608.72kB 11.36% 11.36%  4608.72kB 11.36%  encoding/xml.copyValue
 3168.47kB  7.81% 19.17%  3168.47kB  7.81%  io.ReadAll
-2519.53kB  6.21% 12.96% -2519.53kB  6.21%  os.readFileContents
 1536.02kB  3.79% 16.75%  1536.02kB  3.79%  …/domains/fire/hms.parseDescription
 1391.96kB  3.43% 20.18%  7536.70kB 18.58%  …/domains/fire/hms.Parse
```

- `-sample_index=inuse_space` is live bytes (`inuse_objects` is live counts; `alloc_space` is
  everything ever allocated — the churn view, not the growth view).
- `-base` subtracts the first profile: positive rows grew, negative rows shrank.
- `flat` is the bytes allocated *at* that line; `cum` includes what it called. Read `cum` on the
  function you recognise: `hms.Parse` accounts for 7.5 MB of the 8-hour growth, and its callees are
  the XML decoder's copies (`xml.copyValue`), the inflated archive (`io.ReadAll`) and the per-point
  strings (`parseDescription`).

Then look at the **gauge** that names that structure in the two `counters.json` files:

```
hms.memo.points   77371 -> 95131
```

The HMS fire archive grew from 77k to 95k detections through the day; the memo that holds the parsed
archive grew with it. Across the hourly dumps the post-GC heap stepped **56.0 → 57.1 → 61.0 MB** exactly
when `hms.memo.points` stepped 77k → 81k → 95k, and was flat between steps. So the "growth" is a
*plateau shift driven by the data*, bounded by the archive — not an unbounded term. Re-running the
statistic with the warm-up moved past the last step:

```
$ go run ./tools/slope -in soak.csv -warmup 9h
slope=+0.444 MB/day  95% CI [-15.011, +15.898]   UNCERTIFIABLE
```

flat, and honest about what one day can resolve. The 7-day run is what certifies the 30-day bar.

What Q3 did to that site, for the record: the archive is no longer inflated into memory
(`io.ReadAll` gone), the XML decoder's struct copies are gone (a hand-decoded token walk), and the
satellite/method strings are shared across points (`parseDescription` now holds one string per
distinct value, not one per point).

## 4. Reading a profile without a base

`go tool pprof -sample_index=inuse_space -top heap.pb.gz` lists what holds memory right now; the
first few rows are the structures the gauges name (the HTTP cache bodies under `doAttempt`, the HMS
memo under `hms.Parse`, the assembler's snapshots under `snapshot.harmonize`). `-sample_index=alloc_space`
on `allocs.pb.gz` is the churn view: the frame renderer and the parsers dominate, which is what the
allocation-rate row of §1 measures and what Q3 cut.

`goroutine.pb.gz` with `-top` shows the parked goroutines by stack; the count must be flat across dumps.
`threadcreate.pb.gz` records `<unknown>` for Go's template thread, which is why threads are bounded
**by construction** (soak-profile.md) rather than by profile.

## 5. Pitfalls this pass hit, so you do not

- **Pre-GC samples.** Until 0.10.1, `/debug/counters` read the memory rows without a GC; the hourly
  dumps did GC. The CSV showed 75–98 MB where the dumps showed 57–61 MB, and the slope statistic ran on
  the noise. The hourly dumps were always the truth; from 0.10.1 both read after a GC.
- **A shared cache directory.** Two soaks on one machine share `~/Library/Caches/watchpost/http`;
  the second one's HMS/WFIGS rows then measure the disk tier, not the network. Run one at a time or
  read the shared rows with that in mind.
- **The launch hour.** Request counts "in the hour since launch" include the launch burst; steady-state
  per-hour numbers come from `counters.json` deltas between two dumps.
- **Data-driven plateaus.** A fire-season archive, a storm's alerts, a new location — every one moves the
  live heap by a bounded amount. Read the statistic next to the gauges before calling it a leak.
