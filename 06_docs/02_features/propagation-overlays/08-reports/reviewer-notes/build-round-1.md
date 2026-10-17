---
title: "0.19.0 BUILD exit red team, round 1 - reviewer notes"
date: 2026-10-17
phase: BUILD exit
sev: SEV-0
authority: HUM LEAD
status: "Eight blind reviewers (D-203). Each reviewer's findings, condensed from its report; every finding cites the file and line it read at watchpost 5012fae7, go-ionomaps d7e7f3c, go-tuiMaps b50fc86."
---

# BUILD exit - red team, round 1: the reviewers' notes

Up: [the report](../red-team-build.md)

## code

```text
Code Quality — Do not exit BUILD. Fix first: C-19.
C-1 Important: D-131 cited for single-flight join (propagation.go:4, map_prop_update.go:34, app/mapprop.go:83) — D-131 is memory/RSS ruling.
C-2 Important: Service join can hand reopened mode the cancelled result of previous session (propagation.go:111-135 + map_prop_update.go:88-97): stop cancels, reopen asks, Ask joins still-running cancelled run, gets Canceled, status hides it, next tick 10m/1h/never. Untested combo. Delete Service.running/run/join (Library.Update serialises under l.mu; propBusy per session). Add leave->re-enter test.
C-3 Important: 37 prop* fields loose in mapPane (map_pane.go:65-107); stopPropagation resets hand subset. Group in propSession struct.
C-4 Minor: propAnswer enum + armAnswer switch + wrappers (map_prop_chart.go:79-112) — inline.
C-5 Important: app/mapprop.go:97-127 builds Library with readOnly fetcher just to read Sources (ignores receiver, terms.All); propagationHostList test-only; export ionomaps.Sources().
C-6 Important: P10-04 functions >60 lines not caught: dispatch dashboard.go:1151 (80), modalKeyFor memo.go:326 (71), setupChips setup_layout.go:449, overlayRows map_boxes.go:545 (68), setupSpace setup.go:494 (63), tools/atlas render :77 (75), go-ionomaps loadIGRF tools/tables/igrf.go:31 (82). Suspect diff-scoped funlen blind spot.
C-7 Important (HUM fork): code reshaped for P10 analysers (newLibrary propagation.go:70-72; history Last/Forget/Do; visits==nil unreachable guards history.go:1380,1532,1570; ~45 single-use guard vars go-ionomaps). Fix analysers or ratify pattern.
C-8 Important: go-ionomaps root P10-05 row (p10-ledger.md:23, D-56, 23 funcs) now covers answers/lower/reach/offset/openhours/stations unrenewed; exemption_test.go:32-34 passes when regex stops matching. Re-measure; HUM; fail closed.
C-9 Important: go-ionomaps scripts/gate:172-193 logs green when P10 not run (log_run 0); zero fuzz targets passes in full mode (:154-155).
C-10 Minor: record_test ruling-citation check doesn't verify line exists: D-169 status.go:975 (833 lines), D-201 app/app.go:190 (185 lines), 7 bare filenames.
C-11 Important: go-ionomaps Snapshot.Hours documented up to 25, always 1 (update.go:320); clone drops Typical; API change (HUM) or fix doc+clone.
C-12 Minor: history names stale (yearDoc, pruneYears, seriesFiles.year compat undocumented end).
C-15 Important (HUM fork): go-tuimaps CheckedDialerFor/ProxyFunc no consumer, duplicates watchpost httpx.ProxyExempt; Class.Shade no consumer.
C-16 Important: go-tuimaps scripts/gate:457-468,:651-659 v0.0.0 replace blocks dead; comment false (nested modules require v0.2.0).
C-17 Important (HUM, API): go-tuimaps badColours (internal/overlay/grid.go:52-55) returns UnsortedBreaks for wrong colour count.
C-18 Important (HUM, API): go-tuimaps twoClassHost (field.go:178-186,:268) — any 2-class host type becomes edge mark; should be explicit host request.
C-19 Important: watchpost go.mod:10 pins go-tuimaps 9dc4ff3 not b50fc86 (same as I-1).
C-20 Minor: go-ionomaps dead exports limits.ReceivedPower, magcoords.MLT, Field.CellCentre, Bitset.Set mutable.
C-21 Minor: watchpost map_prop_draw.go:137 hard-codes band edges; go-ionomaps Band.LowMHz/HighMHz unused.
C-22 Minor: go-ionomaps lower.go:89 Noise->Environment relies on enum order; test misses Rural.
C-23 Minor: AP-HIST-01: bench_test.go:285 "0.19.0 batch 22:", map_prop_honesty_test.go:3 "batch 25:", go-ionomaps export_test.go:6-7, go-tuimaps specimens README:20, specimen_test.go:56, sixteen.go:166 "as before".
C-24 Minor: go-ionomaps giro/asker.go:62-66,:91-94 burst failures dropped silently.
C-25 Minor: go-tuimaps FoF2 as last-preset marker in 5 places.
C-26 Minor: go-tuimaps checkHostRampLocked inline cap, only at Set, duplicates per Set.
C-27 Minor: go-tuimaps basemap.go:526 ownCanvas discards NewCanvas error (P10-07).
```

