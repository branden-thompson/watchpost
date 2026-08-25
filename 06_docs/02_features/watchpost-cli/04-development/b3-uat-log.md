# B3 UAT Log (D-21 sessions)

## Session 1 — 2026-08-24 · setup wizard (standalone) · HUM LEAD

| # | Finding (verbatim gist) | Disposition |
|---|---|---|
| UAT-1A | "Box doesn't line up correctly… bubbletea window/modal component? I'll provide a new mock — the M-V6 mock was intended as a modal/dialog that floats on top of the dashboard." | **Fixed (interim)**: wizard box now renders via render.Panel (the hand-rolled border was the bug; D-9 seam owns boxes — lipgloss borders are the underlying charm-stack primitive, bubbletea ships no dialog component). **Pending new mock**: setup-as-modal over the dashboard queued behind the incoming standalone-view mock (M-V6b); floating-modal composition ships with the dashboard modal work (ctrl+a/help). |
| UAT-1B | "Give users instructions on where to get their FIRMS key — CLIAmp does this with links/directions as helper text." | **Fixed**: Q3 helper now reads "Get a free key: https://firms.modaps.eosdis.nasa.gov/api/map_key (sign up with an email; NASA mails you a MAP_KEY — paste it below, or press enter to skip)". CLIAmp-style helper-text pattern adopted as a standing wizard convention. |

| UAT-2A | "Table is incorrect and doesn't seem to be using the go-studs component. Notice my mock has no table lines." + ruling: "the mocks I provided is the output that I want exactly, then I'll provide color instruction." | **Fixed**: table rebuilt as borderless absolute-offset transcription of the mock (group header + column header VERBATIM from the mock file, fidelity-test-diffed against it; marks ›/▶/⚠ at cols 0/2/4; LABEL column added — config.Location.Tag, setup auto-derives 5-char tag; NOW trend arrows ↗/↘ from next-hour delta). It WAS the go-studs AdvancedDataTable — its markdown pipes + byte-padded cells (ºF misalignment) are filed as upstream candidates (M6); table lives in the render seam until go-studs grows a borderless display-width mode. **Standing rule recorded (feedback-mock-fidelity): mocks are the exact spec; color/styling is a separate HUM-LEAD-directed pass.** |

Dashboard walk continues (next UAT pass on the corrected table).

## Session 2 — 2026-08-24 · dashboard vs mock rev2 (125-col) · HUM LEAD

Branden repasted the dashboard mock **optimized for 125 columns** with notes — saved verbatim to
`09-view-mocks/dashboard-mock-rev2-125col.txt` (now a fidelity source alongside the original).
He flagged "…and there's more" — session stays OPEN for further findings + the M-V6b setup mock.

| # | Finding (verbatim gist) | Disposition |
|---|---|---|
| UAT-3.1 | "Add 3 blank lines above the top line to give it some space" | **Fixed**: View() opens with 3 blank lines (test: `TestViewOpensWithThreeBlankLines`). |
| UAT-3.2 | "Blank line missing between title and Alert Section" | **Fixed**: blank line after the header block (same test asserts it). |
| UAT-3.3 | "Radio Mock is not there, we should have it there to test the mock" | **Fixed (static)**: radio player frame renders per rev2 mock — station line ("♪ … radio lands in B4"), VOL bar `███░░ 55` (0–100 per D-19), `■ STOPPED`, visualizer area, `[p]in Selected Location` / `[T] Toggle Player Size`. Audio/behavior lands in B4; this is a layout-test placeholder as requested. |
| — | Rev2 deltas absorbed in the same pass | Alert module: wrapped body + `NN / NN Alerts` pager + `[A] Alert Details   [←] Previous  [→] Next` (left/right bound + tested). PRIORITY header gains `NN/10 Used`. RECENT/SEARCHED section frame + shared table headers + "Showing 0-0 of 0 locations ▼" (persistence/scrolling later in B3). Two-line footer verbatim from rev2 with D-19 key subs (ctrl+a). Color notes logged for the color pass: **High temp = Orange, Low temp = Cyan**. |

Engineering fallout from this pass (caught by gates, not UAT): M1 `firstFull` timer had a data race
under concurrent tier publishes → atomic CAS; `RunDashboard`/`handleKey` re-split for P10-04.

Still queued in B3: recent/searched persistence + scrollbar (▲│█▼), tab section navigation,
enter→detail view, ctrl+a modal, `A` alert-details view, `t` theme chooser, ultra-wide >125col
columns, color pass (HUM LEAD directs).

### Session 2, round 2 (A–E + component ruling)

