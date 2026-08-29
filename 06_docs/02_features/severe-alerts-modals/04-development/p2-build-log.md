# P2 build log — app: publish path, `severeDeck`, narration, seen-store

**Batch:** P2 (Tasks 2.1–2.6) + P3 Tasks 3.1 (tokens/tone) and 3.2 (UI types) pulled forward · **Date:** 2026-08-28 ·
**Gate:** `make verify` ALL GATES GREEN · `make p10` 0 live, 0 unmatched · `go test -race ./app` green

## What landed

| Task | Files | Notes |
|---|---|---|
| 3.2 (pulled forward) | `modes/tty/severe.go` | `SevereTab`, `SevereMsg{Rows, Totals, Updated, Sources, Gen}`, `SevereSource`, `SevereRow`, `SevereRecord`, `severeTabs()` registry — `app/severe.go` imports them, so they precede P2 |
| 3.1 (pulled forward) | `platform/render/{theme,themes,sgr}.go` | `EventCat*BG` fixed tints + Monochrome greys; `CategoryTone(hue, dark)` mixing onto the active substrate (`categoryBlend` 0.6); `mixBG`/`bgRGB`; `UnregisterTheme` — the guard tests land with 3.1's test task in P3 |
| 2.1 | `app/ticker.go` `cycle` | per-source `SourceHealth`; `Locate` on the full set **before** the radius branch; `t.severe.SetFeed(events, health)`; `startTicker(…, severe)` |
| 2.2 | `app/severe.go` | `severeDeck`: `publishMu` end to end; change key = rows (key+sent) + source health + fetch minute; `Updated` = newest OK fetch; records composed only on change; `toSevereRow` zone rule via `platform/tz`; compile-time `Totals` length assert |
| 2.3 | `app/pipelines.go`, `app/dashboard.go` | `startRecent(…, onPublish)`; both publishers poke the deck; `lastSnapshots` and the pipeline assignments under `lp.mu` |
| 2.4 | `app/ticker.go` | tail "Press W in Watchpost for the full report on this event"; burst closing re-pointed; `tapeText` uses `Title()`; `eventNarration` through `render.Plain` |
| 2.5 | `app/severe_test.go` | NFR-1 by construction (the deck has no client — doc block); `BenchmarkSevereDeckTrigger` |
| 2.6 | `app/ticker.go` seen store | `0700`/`0600`; `maxSeenIDs` 20 000, oldest evicted on load |

## Deviations / findings while building

1. **Ordering:** the plan put P3 Task 3.2 after P2, but `app/severe.go` needs the UI types — 3.2 (and 3.1's
   tokens) were built first. The plan index's order note is amended by this log.
2. **`TestRecentPublishPokesTheDeck`** as planned compared pokes to the publisher's publish count; a
   never-run `tea.Program`'s `Send` blocks, so the count lags the poke forever (`publishes 0`). The test now
   asserts the poke arrives and releases the goroutine with `p.Kill()`. The hook counter is atomic (a real
   `-race` hit in the first version of the test).
3. **P10-06** flagged `severeBreakingWindow` (declared in 3.2, used in 3.3) — removed until 3.3 declares it.
4. **Declaration-set guards** in `app`, `modes/tty`, `platform/render` re-captured (intentional additions).
5. The 0.12.0 narration test's expected tail updated to the ratified script (N-1).

## Measurements

- `go test -race -count=6 ./app -run TestRecentPublishPokesTheDeck`: green after the fix.
- `BenchmarkSevereDeckTrigger` (400 feed + 180 location alerts, unchanged index): recorded at P3-9 with the
  frame budgets (one benchmark run, one log entry).