## infosec

```text
InfoSec — Do not exit BUILD. Fix first: I-1.
I-1 Important: watchpost go.mod:10 pins go-tuimaps 9dc4ff3, not b50fc86 (P5/P6 unbuilt in watchpost); go-ionomaps untagged pseudo-version (go.mod:9); build-log:339 says 9dc4ff3 pinned. Action: tag/pin reviewed tips, rerun vuln + gate.
I-2 Important: resolver.go:175-190,:163, geocode.go:68 — coordinates in unrecognised forms (33.2N 117.4W, 33°12'N, 33.2;-117.38, three fields, 10-char locator, locator typo) reach Open-Meteo via ResolveAnywhere->ResolveWorld; FR-4.9 says positions never go online. Action: refuse position-shaped queries locally; add tests.
I-3 Minor: children.go:146-152 — failed End drops to default, record removed, orphan never retried; dashboard.go:78 discards swept count; EndChildren count ignored; fake End always nil. Action: keep record on failure; diagnostics counts; failing fake.
I-4 Minor: children.go:178-180 marker substring not program; ps via PATH; check-then-kill PID reuse gap; non-orphan records kept forever (149). Action: comm == say, /bin/ps, recheck start before kill, age out records.
I-5 Minor: go-ionomaps glotec.go:102 unmarshals all features before 5184 check (:115) — ~110 MB from 8 MB hostile body. Action: stream decoder / tighter cap.
I-6 Minor: propagation.go:163-167 host allowlist ignores port/userinfo. Action: require no User, port empty/443.
I-7 Minor: go-tuimaps CheckedDialerFor has no consumer; watchpost keeps httpx.NewProxyExempt (maptiles.go:40); socks5h missing in go-tuimaps default ports. Action: adopt or delete; add socks5h.
I-8 Minor: Makefile:556 make hooks overrides core.hooksPath silently, disabling other hooks (gitleaks). Action: refuse/warn or chain.
Clean: injection, error leaks, govulncheck x3.
```

## a11y

