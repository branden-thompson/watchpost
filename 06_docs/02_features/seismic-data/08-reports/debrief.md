# Seismic data — DEBRIEF (After Action Report)

**Feature:** `seismic-data` → **shipped 0.11.0** (2026-08-27, `v0.11.0`, CI green). **SEV-0.**
**Flow:** DISCOVER → PLAN → BUILD (P1–P4) → REVIEW → VALIDATE → SHIP → REFLECT.

## What shipped

The fourth hazard domain: USGS earthquakes near a tracked location, filtered by a magnitude-graduated
distance rule. A SEISMIC section in Location Details (○●◉ felt-band ramp, whole list), a strongest-quake
row mark on the main table, a radio Seismic Activity report, and a `[seismic]` config. Requests are
shared canonicalised boxes; the frame budget is unchanged.

## What went well

- **DATA FIRST, literally.** Live-probing USGS at P2 (not guessing) revealed the single wide
  `minmagnitude=0` query pulls ~1 MB of low-magnitude events the rule discards. That measurement — not a
  hunch — produced the concentric near/regional design (4–31 KB/box). Measuring before designing changed
  the architecture.
- **Measurement drove the cadence too.** The 1-hour counters run surfaced that USGS can't answer a
  conditional GET (per-request `Last-Modified`/`generated`), so 304s never fire. That finding drove the
  `feedTTL` decision (HUM LEAD chose A: 10 min) rather than assuming the Q5 revalidation win applied.
- **The SEV-0 red-team earned its keep.** Five adversarial reviewers found **real bugs the green test
  suite missed**: a band-edge radius truncation, a partial-fetch data-loss path, negative-magnitude
  exclusion, and a hostile-value modal-tear. Each is now pinned by a regression test.
- **Reused seams paid off.** Fire's `FireReport`/`FireSegments`, the assembler `FireFor`, the FIRMS
  tile-memo shape, `render.Opts.Glyphs`, and the detail/narration patterns meant seismic added no new
  infrastructure — it rode the seams the quality pass hardened.

## What to carry forward (lessons)

1. **Test the exact boundary, not near it.** The equivalence test probed `edge×0.8` and `×1.2` and passed
   green while a real Keep-visible quake in the `(32.0, 32.187] km` annulus was silently dropped by
   `%.0f` radius truncation. A boundary test must hit the boundary. (Fixed: `math.Ceil` + an exact-edge
   test.)
2. **Consistency across surfaces needs an explicit cross-check.** The radio felt-likelihood and the
   screen felt label diverged (one keyed to the glyph energy ramp at ≥5.0, the other to the felt bands at
   ≥4.5) even though `render.SeismicLevel` was a shared owner — because they were *different* concepts
   sharing one function. The red-team caught the M4.5–5 mismatch. Lesson: when two surfaces describe the
   same thing, pin them against each other, not just against a shared helper.
3. **Best-effort partial results can silently under-report.** The partial-fetch path replaced a complete
   last-good state with one leg's subset. "Publish what we got" is wrong when the pieces are disjoint
   bands; "keep last-good unless whole" is the honest rule. (Fixed.)
4. **Validate untrusted numeric inputs at the boundary.** Even a trusted TLS endpoint deserves a
   plausibility clamp — an unbounded `mag`/`depth` tore the modal frame. Cheap defense-in-depth. (Fixed.)

## Follow-ups (open)

- **Linux relay half of R6** — HUM LEAD to run `WATCHPOST_LIVE=1 go test ./app -run LiveRelay` on Arch.
- **Detail-modal memo** (noted in REVIEW D1) — the modal rebuilds every ~300 ms tick while open; bounded
  at 20 rows and off the hot path, but a memo would remove the per-tick rebuild for all sections. Future
  optimization, not seismic-specific.
- **0.12.0 candidate** — the deferred global severe-weather + significant-quake ticker (its own DISCOVER).

## By the numbers

- 5 batches (P1–P5), 5 releases-worth of work in one minor; ~30 tests added across rules, provider,
  assembler, detail, narration, and the R6 smokes/soak.
- Red-team: 5 reviewers, 13 findings (8 fixed + regression tests, 5 accepted with reason), 0 open.
- Gates at ship: verify green · p10 0 live/0 unmatched, ledger 111 (no new exemptions since P1) ·
  alloc-budget within (now measures the seismic mark) · race clean · 1-hour soak flat · CI green.
