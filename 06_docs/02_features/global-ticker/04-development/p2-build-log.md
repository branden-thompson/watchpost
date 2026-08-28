# P2 build log — Global ticker marquee (the row)

**Batch:** P2 of the global-ticker PLAN (`03-architecture-design/plan.md` §6). **SEV-0** · FULL TDD.
**Branch:** `feature/global-ticker`. **Status:** at gate — unit + goldens green; the live pipeline + audio
are P3, so the running app shows the persistent muted row until then.

## 1. What landed (junior-first)

The single-row marquee, **above the radio panel**, rendered from a UI-level type so `modes/tty` stays
decoupled from the domain (the snapshot-only rule).

- **`modes/tty/ticker.go`** — `TickerItem { ID, Text, Severity }` (the app maps a `globalfeed.Event` onto
  it) and the render: `⚠ N  <the top event scrolling>  [M] Mute`, the **whole row painted in the current
  event's severity background**. A line that fits is static; a longer one scrolls (text + gap, wrapping) —
  the marquee mechanic. Empty ⇒ a **persistent muted** "no active severe events" row (never hidden, so the
  layout never jitters — HUM LEAD).
- **Severity backgrounds** (`render` tokens, 0.12.0): `TickerRedBG` / `TickerOrangeBG` / `TickerYellowBG`
  + `TickerFG` (bold white). **Theme-independent** — set on the base palette, inherited by every theme —
  **except Monochrome**, which overrides them to three greyscale shades (severity by shade, not hue).
- **The stack + interrupt** — `setTicker` replaces the stack; a **new top id** (a different breaking event
  at index 0) **jumps the marquee to it** (breaking-news interrupt); an unchanged top keeps the scroll.
  `advanceTicker` steps the scroll on the wall clock and **rotates to the next event** when the current one
  has fully scrolled.
- **`[M]` mute** — a new binding (`M`, i.e. shift+m, distinct from `m` = radio mode), a TICKER Help group,
  toggling `tickerMuted` (the visual `[M] Mute`/`[M] Muted` label; P3 wires the tone/narration gate).
- **Wiring** — `tickerMarquee` in `writeBody` (above the radio module); `tickNeeded` scrolls while events
  are active; `TickerMsg{Items}` updates the stack; the layout reserves the row (`chromeLines 9 → 11`).

## 2. Tests first (the pins)

| Pin | Guards |
|---|---|
| `TestTickerEmptyIsAPersistentMutedRow` | empty ⇒ a full-width muted row with `[M]`, never hidden |
| `TestTickerShowsCurrentEventOnItsSeverityBackground` | count `⚠ N`, the top event, the severity bg token |
| `TestTickerMuteTogglesTheControlLabel` | `[M] Mute` ↔ `[M] Muted` |
| `TestTickerScrollAdvancesAndRotates` | a full scroll rotates to the next event |
| `TestTickerNewTopEventInterrupts` | a new top id jumps to it; an unchanged top keeps the scroll |

Rendered (colour-off):

```
  ⚠ 1  A Tornado Warning has been declared for the Oklahoma City area                     [M] Mute
  no active severe events                                                                 [M] Muted
```

## 3. Performance (the mandate)

- `make alloc-budget` **unchanged** — the marquee is one row rendered outside the table memo (like the
  radio panel), not on the memoised hot path; it re-renders per frame to scroll, which is one row.
- The row is bounded (one event shown at a time; the stack caps at 30 in the data layer).

## 4. Gate

| Check | Result |
|---|---|
| `go test ./modes/tty/...` | green (unit) |
| `make verify` | ALL GATES GREEN |
| full suite | green |
| `make alloc-budget` | unchanged |
| `make p10` | **0 live · 0 unmatched** · no new exemptions (the `tickerTones` unparam finding was fixed to `tickerBG`) |
| goldens | render + tty declsets regenerated (new tokens/decls); frame goldens regenerated (the marquee row now in the body; the RECENT window gives up its 2 rows) |

## 5. Docs touched

- `modes/tty/ticker.go` (new, + header); `platform/render/theme.go`/`themes.go` (the ticker tokens);
  `modes/tty/dashboard.go`/`body.go`/`layout.go`/`help_about.go` (wiring, binding, chrome).
- Window-size tests updated to the marquee's −2 RECENT rows; frame + declset goldens regenerated.

## 5b. HUM LEAD UAT (2026-08-27)

- **The [M] control is a real key-cap chip** (`o.KeyCap("M")`), following the same pattern as every
  other key-bound control, embedded in the band (its own style, the band bg resuming after it).
- **The marquee is a 3-row band** (a coloured blank row above and below the content) so it breathes; it
  **absorbs the header→ticker and ticker→radio spacer rows**, so the overall frame height is unchanged
  (`chromeLines` stays 11).
- **The muted empty-state band matches the RECENT/SEARCHED group-header background** (`GroupSectionBG`,
  theme-appropriate), not a bespoke muted tile.

## 5c. HUM LEAD UAT round 2 (2026-08-27)

- **[M] moved to the header controls**, after [t] Theme: `[s] Setup  [a] About  [t] Theme  [M] Mute
  Severe Alerts  [?] Help  [q] Quit`. Its label says the action (Mute / Unmute) and **shortens on narrow
  terminals** ("Mute Severe Alerts" → "Mute" → dropped, binding still live) so the row never overflows,
  like line 1s stamp.
- **The marquees right side is cleared** (`tickerRightReserve`) to make room for the coming multi-alert
  **circle viz** (a follow-on).

## 6. Carried forward

- **P3** — the live ticker **pipeline** (fetch the three feeds on a cadence → map to `TickerItem` → send
  `TickerMsg`), the persisted **seen-cache** (cold-start-quiet + prune + bound), the **tone** on a new
  alert, and the **narration** (tone → 2 s → the template), the `[M]` gate. **R6 gate.**
- **P4** — REVIEW + VALIDATE + release `0.12.0` + DEBRIEF.
