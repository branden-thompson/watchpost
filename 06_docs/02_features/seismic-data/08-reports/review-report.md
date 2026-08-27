# Seismic data — REVIEW report (P5, SEV-0 red-team)

**Feature:** `seismic-data` → `0.11.0` · **Phase:** REVIEW · **Date:** 2026-08-27
**Method:** adversarial red-team across the plan §5 axes, run as parallel independent reviewers plus
direct verification (race detector, live USGS re-proof, schema/null-parity). Every confirmed finding was
fixed with a regression test, or accepted with a stated reason.

## Axes covered

| Axis (plan §5) | How |
|---|---|
| Graduated rule correctness & boundaries | reviewer A (rule/equivalence) |
| Shared-box equivalence | reviewer A |
| Swarm / cap bound | reviewer B (concurrency/bounds) + reviewer D (performance) |
| `AsOf` honesty | reviewer C (narration/display) + schema/null-parity check |
| Secret-free | reviewer C |
| R6 (radio) | reviewer C + the Synth/Relay smokes + the soak |
| **+ hostile/malformed feed & terminal-escape injection** (added) | reviewer E |
| **+ performance regression under swarm** (added) | reviewer D |

## Findings and dispositions

### Fixed (with regression tests)

| # | Finding | Fix |
|---|---|---|
| A1 | `%.0f` truncated the near-query radius (`32.187 km → "32"`), dropping a Keep-visible quake in the `(32.0, 32.187] km` annulus at a band edge | Round the query radius **up** (`math.Ceil`) — `usgs.go:queryURL`. `TestBandEdgeQuakeIsNotTruncated` |
| A2 | Negative-magnitude quakes underfoot were Keep-visible but excluded by the near floor of 0 (USGS `ml` runs negative; the sub-1.0 band shows within 3 mi) | Near floor dropped below any real quake — `rules.go:FloorMag` returns −9. `TestNegativeMagnitudeUnderfootIsShown` |
| B1 / A-secondary | **Partial-fetch data loss**: when one concentric leg failed, the partial state replaced complete prior state, silently dropping the other leg's whole band (e.g. a known distant M5 vanishing on a regional-leg blip) | A location keeps last-good unless **every** leg succeeds — `usgs.go:gather`/`Fetch`. `TestPartialFetchDoesNotPublish` |
| B2 | `cloneSeismic` did not deep-copy `Source`'s reference fields (`DistanceKm`, `FillFrom`) — latent isolation gap (currently nil, so not live) | Full deep copy — `merge_seismic.go:cloneSource` |
| C1 | "and 1 more recent **quakes**" — plural noun with a singular count at exactly 4 quakes | Pluralize by count — `synth/seismic.go`. `TestSeismicOverflowNounAgreesWithCount` |
| C2 | Radio said "moderate likelihood" where the screen said "Almost certainly felt" (M4.5–5.0) — the radio's 3-tier likelihood was keyed to the glyph energy ramp (≥5.0), not the felt bands | Felt-likelihood keyed to the **felt bands** (3.5 / 4.5), matching the screen label — `synth/seismic.go:feltLikelihood`. `TestFeltLikelihoodTiers` |
| C3 | A unicode em-dash leaked into the `--ascii` output in `seismicWarnLabel` | ASCII hyphen under `--ascii` — `detail_seismic.go`. Assertion added to `TestDetailSeismicTsunamiReadsWarning` |

### Accepted (with reason)

