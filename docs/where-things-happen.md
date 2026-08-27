# Where things happen

A flow map for someone reading the code for the first time: the event on the left, the function
that handles it on the right. Every `path:Func` here is checked by `cmd/watchpost`'s
`TestWhereThingsHappenNamesRealSymbols`, so this page cannot drift silently. The per-file headers
say what a file holds; this page says where a *thing* happens.

| Event | Where |
|---|---|
| The program starts | `app/dashboard.go:RunDashboard` (composition root) → `app/pipelines.go:startPriority`, `app/pipelines.go:startRecent` |
| A key is pressed | `modes/tty/dashboard.go:handleKey` → a modal opens via `modes/tty/dashboard.go:toggleModal`; the radio keys via `modes/tty/radio_panel.go:toggleRadio` |
| A window opens or closes | `modes/tty/dashboard.go:toggleModal` → `modes/tty/dashboard.go:open` / `modes/tty/dashboard.go:close` (one `modal` value, so opening one closes the rest); drawn by `modes/tty/view.go:modalView` |
| `enter` opens Location Details | `modes/tty/dashboard.go:toggleModal` → the body `modes/tty/detail.go:detailLines` (+ `modes/tty/detail_fire.go:fireRows`, `modes/tty/detail_marine.go:maritimeRows`) |
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
| The Watchlist advances | `app/radio_queue.go:advanceQueue` after `app/radio_queue.go:armDwell` (live) or the synth cycle end in `app/radio.go:onStatus` |
| A voice is chosen or previewed | `app/voices.go:SetVoice`, `app/voices.go:PreviewVoice`; the chooser `modes/tty/modal_chooser.go:handleVoiceKey` |
| The `[S]` modal renders | `modes/tty/status.go:statusLines` ← `app/stats.go:ttyStats` (request/publish counters) |
| A diagnostic dump is written | `app/dump.go:Dump` (SIGUSR1 in `app/dump_unix.go:startDumpTrigger`; `/debug/dump` in `app/debug.go:startDebugProfiles`) |
| A report is printed | `cmd/watchpost/root.go:newReportCmd` → `app/app.go:ReportOnceWithStats` → `modes/report/report.go:RenderPlain` / `RenderJSON` |
| The Setup window finishes | `modes/tty/setup.go:setupFinishCmd` → `app/dashboard.go:setup` (persists, keys FIRMS) → `app/dashboard.go:commit` |
| Alerts are ordered for display | `modes/tty/nav.go:sortAlerts` — on the tty's own copy of the snapshot (the publisher deep-copies) |

## Vocabulary

| Term | Meaning | Defined in |
|---|---|---|
| snapshot | the immutable published view of every tracked location; providers write fragments, the assembler publishes | `platform/snapshot` |
| fragment | one provider's answer for one fetch kind and some locations | `platform/snapshot/types.go` |
| tier | one fetch kind on one cadence inside a scheduler | `platform/sched`, `app/pipelines.go:priorityTiers` |
| priority / RECENT pipeline | the favourites (one batched scheduler, the priority HTTP lane) / the 50-deep list (one scheduler per location) | `app/pipelines.go` |
| publisher | the coalescing window between "new data" and one snapshot (50 ms priority, 5 s RECENT) | `app/pipelines.go:Trigger` |
| body memo | the single slot holding the two rendered tables, keyed on every input they read | `modes/tty/memo.go` |
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
| the seam | `platform/render/table.go` — the only file that imports go-studs | `platform/render/table.go` |

## Record IDs

| Prefix | Meaning | Defined in |
|---|---|---|
| UAT n | a fit-and-finish session of the 0.9.x pass | `06_docs/02_features/watchpost-cli/05-debugging/` |
| B0–B5 | the 0.9.x build batches | `06_docs/02_features/watchpost-cli/04-development/` |
| Qn | a quality-pass batch | `06_docs/02_features/watchpost-performance-quality-pass/03-architecture-design/quality-pass-plan.md` |
| L{1..5}-Fn, LR-n | DISCOVER lens findings | `…/02-analysis/` |
| JD/CQ/PA/PR/A11/BQ/IS/PH/DQ/SC/PF/RT/R2-n | red-team findings | `…/08-reports/red-team-plan.md` |
| P10-nn | safety-critical rules (`a2dh p10 check`) | li-A2DH `02_skills/implementation/p10/` |
| C1–C5, OQ-n, D1/D2 | decisions, open questions, defects of the quality pass | `…/08-reports/discover-report.md`, `project-brief.md` |
