# RN — the regression net (0.19.5 lens survey)

Tree: `4c803d99` (v0.19.3), `feature/quality-pass`. Read-only. Lens: behaviour that no test pins
and that the other four goals (speed, consolidation, docs/comments, readability) are likely to touch.

## How I measured

- `go test -count=1 -coverprofile=cover.out ./...` (no race): every package green, 90.4% of statements
  with cross-package attribution (below).
- `go test -count=1 -coverpkg=./app/...,./modes/...,./platform/...,./domains/...,./pkg/... -coverprofile=coverx.out`
  over the same packages plus `./cmd/...`. This credits a function to ANY package's tests, so "0%"
  below means no test anywhere runs it (the PTY journeys in `cmd/watchpost` drive a built binary and are
  not counted, so wiring they exercise can still show as 0%; I say so where it matters).
- Block-level gaps read with a small script over `coverx.out` (max count per block across test binaries).
- Profiles and the script are in
  `<lens scratch>/rn/`
  (`coverx.out`, `funcx.txt`, `zerox.txt`, `unc.sh <file> [from] [to]`).
- Mutant corpus: 903 files, 167 distinct target files. I checked each finding's target against it.

The net is strong where the project has spent its attention (`platform/lineup`, the router, the
Broadcaster's slots, the severe window, the map). The gaps cluster in three places: code that talks to
the concrete `*player.Engine` (it cannot be faked, so it is never run), app-side adapters whose
domain-side halves are tested, and keyboard paths inside secondary windows.

---

### RN-1: The radio deck's whole voice and relay-fault side never runs under test
- Where: `app/radio.go:905` `duck`, `:925` `tone`, `:939-942` `pause`/`resume`/`stop`, `:953` `fault`,
  `:959` `discard`, `:969` `render`, `:983` `play`, `:995` `restore`, `:1000` `overlay`, `:1006`
  `pushStatus`, `:1056` `setDetail`; also `:499` `SetMode`, `:367` `onSilence`, `:385` `onClipSpent`,
  `:1282` `tuneCallsign`, `:1303` `readSynth`; `app/dashboard.go:1120` `tuneRelay` and `:1132`
  `readReport` (25%, only the nil guard); `domains/radio/player/engine.go:112` `reportSilence`.
- Observed: all 0% across every package's tests. `radioDeck.engine` is a concrete `*player.Engine`
  (`app/radio.go:39`), nil in every test, so any test that reached these would panic. The Director is
  tested through `narrationVoice` (`app/director.go:77`) with `scriptVoice`; the production
  implementation of that interface is never run. None of the 7 mutants on `app/radio.go` targets these
  lines. Behaviour that rides on them, unasserted:
  - `tone` returns 0 and sounds nothing for a muted class; it plays through `PreviewAside`, so the
    visualizer does not follow the tone (`:930-936`).
  - `play` routes a `viz` clip to `Preview` (bars follow) and a takeover clip to `PreviewAside`
    (`:983-989`). Swapping them passes every test.
  - `render` failure words on the player line: `"no voice for %s: %v"`, `"%s could not read: %v"`
    (`:972`, `:976`). These are the only explanation a listener gets for a silent takeover.
  - `onClipSpent` writes `"a read did not finish and was ended"` only when `abandonRead` reports a read
    was on the air (`:397-400`).
  - The relay-fault chain (MVS-D-76): engine silence -> `reportSilence` -> `onSilence` ->
    `RelaySilentMsg{Candidates}`; the window's pick -> `tuneCallsign` (picked station's mounts lead, the
    rest follow, `:1289-1299`); the countdown default -> `readSynth` -> `needsRead(ref, "the relay was
    silent", gen)`. The tty window is tested with injected messages; the app half is not.
  - `SetMode` (`m`, Synth / Nearest Relay): saves the pick always, re-tunes only when the monitor has the
    air and a station is playing (D-91, `:503-514`).
