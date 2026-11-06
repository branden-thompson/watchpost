# Lens PF: performance (0.19.5 quality pass)

Reviewer: PF lens. Tree: `4c803d99` (v0.19.3), read-only. The compositor, the renderer and the Elm
loop are not touched by anything below (F-220, 0.19.0 D-327). Every finding sits in watchpost's own
code.

## How this was measured

- **Allocations and bytes per op are the evidence. Wall-clock times are not used.** Another reviewer
  was running tests on the same machine, and a clean CPU profile of the hit frame was 46 % `kevent`
  (scheduler noise), so wall-clock figures could not be trusted.
- Existing benchmarks (`make quality-bench` set plus `map_*`, `ConsoleFrame*`) ran with
  `-benchmem -count 3`. Allocation profiles came from `-memprofilerate 1`.
- **New benchmarks and every prototype live only in the scratch copies**, never in the repository:
  `<lens scratch>/pf/wp0/` is a pristine rsync of the tree,
  and `<lens scratch>/pf/wp/` is the same tree with the prototypes applied. The scratch benchmarks are in
  `modes/tty/pf_scratch_test.go` in each copy.
- **Each prototype was checked for identical output** (an old-vs-new comparison test, named in the
  findings below). With the prototypes applied, `go test ./modes/tty ./app` passed in the scratch
  copy.
- The existing allocation pins all pass on the tree as it is (`go test -count=1 -run AllocBudget`).
  Some have very little headroom: the 80x24 hit frame measures 395 against a budget of 396, and the
  severe window 1,814 against 1,827.

### What an app beat costs (bench fixture, 133x44 unless stated)

A "beat" is one message going through the **Router**, which is the real `tea.Model`, followed by
the frame Bubble Tea then draws. Bubble Tea v2 calls `View` after every message
(`bubbletea/v2@v2.0.3/tea.go:880`).

| Beat | Shipped | All S-sized fixes below (prototyped) | Change |
|---|---|---|---|
| Idle Observer, 300 ms tick (Router Update + View) | 156.5 KB, 462 allocs | 104.9 KB, 397 allocs | −33 % bytes |
| Radio playing, visualiser on, 50 ms beat | 169.7 KB, 694 allocs | 112.5 KB, 618 allocs | −34 % bytes |
| Console 160x74, 300 ms tick | 310.5 KB, 782 allocs | 161.3 KB, 463 allocs | −48 % bytes, −41 % allocs |
| `BenchmarkFrame_133x44` (memo hit, `Dashboard.View`) | 89.2 KB, 379 allocs | 67.6 KB, 318 allocs | −24 % bytes |
| `BenchmarkFrame_133x70` | 114.9 KB, 397 allocs | 89.6 KB, 278 allocs | −22 % / −30 % |
| `BenchmarkConsoleFrame` | 106.2 KB, 503 allocs | 89.8 KB, 356 allocs | −29 % allocs |

What that means per minute on this fixture:

- **Idle Observer:** about 31 MB/min today, about 21 MB/min with the fixes.
- **Radio with the visualiser on:** about 23 beats a second, so about 200 MB/min today and about
  135 MB/min with the fixes.
- **Console at 160x74:** about 62 MB/min today, about 32 MB/min with the fixes.

The collector's CPU follows the allocation rate; F-220 already puts about 0.3 points of a core into
the collector on the map arm. None of these figures needs the live app.

---

## Findings, most valuable first

### PF-1: Every message copies the 6.5 KB Dashboard and the 7 KB Router onto the heap 4 to 6 times
- **Where:**
  - `modes/tty/router.go:597` (`scopedMessage`, fall-through `return r, nil, false`)
  - `modes/tty/router.go:675` (`routeKey`, fall-through)
  - `modes/tty/router.go:504` (`update`, `return r, cmd`)
  - `modes/tty/router.go:525` (`r.surface().View()`, a `surfaceView` interface conversion)
  - `modes/tty/dashboard.go:1185-1192` (`dispatch` re-boxes the result into `tea.Model`)
  - `modes/tty/dashboard.go:1236`, `:1240` (`onTick`, `vizFrame`, which reach `tea.Model` through
    `takeCmd` / `applyTick`)
  - the same pattern is in all 53 `func (d Dashboard) …(…) (tea.Model, tea.Cmd)` handlers
