# P4 REVIEW report — Global ticker 0.12.0 SEV-0 red-team

**Batch:** P4 of the global-ticker PLAN (REVIEW → VALIDATE → release → DEBRIEF). **SEV-0.**
**Scope reviewed:** the whole 0.12.0 surface — the ticker pipeline (`domains/globalfeed`, `app/ticker.go`),
the marquee (`modes/tty/ticker.go`), the breaking-news takeover + alert audio (`app/radio.go`,
`domains/radio/player`, `domains/radio/synth`), the Setup radius filter, and the two memory-perf changes
(HMS coalescing, the httpx large-entry tier), plus the 7 new themes.

## Method — 5 adversarial axes / personas (parallel)

1. **Correctness & equivalence** — a QA engineer trying to make the ticker show wrong/dup/stale/mis-located events.
2. **Concurrency, bounds & lifecycle** — an adversary firing bursts, radio toggles, theme switches, quits.
3. **Narration, secrets & TTS safety** — an attacker controlling feed text aiming at argv/exec, secret leakage.
4. **Hostile feeds & injection** — a MITM/compromised feed + a malicious theme aiming at crash / escape injection.
5. **Perf-change correctness** — an SRE worried the optimizations introduced staleness/coherence/leak bugs.

## Findings & disposition

| # | Sev | Finding | Disposition |
|---|-----|---------|-------------|
| F1 | **CRITICAL** | Terminal-escape injection — ticker rendered feed `Type`/`Location` without `render.Plain` (OSC-52 clipboard, title spoof, CSI corruption, DSR input-smuggle) | **FIXED** — `tapeText` → `render.Plain` |
| F2 | **HIGH** | `geoPoint` stack-overflow DoS — `json.Unmarshal` into `any` recurses per array level; nested GeoJSON crashes the process | **FIXED** — non-recursive `json.Decoder` token scan |
| F3 | **HIGH** | httpx cross-tier incoherence — `remember()` didn't evict the sibling tier on a size-boundary crossing → stale-serve + `largeBytes` leak | **FIXED** — single-residency (drop both tiers before insert) |
| F4 | **HIGH** | Overflow events past the 30-cap never seen-marked → stale alert resurfaces and fires a breaking takeover | **FIXED** — `cycle` seen-marks ALL active events |
| F5 | MEDIUM | Unbounded alert narration (feed fields) → say/piper render DoS + multi-minute duck lockout | **FIXED** — `clampField` bounds Type/Place at ingestion |
| F6 | MEDIUM | HMS served last-good indefinitely with `AsOf=now` on sustained outage | **FIXED** — `maxLastGood` (30 min) bound |
| F7 | MEDIUM | Radius filter silently fell back to global when no watchlist location | **FIXED** — shows nothing when filtered + no location |
| F8 | MEDIUM | `centerText`/`scrollWindow` clipped by rune not cell; `width<0` panic | **FIXED** — guard `width<=0`, clip by display width |
| F9 | HIGH (plaus.) | HMS held `p.mu` across the 30 s network fetch → stalls all fire fetches + shutdown | **FIXED** — release the lock during the fetch (httpx single-flights it; the large tier serves from memory) |
| — | MEDIUM | Piper voice install could run twice concurrently (new breaking caller) | **FIXED** — install mutex + re-check |
| F10 | LOW | HMS coalesce fast-path dropped `ErrTruncated` | **FIXED** — cached + returned |
| — | LOW-MED | Feed timestamp parse errors → zero time (mis-sort, bogus clock) | **FIXED** — fall back to `now` |
| — | LOW | `Locate` ran the tie at (0,0) for point-less alerts; ignored `geoPoint` `ok` | **FIXED** — skip tie when `!hasPoint`; use `ok` |
| — | LOW | `nearestMetro` no distance cap → foreign event → US metro | **FIXED** — 400 km cap |
| — | LOW | `cleanPlace` sliced original with a lower-cased index | **FIXED** — bound-guarded |
| — | LOW | 90 s rotation/scroll ran under a takeover → didn't resume "where it left off"; audible-but-no-voice → 0 hold | **FIXED** — freeze under takeover; fall back to `breakingHold` |

### Accepted (documented follow-ups, not blockers)

- **Dedup is by source ID only** — a superseded-then-reissued NWS alert (new ID, links prior via `references`)
  can briefly appear twice. Intermittent (the `active` feed usually returns only the latest). Follow-up: honor
  `references`/`replacedBy`.
- **A station tuned mid-takeover plays ducked** until the sequence ends (≤30 s). Transient, self-heals within
  50 ms of restore. Accepted.

### Verified correct (no finding)

TTS argv/shell safety (text via stdin / 0600 temp file only, no shell); no secret reaches the narration/logs;
theme value regex fully constrains user themes (no escape injection); `remember`/`forget`/`evictTier` byte
accounting within each tier; the `running` one-at-a-time guard; the duck/restore pairing (deferred, no leak);
`Merge` cap/dedup/order; `seenStore` round-trip; body-size cap; config parse (fails loud).

## Gate

full suite · `-race` (audio, ticker, cache, globalfeed) · `make verify` · `alloc-budget` unchanged ·
**`make p10` 0 live / 0 unmatched** · declsets regenerated · **live feeds re-proven** (USGS/NHC/NWS parse).
Fixes committed `c2de876`. 13 blocker/high/medium findings fixed + 2 accepted; **0 open blockers** → cleared
to VALIDATE + release 0.12.0.
