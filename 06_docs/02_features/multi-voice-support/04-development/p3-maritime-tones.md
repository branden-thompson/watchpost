# P3 — the maritime report and the tones (multi-voice-support, 0.14.0)

```
Goal:         Coastal listeners get a spoken maritime report — the coastal forecast, the buoy, the tides and the
              currents — read by the Maritime correspondent between the location forecast and the fire report;
              and the five ratified alert tones replace the single classic one.
Architecture: plan.md §2.5–2.6; 02-analysis/maritime-report.md; 02-analysis/tones.md
Branch:       feature/multi-voice-support
Gate:         07-readiness/gates.md §1
```

Task shape only; code at BUILD (`AP-PLANCODE-01`). The PLAN sketches are in `prior-art/p3-maritime-tones-code.md`
— the round-4 lenses ran the marine word set against the scripts (every expected sentence matched) and the five
presets' envelopes, so those parts are proven; the zone resolver and the assembler hook were checked by symbol
only.

## File map

```
CREATE: platform/render/marine.go (+ test)        — SeaState, TideTrend, NextTide, CurrentPhase, FirstOf (lifted)
MODIFY: modes/tty/detail_marine.go                — reads the lifted helpers (one owner for screen and voice)
CREATE: domains/radio/synth/marine.go (+ test)    — MarineReport, MarineSegments, the word helpers
CREATE: domains/radio/script/scripts/maritime-report/*.txt (11)
MODIFY: domains/radio/synth/{products,ugc}.go     — Products.Marine; UGCCodes
MODIFY: domains/radio/synth/tone.go (+ test)      — the other four presets, Presets, PresetByName
MODIFY: platform/snapshot/{assembler,harmonize}.go — MarineFor; mergeMarine shared with harmonizeMarine
CREATE: domains/weather/nws/marinezone.go (+ test) — MarineZoneFor, the centroid memo
CREATE: app/marine.go (+ test)                     — marineFor, coastalForecast, the screen/voice parity test
MODIFY: app/radio.go                               — Reports.Maritime and the report order
```

---

### Task 3.1 — lift the marine words to `platform/render` (RS-11)

**Contract:** the sea state, tide trend, next tide and current phase are computed **once** and shared by the
screen and the voice, so the two can never describe the same sea differently. The lift is a move, not a rewrite:
`modes/tty/detail_marine.go` keeps its output byte-for-byte.

**Test intent:** the Details view's marine rows are unchanged; a parity test (in `app`, which may import both —
`modes/tty` may not import a domain) asserts the spoken sea state equals the screen's for one `Marine` record.

**Verify:** `go test ./platform/render ./modes/tty -run Marine -count=1`

---

### Task 3.2 — `MarineReport` and `MarineSegments` (FR-6; MVS-D-21 wording)

**Files:** `domains/radio/synth/marine.go`.

**Contract:** `MarineReport{Known, State, TZ, Lat, Lon, Forecast}` and `MarineSegments(...)` producing, in order:
head (2 s pause, reusing the existing report-head pause — no new constant) · forecast · buoy provenance · sea and
swell in one sentence · water · wind · tide · next tide · current · the absence line when there are no tides or
currents · the link line. Every segment carries `cast.Maritime`. The **forecast is prose from the network**, so
it goes through the same sentence splitter every other product uses (the listener's `[r]` repeats a sentence,
not a minute of text, and a hand-over can land between sentences), and the whole section is **bounded** — a
product may not turn into an unbounded broadcast. Wording per MVS-D-21: "above the low-water mark", a 12-hour
clock in the location's zone, wind in the listener's unit, currents in knots, "minus" for negative heights. The
station name is `PlainLine`-and-capped at its seam (NFR-6). Complexity: keep the observation sentences in their
own helper (P10-04).

**Test intent:** each sentence against its script with a fixture record, imperial and metric; the absence line
when tides and currents are empty; the section's bound with an oversized forecast; the parity assertion from 3.1.

**Verify:** `go test ./domains/radio/synth -run Marine -count=1`

---

### Task 3.3 — the maritime scripts (FR-6; the HUM LEAD's words)

**Files:** `domains/radio/script/scripts/maritime-report/*.txt` (head, forecast, observed, sea, water, wind,
tide, tide-next, current, absence, link).

**Contract:** each part is a template over named data, defaults as recorded in `maritime-report.md` §7 as amended
by MVS-D-21. **RAT-5 is ratified as written (MVS-D-36)**: the wording ships as recorded; the HUM LEAD may still
reword any part at will, but BUILD does not wait on it. The
script convention test renders every part with one data map, so the map gains this report's keys.