```text
Accessibility — Do not exit BUILD. Fix first: AX-1.
AX-1 Critical: Observer origin = map centre named by viewName() (map_prop_origin.go:52-55; map_describe.go:140-153 falls back to mapPane.region.Name never cleared in prop mode; app/mapnames.go:34-37 "" outside region; map_prop_centre.go:58-59). Centred on Tokyo words say "The contiguous United States, now: ..." ; after a find "Found Tokyo" then US region name; with selected place in no region "the map". Breaks D-64/D-193 worldwide. Fix: name centre by found place, else nearest world-index city, else lat/lon words; test panning outside every region.
AX-2 Important: spoken() adds megahertz to every decimal: found coordinates label "33.2000, -117.3800" -> "Found 33.2000 megahertz, -117.3800 megahertz"; spoken("1.5 hours") -> "1.5 megahertz hours" (map_prop_words.go:264-282; A-57(3) premise false). Drop catch-all; unit at format site; coordinates as degrees.
AX-3 Important: hour ahead reads go-ionomaps TypicalError.Basis citation (map_prop_ahead.go:171-173; go-ionomaps ahead.go:48 "...PLAN's week... (watchpost D-134, D-181; ...week.py section 11, WEEK_TAKE_UNKNOWN=1)"); fixture basis empty. Listener-facing basis separate.
AX-4 Important: typical open hours (FR-1.15) only glyphs (map_prop_chart.go:412-447); words never say hours; --ascii chart still draws ● ◐ ○ ◌ · ▸ ▼ (:247-253,:285). Say spans; ASCII glyph set.
AX-5 Important (UX fork): refresh/hour step moves text under fixed scroll row (focus-limit dropped, updated-at added, hour-ahead inserted; map_prop_words.go:47-49,116-119,39-41); TestARefreshKeepsTheReadersPlace checks integer only; age ticks per minute rewrapping status. Fork: anchor scroll to sentence / stable line count / age only on change.
AX-6 Important: Help doesn't say hour stepping in prop mode (map_pane.go:1445 "space, ⇧←, ⇧→, n Play / Back / On / Now"); FR-1.14 requires; screen reader reads arrows glyphs; rows map keys to names by position (:1446); chart window has no controls line (map_prop_chart.go:268-321). Fix: hour row in prop group; key:name pairs; chart controls line.
AX-7 Important (UX fork): repetition: ~31 lines; 5 continents identical ~45-word lines; station clause on 7 rows + status; weather heading + night line before mode named (map_describe.go:64,130-136). Fork: group identical targets; station once; drop/move weather heading.
AX-8 Minor: abbreviations not expanded (foF2 when MUF drawn, LUF, M(3000)F2, GloTEC, D-RAP); centre words upper-case names (map_prop_centre.go:98); night line 08:00 UTC vs 1:50 PM (map_night.go:93 vs map_prop_words.go:94).
AX-9 Minor: failed answer said "being worked out" forever (map_prop_chart.go:153-164 errors to diagnostics; askAnswer won't re-ask :143-151; words :140-141; chart :269-270; reach map_prop_freq.go:121-122). Say "not available; U tries again"; clear asked key.
AX-10 Minor: failed find wrong fix: world search told "try 'City, ST' or a zip code" (geocode.go:77); network failure "change it and press Enter" (map_prop_find.go:56). Mode-aware hint; offline route.
AX-11 Minor: Esc doesn't cancel find in flight; late answer marks and recentres (map_prop_find.go:90-91,108-127). Bump propFindSeq on Esc.
AX-12 Minor: badges lit/dimmed colour only (map_prop_draw.go:294-317; map_parts.go:57-69); "The red circle" at every depth (map_zone.go:121).
AX-13 Minor: night words "shaded" in prop mode but go-tuimaps never shades field colours (frame.go:916-917; map_night.go:93).
AX-14 Important: watchpost pins go-tuimaps 9dc4ff3 (= I-1/C-19); watchpost never calls ColourDepth -> with b50fc86 terminal decides; at pinned version 16/256-colour terminal gets truecolor ramp. Pin b50fc86; pass depth?; golden at 16 colours.
```

## perf

```text
Performance — Do not exit BUILD. Fix first: P-3 (then P-1).
P-1 Important: Observer origin = map centre; every 0.5° pan re-asks workPropChart (+Reach with freq) (map_pane.go:477, map_prop_origin.go:52-53, map_prop_chart.go:67-74,169-196); chart cost mostly typical-day open hours (answers.go:175,186,203,214; openhours.go:74-96) which block never shows (only chart window); same for centre (map_prop_centre.go:74-86); superseded asks not cancelled (map_prop_chart.go:116-149); held arrow 30 Hz w/ freq ~ >1 core, ~150 MB/s garbage; G1 risk. Fix: open hours only when chart window open; memoise by (day, F10.7, origin, target, radius, noise); ask on settle; cancel superseded.
P-2 Minor: openHours takes [24]typicalHour by value (345 KB heap per place) openhours.go:74,77; closures copy 14.4 KB (openhours.go:110, lower.go:432-433). Pointers.
P-3 Important: Propagation frame misses go-tuiMaps M5 2x on UI goroutine: 400x110 field+reach+night 33.6 ms vs 16 ms target; night blend math.Pow per cell (internal/colour/blend.go:18-29) ~33% CPU (D-96 named; P7 unbuilt); reach edgeMark/edgeAt 15.8 ms; night always on in prop mode; Render in Update (map_pane.go:446); gate bench BenchmarkGlobeFieldPan (globe_bench_test.go:31-69) lacks night/terminator/reach. Fix: memo blend per (ink, night/zone); classify each dot once; bench in M5 shape.
P-4 Important: targets unmeasured/stale: FR-1.17 req:85 cites 4.4 ms placeholder, comment map_prop_chart.go:168 says ~5 ms, measured ~25 ms chart / 8 ms centre; G1 (W10.2) never measured; go-ionomaps R-8.1/G-G1 no Update bench / alloc gate; R-8.1 "keeps day's climatology hours" untrue (Ahead 35 ms rebuild ahead.go:22-37; typicalDay per answer; build remakes 34 ms background even unchanged grid update.go:293-298; watchpost discards hours ahead on every result incl too-soon same Computed map_prop_update.go:111). Correct FR-1.17; run W10.2; G10 gates; cache climatology hours; deferral needs HUM ruling.
P-5 Minor: every Update (incl 20 Hz viz tick) runs prop arm calls (dashboard.go:1119-1122); propCentreAt locks map mutex (map_prop_centre.go:47); armAnswer builds closures w/ snapshot copies regardless (map_prop_chart.go:100-109); modalKeyFor formats propStatusWords each View. Early return; closure only when asking; key memo on age minutes.
P-6 Minor: centre box checks only computed (map_prop_centre.go:95) -> stale answer after hour step / noise change / station change. Compare c.key with propCentreWanted().
Benchmarks: Bands 4.6 ms/1.5 MB, Path 6 continents 20.2 ms/3.2 MB, centre path 8.0 ms, Reach 28 ms, Ahead 35 ms, typicalDay 1.2 ms. No unbounded growth found.
```

