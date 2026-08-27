# P2 build log — Seismic scheduler + shared canonicalised-box requests

**Batch:** P2 of the seismic PLAN (`03-architecture-design/plan.md` §5, Approach B). **SEV-0** · FULL TDD.
**Branch:** `feature/seismic-data`. **Status:** at gate — unit + live green; the 1-hour counters run is the measured gate (§4).

## 0. Freshness argument (plan §0.3 — required for a new cadence)

USGS serves the FDSN query with `Cache-Control: max-age=60`. A person asking *"did my area shake?"* does
not need sub-minute latency — a quake is no more current than the feed, and the felt event they care about
is minutes-to-days old, not seconds. So the seismic tier rides the **fire cadence** exactly (both hazards are
regional, both slow-moving): **5 minutes** on the priority pipeline (favourites), **15 minutes** on the RECENT
pipeline. The client cache (`feedTTL = 2 min` caller floor) and the shared regional box keep steady-state
network near zero — a sub-2-minute re-ask is served entirely from cache (0 requests), and a 5-minute tick
revalidates with a conditional GET that the unchanged feed answers `304` (bytes ≈ 0).

## 1. What landed (junior-first)

The scheduler now fetches earthquakes, and it fetches them **cheaply** — the P1 data layer's simplest-correct
per-location query is replaced by the concentric shared-box request the plan reserved for P2.

- **Two concentric queries, not one wide one.** The graduated rule is concentric — a small quake shows only
  if very close, a large one from far away — so `domains/seismic/rules.go:QueryPlan` splits the fetch to match:
  a **near-field** query (magnitude `[0, 3.5)`, radius 20 mi, centred on the location) and a **regional** query
  (magnitude `[3.5, ∞)`, radius 1000 mi). Unioned (deduped by USGS event id) and `Keep`-filtered, the result is
  **identical** to one wide magnitude-0 query — the equivalence property, unit-pinned two ways
  (`TestQueryPlanEquivalenceWithBruteForce`, `TestConcentricFetchEqualsSingleWideQuery`).
  - *Why:* a single wide `minmagnitude=0` query over the widest (1000 mi) circle pulls the **entire**
    low-magnitude field — ~1 MB, ≈1,450 events for Los Angeles — of which the rule keeps a handful (the sub-M1
    band only reaches 3 mi). Measured live 2026-08-27; the concentric split pulls **4–31 KB** with the same
    visible set. A ~30–250× byte reduction, by construction.
- **The regional box is shared; the near-field query is not.** The regional query snaps its centre to a fixed
  **4° grid** (`domains/seismic/usgs/usgs.go:queries`), so every location in a cell resolves to one URL — the
  httpx cache + singleflight key — and one request serves the whole cell (the Q5 FIRMS-tile precedent). The
  snapped circle adds the cell's half-diagonal so it still covers each location's own reach; `Keep` trims to the
  exact per-location set. The near-field query stays **per-location** (no snap): snapping it would inflate its
  radius by the cell buffer, and low-magnitude counts scale with area — a 0.5° near snap took Ridgecrest from
  28 KB to **254 KB** in an active swarm (measured). The regional query is sparse (M ≥ 3.5) and
  swarm-insensitive, so its buffer is free.
- **A bounded, gauged parse memo.** `domains/seismic/usgs/boxmemo.go` decodes each distinct box body once,
  revalidated by hash — the many locations sharing a regional box (and a box whose body has not changed between
  ticks) parse it a single time. LRU-bounded at `maxBoxes = 160` (≤ 60 near boxes + a handful of regional), and
  surfaced as the `usgs.memo.boxes` / `usgs.memo.parses` gauges in the `[S]` modal and `counters.json`.
- **Wired into both pipelines and the one-shot report.** `KindSeismic` tiers (§0) in `app/pipelines.go`;
  `seismicProviders` / `seismicRules` (`app/seismic.go`, the `fireProviders` sibling) map `[seismic]` config onto
  the domain rules and register the provider; `platform/sched/sched.go:domainFor` routes the kind; `app/app.go`
  adds it to the report's kinds. The section is not rendered yet — that is P3 — but the data now flows into the
  snapshot exactly as fire did before its section landed.

## 2. Tests first (the pins)

