# Accepted costs

Things that look like performance bugs and are not. Each one was measured, priced, and **chosen** —
usually to protect something the listener experiences. This page exists so nobody re-derives the
finding, re-proposes the "fix", and spends a release undoing a decision.

Read this before optimising anything in the render path, the scheduler, or the HTTP client.

**Every entry states what would re-open it.** That is the only route back: not a fresh opinion, not a
profile screenshot, but the stated trigger, measured. If the trigger has tripped, say so with the
number and re-open the decision properly. If it has not, the entry stands.

Every site named below, in package/file.go:Symbol form, is checked by `cmd/watchpost`'s
`TestAcceptedCostsNamesRealSymbols`,
so this page cannot drift from the code. The sites also carry a pointer back here in their own
comments — you should meet this page while editing, not only while reading docs.

---

## 1. `platform/render/panel.go:Overlay` composites through lipgloss — we do not reimplement it

**Cost.** 1,042 µs · 765 KB · 1,346 allocations per frame with a window open — about 86 % of a modal
frame, and 7.6× the cost of an entire frame without one. `lipgloss.NewCompositor` builds a
full-screen cell buffer and re-renders it, so the cost is the area the window does **not** cover: a
*smaller* window allocates *more*.

**Why it stands.** *(HUM LEAD ruling, 2026-08-30.)* The obvious fix is to splice the window's rows
into the base ourselves. Doing so means reimplementing lipgloss's style emitter, because the
compositor does two things a splice would have to reproduce exactly:

1. it **re-emits SGR canonically** — a base carrying `\x1b[0m` comes back carrying `\x1b[m`; and
2. it **pads every line to the widest line of the whole base**, so a partial composite of only the
   window's rows pads to a different width than a full one.

That is a permanent maintenance liability, not a one-off: every lipgloss release would need the patch
re-checked and re-invested, which defeats the purpose of depending on lipgloss at all. **One new SGR
form anywhere in the app would silently change rendering**, and nothing in the test suite would
necessarily catch it.

A memo was also considered and rejected **on measurement**: since 0.12.0 `modes/tty/ticker.go:advanceTicker`
steps the marquee one cell per tick, so the base string changes every frame and a memo keyed on
(base, window, width) would miss every time.

The cost is bounded in a way the frame cost is not — it is paid only while a window is open, which is
seconds at a time, not for the life of the process.

**What would re-open it.** lipgloss offering a cheaper compositing API upstream (ask there, do not
fork); or the cost becoming user-visible — a window that feels sluggish to open, scroll, or type in.
A profile number on its own is not the trigger. Not a splice, and not a vendored patch to lipgloss's
renderer.

**Evidence.** `06_docs/02_features/multi-voice-support/02-analysis/perf-pass-discover.md` §3.3, §9;
`BenchmarkOverlayOnly` in `modes/tty/bench_test.go`.

---

## 2. The marquee keeps a frame drawing for as long as the app is up

**Cost.** `modes/tty/dashboard.go:tickNeeded` returns true whenever the ticker holds events, and there
is essentially always an active national severe event — so a full frame is drawn every 300 ms,
forever. ≈ 13.6 MB/min of the app's ≈ 23.6 MB/min allocation rate.

**Why it stands.** The scrolling ticker **is** the 0.12.0 feature. A marquee that stops moving is a
broken marquee.

**What would re-open it.** Nothing — this is not a candidate for removal. What it changes is where
effort goes: **every per-frame cost is a 24/7 cost**, so the memo-hit allocation pins in
`modes/tty/bench_test.go` are deliberately tight (**365–386 allocations at ×1.05** across the four
sizes, not the flat 6,000 they were until 0.14.0). *(This page said 283–304 until the BUILD-exit red
team: the pins were legitimately RAISED when the bench fixture gained a live marquee — the state the
app actually runs in — and this sentence was not updated with them. A ledger that exists to stop a
settled measurement being re-derived is defeated by a stale number in it.)* Treat a pin failure on the hit path as a real regression, not as a pin that
needs raising.

