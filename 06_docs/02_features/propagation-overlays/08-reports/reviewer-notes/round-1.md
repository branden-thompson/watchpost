---
title: "0.19.0 DISCOVER exit red team, round 1: reviewer notes (condensed)"
date: 2026-10-06
phase: DISCOVER exit
sev: SEV-0
authority: HUM LEAD
status: "CONDENSED by the agent from the eight reviewers' reports as they arrived. The full texts were returned as messages and were not saved (the agent's error E-9); these notes are the record. Finding IDs here are the ones red-team-discover.md cites."
---

# DISCOVER-exit red team, round 1 — reports as received, condensed

## PM (DISCOVER lens) — DO NOT EXIT; fix P1 first
- P1 CRITICAL: PS-1 actor may be prohibited: 47 CFR 97.113(b) "An amateur station shall not engage in any form of broadcasting, nor ... one-way communications except as specifically provided". No regulatory check anywhere. Action: regulatory finding, then HUM re-rules PS-1.
- P2 Imp: issue #25 filed by HUM; problem statements written back from solution; WSPR (observed spots) never considered as the ANSWER for PS-2.
- P3 Imp: B-on-D recreates PS-G's bad outcome (NC GIRO, no contact D-41, undocumented form redirect). Restate PS-G or score G-M1 honestly.
- S1 Imp: alignment = one person; no second HF operator; KC2G/LGDC never contacted. S2 Minor: go-giro-data developer hypothetical.
- R1 CRITICAL: band status from foF2/MUF only; no LUF / D-layer absorption (160/80 m closed by day). P.533 planner silently dropped.
- R2 Imp: L-5 (valued station points) has no watchpost consumer; delete? R3 Imp: F-189 eSSN chart + planner still "0.19.0", not in reqs, no ruling drops them. R4 Minor: v0.3.0 PS = HR list in prose.
- I1 Imp: do-nothing not evaluated (link out + KC2G email).
- IR1 Imp: no bytes/day guardrail; fleet load to GIRO unbounded; default-on. IR2 Minor: UHF service radius reused as HF served area; README says 2-100 mi vs D-28 3-120. IR3 Imp: GIRO "educational and non-commercial research purposes" vs hobby display untested.
- E2 Imp: NOAA D-RAP absorption + R/S/G scales not surveyed. E3 Minor: no competitor comparison (HamClock, VOACAP Online, PSKReporter map).
- K1 Imp: RK-1 "arodland/prop never opened" FALSE (research read via GitHub API; GC-1 admits); D-1's protocol "to be ruled in DISCOVER" never ruled.
- K2 Imp: field biased high -> false "open" -> PS-1 bad outcome; B-on-D signed bias not reported; one day.
- K3 Imp: M1 may be unmeasurable for NVIS (<200 km) with WSPR; dry-run feasibility before exit.
- K4 Minor: RK-2 mitigation stale; RK-6 under-rated; G-M3 still names KC2G.