| # | Finding | Why accepted |
|---|---|---|
| B3 | `frag.Err` is one field; two locations failing in one fragment surface one warning | Observability only (each location's own data still lands; provider still degrades). **Matches the existing fire provider pattern** — not a seismic regression |
| B4 | `parseFeatures` runs under the memo mutex | Correct (prevents duplicate parses); the concentric bodies are small (4–31 KB, not the rejected ~1 MB wide query), so cold-parse contention is negligible. Same shape as the FIRMS tile memo |
| D3 | The box memo holds the full parsed (uncapped) feature list | Correct by design — different locations keep different subsets by distance; bounded by `maxBoxes=160` LRU; the swarm inflates only the tight per-location near box, never the sparse shared regional box |

### Verified clean (probed, no defect)

- **Pivot boundary / half-open guard**: near `[floor, 3.5)` ∪ regional `[3.5, ∞)` is exact-cover, disjoint; M3.5 shows once. Dedup-by-id is a correct safety net.
- **`Valid()` / `QueryPlan`**: the non-decreasing-radius invariant makes the near group a clean prefix; the `break` is sound; configs with no-near / all-near / small-max bands all stay equivalent.
- **Concurrency**: no data race (race detector clean on all seismic paths); every memo and assembler access is under its lock; `SeismicFor`/`Snapshot` clone before handing state out; the stored `a.seismic[k]` pointer aliases nothing live.
- **Bounds**: `maxBoxes` LRU holds under churn; `maxQuakes=20` cap holds; no unbounded growth in `gather`.
- **Honesty**: `AsOf` zero ⇒ "unavailable", answered-empty ⇒ "no recent activity"; a nil `Seismic` marshals `"seismic":null` (schema permits null).
- **Hot path**: the closed-frame table is memoized (`bodyKey` holds the snapshot pointer); `seismicRowLevel` is O(1); seismic cannot force a per-frame re-render.

### Hostile / malformed feed & terminal-escape injection (reviewer E)

The injection surface is **closed by non-use**, confirmed by the reviewer: the attacker-influenceable
strings `place`, `magType` and `alert` are decoded but **never rendered to the TTY frame or spoken** —
the seismic row and the broadcast are entirely computed numbers plus fixed labels (`PAGER`/`Tsunami`,
the felt words); `alert` is only lowercased-and-compared to `"orange"/"red"`. No feed string reaches
`say`/piper. NaN/Inf/out-of-range coordinates fail safe (JSON can't encode NaN; `Keep(mag, NaN)` is
false → dropped); an oversized body is refused by the httpx 32 MB cap before decode → `gather` keeps
last-good. One real break was found and fixed:

| # | Finding | Fix |
|---|---|---|
| E-A | An unbounded `mag` (e.g. `1e300`) or `depthKm` survives decode, sorts first, and renders a ~300-char spaceless token `WrapLines` can't wrap (tearing the modal) and reads a 300-digit number aloud | **Fixed** — `parseFeatures` rejects implausible magnitude (−2..10) and depth (0..1000 km) at decode. `TestImplausibleFeedValuesAreRejected` |
| E-C | Within the 32 MB cap a feed could pack ~500k features; the box memo retains the full parsed slice | **Hardened** — `parseFeatures` caps at `maxFeatures = 20000` (far above any real 7-day box), a pure hostile-payload backstop so the memo entry stays bounded |
| E-D / B4 | The JSON decode runs under the box-memo mutex | **Accepted** — matches the FIRMS tile-memo pattern; the real bodies are small (4–31 KB); off the render path; the 32 MB cap bounds the worst case. Diverging here would trade the serialize-cost for a duplicate-parse cost |

### Performance (reviewer D)

- **D2 (gate gap) — fixed:** the alloc-budget fixture (`benchLoc`) carried no hazard data, so the
  miss-path seismic mark cost was unmeasured. A recent quake was added to the fixture; the gate now
  measures the per-row glyph `Tint` and **passes within the existing budget** (miss 7,682 vs 10,546 —
  the cost fits the headroom, no re-baseline needed).
- **D1 — accepted (HUM LEAD directive):** the detail modal lists all quakes (no cap) and rebuilds each
  ~300 ms tick while open — bounded at 20 rows, modal-open only, off the closed-frame path, and
  consistent with the existing per-tick modal redraw of every section. HUM LEAD explicitly asked for the
  whole list in details (the radio points there for 4–N), so the cap is not re-added; the stale "+K more"
  comments were corrected, the row slice pre-sized. A detail-modal memo is a noted future optimization.
- **D4 — considered, reverted:** hoisting the compass tables to package level shaved a cold-path
  allocation but tripped P10-06 (no globals) and diverged from the codebase's per-call convention; the
  per-call cost is negligible on the cold radio/fetch paths, so the tables stay local.
- **Hot path verified safe:** the closed-frame table is memoized; `seismicRowLevel` is O(1); the box
  memo and quake cap are bounded; the swarm does not inflate the shared regional box.

## Gate

`make verify` GREEN · full suite green · **`make p10` 0 live · 0 unmatched · no new exemptions** ·
`make alloc-budget` within budget (fixture now exercises the seismic mark) · race detector clean ·
live USGS re-proof green · Synth + Relay(macOS) smokes green · 1-hour soak flat.

**Disposition:** 8 findings fixed with regression tests, 5 accepted with reason, the rest verified clean.
No open correctness or security defects. Ready for VALIDATE (soak) → release.
