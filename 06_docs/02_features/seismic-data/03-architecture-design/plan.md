# Seismic data — Plan of Record (FULL PLAN, SEV-0)

**Feature:** `seismic-data` → `0.11.0` · **Phase:** PLAN · **Date:** 2026-08-27
**Branch:** `feature/seismic-data` · **Inputs:** `01-objectives/objectives.md`, `02-analysis/data-shape.md`,
`08-reports/discover-report.md` (all decisions D1–D5 ratified).
**Status:** APPROVED 2026-08-27 (mock approved; seismic glyph/colour must not clash with fire — circle
ramp ○●◉ in violet, recorded §7). **BUILD P1 (data layer) proceeding**; P3 renders the section and
takes HUM LEAD's colour direction.

## 0. Principles that bind every batch (inherited from the quality pass)

The performance mandate is a gate, not a hope. Every batch carries these:

1. **DATA FIRST** — the snapshot payload, the provider and the rule land and are unit-pinned before a
   single row is rendered. The UI reads a proven data layer.
2. **FULL TDD** — a behaviour gets its test first; a golden is captured and only changed with a stated
   reason.
3. **Performance is measured, not assumed** — the frame allocation budget (`make alloc-budget`) must be
   **unchanged** (the seismic section renders only inside the Details modal, off the hot table path —
   §3); any new memo/cache states its bound and reports a **gauge**; USGS `max-age=60` and the Q5
   conditional-GET path are respected; the **byte/request check** on the counters apparatus shows the
   seismic tier within a stated per-hour floor (no regression to M2).
4. **Bounds stated (P10-03)** — the per-location quake list is **capped**; any shared-box parse memo is
   bounded and gauged; `make p10` stays **0 live · 0 unmatched**, non-kit ledger unchanged unless HUM
   LEAD restates a row at a gate.
5. **Junior-first docs** — a per-file header, `docs/where-things-happen.md` rows, `architecture.md` and
   `extending.md` updated, a `04-development/pN-build-log.md` per batch with before/after and a
   docs-touched line.
6. **R6 (radio is sacred)** — the narration batch (P4) ends with the Synth **and** Relay pty smokes and
   a 1-hour soak before its gate; the earlier batches must not touch `domains/radio` at all.
7. **Fix-forward release** — `0.11.0` cut from `main-publish` via the publish protocol; `CHANGELOG`
   keeps `[Unreleased]` between batches; identity verified before any outward action.

## 1. Targets

| Target | Value | How gated |
|---|---|---|
| Frame allocation (133×44, memo hit/miss) | **unchanged** from `0.10.x` (504 / 8,445 allocs) | `make alloc-budget`; the seismic section is modal-only, not in `body`/the table memo |
| Seismic requests/hour (10 + 50 locations) | ≤ a stated floor: shared boxes × the tier cadence; **measured** at the gate | `counters.json` per host (`earthquake.usgs.gov`), Q0-style |
| Bytes/hour | recorded; conditional-GET / `max-age` honoured | `NotModified`, `Bytes304`, `BytesNet` |
| Parsed-box memo | bounded (≤ the distinct boxes a location set touches), LRU, gauged | `seismic.memo.*` gauge; unit-pinned |
| Per-location quakes | capped at **N** (default 20), summarised beyond | unit test |
| P10 | 0 live · 0 unmatched · ledger unchanged | `make p10` snapshot per batch (`07-readiness/p10-pN.json`) |
| Detail goldens | colour-off **and** ASCII, byte-pinned | `modes/tty` goldens |
| Radio (P4) | Synth PLAYING + Relay + 1 h soak flat | pty smokes, `soak.sh` |

## 2. The data model (P1)

Mirror the fire domain exactly — a fourth hazard beside weather / marine / fire.

- **`platform/snapshot`**: `KindSeismic` (a new `FetchKind`); `SeismicState { AsOf time.Time; Quakes []Quake }`
  on `PartialData`; `Quake { Mag float64; MagType, Place string; DepthKm float64; At time.Time;
  DistanceKm float64; Bearing string; Tsunami bool; Alert string; Felt *int; Sig int;
  Source SourceInfo }`. `AsOf.IsZero()` ⇒ **unavailable** (feed cold/down), distinct from
  `len(Quakes)==0` ⇒ **no recent activity** (the FIRE `AsOf` precedent, red-team B5 P3). Merged by the
  assembler like fire.