| Pin | Guards |
|---|---|
| `TestQueryPlanIsConcentric` | default bands ⇒ near `[0,3.5)@20mi` + regional `[3.5,∞)@1000mi`; near ceiling == regional floor (no gap) |
| `TestQueryPlanEquivalenceWithBruteForce` | the plan, unioned + `Keep`-filtered, == a single wide query, at every band boundary and across the pivot |
| `TestConcentricFetchEqualsSingleWideQuery` | the same equivalence end-to-end through the provider against a realistic fake FDSN server |
| `TestRegionalBoxIsSharedAndNearIsPerLocation` | two nearby locations share one regional URL; a distant one differs; near URLs are per-location; the regional centre is snapped |
| `TestBoxMemoParsesSharedBodyOnce` | 10 co-located locations ⇒ 11 boxes / 11 parses (regional shared) and **one** regional network request |
| `TestFetchAppliesTheGraduatedRuleAndSorts` | the two concentric queries go out; filter + sort + distance/bearing/depth (re-pinned on the new path) |
| `TestAsOfDistinguishesNoneFromUnavailable` | answered-empty (AsOf set) vs down (frag.Err, no state) — unchanged |
| `TestFetchCapsASwarm` | 40 in-band quakes → capped at 20 — unchanged |
| `TestSeismicCountersFloor` (`WATCHPOST_SEISMIC_COUNTERS=1`, CI-skipped) | the live 1-hour counters gate (§4): per-host request/byte floor, memo bound |

The fake FDSN server (`usgs_test.go`) now filters its catalog by the circle and the magnitude window exactly as
the real feed does, so the concentric queries return disjoint correct sets — the tests exercise the real sharing
and dedup, not a stub that echoes one body.

**A new invariant.** `Rules.Valid()` now also requires **non-decreasing radius** with magnitude (a larger quake
is felt at least as far) — the property the concentric split relies on, made explicit so `QueryPlan`'s
near/regional break is correct by construction.

## 3. Performance (the mandate)

- **Frame allocation:** `make alloc-budget` **unchanged** (tty 504 / 8,445) — P2 touches no render path.
- **Byte floor (live-measured, 2026-08-27, per box):** single wide `minmag=0` ≈ **1,053 KB** (LA) → concentric
  **8 KB** (LA), **31 KB** (Ridgecrest, mid-swarm), **4 KB** (Houston). The rule's discard is no longer fetched.
- **Sharing (counters smoke, 16 locations, 1 tick):** 29 requests / **108 KB** total (≈ 6.75 KB/location) — the
  SoCal cluster collapsed 6 locations onto shared regional boxes (29 < 32 = 16×2); a sub-2-min re-ask was **0
  requests** (pure cache). The full run is §4.
- **Bounds (P10-03):** per-location list capped at 20; the box memo LRU-bounded at 160 and gauged.

## 4. Gate

| Check | Result |
|---|---|
| `go test ./domains/seismic/...` | green (unit + live proof + concentric equivalence) |
| `make verify` | ALL GATES GREEN |
| `make alloc-budget` | unchanged (504 / 8,445) |
| `make p10` | **0 live · 0 unmatched** · ledger **111** (55 kit + 56 non-kit) — **no new exemptions**; the one new finding (a `QueryPlan` while-loop, P10-02) was **fixed** to a bounded range loop, not exempted · snapshot `07-readiness/p10-p2.json` |
| cadence doc | `app/testdata/cadences.md` regenerated (the two new tiers, freshness recorded §0) |
| 1-hour live counters run | `WATCHPOST_SEISMIC_COUNTERS=1` — **see the SUMMARY below / `07-readiness/counters-p2.log`** |

### 1-hour counters SUMMARY (16 locations, 5-min cadence, 13 ticks) — `07-readiness/counters-p2.log`

| Metric | Result | Reading |
|---|---|---|
| Requests (attempts) | **377** (377/h) | 29/tick × 13; **29 = 16 near + 13 regional** for a dispersed set — the concentric design's structural floor |
| Within-tick cache hits | 3/tick (39 total) | the SoCal cluster's shared regional boxes deduping inside one tick — **sharing works** |
| Bytes (network) | **1,405,001** (1.4 MB/h) | 108 KB/tick × 13 — a **full refetch every tick** |
| **304 renewals** | **0** | ⬅ the finding — the conditional-GET path never engaged |
| Bytes 304 saved | 0 | (follows from the above) |
| Memo boxes | **29** (bound 160) | **bounded and flat** across all 13 ticks — the swarm/churn cannot grow it |
| Memo parses | 377 (29/tick) | one parse per box per tick — the body changes every tick (see below) |
| Per-tick quakes | 13, stable | the visible set is stable; the refetch is pure overhead |

