# P3 build log — Global ticker live pipeline + seen-cache + [M] + alert audio

**Batch:** P3 of the global-ticker PLAN (`03-architecture-design/plan.md` §6). **SEV-0** · FULL TDD.
**Branch:** `feature/global-ticker`. **Status:** code complete — stage 1 (pipeline) + stage 2 (ducking
audio) both landed; the **R6 runtime gate** (pty smokes + a 1-hour soak, which need a real audio device)
is HUM LEAD's UAT/runtime pass. **Stage 1 was gated separately** (§1–5); stage 2 is §7 below.

## 1. What landed (junior-first)

The ticker's **own** pipeline — the global feeds don't fit the per-location snapshot assembler, so they run
on a separate loop that ties each event to a place, stacks them, and streams the marquee into the TTY.

- **`app/ticker.go`** (new) — `tickerDeck` fetches the three `globalfeed.Source`s (USGS / NHC / NWS) every
  `tickerEvery` (2 min; the feeds' own TTLs ride the httpx cache, so a fast tick only hits the network for
  the sources that are due), ties each event to a place (`globalfeed.Locate` → highest watchlist within
  150 km → nearest named city → the fuzzy "the <metro> area"), stacks + dedups (`globalfeed.Merge`), and
  sends `tty.TickerMsg{Items}`. `itemsOf` composes the marquee line — *"A(n) `<Type>` has been `<verb>` for
  `<Location>`"* — the first sentence of the coming narration, no tail.
- **The seen-cache** — `seenStore` persists announced event ids (`id → first-seen`) to
  `userCacheSubdir("ticker")/seen.json`, **pruned to a 7-day window** (the USGS window; HUM LEAD). It drops
  stale ids on load and on every mark, so it stays bounded by the window and the feeds' own sizes.
- **Cold-start-quiet** — the first cycle after launch **seeds the current events silently** (`warm.Swap`)
  and marks them seen, so a fresh start (or one after days away) never alert-storms. Only events that are
  genuinely new *after* warm-up count as fresh.
- **The `[M]` gate plumbing** — `tickerMuteState` is the shared `*atomic.Bool` the pipeline reads (seeded
  from `config.TickerMuted`) plus the hook the dashboard calls on a toggle: it flips the flag and persists
  the preference. `cycle` already checks `!muted.Load()` before it would sound `onNew` — the audio hook is
  wired but **nil** until stage 2, so today the flag is honoured with nothing to sound.
- **Wiring** — `RunDashboard` seeds the mute state, passes `MuteTicker` into `tty.Config`, and launches
  `startTicker` alongside the priority/recent pipelines. The launch watchlist ties events (a live-watchlist
  re-tie is a noted follow-up).

## 2. Tests first (the pins)

| Pin | Guards |
|---|---|
| `TestItemsOfComposesTheMarqueeLine` | the line: `An Earthquake has been recorded for Nepal`; `A Tornado Warning has been declared for the Oklahoma City area`; severity carried through |
| `TestSeenStoreColdStartPruneAndPersist` | a fresh cache is empty; a mark records + prunes an 8-day-old id; ids persist across a reload |
| `TestSeenStoreLoadDropsStale` | a 30-day-old id is dropped on load |

(The feed fetchers + `Locate`/`Merge`/`Sort` carry their own unit + live tests from P1 — live-proven USGS 3,
NHC 2, NWS 40 events.)

## 3. Performance (the mandate)

- `make alloc-budget` **unchanged** — the ticker loop is off the memoised table hot path; a cycle is three
  cached fetches + a bounded map/sort, at 2-min cadence.
- Everything is bounded: the stack caps at 30 (data layer), the seen-cache prunes to the 7-day window, the
  `nearestMetro` scan is capped at the top-300 US cities.

## 4. Gate

| Check | Result |
|---|---|
| full suite (`go test ./...`) | green |
| `make verify` | ALL GATES GREEN |
| `make alloc-budget` | unchanged |
| `make p10` | **0 live · 0 unmatched** |
| declset golden | regenerated (four new `app` decls: `startPipelines`, `ttyConfig`, `tickerDeck.run`, `tickerMuteState`) |

**P10-04 (small functions).** The ticker wiring pushed `RunDashboard` to 42 statements (limit 40). Split
into two single-owner helpers — `ttyConfig` (the TTY hook set) and `startPipelines` (priority + recent +
FIRMS + ticker launch, returning the first-full-snapshot timer). `RunDashboard` is back under budget; no
exemption taken.

