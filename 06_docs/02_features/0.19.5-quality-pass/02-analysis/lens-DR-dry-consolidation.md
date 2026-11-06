# Lens DR: DRY and consolidation (goal 3)

Commit `4c803d99` (v0.19.3), branch `feature/quality-pass`. Read-only. Prefix DR.

## How I looked

- `go run ./tools/dupes` at the gated floor (25 nodes): 2 groups, both ratified. Then the same tool
  with its own `-min` flag at 20, 15, 12 and 8 nodes (no repo change was needed, the flag exists):
  23, 64, 88 and more groups. I read every group at 12 nodes and above.
- A scratch tool (`<lens scratch>/dr/exact/`) that groups functions by **exact** printed body at any
  size, and groups unexported top-level functions that share a **name** across packages. That found
  the small copies the structural floor cannot see.
- Targeted greps for the semantic duplicates a fingerprint cannot see: distances, bearings, compass
  tables, unit conversions, coordinate checks, rune clamps, age and duration wordings, time formats,
  severity ladders, location identity, context sleeps, backoff, host parsing.
- Each candidate was checked by reading every copy and asking whether one change should always apply
  to all of them. Rejected candidates are listed near the end with the reason.

The structural detector's count is a floor, as it says. Most of the findings below are semantic
(different shapes, the same question) or below its floor.

---

## Findings, most valuable first

### DR-1: Felt-band magnitude thresholds live in three parallel switches, kept in step by comments
- Where: `platform/render/units.go:181` (`SeismicLevel`: 3.5, 5.0), `modes/tty/detail_seismic.go:125`
  (`seismicBand`: 3.5, 4.5, 5.0), `domains/radio/synth/seismic.go:89` (`feltLikelihood`: 3.5, 4.5)
- Observed: all three answer one question: which felt band is magnitude m in? The bands are below
  feeling (<3.5), might feel it (3.5 to 4.5), almost certainly felt (4.5 to 5.0) and significant
  (5.0 and up). Each copy carries its own literals. The synth copy says it is kept in step by hand:
  "This keeps the spoken likelihood consistent with the screen's label for the same quake". Its
  test also records that the copies have drifted before: `{4.5, "strong"} ... — matches the screen
  label (REVIEW P5 F2)` (`seismic_test.go:167`).