### The finding — USGS defeats the conditional-GET / body-hash reuse (measured, not a bug)

The Q5 achievement — a stale cache entry revalidates with `If-Modified-Since`/`If-None-Match` and the
server answers `304` (bytes ≈ 0) — **does not engage for USGS**, and the box memo re-parses every tick.
Root cause, confirmed from the live response and body:

- USGS returns `Last-Modified`, **but it equals the response `Date`** (e.g. `last-modified: 17:44:09`,
  `date: 17:44:10`): it is the *per-request generation time*, not a stable resource timestamp. An
  `If-Modified-Since` therefore always draws a `200`, never a `304`.
- The GeoJSON body carries `metadata.generated` (also the request time), so its hash changes every
  request even when the quake set is identical — which is why the memo re-parses each tick.

So neither reuse path is possible against this endpoint; the client **does** send the conditional
request (the Q5 path is honoured), USGS just cannot answer `304`. The byte floor is therefore set by the
**refetch frequency**, i.e. the caller TTL vs the tier cadence. Today `feedTTL = 2 min < 5 min cadence`,
so every priority tick refetches. (`lifetime` is "caller TTL, else server max-age", so `feedTTL`
overrides USGS's `max-age=60`.)

**The lever (for HUM LEAD at the gate — it touches the ratified §0.3 cadence):**

1. **Align `feedTTL` with the cadence (the fire pattern, `TTL = tier cadence`).** Fire uses a 10-min TTL
   for its 10-min tier; seismic uses 2 min for a 5-min tier. Raising `feedTTL` to **5 min** lets the
   RECENT tier and shared-box reads **ride the favourites' refreshes** (a recent location sharing a
   regional box with a favourite reads it fresh, 0 bytes) — a strict improvement, no cadence change.
   *This alone does not cut the favourites' own every-5-min refetch* (the tier drives it).
2. **Trade freshness for bytes** — since "did my area shake" tolerates minutes of staleness, a longer
   `feedTTL` (e.g. **15 min**) makes the cache absorb intervening ticks, cutting the favourites' byte
   floor ~3× (one refetch per 15 min instead of per 5 min). Data up to ~15 min stale — well within the
   §0.3 argument. Equivalent alternative: lengthen the seismic **cadence** itself (e.g. 10/30 min).

**Recommendation:** apply (1) now (`feedTTL → 5 min`, fire-aligned, low-risk); decide (2) at the gate —
I recommend a **10-min `feedTTL`** as the balance (favourites refetch every other tick, RECENT rides
them), keeping the ratified 5/15-min *cadence* untouched. Held for HUM LEAD's call.

**Context (is this a regression?):** the mandate is "no regression to the M2 request floor / bytes
recorded, conditional-GET honoured." The conditional-GET path **is** honoured (the server can't 304);
bytes are recorded. 1.4 MB/h is the 16-favourite worst case (all at 5-min); the production 10-favourite +
50-RECENT mix is lower (RECENT at 15-min, many boxes kept warm by favourites). The frame budget is
untouched. No correctness regression; the byte floor is a tuning decision, above.

## 5. Docs touched

- `docs/where-things-happen.md` — "A seismic request is made" row + the "box memo" vocabulary entry.
- `app/testdata/cadences.md`, `app/testdata/declset.txt` — regenerated (new tiers; new `seismicProviders`/`seismicRules`).
- `04-development/p2-build-log.md` (this file); `07-readiness/p10-p2.json`, `07-readiness/counters-p2.log`.
- `architecture.md` / `extending.md` full seismic pass is **P3** (with the detail section and config wiring).

## 6. Carried forward

- **P3**: the SEISMIC detail section from the approved mock (○●◉ violet), `[seismic]` config knobs surfaced,
  the `[S]` gauge label, attribution, `architecture.md` / `extending.md` / the flow map.
- **P4**: radio narration (R6, last). **P5**: REVIEW + VALIDATE + release `0.11.0`.
