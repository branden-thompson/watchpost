# Where things happen

A flow map for someone reading the code for the first time: the event on the left, the function
that handles it on the right. Every symbol named here — in a file (**file.go:Func**) or in a package
(**pkg:Func**) — is checked by `cmd/watchpost`'s `TestWhereThingsHappenNamesRealSymbols`, so this
page cannot drift silently. Until the 0.14.0 BUILD-exit red team the check required the `.go`, so
12 of the 163 refs were invisible to it, including both rows this release rewrote; the promise in
this paragraph is now true. The per-file headers say what a file holds; this page says where a
*thing* happens.

| Event | Where |
|---|---|
| The program starts | `app/dashboard.go:RunDashboard` (composition root) → `app/pipelines.go:startPriority`, `app/pipelines.go:startRecent` |
| A key is pressed | `modes/tty/dashboard.go:handleKey` → a modal opens via `modes/tty/dashboard.go:toggleModal`; the radio keys via `modes/tty/radio_panel.go:toggleRadio` |
| A window opens or closes | `modes/tty/dashboard.go:toggleModal` → `modes/tty/dashboard.go:open` / `modes/tty/dashboard.go:close` (one `modal` value, so opening one closes the rest); drawn by `modes/tty/view.go:renderModal` through the modal memo `modes/tty/memo.go:modalView` (0.13.0: one render per input change, keyed by `modes/tty/memo.go:modalKeyFor`) |
| `enter` opens Location Details | `modes/tty/dashboard.go:toggleModal` → the body `modes/tty/detail.go:detailLines` (+ `modes/tty/detail_fire.go:fireRows`, `modes/tty/detail_marine.go:maritimeRows`) |
| A word is pronounced | the voice-only pass `domains/radio/synth/normalize.go:Pronounce` (every Say) and the product normaliser `domains/radio/synth/normalize.go:Normalize` load their tables by name from `domains/radio/pronounce/pronounce.go:Table` — `rules/<table>.txt`, one rule per line |
| A report's wording is chosen | by file name in `domains/radio/script/script.go:Text` — `scripts/<report>/<part>.txt`, `global/` lending a head or tail a report lacks, the same tree under the config dir's `scripts/` winning (`app/dashboard.go:scriptsDir`); the event read composes its phrases in `app/severe_read.go:eventScript`, the takeover its lines in `app/burst_words.go:breakingLine` |
| Something is spoken over the radio | every narration runs through the arbiter `app/director.go:Run` with a class — a breaking takeover (the Director's `Speak` effect, performed by `app/executors.go:speak`, highest: it pauses a read on air, which resumes after it — the ticker only PRODUCES the arrivals now, `app/ticker.go:startTakeover`) or an event read (`app/severe_read.go:Read` from `[space]` in the window) — which ducks the broadcast once, serialises the sequences by class and arrival, SUSPENDS a lower one under a higher (its line pauses through `domains/radio/player/engine.go:PausePreview`, its holds stop counting, a line it is rendering waits — `app/director.go:awaitAir`) and resumes it after (`domains/radio/player/engine.go:ResumePreview`), and restores the broadcast when nothing waits or is suspended; the radio deck is the voice (`app/radio.go:render` then `app/radio.go:play`); which narrations the visualizer follows is `app/director.go:vizFor` (every one but a takeover — those play `domains/radio/player/engine.go:PreviewAside`, off the tap) |
| A lookup opens Details before its data lands | the dashboard remembers the lookup (`modes/tty/modal_location.go:handleResolved` sets it) and `modes/tty/nav.go:selectedLocation` answers with an empty record in its name until `modes/tty/dashboard.go:applyRecent` finds the row by identity (`modes/tty/nav.go:lookupIndex`); any other window drops the wait (`modes/tty/dashboard.go:open`) |
| The radio diagnostic is written | `app/radio.go:debugLog` (`WATCHPOST_DEBUG_RADIO=1`, or a lower-case NAME — never a path, `app/radio.go:radioDebugName`): engine statuses (`app/radio.go:logStatus`), every synthesized segment as it reaches the air, why a cycle ended (`app/radio.go:cycleEnded`), and a read ended for not finishing.  It lands under the cache root at `debug/<name>.log`, 0600, rotated once past 8 MiB (`app/radio.go:writeRadioDebug`) |
| A severe event is listed | the app joins the ticker's feed events and the tracked locations' alerts in `app/severe.go:publish` → `domains/severe/severe.go:Union` (one row per event key, the location record winning) → `domains/severe/record.go:RecordOf` (the [A]-shaped record); the window draws the open category through `modes/tty/severe.go:severeBrowseLines` on `platform/render/severe_table.go:SevereTable` with the rail from `platform/render/severe_table.go:Railify` |
| A frame is built | `modes/tty/view.go:View` → geometry once `modes/tty/layout.go:layout` → `modes/tty/body.go:body` → the tables from the memo `modes/tty/memo.go:tables` (rendered on a miss by `modes/tty/body.go:priorityTable` / `modes/tty/body.go:recentSection` → `platform/render/table.go:LocationTable`); modal geometry `modes/tty/view.go:modalWidth`, overlay `platform/render/panel.go:Overlay` |
| The animation tick runs | only while `modes/tty/dashboard.go:tickNeeded` holds (a loading row, a volume blink, the marquee, `[S]`, Details); armed after every Update by `modes/tty/dashboard.go:armTick`, advanced by `modes/tty/dashboard.go:applyTick` |
| A snapshot arrives | `app/pipelines.go:Trigger` (coalesced: 50 ms for the favourites, 5 s for RECENT) → `modes/tty/dashboard.go:applySnapshot` / `modes/tty/dashboard.go:applyRecent` |
| A provider fetches | `platform/sched/sched.go:runTier` (a fixed grid from start: start, +Every, +2·Every …) → `Provider.Fetch` (e.g. `domains/weather/nws/provider.go:Fetch`) → `platform/snapshot/assembler.go:Apply` |
| A fire archive is parsed | once per body change through `domains/fire/memo.go:Get` (HMS: `domains/fire/hms/hms.go:parseKMLReader`, a streaming walk; WFIGS: `domains/fire/wfigs/wfigs.go:decodeLayer`) |
| A request is retried | `platform/httpx/httpx.go:attemptOnce` (one client retry, `Config.MaxRetries`) and `platform/sched/sched.go:fetchWithRetries` (10/20/40 s) — two layers, by design |
| A host is avoided after failures | `platform/httpx/memo.go:noteFailure` arms; `platform/httpx/httpx.go:memoRefusal` consults (normal lane only) |
| A cache miss is revalidated | `platform/httpx/httpx.go:getOrRevalidate` — the stored validators go out as `If-Modified-Since` / `If-None-Match`; a 304 renews through `platform/httpx/cache.go:revalidated` |
| A FIRMS request is made | per tile, never per location: `domains/fire/firms/tiles.go:tilesFor` → `domains/fire/firms/firms.go:hotspotsFor` (the parsed tile from the memo) |
| A seismic request is made | two concentric queries per location: a per-location near-field query and a regional query snapped to a shared 4° grid, built by `domains/seismic/usgs/usgs.go:queries` from `domains/seismic/rules.go:QueryPlan`; the shared body is parsed once through `domains/seismic/usgs/boxmemo.go:features`, then `Keep`-filtered per location in `domains/seismic/usgs/usgs.go:stateFor` |
| The SEISMIC section renders | `modes/tty/detail.go:detailLines` → `modes/tty/detail_seismic.go:seismicRows` (glyph ramp ○●◉ by felt band `seismicBand`, largest-then-nearest; "unavailable" vs "no recent activity"); the state is merged into the snapshot by `platform/snapshot/assembler.go` (Apply stores `a.seismic[k]`, Snapshot deep-copies via `platform/snapshot/merge_seismic.go:cloneSeismic`) |
| A location leaves the lists | `app/dashboard.go:commit` → `domains/weather/nws/points.go:Retain` (grid cache and gridpoint memo follow the set) |
| A cache entry expires or is swept | `platform/httpx/cache.go:get` (fresh or miss), `platform/httpx/cache.go:evictLocked` (memory), `platform/httpx/cache.go:sweep` (disk, allow-list) |
| The radio tunes | `app/radio.go:Tune` → `domains/radio/stream/resolve.go:ResolveWithStatus` → `app/radio.go:tuneList` → the engine `domains/radio/player/engine.go:Start`; synth fallback `app/radio.go:startSynth` |
| A relay directory is read | `domains/radio/stream/directory.go:MountsWithStatus` (5-min failure memo); a down relay warns via `app/radio.go:noteDirectories` |
| The Watchlist advances | the **Director** decides: `platform/lineup/bed.go:advanceBed` on a tick once the dwell has elapsed (live relay), or `:onEnded` when a synthesised cycle finishes. The deck only reports the facts — `app/radio.go:onStatus` tells it where the bed landed and when a cycle ended, through `app/radio.go:tell` — and `app/executors.go:runTune` cuts the bed over without lifting the alert duck |
| A voice is chosen or previewed | in **Settings**, not a chooser of its own (MVS-D-3): `modes/tty/setup_cast.go:cyclePicker` moves a picker, `modes/tty/setup_cast.go:previewOffer` is `p`'s ask-once flow, and `app/voices.go:PreviewVoice` speaks it. `V` opens Settings at the correspondents (`modes/tty/setup.go:openSetupAt`) |
| A list shows which row has the focus | `platform/render/list.go:ListMark` and `:ListLabel` — one owner for every list-shaped surface (the Settings window's picker rows). It is deliberately NOT the dashboard table's focus (`platform/render/table.go:rowStyles`, which also bolds the NAME cell and tints the row's other cells); `list.go`'s header says why the two differ and why merging them would break one of them |
| The Settings window is laid out | one row table owns the focus order, the keyboard rule and the › mark (`modes/tty/setup_rows.go:setupTable`); the body is two columns when they fit (`modes/tty/setup_layout.go:setupBody`), the chips are a pinned footer (`platform/render/panel.go:ScrollPanelFooter`) and the scroll follows the focused row and everything it draws (`modes/tty/setup_layout.go:focusScroll`, given the focused span in the coordinates the panel wraps in by `modes/tty/view.go:floatModalFooter` — the one place the three pinned-footer windows are drawn, sized by `modes/tty/view.go:footerModalChrome` and bodied by `modes/tty/setup_layout.go:focusBody`. It used to ask Setup whatever window was open, which left every other pinned-footer window unscrollable; it then counted in the UNWRAPPED body, which put the ctrl+d window's focused scenario off screen at 80x24) |
| The Radio panel picks a layout | `modes/tty/radio_panel.go:radioBreakpoint` — three fixed layouts by terminal columns, which is what retires the size toggle; the `v` control exists iff the breakpoint has a visualizer area (`modes/tty/radio_panel.go:hasViz`) |
| A role's voice is decided | `domains/radio/cast/resolve.go:Resolve` walks role → group → root → the platform default and says **why** the requested one lost; the deck answers the host questions (`app/cast.go:Discovered`, `app/cast.go:Installed`, `app/cast.go:Default`) and turns the answer into an engine in `app/cast.go:resolveVoice` — find-only, never an install (FR-9) |
| Correspondents hand over | the render goroutine sees the voice change between two segments and renders the line THERE, ahead of the air (`domains/radio/synth/source.go:renderAhead`); the writer only writes it (`domains/radio/synth/source.go:announce`). The one render the writer performs is the listener's own save mid-segment (`domains/radio/synth/source.go:takeOver`), folded with the remainder into a single utterance. The wording is `domains/radio/synth/compose.go:HandoffLine` over `scripts/handover/line.txt` |
| An alert's tone is chosen | `domains/radio/cast/tone.go:Classify` maps the product string to a class (storm beats warning and watch — MVS-D-15), `domains/radio/cast/tone.go:ToneName` names its preset, and `app/radio.go:tone` renders it from constants at `synth.ToneRate` — no voice is resolved, so an alert sounds while its correspondent is still being found |
| The `[S]` modal renders | `modes/tty/status.go:statusLines` ← `app/stats.go:ttyStats` (request/publish counters) |
| A diagnostic dump is written | `app/dump.go:Dump` (SIGUSR1 in `app/dump_unix.go:startDumpTrigger`; `/debug/dump` in `app/debug.go:startDebugProfiles`) |
| A report is printed | `cmd/watchpost/root.go:newReportCmd` → `app/app.go:ReportOnceWithStats` → `modes/report/report.go:RenderPlain` / `RenderJSON` |
| The Setup window finishes | `modes/tty/setup.go:setupFinishCmd` → `app/dashboard.go:setup` (persists, keys FIRMS) → `app/dashboard.go:commit` |
| Alerts are ordered for display | `modes/tty/nav.go:sortAlerts` — on the tty's own copy of the snapshot (the publisher deep-copies) |
| The ticker band scrolls and rotates | `modes/tty/ticker.go:advanceTicker` steps the tape one cell per tick, `modes/tty/ticker.go:advanceTickerCategory` hands the band to the next non-empty lane every `tickerRotate` and parks the old lane's offset so no tape's tail is unreachable; the lane's own name, colour and order come from ONE registry — `platform/category:Of`, `:Label` and `:Lanes` — which the window's tabs read too (F-21) |
| A time is written or spoken | `platform/render/clock.go` is the one owner: `:Time` / `:Since` / `:Stamp` write it, `:Spoken` says it and `:SpokenID` reads a callsign in NATO phonetics under the military convention. The listener's choice is Settings → WATCHPOST UI → Radio Convention |
| The app checks for a newer release | `app/release.go:start` asks ONCE at startup, and only when `update_check` is set (0.15.0 FR-7.1; it polled hourly before); `app/release.go:checkAt` asks GitHub and keeps the parsed numbers, never the published tag |
| A [S] table is laid out | `modes/tty/status.go:providerLines`, `:pipelineLines` and `:issueLines` build cells; `platform/render/status_table.go:StatusTable` lays them out on the go-studs table, and each table drops columns down a ladder rather than clipping one |

## Why something is slow on purpose

`docs/accepted-costs.md` — the register of measured, chosen costs (the modal compositor, the
always-on marquee frame, the per-location schedulers, the sequential provider fan-out) with the
trigger that would re-open each. The sites carry `ACCEPTED COST` comments pointing back at it.

## Vocabulary

| Term | Meaning | Defined in |
|---|---|---|
| snapshot | the immutable published view of every tracked location; providers write fragments, the assembler publishes | `platform/snapshot` |
| fragment | one provider's answer for one fetch kind and some locations | `platform/snapshot/types.go` |
| tier | one fetch kind on one cadence inside a scheduler | `platform/sched`, `app/pipelines.go:priorityTiers` |
| priority / RECENT pipeline | the favourites (one batched scheduler, the priority HTTP lane) / the 50-deep list (one scheduler per location) | `app/pipelines.go` |
| publisher | the coalescing window between "new data" and one snapshot (50 ms priority, 5 s RECENT) | `app/pipelines.go:Trigger` |
| body memo | the single slot holding the two rendered tables, keyed on every input they read | `modes/tty/memo.go` |
| modal memo | the single slot holding the open window's render, keyed on every input any window reads (the per-window extras projected only while that window is open) | `modes/tty/memo.go` |
| modal | the one floating window that can be open (`type modal int`); `modalNone` is the dashboard alone | `modes/tty/dashboard.go` |
| tick predicate | the rule for when the 300 ms animation tick runs at all | `modes/tty/dashboard.go:tickNeeded` |
| the grid | a tier's fire times: start + n·Every, whatever a cycle took | `platform/sched/sched.go:runTier` |
| lane | the client's two pacing queues: normal and priority; only the normal lane consults the failure memo | `platform/httpx/httpx.go:WithPriority` |
| failure memo | the per-host "avoid for 20 s" state armed by transport errors or repeated 5xx | `platform/httpx/memo.go` |
| validators | `ETag` / `Last-Modified` stored with a cache entry so a conditional GET can renew it | `platform/httpx/cache.go` |
| persistence floor | a body reaches disk only past a 5-minute caller TTL (or `Persist()`) | `platform/httpx/cache.go:put` |
| sweep | the disk tier's allow-list deleter (launch and daily) | `platform/httpx/cache.go:sweep` |
| gauge | one bounded structure's size, reported in `counters.json` | `app/stats.go` |
| box memo | the parsed features of the seismic query boxes most recently fetched, LRU-bounded and revalidated by body hash (a shared regional box parses once for the whole cell) | `domains/seismic/usgs/boxmemo.go` |
| parse memo | a feed's whole-country body parsed once per content change (HMS, WFIGS) | `domains/fire/memo.go` |
| tile | the fixed 5° cell a FIRMS request covers — the cache and singleflight key | `domains/fire/firms/tiles.go` |
| revalidation | a conditional GET on stored validators; a 304 renews the entry without a body | `platform/httpx/httpx.go:getOrRevalidate` |
| dump | a profile set + `counters.json` under the cache dir's `profiles/` | `app/dump.go` |
| deck | the radio player's controller: tune, queue, voices, status | `app/radio.go` |
| mount | one relayed transmitter stream on a relay | `domains/radio/stream/directory.go` |
| synth | the synthesized broadcast of the location's own NWS products | `domains/radio/synth` |
| token | a theme colour key (`Tok(name)`), never a literal SGR in views | `platform/render/theme.go` |
| the seam | `platform/render` — the only non-test package that imports go-studs (`table.go` is the table seam) | `platform/render/table.go` |

## Record IDs

| Prefix | Meaning | Defined in |
|---|---|---|
| UAT n | a fit-and-finish session of the 0.9.x pass | `06_docs/02_features/watchpost-cli/05-debugging/` |
| B0–B5 | the 0.9.x build batches | `06_docs/02_features/watchpost-cli/04-development/` |
| Qn | a quality-pass batch | `06_docs/02_features/watchpost-performance-quality-pass/03-architecture-design/quality-pass-plan.md` |
| L{1..5}-Fn, LR-n | DISCOVER lens findings | `…/02-analysis/` |
| JD/CQ/PA/PR/A11/BQ/IS/PH/DQ/SC/PF/RT/R2-n | red-team findings | `…/08-reports/red-team-plan.md` |
| P10-nn | safety-critical rules (`make p10`, the harness CLI's check) | the harness's P10 skill (outside the public tree) |
| C1–C5, OQ-n, D1/D2 | decisions, open questions, defects of the quality pass | `…/08-reports/discover-report.md`, `project-brief.md` |
| A location is typed, and becomes a place | the ONE resolver serves both halves so the suggestions and the commit cannot disagree: `app/resolve.go:newResolver` builds it over the embedded index, `domains/locations/resolver.go:TypeAhead` answers each keystroke from that index alone (never the network — a per-keystroke fetch is how a search box becomes a rate limit), and `domains/locations/resolver.go:Resolve` is the commit, which MAY reach the network and reports `fellBack` when it could not match exactly. `app/resolve.go:resolveHook` is what Settings and Lookup call, and it answers with the build error on every query when the index failed to load, so a broken resolver says why instead of returning nothing |

## Rules, and where they are stated

The table above answers *where does event X happen*. This one answers *where is
rule R stated* — which is the question that kept getting re-derived by
experiment on 2026-09-06, five times in one session, when the answer was in a
comment a few lines from where someone was already looking.

A rule belongs here when knowing it changes what you would do, and when it is
NOT discoverable by grepping for a symbol — you cannot search for "how does
Settings save" unless you already know the answer is `applyOnCloseCmds`.

| Rule | Where it is stated |
|---|---|
| **Settings has exactly two exits, and both write.** `esc` closes and applies; `enter` on a non-input row saves; `enter` on a PICKER row only advances, and the arrows cycle a value forever without committing it. So a path built out of Enters never terminates. | `modes/tty/setup.go:setupSave` and `modes/tty/setup.go:applyOnCloseCmds` — the header at the esc case spells out both exits |
| **Nothing left to WAIT is not nothing to CHECK.** `hold(d)` loops on `d > 0`, so a non-positive duration returns true without ever reaching the air check; the caller must ask separately whether the sequence is still on air. | `app/read_script.go:holdRest` |
| **Which categories the national feed can produce.** `Spec.Watchlist` marks the ones that arrive only through tracked locations. A category WITHOUT it is claimed to be feed-producible — which is how Emergency Orders were found unreachable. | `platform/category:Of` (the `Watchlist` field on the registry's Spec) |
| **In-flight audio is released on stop; HELD audio is not.** `StopPreview` closes the line in flight and leaves `heldOrder` alone. Releasing held lines is a different call, and the arbiter makes it whenever a suspended job ends. | `domains/radio/player/engine.go:StopPreview` and `:DropHeld`; the arbiter's side is `app/director.go:releaseBed` |
| **A mutant's CAUGHT is only evidence if the failure is attributable.** The green baseline is sampled once, so a flaky test can fail on the mutated run and be credited to the mutation. The harness re-runs the named test against the unmutated tree before believing it. | `06_docs/mutants/run.sh`, at the `--- FAIL` branch |
| **The band has ONE writer.** Only `mastercontrol` may construct a takeover message; it was written from two files once and the pair drifted. | `app/mastercontrol.go:cue`; the rule's history is in `app/executors.go`'s header |