**Verify:** `go test ./domains/radio/script -count=1`

---

### Task 3.4 — the hook: `MarineFor`, `marineFor`, `Reports.Maritime`, the order (MVS-D-18)

**Files:** `platform/snapshot/assembler.go`, `harmonize.go`; `app/marine.go`, `app/radio.go`.

**Contract:** the deck reads one location's merged coastal block without cloning the snapshot per cycle. The
field-wise merge across providers already exists in the publisher; **extract it once** and let both callers share
that body (the second-caller rule) rather than re-implementing the loop. The broadcast order is products →
**maritime** → fire → seismic → tail, with the existing air-gap rule between reports.

**Test intent:** the merge picks the same fields the publisher does; an inland or untracked location reports "not
ok"; the three reports are separated by air in the ruled order.

**Verify:** `go test ./platform/snapshot ./app -run 'Marine|Reports' -count=1`

---

### Task 3.5 — the coastal-waters forecast (MVS-D-14; AX-7 — **ruled (B), MVS-D-38**)

**Files:** `domains/radio/synth/products.go`, `ugc.go`; `domains/weather/nws/marinezone.go`; `app/marine.go`.

**Contract:** the office's Coastal Waters Forecast is fetched through the existing products endpoint and cut to
the zone this location belongs to plus the office's synopsis block, then to the first three periods
(`SpokenPeriodsCap = 3` — RAT-6, ratified as MVS-D-37). The period cut happens on the **raw** product text, before
normalisation rewrites the period tags. `UGCCodes` reuses the existing UGC splitter and range expander — no
second parser — and returns the codes in document order (sorted within a block for determinism).
**Zone choice (AX-7) — ruled: (B) only for 0.14.0 (MVS-D-38).** The zone is **the first nearshore block after
the synopsis, with no geometry**. Option (A) — the nearest zone by polygon centroid — is **not built**: MVS-D-14
ruled the CWF came "for free" and (A) is a new NWS endpoint, so it returns in a later release only if UAT reads
the wrong stretch of water. Consequences for this task: `MarineZoneFor` needs no centroid memo, no fan-out cap
and no separate call budget, and `marinezone.go` shrinks to the block selection. Every zone id that reaches a
URL is still regex-validated at the I/O edge, not only at the call site.

**Test intent:** UGC ranges and lists expand correctly; the first nearshore block after the synopsis is the one
chosen, and a product with no nearshore block yields "not ok" rather than a wrong zone; a zone id that fails the
regex never reaches a URL; the forecast is three periods (`SpokenPeriodsCap = 3`, RAT-6 ratified as MVS-D-37); a
coastal tune reads the section.

**Verify:** `go test ./domains/weather/nws ./app -run 'Marine|Zone|Coastal' -count=1`

---

### Task 3.6 — the other four presets (FR-11; tones.md §1)

**Files:** `domains/radio/synth/tone.go` (+ test). The storm class keeps the mock's label **"Maritime"**
(MVS-D-33 — the mock-fidelity default; revisable before the P4 gate).

**Contract:** dual-tone, 1050 Hz, soft chime and low sweep join `Classic()` as **functions**, not package
variables; `Presets()` and `PresetByName` cover all five. The parameters are the contract in `tones.md` §1 —
reproduced from parameters, never from samples. The deck's `tone(class)` renders one per takeover from constants
(~1 ms); there is no memo.

**Test intent:** each preset's length, rate and envelope within tolerance; every ratified name resolves;
`PresetByName` of an unknown name yields the loud default.

**Verify:** `go test ./domains/radio/synth -run 'Tone|Preset' -count=1`

---

### Task 3.7 — P3 gate

Run `07-readiness/gates.md` §1 (declsets before the test line — `platform/render`, `platform/snapshot`,
`domains/weather/nws` and `modes/tty` all move here; stage before `make p10`), plus the R6 soak on a **coastal**
cycle so the hour-long run carries the maritime section.

- **Build log** `p3-build-log.md`: the soak's numbers, ledger decisions, deviations. (E-8 is ruled — (B) — so
  there is no decision left to take here; record only if BUILD finds the block selection insufficient.)

**UAT-able alone:** tune a coastal location — the broadcast reads the coastal forecast, the buoy, the tides and
the currents between the location forecast and the fire report; each alert class opens with its own tone.
