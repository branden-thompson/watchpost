# Answer key, T2: the UV legend's head says "UV INDEX"

> **0.19.6 M4 answer key (D-24's method, JN P2-8), hidden from the blind reviewer.** Built on a scratch
> copy of `e57afeaf` (`git archive`, one commit `base`) by doing the task and leaving out one site at a
> time. Every run is in `<scratch>/plan-KB/runs.log`, its output in `<scratch>/plan-KB/logs/<label>.log`.
> Toolchain go1.27.4, `GOTOOLCHAIN=local`, no network. Lines "in the copy" are the finished change's.

Scratch copy: `<scratch>/plan-KB/t2` (base `faa527b`). The complete change: `<scratch>/plan-KB/t2-complete.diff`;
every touched file: `<scratch>/plan-KB/t2-complete/`.

## The task, as given to the reviewer

> Under the map, in Forecast mode with the UV layer on, the legend row reads `UV │ LOW … EXTREME`.
> Listeners take "UV" for a unit. Make the row's head read `UV INDEX │` instead; nothing else on the row
> changes.

## The change that was made

The head string of the UV case in `tempLegendRow` becomes `"UV INDEX │ "`; the one test that pins the old
head is updated; a CHANGELOG entry is written.

Why this instance: JN-3's route (`map_pane.go:617` → `map_scrub.go:56` `scrubRows` → the three legend
functions) with no walkthrough, one layer, one visible word.

Measured facts it depends on (probe test in the copy, deleted after; `probe-uv`, `probe-uv-after` in
`runs.log`):
- **The UV legend has five classes**, labelled by go-tuimaps v0.3.2: "under 3", "3 to 6", "6 to 8",
  "8 to 11", "11 and above".
- **The row is drawn only in Forecast mode, and only when no rain is drawn.** `scrubRows`
  (`modes/tty/map_scrub.go:58-65` at `e57afeaf`) takes Radar mode's row first, then Forecast mode's rain
  row, and only then `tempLegendRow`. Inside it the `switch` order is the precedence: UV, air quality,
  feels-like, wind, temperature (`modes/tty/map_temp.go:803-815`).
- **The row still fits at the map's floor.** `mapMinBody` is 69 columns (`modes/tty/map_settings.go:20`);
  the text width there is 68. `presetRow` (`map_temp.go:842`) drops the whole row only when
  `room < classes*2`. With the six extra cells, at 68 columns each class gets 9 cells instead of 10 and
  the last label reads "11 and a" where it read "11 and ab". At 40 columns all five are still drawn.
- **No golden draws any legend row.** No file under `modes/tty/testdata` holds `UV │`, `RADAR LEGEND`,
  `TEMPERATURE │` or any other head (grep). So no golden moves, and `-update-golden` is not needed.

## Every site the change needed

| # | File | Symbol | What changes | Line in the copy |
|---|---|---|---|---|
| S1 | `modes/tty/map_temp.go` | `Dashboard.tempLegendRow`, the `UVLayer` case | `"UV │ "` → `"UV INDEX │ "` | 805 |
| S2 | `modes/tty/map_airuv_test.go` | `TestSwitchingUVOnAsksForIt` | its `HasPrefix(row, "UV │")` → `"UV INDEX │"` | 43 |
| S3 | `CHANGELOG.md` | a new `## [Unreleased]` / `### Changed` entry | the visible change in a sentence (the heading does not exist at `e57afeaf`) | 3-9 |

**Counts: 3 sites, in 2 source files and the CHANGELOG; 0 goldens. NO GUARD: 1 (S3).**

## Leave-one-out

Each row: that one site undone in the finished copy, `go test -count=1 ./modes/tty` whole, the file put
back from `t2-complete/` and the tree compared with the complete diff (`RESTORED OK` each time).

