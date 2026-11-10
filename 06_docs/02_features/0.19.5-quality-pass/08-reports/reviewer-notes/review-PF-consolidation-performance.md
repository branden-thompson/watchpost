# Consolidation and performance — REVIEW red team, 0.19.5

`feature/quality-pass`, `7ca44489..ef9b9c27`. Filed whole as the reviewer returned it, its scratch
paths shortened to `<scratch>`.

**Verdict: DO NOT EXIT.** There is one Important finding: the M2 instrument does not measure the program's real idle beat. There are no Critical findings.

I ran my measurements in scratch worktrees under `<scratch>/rt/redteam-perf/`. Both worktrees are removed, and the repo tree is untouched.

## Important

### P-1. The beat pin re-runs one message on the same model, so the idle beat skips a 16 KB per-tick frame copy
OBSERVED, `modes/tty/beat_alloc_test.go:36-46`.

- **What the pin does:** `beatCost` calls `m.Update(b.msg)` on the same `m` every iteration and never sets `m = next`. Each iteration draws the same frame, so `cursorSlot.finish` hits on `s.in == content` (`cursor.go:68`).
- **What the program does:** every 300 ms tick scrolls the marquee, so the frame changes. `finish` then misses and `focusCell` → `render.PaintCell` copies the whole frame.
- **Measured at HEAD** (same beats, but feeding each `next` back in):

  | Beat | Pin measures | Real (threaded) |
  |---|---|---|
  | Idle Observer 133x44 | 59,556 B | **75,993 B** |
  | Radio playing | 68,474 B | 68,476 B |
  | Console 160x74 | 122,394 B | 122,400 B |

- **Where the gap comes from:** a memprofile diff puts all of it in `cursorSlot.finish`/`focusCell`/`PaintCell`, at 16 KB a beat.
- **Base 7ca44489, idle, same method:** 156,527 B as the pin measures it, 172,963 B threaded. So the cut is real (173 → 76 KB) and the 80 KB target is still met.
- **The problem:**
  - The recorded M2 figure and the CHANGELOG ("an idle Observer frame from about 157 KB to 60 KB") understate the real beat by 27%. The real headroom against 80 KB is 5%, not 25%.
  - The pin cannot see a regression in any work that runs only when the frame changes. That is every tick in the app, and the frame hit pins in `bench_test.go:177` share the same blind spot.
- **Fix:** thread the model in `beatCost`, re-pin, and correct the build-log M2 tables (`build-log.md:584,640,971`) and the CHANGELOG figure.

## Minor

### P-2. The frame "miss" pins now run with the masthead and tape memos warm, and the 80x24 snapshot frame is over its pin
OBSERVED, `bench_test.go:178` and `tape_memo_test.go:40-44`.