- **`domains/seismic`**: `Rules` (the ratified bands as an ascending `[]Band{UpperMag, RadiusMi}`,
  `LookbackDays`, `Types []string`) with `Keep(mag, distanceMi) bool` — the pure step-function filter —
  and `RadiusMiFor(mag) float64`. Defaults = §objectives table. Unit-pinned at every boundary
  (m=0.9/1.0/1.1, 3.4/3.5/3.6, … distance just inside/outside each band).
- **`domains/seismic/usgs`**: a `snapshot.Provider` (`ID "usgs"`, `Domains ["seismic"]`,
  `Fetch(KindSeismic)`): build the FDSN query URL (widest band radius, lowest band's magnitude floor,
  `starttime = now − LookbackDays`, `format=geojson`), `GetJSON` (read-only), decode, filter each
  location's box by `Keep` + `Types`, compute distance/bearing (`platform/geo`), cap, sort
  (nearest-first or significance-first — §7), return one `SeismicState` per location. Attribution:
  "USGS Earthquake Hazards Program (earthquake.usgs.gov)", public domain.

## 3. Request architecture — three approaches (recommend **B**)

Earthquakes are regional; a 1,000 mi box (top band) around two nearby cities overlaps almost entirely.

- **A — per-location query.** One FDSN request per location per tier tick. Simplest; up to 60
  requests/tick. Rejected on the performance mandate (needless duplication, the Q5 lesson).
- **B — shared canonicalised box (RECOMMENDED, the Q5 FIRMS-tile precedent).** Snap each location's
  query circle to a **fixed grid box** (e.g. 5°); the box URL is the cache + singleflight key, so every
  location in a box shares one request; a location near an edge fetches the ≤ 4 boxes it touches. The
  parsed box is memoised by body hash (bounded, gauged). `Keep`/distance decide membership afterward, so
  the quakes a location sees are **identical** to the per-location query (the equivalence property).
  One request serves many locations; bytes small.
- **C — one national/global feed.** Fetch a USGS summary feed (`2.5_day`) once and filter locally.
  Fewest requests but the feed is magnitude-floored (misses the sub-M2.5 near-field bands the rule
  wants) and unbounded in payload during a swarm. Rejected for 0.11.0; noted for the 0.12.0 ticker,
  where a global feed is exactly right.

**Decision: B.** It reuses `fire`'s box/tile machinery (`fire.Bounds`, the tile memo shape) and meets
the mandate with the Q0 transaction count deciding the box size at the P2 gate.

## 4. Cadence (P2, §0.3 freshness argument)

USGS serves `max-age=60`; a person asking "did my area shake" does not need sub-minute latency. Tier
`Every` = **5 minutes** on the priority pipeline (favourites) and **15 minutes** on the RECENT
pipeline — the fire cadence exactly (they share the regional nature). Recorded with the M2 floor
re-derived in the P2 build log; the client cache + shared box keep the network near zero at steady
state.

## 5. Batches (TDD order, each a gate)

- **P1 — data layer (DATA FIRST).** snapshot `KindSeismic` + `SeismicState`/`Quake`; `seismic.Rules`
  (the step function, boundary-pinned); `domains/seismic/usgs` provider (query build, decode, filter,
  cap) against `httptest` fixtures captured from the live probe; equivalence + `AsOf` unavailable/none
  tests; a `WATCHPOST_LIVE` real-USGS proof test (skipped in CI). **Gate:** unit green, P10 snapshot,
  no UI yet.
- **P2 — scheduler + shared-box requests.** `KindSeismic` tiers in both pipelines (the §4 cadence with
  its freshness row); the canonicalised-box request (Approach B) with the parsed memo bound + gauge;
  the Q0-style transaction count → box size; conditional-GET honoured. **Gate:** a 1-hour counters run
  showing the seismic request/byte floor, memo bounded, P10 snapshot.
