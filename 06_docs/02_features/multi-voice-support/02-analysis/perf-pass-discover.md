# Performance pass — DISCOVER (0.14.0 pre-BUILD-EXIT, 2026-08-30)

Measured, not inferred. One instrumented run of `dist/watchpost` at `24883ab-dirty` on darwin/arm64
(Apple M3 Pro), 133×44 pty, 10 favourites + 50 RECENT, radio off, no modal open:
`WATCHPOST_DEBUG_PPROF=1 WATCHPOST_DEBUG_PPROF_ADDR=127.0.0.1:6071`, sampled every 30 s from
`/debug/counters`, with paired heap profiles three minutes apart for the steady-state allocation split.
Benchmarks: `go test ./modes/tty ./platform/snapshot ./domains/fire/hms ./platform/render -bench . -benchmem -count 5`.

Prior art read first and **not** re-litigated: `watchpost-performance-quality-pass/` (L3 structure lens,
L4 caching lens, the quality-pass plan, ADR-03 RECENT scheduler, `quality-baseline.md`) and this
release's `07-readiness/perf-protocol.md`.

## 1. Steady state — no leak, two numbers off their baseline

| Gauge | Measured (9.6 min, flat) | Recorded baseline | Verdict |
|---|---|---|---|
| Post-GC heap | 35–37 MB, no trend | — | flat |
| Goroutines | 330–332, flat | 273–280 (`quality-baseline` §5) | **+50: the 6th RECENT tier** |
| `stack_inuse` | 3.80–4.03 MB | ADR-03 re-open threshold: **> 5 MB live** | under, with ~0.65 MB of headroom |
| Threads | 24–25 (runtime) / 28–29 (`ps -M`) | bound by construction ≈ 34 | in band |
| FDs | 17–20 | — | flat |
| RSS | **128–132 MB** | 0.13.0 plateau 86–115 MB; protocol §4 pass line **≤ 126 MB** | **over the line** |
| Allocation rate | **≈ 31 MB/min** (sampler artifact removed) | ≈ 9 MB/min idle, "render's share 0" | **≈ 3.4×** |

The sampler artifact: `app.snapshotBytes` marshals the whole snapshot to size it, and
`/debug/counters` is what calls it — 17.3 MB of the raw 110.9 MB/3 min was my own 30-second polling,
not the app. The 31 MB/min figure has it removed.

## 2. Why the allocation rate moved: the ticker made the frame permanent

`modes/tty/dashboard.go:412` — `case len(d.ticker) > 0: return true` (0.12.0, the marquee scrolls
continuously). There is essentially always a national severe event, so **`tickNeeded()` is now
permanently true and a full frame is drawn every 300 ms, forever.** The recorded "idle ≈ 9 MB/min,
render's share 0 (no frame drawn)" baseline describes a state the app no longer enters.

It reconciles exactly: `BenchmarkFrame_133x44` memo hit is 103,626 B/op; 3.33 frames/s × 103.6 KB =
**20.7 MB/min**, and the remaining ~10 MB/min is the data side — which is what the old baseline
measured on its own.

This is not a defect (the marquee has to animate) but it changes the economics the earlier pass
priced: **every per-frame cost is now a 24/7 cost**, which is precisely the days-to-months runtime and
the future broadcaster mode.

## 3. The per-frame allocation split (3-minute paired diff, steady state)

| Site | MB/3 min | Share | Note |
|---|---|---|---|
| `strings.(*Builder).grow` → `MakeNoZero` | 20.79 | 18.8 % | diffuse string building (regexp replace + render) |
| **`tty.Dashboard.recentLocations`** | **18.47** | **16.7 %** | see 3.1 |
| `regexp.ReplaceAllString` (cum) | 9.50 | 8.6 % | `plaintext.StripSGR`, see 3.2 |
| `render.Opts.BoxTitled` (cum) | 10.50 | 9.5 % | frame chrome |
| `tty.Dashboard.header` (cum) | 8.00 | 7.2 % | frame chrome |
| `snapshot.harmonize` | 5.03 | 4.5 % | publish path |

### 3.1 `numRecent()` builds the list to take its length