- Cost: these functions are short, look alike (one-line forwards to the engine) and sit in a 1,300-line
  file, which makes them the first thing a readability or consolidation pass tidies. A swapped `Preview`
  / `PreviewAside`, a dropped mute check, or a `SetMode` that stops saving would ship green.
- Proposed change: a narrow unexported interface on `radioDeck` for the engine methods it calls
  (`Suppress`, `Preview`, `PreviewAside`, `PausePreview`, `ResumePreview`, `StopPreview`, `DropHeld`,
  `Restore`, `Status`, `Start`, `Halt`, `Volume`), set from `*player.Engine` in `newRadioDeck`. This is a
  seam, not a rearchitecture. Then a recording fake and pins for the bullets above. Write the pins before
  touching `app/radio.go` for any other goal.
- Regression risk: the seam touches every deck call site; pinned indirectly today only by the PTY
  journeys (`cmd/watchpost/journey_test.go`), which do not assert these words or routes. Otherwise
  UNPINNED.
- Size: M
- Confidence: high on the coverage facts. I did not check whether the PTY journey happens to hit `m`
  with audio on.

### RN-2: Muting an alert-tone class in Settings has no test, and its rule is a copy of `cast.Muted`
- Where: `modes/tty/setup_tones.go:141` `toggleClass` (0%), `:101` `classMuted` (40%: the mute-mode
  branches `:105-108` never run), `:92` `toneStateWord` (the "MUTED" branch never runs); callers
  `modes/tty/setup.go:403` and `:540` (both uncovered). The rule it mirrors: `domains/radio/cast/tone.go:321`
  `Muted`.
- Observed: no test in `modes/` or `app/` sets `toneMuted` (grep). `toggleClass` carries the subtle
  case its own comment explains: under mute mode an EMPTY set means "every class", so un-muting one
  class must first write the other five down, and un-muting the last returns the mode to "all on"
  (`:130-140`, `:166-170`). `classMuted` says it "mirrors cast.Muted ... because modes/tty may not import
  the registry".
- Cost: README promises "`M` open Settings at the alert TONES, where each class can be muted". Two
  copies of one rule is exactly what the DRY pass looks for, and neither the tty copy nor the toggle is
  pinned. Concrete wrong input that would pass today: with mode=mute and an empty set, pressing space on
  "Warnings" leaves the set empty, so every class stays muted while the row says "Enabled" for nothing.
- Proposed change: pins first: (a) empty-set mute -> toggle one class -> five keys muted, mode mute;
  (b) toggle the last muted class -> mode back to on; (c) the row word for each state. Then any merge of
  `classMuted` with `cast.Muted` (for example a shared rule in `platform/`) is safe to try.
- Regression risk: UNPINNED (tty side). `cast.Muted` itself is pinned in `domains/radio/cast`.
- Size: S
- Confidence: high.

### RN-3: The seen-store cap test passes for the wrong reason; the alert-record cap has no test
- Where: `app/seen_store.go:64` `capOldest` (trim body `:68-80` 0%), test `TestSeenStoreIsPrivateAndBounded`
  in `app/severe_test.go:164`; `app/alert_store.go:78` `capOldest` (trim body `:82-94` 0%), cap
  `maxAlertRecords = 200` at `:32`.
- Observed: the test writes `maxSeenIDs+10` ids aged 0 to 20,009 seconds, then loads with
  `window = time.Hour` (`severe_test.go:176-185`). `loadSeen` drops everything older than the window
  first (`seen_store.go:43-46`), leaving about 3,600 ids, so `capOldest` returns at its length check and
  `n > maxSeenIDs` cannot fail. Delete the trim, or make it keep the oldest, and the test stays green
  (the coverage profile confirms the trim never runs). `alertStore.capOldest` is never reached by any
  test. The two functions are near-copies (`alert_store.go:72` says so).
