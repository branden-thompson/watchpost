# Regressions and unruled behaviour changes — REVIEW red team, 0.19.5

`feature/quality-pass`, `7ca44489..ef9b9c27`. Filed whole as the reviewer returned it, its scratch
path shortened to `<scratch>`.

## Summary

I found no Critical, Important or Minor unruled behaviour change. Every change to what the program draws, says or persists that I traced or rendered comes back to a ruling row. The main evidence is OBSERVED: I rendered old and new side by side over about 8,000 frames. The rest is HYPOTHETICAL but checked against the code line by line.

## Method

### 1. Old-against-new frame differential (OBSERVED)

I made detached worktrees at 7ca44489 and ef9b9c27 under the scratch directory. I put the same throwaway test file in `modes/tty` of each and diffed what they rendered. Both worktrees are now removed and the repository tree is clean. The harness sources are kept in `<scratch>/rt/redteam-regressions/` (`zz_rtdiff_test.go`, `zz_rtcon_test.go`, `zz_rtradio_test.go`, `cmp.py`, `cmp2.py`). The helpers they call (`benchDash`, `routerOnConsole`, `benchTicker`) exist at both commits.

**Observer: 5,864 frames.**
- Sizes: 60x20, 80x24, 80x30, 100x30, 100x44, 133x44, 160x60, 220x70.
- Settings: colour on and off, imperial and metric, 12-hour, 24-hour and military clock, `--ascii` on and off.
- 17 key sequences: navigation into RECENT, `?`, `a`, `S`, `enter`, `w`, `s` and its tabs, `m`, `r`.
- After each sequence:
  - a publish that changes only provider health (the D-31 no-change path);
  - a publish that changes one location;
  - a publish that changes it back;
  - a ticker cut to 3 items, then to none (the tape memo);
  - an empty watchlist.
- **Result:** frames at 100x44 and taller are byte-identical in every combination, colour SGR included. Every difference is at 60x20, 80x24, 80x30 and 100x30, which are exactly the sizes where the old frame was taller than the terminal (31 or 32 rows, the empty watchlist included). That is D-28/D-30: a scrolling priority table with its rail, RECENT shrunk, overlays re-centred. The new frame fits at every size from 80x24 up. At 60x20 it is still 25 rows, but that is below the README's 80x24 minimum and down from 32.

**Radio player: 1,728 frames.**
- Playing, paused, loading and stopped, visualiser on and off, ascii, colour, 9 widths from 60 to 200, 6 heights from 24 to 70.
- **Result:** the only differences are frames where the old frame was taller than the terminal (D-28/D-30). The PF-3 plan/render split draws byte-identically everywhere else.

**Console through the Router: 960 frames.**
- 80x24 to 220x60; standby; the takeover rail; navigation, `tab`, `1`, `?`, `s`, `S`, `a`; both unit systems; 12-hour and 24-hour clock; ascii; colour.
- **Result:** every plain-text difference is one of:
  - the card top rule, two marks instead of three, with the masthead's give-way (D-18/D-32);
  - "DATA PULL" and "DATA PULLED" stamps in the clock Setting (D-29);
  - the pool in °C and km under metric (D-5).
- No line with identical plain text differs in its SGR anywhere in the colour console set. The Q3.4 single-pass finisher, the `CardRow` move away from go-studs (D-10), the railed-row memo (A-20) and the `Projection` cache (D-22) draw byte-identically.

### 2. Width differential (OBSERVED)

In the new worktree I compared `render.displayWidth` (the PF-2 sum over plain runs) with the shipped `runewidth.StringWidth(stripANSI(s))` on 400,000 random strings. The strings mixed SGR sequences, bare and partial ESC, `ESC[2K`, combining marks, ZWJ/ZWSP, VS15/VS16, emoji and skin-tone modifiers, regional indicators, Hangul jamo, Thai and Devanagari, CJK, the soft hyphen, tabs and newlines, and invalid UTF-8. There were 0 mismatches.

### 3. Settings table (OBSERVED)

I dumped `setupTable()` from both commits with `%+v`. The only difference is row 10, `rowReduceMotion`, where `picker` goes from false to true (D-36). Every other row of the positional-to-keyed rewrite (Q6.4) is identical.

### 4. Code reading of each memo and consolidation (HYPOTHETICAL, checked against code)

**Tables memo, D-31 (`modes/tty/memo.go:97-128`, `dashboard.go:1385`, `:1429`).**
- `priorityTable`, `recentSection`, `row`, `weatherRow`, `recentLocations` and `sharedExtDays` read only `snap.Locations`, `recent.Locations` and fields already in the key (`body.go:197-420`, `nav.go:145`).
- `Providers`, `Warnings` and `GeneratedAt` are read only by the masthead, which has its own memo, and by Status and Setup.
- `d.snap` and `d.recent` are assigned only at the two arrival sites. A write anywhere else stands for itself (`keyFor`).
- `DeepEqual` errs only toward a miss (NaN, monotonic clock).

**Masthead memo (`masthead.go:69-117`).** The key holds Opts without Frame, edition, version, data as-of, provider counts, age words, staleness, theme generation and colour switch. That is everything `mastheadTitle`, `mastheadStamp`, `apiSummaryOf`, `headerRow` and `controlRow` read. The controls are fixed literals.

**Tape memo (`ticker.go:329-394`).**
- `tapeLine` reads `it.*`, `o.ASCII` (glyphs), `o.Clock`, and `Since(t.Local(), d.clock())`.
- `Since` changes only when the date of `now` changes in `now`'s own zone, which is exactly the key's `Year`/`YearDay` of `d.clock()`.
- `setTicker` (`:481`) is the only writer of `d.ticker`, and it bumps a process-wide generation, so copies that share a slot cannot collide.
- `fl.o` and `d.opts()` agree on every field the key reads, so the shared slot does not thrash.

