# PLAN Report — Watchpost Performance & Quality Pass

| Field | Value |
|---|---|
| Phase | PLAN · SEV-0 · HUM LEAD approves architecture · FULL PLAN / DIAGRAMS / TDD |
| Date | 2026-08-26 (v3, after two PLAN-exit red-team rounds) |
| Plan of record | `03-architecture-design/quality-pass-plan.md` v3 (approaches, diagrams, numbered tasks, risks, disposition table) |
| Critical analysis | `08-reports/red-team-plan.md` — round 1: 10 formal lenses + 2 probes, multi-agent; round 2: single-agent over the v2 delta; both `SHIP-WITH-CONDITIONS`; round 2 recommends **no third round** |
| Status | **APPROVED — GO for BUILD Q0** (HUM LEAD "Go", 2026-08-26): C5 ratified as restated in v3; compass 8→16 excluded (any future change via a specific approval request); `--ascii` default taken (wire it in Q3 — flagged for HUM LEAD to veto at the Q3 gate); round 2 complete, its 23 findings applied as v3 |

## Executive Summary

DISCOVER answered the question (no memory growth term found; churn and an unbounded disk tier; one relay
defect). PLAN v1 turned the 23 ranked findings into seven batches. The PLAN-exit red team — four axes, the
Principal-Architect lens, five personas and two probes — returned a unanimous **SHIP-WITH-CONDITIONS**:
the architecture held, the *specification* did not. Ten Critical findings (nine distinct defects), all
fixed in the plan of record v2:

- the **measuring apparatus could not answer the problem statement** — footprint under a 60 MB sawtooth
  cannot see a 1 MB/day leak, `threadcreate` records `<unknown>`, and the 24 h "before" sampler had died
  after one row (restarted; 5-minute cadence; gap recorded);
- the **per-host failure memo would have blackholed `api.weather.gov` including alerts** — measured: one
  station 5xx turns an 11 s first view into 71 s; redesigned and hoisted to Q1;
- the **disk sweep was a deny-list on a `$HOME`-derived directory**; it is now an allow-list with a path
  guard and a planted-file test;
- **NO_COLOR is violated today** by kit-painted cells and the plan had not named it; a pinning test and
  per-theme contrast overrides are now first-class;
- the **go-studs patch process could be destroyed by a resync**; patches move beside the kit, the sync
  script gains a pin check and fail-loud apply, Q4 splits into correctness (local) and performance
  (upstream-first);
- **P10 was not a failing gate** and the Q4 exemption "collapse" was mechanically impossible; every gate
  now snapshots `a2dh p10 check` with the delta in the build log;
- the **"idle ≤ 10 MB/min" target was unreachable** by tick gating and the predicate would have frozen the
  marquee; both restated from measurement.

Plan v2 keeps the batch shape (instrument → fix → move → render → kit → network → seams → prove) with
Q1 grown into the resilience batch, Q4 split, C3 decided as A at PLAN, and every gate labelled CI or
local.

## Decisions requested