- Cost: these two copies are a natural consolidation target (one "keep newest N" helper). A helper that
  sorts the wrong way passes everything. For `alertStore` that means: after 200 notes, the NEWEST burst's
  records are the ones evicted, the executor "declines the build by name" (`alert_store.go:27-30`) and a
  takeover alert is not read.
- Proposed change: give the seen-store test a window longer than the oldest id (or `0` aged ids) so the
  cap is what trims, and assert WHICH ids survive (the newest); add the same for `alertStore` with 201
  notes. Both before any merge.
- Regression risk: UNPINNED (the existing test does not pin the cap).
- Size: S
- Confidence: high.

### RN-4: The Line-Up Request window's keys are unpinned, and its backspace cuts a byte, not a letter
- Where: `modes/tty/request.go:445-457` `requestErase` (0%); `:350` `prev` (0%, the up arrow);
  `:391-399` tab, space and backspace dispatch, `:411` `requestToggle` (50%), `:427` `requestType`
  (43%: typing into Position `:436-441` never runs). The seven other text fields:
  `modes/tty/modal_location.go:68-69`, `setup.go:800-801`, `:832-833`, `:856-857`, `:880-881`,
  `map_prop_find.go:101-102`, `map_prop_freq.go:251-252`, `router.go:1167-1168`.
- Observed: every other field erases a rune (`r := []rune(s); string(r[:len(r)-1])`). The request window
  slices bytes: `d.request.query = d.request.query[:n-1]` (`request.go:449`). Type `Peñ`, press
  backspace: the query becomes `"Pe\xc3"`, an invalid UTF-8 string sent to the locate pause. The same
  file guards typing for exactly this case (`:400-403`, "one byte per key turns Peña into Pea"). Up
  arrow, tab, space on a report row, space on PRIORITIZE, digits into Position, and backspace in either
  field: none is run by a test.
- Cost: an observed defect today, and a DRY target (eight copies of one erase). Merging the eight into
  one helper fixes `request.go` and changes its behaviour, so the change needs a pin that shows the
  difference.
- Proposed change: one test per key in the Request window (up wraps to Position, tab skips report rows,
  space toggles a report and flips PRIORITIZE, digits fill the slot, backspace erases a rune in each
  field, including `Peñ`). Then an `eraseRune` helper for the eight sites.
- Regression risk: UNPINNED for the window's keys. The Lookup window's backspace is run by
  `modal_location_test.go` (its block at `modal_location.go:67-70` is covered); Settings' by
  `setup_esc_save_test.go`. No test presses backspace on a multi-byte letter anywhere.
- Size: S
- Confidence: high (the byte slice is plain Go semantics; I did not run the app).

### RN-5: The spoken Marine and Seismic reports' app-side adapters, and Marine's place in the broadcast, are untested
- Where: `app/marine.go:27` `marineReportOf` (0%), `app/seismic.go:16` `seismicReportOf` (0%);
  `app/marine.go:17` `marineFor`, `app/dashboard.go:932` `fireFor`, `:944` `seismicFor` (each 50%: the
  branch that finds the location never runs); `domains/radio/synth/compose.go:152-155` (the marine block
  in a composed broadcast, 0%) and `domains/radio/synth/manifest.go:48` `marineSpan` (0%).
- Observed: `MarineSegments` and `SeismicSegments` are well tested on their own (94%, 95%), and
  `fireReportOf` is 100%. But no test composes a broadcast that contains a marine report, so the ruled
  order (products -> MARITIME -> fire -> seismic, MVS-D-18), the two-second pause before it and its
  manifest `Source`/`Detail` are unpinned. The adapters carry two ruled behaviours: tide times read in
  the LOCATION's zone, not the listener's (MVS-D-21, `marine.go:33-38`), and "Known only once the USGS feed
  has answered" (`seismic.go:16-20`, `AsOf.IsZero()` -> skip).