`modes/tty/nav.go:116` — `func (d Dashboard) numRecent() int { return len(d.recentLocations()) }`, and
`recentLocations` copies all 50 `snapshot.Location` values into a fresh slice. Called from three
per-frame sites (`dashboard.go:632`, `layout.go:68`, `nav.go:33`). **6.2 MB/min — a fifth of the
app's entire allocation — to compute an integer.**

### 3.2 `render.Width` runs a regexp on every measurement

`platform/render/text.go:245` → `stripANSI` → `plaintext.StripSGR` → `sgrRe.ReplaceAllString`.
`Regexp.replaceAll` allocates a fresh string **even when there is no match**, and `Width`/`PadTo` are
called for every cell of every row of every frame. ≈ 3.2 MB/min plus its share of the builder growth.

Flagged in the L4 lens as item 26 ("666 ns / 4 allocs, hundreds per frame — INFO, render lens") and
never actioned. The vendored go-studs `rendering.DisplayWidth` has the same shape.

### 3.3 `render.Overlay` is 7.6× the entire rest of the frame

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| `Frame_133x44` (memo hit, no modal) | 137,195 | 103,626 | 581 |
| **`OverlayOnly`** | **1,042,286** | **765,448** | **1,346** |
| `Frame_133x44_Severe` | 1,217,284 | 891,286 | 1,946 |
| `Frame_133x44_Help` | 1,110,058 | 917,013 | 2,358 |
| `Setup_133x44` | 1,162,517 | 941,938 | 2,792 |

`platform/render/panel.go:292` composites through `lipgloss.NewCompositor`, which builds a full-screen
cell buffer and re-renders it. The modal itself is cheap; the compositor is 86 % of a modal frame.
Consistent with the earlier finding that a *smaller* modal allocates *more* — the cost is the area the
modal does **not** cover.

## 4. Growth curve for 0.15.0

1. **Goroutines are O(locations × tiers).** 309 of 331 are `sched.(*Scheduler).wait`:
   50 RECENT × 6 tiers + 8 priority + 1 batched-alerts. ADR-03 chose option A knowingly and set the
   re-open threshold at **> 5 MB live**; `stack_inuse` is 3.8–4.0 MB. Each new 0.15.0 feed that earns a
   RECENT tier adds ~50 goroutines / ~0.65 MB. **The documented trigger is about two feeds away.**
2. **`sched.cycle` fans out to providers sequentially** (`sched.go:226-247`): a tier's wall time is the
   *sum* of its providers' latencies, and the last provider's data is that much staler. Linear in the
   number of APIs — the exact axis 0.15.0 grows on.
3. **Disk re-reads scale with eviction pressure.** `cache.stale()` reads the body from disk whenever a
   conditional GET's entry has been evicted from the 8 MB memory tier. Launch shows 64 MB of
   `os.ReadFile` in 33 s; steady state is 0.75 MB/3 min, so this is a *launch* cost today — but more
   feeds mean more eviction, which moves it toward steady state. (L4-F3 fixed the adjacent case where
   memory still holds a stale entry.)

## 5. API budget — healthy; the launch burst is the whole cost

Steady state over 9.6 minutes:

| Host | attempts | net | cache | 304 | net MB | saved by 304 |
|---|---|---|---|---|---|---|
| api.weather.gov | 255 | 140 | 249 | **106** | 1.44 | **12.79 MB** |
| earthquake.usgs.gov | 79 | 79 | 51 | 0 | 0.15 | — |
| firms.modaps.eosdis.nasa.gov | 53 | 53 | 45 | 0 | 0.01 | — |
| www.ndbc.noaa.gov | 4 | 4 | 62 | 0 | 0.20 | — |

Conditional revalidation is doing the heavy lifting exactly as Q5 designed: NWS spent 1.44 MB of
network to avoid 12.79 MB. USGS and FIRMS show no 304s because FDSN and FIRMS send no validators —
their 79 and 53 requests are almost entirely the launch fan-out (+2 and +0 respectively after the
first 33 seconds), and USGS's near-field query is deliberately un-snapped (`usgs.go:113-125`) so one
URL per location is by design.

No action proposed here; the levers for 0.15.0 are recorded, not taken.

## 6. Deliberately not re-opened

