---
title: "0.19.5 VALIDATE — the UAT agenda"
date: 2026-11-10
phase: VALIDATE
sev: SEV-0
status: "UAT on 77776225 done: every item passes, the screenshots kept as they were (D-47 to D-54)"
---

# 0.19.5 VALIDATE: what the UAT looks at

**The build.** `make build` on `77776225` (`watchpost version 0.19.3-64-g77776225`), at
`dist/watchpost`. It is the only live run of this release: the agent does not run the app, which
shares the HUM LEAD's Open-Meteo quota. One sitting with a coastal place in the watchlist covers
every item.

**How a verdict is kept.** Each item's verdict becomes a D-row in `02-analysis/rulings.md` (0.18.0's
L8). A failure is fixed before SHIP or ruled.

## The items

| # | What changed | Where to look | It passes when |
|---|---|---|---|
| 1 | **The console's card boxes** take the masthead's top rule (D-18, D-32) | the Broadcaster console (`B` or `ctrl+b`), at a wide and a narrow terminal | each card's top border has two rule marks before its name and after its grade badge; on a narrow card the name, then the badge, stays in the border as long as it fits |
| 2 | **A correspondent's first name** (D-7) | a synth hand-off on the radio, and the marquee | "This is Eddy", not "Eddy (English (US))"; the marquee shows "Eddy" |
| 3 | **Water temperature under metric** (D-6) | Settings → units metric, then a coastal place's marine report read aloud | "degrees Celsius", as the current conditions say; under imperial a bare "degrees" |
| 4 | **The console honours units and clock** (DEF-2 under D-5, D-21) | Settings → metric and 24-hour, then the console; then change either with the console up | the location pool in °C and km; the console masthead in 24-hour time; a change redraws at once |
| 5 | **Card manifest stamps follow the clock** (D-29) | console cards under 12-hour, 24-hour and military | DATA PULL and DATA PULLED stamps in the clock chosen. **A look the red team asked for (RG O-2):** on a narrow card the 12-hour stamp is three cells longer, so "(2 MIN AGO)" is cut sooner |
| 6 | **Settings chips and `p`** (D-36, A-22, D-33) | Settings → Watchpost UI → Reduce motion; Watchpost Radio → RELAY REPLAY's two rows; `p` on Theme and on a voice row | "←→ Motion"; "←→ Rotation" and "←→ Language"; `p` previews only on a voice row and does nothing on Theme, the HISTORY pickers or the relay rows |
| 7 | **The Observer fits the terminal** (D-28, D-30) | the Observer at 80x24 with more favourites than fit | nothing runs past the bottom; the priority table shows at least three rows with scroll marks and keeps the focused one in view; RECENT keeps a row |
| 8 | **Setup keeps a place by the app's one rule** (D-8) | re-run Setup with a place already in the watchlist | the place stays unless it is the same ZIP, or the same coordinates to four decimals |
| 9 | **Backspace in the Line-Up Request window** (DEF-1 under D-5) | the console's Line-Up Request (`r`): type "Peñ" into the place, then backspace | "Pe", not a broken character |
| 10 | **The screenshots** (A-9) | `docs/img/setup.png`, `broadcaster-settings.png`, `help.png` | retaken on this build: Settings in its tabs. README's alt text already describes them |

**Not a UAT item: M2.** The beats are measured by the allocation pins, threaded since batch 50
(A-25): idle 76.0 KB against 80, radio 68.5 against 90, console 122.4 against 140, all met.

**Optional, and costs nothing live:** an unknown `[radio] mode` in `config.toml` now stops Watchpost
at start, naming the key and its two words (D-9).

## For veto at the same sitting

A-18 (its 0.00 floor, GT M-6), A-24 to A-30. The minor rows are listed in `02-analysis/rulings.md`.

None was vetoed (D-46).

## Verdicts

| # | Verdict | Row |
|---|---|---|
| 1 | pass | D-47 |
| 2 | pass | D-54 |
| 3 | pass | D-48 |
| 4 | pass | D-49 |
| 5 | pass; the narrow 12-hour cut stands | D-50 |
| 6 | pass | D-54 |
| 7 | pass | D-51 |
| 8 | pass | D-52 |
| 9 | pass | D-54 |
| 10 | the 0.16.0 screenshots kept; the retake carried as F-227 | D-53, A-31 |