- **Observed:**
  - `unsafe.Sizeof`: Dashboard is 6,536 bytes (mapPane 3,248, cfg 1,320) and Router is 7,072 bytes.
  - Turning a struct that size into an interface value copies it onto the heap.
  - `Router.Update(tickMsg{})` costs **60.4 KB in 82 allocations**. The profile attributes 7 KB each
    to `Router.update`, `Router.routeKey` and `Router.scopedMessage`, and 6.5 KB each to
    `Dashboard.dispatch`, `Dashboard.onTick` and `Dashboard.armTick`.
  - The fall-through returns box `r` even though the caller throws it away:
    ```go
    // NOT SCOPED TO ANYONE IN PARTICULAR: the key paths get their turn.
    return r, nil, false
    ```
  - `Router.View` costs one alloc and 6.8 KB more than `Dashboard.View`, which is the
    `surface()` boxing.
  - Only one box per message is required: Router's final return to Bubble Tea.
- **Cost:**
  - A tick message carries 60 KB, and about 40 KB of it is model copies. That is roughly 8 MB/min
    idle and roughly 50 MB/min with the visualiser running.
  - **No pin sees it.** The pins count allocations, and this is 5 allocations, not 50,000 (see PF-6).
- **Proposed change:** three S-sized edits, prototyped and measured in the scratch copy:
  1. The two fall-throughs return `nil, nil, false`. Callers read `m` only when `handled`, at
     `router.go:490-496`.
  2. `Router.View` switches on `r.active` and calls `r.observer.View()` /
     `r.broadcaster.View()` directly.
  3. `dispatch` returns `(Dashboard, tea.Model, tea.Cmd, bool)`, so `Update` gets the concrete value
     back without a box.

  Measured effect:
  - Router tick Update: 60.4 KB → **37.2 KB**.
  - Router viz Update: 53.7 KB → **30.6 KB**.
  - Router View: −6.8 KB per frame.

  A follow-on change (M) would get to one box per message: `Router.update` returns `Router`, and the
  Router calls an internal `Dashboard.update` returning `Dashboard`. `Update` keeps its `tea.Model`
  signature, so Bubble Tea is unaffected. That saves about another 13.5 KB per message. The comment
  at `router.go:866-872` ("Observer keeps its tea.Model signature … a signature change would make it
  more than one [delegation]") would need rewording, but the public `Update` does not change.
- **Behaviour-preserving by construction:** the same values move through concrete types instead of
  interfaces.
- **Regression risk:** none in output. Routing is pinned by the Router suites, for example
  `TestARequestSubmittedFromObserverStillLandsInTheTypedSlot` in
  `modes/tty/request_slot_test.go:171`. `TestRouterCostsObserverAlmostNothingPerFrame` pins the
  View overhead in allocations. `go test ./modes/tty
  ./app` passed in the scratch copy with the three edits.
- **Size:** S for the three edits; M for full de-boxing.
- **Confidence:** high. Measured with the Router benchmarks in
  `<lens scratch>/pf/wp/modes/tty/pf_scratch_test.go`
  (`BenchmarkPF_RouterTickUpdate`, `BenchmarkPF_RouterVizUpdate`, `BenchmarkPF_RouterView`).

### PF-2: `render.Width` makes a stripped copy of every styled string it measures
- **Where:** `platform/render/text.go:410`. It is reached from `Width`, `PadTo`, `PadBetween`,
  `FirstFit` and the masthead, the ticker and the radio ladders.
- **Observed:**
  ```go
  func displayWidth(s string) int { return runewidth.StringWidth(stripANSI(s)) }
  ```
  - `plaintext.StripSGR` builds a new string whenever the input holds an escape, which on a colour
    frame is nearly always.
  - `BenchmarkDisplayWidth_styledRow`: 1 alloc and 416 B per call.
  - On the hit frame, `displayWidth` accounts for about **55 allocs of 379 (15 %)**.
- **Cost:** about 55 allocs per frame on every frame, and about 116 on a memo miss.
- **Proposed change:** sum `runewidth.StringWidth` over the plain segments between SGR sequences.
  Those segments are substrings, so nothing is allocated. Two guards keep the result the same:
  - Fall back to today's strip-then-measure when a rune touching an escape could join a grapheme
    cluster across it: combining marks, ZWJ, regional indicators, emoji modifiers, variation
    selectors, Hangul jamo, Prepend.
  - Export `plaintext`'s `sgrLen`, so the grammar keeps one owner.

  Measured effect:
  - `BenchmarkDisplayWidth_styledRow`: 1 alloc / 416 B → **0 / 0**.
  - Hit frame: 379 → **324** allocs, 89.2 → 80.5 KB.
  - Radio frame: 677 → 611.
  - Settings hit: 2,866 → 2,806.
  - Miss frame: 5,676 → 5,560.
- **Behaviour-preserving:** not by construction, but **verified**. One million random strings drawn
  from escapes, CJK, emoji with modifiers, ZWJ, flags, Hangul, Devanagari and box glyphs gave
  identical widths, and so did every row of a 50-row styled table. Without the fallback, 8,708 of
  200,000 adversarial strings differed, all of them clusters split by an escape. That is why the
  fallback is in the proposal.
- **Regression risk:** every width-dependent layout. The golden and layout suites in `modes/tty` and
  `platform/render` passed in the scratch copy. The one failure was
  `TestDeclarationSetUnchanged` (`platform/render`), which is the declaration-set guard reacting
  to the new helpers; it is expected and needs updating. The randomized equivalence test should land
  with the change.
- **Size:** S.
- **Confidence:** high.

### PF-3: The frame builds the radio player in both layouts every time and draws one of them
- **Where:** `modes/tty/layout.go:39`
  ```go
  rows := layoutRows{full: d.radioLines(o, false), compact: d.radioLines(o, true), controlRow: d.controlRow(o)}
  ```
- **Observed:**
  - `layoutWith` reads `built.compact` only when `fl.compact` is true (`layout.go:76-79`).
  - At the narrow breakpoint the two builds are identical: `radioLines` takes `radioCompactRows`
    when `compactMode || bp == radioNarrow` (`radio_panel.go:131`).
  - Measured per build:

    | Size, state | Full build | Compact build | Which is drawn |
    |---|---|---|---|
    | 133x44, radio on | 231 allocs | 192 allocs | compact |
    | 133x70, radio on | 231 allocs | 192 allocs | full |
    | 200x60, radio on | 163 allocs | 189 allocs | full |
    | idle, any size | about 66 allocs | about 65 allocs | — |

  - At 133x44 and 149x38 (the standard workload's size) the frame is compact, so the full player is
    built only for its row count. At 133x70 and 200x60 the compact player is built and thrown away.
- **Cost:**
  - Radio on: up to 28 % of the frame thrown away, about 20 times a second.
  - Idle: about 17 % of the frame thrown away.
- **Proposed change, step 1 (S, behaviour-preserving by construction):** build the compact rows only
  when the layout is compact, and reuse the full rows at the narrow breakpoint. Measured:
  - 133x70: 397 → **332** idle, 695 → **503** radio on (−28 %).
  - 200x60: 400 → **335** idle.
  - 80x24: 377 → **339** idle.
- **Proposed change, step 2 (M):** split `radioMaxRows` into a plan and a render, so the full
  player's height is known without drawing its track or visualiser.
  - The plan covers the head/tail ladder, the narrow decision, the visualiser row count and the
    control rows.
  - Title, state and controls are built once per frame and shared by both.
  - An unshared prototype took 133x44 radio-on from 611 to **479** (−22 %) and matched
    `len(radioLines(o,false))` across 12 widths × visualiser × playing (`TestPF_FullHeightMatches`).
  - Sharing the parts is needed so the idle frame does not pay the plan on top of the build: the
    unshared prototype raised idle 133x70 by 48 allocs.
- **Regression risk:** `TestFrameAllocBudget` and the layout and golden suites in `modes/tty`
  (green with step 1 in the scratch copy). Step 2 needs the height-equals-rows property test to land
  with it. "One builder feeds both rendering and the height budget" (`radio_panel.go:105`) stays
  true if the plan is the builder's first half.
- **Size:** S (step 1); M (step 2).
- **Confidence:** high.

### PF-4: The console's frame finisher copies the whole frame three times and allocates three times per row
- **Where:** `modes/tty/broadcaster.go:726`, `:749`, `:751` (`Broadcaster.clamp`)
  ```go
  lines[i] = render.PadTo(bcLeftInset+render.TruncateCells(l, b.frameWidth()), b.width)
  ...
  out := strings.Join(lines, "\n")
  ...
  out = render.TintKeeping(out, render.FgSGR(render.Tok(render.TextBase)))
  ```
- **Observed:**
  - Per row: a concatenation, a `Width` strip, a `strings.Repeat` and a concatenation.
  - Then `Join` copies the frame, and `TintKeeping` (`platform/render/sgr.go:137`) copies it twice
    more (`ReplaceAll` and a concatenation).
  - At 160x74, clamp is **47 % of the console frame's bytes** (143 KB of about 266 KB).
- **Cost:** the console redraws on every tick, so this is a 24/7 cost while the console is up.
- **Proposed change:** one `strings.Builder`, sized up front, doing the work Observer's `frameText`
  (`modes/tty/view.go`) already does in one pass:
  - write the inset;
  - write the truncated row, swapping each `\x1b[0m` for the base (the `TintKeeping` rule);
  - pad with spaces measured by `Width(t)` plus the inset;
  - close with a reset.

  Measured effect:
  - `BenchmarkConsoleFrame`: 503 → **357** allocs, 106.2 → 89.8 KB.
  - 160x74 console beat: 310 → **208** KB, of which about 21 KB is PF-1.
- **Behaviour-preserving:** byte-identical, verified with colour on at 100x44, 133x44, 160x74 and
  220x60 against the shipped function (`TestPF_ClampSame` in the scratch copy). The reset never
  spans a newline, so folding the tint per line equals tinting the joined frame.
- **Regression risk:** `TestConsoleFrameAllocBudget` (`broadcaster_alloc_test.go`) and the
  console's width and inset tests (D-96, D-102, D-108).
- **Size:** S.
- **Confidence:** high.

### PF-5: The marquee turns the whole tape into a rune slice on every frame to cut out one window
- **Where:** `modes/tty/ticker.go:261` (`scrollWindow`)
  ```go
  loop := []rune(text + gap)
  ```
- **Observed:**
  - A full warnings lane is about 2,400 runes, so this is about 9.6 KB of `[]rune` plus the
    concatenated string.
  - `scrollWindow` is **87.6 MB of 607 MB (14 %) of the hit frame's bytes**, about 12.3 KB per
    frame, in one or two allocations. The count pins cannot see it.
- **Proposed change:** walk the two strings with `utf8.DecodeRuneInString`. Skip `off` runes, then
  take runes, wrapping from the text to the gap.

  Measured effect:
  - Marquee: 21.1 KB → **8.2 KB** per frame.
  - Hit frame: 89.2 → **76.3 KB** (−14 % bytes).
- **Behaviour-preserving:** byte-identical, verified against the shipped function
  (`TestPF_ScrollWindowSame`): every offset from −3 to 260, widths 1 to 120, both gaps, wide runes
  and invalid UTF-8.
- **Regression risk:** the ticker suites in `modes/tty` (`ticker_test.go`), all green in the
  scratch copy.
- **Size:** S.
- **Confidence:** high.

### PF-6: The pins count allocations, not bytes, and none measures the frames the app draws most
- **Where:**
  - `modes/tty/bench_test.go:24-56` (`frameAllocBudget*`)
  - `modes/tty/tick_alloc_test.go:24`
  - the `benchDash` fixture (`bench_test.go:89`)
- **Observed:**
  - **Every pin measures `Dashboard.View` or `advanceTicker`.** The one Router pin,
    `TestRouterCostsObserverAlmostNothingPerFrame` (`broadcaster_alloc_test.go:64`), counts
    `Router.View`'s *extra allocations*. PF-1's 6.8 KB box is one allocation, so that pin passes
    while the box costs a full copy of the model on every frame. Nothing measures:
    - the Router's Update, which is what the app runs: 60 KB per tick, as 82 allocs;
    - `Update` itself: 27.6 KB per Dashboard tick, of which only `advanceTicker` is pinned;
    - the radio-playing frame. `benchDash` has no radio, but the frame drawn at 20 Hz while it plays
      costs **677 allocs and 109 KB**, against the pinned 365.
  - The two byte hogs above (PF-1's boxes and PF-5's rune slice) are 1 to 5 allocations each, so a
    count pin cannot catch either growing.
- **Cost:** the costliest real paths can regress without a gate noticing, and the gains below cannot
  be locked in.
- **Proposed change:**
  - Add a beat pin: `Router.Update(msg)` followed by `View()`, for `tickMsg` (idle fixture) and
    `vizTickMsg` (a radio-playing fixture like `radioDash` in the scratch file).
  - Pin both **allocs and bytes**, using `testing.Benchmark`'s `AllocedBytesPerOp`, at ×1.05 as the
    house rule says.
  - Re-pin the existing frame budgets down after each fix lands, so the headroom is not left for the
    next regression to spend.
- **Behaviour:** none. Tests only.
- **Regression risk:** none.
- **Size:** S.
- **Confidence:** high.

### PF-7: The masthead is rebuilt from scratch on every frame although it changes about once a minute
- **Where:** `modes/tty/body.go:28-37` (`header`, `headerRow`, `apiSummaryOf`) and
  `modes/tty/masthead.go:49` (`mastheadStamp`)
- **Observed:**
  - `BenchmarkPF_Header_133x44`: **86 allocs and 11.4 KB per frame**, about 23 % of the hit frame.
  - It is built from:
    - the version, `o` (width, ascii, clock format) and the theme generation;
    - the snapshot (the API counts, `dataAsOf`);
    - `agoWords(now - at)` (minute buckets) and the 5-minute stale flag.
  - Nothing else in it moves.
  - `headerRow` builds its control forms one at a time and measures each.
- **Cost:** about 86 allocs on all of the roughly 3.3 frames a second, and on every one of the 20 Hz
  visualiser frames.
- **Proposed change (M):** a one-entry memo beside `bodyMemo`, keyed on version, `o` with `Frame`
  zeroed, the snapshot pointer, the age words, the stale flag and `render.ThemeGeneration()`. Cover
  it with the existing reflection completeness guard (`memo_completeness_test.go`). Record it in the
  cache table (quality-baseline §4) with its bound of one entry.
- **What changes on screen:** nothing, provided the key is complete. The guard is what makes that
  checkable.
- **Regression risk:** the golden frame tests in `modes/tty` that draw the masthead.
  This is UNPINNED as a memo until the completeness guard covers it.
- **Size:** M.
- **Confidence:** medium. Measured the cost; the memo was not prototyped.

### PF-8: The ticker tape is rebuilt per frame as well as per tick, and one memo would serve both (F-32, cheaper than it looks)
- **Where:** `modes/tty/ticker.go:153-184` (`tickerMarquee` → `tickerLane` → `tapeLine`) and
  `:294-303` (`tickerLoopLen`, which is F-32)
- **Observed:**
  - The render side builds every tape line again on every frame: **64 allocs and about 3.7 KB per
    frame**, two `time.Format` calls per item.
  - F-32's tick side spends 68 allocs per tick on the same tape.
  - F-32 deferred this because "a cached length needs invalidation over items, width, units and
    ascii". The code shows a narrower dependency:
    - `tapeLine` (`ticker.go:90-110`) reads the item, `o.ASCII`, `o.Clock` and `d.clock()`.
    - `render.Clock.Since` (`platform/render/clock.go:124`) reads `now` only to compare
      **dates**.
    - So the tape depends on the item slice, the lane, ascii, the clock format and the local date.
      It does **not** depend on width or units.
- **Cost:** about 64 + 68 allocs every 300 ms, around the clock, and 64 more on every 20 Hz
  visualiser frame.
- **Proposed change (M):**
  - One tape memo keyed on a ticker generation (bumped in `setTicker`), the lane, ascii, the clock
    format and the local date.
  - Both `tickerMarquee` and `tickerLoopLen` read it.
  - The parked-offset behaviour (F-16) is untouched, because the clamp stays where it is.
- **Disposition:** F-32 is a deferred follow-up. This is offered as "cheap now if the lead wants
  it", not as a new defect.
- **What changes on screen:** nothing, if the key is complete. Note that
  `TestTickAdvanceAllocBudgetWithATestEventOnTheTape` mutates `d.ticker[0]` in place, which a
  generation key would not see. That test would bump the generation or use `setTicker`.
- **Regression risk:**
  - `TestTickAdvanceAllocBudget` and the ticker rotation tests in `modes/tty` (`ticker_test.go`).
  - The midnight rollover of the `Since` form is UNPINNED as far as I found.
- **Size:** M.
- **Confidence:** medium. Costs measured; the memo not prototyped.

### PF-9: Every publish redraws both tables, even when nothing they show has changed
- **Where:**
  - `platform/sched/sched.go:260` (`s.publish()` after every provider of every tier cycle)
  - `modes/tty/memo.go:42` (`bodyKey.snap, recent *snapshot.Snapshot`, keyed by pointer)
  - `app/pipelines.go:127-128` (alerts every 20 s, obs every 90 s)
- **Observed:**
  - Each tier cycle publishes once per provider, whether or not the answer changed.
  - Each publish is a new `*Snapshot`, so the body memo misses, and so does the modal memo, which
    keys on the same pointers.
  - A miss at 133x44 is **5,676 allocs and 400 KB** (`BenchmarkFrame_133x44_Miss`).
  - The priority alerts tier alone publishes about 180 times an hour.
- **Cost (estimated from the cadence table, not measured live):** about 300 priority publishes an
  hour, which comes to roughly 2 MB/min of table re-renders and 1.7M allocations an hour on an
  unchanged screen.
- **Proposed change (M–L):**
  - The Assembler keeps a **content generation** that `Apply` bumps only when a location's drawn
    data changes.
  - The table memo keys on it instead of the pointer.
  - **Not** suppressing the publish: the masthead's `Updated:` stamp reads the providers'
    `FetchedAt`, so a no-change publish still changes the frame, and accepted cost §7 rules the
    publish cadence.
- **What changes on screen:** nothing, if the generation is complete. Its completeness needs the same
  reflection guard as the memo keys.
- **Regression risk:**
  - the memo completeness guard and memo tests in `modes/tty` (`memo_completeness_test.go`,
    `memo_test.go`);
  - the assembler tests in `platform/snapshot`;
  - issue #11 / #12 style "stale row" cases, which are the danger.
- **Size:** L.
- **Confidence:** medium. Mechanism read from the code; the live rate is estimated.

### PF-10: A snapshot copies each hourly forecast up to three times per publish
- **Where:** `platform/snapshot/assembler.go:477` and `platform/snapshot/harmonize.go:100`
- **Observed:**
  - `BenchmarkAssembler25Locations`: **998 KB per publish**.
  - Line profile:
    - `assembler.go:477` `cp.Hourly = append([]Hourly(nil), sec.Hourly...)` is 62 %.
    - `harmonize.go:100` `loc.Hourly = append([]Hourly(nil), sec.Hourly...)` is 31 %.
  - The second copies a slice the first just made for this same immutable snapshot.
  - The first is a shallow copy anyway: `Hourly` holds `*float64` fields that stay shared with the
    assembler.
  - The assembler never writes into an existing `sec.Hourly`; it replaces it with a fresh slice
    (`assembler.go:265`).
- **Cost:**
  - The priority pipeline publishes at least every 20 s (alerts tier).
  - At about 40 KB per location in this fixture, that is about 1 MB/min of copies.
  - Each copy is taken under the assembler lock.
- **Proposed change:**
  - **(S, behaviour-preserving)** `harmonize` shares the provider copy it just received
    (`loc.Hourly = sec.Hourly`). Both live in the same published value, and no consumer writes
    `Hourly` (grep: no writes outside `platform/snapshot`). Expect −31 % of the bytes.
  - **(M)** Share `sec.Hourly` into the snapshot as well, under a pinned rule "Apply replaces,
    never writes into" (a test in the style of the existing
    `published tide events must not alias assembler state`). Expect up to −93 % of the bytes.
  - Do **not** extend either to `Daily`: `harmonize` fills daily fields in place (`FillFrom`).
- **Regression risk:**
  - the assembler alias tests (`assembler_test.go:43`, `snapshot_test.go:341`);
  - the snapshot JSON goldens in `platform/snapshot/testdata`. Output is unchanged.
- **Size:** S (harmonize); M (assembler).
- **Confidence:** high for the measurement; medium for the M part's safety.

### PF-11: On the console, an Observer window works out the whole Observer layout twice per frame to read one field
- **Where:**
  - `modes/tty/cursor.go:340` (`OverlayCursor`: `o := d.layout().o`)
  - `modes/tty/debug.go:286` (`OverlayWindow`: `o := d.layout().o`)
  - both called from `Router.View` (`router.go:528-530`)
- **Observed:**
  - Each `layout()` builds the radio player twice and the control row once, about **136 allocs**
    (profile: `Dashboard.layout`), and the frame keeps only `fl.o`.
  - This runs on every console frame while a card, request, Settings or ctrl+d window is open.
- **Cost:** about 270 allocs per frame on top of the composite while the operator works in a window.
- **Proposed change:** `Router.View` computes `o` once and passes it to both. Or add an unexported
  `opts`-only path. PF-3 step 1 halves each call meanwhile.
- **Behaviour-preserving by construction.**
- **Regression risk:** the console-window cursor tests (D-362) in `modes/tty`.
- **Size:** S.
- **Confidence:** high on the mechanism. The 136 is the observer fixture's figure, not measured on
  the console.

---

## Minor

- `app/seen_store.go:122-133`: the seen store is marshalled and rewritten to disk every two minutes
  whether or not it changed, holding `s.mu` across the write. A dirty flag would skip unchanged
  writes. The file bytes would be the same.
- `modes/tty/dashboard.go` `Dashboard.cfg` is a 1,320-byte `Config` held by value, which is 20 % of
  every model copy and box in PF-1. A pointer would shrink every box, but the `With*` builders
  assign into it, so aliasing needs care. Lower value once PF-1 is done.
- `modes/tty/radio_panel.go:108-130`: `radioLines` builds title, state label and controls
  (`radioControlLines`, about 24 allocs) in each of its two calls. This is subsumed by PF-3.
- `modes/tty/memo.go` `mapModalKey`: `strings.Join` of the notes and the radar timeline, plus
  `tempMemoKey()`, on every map frame. Small next to the composite.
- `modes/tty/memo.go` `statsFingerprint`: `fmt.Sprintf("%+v")` plus sha256 of the whole `Stats` on
  every frame while [S] is open. This is F-4 territory; brief exposure.
- `modes/tty/ticker.go:300`: `tickerLoopLen` measures with `tickerBulletDot` while the frame uses
  `tickerBullet(o)`. Both are the same width today (comment at `ticker.go:111`), so this is not a
  bug, but a junior changing one bullet would desynchronise the loop length. It goes away with PF-8.
- `domains/radio/player/engine.go:603`: a 50 ms poll while playing, which calls `title()` each
  tick. Fine given the audio work beside it.

## What is in good shape (don't spend time here)

- **`platform/httpx`:** singleflight per URL, a fresh-cache re-check inside the flight, a negative
  cache for 4xx only, validators with 304 renewal, and locks released before I/O.
- **`platform/history`:** every lock is released before disk work (`Retain`, `Clear`,
  `ForgetDataset`, `prunePass`). `BenchmarkValuesDecode` takes 2 allocs for 4,096 values.
- **Parse memos:**
  - HMS keeps its 70 ms / 33 MB parse behind `bodymemo` plus an `agememo` coalescer, so it is paid
    once per archive.
  - The FIRMS and USGS memo hits take 0 allocs.
  - `spectrum.Bands` takes 13 µs and 80 B per visualiser beat.
- **The body and modal memos and their reflection completeness guard.** The idea is sound; the
  findings above are about what sits outside them.
- **The overlay compositor's share** (about 80 % of any window frame) is accepted and ruled
  (accepted costs §1, F-220). Nothing here reopens it.

## What I did not cover

- **CPU time.** The machine was shared and the profiles were dominated by scheduler noise, so every
  claim here is allocations and bytes. The CPU saving is inferred from bytes and the collector, and
  was not measured.
- **The live app and network behaviour**, including the real publish rate (PF-9 is estimated from
  the cadence table) and the real hourly series sizes (PF-10 uses the bench fixture's 168 hours per
  provider).
- **The map pane's internals beyond the profile.** F-220 covers them: `Lattice.interpolate` memory
  and go-ionomaps refresh (F-222). Also not covered: Propagation answers (`platform/propanswers`
  takes 19 ms / 3 MB per chart, asynchronous), radar decoding (F-183), Piper and audio decoding,
  `tools/`, and Windows/Linux differences.
- **The domain parsers other than HMS** were not profiled.
- **The prototypes are measurement instruments, not proposed code.** Each needs its old-vs-new
  equivalence test to land with it. The `TestDeclarationSetUnchanged` guards will need their
  expected sets updated wherever a helper is added.
