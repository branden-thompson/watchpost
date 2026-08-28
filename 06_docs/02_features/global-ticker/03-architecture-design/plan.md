# Global event ticker — Plan of Record (FULL PLAN, SEV-0)

**Feature:** `global-ticker` → `0.12.0` · **Phase:** PLAN · **Date:** 2026-08-27
**Branch:** `feature/global-ticker` · **Inputs:** `01-objectives/objectives.md`, `08-reports/discover-report.md`
(decisions D1–D5 ratified). **Status:** PROPOSED — awaiting HUM LEAD approval + colour/tone/placement direction.

## 0. Principles that bind every batch (inherited from seismic/quality-pass)

1. **DATA FIRST** — the Event model, the three source fetchers, the dedup/stack/location-tying land and are
   unit-pinned before a pixel scrolls.
2. **FULL TDD** — behaviour gets its test first; goldens captured, changed only with a stated reason.
3. **Performance is measured** — the marquee is one row and must not regress the frame budget
   (`make alloc-budget` unchanged); the ticker fetch honours `httpx` conditional-GET/`max-age` and a
   stated per-hour byte floor; every new structure (stack, seen-cache, parsed feeds) is bounded + gauged.
4. **Bounds stated (P10-03)** — top-N events, bounded stack, bounded+pruned seen-cache; `make p10`
   0 live · 0 unmatched, ledger unchanged unless HUM LEAD restates a row at a gate.
5. **Junior-first docs** — per-file headers, `where-things-happen.md`, `architecture.md`/`extending.md`,
   a `pN-build-log.md` per batch.
6. **R6 (radio is sacred)** — the audio batch (tone + narration) ends with the Synth **and** Relay pty
   smokes + a 1-hour soak; earlier batches must not touch `domains/radio`.
7. **Fix-forward release** — `0.12.0` from `main-publish` via the publish protocol; identity verified.

## 1. Architecture — the separate pipeline (the DISCOVER finding)

The feeds are **global, not per-location**, so the ticker does **not** use `snapshot.Provider` / the
assembler (keyed by `LocationKey`). Approaches weighed:

- **A — a separate ticker pipeline (RECOMMENDED).** A `TickerFeed` fetches the three global sources on its
  own cadence, maps them to a unified `[]Event`, maintains the ordered stack + new-alert detection, and
  publishes the stack to the TTY. Reuses `platform/httpx` (cache, conditional GET, redaction, byte
  discipline), `domains/locations/geodata` (city index) and `platform/geo` (nearest-city), and the audio
  engine — but not the assembler. Clean separation; the global data never pollutes the watchlist.
- **B — synthetic "global" location in the assembler.** Rejected: a fake `LocationRef` corrupts the
  per-tracked-location semantics (watchlist caps, the detail/table, retention) the whole app assumes.
- **C — derive from existing per-location alert data.** Rejected: misses every event that isn't near a
  tracked location — the opposite of "global".

**Decision: A.** New package `domains/globalfeed` (the Event model + fetchers + stack); `app` owns the
ticker pipeline (cadence, cache, audio triggers) and the TTY renders the marquee row.

## 2. The data model (P1)

```
type Class    int   // ClassQuake | ClassTropical | ClassSevereWx
type Severity int   // SevYellow | SevOrange | SevRed  (the bg tiers)

type Event struct {
    ID       string    // stable source id — the dedup key (USGS id, NHC id, NWS alert id)
    Class    Class
    Severity Severity
    Type     string    // the spoken "<severe alert type>": "Earthquake", "Hurricane", "Tornado Warning"
    Location string    // the tied representative location (D5)
    At       time.Time // event/issue time — stack recency
    Source   string    // "USGS" | "NHC" | "NWS"
}
```

**Fetchers (`domains/globalfeed/<src>`), all keyless/public-domain, live-probed:**

| Source | Endpoint | Maps to | Severity |
|---|---|---|---|
| USGS significant | `feed/v1.0/summary/significant_week.geojson` | Class Quake, Type "Earthquake" (or "Landslide"/"Tsunami"), Location = `place` | by `mag`: ≥6.5 Red · 5.5–6.5 Orange · <5.5 Yellow (tsunami ⇒ Red) |
| NHC | `CurrentStorms.json` (Atlantic + E-Pacific) | Class Tropical, Type by `classification` (Hurricane/Tropical Storm/Tropical Depression), Location = `name` + basin | HU Red · TS Orange · TD Yellow |
| NWS national | `alerts/active?event=<curated severe list>` (probed: `event=` works; the `severity=` multi-param 400s) | Class SevereWx, Type = `event`, Location = `areaDesc` | Tornado/Hurricane/Extreme Red · Severe Tstm/Severe Orange · watches Yellow |

Reuse: the app already parses NWS alerts (`domains/weather/nws`), so the NWS national layer reuses those
types; USGS reuses the geo helpers.

**D5 — event → one representative location** (`domains/globalfeed/locate`): dedup by source `ID` first
(one entry per event), then resolve the location in order — (1) the highest watchlist location the event
applies to (point-in-alert-area / within a radius of the quake/storm), else (2) the nearest named city in
the geodata index to the event point, else (3) a fuzzy area phrase ("the San Diego area"). One entry, one
named place — never the same quake N times.

## 3. The stack + new-alert detection (P1 logic, P3 persistence)

- **Stack:** most-recent-first, deduped by `ID`, bounded (`maxTickerEvents`, e.g. 30). A refresh merges:
  existing ids keep their place; a genuinely new id **injects at the top** (breaking-news) and flags the
  marquee to interrupt and jump to it.
- **New-alert detection:** a persisted **seen-cache** — `id → first-seen` in the cache dir. An id absent
  from the cache is **new** ⇒ tone + narration + inject-at-top. Pruned to **[x] days** (proposed **7**, to
  cover the USGS week feed so a still-active week-old event never re-alerts). Bounded (P10).