- Cost: `marineFor` / `fireFor` / `seismicFor` are three copies of one loop ("ask each assembler for one
  location's block"), an obvious consolidation; only fire's inner adapter is pinned. A merged helper that
  dropped the zone, or that marked a cold seismic state Known (a report read from an empty list, in the
  present tense), passes every test.
- Proposed change: pin `marineReportOf` (zone applied; bad zone name leaves nil), `seismicReportOf`
  (zero `AsOf` -> unknown), and one `Compose` test with marine + fire + seismic asserting segment order,
  pauses and the manifest line.
- Regression risk: UNPINNED for the three bullets.
- Size: S
- Confidence: high.

### RN-6: The operator's own card: its minting, its "Station Operator" label and the Change Position window are unpinned
- Where: `app/mastercontrol.go:232` `RequestCard` (0%), `:218` `MoveCard`, `:222` `DropCard` (0%);
  `app/pool.go:483` `requestCard` (stops at the nil-director guard), `:88` `moveCard`, `:95` `dropCard`
  (0%); `modes/tty/broadcaster_lineup.go:166` `requestedByOf` (40%: "Station Operator" `:169` and
  "Director" `:171` never run); `modes/tty/broadcaster_manage.go:44` `movePrompt` (0%), reached from
  `modes/tty/view.go:73` (uncovered); backspace in the move field `router.go:1167-1170` (uncovered).
- Observed: `RequestCard` is the one place an operator's card is built: `ID: lineup.ReadID(key)`,
  `Slot: LocationReport`, `Origin: FromOperator`, `Reports: kinds` (`mastercontrol.go:234-236`). Two
  near-identical Propose -> `To(Admitted)` sequences live in `platform/lineup/rotation.go:52` and
  `topoff.go:253` (both `FromDirector`, no `Reports`). The tty side colours a `FromOperator` card
  (`broadcaster.go:1735`, tested with a hand-made card) and labels it "Station Operator" (untested). The
  "Change Position" window (`MOVE THIS CARD TO POSITION`, "Positions N to M ...") is never drawn by a
  test; `moveTarget` and `dropPrompt` are 100%.
- Cost: the three Propose->Admitted sites invite a shared helper; a helper that takes the rotation's
  defaults drops `Origin: FromOperator` (the card loses its operator colour and label, and
  `platform/lineup/transitions.go:29`'s rule about operator cards stops applying) or drops `Reports` (the
  card reads the default reports, not the ones the operator ticked). Neither would fail a test.
- Proposed change: an app test that calls `lp.requestCard` with a live director and asserts the
  `Requested` event's card (origin, slot, reports, id); a tty test for `requestedByOf`'s three words; one
  render of the Change Position window (and its error line).
- Regression risk: `platform/lineup` pins what the Director does with `Requested`/`Moved`/`Dropped`
  (many mutants on `director.go`, `operator.go`); the app minting and the words are UNPINNED.
- Size: S
- Confidence: high.

### RN-7: The Status window's "worst feed wins on a shared host" rule has never run
- Where: `modes/tty/status_sources.go:120` (the `worseStatus` half of the condition), `modes/tty/status.go:426`
  `worseStatus`, `:428` `statusRank` (both 0%). Production sharing: `app/endpoints.go:25-29`
  (`nws` and `nws-marine` on `api.weather.gov`; `coops` and `coops-obs` on `api.tidesandcurrents.noaa.gov`).
- Observed: `r.status == "" || worseStatus(p.Status, r.status)`: every test host carries one provider, so
  the right side is never evaluated. The same is true of "the OLDEST fetch" on a shared host
  (`:123-125`, second provider). Concrete input: `nws` OK, `nws-marine` DEGRADED -> the
  `api.weather.gov` row must say DEGRADED; nothing asserts it. `statusRank` orders ok < off < degraded.
- Cost: follow-up F-4 already names this window as building its tables 3-4 times a render with no
  allocation pin, so the performance pass is likely to restructure it. A rebuilt loop that keeps the
  first provider's status, or the newest fetch, hides the degraded marine feed behind a green row.
- Proposed change: one test with two providers on one host (OK + DEGRADED, two fetch times), asserting
  the row's state and age, both orders.
- Regression risk: UNPINNED.
- Size: S
- Confidence: high.

### RN-8: The ten-favourite cap has no test
- Where: `modes/tty/modal_location.go:112` `WatchCap = 10`; `:22-24` (enter is inert when full),
  `:145-148` (refusal "the priority list is full (%d locations)"), `:300-303` (note "Only 10 locations are
  allowed in the priority list"), `modes/tty/dashboard.go:1493` (`canAddFocused`). All three modal blocks
  uncovered; no test names `WatchCap` or any of the words.
- Observed: README "make the focused location a favourite (up to 10)". The note hard-codes "10" while the
  refusal uses `WatchCap`, so the two can drift (a docs/consolidation pass is likely to touch exactly this).
- Cost: a cap that stops applying lets an eleventh favourite through to the priority pipeline
  (`httpx` sizes its favourites' lane for it, `httpx.go:100`); a reworded note goes stale silently.
- Proposed change: a test that fills the watchlist to `WatchCap`, then asserts the note, the muted chip,
  the inert enter and the refusal; derive the note's number from `WatchCap`.
- Regression risk: UNPINNED.
- Size: S
- Confidence: high.

### RN-9: Forty Propagation tests decide "a result came" or "no result came" by a 200 ms wall clock
- Where: `modes/tty/map_prop_update_test.go:79-99` `propMsgs`, used by `runProp` (40 call sites) and
  directly in `map_prop_stations_test.go:35`, `:44`, `:49` and `map_prop_draw_test.go:204`.
- Observed: the helper runs the command on a goroutine and returns nil if nothing arrives in 200 ms
  (`:97`, "a tick, or a wait on the network: not this test's"). The same timeout is the evidence for both
  outcomes: `TestStationsNextAsksAgainAtOnce` requires exactly one result (`:36`, `:44`) and then requires
  NONE ("a fresh snapshot ... waits for the refresh tick", `:49`). This is the class F-190 fixed in `app/`
  and `platform/sched` (2 s ran out on a loaded runner, raised to 10 s), and the class that failed
  `TestTheAppsEndStopsAnUpdateAndKeepsItsState` on the hosted race leg (0.19.0 build log :904). 200 ms is
  a tenth of the limit that already proved too short.
- Cost: a loaded race runner can turn a passing test red (the positive case), and the negative case can
  pass for the wrong reason if a regression makes the immediate ask slow. A perf change that moves work
  into the command makes both more likely.
- Proposed change: classify the command without timing (inspect the batch for a tick command by type,
  run only non-tick commands to completion with a generous bound), so "no immediate ask" is asserted
  structurally.
- Regression risk: test-only.
- Size: M
- Confidence: medium (no recorded failure of these tests found in the build logs; the mechanism is the
  same as two recorded ones).

### RN-10: `TestFirstPublishIsImmediateAndLaunchFillsProgressively` is F-51's sibling and is not in F-51
- Where: `app/pipelines_test.go:159-200`.
- Observed: a real-time publisher with `launchWindow: 40ms`, then `time.Sleep(120ms)` and
  `runs.Load() != 2` (margin 80 ms); a hard `time.Since(start) > 100ms` on the first publish; and a 120 ms
  margin on the steady window. F-51 records `TestPublisherWindowFoldsAWave` in the same file failing on a
  macOS CI leg for exactly this (a 40 ms window spread by load) and proposes the fake clock
  `platform/sched` already has; it names only the one test.
- Cost: the same flake, waiting; and if the perf pass touches the publisher (it is on the refresh path),
  this test is its pin.
- Proposed change: fold it into F-51's fix (inject the clock), or at least add it to F-51's row.
- Regression risk: test-only.
- Size: S (with F-51)
- Confidence: medium (no recorded failure).

### RN-11: Only four surfaces have a whole-frame golden; the Broadcaster console has none
- Where: `modes/tty/testdata/` holds frames for the dashboard (`frame-133x44-*`), Settings
  (`setup-*`), the severe window (`severe-*`) and the map (`map-*`). None for the Broadcaster console
  (`broadcaster.go`, 1,811 lines, plus 9 more `broadcaster_*.go`), Location Details (`detail*.go`),
  Help/About (`help_about.go`), Status (`status*.go`, 1,146 lines) or Diagnostics (`debug.go`).
- Observed: these surfaces have many targeted assertions (a word is present, a column lines up), but a
  readability refactor that shifts padding, reorders two rows or drops a separator between pinned
  substrings passes them. The goldens are what caught such shifts on the four surfaces that have them.
- Cost: goal 5 (junior readability) will rewrite exactly these large render files.
- Proposed change: capture goldens (plain + ASCII, 133x44 and 80x24) for the console on STANDBY and ON
  AIR, Details for a coastal location, Help, Status and Diagnostics, from the existing fixtures, before
  the readability pass starts. `-update-golden` already exists (`golden_test.go:20`).
- Regression risk: test-only.
- Size: M
- Confidence: high on the inventory; I did not measure how many existing assertions a given layout
  shift would slip past.

### RN-12: The map Description's "best answer per layer" choice never runs
- Where: `modes/tty/map_layer_words.go:97` `betterAnswer` (0%), used at `:34`.
- Observed: no test gives two answers for the same layer, so "prefer an answer with data, and of two
  the nearer" is unpinned. Concrete input: a NoData answer at 2 km and a real reading at 40 km; the
  Description must say the 40 km reading.
- Cost: `layerAnswerWords` runs on the map's description every frame the box is open (README's
  Description box), so the perf pass may rework the map; a "first answer wins" loop would read "no data"
  where data exists.
- Proposed change: one table test of `layerAnswerWords` with duplicate answers per layer.
- Regression risk: UNPINNED.
- Size: S
- Confidence: high.

### RN-13: Location Details' marine rows "Swell 2", "Wind Waves" and "Buoy Wind" are never drawn
- Where: `modes/tty/detail_marine.go:129-139` (uncovered); `platform/render/units.go:253` `Wind` is 0%
  across all packages because this is its only caller.
- Observed: the fixtures carry a primary swell only. The buoy wind row (with "(gusts ...)") and its unit
  conversion through `Opts.Wind` are unpinned, as are `titleWord`/`period` fallbacks (`:224`, `:232`).
- Cost: `render/units.go` has already been consolidated once (`Opts.measured`, observer-maps D-205) and is
  a likely target again; `Wind` would change unseen.
- Proposed change: extend a marine Details fixture with a secondary swell, wind waves, wind and gust;
  assert the rows in both unit systems.
- Regression risk: UNPINNED.
- Size: S
- Confidence: high.

---

## Minor

- `modes/tty/router.go:243` `scrollStep` (0%) and `:739-741`: the console's ↑/↓ scroll the running order
  through the router; `scrollQueue` itself is tested directly, the key-to-direction mapping is not
  (swap the sign and nothing fails).
- `modes/tty/view.go:301` `footerModalScrollMax` (0%) and `modes/tty/debug.go:195-208`: scrolling the
  `ctrl+d` Diagnostics window down is unpinned (F-35 covers the window's height, not this).
- `domains/radio/player/alert_test.go:125` `waitFor` polls for 3 s; `app/director_test.go:362` and
  `platform/sched/sched_test.go:248` use 10 s after F-190 found 2 s too short. Three copies of one helper,
  one still short.
- `domains/radio/player/alert_test.go:399-403` (`TestANewStreamStartedUnderADuckPlaysDucked`): "the next
  location to give way too" is `len(players) >= 2 && !last.IsPlaying()`, which a fresh `recordingPlayer`
  satisfies before `Play` is decided (`IsPlaying` is false until `Play`, `:72-82`). The behaviour is
  pinned anyway by `TestARenderedReportStartingUnderAnAlertIsHeldFromTheFirstSample` (`:469`), which
  counts `plays`; this test's wait should check `plays == 0` too.
- `domains/alerts/replay_test.go:88`: `advanceTo` ends with `time.Sleep(5ms)` "let tier goroutines run
  their cycle"; `platform/sched` has `awaitWaiters` for exactly this. A late goroutine stamps a later
  fake time and can push M2's latency past 60 s.
- `platform/history/prune.go:357-363`: `sweep`'s removal of a crashed writer's temp file is unpinned, and
  it reads the wall clock (`time.Since`) where the claim path reads `s.now()`, so it cannot be pinned
  with the store's fake clock as written. Same in `claim.go:76-80` (unreadable claim judged by mod time).
- `app/stats.go:210-231` `sources` (44%): the type switches that pick HMS/WFIGS/FIRMS/USGS for
  Diagnostics never match in tests; a consolidation that drops a case blanks that provider's line.
- `app/voices.go:160` `PreviewVoice` (12%): the Settings correspondents' preview words are unpinned.
- `modes/tty/broadcaster_manage.go:45-47` and `:74-76`: `movePrompt` and `dropPrompt` each define the
  same `centre` closure (only the second is ever run).

## What is in good shape (do not spend time here)

- `platform/render`: with cross-package credit every function but `Wind` is run, and the severe, lineup
  and alert tables have their own tests and goldens.
- `platform/lineup` and `modes/tty/router.go`: the densest mutant coverage in the tree (30 on
  `director.go`, 27 on `router.go`, 25 on `broadcaster.go`).
- Memo keys: `memo_completeness_test.go`, `broadcaster_memo_completeness_test.go`,
  `body_completeness_test.go` and `app/memokeys_test.go` fail when a field is added without a key, which
  is what a perf change to the memos needs.
- Config persistence: unknown keys kept, arrays of tables, hostile keys (`platform/config/radio_test.go`).
- `platform/httpx`: redaction of URLs in transport errors (`httpx_test.go:100`), lanes, conditional
  requests, the hung-host path; 89.6% own-package.
- Degraded words on the main surfaces: "NHC unavailable" (`severe_test.go:269`), "seismic data
  unavailable" (`detail_seismic_test.go:66`), the Propagation states table (`map_prop_states_test.go`).
- The four golden surfaces (dashboard, Settings, severe, map) and the allocation pins
  (`tick_alloc_test.go`, `map_alloc_test.go`, `broadcaster_alloc_test.go`).
- `waitUntil` uses in `app/severe_read_test.go` and `app/dashboard_test.go` that wait on `!busy`: safe,
  because `eventReader.Read` sets `busy` before it returns (`app/severe_read.go:134`).

## What I did not cover

- I did not run `-race`, benchmarks or the mutant corpus, and did not run the app.
- Coverage does not count the PTY journeys in `cmd/watchpost` (they drive a binary), so a few app wiring
  functions reported at 0% (`startPipelines`, `attachRadio`, `buildDirector`) may be exercised there; I
  did not report those.
- `tools/` and `third_party/` were out of scope for this lens (several tools are 0% by design).
- I read the README's key tables against the keymap and spot-checked about fifteen claims; I did not
  check every README sentence against a test.
- I checked `waitUntil`/`waitFor` conditions in `app/`, `platform/sched` and `domains/radio/player`
  for "an earlier state satisfies it"; I did not audit every polling loop written inline in other test
  files (`app/mapradar_test.go:431-472`, `modes/tty/map_radar_test.go:710` use fixed sleeps that I read
  but did not reason through fully).