- **P3 — detail section + config + docs.** `modes/tty/detail_seismic.go` from the **approved mock**
  (§7), colour-off + ASCII goldens; the "no recent activity" / "unavailable" states; `[seismic]` config
  → `seismic.Rules` (`seismicRules` sibling of `fireRules`); wire the provider into the app pipelines
  (`fireProviders` sibling) and the `[S]` gauge; attribution credit; `architecture.md`, `extending.md`,
  the flow map. **Gate:** goldens, `make verify`, `make alloc-budget` **unchanged**, P10, `a2dh
  validate` — this is the "data wired / loaded / displaying correctly" milestone HUM LEAD named for D1.
- **P4 — radio narration (R6, LAST — D1).** The synth broadcast reads recent felt-band quakes for the
  location (HUM LEAD provides the scripts); `domains/radio/synth` compose additions; the felt-likelihood
  wording. **Gate:** Synth **and** Relay pty smokes + a 1-hour soak flat; P10; goldens for any
  compose change.
- **P5 — REVIEW + VALIDATE + release.** SEV-0 red-team pass (axes: the graduated rule's correctness and
  boundaries, the shared-box equivalence, the swarm/cap bound, the `AsOf` honesty, secret-free, R6);
  disposition all findings; the byte/request check; `0.11.0` published via the protocol; DEBRIEF.

## 6. Risks / mitigations

| Risk | Mitigation |
|---|---|
| A swarm (Ridgecrest) spikes a location's count | the cap (N=20) + "+K more, largest M+" summary; the memo is per-box, bounded |
| Blasts/explosions near cities | `types` filter defaults to `["earthquake"]` (D4); extend by config |
| Frame regression | the section is modal-only; `alloc-budget` gate proves the table path is untouched |
| Byte/request regression | shared box + client cache + conditional GET; measured at the P2 gate |
| Depth ignored by the rule (a deep M5 shows as "felt") | v1 is magnitude-only per D2; depth is **shown** so the reader judges; depth-weighting noted for later |
| R6 | P1–P3 do not touch `domains/radio`; P4 is the only radio batch, with the full R6 gate |

## 7. The SEISMIC section — ASCII mock (for HUM LEAD to direct — feedback-mock-fidelity)

Draft, in the Location Details modal, beside the FIRE section. The felt-band is the left label; a
tsunami/PAGER event reads emphasised. **Direct me on the exact columns, order, and wording.**

```
     SEISMIC │ 3 nearby in the last 7 days                                    (USGS)
             │
             │  ● M4.2  12 mi NE   depth 8 km    2h ago   Might feel it
             │  ○ M2.8   4 mi SSW  depth 3 km    1d ago   Below feeling
             │  ◉ M5.1  88 mi N    depth 15 km   3d ago   Almost certainly felt
             │
             │  quiet since — no quakes in reach of this area
```

- **Glyph/colour — DISTINCT from fire (HUM LEAD 2026-08-27).** Fire owns the orange ◆; seismic uses a
  **circle-family ramp** (seismic energy radiates in circles): **○** below-feeling · **●** felt · **◉**
  significant, in a new **violet** `render.SeismicMark` token (default proposal `141` / medium purple —
  distinct from fire-orange 208, alert-red 196, advisory-yellow 220). A `tsunami`/PAGER-alert quake
  reads in the warning tone (red) regardless of band. HUM LEAD directs the exact colour at P3.
- "no recent seismic activity" when the feed answered and nothing matched; "seismic data unavailable"
  when `AsOf` is zero (cold/down) — never a blank.
- `--ascii`: ○/●/◉ → `.`/`o`/`O` from `Opts.Glyphs` (the A11-10 seam), one owner.

## 8. Fit and non-regression

Seismic adds no new infrastructure — it rides the snapshot → scheduler → detail seams the quality pass
hardened, reuses `fire`'s box/memo/`geo` machinery and the `render` seam, and is gated on the same
allocation/counter/P10 budgets. The performance mandate is met by construction (modal-only render,
shared bounded requests) and proven at each gate.
