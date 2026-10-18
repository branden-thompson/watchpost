---
title: "0.19.0 BUILD exit red team, round 2 - reviewer notes"
date: 2026-10-18
phase: BUILD exit
sev: SEV-0
authority: HUM LEAD
status: "Four fresh blind reviewers (D-225). Each reviewer's findings, condensed by the agent from its report; every finding cites the file and line it read at watchpost 559f22db, go-ionomaps 2436c8a, go-tuiMaps 228d6f7. The harness kept the full reports from being written as files; they survive only in the session."
---

# BUILD exit - red team, round 2: the reviewers' notes

Up: [the report](../red-team-build.md)

Prefixes: BL2 (the BUILD lens), CN2 (Code), DN2 (Docs), PN2 (the combined personas: Accessibility, InfoSec, Performance).

## lens (BL2)

```text
BUILD lens, round 2 - Do not exit BUILD. No Critical. Fix first: BL2-1.
Closure: L-F1 NOT closed (partial: reach map_prop_chart.go:101 and hour ahead map_prop_ahead.go:58 outside propanswers; D-212's frequency dropped without a row, build-log.md:526; answers.go:2-3 claims a frequency's reach). L-F2, L-F3 closed. L-F4 closed in library; host never says Unfinished (BL2-7). L-F5, L-F6 closed in record only for the host (BL2-3). L-F7 closed with gap (BL2-5). L-F8 NOT closed: typical-day rows labelled by position hfBands[min(i, len-1)] map_prop_chart.go:419; band edges third copy map_prop_draw.go:144-145. L-F9 closed, reopened for hours ahead (BL2-4). L-F10 closed (stale comment mapprop.go:95-97; propagationHostList test-only). L-F11 closed in library only (map_prop_draw_test.go:100 clears NoData). L-F12, L-F13, L-F14 closed.
BL2-1 Important: reach sentence says "above the MUF everywhere: it reaches nowhere" whenever !Reaches, ignoring ReachField.Limit (map_prop_freq.go:118-131; go-ionomaps reach.go:68); 1.9 MHz at noon is absorbed (reach_test.go:146-153); FR-1.16, D-205. Say the reach by its Limit; host test at 1.9 MHz noon.
BL2-2 Important: D-215 not carried into watchpost: method-value workaround propagation.go:70-72, history.go:698-700, resolver.go:128,162; unreachable visits==nil guards history.go:1384,1536,1574; batch 45 renamed reservoir.offer->keep for P10-01 (scenarios.go:227); propanswers guards duplicate library's with narrower longitude (answers.go:84,96 vs go-ionomaps answers.go:430). Write plainly, exemptions to HUM, delete guards.
BL2-3 Important: library honesty fields reach no listener: Inputs.Refused, GloTECFilled, Reason Unfinished/GIROPaused, Rejected unused outside tests; "no stations: the background alone" said for choice, restart and GIRO failure alike (map_prop_update.go:288-313); architecture.md:129,165; FR-5.2 requirements.md:141. Say each Reason, refused feeds, "by choice"; a test per Reason.
BL2-4 Minor: hour ahead: terminator at the hour's middle (map_prop_draw.go:83-84) but words day/night at its start (map_prop_words.go:108,115); library answers at start (ahead.go:31). One moment (the start); test.
BL2-5 Minor: throttle state saved only after Update returns (propagation.go:172-173): quit mid-burst (up to 36 s) loses it; two instances share one .tmp name (propagation.go:141); host test never bursts (propagation_test.go:303-341).
BL2-6 Minor: M1/M4 verdict a point estimate without interval (report.go:152-168); 2.6 points on 1600 paired, 160 per band; HUM fork: paired bootstrap or McNemar interval.
BL2-7 Minor: raw Go error text read to listener: "context deadline exceeded" (go-ionomaps update.go:180; map_prop_update.go:193). errors.Is for words.
BL2-8 Minor: watchpost never merged main 0.18.1 (8465dc0c not ancestor of 559f22db): CHANGELOG 0.18.1, ci.yml Go 1.27.2, release.yml; red-team-build.md:81 says merged.
BL2-9 Minor: no pre-code READY record for BUILD/remediation; propanswers first under domains/, hook first as shell, B1 ruled before dry run (REFLECT L1 again).
Also: propChartKey.noise duplicates built (map_prop_chart.go:22,25); exported mutable propanswers.Continents aliased (map_prop_chart.go:33). Outside scope: go-tuiMaps uncommitted gate-runs.md.
```