| # | Finding (verbatim gist) | Disposition |
|---|---|---|
| UAT-2A′ | "Prepopulate the tables with 25 major US cities — right now it's hard to tell how the table looks." | **Done**: RECENT/SEARCHED seeds with the top-25 US cities by population from the embedded geodata (`geodata.TopUS` + `locations.Seeds`, offline, deterministic; NYC's zip needed the centroid-scan fallback — place-name drift "New York City" vs "New York"). Names/zips render instantly (assembler publishes refs pre-data); live weather streams from a **second slow-cadence scheduler** (alerts 2m / obs 10m / forecast 1h, start delayed 5s) so the priority pipeline's M1 budget is untouched. Numbering continues after the priority rows (mock: 004.); windowed to 10 rows with the ▲│▼ rail; priority-configured zips are deduped out. Seed list is the placeholder until real search history lands (persistence still queued). |
| UAT-2B | "Buttons/key bindings — CLIAmp pattern: grey background, bold white text" (screenshot) | **Done**: `render.KeyCap` chips (SGR bold + bright-white on grey 237 via go-studs `rendering.WrapSGR`) applied to footer, header `[a]`, alert controls, radio `[p]`/`[T]`. Color-off contexts (NO_COLOR, pipes, tests) degrade to the mock's `[key]` brackets so the affordance never disappears (RS-14). **Upstream candidate (M6): `tui.KeyCap` semantic token / keycap primitive in go-studs.** |
| UAT-2C | "4-col padding all around the viewport like CLIAmp" | **Done**: 4-col left/right viewport padding on every line; top spacing stays the 3 blank lines ruled in UAT-3.1 (call out if you want 4). |
| UAT-2D | "Terminal-width aware: EXTENDED columns >125; narrower drops TOMORROW → LABEL → TODAY HI/LOW" | **Done**: breakpoints at content width 124 (full) / 93 (−TOMORROW) / 84 (−LABEL) / 68 (−TODAY HI/LOW); EXTENDED FORECAST day columns join progressively >125 per the original mock (first cell col 128, pitch 18 — the mock's own 17/18 wobble standardized to 18, group bracket closes col 213 with 5 days, `H hi/lo L` cells, mm/dd headers). Test-pinned per breakpoint. |
| UAT-2E | "Alert Section and Radio Panel also smartly auto-resize" | **Done**: both panels track the padded content width (no more 125 clamp); ANSI-aware `PadBetween` keeps right-anchored elements (VOL bar, controls) on the edge at any width. |
### Session 3 (post-A–E rendered-output review)

| # | Finding (verbatim gist) | Disposition |
|---|---|---|
| UAT-3.1′ | "Both location tables should match — if extended cols display in priority, recent must immediately match." | **Fixed**: one `sharedExtDays()` (max across both snapshots) pins the layout for priority, recent, and the empty recent frame; `LocationTable` now takes the shared day count. |
| UAT-3.2′ | "Columns are not perfectly aligned (see output)." | **Fixed** — root cause: go-studs returns over-wide non-truncatable cells as-is, shifting the row (`PARTLY_CLOUDY` = 13 cells in the 12-cell CONDITIONS column; 3-digit-temp rows). Seam now (a) maps conditions to the mock vocabulary (`P.CLOUDY`/`M.CLOUDY`, underscores→spaces) and (b) hard-clamps every cell to its column width. Regression test pins NOW at col 69 under pathological overflow. |
| UAT-3.3′ | "Temperature colors: HIs orange, LOs cyan." | **Done**: TODAY/TOMORROW HI/LOW columns via `CellStyles` (256-palette 208 / 51, applied after padding so geometry is untouched); EXTENDED cells color hi/lo inline (component padding is ANSI-aware). |
| UAT-3.4′ | "Chip colors wrong — bkg light grey, text BOLD white (like CLIAmp)." | **Fixed**: background 237→245 (light grey), text `1;97` bold bright-white. |
| UAT-3.6′ | "Column groups follow chip style: LOCATION/STATION grey, TODAY light blue @50%, TOMORROW light cyan @50%, EXTENDED light purple @50% — white text; `[ ]` are the bkg edges." | **Done**: group spans render as chips (256-palette 245/67/73/103 — terminals have no alpha, nearest ~50% pastels), brackets swallowed as the chip edges; color-off keeps the bracketed text (RS-14). Forced-color test pins all four backgrounds. |
| ENG | — | go-studs `SGR()/ColorSequence` classifies bare codes as 256 **foregrounds** only — background chips are outside its contract and `StyleDirect` routes back through the same classifier. Seam emits its one raw SGR shape, gated by the go-studs color switch. **Upstream candidates (M6): background support in ColorSequence; `tui.KeyCap` token.** |

### Session 4 (color pass round 2 + focus model)

| # | Finding (gist) | Disposition |
|---|---|---|
| 4.1 | "Extended forecast in recent not showing up immediately" | **Fixed**: the 25-city pipeline now runs as 5 chunked schedulers sharing ONE assembler — each chunk publishes the shared snapshot the moment its own fetches land, so rows (extended included) fill progressively instead of waiting on a single 25-city sweep; start delay cut 5s→1s. |
| 4.2 | "Fixed widths for max 3-digit temps" | **Fixed**: EXTENDED cells use fixed 5-cell slots (hi right-just / lo left-just, cell width 15) — the slash never staggers between 2- and 3-digit rows. Main temp columns were already fixed 5-cell fields. |
| 4.3 | "Group chips 1 col short of the border" | **Fixed**: LOCATION/TODAY/TOMORROW chips grow one col (data extends one past the mock bracket); EXTENDED already flush. |
| 4.4 | "Up/down focus not working; › should move and scroll across the two tables" | **Fixed**: one focus index spans both tables; › renders on the focused row wherever it is; the recent window auto-scrolls (Showing N-M tracks); the alert module follows the focused location. Test-pinned. |
| 4.5 | "n/a should be default grey" | **Fixed**: nil temps render in the base grey (250) everywhere, including inside extended cells. |
| 4.6 | "Alert area yellow or red by statement type" | **Done**: `AlertTone` — severe/extreme or *Warning* events → red border+title (196); watch/advisory-grade → yellow (220). |
| 4.7 | "Focused row name Bold + Yellow (pre-theming)" | **Done**: focused row's NAME styled `1;220`. |
| 4.8 | "NWS green when good" | **Done**: provider glyph green (77) when OK, red (196) otherwise. (First pick 256:40 leaked through go-studs ColorSequence as SGR bg-black — classifier only maps 2-digit 48-89; noted for M6.) |
| 4.9 | "WATCHPOST bold + the reference CLI gradient" | **Done**: per-rune bold truecolor interpolation #DD51D6→#378FE9→#7CE3B3, transcribed from the reference CLI `createGradientText`; plain text when color is off. Theme-dependent gradient comes with theming. |
| 4.10 | "Default text slightly darker grey, a11y-conforming" | **Done**: `TintDefault` opens the frame in grey 250 (~9:1 on black — AA/AAA) and re-arms it after every SGR reset, so explicit colors win and everything else reads base grey. |

### Session 5 (module blocks + fill latency)

| # | Finding (gist) | Disposition |
|---|---|---|
| 5.1 | "Extended forecast still slow - should be near instantaneous" | **Fixed**: chunk size 5 -> 1 — every seed location runs its own scheduler against the ONE shared assembler, so each row publishes the moment its own fetches return (~1-2s to first rows, all 25 in parallel). One-time launch burst ~75 requests; steady cadence unchanged. True warm-launch instant display = the queued "NWS cache refresh" B3 item (disk cache), flagged for that pass. |
| 5.2 | "Appearing/disappearing alerts jarring - reserve the space" | **Fixed**: alert area is a fixed 5-line module; blank-but-reserved when the focused location has no alert (body truncates to one line - full text belongs to the Alert Details view). Height equality test-pinned. |
| 5.3 | "Inner text should match the location" | **Done**: title line now reads `⚠ EVENT · Location Label` for the focused location. (Interpreted as: name the location in the module - say the word if you meant something else.) |
| 5.4 | "Remove borders; 10% background tints (advisory yellow/yellow, alerts red/red, radio grey)" | **Done**: `render.Block` - borderless full-width paint, truecolor ~10% tints (256 palette has no tints that dark): warnings `#280000` bg + red 196 text, advisories `#201c00` bg + yellow 220 text, radio `#1e1e1e` bg + base grey text. Inner SGR resets (chips) re-arm the block tone so the background never tears; blocks close clean; color-off passes text through (R-12a). Dark-terminal variants — the light-bg text variants (dark brown / black) land with theming + background detection. |

### Session 6 (spacing, tints, rail, footer wrap, fill perf)

| # | Finding (gist) | Disposition |
|---|---|---|
| 6.1/6.2 | "Blank line above+below alert area; below radio" | **Done**: blank above the alert area was already present (UAT-3.2); blanks added below the alert area and below the radio module. |
| 6.3 | "Extended data still trickling - deeper performance pass" | **Root cause found**: httpx token bucket defaults to 5 req/s with strict pacing - the ~75-call launch burst drained over 15s+. Dashboard client now runs RatePerSec 30 (~2.5s drain, still polite; steady-state volume near zero). Warm-launch disk cache remains the queued deeper fix. |
| 6.4 | "Red bkg to 5%" | **Done**: warning bg #280000 -> #140000. |
| 6.5 | "Radio grey +~20%" | **Done**: #1e1e1e -> #404040 (~25% grey) - flag if too strong/weak. |
| 6.6 | "Showing line misaligned 1 col left" | **Root cause**: PadBetween's minimum-1 gap pushed the rail glyph one col RIGHT on rows that filled the full row length (15-cell ext cells), while ▲/▼ sat at the true rail column. Rail now pinned via exported PadTo - one column for ▲│▼ at every row width. |
| 6.7 | "Footer help width-aware, smartly wrap" | **Done**: render.WrapSegments greedily packs the chip+label segments to the terminal width (ANSI-aware); narrow terminals get more lines, ultra-wide gets one. |

### Session 7 (module geometry + radio styling)

| # | Finding (gist) | Disposition |
|---|---|---|
| 7.1 | "Bg-padded blank line top+bottom of alert/radio; shrink both to align with the tables (+6 left / +4 right)" | **Done**: blocks render through `blockOpts` (width-10, indented 6) with leading/trailing bg-tinted padding lines; alert area fixed height 5→7 to keep the no-alert reservation identical. Alignment + 9-line block test-pinned. |
| 7.2a-c,e | Radio colors: title green, VOL fill green, timestamp white, `■ STOPPED` bold grey (bold green when playing/paused) | **Done** via new seam `render.Tint` + exported codes (Green/White/BoldGrey/BoldGreen). BoldGreen is wired for B4's playing/paused states. |
| 7.2d | "We'll need to build some visualizers like CLIAmp" | **Queued for B4** (radio milestone): CLIAmp-style visualizer(s) in the reserved area; placeholder stays until audio lands. |

### Session 8 (order, height-aware window, floating help, layout snap)

| # | Finding (gist) | Disposition |
|---|---|---|
| 8.1 | "Switch the order of radio and alerts" | **Done**: radio module renders above the alert area. |
| 8.2 | "Vertical scroll cramped when extended columns show" | **Fixed**: the recent window is height-aware — it shrinks from 10 rows (floor 3) on short terminals so the Showing line and footer stay on screen; focus-follow scrolling uses the same dynamic size. |
| 8.3 | "'?' should float over the dashboard — Charm supports this" | **Done**: lipgloss v2 Canvas/Layer compositing via `render.Overlay` — help panel (compact, 56 cols) floats centered over the live dashboard; esc still closes. v2.0.2 gotcha: `Layer.Draw` ignores X/Y and Layer.Width/Height are unset — positioning must go through the `Compositor` (probe-test caught the blank base). |
| 8.4 | "Sections misalign as columns expand/shrink" | **Fixed**: the WHOLE layout (header, modules, footer) snaps to the table's discrete row length (`opts()` sets Width = TableRowLen) — at a breakpoint every section moves together; between breakpoints nothing drifts. Ultra-wide caps at the 5-day table span. |

### Session 9 (block width, rail gap, group tint dim)

| # | Finding (gist) | Disposition |
|---|---|---|
| 9.1 | "Extend radio+alert 4 cols right - flush with the table edge" | **Done**: blockOpts inset is now left-only (+6); blocks end at the table's right edge. |
| 9.2 | "Rail cramped against the extended column - 1 col space after the last L" | **Done**: rail column moved out one (+2 past row length); Showing line tracks it. |
| 9.3 | "Group label bkg opacity -30% (all)" | **Done**: group chips dimmed to truecolor 70% of the session-3 palette (grey #616161, blue #425e7a, cyan #427a7a, purple #5e5e7a); keycap chips intentionally untouched. |

### Session 10 (window chrome + forecast details)

| # | Finding (gist) | Disposition |
|---|---|---|
| 10.1 | "Help text bright white -> grey" | **Done**: floating modals render through TintDefault (base grey 250). |
| 10.2 | "Window bkg blue-ish grey: #1D2830 dark / #3B5163 light" | **Done**: `View.BackgroundColor` set per terminal mode - Init requests the terminal background (bubbletea `RequestBackgroundColor`), `BackgroundColorMsg.IsDark()` picks the variant (dark default). |
| 10.3 | "3-col top padding -> 2" | **Done**: 2 blank lines above the header; height budget updated. |
| 10.4 | "About window needs a visible vertical scroll control on short screens; expand to fit on tall" | **Done** on the floating-modal component (`render.ScrollPanel`): body windows to the height budget with a ▲│█▼ rail (thumb tracks position) on short terminals, expands rail-less when it fits; ↑↓ scroll the open modal (not the tables). Help uses it now; About inherits it when the `a` view lands. |
| 10.5 | "Non-rounded corners" | **Done**: panel corners ╭╮╰╯ -> ┌┐└┘. |
| 10.6 | "Forecast details as a floating window on enter" | **Done**: `enter` floats FORECAST · {Label} ({zip}) over the dashboard - NOW line (temp/trend/condition, FEELS, HUM), TODAY/TMRW + up to 8 daily rows (HI orange / LO cyan), active-alert count with [A] hint, esc closes, scrolls via the same rail on short terminals. First cut of M-V2 as a floating window - content set awaits your direction. |

### Session 11 (fill column, LABEL hidden, rail thumb)

| # | Finding (gist) | Disposition |
|---|---|---|
| 11.1 | "Tables don't fill the width - NAME should use go-studs fill; 2-col global padding" | **Done**: NAME is now the go-studs Fill column (MinWidth 24, Truncatable to 15) - both tables span the full content width at every terminal size, longer names welcome; global padding trimmed 4->2 cols (rail keeps a 2-col gutter on the right). This replaces the session-8 discrete "snap" - edges are now always flush. Group headers derive from RESOLVED offsets (the fill moves every downstream column); offsets pinned by tests at min-full and widened widths. |
| 11.2 | "Hide LABEL - ZIP identifies" | **Done**: LABEL column hidden (data kept in the row model + config); breakpoints recomputed (full >=115, -TOMORROW >=84, then -TODAY HI/LOW). Departure from the mock is HUM-LEAD-directed; mock constants remain as fidelity anchors in tests. |
| 11.3 | "Recent rail missing its square thumb" | **Done**: █ thumb on the rail tracks the scroll position between ▲ and ▼ (same math as the modal rail). |

### Session 12 (group span, chrome trims, bg split)

| # | Finding (gist) | Disposition |
|---|---|---|
| 12.1 | "LOCATION/STATION group stretches over the prefix cols" | **Done**: group chip spans from col 0 (pointer/radio/alert marks) to ZIP. |
| 12.2 | "Remove the PRIORITY LOCATIONS NN/10 Used row" | **Done**: row removed (cap enforcement stays in the data layer); height budget updated. |
| 12.3 | "Align radio/alert to the new left edge" | **Done**: module blocks flush with the table on BOTH edges now (indent removed); `blockOpts` kept as the single place module insets live. |
| 12.4 | "Grey bkg was for the help tile only; base bkg darker ~#131313" | **Done**: base viewport #131313 (light-mode placeholder #ECECEC — flag at theming); the blue-grey pair (#1D2830/#3B5163) now paints the floating modal tiles (help + forecast details). |
| ENG | "Note what can be modularized; no repeated code; extract helpers; write for refactor/reuse" | **Recorded as standing memory** (feedback-modularity-standards) and applied: help + details modals collapse into ONE `floatModal` helper (scroll panel + tile tone + tint); module insets live only in `blockOpts`; tones/widths remain single-owner constants in the render seam. |

Next from HUM LEAD: mock for the location details view.

### Session 13 (modal color bug)

| # | Finding | Disposition |
|---|---|---|
| 13.1 | Modal color bug (screenshot): grey bands trailing every chip/tinted span inside the forecast/help tiles | **Fixed** - root cause: `floatModal` ran TintDefault BEFORE Block, consuming the SGR resets Block re-arms on; after any styled span the tile background fell through to the terminal default (bg 49) for the rest of the line. Block alone owns both the base-fg and tile-bg re-arm now. Regression test pins the re-arm sequence; probe confirmed the lipgloss/uv compositor was faithful (not the culprit). |

### Session 14 (alert/trend tones, left pad, group banding refactor)

| # | Finding (gist) | Disposition |
|---|---|---|
| 14.1 | "Alerted locations: muted yellow #D0CF89 advisories, muted red #D08989 warnings" | **Done**: location NAME tints by grade via the new single-owner `AlertIsWarning` classifier (module tones ride it too); focus styling outranks the alert tint. NOTE: CellStyles route through go-studs ColorSequence which mangles truecolor — nearest-256 used (186 / 174); M6 gap already filed. |
| 14.2 | "Trend arrows muted orange #A98D40 up / muted cyan #409FA9 down, all views" | **Done**: `render.TrendGlyph` is THE arrow renderer (tables + forecast modal + future views); nearest-256 (137 / 73). |
| 14.3 | "Left global padding back to 3 (good reversion)" | **Done**: viewPadLeft 3 / viewPadRight 2. |
| 14.4 | "Extend group labels to touch mid-gutter; refactor for modularity if needed" | **Refactored as asked**: group banding is now declarative — `groupSpec` (title/tone/member columns) + `tableGeom` (single owner of resolved offsets+widths incl. the fill column). Bands extend into gutter halves so neighbors touch (continuous strip; columns keep spacing); color-off falls back to the bracketed form. Future tables declare their own groupSpec list. `styleGroups` (parse-and-restyle) and the UAT-4.3 expand hack are deleted. |

### Session 15 (band gap + alert wrap)

| # | Finding (gist) | Disposition |
|---|---|---|
| 15.1 | "TOMORROW and EXTENDED not touching" | **Fixed**: the gap there is gutter+spacer (4 cols), not a plain 2-col gutter — bands now stretch to the MIDPOINT of whatever separates them (general rule, any future column arrangement included). |
| 15.2 | "Alert/advisory text wraps (not truncates) on narrow widths" | **Done**: body word-wraps via new single-owner `render.WrapText`. |
| 15.2a | "Give the modal 3 lines of text space so the section doesn't bounce" | **Done**: fixed 3-line body area (module height constant at 9 lines whether the body needs 1, 2, or 3); >3-line overflow belongs to the Alert Details view. Height-parity + wrap tests pinned. |

### Session 16 (ext label, alert order, ctrl+a, SEMANTIC TOKENS)

| # | Finding (gist) | Disposition |
|---|---|---|
| 16.1 | "Narrow EXTENDED band truncates to EXTENDEDFORECA…" | **Fixed**: groupSpec gains a `short` fallback title — the band degrades full → "E X T E N D E D" → unspread → truncate. LOCATION/STATION gets "L O C A T I O N" as its short form too. |
| 16.2 | "Most SEVERE alert always shows first (LA case: red name, advisory shown)" | **Fixed**: alerts sort most-severe-first on snapshot receipt (extreme>severe>moderate>minor; warnings outrank advisories within a tier) — the module's first page, the name tint, and the details view all agree. Test pins the LA case. |
| 16.3 | "ctrl+a doesn't show the add-location modal" | **Wired**: ctrl+a floats the Add Location modal — typing builds the query (global keys are swallowed while typing; 'q' can't quit mid-word), backspace edits, esc cancels. Live type-ahead results need a search hook from app (modes cannot import domains — import lint); that lands with the M-V3 flow + your mock. |
| 16.4 | "ALL colors as SEMANTIC TOKENS before more colors arrive" | **Done**: new `platform/render/theme.go` — a Token vocabulary (36 roles: text/temps/trends/names/providers/chips/groups/alert tiles/radio/modal/window/gradient stops) resolving through one theme table. Every emission site now calls `Tok(role)`; raw codes are gone from render.go and the tty views; the gradient interpolates from theme hex stops. Theming (the 't' chooser) swaps the table — zero call-site changes. Token-completeness test guards against unpainted roles. Theme table's package-level var exempted with recorded reason (it IS the theming surface; ratify at gate). |

### Session 17 (add-modal wrap, alert bg evaluation)

| # | Finding (gist) | Disposition |
|---|---|---|
| 17.1 | "Add-location helper text wraps, not truncates" | **Done**: helper prose wraps via render.WrapText to the modal width. |
| 17.2 | "Hide the advisory/alert bgs, keep text colors — testing the look" | **Done as a theme edit only**: AlertWarnBG/AlertAdvBG → default (49) in the token table; text tones untouched. Restoring = two theme values (kept in the comments). First payoff of the 16.4 token refactor: zero code changed. |

### Session 18 (alert inset, loading shimmer, footer centering, tone tweaks)

| # | Finding (gist) | Disposition |
|---|---|---|
| 18.1 | "Alert/advisory TEXT 4-col padding both sides" | **Done**: title/body/pager inset 4 cols each side inside the block. |
| 18.2 | "n/a while loading is misleading: (a) cache and/or (b) animated '...' shimmer" | **(b) Done**: 4-phase dot sweep ('...' '·..' '.·.' '..·') on a 300ms tick wherever data hasn't arrived; "n/a" is reserved for truly absent data after load (row-level: obs or daily still pending). New spinner style noted as a go-studs upstream candidate (M6). **(a) queued**: the disk cache remains the "NWS cache refresh" B3 item — the assembler needs a prime-from-cache API so live publishes don't regress cached rows; sized for its own pass. |
| 18.3 | "Center the bottom key help lines; keep smart wrapping" | **Done**: `render.CenterBlock` centers each wrapped footer line (ANSI-aware); wrapping unchanged. |
| 18.4 | "Chip bkg → #565656" | **Done**: KeyChip token bg → truecolor #565656 (one theme value). |
| 18.5 | "Hide radio bkg; default #303030 for re-enable" | **Done**: RadioBG token → default(49), re-enable value #303030 recorded beside it (one theme value). |

### Session 19 round 2 (global inset policy)

| # | Finding (gist) | Disposition |
|---|---|---|
| 19.1a-c | "Global inset property: visible bg → 3-col insets + top/bottom pad line; no bg → flush with the title edges, single blank between radio and advisory" | **Done as a seam policy**: `render.Module` + `BGVisible`/`ModuleInnerWidth`/`ModuleHeight` — bg visibility (a theme VALUE: anything ≠ default 49) drives the chrome. Visible: 3-col left/right inset + padded blank top/bottom. Hidden: flush content, no pad lines; the one blank between modules comes from the layout. Blank-alert reservation and the height budget both track the active theme, so re-enabling a bg is still a single theme edit and every inset/height follows automatically. Policy test-pinned. |

### Session 20 (marks column: badge + tones)

| # | Finding (gist) | Disposition |
|---|---|---|
| 20.1 | "Hide the note icon in the header, keep the column" | **Done**: marks column header blank (CLIAmp style); column untouched. |
| 20.2 | "Alert counter beside the glyph: '›  2⚠'" | **Done**: count badge in the marks cell (single slot; caps at 9), position pinned at the exact '›  2⚠' shape. |
| 20.3 | "Color the ⚠ by most-severe alert: bright yellow advisory / red warning" | **Done**: badge + glyph share the tone via the existing AlertLabel/AlertDanger tokens and the AlertIsWarning classifier (severity-first sort from 16.2 keeps everything agreeing). |
| — | "I expect our go-studs table handles the spacing" | Confirmed: all three land inside the marks column spec — zero geometry change (NAME-at-11 pinned in the test). rowMarks split from rowData (P10-04). |

### Session 21 (stateful chips)

| # | Finding (gist) | Disposition |
|---|---|---|
| 21.1 | "Mute chips to ~50% when the press would do nothing; wire into the chip component for all future controls" | **Done as a component behavior**: `render.KeyCapIf(key, enabled)` — the single chip entry point for stateful controls; views pass model-derived flags (ELM: state in, view out). Muted tone is a theme token (KeyChipMuted: half-tone text on #2b2b2b). Alert paging wired per-direction: ← mutes at the first alert, → at the last, both at a single alert (your stated principle "pressing does nothing" applied precisely — with 2 alerts at page 1, ← is still inert so it reads muted; flag it if you want pair-level semantics instead). Color-off keeps the [key] textual affordance. Test-pinned across 1-alert and 2-alert states. |

### Session 22 ([A] alert details modal)

| # | Finding (gist) | Disposition |
|---|---|---|
| 22.1 | "[A] opens the alert modal, muted yellow/red tile; more than the 1 statement - starts, expected duration, etc." | **Done**: [A] floats `ALERTS · {Label} (N active)` on a severity-tinted tile (new AlertModalWarnBG/AlertModalAdvBG tokens - muted red/yellow; tone = worst active alert). Every active alert renders in full from the data layer (which already carried the whole CAP record since B1): class-toned event line + [severity · urgency · certainty], Starts/Ends with ~duration (Onset/Ends preferred over Effective/Expires), Area, wrapped Description, Instructions. Scrolls on short terminals; esc closes; ↑↓ shared modal scroll. `floatModalToned` added as the toned sibling of the one modal renderer. |

### Session 23 ([A] modal paging + per-alert tiles)

| # | Finding (gist) | Disposition |
|---|---|---|
| 23.1 | "←/→ enabled in the alert window for multi-alert locations - no esc round-trip" | **Done**: the [A] modal owns left/right paging; chips ride KeyCapIf per direction. |
| 23.2 | "Split views: one alert per page, bg re-tints (yellow→red→…) per focused alert" | **Done**: title `ALERT N / M · Label`; the tile tint follows the focused alert's class each page. |
| 23.3 | "Short terminals: scroll rail appears and the right inset breathes" | **Confirmed + tuned**: ScrollPanel already adds the ▲│█▼ rail when the body overflows; prose wrap width widened to -9 so text never crowds the rail. |
| ENG | — | handleModalNav split from handleNav (P10-04). Paging + per-page tile tints + muted-chip states test-pinned. |

### Session 24 (volume control placement, [S] status modal, stamp wording)

| # | Finding (gist) | Disposition |
|---|---|---|
| 24.1 | "[+/-] Adjust Radio Volume: footer → player, right of [T]" | **Done**: player control line reads `[p]in … [T] Toggle Player Size   [+/-] Adjust Radio Volume`; footer wraps without it. |
| 24.2 | "[S] Status chip right of the stamp → API diagnostics modal" | **Done**: `[S] Status` chip in the header; S floats API Status — provider health glyph + role/status/data-freshness age, both pipeline snapshot ages + location counts, active warnings from both snapshots. True request latency needs httpx instrumentation — noted in-modal and queued with the multi-provider (B5) work. |
| 24.3 | "'DATA LAST UPDATED:' → 'Last Updated: '" | **Done**. |

### Session 25 (modal wrap guarantee)

| # | Finding (gist) | Disposition |
|---|---|---|
| 25.1 | "Status modal truncates inner content - recurring; make a helper in the modal component" | **Fixed structurally**: `render.WrapLines` (indent-preserving, ANSI-aware) runs INSIDE `floatModalToned` - every floating window wraps its body to the tile width; no caller can reintroduce truncation. `modalWidth()` is now the single width source shared by the render sites and the scroll bounds (which use the wrapped count, so ↑↓ always reaches the true bottom). Test-pinned: wrap behavior + no-…-in-modal. |

### Session 26 (watchlist control row + LIVE add/remove/lookup)

First functional watchlist mutations — the TUI now drives the pipelines.

| # | Finding (gist) | Disposition |
|---|---|---|
| 26.1 | "Control row below the watchlist; move ctrl+a there" | **Done**: `[ctrl+a] Add Location  [shift+del] Remove  [l] Lookup` row under the priority table (controls live where they act); ctrl+a leaves the footer; Remove chip mutes unless a watchlist row is focused. |
| 26.2 | "[shift+del] → confirm modal (#AE7D7E); confirm moves it to top of recent" | **Done**: confirmation tile (new ConfirmBG token), [enter] Confirm / [esc] Cancel; confirm removes from the watchlist, prepends to recent (deduped, cap 25), rebuilds pipelines. |
| 26.3 | "ctrl+a modal works: [enter] Add appends to the watchlist; full list mutes + note" | **Done**: enter resolves the query (offline geodata + online geocoder fallback via app hook) and appends to the watchlist bottom; at the 10-cap the Add chip mutes (KeyCapIf) and the modal leads with the performance note; resolve errors surface in-modal. |
| 26.4 | "[l] Lookup: same modal, Lookup title/controls; opens detail report + tops recent" | **Done**: shared search modal in lookup mode; enter resolves → prepends to recent → focuses the new recent top and floats the forecast detail report. |
| ARCH | — | tty.Config gains `Resolve`/`Commit` hooks (modes cannot import domains): app adapts the resolver and `livePipelines.commit` persists the watchlist to config (tags derived) and rebuilds BOTH schedulers over the new ref sets (mutex-serialized; cap invariant). ELM flow (resolvedMsg/committedMsg) makes every flow testable with fake hooks — add/remove-confirm/remove-cancel/lookup/cap-mute all pinned. toggleModal split from handleKey (P10-04). |

### Session 27 (location detail view per mock)

Mock saved: `09-view-mocks/location-detail-mock.txt`.

| # | Item | Disposition |
|---|---|---|
| 27.1 | Detail view rebuilt per mock | **Done**: title = `{Label} {zip} ─── Updated: {stamp}` (right-aligned in the border); labeled sections with an unbroken `│` divider column (per the mid-turn note — all pipes are box-drawing verticals): CURRENTLY (condition, temp+trend, feels-like with delta, humidity), TODAY (condition + HIGH/LOW, sunrise/sunset in the LOCATION's timezone as `0645  Local Time`), FORECAST (up to 10 daily rows: mm/dd/yyyy, condition, precip %, HIGH/LOW). Footer: `[↑↓] navigate pane [esc] close pane [ctrl+a] add to watchlist` — ctrl+a in the detail view appends the VIEWED location to the watchlist (mutes when full/already watched); pairs with lookup. |
| 27.2 | "Modal lengthens/shortens with terminal height" | Already the ScrollPanel contract (10.4): body windows to the height budget with the ▲│█▼ rail; confirmed against the new longer body. |
| 27.3 | Alert text with the mock's bullet rules | **Done**: full NWS description parsed ("* " items): bullets indent 4 from the text edge, single-line bullets stack tight, multi-line bullets get one blank above and below; right gutter reserved for the rail. Rules test-pinned. |
| 27.4 | "Maritime forecast only for coastal locations" | **Queued (data first)**: NO marine data exists in the schema or providers — swell/conditions/water-temp need a marine source (NWS coastal-waters products or Open-Meteo Marine) plus coastal detection. The section renders only when marine data exists, so it lights up when the provider lands. Sized for its own pass. |
| ENG | — | WrapLines collapses interior spacing on over-wide lines — detail rows are sized INSIDE the modal wrap budget so the divider column can never tear (probe caught the TODAY row collapse). |

### Session 28 (detail-modal alignment + tones; NWS coastal-waters probe)

| # | Item | Disposition |
|---|---|---|
| 28.1/28.2 | "HIGH/LOW aligned for scannability; always account for 3-digit temps" | **Done**: `HIGH %5s / %5s LOW` fixed slots everywhere in the modal; TODAY's pair lands on the forecast rows' column (col 37 of the content). Test-pinned (`HIGH  98ºF /  98ºF LOW` vs `HIGH 108ºF / 100ºF LOW` share one column). |
| 28.3/28.4 | "Advisory text #ACAE7D / warning text #BE5454 in the modal" | **Done**: theme tokens AlertModalAdvFG/AlertModalWarnFG on the raw-SGR path (`render.TintRaw` — truecolor is beyond WrapSGR); applied to the detail view's alert blocks (title + body) and the [A] modal's record head. |
| 28.5 | "Titles bold" | **Done** (bold composite on the same tokens). |
| PROBE | "NWS coastal waters - free; track as a 2nd API in status" | **Live-probed** (Oceanside, grid SGX 52,36): the NWS gridpoint payload already carries marine fields for coastal grids — primarySwellHeight/Direction, secondarySwell*, waveHeight, wavePeriod, wavePeriod2, windWaveHeight (WMO units). Water temperature is NOT there (NDBC buoys). `/zones?type=coastal&point=` returned nothing; coastal detection = marine fields present in the gridpoint. **Plan (own pass, data first)**: snapshot `Marine` section (swell/wave/period, SI internal), NWS provider fetches the raw gridpoint on the forecast tier for coastal locations, registered as a distinct provider id (`nws-marine`) so the [S] status modal tracks it as a 2nd API row automatically; MARITIME section lights up on data; NDBC water temp as a follow-on. |

### Session 29 (coastal waters: NWS marine + NDBC buoys — data first)

| # | Item | Disposition |
|---|---|---|
| 29.1 | "Add the maritime info; see if buoy data is free" | **Both free, both wired.** Schema: `Marine` section (SI: m/s/°true/°C — swell height+dir, wave height+period, wind-wave, water temp, buoy id+distance, observed-at, provenance); `Location.marine` is null inland; new `KindMarine` fetch kind + "marine" provider domain. |
| | NWS coastal waters | `nws-marine` provider (shares the weather provider's resolve cache; own id so [S] status tracks it as a 2nd API): lifts primarySwell*, waveHeight, wavePeriod, windWaveHeight from the raw gridpoint. **Live finding**: cells one step off the coast publish the marine series as ALL ZEROS — presence is not coastal, a non-zero swell/wave height is (Oceanside's ZIP centroid hit exactly that cell in the first smoke). |
| | NDBC buoys | `ndbc` provider (3rd API row): activestations.xml (1,351 stations, daily refresh) → nearest reporting station within 150 km by haversine (Great Lakes buoys included) → realtime2 text product parsed for WVHT/DPD/MWD/WTMP ("MM" = missing). httpx gained `GetText` on the same pacing/retry/redaction path. |
| | Merge | Assembler `harmonizeMarine`: forecast provider first, buoy fills nil fields (water temp only ever comes from the buoy) — field-wise, never blended. Scheduler tiers: marine 30m priority / 1h recent. Report mode registers the same provider set (M5 parity) — `watchpost report 92057 --json` publishes `marine` + 3 provider statuses; schema artifact regenerated. |
| | UI | MARITIME rows in the detail view when marine data exists: `Swell: WNW 2 ft (period 14 s)`, `Conditions: Slight Chop` (Douglas sea-state bands), `Water Temp: 74ºF (buoy 46224, 17 km)`; feet under ºF, metres under ºC. |
| | Tests | nws marine lift + inland-zero rule; ndbc parse + nearest/inland; assembler merge; sched kind; tty MARITIME render; package density exemption recorded for ndbc (ratified pattern). |

### Session 31 (status legibility + width-smart modals)

| # | Item | Disposition |
|---|---|---|
| 31.1a | "Align the providers" | **Done**: name / role / status columns aligned; freshness in a fixed 7-cell slot (`59m 59s`, ` 1m  5s`, `    55s`, ` 2h 05m`). |
| 31.1b | "WARNINGS shows a ton — make it useful" | **Done**: warnings fold into ISSUE classes by code+provider — provider errors first (✘), then by count (⚠) — each with `×N (M locations)` and the latest message; capped at 8 classes with an overflow note; `none` when clean. 25 seeded cities × 3 providers now read as a handful of rows. |
| 31.2 | "Modals stretch to 60% of terminal width" | **Done** for Status, Location Details, Alerts: base widths are the floor, 60% of the terminal the target on wide screens; search/confirm/help stay fixed. Test-pinned (200 cols → 120; 100 cols → floor). |

### Session 32 (detail grid alignment, sunrise/sunset, more maritime data)

| # | Item | Disposition |
|---|---|---|
| 32.1/32.5 | "Humidity aligns with HIGH/LOW; CURRENTLY temps align with the FORECAST condition column" | **Done**: one content grid for the whole detail view — labels at col 0, primary values at col 14 (the FORECAST condition column), secondary values at col 37 (the HIGH/LOW column). `gridRow` is the single owner; CURRENTLY / TODAY / FORECAST / MARITIME share the two vertical scan lines. Test-pinned (display columns, not bytes — º strikes again). |
| 32.2 | "Maritime: align the data" | **Done** on the same grid: `Swell:` / `Conditions:` / `Water Temp:` values at col 14, `(period …)` / `(buoy …)` at col 37. |
| 32.3 | "Other useful maritime data?" | **Added**: secondary swell (height/dir/period) from the gridpoint; wind-wave height; buoy wind speed + gusts (mph/km-h by units); buoy observation age. Schema `Marine` grew the fields; merge + parsers + fixtures updated. |
| 32.4 | "Do we not have sunrise/sunset for Today?" | **We didn't — no provider carries them.** New `platform/astro` (NOAA sunrise equation, no API) fills every Daily row's Sunrise/Sunset in the assembler from lat/lon in the location's timezone (report JSON gets them too). Two live bugs caught by the reference test on the way: the equation's longitude is west-NEGATIVE, and it wants the noon-based Julian day NUMBER (0h JD + 0.5) — each error was worth 7–12 hours. Oceanside 2026-08-24 now computes 06:17 / 19:26 PDT (NOAA ~06:17 / ~19:25). |

### Session 34 (compact modules for short terminals)

| # | Item | Disposition |
|---|---|---|
| 34.1 | "Bottom help always visible: on short terminals collapse the alert to one row and the radio to two" | **Done**: `compact()` measures whether the full layout fits (chrome + modules + watchlist rows + a 3-row recent minimum + footer); when it cannot, the alert module renders as `nn/nn  ⚠ EVENT · Label    [sev] headline...   [A] Alert Details [←] Previous [→] Next` (body truncated to fit) and the radio as the two-row form from the mock (title · ♪ station · clock · progress bar · VOL · state / controls). ~12 rows reclaimed; height budget + blank-alert reservation follow the mode. Radio controls unified to the new set — `[space] Play/Pause [p] Pin/Unpin [r] Repeat [v] Visualizer [T] Toggle Size [+/-] Volume` — as keymap data now (handlers land with B4); the station line names the focused location. Test-pinned at 30 vs 60 rows. |

### Session 35 (width-responsive radio; terminal-centered modals)

| # | Item | Disposition |
|---|---|---|
| 35.1 | "Radio controls wrap smartly by width" | **Done**: the same `WrapSegments` the footer uses; module height follows the wrapped line count (one `radioLines` builder feeds render AND the height budget). |
| 35.2 | "Volume bar expands/shrinks" | **Done**: `volBar` scales with the module width (inner/5, clamped 6-30 cells). |
| 35.3 | "Play line can disappear at narrow widths" | **Done**: full mode drops the progress line under 70 cols; compact mode fits the bar into whatever room remains, then degrades station → clock so the row never exceeds the module. The compact alert row degrades the same way (body → chip labels → title). |
| 35.4 | "Modals off-center on narrow terminals ([S] too far right)" | **Root cause**: an over-wide compact row inflated the overlay base, and `Overlay` centered on the base's widest line. Now centers on the TERMINAL width (parameter), and no row can exceed it. Test-pinned at 70 cols: every row fits, controls wrap, VOL shrinks vs 200 cols, the [S] modal starts at (70-w)/2. |

### Sessions 36-41 (radio player polish)

| # | Item | Disposition |
|---|---|---|
| 36 | Short title `WWRadio` in narrow compact rows | Done — first degrade step. |
| 37 | State-driven chip labels: `[p] Pin|Unpin`, `[r] Repeat: On|Off`, `[v] Viz: On|Off`, `[T] Size: Min|Max` | Done; toggles live for UAT (B4 wires audio); Size: Min renders the two-row player anywhere. `[t]` kept as theme chooser (D-19) — size stays on `T`; flagged. |
| 38/39 | `[+/-] Vol.`; `[space] Play|Pause` with `▶ PLAYING` bold green | Done. |
| 40 | Compact row spans the module (tail right-aligned); VOL floor 10; name outranks the bar (shortens, never vanishes); station bold bright yellow (RadioStation token) | Done. |
| 41 | Volume redesign `VOL [-]█████░░░░░[+] 55` — VOL bold white, level white; `[+]/[-]` blink green/red on press (bar steps at the 10s, blink = per-press feedback); compact VOL fixed at 10 cells so the scrub bar gets the room; `[+/-]` chip removed | Done — ChipFlashUp/Down tokens + `render.KeyCapWith` (component-level feedback state); volume is real state (step 5). Header: `W A T C H P O S T  v0.0.0-dev` (pipe dropped, gradient unchanged). |

### Sessions 42-43 (volume nits; vertical space)

| # | Item | Disposition |
|---|---|---|
| 42 | Level reserves 3 cells; [+]/[-] mute at 100/0 | Done (`%3d`; KeyCapIf). |
| 43.1 | RECENT / SEARCHED separator → full-width group-style band | Done: `render.Band` (chip style, GroupSectionBG token, bracketed when color is off) spanning the table width. |
| 43.2 | No blank lines above/below it | Done. |
| 43.3 | Watchlist controls move ABOVE the watchlist group labels | Done (right-aligned). |
| ENG | Height budget | The fixed-chrome constant was undercounting by 4 rows since the control row landed (session 26) — recounted to 15 after this layout (compact() and windowSize share it). Narrow-terminal test now measures modals by their own box span. |

### Sessions 44-48 (connected tables, tall terminals, cap 60)

| # | Item | Disposition |
|---|---|---|
| 44-45 | Recent table drops both header rows (band connects the tables); band bg #222; `[LOCATION]` label (no `/ STATION`); rail ▲ rides the band | Done. |
| 46 | Recent window expands on tall terminals (footer visible, 2-row bottom inset); `[ctrl+a] Add`; control row left-aligned | Done. |
| 47 | Modules compact BEFORE the table gives up rows (monotonic table height as the terminal shrinks) | Done — pinned across 100→20 rows. |
| 48 | **Location limit 25 → 60 (10 favourites + 50 most-recent)** — HUM LEAD decision, performance impact acknowledged | Done: `recentCap = 50` (seed list + prepend cap). Launch burst grows to ~250 calls (~5 per location) — ~8-10 s at the 30 req/s politeness pace. Mitigations queued: warm-launch disk cache ("NWS cache refresh"), provider-level parallel fetch (with an NWS-politeness ceiling). M1 is measured on the priority pipeline only and is unaffected. |

### Session 49 (compaction breakpoint for the 60-location table)

| # | Item | Disposition |
|---|---|---|
| 49.1 | "Large terminal never expands the modules; table should flex row-by-row until a 20-row breakpoint (10 fav + 10 recent or any split), then modules minimize" | **Done**: `tableBreakpoint = 20`. Full modules stay while the full layout can show ≥ 20 table rows (favourites + recent window); the recent window flexes row-by-row above that; only when the full layout cannot deliver 20 do the alert/radio modules minimize (the table then regains rows — by design). Supersedes the session-47 all-rows rule, which the 50-row list had turned into "always compact". Swept 120→20 rows in the test; 80 rows renders the full modules. |

### Sessions 50-51 (focus tones; [v] Viz wiring)

| # | Item | Disposition |
|---|---|---|
| 50 | Focused row: grey cells light blue (name/temps unchanged); pointer bold white | Done (FocusCell 117, FocusPointer 1;97). |
| 51 | Wire [v] Viz: max player shows/hides its two visualizer rows; min player inserts one visualizer row between status and controls | Done — default off (2 rows saved); module heights follow the row count automatically; placeholders stay until the max/min visualizers land with B4. |

### Session 53 (footer trim; live theme chooser)

| # | Item | Disposition |
|---|---|---|
| 53.1 | Remove `[tab] Navigate Sections` | Done. |
| 53.2 | Theme chooser modal — list of themes, each pointing the app at its token mapping, applied live without restart | **Done**: `render` theme registry (default + built-ins **High Contrast**, **Monochrome**, **Solarized Night** as overrides over the default table; `RegisterTheme/SetTheme/ThemeNames/ThemeName`, RWMutex-guarded) — every emission already resolves through `Tok()`, so a switch repaints on the next frame. `t` floats **Color Theme**: ↑↓ select, enter applies via the app hook (which also persists `theme = "…"` in config so it survives restarts), esc closes; active theme ✔-marked. **User themes**: `~/.config/watchpost/themes/<Name>.json` `{"tokens": {"temp.hi": "208", …}}` register at startup (unlisted tokens inherit; bad files skipped). Tests: registry live switch + inheritance + refusal, user-file loading, chooser flow with a fake hook. |

### Sessions 54-57 (player mock, alert text, control consolidation)

| # | Item | Disposition |
|---|---|---|
| 54 | Max player per mock (title…VOL+state / station…clock / 3-row visualizer / play line / controls) | Done; player builder split into compact/max rows (P10-04). |
| 55 | [A] modal body text white (titles keep tones) | Done (AlertModalText). |
| 56-57 | Controls consolidated where they act: header `[a] About  [?] Help  [t] Theme  [q] Quit`; watchlist row `[enter] Details  [ctrl+a] Add  [shift+del] Remove  [l] Lookup … [↑↓] Navigate` (right-aligned; smart-wraps on narrow widths); **footer removed entirely** — the recent table takes the freed rows (height budget 11 fixed + control-row lines) | Done. Header shortens the stamp before it could overflow on narrow terminals. |

| RULING | "Use the actual component structure — we should not have to hand-roll our own table components; this was the point of go-studs." + the reference CLI reference (spec struct + data struct) | **Adopted**: the hand-rolled table is withdrawn; `render.LocationTable` now assembles a go-studs `DataTableDefinition` (ColumnDefinition spec + EnhancedTableRow data, GutterWidth 2 — the reference CLI pattern). Mock-measured widths + the component's prefix-zone gutter rule reproduce the mock's absolute offsets exactly (fidelity tests diff the go-studs-rendered header against the mock constant character-for-character). The seam keeps only: column-set selection per width (UAT-2D), watchpost value formatting, and the letter-spaced group-header line (**upstream candidate M6: column groups**). Temps right-justify in 5-cell fields per the mock's " 72ºF"/" 00ºF" rows. |

### Session 59 (data gaps: "UNKNOWN n/a" rows; missing maritime for coastal cities)

| # | Item | Disposition |
|---|---|---|
| 59.1 | **Carlsbad, CA shows `UNKNOWN n/a`** — "flag or re-request / rehydrate as appropriate; if API performance, bring in the parallel request mechanisms" | **Root cause (live-probed, not API latency):** the nearest NWS station for 92008 is **CBDSD**, a mesonet site that never publishes a sky condition and reports a temperature on ~1 in 4 observations (`textDescription: ""`, `temperature: null`). `/stations?limit=1` locked every cycle onto it. **Fixes:** (a) `resolve` keeps the **4 nearest stations** as a fallback chain; `fetchObs` walks it until a station reports a *complete* observation (temperature + condition), remembers that station as preferred (1 call next cycle), and otherwise keeps the best partial. Carlsbad now reads **KCRQ** (McClellan-Palomar, 2nd nearest). (b) **Rehydration from the hourly forecast** in the assembler: a sparse observation's missing temperature / unknown condition fills from the forecast period covering now, provenance recorded (`fill_from: forecast`); observed values are never replaced; a location with no observation at all stays a loading state. |
| 59.2 | Batch fail-fast — one bad location blanked every location after it in the same tier cycle (priority pipeline: all 10 favourites) | **Fixed:** NWS obs/forecast and NDBC now **fail-soft** — successes land in `PerLocation`, failures travel joined in `frag.Err`; `Assembler.Apply` merges a partially failed fragment (status still degrades, warning still appended; unserved locations keep prior data). |
| 59.3 | Re-request / rehydration | **Scheduler retry-before-cadence:** locations a provider could not serve are re-requested *alone* on a 10 s / 20 s / 40 s backoff (capped at the tier cadence, max 3) before the regular cadence resumes; each retry publishes so the row fills. A transient NWS 5xx heals in seconds instead of at the next 30-minute forecast tick. Pinned: retry targets only the failed location; bounded when a location keeps failing. |
| 59.4 | Parallel request mechanisms | **Landed (bounded):** per-location fan-out inside one NWS `Fetch` (6 concurrent; the httpx 30 req/s bucket stays the single politeness governor) and **singleflight** on `/points` resolution so the alerts/obs/forecast tiers firing together at launch share one round trip (~⅓ fewer launch calls). Warm-launch disk cache remains queued. |
| 59.5 | **Maritime missing for San Francisco, downtown San Diego (92101), Seattle, Miami; wanted for anywhere < 5 mi of a coast and major lakes (South Lake Tahoe)** | **Root cause (live-probed):** for all four the nearest "active" NDBC station is a NOS **tide gauge** (`pxoc1`, `sdbc1`, `ebsw1`, `vakf1`) — no wave product under the lower-case id the station list carries (404) — and the old fetch aborted the whole fragment on that first 404. **Fixes:** product URLs upper-cased (`SDBC1.txt` exists and carries water temperature 1.2 km from downtown SD; `VAKF1` 4.5 km from Miami); selection is now **waves from the nearest true buoys** (up to 4; SD → SCRIPPS Nearshore 46254 at 19.6 km) **+ water temperature from the nearest station of any kind reporting one** (up to 8 by distance; tide gauges usually do), merged; a station cache (10 min) lets the coast-dense watchlist share one download per buoy. Puget Sound has **no wave-reporting buoy** (UW buoys carry wind only) — Seattle gets water temperature from the Tacoma gauge (38 km) and no sea state; that is a data-source limit, shown with the buoy distance. **South Lake Tahoe: no NDBC station within 150 km** (NDBC covers the Great Lakes, not Tahoe). Free Tahoe sea-state exists only through UC Davis TERC's real-time buoys (unofficial endpoint) — **HUM CALL** whether to add that as a lake provider; nothing lands for Tahoe until then. |

### Session 60 (observing station in the row)

| # | Item | Disposition |
|---|---|---|
| 60.1 | "Does each location have a station callsign like KCRQ?" — investigated and reported before changing anything | Every location resolves to an NWS observation-station id; it is an ICAO airport callsign (KCRQ/KSAN/PAHO/PHTO) in 7 of 8 live samples after the UAT 59 "first complete reporter" rule, and a mesonet/RAWS id (CBDSD, DYNN2, PPSC1) otherwise. It is observed state (the fallback chain can move it), and it carries provenance (distance). |
| 60.2 | Re-add the hidden LABEL column as **`WX STN`** (so it does not collide with the NOAA radio transmitter callsign B4 will show) and add a **`DIST`** column (`nnn km`); both hide **before ZIP** | **Done**: `WX STN` (6) + `DIST` (6) sit between NAME and ZIP inside the LOCATION band from **131 cols** (the 115-col minimal full layout + 16); the 125-col mock layout is unchanged (they are the first columns to leave as the width narrows, ahead of TOMORROW), and **ZIP is now the last identity column to go** — dropped only when NAME would otherwise fall below its 10-cell floor. Distance follows the global units rule like Height (**miles under ºF, km under ºC**) — the mock's `nnn km` is the metric form; say the word if it should be km always. Station distance comes from the `/stations` geometry (haversine moved to `platform/geo`, shared with NDBC at its second caller). Blank while loading / unknown, never a fake 0. Extended day columns now claim width beyond 131 (5 days from 240 cols). |
| 60.3 | Units as-is (display units); **Station + Distance also in the location Detail modal** — a power-user detail the table surfaces only when there is real estate, otherwise one drill-in away | **Done**: CURRENTLY gains `Station   :   KCRQ                   Distance  :  4 mi` on the shared grid (label / col 14 / col 37), omitted while loading. The maritime `(buoy 46224, … )` distance now goes through the same `Distance` formatter, so all distances follow the units rule from one owner. |

### Session 61 (tides & currents)

| # | Item | Disposition |
|---|---|---|
| 61.1 | "Add available tidal info to the maritime section — let's see what the API provides" → approved rows | **Done — `domains/marine/coops`** (NOAA CO-OPS Tides & Currents, free, no key; the same NOS stations NDBC lists as tide gauges). Per location: **high/low predictions** from the nearest of 3,499 tide stations (≤ 60 km), the **observed water level** from the nearest of 301 gauges (≤ 60 km, a nicety — a gauge outage never drops the predictions), and **max-flood / max-ebb / slack predictions** from the nearest of 4,430 current stations (≤ 40 km). Metric + GMT on the wire (metres MLLW, cm/s → m/s magnitude), SI in the snapshot (`tide_level`, `tides[]`, `currents[]`, station names + distance); the API's HTTP-200 error envelope is detected and surfaced. Predictions memoized 1 h, levels 10 min, station lists daily — the memo cache was extracted to `platform/memo` at its second caller (NDBC now uses it too). MARITIME rows: `Tide: Rising 3.7 ft (La Jolla, 24 mi)` · `Next High: 1940 5.7 ft  Next Low: 0249 -0.1 ft` · `Currents: Flood 1.4 kt (slack 1605)` — trend follows the next predicted event, times local, tide heights to a tenth of a foot (cm under ºC), currents in knots under both unit systems. Tide-only coasts (Puget Sound) now get a MARITIME section from tides alone. **Great Lakes** water levels exist in CO-OPS but on lake datums (no MLLW, no tides) — queued as a follow-on; Tahoe remains without a source. |
| 61.2 | Live verification | CO-OPS answers request bursts with **HTTP 403** (observed after three back-to-back one-shot reports; lifted within ~20 s). httpx treats 403 as non-retryable by design (NWS uses it for a missing User-Agent), so the one-shot report simply omits the block; in the dashboard the UAT 59 retry-before-cadence re-requests the affected locations 10/20/40 s later and the memo caches keep steady-state volume at one call per station per TTL. Watch item: if throttling recurs at 60 locations, pace the CO-OPS client separately. |

### Session 62 (tide row alignment)

| # | Item | Disposition |
|---|---|---|
| 62.1 | "Next High / Next Low row is not aligned correctly" | **Done**: times render as the mock's `19:40` (were `1940`, one cell short), and tide heights use a fixed 4-cell numeric slot (`%4.1f ft` / `%4.2f m`) so the height starts at col 22 on both the Tide and Next rows and a negative low (`-0.1 ft`) never shifts `ft`. Pinned for Rising/Falling and positive/negative values. |

### Session 63 (maritime grid)

| # | Item | Disposition |
|---|---|---|
| 63.1 | "Delayed loading makes the Next High / Next Low row not align" (screenshot: `Next High: 19:32 5.5 ft Next Low: 02:42 -0.1 ft`) → separate rows, re-ordered for scannability (mock) | **Root cause was width, not timing**: the combined row was 74 cells with its `MARITIME │ ` prefix and overran the modal's wrap budget (details modal floor 85 → 78), so the wrapper collapsed its spacing. **Done per mock**: MARITIME is its own grid — label 16, first sub-column 8 (direction / trend / time / phase), fixed 4-cell number + unit, note at col 41 — in scan order **Observed · Conditions · Water Temp · Swell (· Swell 2 · Wind Waves · Buoy Wind) · Tide · Next High · Next Low · Currents**; one row per next high / low; swell heights to a tenth (`3.0 ft`); notes capped at 26 cells so no row can exceed 78 (pinned). The whole-foot `Height` formatter had no callers left and was removed. Other sections keep their `label:` / col-14 / col-37 grid — say the word if the colon-less 16-wide labels should propagate. |

### Session 64 (tides missing on the priority watchlist; maritime column alignment)

| # | Item | Disposition |
|---|---|---|
| 64.1 | "Tides are missing in places I would expect it (Oceanside, Carlsbad) but present in others (New York)" | **Reproduced with a wiring-exact probe** (priority batch of the two + 50 recent seeds; the probe's own first version stopped its schedulers early and had to be corrected before it measured anything). Cause: the favourites' requests share one 30 req/s token bucket with the seed pipeline's ~200-call launch burst, and the marine providers made their calls *sequentially* per location — every call re-queued behind the burst (~7 s per wave), so the two-location priority batch took minutes; the scheduler also published only after every provider in the tier finished. **Fixes:** (a) **priority lane in httpx** — `WithPriority(ctx)` paces the favourites' pipeline on its own lane at the same rate (momentary ceiling 2× RatePerSec, still polite), so they never queue behind the seed list; (b) nws-marine, NDBC and CO-OPS fan out **concurrently per location** (`snapshot.FetchEach`, the NWS helper moved to the snapshot package at its second caller) and CO-OPS fetches its three products concurrently, so a batch reserves all its slots up front; (c) the scheduler **publishes after each provider's fragment** (pinned: a publish arrives while a slow provider is still fetching); (d) CO-OPS on its **own 5 req/s client** (30 concurrent prediction calls probed 200; repeated station-list downloads are what drew 403s); (e) memo **errors expire after 30 s** (were held for the value TTL — an hour for predictions), so a throttled call heals on the 10/20/40 s retries; (f) tide stations get a **3-deep fallback chain** — Philadelphia's nearest has no MLLW datum ("No Predictions data was found"). **Measured after the fix:** priority tides for Oceanside and Carlsbad within 5 s, ahead of every recent location. |
| 64.2 | Maritime values 2 cols right of the sections above; `(…)` notes to the left edge of `HIGH` (crowding the scroll rail) | **Done**: MARITIME labels on the shared value column (col 14) and notes on the shared secondary column (col 37, the "H" of HIGH); the section prefix width is now a named constant the wrap budgets derive from; the note cap follows (26 cells). Pinned. |

### Session 65 (detail modal: flush left edge)

| # | Item | Disposition |
|---|---|---|
| 65.1 | Move all detail content left by 2 cols so CURRENTLY aligns with the modal header label (the freed 2 cols become right-hand spacing — useful for long names like Los Angeles); all content flush against that edge, including separator lines and alerts | **Done**: `detailRow` drops its 2-space lead (section label column now starts at the header label's column — measured 28 → 28), the alert divider and `⚠ EVENT` title start flush, alert prose indents 2 and bullets 4 from that edge (continuations 6), and the controls line is flush too. The divider keeps its right edge on the `W` of LOW (its length now derives from `detailPrefixW`, which absorbed the duplicate `detailRowChrome` constant). Wrap budgets are unchanged, so the 2 cells are pure right-hand slack. |

### Session 66 (maritime notes; divider and alert width)

| # | Item | Disposition |
|---|---|---|
| 66.1 | Maritime `(…)` notes still push out — 4 cells between the data and the parenthetical on each line | **Done**: notes follow the value by exactly 4 cells (no fixed column); the section compresses by 3–7 cells per row (`Water Temp    75ºF    (buoy 46224, 11 mi)`, `Swell         SSW      3.0 ft    (period 14 s)`). Pinned. |
| 66.2 | Divider to 3 cells before the vertical scroll control | **Done**: divider length = modal content width − 2 (the rail sits one cell past the content width), supersedes the UAT 33.1 "W of LOW" edge. Pinned against the live modal geometry. |
| 66.3 | Advisory / alert text should use the extra width | **Done**: alert prose and bullets wrap to the divider's right edge (prose +6 cells, bullets +8 per line vs. before). |

### Session 67 (maritime note column)

| # | Item | Disposition |
|---|---|---|
| 67.1 | Notes must align at one gap from the section's WIDEST value (a narrow value like `75ºF` was pulling its note in); then narrow the gutter 4 → 2 (Los Angeles) | **Done**: two-pass layout — rows are collected, the widest value measured, and every note lands in one column 2 cells past it (`Water Temp    75ºF             (buoy …)` / `Swell         SSW      3.0 ft  (period 14 s)`). Scannable and never wider than the data needs. Pinned. |

### Session 68 (About window)

| # | Item | Disposition |
|---|---|---|
| 68.1 | Wire `[a]` About — mock provided (60 cols: centred title + `{v …}`, "Data Provided by" list, "Built with" stack, maker lines) | **Done**: `a` toggles the window (esc / any other modal closes it; ↑↓ scroll on short terminals); untitled panel gets an unbroken top rule. Lines are composed on the mock's 58-cell interior so every offset matches character-for-character (pinned). Version comes from the build, the Go version from the runtime, and the provider list from the live registry (NWS and NOAA first; any future provider's attribution lists itself). Two mock strings corrected as factual errors — **HUM CALL if you'd rather keep them**: "National Oceanographic & Atmospheric Agency" → *National Oceanic & Atmospheric Administration*; "Stylized Terminal UI Design Lanauge" → *… Design System* (go-studs' own name). Corners stay square per UAT 10.5. |
| 68.2 | Help modal controls must use the chip controls like the other modals | **Done**: `[esc] Close   [↑↓] Scroll` now renders through `KeyCap` (chips with colour, bracket fallback without), inset like the status/alert modals. Pinned with colour on. Modals now wrap to their full content width when the body fits without a scroll rail (shared `wrapModal` owns both the render and the scroll bounds). |

### Session 69 (incremental lookups)

| # | Item | Disposition |
|---|---|---|
| 69.1 | A lookup re-requested and refreshed the **entire** list (~250 calls); it should request just the new location, put it on top, drop the oldest, and let the normal cadence rehydrate the new permutation | **Done — commits are incremental.** `Assembler.SetLocations(refs)` reconciles the tracked set in place (kept rows keep every section; newcomers start empty; the order follows the list); `Scheduler.Update(refs)` continues the cadence over the new set and fetches **only the newcomers now, on every tier**. The app's `commit` diffs both pipelines: the RECENT list stops the dropped location's scheduler and starts one for the newcomer (its first cycle lands in seconds); the watchlist scheduler is updated in place. Nothing that was loaded re-requests, so rows never fall back to `n/a`/dots, and the next cadence batches the whole permutation through the parallel fan-out as before. Pinned: newcomer-only immediate fetch, cadence over the updated set, removals fetch nothing, kept data survives reorders. |

### Session 70 (About inset)

| # | Item | Disposition |
|---|---|---|
| 70.1 | About: left-aligned text flush against a 3-col inset (was 4 — the NOAA line sat 2 cells off the right edge); even 3-3 split | **Done**: inset 3; `│   National Oceanic & Atmospheric Administration (NOAA)   │` is 3-52-3. Pinned. |

### Session 71 (TODAY HIGH after sunset; caching strategy)

| # | Item | Disposition |
|---|---|---|
| 71.1 | "Lots of n/a in TODAY HIs — data removed from payloads as the day goes on, or perf?" | **Data shape, not performance**: `/forecast` drops a day's daytime period once local evening starts (first period becomes "Tonight"), so every location past ~6 PM local lost its HIGH — the East Coast first, then Central. **Fixed**: today's missing HIGH/LOW fill from the raw gridpoint `maxTemperature`/`minTemperature` series (kept all day), with `fill_from: nws:gridpoint` on the Daily row; forecast-period values are never overwritten. The gridpoint download is shared with `nws-marine` through the client cache — one request per grid per cycle (pinned). |
| 71.2 | Caching strategy — must make sense, be understood by humans, and be as performant as possible | **Designed and built** — see `03-architecture-design/caching.md`. **One rule**: every GET is cached by URL in the HTTP client for (caller `TTL` → server `max-age`/`Expires` → none); **two tiers** (memory; disk under the OS cache dir so relaunches are warm — one inspectable JSON file per URL); singleflight per URL; 30 s negative cache for non-retryable 4xx; hits never consume a pacing token. Server lifetimes were probed: NWS points/stations 1 day, hourly 1 h, obs 5 min, alerts 5 s, forecast/gridpoint until the next issuance; NDBC files 10 min; CO-OPS declares nothing (`no-store` on astronomical predictions — overridden by a stated TTL). NDBC and CO-OPS now *state* lifetimes instead of keeping memos; `platform/memo` is retired (fewer concepts). Report mode shares the disk tier, so one-shots are warm too. |

### Session 72 (rehydration optimisation — all five recommendations)

| # | Item | Disposition |
|---|---|---|
| 72.1 | Batch the RECENT list's alerts | **Done**: one scheduler over the whole recent list makes a single `/alerts/active?zone=` call every 2 min (was 50 per-location calls — 25 of ~40 requests/min); it is reconciled on lookup like everything else (`Update` fetches a newcomer's alerts at once). |
| 72.2 | NDBC `5day2` instead of `realtime2` | **Done**: identical columns, 23 KB vs 207 KB. |
| 72.3 | Skip the gridpoint for inland grids | **Done**: a grid with no marine series is remembered inland for a day (pinned: one download, not one per cycle). |
| 72.4 | Split marine into observations vs predictions | **Done**: `KindMarineObs` (NDBC buoy files, CO-OPS water level via the new `coops-obs` provider) on a **10-min** tier in both pipelines; `KindMarine` (gridpoint swell/wave, tide + current predictions) stays 30 min / 1 h. Domains `marine` / `marine-obs`; `sched.Serves` exported for one-off hydration. |
| 72.5 | Hourly forecast lazily for RECENT rows | **Done**: `KindForecastHourly` is its own tier (priority: 30 min with the daily tier); the RECENT list has none — a row hydrates when Details opens on it (dashboard `Hydrate` hook → app fetches and publishes) or when it is a fresh lookup. Pinned: recent rows call the hook once per open, priority rows never. |
| 72.6 | "Make sure all this documentation is getting saved and recorded" | **Done**: `architecture.md` §11 (UAT-driven infrastructure addendum — pipelines/tiers, scheduler behaviours, providers, rehydration, caching pointer), `04-development/b3-infra-ledger.md` (one row per infra change with its on-screen trigger, measured baselines, queue), `06-key_learnings/b3-ux-backwards.md` (why working UX-backwards found what the plan could not). |

### Session 73 (adversarial performance / quality pass)

| # | Item | Disposition |
|---|---|---|
| 73.1 | btop: watchpost at 177 threads / 95 MB vs peers at 18–60 threads / 17–88 MB — "take an adversarial pass before more features" | **Measured first** (headless wiring-exact probe: threads 15 → 137 within 5 s of launch; live heap 29 MB of which the caches were 24 MB; heap profile: raw bodies 16 MB + **10 MB of base64 decoding of the disk-cache files**). **Fixed**: pure-Go DNS resolver, 8 connections per host, in-flight caps (16 normal / 8 priority lane) — threads **15**; memory cache **8 MB budget** with expiry-then-LRU eviction, bodies over 2 MB disk-only (the CO-OPS station lists); disk format = JSON header line + **raw body**, written by one goroutine off the request path — resident **80 → 55 MB**, live heap **29 → 15 MB**. Goroutines (~214) are the per-location schedulers by design. Full numbers in `b3-infra-ledger.md`. |
| 73.2 | Quality: staticcheck + golangci-lint | **Clean**: 6 unchecked error returns (tests), 5 style findings (Fprintf, De Morgan, tagged switches) fixed. `-race`, P10, verify unchanged. |

### Session 74 (threads, part 2 — the real process)

| # | Item | Disposition |
|---|---|---|
| 74.1 | "watchpost now shows 82–91 threads — better but not the 15 the tests say" | **Measured the real TUI process** (pprof on loopback, goroutine dumps in the first second): (1) `Assembler.Snapshot()` called `time.LoadLocation` for every location on every publish *under the assembler lock* — a zoneinfo file open per call, ~300 publishes at launch → 140 threads with 200 schedulers queued on the lock; (2) with that fixed, 200 scheduler goroutines starting in the same instant made short cache-file syscalls together → 90 threads in 80 ms. **Fixed**: `platform/tz` memoizes zones (drop-in for `time.LoadLocation`, single owner); the app **coalesces publishes** (50 ms window; `sched.OnPublish` is now a plain notification and the app takes one snapshot per window); disk-cache reads bounded to 4 concurrent; recent schedulers start 10 ms apart. **Real process: 144 → 20 threads, 89 → 76 MB.** The headless probe had hidden both (its cache was pre-warmed by its own first run) — recorded as a learning. `WATCHPOST_DEBUG_PPROF=1` stays as the way to read a live process. |

### Session 75 (About: data-source credits)

| # | Item | Disposition |
|---|---|---|
| 75.1 | About must carry accurate API data-provider credits (compliance) plus "free to use with attribution" | **Done** — audit first: NWS, NDBC and CO-OPS are NOAA public domain (NWS asks only for a User-Agent, which every request carries); the embedded offline index is **GeoNames (CC BY 4.0)** and the geocoder fallback is **Open-Meteo (CC BY 4.0)** — both *require* attribution, and both had been promised "in the About view (OQ-15)" without being there. The About list is now a single app-owned list (`app/credits.go`) fed by each source package's credit line — five lines, each ≤ 52 cells — followed by the notice **"All sources free to use with attribution."** Pinned: every source present, both CC lines present, every line fits. |

### Session 76 (B4 step 1 — NOAA Weather Radio audio, Live path)

| # | Item | Disposition |
|---|---|---|
| 76.1 | "Get the audio to actually play for locations with stations and/or the nearest NOAA broadcast" | **Built the Live path of architecture §5.** *Transmitter table*: `tools/nwrtable` parses NWS's `CCL.js` (the public-domain county-coverage arrays: SAME code, callsign, frequency, **site lat/lon**, power, status) into the vendored `domains/radio/stream/transmitters.csv` — 1,035 transmitters, 107 KB, embedded. *Directories*: wxradio.org and weatherUSA Icecast status documents (5-min TTL through the client cache), MP3 mounts only, merged by callsign. *Resolver*: county UGC from the cached NWS point → SAME → the covering transmitter when relayed, else the nearest relayed transmitters by distance (capped at 3), each labelled with its distance and "(nearest relayed)". *Player* (`domains/radio/player`): Icecast reader (ICY metadata strip, 15 s stall watchdog, `Icy-MetaData` + User-Agent), 3 s preroll, go-mp3 decode, linear resample to one 44.1 kHz oto context (created lazily, never closed), per-mount reconnect backoff 1→30 s ±50 %, mount failover, status callback; Output behind an interface so tests need no device. *Dashboard*: `[space]` tunes the focused location / stops; `[+]/[-]` push volume live; the station line shows the resolved transmitter; state label STOPPED / CONNECTING / RECONNECTING / PLAYING / NO STREAM. **Live smoke on this machine**: Monterey KEC49 reached PLAYING through the real oto device and held. |
| 76.2 | Coverage reality | Relays carry ~118 of 1,035 transmitters (~11 %). Oceanside/Carlsbad's covering transmitter **KEC62 San Diego is not relayed** — they hear the nearest relayed one (Victorville WXM66 / Monterey), clearly labelled, until Synth lands. The row names the unrelayed covering transmitter when nothing is in reach. |
| 76.3 | "Do we consider the teletext → our own voice ('Bruce') fallback an option?" | **Yes — it is the plan (§5 Synth) and the numbers make it essential** (89 % of transmitters unrelayed). Next step: NWS text products → normalizer → TTS → oto, behind the §10.5 exec-safety rules. HUM CALL on the voice engine (see 76.4). |
| 76.4 | "Must work on Windows and Linux (especially Linux) — prefer a general cross-OS solution" | **Live path: proven, with one pinned decision.** oto **v3.4.1 does not build for Linux without cgo** (its Unix driver is cgo); **v3.5.0-alpha.11** carries the purego Linux path AI-5 anticipated and builds `CGO_ENABLED=0` for linux/amd64, linux/arm64, windows/amd64, darwin/arm64, darwin/amd64 — pinned, playback re-verified on this Mac with the new driver. Tracked risk (AI-5's own counter-argument): a pre-release audio driver; mitigation is the pin plus the "external player" escape hatch if a regression lands before 3.5 goes stable. Linux runtime still needs PulseAudio/PipeWire or `libasound.so.2` (standard on desktops). **Synth voice**: no production-quality pure-Go TTS exists; recommended general approach = one external engine if present (Piper or espeak-ng, both cross-OS, spawned by argv only) with per-OS built-ins as fallback (`say` on macOS, SAPI via PowerShell `-File` on Windows) and a text ticker when none — HUM CALL before step 2. |
| 76.5 | Credits / terms | About gains the NWR transmitter-list credit, the relay credit (wxradio.org & weatherUSA, community) and the relays' condition of use ("Relayed audio lags; not for life-safety use."). One connection per listener; directories polled at most every 5 minutes; 403/404 honoured without retry storms — the AI-4 obligations. |

### Session 77 (B4 step 2 — synthesized broadcast, "Bruce")

| # | Item | Disposition |
|---|---|---|
| 77.1 | Approved: Piper; watchpost installs it for users on first run / setup | **Built the Synth path (§5)** — `domains/radio/synth`: NWS text products per office (HWO, SPS, NOW, ZFP in broadcast order; product text cached a day), a **normalizer** (de-wrap, header/UGC/footer removal, "..." and period tags to sentences, CAP bullets, directive abbreviations, shouted words read quietly with proper nouns kept — golden-tested on a live ZFP), a **composer** (station ID + time, current conditions from the location's observation, active alerts, products, sign-off with the "not a substitute for official warnings" line), and a **PCM source** that narrates a cycle, re-plans from fresh products at cycle boundaries (never mid-segment, §10.4), caches rendered segments by issuance key. The player gained `StartSource` for any PCM source; the deck falls to Synth when nothing relays the location **and when a relay fails outright** (Live → Synth). The player's second line shows the sentence being narrated (or install progress). |
| 77.2 | Voice engines | **Piper's macOS release cannot run** — the upstream archive ships no dynamic libraries (verified: `otool` lists `@rpath/libespeak-ng`, none present; no Homebrew formula either). So: **Piper on Linux (x86_64, aarch64, armv7) and Windows (self-contained archives), the built-in `say` on macOS** — one `Voice` seam, one pipeline. All artifacts SHA-256-pinned in code (release 2023.11.14-2; voice `en_US-lessac-medium`, 22.05 kHz, MIT) and verified before extraction; archive entries that would escape the install dir are refused. **Install**: at `watchpost setup` on Linux/Windows (progress on stderr, failure never fails setup) and again on first tune-in if missing (progress in the player). §10.5 held: narration reaches `say` by a 0600 temp file and Piper by stdin — never an argv element; pinned by a test that narrates shell metacharacters through the real `say`. |
| 77.3 | Live verification | On this Mac: real SGX products → normalizer → `say` → oto; the cycle opens "This is Watchpost synthesized weather radio for Oceanside, CA. The time is …", reads current conditions and the heat advisory, then the zone forecast. |

### Session 78 (radio default: location first)

| # | Item | Disposition |
|---|---|---|
| 78.1 | Selecting Oceanside still tuned Victorville — it should default to Synth | **Agreed and done.** A neighbour's broadcast is a neighbour's forecast (Victorville reads the High Desert). Default is now: the covering transmitter plays **live only when it is relayed**; otherwise the location's own products play **synthesized**. The detail line still names the nearest live broadcast (`KEC62 San Diego 162.400 MHz — not relayed · nearest live: WXM66 Victorville 75 mi`) so a later station/voice chooser can offer it as an explicit choice. Pinned. |

### Session 79 (broadcast script; LIVE RADIO line)

| # | Item | Disposition |
|---|---|---|
| 79.1 | New lead and tail scripts | **Done**: lead — "This is Watchpost Weather Radio serving {location}. This forecast is from the National Oceanic and Atmospheric Administration, and is for {Day, Date} until {Day, Date}." (today through the seventh day, in the location's zone); tail — "This is {voice} for Watchpost Weather Radio. You can change your correspondent voice in your Watchpost CLI application settings." The correspondent's name is real: macOS reads its selected voice (e.g. Samantha), Piper voices use their given name (Lessac). Two typos in the dictated script corrected ("forecase", "Wathpost"); NOAA's name kept as approved in UAT 68. |
| 79.2 | "Live radio has no timestamps — show LIVE RADIO in green" | **Done**: the player's second line reads **LIVE RADIO** (bold green) on a relay; the narration marquee stays for Synth. Pinned. |

### Session 80 (playing indicator)

| # | Item | Disposition |
|---|---|---|
| 80.1 | Green ▶ in the table's radio column for the location that is playing (groundwork for repeat: ALL across the watchlist) | **Done**: the dashboard remembers the tuned location's key; its row wears a green ▶ in the ♪ column while connecting/playing; no other row does. Pinned. |

### Session 81 (synth narration fine-tuning)

| # | Item | Disposition |
|---|---|---|
| 81.1 | Broadcast drifted into the Orange County forecast; tail never heard | **Root cause**: a Zone Forecast Product carries every zone the office serves; the whole thing was read. **Fixed**: products are filtered to the blocks whose UGC line covers the location's forecast zone or county (`FilterUGC`, ranges like `CAZ043>048` expanded) plus the preamble; the location's zone comes from the cached NWS point (`ForecastZone`). A cycle is now one location's forecast, and it ends with the tail. |
| 81.2 | Long pause mid-broadcast; marquee stuck on one sentence | **Root cause**: a zone block was one ~2,000-character segment — rendered synchronously (the pause) and shown as one marquee entry (the "stuck" line). **Fixed**: paragraphs split at sentence ends into ≤280-character segments; segments render **one ahead of playback**; the marquee advances per segment. |
| 81.3 | "CA" read as letters | **Fixed**: postal abbreviations expand in place context only — after a comma ("Oceanside, California"), beside a hyphen ("California-San Diego County"), or after a Title-case word ("San Diego California"); words that are also English ("IN EFFECT", "OR") expand only after a comma/hyphen. The lead expands the location's state. |
| 81.4 | "442 PM" read as "four hundred forty-two" | **Fixed**: `hmm AM/PM` becomes `h:mm AM/PM` ("4:42 PM"); AM/PM are kept as read. |
| 81.5 | LAT/LON number runs | **Fixed**: `LAT...LON` and `TIME...MOT...LOC` lines (and their wrapped coordinate rows) are never narrated. |
| 81.6 | ~5 s to stop after space | **Fixed**: the player pauses within 50 ms of cancel and the source's pipe unblocks mid-segment (Halt ≤ 500 ms pinned; source stop ≤ 200 ms pinned). |

### Session 82 (narration rule: NWS)

| # | Item | Disposition |
|---|---|---|
| 82.1 | "NWS" → "National Weather Service" | The rule already existed; the escape was the CAP **headline**, which entered the script raw. **Fixed**: headlines go through the same word rules (`NormalizeLine`: clock times, abbreviations, state names) — "…at 2:08 PM Pacific Daylight Time … by National Weather Service San Diego California". Pinned. |

### Session 83 (marquee pacing; Repeat wiring; ∞ glyph; named voice)

| # | Item | Disposition |
|---|---|---|
| 83.1 | Marquee breaks midway / doesn't keep up | It was a static 60-cell truncation. **Now a speech-paced marquee**: each narrated line arrives with its spoken length; the window follows the voice (the word being spoken sits a third of the way in), short lines stay static, and a new line restarts the clock. Pinned at 0 / 0.5 / 1 progress. |
| 83.2 | Synth repeated with Repeat off | The source looped unconditionally. **Wired**: `[r] Repeat` now reaches the player — off = one broadcast then the stream ends (the player reports *broadcast complete* as a stop, not a failure); on = loop the location's cycle. |
| 83.3 | Repeat on → ∞ instead of ▶ | **Done**: the playing row wears a green ∞ while repeating (`R` under --ascii). |
| 83.4 | Tail says "macOS voice" | macOS no longer exposes the selected voice through the preference the adapter read. **Fixed**: the default macOS voice is explicitly **Samantha** (shipped with every macOS), so `say -v Samantha` runs and the tail names her; `[V] Voice` will make it a preference. |
| 83.5 | "CLI" said as "See El Eye" | **Done**: a voice-only pronunciation table (`Pronounce`) spells "CLI" as "C L I" for the synthesizer while the marquee keeps "CLI". Pinned with a spy voice. |

### Session 84 ([V] Voice)

| # | Item | Disposition |
|---|---|---|
| 84.1 | `[V] Voice: {Name}` after `[v] Viz`; the voice changed after a rebuild | **Done**: `V` floats **Correspondent Voice** — macOS voices from `say -v ?` (English, argv only), the installed Piper voice elsewhere; ↑↓ / enter / esc; the choice is persisted as `voice = "…"` in config and a playing synthesized broadcast re-tunes so the change is heard at once; the chip shows the chosen name. The rebuild surprise was UAT 83 making Samantha the explicit default — now the voice is a saved preference, never a guess. Pinned. |

### Session 85 (voice chooser freeze)

| # | Item | Disposition |
|---|---|---|
| 85.1 | Voice chooser freezes / soft-locks; no pointer; needs chip controls | **Root cause (mine)**: the chooser called the `Voices` hook — `say -v ?`, 1–2 s — on every render. **Fixed**: the app lists voices once in the background at startup; the chooser snapshots the list when it opens and renders from it (pinned: the hook runs once per open across repeated renders); the pointer `›` moves with ↑↓ (pinned); controls read `[↑↓] Move  [enter] Select Voice  [esc] Cancel`. |

### Session 86 (voice preview)

| # | Item | Disposition |
|---|---|---|
| 86.1 | Preview the highlighted voice with "This is {name} for Watchpost Weather Radio" | **Done**: `[p] Preview` in the chooser speaks the sample in the highlighted voice on its own player, mixed over the broadcast, leaving playback state untouched (pinned). Controls: `[↑↓] Move  [p] Preview  [enter] Select Voice  [esc] Cancel`. |

### Session 87 (curated macOS voices)

| # | Item | Disposition |
|---|---|---|
| 87.1 | Only nine macOS voices suit a radio script: Aman, Daniel, Eddy (English US), Karen, Moira, Reed (English US), Rishi, Tara, Tessa | **Done**: the chooser lists exactly those, in that order, when installed (names as `say -v ?` reports them — Eddy and Reed only in their English (US) variants); the default is the first installed curated voice (Samantha remains only a technical fallback when none is). Pinned. |

### Session 88 ("macOS voice"; Samantha back)

| # | Item | Disposition |
|---|---|---|
| 88.1 | "What was 'macOS voice' — a hidden option? I think that was the first voice I heard" | It was the label for `say` with no voice named, i.e. **the Mac's system voice**. Identified empirically: the default renders the test phrase to 70,300 bytes and no voice `say -v ?` lists comes within 12 KB — on modern macOS the system voice is a Siri voice `say` cannot name. **Now a real, first entry: "System Voice"** (runs `say` with no -v; the tail says "This is the System Voice for Watchpost Weather Radio"); it is also the default on a fresh setup, so first-run sounds as it originally did. |
| 88.2 | Re-add Samantha | **Done**, in alphabetical position (after Rishi). The list is now System Voice + ten. |

### Session 89 (player: no timeline placeholder, no play line in the narrow player)

| # | Item | Disposition |
|---|---|---|
| 89.1 | Remove `00:00 / 00:00` when nothing is playing | **Done**: the line is empty until there is a narration or LIVE RADIO to show. |
| 89.2 | Remove the play line from the narrow player | **Done**: the ━━━ bar is gone from the compact/min player (nothing to scrub); the max player's bar stays for now. |

### Session 90 (max player marquee layout)

| # | Item | Disposition |
|---|---|---|
| 90.1 | Marquee fills the rest of the row after the full location name, 4-cell buffer; max player only | **Done**: line 2 = location (full width) + 4 cells + marquee (or LIVE RADIO) to the row's end; the marquee window is whatever room remains. The min player shows no marquee. Pinned. |

### Session 91 (chip label)

| # | Item | Disposition |
|---|---|---|
| 91.1 | `[V] Voice: System Voice`, not "the System Voice" | **Done**: the chip shows the chooser label; the spoken form stays in the tail. On macOS the default is now unambiguously System Voice (always present — no background listing decides it). Pinned. |

### Session 92 (visualizer)

| # | Item | Disposition |
|---|---|---|
| 92.1 | Visualizer effects for Live and Synth; CLIAmp for style; standard audio visualization; 3 rows in the max player, 1 in the min | **Done** — CLIAmp's default *Bars* mode, read from its source (`ui/visualizer.go`, `ui/vis_bars.go`, `player/tap.go`): ten bands, one blank cell between them, each as wide as the room allows; heights in the fractional block glyphs `▁▂▃▄▅▆▇█`; the spectrum gradient by row — bottom third green, middle yellow, top red (with three rows that is exactly one tier per row). Bars sit inside the bracketed frame the UAT 51/54 mocks reserved. **Signal path**: a tap on the engine's PCM (before the volume, so the bars show the broadcast, not the knob) → a 2048-point Hann-windowed FFT → band power on a dB-like scale (CLIAmp's `(10·log10 P + 10)/50`) → fast-attack / slow-decay smoothing (0.6 / 0.25) → a silence gate that decays the bars ×0.8 per frame without an FFT. The same tap serves relays (MP3 → resampler → tap) and the synthesized broadcast. **Deliberate deviations**: (a) band edges are voice-weighted (50 Hz–9 kHz) — CLIAmp's music edges run to 20 kHz, where a 32 kbps NWR relay or a 22.05 kHz Piper voice has nothing, and four of ten bars would sit flat forever; (b) with one row there is no height to grade, so each band takes the tier of its own level (CLIAmp would paint a single row all green). **Wiring**: `modes/tty` stays domain-free — the app hands it a `Spectrum func() []float64` feed (`WithSpectrum`); a 50 ms `vizTick` runs ONLY while Viz is on and something plays (or the bars are still settling), one ticker however many status updates arrive; Viz off = no tick, no FFT. Cost: one FFT per frame on the update loop (19.7 µs, 1 alloc — `BenchmarkBands`) — the one radio hook allowed there, documented at the call site. Three theme tokens (`radio.spectrum.low/mid/high`) with Monochrome (greys) and Solarized (green/yellow/red of that palette) values. Pinned: tap ring semantics, engine tap + clear on Halt, FFT vs textbook DFT, tone lands in its band, smoothing constants, renderer widths/glyphs/gradient, dashboard tick lifecycle. |
| 92.2 | Reference reading | CLIAmp ships 30 modes (`VisBars … VisMirror`) behind a `v` cycle plus Lua plugins. Ours is the one "standard" mode; a `[v]` cycle (Classic Peak with falling caps, Columns, Wave/scope) is a clean extension if wanted — the feed and the frame stay as they are. |

### Session 93 (Repeat: Off | One | Watchlist; Pin retired)

| # | Item | Disposition |
|---|---|---|
| 93.1 | `[r] Repeat: Off \| One \| Watchlist` | **Done**: `[r]` cycles the three; the chip reads the mode (One/Watchlist in the yellow-bold "on" emphasis); the playing row wears ∞ for either. **Off** plays one broadcast cycle and stops (as before). **One** loops the tuned location (the old On). **Watchlist** plays the favourites in turn: the tuned location to the end of its synthesized cycle, then the next favourite, round the list — a location that is not a favourite (a RECENT row) hands off to the top of the list. The dashboard hands the player the favourites in order as the queue on every `[r]` and again whenever the watchlist changes under Watchlist mode (add/remove — an unchanged list sends nothing). The player reports the location it advanced to, so the ▶ row follows it. `tty.RepeatMode` is the one enum; `Radio.SetRepeat(mode, watchlist)`. **Assumption, flagged**: a live relay never ends, so under Watchlist a relayed station gets one NWR cycle — `liveDwell = 5 min` (the single knob in `app/radio.go`) — then the queue moves on; Synth advances at its own end. The advance runs off the engine goroutine (Halt inside Tune waits for it). Pinned: cycle + chip, queue = favourites in order, ▶ follows the reported location, re-send on watchlist change, none on no change, Off clears ∞, `nextInQueue` wrap/top/empty. |
| 93.2 | Remove `[p] Pin/Unpin` — Repeat effectively does that while navigating | **Done**: the binding, the chip, the state and the help entry are gone (Pin had been a label-only state since UAT 37; `[space]` always tuned the focused row). What Pin promised — "keep playing this while I look around" — is what a tuned station already does, and Repeat: One/Watchlist says how long. |

### Session 94 (voice change mid-broadcast — "how can I break it")

| # | Item | Disposition |
|---|---|---|
| 94.1 | Changing the voice mid-synth causes a long pause and resets the broadcast; it should hand over at the same spot, as fast as possible | **Done** — the old path was a full re-tune (re-resolve the county over the network, a new Source from segment 0, the first segment re-rendered before anything played). Now `Source.SetVoice` swaps the voice in place: the writer streams each segment in 100 ms chunks and, at the next chunk after a change, maps the time fraction of the line to its words, renders **only the remainder** (from the word reached) in the new voice, and continues there — the pause is one short render (≈ a second with `say`) after the ≤ 200–400 ms already buffered downstream; nothing restarts, nothing re-resolves. The look-ahead segment rendered by the old voice is re-voiced before it plays; the render cache starts over (it was the old voice's). The sign-off is planned with a `{{voice}}` token that the Source speaks (and shows) as the voice that reaches it, so a switch during the cycle signs off in the right name. **Buffer line (HUM LEAD)**: the new voice opens with "This is {new}, taking over for {old}." — spoken while the remainder renders in parallel, so the audible gap is one short render and the change reads as deliberate; re-selecting the same voice is a quiet no-op. The marquee follows: the full line, the hand-over line, then the remainder. **Race found and pinned**: a render that started before the switch could land old-voice audio in the freshly cleared cache (the tail then played in the old voice, 1 run in 10); renders now capture voice + generation up front and cache only if unchanged, and callers tag audio with the generation that produced it — `-count=40` clean. **Fallback**: a voice at a different sample rate cannot join a running stream (the engine fixed the rate at start) — `SetVoice` refuses and the deck does the old full re-tune; on macOS every `say` voice is 22.05 kHz, on Linux/Windows there is one Piper voice, so in practice the hand-over always applies. Under three words left = no hand-over; the next segment simply follows in the new voice. Pinned: `Remainder` word mapping, hand-over within two chunks with the same-spot remainder + re-voiced next segment + substituted tail + marquee sequence, rate refusal. |

### Session 95 (narration rules)

| # | Item | Disposition |
|---|---|---|
| 95.1 | Web addresses: `www.weather.com/word` → "w w w dot weather dot com slash word" | **Done** (voice-only, in `Pronounce` — the marquee keeps the address as written): a token that is a host with an alphabetic top-level label, optional scheme/www and path is spelled — scheme dropped, `www` letter by letter, dots / slashes / dashes / underscores by name. "3.5", "U.S.", "e.g." are not addresses (the last label must be ≥ 2 letters). Pinned. |
| 95.2 | Alert bullet labels are statements: the System Voice reads "What." as "What?" | **Done**: CAP bullets `* WHAT...Hot` now normalize to "What: Hot." — the colon makes the voice state the label; multi-word labels follow ("Additional details: …"); the rest of the line keeps its casing. Display and voice both carry the colon. Golden updated (UAT 82 compose) + pinned. |

### Session 96 (RECENT / SEARCHED persists)

| # | Item | Disposition |
|---|---|---|
| 96.1 | Save the RECENT / SEARCHED list across runs like the favourites: look up 92020, quit, relaunch — El Cajon is still on top. Stack semantics (newest on top, oldest off the bottom); add → bottom of favourites, remove → top of recents | **Done**: the dashboard already ran the stack (lookup → top, remove-from-watchlist → top, add → bottom of favourites, deduped by zip, capped at 50) and handed it to `Commit` on every change — the app persisted only the favourites. Now `config.Recent` (TOML `[[recent]]`, newest first) is written on every commit and restored at launch: the saved stack on top, the seed cities filling the room below, anything that became a favourite dropped, deduped, capped at 50. One conversion pair (`refsFromConfig` / `configLocations`) serves favourites and recents; `commit` is nil-safe for the recent pipeline. Pinned: config roundtrip with `recent`; restore order (saved above seeds, favourites excluded, cap); commit writes both lists in order. |

### Session 98 (wide-terminal synth pauses + CPU — measured, not guessed)

| # | Item | Disposition |
|---|---|---|
| 98.1 | "On VERY WIDE terminals the synth narrator has very long pauses and CPU ramps WAY UP; closing the app backs it down" | **Measured on the exit build** (real TUI in a pty, Synth playing Oceanside, 90 s each, `ps` on the app and its `say` child): 133 cols — app 0.4–3.6 % CPU (Viz off), 3–11 % (Viz on); 400 cols — 1–10 % / 5–11 %; `say` bursts 10–46 % while it renders a segment (that is macOS TTS itself); PLAYING throughout, no stall. Width costs a few percent of one core, not a ramp, and nothing in the synth path reads the terminal width (the marquee is display-only). **Two causes of exactly this symptom were removed at the exit** and were live in the build being used: a render failure reported as "broadcast complete" made Repeat: Watchlist re-tune the whole list several times a second (CPU ramp, no audio — round-1 F3/C-4), and overlapping starts could orphan a second engine stream (C-1). Disposition: fixed-by-other-findings, **re-test on this build**; if it recurs, launch with `WATCHPOST_DEBUG_PPROF=1` and capture `127.0.0.1:6060/debug/pprof/goroutine?debug=2`. |
| 98.2 | What the measurement did find | RSS climbed 79 → 113 MB (133 cols) and 88 → 131 MB (400 cols) in 90 s of synth: the rendered-audio cache held up to 64 **stereo** segments (~1.3 MB each). Now the cache holds **mono** (widened at write) and is capped at 24 — ~16 MB at the worst instead of ~85 MB under Repeat. Re-measured: 82 → 116 MB — the cache was ≤ 10 MB of the window (a cycle is ~8 segments in 90 s); ~20 MB is the audio context on first use; the rest is not isolated in 90 s and goes to the **1-hour soak at VALIDATE (M8)**. Numbers in the infra ledger. |
| 98.3 | Harness lesson | The first two wide-terminal runs "proved" the app froze in CONNECTING with 0 % CPU — it was the harness: `expect` only drains the pty inside an `expect` call, so a script that `sleep`s while the radio redraws fills the pty buffer and the app blocks on the terminal write (goroutine dump: bubbletea's renderer waiting on its output mutex). Real terminals always drain. Recorded in the debugging ledger. |

### Session 97 (`[m] Mode: Synth | Nearest Relay`)

| # | Item | Disposition |
|---|---|---|
| 97.1 | `[m] Mode: [Synth \| Nearest Relay]` — the explicit source pick queued since UAT 78 | **Done**: `[m]` flips the chip (after `[r] Repeat`; the value in the station tint) and pushes `Radio.SetMode`; a playing location re-tunes under the new mode at once. **Synth** (default, per UAT 78) voices the location's own products — always, even where a covering relay exists. **Nearest Relay** plays the first station the resolver lists with a mount: the covering transmitter when it is relayed, else the nearest relayed one (Victorville for Oceanside); none in reach → Synth with the reason line, and a relay that fails outright still fails over to Synth. The pick is persisted as `radio.mode = "synth" \| "relay"` (one `savePreference` path now serves the voice and the mode) and seeds the chip at launch; the help modal lists it. `tty.RadioMode` is the enum (`ParseRadioMode` / `Key` for the persisted form). **Behaviour change to note**: before, a *covering* relayed transmitter played live by default; now the default is Synth everywhere and live is a deliberate pick. Pinned: chip seed/flip/push + help; `chooseNearest` order (covering first, nearest next, no mount = none); config roundtrip. |

### Session 99 (B5 — fire: HMS / WFIGS / FIRMS)

| # | Item | Disposition |
|---|---|---|
| 99.1 | "Build Exit NOT Approved; we need to include initial FIRMS (fire data) in 0.9.0" — keep HMS/WFIGS; fire is another alert type with a section in the location detail; thresholds configurable | **Done** (data first, then UI): `domains/fire` (rules + pure geometry), providers `hms` (keyless, one 1.4 MB KMZ per 10 min for the continent, ~25k analyst-curated points), `wfigs` (keyless, NIFC incidents, one national query), `firms` (NASA VIIRS NRT, only with a MAP_KEY — registered only when keyed so the API status never says "ok" for nothing). Assembler merges per provider (`mergeFire`: cross-provider hotspot dedupe, incidents by name). `[fire]` config: `radius_km` 25, `incident_radius_km` 50, `min_frp_mw` 5, `bold_frp_mw` 50, `min_confidence` "nominal" — one owner (`app.fireRules`), the bold threshold handed to the dashboard as `Config.FireBoldMW`. Pinned: rules/cluster/bounds, each provider over in-test fixtures (a KMZ built in the test), assembler merge, config defaults/overrides, provider registration by key. |
| 99.2 | UX: the row mark | **Done**: `▲` in its own marks slot (`render.FireMark`, orange 208; Monochrome 255, Solarized 166), bold when any hotspot reads ≥ `bold_frp_mw`; `^` under `--ascii`; never displaces `▶`/`∞`/`⚠`. Pinned. |
| 99.3 | UX: the FIRE section in the detail modal | **Done**: between MARITIME and the alert blocks, always present — `Hotspots  N hotspots nearby` (or `none within the fire ring`), then up to three hotspots nearest first: `▲ 6 mi N    62 MW · GOES-WEST      2h 00m` (bearing via the 16-point compass the swell rows use; strength bold at the threshold; age via `fixedAge`), `… and N more`; incidents `Timber      19 mi · 12,915 ac      26% contained`; `Fire Wx       Red Flag Warning` when the alerts carry a Red Flag / Fire Weather event. Pinned (colour on: bold at 62 MW, plain at 8 MW; the no-fire line). |
| 99.4 | Report and JSON | **Done**: `fire: 1 hotspot(s) nearby — nearest 8.2 km, 62.4 MW, hms/GOES-WEST at Aug 23 21:30 UTC` / `fire: no hotspots nearby`; `incident: Timber — 12915.0 acres, 26.0% contained, 30.5 km (Aug 20)`; `--json` carries `fire`. M5 parity fixtures extended (every rendered number is a JSON leaf). `RenderPlain` split (`alertLines`, `fireLines`) for P10-04. |
| 99.5 | Live smoke (2026-08-25, `watchpost report 92057 --report-only`) | hms ok · wfigs ok · firms not listed (no key). Oceanside: `fire: no hotspots nearby`; `incident: Mission — n/a acres, n/a% contained, 17.4 km (Aug 22)`, `incident: Convoy — … 47.0 km (Aug 22)`. The `n/a` acres were checked against the service: those two young incidents carry no IncidentSize, FinalAcres, DiscoveryAcres or InitialResponseAcres yet — `n/a` is honest; the provider now falls back through the four acreage fields so the first reported size shows. |
| 99.6 | FIRMS key — how to get one and test | README "Fire": register at <https://firms.modaps.eosdis.nasa.gov/api/map_key/> (email; key arrives at once), paste into the Setup window — `s` (masked input, `ctrl+r` reveals; UAT 100) — or `[providers.firms] key = "…"`. Test: `watchpost report <a place near an active fire> --report-only` — `provider firms: ok` appears and hotspots read `firms/N20` or `firms/N21`; the `--json` `fire.hotspots[].source.provider` says which source each point came from. Quota 5,000 requests / 10 min; Watchpost makes 2 per location per 10 min. |
| 99.7 | Credits | About window lists NOAA HMS, NIFC WFIGS, NASA FIRMS (≤ 52 cells each — the first cut of the HMS line was 65 and failed the About-window test). |

### Session 100 (Setup is a window; `[s] Setup`)

| # | Item | Disposition |
|---|---|---|
| 100.1 | "Clean this view up so it goes to the dashboard, immediately opens a setup modal (like all the others), asks the questions; no → default data set, no key; user can invoke setup at any time" | **Done**: the standalone wizard (B2 `SetupModel`, its own `tea.Program`, stderr voice install) is gone. `watchpost` with no locations, `watchpost setup`, or a first run opens the **Setup** window over the dashboard; `tty.Config.OpenSetup` carries the intent. Two questions: **1.** default location — type-ahead through the new `Suggest` hook (embedded index only, never the network per keystroke), `↑↓` pick, `enter` takes the pick or resolves the typed text through `Resolve` (mode "setup"); **2.** "Add a NASA FIRMS key?" — `y` opens a masked key line (`ctrl+r` reveals), `n` / `enter` finishes with the default data set and no key. Finishing runs ONE command: the `Setup` hook (persist: default first, duplicates dropped, other customizations untouched — the B2 red-team #2 rule kept; the key stored when given) then `Commit` (the default on top of the watchlist, the rest kept) — so the two config writes never race. A refused key or a failed save keeps the window open with the reason; `esc` closes without saving (the dashboard has no rows until a location is chosen; `s` reopens). Focus lands on the new default. Pinned: open on `s` / at launch / esc; the no-key flow (hook once, empty key, default leads the watchlist); the key flow (masked, revealed, stored; a refused key stays open with the reason); `applySetup` re-run preserves customizations and drops the duplicate; malformed keys refused without echo. |
| 100.2 | Key binding | HUM LEAD: "if ctrl+shift+s is not possible, [s] is — which might be better anyway as '[s] Setup'". `s` it is (`S` stays API Status): most terminals send the same byte for `ctrl+s` and `ctrl+shift+s`, and some swallow it as flow control. Listed in `?` help. |
| 100.3 | FIRMS turns on without a relaunch | The provider is always registered; `firms.SetKey` (locked, read per fetch) keys it live from the Setup hook; the assembler's new `SetInactive` publishes `off` for an unkeyed FIRMS so the API status never says "ok" for a feed that contributes nothing (`—` glyph, neutral tone, never exit 2). Pinned. |
| 100.4 | Red-team B5 F1 (security lens) folded in | A MAP_KEY is 32 hex characters — the shape httpx redacts. `firms.CheckKey` refuses anything else before it is written or used (the reason never echoes the value), and FIRMS scrubs the key from any transport error text itself, so redaction no longer depends on one layer. |

### Session 101 (second BUILD-exit pass — B5 remediation; Location Details chips)

| # | Item | Disposition |
|---|---|---|
| 101.1 | Location Details: "consolidate chip controls — [↑↓] Scroll [esc] Close [ctrl+a] + Watchlist [shift+del] − Watchlist; ctrl+a / shift+del enabled or disabled by watchlist membership" | **Done**: one chip row — `[↑↓] Scroll  [esc] Close  [ctrl+a] + Watchlist  [shift+del] − Watchlist`; `+` reads enabled when the location is not a favourite and the list has room (`canAddFocused`), `−` when it is one (`canRemoveFocused`; shift+del then opens the Remove confirmation over the details, as it already did from the table). Anatomy test updated. |
| 101.2 | Red-team round 3 (four lenses over `470bd08`): security 7, concurrency/perf 11, UX/parity 12, docs/hygiene 10 — 40 findings, no blocker | Dispositions in the report addendum (`08-reports/red-team-build.md`, Round 3). Fixed in this session: FIRMS key shape validated + scrubbed from any error (S1); HMS cap counts placemarks and reports truncation (S2/P5); `[fire]` validated at load, case-folded, rings bounded (S3/S4); KMZ inflate budget + file cap (S5); non-KML refused, bad placemarks skipped (S6/S7); HMS parse memoized by content hash — one parse per archive change instead of one per RECENT scheduler (P1: 4.5 GB allocated per 15-min tick → ~90 MB; P7 falls out); a failed FIRMS source leaves the location unserved so it retries and keeps prior data (P2); `FireState.as_of` — "fire feed not yet available" / "fire: feed unavailable" before any feed answers, never "none" (P3); FIRMS 400/401/403 → "FIRMS rejected the MAP_KEY — open Setup ([s])", once per cycle (P4); a body that does not parse is forgotten from the cache (P6, `httpx.Forget`); `n/a MW` so an unmeasured GOES point never collides with the age column (U1); zero times read `age n/a` / `n/a` (U2); ▲ moved to marks slot 1 so the gutter before `###.` stays (U3 — HUM LEAD may flip it back at UAT); ASCII separators in the FIRE rows (U4 partial); ellipsised incident names (U5); a trailing space guards the 3-letter bearing at ≥ 100 km (U6); FIRMS satellite codes spelled like HMS (U7); a row-marks legend in `?` (U8); High Contrast `FireMark` 214 (U11); analyst-curated HMS points outrank every `min_confidence` (D3); `docs/extending.md` walkthrough 2 rewritten to the shipped keyed-provider pattern (D4); plan of record reconciled (§11.9 row, tier diagram, §11.10 placed after §11.9, G-10/R2-25 closed, CHANGELOG intro, `ctrl+r`, `finalize` godoc, setup help string, credits "needs key") (D1/D2/D5/D6/D7/D9). Per-location hotspots capped at 300 nearest (`snapshot.MaxHotspots`; the count reads "300+") after the Ross fire clustered into 941. |
| 101.3 | Declined / deferred with rationale | U4 `--ascii` never reaches the dashboard (pre-existing; the B6 `--ascii` flag is the 1.0 item that wires it — the render path is pinned meanwhile); U9 "hot" is bold-only in the row (the fact "fire nearby" is the glyph itself and the MW figure is in the modal — R-12a met; a count in the row is a design call for HUM LEAD); D8 `report` now fails on a corrupt config (it names the file and the fault — hiding a broken `[fire]` table or key from the report would be worse); D10 covered by the legend (U8). |
| 101.4 | Gates at close | `make verify` GREEN (fmt, vet, `-race`, lint-imports, lint-watermark + controls) · P10 0 live (5 density exemptions recorded for the fire packages and config, ratify at gate) · golangci-lint 0 · staticcheck 0 · `a2dh validate` 18/18 · live: Mineral Wells, TX — 300+ hotspots, Ross 50,000 ac 5 %, HMS/WFIGS/FIRMS ok with HUM LEAD's key. |

### Session 102 (masthead for narrow terminals and N APIs)

| # | Item | Disposition |
|---|---|---|
| 102.1 | Mock: `W A T C H P O S T  <version>   Updated: 12/31/2026 23:59:59 PDT   API: ✔8 ⚠0 ✘0 / 8  [S] Status` over `[s] Setup  [a] About  [t] Theme  [?] Help  [q] Quit`; stamp centred between title and API block; reserve 2 columns for the total | **Done**: line 1 = title · centred `Updated:` stamp · `API: ✔n ⚠n ✘n /%3d` + `[S] Status`; line 2 = the five chips in the mock's order. Counts: ✔ ok · ⚠ degraded but has served (stale) · ✘ degraded and never served (down); `off` providers (unkeyed FIRMS) count in none and not in the total (the [S] window lists them). The eight-glyph strip (`NWS ✔  NDBC ✔ …`) is gone from the masthead — it never fit a narrow terminal and no longer scales. Narrow widths: the stamp shortens (time only → no label → gone), then the API block drops "Status", then "API:" — the header never exceeds the width (pinned at 133/100/80/60). |

### Session 103 (About window spacing)

| # | Item | Disposition |
|---|---|---|
| 103.1 | "Add 1 blank line between 'NWR Radio' and 'Relayed audio lags' to improve readability" | **Done**: a spacer entry in the credits list (`app/credits.go`) between the wxradio.org / weatherUSA relay credit and the relays' condition-of-use line; About renders it as a blank row inside the inset. |

### Session 104 (empty states)

| # | Item | Disposition |
|---|---|---|
| 104.1 | Watchlist empty state: 3–5 rows, centred, "Run 's' Setup, 'l'ookup a location, or 'ctrl+a' a searched location to add to your Watchlist"; replaced by the table once a location is added | **Done**: `emptyState` — a blank, the message centred on the table span and wrapped at the mock's 64-cell measure (two lines on wide terminals, exactly the mock's break), a blank — in place of the table (no headers, no "Showing" line). Narrow terminals wrap inside the width, never truncate. The one-line "No locations yet — run 'watchpost setup'…" is gone. |
| 104.2 | Recent / Searched empty state: "NO RECENT LOCATION SEARCHED or DATA-SEEDING FAILED" — a fallback users should not normally see | **Done**: same block under the section band; the rail's ▼ rides its last row. |
| 104.3 | Observation while pinning at 60 cols | The radio panel's VOL line is 61 cells at a 60-column terminal (pre-existing; the supported floor is wider — see the term breakpoints). Noted for the narrow-width sweep, not fixed here. |

### Session 105 (Synthwave '84 theme)

| # | Item | Disposition |
|---|---|---|
| 105.1 | Add a "Synthwave '84" theme (reference: robb0wen/synthwave-vscode) | **Done**: registered as a built-in (`t` chooser lists it). Palette from the extension's colour theme: window `#262335`, tiles `#241b2f`/`#34294f`, neon pink `#ff7edb` (focus name, station, spectrum mid), cyan `#36f9f6` (accent, viz, spectrum high), yellow `#fede5d` (advisory, repeat), orange `#ff8b39` (highs, fire mark, trend up), mint `#72f1b8` (ok, playing, spectrum low), red `#fe4450` (warnings, down), lavender `#848bbd` (body text, muted). Title gradient pink → cyan → yellow. Foregrounds use the nearest 256-palette codes (the Tint path), tiles truecolor (the raw path), like the other built-ins. |

### Session 106 (promote, don't copy)

| # | Item | Disposition |
|---|---|---|
| 106.1 | "Moving a location from recent → watchlist shows it in both places — redundant data. It should leave RECENT at whatever position it was in, later entries move up one, the recent total drops by one while the watchlist grows by one, so the scheduled payload does not change" | **Done**: both add paths — `ctrl+a` in Location Details (`addFocused`) and the Add modal's resolve (`handleResolved` "add") — now commit the watchlist with the location appended **and** the RECENT list with it removed (`withoutRef`, by location key or zip, order kept). The app persists both lists as given, so the RECENT stack shrinks by one on disk too; at relaunch `restoreRecent` already excluded favourites. Pinned on both paths. |

### Session 107 (title gradient follows the theme)

| # | Item | Disposition |
|---|---|---|
| 107.1 | "I really like how the WATCHPOST gradient matches the theme — do that for all themes and any new ones going forward" | **Rule**: every theme sets its own `title.grad.start/mid/end`. Monochrome (white → grey), Solarized Night (magenta → blue → cyan) and Synthwave '84 (pink → cyan → yellow) already did; High Contrast now has white → yellow → cyan (its focus and low-temperature colours) instead of inheriting the default. Pinned by `TestEveryThemeOwnsItsTitleGradient`: a new built-in that forgets the stops fails the test, not the UAT. User theme files may set the same three tokens. |

### Session 108 (High Contrast chips)

| # | Item | Disposition |
|---|---|---|
| 108.1 | "High Contrast — button controls are unreadable: light chip background, text still white. Chip text should be black" | **Root cause, fixed**: the chip token was already black-on-light (`1;16;48;5;252`), but chips ride the raw SGR path where a bare palette index (`16`) is not a colour — the terminal ignored it and the text stayed white. Now `1;38;5;16;48;5;252`. The same latent fault sat in Monochrome's and Synthwave's flash chips and Synthwave's key chip (bare `16` / `255` / `231`); all qualified. Pinned: `TestCompositeThemeTokensAreLegalRawSGR` — every composite colour token in every theme must be a legal raw SGR list, so a future theme cannot reintroduce it. |

### Session 109 (Remove Location modal tone)

| # | Item | Disposition |
|---|---|---|
| 109.1 | "Watchpost default theme — the Remove Location modal background is too light under the light text; change it to #4F0C0C" | **Done**: `confirm.bg` = `48;2;79;12;12` (#4F0C0C) in the default table (was #AE7D7E from UAT 26.2). Synthwave keeps its own `#782850`; the other built-ins inherit the new default. |

### Session 110 (fire mark: counted ◆, wider marks block)

| # | Item | Disposition |
|---|---|---|
| 110.1 | "▲ looks too cluttered — diamond is our fire symbol, orange, with a counter when there are multiple hotspots/evac orders; add spacer columns": mock `›  ▶ 3◆ 2⚠ 001.` | **Done**: the marks block is 11 cells — pointer · two spacers · ▶/∞ · spacer · `n◆` · spacer · `n⚠` · spacer — then `###.`. ◆ is orange (`fire.mark`), bold when a hotspot burns at or above `bold_frp_mw`; `*` under `--ascii`. The count is the named incidents nearby, or 1 for unnamed hotspots (capped at 9 like ⚠). The 5 extra cells come out of NAME's floor (24 → 19 at the minimum widths; NAME fills again from 120 cols), so `###.`@11, NAME@16 and **every column from LABEL on keeps its mock offset** and every breakpoint holds. Mock lines 27–29 of `09-view-mocks/watchpost-cli-view-mocks-with-notes.txt` updated to the new block (the fidelity test diffs against them). At 50 cols NAME reaches its 10-cell floor one step earlier ("Oceansi…"; full from 55). Help legend reads `n◆ fires nearby`. Pinned. |
| 110.2 | "Move the play char 1 col over: `›  ▶ 3◆ 2⚠ 001.` — 2 cols between pointer and play, 1 between play and the fire count" | **Done**: play/∞ in slot 3 (was 2); block width unchanged. Mock row line updated; pinned. |

### Session 111 (Setup shows the stored FIRMS key)

| # | Item | Disposition |
|---|---|---|
| 111.1 | "After first setup the FIRMS key question disappears. It should show that I've added a key and let me re-enter / add a different one if it expires or is corrupted — today the only way to verify it is the FIRMS status" | **Done**: question 2 now reads `NASA FIRMS key: stored (…cdef) — working` (the key's last four characters via `firms.KeyHint`, never more; health from the live snapshot: **working** / **rejected — paste it again** (the provider's own rejection warning) / **degraded (see [S] Status)** / **not active** / **no report yet**), with `[y] Replace the key  [n] Keep it`. `n` finishes with an empty key, which never overwrites a stored one. Without a key the question reads as before. Pinned (stored + working; rejected wording; n keeps). |
| 111.2 | "I don't see it in the setup modal" (build 1b832bf) — the stored-key line sat behind question 1, so a re-run looked unchanged until a location was re-entered | **Done**: on a re-run the first screen shows `Current: Oceanside, CA (92057) — [enter] keeps it, or type a new one` and the stored key's line (`2. NASA FIRMS key: stored (…cdef) — working`) beneath; a bare `enter` keeps the default and moves to the key question. Pinned. |
| 111.3 | "Make all questions visible at once with key navigation between them — the modal didn't make it clear it was multi-step" | **Done**: Setup is one form. Both questions are always on screen, the focused one marked `›`: **1.** default location (`Current: … — [enter] keeps it` on a re-run; type to search, `↑↓` pick, `enter` accepts and moves the focus on); **2.** NASA FIRMS key — `stored (…cdef) — working / rejected — replace it / degraded / not active` with `Paste a new key to replace it — empty keeps it`, or, with no key, the free-key address and `Empty = the default data set, no key`; one masked `Key:` line (`ctrl+r` reveals). `tab` / `shift+tab` move between the questions; `enter` on question 2 saves; saving without a location on a first run returns the focus to question 1 with the reason; `esc` cancels. The y/n step is gone — an empty key *is* "no". Lines fit the 68-cell window without wrapping. Pinned (form anatomy, tab both ways, pick → focus, save, masked/reveal, refused key stays open, stored key + health, rejected wording). |
| 111.4 | "Setup window — controls aren't wrapping within the inset; [ctrl+r] runs up to the edge" | **Done**: the chip row wraps by chip (`WrapSegments`, the radio controls' path) inside the window's inset instead of word-wrapping through the modal body — a chip never splits and nothing reaches the edge. Pinned (every setup line inside the inset; `[ctrl+r] Reveal key` intact). |
| 111.5 | "Make the questions white (not muted); 'working' green, 'rejected' red, with a check / cross for a11y" | **Done**: question titles in `text.bright`; health reads `✔ working` (`provider.ok`, green) / `✘ rejected — replace it` and `✘ degraded (see [S] Status)` (`provider.down`, red) — the glyph carries the state without colour (R-12a). Pinned. |

### Session 112 (broadcast lead points listeners to the live NWR transmitter)

| # | Item | Disposition |
|---|---|---|
| 112.1 | HUM LEAD script for the lead: "This is Watchpost Weather Radio serving <location>. A version of this forecast is also broadcast live from <STATION ID>, <location> and is accessible via NOAA radio devices and receivers. Watchpost Weather Radio forecasts may be delayed and are not intended for life safety use. This forecast is from the National Oceanic and Atmospheric Administration and is for <Day, Date> until <Day, Date>." | **Done**: `synth.Lead(location, station, now)` reads the script verbatim; the station is the transmitter covering the location's county (any status — the point is a radio, not a relay), else the nearest (`Resolver.NearestTransmitter`); with no table the live sentence is left out rather than pointed at nothing. Callsigns are spelled for the voice ("K E C six two"; screen keeps `KEC62`) — a `Pronounce` rule like web addresses. "life-saving purposes" → "life safety use" to match the About disclaimer (HUM LEAD). Goldens updated; pinned (lead with and without a station; callsign spelling). |
| 112.2 | "Add the station frequency after <location> → 'broadcasting on <frequency>'; 165.2024 MHz is read 'one-six-five dot two-zero-two-four mega hertz' (the marquee shows the number)" | **Done**: "…from KEC62, San Diego, California broadcasting on 162.400 MHz and is accessible…"; `Pronounce` reads a decimal figure digit by digit with "dot" and "mega hertz" only when "MHz" follows ("3.5 inches" stays a number). Golden + pinned. |
| 112.3 | "Add a 2-second pause between 'safety use.' and 'This forecast is from…'" | **Done**: `Segment.Pause` — extra silence the source writes after a segment, beyond the standard 400 ms gap; the lead is two segments (the notice, with a 2 s pause; the span). Pinned. |

### Session 113 (narration rule: NOAA)

| # | Item | Disposition |
|---|---|---|
| 113.1 | "'NOAA' reads as 'NO-AH', not 'en oh double-a'" | **Done**: pronunciation-table entry `NOAA → NO-AH` (voice only; the marquee keeps NOAA). Pinned. |
| 113.2 | "'NO-AH' clips mid-syllable — should be 'NOAH' / 'NOWAH'" | **Done**: the table entry is now `Noah` (a word the voices already know; no hyphen to break on). Pinned. |

### Session 114 (the Fire and Hotspot report on air)

| # | Item | Disposition |
|---|---|---|
| 114.1 | HUM LEAD script: after the NOAA forecast, before the tail — "This is the Watchpost Fire and Hotspot report for <location>. This report is derived from data from <feeds, with 'and'>. Data for this report may be delayed or incomplete, and is not intended for life safety use." · 2 s pause · "There are currently {no \| N} hotspot(s) within a <ring> fire ring in your area." · per named fire: distance/direction, size, active-for, percent contained · "Nearby fires outside of your fire ring that may be worth noting are: <name> at a distance of <distance>…" — phrases omitted when the data is missing; skipped entirely when there is no fire data | **Done**: `synth.FireSegments` / `FireReport` (state, rings, spoken feed names, location) composed into the cycle by `Compose` between the products and the tail; the deck's `fire` hook reads the location's fire state from the live pipelines (`livePipelines.fireFor` → `fireReportFrom`, favourites first, then RECENT). Feeds are named as spoken — "NOAA's Hazard Mapping System, the National Interagency Fire Center, and NASA FIRMS" (FIRMS only when keyed). Distances in the display units, whole numbers; directions in words on the 16-point compass ("west-northwest"); durations "5 hours and 35 minutes" / "3 days and 4 hours"; acres grouped ("12,915 acres"); the strongest satellite hotspot gets its own sentence ("…with a fire radiative power of 62 megawatts, detected 2 hours ago by GOES-West"). Not Known (no feed has answered) → no segments, straight to the tail (HUM LEAD). Pinned: the full script over a two-hotspot / two-incident fixture, the quiet metric case, the skip, the word helpers, the snapshot → report mapping. |
| 114.2 | Script notes | "WeatherWatch" in the script read as a slip for **Watchpost** (every other line of the broadcast says Watchpost) — one word to change if it was meant. "<size> (15MW)": for satellite hotspots the figure is fire radiative power in megawatts and is read as such; for named incidents the size is acres. "It is currently moving <direction>" needs two observations of the same fire over time, which no feed gives us — the phrase is omitted, as the script allows; a spread direction from HMS/FIRMS deltas is a 1.0 candidate. |
| 114.3 | HUM LEAD ratified the three calls in 114.2 (Watchpost, not WeatherWatch; MW is fire radiative power, acres for incidents; movement omitted until a feed can give it) | Recorded. |

### Session 115 (air between reports; NOAA's)

| # | Item | Disposition |
|---|---|---|
| 115.1 | "NOAA mispronounced in the fire report" | The possessive "NOAA's" slipped past the word table. **Done**: a possessive keeps its base word's pronunciation ("Noah's Hazard Mapping System"). Pinned. |
| 115.2 | "2 s between any/all reports (NOAA → fire → future), and a special 1 s between the last report and the tail — it runs straight into the fire report and feels jarring" | **Done**: `Compose` gives the last segment before a following report a 2 s pause (`reportPause`) and the last segment before the tail 1 s (`tailPause`) — so forecast (2 s) fire (1 s) tail, or forecast (1 s) tail when there is no fire report; a future report slots in the same way. Pinned. |

### Session 116 (REVIEW — docs-vs-code lens)

| # | Item | Disposition |
|---|---|---|
| 116.1 | README/CHANGELOG still described Setup's `y`/`n` step (a first-run user typing `y` would land it in the key line) | **Fixed**: both rewritten to the form (paste or leave empty; `tab`; `enter` saves; stored key shown). |
| 116.2 | "Piper, installed by `watchpost setup`" (README) | **Fixed**: installed the first time you tune in. |
| 116.3 | `go run ./tools/nwrtable` in extending.md did not execute (needs `-in`) | **Fixed**: the curl + `-in CCL.js` form. |
| 116.4 | Fire report: named fires had no direction — `Incident` carried no point | **Fixed**: `Incident.Lat/Lon` from WFIGS geometry (schema addition at `-rc`); "Timber is 12 miles east of your location…", "Convoy, at a distance of 29 miles to the northeast…"; omitted when the feed gave no point. |
| 116.5 | CHANGELOG session count; shipped-but-unlisted items (stored-key display, themes, empty states, masthead) | **Fixed**. |
| 116.6 | `°` vs `º` in the README key table | **Fixed** to the app's `º`. |
| 116.7 | REVIEW code lens C1 (MEDIUM): fire segments were cached by position — under repeat, a changed count would replay yesterday's audio | **Fixed**: segment keys are content digests (`fire:<sha>`); the notice too (its feed list can change when a key is added). |
| 116.8 | C2: the deck cloned whole pipeline snapshots (up to 50 locations) to read one fire state | **Fixed**: `Assembler.FireFor` / `ProviderStatus` narrow reads under the lock; `fireReportOf` is the pure mapping. Pinned. |
| 116.9 | C3: Setup Q1 — bare enter after a hint was chosen replaced the choice with the old default (or errored on a first run) | **Fixed**: a chosen location moves the focus on. |
| 116.10 | C4: the fire notice credited NASA FIRMS whenever a key was stored, even rejected | **Fixed**: credited only when FIRMS answered ok. Pinned. |
| 116.11 | C5: masthead counted never-answered providers as ✔ on the first frames | **Fixed**: pending providers sit in the total only (a shortfall reads "still loading"). |
| 116.12 | C6: the audio cache (24) was smaller than a busy cycle since the lead split and the fire report | **Fixed**: 40. C7 (NAME floor 19 on 115–119 cols) accepted at UAT 110; C8 (user theme files not validated for raw-SGR shape) deferred to 1.0. |
| 116.13 | REVIEW contract lens M1 (HIGH): the live `--json` failed its own schema — per-provider `marine.tides/currents` published `null` | **Fixed**: `normalizeMarine` runs on every `by_provider` copy; the schema envelope test now applies a marine fragment. |
| 116.14 | M2: checked-in schema stale and its `$id` pointed at a dead URL | **Fixed**: `$id` is the raw path of the checked-in file, regenerated (`make schema`), and `TestPublishedSchemaMatchesGenerator` keeps it byte-identical. |
| 116.15 | M3: `providers[].status`, `warnings[].code`, radio `source`/`status`, provider `role` were free strings to JSON consumers | **Fixed**: `enum:"…"` struct tags emitted into the schema; README documents the status set. |
| 116.16 | M4: exit codes undocumented in README; §10.2 said "≠ ok" while `off` is 0 by design | **Fixed**: README `--json` paragraph; §10.2 reworded. |
| 116.17 | M5–M8, M13: "replace directive" wording (README, release.yml); `$GOMODCACHE` for two machine paths in a spike; one employer name in a quoted D-27 row; the personal email in the brief; CI double-run | **Fixed** (concurrency group for CI). M10: CC BY 4.0 and FIRMS URLs added to the README credits. M9 (licence file over-includes test-only modules) and M11 (schema `-rc` ratification) recorded, not changed. |
| 116.18 | REVIEW release lens R1 (MEDIUM): the Linux validation protocol still described the old wizard, a token-gated private repo and a setup-time voice install | **Fixed**: rewritten for the Setup window, first-tune Piper install (paths, sizes, ceiling), the masthead, the fire mark and report, exit-code checks, the no-ASCII-fallback note. |
| 116.19 | R2–R7, R13: "replace directive" in the checklist; `WATCHPOST_VERSION=0.9.0` 404'd without the `v`; no HTTPS/TLS pin on the download; an empty release could publish; PATH advice missed the Debian re-login path and an unset `HOME`; a binary that cannot start (musl) reported as installed; a cosmetic double space | **Fixed** in `scripts/install.sh` and `release.yml` (`fail_on_unmatched_files`, release notes, 20-minute timeout). R8/R9/R10–R12 confirmed (version reads `0.9.0`; no `--ascii` flag — the protocol says so; workflow, Piper pins and tzdata verified). |