| ID | Decision | Recommendation (v2) |
|---|---|---|
| C5 | Numeric targets | ratify plan §1 v3: idle ≤ 90 MB; plateau ≤ 160 / peak ≤ 220 MB (fragmentation check only); **growth = post-GC heap sampled every 5 min, per-hour minimum, HAC OLS slope, upper 95 % CI × 30 d < 5 % of the post-GC heap plateau on both platforms + every counter flat + no rising pprof site — with the detection floor measured at Q0 proven below that bar first (R2-1)**; threads ≤ constructed bound and stable; allocs ≤ 6,000 (Q3) / ≤ 3,300 (if Q4b) at 133×44 in a non-race CI step, µs recorded at three sizes; allocation attributed (render/publish/parse); outage ≤ 3× and alerts never delayed (lane property); warm launch ≤ 550 ms median of 10; exemptions ≤ 52 with 0 unmatched and zero new exemptions for new code |
| C3 | RECENT scheduler shape | **A — keep the 50 schedulers, record the ~1 MB cost** (decided at PLAN per PA-3/CQ-6/BQ-4/RT-8/PF-5); B re-opened only on a stated threshold after Q3 + Q5; RECENT publish consolidation moves to Q3 |
| C4 | go-studs change process | patches under `third_party/go-studs/patches/` + `LOCAL_CHANGES.md` with a pinned upstream commit; sync script rewritten fail-loud with a test; Q4a correctness patches local (004, 001, 003, 008), Q4b performance patches upstream-first |
| C1 | Dump trigger | `SIGUSR1` in `app/dump_unix.go` (+ `dump_windows.go` env-only), rate-bound, 12 retained, errors into [S]; declined the "env hook only" alternative because OQ-2 is about a running process |
| C2 | Obs cadence | answered by the probe: NWS serves obs `max-age=300`, the 90 s tier is already coalesced — no change; recorded at the Q5 gate |
| OQ-6 | Exclusion list (§0.9) | content changes, cadences without a §0.3 table, kit API, new features are **out**; two items need your call: **compass 8→16** (changes spoken wording — default: excluded) and **`--ascii` wiring** (a documented promise that is unreachable — default: wire it in Q3) |
| OQ-7 | Release cadence | `v0.9.5` (Q0+Q1) · `v0.9.6` (Q2+Q3) · `v0.9.7` (Q4a) · `v0.9.8` (Q5) · `v0.10.0` (Q6) — each via the publish protocol with an `[Unreleased]` section between |
| Step 9 | Convergence | round 2 (single-agent, v2 delta) ran 2026-08-26: 0 Critical / 8 Important / 15 Minor, none architectural, all applied as v3; **no third PLAN round** — R2-1..8 are re-verified by the Q0 BUILD-exit red team, where σ and alloc counts become measured facts |

## Approaches weighed (summary — full text in the plan of record)

| Batch | Alternatives | Chosen | Why |
|---|---|---|---|
| Q0 dump trigger | signal · [S] key · trigger file · env hook only | **signal + env hook** | zero idle cost; attributes a running process; costs bounded (rate, retention, [S] warning) |
| Q0 M1 statistic | footprint slope · post-GC heap slope + counters + pprof diffs · `SetMemoryLimit` | **post-GC heap slope + counters + diffs** | footprint cannot resolve the leak the problem names; a GC ceiling hides rather than proves |
| Q1 disk tier | floor+sweep+skip · bbolt store · drop the tier | **floor + allow-list sweep + one retention rule** | one file, testable, keeps warm relaunches, safe by construction |
| Q1 weatherUSA | http:// directory only · RSA-suite transport · http:// directory + mounts + redirect policy | **http:// both + host pin + redirect policy** | the mounts are already plain HTTP; directory-only left mounts unplayable |
| Q1 retries | scheduler only · memo only · client `MaxRetries` + guarded memo + clamped `Retry-After` + chain on any error | **the guarded design** | a blip heals ≤ 15 s; one 5xx never touches alerts; an outage ≤ 3× |
| Q5 revalidation | none · `If-None-Match` · `If-Modified-Since` primary | **`If-Modified-Since` primary** | NWS mangles ETags; alerts and `/points` cannot 304 — target restated to the kinds that can |
| Q5 FIRMS | regional merge · per-provider tile canonicalisation | **5° tiles** | per-location `Fetch` has nothing to merge; tiles bound span and isolate failure |
| Q6 RECENT | keep 50 schedulers · heap scheduler · per-tier phased list | **keep (A)** | goroutines are not a metric; batching is delivered elsewhere; a pool starves obs rows |
| Q4 width | regexp (today) · x/ansi.StringWidth | **x/ansi, upstream-first (Q4b)** | zero drift measured; Q3 + Q4a reach the gate without it |

## Task breakdown