## code (CN2)

```text
Code Quality, round 2 - Do not exit BUILD. Fix first: CN2-1. Tests and vet pass; p10 --base 5012fae7 0 live (10 exempted); no control-flow function past 60 lines; M1/M4 dry run reproduces exactly (53.4/53.4, 64.2/66.9).
Closure: C-1, C-2 closed. C-3 part (31 prop* fields loose, A-68(5); map_pane.go:66-67 stale). C-4 declined (A-68(6), D-223). C-5 part (propagationHostList test-only, mapprop.go:95-104). C-6 closed. C-7 NOT closed in watchpost (CN2-3). C-8 closed. C-9 part (no-P10 run exits 0, scripts/gate:271-273; hosted runs all such, gate.yml:5-6). C-10 record only (CN2-7). C-11, C-12 closed. C-13, C-14 absent from the condensed notes. C-15 part (Class.Shade no consumer). C-16 NOT closed (go-tuimaps scripts/gate:457-468,651-659). C-17, C-18, C-19, C-20 closed. C-21 declined (A-70(3)). C-22, C-23 closed. C-24 declined (5xx still silent). C-25 NOT closed (blend.go:87, preset.go:214). C-26 part (Set only). C-27 closed.
CN2-1 Important: Observer answers freeze after pan + another window within 600 ms: applyViewSettled drops the tick when map not in front (map_pane.go:1527-1530); armAnswer asks nothing while viewGen != viewAsked (map_prop_chart.go:88); chart, reach, centre "being worked out" until next pan; test helper returns early in exactly this case (map_prop_chart_test.go:28).
CN2-2 Important: corrupt throttle state crashes watchpost each open: go-ionomaps Apply bounds ProbeAt not PollAt (stations.go:284); live[(pollAt+i)%n] overflows (:184); poll=MaxInt64 accepted by Restore, Update panics in unrecovered goroutine (propagation.go:169-175), against D-124; fixed .tmp name (:141). Bound PollAt; FuzzRestore with an update; os.CreateTemp.
CN2-3 Important: D-215 not carried in watchpost: newLibrary (propagation.go:70-72), Last/Forget/Do (history.go:698-700), visits==nil guards (history.go:1384,1536,1574); new density guards: propanswers limits.go:26,47,109,134,179,197,211, words.go:28; answers.go:78-98,127-141 duplicate library's (answers.go:216,429-433) with different bounds (radius 0 allowed here, refused there); A-70(6) says added for density; reachscore spots.go:49-52, replay.go:116,152,164,180, report.go:70,79,126,132,156,173, score.go:52,133,362; reservoir.offer renamed keep for P10-01 (scenarios.go:227). Delete, restore offer, put shortfall and analyser gaps to HUM.
CN2-4 Important: reach words ignore ReachField.Limit (map_prop_freq.go:118-131); D-205 absent from watchpost build log, plan, requirements.
CN2-5 Important: snapshotWords no case for GIROPaused/Unfinished (map_prop_update.go:290-297): read as fresh; Refused, GloTECFilled, Rejected read nowhere; architecture.md:165.
CN2-6 Minor: spokenFrequency regex adds "megahertz" after "under"/"up to" decimal (map_prop_words.go:295), runs over weather lines (map_describe.go:66): "visibility under 1.5 mi" -> "under 1.5 megahertz miles". Unit at the format site.
CN2-7 Important: TestEveryCodeCitationResolves checks only line within file (record_test.go:193-211), no minimum; drifted: requirements.md:150, implementation-plan.md:64,:74,:80.
CN2-8 Important: pre-push hook omits batch 36's two record checks (tools/prepush/main.go:22-26); so does the guard list (record_test.go:282).
CN2-9 Important: reachscore fedAsSaid fails one way only: never requires a grid in background/shipped, grid age, Refused, or stations in shipped (score.go:104-143); empty set exits 0 (report.go:153); no-data counts as not reaching (score.go:177); unknown revision not marked uncitable (report.go:246-256); runlog exits with gate code when its record failed (tools/runlog/main.go:44-47).
CN2-10 Important: go-tuiMaps M5 not enforced: benchmarks once, no threshold (scripts/gate:765); NFR-1 release step no checklist row; refused Edge reported as UnsortedBreaks, fix text "name a preset" (grid.go:53-55,135-141); test only err != nil (grid_test.go:179-187). HUM: error kind (API freeze).
CN2-11 Important: go-ionomaps same climatology in two caches (aheadcache.go:16-59, typical.go:38-62); comments say reset on new day/mean, rule is an entry count. Ahead reads typical day's cells; delete aheadHours.
CN2-12 Minor: colour-depth Setting three parallel tables (map_depth.go:26,42-54,66-81).
CN2-13 Minor: no consumer: propagationHostList, propagationSources (mapprop.go:95-108); go-ionomaps Field.WithNoData (snapshot.go:50-61); go-tuiMaps Class.Shade (facts.go:22); always-true r.memo != nil (frame.go:1125).
CN2-14 Minor: bands matched by position (map_prop_chart.go:278-280, :366-380, :419); hfBands repeats edges (map_prop_draw.go:145-146). Use s.Band.String().
CN2-15 Minor: misleading names before freeze: Inputs.Refused lists unparsed files (snapshot.go:133); Kept() returns state to save (throttle.go:112); GIRO 5xx resets back-off uncounted (asker.go:151-159).
CN2-16 Minor: ~69 production comments cite unprefixed review IDs (map_pane.go:69,97, map_prop_chart.go:22,75, propagation.go:158, go-tuimaps overlays.go:316); map_work.go:6's "C-7" another release's; go-ionomaps state.go:14-17 vs :66-68.
CN2-17 Minor: G10.2's GIRO cases pass when nothing reported (faults_test.go:82-83); G10.1 climatology only (reproduce_test.go:29-41) yet report counts G-M2 built; go-tuiMaps M5 bench turns contours off (globe_bench_test.go:68): 10.3 ms with them, not 6.6.
CN2-18 Minor: hour ahead accepted by Computed (map_prop_ahead.go:66): a same-moment rebuild keeps stale noise; key by propBuilt.
CN2-19 Minor: kept reading place recomputes all words each message (map_prop_place.go:92-169, dashboard.go:1168-1171): 1.57 ms, 4,432 allocs per visualiser tick vs 10 us at top. Memoise.
P10: package rows absorb new code with stale figures: geodata row covers NearWorld (0.00 over 7), quota.go row "0.34 across 93". Re-measure.
```