## docs

```text
Docs — Do not exit BUILD. Fix first: D-F3.
D-F1 Important: requirements contradict later rulings/code: FR-1.8 (req:76) selected place vs D-193 centre (map_role.go:33); req:22,:32,:99 Broadcaster map not in 0.19.0 (D-77) vs FR-2.1/2.4/D-195; REQUIRED-READING:41 rule 3; F-174 follow-ups:126; FR-1.15/glossary miss "weak". Amend under one ruling row; close/rescope F-174.
D-F2 Important: = B-F4 (4 missing tests; build-log:262 "written first"; W4.2 parts not in D-201 list; trace check plan-only); build-log:328 TestEveryUIModeHasAPropagationRole/propRoles gone (now mapRoles). Map names; check existence.
D-F3 Important (HUM fork): Reach ignores lower limit: go-ionomaps reach.go:26-29,14-17 "upper limit only until the lower limit lands (D-175)"; lower limit landed (G6.2b/c); R-2.9 & FR-1.16 promise more; map_prop_freq.go:117-129 can say "no skip zone" for band chart calls absorbed. Rule: apply lower limit to Reach, or record gap D-row+follow-up+say in words.
D-F4 Important: architecture.md "go-ionomaps' shape (signatures only)" stale (:137 Options.Tables; :144-146 UpdateOptions w/o Noise; :210-212 Bands/Path/Reach types; :225 NearestKm; Snapshot fields; :152 Hours 24 ahead vs 1 since D-60; :69 chan type); go-ionomaps design.md:12,:64. Delete hand-copied sigs, point to godoc; fix snapshot.go:10-14.
D-F5 Important: file:line drift in normative docs: FR-6.5 history.go:879; FR-1.10 resolver.go:101-102; architecture.md:68 map_temp.go:167; :69 dashboard.go:114, map_prefs.go:556; check only presence (A-60(2)). Re-point; resolve files/lines.
D-F6 Important: go-ionomaps README:7-9 "in planning. No code yet."
D-F7 Important: go-tuimaps README:61 (only NO_COLOR), :62 (colour alone), :74 UA 0.2.0 vs fetch.go:26 0.3.0; contract.md:199-204 colour-alone "owed to v0.3.0" vs §11/§12; README lists only v0.2.0 breaks.
D-F8 Important: = I-1 pin 9dc4ff3; go-tuimaps build-log:89 claims watchpost draws 256 on xterm-256color; REQUIRED-READING:22; as-built-map.md:113 rc.1. Bump, run, announce UAT (colour change HUM's call).
D-F9 Minor: = P-4 FR-1.17 4.4 ms placeholder; dry-run rests on spike 12.7 ms/hour.
D-F10 Minor: INST-5: typical error basis internal refs (= AX-3); offset.go:18 same; go-tuimaps release-notes:24 0.011° without basis; L-4.2 requirement "0.5°" vs build-log:65.
D-F11 Minor: as-built-map.md:108 "8 MB cap" (per-host now), :111 go-ionomaps GloTEC+solar only, front-matter Batches 1–138 v0.2.0.
D-F12 Minor: REQUIRED-READING:22 550-word history cell; "W4.5/W4.6 awaits mock-ups" false (built batch 25); go-tuimaps requirements:65,:96 "Only alert roles fill an area today"; rulings.md front-matter phase: PLAN; requirements status stops at D-127. Cut Phase row.
D-F13 Minor: memo_positions_test.go:68 floor 12 vs 13 counted. Raise to 13.
D-F14 Minor: go-ionomaps NOTICE items 5/6 contradict; P.372/P.371 unlisted.
D-F15 Minor: go-tuimaps build-log:90 fire first-run pass unexplained, settled on agent's own reason.
```