## 4b. Exemption RATIFIED

- **`app/ticker.go` · `run` · P10-02-BOUNDED-LOOPS** — the ticker's background event loop is a
  `select { ctx.Done() ; ticker.C }`, an unbounded event loop ended by context cancellation, the same shape
  the scheduler tier loops (`platform/sched/sched.go`) already hold a P10-02 row for. There is no meaningful
  iteration count to bound it. **Ratified by HUM LEAD at this P3 gate, 2026-08-27.** (The
  `domains/globalfeed` package P10-05 row was ratified at the P1 gate.)

## 5. Files touched

- `app/ticker.go`, `app/ticker_test.go` (new).
- `app/dashboard.go` (ticker wiring; `ttyConfig` + `startPipelines` extracted).
- `platform/config/config.go` (`TickerMuted` field — already carried from P2 plumbing).
- `modes/tty/dashboard.go` (`TickerMuted` / `MuteTicker` config surface — from P2).
- `app/testdata/declset.txt` (regenerated).

## 6. Design call SETTLED (HUM LEAD, 2026-08-27): DUCK THE RADIO

When a fresh alert fires while the radio is already playing, lower the broadcast's volume, play the alert
tone + narration over it, then restore — **not** interrupt, **not** visual-only. Implemented in §7.

---

## 7. Stage 2 — the alert audio (tone → 2 s → narration, ducked)

### What landed (junior-first)

The audio path reuses the existing radio engine's speaker (`oto/v3`, one shared context that **mixes**
players) rather than opening a second device — the alert is an overlay player on the same context, and the
main broadcast is ducked under it.

- **The tone (new — it did not exist).** `synth.AlertTone(rate)` (`domains/radio/synth/tone.go`) renders the
  attention signal HUM LEAD specified: **three enveloped ~1 kHz pulses** (200 ms, 100 ms gaps) then a **2 s
  pause**, as 16-bit LE **stereo** PCM. (The seismic "tone" was actually a *spoken* notice + silence — there
  was no PCM generator anywhere; this is the first.) The attack/release envelope keeps the pulses click-free.
- **The narration.** `synth.AlertNarration(ctx, v, text)` renders text in the radio voice (mono→stereo),
  reusing the `Voice` adapter (`say`/Piper) — product text stays out of argv (§10.5). The app composes the
  line via `globalfeed.Event.Sentence()` (now the **single owner** of the sentence, shared with the marquee)
  + the radio tail: *"…. Play the Watchpost Radio Broadcast of that location for more details."*
- **Ducking.** `player.Engine` gained a `duck` multiplier (1.0 normally). `Engine.Alert(rate, pcm)` plays the
  clip at the knob volume (itself un-ducked) and sets `duck = alertDuck` (**0.25**); the main stream's watch
  loop re-asserts `volume × duck` every 50 ms, so the broadcast dips within 50 ms and restores within 50 ms
  when the clip drains — the restore runs on the engine's own goroutine, so a caller can never leak a duck. A
  duck with nothing playing is inert (no main player to re-scale).
- **Wiring.** `radioDeck.alertAudio(events)` → `playAlert(text)` renders `AlertTone ++ AlertNarration` and
  calls `Engine.Alert`; the ticker's `onNew` hook (`livePipelines.tickerAlert()`, nil when there's no deck)
  fires it for the **lead** fresh event, already gated by `[M]` and cold-start-quiet in `cycle`. A missing
  voice engine is not fatal — the marquee still shows the alert; only the sound is skipped.

### Tests first (the pins)

| Pin | Guards |
|---|---|
| `TestAlertToneIsThreePulsesThenAPause` | ~2800 ms, stereo (L==R), audible pulses in the first 800 ms, a silent 2 s tail |
| `TestAlertToneRejectsNonPositiveRate` | a non-positive rate → no tone |
| `TestAlertNarrationDoublesMonoToStereo` | Say's mono is doubled to stereo |
| `TestAlertDucksTheBroadcastAndRestores` | the main player dips to `volume×0.25` while the clip plays, restores after (recording-output engine test) |
| `TestAlertNarrationAppendsTheRadioTail` | the exact template; its lead sentence == the marquee line (single owner) |

All race-clean (`go test -race ./domains/radio/... ./app/`).

### Gate

