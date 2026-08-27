# P4 build log — Seismic radio narration (R6, the last batch)

**Batch:** P4 of the seismic PLAN (`03-architecture-design/plan.md` §5). **SEV-0** · FULL TDD · **R6**.
**Branch:** `feature/seismic-data`. **Status:** at gate — Synth + Relay (macOS) smokes green; the 1-hour
soak runs before the final gate; **ready for HUM LEAD synth UAT**.

## 1. What landed (junior-first)

The synth broadcast now reads a Seismic Activity report, from HUM LEAD's script.

- **`domains/radio/synth/seismic.go`** — a `FireReport`/`FireSegments` sibling:
  - `SeismicReport { Known; State; Lat, Lon }`; `SeismicSegments(location, sr, imperial, now)`.
  - **Skipped entirely** when the location has no quakes (HUM LEAD: plays only if there are entries) —
    an unknown *or* answered-empty state returns no segments.
  - **The script:** the USGS notice (+ the delay/safety line), a two-second pause, the count
    ("There has been 1…" / "There have been N…"), the **strongest three** quakes read largest-first, an
    overflow line ("…and N more recent quakes, which can be found in the <location> details report in the
    Watchpost CLI application view."), and the where-to-learn-more tail.
  - **Per quake:** *"A magnitude 5.1 earthquake, 88 miles north of your location, at a depth of 15
    kilometers, recorded 3 days ago. A quake of this magnitude has a strong likelihood of being felt when
    it occurs."* Depth is always in kilometres (seismology convention). Missing fields are omitted, never
    guessed. The felt-likelihood is a **three-tier** (low / moderate / strong) keyed to `render.SeismicLevel`
    — the same one owner as the glyph and the row mark.
  - The outro URL reads through the existing web-address machinery (UAT 95): *"earthquake dot usgs dot gov
    slash earthquakes slash map."*
- **Wiring:** `Compose` gains a `SeismicReport` (the seismic report plays after the fire report, before
  the sign-off); the deck gets a `seismic` hook (`lp.seismicFor` → `seismicReportOf`); the assembler gets
  the narrow read `SeismicFor(ref)` (no per-cycle snapshot clone), mirroring `FireFor`.
- **Detail section shows the whole list (HUM LEAD, P4):** the 3-row cap is gone — the SEISMIC section now
  lists every quake (the provider still caps at 20), because the broadcast sends listeners there for 4–N.

## 2. Tests first (the pins)

| Pin | Guards |
|---|---|
| `TestSeismicSegmentsReadTheScript` | the notice, pause, count and per-quake sentence, verbatim |
| `TestSeismicSkippedWithoutEntries` | unknown and answered-empty ⇒ no report |
| `TestSeismicCountAndOverflowCapAtThree` | count = full total; strongest 3 read largest-first; overflow line names the remainder + location |
| `TestFeltLikelihoodTiers` | low (<3.5) / moderate (3.5–5.0) / strong (≥5.0) |
| `TestQuakeSentenceOmitsMissingFields` | no depth / no bearing ⇒ those clauses left out |
| `TestSynthPlaysTheSeismicBroadcast` | **R6 Synth smoke:** the composed broadcast renders to PCM end-to-end; the seismic report plays before the tail; notice + quake + tail all reach the voice |
| `TestSeismicBroadcastSoak` (`WATCHPOST_SEISMIC_SOAK=1`) | **R6 soak:** goroutines/heap flat under continuous playback |

## 3. R6 gate

| Check | Result |
|---|---|
| `make verify` | ALL GATES GREEN |
| full suite (`go test ./...`) | green |
| `make alloc-budget` | unchanged (no render-path change; the section is modal-only) |
| `make p10` | **0 live · 0 unmatched** · ledger 111 — **no new exemptions** |
| **Synth smoke** | `TestSynthPlaysTheSeismicBroadcast` green (PCM rendered, order correct) |
| **Relay smoke (macOS)** | `WATCHPOST_LIVE=1 … LiveRelay` green — wxradio.org **and** weatherusa.net both played audio |
| **Relay smoke (Linux/Arch)** | HUM LEAD runs `WATCHPOST_LIVE=1 go test ./app -run LiveRelay` on Arch (the Linux half) |
| **1-hour soak** | `WATCHPOST_SEISMIC_SOAK=1 WATCHPOST_SOAK_MINUTES=60 go test ./domains/radio/synth -run Soak` — running; 2-min smoke flat (goroutines 6→6, heap ~4 MB) |

## 4. HUM LEAD synth UAT

Build and listen: `make build` → `./dist/watchpost`. Track a location with recent quakes (e.g.
**Ridgecrest, CA** or **Anchorage, AK**), let the seismic tier populate (≤5 min), then play the radio in
**Synth** mode for it — the Seismic Activity report reads after the fire report. Adjustments expected
(HUM LEAD may tune wording); fix-forward.

## 5. Carried forward

- **P5** — REVIEW (SEV-0 red-team) + VALIDATE + release `0.11.0` + DEBRIEF.
- Open confirmations for HUM LEAD at UAT: the three-tier felt-likelihood wording (low/moderate/strong),
  and the `SeismicMark` colour (kept violet 141).