## hygiene

```text
Hygiene — Do not exit BUILD. Fix first: H-1 (with H-2).
H-1 Important: = I-1 pin 9dc4ff3; no watchpost verify on b50fc86 changes (L-7.1, L-7.2 depth, field/basemap, CheckedDialerFor).
H-2 Important: go-tuimaps 30 ahead / 2 behind main (v0.2.1 2313ed7, #6 043922e); merge-tree conflicts 10 files (gate.yml, fetch.go, basemap.go, repo_test.go...); nested modules require v0.2.0 (main v0.2.1); watchpost behind main by 8465dc0c (0.18.1); CHANGELOG.md:5 no 0.18.1 entry. Merge main into both before re-pin.
H-3 Important: go-ionomaps p10-ledger.md:21 internal/limits row "RATIFIED ... as A-12, batched for veto" while D-200 says A-7..A-14 open; header says all ratified. Put A-7..A-14 to HUM; mark pending.
H-4 Important: = C-8 root row (D-56, 23 funcs, now 84); "Reviewed at G9" trigger exemption_test.go:21 dead; internal/terms row (:16) cites Sources() renamed All().
H-5 Important: go-ionomaps evidence logs (plan-dry-run README:14-15 logs/requests.log, w00-requests.log) gitignored (*.log), untracked; D-141 cites. Add !06_docs/**/*.log; commit logs.
H-6 Important: READMEs stale: go-ionomaps README:7; watchpost README:498-499 Go 1.25.13 floor / go-tuiMaps v0.2.0 vs go.mod 1.26.9; go-tuimaps README:115 gate ~30 min vs ~49.
H-7 Important: go-ionomaps no tags, no 07-readiness checklist; pinned pseudo-version on feature branch (unreachable after squash+delete); plan says v0.1.0 rcs (D-3). Write checklist, tag v0.1.0-rc.1, pin.
H-8 Important: watchpost verify leaves no record; batch 32 committed untracked pre-push; HOSTED CI FAILED make race both OSes (run 38992546746); batch 32 log silent. Log verify runs; record hosted failures.
H-9 Minor: prepush tests working tree not pushed commits; relative core.hooksPath breaks in linked worktree; stale binary; make hooks undocumented.
H-10 Minor: libraries' full hosted gates never run on branches (go-ionomaps zero dispatch; go-tuimaps nightly main only); checklist row 0.4 not "full".
H-11 Minor: go-ionomaps gate -fuzztime 30s (time budget false failures) and govulncheck -version check vs go-tuimaps fixes.
H-12 Minor: follow-ups closing rule "delete" broken (F-215, F-184, F-188, F-187, :131-145).
H-13 Minor: F-216 due 2026-10-19: ci.yml ubuntu-latest, Node 20 actions.
```

## business

```text
Business — Do not exit BUILD. Fix first: B-F1.
B-F1 Critical: Observer's finder makes found place the ORIGIN (recentre; D-193) not a TARGET; propFound never enters chart targets (map_prop_find.go:5-6,123; A-63(3) rulings:294); FR-1.9 (req:77) says found places are answered beside continents as targets; UAT protocol (uat-protocol.md:35) M5 targets drawn from found places; Broadcaster does treat it as target. FORK: (a) found place a target row in Observer's chart, or (b) amend FR-1.9 + protocol.
B-F2 Important: PS-1 rests on assertion (problem-statement:16-18,57,63; D-68 all stakeholders HUM LEAD; D-45 §97.113 occasional; D-76 harm left to follow-up). Action: BUILD/VALIDATE reports say PS-1 addressed in part; HUM LEAD records evidence or accepts hypothesis.
B-F3 Important: no number for any primary metric at BUILD exit; D-201 moved W10/G10 beside REVIEW without amending D-118/D-132 "in BUILD"; D-75 floor miss would cut hybrid that REVIEW reviews; G10.1/G10.2 are short deterministic tests. Action: HUM LEAD amends D-118/D-132; report says floor verdict outstanding; build G10.1/G10.2 in BUILD.
B-F4 Important: 4 requirement-named tests have no func: TestTheOriginIsTheSelectedPlaceOrTheTower, TestEveryRegisteredSourceHasACredit, TestNoRawSourceDataIsStored, TestAnUnchangedFeedCostsA304; build-log:262 says first was "written first"; W4.2 part dropped without ruling though REQUIRED-READING says list built; record check docs-vs-docs only. Action: write TestNoRawSourceDataIsStored; map others (TestTheOriginFollowsTheUIMode, TestEveryPropagationSourceIsCredited, go-ionomaps TestAnUnchangedIndexAsksForNoGrid) or write; extend record check to grep funcs; fix build-log:262.
B-F5 Important: requirements state overturned rules: req:32 (Broadcaster no map, D-77) vs D-195; FR-1.8 selected place vs D-193 centre; follow-ups:126 F-174. Action: amend FR-1.8, FR-1.11, line 32, F-174 under one A-row.
B-F6 Important: go-ionomaps README:21-23 promises GIRO-free mode; Options only Clock (ionomaps.go:44-46); watchpost always fetches GIRO; ack names GIRO but not non-commercial terms. FORK: add skip-stations option (Setting?) or correct README + G-M1.
B-F7 Minor: nothing measured whether GIRO assimilation changes any band call; W10.3 should also score background-alone.
Notes: default refresh 10 min (19 MB/h) vs hourly (3.3 MB/h) is a HUM LEAD fork (not a finding).
```