Q0 8 tasks · Q1 6 · Q2 4 · Q3 3 (14 named tests) · Q4a 6 · Q4b 6 upstream candidates · Q5 10 · Q6 4 ·
Q7 the two soaks and the baseline document — every task test-first and numbered in the plan of record.
Sizing (re-derived per PA-9, R2-16): Q0 2 d · Q1 2 d · Q2 1 d · Q3 1½ d · Q4a 2 d · Q5 3 d · Q6 1 d ·
Q7 3 d elapsed. Every batch ends with: `make verify` (now incl. tidy/vuln/verify), `make p10` snapshot
(local — the CLI and ledger live outside the public tree), lints, `a2dh validate`, the batch's own
measurement (CI vs local named), the build log with its docs-touched line and before/after table, and a
HUM LEAD review.

## Metrics of success — how each batch moves them (v2)

| Metric | Q0 | Q1 | Q2 | Q3 | Q4a | Q4b | Q5 | Q6 | Q7 |
|---|---|---|---|---|---|---|---|---|---|
| M1 stability | **statistic + counters exist**; baseline complete | disk and neg-cache growth terms closed | — | live-heap levers (geodata, HMS) | — | — | `gridInfo` bounded | — | **proven 72 h × 2 platforms** |
| M2 chattiness | measured | outage ≤ 3×; directory ≤ 1/5 min | — | publish rate consolidated | — | — | bytes measured on 304-able kinds; FIRMS tiles; floor re-derived | — | confirmed |
| M3 no regression | alloc pins at baseline | fault tests; relay **plays** on both platforms | tests ↑ | goldens (colour-off + ASCII); tick tests | NO_COLOR + fidelity + contrast | goldens | fault + cache tests | modal test; smokes | Linux re-run |
| M4 P10 | snapshot; ≤ 52 budget | 56 → 52; unmatched 0; kit entries itemised | — | bounds tested | kit entries retired per patch | — | — | — | — |
| M5 record | infra ledger; soak profile; artefacts in-tree | build log + CHANGELOG | flow map; ADRs; docs-touched | build log | patch ledger | upstream PRs | cadence table | build log | baseline doc + extrapolation table |

## Risks

RP-1 impure file moves · RP-2 kit patch drift · RP-3 width goldens · RP-4 under-/over-reacting retries ·
RP-5 304 poisoning · RP-6 tick gating freezes the unseen · RP-7 doc load · RP-8 sweep deletes a user
file · RP-9 plain-HTTP relay on path · RP-10 Arch laptop unavailable — each with a mitigation in the plan
of record §4.

## Critical analysis

`red-team: SHIP-WITH-CONDITIONS · multi-agent · scope:feature · personas:[perf, junior-dev,
safety-critical, infosec, a11y]` — `08-reports/red-team-plan.md`. **Round 1:** 134 raw findings from
10 formal lenses (4 axes, PLAN lens, 5 personas) and 2 probes, consolidated into 24; 10 Critical (9
distinct defects) and 53 Important, all dispositioned; **Round 2** (single-agent over the v2 delta):
every round-1 "Fixed" verified present in v2 (seven partially); 0 Critical / 8 Important / 15 Minor,
all Fixed in plan v3 (one framework backlog item); no third round. Round-1
dispositions: Fixed (plan v2; operational: sampler), Declined ×5 with reasons (env-hook-only dump,
`SetMemoryLimit`, L5-F9 for now, `tools/alertrec`, cross-process freshness), Deferred ×5 (two to HUM
LEAD, one to the Q7 gate, two to Q5b with numbers). Nothing dropped; both rounds' raw outputs stand
verbatim under their syntheses. Step 9: converged (see Decisions).

## Gate

HUM LEAD: provisional approval given 2026-08-26 (architecture, C3 = A, C4, C1; compass excluded).
Remaining: ratify C5 as restated in v3 (the growth criterion now carries a measured detection floor),
decide `--ascii` (default: wire it in Q3), and **GO for BUILD Q0**.