- **Cold start (the alert-storm risk):** on the first fetch after launch with an empty cache, seed all
  current events as seen **quietly** (no tone/narration) — only events that appear *after* launch alert.

## 4. The marquee (P2) — ASCII mock for HUM LEAD to direct

A single full-width row. Proposed placement: **directly under the header stamp, above the radio panel**
(breaking-news reads at the top). One event scrolls at a time; its **background is its severity colour**;
the count and a position dot sit at the insets; `[M]` at the right.

```
 ⚠ 3  ◀ A Tornado Warning has been declared for the Oklahoma City area — play its broadcast for more   ● ○ ○  [M]
 └ count                     └ the scrolling event text (bg = severity: ▮Red ▮Orange ▮Yellow)          └pos  └mute
```

- **Rotation:** the marquee scrolls the top event; when it finishes it advances down the stack ("rotates
  per category" — consecutive events differ in class/colour). A **new** event injects at top and interrupts
  immediately.
- **Colours** are fixed R/O/Y `render` tokens, **theme-independent except monochrome** (which renders them
  greyscale, like every other mark — the token pattern the seismic mark established): a bare `:root` value
  the themes do **not** override, plus a monochrome override.
- **`[M]` mute:** toggles narration + tone (a persisted preference, like the voice); the marquee keeps
  scrolling when muted (visual only). New keybinding.
- `--ascii`: the ⚠ and the colour blocks fall back to ASCII markers.
- **Empty state (HUM LEAD 2026-08-27):** the row is **always present** and shows a **muted** "no active
  severe events" when nothing is active — never hidden, so the layout does not jitter as events come and go.

## 5. Audio (P3, R6) — tone + narration

- **The tone:** three attention tones then a 2-second pause (`<tone><tone><tone> <2 s>`), generated as PCM
  (the synth already renders PCM), played once per **new** alert through the engine. Proposed character: a
  three-pulse ~1 kHz attention tone (not the full EAS SAME burst — general, not an official EAS relay).
  HUM LEAD directs the exact tone.
- **The narration** (general, one template): *"A(n) `<Type>` has been `<verb>` for `<Location>`. Play the
  Watchpost Radio Broadcast of that location for more details."* — verb by class (weather **declared**,
  quake **recorded**, storm **reported**); **A/An** by the Type's initial sound. Reuses the synth voice.
- **Mute** gates both; **not-new** events never sound. Must not collide with a playing radio broadcast
  (duck/queue — a P3 design item).

## 6. Batches (TDD order, each a gate)

- **P1 — data layer (DATA FIRST).** `domains/globalfeed`: the Event model; the three fetchers against
  `httptest` fixtures captured from the live probes; the severity maps; the D5 location-tying (with the
  geodata resolver); the dedup + stack ordering + new-vs-seen split (in-memory). **Gate:** unit green,
  P10 snapshot, no UI/audio.
- **P2 — the marquee.** The single-row scroller, stack/interrupt, the R/O/Y bg tokens
  (theme-independent-except-monochrome), `[M]` visual state, empty state; colour-off + ASCII goldens.
  **Gate:** goldens, `make verify`, `make alloc-budget` **unchanged**, P10.
- **P3 — new-alert detection + audio (R6).** The persisted seen-cache (+ cold-start-quiet + prune + bound),
  the tone generation + play-on-new, the narration (tone→2 s→template, verb/article, D5 location), the
  `[M]` gate, radio ducking. **Gate:** Synth **and** Relay pty smokes + a 1-hour soak flat; P10; the
  byte/request check on the ticker fetch.
- **P4 — REVIEW + VALIDATE + release.** SEV-0 red-team (axes: dedup/location-tying correctness, the
  new-alert/cold-start honesty, the bounds, secret-free, R6, the fixed-colour theming); disposition all;
  `0.12.0` published; DEBRIEF.

## 7. Design decisions — RATIFIED by HUM LEAD (2026-08-27, "GO")

1. **Severity→colour map** (§2 table) — **accepted** (Quake ≥6.5 / HU / Tornado-Extreme Red · 5.5–6.5 /
   TS / Severe Orange · below / TD / watches Yellow · tsunami Red).
2. **Cache window `[x]` = 7 days** — accepted.
3. **Tone** — the three-pulse ~1 kHz attention tone — **accepted for now, to be tested at P3** (not an EAS burst).
4. **Placement — above the radio panel;** the empty state is a **persistent muted row** (no jitter).
5. **Narration verb/article** — weather=declared, quake=recorded, storm=reported; A/An by sound — accepted.

## 8. Risks / mitigations

| Risk | Mitigation |
|---|---|
| NHC misses W-Pacific typhoons | scope B accepts it; UI wording never claims full-global; JTWC a later add |
| NWS national query large in an outbreak | curated `event=` list + top-N by severity + the byte disciplines |
| Cold-start alert storm | seed current events as seen quietly on first fetch (§3) |
| Tone fires on every refresh | the seen-cache gates it to genuinely new ids only |
| Colours unreadable on some themes | fixed R/O/Y with a monochrome greyscale override; contrast-checked like the seismic mark |
| R6 | P1/P2 don't touch `domains/radio`; P3 is the only audio batch, full R6 gate |
| Ticker regresses the frame | one row, off the table memo path; `alloc-budget` gate |

## 9. Fit and non-regression

The ticker adds a **new pipeline** but reuses `httpx`, `geodata`/`geo`, the theme-token/render seam, the
synth voice/engine, and the same performance/P10/junior-doc gates. It touches the snapshot spine **not at
all** — the global data is isolated, so the per-location app cannot regress. Proven at each gate.