Note for anyone reading the older record: `watchpost-performance-quality-pass`'s "idle ≈ 9 MB/min,
render's share 0 (no frame drawn)" describes a state the app **no longer enters**. It is not a
regression from that number; it is a different program.

**Evidence.** `perf-pass-discover.md` §2, §7.

---

## 3. One scheduler per RECENT location — ADR-03, option A

**Cost.** 309 of ~330 goroutines are parked in `platform/sched/sched.go:runTier` (50 RECENT locations ×
6 tiers, plus 8 priority tiers and 1 batched alerts scheduler). `stack_inuse` 2.85–4.03 MB.

**Why it stands.** Each row publishes the moment *its own* fetches land, so the list fills in place
rather than in one late batch. Goroutine count is not a metric; the parked goroutines are bounded and
cheap.

**What would re-open it.** ADR-03's own stated trigger: **> 5 MB live** attributable to the
schedulers, or publish-coalescing failures measured as `recent.publishes` per hour against the Q3
baseline. Re-checked 2026-08-30 — **not tripped** (2.85–4.03 MB; 33–37 publishes/h). If it trips, the
replacement is the per-tier phased due list ADR-03 already specifies, not a design invented fresh.

**Re-measured 2026-09-03 under HEAVY SWITCHING — still not tripped, and closer to the ceiling.**
`stack_inuse` **4.19 MB** against the 5 MB trigger, above the 2.85–4.03 MB band recorded on
2026-08-30. 309 scheduler goroutines across 52 schedulers, which is the number this entry already
predicts (50 RECENT × 6 tiers, plus the priority tiers and the batched alerts scheduler). The
headroom note below is therefore optimistic: at 4.19 MB it is **1.3 tiers**. The figure below has
been corrected to match; it read 1.6 for two days, one paragraph apart, so a reader of either alone
got a different answer.

**Why it surfaced now, and why it is not a leak.** The HUM LEAD saw ~170 MB after a session of heavy
radio switching, alert reads and stream toggling, against 90–120 MB before. A scheduler is created
per RECENT location TOUCHED, so the cost is paid per *distinct location visited* rather than per
minute — an idle app never reaches it. It is **bounded by `RecentCap = 50`** (`modes/tty/modal_location.go:49`),
so it plateaus rather than climbing. Measured across 24 switch-read-toggle cycles: goroutines
377 → 413, `stack_inuse` flat at ~4 MB, footprint 92.5 → 79.1 → 88.6 MB — a band, not a ramp.

**The Director work was not implicated AT THE TIME OF THIS MEASUREMENT (2026-09-03), and that
measurement is now superseded.** Measured then against the shipped v0.13.0 with the same script and
the radio playing: Go `Sys` identical at 89 MB, live heap +4 MB, footprint +10.6 MB. The reason given
was that *"the pump and the executors have no caller in the running app"* — which **T3.10b made false
on 2026-09-05**, two days later: the alert rail now runs through the Director on every burst. The
paragraph is kept rather than deleted because the number is still the honest v0.13.0 comparison; what
cannot be carried forward is its REASON. **Re-measure before quoting this entry about the Director.**

**What this DOES mean for Broadcaster.** The HUM LEAD's own framing — *"in a broadcaster role, this
might happen more often: alerts cut over, there's a scheduled lineup, and the ever-present live
stream bed on standby"* — is a workload that touches locations continuously. At 50 locations the
bound holds; a Broadcaster that raises `RecentCap`, or that schedules against a station's whole
service area, spends the 0.43–0.61 MB per tier directly into the 0.8 MB of remaining headroom. That
is the trigger's most likely route, and it belongs in Broadcaster's own DISCOVER rather than here.

**Headroom, so 0.15.0 can spend it knowingly.** Each new feed that earns a RECENT tier adds ~50
goroutines ≈ 0.43–0.61 MB. That is **1.3 tiers at the high-water reading** (4.19 MB of the 5 MB
trigger, per the correction above). Re-measure before the
*second* new feed, not after.

**Evidence.** `watchpost-performance-quality-pass/03-architecture-design/adr-03-recent-scheduler.md`;
`perf-pass-discover.md` §8.

---

## 4. The scheduler fetches a tier's providers one at a time