- `TestFrameAllocBudget` says its miss is "a snapshot, key or resize", but it resets only `d.memo.ok`. A snapshot or a resize also misses the new `memo.head` (the masthead stamp's `asOf` and counts, or `Opts.Width`).
- Measured with `head.ok` and `tape.ok` reset as well:

  | Size | Pin measures | Head and tape reset too | Budget |
  |---|---|---|---|
  | 133x44 | 5,387 | 5,523 | 5,656 |
  | 80x24 | 985 | **1,136** | **1,034** |

- `missEveryMemo` says it "empties every memo the frame reads" but leaves `head` alone.
- The masthead build itself is covered once, by `TestPublishAllocBudget`, which changes `FetchedAt`. So this is about the pins' stated meaning, not a hole in coverage.
- **Fix:** reset `head` in `missEveryMemo`, use it in `TestFrameAllocBudget`, and re-pin.

### P-3. `TestPublishAllocBudget` compares a shallow copy, not one "as the assembler makes one"
OBSERVED, `same_content_test.go:15-27` and `beat_alloc_test.go:83`.

- `republished` copies the `Location` structs shallowly. Their `ByProvider` maps and `Daily` slices stay pointer-identical, so `reflect.DeepEqual` (`memo.go:116`, D-31) takes its same-pointer shortcuts.
- `Assembler.Snapshot` builds fresh maps, fresh `Daily` and fresh `Current` pointers.
- On `assembler25` (25 locations), one equal comparison costs:
  - two real publishes: 61.7 µs, 50,272 B, 165 allocs;
  - the shallow form: 22.3 µs, 21,272 B, 13 allocs.
- It runs for both the watchlist and the RECENT publish. D-31's "22 allocations and 72 µs" and this pin both understate the real cost, which still beats redrawing the tables.
- **Fix:** build the republished snapshot through an `Assembler`.

### P-4. PF-2's no-allocation `render.Width` falls back to strip-and-measure on coloured braille
OBSERVED, `platform/render/text.go:466-482`.

- `standsAlone` leaves out Braille Patterns U+2800–28FF, which never join a grapheme cluster.
- A 60-cell coloured row: braille costs 1 alloc and 1,152 B per `Width`; box drawing costs 0.
- The map window's braille rows carry colour spans (`map_alloc_test.go:18`), so PF-2 saves nothing there.
- The answer stays correct, and nothing is worse than at base.
- **Fix:** add `r >= 0x2800 && r <= 0x28ff`. The fuzz target already guards correctness.

### C-1. Small DRY slips added by the diff
OBSERVED.

- **Duplicate ESC constant:** `render.plaintextEsc` (`text.go:451`) re-declares `plaintext.escByte`, though `plaintext` now exports `SGRLen` beside it.
- **Duplicate regex:** `tools/authoring/cite.go:71` `releaseNamed` and `tools/ids/main.go:441` `versionIn` are the same version regex. `tools/ids/idset` was created to be their one home.
- **Duplicate test:** `tools/ids/ids_test.go:95` and `:216` are one 209-node test body, the largest new duplicate group in `dupes -tests`.
- **Not a finding:** the console finisher's `writeArmed` (`broadcaster.go`) re-states `render.TintKeeping`'s rule. That is acceptable, because `clampShipped` calls `TintKeeping` live, so the two cannot drift silently.

## Checked and sound
Each was traced against the code at HEAD.

- **Tape memo** (`ticker.go:330-381`):
  - The key is complete: `tapeLine` reads only the ticker, the ascii flag, `o.Clock` and the model clock's date, and `Since` compares the model clock's own day, which is what the key holds.
  - `setTicker` is the only writer (`ticker.go:481`). The tests write through it, and a zero generation skips the memo.
- **Masthead memo** (`masthead.go:70-112`): the key covers everything `headerRow`/`controlRow`/`apiSummaryOf`/`mastheadStamp` read. That includes `staleAfter` and `agoWords` through `ago`/`stale`, plus the theme and colour switch. `Version` is constant.
- **D-31 same-content check:**
  - The tables read only `Locations` (`body.go:199-470`), and nothing there depends on the time.
  - It keeps at most one earlier snapshot alive.
  - A NaN only costs a memo hit.
- **PF-10 shared hours:** `Apply` copies on entry (`assembler.go:269`), and I found no in-place write to a published `Hourly`.
- **Projection cache** (`broadcaster_memo.go:99-108`):
  - The line-up is replaced only at `broadcaster.go:532`, and its generation is now process-wide, which fixes the old per-copy counter.
  - All eight readers only read; the slices in `mainTrack` are re-slices, not writes.
- **Railed rows (A-20):** they read the spans, the width, the ascii flag and the theme, all in the key.
- **Size:** every memo is a single entry.
- **Locking:** `bodyMemo.mu` guards only the table fields. The new `tape`/`head`/`proj` slots go unlocked, like the existing `cursorSlot`. That is safe on Bubble Tea's single event loop (HYPOTHETICAL only if a frame were ever drawn off that loop).
- **Budgets:** none was loosened. Every frame, map, severe, Settings, propagation, tick and console pin went down.
- **Consolidations:**
  - `FeltBand`/`SeismicLevel`/`feltLikelihood`/`feltLabels` match the old boundaries, NaN included.
  - `CompassIndex` and `Compass16` differ from the old forms only at an exact negative half-sector, which no feed sends.
  - The `HostOf` edges are recorded under A-17, and it now drops userinfo.
  - `hourRow` gives each measure its own NaN row.
  - `Class.row`, `ClampRunes` and `OnEarth` are equivalent.
  - The leftover copies (temperature's `missing`, `ToneRepeats`, `radarAheadLabel`/`hoursWords`) are deliberate and recorded.
- **`seenStore.dirty`:** it is set on every change to `ids`, and a failed write stays dirty.

**Verdict: DO NOT EXIT** (P-1 Important; the rest are Minor and can go in one batch).
