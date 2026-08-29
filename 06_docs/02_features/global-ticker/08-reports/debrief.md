# Global event ticker — DEBRIEF (After Action Report)

**Feature:** `global-ticker` → **shipped 0.12.0** (2026-08-27, `v0.12.0`, CI green). **SEV-0.**
**Flow:** DISCOVER → PLAN → BUILD (P1–P4) → REVIEW → VALIDATE → SHIP → REFLECT.

## What shipped

A global event ticker across the top of the dashboard — a separate pipeline from the per-location
snapshots. Three keyless feeds (USGS significant earthquakes, NHC tropical cyclones, NWS US
severe/tornado warnings) map to one Event model, deduped and stacked most-recent-first. The tape
rotates four category lanes (Severe Earthquakes → Tropical Cyclones → Warnings → Watches) every 90 s,
each lane `•`-separated and scrolling with issued/expires times, dropping an event the moment it
expires; a fixed per-category colour and a `[count][glyph]` left indicator. A breaking-news takeover
switches to the event's lane, shows it centred, holds through its narration (≥5 s), and reads it aloud
over the radio (the synth or the live relay ducks and returns; state codes expand, "ND" → "North
Dakota"); simultaneous events queue by severity. A redesigned Setup window groups settings by concern
with an Alert Notification Preference that scopes the whole ticker to N miles. Seven Omarchy Quattro
themes bring the palette to twelve. HMS coalescing + an httpx large-entry cache tier cut the
steady-state churn the ticker's feeds would otherwise add.

## What went well

- **DATA FIRST held under heavy UAT churn.** The data layer (Event model, Sources, Sort/Merge/Active/
  Within/Locate) was built and tested before the marquee, so the many UAT reshapes — marquee → category
  ticker-tape, the radius scope, the breaking takeover, the fixed colours — were **presentation swaps on
  a stable spine**, not rewrites. Each landed in a batch without disturbing the feed contract.
- **UAT caught what a spec couldn't.** Real behaviours only a human running the binary would notice:
  the "double narration" burst collision (>1 event, script >5 s overlapping the next), "VA" read
  letter-by-letter (the ExpandStates comment lied — the call was missing), the 90 s rotation looking
  like a bug when only two lanes were active. Each was a fast, well-scoped fix because the seam was
  already isolated.
- **The measurement, not the memory number, drove the perf work.** A soak showed 381 M vs a prior 111 M
  build. Profiling headless (`WATCHPOST_DEBUG_PPROF`) proved the ticker adds ~0 at rest (0.6 M) — the
  churn was HMS KMZ disk re-reads, unrelated to the feature. Fixing the *actual* cause (HMS
  coalesce/single-flight + httpx large tier) cut churn 528 → 270 M and disk reads 278 → 67 M. The
  headline number was a symptom; the profile named the disease.
- **The SEV-0 red-team earned its keep — again.** Five adversarial reviewers found a **CRITICAL escape
  injection** (feed text rendered raw to the terminal — a hostile feed could drive the display), a
  stack-overflow on deeply-nested GeoJSON `coordinates`, unbounded feed fields, and a superseded-alert
  resurfacing path. A scoped delta red-team on the follow-up fixes caught two more (a duck-reset
  over-reaching to automatic paths; superseded ids not seen-marked). All fixed and pinned.

## What to carry forward (lessons)

1. **A green suite on macOS is not a green suite.** The ship's own CI caught a `-race` failure on
   **ubuntu** that every local macOS run passed: a fire-and-forget goroutine (the ticker) writing an
   httpx cache entry into `t.TempDir()` as cleanup ran → "directory not empty". Linux's stricter
   `RemoveAll` and timing surfaced it; macOS masked it. **Run the Linux race gate before calling a
   concurrency change done** — or trust CI to, and don't ship on the local pass alone.
2. **Every goroutine needs an owner in the shutdown wait set.** `startTicker` did `go t.run(ctx)` and
   returned — bound to ctx cancellation but never *waited*. `stopAll` drained priority/recent but not
   the ticker, so cancel signalled it and returned before its last disk write settled. The fix (a `done`
   channel closed by `run`, waited in `stopAll` *outside* `lp.mu` because the cycle's watchlist tie takes
   that lock) is the pattern: if you `go` something that writes, something must join it.
3. **Keep the feed as data, never as a command.** The escape-injection CRITICAL is the reminder that
   *any* external text (a feed, not just user input) is untrusted on the way to the terminal and the TTS
   argv. `render.Plain` at the boundary, iterative parsing (no unbounded recursion), and length clamps
   are the standing defenses — applied here, worth a checklist for the next feed we add.
4. **Reshapes are cheap when the owner is single.** The colour pass, the mute chip move, the radius
   scope, and the settings-group Setup all landed as edits to *one* owner each (the theme token table,
   the header controls, the cycle filter, the Setup groups) because the codebase's single-owner rule was
   held. Divergence would have made each UAT turn a hunt.

## Follow-ups (open)

- **Linux relay half of R6** — HUM LEAD to run the live relay + audio pty smokes and a 1-hour soak on
  Arch (needs an audio device).
- **Stage-2 ticker audio (0.12.0 P3 deferred)** — duck the radio for the alert on *both* synth and
  relay paths, R6 gate. Stage-1 (pipeline + cache + mute) shipped.
- **Detail-modal per-tick rebuild memo** — carried from the seismic DEBRIEF; the modal still rebuilds
  every tick while open. Bounded and off the hot path; a memo removes the per-tick cost. Not
  ticker-specific.
- **Multi-alert circle viz** — deferred pending HUM LEAD design/mock.

## By the numbers

- 4 build batches (P1–P4) plus a P3 UAT arc that reshaped the marquee into the category ticker-tape,
  added the breaking takeover, the settings-group Setup, and seven themes.
- Red-team: 5 reviewers, 13 findings (fixed + regression tests) + 2 accepted-then-addressed; a delta
  red-team of 2 reviewers on the follow-ups fixed 2 more. 0 open.
- Perf: ticker ~0.6 M at rest; HMS + httpx large-tier work cut steady-state churn 528 → 270 M, disk
  reads 278 → 67 M.
- **SHIP:** CI red on the first tag (the Linux race flake above), fixed-forward — the tag never
  published an artifact, so the fixed commit re-took `v0.12.0`. Second run green on ubuntu + macOS;
  release published all five platform binaries + `checksums.txt`. Gates at ship: verify green · p10
  0 live/0 unmatched · alloc-budget within · race clean.
