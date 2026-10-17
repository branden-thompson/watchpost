# Memo keys: identity against position (#12, FR-8)

Up: [architecture](architecture.md) · [requirements, FR-8](../01-objectives/requirements.md) · 0.19.0 W9.1, batch 33

**What this is.** Issue #12 asked that every memo key be checked against what its window reads: where a
field is a position - an index, an offset, a row - is the identity of what it points at keyed beside
it? A position alone is one "same slot, different contents" away from a stale frame (#11 was one). This
is the audit as the code stands; two tests hold it.

| Held by | What it checks |
|---|---|
| `TestEveryPositionFieldHasAReason` (`modes/tty/memo_positions_test.go`) | every integer field of `bodyKey`, `modalKey` and `consoleKey` named as a position carries a `position:` comment saying what keys the identity it indexes; a field added without one fails |
| `TestTheAppMemoKeysCoverTheirInputs` (`app/memokeys_test.go`) | every field of the map's ask is carried by the temperature answer's key or said not to be its input, each carried field changes the key, and the keys named by their data are pinned field by field |
| `TestTheMemoKeyCoversEverythingTheFrameShows` (`modes/tty/memo_completeness_test.go`, F-30) | two dashboards that draw differently never share a window key, every field walked by reflection |

## The windows' keys: every position, and what keys its identity

| Key | Position | What it indexes | Its identity, keyed by |
|---|---|---|---|
| `bodyKey` | `selected` | the row marked | `snap`, `recent` (pointers: a publish hands over a new snapshot) and `lookup` |
| `bodyKey` | `recentOff` | the scroll into the recent rows | `recent` |
| `modalKey` | `scroll` | the window's lines | every other field of the key |
| `modalKey` | `selected` | as `bodyKey`'s | `snap`, `recent`, `lookup` |
| `modalKey` | `alertIdx` | an alert of the selected place | `snap`, `recent` |
| `modalKey` | `severeRow` | a row of the severe index | `severeGen`, `severeTab` |
| `modalKey` | `voiceIdx` | a voice of the list | the list is taken when the chooser opens and fixed while it is open; `nvoices` |
| `modalKey` | `mapMenuAt` | a row of the Overlays menu | `mapLayers`, `mapDetail`, `mapLevel` and the mode |
| `modalKey` | `faultFocus` | a button of the fault window | a fixed set (`faultLeft` beside it is its clock) |
| `modalKey` | `debugFocus` | a row of the ctrl+d window | a fixed list |
| `modalKey` | `reqAt` | a row of the report kinds | a fixed set |
| `modalKey` | `chartRow` | a row of the chart | the chart's fixed targets |
| `consoleKey` | `selected` | a line-up slot, and the pool's row | `lineupGen`, `areaGen`, `recent`, `priority` |
| `tempMemoKey()` | `fcStep` | a step of the forecast | the timeline's words beside it in the same key |

No position needed to become an identity: each indexes a list whose identity its key already carries,
or a fixed set. The comments at each field say the same, and the test reads them.

## The app's keys

| Key | Built from | Why nothing else changes its answer |
|---|---|---|
| `tempKey` (`app/maptemp.go`) | the ask's region, view (as the region's boxes), source, mode, unit, rain density, hours ahead and anchor, and whether Open-Meteo is held | every other field of the ask is classified in `tempKeyInputs` with why it is no input |
| `frameKey` (`app/mapradar.go`) | source, region, box, time, HRRR's run, size | a radar frame at its time never changes |
| `airGridKey` (`app/mapairuv.go`) | box, hour | AirNow's contours are one file an hour |
| `uvDayKey` (`app/mapairuv.go`) | city, state, date | EPA's forecast is a city's day |
| the area memo (`app/mapinview.go`) | the areas the view touches, by their codes | an area's alerts are the area's; kept two minutes, the ticker's cadence |
| `feedKey` (`modes/tty/map_pane.go`) | the place, and the view's generation | the view's box follows the map's size with no move; keyed on it, answers would be dropped (W14) |
