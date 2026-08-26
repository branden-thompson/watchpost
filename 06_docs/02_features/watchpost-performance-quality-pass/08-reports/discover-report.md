# DISCOVER Report — Watchpost Performance & Quality Pass

| Field | Value |
|---|---|
| Phase | DISCOVER (RCC) · SEV-0 · HUM LEAD · FULL GIT/DOCS/REPORTS/DIAGRAMS/TDD |
| Date | 2026-08-26 |
| Branch | `feature/watchpost-performance-quality-pass` (off `feature/watchpost-cli`; tree = v0.9.4) |
| Inputs | approved project brief; five read-only research lenses (L1 memory, L2 chattiness, L3 structure, L4 caching, L5 go-studs/render); an instrumented 6-hour run (pprof on, counting proxy, Synth on Repeat: Watchlist); HUM LEAD's 9-hour v0.9.4 process (PID 67943, hourly baseline running 24 h) |
| Status | APPROVED — HUM LEAD, 2026-08-26 ("Approved; Go 4 PLAN"); C3 and C5 deferred to PLAN with the recommendations in §4 |

---

## 1. Answer to the problem statement

> *People who leave Watchpost running for days cannot tell whether its slowly rising resource use is normal or a defect … and the record cannot tell them either.*

**The record can now tell them.** With five lenses and two live processes measured:

- **No unbounded memory growth term exists in the code as written** (L1 verdict). Every long-lived
  structure is bounded and enforced; every goroutine is either the fixed scheduler set or cancelled/joined.
  The instrumented process holds 42 MB of live heap at 13 minutes with exactly the goroutine count the
  wiring predicts.
- **The thread count is a ratchet, not a slope.** 23 Go runtime threads at launch (Go never retires idle
  OS threads; the count follows the peak of concurrent blocking syscalls), 5 Apple audio threads on first
  tune-in, and 2–3 macOS libdispatch workqueue threads that appear and retire. Observed 31 → 30 on the
  9-hour process; 27–30 on the instrumented one.
- **What reads as "slowly rising" is churn, not retention.** Rendering allocates ~80 % of all bytes
  (≈ 94 MB/min idle, ≈ 620 MB/min with the visualizer), and the HMS fire-archive parse spikes ~85 MB
  every 10 minutes; together they set the GC's heap ceiling and the 116 → 175 MB footprint sawtooth.
- **Two true growth terms exist, both on disk or human-bounded:** the httpx **disk cache tier has no
  sweeper** (1,376 files / 116 MB after two days on this Mac, 593 of them orphans of a pre-UAT-73
  format; date-keyed URLs add files forever), and the NWS `points` memo grows by ~1 KB per location
  ever looked up (sub-MB per year).

That is the answer for today. OQ-8 raised the bar to weeks/months on both platforms: the pass must also
**prove** it with an attributed multi-day run, which needs the instrumentation gaps closed (§4, C1).

## 2. What the pass found beyond the question

Two findings are defects rather than quality items and would ship even without the pass:

- **D1 — weatherUSA relays have never been reachable from Go** (live-run LR-1). Its directory server
  accepts only RSA key-exchange TLS suites; Go 1.22 dropped them and Go 1.27 removed the `tlsrsakex`
  switch. Effect: four failed handshakes every 5 minutes while tuned (the retry ladder in the wild) and
  ~119 relay mounts never offered — "Nearest Relay" has only ever had wxradio.org. The mounts are plain
  HTTP and stream fine; the plain-HTTP directory URL returns the same document. Fix is a constant plus a
  visible warning.
- **D2 — the table's header and un-styled cells are painted by go-studs' own palette, gated by `$TERM`**
  (L5-F4). The accepted look (purple header, grey 245 cells, white names) comes from the kit and flips
  with `$TERM`; the theme chooser cannot restyle those cells and no test pins it.

And one **outage-amplification risk** (L2-F1): two stacked retry ladders (httpx 4 attempts × scheduler
10/20/40 s) plus an observation station chain that walks all four stations on transport errors —
an estimated **~23,000 attempts/hour during an NWS outage** against ~1,000 healthy. Politeness to the
providers and the thread symptom both point here.

## 3. Consolidated findings, ranked by value ÷ risk