## Business Quality — DO NOT PROCEED; fix F1 first
- F1 CRITICAL: no LUF/absorption (same as PM R1). D-RAP or P.533 absorption.
- F2 Imp: M3 target <=20 min impossible under hourly cadence (D-29/D-39).
- F3 Imp: targets owed in DISCOVER (D-8/D-9/D-12) not set except M1/M4 moved by D-23.
- F4 Imp: M1 reference can't see NVIS paths (same as K3).
- F5 Imp: D-238 "all of it" (hours ahead, station dots, eSSN chart, P.533 planner) cut without record; F-189 unchanged; PS-2 "at a given hour" but B-on-D gives current hour only.
- F6 Imp: go-tuiMaps OW-14..OW-26 owed to v0.3.0, not in v0.3.0 reqs, no ruling moves them.
- F7 Imp: fleet load / IP to UML / 60 MB/day NOAA default-on; G1 no network; no follow-up for "GIRO blocks us".
- F8 Imp: clean-room claim false (same as K1).
- F9 Imp: D-25/D-26 tower clauses have no FR row; can Broadcaster open Propagation mode? #25 asks MUF on a Broadcaster map; fork for HUM.
- F10 Minor: G-M3 KC2G reference unreachable. F11 Minor: drift — wave2 status "path next"; requirements (path) markers; go-giro-data brief paths A/B/C + OQ-G1..4 open; README "before any data is used" vs wave 2 used GIRO; F-195/F-200/F-202 still say 0.19.0/v0.3.0 vs D-15; go-tuimaps brief cites watchpost D-23 for principles (it's M1 baseline; principles are 0.18.0 D-23); brief says issue "asked first" but D-8 says not asked.
- F12 Minor: L-5 no consumer. Cheaper: F-174 map more than PS-1 needs (readout alone); KC2G email; path D alone.

## Agent verification of PM P1 (law.cornell.edu, 2026-10-06)
- 47 CFR 97.113(b): "An amateur station shall not engage in any form of broadcasting, nor may an amateur station transmit one-way communications except as specifically provided in these rules." CONFIRMED.
- 97.113(c): no retransmission of non-amateur programs or signals, except "propagation and weather forecast information intended for use by the general public and originated from United States Government stations" — and these "may not be conducted on a regular basis, but only occasionally, as an incident of normal amateur radio communications."
- 97.111(b) one-way allowed: adjustments, establishing two-way contact, telecommand, emergency communications, Morse practice, information bulletins (defined in 97.3), telemetry.
- => a REGULAR relay of watchpost's weather broadcasts on amateur HF looks prohibited in the US (not legal advice). PS-1 (agent-drafted, HUM-approved D-7) rests on it. Emergency comms (97.403) and two-way nets differ.

## Accessibility persona — DO NOT PROCEED as written; fix AX-C first
- AX-A Imp: mode is a two-state chip (map_radar.go:448-453); no-picture path never names the mode (map_pane.go:492-503); FR-1.1 should require mode named in words in every path.
- AX-H Imp: Broadcaster map keys not required as actions; console Help fixed groups (help_about.go:232-240,250); key clashes + = - (gain vs zoom, router.go:170-171 / map_pane.go:753-754), O (Observer swap vs Overlays, router.go:161 / map_pane.go:757). Add FR-2 row + clash tests.
- AX-M Minor: outside-region message names no way forward (map_pane.go:473-474) -> name the Propagation key.
- AX-B Imp: controls inside Propagation mode unspecified (choose MUF/foF2, UTC hour, move between targets, focus after Lookup); Help already overfull (map_pane.go:1280-1282).
- AX-L Imp: FR-2.7 allows raising 44-row floor -> excludes large-font users; fork: readout shrinks/scrolls vs raise and record exclusion.
- AX-C Imp (FIRST): describeUpTo early returns (map_describe.go:65-70) before any layer sentence; alerts off/never asked -> wrong advice or "loading" forever; D-78 drops selected place out of view (map_describe.go:63,110-116). FR-3.3 instrument must cover 3 cases; D-78-in-Propagation-mode is HUM call.
- AX-F Imp: band status not required in words; R-2.2 yes/no without NO DATA; precedent broadcaster.go:1124-1130. OPEN/CLOSED/NO DATA words.
- AX-K Minor: speech mangles MHz/foF2/"160 m"; house rule units in words (map_describe.go:11-12).
- AX-E Imp: night by colour only; terminator vs contours by colour token; add L-4.4 describe day/night + non-colour terminator mark; TestNightTint with CheckRamp CVD rules + SafeRamps.
- AX-D Imp: no-picture path drops status line (loading/offline/coarse) (map_pane.go:492-503 vs 506-510; mapStatusText 1288-1299) -> build Propagation mode's share of AX-5 with layer (per D-9).
- AX-G Imp: Broadcaster map has no words; "Instead of the map" is Observer-scoped (setup_rows.go:258). Add FR-2.9.
- AX-I Imp: "in words/said/spoken" undefined; closure notice lifetime unset; voice could go ON AIR; terminal no live region -> keep notice until acknowledged or logged in Status.
- AX-J Minor: if hours auto-play, reduce motion arrives later; step hours on a key only.
- AX-N Imp: nothing enforces RK-6; recommend SHIP gate failing while any FR-7 test missing.

## InfoSec persona — PROCEED with IS-1..IS-5 as requirement rows first; fix IS-1 first
- IS-1 Imp: httpx retries 429 (httpx.go:926-928); bare 429 -> sleeps ~1s and re-asks (httpx.go:886-890; memo.go:136-151); clients MaxRetries 1 (app.go:139, dashboard.go:544); breaks D-39. Prod pace 30/s (app.go:132), not 5/s as wave2 says (wave2-findings.md:69). FR: propagation client never retries 429; fake-server test; correct pace in record.
- IS-2 Imp: every GIRO reply carries "# Requested by unknown guest from IP:<addr>" (fc_*.txt:8); goldens/errors/httpx disk cache (cache.go:403-417) could store it -> parser drops header lines; errors never quote input; scrubbed goldens; TestNoRawSourceDataIsStored scans httpx cache + diagnostics; MAP STATUS says host logs IP.
- IS-3 Imp: physical ranges unchecked; NaN/Inf from formulas; one NaN poisons GP; go-tuiMaps checkGrid ranges only speed/direction (field.go:69,85,109-111); future-dated soundings; station codes scraped raw into URLs (burst.py:69-71); provider names from replies -> About/say unsanitised (only cleanWarning, assembler.go:400). Add R row: ranges, caps, reject counts, fuzz no NaN/Inf, ^[A-Z0-9]{5}$ + url.Values, static credits.
- IS-4 Imp: FR-1.10 lookup sends raw query to Open-Meteo geocoder (resolver.go:90-110; geocode.go:54), refuses non-US (geocode.go:65) -> can't simply reuse for global; no coordinate parser; ParseFloat NaN/Inf; geocoder replies cached by URL containing query (httpx.go:9-16). Local coord parse, NoCache, test the cache, disclose geocoder egress.
- IS-5 Imp: GIRO NC access condition binds every fetching copy, not only redistributors (RK-2 project-brief.md:232 wrong); FR-4.1/FR-2.4 notice must state it; default-on new third party with no opt-in (D-29 disclosure gap); history.Dataset no terms field (history.go:66-73).
- IS-6 Minor: "allowed hosts" not in httpx; RefusePrivate/HTTPSOnly opt-in (httpx.go:69-72); 32 MB body cap (httpx.go:291) -> require options + per-source caps.
- IS-7 Minor: wave-2 probe continued after D-38's stop rule (second ad hoc run rows 121-129 drew a 2nd 429); findings present as one run. Record deviation. (AGENT'S OWN DEVIATION — own it.)
- IS-8 Minor: go-giro-data no govulncheck requirement.