## docs (DN2)

```text
Docs Quality, round 2 - Do not exit BUILD yet. No new Critical. Fix first: DN2-1. 205 named tests all exist.
Closure: D-F1, D-F2 closed. D-F3 closed in go-ionomaps, NOT in watchpost (DN2-1). D-F4 part (architecture.md:69 channel type, :34, :70 stale; DN2-7). D-F5 record only (DN2-8). D-F6 closed (status line stale DN2-12). D-F7 closed (contract.md:206 leftover DN2-13). D-F8 part (as-built-map.md:112 rc.1). D-F9 closed (reach cost missing DN2-2). D-F10 part (go-tuimaps release-notes.md:27 "0.011 deg" no basis). D-F11 part (as-built-map.md:7). D-F12 part (rulings.md:4 phase: PLAN; requirements.md:7 stops at D-127; go-tuimaps requirements.md:65). D-F13, D-F14 closed. D-F15 NOT closed (go-tuimaps build-log.md:90). B-F4, B-F5 closed; C-10 record only; H-6 open for go-tuiMaps.
DN2-1 Important: propReachWords (map_prop_freq.go:119-133) ignores ReachField.Limit; "above the MUF everywhere" false when absorbed/weak (D-205 rulings.md:226; FR-1.16 requirements.md:84). Say Limit; absorbed-reach test.
DN2-2 Important: reach cost since D-205 0.96 s first, 0.43 s repeat (M5 Pro; round 1 28 ms) recorded only in go-ionomaps build-log.md:125; FR-1.17 requirements.md:85 omits it; architecture.md:112 "answered at once"; every settled pan with a frequency held asks a reach (map_prop_freq.go:74-84, map_prop_chart.go:88-101). Record; G1 workload; fork if unacceptable.
DN2-3 Important: build-report.md:75-78 misstates floor 2: hours ahead ARE the climatology since D-109 (met by construction; rulings.md:130, requirements.md:240-241); G10.4 measures G-M3 not hours ahead.
DN2-4 Important: D-217 (rulings.md:238) requires BUILD/VALIDATE reports to say PS-1 is addressed only in part; build-report.md silent.
DN2-5 Important: D-22's RBN cross-check (rulings.md:43; problem-statement.md:53, requirements.md:238-239, uat-protocol.md:43) dropped without ruling; instrument never mentions RBN.
DN2-6 Important: as-built-map.md last updated batch 36 (:112 rc.1, :92 the join batch 39 removed, :7 0.18.0 header; nothing on propanswers, throttle state, HF stations, Colours); its test app/maps_test.go:55-75 reads only 0.18.0's log; report says record current (build-report.md:18).
DN2-7 Important: architecture.md:34, :70 say app turns fields into overlays/answers into words; code: app/mapprop.go:29-121 only wires; window converts (map_prop_draw.go:42) and calls propanswers (not in diagram); :69 <-chan Snapshot vs PropagationResult (dashboard.go:243); :113 memo key lacks kept counter (A-68 (2)).
DN2-8 Important: citations drifted: requirements.md:150 history.go:918-920 (maxDocBytes now :922); architecture.md:75 resolver.go:101-102 (gate :119); architecture.md:69 :114 (MapTemperature :133-135); implementation-plan.md:64,:74,:80 (history.go:1478, map_pane.go:466-477, dashboard.go:72); check record_test.go:212-223 only tests line exists. Cite symbols; check matches symbol/quote.
DN2-9 Important: build-report.md:35,:87 "hosted CI read after every push since batch 34" unsupported: build-log.md:563 runs 38-43 cancelled by next push; :572 batch 45 committed while 44's run going; 46, 47 silent. State what happened; record tip's hosted result.
DN2-10 Important: README documents nothing of 0.19.0: README.md:171 "one of two modes", :166 one region, :186-197 no P, :204-209 no Propagation Settings, :223-238 hosts omit GIRO/NOAA SWPC (FR-4.3 requirements.md:125), :248 no propagation-throttle.json, Broadcaster g map absent. README + CHANGELOG before UAT, or a follow-up SHIP is held to.
DN2-11 Minor: instrument page :103, diagram :111 and tools/reachscore/main.go:4 still say B1 band-hour; default now band (score.go:290-291); ruled draw's table (:203-206) lacks by-kind row; "fewer than 40 %" (:178-179) true only of old draw (M4 open 43.6/51.9/51.2 by band).
DN2-12 Minor: status lines: rulings.md:4 phase PLAN; requirements.md:7 D-127; go-ionomaps README.md:8 "rc in preparation"; release-checklist.md:7 rc.1 only; api-review.md:7 "release notes" (none); go-tuimaps README.md:121 "about thirty minutes" (2925-3471 s).
DN2-13 Minor: go-tuiMaps: release-notes.md:13 vs :18 (two-class edge); contract.md:206 OW-18 v0.3.0 vs requirements.md:90 v0.4.0; requirements.md:65 render/basemap.go path gone.
DN2-14 Minor: REQUIRED-READING omits red-team-build.md (:30) and instrument page (:28); build-report.md:37 "from 21.7 ms" vs D-209's 33.6 ms unreconciled.
DN2-15 Minor: report's M1/M4 lack context: constant answer scores 50 % under balance; columns agree with only 29-52 % of heard paths.
Maintainability: citation check and as-built test are the structural causes; FR-1.10, FR-3.1 cells > 200 words. Also: go-tuimaps uncommitted gate-runs.md line (docs gate at dd67d22, 2026-10-18T03:28Z).
```

