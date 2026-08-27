# P1 build log — Seismic data layer (DATA FIRST)

**Batch:** P1 of the seismic PLAN (`03-architecture-design/plan.md` §5). **SEV-0** · FULL TDD.
**Branch:** `feature/seismic-data`. **Status:** APPROVED 2026-08-27 — the two package density exemptions ratified; GO for P2.

## 1. What landed (junior-first)

The data layer for earthquakes, before any UI — the fourth hazard domain beside weather, marine and
fire, built the same way.

- **The snapshot carries quakes.** `platform/snapshot` gains `KindSeismic` (a fetch kind),
  `SeismicState { AsOf, Quakes }` on `PartialData`, and `Quake` (magnitude, place, depth, time, and —
  computed, not from USGS — distance and bearing *from the tracked location*; plus tsunami/PAGER/felt).
  `AsOf` zero means the feed has not answered (**unavailable**), never "no quakes" — the FireState
  precedent so a cold feed never reassures falsely.
- **The graduated rule is a pure step function** (`domains/seismic/rules.go`): `RadiusMiFor(mag)` returns
  the miles a quake of that magnitude is shown within — the first band whose upper magnitude exceeds it
  — and `Keep(mag, distanceKm)` is the whole rule. The ratified bands (incl. HUM LEAD's M4.0→40 mi) are
  the defaults; `Valid()` refuses a non-ascending or empty ruleset. `Sort` orders **largest magnitude
  first, then nearest** (HUM LEAD 2026-08-27).
- **The USGS provider** (`domains/seismic/usgs`) is a `snapshot.Provider`: it queries the FDSN circle
  around each location (widest band radius, the lookback window), decodes the GeoJSON, applies the
  graduated rule and the `types` allowlist, computes distance/bearing with `platform/geo`, sorts and
  **caps at 20** (a swarm cannot grow the section or a memo — P10-03). Keyless, public-domain,
  attribution carried.
- **`[seismic]` config** (`platform/config`): `enabled`, `lookback_days`, `types`, `radius_bands_mi`,
  all defaulted to the ratified rule (`WithDefaults`), mirroring `[fire]`.

## 2. Tests first (the pins)

| Pin | Guards |
|---|---|
| `TestRadiusMiForEveryBand` | the step function at all 18 boundary points (0.99/1.0, 3.49/3.5, 3.99/4.0, …) |
| `TestKeepGraduatedRule` | the objectives' worked cases; the band edge is inside (≤) |
| `TestMaxRadiusAndValidity` | widest reach = 1000 mi in km; non-ascending / empty refused |
| `TestSortLargestThenNearest` | magnitude desc, then distance asc for ties (HUM LEAD) |
| `TestTypesFilterIsConfigurable` | earthquake-only default; `types` extends to blast/explosion (D4) |
| `TestFetchAppliesTheGraduatedRuleAndSorts` | the provider filters + sorts + computes distance/bearing/depth; the query covers the widest band and the window |
| `TestAsOfDistinguishesNoneFromUnavailable` | answered-but-empty (AsOf set, 0 quakes) vs down (frag.Err, no state) |
| `TestFetchCapsASwarm` | 40 in-band quakes → capped at 20 |
| `TestLiveUSGSAnswers` (`WATCHPOST_LIVE=1`, CI-skipped) | the real FDSN feed answers and decodes |

**Live proof (Ridgecrest, run 2026-08-27):** M4.0 & M3.5 at 52 km NNW, M1.8 at 14 km ENE, M1.1 at
15 km WSW — the near-field bands correctly surface the small local quakes, largest-first.

## 3. Performance (the mandate)

- No render path touched (P1 is data only): `make alloc-budget` **unchanged** (tty 504/8,445).
- The per-location query is P1's simplest correct form; **P2 swaps it for shared canonicalised boxes**
  (Approach B) — the decode/filter/cap core is already factored (`stateFor`) so P2 reuses it whole.
- The cap (20) and the future box memo are the stated bounds.

## 4. Gate

| Check | Result |
|---|---|
| `go test ./domains/seismic/...` | green (unit + live proof) |
| `make verify` | ALL GATES GREEN |
| `make alloc-budget` | unchanged |
| `make p10` | **0 live · 0 unmatched** · ledger 111 = 55 kit + **56 non-kit** (§5) · snapshot `07-readiness/p10-p1.json` |
| snapshot declset re-captured (new types — intentional) | green |
| schema (`KindSeismic`/`Quake` in the JSON schema) | green |

## 5. Decision for HUM LEAD (the one exemption)

Non-kit exemptions rise **54 → 56**: two package P10-05 (invariant-density) rows for the new
`domains/seismic` and `domains/seismic/usgs` packages. Both are the **pure-helper-alongside-a-guarded-path**
pattern the quality pass ratified for `term`/`httpx`/`render` and every `domains/fire/*` subpackage
already carries: the arithmetic (the step function) and the GeoJSON decode/format are pure, while
`Valid()` and `Fetch()` hold the real invariants. Padding them with no-op checks is exactly what
P10-05's intent forbids. **RATIFIED by HUM LEAD 2026-08-27** ("ratified; GO 4 P2") — M-P10 restated to ≤ 56.

## 6. Carried forward

- **P2**: the `KindSeismic` scheduler tiers (5 min / 15 min, §0.3 freshness row) and the shared
  canonicalised-box request (Approach B) with a bounded parsed memo + gauge; the byte/request check.
- **P3**: the SEISMIC detail section from the approved mock (○●◉ violet), config wiring, docs.
- **P4**: radio narration (R6, last). **P5**: REVIEW + VALIDATE + release `0.11.0`.