## Docs Quality — DO NOT PROCEED; fix F16 first (absorption). Wave-2 numbers REPRODUCED exactly; request log matches; citations hold.
- F1 Imp: wave2 status stale "IN PROGRESS ... D-20 next"; reader meets 960/day and "weighs against D" before corrections -> open with verdict, mark day-1 superseded.
- F2 Minor: requirements.md:7 + FR-4.1 (path) markers stale.
- F3 Minor: REQUIRED-READING lacks requirements.md, wave1, wave2, GG reqs; line 28 v0.3.0 brief approved (TM D-8).
- F4 Imp: targets: problem-statement.md:46 misstates D-8 ("DISCOVER and PLAN dry run"); M2,M3,M5,M5b,G1,G-M2,G-M3,G-G1 unset; GG ps :7,:35, GG reqs :91 repeat.
- F5 Imp: M3 <=20 min impossible under hourly (D-29/D-39).
- F6 Imp: RK-1 "never opened" false -> "not opened since 0.18.0 research; no code copied".
- F7 Imp: RK-2 stale (960/day, path D mitigation, owner D-20); ~900/day per copy default-on; fleet unknown.
- F8 Imp: 0.18.0 D-238 eSSN + planner dropped without ruling; D-31 MUF/foF2 only.
- F9 Imp: FR-6.1 hourly recording vs FR-4.2 nothing fetched before mode opened: does the recorder fetch GIRO/NOAA for an Observer who never opens the mode? fork.
- F10 Imp: G-M3 KC2G second reference unmeasurable.
- F11 Minor: R-6.1 reproducibility vs R-4.3 no raw fixtures -> inputs re-fetched under throttle.
- F12 Minor: D-28 question "120 mi radius ~190 km across" WRONG: 193 km radius, ~386 km across; conclusion roughly survives. Correcting A-row.
- F13 Minor: wave1:181 compute 2.4e8 wrong; 65,341x24x1,429 ~ 2.2e9; "under a second" unsupported.
- F14 Minor: tower "always marked" no FR row; D-26 Broadcaster-origin overtaken by D-28 unstated.
- F15 Minor batch: brief "KC2G baseline" (:153,:217) vs D-23; brief facts.go:186 (:127,:171); GG GC-3 (:94) licences "unverified" (wave1 checked); TM brief :113 cites watchpost D-23 for P-1..P-3 (should be 0.18.0 D-23); wave2:151 "Europe, Americas and Asia" wrong (no mainland Asia; Pacific islands, Australia, South Africa); loo_hybrid.py docstring says climatology background.
- F16 Imp (FIRST): band status from foF2 only; no LUF/absorption; "open" undefined; M1/M4 need it.
- F17 Imp: M1 WSPR density at <200 km untested.
- F18 Minor: "best at every distance" overstated (1.06 vs 1.08 within ±0.05) -> "best or tied".
- F19 Minor: engineers-first order; fork (a) 10-line summaries (b) leave, point designers to briefs.
- F20 Imp: 0.37/1.00 quoted without limits: <500 km bin = 7 European stations (DB049 FF051 GM037 PQ052 RO041 SO148 VT139); 4 US stations all >1000 km where B-on-D 1.06-1.28 ~ GloTEC; MUF never scored for B-on-D (foF2 only); MUF RMS 4.02/4.29 wider than band gaps; kernel chosen on same day; RK-3 cites 2-station result.

