# P3 build log — Seismic detail section + view seam + docs

**Batch:** P3 of the seismic PLAN (`03-architecture-design/plan.md` §5). **SEV-0** · FULL TDD.
**Branch:** `feature/seismic-data`. **Status:** at gate — this is the "data wired / loaded / displaying
correctly" milestone HUM LEAD named (D1). The mock is reproduced; colour direction is HUM LEAD's to adjust.

## 1. What landed (junior-first)

The earthquakes P1/P2 fetched now **show** — a SEISMIC section in Location Details, beside FIRE.

- **The view seam (the missing half of DATA FIRST).** P1 added the fetch side (`PartialData.Seismic`);
  P3 adds the *assembled* side the UI reads: `Location.Seismic *SeismicState`, and the assembler merge.
  Seismic has one provider, so there is no cross-provider fold like fire's — the assembler keeps the
  latest state per location (`a.seismic[k]`, dropped on prune) and **deep-copies** it into each published
  snapshot (`platform/snapshot/merge_seismic.go:cloneSeismic`, so nothing aliases assembler state).
- **The SEISMIC section** (`modes/tty/detail_seismic.go`, a `fireRows` sibling wired into `detailLines`):
  recent quakes largest-then-nearest, each row *`glyph  M{mag}  {dist} {bearing}  depth {d} km  {age} ago
  {felt label}`*. The felt-likelihood glyph is a **circle ramp** — seismic energy radiates in circles,
  distinct from fire's orange ◆:
  - **○** below feeling (M < 3.5) · **●** felt (M 3.5–5.0: "Might feel it" / "Almost certainly felt") ·
    **◉** significant (M ≥ 5.0), in a new violet **`render.SeismicMark`** token (default `141`).
  - A **tsunami** or an **orange/red PAGER** quake reads in the warning tone (`AlertDanger`) regardless of
    magnitude, and its label carries the reason ("Tsunami — …").
  - `--ascii` maps ○●◉ → `.` `o` `O` (the A11-10 glyph seam).
- **The honest states** (the FIRE `AsOf` precedent): a cold/down feed reads **"seismic data
  unavailable"**; an answered-but-empty feed reads **"no recent seismic activity"** — never a blank or a
  reassuring fake "none".
- **Config, attribution, docs.** `[seismic] lookback_days` words the header ("last N days") through one
  owner (`SeismicDays`, the `FireBoldMW` pattern — `seismicRules(cfg.Seismic)`); the USGS credit joins the
  About window (`credits.go`, trimmed to ≤ 52 cells); `where-things-happen.md`, `extending.md` (a second
  worked example — a keyless hazard with its own section) and `architecture.md` updated.

## 2. Tests first (the pins)

| Pin | Guards |
|---|---|
| `TestDetailSeismicSectionShowsGraduatedRows` | header, `(USGS)`, the ○●◉ ramp + felt labels, depth/age, "and N more" fold, largest-first, the SeismicMark tone |
| `TestDetailSeismicUnavailableVsQuiet` | cold/nil ⇒ "unavailable"; answered-empty ⇒ "no recent seismic activity" |
| `TestDetailSeismicASCIIGlyphs` | `--ascii` renders `.oO` and never emits a unicode glyph |
| `TestDetailSeismicTsunamiReadsWarning` | a tsunami quake labels the reason and reads in the warning tone, not the violet mark |

Rendered against the approved mock (colour-off):

```
   SEISMIC │ 3 nearby in the last 7 days                                (USGS)
           │   ◉ M5.1  88 mi N    depth 15 km  3d ago  Significant
           │   ● M4.2  12 mi NE   depth 8 km   2h ago  Might feel it
           │   ○ M2.8   4 mi SSW  depth 3 km   1d ago  Below feeling
```

## 3. Performance (the mandate)

- **Frame allocation:** `make alloc-budget` **unchanged** — the section renders only inside the Details
  modal, off the hot table/`body` path (the plan's non-regression argument, proven by the gate).
- No new per-frame state; the snapshot deep-copy runs once per publish, bounded by the capped list (20).

## 4. Gate

| Check | Result |
|---|---|
| `go test ./...` | green (new detail tests + regenerated goldens) |
| `make verify` | ALL GATES GREEN |
| `make alloc-budget` | unchanged |
| `make p10` | **0 live · 0 unmatched** · ledger **111** — **no new exemptions** (the one new finding, an unused `seismicHead` param, was **removed**, not exempted) · snapshot `07-readiness/p10-p3.json` |
| `a2dh validate` | 100 % (18/18) |
| goldens regenerated (intentional) | snapshot + tty declsets (new seismic decls), `render` declset (`SeismicMark`), `pkg/schema` (`make schema`: the new `seismic` block) |

**Schema:** `Location` gains a `seismic` block — a new top-level section, so the generator + golden were
regenerated (still `v1.0.0-rc`; an additive, nullable block is backward-compatible).

## 5. Docs touched

- `modes/tty/detail_seismic.go` (+ header), `platform/snapshot/merge_seismic.go` (+ header).
- `docs/where-things-happen.md` (the SEISMIC-render row), `docs/extending.md` (the second worked example),
  `06_docs/.../watchpost-cli/03-architecture-design/architecture.md` (FetchKind list, `SeismicState`).
- `04-development/p3-build-log.md` (this file); `07-readiness/p10-p3.json`.

## 5b. Addendum — the main-table row mark (HUM LEAD 2026-08-27)

HUM LEAD **reversed the objectives' "details-only" call** and asked for the seismic glyph on the main
table too (objectives §3 updated). The strongest recent quake's felt-band glyph — **one glyph, no count**
— joins the marks block between the play and fire marks:

```
›  ▶ ● 5◆ 3⚠ 009. Ridgecrest, CA   …      (a felt quake, playing, 5 fires, 3 alerts)
     ◉       010. Los Angeles, CA  …      (a significant quake, nothing else)
             011. Quiet Town, KS   …      (no recent quake — no mark)
```

- **The marks block grew 11 → 13 cells** (glyph + spacer), absorbed by NAME's floor
  (`marksW + nameMinW = 30`, the UAT-110 invariant), so **every column from LABEL@37 on keeps its
  offset** — only `###.`@13 and `NAME`@18 shift, and the table's minimum content width rises 50 → 52.
- **One owner for the ramp:** `render.SeismicLevel(mag)` (1 ○ / 2 ● / 3 ◉) and `Glyphs.Seismic` (○●◉ /
  `.oO`) are shared by the table mark and the detail section — they cannot disagree. `render.SeismicMark`
  tone (violet). The strongest quake is `Quakes[0]` (the list is sorted largest-first).
- **Gates:** `alloc-budget` **unchanged** (the marks array is fixed-size; the row string grew two cells
  but stays within budget); the offset tests updated to the new spec; the frame goldens regenerated (the
  `-update-golden` reason: the marks block widened for the seismic glyph); `TestSeismicMarkIsStrongestQuakeGlyph`
  pins the ramp, the tone, the position and the empty case; help legend + `where-things-happen` updated;
  p10 0 live / 0 unmatched, **no new exemptions**.

## 6. Carried forward

- **P4** — radio narration (R6, LAST): HUM LEAD provides the scripts; the synth reads recent felt-band
  quakes; full Synth **and** Relay pty smokes + a 1-hour soak before its gate.
- **P5** — REVIEW (SEV-0 red-team) + VALIDATE + release `0.11.0` + DEBRIEF.
- **Colour:** `SeismicMark` defaults to violet `141`; HUM LEAD directs the exact value (a separate pass).