ADR-03 (RECENT scheduler = A) except against its own stated threshold; ADR-05 (`SetMemoryLimit`);
the Q3/Q4a/Q5/Q6 accepted non-decisions in `quality-baseline.md` §6; the HMS byte scanner; the
`layoutFor` package memo; `Tok()`-time precompute.

---

# BUILD — what was changed, and what it bought

HUM LEAD ruling 2026-08-30: **"frame trio + growth re-check"** — the three per-frame costs, then
re-measure against ADR-03's own threshold. No architecture touched. SACRED: no approved behaviour may
regress.

## 7. The two that landed

### 7.1 `numRecent()` is arithmetic (`modes/tty/nav.go`)

It counts the list `recentLocations()` draws instead of building it.
`TestNumRecentAgreesWithTheDrawnList` holds the two together across seven states — nothing, an empty
snapshot, rows without a lookup, a lookup with no snapshot, a lookup not yet in the list, a lookup
already in it, and a lookup into an empty list — because the focus arithmetic spans both tables and a
count that disagrees with the drawn list puts the cursor off the end of it.

### 7.2 `StripSGR` is a scanner with a no-escape fast path (`platform/plaintext/plaintext.go`)

`\x1b\[[0-9;]*m` by hand, returning the **input string itself** when there is no ESC byte — which is
almost every cell measured. `TestStripSGRMatchesTheRegexpItReplaced` rebuilds the original regexp *in
the test* and asserts byte equality over a hand-built corpus (truncated sequences, OSC hyperlinks,
other CSI final bytes, empty parameters, wide runes) **and 20,000 random strings** drawn from the
alphabet that can form and malform a sequence. `TestStripSGRDoesNotCopyPlainText` pins the fast path at
zero allocations. The package had no test file before this.