## lens

```text
BUILD lens — Do not exit BUILD (no Critical). Fix first: L-F2.
L-F1 Important: answers seam (PropagationAnswers; architecture.md:69-70,103; A-36) never built — Bands/Path/Reach/centre computed in tty (map_prop_chart.go:171,180,:104; map_prop_centre.go:78-80); undocumented departure; chart targets fixed package var; future Analyst/Scout must re-implement. Build seam or ratify + fix architecture.md.
L-F2 Important: hours ahead show today's D-RAP disturbance — watchpost splices Ahead hour into live snapshot (map_prop_ahead.go:94-97) keeping Disturbance; Bands/Path apply it (answers.go:179,185,208,213); Hour.Typical never read. Probe: Disturbed bands at +6/+12/+18 h. Fix: library AheadSnapshot or skip disturbance on Typical hours.
L-F3 Important: answer memo keys (propChartKey, propCentreKey, propReachKey) miss the no-readings correction / rebuild at same Computed (update.go:138-139) — stale chart/centre/reach after toggling No readings (and noise? chart key has noise). Add build sequence to snapshot keys; extend key-coverage guard; real-library test.
L-F4 Important: any ctx cancellation (deadline) reported as too soon / notYet "NOAA's solar data could not be read ... after 00:00:00 UTC" (update.go:110-113,135-136,147-148); slow link (<45 KB/s) never gets first field (60 s deadline, 30 s per request, 2.5 MB grid first); deadline in GIRO burst spends hour. Fix: deadline vs cancel, typed errors, solar first, never format zero time.
L-F5 Important: parse failures swallowed (glotec update.go:199-201, drap :250-252, scales :267-269, solar.go:130-133); Filled computed never surfaced. Add Inputs.Refused, GloTECFilled; diagnostics; "partly climatology".
L-F6 Important: GIROPaused (snapshot.go:139) never produced; architecture.md:263 promises "GIRO paused"; Rejected cumulative vs per-snapshot Stations. Produce GIROPaused; per-burst Rejected.
L-F7 Important: GIRO throttle in memory only; restart = fresh 39-request burst; multiple instances each; A-20 dropped FR-4.8 cross-instance claim as agent row (behaviour). Persist throttle state or HUM ratifies risk.
L-F8 Minor: bands matched by position (map_prop_chart.go:209-213,:435; map_prop_origin.go:89-93) with clamps; edges in 3 places; add Band.Name, key by b.Band; agreement test.
L-F9 Minor: sun position thrice; go-tuimaps PlaceReport.Daylight unused, daylightKey churns memo; words' horizon (-0.833) differs from drawn terminator (0) -> "It is night" near line on day side. One horizon.
L-F10 Minor: Library.Sources static via throwaway Library (= C-5); propagationBodyBytes copied 8 MB.
L-F11 Minor: Bitset no exported constructor; no-data path untested from host.
L-F12 Minor: NaN control point -> cellIndex int(NaN) reads arbitrary cell (answers.go:229-230,402-405; path.go:77-79). Guard non-finite.
L-F13 Minor: = C-17 badColours wrong kind; checkHostRampLocked duplicates (= C-26).
L-F14 Minor: hour window [Computed, +1h) vs architecture "hour starting at its UTC hour"; Bands 1 min before Computed errNoHour; Snapshot doc wrong (= C-11).
```