## Performance persona — PROCEED with provisional bounds (G1 incl bytes/day, >=48h; G-G1; M5) + F4 PLAN-blocking; fix F4 first
- F1 Imp: httpx retries 429 (httpx.go:926-928; do :702-716; ~1s ±50% :886-893); app clients 30/s MaxRetries 1 (app.go:132) -> fetcher handed to library must have zero retries; test attempts per 429. (= InfoSec IS-1)
- F2 Minor: hourly update re-downloads whole day (burst.py:70-71) -> ask only from last sounding held.
- F3 Imp: daily live-station list refresh = probing 118 > burst 40 and > bucket ~90; state how learned; if probing, 2/min ~59 min.
- F4 CRITICAL (existing code, data loss): rollUp ignores failed read and writes year back (history.go:1478-1481); read cap 32 MB uncompressed (:857,:879); global 2-field grid roll-up/day 0.51 MB @2°, 2.03 MB @1° -> year doc over cap after 66 days @2°, 16 days @1°; reproduced 68 days -> 1 day, Corrupt=1. Triggered by Data tab 90d/1y/5y presets (app/history.go:470-471) under FR-6.3; @1° even default 30d. Quadratic year rewrite. Retention by time, not bytes (2° day 1.39 MB on disk). Action: refuse write on failed read; split roll-ups (month); byte bound per dataset.
- F5 Imp: []*float64 per value (history.go:1014-1029); day memo 24h up to 256 days (:193); heap 12.5 MB @2°, 50.1 MB @1° per day; ~130k allocs/Put @1°. -> []float64 or keep global grids out of day memo.
- F6 Imp: 2.5 MB GloTEC > small tier 2 MB (cache.go:86-87) -> large tier (:323-334); max-age=60 + ETag -> 24h grace LRU (:112,:358-363); new URL hourly -> ~15 MB dead grids evict hot feeds -> NoCache.
- (3) typed GeoJSON decode 5.1 ms/0.74 MB vs map 10.5 ms/7.6 MB/192k allocs -> require typed (Minor).
- F7 Minor: renderMap on UI goroutine (map_pane.go:346-363); whole-globe pan frame fill 0.36/1.2 ms, with contours 3.9/14.8 ms (200x56/400x110); contours forced at 16/no colour (field.go:139-141); L-3.2 adds pass; cold update ~38 sequential GIRO @0.46 s median, Connection: close -> ~18 s vs M2 30 s. M5 per-frame bound; library Update off UI goroutine.
- F8 Imp: throttle per library object, three callers (readout, Propagation mode, recorder) + several watchpost instances share a store -> 76-114 req/h > bucket. One object per process, concurrent-safe, merged Updates, hour claimed across instances via store Claim, compute once.
- F9 Imp: G1 proposed only, G-G1 no number, M5 unbounded; compute estimate unmeasured; G1 window 1h misses daily costs; no bytes term (~70 MB/day hourly; ~360 MB/day at 6/h). Provisional bounds now; 48h; bytes/day.
- F10 Imp: throttle measured at a pace watchpost doesn't use (wave2 says 5/s "watchpost's way"; real client 30/s, 16 in flight, httpx.go:96). (= IS-1 pace note)
- F11 Imp: M3 <=20 min impossible under D-39 (would need ~152 GIRO req/h); GloTEC publication lag 18-29 min (median 24) from Last-Modified.