`platform/plaintext` was the only test-free package in the tree (L3-F27's neighbour).

### 7.3 Benchmarks (`-count=5`, medians, darwin/arm64 M3 Pro)

| Benchmark | ns before | ns after | | B before | B after | | allocs before | after |
|---|---|---|---|---|---|---|---|---|
| **`Frame_133x44`** (memo hit — the frame that now runs 24/7) | 138,330 | **67,736** | −51.0 % | 103,626 | **67,869** | −34.5 % | 581 | **283** |
| `Frame_133x44_Miss` | 518,355 | 342,741 | −33.9 % | 398,788 | 347,025 | −13.0 % | 7,032 | 5,580 |
| `Frame_133x70` | 170,819 | 97,343 | −43.0 % | 133,021 | 94,139 | −29.2 % | 631 | 301 |
| `Frame_200x60` | 200,225 | 125,173 | −37.5 % | 169,092 | 123,287 | −27.1 % | 640 | 304 |
| `Setup_133x44_Miss` | 1,493,039 | 1,276,369 | −14.5 % | 1,119,906 | 1,019,971 | −8.9 % | 5,570 | 3,666 |
| `Frame_133x44_Severe` | 1,254,534 | 1,123,558 | −10.4 % | 892,907 | 839,015 | −6.0 % | 1,947 | 1,641 |
| `OverlayOnly` | 1,055,138 | 1,050,567 | −0.4 % | 764,795 | 764,976 | +0.0 % | 1,346 | 1,346 |

`OverlayOnly` is flat because it was not touched — see §9.

### 7.4 Live, same protocol as §1 (paired heap profiles four minutes apart, steady state)

| Gauge | Before | After | |
|---|---|---|---|
| Allocation rate | ≈ 31 MB/min | **≈ 23.6 MB/min** | −24 % |
| RSS | 128–132 MB | **105–106 MB** | back **under** the protocol §4 line of ≤ 126 MB |
| Post-GC heap | 35–37 MB | 35.8 MB | flat, unchanged |
| Goroutines · threads · FDs | 331 · 24 · 17–20 | 331 · 24 · 15–18 | unchanged |

Both RSS readings are flat bands across their run's samples rather than single points, and the after
run was sampled from 4 to 8 minutes of uptime against the before run's 0.5 to 25 — not a matched pair
on uptime, but the before run sat at 127–132 MB from its first sample, so the gap is not warm-up.

`recentLocations` and `regexp` no longer appear in the steady-state top ten at all. What is left at the
top is `strings.Builder` growth (the frame's own assembly) and ultraviolet's renderer diff — both
inherent.

### 7.5 Allocation pins re-pinned (the P4 Task 4.11 the protocol asks for)

`frameAllocBudgetHit` was a flat 6,000 — **20× the measurement**, so it had stopped measuring anything.
Every pin is now ×1.05 of the post-change number, with the old value kept in the comment:

| Pin | Was | Now |
|---|---|---|
| frame hit 133×44 · 133×70 · 200×60 · 80×24 | 6,000 · 6,000 · 6,000 · 1,010 | **297 · 316 · 319 · 311** |
| frame miss 133×44 · 133×70 · 200×60 · 80×24 | 10,546 · 16,316 · 21,033 · 3,778 | **5,857 · 12,019 · 15,160 · 2,267** |
| setup hit/miss 133×44 · 80×24 | 2,969/5,823 · 1,923/4,708 | **2,600/3,838 · 1,587/2,715** |
| severe hit/miss | 2,521/7,939 | **1,709/5,394** |

## 8. Growth re-check against ADR-03's own threshold

ADR-03 chose option A (one scheduler per RECENT location) and set the re-open trigger at **> 5 MB
live** attributable to the schedulers, plus publish-coalescing failures. Measured now:

- **309 of 329 goroutines** are `sched.(*Scheduler).wait` — 50 RECENT × 6 tiers + 8 priority + 1
  batched alerts, exactly.
- **`stack_inuse` 2.85–4.03 MB** across runs (8.8–12.5 KB per goroutine).
- **Publish coalescing is healthy**: RECENT 37 publishes in 9.6 minutes with 469 folded, all of the
  folding at launch — in the 33–37/h band the Q3 consolidation set.

**Verdict: the threshold has NOT tripped. A stays.** Headroom, stated so 0.15.0 DISCOVER can spend it
knowingly: each new feed that earns a RECENT tier adds 50 goroutines ≈ **0.43–0.61 MB**, so there is
room for **1.6 tiers at the high-water reading and 5 at the low**. The honest planning number is the
high-water one. **0.15.0 should re-take this measurement before it adds its second feed**, not after.

Two growth terms 0.15.0 should carry into DISCOVER, neither actioned here:

1. **`sched.cycle` fans out to providers sequentially** (`sched.go:226-247`). A tier's wall time is the
   *sum* of its providers' latencies and the last provider's data is that much staler — linear in the
   number of APIs, which is the axis 0.15.0 grows on. Making it concurrent changes publish order and
   interleaving, so it needs ADR-03's own bar: equivalence on publish times over ≥ 3 simulated hours
   with injected failures.
2. **`cache.stale()` reads the body from disk** for any conditional GET whose entry has been evicted
   from the 8 MB memory tier. Today that is a *launch* cost (64 MB of `os.ReadFile` in the first 33 s;
   0.75 MB per 3 min in steady state), but more feeds mean more eviction pressure, which moves it
   toward steady state.

## 9. `render.Overlay` — measured, understood, NOT changed, and why

`panel.go` composites through `lipgloss.NewCompositor`, at **1,042 µs / 765 KB / 1,346 allocs** — 86 %
of a modal frame, and 7.6× the cost of the entire rest of a frame. The obvious fix is to splice the
modal's rows into the base rather than re-composite the screen. **It cannot be made byte-identical, so
it was not done.** Two behaviours were found by probing the real function:

1. **It re-emits SGR canonically** — a base carrying `\x1b[0m` comes back carrying `\x1b[m`. Matching
   that means reimplementing lipgloss's style parser and emitter, and one new SGR form anywhere in the
   app would silently change rendering.
2. **It pads every line to the widest line's display width**, which is a property of the *whole* base,
   so a partial composite of only the modal's row span pads to a different width than the full one.

A memo was considered and rejected on measurement, not taste: since 0.12.0 `advanceTicker` steps the
marquee **one cell per tick** (`dashboard.go:607`), so the base string changes on every frame and a
memo keyed on (base, modal, width) would miss every time.

The cost is also bounded in a way the frame cost is not: it is paid only while a modal is open, which
is seconds at a time, where the frame cost is paid for as long as the app is up. **Recorded as the
largest remaining single item in the render path**, for a release that can afford a byte-equality
corpus against the compositor, or an upstream conversation with lipgloss.