| Check | Result |
|---|---|
| full suite (`go test ./...`) | green |
| `go test -race` (audio + ticker) | green |
| `make verify` | ALL GATES GREEN |
| `make alloc-budget` | unchanged |
| `make p10` | **0 live · 0 unmatched** |
| declset golden | regenerated (`alertNarration`, `radioDeck.alertAudio`/`playAlert`, `livePipelines.tickerAlert`) |
| ticker fetch byte/request floor | analytically bounded — per hour at the 2-min tick: NWS ~30, USGS ~12, NHC ~2 network requests; the httpx per-feed TTLs (2/5/30 min) serve the rest. The **measured** floor comes from the soak. |

### Files (stage 2)

- `domains/radio/synth/tone.go` + `tone_test.go` (new — the tone + narration composition).
- `domains/radio/player/engine.go` (the `duck` field + `Alert`) + `alert_test.go` (new — the ducking test).
- `domains/globalfeed/event.go` (`Event.Sentence()`).
- `app/ticker.go` (`alertNarration`, `onNew` param), `app/radio.go` (`alertAudio`/`playAlert`),
  `app/dashboard.go` (`tickerAlert()` wiring), `app/ticker_test.go` (narration pin).

### R6 — HUM LEAD's runtime/UAT pass (needs a real audio device)

The code gate is green; **R6's runtime half is yours** because it needs sound and wall-clock:
- **Synth pty smoke** — tune the synth broadcast, trigger an alert, hear tone → 2 s → narration; confirm the
  synth ducks under it and returns.
- **Relay pty smoke** — same over a live NWR relay stream (the duck path is identical — the same engine
  watch loop — but proves it against real decoded audio).
- **1-hour soak flat** — `scripts/quality/soak.sh` on the running binary: RSS/heap flat, no leak from the
  overlay players (each `Alert` closes its player on drain).
- Watch for: no deadlock/clipping on an alert that overlaps another, and a **clean restore** every time.

Binary built for UAT: `dist/watchpost` (synth is the default source; `[M]` toggles mute).

---

## 8. Stage 3 — marquee redesign from HUM LEAD UAT (2026-08-27)

The first live UAT reshaped the marquee from a one-event-at-a-time rotator into a **category ticker-tape**.

### What changed (junior-first)

