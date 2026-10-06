---
title: "0.19.0 DISCOVER — wave 1 findings (desk)"
date: 2026-10-05
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "IN PROGRESS — sections land as each wave-1 survey reports (D-18)"
---

# Wave 1 findings

Four read-only surveys (D-18):
- sources and their terms, with `giro.uml.edu` added at the HUM LEAD's request;
- the building blocks' licences and the published method;
- watchpost's code;
- go-tuiMaps' gaps.

Every claim cites `file:line` or a source. A claim the agent spot-checked is marked **checked**.

## go-tuiMaps v0.2.0 (origin/main `d1c4d5e`)

All five intake gaps (HR-1 to HR-5) are real. Two of them are promises the library already wrote down and
never built, and the research missed one gap.

| # | Finding | Evidence |
|---|---|---|
| HR-1 | Fields stop at the shore: `render.Input.FieldsOverWater` is set only by an internal test. `Map.Render` (`map.go:431`) builds `Input` through `paint` and `draw`, neither of which sets it. `ImagesMaskedByWater` has no public setter either. **checked** | `internal/render/frame.go:92`, `:96`; `field.go:182`; `field_test.go:77` |
| HR-1 (new) | **A broken promise.** FR-12 and D-87 say "a host can flip either, per overlay"; the plan names `TestHostCanFlipEither`, and the build log claims the flip. Neither the public path nor the test exists. | `requirements.md:38`; `rulings-discover.md:83`; `implementation-plan.md:355`; `build-log.md:191` |
| HR-2 | A host type has no fill: `classInk` returns 0 for preset 0. The legend's "no ramp" code is `rampToken`, not `facts.go:186` as the research said. The legend **does** carry the unit (`facts.go:97`), which corrects the research. | `internal/render/field.go:88-108`; `facts.go:179`, `:189-207` |
| HR-2 (new) | **A broken promise.** FR-15 promised a ramp for a host's own type from the `low`, `middle` and `high` tokens. The tokens are exported, but they have no default colour and nothing reads them. | `requirements.md:42`; `constants.md:109`; `internal/colour/token.go:58-60`; `preset.go:258-290` |
| HR-2 (new) | A host-type grid without `Lines` draws **nothing** at truecolor and 256 colours, and nothing warns. No test renders a host-type grid. | `internal/render/field.go:146-163` |
| HR-3 | Contour labels have no unit, use one ink, and sit only on east-west crossings, every 24th dot row. MUF's contours run east-west, so they would mostly go unlabelled. `Grid.Marks` (8 a field) can carry "14MHz" today. | `internal/overlay/field.go:167-176`, `:37-41`; `render/field.go:310-323` |
| HR-4 | There is no terminator or night code. A terminator line could use an existing role today, but `track` means a storm track. Night shading is impossible: only alert roles fill areas. | `render/basemap.go:206-208`; `store.go:278` |
| HR-5 | Colour by value exists only through `AirQualityRole` and `UVRole`. Point overlays get no legend. OW-17 already owes a non-colour mark. | `overlays.go:218-240`; `facts.go:129-136` |
| Words (new) | A field's `Answer` has an empty `ValueUnit` except for temperature, so a MUF answer has no "MHz" and no band words. **checked** | `internal/describe/answer.go:138-141`; fix at `describe.go:185`, as wind does |
| Globe | `FitWorld` frames 84°N to 56°S, so the far south can fall off. **checked** No test renders a whole-globe grid; "works today" rests on reading the code. A 2°×2° globe is 16,200 cells against a cap of 1,048,576. | `internal/project/fit.go:158-162`; `overlay/field.go:13` |
| Bound | The library imposes no regional limit. `SetBound(Bound{})` lifts a map's bound, and a separate propagation Map can share caches (`SharedCaches`). Only watchpost's D-8/D-28 forbid the wider frame. | `view.go:224-269`, `:257-258`; `shared.go:52` |
| Safe ramps (new) | `SafeRamps` covers only the tokens from `AlertExtremeOutline` to `High`. The wind, wave, UV, AQI and QPF tokens come after `High`, so they keep host colours with `SafeRamps` on, and `Warnings` stops reporting them. New MUF tokens would inherit this. **checked** | `internal/colour/palette.go:72`, `:99-101`; `sixteen.go:35` |

**Already owed to v0.3.0:**
- OW-14 to OW-20 and OW-24 to OW-26;
- F-1 (the crowded root);
- F-2 / OW-6 (pluggable architecture).

None covers HR-1 to HR-5. OW-24 touches the same legend `Class` as HR-2, and OW-17 is the colour-only problem HR-5 would add to.