## personas (PN2: Accessibility, InfoSec, Performance)

```text
Combined personas, round 2 - Do not exit BUILD. No Critical. Fix first: PN2-2. Tests (-race) pass; govulncheck no called vulnerability.
Closure: AX-1 to AX-14 closed (AX-5 cost PN2-5; AX-6 wording PN2-8; AX-8 country codes as letters PN2-8; AX-11 request not cancelled PN2-7). I-1 closed; I-2 NOT closed (PN2-4); I-3 to I-8 closed. P-1 closed for pans (hour steps ungated, nothing cancelled, PN2-3); P-2 closed; P-3 closed (measured 6.8-7.8 ms at 400x110, 1.4-1.5 at 200x56); P-4 closed for chart (21.6 ms) and centre (7.5 ms), reach cost only in go-ionomaps' record (PN2-3); P-5, P-6 closed.
PN2-1 Important (InfoSec): throttle state with poll=MaxInt64 passes Restore; rotation (pollAt+i)%n overflows, "index out of range [-8]" (go-ionomaps stations.go:188, :284; state.go:73-90); watchpost update goroutine unrecovered (propagation.go:169-176): crash every open until the file is deleted. Refuse PollAt >= len(known) or store modulo; add to TestABadStateIsRefused.
PN2-2 Important (A11y): D-205/FR-1.16 not in the words: map_prop_freq.go:119-134 (:127-128) says "above the MUF everywhere" when absorbed/weak (80 m midday). Say field.Limit; LimitAbsorption test.
PN2-3 Important (Perf): reach 0.42-0.44 s warm, 0.92-0.93 s day's first (BenchmarkReachKept/First; go-ionomaps A-20), record says ~30 ms, FR-1.17 omits it; hour steps not settled, nothing cancelled: holding shift+right through 24 hours queues 24 reaches, 10-22 CPU-s, 0.3-0.4 GB garbage (reach_bench_test.go:8-25; map_prop_freq.go:83; build-log.md:272; requirements.md:85; map_prop_chart.go:94-96,143-151). One latest-wins reach worker, or settle hour steps; record; G1 with a frequency held.
PN2-4 Important (InfoSec): I-2 open: positionShape ASCII-only; not refused: "33°12′N 117°23′W" (U+2032), "33º12'N", "33.2, −117.38" (U+2212), "(33.2, -117.38)", "lat 33.2 lon -117.38", "geo:33.2,-117.38" (resolver.go:199, :210-215); FR-4.9. Refuse two or more digit groups (ZIP+4 excepted).
PN2-5 Minor (Perf): D-211 recomputes words every message when scrolled: 1.58 ms, 4,432 allocs per Update vs 15.6 us (dashboard.go:1167-1173; map_prop_place.go:32-44,145-170). Re-anchor on gen change only.
PN2-6 Minor (InfoSec): geocoder Name/Admin1/CountryCode unsanitised to the terminal (geocode.go:89-94; map_prop_find.go:64; cf nws/alerts.go:185). Clamp at geocode.go:89-94.
PN2-7 Minor (InfoSec): Esc drops a find's answer but not its request: FindPlace no context (map_prop_find.go:81-90,96-101; app/resolve.go:65-69).
PN2-8 Minor (A11y): Help "n Now (an hour, in Propagation)" attaches the hour to Now (map_pane.go:1474); centre "near Tokyo, JP" read as letters (geodata/index.go:51-52).
PN2-9 Minor (A11y/InfoSec): failed online lookup reads httpx error with redacted URL (geocode.go:75; httpx.go:947; map_prop_find.go:58). Plain words; cause to diagnostics.
PN2-10 Minor (A11y): Now's values spoken as "N hours ahead" while the hour ahead is being worked out (map_prop_words.go:92-112; map_prop_ahead.go:92-101,154).
```