- **Ticker-tape (#5).** A lane's active alerts now render as one continuous, `•`-separated tape that scrolls
  right-to-left, instead of showing one event and rotating. `scrollWindow` gained a wrap-gap arg (the bullet)
  so the tape reads continuously across the loop seam.
- **Category lanes + 90 s rotation (#6).** Alerts are grouped into four lanes — **Severe Earthquakes →
  Tropical Cyclones → Warnings → Watches** — and the band rotates to the next **non-empty** lane every 90 s.
  The rotation is driven by the pipeline (`TickerAdvanceMsg` every `tickerRotate`), so it keeps a steady
  wall-clock rhythm independent of the frame tick and the tty stays deterministic (goldens stable). One lane
  present ⇒ no rotation. The lane label + active count sits at the band's left (`WARNINGS 12`).
- **Fixed per-category colour.** Warnings = **Red**, Watches = **Yellow** (HUM LEAD); Earthquakes = Orange and
  Tropical = a new **Blue** token are **PLACEHOLDERS** for the colour pass. New `render.TickerBlueBG` +
  its Monochrome greyscale override.
- **When + how long (#1).** Each tape item reads `<Type> · <Location>  <verb> <t> · expires <t>` (the class
  verb — declared/recorded/reported; `expires` only when the alert has an active window). The **narration**
  now carries the same: *"… declared for … at 3:42 PM, active until 4:15 PM. Play the Watchpost …"*. Times
  render in the local zone (say the word for the affected-area's local time instead).
- **Drop-when-expired (#2).** New `globalfeed.Active(events, now)` prunes any alert past its `Until` before the
  stack is built; NWS now captures `ends`/`expires` into `Until`. An expired lane simply falls out of the
  rotation on the next publish.
- **State pronunciation (#3).** `AlertNarration` now runs `ExpandStates` before `Pronounce`, so the **voice**
  reads "ND" as "North Dakota" (the marquee text keeps the abbreviation).
- **#4** — no circle viz on the right: confirmed intentional; the right reserve stays.

### Tests

New/updated pins: tape+lane render with the category colour, the 90 s lane rotation (Red→Yellow), single-lane
no-rotation, continuous tape scroll+wrap, expired-lane drop; app lane-mapping + tape-line + narration-with-times
(quake has no `expires`); `globalfeed.Active` (drop expired, keep windowless, no mutation). Race-clean.

### Gate

full suite · `-race` · `make verify` · `alloc-budget` unchanged · **p10 0 live / 0 unmatched** · declsets
regenerated (app / tty / render). Binary rebuilt: `dist/watchpost`.

### Open for HUM LEAD

- **Colour pass** — set the final Earthquakes and Tropical band colours (currently Orange / Blue placeholders).
- **Audio scope** — in this UAT only the Tornado Warning sounded, not the earlier Thunderstorm Warnings. That
  is cold-start-quiet doing its job: alerts already present at launch are seeded silently; only alerts that
  arrive *after* warm-up sound. If you'd rather each newly-arriving alert sound (queued so they don't stack),
  say so — today one alert (the lead fresh event) sounds per cycle.
- **Time label wording** — the tape uses the class verb ("declared 3:42 PM · expires 4:15 PM"); your mock
  wrote "issued:/expires:". Tell me if you'd prefer the literal "issued:" label.

---

## 9. Stage 4 — breaking-news takeover + Setup radius filter (UAT)

Two UAT enhancements after the marquee redesign.

### Breaking-news takeover (HUM LEAD 2026-08-27)

A genuinely NEW alert now **takes the marquee over**: the band shows only that event, **centred**, in its
lane colour, holding `breakingHold` (5 s) while its narration plays; then normal rotation resumes where it
left off. Multiple simultaneous fresh events **queue by severity** (`byBreakingPriority`, most-severe then
most-recent) and are read through one at a time. **Audio: narrate each** (HUM LEAD) — the tone sounds once
up front, then a narration per event as the visual steps; muted ⇒ silent, the visual still steps.

- **Pipeline** (`app/ticker.go`): on fresh events `cycle` spawns `breaking(ctx, fresh)` under a `running`
  guard (one takeover at a time — a second burst mid-sequence is seen-marked and shows in normal rotation,
  never a doubled takeover), capped at `maxBreaking` (8; the overflow rotates normally). It sends a
  `TickerBreakingMsg` per event, holds 5 s, then `TickerBreakingDoneMsg`.
- **Audio** (`domains/radio/player`): `Engine.Alert` split into `Duck()` / `Restore()` primitives; the
  sequence ducks once, overlays the tone then a narration per event (reusing `Preview`, which rides the
  un-ducked knob), and restores at the end (deferred, so a duck can never leak). The deck implements
  `tickerAudio` (`startBreaking` / `narrate` / `endBreaking`); a nil voice ⇒ the visual still runs.
- **tty**: a `breaking *TickerItem` state; `tickerMarquee` renders it centred (`centerText`) in the lane
  colour, overriding the tape until Done. The four ticker messages route through one `handleTicker` (kept
  `dispatch` under the P10-04 complexity limit).

### Setup — Alert Notification Preference (radius filter) + fixes

- Setup/Configs modal grouped into **Data Access** and **Severe Weather / Disaster Events**; new Q3, the
  **Alert Notification Preference** (`● All` / `○ Filtered to [N] Mi of my location`) — the radius scopes
  the **whole ticker** (HUM LEAD). `globalfeed.Within` filters; NWS alerts now carry a point parsed from
  their GeoJSON geometry (`geoPoint` + `HasPoint`), zone-only alerts excluded when a radius is set.
  `config.TickerRadiusMi` persists; `tickerRadiusState` feeds the live pipeline.
- Left indicator returned to `[count] [glyph]`; final lane colours (Earthquake Red, Warnings Orange,
  Watches Yellow, Tropical Blue). Narration bug fixed: `AlertNarration` runs `ExpandStates` ("VA" →
  "Virginia").

### Gate

full suite · `-race` (audio + ticker) · `make verify` · `alloc-budget` unchanged · **p10 0 live / 0
unmatched** · declsets regenerated. New pins: breaking render (centred, lane colour, resume),
`byBreakingPriority`, `breakingItem`, `Duck`/`Restore` (main-only, overlay un-ducked), the radius/geoPoint
tests, Q3 toggle/persist.

### Open for HUM LEAD

- The 5 s hold and the duck depth (25 %) are one-line constants — tune after hearing/seeing it.
- `maxBreaking` = 8 per burst (overflow rotates normally) — adjustable.