**Console.**
- `units` and `clock` were added to `consoleKey`.
- `railTables` reads only the spans, `frameWidth` and ascii.
- `projection` (`broadcaster_memo.go:95-109`) depends only on the line-up. `b.lineup` is written only at `broadcaster.go:532`, which bumps the global `lineupGens`; the old per-copy `+1` could collide across copies, the new counter cannot.
- No reader writes into the shared cards: I checked `broadcaster_air.go:130`, `detail.go:56-118`, `upnext.go:189` and `broadcaster.go:318`.

**PF-10 shared Hourly (`assembler.go:487`, `harmonize.go:100`).** I found no in-place write or append on `Location.Hourly` in app, domains, modes, platform or cmd.

**Consolidations, each equivalent on reachable inputs:**
- `FeltBand`: old and new boundaries agree at 3.5, 4.5 and 5.0, including NaN.
- `Compass16` and the 8-point `CompassIndex`: integer degrees give identical boundaries.
- `severe.Stamp`: it now defaults a nil zone instead of panicking.
- `WindowParts` and `spokenWindow`: no window, under 30 s ("0 minutes"), `1h30m` and `≥1d` all say the same as the shipped parser.
- `hourRow` (DR-13).
- `ClampRunes`.
- `units.MilesOf`: it is `km/KmPerMile`, the same arithmetic.
- `plural`, `eraseRune` (DEF-1).
- `providerFamilies.all`: the same provider order on both paths.
- `seenStore.dirty`: it starts dirty and covers add, prune and cap.
- `d.after` and `d.clock()`: both are `tea.Tick` and `time.Now` in production.

**`Router.update` and `Update` (Q3.1).** The old post-processing ran only on the `Router` branch, and `update` now always returns a `Router`. `overlay` (`debug.go:290`) computes the cursor on the base before overlaying, as `OverlayCursor` did.

**Arbiter rename.** Diffing old `director.go` (with names substituted) against `arbiter.go` with comments removed shows only renames: `v`→`voice`, `d`→`arbiter`.

**`executors.go`, `schedule.go`, `read_script.go`, `radio.go`, `pool.go`.** Only renames (A-23) and comments changed.

### 5. Tests and goldens

**Removed tests:**
- The `TestDirector*` tests are renamed to their arbiter form.
- `TestSpokenNameDropsTheLanguageSuffixAndTheSentinel` was deleted along with the dead `app.spokenName` (D-7).

**Assertions:** the changed ones follow D-18 (`broadcaster_cardbox_test.go`), D-29 (the clock stamps) and A-4 (`spokenWindow` takes a `Duration`). The memo hit-path bound changed from relative to absolute (62 × 1.10 allocations). Alloc budgets were only lowered. The four new `t.Skip` calls are race-detector skips on allocation pins.

**Goldens edited in place:**
- `map-80x24` and `setup-80x24` changed in batch 4 (D-28). The old frames were 31 rows, so the bottom seven rows never reached the screen.
- Every other golden is new, captured in batch 3 before the code it covers changed, plus `prefs-saved`, `model-from-config`, `setup-arrows` and `router-trace` as Q-task pins.

## Edges that change, each already recorded (no finding)

These are recorded in the build log under A-17 ("recorded as unreachable with its evidence"):
- `geo.OnEarth` now refuses NaN in `ParseCoordinates` and the GeoJSON walker. Unreachable: `coords.go` rejects NaN and Inf before the bound, and JSON cannot carry NaN.
- `httpx.HostOf`:
  - app's map-source column now returns the URL instead of "" when there is no host;
  - a URL that does not parse now comes back whole;
  - userinfo is dropped.

  Every caller passes a closed, constant base URL, so this cannot change today's Status window.
- The new layout counts the empty watchlist's 3-to-5-row message (`layout.go:84-91`, `priorityHeight`). Before, that frame was 1 to 3 rows taller than the terminal at every size, so this falls under D-28's "never exceeds the terminal, at any size".

## Observations (not findings; no rating)

- **O-1, HYPOTHETICAL.** `synth/source.go:367` and `:458` now decide a hand-off by the voice's full name (`voiceID`), while D-7 strips the language suffix from what is spoken. Two configured voices that share a first name in different locales would announce "This is Eddy, taking over from Eddy". The curated list (`app/voices.go:291-292`) has no shared first names, so only a hand-edited config could reach it. This is a consequence D-7 accepts ("the full name stays where it names the engine"), not an unruled change.
- **O-2, OBSERVED.** At narrow card widths the D-29 12-hour stamp (`5:45:00 PM`) is three cells longer than `17:45:00`, so "(2 MIN AGO)" is cut sooner. This is a consequence of D-29 at the widths my harness drew (for example the 80-column console). It is worth one look at the screenshot retake (A-9), not a finding.
- **O-3.** `fitPriority` stops at three favourites. Below about 22 rows the frame still runs past the terminal: 25 rows at 60x20, against 32 before. That is outside D-28's "down to 80x24" and improved, not regressed.

## Not covered

- I did not run the app, any network call, `make verify`, the mutation instruments or fuzzing.
- `tools/` and `scripts/` were not reviewed: they change gate tooling, not program behaviour, and the Makefile path changes are D-16 and D-39.
- The live map window was not rendered: it needs a map builder. Its changes are `viewWidthKm` (DR-10, identical arithmetic) and `d.after` timers.

**Verdict: EXIT** (no Critical or Important findings)