- Cost: moving one boundary (say USGS's 4.5 becomes 4.0) needs three edits in three layers. If one
  is missed, the radio says "moderate" while the screen says "Almost certainly felt" for the same
  quake, and every test still passes because each one pins only its own copy.
- Proposed change: one `render.FeltBand(mag) int` (0 to 3) beside `SeismicLevel`. render is already
  imported by synth (`synth/marine.go` calls `render.PlainLine`), and modes may not import domains.
  `SeismicLevel` becomes a mapping of the band, `seismicBand` keeps its labels indexed by band, and
  `feltLikelihood` keeps its words indexed by band. The words stay with each surface. Only the
  boundaries get one owner.
- Regression risk: thresholds only, with identical outputs. Pinned by `TestFeltLikelihoodTiers`
  (`domains/radio/synth/seismic_test.go:160`), `TestDetailSeismicSectionShowsGraduatedRows`
  (`modes/tty/detail_seismic_test.go`) and `TestSeismicSegmentsReadTheScript`. `SeismicLevel` has no
  direct boundary test that I found. Add a table test at 3.49/3.5/4.49/4.5/4.99/5.0 first.
- Size: S
- Confidence: high

### DR-2: The radio's spoken units are written several times in synth, and two temperature wordings already disagree
- Where: `domains/radio/synth/compose.go:268,275-277,286` (`degrees`, inline wind wording),
  `domains/radio/synth/marine.go:155` (`tempWords`), `:168` (`windWords`),
  `domains/radio/synth/seismic.go:101` (`depthWords`), `domains/radio/synth/fire.go:228,244`
  (`distanceWords`, `ringWords`)
- Observed:
  - **Temperature, metric.** `degrees` (current conditions) returns `"%.0f degrees Celsius"`.
    `tempWords` (marine water temperature) returns `"%.0f degrees"`, and its comment claims "exactly
    as the location forecast reads temperatures". That claim is false under metric. The same 23 °C
    is spoken as "temperature 23 degrees Celsius" in one report and "Water temperature 23 degrees."
    in the next (`scripts/marine-report/water.txt`).
  - **Wind.** `conditions` spells out the same two `Sprintf`s that `windWords` owns in the same
    package (`compose.go:275-277` against `marine.go:173-175`).
  - **Depth.** `depthWords(km)` is `distanceWords(km, false)` line for line.
- Cost: `TestTheConditionsSayTheListenersUnits` says the rule is "the listener's units, in one
  spelling", and the package does not hold it. A wording change (for example, dropping "Celsius")
  has to find four functions in three files. The water-temperature mismatch is audible today to any
  metric listener.
- Proposed change: gather the spoken-quantity helpers in synth into one file. `conditions` calls
  `windWords`. `depthWords` calls `distanceWords(km, false)`, keeping its own comment about why depth
  is always kilometres. `tempWords` delegates to `degrees`. That last step **changes metric output**
  ("Water temperature 23 degrees Celsius"), so it needs a HUM LEAD ruling on which wording is right.
  Related, for the same ruling: heights say "metres" (`marine.go:150`) while distances say
  "kilometers", and the map's screen-reader words say "kilometres an hour"
  (`modes/tty/map_layer_words.go:214`).
- Regression risk: `TestTheConditionsSayTheListenersUnits` (`synth_test.go:794`),
  `TestMarineReadsTheListenersUnitsButCurrentsInKnots` (checks only "feet", " m" and "knots"),
  `TestSeismicSegmentsReadTheScript` ("15 kilometers"), `TestFireWords`. The metric water
  temperature is **UNPINNED**. Pin the chosen wording first.
- Size: S (M with the ruling)
- Confidence: high

### DR-3: `app.spokenName` has never had a caller, and its tested contract contradicts the live owner
- Where: `app/cast.go:304` (dead), `domains/radio/synth/source.go:227` (live), test
  `TestSpokenNameDropsTheLanguageSuffixAndTheSentinel` (`app/cast_test.go:253`)
- Observed: both functions answer "what is this voice called when spoken or shown". The app copy
  strips `" (English (US))"` and blanks the system sentinel. It has **zero production callers**, and
  `git log -S'spokenName(' -- app/` shows it arrived with only its test in 0.14.0 (`1abf27dc`) and
  was never wired. The live owner, `synth.spokenName`, does not strip the suffix. Its comment even
  sizes the cap "for 'Aman (English (India))'". The curated voice list contains such names
  (`app/voices.go:291`: "Aman (English (India))", "Eddy (English (US))", "Reed (English (US))",
  "Tara (English (India))"), and `SayVoice.Name()` returns the full name (`synth/voice.go:61`).
- Cost: a junior reading app's test believes hand-offs say "Eddy". The live path hands
  "Eddy (English (US))" to `handoffLine` and the marquee (`source.go:363,535`). It is one question
  with two answers, and the tested one is dead.
- Proposed change: HUM LEAD rules which name is right. If the suffix should go, move the strip into
  `synth.spokenName` and delete the app copy. If not, delete the app copy and its test. Either way,
  one owner remains.
- Regression risk: deleting app's copy changes no behaviour. `app/testdata/declset.txt` lists it, so
  `TestDeclarationSetUnchanged` (`app/declset_test.go`) needs `-update-declset`. If the strip moves:
  `TestSpokenNameIsPlainBoundedAndNamesTheUnnamed` (`synth/source_roles_test.go`) pins the live
  owner, and the hand-off goldens will change.
- Size: S
- Confidence: high that it is dead and that the two disagree. Medium on what is actually heard: I
  did not play audio.

### DR-4: First-run setup has its own "same place" rule beside `snapshot.PlaceID`, which calls itself the one definition
- Where: `app/setup.go:63` against `platform/snapshot/types.go:459` (`PlaceID`) and
  `modes/tty/modal_location.go:262` (`sameLocation`)
- Observed: `PlaceID` says "The one definition; the lists never compare ZIPs themselves". It uses
  the ZIP when there is one, otherwise the 4-decimal `Key`. `applySetup` dedupes the same config
  list inline with `(l.Zip == loc.Zip && l.Zip != "") || (l.Lat == loc.Lat && l.Lon == loc.Lon)`.
  Concrete divergences:
  - Exact coordinates with different ZIPs: setup drops the old entry, `PlaceID` keeps both.
  - A ZIP-less entry at the same coordinates as a new default that has a ZIP: setup drops it,
    `PlaceID` keeps both.
  - 33.20001 against 33.2: `PlaceID` treats them as the same place, setup keeps both.
- Cost: whether a re-run of Setup keeps or drops a watched place depends on which of two definitions
  it hits. A fix to identity (#23 was one) reaches the lists but not Setup.
- Proposed change: build `LocationRef`s from both entries and compare `snapshot.PlaceID`. This is a
  behaviour change in the divergent cases, so the HUM LEAD rules which identity Setup should use.
- Regression risk: `TestApplySetupRerunPreservesCustomizations` (`app/setup_test.go:53`) pins only
  the case where both rules agree (same ZIP and same coordinates). The divergent cases are
  **UNPINNED**. Write those pins first.
- Size: S
- Confidence: high

### DR-5: The severe window's row times and its record times are two implementations of one format
- Where: `app/severe.go:519,525` (`r.At.In(in).Format("01/02 15:04 MST")`) and
  `domains/severe/record.go:30` (`stamp`)
- Observed: the same moment, in the same zone, in the same layout. The app copy keeps in step by
  comment: "a zero clock reads blank, as the record's stamp does". `stamp` also guards a nil zone,
  and the app copy would panic in `t.In(nil)`.
- Cost: honouring the Clock preference, or changing the layout, needs both edits, or the row says
  one time format and the record under it another.
- Proposed change: export `severe.Stamp` and call it from `toSevereRow`. app already imports severe.
- Regression risk: none intended. `TestSevereDeckPublishesOnlyWhenTheIndexChanges`
  (`app/severe_test.go:40`, zone not UTC) and `TestEventScriptReadsTheAlertItself` pin parts of it.
  The row's exact layout is **UNPINNED**.
- Size: S
- Confidence: high

### DR-6: The alert window is formatted to text in one package and parsed back out of that text in another
- Where: `domains/severe/record.go:88,95` (`shortDur` writes `"(~1h30m)"` into `Record.Timing`),
  `app/severe_read.go:332` (`spokenWindow` parses it back into "1 hour 30 minutes"). A third duration
  wording is `domains/radio/synth/fire.go:260` (`durationWords`: "1 hour and 30 minutes").
- Observed: the spoken read finds the window by scanning the display string for `(~` and a closing
  `)` and walking its runes. The two halves share only an unwritten text format.
- Cost: any change to `shortDur`'s shape (a space, "min", a decimal) passes `TestShortDur`, which is
  updated with it, and passes the read's test, whose Timing is a hand-written literal. Only the live
  read breaks, and it breaks silently (`spokenWindow` returns "" and the window line is dropped).
- Proposed change: carry the window as a value (a `time.Duration` on `tty.SevereRecord`, set in
  `toSevereRow` from the domain row) and word it once. Using `synth.durationWords` would add "and"
  to the spoken output, which needs a ruling. Otherwise keep `spokenWindow`'s wording but feed it
  the duration.
- Regression risk: `TestShortDur` (`domains/severe/record_test.go:40`),
  `TestEventScriptReadsTheAlertItself` (`app/severe_read_test.go:32,47`). The coupling itself is
  **UNPINNED**: no test feeds `shortDur`'s real output to `spokenWindow`.
- Size: M
- Confidence: high

### DR-7: Compass arithmetic exists in three forms, and the 16-point table in three copies, despite a declared single owner
- Where:
  - arithmetic: `platform/geo/geo.go:33` (`CompassIndex`, "the one owner of the arithmetic"),
    `domains/radio/synth/marine.go:315` (`compassPoint`, its own `math.Round(deg/22.5)`),
    `domains/severe/record.go:41` (`compass`, integer form)
  - 16-point abbreviation table: `domains/seismic/usgs/usgs.go:338`, `domains/radio/synth/marine.go:316`,
    `modes/tty/detail_marine.go:245`. `synth.bearingLong` (`seismic.go:113`) is keyed on the strings
    those tables produce.
- Observed: the 0.9.5 pass (Q6, L3-F9) made `CompassIndex` the owner, and `compassPoint` escaped it.
  I measured both forms over [0, 360] in 0.0001° steps (3.6 million points): identical. They differ
  for negative input: `CompassIndex(-20, 16)` = N, `compassPoint(-20)` = NNW. That input is not
  reachable from NDBC today, which sends 0 to 360. The swell direction is drawn by tty through
  `CompassIndex` and spoken by synth through `compassPoint`, so the screen and the radio use
  different code for one field.
- Cost: one boundary fix (for example, normalising negatives) has to be found in three places. A
  typo in any of three identical tables silently empties `bearingLong`'s lookup, which returns "".
- Proposed change:
  - (a) `compassPoint` calls `geo.CompassIndex`, after `CompassIndex` normalises negatives with
    `math.Mod(deg, 360) + 360`. That is one line, and it would otherwise regress `compassPoint`'s
    handling of negatives.
  - (b) Optional: a `geo.Compass16(deg) string` owning the abbreviation table, used by usgs, synth
    and tty. This revisits L3-F9's "consumers keep word tables". That ruling was written when only
    one abbreviation table existed. The word tables (spoken, 8-point) stay with their consumers.
- Regression risk: `TestCompassIndexMatchesTheThreeOldFormulas` (`platform/geo/compass_q6_test.go`)
  checks [0, 360) only. `usgs_test.go:141` (Bearing "N"), `TestMarineSegmentsReadTheWholeReport`
  ("from the west") and the tty marine detail tests cover single points. Negatives are **UNPINNED**.
- Size: S
- Confidence: high

### DR-8: `plaintext.ClampRunes` is re-implemented by hand four times
- Where: `platform/plaintext/plaintext.go:169` (owner). Copies: `domains/globalfeed/detail.go:82`
  (`clampProse`), `domains/globalfeed/event.go:87` (`clampID`), `domains/radio/synth/marine.go:299`
  (`capName`), `domains/radio/synth/source.go:236` (inline).
- Observed: each is `r := []rune(s); if len(r) > N { return string(r[:N]) }`. The owner's comment
  gives the reason it exists (runes, never bytes, so UTF-8 is never cut in half). globalfeed calls
  `plaintext.ClampField` on the same line as `clampProse` (`nws.go:183`), and synth already imports
  plaintext.
- Cost: four places to get the rune rule wrong. A junior meeting `clampProse` has to read it to learn
  that it is the platform function.
- Proposed change: each copy becomes `plaintext.ClampRunes(s, maxX)`. The named constants stay where
  they are.
- Regression risk: `TestClampProseBoundsHostileText` (`domains/globalfeed/detail_test.go`), and the
  48-rune cap in `TestSpokenNameIsPlainBoundedAndNamesTheUnnamed`. `clampID` and `capName` are
  **UNPINNED** for length (the change is behaviour-identical by construction).
- Size: S
- Confidence: high

### DR-9: "Is this coordinate on Earth?" is written seven times, in two forms that differ on NaN
- Where: `domains/airquality/airnow.go:133` and `domains/fire/wfigs/wfigs.go:307` (`onEarth`,
  byte-identical), `domains/locations/openmeteo/geocode.go:90`, `platform/geo/coords.go:38`,
  `platform/geo/geojson.go:215`, `platform/astro/solar.go:22`, `platform/propanswers/answers.go:41`
  (latitude only), plus `tools/reachscore/spots.go:326`
- Observed: the positive form (`lat >= -90 && lat <= 90 && ...`) refuses NaN. The negative form
  (`v[0] < -90 || v[0] > 90 || ...`, in `coords.go` and `geojson.go`) **accepts** NaN. It is safe
  today only because `ParseCoordinates` rejects NaN separately and the JSON decoder cannot yield one.
  `airnow.go`'s comment explains that the positive form was chosen for exactly this (D-248).
- Cost: a new caller copying the wrong one of seven gets a NaN place. platform/geo already owns
  coordinates (`ParseCoordinates`, `LatLon`).
- Proposed change: `geo.OnEarth(lat, lon float64) bool` in the NaN-refusing form, called from each
  site. airquality, wfigs, openmeteo and astro gain an import of the leaf `platform/geo`.
- Regression risk: `TestARowThatIsNoPlaceOrNoNumberIsLeftOut` (`airnow_test.go:122`),
  `TestARingsVertexThatIsNoPlaceIsLeftOut` (`contours_test.go:161`), `TestNaNCoordinatesAreRefused`
  (`geo/coords_test.go:12`), `TestTheGeocodersTextIsCleaned` (`geocode_test.go:170`),
  `TestSunTimesRefusesBadInput` (`astro/solar_test.go:47`). The fuzz tests in airquality and wfigs
  call the local `onEarth` and would switch to `geo.OnEarth`.
- Size: S
- Confidence: high

### DR-10: The map view's width in km is computed twice, identically, for the one namer
- Where: `modes/tty/map_boxes.go:1014` (`mapTitleAt`), `modes/tty/map_describe.go:150` (`viewName`)
- Observed: both are `(v.E - v.W) * 111.32 * math.Cos(centre.Lat*math.Pi/180)`, passed to
  `cfg.MapAreaName`. One names the title, the other names the spoken description, and they must name
  the same area. The km-per-degree constant is also written as `111.32` (here and
  `usgs.go:136`) and `111.0` (`fire.go:111`), while `geo.EarthRadiusKM` implies 111.19.
- Cost: a change to the width rule (for example, measuring at the view's south edge) applied to one
  copy makes the title and the screen-reader description disagree about where the map is.
- Proposed change: one tty helper `viewWidthKm(v, centre)`, or `geo.Box.WidthKmAt(lat)`, plus a
  `geo.KmPerDegree` constant. Choosing 111.19 against 111.32 changes the namer's scale thresholds
  very slightly, so keep 111.32 unless the lead wants it.
- Regression risk: `TestTheTitleNamesWhatIsInView` (`modes/tty/map_uat1_test.go:133`) pins the title
  through its width thresholds. For the description, `TestTheDescriptionHasOneName`
  (`map_names_test.go:45`) exists, but I did not confirm that it exercises the width.
- Size: S
- Confidence: high

### DR-11: `render.Plain` and `render.PlainLine` are aliases of `plaintext.Text` and `plaintext.Line`, and both names are in use
- Where: `platform/render/text.go:249,407`. Call sites: `render.Plain` 48, `plaintext.Text` 5,
  `render.PlainLine` 24, `plaintext.Line` 16. Four domain packages import render, a UI package, only
  for these: `domains/radio/script/script.go`, `domains/radio/synth/marine.go`,
  `domains/severe/record.go`, `domains/severe/detection.go`.
- Observed: one operation with two public names. In synth, `source.go` calls `plaintext.Line` and
  `marine.go` calls `render.PlainLine` for the same job.
- Cost: a junior has to open both to learn that they are the same. Domains depend on the renderer to
  sanitise text.
- Proposed change: the four domain files call plaintext directly (behaviour-identical). Retiring the
  render aliases across modes is optional and mechanical. Keep them if the lead values "modes talk
  to render" as a convention.
- Regression risk: none; the aliases are one-line delegations. plaintext's own tests pin behaviour.
- Size: S (domains only), M (retire the aliases)
- Confidence: high

### DR-12: package `app` has two byte-identical "n missing values" helpers
- Where: `app/history.go:484` (`missingRow`), `app/maptemp.go:608` (`missingValues`). A third is
  `domains/temperature/temperature.go:152` (`missing`).
- Observed: the same seven lines and the same question, twice in one package.
- Cost: small. It is the clearest case of "which one do I call?" in the tree.
- Proposed change: delete `missingValues` and call `missingRow` (5 calls at `maptemp.go:565`). Leave
  temperature's private copy, since a cross-package export for seven lines is not worth it.
- Regression risk: behaviour-identical. Exercised by the maptemp and history tests. Needs
  `-update-declset` for `app/testdata/declset.txt`.
- Size: S
- Confidence: high

### DR-13: `Series.hourIndex` re-implements `hourRow`, which its two siblings already share
- Where: `domains/temperature/temperature.go:165` against `domains/temperature/rain.go:165`
  (`hourRow`, used by `Rain.hourIndex` and `Measure.hourIndex`)
- Observed: the same operation, finding an hour and otherwise inserting it in order with a row of
  NaN beside it. `Series` needs six parallel row slices where the others need one, so it wrote its
  own two-pass copy.
- Cost: a fix to the hour rule (truncation, zone, ordering) made in `hourRow` does not reach the
  temperature series, which is the one the map uses most.
- Proposed change: `hourRow` takes `rows ...*[][]float64`, and `Series.hourIndex` passes its six.
- Regression risk: `TestAnHourGoesInItsPlace` (`domains/temperature/hourindex_test.go:10`) pins
  Series insertion and the rows moving with their hours. The waves tests pin `Measure`.
- Size: S
- Confidence: high

### DR-14: `cast.Class` repeats its out-of-range fallback in four accessors, which the project already collapsed for `Slot`
- Where: `domains/radio/cast/tone.go:147,158,167,176` (`ToneRank`, `Key`, `String`, `ToneName`)
- Observed: each method is `if !c.valid() { return classes[ClassWarning].x }; return classes[c].x`.
  `ToneRank`'s comment is about exactly this risk: "Two answers for one question, agreeing by
  accident". The ledger records the same shape collapsed into `Slot.row()` for lineup.
- Cost: a fifth accessor can pick a different fallback, which is the failure the comment names.
- Proposed change: `func (c Class) row() classRow` with the Warning fallback, and each accessor
  returns a field of it. `ToneRepeats` keeps its own rule (1 when unset).
- Regression risk: `TestToneNameNamesThePresetAndIsLoudOutOfRange` (`cast_test.go:244`) pins the
  out-of-range answers.
- Size: S
- Confidence: high

### DR-15: The Broadcaster's card box draws its top rule with a look-alike of render's box rule
- Where: `modes/tty/broadcaster.go:1665` (`boxRule`) against `platform/render/panel.go:133`
  (`boxRule`, behind `BoxTitled`)
- Observed: both draw a top border with a title at the left and a tag at the right. The tty copy says
  it is "in the masthead's shape", but it opens with three rule marks where the masthead has two,
  and its give-way rules differ (it drops the badge first; render drops the stamp, then the title).
- Cost: the reuse-control-components rule (D-147) asks for the app's own component, not a
  look-alike. The comment misleads a reader about which shape is drawn.
- Proposed change: ask the HUM LEAD whether card boxes should match the masthead. If yes, export a
  render rule taking a tinted title and a badge, and use it (golden change). If no, correct the
  comment to say the card's rule is its own.
- Regression risk: `TestTheBoxsRuleCarriesTheGrade`, `TestACardFillsItsBoxAtEveryWidth`
  (`modes/tty/broadcaster_cardbox_test.go`) and the Broadcaster goldens.
- Size: M (S for the comment)
- Confidence: medium. I did not judge whether the 3-mark rule was a deliberate reference match.

### DR-16: Three "host of a URL" functions with three edge behaviours
- Where: `app/maps.go:246`, `domains/temperature/quota.go:90`, `platform/httpx/httpx.go:340`
- Observed: app returns `""` for a URL with no host. temperature returns the input. httpx slices
  strings and keeps any userinfo and port. app and temperature both import httpx already.
- Cost: low today, because every URL involved is well-formed. Status rows are matched by host, so a
  difference here would show up as a missing row rather than an error.
- Proposed change: export one `httpx.HostOf` (the `url.Parse` form) and use it in all three.
- Regression risk: **UNPINNED** directly. The Status window tests exercise app's copy indirectly.
- Size: S
- Confidence: medium (I did not trace every Status join)

---

## Minor

- `modes/tty/map_prop_ahead.go:186` `hoursWords` is byte-identical to `modes/tty/map_prefs.go:345`
  `radarAheadLabel`. Keep both only if a settings label may diverge from prose.
- `modes/tty/map_layer_words.go:212` writes `kmh/1.609344` inline, while `platform/units` says it is
  "every unit conversion the app makes, in one place" (`units.MilesOf`).
- `modes/tty/broadcaster_lineup.go:187` converts km to miles, and the caller converts straight back
  (`broadcaster_pool.go:52`, `units.KmOf(*mi)`). Return km.
- `domains/severe/record.go:288` `abs(float64)` duplicates `math.Abs`.
- `domains/fire/fire.go:106` `Age` has no callers anywhere, tests included.
- `app/ticker.go:583` and `domains/radio/synth/source.go:644` `sleepCtx`, plus
  `domains/radio/player/engine.go:638` `Engine.sleep`, `platform/sched/sched.go:207` `wait`, and four
  inline selects: one 6-line idiom with no shared home. Leave it unless a platform package is
  touched anyway.
- `modes/tty/detail_fire.go:273` `plural(n)` exists, yet `n == 1` is inlined at `map_pane.go:761`,
  `map_prefs.go:346`, `map_prop_ahead.go:187`, `map_prop_update.go:320` and `map_boxes.go:295`.
- Six settings lookups hand-roll `for _, c := range choices { if c == v ...}`. `slices.Contains`
  reads plainer: `map_prefs.go:232,256,335,361`, `map_depth.go:33`, `map_prop_settings.go:26`.
- Ledger row `7dbf0b6d0c66` (`Director.stepBed · stepOperator`) in `06_docs/duplicates-ratified.md`
  matches no group any more: `stepBed` gained `CutOver`, so the shapes differ. `tools/dupes` has no
  stale-row report (`tools/wires` has one), so stale exemptions accumulate silently.
- Tools: the "which trees do we read" skip lists differ. `tools/dupes/main.go:204` and
  `tools/authoring/main.go:41` skip `.git`, `third_party`, `mutants` and `dist` but not `.claude`, so a
  worktree under `.claude/worktrees` doubles every function in `make dupes`. `tools/wires` skips all
  dot-directories, and `tools/plancode` only `.git`. Separately, `samples.csv` is parsed twice
  (`tools/perfsum/main.go:227`, `tools/slope/main.go:122`), and `readLedger` exists in both dupes and
  wires.
- Observer's location window words an age three ways: marine "39m  0s ago" (`detail_marine.go:87`,
  `fixedAgeTrim`), seismic "39m ago" (`detail_seismic.go:167`), masthead "39 Minutes Ago". This is a
  UX ruling, not a consolidation: seismic must span a week.

## Test helpers (lower priority; not metric D)

- `func f64(v float64) *float64 { return &v }` and its kin (`fx`, `pf`, `fp`, `f64ptr`, `ptr`) appear
  in 13 test files across 11 packages. `go.mod` is `go 1.26`, so `new(1.5)` replaces every one with
  no helper at all.
- `equalStrings` / `same` / `equal` (`app/executors_test.go:633`, `platform/category/category_test.go:133`,
  `platform/lineup/lineup_test.go:28`) are `slices.Equal`. `has` / `containsExact` are
  `slices.Contains`. `indexOf` / `indexIn` are `slices.Index`. `keys` (`cmd/watchpost/gates_test.go:678`,
  `platform/lineup/merge_property_test.go:247`) is `slices.Sorted(maps.Keys(m))`.
- `fixture(t, name)` in coops, ndbc, nws, radar and temperature tests, and `client(t)` in globalfeed
  and usgs, are per-package by necessity (test helpers cannot be shared without a testutil package).
  Leave them.

## Considered and rejected (they answer different questions)

- `app/eventrecord.go:255` `digest` and `synth/fire.go:163` `contentKey`: byte-identical SHA-256
  prefixes. But one names history **on disk** (changing it orphans stored history) and the other is a
  throwaway audio-cache key. One change must not apply to both.
- Four identical `Getter` interfaces (airquality, uv, radar, temperature): Go's consumer-defined
  interface idiom. Sharing one would couple packages for nothing.
- `cast.Classify` (tone class) against `severe.Classify` (window tab): both read product names, with
  different outputs and a deliberately different rule order.
- `app.severityOf` (library enum), `severe.severityOf` (colour tier), `tty.severityLevel` (sort
  rank): three questions about one input.
- `broadcaster_manifest.go:66` `dataAgeTone` against `render.FreshnessTone`: ruled separate, and
  `freshness.go` says so.
- Observer/Broadcaster pairs (`clock`, `heldCursor`, memo `stats`, `armTick`): accessors on two types,
  already sharing `armShimmer` and `memoStats.counts()`. Their comments cite the ratified accessor
  ruling. No accidental copies between the two surfaces surfaced, even at an 8-node floor.
- `platform/category` and `platform/report` `all`/`Of`/`All`: two registries modelled on one another
  on purpose.
- `radar` and `temperature` `Hosts`/`ClientConfig`/`NewClient`, and provider constructors
  (`NewHRRR`, `NewNDFD`, `NewEPA` ...): per-domain configuration with the same shape.
- Per-setting `*ByKey` + label + cycle families in `map_prefs.go`: each setting is its own question.
  A settings framework would be the new abstraction layer the brief warns against.
- `player.backoff` against `httpx.sleepBackoff`: the same jitter formula, but different layers and
  caps, and no shared home short of a new package.
- `seismicAge` / `agoWords` / `longAge` / `fixedAge`: different slots and ranges (`shortAgo` is
  already derived from `agoWords`).
- `tty.distanceWords` against `synth.distanceWords`: screen-reader words against radio words, and
  modes may not import domains.
- Observer's `radioBP` against `term.BreakpointFor`: known F-68, with differently ruled boundaries.

## What is in good shape

- Display width and truncation: every measure goes through `render.Width`/`TruncateCells`.
  `runewidth` and raw escapes do not appear outside `platform/render` (except `view.go:481`, which
  builds its SGR from render tokens).
- Distances: every haversine call goes through `geo.HaversineKM`. Unit conversions are in
  `platform/units` (one stray constant, Minor).
- Memos: `agememo` and `bodymemo` are the single owners, and `app/lazymemo.go` builds on them.
- Map layers register through one `registerMapLayer`. Credits and Status host lists are
  test-guarded (`app/credits_sources_test.go`).
- The masthead, API summary and shimmer arming are shared between the surfaces by design and say so.
- Metric D at the gated floor is honest: the two groups it reports are correctly ratified.

## What I did not cover

- `third_party/` was not swept.
- I did not run the app or hear audio (DR-3's audible effect is inferred from code).
- I did not run `go run ./tools/dupes -tests` beyond reading the ledger's count (17). The test
  clusters above came from my exact-body sweep.
- I read the groups at 12 nodes and above closely. The 8-node sweep was used only to look for
  copies across the two surfaces.
- I did not open every as-built map to confirm that no consolidation above conflicts with a ruling I
  have not seen. DR-7(b) and DR-15 explicitly ask for one.

## Observed outside this lens (for the regression lens, unverified)

- `modes/tty/broadcaster.go:578`: the Broadcaster's `render.Opts` sets only Width, ASCII and Frame,
  with no `Units` and no `Clock`. 0.16.0 FR-6.2 says units and clock are **shared** between the
  surfaces, and the pool table renders temperatures and distances through `Opts`. A metric user may
  therefore see °F in the Broadcaster. I did not run it, and I did not find a ruling that exempts
  the console.
