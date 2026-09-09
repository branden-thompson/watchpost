# Performance measurement — the standing method, and what we deliberately did NOT build

**Repo-level and release-spanning.** `perf-protocol.md` lives inside a feature folder and records one
release's measurements; this file records how measurement is DONE here and which decisions have
already been made, so a new session does not re-litigate them.

## The ruling (HUM LEAD, 2026-09-09): no performance-toolset project before 0.16.0

A three-tier profiling toolset — short, mid and long term — was proposed and **declined after
rubber-ducking, because two of the three tiers already exist and the third has no reported need.**
Recorded here rather than dropped, because "we considered this and chose not to" is worth more to the
next session than silence, and the reasons are the part that will still be true later.

**What already exists, verified 2026-09-09 rather than assumed:**

| Tier | Status | How |
|---|---|---|
| Short term (1–3 h, frequent samples) | **Exists — as arguments** | `scripts/quality/soak.sh <pid> <hours> <out.csv> [interval_s]` already takes both. 3 h at 30 s is `soak.sh <pid> 3 out.csv 30` |
| Mid term (8 h+) | **Exists** | The same script; `tools/slope` rules on the series |
| Trend analysis at SHORT horizons | **Exists** | `slope` has `-window` and `-warmup`. The "8 buckets" floor is STATISTICAL, not a duration — 3 h at `-window 20m` clears it |
| Long term (days–weeks, live stations) | **Not built, deliberately** | See below |

`soak.sh` already records 14 columns per sample — `rss_kb`, `footprint_kb`, `threads`, `pcpu`,
`heap_alloc`, `heap_inuse`, `heap_objects`, `goroutines`, `fds`, the disk gauges and
`publishes_recent` — and `/debug/dump` carries `runtime.ReadMemStats` including `NumGC` on demand.
**There is no instrument the standard Go toolchain offers that this repo is not already using.**

## The real gap, which is not a tool

**The failure that started this was never an inability to measure.** An hour of sampling with existing
tools gave a clean answer. The failure was that the 0.13.0 baseline used **50 locations with the radio
off** and the observation used **10 with it on**, so a good number could not be compared to a good
number.

**No tool fixes that; a defined workload does.** The missing thing is a versioned standard workload —
one fixture config, a stated terminal size, radio-off and radio-on variants — plus one command that
runs against it and emits a comparable record. **Estimated at an afternoon, and it is the only piece
of this that 0.16.0 actually needs**, because a Broadcaster-UI "after" number is worthless without a
"before" taken on the same workload.

## Why the long-term tier waits

Days-to-weeks on live stations is **not a soak harness — it is telemetry from someone else's machine.**
That drags in what is collected, where it is stored, whether it ever leaves the box, and how a user
hands it back: product and privacy decisions for an app whose whole posture is local-first.
Off-the-shelf continuous profiling (Parca, Pyroscope) is the do-not-reinvent answer **if** it is ever
wanted, and attaching a profiling agent to a personal weather station is a heavy thing to carry for a
question nobody has asked. **Build it when there is a report, and let the report choose the shape.**

## Why not before 0.16.0

- **It would tune instruments for a surface not yet visible.** Broadcaster UI is a RENDERING feature;
  its plausible costs are frame time and allocations per frame, already pinned by
  `modes/tty/bench_test.go` and `make alloc-budget` — not by RSS sampling. A memory-focused toolset
  built now may instrument the wrong axis.
- **SEV-0 process buys nothing here.** Severity should match blast radius, and this one's is a CSV.
- **New tooling owes new controls.** After 0.15.0, an instrument is not believed until it has been
  watched failing (INST-1..5) — the right price for a gate, a poor one for something used twice
  before 0.16.0 changes the question.

## What to do instead — hours, not a project

1. **Run the 8-hour soak overnight.** Free; settles the RSS-vs-footprint divergence below and gives
   `slope` a series it will actually rule on.
2. **Define the standard workload as a fixture** and take the 0.15.0 baseline against it, radio-off and
   radio-on.
3. **Record both in `perf-protocol.md`** as the numbers 0.16.0 is compared against.

## What would reopen the toolset decision

- The overnight run showing a real upward trend in **physical footprint** — not RSS; see below.
- The Broadcaster UI plan adding a **background writer or a second render path**, which would put
  measurement on the critical path.

Either makes the question specific, which is what the three-tier proposal lacked.

## The 0.15.0 macOS measurement (2026-09-09)

Live instance, **10 locations, radio configured**, 12 samples over an hour at 5-minute spacing, from
13 → 68 min uptime. Series: `02_features/0.15.0-pre-broadcaster-ui-improvements/07-readiness/soak-macos-1h.csv`.

| Measure | Range | Median | Trend | Verdict |
|---|---|---|---|---|
| RSS | 106.8 – **115.3** MB | 109.6 | +6.31 MB/h — **an artifact, see below** | **PASS** vs the ≤126 MB gate, 10.7 MB margin |
| Physical footprint | 93.7 – 100.2 MB | 96.2 | **−0.28 MB/h** | flat |
| Threads | **26, flat** | | none | better than 0.13.0 (26 → 32 after 25 min) |

**The ~127 MB impression did not reproduce**; the highest single sample was 115.3 MB and the run ended
at 110.1 MB.

**RSS and footprint disagree, and the disagreement is the finding.** Physical footprint is what macOS
charges a process — dirty private pages plus compressed memory. RSS also counts clean, file-backed,
reclaimable pages. RSS rising while footprint does not is the signature of mapped-file residency, not
a growing heap. The series shows it: ~109–110 MB for seven samples, a 115.3 MB shelf for three around
48–58 min, then **back down** to 110.1. A least-squares line through twelve oscillating points always
returns a slope; that one's +151 MB/day projection is an artifact of fitting a line to a plateau.

**`tools/slope` refused the series** — `only 0 buckets after warm-up; need at least 8` — which is the
correct behaviour and the honest verdict: **no leak is demonstrated and none is excluded.**

**Two limits, stated so the numbers are not over-quoted later.** The workload is not the 0.13.0
baseline's, so this is a fresh self-baseline for real usage rather than a 0.13.0 → 0.15.0 comparison.
And no launch burst was captured — the first sample was at 13 min uptime, so the baseline's "116 MB at
launch" row has no counterpart.

## Cross-platform comparisons: read the column, not the number

`soak.sh` deliberately records **different quantities per platform** — `vmmap` **physical footprint**
on Darwin, **Pss** from `smaps_rollup` on Linux. Pss divides shared pages by their sharer count;
footprint counts compressed memory and dirty private pages; btop's `MemB` is closer to RSS again.
**Three definitions.** The HUM LEAD's standing observation that Arch numbers read consistently lower
than macOS is therefore partly a measurement-definition artifact, not only a platform difference.
**`rss_kb` is the one column defined the same on both** — compare that, or compare nothing.
