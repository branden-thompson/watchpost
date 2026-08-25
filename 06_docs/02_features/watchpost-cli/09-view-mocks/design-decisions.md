# View-Mock Design Decisions (running log)

Source mocks: `watchpost-cli-view-mocks-with-notes.txt` (HUM LEAD; NOT set in stone — iterate freely).

| # | Decision (D-19, 2026-08-23) | Consequence |
|---|---|---|
| 1 | `a` About · `A` Alert Details · `ctrl+a` Add Location | Default KeyMaps; resolves the Shift+A collision; all remain config-swappable (D-15), only `?` locked |
| 2 | Location services → v0.2 | v0.1 "default location" = first priority location; setup question ships greyed "coming soon" |
| 3 | Units: `f`/`c` global live-swap of DISPLAY units | SI internal unchanged; `--json` stays SI; render layer carries a Units state consulted by every formatter; live-swap = re-render, no refetch |
| 4 | Recent/searched history cap ~50, evicted oldest | config persistence rule for B2/B3 |
| 5 | One header glyph per enabled provider; FIRS≡FIRMS | header component reads ProviderStatus roster |
| 6 | Volume UI 0–100 | dB display dropped; player state 0–100 |

Extracted component/UX rules (from mock notes): alert banner only for default/highlighted location, yellow=statement/warning, red=evacuation, paged with ←/→; radio player two sizes (`Shift+T`); priority list capped 10 (`n/10 Used`); viewport-aware recent-list row count (floating modals don't affect it); ultra-wide >125 col adds extended-forecast columns (BreakWide); temp colors: high=orange, low=cyan; trend arrow ↗/↘ + prose trend; detail-view alert typography: bullets indent 4 cols from text edge (not icon), multi-line bullets get breather lines above+below, single-line bullets none, right scrollbar gutter always reserved.