Severity: **HIGH** = defect or growth term · **MED** = measurable waste / correctness seam · **LOW** =
polish. Risk: how likely the change is to regress behaviour (M3) — *nil* = pure move or measured
no-drift; *low* = pinned by existing tests; *med* = touches a hot seam.

| # | Sev | Finding (lens IDs) | Evidence | Direction | Risk |
|---|---|---|---|---|---|
| 1 | HIGH | weatherUSA directory TLS failure; 4 retries / 5 min; no user-visible warning (LR-1) | reproduced with the app's client; bisected to RSA-kex-only server | plain-HTTP directory URL (+ optional RSA-suite transport); `radio_unavailable` warning when a directory fails | low |
| 2 | HIGH | Disk cache tier unbounded: no sweep, orphaned format, date-keyed URLs; 95 % of writes (~45k/day, 0.5–0.7 GB/day) persist entries that can never serve a relaunch; expired files re-read on every miss (L4-F1/F2/F3, L1-F10) | 1,376 files / 116 MB / 593 orphans on this Mac; 473 µs per write | sweep at start + daily; persist only TTL ≥ 5 min; skip the known-expired read — one file | low |
| 3 | HIGH | Outage amplification ~20× (L2-F1); `Retry-After` unhandled | cadence math over the two ladders | one retry layer; station chain only on 404/incomplete; per-host failure memo; honour `Retry-After` | med (sched + httpx) |
| 4 | MED | go-studs probes `/dev/tty` twice per frame (L5-F1) | 60 % of sampled CPU in `View()`; −22 % frame time with a memo | lazy capability probe; `x/term.GetSize` (retires an `unsafe` exemption) | low — **kit, HUM LEAD** |
| 5 | MED | Regexp ANSI width strip in app and kit; `SGR()` 5 allocs per cell (L5-F2/F3, L1-F3) | 19 % of allocation; prototype −49 % time / −68 % allocs, all goldens green | `x/ansi.StringWidth` as the one width authority; memoised `SGR` | low — **kit half HUM LEAD** |
| 6 | MED | Render churn: unconditional 300 ms shimmer tick; layout facts recomputed ~8× per frame; column spec rebuilt per frame (L1-F1/F2/F4, L5-F6/F11/F16) | 94 MB/min idle, 620 MB/min viz; ~15 % of frame bytes thrown away | gate the tick like `vizTick`; per-frame `frameLayout`; memo `layoutFor` | low (app only) |
| 7 | MED | No conditional GETs though every NWS product carries an ETag (L4-F4) | full downloads of unchanged bodies every expiry; 60–80 % of NWS bytes | store validators; `If-None-Match`; 304 renews in place — request counts unchanged | med (httpx core, well tested) |
| 8 | MED | Table colours come from the kit and `$TERM` (L5-F4 = D2); `ColorSequence` mangles composites so user theme files can render garbage (L5-F5) | measured on both | app sets every column's style from `Tok()` (new tokens defaulting to today's values); composite-aware `SGR` collapses `sgrRaw`/`TintRaw` | low — **visual + kit, HUM LEAD** |
| 9 | MED | FIRMS: 120 per-location boxes per 10 min for one continental product (L2-F2) | ~500/h vs ~20/h possible at the same cadence | merge boxes regionally; filter with `fire.Near` as today | low |
| 10 | MED | `modes/tty/dashboard.go` 3,332 lines / 12 responsibilities; `render.go`, `app/dashboard.go`, `app/radio.go`, `nws/provider.go`, `assembler.go` likewise (L3-F1–F6) | line ranges mapped per responsibility | pure file moves, one commit per package, with file-map comments | nil |
| 11 | MED | Ten modal booleans with hand-maintained exclusivity — already inconsistent at three sites; no test pins it (L3-F15) | `:338`, `:544`, `:3275` | exclusivity table test first, then `type modal int` | med (mechanical) |
| 12 | MED | Geodata index loaded twice at launch; WFIGS 208 KB re-decoded per Fetch (~57 MB/h); `GetText` copies 1.4 MB per HMS Fetch; HMS memo pins 4 MB of substrings (L4-F5/F6, L1-F7/F9) | 36 ms + 19 MB; 0.98 ms × 200/h; measured | load once; body-hash memo (shared `fire` helper); return the immutable slice; intern two fields | nil |
| 13 | MED | HMS parse spikes ~85 MB every 10 min (L1-F8) | 75 MB / 1.05 M allocs per parse | stream the zip entry into the decoder; `strings.Cut` instead of a map per placemark | low |
| 14 | MED | RECENT runs one scheduler per location: 258 parked goroutines, 250 tier timers, the 4× launch burst, per-location fire fetches (LR-2, L1-F17, L2-F9, L4-F6) | goroutine census | one scheduler with per-location tickers **or** keep — the per-row "publish when *my* fetch lands" is a UX choice for PLAN | med |
| 15 | MED | `report` one-shot fans out every kind × provider serially at 5 req/s (L2-F8; OQ-9) | 4–10 s cold, ~1–3 s warm | `errgroup` over kinds; one `newClient()` owner | low |
| 16 | LOW | CO-OPS astronomical predictions refetched hourly (L2-F3); evening gridpoint fill 16 MB/h (L2-F6); synth re-fetches obs/alerts (L2-F7); two alert schedulers (L2-F15) | cadence math | TTL to next UTC midnight; memo max/min per issuance; narrow assembler read; optional merge | low |
| 17 | LOW | `gridInfo` never expires or refreshes (L4-F7, L1-F15, L2-F13) | ~1 KB/entry forever; doc promises a daily refresh | `resolvedAt` + 24 h; drop on removal | low |
| 18 | LOW | Duplicates and single-owner knobs: `thousands` ×2, three compass tables, `displayCond` ×2, `recentCap` ×2, watch cap literal ×3, nine hand-built chip rows, `WrapSegments` re-splits (L3-F8–F14, L5-F15) | listed with lines | extract at the second caller; `tty.RecentCap`/`WatchCap`; `controlsRow` helper | low |
| 19 | LOW | P10 exemptions: 3 stale (`app/setup.go` symbols gone), 2 removable by one-liners; 76 kit entries collapsible to per-rule package exemptions with the real items tracked (L3-F17/F20, L5-F18) | 56 → 51 without deleting a check | do it | nil |
| 20 | LOW | Tests: `app` 16 % (pure logic untested), `cmd` 33 %, no plain-report golden, `invariant` 0 %, a theme value pinned in a tty test, pty smokes not in CI (L3-F22–F27) | coverage run | targeted tests in the split files; golden; note the manual smokes | nil |
| 21 | LOW | Transports: ICY and Piper build their own (outside the pure-Go-resolver rule); idle 90 s vs 10-min cadences re-handshake each tick; TLS resumption unverified; `CacheStats` dead; no request counter (L2-F10–F12/F16, L4-F8/F13, L1-F19) | — | `httpx.NewTransport()` one owner; ~11 min idle; counters into [S] | low |
| 22 | LOW | Kit correctness off the app's path: byte-width family wrong for non-ASCII; O(n²) truncation; `tableGeom` duplicates the kit's fill math; `clampCells` patches a kit gap; chroma (a syntax highlighter) linked in via `go-studs/components` — 0.5 MB init heap and binary weight (L5-F7/F8/F13/F14, LR) | measured | `Geometry()` API; `Clamp`; delegate width family; O(n) truncate; drop the chroma import | low — **kit, HUM LEAD** |
| 23 | LOW | Docs stale: architecture §4 (`modes/report` does import `render`; phantom primitives; `platform/audio`), stale synth-cache comment, launch-burst comment, `NOTICE.md`/sync script contradict the go-studs change policy (L3-F30, L1-F13, L2-F9, L5-F19) | — | as-built import graph; per-package file maps; a patch ledger the sync script re-applies | nil |