**What v0.3.0's brief must decide (for its own DISCOVER):**
- the water flip per overlay (FR-12's promise) or map-wide;
- MUF and foF2 presets, or the generic host ramp FR-15 promised;
- units on labels and answers;
- whether a terminator and night shading are the library's job.

## watchpost (`48616c69`; no Go file changed since the brief's `538f3dbb`)

| Area | Finding | Evidence |
|---|---|---|
| Layers | A layer registers itself (`registerMapLayer`) and reaches the window through `Config` funcs (`MapFeed`, `MapTemperature`, ...), because `modes/` cannot import `domains/`. Field fetches go box by box, and a region is split at the antimeridian, which a grid cannot cross. | `app/maplayers.go:157-203`; `modes/tty/dashboard.go:86-131`; `app/maptemp.go:637-682` |
| Bound | There are six fixed US region boxes. **A place outside every region gets no map and no description** (the outside branch returns before any words). **checked** A non-US ham gets nothing, so R-1.3's words path inherits the region gate as built. | `platform/geo/regions.go:30-52`; `modes/tty/map_pane.go:466-477` |
| Words | The description is alerts only (F-205). A layer cannot add a sentence today, only a note (`mapNotesNow`). M5b needs a new seam in `describeUpTo`, or in the notes, plus a `modalKey` input. | `modes/tty/map_describe.go:57-107`; `map_pane.go:581-587`; `memo.go:337-347` |
| Broadcaster | The console has no windows of its own: Observer windows are composited over it, and the Observer's map follows the Observer's selected place, not the tower (F-174's "not the Observer's"). The 44-row floor is "fixed chrome plus one readable card". The tower and service radius are already first-class (`config.Broadcaster`, `StationAreaMsg`). | `modes/tty/router.go:513-518`; `broadcaster.go:621-668`; `platform/config/broadcaster.go:17-47` |
| Bands | **watchpost knows no operator band or frequency.** The only MHz values are NOAA Weather Radio's relay transmitters. "band" already names the console's painted notice strips. | `domains/radio/stream/table.go:29-32`; `broadcaster.go:326-447` |
| Settings | Rows are constants whose order is the focus order. Each tab has a notice area (D-237); the Broadcaster tab has no notice case yet. | `modes/tty/setup_rows.go:59-277`; `setup_notices.go:113-159` |
| Credits | `creditGroups()` lists credits by hand. **No test ties a registered source to its credit.** | `app/credits.go:16-71`; `app/credits_test.go:29-67` |
| Egress | `mapSourceList()` builds MAP STATUS's disclosure; a new host needs an entry and a `Sent` value. httpx is the single client package, with a mandatory User-Agent and **no allowed-hosts list**. go-tuiMaps fetches with its own transport and User-Agent. | `app/maps.go:213-225`; `platform/httpx/httpx.go:58-74`; `app/maptiles.go:35-44` |
| History | The store takes a per-station series and a global grid per hour (no cell cap; its doc comments use MUF as the example). But it **opens only when an NDFD temperature source exists**, and **one retention from the Data tab overrides every dataset's own** (D-175, D-231). **checked** | `platform/history/history.go:55-125`, `:566-588`; `app/history.go:147-156`, `:496-505` |
| #12 | Memo keys and what the completeness guards cover: see below. | `modes/tty/memo.go:36-387`; `memo_completeness_test.go:37-245` |
| F-187/F-188 | `say` is found on PATH, not at `/usr/bin/say`. Its only kill is CommandContext's SIGKILL to one PID. No process group, `WaitDelay` or termination-signal handler exists. **checked** A watchpost killed outright leaves `say` and its temp dir behind. F-187's suggested sweep names the temp-dir pattern, not the process. `make test-say` has no `-timeout`. | `domains/radio/synth/voice.go:94-121`; `app/dump_unix.go:28`; `Makefile:102-103` |

**#12, memo keys:**
- Every guard checks the same property: a field that moves the frame must move the key. It perturbs only top-level scalars and pointers, one level deep, and never uint, slice or map fields.
- None asks the identity question #12 raises: can what sits at a position change while the position and the keyed identities stay the same?
- The keys holding positions are `bodyKey`, `modalKey`, `consoleKey` and `tempMemoKey`.
- The app-side keys (`tempKey`, `airGridKey`, `uvDayKey`, `frameKey`, the area memo, `feedKey`) carry identities only, and **have no completeness guard**.

**Minor:** `setup_rows.go:63-71` says the transmitter rows sit in DATA; they sit on the Broadcaster tab (`setup_tabs.go:50-51`).