## Code Quality — DO NOT PROCEED; fix E2 first. Numbers reproduce at F10.7=100.
- U1 Imp: record contradicts itself (path markers, RK-2 960/day, wave2 status, "OQ-G2 to be ruled" wave2:143, FR-4.1 no PyIRI/NRL notice). U2 Minor: "DRAFT — approved at" -> "for approval at". U3 Minor: path C dropped without ruling (B absorbed C).
- M1 Imp: throttle has three owners (D-39, FR-4.5 + GG R-5.3 same tests); delete FR-4.5's three tests, keep host test in T1. M2 Imp: M3 unreachable (= others).
- C1 Imp: PyIRI port largest item; baseline role needs no Go (offline Python); fallback covers GloTEC outages, availability unmeasured; delete "and the baseline"; measure GloTEC availability in dry run before planning port. C2 Minor: R-1.3 P.533 for MUF(3000) = foF2×M3000 by definition; move under R-2.2.
- N1 Minor R-9.2 conditional -> PLAN question. N2 R-5.2 cadence clause dup. N3 NFR-2 dup R-8.1. N4 FR-4.5 "rate headers are read" no behaviour. N5 FR-4.5 tests. N6 Imp L-5 no consumer + raw readings on map vs D-41. N7 G-M3 KC2G. N8 R-2.3 many-points (rec only). N9 FR-10.1/.4/.5 document instruments can't fail -> process commitments; FR-10.4 docs-reading test.
- S1 Minor TestTheBackgroundMatchesPyIRI should be Fallback. S2 Minor loo_hybrid docstring/label. S3 Minor: name "go-giro-data" no longer describes library (2 of 3 inputs not GIRO) -> rule name before PLAN.
- D1 Minor pairs.py usage doc wrong (burst-{D}/). D2 Minor F-206 "already records hourly" not built; route 2 circular.
- T1 Imp: httpx retries 429 (httpx.go:5,:890,:926-927, consumes Retry-After :846) -> MaxRetries 0 + pass status/headers; test with real httpx + counting 429 server. T2 Imp: foF2-only band status (absorption). T3 Minor L-4.2 tolerance unstated.
- E1 Minor pairs.py exits 0 with zero pairs -> fail. E2 Imp (FIRST): MUF never scored for B-on-D; agent's run: B-on-D foF2 × GloTEC M3000 -> MUF RMS 3.97 vs 4.02 GloTEC; with measured M(3000) 3.22 -> M3000 error (Shimazaki inverse) dominates; R-9.3 must assimilate M(3000)F2 residuals too; dry run scores MUF LOO. E3 Imp: climatology baseline depends on F10.7 (103 -> clim RMS 1.47 not 1.38); PyIRI 0.0.4 raw CCIR scored, NRL refits (to ship) not scored; write F10.7 rule + coefficient set before N. E4 Minor: wave2 request count "GIRO 5 of 5" wrong (120 GIRO + 24 NOAA same UTC day). E5 Minor: subset table and throttle phases 3-4 have no script (ad hoc) — keep scripts. E6 Imp: live count 38 of 116 (SMK29, TR169 answered by probes saving nothing... note); D-39 cap 40 leaves 2 spare; no requirement for >40 live. E7 Minor best-at-every-distance overclaim.
- V1 Imp: v0.3.0 M4 check proves existence not failure -> mutant anchors. V2 Imp: R-6.1 golden outside tree -> skips green in CI; synthetic in-tree + dry-run instrument fails when real golden missing. V3 Minor: no-raw-data allowed list -> scan content; D-row for list change. V4 Imp: targets missing -> metrics can't fail.

## Project Hygiene — PROCEED after one docs pass + 3 rulings (live-list, M3, targets); fix F5+F10 first
- Trees clean; IP only in scratch. F1 Minor dead branches (watchpost feature/map-drawing, go-tuimaps feature/radar-loops, local feature/go-tuimaps) -> delete on HUM confirm.
- Docs lanes green from clean clone (watchpost 3m23s, go-tuimaps 5m53s); go-giro-data empty module honest. origin SSH alias unresolvable; go-giro-data local origin/feature/discover stale.
- Gates real: go-tuimaps gate-runs rows match recomputed trees; watchpost hosted CI passed on 3d55a09e.
- F2 Imp live-station list learning has no home (118 probe > cap/bucket). F3 Imp M3. F4 Imp targets slipped by wording. F5 Imp evidence scripts in temp dir; NFR-3 headers only in scratch -> commit scripts + request log + run note, no data. F6 Minor stale rows (wave2:7,:143; reqs :7,:67; RK-2 :138; RK-3 :139; problem-statement :65 band question answered by D-30; follow-ups :89,:92,:93,:97,:98 still "D-256 first"). F7 Minor OQ-G4 unruled; G-M3 KC2G; F-206 not pointed from go-giro-data. F8 Minor F-206 route 1 premise false (GloTEC relaxes to IRI). F9 Minor go-giro-data reqs :46-47 bare D-2; TM brief :113 D-23; :114 issue promise. F10 Imp: run went past D-38's stop; requests 98-120 from code not in evidence; bucket model fitted to 2nd refusal; tell HUM. F11 Minor TR169 served on probe, body discarded -> 39 stations. F12 Minor script docs; day.sh never ran but described. F13 Minor "approved at" wording.