**Refuted (recorded so the next pass does not re-chase them):** `Assembler.Snapshot()` deep copy
(0.18–0.7 ms, < 5 % of churn — L1-F5); `Tok()` under the RWMutex (15–18 ns — L4-F10); the HMS proximity
sweep (3 ms, 0 alloc — a spatial index buys nothing — L4-F11); `TintDefault`/`TitleGradient` (L5-F10);
glyph widths (every app glyph measures 1 cell in every library — L5-F12); the memory caches as the
source of the footprint swing (they sum to < 50 MB — L4-F15).

## 4. What DISCOVER could not settle (for PLAN)

| ID | Question | Why it is open |
|---|---|---|
| C1 | Attribution over days | the shipped hook is opt-in at launch only; the 9-hour process is opaque. OQ-2 approved a hook — PLAN chooses signal/file trigger and the dump set (heap, goroutine, threadcreate, request counters) |
| C2 | Obs cadence 90 s vs NWS `s-maxage=300` (L2-F5) | needs the header observed on the instrumented run before any cadence argument; the run's proxy counts connections, not requests — a counter (L2-F16) lands first |
| C3 | RECENT scheduler shape (#14) | a UX decision about per-row publish timing, not a code question |
| C4 | go-studs change process (L5-F19) | policy: a patch ledger the sync script re-applies, and what "approval" records |
| C5 | Provisional targets (OQ-5) | measured today: idle 67–78 MB footprint, radio-on plateau 83–100 MB (instrumented) / 116–175 MB (9-hour), threads ≤ 31, ~1,000–1,900 req/h. Recommendation for PLAN: idle ≤ 90 MB, radio-on plateau ≤ 160 MB, peak ≤ 220 MB, threads ≤ 32 attributed — and **zero growth over 72 h on both platforms** as the M1 bar |

## 5. Proposed batch plan for PLAN (each batch: tests first, gates, record, HUM LEAD review)

| Batch | Content | Findings | Risk | Ships as |
|---|---|---|---|---|
| Q0 | Instrumentation: request/cache counters into [S]; signal-triggered profile dump; the sampling harness in `scripts/` | C1, C2, L2-F16, L4-F8 | nil | tooling |
| Q1 | Defect + hygiene: weatherUSA directory; disk-tier sweep/floor/skip; stale exemptions; stray files | #1, #2, #19 | low | `v0.9.5` |
| Q2 | Pure structure: file moves (tty, render, app, nws, assembler) with file maps; cheap tests (invariant, plain golden, app logic) | #10, #20, #23 | nil | with Q3 |
| Q3 | Render path, app side: gated tick, `frameLayout`, `layoutFor` memo, geodata once, WFIGS memo, HMS streaming parse + interning, `GetText` no-copy | #6, #12, #13 | low | `v0.9.6` |
| Q4 | go-studs (HUM LEAD approvals, patch ledger): lazy tty probe, `x/ansi` width, `SGR` memo, composite-aware `SGR`, theme-owned table colours, `Geometry()`, `Clamp`, drop chroma | #4, #5, #8, #22, C4 | low–med | `v0.9.7` |
| Q5 | Network: single retry layer + `Retry-After` + failure memo; conditional GETs; FIRMS box merge; CO-OPS TTL; gridpoint memo; synth narrow read; `gridInfo` expiry; one transport owner; `report` errgroup | #3, #7, #9, #15–17, #21 | med | `v0.9.8` |
| Q6 | Seams: modal enum (test first), duplicates/knobs, RECENT scheduler decision (C3) | #11, #14, #18 | med | `v0.9.9` or `v0.10.0` |
| Q7 | Proof: 72 h attributed runs on macOS and Arch; the quality baseline document future passes diff against | M1, M5, OQ-8 | — | record |

Each batch is independently shippable and reversible; Q0/Q1 first because everything after them is
measured against their counters and their cleaner cache.

## 6. Metrics status at DISCOVER exit

| Metric | Baseline captured | Note |
|---|---|---|
| M1 `M-STAB` | 9-h process: footprint 116–175 MB, 30–31 threads; instrumented: 83–100 MB, 27–30 threads, 279–281 goroutines | hourly baseline continues 24 h; 6-h run continues; both appended as addenda |
| M2 `M-CHAT` | derived: ~1,000/h NWS, ~1,500–1,900/h total; floor ~1,000/h | measured confirmation waits on Q0's counter |
| M3 `M-REG` | all gates green at `b13d7f4`; per-frame render baseline 670 µs / 437 KB / 10,044 allocs; warm launch 550 ms | the numbers every batch must not exceed |
| M4 `M-P10` | 0 live; 56 non-kit exemptions (3 stale) | target ≤ 51 |
| M5 `M-DOC` | 02-analysis holds L1–L5 + live-run findings + baseline log | the corpus has started |

## 7. Gate

DISCOVER asks HUM LEAD to (a) accept the answer in §1 and the defect finding D1, (b) ratify the batch
order in §5 as PLAN's starting point (PLAN will propose approaches per batch), and (c) decide C3 (RECENT
scheduler shape) and C5 (targets) — or defer both to PLAN with the recommendations above.