| Site left out | What fails (package `modes/tty`) | Run |
|---|---|---|
| S1 | `TestSwitchingUVOnAsksForIt`: `the key row is "UV │ LOW under 3 … EXTREME"; want UV's, low to extreme` | `t2-omit-S1` |
| S2 | `TestSwitchingUVOnAsksForIt`: `the key row is "UV INDEX │ LOW …"; want UV's, low to extreme` (the old pin fails against the new head) | `t2-omit-S2` |
| S3 | **NO GUARD**: no test reads `CHANGELOG.md` (grep of every `_test.go` and `.go` file for `CHANGELOG` and `Unreleased`: none) | by grep, no run |

The complete change: `go test -count=1 ./modes/tty` **ok** (`t2-complete`). The baseline at `e57afeaf`:
**ok** (`base-tty`).

**The guard is a single behaviour test, and its message misleads.** The one test that sees the head says
"want UV's" whichever head it got; a junior reading the failure is not told the head is pinned at line 43.

## The docs that route a junior to each site today (`e57afeaf`)

| Site | Route | Gap |
|---|---|---|
| S1 | No Start-here document names the legend. The nearest: `docs/where-things-happen.md:24` ("The map opens") names `modes/tty/map_pane.go:mapBodyLines`; its body appends `d.scrubRows(width)` at `map_pane.go:617`; `scrubRows` (`map_scrub.go:56`) calls `tempLegendRow` (`map_temp.go:802`). Two hops by reading code. `modes/tty/doc.go:31-32` covers it only by the `map_*` wildcard. No walkthrough: Walkthrough 5 (`docs/extending.md:143-160`) adds a layer and never mentions the row under the map | **NONE** direct |
| S2 | None; the failing test names itself | NONE (the guard routes) |
| S3 | `docs/extending.md:242` (Walkthrough 8, step 7) says a change "the operator can see" goes under `[Unreleased]`. That is the console's walkthrough; nothing says it for an Observer change | NONE for this task |

## Found along the way (at `e57afeaf`)

1. **A stale route to the legend.** `modes/tty/map_boxes.go:7-8`: "The legend over the upper right is
   map_pane.go's (D-44)". `map_pane.go` draws no legend over the upper right; the legend is a row under
   the map (JN-3 found it first; still there).
2. **Three names for one row.** The functions say "legend" (`radarLegendRow`, `map_radar.go:391`;
   `tempLegendRow`, `map_temp.go:802`; `propLegendRows`, `map_prop_draw.go:170`). `map_scrub.go:3-4`
   calls it "the colour row". The map's as-built text calls it "the colour row" and also says "no legend
   (retired, D-103)" (`06_docs/02_features/observer-maps/03-architecture-design/as-built-map.md:34`). A
   junior who searches the as-built for "legend" is told there is none.
3. **The default legend's head has no guard at all.** `"TEMPERATURE │ "` (`map_temp.go:815`) is pinned
   by no test and no golden (grep). The other Forecast heads each have one `HasPrefix` or `Contains` in a
   behaviour test: UV `map_airuv_test.go:43`, air quality `:60`, feels-like `map_feels_test.go:101`, wind
   `map_temp_test.go:660` (`Contains "WIND"` only).
4. **The precedence is said nowhere but the code.** That rain wins over every Forecast tint, and UV over
   the rest, lives only in the order of `map_scrub.go:58-65` and `map_temp.go:803-815`; the comment at
   `:803` says only "the key of what is drawn".
5. UNVERIFIED: whether the literal `│` in every head (`map_temp.go:805-815`, `map_radar.go:392`)
   is right under `--ascii`. Walkthrough 8 says a literal `…` breaks `--ascii`
   (`docs/extending.md:238`). No ascii-parity test was run here.

## After 0.19.6

S1 and S2 move with the map package (TS Map II); S3 does not move. Unless the map package's `doc.go` names
the four rows under the map with one word for the legend (JN-3's recommendation), the route stays as
above, one package further from `where-things-happen.md:24`.