**Cost.** `platform/sched/sched.go:cycle` walks the providers serving a tier sequentially, so the
cycle's wall time is the **sum** of their latencies and the last provider's data is that much staler.
Linear in the number of APIs — the axis 0.15.0 grows on.

**Why it stands.** Two reasons, both about what the listener sees. `cycle` publishes **after each
provider**, so a slow provider never holds the others' data off screen (UAT 64). And a shared worker
pool would let a rate-limited provider (CO-OPS paces at 5/s) starve the observation rows behind it.

**What would re-open it.** 0.15.0's feed count making a tier's cycle longer than its own cadence — the
point at which sequential fan-out stops keeping up. The bar for changing it is ADR-03's: equivalence
on **publish times** over ≥ 3 simulated hours with injected failures, because concurrency changes
publish order and interleaving, which is user-visible.

**Evidence.** `watchpost-performance-quality-pass/03-architecture-design/quality-pass-plan.md` §2.4;
`perf-pass-discover.md` §8.

---

## 5. USGS asks once per location for the near field

**Cost.** `domains/seismic/usgs/usgs.go:queries` snaps the **regional** query to a shared 4° grid so
nearby locations resolve to one URL, but leaves the **near-field** query centred on the location
itself — one URL per location, and FDSN sends no validators, so none of them can be a 304.

**Why it stands.** Snapping the near field would need a buffer radius, and a buffer balloons during a
swarm: the query would return a mass of events the location cannot feel, to be filtered client-side.
The measured cost is a launch burst (79 requests in the first 33 s, then +2 over the next nine
minutes), not a steady-state one.

**What would re-open it.** USGS rate-limiting us, or the near-field fan-out appearing in *steady-state*
counters rather than the launch burst.

**Evidence.** `perf-pass-discover.md` §5.

---

## 6. A conditional GET re-reads its body from disk when the entry was evicted

**Cost.** `platform/httpx/cache.go:stale` needs a body to offer a revalidation, so an entry pushed out
of the 8 MB memory tier is read back from disk. 64 MB of `os.ReadFile` in the first 33 seconds.

**Why it stands.** It is a **launch** cost — 0.75 MB per 3 minutes in steady state — and the 8 MB
memory bound is itself a deliberate ceiling. The revalidation it pays for is the single biggest saving
in the app: NWS spends 1.44 MB of network to avoid 12.79 MB.

**What would re-open it.** More feeds raising eviction pressure until this appears in the steady-state
allocation profile. The shape of the fix is known — keep the validators resident and read the body
only after a 304 answers — but it is not worth the complexity while the cost is launch-only.

**Evidence.** `perf-pass-discover.md` §4.3, §5. Related, already fixed: L4-F3 in
`watchpost-performance-quality-pass/02-analysis/lens-L4-caching-memoization.md`.

---

## 7. The publish coalescers are tuned for how rows FILL, not for fewer publishes

**Cost.** `app/pipelines.go:startRecent` holds a 5-second window in steady state but a 250 ms window
for the first 90 seconds, and `app/pipelines.go:startStaggered` starts the 50 schedulers 10 ms apart
rather than together.

**Why it stands.** Both numbers exist so seed rows rehydrate **as their fetches land** instead of
appearing all at once after a long window; the stagger also keeps the launch burst from costing ~90 OS
threads. Widening the window or starting together would look like an optimisation on a counter and be
a regression on screen.

**What would re-open it.** Nothing on allocation grounds. Change these only for a stated
user-experience reason, with the launch re-watched on a pty.

**Evidence.** `perf-pass-discover.md` §8; the constants' own comments in `app/pipelines.go`.

---

## Older accepted non-decisions

The 0.10.x quality pass recorded its own list, still in force:
`06_docs/02_features/watchpost-performance-quality-pass/06-key_learnings/quality-baseline.md` §6 —
`SetMemoryLimit` (ADR-05), the shared disk cache for the two clients, the two alert schedulers,
compass 8 → 16, `Tok()`-time precompute, the `layoutFor` package memo, and the bespoke HMS byte
scanner.

Its §4 table states the bound on every cache and memo in the app. If you are about to add one, put its
bound and its gauge there.
